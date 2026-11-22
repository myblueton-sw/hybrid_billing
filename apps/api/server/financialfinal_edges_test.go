package server

import (
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"strconv"
	"sync"
	"testing"
)

func finalCommandWorld(t *testing.T, beforeFinance ...func(world)) (world, actor, actor, string) {
	t.Helper()
	w, digest := finalHTTPWorld(t, beforeFinance...)
	f := w.f
	provisionFinalHTTPRights(t, w)
	fields := financialAuthorityFields(f)
	fields["recipient"], fields["limit"] = SampleAdmin, "16"
	p := f.run(w.actors[Admin2Login], "erp_final_authority.propose", "command-gross-limit", fields)
	f.ok(t, "limit proposal", p, "pending", "")
	f.ok(t, "limit confirm", f.run(w.actors[Admin3Login], "erp_final_authority.confirm", "command-gross-limit", map[string]string{"digest": field(p, "digest")}), "confirmed", "")
	return f, w.actors[Admin3Login], w.actors[AdminLogin], digest
}

func finalRequest(f world, a actor, id, digest string) reply {
	return f.run(a, "erp_final.request", id, map[string]string{"result": "public-result", "digest": digest})
}
func TestFinancialFinalReplayRequiresCurrentPermission(t *testing.T) {
	f, requester, decider, digest := finalCommandWorld(t)
	body := chargeBody(f, "request-replay", "historical", map[string]string{"result": "public-result", "digest": digest})
	p := f.call("erp_final.request", body, requester.cookie, requester.csrf)
	f.ok(t, "request", p, approvals.StatePending, "")
	if got := f.call("erp_final.request", body, requester.cookie, requester.csrf); got.status != 200 || got.body["replay"] != true {
		t.Fatal(got.raw)
	}

	decision := chargeBody(f, "decision-replay", "historical", map[string]string{"digest": field(p, "digest")})
	f.ok(t, "approve", f.call("erp_final.approve", decision, decider.cookie, decider.csrf), approvals.StateApproved, "")
	if got := f.call("erp_final.approve", decision, decider.cookie, decider.csrf); got.status != 200 || got.body["replay"] != true {
		t.Fatal(got.raw)
	}
	for _, g := range f.s.admin.Grants() {
		if g.Terms.Kind == "erp_final_approval" && (g.SubjectID == SampleAdmin || g.SubjectID == SampleAdmin3) {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	f.code(t, "decision replay revoked", f.call("erp_final.approve", decision, decider.cookie, decider.csrf), 403, string(commands.PermissionDenied))
	f.code(t, "request replay revoked", f.call("erp_final.request", body, requester.cookie, requester.csrf), 403, string(commands.PermissionDenied))
	r := documentRead(f, decider, "erp_final.review", "historical", map[string]string{})
	if r.status != 200 || field(r, "recorded_state") != approvals.StateApproved || field(r, "current_eligibility") != "unchecked" {
		t.Fatal(r.raw)
	}
}

func TestFinancialFinalPreflightNeverSupersedes(t *testing.T) {
	f, requester, _, digest := finalCommandWorld(t)
	f.ok(t, "existing", finalRequest(f, requester, "existing", digest), approvals.StatePending, "")
	saved := f.s.cfg.Commands
	for _, mode := range []string{"fields", "value", "binding-capacity", "wire-amount"} {
		fields := map[string]string{"result": "public-result", "digest": digest}
		status, code := 501, commands.UnsupportedCapability
		switch mode {
		case "fields":
			f.s.cfg.Commands.Fields = 23
		case "value":
			f.s.cfg.Commands.Value = 1023
		case "binding-capacity":
			for i := len(f.s.financialFinals); i < approvalCap; i++ {
				f.s.financialFinals["capacity-"+approvalHash(string(rune(i)))] = financialFinalBinding{}
			}
			status, code = 422, commands.PrerequisiteFailed
		case "wire-amount":
			fields["amount"] = "1"
			status, code = 422, commands.InvalidRequest
		}
		f.code(t, mode, f.run(requester, "erp_final.request", "preflight", fields), status, string(code))
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

func TestFinancialFinalConcurrentApproveReject(t *testing.T) {
	f, requester, decider, digest := finalCommandWorld(t)
	p := finalRequest(f, requester, "competing", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	approve := chargeBody(f, "competing-approve", "competing", map[string]string{"digest": field(p, "digest")})
	reject := chargeBody(f, "competing-reject", "competing", map[string]string{"digest": field(p, "digest"), "reason": "other"})
	results := make(chan reply, 2)
	var wg sync.WaitGroup
	for command, body := range map[string]string{"erp_final.approve": approve, "erp_final.reject": reject} {
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

func TestFinancialFinalAllContributorsExcluded(t *testing.T) {
	w, digest := finalHTTPWorld(t)
	f := w.f
	provisionFinalHTTPRights(t, w)
	for _, recipient := range []string{SampleMember, SampleAdmin2, SampleAdmin3} {
		first, second := w.actors[AdminLogin], w.actors[Admin2Login]
		if recipient == SampleAdmin2 {
			second = w.actors[Admin3Login]
		}
		fields := financialPermissionFields(f)
		fields["recipient"] = recipient
		id := "excluded-final-" + recipient
		p := f.run(first, "erp_final_permission.propose", id, fields)
		f.ok(t, "contributor permission proposal", p, "pending", "")
		finalPermissionApprove(t, f, second, "erp_final_permission", id, p)
	}
	refreshResultActors(w)
	member := f.as(MemberLogin)
	p := finalRequest(f, w.actors[Admin3Login], "contributor-request", digest)
	f.ok(t, "request", p, "pending", "")
	for index, a := range []actor{member, w.actors[Admin2Login], w.actors[Admin3Login]} {
		for _, verdict := range []string{"approve", "reject"} {
			fields := map[string]string{"digest": field(p, "digest")}
			if verdict == "reject" {
				fields["reason"] = "other"
			}
			f.code(t, "contributor cannot decide", f.call("erp_final."+verdict, chargeBody(f, "exclude-"+verdict+"-"+strconv.Itoa(index), "contributor-request", fields), a.cookie, a.csrf), 403, "self_approval_denied")
		}
	}
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "contributor-request")
	f.must(err)
	if v.State != approvals.StatePending || len(v.Decisions) != 0 {
		t.Fatal("contributor recorded decision")
	}
}

func TestFinancialFinalBulkImpactFailureHasNoSnapshot(t *testing.T) {
	f, requester, decider, digest := finalCommandWorld(t)
	f.ok(t, "request", finalRequest(f, requester, "impact", digest), approvals.StatePending, "")
	staged := f.stage(decider, "", SampleAdmin3)
	f.ok(t, "stage", staged, "draft", "")
	before := f.s.Revision()
	f.s.fault = func(at string) bool { return at == "erp-final-impact" }
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
		if row["store"] == "erp_final_requests" {
			found = true
			if row["count"] != 1.0 || row["rechecked"] != false {
				t.Fatal(row)
			}
		}
	}
	if !found {
		t.Fatal("pending final request missing")
	}
}

func TestFinancialFinalReviewNeedsIndependentReadPermission(t *testing.T) {
	f, requester, decider, digest := finalCommandWorld(t)
	p := finalRequest(f, requester, "read-final", digest)
	f.ok(t, "request", p, "pending", "")
	read := func() reply { return documentRead(f, decider, "erp_final.review", "read-final", map[string]string{}) }
	r := read()
	if r.status != 200 || field(r, "amount") != "16" || field(r, "amount_basis") != financialFinalAmountProfile || field(r, "result_digest") != digest || field(r, "current_eligibility") != "unchecked" {
		t.Fatal(r.raw)
	}
	revoked := 0
	for _, grant := range f.s.admin.Grants() {
		if grant.SubjectID == SampleAdmin && grant.Terms.Action == gate.View && grant.Terms.Kind == approvalKind {
			f.must(f.s.admin.RevokeGrant(grant.ID, grant.Version))
			revoked++
		}
	}
	if revoked == 0 {
		t.Fatal("no read grants revoked")
	}
	f.code(t, "decision permission cannot substitute for view", read(), 403, "permission_denied")
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "read-final")
	f.must(err)
	if v.State != approvals.StatePending || len(v.Decisions) != 0 {
		t.Fatal("historical read mutated request")
	}
}
