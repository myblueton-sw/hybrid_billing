import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, adminClient } from "./admin";
import { holdRowOf, ownersOf, personOf, policyRowOf, proposalOf, retentionOf, runRefOf, runRowOf, artifactRowOf, impactNowOf } from "./retention";
import { sample, sampleDigest } from "./retention.sample";
import { fieldReaction, forget, pendingFor, pendingNow, react, setScope, settle } from "../modules/administration/pages/retention/live-model";

type Call = { url: string; init: RequestInit };

function fake(status: number, body: unknown) {
  const calls: Call[] = [];
  const f = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    return new Response(JSON.stringify(body), { status });
  }) as unknown as typeof fetch;
  return { f, calls };
}

const digest = sampleDigest;

test("the retention read parses every section and keeps unknowns visible", () => {
  const p = retentionOf(sample);
  assert.equal(p.revision, "r9");
  assert.equal(p.policies?.length, 2);
  assert.equal(p.policies?.[0].policy, null);
  assert.equal(p.policies?.[1].policy?.period_days, 365);
  assert.equal(p.policies?.[1].pending, 7);
  assert.equal(p.policies?.[1].can_propose, false);
  assert.equal(p.policies?.[1].propose_blocked, "already_proposed");
  const g = p.expiry?.[0];
  assert.equal(g?.counts.eligible, 0);
  assert.equal(g?.counts.deleted, 1);
  assert.equal(g?.artifacts?.[0].decision, "not_connected");
  assert.equal(g?.artifacts?.[1].decision, "");
  assert.equal(g?.artifacts?.[1].due_at, null);
  assert.equal(g?.artifacts?.[1].hold_blocked, "target_too_long");
  const h = p.holds.rows?.[0];
  assert.deepEqual(h?.scope, { artifacts: [], workspace: true, classes: ["temporary"] });
  assert.deepEqual(holdRowOf({ scope: { artifacts: null, workspace: true, classes: null } }).scope, { artifacts: null, workspace: true, classes: null });
  assert.deepEqual(h?.outcomes?.[0].stores, [{ store: "sample-object-store", result: "deleted" }]);
  assert.equal(h?.release, 6);
  assert.equal(p.deletion.rows?.[0].counts.pending, 2);
  assert.equal(p.deletion.rows?.[0].stopper, null);
  assert.equal(p.deletion.rows?.[0].stopper_known, true);
  assert.equal(p.restore.state, "not_connected");
  assert.equal(p.decisions.tombstones.rows?.[0].run, null);
  assert.equal(p.decisions.tombstones.rows?.[0].other_scope, true);
  const p7 = p.proposals.rows?.[0];
  assert.equal(p7?.policy?.impact_at_proposal?.count, 2);
  assert.notEqual(p7?.impact_now, "invalid");
  assert.equal(p.proposals.rows?.[1].release?.hold, "sample-hold-temporary");
});

test("hardened parsers: wrong types, unsafe and negative integers, missing arrays and keys", () => {
  const r = policyRowOf({ class: "source", policy: "x", can_propose: "yes", propose_blocked: "" });
  assert.equal(r.can_propose, false);
  assert.equal(r.propose_blocked, "missing");
  assert.equal(r.policy?.revision, null);
  const big = policyRowOf({ policy: { revision: 2 ** 64, period_days: -1, grace_days: 1.5 }, can_propose: true, propose_blocked: "already_proposed" });
  assert.deepEqual([big.policy?.revision, big.policy?.period_days, big.policy?.grace_days], [null, null, null]);
  assert.equal(big.can_propose, false);
  assert.equal(big.propose_blocked, "already_proposed");
  const a = artifactRowOf({ id: "x", held: "true", damaged: 1, stores: "db", due_at: 5 });
  assert.deepEqual([a.held, a.damaged, a.stores, a.due_at], [null, null, null, ""]);
  const h = holdRowOf({ id: "h", counts: null, counts_scope: "other_scope", outcomes: "x" });
  assert.equal(h.counts, null);
  assert.equal(h.outcomes, null);
  const run = runRowOf({ counts: { pending: 1 }, stopper: "x" });
  assert.equal(run.counts.deleted, null);
  assert.equal(run.counts.pending, 1);
  assert.equal(run.stopper_known, false);
  assert.deepEqual(personOf({ id: "u", login: null }), { id: "u", login: "" });
  assert.equal(personOf(null), null);
  assert.equal(personOf({ login: "x" }), null);
  assert.equal(runRefOf(null), null);
  const page = retentionOf({ revision: 1, state: "configured" });
  assert.equal(page.revision, "");
  assert.equal(page.policies, null);
  assert.equal(page.holds.rows, null);
  assert.equal(page.proposals.rows, null);
  assert.equal(page.decisions.watermark, null);
});

test("summaries take only the member of their kind, and impact_now distinguishes null from invalid", () => {
  const wrong = proposalOf({ number: 3, kind: "policy", summary: { release: { hold: "h", reason: "r" } } });
  assert.equal(wrong.policy, null);
  assert.equal(wrong.release, null);
  assert.equal(proposalOf({ number: 0 }).number, null);
  assert.equal(proposalOf({ number: 2 ** 60 }).number, null);
  assert.equal(impactNowOf(null), null);
  assert.equal(impactNowOf("x"), "invalid");
  assert.equal(impactNowOf(undefined), "invalid");
  const i = impactNowOf({ as_of: "t", count: -1, digest: "d", workspace_count: 1 });
  assert.ok(i && i !== "invalid" && i.count === null);
});

test("whole-page not_configured and unknown states, and --seed none shapes with real zeros", () => {
  assert.equal(retentionOf({ revision: "r1", now: "x", state: "not_configured" }).state, "not_configured");
  assert.equal(retentionOf({ state: "odd" }).state, "odd");
  const none = retentionOf({
    revision: "r1", state: "configured", policies: [], expiry: [{ class: "identity", counts: { eligible: 0, held: 0, referenced: 0, not_due: 0, no_policy: 0, in_run: 0, deleted: 0 }, total: 0, more: false, artifacts: [] }],
    holds: { total: 0, more: false, rows: [], references: { total: 0, more: false, rows: [] } }, deletion: { total: 0, more: false, rows: [] }, restore: { state: "archived" }, proposals: { total: 0, more: false, rows: [] },
  });
  assert.equal(none.expiry?.[0].total, 0);
  assert.deepEqual(none.expiry?.[0].artifacts, []);
  assert.deepEqual(none.holds.rows, []);
  assert.equal(none.restore.state, "archived");
});

test("owner list keeps only valid IDs and maps a missing login to empty", () => {
  assert.deepEqual(ownersOf([{ user_id: "u1", login: "a@example.invalid" }, { user_id: "bad id" }, { user_id: "u2", login: null }, "x"]), [
    { id: "u1", login: "a@example.invalid" },
    { id: "u2", login: "" },
  ]);
  assert.deepEqual(ownersOf(undefined), []);
});

test("the client reads /retention and sends command envelopes verbatim", async () => {
  const read = fake(200, sample);
  const page = await adminClient("csrf", read.f).retention();
  assert.equal(read.calls[0].url, "/api/v1/retention");
  assert.equal(page.policies?.length, 2);
  const cases: [string, Record<string, string>][] = [
    ["retention.approve", { impact_as_of: "2026-11-17T09:00:00.456789Z", impact_count: "2", impact_digest: digest }],
    ["retention.approve", {}],
    ["retention.propose_policy", { period: "7", grace: "0" }],
    ["retention.place_hold", { owner: "u1", reason: "Hold" }],
    ["retention.place_hold", { owner: "u1", reason: "Hold", class: "final" }],
    ["retention.propose_release", { reason: "Done" }],
    ["retention.withdraw", {}],
    ["retention.stop_run", {}],
  ];
  for (const [id, values] of cases) {
    const ok = fake(200, { business_state: "pending", fields: { proposal: "8", target: "retention-proposal-8" } });
    const out = await adminClient("csrf", ok.f).command(id as "retention.approve", "t", "r9", "why", values, { request: "q", key: "k" });
    const body = JSON.parse(String(ok.calls[0].init.body));
    assert.equal(ok.calls[0].url, "/api/v1/commands/" + id);
    assert.deepEqual(body.body, values);
    assert.equal(body.rationale, "why");
    assert.equal(body.expected_revision, "r9");
    assert.deepEqual(out.fields, { proposal: "8", target: "retention-proposal-8" });
  }
  const hold = fake(200, { business_state: "partially_preserved", fields: { hold: "retention-hold-2", not_preserved: "1", outcome_pending: "0", partially_deleted: "0" } });
  const out = await adminClient("csrf", hold.f).command("retention.place_hold", "retention-artifact:x", "r", "w", { owner: "u", reason: "r" }, { request: "q", key: "k" });
  assert.equal(out.state, "partially_preserved");
  assert.equal(out.fields.not_preserved, "1");
});

test("refusals map to failure classes and reactions", async () => {
  const run = async (status: number, body: unknown) => {
    try {
      await adminClient("csrf", fake(status, body).f).command("retention.withdraw", "t", "r", "w", {}, { request: "q", key: "k" });
    } catch (e) {
      return e as AdminError;
    }
    throw new Error("no failure");
  };
  assert.equal((await run(409, { code: "invalid_stage" })).failure, "stage");
  assert.equal((await run(409, { code: "prerequisite_failed" })).failure, "refused");
  for (const f of ["period", "grace", "owner", "reason", "class", "impact_digest"]) {
    const e = await run(400, { code: "invalid_request", fields: [f] });
    assert.equal(e.failure, "invalid");
    assert.deepEqual(e.fields, [f]);
  }
  const partial = await run(500, { code: "internal_error", operation_id: "op-1" });
  assert.equal(partial.failure, "partial");
  assert.equal(partial.operation, "op-1");
  assert.equal((await run(500, { code: "internal_error" })).failure, "unavailable");
  assert.equal(react("stage", []), "hint");
  assert.equal(fieldReaction("retention.approve", ["impact_digest"]), "impact");
});

test("resend identity: identical values reuse the key, edits renew it", () => {
  forget();
  setScope("w", "u", "s");
  let n = 0;
  const fresh = () => ({ request: "q" + ++n, key: "k" + n });
  const a = pendingFor("retention.place_hold", "retention-artifact:x", { owner: "u", reason: "r", rationale: "why" }, "r1", fresh);
  const b = pendingFor("retention.place_hold", "retention-artifact:x", { owner: "u", reason: "r", rationale: "why" }, "r2", fresh);
  assert.equal(a.key, b.key);
  assert.equal(b.expected, "r1");
  const c = pendingFor("retention.place_hold", "retention-artifact:x", { owner: "u", reason: "r2", rationale: "why" }, "r2", fresh);
  assert.notEqual(c.key, a.key);
  settle("retention.place_hold", "retention-artifact:x", "retry");
  assert.ok(pendingNow("retention.place_hold", "retention-artifact:x"));
  settle("retention.place_hold", "retention-artifact:x", "partial");
  assert.equal(pendingNow("retention.place_hold", "retention-artifact:x"), undefined);
  forget();
});
