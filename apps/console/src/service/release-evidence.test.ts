import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, adminClient } from "./admin";
import { budgetOf, evidenceOf, pathOf, personOf, profileOf, proposalOf, requirementOf, runOf, summaryOf, testOf } from "./release-evidence";
import { sample } from "./release-evidence.sample";

type Call = { url: string; init: RequestInit };

function fake(...replies: Array<[number, unknown]>) {
  const calls: Call[] = [];
  let i = 0;
  const f = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    const [status, body] = replies[Math.min(i++, replies.length - 1)];
    return new Response(JSON.stringify(body), { status });
  }) as unknown as typeof fetch;
  return { f, calls };
}

const clone = () => JSON.parse(JSON.stringify(sample)) as Record<string, any>;

test("the seeded read parses every section", () => {
  const p = evidenceOf(sample);
  assert.equal(p.state, "configured");
  assert.deepEqual(p.candidate, { version: "sample-candidate-1", build: "0.0.2", release: "not_connected" });
  const xlsx = p.matrix.paths.rows?.find((x) => x.id === "sample-path-xlsx");
  assert.deepEqual(xlsx?.columns.design, { count: 2, of: 2 });
  assert.deepEqual(xlsx?.columns.customer_proof, { count: 0, of: 2 });
  assert.equal(xlsx?.columns.failing, 0);
  assert.equal(xlsx?.build_set, false);
  assert.equal(xlsx?.blockers?.length, 4);
  assert.equal(xlsx?.reconciliations.rows?.[0].differences, 3);
  assert.equal(xlsx?.reconciliations.rows?.[0].matched, false);
  const reqs = p.requirements.requirements.rows ?? [];
  assert.equal(reqs.length, 6);
  assert.equal(reqs[0].retire_blocked, "no_replacement");
  assert.equal(reqs[0].replacement_required, true);
  assert.equal(reqs[3].tests?.rows?.[0].not_run_reason, "no_result_recorded");
  assert.equal(reqs[5].retire, 19);
  assert.equal(p.requirements.gate?.ready, false);
  assert.equal(p.requirements.gate?.problems.total, 5);
  assert.equal(p.artifacts.results.rows?.[0].not_counted_reason, "build_predates_implementation");
  const month = p.load.profiles.rows?.find((x) => x.id === "sample-month");
  assert.equal(month?.retire, 9);
  assert.equal(month?.active_budget, null);
  assert.equal(month?.budget_invalid, false);
  const e2e = p.load.profiles.rows?.find((x) => x.id === "sample-e2e");
  assert.equal(e2e?.active_budget?.approval_set, false);
  assert.equal(e2e?.runs.rows?.[0].metrics?.resume_ms, null);
  assert.equal(p.load.readiness?.gates?.[2].status, "failing");
  assert.equal(p.decisions.ledger, "in_memory");
  const perf9 = p.decisions.proposals.rows?.find((x) => x.target === "perf-proposal-9");
  assert.equal(perf9?.can_approve, true);
  assert.equal(perf9?.withdraw_blocked, "not_proposer");
  const harness = p.decisions.proposals.rows?.find((x) => x.target === "perf-proposal-8");
  assert.deepEqual(harness?.proposer, { id: "sample-evidence-reviewer-1", login: null, directory: false });
});

test("hardened parsers: wrong types, unsafe and negative integers, missing arrays and keys", () => {
  const r = requirementOf({ target: 1, can_retire: "yes", retire_blocked: "", retire: 2 ** 64, interfaces: "x", tests: [], replacement_required: "no" });
  assert.equal(r.target, "");
  assert.equal(r.can_retire, false);
  assert.equal(r.retire_blocked, "missing");
  assert.equal(r.retire, null);
  assert.equal(r.interfaces, null);
  assert.equal(r.tests, null);
  assert.equal(r.replacement_required, null);
  assert.equal(r.replacements.rows, null);
  assert.equal(requirementOf({ can_retire: true }).retire_blocked, "missing");
  assert.equal(requirementOf({ can_retire: true, retire_blocked: "" }).can_retire, true);
  const p = pathOf({ columns: { design: { count: -1, of: 1.5 }, test: "x", failing: Number.MAX_SAFE_INTEGER + 2 }, blockers: "none", categories: { amount: "yes" }, build: "0.0.1" });
  assert.deepEqual(p.columns.design, { count: null, of: null });
  assert.equal(p.columns.test, null);
  assert.equal(p.columns.failing, null);
  assert.equal(p.blockers, null);
  assert.deepEqual(p.categories, { amount: null });
  assert.equal(p.build_set, true);
  assert.equal(pathOf({ build: null }).build_set, false);
  assert.equal(pathOf({}).requirements.rows, null);
  const t = testOf({ id: "t", counted: 1, latest_outcome: 3, passes: -2 });
  assert.deepEqual([t.counted, t.latest_outcome, t.passes, t.failures], [null, null, null, null]);
  const run = runOf({ number: 0, metrics: { p95_ms: null, p99_ms: "1", wal_bytes: 3 } });
  assert.equal(run.number, null);
  assert.deepEqual(run.metrics, { p95_ms: null, wal_bytes: 3 });
  assert.equal(runOf({}).metrics, null);
});

test("people: null, unknown to the directory, null login and malformed", () => {
  assert.equal(personOf(null), null);
  assert.deepEqual(personOf({ id: "x", login: null, directory: false }), { id: "x", login: null, directory: false });
  assert.deepEqual(personOf({ id: "y" }), { id: "y", login: null, directory: null });
  assert.deepEqual(personOf("z"), { id: "", login: null, directory: null });
});

test("budgets: approval fields mark the budget, a malformed budget is flagged", () => {
  assert.equal(budgetOf({ state: "proposed", approval_by: null, approved_by: null, approved_at: null })?.approval_set, false);
  assert.equal(budgetOf({ state: "proposed", approval_by: { id: "a" } })?.approval_set, true);
  assert.equal(budgetOf({ state: "proposed", approved_at: "2026-11-17T00:00:00Z" })?.approval_set, true);
  assert.equal(profileOf({ active_budget: "x" }).budget_invalid, true);
  assert.equal(profileOf({ active_budget: null }).budget_invalid, false);
});

test("summaries keep plain values and drop unsafe numbers", () => {
  assert.deepEqual(summaryOf({ label: "a/b", replacement: null, matched: false, differences: 3, big: 2 ** 60, nested: { x: 1 } }), { label: "a/b", replacement: null, matched: false, differences: 3, big: "" });
  assert.equal(proposalOf({ number: -3 }).number, null);
});

test("the no-candidate shape reads documented nulls as nulls", () => {
  const s = clone();
  s.candidate = null;
  s.requirements.gate = null;
  s.load.readiness = null;
  s.matrix.paths.rows[0].columns.test = null;
  s.matrix.paths.rows[0].columns.customer_proof = null;
  s.matrix.paths.rows[0].columns.failing = null;
  s.matrix.paths.rows[0].blockers = null;
  s.matrix.paths.rows[0].categories = null;
  s.matrix.paths.rows[0].last_observed = null;
  s.requirements.requirements.rows[0].tests = null;
  s.load.profiles.rows[0].status = null;
  const p = evidenceOf(s);
  assert.equal(p.candidate, null);
  assert.equal(p.requirements.gate, null);
  assert.equal(p.load.readiness, null);
  const x = p.matrix.paths.rows?.[0];
  assert.deepEqual([x?.columns.test, x?.columns.customer_proof, x?.columns.failing, x?.blockers, x?.categories, x?.last_observed], [null, null, null, null, null, null]);
  assert.equal(p.requirements.requirements.rows?.[0].tests, null);
  assert.equal(p.load.profiles.rows?.[0].status, null);
});

test("whole-page states: not configured and unknown", () => {
  const n = evidenceOf({ revision: "r1", now: "2026-11-17T00:00:00Z", state: "not_configured" });
  assert.equal(n.state, "not_configured");
  assert.equal(n.matrix.paths.rows, null);
  assert.equal(evidenceOf({ state: "odd" }).state, "odd");
  assert.equal(evidenceOf({ candidate: "x" }).candidate?.version, "");
});

test("the read sends the given request ID", async () => {
  const { f, calls } = fake([200, sample]);
  const page = await adminClient("csrf", f).releaseEvidence("m01-abc");
  assert.equal(calls[0].url, "/api/v1/release-evidence");
  assert.equal((calls[0].init.headers as Record<string, string>)["X-Request-Id"], "m01-abc");
  assert.equal(page.revision, "r21");
});

test("command envelopes for every release evidence command", async () => {
  const { f, calls } = fake([200, { business_state: "pending", fields: { module: "release_trace", proposal: "20", target: "trace-proposal-20" } }]);
  const api = adminClient("csrf", f);
  const out = await api.command("release_evidence.retire_requirement", "trace-requirement-2", "r21", "why", { reason: "Scope", replacement: "trace-requirement-3" }, { request: "q1", key: "k1" });
  assert.deepEqual(out, { state: "pending", fields: { module: "release_trace", proposal: "20", target: "trace-proposal-20" } });
  assert.equal(calls[0].url, "/api/v1/commands/release_evidence.retire_requirement");
  assert.deepEqual(JSON.parse(String(calls[0].init.body)), { request_id: "q1", target_id: "trace-requirement-2", idempotency_key: "k1", expected_revision: "r21", rationale: "why", body: { reason: "Scope", replacement: "trace-requirement-3" } });
  await api.command("release_evidence.retire_requirement", "trace-requirement-4", "r21", "why", { reason: "Scope" }, { request: "q2", key: "k2" });
  assert.deepEqual(JSON.parse(String(calls[1].init.body)).body, { reason: "Scope" });
  for (const [id, target] of [["release_evidence.retire_profile", "perf-profile:sample-month"], ["release_evidence.approve", "perf-proposal-9"], ["release_evidence.withdraw", "trace-proposal-19"]] as const) {
    await api.command(id, target, "r21", "why", {}, { request: "q", key: "k" });
    const last = calls[calls.length - 1];
    assert.equal(last.url, "/api/v1/commands/" + id);
    assert.deepEqual(JSON.parse(String(last.init.body)).body, {});
  }
});

test("a resend with the same attempt is byte identical", async () => {
  const { f, calls } = fake([503, { code: "dependency_unavailable" }], [200, { business_state: "pending", fields: {} }]);
  const api = adminClient("csrf", f);
  const send = () => api.command("release_evidence.retire_profile", "perf-profile:sample-resume", "r21", "why", {}, { request: "q", key: "k" });
  await assert.rejects(send, (e: unknown) => e instanceof AdminError && e.failure === "busy");
  await send();
  assert.equal(calls[0].init.body, calls[1].init.body);
});

test("refusal mapping: stage, refused, invalid with fields, internal error with and without an operation", async () => {
  const cases: Array<[number, Record<string, unknown>, string, string[]]> = [
    [409, { code: "invalid_stage" }, "stage", []],
    [409, { code: "prerequisite_failed" }, "refused", []],
    [400, { code: "invalid_request", fields: ["reason"] }, "invalid", ["reason"]],
    [400, { code: "invalid_request", fields: ["replacement"] }, "invalid", ["replacement"]],
    [400, { code: "invalid_request", fields: ["rationale"] }, "invalid", ["rationale"]],
    [400, { code: "invalid_request", fields: ["target_id"] }, "invalid", ["target_id"]],
    [500, { code: "internal_error", operation_id: "op-1" }, "partial", []],
    [500, { code: "internal_error" }, "unavailable", []],
  ];
  for (const [status, body, failure, fields] of cases) {
    const { f } = fake([status, body]);
    await assert.rejects(adminClient("csrf", f).command("release_evidence.approve", "perf-proposal-9", "r21", "why", {}, { request: "q", key: "k" }), (e: unknown) => {
      return e instanceof AdminError && e.failure === failure && JSON.stringify(e.fields) === JSON.stringify(fields) && e.code === String(body.code);
    });
  }
});

test("step-up: one byte-identical resend after confirmation, cancel gives reauth_cancelled", async () => {
  const refusal = { code: "reauthentication_required" };
  const ok = fake([403, refusal], [200, { business_state: "applied", fields: {} }]);
  let asked = 0;
  const out = await adminClient("csrf", ok.f, async () => { asked++; return true; }).command("release_evidence.approve", "perf-proposal-9", "r21", "why", {}, { request: "q", key: "k" });
  assert.equal(out.state, "applied");
  assert.equal(asked, 1);
  assert.equal(ok.calls.length, 2);
  assert.equal(ok.calls[0].init.body, ok.calls[1].init.body);
  assert.equal((ok.calls[1].init.headers as Record<string, string>)["X-Request-Id"], "q");
  const no = fake([403, refusal]);
  await assert.rejects(adminClient("csrf", no.f, async () => false).command("release_evidence.approve", "perf-proposal-9", "r21", "why", {}, { request: "q", key: "k" }), (e: unknown) => e instanceof AdminError && e.failure === "reauth_cancelled");
  assert.equal(no.calls.length, 1);
  const again = fake([403, refusal], [403, refusal]);
  await assert.rejects(adminClient("csrf", again.f, async () => true).command("release_evidence.approve", "perf-proposal-9", "r21", "why", {}, { request: "q", key: "k" }), (e: unknown) => e instanceof AdminError && e.failure === "reauth");
  assert.equal(again.calls.length, 2);
});
