package server

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

type publicProtectedCall struct {
	w            *publicProtectedWorld
	who          actor
	kind, family string
	id, claim    string
	fields       map[string]string
	recovered    string
}

func publicProtectedStage(t *testing.T, kind string) publicProtectedCall {
	t.Helper()
	rules := map[string]string{"rules": "public-rules", "approvals": "1", "validity_seconds": "172800", "max_lead_seconds": "604800", "channel_age_seconds": "86400"}
	switch kind {
	case "propose_protected_rules", "confirm_protected_rules":
		f := setup(t, SeedSample).ready()
		f.setPassword(MemberLogin, pw)
		p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
		f.must(err)
		c, err := confirmConfigurationAs(f, f.as(Admin3Login), p)
		f.must(err)
		grantProtectedInspection(f, SampleAdmin)
		grantProtectedInspection(f, SampleAdmin2)
		w := &publicProtectedWorld{f: f, c: c, channels: map[string][2]string{}}
		if kind == "propose_protected_rules" {
			return publicProtectedCall{w: w, who: f.as(AdminLogin), kind: kind, family: "rules", id: "public-rules", fields: rules, recovered: "proposed"}
		}
		proposed := protectedWrite(f, f.as(AdminLogin), "propose_protected_rules", c, rules)
		f.ok(t, "propose rules", proposed, "proposed", "")
		return publicProtectedCall{w: w, who: f.as(Admin2Login), kind: kind, family: "rules", id: "public-rules", fields: map[string]string{"rules": "public-rules", "rules_digest": field(proposed, "rules_digest")}, recovered: "confirmed"}
	case "register_channel", "admit_channel":
		w := publicProtectedRules(t)
		f := w.f
		admitter := activateProtectedMember(f, "public-admitter")
		grantChannelCapability(f, protectedauthority.RegisterChannel, SampleMember)
		grantChannelCapability(f, protectedauthority.AdmitChannel, "public-admitter")
		w.registrar, w.admitter = f.as(MemberLogin), f.as(admitter)
		id := "staged-channel"
		claim := w.claim(t, w.registrar, "channel_registration", id, "")
		register := map[string]string{"channel": id, "subject_kind": "document_kind", "kind": protected.ChannelPortal, "reference": "synthetic_channel_document_kind_a", "attestation": claim}
		if kind == "register_channel" {
			return publicProtectedCall{w: w, who: w.registrar, kind: kind, family: "channel", id: id, claim: claim, fields: register, recovered: "registered"}
		}
		registered := protectedWrite(f, w.registrar, "register_channel", w.c, register)
		f.ok(t, "register", registered, "registered", "")
		digest := field(registered, "channel_digest")
		claim = w.claim(t, w.admitter, "channel_admission", id, digest)
		return publicProtectedCall{w: w, who: w.admitter, kind: kind, family: "channel", id: id, claim: claim, fields: map[string]string{"channel": id, "channel_digest": digest, "attestation": claim}, recovered: "admitted"}
	}
	w := publicProtectedMature(t)
	f, id := w.f, "staged-change"
	request := w.request(t, id, protected.FieldDocumentTemplate, "document_kind", "synthetic_template_a", nil)
	if kind == "request_change" {
		return publicProtectedCall{w: w, who: w.requester, kind: kind, family: "change", id: id, claim: request["attestation"], fields: request, recovered: protected.StateRequested}
	}
	requested := protectedWrite(f, w.requester, "request_change", w.c, request)
	f.ok(t, "request", requested, protected.StateRequested, "")
	digest := field(requested, "change_digest")
	claim := w.claim(t, w.verifier, "change_verification", id, digest)
	verify := map[string]string{"change": id, "change_digest": digest, "outcome": protected.OutcomeConfirmed, "attestation": claim}
	if kind == "verify_change" {
		return publicProtectedCall{w: w, who: w.verifier, kind: kind, family: "change", id: id, claim: claim, fields: verify, recovered: protected.StateConfirmed}
	}
	f.ok(t, "verify", protectedWrite(f, w.verifier, "verify_change", w.c, verify), protected.StateConfirmed, "")
	return publicProtectedCall{w: w, who: w.approver, kind: kind, family: "change", id: id, fields: map[string]string{"change": id, "change_digest": digest, "verdict": protected.VerdictApprove}, recovered: protected.StateScheduled}
}

func TestFinancialProtectedWritesFaultOutcomes(t *testing.T) {
	cases := []struct {
		kind, fault string
		clean       bool
	}{
		{"propose_protected_rules", "issuance-protected-rules-before-propose", true}, {"propose_protected_rules", "issuance-protected-rules-after-propose", false}, {"propose_protected_rules", "issuance-protected-rules-after-propose-publication", false},
		{"confirm_protected_rules", "issuance-protected-rules-before-confirm", true}, {"confirm_protected_rules", "issuance-protected-rules-after-confirm", false}, {"confirm_protected_rules", "issuance-protected-rules-after-confirm-publication", false},
		{"register_channel", "issuance-channel-lookup", true}, {"register_channel", "issuance-channel-before-register", true}, {"register_channel", "issuance-channel-after-register", false}, {"register_channel", "issuance-channel-after-registration-publication", false},
		{"admit_channel", "issuance-channel-lookup", true}, {"admit_channel", "issuance-channel-before-admit", true}, {"admit_channel", "issuance-channel-after-admit", false}, {"admit_channel", "issuance-channel-after-admission-publication", false},
		{"request_change", "issuance-change-lookup", true}, {"request_change", "issuance-change-before-request", true}, {"request_change", "issuance-change-after-request", false}, {"request_change", "issuance-change-after-request-publication", false},
		{"verify_change", "issuance-change-lookup", true}, {"verify_change", "issuance-change-before-verify", true}, {"verify_change", "issuance-change-after-verify", false}, {"verify_change", "issuance-change-after-verification-publication", false},
		{"decide_change", "issuance-change-before-decide", true}, {"decide_change", "issuance-change-after-decide", false}, {"decide_change", "issuance-change-after-decision-publication", false},
	}
	for _, tc := range cases {
		t.Run(tc.kind+"/"+tc.fault, func(t *testing.T) {
			call := publicProtectedStage(t, tc.kind)
			f, a := call.w.f, call.w.f.s.financialIssuance
			observe := func() protectedWriteObservation {
				var out protectedWriteObservation
				f.must(inputAuthorityAs(f, call.who, func() error {
					out = a.observeProtectedWrite(SampleWorkspace, call.family, call.id, call.claim)
					return nil
				}))
				return out
			}
			before, revision := observe(), f.s.Revision()
			f.s.fault = func(at string) bool { return at == tc.fault }
			body := protectedWriteBody(f, "faulted-write", call.kind, call.w.c, call.fields)
			got := f.call("issuance_setup."+call.kind, body, call.who.cookie, call.who.csrf)
			if tc.clean {
				f.code(t, "clean refusal", got, 503, string(commands.DependencyUnavailable))
				if got.body["operation_id"] != nil || f.s.Revision() != revision || !reflect.DeepEqual(before, observe()) {
					t.Fatal("clean refusal changed state or carried an operation", got.raw)
				}
			} else {
				if got.status != 500 || got.body["operation_id"] == nil || f.s.Revision() == revision || reflect.DeepEqual(before, observe()) {
					t.Fatal("effect not reported as uncertain", got.raw)
				}
				replay := f.call("issuance_setup."+call.kind, body, call.who.cookie, call.who.csrf)
				if replay.status != 500 || replay.body["operation_id"] != got.body["operation_id"] {
					t.Fatal("same key did not replay the stored outcome", replay.raw)
				}
				unchanged, second := observe(), f.s.Revision()
				again := protectedWrite(f, call.who, call.kind, call.w.c, call.fields)
				if strings.HasSuffix(tc.fault, "-publication") || call.family == "change" {
					f.ok(t, "retained effect reconciled", again, call.recovered, "")
				} else {
					f.code(t, "repeated failure at the same point", again, 503, string(commands.DependencyUnavailable))
					if again.body["operation_id"] != nil || f.s.Revision() != second || !reflect.DeepEqual(unchanged, observe()) {
						t.Fatal("repeated failure was not a clean refusal", again.raw)
					}
				}
			}
			f.s.fault = nil
			f.ok(t, "new identical request", protectedWrite(f, call.who, call.kind, call.w.c, call.fields), call.recovered, "")
			if call.claim != "" {
				if p := a.attestations[call.claim]; p.Consumed == nil || p.Consumed.Target != call.id {
					t.Fatal("recovered operation lost its evidence")
				}
			}
		})
	}
}

func TestFinancialProtectedWritesAuthorityAndSeparation(t *testing.T) {
	w := publicProtectedMature(t)
	f, a := w.f, w.f.s.financialIssuance
	root := f.as(AdminLogin)
	request := w.request(t, "authority-change", protected.FieldDocumentTemplate, "document_kind", "synthetic_template_a", nil)
	for name, who := range map[string]actor{"another capability": w.verifier, "administrator": root, "custodian": w.admitter} {
		f.code(t, "request by "+name, protectedWrite(f, who, "request_change", w.c, request), 403, string(commands.PermissionDenied))
	}
	register := map[string]string{"channel": "authority-channel", "subject_kind": "customer", "kind": protected.ChannelEmail, "reference": "synthetic_channel_customer_b", "attestation": w.claim(t, w.registrar, "channel_registration", "authority-channel", "")}
	for name, who := range map[string]actor{"another capability": w.requester, "administrator": root} {
		f.code(t, "register by "+name, protectedWrite(f, who, "register_channel", w.c, register), 403, string(commands.PermissionDenied))
	}
	rules := map[string]string{"rules": "other-rules", "approvals": "1", "validity_seconds": "172800", "max_lead_seconds": "604800", "channel_age_seconds": "86400"}
	for name, who := range map[string]actor{"step capability": w.registrar, "governor without inspection": f.as(Admin3Login)} {
		f.code(t, "rules by "+name, protectedWrite(f, who, "propose_protected_rules", w.c, rules), 403, string(commands.PermissionDenied))
	}
	terms, err := protectedAuthorityTerms(SampleWorkspace, protectedauthority.ApproveChange)
	f.must(err)
	if f.s.admin.AddDelegation(gate.Delegation{ID: "delegated-approval", DelegatorID: SampleAdmin, DelegateID: SampleMember, Terms: terms, Validity: gate.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}}) == nil {
		t.Fatal("approval capability could be delegated")
	}

	grantChannelCapabilityAt(f, "custodian-", protectedauthority.RequestChange, SampleMember)
	grantChannelCapabilityAt(f, "self-admit-", protectedauthority.AdmitChannel, SampleMember)
	custodian := w.request(t, "custodian-change", protected.FieldDocumentTemplate, "document_kind", "synthetic_template_a", map[string]string{"attestation": w.claim(t, w.registrar, "change_instruction", "custodian-change", "")})
	f.code(t, "custodian requests", protectedWrite(f, w.registrar, "request_change", w.c, custodian), 403, string(commands.SelfApprovalDenied))
	pending := protectedWrite(f, w.registrar, "register_channel", w.c, register)
	f.ok(t, "register", pending, "registered", "")
	f.code(t, "registrar admits", protectedWrite(f, w.registrar, "admit_channel", w.c, map[string]string{"channel": "authority-channel", "channel_digest": field(pending, "channel_digest"), "attestation": register["attestation"]}), 409, string(commands.EvidenceStale))

	through := w.request(t, "same-channel-change", protected.FieldDocumentTemplate, "document_kind", "synthetic_template_a", map[string]string{"instruction": "registered_channel", "instruction_channel": w.channels["document_kind"][0]})
	f.code(t, "verification through the instruction channel", protectedWrite(f, w.requester, "request_change", w.c, through), 422, string(commands.InvalidRequest))

	requested := protectedWrite(f, w.requester, "request_change", w.c, request)
	f.ok(t, "request", requested, protected.StateRequested, "")
	digest := field(requested, "change_digest")
	grantChannelCapabilityAt(f, "self-verify-", protectedauthority.VerifyChange, "public-requester")
	grantChannelCapabilityAt(f, "self-approve-", protectedauthority.ApproveChange, "public-requester")
	grantChannelCapabilityAt(f, "verifier-approve-", protectedauthority.ApproveChange, "public-verifier")
	f.code(t, "requester verifies", protectedWrite(f, w.requester, "verify_change", w.c, map[string]string{"change": "authority-change", "change_digest": digest, "outcome": protected.OutcomeConfirmed, "attestation": request["attestation"]}), 409, string(commands.EvidenceStale))
	f.ok(t, "verify", protectedWrite(f, w.verifier, "verify_change", w.c, map[string]string{"change": "authority-change", "change_digest": digest, "outcome": protected.OutcomeConfirmed, "attestation": w.claim(t, w.verifier, "change_verification", "authority-change", digest)}), protected.StateConfirmed, "")
	decide := map[string]string{"change": "authority-change", "change_digest": digest, "verdict": protected.VerdictApprove}
	for name, who := range map[string]actor{"requester": w.requester, "verifier": w.verifier} {
		f.code(t, name+" decides", protectedWrite(f, who, "decide_change", w.c, decide), 403, string(commands.SelfApprovalDenied))
	}
	if a.changes["authority-change"].Module.State != protected.StateConfirmed || len(a.changes) != 1 {
		t.Fatal("refused separation case changed retained changes")
	}

	f.s.enterHold("Protected writes held")
	f.code(t, "held", protectedWrite(f, w.approver, "decide_change", w.c, decide), 503, string(commands.DependencyUnavailable))
	f.s.hold = nil
	f.s.endProbe(w.approver.verifier(f))
	f.code(t, "ended session", protectedWrite(f, w.approver, "decide_change", w.c, decide), 401, string(commands.Unauthenticated))
	f.s.cfg.Seed = SeedNone
	f.code(t, "non sample", protectedWrite(f, root, "confirm_protected_rules", w.c, map[string]string{"rules": "public-rules", "rules_digest": strings.Repeat("a", 64)}), 501, string(commands.UnsupportedCapability))
}

func TestFinancialProtectedWritesClosedValuesBindingAndEvidence(t *testing.T) {
	w := publicProtectedMature(t)
	f, a := w.f, w.f.s.financialIssuance
	changes, channels := len(a.changes), len(a.channels)
	claim := w.claim(t, w.registrar, "channel_registration", "closed-channel", "")
	register := func(change map[string]string) map[string]string {
		fields := map[string]string{"channel": "closed-channel", "subject_kind": "customer", "kind": protected.ChannelEmail, "reference": "synthetic_channel_customer_b", "attestation": claim}
		for k, v := range change {
			fields[k] = v
		}
		return fields
	}
	for name, change := range map[string]map[string]string{
		"subject kind": {"subject_kind": "tenant"}, "channel kind": {"kind": "fax"}, "reference of another use": {"reference": "synthetic_channel_entity_a"},
		"unknown reference": {"reference": "synthetic_channel_customer_z"}, "field reference as channel": {"reference": "synthetic_account_a"},
	} {
		f.code(t, name, protectedWrite(f, w.registrar, "register_channel", w.c, register(change)), 422, string(commands.InvalidRequest))
	}
	instruction := w.claim(t, w.requester, "change_instruction", "closed-change", "")
	request := func(change map[string]string) map[string]string {
		change["attestation"] = instruction
		return w.request(t, "closed-change", protected.FieldDocumentRecipient, "customer", "synthetic_recipient_a", change)
	}
	for name, change := range map[string]map[string]string{
		"field": {"field": protected.FieldTaxIdentifier}, "reference of another field": {"reference": "synthetic_account_a"}, "instruction form": {"instruction": "email"},
		"instruction channel with another form": {"instruction_channel": w.channels["customer"][0]}, "registered channel without a channel": {"instruction": "registered_channel"},
		"effective format": {"effective": "tomorrow"}, "provenance": {"provenance": "rumour"},
	} {
		f.code(t, name, protectedWrite(f, w.requester, "request_change", w.c, request(change)), 422, string(commands.InvalidRequest))
	}
	for name, change := range map[string]map[string]string{"approvals": {"approvals": "4"}, "channel age": {"channel_age_seconds": "3600"}, "validity": {"validity_seconds": "60"}, "lead": {"max_lead_seconds": "0"}, "leading zero": {"approvals": "01"}} {
		rules := map[string]string{"rules": "other-rules", "approvals": "1", "validity_seconds": "172800", "max_lead_seconds": "604800", "channel_age_seconds": "86400"}
		for k, v := range change {
			rules[k] = v
		}
		f.code(t, "rules "+name, protectedWrite(f, f.as(AdminLogin), "propose_protected_rules", w.c, rules), 422, string(commands.InvalidRequest))
	}

	private := financialChannelSpec{ID: "private-channel", Kind: protected.ChannelPortal, Ref: protected.Ref{ID: "private-reference", Fingerprint: strings.Repeat("e", 64)}, Evidence: "private-evidence", Reason: "Private fixture channel", Selector: financialChannelSelector{Kind: "customer", Contract: w.c.Spec.Contract, Revision: w.c.Spec.ContractRevision}}
	fixture, err := registerChannelAs(f, w.registrar, private)
	f.must(err)
	channels = len(a.channels)
	revision := f.s.Revision()
	for name, change := range map[string]map[string]string{
		"missing verification channel": {"channel": "absent-channel"}, "private verification channel": {"channel": fixture.Spec.ID, "channel_digest": fixture.Digest},
		"missing instruction channel": {"instruction": "registered_channel", "instruction_channel": "absent-channel"}, "channel of another subject": {"channel": w.channels["entity"][0], "channel_digest": w.channels["entity"][1]},
	} {
		f.code(t, name, protectedWrite(f, w.requester, "request_change", w.c, request(change)), 409, string(commands.EvidenceStale))
	}
	f.code(t, "missing change", protectedWrite(f, w.approver, "decide_change", w.c, map[string]string{"change": "absent-change", "change_digest": strings.Repeat("a", 64), "verdict": protected.VerdictApprove}), 409, string(commands.EvidenceStale))
	f.code(t, "missing channel", protectedWrite(f, w.admitter, "admit_channel", w.c, map[string]string{"channel": "absent-channel", "channel_digest": strings.Repeat("a", 64), "attestation": claim}), 409, string(commands.EvidenceStale))
	if f.s.Revision() != revision {
		t.Fatal("refused binding advanced the revision")
	}

	other := w.claim(t, w.registrar, "channel_registration", "another-channel", "")
	withdrawn := w.claim(t, w.registrar, "channel_registration", "closed-channel", "")
	f.ok(t, "withdraw", attestRun(f, w.registrar, "withdraw_channel_registration", w.c, map[string]string{"attestation": withdrawn}), "withdrawn", "")
	foreign := w.claim(t, w.requester, "change_instruction", "closed-channel", "")
	for name, id := range map[string]string{"missing": "absent-claim", "another target": other, "withdrawn": withdrawn, "another purpose and actor": foreign} {
		f.code(t, name+" attestation", protectedWrite(f, w.registrar, "register_channel", w.c, register(map[string]string{"attestation": id})), 409, string(commands.EvidenceStale))
	}
	if len(a.changes) != changes || len(a.channels) != channels {
		t.Fatal("refused requests retained records")
	}

	registered := protectedWrite(f, w.registrar, "register_channel", w.c, register(nil))
	f.ok(t, "register", registered, "registered", "")
	f.ok(t, "identical repeat", protectedWrite(f, w.registrar, "register_channel", w.c, register(nil)), "registered", "")
	digest := field(registered, "channel_digest")
	admission := w.claim(t, w.admitter, "channel_admission", "closed-channel", digest)
	used := a.attestations[claim]
	saved := *used.Consumed
	altered := saved
	altered.TargetDigest = strings.Repeat("f", 64)
	used.Consumed = &altered
	a.attestations[claim] = used
	f.code(t, "held attestation altered", protectedWrite(f, w.admitter, "admit_channel", w.c, map[string]string{"channel": "closed-channel", "channel_digest": digest, "attestation": admission}), 409, string(commands.EvidenceStale))
	used.Consumed = &saved
	a.attestations[claim] = used
	f.ok(t, "admit", protectedWrite(f, w.admitter, "admit_channel", w.c, map[string]string{"channel": "closed-channel", "channel_digest": digest, "attestation": admission}), "admitted", "")
	delete(a.attestations, admission)
	young := w.request(t, "young-change", protected.FieldDocumentRecipient, "customer", "synthetic_recipient_a", map[string]string{"channel": "closed-channel", "channel_digest": digest})
	f.code(t, "held attestation removed", protectedWrite(f, w.requester, "request_change", w.c, young), 409, string(commands.EvidenceStale))
}

func TestFinancialProtectedWritesTimingMargin(t *testing.T) {
	w := publicProtectedMature(t)
	f, a := w.f, w.f.s.financialIssuance
	instant := func(d time.Duration) string { return f.s.clock().Add(d).UTC().Format(time.RFC3339Nano) }
	inside := w.request(t, "margin-change", protected.FieldDocumentTemplate, "document_kind", "synthetic_template_a", map[string]string{"effective": instant(protectedWriteMargin - time.Nanosecond)})
	f.code(t, "effective inside the margin", protectedWrite(f, w.requester, "request_change", w.c, inside), 422, string(commands.PrerequisiteFailed))
	if len(a.changes) != 0 || a.attestations[inside["attestation"]].Consumed != nil {
		t.Fatal("margin refusal reserved the change or consumed its evidence")
	}
	inside["effective"] = instant(protectedWriteMargin)
	requested := protectedWrite(f, w.requester, "request_change", w.c, inside)
	f.ok(t, "effective at the margin", requested, protected.StateRequested, "")
	digest := field(requested, "change_digest")
	claim := w.claim(t, w.verifier, "change_verification", "margin-change", digest)
	verify := map[string]string{"change": "margin-change", "change_digest": digest, "outcome": protected.OutcomeConfirmed, "attestation": claim}
	f.advance(time.Nanosecond)
	f.code(t, "verification deadline inside the margin", protectedWrite(f, w.verifier, "verify_change", w.c, verify), 422, string(commands.PrerequisiteFailed))
	if a.changes["margin-change"].Verification != nil || a.attestations[claim].Consumed != nil {
		t.Fatal("margin refusal pinned the verification")
	}

	late := w.request(t, "late-change", protected.FieldDocumentRecipient, "customer", "synthetic_recipient_a", map[string]string{"effective": instant(5 * time.Minute)})
	requested = protectedWrite(f, w.requester, "request_change", w.c, late)
	f.ok(t, "request", requested, protected.StateRequested, "")
	digest = field(requested, "change_digest")
	claim = w.claim(t, w.verifier, "change_verification", "late-change", digest)
	verify = map[string]string{"change": "late-change", "change_digest": digest, "outcome": protected.OutcomeConfirmed, "attestation": claim}
	f.s.fault = func(at string) bool { return at == "issuance-change-after-verify" }
	if got := protectedWrite(f, w.verifier, "verify_change", w.c, verify); got.status != 500 || got.body["operation_id"] == nil {
		t.Fatal(got.raw)
	}
	f.s.fault = nil
	f.advance(6 * time.Minute)
	recovered := protectedWrite(f, w.verifier, "verify_change", w.c, verify)
	if recovered.status != 200 || a.changes["late-change"].Phase != "recorded" || a.changes["late-change"].Module.Verification == nil {
		t.Fatal("recovery after the deadline was refused", recovered.raw)
	}
	decide := map[string]string{"change": "late-change", "change_digest": digest, "verdict": protected.VerdictApprove}
	f.code(t, "decision after lapse", protectedWrite(f, w.approver, "decide_change", w.c, decide), 422, string(commands.PrerequisiteFailed))
}
