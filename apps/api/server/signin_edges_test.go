package server

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

func TestSignInEdges(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if f.s.Passwords() == nil {
		t.Fatal("passwords")
	}
	pre := f.pre()
	for _, body := range []string{`{1:"x"}`, `{"login":1}`, `{"login":"a"`, `{"login":"a"}x`} {
		f.code(t, "decode "+body, f.auth("login", body, pre), 422, string(commands.InvalidRequest))
	}
	f.s.identity.Installation().CreateUser(identity.Operator{Reference: "t", Reason: "t"}, "lone", LoginTriple("lone@example.invalid"), "lone@example.invalid")
	rec, _ := f.s.passwords.Hash(LoginKey("lone@example.invalid"), pw)
	f.s.passwords.Put(LoginKey("lone@example.invalid"), rec)
	f.code(t, "no membership", f.auth("login", loginBodyFor("lone@example.invalid", pw), pre), 401, string(commands.Unauthenticated))
	if len(f.s.sessions.byID) != 0 {
		t.Error("access refusal ends session")
	}
	f.setPassword(AdminLogin, pw)
	for i := 0; i < 10; i++ {
		v := verifierOf(random(32))
		f.s.identity.Boundary().OpenSession(AdminTriple(), v, time.Minute, time.Minute)
	}
	f.code(t, "limit without own records", f.auth("login", loginBodyFor(AdminLogin, pw), pre), 401, string(commands.Unauthenticated))
	f.advance(2 * time.Minute)
	got := f.auth("login", loginBodyFor(AdminLogin, pw), pre)
	if got.status != 200 {
		t.Fatalf("login %+v", got)
	}
	f.s.sessions.byID[identity.Verifier{7}] = &session{verifier: identity.Verifier{7}, created: f.s.clock().Add(-13 * time.Hour)}
	f.auth("login", loginBodyFor(AdminLogin, pw), pre)
	if _, ok := f.s.sessions.byID[identity.Verifier{7}]; ok {
		t.Error("prune past absolute lifetime")
	}
	tok := f.links()[Admin2Login]
	f.s.pause = func() {
		f.s.pause = nil
		other, _ := f.s.passwords.Hash(LoginKey(Admin2Login), "another long password")
		f.s.passwords.Put(LoginKey(Admin2Login), other)
	}
	f.code(t, "link replaced while hashing", f.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`"}`, pre), 422, CodeLinkInvalid)
	g := setup(t, SeedNone).ready()
	gp := g.pre()
	token := g.links()[""]
	body := `{"token":"` + token + `","password":"` + pw + `","login":"first@example.invalid","email":"first@example.invalid"}`
	g.code(t, "cancelled activation", g.cancelled("redeem", body, gp), 503, string(commands.DependencyUnavailable))
	g.s.pause = func() { g.s.pause = nil; g.advance(time.Hour) }
	g.code(t, "expired while hashing", g.auth("redeem", body, gp), 422, CodeLinkInvalid)
	h := setup(t, SeedNone).ready()
	hp := h.pre()
	htoken := h.links()[""]
	hbody := `{"token":"` + htoken + `","password":"` + pw + `","login":"first@example.invalid","email":"first@example.invalid"}`
	if got := h.auth("redeem", hbody, hp); got.status != 204 {
		t.Fatal("activate")
	}
	h.s.boot.live = true
	h.code(t, "bootstrap consumed", h.auth("redeem", hbody, hp), 422, CodeLinkInvalid)
	k := setup(t, SeedSample)
	k = k.ready()
	if err := k.s.seedSecondAdmin(k.s.clock()); err == nil {
		t.Error("second admin twice")
	}
	full := fastPasswords()
	full.Links = 1
	q, _ := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Passwords: full})
	if q.Start() == nil {
		t.Error("links cap stops the seed")
	}
	f.code(t, "redeem without csrf", f.auth("redeem", `{"token":"x"}`, nil), 403, CodeCSRF)
	_ = passwords.PurposeSet
	_ = http.StatusOK
}

func TestSignInLiterals(t *testing.T) {
	if derivations != 4 || limiterCapacity != 30 || limiterRefill != 2*time.Second || SampleAdmin2 != "sample-admin-2" || setLinkValidity != 168*time.Hour {
		t.Error("constants")
	}
	f := setup(t, SeedSample).ready()
	if cap(f.s.slots) != 4 {
		t.Error("slots")
	}
	if p, err := f.s.policy.Policy(SampleWorkspace); err != nil || p.Revision.MaxAge != 12*time.Hour || len(p.Revision.Methods) != 1 {
		t.Errorf("policy %+v", p)
	}
	l := &limiter{}
	now := time.Unix(0, 0)
	if ok, wait := l.take(now); !ok || wait != 0 {
		t.Error("take returns no wait")
	}
	for i := 0; i < 29; i++ {
		l.take(now)
	}
	l.refund()
	if ok, _ := l.take(now); !ok {
		t.Error("one refund")
	}
	if ok, wait := l.take(now); ok || wait != 2*time.Second {
		t.Errorf("one refund gives one token %v", wait)
	}
	if ok, _ := l.take(now.Add(-time.Hour)); ok {
		t.Error("clock going back adds nothing")
	}
	for in, want := range map[string]bool{" ab": true, "a\u009fb": false, "a b": true, "a\u0080b": false, "a~b": true} {
		if _, ok := NormalizeLogin(in); ok != want {
			t.Errorf("login %q", in)
		}
	}
	for in, want := range map[string]bool{"!a": false, "aZ": true, "Za": true, "a!": false} {
		if validWorkspace(in) != want {
			t.Errorf("workspace %q", in)
		}
	}
	if retryAfter(11*time.Second) != "11" {
		t.Error("retry base")
	}
	pre := f.pre()
	if pre.Secure {
		t.Error("presession not secure over http")
	}
	tlsReply := f.do("GET", "/api/v1/auth/csrf", "", func(r *http.Request) { r.Header.Set(clientHeader, ConsoleClient); r.TLS = &tls.ConnectionState{} })
	if !strings.Contains(tlsReply.header.Get("Set-Cookie"), "Secure") {
		t.Error("presession secure over tls")
	}
	f.setPassword(AdminLogin, pw)
	before := f.s.counter
	dup := `{"workspace":"` + SampleWorkspace + `","workspace":"` + SampleWorkspace + `","login":"` + AdminLogin + `","password":"` + pw + `"}`
	f.code(t, "duplicate allowed key", f.auth("login", dup, pre), 422, string(commands.InvalidRequest))
	got := f.auth("login", loginBodyFor(AdminLogin, pw), pre)
	if f.s.counter != before+1 {
		t.Errorf("login counted %d %d", before, f.s.counter)
	}
	var session *http.Cookie
	for _, c := range readCookies(got.header) {
		session = c
	}
	f.do("POST", "/api/v1/auth/logout", "", func(r *http.Request) { r.AddCookie(session); r.Header.Set(csrfHeader, got.body["csrf_token"].(string)) })
	if f.s.counter != before+2 {
		t.Error("logout counted")
	}
	f.s.limit = &limiter{}
	for i := 0; i < 30; i++ {
		f.auth("link", `{"token":"x"}`, pre)
	}
	f.advance(time.Second)
	if r := f.auth("link", `{"token":"x"}`, pre); fmt.Sprint(r.body["retry_after_ms"]) != "1000" {
		t.Errorf("retry ms %v", r.body["retry_after_ms"])
	}
	f.advance(time.Hour)
	f.s.limit = &limiter{}
	c := f.s.counter
	f.setPassword(Admin2Login, pw)
	if f.s.counter != c+1 {
		t.Error("password set counted")
	}
	g := setup(t, SeedNone).ready()
	gp := g.pre()
	g.code(t, "garbage while live", g.auth("link", `{"token":"!!"}`, gp), 422, CodeLinkInvalid)
	token := g.links()[""]
	for name, email := range map[string]string{"empty": "", "no at": "abc", "long": strings.Repeat("a", 250) + "@b.cd"} {
		body := `{"token":"` + token + `","password":"` + pw + `","login":"first@example.invalid","email":"` + email + `"}`
		g.code(t, "email "+name, g.auth("redeem", body, gp), 422, CodePasswordForm)
	}
	g.code(t, "bad login", g.auth("redeem", `{"token":"`+token+`","password":"`+pw+`","login":"x","email":"a@b"}`, gp), 422, CodePasswordForm)
	gc := g.s.counter
	edge := strings.Repeat("a", 252) + "@b"
	g.code(t, "201-byte login", g.auth("redeem", `{"token":"`+token+`","password":"`+pw+`","login":"`+strings.Repeat("a", 201)+`","email":"a@b"}`, gp), 422, CodePasswordForm)
	if r := g.auth("redeem", `{"token":"`+token+`","password":"`+pw+`","login":"first@example.invalid","email":"`+edge+`"}`, gp); r.status != 204 || g.s.counter != gc+1 {
		t.Errorf("254-byte email activates and counts %+v", r)
	}
	h := setup(t, SeedSample).ready()
	h.setPassword(AdminLogin, pw)
	h.setPassword(Admin2Login, pw)
	hp := h.pre()
	h.auth("login", loginBodyFor(Admin2Login, pw), hp)
	h.advance(time.Second)
	for i := 0; i < 11; i++ {
		h.auth("login", loginBodyFor(AdminLogin, pw), hp)
	}
	found := false
	for _, s := range h.s.sessions.byID {
		found = found || s.login == Admin2Login
	}
	if !found {
		t.Error("another user's older session evicted")
	}
}

func TestNoPolicyConditions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.s.policy, _ = newPolicy(f)
	now := f.s.clock()
	b := f.s.identity.Boundary()
	f.s.identity.Installation().CreateUser(identity.Operator{Reference: "t", Reason: "t"}, "member-1", LoginTriple("member@example.invalid"), "member@example.invalid")
	admin, member, invite := verifierOf(random(32)), verifierOf(random(32)), verifierOf(random(32))
	b.OpenSession(AdminTriple(), admin, time.Minute, time.Minute)
	if err := f.s.identity.Workspace().Invite(admin, SampleWorkspace, "inv-member", identity.Member, invite, identity.Validity{NotBefore: now, NotAfter: now.Add(time.Hour)}, "member-1"); err != nil {
		t.Fatal(err)
	}
	b.OpenSession(LoginTriple("member@example.invalid"), member, time.Minute, time.Minute)
	if err := f.s.identity.Workspace().Accept(SampleWorkspace, "inv-member", invite, member); err != nil {
		t.Fatal(err)
	}
	b.EndSession(member)
	rec, _ := f.s.passwords.Hash(LoginKey("member@example.invalid"), pw)
	f.s.passwords.Put(LoginKey("member@example.invalid"), rec)
	pre := f.pre()
	f.code(t, "member without policy", f.auth("login", loginBodyFor("member@example.invalid", pw), pre), 401, string(commands.Unauthenticated))
	if len(f.s.sessions.byID) != 0 {
		t.Error("member refusal leaves a session")
	}
	f.setPassword(AdminLogin, pw)
	if r := f.auth("login", loginBodyFor(AdminLogin, pw), pre); r.status != 200 {
		t.Fatalf("admin without policy %+v", r)
	}
	g := setup(t, SeedSample).ready()
	g.setPassword(AdminLogin, pw)
	auth := authpolicy.Authentication{Method: authpolicy.MethodPassword, Factors: []string{authpolicy.FactorOTP}, At: g.s.clock()}
	adminP := func(user string) authpolicy.Principal {
		return authpolicy.Principal{User: user, Workspace: SampleWorkspace, Role: authpolicy.RoleAdministrator, Kind: authpolicy.KindHuman}
	}
	p, err := g.s.policy.Propose(authpolicy.Change{Kind: authpolicy.ChangeRevision, Revision: authpolicy.Revision{Workspace: SampleWorkspace,
		Methods: []string{authpolicy.MethodPassword}, MFA: authpolicy.MFAEveryone, MaxAge: 12 * time.Hour}}, adminP(SampleAdmin), auth)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := g.s.policy.Approve(p.Number, adminP(SampleAdmin2), auth); err != nil {
		t.Fatal(err)
	}
	gp := g.pre()
	g.code(t, "admin refused by policy", g.auth("login", loginBodyFor(AdminLogin, pw), gp), 401, string(commands.Unauthenticated))
	if len(g.s.sessions.byID) != 0 {
		t.Error("policy refusal leaves a session")
	}
}

func TestEvictionBounds(t *testing.T) {
	if retryMillis(1) != 1 || retryMillis(time.Millisecond) != 1 || retryMillis(time.Millisecond+1) != 2 {
		t.Error("retry millis")
	}
	for fakes, want := range map[int]int{19: 200, 20: 401} {
		f := setup(t, SeedSample).ready()
		f.setPassword(AdminLogin, pw)
		pre := f.pre()
		for i := 0; i < 10; i++ {
			f.advance(time.Second)
			if r := f.auth("login", loginBodyFor(AdminLogin, pw), pre); r.status != 200 {
				t.Fatal(r.raw)
			}
		}
		var first *session
		for _, s := range f.s.sessions.byID {
			if first == nil || s.created.Before(first.created) {
				first = s
			}
		}
		for i := 0; i < fakes; i++ {
			v := identity.Verifier{byte(i), 9, 9}
			f.s.sessions.byID[v] = &session{verifier: v, login: AdminLogin, created: start.Add(-time.Duration(fakes-i) * time.Minute)}
		}
		if r := f.auth("login", loginBodyFor(AdminLogin, pw), pre); r.status != want {
			t.Errorf("fakes %d status %d", fakes, r.status)
		}
		_, kept := f.s.sessions.byID[first.verifier]
		if want == 200 && kept {
			t.Error("oldest real session evicted first")
		}
		if want == 401 && !kept {
			t.Error("eviction bounded at 20")
		}
	}
}
