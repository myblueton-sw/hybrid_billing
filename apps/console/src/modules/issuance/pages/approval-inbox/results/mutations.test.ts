import test from "node:test";
import assert from "node:assert/strict";
import { resultMutationTransport, type Intent } from "./mutations";
import { ResultError } from "./contract";
const hash="a".repeat(64), session={user:"actor",workspace:"workspace",csrf:"synthetic",role:"operator",login:"synthetic"};
const intent: Intent={action:"apply",target:"result",fields:{digest:hash},rationale:"Apply independent synthetic evidence",revision:"r1",attempt:{request:"request",key:"key"}};
const current=()=>Response.json({...session,csrf_token:session.csrf});
const receipt=()=>({request_id:"request",target_id:"result",operation_id:"operation",status:"completed",result_revision:"r2",business_state:"recorded",replay:false,as_of:"2026-11-22T00:00:00Z",fields:{mode:"synthetic",result:"result",digest:hash,recorded_state:"confirmed",receipt:"historical_decision",application_state:"recorded",application_receipt:"b".repeat(64)}});
test("result apply requires a bound recorded application receipt and freezes the request",async()=>{
 let body="";
 const submit=resultMutationTransport(session,async(input,init)=>{if(String(input).endsWith("/session"))return current();body=String(init?.body);assert.ok(String(input).endsWith("/erp_result.apply"));return Response.json(receipt());});
 const got=await submit(intent);assert.equal(got.applicationState,"recorded");assert.equal(JSON.parse(body).idempotency_key,"key");assert.ok(Object.isFrozen(got));
});
test("result mutation separates clean refusal from possible effects",async()=>{
 for(const [status,payload,effect] of [
  [403,{request_id:"request",code:"permission_denied",retry:false,next_action:"none"},"none"],
  [403,{request_id:"request",code:"permission_denied",retry:false,next_action:"none",operation_id:"operation"},"unknown"],
  [503,{request_id:"request",code:"dependency_unavailable",operation_id:"operation"},"unknown"],
  [200,{...receipt(),fields:{...receipt().fields,application_receipt:""}},"unknown"],
  [200,{...receipt(),target_id:"other"},"unknown"],
 ] as const){
  const submit=resultMutationTransport(session,async input=>String(input).endsWith("/session")?current():Response.json(payload,{status}));
  await assert.rejects(submit(intent),(e:unknown)=>e instanceof ResultError&&e.effect===effect);
 }
});
test("transport loss after dispatch is unknown and changed session never dispatches",async()=>{
 let calls=0;
 const submit=resultMutationTransport(session,async input=>{calls++;if(String(input).endsWith("/session"))return current();throw new Error("lost");});
 await assert.rejects(submit(intent),(e:unknown)=>e instanceof ResultError&&e.effect==="unknown");assert.equal(calls,2);
 calls=0;const changed=resultMutationTransport(session,async()=>{calls++;return Response.json({...session,user:"other",csrf_token:session.csrf});});
 await assert.rejects(changed(intent),(e:unknown)=>e instanceof ResultError&&e.code==="session_changed"&&e.effect==="none");assert.equal(calls,1);
});

test("fresh-key idempotent proposal and confirmation accept historical application receipts",async()=>{
 for(const action of ["propose","confirm"] as const)for(const state of (action==="propose"?["confirmed","revoked"]:["confirmed"])){
  const response={...receipt(),business_state:state,fields:{...receipt().fields,recorded_state:state}};
  const submit=resultMutationTransport(session,async input=>String(input).endsWith("/session")?current():Response.json(response));
  const got=await submit({...intent,action,fields:action==="propose"?{packet:"{}"}:{digest:hash}});
  assert.equal(got.state,state);assert.equal(got.applicationState,"recorded");
 }
});
