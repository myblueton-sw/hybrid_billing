package server

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/protected"
)

func protectedSetupRead(f world, who actor, command string, c financialIssuanceConfiguration, extra map[string]string) reply {
	fields := map[string]string{"configuration": c.Spec.ID, "digest": c.Digest}
	for name, value := range extra {
		fields[name] = value
	}
	return documentRead(f, who, "issuance_setup."+command, c.Spec.Contract, fields)
}

func protectedSetupRow(t *testing.T, got reply, n int) map[string]any {
	t.Helper()
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	var row map[string]any
	if err := json.Unmarshal([]byte(field(got, fmt.Sprintf("row_%02d", n))), &row); err != nil {
		t.Fatal(err, got.raw)
	}
	return row
}

type protectedSetupState struct {
	rules    *financialProtectedRules
	channels map[string]financialChannelRecord
	changes  map[string]financialChangeRecord
	orders   map[string][]string
	module   protected.RulesState
	stored   map[string]protected.View
	revision string
}

func captureProtectedSetup(f world) protectedSetupState {
	a := f.s.financialIssuance
	s := protectedSetupState{channels: map[string]financialChannelRecord{}, changes: map[string]financialChangeRecord{}, orders: map[string][]string{}, stored: map[string]protected.View{}, revision: f.s.Revision()}
	if a.protectedRules != nil {
		p := *a.protectedRules
		s.rules = &p
	}
	for id, p := range a.channels {
		s.channels[id] = copyChannelRecord(p)
	}
	for id, p := range a.changes {
		s.changes[id] = copyChange(p)
		if v, err := a.fields.Peek(p.Workspace, id); err == nil {
			s.stored[id] = v
		}
	}
	for key, ids := range a.changeOrders {
		s.orders[key] = append([]string(nil), ids...)
	}
	s.module, _ = a.fields.RulesOf(SampleWorkspace)
	return s
}

func TestFinancialSetupProtectedCompletedHistory(t *testing.T) {
	w := configuredSetup(t)
	f := w.f
	grantSetupInspectionForTest(f, true)
	reader := f.as(MemberLogin)
	changes, _ := completeConfiguredSetup(t, w)
	c := w.configuration
	before := captureProtectedSetup(f)

	got := protectedSetupRead(f, reader, "protected", c, nil)
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	fields, _ := got.body["fields"].(map[string]any)
	if len(fields) != 39 {
		t.Fatalf("field count %d", len(fields))
	}
	rules := f.s.financialIssuance.protectedRules
	want := map[string]string{
		"schema": "hb_issuance_protected_setup_v1", "contract": c.Spec.Contract, "configuration": c.Spec.ID, "digest": c.Digest, "current_eligibility": "unchecked",
		"rules_state": "confirmed", "rules_observation": "active", "rules_id": rules.ID, "rules_digest": rules.Digest, "rules_approvals": "1", "rules_channel_age_seconds": "86400",
		"rules_proposer": SampleAdmin, "rules_confirmer": SampleAdmin2, "channels": "3", "channels_other_provenance": "0", "channels_unowned_module": "0",
		"recipient_subject_kind": "customer", "template_subject_kind": "document_kind", "account_subject_kind": "entity",
	}
	for _, prefix := range []string{"recipient", "template", "account"} {
		want[prefix+"_changes"], want[prefix+"_other_provenance"], want[prefix+"_unowned_module"] = "1", "0", "0"
	}
	want["recipient_latest_scheduled"], want["template_latest_scheduled"], want["account_latest_scheduled"] = changes[protected.FieldDocumentRecipient].Spec.ID, changes[protected.FieldDocumentTemplate].Spec.ID, changes[protected.FieldPaymentAccount].Spec.ID
	want["template_latest_scheduled_effective"] = setupInstant(changes[protected.FieldDocumentTemplate].Spec.Effective)
	for name, value := range want {
		if field(got, name) != value {
			t.Fatalf("%s = %q, want %q", name, field(got, name), value)
		}
	}
	if field(got, "rules_proposed_at") == "" || field(got, "rules_confirmed_at") == "" {
		t.Fatal("rules instants missing")
	}

	page := protectedSetupRead(f, reader, "protected_channels", c, nil)
	if field(page, "count") != "3" || field(page, "next") != "" || field(page, "row_03") != "" || field(page, "current_eligibility") != "unchecked" || field(page, "schema") != "hb_issuance_protected_channel_page_v1" {
		t.Fatal(page.raw)
	}
	kinds := map[string]bool{}
	for i := 0; i < 3; i++ {
		row := protectedSetupRow(t, page, i)
		p := f.s.financialIssuance.channels[row["channel"].(string)]
		kinds[row["subject_kind"].(string)] = true
		if row["state"] != "admitted" || row["observation"] != "admitted" || row["rules"] != "current" || row["digest"] != p.Digest || row["registrar"] != p.Registrar || row["admitter"] != p.Admitter || row["request_not_before"] != setupInstant(p.Module.Admitted.Add(24*time.Hour)) || row["retired_at"] != "" {
			t.Fatalf("channel row %v", row)
		}
		for _, hidden := range []string{p.Spec.Ref.ID, p.Spec.Ref.Fingerprint, p.Spec.Evidence, p.Admission, p.Spec.Reason, p.Subject.Subject} {
			if hidden != "" && strings.Contains(page.raw, hidden) {
				t.Fatalf("channel page disclosed %q", hidden)
			}
		}
	}
	if len(kinds) != 3 || !kinds["customer"] || !kinds["entity"] || !kinds["document_kind"] {
		t.Fatalf("subject kinds %v", kinds)
	}

	for fieldName, p := range changes {
		trail := protectedSetupRead(f, reader, "protected_changes", c, map[string]string{"field": fieldName})
		row := protectedSetupRow(t, trail, 0)
		decisions, _ := row["decisions"].([]any)
		if field(trail, "count") != "1" || field(trail, "field") != fieldName || row["change"] != p.Spec.ID || row["phase"] != "recorded" || row["state"] != protected.StateScheduled || row["observation"] != protected.StateScheduled || row["requester"] != p.Requester || row["verifier"] != p.Verification.Actor || row["verification_outcome"] != protected.OutcomeConfirmed || row["approvals_required"] != "1" || len(decisions) != 1 || row["pending_decision_actor"] != "" || row["pending_verifier"] != "" || row["channel"] != p.Spec.Channel || row["effective"] != setupInstant(p.Spec.Effective) {
			t.Fatalf("change row %v", row)
		}
		if d := decisions[0].(map[string]any); d["approver"] != p.Decisions[0].Actor || d["verdict"] != protected.VerdictApprove || d["at"] == "" {
			t.Fatalf("decision %v", d)
		}
		for _, hidden := range []string{p.Spec.After.ID, p.Spec.After.Fingerprint, p.Spec.Instruction.Evidence, p.Verification.Evidence, p.Module.Digest, p.Module.Verification.Digest, p.Spec.Reason, p.Channel.Subject.Subject} {
			if hidden != "" && strings.Contains(trail.raw, hidden) {
				t.Fatalf("change page disclosed %q", hidden)
			}
		}
	}
	if after := captureProtectedSetup(f); !reflect.DeepEqual(before, after) {
		t.Fatal("protected setup read changed state")
	}
	f.code(t, "unknown field", protectedSetupRead(f, reader, "protected_changes", c, map[string]string{"field": protected.FieldTaxIdentifier}), 422, string(commands.InvalidRequest))
}

func grantSetupInspectionAs(f world, recipient string) {
	spec := protectedPermissionSpec(f, "inspect")
	spec.Kind, spec.Recipient = inputauthority.Human, recipient
	for id, ceiling := range f.s.protectedAuthority.ceilings {
		if ceiling.Capability == spec.Capability {
			spec.CeilingID = id
		}
	}
	p, err := protectedPermissionProposal(f, f.as(AdminLogin), inputauthority.Proposal{ID: "protected-reader-" + recipient, Spec: spec, Rationale: "Separate authorized protected setup viewer"})
	f.must(err)
	_, err = protectedPermissionDecision(f, f.as(Admin2Login), p)
	f.must(err)
}
