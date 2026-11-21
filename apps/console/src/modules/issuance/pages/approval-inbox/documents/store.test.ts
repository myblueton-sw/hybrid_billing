import test from "node:test";
import assert from "node:assert/strict";
import { DocumentStore, preparationSettled } from "./store";
import type { DocumentGateway } from "./gateway";
import { DocumentError, type FormContext, type RequestIdentity } from "./contract";
import type { Intent, Receipt } from "./transport";
const hash = "a".repeat(64), scope = { workspace: "workspace", request: "request", digest: hash };
const session = { user: "writer", workspace: "workspace", role: "member", login: "writer", csrf: "synthetic-session" };
const row = { document: "document", digest: hash, recorded_state: "confirmed" as const, proposed_by: "proposer", proposed_at: "2026-11-21T00:00:00Z" };
const prep = { preparation: "preparation", document: "document", document_digest: hash, actor: "writer", recorded_receipt: "" };
function deferred<T>() { let resolve!: (v: T) => void; const promise = new Promise<T>(r => { resolve = r; }); return { promise, resolve }; }
function context(s: RequestIdentity = scope): FormContext { return { ...s, revision: "r7", header: { candidate: "candidate", candidate_digest: hash, profile: "synthetic_individual_net_v1", currency: "USD", total: "1", customer: "customer", period: "2026-10", organization: { WorkspaceID: s.workspace, IssuerLegalEntityID: "issuer", BuyerLegalEntityID: "buyer", PayerID: "payer", Currency: "USD", Path: "external_invoice" }, erp: { connection: "erp", product: "product", tax_code: "tax", scale: 2 } }, permission: { state: "allowed", code: "" }, lines: [{ line: "line", rule: "rule", kind: "rated", amount: "1" }] }; }
function detail(): Awaited<ReturnType<DocumentGateway["document"]>> { return { inspection: { row, revision: "r7" }, advice: { document: "document", digest: hash, revision: "r7", permissions: { propose: { state: "allowed", code: "" }, confirm: { state: "allowed", code: "" }, revoke: { state: "allowed", code: "" }, prepare: { state: "allowed", code: "" } }, readiness: { confirm: { state: "ready", code: "" }, revoke: { state: "ready", code: "" }, prepare: { state: "ready", code: "" } } } } as Awaited<ReturnType<DocumentGateway["document"]>>; }
function observation(): Awaited<ReturnType<DocumentGateway["preparation"]>> { return { row: prep, revision: "r7", observation: "missing", observedDigest: "", submission: {}, total: {}, mappings: [] }; }
const receipt = (i: Intent): Receipt => ({ target: i.target, digest: hash, state: i.action === "prepare" ? "prepared" : "confirmed", documentDigest: i.fields.digest, revision: "r8", replay: false, operation: "operation" });
function setup(patch: Partial<DocumentGateway> = {}) {
 const sent: Intent[] = [];
 const client: DocumentGateway = { context: async s => context(s), history: async (_s, kind) => ({ revision: "r7", rows: kind === "documents" ? [row] : [] }), document: async () => detail(), preparation: async () => observation(), submit: async i => { sent.push(i); return receipt(i); }, ...patch };
 const store = new DocumentStore(session, client); store.select("selection", scope); return { store, sent, client };
}
async function ready(store: DocumentStore) { await store.loadHistory("documents"); await store.loadHistory("preparations"); await store.inspectDocument("document"); }
async function proposal(store: DocumentStore) { await store.loadContext(); store.edit({ selectedERP: true, unit: "unit", evidence: "evidence", net: "1", tax: "0", rationale: "Explicit facts", lines: { line: { quantity: "1", tax: "0", evidence: "evidence" } } }); store.reviewProposal(); }
test("late context and history cannot replace a changed selection", async () => {
 const old = deferred<FormContext>(), history = deferred<Awaited<ReturnType<DocumentGateway["history"]>>>();
 const { store } = setup({ context: async s => s.request === "request" ? old.promise : context(s), history: async () => history.promise });
 const first = store.loadContext(), stale = store.loadHistory("documents"); store.select("other", { ...scope, request: "other" }); await store.loadContext();
 old.resolve(context()); history.resolve({ revision: "r7", rows: [row] }); await Promise.all([first, stale]);
 assert.equal(store.snapshot().context.value?.request, "other"); assert.equal(store.snapshot().documents.phase, "idle");
});
test("review freezes explicit packet and editing discards only unsent intent", async () => {
 const { store, sent } = setup(); await proposal(store); const review = store.snapshot().review!;
 assert.ok(Object.isFrozen(review.fields)); assert.match(review.fields.packet, /"Quantity":"1"/);
 store.edit({ ...store.snapshot().draft!, net: "999" }); assert.equal(store.snapshot().review, review);
 store.cancelReview(); assert.equal(sent.length, 0); assert.equal(store.snapshot().draft?.net, "1");
});
test("one pending submit survives navigation and ignores a duplicate click", async () => {
 const result = deferred<Receipt>(); let calls = 0;
 const { store } = setup({ submit: async () => { calls++; return result.promise; } }); await proposal(store);
 const review = store.snapshot().review!, run = store.submit(); await store.submit();
 store.select("other", { ...scope, request: "other" }); await store.loadContext();
 assert.equal(store.snapshot().entries[0].phase, "pending"); result.resolve(receipt(review)); await run;
 assert.equal(calls, 1); assert.equal(store.snapshot().entries[0].phase, "recorded"); assert.equal(store.snapshot().context.value?.request, "other");
});
test("same-request reads started during submit are invalidated on completion", async () => {
 const result = deferred<Receipt>(), late = deferred<FormContext>();
 const { store, client } = setup({ submit: async () => result.promise }); await proposal(store); const intent = store.snapshot().review!;
 const pending = store.submit(); client.context = async () => late.promise; const reading = store.loadContext();
 result.resolve(receipt(intent)); await pending; late.resolve(context()); await reading;
 assert.equal(store.snapshot().context.phase, "idle"); assert.equal(store.snapshot().entries[0].phase, "recorded");
});
test("recorded receipt survives refresh failure", async () => {
 const { store, client } = setup(); await proposal(store); await store.submit(); client.history = async () => { throw new DocumentError("permission_denied"); };
 await store.loadHistory("documents"); assert.equal(store.snapshot().documents.phase, "failed"); assert.equal(store.snapshot().entries[0].phase, "recorded");
});
test("unknown retry preserves exact frozen intent and original operation after later refusal", async () => {
 const calls: Intent[] = []; const { store } = setup({ submit: async i => { calls.push(i); throw calls.length === 1 ? new DocumentError("service_unavailable", "unknown", "original-operation") : new DocumentError("permission_denied"); } });
 await proposal(store); await store.submit(); const key = store.snapshot().entries[0].intent.attempt.key; store.select("elsewhere"); await store.retry(key);
 const entry = store.snapshot().entries[0]; assert.equal(entry.phase, "unknown"); assert.equal(entry.operation, "original-operation"); assert.equal(calls[0], calls[1]); assert.equal(entry.error, "permission_denied");
});
test("clean first refusal permits new reviewed work without erasing evidence", async () => {
 const { store, client } = setup({ submit: async () => { throw new DocumentError("reauthentication_required"); } }); await proposal(store); await store.submit();
 assert.equal(store.snapshot().entries[0].phase, "refused"); client.submit = async i => receipt(i); await proposal(store); await store.submit(); assert.equal(store.snapshot().entries.length, 2); assert.equal(store.snapshot().entries[1].phase, "recorded");
});
for (const reason of ["denied", "blocked", "stale-history", "unresolved-history"]) test(`prepare refuses ${reason}`, async () => {
 const { store, sent } = setup({ document: async () => { const d = detail(); if (reason === "denied") return { ...d, advice: { ...d.advice, permissions: { ...d.advice.permissions, prepare: { state: "denied", code: "permission_denied" } } } }; if (reason === "blocked") return { ...d, advice: { ...d.advice, readiness: { ...d.advice.readiness, prepare: { state: "blocked", code: "prerequisite_failed" } } } }; return d; }, history: async (_s, kind) => ({ revision: reason === "stale-history" ? "r6" : "r7", rows: kind === "documents" ? [row] : reason === "unresolved-history" ? [prep] : [] }) });
 await ready(store); store.reviewAction("prepare", "Prepare"); await store.submit(); assert.equal(sent.length, 0); assert.ok(store.snapshot().error);
});
for (const reason of ["actor", "document", "digest", "revision", "receipt", "conflict"]) test(`fresh-key recovery refuses ${reason}`, async () => {
 const { store, sent } = setup({ history: async (_s, kind) => ({ revision: "r7", rows: kind === "documents" ? [row] : [prep] }), preparation: async () => { const o = observation(); return { ...o, revision: reason === "revision" ? "r6" : "r7", observation: reason === "conflict" ? "conflict" : "missing", row: { ...o.row, actor: reason === "actor" ? "foreign" : "writer", document: reason === "document" ? "foreign" : "document", document_digest: reason === "digest" ? "b".repeat(64) : hash, recorded_receipt: reason === "receipt" ? hash : "" } }; } });
 await ready(store); await store.inspectPreparation("preparation"); store.reviewAction("prepare", "Reconcile owned key", true); await store.submit(); assert.equal(sent.length, 0); assert.ok(store.snapshot().error);
});
test("fresh HTTP recovery preserves owned preparation key and original unknown evidence", async () => {
 const calls: Intent[] = []; let unresolved = false;
 const { store } = setup({ history: async (_s, kind) => ({ revision: "r7", rows: kind === "documents" ? [row] : unresolved ? [{ ...prep, preparation: calls[0].target }] : [] }), preparation: async () => ({ ...observation(), row: { ...prep, preparation: calls[0].target } }), submit: async i => { calls.push(i); if (calls.length === 1) { unresolved = true; throw new DocumentError("service_unavailable", "unknown"); } return receipt(i); } });
 await ready(store); store.reviewAction("prepare", "Prepare exact owned document"); await store.submit();
 await ready(store); await store.inspectPreparation(calls[0].target); store.reviewAction("prepare", "Inspect and recover same owned key", true); await store.submit();
 assert.equal(calls.length, 2); assert.equal(calls[0].target, calls[1].target); assert.deepEqual(calls[0].fields, calls[1].fields); assert.notEqual(calls[0].attempt.key, calls[1].attempt.key);
 assert.equal(store.snapshot().entries[0].phase, "unknown"); assert.equal(store.snapshot().entries[1].phase, "recorded"); assert.equal(preparationSettled(store.snapshot().entries,store.snapshot().entries[0]),true); await store.retry(calls[0].attempt.key); assert.equal(calls.length, 2);
 await ready(store); store.reviewAction("revoke", "Revoke after recovery"); assert.equal(store.snapshot().review?.action, "revoke");
});

test("unknown proposal blocks a replacement after reload and navigation", async () => {
 let calls = 0; const { store } = setup({ submit: async () => { calls++; throw new DocumentError("service_unavailable", "unknown"); } });
 await proposal(store); await store.submit(); store.select("elsewhere"); store.select("returned", scope); await proposal(store); await store.submit();
 assert.equal(store.snapshot().error, "unresolved_document_attempt"); assert.equal(store.snapshot().review, undefined); assert.equal(calls, 1); assert.equal(store.snapshot().entries[0].phase, "unknown");
});
test("retry waits while a different target on the same request is pending", async () => {
 const pending = deferred<Receipt>(); const sent: Intent[] = [];
 const { store } = setup({ submit: async i => { sent.push(i); if (sent.length === 1) throw new DocumentError("service_unavailable", "unknown"); return pending.promise; } });
 await proposal(store); await store.submit(); const original = store.snapshot().entries[0];
 await ready(store); store.reviewAction("confirm", "Independent confirmation"); const reviewed = store.snapshot().review!; const sending = store.submit();
 await store.retry(original.intent.attempt.key); assert.equal(sent.length, 2);
 pending.resolve(receipt(reviewed)); await sending;
});
test("a stale document inspection cannot enable an action after navigation", async () => {
 const stale = deferred<Awaited<ReturnType<DocumentGateway["document"]>>>(); const { store, sent } = setup({ document: async () => stale.promise });
 await store.loadHistory("documents"); const checking = store.inspectDocument("document"); store.select("other", { ...scope, request: "other" });
 stale.resolve(detail()); await checking; store.reviewAction("confirm", "Confirm"); await store.submit();
 assert.equal(store.snapshot().document.phase, "idle"); assert.equal(sent.length, 0);
});
