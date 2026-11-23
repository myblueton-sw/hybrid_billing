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
