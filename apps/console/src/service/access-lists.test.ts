import { test } from "node:test";
import assert from "node:assert/strict";
import { adminClient, AdminError } from "./admin";

const pending = { id: "access-request-5", change: "revoke", subject: "sample-admin", grant: "reader-sample-admin", action: "view", kind: "focus_data", fields: ["rows"], customer: "all", days: 0,
  digest: "668654c4bcbe3cc0bd526cd6f66da3cb5842938430375d9f3e3bb7e546d49461", state: "pending", current: true, proposer: "sample-admin-2", proposed_at: "2026-11-16T09:00:00Z",
  expires_at: "2026-11-23T09:00:00Z", reason: "sample reason", decided_by: "", decided_at: "", decision_reason: "", result: "", eligible_approvers: 2, can_approve: true, can_reject: false,
  can_withdraw: false, may_end: ["qd-00", "qd-01"], subject_login: "admin@example.invalid", proposer_login: "admin2@example.invalid", decided_by_login: "" };
const applied = { ...pending, id: "access-request-3", change: "add", subject: "sample-member", grant: "", days: 10, state: "applied", current: false, proposer: "sample-admin",
  decided_by: "sample-admin-2", decided_at: "2026-11-16T09:00:00Z", result: "grant-6af86287340b8bd7", eligible_approvers: 0, can_approve: false, may_end: [],
  subject_login: "member@example.invalid", proposer_login: "admin@example.invalid", decided_by_login: "admin2@example.invalid" };
const revocation = { id: "revocation-1", origin: "this_process", recorded_at: "2026-11-16T09:00:00Z", target_kind: "grant", target: "grant-8f02cad10c1d560b", target_version: 1, target_present: true,
  replay: "", state: "revocation_complete", history: [{ state: "received", at: "2026-11-16T09:00:00Z" }, { state: "revocation_complete", at: "2026-11-16T09:00:01Z" }], ended_delegations: [],
  deny_generation: 93, job_state: "succeeded", job_reason: "", requested_by: "sample-admin", approved_by: "sample-admin-2", request: "access-request-2", received_at: "2026-11-16T09:00:00Z",
  deny_at: "2026-11-16T09:00:00Z", receipt: { sequence: 2, watermark: "2:edd4ef79", reason: "" }, verified: true,
  cleanup: [{ item: "C1", state: "done", reason: "deny_generation", refs: [] }, { item: "C5", state: "outside_scope", reason: "cannot_recall", refs: [] }], completed_at: "2026-11-16T09:00:00Z",
  target_subject: "sample-member", target_subject_login: "member@example.invalid", target_action: "view", target_resource_kind: "focus_data", target_fields: ["rows"] };
const people = [{ user_id: "sample-admin", login: "admin@example.invalid", revocation_unconfirmed: false }, { user_id: "sample-admin-2", login: "admin2@example.invalid", revocation_unconfirmed: true }];
const pageOf = (rows: unknown[], order: string, query: Record<string, unknown>, change: Record<string, unknown> = {}) => ({ rows, search_input: "", query, next: "",
  reference_at: "2026-11-16T09:00:00Z", observed_at: "2026-11-16T09:00:00Z", revision: "r11", order, count: { value: String(rows.length), certainty: "exact" }, work_units: rows.length, scan_stopped: false, ...change });
const requests = (rows: unknown[], change: Record<string, unknown> = {}, query: Record<string, unknown> = {}) => pageOf(rows, "number_desc", { q: "", state: "", limit: 25, ...query }, change);
const revocations = (rows: unknown[], change: Record<string, unknown> = {}, query: Record<string, unknown> = {}) => pageOf(rows, "recorded_desc", { q: "", state: "", limit: 25, ...query }, change);
const subjects = (rows: unknown[], change: Record<string, unknown> = {}, query: Record<string, unknown> = {}) => pageOf(rows, "user_id_asc", { q: "", id: "", limit: 25, ...query }, change);

function client(body: unknown, inspect?: (path: string, init: RequestInit) => void) {
  return adminClient("csrf", (async (path, init) => { inspect?.(String(path), init!); return new Response(JSON.stringify(body)); }) as typeof fetch);
}
const invalid = (e: unknown) => e instanceof AdminError && e.failure === "invalid";

test("M03 lists: pages as the service answers them are accepted, each as one bounded read", async () => {
  const paths: string[] = [];
  const see = (path: string, init: RequestInit) => { paths.push(path); assert.equal(init.method, "GET"); };
  const a = await client(requests([pending, applied]), see).accessRequests({});
  const b = await client(revocations([revocation]), see).accessRevocations({});
  const c = await client(subjects(people), see).accessSubjects({});
  assert.deepEqual(paths, ["/api/v1/identity/access/requests", "/api/v1/identity/access/revocations", "/api/v1/identity/access/subjects"]);
  assert.deepEqual([b.rows[0].target_subject_login, b.rows[0].target_action, b.rows[0].target_fields], ["member@example.invalid", "view", ["rows"]]);
  assert.deepEqual([a.rows.map((r) => r.id), b.rows[0].target_version, c.rows.map((r) => r.login)], [["access-request-5", "access-request-3"], 1, ["admin@example.invalid", "admin2@example.invalid"]]);
  assert.equal(a.rows[0].proposer_login, "admin2@example.invalid");
  assert.deepEqual(c.rows.map((r) => r.revocation_unconfirmed), [false, true]);
});

test("M03 lists: filters are sent as given and must be echoed and honoured", async () => {
  let sent = "";
  const see = (path: string) => { sent = path; };
  await client(requests([pending], { search_input: " ADMIN " }, { q: "admin", state: "pending", limit: 5 }), see).accessRequests({ q: " ADMIN ", state: "pending", limit: 5 }).then(() => assert.fail("the echo of an untrimmed search must be refused"), (e) => assert.ok(invalid(e)));
  const got = await client(requests([pending], { search_input: "ADMIN" }, { q: "admin", state: "pending", limit: 5 }), see).accessRequests({ q: " ADMIN ", state: "pending", limit: 5 });
  assert.equal(sent, "/api/v1/identity/access/requests?q=ADMIN&state=pending&limit=5");
  assert.equal(got.rows.length, 1);
  await assert.rejects(client(requests([applied], {}, { state: "pending" })).accessRequests({ state: "pending" }), invalid);
  await assert.rejects(client(requests([pending], {}, { state: "ended" })).accessRequests({ state: "ended" }), invalid);
  await assert.rejects(client(requests([pending], {}, { state: "pending" })).accessRequests({}), invalid);
  await assert.rejects(client(requests([applied], { search_input: "zzz" }, { q: "zzz" })).accessRequests({ q: "zzz" }), invalid);
  await assert.rejects(client(revocations([revocation], { search_input: "other" }, { q: "other" })).accessRevocations({ q: "other" }), invalid);
  await client(revocations([revocation], { search_input: "8f02" }, { q: "8f02" }), see).accessRevocations({ q: "8f02" });
  assert.equal(sent, "/api/v1/identity/access/revocations?q=8f02");
  await client(subjects([people[1]], {}, { id: "sample-admin-2" }), see).accessSubjects({ id: "sample-admin-2" });
  assert.equal(sent, "/api/v1/identity/access/subjects?id=sample-admin-2");
  await assert.rejects(client(subjects([people[0]], {}, { id: "sample-admin-2" })).accessSubjects({ id: "sample-admin-2" }), invalid);
  assert.equal((await client(subjects([{ user_id: "user-1", login: "ΟΔΥΣΣΕΥΣ@example.invalid", revocation_unconfirmed: false }], { search_input: "σσευς" }, { q: "σσευς" })).accessSubjects({ q: "σσευς" })).rows.length, 1);
});

test("M03 lists: a request is refused before sending when it could not be a valid query", async () => {
  let calls = 0;
  const c = client(requests([]), () => { calls++; });
  for (const bad of [{ limit: 0 }, { limit: 101 }, { q: "   " }, { q: "x".repeat(65) }, { state: "open" }, { after: "x".repeat(4097) }, { subject: "a" }])
    await assert.rejects(c.accessRequests(bad as never), invalid);
  await assert.rejects(c.accessRevocations({ state: "pending" } as never), invalid);
  for (const bad of [{ id: "a b" }, { id: "sample-admin", q: "a" }, { id: "sample-admin", after: "a.b" }]) await assert.rejects(c.accessSubjects(bad), invalid);
  assert.equal(calls, 0);
});

test("M03 lists: answers that are not what the service can say are refused", async () => {
  const bad: [string, unknown, "r" | "v" | "s"][] = [
    ["an unknown field on a request", requests([{ ...pending, extra: 1 }]), "r"],
    ["an ended request that can be approved", requests([{ ...applied, can_approve: true }]), "r"],
    ["an ended request with approvers", requests([{ ...applied, eligible_approvers: 1 }]), "r"],
    ["an added grant with no subject", requests([{ ...applied, subject: "", subject_login: "" }]), "r"],
    ["a revocation request with no grant", requests([{ ...pending, grant: "" }]), "r"],
    ["a sign-in name for nobody", requests([{ ...pending, decided_by_login: "x@example.invalid" }]), "r"],
    ["an unknown request state", requests([{ ...pending, state: "open" }]), "r"],
    ["the same request twice", requests([pending, pending]), "r"],
    ["a count below the rows", requests([pending, applied], { count: { value: "1", certainty: "exact" } }), "r"],
    ["a cursor with an exact count", requests([pending], { next: "a.b" }), "r"],
    ["a lower bound with no cursor", requests([pending], { count: { value: "1", certainty: "lower_bound" } }), "r"],
    ["more rows than the limit", requests([pending, applied], {}, { limit: 1 }), "r"],
    ["another order", requests([pending], { order: "recorded_desc" }), "r"],
    ["a reference after the observation", requests([pending], { reference_at: "2026-11-16T09:00:01Z" }), "r"],
    ["an unknown field on a revocation", revocations([{ ...revocation, extra: 1 }]), "v"],
    ["a revocation with no history", revocations([{ ...revocation, history: [] }]), "v"],
    ["a received time that is not the first transition", revocations([{ ...revocation, received_at: "2026-11-16T09:00:05Z" }]), "v"],
    ["verified with a failed receipt", revocations([{ ...revocation, receipt: { ...revocation.receipt, reason: "mismatch" } }]), "v"],
    ["the same revocation twice", revocations([revocation, revocation]), "v"],
    ["a verified flag that is not a boolean", revocations([{ ...revocation, verified: "true" }]), "v"],
    ["a present flag that is not a boolean", revocations([{ ...revocation, target_present: 1 }]), "v"],
    ["no receipt", revocations([{ ...revocation, receipt: null }]), "v"],
    ["a receipt sequence that is not a safe integer", revocations([{ ...revocation, receipt: { ...revocation.receipt, sequence: Number.MAX_SAFE_INTEGER + 1 } }]), "v"],
    ["a history time that is not text", revocations([{ ...revocation, history: [{ state: "received", at: 123 }] }]), "v"],
    ["an ended delegation that is not text", revocations([{ ...revocation, ended_delegations: [1] }]), "v"],
    ["a cleanup reference that is not text", revocations([{ ...revocation, cleanup: [{ item: "C9", state: "done", reason: "chain_rules", refs: [false] }] }]), "v"],
    ["an unknown cleanup state", revocations([{ ...revocation, cleanup: [{ item: "C9", state: "future", reason: "", refs: [] }] }]), "v"],
    ["an unknown revocation state", revocations([{ ...revocation, state: "future" }]), "v"],
    ["an unknown job state", revocations([{ ...revocation, job_state: "future" }]), "v"],
    ["cleanup that is not a list", revocations([{ ...revocation, cleanup: {} }]), "v"],
    ["history that is not a list", revocations([{ ...revocation, history: "none" }]), "v"],
    ["a revocation without the target's facts", revocations([(({ target_subject: _omitted, ...rest }) => rest)(revocation)]), "v"],
    ["a sign-in name for a target with no holder", revocations([{ ...revocation, target_subject: "" }]), "v"],
    ["a target action without its resource kind", revocations([{ ...revocation, target_resource_kind: "" }]), "v"],
    ["target fields with no action", revocations([{ ...revocation, target_action: "", target_resource_kind: "" }]), "v"],
    ["a state echo on revocations", revocations([revocation], {}, { state: "pending" }), "v"],
    ["an unknown field on a person", subjects([{ ...people[0], role: "administrator" }]), "s"],
    ["people out of order", subjects([people[1], people[0]]), "s"],
    ["a person without the revocation flag", subjects([{ user_id: "sample-admin", login: "admin@example.invalid" }]), "s"],
    ["a revocation flag that is not a boolean", subjects([{ ...people[0], revocation_unconfirmed: "no" }]), "s"],
    ["a person twice", subjects([people[0], people[0]]), "s"],
    ["a sign-in name beyond its length", subjects([{ user_id: "sample-admin", login: "a".repeat(321), revocation_unconfirmed: false }]), "s"],
  ];
  for (const [name, body, kind] of bad) {
    const c = client(body);
    await assert.rejects(kind === "r" ? c.accessRequests({}) : kind === "v" ? c.accessRevocations({}) : c.accessSubjects({}), invalid, name);
  }
  const later = { ...revocation, id: "revocation-2", recorded_at: "2026-11-17T09:00:00Z", received_at: "2026-11-17T09:00:00Z", history: [{ state: "received", at: "2026-11-17T09:00:00Z" }] };
  assert.equal((await client(revocations([later, revocation])).accessRevocations({})).rows.length, 2);
  assert.equal((await client(revocations([revocation, later])).accessRevocations({})).rows.length, 2);
  const replayed = { ...revocation, id: "revocation-3", origin: "ledger_replay", replay: "absent", target_present: false, state: "decision_unconfirmed", job_state: "waiting", job_reason: "receipt_unverified",
    approved_by: "", request: "", receipt: { sequence: 0, watermark: "", reason: "" }, verified: false, completed_at: "", cleanup: [{ item: "C1", state: "not_started", reason: "", refs: [] }],
    target_subject: "", target_subject_login: "", target_action: "", target_resource_kind: "", target_fields: [] };
  assert.equal((await client(revocations([replayed])).accessRevocations({})).rows[0].state, "decision_unconfirmed");
  const strange = await client(requests([{ ...pending, reason: "line one\nline two\u0001" }, { ...applied, state: "expired", decided_by: "", decided_by_login: "", decided_at: "", result: "" },
    { ...pending, id: "access-request-9", action: "", kind: "", fields: [], customer: "", can_approve: false, current: false }])).accessRequests({});
  assert.deepEqual(strange.rows.map((r) => r.state), ["pending", "expired", "pending"]);
  assert.equal(strange.rows[0].reason, "line one\nline two\u0001");
  const more = await client(requests([pending], { next: "a.b", count: { value: "1", certainty: "lower_bound" }, scan_stopped: true })).accessRequests({});
  assert.equal(more.next, "a.b");
  const second = await client(requests([applied], { count: { value: "7", certainty: "exact" } })).accessRequests({ after: "a.b" });
  assert.equal(second.count.value, "7");
  await assert.rejects(client(requests([applied], { count: { value: "7", certainty: "exact" } })).accessRequests({}), invalid);
});
