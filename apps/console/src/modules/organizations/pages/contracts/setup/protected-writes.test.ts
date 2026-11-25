import assert from "node:assert/strict";
import test from "node:test";
import { SetupError } from "./context";
import { protectedWriteClient, protectedWriteIntent, protectedWriteRules, type ProtectedWriteIntent } from "./protected-writes";

const hash = "a".repeat(64), other = "b".repeat(64);
const scope = { contract: "contract-1", configuration: "config-1", digest: hash };
const session = { user: "writer", workspace: "workspace", csrf: "synthetic-csrf", role: "member", login: "writer" };
const args: Record<string, Record<string, string>> = {
  propose_protected_rules: { rules: "rules-1", approvals: "1", validity_seconds: "172800", max_lead_seconds: "604800", channel_age_seconds: "86400" },
  confirm_protected_rules: { rules: "rules-1", rules_digest: hash },
  register_channel: { channel: "channel-1", subject_kind: "customer", kind: "email", reference: "synthetic_channel_customer_a", attestation: "claim-1" },
  admit_channel: { channel: "channel-1", channel_digest: hash, attestation: "claim-2" },
  request_change: { change: "change-1", field: "payment_account", reference: "synthetic_account_a", channel: "channel-1", channel_digest: hash, provenance: "staff_entry", instruction: "in_person", effective: "2026-11-24T01:00:00Z", attestation: "claim-3" },
  verify_change: { change: "change-1", change_digest: hash, outcome: "confirmed", attestation: "claim-4" },
  decide_change: { change: "change-1", change_digest: hash, verdict: "approve" },
};
const intent = (operation: string, change: Record<string, string | undefined> = {}, extra: Partial<ProtectedWriteIntent> = {}): ProtectedWriteIntent => {
  const merged: Record<string, string> = {};
  for (const [k, v] of Object.entries({ ...args[operation], ...change })) if (v !== undefined) merged[k] = v;
  return { operation, scope, args: merged, rationale: protectedWriteRules[operation]?.rationale ? "Synthetic protected setup" : "", actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision: "r4", attempt: { request: "request-1", key: "key-1" }, ...extra };
};
const rejects = (run: () => unknown) => assert.throws(run, (e: unknown) => e instanceof SetupError && e.code === "invalid_setup_request");

test("every command has a valid intent and freezes its arguments", () => {
  assert.equal(Object.keys(protectedWriteRules).length, 7);
  for (const operation of Object.keys(args)) {
    const frozen = protectedWriteIntent(intent(operation));
    assert.ok(Object.isFrozen(frozen.args)); assert.equal(frozen.operation, operation);
  }
  rejects(() => protectedWriteIntent(intent("withdraw_change")));
  rejects(() => protectedWriteIntent(intent("admit_channel", {}, { rationale: "not taken" })));
  rejects(() => protectedWriteIntent(intent("register_channel", {}, { rationale: "" })));
  rejects(() => protectedWriteIntent(intent("register_channel", {}, { revision: "" })));
});

test("closed values and caller-supplied identifiers are refused before any request", () => {
  const bad: [string, Record<string, string | undefined>][] = [
    ["propose_protected_rules", { approvals: "4" }], ["propose_protected_rules", { approvals: "01" }], ["propose_protected_rules", { channel_age_seconds: "3600" }], ["propose_protected_rules", { validity_seconds: "60" }],
    ["confirm_protected_rules", { rules_digest: "short" }],
    ["register_channel", { subject_kind: "tenant" }], ["register_channel", { kind: "fax" }], ["register_channel", { reference: "synthetic_channel_entity_a" }], ["register_channel", { reference: "vault:contact" }], ["register_channel", { fingerprint: hash }],
    ["admit_channel", { channel_digest: undefined }],
    ["request_change", { field: "tax_identifier" }], ["request_change", { provenance: "assistant_generated" }], ["request_change", { instruction: "email" }], ["request_change", { instruction_channel: "channel-2" }],
    ["request_change", { instruction: "registered_channel" }], ["request_change", { instruction: "registered_channel", instruction_channel: "channel-1" }], ["request_change", { effective: "tomorrow" }], ["request_change", { sender: hash }],
    ["verify_change", { outcome: "maybe" }], ["decide_change", { verdict: "approve", reason: "other" }], ["decide_change", { verdict: "reject" }], ["decide_change", { verdict: "reject", reason: "dislike" }],
  ];
  for (const [operation, change] of bad) rejects(() => protectedWriteIntent(intent(operation, change)));
  assert.equal(protectedWriteIntent(intent("request_change", { instruction: "registered_channel", instruction_channel: "channel-2" })).args.instruction_channel, "channel-2");
  assert.equal(protectedWriteIntent(intent("request_change", { instruction: "unregistered_sender" })).args.instruction, "unregistered_sender");
  assert.equal(protectedWriteIntent(intent("decide_change", { verdict: "reject", reason: "missing_evidence" })).args.reason, "missing_evidence");
});

const receipts: Record<string, [string, Record<string, string>]> = {
  propose_protected_rules: ["proposed", { schema: "hb_issuance_protected_rules_receipt_v1", rules: "rules-1", rules_digest: hash }],
  confirm_protected_rules: ["confirmed", { schema: "hb_issuance_protected_rules_receipt_v1", rules: "rules-1", rules_digest: hash }],
  register_channel: ["registered", { schema: "hb_issuance_protected_channel_receipt_v1", channel: "channel-1", channel_digest: hash }],
  admit_channel: ["admitted", { schema: "hb_issuance_protected_channel_receipt_v1", channel: "channel-1", channel_digest: hash }],
  request_change: ["requested", { schema: "hb_issuance_protected_change_receipt_v1", change: "change-1", change_digest: hash, phase: "recorded" }],
  verify_change: ["confirmed", { schema: "hb_issuance_protected_change_receipt_v1", change: "change-1", change_digest: hash, phase: "recorded" }],
  decide_change: ["scheduled", { schema: "hb_issuance_protected_change_receipt_v1", change: "change-1", change_digest: hash, phase: "recorded" }],
};
function server(change: (raw: Record<string, unknown>) => void = () => {}, status = 200) {
  const requests: { url: string; body: Record<string, unknown>; raw: string }[] = [];
  const fetcher: typeof fetch = async (input, init) => {
    const url = String(input);
    if (url === "/api/v1/session") return Response.json({ user: session.user, workspace: session.workspace, csrf_token: session.csrf });
    const body = JSON.parse(String(init?.body)); requests.push({ url, body, raw: String(init?.body) });
    const [state, fields] = receipts[url.split("issuance_setup.")[1]];
    const raw: Record<string, unknown> = { request_id: body.request_id, status: "completed", operation_id: "operation-1", target_id: body.target_id, result_revision: "r5", business_state: state, replay: false, as_of: "2026-11-23T00:00:00Z",
      fields: { contract: scope.contract, configuration: scope.configuration, current_eligibility: "unchecked", state, ...fields } };
    change(raw); return Response.json(raw, { status });
  };
  return { fetcher, requests };
}

test("each command sends its exact envelope and binds its receipt", async () => {
  for (const operation of Object.keys(args)) {
    const s = server(), receipt = await protectedWriteClient(session, s.fetcher)(intent(operation));
    assert.equal(s.requests[0].url, `/api/v1/commands/issuance_setup.${operation}`);
    assert.deepEqual(s.requests[0].body.body, { configuration: "config-1", digest: hash, ...args[operation] });
    assert.equal(Object.hasOwn(s.requests[0].body, "rationale"), protectedWriteRules[operation].rationale);
    assert.equal(s.requests[0].body.target_id, "contract-1"); assert.equal(s.requests[0].raw.includes(session.csrf), false);
    assert.equal(receipt.state, receipts[operation][0]); assert.equal(receipt.digest, hash); assert.ok(Object.isFrozen(receipt));
  }
  const s = server(), client = protectedWriteClient(session, s.fetcher), frozen = protectedWriteIntent(intent("request_change"));
  await client(frozen); await client(frozen);
  assert.equal(s.requests[0].raw, s.requests[1].raw);
  const repeated = await protectedWriteClient(session, server(r => { r.business_state = "confirmed"; r.fields = { ...r.fields as object, state: "confirmed" }; }).fetcher)(intent("propose_protected_rules"));
  assert.equal(repeated.state, "confirmed");
  const lapsed = await protectedWriteClient(session, server(r => { r.business_state = "lapsed"; r.fields = { ...r.fields as object, state: "lapsed" }; }).fetcher)(intent("verify_change"));
  assert.equal(lapsed.state, "lapsed");
});

test("misbound receipts stay uncertain and only documented refusals are clean", async () => {
  const uncertain = (e: unknown) => e instanceof SetupError && e.effect === "unknown" && e.code === "invalid_setup_receipt";
  const changes: [string, (r: Record<string, unknown>) => void][] = [
    ["admit_channel", r => { r.target_id = "other"; }], ["admit_channel", r => { r.fields = { ...r.fields as object, channel: "channel-9" }; }],
    ["admit_channel", r => { r.fields = { ...r.fields as object, channel_digest: other }; }], ["admit_channel", r => { r.business_state = "registered"; }],
    ["verify_change", r => { r.fields = { ...r.fields as object, configuration: "config-2" }; }], ["verify_change", r => { r.fields = { ...r.fields as object, current_eligibility: "eligible" }; }],
    ["verify_change", r => { r.fields = { ...r.fields as object, reference: "vault:account" }; }], ["verify_change", r => { r.fields = { ...r.fields as object, phase: "done" }; }],
    ["verify_change", r => { r.business_state = "requested"; r.fields = { ...r.fields as object, state: "requested" }; }], ["register_channel", r => { r.fields = { ...r.fields as object, schema: "hb_issuance_protected_change_receipt_v1" }; }],
  ];
  for (const [operation, change] of changes) await assert.rejects(protectedWriteClient(session, server(change).fetcher)(intent(operation)), uncertain);
  for (const operation of [false, true]) {
    const s = server(r => { const request = r.request_id; for (const k of Object.keys(r)) delete r[k]; Object.assign(r, { request_id: request, code: "self_approval_denied", next_action: "none", retry: false, ...(operation ? { operation_id: "operation-1" } : {}) }); }, 403);
    await assert.rejects(protectedWriteClient(session, s.fetcher)(intent("decide_change")), (e: unknown) => e instanceof SetupError && e.code === "self_approval_denied" && e.effect === (operation ? "unknown" : "none"));
  }
  const s = server();
  await assert.rejects(protectedWriteClient(session, s.fetcher)(intent("decide_change", {}, { actor: "other" })), (e: unknown) => e instanceof SetupError && e.code === "session_changed");
  assert.equal(s.requests.length, 0);
  await assert.rejects(protectedWriteClient(session, async input => { if (String(input) === "/api/v1/session") return Response.json({ user: session.user, workspace: session.workspace, csrf_token: session.csrf }); throw new TypeError("network"); })(intent("decide_change")), (e: unknown) => e instanceof SetupError && e.effect === "unknown");
});
