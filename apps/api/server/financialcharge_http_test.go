package server

import (
	"encoding/json"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func chargeRequestGrant(t *testing.T, f world, recipient, id, ceiling string) string {
	t.Helper()
	fields := requestPermissionTimes(f)
	fields["recipient"], fields["ceiling"] = recipient, ceiling
	p := f.run(f.as(AdminLogin), "financial_request_permission.propose_human", id, fields)
	f.ok(t, "request permission proposal", p, inputauthority.Pending, "")
	c := requestPermissionApprove(f, f.as(Admin2Login), id, p)
	f.ok(t, "request permission approval", c, inputauthority.Applied, "")
	return field(c, "grant")
}

func chargeDecisionGrant(t *testing.T, f world, recipient, id string) string {
	t.Helper()
	fields := financialPermissionFields(f)
	fields["recipient"] = recipient
	p := f.run(f.as(Admin2Login), "financial_permission.propose", id, fields)
	f.ok(t, "decision permission proposal", p, inputauthority.Pending, "")
	c := f.run(f.as(Admin3Login), "financial_permission.approve", id, map[string]string{"digest": field(p, "digest")})
	f.ok(t, "decision permission approval", c, inputauthority.Applied, "")
	return field(c, "grant")
}

func chargeWorld(t *testing.T, beforeFinance ...func(world)) (world, actor, actor, string) {
	t.Helper()
	f, _, _, _, digest := subjectWorld(t, beforeFinance...)
	approvalAdminister(t, f.s, approvals.Proposal{ID: "charge-governor", Kind: approvals.ProposeAddAdmin, Admin: SampleAdmin3})
	ceiling := provisionRequestCeiling(t, f)
	chargeRequestGrant(t, f, SampleAdmin3, "requester", field(ceiling, "grant"))
	chargeDecisionGrant(t, f, SampleAdmin, "decider")
	fields := financialAuthorityFields(f)
	fields["recipient"], fields["limit"] = SampleAdmin, "15"
	p := f.run(f.as(Admin2Login), "financial_authority.propose", "charge-limit", fields)
	f.ok(t, "limit proposal", p, approvals.StatePending, "")
	f.ok(t, "limit confirm", f.run(f.as(Admin3Login), "financial_authority.confirm", "charge-limit", map[string]string{"digest": field(p, "digest")}), approvals.StateConfirmed, "")
	return f, f.as(Admin3Login), f.as(AdminLogin), digest
}

func chargeRead(f world, a actor, id string) reply {
	body, _ := json.Marshal(map[string]any{"request_id": "charge-review", "target_id": id, "body": map[string]string{}})
	return f.call("financial_charge.review", string(body), a.cookie, a.csrf)
}

func chargeRequest(f world, a actor, id, digest string) reply {
	return f.runWith(a, f.s.Revision(), "financial_charge.request", id, map[string]string{"candidate": "subject-candidate", "digest": digest})
}

func TestFinancialChargeActualRequestReviewAndDecision(t *testing.T) {
	f, requester, decider, digest := chargeWorld(t)
	before := len(f.s.approvalIndex)
	p := chargeRequest(f, requester, "charge-one", digest)
	f.ok(t, "actual request", p, approvals.StatePending, "")
	if field(p, "amount") != "" || field(p, "receipt") != "historical_decision" {
		t.Fatal(p.raw)
	}
	r := chargeRead(f, decider, "charge-one")
	if r.status != 200 || field(r, "amount") != "15" || field(r, "currency") != "USD" || field(r, "author") != SampleMember || field(r, "requester") != SampleAdmin3 || field(r, "amount_basis") != "rated_candidate_total" || field(r, "receipt") != "historical_evidence" || field(r, "current_eligibility") != "unchecked" {
		t.Fatal(r.raw)
	}
	f.code(t, "requester cannot decide", f.run(requester, "financial_charge.approve", "charge-one", map[string]string{"digest": field(p, "digest")}), 403, string(commands.PermissionDenied))
	c := f.run(decider, "financial_charge.approve", "charge-one", map[string]string{"digest": field(p, "digest")})
	f.ok(t, "actual decision", c, approvals.StateApproved, "")
	v, err := f.s.approvals.RequestOf(SampleWorkspace, "charge-one")
	if err != nil || len(v.Decisions) != 1 || v.Decisions[0].Actor != SampleAdmin || v.Subject.Author != SampleMember {
		t.Fatal(v, err)
	}
	if len(f.s.approvalIndex) != before {
		t.Fatal("unproven gross basis indexed")
	}
	f.code(t, "fresh decision cannot repeat", f.run(decider, "financial_charge.approve", "charge-one", map[string]string{"digest": field(p, "digest")}), 409, string(commands.EvidenceStale))
	f.code(t, "request cannot revive", chargeRequest(f, requester, "charge-one", digest), 409, string(commands.EvidenceStale))
}

func TestFinancialChargeNoDefaultOrReadInference(t *testing.T) {
	f, maker, _, _, digest := subjectWorld(t)
	f.code(t, "no default request", chargeRequest(f, maker, "none", digest), 403, string(commands.PermissionDenied))
	ceiling := provisionRequestCeiling(t, f)
	chargeRequestGrant(t, f, SampleMember, "request-member", field(ceiling, "grant"))
	p := chargeRequest(f, maker, "member-request", digest)
	f.ok(t, "member request", p, approvals.StatePending, "")
	f.code(t, "request does not grant read", chargeRead(f, maker, "member-request"), 403, string(commands.PermissionDenied))
	for _, id := range []string{"member-request", "missing", "sample-approval-001"} {
		f.code(t, "no default approval", f.run(f.as(AdminLogin), "financial_charge.approve", id, map[string]string{"digest": field(p, "digest")}), 403, string(commands.PermissionDenied))
	}
}

func TestFinancialChargeSelfApprovalThroughAnotherRequester(t *testing.T) {
	f, requester, _, digest := chargeWorld(t)
	chargeDecisionGrant(t, f, SampleMember, "author-decision")
	p := chargeRequest(f, requester, "self-author", digest)
	f.ok(t, "request", p, approvals.StatePending, "")
	f.code(t, "original author", f.runWith(f.as(MemberLogin), f.s.Revision(), "financial_charge.approve", "self-author", map[string]string{"digest": field(p, "digest")}), 403, string(commands.SelfApprovalDenied))
	fields := financialPermissionFields(f)
	fields["recipient"] = SampleAdmin3
	q := f.run(f.as(AdminLogin), "financial_permission.propose", "requester-decision", fields)
	f.ok(t, "requester permission proposal", q, inputauthority.Pending, "")
	f.ok(t, "requester permission", f.run(f.as(Admin2Login), "financial_permission.approve", "requester-decision", map[string]string{"digest": field(q, "digest")}), inputauthority.Applied, "")
	f.code(t, "requester self approval", f.run(requester, "financial_charge.approve", "self-author", map[string]string{"digest": field(p, "digest")}), 403, string(commands.SelfApprovalDenied))
}
