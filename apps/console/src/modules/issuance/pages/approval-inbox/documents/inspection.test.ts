import test from "node:test";
import assert from "node:assert/strict";
import { adviceOf, inspectionOf, preparationInspectionOf } from "./inspection";
import { DocumentError, type ReadEnvelope } from "./contract";
const hash = "a".repeat(64), at = "2026-11-21T00:00:00.123456789Z";
const scope = { request: "request", digest: hash, workspace: "workspace" };
const row = { document: "document", digest: hash, recorded_state: "pending" as const, proposed_by: "proposer", proposed_at: at };
const prep = { preparation: "preparation", document: "document", document_digest: hash, actor: "proposer", recorded_receipt: "" };
const envelope = (fields: Record<string, string>): ReadEnvelope => ({ revision: "r7", at, fields });
function basis() {
 return { ID: "document", Mode: "synthetic", Profile: "synthetic_individual_net_v1", Evidence: "evidence", Workspace: "workspace", Request: "request", RequestDigest: hash, Candidate: "candidate", CandidateDigest: hash, Path: "direct", Document: { WorkspaceID: "workspace", IssuerLegalEntityID: "issuer", BuyerLegalEntityID: "buyer", PayerID: "payer", Currency: "USD", Path: "external_invoice" }, Connection: { Connection: { ID: "erp", Entity: "issuer", System: "system", Transport: "api", Capabilities: ["prepare"], Currency: "USD", Scale: 2, Interval: 1, Owner: "owner", Template: "" }, Author: "author", At: at, Verified: false, Evidence: "", Verifier: "" }, Mappings: ["customer", "product", "tax"].map(kind => ({ Mapping: { Connection: "erp", Kind: kind, Platform: `${kind}:value`, External: "external" }, Revision: 1, Author: "author", At: at })), Survey: { Survey: { Connection: "erp", Customer: "customer:value", Period: "2026-10", Outcome: "complete", Evidence: "evidence" }, Author: "author", At: at }, ExpectedNet: "15", ExpectedTax: "1", Lines: [{ Line: "line", Rule: "rule", Kind: "rated", Product: "product:value", Quantity: "2", Unit: "unit", TaxCode: "tax:value", Tax: "1", Evidence: "evidence" }] };
}
function document() { return { mode: "synthetic", document: "document", digest: hash, recorded_state: "pending", receipt: "historical_evidence", basis: JSON.stringify(basis()), basis_digest: hash, current_eligibility: "unchecked", proposed_by: "proposer", proposal_reason: "Explicit synthetic evidence", proposed_at: at }; }
function advice() {
 const permissions = Object.fromEntries(["propose", "confirm", "revoke", "prepare"].map(k => [k, { state: "allowed", code: "" }]));
 const readiness = { confirm: { state: "ready", code: "" }, revoke: { state: "ready", code: "" }, prepare: { state: "inapplicable", code: "prerequisite_failed" } };
 return { mode: "synthetic", document: "document", digest: hash, current_eligibility: "advisory", permissions: JSON.stringify(permissions), readiness: JSON.stringify(readiness), preparation_scope: "packet_only_key_capacity_and_source_coverage_unchecked" };
}
function preparation() {
 return { mode: "synthetic", preparation: "preparation", document_digest: hash, recorded_receipt: "", observed_digest: hash, ledger_observation: "matching_prepared", current_eligibility: "unchecked", actor: "proposer", submission: JSON.stringify({ Key: "preparation", Connection: "erp", Purpose: "customer_invoice", Customer: "customer:value", Period: "2026-10", Path: "direct", Anchor: "", Lines: [{ ID: `line-${hash}`, Source: `source-${hash}`, Product: "product:value", TaxCode: "tax:value", Quantity: "2", Net: "15", Tax: "1" }] }), total: JSON.stringify({ Net: "15.00", Tax: "1.00", Quantity: "2.00" }), mappings: JSON.stringify(["customer:customer:value:external:1", "product:product:value:external:1", "tax:tax:value:external:1"]) };
}
test("immutable historical inspections do not assert current eligibility or settle an observation", () => {
 const got = inspectionOf(envelope(document()), scope, row);
 assert.equal(got.basis.ExpectedNet, "15"); assert.ok(Object.isFrozen(got.basis.Lines[0]));
 assert.equal(adviceOf(envelope(advice()), row).readiness.confirm.state, "ready");
 const observed = preparationInspectionOf(envelope(preparation()), prep);
 assert.equal(observed.row.recorded_receipt, ""); assert.equal(observed.observedDigest, hash);
});
for (const change of ["workspace", "request", "digest", "document", "currency", "tax", "mapping-kind-collision", "mapping-duplicate", "line-duplicate", "quantity", "foreign-connection"]) test(`basis refuses ${change}`, () => {
 const b = basis();
 if (change === "workspace") b.Workspace = "foreign";
 if (change === "request") b.Request = "foreign";
 if (change === "digest") b.RequestDigest = "b".repeat(64);
 if (change === "document") b.ID = "foreign";
 if (change === "currency") b.Connection.Connection.Currency = "EUR";
 if (change === "tax") b.ExpectedTax = "2";
 if (change === "mapping-kind-collision") { b.Mappings[0].Mapping.Kind = "customer:customer"; b.Mappings[0].Mapping.Platform = "value"; }
 if (change === "mapping-duplicate") b.Mappings[1] = b.Mappings[0];
 if (change === "line-duplicate") b.Lines.push(b.Lines[0]);
 if (change === "quantity") b.Lines[0].Quantity = "0";
 if (change === "foreign-connection") b.Survey.Survey.Connection = "foreign";
 assert.throws(() => inspectionOf(envelope({ ...document(), basis: JSON.stringify(b) }), scope, row), DocumentError);
});
for (const change of ["self", "nanosecond", "missing-event", "incomplete-event", "pending-event", "revocation-order"]) test(`inspection refuses ${change}`, () => {
 const f: Record<string,string> = { ...document(), recorded_state: "confirmed", confirmation_by: "confirmer", confirmation_reason: "Independent confirmation", confirmation_at: at };
 if (change === "self") f.confirmation_by = "proposer";
 if (change === "nanosecond") f.confirmation_at = at.replace("789", "788");
 if (change === "missing-event") for (const k of ["confirmation_by", "confirmation_reason", "confirmation_at"]) delete f[k];
 if (change === "incomplete-event") delete f.confirmation_reason;
 if (change === "pending-event") f.recorded_state = "pending";
 if (change === "revocation-order") { f.recorded_state = "revoked"; f.revocation_by = "revoker"; f.revocation_reason = "Revoke"; f.revocation_at = at; f.confirmation_at = at.replace("789", "790"); }
 assert.throws(() => inspectionOf(envelope(f), scope, row), DocumentError);
});
test("equivalent fractional timestamps and newer historical lifecycle are accepted", () => {
 const proposed_at = "2026-11-21T00:00:00.1Z";
 const f = { ...document(), proposed_at, recorded_state: "confirmed", confirmation_by: "confirmer", confirmation_reason: "Confirm", confirmation_at: "2026-11-21T00:00:00.100000000Z" };
 assert.equal(inspectionOf(envelope(f), scope, { ...row, proposed_at }).row.recorded_state, "confirmed");
});
for (const change of ["foreign", "denied-ready", "unchecked-allowed", "unknown-code", "scope"]) test(`advice refuses ${change}`, () => {
 const f = advice(), permissions = JSON.parse(f.permissions), readiness = JSON.parse(f.readiness);
 if (change === "foreign") f.document = "foreign";
 if (change === "denied-ready") permissions.confirm = { state: "denied", code: "permission_denied" };
 if (change === "unchecked-allowed") readiness.confirm = { state: "unchecked", code: "" };
 if (change === "unknown-code") readiness.confirm = { state: "blocked", code: "anything" };
 if (change === "scope") f.preparation_scope = "ready";
 f.permissions = JSON.stringify(permissions); f.readiness = JSON.stringify(readiness);
 assert.throws(() => adviceOf(envelope(f), row), DocumentError);
});
for (const change of ["actor", "key", "document", "receipt", "observation", "total", "source", "duplicate", "negative", "mapping-duplicate"]) test(`preparation observation refuses ${change}`, () => {
 const f = preparation(), sub = JSON.parse(f.submission);
 if (change === "actor") f.actor = "foreign";
 if (change === "key") sub.Key = "foreign";
 if (change === "document") f.document_digest = "b".repeat(64);
 if (change === "receipt") f.recorded_receipt = "b".repeat(64);
 if (change === "observation") f.ledger_observation = "missing";
 if (change === "total") f.total = JSON.stringify({ Net: "16.00", Tax: "1.00", Quantity: "2.00" });
 if (change === "source") sub.Lines[0].Source = `source-${"b".repeat(64)}`;
 if (change === "duplicate") sub.Lines.push(sub.Lines[0]);
 if (change === "negative") sub.Lines[0].Tax = "-1";
 if (change === "mapping-duplicate") f.mappings = '["same","same"]';
 f.submission = JSON.stringify(sub);
 assert.throws(() => preparationInspectionOf(envelope(f), prep), DocumentError);
});
test("observation preserves unresolved status and cannot erase a recorded receipt", () => {
 for (const ledger_observation of ["missing", "unavailable", "conflict"]) {
  const got = preparationInspectionOf(envelope({ ...preparation(), ledger_observation, observed_digest: "" }), prep);
  assert.equal(got.row.recorded_receipt, ""); assert.equal(got.observedDigest, "");
 }
 const f = { ...preparation(), recorded_receipt: hash };
 assert.equal(preparationInspectionOf(envelope(f), prep).row.recorded_receipt, hash);
 assert.throws(() => preparationInspectionOf(envelope({ ...f, recorded_receipt: "", observed_digest: "", ledger_observation: "missing" }), { ...prep, recorded_receipt: hash }), DocumentError);
});
