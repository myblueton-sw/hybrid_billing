package incidents

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

func (c *testClock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

var start = time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)

func human(id string) Actor { return Actor{ID: id, Kind: KindHuman} }

func assistant(id string) Actor { return Actor{ID: id, Kind: KindAssistant} }

func system(id string) Actor { return Actor{ID: id, Kind: KindSystem} }

func reason(err error) string {
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
	if got := reason(err); got != want {
		t.Fatalf("%s: got %s, want %s", label, got, want)
	}
}

type permissions struct {
	mu      sync.Mutex
	denied  map[[2]string]bool
	failing map[string]bool
}

func (p *permissions) Allowed(workspace, person, subject string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failing[person] {
		return false, errors.New("down")
	}
	return !p.denied[[2]string{person, subject}], nil
}

func (p *permissions) deny(person, subject string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.denied[[2]string{person, subject}] = true
}

func (p *permissions) allow(person, subject string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.denied, [2]string{person, subject})
}

type fixture struct {
	r     *Registry
	clock *testClock
	perms *permissions
}

func testLimits() Limits {
	return Limits{Workspaces: 2, Incidents: 50, Signals: 200, Recipients: 6, Subscriptions: 20, Notifications: 400}
}

func open(t testing.TB, limits Limits) fixture {
	t.Helper()
	clock := &testClock{now: start}
	perms := &permissions{denied: map[[2]string]bool{}, failing: map[string]bool{}}
	r, err := NewRegistry(Config{Permissions: perms, Zones: time.LoadLocation}, limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{r: r, clock: clock, perms: perms}
}

func signal(id, condition, subject, severity string, blocking bool, at time.Time) Signal {
	return Signal{ID: id, Condition: condition, Subject: subject, Severity: severity, Blocking: blocking, Occurred: at}
}

func (f fixture) raise(t testing.TB, s Signal) Incident {
	t.Helper()
	return must(f.r.Raise("ws", s, system("monitor")))
}

func (f fixture) subscribe(t testing.TB, who, condition, minimum string) {
	t.Helper()
	must(f.r.Subscribe("ws", Subscription{Recipient: who, Condition: condition, Minimum: minimum}, human(who)))
}

func (f fixture) cycle(t testing.TB) []Notification {
	t.Helper()
	var out []Notification
	for _, n := range must(f.r.Plan("ws", system("planner"))) {
		out = append(out, must(f.r.Dispatch("ws", n.ID, system("sender"))))
	}
	return out
}
