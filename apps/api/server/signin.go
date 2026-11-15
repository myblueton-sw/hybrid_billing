package server

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

const (
	PresessionCookie = "hb_presession"
	presessionPath   = "/api/v1/auth"
	presessionAge    = 10 * time.Minute
	CodeLinkInvalid  = "link_invalid"
	CodePasswordForm = "password_form"
	limiterCapacity  = 30
	limiterRefill    = 2 * time.Second
	derivations      = 4
	setLinkValidity  = 7 * 24 * time.Hour
)

type limiter struct {
	mu     sync.Mutex
	tokens int
	last   time.Time
}

func (l *limiter) take(now time.Time) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.last.IsZero() {
		l.tokens, l.last = limiterCapacity, now
	}
	gained := max(0, int(now.Sub(l.last)/limiterRefill))
	l.tokens = min(limiterCapacity, l.tokens+gained)
	l.last = l.last.Add(time.Duration(gained) * limiterRefill)
	if l.tokens == 0 {
		return false, l.last.Add(limiterRefill).Sub(now)
	}
	l.tokens--
	return true, 0
}

func (l *limiter) refund() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.tokens = min(limiterCapacity, l.tokens+1)
}

type activation struct {
	verifier identity.Verifier
	expires  time.Time
	live     bool
	file     string
}

func NormalizeLogin(login string) (string, bool) {
	if !utf8.ValidString(login) || len(login) < 3 || len(login) > 200 {
		return "", false
	}
	out := []byte(login)
	for _, r := range login {
		if r < 0x20 || r == 0x7f || r >= 0x80 && r < 0xa0 {
			return "", false
		}
	}
	for i, c := range out {
		if c >= 'A' && c <= 'Z' {
			out[i] = c + 'a' - 'A'
		}
	}
	return string(out), true
}

func LoginTriple(login string) identity.Triple {
	return identity.Triple{Connection: PasswordLink, Issuer: Installation, Subject: login}
}

func LoginKey(login string) passwords.Key { return passwords.Key{Issuer: Installation, Subject: login} }

func decodeStrings(r io.Reader, allowed ...string) (map[string]string, error) {
	d := json.NewDecoder(r)
	if err := expectDelim(d, '{'); err != nil {
		return nil, err
	}
	out := map[string]string{}
	for d.More() {
		name, err := stringValue(d)
		if err != nil {
			return nil, err
		}
		if _, dup := out[name]; dup || !contains(allowed, name) {
			return nil, errShape
		}
		if out[name], err = stringValue(d); err != nil {
			return nil, err
		}
	}
	if err := expectDelim(d, '}'); err != nil {
		return nil, err
	}
	if _, err := d.Token(); err != io.EOF {
		if tooLarge(err) {
			return nil, err
		}
		return nil, errShape
	}
	return out, nil
}

func contains(list []string, v string) bool {
	for _, item := range list {
		if item == v {
			return true
		}
	}
	return false
}

func (s *Server) presession(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	if r.Header.Get(clientHeader) != ConsoleClient {
		fail(w, http.StatusForbidden, CodeOrigin, request)
		return
	}
	token := encode(random(tokenBytes))
	http.SetCookie(w, &http.Cookie{Name: PresessionCookie, Value: token, Path: presessionPath, HttpOnly: true, SameSite: http.SameSiteStrictMode,
		Secure: r.TLS != nil, MaxAge: int(presessionAge / time.Second)})
	writeJSON(w, http.StatusOK, request, map[string]string{"csrf_token": token})
}

func preChecked(r *http.Request) bool {
	c, err := r.Cookie(PresessionCookie)
	header := r.Header.Get(csrfHeader)
	return err == nil && header != "" && subtle.ConstantTimeCompare([]byte(c.Value), []byte(header)) == 1
}

func (s *Server) unauthenticated(w http.ResponseWriter, r *http.Request, allowed ...string) (map[string]string, bool) {
	request := r.Header.Get("X-Request-Id")
	if !preChecked(r) {
		fail(w, http.StatusForbidden, CodeCSRF, request)
		return nil, false
	}
	body, err := decodeStrings(r.Body, allowed...)
	if err != nil {
		if tooLarge(err) {
			fail(w, http.StatusRequestEntityTooLarge, CodeBodyTooLarge, request)
		} else {
			failCode(w, commands.InvalidRequest, request)
		}
		return nil, false
	}
	if ok, wait := s.limit.take(s.clock()); !ok {
		w.Header().Set("Retry-After", retryAfter(wait))
		writeJSON(w, http.StatusTooManyRequests, request, errorBody{Code: string(commands.RateLimited), Retry: true, NextAction: string(commands.NextRetry),
			RequestID: request, RetryAfterMS: retryMillis(wait)})
		return nil, false
	}
	return body, true
}

func retryMillis(wait time.Duration) int64 {
	return int64((wait + time.Millisecond - 1) / time.Millisecond)
}

func retryAfter(wait time.Duration) string {
	return strconv.FormatInt(int64((wait+time.Second-1)/time.Second), 10)
}

func (s *Server) derive(ctx context.Context, work func()) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case s.slots <- struct{}{}:
	case <-ctx.Done():
		return false
	}
	defer func() { <-s.slots }()
	work()
	return true
}

type loginBody struct {
	User      string `json:"user"`
	Workspace string `json:"workspace"`
	Role      string `json:"role"`
	CSRF      string `json:"csrf_token"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	body, ok := s.unauthenticated(w, r, "workspace", "login", "password")
	if !ok {
		return
	}
	login, valid := NormalizeLogin(body["login"])
	workspace := body["workspace"]
	if !valid || !validWorkspace(workspace) {
		failCode(w, commands.InvalidRequest, request)
		return
	}
	var result string
	if !s.derive(r.Context(), func() { result = s.passwords.Verify(LoginKey(login), body["password"]) }) {
		failCode(w, commands.DependencyUnavailable, request)
		return
	}
	if result != passwords.ResultOK {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	cookie, answer, csrf, ok := s.signIn(login, workspace, r.TLS != nil)
	if !ok {
		failCode(w, commands.Unauthenticated, request)
		return
	}
	s.limit.refund()
	http.SetCookie(w, cookie)
	writeJSON(w, http.StatusOK, request, loginBody{User: answer.User(), Workspace: answer.Workspace(), Role: string(answer.Role()), CSRF: csrf})
}

func validWorkspace(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == ':' || c == '-') {
			return false
		}
	}
	return true
}

func (s *Server) signIn(login, workspace string, secure bool) (*http.Cookie, identity.Answer, string, bool) {
	s.domain.Lock()
	defer s.domain.Unlock()
	now := s.clock()
	s.sessions.prune(now.Add(-sessionAbsolute))
	cookie, csrf, v, err := s.openSession(LoginTriple(login), workspace, secure)
	if err != nil {
		return nil, identity.Answer{}, "", false
	}
	answer, err := s.identity.Boundary().Access(v, workspace)
	if err == nil {
		_, err = s.policy.Evaluate(authpolicy.Principal{User: answer.User(), Workspace: workspace, Role: string(answer.Role()), Kind: authpolicy.KindHuman},
			authpolicy.Authentication{Method: authpolicy.MethodPassword, At: now}, "")
		var refusal authpolicy.Refusal
		if errors.As(err, &refusal) && refusal.Reason == authpolicy.ReasonNoPolicy && workspace == s.bootWorkspace && answer.Role() == identity.Administrator {
			err = nil
		}
	}
	if err != nil {
		_ = s.identity.Boundary().EndSession(v)
		s.sessions.drop(v)
		return nil, identity.Answer{}, "", false
	}
	s.counter++
	return cookie, answer, csrf, true
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	clear := &http.Cookie{Name: CookieName, Value: "", Path: CookiePath, HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: -1}
	found, ok := s.lookup(r)
	if ok {
		if !sameToken(r.Header.Get(csrfHeader), found.csrf) {
			fail(w, http.StatusForbidden, CodeCSRF, request)
			return
		}
		s.domain.Lock()
		_ = s.identity.Boundary().EndSession(found.verifier)
		s.sessions.drop(found.verifier)
		s.counter++
		s.domain.Unlock()
	}
	http.SetCookie(w, clear)
	headers(w, request)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) activating(token string) bool {
	if !s.boot.live || !s.clock().Before(s.boot.expires) {
		return false
	}
	raw, err := decodeToken(token)
	v := verifierOf(raw)
	return err == nil && subtle.ConstantTimeCompare(v[:], s.boot.verifier[:]) == 1
}

func decodeToken(token string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != tokenBytes {
		return nil, errShape
	}
	return raw, nil
}

func (s *Server) peek(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	body, ok := s.unauthenticated(w, r, "token")
	if !ok {
		return
	}
	s.domain.RLock()
	defer s.domain.RUnlock()
	if s.activating(body["token"]) {
		s.limit.refund()
		writeJSON(w, http.StatusOK, request, map[string]string{"purpose": "activate"})
		return
	}
	link, err := s.passwords.Peek(body["token"])
	if err != nil {
		fail(w, http.StatusUnprocessableEntity, CodeLinkInvalid, request)
		return
	}
	s.limit.refund()
	writeJSON(w, http.StatusOK, request, map[string]string{"purpose": link.Purpose, "login": link.Key.Subject})
}

func (s *Server) redeem(w http.ResponseWriter, r *http.Request) {
	request := r.Header.Get("X-Request-Id")
	body, ok := s.unauthenticated(w, r, "token", "password", "login", "email")
	if !ok {
		return
	}
	s.domain.RLock()
	activate := s.activating(body["token"])
	s.domain.RUnlock()
	var code string
	if activate {
		code = s.activate(r.Context(), body)
	} else {
		code = s.setPassword(r.Context(), body)
	}
	switch code {
	case "":
		s.limit.refund()
		headers(w, request)
		w.WriteHeader(http.StatusNoContent)
	case string(commands.DependencyUnavailable):
		failCode(w, commands.DependencyUnavailable, request)
	default:
		fail(w, http.StatusUnprocessableEntity, code, request)
	}
}

func (s *Server) setPassword(ctx context.Context, body map[string]string) string {
	token := body["token"]
	s.domain.RLock()
	link, err := s.passwords.Reserve(token)
	s.domain.RUnlock()
	if err != nil {
		return CodeLinkInvalid
	}
	defer s.passwords.Release(token, link.Hold)
	var record passwords.Record
	if !s.derive(ctx, func() { record, err = s.passwords.Hash(link.Key, body["password"]) }) {
		return string(commands.DependencyUnavailable)
	}
	if err != nil {
		return CodePasswordForm
	}
	if s.pause != nil {
		s.pause()
	}
	s.domain.Lock()
	defer s.domain.Unlock()
	if _, err := s.passwords.Commit(token, record); err != nil {
		return CodeLinkInvalid
	}
	s.counter++
	return ""
}

func (s *Server) activate(ctx context.Context, body map[string]string) string {
	login, ok := NormalizeLogin(body["login"])
	email := body["email"]
	if !ok || email == "" || len(email) > 254 || !strings.Contains(email, "@") {
		return CodePasswordForm
	}
	var record passwords.Record
	var err error
	if !s.derive(ctx, func() { record, err = s.passwords.Hash(LoginKey(login), body["password"]) }) {
		return string(commands.DependencyUnavailable)
	}
	if err != nil {
		return CodePasswordForm
	}
	if s.pause != nil {
		s.pause()
	}
	s.domain.Lock()
	defer s.domain.Unlock()
	if !s.activating(body["token"]) {
		return CodeLinkInvalid
	}
	if err := s.identity.Boundary().Bootstrap(s.boot.verifier, s.bootUser, LoginTriple(login), email); err != nil {
		return CodeLinkInvalid
	}
	_ = s.passwords.Put(LoginKey(login), record)
	s.boot.live = false
	_ = os.Remove(s.boot.file)
	s.counter++
	return ""
}
