import { test } from "node:test";
import assert from "node:assert/strict";
import { integrationsOf, pluginProposalOf, type IntegrationsPage } from "../../../../service/integrations";
import {
  blockerPolicy, compareVersions, coveredBy, deliveriesCut, deliveriesShown, deniedOutcome, existingProposal, discard, draftOf, duplicateIn, fieldReaction, forget, frozenValues, keepDraft, keepsAttempt, letterTarget,
  commandsAllowed, normalTypes, ordered, pendingFor, pendingNow, pluginTarget, proposalTarget, proposeOffer, proposeValues, pruneRefusals, react, reasonProblem, refusedNow, selectionOf, setScope,
  settle, snapshotOf, subscribeBlocker, subscribeProblems, subscriptionTarget, summaryOf, timeKnown, utf8Bytes, vanished, viewOf, waitsForResume, capabilityOf, type CommandKey,
  dryRunBlocked, refusedByList, refusedLicense, refusedValues, runnable,
} from "./live-model";
import { k, storageText } from "./live-copy";
import { allWordTables, blockerWords } from "./live-codes";

const locales = ["en", "ko", "ar"] as const;
const me = "u1";
const other = "u2";
const avail = (blocked = "") => ({ allowed: blocked === "", blocked });

const version = (v: string, over: Record<string, unknown> = {}) => ({ plugin: "p", version: v, publisher: "pub", key_version: "k1", vulnerable: null, license: "MIT", license_state: "permitted", ...over });
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
  const snap = snapshotOf(page(), "integrations.subscription_propose");
  assert.deepEqual(snap, ["s1"]);
  const tried = pendingFor("integrations.subscription_propose", "webhook-channel:c", { rationale: "x" }, "r1", fresh, snap);
  assert.equal(duplicateIn(tried.snapshot, "s1"), true);
  assert.equal(duplicateIn(tried.snapshot, "s2"), false);
  const resend = pendingFor("integrations.subscription_propose", "webhook-channel:c", { rationale: "x" }, "r2", fresh, ["s2", "s1"]);
  assert.equal(resend, tried);
  assert.equal(duplicateIn(resend.snapshot, "s2"), false);
  assert.deepEqual(snapshotOf(page({ modules: { proposals: [proposal(8, "upgrade", "pending", other)] } }), "integrations.plugin_propose"), []);
  assert.equal(existingProposal({ proposal: "8", existing: "true" }), true);
  assert.deepEqual([existingProposal({ proposal: "8" }), existingProposal({ existing: "false" }), existingProposal({ existing: "" })], [false, false, false]);
  assert.ok(k.duplicateProposal("en", "8").includes("No new proposal was created"));
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
  assert.deepEqual(proposeOffer(page({ modules: { installations: null } }), me, "p"), { options: [], blocker: { code: "unknown_state" }, withheld: [] });
  assert.deepEqual(proposeOffer(page({ modules: { proposals: null } }), me, "p"), { options: [], blocker: { code: "unknown_list" }, withheld: [] });
  assert.deepEqual(proposeOffer(page(), me, "p/x"), { options: [], blocker: { code: "no_command" }, withheld: [] });
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

test("a license that is not permitted, or unknown, keeps a version from being offered", () => {
  const closed = (state: string) => version("1.2.0", { license: "LicenseRef-closed", license_state: state });
  assert.equal(runnable(page(), page().modules.plugins![0]), true);
  for (const state of ["not_permitted", "unknown", "PERMITTED", ""]) {
    const p = page({ modules: { plugins: [version("1.0.0"), version("1.1.0"), closed(state)] } });
    assert.equal(runnable(p, p.modules.plugins![2]), false, state);
    assert.deepEqual(kinds(p), ["upgrade:1.1.0", "remove"], state);
    assert.deepEqual(proposeOffer(p, me, "p").withheld.map((v) => v.version), ["1.2.0"], state);
  }
  const fresh = page({ modules: { installations: [], plugins: [closed("not_permitted"), version("1.3.0", { license: "GPL-3.0-only", license_state: "not_permitted" })] } });
  assert.deepEqual(kinds(fresh), []);
  assert.deepEqual(proposeOffer(fresh, me, "p").blocker, { code: "license" });
  assert.deepEqual(proposeOffer(fresh, me, "p").withheld.map((v) => v.version), ["1.2.0", "1.3.0"]);
  const unknownOnly = page({ modules: { installations: [], plugins: [closed("not_permitted"), version("1.3.0", { license_state: "odd" })] } });
  assert.deepEqual(proposeOffer(unknownOnly, me, "p").blocker, { code: "license_unknown" });
  assert.equal(refusedByList(unknownOnly.modules.plugins![0]), true);
  assert.equal(refusedByList(unknownOnly.modules.plugins![1]), false);
  assert.equal(refusedByList(undefined), false);
  const mixed = page({ modules: { installations: [], plugins: [closed("not_permitted"), version("1.3.0", { vulnerable: { by: "u" } })] } });
  assert.deepEqual(proposeOffer(mixed, me, "p").blocker, { code: "untrusted" });
  const vulnerableClosed = page({ modules: { installations: [], plugins: [version("1.2.0", { license_state: "not_permitted", vulnerable: { by: "u" } })] } });
  assert.deepEqual(proposeOffer(vulnerableClosed, me, "p").withheld, []);
  const stopped = page({ modules: { plugins: [version("1.0.0", { license_state: "not_permitted" }), version("1.1.0")], installations: [{ plugin: "p", version: "1.0.0", state: "stopped" }] } });
  assert.deepEqual(kinds(stopped), ["upgrade:1.1.0", "remove"]);
  assert.deepEqual(proposeOffer(stopped, me, "p").withheld.map((v) => v.version), ["1.0.0"]);
  const enabled = page({ modules: { plugins: [version("1.0.0", { license_state: "not_permitted" }), version("1.1.0")] } });
  assert.deepEqual(proposeOffer(enabled, me, "p").withheld, []);
  for (const l of locales) {
    assert.ok(k.proposeBlocker(l, "license", "").length > 0);
    assert.notEqual(k.proposeBlocker(l, "license", ""), k.proposeBlocker(l, "zzz", ""));
  }
});

test("the license refusal reason and the blocked sentence", () => {
  assert.ok(dryRunBlocked.includes("license_not_permitted"));
  for (const l of locales) {
    assert.equal(blockerWords.license_not_permitted.length, 3);
    assert.equal(k.blocker(l, "license_not_permitted", "plugin_approve"), blockerWords.license_not_permitted[locales.indexOf(l)]);
    assert.ok(k.blockerExtra(l, "license_not_permitted").length > 0);
  }
  assert.equal(k.blocker("en", "license_not_permitted", "plugin_approve"), "The version's license is not on this installation's list of permitted plugin licenses.");
  assert.equal(k.blockerExtra("en", "license_not_permitted"), "Unverified version, destination or permission. It cannot be activated.");
  const fixture = page({ modules: { proposals: [proposal(10, "upgrade", "pending", other, { version: "1.1.0" }, { can_approve: false, approve_blocked: "license_not_permitted" })] } });
  const p = fixture.modules.proposals![0];
  assert.equal(p.can_approve, false);
  assert.equal(p.approve_blocked, "license_not_permitted");
  assert.equal(blockerPolicy(p.approve_blocked, true), "explain");
});

test("a refusal names the license only when the target version is not permitted in the read fetched after it", () => {
  setScope("ws", me, "license-session");
  const p = page({ modules: { plugins: [version("1.0.0"), version("1.1.0"), version("1.2.0", { license: "LicenseRef-closed", license_state: "not_permitted" })] } });
  assert.equal(refusedLicense(p, "p", "upgrade", "1.2.0"), "1.2.0");
  assert.equal(refusedLicense(p, "p", "upgrade", "1.1.0"), "");
  assert.equal(refusedLicense(p, "p", "upgrade", "9.9.9"), "9.9.9");
  assert.equal(refusedLicense(p, "p", "remove", ""), "");
  assert.equal(refusedLicense(p, "p", "enable", ""), "");
  const off = page({ modules: { plugins: [version("1.0.0", { license_state: "not_permitted" })] } });
  assert.equal(refusedLicense(off, "p", "enable", ""), "1.0.0");
  assert.equal(refusedLicense(off, "p", "rollback", ""), "");
  const target = pluginTarget("p");
  pendingFor("integrations.plugin_propose", target, { kind: "upgrade", version: "1.2.0" }, "r1", () => ({ request: "q1", key: "k1" }));
  settle("integrations.plugin_propose", target, "hint");
  assert.equal(refusedNow("integrations.plugin_propose", target), true);
  assert.deepEqual(refusedValues("integrations.plugin_propose", target), { kind: "upgrade", version: "1.2.0" });
  settle("integrations.plugin_propose", target, "ok");
  assert.deepEqual(refusedValues("integrations.plugin_propose", target), {});
  settle("integrations.plugin_propose", target, "hint");
  assert.deepEqual(refusedValues("integrations.plugin_propose", target), {});
  for (const l of locales) assert.ok(k.keptLicense(l, "1.2.0", "LicenseRef-closed").includes("LicenseRef-closed"));
  forget();
});

test("license copy exists in every language and never claims a legal finding", () => {
  for (const l of locales) {
    for (const s of ["default", "start_flag"]) assert.ok(k.licensesTitle(l, s).length > 0);
    assert.notEqual(k.licensesTitle(l, "odd"), k.licensesTitle(l, "default"));
    for (const s of ["permitted", "not_permitted", "unknown"]) assert.ok(k.licenseState(l, s).length > 0);
    for (const text of [k.licensesNote(l), k.licensesEmpty(l), k.licensesUnknown(l), k.notOffered(l, "1.2.0", "X"), k.licenseUnknown(l), k.targetLicense(l, "1.0.0")]) assert.ok(text.length > 0);
  }
  assert.equal(k.licenseState("en", "not_permitted"), "Not permitted on this installation.");
  assert.equal(k.licenseConsequence("en"), "It cannot be installed, upgraded to or enabled.");
  assert.equal(k.licenseState("en", "unknown"), "Unknown. Treated as not permitted.");
  assert.equal(k.licensesTitle("en", "default"), "Permitted plugin licenses (installation default)");
  assert.equal(k.licensesTitle("en", "start_flag"), "Permitted plugin licenses (set when the server started)");
  assert.equal(k.licensesTitle("ko", "start_flag"), "허용 플러그인 라이선스(서버 시작 시 지정)");
  assert.equal(k.licensesTitle("ar", "start_flag"), "تراخيص الإضافات المسموح بها (حُدّد عند بدء الخادم)");
  assert.equal(k.licensesNote("en"), "An installation setting: it changes only when the server is restarted with a different list. This list is an installation policy, not a legal review.");
  assert.equal(k.keptLicense("en", "1.2.0", "X"), "Version ⁨1.2.0⁩ has the license ⁨X⁩, which is not permitted on this installation.");
  assert.equal(k.notOfferedUnknown("en", "1.2.0"), "Version ⁨1.2.0⁩ is not offered: its license is unknown, so it is treated as not permitted.");
  assert.equal(k.enableNotOffered("en", "1.0.0", "X"), "Enable is not offered: the installed version ⁨1.0.0⁩ has the license ⁨X⁩, which is not permitted on this installation.");
  for (const l of locales) {
    for (const text of [k.licenseConsequence(l), k.notOfferedUnknown(l, "1.2.0"), k.enableNotOffered(l, "1.0.0", "X"), k.enableNotOfferedUnknown(l, "1.0.0"), k.keptLicenseUnknown(l, "1.2.0")]) assert.ok(text.length > 0);
    for (const text of [k.notOfferedUnknown(l, "1.2.0"), k.enableNotOfferedUnknown(l, "1.0.0"), k.keptLicenseUnknown(l, "1.2.0")]) {
      assert.ok(!text.includes(k.notOffered(l, "", "").slice(-12)), l + " " + text);
    }
    assert.notEqual(k.proposeBlocker(l, "license_unknown", ""), k.proposeBlocker(l, "license", ""));
  }
  assert.ok(!k.notOfferedUnknown("en", "1.2.0").includes("is not permitted on this installation"));
  assert.ok(!k.keptLicenseUnknown("en", "1.2.0").includes("which is not permitted"));
  assert.equal(k.notOffered("en", "1.2.0", "X"), "Version ⁨1.2.0⁩ is not offered: its license ⁨X⁩ is not permitted on this installation.");
  assert.ok(!/\b(compliant|legal(ly)? (approved|cleared))\b/i.test(k.licensesNote("en")));
});

test("the modules view renders licenses, withheld versions and a license-blocked approval without an Approve button", async () => {
  const nodeModule = await import("node:module");
  const stub = /\.(css|svg)(\?.*)?$/;
  if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (stub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
  const { createElement } = await import("react");
  const { renderToStaticMarkup } = await import("react-dom/server");
  const { ModulesView } = await import("./LiveModules");
  const { keptLicenseText } = await import("./LiveProposals");
  const p = page({
    modules: {
      licenses: { permitted: ["Apache-2.0", "MIT"], source: "default" },
      plugins: [version("1.0.0"), version("1.1.0"), version("1.2.0", { license: "LicenseRef-closed", license_state: "not_permitted" })],
      proposals: [proposal(10, "upgrade", "pending", other, { version: "1.2.0" }, { can_approve: false, approve_blocked: "license_not_permitted", can_withdraw: false, withdraw_blocked: "not_proposer" })],
    },
  });
  const html = (l: "en" | "ko" | "ar", q = p) =>
    renderToStaticMarkup(createElement(ModulesView, { l, page: q, api: {} as never, me, selection: { proposal: "", subscription: "", delivery: "" }, canCommand: true, onDone: () => undefined,
      onNotice: () => undefined, onReload: async () => null, onSignedOut: () => undefined, onLostAccess: () => undefined }));
  const text = (l: "en" | "ko" | "ar", q = p) => html(l, q).replace(/<[^>]+>/g, " ").replace(/&#x27;/g, "'").replace(/\s+/g, " ");
  const en = text("en");
  const raw = html("en");
  assert.ok(en.includes("Permitted plugin licenses (installation default)"), en);
  assert.ok(en.includes("This list is an installation policy, not a legal review."));
  assert.ok(/License: .{0,12}MIT.{0,12} Permitted on this installation/.test(en));
  assert.ok(/<span class="p02-license-line">License: <bdi dir="ltr" class="live-mono">LicenseRef-closed<\/bdi><\/span><span class="p02-license-line"><strong class="p02-license-flag">Not permitted on this installation\.<\/strong> It cannot be installed, upgraded to or enabled\.<\/span>/.test(raw));
  assert.ok(/Version .{0,4}1\.2\.0.{0,4} is not offered: its license .{0,4}LicenseRef-closed.{0,4} is not permitted on this installation\./.test(en));
  assert.ok(raw.includes('<bdi dir="ltr" class="live-mono">LicenseRef-closed</bdi>') && /p02-withheld[^>]*>.*<bdi dir="ltr" class="live-mono">1\.2\.0<\/bdi>/.test(raw));
  assert.ok(/License of .{0,4}1\.2\.0.{0,4}: .{0,4}LicenseRef-closed.{0,4} Not permitted/.test(en));
  assert.ok(en.includes("The version's license is not on this installation's list of permitted plugin licenses."));
  assert.ok(en.includes("Unverified version, destination or permission. It cannot be activated."));
  assert.ok(!/>\s*Approve\s*</.test(renderToStaticMarkup(createElement(ModulesView, { l: "en", page: p, api: {} as never, me, selection: { proposal: "", subscription: "", delivery: "" }, canCommand: true,
    onDone: () => undefined, onNotice: () => undefined, onReload: async () => null, onSignedOut: () => undefined, onLostAccess: () => undefined }))));
  for (const l of ["ko", "ar"] as const) assert.ok(text(l).includes(k.licensesTitle(l, "default")));
  const empty = text("en", page({ modules: { licenses: { permitted: [], source: "start_flag" } } }));
  assert.ok(empty.includes("Permitted plugin licenses (set when the server started)") && empty.includes("No plugin license is permitted on this installation"));
  const unknownVersion = text("en", page({ modules: { plugins: [version("1.0.0"), version("1.1.0"), version("1.2.0", { license: "", license_state: "odd" })] } }));
  assert.ok(unknownVersion.includes("is not offered: its license is unknown, so it is treated as not permitted.") && unknownVersion.includes("Unknown. Treated as not permitted."));
  const stopped = text("en", page({ modules: { plugins: [version("1.0.0", { license: "LicenseRef-closed", license_state: "not_permitted" }), version("1.1.0")],
    installations: [{ plugin: "p", version: "1.0.0", state: "stopped" }], proposals: [proposal(11, "enable", "pending", other, {})] } }));
  assert.ok(/Enable is not offered: the installed version .{0,4}1\.0\.0.{0,4} has the license .{0,4}LicenseRef-closed.{0,4}, which is not permitted on this installation\./.test(stopped), stopped);
  assert.ok(/License of .{0,4}1\.0\.0.{0,4}: .{0,4}LicenseRef-closed.{0,4} Not permitted/.test(stopped));
  const closedPage = page({ modules: { plugins: [version("1.0.0"), version("1.2.0", { license: "LicenseRef-closed", license_state: "not_permitted" }), version("1.3.0", { license_state: "odd" })] } });
  const lead = "The plugin registry refused this change. Nothing changed. ";
  const closedEn = "Version ⁨1.2.0⁩ has the license ⁨LicenseRef-closed⁩, which is not permitted on this installation. ";
  const unknownEn = "The license of version ⁨1.3.0⁩ is unknown, so it is treated as not permitted. ";
  const next = {
    propose: "Choose another version; the permitted list changes only when the server is restarted with a different list.",
    enable: "Propose an upgrade or rollback to a permitted version instead.",
    approve: "This proposal cannot be approved; the proposer can withdraw it. The permitted list changes only when the server is restarted with a different list.",
  } as const;
  for (const context of ["propose", "enable", "approve"] as const) {
    assert.equal(keptLicenseText("en", closedPage, "p", "1.2.0", context), lead + closedEn + next[context]);
    assert.equal(keptLicenseText("en", closedPage, "p", "1.3.0", context), lead + unknownEn + next[context]);
    assert.ok(keptLicenseText("en", closedPage, "p", "9.9.9", context).includes("is unknown"));
    for (const l of locales) {
      const text = keptLicenseText(l, closedPage, "p", "1.2.0", context);
      assert.ok(text.startsWith(k.kept(l, "plugin")) && text.endsWith(k.keptLicenseNext(l, context)), l + " " + context);
      assert.ok(keptLicenseText(l, closedPage, "p", "1.3.0", context).endsWith(k.keptLicenseNext(l, context)));
    }
  }
  assert.equal(k.keptLicenseNext("ko", "approve"), "이 제안은 승인할 수 없습니다. 제안자가 철회할 수 있습니다. 허용 목록은 다른 목록으로 서버를 다시 시작해야 바뀝니다.");
  assert.equal(k.keptLicenseNext("ar", "approve"), "لا يمكن اعتماد هذا المقترح، ويمكن لصاحبه سحبه. لا تتغيّر القائمة المسموح بها إلا بإعادة تشغيل الخادم بقائمة مختلفة.");
  assert.equal(k.keptLicenseNext("ko", "enable"), "대신 허용된 버전으로 업그레이드 또는 롤백을 제안하세요.");
  assert.equal(k.keptLicenseNext("ar", "enable"), "اقترح بدلًا من ذلك ترقية أو إرجاعًا إلى إصدار مسموح به.");
  assert.equal(k.keptLicenseNext("ko", "propose"), "다른 버전을 선택하세요. 허용 목록은 다른 목록으로 서버를 다시 시작해야 바뀝니다.");
  assert.equal(k.keptLicenseNext("ar", "propose"), "اختر إصدارًا آخر؛ لا تتغيّر القائمة المسموح بها إلا بإعادة تشغيل الخادم بقائمة مختلفة.");
  assert.ok(new Set(["propose", "enable", "approve"].map((c) => k.keptLicenseNext("en", c))).size === 3);
  const unknown = text("en", page({ modules: { licenses: null } }));
  assert.ok(unknown.includes("The list of permitted plugin licenses is unknown"));
});

const allRights = { administer_api_clients: true, administer_modules: true, approve_modules: true, stop_modules: true, administer_channels: true, approve_channels: true, stop_channels: true, reprocess_events: true };

test("a refusal after the right is gone reads as no_right, never as gone, and only for right-checked commands", async () => {
  const { commandRight, authorOnly } = await import("./live-model");
  const lost = integrationsOf({ ...page(), rights: { ...allRights, approve_modules: false } } as unknown as Record<string, unknown>);
  lost.modules = page().modules;
  assert.equal(deniedOutcome("integrations.plugin_approve", "plugin-proposal-9", true, lost), "no_right");
  assert.equal(deniedOutcome("integrations.plugin_withdraw", "plugin-proposal-9", false, lost), "gone");
  const unknown = page();
  assert.equal(unknown.rights, null);
  assert.equal(deniedOutcome("integrations.plugin_approve", "plugin-proposal-9", false, unknown), "gone");
  assert.equal(commandRight["integrations.redelivery_reapprove"], "reprocess_events");
  assert.equal(commandRight["integrations.subscription_revoke"], "stop_channels");
  assert.ok(authorOnly("integrations.letter_withdraw_dismiss") && !authorOnly("integrations.letter_approve_dismiss"));
});

test("withheld areas, own pending items and applicability", async () => {
  const { withheld, ownPendingFor, subscriptionApplies, letterApplies, installApplies, heldRights, markLost, lostRight, pruneLost, forgetLost } = await import("./live-rights");
  const p = integrationsOf({
    revision: "r1", now: "", state: "configured", rights: { stop_modules: true, reprocess_events: true },
    clients: { state: "not_permitted" }, modules: { state: "configured" }, channels: { state: "configured" }, events: { state: "configured" },
    own_pending: [
      { kind: "plugin_proposal", id: "9", target: "plugin-proposal-9", object: "p", state: "pending" },
      { kind: "resume_request", id: "resume-request-1", target: "webhook-resume-request:resume-request-1", object: "s1", state: "pending" },
      { kind: "dismiss_request", id: "letter-request-2", target: "webhook-letter-request:letter-request-2", object: "d1", state: "pending" },
      { kind: "dismiss_request", id: "x", target: "bad target", object: "d1", state: "pending" },
    ],
  });
  assert.equal(withheld(p, "clients"), true);
  assert.equal(withheld(p, "modules"), false);
  assert.deepEqual(heldRights(p), ["stop_modules", "reprocess_events"]);
  assert.deepEqual(ownPendingFor(p, "modules").map((x) => x.id), ["9"]);
  assert.deepEqual(ownPendingFor(p, "channels").map((x) => x.id), ["resume-request-1"]);
  assert.deepEqual(ownPendingFor(p, "events").map((x) => x.id), ["letter-request-2"]);
  assert.deepEqual(ownPendingFor(p, "clients"), []);
  const s = page({ channels: { subscriptions: [sub("s1", "active", other, ["a"])] } }).channels.subscriptions?.[0];
  assert.ok(s && subscriptionApplies(s, "pause") && subscriptionApplies(s, "revoke") && !subscriptionApplies(s, "approve") && !subscriptionApplies(s, "approve_resume"));
  const d = page({ events: { deliveries: [letter("d1", { redeliver_request: { by: other } })] } }).events.deliveries?.[0];
  assert.ok(d && letterApplies(d, "approve_redeliver") && !letterApplies(d, "request_redeliver") && letterApplies(d, "request_dismiss") && !letterApplies(d, "approve_dismiss") && !letterApplies(d, "reapprove"));
  assert.ok(installApplies({ state: "stopped" } as never, "disable") && !installApplies({ state: "stopped" } as never, "stop"));
  forgetLost();
  markLost("integrations.plugin_stop", "plugin:p", "stop_modules");
  assert.equal(lostRight("integrations.plugin_stop", "plugin:p"), "stop_modules");
  pruneLost(integrationsOf({ rights: { stop_modules: false } }));
  assert.equal(lostRight("integrations.plugin_stop", "plugin:p"), "stop_modules");
  pruneLost(p);
  assert.equal(lostRight("integrations.plugin_stop", "plugin:p"), "");
});

test("rights copy in EN, KO and AR", async () => {
  const { r, rightLabel, rightList } = await import("./rights-copy");
  const { blockerLines } = await import("./LiveCommand");
  assert.equal(rightList("en", ["reprocess_events", "stop_modules", "stop_channels"]), "module stop; subscription and channel stop; event reprocessing");
  assert.equal(rightList("en", ["administer_api_clients", "stop_modules"]), "API client administration; module stop");
  assert.equal(rightList("ar", ["stop_modules", "stop_channels", "reprocess_events"]), "إيقاف الوحدات، إيقاف الاشتراكات والقنوات، إعادة معالجة الأحداث");
  assert.equal(rightList("ko", ["stop_modules", "stop_channels", "reprocess_events"]), "모듈 중지, 구독·채널 중지, 이벤트 재처리");
  assert.equal(r.noRight("en", "approve_modules"), "Needs the module approval right. You don't hold it.");
  assert.equal(r.noRight("en", "administer_api_clients"), "Needs the API client administration right. You don't hold it.");
  assert.equal(r.noRight("ko", "approve_modules"), "모듈 승인 권한이 필요합니다. 현재 이 권한이 없습니다.");
  assert.equal(r.noLongerHeld("en", "reprocess_events"), "You no longer hold the event reprocessing right. Nothing was changed.");
  assert.equal(r.withdrawName("en", r.pendingKind("en", "plugin_proposal"), "9"), "Withdraw plugin change proposal 9");
  assert.equal(r.held("ar"), "محتجَز");
  assert.notEqual(r.held("ar"), r.pendingTag("ar"));
  assert.ok(r.heldStatus("ar").startsWith("محتجَز:") && r.heldStatus("en").includes("event reprocessing right"));
  assert.ok(r.againImpact("ar").includes("المُرسِل غير متصل"));
  assert.equal(r.pendingTag("ko"), "승인 대기");
  assert.equal(r.forTarget("ar"), "الهدف");
  assert.equal(r.closedAutomatically("ko"), "자동 종료");
  assert.equal(r.loadFailed("en"), "The integrations could not be loaded.");
  assert.equal(blockerLines("en", "no_right", "plugin_approve", "Approve")[0].text, "Needs the module approval right. You don't hold it.");
  assert.equal(blockerLines("en", "no_right", "approve_redeliver", "Approve redelivery").length, 1);
  assert.equal(blockerLines("en", "requester_lost_right", "approve", "Approve")[0].text, "The person who started this no longer holds the right, so it can't be approved.");
  assert.equal(r.closedLost("en", "Proposal 8"), "Proposal 8: Not approved. It was closed because the person who started it no longer holds the right.");
  for (const l of locales) {
    for (const id of ["administer_api_clients", "administer_modules", "approve_modules", "stop_modules", "administer_channels", "approve_channels", "stop_channels", "reprocess_events"] as const) {
      assert.ok(rightLabel(l, id).length > 0 && !rightLabel(l, id).includes("_"), l + id);
    }
    for (const text of [r.closedLost(l, "x"), r.noLongerHeld(l, "stop_modules"), r.withheld(l, "clients"), r.rightsUnknown(l), r.starterLost(l), r.heldStatus(l)]) {
      assert.ok(text.length > 0);
      assert.ok(!/confidential|secret|protected|기밀|비밀|سري/i.test(text), text);
    }
    assert.ok(!r.closedLost(l, "x").includes(r.lastActionLost(l).split(".")[1] ?? "###"), l);
  }
  assert.ok(!r.closedLost("en", "x").includes("Nothing was changed"));
  assert.notEqual(r.rightsUnknown("en"), r.rightsNoneLead("en") + " " + r.rightsNoneRest("en"));
});

test("an operator without approval and administration rights sees reasons instead of buttons", async () => {
  const { createElement } = await import("react");
  const { renderToStaticMarkup } = await import("react-dom/server");
  const { ModulesView } = await import("./LiveModules");
  const { ChannelsView } = await import("./LiveChannels");
  const p = page({
    modules: {
      installations: [{ plugin: "p", version: "1.0.0", state: "enabled", can_stop: true, stop_blocked: "", can_disable: true, disable_blocked: "" }],
      proposals: [proposal(10, "upgrade", "pending", other, { version: "1.1.0" }, { can_approve: false, approve_blocked: "no_right", can_withdraw: false, withdraw_blocked: "not_proposer" })],
      actions: { propose: { allowed: false, blocked: "no_right" } },
    },
    channels: {
      subscriptions: [sub("s1", "active", other, ["a"], { actions: { approve: avail("no_right"), pause: avail(), request_resume: avail("no_right"), approve_resume: avail("no_right"), revoke: avail() } })],
      actions: { propose: { allowed: false, blocked: "no_right" } },
    },
  });
  const shared = { l: "en" as const, page: p, api: {} as never, me, selection: { proposal: "", subscription: "", delivery: "" }, canCommand: true, onDone: () => undefined,
    onNotice: () => undefined, onReload: async () => null, onSignedOut: () => undefined, onLostAccess: () => undefined };
  const modules = renderToStaticMarkup(createElement(ModulesView, shared)).replace(/<[^>]+>/g, " ").replace(/&#x27;/g, "'").replace(/\s+/g, " ");
  assert.ok(modules.includes("Approve: Needs the module approval right. You don't hold it."), modules);
  assert.ok(modules.includes("Propose a change: Needs the module administration right. You don't hold it."));
  assert.ok(!modules.includes("You proposed this"));
  assert.ok(/>\s*Stop\s*</.test(renderToStaticMarkup(createElement(ModulesView, shared))));
  const channels = renderToStaticMarkup(createElement(ChannelsView, { ...shared, section: "configured" })).replace(/<[^>]+>/g, " ").replace(/&#x27;/g, "'").replace(/\s+/g, " ");
  assert.ok(!channels.includes("Approve: Needs"), channels);
  assert.ok(!channels.includes("Approve resume: Needs"));
  assert.ok(channels.includes("Needs the subscription and channel administration right. You don't hold it."), channels);
  assert.ok(!channels.includes("Propose a subscription: Needs"), "the label repeats under its own heading");
  const readOnly = renderToStaticMarkup(createElement(ChannelsView, { ...shared, canCommand: false, page: page(), section: "configured" }));
  assert.ok(!readOnly.includes("p02-propose-sub"), "an empty propose heading was rendered");
});

test("a closure is credited to nobody and the trigger only appears inside the found-when line", async () => {
  const { createElement } = await import("react");
  const { renderToStaticMarkup } = await import("react-dom/server");
  const { ModulesView } = await import("./LiveModules");
  const { ChannelsView } = await import("./LiveChannels");
  const p = page({
    modules: { proposals: [proposal(11, "remove", "closed", other, {}, { closing_reason: "requester_lost_right", closed_request: "req-77", closure_trigger: "u9", closure_trigger_login: "nine@example.invalid", closed_by: "" })] },
    channels: { subscriptions: [sub("s1", "revoked", other, ["a"], { reason: "requester_lost_right", history: [{ action: "proposal_closed", actor: "u9", actor_login: "nine@example.invalid", reason: "requester_lost_right", request: "req-78" }] })] },
  });
  const shared = { l: "en" as const, page: p, api: {} as never, me, selection: { proposal: "", subscription: "", delivery: "" }, canCommand: true, onDone: () => undefined,
    onNotice: () => undefined, onReload: async () => null, onSignedOut: () => undefined, onLostAccess: () => undefined };
  const modules = renderToStaticMarkup(createElement(ModulesView, shared));
  assert.ok(modules.replace(/<[^>]+>/g, "").includes("Found when nine@example.invalid tried to approve (request req-77)."), modules);
  assert.ok(modules.includes('<bdi dir="ltr" class="live-mono">req-77</bdi>') && modules.includes('<bdi dir="ltr" class="live-mono">nine@example.invalid</bdi>'));
  assert.ok(!modules.includes("Closed by") && !modules.includes("Approved by ⁨nine"));
  const channels = renderToStaticMarkup(createElement(ChannelsView, { ...shared, section: "configured" }));
  assert.ok(channels.includes("Closed automatically") && /req-78<\/bdi>/.test(channels), channels);
});

test("lost-right marks are cleared when the signed-in user changes", async () => {
  const { markLost, lostRight } = await import("./live-rights");
  setScope("w1", "u1", "s1");
  markLost("integrations.plugin_stop", "plugin:p", "stop_modules");
  setScope("w1", "u1", "s1");
  assert.equal(lostRight("integrations.plugin_stop", "plugin:p"), "stop_modules");
  setScope("w1", "u2", "s1");
  assert.equal(lostRight("integrations.plugin_stop", "plugin:p"), "");
});


test("retained diagnostics bind every grouped action to its exact target and permission-loss marker", async () => {
  const { feedbackDiagnostics } = await import("./live-model");
  const { markLost, pruneLost, lostFor } = await import("./live-rights");
  forget(); setScope("ws", me, "s");
  const stop: [CommandKey, string] = ["integrations.plugin_stop", "plugin:p"];
  const disable: [CommandKey, string] = ["integrations.plugin_disable", "plugin:p"];
  const reference = ([command, target]: [CommandKey, string], code: string, request: string) => ({ command, detail: { code, request, target, client: "", cancelled: false } });
  const ids = (choices: Array<[CommandKey, string]>) => feedbackDiagnostics(choices).map((v) => v.detail.request);
  settle(...stop, "hint", reference(stop, "invalid_stage", "stop-response"));
  settle(...disable, "hint", reference(disable, "prerequisite_failed", "disable-response"));
  assert.deepEqual(ids([stop, disable]), ["stop-response", "disable-response"]);
  settle(...stop, "ok");
  assert.deepEqual(ids([stop, disable]), ["disable-response"]);
  markLost(...stop, "stop_modules", reference(stop, "permission_denied", "denied-response"));
  assert.deepEqual(ids([stop, disable]), ["denied-response", "disable-response"]);
  assert.equal(lostFor([stop]), "stop_modules");
  pruneLost({ ...page(), rights: { stop_modules: true } } as IntegrationsPage);
  assert.deepEqual(ids([stop, disable]), ["disable-response"]);
  assert.deepEqual(ids([[stop[0], "plugin:another"]]), []);
  pruneRefusals(page({ modules: { plugins: [], installations: [] } }), me);
  assert.deepEqual(ids([disable]), []);
  const resume: [CommandKey, string] = ["integrations.subscription_withdraw_resume", "webhook-resume-request:request-1"];
  const dismiss: [CommandKey, string] = ["integrations.letter_withdraw_dismiss", "webhook-letter-request:request-2"];
  settle(...resume, "hint", reference(resume, "invalid_stage", "resume-response"));
  markLost(...dismiss, "reprocess_events", reference(dismiss, "permission_denied", "dismiss-response"));
  assert.deepEqual(ids([resume, dismiss]), ["resume-response", "dismiss-response"]);
  assert.equal(lostFor([[dismiss[0], "webhook-letter-request:wrong"], dismiss]), "reprocess_events");
  setScope("ws", me, "new-session");
  assert.deepEqual(ids([stop, disable, resume, dismiss]), []);
});

test("the delivery list is called cut only for a subscription the service names, and unknown when it does not say", () => {
  const named = page({ events: { deliveries_more: ["s1"] } });
  assert.deepEqual([deliveriesCut(named, "s1"), deliveriesCut(named, "s2"), deliveriesCut(named, "")], ["cut", "complete", "cut"]);
  const none = page({ events: { deliveries_more: [] } });
  assert.deepEqual([deliveriesCut(none, "s1"), deliveriesCut(none, "")], ["complete", "complete"]);
  assert.equal(deliveriesCut(integrationsOf({}), "s1"), "unknown");
  assert.equal(integrationsOf({ events: { deliveries_more: "s1" } }).events.deliveries_more, null);
});
