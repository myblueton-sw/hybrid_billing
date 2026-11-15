package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func (f world) pre() *http.Cookie {
	f.t.Helper()
	got := f.do("GET", "/api/v1/auth/csrf", "", func(r *http.Request) { r.Header.Set(clientHeader, ConsoleClient) })
	for _, c := range readCookies(got.header) {
		if c.Name == PresessionCookie {
			if c.Value != got.body["csrf_token"] || c.Path != "/api/v1/auth" || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.MaxAge != 600 {
				f.t.Fatalf("presession cookie %+v", c)
			}
			return c
		}
	}
	f.t.Fatal("no presession cookie")
	return nil
}

func readCookies(h http.Header) []*http.Cookie {
	return (&http.Response{Header: h}).Cookies()
}

func (f world) auth(path, body string, pre *http.Cookie) reply {
	f.t.Helper()
	return f.do("POST", "/api/v1/auth/"+path, body, func(r *http.Request) {
		if pre != nil {
			r.AddCookie(pre)
			r.Header.Set(csrfHeader, pre.Value)
		}
	})
}

func (f world) links() map[string]string {
	f.t.Helper()
	data, err := os.ReadFile(filepath.Join(f.s.cfg.Data, ActivationFile))
	if err != nil {
		f.t.Fatal(err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 1 {
			parts = []string{"", parts[0]}
		}
		out[parts[0]] = parts[1][strings.Index(parts[1], "#")+1:]
	}
	return out
}

const pw = "a long sample password"

func (f world) setPassword(login, password string) {
	f.t.Helper()
	pre := f.pre()
	got := f.auth("redeem", `{"token":"`+f.links()[login]+`","password":"`+password+`"}`, pre)
	if got.status != 204 {
		f.t.Fatalf("redeem %+v", got)
	}
}

func loginBodyFor(login, password string) string {
	return `{"workspace":"` + SampleWorkspace + `","login":"` + login + `","password":"` + password + `"}`
}

func TestSignInFlow(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.code(t, "csrf without client", f.do("GET", "/api/v1/auth/csrf", "", nil), 403, CodeOrigin)
	pre := f.pre()
	if got := f.auth("link", `{"token":"`+f.links()[AdminLogin]+`"}`, pre); got.status != 200 || got.body["purpose"] != "set" || got.body["login"] != AdminLogin {
		t.Fatalf("peek %+v", got)
	}
	f.code(t, "peek without csrf", f.auth("link", `{"token":"x"}`, nil), 403, CodeCSRF)
	f.code(t, "peek wrong", f.auth("link", `{"token":"x"}`, pre), 422, CodeLinkInvalid)
	f.code(t, "short password", f.auth("redeem", `{"token":"`+f.links()[AdminLogin]+`","password":"short"}`, pre), 422, CodePasswordForm)
	f.code(t, "same as login", f.auth("redeem", `{"token":"`+f.links()[AdminLogin]+`","password":"`+AdminLogin+`"}`, pre), 422, CodePasswordForm)
	f.setPassword(AdminLogin, pw)
	f.code(t, "reused", f.auth("redeem", `{"token":"`+f.links()[AdminLogin]+`","password":"`+pw+`"}`, pre), 422, CodeLinkInvalid)
	got := f.auth("login", loginBodyFor("Admin@Example.Invalid", pw), pre)
	if got.status != 200 || got.body["user"] != SampleAdmin || got.body["role"] != "administrator" || got.body["workspace"] != SampleWorkspace || got.body["csrf_token"] == "" {
		t.Fatalf("login %+v", got)
	}
	var session *http.Cookie
	for _, c := range readCookies(got.header) {
		if c.Name == CookieName {
			session = c
		}
	}
	if session == nil || session.Secure {
		t.Fatal("session cookie")
	}
	csrf := got.body["csrf_token"].(string)
	with := func(r *http.Request) { r.Header.Set(clientHeader, ConsoleClient); r.AddCookie(session) }
	if me := f.do("GET", "/api/v1/session", "", with); me.status != 200 || me.body["user"] != SampleAdmin {
		t.Errorf("session after login %+v", me)
	}
	f.code(t, "logout wrong csrf", f.do("POST", "/api/v1/auth/logout", "", func(r *http.Request) { r.AddCookie(session); r.Header.Set(csrfHeader, "x") }), 403, CodeCSRF)
	out := f.do("POST", "/api/v1/auth/logout", "", func(r *http.Request) { r.AddCookie(session); r.Header.Set(csrfHeader, csrf) })
	if out.status != 204 || !strings.Contains(out.header.Get("Set-Cookie"), "Max-Age=0") {
		t.Errorf("logout %+v", out)
	}
	f.code(t, "after logout", f.do("GET", "/api/v1/session", "", with), 401, string(commands.Unauthenticated))
	if again := f.do("POST", "/api/v1/auth/logout", "", nil); again.status != 204 {
		t.Error("logout without session")
	}
	failures := []string{loginBodyFor(AdminLogin, "wrong password here"), loginBodyFor("nobody@example.invalid", pw),
		`{"workspace":"other-workspace","login":"` + AdminLogin + `","password":"` + pw + `"}`}
	for i := 0; i < 5; i++ {
		f.s.passwords.Verify(LoginKey(Admin2Login), "nope nope nope")
	}
	rec2, _ := f.s.passwords.Hash(LoginKey(Admin2Login), pw)
	f.s.passwords.Put(LoginKey(Admin2Login), rec2)
	for i := 0; i < 5; i++ {
		f.s.passwords.Verify(LoginKey(Admin2Login), "nope nope nope")
	}
	failures = append(failures, loginBodyFor(Admin2Login, pw))
	bodies := map[string]bool{}
	for _, body := range failures {
		r := f.auth("login", body, pre)
		if r.status != 401 || r.body["code"] != string(commands.Unauthenticated) {
			t.Errorf("failure %s %+v", body, r)
		}
		delete(r.body, "request_id")
		raw, _ := json.Marshal(r.body)
		bodies[string(raw)+r.header.Get("Content-Type")+fmt.Sprint(len(r.header))] = true
	}
	if len(bodies) != 1 {
		t.Errorf("failures differ %v", bodies)
	}
	if len(f.s.sessions.byID) != 0 {
		t.Errorf("refused logins leave sessions %d", len(f.s.sessions.byID))
	}
	for _, body := range []string{`{"workspace":"bad ws","login":"a@b.c","password":"x"}`, `{"workspace":"w","login":"ab","password":"x"}`,
		`{"workspace":"w","login":"a\u0001b","password":"x"}`, `{"login":"x","login":"y"}`, `{"other":"x"}`, `[]`, `{"workspace":1}`} {
		f.code(t, "form "+body, f.auth("login", body, pre), 422, string(commands.InvalidRequest))
	}
	f.code(t, "login without csrf", f.auth("login", loginBodyFor(AdminLogin, pw), nil), 403, CodeCSRF)
	wrong := &http.Cookie{Name: PresessionCookie, Value: pre.Value}
	f.code(t, "header mismatch", f.do("POST", "/api/v1/auth/login", loginBodyFor(AdminLogin, pw), func(r *http.Request) { r.AddCookie(wrong); r.Header.Set(csrfHeader, "other") }), 403, CodeCSRF)
	f.code(t, "large", f.auth("login", `{"login":"`+strings.Repeat("a", 1<<20)+`"}`, pre), 413, CodeBodyTooLarge)
	f.code(t, "large trailing", f.auth("login", `{"login":"a"}`+strings.Repeat(" ", 1<<20), pre), 413, CodeBodyTooLarge)
}

func TestLoginPolicyRules(t *testing.T) {
	f := setup(t, SeedNone).ready()
	pre := f.pre()
	token := f.links()[""]
	if got := f.auth("link", `{"token":"`+token+`"}`, pre); got.status != 200 || got.body["purpose"] != "activate" {
		t.Fatalf("activation peek %+v", got)
	}
	for i := 0; i < 6; i++ {
		f.auth("redeem", `{"token":"`+encode(random(32))+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, pre)
	}
	f.code(t, "activation form", f.auth("redeem", `{"token":"`+token+`","password":"`+pw+`","login":"x","email":"first@example.invalid"}`, pre), 422, CodePasswordForm)
	f.code(t, "activation email", f.auth("redeem", `{"token":"`+token+`","password":"`+pw+`","login":"first@example.invalid","email":"none"}`, pre), 422, CodePasswordForm)
	f.code(t, "activation password", f.auth("redeem", `{"token":"`+token+`","password":"short","login":"first@example.invalid","email":"a@b"}`, pre), 422, CodePasswordForm)
	if got := f.auth("redeem", `{"token":"`+token+`","password":"`+pw+`","login":"First@Example.Invalid","email":"first@example.invalid"}`, pre); got.status != 204 {
		t.Fatalf("activation after wrong tokens %+v", got)
	}
	if _, err := os.Stat(filepath.Join(f.s.cfg.Data, ActivationFile)); !os.IsNotExist(err) {
		t.Error("activation file removed")
	}
	f.code(t, "activation reused", f.auth("redeem", `{"token":"`+token+`","password":"`+pw+`","login":"first@example.invalid","email":"a@b"}`, pre), 422, CodeLinkInvalid)
	got := f.auth("login", `{"workspace":"`+EmptyWorkspace+`","login":"first@example.invalid","password":"`+pw+`"}`, pre)
	if got.status != 200 || got.body["user"] != EmptyAdmin {
		t.Fatalf("no-policy bootstrap admin %+v", got)
	}
	g := setup(t, SeedNone).ready()
	g.advance(time.Hour)
	gp := g.pre()
	g.code(t, "activation expired", g.auth("link", `{"token":"`+g.links()[""]+`"}`, gp), 422, CodeLinkInvalid)
	h := setup(t, SeedSample).ready()
	h.setPassword(AdminLogin, pw)
	h.s.bootWorkspace = "elsewhere"
	h.s.policy, _ = newPolicy(h)
	hp := h.pre()
	h.code(t, "no policy outside bootstrap workspace", h.auth("login", loginBodyFor(AdminLogin, pw), hp), 401, string(commands.Unauthenticated))
	if len(h.s.sessions.byID) != 0 {
		t.Error("policy refusal ends session")
	}
}

func TestLimiterAndSlots(t *testing.T) {
	f := setup(t, SeedSample).ready()
	pre := f.pre()
	for i := 0; i < 30; i++ {
		f.auth("link", `{"token":"x"}`, pre)
	}
	got := f.auth("login", loginBodyFor(AdminLogin, pw), pre)
	if got.status != 429 || got.header.Get("Retry-After") != "2" || got.body["retry_after_ms"] == nil {
		t.Fatalf("limited %+v", got)
	}
	f.advance(2 * time.Second)
	f.code(t, "one token", f.auth("link", `{"token":"x"}`, pre), 422, CodeLinkInvalid)
	f.code(t, "empty again", f.auth("link", `{"token":"x"}`, pre), 429, string(commands.RateLimited))
	f.advance(time.Hour)
	f.setPassword(AdminLogin, pw)
	for i := 0; i < 40; i++ {
		if r := f.auth("login", loginBodyFor(AdminLogin, pw), pre); r.status != 200 {
			t.Fatalf("successes refund %d %+v", i, r)
		}
	}
	l := &limiter{}
	now := time.Unix(0, 0)
	for i := 0; i < 30; i++ {
		l.take(now)
	}
	l.refund()
	if ok, _ := l.take(now); !ok {
		t.Error("refund")
	}
	for i := 0; i < 5; i++ {
		l.refund()
	}
	now = now.Add(time.Hour)
	n := 0
	for ok, _ := l.take(now); ok; ok, _ = l.take(now) {
		n++
	}
	if n != 30 {
		t.Errorf("capacity %d", n)
	}
	for i := 0; i < derivations; i++ {
		f.s.slots <- struct{}{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if f.s.derive(ctx, func() { t.Error("ran without a slot") }) {
		t.Error("cancelled wait")
	}
	waiting, stop := context.WithCancel(context.Background())
	result := make(chan bool)
	go func() { result <- f.s.derive(waiting, func() { t.Error("ran without a slot") }) }()
	time.Sleep(10 * time.Millisecond)
	stop()
	if <-result {
		t.Error("wait cancelled while queued")
	}
	if got := f.cancelled("login", loginBodyFor(AdminLogin, pw), pre); got.status != 503 {
		t.Errorf("cancelled login %d", got.status)
	}
	tok := f.links()[Admin2Login]
	f.code(t, "cancelled redeem", f.cancelled("redeem", `{"token":"`+tok+`","password":"`+pw+`"}`, pre), 503, string(commands.DependencyUnavailable))
	for i := 0; i < derivations; i++ {
		<-f.s.slots
	}
	if got := f.auth("link", `{"token":"`+tok+`"}`, pre); got.status != 200 {
		t.Error("cancelled redeem released its reservation")
	}
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{"A@B.C": "a@b.c", "abc": "abc", strings.Repeat("a", 200): strings.Repeat("a", 200), "\u212Aelvin": "\u212Aelvin", "ZÉ": "zÉ"} {
		if got, ok := NormalizeLogin(in); !ok || got != want {
			t.Errorf("%s", in)
		}
	}
	for _, in := range []string{"ab", strings.Repeat("a", 201), "a\x7fb", "a\u0085b", "\xff\xfe\xfd"} {
		if _, ok := NormalizeLogin(in); ok {
			t.Errorf("%q accepted", in)
		}
	}
	if validWorkspace("") || validWorkspace(strings.Repeat("a", 129)) || !validWorkspace(strings.Repeat("a", 128)) || !validWorkspace("A-z.0_9:") {
		t.Error("workspace")
	}
	if retryAfter(1) != "1" || retryAfter(time.Second) != "1" || retryAfter(time.Second+1) != "2" {
		t.Error("retry after")
	}
}

func (f world) cancelled(path, body string, pre *http.Cookie) reply {
	f.t.Helper()
	ctx, stop := context.WithCancel(context.Background())
	stop()
	r := httptest.NewRequestWithContext(ctx, "POST", "/api/v1/auth/"+path, strings.NewReader(body))
	r.Host = "127.0.0.1:8820"
	r.Header.Set("Origin", "http://127.0.0.1:5173")
	r.AddCookie(pre)
	r.Header.Set(csrfHeader, pre.Value)
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, r)
	out := reply{status: rec.Code, header: rec.Header(), raw: rec.Body.String()}
	_ = json.Unmarshal(rec.Body.Bytes(), &out.body)
	return out
}
