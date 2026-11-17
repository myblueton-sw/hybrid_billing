package server

import (
	"net/http"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

func (s *Server) reauthenticate(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	body, err := decodeStrings(r.Body, "password")
	s.domain.Lock()
	found, answer, ok := s.self(w, r)
	if ok && (err != nil || len(body) != 1) {
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
		s.cfg.Log.Info("reauthentication_failed", "user", answer.User(), "result", "ended")
		failCode(w, commands.Unauthenticated, request)
	}
	generation := s.dir.generations[found.login]
	s.domain.Unlock()
	if !ok {
		return
	}
	var result string
	if !s.derive(r.Context(), func() { result = s.passwords.VerifyHolder(LoginKey(found.login), body["password"]) }) {
		s.giveBack(found.verifier)
		s.cfg.Log.Info("reauthentication_failed", "user", answer.User(), "result", "unavailable")
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	if s.pause != nil {
		s.pause()
	}
	s.domain.Lock()
	defer s.domain.Unlock()
	outcome := s.commitReauth(w, found, result, generation, request)
	if outcome == "" {
		s.cfg.Log.Info("reauthenticated", "user", answer.User())
		return
	}
	s.cfg.Log.Info("reauthentication_failed", "user", answer.User(), "result", outcome)
}

func (s *Server) commitReauth(w http.ResponseWriter, found session, result string, generation uint64, request string) string {
	if _, live := s.sessions.get(found.verifier); !live {
		failCode(w, commands.Unauthenticated, request)
		return "ended"
	}
	if result != passwords.ResultOK {
		if s.failures[found.verifier] >= selfAttempts {
			delete(s.failures, found.verifier)
			s.endOne(found.verifier)
			s.counter++
			failCode(w, commands.Unauthenticated, request)
			return "ended"
		}
		fail(w, http.StatusUnprocessableEntity, CodeWrongPassword, request)
		return "wrong"
	}
	if _, code := s.access(found); code != "" {
		failCode(w, code, request)
		return "access"
	}
	if s.dir.generations[found.login] != generation {
		fail(w, http.StatusConflict, string(commands.RevisionConflict), request)
		return "conflict"
	}
	if s.faulty("reauthenticate") || !s.sessions.reauthenticated(found.verifier, s.clock()) {
		failCode(w, commands.Unauthenticated, request)
		return "ended"
	}
	delete(s.failures, found.verifier)
	headers(w, request)
	w.WriteHeader(http.StatusNoContent)
	return ""
}
