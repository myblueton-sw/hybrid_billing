package server

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const sessionOrder = "current_then_signed_in_desc_handle_asc"

type sessionQuery struct {
	Session   string `json:"session"`
	Workspace string `json:"workspace"`
	Since     string `json:"since"`
	Until     string `json:"until"`
	Q         string `json:"q"`
	Limit     int    `json:"limit"`
}

type queriedSession struct {
	selfSession
	State  string `json:"state"`
	CanEnd bool   `json:"can_end"`
}

type sessionCount struct {
	Value     int    `json:"value"`
	Certainty string `json:"certainty"`
}
type sessionQueryPage struct {
	Rows        []queriedSession `json:"rows"`
	Next        string           `json:"next"`
	Query       sessionQuery     `json:"query"`
	Reference   string           `json:"reference_at"`
	Generation  string           `json:"generation"`
	Order       string           `json:"order"`
	Count       sessionCount     `json:"count"`
	ScanStopped bool             `json:"scan_stopped"`
	Settings    []settingRow     `json:"settings"`
}

func parseSessionQuery(values url.Values, workspace string, defaultLimit int) (sessionQuery, string, string) {
	out := sessionQuery{Session: "all", Workspace: workspace, Limit: defaultLimit}
	after := ""
	keys := make([]string, 0, len(values))
	for name := range values {
		keys = append(keys, name)
	}
	slices.Sort(keys)
	for _, name := range keys {
		list := values[name]
		if len(list) != 1 || list[0] == "" {
			return out, "", name
		}
		raw := list[0]
		switch name {
		case "session":
			if raw != "all" && raw != "current" && raw != "other" {
				return out, "", name
			}
			out.Session = raw
		case "workspace":
			if raw != workspace {
				return out, "", name
			}
			out.Workspace = raw
		case "q":
			q, ok := parseSearch(raw)
			if !ok || strings.ContainsFunc(q, unicode.IsControl) {
				return out, "", name
			}
			out.Q = strings.ToLower(q)
		case "since", "until":
			if len(raw) > 40 {
				return out, "", name
			}
			at, valid := parseSessionStamp(raw)
			if !valid {
				return out, "", name
			}
			if name == "since" {
				out.Since = at.UTC().Format(time.RFC3339Nano)
			} else {
				out.Until = at.UTC().Format(time.RFC3339Nano)
			}
		case "limit":
			n, err := strconv.Atoi(raw)
			if err != nil || n < 1 || n > 100 {
				return out, "", name
			}
			out.Limit = n
		case "after":
			if len(raw) > 4096 {
				return out, "", name
			}
			after = raw
		default:
			return out, "", name
		}
	}
	if out.Since != "" && out.Until != "" {
		since, _ := time.Parse(time.RFC3339, out.Since)
		until, _ := time.Parse(time.RFC3339, out.Until)
		if !since.Before(until) {
			return out, "", "until"
		}
	}
	return out, after, ""
}

func (q sessionQuery) digest() string {
	values := url.Values{"session": {q.Session}, "workspace": {q.Workspace}, "since": {q.Since}, "until": {q.Until}, "q": {q.Q}, "limit": {strconv.Itoa(q.Limit)}}
	sum := sha256.Sum256([]byte(values.Encode()))
	return hex.EncodeToString(sum[:])
}

func (q sessionQuery) matches(found session, caller identity.Verifier) bool {
	if q.Session == "current" && found.verifier != caller || q.Session == "other" && found.verifier == caller {
		return false
	}
	if q.Since != "" {
		at, _ := time.Parse(time.RFC3339, q.Since)
		if found.created.Before(at) {
			return false
		}
	}
	if q.Until != "" {
		at, _ := time.Parse(time.RFC3339, q.Until)
		if !found.created.Before(at) {
			return false
		}
	}
	return q.Q == "" || strings.Contains(strings.ToLower(found.handle+" "+found.created.UTC().Format(time.RFC3339Nano)+" "+found.created.Add(sessionAbsolute).UTC().Format(time.RFC3339Nano)), q.Q)
}

func (s *Server) sessionView(found session, caller identity.Verifier) queriedSession {
	row := queriedSession{selfSession: selfSession{Handle: found.handle, Current: found.verifier == caller, SignedIn: found.created.UTC().Format(time.RFC3339Nano), Ends: found.created.Add(sessionAbsolute).UTC().Format(time.RFC3339Nano)}, State: "active"}
	if s.sessions.isDenied(found.verifier) || s.stale(found) {
		row.State = "revoked"
		return row
	}
	answer, _, err := s.identity.Boundary().Peek(found.verifier, found.workspace)
	switch {
	case errors.Is(err, identity.ErrUnavailable):
		row.State = "unavailable"
	case err != nil:
		row.State = "revoked"
		if identity.ReasonOf(err) == identity.ReasonExpired || identity.ReasonOf(err) == identity.ReasonSession && (!s.clock().Before(found.created.Add(sessionAbsolute)) || !s.clock().Before(found.idleUntil)) {
			row.State = "expired"
		}
	case answer.User() != found.owner.User || answer.Epoch() != found.owner.Epoch:
		row.State = "revoked"
	default:
		if fresh, reason := s.freshness(answer, found); !fresh {
			row.State = "blocked"
			if reason == authpolicy.ReasonStale {
				row.State = "expired"
			} else if reason == "" {
				row.State = "unavailable"
			}
		}
	}
	row.CanEnd = row.State == "active" && !row.Current
	return row
}

func (s *Server) sessionBasis(found session, answer identity.Answer) (sessionIndexState, time.Time) {
	now := s.clock()
	var age time.Duration
	if policy, err := s.policy.Policy(found.workspace); err == nil {
		age = policy.Revision.MaxAge
	}
	index := s.sessions.indexState(found.owner, now, age)
	deadline := earlier(answer.NotAfter(), now.Add(time.Duration(s.cfg.Access.SessionQuerySeconds)*time.Second))
	deadline = earlier(deadline, index.Deadline)
	return index, deadline
}

func (s *Server) querySessions(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.self(w, r)
	if !ok {
		return
	}
	if len(r.URL.RawQuery) > 8192 {
		invalidFields(w, request, "query")
		return
	}
	values, parseErr := url.ParseQuery(r.URL.RawQuery)
	if parseErr != nil {
		invalidFields(w, request, "query")
		return
	}
	query, after, field := parseSessionQuery(values, answer.Workspace(), s.cfg.Access.SessionQueryPage)
	if field != "" {
		invalidFields(w, request, field)
		return
	}
	if found.owner != (sessionOwner{answer.User(), answer.Workspace(), answer.Epoch()}) {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	index, expires := s.sessionBasis(found, answer)
	now := s.clock()
	cursor := sessionCursor{Owner: found.owner, Caller: found.handle, Query: query.digest(), Version: index.Version, Revision: s.Revision(), Reference: now, Expires: expires}
	if after != "" {
		prior, valid := s.openSessionCursor(after)
		if !valid || prior.Owner != cursor.Owner || prior.Caller != cursor.Caller || prior.Query != cursor.Query || prior.Version != cursor.Version || prior.Revision != cursor.Revision || !now.Before(prior.Expires) || prior.Reference.After(now) || prior.Matched < 0 || prior.Matched > maxSessions {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		cursor = prior
	}
	page := sessionQueryPage{Rows: []queriedSession{}, Query: query, Reference: cursor.Reference.UTC().Format(time.RFC3339Nano), Generation: strconv.FormatUint(cursor.Version, 10), Order: sessionOrder, Count: sessionCount{Value: cursor.Matched, Certainty: "lower_bound"}, Settings: s.rowsOf(sessionQuerySettings)}
	if !cursor.Pinned {
		cursor.Pinned = true
		if query.Session != "other" && query.matches(found, found.verifier) {
			page.Rows = append(page.Rows, s.sessionView(found, found.verifier))
			cursor.Matched++
		}
	}
	more := query.Session != "current" && index.Count > 1
	scanned := 0
	for more && len(page.Rows) < query.Limit && scanned < s.cfg.Access.SessionQueryScan {
		batch, remaining, valid := s.sessions.indexed(found.owner, index.Version, cursor.After, min(s.cfg.Access.SessionQueryScan-scanned, query.Limit-len(page.Rows)))
		if !valid {
			failRead(w, commands.RevisionConflict, request, "after")
			return
		}
		more = remaining
		for _, candidate := range batch {
			scanned++
			cursor.After = candidate.handle
			if candidate.verifier == found.verifier || !query.matches(candidate, found.verifier) {
				continue
			}
			page.Rows = append(page.Rows, s.sessionView(candidate, found.verifier))
			cursor.Matched++
		}
	}
	page.Count.Value = cursor.Matched
	page.ScanStopped = more && scanned >= s.cfg.Access.SessionQueryScan
	if more {
		page.Next = s.sealSessionCursor(cursor)
	} else {
		page.Count.Certainty = "exact"
	}
	writeJSON(w, http.StatusOK, request, page)
}

func parseSessionStamp(raw string) (time.Time, bool) {
	if at := strings.IndexAny(raw, ".,"); at >= 0 {
		digits := 0
		for _, value := range raw[at+1:] {
			if value < '0' || value > '9' {
				break
			}
			digits++
		}
		if digits > 9 {
			return time.Time{}, false
		}
	}
	stamp, err := time.Parse(time.RFC3339Nano, raw)
	return stamp, err == nil
}
