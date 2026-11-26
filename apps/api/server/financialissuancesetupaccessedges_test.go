package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
	"testing"
	"time"
)

func grantSetupInspectionForTest(f world, protected bool) {
	proposal, decision, spec := termPermissionProposal, termPermissionDecision, termPermissionSpec(f, "inspect")
	if protected {
		proposal, decision, spec = protectedPermissionProposal, protectedPermissionDecision, protectedPermissionSpec(f, "inspect")
	}
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p, err := proposal(f, root, inputauthority.Proposal{ID: "edge-ceiling", Spec: spec, Rationale: "Explicit evidence ceiling"})
	f.must(err)
	c, err := decision(f, second, p)
	f.must(err)
	spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, SampleMember, c.Decision.GrantID
	p, err = proposal(f, root, inputauthority.Proposal{ID: "edge-human", Spec: spec, Rationale: "Explicit evidence recipient"})
	f.must(err)
	_, err = decision(f, second, p)
	f.must(err)
}
func TestFinancialSetupTermAccessRejectsPriorMembershipGrant(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	grantSetupInspectionForTest(f, false)
	member := f.as(MemberLogin)
	var old gate.GrantView
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember && g.Terms.Action == gate.ViewCost {
			old = g
		}
	}
	if old.ID == "" {
		t.Fatal("owned human grant missing")
	}
	root := f.as(AdminLogin)
	f.must(f.s.identity.Workspace().RemoveMembership(root.verifier(f), SampleWorkspace, SampleMember, f.membership(SampleMember).Version))
	invitation := verifierOf([]byte("synthetic-term-rejoin"))
	f.must(f.s.identity.Workspace().Invite(root.verifier(f), SampleWorkspace, "term-rejoin", identity.Member, invitation, identity.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}, SampleMember))
	f.must(f.s.identity.Workspace().Accept(SampleWorkspace, "term-rejoin", invitation, member.verifier(f)))
	stale, ok := f.s.grantView(old.ID)
	if !ok || !stale.Active || stale.Effective {
		t.Fatal("fixture did not retain active ineffective old grant")
	}
	unrelated := old.Grant
	unrelated.ID = "unowned-current-term-grant"
	f.must(f.s.admin.AddGrant(unrelated))
	member = f.as(MemberLogin)
	f.must(inputAuthorityAs(f, member, func() error {
		req := gate.Request{Fields: []string{termsauthority.EvidenceField}, Action: gate.ViewCost, Kind: termsauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("current unowned grant did not satisfy general gate")
		}
		return nil
	}))
	err := inputAuthorityAs(f, member, func() error {
		return f.s.financialIssuance.termReadCaller(SampleWorkspace, SampleMember)
	})
	documentFailure(t, err, commands.PermissionDenied)
}

func TestFinancialSetupTermInspectionRejectsActualDelegation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	terms, err := termAuthorityTerms(SampleWorkspace, termsauthority.Inspect)
	f.must(err)
	validity := gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "delegation-ceiling", ClientID: ConsoleClient, Terms: terms, Validity: validity}))
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "delegator-inspection", SubjectID: SampleAdmin, Terms: terms, Validity: validity}))
	unrelated, err := termAuthorityTerms(SampleWorkspace, termsauthority.RecordTerm)
	f.must(err)
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "unrelated-member-standing", SubjectID: SampleMember, Terms: unrelated, Validity: validity}))
	f.must(f.s.admin.AddDelegation(gate.Delegation{ID: "draft-inspection-delegation", DelegatorID: SampleAdmin, DelegateID: SampleMember, Terms: terms, Validity: validity}))
	member := f.as(MemberLogin)
	f.must(inputAuthorityAs(f, member, func() error {
		req := gate.Request{Action: gate.ViewCost, Kind: termsauthority.ResourceKind, Fields: []string{termsauthority.EvidenceField}, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		got := f.s.gate.Decide(f.s.answer, ConsoleClient, req)
		if !got.Allowed || len(got.Delegations) == 0 {
			t.Fatal("fixture did not establish actual delegated inspection")
		}
		documentFailure(t, f.s.financialIssuance.termReadCaller(SampleWorkspace, SampleMember), commands.PermissionDenied)
		return nil
	}))
}

func TestFinancialSetupProtectedAccessRejectsPriorMembershipGrant(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	grantSetupInspectionForTest(f, true)
	member := f.as(MemberLogin)
	var old gate.GrantView
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember && g.Terms.Action == gate.ViewCost {
			old = g
		}
	}
	if old.ID == "" {
		t.Fatal("owned human grant missing")
	}
	root := f.as(AdminLogin)
	f.must(f.s.identity.Workspace().RemoveMembership(root.verifier(f), SampleWorkspace, SampleMember, f.membership(SampleMember).Version))
	invitation := verifierOf([]byte("synthetic-term-rejoin"))
	f.must(f.s.identity.Workspace().Invite(root.verifier(f), SampleWorkspace, "term-rejoin", identity.Member, invitation, identity.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}, SampleMember))
	f.must(f.s.identity.Workspace().Accept(SampleWorkspace, "term-rejoin", invitation, member.verifier(f)))
	stale, ok := f.s.grantView(old.ID)
	if !ok || !stale.Active || stale.Effective {
		t.Fatal("fixture did not retain active ineffective old grant")
	}
	unrelated := old.Grant
	unrelated.ID = "unowned-current-term-grant"
	f.must(f.s.admin.AddGrant(unrelated))
	member = f.as(MemberLogin)
	f.must(inputAuthorityAs(f, member, func() error {
		req := gate.Request{Fields: []string{protectedauthority.EvidenceField}, Action: gate.ViewCost, Kind: protectedauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("current unowned grant did not satisfy general gate")
		}
		return nil
	}))
	err := inputAuthorityAs(f, member, func() error {
		return f.s.financialIssuance.protectedReadCaller(SampleWorkspace, SampleMember)
	})
	documentFailure(t, err, commands.PermissionDenied)
}

func TestFinancialSetupProtectedInspectionRejectsActualDelegation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	terms, err := protectedAuthorityTerms(SampleWorkspace, protectedauthority.Inspect)
	f.must(err)
	validity := gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "delegation-ceiling", ClientID: ConsoleClient, Terms: terms, Validity: validity}))
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "delegator-inspection", SubjectID: SampleAdmin, Terms: terms, Validity: validity}))
	unrelated, err := termAuthorityTerms(SampleWorkspace, termsauthority.RecordTerm)
	f.must(err)
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "unrelated-member-standing", SubjectID: SampleMember, Terms: unrelated, Validity: validity}))
	f.must(f.s.admin.AddDelegation(gate.Delegation{ID: "draft-inspection-delegation", DelegatorID: SampleAdmin, DelegateID: SampleMember, Terms: terms, Validity: validity}))
	member := f.as(MemberLogin)
	f.must(inputAuthorityAs(f, member, func() error {
		req := gate.Request{Action: gate.ViewCost, Kind: protectedauthority.ResourceKind, Fields: []string{protectedauthority.EvidenceField}, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		got := f.s.gate.Decide(f.s.answer, ConsoleClient, req)
		if !got.Allowed || len(got.Delegations) == 0 {
			t.Fatal("fixture did not establish actual delegated inspection")
		}
		documentFailure(t, f.s.financialIssuance.protectedReadCaller(SampleWorkspace, SampleMember), commands.PermissionDenied)
		return nil
	}))
}
