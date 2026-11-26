import { test } from "node:test";
import assert from "node:assert/strict";
import { adminClient, AdminError } from "./admin";
import { ReadParseError } from "./strict-read";
import { listedTrustChangeOf, trustChangeDetailOf, trustChangePageOf } from "./trust-changes";
import { trustWire } from "./trust-changes.sample";

const copy = (v: unknown): any => JSON.parse(JSON.stringify(v));
function client(body: unknown, inspect?: (path: string, init: RequestInit) => void) {
  return adminClient("csrf", (async (path, init) => { inspect?.(String(path), init!); return new Response(JSON.stringify(body)); }) as typeof fetch);
}
const invalid = (e: unknown) => e instanceof AdminError && e.failure === "invalid";
type Loose = Record<string, unknown> & { rows: Record<string, unknown>[]; query: Record<string, unknown>; count: Record<string, unknown> };
const whole = (): Loose => { const p = copy(trustWire.list) as Loose; p.next = ""; p.query.limit = 25; p.count.value = "3"; return p; };

test("M02 changes: the service's own answers are accepted as it sends them", async () => {
  const paths: string[] = [];
  const see = (path: string, init: RequestInit) => { paths.push(path); assert.equal(init.method, "GET"); };
  const first = await client(trustWire.list, see).trustChanges({ limit: 3 });
  const open = await client(trustWire.open, see).trustChanges({ state: "open", q: " TRUST- " });
  const held = await client(trustWire.held, see).trustChanges({});
  const detail = await client(trustWire.detail, see).trustChange(trustWire.detail.change.id);
  assert.deepEqual(paths, ["/api/v1/identity/policy/changes?limit=3", "/api/v1/identity/policy/changes?q=TRUST-&state=open", "/api/v1/identity/policy/changes",
    "/api/v1/identity/policy/changes/" + trustWire.detail.change.id]);
  assert.deepEqual([first.rows.length, first.next, first.count], [3, "cursor-1", { value: "7", certainty: "exact" }]);
  assert.deepEqual(first.rows.map((r) => r.state), ["independently_approved", "validated", "draft"]);
  assert.deepEqual([first.rows[0].can, first.rows[0].why_not, first.rows[0].decide_in_detail, first.rows[0].evidence_current],
    [{ revise: true, validate: true, reject: false, cancel: true }, { approve: "self", activate: "self" }, true, "not_evaluated"]);
  assert.ok(open.rows.length > 0 && open.rows.every((r) => ["draft", "validated", "independently_approved", "activation_unknown"].includes(r.state)));
  assert.deepEqual([held.rows[0].state, held.rows[0].decide_in_detail, held.rows[0].why_not.activate], ["activation_unknown", false, "trust_unavailable"]);
  assert.deepEqual([detail.change.id, detail.change.state, detail.change.history.length > 0, detail.revision], [trustWire.detail.change.id, trustWire.detail.change.state, true, "r11"]);
  const closed = await client(trustWire.closed, see).trustChanges({ state: "closed" });
  assert.deepEqual(closed.rows.map((r) => r.state).sort(), ["activated", "activated", "cancelled", "rejected", "superseded"]);
  const earlier = closed.rows.find((r) => r.earlier_run)!;
  const superseded = closed.rows.find((r) => r.state === "superseded")!;
  const activated = closed.rows.find((r) => r.state === "activated" && !r.earlier_run)!;
  assert.deepEqual([earlier.number, earlier.run_started_at !== "", earlier.proposer], [0, true, ""]);
  assert.deepEqual([superseded.superseded_by, activated.invalidation_state], [activated.id, "complete"]);
  const second = await client({ ...copy(trustWire.list), next: "" }, see).trustChanges({ limit: 3, after: "cursor-1" });
  assert.equal(paths[5], "/api/v1/identity/policy/changes?limit=3&after=cursor-1");
  assert.equal(second.count.value, "7");
});

test("M02 changes: a page that is not what was asked for is refused", async () => {
  const cases: [string, unknown, Parameters<ReturnType<typeof client>["trustChanges"]>[0]][] = [
    ["another state than asked", trustWire.open, { state: "closed", q: "TRUST-" }],
    ["a state filter that was not asked", trustWire.open, { q: "TRUST-" }],
    ["no state filter where one was asked", { ...copy(trustWire.open), query: { q: "trust-", state: "", limit: 25 } }, { state: "open", q: "TRUST-" }],
    ["another search input than typed", trustWire.open, { state: "open", q: "trust-" }],
    ["another search than asked", { ...copy(trustWire.open), query: { q: "trust", state: "open", limit: 25 } }, { state: "open", q: "TRUST-" }],
    ["a search that was not asked", trustWire.open, { state: "open" }],
    ["another limit than asked", trustWire.list, { limit: 4 }],
    ["a row outside the filter", { ...copy(trustWire.open), count: { value: "4", certainty: "exact" }, rows: [...copy(trustWire.open.rows), { ...copy(trustWire.held.rows[0]), state: "cancelled", why_not: { approve: "not_validated", activate: "needs_approval" } }] }, { state: "open", q: "TRUST-" }],
    ["a row the search does not match", { ...whole(), query: { q: "zzz", state: "", limit: 25 }, search_input: "zzz" }, { q: "zzz" }],
    ["a whole list that counts more than it carries", { ...whole(), count: { value: "4", certainty: "exact" } }, {}],
  ];
  for (const [name, body, query] of cases) await assert.rejects(client(body).trustChanges(query), invalid, name);
  await assert.rejects(client(trustWire.detail).trustChange("trust-00000000-1"), invalid, "a detail of another change");
  const beyond = { ...whole(), rows: [], count: { value: "0", certainty: "exact" }, search_input: "İx", query: { q: "i̇x", state: "", limit: 25 } };
  assert.equal((await client(beyond).trustChanges({ q: "İx" })).rows.length, 0, "a search the service lower-cases by its own rules");
  await assert.rejects(client({ ...beyond, query: { q: "", state: "", limit: 25 } }).trustChanges({ q: "İx" }), invalid, "a search that was dropped");
});

test("M02 changes: a malformed page or row is refused", () => {
  assert.doesNotThrow(() => trustChangePageOf(whole()));
  const page = (change: (p: Loose) => void) => { const p = whole(); change(p); return p; };
  const row = (change: (r: Record<string, unknown>) => void, at = 0) => page((p) => change(p.rows[at]));
  const bad: [string, unknown][] = [
    ["an unknown field on the page", page((p) => { p.extra = 1; })],
    ["a missing field on the page", page((p) => { delete p.order; })],
    ["another order", page((p) => { p.order = "number_desc"; })],
    ["a lower-bound count", page((p) => { p.count.certainty = "lower_bound"; })],
    ["a count below the rows", page((p) => { p.count.value = "2"; })],
    ["more rows than the limit", page((p) => { p.query.limit = 2; })],
    ["a continuation after a short page", page((p) => { p.next = "cursor-2"; })],
    ["a stopped scan without a continuation", page((p) => { p.scan_stopped = true; })],
    ["an observation before the reference", page((p) => { p.observed_at = "2020-01-01T00:00:00Z"; })],
    ["a repeated change", page((p) => { p.rows[1] = copy(p.rows[0]); })],
    ["an unknown field on a row", row((r) => { r.impact = null; })],
    ["a full row's answer to approve", row((r) => { (r.can as Record<string, unknown>).approve = true; })],
    ["an unknown state", row((r) => { r.state = "pending"; })],
    ["an unknown reason", row((r) => { (r.why_not as Record<string, unknown>).approve = "evidence_stale"; })],
    ["an awaiting change that is not left to the detail", row((r) => { r.decide_in_detail = false; })],
    ["evidence called not applicable on a validated awaiting change", row((r) => { r.evidence_current = "not_applicable"; })],
    ["evidence called not evaluated on a draft", row((r) => { r.evidence_current = "not_evaluated"; }, 2)],
    ["a successor on a change that is not superseded", row((r) => { r.superseded_by = "trust-00000000-9"; })],
    ["invalidation on a change that is not activated", row((r) => { r.invalidation_state = "complete"; })],
    ["a proposer who may also reject", row((r) => { (r.can as Record<string, unknown>).reject = true; })],
    ["a sign-in lifetime of zero", row((r) => { (r.candidate as Record<string, unknown>).max_age_seconds = 0; })],
    ["no sign-in method", row((r) => { (r.candidate as Record<string, unknown>).methods = []; })],
    ["a malformed time", row((r) => { r.act_until = "soon"; })],
    ["a number as text", row((r) => { r.number = "7"; })],
    ["an echoed limit of zero", page((p) => { p.query.limit = 0; p.rows = []; p.count.value = "0"; })],
    ["an echoed limit above a hundred", page((p) => { p.query.limit = 101; })],
    ["a continuation longer than a cursor", page((p) => { p.next = "c".repeat(4097); p.query.limit = 3; })],
    ["more work than a page can take", page((p) => { p.work_units = 10001; })],
    ["nine sign-in methods", row((r) => { (r.candidate as Record<string, unknown>).methods = ["a", "b", "c", "d", "e", "f", "g", "h", "i"]; })],
    ["a superseded change without a successor", page((p) => { p.rows = [{ ...copy(trustWire.closed.rows.find((r) => r.state === "superseded")), superseded_by: "" }]; p.count.value = "1"; })],
  ];
  for (const [name, body] of bad) assert.throws(() => trustChangePageOf(body), ReadParseError, name);
  const closed = { ...copy(trustWire.held.rows[0]), state: "cancelled", why_not: { approve: "not_validated", activate: "needs_approval" } };
  assert.doesNotThrow(() => listedTrustChangeOf(closed));
  assert.throws(() => listedTrustChangeOf({ ...closed, can: { revise: false, validate: true, reject: false, cancel: false } }), ReadParseError, "an ended change that may be validated");
  assert.throws(() => listedTrustChangeOf({ ...closed, decide_in_detail: true }), ReadParseError, "an ended change left to the detail");
  assert.doesNotThrow(() => trustChangeDetailOf(copy(trustWire.detail)));
  for (const [name, change] of [["an unknown field", { extra: 1 }], ["no revision", { revision: "" }], ["an unknown state", { change: { ...copy(trustWire.detail.change), state: "pending" } }],
    ["no identifier", { change: { ...copy(trustWire.detail.change), id: "" } }], ["a list for a change", { change: [] }]] as [string, Record<string, unknown>][])
    assert.throws(() => trustChangeDetailOf({ ...copy(trustWire.detail), ...change }), ReadParseError, name);
});

test("M02 changes: a query the service would refuse is not sent", async () => {
  let calls = 0;
  const c = client(trustWire.list, () => { calls++; });
  const queries: unknown[] = [{ limit: 0 }, { limit: 101 }, { limit: 1.5 }, { state: "pending" }, { q: "   " }, { q: "x".repeat(65) }, { after: "x".repeat(4097) }, { sort: "number" }];
  for (const q of queries) await assert.rejects(c.trustChanges(q as never), invalid, JSON.stringify(q));
  for (const id of ["", "a b", "x".repeat(129), "../policy"]) await assert.rejects(c.trustChange(id), invalid, id);
  assert.equal(calls, 0);
});

test("M02 policy read: the two summary facts are read", async () => {
  const page = await client({ waiting_activation: "trust-0a1b2c3d-4", earlier_run_kept: true }).policy();
  assert.deepEqual([page.waiting_activation, page.earlier_run_kept], ["trust-0a1b2c3d-4", true]);
  const none = await client({}).policy();
  assert.deepEqual([none.waiting_activation, none.earlier_run_kept], ["", false]);
});
