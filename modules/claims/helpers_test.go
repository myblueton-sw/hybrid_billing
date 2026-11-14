package claims

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func testProfile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "claims-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Contracts: 4, Revisions: 3, Components: 4, Applications: 12, Sources: 3, Reversals: 3, Batch: 4}
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

var start = time.Date(2026, 11, 14, 3, 0, 0, 0, time.UTC)

func openWith(t testing.TB) (*Registry, *testClock) {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testProfile(t), testLimits(), clock.read)
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
	return fmt.Sprintf("other:%v", err)
}

func comp(id, k string, billable bool, route string, instances ...string) Component {
	c := Component{ID: id, Kind: k, Billable: billable, Route: route, Instances: instances}
	if route == RouteExternal {
		c.Reference = "erp-order-1"
	}
	return c
}

func resale() Revision {
	return Revision{From: "2026-01-01", To: "2027-01-01", Path: PathResale, Components: []Component{
		comp("usage", KindCSPUsage, true, RoutePlatform, "main"),
		comp("fee", KindMSPService, true, RoutePlatform, "fee-1", "fee-2"),
		comp("support", KindOther, false, RoutePlatform, "s"),
	}}
}

func direct() Revision {
	return Revision{From: "2026-01-01", To: "2027-01-01", Path: PathDirect, Components: []Component{
		comp("usage", KindCSPUsage, false, RoutePlatform, "main"),
		comp("fee", KindMSPService, true, RoutePlatform, "fee-1"),
	}}
}

func eligible(t testing.TB, r *Registry, contract string, number int, rev Revision) {
	t.Helper()
	if _, err := r.RecordEligibility("ws", contract, number, rev, "author-1"); err != nil {
		t.Fatalf("record: %v", err)
	}
	if _, err := r.ApproveEligibility("ws", contract, number, "approver-1"); err != nil {
		t.Fatalf("approve: %v", err)
	}
}

func monthly(id, contract, instance, startDate, end, amount string) Application {
	return Application{ID: id, Scope: Scope{Contract: contract, Component: "fee", Instance: instance}, Start: startDate, End: end, Mode: ModeInterval,
		Quantity: "1", Amount: amount, Currency: "USD", Calculation: "calc-1", Lifetime: 3600}
}

func usage(id, contract, src, quantity, amount string) Application {
	return Application{ID: id, Scope: Scope{Contract: contract, Component: "usage", Instance: "main"}, Start: "2026-03-01", End: "2026-04-01", Mode: ModeSource,
		Source: src, CapQuantity: "100", CapAmount: "50", Quantity: quantity, Amount: amount, Currency: "USD", Calculation: "calc-1", Lifetime: 3600}
}

func reserve(t testing.TB, r *Registry, a Application) Decision {
	t.Helper()
	d, err := r.Reserve("ws", a, "runner-1")
	if err != nil {
		t.Fatalf("reserve %s: %v", a.ID, err)
	}
	return d
}

func reserved(t testing.TB, r *Registry, a Application) {
	t.Helper()
	if d := reserve(t, r, a); d.Reason != "" || d.Application.State != StateReserved {
		t.Fatalf("reserve %s: %+v", a.ID, d)
	}
}

func confirm(t testing.TB, r *Registry, ids ...string) {
	t.Helper()
	if _, refusal, err := r.Confirm("ws", ids, "runner-1"); err != nil {
		t.Fatalf("confirm %v: %v %+v", ids, err, refusal)
	}
}

func billed(t testing.TB, r *Registry, a Application) {
	t.Helper()
	reserved(t, r, a)
	confirm(t, r, a.ID)
}

func reverse(t testing.TB, r *Registry, id, app, quantity, amount string) {
	t.Helper()
	if _, err := r.Reverse("ws", Reversal{ID: id, Application: app, Quantity: quantity, Amount: amount}, "clerk-1"); err != nil {
		t.Fatalf("reverse: %v", err)
	}
	if _, err := r.ApproveReversal("ws", id, "lead-1"); err != nil {
		t.Fatalf("approve reversal: %v", err)
	}
}

func refused(t testing.TB, r *Registry, a Application) string {
	t.Helper()
	d, err := r.Reserve("ws", a, "runner-2")
	if err != nil {
		t.Fatalf("reserve %s: %v", a.ID, err)
	}
	return d.Reason
}

func printed(fields ...string) string {
	sum := sha256.New()
	for _, field := range fields {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(field)))
		sum.Write(size[:])
		sum.Write([]byte(field))
	}
	return hex.EncodeToString(sum.Sum(nil))
}

func readSource(file string) string {
	body, _ := os.ReadFile(file)
	return string(body)
}
