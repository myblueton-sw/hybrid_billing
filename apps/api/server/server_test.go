package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
)

func TestBoundary(t *testing.T) {
	f := setup(t, SeedSample).ready()
	for _, host := range []string{"127.0.0.1:8820", "localhost:8820"} {
		if got := f.do("GET", "/api/v1/health", "", func(r *http.Request) { r.Host = host }); got.status != 200 || got.body["status"] != "ok" || len(got.body) != 1 {
			t.Errorf("host %s %+v", host, got)
		}
	}
	for _, host := range []string{"evil.example:8820", "127.0.0.1:8821", "[::1]:8820", "127.0.0.1"} {
		f.code(t, "host "+host, f.do("GET", "/api/v1/health", "", func(r *http.Request) { r.Host = host }), 421, CodeMisdirected)
	}
	f.code(t, "no origin", f.do("POST", "/api/v1/commands/x", "{}", func(r *http.Request) { r.Header.Del("Origin") }), 403, CodeOrigin)
	f.code(t, "foreign origin", f.do("POST", "/api/v1/commands/x", "{}", func(r *http.Request) { r.Header.Set("Origin", "http://evil.example") }), 403, CodeOrigin)
	for _, site := range []string{"same-site", "cross-site"} {
		f.code(t, site, f.do("GET", "/api/v1/health", "", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", site) }), 403, CodeOrigin)
	}
	for _, site := range []string{"same-origin", "none"} {
		if got := f.do("GET", "/api/v1/health", "", func(r *http.Request) { r.Header.Set("Sec-Fetch-Site", site) }); got.status != 200 {
			t.Errorf("site %s", site)
		}
	}
	for _, origin := range DefaultOrigins {
		if got := f.do("POST", "/api/v1/commands/x", "{}", func(r *http.Request) { r.Header.Set("Origin", origin) }); got.status != 401 {
			t.Errorf("origin %s %d", origin, got.status)
		}
	}
	got := f.do("GET", "/api/v1/health", "", nil)
	for name := range got.header {
		if strings.HasPrefix(name, "Access-Control") {
			t.Error("cors header")
		}
	}
	if got.header.Get("Cache-Control") != "no-store" || got.header.Get("X-Content-Type-Options") != "nosniff" || got.header.Get("Referrer-Policy") != "no-referrer" ||
		len(got.header.Get("X-Request-Id")) != 22 {
		t.Errorf("headers %+v", got.header)
	}
	other := f.do("GET", "/api/v1/health", "", nil)
	if other.header.Get("X-Request-Id") == got.header.Get("X-Request-Id") {
		t.Error("request ids repeat")
	}
	f.code(t, "unknown path", f.do("GET", "/api/v2/x", "", nil), 404, string(commands.UnsupportedCommand))
	v6 := setup(t, SeedSample)
	if err := v6.s.Bind(&net.TCPAddr{IP: net.IPv6loopback, Port: 9000}); err != nil {
		t.Fatal(err)
	}
	v6 = v6.ready()
	for host, want := range map[string]int{"[::1]:9000": 200, "localhost:9000": 200, "127.0.0.1:9000": 421} {
		if got := v6.do("GET", "/api/v1/health", "", func(r *http.Request) { r.Host = host }); got.status != want {
			t.Errorf("v6 host %s %d", host, got.status)
		}
	}
	for _, addr := range []net.Addr{&net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 1}, &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}, &net.UnixAddr{Name: "x"}} {
		if v6.s.Bind(addr) == nil {
			t.Errorf("bind %v", addr)
		}
	}
}

func TestLoopback(t *testing.T) {
	for addr, want := range map[string]bool{"127.0.0.1:8820": true, "[::1]:8820": true, "localhost:1": true, "0.0.0.0:8820": false, "10.0.0.1:1": false,
		"127.0.0.2:1": false, "example.com:1": false, "8820": false, "[::ffff:127.0.0.1]:1": false} {
		if Loopback(addr) != want {
			t.Errorf("%s", addr)
		}
	}
}

func TestConfig(t *testing.T) {
	for name, cfg := range map[string]Config{
		"no data": {Seed: SeedSample}, "seed": {Data: "x", Seed: "other"}, "origins": {Data: "x", Seed: SeedSample, Origins: []string{"http://a:1", "http://a:2", "http://a:3", "http://a:4", "http://a:5"}},
		"bad origin": {Data: "x", Seed: SeedSample, Origins: []string{"127.0.0.1:5173"}}, "no port": {Data: "x", Seed: SeedSample, Origins: []string{"http://x"}},
		"port": {Data: "x", Seed: SeedSample, Origins: []string{"http://x:99999"}}, "empty host": {Data: "x", Seed: SeedSample, Origins: []string{"http://:80"}},
	} {
		if _, err := New(cfg); err != ErrConfig {
			t.Errorf("%s: %v", name, err)
		}
	}
	s, err := New(Config{Data: "x", Seed: SeedNone, Origins: []string{"https://console.local:443"}, Log: slog.New(slog.DiscardHandler)})
	if err != nil || !s.origins["https://console.local:443"] || len(s.origins) != 1 {
		t.Errorf("origin flag replaces defaults %v", err)
	}
	if s.Identity() == nil || s.Gate() == nil || s.Policy() == nil || s.Commands() == nil {
		t.Error("assembly")
	}
	if !limitsOK() {
		t.Error("limits")
	}
}

func limitsOK() bool {
	l := CommandLimits()
	sustained := int(l.Retention/l.Refill) + l.Burst
	return l.Workspaces*l.WorkspaceJournal <= l.Journal && l.Users*l.UserJournal <= l.WorkspaceJournal && l.Retention > l.Replay &&
		l.UserJournal >= sustained && l.UserOperations >= sustained && l.Retention >= 8*time.Hour && l.Refill <= time.Second
}

func TestSessionEndpoint(t *testing.T) {
	f := setup(t, SeedSample).ready()
	client := func(r *http.Request) { r.Header.Set(clientHeader, ConsoleClient) }
	f.code(t, "no client header", f.do("GET", "/api/v1/session", "", nil), 403, CodeOrigin)
	f.code(t, "no cookie", f.do("GET", "/api/v1/session", "", client), 401, string(commands.Unauthenticated))
	f.code(t, "bad cookie", f.do("GET", "/api/v1/session", "", func(r *http.Request) {
		client(r)
		r.AddCookie(&http.Cookie{Name: CookieName, Value: "!!"})
	}), 401, string(commands.Unauthenticated))
	f.code(t, "short cookie", f.do("GET", "/api/v1/session", "", func(r *http.Request) {
		client(r)
		r.AddCookie(&http.Cookie{Name: CookieName, Value: encode([]byte("short"))})
	}), 401, string(commands.Unauthenticated))
	cookie, csrf := f.login()
	if cookie.Path != "/api" || !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge != 43200 || len(cookie.Value) != 43 {
		t.Errorf("cookie %+v", cookie)
	}
	with := func(r *http.Request) {
		client(r)
		r.AddCookie(cookie)
	}
	a := f.do("GET", "/api/v1/session", "", with)
	b := f.do("GET", "/api/v1/session", "", with)
	if a.status != 200 || a.body["user"] != SampleAdmin || a.body["workspace"] != SampleWorkspace || a.body["role"] != "administrator" || a.body["csrf_token"] != csrf || b.body["csrf_token"] != csrf {
		t.Errorf("session %+v %+v", a, b)
	}
	f.advance(31 * time.Minute)
	f.code(t, "idle ended", f.do("GET", "/api/v1/session", "", with), 401, string(commands.Unauthenticated))
	if len(f.s.sessions.byID) != 0 {
		t.Error("ended session dropped")
	}
}

func TestCommands(t *testing.T) {
	f := setup(t, SeedSample).ready()
	p := &probe{s: f.s, value: f.value}
	f.register("probe", p, true)
	f.register("peek", reader{f.s}, false)
	f.grant(SampleAdmin)
	cookie, csrf := f.login()
	f.code(t, "no session", f.call("probe", mutation("k1", "r2"), nil, csrf), 401, string(commands.Unauthenticated))
	f.code(t, "csrf missing", f.call("probe", mutation("k1", "r2"), cookie, ""), 403, CodeCSRF)
	f.code(t, "csrf wrong", f.call("probe", mutation("k1", "r2"), cookie, csrf+"x"), 403, CodeCSRF)
	for _, body := range []string{"", "[]", `{"request_id":1}`, `{"request_id":"a","request_id":"b"}`, `{"other":"x"}`, `{"body":{"a":"1","a":"2"}}`, `{"body":[]}`,
		`{"body":{"a":1}}`, `{"body":{1}}`, `{} {}`, `{"request_id":"a"`, `{"body":{"a":"1"`, `{"body":`, `{"body":{"a"`, `{"x"`, `{1:2}`} {
		f.code(t, "shape "+body, f.call("probe", body, cookie, csrf), 422, string(commands.InvalidRequest))
	}
	f.code(t, "too large", f.call("probe", `{"rationale":"`+strings.Repeat("a", maxBodyBytes)+`"}`, cookie, csrf), 413, CodeBodyTooLarge)
	if f.s.counter != 2 {
		t.Fatalf("seed counter %d", f.s.counter)
	}
	got := f.call("probe", mutation("k1", "r2"), cookie, csrf)
	if got.status != 200 || got.body["status"] != "completed" || got.body["result_revision"] != "r3" || got.body["request_id"] != "req-1" || got.header.Get("X-Request-Id") != "req-1" ||
		got.body["business_state"] != "done" || got.body["replay"] != false || got.body["as_of"] == nil || f.s.counter != 3 {
		t.Fatalf("completed %+v counter %d", got, f.s.counter)
	}
	var rev string
	f.s.Read(func(r string) { rev = r })
	if rev != "r3" {
		t.Errorf("next read %s", rev)
	}
	again := f.call("probe", mutation("k1", "r2"), cookie, csrf)
	if again.status != 200 || again.body["replay"] != true || f.s.counter != 3 || *f.value != 1 {
		t.Errorf("replay %+v", again)
	}
	f.code(t, "stale", f.call("probe", mutation("k2", "r2"), cookie, csrf), 409, string(commands.RevisionConflict))
	f.code(t, "key reuse", f.call("probe", `{"request_id":"req-1","target_id":"t2","idempotency_key":"k1","expected_revision":"r3","body":{}}`, cookie, csrf), 409, string(commands.IdempotencyConflict))
	f.code(t, "missing target", f.call("probe", `{"request_id":"r","target_id":"missing","idempotency_key":"k9","expected_revision":"r3","body":{}}`, cookie, csrf), 403, string(commands.PermissionDenied))
	f.code(t, "unknown command", f.call("nope", mutation("k3", "r3"), cookie, csrf), 404, string(commands.UnsupportedCommand))
	if f.s.counter != 3 {
		t.Errorf("refusals do not count %d", f.s.counter)
	}
	read := f.call("peek", `{"request_id":"r","target_id":"t1","body":{}}`, cookie, csrf)
	if read.status != 200 || f.s.counter != 3 {
		t.Errorf("reading command %+v %d", read, f.s.counter)
	}
	p.named = true
	f.code(t, "named failure", f.call("probe", mutation("k4", "r3"), cookie, csrf), 422, string(commands.PrerequisiteFailed))
	if f.s.counter != 3 {
		t.Error("named failure counted")
	}
	p.named, p.fail = false, true
	f.code(t, "broken", f.call("probe", mutation("k5", "r3"), cookie, csrf), 500, string(commands.InternalError))
	if f.s.counter != 4 {
		t.Errorf("reached apply counts %d", f.s.counter)
	}
	p.fail, p.panicky = false, true
	f.code(t, "panic", f.call("probe", mutation("k6", "r4"), cookie, csrf), 500, string(commands.InternalError))
	if f.s.counter != 5 {
		t.Errorf("panic counts %d", f.s.counter)
	}
	p.panicky, p.accepted = false, true
	if got := f.call("probe", mutation("k7", "r5"), cookie, csrf); got.status != 202 || got.body["job_id"] != "job-1" || f.s.counter != 6 {
		t.Errorf("accepted %+v", got)
	}
	f.advance(31 * time.Minute)
	f.code(t, "ended", f.call("probe", mutation("k8", "r6"), cookie, csrf), 401, string(commands.Unauthenticated))
}

func TestRateAndAudit(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.register("probe", &probe{s: f.s, value: f.value}, true)
	f.register("peek", reader{f.s}, false)
	f.grant(SampleAdmin)
	cookie, csrf := f.login()
	limited := false
	for i := 0; i < 70 && !limited; i++ {
		got := f.call("peek", `{"request_id":"r","target_id":"t1","body":{}}`, cookie, csrf)
		if got.status == 429 {
			limited = true
			if got.header.Get("Retry-After") != "1" || got.body["retry_after_ms"] == nil {
				t.Errorf("retry after %+v", got)
			}
		}
	}
	if !limited {
		t.Fatal("burst")
	}
	g := setup(t, SeedSample).ready()
	g.register("peek", reader{g.s}, false)
	g.grant(SampleAdmin)
	gc, gcsrf := g.login()
	for i := 0; i < 90*60; i++ {
		if got := g.call("peek", `{"request_id":"r","target_id":"t1","body":{}}`, gc, gcsrf); got.status != 200 {
			t.Fatalf("call %d %+v", i, got)
		}
		g.advance(time.Second)
	}
}

func TestSeed(t *testing.T) {
	f := setup(t, SeedNone)
	dir := f.s.cfg.Data
	f = f.ready()
	link := filepath.Join(dir, ActivationFile)
	info, err := os.Stat(link)
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("link %v", err)
	}
	data, _ := os.ReadFile(link)
	if !strings.HasPrefix(string(data), "http://127.0.0.1:5173/activate#") || len(strings.TrimSpace(string(data))) != len("http://127.0.0.1:5173/activate#")+43 {
		t.Errorf("link %q", data)
	}
	if d, _ := os.Stat(dir); d.Mode().Perm() != 0o700 {
		t.Error("data mode")
	}
	if f.s.Workspace() != EmptyWorkspace || f.s.counter != 1 {
		t.Errorf("empty start %d", f.s.counter)
	}
	f.s.Close()
	g, _ := New(Config{Data: dir, Seed: SeedNone})
	g.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1})
	if err := g.Start(); err != nil {
		t.Fatal(err)
	}
	again, _ := os.ReadFile(link)
	if bytes.Equal(again, data) {
		t.Error("old link replaced")
	}
	g.Close()
	h, _ := New(Config{Data: dir, Seed: SeedSample})
	if err := h.Start(); err != nil {
		t.Fatal(err)
	}
	sample, err := os.ReadFile(link)
	lines := strings.Split(strings.TrimSpace(string(sample)), "\n")
	if err != nil || len(lines) != 4 || !strings.HasPrefix(lines[0], AdminLogin+" http://127.0.0.1:5173/activate#") || !strings.HasPrefix(lines[1], Admin2Login+" ") || bytes.Equal(sample, again) {
		t.Errorf("sample links %q", sample)
	}
	ran := []string{}
	k, _ := New(Config{Data: t.TempDir() + "/d", Seed: SeedSample})
	k.AddStep("one", func() (bool, error) { ran = append(ran, "one"); return true, nil })
	k.AddStep("quiet", func() (bool, error) { ran = append(ran, "quiet"); return false, nil })
	k.AddStep("bad", func() (bool, error) { return false, os.ErrInvalid })
	k.AddStep("never", func() (bool, error) { ran = append(ran, "never"); return true, nil })
	if err := k.Start(); err != os.ErrInvalid || strings.Join(ran, ",") != "one,quiet" || k.counter != 24 {
		t.Errorf("steps %v %v %d", err, ran, k.counter)
	}
	m, _ := New(Config{Data: t.TempDir() + "/d", Seed: SeedNone})
	m.AddStep("ignored", func() (bool, error) { t.Error("sample steps run without seed"); return false, nil })
	m.Start()
}

func TestDataDirectory(t *testing.T) {
	base := t.TempDir()
	open := filepath.Join(base, "open")
	os.Mkdir(open, 0o755)
	os.Chmod(open, 0o755)
	file := filepath.Join(base, "file")
	os.WriteFile(file, nil, 0o600)
	link := filepath.Join(base, "link")
	ok := filepath.Join(base, "ok")
	os.Mkdir(ok, 0o700)
	os.Symlink(ok, link)
	for name, dir := range map[string]string{"group readable": open, "not a directory": file, "symbolic link": link} {
		s, _ := New(Config{Data: dir, Seed: SeedSample})
		if err := s.Start(); err != ErrConfig {
			t.Errorf("%s: %v", name, err)
		}
	}
	s, _ := New(Config{Data: filepath.Join(file, "child"), Seed: SeedSample})
	if s.Start() == nil {
		t.Error("child of file")
	}
	busy := filepath.Join(base, "busy")
	os.Mkdir(busy, 0o700)
	os.WriteFile(filepath.Join(busy, "other"), nil, 0o600)
	if err := writeExclusive(filepath.Join(busy, "other"), []byte("x")); err == nil {
		t.Error("exclusive create")
	}
	if err := writeExclusive(filepath.Join(busy, "missing", "x"), []byte("x")); err == nil {
		t.Error("missing dir")
	}
	locked := filepath.Join(base, "locked")
	os.Mkdir(locked, 0o700)
	os.WriteFile(filepath.Join(locked, ActivationFile), nil, 0o600)
	os.Chmod(locked, 0o500)
	defer os.Chmod(locked, 0o700)
	n, _ := New(Config{Data: locked, Seed: SeedNone})
	if n.Start() == nil {
		t.Error("cannot remove old link")
	}
}

func TestOpenAPI(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.register("probe", &probe{s: f.s, value: f.value}, true)
	f.grant(SampleAdmin)
	got := f.do("GET", "/api/v1/openapi.json", "", nil)
	var doc map[string]any
	if err := json.Unmarshal([]byte(got.raw), &doc); err != nil || got.status != 200 {
		t.Fatal(err)
	}
	if doc["servers"].([]any)[0].(map[string]any)["url"] != "/api/v1" || doc["info"].(map[string]any)["description"] != description {
		t.Error("servers")
	}
	codes := map[string]bool{}
	for _, c := range doc["components"].(map[string]any)["schemas"].(map[string]any)["Error"].(map[string]any)["properties"].(map[string]any)["code"].(map[string]any)["enum"].([]any) {
		codes[c.(string)] = true
	}
	listed := map[string]string{}
	for _, e := range doc["x-errors"].([]any) {
		listed[e.(map[string]any)["code"].(string)] = fmt.Sprint(e.(map[string]any)["status"])
	}
	for _, b := range boundaryCodes {
		if !codes[b.code] || listed[b.code] != fmt.Sprint(b.status) {
			t.Errorf("boundary %s", b.code)
		}
	}
	op := doc["paths"].(map[string]any)["/commands/probe"].(map[string]any)["post"].(map[string]any)
	props := op["requestBody"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)["properties"].(map[string]any)
	body := map[string]any{}
	for name := range props {
		switch name {
		case "body":
			body[name] = map[string]string{"note": "from document"}
		case "expected_revision":
			body[name] = "r2"
		default:
			body[name] = "x-" + name
		}
	}
	raw, _ := json.Marshal(body)
	cookie, csrf := f.login()
	if res := f.call("probe", string(raw), cookie, csrf); res.status != 200 {
		t.Errorf("document request %+v", res)
	}
	boundary := f.call("probe", "{}", cookie, "")
	for name := range boundary.body {
		if _, ok := doc["components"].(map[string]any)["schemas"].(map[string]any)["Error"].(map[string]any)["properties"].(map[string]any)[name]; !ok {
			t.Errorf("boundary error field %s", name)
		}
	}
	broken := &Server{commands: nil}
	if _, err := broken.Document(); err == nil {
		t.Error("broken document")
	}
	b := setup(t, SeedSample).ready()
	b.s.commands = nil
	b.code(t, "document error", b.do("GET", "/api/v1/openapi.json", "", nil), 500, string(commands.InternalError))
}

func TestPanicAndConcurrency(t *testing.T) {
	f := setup(t, SeedSample).ready()
	h := f.s.guard(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("x") }))
	rec := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Host = "127.0.0.1:8820"
	h.ServeHTTP(rec, r)
	if rec.Code != 500 {
		t.Error("panic recovered")
	}
	abort := f.s.guard(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	func() {
		defer func() {
			if recover() != http.ErrAbortHandler {
				t.Error("abort re-panicked")
			}
		}()
		abort.ServeHTTP(httptest.NewRecorder(), r)
	}()
	f.register("probe", &probe{s: f.s, value: f.value}, true)
	f.grant(SampleAdmin)
	cookie, csrf := f.login()
	var wg sync.WaitGroup
	for n := 0; n < 8; n++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for m := 0; m < 10; m++ {
				var rev string
				f.s.Read(func(r string) { rev = r })
				f.call("probe", mutation("k"+string(rune('a'+n))+string(rune('a'+m)), rev), cookie, csrf)
				f.do("GET", "/api/v1/session", "", func(r *http.Request) { r.Header.Set(clientHeader, ConsoleClient); r.AddCookie(cookie) })
			}
		}(n)
	}
	wg.Wait()
	if uint64(*f.value)+2 != f.s.counter {
		t.Errorf("every apply counted once: %d %d", *f.value, f.s.counter)
	}
}
