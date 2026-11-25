package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func draftPermissionProposal(f world, who actor, p inputauthority.Proposal) (inputauthority.Record, error) {
	var out inputauthority.Record
	err := inputAuthorityAs(f, who, func() error {
		p.Proposer = f.s.answer.User()
		var err error
		out, err = f.s.draftAuthority.proposeLocked(p)
		return err
	})
	return out, err
}
func draftPermissionDecision(f world, who actor, p inputauthority.Record) (inputauthority.Record, error) {
	var out inputauthority.Record
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.draftAuthority.approveLocked(inputauthority.Intent{Workspace: SampleWorkspace, ID: p.Proposal.ID, Digest: p.Digest, Actor: f.s.answer.User(), Rationale: "Independent draft permission review"})
		return err
	})
	return out, err
}
func draftPermissionSpec(f world, capability string) inputauthority.Spec {
	return inputauthority.Spec{Workspace: SampleWorkspace, Capability: capability, Kind: inputauthority.Console, Recipient: ConsoleClient, NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
}
func TestDraftAuthorityActualHumanAndConsole(t *testing.T) {
	expected := []struct {
		capability string
		action     gate.Action
	}{{"submit", gate.SubmitIssuanceDraft}, {"approve", gate.ApproveIssuanceDraft}, {"inspect", gate.ViewCost}}
	for _, want := range expected {
		t.Run(want.capability, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
			for _, g := range f.s.admin.Grants() {
				if draftGrantTerms(g.Terms.Action, g.Terms.Kind, g.Terms.Fields) {
					t.Fatal("seeded draft permission")
				}
			}
			spec := draftPermissionSpec(f, want.capability)
			p, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: "ceiling", Spec: spec, Rationale: "Explicit capability ceiling"})
			f.must(err)
			_, err = draftPermissionDecision(f, root, p)
			if !errors.Is(err, inputauthority.ErrDenied) {
				t.Fatalf("self grant approval: %v", err)
			}
			ceiling, err := draftPermissionDecision(f, second, p)
			f.must(err)
			terms, err := draftAuthorityTerms(SampleWorkspace, want.capability)
			f.must(err)
			if terms.Action != want.action || terms.Kind != "synthetic_issuance_draft" {
				t.Fatal("incorrect fixed mapping")
			}
			if want.capability == "inspect" {
				if len(terms.Fields) != 1 || terms.Fields[0] != "issuance_draft" {
					t.Fatal("wrong Cost field")
				}
			} else if len(terms.Fields) != 0 {
				t.Fatal("mutation has fields")
			}
			req := gate.Request{Fields: terms.Fields, Action: want.action, Kind: "synthetic_issuance_draft", Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
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
			p, err = draftPermissionProposal(f, root, inputauthority.Proposal{ID: "human", Spec: spec, Rationale: "Explicit human capability"})
			f.must(err)
			granted, err := draftPermissionDecision(f, second, p)
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
						bad.Fields = nil
						if other.capability == "inspect" {
							bad.Fields = []string{"issuance_draft"}
						}
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
			f.code(t, "generic field bypass", f.requestGrant(root, SampleMember, map[string]string{"action": "view_cost", "kind": ReaderKind, "fields": "issuance_draft"}), 422, "invalid_request")
			f.code(t, "generic kind bypass", f.requestGrant(root, SampleMember, map[string]string{"action": "request_approval", "kind": "synthetic_issuance_draft", "fields": ""}), 422, "invalid_request")
			c, ok := f.s.grantView(ceiling.Decision.GrantID)
			if !ok {
				t.Fatal("missing actual ceiling")
			}
			f.must(f.s.admin.RevokeGrant(c.ID, c.Version))
			check(false)
		})
	}
}

func TestDraftAuthorityUnknownEffectRecovery(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: "uncertain", Spec: draftPermissionSpec(f, draftauthority.Submit), Rationale: "Explicit draft ceiling"})
	f.must(err)
	f.s.fault = func(at string) bool { return at == "issuance-draft-authority-after-grant" }
	_, err = draftPermissionDecision(f, second, p)
	if err == nil {
		t.Fatal("lost response reported success")
	}
	f.s.fault = nil
	id := "issuance-draft-grant-" + p.Digest
	if _, ok := f.s.grantView(id); !ok {
		t.Fatal("actual uncertain grant absent")
	}
	if len(f.s.draftAuthority.ceilings) != 0 {
		t.Fatal("unknown ceiling published")
	}
	recovered, err := draftPermissionDecision(f, second, p)
	f.must(err)
	if recovered.State != inputauthority.Applied || recovered.Decision.GrantID != id || len(f.s.draftAuthority.ceilings) != 1 || len(f.s.draftAuthority.bindings) != 1 {
		t.Fatal("uncertain effect not reconciled")
	}
}

func TestDraftAuthorityClosedFormsAndNonSample(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, capability := range []string{"", "register_channel", "issue", "confirm", "apply", "approve_issuance_term", "register_issuance_channel"} {
		_, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: "invalid", Spec: draftPermissionSpec(f, capability), Rationale: "Rejected unknown capability"})
		if !errors.Is(err, inputauthority.ErrInvalid) {
			t.Fatalf("unknown capability %s: %v", capability, err)
		}
	}
	spec := draftPermissionSpec(f, draftauthority.Submit)
	spec.Kind = "direct_human_grant"
	spec.Recipient = SampleAdmin
	_, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: "direct", Spec: spec, Rationale: "No ceiling bypass"})
	if !errors.Is(err, inputauthority.ErrInvalid) {
		t.Fatal("direct mode accepted", err)
	}
	none := setup(t, SeedNone).ready()
	if none.s.draftAuthority != nil {
		t.Fatal("non-sample authority initialized")
	}
	if len(f.s.draftAuthority.bindings) != 0 {
		t.Fatal("invalid intent retained grant binding")
	}
}

func TestDraftAuthorityForeignCeilingAndRuleSeparation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	foreign, err := termPermissionProposal(f, root, inputauthority.Proposal{ID: "term-ceiling", Spec: termPermissionSpec(f, "record"), Rationale: "Independent term domain ceiling"})
	f.must(err)
	foreign, err = termPermissionDecision(f, second, foreign)
	f.must(err)
	spec := draftPermissionSpec(f, draftauthority.Approve)
	spec.Kind = inputauthority.Human
	spec.Recipient = SampleMember
	spec.CeilingID = foreign.Decision.GrantID
	_, err = draftPermissionProposal(f, root, inputauthority.Proposal{ID: "foreign", Spec: spec, Rationale: "Reject foreign term ceiling"})
	if !errors.Is(err, inputauthority.ErrDenied) {
		t.Fatal("foreign ceiling accepted", err)
	}
	p, err := draftPermissionProposal(f, root, inputauthority.Proposal{ID: "draft-ceiling", Spec: draftPermissionSpec(f, draftauthority.Approve), Rationale: "Explicit draft approval ceiling"})
	f.must(err)
	ceiling, err := draftPermissionDecision(f, second, p)
	f.must(err)
	spec.CeilingID = ceiling.Decision.GrantID
	p, err = draftPermissionProposal(f, root, inputauthority.Proposal{ID: "draft-human", Spec: spec, Rationale: "Explicit draft approver"})
	f.must(err)
	granted, err := draftPermissionDecision(f, second, p)
	f.must(err)
	req := gate.Request{Action: gate.ApproveIssuanceDraft, Kind: draftauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
	f.must(inputAuthorityAs(f, member, func() error {
		if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("actual governed approver refused")
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
