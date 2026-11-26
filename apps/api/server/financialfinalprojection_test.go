package server

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
)

func TestFinancialFinalMixedProjectionCapacity(t *testing.T) {
	f, requester, _, digest := finalCommandWorld(t)
	initial := f.s.financialApprovalCount()
	if initial < 2 || len(f.s.approvalIndex) == 0 || len(f.s.financialCharges) == 0 {
		t.Fatal("missing mixed sources")
	}
	for i := initial; i < approvalCap; i++ {
		if (i-initial)%5 == 0 {
			f.advance(time.Minute)
			requester = f.as(Admin3Login)
		}
		p := finalRequest(f, requester, "mixed-final-"+strconv.Itoa(i), digest)
		f.ok(t, "final request", p, "pending", "")
	}
	rows, err := f.s.approvalRows(SampleWorkspace)
	f.must(err)
	if len(rows) != approvalCap {
		t.Fatalf("rows=%d", len(rows))
	}
	governed, seeded, charge := 0, 0, 0
	for _, row := range rows {
		switch row.Amounts[0].Basis {
		case financialFinalAmountProfile:
			governed++
			if row.Kind != "final_erp_amount" || row.Evidence.Blockers[0].Code != "current_checks_unverified" || *row.Amounts[0].Candidate != "16" || row.Authority.Limit != "unknown" {
				t.Fatal("final projection changed meaning")
			}
		case "gross":
			seeded++
			if row.Status == "pending" {
				t.Fatal("unbound seed became actionable")
			}
		case "rated_candidate_total":
			charge++
		}
	}
	if governed != approvalCap-initial || seeded == 0 || charge == 0 {
		t.Fatal("mixed projection omitted source")
	}
	f.code(t, "combined capacity refuses final", finalRequest(f, requester, "mixed-overflow", digest), 422, "prerequisite_failed")
	f.code(t, "combined capacity refuses charge", chargeRequest(f, requester, "mixed-charge-overflow", f.s.financialCharges["document-charge"].CandidateDigest), 422, "prerequisite_failed")
	for _, id := range []string{"mixed-overflow", "mixed-charge-overflow"} {
		_, err := f.s.approvals.RequestOf(SampleWorkspace, id)
		if !errors.Is(err, approvals.ErrNotFound) {
			t.Fatal("capacity refusal recorded request")
		}
	}
	last, err := f.s.approvals.RequestOf(SampleWorkspace, "mixed-final-"+strconv.Itoa(approvalCap-1))
	f.must(err)
	if last.State != approvals.StatePending {
		t.Fatal("capacity refusal superseded request")
	}
}
