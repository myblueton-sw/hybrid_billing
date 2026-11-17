import { test } from "node:test";
import assert from "node:assert/strict";
import { integrationsOf, pluginProposalOf, type IntegrationsPage } from "../../../../service/integrations";
import {
  blockerPolicy, compareVersions, coveredBy, deliveriesShown, deniedOutcome, discard, draftOf, duplicateIn, fieldReaction, forget, frozenValues, keepDraft, keepsAttempt, letterTarget,
  commandsAllowed, normalTypes, ordered, pendingFor, pendingNow, pluginTarget, proposalTarget, proposeOffer, proposeValues, pruneRefusals, react, reasonProblem, refusedNow, selectionOf, setScope,
  settle, snapshotOf, subscribeBlocker, subscribeProblems, subscriptionTarget, summaryOf, timeKnown, utf8Bytes, vanished, viewOf, waitsForResume, capabilityOf, type CommandKey,
} from "./live-model";
import { k, storageText } from "./live-copy";
import { allWordTables, blockerWords } from "./live-codes";

const locales = ["en", "ko", "ar"] as const;
const me = "u1";
const other = "u2";
const avail = (blocked = "") => ({ allowed: blocked === "", blocked });

const version = (v: string, over: Record<string, unknown> = {}) => ({ plugin: "p", version: v, publisher: "pub", key_version: "k1", vulnerable: null, ...over });
const proposal = (n: number, kind: string, state: string, proposer: string, summary: Record<string, unknown> = {}, over: Record<string, unknown> = {}) => ({
  number: n, target: "plugin-proposal-" + n, kind, state, proposer, summary: { plugin: "p", ...summary }, can_approve: false, approve_blocked: "own_proposal", can_withdraw: true, withdraw_blocked: "", ...over,
});
const sub = (id: string, state: string, proposer: string, types: string[], over: Record<string, unknown> = {}) => ({
  id, target: "webhook-subscription:" + id, subscriber: "sample-erp-connector", destination: "erp", types, state, proposer,
  actions: { approve: avail("not_pending"), pause: avail(), request_resume: avail("not_pending"), approve_resume: avail("not_pending"), revoke: avail() }, ...over,
});
const letter = (id: string, over: Record<string, unknown> = {}) => ({
  id, target: "webhook-letter:" + id, subscription: "s1", sequence: 2, event_id: "ev-" + id, kind: "dead", state: "failed", attempts: 1, outcome_unknown: false, redeliver_request: null, dismiss_request: null,
  actions: { request_redeliver: avail(), approve_redeliver: avail("not_requested"), request_dismiss: avail(), approve_dismiss: avail("not_requested") }, ...over,
});

function page(over: { modules?: Record<string, unknown>; channels?: Record<string, unknown>; events?: Record<string, unknown> } = {}): IntegrationsPage {
  return integrationsOf({
    revision: "r1",
    now: "2026-11-17T09:00:00Z",
    state: "configured",
    clients: { state: "configured", clients: [], service_users: [], manage_on: "M04", oauth_clients: "not_connected" },
    modules: {
      state: "configured",
      workspace: { id: "ws", runtime: "rt", verification: false },
      publishers: [{ id: "pub", name: "P", revocation: null, keys: [{ version: "k1", retired: false, compromise: null }] }],
      plugins: [version("1.0.0"), version("1.1.0")],
      installations: [{ plugin: "p", version: "1.0.0", state: "enabled", can_stop: true, stop_blocked: "", can_disable: true, disable_blocked: "" }],
      proposals: [],
      ...over.modules,
    },
    channels: {
      state: "configured",
      destinations: [{ id: "erp", state: "active", source: "sample_allowlist" }, { id: "legacy", state: "inactive", source: "sample_allowlist" }],
      subscribers: ["sample-erp-connector"],
      subscriptions: [sub("s1", "active", other, ["invoice.issued"])],
      ...over.channels,
    },
    events: {
      state: "configured",
      types: [{ name: "invoice.issued", state: "active" }, { name: "payment.received", state: "active" }],
      outbox: { floor: 0, head: 3 },
      deliveries: [letter("delivery-3")],
      dispatcher: "not_connected",
      ...over.events,
    },
  });
}

const kinds = (p: IntegrationsPage, plugin = "p") => proposeOffer(p, me, plugin).options.map((o) => o.kind + (o.version ? ":" + o.version : ""));

test("views fall back to clients", () => {
  assert.equal(viewOf("events"), "events");
  assert.equal(viewOf("modules"), "modules");
  assert.equal(viewOf("nope"), "clients");
});

test("targets match their pattern, their ID and the command ID set", () => {
  const p = (n: unknown, target: string) => pluginProposalOf({ number: n, target });
  assert.equal(proposalTarget(p(8, "plugin-proposal-8")), "plugin-proposal-8");
  assert.equal(proposalTarget(p(8, "plugin-proposal-9")), "");
  assert.equal(proposalTarget(p(8, "plugin-proposal-08")), "");
  assert.equal(proposalTarget(p(null, "plugin-proposal-8")), "");
  const pg = page();
  const s = pg.channels.subscriptions![0];
  assert.equal(subscriptionTarget(s), "webhook-subscription:s1");
  assert.equal(subscriptionTarget({ ...s, target: "webhook-subscription:s2" }), "");
  assert.equal(subscriptionTarget({ ...s, id: "a b", target: "webhook-subscription:a b" }), "");
  assert.equal(subscriptionTarget({ ...s, id: "x".repeat(130), target: "webhook-subscription:" + "x".repeat(130) }), "");
  const d = pg.events.deliveries![0];
  assert.equal(letterTarget(d), "webhook-letter:delivery-3");
  assert.equal(letterTarget({ ...d, target: "webhook-letter:other" }), "");
  assert.equal(pluginTarget("p"), "plugin:p");
  assert.equal(pluginTarget(""), "");
  assert.equal(pluginTarget("p/../x"), "");
});

test("the blocker display policy hides only codes the row already states", () => {
  for (const c of ["not_active", "not_dead", "not_proposer", "not_enabled", "not_allowed_state"]) assert.equal(blockerPolicy(c), "hidden");
  assert.equal(blockerPolicy("not_pending"), "hidden");
  assert.equal(blockerPolicy("not_pending", true), "explain");
  for (const c of ["own_proposal", "own_request", "already_requested", "not_requested", "destination_inactive", "subscribe_refused", "dependency_unavailable", "revoked", "missing", "odd", "direction", "limit"]) {
    assert.equal(blockerPolicy(c), "explain");
  }
});

test("blocker sentences exist for every code in every language with fallbacks", () => {
  for (const l of locales) {
    for (const code of Object.keys(blockerWords)) assert.ok(k.blocker(l, code).length > 0);
    assert.notEqual(k.blocker(l, "missing"), k.blocker(l, "odd"));
    assert.ok(k.blocker(l, "odd").includes("odd"));
    assert.notEqual(k.blocker(l, "revoked", "approve_redeliver"), k.blocker(l, "revoked"));
    assert.ok(k.blockerExtra(l, "signature_invalid").length > 0);
    assert.equal(k.blockerExtra(l, "own_proposal"), "");
  }
  assert.equal(k.blocker("en", "own_proposal"), "You proposed this. Another administrator must approve it.");
  assert.equal(k.blocker("en", "already_requested"), "A request is already waiting for approval.");
  assert.ok(k.blocker("en", "revoked", "approve_redeliver").includes("can still be dismissed"));
  assert.equal(k.blocker("en", "not_pending", "revoke"), "The subscription is already revoked.");
});

test("every vocabulary key has EN, KO and AR copy and no copy claims a dispatch, send, signature or token", () => {
  for (const [name, table] of Object.entries(allWordTables)) {
    for (const [key, w] of Object.entries(table)) {
      assert.equal(w.length, 3, name + "." + key);
      for (const s of w) assert.ok(s.length > 0, name + "." + key);
    }
  }
  for (const l of locales) assert.ok(k.installState(l, "odd").includes("odd"));
  const claims = /\b(was|were|has been|have been) (sent|dispatched|signed|issued)\b/i;
  for (const l of ["en"] as const) {
    for (const s of [k.approveImpact(l), k.proposeImpact(l), k.subImpact(l, "approve"), k.subImpact(l, "approve_resume"), k.letterImpact(l, "approve_redeliver"), k.letterDone(l, "approve_redeliver", "d7"), k.statement(l), k.eventsLead(l)]) {
      assert.ok(!claims.test(s), s);
    }
  }
});

test("proposal summaries per kind and the unknown fallback; pending first, newest first", () => {
  for (const kind of ["publisher", "key", "version", "workspace", "runtime", "install", "upgrade", "rollback", "enable", "remove", "endpoint"]) {
    assert.equal(summaryOf(pluginProposalOf(proposal(1, kind, "pending", me))).known, true);
  }
  assert.equal(summaryOf(pluginProposalOf(proposal(1, "odd", "pending", me))).known, false);
  const list = page({ modules: { proposals: [proposal(1, "install", "applied", me), proposal(9, "endpoint", "pending", me), proposal(8, "upgrade", "pending", other), proposal(5, "enable", "withdrawn", me)] } }).modules.proposals!;
  const o = ordered(list);
  assert.deepEqual(o.pending.map((p) => p.number), [9, 8]);
  assert.deepEqual(o.closed.map((p) => p.number), [5, 1]);
});

test("plugin propose preconditions", () => {
  assert.equal(compareVersions("1.1.0", "1.0.0"), 1);
  assert.equal(compareVersions("1.0.0", "1.10.0"), -1);
  assert.equal(compareVersions("1.0", "1.0.0"), null);
  assert.deepEqual(kinds(page()), ["upgrade:1.1.0", "remove"]);
  assert.deepEqual(proposeOffer(page({ modules: { workspace: null } }), me, "p").blocker, { code: "no_workspace" });
  assert.deepEqual(proposeOffer(page({ modules: { plugins: [version("1.0")] } }), me, "p").blocker, { code: "no_version" });
  const inst = (state: string, v = "1.0.0") => page({ modules: { installations: [{ plugin: "p", version: v, state }] } });
  assert.deepEqual(kinds(inst("installed")), ["upgrade:1.1.0", "enable", "remove"]);
  assert.deepEqual(kinds(inst("stopped", "1.1.0")), ["rollback:1.0.0", "enable", "remove"]);
  assert.deepEqual(kinds(inst("removed")), ["install:1.0.0", "install:1.1.0"]);
  assert.deepEqual(kinds(page({ modules: { installations: [] } })), ["install:1.0.0", "install:1.1.0"]);
  assert.deepEqual(proposeOffer(inst("odd"), me, "p").blocker, { code: "unknown_state" });
  const vuln = (verification: boolean) =>
    page({ modules: { workspace: { id: "ws", verification }, plugins: [version("1.0.0", { vulnerable: { by: "u" } }), version("1.1.0", { vulnerable: { by: "u" } })], installations: [{ plugin: "p", version: "1.0.0", state: "stopped" }] } });
  assert.deepEqual(kinds(vuln(false)), ["remove"]);
  assert.deepEqual(kinds(vuln(true)), ["upgrade:1.1.0", "enable", "remove"]);
  const revoked = page({ modules: { publishers: [{ id: "pub", revocation: { by: "u" }, keys: [{ version: "k1", compromise: null }] }] } });
  assert.deepEqual(kinds(revoked), ["remove"]);
  const compromised = page({ modules: { publishers: [{ id: "pub", revocation: null, keys: [{ version: "k1", compromise: { by: "u" } }] }], installations: [] } });
  assert.deepEqual(kinds(compromised), []);
  assert.deepEqual(proposeOffer(compromised, me, "p").blocker, { code: "untrusted" });
  const mine = page({ modules: { proposals: [proposal(10, "upgrade", "pending", me, { version: "1.1.0" }), proposal(11, "remove", "pending", other)] } });
  assert.deepEqual(kinds(mine), []);
  assert.deepEqual(proposeOffer(mine, me, "p").blocker, { code: "duplicate", number: "10", own: true });
  const theirs = page({ modules: { proposals: [proposal(12, "upgrade", "pending", other, { version: "1.1.0" })] } });
  assert.deepEqual(kinds(theirs), ["remove"]);
  const both = page({ modules: { proposals: [proposal(12, "upgrade", "pending", other, { version: "1.1.0" }), proposal(13, "remove", "pending", other)] } });
  assert.deepEqual(proposeOffer(both, me, "p").blocker, { code: "duplicate", number: "12", own: false });
  assert.deepEqual(proposeValues({ kind: "enable", version: "" }), { kind: "enable" });
  assert.deepEqual(proposeValues({ kind: "upgrade", version: "1.1.0" }), { kind: "upgrade", version: "1.1.0" });
  for (const l of locales) for (const c of ["no_workspace", "no_version", "untrusted", "unknown_state", "duplicate", "duplicate_own"]) assert.ok(k.proposeBlocker(l, c, "8").length > 0);
  assert.equal(k.proposeBlocker("en", "duplicate_own", "8"), "Your proposal #8 already asks for this.");
});

test("subscription propose preconditions and coverage", () => {
  assert.equal(subscribeBlocker(page()), null);
  assert.equal(subscribeBlocker(page({ channels: { state: "not_configured", subscriptions: [] } })), null);
  assert.equal(subscribeBlocker(page({ events: { types: [] } })), "no_types");
  assert.equal(subscribeBlocker(page({ channels: { destinations: [{ id: "erp", state: "inactive" }] } })), "no_destination");
  assert.equal(subscribeBlocker(page({ channels: { subscribers: ["a", "b"] } })), "no_subscriber");
  assert.equal(subscribeBlocker(page({ channels: { subscribers: [] } })), "no_subscriber");
  assert.deepEqual(normalTypes(["b", "a", "b"]), ["a", "b"]);
  assert.deepEqual(coveredBy(page(), me, "erp", ["invoice.issued"]), { id: "s1", own: false });
  assert.equal(coveredBy(page(), me, "erp", ["invoice.issued", "payment.received"]), null);
  const own = page({ channels: { subscriptions: [sub("s2", "proposed", me, ["payment.received", "invoice.issued"])] } });
  assert.deepEqual(coveredBy(own, me, "erp", ["invoice.issued", "payment.received"]), { id: "s2", own: true });
  const paused = page({ channels: { subscriptions: [sub("s3", "paused", me, ["invoice.issued"])] } });
  assert.equal(coveredBy(paused, me, "erp", ["invoice.issued"]), null);
  assert.deepEqual(subscribeProblems(page(), "legacy", []), ["destination", "types_none"]);
  assert.deepEqual(subscribeProblems(page(), "erp", ["a", "b", "c", "d", "e", "f", "g", "h", "i"]), ["types_many"]);
  assert.deepEqual(subscribeProblems(page(), "erp", ["invoice.issued"]), []);
});

test("storage wording keeps a real zero", () => {
  const s = (deletion: string, retention_days: number | null) => ({ namespace: "n", owner: "o", retention_days, deletion, migration: "none" });
  assert.equal(storageText("en", s("retain_for_evidence", 30)), "Kept for evidence for 30 days after removal");
  assert.equal(storageText("en", s("retain_for_evidence", 0)), "Kept for evidence for 0 days after removal");
  assert.ok(storageText("en", s("retain_for_evidence", null)).includes("unknown"));
  assert.equal(storageText("en", s("delete_on_removal", 30)), "Deleted on removal");
  assert.equal(storageText("en", s("odd", 1)), "Unknown (odd)");
});

test("unknown versus zero, null lists, null flags and a year-1 time", () => {
  const p = integrationsOf({ state: "configured", events: { outbox: { floor: 0 }, deliveries: [{ attempts: 0, outcome_unknown: 1 }] }, channels: { subscriptions: [{ cursor: 0 }] } });
  assert.equal(p.events.outbox.floor, 0);
  assert.equal(p.events.outbox.head, null);
  assert.equal(p.events.deliveries![0].attempts, 0);
  assert.equal(p.events.deliveries![0].sequence, null);
  assert.equal(p.events.deliveries![0].outcome_unknown, null);
  assert.equal(p.channels.subscriptions![0].cursor, 0);
  assert.equal(p.channels.subscriptions![0].lag, null);
  assert.equal(p.channels.subscriptions![0].types, null);
  assert.equal(p.events.types, null);
  assert.equal(timeKnown("0001-01-01T00:00:00Z"), false);
  assert.equal(timeKnown(""), false);
  assert.equal(timeKnown("2026-11-17T09:00:00Z"), true);
  assert.equal(deliveriesShown(p, ""), p.events.deliveries);
  assert.equal(deliveriesShown(integrationsOf({}), "s1"), null);
});

test("failure reactions and field mappings", () => {
  assert.equal(react("invalid", ["version"]), "field");
  assert.equal(react("invalid", []), "reason");
  assert.equal(react("stage", []), "hint");
  assert.equal(react("refused", []), "hint");
  assert.equal(react("conflict", []), "conflict");
  assert.equal(react("denied", []), "denied");
  assert.equal(react("busy", []), "retry");
  assert.equal(react("unavailable", []), "retry");
  assert.equal(react("stale", []), "renew");
  assert.equal(react("partial", []), "partial");
  assert.equal(react("odd", []), "failed");
  assert.equal(keepsAttempt("retry"), true);
  assert.equal(keepsAttempt("hint"), false);
  const pp: CommandKey = "integrations.plugin_propose";
  const sp: CommandKey = "integrations.subscription_propose";
  assert.equal(fieldReaction(pp, ["version"]), "changed");
  assert.equal(fieldReaction(pp, ["kind"]), "kind");
  assert.equal(fieldReaction(sp, ["destination"]), "choice");
  assert.equal(fieldReaction(sp, ["types"]), "choice");
  assert.equal(fieldReaction(sp, ["rationale"]), "reason");
  for (const f of ["target_id", "expected_revision", "idempotency_key", "request_id", "zzz"]) assert.equal(fieldReaction(sp, [f]), "built");
  assert.equal(fieldReaction("integrations.subscription_pause", ["destination"]), "built");
  assert.equal(fieldReaction(pp, ["destination"]), "built");
});

test("reason trimming, whitespace rejection and UTF-8 byte counting", () => {
  assert.equal(reasonProblem("   "), "missing");
  assert.equal(reasonProblem(""), "missing");
  assert.equal(reasonProblem("  ok  "), "");
  assert.equal(utf8Bytes("가"), 3);
  assert.equal(utf8Bytes("ع"), 2);
  assert.equal(reasonProblem("가".repeat(85) + "a"), "");
  assert.equal(reasonProblem("가".repeat(86)), "long");
  assert.equal(reasonProblem(" " + "a".repeat(256) + " "), "");
});

const fresh = (() => {
  let n = 0;
  return () => ({ request: "r" + ++n, key: "k" + ++n });
})();

test("the attempt store keeps identical values only, is scoped by session and cleared by forget", () => {
  forget();
  setScope("ws", "u1", "s1");
  const a = pendingFor("integrations.subscription_pause", "t", { rationale: "x" }, "r1", fresh);
  assert.equal(pendingFor("integrations.subscription_pause", "t", { rationale: "x" }, "r2", fresh), a);
  assert.equal(a.expected, "r1");
  discard("integrations.subscription_pause", "t", { rationale: "x" });
  assert.equal(pendingNow("integrations.subscription_pause", "t"), a);
  discard("integrations.subscription_pause", "t", { rationale: "y" });
  assert.equal(pendingNow("integrations.subscription_pause", "t"), undefined);
  const b = pendingFor("integrations.subscription_pause", "t", { rationale: "x" }, "r1", fresh);
  settle("integrations.subscription_pause", "t", "retry");
  assert.equal(pendingNow("integrations.subscription_pause", "t"), b);
  assert.deepEqual(frozenValues(b), { rationale: "x" });
  settle("integrations.subscription_pause", "t", "conflict");
  assert.equal(pendingNow("integrations.subscription_pause", "t"), undefined);
  keepDraft("integrations.subscription_propose", "webhooks", { destination: "erp" });
  keepDraft("integrations.subscription_propose", "webhooks", { rationale: "r" });
  assert.deepEqual(draftOf("integrations.subscription_propose", "webhooks"), { destination: "erp", rationale: "r" });
  pendingFor("integrations.plugin_stop", "plugin:p", { rationale: "x" }, "r1", fresh);
  setScope("ws", "u1", "s2");
  assert.equal(pendingNow("integrations.plugin_stop", "plugin:p"), undefined);
  assert.deepEqual(draftOf("integrations.subscription_propose", "webhooks"), {});
  pendingFor("integrations.plugin_stop", "plugin:p", { rationale: "x" }, "r1", fresh);
  forget();
  setScope("ws", "u1", "s2");
  assert.equal(pendingNow("integrations.plugin_stop", "plugin:p"), undefined);
});

test("kept refusals for every command clear when the row shows a blocker, closes or leaves", () => {
  forget();
  setScope("ws", me, "s");
  const base = page({ modules: { proposals: [proposal(8, "upgrade", "pending", other, { version: "1.1.0" }, { can_approve: true, approve_blocked: "" })] } });
  const cases: [CommandKey, string][] = [
    ["integrations.plugin_approve", "plugin-proposal-8"],
    ["integrations.plugin_stop", "plugin:p"],
    ["integrations.plugin_disable", "plugin:p"],
    ["integrations.plugin_propose", "plugin:p"],
    ["integrations.subscription_propose", "webhooks"],
    ["integrations.subscription_pause", "webhook-subscription:s1"],
    ["integrations.letter_request_redeliver", "webhook-letter:delivery-3"],
  ];
  for (const [c, t] of cases) {
    settle(c, t, "hint");
    pruneRefusals(base, me);
    assert.equal(refusedNow(c, t), true, c);
    settle(c, t, "ok");
    assert.equal(refusedNow(c, t), false, c);
    settle(c, t, "hint");
  }
  for (const g of ["proposal", "propose_plugin", "plugin", "propose_subscription", "subscription", "letter"] as const) for (const l of locales) assert.ok(k.kept(l, g).length > 0 && k.gone(l, g).length > 0);
  const blocked = page({
    modules: { proposals: [proposal(8, "upgrade", "pending", other, { version: "1.1.0" }, { can_approve: false, approve_blocked: "direction" })], installations: [{ plugin: "p", version: "1.0.0", state: "installed", stop_blocked: "not_enabled", disable_blocked: "not_allowed_state" }] },
    channels: { subscriptions: [sub("s1", "paused", other, ["invoice.issued"], { actions: { pause: avail("not_active") } })] },
    events: { deliveries: [] },
  });
  pruneRefusals(blocked, me);
  assert.equal(refusedNow("integrations.plugin_approve", "plugin-proposal-8"), false);
  assert.equal(refusedNow("integrations.plugin_stop", "plugin:p"), false);
  assert.equal(refusedNow("integrations.plugin_disable", "plugin:p"), false);
  assert.equal(refusedNow("integrations.plugin_propose", "plugin:p"), true);
  assert.equal(refusedNow("integrations.subscription_pause", "webhook-subscription:s1"), false);
  assert.equal(refusedNow("integrations.letter_request_redeliver", "webhook-letter:delivery-3"), false);
  assert.equal(refusedNow("integrations.subscription_propose", "webhooks"), true);
  pruneRefusals(page({ events: { types: [] } }), me);
  assert.equal(refusedNow("integrations.subscription_propose", "webhooks"), false);
  forget();
});

test("a denied approval of a vanished letter or proposal after an unknown outcome says it may have been applied", () => {
  forget();
  setScope("ws", me, "s");
  const gone = page({ events: { deliveries: [] }, modules: { proposals: [] } });
  const still = page();
  assert.equal(deniedOutcome("integrations.letter_approve_redeliver", "webhook-letter:delivery-3", true, gone), "applied_letter");
  assert.equal(deniedOutcome("integrations.letter_approve_dismiss", "webhook-letter:delivery-3", true, gone), "applied_letter");
  assert.equal(deniedOutcome("integrations.letter_approve_redeliver", "webhook-letter:delivery-3", false, gone), "gone");
  assert.equal(deniedOutcome("integrations.letter_approve_redeliver", "webhook-letter:delivery-3", true, still), "gone");
  assert.equal(deniedOutcome("integrations.plugin_approve", "plugin-proposal-8", true, gone), "applied_proposal");
  assert.equal(deniedOutcome("integrations.subscription_pause", "webhook-subscription:zz", true, gone), "gone");
  pendingFor("integrations.letter_approve_redeliver", "webhook-letter:delivery-3", { rationale: "x" }, "r1", fresh, [], { event: "ev-delivery-3" });
  assert.deepEqual(vanished(still, "events"), []);
  assert.deepEqual(vanished(gone, "events"), [{ command: "integrations.letter_approve_redeliver", target: "webhook-letter:delivery-3", context: { event: "ev-delivery-3" } }]);
  assert.deepEqual(vanished(gone, "channels"), []);
  settle("integrations.letter_approve_redeliver", "webhook-letter:delivery-3", "denied");
  assert.deepEqual(vanished(gone, "events"), []);
  assert.ok(k.vanishedHint("en", "ev-1").includes("ev-1"));
  forget();
});

test("duplicate announcements compare against the snapshot taken before the first send", () => {
  forget();
  setScope("ws", me, "s");
  const before = page({ modules: { proposals: [proposal(8, "upgrade", "pending", other)] } });
  const snap = snapshotOf(before, "integrations.plugin_propose");
  assert.deepEqual(snap, ["8"]);
  const tried = pendingFor("integrations.plugin_propose", "plugin:p", { kind: "remove", rationale: "x" }, "r1", fresh, snap);
  assert.equal(duplicateIn(tried.snapshot, "8"), true);
  assert.equal(duplicateIn(tried.snapshot, "10"), false);
  const after = page({ modules: { proposals: [proposal(10, "remove", "pending", me), proposal(8, "upgrade", "pending", other)] } });
  const resend = pendingFor("integrations.plugin_propose", "plugin:p", { kind: "remove", rationale: "x" }, "r2", fresh, snapshotOf(after, "integrations.plugin_propose"));
  assert.equal(resend, tried);
  assert.equal(duplicateIn(resend.snapshot, "10"), false);
  assert.deepEqual(snapshotOf(page(), "integrations.subscription_propose"), ["s1"]);
  forget();
});

test("URL selections are validated against the read", () => {
  const p = page({ modules: { proposals: [proposal(8, "upgrade", "pending", other)] } });
  const q = (s: string) => new URLSearchParams(s);
  assert.equal(selectionOf(q("proposal=8"), p, "modules").proposal, "8");
  assert.equal(selectionOf(q("proposal=08"), p, "modules").proposal, "");
  assert.equal(selectionOf(q("proposal=9"), p, "modules").proposal, "");
  assert.equal(selectionOf(q("proposal=8"), p, "events").proposal, "");
  assert.equal(selectionOf(q("subscription=s1"), p, "events").subscription, "s1");
  assert.equal(selectionOf(q("subscription=zz"), p, "channels").subscription, "");
  assert.equal(selectionOf(q("delivery=delivery-3"), p, "events").delivery, "delivery-3");
  assert.equal(selectionOf(q("delivery=nope"), p, "events").delivery, "");
  assert.equal(selectionOf(q("delivery=delivery-3"), null, "events").delivery, "");
  assert.deepEqual(deliveriesShown(p, "zz"), []);
});

test("waiting deliveries of a paused subscription wait for the resume; capabilities split or stay whole", () => {
  const p = page({ channels: { subscriptions: [sub("s1", "paused", other, ["invoice.issued"])] }, events: { deliveries: [letter("d5", { kind: "open", state: "waiting" })] } });
  assert.equal(waitsForResume(p, p.events.deliveries![0]), true);
  assert.equal(waitsForResume(page({ events: { deliveries: [letter("d5", { kind: "open", state: "waiting" })] } }), page({ events: { deliveries: [letter("d5", { kind: "open", state: "waiting" })] } }).events.deliveries![0]), false);
  assert.deepEqual(capabilityOf("read_events:invoice.issued"), { kind: "read_events", target: "invoice.issued" });
  assert.deepEqual(capabilityOf("odd:x"), { raw: "odd:x" });
  assert.deepEqual(capabilityOf("plain"), { raw: "plain" });
});

test("null lists fail closed and a malformed plugin ID has its own blocker", () => {
  assert.deepEqual(proposeOffer(page({ modules: { installations: null } }), me, "p"), { options: [], blocker: { code: "unknown_state" } });
  assert.deepEqual(proposeOffer(page({ modules: { proposals: null } }), me, "p"), { options: [], blocker: { code: "unknown_list" } });
  assert.deepEqual(proposeOffer(page(), me, "p/x"), { options: [], blocker: { code: "no_command" } });
  assert.deepEqual(proposeOffer(page({ modules: { workspace: null } }), me, "p/x").blocker, { code: "no_command" });
  assert.equal(subscribeBlocker(page({ channels: { subscriptions: null } })), "unknown_list");
  assert.deepEqual(coveredBy(page({ channels: { subscriptions: null } }), me, "erp", ["invoice.issued"]), { id: "", own: false });
  for (const l of locales) {
    assert.equal(k.proposeBlocker(l, "no_command", ""), k.noCommand(l));
    assert.ok(k.proposeBlocker(l, "unknown_list", "").length > 0);
    assert.notEqual(k.subBlocker(l, "unknown_list"), k.subBlocker(l, "no_subscriber"));
  }
});

test("commands are offered only with a known revision", () => {
  assert.equal(commandsAllowed(page()), true);
  assert.equal(commandsAllowed(integrationsOf({ state: "configured", revision: "" })), false);
  assert.equal(commandsAllowed(integrationsOf({ state: "configured", revision: 7 })), false);
  for (const l of locales) assert.ok(k.readOnly(l).length > 0);
});

const stripMarks = (v: string) => v.replace(/[\u064B-\u0652\u2068\u2069]/g, "");
const positiveClaims: Record<(typeof locales)[number], RegExp> = {
  en: /\b(was|were|has been|have been) (sent|dispatched|signed|issued|delivered)\b/i,
  ko: /(보냈습니다|발송했습니다|발송되었습니다|전송했습니다|전송되었습니다|서명했습니다|서명되었습니다|발급했습니다|발급되었습니다)/,
  ar: /((?<=^|\s)(?<!شيء\s)(أرسل|أرسلت|وقع|وقعت|أصدر|أصدرت)(?=$|[\s.،:])|تم (إرسال|الإرسال|توقيع|إصدار))/,
};

test("decision, impact and done copy in every language never claims a dispatch, send, signature or token", () => {
  for (const l of locales) {
    const list = [
      k.statement(l), k.clientsLead(l), k.modulesLead(l), k.channelsLead(l), k.eventsLead(l),
      k.approveImpact(l), k.withdrawImpact(l), k.proposeImpact(l), k.stopImpact(l), k.disableImpact(l), k.proposeSubImpact(l),
      ...["approve", "pause", "request_resume", "approve_resume", "revoke"].flatMap((a) => [k.subImpact(l, a), k.subDone(l, "s1", "Active")]),
      ...["approve_redeliver", "request_redeliver", "request_dismiss", "approve_dismiss"].flatMap((a) => [k.letterImpact(l, a), k.letterDone(l, a, "d7")]),
      k.approvedDone(l, "8", "Applied"), k.withdrawnDone(l, "8"), k.proposedDone(l, "10"), k.proposedSubDone(l, "s3"), k.runtimeDone(l, "p", "Stopped"),
      k.duplicateProposal(l, "8"), k.duplicateSub(l, "s1"), k.vanishedLetter(l), k.vanishedProposal(l), k.vanishedHint(l, "e"), k.mayApplied(l),
    ];
    for (const s of list) {
      assert.ok(s.length > 0);
      assert.ok(!positiveClaims[l].test(stripMarks(s)), l + ": " + s);
    }
  }
});

test("a step-up refusal or cancellation keeps the attempt and the form", () => {
  for (const f of ["reauth", "reauth_cancelled"]) {
    assert.equal(react(f, []), "reauth");
    assert.equal(keepsAttempt(react(f, [])), true);
  }
});
