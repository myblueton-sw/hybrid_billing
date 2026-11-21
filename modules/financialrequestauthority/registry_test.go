package financialrequestauthority

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
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
	p := Proposal{ID: "p1", Spec: Spec{Capability: Request, Workspace: "w1", Kind: Console, Recipient: "console", NotBefore: at, NotAfter: at.Add(12 * time.Hour)}, Proposer: "alice", Rationale: "Reviewed request permission"}
	return r, f, p, limits, clock
}
func intent(r Record) Intent {
	return Intent{Workspace: "w1", ID: r.Proposal.ID, Actor: "bob", Digest: r.Digest, Rationale: "Independent review"}
}

func TestFixedRequestCapabilityAndIndependentGovernors(t *testing.T) {
	r, f, p, _, _ := setup(t)
	for _, capability := range []string{"", "approve", "inspect", "request_approval", "use"} {
		bad := p
		bad.Spec.Capability = capability
		if _, err := r.Propose(bad); !errors.Is(err, ErrInvalid) {
			t.Fatal("open capability", capability, err)
		}
	}
	bad := p
	bad.Spec.Kind = financialpermission.DirectHuman
	if _, err := r.Propose(bad); !errors.Is(err, ErrInvalid) {
		t.Fatal("direct decision shape accepted", err)
	}
	bad = p
	bad.Spec.CeilingID = "parent"
	if _, err := r.Propose(bad); !errors.Is(err, ErrInvalid) {
		t.Fatal("console has parent ceiling", err)
	}
	q, err := r.Propose(p)
	if err != nil {
		t.Fatal(err)
	}
	self := intent(q)
	self.Actor = p.Proposer
	if _, err := r.Approve(self); !errors.Is(err, ErrDenied) {
		t.Fatal("self approval", err)
	}
	f.denied[p.Proposer] = true
	if _, err := r.Approve(intent(q)); !errors.Is(err, ErrDenied) {
		t.Fatal("lost governor", err)
	}
	delete(f.denied, p.Proposer)
	got, err := r.Approve(intent(q))
	if err != nil || got.State != Applied || !strings.HasPrefix(got.Decision.GrantID, "financial-request-") || len(f.effects) != 1 {
		t.Fatal(got, err)
	}
	if _, err := r.Approve(intent(q)); err != nil || len(f.effects) != 1 {
		t.Fatal("duplicate effect", err)
	}
}

func TestRequestAndDecisionDomainsRemainSeparate(t *testing.T) {
	r, f, p, limits, clock := setup(t)
	direct, err := financialpermission.NewRegistry(limits, f, f, f, clock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := direct.Propose(p); !errors.Is(err, ErrInvalid) {
		t.Fatal("request became direct decision", err)
	}
	p.Spec.Kind = financialpermission.DirectHuman
	p.Spec.Capability = financialpermission.Approve
	p.Spec.Recipient = "carol"
	if _, err := r.Propose(p); !errors.Is(err, ErrInvalid) {
		t.Fatal("decision became request", err)
	}
	if _, err := direct.Propose(p); err != nil {
		t.Fatal("direct decision contract changed", err)
	}
}
