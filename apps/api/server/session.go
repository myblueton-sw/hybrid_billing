package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const (
	CookieName   = "hb_session"
	CookiePath   = "/api"
	tokenBytes   = 32
	maxSessions  = 10000
	maxEvictions = 20
	clientHeader = "X-HB-Client"
	csrfHeader   = "X-CSRF-Token"
)

type session struct {
	verifier  identity.Verifier
	workspace string
	csrf      string
	created   time.Time
	login     string
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

func (s *sessions) prune(before time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for v, found := range s.byID {
		if found.created.Before(before) {
			delete(s.byID, v)
		}
	}
}

func (s *sessions) oldest(login string, except identity.Verifier) (identity.Verifier, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var best *session
	for _, found := range s.byID {
		if found.login == login && found.verifier != except && (best == nil || found.created.Before(best.created)) {
			best = found
		}
	}
	if best == nil {
		return identity.Verifier{}, false
	}
	return best.verifier, true
}

func (s *sessions) put(v identity.Verifier, workspace, login string, now time.Time) (string, error) {
	raw := random(tokenBytes)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.byID) >= maxSessions {
		return "", ErrStart
	}
	csrf := encode(raw)
	s.byID[v] = &session{verifier: v, workspace: workspace, csrf: csrf, created: now, login: login}
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

func (s *sessions) byLogin(login string) []identity.Verifier {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []identity.Verifier{}
	for v, found := range s.byID {
		if found.login == login {
			out = append(out, v)
		}
	}
	return out
}

func (s *sessions) drop(v identity.Verifier) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byID, v)
}

func (s *Server) OpenSession(triple identity.Triple, workspace string, secure bool) (cookie *http.Cookie, csrf string, err error) {
	cookie, csrf, _, err = s.openSession(triple, workspace, secure)
	return cookie, csrf, err
}

func (s *Server) openSession(triple identity.Triple, workspace string, secure bool) (*http.Cookie, string, identity.Verifier, error) {
	raw := random(tokenBytes)
	v := verifierOf(raw)
	csrf, err := s.sessions.put(v, workspace, triple.Subject, s.clock())
	if err != nil {
		return nil, "", v, err
	}
	err = s.identity.Boundary().OpenSession(triple, v, sessionIdle, sessionAbsolute)
	for tries := 0; errors.Is(err, identity.ErrLimit) && tries < maxEvictions; tries++ {
		old, ok := s.sessions.oldest(triple.Subject, v)
		if !ok {
			break
		}
		_ = s.identity.Boundary().EndSession(old)
		s.sessions.drop(old)
		err = s.identity.Boundary().OpenSession(triple, v, sessionIdle, sessionAbsolute)
	}
	if err != nil {
		s.sessions.drop(v)
		return nil, "", v, err
	}
	return &http.Cookie{Name: CookieName, Value: encode(raw), Path: CookiePath, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: secure,
		MaxAge: int(sessionAbsolute / time.Second)}, csrf, v, nil
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
