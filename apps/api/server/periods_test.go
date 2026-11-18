package server

import (
	"bytes"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestRolePeriodRefusals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	run := func(fields map[string]string) reply {
		return f.run(root, "identity.set_role_period", SampleMember, fields)
	}
	for name, fields := range map[string]map[string]string{
		"an end without a time":   {"end": endAt},
		"no end with a time":      {"end": endNone, "role_until": f.at(time.Hour)},
		"an offset":               {"end": endAt, "role_until": f.s.clock().Add(time.Hour).Truncate(time.Minute).In(time.FixedZone("x", 9*3600)).Format(time.RFC3339)},
		"seconds":                 {"end": endAt, "role_until": f.s.clock().Add(time.Hour).Truncate(time.Minute).Add(time.Second).Format(time.RFC3339)},
		"less than a minute away": {"end": endAt, "role_until": f.at(0)},
		"beyond the longest":      {"end": endAt, "role_until": f.at(367 * 24 * time.Hour)},
		"not a time":              {"end": endAt, "role_until": "soon"},
	} {
		got := run(fields)
		if got.status != 422 || got.body["code"] != string(commands.InvalidRequest) {
			t.Errorf("%s: %s", name, got.raw)
		}
	}
	f.failed(t, "the setter's own period", f.period(root, SampleAdmin, f.at(time.Hour)), commands.PrerequisiteFailed)
	other, _ := f.created(root, "other@example.invalid", "member")
	f.failed(t, "no membership", f.period(root, other, f.at(time.Hour)), commands.InvalidStage)
	f.failed(t, "an equal open period", f.period(root, SampleMember, ""), commands.InvalidStage)
	f.failed(t, "an unknown person", f.period(root, "user-nobody", ""), commands.PermissionDenied)
	member := f.as(MemberLogin)
	f.failed(t, "a member", f.runWith(member, f.revision(root), "identity.set_role_period", SampleAdmin2, map[string]string{"end": endNone}), commands.PermissionDenied)
	until := f.at(366 * 24 * time.Hour)
	f.ok(t, "the longest end", f.period(root, SampleMember, until), "role_period_set", "map[ledger:confirmed role_until:"+until+"]")
	f.failed(t, "the same end again", f.period(root, SampleMember, until), commands.InvalidStage)
}

func TestRolePeriodEndsAccessAndIsSwept(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	member := f.withMember()
	signed := f.signedIn(MemberLogin, pw)
	until := f.at(3 * time.Minute)
	before := len(f.records(t, ""))
	f.ok(t, "set", f.period(root, SampleMember, until), "role_period_set", "")
	set := f.records(t, decisionledger.RolePeriodSet)
	if len(set) != 1 || set[0]["until"] != until || set[0]["role"] != "member" || set[0]["v"] != 2.0 || set[0]["requested_by"] != SampleAdmin ||
		len(f.records(t, "")) != before+1 {
		t.Fatalf("records %+v", set)
	}
	if data, _ := readLedger(f); bytes.Contains(data, []byte("sample reason")) || bytes.Contains(data, []byte(MemberLogin)) {
		t.Error("the ledger holds a rationale or a login")
	}
	row := f.row(root, SampleMember)["membership"].(map[string]any)
	if row["role_period"] != "ends_soon" || row["role_until"] != until || row["role_from"] == "" || row["ledger"] != ledgerOK || row["suspended_for"] != "" {
		t.Errorf("row %+v", row)
	}
	end, _ := time.Parse(time.RFC3339, until)
	f.advance(end.Sub(f.s.clock()) - time.Second)
	if got := f.me(signed); got.status != 200 {
		t.Fatalf("a second before the end %+v", got)
	}
	f.advance(time.Second)
	if got := f.me(signed); got.status != 401 {
		t.Errorf("at the end %+v", got)
	}
	if got := f.me(member); got.status != 401 {
		t.Errorf("another session at the end %+v", got)
	}
	if m := f.membership(SampleMember); m.State != identity.MembershipActive || m.Effective {
		t.Errorf("before the sweep %+v", m)
	}
	if row := f.row(root, SampleMember)["membership"].(map[string]any); row["role_period"] != "ended" {
		t.Errorf("the read shows an ended period before the sweep %+v", row)
	}
	f.failed(t, "the first command after the end meets the sweep", f.run(root, "identity.reinstate_membership", SampleMember, nil), commands.RevisionConflict)
	f.failed(t, "reinstating a period-ended membership", f.run(root, "identity.reinstate_membership", SampleMember, nil), commands.InvalidStage)
	ended := f.records(t, decisionledger.RolePeriodEnded)
	if len(ended) != 1 || ended[0]["revocation"] != "role-end-1" || ended[0]["requested_by"] != decisionledger.RoleSystem || ended[0]["until"] != until {
		t.Fatalf("ended %+v", ended)
	}
	if m := f.membership(SampleMember); m.State != identity.MembershipSuspended || m.SuspendedFor != identity.RolePeriodEnded {
		t.Errorf("after the sweep %+v", m)
	}
	f.failed(t, "changing the role of a period-ended membership", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}),
		commands.InvalidStage)
	later := f.at(24 * time.Hour)
	f.ok(t, "restore", f.period(root, SampleMember, later), "role_period_restored", "map[durable:yes ledger:confirmed role_until:"+later+"]")
	intents, confirms := f.records(t, decisionledger.IncreaseIntent), f.records(t, decisionledger.IncreaseConfirmed)
	if len(intents) != 1 || len(confirms) != 1 || confirms[0]["intent_sequence"] != intents[0]["seq"] || intents[0]["increase"] != decisionledger.IncreaseRolePeriod {
		t.Errorf("increase records %+v %+v", intents, confirms)
	}
	if m := f.membership(SampleMember); !m.Effective || m.SuspendedFor != "" {
		t.Errorf("restored %+v", m)
	}
	f.ok(t, "lengthen", f.period(root, SampleMember, f.at(48*time.Hour)), "role_period_set", "")
	f.ok(t, "remove the end", f.period(root, SampleMember, ""), "role_period_set", "map[durable:yes ledger:confirmed role_until:none]")
	if got := f.signedIn(MemberLogin, pw); got.cookie == nil {
		t.Error("the member cannot sign in after the restore")
	}
}

func TestRolePeriodCeilingAndOmissions(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	own := f.at(48 * time.Hour)
	f.ok(t, "an end on the second administrator", f.period(root, SampleAdmin2, own), "role_period_set", "")
	second := f.as(Admin2Login)
	f.fieldRefused(t, "a later end than the setter's own", f.period(second, SampleMember, f.at(72*time.Hour)), fieldUntil)
	f.fieldRefused(t, "no end by a setter with an end", f.period(second, SampleMember, ""), fieldUntil)
	f.ok(t, "the setter's own end", f.period(second, SampleMember, own), "role_period_set", "")
	f.fieldRefused(t, "create without an end", f.run(second, "identity.create_user", usersTarget, map[string]string{"login": "x@example.invalid",
		"email": "x@example.invalid", "role": "member"}), fieldUntil)
	f.fieldRefused(t, "a demotion that removes the end", f.run(second, "identity.change_role", SampleAdmin3, map[string]string{"role": "member", "end": endNone}), fieldUntil)
	f.fieldRefused(t, "an end without a time", f.run(second, "identity.change_role", SampleAdmin3, map[string]string{"role": "member", "end": endAt}), fieldUntil)
	f.fieldRefused(t, "no end with a time", f.run(second, "identity.change_role", SampleAdmin3, map[string]string{"role": "member", "end": endNone, "role_until": own}), fieldUntil)
	got := f.run(second, "identity.create_user", usersTarget, map[string]string{"login": "y@example.invalid", "email": "y@example.invalid", "role": "member",
		"role_until": f.at(24 * time.Hour)})
	if got.status != 200 {
		t.Fatalf("create with an end %s", got.raw)
	}
	end, _ := time.Parse(time.RFC3339, f.at(24*time.Hour))
	if inv := f.s.dir.invites["y@example.invalid"]; !inv.expires.Equal(end) {
		t.Errorf("the invitation outlives the role %+v", inv)
	}
	f.fieldRefused(t, "invite without an end", f.run(second, "identity.invite", SampleMember, map[string]string{"role": "member"}), fieldUntil)
}

func TestAnchorAdministrators(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second, third := f.as(AdminLogin), f.as(Admin2Login), f.as(Admin3Login)
	f.ok(t, "an end on the second", f.period(root, SampleAdmin2, f.at(time.Hour)), "role_period_set", "")
	f.ok(t, "an end on the third", f.period(root, SampleAdmin3, f.at(time.Hour)), "role_period_set", "")
	f.failed(t, "an end on the last anchor", f.period(second, SampleAdmin, f.at(time.Hour)), commands.PrerequisiteFailed)
	f.failed(t, "suspending the last anchor", f.run(second, "identity.suspend_user", SampleAdmin, nil), commands.PrerequisiteFailed)
	f.failed(t, "demoting the last anchor", f.run(third, "identity.change_role", SampleAdmin, map[string]string{"role": "member", "role_until": f.at(time.Hour)}),
		commands.PrerequisiteFailed)
	f.ok(t, "suspending an administrator with an end", f.run(root, "identity.suspend_membership", SampleAdmin3, nil), "membership_suspended", "")
	f.ok(t, "removing the end of the second", f.period(root, SampleAdmin2, ""), "role_period_set", "")
	f.ok(t, "now an end on the first", f.period(f.as(Admin2Login), SampleAdmin, f.at(time.Hour)), "role_period_set", "")
	f.ok(t, "a promotion with an end", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "member", "role_until": f.at(time.Hour)}),
		"role_member", "")
}

func TestRoleChangesAndPeriods(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	until := f.at(2 * time.Hour)
	f.ok(t, "promote with an end", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator", "role_until": until}), "role_administrator",
		"map[durable:yes ledger:confirmed role_until:"+until+"]")
	shorter := f.at(time.Hour)
	f.ok(t, "demote with a shorter end", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "member", "role_until": shorter}), "role_member",
		"map[ledger:confirmed role_until:"+shorter+"]")
	f.ok(t, "promote keeping the end", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator", "role_until": shorter}), "role_administrator", "")
	f.ok(t, "demote keeping the end", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "member", "role_until": shorter}), "role_member",
		"map[role_until:"+shorter+"]")
	f.stoppedLedger()
	f.failed(t, "a change that removes the end while stopped", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "member", "end": endNone}),
		commands.DependencyUnavailable)
	f.failed(t, "a promotion while stopped", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator", "role_until": shorter}),
		commands.DependencyUnavailable)
	f.ok(t, "a shorter end while stopped", f.period(root, SampleMember, f.at(30*time.Minute)), "role_period_set", "")
	if row := f.row(root, SampleMember)["membership"].(map[string]any); row["ledger"] != ledgerNot {
		t.Errorf("row %+v", row)
	}
	f.failed(t, "a longer end while stopped", f.period(root, SampleMember, f.at(time.Hour)), commands.DependencyUnavailable)
	f.failed(t, "a reinstatement while stopped", f.run(root, "identity.reinstate_user", SampleMember, nil), commands.DependencyUnavailable)
}

func TestSweepRunsBeforeCommandsSignInAndRedeem(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.ok(t, "set", f.period(root, SampleMember, f.at(time.Minute)), "role_period_set", "")
	f.advance(time.Minute)
	got := f.signin(MemberLogin, pw)
	if got.status != 401 || len(f.records(t, decisionledger.RolePeriodEnded)) != 1 {
		t.Errorf("sign-in after the end %+v", got)
	}
	g := setup(t, SeedSample).ready()
	groot := g.as(AdminLogin)
	g.withMember()
	g.ok(t, "set", g.period(groot, SampleMember, g.at(time.Minute)), "role_period_set", "")
	g.advance(time.Minute)
	g.setPassword(Admin2Login, pw)
	if len(g.records(t, decisionledger.RolePeriodEnded)) != 1 {
		t.Error("redeem did not sweep")
	}
	h := setup(t, SeedSample).ready()
	hroot := h.as(AdminLogin)
	h.withMember()
	h.ok(t, "set", h.period(hroot, SampleMember, h.at(time.Minute)), "role_period_set", "")
	h.advance(time.Minute)
	h.run(hroot, "identity.issue_set_link", SampleAdmin2, nil)
	if len(h.records(t, decisionledger.RolePeriodEnded)) != 1 {
		t.Error("a command did not sweep first")
	}
}

func TestTheWindowBeforeTheSweep(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.ok(t, "an end on the second", f.period(root, SampleAdmin2, f.at(time.Hour)), "role_period_set", "")
	second := f.as(Admin2Login)
	got := f.run(second, "identity.create_user", usersTarget, map[string]string{"login": "invitee@example.invalid", "email": "invitee@example.invalid", "role": "member",
		"role_until": f.at(time.Hour)})
	if got.status != 200 {
		t.Fatalf("create %s", got.raw)
	}
	proposed := f.requestGrant(second, SampleMember, nil)
	if proposed.status != 200 {
		t.Fatalf("proposal %s", proposed.raw)
	}
	f.advance(time.Hour)
	root = f.as(AdminLogin)
	if _, member := f.s.identity.Subjects().Member(SampleWorkspace, SampleAdmin2); member {
		t.Error("the subject view keeps the ended administrator")
	}
	row := f.row(root, SampleAdmin2)["membership"].(map[string]any)
	if row["role_period"] != "ended" || row["state"] != "active" || len(f.records(t, decisionledger.RolePeriodEnded)) != 0 {
		t.Errorf("read before the sweep %+v", row)
	}
	id := proposed.body["fields"].(map[string]any)["request"].(string)
	if req := f.requestRow(root, id); req["current"] != false {
		t.Errorf("the request of an ended proposer is current %+v", req)
	}
	invitee, _ := f.s.dir.userOf("invitee@example.invalid")
	f.s.domain.Lock()
	v, _ := f.s.probe("invitee@example.invalid")
	accepted := f.s.acceptPending("invitee@example.invalid", v, f.s.clock())
	f.s.domain.Unlock()
	if m := f.membership(invitee); !accepted || m.User != "" {
		t.Errorf("an invitation of an ended inviter was accepted %+v", m)
	}
	f.failed(t, "the ended administrator acts", f.runWith(f.as(Admin2Login), f.revision(root), "identity.suspend_user", SampleMember, map[string]string{}), commands.Unauthenticated)
}
