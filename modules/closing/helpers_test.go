package closing

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

func (c *testClock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var start = time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)

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

type fixture struct {
	r     *Registry
	clock *testClock
}

func testLimits() Limits {
	return Limits{Workspaces: 2, Calendars: 6, Cycles: 6, Targets: 200, History: 2000, Claims: 50, Lead: 24 * time.Hour}
}

func open(t testing.TB, limits Limits) fixture {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(Config{Zones: time.LoadLocation}, limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{r: r, clock: clock}
}

func seoul() Calendar {
	return Calendar{ID: "kr", Zone: "Asia/Seoul", Weekend: []time.Weekday{time.Saturday, time.Sunday}, Holidays: []string{"2026-11-03"}, Close: 18 * 60, Covers: Range{From: "2026-01-01", To: "2027-12-31"}}
}

func (f fixture) calendar(t testing.TB, workspace string, c Calendar) CalendarView {
	t.Helper()
	v := must(f.r.ProposeCalendar(workspace, c, human("planner")))
	return must(f.r.ApproveCalendar(workspace, c.ID, v.Revision, human("controller")))
}

func stages() []StageSpec {
	return []StageSpec{
		{Stage: StageCollect, Rule: Rule{Days: 1, Business: true}, Basis: BasisFile},
		{Stage: StageValidate, Rule: Rule{Days: 2, Business: true}, Basis: BasisInternal},
		{Stage: StageIssue, Rule: Rule{Days: 3, Business: true}, Basis: BasisERP},
		{Stage: StageSettle, Rule: Rule{Days: 10, Business: false}, Basis: BasisInternal},
	}
}

func (f fixture) cycle(t testing.TB, id string, targets ...string) CycleView {
	t.Helper()
	f.calendar(t, "ws", seoul())
	v := must(f.r.OpenCycle("ws", Cycle{ID: id, Period: "2026-10", Calendar: "kr", Revision: 1, Stages: stages()}, human("planner")))
	if len(targets) > 0 {
		v = must(f.r.AddTargets("ws", id, Addition{Targets: targets, Reason: "initial"}, human("planner")))
	}
	return v
}

func evidence(kind string) Evidence { return Evidence{Kind: kind, Reference: "ref-" + kind} }
