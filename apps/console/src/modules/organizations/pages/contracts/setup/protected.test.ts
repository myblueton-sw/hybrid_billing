import assert from "node:assert/strict";
import test from "node:test";
import { setupReadClient } from "./client";
import { SetupError } from "./context";
import { protectedChangeOf, protectedChangesOf, protectedChannelOf, protectedChannelsOf, protectedSummaryOf } from "./protected";

const hash = "a".repeat(64), at = "2026-11-23T01:00:00.000000001Z", later = "2026-11-24T01:00:00.000000001Z";
const scope = { contract: "contract-1", configuration: "config-1", digest: hash };
const head = { contract: scope.contract, configuration: scope.configuration, digest: hash, current_eligibility: "unchecked" };
const group = (prefix: string, kind: string) => ({ [`${prefix}_subject_kind`]: kind, [`${prefix}_changes`]: "0", [`${prefix}_other_provenance`]: "0", [`${prefix}_unowned_module`]: "0", [`${prefix}_latest_scheduled`]: "", [`${prefix}_latest_scheduled_effective`]: "" });
const summary = (): Record<string, string> => ({
  ...head, schema: "hb_issuance_protected_setup_v1", channels: "0", channels_other_provenance: "0", channels_unowned_module: "0",
  rules_state: "confirmation_unknown", rules_observation: "pending", rules_id: "rules-1", rules_digest: hash, rules_approvals: "1",
  rules_validity_seconds: "172800", rules_max_lead_seconds: "604800", rules_channel_age_seconds: "86400", rules_proposer: "first",
  rules_confirmer: "", rules_pending_confirmer: "second", rules_proposed_at: at, rules_confirmed_at: "",
  ...group("recipient", "customer"), ...group("template", "document_kind"), ...group("account", "entity"),
});
const channel = (): Record<string, string> => ({ channel: "channel-01", digest: hash, subject_kind: "customer", kind: "email", state: "admission_unknown", observation: "registered", registrar: "first", admitter: "", pending_admitter: "second", registered_at: at, admitted_at: "", retired_by: "", retired_at: "", rules: "current", request_not_before: "" });
const change = (): Record<string, unknown> => ({ change: "change-01", digest: hash, field: "document_template", phase: "decision_unknown", state: "confirmed", observation: "confirmed", provenance: "staff_entry", instruction_kind: "in_person", channel: "channel-01", rules: "current", requester: "first", requested_at: at, effective: later, approvals_required: "1", verifier: "second", verification_outcome: "confirmed", verified_at: at, pending_verifier: "", pending_decision_actor: "third", ended_by: "", ended_at: "", decisions: [] });
const envelope = (fields: Record<string, string>) => ({ revision: "r1", at: "2026-11-23T00:00:00Z", fields });
const rejects = (run: () => unknown) => assert.throws(run, SetupError);

test("summary keeps pinned actors pending and never reads completion into uncertain rules", () => {
  const got = protectedSummaryOf(envelope(summary()), scope);
  assert.equal(got.rules.confirmer, ""); assert.equal(got.rules.pendingConfirmer, "second"); assert.equal(got.currentEligibility, "unchecked");
  assert.equal(Object.keys(summary()).length, 39);
  const active = { ...summary(), rules_observation: "active", rules_confirmer: "second", rules_pending_confirmer: "", rules_confirmed_at: later };
  assert.equal(protectedSummaryOf(envelope(active), scope).rules.state, "confirmation_unknown");
  const none: Record<string, string> = { ...summary(), rules_state: "none", rules_observation: "unowned" };
  for (const key of Object.keys(none)) if (key.startsWith("rules_") && key !== "rules_state" && key !== "rules_observation") none[key] = "";
  assert.equal(protectedSummaryOf(envelope(none), scope).rules.id, "");
  const invalid: Record<string, string>[] = [{ rules_confirmer: "second" }, { rules_state: "confirmed" }, { rules_observation: "unowned" }, { current_eligibility: "eligible" }, { digest: "b".repeat(64) },
    { rules_pending_confirmer: "first" }, { template_latest_scheduled: "change-01" }, { account_subject_kind: "customer" }, { ready: "true" }, { rules_approvals: "4" }];
  for (const bad of invalid) {
    rejects(() => protectedSummaryOf(envelope({ ...summary(), ...bad }), scope));
  }
  rejects(() => protectedSummaryOf(envelope({ ...none, rules_digest: hash }), scope));
  const scheduled = { ...summary(), template_changes: "1", template_latest_scheduled: "change-01", template_latest_scheduled_effective: later };
  assert.equal(protectedSummaryOf(envelope(scheduled), scope).fields.document_template.latestScheduled, "change-01");
});

test("channel rows separate observed admission, pending admitter, retirement and the maturity fact", () => {
  assert.equal(protectedChannelOf(channel()).pending_admitter, "second");
  const admitted = { ...channel(), state: "admitted", observation: "admitted", admitter: "second", pending_admitter: "", admitted_at: at, request_not_before: later };
  assert.equal(protectedChannelOf(admitted).request_not_before, later);
  assert.equal(protectedChannelOf({ ...admitted, observation: "retired", retired_by: "other", retired_at: later, request_not_before: "" }).state, "admitted");
  assert.equal(protectedChannelOf({ ...channel(), state: "admission_unknown", observation: "admitted", admitter: "second", pending_admitter: "", admitted_at: at }).state, "admission_unknown");
  for (const bad of [{ admitter: "second" }, { state: "admitted" }, { request_not_before: later }, { observation: "retired" }, { pending_admitter: "first" }, { kind: "fax" }, { reference: "vault" }, { registered_at: "" }]) {
    rejects(() => protectedChannelOf({ ...channel(), ...bad }));
  }
  for (const bad of [{ rules: "other" }, { observation: "retired", retired_by: "other", retired_at: later }, { state: "admission_unknown" }]) {
    rejects(() => protectedChannelOf({ ...admitted, ...bad }));
  }
});

test("change rows keep unknown phases, project lapse and refuse invented completion", () => {
  const got = protectedChangeOf(change(), "document_template");
  assert.equal(got.pending_decision_actor, "third"); assert.equal(got.decisions.length, 0); assert.ok(Object.isFrozen(got.decisions));
  assert.equal(protectedChangeOf({ ...change(), observation: "lapsed", ended_at: later }, "document_template").state, "confirmed");
  const absent = { ...change(), phase: "request_unknown", state: "", observation: "absent", requested_at: "", approvals_required: "", verifier: "", verification_outcome: "", verified_at: "", pending_decision_actor: "" };
  assert.equal(protectedChangeOf(absent, "document_template").requester, "first");
  const scheduled = { ...change(), state: "scheduled", observation: "scheduled", decisions: [{ approver: "third", verdict: "approve", at: later }], pending_decision_actor: "" };
  assert.equal(protectedChangeOf(scheduled, "document_template").phase, "decision_unknown");
  rejects(() => protectedChangeOf(change(), "payment_account"));
  for (const bad of [{ observation: "scheduled" }, { pending_verifier: "fourth" }, { phase: "recorded" }, { verifier: "first" }, { ended_at: later }, { after: "vault" },
    { decisions: [{ approver: "third", verdict: "approve", at: later }, { approver: "third", verdict: "approve", at: later }] }, { decisions: [{ approver: "third", verdict: "approve", at: later, reason: "other" }] },
    { verification_outcome: "" }, { observation: "lapsed" }, { approvals_required: "0" }]) {
    rejects(() => protectedChangeOf({ ...change(), ...bad }, "document_template"));
  }
  rejects(() => protectedChangeOf({ ...absent, state: "requested" }, "document_template"));
  rejects(() => protectedChangeOf({ ...absent, phase: "recorded" }, "document_template"));
});

test("pages bind scope, snapshot, order and exact counts", () => {
  const page = (rows: Record<string, unknown>[], extra: Record<string, string> = {}) => envelope({ ...head, schema: "hb_issuance_protected_channel_page_v1", snapshot: hash, count: String(rows.length), next: "", other_provenance: "1", unowned_module: "2", ...Object.fromEntries(rows.map((r, i) => [`row_${String(i).padStart(2, "0")}`, JSON.stringify(r)])), ...extra });
  const got = protectedChannelsOf(page([channel(), { ...channel(), channel: "channel-02" }]), scope);
  assert.equal(got.rows.length, 2); assert.equal(got.otherProvenance, 1); assert.equal(got.unownedModule, 2);
  rejects(() => protectedChannelsOf(page([{ ...channel(), channel: "channel-02" }, channel()]), scope));
  rejects(() => protectedChannelsOf(page([channel()]), { ...scope, configuration: "config-2" }));
  rejects(() => protectedChannelsOf(page([channel()]), scope, "channel-00", "b".repeat(64)));
  rejects(() => protectedChannelsOf(page([channel()]), scope, "channel-01", hash));
  rejects(() => protectedChannelsOf(page([channel()], { count: "11" }), scope));
  rejects(() => protectedChannelsOf(page([channel()], { field: "document_template" }), scope));
  assert.equal(protectedChannelsOf(page([{ ...channel(), channel: "channel-11" }], { count: "11" }), scope, "channel-10", hash).rows[0].channel, "channel-11");

  const trail = (rows: Record<string, unknown>[], extra: Record<string, string> = {}) => envelope({ ...page(rows, extra).fields, schema: "hb_issuance_protected_change_page_v1", field: "document_template", ...extra });
  const ordered = protectedChangesOf(trail([{ ...change(), change: "z-first", phase: "recorded", pending_decision_actor: "" }, change()]), scope, "document_template");
  assert.deepEqual(ordered.rows.map(r => r.change), ["z-first", "change-01"]);
  rejects(() => protectedChangesOf(trail([change(), change()]), scope, "document_template"));
  rejects(() => protectedChangesOf(trail([change()]), scope, "payment_account"));
  rejects(() => protectedChangesOf(trail([change()], { schema: "hb_issuance_protected_channel_page_v1" }), scope, "document_template"));
});

test("client sends the exact scope and cursor and refuses malformed input before any request", async () => {
  const session = { user: "reader", workspace: "workspace", csrf: "synthetic-csrf", role: "member", login: "reader" };
  const requests: { url: string; body: Record<string, unknown> }[] = [];
  const fetcher: typeof fetch = async (input, init) => {
    const url = String(input);
    if (url === "/api/v1/session") return Response.json({ user: session.user, workspace: session.workspace, csrf_token: session.csrf });
    const body = JSON.parse(String(init?.body)); requests.push({ url, body });
    const fields = url.endsWith("protected") ? summary() : { ...head, schema: url.endsWith("channels") ? "hb_issuance_protected_channel_page_v1" : "hb_issuance_protected_change_page_v1", snapshot: hash, count: "0", next: "", other_provenance: "0", unowned_module: "0", ...(url.endsWith("changes") ? { field: body.body.field } : {}) };
    return Response.json({ request_id: body.request_id, status: "completed", target_id: body.target_id, result_revision: "r4", replay: false, as_of: "2026-11-23T00:00:00Z", fields });
  };
  const client = setupReadClient(session, fetcher);
  assert.equal((await client.protectedSummary(scope)).rules.state, "confirmation_unknown");
  assert.equal((await client.protectedChannels(scope)).count, 0);
  assert.equal((await client.protectedChanges(scope, "payment_account")).count, 0);
  assert.deepEqual(requests.map(r => r.url.split(".").at(-1)), ["protected", "protected_channels", "protected_changes"]);
  assert.deepEqual(requests[0].body.body, { configuration: "config-1", digest: hash });
  assert.deepEqual(requests[2].body.body, { configuration: "config-1", digest: hash, field: "payment_account" });
  assert.equal(requests[2].body.target_id, "contract-1");
  const sent = requests.length;
  await assert.rejects(client.protectedSummary({ ...scope, digest: "short" }), SetupError);
  await assert.rejects(client.protectedChannels(scope, "channel-01"), SetupError);
  await assert.rejects(client.protectedChanges(scope, "tax_identifier" as never), SetupError);
  assert.equal(requests.length, sent);
});
