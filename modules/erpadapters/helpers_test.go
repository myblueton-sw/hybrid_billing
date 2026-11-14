package erpadapters

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) read() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var start = time.Date(2026, 11, 15, 9, 0, 0, 0, time.UTC)

func human(id string) Actor { return Actor{ID: id, Kind: KindHuman} }

func assistant(id string) Actor { return Actor{ID: id, Kind: KindAssistant} }

func system(id string) Actor { return Actor{ID: id, Kind: KindSystem} }

func kind(err error) string {
	var refusal Refusal
	switch {
	case err == nil:
		return "ok"
	case errors.As(err, &refusal):
		return refusal.Reason
	case errors.Is(err, ErrInvalid):
		return "invalid"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrConflict):
		return "conflict"
	case errors.Is(err, ErrRejected):
		return "rejected"
	case errors.Is(err, ErrLimit):
		return "limit"
	}
	return "other: " + err.Error()
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func second[T any](_ T, err error) error { return err }

func expect(t *testing.T, label string, err error, want string) {
	t.Helper()
	if got := kind(err); got != want {
		t.Fatalf("%s: got %s, want %s", label, got, want)
	}
}

type fixture struct {
	r     *Registry
	clock *testClock
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(Limits{Workspaces: 3, Profiles: 5, Adapters: 5, Diagnoses: 8}, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{r: r, clock: clock}
	must(r.Profile("w1", Profile{ID: "odoo17", Vendor: "odoo", Product: "odoo", Version: "17.0", Deployment: "cloud", Modules: []string{"account", "sale"}}, human("arch")))
	must(r.Adapter("w1", odooAdapter("a1"), human("arch")))
	return f
}

func allMethods() []Method {
	var out []Method
	for _, c := range commands {
		out = append(out, Method{Command: c, Name: "odoo." + c, Transport: "api"})
	}
	return out
}

func odooAdapter(id string) Adapter {
	return Adapter{ID: id, Profile: "odoo17", Methods: allMethods(), MaxRows: 500, CallsPerMinute: 60}
}

func fixtureEvidence(level string) Evidence {
	return Evidence{Level: level, Reference: "test-fixture-" + level}
}

func (f fixture) raiseAll(t *testing.T, adapter, level string) {
	t.Helper()
	for _, c := range lifecycle {
		must(f.r.Evidence("w1", adapter, c, fixtureEvidence(level), human("qa")))
	}
}
