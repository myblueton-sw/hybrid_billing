import test from "node:test";
import assert from "node:assert/strict";
import {capabilities,type Ceiling} from "./contract";
import {reviewProposal,ProposalError,type ProposalContext,type ProposalDraft} from "./proposal-model";
const hash="a".repeat(64),at="2026-11-22T09:00:00.000000001Z";
const ceiling:Ceiling={ceiling:`result-grant-${hash}`,capability:"apply",not_before:at,not_after:"2026-11-22T12:00:00.000000001Z"};
const page={revision:"r1",at,snapshot:hash,next:"",capability:"apply" as const,policy:{capabilities,proposal_lifetime_seconds:"3600",max_grant_validity_seconds:"86400"}};
const context:ProposalContext={kind:"human_grant",capability:"apply",ceilings:{...page,collection:"ceilings",rows:[ceiling]},recipients:{...page,collection:"recipients",rows:[{user:"member",login:"synthetic"}]}};
const draft:ProposalDraft={recipient:"member",ceiling:ceiling.ceiling,notBefore:at,notAfter:ceiling.not_after,rationale:"Reviewed synthetic permission"},attempt={request:"request",key:"key"};
test("proposal review binds selected current options and lossless validity into frozen intent",()=>{
 const review=reviewProposal(context,draft,"proposal",attempt);assert.equal(review.intent.action,"propose_human");assert.deepEqual(review.intent.fields,{capability:"apply",recipient:"member",ceiling:ceiling.ceiling,not_before:at,not_after:ceiling.not_after});assert.equal(review.intent.revision,"r1");assert.ok(Object.isFrozen(review.draft));assert.ok(Object.isFrozen(review.context.ceilings.rows));
});
test("proposal refuses foreign options mixed revisions and one-nanosecond ceiling escape",()=>{
 for(const changed of [{...draft,recipient:"other"},{...draft,ceiling:`result-grant-${"b".repeat(64)}`},{...draft,notBefore:"2026-11-22T09:00:00Z"},{...draft,notAfter:"2026-11-22T12:00:00.000000002Z"},{...draft,rationale:"two\nlines"}])assert.throws(()=>reviewProposal(context,changed,"proposal",attempt),ProposalError);
 assert.throws(()=>reviewProposal({...context,recipients:{...context.recipients!,revision:"r2"}},draft,"proposal",attempt),ProposalError);
});
test("console proposal never carries hidden person or ceiling selections",()=>{
 const c={...context,kind:"console_ceiling" as const,recipients:undefined};
 assert.throws(()=>reviewProposal(c,draft,"proposal",attempt),ProposalError);
 const review=reviewProposal(c,{...draft,recipient:"",ceiling:""},"proposal",attempt);assert.equal(review.intent.action,"propose_console");assert.deepEqual(Object.keys(review.intent.fields).sort(),["capability","not_after","not_before"]);
});
