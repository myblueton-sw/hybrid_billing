import test from "node:test";
import assert from "node:assert/strict";
import { ResultAttempts } from "./attempts";
import { ResultError, type Preparation } from "./contract";
import type { Intent, Receipt } from "./mutations";
const hash="a".repeat(64),scope:Preparation={preparation:"prep",preparation_digest:hash,document:"doc",document_digest:hash,connection:"erp",customer:"customer",period:"2026-10",currency:"USD",preparation_actor:"actor"};
const intent:Intent={action:"propose",target:"result",fields:{packet:JSON.stringify({Preparation:scope.preparation,PreparationDigest:hash,Connection:scope.connection,Currency:scope.currency,Customer:scope.customer,Period:scope.period})},rationale:"Synthetic",revision:"r1",attempt:{request:"request",key:"key"}};
test("attempt admission blocks duplicate dispatch before first await and retains immutable intent",async()=>{
 let resolve!:(v:Receipt)=>void,calls=0;
 const store=new ResultAttempts(async()=>{calls++;return new Promise(r=>{resolve=r;});});
 const pending=store.submit(scope,intent);
 assert.throws(()=>store.submit(scope,intent),ResultError);assert.equal(calls,1);assert.ok(Object.isFrozen(store.snapshot()[0].intent.fields));
 resolve({target:"result",digest:hash,state:"pending",applicationState:"not_started",applicationReceipt:"",revision:"r2",replay:false,operation:"op"});await pending;
 assert.equal(store.snapshot()[0].phase,"recorded");
});
test("unknown survives subsequent clean refusal and preserves original operation and envelope",async()=>{
 const sent:Intent[]=[];const store=new ResultAttempts(async i=>{sent.push(i);throw new ResultError(sent.length===1?"dependency_unavailable":"permission_denied",sent.length===1?"unknown":"none",sent.length===1?"operation":"");});
 await store.submit(scope,intent);await store.retry("key");
 const entry=store.snapshot()[0];assert.equal(entry.phase,"unknown");assert.equal(entry.operation,"operation");assert.equal(entry.error,"permission_denied");assert.deepEqual(sent[0],sent[1]);
 assert.throws(()=>store.submit(scope,{...intent,attempt:{request:"new",key:"new"}}),ResultError);
});

test("misbound proposal scope never dispatches",()=>{
 let calls=0;const store=new ResultAttempts(async()=>{calls++;throw new Error();});
 assert.throws(()=>store.submit({...scope,preparation:"other"},intent),ResultError);assert.equal(calls,0);assert.equal(store.snapshot().length,0);
});
