package financialpermission

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
)

type fixture struct {
	now         time.Time
	effects     map[string]Effect
	denied      map[string]bool
	target      string
	applyError  bool
	create      bool
	lookupError bool
	calls       int
}

func (f *fixture) Check(workspace, actor string) (string, error) {
	if workspace != "w1" || f.denied[actor] {
		return "", ErrDenied
	}
	return actor + "-facts", nil
}
func (f *fixture) Current(Spec) (string, error) { return f.target, nil }
func (f *fixture) Lookup(id string) (Effect, error) {
	if f.lookupError {
		return Effect{}, ErrUnknown
	}
	return f.effects[id], nil
}
func (f *fixture) Apply(id string, s Spec) error {
	f.calls++
	if f.create {
		f.effects[id] = Effect{Found: true, Spec: s, Status: ActiveGrant}
	}
	if f.applyError {
		return ErrUnknown
	}
	return nil
}
func setup(t *testing.T) (*Registry, *fixture, Proposal) {
	t.Helper()
	f := &fixture{now: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), effects: map[string]Effect{}, denied: map[string]bool{}, target: "target-v1", create: true}
	r, err := NewRegistry(Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 2}, f, f, f, func() time.Time { return f.now })
	if err != nil {
		t.Fatal(err)
	}
	p := Proposal{ID: "p1", Spec: Spec{Capability: Approve, Workspace: "w1", Kind: DirectHuman, Recipient: "carol", NotBefore: f.now, NotAfter: f.now.Add(12 * time.Hour)}, Proposer: "alice", Rationale: "Explicit direct approval permission"}
	return r, f, p
}
func intent(r Record) Intent {
	return Intent{Workspace: "w1", ID: r.Proposal.ID, Actor: "bob", Digest: r.Digest, Rationale: "Independent direct permission review"}
}
func propose(t *testing.T, r *Registry, p Proposal) Record {
	t.Helper()
	out, err := r.Propose(p)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDirectPermissionShapesAndSeparation(t *testing.T) {
	r, f, p := setup(t)
	for _, change := range []func(*Proposal){
		func(p *Proposal) { p.Spec.Kind = inputauthority.Console }, func(p *Proposal) { p.Spec.Kind = inputauthority.Human; p.Spec.CeilingID = "ceiling" }, func(p *Proposal) { p.Spec.Kind = inputauthority.Human }, func(p *Proposal) { p.Spec.CeilingID = "ceiling" }, func(p *Proposal) { p.Spec.Capability = "inspect" }, func(p *Proposal) { p.Spec.Kind = "unknown" },
	} {
		bad := p
		change(&bad)
		if _, err := r.Propose(bad); !errors.Is(err, ErrInvalid) {
			t.Fatal("mixed shape", bad, err)
		}
	}
	self := p
	self.Proposer = p.Spec.Recipient
	if _, err := r.Propose(self); !errors.Is(err, ErrDenied) {
		t.Fatal("recipient proposed", err)
	}
	record := propose(t, r, p)
	for _, actor := range []string{p.Proposer, p.Spec.Recipient} {
		i := intent(record)
		i.Actor = actor
		if _, err := r.Approve(i); !errors.Is(err, ErrDenied) {
			t.Fatal("self approval", actor, err)
		}
		if _, err := r.Reject(i); !errors.Is(err, ErrDenied) {
			t.Fatal("self reject", actor, err)
		}
	}
	withdrawn := intent(record)
	withdrawn.Actor = p.Spec.Recipient
	if _, err := r.Withdraw(withdrawn); !errors.Is(err, ErrDenied) {
		t.Fatal("recipient withdrawal", err)
	}
	applied, err := r.Approve(intent(record))
	if err != nil || applied.State != Applied || f.calls != 1 || !strings.HasPrefix(applied.Decision.GrantID, "financial-permission-") {
		t.Fatal(applied, err)
	}
	original := *applied.Decision
	applied.Decision.Actor = "mutated"
	again, err := r.Record("w1", p.ID)
	if err != nil || *again.Decision != original {
		t.Fatal("alias", again, err)
	}
	replay, err := r.Approve(intent(record))
	if err != nil || f.calls != 1 || *replay.Decision != original {
		t.Fatal("recreated grant", err)
	}
}

func TestDirectPermissionUnknownRecovery(t *testing.T) {
	for _, status := range []string{ActiveGrant, RevokedGrant, ExpiredGrant} {
		t.Run(status, func(t *testing.T) {
			r, f, p := setup(t)
			record := propose(t, r, p)
			f.applyError = true
			if _, err := r.Approve(intent(record)); !errors.Is(err, ErrUnknown) {
				t.Fatal(err)
			}
			stored, err := r.Record("w1", p.ID)
			if err != nil || stored.State != Unknown {
				t.Fatal(stored, err)
			}
			for _, operation := range []struct {
				run   func(Intent) (Record, error)
				actor string
			}{{r.Reject, "bob"}, {r.Withdraw, p.Proposer}} {
				i := intent(record)
				i.Actor = operation.actor
				if _, err := operation.run(i); !errors.Is(err, ErrConflict) {
					t.Fatal("unknown terminated", err)
				}
			}
			effect := f.effects[stored.Decision.GrantID]
			effect.Status = status
			f.effects[stored.Decision.GrantID] = effect
			f.applyError = false
			changed := intent(record)
			changed.Rationale = "Changed reason"
			if _, err := r.Approve(changed); !errors.Is(err, ErrConflict) {
				t.Fatal("intent rewritten", err)
			}
			f.denied[p.Proposer] = true
			if _, err := r.Approve(intent(record)); !errors.Is(err, ErrDenied) {
				t.Fatal("revoked proposer retry", err)
			}
			f.denied[p.Proposer] = false
			f.lookupError = true
			if _, err := r.Approve(intent(record)); !errors.Is(err, ErrUnknown) {
				t.Fatal("lookup became absence", err)
			}
			f.lookupError = false
			recovered, err := r.Approve(intent(record))
			if err != nil || recovered.State != Applied || recovered.Decision.GrantStatus != status || f.calls != 1 {
				t.Fatal(recovered, err, f.calls)
			}
		})
	}
}

func TestDirectPermissionBeforeEffectAndExactReconciliation(t *testing.T) {
	r, f, p := setup(t)
	record := propose(t, r, p)
	f.create = false
	f.applyError = true
	if _, err := r.Approve(intent(record)); !errors.Is(err, ErrUnknown) {
		t.Fatal(err)
	}
	f.target = "changed"
	if _, err := r.Approve(intent(record)); !errors.Is(err, ErrConflict) || f.calls != 1 {
		t.Fatal("stale target applied", err)
	}
	f.target = "target-v1"
	f.create = true
	f.applyError = false
	made, err := r.Approve(intent(record))
	if err != nil || made.State != Applied || f.calls != 2 {
		t.Fatal(made, err)
	}
	effect := f.effects[made.Decision.GrantID]
	effect.Spec.Recipient = "someone-else"
	f.effects[made.Decision.GrantID] = effect
	if _, err := r.Approve(intent(record)); !errors.Is(err, ErrUnknown) || f.calls != 2 {
		t.Fatal("mismatch recreated", err)
	}
}

func TestDirectPermissionCapacityAndDomains(t *testing.T) {
	r, f, p := setup(t)
	first := propose(t, r, p)
	for n := 2; n <= 3; n++ {
		next := p
		next.ID = fmt.Sprintf("p%d", n)
		_, err := r.Propose(next)
		if n == 2 && err != nil || n == 3 && !errors.Is(err, ErrLimit) {
			t.Fatal(n, err)
		}
	}
	limits := Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 2}
	clock := func() time.Time { return f.now }
	input, err := inputauthority.NewRegistry(limits, f, f, f, clock)
	if err != nil {
		t.Fatal(err)
	}
	pricing, err := pricingauthority.NewRegistry(limits, f, f, f, clock)
	if err != nil {
		t.Fatal(err)
	}
	for _, legacy := range []*Registry{input, pricing} {
		if _, err := legacy.Propose(p); !errors.Is(err, ErrInvalid) {
			t.Fatal("legacy accepted direct shape", err)
		}
	}
	ip := p
	ip.Spec.Kind = inputauthority.Console
	ip.Spec.Recipient = "console"
	ip.Spec.Capability = inputauthority.Inspect
	ir, err := input.Propose(ip)
	if err != nil {
		t.Fatal(err)
	}
	pp := ip
	pp.Spec.Capability = pricingauthority.Approve
	pr, err := pricing.Propose(pp)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest == ir.Digest || first.Digest == pr.Digest {
		t.Fatal("domain collision")
	}
	foreign := intent(first)
	foreign.Digest = pr.Digest
	if _, err := r.Approve(foreign); !errors.Is(err, ErrConflict) {
		t.Fatal("foreign digest", err)
	}
	rows, err := r.Records("w1")
	if err != nil {
		t.Fatal(err)
	}
	copy := rows[0]
	rows[0].Proposal.Spec.Recipient = "alias"
	reread, err := r.Record("w1", copy.Proposal.ID)
	if err != nil || !reflect.DeepEqual(copy, reread) {
		t.Fatal("record alias", err)
	}
}

func TestDirectPendingTerminationAndFreshGovernor(t *testing.T) {
	r, f, p := setup(t)
	record := propose(t, r, p)
	f.denied["bob"] = true
	if _, err := r.Approve(intent(record)); !errors.Is(err, ErrDenied) || f.calls != 0 {
		t.Fatal("revoked confirmer", err)
	}
	f.denied["bob"] = false
	rejected, err := r.Reject(intent(record))
	if err != nil || rejected.State != Rejected || f.calls != 0 {
		t.Fatal(rejected, err)
	}
	p.ID = "withdraw"
	record = propose(t, r, p)
	i := intent(record)
	i.Actor = p.Proposer
	withdrawn, err := r.Withdraw(i)
	if err != nil || withdrawn.State != Withdrawn || f.calls != 0 {
		t.Fatal(withdrawn, err)
	}
}
