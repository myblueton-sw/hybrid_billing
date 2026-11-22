import test from "node:test";
import assert from "node:assert/strict";
import source from "./fixtures.json";
import type { ApprovalRecord } from "./contract";
import { decisionClient, DecisionError, preparationOf, submissionOf, type Receipt, type Preparation } from "./decision-api";
import { DecisionStore, sessionDecisionStore } from "./decision-store";

const session = { user: "reviewer", workspace: "workspace", role: "member", login: "reviewer", csrf: "synthetic-session" };
function record(): ApprovalRecord {
  const r = structuredClone(source.records[0]) as ApprovalRecord;
  r.id = "request-one"; r.kind = "billing_result"; r.status = "pending";
  r.author = { id: "author", label: "Author" }; r.requester = { id: "requester", label: "Requester" }; r.policy_revision = 2;
  r.evidence.revision = "a".repeat(64); r.evidence.blockers = [{ code: "current_checks_unverified", text: "unchecked" }];
  r.amounts = [{ currency: "USD", basis: "rated_candidate_total", candidate: "15", base: null, difference: null, erp: null }];
  return r;
}
function fields() { return { mode: "synthetic", receipt: "current_decision_preparation", request: "request-one", digest: "a".repeat(64), recorded_state: "pending", author: "author", requester: "requester", amount: "15", currency: "USD", amount_basis: "rated_candidate_total", policy_revision: "2", checked_at: "2026-11-16T09:00:00Z", revision: "r42", approve: "eligible", reject: "eligible" }; }
const prep = () => preparationOf(fields(), record());
const submission = () => submissionOf(prep(), "approve", "Checked synthetic evidence", "", { request: "attempt-one", key: "key-one" });
const receipt: Receipt = { request: "attempt-one", target: "request-one", digest: "a".repeat(64), state: "approved", revision: "r43", replay: false };
const json = (v: unknown, status = 200) => new Response(JSON.stringify(v), { status });
function wire(tried: string) { return { request_id: tried, status: "completed", target_id: "request-one", result_revision: "r42", replay: false, as_of: "2026-11-16T09:00:00Z", fields: fields() }; }
function success() { return { ...wire("attempt-one"), business_state: "approved", result_revision: "r43", operation_id: "op-one", fields: { mode: "synthetic", request: "request-one", digest: "a".repeat(64), recorded_state: "approved", receipt: "historical_decision" } }; }
function server(command: (init: RequestInit) => Response | Promise<Response>): typeof fetch {
  return async (input, init) => String(input).endsWith("/session") ? json({ ...session, csrf_token: session.csrf }) : command(init!);
}
function deferred<T>() { let resolve!: (v: T) => void; let reject!: (e: unknown) => void; const promise = new Promise<T>((a,b) => { resolve=a; reject=b; }); return { promise, resolve, reject }; }

test("preparation matches exact selected facts and rejects malformed or fabricated eligibility", () => {
  assert.equal(prep().amount, "15");
  for (const patch of [{ amount: "15.0" }, { currency: "EUR" }, { author: "other" }, { requester: "author" }, { policy_revision: "3" }, { request: "other" }, { digest: "b".repeat(64) }, { revision: "42" }, { checked_at: "2026-02-30T09:00:00Z" }, { checked_at: "0000-01-01T09:00:00Z" }, { checked_at: "2026-11-16T24:00:00Z" }, { approve: "maybe" }, { recorded_state: "approved" }, { extra: "unreviewed" }, { amount: 15 }]) assert.throws(() => preparationOf({ ...fields(), ...patch }, record()), DecisionError);
  assert.throws(() => submissionOf(prep(), "reject", "reason", ""), DecisionError);
  assert.throws(() => submissionOf(prep(), "approve", " ", ""), DecisionError);
  assert.throws(() => submissionOf(prep(), "approve", "가".repeat(667), ""), DecisionError);
});

test("raw client checks the session and exact command envelopes without coercion", async () => {
  let calls=0;
  const client = decisionClient(session, server(init => { calls++; const b=JSON.parse(String(init.body)); assert.equal(new Headers(init.headers).get("X-CSRF-Token"), session.csrf); return json(wire(b.request_id)); }));
  assert.equal((await client.prepare(record())).revision, "r42");
  assert.equal(calls,1);
  const changed = decisionClient(session, async () => json({ ...session, csrf_token: "different" }));
  await assert.rejects(changed.submit(submission()), (e: DecisionError) => e.code === "session_changed" && e.effect === "none");
  const p = decisionClient(session,server(() => json(success())));
  assert.equal((await p.submit(submission())).state,"approved");
  for (const patch of [{ business_state: "rejected" }, { target_id: "other" }, { fields: { ...success().fields, digest: "b".repeat(64) } }, { replay: "false" }, { result_revision: 43 }]) {
    await assert.rejects(decisionClient(session,server(() => json({ ...success(), ...patch }))).submit(submission()), (e: DecisionError) => e.effect === "unknown");
  }
});

test("response loss and malformed operation references never imply a no-effect refusal", async () => {
  for (const response of [() => { throw new Error("network"); }, () => new Response("not json"), () => json({ code: "permission_denied", request_id: "attempt-one", operation_id: "invalid operation" },403)]) {
    await assert.rejects(decisionClient(session,server(response)).submit(submission()), (e: DecisionError) => e.effect === "unknown");
  }
  await assert.rejects(decisionClient(session,server(() => json({ code: "permission_denied", request_id: "attempt-one" },403))).submit(submission()), (e: DecisionError) => e.effect === "none");
});

test("late preparation cannot cross selected scope and same-session store survives remount", async () => {
  const delayed=deferred<Preparation>();
  const store=new DecisionStore({ prepare: () => delayed.promise, submit: async () => receipt });
  store.select("first",record()); const checking=store.check(); store.select("second",record()); delayed.resolve(prep()); await checking;
  assert.equal(store.snapshot().preparation,undefined);
  const a=sessionDecisionStore(session); a.select("first",record());
  assert.equal(sessionDecisionStore({ ...session }),a);
  assert.notEqual(sessionDecisionStore({ ...session,csrf:"new-session" }),a);
});

test("pending duplicate submissions share one frozen attempt and survive selection changes", async () => {
  const delayed=deferred<Receipt>(); let calls=0;
  const store=new DecisionStore({ prepare: async () => prep(), submit: async () => { calls++; return delayed.promise; } });
  store.select("first",record()); await store.check(); store.choose("approve");store.edit("Checked","");
  const sending=store.submit(); await store.submit();assert.equal(calls,1);
  store.select("other");assert.equal(store.snapshot().entries["request-one"].phase,"pending");
  delayed.resolve(receipt);await sending;assert.equal(store.snapshot().entries["request-one"].phase,"recorded");
  store.select("first",record());await store.check();assert.equal(store.snapshot().preparation,undefined);
});

test("unknown retry preserves payload and uncertainty even after permission refusal", async () => {
  const seen: unknown[]=[];
  const store=new DecisionStore({ prepare: async () => prep(), submit: async s => { seen.push(s);throw new DecisionError(seen.length===1?"service_unavailable":"permission_denied",seen.length===1?"unknown":"none",seen.length===1?"op-one":""); } });
  store.select("scope",record());await store.check();store.choose("reject");store.edit("Incorrect amount","incorrect_amount");await store.submit();
  const original=store.snapshot().entries["request-one"].submission;
  store.select("other");await store.retry("request-one");
  assert.equal(store.snapshot().entries["request-one"].phase,"unknown");assert.equal(store.snapshot().entries["request-one"].operation,"op-one");
  assert.equal(seen[0],seen[1]);assert.equal(original.reason,"incorrect_amount");assert.equal(Object.isFrozen(original),true);assert.equal(Object.isFrozen(original.attempt),true);
  store.select("scope",record());await store.check();assert.equal(store.snapshot().preparation,undefined);
});

test("unmatched error identity cannot attach another operation reference", async () => {
 await assert.rejects(decisionClient(session,server(() => json({ code: "internal_error", request_id: "other-attempt", operation_id: "other-operation" },500))).submit(submission()), (e: DecisionError) => e.effect === "unknown" && e.operation === "");
});

test("selection snapshots cannot be changed through public state", async () => {
 const store=new DecisionStore({ prepare: async () => prep(), submit: async () => receipt });
 store.select("scope",record());
 assert.throws(() => { store.snapshot().record!.amounts[0].candidate="0"; });
 assert.equal(store.snapshot().record!.amounts[0].candidate,"15");
});

test("refresh invalidation clears unchanged-scope preparation but retains unknown attempts", async () => {
 const store=new DecisionStore({ prepare: async () => prep(), submit: async () => { throw new DecisionError("service_unavailable","unknown"); } });
 store.select("scope",record());await store.check();store.invalidatePreparation();assert.equal(store.snapshot().preparation,undefined);
 await store.check();store.choose("approve");store.edit("Checked","");await store.submit();
 store.invalidatePreparation();assert.equal(store.snapshot().entries["request-one"].phase,"unknown");
});

test("a timed-out dispatched command remains unknown and aborts the fetch", async t => {
 t.mock.timers.enable({ apis: ["setTimeout"] });
 let aborted=false;
 const client=decisionClient(session,server(init => new Promise<Response>((_,reject)=>{
  init.signal!.addEventListener("abort",()=>{aborted=true;reject(new Error("aborted"));});
 })));
 const submitted=client.submit(submission());
 const rejected=assert.rejects(submitted,(e: DecisionError)=>e.code==="request_timeout"&&e.effect==="unknown");
 await new Promise(resolve=>setImmediate(resolve));
 t.mock.timers.tick(30000);await rejected;assert.equal(aborted,true);
});

test("session-check timeout never dispatches a decision", async t => {
 t.mock.timers.enable({ apis: ["setTimeout"] });let calls=0;
 const client=decisionClient(session,async (_input,init)=>{calls++;return new Promise<Response>((_,reject)=>{init!.signal!.addEventListener("abort",()=>reject(new Error("aborted")));});});
 const failed=assert.rejects(client.submit(submission()),(e: DecisionError)=>e.effect==="none"&&e.code==="session_unavailable");
 await new Promise(resolve=>setImmediate(resolve));t.mock.timers.tick(30000);await failed;assert.equal(calls,1);
});

test("stalled response body stays unknown after the command deadline", async t => {
 t.mock.timers.enable({ apis: ["setTimeout"] });
 const client=decisionClient(session,server(()=>new Response(new ReadableStream({start(){}}))));
 const failed=assert.rejects(client.submit(submission()),(e: DecisionError)=>e.effect==="unknown"&&e.code==="request_timeout");
 await new Promise(resolve=>setImmediate(resolve));t.mock.timers.tick(30000);await failed;
});

test("valid request IDs cannot collide with object prototype members", async () => {
 for(const id of ["constructor","toString","__proto__"]){
  const selected=record();selected.id=id;
  const store=new DecisionStore({prepare:async()=>Object.freeze({...prep(),request:id}),submit:async()=>receipt});
  store.select("scope",selected);await store.check();assert.equal(store.snapshot().preparation?.request,id);
  store.choose("approve");store.edit("Checked","");await store.submit();assert.equal(store.snapshot().entries[id].phase,"recorded");
 }
});

test("final decisions freeze their family and reject charge preparations and receipts", async () => {
 const r=record();r.kind="final_erp_amount";r.amounts[0].basis="synthetic_reported_net_plus_tax_v1";r.amounts[0].candidate="16";
 const finalFields={...fields(),amount:"16",amount_basis:"synthetic_reported_net_plus_tax_v1"};
 const p=preparationOf(finalFields,r);
 assert.equal(p.family,"erp_final");
 assert.throws(()=>preparationOf(fields(),r),DecisionError);
 assert.throws(()=>preparationOf(finalFields,record()),DecisionError);
 const s=submissionOf(p,"approve","Checked reported gross","",{request:"attempt-one",key:"key-one"});
 assert.equal(s.family,"erp_final");assert.equal(s.amount_basis,"synthetic_reported_net_plus_tax_v1");assert.equal(Object.isFrozen(s),true);
 const urls:string[]=[];
 const response={...success(),fields:{...success().fields,stage:"erp_final_approval"}};
 const fetcher:typeof fetch=async(input)=>{
  if(String(input).endsWith("/session"))return json({...session,csrf_token:session.csrf});
  urls.push(String(input));return json(response);
 };
 assert.equal((await decisionClient(session,fetcher).submit(s)).state,"approved");
 assert.deepEqual(urls,["/api/v1/commands/erp_final.approve"]);
 await assert.rejects(decisionClient(session,server(()=>json(success()))).submit(s),(e:DecisionError)=>e.code==="invalid_receipt"&&e.effect==="unknown");
 await assert.rejects(decisionClient(session,server(()=>json(response))).submit(submission()),(e:DecisionError)=>e.code==="invalid_receipt"&&e.effect==="unknown");
 await assert.rejects(decisionClient(session,fetcher).submit({...s,family:"financial_charge"}),(e:DecisionError)=>e.code==="invalid_decision"&&e.effect==="none");
});

test("unknown final retry cannot switch family after selecting a charge row",async()=>{
 const r=record();r.kind="final_erp_amount";r.amounts[0].basis="synthetic_reported_net_plus_tax_v1";r.amounts[0].candidate="16";
 const p=preparationOf({...fields(),amount:"16",amount_basis:"synthetic_reported_net_plus_tax_v1"},r);
 const families:string[]=[];
 const store=new DecisionStore({prepare:async()=>p,submit:async s=>{families.push(s.family);throw new DecisionError("service_unavailable","unknown");}});
 store.select("final",r);await store.check();store.choose("approve");store.edit("Checked gross","");await store.submit();
 store.select("charge",record());await store.retry("request-one");
 assert.deepEqual(families,["erp_final","erp_final"]);
 assert.equal(store.snapshot().entries["request-one"].submission.amount,"16");
});
