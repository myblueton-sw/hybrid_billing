package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/financialpermission"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
)

func financialPermissionFields(f world) map[string]string {
	return map[string]string{"recipient": SampleMember, "not_before": f.s.clock().Format(time.RFC3339Nano), "not_after": f.s.clock().Add(12 * time.Hour).Format(time.RFC3339Nano)}
}

func financialPermissionRead(f world, a actor, id string) reply {
	body, _ := json.Marshal(map[string]any{"request_id": "permission-read", "target_id": id, "body": map[string]string{}})
	return f.call("financial_permission.inspect", string(body), a.cookie, a.csrf)
}

func TestFinancialPermissionHTTPGrantAndActualM03Revoke(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	grants, clients, limits := len(f.s.admin.Grants()), clientGrants(f), len(f.s.financialAuthorities)
	request := gate.Request{Action: gate.Approve, Kind: financialpermission.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
	check := func(want bool) {
		t.Helper()
		if err := inputAuthorityAs(f, member, func() error {
			if got := f.s.gate.CheckApproval(f.s.answer, SampleAdmin, request); got.Allowed != want {
				t.Fatalf("approval allowed %v want %v: %s", got.Allowed, want, got.Reason)
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
	check(false)
	p := f.run(root, "financial_permission.propose", "permission-one", financialPermissionFields(f))
	f.ok(t, "proposal", p, inputauthority.Pending, "")
	digest := map[string]string{"digest": field(p, "digest")}
	f.code(t, "self approval", f.run(root, "financial_permission.approve", "permission-one", digest), 403, string(commands.PermissionDenied))
	c := f.run(second, "financial_permission.approve", "permission-one", digest)
	f.ok(t, "approval", c, inputauthority.Applied, "")
	if field(c, "receipt") != "historical_decision" || len(f.s.admin.Grants()) != grants+1 || clientGrants(f) != clients || len(f.s.financialAuthorities) != limits {
		t.Fatal(c.raw)
	}
	g, ok := f.s.grantView(field(c, "grant"))
	if !ok || g.SubjectID != SampleMember || g.ClientID != "" || g.Terms.Action != gate.Approve || g.Terms.Kind != financialpermission.ResourceKind || !fullScope(g.Terms) || len(g.Terms.Fields) != 0 {
		t.Fatal(g)
	}
	check(true)
	f.code(t, "member inspection", financialPermissionRead(f, member, "permission-one"), 403, string(commands.PermissionDenied))
	if got := f.revokeVia(root, second, g.ID); got.status != 202 || got.body["job_id"] == "" {
		t.Fatal(got.raw)
	}
	check(false)
	f.ok(t, "historical replay", f.run(second, "financial_permission.approve", "permission-one", digest), inputauthority.Applied, "")
	check(false)
	if len(f.s.admin.Grants()) != grants+1 {
		t.Fatal("replay recreated grant")
	}
}

func TestFinancialPermissionHTTPUnknownRevokedReconciliation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	p := f.run(root, "financial_permission.propose", "unknown", financialPermissionFields(f))
	f.ok(t, "proposal", p, inputauthority.Pending, "")
	digest := map[string]string{"digest": field(p, "digest")}
	f.s.fault = func(at string) bool { return at == "financial-permission-after-grant" }
	f.code(t, "unknown", f.run(second, "financial_permission.approve", "unknown", digest), 500, string(commands.InternalError))
	f.s.fault = nil
	id := "financial-permission-" + field(p, "digest")
	if got := f.revokeVia(root, second, id); got.status != 202 || got.body["job_id"] == "" {
		t.Fatal(got.raw)
	}
	count := len(f.s.admin.Grants())
	c := f.run(second, "financial_permission.approve", "unknown", digest)
	f.ok(t, "reconciliation", c, inputauthority.Applied, "")
	if field(c, "recorded_grant_status") != inputauthority.RevokedGrant || len(f.s.admin.Grants()) != count {
		t.Fatal(c.raw)
	}
}

func TestFinancialPermissionReplayRechecksModuleGovernor(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	_, err := f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "add-governor", Kind: approvals.ProposeAddAdmin, Admin: SampleAdmin3}, approvals.Actor{ID: SampleAdmin, Kind: approvals.KindHuman})
	f.must(err)
	_, err = f.s.approvals.Confirm(SampleWorkspace, "add-governor", approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	f.must(err)
	body, _ := json.Marshal(map[string]any{"request_id": "replay", "target_id": "replay", "idempotency_key": "permission-replay", "expected_revision": f.s.Revision(), "rationale": "Synthetic permission request", "body": financialPermissionFields(f)})
	f.ok(t, "proposal", f.call("financial_permission.propose", string(body), root.cookie, root.csrf), inputauthority.Pending, "")
	_, err = f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "remove-governor", Kind: approvals.ProposeRemoveAdmin, Admin: SampleAdmin}, approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	f.must(err)
	_, err = f.s.approvals.Confirm(SampleWorkspace, "remove-governor", approvals.Actor{ID: SampleAdmin3, Kind: approvals.KindHuman})
	f.must(err)
	before := len(f.s.admin.Grants())
	f.code(t, "same-key replay after removal", f.call("financial_permission.propose", string(body), root.cookie, root.csrf), 403, string(commands.PermissionDenied))
	f.code(t, "inspect after removal", financialPermissionRead(f, root, "replay"), 403, string(commands.PermissionDenied))
	if len(f.s.admin.Grants()) != before {
		t.Fatal("lost governor produced effect")
	}
}

func TestFinancialPermissionHTTPFixedShapeAndPreflight(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	for _, name := range []string{"capability", "ceiling", "kind", "action", "workspace"} {
		fields := financialPermissionFields(f)
		fields[name] = "unexpected"
		f.code(t, name, f.run(root, "financial_permission.propose", "forged-"+name, fields), 422, string(commands.InvalidRequest))
	}
	saved := f.s.cfg.Commands
	for _, change := range []func(){func() { f.s.cfg.Commands.Fields = 19 }, func() { f.s.cfg.Commands.Value = 1023 }} {
		change()
		f.code(t, "capacity", f.run(root, "financial_permission.propose", "capacity", financialPermissionFields(f)), 501, string(commands.UnsupportedCapability))
		f.s.cfg.Commands = saved
	}
	if len(f.s.financialPermission.bindings) != 0 {
		t.Fatal("preflight created bindings")
	}
	fields := financialPermissionFields(f)
	fields["recipient"] = SampleAdmin
	f.code(t, "recipient proposes", f.run(root, "financial_permission.propose", "self-recipient", fields), 403, string(commands.PermissionDenied))
	f.code(t, "generic grant bypass", f.requestGrant(root, SampleMember, map[string]string{"action": "approve", "kind": financialpermission.ResourceKind, "fields": ""}), 422, string(commands.InvalidRequest))
}
