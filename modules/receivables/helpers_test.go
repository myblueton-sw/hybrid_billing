package receivables

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func testProfile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "receivables-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Workspaces: 3, Terms: 6, Bindings: 4, Receivables: 12, Allocations: 4, Extensions: 3, Holds: 2, NetDays: 120, Freshness: 48 * time.Hour}
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

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var start = time.Date(2026, 10, 1, 3, 0, 0, 0, time.UTC)

func digest(c byte) string { return strings.Repeat(string(c), 64) }

func openWith(t testing.TB, limits Limits) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testProfile(t), limits, clock.read, time.LoadLocation)
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

func setup(t testing.TB, r *Registry, contract, zone string, net int) {
	t.Helper()
	id := "net" + itoa(net)
	must(r.RecordTerm("ws", Term{ID: id, Revision: 1, NetDays: net, Mode: ModeCalendar}, "clerk"))
	if _, err := r.ApproveTerm("ws", id, 1, "lead"); err != nil {
		t.Fatal(err)
	}
	must(r.Bind("ws", Binding{ID: contract + "-b", Contract: contract, Term: id, Revision: 1, Zone: zone, From: "2026-01-01", Reason: "initial"}, "clerk"))
	must(r.ApproveBinding("ws", contract+"-b", "lead"))
}

func invoice(id, contract, amount string, at time.Time) Receivable {
	return Receivable{Document: id, Customer: "cu", Entity: "en", Currency: "USD", Contract: contract, Amount: amount, IssuedAt: at, Obligation: ObligationPayable}
}

func receipt(id, document, amount, effective string) Allocation {
	return Allocation{ID: id, Document: document, Kind: "receipt", Amount: amount, Effective: effective, Source: SourceAuthority, Ledger: "erp", Revision: "r1"}
}
