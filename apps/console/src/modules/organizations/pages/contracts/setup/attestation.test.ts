import assert from "node:assert/strict";
import test from "node:test";
import { attestationClient, attestationIntent, attestationOf, attestationsOf, attestationText, referencesOf, type AttestationIntent } from "./attestation";
import { setupReadClient } from "./client";
import { SetupError } from "./context";

const hash = "a".repeat(64), at = "2026-11-23T01:00:00.000000001Z";
const scope = { contract: "contract-1", configuration: "config-1", digest: hash };
const session = { user: "writer", workspace: "workspace", csrf: "synthetic-csrf", role: "member", login: "writer" };
const head = { contract: scope.contract, configuration: scope.configuration, digest: hash, current_eligibility: "unchecked" };
const row = (): Record<string, string> => ({ attestation: "claim-01", digest: hash, purpose: "channel_registration", target: "channel-01", actor: "writer", recorded_at: at, statement: "Taken from the signed record.", locator: "", state: "unconsumed", assurance: "human_attestation" });
const envelope = (fields: Record<string, string>) => ({ revision: "r1", at: "2026-11-23T00:00:00Z", fields });
const rejects = (run: () => unknown) => assert.throws(run, SetupError);
const intent = (change: Partial<AttestationIntent> = {}): AttestationIntent => ({ action: "attest", purpose: "channel_registration", scope, attestation: "claim-01", target: "channel-01", targetDigest: "", statement: "Taken from the signed record.", locator: "", actor: session.user, workspace: session.workspace, sessionCSRF: session.csrf, revision: "r4", attempt: { request: "request-1", key: "key-1" }, ...change });

test("statement text is normalized and refused before any request", () => {
  assert.equal(attestationText("one\r\ntwo\rthree", 2000, true), "one\ntwo\nthree");
  assert.equal(attestationText("\ttabbed", 2000, true), "\ttabbed");
  assert.equal(attestationText("", 512, false), "");
  for (const bad of [" \n", "a".repeat(2001), "broken \uFFFD", "nul\u0000", "escape\u001b", "c1\u0085", "lone\uD800"]) rejects(() => attestationText(bad, 2000, true));
  rejects(() => attestationText("한".repeat(171), 512, false));
});

test("rows keep the claim an accountable statement and never a verified fact", () => {
  assert.equal(attestationOf(row()).assurance, "human_attestation");
  assert.equal(attestationOf({ ...row(), state: "withdrawn" }).state, "withdrawn");
  assert.equal(attestationOf({ ...row(), statement: "\uFEFF" }).statement, "\uFEFF");
  assert.equal(attestationOf({ ...row(), statement: " " }).statement, " ");
  assert.equal(attestationOf({ ...row(), statement: "a".repeat(2000), locator: "b".repeat(512) }).locator.length, 512);
  for (const bad of [{ assurance: "verified" }, { state: "confirmed" }, { purpose: "change_approval" }, { statement: "" }, { statement: "a".repeat(2001) }, { digest: "short" }, { verified: "true" }, { recorded_at: "" }]) rejects(() => attestationOf({ ...row(), ...bad }));
});

test("attestation pages and references bind scope, order and closed values", () => {
  const page = (rows: Record<string, string>[], extra: Record<string, string> = {}) => envelope({ ...head, schema: "hb_issuance_setup_attestation_page_v1", snapshot: hash, count: String(rows.length), next: "", ...Object.fromEntries(rows.map((r, i) => [`row_${String(i).padStart(2, "0")}`, JSON.stringify(r)])), ...extra });
  assert.equal(attestationsOf(page([row(), { ...row(), attestation: "claim-02", state: "consumed" }]), scope).rows[1].state, "consumed");
  rejects(() => attestationsOf(page([{ ...row(), attestation: "claim-02" }, row()]), scope));
  rejects(() => attestationsOf(page([row()]), { ...scope, configuration: "config-2" }));
  rejects(() => attestationsOf(page([row()]), scope, "claim-00", "b".repeat(64)));
  rejects(() => attestationsOf(page([row()], { count: "11" }), scope));
  rejects(() => attestationsOf(page([row()], { other_provenance: "0" }), scope));
  assert.equal(attestationsOf(page([{ ...row(), attestation: "claim-11" }], { count: "11" }), scope, "claim-10", hash).count, 11);

  const entry = (code: string, use: string) => JSON.stringify({ code, use, applicability: "synthetic_only" });
  const references = (entries: string[], extra: Record<string, string> = {}) => envelope({ ...head, schema: "hb_issuance_setup_references_v1", ...Object.fromEntries(entries.map((v, i) => [`reference_${String(i).padStart(2, "0")}`, v])), ...extra });
  const got = referencesOf(references([entry("synthetic_account_a", "payment_account"), entry("synthetic_unregistered_sender", "unregistered_sender")]), scope);
  assert.deepEqual(got.entries.map(v => v.use), ["payment_account", "unregistered_sender"]); assert.ok(Object.isFrozen(got.entries));
  rejects(() => referencesOf(references([entry("synthetic_account_a", "payment_account"), entry("synthetic_account_a", "payment_account")]), scope));
  rejects(() => referencesOf(references([entry("synthetic_account_a", "tax_identifier")]), scope));
  rejects(() => referencesOf(references([JSON.stringify({ code: "synthetic_account_a", use: "payment_account", applicability: "synthetic_only", fingerprint: hash })]), scope));
  rejects(() => referencesOf(references([JSON.stringify({ code: "synthetic_account_a", use: "payment_account", applicability: "statutory" })]), scope));
  rejects(() => referencesOf(references([]), scope));
  rejects(() => referencesOf(references([entry("real_account_a", "payment_account")]), scope));
  const catalog: [string, string][] = [["synthetic_recipient_a", "document_recipient"], ["synthetic_recipient_b", "document_recipient"], ["synthetic_template_a", "document_template"], ["synthetic_template_b", "document_template"],
    ["synthetic_account_a", "payment_account"], ["synthetic_account_b", "payment_account"], ["synthetic_channel_customer_a", "channel_customer"], ["synthetic_channel_customer_b", "channel_customer"],
    ["synthetic_channel_document_kind_a", "channel_document_kind"], ["synthetic_channel_document_kind_b", "channel_document_kind"], ["synthetic_channel_entity_a", "channel_entity"], ["synthetic_channel_entity_b", "channel_entity"],
    ["synthetic_unregistered_sender", "unregistered_sender"]];
  assert.equal(referencesOf(references(catalog.map(([code, use]) => entry(code, use))), scope).entries.length, 13);
  const full = Array.from({ length: 10 }, (_, i) => ({ ...row(), attestation: `claim-${String(i + 1).padStart(2, "0")}` }));
  assert.equal(attestationsOf(page(full, { count: "12", next: "claim-10" }), scope).next, "claim-10");
  rejects(() => attestationsOf(page(full, { count: "12" }), scope));
});

test("intents require a bound target digest exactly where the target must exist", () => {
  assert.equal(attestationIntent(intent({ statement: "a\r\nb" })).statement, "a\nb");
  assert.equal(attestationIntent(intent({ purpose: "change_verification", targetDigest: hash })).targetDigest, hash);
  for (const bad of [{ targetDigest: hash }, { purpose: "channel_admission" as const }, { statement: "" }, { target: "" }, { revision: "" }, { purpose: "change_approval" as never }]) rejects(() => attestationIntent(intent(bad)));
  assert.equal(attestationIntent(intent({ action: "withdraw", target: "", statement: "" })).action, "withdraw");
  rejects(() => attestationIntent(intent({ action: "withdraw" })));
});

function server(change: (raw: Record<string, unknown>, body: Record<string, unknown>) => void = () => {}, status = 200) {
  const requests: { url: string; body: Record<string, unknown>; raw: string }[] = [];
  const fetcher: typeof fetch = async (input, init) => {
    const url = String(input);
    if (url === "/api/v1/session") return Response.json({ user: session.user, workspace: session.workspace, csrf_token: session.csrf });
    const body = JSON.parse(String(init?.body)); requests.push({ url, body, raw: String(init?.body) });
    const withdraw = url.includes(".withdraw_");
    const raw: Record<string, unknown> = { request_id: body.request_id, status: "completed", operation_id: "operation-1", target_id: body.target_id, result_revision: "r5", business_state: withdraw ? "withdrawn" : "recorded", replay: false, as_of: "2026-11-23T00:00:00Z",
      fields: { schema: "hb_issuance_setup_attestation_v1", contract: scope.contract, configuration: scope.configuration, attestation: body.body.attestation, purpose: url.split(/\.(?:attest|withdraw)_/)[1], target: body.body.target ?? "channel-01", actor: session.user, recorded_at: at, digest: hash, state: withdraw ? "withdrawn" : "unconsumed", assurance: "human_attestation" } };
    change(raw, body); return Response.json(raw, { status });
  };
  return { fetcher, requests };
}

test("writes send the exact command without a rationale and bind the receipt", async () => {
  const s = server(), client = attestationClient(session, s.fetcher);
  const receipt = await client(intent({ locator: "Binder 4" }));
  assert.equal(receipt.state, "unconsumed"); assert.equal(receipt.operation, "operation-1");
  assert.equal(s.requests[0].url, "/api/v1/commands/issuance_setup.attest_channel_registration");
  assert.deepEqual(s.requests[0].body, { request_id: "request-1", target_id: "contract-1", idempotency_key: "key-1", expected_revision: "r4", body: { configuration: "config-1", digest: hash, attestation: "claim-01", target: "channel-01", statement: "Taken from the signed record.", locator: "Binder 4" } });
  assert.equal(s.requests[0].raw.includes(session.csrf), false);
  await client(intent({ purpose: "change_verification", targetDigest: hash, target: "change-01" }));
  assert.equal(s.requests[1].body.body && (s.requests[1].body.body as Record<string, string>).target_digest, hash);
  const withdrawn = await client(intent({ action: "withdraw", target: "", statement: "" }));
  assert.equal(withdrawn.state, "withdrawn"); assert.equal(s.requests[2].url.endsWith("withdraw_channel_registration"), true);
  assert.deepEqual(s.requests[2].body.body, { configuration: "config-1", digest: hash, attestation: "claim-01" });
});

test("misbound receipts stay uncertain and only documented refusals are clean", async () => {
  for (const change of [(r: Record<string, unknown>) => { r.target_id = "other"; }, (r: Record<string, unknown>) => { r.business_state = "withdrawn"; }, (r: Record<string, unknown>) => { r.fields = { ...r.fields as object, actor: "other" }; },
    (r: Record<string, unknown>) => { r.fields = { ...r.fields as object, assurance: "verified" }; }, (r: Record<string, unknown>) => { r.fields = { ...r.fields as object, statement: "echo" }; }, (r: Record<string, unknown>) => { r.fields = { ...r.fields as object, target: "other" }; }]) {
    await assert.rejects(attestationClient(session, server(change).fetcher)(intent()), (e: unknown) => e instanceof SetupError && e.effect === "unknown" && e.code === "invalid_setup_receipt");
  }
  for (const state of ["withdrawn", "consumed"]) {
    const retried = await attestationClient(session, server(r => { r.fields = { ...r.fields as object, state }; }).fetcher)(intent());
    assert.equal(retried.state, state);
  }
  await assert.rejects(attestationClient(session, server(r => { r.fields = { ...r.fields as object, state: "unconsumed" }; }).fetcher)(intent({ action: "withdraw", target: "", statement: "" })), (e: unknown) => e instanceof SetupError && e.effect === "unknown");
  await assert.rejects(attestationClient(session, async input => { if (String(input) === "/api/v1/session") return Response.json({ user: session.user, workspace: session.workspace, csrf_token: session.csrf }); throw new TypeError("network"); })(intent()), (e: unknown) => e instanceof SetupError && e.effect === "unknown");
  for (const operation of [false, true]) {
    const s = server(r => { const request = r.request_id; for (const k of Object.keys(r)) delete r[k]; Object.assign(r, { request_id: request, code: "evidence_stale", next_action: "inspect_current_state", retry: false, ...(operation ? { operation_id: "operation-1" } : {}) }); }, 409);
    await assert.rejects(attestationClient(session, s.fetcher)(intent()), (e: unknown) => e instanceof SetupError && e.code === "evidence_stale" && e.effect === (operation ? "unknown" : "none"));
  }
  const s = server();
  await assert.rejects(attestationClient(session, s.fetcher)(intent({ actor: "other" })), (e: unknown) => e instanceof SetupError && e.code === "session_changed");
  assert.equal(s.requests.length, 0);
});

test("read client requests the scoped pages", async () => {
  const requests: Record<string, unknown>[] = [];
  const fetcher: typeof fetch = async (input, init) => {
    const url = String(input);
    if (url === "/api/v1/session") return Response.json({ user: session.user, workspace: session.workspace, csrf_token: session.csrf });
    const body = JSON.parse(String(init?.body)); requests.push({ url, ...body });
    const fields = url.endsWith("references") ? { ...head, schema: "hb_issuance_setup_references_v1", reference_00: JSON.stringify({ code: "synthetic_template_a", use: "document_template", applicability: "synthetic_only" }) } : { ...head, schema: "hb_issuance_setup_attestation_page_v1", snapshot: hash, count: "0", next: "" };
    return Response.json({ request_id: body.request_id, status: "completed", target_id: body.target_id, result_revision: "r4", replay: false, as_of: "2026-11-23T00:00:00Z", fields });
  };
  const client = setupReadClient(session, fetcher);
  assert.equal((await client.attestations(scope)).count, 0);
  assert.equal((await client.references(scope)).entries[0].code, "synthetic_template_a");
  assert.deepEqual(requests.map(r => String(r.url).split(".").at(-1)), ["attestations", "references"]);
  assert.deepEqual(requests[0].body, { configuration: "config-1", digest: hash });
  await assert.rejects(client.attestations(scope, "claim-01"), SetupError);
  assert.equal(requests.length, 2);
});
