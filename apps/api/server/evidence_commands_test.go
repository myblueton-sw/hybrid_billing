package server

import (
	"fmt"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/perfgate"
	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
)

func (f world) evChecked(t *testing.T) {
	t.Helper()
	f.neverReleased(t, f.evidenceRead(f.as(AdminLogin)))
}

func (f world) traceRequirementView(id string) releasetrace.RequirementView {
	f.t.Helper()
	v, err := f.s.trace.Requirement(evKey(id), evidenceBuild)
	if err != nil {
		f.t.Fatal(err)
	}
	return v
}

func TestEvidenceApproveRequirementRetirement(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	before, err := f.s.trace.Gate(*f.s.ev.candidate)
	if err != nil {
		t.Fatal(err)
	}
	f.unchanged(t, "self approval", f.run(root, "release_evidence.approve", traceTarget(19), nil), "prerequisite_failed")
	f.ok(t, "approve", f.run(second, "release_evidence.approve", traceTarget(19), nil), "applied", "map[module:release_trace proposal:19 target:trace-proposal-19]")
	if v := f.traceRequirementView("sample-legacy-report"); !v.Retired {
		t.Errorf("not retired %+v", v)
	}
	after, err := f.s.trace.Gate(*f.s.ev.candidate)
	if err != nil || fmt.Sprint(after.Problems, after.Ready) != fmt.Sprint(before.Problems, before.Ready) || fmt.Sprint(after.RetiredBlocking) != "[{sample sample-legacy-report}]" {
		t.Errorf("gate %+v", after)
	}
	f.evChecked(t)
	read := f.evidenceRead(root)
	row := evProposal(t, read, traceTarget(19))
	if s := fmt.Sprint(row["state"], " ", row["approver"], " ", row["closed_by"], " ", row["closing_rationale"], " ", evBlocked(row)); s != "applied map[directory:true "+
		"id:sample-admin-2 login:admin2@example.invalid] map[directory:true id:sample-admin-2 login:admin2@example.invalid] sample reason false not_pending false not_pending" {
		t.Errorf("row %s", s)
	}
	if s := fmt.Sprint(evMap(read.body, "requirements", "gate")["retired_blocking"], " ", evRetire(evReq(t, read, "sample-legacy-report"))); s !=
		"[sample/sample-legacy-report] false retired <nil>" {
		t.Errorf("read %s", s)
	}
	revision := f.s.Revision()
	f.unchanged(t, "repeat approval", f.run(second, "release_evidence.approve", traceTarget(19), nil), "invalid_stage")
	if f.s.Revision() != revision {
		t.Error("repeat moved the revision")
	}
	f.unchanged(t, "withdraw applied", f.run(root, "release_evidence.withdraw", traceTarget(19), nil), "invalid_stage")
	f.unchanged(t, "retired target", f.run(root, "release_evidence.retire_requirement", traceRequirementTarget+"6", map[string]string{"reason": "Again"}), "invalid_stage")
	f.evChecked(t)
}

func TestEvidenceApproveProfileRetirement(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.unchanged(t, "self approval", f.run(second, "release_evidence.approve", perfTarget(9), nil), "prerequisite_failed")
	f.ok(t, "approve", f.run(root, "release_evidence.approve", perfTarget(9), nil), "applied", "map[module:perf_gate proposal:9 target:perf-proposal-9]")
	if _, retired, err := f.s.perf.Profile("sample-month"); err != nil || !retired {
		t.Errorf("profile %v %v", retired, err)
	}
	v, err := f.s.perf.Gate(perfgate.GateMonthlyClose, evidenceBuild)
	if err != nil || v.Status != perfgate.StatusNotRun || len(v.Retired) != 1 || v.Retired[0].Profile != "sample-month" {
		t.Errorf("monthly close %+v %v", v, err)
	}
	f.evChecked(t)
	f.unchanged(t, "repeat", f.run(root, "release_evidence.approve", perfTarget(9), nil), "invalid_stage")
	f.unchanged(t, "retired profile", f.run(second, "release_evidence.retire_profile", perfProfileTarget+"sample-month", nil), "invalid_stage")
	proposed := f.run(root, "release_evidence.retire_profile", perfProfileTarget+"sample-resume", nil)
	f.ok(t, "propose", proposed, "pending", "map[module:perf_gate proposal:10 target:perf-proposal-10]")
	if p, err := f.s.perf.Proposal(10); err != nil || p.Proposer != SampleAdmin || p.Change.Retire.Profile != "sample-resume" {
		t.Errorf("proposal %+v %v", p, err)
	}
	f.unchanged(t, "duplicate", f.run(root, "release_evidence.retire_profile", perfProfileTarget+"sample-resume", nil), "invalid_stage")
	f.unchanged(t, "duplicate by another", f.run(second, "release_evidence.retire_profile", perfProfileTarget+"sample-resume", nil), "invalid_stage")
	f.unchanged(t, "not proposer", f.run(second, "release_evidence.withdraw", perfTarget(10), nil), "prerequisite_failed")
	f.ok(t, "withdraw", f.run(root, "release_evidence.withdraw", perfTarget(10), nil), "withdrawn", "map[module:perf_gate proposal:10 target:perf-proposal-10]")
	f.unchanged(t, "approve after withdraw", f.run(second, "release_evidence.approve", perfTarget(10), nil), "invalid_stage")
	f.unchanged(t, "withdraw after withdraw", f.run(root, "release_evidence.withdraw", perfTarget(10), nil), "invalid_stage")
	f.ok(t, "propose again", f.run(second, "release_evidence.retire_profile", perfProfileTarget+"sample-resume", nil), "pending", "")
	f.ok(t, "approve again", f.run(root, "release_evidence.approve", perfTarget(11), nil), "applied", "")
	f.evChecked(t)
}

func TestEvidenceRetireRequirement(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	target := func(n int) string { return fmt.Sprint(traceRequirementTarget, n) }
	retire := func(who actor, n int, fields map[string]string) reply {
		return f.run(who, "release_evidence.retire_requirement", target(n), fields)
	}
	f.fieldRefused(t, "control reason", retire(root, 2, map[string]string{"reason": "a\u0085b"}), "reason")
	f.fieldRefused(t, "long reason", retire(root, 2, map[string]string{"reason": strings.Repeat("r", 201)}), "reason")
	f.fieldRefused(t, "unknown replacement", retire(root, 2, map[string]string{"reason": "x", "replacement": target(9)}), "replacement")
	f.fieldRefused(t, "self replacement", retire(root, 2, map[string]string{"reason": "x", "replacement": target(2)}), "replacement")
	f.fieldRefused(t, "other replacement form", retire(root, 2, map[string]string{"reason": "x", "replacement": "sample-recovery"}), "replacement")
	f.unchanged(t, "common without replacement", retire(root, 1, map[string]string{"reason": "x"}), "invalid_stage")
	f.unchanged(t, "weaker replacement", retire(root, 3, map[string]string{"reason": "x", "replacement": target(5)}), "invalid_stage")
	f.unchanged(t, "older replacement", retire(root, 5, map[string]string{"reason": "x", "replacement": target(2)}), "invalid_stage")
	f.unchanged(t, "identical by proposer", retire(root, 6, map[string]string{"reason": evidenceRetireReason}), "invalid_stage")
	f.unchanged(t, "identical by another", retire(second, 6, map[string]string{"reason": evidenceRetireReason}), "invalid_stage")
	f.unchanged(t, "different by another", retire(second, 6, map[string]string{"reason": "Other"}), "invalid_stage")
	if len(f.s.ev.entries) != 28 {
		t.Errorf("refused proposals stored %d", len(f.s.ev.entries))
	}
	proposed := retire(second, 2, map[string]string{"reason": "Merged into recovery", "replacement": target(3)})
	f.ok(t, "propose", proposed, "pending", "map[module:release_trace proposal:20 target:trace-proposal-20]")
	if p, err := f.s.trace.Proposal(20); err != nil || p.Proposer != SampleAdmin2 || p.Change.Retire != (releasetrace.Retire{Key: evKey("sample-approval"),
		Replacement: evKey("sample-recovery"), Reason: "Merged into recovery"}) {
		t.Errorf("proposal %+v %v", p, err)
	}
	f.unchanged(t, "pending", retire(root, 2, map[string]string{"reason": "x"}), "invalid_stage")
	f.ok(t, "approve", f.run(root, "release_evidence.approve", traceTarget(20), nil), "applied", "")
	if v := f.traceRequirementView("sample-approval"); !v.Retired || v.Replacement != evKey("sample-recovery") {
		t.Errorf("retired %+v", v)
	}
	if r := evReq(t, f.evidenceRead(root), "sample-approval"); fmt.Sprint(r["retired"], " ", r["replacement"]) != "true sample/sample-recovery" {
		t.Errorf("read replacement %v %v", r["retired"], r["replacement"])
	}
	f.evChecked(t)
	f.ok(t, "propose with a used replacement", retire(root, 4, map[string]string{"reason": "x", "replacement": target(5)}), "pending", "")
	f.ok(t, "propose another", retire(root, 5, map[string]string{"reason": "x", "replacement": target(6)}), "pending", "")
	f.ok(t, "approve the second", f.run(second, "release_evidence.approve", traceTarget(22), nil), "applied", "")
	f.unchanged(t, "replacement weakened meanwhile", f.run(second, "release_evidence.approve", traceTarget(21), nil), "invalid_stage")
	if e := f.s.ev.entry(moduleTrace, 21); e.state != entryPending {
		t.Errorf("refused approval changed the entry %s", e.state)
	}
	f.ok(t, "withdraw", f.run(root, "release_evidence.withdraw", traceTarget(21), nil), "withdrawn", "")
	row := evProposal(t, f.evidenceRead(root), traceTarget(21))
	if s := fmt.Sprint(row["state"], " ", row["closed_by"], " ", evBlocked(row)); s != "withdrawn map[directory:true id:sample-admin login:admin@example.invalid] "+
		"false not_pending false not_pending" {
		t.Errorf("withdrawn %s", s)
	}
	f.evChecked(t)
}

func TestEvidenceNonConsoleKindsStayClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	e, err := f.s.tracePropose(releasetrace.Change{Kind: releasetrace.ChangeImplemented, Implemented: releasetrace.Implemented{Key: evKey("sample-erp-template"),
		Commit: sampleHash("x")[:40]}}, evA1, "injected")
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.s.perfPropose(perfgate.Change{Kind: perfgate.ChangeHardware, Hardware: perfgate.Hardware{ID: "test-hw", CPUs: 1, MemoryMiB: 1, DiskGiB: 1,
		Storage: perfgate.StorageHDD, Spec: sampleHash("hw")}}, pfA1, "injected")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	f.s.fault = func(step string) bool {
		called = called || step == "trace-approve" || step == "perf-approve"
		return false
	}
	f.unchanged(t, "approve implemented", f.run(second, "release_evidence.approve", traceTarget(e.number), nil), "invalid_stage")
	f.unchanged(t, "approve hardware", f.run(second, "release_evidence.approve", perfTarget(p.number), nil), "invalid_stage")
	f.s.fault = nil
	if called {
		t.Error("a non-retire approval called a module")
	}
	if v := f.traceRequirementView("sample-erp-template"); v.Commit != "" {
		t.Errorf("implemented %+v", v)
	}
	f.ok(t, "withdraw implemented", f.run(root, "release_evidence.withdraw", traceTarget(e.number), nil), "withdrawn", "")
	f.ok(t, "withdraw hardware", f.run(root, "release_evidence.withdraw", perfTarget(p.number), nil), "withdrawn", "")
	if _, err := f.s.trace.Proposal(e.number); err == nil {
		t.Error("withdrawn proposal kept")
	}
	f.evChecked(t)
}

func TestEvidenceStoreRetiredEntries(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if _, err := f.s.trace.Withdraw(19, evA1); err != nil {
		t.Fatal(err)
	}
	f.unchanged(t, "approve withdrawn in the module", f.run(second, "release_evidence.approve", traceTarget(19), nil), "invalid_stage")
	if f.s.ev.entry(moduleTrace, 19).state != storeRetired {
		t.Errorf("state %s", f.s.ev.entry(moduleTrace, 19).state)
	}
	f.unchanged(t, "withdraw store retired", f.run(root, "release_evidence.withdraw", traceTarget(19), nil), "invalid_stage")
	if s := evBlocked(evProposal(t, f.evidenceRead(root), traceTarget(19))); s != "false store_retired false store_retired" {
		t.Errorf("read %s", s)
	}
	if s := evRetire(evReq(t, f.evidenceRead(root), "sample-legacy-report")); s != "true  <nil>" {
		t.Errorf("store retired blocks retirement %s", s)
	}
	if _, err := f.s.perf.Approve(9, pfA1); err != nil {
		t.Fatal(err)
	}
	f.unchanged(t, "approve applied in the module", f.run(second, "release_evidence.approve", perfTarget(9), nil), "invalid_stage")
	if f.s.ev.entry(modulePerf, 9).state != storeRetired {
		t.Errorf("perf state %s", f.s.ev.entry(modulePerf, 9).state)
	}
	e, err := f.s.tracePropose(releasetrace.Change{Kind: releasetrace.ChangeRetire, Retire: releasetrace.Retire{Key: evKey("sample-api-export"), Reason: "x"}}, evA1, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.trace.Withdraw(e.number, evA1); err != nil {
		t.Fatal(err)
	}
	f.unchanged(t, "withdraw withdrawn in the module", f.run(root, "release_evidence.withdraw", traceTarget(e.number), nil), "invalid_stage")
	p, err := f.s.perfPropose(perfgate.Change{Kind: perfgate.ChangeRetire, Retire: perfgate.Retire{Profile: "sample-e2e"}}, pfA1, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.perf.Withdraw(p.number, pfA1); err != nil {
		t.Fatal(err)
	}
	f.unchanged(t, "perf withdraw withdrawn in the module", f.run(root, "release_evidence.withdraw", perfTarget(p.number), nil), "invalid_stage")
	if e.state != storeRetired || p.state != storeRetired {
		t.Errorf("withdraw retire %s %s", e.state, p.state)
	}
	applied := f.s.ev.entry(moduleTrace, 1)
	if err := retireEntry(applied, true, releasetrace.ErrNotFound); err == nil || applied.state != releasetrace.ProposalApplied {
		t.Error("an applied entry was retired")
	}
	f.evChecked(t)
}

func TestEvidenceLimitMapsToPrerequisite(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	for n := 0; ; n++ {
		if _, err := f.s.trace.Propose(releasetrace.Change{Kind: releasetrace.ChangeBuild, Build: releasetrace.Build{Build: fmt.Sprint("9.0.", n), Commit: sampleHash("b")[:40]}},
			evA1); err != nil {
			break
		}
	}
	for n := 0; ; n++ {
		if _, err := f.s.perf.Propose(perfgate.Change{Kind: perfgate.ChangeHardware, Hardware: perfgate.Hardware{ID: fmt.Sprint("hw-", n), CPUs: 1, MemoryMiB: 1, DiskGiB: 1,
			Storage: perfgate.StorageHDD, Spec: sampleHash("hw")}}, pfA1); err != nil {
			break
		}
	}
	if s := evRetire(evReq(t, f.evidenceRead(root), "sample-api-export")); s != "true  <nil>" {
		t.Errorf("read before the limit %s", s)
	}
	f.unchanged(t, "trace limit", f.run(root, "release_evidence.retire_requirement", traceRequirementTarget+"5", map[string]string{"reason": "x"}), "prerequisite_failed")
	f.unchanged(t, "perf limit", f.run(root, "release_evidence.retire_profile", perfProfileTarget+"sample-e2e", nil), "prerequisite_failed")
	f.ok(t, "the other administrator", f.run(second, "release_evidence.retire_requirement", traceRequirementTarget+"5", map[string]string{"reason": "x"}), "pending", "")
	f.ok(t, "the other administrator's profile", f.run(second, "release_evidence.retire_profile", perfProfileTarget+"sample-e2e", nil), "pending", "")
}

func TestEvidenceStaleRevisionAndReplay(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	old := f.revision(root)
	if got := f.runWith(root, old, "release_evidence.retire_profile", perfProfileTarget+"sample-e2e", map[string]string{}); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.code(t, "stale", f.runWith(second, old, "release_evidence.approve", traceTarget(19), map[string]string{}), 409, "revision_conflict")
	bodies := map[string]string{}
	send := func(who actor, command, target, key, fields string) reply {
		if bodies[key] == "" {
			bodies[key] = fmt.Sprintf(`{"request_id":"req-%s","target_id":"%s","idempotency_key":"%s","expected_revision":"%s","rationale":"r","body":{%s}}`, key, target, key,
				f.revision(who), fields)
		}
		return f.call(command, bodies[key], who.cookie, who.csrf)
	}
	twice := func(who actor, command, target, key, fields string, after func()) reply {
		t.Helper()
		first := send(who, command, target, key, fields)
		if after != nil {
			after()
		}
		revision := f.s.Revision()
		again := send(who, command, target, key, fields)
		if first.status != 200 || again.status != 200 || again.body["replay"] != true || fmt.Sprint(first.body["fields"]) != fmt.Sprint(again.body["fields"]) ||
			f.s.Revision() != revision {
			t.Fatalf("%s replay %s %s", command, first.raw, again.raw)
		}
		return first
	}
	twice(second, "release_evidence.approve", traceTarget(19), "k-approve", "", nil)
	twice(root, "release_evidence.retire_requirement", traceRequirementTarget+"5", "k-retire", `"reason":"Replay"`, func() {
		f.ok(t, "approve", f.run(second, "release_evidence.approve", traceTarget(20), nil), "applied", "")
	})
	twice(second, "release_evidence.retire_profile", perfProfileTarget+"sample-daily", "k-profile", "", func() {
		f.ok(t, "withdraw", f.run(second, "release_evidence.withdraw", perfTarget(11), nil), "withdrawn", "")
	})
	twice(root, "release_evidence.withdraw", perfTarget(10), "k-withdraw", "", nil)
	if f.s.ev.entry(moduleTrace, 20).state != releasetrace.ProposalApplied || f.s.ev.entry(modulePerf, 10).state != perfgate.ProposalWithdrawn ||
		f.s.ev.entry(modulePerf, 11).state != perfgate.ProposalWithdrawn {
		t.Error("replays applied twice")
	}
	before := f.evModule()
	f.s.fault = func(step string) bool { return step == "trace-propose" }
	prior := f.s.Revision()
	failed := send(root, "release_evidence.retire_requirement", traceRequirementTarget+"3", "k-fault", `"reason":"x"`)
	if f.s.Revision() == prior {
		t.Errorf("a journaled internal error moves the revision %s", prior)
	}
	f.s.fault = nil
	revision := f.s.Revision()
	again := send(root, "release_evidence.retire_requirement", traceRequirementTarget+"3", "k-fault", `"reason":"x"`)
	if failed.status != 500 || failed.body["code"] != "internal_error" || again.status != 500 || again.body["operation_id"] != failed.body["operation_id"] ||
		f.s.Revision() != revision || f.s.ev.pendingRetire(evKey("sample-recovery")) != nil || f.evModule() != before {
		t.Errorf("internal error replay %s %s", failed.raw, again.raw)
	}
	f.evChecked(t)
}

func TestEvidenceCommandsActAsTheHumanReader(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.ok(t, "approve", f.run(second, "release_evidence.approve", traceTarget(19), nil), "applied", "")
	if v, err := f.s.trace.Proposal(19); err != nil || v.Approver != SampleAdmin2 || v.Proposer != SampleAdmin {
		t.Errorf("approver %+v %v", v, err)
	}
	f.ok(t, "approve profile", f.run(root, "release_evidence.approve", perfTarget(9), nil), "applied", "")
	if v, err := f.s.perf.Proposal(9); err != nil || v.Approver != SampleAdmin {
		t.Errorf("perf approver %+v %v", v, err)
	}
	f.ok(t, "propose", f.run(second, "release_evidence.retire_requirement", traceRequirementTarget+"5", map[string]string{"reason": "x"}), "pending", "")
	if v, err := f.s.trace.Proposal(20); err != nil || v.Proposer != SampleAdmin2 {
		t.Errorf("proposer %+v %v", v, err)
	}
	f.ok(t, "propose profile", f.run(root, "release_evidence.retire_profile", perfProfileTarget+"sample-e2e", nil), "pending", "")
	if v, err := f.s.perf.Proposal(10); err != nil || v.Proposer != SampleAdmin {
		t.Errorf("perf proposer %+v %v", v, err)
	}
	f.evChecked(t)
}
