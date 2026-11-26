package grantauthority

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type domainPort struct{ effects map[string]Effect }

func (p *domainPort) Check(workspace, actor string) (string, error) { return actor + "-facts", nil }
func (p *domainPort) Current(s Spec) (string, error)                { return "target-facts", nil }
func (p *domainPort) Lookup(id string) (Effect, error)              { return p.effects[id], nil }
func (p *domainPort) Apply(id string, s Spec) error {
	p.effects[id] = Effect{Found: true, Spec: s, Status: ActiveGrant}
	return nil
}
func domainRegistry(d Domain, p *domainPort) (*Registry, error) {
	return NewRegistry(d, Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 10}, p, p, p, func() time.Time { return time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC) })
}
func domainProposal() Proposal {
	at := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	return Proposal{ID: "p1", Spec: Spec{Capability: "inspect", Workspace: "w1", Kind: Console, Recipient: "console", NotBefore: at, NotAfter: at.Add(12 * time.Hour)}, Proposer: "alice", Rationale: "Reviewed authority"}
}
func TestDomainConfigurationBoundsAndIsolation(t *testing.T) {
	for _, d := range []Domain{
		{}, {Version: "v1", GrantPrefix: "grant-"},
		{Version: "bad/version", GrantPrefix: "grant-", Capabilities: []string{"inspect"}},
		{Version: "v1", GrantPrefix: strings.Repeat("x", 65), Capabilities: []string{"inspect"}},
		{Version: "v1", GrantPrefix: "bad/", Capabilities: []string{"inspect"}},
		{Version: "v1", GrantPrefix: "grant-", Capabilities: []string{"inspect", "inspect"}},
		{Version: "v1", GrantPrefix: "grant-", Capabilities: []string{""}},
		{Version: "v1", GrantPrefix: "grant-", Capabilities: make([]string, 33)},
	} {
		if _, err := domainRegistry(d, &domainPort{effects: map[string]Effect{}}); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid domain accepted", err)
		}
	}
	caps := []string{"inspect"}
	r, err := domainRegistry(Domain{Version: "v1", GrantPrefix: "grant-", Capabilities: caps}, &domainPort{effects: map[string]Effect{}})
	if err != nil {
		t.Fatal(err)
	}
	caps[0] = "other"
	if _, err := r.Propose(domainProposal()); err != nil {
		t.Fatal("caller changed registry capabilities", err)
	}
	p := domainProposal()
	p.ID = "p2"
	p.Spec.Capability = "other"
	if _, err := r.Propose(p); !errors.Is(err, ErrInvalid) {
		t.Fatal("mutated capability accepted", err)
	}
}
func TestDomainVersionSeparatesDecisionAndEffectIdentity(t *testing.T) {
	port := &domainPort{effects: map[string]Effect{}}
	first, err := domainRegistry(Domain{Version: "domain-one-v1", GrantPrefix: "one-grant-", Capabilities: []string{"inspect"}}, port)
	if err != nil {
		t.Fatal(err)
	}
	second, err := domainRegistry(Domain{Version: "domain-two-v1", GrantPrefix: "two-grant-", Capabilities: []string{"inspect"}}, port)
	if err != nil {
		t.Fatal(err)
	}
	a, err := first.Propose(domainProposal())
	if err != nil {
		t.Fatal(err)
	}
	b, err := second.Propose(domainProposal())
	if err != nil {
		t.Fatal(err)
	}
	if a.Digest == b.Digest {
		t.Fatal("version namespaces collided")
	}
	intent := Intent{Workspace: "w1", ID: "p1", Actor: "bob", Digest: a.Digest, Rationale: "Reviewed decision"}
	if _, err := second.Approve(intent); !errors.Is(err, ErrConflict) {
		t.Fatal("foreign decision accepted", err)
	}
	done, err := first.Approve(intent)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := port.effects["two-grant-"+b.Digest]; ok {
		t.Fatal("foreign effect present")
	}
	intent.Digest = b.Digest
	other, err := second.Approve(intent)
	if err != nil {
		t.Fatal(err)
	}
	if len(port.effects) != 2 || done.Decision.GrantID == other.Decision.GrantID {
		t.Fatal("foreign effect reused")
	}
}

func TestClosedModesAndImmutableSelection(t *testing.T) {
	port := &domainPort{effects: map[string]Effect{}}
	domain := Domain{Mode: Mode(2), Version: "direct-v1", GrantPrefix: "direct-", Capabilities: []string{"inspect"}}
	if _, err := domainRegistry(domain, port); !errors.Is(err, ErrInvalid) {
		t.Fatal("unknown mode", err)
	}
	domain.Mode = DirectHumanMode
	r, err := domainRegistry(domain, port)
	if err != nil {
		t.Fatal(err)
	}
	domain.Mode = CeilingMode
	p := domainProposal()
	p.Spec.Kind = DirectHuman
	p.Spec.Recipient = "carol"
	if _, err := r.Propose(p); err != nil {
		t.Fatal("caller changed mode", err)
	}
	p.ID = "legacy"
	p.Spec.Kind = Console
	if _, err := r.Propose(p); !errors.Is(err, ErrInvalid) {
		t.Fatal("mixed mode", err)
	}
}
