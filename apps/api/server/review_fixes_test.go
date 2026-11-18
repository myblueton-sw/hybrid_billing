package server

import (
	"net"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/jobs"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func TestChangeRoleClassifiesByDirection(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.ok(t, "an end on the third", f.period(root, SampleAdmin3, f.at(2*time.Hour)), "role_period_set", "")
	shorter := f.at(time.Hour)
	before := len(f.records(t, decisionledger.IncreaseIntent))
	f.ok(t, "the same administrator role with a shorter end", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "administrator", "role_until": shorter}),
		"role_administrator", "map[ledger:confirmed role_until:"+shorter+"]")
	if len(f.records(t, decisionledger.IncreaseIntent)) != before || len(f.records(t, decisionledger.RolePeriodSet)) != 2 {
		t.Error("a shortening was recorded as an increase")
	}
	f.failed(t, "the same role and end", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "administrator"}), commands.InvalidStage)
	f.stoppedLedger()
	even := f.at(30 * time.Minute)
	f.ok(t, "a shortening while the ledger cannot record", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "administrator", "role_until": even}),
		"role_administrator", "map[ledger:unconfirmed role_until:"+even+"]")
	f.failed(t, "a lengthening while the ledger cannot record", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "administrator", "role_until": shorter}),
		commands.DependencyUnavailable)
	g := setup(t, SeedSample).ready()
	groot := g.as(AdminLogin)
	g.ok(t, "an end", g.period(groot, SampleAdmin3, g.at(2*time.Hour)), "role_period_set", "")
	g.ok(t, "a shorter end through change_role", g.run(groot, "identity.change_role", SampleAdmin3, map[string]string{"role": "administrator", "role_until": g.at(time.Hour)}),
		"role_administrator", "")
	g.appendGarbage(t)
	h := g.restart(nil)
	end, _ := time.Parse(time.RFC3339, g.at(time.Hour))
	if m := h.membership(SampleAdmin3); h.s.ledger.Status().State != decisionledger.StateIsolated || !m.Until.Equal(end) {
		t.Errorf("degraded replay did not keep the shorter end %+v", m)
	}
}

func TestChangeRoleKeepsAnOmittedEnd(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	until := f.at(2 * time.Hour)
	f.ok(t, "an end", f.period(root, SampleMember, until), "role_period_set", "")
	f.ok(t, "a promotion without a role end keeps the end", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}), "role_administrator",
		"map[durable:yes ledger:confirmed role_until:"+until+"]")
	if m := f.membership(SampleMember); m.Until.IsZero() {
		t.Errorf("an omitted end was removed %+v", m)
	}
	f.ok(t, "an explicit removal", f.run(root, "identity.change_role", SampleMember, map[string]string{"role": "member", "end": endNone}), "role_member", "")
	if m := f.membership(SampleMember); !m.Until.IsZero() || m.Role != identity.Member {
		t.Errorf("an explicit no end %+v", m)
	}
}

func TestPreparedListsAreTheOwnersToRunOrDiscard(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.withMember()
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember)
	for _, command := range []string{"bulk_revocation.execute", "bulk_revocation.discard"} {
		fields := map[string]string{}
		if command == "bulk_revocation.execute" {
			fields["digest"] = digest
		}
		other, unknown := f.run(second, command, id, fields), f.run(second, command, "bulk-revocation-99", fields)
		if other.status != unknown.status || other.body["code"] != unknown.body["code"] || other.body["code"] != string(commands.PermissionDenied) {
			t.Errorf("%s by another holder %s, unknown %s", command, other.raw, unknown.raw)
		}
	}
	if got := f.bulkRead(second, "/"+id); got.status != 200 || f.bulkRead(second, "/"+id+"/targets").status != 404 {
		t.Errorf("another holder still reads the counts %s", got.raw)
	}
	if got := f.bulkRead(root, "/"+id+"/targets?kind=excluded"); got.status != 200 || got.body["total"] != 0.0 {
		t.Errorf("exclusions %s", got.raw)
	}
	if got := f.bulkRead(second, "/"+id+"/targets?kind=excluded"); got.status != 404 {
		t.Errorf("exclusions for another holder %s", got.raw)
	}
	if got := f.execute(root, id, digest); got.status != 202 {
		t.Errorf("the owner executes %s", got.raw)
	}
	staged := f.stage(root, "", SampleAdmin, "user-00000000000000ff")
	prepared := f.prepare(root, field(staged, "draft"), "suspend_membership")
	_ = prepared
	if got := f.bulkRead(root, "/"+field(staged, "draft")+"/targets?kind=excluded"); got.status != 404 {
		t.Errorf("exclusions of a draft %s", got.raw)
	}
}

func TestExclusionsReadPages(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	id, _ := f.prepared(t, root, "suspend_membership", SampleMember, SampleAdmin, "user-00000000000000ff")
	got := f.bulkRead(root, "/"+id+"/targets?kind=excluded&limit=1")
	rows := got.body["exclusions"].([]any)
	if got.status != 200 || got.body["total"] != 2.0 || got.body["next"] != "1" || len(rows) != 1 || rows[0].(map[string]any)["reason"] != "self" {
		t.Errorf("exclusions %s", got.raw)
	}
}

func TestChangeRoleKeepsAnAnchorAndAllHeldListsDenyNobody(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.ok(t, "an end on the third", f.period(root, SampleAdmin3, f.at(time.Hour)), "role_period_set", "")
	f.ok(t, "an end on the second", f.period(root, SampleAdmin2, f.at(time.Hour)), "role_period_set", "")
	f.failed(t, "an end on the last anchor through change_role", f.run(f.as(Admin2Login), "identity.change_role", SampleAdmin, map[string]string{"role": "administrator",
		"role_until": f.at(time.Hour)}), commands.PrerequisiteFailed)
	f.withMember()
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember)
	f.ok(t, "a change after the preview", f.period(root, SampleMember, f.at(time.Hour)), "role_period_set", "")
	got := f.execute(root, id, digest)
	if got.status != 202 || field(got, "held") != "1" || f.membership(SampleMember).State != identity.MembershipActive {
		t.Errorf("every target held %s", got.raw)
	}
}

func TestRestartKeepsABulkSuspensionUnderALaterPeriodIncrease(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendMembership, Target: SampleAdmin3, TargetVersion: 1, Revocation: "bulk-revocation-4:0",
		Request: "bulk-revocation-4", RequestDigest: digestOf("x"), RequestedBy: SampleAdmin})
	intent := f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseIntent, Target: SampleAdmin3, TargetVersion: 2, Increase: decisionledger.IncreaseRolePeriod,
		Role: "administrator", Until: stamp(f.s.clock().Add(time.Hour)), RequestedBy: SampleAdmin})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseConfirmed, Target: SampleAdmin3, IntentSequence: intent})
	g := f.restart(nil)
	if m := g.membership(SampleAdmin3); m.State != identity.MembershipSuspended || m.Until.IsZero() {
		t.Errorf("a later period increase lifted a bulk suspension on replay %+v", m)
	}
}

func TestAFinalCleanupFailureMarksOnlyItsOwnTask(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.s.fault = func(at string) bool { return at == "cleanup-create" }
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember, SampleAdmin3)
	if got := f.execute(root, id, digest); got.status != 202 {
		t.Fatal(got.raw)
	}
	v, _ := f.s.bulk.Get(f.s.bootWorkspace, "", id)
	f.s.domain.Lock()
	f.s.jobs, _ = jobs.NewRegistry(cleanupLimits(f.s.cfg.Access.BulkMaxMatching), f.s.clock)
	_, err := f.s.jobs.Create(v.Workspace, v.ID, cleanupKind, cleanupPolicy, []jobs.TaskSpec{{ID: "chunk-1", Input: "0:0", Repeatable: true}, {ID: "chunk-2", Input: "1:1", Repeatable: true}})
	_ = f.s.bulk.Cleanup(v.Workspace, v.ID, true)
	f.s.domain.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	first := func() bulkrevoke.Item { now, _ := f.s.bulk.Get(f.s.bootWorkspace, "", id); return now.Items[0] }
	f.s.fault = func(at string) bool { return at == "cleanup" && first().Outcome != bulkrevoke.OutcomeFailed }
	f.s.drainCleanup()
	if job := f.job(root, id); job["counts"].(map[string]any)["failed"] != 0.0 {
		t.Errorf("a retryable failure marked items failed %+v", job)
	}
	for i := 0; i < 3; i++ {
		f.advance(time.Minute)
		f.s.drainCleanup()
	}
	f.s.fault = nil
	counts := f.job(root, id)["counts"].(map[string]any)
	if counts["failed"] != 1.0 || counts["succeeded"] != 1.0 {
		t.Errorf("a final failure reached past its task %+v", counts)
	}
	for _, item := range f.items(root, id, "") {
		if item["outcome"] == "succeeded" {
			for _, c := range item["cleanup"].([]any) {
				if c.(map[string]any)["item"] == "B2" && c.(map[string]any)["reason"] != "sessions_ended" {
					t.Errorf("a live cleanup reads as a restart %+v", c)
				}
			}
		}
	}
}

func TestCleanupDropsOnlyThePersonsPickups(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	member := f.withMember()
	f.as(Admin2Login)
	f.s.domain.Lock()
	var own identity.Verifier
	for _, v := range f.s.sessionsOf(SampleMember) {
		own = v
	}
	other := f.s.sessionsOf(SampleAdmin2)
	f.s.dir.pickups["mine"] = pickup{session: own, target: SampleAdmin3, expires: f.s.clock().Add(time.Hour)}
	f.s.dir.pickups["theirs"] = pickup{session: other[0], target: SampleAdmin3, expires: f.s.clock().Add(time.Hour)}
	f.s.domain.Unlock()
	_ = member
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember)
	f.execute(root, id, digest)
	f.s.drainCleanup()
	f.s.domain.Lock()
	_, mine := f.s.dir.pickups["mine"]
	_, theirs := f.s.dir.pickups["theirs"]
	f.s.domain.Unlock()
	if mine || !theirs {
		t.Errorf("pickups after cleanup: the person's %v, another's %v", mine, theirs)
	}
}

func TestRestartAppliesAConfirmedUserReinstatement(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendUser, Target: SampleAdmin3, TargetVersion: 1, Revocation: "bulk-revocation-4:0",
		Request: "bulk-revocation-4", RequestDigest: digestOf("x"), RequestedBy: SampleAdmin})
	intent := f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseIntent, Target: SampleAdmin3, TargetVersion: 2, Increase: decisionledger.IncreaseReinstateUser,
		RequestedBy: SampleAdmin})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseConfirmed, Target: SampleAdmin3, IntentSequence: intent})
	g := f.restart(nil)
	if m := g.membership(SampleAdmin3); m.UserState != identity.Active {
		t.Errorf("a confirmed user reinstatement was not applied %+v", m)
	}
}

func TestBulkImpactCountsOnlyThePersonsOpenProposals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	p := f.trustAdmins()
	f.trustPropose(p.b, "60", "0")
	f.trustPropose(p.a, "61", "0")
	f.requestGrant(f.as(Admin2Login), SampleMember, nil)
	f.requestGrant(root, SampleAdmin3, nil)
	id, _ := f.prepared(t, root, "suspend_user", SampleAdmin2)
	impact := f.bulkRead(root, "/"+id).body["impact"].(map[string]any)
	counts, rechecked := map[string]int{}, map[string]bool{}
	for _, row := range impact["pending"].([]any) {
		r := row.(map[string]any)
		counts[r["store"].(string)], rechecked[r["store"].(string)] = whole(r["count"]), r["rechecked"] == true
	}
	if counts["trust_changes"] != 1 || counts["access_requests"] != 1 {
		t.Errorf("another person's proposals were counted %+v", counts)
	}
	for _, store := range proposalStores {
		if rechecked[store.name] != (store.name == "trust_changes" || store.name == "access_requests") {
			t.Errorf("%s rechecked %v", store.name, rechecked[store.name])
		}
	}
}

func TestChangedSinceComparesEveryField(t *testing.T) {
	base := bulkrevoke.Target{User: "u", Role: "member", Version: 2, Epoch: 3, UserState: string(identity.Active), MembershipState: string(identity.MembershipActive)}
	r := func(edit func(u *identity.UserInfo, m *identity.MemberInfo)) roster {
		u := identity.UserInfo{ID: "u", State: identity.Active, Epoch: 3}
		m := identity.MemberInfo{User: "u", Role: identity.Member, State: identity.MembershipActive, Version: 2, Effective: true}
		edit(&u, &m)
		return roster{users: map[string]identity.UserInfo{"u": u}, members: map[string]identity.MemberInfo{"u": m}}
	}
	if changedSince(base, r(func(*identity.UserInfo, *identity.MemberInfo) {})) {
		t.Error("an unchanged target")
	}
	for name, edit := range map[string]func(u *identity.UserInfo, m *identity.MemberInfo){
		"epoch":      func(u *identity.UserInfo, _ *identity.MemberInfo) { u.Epoch++ },
		"user state": func(u *identity.UserInfo, _ *identity.MemberInfo) { u.State = identity.Suspended },
		"version":    func(_ *identity.UserInfo, m *identity.MemberInfo) { m.Version++ },
		"membership": func(_ *identity.UserInfo, m *identity.MemberInfo) { m.Effective = false },
		"role":       func(_ *identity.UserInfo, m *identity.MemberInfo) { m.Role = identity.Administrator },
	} {
		if !changedSince(base, r(edit)) {
			t.Errorf("a changed %s was not seen", name)
		}
	}
	if !changedSince(base, roster{users: map[string]identity.UserInfo{}, members: map[string]identity.MemberInfo{}}) {
		t.Error("a missing target")
	}
}

func TestCleanupProgressCountsSettledItems(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember, SampleAdmin3)
	f.execute(root, id, digest)
	cleanup := func() map[string]any { return f.job(root, id)["cleanup"].(map[string]any) }
	if c := cleanup(); c["done"] != 0.0 || c["total"] != 2.0 {
		t.Errorf("before cleanup %+v", c)
	}
	f.s.drainCleanup()
	if c := cleanup(); c["done"] != 2.0 || c["total"] != 2.0 {
		t.Errorf("after cleanup %+v", c)
	}
}

func startCount(t *testing.T, f world) (*Server, uint64) {
	t.Helper()
	f.s.Close()
	s, err := New(f.s.cfg)
	if err != nil {
		t.Fatal(err)
	}
	s.manual = true
	if err := s.Bind(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8820}); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, s.counter
}

func TestReplayCountsAndNotesWhatItReapplied(t *testing.T) {
	plain := setup(t, SeedSample).ready()
	_, base := startCount(t, plain)
	f := setup(t, SeedSample).ready()
	seq := f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendMembership, Target: SampleAdmin3, TargetVersion: 1, Revocation: "bulk-revocation-4:0",
		Request: "bulk-revocation-4", RequestDigest: digestOf("x"), RequestedBy: SampleAdmin})
	s, n := startCount(t, f)
	if n != base+1 || s.replayNotes[itoa(int(seq))] != noteReapplied {
		t.Errorf("a re-applied suspension: steps %d of %d, note %q", n, base, s.replayNotes[itoa(int(seq))])
	}
	g := setup(t, SeedSample).ready()
	g.craft(t, decisionledger.Decision{Decision: decisionledger.RolePeriodSet, Target: SampleAdmin3, TargetVersion: 1, Role: "administrator",
		Until: stamp(g.s.clock().Add(time.Hour)), RequestedBy: SampleAdmin})
	if _, n := startCount(t, g); n != base+1 {
		t.Errorf("a re-applied end: steps %d of %d", n, base)
	}
}

func TestReplayWithoutAnyAnchorBeforeDoesNotStrand(t *testing.T) {
	f := setup(t, SeedNone).ready()
	f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendMembership, Target: "user-0000000000000001", TargetVersion: 1, Revocation: "bulk-revocation-4:0",
		Request: "bulk-revocation-4", RequestDigest: digestOf("x"), RequestedBy: "user-0000000000000002"})
	s, _ := startCount(t, f)
	if len(s.stranded) != 0 {
		t.Errorf("stranded with no anchor before replay %v", s.stranded)
	}
}

func TestAnOmittedEndStillHonoursTheSettersEnd(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.ok(t, "an end on the second administrator", f.period(root, SampleAdmin2, f.at(2*time.Hour)), "role_period_set", "")
	second := f.as(Admin2Login)
	f.fieldRefused(t, "a promotion keeping no end", f.run(second, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}), fieldUntil)
	f.ok(t, "a later end on the member", f.period(root, SampleMember, f.at(3*time.Hour)), "role_period_set", "")
	f.fieldRefused(t, "a promotion keeping a later end", f.run(second, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}), fieldUntil)
	f.ok(t, "an earlier end on the member", f.period(root, SampleMember, f.at(time.Hour)), "role_period_set", "")
	if got := f.run(second, "identity.change_role", SampleMember, map[string]string{"role": "administrator"}); got.status != 200 {
		t.Errorf("a promotion keeping an end within the setter's %s", got.raw)
	}
}

func TestTargetsReadRefusesAnUnknownKind(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	id, _ := f.prepared(t, root, "suspend_membership", SampleMember)
	if got := f.bulkRead(root, "/"+id+"/targets?kind=held"); got.status != 422 && got.status != 400 || got.body["code"] != string(commands.InvalidRequest) {
		t.Errorf("an unknown kind %d %s", got.status, got.raw)
	}
}
