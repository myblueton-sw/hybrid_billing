package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
)

func attestBody(f world, key string, c financialIssuanceConfiguration, fields map[string]string) string {
	body := map[string]string{"configuration": c.Spec.ID, "digest": c.Digest}
	for name, value := range fields {
		body[name] = value
	}
	raw, _ := json.Marshal(map[string]any{"request_id": key, "target_id": c.Spec.Contract, "idempotency_key": key, "expected_revision": f.s.Revision(), "body": body})
	return string(raw)
}

func attestRun(f world, who actor, command string, c financialIssuanceConfiguration, fields map[string]string) reply {
	f.t.Helper()
	return f.call("issuance_setup."+command, attestBody(f, fmt.Sprintf("attest-%d", keys.Add(1)), c, fields), who.cookie, who.csrf)
}

func registrationStatement(id, target string) map[string]string {
	return map[string]string{"attestation": id, "target": target, "statement": "Contact reference taken from the signed onboarding record.", "locator": "Contract binder, section 4"}
}

func attestationWorld(t *testing.T) (world, actor, actor, financialChannelSpec, financialIssuanceConfiguration) {
	t.Helper()
	f, registrar, admitter, spec := configurationChannelWorld(t)
	return f, registrar, admitter, spec, f.s.financialIssuance.configurations[spec.Selector.Configuration.ID]
}

func TestFinancialSetupAttestationRecordsOwnedClaim(t *testing.T) {
	f, registrar, admitter, spec, c := attestationWorld(t)
	a := f.s.financialIssuance
	input := registrationStatement("registration-claim", spec.ID)
	got := attestRun(f, registrar, "attest_channel_registration", c, input)
	f.ok(t, "attest", got, "recorded", "")
	p := a.attestations["registration-claim"]
	digest, err := attestationDigest(p)
	f.must(err)
	want := map[string]string{"schema": "hb_issuance_setup_attestation_v1", "contract": c.Spec.Contract, "configuration": c.Spec.ID, "attestation": "registration-claim", "purpose": "channel_registration", "target": spec.ID, "actor": SampleMember, "digest": digest, "state": "unconsumed", "assurance": "human_attestation", "recorded_at": setupInstant(p.RecordedAt)}
	fields, _ := got.body["fields"].(map[string]any)
	if len(fields) != len(want) || p.Digest != digest || p.Context != configurationReference(c) || p.RecordedAt.IsZero() || p.TargetDigest != "" {
		t.Fatal(got.raw)
	}
	for name, value := range want {
		if field(got, name) != value {
			t.Fatalf("%s = %q, want %q", name, field(got, name), value)
		}
	}
	if strings.Contains(got.raw, "onboarding record") || strings.Contains(got.raw, "Contract binder") {
		t.Fatal("receipt echoed the statement")
	}

	again := attestRun(f, registrar, "attest_channel_registration", c, input)
	f.ok(t, "identical new request", again, "recorded", "")
	if field(again, "recorded_at") != field(got, "recorded_at") || field(again, "digest") != digest || len(a.attestations) != 1 {
		t.Fatal("identical request created another claim", again.raw)
	}
	for name, value := range map[string]string{"statement": "A different statement.", "target": "another-channel", "locator": ""} {
		changed := registrationStatement("registration-claim", spec.ID)
		changed[name] = value
		got := attestRun(f, registrar, "attest_channel_registration", c, changed)
		f.code(t, "changed "+name, got, 409, string(commands.EvidenceStale))
		if strings.Contains(got.raw, "onboarding") || strings.Contains(got.raw, SampleMember) {
			t.Fatal("collision disclosed the existing claim")
		}
	}
	if !reflect.DeepEqual(a.attestations["registration-claim"], p) {
		t.Fatal("refused request changed the claim")
	}

	root := f.as(AdminLogin)
	for name, who := range map[string]actor{"another capability": admitter, "administrator": root} {
		f.code(t, name, attestRun(f, who, "attest_channel_registration", c, registrationStatement("other-claim", "other-channel")), 403, string(commands.PermissionDenied))
	}
	for _, command := range []string{"attest_channel_admission", "attest_change_instruction", "attest_change_verification"} {
		fields := registrationStatement("other-claim", "other-channel")
		if command != "attest_change_instruction" {
			fields["target_digest"] = strings.Repeat("a", 64)
		}
		f.code(t, command, attestRun(f, registrar, command, c, fields), 403, string(commands.PermissionDenied))
	}
	f.code(t, "existing channel identifier", func() reply {
		_, err := registerChannelAs(f, registrar, spec)
		f.must(err)
		return attestRun(f, registrar, "attest_channel_registration", c, registrationStatement("late-claim", spec.ID))
	}(), 409, string(commands.EvidenceStale))
	if len(a.attestations) != 1 {
		t.Fatal("refused requests retained claims")
	}
}

func TestFinancialSetupAttestationShapeAndText(t *testing.T) {
	f, registrar, _, spec, c := attestationWorld(t)
	cases := map[string]map[string]string{
		"blank statement":         {"statement": " \n\t"},
		"oversized statement":     {"statement": strings.Repeat("a", 2001)},
		"oversized locator":       {"locator": strings.Repeat("a", 513)},
		"replacement character":   {"statement": "Altered \uFFFD text"},
		"carriage return":         {"statement": "Line one\r\nline two"},
		"control character":       {"locator": "Binder\x00"},
		"escaped beyond the row":  {"statement": strings.Repeat("<", 2000)},
		"escaped beyond the page": {"statement": strings.Repeat(`"`, 1700)},
		"digest on a new target":  {"target_digest": strings.Repeat("a", 64)},
		"malformed attestation":   {"attestation": "bad id"},
		"oversized target":        {"target": strings.Repeat("t", 129)},
	}
	for name, change := range cases {
		fields := registrationStatement("shape-claim", spec.ID)
		for k, v := range change {
			fields[k] = v
		}
		f.code(t, name, attestRun(f, registrar, "attest_channel_registration", c, fields), 422, string(commands.InvalidRequest))
	}
	stale := c
	stale.Digest = strings.Repeat("0", 64)
	f.code(t, "stale configuration", attestRun(f, registrar, "attest_channel_registration", stale, registrationStatement("shape-claim", spec.ID)), 409, string(commands.EvidenceStale))
	missing := c
	missing.Spec.ID = "missing-configuration"
	f.code(t, "unknown configuration", attestRun(f, registrar, "attest_channel_registration", missing, registrationStatement("shape-claim", spec.ID)), 403, string(commands.PermissionDenied))
	if len(f.s.financialIssuance.attestations) != 0 {
		t.Fatal("refused shape retained a claim")
	}
	for id, statement := range map[string]string{"largest-plain": strings.Repeat("a", 2000), "escaped-within-bounds": strings.Repeat(`"`, 1300), "multiline": "First line\n\tsecond line"} {
		fields := registrationStatement(id, spec.ID)
		fields["statement"], fields["locator"] = statement, strings.Repeat("b", 512)
		if id != "largest-plain" {
			fields["locator"] = ""
		}
		f.ok(t, id, attestRun(f, registrar, "attest_channel_registration", c, fields), "recorded", "")
	}
	grantSetupInspectionForTest(f, true)
	page := protectedSetupRead(f, f.as(MemberLogin), "attestations", c, nil)
	if page.status != 200 || field(page, "count") != "3" {
		t.Fatal("admitted claims do not fit one page", page.raw)
	}
}

func TestFinancialSetupAttestationCapacityFaultsAndWithdrawal(t *testing.T) {
	f, registrar, admitter, spec, c := attestationWorld(t)
	a := f.s.financialIssuance
	revision := f.s.Revision()
	f.s.fault = func(at string) bool { return at == "issuance-attestation-before-retention" }
	f.code(t, "before retention", attestRun(f, registrar, "attest_channel_registration", c, registrationStatement("faulted", spec.ID)), 503, string(commands.DependencyUnavailable))
	if len(a.attestations) != 0 || f.s.Revision() != revision {
		t.Fatal("clean failure retained a claim or advanced the revision")
	}
	f.s.fault = func(at string) bool { return at == "issuance-attestation-after-retention" }
	body := attestBody(f, "uncertain-attestation", c, registrationStatement("faulted", spec.ID))
	uncertain := f.call("issuance_setup.attest_channel_registration", body, registrar.cookie, registrar.csrf)
	f.s.fault = nil
	if uncertain.status != 500 || uncertain.body["operation_id"] == nil || len(a.attestations) != 1 || f.s.Revision() == revision {
		t.Fatal("uncertain outcome", uncertain.raw)
	}
	replay := f.call("issuance_setup.attest_channel_registration", body, registrar.cookie, registrar.csrf)
	if replay.status != 500 || replay.body["operation_id"] != uncertain.body["operation_id"] {
		t.Fatal("same key did not replay the stored outcome", replay.raw)
	}
	f.ok(t, "new identical request", attestRun(f, registrar, "attest_channel_registration", c, registrationStatement("faulted", spec.ID)), "recorded", "")

	for i := 1; i < financialAttestationPending; i++ {
		f.ok(t, "fill", attestRun(f, registrar, "attest_channel_registration", c, registrationStatement(fmt.Sprintf("pending-%02d", i), spec.ID)), "recorded", "")
	}
	f.code(t, "per-actor bound", attestRun(f, registrar, "attest_channel_registration", c, registrationStatement("pending-over", spec.ID)), 422, string(commands.PrerequisiteFailed))

	withdraw := map[string]string{"attestation": "pending-01"}
	f.code(t, "another actor", attestRun(f, admitter, "withdraw_channel_admission", c, withdraw), 403, string(commands.PermissionDenied))
	f.code(t, "missing claim", attestRun(f, registrar, "withdraw_channel_registration", c, map[string]string{"attestation": "absent-claim"}), 403, string(commands.PermissionDenied))
	f.s.fault = func(at string) bool { return at == "issuance-attestation-before-withdrawal" }
	f.code(t, "before withdrawal", attestRun(f, registrar, "withdraw_channel_registration", c, withdraw), 503, string(commands.DependencyUnavailable))
	f.s.fault = nil
	if !a.attestations["pending-01"].WithdrawnAt.IsZero() {
		t.Fatal("clean failure withdrew the claim")
	}
	done := attestRun(f, registrar, "withdraw_channel_registration", c, withdraw)
	f.ok(t, "withdraw", done, "withdrawn", "")
	repeated := attestRun(f, registrar, "withdraw_channel_registration", c, withdraw)
	f.ok(t, "repeat", repeated, "withdrawn", "")
	p := a.attestations["pending-01"]
	digest, err := attestationDigest(p)
	f.must(err)
	if field(done, "state") != "withdrawn" || p.WithdrawnAt.IsZero() || digest != p.Digest || field(repeated, "digest") != p.Digest {
		t.Fatal("withdrawal changed the claim identity", done.raw)
	}
	f.ok(t, "freed slot", attestRun(f, registrar, "attest_channel_registration", c, registrationStatement("pending-over", spec.ID)), "recorded", "")

	used := a.attestations["pending-02"]
	used.Consumed = &financialAttestationUse{Purpose: used.Purpose, Target: used.Target, TargetDigest: strings.Repeat("c", 64)}
	a.attestations["pending-02"] = used
	f.code(t, "consumed claim", attestRun(f, registrar, "withdraw_channel_registration", c, map[string]string{"attestation": "pending-02"}), 422, string(commands.PrerequisiteFailed))

	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == SampleMember && grant.Terms.Kind == protectedauthority.ResourceKind {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
		}
	}
	f.code(t, "capability lost", attestRun(f, registrar, "withdraw_channel_registration", c, map[string]string{"attestation": "pending-03"}), 403, string(commands.PermissionDenied))
	f.s.enterHold("Attestation writes held")
	f.code(t, "held", attestRun(f, admitter, "attest_channel_admission", c, map[string]string{"attestation": "held", "target": spec.ID, "target_digest": strings.Repeat("a", 64), "statement": "Held"}), 503, string(commands.DependencyUnavailable))
}

func TestFinancialSetupAttestationExistingTargets(t *testing.T) {
	f, registrar, admitter, spec, c := attestationWorld(t)
	a := f.s.financialIssuance
	admission := func(id, target, digest string) map[string]string {
		return map[string]string{"attestation": id, "target": target, "target_digest": digest, "statement": "Compared the contact reference with the independent customer record."}
	}
	f.code(t, "target does not exist", attestRun(f, admitter, "attest_channel_admission", c, admission("early", spec.ID, strings.Repeat("a", 64))), 409, string(commands.EvidenceStale))
	channel, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	f.code(t, "wrong target digest", attestRun(f, admitter, "attest_channel_admission", c, admission("early", spec.ID, strings.Repeat("a", 64))), 409, string(commands.EvidenceStale))
	missingDigest := admission("early", spec.ID, channel.Digest)
	delete(missingDigest, "target_digest")
	f.code(t, "digest required", attestRun(f, admitter, "attest_channel_admission", c, missingDigest), 422, string(commands.InvalidRequest))
	grantChannelCapabilityAt(f, "self-", protectedauthority.AdmitChannel, SampleMember)
	f.code(t, "registrar attests admission", attestRun(f, registrar, "attest_channel_admission", c, admission("self", spec.ID, channel.Digest)), 403, string(commands.SelfApprovalDenied))
	got := attestRun(f, admitter, "attest_channel_admission", c, admission("admission-claim", spec.ID, channel.Digest))
	f.ok(t, "admission", got, "recorded", "")
	if p := a.attestations["admission-claim"]; p.TargetDigest != channel.Digest || p.Actor == SampleMember || len(a.attestations) != 1 {
		t.Fatal("admission claim not bound to the registered channel")
	}
	_, err = admitChannelAs(f, admitter, channel)
	f.must(err)
	f.code(t, "already admitted", attestRun(f, admitter, "attest_channel_admission", c, admission("late", spec.ID, channel.Digest)), 409, string(commands.EvidenceStale))
	f.ok(t, "identity precedes target state", attestRun(f, admitter, "attest_channel_admission", c, admission("admission-claim", spec.ID, channel.Digest)), "recorded", "")
}

func TestFinancialSetupAttestationChangeTargets(t *testing.T) {
	w := configuredSetup(t)
	f, c, a := w.f, w.configuration, w.f.s.financialIssuance
	instruction := map[string]string{"attestation": "instruction-claim", "target": w.change.ID, "statement": "Received the template change request in person from the named contact."}
	f.code(t, "verifier attests instruction", attestRun(f, w.verifier, "attest_change_instruction", c, instruction), 403, string(commands.PermissionDenied))
	f.ok(t, "instruction", attestRun(f, w.requester, "attest_change_instruction", c, instruction), "recorded", "")
	verification := func(id, digest string) map[string]string {
		return map[string]string{"attestation": id, "target": w.change.ID, "target_digest": digest, "statement": "Called the registered contact, who confirmed the requested template."}
	}
	f.code(t, "change does not exist", attestRun(f, w.verifier, "attest_change_verification", c, verification("early", strings.Repeat("a", 64))), 409, string(commands.EvidenceStale))
	p, err := requestChangeAs(f, w.requester, w.change)
	f.must(err)
	f.code(t, "existing change identifier", attestRun(f, w.requester, "attest_change_instruction", c, map[string]string{"attestation": "late", "target": w.change.ID, "statement": "Late"}), 409, string(commands.EvidenceStale))
	f.code(t, "wrong change digest", attestRun(f, w.verifier, "attest_change_verification", c, verification("early", strings.Repeat("a", 64))), 409, string(commands.EvidenceStale))
	grantChannelCapabilityAt(f, "self-", protectedauthority.VerifyChange, "setup-requester")
	f.code(t, "requester attests verification", attestRun(f, w.requester, "attest_change_verification", c, verification("self", p.Digest)), 403, string(commands.SelfApprovalDenied))
	f.ok(t, "verification", attestRun(f, w.verifier, "attest_change_verification", c, verification("verification-claim", p.Digest)), "recorded", "")
	if got := a.attestations["verification-claim"]; got.TargetDigest != p.Digest || got.Actor != "setup-verifier" || got.Purpose != "change_verification" {
		t.Fatal("verification claim not bound to the requested change")
	}
	_, err = verifyChangeAs(f, w.verifier, p)
	f.must(err)
	f.code(t, "already verified", attestRun(f, w.verifier, "attest_change_verification", c, verification("late-verification", p.Digest)), 409, string(commands.EvidenceStale))
	if p.Module.State != protected.StateRequested || len(a.attestations) != 2 {
		t.Fatal("fixture or retained claims differ")
	}
}

func TestFinancialSetupAttestationsAndReferencesRead(t *testing.T) {
	f, registrar, _, spec, c := attestationWorld(t)
	a := f.s.financialIssuance
	channel, err := registerChannelAs(f, registrar, spec)
	f.must(err)
	for _, id := range []string{"claim-a", "claim-b", "claim-c"} {
		f.ok(t, id, attestRun(f, registrar, "attest_channel_registration", c, registrationStatement(id, "future-"+id)), "recorded", "")
	}
	f.ok(t, "withdraw", attestRun(f, registrar, "withdraw_channel_registration", c, map[string]string{"attestation": "claim-b"}), "withdrawn", "")
	f.code(t, "inspection required", protectedSetupRead(f, registrar, "attestations", c, nil), 403, string(commands.PermissionDenied))
	f.code(t, "inspection required", protectedSetupRead(f, registrar, "references", c, nil), 403, string(commands.PermissionDenied))
	grantSetupInspectionForTest(f, true)
	reader := f.as(MemberLogin)

	used := a.attestations["claim-c"]
	used.Target = spec.ID
	used.Digest, err = attestationDigest(used)
	f.must(err)
	used.Consumed = &financialAttestationUse{Purpose: used.Purpose, Target: spec.ID, TargetDigest: channel.Digest}
	used.Context = financialConfigurationReference{Schema: used.Context.Schema, ID: "another-configuration", Digest: strings.Repeat("d", 64)}
	used.Digest, err = attestationDigest(used)
	f.must(err)
	a.attestations["claim-c"] = used
	foreign := a.attestations["claim-a"]
	foreign.ID, foreign.Context = "claim-other-context", used.Context
	foreign.Digest, err = attestationDigest(foreign)
	f.must(err)
	a.attestations[foreign.ID] = foreign

	before, revision := len(a.attestations), f.s.Revision()
	page := protectedSetupRead(f, reader, "attestations", c, nil)
	if page.status != 200 || field(page, "count") != "3" || field(page, "schema") != "hb_issuance_setup_attestation_page_v1" || field(page, "current_eligibility") != "unchecked" || field(page, "next") != "" {
		t.Fatal(page.raw)
	}
	states := map[string]string{}
	for i := 0; i < 3; i++ {
		row := protectedSetupRow(t, page, i)
		states[row["attestation"].(string)] = row["state"].(string)
		if row["assurance"] != "human_attestation" || row["statement"] == "" || row["actor"] != SampleMember || len(row) != 10 {
			t.Fatalf("row %v", row)
		}
	}
	if !reflect.DeepEqual(states, map[string]string{"claim-a": "unconsumed", "claim-b": "withdrawn", "claim-c": "consumed"}) {
		t.Fatalf("scope and states %v", states)
	}
	if len(a.attestations) != before || f.s.Revision() != revision {
		t.Fatal("read changed state")
	}
	f.s.enterHold("Attestation history inspection")
	if held := protectedSetupRead(f, reader, "attestations", c, nil); held.status != 200 || field(held, "snapshot") != field(page, "snapshot") {
		t.Fatal("held read", held.raw)
	}

	references := protectedSetupRead(f, reader, "references", c, nil)
	fields, _ := references.body["fields"].(map[string]any)
	if references.status != 200 || len(fields) != 18 || field(references, "schema") != "hb_issuance_setup_references_v1" || field(references, "current_eligibility") != "unchecked" {
		t.Fatal(references.raw)
	}
	fingerprints := map[string]bool{}
	for i, entry := range setupReferenceCatalog {
		var row map[string]string
		if err := json.Unmarshal([]byte(field(references, fmt.Sprintf("reference_%02d", i))), &row); err != nil || row["code"] != entry.Code || row["use"] != entry.Use || row["applicability"] != "synthetic_only" || len(row) != 3 {
			t.Fatalf("reference %d %v", i, row)
		}
		ref, ok := setupReference(entry.Code, entry.Use)
		if !ok || fingerprints[ref.Fingerprint] || !validPlanHash(ref.Fingerprint) || strings.Contains(references.raw, ref.ID) || strings.Contains(references.raw, ref.Fingerprint) {
			t.Fatalf("reference %s resolved or disclosed incorrectly", entry.Code)
		}
		fingerprints[ref.Fingerprint] = true
	}
	if _, ok := setupReference("synthetic_account_a", protected.FieldDocumentRecipient); ok || len(setupReferenceCatalog) != 13 {
		t.Fatal("reference resolved for another use")
	}
	f.code(t, "cursor on references", protectedSetupRead(f, reader, "references", c, map[string]string{"after": "x", "snapshot": strings.Repeat("a", 64)}), 422, string(commands.InvalidRequest))

	for _, mode := range []string{"statement", "consumed target", "consumed digest"} {
		saved := a.attestations["claim-c"]
		changed := saved
		use := *saved.Consumed
		switch mode {
		case "statement":
			changed.Statement = "Rewritten after the fact."
		case "consumed target":
			use.Target = "another-channel"
		case "consumed digest":
			use.TargetDigest = strings.Repeat("e", 64)
		}
		changed.Consumed = &use
		a.attestations["claim-c"] = changed
		f.code(t, mode, protectedSetupRead(f, reader, "attestations", c, nil), 409, string(commands.EvidenceStale))
		a.attestations["claim-c"] = saved
	}
}
