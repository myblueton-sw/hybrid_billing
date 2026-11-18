package server

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	selfEndsKept    = 5
	selfEndAttempts = 4
	selfEndHandles  = 64
	selfEndLabel    = "self_session_end_v1"
	kindOthers      = "others"
	kindOne         = "one"
	resultEnded     = "ended"
	resultGone      = "already_gone"
	itemDone        = "done"
	itemPending     = "pending"
	itemFailed      = "failed"
	itemCleared     = "already_ended"
	itemOutside     = "outside_scope"
	itemNotApply    = "not_applicable"
	stateRunning    = "cleanup_running"
	stateComplete   = "complete"
	stateIncomplete = "cleanup_incomplete"
)

var selfEndBackoff = []time.Duration{time.Second, 4 * time.Second, 16 * time.Second}

var errSelfEndUnexpected = errors.New("unexpected session end error")

type selfItem struct {
	Item   string `json:"item"`
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

type selfEndRow struct {
	Handle   string     `json:"handle"`
	SignedIn string     `json:"signed_in"`
	Result   string     `json:"result"`
	Items    []selfItem `json:"items"`
}

type selfRef struct {
	Handle   string `json:"handle"`
	SignedIn string `json:"signed_in"`
}

type selfCounts struct {
	Done         int `json:"done"`
	AlreadyEnded int `json:"already_ended"`
	Failed       int `json:"failed"`
	Pending      int `json:"pending"`
}

type selfEndView struct {
	Operation         string       `json:"operation"`
	Kind              string       `json:"kind"`
	Count             int          `json:"count"`
	State             string       `json:"state"`
	BlockedAt         string       `json:"blocked_at"`
	ListGeneration    string       `json:"list_generation"`
	CurrentGeneration string       `json:"current_generation"`
	GenerationChanged bool         `json:"generation_changed"`
	Digest            string       `json:"digest"`
	Cleanup           selfCounts   `json:"cleanup"`
	Sessions          []selfEndRow `json:"sessions"`
	NotIncluded       []selfRef    `json:"not_included"`
	Replay            bool         `json:"replay"`
}

type selfEnd struct {
	id          string
	kind        string
	blockedAt   time.Time
	before      string
	listed      string
	after       string
	digest      string
	rows        []selfEndRow
	notIncluded []selfRef
}

type selfRetry struct {
	op       *selfEnd
	row      int
	attempts int
	next     time.Time
	final    bool
}

type endBody struct {
	Session    string    `json:"session"`
	Generation *string   `json:"generation"`
	Handles    *[]string `json:"handles"`
}

func endedItems(c3, reason string) []selfItem {
	return []selfItem{
		{Item: "C1", State: itemDone, Reason: "deny_applied"},
		{Item: "C2", State: itemDone, Reason: "deny_applied"},
		{Item: "C3", State: c3, Reason: reason},
		{Item: "C4", State: itemDone, Reason: "pickups_dropped"},
		{Item: "C5", State: itemDone, Reason: "attempts_cleared"},
		{Item: "C6", State: itemOutside, Reason: "in_flight_requests_finish"},
		{Item: "C7", State: itemOutside, Reason: "cannot_recall"},
		{Item: "C8", State: itemNotApply, Reason: "no_jobs_or_api_keys"},
	}
}

func endDigest(kind, listed string, handles []string) string {
	sorted := append([]string{}, handles...)
	sort.Strings(sorted)
	h := sha256.New()
	_, _ = io.WriteString(h, selfEndLabel+"\n"+kind+"\n"+listed+"\n"+strings.Join(sorted, "\n"))
	return hex.EncodeToString(h.Sum(nil))
}

func isHex(v string, n int) bool {
	if len(v) != n {
		return false
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func decodeEnd(r io.Reader) (endBody, string, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return endBody{}, "", err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	var body endBody
	if err := d.Decode(&body); err != nil {
		return body, "", errShape
	}
	if _, err := d.Token(); err != io.EOF {
		return body, "", errShape
	}
	if body.Session == kindOthers {
		if body.Generation == nil || !isHex(*body.Generation, 32) {
			return body, "generation", errShape
		}
		if body.Handles == nil || len(*body.Handles) == 0 || len(*body.Handles) > selfEndHandles {
			return body, "handles", errShape
		}
		seen := map[string]bool{}
		for _, h := range *body.Handles {
			if !isHex(h, 16) || seen[h] {
				return body, "handles", errShape
			}
			seen[h] = true
		}
		return body, "", nil
	}
	if body.Generation != nil || body.Handles != nil {
		return body, "session", errShape
	}
	if !isHex(body.Session, 16) {
		return body, "session", errShape
	}
	return body, "", nil
}

func invalidFields(w http.ResponseWriter, request string, fields ...string) {
	rule := moduleRules[commands.InvalidRequest]
	body := errorBody{Code: string(commands.InvalidRequest), NextAction: string(rule.next), RequestID: request}
	for _, f := range fields {
		if f != "" {
			body.Fields = append(body.Fields, f)
		}
	}
	writeJSON(w, rule.status, request, body)
}

func (s *Server) endModule(v identity.Verifier) error {
	if s.faulty("self-end-unavailable") {
		return identity.ErrUnavailable
	}
	if s.faulty("self-end-unexpected") {
		return errSelfEndUnexpected
	}
	return s.identity.Boundary().EndSession(v)
}

func settleOf(err error) (string, string) {
	switch {
	case err == nil:
		return itemDone, "session_ended"
	case identity.ReasonOf(err) == identity.ReasonSession:
		return itemCleared, "already_closed"
	case errors.Is(err, identity.ErrUnavailable):
		return itemPending, "retrying"
	}
	return itemFailed, "unexpected_error"
}

func (op *selfEnd) view(replay bool) selfEndView {
	v := selfEndView{Operation: op.id, Kind: op.kind, BlockedAt: stamp(op.blockedAt), ListGeneration: op.listed, CurrentGeneration: op.after,
		GenerationChanged: op.kind == kindOthers && op.listed != op.before, Digest: op.digest, Sessions: []selfEndRow{}, NotIncluded: append([]selfRef{}, op.notIncluded...),
		Replay: replay}
	for _, row := range op.rows {
		copied := row
		copied.Items = append([]selfItem{}, row.Items...)
		v.Sessions = append(v.Sessions, copied)
		if row.Result != resultEnded {
			continue
		}
		v.Count++
		switch row.Items[2].State {
		case itemDone:
			v.Cleanup.Done++
		case itemCleared:
			v.Cleanup.AlreadyEnded++
		case itemFailed:
			v.Cleanup.Failed++
		default:
			v.Cleanup.Pending++
		}
	}
	switch {
	case v.Cleanup.Pending > 0:
		v.State = stateRunning
	case v.Cleanup.Failed > 0:
		v.State = stateIncomplete
	default:
		v.State = stateComplete
	}
	return v
}

func (s *Server) selfEndViews(user string) []selfEndView {
	out := []selfEndView{}
	for _, op := range s.selfEnds[user] {
		out = append(out, op.view(false))
	}
	return out
}

func (s *Server) remember(user string, op *selfEnd) {
	list := append([]*selfEnd{op}, s.selfEnds[user]...)
	if len(list) > selfEndsKept {
		list = list[:selfEndsKept]
	}
	s.selfEnds[user] = list
}

func (s *Server) endSelfSessions(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.Lock()
	defer s.domain.Unlock()
	found, answer, ok := s.self(w, r)
	if !ok || !s.stepUpRoute(w, found, answer.User(), "sessions/end", request) {
		return
	}
	body, field, err := decodeEnd(r.Body)
	if err != nil {
		if tooLarge(err) {
			fail(w, http.StatusRequestEntityTooLarge, CodeBodyTooLarge, request)
			return
		}
		invalidFields(w, request, field)
		return
	}
	current := handleOf(found.verifier)
	kind, listed, handles := kindOne, "", []string{body.Session}
	if body.Session == kindOthers {
		kind, listed, handles = kindOthers, *body.Generation, *body.Handles
	}
	for _, h := range handles {
		if h == current {
			invalidFields(w, request, map[string]string{kindOthers: "handles", kindOne: "session"}[kind])
			return
		}
	}
	digest := endDigest(kind, listed, handles)
	if latest := s.selfEnds[answer.User()]; len(latest) > 0 && latest[0].digest == digest {
		writeJSON(w, http.StatusAccepted, request, latest[0].view(true))
		return
	}
	if kind == kindOne {
		known := false
		for _, other := range s.sessions.snapshot(found.login).sessions {
			known = known || (other.verifier != found.verifier && handleOf(other.verifier) == body.Session)
		}
		if !known {
			invalidFields(w, request, "session")
			return
		}
	}
	if s.selfEndHook != nil {
		s.selfEndHook()
	}
	before := s.sessions.generation(found.login)
	result := s.sessions.endManifest(found.login, found.verifier, handles, kind == kindOthers)
	if !result.ok {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	op := s.denyManifest(found, kind, listed, digest, result)
	s.remember(answer.User(), op)
	view := op.view(false)
	if len(result.ended) > 0 {
		s.counter++
	}
	s.cfg.Log.Info("sessions_ended", "user", answer.User(), "operation", op.id, "ended", view.Count, "already_gone", len(result.gone),
		"not_included", len(result.notIncluded), "generation_changed", view.GenerationChanged, "before", before, "after", s.sessions.generation(found.login))
	select {
	case s.wake <- struct{}{}:
	default:
	}
	writeJSON(w, http.StatusAccepted, request, view)
}

func (s *Server) denyManifest(found session, kind, listed, digest string, result manifestResult) *selfEnd {
	now := s.clock()
	s.selfSeq++
	op := &selfEnd{id: "session-end-" + strconv.FormatUint(s.selfSeq, 10), kind: kind, blockedAt: now, before: result.before, listed: listed, after: result.after,
		digest: digest, notIncluded: []selfRef{}}
	for _, ended := range result.ended {
		v := ended.verifier
		state, reason := settleOf(s.endModule(v))
		row := selfEndRow{Handle: handleOf(v), SignedIn: stamp(ended.created), Result: resultEnded, Items: endedItems(state, reason)}
		op.rows = append(op.rows, row)
		s.dir.dropPickups(func(p pickup) bool { return p.session == v })
		delete(s.failures, v)
		if state == itemDone || state == itemCleared {
			continue
		}
		if !s.sessions.deny(v, found.login, now) {
			op.rows[len(op.rows)-1].Items[2] = selfItem{Item: "C3", State: itemFailed, Reason: "denied_set_full"}
			continue
		}
		retry := &selfRetry{op: op, row: len(op.rows) - 1, attempts: 1, next: now.Add(selfEndBackoff[0]), final: state == itemFailed}
		s.retries[v] = retry
	}
	for _, h := range result.gone {
		op.rows = append(op.rows, selfEndRow{Handle: h, Result: resultGone, Items: []selfItem{}})
	}
	for _, other := range result.notIncluded {
		op.notIncluded = append(op.notIncluded, selfRef{Handle: handleOf(other.verifier), SignedIn: stamp(other.created)})
	}
	return op
}

func (s *Server) settle(v identity.Verifier, retry *selfRetry, state, reason string) {
	item := &retry.op.rows[retry.row].Items[2]
	item.State, item.Reason = state, reason
	if state == itemDone || state == itemCleared {
		s.sessions.release(v)
		delete(s.retries, v)
		return
	}
	if state == itemFailed {
		retry.final = true
	}
}

func (s *Server) drainSelfEnds() int {
	now := s.clock()
	worked := 0
	keys := make([]identity.Verifier, 0, len(s.retries))
	for v := range s.retries {
		keys = append(keys, v)
	}
	sort.Slice(keys, func(a, b int) bool { return handleOf(keys[a]) < handleOf(keys[b]) })
	for _, v := range keys {
		retry := s.retries[v]
		if retry.final || now.Before(retry.next) {
			continue
		}
		worked++
		retry.attempts++
		state, reason := settleOf(s.endModule(v))
		if state == itemPending {
			if retry.attempts >= selfEndAttempts {
				state, reason = itemFailed, "retry_budget_spent"
			} else {
				retry.next = now.Add(selfEndBackoff[retry.attempts-1])
			}
		}
		s.settle(v, retry, state, reason)
	}
	for _, v := range s.sessions.sweep(now) {
		delete(s.retries, v)
	}
	return worked
}

func (s *Server) settleEnded(v identity.Verifier, err error) {
	retry := s.retries[v]
	if retry == nil {
		return
	}
	if state, reason := settleOf(err); state == itemDone || state == itemCleared {
		s.settle(v, retry, state, reason)
	}
}

func (s *Server) settleDenied(login string) {
	for _, v := range s.sessions.deniedOf(login) {
		s.settleEnded(v, s.endModule(v))
	}
}
