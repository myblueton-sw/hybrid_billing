import test from "node:test";
import assert from "node:assert/strict";
import { reviewAction, type ResultDetail } from "./actions";
import { ResultAttempts } from "./attempts";
import { ResultError, type Preparation } from "./contract";
const hash="a".repeat(64),scope:Preparation={preparation:"prep",preparation_digest:hash,document:"doc",document_digest:hash,connection:"erp",customer:"customer",period:"2026-10",currency:"USD",preparation_actor:"actor"};
const allowed={state:"allowed",code:""} as const,ready={state:"ready",code:""} as const;
const initialContext={kind:"initial",position:"current",predecessor:"",predecessor_digest:"",predecessor_revision:0,predecessor_receipt:"",current_result:"result",current_state:"unknown"} as const;
function detail(): { inspection: ResultDetail["inspection"]; advice: ResultDetail["advice"] }{return {inspection:{result:"result",digest:hash,revision:"r1",state:"confirmed",packet:{Mode:"synthetic",Profile:"synthetic_result_one_to_one_v1",Evidence:"source",Preparation:"prep",PreparationDigest:hash,Connection:"erp",External:"external",Currency:"USD",Customer:"customer",Period:"2026-10",Revision:1,Rows:[]},source:"{}",sourceDigest:hash,packetDigest:hash,comparison:[],proposedBy:"proposer",proposedAt:"2026-11-22T00:00:00Z",reason:"Synthetic",confirmation:{actor:"confirmer",at:"2026-11-22T00:00:01Z",reason:"Confirm synthetic"},applicationContext:initialContext,applicationState:"unknown",applicationReceipt:""},advice:{result:"result",digest:hash,revision:"r1",permissions:{confirm:allowed,revoke:allowed,apply:allowed},readiness:{confirm:ready,revoke:ready,apply:ready,recover:ready},applicationContext:initialContext,applicationState:"unknown",applicationReceipt:"",applyingActor:"actor"}};}
const tried={request:"new-request",key:"new-key"};
test("recovery requires exact current scope revision owner and matching evidence",()=>{
 assert.equal(reviewAction(scope,detail(),"actor","recover","Recover synthetic",tried).intent.action,"apply");
 assert.throws(()=>reviewAction(scope,detail(),"other","recover","Recover synthetic",tried),ResultError);
 for(const edit of [(v:ReturnType<typeof detail>)=>{v.advice={...v.advice,revision:"r2"};},(v:ReturnType<typeof detail>)=>{v.inspection={...v.inspection,comparison:["net_mismatch"]};},(v:ReturnType<typeof detail>)=>{v.advice={...v.advice,applicationReceipt:hash};}]){const v=detail();edit(v);assert.throws(()=>reviewAction(scope,v,"actor","recover","Recover synthetic",tried),ResultError);}
 assert.throws(()=>reviewAction({...scope,preparation:"other"},detail(),"actor","recover","Recover synthetic",tried),ResultError);
});
test("recovery uses a fresh request while preserving original unknown attempt",async()=>{
 let count=0;const store=new ResultAttempts(async intent=>{count++;if(count===1)throw new ResultError("dependency_unavailable","unknown","first-operation");return {target:intent.target,digest:hash,state:"confirmed",applicationState:"recorded",applicationReceipt:"b".repeat(64),revision:"r2",replay:false,operation:"recovery-operation"};},"actor");
 const initial=detail();initial.inspection={...initial.inspection,applicationState:"not_started"};initial.advice={...initial.advice,applicationState:"not_started",applyingActor:""};
 await store.submitAction(reviewAction(scope,initial,"actor","apply","Apply synthetic",{request:"first",key:"first"}));
 const prepared=reviewAction(scope,detail(),"actor","recover","Recover synthetic",tried);await store.submitAction(prepared);
 assert.equal(store.snapshot().length,2);const old=store.snapshot().find(e=>e.intent.attempt.key==="first")!;assert.equal(old.phase,"unknown");assert.equal(old.operation,"first-operation");assert.ok(store.settled(old));await store.retry("first");assert.equal(count,2);
 const packet=JSON.stringify({Preparation:scope.preparation,PreparationDigest:hash,Connection:scope.connection,Currency:scope.currency,Customer:scope.customer,Period:scope.period});
 await store.submit(scope,{action:"propose",target:"next-result",fields:{packet},rationale:"New independent evidence",revision:"r2",attempt:{request:"next",key:"next"}});assert.equal(count,3);
});

test("fresh authorized revocation remains possible after unknown application",async()=>{
 let count=0;const store=new ResultAttempts(async intent=>{count++;if(count===1)throw new ResultError("dependency_unavailable","unknown","operation");return {target:intent.target,digest:hash,state:"revoked",applicationContext:initialContext,applicationState:"unknown",applicationReceipt:"",revision:"r2",replay:false,operation:"revocation"};},"actor");
 const initial=detail();initial.inspection={...initial.inspection,applicationState:"not_started"};initial.advice={...initial.advice,applicationState:"not_started",applyingActor:""};
 await store.submitAction(reviewAction(scope,initial,"actor","apply","Apply synthetic",{request:"first",key:"first"}));
 await store.submitAction(reviewAction(scope,detail(),"actor","revoke","Withdraw evidence without rollback",tried));
 assert.equal(count,2);assert.equal(store.snapshot()[0].phase,"unknown");assert.equal(store.snapshot()[1].receipt?.state,"revoked");
});

test("review freezes matching replacement context and rejects a changed inspected head",()=>{
 const v=detail();
 const context={kind:"replacement",position:"current",predecessor:"old",predecessor_digest:hash,predecessor_revision:1,predecessor_receipt:hash,current_result:"result",current_state:"unknown"} as const;
 v.inspection={...v.inspection,applicationContext:context};v.advice={...v.advice,applicationContext:context};
 const review=reviewAction(scope,v,"actor","recover","Recover owned replacement",tried);
 assert.equal(review.detail.advice.applicationContext.predecessor,"old");assert.ok(Object.isFrozen(review.detail.advice.applicationContext));
 const changed=detail();changed.inspection={...changed.inspection,applicationContext:context};
 assert.throws(()=>reviewAction(scope,changed,"actor","recover","Recover owned replacement",tried),ResultError);
});
