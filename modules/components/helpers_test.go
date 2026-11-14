package components

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func testProfile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "components-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Workspaces: 3, Components: 8, Inclusions: 3, Readings: 6, Incidents: 6, IncidentsPerCredit: 2, Clauses: 3, Credits: 3}
}

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) read() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testClock) set(at time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = at
}

var start = time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)

func digest(c byte) string { return strings.Repeat(string(c), 64) }

func openWith(t testing.TB, limits Limits) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testProfile(t), limits, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return r, clock
}

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

func expect(t testing.TB, want string, err error, what string) {
	t.Helper()
	if kind(err) != want {
		t.Errorf("%s: want %s, got %s", what, want, kind(err))
	}
}

func rack(id, fee string, includes ...Inclusion) Component {
	return Component{ID: id, Contract: "k", Currency: "USD", Kind: KindRack, From: "2026-01-01", Fee: fee, Includes: includes}
}

func usage(id, meter, rate string) Component {
	return Component{ID: id, Contract: "k", Currency: "USD", Kind: KindUsage, From: "2026-01-01", Meter: meter, Rate: rate}
}

func priced(t testing.TB, r *Registry) {
	t.Helper()
	must(r.SetPricing("ws", Pricing{Contract: "k", Scale: 2, Mode: "half_even"}, "admin"))
}

func approved(t testing.TB, r *Registry, c Component) {
	t.Helper()
	must(r.Define("ws", c, "clerk"))
	must(r.ApproveComponent("ws", c.ID, "lead"))
}

func november(usage ...Reading) Request {
	return Request{Contract: "k", PeriodStart: "2026-11-01", PeriodEnd: "2026-11-30", Usage: usage}
}
