package server

import (
	"errors"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func termPermissionProposal(f world, who actor, p inputauthority.Proposal) (inputauthority.Record, error) {
	var out inputauthority.Record
	err := inputAuthorityAs(f, who, func() error {
		p.Proposer = f.s.answer.User()
		var err error
		out, err = f.s.termAuthority.proposeLocked(p)
		return err
	})
	return out, err
}
func termPermissionDecision(f world, who actor, p inputauthority.Record) (inputauthority.Record, error) {
	var out inputauthority.Record
	err := inputAuthorityAs(f, who, func() error {
		var err error
		out, err = f.s.termAuthority.approveLocked(inputauthority.Intent{Workspace: SampleWorkspace, ID: p.Proposal.ID, Digest: p.Digest, Actor: f.s.answer.User(), Rationale: "Independent term permission review"})
		return err
	})
	return out, err
}
func termPermissionSpec(f world, capability string) inputauthority.Spec {
	return inputauthority.Spec{Workspace: SampleWorkspace, Capability: capability, Kind: inputauthority.Console, Recipient: ConsoleClient, NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}
}
func TestTermAuthorityActualHumanAndConsole(t *testing.T) {
	expected := []struct {
		capability string
		action     gate.Action
	}{{"record", gate.RecordIssuanceTerm}, {"approve", gate.ApproveIssuanceTerm}, {"bind", gate.BindIssuanceTerm}, {"approve_binding", gate.ApproveIssuanceTermBinding}, {"inspect", gate.ViewCost}}
	for _, want := range expected {
		t.Run(want.capability, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
			for _, g := range f.s.admin.Grants() {
				if termGrantTerms(g.Terms.Action, g.Terms.Kind, g.Terms.Fields) {
					t.Fatal("seeded term permission")
				}
			}
			spec := termPermissionSpec(f, want.capability)
			p, err := termPermissionProposal(f, root, inputauthority.Proposal{ID: "ceiling", Spec: spec, Rationale: "Explicit capability ceiling"})
			f.must(err)
			_, err = termPermissionDecision(f, root, p)
			if !errors.Is(err, inputauthority.ErrDenied) {
				t.Fatalf("self grant approval: %v", err)
			}
			ceiling, err := termPermissionDecision(f, second, p)
			f.must(err)
			terms, err := termAuthorityTerms(SampleWorkspace, want.capability)
			f.must(err)
			if terms.Action != want.action || terms.Kind != "synthetic_issuance_terms" {
				t.Fatal("incorrect fixed mapping")
			}
			if want.capability == "inspect" {
				if len(terms.Fields) != 1 || terms.Fields[0] != "issuance_terms" {
					t.Fatal("wrong evidence field")
				}
			} else if len(terms.Fields) != 0 {
				t.Fatal("mutation has fields")
			}
			req := gate.Request{Fields: terms.Fields, Action: want.action, Kind: "synthetic_issuance_terms", Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
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
			p, err = termPermissionProposal(f, root, inputauthority.Proposal{ID: "human", Spec: spec, Rationale: "Explicit human capability"})
			f.must(err)
			granted, err := termPermissionDecision(f, second, p)
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
							bad.Fields = []string{"issuance_terms"}
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
			f.code(t, "generic kind bypass", f.requestGrant(root, SampleMember, map[string]string{"action": "request_approval", "kind": "synthetic_issuance_terms", "fields": ""}), 422, "invalid_request")
			c, ok := f.s.grantView(ceiling.Decision.GrantID)
			if !ok {
				t.Fatal("missing actual ceiling")
			}
			f.must(f.s.admin.RevokeGrant(c.ID, c.Version))
			check(false)
		})
	}
}

func TestTermAuthorityUnknownEffectRecovery(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p, err := termPermissionProposal(f, root, inputauthority.Proposal{ID: "uncertain", Spec: termPermissionSpec(f, termsauthority.RecordTerm), Rationale: "Explicit term ceiling"})
	f.must(err)
	f.s.fault = func(at string) bool { return at == "issuance-term-authority-after-grant" }
	_, err = termPermissionDecision(f, second, p)
	if err == nil {
		t.Fatal("lost response reported success")
	}
	f.s.fault = nil
	id := "issuance-term-grant-" + p.Digest
	if _, ok := f.s.grantView(id); !ok {
		t.Fatal("actual uncertain grant absent")
	}
	if len(f.s.termAuthority.ceilings) != 0 {
		t.Fatal("unknown ceiling published")
	}
	recovered, err := termPermissionDecision(f, second, p)
	f.must(err)
	if recovered.State != inputauthority.Applied || recovered.Decision.GrantID != id || len(f.s.termAuthority.ceilings) != 1 || len(f.s.termAuthority.bindings) != 1 {
		t.Fatal("uncertain effect not reconciled")
	}
}

func TestTermAuthorityClosedFormsAndNonSample(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	for _, capability := range []string{"", "inspect_all", "issue", "confirm", "apply", "approve_issuance_term"} {
		_, err := termPermissionProposal(f, root, inputauthority.Proposal{ID: "invalid", Spec: termPermissionSpec(f, capability), Rationale: "Rejected unknown capability"})
		if !errors.Is(err, inputauthority.ErrInvalid) {
			t.Fatalf("unknown capability %s: %v", capability, err)
		}
	}
	spec := termPermissionSpec(f, termsauthority.RecordTerm)
	spec.Kind = "direct_human_grant"
	spec.Recipient = SampleAdmin
	_, err := termPermissionProposal(f, root, inputauthority.Proposal{ID: "direct", Spec: spec, Rationale: "No ceiling bypass"})
	if !errors.Is(err, inputauthority.ErrInvalid) {
		t.Fatal("direct mode accepted", err)
	}
	none := setup(t, SeedNone).ready()
	if none.s.termAuthority != nil {
		t.Fatal("non-sample authority initialized")
	}
	if len(f.s.termAuthority.bindings) != 0 {
		t.Fatal("invalid intent retained grant binding")
	}
}

func TestTermAuthorityForeignCeilingAndRevokedHuman(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	spec := termPermissionSpec(f, termsauthority.ApproveTerm)
	var foreign inputauthority.Record
	f.must(inputAuthorityAs(f, root, func() error {
		var err error
		foreign, err = f.s.pricingAuthority.proposeLocked(inputauthority.Proposal{ID: "foreign-ceiling", Spec: spec, Proposer: SampleAdmin, Rationale: "Separate pricing capability"})
		return err
	}))
	f.must(inputAuthorityAs(f, second, func() error {
		var err error
		foreign, err = f.s.pricingAuthority.approveLocked(inputauthority.Intent{Workspace: SampleWorkspace, ID: foreign.Proposal.ID, Digest: foreign.Digest, Actor: SampleAdmin2, Rationale: "Independent pricing review"})
		return err
	}))
	human := spec
	human.Kind = inputauthority.Human
	human.Recipient = SampleMember
	human.CeilingID = foreign.Decision.GrantID
	_, err := termPermissionProposal(f, root, inputauthority.Proposal{ID: "foreign-human", Spec: human, Rationale: "Cannot use pricing ceiling"})
	if !errors.Is(err, inputauthority.ErrDenied) {
		t.Fatal("foreign ceiling not denied", err)
	}
	p, err := termPermissionProposal(f, root, inputauthority.Proposal{ID: "foreign-ceiling", Spec: spec, Rationale: "Separate pricing capability"})
	f.must(err)
	ceiling, err := termPermissionDecision(f, second, p)
	f.must(err)
	if p.Digest == foreign.Digest || ceiling.Decision.GrantID == foreign.Decision.GrantID {
		t.Fatal("domain identity collision")
	}
	human.CeilingID = ceiling.Decision.GrantID
	p, err = termPermissionProposal(f, root, inputauthority.Proposal{ID: "term-human", Spec: human, Rationale: "Explicit term human"})
	f.must(err)
	granted, err := termPermissionDecision(f, second, p)
	f.must(err)
	req := gate.Request{Action: gate.ApproveIssuanceTerm, Kind: termsauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
	f.must(inputAuthorityAs(f, member, func() error {
		if !f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("owned current rights denied")
		}
		return nil
	}))
	g, ok := f.s.grantView(granted.Decision.GrantID)
	if !ok {
		t.Fatal("actual human grant absent")
	}
	f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
	f.must(inputAuthorityAs(f, member, func() error {
		if f.s.gate.Decide(f.s.answer, ConsoleClient, req).Allowed {
			t.Fatal("revoked human grant allowed")
		}
		return nil
	}))
}
