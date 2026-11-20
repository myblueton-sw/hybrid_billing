package server

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
)

func chargeBody(f world, key, id string, fields map[string]string) string {
	b, _ := json.Marshal(map[string]any{"request_id": key, "target_id": id, "idempotency_key": key, "expected_revision": f.s.Revision(), "rationale": "Synthetic financial review", "body": fields})
	return string(b)
}

func TestFinancialChargeReplayRequiresCurrentPermission(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	body := chargeBody(f, "request-replay", "historical", map[string]string{"candidate": "subject-candidate", "digest": digest})
	p := f.call("financial_charge.request", body, requester.cookie, requester.csrf)
	f.ok(t, "request", p, approvals.StatePending, "")
	decision := chargeBody(f, "decision-replay", "historical", map[string]string{"digest": field(p, "digest")})
	f.ok(t, "approve", f.call("financial_charge.approve", decision, decider.cookie, decider.csrf), approvals.StateApproved, "")
	if got := f.call("financial_charge.approve", decision, decider.cookie, decider.csrf); got.status != 200 || got.body["replay"] != true {
		t.Fatal(got.raw)
	}
	for _, g := range f.s.admin.Grants() {
		if g.Terms.Kind == "charge_approval" && (g.SubjectID == SampleAdmin || g.SubjectID == SampleAdmin3) {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "decision replay revoked", f.call("financial_charge.approve", decision, decider.cookie, decider.csrf), 403, string(commands.PermissionDenied))
	f.code(t, "request replay revoked", f.call("financial_charge.request", body, requester.cookie, requester.csrf), 403, string(commands.PermissionDenied))
	r := chargeRead(f, decider, "historical")
	if r.status != 200 || field(r, "recorded_state") != approvals.StateApproved || field(r, "current_eligibility") != "unchecked" {
		t.Fatal(r.raw)
	}
}

func TestFinancialChargeStalePolicyAndSupersession(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "older", digest)
	f.ok(t, "older", p, approvals.StatePending, "")
	q := chargeRequest(f, requester, "newer", digest)
	f.ok(t, "newer", q, approvals.StatePending, "")
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "older")
	if err != nil || v.State != approvals.StateSuperseded {
		t.Fatal(v, err)
	}
	f.code(t, "superseded cannot decide", f.run(decider, "financial_charge.approve", "older", map[string]string{"digest": field(p, "digest")}), 409, string(commands.EvidenceStale))
	subjectPolicy(t, f, "new-policy", "3", "attribution,input,mapping,result,rule")
	f.code(t, "stale policy", f.run(f.as(AdminLogin), "financial_charge.approve", "newer", map[string]string{"digest": field(q, "digest")}), 409, string(commands.EvidenceStale))
}

func TestFinancialChargePreflightNeverSupersedes(t *testing.T) {
	f, requester, _, digest := chargeWorld(t)
	f.ok(t, "existing", chargeRequest(f, requester, "existing", digest), approvals.StatePending, "")
	saved := f.s.cfg.Commands
	for _, mode := range []string{"fields", "value", "binding-capacity", "wire-amount"} {
		fields := map[string]string{"candidate": "subject-candidate", "digest": digest}
		status, code := 501, commands.UnsupportedCapability
		switch mode {
		case "fields":
			f.s.cfg.Commands.Fields = 23
		case "value":
			f.s.cfg.Commands.Value = 1023
		case "binding-capacity":
			for i := len(f.s.financialCharges); i < approvalCap; i++ {
				f.s.financialCharges["capacity-"+approvalHash(string(rune(i)))] = financialChargeBinding{}
			}
			status, code = 422, commands.PrerequisiteFailed
		case "wire-amount":
			fields["amount"] = "1"
			status, code = 422, commands.InvalidRequest
		}
		f.code(t, mode, f.run(requester, "financial_charge.request", "preflight", fields), status, string(code))
		f.s.cfg.Commands = saved
		v, err := f.s.approvals.RequestOf(SampleWorkspace, "existing")
		if err != nil || v.State != approvals.StatePending {
			t.Fatal("refusal superseded old request", v, err)
		}
		if _, err = f.s.approvals.RequestOf(SampleWorkspace, "preflight"); err == nil {
			t.Fatal("partial request effect")
		}
	}
}

func TestFinancialChargeRejectNeedsPermissionButNotAmountAuthority(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "rejection", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	record, err := f.s.financialAuthorityRecord(SampleWorkspace, "charge-limit")
	f.must(err)
	f.ok(t, "revoke limit", f.run(f.as(Admin2Login), "financial_authority.revoke", "charge-limit", map[string]string{"digest": record.proposal.Digest}), "revoked", "")
	f.code(t, "approve without amount authority", f.run(decider, "financial_charge.approve", "rejection", map[string]string{"digest": field(p, "digest")}), 403, string(commands.PermissionDenied))
	f.code(t, "closed reason", f.run(decider, "financial_charge.reject", "rejection", map[string]string{"digest": field(p, "digest"), "reason": "invented"}), 422, string(commands.InvalidRequest))
	f.ok(t, "reject", f.run(decider, "financial_charge.reject", "rejection", map[string]string{"digest": field(p, "digest"), "reason": "incorrect_amount"}), approvals.StateRejected, "")
}

func TestFinancialChargeConcurrentRequestRevision(t *testing.T) {
	f, requester, _, digest := chargeWorld(t)
	left := chargeBody(f, "concurrent-one", "concurrent-one", map[string]string{"candidate": "subject-candidate", "digest": digest})
	right := chargeBody(f, "concurrent-two", "concurrent-two", map[string]string{"candidate": "subject-candidate", "digest": digest})
	results := make(chan reply, 2)
	var wg sync.WaitGroup
	for _, body := range []string{left, right} {
		wg.Add(1)
		go func(body string) {
			defer wg.Done()
			results <- f.call("financial_charge.request", body, requester.cookie, requester.csrf)
		}(body)
	}
	wg.Wait()
	close(results)
	pass, conflict := 0, 0
	for r := range results {
		if r.status == 200 {
			pass++
		} else if r.status == 409 && r.body["code"] == string(commands.RevisionConflict) {
			conflict++
		} else {
			t.Fatal(r.raw)
		}
	}
	if pass != 1 || conflict != 1 || len(f.s.financialCharges) != 1 {
		t.Fatal(pass, conflict, len(f.s.financialCharges))
	}
}

func TestFinancialChargeBoundaryDoesNotAlterOtherCommandClients(t *testing.T) {
	f := setup(t, SeedSample).ready()
	for _, command := range []string{"financial_permission.approve", "financial_request_permission.approve", "access.approve_request", "financial_charge.approve.extra", "financial_charge.unknown"} {
		call := commands.Call{Client: ConsoleClient}
		if code := f.s.financialChargeBoundary(command, &call); code != "" || call.Client != ConsoleClient {
			t.Fatal(command, code, call.Client)
		}
	}
	if f.s.gate.Describes(gate.Approve, "charge_approval", nil) != true {
		t.Fatal("missing fixed kind")
	}
}

func TestFinancialChargeConcurrentApproveReject(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	p := chargeRequest(f, requester, "competing", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	approve := chargeBody(f, "competing-approve", "competing", map[string]string{"digest": field(p, "digest")})
	reject := chargeBody(f, "competing-reject", "competing", map[string]string{"digest": field(p, "digest"), "reason": "other"})
	results := make(chan reply, 2)
	var wg sync.WaitGroup
	for command, body := range map[string]string{"financial_charge.approve": approve, "financial_charge.reject": reject} {
		wg.Add(1)
		go func(command, body string) {
			defer wg.Done()
			results <- f.call(command, body, decider.cookie, decider.csrf)
		}(command, body)
	}
	wg.Wait()
	close(results)
	pass, conflict := 0, 0
	for r := range results {
		if r.status == 200 {
			pass++
		} else if r.status == 409 && r.body["code"] == string(commands.RevisionConflict) {
			conflict++
		} else {
			t.Fatal(r.raw)
		}
	}
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "competing")
	if pass != 1 || conflict != 1 || err != nil || len(v.Decisions) != 1 {
		t.Fatal(pass, conflict, v, err)
	}
}

func TestFinancialChargeBulkImpactFailureHasNoSnapshot(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	f.ok(t, "request", chargeRequest(f, requester, "impact", digest), approvals.StatePending, "")
	staged := f.stage(decider, "", SampleAdmin3)
	f.ok(t, "stage", staged, "draft", "")
	before := f.s.Revision()
	f.s.fault = func(at string) bool { return at == "financial-charge-impact" }
	f.code(t, "prepare unavailable", f.prepare(decider, field(staged, "draft"), "suspend_membership"), 503, string(commands.DependencyUnavailable))
	if f.s.Revision() != before {
		t.Fatal("failure mutated revision")
	}
	f.s.fault = nil
	p := f.prepare(decider, field(staged, "draft"), "suspend_membership")
	f.ok(t, "prepare", p, "prepared", "")
	if field(p, "bulk") != "bulk-revocation-1" {
		t.Fatal("failure consumed snapshot", p.raw)
	}
	rows := f.bulkRead(decider, "/"+field(p, "bulk")).body["impact"].(map[string]any)["pending"].([]any)
	found := false
	for _, raw := range rows {
		row := raw.(map[string]any)
		if row["store"] == "financial_charge_requests" {
			found = true
			if row["count"] != 1.0 || row["rechecked"] != false {
				t.Fatal(row)
			}
		}
	}
	if !found {
		t.Fatal("pending charge missing")
	}
}
