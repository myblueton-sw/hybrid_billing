import test from "node:test";
import assert from "node:assert/strict";
import { loadAdvice } from "./advice";
import { loadHistory } from "./history";
import { permission, type Preparation } from "./contract";
const hash = "a".repeat(64), at = "2026-11-22T00:00:00Z";
const scope: Preparation = { preparation: "prep", preparation_digest: hash, document: "doc", document_digest: hash, connection: "erp", customer: "customer", period: "2026-10", currency: "USD", preparation_actor: "actor" };
const envelope = (fields: Record<string, string>) => ({ revision: "r1", at, fields });
const allowed = { state: "allowed", code: "" }, ready = { state: "ready", code: "" }, inapplicable = { state: "inapplicable", code: "prerequisite_failed" };
function advice() {
 return { mode: "synthetic", result: "result", digest: hash, revision: "r1", permissions: JSON.stringify({ confirm: allowed, revoke: allowed, apply: allowed }), readiness: JSON.stringify({ confirm: inapplicable, revoke: ready, apply: inapplicable, recover: ready }), application_state: "unknown", applying_actor: "actor", application_receipt: "", application_context: JSON.stringify({kind:"initial",position:"current",predecessor:"",predecessor_digest:"",predecessor_revision:0,predecessor_receipt:"",current_result:"result",current_state:"unknown"}), current_eligibility: "advisory_only" };
}
test("advice retains unknown ownership and independently checked recovery readiness", async () => {
 const got = await loadAdvice(async () => envelope(advice()), "result", hash);
 assert.equal(got.applicationState, "unknown"); assert.equal(got.applyingActor, "actor");
 assert.equal(got.readiness.recover.state, "ready"); assert.ok(Object.isFrozen(got.readiness));
});
test("advice rejects identity revision permission and application contradictions", async () => {
 for (const override of [
  { result: "other" }, { digest: "b".repeat(64) }, { revision: "r2" }, { applying_actor: "" },
  { application_receipt: hash }, { application_state: "not_started", applying_actor: "" },
  { permissions: JSON.stringify({ confirm: allowed, revoke: allowed, apply: { state: "denied", code: "permission_denied" } }) },
 ]) await assert.rejects(loadAdvice(async () => envelope({ ...advice(), ...override }), "result", hash));
 assert.throws(() => permission({ state: "denied", code: ["permission_denied"] }));
});
test("result history preserves revoked unknown evidence and refuses cross-preparation data", async () => {
 const row = { mode: "synthetic", result: "result", digest: hash, recorded_state: "revoked", receipt: "historical_decision", application_state: "unknown", application_receipt: "", proposed_by: "actor", proposed_at: at };
 const fields = { mode: "synthetic", preparation: "prep", digest: hash, current_eligibility: "unchecked", snapshot: "r1", next: "", row_00: JSON.stringify(row) };
 const got = await loadHistory(async () => envelope(fields), scope);
 assert.equal(got.rows[0].recorded_state, "revoked"); assert.equal(got.rows[0].application_state, "unknown");
 for (const override of [{ preparation: "other" }, { digest: "b".repeat(64) }, { snapshot: "r2" }, { row_00: JSON.stringify({ ...row, application_receipt: hash }) }, { row_00: JSON.stringify({ ...row, recorded_state: "pending" }) }, { row_00: JSON.stringify({ ...row, recorded_state: ["pending"], application_state: "recorded", application_receipt: hash }) }]) await assert.rejects(loadHistory(async () => envelope({ ...fields, ...override }), scope));
});

test("application context strictly distinguishes prospective replacement and retained history", async () => {
 const {applicationContext}=await import("./relation");
 const initial={kind:"initial",position:"not_started",predecessor:"",predecessor_digest:"",predecessor_revision:0,predecessor_receipt:"",current_result:"",current_state:"none"};
 assert.equal(applicationContext(JSON.stringify(initial),"next","not_started").kind,"initial");
 const replacement={kind:"replacement",position:"not_started",predecessor:"previous",predecessor_digest:hash,predecessor_revision:2,predecessor_receipt:hash,current_result:"previous",current_state:"recorded"};
 assert.equal(applicationContext(JSON.stringify(replacement),"next","not_started").predecessor,"previous");
 assert.equal(applicationContext(JSON.stringify({...replacement,predecessor_receipt:"",current_state:"unknown"}),"next","not_started").current_state,"unknown");
 const current={...replacement,position:"current",current_result:"next",current_state:"unknown"};
 assert.equal(applicationContext(JSON.stringify(current),"next","unknown").position,"current");
 const historical={...replacement,position:"historical",current_result:"newest",current_state:"unknown"};
 assert.equal(applicationContext(JSON.stringify(historical),"next","recorded").position,"historical");
 for(const change of [{kind:["replacement"]},{position:["historical"]},{current_state:["unknown"]},{predecessor_receipt:""},{current_result:"next"},{current_result:"previous"},{current_state:"none"},{predecessor_revision:0},{predecessor:"next"},{predecessor_digest:"bad"},{extra:"unexpected"}])assert.throws(()=>applicationContext(JSON.stringify({...historical,...change}),"next","recorded"));
 assert.throws(()=>applicationContext(JSON.stringify(historical),"next","unknown"));
 assert.throws(()=>applicationContext(JSON.stringify({...current,predecessor_receipt:""}),"next","unknown"));
});
