package marketplace

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
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
	made, err := decimal.NewProfile(decimal.ProfileSpec{ID: "market-test", Revision: "1", InputIntegerDigits: 14, InputScale: 8, ResultIntegerDigits: 16, ResultScale: 8, IntermediateDigits: 80})
	if err != nil {
		t.Fatal(err)
	}
	return made
}

func testLimits() Limits {
	return Limits{Transactions: 6, Revisions: 3, Lines: 8, Recognitions: 3, Settlements: 3, Policies: 6, Digits: 60}
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

func (c *testClock) set(now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = now
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

func open(t testing.TB) *Registry {
	t.Helper()
	r, _ := openWith(t)
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

var (
	evidenceA = strings.Repeat("a", 64)
	evidenceB = strings.Repeat("b", 64)
	evidenceC = strings.Repeat("c", 64)
)

func deal(id, role, channel, offer, collector string) Transaction {
	t := Transaction{ID: id, Provider: "aws", Agreement: "agr-1", Role: role, Channel: channel, Type: offer, Collector: collector,
		Participants: Participants{SoftwareProvider: "isv-1", Seller: "isv-1", Issuer: "aws"},
		References:   References{Product: "prod-1", Offer: "offer-1"},
		Term:         Term{Accepted: "2026-01", Start: "2026-01", End: "2026-12", Currency: "USD"}}
	if role == RoleReseller {
		t.Participants.Reseller = "msp-1"
	}
	if role != RoleBuyer {
		t.Participants.Payee = "msp-1"
	}
	return t
}

func buyerPrivate(id string) Transaction {
	return deal(id, RoleBuyer, ChannelMarketplace, TypePrivate, CollectorMSP)
}

func line(source, lineKind, charged string, discount Benefit) Line {
	return Line{Source: source, Kind: lineKind, Account: "acct-1", Charged: charged, Discount: discount}
}

func applied(amount string) Benefit {
	return Benefit{State: Applied, Amount: amount, Evidence: evidenceA}
}
func notApplied() Benefit { return Benefit{State: NotApplied, Evidence: evidenceA} }
func unknown() Benefit    { return Benefit{State: Unknown} }
func recognized(amount string) Recognition {
	return Recognition{Program: "macc-1", State: Applied, Amount: amount, Evidence: evidenceB}
}

func record(t testing.TB, r *Registry, tx Transaction) {
	t.Helper()
	if _, err := r.RecordTransaction("ws", tx); err != nil {
		t.Fatalf("transaction: %v", err)
	}
}

func write(t testing.TB, r *Registry, id, month string, number int, facts Facts) {
	t.Helper()
	if _, err := r.RecordStatement("ws", id, month, number, facts); err != nil {
		t.Fatalf("statement %s %s %d: %v", id, month, number, err)
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

func sealed(t testing.TB, r *Registry, id, month string, number int) Statement {
	t.Helper()
	if state := seal(t, r, id, month, number); state.State != StateSealed {
		t.Fatalf("held: %v", state.Reasons)
	}
	s, err := r.Statement("ws", id, month, number)
	if err != nil {
		t.Fatalf("statement: %v", err)
	}
	return s
}

func terms(id, version, transaction, rate string, scope []string, source string) PolicyTerms {
	p := PolicyTerms{ID: id, Version: version, Customer: "cust-1", Transaction: transaction, First: "2026-01", Last: "2026-12", Rate: rate, Scope: scope,
		SourceDiscounted: source, Quantum: "0.01"}
	if rate != "0" {
		p.Evidence = evidenceC
	}
	if source == Included {
		p.SourceBasis = evidenceB
	}
	return p
}

func approve(t testing.TB, r *Registry, p PolicyTerms) {
	t.Helper()
	if _, err := r.RecordPolicy("ws", p, "author-1"); err != nil {
		t.Fatalf("policy: %v", err)
	}
	if _, err := r.ApprovePolicy("ws", p.ID, p.Version, "approver-1"); err != nil {
		t.Fatalf("approve: %v", err)
	}
}

func policyFor(t testing.TB, r *Registry, transaction, month string) Policy {
	t.Helper()
	p, err := r.PolicyFor("ws", transaction, month)
	if err != nil {
		t.Fatalf("policy for: %v", err)
	}
	return p
}

func compute(t testing.TB, s Statement, p Policy) ChargeView {
	t.Helper()
	c, err := ComputeCharge(s, p)
	if err != nil {
		t.Fatalf("charge: %v", err)
	}
	return c.View()
}

func amounts(v ChargeView) string {
	if v.Hold != "" {
		return "hold " + v.Hold
	}
	parts := []string{}
	for _, l := range v.Lines {
		parts = append(parts, l.Source+"="+l.Amount+"/-"+l.Discount)
	}
	return strings.Join(parts, " ") + " | total " + v.Total + " charged " + v.ChargedTotal + " margin " + v.Margin
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

func exampleFacts() Facts {
	sw := line("sw-1", KindSoftware, "800", applied("200"))
	sw.List = "1000"
	return Facts{Lines: []Line{sw, line("infra-1", KindInfrastructure, "200", notApplied())}, Recognitions: []Recognition{recognized("800")}}
}

func example(t testing.TB) (*Registry, Statement) {
	t.Helper()
	r := open(t)
	record(t, r, buyerPrivate("tx-1"))
	write(t, r, "tx-1", "2026-03", 1, exampleFacts())
	return r, sealed(t, r, "tx-1", "2026-03", 1)
}
