import assert from "node:assert/strict";
import test from "node:test";
import { draftReadClient } from "./client";
import { DraftError, fixedMoney } from "./contract";
const session={user:"reviewer",workspace:"workspace",csrf:"synthetic-csrf",role:"member",login:"reviewer"};
const hash="a".repeat(64);
const empty={mode:"synthetic",current_eligibility:"unchecked",snapshot:hash,next:"",count:"0"};
function server(change:(raw:Record<string,unknown>)=>void=()=>{}) {
 const requests:{url:string;body:Record<string,unknown>}[]=[];
 const fetcher:typeof fetch=async(input,init)=>{
  const url=String(input);
  if(url==="/api/v1/session")return Response.json({user:session.user,workspace:session.workspace,csrf_token:session.csrf});
  const body=JSON.parse(String(init?.body));requests.push({url,body});
  const raw={request_id:body.request_id,status:"completed",target_id:body.target_id,result_revision:"r4",replay:false,as_of:"2026-11-23T00:00:00Z",fields:{...empty}};
  change(raw);return Response.json(raw);
 };
 return {fetcher,requests};
}
test("draft list stays within current workspace and snapshot",async()=>{
 const s=server(),page=await draftReadClient(session,s.fetcher).list();
 assert.equal(s.requests[0].url,"/api/v1/commands/issuance_draft.list");
 assert.equal(s.requests[0].body.target_id,session.workspace);
 assert.deepEqual(s.requests[0].body.body,{});
 assert.equal(page.snapshot,hash);assert.equal(page.count,0);assert.ok(Object.isFrozen(page.rows));
});
test("changed session prevents draft access",async()=>{
 let calls=0;
 const f:typeof fetch=async()=>{calls++;return Response.json({user:"other",workspace:session.workspace,csrf_token:session.csrf});};
 await assert.rejects(draftReadClient(session,f).list(),(e:unknown)=>e instanceof DraftError&&e.code==="session_changed");
 assert.equal(calls,1);
});
test("mismatched identity and mutation envelopes are rejected",async()=>{
 for(const change of [(r:Record<string,unknown>)=>{r.target_id="other";},(r:Record<string,unknown>)=>{r.request_id="other";},(r:Record<string,unknown>)=>{r.operation_id="operation";},(r:Record<string,unknown>)=>{r.replay=true;}]) {
  await assert.rejects(draftReadClient(session,server(change).fetcher).list(),DraftError);
 }
});
test("cursor requires its snapshot and canonical line index",async()=>{
 const s=server(),client=draftReadClient(session,s.fetcher);
 await assert.rejects(client.list("draft"),DraftError);
 await assert.rejects(client.list("",hash),DraftError);
 await assert.rejects(client.inspect("draft","09",hash),DraftError);
 assert.equal(s.requests.length,0);
});
test("paged reads reject changed snapshot",async()=>{
 const s=server(r=>{r.fields={...empty,snapshot:"b".repeat(64)};});
 await assert.rejects(draftReadClient(session,s.fetcher).list("draft",hash),DraftError);
 assert.deepEqual(s.requests[0].body.body,{after:"draft",snapshot:hash});
});
test("money preserves precision beyond binary integer range",()=>{
 assert.equal(fixedMoney("9007199254740993.12",4),"9007199254740993.1200");
 assert.equal(fixedMoney("0",0),"0");
 assert.throws(()=>fixedMoney("1.001",2),DraftError);
});

test("initial list cannot hide retained rows as an empty or truncated page",async()=>{
 for(const count of ["1","11"]) {
  const s=server(r=>{r.fields={...empty,count};});
  await assert.rejects(draftReadClient(session,s.fetcher).list(),DraftError);
 }
});
