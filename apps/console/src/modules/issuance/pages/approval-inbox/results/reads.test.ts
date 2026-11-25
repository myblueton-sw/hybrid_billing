import test from "node:test";
import assert from "node:assert/strict";
import { loadContext, loadPreparations } from "./reads";
import type { Preparation, ReadEnvelope } from "./contract";

const hash = "a".repeat(64);
const preparation = (id: string): Preparation => ({ preparation: id, preparation_digest: hash, document: "document", document_digest: hash, connection: "erp", customer: "customer", period: "2026-10", currency: "USD", preparation_actor: "actor" });
const envelope = (fields: Record<string, string>): { revision: string; at: string; fields: Record<string,string> } => ({ revision: "r1", at: "2026-11-22T00:00:00Z", fields });
function discovery(start: number, count: number, next: string) {
 const fields: Record<string, string> = { mode: "synthetic", workspace: "workspace", current_eligibility: "unchecked", snapshot: "r1", next };
 for (let i = 0; i < count; i++) fields[`row_${String(i).padStart(2, "0")}`] = JSON.stringify(preparation(`prep-${String(start + i).padStart(2, "0")}`));
 return envelope(fields);
}
function context(selected = preparation("prep-00")) {
 const { preparation_actor: _, ...identity } = selected;
 return envelope({ ...identity, mode: "synthetic", profile: "synthetic_result_one_to_one_v1", schema: "hb_erp_result_packet_v1", current_eligibility: "unchecked", snapshot: "r1", next: "", permissions: JSON.stringify({ propose: { state: "allowed", code: "" } }), proposal_readiness: JSON.stringify({ state: "ready", code: "" }), row_00: JSON.stringify({ line: "line", expected_product: "reference-product", expected_unit: "hour" }) });
}

test("result preparation discovery follows exact bounded continuation", async () => {
 const calls: unknown[] = [];
 const got = await loadPreparations(async (command, target, fields) => { calls.push({ command, target, fields }); return calls.length === 1 ? discovery(0, 10, "prep-09") : discovery(10, 2, ""); }, "workspace");
 assert.equal(got.rows.length, 12);
 assert.deepEqual(calls[1], { command: "result_evidence.preparations", target: "workspace", fields: { after: "prep-09", snapshot: "r1" } });
 assert.ok(Object.isFrozen(got.rows[0]));
});
test("result discovery rejects changed scope, stale pages and malformed rows", async () => {
 for (const mode of ["workspace", "revision", "repeat", "unknown", "gap", "extra-result"]) {
  let calls = 0;
  await assert.rejects(loadPreparations(async () => {
   calls++; if (calls === 1) return discovery(0, 10, "prep-09");
   const p = structuredClone(discovery(10, 1, ""));
   if (mode === "workspace") p.fields.workspace = "other";
   if (mode === "revision") p.revision = "r2";
   if (mode === "repeat") p.fields.row_00 = JSON.stringify(preparation("prep-09"));
   if (mode === "unknown") p.fields.extra = "unexpected";
   if (mode === "gap") { p.fields.row_01 = p.fields.row_00; delete p.fields.row_00; }
   if (mode === "extra-result") p.fields.row_00 = JSON.stringify({ ...preparation("prep-10"), reported_net: "15" });
   return p;
  }, "workspace"), /invalid_result_response/);
 }
});
test("result context preserves reference-only facts and independent permissions", async () => {
 const got = await loadContext(async () => context(), preparation("prep-00"));
 assert.deepEqual({ ...got.references[0] }, { line: "line", expected_product: "reference-product", expected_unit: "hour" });
 assert.equal(got.permission.state, "allowed"); assert.equal(got.readiness.state, "ready");
 assert.ok(!JSON.stringify(got).includes("reported_net"));
 for (const mode of ["digest", "currency", "permission", "reported", "duplicate-json"]) {
  const p = structuredClone(context());
  if (mode === "digest") p.fields.preparation_digest = "b".repeat(64);
  if (mode === "currency") p.fields.currency = "KRW";
  if (mode === "permission") p.fields.permissions = JSON.stringify({ propose: { state: "denied", code: "permission_denied" } });
  if (mode === "reported") p.fields.row_00 = JSON.stringify({ line: "line", expected_product: "reference-product", expected_unit: "hour", Net: "15" });
  if (mode === "duplicate-json") p.fields.permissions = '{"propose":{"state":"allowed","state":"denied","code":""}}';
  await assert.rejects(loadContext(async () => p, preparation("prep-00")), /invalid_result_response/);
 }
});
