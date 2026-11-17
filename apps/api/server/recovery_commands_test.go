package server

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/recovery"
)

func (f world) lostRefused(t *testing.T, name string, got reply) {
	t.Helper()
	f.unchanged(t, name, got, "invalid_request")
	if fmt.Sprint(got.body["fields"]) != "[lost]" {
		t.Errorf("%s fields %s", name, got.raw)
	}
}

func (f world) sameAsStatus(t *testing.T, name string) {
	t.Helper()
	read := f.recoveryRead(f.as(AdminLogin))
	for _, id := range []string{recoveryPrimary, recoveryStandby} {
		if got, want := f.isolationOf(read, id), f.statusOf(id); got != want {
			t.Errorf("%s: %s reads %s, status %s", name, id, got, want)
		}
		if set := sectionRow(t, read, "sets", id); fmt.Sprint(set["state"], " ", set["generation"]) != strings.Join(strings.Fields(f.statusOf(id))[:2], " ") {
			t.Errorf("%s: %s set %v", name, id, set)
		}
	}
}

func TestRecoveryLockReleaseCycle(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	standby := installTargetOf(recoveryStandby)
	f.ok(t, "approve release", f.run(root, "recovery.approve", proposalTargetOf(10), nil), "applied", "map[proposal:10 target:recovery-proposal-10]")
	if s := f.statusOf(recoveryStandby); s != "serving 2 " {
		t.Errorf("released %s", s)
	}
	f.sameAsStatus(t, "released")
	read := f.recoveryRead(second)
	if s := unlockState(sectionRow(t, read, "resume", recoveryStandby)) + "|" + recoveryBlocked(recoveryProposalRowOf(t, read, 10)); s !=
		"none not_recovering false not_recovering|false not_pending false not_pending" {
		t.Errorf("after release %s", s)
	}
	if iso := sectionRow(t, read, "isolation", recoveryStandby); iso["can_lock"] != true || iso["lock_blocked"] != "" || iso["stopped_at"] != "" {
		t.Errorf("isolation %v", iso)
	}
	release := recoveryProposalRowOf(t, read, 10)
	if s := fmt.Sprint(release["approver"], " ", release["closed_by"], " ", release["closing_rationale"]); s != "sample-admin sample-admin sample reason" {
		t.Errorf("closed %s", s)
	}
	f.unchanged(t, "approve applied by its approver", f.run(root, "recovery.approve", proposalTargetOf(10), nil), "invalid_stage")
	f.unchanged(t, "approve applied by another", f.run(second, "recovery.approve", proposalTargetOf(10), nil), "invalid_stage")
	f.unchanged(t, "withdraw applied by a non-proposer", f.run(root, "recovery.withdraw", proposalTargetOf(10), nil), "invalid_stage")
	f.unchanged(t, "unlock while serving", f.run(root, "recovery.propose_unlock", standby, map[string]string{"lost": sampleLost}), "invalid_stage")
	f.unchanged(t, "state before digest", f.run(root, "recovery.propose_unlock", standby, map[string]string{"lost": "nope"}), "invalid_stage")
	f.advance(time.Minute)
	f.ok(t, "lock", f.run(root, "recovery.lock", standby, nil), "locked", "map[installation:sample-standby target:recovery-installation:sample-standby]")
	if s := f.statusOf(recoveryStandby); s != "locked 3 2026-11-16T09:01:00Z" {
		t.Errorf("locked %s", s)
	}
	f.sameAsStatus(t, "locked")
	f.unchanged(t, "lock locked", f.run(second, "recovery.lock", standby, nil), "invalid_stage")
	f.lostRefused(t, "malformed lost", f.run(root, "recovery.propose_unlock", standby, map[string]string{"lost": "ABC"}))
	f.lostRefused(t, "release with a lost digest", f.run(root, "recovery.propose_unlock", standby, map[string]string{"lost": sampleHash("other")}))
	proposed := f.run(second, "recovery.propose_unlock", standby, map[string]string{"lost": sampleLost})
	f.ok(t, "propose", proposed, "pending", "map[proposal:11 target:recovery-proposal-11]")
	if v, err := f.s.recovery.Proposal(11); err != nil || v.Proposer != SampleAdmin2 || v.Change.Unlock.Generation != 3 || v.Change.Unlock.Lost != sampleLost {
		t.Errorf("proposal %+v %v", v, err)
	}
	f.unchanged(t, "same proposal again", f.run(second, "recovery.propose_unlock", standby, map[string]string{"lost": sampleLost}), "invalid_stage")
	f.unchanged(t, "proposal by the other", f.run(root, "recovery.propose_unlock", standby, map[string]string{"lost": sampleLost}), "invalid_stage")
	read = f.recoveryRead(root)
	if s := unlockState(sectionRow(t, read, "resume", recoveryStandby)) + "|" + recoveryBlocked(recoveryProposalRowOf(t, read, 11)) + "|" +
		recoveryBlocked(recoveryProposalRowOf(t, f.recoveryRead(second), 11)); s != "lock_release not_recovering false already_proposed|true  false not_proposer|false own_proposal true " {
		t.Errorf("pending %s", s)
	}
	if locks := sectionRow(t, read, "isolation", recoveryStandby)["locks"].([]any); len(locks) != 2 || fmt.Sprint(locks[1]) !=
		"map[at:2026-11-16T09:01:00Z by:sample-admin by_login:admin@example.invalid rationale:sample reason]" {
		t.Errorf("locks %v", locks)
	}
	f.unchanged(t, "self approval", f.run(second, "recovery.approve", proposalTargetOf(11), nil), "prerequisite_failed")
	f.unchanged(t, "not proposer", f.run(root, "recovery.withdraw", proposalTargetOf(11), nil), "prerequisite_failed")
	f.ok(t, "withdraw", f.run(second, "recovery.withdraw", proposalTargetOf(11), nil), "withdrawn", "map[proposal:11 target:recovery-proposal-11]")
	gone := recoveryProposalRowOf(t, f.recoveryRead(root), 11)
	if s := fmt.Sprint(gone["state"], " ", gone["closed_by"], " ", recoveryBlocked(gone)); s != "withdrawn sample-admin-2 false not_pending false not_pending" {
		t.Errorf("withdrawn %s", s)
	}
	f.unchanged(t, "approve after withdraw", f.run(root, "recovery.approve", proposalTargetOf(11), nil), "invalid_stage")
	f.unchanged(t, "withdraw after withdraw", f.run(second, "recovery.withdraw", proposalTargetOf(11), nil), "invalid_stage")
	if f.s.rec.entry(11).view.State != recovery.ProposalWithdrawn {
		t.Error("a withdrawn entry was retired")
	}
	f.ok(t, "propose again", f.run(root, "recovery.propose_unlock", standby, map[string]string{"lost": sampleLost}), "pending", "map[proposal:12 target:recovery-proposal-12]")
	f.ok(t, "approve again", f.run(second, "recovery.approve", proposalTargetOf(12), nil), "applied", "")
	if s := f.statusOf(recoveryStandby); s != "serving 3 " {
		t.Errorf("released again %s", s)
	}
	f.sameAsStatus(t, "released again")
	f.ok(t, "lock again", f.run(second, "recovery.lock", standby, nil), "locked", "")
	if s := f.statusOf(recoveryStandby); s != "locked 4 2026-11-16T09:01:00Z" {
		t.Errorf("locked again %s", s)
	}
	f.sameAsStatus(t, "locked again")
}

func TestRecoveryPrimaryRefusals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	primary := installTargetOf(recoveryPrimary)
	f.unchanged(t, "lock recovering", f.run(root, "recovery.lock", primary, nil), "invalid_stage")
	f.unchanged(t, "unlock incomplete", f.run(root, "recovery.propose_unlock", primary, map[string]string{"lost": sampleLost}), "invalid_stage")
	f.unchanged(t, "unlock incomplete with another digest", f.run(root, "recovery.propose_unlock", primary, map[string]string{"lost": sampleHash("x")}), "invalid_stage")
	f.lostRefused(t, "malformed lost while recovering", f.run(root, "recovery.propose_unlock", primary, map[string]string{"lost": "1"}))
	if s := f.statusOf(recoveryPrimary); s != "recovering 2 2026-11-16T09:00:00Z" {
		t.Errorf("primary %s", s)
	}
	if len(f.s.rec.entries) != 10 {
		t.Errorf("entries %d", len(f.s.rec.entries))
	}
}

func TestRecoveryStaleGenerationAndAbandon(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.moduleApprove(f.sampleBegin(recoveryStandby, 2))
	if s := f.statusOf(recoveryStandby); s != "recovering 3 2026-11-16T09:00:00Z" {
		t.Errorf("begin %s", s)
	}
	read := f.recoveryRead(root)
	if s := recoveryBlocked(recoveryProposalRowOf(t, read, 10)) + "|" + unlockState(sectionRow(t, read, "resume", recoveryStandby)); s !=
		"false generation false not_proposer|recovery_resume verify false already_proposed" {
		t.Errorf("stale %s", s)
	}
	f.unchanged(t, "approve stale", f.run(root, "recovery.approve", proposalTargetOf(10), nil), "invalid_stage")
	f.ok(t, "withdraw stale", f.run(second, "recovery.withdraw", proposalTargetOf(10), nil), "withdrawn", "")
	f.moduleApprove(recovery.Change{Kind: recovery.ChangeAbandon, Abandon: recovery.Abandon{Installation: recoveryStandby, Recovery: "recovery-1", Generation: 3}})
	read = f.recoveryRead(root)
	if s := unlockState(sectionRow(t, read, "resume", recoveryStandby)); s != "recovery_resume verify false no_restore_point" {
		t.Errorf("abandoned %s", s)
	}
	set := sectionRow(t, read, "sets", recoveryStandby)
	if s := fmt.Sprint(set["state"], " ", set["generation"], " ", set["recovery"].(map[string]any)["as_of"], " ", set["history"]); s != "recovering 4  [map[generation:4 lost_count:0 "+
		"manifest_digest:"+recovery.ManifestDigest(recoveryManifest(""))+" outcome:abandoned recovery:recovery-1]]" {
		t.Errorf("abandoned set %s", s)
	}
	f.unchanged(t, "unlock without a restore point", f.run(root, "recovery.propose_unlock", installTargetOf(recoveryStandby), map[string]string{"lost": sampleLost}), "invalid_stage")
}

func TestRecoveryOpenCallsBlockRelease(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	standby := installTargetOf(recoveryStandby)
	f.ok(t, "release", f.run(root, "recovery.approve", proposalTargetOf(10), nil), "applied", "")
	tok, err := f.s.recovery.Permit(recoveryStandby, recoveryStandbyW, standbyActor)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.recovery.Call(recoveryStandby, recoveryStandbyW, tok, recovery.Target{Workspace: SampleWorkspace, Key: "sample-delivery-1"}, standbyActor); err != nil {
		t.Fatal(err)
	}
	f.ok(t, "lock", f.run(root, "recovery.lock", standby, nil), "locked", "")
	read := f.recoveryRead(root)
	if s := unlockState(sectionRow(t, read, "resume", recoveryStandby)); s != "lock_release not_recovering false open_calls" {
		t.Errorf("open calls %s", s)
	}
	if iso := sectionRow(t, read, "isolation", recoveryStandby); iso["open_calls_source"] != "not_shown" {
		t.Errorf("isolation %v", iso)
	}
	f.unchanged(t, "unlock with open calls", f.run(root, "recovery.propose_unlock", standby, map[string]string{"lost": sampleLost}), "invalid_stage")
	revision := f.s.Revision()
	f.advance(2 * time.Minute)
	if s := unlockState(sectionRow(t, f.recoveryRead(root), "resume", recoveryStandby)); s != "lock_release not_recovering true " || f.s.Revision() != revision {
		t.Errorf("after the deadline %s", s)
	}
	f.ok(t, "unlock", f.run(root, "recovery.propose_unlock", standby, map[string]string{"lost": sampleLost}), "pending", "")
}

func TestRecoveryInjectedProposals(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	first := recovery.Actor{ID: SampleAdmin, Kind: recovery.KindHuman}
	prune, err := f.s.recoveryPropose(recovery.Change{Kind: recovery.ChangePrune, Prune: recovery.Prune{Installation: recoveryStandby, Before: start}}, first, "test")
	if err != nil {
		t.Fatal(err)
	}
	begin, err := f.s.recoveryPropose(f.sampleBegin(recoveryStandby, 2), first, "test")
	if err != nil {
		t.Fatal(err)
	}
	read, other := f.recoveryRead(root), f.recoveryRead(second)
	for _, e := range []*recoveryEntry{prune, begin} {
		if s := recoveryBlocked(recoveryProposalRowOf(t, read, e.view.Number)) + "|" + recoveryBlocked(recoveryProposalRowOf(t, other, e.view.Number)); s !=
			"false not_console_kind true |false not_console_kind false not_proposer" {
			t.Errorf("%s blockers %s", e.view.Change.Kind, s)
		}
	}
	called := false
	f.s.fault = func(step string) bool {
		called = called || step == "recovery-approve"
		return false
	}
	for _, e := range []*recoveryEntry{prune, begin} {
		f.unchanged(t, "approve "+e.view.Change.Kind, f.run(second, "recovery.approve", proposalTargetOf(e.view.Number), nil), "invalid_stage")
	}
	f.s.fault = nil
	if called {
		t.Error("approval of another kind called the module")
	}
	if v, err := f.s.recovery.Proposal(begin.view.Number); err != nil || v.State != recovery.ProposalPending {
		t.Errorf("begin %+v %v", v, err)
	}
	f.ok(t, "withdraw prune", f.run(root, "recovery.withdraw", proposalTargetOf(prune.view.Number), nil), "withdrawn", "")
	f.ok(t, "withdraw begin", f.run(root, "recovery.withdraw", proposalTargetOf(begin.view.Number), nil), "withdrawn", "")
	if begin, err = f.s.recoveryPropose(f.sampleBegin(recoveryStandby, 2), first, "test"); err != nil {
		t.Fatal(err)
	}
	f.ok(t, "release", f.run(root, "recovery.approve", proposalTargetOf(10), nil), "applied", "")
	f.ok(t, "lock retires the begin", f.run(root, "recovery.lock", installTargetOf(recoveryStandby), nil), "locked", "")
	if _, err := f.s.recovery.Proposal(begin.view.Number); err == nil {
		t.Fatal("the begin is still in the module")
	}
	if s := recoveryBlocked(recoveryProposalRowOf(t, f.recoveryRead(root), begin.view.Number)); s != "false not_console_kind true " {
		t.Errorf("before retirement %s", s)
	}
	f.unchanged(t, "approve retired", f.run(second, "recovery.approve", proposalTargetOf(begin.view.Number), nil), "invalid_stage")
	f.unchanged(t, "withdraw retired", f.run(root, "recovery.withdraw", proposalTargetOf(begin.view.Number), nil), "invalid_stage")
	row := recoveryProposalRowOf(t, f.recoveryRead(root), begin.view.Number)
	if s := fmt.Sprint(row["state"], " ", recoveryBlocked(row)); s != "retired false retired false retired" {
		t.Errorf("retired %s", s)
	}
	f.unchanged(t, "approve after retirement", f.run(second, "recovery.approve", proposalTargetOf(begin.view.Number), nil), "invalid_stage")
	f.unchanged(t, "withdraw after retirement", f.run(root, "recovery.withdraw", proposalTargetOf(begin.view.Number), nil), "invalid_stage")
	ghost, err := f.s.recoveryPropose(f.sampleBegin(recoveryStandby, 3), first, "test")
	if err != nil {
		t.Fatal(err)
	}
	f.moduleApprove(recovery.Change{Kind: recovery.ChangeUnlock, Unlock: recovery.Unlock{Installation: recoveryStandby, Generation: 3, Lost: sampleLost}})
	f.ok(t, "lock to retire", f.run(root, "recovery.lock", installTargetOf(recoveryStandby), nil), "locked", "")
	if err := f.s.recoveryApprove(ghost, recovery.Actor{ID: SampleAdmin2, Kind: recovery.KindHuman}, "x"); err == nil || ghost.view.State != recoveryRetired {
		t.Errorf("approve of a deleted proposal %v %s", err, ghost.view.State)
	}
}

func TestRecoveryLimitMapsToPrerequisite(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.ok(t, "withdraw seeded unlock", f.run(second, "recovery.withdraw", proposalTargetOf(10), nil), "withdrawn", "")
	first := recovery.Actor{ID: SampleAdmin, Kind: recovery.KindHuman}
	for index := 0; ; index++ {
		_, err := f.s.recovery.Propose(recovery.Change{Kind: recovery.ChangeWriter, Writer: recovery.Writer{Installation: recoveryStandby, ID: fmt.Sprintf("test-writer-%d", index),
			Kind: recovery.WriterERP, MaxCall: time.Minute}}, first)
		if err != nil {
			break
		}
	}
	if s := unlockState(sectionRow(t, f.recoveryRead(root), "resume", recoveryStandby)); s != "lock_release not_recovering true " {
		t.Errorf("read before the limit %s", s)
	}
	f.unchanged(t, "limit", f.run(root, "recovery.propose_unlock", installTargetOf(recoveryStandby), map[string]string{"lost": sampleLost}), "prerequisite_failed")
	f.ok(t, "the other administrator", f.run(second, "recovery.propose_unlock", installTargetOf(recoveryStandby), map[string]string{"lost": sampleLost}), "pending", "")
}

func TestRecoveryStaleRevisionAndReplay(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	old := f.revision(root)
	if got := f.runWith(root, old, "recovery.approve", proposalTargetOf(10), map[string]string{}); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.code(t, "stale", f.runWith(second, old, "recovery.lock", installTargetOf(recoveryStandby), map[string]string{}), 409, "revision_conflict")
	bodies := map[string]string{}
	send := func(who actor, command, target, key, lost string) reply {
		if bodies[key] == "" {
			bodies[key] = fmt.Sprintf(`{"request_id":"req-%s","target_id":"%s","idempotency_key":"%s","expected_revision":"%s","rationale":"r","body":{%s}}`, key, target, key,
				f.revision(who), lost)
		}
		return f.call(command, bodies[key], who.cookie, who.csrf)
	}
	twice := func(who actor, command, target, key, lost string, after func()) {
		t.Helper()
		first := send(who, command, target, key, lost)
		if after != nil {
			after()
		}
		revision := f.s.Revision()
		again := send(who, command, target, key, lost)
		if first.status != 200 || again.status != 200 || again.body["replay"] != true || fmt.Sprint(first.body["fields"]) != fmt.Sprint(again.body["fields"]) ||
			f.s.Revision() != revision {
			t.Fatalf("%s replay %s %s", command, first.raw, again.raw)
		}
	}
	standby := installTargetOf(recoveryStandby)
	lost := `"lost":"` + sampleLost + `"`
	twice(second, "recovery.lock", standby, "k-lock", "", nil)
	twice(root, "recovery.propose_unlock", standby, "k-propose", lost, func() {
		f.ok(t, "approve", f.run(second, "recovery.approve", proposalTargetOf(11), nil), "applied", "")
	})
	f.ok(t, "lock", f.run(second, "recovery.lock", standby, nil), "locked", "")
	f.ok(t, "propose", f.run(root, "recovery.propose_unlock", standby, map[string]string{"lost": sampleLost}), "pending", "")
	twice(second, "recovery.approve", proposalTargetOf(12), "k-approve", "", func() {
		f.ok(t, "lock after approval", f.run(second, "recovery.lock", standby, nil), "locked", "")
	})
	f.ok(t, "propose to withdraw", f.run(root, "recovery.propose_unlock", standby, map[string]string{"lost": sampleLost}), "pending", "")
	twice(root, "recovery.withdraw", proposalTargetOf(13), "k-withdraw", "", nil)
	if f.s.rec.entry(13).view.State != recovery.ProposalWithdrawn || len(f.s.rec.install(recoveryStandby).locks) != 4 {
		t.Errorf("replays applied twice %d", len(f.s.rec.install(recoveryStandby).locks))
	}
	f.s.fault = func(step string) bool { return step == "recovery-propose" }
	prior := f.s.Revision()
	failed := send(root, "recovery.propose_unlock", standby, "k-fault", lost)
	if f.s.Revision() == prior {
		t.Errorf("a journaled internal error keeps its operation and moves the revision like any operation %s", prior)
	}
	f.s.fault = nil
	revision := f.s.Revision()
	again := send(root, "recovery.propose_unlock", standby, "k-fault", lost)
	if failed.status != 500 || failed.body["code"] != "internal_error" || again.status != 500 || again.body["code"] != "internal_error" ||
		again.body["operation_id"] != failed.body["operation_id"] || f.s.Revision() != revision || f.s.rec.pendingUnlock(recoveryStandby) {
		t.Errorf("internal error replay %s %s", failed.raw, again.raw)
	}
}

func TestRecoveryPendingInstallationProposalResolves(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	first := recovery.Actor{ID: SampleAdmin, Kind: recovery.KindHuman}
	e, err := f.s.recoveryPropose(recovery.Change{Kind: recovery.ChangeInstallation, Installation: recovery.Installation{ID: "sample-third", Tenant: SampleWorkspace,
		Skew: recoverySkew}}, first, "test")
	if err != nil {
		t.Fatal(err)
	}
	f.ok(t, "withdraw pending installation", f.run(root, "recovery.withdraw", proposalTargetOf(e.view.Number), nil), "withdrawn",
		fmt.Sprintf("map[proposal:%d target:recovery-proposal-%d]", e.view.Number, e.view.Number))
}
