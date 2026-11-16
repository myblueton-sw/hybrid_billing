package server

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/authpolicy"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/passwords"
)

var start = time.Date(2026, 11, 16, 9, 0, 0, 0, time.UTC)

type world struct {
	t     testing.TB
	s     *Server
	h     http.Handler
	now   *time.Time
	mu    *sync.Mutex
	value *int
}

func setup(t testing.TB, seed string) world {
	t.Helper()
	now := start
	mu := &sync.Mutex{}
	s, err := New(Config{Data: t.TempDir() + "/data", Seed: seed, Fields: map[string]gate.FieldClass{"state": gate.General}, Passwords: fastPasswords(), Clock: func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return now
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820}); err != nil {
		t.Fatal(err)
	}
	value := 0
	w := world{t: t, s: s, now: &now, mu: mu, value: &value}
	return w
}

func (f world) ready() world {
	f.t.Helper()
	if err := f.s.Start(); err != nil {
		f.t.Fatal(err)
	}
	if f.s.cfg.Seed == SeedSample && f.s.counter != 9 {
		f.t.Fatalf("sample seed steps %d", f.s.counter)
	}
	f.s.counter = map[string]uint64{SeedSample: 2, SeedNone: 1}[f.s.cfg.Seed]
	f.h = f.s.Handler()
	return f
}

func (f world) advance(d time.Duration) {
	f.mu.Lock()
	*f.now = f.now.Add(d)
	f.mu.Unlock()
}

type probe struct {
	s        *Server
	value    *int
	fail     bool
	named    bool
	panicky  bool
	accepted bool
}

func (p *probe) Resolve(workspace, target string) (commands.Target, commands.Resolution) {
	if target == "missing" {
		return commands.Target{}, commands.NotFound
	}
	return commands.Target{Revision: p.s.Revision()}, commands.Found
}

func (p *probe) Apply(c commands.Context) (commands.Outcome, error) {
	switch {
	case p.named:
		return commands.Outcome{}, commands.Fail(commands.PrerequisiteFailed)
	case p.panicky:
		*p.value++
		panic("boom")
	case p.fail:
		*p.value++
		return commands.Outcome{}, errors.New("broken")
	}
	*p.value++
	if p.accepted {
		return commands.Outcome{Accepted: true, JobID: "job-1"}, nil
	}
	return commands.Outcome{ResultRevision: p.s.Next(), BusinessState: "done", Fields: map[string]string{"value": "x", "note": c.Fields["note"]}}, nil
}

type reader struct{ s *Server }

func (p reader) Resolve(string, string) (commands.Target, commands.Resolution) {
	return commands.Target{Revision: p.s.Revision()}, commands.Found
}

func (p reader) Apply(commands.Context) (commands.Outcome, error) {
	return commands.Outcome{Fields: map[string]string{"state": "read"}}, nil
}

func (f world) register(id string, p commands.Handler, mutating bool) {
	f.t.Helper()
	perm := commands.Permission{Action: gate.ChangeCustomerPolicy, Kind: "installation"}
	if !mutating {
		perm = commands.Permission{Action: gate.View, Kind: "installation", Fields: []string{"state"}}
	}
	d := commands.Definition{ID: id, Permission: perm, Mutating: mutating,
		Fields: []commands.Field{{Name: "note", Type: commands.Text}}, Handler: p}
	if err := f.s.Register(d); err != nil {
		f.t.Fatal(err)
	}
}

func (f world) grant(user string) {
	f.t.Helper()
	for _, action := range []gate.Action{gate.View, gate.ChangeCustomerPolicy} {
		terms := gate.Terms{Action: action, Kind: "installation", Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}, Period: gate.AnyPeriod()}
		if action == gate.View {
			terms.Fields = []string{"state"}
		}
		validity := gate.Validity{NotBefore: start.Add(-time.Hour), NotAfter: start.Add(1000 * time.Hour)}
		for _, g := range []gate.Grant{{ID: "g-" + user + "-" + string(action), SubjectID: user, Terms: terms, Validity: validity},
			{ID: "c-" + user + "-" + string(action), ClientID: ConsoleClient, Terms: terms, Validity: validity}} {
			if err := f.s.Administration().AddGrant(g); err != nil && !errors.Is(err, gate.ErrConflict) {
				f.t.Fatal(err)
			}
		}
	}
}

func (f world) login() (*http.Cookie, string) {
	f.t.Helper()
	cookie, csrf, err := f.s.OpenSession(AdminTriple(), SampleWorkspace, false)
	if err != nil {
		f.t.Fatal(err)
	}
	return cookie, csrf
}

type reply struct {
	status int
	header http.Header
	body   map[string]any
	raw    string
}

func (f world) do(method, path string, body string, set func(*http.Request)) reply {
	f.t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Host = "127.0.0.1:8820"
	if method == http.MethodPost {
		r.Header.Set("Origin", "http://127.0.0.1:5173")
	}
	if set != nil {
		set(r)
	}
	rec := httptest.NewRecorder()
	f.h.ServeHTTP(rec, r)
	raw, _ := io.ReadAll(rec.Body)
	out := reply{status: rec.Code, header: rec.Header(), raw: string(raw)}
	_ = json.Unmarshal(raw, &out.body)
	return out
}

func (f world) call(command, body string, cookie *http.Cookie, csrf string) reply {
	f.t.Helper()
	return f.do(http.MethodPost, "/api/v1/commands/"+command, body, func(r *http.Request) {
		if cookie != nil {
			r.AddCookie(cookie)
		}
		r.Header.Set(csrfHeader, csrf)
	})
}

func mutation(key, expected string) string {
	return `{"request_id":"req-1","target_id":"t1","idempotency_key":"` + key + `","expected_revision":"` + expected + `","body":{"note":"hi"}}`
}

func (f world) code(t testing.TB, name string, got reply, status int, code string) {
	t.Helper()
	if got.status != status || got.body["code"] != code {
		t.Errorf("%s: got %d %v want %d %s", name, got.status, got.raw, status, code)
	}
}

func fastPasswords() passwords.Params {
	p := passwords.DefaultParams()
	p.Iterations = 1000
	return p
}

func newPolicy(f world) (*authpolicy.Registry, error) { return authpolicy.NewRegistry(f.s.clock) }
