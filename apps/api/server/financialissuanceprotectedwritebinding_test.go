package server

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func (w *publicProtectedWorld) verify(t *testing.T, id, digest string) map[string]string {
	t.Helper()
	return map[string]string{"change": id, "change_digest": digest, "outcome": protected.OutcomeConfirmed, "attestation": w.claim(t, w.verifier, "change_verification", id, digest)}
}

func TestFinancialProtectedWritesBindTargetsToTheNamedConfiguration(t *testing.T) {
	w := publicProtectedMature(t)
	f, a := w.f, w.f.s.financialIssuance
	base, err := f.s.orgs.ContractRevision(SampleWorkspace, w.c.Spec.Contract, w.c.Spec.ContractRevision)
	f.must(err)
	base.Revision++
	f.must(f.s.orgs.AppendContractRevision(base))
	next := configurationSpec()
	next.ID, next.ContractRevision = "second-configuration", base.Revision
	proposed, err := proposeConfigurationAs(f, f.as(AdminLogin), next)
	f.must(err)
	other, err := confirmConfigurationAs(f, f.as(Admin3Login), proposed)
	f.must(err)
	if protectedSetupChannelKind(other, a.channels[w.channels["document_kind"][0]]) != "" || protectedSetupChannelKind(other, a.channels[w.channels["entity"][0]]) != "entity" {
		t.Fatal("fixture configurations do not separate document-kind channels and share the entity channel")
	}

	pending := map[string]string{"channel": "pending-channel", "subject_kind": "document_kind", "kind": protected.ChannelEmail, "reference": "synthetic_channel_document_kind_b", "attestation": w.claim(t, w.registrar, "channel_registration", "pending-channel", "")}
	registered := protectedWrite(f, w.registrar, "register_channel", w.c, pending)
	f.ok(t, "register", registered, "registered", "")
	admission := map[string]string{"channel": "pending-channel", "channel_digest": field(registered, "channel_digest"), "attestation": w.claim(t, w.admitter, "channel_admission", "pending-channel", field(registered, "channel_digest"))}
	requested := protectedWrite(f, w.requester, "request_change", w.c, w.request(t, "bound-change", protected.FieldPaymentAccount, "entity", "synthetic_account_a", nil))
	f.ok(t, "request", requested, protected.StateRequested, "")
	digest := field(requested, "change_digest")
	verification := w.verify(t, "bound-change", digest)

	foreign := other
	foreign.Spec.Contract = "sample-contract-02"
	cases := []struct {
		name, kind string
		who        actor
		fields     map[string]string
	}{
		{"admit a channel of another configuration", "admit_channel", w.admitter, admission},
		{"verify a change of another configuration", "verify_change", w.verifier, verification},
		{"decide a change of another configuration", "decide_change", w.approver, map[string]string{"change": "bound-change", "change_digest": digest, "verdict": protected.VerdictApprove}},
		{"request naming a change of another configuration", "request_change", w.requester, w.request(t, "bound-change", protected.FieldPaymentAccount, "entity", "synthetic_account_a", map[string]string{"attestation": a.changes["bound-change"].Spec.Instruction.Evidence})},
		{"register naming a channel that is not a row", "register_channel", w.registrar, map[string]string{"channel": "pending-channel", "subject_kind": "document_kind", "kind": protected.ChannelEmail, "reference": "synthetic_channel_document_kind_b", "attestation": pending["attestation"]}},
	}
	revision := f.s.Revision()
	for _, tc := range cases {
		f.code(t, tc.name, protectedWrite(f, tc.who, tc.kind, other, tc.fields), 409, string(commands.EvidenceStale))
		f.code(t, tc.name+" through another contract", protectedWrite(f, tc.who, tc.kind, foreign, tc.fields), 403, string(commands.PermissionDenied))
	}
	if f.s.Revision() != revision || a.channels["pending-channel"].State != "registered" || a.changes["bound-change"].Verification != nil || a.attestations[admission["attestation"]].Consumed != nil || a.attestations[verification["attestation"]].Consumed != nil {
		t.Fatal("a refused cross-configuration request changed state")
	}
	f.ok(t, "admit under its own configuration", protectedWrite(f, w.admitter, "admit_channel", w.c, admission), "admitted", "")
	f.ok(t, "verify under its own configuration", protectedWrite(f, w.verifier, "verify_change", w.c, verification), protected.StateConfirmed, "")
}

func TestFinancialProtectedWritesAuthorityForEveryCommand(t *testing.T) {
	valid := func(w *publicProtectedWorld) map[string]map[string]string {
		hash := strings.Repeat("a", 64)
		return map[string]map[string]string{
			"propose_protected_rules": {"rules": "authority-rules", "approvals": "1", "validity_seconds": "172800", "max_lead_seconds": "604800", "channel_age_seconds": "86400"},
			"confirm_protected_rules": {"rules": "public-rules", "rules_digest": hash},
			"register_channel":        {"channel": "authority-channel", "subject_kind": "customer", "kind": protected.ChannelEmail, "reference": "synthetic_channel_customer_b", "attestation": "authority-claim"},
			"admit_channel":           {"channel": w.channels["customer"][0], "channel_digest": w.channels["customer"][1], "attestation": "authority-claim"},
			"request_change":          {"change": "authority-change", "field": protected.FieldDocumentRecipient, "reference": "synthetic_recipient_a", "channel": w.channels["customer"][0], "channel_digest": w.channels["customer"][1], "provenance": protected.ProvenanceStaff, "instruction": "in_person", "effective": w.f.s.clock().Add(time.Hour).UTC().Format(time.RFC3339Nano), "attestation": "authority-claim"},
			"verify_change":           {"change": "authority-change", "change_digest": hash, "outcome": protected.OutcomeConfirmed, "attestation": "authority-claim"},
			"decide_change":           {"change": "authority-change", "change_digest": hash, "verdict": protected.VerdictApprove},
		}
	}
	holder := func(w *publicProtectedWorld, kind string) actor {
		return map[string]actor{"propose_protected_rules": w.f.as(AdminLogin), "confirm_protected_rules": w.f.as(Admin2Login), "register_channel": w.registrar, "admit_channel": w.admitter, "request_change": w.requester, "verify_change": w.verifier, "decide_change": w.approver}[kind]
	}
	t.Run("capability", func(t *testing.T) {
		w := publicProtectedMature(t)
		f := w.f
		inspector := activateProtectedMember(f, "inspect-only")
		grantProtectedInspection(f, "inspect-only")
		callers := map[string]actor{"registrar": w.registrar, "admitter": w.admitter, "requester": w.requester, "verifier": w.verifier, "approver": w.approver, "inspection only": f.as(inspector), "administrator without inspection": f.as(Admin3Login)}
		revision := f.s.Revision()
		for _, step := range protectedWriteSteps {
			for name, who := range callers {
				if who == holder(w, step.Kind) {
					continue
				}
				f.code(t, step.Kind+" by "+name, protectedWrite(f, who, step.Kind, w.c, valid(w)[step.Kind]), 403, string(commands.PermissionDenied))
			}
		}
		if f.s.Revision() != revision || len(f.s.financialIssuance.changes) != 0 || len(f.s.financialIssuance.channels) != 3 {
			t.Fatal("a refused caller changed state")
		}
	})
	for _, mode := range []string{"human grant", "console ceiling", "hold"} {
		t.Run(mode, func(t *testing.T) {
			w := publicProtectedMature(t)
			f := w.f
			status, code := 403, string(commands.PermissionDenied)
			if mode == "hold" {
				f.s.enterHold("Protected writes held")
				status, code = 503, string(commands.DependencyUnavailable)
			}
			for _, step := range protectedWriteSteps {
				if step.Capability == "" && mode != "hold" {
					continue
				}
				if mode != "hold" {
					revoked := 0
					for _, grant := range f.s.admin.Grants() {
						_, ceiling := f.s.protectedAuthority.ceilings[grant.ID]
						if grant.Active && grant.Terms.Action == step.Action && ceiling == (mode == "console ceiling") {
							f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
							revoked++
						}
					}
					if revoked == 0 {
						t.Fatalf("%s: nothing revoked", step.Kind)
					}
				}
				f.code(t, step.Kind, protectedWrite(f, holder(w, step.Kind), step.Kind, w.c, valid(w)[step.Kind]), status, code)
			}
		})
	}
}

func TestFinancialProtectedWritesCustodianSeparationAndClosedOutcomes(t *testing.T) {
	w := publicProtectedMature(t)
	f, a := w.f, w.f.s.financialIssuance
	requested := protectedWrite(f, w.requester, "request_change", w.c, w.request(t, "custody-change", protected.FieldDocumentTemplate, "document_kind", "synthetic_template_a", nil))
	f.ok(t, "request", requested, protected.StateRequested, "")
	digest := field(requested, "change_digest")
	grantChannelCapabilityAt(f, "custodian-verify-", protectedauthority.VerifyChange, SampleMember)
	grantChannelCapabilityAt(f, "custodian-approve-", protectedauthority.ApproveChange, SampleMember)
	custodian := map[string]string{"change": "custody-change", "change_digest": digest, "outcome": protected.OutcomeConfirmed, "attestation": w.claim(t, w.registrar, "change_verification", "custody-change", digest)}
	f.code(t, "custodian verifies", protectedWrite(f, w.registrar, "verify_change", w.c, custodian), 403, string(commands.SelfApprovalDenied))
	if a.changes["custody-change"].Verification != nil || a.attestations[custodian["attestation"]].Consumed != nil {
		t.Fatal("custodian verification reserved")
	}
	verify := w.verify(t, "custody-change", digest)
	for name, outcome := range map[string]string{"empty": "", "unknown": "maybe"} {
		invalid := map[string]string{"change": "custody-change", "change_digest": digest, "outcome": outcome, "attestation": verify["attestation"]}
		if outcome == "" {
			delete(invalid, "outcome")
		}
		f.code(t, name+" outcome", protectedWrite(f, w.verifier, "verify_change", w.c, invalid), 422, string(commands.InvalidRequest))
	}
	f.ok(t, "verify", protectedWrite(f, w.verifier, "verify_change", w.c, verify), protected.StateConfirmed, "")
	for name, fields := range map[string]map[string]string{"unknown verdict": {"verdict": "maybe"}, "approval with a reason": {"verdict": protected.VerdictApprove, "reason": "other"}, "rejection without a reason": {"verdict": protected.VerdictReject}, "unknown reason": {"verdict": protected.VerdictReject, "reason": "dislike"}} {
		fields["change"], fields["change_digest"] = "custody-change", digest
		f.code(t, name, protectedWrite(f, w.approver, "decide_change", w.c, fields), 422, string(commands.InvalidRequest))
	}
	decide := map[string]string{"change": "custody-change", "change_digest": digest, "verdict": protected.VerdictApprove}
	f.code(t, "custodian decides", protectedWrite(f, w.registrar, "decide_change", w.c, decide), 403, string(commands.SelfApprovalDenied))
	if len(a.changes["custody-change"].Decisions) != 0 {
		t.Fatal("refused decision retained")
	}
	rejected := protectedWrite(f, w.approver, "decide_change", w.c, map[string]string{"change": "custody-change", "change_digest": digest, "verdict": protected.VerdictReject, "reason": "missing_evidence"})
	f.ok(t, "reject", rejected, protected.StateRejected, "")

	young := map[string]string{"channel": "young-channel", "subject_kind": "customer", "kind": protected.ChannelEmail, "reference": "synthetic_channel_customer_b", "attestation": w.claim(t, w.registrar, "channel_registration", "young-channel", "")}
	registered := protectedWrite(f, w.registrar, "register_channel", w.c, young)
	f.ok(t, "register", registered, "registered", "")
	channel := field(registered, "channel_digest")
	f.ok(t, "admit", protectedWrite(f, w.admitter, "admit_channel", w.c, map[string]string{"channel": "young-channel", "channel_digest": channel, "attestation": w.claim(t, w.admitter, "channel_admission", "young-channel", channel)}), "admitted", "")
	early := w.request(t, "young-change", protected.FieldDocumentRecipient, "customer", "synthetic_recipient_a", map[string]string{"channel": "young-channel", "channel_digest": channel})
	f.code(t, "channel younger than the channel age", protectedWrite(f, w.requester, "request_change", w.c, early), 422, string(commands.PrerequisiteFailed))
	if _, exists := a.changes["young-change"]; exists || a.attestations[early["attestation"]].Consumed != nil {
		t.Fatal("request through a young channel reserved")
	}

	private := financialChannelSpec{ID: "private-compatibility", Kind: protected.ChannelPortal, Ref: protected.Ref{ID: "private-reference", Fingerprint: strings.Repeat("e", 64)}, Evidence: "private-evidence", Reason: "Private fixture channel", Selector: financialChannelSelector{Kind: "customer", Contract: w.c.Spec.Contract, Revision: w.c.Spec.ContractRevision}}
	fixture, err := registerChannelAs(f, w.registrar, private)
	f.must(err)
	raw, err := json.Marshal(fixture)
	f.must(err)
	public, err := json.Marshal(a.channels["young-channel"])
	f.must(err)
	if strings.Contains(string(raw), "Attestation") || !strings.Contains(string(public), `"Attestation"`) || !strings.Contains(string(public), `"AdmissionAttestation"`) {
		t.Fatal("attestation fields are not confined to public records")
	}
}

func TestFinancialProtectedWritesMarginForVerificationAndDecision(t *testing.T) {
	w := publicProtectedMature(t)
	f, a := w.f, w.f.s.financialIssuance
	instant := func(d time.Duration) string { return f.s.clock().Add(d).UTC().Format(time.RFC3339Nano) }
	stage := func(id, fieldName, subject, reference string, lead time.Duration) (string, map[string]string) {
		requested := protectedWrite(f, w.requester, "request_change", w.c, w.request(t, id, fieldName, subject, reference, map[string]string{"effective": instant(lead)}))
		f.ok(t, "request "+id, requested, protected.StateRequested, "")
		return field(requested, "change_digest"), map[string]string{"change": id, "change_digest": field(requested, "change_digest"), "verdict": protected.VerdictApprove}
	}
	edge, _ := stage("verify-at-margin", protected.FieldDocumentRecipient, "customer", "synthetic_recipient_a", protectedWriteMargin)
	f.ok(t, "verification exactly at the margin", protectedWrite(f, w.verifier, "verify_change", w.c, w.verify(t, "verify-at-margin", edge)), protected.StateConfirmed, "")

	at, decideAt := stage("decide-at-margin", protected.FieldDocumentTemplate, "document_kind", "synthetic_template_a", 3*time.Minute)
	inside, decideInside := stage("decide-inside-margin", protected.FieldPaymentAccount, "entity", "synthetic_account_a", 3*time.Minute-time.Nanosecond)
	f.ok(t, "verify", protectedWrite(f, w.verifier, "verify_change", w.c, w.verify(t, "decide-at-margin", at)), protected.StateConfirmed, "")
	f.ok(t, "verify", protectedWrite(f, w.verifier, "verify_change", w.c, w.verify(t, "decide-inside-margin", inside)), protected.StateConfirmed, "")
	f.advance(2 * time.Minute)
	f.code(t, "decision deadline inside the margin", protectedWrite(f, w.approver, "decide_change", w.c, decideInside), 422, string(commands.PrerequisiteFailed))
	if len(a.changes["decide-inside-margin"].Decisions) != 0 || a.changes["decide-inside-margin"].Phase != "recorded" {
		t.Fatal("margin refusal reserved the decision")
	}
	f.s.fault = func(at string) bool { return at == "issuance-change-after-decide" }
	if got := protectedWrite(f, w.approver, "decide_change", w.c, decideAt); got.status != 500 || got.body["operation_id"] == nil {
		t.Fatal("decision exactly at the margin was not attempted", got.raw)
	}
	f.s.fault = nil
	f.advance(2 * time.Minute)
	recovered := protectedWrite(f, w.approver, "decide_change", w.c, decideAt)
	if recovered.status != 200 || a.changes["decide-at-margin"].Phase != "recorded" || a.changes["decide-at-margin"].Module.State != protected.StateScheduled {
		t.Fatal("decision recovery after the deadline was refused", recovered.raw)
	}
}

func TestFinancialProtectedWritesRecoveryOwnershipAndConsumedEvidence(t *testing.T) {
	call := publicProtectedStage(t, "request_change")
	w, f, a := call.w, call.w.f, call.w.f.s.financialIssuance
	f.s.fault = func(at string) bool { return at == "issuance-change-after-request" }
	if got := protectedWrite(f, call.who, call.kind, w.c, call.fields); got.status != 500 || got.body["operation_id"] == nil {
		t.Fatal(got.raw)
	}
	f.s.fault = nil
	grantChannelCapabilityAt(f, "second-requester-", protectedauthority.RequestChange, "public-verifier")
	f.code(t, "another actor recovers", protectedWrite(f, w.verifier, call.kind, w.c, call.fields), 409, string(commands.EvidenceStale))
	if a.changes[call.id].Phase != "request_unknown" || a.changes[call.id].Requester != "public-requester" {
		t.Fatal("another actor changed the uncertain request")
	}
	f.advance(11 * time.Minute)
	recovered := protectedWrite(f, call.who, call.kind, w.c, call.fields)
	if recovered.status != 200 || a.changes[call.id].Phase != "recorded" {
		t.Fatal("request recovery after the deadline was refused", recovered.raw)
	}

	taken := w.claim(t, w.requester, "change_instruction", "taken-change", "")
	used := a.attestations[taken]
	used.Consumed = &financialAttestationUse{Purpose: "change_instruction", Target: "taken-change", TargetDigest: strings.Repeat("c", 64)}
	a.attestations[taken] = used
	other := w.request(t, "taken-change", protected.FieldDocumentRecipient, "customer", "synthetic_recipient_a", map[string]string{"attestation": taken})
	f.code(t, "attestation consumed by another operation", protectedWrite(f, w.requester, "request_change", w.c, other), 409, string(commands.EvidenceStale))

	fresh := protectedWrite(f, w.requester, "request_change", w.c, w.request(t, "rebound-change", protected.FieldDocumentRecipient, "customer", "synthetic_recipient_a", nil))
	f.ok(t, "request", fresh, protected.StateRequested, "")
	verify := w.verify(t, "rebound-change", field(fresh, "change_digest"))
	moved := a.attestations[verify["attestation"]]
	moved.TargetDigest = strings.Repeat("d", 64)
	var err error
	moved.Digest, err = attestationDigest(moved)
	f.must(err)
	a.attestations[verify["attestation"]] = moved
	f.code(t, "bound target digest changed", protectedWrite(f, w.verifier, "verify_change", w.c, verify), 409, string(commands.EvidenceStale))
	if a.changes["rebound-change"].Verification != nil || len(a.changes) != 2 {
		t.Fatal("refused evidence reserved an operation")
	}
}
