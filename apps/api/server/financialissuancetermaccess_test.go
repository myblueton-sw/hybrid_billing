package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func grantTermCapability(t *testing.T, f world, capability, recipient string) {
	t.Helper()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	spec := termPermissionSpec(f, capability)
	p, err := termPermissionProposal(f, root, inputauthority.Proposal{ID: capability + "-ceiling", Spec: spec, Rationale: "Explicit synthetic term capability"})
	f.must(err)
	ceiling, err := termPermissionDecision(f, second, p)
	f.must(err)
	spec.Kind = inputauthority.Human
	spec.Recipient = recipient
	spec.CeilingID = ceiling.Decision.GrantID
	p, err = termPermissionProposal(f, root, inputauthority.Proposal{ID: capability + "-human-" + recipient, Spec: spec, Rationale: "Explicit synthetic term operator"})
	f.must(err)
	_, err = termPermissionDecision(f, second, p)
	f.must(err)
}
func termProfileWorld(t *testing.T) (world, actor, financialIssuanceProfile) {
	t.Helper()
	f, second, first, digest := issuanceSourceWorld(t)
	p, err := proposeIssuanceProfile(f, first, digest, issuanceSpec())
	f.must(err)
	p, err = confirmIssuanceProfile(f, second, p)
	f.must(err)
	grantTermCapability(t, f, termsauthority.RecordTerm, SampleMember)
	return f, f.as(MemberLogin), p
}
func consumeTermProfile(f world, who actor, p financialIssuanceProfile) (financialIssuanceProfile, error) {
	var out financialIssuanceProfile
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.financialIssuance.termProfile(SampleWorkspace, p.Spec.ID, p.Digest, f.s.answer.User(), termsauthority.RecordTerm)
		return err
	})
	return out, err
}
func TestFinancialIssuanceTermProfileOrdinaryOperator(t *testing.T) {
	f, member, p := termProfileWorld(t)
	f.must(inputAuthorityAs(f, member, func() error {
		decision := f.s.gate.Decide(f.s.answer, ConsoleClient, gate.Request{Action: gate.AdministerInstallation, Kind: gate.InstallationKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}})
		if decision.Allowed {
			t.Fatal("ordinary operator gained administrator authority")
		}
		return nil
	}))
	before, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	got, err := consumeTermProfile(f, member, p)
	f.must(err)
	if !reflect.DeepEqual(got, p) {
		t.Fatal("ordinary operator changed profile facts")
	}
	got.Source.Basis.Rows[0].Net = "999"
	again, err := consumeTermProfile(f, member, p)
	f.must(err)
	if !reflect.DeepEqual(again, p) {
		t.Fatal("consumer aliased retained profile")
	}
	after, err := f.s.approvals.RequestOf(SampleWorkspace, "issuance-final")
	f.must(err)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("consumption spent financial approval")
	}
	f.must(inputAuthorityAs(f, member, func() error {
		if f.s.financialIssuance.profileCaller(SampleWorkspace, SampleMember) == nil {
			t.Fatal("business authority became profile governance")
		}
		return nil
	}))
}
func TestFinancialIssuanceTermProfileLostStanding(t *testing.T) {
	for _, mode := range []string{"proposer", "confirmer", "membership", "demoted", "user", "console", "term_human", "term_ceiling", "unavailable"} {
		t.Run(mode, func(t *testing.T) {
			f, member, p := termProfileWorld(t)
			switch mode {
			case "proposer", "confirmer", "console", "term_human", "term_ceiling":
				target := SampleAdmin
				if mode == "confirmer" {
					target = SampleAdmin3
				}
				changed := 0
				for _, g := range f.s.admin.Grants() {
					match := g.Terms.Action == gate.AdministerInstallation && g.SubjectID == target
					if mode == "console" {
						match = g.Terms.Action == gate.AdministerInstallation && g.ClientID == ConsoleClient
					}
					if mode == "term_human" {
						match = g.Terms.Action == gate.RecordIssuanceTerm && g.SubjectID == SampleMember
					}
					if mode == "term_ceiling" {
						match = g.Terms.Action == gate.RecordIssuanceTerm && g.ClientID == ConsoleClient
					}
					if match {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
						changed++
					}
				}
				if changed == 0 {
					t.Fatal("no actual grant revoked")
				}
			case "demoted":
				f.must(f.s.identity.Workspace().ChangeRole(f.as(Admin2Login).verifier(f), SampleWorkspace, SampleAdmin3, identity.Member, f.membership(SampleAdmin3).Version))
			case "membership":
				f.must(f.s.identity.Workspace().SuspendMembership(f.as(Admin2Login).verifier(f), SampleWorkspace, SampleAdmin3, f.membership(SampleAdmin3).Version))
			case "user":
				u, _, err := f.s.findUser(SampleAdmin3)
				f.must(err)
				f.must(f.s.identity.Installation().SuspendUser(seedOperator, SampleAdmin3, u.Epoch))
			case "unavailable":
				f.s.fault = func(at string) bool { return at == "users" }
			}
			_, err := consumeTermProfile(f, member, p)
			expected := commands.PermissionDenied
			if mode == "unavailable" {
				expected = commands.AuthorityUnavailable
			}
			documentFailure(t, err, expected)
			if !reflect.DeepEqual(f.s.financialIssuance.profiles[p.Spec.ID], p) {
				t.Fatal("standing refusal mutated retained profile")
			}
		})
	}
}
func TestFinancialIssuanceTermProfileBoundary(t *testing.T) {
	f, member, p := termProfileWorld(t)
	f.s.fault = func(at string) bool { return at == "users" }
	missing := p
	missing.Spec.ID = "foreign-profile"
	_, err := consumeTermProfile(f, member, missing)
	documentFailure(t, err, commands.EvidenceStale)
	f.s.fault = nil
	err = inputAuthorityAs(f, member, func() error {
		_, err := f.s.financialIssuance.termProfile("foreign", p.Spec.ID, p.Digest, SampleMember, termsauthority.RecordTerm)
		return err
	})
	documentFailure(t, err, commands.PermissionDenied)
	err = inputAuthorityAs(f, member, func() error {
		return f.s.financialIssuance.termCaller(SampleWorkspace, SampleMember, termsauthority.BindTerm)
	})
	documentFailure(t, err, commands.PermissionDenied)
	err = inputAuthorityAs(f, member, func() error {
		f.s.sessions.remove(f.s.caller)
		_, err := f.s.financialIssuance.termProfile(SampleWorkspace, p.Spec.ID, p.Digest, SampleMember, termsauthority.RecordTerm)
		return err
	})
	documentFailure(t, err, commands.Unauthenticated)
}
func TestFinancialIssuanceTermProfileRejectsUnownedGrant(t *testing.T) {
	f, member, p := termProfileWorld(t)
	for id, spec := range f.s.termAuthority.bindings {
		if spec.Kind == inputauthority.Human {
			delete(f.s.termAuthority.bindings, id)
		}
	}
	_, err := consumeTermProfile(f, member, p)
	documentFailure(t, err, commands.PermissionDenied)
}

func TestFinancialIssuanceTermProfileRejectsPriorMembershipGrant(t *testing.T) {
	f, member, p := termProfileWorld(t)
	var old gate.GrantView
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember && g.Terms.Action == gate.RecordIssuanceTerm {
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
		req := gate.Request{Action: gate.RecordIssuanceTerm, Kind: termsauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("current unowned grant did not satisfy general gate")
		}
		return nil
	}))
	_, err := consumeTermProfile(f, member, p)
	documentFailure(t, err, commands.PermissionDenied)
}

func TestFinancialIssuanceTermActorDoesNotImpersonateSession(t *testing.T) {
	f, member, _ := termProfileWorld(t)
	f.s.sessions.remove(member.verifier(f))
	f.must(inputAuthorityAs(f, f.as(Admin2Login), func() error {
		return f.s.financialIssuance.termActor(SampleWorkspace, SampleMember, termsauthority.RecordTerm)
	}))
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleMember && g.Terms.Action == gate.RecordIssuanceTerm {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	err := inputAuthorityAs(f, f.as(Admin2Login), func() error {
		return f.s.financialIssuance.termActor(SampleWorkspace, SampleMember, termsauthority.RecordTerm)
	})
	documentFailure(t, err, commands.PermissionDenied)
}

func TestFinancialIssuanceTermProfileUnknownHumanGrant(t *testing.T) {
	f, member, p := termProfileWorld(t)
	var spec inputauthority.Spec
	for id, binding := range f.s.termAuthority.bindings {
		if binding.Kind == inputauthority.Human && binding.Recipient == SampleMember {
			spec = binding
			g, ok := f.s.grantView(id)
			if !ok {
				t.Fatal("missing human grant")
			}
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	proposed, err := termPermissionProposal(f, f.as(AdminLogin), inputauthority.Proposal{ID: "unknown-new-human", Spec: spec, Rationale: "Explicit replacement human permission"})
	f.must(err)
	f.s.fault = func(at string) bool { return at == "issuance-term-authority-after-grant" }
	_, err = termPermissionDecision(f, f.as(Admin2Login), proposed)
	if err == nil {
		t.Fatal("unknown write reported success")
	}
	f.s.fault = nil
	f.must(inputAuthorityAs(f, member, func() error {
		req := gate.Request{Action: gate.RecordIssuanceTerm, Kind: termsauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("module write did not create actual grant")
		}
		return nil
	}))
	_, err = consumeTermProfile(f, member, p)
	documentFailure(t, err, commands.PermissionDenied)
	_, err = termPermissionDecision(f, f.as(Admin2Login), proposed)
	f.must(err)
	_, err = consumeTermProfile(f, member, p)
	f.must(err)
}
