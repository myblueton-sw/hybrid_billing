package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
)

func TestApprovalReadSupersededDecisionRetainsHistoricalPolicy(t *testing.T) {
	f := setup(t, SeedSample).ready()
	approvalAdminister(t, f.s, approvals.Proposal{ID: "history-authority", Kind: approvals.ProposeGrant, Authority: approvals.Authority{ID: "history-grant", Approver: SampleAdmin3, Stage: approvals.StageCharge, Currency: "USD", Limit: "5000", From: f.s.clock().Format(time.DateOnly)}})
	decision, err := f.s.approvals.Decide(SampleWorkspace, "sample-approval-001", approvals.Decision{Verdict: approvals.VerdictApprove}, approvalHuman(SampleAdmin3))
	if err != nil {
		t.Fatal(err)
	}
	f.advance(time.Minute)
	approvalAdminister(t, f.s, approvals.Proposal{ID: "history-policy", Kind: approvals.ProposePolicy, Policy: approvals.Policy{Stage: approvals.StageCharge, Revision: 2, Approvals: 2, Required: []string{"input", "result"}, Validity: time.Hour}})
	got := f.approvalRead(f.as(AdminLogin), "view=history&q=sample-approval-001")
	if got.status != 200 {
		t.Fatal(got.raw)
	}
	row := got.body["records"].([]any)[0].(map[string]any)
	if row["status"] != "superseded" || row["approver"].(map[string]any)["id"] != SampleAdmin3 || row["decided_at"] != stamp(decision.Decisions[0].At) || row["updated_at"] != stamp(f.s.clock()) || row["authority"].(map[string]any)["limit"] != "unknown" {
		t.Fatal(row)
	}
	if row["evidence"].(map[string]any)["dependencies"] != "unknown" || row["amounts"].([]any)[0].(map[string]any)["candidate"] != "1234.56789012" {
		t.Fatal("history altered evidence", row)
	}
	rows, err := f.s.approvalRows(SampleWorkspace)
	if err != nil || len(rows) != 30 || rows[2].Status != "superseded" || rows[2].Approver != nil || rows[29].Status != "rejected" {
		t.Fatal("other histories changed", err)
	}
	old, err := f.s.approvals.PolicyAt(SampleWorkspace, approvals.StageCharge, 1)
	if err != nil || old.Approvals != 1 {
		t.Fatal(old, err)
	}
	current, err := f.s.approvals.PolicyOf(SampleWorkspace, approvals.StageCharge)
	if err != nil || current.Revision != 2 || current.Approvals != 2 {
		t.Fatal(current, err)
	}
}
