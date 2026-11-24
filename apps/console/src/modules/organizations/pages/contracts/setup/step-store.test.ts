import test from "node:test";
import assert from "node:assert/strict";
import { SetupError } from "./context";
import { sessionStepStore, StepWriteStore, type StepIntent } from "./step-store";
const owner={user:"actor",workspace:"workspace",csrf:"csrf"};
type Sent=StepIntent&{args?:Record<string,string>;scope?:Record<string,string>};
const intent=(n:string,change:Partial<Sent>={}):Sent=>({operation:"record_term",actor:"actor",workspace:"workspace",sessionCSRF:"csrf",revision:"r1",attempt:{request:"request-"+n,key:"key-"+n},...change});
const refuse=(store:StepWriteStore<StepIntent,string>,run:()=>Promise<void>,code:string)=>assert.rejects(run,(e:unknown)=>e instanceof SetupError&&e.code===code);
test("a recorded request keeps its receipt and a clean refusal records nothing as unknown",async()=>{
 let fail="";const sent:StepIntent[]=[];
 const store=new StepWriteStore<StepIntent,string>(owner,async i=>{sent.push(i);if(fail)throw new SetupError(fail,"none");return "receipt";});
 await store.submit(intent("a"),"term");assert.deepEqual(store.snapshot().entries.map(e=>[e.phase,e.receipt,e.subject]),[["recorded","receipt","term"]]);
 fail="prerequisite_failed";await store.submit(intent("b"),"term");
 assert.deepEqual(store.snapshot().entries.map(e=>[e.phase,e.error]),[["recorded",undefined],["refused","prerequisite_failed"]]);assert.equal(store.unresolved(),false);assert.equal(sent.length,2);
});
test("an unknown outcome blocks other requests and the unchanged retry reuses the whole request",async()=>{
 let mode="unknown";const sent:StepIntent[]=[];
 const store=new StepWriteStore<StepIntent,string>(owner,async i=>{sent.push(i);if(mode==="unknown")throw new SetupError("service_unavailable","unknown","operation-1");if(mode==="refuse")throw new SetupError("evidence_stale","none");return "receipt";});
 await store.submit(intent("a"),"term");
 assert.deepEqual(store.snapshot().entries.map(e=>[e.phase,e.operationId]),[["unknown","operation-1"]]);assert.equal(store.unresolved(),true);assert.equal(store.dirty(),true);
 await refuse(store,()=>store.submit(intent("b"),"other"),"unresolved_operation");
 mode="refuse";await store.retry("key-a");
 assert.equal(store.snapshot().entries[0].phase,"unknown","a clean refusal of a retry cannot disprove the earlier effect");assert.equal(store.snapshot().entries[0].error,"evidence_stale");assert.deepEqual(sent[1],sent[0]);
 mode="ok";await store.retry("key-a");assert.equal(store.snapshot().entries[0].phase,"recorded");
 await refuse(store,()=>store.retry("key-a"),"retry_unavailable");
});
test("recovery sends the same inputs under a new key and the current revision, and settles the unknown entry only when recorded",async()=>{
 let mode="unknown";const sent:StepIntent[]=[];
 const store=new StepWriteStore<StepIntent,string>(owner,async i=>{sent.push(i);if(mode==="unknown")throw new SetupError("service_unavailable","unknown");if(mode==="refuse")throw new SetupError("prerequisite_failed","none");return "receipt";});
 await store.submit(intent("a"),"term");
 mode="refuse";await store.recover("key-a",{request:"request-b",key:"key-b"},"r2");
 assert.deepEqual(store.snapshot().entries.map(e=>[e.intent.attempt.key,e.phase,e.recovers]),[["key-a","unknown",undefined],["key-b","refused","key-a"]]);
 assert.equal(sent[1].operation,sent[0].operation);assert.equal(sent[1].revision,"r2");assert.notEqual(sent[1].attempt.key,sent[0].attempt.key);
 mode="ok";await store.recover("key-a",{request:"request-c",key:"key-c"},"r3");
 assert.deepEqual(store.snapshot().entries.map(e=>[e.intent.attempt.key,e.phase]),[["key-a","reconciled"],["key-b","refused"],["key-c","recorded"]]);
 assert.equal(store.snapshot().entries[0].receipt,"receipt");assert.equal(store.unresolved(),false);
 await refuse(store,()=>store.recover("key-a",{request:"request-d",key:"key-d"},"r3"),"recovery_unavailable");
});
test("a recovery whose own outcome is unknown is retried unchanged and then settles the original",async()=>{
 let mode="unknown";
 const store=new StepWriteStore<StepIntent,string>(owner,async()=>{if(mode==="unknown")throw new SetupError("service_unavailable","unknown");return "receipt";});
 await store.submit(intent("a"),"term");await store.recover("key-a",{request:"request-b",key:"key-b"},"r2");
 assert.deepEqual(store.snapshot().entries.map(e=>e.phase),["unknown","unknown"]);
 await refuse(store,()=>store.recover("key-a",{request:"request-c",key:"key-c"},"r2"),"recovery_unavailable");
 await refuse(store,()=>store.recover("key-b",{request:"request-c",key:"key-c"},"r2"),"recovery_unavailable");
 mode="ok";await store.retry("key-b");
 assert.deepEqual(store.snapshot().entries.map(e=>[e.intent.attempt.key,e.phase]),[["key-a","reconciled"],["key-b","recorded"]]);assert.equal(store.unresolved(),false);
});
test("the original recorded by its own retry settles a recovery that is still unknown",async()=>{
 let mode="unknown";
 const store=new StepWriteStore<StepIntent,string>(owner,async()=>{if(mode==="unknown")throw new SetupError("service_unavailable","unknown");return "receipt";});
 await store.submit(intent("a"),"term");await store.recover("key-a",{request:"request-b",key:"key-b"},"r2");
 mode="ok";await store.retry("key-a");
 assert.deepEqual(store.snapshot().entries.map(e=>[e.intent.attempt.key,e.phase]),[["key-b","reconciled"],["key-a","recorded"]]);assert.equal(store.unresolved(),false);assert.equal(store.dirty(),false);
});
test("recovery resends the generated target, the arguments and the scope unchanged",async()=>{
 let mode="unknown";const sent:Sent[]=[];
 const store=new StepWriteStore<Sent,string>(owner,async i=>{sent.push(i);if(mode==="unknown")throw new SetupError("service_unavailable","unknown");return "receipt";});
 await store.submit(intent("a",{operation:"bind_term",args:{binding:"binding-generated",term:"term-1",from:"2026-11-01",reason:"initial"},scope:{contract:"contract-a",configuration:"configuration-a",digest:"d"}}),"binding-generated","apply:contract-a/configuration-a");
 mode="ok";await store.recover("key-a",{request:"request-b",key:"key-b"},"r9");
 assert.deepEqual(sent[1].args,sent[0].args);assert.deepEqual(sent[1].scope,sent[0].scope);assert.equal(sent[1].operation,"bind_term");assert.equal(sent[1].actor,"actor");assert.equal(sent[1].revision,"r9");
 assert.equal(store.snapshot().entries.find(e=>e.intent.attempt.key==="key-b")?.subject,"binding-generated");
});
test("an unexpected failure is unknown, and nothing is retried or recovered while a request is in flight",async()=>{
 let release!:()=>void;const gate=new Promise<void>(done=>{release=done;});let mode="throw";
 const store=new StepWriteStore<StepIntent,string>(owner,async()=>{if(mode==="throw")throw new Error("boom");await gate;return "receipt";});
 await store.submit(intent("a"),"term");assert.deepEqual(store.snapshot().entries.map(e=>[e.phase,e.error]),[["unknown","service_unavailable"]]);
 mode="wait";const running=store.retry("key-a");
 await refuse(store,()=>store.retry("key-a"),"retry_unavailable");await refuse(store,()=>store.recover("key-a",{request:"request-b",key:"key-b"},"r2"),"recovery_unavailable");
 release();await running;assert.equal(store.snapshot().entries[0].phase,"recorded");
});
test("the unsent input of a form is cleared when its request is recorded, directly or through recovery, and kept after a refusal",async()=>{
 let mode="refuse";
 const store=new StepWriteStore<StepIntent,string>(owner,async()=>{if(mode==="refuse")throw new SetupError("prerequisite_failed","none");if(mode==="unknown")throw new SetupError("service_unavailable","unknown");return "receipt";});
 store.edit("record",{days:"30"});await store.submit(intent("a"),"term","record");assert.deepEqual(store.draft("record"),{days:"30"});
 mode="unknown";await store.submit(intent("b"),"term","record");assert.deepEqual(store.draft("record"),{days:"30"});
 mode="ok";await store.recover("key-b",{request:"request-c",key:"key-c"},"r2");assert.deepEqual(store.draft("record"),{});assert.equal(store.dirty(),false);
 store.edit("apply",{from:"2026-11-01"});await store.submit(intent("d"),"binding","apply");assert.deepEqual(store.draft("apply"),{});
});
test("another session, a reused attempt and a malformed revision are refused before sending",async()=>{
 let calls=0;const store=new StepWriteStore<StepIntent,string>(owner,async()=>{calls++;return "receipt";});
 await refuse(store,()=>store.submit(intent("a",{actor:"other"}),"term"),"session_changed");
 await refuse(store,()=>store.submit(intent("a",{sessionCSRF:"changed"}),"term"),"session_changed");
 await refuse(store,()=>store.submit(intent("a",{revision:""}),"term"),"invalid_setup_request");
 await store.submit(intent("a"),"term");await refuse(store,()=>store.submit(intent("a"),"term"),"duplicate_attempt");
 await refuse(store,()=>store.submit({...intent("b"),attempt:{request:"request-a",key:"key-z"}},"term"),"duplicate_attempt");assert.equal(calls,1);
});
test("unsent input is kept per form in the session and counts as unsaved until cleared",()=>{
 const store=new StepWriteStore<StepIntent,string>(owner,async()=>"receipt");let calls=0;store.subscribe(()=>{calls++;});
 assert.equal(store.dirty(),false);store.edit("record",{days:"30"});store.edit("record",{reason:"Agreed"});store.edit("apply",{from:""});
 assert.deepEqual(store.draft("record"),{days:"30",reason:"Agreed"});assert.deepEqual(store.draft("missing"),{});assert.equal(store.dirty(),true);
 store.clear("record");assert.equal(store.dirty(),false);assert.equal(calls,4);
 const first=sessionStepStore("terms",owner,()=>async()=>"receipt");
 assert.equal(sessionStepStore("terms",owner,()=>async()=>"other"),first);assert.notEqual(sessionStepStore("protected",owner,()=>async()=>"receipt"),first);assert.notEqual(sessionStepStore("terms",{...owner,csrf:"next"},()=>async()=>"receipt"),first);
});
test("a retry answered by a clean revision conflict settles as not recorded only for a fresh attempt that never carried an operation",async()=>{
 const conflict=()=>new SetupError("revision_conflict","none");
 const build=(first:()=>SetupError,clock:{t:number})=>{let n=0;return new StepWriteStore<StepIntent,string>(owner,async()=>{n++;throw n===1?first():conflict();},()=>clock.t,()=>clock.t);};
 const start=Date.parse("2026-11-24T00:00:00Z");
 let clock={t:start},store=build(()=>new SetupError("service_unavailable","unknown"),clock);
 await store.submit(intent("a"),"term");assert.equal(store.snapshot().entries[0].phase,"unknown");
 clock.t=start+29*60*1000;await store.retry("key-a");
 assert.deepEqual([store.snapshot().entries[0].phase,store.snapshot().entries[0].error,store.snapshot().entries[0].noRecord,store.unresolved()],["refused","revision_conflict",true,false]);
 await refuse(store,()=>store.retry("key-a"),"retry_unavailable");
 clock={t:start};store=build(()=>new SetupError("service_unavailable","unknown"),clock);
 await store.submit(intent("a"),"term");clock.t=start+30*60*1000;await store.retry("key-a");
 assert.deepEqual([store.snapshot().entries[0].phase,store.snapshot().entries[0].noRecord],["unknown",undefined],"an attempt at the bound stays unknown");
 clock={t:start};store=build(()=>new SetupError("service_unavailable","unknown"),clock);
 await store.submit(intent("a"),"term");clock.t=start-1;await store.retry("key-a");
 assert.equal(store.snapshot().entries[0].phase,"unknown","a clock set back keeps the attempt unknown");
 clock={t:start};store=build(()=>new SetupError("service_unavailable","unknown","operation-1"),clock);
 await store.submit(intent("a"),"term");await store.retry("key-a");
 assert.deepEqual([store.snapshot().entries[0].phase,store.snapshot().entries[0].operationId],["unknown","operation-1"],"an attempt that carried an operation stays unknown");
 for(const other of [new SetupError("revision_conflict","unknown"),new SetupError("revision_conflict","none","operation-2"),new SetupError("evidence_stale","none"),new SetupError("permission_denied","none"),new SetupError("idempotency_key_expired","unknown")]){
  let n=0;const s2=new StepWriteStore<StepIntent,string>(owner,async()=>{n++;throw n===1?new SetupError("service_unavailable","unknown"):other;},()=>start,()=>0);
  await s2.submit(intent("a"),"term");await s2.retry("key-a");assert.equal(s2.snapshot().entries[0].phase,"unknown",other.code+" "+other.effect);assert.equal(s2.snapshot().entries[0].noRecord,undefined);
 }
 const firstSend=new StepWriteStore<StepIntent,string>(owner,async()=>{throw conflict();},()=>start,()=>0);
 await firstSend.submit(intent("a"),"term");assert.deepEqual([firstSend.snapshot().entries[0].phase,firstSend.snapshot().entries[0].noRecord],["refused",undefined],"a first send is an ordinary refusal");
});
test("a recovery attempt settled as not recorded makes recovery of the original available again",async()=>{
 const start=Date.parse("2026-11-24T00:00:00Z");let mode="unknown";const sent:StepIntent[]=[];
 const store=new StepWriteStore<StepIntent,string>(owner,async i=>{sent.push(i);if(mode==="unknown")throw new SetupError("service_unavailable","unknown");if(mode==="conflict")throw new SetupError("revision_conflict","none");return "receipt";},()=>start,()=>0);
 await store.submit(intent("a"),"term");
 await store.recover("key-a",{request:"request-b",key:"key-b"},"r2");
 assert.deepEqual(store.snapshot().entries.map(e=>[e.intent.attempt.key,e.phase]),[["key-a","unknown"],["key-b","unknown"]]);
 await refuse(store,()=>store.recover("key-a",{request:"request-c",key:"key-c"},"r3"),"recovery_unavailable");
 mode="conflict";await store.retry("key-b");
 assert.deepEqual(store.snapshot().entries.map(e=>[e.intent.attempt.key,e.phase,e.noRecord]),[["key-a","unknown",undefined],["key-b","refused",true]]);
 mode="ok";await store.recover("key-a",{request:"request-c",key:"key-c"},"r3");
 assert.deepEqual(store.snapshot().entries.map(e=>[e.intent.attempt.key,e.phase]),[["key-a","reconciled"],["key-b","refused"],["key-c","recorded"]]);
});
test("the not-recorded bound counts from the first send on both clocks, and an operation learnt on a later retry keeps the attempt unknown",async()=>{
 const start=Date.parse("2026-11-24T00:00:00Z"),min=60*1000;
 const run=async(answers:SetupError[],times:{wall:number;mono:number}[])=>{
  let n=0;const clock={wall:start,mono:1000};
  const store=new StepWriteStore<StepIntent,string>(owner,async()=>{throw answers[n++];},()=>clock.wall,()=>clock.mono);
  await store.submit(intent("a"),"term");
  for(const at of times){clock.wall=start+at.wall;clock.mono=1000+at.mono;await store.retry("key-a").catch(()=>undefined);}
  return store.snapshot().entries[0];
 };
 const lost=()=>new SetupError("service_unavailable","unknown"),conflict=()=>new SetupError("revision_conflict","none");
 assert.equal((await run([lost(),new SetupError("permission_denied","none"),conflict()],[{wall:20*min,mono:20*min},{wall:31*min,mono:31*min}])).phase,"unknown","a retry does not restart the bound");
 assert.equal((await run([lost(),conflict()],[{wall:5*min,mono:31*min}])).phase,"unknown","elapsed time past the bound keeps it unknown although the wall clock was set back");
 assert.equal((await run([lost(),conflict()],[{wall:31*min,mono:5*min}])).phase,"unknown","wall time past the bound keeps it unknown although the timer stood still");
 assert.equal((await run([lost(),conflict()],[{wall:5*min,mono:5*min}])).noRecord,true);
 const learnt=await run([lost(),new SetupError("internal_error","unknown","operation-9"),conflict()],[{wall:min,mono:min},{wall:2*min,mono:2*min}]);
 assert.deepEqual([learnt.phase,learnt.operationId,learnt.noRecord],["unknown","operation-9",undefined]);
});
test("an unresolved request of an earlier sign-in of the same person stays visible to the store of the new session",async()=>{
 const send=()=>async()=>{throw new SetupError("service_unavailable","unknown");};
 const before={user:"person-e",workspace:"workspace",csrf:"csrf-before"},after={...before,csrf:"csrf-after"};
 const old=sessionStepStore<StepIntent,string>("kind-e",before,send);
 await old.submit(intent("e1",{actor:"person-e",sessionCSRF:"csrf-before"}),"term");
 assert.equal(old.snapshot().entries[0].phase,"unknown");assert.deepEqual(old.earlier(),[]);
 const next=sessionStepStore<StepIntent,string>("kind-e",after,send);
 assert.deepEqual(next.snapshot().entries,[]);assert.deepEqual(next.earlier().map(e=>[e.intent.attempt.key,e.phase]),[["key-e1","unknown"]]);
 assert.deepEqual(sessionStepStore<StepIntent,string>("kind-other",after,send).earlier(),[],"another kind of store does not show it");
 assert.deepEqual(sessionStepStore<StepIntent,string>("kind-e",{...after,user:"someone-else"},send).earlier(),[],"another person does not see it");
 await refuse(next,()=>next.retry("key-e1"),"retry_unavailable");
});
