import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {historyPage,contextPage} from "./reads";
import {inspectionOf} from "./inspection";
import {AuthorityError,capabilities,type Envelope,type Read} from "./contract";
const hash="a".repeat(64),now="2026-11-22T09:00:00Z";
const historyRow=(proposal:string)=>({proposal,digest:hash,kind:"console_ceiling",capability:"apply",recipient:"console",state:"pending",proposer:"other",proposed_at:now,expires_at:"2026-11-22T10:00:00Z",decision_actor:""});
const variants=(...values:Record<string,string>[])=>values;
const page=(fields:Record<string,string>):Envelope=>({revision:"r1",at:now,fields:{mode:"synthetic",workspace:"workspace",actor:"actor",current_eligibility:"unchecked",snapshot:hash,next:"",...fields}});
test("history validates actor, fixed snapshot, contiguous sorted unique rows and continuation",async()=>{
 const good=page({row_00:JSON.stringify(historyRow("p1"))});assert.equal((await historyPage(async()=>good,"workspace","actor")).rows.length,1);
 for(const fields of variants({actor:"other"},{snapshot:"bad"},{row_01:good.fields.row_00},{row_01:good.fields.row_00,row_00:good.fields.row_00},{next:"p1"}))await assert.rejects(historyPage(async()=>({...good,fields:{...good.fields,...fields}}),"workspace","actor"),AuthorityError);
 await assert.rejects(historyPage(async()=>good,"workspace","actor",{after:"p0",snapshot:"b".repeat(64)}),e=>e instanceof AuthorityError&&e.code==="evidence_stale");
});
test("context rejects wrong capability, self recipient, duplicate policy and foreign grant prefix",async()=>{
 const policy=JSON.stringify({capabilities,proposal_lifetime_seconds:"3600",max_grant_validity_seconds:"86400"});
 const good=page({collection:"recipients",capability:"apply",policy,row_00:JSON.stringify({user:"member",login:"member@example.invalid"})});
 assert.equal((await contextPage(async()=>good,"workspace","actor","recipients","apply")).rows.length,1);
 for(const fields of variants({capability:"inspect"},{row_00:JSON.stringify({user:"actor",login:"actor"})},{policy:'{"capabilities":[],"capabilities":[]}'}))await assert.rejects(contextPage(async()=>({...good,fields:{...good.fields,...fields}}),"workspace","actor","recipients","apply"),AuthorityError);
 await assert.rejects(contextPage(async()=>page({collection:"ceilings",capability:"apply",policy,row_00:JSON.stringify({ceiling:`document-grant-${hash}`,capability:"apply",not_before:now,not_after:"2026-11-23T09:00:00Z"})}),"workspace","actor","ceilings","apply"),AuthorityError);
});
test("actual Go authority projection interoperates without truncating fractional validity",{skip:!process.env.HB_RESULT_AUTHORITY_CLIENT_FIXTURE},async()=>{
 const wire=JSON.parse(readFileSync(process.env.HB_RESULT_AUTHORITY_CLIENT_FIXTURE!,"utf8"));
 const convert=(v:any):Envelope=>({revision:v.result_revision,at:v.as_of,fields:v.fields});
 const read:Read=async(command,_target,fields)=>convert(command.endsWith("history")?wire.history:wire[fields.collection]);
 const workspace=wire.history.fields.workspace,actor=wire.history.fields.actor;
 const history=await historyPage(read,workspace,actor);assert.equal(history.rows.length,2);
 const recipients=await contextPage(read,workspace,actor,"recipients","apply");assert.ok(recipients.rows.some(row=>"user" in row&&row.user==="sample-member"));
 const ceilings=await contextPage(read,workspace,actor,"ceilings","apply");assert.equal(ceilings.rows.length,1);assert.ok("not_before" in ceilings.rows[0]&&ceilings.rows[0].not_before.endsWith(".5Z"));
 const detail=inspectionOf(convert(wire.inspection),"client-human");assert.equal(detail.row.state,"pending");assert.equal(detail.notBefore,"2026-11-16T09:00:00.5Z");
});
