package server

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"sort"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

const (
	CodeWrongPassword  = "current_password_wrong"
	CodePasswordLocked = "password_locked"
	selfAttempts       = 5
	othersHandle       = "others"
)

type selfSession struct {
	Handle   string `json:"handle"`
	Current  bool   `json:"current"`
	SignedIn string `json:"signed_in"`
	Ends     string `json:"ends"`
}

type selfIdentity struct {
	Method string `json:"method"`
	Login  string `json:"login"`
}

type selfPage struct {
	UserID          string         `json:"user_id"`
	Login           string         `json:"login"`
	Email           string         `json:"email"`
	Workspace       string         `json:"workspace"`
	Role            string         `json:"role"`
	Identities      []selfIdentity `json:"identities"`
	PasswordSet     bool           `json:"password_set"`
	Installation    bool           `json:"installation_administrator"`
	Sessions        []selfSession  `json:"sessions"`
	IdleMinutes     int            `json:"idle_minutes"`
	MaxAge          int64          `json:"max_age_seconds"`
	SignedInAt      string         `json:"signed_in_at"`
	AuthenticatedAt string         `json:"authenticated_at"`
	Now             string         `json:"now"`
}

func handleOf(v identity.Verifier) string {
	sum := sha256.Sum256(append([]byte("hb-session-handle"), v[:]...))
	return hex.EncodeToString(sum[:8])
}

func (s *Server) self(w http.ResponseWriter, r *http.Request) (session, identity.Answer, bool) {
	request := r.Header.Get("X-Request-Id")
	found, ok := s.lookup(r)
	if !ok {
		failCode(w, commands.Unauthenticated, request)
		return found, identity.Answer{}, false
	}
	if !sameToken(r.Header.Get(csrfHeader), found.csrf) {
		fail(w, http.StatusForbidden, CodeCSRF, request)
		return found, identity.Answer{}, false
	}
	answer, code := s.access(found)
	if code != "" {
		failCode(w, code, request)
		return found, answer, false
	}
	return found, answer, true
}

func (s *Server) ownSessions(found session) []session {
	out := []session{}
	for _, v := range s.sessions.byLogin(found.login) {
		if got, ok := s.sessions.get(v); ok {
			out = append(out, got)
		}
	}
	sort.Slice(out, func(a, b int) bool {
		x, y := out[a].verifier == found.verifier, out[b].verifier == found.verifier
		if x != y {
			return x
		}
		return out[a].created.After(out[b].created)
	})
	return out
}

func (s *Server) readSelf(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.self(w, r)
	if !ok {
		return
	}
	now := s.clock()
	person := s.dir.people[answer.User()]
	page := selfPage{UserID: answer.User(), Login: found.login, Email: person.email, Workspace: answer.Workspace(), Role: string(answer.Role()),
		Identities: []selfIdentity{{Method: "local-password", Login: found.login}}, PasswordSet: s.passwords.Has(LoginKey(found.login)),
		Installation: s.dir.holds(answer.User(), now), Sessions: []selfSession{}, IdleMinutes: int(sessionIdle / time.Minute),
		SignedInAt: stamp(found.created), AuthenticatedAt: stamp(found.authenticated), Now: stamp(now)}
	if current, err := s.policy.Policy(answer.Workspace()); err == nil {
		page.MaxAge = int64(current.Revision.MaxAge / time.Second)
	}
	for _, other := range s.ownSessions(found) {
		page.Sessions = append(page.Sessions, selfSession{Handle: handleOf(other.verifier), Current: other.verifier == found.verifier,
			SignedIn: stamp(other.created), Ends: stamp(other.created.Add(sessionAbsolute))})
	}
	writeJSON(w, http.StatusOK, request, page)
}

func (s *Server) endOne(v identity.Verifier) {
	_ = s.identity.Boundary().EndSession(v)
	s.sessions.drop(v)
	s.dir.dropPickups(func(p pickup) bool { return p.session == v })
}

func (s *Server) endOthers(found session) int {
	n := 0
	for _, other := range s.ownSessions(found) {
		if other.verifier != found.verifier {
			s.endOne(other.verifier)
			n++
		}
	}
	return n
}

func (s *Server) endSelfSessions(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.Lock()
	defer s.domain.Unlock()
	found, answer, ok := s.self(w, r)
	if !ok || !s.stepUpRoute(w, found, answer.User(), "sessions/end", request) {
		return
	}
	body, err := decodeStrings(r.Body, "session")
	if err != nil && tooLarge(err) {
		fail(w, http.StatusRequestEntityTooLarge, CodeBodyTooLarge, request)
		return
	}
	if err != nil || body["session"] == "" {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	n := 0
	if body["session"] == othersHandle {
		n = s.endOthers(found)
	} else {
		for _, other := range s.ownSessions(found) {
			if other.verifier != found.verifier && handleOf(other.verifier) == body["session"] {
				s.endOne(other.verifier)
				n = 1
			}
		}
		if n == 0 {
			failCode(w, commands.InvalidRequest, request)
			return
		}
	}
	s.counter++
	s.cfg.Log.Info("sessions_ended", "user", answer.User(), "count", n)
	writeJSON(w, http.StatusOK, request, map[string]int{"ended": n})
}

func (s *Server) reserveAttempt(found session) bool {
	for v := range s.failures {
		if _, live := s.sessions.get(v); !live {
			delete(s.failures, v)
		}
	}
	if s.failures[found.verifier] >= selfAttempts {
		return false
	}
	s.failures[found.verifier]++
	return true
}

func (s *Server) releaseAttempt(v identity.Verifier) {
	if s.failures[v] <= 1 {
		delete(s.failures, v)
		return
	}
	s.failures[v]--
}

func (s *Server) giveBack(v identity.Verifier) {
	s.domain.Lock()
	defer s.domain.Unlock()
	s.releaseAttempt(v)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	body, err := decodeStrings(r.Body, "current", "new")
	s.domain.Lock()
	found, _, ok := s.self(w, r)
	if ok && err != nil {
		ok = false
		if tooLarge(err) {
			fail(w, http.StatusRequestEntityTooLarge, CodeBodyTooLarge, request)
		} else {
			failCode(w, commands.InvalidRequest, request)
		}
	}
	if ok && !s.reserveAttempt(found) {
		ok = false
		s.endOne(found.verifier)
		s.counter++
		failCode(w, commands.Unauthenticated, request)
	}
	generation := s.dir.generations[found.login]
	s.domain.Unlock()
	if !ok {
		return
	}
	if ok, wait := s.limit.take(s.clock()); !ok {
		s.giveBack(found.verifier)
		w.Header().Set("Retry-After", retryAfter(wait))
		writeJSON(w, http.StatusTooManyRequests, request, errorBody{Code: string(commands.RateLimited), Retry: true, NextAction: string(commands.NextRetry),
			RequestID: request, RetryAfterMS: retryMillis(wait)})
		return
	}
	key := LoginKey(found.login)
	var result string
	var record passwords.Record
	var formErr error
	if !s.derive(r.Context(), func() {
		result = s.passwords.Verify(key, body["current"])
		if result == passwords.ResultOK {
			record, formErr = s.passwords.Hash(key, body["new"])
		}
	}) {
		s.giveBack(found.verifier)
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	if s.pause != nil {
		s.pause()
	}
	s.domain.Lock()
	defer s.domain.Unlock()
	if _, live := s.sessions.get(found.verifier); !live {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	switch {
	case result == passwords.ResultLocked:
		s.releaseAttempt(found.verifier)
		fail(w, http.StatusTooManyRequests, CodePasswordLocked, request)
		return
	case result != passwords.ResultOK:
		if s.failures[found.verifier] >= selfAttempts {
			delete(s.failures, found.verifier)
			s.endOne(found.verifier)
			s.counter++
			failCode(w, commands.Unauthenticated, request)
			return
		}
		fail(w, http.StatusUnprocessableEntity, CodeWrongPassword, request)
		return
	}
	delete(s.failures, found.verifier)
	if formErr != nil || body["new"] == body["current"] {
		fail(w, http.StatusUnprocessableEntity, CodePasswordForm, request)
		return
	}
	answer, code := s.access(found)
	if code != "" {
		failCode(w, code, request)
		return
	}
	if s.dir.generations[found.login] != generation {
		fail(w, http.StatusConflict, string(commands.RevisionConflict), request)
		return
	}
	if s.faulty("password") || s.passwords.Put(key, record) != nil {
		failCode(w, commands.InternalError, request)
		return
	}
	s.limit.refund()
	delete(s.dir.links, found.login)
	s.dir.generations[found.login]++
	n := s.endOthers(found)
	s.counter++
	s.cfg.Log.Info("password_changed", "user", answer.User(), "sessions_ended", n)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
