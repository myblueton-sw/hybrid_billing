package server

import (
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
)

func (f world) craft(t *testing.T, d decisionledger.Decision) uint64 {
	t.Helper()
	f.s.domain.Lock()
	defer f.s.domain.Unlock()
	r, reason := f.s.commitIdentity(d)
	if reason != "" {
		t.Fatal(reason)
	}
	return r.Sequence
}

func TestRestartReappliesBulkRevocationsAndRebuildsJobs(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember, SampleAdmin3, SampleAdmin)
	if got := f.execute(root, id, digest); got.status != 202 {
		t.Fatal(got.raw)
	}
	f.ok(t, "reinstate the third", f.run(root, "identity.reinstate_membership", SampleAdmin3, nil), "membership_active", "")
	f.s.drainCleanup()
	g := f.restart(nil)
	if m := g.membership(SampleMember); m.User != "" {
		t.Errorf("the seed member before acceptance %+v", m)
	}
	g.withMember()
	if m := g.membership(SampleMember); m.State != identity.MembershipSuspended || m.SuspendedFor != identity.BulkRevocation {
		t.Errorf("the bulk-suspended member after restart %+v", m)
	}
	if m := g.membership(SampleAdmin3); m.State != identity.MembershipActive {
		t.Errorf("the reinstated administrator after restart %+v", m)
	}
	groot := g.as(AdminLogin)
	job := g.job(groot, id)
	counts := job["counts"].(map[string]any)
	if job["origin"] != bulkrevoke.OriginReplay || job["selection"] != bulkrevoke.NotRecorded || job["filter"] != nil || counts["targets"] != 2.0 || counts["excluded"] != 1.0 || counts["succeeded"] != 2.0 || job["job_state"] != "succeeded" ||
		job["recorded_at"] == "" {
		t.Errorf("rebuilt %+v", job)
	}
	items := g.items(groot, id, "")
	if len(items) != 2 || items[0]["replay_note"] != noteLater || items[1]["replay_note"] != noteAwaiting || items[0]["cleanup"].([]any)[1].(map[string]any)["reason"] != "server_restarted" {
		t.Errorf("rebuilt items %+v", items)
	}
	staged := g.stage(groot, "", SampleAdmin2)
	if field(staged, "draft") != "bulk-draft-2" {
		t.Errorf("numbers do not continue %s", staged.raw)
	}
}

func TestRestartReappliesRoleEndsAndSweeps(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	f.ok(t, "an end on the member", f.period(root, SampleMember, f.at(2*time.Hour)), "role_period_set", "")
	f.ok(t, "an end on the second", f.period(root, SampleAdmin2, f.at(5*time.Hour)), "role_period_set", "")
	f.advance(3 * time.Hour)
	g := f.restart(nil)
	if m := g.membership(SampleAdmin2); m.Until.IsZero() || !m.Effective {
		t.Errorf("a future end after restart %+v", m)
	}
	g.withMember()
	if m := g.membership(SampleMember); m.Effective || m.Until.IsZero() {
		t.Errorf("an end due while down is not in effect at acceptance %+v", m)
	}
	g.sweep()
	if m := g.membership(SampleMember); m.State != identity.MembershipSuspended || m.SuspendedFor != identity.RolePeriodEnded {
		t.Errorf("swept %+v", m)
	}
	if ended := g.records(t, decisionledger.RolePeriodEnded); len(ended) != 1 || ended[0]["revocation"] != "role-end-1" {
		t.Fatalf("ended %+v", ended)
	}
	g.advance(3 * time.Hour)
	g.sweep()
	if ended := g.records(t, decisionledger.RolePeriodEnded); len(ended) != 2 || ended[1]["revocation"] != "role-end-2" {
		t.Errorf("counters %+v", ended)
	}
	h := g.restart(nil)
	if m := h.membership(SampleAdmin2); m.State != identity.MembershipSuspended || m.SuspendedFor != identity.RolePeriodEnded {
		t.Errorf("a swept end after restart %+v", m)
	}
	h.advance(time.Hour)
	i := h.restart(nil)
	if m := i.membership(SampleAdmin2); m.State != identity.MembershipSuspended {
		t.Errorf("a third start %+v", m)
	}
}

func TestRestartRoleGuardAndAnchorKeeping(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.withMember()
	until := stamp(f.s.clock().Add(24 * time.Hour))
	f.craft(t, decisionledger.Decision{Decision: decisionledger.RolePeriodSet, Target: SampleMember, TargetVersion: 2, Role: "administrator", Until: until, RequestedBy: SampleAdmin})
	for _, user := range []string{SampleAdmin3, SampleAdmin, SampleAdmin2} {
		f.craft(t, decisionledger.Decision{Decision: decisionledger.RolePeriodSet, Target: user, TargetVersion: 1, Role: "administrator", Until: until, RequestedBy: "user-gone"})
	}
	g := f.restart(nil)
	g.withMember()
	if m := g.membership(SampleMember); !m.Until.IsZero() {
		t.Errorf("a role end of a different role was applied %+v", m)
	}
	if m := g.membership(SampleAdmin); !m.Until.IsZero() {
		t.Errorf("the first administrator was not kept without an end %+v", m)
	}
	for _, user := range []string{SampleAdmin2, SampleAdmin3} {
		if m := g.membership(user); m.Until.IsZero() {
			t.Errorf("%s lost its end %+v", user, m)
		}
	}
	notes := map[string]int{}
	for _, note := range g.s.replayNotes {
		notes[note]++
	}
	if notes[noteRole] != 1 || notes[noteAnchor] != 1 || notes[noteReapplied] != 2 || len(g.s.stranded) != 0 {
		t.Errorf("notes %v stranded %v", notes, g.s.stranded)
	}
}

func TestRestartIntoOperatorRecovery(t *testing.T) {
	f := setup(t, SeedSample).ready()
	for _, user := range []string{SampleAdmin, SampleAdmin2, SampleAdmin3} {
		f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendUser, Target: user, TargetVersion: 1, Revocation: "bulk-revocation-9:0",
			Request: "bulk-revocation-9", RequestDigest: digestOf("x"), RequestedBy: "user-gone"})
	}
	g := f.restart(nil)
	if len(g.s.stranded) != 3 {
		t.Fatalf("stranded %v", g.s.stranded)
	}
	rows := g.s.clockRows(g.s.clock())
	if !rows.Recovery.Stranded || len(rows.Recovery.Records) != 3 {
		t.Errorf("rows %+v", rows.Recovery)
	}
	v := verifierOf(random(tokenBytes))
	g.s.sessions.put(v, SampleWorkspace, AdminLogin, g.s.clock(), g.s.currentEpoch(SampleWorkspace))
	result := g.s.execute("identity.suspend_user", commands.Call{Command: "identity.suspend_user", Request: "r1"}, v)
	if result.Err == nil || result.Err.Code != commands.DependencyUnavailable {
		t.Errorf("a command in operator recovery %+v", result)
	}
	if got := g.s.execute("bulk_revocation.discard", commands.Call{Command: "bulk_revocation.discard", Request: "r2"}, v); got.Err == nil || got.Err.Code != commands.DependencyUnavailable {
		t.Errorf("a bulk command in operator recovery %+v", got)
	}
}

func digestOf(s string) string {
	out := []byte{}
	for len(out) < 64 {
		out = append(out, 'a')
	}
	return string(out)
}

func TestRestartIncreasesNeedTheirConfirmation(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.withMember()
	for _, user := range []string{SampleMember, SampleAdmin3} {
		f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendMembership, Target: user, TargetVersion: 1, Revocation: "bulk-revocation-4:0",
			Request: "bulk-revocation-4", RequestDigest: digestOf("x"), RequestedBy: SampleAdmin})
	}
	confirmed := f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseIntent, Target: SampleMember, TargetVersion: 2, Increase: decisionledger.IncreaseReinstateMembership,
		RequestedBy: SampleAdmin})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseConfirmed, Target: SampleMember, IntentSequence: confirmed})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseIntent, Target: SampleAdmin3, TargetVersion: 2, Increase: decisionledger.IncreaseReinstateMembership,
		RequestedBy: SampleAdmin})
	aborted := f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseIntent, Target: SampleAdmin2, TargetVersion: 1, Increase: decisionledger.IncreaseRolePeriod,
		Role: "administrator", Until: stamp(f.s.clock().Add(time.Hour)), RequestedBy: SampleAdmin})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseAborted, Target: SampleAdmin2, IntentSequence: aborted})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseConfirmed, Target: SampleAdmin2, IntentSequence: aborted})
	g := f.restart(nil)
	g.withMember()
	if m := g.membership(SampleMember); m.State != identity.MembershipActive {
		t.Errorf("a confirmed reinstatement was not applied %+v", m)
	}
	if m := g.membership(SampleAdmin3); m.State != identity.MembershipSuspended {
		t.Errorf("an unconfirmed reinstatement was applied %+v", m)
	}
	if m := g.membership(SampleAdmin2); !m.Until.IsZero() {
		t.Errorf("an aborted increase was applied %+v", m)
	}
	g.breakLine(t, len(g.ledgerLines(t)))
	h := g.restart(nil)
	h.withMember()
	if m := h.membership(SampleMember); m.State != identity.MembershipSuspended {
		t.Errorf("degraded replay applied a confirmed increase %+v", m)
	}
	if h.s.ledger.Status().State != decisionledger.StateIsolated {
		t.Errorf("state %+v", h.s.ledger.Status())
	}
}

func TestReplayOfCraftedIdentityRecords(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.withMember()
	end := stamp(f.s.clock().Add(time.Hour))
	f.craft(t, decisionledger.Decision{Decision: decisionledger.RolePeriodSet, Target: SampleAdmin2, TargetVersion: 1, Role: "member", Until: end, RequestedBy: SampleAdmin})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendUser, Target: "user-gone", TargetVersion: 1, Revocation: "bulk-revocation-3:0",
		Request: "bulk-revocation-3", RequestDigest: digestOf("x"), RequestedBy: SampleAdmin})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.BulkRevocationJob, Target: "bulk-x", RequestDigest: digestOf("x"), Effect: "suspend_user",
		RequestedBy: SampleAdmin, ApprovedBy: SampleAdmin2, Targets: 1})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.BulkRevocationJob, Target: "bulk-revocation-5", RequestDigest: digestOf("x"), Effect: "suspend_user",
		RequestedBy: SampleAdmin, ApprovedBy: SampleAdmin2, Targets: 3, Held: 1})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendUser, Target: SampleAdmin3, TargetVersion: 1, Revocation: "bulk-revocation-5:x",
		Request: "bulk-revocation-5", RequestDigest: digestOf("x"), RequestedBy: SampleAdmin2})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.SuspendUser, Target: SampleAdmin3, TargetVersion: 1, Revocation: "other:0",
		Request: "bulk-revocation-5", RequestDigest: digestOf("x"), RequestedBy: SampleAdmin2})
	f.craft(t, decisionledger.Decision{Decision: decisionledger.RolePeriodEnded, Target: SampleMember, TargetVersion: 2, Role: "member", Until: end,
		Revocation: "role-end-4", RequestedBy: decisionledger.RoleSystem})
	for _, until := range []string{decisionledger.NoEnd, stamp(f.s.clock().Add(48 * time.Hour))} {
		intent := f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseIntent, Target: SampleMember, TargetVersion: 3, Increase: decisionledger.IncreaseRolePeriod,
			Role: "member", Until: until, RequestedBy: SampleAdmin})
		f.craft(t, decisionledger.Decision{Decision: decisionledger.IncreaseConfirmed, Target: SampleMember, IntentSequence: intent})
	}
	g := f.restart(nil)
	if m := g.membership(SampleAdmin2); !m.Until.IsZero() {
		t.Errorf("a role end of another role %+v", m)
	}
	g.withMember()
	if m := g.membership(SampleMember); m.State != identity.MembershipActive || m.Until.IsZero() {
		t.Errorf("a confirmed restoring period %+v", m)
	}
	job := g.job(g.as(AdminLogin), "bulk-revocation-5")
	if job["counts"].(map[string]any)["unprocessed"] != 2.0 || job["ledger"].(map[string]any)["reason"] != bulkrevoke.RecordNotConfirmed {
		t.Errorf("a job without item records %+v", job)
	}
	if g.s.roleEnds != 4 {
		t.Errorf("role ends %d", g.s.roleEnds)
	}
	if _, ok := g.s.bulk.Get(SampleWorkspace, SampleAdmin, "bulk-x"); ok {
		t.Error("a malformed job was rebuilt")
	}
}
