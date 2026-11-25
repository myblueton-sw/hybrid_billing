import assert from "node:assert/strict";
import test from "node:test";
import { setupReadClient } from "./client";
import { SetupError } from "./context";
import { paymentTermsOf, termBindingOf, termWriteClient, termWriteIntent, termWriteRules, type TermWriteIntent, type TermWriteOperation } from "./terms";

const hash = "a".repeat(64), other = "b".repeat(64), at = "2026-11-23T01:00:00.000000001Z", later = "2026-11-24T01:00:00Z";
const scope = { contract: "contract-1", configuration: "config-1", digest: hash };
const session = { user: "writer", workspace: "workspace", csrf: "synthetic-csrf", role: "member", login: "writer" };
const none = { term_state: "none", term_observation: "absent", term: "", term_digest: "", term_revision: "", net_days: "", mode: "", term_author: "", term_approver: "", term_pending_approver: "", term_written_at: "", term_approved_at: "" };
const term = { term_state: "approval_unknown", term_observation: "pending", term: "term-1", term_digest: hash, term_revision: "1", net_days: "45", mode: "calendar", term_author: "first", term_approver: "", term_pending_approver: "second", term_written_at: at, term_approved_at: "" };
const row = (): Record<string, string> => ({ binding: "binding-01", digest: hash, origin: "configuration", configuration: "config-1", term: "term-1", term_revision: "1", net_days: "45", mode: "calendar", zone: "UTC", from: "2026-11-01", to: "2027-10-31", reason: "initial", state: "pending", observation: "pending", author: "first", approver: "", pending_approver: "", written_at: at, approved_at: "" });
const page = (block: Record<string, string>, rows: Record<string, string>[], extra: Record<string, string> = {}) => ({ revision: "r1", at: "2026-11-23T00:00:00Z", fields: { schema: "hb_issuance_setup_terms_v1", contract: scope.contract, configuration: scope.configuration, digest: hash, zone: "UTC", current_eligibility: "unchecked", ...block, snapshot: hash, count: String(rows.length), next: "", ...Object.fromEntries(rows.map((r, i) => [`row_${String(i).padStart(2, "0")}`, JSON.stringify(r)])), ...extra } });
const rejects = (run: () => unknown) => assert.throws(run, SetupError);

test("the term block keeps a pinned approver pending and offers no default", () => {
  assert.equal(paymentTermsOf(page(none, []), scope).term, null);
  const got = paymentTermsOf(page(term, []), scope);
  assert.equal(got.term?.pendingApprover, "second"); assert.equal(got.term?.approver, ""); assert.equal(got.term?.netDays, "45"); assert.equal(got.currentEligibility, "unchecked");
  const approved = { ...term, term_state: "approved", term_observation: "approved", term_approver: "second", term_pending_approver: "", term_approved_at: later };
  assert.equal(paymentTermsOf(page(approved, []), scope).term?.state, "approved");
  assert.equal(paymentTermsOf(page({ ...approved, term_state: "approval_unknown" }, []), scope).term?.state, "approval_unknown");
  const absent = { ...term, term_state: "registration_unknown", term_observation: "absent", term_pending_approver: "", term_written_at: "" };
  assert.equal(paymentTermsOf(page(absent, []), scope).term?.observation, "absent");
  const bad: Record<string, string>[] = [{ term_approver: "second" }, { term_state: "approved" }, { term_pending_approver: "first" }, { net_days: "366" }, { net_days: "030" }, { mode: "business_day" }, { term_revision: "2" },
    { term_state: "pending" }, { term_observation: "approved" }, { term_written_at: "" }];
  for (const change of bad) rejects(() => paymentTermsOf(page({ ...term, ...change }, []), scope));
  rejects(() => paymentTermsOf(page({ ...none, net_days: "30" }, []), scope));
  rejects(() => paymentTermsOf(page(none, [], { current_eligibility: "due" }), scope));
  rejects(() => paymentTermsOf(page(none, [], { due_date: "2026-12-01" }), scope));
  rejects(() => paymentTermsOf(page(none, []), { ...scope, digest: other }));
});

test("binding rows name their origin and never read an uncertain approval as completed", () => {
  assert.equal(termBindingOf(row()).origin, "configuration");
  assert.equal(termBindingOf({ ...row(), origin: "profile", configuration: "", term_revision: "3" }).origin, "profile");
  assert.equal(termBindingOf({ ...row(), to: "" }).to, "");
  assert.equal(termBindingOf({ ...row(), state: "approval_unknown", pending_approver: "second" }).pending_approver, "second");
  assert.equal(termBindingOf({ ...row(), state: "approved", observation: "approved", approver: "second", approved_at: later }).state, "approved");
  for (const change of [{ origin: "profile" }, { origin: "configuration", configuration: "" }, { to: "2026-10-31" }, { from: "2026-02-30" }, { reason: "preference" }, { state: "approved" }, { approver: "second" },
    { pending_approver: "second" }, { state: "approval_unknown" }, { mode: "business_day" }, { zone: "" }, { due_date: "2026-12-01" }]) rejects(() => termBindingOf({ ...row(), ...change }));
});

test("pages bind scope, order, snapshot and exact counts", () => {
  const got = paymentTermsOf(page(term, [row(), { ...row(), binding: "binding-02" }]), scope);
  assert.equal(got.rows.length, 2); assert.equal(got.zone, "UTC"); assert.ok(Object.isFrozen(got.rows));
  rejects(() => paymentTermsOf(page(term, [{ ...row(), binding: "binding-02" }, row()]), scope));
  rejects(() => paymentTermsOf(page(term, [row()], { count: "11" }), scope));
  rejects(() => paymentTermsOf(page(term, [row()]), scope, "binding-00", other));
  rejects(() => paymentTermsOf(page(term, [row()]), scope, "binding-01", hash));
  const full = Array.from({ length: 10 }, (_, i) => ({ ...row(), binding: `binding-${String(i + 1).padStart(2, "0")}` }));
  assert.equal(paymentTermsOf(page(term, full, { count: "12", next: "binding-10" }), scope).next, "binding-10");
  rejects(() => paymentTermsOf(page(term, full, { count: "12" }), scope));
  assert.equal(paymentTermsOf(page(term, [{ ...row(), binding: "binding-11" }], { count: "11" }), scope, "binding-10", hash).count, 11);
});

const args: Record<TermWriteOperation, Record<string, string>> = {
  record_term: { term: "term-1", net_days: "45" }, approve_term: { term: "term-1", term_digest: hash },
  bind_term: { binding: "binding-1", term: "term-1", term_digest: hash, from: "2026-11-01", to: "2027-10-31", reason: "initial" }, approve_term_binding: { binding: "binding-1", binding_digest: hash },
};
const intent = (operation: TermWriteOperation, change: Record<string, string | undefined> = {}, extra: Partial<TermWriteIntent> = {}): TermWriteIntent => {
  const merged: Record<string, string> = {};
  for (const [k, v] of Object.entries({ ...args[operation], ...change })) if (v !== undefined) merged[k] = v;
  return { operation, scope, args: merged, rationale: termWriteRules[operation]?.rationale ? "Agreed payment terms" : "", actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision: "r4", attempt: { request: "request-1", key: "key-1" }, ...extra };
};

test("intents require explicit net days and refuse derived or unknown values", () => {
  for (const operation of Object.keys(args) as TermWriteOperation[]) assert.ok(Object.isFrozen(termWriteIntent(intent(operation)).args));
  assert.equal(termWriteIntent(intent("record_term", { net_days: "0" })).args.net_days, "0");
  assert.equal(termWriteIntent(intent("record_term", { net_days: "365" })).args.net_days, "365");
  assert.equal(termWriteIntent(intent("bind_term", { to: undefined })).args.to, undefined);
  const bad: [TermWriteOperation, Record<string, string | undefined>, Partial<TermWriteIntent>][] = [
    ["record_term", { net_days: undefined }, {}], ["record_term", { net_days: "366" }, {}], ["record_term", { net_days: "-1" }, {}], ["record_term", { net_days: "030" }, {}], ["record_term", { mode: "calendar" }, {}],
    ["record_term", { revision: "2" }, {}], ["record_term", {}, { rationale: "" }], ["approve_term", {}, { rationale: "not taken" }], ["approve_term", { term_digest: "short" }, {}],
    ["bind_term", { zone: "UTC" }, {}], ["bind_term", { contract: "contract-2" }, {}], ["bind_term", { to: "2026-10-31" }, {}], ["bind_term", { reason: "preference" }, {}], ["bind_term", { from: "2026-02-30" }, {}],
    ["bind_term", {}, { rationale: "not taken" }], ["approve_term_binding", { binding_digest: undefined }, {}],
  ];
  for (const [operation, change, extra] of bad) assert.throws(() => termWriteIntent(intent(operation, change, extra)), (e: unknown) => e instanceof SetupError && e.code === "invalid_setup_request");
  assert.throws(() => termWriteIntent(intent("withdraw_term" as TermWriteOperation)), SetupError);
});

const receipts: Record<TermWriteOperation, [string, Record<string, string>]> = {
  record_term: ["pending", { schema: "hb_issuance_term_receipt_v1", term: "term-1", term_digest: hash }], approve_term: ["approved", { schema: "hb_issuance_term_receipt_v1", term: "term-1", term_digest: hash }],
  bind_term: ["pending", { schema: "hb_issuance_term_binding_receipt_v1", binding: "binding-1", binding_digest: hash }], approve_term_binding: ["approved", { schema: "hb_issuance_term_binding_receipt_v1", binding: "binding-1", binding_digest: hash }],
};
function server(change: (raw: Record<string, unknown>) => void = () => {}, status = 200) {
  const requests: { url: string; body: Record<string, unknown>; raw: string }[] = [];
  const fetcher: typeof fetch = async (input, init) => {
    const url = String(input);
    if (url === "/api/v1/session") return Response.json({ user: session.user, workspace: session.workspace, csrf_token: session.csrf });
    const body = JSON.parse(String(init?.body)); requests.push({ url, body, raw: String(init?.body) });
    const [state, fields] = receipts[url.split("issuance_setup.")[1] as TermWriteOperation];
    const raw: Record<string, unknown> = { request_id: body.request_id, status: "completed", operation_id: "operation-1", target_id: body.target_id, result_revision: "r5", business_state: state, replay: false, as_of: "2026-11-23T00:00:00Z",
      fields: { contract: scope.contract, configuration: scope.configuration, current_eligibility: "unchecked", state, ...fields } };
    change(raw); return Response.json(raw, { status });
  };
  return { fetcher, requests };
}

test("each command sends its exact envelope and binds its receipt", async () => {
  for (const operation of Object.keys(args) as TermWriteOperation[]) {
    const s = server(), receipt = await termWriteClient(session, s.fetcher)(intent(operation));
    assert.equal(s.requests[0].url, `/api/v1/commands/issuance_setup.${operation}`);
    assert.deepEqual(s.requests[0].body.body, { configuration: "config-1", digest: hash, ...args[operation] });
    assert.equal(Object.hasOwn(s.requests[0].body, "rationale"), termWriteRules[operation].rationale);
    assert.equal(s.requests[0].raw.includes(session.csrf), false);
    assert.equal(receipt.state, receipts[operation][0]); assert.ok(Object.isFrozen(receipt));
  }
  const repeated = await termWriteClient(session, server(r => { r.business_state = "approved"; r.fields = { ...r.fields as object, state: "approved" }; }).fetcher)(intent("record_term"));
  assert.equal(repeated.state, "approved");
  const uncertain = (e: unknown) => e instanceof SetupError && e.effect === "unknown" && e.code === "invalid_setup_receipt";
  const changes: [TermWriteOperation, (r: Record<string, unknown>) => void][] = [
    ["approve_term", r => { r.fields = { ...r.fields as object, term_digest: other }; }], ["approve_term", r => { r.fields = { ...r.fields as object, term: "term-9" }; }], ["approve_term", r => { r.business_state = "pending"; }],
    ["approve_term_binding", r => { r.fields = { ...r.fields as object, binding_digest: other }; }], ["approve_term_binding", r => { r.target_id = "contract-2"; }],
    ["record_term", r => { r.fields = { ...r.fields as object, state: "approval_unknown" }; r.business_state = "approval_unknown"; }], ["record_term", r => { r.fields = { ...r.fields as object, due_date: "2026-12-01" }; }],
    ["bind_term", r => { r.fields = { ...r.fields as object, schema: "hb_issuance_term_receipt_v1" }; }],
  ];
  for (const [operation, change] of changes) await assert.rejects(termWriteClient(session, server(change).fetcher)(intent(operation)), uncertain);
  for (const operation of [false, true]) {
    const s = server(r => { const request = r.request_id; for (const k of Object.keys(r)) delete r[k]; Object.assign(r, { request_id: request, code: "prerequisite_failed", next_action: "correct_request", retry: false, ...(operation ? { operation_id: "operation-1" } : {}) }); }, 422);
    await assert.rejects(termWriteClient(session, s.fetcher)(intent("bind_term")), (e: unknown) => e instanceof SetupError && e.code === "prerequisite_failed" && e.effect === (operation ? "unknown" : "none"));
  }
  const s = server();
  await assert.rejects(termWriteClient(session, s.fetcher)(intent("record_term", {}, { actor: "other" })), (e: unknown) => e instanceof SetupError && e.code === "session_changed");
  assert.equal(s.requests.length, 0);
});

test("read client requests the scoped terms page", async () => {
  const requests: Record<string, unknown>[] = [];
  const fetcher: typeof fetch = async (input, init) => {
    const url = String(input);
    if (url === "/api/v1/session") return Response.json({ user: session.user, workspace: session.workspace, csrf_token: session.csrf });
    const body = JSON.parse(String(init?.body)); requests.push({ url, ...body });
    return Response.json({ request_id: body.request_id, status: "completed", target_id: body.target_id, result_revision: "r4", replay: false, as_of: "2026-11-23T00:00:00Z", fields: page(term, [row()]).fields });
  };
  const client = setupReadClient(session, fetcher);
  assert.equal((await client.terms(scope)).rows[0].binding, "binding-01");
  assert.equal(String(requests[0].url).endsWith("issuance_setup.terms"), true);
  assert.deepEqual(requests[0].body, { configuration: "config-1", digest: hash });
  await assert.rejects(client.terms(scope, "binding-01"), SetupError);
  assert.equal(requests.length, 1);
});
