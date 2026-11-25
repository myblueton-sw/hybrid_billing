import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, adminClient } from "./admin";
import { availOf, deliveryOf, hint, installOf, integrationsOf, licenseStateOf, licensesOf, markOf, pluginProposalOf, requestOf, storageOf, subscriptionOf } from "./integrations";
import { fieldReaction, react } from "../modules/administration/pages/integrations/live-model";

type Call = { url: string; init: RequestInit };

function fake(status: number, body: unknown) {
  const calls: Call[] = [];
  const f = (async (url: string, init: RequestInit) => {
    calls.push({ url, init });
    return new Response(JSON.stringify(body), { status });
  }) as unknown as typeof fetch;
  return { f, calls };
}

const sample = {
  revision: "r9",
  now: "2026-11-17T09:00:00Z",
  state: "configured",
  clients: {
    state: "configured",
    clients: [{ id: "console", kind: "first_party_console", state: "active", grants: [{ id: "g1", action: "view", kind: "focus_data", scope: { customer: "all", legal_entity: "all", contract: "all", period: "any" }, not_before: "2026-01-01T00:00:00Z", not_after: "2027-01-01T00:00:00Z", state: "active" }] }],
    service_users: [{ id: "svc-1", state: "active", owner: "u1", owner_login: "admin@example.invalid", usable_credentials: 0 }, { id: "svc-2", state: "active", usable_credentials: "2" }],
    manage_on: "M04",
    oauth_clients: "not_connected",
  },
  modules: {
    state: "configured",
    workspace: { id: "ws", runtime: "rt", verification: false },
    publishers: [{ id: "pub", name: "Sample Publisher", revocation: null, keys: [{ version: "k1", not_before: "", not_after: "", retired: false, compromise: null }] }],
    plugins: [{ plugin: "p", version: "1.0.0", capabilities: ["read_events:invoice.issued", 3], egress: [], endpoints: ["api"], timeout_seconds: 30, memory_mib: 1.5, concurrency: -1, storage: { namespace: "ns", retention_days: 0, deletion: "retain_for_evidence" }, screens: [{ id: "s", component: "table", label: "Overview", fields: null }], vulnerable: null }],
    installations: [{ plugin: "p", version: "1.0.0", state: "enabled", reason: "", revision: 2, endpoints: [], disposal: null, can_stop: true, stop_blocked: "", can_disable: true, disable_blocked: "", tokens: "not_issued", runtime: "not_connected" }],
    proposals: [{ number: 8, target: "plugin-proposal-8", kind: "upgrade", state: "pending", summary: { plugin: "p", version: "1.1.0" }, can_approve: true, approve_blocked: "", can_withdraw: false, withdraw_blocked: "not_proposer" }],
  },
  channels: {
    state: "configured",
    destinations: [{ id: "sample-erp-webhook", state: "active", source: "sample_allowlist" }],
    subscribers: ["sample-erp-connector"],
    subscriptions: [{ id: "s1", target: "webhook-subscription:s1", types: ["invoice.issued"], state: "active", cursor: 3, lag: 0, gaps: [{ after: 1, to: "x" }], actions: { pause: { allowed: true, blocked: "" }, revoke: { allowed: true, blocked: "x" } } }],
  },
  events: {
    state: "configured",
    types: [{ name: "invoice.issued", schema_version: "1", fields: ["id"], state: "active" }],
    outbox: { floor: 0, head: 9007199254740993 },
    deliveries: [{ id: "delivery-3", target: "webhook-letter:delivery-3", subscription: "s1", sequence: 2, kind: "dead", state: "failed", reason: "rejected", attempts: 0, outcome_unknown: "no", redeliver_request: { by: "u1", rationale: "r" }, dismiss_request: null, actions: "bogus" }],
    dispatcher: "not_connected",
  },
};

test("the integrations read parses every section and keeps unknowns visible", async () => {
  const r = fake(200, sample);
  const got = await adminClient("t", r.f).integrations();
  assert.equal(r.calls[0].url, "/api/v1/integrations");
  assert.equal(r.calls[0].init.method, "GET");
  assert.equal(got.revision, "r9");
  assert.equal(got.clients.clients?.[0].grants?.[0].scope.period, "any");
  assert.equal(got.clients.service_users?.[0].usable_credentials, 0);
  assert.equal(got.clients.service_users?.[1].usable_credentials, null);
  assert.equal(got.clients.service_users?.[1].owner_login, "");
  assert.deepEqual(got.modules.plugins?.[0].capabilities, ["read_events:invoice.issued"]);
  assert.equal(got.modules.plugins?.[0].timeout_seconds, 30);
  assert.equal(got.modules.plugins?.[0].memory_mib, null);
  assert.equal(got.modules.plugins?.[0].concurrency, null);
  assert.equal(got.modules.plugins?.[0].storage.retention_days, 0);
  assert.equal(got.modules.plugins?.[0].storage.owner, "");
  assert.equal(got.modules.plugins?.[0].screens?.[0].fields, null);
  assert.deepEqual(got.modules.plugins?.[0].egress, []);
  assert.equal(got.modules.workspace?.verification, false);
  assert.equal(got.modules.proposals?.[0].can_approve, true);
  assert.equal(got.modules.proposals?.[0].withdraw_blocked, "not_proposer");
  const s = got.channels.subscriptions?.[0];
  assert.equal(s?.lag, 0);
  assert.deepEqual(s?.gaps, [{ after: 1, to: null }]);
  assert.equal(s?.history, null);
  assert.deepEqual(s?.actions.pause, { allowed: true, blocked: "" });
  assert.deepEqual(s?.actions.revoke, { allowed: false, blocked: "x" });
  assert.deepEqual(s?.actions.approve, { allowed: false, blocked: "missing" });
  assert.equal(got.events.outbox.floor, 0);
  assert.equal(got.events.outbox.head, null);
  const d = got.events.deliveries?.[0];
  assert.equal(d?.attempts, 0);
  assert.equal(d?.outcome_unknown, null);
  assert.deepEqual(d?.redeliver_request, { id: "", target: "", by: "u1", by_login: "", rationale: "r" });
  assert.equal(d?.dismiss_request, null);
  assert.deepEqual(d?.actions.approve_redeliver, { allowed: false, blocked: "missing" });
});

test("whole-page and section not_configured, and missing sections, stay distinguishable", () => {
  const other = integrationsOf({ revision: "r1", now: "x", state: "not_configured" });
  assert.equal(other.state, "not_configured");
  assert.equal(other.modules.state, "");
  assert.equal(other.modules.proposals, null);
  assert.equal(other.channels.subscriptions, null);
  const empty = integrationsOf({ ...sample, channels: { state: "not_configured", destinations: [], subscribers: ["c"], subscriptions: [] } });
  assert.equal(empty.channels.state, "not_configured");
  assert.deepEqual(empty.channels.subscriptions, []);
  const broken = integrationsOf({ state: "configured", modules: { workspace: "x", publishers: "no", plugins: [null, 1] } });
  assert.equal(broken.modules.workspace, null);
  assert.equal(broken.modules.publishers, null);
  assert.deepEqual(broken.modules.plugins, []);
});

test("availability and flat pairs fail closed", () => {
  assert.deepEqual(availOf(undefined), { allowed: false, blocked: "missing" });
  assert.deepEqual(availOf({ allowed: true }), { allowed: false, blocked: "missing" });
  assert.deepEqual(availOf({ allowed: true, blocked: "own_request" }), { allowed: false, blocked: "own_request" });
  assert.deepEqual(availOf({ allowed: false, blocked: "" }), { allowed: false, blocked: "missing" });
  assert.deepEqual(availOf({ allowed: "true", blocked: "" }), { allowed: false, blocked: "missing" });
  assert.deepEqual(availOf({ allowed: true, blocked: 3 }), { allowed: false, blocked: "missing" });
  assert.deepEqual(hint(true, "own_proposal"), { allowed: false, blocked: "own_proposal" });
  assert.deepEqual(hint(null, ""), { allowed: false, blocked: "missing" });
  assert.deepEqual(hint(true, ""), { allowed: true, blocked: "" });
  const i = installOf({ can_stop: true, stop_blocked: "not_enabled", can_disable: "yes", disable_blocked: "", revision: 1.5 });
  assert.equal(i.can_stop, false);
  assert.equal(i.stop_blocked, "not_enabled");
  assert.equal(i.can_disable, false);
  assert.equal(i.revision, null);
  const p = pluginProposalOf({ number: 0, can_approve: true, approve_blocked: "own_proposal" });
  assert.equal(p.number, null);
  assert.equal(p.can_approve, false);
  assert.equal(subscriptionOf({ cursor: -1, lag: 2 ** 60 }).cursor, null);
  assert.equal(subscriptionOf({ cursor: -1, lag: 2 ** 60 }).lag, null);
  assert.equal(deliveryOf({ sequence: "2" }).sequence, null);
});

test("marks and requests distinguish none from unknown", () => {
  assert.equal(markOf(null), null);
  assert.equal(markOf(undefined), "unknown");
  assert.equal(markOf("x"), "unknown");
  assert.deepEqual(markOf({ by: "u" }), { by: "u", by_login: "", reason: "", confirmed_by: "", confirmed_by_login: "" });
  assert.equal(requestOf(null), null);
  assert.equal(requestOf(7), "unknown");
  assert.deepEqual(storageOf(null), { namespace: "", owner: "", retention_days: null, deletion: "", migration: "" });
});

test("commands carry the envelope with and without fields and keep the resend identity", async () => {
  const r = fake(200, { business_state: "pending", fields: { proposal: "10", target: "plugin-proposal-10" } });
  const api = adminClient("csrf", r.f);
  const tried = { request: "m01-a", key: "m01-b" };
  const out = await api.command("integrations.plugin_propose", "plugin:p", "r9", "qa", { kind: "upgrade", version: "1.1.0" }, tried);
  assert.equal(r.calls[0].url, "/api/v1/commands/integrations.plugin_propose");
  const body = JSON.parse(String(r.calls[0].init.body));
  assert.deepEqual(body, { request_id: "m01-a", target_id: "plugin:p", idempotency_key: "m01-b", expected_revision: "r9", rationale: "qa", body: { kind: "upgrade", version: "1.1.0" } });
  assert.equal((r.calls[0].init.headers as Record<string, string>)["X-Request-Id"], "m01-a");
  assert.deepEqual({ state: out.state, fields: out.fields }, { state: "pending", fields: { proposal: "10", target: "plugin-proposal-10" } });
  await api.command("integrations.plugin_propose", "plugin:p", "r9", "qa", { kind: "upgrade", version: "1.1.0" }, tried);
  const again = JSON.parse(String(r.calls[1].init.body));
  assert.equal(again.idempotency_key, body.idempotency_key);
  assert.equal(again.request_id, body.request_id);
  const sub = fake(200, { business_state: "proposed", fields: { subscription: "subscription-3" } });
  const o2 = await adminClient("c", sub.f).command("integrations.subscription_propose", "webhooks", "r1", "x", { destination: "d", types: "a,b" }, tried);
  assert.equal(o2.fields.subscription, "subscription-3");
  const letter = fake(200, { business_state: "queued", fields: { delivery: "delivery-3", redelivery: "delivery-7" } });
  const o3 = await adminClient("c", letter.f).command("integrations.letter_approve_redeliver", "webhook-letter:delivery-3", "r1", "x", {}, tried);
  assert.deepEqual(JSON.parse(String(letter.calls[0].init.body)).body, {});
  assert.equal(o3.fields.redelivery, "delivery-7");
  assert.equal(o3.state, "queued");
});

test("a duplicate subscription from another proposer is a stage refusal", async () => {
  const r = fake(409, { code: "invalid_stage" });
  await assert.rejects(
    adminClient("c", r.f).command("integrations.subscription_propose", "webhooks", "r1", "x", { destination: "d", types: "a" }, { request: "a", key: "b" }),
    (e: unknown) => e instanceof AdminError && e.failure === "stage" && react(e.failure, e.fields) === "hint",
  );
  const f = fake(400, { code: "invalid_request", fields: ["types"] });
  await assert.rejects(
    adminClient("c", f.f).command("integrations.subscription_propose", "webhooks", "r1", "x", { destination: "d", types: "a" }, { request: "a", key: "b" }),
    (e: unknown) => e instanceof AdminError && e.failure === "invalid" && fieldReaction("integrations.subscription_propose", e.fields) === "choice",
  );
});

test("license fields fail closed", () => {
  for (const [raw, want] of [["permitted", "permitted"], ["not_permitted", "not_permitted"], ["PERMITTED", "unknown"], ["", "unknown"], [null, "unknown"], [1, "unknown"], [undefined, "unknown"]] as const) {
    assert.equal(licenseStateOf(raw), want, String(raw));
  }
  assert.deepEqual(licensesOf(undefined), { permitted: null, source: "unknown" });
  assert.deepEqual(licensesOf({ permitted: ["MIT", 3, "Zlib"], source: "start_flag" }), { permitted: ["MIT", "Zlib"], source: "start_flag" });
  assert.deepEqual(licensesOf({ permitted: [], source: "installation_setting" }), { permitted: [], source: "unknown" });
  const page = integrationsOf({ modules: { licenses: { permitted: ["Apache-2.0"], source: "default" }, plugins: [{ plugin: "p", version: "1.0.0", license: "Apache-2.0", license_state: "permitted" }, { plugin: "p", version: "1.1.0", license: 7 }] } });
  assert.deepEqual(page.modules.licenses, { permitted: ["Apache-2.0"], source: "default" });
  assert.deepEqual(page.modules.plugins!.map((v) => [v.license, v.license_state]), [["Apache-2.0", "permitted"], ["", "unknown"]]);
});

test("rights, own pending items and withheld areas are read fail closed", () => {
  const got = integrationsOf({
    revision: "r1", now: "", state: "configured",
    rights: { administer_modules: true, stop_modules: "yes", unknown_right: true },
    own_pending: [{ kind: "resume_request", id: "resume-request-1", target: "webhook-resume-request:resume-request-1", object: "s1", state: "pending" }, { kind: "other", id: "x" }, "bad"],
    clients: { state: "not_permitted" }, modules: { state: "configured", actions: { propose: { allowed: false, blocked: "no_right" } } },
  });
  assert.deepEqual(got.rights, {
    administer_api_clients: false, administer_modules: true, approve_modules: false, stop_modules: false,
    administer_channels: false, approve_channels: false, stop_channels: false, reprocess_events: false,
  });
  assert.equal(Object.keys(got.rights ?? {}).includes("unknown_right"), false);
  assert.deepEqual(got.own_pending, [{ kind: "resume_request", id: "resume-request-1", target: "webhook-resume-request:resume-request-1", object: "s1", state: "pending" }]);
  assert.equal(got.clients.state, "not_permitted");
  assert.equal(got.clients.clients, null);
  assert.deepEqual(got.modules.actions.propose, { allowed: false, blocked: "no_right" });
  assert.deepEqual(got.channels.actions.propose, { allowed: false, blocked: "missing" });
  const none = integrationsOf({ revision: "r1", now: "", state: "configured" });
  assert.equal(none.rights, null);
  assert.equal(none.own_pending, null);
  assert.equal(integrationsOf({ rights: [true], own_pending: {} }).rights, null);
});

test("held redeliveries and request identities are parsed", () => {
  const got = integrationsOf({
    events: { deliveries: [{ id: "d9", target: "webhook-redelivery:d9", kind: "redelivery", hold: "approver_lost_right", requester: "a", approver: "b",
      actions: { reapprove: { allowed: true, blocked: "" } } }] },
    channels: { subscriptions: [{ id: "s1", resume_request: { id: "resume-request-2", target: "webhook-resume-request:resume-request-2", by: "a" },
      history: [{ action: "proposal_closed", actor: "b", reason: "requester_lost_right", request: "req-1" }] }] },
  });
  const d = got.events.deliveries?.[0];
  assert.equal(d?.hold, "approver_lost_right");
  assert.equal(d?.requester, "a");
  assert.equal(d?.approver, "b");
  assert.deepEqual(d?.actions.reapprove, { allowed: true, blocked: "" });
  assert.deepEqual(d?.actions.withdraw_redeliver, { allowed: false, blocked: "missing" });
  const s = got.channels.subscriptions?.[0];
  assert.deepEqual(s?.resume_request, { id: "resume-request-2", target: "webhook-resume-request:resume-request-2", by: "a", by_login: "", rationale: "" });
  assert.deepEqual(s?.history?.[0], { number: null, action: "proposal_closed", actor: "b", actor_login: "", rationale: "", reason: "requester_lost_right", request: "req-1" });
});

test("a subscription's history count, cursor and further page are read strictly", async () => {
  const { historyEntryOf, historyPageOf, olderFactsOf } = await import("./integrations");
  const { newestFirst } = await import("../shared/older-rows");
  assert.equal(historyEntryOf({ number: 7, action: "pause" }).number, 7);
  assert.equal(historyEntryOf({ number: "7" }).number, null);
  assert.deepEqual(olderFactsOf({ total: 120, next: "71" }), { total: 120, next: "71" });
  for (const bad of [null, [], { total: "1", next: "" }, { total: 1, next: 5 }, { next: "" }]) assert.equal(olderFactsOf(bad), null, JSON.stringify(bad));
  for (const bad of [{}, { history: [], next: "" }, { history: "x", next: "", total: 0 }, { history: [], next: 4, total: 0 }]) assert.equal(historyPageOf(bad), null, JSON.stringify(bad));
  const page = historyPageOf({ history: [{ number: 70, action: "pause" }, null, "x"], next: "21", total: 120 });
  assert.deepEqual([page?.rows.length, page?.rows[0].number, page?.next, page?.total], [1, 70, "21", 120]);
  const n = (x: number | null) => historyEntryOf({ number: x });
  assert.deepEqual(newestFirst([n(71), n(72), n(73)], [n(71), n(70), n(69)]).map((x) => x.number), [73, 72, 71, 70, 69]);
  assert.equal(newestFirst([n(2)], [n(null), n(null)]).length, 3);
  const calls: string[] = [];
  const f = (async (url: RequestInfo | URL) => {
    calls.push(String(url));
    return new Response(JSON.stringify(calls.length === 1 ? { history: [{ number: 70 }], next: "", total: 120 } : { history: [] }), { status: 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;
  const api = adminClient("t", f);
  assert.equal((await api.subscriptionHistory("sample erp", "71")).rows[0].number, 70);
  assert.equal(calls[0], "/api/v1/integrations/subscription-history?subscription=sample%20erp&limit=50&after=71");
  await assert.rejects(api.subscriptionHistory("s", "2"), (e: AdminError) => e.failure === "invalid");
});
