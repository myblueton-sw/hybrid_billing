import { currentScope } from "./live-model";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import type { AdminClient } from "../../../../service/admin";
import { refusalDiagnostics } from "./live-model";
import type { RowDiagnostic } from "../../../../service/command-diagnostics";
import { test } from "node:test";
import assert from "node:assert/strict";
import { installationOf, proposalOf, type InstallationPage } from "../../../../service/installation";
import {
  clearDraft, commandTarget, endpoint, notApplicableOf, showsEffects, discard, draftOf, duplicateOf, forget, hashChanged, keepDraft, keepsAttempt, knownApproveBlockers, latestAcceptance, listCount, observationBlocks,
  ordered, pendingFor, pendingNow, planChangedField, planTarget, previewBasis, proposeState, pruneRefusals, react, reasonTooLong, refusedNow, reportPassed, setScope,
  settle as settleRecorded, summaryParts, timeKnown, utf8Bytes, viewOf, withdrawSentence,
} from "./live-model";
import { k } from "./live-copy";
import { actionWords, artifactWords, causeWords, checkWords, conflictWords, effectWords, failureWords, permissionWords, proposalKindWords, stepWords, type Words } from "./live-codes";

const hash = "b".repeat(64);
const other = "c".repeat(64);
const locales = ["en", "ko", "ar"] as const;

function page(over: Record<string, unknown> = {}): InstallationPage {
  return installationOf({
    revision: "r1",
    now: "2026-11-17T09:00:00Z",
    state: "configured",
    preview: { state: "available", kind: "install", version: "1.0.0", hash, services: [], ports: [], volumes: [], data_effects: [], permissions: [], leftovers: [], conflicts: [] },
    proposals: [],
    ...over,
  });
}

const plan = (n: number, state: string, proposer: string, h = hash, version = "1.0.0") =>
  ({ number: n, target: "installation-proposal-" + n, kind: "plan", state, proposer, summary: { plan_version: version, plan_hash: h }, can_approve: false, approve_blocked: "own_proposal", can_withdraw: true, withdraw_blocked: "" });

test("views fall back to the plan", () => {
  assert.equal(viewOf("diagnostics"), "diagnostics");
  assert.equal(viewOf("support"), "support");
  assert.equal(viewOf("nope"), "plan");
});

test("propose is offered only for an available, conflict-free, hashed and unaccepted plan", () => {
  assert.deepEqual(proposeState(page(), "u1"), { offered: true, blocker: "", stale: null });
  assert.equal(proposeState(page({ preview: { state: "unavailable", reason: "not_observed", version: "1.0.0" } }), "u1").blocker, "unavailable");
  assert.equal(proposeState(page({ preview: { state: "not_applicable" } }), "u1").blocker, "not_applicable");
  assert.equal(proposeState(page({ preview: { state: "odd", version: "1.0.0" } }), "u1").blocker, "unknown_state");
  assert.equal(proposeState(page({ preview: { state: "available", version: "", hash } }), "u1").blocker, "unknown_state");
  assert.equal(proposeState(page({ preview: { state: "available", version: "1.0.0", hash } }), "u1").blocker, "conflicts_unknown");
  assert.equal(proposeState(page({ preview: { state: "available", version: "1.0.0", hash, conflicts: [{ target: "x", conflict: "port_in_use" }] } }), "u1").blocker, "conflicts");
  assert.equal(proposeState(page({ preview: { state: "available", version: "1.0.0", hash: "ABC", conflicts: [] } }), "u1").blocker, "hash");
  assert.equal(proposeState(page({ preview: { state: "available", version: "1.0.0", hash: hash.toUpperCase(), conflicts: [] } }), "u1").blocker, "hash");
  assert.equal(proposeState(page({ proposals: [plan(3, "applied", "u2")] }), "u1").blocker, "accepted");
  assert.equal(proposeState(page({ proposals: [plan(3, "applied", "u2", hash, "1.1.0")] }), "u1").offered, true);
  assert.equal(proposeState(page({ proposals: [plan(4, "pending", "u1")] }), "u1").blocker, "waiting");
  assert.equal(proposeState(page({ proposals: [plan(4, "pending", "u2")] }), "u1").offered, true);
  assert.equal(proposeState(page({ proposals: [plan(4, "withdrawn", "u1")] }), "u1").offered, true);
  const stale = proposeState(page({ proposals: [plan(4, "pending", "u1", other)] }), "u1");
  assert.equal(stale.offered, true);
  assert.equal(stale.stale?.number, 4);
  for (const l of locales) {
    for (const b of ["unavailable", "not_applicable", "unknown_state", "conflicts", "conflicts_unknown", "hash", "accepted", "waiting"]) assert.ok(k.proposeBlocked(l, b, "1.0.0").length > 0);
    assert.ok(k.proposeBlocked("en", "waiting", "1.0.0").includes("already waiting"));
    assert.ok(k.staleOwn(l).length > 0);
  }
});

test("the previewed version states why it was chosen and the latest acceptance", () => {
  assert.equal(previewBasis(page({ preview: { state: "not_applicable" } })), "");
  assert.equal(previewBasis(page({ preview_basis: "newest_release" })), "release");
  assert.equal(previewBasis(page({ preview_basis: "pending_plan" })), "pending");
  assert.equal(previewBasis(page({ preview_basis: "newest_release", proposals: [plan(4, "pending", "u2")] })), "release", "the served basis is used, not a guess from the listed proposals");
  assert.equal(previewBasis(page({ preview_basis: "pending_plan", proposals: [] })), "pending", "a pending plan outside the listed proposals is still the basis");
  assert.equal(previewBasis(page()), "", "without a served basis nothing is claimed");
  const list = page({ proposals: [plan(2, "applied", "u2"), plan(6, "applied", "u2", hash, "1.1.0"), plan(7, "pending", "u1")] }).proposals;
  assert.equal(latestAcceptance(list)?.number, 6);
  assert.equal(latestAcceptance([]), null);
  assert.ok(k.accepted("en", "admin")[1].includes("Not installed"));
});

test("proposals list pending first then closed, newest first", () => {
  const list = page({ proposals: [plan(1, "applied", "u"), plan(5, "pending", "u"), plan(3, "pending", "u"), plan(4, "withdrawn", "u"), { kind: "key", state: "pending" }] }).proposals;
  const o = ordered(list);
  assert.deepEqual(o.pending.map((p) => p.number), [5, 3, null]);
  assert.deepEqual(o.closed.map((p) => p.number), [4, 1]);
});

test("every proposal kind has a summary and unknown kinds fall back", () => {
  const p = (kind: string, summary: Record<string, unknown>) => proposalOf({ number: 1, target: "installation-proposal-1", kind, summary });
  assert.deepEqual(summaryParts(p("key", { key_version: "k1" })).values, ["k1"]);
  assert.deepEqual(summaryParts(p("release", { release_version: "1.1.0" })).values, ["1.1.0"]);
  for (const kind of ["site", "probe", "config"]) assert.deepEqual(summaryParts(p(kind, { site: "s1" })).values, ["s1"]);
  assert.deepEqual(summaryParts(p("destination", { host: "h.example.invalid", port: 443, ranges: ["198.51.100.0/24", "192.0.2.0/24"] })).values, ["h.example.invalid", "443", "198.51.100.0/24, 192.0.2.0/24"]);
  assert.deepEqual(summaryParts(p("destination", { host: "h" })).values, ["h", "", ""]);
  for (const kind of ["plan", "upgrade", "rollback"]) assert.deepEqual(summaryParts(p(kind, { plan_version: "1.0.0" })).values, ["1.0.0"]);
  assert.deepEqual(summaryParts(p("mystery", {})), { kind: "mystery", known: false, values: [] });
  for (const kind of ["key", "release", "site", "probe", "config", "destination", "plan", "upgrade", "rollback"]) {
    assert.equal(summaryParts(p(kind, {})).known, true);
    for (const l of locales) {
      assert.notEqual(k.kind(l, kind), k.unknownCode(l, kind));
      assert.notEqual(k.summaryLead(l, kind), k.unknownCode(l, kind));
    }
  }
  assert.equal(k.kind("en", "mystery"), "Unknown (mystery)");
  assert.equal(k.state("en", "odd"), "Unknown (odd)");
});

test("blocker sentences cover every known code and unknown codes fall back", () => {
  for (const l of locales) {
    const generic = k.approveBlocked(l, "zzz");
    for (const b of knownApproveBlockers) assert.notEqual(k.approveBlocked(l, b), generic, b + l);
    assert.notEqual(k.approveBlocked(l, "already_installed"), generic);
    assert.equal(k.approveBlocked(l, "missing"), generic);
  }
  assert.equal(k.approveBlocked("en", "own_proposal"), "You proposed this. Another administrator must approve it.");
  assert.equal(k.approveBlocked("en", "zzz"), "Approval is not available for this proposal now.");
  assert.equal(withdrawSentence(proposalOf({ can_withdraw: false, withdraw_blocked: "not_proposer" })), false);
  assert.equal(withdrawSentence(proposalOf({ can_withdraw: false, withdraw_blocked: "not_pending" })), false);
  assert.equal(withdrawSentence(proposalOf({ can_withdraw: true, withdraw_blocked: "" })), false);
  assert.equal(withdrawSentence(proposalOf({})), true);
});

test("plan acceptances show the current hash only when it differs", () => {
  assert.equal(hashChanged(proposalOf({ summary: { plan_hash: hash }, current_hash: "" })), false);
  assert.equal(hashChanged(proposalOf({ summary: { plan_hash: hash }, current_hash: hash })), false);
  assert.equal(hashChanged(proposalOf({ summary: { plan_hash: hash }, current_hash: other })), true);
});

test("commands use the row target only when it is well formed and matches the number", () => {
  assert.equal(commandTarget(proposalOf({ number: 12, target: "installation-proposal-12" })), "installation-proposal-12");
  assert.equal(commandTarget(proposalOf({ number: 12, target: "installation-proposal-13" })), "");
  assert.equal(commandTarget(proposalOf({ number: 12, target: "installation-proposal-012" })), "");
  assert.equal(commandTarget(proposalOf({ number: 12, target: "proposal-12" })), "");
  assert.equal(commandTarget(proposalOf({ target: "installation-proposal-12" })), "");
  assert.equal(commandTarget(proposalOf({ number: 12 })), "");
});

test("observation times of year 1 or empty read unknown", async () => {
  const { stamp } = await import("./live-format");
  assert.equal(timeKnown("0001-01-01T00:00:00Z"), false);
  assert.equal(timeKnown(""), false);
  assert.equal(timeKnown("garbage"), false);
  assert.equal(timeKnown("2026-11-17T09:00:00Z"), true);
  assert.equal(stamp("en", "0001-01-01T00:00:00Z"), "Unknown");
  assert.equal(stamp("ko", ""), "알 수 없음");
  assert.ok(stamp("en", "2026-11-17T09:00:00Z").endsWith(" UTC"));
  assert.equal(observationBlocks("stale"), true);
  assert.equal(observationBlocks("missing"), true);
  assert.equal(observationBlocks("fresh"), false);
  assert.equal(observationBlocks("unknown"), false);
  for (const l of locales) {
    for (const s of ["fresh", "stale", "missing", "unknown"]) assert.notEqual(k.obsTag(l, s), k.unknownCode(l, s));
    assert.ok(k.obsMissing(l) && k.obsUnknown(l) && k.obsRestart(l) && k.obsBlocks(l));
  }
});

test("missing lists read unknown and empty lists read none, never zero", () => {
  assert.equal(listCount(null), "unknown");
  assert.equal(listCount([]), "none");
  assert.equal(listCount([1, 2]), 2);
});

test("reports read passed, not resolved and direct", () => {
  assert.equal(reportPassed("ok"), true);
  assert.equal(reportPassed("unknown_authority"), false);
  assert.equal(k.failure("en", "ok"), "Passed – no action");
  assert.equal(k.notResolved("en"), "Not resolved");
  assert.equal(k.direct("en"), "Direct");
});

test("every module code has copy in every language and none suggests turning off verification", () => {
  const all: [Words, string[]][] = [
    [checkWords, ["dns", "tcp", "proxy", "tls"]],
    [failureWords, ["ok", "unknown", "unknown_authority", "hostname_mismatch", "expired", "not_yet_valid", "weak_signature", "proxy_auth", "proxy_refused", "dns_failure", "refused", "timeout"]],
    [causeWords, ["none", "unclassified", "untrusted_issuer", "certificate_name", "validity_window", "weak_algorithm", "proxy_credentials", "proxy_policy", "name_resolution", "network_path"]],
    [stepWords, ["collect_details", "add_internal_ca_to_product_trust", "verify_chain", "use_certificate_hostname", "reissue_certificate", "check_clock", "renew_certificate", "configure_proxy_credentials", "configure_proxy_allowlist", "check_dns", "check_firewall", "check_service"]],
    [conflictWords, ["port_moved", "volume_remapped", "data_volume_removed", "owned_missing", "service_exists", "port_in_use", "port_duplicate", "existing_volume", "volume_duplicate", "config_mismatch"]],
    [actionWords, ["create", "bind", "replace", "remove", "reuse", "retain", "detach", "reattached", "adopt_created", "restart", "leftover"]],
    [effectWords, ["new", "untouched", "retained", "stale_content"]],
    [permissionWords, ["run_service", "write_config", "listen_port", "create_volume"]],
    [artifactWords, ["container_image", "archive", "package"]],
    [proposalKindWords, ["key", "release", "site", "probe", "config", "destination", "plan", "upgrade", "rollback"]],
  ];
  for (const [table, codes] of all) for (const c of codes) {
    assert.ok(table[c], c);
    for (const w of table[c]) assert.ok(w.length > 0, c);
  }
  for (const words of Object.values(stepWords)) assert.doesNotMatch(words[0], /disable|turn off|skip|ignore/i);
  assert.equal(k.step("en", "zap"), "Unknown (zap)");
  assert.equal(k.cause("ar", "zap"), "غير معروف (zap)");
});

test("failure reactions keep the attempt only for an unknown outcome", () => {
  assert.equal(react("invalid", ["hash"]), "field");
  assert.equal(react("invalid", []), "reason");
  assert.equal(react("stage", []), "hint");
  assert.equal(react("refused", []), "hint");
  assert.equal(react("conflict", []), "conflict");
  assert.equal(react("denied", []), "denied");
  assert.equal(react("unauthenticated", []), "signin");
  assert.equal(react("limited", []), "wait");
  assert.equal(react("busy", []), "retry");
  assert.equal(react("unavailable", []), "retry");
  assert.equal(react("stale", []), "renew");
  assert.equal(react("partial", []), "partial");
  assert.equal(react("weird", []), "failed");
  assert.equal(keepsAttempt("retry"), true);
  for (const r of ["field", "reason", "hint", "conflict", "denied", "signin", "wait", "renew", "partial", "failed"] as const) assert.equal(keepsAttempt(r), false);
  assert.equal(planChangedField(["version"]), true);
  assert.equal(planChangedField(["hash"]), true);
  assert.equal(planChangedField(["rationale"]), false);
});

test("reasons are measured in UTF-8 bytes", () => {
  assert.equal(utf8Bytes("abc"), 3);
  assert.equal(utf8Bytes("가"), 3);
  assert.equal(utf8Bytes("م"), 2);
  assert.equal(reasonTooLong("a".repeat(256)), false);
  assert.equal(reasonTooLong("가".repeat(86)), true);
});

test("attempts are reused only for identical values and scoped by session", () => {
  forget();
  setScope("ws", "u1", "s1");
  let n = 0;
  const fresh = () => ({ request: "r" + ++n, key: "k" + n });
  const a = pendingFor("installation.approve", "installation-proposal-1", { rationale: "x" }, "rev1", fresh);
  const b = pendingFor("installation.approve", "installation-proposal-1", { rationale: "x" }, "rev2", fresh);
  assert.deepEqual(b, a);
  assert.equal(b.expected, "rev1");
  const c = pendingFor("installation.approve", "installation-proposal-1", { rationale: "y" }, "rev2", fresh);
  assert.notEqual(c.key, a.key);
  settle("installation.approve", "installation-proposal-1", "retry");
  assert.ok(pendingNow("installation.approve", "installation-proposal-1"));
  discard("installation.approve", "installation-proposal-1", { rationale: "y" });
  assert.ok(pendingNow("installation.approve", "installation-proposal-1"));
  discard("installation.approve", "installation-proposal-1", { rationale: "z" });
  assert.equal(pendingNow("installation.approve", "installation-proposal-1"), undefined);
  pendingFor("installation.propose_plan", planTarget, { version: "1.0.0", hash, rationale: "x" }, "rev1", fresh);
  settle("installation.propose_plan", planTarget, "ok");
  assert.equal(pendingNow("installation.propose_plan", planTarget), undefined);
  pendingFor("installation.withdraw", "installation-proposal-2", { rationale: "x" }, "rev1", fresh);
  setScope("ws", "u1", "s2");
  assert.equal(pendingNow("installation.withdraw", "installation-proposal-2"), undefined);
  keepDraft("installation.withdraw", "installation-proposal-2", { rationale: "draft" });
  assert.deepEqual(draftOf("installation.withdraw", "installation-proposal-2"), { rationale: "draft" });
  clearDraft("installation.withdraw", "installation-proposal-2");
  assert.deepEqual(draftOf("installation.withdraw", "installation-proposal-2"), {});
  keepDraft("installation.withdraw", "installation-proposal-2", { rationale: "draft" });
  forget();
  assert.deepEqual(draftOf("installation.withdraw", "installation-proposal-2"), {});
});

test("a refusal sentence is kept until a definite answer or the row shows a blocker or closes", () => {
  forget();
  setScope("ws", "u1", "s1");
  const t = "installation-proposal-9";
  const row = (over: Record<string, unknown>) => ({ number: 9, target: t, kind: "destination", state: "pending", can_approve: true, approve_blocked: "", can_withdraw: false, withdraw_blocked: "not_proposer", ...over });
  settle("installation.approve", t, "hint");
  assert.equal(refusedNow("installation.approve", t), true);
  pruneRefusals(page({ proposals: [row({})] }), "u1");
  assert.equal(refusedNow("installation.approve", t), true);
  settle("installation.approve", t, "retry");
  assert.equal(refusedNow("installation.approve", t), true);
  settle("installation.approve", t, "ok");
  assert.equal(refusedNow("installation.approve", t), false);
  settle("installation.approve", t, "hint");
  settle("installation.approve", t, "conflict");
  assert.equal(refusedNow("installation.approve", t), false);
  settle("installation.approve", t, "hint");
  pruneRefusals(page({ proposals: [row({ can_approve: false, approve_blocked: "conflicts" })] }), "u1");
  assert.equal(refusedNow("installation.approve", t), false);
  settle("installation.approve", t, "hint");
  pruneRefusals(page({ proposals: [row({ state: "applied" })] }), "u1");
  assert.equal(refusedNow("installation.approve", t), false);
  settle("installation.approve", t, "hint");
  pruneRefusals(page({ proposals: [] }), "u1");
  assert.equal(refusedNow("installation.approve", t), false);
  settle("installation.propose_plan", planTarget, "hint");
  pruneRefusals(page(), "u1");
  assert.equal(refusedNow("installation.propose_plan", planTarget), true);
  pruneRefusals(page({ proposals: [plan(4, "pending", "u1")] }), "u1");
  assert.equal(refusedNow("installation.propose_plan", planTarget), false);
  for (const l of locales) {
    assert.ok(k.keptApprove(l).length > 0 && k.keptPropose(l).length > 0);
    assert.notEqual(k.gone(l), k.goneProposal(l));
  }
  assert.equal(k.gone("en"), "This proposal is no longer available.");
  assert.equal(k.goneProposal("en"), "The plan can no longer be proposed here.");
  forget();
});

test("a duplicate is announced only when the service says the answer is an existing proposal", () => {
  assert.equal(duplicateOf("true"), true);
  assert.deepEqual([duplicateOf(undefined), duplicateOf(""), duplicateOf("false"), duplicateOf("TRUE")], [false, false, false, false]);
  assert.equal(k.duplicate("en", "4"), "This plan was already proposed as #4. No new proposal was created, and that proposal keeps its original reason.");
});

test("conflicts and effects show only for an available preview", () => {
  assert.equal(showsEffects("available"), true);
  for (const s of ["unavailable", "not_applicable", "", "odd"]) assert.equal(showsEffects(s), false);
});

test("not applicable previews say why from the installed state", () => {
  assert.equal(notApplicableOf("install", "installed"), "no_release");
  assert.equal(notApplicableOf("upgrade", "not_installed"), "nothing_installed");
  assert.equal(notApplicableOf("rollback", "not_installed"), "nothing_installed");
  assert.equal(notApplicableOf("upgrade", "installed"), "no_newer");
  assert.equal(notApplicableOf("rollback", "upgrade_failed"), "no_earlier");
  for (const l of locales) {
    const all = ["no_release", "nothing_installed", "no_newer", "no_earlier"].map((w) => k.notApplicable(l, w));
    assert.equal(new Set(all).size, 4);
  }
  assert.equal(k.notApplicable("en", "no_newer"), "Not applicable: no other release to upgrade to");
  assert.equal(k.notApplicable("en", "no_earlier"), "Not applicable: no earlier release to roll back to");
});

test("a destination endpoint is one string with unknown parts named", () => {
  assert.equal(endpoint("h.example.invalid", "443", "Unknown"), "h.example.invalid:443");
  assert.equal(endpoint("h", "", "Unknown"), "h:Unknown");
  assert.equal(endpoint("", "", "غير معروف"), "غير معروف:غير معروف");
  assert.notEqual(k.form.checkReason("en"), k.form.rationaleLong("en"));
});

test("a step-up refusal or cancellation keeps the attempt and the form", () => {
  for (const f of ["reauth", "reauth_cancelled"]) {
    assert.equal(react(f, []), "reauth");
    assert.equal(keepsAttempt(react(f, [])), true);
  }
});


test("retained refusal metadata follows command, target, session and pruning", () => {
  forget();
  setScope("w", "u", "s");
  const approve = { code: "invalid_stage", request: "server-approve" };
  const withdraw = { code: "prerequisite_failed", request: "server-withdraw" };
  settle("installation.approve", "missing-target", "hint", approve);
  settle("installation.withdraw", "missing-target", "hint", withdraw);
  assert.deepEqual(refusedDetail("installation.approve", "missing-target"), approve);
  assert.deepEqual(refusedDetail("installation.withdraw", "missing-target"), withdraw);
  assert.equal(refusedDetail("installation.approve", "other-target"), undefined);
  settle("installation.approve", "missing-target", "ok");
  assert.equal(refusedDetail("installation.approve", "missing-target"), undefined);
  assert.deepEqual(refusedDetail("installation.withdraw", "missing-target"), withdraw);
  setScope("w", "u", "next-session");
  assert.equal(refusedDetail("installation.withdraw", "missing-target"), undefined);
  settle("installation.approve", "missing-target", "hint", approve);
  pruneRefusals(page(), "u");
  assert.equal(refusedDetail("installation.approve", "missing-target"), undefined);
  settle("installation.approve", "missing-target", "hint", approve);
  settle("installation.approve", "missing-target", "hint");
  assert.equal(refusedDetail("installation.approve", "missing-target"), undefined);
  forget();
});


test("grouped retained note displays the matching server receipt in every locale", async () => {
  const nodeModule = await import("node:module");
  const assets = /\.(css|svg)(\?.*)?$/;
  if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => assets.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context) });
  else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assets}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
  const { Proposals } = await import("./LiveProposals");
  forget();
  setScope("w", "u", "s");
  settle("installation.approve", "installation-proposal-1", "hint", { code: "invalid_stage", request: "server-approve" });
  settle("installation.withdraw", "installation-proposal-1", "hint", { code: "prerequisite_failed", request: "server-withdraw" });
  const p = page({ proposals: [plan(1, "pending", "u")] });
  for (const l of locales) {
    const h = renderToStaticMarkup(createElement(Proposals, {
      l, page: p, api: {} as AdminClient, me: "u", selected: "", stale: false, onDenied: () => {}, onReload: async () => true,
      onDone: () => {}, onNotice: () => {}, onSignedOut: () => {},
    }));
    assert.equal(/<button[^>]*disabled/.test(h), false, l);
    const kept = renderToStaticMarkup(createElement(Proposals, {
      l, page: p, api: {} as AdminClient, me: "u", selected: "", stale: true, onDenied: () => {}, onReload: async () => true,
      onDone: () => {}, onNotice: () => {}, onSignedOut: () => {},
    }));
    const buttons = kept.match(/<button[^>]*>/g) ?? [];
    assert.ok(buttons.length > 0 && buttons.every(b => b.includes("disabled") && b.includes('aria-describedby="p01-stale"')), l + " " + buttons.join(" "));
    const { CommandForm } = await import("./LiveProposals");
    const form = renderToStaticMarkup(createElement(CommandForm, {
      l, page: p, api: {} as AdminClient, me: "u", selected: "", stale: true, onDenied: () => {}, onReload: async () => true,
      onDone: () => {}, onNotice: () => {}, onSignedOut: () => {}, onClose: () => {},
      command: "installation.approve", target: "installation-proposal-1", values: {}, title: "t", facts: null, impact: "i", verb: "v", done: () => "", gone: "g",
    } as Parameters<typeof CommandForm>[0]));
    const submit = (form.match(/<button[^>]*type="submit"[^>]*>/g) ?? []);
    assert.ok(submit.length === 1 && submit[0].includes("disabled") && submit[0].includes('aria-describedby="p01-stale"'), l + " " + submit.join(" "));
    assert.ok(h.includes("server-approve"), l);
    assert.ok(h.includes("server-withdraw") && h.includes("installation.withdraw"), l);
    assert.match(h, /<bdi[^>]*dir="ltr"[^>]*>invalid_stage<\/bdi>/, l);
  }
  forget();
});


test("scope generation rejects responses from replaced and forgotten sessions", () => {
  forget();
  setScope("w", "u", "s");
  const started = currentScope();
  setScope("w", "u", "s");
  assert.equal(currentScope(), started);
  setScope("w", "u", "next");
  assert.notEqual(currentScope(), started);
  setScope("w", "u", "s");
  assert.notEqual(currentScope(), started);
  const restored = currentScope();
  forget();
  setScope("w", "u", "s");
  assert.notEqual(currentScope(), restored);
  forget();
});


test("a new attempt clears the prior retained refusal receipt", () => {
  forget();
  setScope("w", "u", "s");
  settle("installation.approve", "t", "hint", { code: "invalid_stage", request: "old-server-id" });
  pendingFor("installation.approve", "t", { rationale: "retry" }, "r", () => ({ request: "next-client-id", key: "next-key" }));
  assert.equal(refusedDetail("installation.approve", "t"), undefined);
  assert.ok(!refusedNow("installation.approve", "t"));
  forget();
});

type Receipt = { code: string; request: string };
function reference(command: string, target: string, detail?: Receipt): RowDiagnostic | undefined {
  return detail && { command, detail: { ...detail, target, client: "", cancelled: false } };
}
function refusedDetail(command: string, target: string): Receipt | undefined {
  const entry = refusalDiagnostics(target).find((v) => v.command === command);
  return entry && { code: entry.detail.code, request: entry.detail.request };
}
function settle(command: string, target: string, reaction: Parameters<typeof settleRecorded>[2], detail?: Receipt) {
  settleRecorded(command, target, reaction, reference(command, target, detail));
}
