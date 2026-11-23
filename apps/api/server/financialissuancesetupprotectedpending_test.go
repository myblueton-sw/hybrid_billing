package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func TestFinancialSetupProtectedPendingRulesActors(t *testing.T) {
	f, reader, c := confirmedSetupConfiguration(t)
	a := f.s.financialIssuance
	intent := financialProtectedRules{ID: "pending-rules", Workspace: SampleWorkspace, Proposer: SampleAdmin, Reason: "Explicit synthetic protected rules", Rules: issuanceRules(), State: "registration_unknown"}
	var err error
	intent.Digest, err = protectedRuleDigest(intent)
	f.must(err)
	a.protectedRules = &intent
	got := protectedSetupRead(f, reader, "protected", c, nil)
	if got.status != 200 || field(got, "rules_state") != "registration_unknown" || field(got, "rules_observation") != "absent" || field(got, "rules_proposer") != SampleAdmin || field(got, "rules_id") != "pending-rules" || field(got, "rules_proposed_at") != "" || field(got, "rules_confirmer") != "" || field(got, "rules_pending_confirmer") != "" {
		t.Fatal(got.raw)
	}
	a.protectedRules = nil
	p, err := proposeRulesAs(f, f.as(AdminLogin), "pending-rules", issuanceRules())
	f.must(err)
	p.State, p.Confirmer = "confirmation_unknown", SampleAdmin2
	a.protectedRules = &p
	got = protectedSetupRead(f, reader, "protected", c, nil)
	if got.status != 200 || field(got, "rules_state") != "confirmation_unknown" || field(got, "rules_observation") != "pending" || field(got, "rules_pending_confirmer") != SampleAdmin2 || field(got, "rules_confirmer") != "" || field(got, "rules_confirmed_at") != "" || field(got, "rules_proposed_at") == "" {
		t.Fatal("pinned confirmer shown as completed", got.raw)
	}
	if a.protectedRules.State != "confirmation_unknown" {
		t.Fatal("read reconciled rules")
	}
}

func TestFinancialSetupProtectedPendingChannelActors(t *testing.T) {
	f, registrar, _, spec := configurationChannelWorld(t)
	grantSetupInspectionForTest(f, true)
	reader := f.as(MemberLogin)
	a := f.s.financialIssuance
	c := a.configurations[spec.Selector.Configuration.ID]
	p, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	if p.Registrar != SampleMember {
		t.Fatal("fixture reader is not the registrar")
	}
	absent := copyChannelRecord(p)
	absent.Spec.ID, absent.Module, absent.State = "absent-channel", protected.ChannelView{}, "registration_unknown"
	absent.Digest, err = channelDigest(absent)
	f.must(err)
	a.channels[absent.Spec.ID] = absent
	pinned := copyChannelRecord(p)
	pinned.State, pinned.Admitter, pinned.Admission = "admission_unknown", "configured-channel-admitter", "synthetic-admission"
	a.channels[p.Spec.ID] = pinned

	got := protectedSetupRead(f, reader, "protected_channels", c, nil)
	first, second := protectedSetupRow(t, got, 0), protectedSetupRow(t, got, 1)
	if field(got, "count") != "2" || field(got, "current_eligibility") != "unchecked" || first["channel"] != "absent-channel" || first["state"] != "registration_unknown" || first["observation"] != "absent" || first["registrar"] != SampleMember || first["registered_at"] != "" || first["pending_admitter"] != "" || first["request_not_before"] != "" {
		t.Fatalf("absent row %v", first)
	}
	if second["state"] != "admission_unknown" || second["observation"] != "registered" || second["registrar"] != SampleMember || second["pending_admitter"] != "configured-channel-admitter" || second["admitter"] != "" || second["admitted_at"] != "" || second["request_not_before"] != "" {
		t.Fatalf("pinned admitter shown as completed %v", second)
	}
	if _, err := a.fields.Register(SampleWorkspace, absent.channel(), protected.Actor{ID: "foreign-registrar", Kind: protected.KindHuman}); err != nil {
		t.Fatal(err)
	}
	f.code(t, "unretained module channel under an absent intent", protectedSetupRead(f, reader, "protected_channels", c, nil), 409, string(commands.EvidenceStale))
}

func TestFinancialSetupProtectedPendingChangeActors(t *testing.T) {
	template := map[string]string{"field": protected.FieldDocumentTemplate}
	open := func(t *testing.T) (configuredSetupWorld, actor, financialChangeRecord) {
		w := configuredSetup(t)
		grantSetupInspectionForTest(w.f, true)
		p, err := requestChangeAs(w.f, w.requester, w.change)
		w.f.must(err)
		return w, w.f.as(MemberLogin), p
	}
	t.Run("pending verifier", func(t *testing.T) {
		w, reader, p := open(t)
		p.Phase, p.Verification = "verification_unknown", &financialChangeVerification{Actor: "setup-verifier", Evidence: "synthetic-verification", Outcome: protected.OutcomeConfirmed}
		w.f.s.financialIssuance.changes[p.Spec.ID] = p
		row := protectedSetupRow(t, protectedSetupRead(w.f, reader, "protected_changes", w.configuration, template), 0)
		if row["phase"] != "verification_unknown" || row["observation"] != protected.StateRequested || row["pending_verifier"] != "setup-verifier" || row["verifier"] != "" || row["verification_outcome"] != "" || row["verified_at"] != "" {
			t.Fatalf("pinned verifier shown as completed %v", row)
		}
	})
	t.Run("pending approver", func(t *testing.T) {
		w, reader, p := open(t)
		p, err := verifyChangeAs(w.f, w.verifier, p)
		w.f.must(err)
		p.Phase, p.Decisions = "decision_unknown", []financialChangeDecision{{Actor: "setup-approver", Approval: protected.Approval{Verdict: protected.VerdictApprove, Digest: p.Module.Verification.Digest}}}
		w.f.s.financialIssuance.changes[p.Spec.ID] = p
		row := protectedSetupRow(t, protectedSetupRead(w.f, reader, "protected_changes", w.configuration, template), 0)
		decisions, _ := row["decisions"].([]any)
		if row["phase"] != "decision_unknown" || row["observation"] != protected.StateConfirmed || row["pending_decision_actor"] != "setup-approver" || len(decisions) != 0 || row["verifier"] != "setup-verifier" {
			t.Fatalf("pinned approver shown as completed %v", row)
		}
	})
	t.Run("absent request", func(t *testing.T) {
		w, reader, p := open(t)
		f, a := w.f, w.f.s.financialIssuance
		p, err := verifyChangeAs(f, w.verifier, p)
		f.must(err)
		p, err = decideChangeAs(f, w.approver, p)
		f.must(err)
		absent := copyChange(p)
		absent.Spec.ID, absent.Spec.After.ID, absent.Spec.Effective = "absent-change", "synthetic-template-next", p.Spec.Effective.Add(time.Hour)
		absent.Before, absent.Module, absent.Verification, absent.Decisions, absent.Phase = p.Spec.After, protected.View{}, nil, nil, "request_unknown"
		absent.Digest, err = changeDigest(absent)
		f.must(err)
		key := issuanceChangeScope(p.Channel.Subject.Subject, p.Spec.Field)
		a.changes[absent.Spec.ID] = absent
		a.changeOrders[key] = append(a.changeOrders[key], absent.Spec.ID)
		got := protectedSetupRead(f, reader, "protected_changes", w.configuration, template)
		row := protectedSetupRow(t, got, 1)
		if field(got, "count") != "2" || row["change"] != "absent-change" || row["phase"] != "request_unknown" || row["state"] != "" || row["observation"] != "absent" || row["requester"] != absent.Requester || row["requested_at"] != "" || row["approvals_required"] != "" || row["effective"] != setupInstant(absent.Spec.Effective) {
			t.Fatalf("absent row %v", row)
		}
		a.changeOrders[key] = []string{absent.Spec.ID, p.Spec.ID}
		f.code(t, "absent request before another entry", protectedSetupRead(f, reader, "protected_changes", w.configuration, template), 409, string(commands.EvidenceStale))
		a.changeOrders[key] = []string{p.Spec.ID, absent.Spec.ID}
		elsewhere := protected.Change{ID: "absent-change", Subject: protectedSetupSubject(w.configuration, "entity"), Field: protected.FieldTaxIdentifier, After: absent.Spec.After, Provenance: protected.ProvenanceStaff, Instruction: protected.Instruction{Kind: protected.ChannelInPerson, Evidence: "module-only-evidence"}, Effective: f.s.clock().Add(time.Hour)}
		if _, err := a.fields.Request(SampleWorkspace, elsewhere, protected.Actor{ID: "foreign-requester", Kind: protected.KindHuman}); err != nil {
			t.Fatal(err)
		}
		f.code(t, "module holds the absent identifier elsewhere", protectedSetupRead(f, reader, "protected_changes", w.configuration, template), 409, string(commands.EvidenceStale))
	})
	t.Run("unowned entry before owned", func(t *testing.T) {
		w := configuredSetup(t)
		f, a, c := w.f, w.f.s.financialIssuance, w.configuration
		grantSetupInspectionForTest(f, true)
		reader := f.as(MemberLogin)
		channel := w.channels["configuration_kind"]
		foreign := protected.Change{ID: "module-only-first", Subject: channel.Subject.Subject, Field: protected.FieldDocumentTemplate, After: protected.Ref{ID: "module-only-reference", Fingerprint: w.change.After.Fingerprint}, Provenance: protected.ProvenanceStaff, Instruction: protected.Instruction{Kind: protected.ChannelInPerson, Evidence: "module-only-evidence"}, Effective: f.s.clock().Add(time.Hour)}
		actor := protected.Actor{ID: "foreign-requester", Kind: protected.KindHuman}
		if _, err := a.fields.Request(SampleWorkspace, foreign, actor); err != nil {
			t.Fatal(err)
		}
		if _, err := a.fields.Withdraw(SampleWorkspace, foreign.ID, actor); err != nil {
			t.Fatal(err)
		}
		if got := protectedSetupRead(f, reader, "protected_changes", c, template); got.status != 200 || field(got, "count") != "0" || field(got, "unowned_module") != "1" {
			t.Fatal(got.raw)
		}
		p := financialChangeRecord{Configuration: &c, Workspace: SampleWorkspace, Requester: "setup-requester", RulesDigest: a.protectedRules.Digest, Spec: w.change, Channel: copyChannelRecord(channel), Phase: "recorded"}
		v, err := a.fields.Request(SampleWorkspace, p.change(), protected.Actor{ID: p.Requester, Kind: protected.KindHuman})
		if err != nil {
			t.Fatal(err)
		}
		p.Module, p.Before = v, v.Before
		p.Digest, err = changeDigest(p)
		f.must(err)
		key := issuanceChangeScope(channel.Subject.Subject, p.Spec.Field)
		a.changes[p.Spec.ID] = p
		a.changeOrders[key] = []string{p.Spec.ID}
		f.code(t, "unowned entry precedes an owned entry", protectedSetupRead(f, reader, "protected_changes", c, template), 409, string(commands.EvidenceStale))
	})
}

func TestFinancialSetupProtectedCeilingAndDelegation(t *testing.T) {
	f, reader, c := confirmedSetupConfiguration(t)
	if got := protectedSetupRead(f, reader, "protected", c, nil); got.status != 200 {
		t.Fatal(got.raw)
	}
	revoked := 0
	for _, grant := range f.s.admin.Grants() {
		if _, ceiling := f.s.protectedAuthority.ceilings[grant.ID]; ceiling {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
			revoked++
		}
	}
	if revoked != 1 {
		t.Fatalf("revoked %d console ceilings", revoked)
	}
	for _, command := range []string{"protected", "protected_channels"} {
		f.code(t, "revoked console ceiling", protectedSetupRead(f, reader, command, c, nil), 403, string(commands.PermissionDenied))
	}
	f.code(t, "revoked console ceiling", protectedSetupRead(f, reader, "protected_changes", c, map[string]string{"field": protected.FieldPaymentAccount}), 403, string(commands.PermissionDenied))

	d := setup(t, SeedSample).ready()
	d.setPassword(MemberLogin, pw)
	p, err := proposeConfigurationAs(d, d.as(AdminLogin), configurationSpec())
	d.must(err)
	c, err = confirmConfigurationAs(d, d.as(Admin3Login), p)
	d.must(err)
	terms, err := protectedAuthorityTerms(SampleWorkspace, protectedauthority.Inspect)
	d.must(err)
	validity := gate.Validity{NotBefore: d.s.clock(), NotAfter: d.s.clock().Add(time.Hour)}
	d.must(d.s.admin.AddGrant(gate.Grant{ID: "delegation-ceiling", ClientID: ConsoleClient, Terms: terms, Validity: validity}))
	d.must(d.s.admin.AddGrant(gate.Grant{ID: "delegator-inspection", SubjectID: SampleAdmin, Terms: terms, Validity: validity}))
	unrelated, err := termAuthorityTerms(SampleWorkspace, termsauthority.RecordTerm)
	d.must(err)
	d.must(d.s.admin.AddGrant(gate.Grant{ID: "member-standing", SubjectID: SampleMember, Terms: unrelated, Validity: validity}))
	d.must(d.s.admin.AddDelegation(gate.Delegation{ID: "protected-inspection-delegation", DelegatorID: SampleAdmin, DelegateID: SampleMember, Terms: terms, Validity: validity}))
	member := d.as(MemberLogin)
	d.must(inputAuthorityAs(d, member, func() error {
		req := gate.Request{Action: gate.ViewCost, Kind: protectedauthority.ResourceKind, Fields: []string{protectedauthority.EvidenceField}, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
		if got := d.s.gate.Decide(d.s.answer, ConsoleClient, req); !got.Allowed || len(got.Delegations) == 0 {
			t.Fatal("fixture did not establish actual delegated inspection")
		}
		return nil
	}))
	for _, command := range []string{"protected", "protected_channels"} {
		d.code(t, "delegated inspection", protectedSetupRead(d, member, command, c, nil), 403, string(commands.PermissionDenied))
	}
	d.code(t, "delegated inspection", protectedSetupRead(d, member, "protected_changes", c, map[string]string{"field": protected.FieldPaymentAccount}), 403, string(commands.PermissionDenied))
}
