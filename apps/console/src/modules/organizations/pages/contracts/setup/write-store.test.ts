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
