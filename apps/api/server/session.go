package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	CookieName   = "hb_session"
	CookiePath   = "/api/v1"
	tokenBytes   = 32
	maxSessions  = 10000
	clientHeader = "X-HB-Client"
	csrfHeader   = "X-CSRF-Token"
)

type session struct {
	verifier  identity.Verifier
	workspace string
	csrf      string
}

type sessions struct {
	mu   sync.Mutex
	byID map[identity.Verifier]*session
}

func newSessions() *sessions { return &sessions{byID: map[identity.Verifier]*session{}} }

func random(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

func encode(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func verifierOf(token []byte) identity.Verifier { return identity.Verifier(sha256.Sum256(token)) }

func (s *sessions) put(v identity.Verifier, workspace string) (string, error) {
	raw := random(tokenBytes)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.byID) >= maxSessions {
		return "", ErrStart
	}
	csrf := encode(raw)
	s.byID[v] = &session{verifier: v, workspace: workspace, csrf: csrf}
	return csrf, nil
}

func (s *sessions) get(v identity.Verifier) (session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := s.byID[v]
	if found == nil {
		return session{}, false
	}
	return *found, true
}

func (s *sessions) drop(v identity.Verifier) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byID, v)
}

func (s *Server) OpenSession(triple identity.Triple, workspace string, secure bool) (cookie *http.Cookie, csrf string, err error) {
	raw := random(tokenBytes)
	v := verifierOf(raw)
	if csrf, err = s.sessions.put(v, workspace); err != nil {
		return nil, "", err
	}
	if err := s.identity.Boundary().OpenSession(triple, v, sessionIdle, sessionAbsolute); err != nil {
		s.sessions.drop(v)
		return nil, "", err
	}
	return &http.Cookie{Name: CookieName, Value: encode(raw), Path: CookiePath, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: secure,
		MaxAge: int(sessionAbsolute / time.Second)}, csrf, nil
}

func (s *Server) lookup(r *http.Request) (session, bool) {
	c, err := r.Cookie(CookieName)
	if err != nil {
		return session{}, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(c.Value)
	if err != nil {
		return session{}, false
	}
	return s.sessions.get(verifierOf(raw))
}

func sameToken(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
