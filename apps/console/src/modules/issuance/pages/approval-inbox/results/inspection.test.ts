import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { inspectWithAdvice } from "./inspection";
import { loadPreparations, loadContext } from "./reads";
import { loadHistory } from "./history";
import type { ReadCommand } from "./contract";
const path=process.env.HB_RESULT_CLIENT_FIXTURE;
test("actual server synthetic result replies pass discovery context history inspection and advice",{skip:!path},async()=>{
 const fixture=JSON.parse(readFileSync(path!,"utf8"));
 const commands:Record<string,string>={"result_evidence.preparations":"preparations","result_evidence.context":"context","result_evidence.history":"history","result_evidence.inspect":"inspection","result_evidence.advice":"advice"};
 const read:ReadCommand=async(command)=>{const p=fixture[commands[command]];return {revision:p.result_revision,at:p.as_of,fields:p.fields};};
 const list=await loadPreparations(read,fixture.preparations.fields.workspace);assert.equal(list.rows.length,1);
 const context=await loadContext(read,list.rows[0]);assert.ok(context.references.length>0);
 const history=await loadHistory(read,list.rows[0]);assert.ok(history.rows.length>0);
 const value=await inspectWithAdvice(read,list.rows[0],history.rows[0],fixture.preparations.fields.workspace);assert.equal(value.inspection.state,"confirmed");assert.equal(value.inspection.applicationState,"not_started");
 for(const key of ["source_digest","document_digest"]){const original=fixture.inspection.fields[key];fixture.inspection.fields[key]="f".repeat(64);await assert.rejects(inspectWithAdvice(read,list.rows[0],history.rows[0],fixture.preparations.fields.workspace));fixture.inspection.fields[key]=original;}
 for(const state of ["unknown","recorded"]){
  const original={...fixture.inspection.fields};
  fixture.inspection.fields={...original,recorded_state:"revoked",revocation_by:"revoker",revocation_reason:"Synthetic revocation",revocation_at:original.confirmation_at,application_state:state,application_receipt:state==="recorded"?"b".repeat(64):""};
  for(const key of ["confirmation_by","confirmation_reason","confirmation_at"])delete fixture.inspection.fields[key];
  await assert.rejects(inspectWithAdvice(read,list.rows[0],history.rows[0],fixture.preparations.fields.workspace));fixture.inspection.fields=original;
 }
 fixture.advice.result_revision="r999";fixture.advice.fields.revision="r999";await assert.rejects(inspectWithAdvice(read,list.rows[0],history.rows[0],fixture.preparations.fields.workspace));
});
