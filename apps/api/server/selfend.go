package server

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesshistory"
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
	endedCount  int
	goneCount   int
	cleanup     selfCounts
	id          string
	record      *sessionRecord
	sequence    uint64
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
	Selection  *string   `json:"selection"`
	Digest     *string   `json:"digest"`
	Count      *int      `json:"count"`
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
	values, err := sessionObject(r, "session", "generation", "handles", "selection", "digest", "count")
	if err != nil {
		return endBody{}, "", err
	}
	raw, _ := json.Marshal(values)
	var body endBody
	if json.Unmarshal(raw, &body) != nil {
		return body, "", errShape
	}
	if _, selected := values["selection"]; selected {
		if len(values) != 3 {
			return body, "selection", errShape
		}
		if body.Selection == nil || !strings.HasPrefix(*body.Selection, "session-selection-") || !isHex(strings.TrimPrefix(*body.Selection, "session-selection-"), 32) || body.Digest == nil || !isHex(*body.Digest, 64) || body.Count == nil || *body.Count < 1 || *body.Count > maxSessions || body.Session != "" || body.Generation != nil || body.Handles != nil {
			return body, "selection", errShape
		}
		if _, exists := values["session"]; exists {
			return body, "session", errShape
		}
		return body, "", nil
	}
	if values["digest"] != nil || values["count"] != nil {
		return body, "selection", errShape
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
	if len(values) != 1 {
		return body, "session", errShape
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
	var dropped []*selfEnd
	if len(list) > selfEndsKept {
		dropped = append(dropped, list[selfEndsKept:]...)
		clear(list[selfEndsKept:])
		list = list[:selfEndsKept]
	}
	s.selfEnds[user] = list
	for _, old := range dropped {
		if old.record != nil && old.record.retries == 0 {
			s.forgetSessionRecord(old.record)
		}
	}

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
	if body.Selection != nil {
		s.acceptSessionSelection(w, r, found, answer, body)
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
		if latest[0].record == nil || !s.ownsSessionRecord(latest[0].record, found) {
			failCode(w, commands.UnsupportedCommand, request)
			return
		}
		writeJSON(w, http.StatusAccepted, request, latest[0].view(true))
		return
	}
	if kind == kindOne {
		known := false
		other, present := s.sessions.indexedSession(found.owner, body.Session)
		known = present && other.verifier != found.verifier
		if !known {
			invalidFields(w, request, "session")
			return
		}
	}
	record := s.legacySessionRecord(found, kind, listed, digest, handles)
	if record == nil {
		s.selectionLimit(w, request)
		return
	}
	_, status := s.applySessionRecord(found, record, kind)
	if status != http.StatusAccepted {
		s.forgetSessionRecord(record)
		if status == http.StatusUnauthorized {
			failCode(w, commands.Unauthenticated, request)
		} else {
			s.selectionLimit(w, request)
		}
		return
	}
	view := record.op.view(false)
	s.cfg.Log.Info("sessions_ended", "user", answer.User(), "operation", record.op.id, "ended", view.Count, "not_included", len(view.NotIncluded))
	writeJSON(w, http.StatusAccepted, request, view)
}

func (s *Server) populateSessionEnd(op *selfEnd, found session, result manifestResult) {
	op.record.resultPositions = map[string]int{}
	op.record.heldPositions = map[string]int{}
	now := op.blockedAt
	for _, ended := range result.ended {
		op.endedCount++
		s.recordEnded(ended, accesshistory.CauseByYou)
		v := ended.verifier
		state, reason := settleOf(s.retryModuleEnd(v))
		row := selfEndRow{Handle: handleOf(v), SignedIn: stamp(ended.created), Result: resultEnded, Items: endedItems(state, reason)}
		op.record.resultPositions[row.Handle] = len(op.rows)
		op.rows = append(op.rows, row)
		s.dir.dropPickups(func(p pickup) bool { return p.session == v })
		delete(s.failures, v)
		if state == itemDone || state == itemCleared {
			op.countCleanup(state, 1)
			continue
		}
		if !s.sessions.deny(v, found.login, now) {
			op.rows[len(op.rows)-1].Items[2] = selfItem{Item: "C3", State: itemFailed, Reason: "denied_set_full"}
			op.countCleanup(itemFailed, 1)
			continue
		}
		op.countCleanup(state, 1)
		retry := &selfRetry{op: op, row: len(op.rows) - 1, attempts: 1, next: now.Add(selfEndBackoff[0]), final: state == itemFailed}
		s.retries[v] = retry
		op.record.retries++
	}
	for _, h := range result.gone {
		op.goneCount++
		op.record.resultPositions[h] = len(op.rows)
		op.rows = append(op.rows, selfEndRow{Handle: h, Result: resultGone, Items: []selfItem{}})
	}
	for _, other := range result.notIncluded {
		op.record.heldPositions[handleOf(other.verifier)] = len(op.notIncluded)
		op.notIncluded = append(op.notIncluded, selfRef{Handle: handleOf(other.verifier), SignedIn: stamp(other.created)})
	}
}

func (s *Server) settle(v identity.Verifier, retry *selfRetry, state, reason string) {
	item := &retry.op.rows[retry.row].Items[2]
	retry.op.countCleanup(item.State, -1)
	retry.op.countCleanup(state, 1)
	item.State, item.Reason = state, reason
	if record := retry.op.record; record != nil && record.indexed {
		s.operationChanged(record.owner)
	}
	if state == itemDone || state == itemCleared {
		s.sessions.release(v)
		s.releaseSessionRetry(v, retry)
		return
	}
	if state == itemFailed {
		retry.final = true
	}
}

func (s *Server) drainSelfEnds() int {
	now := s.clock()
	s.expireSessionRecords(now)
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
		state, reason := settleOf(s.retryModuleEnd(v))
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
		if retry := s.retries[v]; retry != nil {
			s.releaseSessionRetry(v, retry)
		}
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
		s.settleEnded(v, s.retryModuleEnd(v))
	}
}

func (s *Server) releaseSessionRetry(v identity.Verifier, retry *selfRetry) {
	if s.retries[v] != retry {
		return
	}
	delete(s.retries, v)
	if record := retry.op.record; record != nil {
		record.retries--
		s.reconcileSessionRecord(record)
		retained := false
		for _, op := range s.selfEnds[record.owner.User] {
			retained = retained || op == record.op
		}
		if record.retries == 0 && !retained {
			s.forgetSessionRecord(record)
		}
	}
}
