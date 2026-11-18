package server

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
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
	verifier      identity.Verifier
	workspace     string
	csrf          string
	created       time.Time
	authenticated time.Time
	login         string
	trustEpoch    uint64
}

type sessions struct {
	mu       sync.Mutex
	byID     map[identity.Verifier]*session
	seq      uint64
	gen      map[string]uint64
	token    map[string]string
	count    map[string]int
	denied   map[identity.Verifier]denial
	overflow map[string]time.Time
}

type denial struct {
	login string
	at    time.Time
}

func newSessions() *sessions {
	return &sessions{byID: map[identity.Verifier]*session{}, gen: map[string]uint64{}, token: map[string]string{}, count: map[string]int{},
		denied: map[identity.Verifier]denial{}, overflow: map[string]time.Time{}}
}

func (s *sessions) changed(login string) {
	if s.count[login] <= 0 {
		delete(s.count, login)
		delete(s.gen, login)
		delete(s.token, login)
		return
	}
	s.seq++
	s.gen[login] = s.seq
	s.token[login] = hex.EncodeToString(random(16))
}

func (s *sessions) remove(v identity.Verifier) bool {
	found := s.byID[v]
	if found == nil {
		return false
	}
	delete(s.byID, v)
	s.count[found.login]--
	return true
}

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
	touched := map[string]bool{}
	for v, found := range s.byID {
		if found.created.Before(before) && s.remove(v) {
			touched[found.login] = true
		}
	}
	for login := range touched {
		s.changed(login)
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

func (s *sessions) put(v identity.Verifier, workspace, login string, now time.Time, epoch uint64) (string, error) {
	raw := random(tokenBytes)
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.byID) >= maxSessions {
		return "", ErrStart
	}
	csrf := encode(raw)
	if s.byID[v] == nil {
		s.count[login]++
	}
	s.byID[v] = &session{verifier: v, workspace: workspace, csrf: csrf, created: now, authenticated: now, login: login, trustEpoch: epoch}
	s.changed(login)
	return csrf, nil
}

func (s *sessions) reauthenticated(v identity.Verifier, at time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	found := s.byID[v]
	if found == nil {
		return false
	}
	found.authenticated = at
	return true
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

func (s *sessions) logins() map[string][]identity.Verifier {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string][]identity.Verifier{}
	for v, found := range s.byID {
		out[found.login] = append(out[found.login], v)
	}
	return out
}

func (s *sessions) below(workspace string, epoch uint64) []identity.Verifier {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []identity.Verifier{}
	for v, found := range s.byID {
		if found.workspace == workspace && found.trustEpoch < epoch {
			out = append(out, v)
		}
	}
	return out
}

func (s *sessions) drop(v identity.Verifier) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if found := s.byID[v]; found != nil && s.remove(v) {
		s.changed(found.login)
	}
}

func (s *Server) openFor(triple identity.Triple, workspace string, secure bool) (cookie *http.Cookie, csrf string, err error) {
	cookie, csrf, _, err = s.openSession(triple, workspace, secure)
	return cookie, csrf, err
}

func (s *Server) openSession(triple identity.Triple, workspace string, secure bool) (*http.Cookie, string, identity.Verifier, error) {
	raw := random(tokenBytes)
	v := verifierOf(raw)
	if s.sessions.overflowed(triple.Subject) {
		return nil, "", v, errDeniedFull
	}
	csrf, err := s.sessions.put(v, workspace, triple.Subject, s.clock(), s.currentEpoch(workspace))
	if err != nil {
		return nil, "", v, err
	}
	err = s.identity.Boundary().OpenSession(triple, v, sessionIdle, sessionAbsolute)
	if errors.Is(err, identity.ErrLimit) {
		s.settleDenied(triple.Subject)
		err = s.identity.Boundary().OpenSession(triple, v, sessionIdle, sessionAbsolute)
	}
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

var errDeniedFull = errors.New("denied session set is full")

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
