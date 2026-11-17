package server

import (
	"io"
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
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

type seen struct {
	header http.Header
	host   string
	url    string
}

func upstream(t *testing.T, got *seen) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.header, got.host, got.url = r.Header.Clone(), r.Host, r.URL.String()
		w.Header().Set("Set-Cookie", "leak=1")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ok":true}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func readerWorld(t *testing.T, target string) world {
	now := start
	mu := &sync.Mutex{}
	s, err := New(Config{Data: t.TempDir() + "/data", Seed: SeedSample, Passwords: fastPasswords(), Readers: map[string]string{"/api/focus-overview": target},
		Clock: func() time.Time { mu.Lock(); defer mu.Unlock(); return now }})
	if err != nil {
		t.Fatal(err)
	}
	s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820})
	f := world{t: t, s: s, now: &now, mu: mu, value: new(int)}
	return f.ready()
}

func TestReaderGate(t *testing.T) {
	var got seen
	up := upstream(t, &got)
	f := readerWorld(t, up.URL)
	secret, err := os.ReadFile(filepath.Join(f.s.cfg.Data, ReaderSecretFile))
	info, _ := os.Stat(filepath.Join(f.s.cfg.Data, ReaderSecretFile))
	if err != nil || strings.TrimSpace(string(secret)) != f.s.readerSecret || len(f.s.readerSecret) != 43 || info.Mode().Perm() != 0o600 {
		t.Fatalf("secret file %v", err)
	}
	f.code(t, "no session", f.do("GET", "/api/focus-overview?provider=aws", "", nil), 401, string(commands.Unauthenticated))
	if got.url != "" {
		t.Fatal("forwarded without a session")
	}
	cookie, _ := f.login()
	r := f.do("GET", "/api/focus-overview?provider=aws", "", func(r *http.Request) {
		r.AddCookie(cookie)
		r.Header.Set("Origin", "http://127.0.0.1:5173")
		r.Header.Set("Sec-Fetch-Site", "same-origin")
		r.Header.Set("X-CSRF-Token", "x")
		r.Header.Set("Accept", "application/json")
		r.Header.Set("X-HB-Gate-Secret", "forged")
		r.Header.Set("X-Forwarded-For", "10.0.0.9")
		r.Header.Set("X-Forwarded-Host", "evil.example")
		r.Header.Set("Connection", "keep-alive, X-Custom")
		r.Header.Set("X-Custom", "hop")
		r.Header.Set("Upgrade", "websocket")
	})
	if r.status != 200 || r.raw != `{"ok":true}` {
		t.Fatalf("forwarded %+v", r)
	}
	if got.url != "/api/focus-overview?provider=aws" || got.host != strings.TrimPrefix(up.URL, "http://") {
		t.Errorf("target %s %s", got.url, got.host)
	}
	if got.header.Get("X-HB-Local-Client") != "1" || got.header.Get("X-HB-Gate-Secret") != f.s.readerSecret || got.header.Get("Accept") != "application/json" {
		t.Errorf("added headers %v", got.header)
	}
	for _, name := range []string{"Cookie", "Origin", "Sec-Fetch-Site", "X-Csrf-Token", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Custom", "Upgrade"} {
		if got.header.Get(name) != "" {
			t.Errorf("leaked %s", name)
		}
	}
	if r.header.Get("Set-Cookie") != "" || r.header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("response headers filtered")
	}
	f.code(t, "post not routed", f.do("POST", "/api/focus-overview", "", func(r *http.Request) { r.AddCookie(cookie) }), 404, string(commands.UnsupportedCommand))
	f.advance(31 * time.Minute)
	f.code(t, "ended session", f.do("GET", "/api/focus-overview", "", func(r *http.Request) { r.AddCookie(cookie) }), 401, string(commands.Unauthenticated))
}

func TestReaderPermissionAndUpstream(t *testing.T) {
	var got seen
	up := upstream(t, &got)
	f := readerWorld(t, up.URL)
	if err := f.s.admin.RevokeGrant("reader-"+SampleAdmin, 1); err != nil {
		t.Fatalf("revoke %v", err)
	}
	cookie, _ := f.login()
	f.code(t, "no grant", f.do("GET", "/api/focus-overview", "", func(r *http.Request) { r.AddCookie(cookie) }), 403, string(commands.PermissionDenied))
	if got.url != "" {
		t.Error("forwarded without permission")
	}
	down := readerWorld(t, "http://127.0.0.1:1")
	c2, _ := down.login()
	down.code(t, "upstream down", down.do("GET", "/api/focus-overview", "", func(r *http.Request) { r.AddCookie(c2) }), 503, string(commands.DependencyUnavailable))
	r := down.do("GET", "/api/focus-overview", "", func(r *http.Request) { r.AddCookie(c2) })
	if r.body["request_id"] == "" || r.header.Get("X-Request-Id") == "" {
		t.Errorf("request id on 503 %+v", r)
	}
	head := down.do("HEAD", "/api/focus-overview", "", func(r *http.Request) { r.AddCookie(c2) })
	if head.status != 404 {
		t.Errorf("head %d", head.status)
	}
}

func TestReaderConfig(t *testing.T) {
	if len(DefaultReaders()) != 6 || DefaultReaders()["/api/focus-receipts"] != "http://127.0.0.1:8810" || DefaultReaders()["/api/focus-overview"] != "http://127.0.0.1:8806" {
		t.Error("defaults")
	}
	for name, readers := range map[string]map[string]string{
		"path": {"/api/other": "http://127.0.0.1:1"}, "scheme": {"/api/focus-overview": "https://127.0.0.1:1"},
		"remote": {"/api/focus-overview": "http://10.0.0.1:1"}, "path in target": {"/api/focus-overview": "http://127.0.0.1:1/x"},
		"query": {"/api/focus-overview": "http://127.0.0.1:1?a=1"}, "user": {"/api/focus-overview": "http://u@127.0.0.1:1"},
		"bad url": {"/api/focus-overview": "http://[::1"}, "localhost": {"/api/focus-overview": "http://localhost:1"},
		"ipv6": {"/api/focus-overview": "http://[::1]:1"}, "no port": {"/api/focus-overview": "http://127.0.0.1"},
	} {
		if _, err := New(Config{Data: "x", Seed: SeedSample, Readers: readers}); err != ErrConfig {
			t.Errorf("%s accepted", name)
		}
	}
	none, err := New(Config{Data: t.TempDir() + "/d", Seed: SeedNone, Readers: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	none.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1})
	if err := none.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(none.cfg.Data, ReaderSecretFile)); !os.IsNotExist(err) {
		t.Error("no readers, no secret")
	}
	dir := t.TempDir() + "/blocked"
	os.Mkdir(dir, 0o700)
	os.MkdirAll(filepath.Join(dir, ReaderSecretFile, "inner"), 0o700)
	blocked, _ := New(Config{Data: dir, Seed: SeedNone})
	if blocked.Start() == nil {
		t.Error("cannot replace the secret")
	}
	first := setup(t, SeedSample)
	dataDir := first.s.cfg.Data
	first.ready()
	old, _ := os.ReadFile(filepath.Join(dataDir, ReaderSecretFile))
	first.s.Close()
	again, _ := New(Config{Data: dataDir, Seed: SeedSample, Passwords: fastPasswords()})
	if err := again.Start(); err != nil {
		t.Fatal(err)
	}
	fresh, _ := os.ReadFile(filepath.Join(dataDir, ReaderSecretFile))
	if string(old) == string(fresh) || strings.TrimSpace(string(fresh)) != again.readerSecret {
		t.Error("restart replaces the secret")
	}
	if p, _ := first.s.policy.Policy(SampleWorkspace); p.Number == 0 {
		t.Error("policy")
	}
	_ = gate.View
}
