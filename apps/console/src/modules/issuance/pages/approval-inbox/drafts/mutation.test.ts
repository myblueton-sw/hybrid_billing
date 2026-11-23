import assert from "node:assert/strict";
import test from "node:test";
import { DraftError } from "./contract";
import { draftIntent, draftMutationClient, type DraftIntent } from "./mutation";
const hash="a".repeat(64),reviewHash="b".repeat(64);
const session={user:"reviewer",workspace:"workspace",csrf:"synthetic-csrf",role:"member",login:"reviewer"};
function intent(action:DraftIntent["action"]="submit"):DraftIntent {
 return draftIntent({action,target:action==="submit"?"draft":"review",actor:session.user,workspace:session.workspace,revision:"r2",draft:"draft",draftDigest:action==="submit"?"":hash,arguments:action==="submit"?{configuration:"configuration",configuration_digest:hash,final_request:"final",final_digest:hash,language:"ko"}:action==="prepare_approval"?{draft:"draft",draft_digest:hash,binding:"binding",binding_digest:hash}:{digest:reviewHash},rationale:"Verified source and payment terms",attempt:{request:"request",key:"key"}});
}
function server(reply:(body:Record<string,unknown>,count:number)=>Response|Promise<Response>) {
 const bodies:string[]=[];
 const f:typeof fetch=async(input,init)=>{
  if(String(input)==="/api/v1/session")return Response.json({user:session.user,workspace:session.workspace,csrf_token:session.csrf});
  bodies.push(String(init?.body));return reply(JSON.parse(bodies.at(-1)!),bodies.length);
 };
 return {f,bodies};
}
function receipt(body:Record<string,unknown>,action:DraftIntent["action"],state="approved") {
 const review=action!=="submit";
 return {request_id:body.request_id,status:"completed",operation_id:"operation",target_id:body.target_id,result_revision:"r3",business_state:review?state:"submitted",replay:false,as_of:"2026-11-23T00:00:00Z",fields:{mode:"synthetic",draft:"draft",digest:hash,submission_state:"submitted",receipt:review?state==="approved"?"historical_approval":"review_attempt":"submission_record",current_eligibility:"unchecked",...(review?{review:"review",review_digest:reviewHash,review_state:state}:{})}};
}
test("all draft operations accept correlated historical receipts",async()=>{
 for(const action of ["submit","prepare_approval","approve"] as const) {
  const state=action==="prepare_approval"?"prepared":"approved",s=server(body=>Response.json(receipt(body,action,state)));
  const value=await draftMutationClient(session,s.f)(intent(action));
  assert.equal(value.draft,"draft");assert.equal(value.reviewState,action==="submit"?undefined:state);
  assert.equal(JSON.parse(s.bodies[0]).expected_revision,"r2");
 }
});
test("transport uncertainty retains exact immutable intent for retry",async()=>{
 const source=intent(),s=server((body,count)=>{if(count===1)throw new Error("connection lost");return Response.json(receipt(body,"submit"));}),send=draftMutationClient(session,s.f);
 await assert.rejects(send(source),(e:unknown)=>e instanceof DraftError&&e.effect==="unknown");
 await send(source);assert.equal(s.bodies[0],s.bodies[1]);assert.ok(Object.isFrozen(source.arguments));
});
test("clean unavailable differs from an effect-bearing or malformed failure",async()=>{
 for(const [code,status] of [["dependency_unavailable",503],["authority_unavailable",503]] as const) {
  for(const variation of ["clean","operation","mismatch","retry"] as const) {
   const s=server(body=>Response.json({request_id:variation==="mismatch"?"other":body.request_id,code,retry:variation!=="retry",next_action:"retry_later",...(variation==="operation"?{operation_id:"operation"}:{})},{status}));
   await assert.rejects(draftMutationClient(session,s.f)(intent()),(e:unknown)=>e instanceof DraftError&&e.effect===(variation==="clean"?"none":"unknown"));
  }
 }
});
test("wrong draft, digest or review receipt stays uncertain",async()=>{
 for(const field of ["draft","digest","review","review_digest"] as const) {
  const s=server(body=>{const r=receipt(body,"approve");Object.assign(r.fields,{[field]:field.includes("digest")?"c".repeat(64):"other"});return Response.json(r);});
  await assert.rejects(draftMutationClient(session,s.f)(intent("approve")),(e:unknown)=>e instanceof DraftError&&e.effect==="unknown"&&e.operation==="operation");
 }
});
test("foreign actor and mismatched prepared draft never reach transport",async()=>{
 const s=server(()=>{throw new Error("must not send");}),send=draftMutationClient(session,s.f);
 await assert.rejects(send({...intent(),actor:"other"}),DraftError);
 assert.throws(()=>draftIntent({...intent("prepare_approval"),draft:"other"}),DraftError);
 assert.equal(s.bodies.length,0);
});
test("known refused review is a receipt rather than fabricated approval",async()=>{
 const s=server(body=>Response.json(receipt(body,"approve","refused")));
 const result=await draftMutationClient(session,s.f)(intent("approve"));
 assert.equal(result.reviewState,"refused");assert.equal(result.receipt,"review_attempt");
});

test("nonterminal success cannot settle submission or approval uncertainty",async()=>{
 for(const state of ["submission_unknown","submission_refused"]) {
  const s=server(body=>{const r=receipt(body,"submit");r.business_state=state;r.fields.submission_state=state;return Response.json(r);});
  await assert.rejects(draftMutationClient(session,s.f)(intent()),(e:unknown)=>e instanceof DraftError&&e.effect==="unknown");
 }
 for(const state of ["prepared","approval_unknown"]) {
  const s=server(body=>Response.json(receipt(body,"approve",state)));
  await assert.rejects(draftMutationClient(session,s.f)(intent("approve")),(e:unknown)=>e instanceof DraftError&&e.effect==="unknown");
 }
});
