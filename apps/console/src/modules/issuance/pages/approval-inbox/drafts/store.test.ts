import assert from "node:assert/strict";
import test from "node:test";
import { DraftStore } from "./store";
import { DraftError, type Page, type Summary } from "./contract";
import { draftIntent, type DraftReceipt } from "./mutation";
import type { Inspection } from "./inspection";
import type { Context } from "./context";
const hash="a".repeat(64);
const intent=draftIntent({action:"submit",target:"draft",draft:"draft",draftDigest:"",actor:"actor",workspace:"workspace",revision:"r1",arguments:{configuration:"config",configuration_digest:hash,final_request:"final",final_digest:hash,language:"en"},rationale:"Verified draft",attempt:{request:"request",key:"key"}});
const receipt:DraftReceipt={draft:"draft",digest:hash,submissionState:"submitted",receipt:"submission_record",revision:"r2",operation:"operation",replay:false};
const page:Page<Summary>={revision:"r1",snapshot:hash,next:"",count:0,rows:[]};
function deferred<T>(){let resolve!:(v:T)=>void;const promise=new Promise<T>(r=>{resolve=r;});return {resolve,promise};}
const reads={list:async()=>page,inspect:async(id:string)=>({draft:id,final_request:id+"-final",final_digest:hash}) as Inspection,context:async(id:string)=>({final_request:id}) as Context};
test("late prior selection cannot replace newer draft or start its context",async()=>{
 const slow=deferred<Inspection>(),contexts:string[]=[];
 const store=new DraftStore({...reads,inspect:async id=>id==="old"?slow.promise:reads.inspect(id),context:async id=>{contexts.push(id);return reads.context(id);}},async()=>receipt);
 const old=store.open("old");await store.open("new");slow.resolve(await reads.inspect("old"));await old;
 assert.equal(store.snapshot().inspection.value?.draft,"new");assert.deepEqual(contexts,["new-final"]);
});
test("mutation invalidates in-flight reads",async()=>{
 const slow=deferred<Page<Summary>>(),store=new DraftStore({...reads,list:()=>slow.promise},async()=>receipt);
 const loading=store.list();await store.execute(intent);slow.resolve(page);await loading;
 assert.equal(store.snapshot().list.value,undefined);assert.equal(store.snapshot().entries[0].phase,"recorded");
});
test("unknown operation survives selection changes and clean retry refusal",async()=>{
 const sent:unknown[]=[];
 const store=new DraftStore(reads,async value=>{sent.push(value);throw new DraftError(sent.length===1?"connection_lost":"permission_denied",sent.length===1?"unknown":"none");});
 await store.execute(intent);store.clearSelection();await store.list();
 assert.equal(store.unresolved(),true);await assert.rejects(store.execute({...intent,target:"other",draft:"other"}),DraftError);
 await store.retry("key");assert.equal(store.snapshot().entries[0].phase,"unknown");assert.deepEqual(sent[0],sent[1]);
});
test("pending mutation cannot be duplicated and successful retry settles it",async()=>{
 const pending=deferred<DraftReceipt>();let calls=0;
 const store=new DraftStore(reads,async()=>{calls++;if(calls===1)throw new DraftError("lost","unknown");return pending.promise;});
 await store.execute(intent);const retry=store.retry("key");await store.retry("key");assert.equal(calls,2);
 pending.resolve(receipt);await retry;assert.equal(store.unresolved(),false);assert.equal(store.snapshot().entries.length,1);
});

test("reads started during mutation cannot survive either completion order",async()=>{
 for(const readFirst of [true,false]) {
  const read=deferred<Page<Summary>>(),mutation=deferred<DraftReceipt>();
  const store=new DraftStore({...reads,list:()=>read.promise},()=>mutation.promise);
  const sending=store.execute(intent),loading=store.list();
  if(readFirst){read.resolve(page);await loading;}
  mutation.resolve(receipt);await sending;
  if(!readFirst){read.resolve(page);await loading;}
  assert.equal(store.snapshot().list.value,undefined);assert.equal(store.snapshot().list.loading,false);
 }
});

test("fresh owned recovery resolves business uncertainty without pretending original response succeeded",async()=>{
 let calls=0;
 const inspected={...(await reads.inspect("draft")),revision:"r2",recoveries:[{operation:"submit",business_id:"draft",original_actor:"actor",permission:"allowed",arguments:intent.arguments}]} as Inspection;
 const store=new DraftStore({...reads,inspect:async()=>inspected},async()=>{if(++calls===1)throw new DraftError("cached_internal_error","unknown");return receipt;});
 await store.execute(intent);await store.open("draft");
 const recovery={...intent,revision:"r2",attempt:{request:"new-request",key:"new-key"}};
 assert.equal(store.canRecover({...recovery,actor:"other"}),false);
 assert.equal(store.canRecover({...recovery,revision:"r1"}),false);
 await store.recover(recovery);assert.equal(store.unresolved(),false);
 assert.equal(store.snapshot().entries[0].phase,"reconciled");assert.equal(store.snapshot().entries[0].error,"cached_internal_error");
 assert.equal(store.snapshot().entries[1].phase,"recorded");
});
