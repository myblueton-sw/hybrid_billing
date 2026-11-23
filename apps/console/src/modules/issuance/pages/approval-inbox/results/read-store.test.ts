import test from "node:test";
import assert from "node:assert/strict";
import { ResultReadStore } from "./read-store";
import { ResultError, type ReadEnvelope } from "./contract";
const hash = "a".repeat(64);
const prep = (preparation: string) => ({ preparation, preparation_digest: hash, document: "doc", document_digest: hash, connection: "erp", customer: "customer", period: "2026-10", currency: "USD", preparation_actor: "actor" });
const envelope = (fields: Record<string, string>): ReadEnvelope => ({ revision: "r1", at: "2026-11-22T00:00:00Z", fields });
const discovery = () => envelope({ mode: "synthetic", workspace: "workspace", current_eligibility: "unchecked", snapshot: "r1", next: "", row_00: JSON.stringify(prep("a")), row_01: JSON.stringify(prep("b")) });
function context(target: string) {
 const { preparation_actor: _, ...identity } = prep(target);
 return envelope({ ...identity, mode: "synthetic", profile: "synthetic_result_one_to_one_v1", schema: "hb_erp_result_packet_v1", current_eligibility: "unchecked", snapshot: "r1", next: "", permissions: JSON.stringify({ propose: { state: "allowed", code: "" } }), proposal_readiness: JSON.stringify({ state: "ready", code: "" }), row_00: JSON.stringify({ line: "line", expected_product: "product", expected_unit: "hour" }) });
}
const history = (target: string) => envelope({ mode: "synthetic", preparation: target, digest: hash, current_eligibility: "unchecked", snapshot: "r1", next: "" });
function deferred<T>() { let resolve!: (v: T) => void; const promise = new Promise<T>(r => { resolve = r; }); return { promise, resolve }; }

test("late previous-selection reads cannot overwrite the selected preparation", async () => {
 const oldContext = deferred<ReadEnvelope>(), oldHistory = deferred<ReadEnvelope>();
 const store = new ResultReadStore("workspace", async (command, target) => {
  if (command.endsWith("preparations")) return discovery();
  if (target === "a") return command.endsWith("context") ? oldContext.promise : oldHistory.promise;
  return command.endsWith("context") ? context(target) : history(target);
 });
 await store.refresh(); const previous = store.select("a"); await store.select("b");
 const current = store.snapshot(); assert.equal(current.selected?.preparation, "b"); assert.equal(current.context.phase, "loaded");
 oldContext.resolve(context("a")); oldHistory.resolve(history("a")); await previous;
 assert.equal(store.snapshot(), current);
});

test("session disposal prevents in-flight discovery publication or new reads", async () => {
 const pending = deferred<ReadEnvelope>(); let calls = 0;
 const store = new ResultReadStore("workspace", async () => { calls++; return pending.promise; });
 const reading = store.refresh(); store.dispose(); pending.resolve(discovery()); await reading;
 await store.refresh(); await store.select("a");
 assert.equal(store.snapshot().preparations.phase, "idle"); assert.equal(calls, 1);
});

test("failed refresh clears old selection and readiness while context failure leaves history independent", async () => {
 let deny = false;
 const store = new ResultReadStore("workspace", async (command, target) => {
  if (deny || command.endsWith("context")) throw new ResultError("permission_denied");
  return command.endsWith("preparations") ? discovery() : history(target);
 });
 await store.refresh(); await store.select("a");
 assert.equal(store.snapshot().context.phase, "error"); assert.equal(store.snapshot().history.phase, "loaded");
 deny = true; await store.refresh();
 assert.equal(store.snapshot().selected, undefined); assert.equal(store.snapshot().context.phase, "idle"); assert.equal(store.snapshot().preparations.phase, "error");
 await store.select("not-visible"); assert.equal(store.snapshot().selected, undefined);
});

test("mutation completion reselects only freshly read matching preparation and clears old readiness while pending", async () => {
 const pending = deferred<void>();
 const store = new ResultReadStore("workspace", async (command,target) => command.endsWith("preparations") ? discovery() : command.endsWith("context") ? context(target) : history(target));
 await store.refresh(); await store.select("a");
 const following=store.followMutation(prep("a"),"result",pending.promise);
 assert.equal(store.snapshot().selected,undefined); assert.equal(store.snapshot().context.phase,"idle");
 pending.resolve(); await following;
 assert.equal(store.snapshot().selected?.preparation,"a"); assert.equal(store.snapshot().context.phase,"loaded"); assert.equal(store.snapshot().history.phase,"loaded");
});

test("mutation follow-up never steals a newer selection or resurrects disposed session", async () => {
 for (const mode of ["selection","dispose"] as const) {
  const pending=deferred<void>(); let calls=0;
  const store=new ResultReadStore("workspace",async(command,target)=>{calls++;return command.endsWith("preparations")?discovery():command.endsWith("context")?context(target):history(target);});
  await store.refresh(); await store.select("a");
  const following=store.followMutation(prep("a"),"result",pending.promise);
  if(mode==="dispose")store.dispose();else{await store.refresh();await store.select("b");}
  const before=store.snapshot(),count=calls;pending.resolve();await following;
  assert.equal(store.snapshot(),before);assert.equal(calls,count);
 }
});

test("changed preparation digest does not automatically restore old mutation scope", async()=>{
 let changed=false;
 const store=new ResultReadStore("workspace",async(command,target)=>{
  if(command.endsWith("preparations")){const value=discovery();return changed?envelope({...value.fields,row_00:JSON.stringify({...prep("a"),preparation_digest:"b".repeat(64)})}):value;}
  return command.endsWith("context")?context(target):history(target);
 });
 await store.refresh();await store.select("a");changed=true;
 await store.followMutation(prep("a"),"result",Promise.resolve());
 assert.equal(store.snapshot().selected,undefined);assert.equal(store.snapshot().preparations.phase,"loaded");
});

test("late automatic context cannot replace a newer manual result inspection", async()=>{
 const waitingContext=deferred<ReadEnvelope>(),historyReady=deferred<void>();const inspected:string[]=[];
 const rows=["result-a","result-b"].map(result=>JSON.stringify({mode:"synthetic",result,digest:hash,recorded_state:"pending",receipt:"historical_decision",application_state:"not_started",application_receipt:"",proposed_by:"actor",proposed_at:"2026-11-22T00:00:00Z"}));
 const store=new ResultReadStore("workspace",async(command,target)=>{
  if(command.endsWith("preparations"))return discovery();
  if(command.endsWith("context"))return waitingContext.promise;
  if(command.endsWith("history")){queueMicrotask(()=>historyReady.resolve());return envelope({...history(target).fields,row_00:rows[0],row_01:rows[1]});}
  inspected.push(target);throw new ResultError("permission_denied");
 });
 const following=store.followMutation(prep("a"),"result-a",Promise.resolve());
 await historyReady.promise;
 await new Promise<void>(resolve=>setImmediate(resolve));
 assert.equal(store.snapshot().history.phase,"loaded");
 await store.inspect("result-b");const manual=store.snapshot().detail;
 waitingContext.resolve(context("a"));await following;
 assert.equal(store.snapshot().detail,manual);assert.ok(inspected.length>0);assert.ok(inspected.every(result=>result==="result-b"));
});
