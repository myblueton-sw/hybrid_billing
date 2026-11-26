package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func grantDraftCapability(f world, capability string) inputauthority.Record {
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	defer f.s.endProbe(root.verifier(f))
	defer f.s.endProbe(second.verifier(f))
	spec := draftPermissionSpec(f, capability)
	p, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: capability + "-ceiling", Spec: spec, Rationale: "Explicit draft ceiling"})
	f.must(err)
	c, err := draftPermissionDecision(f, second, p)
	f.must(err)
	spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, SampleMember, c.Decision.GrantID
	p, err = draftPermissionProposal(f, root, inputauthority.Proposal{ID: capability + "-human", Spec: spec, Rationale: "Explicit draft operator"})
	f.must(err)
	p, err = draftPermissionDecision(f, second, p)
	f.must(err)
	return p
}

func TestFinancialDraftOperationNeedsSeparateInspection(t *testing.T) {
	for _, first := range []string{draftauthority.Submit, draftauthority.Inspect} {
		t.Run(first, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			grantDraftCapability(f, first)
			member := f.as(MemberLogin)
			check := func(capability string) error {
				return inputAuthorityAs(f, member, func() error { return f.s.financialIssuance.draftOperation(SampleWorkspace, SampleMember, capability) })
			}
			documentFailure(t, check(draftauthority.Submit), commands.PermissionDenied)
			second := draftauthority.Inspect
			if first == second {
				second = draftauthority.Submit
			}
			grantDraftCapability(f, second)
			f.must(check(draftauthority.Submit))
			documentFailure(t, check(draftauthority.Approve), commands.PermissionDenied)
			documentFailure(t, check(draftauthority.Inspect), commands.InvalidRequest)
			for _, g := range f.s.admin.Grants() {
				if g.SubjectID == SampleMember && g.Terms.Kind == draftauthority.ResourceKind && len(g.Terms.Fields) == 1 {
					f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
				}
			}
			documentFailure(t, check(draftauthority.Submit), commands.PermissionDenied)
		})
	}
}

func TestFinancialDraftOwnershipAndSessionRefusals(t *testing.T) {
	for _, mode := range []string{"binding", "ceiling", "actor", "session"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			grantDraftCapability(f, draftauthority.Submit)
			inspection := grantDraftCapability(f, draftauthority.Inspect)
			member := f.as(MemberLogin)
			err := inputAuthorityAs(f, member, func() error {
				a := f.s.financialIssuance
				f.must(a.draftOperation(SampleWorkspace, SampleMember, draftauthority.Submit))
				actor := SampleMember
				expected := commands.PermissionDenied
				switch mode {
				case "binding":
					delete(f.s.draftAuthority.bindings, inspection.Decision.GrantID)
				case "ceiling":
					delete(f.s.draftAuthority.ceilings, inspection.Proposal.Spec.CeilingID)
				case "actor":
					actor = SampleAdmin
				case "session":
					f.s.endProbe(member.verifier(f))
					expected = commands.Unauthenticated
				}
				documentFailure(t, a.draftOperation(SampleWorkspace, actor, draftauthority.Submit), expected)
				return nil
			})
			f.must(err)
		})
	}
}

func TestFinancialDraftApprovalInspectionRevocation(t *testing.T) {
	for _, revoked := range []string{draftauthority.Approve, draftauthority.Inspect} {
		t.Run(revoked, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			rights := map[string]inputauthority.Record{}
			for _, cap := range []string{draftauthority.Approve, draftauthority.Inspect} {
				rights[cap] = grantDraftCapability(f, cap)
			}
			member := f.as(MemberLogin)
			check := func() error {
				return inputAuthorityAs(f, member, func() error {
					return f.s.financialIssuance.draftOperation(SampleWorkspace, SampleMember, draftauthority.Approve)
				})
			}
			f.must(check())
			g, ok := f.s.grantView(rights[revoked].Decision.GrantID)
			if !ok {
				t.Fatal("missing actual right")
			}
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
			documentFailure(t, check(), commands.PermissionDenied)
		})
	}
}
