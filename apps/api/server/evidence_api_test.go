package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/myblueton-sw/hybrid_billing/modules/commands"
	"github.com/myblueton-sw/hybrid_billing/modules/perfgate"
	"github.com/myblueton-sw/hybrid_billing/modules/releasetrace"
)

var (
	evA1     = releasetrace.Actor{ID: SampleAdmin, Kind: releasetrace.KindHuman}
	evA2     = releasetrace.Actor{ID: SampleAdmin2, Kind: releasetrace.KindHuman}
	evAnn    = releasetrace.Actor{ID: "test-ann", Kind: releasetrace.KindHuman}
	evBob    = releasetrace.Actor{ID: "test-bob", Kind: releasetrace.KindHuman}
	evCat    = releasetrace.Actor{ID: "test-cat", Kind: releasetrace.KindHuman}
	evDan    = releasetrace.Actor{ID: "test-dan", Kind: releasetrace.KindHuman}
	evRunner = releasetrace.Actor{ID: "test-trace-runner", Kind: releasetrace.KindSystem}
	pfA1     = perfgate.Actor{ID: SampleAdmin, Kind: perfgate.KindHuman}
	pfA2     = perfgate.Actor{ID: SampleAdmin2, Kind: perfgate.KindHuman}
	pfAnn    = perfgate.Actor{ID: "test-ann", Kind: perfgate.KindHuman}
	pfBob    = perfgate.Actor{ID: "test-bob", Kind: perfgate.KindHuman}
	pfCat    = perfgate.Actor{ID: "test-cat", Kind: perfgate.KindHuman}
	pfDan    = perfgate.Actor{ID: "test-dan", Kind: perfgate.KindHuman}
	pfRunner = perfgate.Actor{ID: "test-perf-runner", Kind: perfgate.KindSystem}
)

func (f world) evidenceRead(a actor) reply {
	f.t.Helper()
	return f.do("GET", "/api/v1/release-evidence", "", func(r *http.Request) {
		if a.cookie != nil {
			r.AddCookie(a.cookie)
		}
		r.Header.Set(csrfHeader, a.csrf)
	})
}

func evMap(v any, path ...string) map[string]any {
	m, _ := v.(map[string]any)
	for _, name := range path {
		m, _ = m[name].(map[string]any)
	}
	return m
}

func evRows(list map[string]any) []map[string]any {
	out := []map[string]any{}
	rows, _ := list["rows"].([]any)
	for _, item := range rows {
		out = append(out, item.(map[string]any))
	}
	return out
}

func evFind(t testing.TB, rows []map[string]any, key, value string) map[string]any {
	t.Helper()
	for _, row := range rows {
		if fmt.Sprint(row[key]) == value {
			return row
		}
	}
	t.Fatalf("no row %s=%s", key, value)
	return nil
}

func evPath(t testing.TB, got reply, id string) map[string]any {
	t.Helper()
	return evFind(t, evRows(evMap(got.body, "matrix", "paths")), "id", id)
}

func evReq(t testing.TB, got reply, id string) map[string]any {
	t.Helper()
	return evFind(t, evRows(evMap(got.body, "requirements", "requirements")), "id", id)
}

func evProfile(t testing.TB, got reply, id string) map[string]any {
	t.Helper()
	return evFind(t, evRows(evMap(got.body, "load", "profiles")), "id", id)
}

func evProposal(t testing.TB, got reply, target string) map[string]any {
	t.Helper()
	return evFind(t, evRows(evMap(got.body, "decisions", "proposals")), "target", target)
}

func evBlocked(row map[string]any) string {
	return fmt.Sprint(row["can_approve"], " ", row["approve_blocked"], " ", row["can_withdraw"], " ", row["withdraw_blocked"])
}

func evRetire(row map[string]any) string {
	return fmt.Sprint(row["can_retire"], " ", row["retire_blocked"], " ", row["retire"])
}

func evJSON(v any) string {
	data, _ := json.Marshal(v)
	return string(data)
}

func evKey(id string) releasetrace.Key { return releasetrace.Key{Namespace: evidenceNamespace, ID: id} }

func traceTarget(n uint64) string { return fmt.Sprint(traceProposalTarget, n) }

func perfTarget(n uint64) string { return fmt.Sprint(perfProposalTarget, n) }

func (f world) evApproved(c releasetrace.Change) *evidenceEntry {
	f.t.Helper()
	e, err := f.s.tracePropose(c, evAnn, "injected")
	if err != nil {
		f.t.Fatal(err)
	}
	if err := f.s.traceApprove(e, evBob, "injected"); err != nil {
		f.t.Fatal(err)
	}
	return e
}

func (f world) pfApproved(c perfgate.Change) {
	f.t.Helper()
	e, err := f.s.perfPropose(c, pfAnn, "injected")
	if err != nil {
		f.t.Fatal(err)
	}
	if err := f.s.perfApprove(e, pfBob, "injected"); err != nil {
		f.t.Fatal(err)
	}
}

func evRequirement(id string, common, blocking bool, tests ...releasetrace.TestRef) releasetrace.Change {
	return releasetrace.Change{Kind: releasetrace.ChangeRequirement, Requirement: releasetrace.Requirement{Namespace: evidenceNamespace, ID: id, Source: "test-source",
		Owner: "test-owner", Tests: tests, Common: common, Blocking: blocking, Design: sampleHash("test:" + id)}}
}

func (f world) evAddRequirement(id string, common, blocking bool, tests ...releasetrace.TestRef) {
	f.t.Helper()
	f.evApproved(evRequirement(id, common, blocking, tests...))
	f.s.ev.requirements = append(f.s.ev.requirements, evKey(id))
}

func (f world) evHarness() {
	f.t.Helper()
	p, err := f.s.trace.Propose(releasetrace.Change{Kind: releasetrace.ChangeHarness, Harness: releasetrace.Harness{ID: "test-harness", Identity: evRunner.ID}}, evCat)
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.s.trace.Approve(p.Number, evDan); err != nil {
		f.t.Fatal(err)
	}
}

func (f world) evRecord(test, build string, passed bool) {
	f.t.Helper()
	k := evidenceResultKey{namespace: evidenceNamespace, test: test, build: build}
	if _, err := f.s.trace.Record(releasetrace.Result{Namespace: k.namespace, Test: test, Build: build, Passed: passed, Evidence: sampleHash("test:" + test)}, evRunner); err != nil {
		f.t.Fatal(err)
	}
	f.s.ev.addResult(k)
}

func (f world) perfHarness() {
	f.t.Helper()
	p, err := f.s.perf.Propose(perfgate.Change{Kind: perfgate.ChangeHarness, Harness: perfgate.Harness{ID: "test-perf-harness", Identity: pfRunner.ID}}, pfCat)
	if err != nil {
		f.t.Fatal(err)
	}
	if _, err := f.s.perf.Approve(p.Number, pfDan); err != nil {
		f.t.Fatal(err)
	}
}

func (f world) neverReleased(t *testing.T, got reply) {
	t.Helper()
	for _, word := range []string{`"ready":true`, `"state":"active"`, releasetrace.StateCustomerProven, `"verdict":"passed"`, `"status":"passed"`} {
		if strings.Contains(got.raw, word) {
			t.Errorf("the read carries %s", word)
		}
	}
	if c := f.s.ev.candidate; c != nil {
		if g, err := f.s.trace.Gate(*c); err != nil || g.Ready {
			t.Errorf("gate %+v %v", g, err)
		}
		if v, err := f.s.perf.Readiness(c.Build); err != nil || v.Ready {
			t.Errorf("readiness %+v %v", v, err)
		}
	}
	for _, id := range f.s.ev.paths {
		if v, err := f.s.trace.Path(id); err != nil || v.State != releasetrace.PathDesigned {
			t.Errorf("path %s %+v %v", id, v, err)
		}
	}
}

func TestEvidenceReadShape(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.evidenceRead(root)
	if got.status != 200 || got.body["state"] != "configured" || got.body["now"] != stamp(start) || got.body["revision"] != f.s.Revision() ||
		fmt.Sprint(got.body["candidate"]) != "map[build:0.0.2 release:not_connected version:sample-candidate-1]" {
		t.Fatalf("read %s", got.raw)
	}
	for _, section := range []string{"matrix", "requirements", "artifacts", "load", "decisions"} {
		if evMap(got.body, section)["origin"] != evidenceOriginSeed {
			t.Errorf("%s origin %v", section, evMap(got.body, section)["origin"])
		}
	}
	f.neverReleased(t, got)
	api := evPath(t, got, "sample-path-api")
	if s := fmt.Sprint(api["state"], api["build"], api["template"], api["issuance"], " ", api["columns"], " ", api["categories"], " ", api["last_observed"], " ",
		api["activation"], api["support_decision"], api["customer_proof_intake"]); s != "designed<nil> <nil>api map[customer_proof:map[count:0 of:3] design:map[count:3 of:3] "+
		"failing:1 implementation:map[count:3 of:3] test:map[count:2 of:3]] map[amount:true approval:true bulk:false duplicate:true isolation:true recovery:true] "+
		"2026-11-16T09:00:00Z not_connectednot_connectednot_connected" {
		t.Errorf("api path %s", s)
	}
	if s := fmt.Sprint(api["requirements"], " ", api["reconciliations"], " ", api["blockers"]); s != "map[more:false rows:[map[label:sample/sample-approval state:tested "+
		"target:trace-requirement-2] map[label:sample/sample-common-xlsx state:tested target:trace-requirement-1] map[label:sample/sample-recovery state:failing "+
		"target:trace-requirement-3]] total:3] map[more:false rows:[map[differences:0 matched:true period:2026-10]] total:1] [map[problem:blocking "+
		"subject:sample/sample-recovery] map[problem:not_tested subject:sample/sample-recovery] map[problem:path_evidence subject:sample-path-api] "+
		"map[problem:path_inactive subject:sample-path-api]]" {
		t.Errorf("api path lists %s", s)
	}
	xlsx := evPath(t, got, "sample-path-xlsx")
	if s := fmt.Sprint(xlsx["template"], " ", xlsx["columns"], " ", xlsx["blockers"], " ", xlsx["reconciliations"]); s != "sample-template-xlsx "+
		"map[customer_proof:map[count:0 of:2] design:map[count:2 of:2] failing:0 implementation:map[count:1 of:2] test:map[count:1 of:2]] "+
		"[map[problem:not_tested subject:sample/sample-erp-template] map[problem:path_evidence subject:sample-path-xlsx] map[problem:path_inactive subject:sample-path-xlsx] "+
		"map[problem:path_reconciliation subject:sample-path-xlsx]] map[more:false rows:[map[differences:3 matched:false period:2026-10]] total:1]" {
		t.Errorf("xlsx path %s", s)
	}
	states := []string{}
	for _, row := range evRows(evMap(got.body, "requirements", "requirements")) {
		states = append(states, fmt.Sprint(row["target"], ":", row["id"], ":", row["states"], ":", row["builds_more"], ":", evRetire(row), ":", row["replacement_required"]))
	}
	if s := strings.Join(states, "|"); s != "trace-requirement-1:sample-common-xlsx:map[0.0.1:implemented 0.0.2:tested]:false:false no_replacement <nil>:true|"+
		"trace-requirement-2:sample-approval:map[0.0.1:implemented 0.0.2:tested]:false:true  <nil>:false|"+
		"trace-requirement-3:sample-recovery:map[0.0.1:implemented 0.0.2:failing]:false:true  <nil>:false|"+
		"trace-requirement-4:sample-erp-template:map[0.0.1:designed 0.0.2:designed]:false:true  <nil>:false|"+
		"trace-requirement-5:sample-api-export:map[0.0.1:implemented 0.0.2:implemented]:false:true  <nil>:false|"+
		"trace-requirement-6:sample-legacy-report:map[0.0.1:implemented 0.0.2:implemented]:false:false already_proposed 19:false" {
		t.Errorf("requirements %s", s)
	}
	approval := evReq(t, got, "sample-approval")
	if s := fmt.Sprint(approval["tests"], " ", approval["replacements"], " ", approval["commit"], " ", approval["interfaces"], " ", approval["source"], " ",
		approval["owner"], " ", approval["common"], approval["blocking"], approval["retired"], approval["replacement"], " ", approval["design"] == sampleHash("sample:design:sample-approval")); s !=
		"map[more:false rows:[map[category:approval counted:true failures:1 id:t-approval latest_outcome:pass not_counted_reason:<nil> not_run_reason:<nil> passes:1] "+
			"map[category:duplicate counted:true failures:0 id:t-duplicate latest_outcome:pass not_counted_reason:<nil> not_run_reason:<nil> passes:1]] total:2] map[more:false rows:[map[label:sample/sample-recovery target:trace-requirement-3] "+
			"map[label:sample/sample-erp-template target:trace-requirement-4] map[label:sample/sample-api-export target:trace-requirement-5] "+
			"map[label:sample/sample-legacy-report target:trace-requirement-6]] total:4] "+sampleHash("sample:commit:1")[:40]+" [sample-api] sample-source sample-owner "+
			"false false false <nil> true" {
		t.Errorf("approval %s", s)
	}
	if s := fmt.Sprint(evReq(t, got, "sample-recovery")["tests"], " ", evReq(t, got, "sample-erp-template")["tests"], " ", evReq(t, got, "sample-erp-template")["commit"]); s !=
		"map[more:false rows:[map[category:bulk counted:true failures:1 id:t-bulk latest_outcome:fail not_counted_reason:<nil> not_run_reason:<nil> passes:0] "+
			"map[category:recovery counted:true failures:0 id:t-recovery latest_outcome:pass not_counted_reason:<nil> not_run_reason:<nil> passes:1]] total:2] "+
			"map[more:false rows:[map[category:<nil> counted:false failures:0 id:t-erp-template latest_outcome:<nil> not_counted_reason:<nil> "+
			"not_run_reason:no_result_recorded passes:0]] total:1] <nil>" {
		t.Errorf("tests %s", s)
	}
	if s := fmt.Sprint(evMap(got.body, "requirements", "gate")); s != "map[problems:map[more:false rows:[map[problem:blocking subject:sample/sample-recovery] "+
		"map[problem:not_tested subject:sample/sample-api-export] map[problem:not_tested subject:sample/sample-recovery] map[problem:path_evidence subject:sample-path-api] "+
		"map[problem:path_inactive subject:sample-path-api]] total:5] ready:false release:not_connected retired_blocking:[] templates:[]]" {
		t.Errorf("gate %s", s)
	}
	artifacts := evMap(got.body, "artifacts")
	if s := fmt.Sprint(artifacts["builds"], " ", artifacts["harnesses"], " ", artifacts["files"], artifacts["logs"], artifacts["environment"], artifacts["content"]); s !=
		"map[more:false rows:[map[build:0.0.2 commit:"+sampleHash("sample:commit:2")[:40]+" order:2] map[build:0.0.1 commit:"+sampleHash("sample:commit:0")[:40]+
			" order:1]] total:2] map[more:false rows:[map[id:sample-trace-harness module:release_trace revocation:not_connected] map[id:sample-perf-harness "+
			"module:perf_gate revocation:not_connected]] total:2] not_connectednot_connectednot_connectednot_connected" {
		t.Errorf("artifacts %s", s)
	}
	results := []string{}
	for _, row := range evRows(evMap(artifacts, "results")) {
		results = append(results, fmt.Sprint(row["test"], "@", row["build"], ":", row["counted"], ":", row["latest_outcome"], ":", row["passes"], "/", row["failures"], ":",
			row["harness"], ":", row["at"]))
	}
	if s := strings.Join(results, " "); s != "t-xlsx-amount@0.0.1:false:pass:1/0:sample-trace-harness:2026-11-16T09:00:00Z "+
		"t-bulk@0.0.2:true:fail:0/1:sample-trace-harness:2026-11-16T09:00:00Z t-recovery@0.0.2:true:pass:1/0:sample-trace-harness:2026-11-16T09:00:00Z "+
		"t-duplicate@0.0.2:true:pass:1/0:sample-trace-harness:2026-11-16T09:00:00Z t-approval@0.0.2:true:pass:1/1:sample-trace-harness:2026-11-16T09:00:00Z "+
		"t-xlsx-isolation@0.0.2:true:pass:1/0:sample-trace-harness:2026-11-16T09:00:00Z t-xlsx-amount@0.0.2:true:pass:1/0:sample-trace-harness:2026-11-16T09:00:00Z" {
		t.Errorf("results %s", s)
	}
	load := evMap(got.body, "load")
	if s := fmt.Sprint(load["hardware"], " ", load["measurement"], load["planning_targets"]); s != "map[more:false rows:[map[cpus:8 disk_gib:500 id:sample-hw-small "+
		"memory_mib:32768 spec:"+sampleHash("sample:hardware:sample-hw-small")+" storage:ssd]] total:1] not_connectednot_connected" {
		t.Errorf("hardware %s", s)
	}
	profiles := []string{}
	for _, row := range evRows(evMap(load, "profiles")) {
		verdicts := []string{}
		for _, run := range evRows(evMap(row, "runs")) {
			verdicts = append(verdicts, fmt.Sprint(run["verdict"]))
		}
		budget := "<nil>"
		if b := evMap(row, "active_budget"); b != nil {
			budget = fmt.Sprint(b["id"], ":", b["state"])
		}
		profiles = append(profiles, fmt.Sprint(row["target"], ":", row["gate"], ":", row["status"], ":", budget, ":", verdicts, ":", evRetire(row)))
	}
	if s := strings.Join(profiles, "|"); s != "perf-profile:sample-e2e:e2e:provisional:sample-budget-e2e:proposed:[provisional]:true  <nil>|"+
		"perf-profile:sample-daily:daily_inflow:over_budget:sample-budget-daily:proposed:[over_budget]:true  <nil>|"+
		"perf-profile:sample-month:monthly_close:incomplete:<nil>:[incomplete]:false already_proposed 9|perf-profile:sample-resume:interrupt_resume:not_run:<nil>:[]:true  <nil>" {
		t.Errorf("profiles %s", s)
	}
	daily := evProfile(t, got, "sample-daily")
	run := evRows(evMap(daily, "runs"))[0]
	if s := evJSON(run["metrics"]) + " " + evJSON(run["rows"]) + fmt.Sprint(" ", run["duration_s"], " ", run["harness"], " ", run["build"], " ", run["hardware"]); s !=
		`{"completion_ms":86400000,"db_io_bytes":1073741824,"lock_wait_ms":40,"max_rss_bytes":3221225472,"p95_ms":400,"p99_ms":1500,"resume_ms":null,`+
			`"throughput_rows_s":13,"wal_bytes":536870912} 1200000 86400 sample-perf-harness 0.0.2 sample-hw-small` {
		t.Errorf("daily run %s", s)
	}
	if s := evJSON(evMap(daily, "active_budget")) + " " + evJSON(evMap(daily, "axes")); s != `{"approval_by":null,"approved_at":null,"approved_by":null,`+
		`"approver":{"directory":true,"id":"sample-admin-2","login":"admin2@example.invalid"},"hardware":"sample-hw-small","id":"sample-budget-daily",`+
		`"limits":{"completion_ms":86400000,"max_rss_bytes":4294967296,"p95_ms":500,"p99_ms":1000,"throughput_rows_s":20},`+
		`"proposer":{"directory":true,"id":"sample-admin","login":"admin@example.invalid"},"state":"proposed"} {"accounts":0,"backlog_rows":0,"concurrent_closes":0,`+
		`"customers":0,"failure_injections":0,"fan_out":0,"peak_window_minutes":0,"retention_months":0,"rows_day":1200000,"rows_month":0,"rows_run":0,"shared_pools":0,`+
		`"skew_permille":0}` {
		t.Errorf("daily %s", s)
	}
	gates := []string{}
	for _, item := range evMap(load, "readiness")["gates"].([]any) {
		g := item.(map[string]any)
		gates = append(gates, fmt.Sprint(g["gate"], ":", g["status"]))
	}
	if s := strings.Join(gates, " ") + fmt.Sprint(" ", evMap(load, "readiness")["ready"]); s != "e2e:provisional daily_inflow:failing monthly_close:failing "+
		"interrupt_resume:not_run catch_up:not_run false" {
		t.Errorf("readiness %s", s)
	}
	decisions := evMap(got.body, "decisions")
	if s := fmt.Sprint(decisions["support_decisions"], decisions["activation_review"], decisions["ledger"], " ", evMap(decisions, "proposals")["total"]); s !=
		"not_connectednot_connectedin_memory 28" {
		t.Errorf("decisions %s", s)
	}
	rows := []string{}
	for _, row := range evRows(evMap(decisions, "proposals")) {
		rows = append(rows, fmt.Sprint(row["target"], ":", row["kind"], ":", row["state"], ":", evBlocked(row)))
	}
	if s := strings.Join(rows[:4], ",") + "|" + strings.Join(rows[9:11], ","); s != "perf-proposal-9:retire:pending:true  false not_proposer,"+
		"perf-proposal-8:harness:applied:false not_pending false not_pending,perf-proposal-7:budget:applied:false not_pending false not_pending,"+
		"perf-proposal-6:budget:applied:false not_pending false not_pending|trace-proposal-19:retire:pending:false own_proposal true ,"+
		"trace-proposal-18:reconcile:applied:false not_pending false not_pending" {
		t.Errorf("proposals %s", s)
	}
	harness := evProposal(t, got, traceTarget(14))
	if s := fmt.Sprint(harness["proposer"], " ", harness["approver"], " ", harness["closed_by"], " ", harness["rationale"], " ", harness["created_at"], " ",
		harness["summary"], " ", harness["module"], " ", harness["number"]); s != "map[directory:false id:sample-evidence-reviewer-1 login:<nil>] map[directory:false "+
		"id:sample-evidence-reviewer-2 login:<nil>] map[directory:false id:sample-evidence-reviewer-2 login:<nil>] Sample data 2026-11-16T09:00:00Z "+
		"map[harness:sample-trace-harness] release_trace 14" {
		t.Errorf("harness proposal %s", s)
	}
	summaries := []string{}
	for _, target := range []string{traceTarget(1), traceTarget(7), traceTarget(8), traceTarget(15), traceTarget(18), traceTarget(19), perfTarget(1), perfTarget(2),
		perfTarget(6), perfTarget(8), perfTarget(9)} {
		summaries = append(summaries, fmt.Sprint(evProposal(t, got, target)["summary"]))
	}
	if s := strings.Join(summaries, "|"); s != "map[label:sample/sample-common-xlsx]|map[build:0.0.1 commit:"+sampleHash("sample:commit:0")[:40]+"]|"+
		"map[commit:"+sampleHash("sample:commit:1")[:40]+" label:sample/sample-common-xlsx]|map[path:sample-path-api]|"+
		"map[differences:3 matched:false path:sample-path-xlsx period:2026-10]|map[label:sample/sample-legacy-report reason:Sample scope change replacement:<nil>]|"+
		"map[hardware:sample-hw-small]|map[gate:e2e profile:sample-e2e]|map[budget:sample-budget-e2e hardware:sample-hw-small profile:sample-e2e]|"+
		"map[harness:sample-perf-harness]|map[profile:sample-month]" {
		t.Errorf("summaries %s", s)
	}
	second := f.evidenceRead(f.as(Admin2Login))
	if s := evBlocked(evProposal(t, second, perfTarget(9))) + "|" + evBlocked(evProposal(t, second, traceTarget(19))); s != "false own_proposal true |true  false not_proposer" {
		t.Errorf("second view %s", s)
	}
}

func TestEvidenceReadWithoutSeed(t *testing.T) {
	none := setup(t, SeedNone).ready()
	tok := none.links()[""]
	if got := none.auth("redeem", `{"token":"`+tok+`","password":"`+pw+`","login":"first@example.invalid","email":"first@example.invalid"}`, none.pre()); got.status != 204 {
		t.Fatalf("activate %+v", got)
	}
	cookie, csrf, err := none.s.openFor(LoginTriple("first@example.invalid"), EmptyWorkspace, false)
	if err != nil {
		t.Fatal(err)
	}
	a := actor{cookie, csrf}
	called := []string{}
	none.s.fault = func(step string) bool {
		if step == "trace-gate" || step == "perf-readiness" {
			called = append(called, step)
			return true
		}
		return false
	}
	got := none.evidenceRead(a)
	none.s.fault = nil
	if got.status != 200 || got.body["state"] != "configured" || len(called) != 0 {
		t.Fatalf("empty %s %v", got.raw, called)
	}
	if s := fmt.Sprint(got.body["candidate"], " ", evMap(got.body, "matrix"), " ", evMap(got.body, "requirements"), " ", evMap(got.body, "artifacts"), " ",
		evMap(got.body, "load"), " ", evMap(got.body, "decisions")); s != "<nil> map[origin:empty paths:map[more:false rows:[] total:0]] "+
		"map[gate:<nil> next: origin:empty requirements:map[more:false rows:[] total:0]] map[builds:map[more:false rows:[] total:0] builds_next: content:not_connected "+
		"environment:not_connected files:not_connected harnesses:map[more:false rows:[] total:0] logs:not_connected origin:empty results:map[more:false rows:[] total:0] results_next:] "+
		"map[hardware:map[more:false rows:[] total:0] measurement:not_connected origin:empty planning_targets:not_connected profiles:map[more:false rows:[] total:0] "+
		"readiness:<nil>] map[activation_review:not_connected ledger:in_memory origin:empty proposal_history:map[dropped:0 kept:200 next: total:0] proposals:map[more:false rows:[] total:0] "+
		"support_decisions:not_connected]" {
		t.Errorf("empty read %s", s)
	}
	for _, c := range []struct{ kind, target string }{{evidenceApproveKind, traceTarget(1)}, {evidenceWithdrawKind, perfTarget(1)},
		{evidenceRetireReqKind, traceRequirementTarget + "1"}, {evidenceRetireProfKind, perfProfileTarget + "sample-e2e"}} {
		none.code(t, c.kind, none.runWith(a, none.revision(a), evidenceCommand+c.kind, c.target, evFields(c.kind)), 403, "permission_denied")
	}
}

func TestEvidenceWithoutCandidate(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	f.s.ev.candidate = nil
	called := []string{}
	f.s.fault = func(step string) bool {
		if step == "trace-gate" || step == "perf-readiness" {
			called = append(called, step)
			return true
		}
		return false
	}
	got := f.evidenceRead(root)
	f.s.fault = nil
	if got.status != 200 || len(called) != 0 || got.body["candidate"] != nil || evMap(got.body, "requirements")["gate"] != nil || evMap(got.body, "load")["readiness"] != nil {
		t.Fatalf("no candidate %s %v", got.raw, called)
	}
	for _, id := range []string{"sample-path-api", "sample-path-xlsx"} {
		if p := evPath(t, got, id); p["blockers"] != nil || p["last_observed"] != nil {
			t.Errorf("path %v", p)
		}
	}
	for _, row := range evRows(evMap(got.body, "requirements", "requirements")) {
		if row["tests"] != nil || fmt.Sprint(row["states"]) != "map[]" || row["builds_more"] != false {
			t.Errorf("requirement %v", row)
		}
	}
	for _, row := range evRows(evMap(got.body, "load", "profiles")) {
		if row["status"] != nil {
			t.Errorf("profile %v", row)
		}
	}
	api := evPath(t, got, "sample-path-api")
	if s := evJSON(api["requirements"]) + " " + evJSON(api["columns"]) + " " + evJSON(api["categories"]); s != `{"more":false,"rows":[{"label":"sample/sample-approval",`+
		`"state":null,"target":"trace-requirement-2"},{"label":"sample/sample-common-xlsx","state":null,"target":"trace-requirement-1"},{"label":"sample/sample-recovery",`+
		`"state":null,"target":"trace-requirement-3"}],"total":3} {"customer_proof":null,"design":{"count":3,"of":3},"failing":null,"implementation":{"count":3,"of":3},`+
		`"test":null} null` {
		t.Errorf("path without a candidate %s", s)
	}
	for _, step := range []string{"trace-path", "trace-requirement"} {
		f.s.fault = func(name string) bool { return name == step }
		f.code(t, step+" without a candidate", f.evidenceRead(root), 503, "dependency_unavailable")
	}
	f.s.fault = nil
	f.s.ev.paths = append(f.s.ev.paths, "sample-absent")
	f.code(t, "unknown stored path without a candidate", f.evidenceRead(root), 503, "dependency_unavailable")
	f.s.ev.paths = f.s.ev.paths[:len(f.s.ev.paths)-1]
	f.ok(t, "retire", f.run(f.as(Admin2Login), "release_evidence.retire_requirement", traceRequirementTarget+"5", map[string]string{"reason": "Unused"}), "pending", "")
}

func evFields(kind string) map[string]string {
	if kind == evidenceRetireReqKind {
		return map[string]string{"reason": "x"}
	}
	return map[string]string{}
}

func TestEvidenceAccess(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.setPassword(MemberLogin, pw)
	member, root := f.as(MemberLogin), f.as(AdminLogin)
	f.code(t, "member read", f.evidenceRead(member), 403, "permission_denied")
	f.code(t, "anonymous", f.evidenceRead(actor{}), 401, "unauthenticated")
	targets := map[string]string{evidenceApproveKind: perfTarget(9), evidenceWithdrawKind: traceTarget(19), evidenceRetireReqKind: traceRequirementTarget + "5",
		evidenceRetireProfKind: perfProfileTarget + "sample-e2e"}
	before := f.s.Revision()
	for kind, target := range targets {
		f.code(t, "member "+kind, f.runWith(member, f.revision(root), evidenceCommand+kind, target, evFields(kind)), 403, "permission_denied")
		h := evidenceHandler{s: f.s, kind: kind}
		if _, res := h.Resolve("other-ws", target); res != commands.NotFound {
			t.Errorf("%s resolved in another workspace", kind)
		}
		if _, res := h.Resolve(SampleWorkspace, target); res != commands.Found {
			t.Errorf("%s did not resolve %s", kind, target)
		}
	}
	f.s.bootWorkspace = "elsewhere"
	if got := f.evidenceRead(root); got.status != 200 || got.body["state"] != "not_configured" || got.body["matrix"] != nil {
		t.Errorf("other workspace %s", got.raw)
	}
	for kind, target := range targets {
		f.code(t, "other workspace "+kind, f.runWith(root, f.s.Revision(), evidenceCommand+kind, target, evFields(kind)), 403, "permission_denied")
	}
	f.s.bootWorkspace = SampleWorkspace
	for _, c := range []struct{ kind, target string }{
		{evidenceApproveKind, "trace-proposal-019"}, {evidenceApproveKind, "trace-proposal-x"}, {evidenceApproveKind, "trace-proposal-99"}, {evidenceApproveKind, "19"},
		{evidenceApproveKind, "perf-proposal-10"}, {evidenceApproveKind, "perf-proposal-"}, {evidenceWithdrawKind, "trace-proposal-18446744073709551616"},
		{evidenceApproveKind, "retention-proposal-7"}, {evidenceApproveKind, traceRequirementTarget + "1"}, {evidenceRetireReqKind, traceRequirementTarget + "0"},
		{evidenceRetireReqKind, traceRequirementTarget + "7"}, {evidenceRetireReqKind, traceRequirementTarget + "01"}, {evidenceRetireReqKind, traceRequirementTarget},
		{evidenceRetireReqKind, "sample-approval"}, {evidenceRetireReqKind, traceTarget(19)}, {evidenceRetireProfKind, perfProfileTarget + "sample-absent"},
		{evidenceRetireProfKind, "sample-e2e"}, {evidenceRetireProfKind, perfProfileTarget}, {evidenceRetireProfKind, perfTarget(9)},
	} {
		f.code(t, c.kind+" "+c.target, f.run(root, evidenceCommand+c.kind, c.target, evFields(c.kind)), 403, "permission_denied")
	}
	if f.s.Revision() != before {
		t.Errorf("refused commands moved the revision %s %s", before, f.s.Revision())
	}
	long := strings.Repeat("r", maxRationale+1)
	f.unchanged(t, "long rationale", f.admin(root, "release_evidence.withdraw", traceTarget(19), long, map[string]string{}), "invalid_request")
	f.unchanged(t, "no rationale", f.admin(root, "release_evidence.withdraw", traceTarget(19), "", map[string]string{}), "invalid_request")
	f.fieldRefused(t, "no reason", f.run(root, "release_evidence.retire_requirement", traceRequirementTarget+"5", nil), "reason")
	f.fieldRefused(t, "bad replacement form", f.run(root, "release_evidence.retire_requirement", traceRequirementTarget+"5",
		map[string]string{"reason": "x", "replacement": "a/b"}), "replacement")
}

func (f world) evAgrees(t *testing.T, got reply) {
	t.Helper()
	s := f.s
	build := ""
	if s.ev.candidate != nil {
		build = s.ev.candidate.Build
	}
	for _, row := range evRows(evMap(got.body, "matrix", "paths")) {
		v, err := s.trace.Path(row["id"].(string))
		if err != nil {
			t.Fatal(err)
		}
		if row["state"] != v.State || row["provider"] != v.Path.Provider || row["contract"] != v.Path.Contract || row["role"] != v.Path.Role ||
			row["version"] != v.Path.Version || fmtInt(evMap(row, "reconciliations")["total"]) != len(v.Reconciliations) {
			t.Errorf("path %v %+v", row, v)
		}
		tested, retired, categories := 0, false, map[string]bool{}
		refs := evRows(evMap(row, "requirements"))
		for index, k := range v.Path.Requirements {
			q, err := s.trace.Requirement(k, build)
			if err != nil {
				t.Fatal(err)
			}
			if index < len(refs) && (refs[index]["label"] != label(k) || refs[index]["state"] != q.State) {
				t.Errorf("path requirement %v %+v", refs[index], q)
			}
			if q.State == releasetrace.StateTested || q.State == releasetrace.StateCustomerProven {
				tested++
			}
			retired = retired || q.Retired
			for i, x := range q.Tests {
				categories[q.Requirement.Tests[i].Category] = categories[q.Requirement.Tests[i].Category] || x.Counted && x.Passed
			}
		}
		all := true
		for _, c := range releasetrace.Categories {
			all = all && categories[c]
			if evMap(row, "categories")[c] != categories[c] {
				t.Errorf("%s category %s", row["id"], c)
			}
		}
		if fmtInt(evMap(row, "columns", "test")["count"]) != tested {
			t.Errorf("%s test column %v", row["id"], row["columns"])
		}
		if s.ev.candidate == nil {
			continue
		}
		g, err := s.trace.Gate(releasetrace.Release{Version: s.ev.candidate.Version, Build: build, Requirements: v.Path.Requirements, Paths: []string{v.Path.ID}})
		if err != nil {
			t.Fatal(err)
		}
		evidence := false
		for _, p := range g.Problems {
			evidence = evidence || p.Problem == releasetrace.ProblemPathEvidence
		}
		blocked := strings.Contains(fmt.Sprint(row["blockers"]), "problem:path_evidence")
		if evidence != blocked || evidence == (tested == len(v.Path.Requirements) && !retired && all) {
			t.Errorf("%s path_evidence %v blockers %v tested %d retired %v categories %v", row["id"], evidence, blocked, tested, retired, categories)
		}
	}
	for _, row := range evRows(evMap(got.body, "requirements", "requirements")) {
		k := releasetrace.Key{Namespace: row["namespace"].(string), ID: row["id"].(string)}
		q, err := s.trace.Requirement(k, build)
		if err != nil {
			t.Fatal(err)
		}
		if row["retired"] != q.Retired || fmt.Sprint(row["commit"]) != fmt.Sprint(map[bool]any{true: q.Commit, false: nil}[q.Commit != ""]) || row["design"] != q.Requirement.Design {
			t.Errorf("requirement %v %+v", row, q)
		}
		for b, state := range evMap(row, "states") {
			if v, err := s.trace.Requirement(k, b); err != nil || v.State != state {
				t.Errorf("%s at %s: %v %+v", k.ID, b, state, v)
			}
		}
		for index, test := range evRows(evMap(row, "tests")) {
			x := q.Tests[index]
			if test["id"] != x.Test || test["counted"] != x.Counted || fmtInt(test["passes"]) != x.Passes || fmtInt(test["failures"]) != x.Failures ||
				(test["latest_outcome"] == nil) != (x.Passes+x.Failures == 0) {
				t.Errorf("test %v %+v", test, x)
			}
		}
		for _, r := range evRows(evMap(row, "replacements")) {
			m := s.ev.requirements[fmtInt(strings.TrimPrefix(r["target"].(string), traceRequirementTarget))-1]
			if err := s.trace.Check(retireCheck(k, m), SampleAdmin, ""); err != nil || label(m) != r["label"] {
				t.Errorf("replacement %v %v", r, err)
			}
		}
	}
	for _, row := range evRows(evMap(got.body, "artifacts", "results")) {
		v, err := s.trace.Results(row["namespace"].(string), row["test"].(string), row["build"].(string))
		if err != nil || row["counted"] != v.Counted || row["evidence"] != v.Result.Evidence || row["harness"] != v.Harness || row["at"] != stamp(v.At) ||
			fmtInt(row["passes"]) != v.Passes || fmtInt(row["failures"]) != v.Failures {
			t.Errorf("result %v %+v %v", row, v, err)
		}
	}
	for _, row := range evRows(evMap(got.body, "load", "hardware")) {
		h, err := s.perf.Hardware(row["id"].(string))
		if err != nil || fmtInt(row["cpus"]) != int(h.CPUs) || row["storage"] != h.Storage || row["spec"] != h.Spec {
			t.Errorf("hardware %v %+v", row, h)
		}
	}
	for _, row := range evRows(evMap(got.body, "load", "profiles")) {
		id := row["id"].(string)
		p, retired, err := s.perf.Profile(id)
		if err != nil || row["retired"] != retired || row["gate"] != p.Gate || row["dataset"] != p.Dataset || fmtInt(evMap(row, "axes")["rows_run"]) != int(p.Axes.RowsRun) {
			t.Errorf("profile %v %+v", row, p)
		}
		b, err := s.perf.ActiveBudget(id)
		if (err != nil) != (row["active_budget"] == nil) || err == nil && (evMap(row, "active_budget")["id"] != b.Budget.ID || evMap(row, "active_budget")["state"] != b.State) {
			t.Errorf("budget %v %+v", row["active_budget"], b)
		}
		runs, err := s.perf.Runs(id, 0, 1000)
		if err != nil || fmtInt(evMap(row, "runs")["total"]) != len(runs) {
			t.Errorf("runs %v %v", row["runs"], err)
		}
		for index, r := range evRows(evMap(row, "runs")) {
			v := runs[len(runs)-1-index]
			if fmtInt(r["number"]) != int(v.Number) || r["verdict"] != v.Verdict || r["at"] != stamp(v.At) {
				t.Errorf("run %v %+v", r, v)
			}
		}
	}
	if s.ev.candidate != nil {
		v, err := s.perf.Readiness(build)
		if err != nil || evMap(got.body, "load", "readiness")["ready"] != v.Ready {
			t.Errorf("readiness %+v", v)
		}
		for index, item := range evMap(got.body, "load", "readiness")["gates"].([]any) {
			if g := item.(map[string]any); g["status"] != v.Gates[index].Status || g["gate"] != v.Gates[index].Gate {
				t.Errorf("gate %v %+v", g, v.Gates[index])
			}
		}
		g, err := s.trace.Gate(*s.ev.candidate)
		if err != nil || evMap(got.body, "requirements", "gate")["ready"] != g.Ready || fmtInt(evMap(got.body, "requirements", "gate", "problems")["total"]) != len(g.Problems) {
			t.Errorf("gate %+v", g)
		}
	}
	for _, row := range evRows(evMap(got.body, "decisions", "proposals")) {
		number := uint64(fmtInt(row["number"]))
		var state, kind string
		var err error
		if row["module"] == moduleTrace {
			var v releasetrace.ProposalView
			v, err = s.trace.Proposal(number)
			state, kind = v.State, v.Change.Kind
		} else {
			var v perfgate.ProposalView
			v, err = s.perf.Proposal(number)
			state, kind = v.State, v.Change.Kind
		}
		switch {
		case err == nil && (row["state"] != state || row["kind"] != kind):
			t.Errorf("proposal %v %s %s", row, state, kind)
		case err != nil && row["state"] != releasetrace.ProposalWithdrawn && row["state"] != storeRetired:
			t.Errorf("proposal %v %v", row, err)
		}
	}
}

func TestEvidenceReadAgreesWithModules(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.evidenceRead(root)
	f.evAgrees(t, got)
	f.evHarness()
	f.evRecord("t-bulk", evidenceBuild, true)
	read := f.evidenceRead(root)
	f.evAgrees(t, read)
	f.neverReleased(t, read)
	api := evPath(t, read, "sample-path-api")
	if s := fmt.Sprint(api["blockers"], " ", evMap(api, "categories")["bulk"], " ", evReq(t, read, "sample-recovery")["states"]); s !=
		"[map[problem:path_inactive subject:sample-path-api]] true map[0.0.1:implemented 0.0.2:tested]" {
		t.Errorf("completed path %s", s)
	}
	f.evRecord("t-xlsx-isolation", evidenceBuild, false)
	read = f.evidenceRead(root)
	f.evAgrees(t, read)
	if s := fmt.Sprint(evPath(t, read, "sample-path-api")["blockers"], " ", evMap(evPath(t, read, "sample-path-api"), "categories")["isolation"]); s !=
		"[map[problem:not_tested subject:sample/sample-common-xlsx] map[problem:path_evidence subject:sample-path-api] map[problem:path_inactive subject:sample-path-api]] false" {
		t.Errorf("broken path %s", s)
	}
	f.evRecord("t-xlsx-isolation", evidenceBuild, true)
	f.evApproved(releasetrace.Change{Kind: releasetrace.ChangeRetire, Retire: releasetrace.Retire{Key: evKey("sample-approval"), Reason: "injected"}})
	read = f.evidenceRead(root)
	f.evAgrees(t, read)
	if s := fmt.Sprint(evPath(t, read, "sample-path-api")["blockers"]); s != "[map[problem:path_evidence subject:sample-path-api] map[problem:path_inactive subject:sample-path-api] "+
		"map[problem:retired subject:sample/sample-approval]]" {
		t.Errorf("retired path requirement %s", s)
	}
	f.neverReleased(t, read)
	if s := fmt.Sprint(evMap(read.body, "requirements", "gate")["retired_blocking"]); s != "[sample/sample-approval]" {
		t.Errorf("retired blocking %s", s)
	}
}

func TestEvidenceReadBlockers(t *testing.T) {
	f := setup(t, SeedSample).ready()
	f.withoutThirdAdmin()
	root, second := f.as(AdminLogin), f.as(Admin2Login)
	e, err := f.s.tracePropose(releasetrace.Change{Kind: releasetrace.ChangeImplemented, Implemented: releasetrace.Implemented{Key: evKey("sample-erp-template"),
		Commit: sampleHash("x")[:40]}}, evA1, "injected")
	if err != nil {
		t.Fatal(err)
	}
	if s := evBlocked(evProposal(t, f.evidenceRead(root), traceTarget(e.number))) + "|" + evBlocked(evProposal(t, f.evidenceRead(second), traceTarget(e.number))); s !=
		"false not_console_kind true |false not_console_kind false not_proposer" {
		t.Errorf("implemented blockers %s", s)
	}
	f.s.ev.entry(moduleTrace, 19).state = storeRetired
	if s := evBlocked(evProposal(t, f.evidenceRead(root), traceTarget(19))); s != "false store_retired false store_retired" {
		t.Errorf("store retired %s", s)
	}
	f.s.ev.entry(moduleTrace, 19).state = entryPending
	if _, err := f.s.perf.Approve(9, pfAnn); err != nil {
		t.Fatal(err)
	}
	if s := evBlocked(evProposal(t, f.evidenceRead(root), perfTarget(9))) + " " + evRetire(evProfile(t, f.evidenceRead(root), "sample-month")); s !=
		"false retired false not_proposer false retired 9" {
		t.Errorf("retired profile %s", s)
	}
	if got := f.run(root, "identity.suspend_user", SampleAdmin2, nil); got.status != 200 {
		t.Fatalf("suspend %s", got.raw)
	}
	read := f.evidenceRead(root)
	for _, id := range []string{"sample-approval", "sample-recovery", "sample-api-export"} {
		if s := evRetire(evReq(t, read, id)); s != "false no_eligible_approver <nil>" {
			t.Errorf("%s %s", id, s)
		}
	}
	if s := evRetire(evReq(t, read, "sample-legacy-report")) + "|" + evRetire(evProfile(t, read, "sample-e2e")); s != "false already_proposed 19|false no_eligible_approver <nil>" {
		t.Errorf("suspended %s", s)
	}
	if got := f.run(root, "identity.reinstate_user", SampleAdmin2, nil); got.status != 200 {
		t.Fatalf("reinstate %s", got.raw)
	}
	second = f.as(Admin2Login)
	f.evApproved(releasetrace.Change{Kind: releasetrace.ChangeRetire, Retire: releasetrace.Retire{Key: evKey("sample-api-export"), Reason: "injected"}})
	if s := evRetire(evReq(t, f.evidenceRead(root), "sample-api-export")); s != "false retired <nil>" {
		t.Errorf("retired requirement %s", s)
	}
	for _, size := range []int{115, 116} {
		id := strings.Repeat("p", size)
		f.pfApproved(perfgate.Change{Kind: perfgate.ChangeProfile, Profile: perfgate.Profile{ID: id, Gate: perfgate.GateE2E, Axes: perfgate.Axes{RowsRun: 100000},
			Dataset: sampleHash("d"), Expected: sampleHash("e")}})
		f.s.ev.profiles = append(f.s.ev.profiles, id)
		row := evProfile(t, f.evidenceRead(root), id)
		if s := evRetire(row) + " " + fmt.Sprint(len(row["target"].(string))); s != map[int]string{115: "true  <nil> 128", 116: "false target_too_long <nil> 129"}[size] {
			t.Errorf("%d: %s", size, s)
		}
	}
	f.ok(t, "longest target", f.run(root, "release_evidence.retire_profile", perfProfileTarget+strings.Repeat("p", 115), nil), "pending", "")
	f.code(t, "too long target", f.run(root, "release_evidence.retire_profile", perfProfileTarget+strings.Repeat("p", 116), nil), 422, "invalid_request")
	f.s.fault = func(step string) bool { return step == "trace-check" }
	f.code(t, "check fault", f.evidenceRead(root), 503, "dependency_unavailable")
	f.code(t, "check fault for the other administrator", f.evidenceRead(second), 503, "dependency_unavailable")
	f.s.fault = nil
}

func TestEvidenceReplacements(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	if !releasetrace.ValidText(evidenceCheckText) || f.s.trace.Check(retireCheck(evKey("sample-approval"), releasetrace.Key{}), SampleAdmin, "") != nil {
		t.Fatal("the read's check reason")
	}
	bare := retireCheck(evKey("sample-approval"), releasetrace.Key{})
	bare.Retire.Reason = ""
	if f.s.trace.Check(bare, SampleAdmin, "") != releasetrace.ErrInvalid {
		t.Error("a check without a reason is not invalid")
	}
	common := evReq(t, f.evidenceRead(root), "sample-common-xlsx")
	if s := fmt.Sprint(common["replacement_required"], " ", common["replacements"], " ", evRetire(common)); s != "true map[more:false rows:[] total:0] false no_replacement <nil>" {
		t.Errorf("common %s", s)
	}
	f.evAddRequirement("test-common-2", true, false, releasetrace.TestRef{ID: "t-common-2"})
	read := f.evidenceRead(root)
	f.evAgrees(t, read)
	common = evReq(t, read, "sample-common-xlsx")
	if s := fmt.Sprint(common["replacement_required"], " ", common["replacements"], " ", evRetire(common)); s !=
		"true map[more:false rows:[map[label:sample/test-common-2 target:trace-requirement-7]] total:1] true  <nil>" {
		t.Errorf("common with a replacement %s", s)
	}
	stored := f.s.ev.requirements
	for index, k := range stored {
		want := []string{}
		for m, other := range stored {
			if m != index && f.s.trace.Check(retireCheck(k, other), SampleAdmin, "") == nil {
				want = append(want, label(other))
			}
		}
		got := []string{}
		for _, r := range evRows(evMap(evReq(t, read, k.ID), "replacements")) {
			got = append(got, r["label"].(string))
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s replacements %v want %v", k.ID, got, want)
		}
	}
	if s := fmt.Sprint(evMap(evReq(t, read, "sample-recovery"), "replacements")["total"], evMap(evReq(t, read, "sample-erp-template"), "replacements")["total"]); s != "0 3" {
		t.Errorf("recovery and template %s", s)
	}
	f.ok(t, "propose with a replacement", f.run(root, "release_evidence.retire_requirement", traceRequirementTarget+"4", map[string]string{"reason": "Merged",
		"replacement": traceRequirementTarget + "5"}), "pending", "")
	read = f.evidenceRead(root)
	for _, id := range []string{"sample-approval", "sample-common-xlsx"} {
		if strings.Contains(fmt.Sprint(evReq(t, read, id)["replacements"]), "sample-api-export") {
			t.Errorf("%s offers a replacement named by a pending retirement", id)
		}
	}
	if s := fmt.Sprint(evProposal(t, read, traceTarget(21))["summary"]); s != "map[label:sample/sample-erp-template reason:Merged replacement:sample/sample-api-export]" {
		t.Errorf("summary %s", s)
	}
}

func TestEvidenceReadLeavesModulesUnchanged(t *testing.T) {
	f := setup(t, SeedSample).ready()
	before := f.evModule()
	revision := f.s.Revision()
	for _, login := range []string{AdminLogin, Admin2Login} {
		if got := f.evidenceRead(f.as(login)); got.status != 200 {
			t.Fatal(got.raw)
		}
	}
	if f.evModule() != before || f.s.Revision() != revision {
		t.Error("the read changed a module")
	}
	f.s.fault = func(step string) bool { return step == "roster" }
	f.code(t, "roster", f.evidenceRead(f.as(AdminLogin)), 503, "dependency_unavailable")
	f.s.fault = nil
}

func (f world) evModule() string {
	f.t.Helper()
	s := f.s
	out := []string{}
	for n := uint64(1); n <= 40; n++ {
		v, err := s.trace.Proposal(n)
		w, perr := s.perf.Proposal(n)
		out = append(out, fmt.Sprint(v, err, w, perr))
	}
	keys := append([]releasetrace.Key{}, s.ev.requirements...)
	sort.Slice(keys, func(a, b int) bool { return label(keys[a]) < label(keys[b]) })
	for _, k := range keys {
		for _, b := range []string{evidenceOldBuild, evidenceBuild} {
			v, err := s.trace.Requirement(k, b)
			out = append(out, fmt.Sprint(v, err))
		}
	}
	for _, k := range s.ev.results {
		v, err := s.trace.Results(k.namespace, k.test, k.build)
		out = append(out, fmt.Sprint(v, err))
	}
	for _, id := range s.ev.paths {
		v, err := s.trace.Path(id)
		out = append(out, fmt.Sprint(v, err))
	}
	for _, id := range s.ev.profiles {
		p, retired, err := s.perf.Profile(id)
		runs, rerr := s.perf.Runs(id, 0, 1000)
		b, berr := s.perf.ActiveBudget(id)
		out = append(out, fmt.Sprint(p, retired, err, runs, rerr, b, berr))
	}
	v, err := s.perf.Readiness(evidenceBuild)
	return strings.Join(out, "|") + fmt.Sprint(v, err)
}

func evTest(t testing.TB, got reply, requirement, test string) string {
	t.Helper()
	row := evFind(t, evRows(evMap(evReq(t, got, requirement), "tests")), "id", test)
	return fmt.Sprint(row["counted"], " ", row["latest_outcome"], " ", row["not_run_reason"], " ", row["not_counted_reason"])
}

func TestEvidenceNotRunReasons(t *testing.T) {
	f := setup(t, SeedSample).ready()
	root := f.as(AdminLogin)
	got := f.evidenceRead(root)
	if s := evTest(t, got, "sample-erp-template", "t-erp-template") + "|" + evTest(t, got, "sample-api-export", "t-api-export") + "|" +
		evTest(t, got, "sample-common-xlsx", "t-xlsx-amount"); s != "false <nil> no_result_recorded <nil>|false <nil> no_result_recorded <nil>|true pass <nil> <nil>" {
		t.Errorf("candidate build %s", s)
	}
	old := evFind(t, evRows(evMap(got.body, "artifacts", "results")), "build", evidenceOldBuild)
	if s := fmt.Sprint(old["counted"], " ", old["not_counted_reason"]); s != "false build_predates_implementation" {
		t.Errorf("0.0.1 row %s", s)
	}
	for _, row := range evRows(evMap(got.body, "artifacts", "results")) {
		if row["counted"] == true && row["not_counted_reason"] != nil {
			t.Errorf("counted row with a reason %v", row)
		}
	}
	f.s.ev.candidate.Build = evidenceOldBuild
	got = f.evidenceRead(root)
	if s := evTest(t, got, "sample-common-xlsx", "t-xlsx-amount") + "|" + evTest(t, got, "sample-common-xlsx", "t-xlsx-isolation"); s !=
		"false pass <nil> build_predates_implementation|false <nil> no_result_recorded <nil>" {
		t.Errorf("old candidate %s", s)
	}
	f.s.ev.candidate.Build = evidenceBuild
	f.evHarness()
	f.evRecord("t-erp-template", evidenceBuild, true)
	got = f.evidenceRead(root)
	if s := evTest(t, got, "sample-erp-template", "t-erp-template"); s != "false pass <nil> no_implementation" {
		t.Errorf("designed %s", s)
	}
	if row := evFind(t, evRows(evMap(got.body, "artifacts", "results")), "test", "t-erp-template"); row["not_counted_reason"] != noImplementation {
		t.Errorf("designed result row %v", row)
	}
	f.evRecord("t-api-export", evidenceBuild, true)
	if err := f.s.trace.RevokeHarness("test-harness", evAnn); err != nil {
		t.Fatal(err)
	}
	if s := evTest(t, f.evidenceRead(root), "sample-api-export", "t-api-export"); s != "false pass <nil> not_counted_by_module" {
		t.Errorf("revoked %s", s)
	}
	f.s.ev.entries = f.s.ev.entries[:0]
	got = f.evidenceRead(root)
	if s := evTest(t, got, "sample-api-export", "t-api-export"); s != "false pass <nil> not_counted_by_module" {
		t.Errorf("without entries %s", s)
	}
	if row := evFind(t, evRows(evMap(got.body, "artifacts", "results")), "test", "t-api-export"); row["not_counted_reason"] != notCountedByModule {
		t.Errorf("without entries result row %v", row)
	}
}
