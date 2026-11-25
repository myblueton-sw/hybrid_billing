import assert from "node:assert/strict";
import test from "node:test";
import { contextOf } from "./context";
import { submissionIntent, recoveryIntent } from "./intents";
import { inspectionOf } from "./inspection";
import { DraftError } from "./contract";
const hash="a".repeat(64),session={user:"actor",workspace:"workspace",csrf:"synthetic-csrf",role:"member",login:"actor"};
const fields={mode:"synthetic",final_request:"final",final_digest:hash,final_state:"approved",customer_id:"customer",period:"2026-11",currency:"USD",scale:"2",total:"16",current_eligibility:"eligible",readiness:"ready",submit_permission:"allowed",approve_permission:"allowed",submit_action:"ready",prepare_action:"draft_required",approve_action:"draft_required",step_up:"satisfied",snapshot:hash,next_binding:"",binding_count:"0",configuration:"configuration",configuration_digest:hash,languages:'["en","ko"]'};
function context(){return contextOf({revision:"r3",at:"2026-11-23T00:00:00Z",fields},"final",hash);}
test("submission derives frozen source and generated target without typed identifiers",()=>{
 const c=context(),a=submissionIntent(session,c,"ko","Create reviewed draft"),b=submissionIntent(session,c,"ko","Create reviewed draft");
 assert.notEqual(a.target,b.target);assert.equal(a.target,a.draft);assert.notEqual(a.target,a.attempt.request);
 assert.equal(a.arguments.configuration,c.configuration);assert.equal(a.arguments.final_digest,c.final_digest);
 assert.ok(Object.isFrozen(a.arguments));
});
test("blocked readiness and unsupported language cannot form a new intent",()=>{
 const c=context();
 assert.throws(()=>submissionIntent(session,{...c,submit_action:"owned_by_other"},"ko","Create"),DraftError);
 assert.throws(()=>submissionIntent(session,c,"ar","Create"),DraftError);
 assert.throws(()=>submissionIntent(session,{...c,step_up:"reauthentication_required"},"ko","Create"),DraftError);
});
test("recovery preserves original business arguments and requires original actor",()=>{
 const args={configuration:"configuration",configuration_digest:hash,final_request:"final",final_digest:hash,language:"ko"};
 const draft=inspectionOf({revision:"r4",at:"2026-11-23T00:00:00Z",fields:{mode:"synthetic",current_eligibility:"unchecked",snapshot:hash,next_line:"",line_count:"1",draft:"draft",digest:hash,origin:"configured",final_request:"final",final_digest:hash,customer_id:"customer",period:"2026-11",currency:"USD",scale:"2",total:"16",language:"ko",submitter:"actor",final_requester:"requester",submission_state:"submission_unknown",module_observation:"unavailable",receipt:"submission_record",configuration:"configuration",configuration_digest:hash,line_00:JSON.stringify({index:"0",text:"Service",amount:"16",tax:false}),recovery_submit:JSON.stringify({operation:"submit",original_actor:"actor",business_id:"draft",state:"submission_unknown",permission:"allowed",arguments:args})}},"draft","actor");
 const r=draft.recoveries[0],a=recoveryIntent(session,draft,r,"Reconcile original submission");
 assert.equal(a.target,"draft");assert.deepEqual(a.arguments,args);assert.equal(a.revision,"r4");
 assert.throws(()=>recoveryIntent({...session,user:"other"},draft,r,"Recover"),DraftError);
 assert.throws(()=>recoveryIntent(session,draft,{...r,arguments:{...args,language:"en"}},"Recover"),DraftError);
});

test("initial context cannot hide binding choices",()=>{
 for(const count of ["1","11"])assert.throws(()=>contextOf({revision:"r3",at:"2026-11-23T00:00:00Z",fields:{...fields,binding_count:count}},"final",hash),DraftError);
});
