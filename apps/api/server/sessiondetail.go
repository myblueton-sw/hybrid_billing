package server

import (
	"net/http"
	"net/url"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (s *sessions) indexedSession(key sessionOwner, handle string) (session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.owners[key]
	if index == nil || index.handles[handle] == nil {
		return session{}, false
	}
	return *index.handles[handle], true
}

func (s *Server) readSessionDetail(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	s.domain.RLock()
	defer s.domain.RUnlock()
	found, answer, ok := s.self(w, r)
	if !ok {
		return
	}
	handle := r.PathValue("handle")
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil || len(values) > 0 || !isHex(handle, 16) {
		invalidFields(w, request, "session")
		return
	}
	if found.owner != (sessionOwner{answer.User(), answer.Workspace(), answer.Epoch()}) {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	target, known := s.sessions.indexedSession(found.owner, handle)
	if !known {
		failCode(w, commands.UnsupportedCommand, request)
		return
	}
	writeJSON(w, http.StatusOK, request, struct {
		Session   queriedSession `json:"session"`
		Reference string         `json:"reference_at"`
	}{s.taggedSessionView(target, found.verifier), s.clock().UTC().Format(time.RFC3339Nano)})
}
