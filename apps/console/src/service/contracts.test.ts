import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, adminClient } from "./admin";
import { actionKeys, contractDetailOf, contractsPageOf } from "./contracts";

function fake(status: number, body: unknown) {
  const calls: string[] = [];
  const f = (async (url: string) => {
    calls.push(url);
    return new Response(JSON.stringify(body), { status });
  }) as unknown as typeof fetch;
  return { f, calls };
}

test("contracts list parses rows and keeps missing numbers as null", async () => {
  const r = fake(200, {
    revision: "r9",
    on: "2026-11-16",
    today: "2026-11-16",
    impact: { money: "not_computed", proration: "" },
    contracts: [{ id: "c1", state: "active", lifecycle_revision: 2, sequence: 3, open_gaps: 0, start: "2026-01-01" }, null, { id: "c2", open_gaps: "1", lifecycle_revision: null }, [1]],
  });
  const got = await adminClient("t", r.f).contracts("2026-11-16");
  assert.equal(r.calls[0], "/api/v1/contracts?on=2026-11-16");
  assert.equal(got.revision, "r9");
  assert.deepEqual(got.impact, { money: "not_computed", proration: "unknown", publication: "unknown", erp: "unknown", final_amount_decision: "unknown" });
  assert.equal(got.contracts.length, 2);
  assert.deepEqual([got.contracts[0].lifecycle_revision, got.contracts[0].sequence, got.contracts[0].open_gaps], [2, 3, 0]);
  assert.deepEqual([got.contracts[1].open_gaps, got.contracts[1].lifecycle_revision, got.contracts[1].sequence, got.contracts[1].end], [null, null, null, ""]);
  const bare = await adminClient("t", fake(200, {}).f).contracts();
  assert.deepEqual(bare.contracts, []);
  assert.equal(contractsPageOf({ contracts: "x" }).contracts.length, 0);
});

test("contract detail parses every part and fails hints closed", async () => {
  const body = {
    revision: "r4",
    today: "2026-11-16",
    from: "2026-11-01",
    until: "2026-12-01",
    contract: { id: "c1", name: "C1" },
    terms: [{ revision: 1, customer: "le", seller: "", path: "direct", components: [{ id: "x", kind: "provider_charge", seller_billable: false }, { id: "y" }, null] }],
    events: [{ id: "e1", kind: "start", from: "2026-01-01", revision: 1, withdraw_blocked: "not_withdrawable", cancel_required: false }, { id: "e2", kind: "end" }],
    segments: [{ from: "2026-11-01", until: "2026-12-01", days: 30, state: "active", revision: 0 }, { from: "a" }],
    reopenings: 1,
    evidence: [{ sequence: 2, write: "cw-1", kind: "missing", resolves: ["a", 3], resolved: "yes" }],
    records: [{ sequence: 1, at: "2026-11-16T09:00:00Z", added: ["e1"], withdrawn: null, actor: "seed" }],
    actions: {
      "contract.suspend": { allowed: true, blocked: "", earliest: "2026-11-16" },
      "contract.resume": { allowed: false, blocked: "not_suspended" },
      "contract.renew": { allowed: false, blocked: "" },
      "contract.end": { allowed: true, blocked: "ended", earliest: "2026-11-16" },
    },
  };
  const r = fake(200, body);
  const got = await adminClient("t", r.f).contract("c 1", "2026-11-01", "2026-12-01");
  assert.equal(r.calls[0], "/api/v1/contracts/c%201?from=2026-11-01&until=2026-12-01");
  assert.notEqual(got, "missing");
  if (got === "missing") return;
  assert.equal(got.terms[0].components.length, 2);
  assert.deepEqual(got.terms[0].components[1], { id: "y", kind: "", provider_payment_responsible: "", seller_billable: null });
  assert.equal(got.terms[0].components[0].seller_billable, false);
  assert.deepEqual(got.events[1], { id: "e2", kind: "end", from: "", revision: null, withdraw_blocked: "missing", cancel_required: false });
  assert.deepEqual([got.segments[0].days, got.segments[0].revision, got.segments[1].days], [30, 0, null]);
  assert.deepEqual([got.evidence[0].resolves, got.evidence[0].resolved], [["a"], false]);
  assert.deepEqual([got.records[0].withdrawn, got.records[0].actor_login, got.reopenings, got.earlier_records], [[], "", 1, null]);
  assert.deepEqual(got.actions["contract.suspend"], { allowed: true, blocked: "", earliest: "2026-11-16" });
  assert.deepEqual(got.actions["contract.resume"], { allowed: false, blocked: "not_suspended", earliest: "" });
  assert.equal(got.actions["contract.renew"].blocked, "unknown");
  assert.deepEqual(got.actions["contract.end"], { allowed: false, blocked: "ended", earliest: "" });
  assert.deepEqual(got.actions["contract.reopen"], { allowed: false, blocked: "missing", earliest: "" });
  assert.equal(Object.keys(got.actions).length, actionKeys.length);
  const empty = contractDetailOf({});
  assert.deepEqual([empty.terms, empty.events, empty.contract.id, empty.reopenings], [[], [], "", null]);
  const q = fake(200, body);
  await adminClient("t", q.f).contract("c1");
  assert.equal(q.calls[0], "/api/v1/contracts/c1");
});

test("an unknown contract read is missing; other failures still throw", async () => {
  assert.equal(await adminClient("t", fake(404, { code: "unsupported_command" }).f).contract("nope"), "missing");
  await assert.rejects(adminClient("t", fake(404, {}).f).contract("x"), (e: AdminError) => e.failure === "unavailable" && e.code === "");
  await assert.rejects(adminClient("t", fake(403, { code: "permission_denied" }).f).contract("x"), (e: AdminError) => e.failure === "denied" && e.code === "permission_denied");
  await assert.rejects(adminClient("t", fake(500, { code: "internal_error" }).f).contracts(), (e: AdminError) => e.failure === "unavailable" && e.code === "internal_error");
});

test("contract commands post the envelope and a resend reuses the attempt identity", async () => {
  const bodies: Record<string, unknown>[] = [];
  const urls: string[] = [];
  const f = (async (url: string, init: RequestInit) => {
    urls.push(url);
    bodies.push(JSON.parse(String(init.body)));
    return new Response(JSON.stringify({ code: "dependency_unavailable" }), { status: 503 });
  }) as unknown as typeof fetch;
  const api = adminClient("t", f);
  const tried = { request: "req-a", key: "key-a" };
  await assert.rejects(api.command("contract.suspend", "sample-contract-01", "r7", "why", { from: "2026-12-01" }, tried), (e: AdminError) => e.failure === "busy");
  await assert.rejects(api.command("contract.suspend", "sample-contract-01", "r7", "why", { from: "2026-12-01" }, tried), (e: AdminError) => e.failure === "busy");
  assert.equal(urls[0], "/api/v1/commands/contract.suspend");
  assert.deepEqual(bodies[0], { request_id: "req-a", target_id: "sample-contract-01", idempotency_key: "key-a", expected_revision: "r7", rationale: "why", body: { from: "2026-12-01" } });
  assert.deepEqual(bodies[1], bodies[0]);
  const ok = (async () => new Response(JSON.stringify({ business_state: "pending_final", fields: { write: "cw-1", sequence: 4 } }), { status: 200 })) as unknown as typeof fetch;
  const out = await adminClient("t", ok).command("contract.evidence", "c3", "r8", "why", { kind: "secured", reference: "x", covered_until: "2026-10-01" }, tried);
  assert.deepEqual(out, { state: "pending_final", fields: { write: "cw-1", sequence: "4" } });
});
