package server

import (
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func finalPermissionApprove(t *testing.T, f world, a actor, prefix, id string, p reply) reply {
	t.Helper()
	r := f.run(a, prefix+".approve", id, map[string]string{"digest": field(p, "digest")})
	f.ok(t, "permission approval", r, inputauthority.Applied, "")
	return r
}

func TestERPFinalDecisionPermissionStageAndRevocation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	first, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	check := func(kind string, want bool) {
		t.Helper()
		f.must(inputAuthorityAs(f, member, func() error {
			got := f.s.gate.CheckApproval(f.s.answer, SampleAdmin, gate.Request{Action: gate.Approve, Kind: kind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}})
			if got.Allowed != want {
				t.Fatalf("%s: %v want %v", kind, got.Allowed, want)
			}
			return nil
		}))
	}
	check(financialpermission.ERPFinalResourceKind, false)
	charge := f.run(first, "financial_permission.propose", "same-id", financialPermissionFields(f))
	f.ok(t, "charge proposal", charge, inputauthority.Pending, "")
	final := f.run(first, "erp_final_permission.propose", "same-id", financialPermissionFields(f))
	f.ok(t, "final proposal", final, inputauthority.Pending, "")
	if field(charge, "digest") == field(final, "digest") {
		t.Fatal("stage digest collision")
	}
	finalPermissionApprove(t, f, second, "financial_permission", "same-id", charge)
	check(financialpermission.ResourceKind, true)
	check(financialpermission.ERPFinalResourceKind, false)
	clients := clientGrants(f)
	f.code(t, "changed grant generation invalidates earlier final proposal", f.run(second, "erp_final_permission.approve", "same-id", map[string]string{"digest": field(final, "digest")}), 422, "invalid_stage")
	final = f.run(first, "erp_final_permission.propose", "final-current", financialPermissionFields(f))
	f.ok(t, "fresh final proposal", final, inputauthority.Pending, "")
	c := finalPermissionApprove(t, f, second, "erp_final_permission", "final-current", final)
	check(financialpermission.ERPFinalResourceKind, true)
	g, ok := f.s.grantView(field(c, "grant"))
	if !ok || g.Terms.Kind != financialpermission.ERPFinalResourceKind || g.SubjectID != SampleMember || g.ClientID != "" || clientGrants(f) != clients {
		t.Fatal("not direct final human authority")
	}
	f.code(t, "generic bypass", f.requestGrant(first, SampleMember, map[string]string{"action": "approve", "kind": financialpermission.ERPFinalResourceKind, "fields": ""}), 422, "invalid_request")
	if got := f.revokeVia(first, second, g.ID); got.status != 202 {
		t.Fatal(got.raw)
	}
	check(financialpermission.ERPFinalResourceKind, false)
	count := len(f.s.admin.Grants())
	finalPermissionApprove(t, f, second, "erp_final_permission", "final-current", final)
	check(financialpermission.ERPFinalResourceKind, false)
	if len(f.s.admin.Grants()) != count {
		t.Fatal("historical replay recreated grant")
	}
	_, _, err := f.s.financialPermissionStage(financialPolicyStage(99))
	documentFailure(t, err, commands.UnsupportedCapability)
}

func TestERPFinalRequestPermissionOwnCeilingIntersection(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	first, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	check := func(want bool) {
		t.Helper()
		f.must(inputAuthorityAs(f, member, func() error {
			got := f.s.gate.Decide(f.s.answer, ConsoleClient, gate.Request{Action: gate.RequestApproval, Kind: financialpermission.ERPFinalResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}})
			if got.Allowed != want {
				t.Fatalf("final request %v want %v", got.Allowed, want)
			}
			return nil
		}))
	}
	check(false)
	charge := provisionRequestCeiling(t, f)
	fields := requestPermissionTimes(f)
	fields["recipient"] = SampleMember
	fields["ceiling"] = field(charge, "grant")
	f.code(t, "charge ceiling cannot qualify final", f.run(first, "erp_final_request_permission.propose_human", "wrong-ceiling", fields), 403, "permission_denied")
	p := f.run(first, "erp_final_request_permission.propose_console", "final-console", requestPermissionTimes(f))
	f.ok(t, "final console proposal", p, inputauthority.Pending, "")
	c := finalPermissionApprove(t, f, second, "erp_final_request_permission", "final-console", p)
	check(false)
	fields["ceiling"] = field(c, "grant")
	f.code(t, "final ceiling cannot qualify charge", f.run(first, "financial_request_permission.propose_human", "wrong-ceiling", fields), 403, "permission_denied")
	h := f.run(first, "erp_final_request_permission.propose_human", "final-human", fields)
	f.ok(t, "final human proposal", h, inputauthority.Pending, "")
	granted := finalPermissionApprove(t, f, second, "erp_final_request_permission", "final-human", h)
	check(true)
	count := len(f.s.admin.Grants())
	if got := f.revokeVia(first, second, field(granted, "grant")); got.status != 202 {
		t.Fatal(got.raw)
	}
	ceiling, _ := f.s.grantView(field(c, "grant"))
	f.must(f.s.admin.RevokeGrant(ceiling.ID, ceiling.Version))
	finalPermissionApprove(t, f, second, "erp_final_request_permission", "final-human", h)
	check(false)
	if len(f.s.admin.Grants()) != count {
		t.Fatal("historical replay recreated request permission")
	}
}
