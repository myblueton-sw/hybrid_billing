package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

type publicProtectedWorld struct {
	f                                                  world
	c                                                  financialIssuanceConfiguration
	registrar, admitter, requester, verifier, approver actor
	channels                                           map[string][2]string
	claims                                             int
}

func protectedWriteBody(f world, key, kind string, c financialIssuanceConfiguration, fields map[string]string) string {
	body := map[string]string{"configuration": c.Spec.ID, "digest": c.Digest}
	for name, value := range fields {
		body[name] = value
	}
	envelope := map[string]any{"request_id": key, "target_id": c.Spec.Contract, "idempotency_key": key, "expected_revision": f.s.Revision(), "body": body}
	if step, _ := protectedWriteCommand(kind); step.Rationale {
		envelope["rationale"] = "Synthetic protected setup"
	}
	raw, _ := json.Marshal(envelope)
	return string(raw)
}

func protectedWrite(f world, who actor, kind string, c financialIssuanceConfiguration, fields map[string]string) reply {
	f.t.Helper()
	return f.call("issuance_setup."+kind, protectedWriteBody(f, fmt.Sprintf("protected-write-%d", keys.Add(1)), kind, c, fields), who.cookie, who.csrf)
}

func grantProtectedInspection(f world, recipient string) {
	governors := []string{}
	for _, u := range []struct{ id, login string }{{SampleAdmin, AdminLogin}, {SampleAdmin2, Admin2Login}, {SampleAdmin3, Admin3Login}} {
		if u.id != recipient {
			governors = append(governors, u.login)
		}
	}
	root, second := f.as(governors[0]), f.as(governors[1])
	spec := protectedPermissionSpec(f, protectedauthority.Inspect)
	p, err := protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "inspect-ceiling-" + recipient, Spec: spec, Rationale: "Explicit evidence ceiling"})
	f.must(err)
	c, err := protectedPermissionDecision(f, second, p)
	f.must(err)
	spec.Kind, spec.Recipient, spec.CeilingID = inputauthority.Human, recipient, c.Decision.GrantID
	p, err = protectedPermissionProposal(f, root, inputauthority.Proposal{ID: "inspect-human-" + recipient, Spec: spec, Rationale: "Explicit evidence recipient"})
	f.must(err)
	_, err = protectedPermissionDecision(f, second, p)
	f.must(err)
}

func (w *publicProtectedWorld) claim(t *testing.T, who actor, purpose, target, digest string) string {
	t.Helper()
	w.claims++
	id := fmt.Sprintf("claim-%s-%02d", purpose, w.claims)
	fields := map[string]string{"attestation": id, "target": target, "statement": "Synthetic statement of the act for " + purpose + "."}
	if digest != "" {
		fields["target_digest"] = digest
	}
	w.f.ok(t, "attest "+purpose, attestRun(w.f, who, "attest_"+purpose, w.c, fields), "recorded", "")
	return id
}

func publicProtectedRules(t *testing.T) *publicProtectedWorld {
	t.Helper()
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
	f.must(err)
	c, err := confirmConfigurationAs(f, f.as(Admin3Login), p)
	f.must(err)
	grantProtectedInspection(f, SampleAdmin)
	grantProtectedInspection(f, SampleAdmin2)
	w := &publicProtectedWorld{f: f, c: c, channels: map[string][2]string{}}
	proposed := protectedWrite(f, f.as(AdminLogin), "propose_protected_rules", c, map[string]string{"rules": "public-rules", "approvals": "1", "validity_seconds": "172800", "max_lead_seconds": "604800", "channel_age_seconds": "86400"})
	f.ok(t, "propose rules", proposed, "proposed", "")
	confirmed := protectedWrite(f, f.as(Admin2Login), "confirm_protected_rules", c, map[string]string{"rules": "public-rules", "rules_digest": field(proposed, "rules_digest")})
	f.ok(t, "confirm rules", confirmed, "confirmed", "")
	if field(confirmed, "schema") != "hb_issuance_protected_rules_receipt_v1" || field(confirmed, "current_eligibility") != "unchecked" || field(confirmed, "rules_digest") != field(proposed, "rules_digest") {
		t.Fatal(confirmed.raw)
	}
	return w
}

func publicProtectedChannels(t *testing.T) *publicProtectedWorld {
	t.Helper()
	w := publicProtectedRules(t)
	f := w.f
	admitter := activateProtectedMember(f, "public-admitter")
	grantChannelCapability(f, protectedauthority.RegisterChannel, SampleMember)
	grantChannelCapability(f, protectedauthority.AdmitChannel, "public-admitter")
	w.registrar, w.admitter = f.as(MemberLogin), f.as(admitter)
	for _, subject := range []string{"customer", "entity", "document_kind"} {
		id := "public-channel-" + subject
		registered := protectedWrite(f, w.registrar, "register_channel", w.c, map[string]string{"channel": id, "subject_kind": subject, "kind": protected.ChannelPortal, "reference": "synthetic_channel_" + subject + "_a", "attestation": w.claim(t, w.registrar, "channel_registration", id, "")})
		f.ok(t, "register "+subject, registered, "registered", "")
		digest := field(registered, "channel_digest")
		admitted := protectedWrite(f, w.admitter, "admit_channel", w.c, map[string]string{"channel": id, "channel_digest": digest, "attestation": w.claim(t, w.admitter, "channel_admission", id, digest)})
		f.ok(t, "admit "+subject, admitted, "admitted", "")
		if field(admitted, "channel_digest") != digest || field(admitted, "schema") != "hb_issuance_protected_channel_receipt_v1" {
			t.Fatal(admitted.raw)
		}
		w.channels[subject] = [2]string{id, digest}
	}
	return w
}

func publicProtectedMature(t *testing.T) *publicProtectedWorld {
	t.Helper()
	w := publicProtectedChannels(t)
	f := w.f
	f.advance(24 * time.Hour)
	renewInvoiceAuthorities(t, f)
	logins := map[string]string{}
	for _, entry := range []struct{ id, capability string }{{"public-requester", protectedauthority.RequestChange}, {"public-verifier", protectedauthority.VerifyChange}, {"public-approver", protectedauthority.ApproveChange}} {
		logins[entry.id] = activateProtectedMember(f, entry.id)
		grantChannelCapabilityAt(f, entry.id+"-", entry.capability, entry.id)
	}
	w.registrar, w.admitter = f.as(MemberLogin), f.as("public-admitter@example.invalid")
	w.requester, w.verifier, w.approver = f.as(logins["public-requester"]), f.as(logins["public-verifier"]), f.as(logins["public-approver"])
	return w
}

var publicProtectedFields = []struct{ field, subject, reference string }{
	{protected.FieldDocumentRecipient, "customer", "synthetic_recipient_a"},
	{protected.FieldDocumentTemplate, "document_kind", "synthetic_template_a"},
	{protected.FieldPaymentAccount, "entity", "synthetic_account_a"},
}

func (w *publicProtectedWorld) request(t *testing.T, id, fieldName, subject, reference string, extra map[string]string) map[string]string {
	t.Helper()
	fields := map[string]string{"change": id, "field": fieldName, "reference": reference, "channel": w.channels[subject][0], "channel_digest": w.channels[subject][1], "provenance": protected.ProvenanceStaff, "instruction": "in_person", "effective": w.f.s.clock().Add(10 * time.Minute).UTC().Format(time.RFC3339Nano)}
	for name, value := range extra {
		fields[name] = value
	}
	if fields["attestation"] == "" {
		fields["attestation"] = w.claim(t, w.requester, "change_instruction", id, "")
	}
	return fields
}

func TestFinancialProtectedWritesPublicSequence(t *testing.T) {
	w := publicProtectedMature(t)
	f, a := w.f, w.f.s.financialIssuance
	changes := map[string]string{}
	for _, entry := range publicProtectedFields {
		id := "public-change-" + entry.subject
		requested := protectedWrite(f, w.requester, "request_change", w.c, w.request(t, id, entry.field, entry.subject, entry.reference, nil))
		f.ok(t, "request "+entry.field, requested, protected.StateRequested, "")
		digest := field(requested, "change_digest")
		if field(requested, "phase") != "recorded" || field(requested, "schema") != "hb_issuance_protected_change_receipt_v1" || field(requested, "current_eligibility") != "unchecked" {
			t.Fatal(requested.raw)
		}
		verified := protectedWrite(f, w.verifier, "verify_change", w.c, map[string]string{"change": id, "change_digest": digest, "outcome": protected.OutcomeConfirmed, "attestation": w.claim(t, w.verifier, "change_verification", id, digest)})
		f.ok(t, "verify "+entry.field, verified, protected.StateConfirmed, "")
		decided := protectedWrite(f, w.approver, "decide_change", w.c, map[string]string{"change": id, "change_digest": digest, "verdict": protected.VerdictApprove})
		f.ok(t, "decide "+entry.field, decided, protected.StateScheduled, "")
		for _, got := range []reply{requested, verified, decided} {
			for _, hidden := range []string{"synthetic-reference:", a.changes[id].Spec.After.Fingerprint, "Synthetic statement"} {
				if strings.Contains(got.raw, hidden) {
					t.Fatalf("receipt disclosed %q", hidden)
				}
			}
		}
		changes[entry.field] = id
	}
	if len(a.channels) != 3 || len(a.changes) != 3 || len(a.attestations) != 12 || a.protectedRules.State != "confirmed" {
		t.Fatal("retained records differ from the public sequence")
	}
	for id, p := range a.attestations {
		if p.Consumed == nil || !p.WithdrawnAt.IsZero() {
			t.Fatalf("attestation %s not consumed", id)
		}
	}
	for id, p := range a.channels {
		if p.Spec.Attestation == "" || p.AdmissionAttestation == "" || p.Spec.Evidence != a.attestations[p.Spec.Evidence].ID || p.State != "admitted" {
			t.Fatalf("channel %s lacks retained evidence", id)
		}
	}
	for id, p := range a.changes {
		if p.Spec.Attestation == "" || p.Verification == nil || p.Verification.Attestation == "" || p.Module.State != protected.StateScheduled {
			t.Fatalf("change %s lacks retained evidence", id)
		}
	}

	grantProtectedInspection(f, SampleAdmin3)
	reader := f.as(Admin3Login)
	summary := protectedSetupRead(f, reader, "protected", w.c, nil)
	if summary.status != 200 || field(summary, "rules_state") != "confirmed" || field(summary, "rules_observation") != "active" || field(summary, "channels") != "3" {
		t.Fatal(summary.raw)
	}
	for prefix, fieldName := range map[string]string{"recipient": protected.FieldDocumentRecipient, "template": protected.FieldDocumentTemplate, "account": protected.FieldPaymentAccount} {
		if field(summary, prefix+"_changes") != "1" || field(summary, prefix+"_latest_scheduled") != changes[fieldName] || field(summary, prefix+"_unowned_module") != "0" {
			t.Fatalf("%s summary %s", prefix, summary.raw)
		}
	}
	first := protectedSetupRead(f, reader, "attestations", w.c, nil)
	if first.status != 200 || field(first, "count") != "12" {
		t.Fatal(first.raw)
	}
	for i := 0; i < 10; i++ {
		if row := protectedSetupRow(t, first, i); row["state"] != "consumed" {
			t.Fatalf("attestation row %v", row)
		}
	}
	f.advance(11 * time.Minute)
	f.must(inputAuthorityAs(f, w.requester, func() error {
		resolved, err := a.configuredReferences(w.c, f.s.clock())
		if err != nil {
			return err
		}
		if len(resolved) != 3 {
			t.Fatalf("resolved %d references", len(resolved))
		}
		for _, r := range resolved {
			want, _ := setupReference(map[string]string{protected.FieldDocumentRecipient: "synthetic_recipient_a", protected.FieldDocumentTemplate: "synthetic_template_a", protected.FieldPaymentAccount: "synthetic_account_a"}[r.Field], r.Field)
			if r.Ref != want || r.Change != changes[r.Field] {
				t.Fatalf("resolved reference %+v", r)
			}
		}
		return nil
	}))
}

func TestFinancialProtectedWritesEmailOnlyChangeNeverAppliesAlone(t *testing.T) {
	w := publicProtectedMature(t)
	f, a := w.f, w.f.s.financialIssuance
	fields := w.request(t, "email-only-change", protected.FieldPaymentAccount, "entity", "synthetic_account_a", map[string]string{"instruction": "unregistered_sender", "provenance": protected.ProvenanceDocument})
	requested := protectedWrite(f, w.requester, "request_change", w.c, fields)
	f.ok(t, "email-only request", requested, protected.StateRequested, "")
	p := a.changes["email-only-change"]
	sender, _ := setupReference("synthetic_unregistered_sender", "unregistered_sender")
	if p.Spec.Instruction.Kind != protected.ChannelEmail || p.Spec.Instruction.Sender != sender.Fingerprint || p.Spec.Instruction.Channel != "" {
		t.Fatalf("instruction %+v", p.Spec.Instruction)
	}
	f.code(t, "approval without verification", protectedWrite(f, w.approver, "decide_change", w.c, map[string]string{"change": p.Spec.ID, "change_digest": p.Digest, "verdict": protected.VerdictApprove}), 422, string(commands.PrerequisiteFailed))
	f.advance(11 * time.Minute)
	f.must(inputAuthorityAs(f, w.requester, func() error {
		if _, err := a.fields.Resolve(SampleWorkspace, p.Channel.Subject.Subject, protected.FieldPaymentAccount, f.s.clock()); err == nil {
			t.Fatal("email-only change applied")
		}
		view, err := a.fields.Peek(SampleWorkspace, p.Spec.ID)
		if err != nil || view.State != protected.StateLapsed {
			t.Fatalf("email-only change did not lapse: %v %v", view.State, err)
		}
		return nil
	}))
	generated := w.request(t, "assistant-change", protected.FieldDocumentRecipient, "customer", "synthetic_recipient_a", map[string]string{"provenance": protected.ProvenanceAssistant})
	f.code(t, "assistant-generated provenance", protectedWrite(f, w.requester, "request_change", w.c, generated), 422, string(commands.InvalidRequest))
	if _, exists := a.changes["assistant-change"]; exists {
		t.Fatal("assistant-generated request retained")
	}
}
