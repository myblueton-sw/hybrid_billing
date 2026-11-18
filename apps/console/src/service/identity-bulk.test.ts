import { test } from "node:test";
import assert from "node:assert/strict";
import { adminClient, AdminError, attempt } from "./admin";
import { batches, bulkClient, bulkPageOf, detailOf, stage, StageError, stagedBatches } from "./identity-bulk";

type Call = { url: string; body: Record<string, unknown> };

function scripted(replies: [number, unknown, Record<string, string>?][]) {
  const calls: Call[] = [];
  const f = (async (url: string, init: RequestInit) => {
    calls.push({ url, body: init.body ? JSON.parse(String(init.body)) : {} });
    const [status, body, headers] = replies.shift() ?? [200, {}];
    return new Response(JSON.stringify(body), { status, headers: headers ?? {} });
  }) as unknown as typeof fetch;
  return { f, calls };
}

const id = (n: number) => "user-" + String(n).padStart(16, "0");

test("batches stay within 4096 bytes, keep the order and count multi-byte characters", () => {
  const ids = Array.from({ length: 500 }, (_, i) => id(i));
  const groups = batches(ids);
  assert.deepEqual(groups.flat(), ids);
  for (const g of groups) assert.ok(new TextEncoder().encode(g.join("\n")).length <= 4096);
  assert.equal(groups[0].length, 186);
  const wide = batches(["가".repeat(10), "가".repeat(10)], 60);
  assert.equal(batches(["가".repeat(10), "가".repeat(10)], 61).length, 1);
  assert.equal(wide.length, 2);
  assert.equal(batches(["a", "b"], 3).length, 1);
  assert.deepEqual(batches([]), []);
  assert.equal(stagedBatches(ids, 186), 1);
  assert.equal(stagedBatches(ids, 0), 0);
  assert.equal(stagedBatches(ids, 500), 3);
});

test("staging chains revisions, resends after a conflict with the same key and waits on a rate limit", async () => {
  const ids = Array.from({ length: 300 }, (_, i) => id(i));
  const { f, calls } = scripted([
    [200, { status: "completed", result_revision: "r2", fields: { draft: "bulk-draft-1", count: "186" } }],
    [409, { code: "revision_conflict" }],
    [429, { code: "rate_limited" }, { "Retry-After": "2" }],
    [200, { status: "completed", fields: { draft: "bulk-draft-1", count: "300" } }],
  ]);
  const progress: unknown[] = [];
  const waits: number[] = [];
  let reloads = 0;
  const out = await stage(adminClient("t", f), ids, { rationale: "why", revision: "r1", reload: async () => `r${10 + reloads++}`, progress: (p) => progress.push(p),
    wait: async (s) => { waits.push(s); } });
  assert.deepEqual(out, { draft: "bulk-draft-1", staged: 300, revision: "r11" });
  assert.equal(calls[0].body.expected_revision, "r1");
  assert.equal((calls[0].body.body as Record<string, string>).draft, undefined);
  assert.equal(calls[1].body.expected_revision, "r2");
  assert.equal(calls[2].body.expected_revision, "r10");
  assert.equal(calls[1].body.idempotency_key, calls[2].body.idempotency_key);
  assert.equal(calls[2].body.idempotency_key, calls[3].body.idempotency_key);
  assert.equal((calls[1].body.body as Record<string, string>).draft, "bulk-draft-1");
  assert.deepEqual(waits, [2]);
  assert.deepEqual(progress, [{ sent: 186, total: 300, waiting: false }, { sent: 186, total: 300, waiting: false, resent: true }, { sent: 186, total: 300, waiting: true },
    { sent: 300, total: 300, waiting: false }]);
});

test("any other failure stops with the exact staged count and the draft, and staging can resume", async () => {
  const ids = Array.from({ length: 300 }, (_, i) => id(i));
  const { f } = scripted([[200, { status: "completed", result_revision: "r2", fields: { draft: "bulk-draft-4" } }], [503, { code: "dependency_unavailable" }], [429, { code: "rate_limited" }]]);
  const tries = [attempt(), attempt()];
  try {
    await stage(adminClient("t", f), ids, { rationale: "x", revision: "r1", reload: async () => "r1", progress: () => {}, tries, wait: async () => {} });
    assert.fail("no error");
  } catch (e) {
    assert.ok(e instanceof StageError);
    assert.equal(e.staged, 186);
    assert.equal(e.draft, "bulk-draft-4");
    assert.equal(e.revision, "r2");
    assert.ok(e.cause instanceof AdminError && e.cause.failure === "busy");
  }
  const again = scripted([[200, { status: "completed", result_revision: "r5", fields: {} }]]);
  const out = await stage(adminClient("t", again.f), ids, { rationale: "x", revision: "r4", reload: async () => "r4", progress: () => {}, tries, draft: "bulk-draft-4", from: 1 });
  assert.deepEqual(out, { draft: "bulk-draft-4", staged: 300, revision: "r5" });
  assert.equal(again.calls[0].body.idempotency_key, tries[1].key);
  const timed = scripted([[429, { code: "rate_limited" }], [200, { status: "completed", result_revision: "r2", fields: { draft: "bulk-draft-1" } }]]);
  const started = Date.now();
  await stage(adminClient("t", timed.f), [id(1)], { rationale: "x", revision: "r1", reload: async () => "r1", progress: () => {} });
  assert.ok(Date.now() - started >= 900);
});

test("the bulk reads build their paths and default missing parts", async () => {
  const { f, calls } = scripted([
    [200, { revision: "r1", can_prepare: true, jobs: [{ id: "bulk-revocation-1", state: "executed", effect: "suspend_user", job_state: "running", counts: { targets: 2 } }], next: "bulk-revocation-1" }],
    [200, { job: { id: "bulk-revocation-1", state: "prepared" }, impact: { people: 2, pending: [{ store: "m03", count: 1, rechecked: true }] }, exclusions: { self: 1 }, outside_scope: ["B7"] }],
    [200, { total: 3, next: "2", items: [{ user_id: "u1", outcome: "held", receipt: { sequence: 4 }, cleanup: [{ item: "B1", state: "done" }] }] }],
    [200, { total: 1, targets: [{ user_id: "u1", installation_administrator: true }] }],
    [200, { total: 2, next: "1", exclusions: [{ user_id: "u2", login: "bo", reason: "self" }, {}] }],
  ]);
  const b = bulkClient(adminClient("t", f));
  const page = await b.list("bulk-revocation-9", 10);
  assert.equal(calls[0].url, "/api/v1/identity/bulk-revocations?limit=10&after=bulk-revocation-9");
  assert.deepEqual([page.can_prepare, page.jobs[0].effect, page.jobs[0].counts.targets, page.next], [true, "suspend_user", 2, "bulk-revocation-1"]);
  const detail = await b.detail("bulk-revocation-1");
  assert.deepEqual([detail.impact.people, detail.impact.pending[0], detail.exclusions.self, detail.outside_scope], [2, { store: "m03", count: 1, rechecked: true }, 1, ["B7"]]);
  const items = await b.items("bulk-revocation-1", 2, 1, "held");
  assert.equal(calls[2].url, "/api/v1/identity/bulk-revocations/bulk-revocation-1/items?limit=1&after=2&outcome=held");
  assert.deepEqual([items.total, items.next, items.rows[0].outcome, items.rows[0].receipt.sequence, items.rows[0].cleanup[0].item], [3, "2", "held", 4, "B1"]);
  const targets = await b.targets("bulk-revocation-1");
  assert.equal(calls[3].url, "/api/v1/identity/bulk-revocations/bulk-revocation-1/targets?limit=50");
  assert.equal(targets.rows[0].installation_administrator, true);
  const excluded = await b.excluded("bulk-revocation-1", 1, 1);
  assert.equal(calls[4].url, "/api/v1/identity/bulk-revocations/bulk-revocation-1/targets?kind=excluded&limit=1&after=1");
  assert.deepEqual([excluded.total, excluded.rows[0], excluded.rows[1]], [2, { user_id: "u2", login: "bo", reason: "self" }, { user_id: "", login: "", reason: "" }]);
  assert.equal(bulkPageOf({ jobs: [{ staged: 7 }] }).jobs[0].staged, 7);
  const empty = bulkPageOf({});
  assert.deepEqual([empty.jobs, empty.can_prepare, empty.clock.wall_lead], [[], false, null]);
  const odd = bulkPageOf({ jobs: [{ effect: "remove", job_state: "done", counts: { targets: "x" } }] });
  assert.deepEqual([odd.jobs[0].effect, odd.jobs[0].job_state, odd.jobs[0].counts.targets], ["", "", 0]);
  assert.deepEqual(detailOf({}).impact.pending, []);
});
