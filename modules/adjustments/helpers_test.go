package adjustments

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
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "adjustments-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Workspaces: 3, Drafts: 6, Lines: 4, Entries: 6, Evidence: 2}
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

func (c *testClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

var start = time.Date(2026, 11, 14, 9, 0, 0, 0, time.UTC)

func digest(c byte) string { return strings.Repeat(string(c), 64) }

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

func expect(t testing.TB, want string, err error, what string) {
	t.Helper()
	if kind(err) != want {
		t.Errorf("%s: want %s, got %s", what, want, kind(err))
	}
}

func setup(t testing.TB) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testProfile(t), testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return r, clock
}

func base(issued bool) Base {
	b := Base{Run: "run-1", Digest: digest('a'), Currency: "USD", Scale: 2, Mode: "half_even"}
	if issued {
		b.Issued, b.Document = true, "INV-1"
	}
	return b
}

func sample(id string, issued bool) Draft {
	return Draft{ID: id, Customer: "c1", Contract: "k1", From: "2026-10-01", To: "2026-10-31", Base: base(issued), Lines: []SourceLine{
		{ID: "l1", Charge: "ch-1", Quantity: "10", Price: "2.5", Amount: "25"},
		{ID: "l2", Charge: "ch-2", Quantity: "1", Amount: "75"},
	}}
}

func item(id, k, line, target, amount string) Entry {
	return Entry{ID: id, Kind: k, Line: line, Target: target, Amount: amount, Reason: ReasonContractTerm, Evidence: []string{"ev-" + id}, From: "2026-10-01", To: "2026-10-31"}
}

func opened(t testing.TB, r *Registry, d Draft) View {
	t.Helper()
	return must(r.Open("ws", d, human("writer")))
}

func check(v View, tax, final string) Check {
	return Check{Revision: v.Revision, Digest: v.Digest, Tax: digest('c'), Allocation: digest('d'), TaxAmount: tax, Final: final}
}

func release(request string, v View, approvers ...Actor) Release {
	return Release{Request: request, Revision: v.Revision, Subject: v.Subject, Base: v.Draft.Base, Approval: "ap-" + request, Approvers: approvers}
}

func ready(t testing.TB, r *Registry, id string, entries ...Entry) View {
	t.Helper()
	v := opened(t, r, sample(id, false))
	for _, e := range entries {
		v = must(r.Add("ws", id, v.Revision, e, human("clerk")))
	}
	return must(r.Validate("ws", id, check(v, "0", v.Total), system("tax-engine")))
}
