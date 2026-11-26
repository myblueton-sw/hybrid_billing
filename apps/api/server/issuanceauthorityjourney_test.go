package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

type issuanceJourney struct {
	t      *testing.T
	f      world
	grants int
}

func (j *issuanceJourney) grant(namespace, capability, recipient string) {
	j.t.Helper()
	j.grants++
	j.f.advance(time.Minute)
	provisionIssuanceCapability(j.t, j.f, namespace, capability, recipient, fmt.Sprintf("journey-%02d-", j.grants))
}

func TestIssuanceSetupJourneyThroughPublicProvisioningOnly(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	j := &issuanceJourney{t: t, f: f}
	for _, ns := range issuanceAuthorityNamespaces {
		if issuanceGrantCount(f, ns.domain) != 0 {
			t.Fatal("the installation starts with an issuance grant")
		}
	}
	p, err := proposeConfigurationAs(f, f.as(AdminLogin), configurationSpec())
	f.must(err)
	c, err := confirmConfigurationAs(f, f.as(Admin3Login), p)
	f.must(err)
	logins := map[string]string{"registrar": MemberLogin}
	people := map[string]string{"registrar": SampleMember}
	for _, role := range []string{"admitter", "requester", "verifier", "approver"} {
		people[role] = "journey-" + role
		logins[role] = activateProtectedMember(f, people[role])
	}
	as := func(role string) actor { return f.as(logins[role]) }
	claims := 0
	claim := func(role, purpose, target, digest string) string {
		t.Helper()
		claims++
		id := fmt.Sprintf("journey-claim-%02d", claims)
		fields := map[string]string{"attestation": id, "target": target, "statement": "Synthetic statement of the act for " + purpose + "."}
		if digest != "" {
			fields["target_digest"] = digest
		}
		f.ok(t, "attest "+purpose, attestRun(f, as(role), "attest_"+purpose, c, fields), "recorded", "")
		return id
	}

	rules := map[string]string{"rules": "journey-rules", "approvals": "1", "validity_seconds": "172800", "max_lead_seconds": "604800", "channel_age_seconds": "86400"}
	f.code(t, "rules before any grant", protectedWrite(f, f.as(AdminLogin), "propose_protected_rules", c, rules), 403, string(commands.PermissionDenied))
	j.grant("issuance_protected_authority", protectedauthority.Inspect, SampleAdmin)
	j.grant("issuance_protected_authority", protectedauthority.Inspect, SampleAdmin2)
	proposed := protectedWrite(f, f.as(AdminLogin), "propose_protected_rules", c, rules)
	f.ok(t, "propose rules", proposed, "proposed", "")
	f.ok(t, "confirm rules", protectedWrite(f, f.as(Admin2Login), "confirm_protected_rules", c, map[string]string{"rules": "journey-rules", "rules_digest": field(proposed, "rules_digest")}), "confirmed", "")

	j.grant("issuance_protected_authority", protectedauthority.RegisterChannel, people["registrar"])
	j.grant("issuance_protected_authority", protectedauthority.AdmitChannel, people["admitter"])
	registered := protectedWrite(f, as("registrar"), "register_channel", c, map[string]string{"channel": "journey-channel", "subject_kind": "entity", "kind": protected.ChannelPortal, "reference": "synthetic_channel_entity_a", "attestation": claim("registrar", "channel_registration", "journey-channel", "")})
	f.ok(t, "register", registered, "registered", "")
	channel := field(registered, "channel_digest")
	f.ok(t, "admit", protectedWrite(f, as("admitter"), "admit_channel", c, map[string]string{"channel": "journey-channel", "channel_digest": channel, "attestation": claim("admitter", "channel_admission", "journey-channel", channel)}), "admitted", "")

	j.grant("issuance_term_authority", termsauthority.RecordTerm, people["registrar"])
	j.grant("issuance_term_authority", termsauthority.BindTerm, people["registrar"])
	j.grant("issuance_term_authority", termsauthority.ApproveTerm, people["admitter"])
	j.grant("issuance_term_authority", termsauthority.ApproveBinding, people["admitter"])
	recorded := termWrite(f, as("registrar"), "record_term", c, map[string]string{"term": "journey-term", "net_days": "30"})
	f.ok(t, "record term", recorded, "pending", "")
	term := field(recorded, "term_digest")
	f.ok(t, "approve term", termWrite(f, as("admitter"), "approve_term", c, map[string]string{"term": "journey-term", "term_digest": term}), "approved", "")
	bound := termWrite(f, as("registrar"), "bind_term", c, map[string]string{"binding": "journey-binding", "term": "journey-term", "term_digest": term, "from": "2026-11-01", "to": "2027-10-31", "reason": "initial"})
	f.ok(t, "bind term", bound, "pending", "")
	f.ok(t, "approve binding", termWrite(f, as("admitter"), "approve_term_binding", c, map[string]string{"binding": "journey-binding", "binding_digest": field(bound, "binding_digest")}), "approved", "")

	f.advance(24 * time.Hour)
	request := func() map[string]string {
		return map[string]string{"change": "journey-change", "field": protected.FieldPaymentAccount, "reference": "synthetic_account_a", "channel": "journey-channel", "channel_digest": channel, "provenance": protected.ProvenanceStaff, "instruction": "in_person", "effective": f.s.clock().Add(10 * time.Minute).UTC().Format(time.RFC3339Nano)}
	}
	early := request()
	early["attestation"] = "journey-claim-absent"
	f.code(t, "request after every grant expired", protectedWrite(f, as("requester"), "request_change", c, early), 403, string(commands.PermissionDenied))
	j.grant("issuance_protected_authority", protectedauthority.RequestChange, people["requester"])
	j.grant("issuance_protected_authority", protectedauthority.VerifyChange, people["verifier"])
	j.grant("issuance_protected_authority", protectedauthority.ApproveChange, people["approver"])
	fields := request()
	fields["attestation"] = claim("requester", "change_instruction", "journey-change", "")
	f.code(t, "request while the channel's people hold expired grants", protectedWrite(f, as("requester"), "request_change", c, fields), 403, string(commands.PermissionDenied))
	j.grant("issuance_protected_authority", protectedauthority.RegisterChannel, people["registrar"])
	j.grant("issuance_protected_authority", protectedauthority.AdmitChannel, people["admitter"])
	requested := protectedWrite(f, as("requester"), "request_change", c, fields)
	f.ok(t, "request change", requested, protected.StateRequested, "")
	change := field(requested, "change_digest")
	f.ok(t, "verify change", protectedWrite(f, as("verifier"), "verify_change", c, map[string]string{"change": "journey-change", "change_digest": change, "outcome": protected.OutcomeConfirmed, "attestation": claim("verifier", "change_verification", "journey-change", change)}), protected.StateConfirmed, "")
	f.ok(t, "decide change", protectedWrite(f, as("approver"), "decide_change", c, map[string]string{"change": "journey-change", "change_digest": change, "verdict": protected.VerdictApprove}), protected.StateScheduled, "")

	j.grant("issuance_protected_authority", protectedauthority.Inspect, SampleAdmin3)
	j.grant("issuance_term_authority", termsauthority.Inspect, SampleAdmin3)
	reader := f.as(Admin3Login)
	summary := protectedSetupRead(f, reader, "protected", c, nil)
	if summary.status != 200 || field(summary, "rules_state") != "confirmed" || field(summary, "channels") != "1" || field(summary, "account_latest_scheduled") != "journey-change" {
		t.Fatal(summary.raw)
	}
	terms := termsRead(f, reader, c, nil)
	if terms.status != 200 || field(terms, "term_state") != "approved" || field(terms, "net_days") != "30" || field(terms, "count") != "1" {
		t.Fatal(terms.raw)
	}
}

func TestIssuanceAuthorityReplayRechecksTheGovernorAndTheSession(t *testing.T) {
	for _, ns := range issuanceAuthorityNamespaces {
		t.Run(ns.name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root, second := f.as(AdminLogin), f.as(Admin2Login)
			capability := ns.capabilities[0]
			long := issuanceAuthorityTimes(f, capability)
			long["not_after"] = f.s.clock().Add(25 * time.Hour).Format(time.RFC3339Nano)
			f.code(t, "validity beyond the registry maximum", f.run(root, ns.name+".propose_console", "too-long", long), 422, string(commands.InvalidRequest))
			ceiling := f.run(root, ns.name+".propose_console", "replayed", issuanceAuthorityTimes(f, capability))
			f.ok(t, "proposal", ceiling, "pending", "")
			body := fmt.Sprintf(`{"request_id":"replay-request","target_id":"replayed","idempotency_key":"replay-key","expected_revision":%q,"rationale":"sample reason","body":{"digest":%q}}`, f.s.Revision(), field(ceiling, "digest"))
			first := f.call(ns.name+".approve", body, second.cookie, second.csrf)
			f.ok(t, "approval", first, "applied", "")
			again := f.call(ns.name+".approve", body, second.cookie, second.csrf)
			if again.status != 200 || again.body["replay"] != true || field(again, "grant") != field(first, "grant") {
				t.Fatal("an identical retry was not replayed", again.raw)
			}
			f.ok(t, "suspend the governor's membership", f.run(root, "identity.suspend_membership", SampleAdmin2, nil), "membership_suspended", "")
			if got := f.call(ns.name+".approve", body, second.cookie, second.csrf); got.status == 200 {
				t.Fatal("a cached approval was replayed to a former governor", got.raw)
			}
			if got := issuanceAuthorityRead(f, second, ns.name, "replayed"); got.status == 200 {
				t.Fatal("a former governor inspected", got.raw)
			}
			f.advance(31 * time.Minute)
			f.code(t, "ended session", f.runWith(root, f.s.Revision(), ns.name+".propose_console", "after-idle", issuanceAuthorityTimes(f, capability)), 401, string(commands.Unauthenticated))
			if issuanceGrantCount(f, ns.domain) != 1 {
				t.Fatal("a refusal changed the grants")
			}
		})
	}
}
