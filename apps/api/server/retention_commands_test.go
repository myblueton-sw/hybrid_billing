package server

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/identity"
	"github.com/myblueton-sw/hybrid_billing/modules/retention"
)

func (f world) fieldRefused(t *testing.T, name string, got reply, field string) {
	t.Helper()
	f.unchanged(t, name, got, "invalid_request")
	if fmt.Sprint(got.body["fields"]) != "["+field+"]" {
		t.Errorf("%s fields %s", name, got.raw)
	}
}

func impactFields(now map[string]any) map[string]string {
	return map[string]string{"impact_as_of": now["as_of"].(string), "impact_count": fmt.Sprint(now["count"]), "impact_digest": now["digest"].(string)}
}

func (f world) currentPolicy(class string) string {
	f.t.Helper()
	v, err := f.s.retention.Policy(class)
	if err != nil {
		return err.Error()
	}
	return fmt.Sprint(v.Number, " ", v.Policy.Period, " ", v.Policy.Grace, " ", v.Proposer, " ", v.Approver)
}

func TestRetentionApprovePolicy(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	target := retTarget(7)
	now := impactFields(retProposal(t, f.retentionRead(root), 7)["impact_now"].(map[string]any))
	f.unchanged(t, "no impact", f.run(root, "retention.approve", target, nil), "invalid_stage")
	wrong := impactFields(retProposal(t, f.retentionRead(root), 7)["impact_now"].(map[string]any))
	wrong["impact_count"] = "3"
	f.unchanged(t, "wrong count", f.run(root, "retention.approve", target, wrong), "invalid_stage")
	wrong["impact_count"], wrong["impact_digest"] = now["impact_count"], sampleHash("other")
	f.unchanged(t, "wrong digest", f.run(root, "retention.approve", target, wrong), "invalid_stage")
	for _, missing := range retentionImpactFields {
		partial := impactFields(retProposal(t, f.retentionRead(root), 7)["impact_now"].(map[string]any))
		delete(partial, missing)
		f.fieldRefused(t, "without "+missing, f.run(root, "retention.approve", target, partial), missing)
	}
	for field, value := range map[string]string{"impact_count": "-1", "impact_digest": strings.ToUpper(now["impact_digest"]), "impact_as_of": "2026-11-16"} {
		bad := impactFields(retProposal(t, f.retentionRead(root), 7)["impact_now"].(map[string]any))
		bad[field] = value
		f.fieldRefused(t, "malformed "+field, f.run(root, "retention.approve", target, bad), field)
	}
	big := impactFields(retProposal(t, f.retentionRead(root), 7)["impact_now"].(map[string]any))
	big["impact_count"] = "99999999999999999999"
	f.fieldRefused(t, "huge count", f.run(root, "retention.approve", target, big), "impact_count")
	f.advance(61 * time.Minute)
	root = f.as(AdminLogin)
	f.unchanged(t, "stale impact", f.run(root, "retention.approve", target, now), "invalid_stage")
	fresh := retProposal(t, f.retentionRead(root), 7)["impact_now"].(map[string]any)
	if fresh["as_of"] != "2026-11-16T10:01:00Z" {
		t.Errorf("fresh %v", fresh)
	}
	f.unchanged(t, "self approval", f.run(f.as(Admin2Login), "retention.approve", target, impactFields(fresh)), "prerequisite_failed")
	f.ok(t, "approve", f.run(root, "retention.approve", target, impactFields(fresh)), "applied", "map[proposal:7 target:retention-proposal-7]")
	if s := f.currentPolicy(retention.ClassSource); s != "2 180 30 sample-admin-2 sample-admin" {
		t.Errorf("policy %s", s)
	}
	read := f.retentionRead(root)
	row := retProposal(t, read, 7)
	if s := fmt.Sprint(row["state"], " ", row["closed_by"], " ", row["closing_rationale"], " ", retBlocked(row), " ", row["impact_required"], " ", row["impact_now"]); s !=
		"applied map[id:sample-admin login:admin@example.invalid] sample reason false not_pending false not_pending false <nil>" {
		t.Errorf("applied %s", s)
	}
	if s := fmt.Sprint(retArtifact(t, read, "sample-source-5")["standing"], " ", retArtifact(t, read, "sample-source-6")["standing"]); s != "eligible eligible" {
		t.Errorf("after shortening %s", s)
	}
	f.unchanged(t, "repeat approval", f.run(root, "retention.approve", target, impactFields(fresh)), "invalid_stage")
	f.unchanged(t, "withdraw applied", f.run(f.as(Admin2Login), "retention.withdraw", target, nil), "invalid_stage")
	f.policyCycle(t)
}

func (f world) policyCycle(t *testing.T) {
	t.Helper()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	claim := retentionClassTarget + retention.ClassClaim
	for _, c := range []struct{ period, grace, field string }{{"0", "0", "period"}, {"36501", "0", "period"}, {"1", "-1", "grace"}, {"1", "366", "grace"},
		{"99999999999999999999", "0", "period"}} {
		f.fieldRefused(t, "range "+c.period+" "+c.grace, f.run(root, "retention.propose_policy", claim, map[string]string{"period": c.period, "grace": c.grace}), c.field)
	}
	f.fieldRefused(t, "not an integer", f.run(root, "retention.propose_policy", claim, map[string]string{"period": "1.5", "grace": "0"}), "period")
	f.unchanged(t, "unchanged", f.run(root, "retention.propose_policy", retentionClassTarget+retention.ClassFinal, map[string]string{"period": "3650", "grace": "0"}),
		"invalid_stage")
	proposed := f.run(second, "retention.propose_policy", claim, map[string]string{"period": "36500", "grace": "365"})
	f.ok(t, "propose", proposed, "pending", "")
	number := proposed.body["fields"].(map[string]any)["proposal"].(string)
	if v, err := f.s.retention.Proposal(uint64(fmtInt(number))); err != nil || v.Proposer != SampleAdmin2 || v.Change.Policy != (retention.Policy{Class: retention.ClassClaim,
		Period: 36500, Grace: 365}) {
		t.Errorf("proposal %+v %v", v, err)
	}
	f.unchanged(t, "same content", f.run(second, "retention.propose_policy", claim, map[string]string{"period": "36500", "grace": "365"}), "invalid_stage")
	f.unchanged(t, "other administrator", f.run(root, "retention.propose_policy", claim, map[string]string{"period": "10", "grace": "0"}), "invalid_stage")
	f.unchanged(t, "not proposer", f.run(root, "retention.withdraw", retentionProposalTarget+number, nil), "prerequisite_failed")
	f.ok(t, "approve lengthening without impact", f.run(root, "retention.approve", retentionProposalTarget+number, nil), "applied", "")
	if s := f.currentPolicy(retention.ClassClaim); s != "1 36500 365 sample-admin-2 sample-admin" {
		t.Errorf("claim policy %s", s)
	}
}

func TestRetentionRelease(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.fieldRefused(t, "impact on a release", f.run(second, "retention.approve", retTarget(6), map[string]string{"impact_count": "1", "impact_digest": sampleHash("x")}),
		"impact_count")
	f.unchanged(t, "self approval", f.run(root, "retention.approve", retTarget(6), nil), "prerequisite_failed")
	f.ok(t, "approve release", f.run(second, "retention.approve", retTarget(6), nil), "applied", "map[proposal:6 target:retention-proposal-6]")
	if v := f.retHoldView(retentionTemporary); !v.Released || !f.s.ret.hold(retentionTemporary).released || f.s.ret.hold(retentionDispute).released {
		t.Errorf("hold %+v, the server's note of it %v", v, f.s.ret.hold(retentionTemporary).released)
	}
	if section := retSection(f.retentionRead(root), "holds"); fmtInt(section["released"].(map[string]any)["total"]) != 1 || retRows(section)[0]["released"] == nil {
		t.Errorf("the read counts one released hold: %v", section["released"])
	}
	if f.retStanding("sample-temporary-2").Held {
		t.Error("still held")
	}
	f.unchanged(t, "repeat by approver", f.run(second, "retention.approve", retTarget(6), nil), "invalid_stage")
	tmp := retentionHoldTarget + retentionTemporary
	f.unchanged(t, "release released", f.run(root, "retention.propose_release", tmp, map[string]string{"reason": "again"}), "invalid_stage")
	dispute := retentionHoldTarget + retentionDispute
	f.fieldRefused(t, "bad reason", f.run(second, "retention.propose_release", dispute, map[string]string{"reason": "a\u0085b"}), "reason")
	f.fieldRefused(t, "long reason", f.run(second, "retention.propose_release", dispute, map[string]string{"reason": strings.Repeat("r", 201)}), "reason")
	f.unchanged(t, "placer proposes", f.run(root, "retention.propose_release", dispute, map[string]string{"reason": "done"}), "invalid_stage")
	proposed := f.run(second, "retention.propose_release", dispute, map[string]string{"reason": "Dispute settled"})
	f.ok(t, "propose", proposed, "pending", "map[proposal:8 target:retention-proposal-8]")
	if v, err := f.s.retention.Proposal(8); err != nil || v.Change.Release != (retention.Release{Hold: retentionDispute, Reason: "Dispute settled"}) {
		t.Errorf("proposal %+v %v", v, err)
	}
	f.unchanged(t, "duplicate", f.run(second, "retention.propose_release", dispute, map[string]string{"reason": "Dispute settled"}), "invalid_stage")
	f.unchanged(t, "duplicate by another", f.run(root, "retention.propose_release", dispute, map[string]string{"reason": "Other"}), "invalid_stage")
	f.unchanged(t, "placer approves", f.run(root, "retention.approve", retTarget(8), nil), "invalid_stage")
	if v := f.retHoldView(retentionDispute); v.Released {
		t.Error("released by the placer")
	}
	f.ok(t, "withdraw", f.run(second, "retention.withdraw", retTarget(8), nil), "withdrawn", "map[proposal:8 target:retention-proposal-8]")
	f.unchanged(t, "approve after withdraw", f.run(root, "retention.approve", retTarget(8), nil), "invalid_stage")
	f.unchanged(t, "withdraw after withdraw", f.run(second, "retention.withdraw", retTarget(8), nil), "invalid_stage")
	row := retProposal(t, f.retentionRead(root), 8)
	if s := fmt.Sprint(row["state"], " ", row["closed_by"], " ", retBlocked(row)); s != "withdrawn map[id:sample-admin-2 login:admin2@example.invalid] false not_pending false not_pending" {
		t.Errorf("withdrawn %s", s)
	}
}

func TestRetentionPlanProposalsStayClosed(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	e, err := f.s.retentionPropose(retention.Change{Kind: retention.ChangePlan, Plan: retention.DeletionPlan{Class: retention.ClassSource,
		Artifacts: []string{"sample-source-5"}, Chunk: 1, Lease: time.Hour, Reason: "test"}}, retA1, SampleWorkspace, "test")
	if err != nil {
		t.Fatal(err)
	}
	called := false
	f.s.fault = func(step string) bool {
		called = called || step == "retention-approve"
		return false
	}
	f.unchanged(t, "approve plan", f.run(second, "retention.approve", retTarget(e.view.Number), nil), "invalid_stage")
	f.s.fault = nil
	if called {
		t.Error("plan approval called the module")
	}
	if v, err := f.s.retention.Proposal(e.view.Number); err != nil || v.State != retention.ProposalPending || f.retStanding("sample-source-5").Run != "" {
		t.Errorf("plan %+v %v", v, err)
	}
	f.ok(t, "withdraw plan", f.run(root, "retention.withdraw", retTarget(e.view.Number), nil), "withdrawn", "")
	if _, err := f.s.retention.Proposal(e.view.Number); err == nil {
		t.Error("plan kept")
	}
}

func TestRetentionPlaceHold(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root := f.as(AdminLogin)
	place := func(target string, fields map[string]string) reply {
		return f.run(root, "retention.place_hold", target, fields)
	}
	art := func(id string) string { return retentionArtifactTarget + id }
	tenant := retentionTenantTarget + SampleWorkspace
	f.ok(t, "artifact", place(art("sample-source-5"), map[string]string{"owner": SampleMember, "reason": "Audit"}), "held",
		"map[hold:retention-hold-1 not_preserved:0 outcome_pending:0 partially_deleted:0 target:retention-hold:retention-hold-1]")
	if v := f.retHoldView("retention-hold-1"); fmt.Sprint(v.Hold, " ", v.Placer) != "{retention-hold-1 sample-member Audit [sample-source-5] [] []} sample-admin" ||
		!f.retStanding("sample-source-5").Held {
		t.Errorf("hold %+v", v)
	}
	f.ok(t, "deleted artifact", place(art("sample-source-1"), map[string]string{"owner": SampleAdmin2, "reason": "Late"}), "partially_preserved",
		"map[hold:retention-hold-2 not_preserved:1 outcome_pending:0 partially_deleted:0 target:retention-hold:retention-hold-2]")
	f.ok(t, "tenant class", place(tenant, map[string]string{"owner": SampleAdmin, "reason": "Temp", "class": retention.ClassTemporary}), "partially_preserved",
		"map[hold:retention-hold-3 not_preserved:1 outcome_pending:0 partially_deleted:0 target:retention-hold:retention-hold-3]")
	f.ok(t, "tenant", place(tenant, map[string]string{"owner": SampleAdmin, "reason": "All"}), "partially_preserved",
		"map[hold:retention-hold-4 not_preserved:2 outcome_pending:0 partially_deleted:1 target:retention-hold:retention-hold-4]")
	if v := f.retHoldView("retention-hold-4"); fmt.Sprint(v.Hold.Tenants, v.Hold.Classes, v.Hold.Artifacts) != "[sample-workspace] [] []" {
		t.Errorf("tenant hold %+v", v.Hold)
	}
	if h := retHold(t, f.retentionRead(root), "retention-hold-3"); fmt.Sprint(h["scope"], " ", h["rationale"], " ", h["placer"]) !=
		"map[artifacts:[] classes:[temporary] workspace:true] sample reason map[id:sample-admin login:admin@example.invalid]" {
		t.Errorf("read %v", h)
	}
	before := len(f.s.ret.holds)
	f.fieldRefused(t, "unknown owner", place(art("sample-source-6"), map[string]string{"owner": "nobody", "reason": "x"}), "owner")
	f.fieldRefused(t, "class on an artifact", place(art("sample-source-6"), map[string]string{"owner": SampleAdmin, "reason": "x", "class": retention.ClassSource}), "class")
	f.fieldRefused(t, "control reason", place(art("sample-source-6"), map[string]string{"owner": SampleAdmin, "reason": "a\u0001"}), "reason")
	f.fieldRefused(t, "long reason", place(art("sample-source-6"), map[string]string{"owner": SampleAdmin, "reason": strings.Repeat("é", 101)}), "reason")
	f.fieldRefused(t, "unknown class", place(tenant, map[string]string{"owner": SampleAdmin, "reason": "x", "class": "other"}), "class")
	revision := f.revision(root)
	f.s.fault = func(step string) bool { return step == "roster" }
	f.code(t, "roster", f.runWith(root, revision, "retention.place_hold", art("sample-source-6"), map[string]string{"owner": SampleAdmin, "reason": "x"}), 503,
		"dependency_unavailable")
	f.s.fault = nil
	if len(f.s.ret.holds) != before {
		t.Error("refused holds stored")
	}
	f.ok(t, "after failures", place(art("sample-source-6"), map[string]string{"owner": SampleAdmin, "reason": "x"}), "held", "")
	if f.s.ret.holds[len(f.s.ret.holds)-1].id != "retention-hold-10" {
		t.Errorf("counter %s", f.s.ret.holds[len(f.s.ret.holds)-1].id)
	}
	if _, err := f.s.retention.PlaceHold(retention.Hold{ID: "retention-hold-11", Owner: SampleAdmin, Reason: "x", Artifacts: []string{"sample-source-7"}}, retA1); err != nil {
		t.Fatal(err)
	}
	f.ok(t, "skip known", place(art("sample-source-7"), map[string]string{"owner": SampleAdmin, "reason": "x"}), "held", "")
	if f.s.ret.holds[len(f.s.ret.holds)-1].id != "retention-hold-12" {
		t.Errorf("skipped %s", f.s.ret.holds[len(f.s.ret.holds)-1].id)
	}
	if got := f.run(root, "identity.suspend_user", SampleMember, nil); got.status != 200 {
		t.Fatalf("suspend %s", got.raw)
	}
	f.fieldRefused(t, "suspended owner", place(art("sample-source-6"), map[string]string{"owner": SampleMember, "reason": "x"}), "owner")
	if got := f.run(root, "identity.reinstate_user", SampleMember, nil); got.status != 200 {
		t.Fatalf("reinstate %s", got.raw)
	}
	if got := f.run(root, "identity.remove_membership", SampleMember, nil); got.status != 200 {
		t.Fatalf("remove %s", got.raw)
	}
	f.fieldRefused(t, "removed owner", place(art("sample-source-6"), map[string]string{"owner": SampleMember, "reason": "x"}), "owner")
}

func TestRetentionStopRun(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	f.unchanged(t, "finished", f.run(root, "retention.stop_run", "retention-run-1", nil), "invalid_stage")
	f.ok(t, "stop", f.run(second, "retention.stop_run", "retention-run-2", nil), "done", "map[run:run/2 stopped:2 target:retention-run-2]")
	v := f.retRunView("run/2")
	if v.State != retention.RunDone || v.Stopper != SampleAdmin2 || v.Counts[retention.OutcomeStopped] != 2 {
		t.Errorf("run %+v", v)
	}
	read := f.retentionRead(root)
	f.retBalanced(t, read)
	run := retRun(t, read, "run/2")
	if s := fmt.Sprint(run["stores"], " ", run["stopper"], " ", run["can_stop"], " ", run["stop_blocked"]); s != "[map[deleted:1 excluded:2 failed:0 pending:0 store:sample-object-store "+
		"target:3]] map[id:sample-admin-2 login:admin2@example.invalid] false finished" {
		t.Errorf("stopped %s", s)
	}
	if a := retArtifact(t, read, "sample-temporary-2"); fmt.Sprint(a["access"], " ", a["standing"], " ", a["run"]) != "ok held <nil>" {
		t.Errorf("released artifact %v", a)
	}
	f.unchanged(t, "again", f.run(root, "retention.stop_run", "retention-run-2", nil), "invalid_stage")
}

func TestRetentionRetiredEntries(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if _, err := f.s.retention.Withdraw(7, retA2); err != nil {
		t.Fatal(err)
	}
	now := map[string]string{"impact_as_of": "2026-11-16T09:00:00Z", "impact_count": "2", "impact_digest": sampleHash("x")}
	f.unchanged(t, "approve withdrawn in the module", f.run(root, "retention.approve", retTarget(7), now), "invalid_stage")
	if f.s.ret.entry(7).view.State != retentionRetired {
		t.Errorf("state %s", f.s.ret.entry(7).view.State)
	}
	f.unchanged(t, "withdraw retired", f.run(second, "retention.withdraw", retTarget(7), nil), "invalid_stage")
	if s := retBlocked(retProposal(t, f.retentionRead(root), 7)); s != "false retired false retired" {
		t.Errorf("read %s", s)
	}
	if p := f.retentionRead(root).body["policies"].([]any)[5].(map[string]any); p["can_propose"] != true {
		t.Errorf("retired blocks proposals %v", p)
	}
	if _, err := f.s.retention.Approve(6, retA2, retention.Impact{}); err != nil {
		t.Fatal(err)
	}
	f.unchanged(t, "approve applied in the module", f.run(root, "retention.approve", retTarget(6), nil), "invalid_stage")
	if f.s.ret.entry(6).view.State != retentionRetired {
		t.Errorf("state %s", f.s.ret.entry(6).view.State)
	}
	e, err := f.s.retentionPropose(retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: retention.ClassIdentity, Period: 5}}, retA1, SampleWorkspace, "x")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.retention.Withdraw(e.view.Number, retA1); err != nil {
		t.Fatal(err)
	}
	f.unchanged(t, "withdraw withdrawn in the module", f.run(root, "retention.withdraw", retTarget(e.view.Number), nil), "invalid_stage")
	if e.view.State != retentionRetired {
		t.Errorf("withdraw retire %s", e.view.State)
	}
	applied := f.s.ret.entry(1)
	if err := f.s.retireRetention(applied, retention.ErrNotFound); err == nil || applied.view.State != retention.ProposalApplied {
		t.Error("an applied entry was retired")
	}
}

func TestRetentionLimitMapsToPrerequisite(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	for period := 1; ; period++ {
		if _, err := f.s.retention.Propose(retention.Change{Kind: retention.ChangePolicy, Policy: retention.Policy{Class: retention.ClassProjection, Period: period}},
			retA1); err != nil {
			break
		}
	}
	if p := f.retentionRead(root).body["policies"].([]any)[3].(map[string]any); p["class"] != retention.ClassIdentity || p["can_propose"] != true {
		t.Errorf("read before the limit %v", p)
	}
	f.unchanged(t, "limit", f.run(root, "retention.propose_policy", retentionClassTarget+retention.ClassIdentity, map[string]string{"period": "5", "grace": "0"}),
		"prerequisite_failed")
	f.ok(t, "the other administrator", f.run(second, "retention.propose_policy", retentionClassTarget+retention.ClassIdentity, map[string]string{"period": "5", "grace": "0"}),
		"pending", "")
}

func TestRetentionStaleRevisionAndReplay(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	old := f.revision(root)
	if got := f.runWith(root, old, "retention.stop_run", "retention-run-2", map[string]string{}); got.status != 200 {
		t.Fatal(got.raw)
	}
	f.code(t, "stale", f.runWith(second, old, "retention.withdraw", retTarget(7), map[string]string{}), 409, "revision_conflict")
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
	hold := `"owner":"` + SampleMember + `","reason":"Replay"`
	first := twice(root, "retention.place_hold", retentionArtifactTarget+"sample-source-5", "k-hold", hold, nil)
	if fmt.Sprint(first.body["fields"].(map[string]any)["hold"]) != "retention-hold-1" || len(f.s.ret.holds) != 3 {
		t.Errorf("replayed hold %s %d", first.raw, len(f.s.ret.holds))
	}
	f.ok(t, "new key", f.run(root, "retention.place_hold", retentionArtifactTarget+"sample-source-5", map[string]string{"owner": SampleMember, "reason": "Replay"}), "held",
		"")
	if len(f.s.ret.holds) != 4 || f.s.ret.holds[3].id != "retention-hold-2" {
		t.Errorf("second hold %d", len(f.s.ret.holds))
	}
	twice(second, "retention.propose_release", retentionHoldTarget+retentionDispute, "k-release", `"reason":"Replay"`, func() {
		f.ok(t, "withdraw", f.run(second, "retention.withdraw", retTarget(8), nil), "withdrawn", "")
	})
	twice(root, "retention.propose_policy", retentionClassTarget+retention.ClassClaim, "k-policy", `"period":"9","grace":"0"`, func() {
		f.ok(t, "approve", f.run(second, "retention.approve", retTarget(9), nil), "applied", "")
	})
	twice(second, "retention.approve", retTarget(6), "k-approve", "", nil)
	twice(root, "retention.withdraw", retTarget(func() uint64 {
		f.ok(t, "propose", f.run(root, "retention.propose_policy", retentionClassTarget+retention.ClassIdentity, map[string]string{"period": "9", "grace": "0"}), "pending", "")
		return 10
	}()), "k-withdraw", "", nil)
	if f.s.ret.entry(10).view.State != retention.ProposalWithdrawn || f.s.ret.entry(9).view.State != retention.ProposalApplied {
		t.Error("replays applied twice")
	}
	f.s.fault = func(step string) bool { return step == "retention-propose" }
	prior := f.s.Revision()
	failed := send(root, "retention.propose_policy", retentionClassTarget+retention.ClassConversation, "k-fault", `"period":"9","grace":"0"`)
	if f.s.Revision() == prior {
		t.Errorf("a journaled internal error moves the revision %s", prior)
	}
	f.s.fault = nil
	revision := f.s.Revision()
	again := send(root, "retention.propose_policy", retentionClassTarget+retention.ClassConversation, "k-fault", `"period":"9","grace":"0"`)
	if failed.status != 500 || failed.body["code"] != "internal_error" || again.status != 500 || again.body["operation_id"] != failed.body["operation_id"] ||
		f.s.Revision() != revision || f.s.ret.pendingPolicy(retention.ClassConversation) != nil {
		t.Errorf("internal error replay %s %s", failed.raw, again.raw)
	}
}

func TestRetentionLiveHoldCap(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	if _, err := f.s.retentionPlaceHold(retention.Hold{ID: "other-workspace-hold", Owner: SampleAdmin, Reason: "x", Artifacts: []string{"sample-claim-1"}}, retAnn,
		"other-tenant", "x"); err != nil {
		t.Fatal(err)
	}
	for n := len(f.s.ret.holds) - 1; n < retentionLiveHolds; n++ {
		if _, err := f.s.retentionPlaceHold(retention.Hold{ID: fmt.Sprint("cap-", n), Owner: SampleAdmin, Reason: "x", Artifacts: []string{"sample-claim-1"}}, retAnn,
			SampleWorkspace, "x"); err != nil {
			t.Fatal(err)
		}
	}
	place := map[string]string{"owner": SampleAdmin, "reason": "x"}
	target := retentionArtifactTarget + "sample-claim-1"
	f.unchanged(t, "cap", f.run(root, "retention.place_hold", target, place), "prerequisite_failed")
	f.unchanged(t, "tenant cap", f.run(root, "retention.place_hold", retentionTenantTarget+SampleWorkspace, place), "prerequisite_failed")
	f.ok(t, "release one", f.run(second, "retention.approve", retTarget(6), nil), "applied", "")
	f.ok(t, "after release", f.run(root, "retention.place_hold", target, place), "held", "")
	f.unchanged(t, "cap again", f.run(root, "retention.place_hold", target, place), "prerequisite_failed")
	at, calls, placed := 0, 0, false
	f.s.fault = func(step string) bool {
		if step == "retention-hold" {
			calls++
		}
		placed = placed || step == "retention-place_hold"
		return step == "retention-hold" && calls == at
	}
	f.ok(t, "release another", f.run(second, "retention.propose_release", retentionHoldTarget+"cap-2", map[string]string{"reason": "x"}), "pending", "")
	calls, at = 0, 1000
	f.ok(t, "approve", f.run(root, "retention.approve", retTarget(8), nil), "applied", "")
	revision := f.revision(root)
	calls, at = 0, len(f.s.ret.holds)
	placed = false
	f.code(t, "id probe fault", f.runWith(root, revision, "retention.place_hold", target, place), 500, "internal_error")
	f.s.fault = nil
	if placed {
		t.Error("a failed hold-ID probe still placed a hold")
	}
}

func TestRetentionLiveHoldCountFault(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	revision := f.revision(root)
	before := len(f.s.ret.holds)
	calls, placed := 0, false
	f.s.fault = func(step string) bool {
		if step == "retention-hold" {
			calls++
		}
		placed = placed || step == "retention-place_hold"
		return step == "retention-hold" && calls == 1
	}
	f.code(t, "count fault", f.runWith(root, revision, "retention.place_hold", retentionArtifactTarget+"sample-claim-1", map[string]string{"owner": SampleAdmin, "reason": "x"}),
		500, "internal_error")
	f.s.fault = nil
	if placed || calls != 1 || len(f.s.ret.holds) != before {
		t.Errorf("a failed live-hold count went on: placed %v, %d hold reads, %d holds", placed, calls, len(f.s.ret.holds))
	}
	if _, err := f.s.retention.Hold("retention-hold-1"); err == nil {
		t.Error("the module holds a new hold")
	}
}

func TestRetentionRequiredFields(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	absentHold, absentArtifact := retentionHoldTarget+"sample-absent", retentionArtifactTarget+"sample-absent"
	for _, c := range []struct {
		kind, target, field string
		fields              map[string]string
	}{
		{retentionPolicyKind, retentionClassTarget + "unknown", "period", map[string]string{"grace": "0"}},
		{retentionPolicyKind, retentionClassTarget + "unknown", "grace", map[string]string{"period": "1"}},
		{retentionHoldKind, absentArtifact, "owner", map[string]string{"reason": "x"}},
		{retentionHoldKind, absentArtifact, "reason", map[string]string{"owner": SampleAdmin}},
		{retentionReleaseKind, absentHold, "reason", map[string]string{}},
	} {
		f.fieldRefused(t, c.kind+" without "+c.field, f.run(root, retentionCommand+c.kind, c.target, c.fields), c.field)
	}
	if f.s.ret.holdSeq != 0 {
		t.Errorf("refused shapes advanced the hold counter to %d", f.s.ret.holdSeq)
	}
	for _, c := range []struct {
		name   string
		fields map[string]string
		low    int
		high   int
	}{{"missing", map[string]string{}, 0, 365}, {"text", map[string]string{"grace": "abc"}, 0, 365}, {"overflow", map[string]string{"grace": "99999999999999999999"}, 0, 365},
		{"low", map[string]string{"grace": "-1"}, 0, 365}, {"high", map[string]string{"grace": "366"}, 0, 365}} {
		if _, err := boundedField(c.fields, "grace", c.low, c.high); err == nil {
			t.Errorf("%s accepted", c.name)
		}
	}
	for value, want := range map[string]int{"0": 0, "365": 365} {
		if got, err := boundedField(map[string]string{"grace": value}, "grace", 0, 365); err != nil || got != want {
			t.Errorf("%s: %d %v", value, got, err)
		}
	}
}

func TestRetentionPolicyEdges(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	source := retentionClassTarget + retention.ClassSource
	impactOf := func(number int) string {
		row := retProposal(t, f.retentionRead(root), uint64(number))
		return fmt.Sprint(row["state"], " ", row["impact_required"], " ", row["impact_now"])
	}
	f.ok(t, "withdraw shortening", f.run(second, "retention.withdraw", retTarget(7), nil), "withdrawn", "")
	if s := impactOf(7); s != "withdrawn false <nil>" {
		t.Errorf("withdrawn shortening %s", s)
	}
	f.ok(t, "same sum", f.run(root, "retention.propose_policy", source, map[string]string{"period": "395", "grace": "0"}), "pending", "map[proposal:8 target:retention-proposal-8]")
	if s := impactOf(8); s != "pending false <nil>" {
		t.Errorf("same sum %s", s)
	}
	f.ok(t, "withdraw same sum", f.run(root, "retention.withdraw", retTarget(8), nil), "withdrawn", "")
	f.ok(t, "empty shortening", f.run(root, "retention.propose_policy", source, map[string]string{"period": "390", "grace": "0"}), "pending", "")
	now := retProposal(t, f.retentionRead(second), 9)["impact_now"].(map[string]any)
	if fmt.Sprint(now["count"], " ", now["workspace_count"]) != "0 0" {
		t.Errorf("empty impact %v", now)
	}
	f.ok(t, "approve with count 0", f.run(second, "retention.approve", retTarget(9), impactFields(now)), "applied", "")
	if s := f.currentPolicy(retention.ClassSource); s != "2 390 0 sample-admin sample-admin-2" {
		t.Errorf("policy %s", s)
	}
	f.ok(t, "same period, other grace", f.run(second, "retention.propose_policy", source, map[string]string{"period": "390", "grace": "30"}), "pending", "")
	withdrew := false
	f.s.fault = func(step string) bool {
		withdrew = withdrew || step == "retention-withdraw"
		return false
	}
	f.unchanged(t, "non-proposer withdraws applied", f.run(second, "retention.withdraw", retTarget(1), nil), "invalid_stage")
	f.s.fault = nil
	if withdrew {
		t.Error("withdrawing an applied entry called the module")
	}
}

func TestRetentionTargetEdges(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	hold, stop := retentionHandler{s: f.s, kind: retentionHoldKind}, retentionHandler{s: f.s, kind: retentionStopKind}
	if _, res := hold.Resolve("other-ws", retentionTenantTarget+"other-ws"); res != commands.NotFound {
		t.Error("a workspace other than the boot workspace placed a tenant hold")
	}
	for _, target := range []string{"1", "2", "run-1"} {
		if _, res := stop.Resolve(SampleWorkspace, target); res != commands.NotFound {
			t.Errorf("run target %s resolved", target)
		}
	}
	f.code(t, "bare run number", f.run(root, "retention.stop_run", "2", nil), 403, "permission_denied")
	if v := f.retRunView("run/2"); v.State != retention.RunRunning {
		t.Errorf("run %s", v.State)
	}
	f.s.fault = func(step string) bool { return step == "retention-read" }
	_, err := f.s.retentionPage(identity.Verifier{}, SampleAdmin, SampleWorkspace, start)
	f.s.fault = nil
	if !errors.Is(err, retention.ErrInvalid) {
		t.Errorf("read fault %v", err)
	}
}
