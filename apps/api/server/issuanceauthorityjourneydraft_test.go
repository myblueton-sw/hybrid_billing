package server

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

type journeySetup struct {
	configuration          financialIssuanceConfiguration
	binding, bindingDigest string
	logins                 map[string]string
	grants                 int
}

func (j *journeySetup) grant(t *testing.T, f world, namespace, capability, recipient string) {
	t.Helper()
	j.grants++
	f.advance(time.Minute)
	provisionIssuanceCapability(t, f, namespace, capability, recipient, fmt.Sprintf("journey-draft-%02d-", j.grants))
}

func completeJourneySetup(t *testing.T, f world, skip string) *journeySetup {
	t.Helper()
	j := &journeySetup{logins: map[string]string{"registrar": MemberLogin}}
	for _, ns := range issuanceAuthorityNamespaces {
		if issuanceGrantCount(f, ns.domain) != 0 {
			t.Fatal("the installation starts with an issuance grant")
		}
	}
	if len(f.s.financialIssuance.configurations) != 0 {
		t.Fatal("the installation starts with an issuance configuration")
	}
	j.grant(t, f, "issuance_protected_authority", protectedauthority.Inspect, SampleAdmin)
	j.grant(t, f, "issuance_protected_authority", protectedauthority.Inspect, SampleAdmin2)
	context := setupContextRead(f, f.as(AdminLogin), "sample-contract-01", "2026-11-16")
	if context.status != 200 {
		t.Fatal(context.raw)
	}
	spec := configurationSpec()
	settings := map[string]string{"contract": spec.Contract, "on": "2026-11-16", "scope_digest": field(context, "scope_digest"), "mode": spec.Mode, "profile": spec.Profile, "region": spec.Region, "tax": spec.Tax, "zone": spec.Zone, "languages": "en,ko,ar"}
	proposal := draftRun(f, f.as(AdminLogin), "issuance_setup.propose_configuration", "journey-draft-configuration", settings)
	f.ok(t, "propose configuration", proposal, "proposed", "")
	confirmation := draftRun(f, f.as(Admin2Login), "issuance_setup.confirm_configuration", "journey-draft-configuration", map[string]string{"digest": field(proposal, "digest")})
	f.ok(t, "confirm configuration", confirmation, "confirmed", "")
	c, ok := f.s.financialIssuance.configurations["journey-draft-configuration"]
	if !ok || c.Digest != field(confirmation, "digest") || c.State != "confirmed" {
		t.Fatal("retained configuration differs from the public receipt")
	}
	j.configuration = c
	people := map[string]string{"registrar": SampleMember}
	for _, role := range []string{"admitter", "requester", "verifier", "approver"} {
		people[role] = "journey-draft-" + role
		j.logins[role] = activateProtectedMember(f, people[role])
	}
	as := func(role string) actor { return f.as(j.logins[role]) }
	claims := 0
	claim := func(role, purpose, target, digest string) string {
		t.Helper()
		claims++
		id := fmt.Sprintf("journey-draft-claim-%02d", claims)
		fields := map[string]string{"attestation": id, "target": target, "statement": "Synthetic statement of the act for " + purpose + "."}
		if digest != "" {
			fields["target_digest"] = digest
		}
		f.ok(t, "attest "+purpose, attestRun(f, as(role), "attest_"+purpose, c, fields), "recorded", "")
		return id
	}
	rules := map[string]string{"rules": "journey-draft-rules", "approvals": "1", "validity_seconds": "172800", "max_lead_seconds": "604800", "channel_age_seconds": "86400"}
	proposed := protectedWrite(f, f.as(AdminLogin), "propose_protected_rules", c, rules)
	f.ok(t, "propose rules", proposed, "proposed", "")
	f.ok(t, "confirm rules", protectedWrite(f, f.as(Admin2Login), "confirm_protected_rules", c, map[string]string{"rules": "journey-draft-rules", "rules_digest": field(proposed, "rules_digest")}), "confirmed", "")

	j.grant(t, f, "issuance_protected_authority", protectedauthority.RegisterChannel, people["registrar"])
	j.grant(t, f, "issuance_protected_authority", protectedauthority.AdmitChannel, people["admitter"])
	subjects := []struct{ kind, field, reference string }{{"entity", protected.FieldPaymentAccount, "synthetic_account_a"}, {"customer", protected.FieldDocumentRecipient, "synthetic_recipient_a"}, {"document_kind", protected.FieldDocumentTemplate, "synthetic_template_a"}}
	channels := map[string]string{}
	for _, s := range subjects {
		id := "journey-draft-channel-" + s.kind
		registered := protectedWrite(f, as("registrar"), "register_channel", c, map[string]string{"channel": id, "subject_kind": s.kind, "kind": protected.ChannelPortal, "reference": "synthetic_channel_" + s.kind + "_a", "attestation": claim("registrar", "channel_registration", id, "")})
		f.ok(t, "register "+s.kind, registered, "registered", "")
		channels[s.kind] = field(registered, "channel_digest")
		f.ok(t, "admit "+s.kind, protectedWrite(f, as("admitter"), "admit_channel", c, map[string]string{"channel": id, "channel_digest": channels[s.kind], "attestation": claim("admitter", "channel_admission", id, channels[s.kind])}), "admitted", "")
	}

	f.advance(24 * time.Hour)
	for _, right := range []struct{ capability, role string }{{protectedauthority.RegisterChannel, "registrar"}, {protectedauthority.AdmitChannel, "admitter"}, {protectedauthority.RequestChange, "requester"}, {protectedauthority.VerifyChange, "verifier"}, {protectedauthority.ApproveChange, "approver"}} {
		j.grant(t, f, "issuance_protected_authority", right.capability, people[right.role])
	}
	for _, s := range subjects {
		if s.kind == skip {
			continue
		}
		id := "journey-draft-change-" + s.kind
		requested := protectedWrite(f, as("requester"), "request_change", c, map[string]string{"change": id, "field": s.field, "reference": s.reference, "channel": "journey-draft-channel-" + s.kind, "channel_digest": channels[s.kind], "provenance": protected.ProvenanceStaff, "instruction": "in_person", "effective": f.s.clock().Add(10 * time.Minute).UTC().Format(time.RFC3339Nano), "attestation": claim("requester", "change_instruction", id, "")})
		f.ok(t, "request "+s.kind, requested, protected.StateRequested, "")
		digest := field(requested, "change_digest")
		f.ok(t, "verify "+s.kind, protectedWrite(f, as("verifier"), "verify_change", c, map[string]string{"change": id, "change_digest": digest, "outcome": protected.OutcomeConfirmed, "attestation": claim("verifier", "change_verification", id, digest)}), protected.StateConfirmed, "")
		f.ok(t, "decide "+s.kind, protectedWrite(f, as("approver"), "decide_change", c, map[string]string{"change": id, "change_digest": digest, "verdict": protected.VerdictApprove}), protected.StateScheduled, "")
	}

	j.grant(t, f, "issuance_term_authority", termsauthority.RecordTerm, people["registrar"])
	j.grant(t, f, "issuance_term_authority", termsauthority.BindTerm, people["registrar"])
	j.grant(t, f, "issuance_term_authority", termsauthority.ApproveTerm, people["admitter"])
	j.grant(t, f, "issuance_term_authority", termsauthority.ApproveBinding, people["admitter"])
	recorded := termWrite(f, as("registrar"), "record_term", c, map[string]string{"term": "journey-draft-term", "net_days": "30"})
	f.ok(t, "record term", recorded, "pending", "")
	term := field(recorded, "term_digest")
	f.ok(t, "approve term", termWrite(f, as("admitter"), "approve_term", c, map[string]string{"term": "journey-draft-term", "term_digest": term}), "approved", "")
	bound := termWrite(f, as("registrar"), "bind_term", c, map[string]string{"binding": "journey-draft-binding", "term": "journey-draft-term", "term_digest": term, "from": "2026-01-01", "to": "2026-12-31", "reason": "initial"})
	f.ok(t, "bind term", bound, "pending", "")
	j.binding, j.bindingDigest = "journey-draft-binding", field(bound, "binding_digest")
	f.ok(t, "approve binding", termWrite(f, as("admitter"), "approve_term_binding", c, map[string]string{"binding": j.binding, "binding_digest": j.bindingDigest}), "approved", "")
	f.advance(15 * time.Minute)
	return j
}

func provisionJourneyDrafts(t *testing.T, f world, j *journeySetup) string {
	t.Helper()
	reviewer := activateProtectedMember(f, "journey-draft-reviewer")
	for _, right := range []struct{ capability, recipient string }{{draftauthority.Submit, SampleMember}, {draftauthority.Inspect, SampleMember}, {draftauthority.Approve, "journey-draft-reviewer"}, {draftauthority.Inspect, "journey-draft-reviewer"}} {
		j.grant(t, f, "issuance_draft_authority", right.capability, right.recipient)
	}
	return reviewer
}

func journeySubmission(c financialIssuanceConfiguration, request, digest string) map[string]string {
	return map[string]string{"configuration": c.Spec.ID, "configuration_digest": c.Digest, "final_request": request, "final_digest": digest, "language": "en"}
}

func TestIssuanceJourneyReachesApprovedDraftsForTwoInvoices(t *testing.T) {
	var j *journeySetup
	f, _, _, final := issuanceSourceWorld(t, func(f world) { j = completeJourneySetup(t, f, "") })
	a, c := f.s.financialIssuance, j.configuration
	f.code(t, "submit before any draft grant", draftRun(f, f.as(MemberLogin), "issuance_draft.submit", "journey-draft-early", journeySubmission(c, "issuance-final", final)), 403, string(commands.PermissionDenied))
	reviewer := provisionJourneyDrafts(t, f, j)
	setupBefore := []int{len(a.configurations), len(a.channels), len(a.changes), len(a.attestations), len(a.termIntents), len(a.bindingIntents)}
	grants := func() []int {
		out := []int{}
		for _, ns := range issuanceAuthorityNamespaces {
			out = append(out, issuanceGrantCount(f, ns.domain))
		}
		return out
	}
	approveDraft := func(name, request, digest string) {
		t.Helper()
		submitted := draftRun(f, f.as(MemberLogin), "issuance_draft.submit", name, journeySubmission(c, request, digest))
		f.ok(t, "submit "+name, submitted, "submitted", "")
		draft, ok := a.drafts[name]
		if !ok || draft.Application == nil || draft.Application.Configuration.Spec.ID != c.Spec.ID || draft.Application.Configuration.Digest != c.Digest {
			t.Fatal("draft does not consume the journey configuration")
		}
		review := map[string]string{"draft": name, "draft_digest": draft.Digest, "binding": j.binding, "binding_digest": j.bindingDigest}
		f.code(t, "submitter reviews own draft "+name, draftRun(f, f.as(MemberLogin), "issuance_draft.prepare_approval", name+"-own-review", review), 403, string(commands.PermissionDenied))
		prepared := draftRun(f, f.as(reviewer), "issuance_draft.prepare_approval", name+"-review", review)
		f.ok(t, "prepare "+name, prepared, "prepared", "")
		f.ok(t, "approve "+name, draftRun(f, f.as(reviewer), "issuance_draft.approve", name+"-review", map[string]string{"digest": field(prepared, "review_digest")}), "approved", "")
		r := a.draftReviews[name+"-review"]
		if len(r.References) != 3 || r.State != "approved" || r.BindingID != j.binding {
			t.Fatal("approved draft lacks the three protected references or the journey binding")
		}
		for _, reference := range r.References {
			if reference.Change != "journey-draft-change-entity" && reference.Change != "journey-draft-change-customer" && reference.Change != "journey-draft-change-document_kind" {
				t.Fatal("approved draft consumed a change the journey did not make: " + reference.Change)
			}
		}
	}
	approveDraft("journey-draft-a", "issuance-final", final)
	granted := grants()

	from, to := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	candidate, candidateDigest := captureInvoiceInWorld(t, f, f.as(MemberLogin), f.as(Admin3Login), "journey-b", "2026-11", from, to)
	second, secondDigest := finalizeInvoiceInWorld(t, f, "journey-b", "2026-11", candidate, candidateDigest)
	approveDraft("journey-draft-b", second, secondDigest)
	if got := []int{len(a.configurations), len(a.channels), len(a.changes), len(a.attestations), len(a.termIntents), len(a.bindingIntents)}; !reflect.DeepEqual(got, setupBefore) {
		t.Fatalf("second invoice changed setup records: %v, was %v", got, setupBefore)
	}
	if got := grants(); !reflect.DeepEqual(got, granted) {
		t.Fatalf("second invoice needed further issuance grants: %v, was %v", got, granted)
	}
	first, next := a.drafts["journey-draft-a"], a.drafts["journey-draft-b"]
	if first.Application.Configuration.Digest != next.Application.Configuration.Digest || first.Application.Source.Digest == next.Application.Source.Digest || first.Application.Source.Basis.Subject.Target == next.Application.Source.Basis.Subject.Target {
		t.Fatal("second invoice did not reuse the setup for a different source")
	}
}

func TestIssuanceJourneyDraftApprovalIsRefusedWhileAProtectedReferenceIsMissing(t *testing.T) {
	for _, missing := range []string{"entity", "customer", "document_kind"} {
		t.Run(missing, func(t *testing.T) {
			var j *journeySetup
			f, _, _, final := issuanceSourceWorld(t, func(f world) { j = completeJourneySetup(t, f, missing) })
			reviewer := provisionJourneyDrafts(t, f, j)
			a := f.s.financialIssuance
			f.ok(t, "submit", draftRun(f, f.as(MemberLogin), "issuance_draft.submit", "journey-draft-missing", journeySubmission(j.configuration, "issuance-final", final)), "submitted", "")
			got := draftRun(f, f.as(reviewer), "issuance_draft.prepare_approval", "journey-draft-missing-review", map[string]string{"draft": "journey-draft-missing", "draft_digest": a.drafts["journey-draft-missing"].Digest, "binding": j.binding, "binding_digest": j.bindingDigest})
			f.code(t, "prepare without the "+missing+" reference", got, 422, string(commands.PrerequisiteFailed))
			if len(a.draftReviews) != 0 || len(a.draftReviewOrders["journey-draft-missing"]) != 0 {
				t.Fatal("a review of a draft without the " + missing + " reference is retained")
			}
		})
	}
}
