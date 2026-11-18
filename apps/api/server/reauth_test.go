package server

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

func (f world) authenticatedAt(a actor) time.Time {
	f.t.Helper()
	found, ok := f.s.sessions.get(a.verifier(f))
	if !ok {
		f.t.Fatal("no session")
	}
	return found.authenticated
}

func TestReauthenticateRefreshesOnlyThisSession(t *testing.T) {
	f, a := probeWorld(t)
	b := f.signedIn(AdminLogin, pw)
	f.advance(10 * time.Minute)
	before := f.s.counter
	got := f.reauth(a, pw)
	if got.status != 204 || got.header.Get("Cache-Control") != "no-store" || got.raw != "" || f.s.counter != before {
		t.Fatalf("reauthenticate %+v", got)
	}
	if !f.authenticatedAt(a).Equal(start.Add(10*time.Minute)) || !f.authenticatedAt(b).Equal(start) {
		t.Error("only this session is refreshed")
	}
	created, _ := f.s.sessions.get(a.verifier(f))
	if !created.created.Equal(start) {
		t.Error("sign-in time changed")
	}
	f.advance(5*time.Minute + time.Nanosecond)
	if got := probeCall(f, a, "k1", "req-1"); got.status != 200 {
		t.Errorf("refreshed session %+v", got)
	}
	refused(t, "other session", probeCall(f, b, "k2", "req-2"))
	me := f.me(a).body
	if me["authenticated_at"] != stamp(start.Add(10*time.Minute)) || me["signed_in_at"] != stamp(start) {
		t.Errorf("me %+v", me)
	}
	page := f.policyRead(a).body
	if page["authenticated_at"] != stamp(start.Add(10*time.Minute)) || page["signed_in_at"] != stamp(start) || page["act_until"] != stamp(start.Add(25*time.Minute)) {
		t.Errorf("policy page %+v", page)
	}
	if f.s.passwords.Verify(LoginKey(AdminLogin), pw) != passwords.ResultOK {
		t.Error("password unchanged")
	}
}

func TestReauthenticateRefusals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	f.code(t, "no session", f.reauth(actor{}, pw), 401, string(commands.Unauthenticated))
	f.code(t, "bad csrf", f.reauth(actor{a.cookie, "wrong"}, pw), 403, CodeCSRF)
	for name, body := range map[string]string{"empty": `{}`, "extra": `{"password":"x","other":"y"}`, "wrong key": `{"current":"x"}`, "array": `[]`, "number": `{"password":1}`,
		"trailing": `{"password":"x"} {}`, "duplicate": `{"password":"x","password":"y"}`} {
		f.code(t, name, f.selfPost(a, "reauthenticate", body), 422, string(commands.InvalidRequest))
	}
	f.code(t, "too large", f.selfPost(a, "reauthenticate", `{"password":"`+strings.Repeat("x", 2<<20)+`"}`), 413, CodeBodyTooLarge)
	if _, used := f.s.failures[a.verifier(f)]; used {
		t.Error("shape refusals used attempts")
	}
	got := f.reauth(a, "wrong password here")
	f.code(t, "wrong", got, 422, CodeWrongPassword)
	if got.body["request_id"] != got.header.Get("X-Request-Id") || f.s.failures[a.verifier(f)] != 1 {
		t.Errorf("wrong %+v %v", got, f.s.failures)
	}
	if f.reauth(a, pw).status != 204 {
		t.Fatal("right password")
	}
	if _, kept := f.s.failures[a.verifier(f)]; kept {
		t.Error("success keeps the attempt count")
	}
}

func TestFifthWrongAttemptEndsTheSession(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	other := f.signedIn(MemberLogin, pw)
	f.code(t, "change 1", f.selfPost(a, "password", pwBody("wrong", newPw)), 422, CodeWrongPassword)
	f.code(t, "change 2", f.selfPost(a, "password", pwBody("wrong", newPw)), 422, CodeWrongPassword)
	f.code(t, "reauth 3", f.reauth(a, "wrong"), 422, CodeWrongPassword)
	f.code(t, "reauth 4", f.reauth(a, "wrong"), 422, CodeWrongPassword)
	before := f.s.counter
	f.code(t, "reauth 5", f.reauth(a, "wrong"), 401, string(commands.Unauthenticated))
	if f.s.counter != before+1 {
		t.Errorf("counter %d -> %d", before, f.s.counter)
	}
	f.code(t, "ended", f.me(a), 401, string(commands.Unauthenticated))
	if f.me(other).status != 200 {
		t.Error("another session ended")
	}
	if _, kept := f.s.failures[a.verifier(f)]; kept {
		t.Error("count kept")
	}
	b := f.signedIn(MemberLogin, pw)
	f.s.failures[b.verifier(f)] = selfAttempts
	before = f.s.counter
	f.code(t, "no attempts left", f.reauth(b, pw), 401, string(commands.Unauthenticated))
	if f.s.counter != before+1 || f.me(b).status != 401 {
		t.Error("no attempts left ends the session")
	}
}

func TestReauthenticateRaces(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	f.s.pause = func() { f.s.dir.generations[MemberLogin]++ }
	f.code(t, "generation", f.reauth(a, pw), 409, string(commands.RevisionConflict))
	f.s.pause = func() { f.s.identity.Boundary().EndSession(a.verifier(f)) }
	f.code(t, "suspended or ended in identity", f.reauth(a, pw), 401, string(commands.Unauthenticated))
	a = f.signedIn(MemberLogin, pw)
	f.s.pause = func() { f.s.sessions.drop(a.verifier(f)) }
	f.code(t, "dropped during derivation", f.reauth(a, pw), 401, string(commands.Unauthenticated))
	a = f.signedIn(MemberLogin, pw)
	f.s.failures[a.verifier(f)] = selfAttempts - 1
	before := f.s.counter
	f.code(t, "wrong after the session ended", f.reauth(a, "wrong"), 401, string(commands.Unauthenticated))
	if f.s.counter != before {
		t.Errorf("liveness first bumped the counter %d -> %d", before, f.s.counter)
	}
	a = f.signedIn(MemberLogin, pw)
	signed := f.authenticatedAt(a)
	f.advance(5 * time.Minute)
	f.s.pause = nil
	f.s.fault = func(at string) bool { return at == "reauthenticate" }
	f.code(t, "session gone at the atomic setter", f.reauth(a, pw), 401, string(commands.Unauthenticated))
	f.s.fault = nil
	if got := f.authenticatedAt(a); !got.Equal(signed) || got.Equal(f.s.clock()) {
		t.Errorf("refreshed despite the refusal: %v", got)
	}
	a = f.signedIn(MemberLogin, pw)
	f.setPassword(AdminLogin, pw)
	admin := f.signedIn(AdminLogin, pw)
	f.s.pause = func() {
		f.s.pause = nil
		if got := f.run(admin, "identity.suspend_user", SampleMember, nil); got.status != 200 {
			t.Errorf("suspend %+v", got)
		}
	}
	f.code(t, "suspended between reserve and commit", f.reauth(a, pw), 401, string(commands.Unauthenticated))
	if _, live := f.s.sessions.get(a.verifier(f)); live {
		t.Error("suspended session kept")
	}
	if f.s.sessions.reauthenticated(identity.Verifier{7}, f.s.clock()) {
		t.Error("absent session")
	}
}

func TestReauthenticateDeriveUnavailable(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", "/api/v1/me/reauthenticate", strings.NewReader(`{"password":"`+pw+`"}`))
	req.Host = "127.0.0.1:8820"
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	req.AddCookie(a.cookie)
	req.Header.Set(csrfHeader, a.csrf)
	w := httptest.NewRecorder()
	f.h.ServeHTTP(w, req)
	if w.Code != 503 || !strings.Contains(w.Body.String(), string(commands.DependencyUnavailable)) {
		t.Errorf("derive unavailable %d %s", w.Code, w.Body.String())
	}
	if _, used := f.s.failures[a.verifier(f)]; used {
		t.Error("attempt not given back")
	}
}

func TestMaximumAgeRenewedButBounded(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.applyPolicy("60", "0")
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	for i := 0; i < 24; i++ {
		f.advance(29 * time.Minute)
		if got := f.reauth(a, pw); got.status != 204 {
			t.Fatalf("round %d %+v", i, got)
		}
	}
	if f.me(a).status != 200 {
		t.Fatal("renewed maximum age")
	}
	f.advance(29 * time.Minute)
	f.code(t, "absolute lifetime", f.reauth(a, pw), 401, string(commands.Unauthenticated))
	b := f.signedIn(MemberLogin, pw)
	f.advance(29 * time.Minute)
	f.me(b)
	f.advance(29 * time.Minute)
	f.me(b)
	f.advance(3 * time.Minute)
	f.code(t, "maximum age passed before reauthentication", f.reauth(b, pw), 401, string(commands.Unauthenticated))
}

func TestReauthenticationCannotBeBlocked(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	pre := f.pre()
	for i := 0; i < limiterCapacity+5; i++ {
		f.auth("login", loginBodyFor("nobody@example.invalid", "wrong password here"), pre)
	}
	f.code(t, "limiter exhausted", f.auth("login", loginBodyFor(MemberLogin, pw), pre), 429, string(commands.RateLimited))
	if got := f.reauth(a, pw); got.status != 204 {
		t.Errorf("reauthentication with the limiter exhausted %+v", got)
	}
	f.advance(2 * time.Minute)
	for i := 0; i < 5; i++ {
		f.s.limit = &limiter{}
		f.code(t, "third-party guess", f.signin(MemberLogin, "wrong password here"), 401, string(commands.Unauthenticated))
	}
	f.s.limit = &limiter{}
	f.code(t, "sign-in locked", f.signin(MemberLogin, pw), 401, string(commands.Unauthenticated))
	locked := f.s.passwords.Verify(LoginKey(MemberLogin), pw)
	if got := f.reauth(a, pw); got.status != 204 || locked != passwords.ResultLocked {
		t.Errorf("reauthentication with the login locked %+v %s", got, locked)
	}
	f.s.limit = &limiter{}
	f.code(t, "sign-in still locked", f.signin(MemberLogin, pw), 401, string(commands.Unauthenticated))
	f.advance(15 * time.Minute)
	f.s.limit = &limiter{}
	b := f.signedIn(MemberLogin, pw)
	for i := 0; i < selfAttempts-1; i++ {
		f.code(t, "wrong reauthentication", f.reauth(b, "wrong password here"), 422, CodeWrongPassword)
	}
	f.s.limit = &limiter{}
	if f.signin(MemberLogin, pw).status != 200 {
		t.Error("wrong reauthentication locked sign-in")
	}
}

func TestReauthenticationNeverLeaksThePassword(t *testing.T) {
	logs := &syncWriter{}
	f := setup(t, SeedSample).ready()
	f.s.cfg.Log = captureLog(logs)
	f.register("probe", &probe{s: f.s, value: f.value}, true)
	f.grant(SampleAdmin)
	secret := "distinctive secret passphrase"
	f.setPassword(AdminLogin, secret)
	a := f.signedIn(AdminLogin, secret)
	f.advance(stepUpDefault + time.Nanosecond)
	bodies := []string{probeCall(f, a, "k1", "req-1").raw}
	for _, password := range []string{"wrong secret passphrase", secret} {
		got := f.reauth(a, password)
		bodies = append(bodies, got.raw, fmt.Sprint(got.header))
	}
	bodies = append(bodies, probeCall(f, a, "k1", "req-1").raw, f.me(a).raw, f.policyRead(a).raw, f.do("GET", "/api/v1/openapi.json", "", nil).raw)
	journal, _ := json.Marshal(f.s.commands.Journal(0, 1<<20))
	bodies = append(bodies, string(journal), logs.String())
	for _, body := range bodies {
		if strings.Contains(body, secret) || strings.Contains(body, "wrong secret") {
			t.Fatalf("password leaked: %s", body)
		}
	}
	declared := map[string]string{"reauthenticated": "level,msg,time,user", "reauthentication_failed": "level,msg,result,time,user"}
	for msg, keys := range declared {
		lines := logLines(t, logs.String(), msg)
		if len(lines) != 1 {
			t.Fatalf("%s lines %v", msg, lines)
		}
		got := []string{}
		for k := range lines[0] {
			got = append(got, k)
		}
		sort.Strings(got)
		if strings.Join(got, ",") != keys || lines[0]["user"] != SampleAdmin {
			t.Errorf("%s %v", msg, lines[0])
		}
	}
	f.s.failures[a.verifier(f)] = selfAttempts
	f.reauth(a, secret)
	if lines := logLines(t, logs.String(), "reauthentication_failed"); len(lines) != 2 || lines[1]["result"] != "ended" {
		t.Errorf("ended line %v", lines)
	}
}

func TestSignInAndSessionCarryLogin(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(AdminLogin, pw)
	got := f.auth("login", loginBodyFor("Admin@Example.Invalid", pw), f.pre())
	if got.status != 200 || got.body["login"] != AdminLogin {
		t.Fatalf("sign-in %+v", got)
	}
	var cookie *http.Cookie
	for _, c := range readCookies(got.header) {
		if c.Name == CookieName {
			cookie = c
		}
	}
	session := f.do("GET", "/api/v1/session", "", func(r *http.Request) { r.Header.Set(clientHeader, ConsoleClient); r.AddCookie(cookie) })
	if session.status != 200 || session.body["login"] != AdminLogin {
		t.Errorf("session %+v", session)
	}
	if me := f.me(actor{cookie, got.body["csrf_token"].(string)}); me.body["authenticated_at"] != stamp(start) {
		t.Errorf("me %+v", me.body)
	}
}

func TestReauthenticationRaces(t *testing.T) {
	f, admin := probeWorld(t)
	f.setPassword(MemberLogin, pw)
	members := []actor{f.signedIn(MemberLogin, pw), f.signedIn(MemberLogin, pw), f.signedIn(MemberLogin, pw)}
	var wg sync.WaitGroup
	for n := 0; n < 6; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for m := 0; m < 6; m++ {
				switch (n + m) % 6 {
				case 0:
					f.reauth(admin, pw)
				case 1:
					probeCall(f, admin, fmt.Sprintf("k-%d-%d", n, m), "req-1")
				case 2:
					f.reauth(members[m%3], pw)
				case 3:
					f.selfPost(members[m%3], "sessions/end", `{"session":"others"}`)
				case 4:
					f.selfPost(members[m%3], "password", pwBody("wrong", newPw))
				default:
					f.do("POST", "/api/v1/auth/logout", "", func(r *http.Request) { r.AddCookie(members[m%3].cookie); r.Header.Set(csrfHeader, members[m%3].csrf) })
				}
				f.advance(time.Minute)
			}
		}(n)
	}
	wg.Wait()
	if *f.value < 0 {
		t.Error("value")
	}
}

func TestReauthenticationTiming(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	f := setup(t, SeedSample)
	f.s.passwords, _ = passwords.New(f.s.clock, passwords.DefaultParams())
	f = f.ready()
	rec, _ := f.s.passwords.Hash(LoginKey(AdminLogin), pw)
	f.s.passwords.Put(LoginKey(AdminLogin), rec)
	runs := [2][]time.Duration{}
	cases := []struct {
		password string
		want     int
	}{{pw, 204}, {"wrong password here", 422}}
	for i := 0; i < 41; i++ {
		for k := range cases {
			n := (i + k) % len(cases)
			c := cases[n]
			a := f.as(AdminLogin)
			began := time.Now()
			if got := f.reauth(a, c.password); got.status != c.want {
				t.Fatalf("status %d", got.status)
			}
			runs[n] = append(runs[n], time.Since(began))
		}
	}
	median := func(d []time.Duration) time.Duration {
		sort.Slice(d, func(a, b int) bool { return d[a] < d[b] })
		return d[len(d)/2]
	}
	right, wrong := median(runs[0]), median(runs[1])
	if max(right, wrong)*2 > min(right, wrong)*3 {
		t.Errorf("timing differs: right %v wrong %v", right, wrong)
	}
}

func sourceFiles(t *testing.T) map[string]*ast.File {
	t.Helper()
	names, _ := filepath.Glob("*.go")
	out := map[string]*ast.File{}
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		out[name] = file
	}
	return out
}

func enclosing(file *ast.File, pos token.Pos) string {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || pos < fn.Pos() || pos >= fn.End() {
			continue
		}
		owner := ""
		if fn.Recv != nil {
			typ := fn.Recv.List[0].Type
			if star, ok := typ.(*ast.StarExpr); ok {
				typ = star.X
			}
			owner = typ.(*ast.Ident).Name
		}
		return owner + "." + fn.Name.Name
	}
	return ""
}

func TestSourceStepUpRules(t *testing.T) {
	assigned, setter, execute, holder, verify := []string{}, []string{}, []string{}, []string{}, []string{}
	positions := map[string]token.Pos{}
	for name, file := range sourceFiles(t) {
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.AssignStmt:
				for _, lhs := range x.Lhs {
					if sel, ok := lhs.(*ast.SelectorExpr); ok && sel.Sel.Name == "authenticated" {
						assigned = append(assigned, enclosing(file, x.Pos()))
					}
				}
			case *ast.KeyValueExpr:
				if id, ok := x.Key.(*ast.Ident); ok && id.Name == "authenticated" {
					assigned = append(assigned, enclosing(file, x.Pos()))
				}
			case *ast.SelectorExpr:
				where := name + ":" + enclosing(file, x.Pos())
				switch {
				case x.Sel.Name == "reauthenticated" && selectorOn(x.X, "sessions"):
					setter = append(setter, where)
				case x.Sel.Name == "Execute" && selectorOn(x.X, "commands"):
					execute = append(execute, where)
				case x.Sel.Name == "VerifyHolder":
					holder = append(holder, where)
					positions["holder"] = x.Pos()
				case x.Sel.Name == "Verify" && selectorOn(x.X, "passwords"):
					verify = append(verify, where)
				case x.Sel.Name == "self" && name == "reauth.go":
					positions["self"] = x.Pos()
				case x.Sel.Name == "reserveAttempt" && name == "reauth.go":
					positions["reserve"] = x.Pos()
				}
			}
			return true
		})
	}
	sort.Strings(assigned)
	sort.Strings(verify)
	if strings.Join(assigned, ",") != "sessions.put,sessions.reauthenticated" {
		t.Errorf("authenticated assigned in %v", assigned)
	}
	if strings.Join(setter, ",") != "reauth.go:Server.commitReauth" || strings.Join(execute, ",") != "commands.go:Server.execute" {
		t.Errorf("setter %v execute %v", setter, execute)
	}
	if strings.Join(holder, ",") != "reauth.go:Server.reauthenticate" || !(positions["self"] < positions["reserve"] && positions["reserve"] < positions["holder"]) {
		t.Errorf("holder %v %v", holder, positions)
	}
	if strings.Join(verify, ",") != "self.go:Server.changePassword,signin.go:Server.login" {
		t.Errorf("verify %v", verify)
	}
	file := sourceFiles(t)["reauth.go"]
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Info" {
			for _, arg := range call.Args {
				ast.Inspect(arg, func(m ast.Node) bool {
					if id, ok := m.(*ast.Ident); ok && (id.Name == "body" || id.Name == "result") {
						t.Errorf("a log call reads %s", id.Name)
					}
					return true
				})
			}
		}
		return true
	})
}

func TestReauthenticationLogOutcome(t *testing.T) {
	logs := &syncWriter{}
	f := setup(t, SeedSample).ready()
	f.s.cfg.Log = captureLog(logs)
	f.setPassword(MemberLogin, pw)
	a := f.signedIn(MemberLogin, pw)
	if f.reauth(a, pw).status != 204 {
		t.Fatal("reauthenticate")
	}
	ok, failed := logLines(t, logs.String(), "reauthenticated"), logLines(t, logs.String(), "reauthentication_failed")
	if len(ok) != 1 || ok[0]["user"] != SampleMember || len(failed) != 0 {
		t.Fatalf("success logs %v %v", ok, failed)
	}
	f.code(t, "wrong", f.reauth(a, "wrong password here"), 422, CodeWrongPassword)
	ok, failed = logLines(t, logs.String(), "reauthenticated"), logLines(t, logs.String(), "reauthentication_failed")
	if len(ok) != 1 || len(failed) != 1 || failed[0]["result"] != "wrong" || failed[0]["user"] != SampleMember {
		t.Errorf("wrong password logs %v %v", ok, failed)
	}
}
