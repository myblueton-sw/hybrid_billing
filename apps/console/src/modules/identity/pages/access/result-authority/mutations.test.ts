import test from "node:test";
import assert from "node:assert/strict";
import {AuthorityError} from "./contract";
import {freezeIntent,mutationTransport,type Intent} from "./mutations";
const hash="a".repeat(64),session={user:"governor",workspace:"workspace",csrf:"synthetic",role:"admin",login:"synthetic"};
const intent:Intent={action:"propose_console",target:"proposal",fields:{capability:"apply",not_before:"2026-11-22T09:00:00.000000001Z",not_after:"2026-11-22T10:00:00Z"},rationale:"Synthetic review",revision:"r1",attempt:{request:"request",key:"key"}};
const fields={mode:"synthetic",receipt:"historical_decision",proposal:"proposal",digest:hash,state:"pending",kind:"console_ceiling",capability:"apply",recipient:"console",ceiling:"",proposer:session.user,not_before:intent.fields.not_before,not_after:intent.fields.not_after,proposed_at:"2026-11-22T08:59:00Z",expires_at:"2026-11-22T09:59:00Z",rationale:intent.rationale};
const sessionReply=()=>Response.json({user:session.user,workspace:session.workspace,csrf_token:session.csrf});
const receipt=(overrides={})=>({request_id:"request",status:"completed",operation_id:"operation",target_id:"proposal",result_revision:"r2",business_state:"pending",replay:false,as_of:"2026-11-22T09:00:00Z",fields,...overrides});
test("authority proposal submits immutable revision-bound intent and validates exact receipt",async()=>{
 const submit=mutationTransport(session,async(url,init)=>{
  if(String(url).endsWith("/session"))return sessionReply();
  assert.equal(String(url),"/api/v1/commands/result_authority.propose_console");
  assert.deepEqual(JSON.parse(String(init?.body)),{request_id:"request",target_id:"proposal",idempotency_key:"key",expected_revision:"r1",rationale:intent.rationale,body:intent.fields});
  return Response.json(receipt());
 });
 const result=await submit(intent);assert.equal(result.inspection.notBefore,intent.fields.not_before);assert.ok(Object.isFrozen(result.inspection.row));
 const frozen=freezeIntent(intent);assert.notEqual(frozen,intent);assert.ok(Object.isFrozen(frozen.fields));
});
test("authority mutation correlates effect-free refusal conservatively",async()=>{
 for(const ambiguous of [false,true]){
  const submit=mutationTransport(session,async url=>String(url).endsWith("/session")?sessionReply():Response.json({code:"permission_denied",request_id:ambiguous?"other":"request",retry:false,next_action:"none"},{status:403}));
  await assert.rejects(submit(intent),e=>e instanceof AuthorityError&&e.effect===(ambiguous?"unknown":"none"));
 }
});
test("authority mutation never treats interrupted or mismatched success as no effect",async()=>{
 for(const variant of ["disconnect","actor","time","rationale","state","duplicate"]){
  const submit=mutationTransport(session,async url=>{
   if(String(url).endsWith("/session"))return sessionReply();
   if(variant==="disconnect")throw new Error("interrupted");
   if(variant==="duplicate")return new Response('{"fields":{},"fields":{}}');
   const changed={...fields,...({actor:{proposer:"other"},time:{not_before:"2026-11-22T09:00:00.000000002Z"},rationale:{rationale:"other"},state:{state:"applied"}}[variant])};
   return Response.json(receipt({fields:changed}));
  });
  await assert.rejects(submit(intent),e=>e instanceof AuthorityError&&e.effect==="unknown",variant);
 }
});
test("authority decision requires matching digest actor rationale and outcome",async()=>{
 const decision:Intent={...intent,action:"approve",fields:{digest:hash}};
 const applied={...fields,proposer:"other",state:"applied",decided_at:"2026-11-22T09:00:00Z",decided_by:session.user,decision_reason:decision.rationale,grant:`result-grant-${hash}`,recorded_grant_status:"active"};
 for(const corrupt of [false,true]){
  const submit=mutationTransport(session,async url=>String(url).endsWith("/session")?sessionReply():Response.json(receipt({business_state:"applied",fields:{...applied,...(corrupt?{decided_by:"someone-else"}:{})}})));
  if(corrupt)await assert.rejects(submit(decision),e=>e instanceof AuthorityError&&e.effect==="unknown");
  else assert.equal((await submit(decision)).inspection.decision?.grant,`result-grant-${hash}`);
 }
});
test("invalid intent or changed session cannot dispatch authority mutation",async()=>{
 let calls=0;
 const submit=mutationTransport(session,async()=>{calls++;return Response.json({user:"changed",workspace:session.workspace,csrf_token:session.csrf});});
 await assert.rejects(submit({...intent,fields:{...intent.fields,not_after:intent.fields.not_before}}),AuthorityError);assert.equal(calls,0);
 await assert.rejects(submit(intent),e=>e instanceof AuthorityError&&e.code==="session_changed"&&e.effect==="none");assert.equal(calls,1);
});

test("authority rationale follows the registry byte and control-character contract",async()=>{
 for(const rationale of ["x".repeat(1025),"line\nbreak","tab\tvalue","\u007f","\ud800"]){
  assert.throws(()=>freezeIntent({...intent,rationale}),AuthorityError);
 }
 assert.equal(freezeIntent({...intent,rationale:"x".repeat(1024)}).rationale.length,1024);
 assert.throws(()=>freezeIntent({...intent,rationale:"가".repeat(342)}),AuthorityError);
});

test("stage refusal is effect-free only with the complete correlated refusal shape",async()=>{
 for(const extra of [{},{operation_id:"operation"},{retry:true},{next_action:"none"}]){
  const submit=mutationTransport(session,async url=>String(url).endsWith("/session")?sessionReply():Response.json({code:"invalid_stage",request_id:"request",retry:false,next_action:"correct_request",...extra},{status:422}));
  await assert.rejects(submit(intent),e=>e instanceof AuthorityError&&e.effect===(Object.keys(extra).length?"unknown":"none"));
 }
});
test("historical proposal replay and revoked or expired approval receipts do not imply active rights",async()=>{
 for(const status of ["active","revoked","expired"]){
  const history={...fields,state:"applied",decided_at:"2026-11-22T09:00:00Z",decided_by:"other",decision_reason:"Independent approval",grant:`result-grant-${hash}`,recorded_grant_status:status};
  const propose=mutationTransport(session,async url=>String(url).endsWith("/session")?sessionReply():Response.json(receipt({business_state:"applied",replay:true,fields:history})));
  const replay=await propose(intent);assert.equal(replay.inspection.decision?.recordedGrantStatus,status);assert.equal(replay.replay,true);
  const approve=mutationTransport(session,async url=>String(url).endsWith("/session")?sessionReply():Response.json(receipt({business_state:"applied",fields:{...history,proposer:"other",decided_by:session.user,decision_reason:intent.rationale}})));
  const applied=await approve({...intent,action:"approve",fields:{digest:hash}});assert.equal(applied.inspection.decision?.recordedGrantStatus,status);
 }
});
