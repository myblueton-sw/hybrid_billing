package server

import (
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/bulkrevoke"
	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/decisionledger"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/jobs"
)

func TestBulkExecuteWhileTheLedgerCannotRecord(t *testing.T) {
	for name, prepare := range map[string]func(world){
		"stopped":  func(f world) { f.stoppedLedger() },
		"missing":  func(f world) { f.s.ledger = nil },
		"isolated": func(f world) { f.s.ledger.Close(); f.s.ledger = nil },
	} {
		t.Run(name, func(t *testing.T) {
			f := setup(t, SeedSample).ready()
			root := f.as(AdminLogin)
			f.withMember()
			id, digest := f.prepared(t, root, "suspend_membership", SampleMember)
			prepare(f)
			if got := f.execute(root, id, digest); got.status != 202 {
				t.Fatalf("execute %s", got.raw)
			}
			if m := f.membership(SampleMember); m.State != identity.MembershipSuspended {
				t.Errorf("the deny was not applied %+v", m)
			}
			job := f.job(root, id)
			if job["job_state"] != "waiting" || job["job_reason"] != bulkrevoke.ReasonUnconfirmed || job["ledger"].(map[string]any)["reason"] == "" {
				t.Errorf("job %+v", job)
			}
			if items := f.items(root, id, ""); items[0]["state"] != bulkrevoke.Unconfirmed {
				t.Errorf("items %+v", items)
			}
			if f.s.drainCleanup() != 0 {
				t.Error("cleanup ran for an unconfirmed item")
			}
			if row := f.row(root, SampleMember); row["membership"].(map[string]any)["ledger"] != ledgerNot {
				t.Errorf("row %+v", row)
			}
		})
	}
}

func (f world) manyMembers(t *testing.T, n int) []string {
	t.Helper()
	admin := verifierOf(random(tokenBytes))
	f.must(f.s.identity.Boundary().OpenSession(AdminTriple(), admin, 30*time.Minute, time.Hour))
	out := []string{}
	for i := 0; i < n; i++ {
		user := "user-" + leftPad(strings.Repeat("a", 1)+itoa(i), 16)
		login := user + "@example.invalid"
		f.must(f.s.identity.Installation().CreateUser(seedOperator, user, LoginTriple(login), login))
		f.s.dir.add(user, login, login)
		inv, secret, session := "inv-"+user, verifierOf(random(tokenBytes)), verifierOf(random(tokenBytes))
		f.must(f.s.identity.Workspace().Invite(admin, SampleWorkspace, inv, identity.Member, secret, identity.Validity{NotBefore: f.s.clock(), NotAfter: f.s.clock().Add(time.Hour)}, user))
		f.must(f.s.identity.Boundary().OpenSession(LoginTriple(login), session, 30*time.Minute, time.Hour))
		f.must(f.s.identity.Workspace().Accept(SampleWorkspace, inv, secret, session))
		f.must(f.s.identity.Boundary().EndSession(session))
		out = append(out, user)
	}
	return out
}

func itoa(i int) string {
	digits := "0123456789"
	if i < 10 {
		return digits[i : i+1]
	}
	return itoa(i/10) + digits[i%10:i%10+1]
}

func TestBulkLedgerChunksAndCleanupTasks(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	users := f.manyMembers(t, 300)
	big := f.stage(root, "", users[:180]...)
	f.ok(t, "second batch", f.stage(root, field(big, "draft"), users[180:]...), "draft", "")
	prepared := f.prepare(root, field(big, "draft"), "suspend_membership")
	id := field(prepared, "bulk")
	syncs := 0
	f.s.fault = func(at string) bool {
		if at == "ledger-sync" {
			syncs++
			return syncs == 3
		}
		return false
	}
	got := f.execute(root, id, field(prepared, "digest"))
	f.s.fault = nil
	if got.status != 202 {
		t.Fatalf("execute %s", got.raw)
	}
	job := f.job(root, id)
	if job["counts"].(map[string]any)["targets"] != 300.0 || job["job_state"] != "waiting" || job["ledger"].(map[string]any)["reason"] != decisionledger.ReasonCommitUnknown {
		t.Errorf("job %+v", job)
	}
	if got := f.items(root, id, "?after=299&limit=1"); got[0]["receipt"].(map[string]any)["reason"] != decisionledger.ReasonCommitUnknown {
		t.Errorf("the last item %+v", got)
	}
	if n := f.s.drainCleanup(); n != 3 {
		t.Errorf("tasks for 256 confirmed items %d", n)
	}
	staged := f.stage(root, "", users[0])
	f.failed(t, "nobody left to revoke", f.prepare(root, field(staged, "draft"), "suspend_membership"), commands.PrerequisiteFailed)
}

func TestBulkCleanupRetriesAndFencing(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	member := f.signedIn(MemberLogin, pw)
	f.s.dir.pickups["p1"] = pickup{target: SampleMember, expires: f.s.clock().Add(time.Hour)}
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember)
	if got := f.execute(root, id, digest); got.status != 202 {
		t.Fatal(got.raw)
	}
	f.ok(t, "reinstate before cleanup", f.run(root, "identity.reinstate_membership", SampleMember, nil), "membership_active", "")
	f.advance(time.Second)
	after := f.signedIn(MemberLogin, pw)
	f.s.fault = func(at string) bool { return at == "cleanup" }
	f.s.drainCleanup()
	if job := f.job(root, id); job["job_state"] != "running" {
		t.Errorf("after one failure %+v", job)
	}
	f.advance(time.Minute)
	f.s.drainCleanup()
	f.advance(time.Minute)
	f.s.drainCleanup()
	f.s.fault = nil
	job := f.job(root, id)
	if job["job_state"] != "succeeded" || job["counts"].(map[string]any)["failed"] != 1.0 {
		t.Errorf("after the budget %+v", job)
	}
	g := setup(t, SeedSample).ready()
	groot := g.as(AdminLogin)
	g.withMember()
	before := g.signedIn(MemberLogin, pw)
	g.s.dir.pickups["p1"] = pickup{target: SampleMember, expires: g.s.clock().Add(time.Hour)}
	gid, gdigest := g.prepared(t, groot, "suspend_membership", SampleMember)
	g.execute(groot, gid, gdigest)
	g.ok(t, "reinstate", g.run(groot, "identity.reinstate_membership", SampleMember, nil), "membership_active", "")
	g.advance(time.Second)
	kept := g.signedIn(MemberLogin, pw)
	g.ok(t, "suspend the executor's peer", g.run(g.as(Admin2Login), "identity.suspend_user", SampleAdmin, nil), "user_suspended", "")
	g.s.drainCleanup()
	if got := g.me(before); got.status != 401 {
		t.Errorf("a session opened before the deny survived %+v", got)
	}
	if got := g.me(kept); got.status != 200 {
		t.Errorf("a session opened after a reinstatement was ended %+v", got)
	}
	if _, ok := g.s.dir.pickups["p1"]; ok {
		t.Error("the pickup survived")
	}
	if job := g.job(g.as(Admin2Login), gid); job["job_state"] != "succeeded" {
		t.Errorf("cleanup stopped with the executor %+v", job)
	}
	_ = member
	_ = after
}

func TestCleanupSkipsHeldItemsInsideAChunk(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	id, digest := f.prepared(t, root, "suspend_membership", SampleAdmin2, SampleAdmin3, SampleMember)
	other, odigest := f.prepared(t, f.as(Admin2Login), "suspend_membership", SampleMember)
	f.ok(t, "a change in the middle", f.run(root, "identity.change_role", SampleAdmin3, map[string]string{"role": "member"}), "role_member", "")
	if got := f.execute(root, id, digest); got.status != 202 || field(got, "held") != "1" {
		t.Fatal(got.raw)
	}
	f.execute(root, other, odigest)
	f.s.drainCleanup()
	if job := f.job(root, id); job["job_state"] != "succeeded" || job["counts"].(map[string]any)["succeeded"] != 2.0 {
		t.Errorf("job %+v", job)
	}
	draft := field(f.stage(root, "", SampleMember), "draft")
	f.s.fault = func(at string) bool { return at == "users" }
	f.failed(t, "prepare without the users", f.runWith(root, f.s.Revision(), "bulk_revocation.prepare", draft, map[string]string{"effect": "suspend_user"}),
		commands.DependencyUnavailable)
	f.s.fault = nil
}

func TestBulkCleanupNotStartedAndStaleLeases(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.withMember()
	id, digest := f.prepared(t, root, "suspend_membership", SampleMember)
	f.s.fault = func(at string) bool { return at == "cleanup-create" }
	got := f.execute(root, id, digest)
	f.s.fault = nil
	if got.status != 202 {
		t.Fatal(got.raw)
	}
	if job := f.job(root, id); job["job_state"] != "waiting" || job["job_reason"] != bulkrevoke.ReasonNotStarted {
		t.Errorf("not started %+v", job)
	}
	g := setup(t, SeedSample).ready()
	groot := g.as(AdminLogin)
	g.withMember()
	gid, gdigest := g.prepared(t, groot, "suspend_membership", SampleMember)
	g.execute(groot, gid, gdigest)
	leases, err := g.s.jobs.Lease(cleanupWorker, []string{cleanupKind}, 1)
	if err != nil || len(leases) != 1 {
		t.Fatal(err)
	}
	g.advance(time.Minute)
	g.s.jobs.Sweep()
	g.s.cleanTask(leases[0])
	if items := g.items(groot, gid, ""); items[0]["state"] != bulkrevoke.DenyConfirmed {
		t.Errorf("a stale lease cleaned %+v", items)
	}
	g.advance(time.Minute)
	g.s.drainCleanup()
	if items := g.items(groot, gid, ""); items[0]["state"] != bulkrevoke.Complete {
		t.Errorf("after a fresh lease %+v", items)
	}
	_, err = g.s.jobs.Create(SampleWorkspace, "bulk-revocation-77", cleanupKind, cleanupPolicy, []jobs.TaskSpec{{ID: "chunk-1", Input: "0:0", Repeatable: true}})
	if err != nil {
		t.Fatal(err)
	}
	g.s.drainCleanup()
	if v, _ := g.s.jobs.Job(SampleWorkspace, "bulk-revocation-77"); v.State != jobs.Failed {
		t.Errorf("a task without its job %+v", v.State)
	}
	h := setup(t, SeedSample).ready()
	hroot := h.as(AdminLogin)
	h.withMember()
	hid, hdigest := h.prepared(t, hroot, "suspend_membership", SampleMember)
	h.execute(hroot, hid, hdigest)
	lease, _ := h.s.jobs.Lease(cleanupWorker, []string{cleanupKind}, 1)
	h.s.fault = func(at string) bool { return at == "cleanup" }
	h.s.cleanTask(lease[0])
	h.s.fault = nil
	h.advance(time.Minute)
	again, _ := h.s.jobs.Lease(cleanupWorker, []string{cleanupKind}, 1)
	_ = h.s.jobs.Checkpoint(again[0].Ref(), "0")
	_, _ = h.s.jobs.Fail(again[0].Ref(), "x", false)
	h.advance(time.Minute)
	h.s.drainCleanup()
	if job := h.job(hroot, hid); job["job_state"] != "succeeded" {
		t.Errorf("a resumed task %+v", job)
	}
}
