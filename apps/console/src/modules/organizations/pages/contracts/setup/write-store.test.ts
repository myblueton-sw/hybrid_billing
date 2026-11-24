import test from "node:test";
import assert from "node:assert/strict";
import { SetupWriteStore } from "./write-store";
import { retainedIntent } from "./intents";
import { SetupError } from "./context";
import { testAction,testDigest,testIntent,testRow,testSession } from "./write-fixture";
const page=()=>({revision:"r4",at:"2026-11-16T00:00:00Z",contract:"contract",snapshot:testDigest,next:"",count:1,rows:[testRow(true)]});
const reads={action:async(request:Parameters<typeof testAction>[0])=>testAction(request),configurations:async()=>page()};
test("synchronous pending admission prevents duplicate submissions",async()=>{
 let release!:()=>void,calls=0;
 const store=new SetupWriteStore(testSession,reads,async()=>{calls++;await new Promise<void>(r=>{release=r;});throw new SetupError("service_unavailable","unknown");});
 await store.inspect(testAction().request);const first=store.submit(testIntent());
 await assert.rejects(store.submit(testIntent()),SetupError);assert.equal(calls,1);release();await first;assert.equal(store.snapshot().entries[0].phase,"unknown");
});
test("retry denial and view invalidation preserve original uncertainty and exact identity",async()=>{
 const sent:unknown[]=[];
 const store=new SetupWriteStore(testSession,reads,async input=>{sent.push(input);throw new SetupError(sent.length===1?"service_unavailable":"permission_denied",sent.length===1?"unknown":"none");});
 await store.inspect(testAction().request);await store.submit(testIntent());store.invalidate();await store.retry("key-1");
 assert.deepEqual(sent[0],sent[1]);assert.equal(store.snapshot().entries[0].phase,"unknown");assert.equal(store.snapshot().entries[0].error,"permission_denied");
});
test("fresh owned recovery reconciles only the matching uncertain operation",async()=>{
 let calls=0;
 const store=new SetupWriteStore(testSession,reads,async()=>{if(++calls===1)throw new SetupError("service_unavailable","unknown");return {configuration:testRow(),revision:"r5",operation:"operation-2",replay:false};});
 await store.inspect(testAction().request);await store.submit(testIntent());
 await store.inspect(testAction().request,{after:"",snapshot:""});
 const inspected=store.snapshot().inspection!;
 const recovery=retainedIntent(testSession,inspected.page!,inspected.page!.rows[0],inspected.action,"Ignored new proposal reason",{request:"request-2",key:"key-2"});
 assert.equal(recovery.rationale,"Synthetic setup");await store.recover(recovery);
 assert.deepEqual(store.snapshot().entries.map(e=>e.phase),["reconciled","recorded"]);
});
test("mixed history/action revisions cannot authorize recovery",async()=>{
 const store=new SetupWriteStore(testSession,{...reads,configurations:async()=>({...page(),revision:"r3"})},async()=>{throw new Error("must not dispatch");});
 await store.inspect(testAction().request,{after:"",snapshot:""});assert.equal(store.snapshot().inspection,undefined);assert.equal(store.snapshot().error,"evidence_stale");
});
test("settled writes invalidate guidance obtained while the write was pending",async()=>{
 let finish!:()=>void;
 const store=new SetupWriteStore(testSession,reads,async()=>{await new Promise<void>(r=>{finish=r;});throw new SetupError("service_unavailable","unknown");});
 await store.inspect(testAction().request);const submission=store.submit(testIntent());
 await store.inspect(testAction().request);assert.ok(store.snapshot().inspection);
 finish();await submission;assert.equal(store.snapshot().inspection,undefined);
});
test("a retry answered by a clean revision conflict settles as not recorded and unblocks the session only inside the bound on both clocks",async()=>{
 const start=Date.parse("2026-11-24T00:00:00Z"),min=60*1000;
 const run=async(answers:SetupError[],times:{wall:number;mono:number}[])=>{
  let n=0;const clock={wall:start,mono:500};
  const store=new SetupWriteStore(testSession,reads,async()=>{throw answers[n++];},()=>clock.wall,()=>clock.mono);
  await store.inspect(testAction().request);await store.submit(testIntent());
  for(const at of times){clock.wall=start+at.wall;clock.mono=500+at.mono;await store.retry("key-1").catch(()=>undefined);}
  return store;
 };
 const lost=()=>new SetupError("service_unavailable","unknown"),conflict=()=>new SetupError("revision_conflict","none");
 const settled=await run([lost(),conflict()],[{wall:5*min,mono:5*min}]),entry=settled.snapshot().entries[0];
 assert.deepEqual([entry.phase,entry.error,entry.noRecord,entry.at],["refused","revision_conflict",true,"2026-11-24T00:00:00.000Z"]);
 await assert.rejects(settled.retry("key-1"),SetupError);
 const other={...testIntent(),attempt:{request:"request-2",key:"key-2"}};
 await settled.inspect(testAction().request);await settled.submit(other);
 assert.deepEqual(settled.snapshot().entries.map(e=>e.intent.attempt.key),["key-1","key-2"],"another request is admitted once the unknown entry is settled");
 const blocked=await run([lost(),new SetupError("evidence_stale","none")],[{wall:min,mono:min}]);
 await blocked.inspect(testAction().request);
 await assert.rejects(blocked.submit(other),(e:unknown)=>e instanceof SetupError&&e.code==="unresolved_operation","an unsettled entry still blocks another request");
 for(const [name,answers,times] of [
  ["at the bound",[lost(),conflict()],[{wall:30*min,mono:30*min}]],
  ["wall clock past the bound",[lost(),conflict()],[{wall:31*min,mono:5*min}]],
  ["timer past the bound",[lost(),conflict()],[{wall:5*min,mono:31*min}]],
  ["wall clock set back",[lost(),conflict()],[{wall:-1,mono:5*min}]],
  ["bound not restarted by a retry",[lost(),new SetupError("permission_denied","none"),conflict()],[{wall:20*min,mono:20*min},{wall:31*min,mono:31*min}]],
  ["operation carried from the first answer",[new SetupError("service_unavailable","unknown","operation-1"),conflict()],[{wall:min,mono:min}]],
  ["operation learnt on a later retry",[lost(),new SetupError("internal_error","unknown","operation-2"),conflict()],[{wall:min,mono:min},{wall:2*min,mono:2*min}]],
  ["conflict with unknown effect",[lost(),new SetupError("revision_conflict","unknown")],[{wall:min,mono:min}]],
  ["conflict carrying an operation",[lost(),new SetupError("revision_conflict","none","operation-3")],[{wall:min,mono:min}]],
  ["another clean refusal",[lost(),new SetupError("evidence_stale","none")],[{wall:min,mono:min}]],
  ["key expired",[lost(),new SetupError("idempotency_key_expired","unknown")],[{wall:min,mono:min}]],
 ] as const){
  const e=(await run([...answers],[...times])).snapshot().entries[0];
  assert.deepEqual([e.phase,e.noRecord],["unknown",undefined],name);
 }
 const first=new SetupWriteStore(testSession,reads,async()=>{throw conflict();},()=>start,()=>0);
 await first.inspect(testAction().request);await first.submit(testIntent());
 assert.deepEqual([first.snapshot().entries[0].phase,first.snapshot().entries[0].noRecord],["refused",undefined],"a first send is an ordinary refusal");
});
