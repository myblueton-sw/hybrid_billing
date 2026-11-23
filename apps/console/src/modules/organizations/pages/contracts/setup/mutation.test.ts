import test from "node:test";
import assert from "node:assert/strict";
import { setupMutationClient } from "./mutation";
import { setupIntent } from "./intents";
import { SetupError } from "./context";
import { testDigest,testIntent,testRow,testSession } from "./write-fixture";
function server(change:(raw:Record<string,unknown>)=>void=()=>{},status=200){
 const requests:string[]=[];
 const fetcher:typeof fetch=async(url,init)=>{
  if(String(url)==="/api/v1/session")return Response.json({user:testSession.user,workspace:testSession.workspace,csrf_token:testSession.csrf});
  requests.push(String(init?.body));const body=JSON.parse(String(init?.body));
  const raw:Record<string,unknown>={request_id:body.request_id,status:"completed",operation_id:"operation-1",target_id:body.target_id,result_revision:"r5",business_state:"proposed",replay:false,as_of:"2026-11-16T00:00:00Z",fields:{...testRow(),receipt:"historical_configuration",contract:"contract",current_eligibility:"unchecked"}};
  change(raw);return Response.json(raw,{status});
 };return {fetcher,requests};
}
test("setup request freezes mutable inputs and exact retries reuse the envelope",async()=>{
 const original=testIntent(),input={...original,args:{...original.args},attempt:{...original.attempt}};
 const frozen=setupIntent(input);input.args.languages="ar";input.attempt.key="changed";
 const s=server(),client=setupMutationClient(testSession,s.fetcher);
 const receipt=await client(frozen);await client(frozen);
 assert.equal(receipt.configuration.languages,"en,ko");assert.equal(s.requests[0],s.requests[1]);assert.ok(Object.isFrozen(frozen.args));assert.equal(s.requests[0].includes(testSession.csrf),false);
});
test("misbound or altered successful receipts remain uncertain",async()=>{
 for(const change of [(r:Record<string,unknown>)=>{r.target_id="other";},(r:Record<string,unknown>)=>{r.fields={...r.fields as object,tax:"other"};},(r:Record<string,unknown>)=>{r.fields={...r.fields as object,billing_revision:"4"};},(r:Record<string,unknown>)=>{r.business_state="confirmed";}])await assert.rejects(setupMutationClient(testSession,server(change).fetcher)(testIntent()),(e:unknown)=>e instanceof SetupError&&e.effect==="unknown");
});
test("only correlated documented refusals without an operation are clean",async()=>{
 for(const operation of [false,true]){
  const s=server(r=>{const request=r.request_id;for(const k of Object.keys(r))delete r[k];Object.assign(r,{request_id:request,code:"dependency_unavailable",next_action:"retry_later",retry:true,...(operation?{operation_id:"operation-1"}:{})});},503);
  await assert.rejects(setupMutationClient(testSession,s.fetcher)(testIntent()),(e:unknown)=>e instanceof SetupError&&e.effect===(operation?"unknown":"none"));
 }
 const s=server(r=>{for(const k of Object.keys(r))delete r[k];Object.assign(r,{request_id:"other",code:"permission_denied",next_action:"none",retry:false});},403);
 await assert.rejects(setupMutationClient(testSession,s.fetcher)(testIntent()),(e:unknown)=>e instanceof SetupError&&e.effect==="unknown");
});
test("wrong actor cannot dispatch an intent from another session",async()=>{
 const s=server();await assert.rejects(setupMutationClient({...testSession,user:"other"},s.fetcher)(testIntent()),SetupError);assert.equal(s.requests.length,0);
});
test("successful confirmation binds digest and confirmer while historical proposal replay stays historical",async()=>{
 const original=testIntent(),at="2026-11-16T01:00:00Z";
 const confirmed=(proposer:string,confirmer:string)=>({...testRow(),proposer,confirmer,state:"confirmed",module_observation:"active",approval_observed_at:at,activated_at:at,receipt:"historical_configuration",contract:"contract",current_eligibility:"unchecked"});
 const confirmation=setupIntent({...original,operation:"confirm_configuration",args:{digest:testDigest},rationale:"Independent confirmation",expected:{...original.expected,proposer:"second",digest:testDigest}});
 const s=server(r=>{r.business_state="confirmed";r.fields=confirmed("second","first");});
 assert.equal((await setupMutationClient(testSession,s.fetcher)(confirmation)).configuration.confirmer,"first");
 const wrong=server(r=>{r.business_state="confirmed";r.fields=confirmed("second","third");});
 await assert.rejects(setupMutationClient(testSession,wrong.fetcher)(confirmation),(e:unknown)=>e instanceof SetupError&&e.effect==="unknown");
 const replay=server(r=>{r.business_state="confirmed";r.replay=true;r.fields=confirmed("first","second");});
 const receipt=await setupMutationClient(testSession,replay.fetcher)(original);
 assert.equal(receipt.configuration.state,"confirmed");assert.equal(receipt.replay,true);
});
test("same actor with a different session cannot submit a retained envelope",async()=>{
 let calls=0;
 const fetcher:typeof fetch=async()=>{calls++;throw new Error("must not dispatch");};
 await assert.rejects(setupMutationClient({...testSession,csrf:"different-session"},fetcher)(testIntent()),(e:unknown)=>e instanceof SetupError&&e.code==="session_changed"&&e.effect==="none");
 assert.equal(calls,0);
});
