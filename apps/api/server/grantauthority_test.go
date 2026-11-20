package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"testing"
	"time"
)

func TestGrantAuthorityDomainIsClosed(t *testing.T) {
	f, _, root, _ := inputAuthoritySetup(t)
	limits := inputauthority.Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 20}
	for _, domain := range []grantAuthorityDomain{0, 2, 255} {
		if _, err := newServerGrantAuthority(f.s, limits, domain); !errors.Is(err, inputauthority.ErrInvalid) {
			t.Fatal("unknown domain constructed", err)
		}
		if _, err := domain.terms(SampleWorkspace, inputauthority.Confirm); !errors.Is(err, inputauthority.ErrInvalid) {
			t.Fatal("unknown domain mapped", err)
		}
	}
	a, err := newServerInputAuthority(f.s, limits)
	if err != nil {
		t.Fatal(err)
	}
	for _, capability := range []string{"", "approve", "view_cost", "confirm_input"} {
		spec := inputAuthorityProposal(f, "invalid", inputauthority.Console, "").Spec
		spec.Capability = capability
		if _, err := a.grant("invalid", spec); !errors.Is(err, inputauthority.ErrInvalid) {
			t.Fatal("unknown capability mapped", err)
		}
		if a.sameGrant(gate.GrantView{}, "invalid", spec) {
			t.Fatal("invalid grant matched")
		}
		err := inputAuthorityAs(f, root, func() error { _, err := a.Current(spec); return err })
		if !errors.Is(err, inputauthority.ErrInvalid) {
			t.Fatal("unknown target capability", err)
		}
	}
	first, err := inputGrantAuthority.terms(SampleWorkspace, inputauthority.Inspect)
	if err != nil {
		t.Fatal(err)
	}
	first.Fields[0] = "changed"
	second, err := inputGrantAuthority.terms(SampleWorkspace, inputauthority.Inspect)
	if err != nil || len(second.Fields) != 1 || second.Fields[0] != inputauthority.EvidenceField {
		t.Fatal("mapping fields aliased", err)
	}
}
func TestGrantAuthorityInstanceDoesNotAdoptForeignCeilingsOrBindings(t *testing.T) {
	f, a, root, second := inputAuthoritySetup(t)
	limits := inputauthority.Limits{Lifetime: time.Hour, MaxGrantValidity: 24 * time.Hour, Capacity: 20}
	other, err := newServerInputAuthority(f.s, limits)
	if err != nil {
		t.Fatal(err)
	}
	p := proposeInputAuthority(t, f, a, root, inputAuthorityProposal(f, "owned", inputauthority.Console, ""))
	done := approveInputAuthority(t, f, a, second, p)
	id := done.Decision.GrantID
	err = inputAuthorityAs(f, root, func() error {
		if _, err := other.Lookup(id); !errors.Is(err, inputauthority.ErrConflict) {
			t.Fatal("foreign binding adopted", err)
		}
		human := inputAuthorityProposal(f, "human", inputauthority.Human, id)
		if _, err := other.proposeLocked(human); !errors.Is(err, inputauthority.ErrDenied) {
			t.Fatal("foreign approved ceiling adopted", err)
		}
		if err := other.Apply(id, p.Proposal.Spec); !errors.Is(err, inputauthority.ErrConflict) {
			t.Fatal("foreign effect applied", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(other.bindings) != 0 || len(other.ceilings) != 0 {
		t.Fatal("foreign ownership retained")
	}
}
