package server

import (
	"errors"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"testing"
	"time"
)

func TestFinancialIssuanceDraftAccessRejectsPriorMembershipGrant(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	grantDraftCapability(f, draftauthority.Submit)
	member := f.as(MemberLogin)
	var old gate.GrantView
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember && g.Terms.Action == gate.SubmitIssuanceDraft {
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
		req := gate.Request{Action: gate.SubmitIssuanceDraft, Kind: draftauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("current unowned grant did not satisfy general gate")
		}
		return nil
	}))
	err := inputAuthorityAs(f, member, func() error {
		return f.s.financialIssuance.draftCaller(SampleWorkspace, SampleMember, draftauthority.Submit)
	})
	documentFailure(t, err, commands.PermissionDenied)
}

func TestFinancialIssuanceDraftAccessUnknownHumanGrant(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	grantDraftCapability(f, draftauthority.Submit)
	member := f.as(MemberLogin)
	var spec inputauthority.Spec
	for id, binding := range f.s.draftAuthority.bindings {
		if binding.Kind == inputauthority.Human && binding.Recipient == SampleMember {
			spec = binding
			g, ok := f.s.grantView(id)
			if !ok {
				t.Fatal("missing human grant")
			}
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	proposed, err := draftPermissionProposal(f, f.as(AdminLogin), inputauthority.Proposal{ID: "unknown-new-human", Spec: spec, Rationale: "Explicit replacement human permission"})
	f.must(err)
	f.s.fault = func(at string) bool { return at == "issuance-draft-authority-after-grant" }
	_, err = draftPermissionDecision(f, f.as(Admin2Login), proposed)
	if err == nil {
		t.Fatal("unknown write reported success")
	}
	f.s.fault = nil
	f.must(inputAuthorityAs(f, member, func() error {
		req := gate.Request{Action: gate.SubmitIssuanceDraft, Kind: draftauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("module write did not create actual grant")
		}
		return nil
	}))
	err = inputAuthorityAs(f, member, func() error {
		return f.s.financialIssuance.draftCaller(SampleWorkspace, SampleMember, draftauthority.Submit)
	})
	documentFailure(t, err, commands.PermissionDenied)
	_, err = draftPermissionDecision(f, f.as(Admin2Login), proposed)
	f.must(err)
	err = inputAuthorityAs(f, member, func() error {
		return f.s.financialIssuance.draftCaller(SampleWorkspace, SampleMember, draftauthority.Submit)
	})
	f.must(err)
}

func TestFinancialDraftInspectionRejectsActualDelegation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	terms, err := draftAuthorityTerms(SampleWorkspace, draftauthority.Inspect)
	f.must(err)
	validity := gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "delegation-ceiling", ClientID: ConsoleClient, Terms: terms, Validity: validity}))
	f.must(f.s.admin.AddGrant(gate.Grant{ID: "delegator-inspection", SubjectID: SampleAdmin, Terms: terms, Validity: validity}))
	grantDraftCapability(f, draftauthority.Submit)
	f.must(f.s.admin.AddDelegation(gate.Delegation{ID: "draft-inspection-delegation", DelegatorID: SampleAdmin, DelegateID: SampleMember, Terms: terms, Validity: validity}))
	member := f.as(MemberLogin)
	f.must(inputAuthorityAs(f, member, func() error {
		req := gate.Request{Action: gate.ViewCost, Kind: draftauthority.ResourceKind, Fields: []string{draftauthority.EvidenceField}, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		got := f.s.gate.Decide(f.s.answer, ConsoleClient, req)
		if !got.Allowed || len(got.Delegations) == 0 {
			t.Fatal("fixture did not establish actual delegated inspection")
		}
		documentFailure(t, f.s.financialIssuance.draftCaller(SampleWorkspace, SampleMember, draftauthority.Inspect), commands.PermissionDenied)
		return nil
	}))
}

func TestFinancialDraftInspectDomainIsolation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	spec := draftPermissionSpec(f, draftauthority.Inspect)
	seenDigests, seenEffects := map[string]bool{}, map[string]bool{}
	for _, domain := range []grantAuthorityDomain{draftGrantAuthority, resultGrantAuthority, inputGrantAuthority} {
		authority := f.s.grantAuthority(domain)
		var p inputauthority.Record
		f.must(inputAuthorityAs(f, root, func() error {
			var err error
			p, err = authority.proposeLocked(inputauthority.Proposal{ID: "same-inspect", Spec: spec, Proposer: SampleAdmin, Rationale: "Identical domain isolation input"})
			return err
		}))
		f.must(inputAuthorityAs(f, second, func() error {
			var err error
			p, err = authority.approveLocked(inputauthority.Intent{Workspace: SampleWorkspace, ID: p.Proposal.ID, Digest: p.Digest, Actor: SampleAdmin2, Rationale: "Independent identical inspection review"})
			return err
		}))
		if seenDigests[p.Digest] || seenEffects[p.Decision.GrantID] {
			t.Fatal("authority domain collision")
		}
		seenDigests[p.Digest], seenEffects[p.Decision.GrantID] = true, true
		human := spec
		human.Kind, human.Recipient, human.CeilingID = inputauthority.Human, SampleMember, p.Decision.GrantID
		if domain == draftGrantAuthority {
			owned, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: "owned-inspect-human", Spec: human, Rationale: "Actual eligible inspection recipient"})
			f.must(err)
			_, err = draftPermissionDecision(f, second, owned)
			f.must(err)
		}
		if domain != draftGrantAuthority {
			human := spec
			human.Kind, human.Recipient, human.CeilingID = inputauthority.Human, SampleMember, p.Decision.GrantID
			_, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: "foreign-inspect", Spec: human, Rationale: "Foreign inspect ceiling must refuse"})
			if !errors.Is(err, inputauthority.ErrDenied) {
				t.Fatal("foreign inspect ceiling not denied", err)
			}
		}
	}
}
