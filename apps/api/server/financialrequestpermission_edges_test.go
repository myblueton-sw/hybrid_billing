package server

import (
	"encoding/json"
	"github.com/myblueton-sw/hybrid_billing/modules/approvals"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"testing"
	"time"
)

func TestRequestPermissionHTTPImpactCountsUnknownAndFailsClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	unknown := f.run(second, "financial_request_permission.propose_console", "uncertain", requestPermissionTimes(f))
	f.ok(t, "proposal", unknown, inputauthority.Pending, "")
	f.s.fault = func(at string) bool { return at == "financial-request-after-grant" }
	f.code(t, "unknown", f.run(root, "financial_request_permission.approve", "uncertain", map[string]string{"digest": field(unknown, "digest")}), 500, string(commands.InternalError))
	f.s.fault = nil
	pending := f.run(second, "financial_request_permission.propose_console", "pending", requestPermissionTimes(f))
	f.ok(t, "pending", pending, inputauthority.Pending, "")
	staged := f.stage(root, "", SampleAdmin2)
	f.ok(t, "stage", staged, "draft", "")
	draft := field(staged, "draft")
	before := f.s.counter
	f.s.fault = func(at string) bool { return at == "financial-request-impact" }
	f.code(t, "impact unavailable", f.prepare(root, draft, "suspend_membership"), 503, string(commands.DependencyUnavailable))
	if f.s.counter != before {
		t.Fatal("failed impact prepared a snapshot")
	}
	f.s.fault = nil
	prepared := f.prepare(root, draft, "suspend_membership")
	f.ok(t, "prepare", prepared, "prepared", "")
	if field(prepared, "bulk") != "bulk-revocation-1" {
		t.Fatal("failed attempt reserved snapshot")
	}
	detail := f.bulkRead(root, "/"+field(prepared, "bulk"))
	rows := detail.body["impact"].(map[string]any)["pending"].([]any)
	found := false
	for _, raw := range rows {
		row := raw.(map[string]any)
		if row["store"] == "financial_request_permission_proposals" {
			found = true
			if row["count"] != 2.0 || row["rechecked"] != true {
				t.Fatalf("input impact %+v", row)
			}
		}
	}
	if !found {
		t.Fatal("new proposal namespace omitted")
	}
}

func TestRequestPermissionHTTPHoldAndStepUp(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	proposed := f.run(root, "financial_request_permission.propose_console", "hold-proposal", requestPermissionTimes(f))
	f.ok(t, "proposal", proposed, inputauthority.Pending, "")
	f.advance(16 * time.Minute)
	f.code(t, "stepup required", f.run(second, "financial_request_permission.approve", "hold-proposal", map[string]string{"digest": field(proposed, "digest")}), 403, CodeReauthentication)
	root, second = f.as(AdminLogin), f.as(Admin2Login)
	f.s.domain.Lock()
	f.s.enterHold("synthetic-test")
	f.s.domain.Unlock()
	f.code(t, "held approval", f.run(second, "financial_request_permission.approve", "hold-proposal", map[string]string{"digest": field(proposed, "digest")}), 503, string(commands.DependencyUnavailable))
	if got := requestPermissionRead(f, root, "hold-proposal"); got.status != 200 {
		t.Fatalf("held inspect %s", got.raw)
	}
	rejected := f.run(second, "financial_request_permission.reject", "hold-proposal", map[string]string{"digest": field(proposed, "digest")})
	f.ok(t, "held rejection", rejected, inputauthority.Rejected, "")
}

func TestRequestPermissionReplayRechecksModuleGovernor(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	_, err := f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "add-governor", Kind: approvals.ProposeAddAdmin, Admin: SampleAdmin3}, approvals.Actor{ID: SampleAdmin, Kind: approvals.KindHuman})
	f.must(err)
	_, err = f.s.approvals.Confirm(SampleWorkspace, "add-governor", approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	f.must(err)
	body, _ := json.Marshal(map[string]any{"request_id": "replay", "target_id": "replay", "idempotency_key": "permission-replay", "expected_revision": f.s.Revision(), "rationale": "Synthetic permission request", "body": requestPermissionTimes(f)})
	f.ok(t, "proposal", f.call("financial_request_permission.propose_console", string(body), root.cookie, root.csrf), inputauthority.Pending, "")
	_, err = f.s.approvals.Propose(SampleWorkspace, approvals.Proposal{ID: "remove-governor", Kind: approvals.ProposeRemoveAdmin, Admin: SampleAdmin}, approvals.Actor{ID: SampleAdmin2, Kind: approvals.KindHuman})
	f.must(err)
	_, err = f.s.approvals.Confirm(SampleWorkspace, "remove-governor", approvals.Actor{ID: SampleAdmin3, Kind: approvals.KindHuman})
	f.must(err)
	before := len(f.s.admin.Grants())
	f.code(t, "same-key replay after removal", f.call("financial_request_permission.propose_console", string(body), root.cookie, root.csrf), 403, string(commands.PermissionDenied))
	f.code(t, "inspect after removal", requestPermissionRead(f, root, "replay"), 403, string(commands.PermissionDenied))
	if len(f.s.admin.Grants()) != before {
		t.Fatal("lost governor produced effect")
	}
}

func TestRequestPermissionCurrentActors(t *testing.T) {
	for _, mode := range []string{"recipient_removed", "recipient_suspended", "proposer_removed", "proposer_revoked", "confirmer_revoked", "console_revoked", "session_ended"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second := f.as(AdminLogin), f.as(Admin2Login)
			console := provisionRequestCeiling(t, f)
			fields := requestPermissionTimes(f)
			fields["recipient"], fields["ceiling"] = SampleMember, field(console, "grant")
			p := f.run(root, "financial_request_permission.propose_human", "actors", fields)
			f.ok(t, "propose", p, inputauthority.Pending, "")
			switch mode {
			case "recipient_removed":
				f.must(f.s.identity.Workspace().RemoveMembership(root.verifier(f), SampleWorkspace, SampleMember, f.membership(SampleMember).Version))
			case "recipient_suspended":
				f.must(f.s.identity.Workspace().SuspendMembership(root.verifier(f), SampleWorkspace, SampleMember, f.membership(SampleMember).Version))
			case "proposer_removed":
				f.must(f.s.identity.Workspace().RemoveMembership(second.verifier(f), SampleWorkspace, SampleAdmin, f.membership(SampleAdmin).Version))
			case "session_ended":
				f.must(f.s.identity.Boundary().EndSession(second.verifier(f)))
			default:
				for _, g := range f.s.admin.Grants() {
					if g.Terms.Action == gate.AdministerInstallation && (mode == "console_revoked" && g.ClientID == ConsoleClient || mode == "proposer_revoked" && g.SubjectID == SampleAdmin || mode == "confirmer_revoked" && g.SubjectID == SampleAdmin2) {
						f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
					}
				}
			}
			status, code := 403, string(commands.PermissionDenied)
			if mode == "session_ended" {
				status, code = 401, string(commands.Unauthenticated)
			}
			f.code(t, mode, f.runWith(second, f.s.Revision(), "financial_request_permission.approve", "actors", map[string]string{"digest": field(p, "digest")}), status, code)
			if _, found := f.s.grantView("financial-request-" + field(p, "digest")); found {
				t.Fatal("refusal created grant")
			}

		})
	}
}
