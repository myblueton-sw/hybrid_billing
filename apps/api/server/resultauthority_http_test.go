package server

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"github.com/myblueton-sw/hybrid_billing/modules/resultauthority"
)

func resultPermissionTimes(f world) map[string]string {
	return map[string]string{"capability": "apply", "not_before": f.s.clock().Format(time.RFC3339Nano), "not_after": f.s.clock().Add(12 * time.Hour).Format(time.RFC3339Nano)}
}
func resultPermissionApprove(f world, a actor, id string, p reply) reply {
	return f.run(a, "result_authority.approve", id, map[string]string{"digest": field(p, "digest")})
}
func resultPermissionRead(f world, a actor, id string) reply {
	body, _ := json.Marshal(map[string]any{"request_id": "request-permission-read", "target_id": id, "body": map[string]string{}})
	return f.call("result_authority.inspect", string(body), a.cookie, a.csrf)
}
func provisionResultCeiling(t *testing.T, f world) reply {
	t.Helper()
	p := f.run(f.as(AdminLogin), "result_authority.propose_console", "request-console", resultPermissionTimes(f))
	f.ok(t, "console proposal", p, inputauthority.Pending, "")
	got := resultPermissionApprove(f, f.as(Admin2Login), "request-console", p)
	f.ok(t, "console approval", got, inputauthority.Applied, "")
	return got
}

func TestResultPermissionActualIntersectionAndM03Revocation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
	request := gate.Request{Action: gate.ApplyERPResult, Kind: resultauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}}
	check := func(want bool) {
		t.Helper()
		f.must(inputAuthorityAs(f, member, func() error {
			if got := f.s.gate.Decide(f.s.answer, ConsoleClient, request); got.Allowed != want {
				t.Fatal(got, want)
			}
			decision := request
			decision.Action = gate.Approve
			if got := f.s.gate.CheckApproval(f.s.answer, SampleAdmin, decision); got.Allowed {
				t.Fatal("request permission granted approval")
			}
			return nil
		}))
	}
	before := len(f.s.admin.Grants())
	check(false)
	console := provisionResultCeiling(t, f)
	check(false)
	fields := resultPermissionTimes(f)
	fields["recipient"] = SampleMember
	fields["ceiling"] = field(console, "grant")
	p := f.run(root, "result_authority.propose_human", "request-human", fields)
	f.ok(t, "human proposal", p, inputauthority.Pending, "")
	c := resultPermissionApprove(f, second, "request-human", p)
	f.ok(t, "human approval", c, inputauthority.Applied, "")
	if field(c, "capability") != resultauthority.Apply || field(c, "receipt") != "historical_decision" || len(f.s.admin.Grants()) != before+2 {
		t.Fatal(c.raw)
	}
	check(true)
	for _, id := range []string{field(console, "grant"), field(c, "grant")} {
		g, ok := f.s.grantView(id)
		if !ok || g.Terms.Action != gate.ApplyERPResult || g.Terms.Kind != resultauthority.ResourceKind || len(g.Terms.Fields) != 0 || !fullScope(g.Terms) {
			t.Fatal(g)
		}
	}
	if got := f.run(root, "access.request_revoke", field(console, "grant"), nil); got.status == 200 || got.status == 202 {
		t.Fatal("console protection removed", got.raw)
	}
	if got := f.revokeVia(root, second, field(c, "grant")); got.status != 202 {
		t.Fatal(got.raw)
	}
	check(false)
	f.ok(t, "historical replay", resultPermissionApprove(f, second, "request-human", p), inputauthority.Applied, "")
	check(false)
	if len(f.s.admin.Grants()) != before+2 {
		t.Fatal("revoked grant recreated")
	}
	f.code(t, "member inspect", resultPermissionRead(f, member, "request-human"), 403, string(commands.PermissionDenied))
}

func TestResultPermissionRevokedAndForeignCeilings(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	console := provisionResultCeiling(t, f)
	fields := resultPermissionTimes(f)
	fields["recipient"] = SampleMember
	fields["ceiling"] = field(console, "grant")
	p := f.run(root, "result_authority.propose_human", "pending-human", fields)
	f.ok(t, "human proposal", p, inputauthority.Pending, "")
	f.must(f.s.admin.RevokeGrant(field(console, "grant"), 1))
	f.code(t, "revoked ceiling proposal", f.run(root, "result_authority.propose_human", "later-human", fields), 403, string(commands.PermissionDenied))
	f.code(t, "revoked ceiling approval", resultPermissionApprove(f, second, "pending-human", p), 403, string(commands.PermissionDenied))
	other := f.run(root, "input_authority.propose_console", "other-domain", authorityTimes(f))
	f.ok(t, "input proposal", other, inputauthority.Pending, "")
	other = authorityApprove(f, second, "other-domain", other)
	f.ok(t, "input approval", other, inputauthority.Applied, "")
	fields["ceiling"] = field(other, "grant")
	f.code(t, "foreign ceiling", f.run(root, "result_authority.propose_human", "foreign", fields), 403, string(commands.PermissionDenied))
}

func TestResultPermissionUnknownRevokedHumanDoesNotRecreate(t *testing.T) {
	for _, fault := range []string{"before-grant", "after-grant"} {
		t.Run(fault, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second := f.as(AdminLogin), f.as(Admin2Login)
			console := provisionResultCeiling(t, f)
			fields := resultPermissionTimes(f)
			fields["recipient"] = SampleMember
			fields["ceiling"] = field(console, "grant")
			p := f.run(root, "result_authority.propose_human", "uncertain", fields)
			f.ok(t, "proposal", p, inputauthority.Pending, "")
			f.s.fault = func(at string) bool { return at == "result-authority-"+fault }
			f.code(t, "unknown", resultPermissionApprove(f, second, "uncertain", p), 500, string(commands.InternalError))
			f.s.fault = nil
			before := len(f.s.admin.Grants())
			if fault == "after-grant" {
				if got := f.revokeVia(root, second, "result-grant-"+field(p, "digest")); got.status != 202 {
					t.Fatal(got.raw)
				}
			}
			got := resultPermissionApprove(f, second, "uncertain", p)
			f.ok(t, "resolution", got, inputauthority.Applied, "")
			if fault == "after-grant" && (field(got, "recorded_grant_status") != inputauthority.RevokedGrant || len(f.s.admin.Grants()) != before) {
				t.Fatal("historical effect recreated", got.raw)
			}
			if fault == "before-grant" && len(f.s.admin.Grants()) != before+1 {
				t.Fatal("absent effect not applied once")
			}
		})
	}
}

func TestResultPermissionUnknownAfterCeilingRevocation(t *testing.T) {
	for _, fault := range []string{"before-grant", "after-grant"} {
		t.Run(fault, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second, member := f.as(AdminLogin), f.as(Admin2Login), f.as(MemberLogin)
			console := provisionResultCeiling(t, f)
			fields := resultPermissionTimes(f)
			fields["recipient"], fields["ceiling"] = SampleMember, field(console, "grant")
			p := f.run(root, "result_authority.propose_human", "uncertain-ceiling", fields)
			f.ok(t, "proposal", p, inputauthority.Pending, "")
			f.s.fault = func(at string) bool { return at == "result-authority-"+fault }
			f.code(t, "unknown", resultPermissionApprove(f, second, "uncertain-ceiling", p), 500, string(commands.InternalError))
			f.s.fault = nil
			f.must(f.s.admin.RevokeGrant(field(console, "grant"), 1))
			count := len(f.s.admin.Grants())
			got := resultPermissionApprove(f, second, "uncertain-ceiling", p)
			if fault == "after-grant" {
				f.ok(t, "historical existing effect", got, inputauthority.Applied, "")
			} else {
				f.code(t, "absent effect refused", got, 403, string(commands.PermissionDenied))
			}
			if len(f.s.admin.Grants()) != count {
				t.Fatal("ceiling revocation caused new effect")
			}
			f.must(inputAuthorityAs(f, member, func() error {
				got := f.s.gate.Decide(f.s.answer, ConsoleClient, gate.Request{Action: gate.ApplyERPResult, Kind: resultauthority.ResourceKind, Resource: gate.Resource{WorkspaceID: SampleWorkspace}})
				if got.Allowed {
					t.Fatal("revoked console still permitted request")
				}
				return nil
			}))
		})
	}
}
