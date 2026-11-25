package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/pricingpolicy"
)

func provisionPricingCapability(t *testing.T, f world, capability, recipient string) (string, string) {
	t.Helper()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	fields := pricingTimes(f)
	fields["capability"] = capability
	id := "policy-console-" + capability
	p := f.run(root, "pricing_authority.propose_console", id, fields)
	f.ok(t, "console proposal", p, pricingauthority.Pending, "")
	c := pricingApproveHTTP(f, second, id, p)
	f.ok(t, "console approval", c, pricingauthority.Applied, "")
	fields["recipient"], fields["ceiling"] = recipient, field(c, "grant")
	id = "policy-human-" + capability
	p = f.run(root, "pricing_authority.propose_human", id, fields)
	f.ok(t, "human proposal", p, pricingauthority.Pending, "")
	h := pricingApproveHTTP(f, second, id, p)
	f.ok(t, "human approval", h, pricingauthority.Applied, "")
	return field(c, "grant"), field(h, "grant")
}

func TestPricingPolicyAuthorityActualMemberActions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	a := serverPricingPolicyAuthority{s: f.s}
	scope := pricingpolicy.Scope{Workspace: SampleWorkspace}
	for _, test := range []struct{ capability, action string }{
		{"propose", pricingpolicy.ProposeAction}, {"approve", pricingpolicy.ApproveAction},
		{"revoke", pricingpolicy.RevokeAction}, {"use", pricingpolicy.UseAction},
	} {
		t.Run(test.capability, func(t *testing.T) {
			check := func() error { return a.Check(scope, SampleMember, test.action) }
			if err := inputAuthorityAs(f, member, check); !errors.Is(err, pricingpolicy.ErrRejected) {
				t.Fatal("default authority", err)
			}
			ceiling, _ := provisionPricingCapability(t, f, test.capability, SampleMember)
			if err := inputAuthorityAs(f, member, check); err != nil {
				t.Fatal("ordinary member with separate grants", err)
			}
			if err := f.s.admin.RevokeGrant(ceiling, 1); err != nil {
				t.Fatal(err)
			}
			if err := inputAuthorityAs(f, member, check); !errors.Is(err, pricingpolicy.ErrRejected) {
				t.Fatal("revoked ceiling", err)
			}
		})
	}
	if f.s.chargeProducer.pricingPolicies == nil || setup(t, SeedNone).s.chargeProducer.pricingPolicies != nil {
		t.Fatal("incorrect sample registry boundary")
	}
}

func TestPricingPolicyAuthorityStoredProposer(t *testing.T) {
	for _, cause := range []string{"none", "revoked", "suspended", "removed", "unavailable", "caller-ceiling"} {
		t.Run(cause, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			_, grant := provisionPricingCapability(t, f, "propose", SampleAdmin3)
			ceiling, _ := provisionPricingCapability(t, f, "approve", SampleMember)
			member := f.as(MemberLogin)
			switch cause {
			case "revoked":
				if err := f.s.admin.RevokeGrant(grant, 1); err != nil {
					t.Fatal(err)
				}
			case "suspended":
				u, _, _ := f.s.findUser(SampleAdmin3)
				if err := f.s.identity.Installation().SuspendUser(seedOperator, SampleAdmin3, u.Epoch); err != nil {
					t.Fatal(err)
				}
			case "removed":
				root := f.as(AdminLogin).verifier(f)
				members, err := f.s.roster(root)
				if err != nil {
					t.Fatal(err)
				}
				if err := f.s.identity.Workspace().RemoveMembership(root, SampleWorkspace, SampleAdmin3, members[SampleAdmin3].Version); err != nil {
					t.Fatal(err)
				}
			case "unavailable":
				f.s.fault = func(at string) bool { return at == "users" }
			case "caller-ceiling":
				if err := f.s.admin.RevokeGrant(ceiling, 1); err != nil {
					t.Fatal(err)
				}
			}
			a := serverPricingPolicyAuthority{s: f.s}
			err := inputAuthorityAs(f, member, func() error {
				return a.Check(pricingpolicy.Scope{Workspace: SampleWorkspace}, SampleAdmin3, pricingpolicy.ProposeAction)
			})
			if cause == "none" {
				if err != nil {
					t.Fatal("member approver without proposal permission", err)
				}
			} else if cause == "unavailable" {
				if !errors.Is(err, gate.ErrUnavailable) {
					t.Fatal(err)
				}
			} else if !errors.Is(err, pricingpolicy.ErrRejected) {
				t.Fatal("stale proposer/caller accepted", err)
			}
		})
	}
}

func TestPricingPolicyAuthorityInspectionAndSession(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member := f.as(MemberLogin)
	_, grant := provisionPricingCapability(t, f, "inspect", SampleAdmin3)
	provisionPricingCapability(t, f, "use", SampleMember)
	a := serverPricingPolicyAuthority{s: f.s}
	terms, _ := pricingGrantAuthority.terms(SampleWorkspace, "inspect")
	if err := f.s.admin.AddDelegation(gate.Delegation{ID: "policy-inspect-delegated", DelegatorID: SampleAdmin3, DelegateID: SampleMember, Terms: terms, Validity: gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	err := inputAuthorityAs(f, member, func() error {
		d := f.s.gate.Decide(f.s.answer, ConsoleClient, gate.Request{Action: gate.ViewCost, Kind: pricingauthority.ResourceKind, Fields: []string{pricingauthority.EvidenceField}, Resource: gate.Resource{WorkspaceID: SampleWorkspace}})
		if !d.Allowed || len(d.Delegations) == 0 {
			t.Fatal("delegated fixture not effective", d)
		}
		return a.inspect(SampleWorkspace)
	})
	if !errors.Is(err, pricingpolicy.ErrRejected) {
		t.Fatal("delegated policy inspection", err)
	}
	third := f.as(Admin3Login)
	if err := inputAuthorityAs(f, third, func() error { return a.inspect(SampleWorkspace) }); err != nil {
		t.Fatal(err)
	}
	if err := f.s.admin.RevokeGrant(grant, 1); err != nil {
		t.Fatal(err)
	}
	if err := inputAuthorityAs(f, third, func() error { return a.inspect(SampleWorkspace) }); !errors.Is(err, pricingpolicy.ErrRejected) {
		t.Fatal(err)
	}
	if err := inputAuthorityAs(f, member, func() error {
		for _, check := range []func() error{
			func() error { return a.Check(pricingpolicy.Scope{Workspace: SampleWorkspace}, SampleMember, "unknown") },
			func() error {
				return a.Check(pricingpolicy.Scope{Workspace: "other"}, SampleMember, pricingpolicy.UseAction)
			},
			func() error {
				return a.Check(pricingpolicy.Scope{Workspace: SampleWorkspace}, SampleAdmin3, pricingpolicy.UseAction)
			},
		} {
			if !errors.Is(check(), pricingpolicy.ErrRejected) {
				t.Fatal("invalid caller/action accepted")
			}
		}
		if err := a.Check(pricingpolicy.Scope{Workspace: SampleWorkspace}, SampleMember, pricingpolicy.UseAction); err != nil {
			t.Fatal("live caller prerequisite", err)
		}
		f.s.dropSession(f.s.caller, "")
		return a.Check(pricingpolicy.Scope{Workspace: SampleWorkspace}, SampleMember, pricingpolicy.UseAction)
	}); !errors.Is(err, pricingpolicy.ErrRejected) {
		t.Fatal("ended session", err)
	}
	if pricingPolicyDecision(true, gate.Allowed, []string{"delegated"}) == nil {
		t.Fatal("delegation accepted")
	}
}
