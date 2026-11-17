import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, adminClient } from "./admin";
import { counts, openRecoveryOf, recoveryOf, recoveryProposalOf, recoverySummaryOf, isolationRowOf, reconciliationRowOf, resumeRowOf, setRowOf, signed } from "./recovery";
import { fieldReaction, forget, pendingFor, react, refusedNow, setScope, settle } from "../modules/administration/pages/recovery/live-model";
import { k } from "../modules/administration/pages/recovery/live-copy";

type Call = { url: string; init: RequestInit };

function fake(status: number, body: unknown) {
  const calls: Call[] = [];
  const f = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    return new Response(JSON.stringify(body), { status });
  }) as unknown as typeof fetch;
  return { f, calls };
}

const digest = "a".repeat(64);

export const sample = {
  revision: "r7",
  now: "2026-11-17T09:00:00Z",
  state: "configured",
  sets: {
    state: "configured",
    rows: [
      {
        installation: "sample-primary", state: "recovering", generation: 2,
        recovery: {
          state: "open", id: "recovery-1", phase: "verify", as_of: "2026-11-17T08:00:00Z", point_generation: 1, backup_reference: digest, manifest_digest: digest,
          domains: [{ domain: "database", status: "verified", mismatch_chunks: [] }, { domain: "outbox", status: "mismatch", mismatch_chunks: [1] }],
        },
        history: [], history_total: 0, begin: "not_connected", abandon: "not_connected",
      },
      { installation: "sample-standby", state: "locked", generation: 2, recovery: { state: "none", id: "", domains: [] }, history: [], history_total: 0, begin: "not_connected", abandon: "not_connected" },
    ],
  },
  isolation: {
    state: "configured",
    rows: [
      {
        installation: "sample-primary", state: "recovering", generation: 2, stopped_at: "2026-11-17T08:30:00Z", can_lock: false, lock_blocked: "recovering", writers: ["sample-erp-writer"],
        verifier: "sample-verifier", open_calls_source: "recovery", open_calls: [{ id: "call-2", writer: "sample-erp-writer", deadline: "2026-11-17T09:01:00Z", key: "sample-invoice-2", series: "s", number: -2 }],
        open_calls_total: 1, other_scope_present: false, permits: "not_shown", locks: [],
      },
      {
        installation: "sample-standby", state: "locked", generation: 2, stopped_at: "2026-11-17T08:30:00Z", can_lock: false, lock_blocked: "locked", writers: ["sample-standby-writer"], verifier: "",
        open_calls_source: "not_shown", open_calls: [], open_calls_total: 0, other_scope_present: "no", permits: "not_shown",
        locks: [{ by: "sample-admin", by_login: "admin@example.invalid", rationale: "Sample data", at: "2026-11-17T08:30:00Z" }],
      },
    ],
  },
  reconciliation: {
    state: "configured",
    rows: [
      {
        installation: "sample-primary", externals: [{ id: "sample-erp-ledger", kind: "erp", status: "pending" }], unknown_targets: [{ key: "sample-invoice-2", series: "s", number: 2, fenced: true, high_water: 2 }, { key: "k", series: "", number: 1, fenced: "yes", high_water: null }],
        unknown_total: 2, other_scope_present: false, skipped: 0, reconciler: "not_connected",
      },
    ],
  },
  migration: { state: "not_connected", modules: ["migration", "balances", 7] },
  resume: {
    state: "configured",
    rows: [
      {
        installation: "sample-primary", mode: "recovery_resume", phase: "verify", pending_events: [{ sequence: 1, kind: "delete", domain: "evidence", at: "2026-11-17T08:10:00Z" }], pending_total: 2,
        lost: [], lost_digest: digest, can_propose_unlock: false, unlock_blocked: "incomplete",
      },
      { installation: "sample-standby", mode: "lock_release", phase: "not_recovering", pending_events: [], pending_total: 0, lost: [], lost_digest: digest, can_propose_unlock: false, unlock_blocked: "already_proposed" },
    ],
  },
  proposals: [
    {
      number: 10, target: "recovery-proposal-10", kind: "unlock", installation: "sample-standby", state: "pending", created_at: "2026-11-17T08:30:00Z", proposer: "sample-admin-2",
      proposer_login: "admin2@example.invalid", approver: "", approver_login: "", rationale: "Sample data", closed_by: "", closed_by_login: "", closing_rationale: "",
      summary: { installation: "sample-standby", skew_seconds: 0, writer: "", unlock_generation: 2, lost_digest: digest }, can_approve: true, approve_blocked: "", can_withdraw: false, withdraw_blocked: "not_proposer",
    },
    {
      number: 1, target: "recovery-proposal-1", kind: "installation", installation: "sample-primary", state: "applied", summary: { installation: "sample-primary", skew_seconds: 60, unlock_generation: 0 },
      can_approve: false, approve_blocked: "not_pending", can_withdraw: false, withdraw_blocked: "not_pending",
    },
  ],
};

test("the recovery read parses every section and keeps unknowns visible", async () => {
  const r = fake(200, sample);
  const got = await adminClient("t", r.f).recovery();
  assert.equal(r.calls[0].url, "/api/v1/recovery");
  assert.equal(r.calls[0].init.method, "GET");
  assert.equal(got.revision, "r7");
  assert.equal(got.state, "configured");
  const set = got.sets.rows![0];
  assert.equal(set.recovery.id, "recovery-1");
  assert.deepEqual(set.recovery.domains![1].mismatch_chunks, [1]);
  assert.equal(got.sets.rows![1].recovery.state, "none");
  assert.deepEqual(got.sets.rows![1].recovery.domains, []);
  assert.equal(got.sets.rows![1].recovery.point_generation, null);
  assert.equal(got.isolation.rows![0].open_calls![0].number, -2);
  assert.equal(got.isolation.rows![1].other_scope_present, null);
  assert.equal(got.isolation.rows![1].can_lock, false);
  assert.equal(got.isolation.rows![1].lock_blocked, "locked");
  assert.equal(got.reconciliation.rows![0].unknown_targets![0].high_water, 2);
  assert.equal(got.reconciliation.rows![0].unknown_targets![1].high_water, null);
  assert.equal(got.reconciliation.rows![0].unknown_targets![1].fenced, null);
  assert.deepEqual(got.migration.modules, ["migration", "balances"]);
  assert.equal(got.resume.rows![1].unlock_blocked, "already_proposed");
  assert.equal(got.proposals![0].can_approve, true);
  assert.equal(got.proposals![0].summary.unlock_generation, 2);
  assert.equal(got.proposals![0].summary.skew_seconds, null);
  assert.equal(got.proposals![1].summary.skew_seconds, 60);
  assert.equal(got.proposals![1].summary.unlock_generation, null);
});

test("whole-page and section states pass through unchanged", () => {
  const none = recoveryOf({ revision: "r1", now: "2026-11-17T09:00:00Z", state: "not_configured" });
  assert.equal(none.state, "not_configured");
  assert.equal(none.sets.rows, null);
  assert.equal(none.proposals, null);
  const odd = recoveryOf({ revision: "r1", state: "paused", reconciliation: { state: "not_configured", rows: [] }, migration: { state: "not_connected" } });
  assert.equal(odd.state, "paused");
  assert.equal(odd.reconciliation.state, "not_configured");
  assert.deepEqual(odd.reconciliation.rows, []);
  assert.equal(odd.migration.state, "not_connected");
  assert.equal(odd.migration.modules, null);
});

test("hardened parsers: wrong types, unsafe and negative integers, missing arrays", () => {
  assert.equal(signed(-3), -3);
  assert.equal(signed(1.5), null);
  assert.equal(signed(9007199254740993), null);
  assert.deepEqual(counts([1, 2]), [1, 2]);
  assert.equal(counts([1, -2]), null);
  assert.equal(counts([1, "2"]), null);
  assert.equal(counts("1"), null);
  const set = setRowOf({ installation: 3, generation: -1, recovery: { state: "open", point_generation: "1", domains: "x" }, history: null, history_total: 1.2 });
  assert.equal(set.installation, "");
  assert.equal(set.generation, null);
  assert.equal(set.recovery.point_generation, null);
  assert.equal(set.recovery.domains, null);
  assert.equal(set.history, null);
  assert.equal(set.history_total, null);
  assert.equal(openRecoveryOf({ state: "weird" }).domains, null);
  const iso = isolationRowOf({ can_lock: true, lock_blocked: "x", writers: "w", open_calls: null, open_calls_total: -1, locks: {} });
  assert.equal(iso.can_lock, false);
  assert.equal(iso.lock_blocked, "x");
  assert.equal(iso.writers, null);
  assert.equal(iso.open_calls, null);
  assert.equal(iso.open_calls_total, null);
  assert.equal(iso.locks, null);
  assert.equal(isolationRowOf({ can_lock: true, lock_blocked: 3 }).lock_blocked, "missing");
  assert.equal(isolationRowOf({ can_lock: "true", lock_blocked: "" }).lock_blocked, "missing");
  assert.equal(isolationRowOf({}).can_lock, false);
  const rec = reconciliationRowOf({ externals: 1, unknown_targets: [{ number: "2", high_water: 1.5 }], skipped: -1 });
  assert.equal(rec.externals, null);
  assert.equal(rec.unknown_targets![0].number, null);
  assert.equal(rec.unknown_targets![0].high_water, null);
  assert.equal(rec.skipped, null);
  const res = resumeRowOf({ lost: [1, "x"], pending_total: "2", can_propose_unlock: true, unlock_blocked: "" });
  assert.equal(res.lost, null);
  assert.equal(res.pending_total, null);
  assert.equal(res.can_propose_unlock, true);
  assert.equal(resumeRowOf({ can_propose_unlock: true }).can_propose_unlock, false);
  const p = recoveryProposalOf({ number: 0, can_approve: true, approve_blocked: "own_proposal", can_withdraw: null, withdraw_blocked: "" });
  assert.equal(p.number, null);
  assert.equal(p.can_approve, false);
  assert.equal(p.approve_blocked, "own_proposal");
  assert.equal(p.can_withdraw, false);
  assert.equal(p.withdraw_blocked, "missing");
  const s = recoverySummaryOf({ installation: "i", writer: "w", as_of: "x", unlock_generation: 3 }, "writer");
  assert.equal(s.writer, "w");
  assert.equal(s.as_of, "");
  assert.equal(s.unlock_generation, null);
});

test("commands send the envelope with and without a field and return the result fields", async () => {
  const r = fake(200, { business_state: "pending", fields: { proposal: "11", target: "recovery-proposal-11" } });
  const api = adminClient("csrf", r.f);
  const out = await api.command("recovery.propose_unlock", "recovery-installation:sample-standby", "r7", "qa", { lost: digest }, { request: "req-1", key: "key-1" });
  assert.equal(r.calls[0].url, "/api/v1/commands/recovery.propose_unlock");
  const body = JSON.parse(String(r.calls[0].init.body));
  assert.deepEqual(body, { request_id: "req-1", target_id: "recovery-installation:sample-standby", idempotency_key: "key-1", expected_revision: "r7", rationale: "qa", body: { lost: digest } });
  assert.equal((r.calls[0].init.headers as Record<string, string>)["X-Request-Id"], "req-1");
  assert.deepEqual({ state: out.state, fields: out.fields }, { state: "pending", fields: { proposal: "11", target: "recovery-proposal-11" } });
  const l = fake(200, { business_state: "locked", fields: { installation: "sample-standby", target: "recovery-installation:sample-standby" } });
  const locked = await adminClient("csrf", l.f).command("recovery.lock", "recovery-installation:sample-standby", "r8", "qa", {}, { request: "a", key: "b" });
  assert.deepEqual(JSON.parse(String(l.calls[0].init.body)).body, {});
  assert.equal(locked.fields.installation, "sample-standby");
});

test("a resend reuses the request and key only for identical values", () => {
  forget();
  setScope("ws", "u1", "csrf");
  let n = 0;
  const fresh = () => ({ request: "r" + ++n, key: "k" + n });
  const a = pendingFor("recovery.propose_unlock", "recovery-installation:x", { lost: digest, rationale: "why" }, "r1", fresh);
  const b = pendingFor("recovery.propose_unlock", "recovery-installation:x", { lost: digest, rationale: "why" }, "r9", fresh);
  assert.equal(b.key, a.key);
  assert.equal(b.expected, "r1");
  const c = pendingFor("recovery.propose_unlock", "recovery-installation:x", { lost: digest, rationale: "other" }, "r9", fresh);
  assert.notEqual(c.key, a.key);
});

test("stubbed refusals map to the kept sentences and partial needs an operation", async () => {
  forget();
  setScope("ws", "u1", "csrf");
  for (const [code, failure] of [["invalid_stage", "stage"], ["prerequisite_failed", "refused"]] as const) {
    const r = fake(409, { code });
    try {
      await adminClient("c", r.f).command("recovery.approve", "recovery-proposal-10", "r1", "x", {}, { request: "a", key: "b" });
      assert.fail("expected a refusal");
    } catch (e) {
      assert.ok(e instanceof AdminError);
      assert.equal(e.failure, failure);
      assert.equal(react(e.failure, e.fields), "hint");
      settle("recovery.approve", "recovery-proposal-10", react(e.failure, e.fields));
      assert.equal(refusedNow("recovery.approve", "recovery-proposal-10"), true);
      assert.match(k.kept("en", "recovery.approve"), /refused this approval/);
    }
  }
  const partial = fake(500, { code: "internal_error", operation_id: "op-9" });
  try {
    await adminClient("c", partial.f).command("recovery.lock", "recovery-installation:x", "r1", "x", {}, { request: "a", key: "b" });
  } catch (e) {
    assert.ok(e instanceof AdminError);
    assert.equal(e.failure, "partial");
    assert.equal(e.operation, "op-9");
  }
  const bare = fake(500, { code: "internal_error" });
  try {
    await adminClient("c", bare.f).command("recovery.lock", "recovery-installation:x", "r1", "x", {}, { request: "a", key: "b" });
  } catch (e) {
    assert.ok(e instanceof AdminError);
    assert.equal(e.failure, "unavailable");
  }
  const lost = fake(400, { code: "invalid_request", fields: ["lost"] });
  try {
    await adminClient("c", lost.f).command("recovery.propose_unlock", "recovery-installation:x", "r1", "x", { lost: digest }, { request: "a", key: "b" });
  } catch (e) {
    assert.ok(e instanceof AdminError);
    assert.equal(fieldReaction("recovery.propose_unlock", e.fields), "changed");
  }
});
