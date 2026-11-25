package server

import (
	"net/url"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
)

func projectionRow(t *testing.T, f world, id string) approvalRow {
	t.Helper()
	rows, err := f.s.approvalRows(SampleWorkspace)
	f.must(err)
	for _, row := range rows {
		if row.ID == id {
			return row
		}
	}
	t.Fatal("missing projected row", id)
	return approvalRow{}
}

func TestApprovalProjectionMixedServiceAndSampleHistory(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "projected-request", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	row := projectionRow(t, f, "projected-request")
	if row.Status != "pending" || row.Author.ID != SampleMember || row.Requester.ID != SampleAdmin3 || row.Policy != 2 || *row.Evidence.Revision != field(p, "digest") || row.Amounts[0].Basis != "rated_candidate_total" || *row.Amounts[0].Candidate != "15" || row.Evidence.Blockers[0].Code != "current_checks_unverified" {
		t.Fatal(row)
	}
	if row.Authority.Limit != "unknown" || row.Deadline != nil || row.ERP.Applicability != "unknown" {
		t.Fatal("inferred current eligibility", row)
	}
	sample := projectionRow(t, f, "sample-approval-001")
	if sample.Amounts[0].Basis != "gross" || sample.Evidence.Blockers[0].Code != "producer_unbound" || sample.Policy != 1 {
		t.Fatal(sample)
	}
	f.ok(t, "approve", f.run(decider, "financial_charge.approve", "projected-request", map[string]string{"digest": field(p, "digest")}), approvals.StateApproved, "")
	row = projectionRow(t, f, "projected-request")
	if row.Status != "approved" || row.Approver.ID != SampleAdmin || row.Decided == nil || row.Authority.Limit != "unknown" {
		t.Fatal(row)
	}
	q := chargeRequest(f, requester, "replacement-request", digest)
	f.ok(t, "replacement", q, approvals.StatePending, "")
	row = projectionRow(t, f, "projected-request")
	if row.Status != "superseded" || row.Approver.ID != SampleAdmin || row.Decided == nil {
		t.Fatal(row)
	}
	f.ok(t, "reject", f.run(decider, "financial_charge.reject", "replacement-request", map[string]string{"digest": field(q, "digest"), "reason": "other"}), approvals.StateRejected, "")
	if row := projectionRow(t, f, "replacement-request"); row.Status != "rejected" || row.Approver.ID != SampleAdmin {
		t.Fatal(row)
	}
}

func TestApprovalProjectionUnionRefusesCorruptionAndBounds(t *testing.T) {
	for _, mode := range []string{"duplicate", "corrupt", "capacity", "foreign"} {
		t.Run(mode, func(t *testing.T) {
			f, requester, _, digest := chargeWorld(t)
			f.ok(t, "request", chargeRequest(f, requester, "union", digest), approvals.StatePending, "")
			b := f.s.financialCharges["union"]
			switch mode {
			case "duplicate":
				f.s.approvalIndex = append(f.s.approvalIndex, approvalIndex{Workspace: SampleWorkspace, ID: "union", Target: b.Subject.Target, Basis: "gross"})
			case "corrupt":
				b.Digest = "bad"
				f.s.financialCharges["union"] = b
			case "capacity":
				for len(f.s.approvalIndex) < approvalCap {
					f.s.approvalIndex = append(f.s.approvalIndex, approvalIndex{Workspace: "foreign"})
				}
			case "foreign":
				b.Workspace = "foreign"
				f.s.financialCharges["union"] = b
			}
			rows, err := f.s.approvalRows(SampleWorkspace)
			if mode == "foreign" {
				if err != nil || len(rows) != 30 {
					t.Fatal(rows, err)
				}
				return
			}
			if err == nil || rows != nil {
				t.Fatal("partial or corrupt projection", rows, err)
			}
		})
	}
}

func TestApprovalProjectionSnapshotChangesWithActualMutations(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	read := func() reply {
		r := f.approvalRead(decider, "view=history")
		if r.status != 200 {
			t.Fatal(r.raw)
		}
		return r
	}
	old := read()
	p := chargeRequest(f, requester, "snapshot-request", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	current := read()
	if old.body["snapshot"] == current.body["snapshot"] {
		t.Fatal("request did not change snapshot")
	}
	q := url.Values{"view": {"history"}, "snapshot": {old.body["snapshot"].(string)}, "permission_revision": {old.body["permission_revision"].(string)}, "cursor": {old.body["next_cursor"].(string)}}
	if got := f.approvalRead(decider, q.Encode()); got.status != 409 {
		t.Fatal(got.raw)
	}
	f.ok(t, "decision", f.run(decider, "financial_charge.approve", "snapshot-request", map[string]string{"digest": field(p, "digest")}), approvals.StateApproved, "")
	if current.body["snapshot"] == read().body["snapshot"] {
		t.Fatal("decision did not change snapshot")
	}
}

func TestApprovalDecisionCapabilityIsExplicitAndBounded(t *testing.T) {
	f := setup(t, SeedSample).ready()
	if got := f.s.approvalCommandCapability(); got != "financial_approval_v2" {
		t.Fatal(got)
	}
	f.s.cfg.Commands.Fields = 23
	if got := f.s.approvalCommandCapability(); got != "unconnected" {
		t.Fatal(got)
	}
	f.s.cfg.Commands.Fields = 64
	f.s.cfg.Seed = SeedNone
	if got := f.s.approvalCommandCapability(); got != "unconnected" {
		t.Fatal(got)
	}
}
