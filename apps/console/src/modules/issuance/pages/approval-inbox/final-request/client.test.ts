import test from "node:test";
import assert from "node:assert/strict";
import { previewOf,freezeIntent,type Preview,type Receipt } from "./contract";
import { finalRequestClient } from "./client";
import { FinalRequestStore } from "./store";
import { ResultError } from "../results/contract";
const scope={result:"reported",digest:"a".repeat(64),author:"author",currency:"USD"};
const session={user:"requester",workspace:"workspace",csrf:"synthetic-csrf",role:"member",login:"requester"};
const fields=()=>({mode:"synthetic",receipt:"current_request_preview",request:"final-request",result:scope.result,result_digest:scope.digest,candidate_digest:"b".repeat(64),author:scope.author,requester:session.user,contributors:'["author","capture"]',amount:"16",currency:"USD",net:"15.00",tax:"1.00",amount_basis:"synthetic_reported_net_plus_tax_v1",policy_revision:"2",checked_at:"2026-11-16T09:00:00Z",revision:"r42",request_status:"evidence_ready"});
const preview=()=>previewOf(fields(),scope,"final-request",session.user);
const intent=()=>freezeIntent(preview(),"Checked reported gross",{request:"attempt-one",key:"key-one"});
const receipt:Receipt={target:"final-request",digest:"c".repeat(64),state:"pending",revision:"r43",operation:"op-one",replay:false};
const json=(raw:unknown,status=200)=>new Response(JSON.stringify(raw),{status});
const success=()=>({request_id:"attempt-one",status:"completed",operation_id:"op-one",target_id:"final-request",result_revision:"r43",business_state:"pending",replay:false,as_of:"2026-11-16T09:00:00Z",fields:{mode:"synthetic",request:"final-request",digest:"c".repeat(64),recorded_state:"pending",receipt:"historical_decision",stage:"erp_final_approval"}});
function server(command:(url:string,init:RequestInit)=>Response|Promise<Response>):typeof fetch{return async(input,init)=>String(input).endsWith("/session")?json({...session,csrf_token:session.csrf}):command(String(input),init!);}
function deferred<T>(){let resolve!:(v:T)=>void;let reject!:(e:unknown)=>void;const promise=new Promise<T>((r,j)=>{resolve=r;reject=j;});return {promise,resolve,reject};}

test("preview binds selected source and exact reported amount without inference",()=>{
 const p=preview();assert.equal(p.amount,"16");assert.equal(p.net,"15.00");assert.equal(p.tax,"1.00");assert.equal(Object.isFrozen(p.scope),true);
 for(const patch of [{result:"other"},{result_digest:"d".repeat(64)},{author:"other"},{requester:"other"},{currency:"EUR"},{request:"other"},{amount:"NaN"},{tax:"-1"},{amount_basis:"gross"},{contributors:'["capture","author"]'},{contributors:'["author","author"]'},{contributors:'["other"]'},{extra:"unexpected"},{checked_at:"2026-02-30T09:00:00Z"},{request_status:"eligible"}])assert.throws(()=>previewOf({...fields(),...patch},scope,"final-request",session.user),ResultError);
 const blocked=previewOf({...fields(),request_status:"hold_active"},scope,"final-request",session.user);
 assert.throws(()=>freezeIntent(blocked,"Checked",{request:"a",key:"b"}),ResultError);
});

test("request transport uses fixed family and requires exact final-stage receipt",async()=>{
 const bodies:string[]=[];
 const client=finalRequestClient(session,server((url,init)=>{assert.equal(url,"/api/v1/commands/erp_final.request");bodies.push(String(init.body));return json(success());}));
 assert.equal((await client.submit(intent())).state,"pending");
 const body=JSON.parse(bodies[0]);assert.equal(body.expected_revision,"r42");assert.deepEqual(body.body,{result:"reported",digest:scope.digest});assert.equal(body.rationale,"Checked reported gross");
 for(const patch of [{stage:"charge_approval"},{request:"other"},{recorded_state:"approved"},{digest:"invalid"}])await assert.rejects(finalRequestClient(session,server(()=>json({...success(),fields:{...success().fields,...patch}}))).submit(intent()),(e:ResultError)=>e.effect==="unknown");
 const changed=finalRequestClient(session,async()=>json({...session,csrf_token:"changed"}));
 await assert.rejects(changed.submit(intent()),(e:ResultError)=>e.code==="session_changed"&&e.effect==="none");
});

test("preview envelope is tied to the returned global revision",async()=>{
 const client=finalRequestClient(session,server((url,init)=>{assert.equal(url,"/api/v1/commands/erp_final.preview");const b=JSON.parse(String(init.body));assert.equal(b.idempotency_key,undefined);return json({request_id:b.request_id,status:"completed",target_id:b.target_id,result_revision:"r42",replay:false,as_of:"2026-11-16T09:00:00Z",fields:fields()});}));
 assert.equal((await client.preview(scope,"final-request")).revision,"r42");
});

test("late preview cannot cross source selection or edits",async()=>{
 const pending=deferred<Preview>();const store=new FinalRequestStore({preview:()=>pending.promise,submit:async()=>receipt});
 store.select(scope);store.edit("final-request","Checked");const checking=store.check();store.select({...scope,result:"other"});pending.resolve(preview());await checking;
 assert.equal(store.snapshot().preview,undefined);assert.equal(store.snapshot().scope?.result,"other");
});

test("pending duplicate and unknown retry retain identical immutable request",async()=>{
 const pending=deferred<Receipt>();const seen:unknown[]=[];
 const store=new FinalRequestStore({preview:async()=>preview(),submit:async i=>{seen.push(i);if(seen.length===1)return pending.promise;throw new ResultError("permission_denied","none");}});
 store.select(scope);store.edit("final-request","Checked");await store.check();const sending=store.submit();await store.submit();assert.equal(seen.length,1);
 store.edit("different","different");assert.equal(store.snapshot().target,"final-request");
 pending.reject(new ResultError("response_lost","unknown"));await sending;
 const original=store.snapshot().entries[0].intent;
 store.select({...scope,result:"other"});await store.retry(original.attempt.key);
 assert.equal(seen[0],seen[1]);assert.equal(Object.isFrozen(original.preview.scope),true);assert.equal(store.snapshot().entries[0].phase,"unknown");assert.equal(store.unresolved(),true);
});

test("known request receipt clears only its own draft and retains immutable evidence",async()=>{
 const store=new FinalRequestStore({preview:async()=>preview(),submit:async()=>receipt});
 store.select(scope);store.edit("final-request","Checked");await store.check();await store.submit();
 assert.equal(store.snapshot().target,"");assert.equal(store.snapshot().rationale,"");assert.equal(store.unresolved(),false);
 assert.equal(store.snapshot().entries[0].intent.preview.amount,"16");assert.equal(store.snapshot().entries[0].phase,"recorded");
});

test("preview freezes caller scope before asynchronous session verification",async()=>{
 const pending=deferred<Response>();let sent:Record<string,unknown>|undefined;
 const mutable={...scope};
 const client=finalRequestClient(session,async(input,init)=>{
  if(String(input).endsWith("/session"))return pending.promise;
  const b=JSON.parse(String(init?.body));sent=b.body;
  return json({request_id:b.request_id,status:"completed",target_id:b.target_id,result_revision:"r42",replay:false,as_of:"2026-11-16T09:00:00Z",fields:fields()});
 });
 const checking=client.preview(mutable,"final-request");
 Object.assign(mutable,{result:"other",digest:"d".repeat(64),author:"other-author",currency:"EUR"});
 pending.resolve(json({...session,csrf_token:session.csrf}));
 const p=await checking;
 assert.deepEqual(sent,{result:scope.result,digest:scope.digest});assert.deepEqual(p.scope,scope);
 assert.throws(()=>previewOf({...fields(),amount:"16.00"},scope,"final-request",session.user),ResultError);
});
