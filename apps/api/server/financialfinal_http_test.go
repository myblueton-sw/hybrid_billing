package server

import (
	"reflect"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func finalHTTPWorld(t *testing.T) (documentWorld, string) {
	t.Helper()
	w, _, digest := publicResult(t)
	f := w.f
	f.ok(t, "apply result", f.run(w.actors[Admin2Login], "erp_result.apply", "public-result", map[string]string{"digest": digest}), "recorded", "")
	p := f.run(w.actors[AdminLogin], "erp_final_policy.propose", "http-final-policy", financialFields("2"))
	f.ok(t, "final policy proposal", p, "pending", "")
	f.ok(t, "final policy confirm", f.run(w.actors[Admin2Login], "erp_final_policy.confirm", "http-final-policy", map[string]string{"digest": field(p, "digest")}), "confirmed", "")
	return w, digest
}

func provisionFinalHTTPRights(t *testing.T, w documentWorld) {
	t.Helper()
	f := w.f
	first, second := w.actors[AdminLogin], w.actors[Admin2Login]
	p := f.run(first, "erp_final_request_permission.propose_console", "http-final-console", requestPermissionTimes(f))
	f.ok(t, "final console proposal", p, "pending", "")
	ceiling := finalPermissionApprove(t, f, second, "erp_final_request_permission", "http-final-console", p)
	fields := requestPermissionTimes(f)
	fields["recipient"], fields["ceiling"] = SampleAdmin3, field(ceiling, "grant")
	p = f.run(first, "erp_final_request_permission.propose_human", "http-final-requester", fields)
	f.ok(t, "final requester proposal", p, "pending", "")
	finalPermissionApprove(t, f, second, "erp_final_request_permission", "http-final-requester", p)
	first, second = w.actors[Admin2Login], w.actors[Admin3Login]
	fields = financialPermissionFields(f)
	fields["recipient"] = SampleAdmin
	p = f.run(first, "erp_final_permission.propose", "http-final-decider", fields)
	f.ok(t, "final decider proposal", p, "pending", "")
	finalPermissionApprove(t, f, second, "erp_final_permission", "http-final-decider", p)
	refreshResultActors(w)
}

func TestFinancialFinalHTTPRejectionWithoutAmountCoverage(t *testing.T) {
	w, digest := finalHTTPWorld(t)
	f := w.f
	body := map[string]string{"result": "public-result", "digest": digest}
	f.code(t, "charge request grant cannot substitute", f.run(w.actors[Admin3Login], "erp_final.request", "http-final-request", body), 403, "permission_denied")
	provisionFinalHTTPRights(t, w)
	requested := f.run(w.actors[Admin3Login], "erp_final.request", "http-final-request", body)
	f.ok(t, "final request", requested, approvals.StatePending, "")
	fields := map[string]string{"digest": field(requested, "digest")}
	f.code(t, "approval needs full gross authority", f.run(w.actors[AdminLogin], "erp_final.approve", "http-final-request", fields), 422, "prerequisite_failed")
	fields["reason"] = "other"
	f.ok(t, "rejection does not require amount coverage", f.run(w.actors[AdminLogin], "erp_final.reject", "http-final-request", fields), approvals.StateRejected, "")
}

func TestFinancialFinalHTTPFullGrossApprovalAndCurrent(t *testing.T) {
	w, digest := finalHTTPWorld(t)
	f := w.f
	provisionFinalHTTPRights(t, w)
	requested := f.run(w.actors[Admin3Login], "erp_final.request", "gross-final-request", map[string]string{"result": "public-result", "digest": digest})
	f.ok(t, "final request", requested, approvals.StatePending, "")
	amountFields := financialAuthorityFields(f)
	amountFields["recipient"], amountFields["limit"] = SampleAdmin, "15"
	p := f.run(w.actors[Admin2Login], "erp_final_authority.propose", "net-only-limit", amountFields)
	f.ok(t, "net-only limit proposal", p, "pending", "")
	f.ok(t, "net-only limit confirm", f.run(w.actors[Admin3Login], "erp_final_authority.confirm", "net-only-limit", map[string]string{"digest": field(p, "digest")}), "confirmed", "")
	decision := map[string]string{"digest": field(requested, "digest")}
	f.code(t, "net authority does not cover gross", f.run(w.actors[AdminLogin], "erp_final.approve", "gross-final-request", decision), 422, "prerequisite_failed")
	amountFields["limit"] = "16"
	p = f.run(w.actors[Admin2Login], "erp_final_authority.propose", "gross-limit", amountFields)
	f.ok(t, "gross limit proposal", p, "pending", "")
	f.ok(t, "gross limit confirm", f.run(w.actors[Admin3Login], "erp_final_authority.confirm", "gross-limit", map[string]string{"digest": field(p, "digest")}), "confirmed", "")
	f.ok(t, "approve exact gross", f.run(w.actors[AdminLogin], "erp_final.approve", "gross-final-request", decision), approvals.StateApproved, "")
	f.must(inputAuthorityAs(f, w.actors[Admin3Login], func() error {
		subject, err := f.s.currentFinancialFinalLocked(SampleWorkspace, "gross-final-request", field(requested, "digest"), SampleAdmin3)
		if err == nil && subject.Amount != "16" {
			t.Fatal("current consumer lost gross")
		}
		return err
	}))
	f.ok(t, "revoke gross limit", f.run(w.actors[Admin2Login], "erp_final_authority.revoke", "gross-limit", map[string]string{"digest": field(p, "digest")}), "revoked", "")
	err := inputAuthorityAs(f, w.actors[Admin3Login], func() error {
		_, err := f.s.currentFinancialFinalLocked(SampleWorkspace, "gross-final-request", field(requested, "digest"), SampleAdmin3)
		return err
	})
	if err == nil {
		t.Fatal("revoked gross authority remained current")
	}
	revoked := 0
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == SampleAdmin && grant.Terms.Kind == "erp_final_approval" && grant.Terms.Action == gate.Approve {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
			revoked++
		}
	}
	if revoked == 0 {
		t.Fatal("no final permission revoked")
	}
	f.s.documentERP = nil
	err = inputAuthorityAs(f, w.actors[Admin3Login], func() error {
		_, err := f.s.currentFinancialFinalLocked(SampleWorkspace, "gross-final-request", field(requested, "digest"), SampleAdmin3)
		return err
	})
	documentFailure(t, err, commands.PermissionDenied)

}

func TestFinancialFinalHTTPHeldRejectionPreservesLedger(t *testing.T) {
	w, digest := finalHTTPWorld(t)
	f := w.f
	provisionFinalHTTPRights(t, w)
	p := f.run(w.actors[Admin3Login], "erp_final.request", "held-final-request", map[string]string{"result": "public-result", "digest": digest})
	f.ok(t, "request", p, "pending", "")
	before, err := w.ledger.ResultOf(SampleWorkspace, "result-preparation")
	f.must(err)
	f.s.enterHold("final-rejection-test")
	prepared := documentRead(f, w.actors[AdminLogin], "erp_final.prepare", "held-final-request", map[string]string{"digest": field(p, "digest")})
	if prepared.status != 200 || field(prepared, "approve") != "hold_active" || field(prepared, "reject") != "eligible" {
		t.Fatalf("held preparation: %s", prepared.raw)
	}
	f.ok(t, "reject during hold", f.run(w.actors[AdminLogin], "erp_final.reject", "held-final-request", map[string]string{"digest": field(p, "digest"), "reason": "other"}), "rejected", "")
	after, err := w.ledger.ResultOf(SampleWorkspace, "result-preparation")
	f.must(err)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("held rejection changed ledger")
	}
}
