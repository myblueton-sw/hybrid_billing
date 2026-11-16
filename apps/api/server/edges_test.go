package server

import (
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestEdges(t *testing.T) {
	bad := []Config{
		{Data: "x", Seed: SeedSample, Identity: identity.Limits{BootstrapValidity: 1}},
		{Data: "x", Seed: SeedSample, Kinds: []string{"bad kind"}},
		{Data: "x", Seed: SeedSample, Commands: commands.Limits{Fields: 1}},
	}
	for i, cfg := range bad {
		if _, err := New(cfg); err != ErrStart {
			t.Errorf("assembly %d: %v", i, err)
		}
	}
	f := setup(t, SeedSample).ready()
	if err := f.s.Register(commands.Definition{ID: "bad id!"}); err == nil {
		t.Error("register refused")
	}
	if f.s.mutating["bad id!"] {
		t.Error("refused definition recorded")
	}
	f.register("probe", &probe{s: f.s, value: f.value}, true)
	f.grant(SampleAdmin)
	cookie, csrf := f.login()
	for _, body := range []string{`{"body":{"a":"1"}x`, `{"request_id":"a"}x`, `{"body":{"a"x}}`, `{"body":{x}}`, `{"body":x}`, `x`, `{"rationale"x}`} {
		f.code(t, "token "+body, f.call("probe", body, cookie, csrf), 422, string(commands.InvalidRequest))
	}
	f.code(t, "two fields", f.call("probe", `{"request_id":"r","target_id":"t1","idempotency_key":"q","expected_revision":"r2","rationale":"x","body":{"b":"1","a":"2"}}`, cookie, csrf), 422, string(commands.InvalidRequest))
	if got := f.call("probe", `{"request_id":"r","target_id":"t1","idempotency_key":"q2","expected_revision":"r2"}`, cookie, csrf); got.status != 200 {
		t.Errorf("no body %+v", got)
	}
	ro := t.TempDir() + "/ro"
	n, _ := New(Config{Data: ro, Seed: SeedNone})
	if err := checkData(ro); err != nil {
		t.Fatal(err)
	}
	os.Chmod(ro, 0o500)
	if n.Start() == nil {
		t.Error("link write fails")
	}
	os.Chmod(ro, 0o700)
	v := verifierOf([]byte(strings.Repeat("v", 32)))
	f.s.sessions.byID[v] = &session{verifier: v, workspace: "bad workspace!", csrf: "c"}
	raw := []byte(strings.Repeat("v", 32))
	crafted := &http.Cookie{Name: CookieName, Value: encode(raw)}
	f.code(t, "invalid workspace", f.do("GET", "/api/v1/session", "", func(r *http.Request) { r.Header.Set(clientHeader, ConsoleClient); r.AddCookie(crafted) }), 500, string(commands.InternalError))
	g := setup(t, SeedSample).ready()
	gc, _ := g.login()
	g.s.identity = &identity.Registry{}
	g.code(t, "identity unavailable", g.do("GET", "/api/v1/session", "", func(r *http.Request) { r.Header.Set(clientHeader, ConsoleClient); r.AddCookie(gc) }), 503, string(commands.AuthorityUnavailable))
	h := setup(t, SeedSample).ready()
	h.s.sessions.byID = map[identity.Verifier]*session{}
	for i := 0; i < maxSessions; i++ {
		h.s.sessions.byID[identity.Verifier{byte(i), byte(i >> 8), 1}] = &session{}
	}
	if _, _, err := h.s.openFor(AdminTriple(), SampleWorkspace, false); err == nil {
		t.Error("session cap")
	}
	if _, _, err := h.s.openFor(identity.Triple{Connection: "x", Issuer: "y", Subject: "z"}, SampleWorkspace, true); err == nil {
		t.Error("unknown identity")
	}
	if _, ok := h.s.sessions.get(identity.Verifier{9}); ok {
		t.Error("get missing")
	}
	k, _ := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample})
	k.cfg.Data = "/dev/null/x"
	if k.Start() == nil {
		t.Error("data error")
	}
}

func TestCapRefuses(t *testing.T) {
	limits := CommandLimits()
	limits.UserJournal, limits.UserOperations, limits.Users = 3, 3, 20
	now := start
	s, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample, Commands: limits, Fields: map[string]gate.FieldClass{"state": gate.General}, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	f := world{t: t, s: s, now: &now, value: new(int)}
	s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820})
	f = f.ready()
	f.register("peek", reader{f.s}, false)
	f.grant(SampleAdmin)
	cookie, csrf := f.login()
	codes := []int{}
	for i := 0; i < 5; i++ {
		codes = append(codes, f.call("peek", `{"request_id":"r","target_id":"t1","body":{}}`, cookie, csrf).status)
	}
	if codes[0] != 200 || codes[4] != 503 {
		t.Errorf("cap %v", codes)
	}
	f.code(t, "cap code", f.call("peek", `{"request_id":"r","target_id":"t1","body":{}}`, cookie, csrf), 503, string(commands.DependencyUnavailable))
}

func TestNoOrphans(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if _, _, err := f.s.openFor(identity.Triple{Connection: "x", Issuer: "y", Subject: "z"}, SampleWorkspace, false); err == nil || len(f.s.sessions.byID) != 0 {
		t.Error("failed identity session leaves no record")
	}
	for i := 0; i < maxSessions; i++ {
		f.s.sessions.byID[identity.Verifier{byte(i), byte(i >> 8), 3}] = &session{}
	}
	for i := 0; i < 12; i++ {
		f.s.openFor(AdminTriple(), SampleWorkspace, false)
	}
	delete(f.s.sessions.byID, identity.Verifier{0, 0, 3})
	if _, _, err := f.s.openFor(AdminTriple(), SampleWorkspace, false); err != nil {
		t.Errorf("full store opened no identity sessions %v", err)
	}
}
