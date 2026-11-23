import assert from "node:assert/strict";
import test from "node:test";
import { setupReadClient } from "./client";
import { SetupError } from "./context";
const session = { user: "reader", workspace: "workspace", csrf: "synthetic-csrf", role: "member", login: "reader" };
const hash = "a".repeat(64);
const context = { schema: "hb_issuance_setup_context_v1", contract: "contract-1", contract_name: "Example", on: "2024-02-29", lifecycle_sequence: "2", contract_revision: "1", billing_profile: "profile-1", billing_profile_name: "Example profile", billing_revision: "3", buyer: "buyer", payer: "payer", issuer: "issuer", currency: "USD", scope_digest: hash, profile_basis: "current", current_eligibility: "unchecked" };
const options = { schema: "hb_issuance_setup_options_v1", contract: "contract-1", on: context.on, scope_digest: hash, current_eligibility: "unchecked", option: "synthetic_erp_sample_v1", applicability: "synthetic_only", mode: "erp_issuance", profile: "synthetic_erp_document_v1", region: "US", tax: "synthetic-tax", zone: "UTC", languages: "ar,en,ko" };
function server(change: (raw: Record<string, unknown>) => void = () => {}) {
 const requests: {url:string;body:Record<string,unknown>;init:RequestInit|undefined}[]=[];
 const fetcher:typeof fetch=async(input,init)=>{
  const url=String(input);
  if(url==="/api/v1/session")return Response.json({user:session.user,workspace:session.workspace,csrf_token:session.csrf});
  const body=JSON.parse(String(init?.body)); requests.push({url,body,init});
  const raw:Record<string,unknown>={request_id:body.request_id,status:"completed",target_id:body.target_id,result_revision:"r4",replay:false,as_of:"2026-11-23T00:00:00Z",fields:{...(url.endsWith("options")?options:context)}};
  change(raw);return Response.json(raw);
 };
 return {fetcher,requests};
}
test("context then options bind actual selected scope and carry session protection",async()=>{
 const s=server(),client=setupReadClient(session,s.fetcher);
 const scope=await client.context("contract-1",context.on), result=await client.options(scope);
 assert.deepEqual(s.requests[1].body.body,{on:context.on,scope_digest:hash});
 assert.equal(s.requests[1].url,"/api/v1/commands/issuance_setup.options");
 assert.equal(s.requests[1].init?.cache,"no-store");
 assert.equal(new Headers(s.requests[1].init?.headers).get("X-CSRF-Token"),session.csrf);
 assert.equal(result.applicability,"synthetic_only");assert.equal(result.currentEligibility,"unchecked");
 assert.ok(Object.isFrozen(result.languages));
});
test("malformed input and changed session never issue setup commands",async()=>{
 const s=server(),client=setupReadClient(session,s.fetcher);
 await assert.rejects(client.context("contract-1","2025-02-29"),SetupError);
 assert.equal(s.requests.length,0);
 let calls=0;
 const f:typeof fetch=async()=>{calls++;return Response.json({user:"other",workspace:session.workspace,csrf_token:session.csrf});};
 await assert.rejects(setupReadClient(session,f).context("contract-1",context.on),(e:unknown)=>e instanceof SetupError&&e.code==="session_changed");
 assert.equal(calls,1);
});
test("cross-request envelopes, stale scope and false legal applicability are refused",async()=>{
 for(const change of [(r:Record<string,unknown>)=>{r.request_id="other";},(r:Record<string,unknown>)=>{r.target_id="other";},(r:Record<string,unknown>)=>{r.replay=true;},(r:Record<string,unknown>)=>{r.operation_id="mutation";}]) {
  await assert.rejects(setupReadClient(session,server(change).fetcher).context("contract-1",context.on),SetupError);
 }
 const scope=await setupReadClient(session,server().fetcher).context("contract-1",context.on);
 for(const change of [{scope_digest:"b".repeat(64)},{on:"2026-11-23"},{applicability:"statutory"},{current_eligibility:"eligible"},{languages:"ko,en,ar"}]) {
  const s=server(r=>{r.fields={...options,...change};});
  await assert.rejects(setupReadClient(session,s.fetcher).options(scope),SetupError);
 }
});
