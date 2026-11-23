import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { contextOf } from "./context";
import { inspectionOf } from "./inspection";
import { DraftError, listPage, type Envelope } from "./contract";
import { draftIntent,draftMutationClient,type DraftIntent } from "./mutation";
const path=process.env.HB_DRAFT_CLIENT_FIXTURE;
test("actual server HTTP journey is accepted by draft decoders and mutation clients",{skip:!path},async()=>{
 const fixture=JSON.parse(readFileSync(path!,"utf8"));
 const envelope=(name:string):Envelope=>({revision:fixture[name].result_revision,at:fixture[name].as_of,fields:fixture[name].fields});
 const request=fixture.createContext.fields.final_request,finalDigest=fixture.createContext.fields.final_digest;
 const context=contextOf(envelope("createContext"),request,finalDigest);
 assert.equal(context.submit_action,"ready");
 const page=listPage(envelope("list"));assert.equal(page.count,1);
 const draft=page.rows[0].draft,actor=fixture.meta.reviewer;
 const submitted=inspectionOf(envelope("submitted"),draft,actor);
 assert.equal(submitted.submission_state,"submitted");
 const ready=contextOf(envelope("prepareContext"),request,finalDigest);assert.equal(ready.prepare_action,"ready");
 const prepared=inspectionOf(envelope("prepared"),draft,actor);assert.equal(prepared.review?.state,"prepared");assert.equal(prepared.references.length,3);assert.equal(prepared.recoveries.length,2);
 for(const corrupt of ["reference_array","refusal_array","review_on_unknown"]) {
  const changed=structuredClone(envelope("prepared")),f={...changed.fields};
  if(corrupt==="reference_array"){const ref=JSON.parse(f.reference_00);ref.field=[ref.field];f.reference_00=JSON.stringify(ref);}
  if(corrupt==="refusal_array"){const review=JSON.parse(f.review);review.state="refused";review.refusal=["reference_changed"];f.review=JSON.stringify(review);delete f.recovery_prepare;delete f.recovery_approve;}
  if(corrupt==="review_on_unknown")f.submission_state="submission_unknown";
  assert.throws(()=>inspectionOf({...changed,fields:f},draft,actor),DraftError);
 }
 const approve=contextOf(envelope("approveContext"),request,finalDigest);assert.equal(approve.approve_action,"ready");assert.equal(approve.revision,prepared.revision);
 const approved=inspectionOf(envelope("approved"),draft,actor);assert.equal(approved.review?.state,"approved");assert.equal(approved.recoveries.length,0);
 for(const action of ["submit","prepare_approval","approve"] as const) {
  const key=action==="prepare_approval"?"prepare":action,wire=fixture[key],user=action==="submit"?fixture.meta.submitter:actor;
  const args:Record<string,string>=action==="submit"?{configuration:context.configuration!,configuration_digest:context.configuration_digest!,final_request:request,final_digest:finalDigest,language:submitted.language}:action==="prepare_approval"?{draft,draft_digest:submitted.digest,binding:ready.rows[0].binding,binding_digest:ready.rows[0].binding_digest}:{digest:prepared.review!.digest};
  const input:DraftIntent=draftIntent({action,target:wire.target_id,actor:user,workspace:fixture.meta.workspace,revision:"r1",draft,draftDigest:action==="submit"?"":submitted.digest,arguments:args,rationale:"Verify actual synthetic server receipt",attempt:{request:wire.request_id,key:"fixture-key"}});
  const session={user,workspace:fixture.meta.workspace,csrf:"synthetic-client-only",role:"member",login:"fixture"};
  const fetcher:typeof fetch=async url=>String(url)==="/api/v1/session"?Response.json({user,workspace:session.workspace,csrf_token:session.csrf}):Response.json(wire);
  const receipt=await draftMutationClient(session,fetcher)(input);
  assert.equal(receipt.draft,draft);assert.equal(receipt.submissionState,"submitted");
 }
});
