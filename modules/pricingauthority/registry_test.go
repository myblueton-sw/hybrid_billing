package pricingauthority

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"strings"
	"testing"
	"time"
)

type fixture struct {
	effects map[string]Effect
	denied  map[string]bool
}

func (f *fixture) Check(workspace, actor string) (string, error) {
	if workspace != "w1" || f.denied[actor] {
		return "", ErrDenied
	}
	return actor + "-current", nil
}
func (f *fixture) Current(s Spec) (string, error)   { return "current-target", nil }
func (f *fixture) Lookup(id string) (Effect, error) { return f.effects[id], nil }
func (f *fixture) Apply(id string, s Spec) error {
	f.effects[id] = Effect{Found: true, Spec: s, Status: ActiveGrant}
	return nil
}
func setup(t *testing.T) (*Registry, *fixture, Proposal, Limits, func() time.Time) {
	t.Helper()
	at := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return at }
	f := &fixture{effects: map[string]Effect{}, denied: map[string]bool{}}
	limits := Limits{Capacity: 20, Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour}
	r, err := NewRegistry(limits, f, f, f, clock)
	if err != nil {
		t.Fatal(err)
	}
	p := Proposal{ID: "p1", Spec: Spec{Capability: Inspect, Workspace: "w1", Kind: Console, Recipient: "console", NotBefore: at, NotAfter: at.Add(12 * time.Hour)}, Proposer: "alice", Rationale: "Reviewed pricing scope"}
	return r, f, p, limits, clock
}
func intent(r Record) Intent {
	return Intent{Workspace: "w1", ID: r.Proposal.ID, Actor: "bob", Digest: r.Digest, Rationale: "Independent review"}
}
func TestClosedCapabilitiesAndIndependentGovernors(t *testing.T) {
	for _, capability := range []string{Propose, Approve, Revoke, Use, Inspect} {
		t.Run(capability, func(t *testing.T) {
			r, f, p, _, _ := setup(t)
			p.Spec.Capability = capability
			q, err := r.Propose(p)
			if err != nil {
				t.Fatal(err)
			}
			self := intent(q)
			self.Actor = "alice"
			if _, err := r.Approve(self); !errors.Is(err, ErrDenied) {
				t.Fatal("self approval", err)
			}
			f.denied["alice"] = true
			if _, err := r.Approve(intent(q)); !errors.Is(err, ErrDenied) {
				t.Fatal("revoked proposer", err)
			}
			delete(f.denied, "alice")
			got, err := r.Approve(intent(q))
			if err != nil || got.State != Applied || !strings.HasPrefix(got.Decision.GrantID, "pricing-grant-") {
				t.Fatal("pricing grant failed", err)
			}
			if len(f.effects) != 1 || f.effects[got.Decision.GrantID].Spec.Capability != capability {
				t.Fatal("wrong capability effect")
			}
			again, err := r.Approve(intent(q))
			if err != nil || again.Decision.GrantID != got.Decision.GrantID || len(f.effects) != 1 {
				t.Fatal("retry changed effect", err)
			}
		})
	}
	for _, capability := range []string{"", "confirm", "approve_pricing_policy", "view_cost", "unknown"} {
		r, _, p, _, _ := setup(t)
		p.Spec.Capability = capability
		if _, err := r.Propose(p); !errors.Is(err, ErrInvalid) {
			t.Fatal("open capability", capability, err)
		}
	}
}
func TestInputAndPricingAuthorityCannotSubstitute(t *testing.T) {
	pricing, f, p, limits, clock := setup(t)
	input, err := inputauthority.NewRegistry(limits, f, f, f, clock)
	if err != nil {
		t.Fatal(err)
	}
	a, err := pricing.Propose(p)
	if err != nil {
		t.Fatal(err)
	}
	b, err := input.Propose(p)
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest == b.Digest {
		t.Fatal("domain digest collision")
	}
	if _, err := pricing.Approve(intent(b)); !errors.Is(err, ErrConflict) {
		t.Fatal("input decision approved pricing", err)
	}
	if _, err := input.Approve(intent(a)); !errors.Is(err, ErrConflict) {
		t.Fatal("pricing decision approved input", err)
	}
	x, err := pricing.Approve(intent(a))
	if err != nil {
		t.Fatal(err)
	}
	y, err := input.Approve(intent(b))
	if err != nil {
		t.Fatal(err)
	}
	if x.Decision.GrantID == y.Decision.GrantID || len(f.effects) != 2 {
		t.Fatal("effect namespace collision")
	}
	p.ID = "p2"
	p.Spec.Capability = Approve
	if _, err := input.Propose(p); !errors.Is(err, ErrInvalid) {
		t.Fatal("input wrapper expanded", err)
	}
}
