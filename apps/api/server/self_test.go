package server

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

const newPw = "another long sample password"

func (f world) signedIn(login, password string) actor {
	f.t.Helper()
	got := f.signin(login, password)
	if got.status != 200 {
		f.t.Fatalf("sign in %s %+v", login, got)
	}
	for _, c := range readCookies(got.header) {
		if c.Name == CookieName {
			return actor{c, got.body["csrf_token"].(string)}
		}
	}
	f.t.Fatal("no session cookie")
	return actor{}
}

func (f world) me(a actor) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/me", "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func (f world) selfPost(a actor, path, body string) reply {
	f.t.Helper()
	return f.do("POST", "/api/v1/me/"+path, body, func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func pwBody(current, next string) string {
	return `{"current":"` + current + `","new":"` + next + `"}`
}

func TestSelfReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	f.setPassword(AdminLogin, pw)
	first := f.signedIn(MemberLogin, pw)
	f.advance(time.Minute)
	second := f.signedIn(MemberLogin, pw)
	admin := f.signedIn(AdminLogin, pw)
	got := f.me(second)
	if got.status != 200 || got.body["user_id"] != SampleMember || got.body["login"] != MemberLogin || got.body["role"] != "member" ||
		got.body["workspace"] != SampleWorkspace || got.body["password_set"] != true || got.body["installation_administrator"] != false ||
		fmt.Sprint(got.body["idle_minutes"]) != "30" || fmt.Sprint(got.body["absolute_minutes"]) != "720" || got.body["signed_in_at"] != stamp(start.Add(time.Minute)) {
		t.Fatalf("read %+v", got)
	}
	if fmt.Sprint(got.body["identities"]) != fmt.Sprintf("[map[login:%s method:local-password]]", MemberLogin) {
		t.Errorf("identities %v", got.body["identities"])
	}
	rows := f.testSessionSnapshot(second)["sessions"].([]any)
	if len(rows) != 2 {
		t.Fatalf("sessions %v", rows)
	}
	current, other := rows[0].(map[string]any), rows[1].(map[string]any)
	handle := regexp.MustCompile(`^[0-9a-f]{16}$`)
	if current["current"] != true || other["current"] != false || !handle.MatchString(current["handle"].(string)) || !handle.MatchString(other["handle"].(string)) ||
		current["signed_in"] != stamp(start.Add(time.Minute)) || current["ends"] != stamp(start.Add(time.Minute+sessionAbsolute)) {
		t.Errorf("rows %+v", rows)
	}
	for _, secret := range []string{first.csrf, second.csrf, admin.csrf, first.cookie.Value, second.cookie.Value, admin.cookie.Value} {
		if strings.Contains(got.raw, secret) {
			t.Error("a token or csrf value leaked")
		}
	}
	if current := f.me(first).body["current_session"].(map[string]any); current["current"] != true || current["signed_in"] != stamp(start) {
		t.Errorf("the current session is unavailable %v", current)
	}
	if strings.Contains(got.raw, AdminLogin) {
		t.Error("another person's session leaked")
	}
	if a := f.me(admin); a.body["installation_administrator"] != true || a.body["role"] != "administrator" || countOf(a.body["recorded_session_count"]) != 1 {
		t.Errorf("admin %+v", a.body)
	}
	if fmt.Sprint(got.body["max_age_seconds"]) == "0" {
		t.Error("sample seed policy has no maximum age")
	}
	f.code(t, "no session", f.me(actor{}), 401, string(commands.Unauthenticated))
	f.code(t, "bad csrf", f.me(actor{second.cookie, "wrong"}), 403, CodeCSRF)
}

func TestSelfReadWithoutPolicy(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	empty, err := newPolicy(f)
	if err != nil {
		t.Fatal(err)
	}
	f.s.policy = empty
	if got := f.me(root); got.status != 200 || fmt.Sprint(got.body["max_age_seconds"]) != "0" {
		t.Errorf("no policy %+v", got)
	}
}

func TestChangePassword(t *testing.T) {
	var logs bytes.Buffer
	f := setupWith(t, func(c *Config) { c.Log = slog.New(slog.NewTextHandler(&logs, nil)) })
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	b := f.signedIn(MemberLogin, pw)
	before := f.s.counter
	got := f.selfPost(a, "password", pwBody(pw, newPw))
	if f.s.counter != before+1 {
		t.Errorf("counter %d -> %d", before, f.s.counter)
	}
	if got.status != 204 || got.header.Get("Cache-Control") != "no-store" {
		t.Fatalf("change %+v", got)
	}
	if f.me(a).status != 200 {
		t.Error("current session ended")
	}
	f.code(t, "other session", f.me(b), 401, string(commands.Unauthenticated))
	f.code(t, "old password", f.signin(MemberLogin, pw), 401, string(commands.Unauthenticated))
	if f.signin(MemberLogin, newPw).status != 200 {
		t.Error("new password refused")
	}
	if !strings.Contains(logs.String(), "password_changed") || strings.Contains(logs.String(), newPw) || strings.Contains(logs.String(), pw) {
		t.Errorf("logs %q", logs.String())
	}
	f.code(t, "same password", f.selfPost(a, "password", pwBody(newPw, newPw)), 422, CodePasswordForm)
	f.code(t, "short password", f.selfPost(a, "password", pwBody(newPw, "short")), 422, CodePasswordForm)
	f.code(t, "extra key", f.selfPost(a, "password", `{"current":"x","new":"y","other":"z"}`), 422, string(commands.InvalidRequest))
	f.code(t, "too large", f.selfPost(a, "password", `{"current":"`+strings.Repeat("x", 2<<20)+`","new":"y"}`), 413, CodeBodyTooLarge)
	f.code(t, "no session", f.selfPost(actor{}, "password", pwBody(newPw, pw)), 401, string(commands.Unauthenticated))
	f.code(t, "bad csrf", f.selfPost(actor{a.cookie, "wrong"}, "password", pwBody(newPw, pw)), 403, CodeCSRF)
}

func TestWrongCurrentPasswordEndsTheSessionAtTheFifth(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	b := f.signedIn(MemberLogin, pw)
	for i := 1; i < selfAttempts; i++ {
		f.code(t, fmt.Sprintf("wrong %d", i), f.selfPost(a, "password", pwBody("wrong", newPw)), 422, CodeWrongPassword)
	}
	before := f.s.counter
	f.code(t, "fifth", f.selfPost(a, "password", pwBody("wrong", newPw)), 401, string(commands.Unauthenticated))
	if f.s.counter != before+1 {
		t.Errorf("counter %d -> %d", before, f.s.counter)
	}
	f.code(t, "session ended", f.me(a), 401, string(commands.Unauthenticated))
	if _, kept := f.s.failures[a.verifier(f)]; kept {
		t.Error("failure count kept after the session ended")
	}
	f.code(t, "sign-in shares the lockout", f.signin(MemberLogin, pw), 401, string(commands.Unauthenticated))
	f.code(t, "locked with the right password", f.selfPost(b, "password", pwBody(pw, newPw)), 429, CodePasswordLocked)
	if f.s.failures[b.verifier(f)] != 0 || f.me(b).status != 200 {
		t.Errorf("a locked result counted against the session %v", f.s.failures)
	}
	f.advance(16 * time.Minute)
	b = f.signedIn(MemberLogin, pw)
	if got := f.selfPost(b, "password", pwBody(pw, newPw)); got.status != 204 {
		t.Fatalf("after the lockout %+v", got)
	}
	c := f.signedIn(MemberLogin, newPw)
	f.s.failures[c.verifier(f)] = selfAttempts
	before = f.s.counter
	f.code(t, "no attempts left", f.selfPost(c, "password", pwBody(newPw, pw)), 401, string(commands.Unauthenticated))
	if f.s.counter != before+1 {
		t.Errorf("counter %d -> %d", before, f.s.counter)
	}
	d := f.signedIn(MemberLogin, newPw)
	f.s.failures[identity.Verifier{3}] = 1
	f.selfPost(d, "password", pwBody("wrong", pw))
	if _, kept := f.s.failures[identity.Verifier{3}]; kept || f.s.failures[d.verifier(f)] != 1 {
		t.Errorf("stale counts %v", f.s.failures)
	}
}

func TestChangePasswordRacesAndFaults(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	f.s.pause = func() { f.s.dir.generations[MemberLogin]++ }
	f.code(t, "generation race", f.selfPost(a, "password", pwBody(pw, newPw)), 409, string(commands.RevisionConflict))
	f.s.pause = func() { f.s.identity.Boundary().EndSession(a.verifier(f)) }
	f.code(t, "access lost while hashing", f.selfPost(a, "password", pwBody(pw, newPw)), 401, string(commands.Unauthenticated))
	a = f.signedIn(MemberLogin, pw)
	f.s.pause = nil
	f.s.fault = func(at string) bool { return at == "password" }
	f.code(t, "store refused", f.selfPost(a, "password", pwBody(pw, newPw)), 500, string(commands.InternalError))
	f.s.fault = nil
	if f.me(a).status != 200 || f.signin(MemberLogin, pw).status != 200 {
		t.Error("a refused store changed something")
	}
	f.advance(13 * time.Hour)
	f.code(t, "too old", f.me(a), 401, string(commands.Unauthenticated))
	a = f.signedIn(MemberLogin, pw)
	f.s.pause = func() { f.s.sessions.drop(a.verifier(f)) }
	f.code(t, "session gone", f.selfPost(a, "password", pwBody(pw, newPw)), 401, string(commands.Unauthenticated))
	f.s.pause = nil
	b := f.signedIn(MemberLogin, pw)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", "/api/v1/me/password", strings.NewReader(pwBody(pw, newPw)))
	req.Host = "127.0.0.1:8820"
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	req.AddCookie(b.cookie)
	req.Header.Set(csrfHeader, b.csrf)
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, req)
	if w.Code != 503 {
		t.Errorf("derive unavailable %d %s", w.Code, w.Body.String())
	}
	for i := 0; i <= selfAttempts; i++ {
		f.s.limit.tokens, f.s.limit.last = 0, f.s.clock()
		limited := f.selfPost(b, "password", pwBody(pw, newPw))
		f.code(t, "limited", limited, 429, string(commands.RateLimited))
		if limited.body["retry"] != true || limited.header.Get("Retry-After") == "" {
			t.Errorf("limited body %+v", limited)
		}
	}
	if _, kept := f.s.failures[b.verifier(f)]; kept || f.me(b).status != 200 {
		t.Errorf("limited or unavailable calls used attempts %v", f.s.failures)
	}
	f.s.limit.tokens = limiterCapacity
	f.s.failures[b.verifier(f)] = 3
	f.s.releaseAttempt(b.verifier(f))
	if f.s.failures[b.verifier(f)] != 2 {
		t.Errorf("release %v", f.s.failures)
	}
	f.code(t, "end-sessions body too large", f.selfPost(b, "sessions/end", `{"session":"`+strings.Repeat("x", 2<<20)+`"}`), 413, CodeBodyTooLarge)
}

func TestEndOwnSessions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	f.setPassword(AdminLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	b := f.signedIn(MemberLogin, pw)
	c := f.signedIn(MemberLogin, pw)
	admin := f.signedIn(AdminLogin, pw)
	handles := map[bool]string{}
	for _, row := range f.testSessionSnapshot(a)["sessions"].([]any) {
		m := row.(map[string]any)
		if m["current"] == false && handles[false] == "" {
			handles[false] = m["handle"].(string)
		}
		if m["current"] == true {
			handles[true] = m["handle"].(string)
		}
	}
	adminHandle := f.me(admin).body["current_session"].(map[string]any)["handle"].(string)
	for name, body := range map[string]string{"current": handles[true], "foreign": adminHandle, "unknown": "0000000000000000", "empty": "", "extra": "x\",\"y\":\"z"} {
		f.code(t, name, f.selfPost(a, "sessions/end", `{"session":"`+body+`"}`), 422, string(commands.InvalidRequest))
	}
	before := f.s.counter
	got := f.selfPost(a, "sessions/end", `{"session":"`+handles[false]+`"}`)
	if f.s.counter != before+1 {
		t.Errorf("counter %d -> %d", before, f.s.counter)
	}
	if got.status != 202 || fmt.Sprint(got.body["count"]) != "1" || got.body["kind"] != "one" || len(got.body["not_included"].([]any)) != 0 {
		t.Fatalf("end one %+v", got)
	}
	if n := len(f.testSessionSnapshot(a)["sessions"].([]any)); n != 2 {
		t.Errorf("after one %d", n)
	}
	got = f.selfPost(a, "sessions/end", f.othersBody(a))
	if got.status != 202 || fmt.Sprint(got.body["count"]) != "1" || got.body["state"] != "complete" {
		t.Fatalf("end others %+v", got)
	}
	f.code(t, "b ended", f.me(b), 401, string(commands.Unauthenticated))
	f.code(t, "c ended", f.me(c), 401, string(commands.Unauthenticated))
	if f.me(admin).status != 200 || f.me(a).status != 200 {
		t.Error("an unrelated or the current session ended")
	}
	f.code(t, "no session", f.selfPost(actor{}, "sessions/end", `{"session":"others"}`), 401, string(commands.Unauthenticated))
}

func TestEndingASessionDropsItsPickups(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(AdminLogin, pw)
	a := f.signedIn(AdminLogin, pw)
	b := f.signedIn(AdminLogin, pw)
	got := f.run(b, "identity.create_user", usersTarget, map[string]string{"login": "pick@example.invalid", "email": "pick@example.invalid", "role": "member"})
	ref := got.body["fields"].(map[string]any)["link_ref"].(string)
	if _, ok := f.s.dir.pickups[ref]; !ok {
		t.Fatal("no pickup")
	}
	f.selfPost(a, "sessions/end", f.othersBody(a))
	if _, ok := f.s.dir.pickups[ref]; ok {
		t.Error("an ended session's pickup was kept")
	}
}
