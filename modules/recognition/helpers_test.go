package recognition

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

func testNumbers(t testing.TB) decimal.Profile {
	t.Helper()
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "recognition-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Workspaces: 3, Profiles: 4, Batches: 40, Lines: 4, Adjustments: 10}
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

func snapshot(c byte) string { return strings.Repeat(string(c), 64) }

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

func testProfile() Profile {
	return Profile{ID: "p1", Entity: "buyer-kr", Ledger: "gl-main", Contract: "k1", Basis: "provider_cost", TaxBasis: "tax_separate", Currency: "KRW", Scale: 0, Provisional: true, Method: "reverse_then_confirm"}
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	clock := &testClock{now: start}
	r, err := NewRegistry(testNumbers(t), testLimits(), clock.read)
	if err != nil {
		t.Fatal(err)
	}
	f := fixture{r: r, clock: clock}
	must(r.Define("w1", testProfile(), human("controller")))
	return f
}

func line(id, net, tax string) Line { return Line{ID: id, Lineage: "src:" + id, Net: net, Tax: tax} }

func initial(id, month, net, tax string) Batch {
	return Batch{ID: id, Profile: "p1", Month: month, Posting: month, Kind: BatchInitial, Snapshot: snapshot('a'), Lines: []Line{line(id+"-1", net, tax)}}
}

func amendment(id, month, posting, net, tax string) Batch {
	return Batch{ID: id, Profile: "p1", Month: month, Posting: posting, Kind: BatchAmendment, Snapshot: snapshot('b'), Lines: []Line{line(id+"-1", net, tax)}}
}

func reversal(id, month, posting, target string) Batch {
	return Batch{ID: id, Profile: "p1", Month: month, Posting: posting, Kind: BatchReversal, Reverses: target, Snapshot: snapshot('c')}
}

func (f fixture) approved(t *testing.T, b Batch) BatchView {
	t.Helper()
	v := must(f.r.Open("w1", b, human("accountant")))
	return must(f.r.Approve("w1", b.ID, v.Digest, human("controller")))
}

func (f fixture) post(t *testing.T, b Batch) BatchView {
	t.Helper()
	f.approved(t, b)
	must(f.r.Submit("w1", b.ID, "key-"+b.ID, human("accountant")))
	must(f.r.Record("w1", "key-"+b.ID, Result{State: ResultPosted, External: "je-" + b.ID}, system("erp")))
	return must(f.r.BatchOf("w1", b.ID))
}

func quarter(id string, billedNet, billedTax string, months ...string) Quarter {
	return Quarter{ID: id, Months: months, Currency: "KRW", Basis: "provider_cost", TaxBasis: "tax_separate", Billed: Amounts{Net: billedNet, Tax: billedTax}, Final: true}
}

func (f fixture) state(t *testing.T, q Quarter) Reconciliation {
	t.Helper()
	return must(f.r.Reconcile("w1", "p1", q))
}
