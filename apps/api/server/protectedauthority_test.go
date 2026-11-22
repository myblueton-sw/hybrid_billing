package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func protectedPermissionProposal(f world, who actor, p inputauthority.Proposal) (inputauthority.Record, error) {
	var out inputauthority.Record
	err := inputAuthorityAs(f, who, func() error {
		p.Proposer = f.s.answer.User()
		var err error
		out, err = f.s.protectedAuthority.proposeLocked(p)
		return err
	})
	return out, err
}
func protectedPermissionDecision(f world, who actor, p inputauthority.Record) (inputauthority.Record, error) {
	var out inputauthority.Record
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.protectedAuthority.approveLocked(inputauthority.Intent{Workspace: SampleWorkspace, ID: p.Proposal.ID, Digest: p.Digest, Actor: f.s.answer.User(), Rationale: "Independent protected permission review"})
		return err
	})
	return out, err
}
func protectedPermissionSpec(f world, capability string) inputauthority.Spec {
	return inputauthority.Spec{Workspace: SampleWorkspace, Capability: capability, Kind: inputauthority.Console, Recipient: ConsoleClient, NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
}
func TestProtectedAuthorityActualHumanAndConsole(t *testing.T) {
	expected := []struct {
		capability string
		action     gate.Action
	}{{"register_channel", gate.RegisterIssuanceChannel}, {"admit_channel", gate.AdmitIssuanceChannel}, {"request_change", gate.RequestIssuanceProtectedChange}, {"verify_change", gate.VerifyIssuanceProtectedChange}, {"approve_change", gate.ApproveIssuanceProtectedChange}}
	for _, want := range expected {
		t.Run(want.capability, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
			for _, g := range f.s.admin.Grants() {
				if protectedGrantTerms(g.Terms.Action, g.Terms.Kind, g.Terms.Fields) {
					t.Fatal("seeded protected permission")
				}
			}
			spec := protectedPermissionSpec(f, want.capability)
			p, err := protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "ceiling", Spec: spec, Rationale: "Explicit capability ceiling"})
			f.must(err)
			_, err = protectedPermissionDecision(f, root, p)
			if !errors.Is(err, inputauthority.ErrDenied) {
				t.Fatalf("self grant approval: %v", err)
			}
			ceiling, err := protectedPermissionDecision(f, second, p)
			f.must(err)
			terms, err := protectedAuthorityTerms(SampleWorkspace, want.capability)
			f.must(err)
			if terms.Action != want.action || terms.Kind != "synthetic_issuance_protected" || len(terms.Fields) != 0 {
				t.Fatal("incorrect fixed mapping")
			}
			req := gate.Request{Action: want.action, Kind: "synthetic_issuance_protected", Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
			check := func(allowed bool) {
				t.Helper()
				f.must(inputAuthorityAs(f, member, func() error {
					if f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed != allowed {
						t.Fatal("wrong actual grant decision")
					}
					return nil
				}))
			}
			check(false)
			spec.Kind = inputauthority.Human
			spec.Recipient = SampleMember
			spec.CeilingID = ceiling.Decision.GrantID
			p, err = protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "human", Spec: spec, Rationale: "Explicit human capability"})
			f.must(err)
			granted, err := protectedPermissionDecision(f, second, p)
			f.must(err)
			check(true)
			g, ok := f.s.grantView(granted.Decision.GrantID)
			if !ok || g.SubjectID != SampleMember || g.ClientID != "" || g.Terms.Action != want.action {
				t.Fatal("wrong actual human grant")
			}
			f.must(inputAuthorityAs(f, member, func() error {
				for _, other := range expected {
					if other.action != want.action {
						bad := req
						bad.Action = other.action
						if f.s.gate.Decide(f.s.answer, ConsoleClient, bad).Allowed {
							t.Fatal("cross capability authority")
						}
					}
				}
				bad := req
				bad.Kind = "erp_final_approval"
				if f.s.gate.Decide(f.s.answer, ConsoleClient, bad).Allowed {
					t.Fatal("cross resource authority")
				}
				return nil
			}))
			f.code(t, "generic action bypass", f.requestGrant(root, SampleMember, map[string]string{"action": string(want.action), "kind": ReaderKind, "fields": ""}), 422, "invalid_request")
			f.code(t, "generic kind bypass", f.requestGrant(root, SampleMember, map[string]string{"action": "request_approval", "kind": "synthetic_issuance_protected", "fields": ""}), 422, "invalid_request")
			c, ok := f.s.grantView(ceiling.Decision.GrantID)
			if !ok {
				t.Fatal("missing actual ceiling")
			}
			f.must(f.s.admin.RevokeGrant(c.ID, c.Version))
			check(false)
		})
	}
}

func TestProtectedAuthorityUnknownEffectRecovery(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p, err := protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "uncertain", Spec: protectedPermissionSpec(f, protectedauthority.RegisterChannel), Rationale: "Explicit protected ceiling"})
	f.must(err)
	f.s.fault = func(at string) bool { return at == "issuance-protected-authority-after-grant" }
	_, err = protectedPermissionDecision(f, second, p)
	if err == nil {
		t.Fatal("lost response reported success")
	}
	f.s.fault = nil
	id := "issuance-protected-grant-" + p.Digest
	if _, ok := f.s.grantView(id); !ok {
		t.Fatal("actual uncertain grant absent")
	}
	if len(f.s.protectedAuthority.ceilings) != 0 {
		t.Fatal("unknown ceiling published")
	}
	recovered, err := protectedPermissionDecision(f, second, p)
	f.must(err)
	if recovered.State != inputauthority.Applied || recovered.Decision.GrantID != id || len(f.s.protectedAuthority.ceilings) != 1 || len(f.s.protectedAuthority.bindings) != 1 {
		t.Fatal("uncertain effect not reconciled")
	}
}

func TestProtectedAuthorityClosedFormsAndNonSample(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, capability := range []string{"", "inspect", "issue", "confirm", "apply", "approve_issuance_term", "register_issuance_channel"} {
		_, err := protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "invalid", Spec: protectedPermissionSpec(f, capability), Rationale: "Rejected unknown capability"})
		if !errors.Is(err, inputauthority.ErrInvalid) {
			t.Fatalf("unknown capability %s: %v", capability, err)
		}
	}
	spec := protectedPermissionSpec(f, protectedauthority.RegisterChannel)
	spec.Kind = "direct_human_grant"
	spec.Recipient = SampleAdmin
	_, err := protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "direct", Spec: spec, Rationale: "No ceiling bypass"})
	if !errors.Is(err, inputauthority.ErrInvalid) {
		t.Fatal("direct mode accepted", err)
	}
	none := setup(t, SeedNone).ready()
	if none.s.protectedAuthority != nil {
		t.Fatal("non-sample authority initialized")
	}
	if len(f.s.protectedAuthority.bindings) != 0 {
		t.Fatal("invalid intent retained grant binding")
	}
}

func TestProtectedAuthorityForeignCeilingAndRuleSeparation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	foreign, err := termPermissionProposal(f, root, inputauthority.Proposal{ID: "term-ceiling", Spec: termPermissionSpec(f, "record"), Rationale: "Independent term domain ceiling"})
	f.must(err)
	foreign, err = termPermissionDecision(f, second, foreign)
	f.must(err)
	spec := protectedPermissionSpec(f, protectedauthority.VerifyChange)
	spec.Kind = inputauthority.Human
	spec.Recipient = SampleMember
	spec.CeilingID = foreign.Decision.GrantID
	_, err = protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "foreign", Spec: spec, Rationale: "Reject foreign term ceiling"})
	if !errors.Is(err, inputauthority.ErrDenied) {
		t.Fatal("foreign ceiling accepted", err)
	}
	p, err := protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "protected-ceiling", Spec: protectedPermissionSpec(f, protectedauthority.VerifyChange), Rationale: "Explicit protected verification ceiling"})
	f.must(err)
	ceiling, err := protectedPermissionDecision(f, second, p)
	f.must(err)
	spec.CeilingID = ceiling.Decision.GrantID
	p, err = protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "protected-human", Spec: spec, Rationale: "Explicit protected verifier"})
	f.must(err)
	granted, err := protectedPermissionDecision(f, second, p)
	f.must(err)
	req := gate.Request{Action: gate.VerifyIssuanceProtectedChange, Kind: protectedauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
	f.must(inputAuthorityAs(f, member, func() error {
		if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("actual governed verifier refused")
		}
		return nil
	}))
	_, err = proposeRulesAs(f, member, "unauthorized-rules", issuanceRules())
	if !errors.Is(err, inputauthority.ErrDenied) {
		t.Fatal("business permission authorized rules", err)
	}
	if f.s.financialIssuance.protectedRules != nil {
		t.Fatal("business user reserved rules")
	}
	g, ok := f.s.grantView(granted.Decision.GrantID)
	if !ok {
		t.Fatal("missing human grant")
	}
	f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
	f.must(inputAuthorityAs(f, member, func() error {
		if f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("revoked human permission allowed")
		}
		return nil
	}))
}
