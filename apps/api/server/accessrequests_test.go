package server

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/accesschange"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/gate"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func requestOf(got reply) string { return got.body["fields"].(map[string]any)["request"].(string) }

func (f world) requestRow(a actor, id string) map[string]any {
	f.t.Helper()
	for _, r := range f.accessRead(a, "").body["requests"].([]any) {
		if row := r.(map[string]any); row["id"] == id {
			return row
		}
	}
	return nil
}

func (f world) grantCount(subject string) int {
	n := 0
	for _, g := range f.s.admin.Grants() {
		if g.SubjectID == subject && !g.Closed {
			n++
		}
	}
	return n
}

func TestRequestRejectAndWithdraw(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	asked := f.requestGrant(root, SampleMember, nil)
	id := requestOf(asked)
	row := f.requestRow(root, id)
	if row["state"] != "pending" || row["current"] != true || row["can_withdraw"] != true || row["can_approve"] != false || row["can_reject"] != false ||
		fmt.Sprint(row["eligible_approvers"]) != "1" || row["proposer"] != SampleAdmin || row["reason"] != "sample reason" || row["change"] != "add" || fmt.Sprint(row["days"]) != "10" {
		t.Errorf("proposer view %+v", row)
	}
	other := f.requestRow(second, id)
	if other["can_approve"] != true || other["can_reject"] != true || other["can_withdraw"] != false {
		t.Errorf("approver view %+v", other)
	}
	f.code(t, "reject by the proposer", f.run(root, "access.reject_request", id, nil), 422, string(commands.PrerequisiteFailed))
	f.code(t, "withdraw by another", f.run(second, "access.withdraw_request", id, nil), 422, string(commands.PrerequisiteFailed))
	f.code(t, "approve by the proposer", f.approveRequest(root, asked), 422, string(commands.PrerequisiteFailed))
	if got := f.run(second, "access.reject_request", id, nil); got.status != 200 || got.body["business_state"] != "rejected" {
		t.Fatalf("reject %s", got.raw)
	}
	if row := f.requestRow(root, id); row["state"] != "rejected" || row["decided_by"] != SampleAdmin2 || row["decision_reason"] != "sample reason" || row["can_withdraw"] != false {
		t.Errorf("rejected row %+v", row)
	}
	f.code(t, "approve a rejected request", f.approveRequest(second, asked), 422, string(commands.InvalidStage))
	again := f.requestGrant(root, SampleMember, nil)
	if again.status != 200 {
		t.Fatalf("again %s", again.raw)
	}
	if got := f.run(root, "access.withdraw_request", requestOf(again), nil); got.status != 200 || got.body["business_state"] != "withdrawn" {
		t.Errorf("withdraw %s", got.raw)
	}
	if f.grantCount(SampleMember) != 0 {
		t.Error("a rejected or withdrawn request granted access")
	}
	for _, id := range []string{"access-request-999", "access-request-01", "access-request-0", "grant-x"} {
		f.code(t, "tampered "+id, f.run(second, "access.approve_request", id, map[string]string{"digest": "x"}), 403, string(commands.PermissionDenied))
	}
	if _, res := (requestCommand{s: f.s, kind: "reject"}).Resolve("elsewhere", id); res != commands.NotFound {
		t.Error("a request from another workspace")
	}
	f.code(t, "approve without digest", f.run(second, "access.approve_request", requestOf(again), nil), 422, string(commands.InvalidRequest))
	third := f.requestGrant(root, SampleMember, nil)
	wrong := f.run(second, "access.approve_request", requestOf(third), map[string]string{"digest": strings.Repeat("0", 64)})
	if wrong.status != 422 || !strings.Contains(wrong.raw, `"digest"`) {
		t.Errorf("wrong digest %s", wrong.raw)
	}
}

func TestNoEligibleApprover(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if got := f.revokeVia(root, second, "reader-"+SampleAdmin2); got.status != 202 {
		t.Fatalf("revoke %s", got.raw)
	}
	asked := f.requestGrant(root, SampleAdmin2, nil)
	if asked.status != 200 {
		t.Fatalf("a request with no eligible approver is still accepted %s", asked.raw)
	}
	row := f.requestRow(second, requestOf(asked))
	if fmt.Sprint(row["eligible_approvers"]) != "0" || row["can_approve"] != false || row["can_reject"] != true {
		t.Errorf("row %+v", row)
	}
	f.code(t, "the subject approves", f.approveRequest(second, asked), 422, string(commands.PrerequisiteFailed))
}

func TestHolderApprovesButCannotRejectTheirRevocation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	asked := f.run(root, "access.request_revoke", "reader-"+SampleAdmin2, nil)
	row := f.requestRow(second, requestOf(asked))
	if row["can_reject"] != false || row["can_approve"] != true || row["subject"] != SampleAdmin2 || row["action"] != "view" || row["grant"] != "reader-"+SampleAdmin2 ||
		fmt.Sprint(row["eligible_approvers"]) != "1" {
		t.Errorf("holder view %+v", row)
	}
	f.code(t, "holder reject", f.run(second, "access.reject_request", requestOf(asked), nil), 422, string(commands.PrerequisiteFailed))
	if got := f.approveRequest(second, asked); got.status != 202 || got.body["fields"].(map[string]any)["request"] != requestOf(asked) {
		t.Errorf("holder approve %s", got.raw)
	}
	if g, _ := f.s.grantView("reader-" + SampleAdmin2); !g.Closed {
		t.Error("not revoked")
	}
}

func TestDuplicateGrantRegression(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	first, again := f.requestGrant(root, SampleMember, nil), f.requestGrant(root, SampleMember, nil)
	if requestOf(first) != requestOf(again) || len(f.accessRead(root, "").body["requests"].([]any)) != 1 {
		t.Fatalf("two submits gave two requests %s %s", first.raw, again.raw)
	}
	f.code(t, "another proposer for the same pair", f.requestGrant(second, SampleMember, nil), 422, string(commands.InvalidStage))
	f.s.fault = func(at string) bool { return at == "access-approve" }
	partial := f.approveRequest(second, first)
	f.code(t, "fault after apply", partial, 500, string(commands.InternalError))
	if partial.body["operation_id"] == nil {
		t.Error("partial without operation")
	}
	f.s.fault = nil
	f.code(t, "second approval", f.approveRequest(second, first), 422, string(commands.InvalidStage))
	if n := f.grantCount(SampleMember); n != 1 {
		t.Errorf("grants %d", n)
	}
	f.code(t, "an active equivalent grant", f.requestGrant(root, SampleMember, nil), 422, string(commands.InvalidStage))
}

func TestRequestsAreSupersededByBasisChanges(t *testing.T) {
	cases := map[string]func(f world, root, second actor){
		"ceiling": func(f world, root, second actor) {
			terms := gate.Terms{Action: gate.View, Kind: ReaderKind, Fields: []string{ReaderField}, Period: gate.AnyPeriod(),
				Scope: gate.Scope{WorkspaceID: SampleWorkspace, Customer: gate.All(), LegalEntity: gate.All(), Contract: gate.All()}}
			f.must(f.s.admin.AddGrant(gate.Grant{ID: "reader-console-later", ClientID: ConsoleClient, Terms: terms, Validity: gate.Validity{NotBefore: start, NotAfter: start.Add(2 * readerValidity)}}))
		},
		"subject membership": func(f world, root, second actor) {
			f.run(root, "identity.suspend_membership", SampleMember, nil)
			f.run(root, "identity.reinstate_membership", SampleMember, nil)
		},
		"subject epoch": func(f world, root, second actor) {
			f.run(root, "identity.suspend_user", SampleMember, nil)
			f.run(root, "identity.reinstate_user", SampleMember, nil)
		},
		"proposer role": func(f world, root, second actor) {
			f.run(second, "identity.change_role", SampleAdmin, map[string]string{"role": "member"})
		},
		"proposer administration": func(f world, root, second actor) {
			f.must(f.s.admin.RevokeGrant("installation-"+SampleAdmin, 1))
		},
		"ceiling tie with a lower ID": func(f world, root, second actor) {
			console, _ := f.s.grantView("reader-console")
			f.must(f.s.admin.AddGrant(gate.Grant{ID: "reader-a-ceiling", ClientID: ConsoleClient, Terms: console.Terms, Validity: console.Validity}))
		},
		"ceiling removed": func(f world, root, second actor) {
			f.must(f.s.admin.RevokeGrant("reader-console", 1))
		},
		"proposer coverage": func(f world, root, second actor) {
			f.must(f.s.admin.RevokeGrant("reader-"+SampleAdmin, 1))
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			f.setPassword(MemberLogin, pw)
			root, second := f.as(AdminLogin), f.as(Admin2Login)
			asked := f.requestGrant(root, SampleMember, nil)
			change(f, root, second)
			if row := f.requestRow(second, requestOf(asked)); row["current"] != false || row["can_approve"] != false || row["state"] != "pending" {
				t.Errorf("out-of-date row %+v", row)
			}
			got := f.approveRequest(second, asked)
			want := commands.InvalidStage
			if name == "proposer coverage" {
				want = commands.PrerequisiteFailed
			}
			f.code(t, name, got, 422, string(want))
			if f.grantCount(SampleMember) != 0 {
				t.Error("granted on a changed basis")
			}
			if name != "proposer coverage" {
				if row := f.requestRow(second, requestOf(asked)); row["state"] != "superseded" {
					t.Errorf("recorded %+v", row)
				}
			}
		})
	}
}

func TestRevokeGuardsAtApproval(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	second := f.as(Admin2Login)
	v, _, err := f.s.changes.Propose(accesschange.Proposal{Workspace: SampleWorkspace, Change: accesschange.Change{Kind: accesschange.ChangeRevoke, Grant: "reader-console"},
		Holder: "client", Basis: "x", Proposer: accesschange.Actor{User: SampleAdmin, Human: true}, Allowed: true})
	if err != nil {
		t.Fatal(err)
	}
	f.code(t, "a protected target at approval", f.run(second, "access.approve_request", v.ID, map[string]string{"digest": v.Digest}), 422, string(commands.PrerequisiteFailed))
	if g, _ := f.s.grantView("reader-console"); g.Closed {
		t.Error("a protected grant was closed")
	}
}

func TestGrantorCoverage(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if got := f.revokeVia(second, root, "reader-"+SampleAdmin); got.status != 202 {
		t.Fatalf("holder approves the revocation of their own grant %s", got.raw)
	}
	f.code(t, "a proposer without the pair", f.requestGrant(root, SampleMember, nil), 422, string(commands.PrerequisiteFailed))
	asked := f.requestGrant(second, SampleMember, nil)
	f.code(t, "an approver without the pair", f.approveRequest(root, asked), 422, string(commands.PrerequisiteFailed))
	if row := f.requestRow(root, requestOf(asked)); row["can_approve"] != false || row["state"] != "pending" {
		t.Errorf("row %+v", row)
	}
}

func TestRequestLimitsAndExpiry(t *testing.T) {
	f := setupWith(t, func(c *Config) {
		c.Access = DefaultAccessSettings()
		c.Access.PendingPerProposer, c.Access.PendingPerWorkspace, c.Access.RequestHours = 1, 1, 1
	})
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	asked := f.requestGrant(root, SampleMember, nil)
	f.code(t, "pending per proposer", f.run(root, "access.request_revoke", "reader-"+SampleAdmin2, nil), 422, string(commands.PrerequisiteFailed))
	f.code(t, "pending per workspace", f.run(second, "access.request_revoke", "reader-"+SampleAdmin, nil), 422, string(commands.PrerequisiteFailed))
	f.advance(time.Hour)
	root, second = f.as(AdminLogin), f.as(Admin2Login)
	if row := f.requestRow(root, requestOf(asked)); row["state"] != "expired" {
		t.Errorf("expired row %+v", row)
	}
	f.code(t, "approve expired", f.approveRequest(second, asked), 422, string(commands.InvalidStage))
	if got := f.requestGrant(root, SampleMember, nil); got.status != 200 {
		t.Errorf("a new request after expiry %s", got.raw)
	}
}

func TestMapChange(t *testing.T) {
	for reason, want := range map[accesschange.Reason]commands.Code{
		accesschange.ReasonNotPending: commands.InvalidStage, accesschange.ReasonExpired: commands.InvalidStage, accesschange.ReasonSuperseded: commands.InvalidStage,
		accesschange.ReasonPendingTarget: commands.InvalidStage, accesschange.ReasonNotFound: commands.InvalidStage, accesschange.ReasonDigest: commands.InvalidRequest,
		accesschange.ReasonExpiry: commands.InvalidRequest, accesschange.ReasonNotHuman: commands.PrerequisiteFailed, accesschange.ReasonHolderReject: commands.PrerequisiteFailed,
		accesschange.ReasonLimit: commands.PrerequisiteFailed, accesschange.ReasonNotAllowed: commands.PrerequisiteFailed,
	} {
		if got := mapChange(accesschange.Refusal{Reason: reason}).(*commands.Failure).Code; got != want {
			t.Errorf("%s: %s", reason, got)
		}
	}
	if mapChange(accesschange.ErrInvalid).(*commands.Failure).Code != commands.InvalidRequest || mapChange(gate.ErrUnavailable).(*commands.Failure).Code != commands.DependencyUnavailable ||
		mapChange(commands.Fail(commands.InvalidStage)).(*commands.Failure).Code != commands.InvalidStage {
		t.Error("other errors")
	}
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	if err := f.s.humanActor("nobody"); err.(*commands.Failure).Code != commands.PrerequisiteFailed {
		t.Errorf("a non-human or unknown actor %v", err)
	}
	f.s.caller = f.as(AdminLogin).verifier(f)
	defer func() { f.s.caller = identity.Verifier{} }()
	f.s.fault = func(at string) bool { return at == "users" }
	if _, err := (requestCommand{s: f.s, kind: "reject"}).Apply(commands.Context{User: SampleAdmin}); err.(*commands.Failure).Code != commands.DependencyUnavailable {
		t.Errorf("apply without users %v", err)
	}
	calls := 0
	f.s.fault = func(at string) bool { calls++; return at == "users" && calls > 1 }
	if _, err := (requestCommand{s: f.s, kind: "reject"}).Apply(commands.Context{User: SampleAdmin}); err.(*commands.Failure).Code != commands.DependencyUnavailable {
		t.Errorf("facts without users %v", err)
	}
	f.s.fault = nil
	if mapChange(errors.New("x")).(*commands.Failure).Code != commands.InvalidRequest {
		t.Error("unknown error")
	}
}
