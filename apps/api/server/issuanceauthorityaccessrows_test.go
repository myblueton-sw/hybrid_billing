package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/draftauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protectedauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/termsauthority"
)

func TestIssuanceGrantsAppearInAccessRows(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	want := map[string][2]string{}
	for index, right := range []struct{ namespace, capability, action, kind, field string }{
		{"issuance_protected_authority", protectedauthority.RegisterChannel, "register_issuance_channel", protectedauthority.ResourceKind, ""},
		{"issuance_protected_authority", protectedauthority.AdmitChannel, "admit_issuance_channel", protectedauthority.ResourceKind, ""},
		{"issuance_protected_authority", protectedauthority.RequestChange, "request_issuance_protected_change", protectedauthority.ResourceKind, ""},
		{"issuance_protected_authority", protectedauthority.VerifyChange, "verify_issuance_protected_change", protectedauthority.ResourceKind, ""},
		{"issuance_protected_authority", protectedauthority.ApproveChange, "approve_issuance_protected_change", protectedauthority.ResourceKind, ""},
		{"issuance_protected_authority", protectedauthority.Inspect, "view_cost", protectedauthority.ResourceKind, protectedauthority.EvidenceField},
		{"issuance_term_authority", termsauthority.RecordTerm, "record_issuance_term", termsauthority.ResourceKind, ""},
		{"issuance_term_authority", termsauthority.ApproveTerm, "approve_issuance_term", termsauthority.ResourceKind, ""},
		{"issuance_term_authority", termsauthority.BindTerm, "bind_issuance_term", termsauthority.ResourceKind, ""},
		{"issuance_term_authority", termsauthority.ApproveBinding, "approve_issuance_term_binding", termsauthority.ResourceKind, ""},
		{"issuance_term_authority", termsauthority.Inspect, "view_cost", termsauthority.ResourceKind, termsauthority.EvidenceField},
		{"issuance_draft_authority", draftauthority.Submit, "submit_issuance_draft", draftauthority.ResourceKind, ""},
		{"issuance_draft_authority", draftauthority.Approve, "approve_issuance_draft", draftauthority.ResourceKind, ""},
		{"issuance_draft_authority", draftauthority.Inspect, "view_cost", draftauthority.ResourceKind, draftauthority.EvidenceField},
	} {
		provisionIssuanceCapability(t, f, right.namespace, right.capability, SampleMember, "rows-"+itoa(index)+"-")
		want[right.action+"|"+right.kind] = [2]string{right.field, ""}
	}
	got := f.accessRead(f.as(AdminLogin), "")
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	rows, _ := got.body["grants"].([]any)
	seen := map[string]bool{}
	for _, raw := range rows {
		row := raw.(map[string]any)
		key := row["action"].(string) + "|" + row["kind"].(string)
		expected, ok := want[key]
		if !ok || row["subject"] != SampleMember {
			continue
		}
		fields, _ := row["fields"].([]any)
		if expected[0] == "" && len(fields) != 0 || expected[0] != "" && (len(fields) != 1 || fields[0] != expected[0]) {
			t.Fatalf("%s fields %v", key, fields)
		}
		seen[key] = true
	}
	if len(seen) != len(want) {
		t.Fatalf("issuance grants in access rows: %v of %d", seen, len(want))
	}
}
