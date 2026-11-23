import test from "node:test";
import assert from "node:assert/strict";
import { configurationOf, configurationsOf } from "./history";
import { SetupError } from "./context";
const hash="a".repeat(64);
function row(){return {configuration:"config-01",digest:hash,state:"proposal_unknown",module_observation:"absent",revision:"1",contract_revision:"9007199254740993",billing_profile:"profile",billing_revision:"1",mode:"erp_issuance",profile:"synthetic_erp_document_v1",region:"KR",tax:"retained-tax",zone:"Asia/Seoul",languages:"ar,en,ko",proposer:"first",confirmer:"",setup_on:"2026-11-16",scope_digest:hash,proposal_reason:"Synthetic reason"};}
test("retained nonpreset unknown settings are preserved, never promoted by module observation",()=>{
 const r=row();assert.equal(configurationOf(r).tax,"retained-tax");
 const proposed={...r,proposed_at:"2026-11-16T01:00:00Z",module_observation:"proposed"};
 assert.equal(configurationOf(proposed).state,"proposal_unknown");
 const active={...proposed,state:"confirmation_unknown",confirmer:"second",approval_observed_at:"2026-11-16T02:00:00Z",module_observation:"active"};
 assert.equal(configurationOf(active).state,"confirmation_unknown");
 assert.throws(()=>configurationOf({...active,activated_at:active.approval_observed_at}),SetupError);
 assert.equal(configurationOf({...active,state:"confirmed",activated_at:active.approval_observed_at}).state,"confirmed");
});
test("legacy rows are readable but partial setup and false states are rejected",()=>{
 const r:Record<string,string>=row();delete r.setup_on;delete r.scope_digest;delete r.proposal_reason;
 assert.equal(configurationOf(r).setup_on,undefined);
 for(const change of [{setup_on:"2026-11-16"},{state:"confirmed"},{module_observation:"active"},{languages:"en,ar"},{confirmer:"first"}])assert.throws(()=>configurationOf({...r,...change}),SetupError);
});
test("approval ordering preserves nanoseconds and equivalent fractional widths",()=>{
 const r={...row(),state:"confirmation_unknown",confirmer:"second",module_observation:"active",proposed_at:"2026-11-16T01:00:00.000000002Z",approval_observed_at:"2026-11-16T01:00:00.000000001Z"};
 assert.throws(()=>configurationOf(r),SetupError);
 assert.equal(configurationOf({...r,approval_observed_at:r.proposed_at}).state,"confirmation_unknown");
 assert.equal(configurationOf({...r,proposed_at:"2026-11-16T01:00:00.1Z",approval_observed_at:"2026-11-16T01:00:00.100000000Z"}).state,"confirmation_unknown");
 assert.equal(configurationOf({...r,proposed_at:"2026-11-16T01:00:00Z",approval_observed_at:"2026-11-16T01:00:00.000000001Z"}).state,"confirmation_unknown");
});
test("history rejects hidden rows, stale snapshots and out of order continuation",()=>{
 const e={revision:"r1",at:"2026-11-16T00:00:00Z",fields:{schema:"hb_issuance_configuration_page_v1",contract:"contract",current_eligibility:"unchecked",snapshot:hash,count:"1",next:"",row_00:JSON.stringify(row())}};
 assert.equal(configurationsOf(e,"contract").rows[0].contract_revision,"9007199254740993");
 assert.throws(()=>configurationsOf(e,"other"),SetupError);
 assert.throws(()=>configurationsOf(e,"contract","config-01",hash),SetupError);
 assert.throws(()=>configurationsOf(e,"contract","config-00","b".repeat(64)),SetupError);
 assert.throws(()=>configurationsOf({...e,fields:{...e.fields,count:"11"}},"contract"),SetupError);
 const missing:Record<string,string>={...e.fields};delete missing.row_00;
 assert.throws(()=>configurationsOf({...e,fields:missing},"contract"),SetupError);
});
