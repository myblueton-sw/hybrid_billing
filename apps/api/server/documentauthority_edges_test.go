package server

import (
	"encoding/json"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/inputauthority"
	"testing"
	"time"
)

func TestDocumentPermissionHTTPImpactCountsUnknownAndFailsClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	unknown := f.run(second, "document_authority.propose_console", "uncertain", documentPermissionTimes(f))
	f.ok(t, "proposal", unknown, inputauthority.Pending, "")
	f.s.fault = func(at string) bool { return at == "document-authority-after-grant" }
	f.code(t, "unknown", f.run(root, "document_authority.approve", "uncertain", map[string]string{"digest": field(unknown, "digest")}), 500, string(commands.InternalError))
	f.s.fault = nil
	pending := f.run(second, "document_authority.propose_console", "pending", documentPermissionTimes(f))
	f.ok(t, "pending", pending, inputauthority.Pending, "")
	staged := f.stage(root, "", SampleAdmin2)
	f.ok(t, "stage", staged, "draft", "")
	draft := field(staged, "draft")
	before := f.s.counter
	f.s.fault = func(at string) bool { return at == "document-authority-impact" }
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
		if row["store"] == "document_authority_proposals" {
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

func TestDocumentPermissionHTTPHoldAndStepUp(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	proposed := f.run(root, "document_authority.propose_console", "hold-proposal", documentPermissionTimes(f))
	f.ok(t, "proposal", proposed, inputauthority.Pending, "")
	f.advance(16 * time.Minute)
	f.code(t, "stepup required", f.run(second, "document_authority.approve", "hold-proposal", map[string]string{"digest": field(proposed, "digest")}), 403, CodeReauthentication)
	root, second = f.as(AdminLogin), f.as(Admin2Login)
	f.s.domain.Lock()
	f.s.enterHold("synthetic-test")
	f.s.domain.Unlock()
	f.code(t, "held approval", f.run(second, "document_authority.approve", "hold-proposal", map[string]string{"digest": field(proposed, "digest")}), 503, string(commands.DependencyUnavailable))
	if got := documentPermissionRead(f, root, "hold-proposal"); got.status != 200 {
		t.Fatalf("held inspect %s", got.raw)
	}
	rejected := f.run(second, "document_authority.reject", "hold-proposal", map[string]string{"digest": field(proposed, "digest")})
	f.ok(t, "held rejection", rejected, inputauthority.Rejected, "")
}

func TestDocumentPermissionCurrentActors(t *testing.T) {
	for _, mode := range []string{"recipient_removed", "recipient_suspended", "proposer_removed", "proposer_revoked", "confirmer_revoked", "console_revoked", "session_ended"} {
		t.Run(mode, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second := f.as(AdminLogin), f.as(Admin2Login)
			console := provisionDocumentCeiling(t, f)
			fields := documentPermissionTimes(f)
			fields["recipient"], fields["ceiling"] = SampleMember, field(console, "grant")
			p := f.run(root, "document_authority.propose_human", "actors", fields)
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
			f.code(t, mode, f.runWith(second, f.s.Revision(), "document_authority.approve", "actors", map[string]string{"digest": field(p, "digest")}), status, code)
			if _, found := f.s.grantView("document-grant-" + field(p, "digest")); found {
				t.Fatal("refusal created grant")
			}

		})
	}
}

func TestDocumentPermissionReplayRechecksInstallationGovernor(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	body, _ := json.Marshal(map[string]any{"request_id": "document-replay", "target_id": "document-replay", "idempotency_key": "document-replay", "expected_revision": f.s.Revision(), "rationale": "Synthetic permission request", "body": documentPermissionTimes(f)})
	f.ok(t, "proposal", f.call("document_authority.propose_console", string(body), root.cookie, root.csrf), inputauthority.Pending, "")
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == SampleAdmin && g.Terms.Action == gate.AdministerInstallation {
			f.must(f.s.admin.RevokeGrant(g.ID, g.Version))
		}
	}
	before := len(f.s.admin.Grants())
	f.code(t, "cached receipt refused", f.call("document_authority.propose_console", string(body), root.cookie, root.csrf), 403, string(commands.PermissionDenied))
	f.code(t, "inspect refused", documentPermissionRead(f, root, "document-replay"), 403, string(commands.PermissionDenied))
	if len(f.s.admin.Grants()) != before {
		t.Fatal("replay created effect")
	}
}
