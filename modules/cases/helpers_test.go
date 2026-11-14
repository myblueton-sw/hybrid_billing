package cases

import (
	"errors"
	"strings"
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

func (c *testClock) set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = at
}

var start = time.Date(2026, 11, 14, 3, 0, 0, 0, time.UTC)

func digest(c byte) string { return strings.Repeat(string(c), 64) }

func testLimits() Limits {
	return Limits{Workspaces: 3, Findings: 6, Cases: 8, CustomerCases: 5, Steps: 4, Messages: 3, Entries: 4, Subjects: 3, Page: 50}
}

func openWith(t testing.TB, limits Limits) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return r, clock
}

func kind(err error) string {
	switch {
	case err == nil:
		return "ok"
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

func op(id string) Actor { return Actor{ID: id, Side: SideOperator} }

func cust(id, customer string) Actor { return Actor{ID: id, Side: SideCustomer, Customer: customer} }

func spike(id, customer string) Finding {
	return Finding{ID: id, Kind: "cost_spike", Target: Target{Customer: customer}, PeriodStart: "2026-10-01", PeriodEnd: "2026-10-31", Meaning: MeaningCost,
		Amount: "120.5", Currency: "USD", Evidence: digest('a'), Severity: "high", Owner: "owner-1", Source: SourceManual}
}

func because(step, reason string) Transition {
	return Transition{Step: step, Reason: reason, Evidence: digest('b')}
}

func inquiry(id, customer string) Case {
	return Case{ID: id, Kind: KindInquiry, Customer: customer, Subjects: []Subject{{Document: "inv-1"}}, Topic: "amount", Body: digest('c')}
}

func dispute(id, customer, document string) Case {
	return Case{ID: id, Kind: KindDispute, Customer: customer, Subjects: []Subject{{Document: document, Line: "l1"}}, Topic: "rate", Amount: "40", Currency: "USD", Body: digest('d')}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func expect(t testing.TB, want string, err error, what string) {
	t.Helper()
	if kind(err) != want {
		t.Errorf("%s: want %s, got %s", what, want, kind(err))
	}
}
