package commitment

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func testProfile(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "commit-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Commitments: 6, Revisions: 3, Lines: 8, Customers: 4, Digits: 60}
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

var start = time.Date(2026, 11, 13, 3, 0, 0, 0, time.UTC)

func open(t testing.TB) *Registry {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testProfile(t), testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	return r
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
	case errors.Is(err, ErrSelfCheck):
		return "self_check"
	}
	return fmt.Sprintf("other:%v", err)
}

func commitmentOf(id, payment, prepaid, recovery string) Commitment {
	return Commitment{ID: id, Provider: "aws", PayerAccount: "payer-1", Kind: KindSavings, Currency: "USD", First: "2026-01", Last: "2026-12",
		Payment: payment, Prepaid: prepaid, Quantum: "0.01", Recovery: recovery}
}

func used(source, account, recurring, amortized, list string) Line {
	return Line{Source: source, Kind: LineUsed, Commitment: "c1", Account: account, Recurring: recurring, Amortized: amortized, List: list}
}

func unused(source, recurring, amortized string) Line {
	return Line{Source: source, Kind: LineUnused, Commitment: "c1", Recurring: recurring, Amortized: amortized}
}

func record(t testing.TB, r *Registry, c Commitment) {
	t.Helper()
	if _, err := r.RecordCommitment("ws", c); err != nil {
		t.Fatalf("commitment: %v", err)
	}
}

func write(t testing.TB, r *Registry, id, month string, number int, recurring, amortized string, lines ...Line) {
	t.Helper()
	if _, err := r.RecordPeriod("ws", id, month, number, Facts{Recurring: recurring, Amortized: amortized, Lines: lines}); err != nil {
		t.Fatalf("period %s %s %d: %v", id, month, number, err)
	}
}

func seal(t testing.TB, r *Registry, id, month string, number int) RevisionState {
	t.Helper()
	state, err := r.Seal("ws", id, month, number)
	if err != nil {
		t.Fatalf("seal %s %s %d: %v", id, month, number, err)
	}
	return state
}

func sealed(t testing.TB, r *Registry, id, month string, number int) Period {
	t.Helper()
	if state := seal(t, r, id, month, number); state.State != StateSealed {
		t.Fatalf("held: %v", state.Reasons)
	}
	p, err := r.Revision("ws", id, month, number)
	if err != nil {
		t.Fatalf("revision: %v", err)
	}
	return p
}

func accounts() Attribution {
	return Attribution{ID: "attr", Version: "1", Mappings: []Mapping{{Account: "acct-a", Customer: "A"}, {Account: "acct-b", Customer: "B"}}}
}

func bear(bearing, customer string) AllocationPolicy {
	return AllocationPolicy{ID: "alloc", Version: "1", Bearing: bearing, Customer: customer}
}

func costPlus(customer, fee string, onUnused bool) SalePolicy {
	return SalePolicy{Customer: customer, ID: "sale-" + customer, Version: "1", Method: MethodCostPlus, Fee: fee, FeeOnUnused: onUnused, Mode: "half_even"}
}

func share(customer, rate string) SalePolicy {
	return SalePolicy{Customer: customer, ID: "share-" + customer, Version: "1", Method: MethodSavingsShare, Share: rate, Mode: "half_even"}
}

func charges(s Sale) string {
	parts := []string{}
	for _, c := range s.Charges {
		if c.Hold != "" {
			parts = append(parts, c.Customer+":"+c.Hold)
		} else {
			parts = append(parts, c.Customer+"="+c.Amount)
		}
	}
	return strings.Join(parts, " ") + fmt.Sprintf(" | total %s cost %s recovered %s result %s below %v priced %v", s.Total, s.CostCompared, s.Recovered, s.MSPResult, s.BelowCost, s.AllCustomersPriced)
}

func example(t testing.TB) (*Registry, Period) {
	t.Helper()
	r := open(t)
	record(t, r, commitmentOf("c1", PaymentNone, "0", RecoveryCharged))
	write(t, r, "c1", "2026-03", 1, "100", "0", used("l1", "acct-a", "60", "0", "100"), used("l2", "acct-b", "25", "0", "50"), unused("l3", "15", "0"))
	return r, sealed(t, r, "c1", "2026-03", 1)
}

func readSource(file string) string {
	body, _ := os.ReadFile(file)
	return string(body)
}
