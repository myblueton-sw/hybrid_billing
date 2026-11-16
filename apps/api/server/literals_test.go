package server

import (
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
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestLiterals(t *testing.T) {
	if IdentityLimits() != (identity.Limits{BootstrapValidity: time.Hour, InvitationValidity: 168 * time.Hour, CredentialValidity: 720 * time.Hour,
		SessionIdle: 30 * time.Minute, SessionAbsolute: 12 * time.Hour, Reauthentication: 15 * time.Minute, Attempts: 5, PendingInvitations: 1000,
		SessionsPerUser: 10, CredentialsPerServiceUser: 10, PageSize: 100}) {
		t.Error("identity limits")
	}
	if CommandLimits() != (commands.Limits{Fields: 64, Value: 4096, Rationale: 2000, Operations: 1200000, Journal: 1200000, Workspaces: 2, Users: 20,
		WorkspaceJournal: 600000, UserJournal: 30000, WorkspaceOperations: 600000, UserOperations: 30000, Replay: time.Hour, Retention: 2 * time.Hour,
		Burst: 60, Refill: 250 * time.Millisecond}) {
		t.Error("command limits")
	}
	if strings.Join(DefaultOrigins, " ") != "http://127.0.0.1:5173 http://localhost:5173 http://127.0.0.1:4173 http://localhost:4173" {
		t.Error("origins")
	}
	if MaxHeaderBytes != 16384 || maxBodyBytes != 1048576 || maxSessions != 10000 || maxOrigins != 4 {
		t.Error("bounds")
	}
	four := []string{"http://a:1", "http://a:2", "http://a:3", "http://a:4"}
	if _, err := New(Config{Data: "x", Seed: SeedSample, Origins: four}); err != nil {
		t.Error("four origins")
	}
	for _, o := range []string{"http://x:5a", "http://", "ftp://x:1", "http://x:"} {
		if validOrigin(o) {
			t.Errorf("origin %s", o)
		}
	}
	if !validOrigin("https://x:65535") || validOrigin("https://x:65536") {
		t.Error("port range")
	}
}

func TestBodyAndTiming(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.register("probe", &probe{s: f.s, value: f.value}, true)
	f.grant(SampleAdmin)
	cookie, csrf := f.login()
	pad := func(n int) string {
		head := `{"request_id":"r","target_id":"t1","idempotency_key":"big","expected_revision":"r2","body":{"note":"`
		return head + strings.Repeat("a", n-len(head)-3) + `"}}`
	}
	if got := f.call("probe", pad(1<<20), cookie, csrf); got.status == 413 {
		t.Error("exact limit refused")
	}
	f.code(t, "one over", f.call("probe", pad(1<<20+1), cookie, csrf), 413, CodeBodyTooLarge)
	got := f.call("probe", `{"request_id":"r","target_id":"t1","idempotency_key":"n1","expected_revision":"r2","body":{"note":"kept"}}`, cookie, csrf)
	if got.body["fields"].(map[string]any)["note"] != "kept" {
		t.Errorf("fields reach the handler %+v", got.body)
	}
	f.code(t, "trailing", f.call("probe", `{"request_id":"r","target_id":"t1","idempotency_key":"n2","expected_revision":"r3","body":{"note":"x"}} {}`, cookie, csrf), 422, string(commands.InvalidRequest))
	for ms, want := range map[int64]string{1: "1", 999: "1", 1000: "1", 1001: "2", 1500: "2", 2000: "2", 2001: "3", 10001: "11"} {
		rec := httptest.NewRecorder()
		writeResult(rec, commands.Result{Request: "r", Err: &commands.Error{Code: commands.RateLimited, Status: 429, RetryAfter: time.Duration(ms) * time.Millisecond}})
		if rec.Header().Get("Retry-After") != want {
			t.Errorf("retry %d got %s", ms, rec.Header().Get("Retry-After"))
		}
	}
	rec := httptest.NewRecorder()
	writeResult(rec, commands.Result{Request: "r", Err: &commands.Error{Code: commands.PermissionDenied, Status: 403}})
	if rec.Header().Get("Retry-After") != "" || strings.Contains(rec.Body.String(), "retry_after") {
		t.Error("no retry header")
	}
	rec = httptest.NewRecorder()
	writeResult(rec, commands.Result{Request: "r", Status: commands.Completed, AsOf: time.Date(2026, 1, 2, 3, 4, 5, 60, time.FixedZone("x", 3600))})
	if !strings.Contains(rec.Body.String(), `"as_of":"2026-01-02T02:04:05.00000006Z"`) {
		t.Errorf("as_of %s", rec.Body.String())
	}
	doc := f.do("GET", "/api/v1/openapi.json", "", nil)
	if !strings.Contains(doc.raw, `"version": "0.1.0"`) || !strings.Contains(doc.raw, `"title": "Hybrid Billing pre-launch API"`) {
		t.Error("document version")
	}
	if len(f.s.sessions.byID) != 1 {
		t.Error("one session")
	}
	f.s.sessions.byID = map[identity.Verifier]*session{}
	for i := 0; i < 9999; i++ {
		f.s.sessions.byID[identity.Verifier{byte(i), byte(i >> 8), 2}] = &session{}
	}
	if _, _, err := f.s.openFor(AdminTriple(), SampleWorkspace, false); err != nil {
		t.Errorf("ten thousandth session %v", err)
	}
	g := setup(t, SeedNone)
	dir := g.s.cfg.Data
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, ActivationFile), []byte("old"), 0o600)
	g = g.ready()
	data, err := os.ReadFile(filepath.Join(dir, ActivationFile))
	if err != nil || string(data) == "old" || g.s.counter != 1 {
		t.Errorf("old link replaced and bootstrap armed %q %d", data, g.s.counter)
	}
	h := setup(t, SeedSample).ready()
	h.do("GET", "/api/v1/session", "", func(r *http.Request) {
		r.Header.Set(clientHeader, ConsoleClient)
		r.AddCookie(&http.Cookie{Name: CookieName, Value: encode([]byte(strings.Repeat("z", 32)))})
	})
}

func TestLargeAtEveryPosition(t *testing.T) {
	f := setup(t, SeedSample).ready()
	cookie, csrf := f.login()
	big := strings.Repeat(" ", 1<<20)
	for name, body := range map[string]string{"before object": big + "{}", "before key": "{" + big, "before body": `{"body":` + big, "before field": `{"body":{` + big} {
		f.code(t, name, f.call("probe", body, cookie, csrf), 413, CodeBodyTooLarge)
	}
}

func TestIdleBoundary(t *testing.T) {
	f := setup(t, SeedSample).ready()
	cookie, _ := f.login()
	with := func(r *http.Request) { r.Header.Set(clientHeader, ConsoleClient); r.AddCookie(cookie) }
	f.advance(29*time.Minute + 59*time.Second)
	if got := f.do("GET", "/api/v1/session", "", with); got.status != 200 {
		t.Fatal("alive before idle")
	}
	f.advance(30*time.Minute + time.Second)
	f.code(t, "idle", f.do("GET", "/api/v1/session", "", with), 401, string(commands.Unauthenticated))
}

func TestReviewTightening(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.code(t, "host before origin", f.do("POST", "/api/v1/commands/x", "{}", func(r *http.Request) {
		r.Host = "evil.example:8820"
		r.Header.Set("Origin", "http://evil.example")
	}), 421, CodeMisdirected)
	doc := f.do("GET", "/api/v1/openapi.json", "", nil)
	var parsed map[string]any
	if err := jsonUnmarshal(doc.raw, &parsed); err != nil {
		t.Fatal(err)
	}
	table := map[string]map[string]any{}
	for _, e := range parsed["x-errors"].([]any) {
		entry := e.(map[string]any)
		table[entry["code"].(string)] = entry
	}
	for code, rule := range moduleRules {
		entry := table[string(code)]
		if entry == nil || fmtInt(entry["status"]) != rule.status || entry["retry"] != rule.retry || entry["next_action"] != string(rule.next) {
			t.Errorf("rule %s %+v", code, entry)
		}
	}
	cookie, csrf := f.login()
	f.code(t, "large trailing", f.call("probe", `{"request_id":"r"}`+strings.Repeat(" ", 1<<20), cookie, csrf), 413, CodeBodyTooLarge)
	f.register("probe", &probe{s: f.s, value: f.value}, true)
	f.grant(SampleAdmin)
	got := f.call("probe", `{"target_id":"t1","idempotency_key":"e1","expected_revision":"r2","body":{}}`, cookie, csrf)
	if got.header.Get("X-Request-Id") == "" {
		t.Errorf("request id fallback %+v", got)
	}
	secure, _, err := f.s.openFor(AdminTriple(), SampleWorkspace, true)
	if err != nil || !secure.Secure {
		t.Error("secure cookie")
	}
}

func jsonUnmarshal(raw string, v any) error { return json.Unmarshal([]byte(raw), v) }

func fmtInt(v any) int {
	var n int
	fmt.Sscan(fmt.Sprint(v), &n)
	return n
}
