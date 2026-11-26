package pricingpolicy

import (
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/rating"
	"github.com/myblueton-sw/hybrid_billing/packages/primitives/decimal"
)

type testAuthority struct{ denied map[string]bool }

func (a *testAuthority) Check(scope Scope, actor, action string) error {
	if a.denied[actor] || scope.Workspace != "ws" {
		return ErrRejected
	}
	if action == ProposeAction && actor == "maker" || action == ApproveAction && (actor == "checker" || actor == "checker2" || actor == "maker") || action == RevokeAction && actor == "revoker" || action == UseAction && actor == "consumer" {
		return nil
	}
	return ErrRejected
}

type testTargets struct {
	value Target
	err   error
}

func (t *testTargets) Current(Scope, time.Time, time.Time) (Target, error) { return t.value, t.err }

type fixture struct {
	r         *Registry
	authority *testAuthority
	targets   *testTargets
	now       time.Time
	spec      Spec
}

func setup(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{authority: &testAuthority{map[string]bool{}}, targets: &testTargets{value: Target{ContractRevision: 1, BillingRevision: 1, Currency: "USD", Path: rating.Resale, Fingerprint: strings.Repeat("a", 64)}}, now: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)}
	f.spec = Spec{Mode: "synthetic", Scope: Scope{"ws", "customer", "entity", "contract", "billing"}, From: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		Profile: decimal.ProfileSpec{ID: "numeric", Revision: "1", InputIntegerDigits: 12, InputScale: 4, ResultIntegerDigits: 14, ResultScale: 4, IntermediateDigits: 60}, Limits: rating.Limits{Rules: 10, Tiers: 10, Discounts: 10, Lines: 100, Intermediate: 60},
		Policy: rating.Policy{ID: "price", Version: "1", Currency: "USD", CostBasis: rating.Billed, Quantum: "0.01", Mode: "half_even", Path: rating.Resale, Eligible: []rating.Class{rating.CSPUsage}, Rules: []rating.Rule{{ID: "base", Level: rating.Default, Type: rating.PassThrough}}}}
	var err error
	f.r, err = New(20, f.authority, f.targets, func() time.Time { return f.now })
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *fixture) propose(t *testing.T, id string) Record {
	t.Helper()
	r, err := f.r.Propose(id, "maker", "Synthetic policy", f.spec)
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func (f *fixture) approve(t *testing.T, id string) Record {
	t.Helper()
	p := f.propose(t, id)
	r, err := f.r.Approve("ws", id, p.Digest, "checker", "Independent review")
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func TestGovernedSelectionRetainsImmutableHistory(t *testing.T) {
	f := setup(t)
	p := f.propose(t, "p")
	if _, err := f.r.Resolve(f.spec.Scope, "p", p.Digest, "consumer", f.spec.From, f.spec.To); !errors.Is(err, ErrRejected) {
		t.Fatal("pending selected", err)
	}
	a := f.approve(t, "p")
	got, err := f.r.Resolve(f.spec.Scope, "p", a.Digest, "consumer", f.spec.From, f.spec.To)
	if err != nil || !reflect.DeepEqual(got, a) {
		t.Fatal(got, err)
	}
	replay, err := f.r.Approve("ws", "p", a.Digest, "checker", "Independent review")
	if err != nil || !reflect.DeepEqual(replay, a) {
		t.Fatal("changed replay")
	}
	f.authority.denied["checker"] = true
	if _, err = f.r.Resolve(f.spec.Scope, "p", a.Digest, "consumer", f.spec.From, f.spec.To); err != nil {
		t.Fatal("historic approver role loss became automatic policy revocation", err)
	}
	revoked, err := f.r.Revoke("ws", "p", a.Digest, "revoker", "Stop new use")
	if err != nil || revoked.State != Revoked || !reflect.DeepEqual(revoked.Approval, a.Approval) {
		t.Fatal(revoked, err)
	}
	if _, err = f.r.Resolve(f.spec.Scope, "p", a.Digest, "consumer", f.spec.From, f.spec.To); !errors.Is(err, ErrRejected) {
		t.Fatal("revoked selected", err)
	}
	if !reflect.DeepEqual(a.Proposal, revoked.Proposal) || a.Revocation != nil {
		t.Fatal("history changed")
	}
}
func TestApprovalOverlapAdjacentIntervalsAndConcurrentWinners(t *testing.T) {
	f := setup(t)
	a := f.propose(t, "a")
	b := f.propose(t, "b")
	var wg sync.WaitGroup
	results := make([]error, 2)
	for i, p := range []Record{a, b} {
		wg.Add(1)
		go func(i int, p Record) {
			defer wg.Done()
			_, results[i] = f.r.Approve("ws", p.Proposal.ID, p.Digest, "checker", "Independent review")
		}(i, p)
	}
	wg.Wait()
	wins := 0
	for _, err := range results {
		if err == nil {
			wins++
		} else if !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
	}
	if wins != 1 {
		t.Fatal("overlapping winners", wins)
	}
	f.spec.From = f.spec.To
	f.spec.To = f.spec.To.AddDate(0, 1, 0)
	next := f.approve(t, "next")
	if _, err := f.r.Resolve(f.spec.Scope, "next", next.Digest, "consumer", f.spec.From.Add(-time.Nanosecond), f.spec.To); !errors.Is(err, ErrRejected) {
		t.Fatal("stitched across boundary", err)
	}
	if _, err := f.r.Resolve(f.spec.Scope, "next", next.Digest, "consumer", f.spec.From, f.spec.To); err != nil {
		t.Fatal("adjacent refused", err)
	}
}
func TestCurrentAuthorityTargetScopeAndExactDecision(t *testing.T) {
	for _, which := range []string{"self", "bot", "proposer-lost", "target", "currency", "digest", "actor-retry", "reason-retry"} {
		t.Run(which, func(t *testing.T) {
			f := setup(t)
			p := f.propose(t, "p")
			actor, reason, want := "checker", "Independent review", p.Digest
			switch which {
			case "self":
				actor = "maker"
			case "bot":
				actor = "assistant"
			case "proposer-lost":
				f.authority.denied["maker"] = true
			case "target":
				f.targets.value.Fingerprint = strings.Repeat("b", 64)
			case "currency":
				f.targets.value.Currency = "EUR"
			case "digest":
				want = strings.Repeat("c", 64)
			case "actor-retry":
				f.approve(t, "p")
				actor = "checker2"
			case "reason-retry":
				f.approve(t, "p")
				reason = "Changed intent"
			}
			if _, err := f.r.Approve("ws", "p", want, actor, reason); err == nil {
				t.Fatal("unexpected approval")
			}
		})
	}
	f := setup(t)
	a := f.approve(t, "p")
	for _, change := range []func(*Scope){func(s *Scope) { s.Workspace = "other" }, func(s *Scope) { s.Customer = "other" }, func(s *Scope) { s.Entity = "other" }, func(s *Scope) { s.Contract = "other" }, func(s *Scope) { s.BillingProfile = "other" }} {
		scope := f.spec.Scope
		change(&scope)
		if _, err := f.r.Resolve(scope, "p", a.Digest, "consumer", f.spec.From, f.spec.To); err == nil {
			t.Fatal("wrong scope selected")
		}
	}
	f.authority.denied["consumer"] = true
	if _, err := f.r.Resolve(f.spec.Scope, "p", a.Digest, "consumer", f.spec.From, f.spec.To); err == nil {
		t.Fatal("revoked use authority")
	}
}
func TestNestedCopiesAndPolicyVersionIdentity(t *testing.T) {
	f := setup(t)
	f.spec.Policy.Rules[0].Discounts = []rating.Discount{{Percent: "0.1"}}
	p := f.propose(t, "p")
	original := clone(p)
	f.spec.Policy.Rules[0].Discounts[0].Percent = "0.2"
	p.Proposal.Spec.Policy.Rules[0].Discounts[0].Percent = "0.3"
	p.Proposal.Spec.Policy.Eligible[0] = rating.Other
	history, err := f.r.Record("ws", "p")
	if err != nil || !reflect.DeepEqual(history, original) {
		t.Fatal("alias changed retained policy")
	}
	if _, err = f.r.Propose("p", "maker", "Synthetic policy", f.spec); !errors.Is(err, ErrConflict) {
		t.Fatal("same id changed", err)
	}
	f.spec.From = f.spec.To
	f.spec.To = f.spec.To.AddDate(0, 1, 0)
	if _, err = f.r.Propose("other", "maker", "Synthetic policy", f.spec); !errors.Is(err, ErrConflict) {
		t.Fatal("same policy version changed terms", err)
	}
	f.spec.Policy.Version = "2"
	f.propose(t, "other")
}
func TestBoundsClockFailureAndTargetUnavailability(t *testing.T) {
	for _, alter := range []func(*Spec){func(s *Spec) { s.Profile.IntermediateDigits = 1000000 }, func(s *Spec) { s.Limits.Lines = 1001 }, func(s *Spec) { s.Policy.Rules[0].ID = strings.Repeat("x", MaxBytes) }, func(s *Spec) { s.Policy.Rules[0].ID = "bad\xff" }, func(s *Spec) { s.To = s.From }, func(s *Spec) { s.Mode = "production" }, func(s *Spec) { s.Policy.Rules[0].Rate = "not-number" }} {
		f := setup(t)
		alter(&f.spec)
		if _, err := f.r.Propose("p", "maker", "Synthetic policy", f.spec); err == nil {
			t.Fatal("invalid proposal accepted")
		}
	}
	f := setup(t)
	a := f.approve(t, "p")
	f.now = f.now.Add(-time.Second)
	if _, err := f.r.Resolve(f.spec.Scope, "p", a.Digest, "consumer", f.spec.From, f.spec.To); !errors.Is(err, ErrRejected) {
		t.Fatal("clock rollback", err)
	}
	f = setup(t)
	p := f.propose(t, "p")
	f.targets.err = errors.New("unavailable")
	if _, err := f.r.Approve("ws", "p", p.Digest, "checker", "Independent review"); err != f.targets.err {
		t.Fatal("unavailable swallowed", err)
	}
	f = setup(t)
	f.r.limit = 1
	f.propose(t, "p")
	if _, err := f.r.Propose("q", "maker", "Synthetic policy", f.spec); !errors.Is(err, ErrLimit) {
		t.Fatal("capacity", err)
	}
}

func TestMandatoryPortsRejectTypedNil(t *testing.T) {
	f := setup(t)
	var authority *testAuthority
	var targets *testTargets
	if _, err := New(1, authority, f.targets, func() time.Time { return f.now }); !errors.Is(err, ErrInvalid) {
		t.Fatal("nil authority accepted", err)
	}
	if _, err := New(1, f.authority, targets, func() time.Time { return f.now }); !errors.Is(err, ErrInvalid) {
		t.Fatal("nil targets accepted", err)
	}
}

func TestSeparateContractsAndFullIntervalCoverage(t *testing.T) {
	f := setup(t)
	a := f.approve(t, "first")
	f.spec.Scope.Contract = "other-contract"
	f.approve(t, "second")
	for _, period := range [][2]time.Time{{a.Proposal.Spec.From.Add(-time.Nanosecond), a.Proposal.Spec.To}, {a.Proposal.Spec.From, a.Proposal.Spec.To.Add(time.Nanosecond)}} {
		if _, err := f.r.Resolve(a.Proposal.Spec.Scope, "first", a.Digest, "consumer", period[0], period[1]); !errors.Is(err, ErrRejected) {
			t.Fatal("partial approved coverage used", err)
		}
	}
	if _, err := f.r.Resolve(a.Proposal.Spec.Scope, "first", a.Digest, "consumer", a.Proposal.Spec.From.Add(time.Hour), a.Proposal.Spec.To.Add(-time.Hour)); err != nil {
		t.Fatal("contained interval refused", err)
	}
}

func TestEncodedBudgetBeforeCompileAndTierAliases(t *testing.T) {
	f := setup(t)
	oversized := f.spec
	oversized.Policy.Rules = append([]rating.Rule(nil), oversized.Policy.Rules...)
	oversized.Policy.Rules[0].Key = strings.Repeat("\"", MaxBytes/2)
	if !form(oversized) {
		t.Fatal("fixture must fit raw text budget")
	}
	if _, err := digest(Proposal{Schema: Schema, Spec: oversized}); !errors.Is(err, ErrInvalid) {
		t.Fatal("escaped payload exceeds encoded budget", err)
	}
	f.spec.Policy.Rules[0] = rating.Rule{ID: "tiered", Level: rating.Default, Type: rating.Graduated, Tiers: []rating.Tier{{UpTo: "10", Price: rating.Price{Amount: "2", Per: 1}}, {Price: rating.Price{Amount: "1", Per: 1}}}}
	p := f.propose(t, "tiered")
	f.spec.Policy.Rules[0].Tiers[0].Price.Amount = "999"
	p.Proposal.Spec.Policy.Rules[0].Tiers[1].Price.Amount = "999"
	stored, err := f.r.Record("ws", "tiered")
	if err != nil || stored.Proposal.Spec.Policy.Rules[0].Tiers[0].Price.Amount != "2" || stored.Proposal.Spec.Policy.Rules[0].Tiers[1].Price.Amount != "1" {
		t.Fatal("tier alias changed retained policy", err)
	}
}

func TestResolutionRevalidatesTargetsAndRevocationCannotReopen(t *testing.T) {
	for _, change := range []func(*testTargets){func(t *testTargets) { t.value.Fingerprint = strings.Repeat("b", 64) }, func(t *testTargets) { t.value.Path = rating.Direct }, func(t *testTargets) { t.err = errors.New("target unavailable") }} {
		f := setup(t)
		a := f.approve(t, "p")
		change(f.targets)
		if _, err := f.r.Resolve(f.spec.Scope, "p", a.Digest, "consumer", f.spec.From, f.spec.To); err == nil {
			t.Fatal("stale current target used")
		}
	}
	f := setup(t)
	a := f.approve(t, "p")
	first, err := f.r.Revoke("ws", "p", a.Digest, "revoker", "Stop new use")
	if err != nil {
		t.Fatal(err)
	}
	f.now = f.now.Add(time.Hour)
	again, err := f.r.Revoke("ws", "p", a.Digest, "revoker", "Stop new use")
	if err != nil || !reflect.DeepEqual(first, again) {
		t.Fatal("revocation retry changed history")
	}
	if _, err = f.r.Approve("ws", "p", a.Digest, "checker", "Independent review"); !errors.Is(err, ErrRejected) {
		t.Fatal("revocation reopened", err)
	}
	proposal, err := f.r.Propose("p", "maker", "Synthetic policy", f.spec)
	if err != nil || proposal.State != Revoked {
		t.Fatal("proposal retry reopened", err)
	}
}
