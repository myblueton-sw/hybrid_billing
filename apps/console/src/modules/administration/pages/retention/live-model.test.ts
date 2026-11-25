import { currentScope } from "./live-model";
import type { AdminClient } from "../../../../service/admin";
import { refusalDiagnostics } from "./live-model";
import type { RowDiagnostic } from "../../../../service/command-diagnostics";
import { test } from "node:test";
import assert from "node:assert/strict";
import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { retentionOf, proposalOf, type RetentionPage } from "../../../../service/retention";
import { sample } from "../../../../service/retention.sample";
import {
  approveNarrow, approveValuesFor, artifactIdsInUse, artifactOf, artifactTarget, missingArtifacts, withHeldArtifacts, withMoreReferences, blockerPolicy, canonical, changeOf, classTarget, clearReviewed, currentContext, currentFrom, currentOf, decisionLine, deniedOutcome, discard, draftOf,
  duplicateIn, fieldReaction, holdValues, keepsAttempt, forget, frozenTargetOk, goneKind, graceProblem, holdMatch, holdTarget, impactValues, keepDraft, keepReviewed, keptKey, listed, partialCounts, pendingFor,
  pendingNow, pendingPolicyFor, pendingReleaseFor, periodProblem, proposalTarget, pruneRefusals, react, reasonProblem, refusedNow, reviewedChanged, reviewedOf, runNumber, runTarget,
  sameAsCurrent, selectionOf, setScope, settle as settleRecorded, snapshotOf, stateSentence, stillAllowed, storeSum, tenantTarget, textProblem, timeKnown, utf8Bytes, vanished, viewOf,
} from "./live-model";
import { k, plain } from "./live-copy";
import { allWordTables, category, classes, counted, plurals } from "./live-codes";
import { Free, Who, holdTextError, href, stampParts } from "./live-format";

const locales = ["en", "ko", "ar"] as const;
const digest = "e".repeat(64);
type Over = Record<string, unknown>;

function page(mut: (s: typeof sample) => void = () => {}): RetentionPage {
  const s = JSON.parse(JSON.stringify(sample)) as typeof sample;
  mut(s);
  return retentionOf(s as unknown as Record<string, unknown>);
}

const prop = (over: Over) => proposalOf({ number: 7, target: "retention-proposal-7", kind: "policy", state: "pending", impact_required: false, impact_now: null, can_approve: true, approve_blocked: "", ...over });

test("views fall back to policies", () => {
  assert.equal(viewOf("holds"), "holds");
  assert.equal(viewOf("bogus"), "policies");
});

test("targets must match their pattern and equal the value built from the row", () => {
  const p = page();
  assert.equal(classTarget(p.policies![0]), "retention-class:claim");
  assert.equal(classTarget({ ...p.policies![0], target: "retention-class:source" }), "");
  assert.equal(classTarget({ ...p.policies![0], class: "other", target: "retention-class:other" }), "");
  assert.equal(proposalTarget(prop({})), "retention-proposal-7");
  assert.equal(proposalTarget(prop({ target: "retention-proposal-07" })), "");
  assert.equal(proposalTarget(prop({ number: 8 })), "");
  const a = p.expiry![0].artifacts![0];
  assert.equal(artifactTarget(a), "retention-artifact:sample-source-1");
  const id109 = "a".repeat(109);
  const id110 = "a".repeat(110);
  assert.equal(artifactTarget({ ...a, id: id109, hold_target: "retention-artifact:" + id109 }), "retention-artifact:" + id109);
  assert.equal(artifactTarget({ ...a, id: id110, hold_target: "retention-artifact:" + id110 }), "");
  assert.equal(tenantTarget(p, "sample-workspace"), "retention-tenant:sample-workspace");
  assert.equal(tenantTarget(p, "other"), "");
  assert.equal(holdTarget(p.holds.rows![0]), "retention-hold:sample-hold-temporary");
  assert.equal(holdTarget({ ...p.holds.rows![0], target: "retention-hold:x" }), "");
  const run = p.deletion.rows![0];
  assert.equal(runTarget(run), "retention-run-2");
  assert.equal(runNumber({ target: "retention-run-2", label: "run/3" }), "");
  assert.equal(runNumber({ target: "retention-run-02", label: "run/02" }), "");
  assert.equal(runNumber(null), "");
  assert.ok(frozenTargetOk("retention.stop_run", "retention-run-2"));
  assert.ok(!frozenTargetOk("retention.stop_run", "retention-run-x"));
  assert.ok(frozenTargetOk("retention.place_hold", "retention-tenant:w"));
  assert.ok(!frozenTargetOk("retention.propose_policy", "retention-class:nope"));
});

test("availability fails closed for all six pairs", () => {
  const p = page((s) => {
    s.policies[0].can_propose = true;
    (s.policies[0] as Over).propose_blocked = "x";
    (s.expiry[0].artifacts[0] as Over).holdable = "true";
    (s.holds.rows[0] as Over).can_propose_release = true;
    (s.holds.rows[0] as Over).release_blocked = 5;
    (s.deletion.rows[0] as Over).can_stop = null;
    (s.proposals.rows[0] as Over).approve_blocked = undefined;
    (s.proposals.rows[1] as Over).can_withdraw = "true";
  });
  assert.equal(p.policies![0].can_propose, false);
  assert.equal(p.expiry![0].artifacts![0].holdable, false);
  assert.equal(p.holds.rows![0].can_propose_release, false);
  assert.equal(p.holds.rows![0].release_blocked, "missing");
  assert.equal(p.deletion.rows![0].can_stop, false);
  assert.equal(p.proposals.rows![0].can_approve, false);
  assert.equal(p.proposals.rows![1].can_withdraw, false);
});

test("blocker display policy and the hidden-only state sentences", () => {
  assert.equal(blockerPolicy("not_pending", "approve"), "hidden");
  assert.equal(blockerPolicy("not_pending", "approve", { pendingRow: true }), "explain");
  assert.equal(blockerPolicy("not_proposer", "withdraw", { pendingRow: true }), "hidden");
  assert.equal(blockerPolicy("released", "propose_release"), "hidden");
  assert.equal(blockerPolicy("released", "approve"), "explain");
  assert.equal(blockerPolicy("stopped", "stop_run"), "hidden");
  assert.equal(blockerPolicy("finished", "stop_run"), "hidden");
  assert.equal(blockerPolicy("no_eligible_approver", "propose_release"), "explain");
  for (const [code, action] of [["not_proposer", "withdraw"], ["released", "propose_release"], ["stopped", "stop_run"], ["finished", "stop_run"]] as const) {
    const s = stateSentence(code, action);
    assert.equal(s, code);
    for (const l of locales) assert.ok(k.stateSentence(l, s).length > 5);
  }
  for (const code of ["own_proposal", "not_pending", "retired", "not_console_kind", "no_eligible_approver", "released", "impact_changed", "target_too_long", "not_found", "limit", "conflict", "invalid"]) {
    for (const l of locales) assert.ok(!k.blocker(l, code).includes("("), code + " " + l);
  }
  assert.equal(k.blocker("en", "missing"), "This is not available now.");
  assert.ok(plain(k.blocker("en", "weird")).includes("(weird)"));
  assert.ok(k.placer("en", "approve").includes("cannot approve"));
  assert.ok(k.placer("en", "propose_release").includes("cannot propose"));
});

test("already proposed names the number, own and other, and falls back", () => {
  const p = page();
  const pol = pendingPolicyFor(p, p.policies![1]);
  assert.deepEqual(pol, { number: "7", own: false });
  assert.equal(plain(k.alreadyPolicy("en", "7", false)), "Policy proposal #7 for this class is already waiting for approval.");
  assert.ok(plain(k.alreadyPolicy("en", "8", true)).startsWith("Your policy proposal #8"));
  assert.equal(k.alreadyPolicy("en", "", false), "A proposal is already waiting for approval.");
  assert.deepEqual(pendingReleaseFor(p, p.holds.rows![0]), { number: "6", own: true });
  assert.equal(plain(k.alreadyRelease("en", "8", false)), "Release proposal #8 is waiting for approval.");
  assert.equal(plain(k.alreadyRelease("en", "8", true)), "Your release proposal #8 for this hold is waiting for another administrator's approval.");
});

test("propose-policy validation and canonical integers", () => {
  assert.equal(periodProblem("0"), "range");
  assert.equal(periodProblem("1"), "");
  assert.equal(periodProblem("36500"), "");
  assert.equal(periodProblem("36501"), "range");
  assert.equal(periodProblem("1.5"), "range");
  assert.equal(periodProblem("-1"), "range");
  assert.equal(periodProblem(""), "missing");
  assert.equal(graceProblem("365"), "");
  assert.equal(graceProblem("366"), "range");
  assert.equal(canonical("007"), "7");
  const known = currentOf(page().policies![1]);
  assert.ok(sameAsCurrent(known, "365", "030"));
  assert.ok(!sameAsCurrent(known, "365", "31"));
  assert.ok(!sameAsCurrent(currentOf(page().policies![0]), "1", "0"));
  assert.ok(!sameAsCurrent({ kind: "unknown" }, "1", "0"));
  assert.deepEqual(changeOf(known, 180, 30), { kind: "shorter", days: 185 });
  assert.deepEqual(changeOf(known, 400, 30), { kind: "longer", days: 35 });
  assert.deepEqual(changeOf({ kind: "none" }, 1, 0), { kind: "new" });
  assert.deepEqual(changeOf({ kind: "unknown" }, 1, 0), { kind: "unknown" });
  assert.deepEqual(currentFrom(currentContext(known)), known);
  assert.deepEqual(currentFrom({ current: "known", revision: "x" }), { kind: "unknown" });
  const odd = page((s) => { (s.policies[1] as Over).policy = { revision: 1, period_days: "365", grace_days: 30 }; });
  assert.equal(currentOf(odd.policies![1]).kind, "unknown");
  assert.equal(k.proposeImpact("en", null, "Source records"), "Another administrator must approve it. Approval adds a new revision; it deletes nothing, because deletion plans are not connected.");
  assert.ok(k.proposeImpact("en", 2, "Source records").includes("revision 2 for Source records"));
});

test("approve narrowing and the impact sent verbatim", () => {
  const ok = prop({ impact_required: true, impact_now: { as_of: "2026-11-17T09:00:00.456Z", count: 2, digest, workspace_count: 1 } });
  assert.equal(approveNarrow(ok), "");
  assert.deepEqual(impactValues(ok), { impact_as_of: "2026-11-17T09:00:00.456Z", impact_count: "2", impact_digest: digest });
  assert.equal(approveNarrow(prop({ impact_required: true, impact_now: { as_of: "x", count: 2, digest, workspace_count: 1 } })), "impact_unreadable");
  assert.equal(approveNarrow(prop({ impact_required: true, impact_now: { as_of: "2026-11-17T09:00:00Z", count: 2, digest: "ABC", workspace_count: 1 } })), "impact_unreadable");
  assert.equal(approveNarrow(prop({ impact_required: true, impact_now: null })), "impact_unreadable");
  assert.equal(approveNarrow(prop({ impact_required: false, impact_now: { as_of: "x" } })), "impact_unreadable");
  assert.equal(approveNarrow(prop({ impact_required: "yes" })), "impact_unreadable");
  assert.equal(approveNarrow(prop({ impact_required: false })), "");
  assert.deepEqual(impactValues(prop({})), {});
  assert.equal(approveNarrow(prop({ kind: "plan" })), "no_command");
  assert.equal(approveNarrow(prop({ kind: "release" })), "");
});

test("reviewed impact is kept in the draft store and cleared", () => {
  forget();
  setScope("w", "u", "s");
  assert.equal(reviewedOf("t"), null);
  keepReviewed("t", "2", digest);
  assert.ok(!reviewedChanged(reviewedOf("t"), "2", digest));
  assert.ok(reviewedChanged(reviewedOf("t"), "3", digest));
  assert.ok(reviewedChanged(reviewedOf("t"), "2", "f".repeat(64)));
  assert.equal(plain(k.impactChanged("en", "2 → 3")), "The impact changed since you reviewed it: ⁨2 → 3⁩ items.");
  clearReviewed("t");
  assert.equal(reviewedOf("t"), null);
  keepReviewed("t", "2", digest);
  forget();
  setScope("w", "u", "s");
  assert.equal(reviewedOf("t"), null);
});

test("hold and release text rule", () => {
  assert.equal(textProblem(""), "missing");
  assert.equal(textProblem("   "), "missing");
  assert.equal(textProblem("a".repeat(200)), "");
  assert.equal(textProblem("a".repeat(201)), "long");
  assert.equal(textProblem("가".repeat(66) + "aa"), "");
  assert.equal(textProblem("가".repeat(67)), "long");
  assert.equal(textProblem("ع".repeat(100)), "");
  assert.equal(textProblem("ع".repeat(101)), "long");
  assert.equal(textProblem("a\nb"), "control");
  assert.equal(textProblem("a\u0085b"), "control");
  assert.equal(holdTextError("en", ""), "Enter a hold reason.");
  assert.equal(holdTextError("en", "a".repeat(201)), "The hold reason is too long (at most 200 bytes).");
  assert.equal(holdTextError("en", "a\nb"), "Remove line breaks and control characters.");
  assert.equal(holdTextError("en", "", true), "Enter a release reason.");
  assert.equal(reasonProblem("가".repeat(86)), "long");
  assert.equal(reasonProblem("  "), "missing");
  assert.equal(utf8Bytes("가"), 3);
});

test("place-hold values never send class with an item target, and owner options", () => {
  assert.deepEqual(holdValues({ owner: "u", reason: " r ", class: "final" }, false), { owner: "u", reason: "r" });
  assert.deepEqual(holdValues({ owner: "u", reason: "r", class: "final" }, true), { owner: "u", reason: "r", class: "final" });
  assert.deepEqual(holdValues({ owner: "u", reason: "r", class: "" }, true), { owner: "u", reason: "r" });
  assert.deepEqual(holdValues({ owner: "u", reason: "r", class: "bogus" }, true), { owner: "u", reason: "r" });
  assert.equal(k.form.ownerChange("en"), "Change");
  assert.equal(k.form.ownerMissing("en"), "Choose an owner.");
  assert.equal(k.form.ownersFailed("en"), "The list of possible owners could not be read.");
  assert.equal(k.form.ownersEmpty("en"), "No active member can own a hold.");
});

test("place-hold confirm sentences", () => {
  assert.equal(k.itemDeleted("en"), "This item is already recorded as deleted. The hold cannot preserve it.");
  assert.ok(k.itemDamaged("en").includes("partly deleted"));
  assert.ok(plain(k.itemInRun("en", "run/2")).includes("deletion run run/2"));
  assert.ok(k.workspaceCovers("en", "Final documents and manifests").includes("of class Final documents and manifests"));
  assert.ok(!k.workspaceCovers("en", "").includes("of class"));
  assert.ok(k.holdStrong("en").includes("you alone place this hold"));
});

test("place-hold result sentences use plural forms and never describe pending as lost", () => {
  assert.equal(plain(k.holdPartial("en", "retention-hold-2", 1, 0, 0)), "Hold retention-hold-2 is in force, but it cannot preserve 1 item already deleted.");
  assert.equal(plain(k.holdPartial("en", "h", 2, 0, 1)), "Hold h is in force, but it cannot preserve 2 items already deleted or 1 item partly deleted.");
  assert.equal(plain(k.holdPartial("en", "h", 0, 2, 0)), "Hold h is in force. 2 items have a deletion outcome pending.");
  assert.equal(plain(k.holdPartial("en", "h", null, 0, 0)), "Hold h is in force. Items already deleted or partly deleted: Unknown.");
  assert.equal(counted("en", 1, plurals.impactNow, { w: "1", at: "t" }), "Impact now: 1 item in the installation becomes due earlier (1 in this workspace; computed t).");
  assert.deepEqual(partialCounts({ not_preserved: "1", outcome_pending: "x", partially_deleted: "0" }), { a: 1, b: null, c: 0 });
});

test("stop confirm and result", () => {
  assert.equal(plain(k.stopDone("en", "run/2", 2, "Finished, not complete")), "run/2 stopped: 2 unclaimed items were released, not deleted. Nothing was restored. Finished, not complete");
  assert.ok(plain(k.stopDone("en", "run/2", 1, "")).includes("1 unclaimed item was released"));
  assert.ok(plain(k.stopDone("en", "run/2", null, "")).includes("unknown"));
  assert.ok(k.stopBody("en").includes("stopping is not a restore"));
});

test("per-store sum check", () => {
  assert.equal(storeSum({ store: "s", target: 4, deleted: 2, failed: 0, excluded: 2, pending: 0 }), "ok");
  assert.equal(storeSum({ store: "s", target: 4, deleted: 2, failed: 0, excluded: 1, pending: 0 }), "mismatch");
  assert.equal(storeSum({ store: "s", target: 4, deleted: null, failed: 0, excluded: 2, pending: 0 }), "mismatch");
});

test("decision line from the field value", () => {
  assert.deepEqual(decisionLine("not_connected", true), { kind: "not_connected" });
  assert.deepEqual(decisionLine("durable", false), { kind: "code", code: "durable" });
  assert.deepEqual(decisionLine("", true), { kind: "unknown" });
  assert.deepEqual(decisionLine("", false), { kind: "none" });
  assert.equal(k.decision("en", "not_connected"), "Decision record: not connected (not confirmed)");
});

test("run state words, chunks and stopper", () => {
  assert.equal(k.runState("en", "running"), "Running (no worker connected)");
  assert.equal(k.runState("en", "stopped"), "Stopped (a claimed chunk is still open)");
  assert.equal(k.runState("en", "done"), "Finished, not complete");
  assert.equal(k.runState("en", "complete"), "Closed without failures");
  assert.equal(k.runState("en", "odd"), "Unknown (odd)");
  assert.equal(k.runStateNote("en", "running"), "");
  assert.equal(counted("en", 3, plurals.chunks, { next: "3" }), "3 of 3 chunks finished (a stop finishes unclaimed chunks)");
  assert.equal(plain(k.stoppedBy("en", "admin@example.invalid")), "Stopped by admin@example.invalid: unclaimed items were released, not deleted.");
});

test("plural forms through Intl.PluralRules", () => {
  assert.deepEqual([0, 1, 2, 3, 11, 100].map((n) => category("ar", n)), ["zero", "one", "two", "few", "many", "other"]);
  assert.deepEqual([0, 1, 2].map((n) => counted("en", n, plurals.items)), ["0 items", "1 item", "2 items"]);
  assert.deepEqual([0, 1, 11].map((n) => counted("ko", n, plurals.items)), ["0건", "1건", "11건"]);
  const ar = [0, 1, 2, 3, 11, 100].map((n) => counted("ar", n, plurals.items));
  assert.deepEqual(ar, ["0 عنصر", "عنصر واحد", "عنصران", "3 عناصر", "11 عنصرًا", "100 عنصر"]);
  for (const forms of Object.values(plurals)) {
    assert.equal(forms.ar.length, 6);
    for (const n of [0, 1, 2, 3, 11, 100]) for (const l of locales) assert.ok(counted(l, n, forms, { w: "1", at: "x", t: "1", next: "1", run: "r" }).length > 0);
  }
  assert.equal(counted("en", 50, plurals.showingOf), "Showing 50 of 50.");
});

test("every vocabulary key has EN, KO and AR, and copy claims no deletion", () => {
  for (const [name, table] of Object.entries(allWordTables)) {
    for (const [key, row] of Object.entries(table)) {
      assert.equal(row.length, 3, name + "." + key);
      for (const s of row) assert.ok(s.trim().length > 0, name + "." + key);
    }
  }
  for (const c of classes) for (const l of locales) assert.ok(!k.className(l, c).startsWith("Unknown"));
  for (const l of locales) assert.ok(k.statement(l).length > 20);
  assert.ok(!/\bwas deleted\b/.test(k.notShortening("en")));
  assert.ok(k.policyApplied("en", "7").includes("Nothing was deleted"));
});

test("unknown versus zero versus not connected versus other scope", () => {
  assert.equal(k.connection("en", "not_connected"), "Not connected");
  assert.equal(k.connection("en", "x"), "Unknown (x)");
  assert.equal(k.otherScopeCounts("en"), "Not shown: this hold also covers data outside this workspace.");
  assert.equal(k.runOutside("en"), "Owned by a run that is not shown in this workspace");
  assert.ok(!timeKnown("0001-01-01T00:00:00Z"));
  assert.ok(!timeKnown(""));
  assert.equal(stampParts("en", "0001-01-01T00:00:00Z"), null);
  const p = page((s) => { (s.expiry[0] as Over).counts = { eligible: 0 }; });
  assert.equal(p.expiry![0].counts.eligible, 0);
  assert.equal(p.expiry![0].counts.held, null);
});

test("reactions: field mapping and fieldless invalid", () => {
  assert.equal(react("invalid", []), "reason");
  assert.equal(react("invalid", ["owner"]), "field");
  assert.equal(fieldReaction("retention.place_hold", ["owner"]), "owner");
  assert.equal(fieldReaction("retention.place_hold", ["reason"]), "text");
  assert.equal(fieldReaction("retention.propose_release", ["reason"]), "text");
  assert.equal(fieldReaction("retention.propose_policy", ["period"]), "period");
  assert.equal(fieldReaction("retention.propose_policy", ["grace"]), "grace");
  assert.equal(fieldReaction("retention.approve", ["impact_as_of"]), "impact");
  assert.equal(fieldReaction("retention.approve", ["rationale"]), "reason");
  assert.equal(fieldReaction("retention.place_hold", ["class"]), "built");
  assert.equal(fieldReaction("retention.stop_run", ["target_id"]), "built");
  assert.equal(fieldReaction("retention.withdraw", ["expected_revision"]), "built");
});

test("kept sentences per command, their clearing and pruning", () => {
  forget();
  setScope("w", "u", "s");
  const p = page();
  settle("retention.stop_run", "retention-run-2", "hint", "stage");
  assert.equal(refusedNow("retention.stop_run", "retention-run-2"), "stage");
  pruneRefusals(p);
  assert.equal(refusedNow("retention.stop_run", "retention-run-2"), "stage");
  pruneRefusals(page((s) => { (s.deletion.rows[0] as Over).can_stop = false; (s.deletion.rows[0] as Over).stop_blocked = "finished"; }));
  assert.equal(refusedNow("retention.stop_run", "retention-run-2"), "");
  settle("retention.place_hold", "retention-artifact:sample-source-1", "hint", "refused");
  assert.equal(keptKey("retention.place_hold", refusedNow("retention.place_hold", "retention-artifact:sample-source-1")), "retention.place_hold:refused");
  assert.ok(k.kept("en", "retention.place_hold:refused").includes("100 holds"));
  assert.ok(k.kept("en", "retention.place_hold:stage").includes("Reload and try again"));
  settle("retention.place_hold", "retention-artifact:sample-source-1", "ok");
  assert.equal(refusedNow("retention.place_hold", "retention-artifact:sample-source-1"), "");
  for (const c of ["retention.approve", "retention.withdraw", "retention.propose_policy", "retention.propose_release", "retention.stop_run"]) for (const l of locales) assert.ok(!k.kept(l, c).startsWith("Unknown"));
  forget();
});

test("denied outcomes and the vanished-attempt notice", () => {
  forget();
  setScope("w", "u", "s");
  const p = page();
  assert.equal(deniedOutcome("retention.approve", "retention-proposal-99", true, p), "applied_proposal");
  assert.equal(deniedOutcome("retention.approve", "retention-proposal-7", true, p), "gone");
  assert.equal(deniedOutcome("retention.approve", "retention-proposal-99", false, p), "gone");
  assert.equal(goneKind("retention.place_hold", "retention-tenant:w"), "workspace");
  assert.equal(goneKind("retention.place_hold", "retention-artifact:x"), "item");
  assert.equal(goneKind("retention.stop_run", "retention-run-2"), "run");
  for (const kind of ["proposal", "hold", "run", "item", "class", "workspace"]) assert.ok(k.gone("en", kind).includes("no longer available"));
  pendingFor("retention.stop_run", "retention-run-9", { rationale: "x" }, "r", () => ({ request: "q", key: "k" }), [], { view: "deletion" });
  assert.equal(vanished(p, "deletion").length, 1);
  assert.equal(vanished(p, "holds").length, 0);
  assert.ok(listed(p, "retention.stop_run", "retention-run-2"));
  assert.ok(stillAllowed(p, "retention.stop_run", "retention-run-2"));
  forget();
});

test("place-hold stale: no automatic renewal and the matching hold", () => {
  forget();
  setScope("w", "m", "s");
  const t = "retention-tenant:sample-workspace";
  const first = pendingFor("retention.place_hold", t, { owner: "m", reason: "Sample export review hold", class: "temporary", rationale: "x" }, "r", () => ({ request: "q1", key: "k1" }));
  settle("retention.place_hold", t, "renew");
  const kept = pendingNow("retention.place_hold", t);
  assert.ok(kept && kept.stale && kept.key === first.key);
  const again = pendingFor("retention.place_hold", t, { owner: "m", reason: "Sample export review hold", class: "temporary", rationale: "x" }, "r", () => ({ request: "q2", key: "k2" }));
  assert.equal(again.key, "k2");
  const p = page();
  assert.equal(holdMatch(p, "m", t, { owner: "m", reason: "Sample export review hold", class: "temporary" }), "sample-hold-temporary");
  assert.equal(holdMatch(p, "m", t, { owner: "m", reason: "Sample export review hold" }), "");
  assert.equal(holdMatch(p, "other", t, { owner: "m", reason: "Sample export review hold", class: "temporary" }), "");
  assert.ok(plain(k.staleMatch("en", "retention-hold-3")).includes("placed by you with the same target, owner and reason"));
  pendingFor("retention.stop_run", "retention-run-2", { rationale: "x" }, "r", () => ({ request: "q3", key: "k3" }));
  settle("retention.stop_run", "retention-run-2", "renew");
  assert.equal(pendingNow("retention.stop_run", "retention-run-2"), undefined);
  forget();
});

test("place hold on an item: the matching hold is found by its artifact scope", () => {
  const p = page((s) => {
    (s.holds.rows as Over[]).push({
      id: "retention-hold-2", target: "retention-hold:retention-hold-2", owner: { id: "a2", login: null }, reason: "qa", placer: { id: "a1", login: "a1@example.invalid" },
      released: false, scope: { artifacts: ["sample-source-7"], workspace: false, classes: [] }, counts: {}, counts_scope: "workspace", outcomes: [], outcomes_more: false,
    });
  });
  assert.equal(holdMatch(p, "a1", "retention-artifact:sample-source-7", { owner: "a2", reason: "qa" }), "retention-hold-2");
  assert.equal(holdMatch(p, "a1", "retention-artifact:sample-source-6", { owner: "a2", reason: "qa" }), "");
  assert.equal(holdMatch(p, "a1", "retention-artifact:sample-source-7", { owner: "a2", reason: "other" }), "");
});

test("limited keeps the place-hold attempt and partial drops it", () => {
  forget();
  setScope("w", "u", "s");
  const t = "retention-artifact:x";
  pendingFor("retention.place_hold", t, { owner: "u", reason: "r", rationale: "x" }, "r", () => ({ request: "q", key: "k" }));
  settle("retention.place_hold", t, react("limited", []));
  assert.equal(pendingNow("retention.place_hold", t)?.key, "k");
  settle("retention.place_hold", t, react("partial", []));
  assert.equal(pendingNow("retention.place_hold", t), undefined);
  forget();
});

test("an approve resend sends the frozen impact, not the reloaded one", () => {
  forget();
  setScope("w", "u", "s");
  const first = prop({ impact_required: true, impact_now: { as_of: "2026-11-17T09:00:00.1Z", count: 2, digest, workspace_count: 1 } });
  const values = approveValuesFor(first, undefined);
  const kept = pendingFor("retention.approve", "retention-proposal-7", { ...values, rationale: "x" }, "r", () => ({ request: "q", key: "k" }));
  const reloaded = prop({ impact_required: true, impact_now: { as_of: "2026-11-17T09:05:00Z", count: 3, digest: "f".repeat(64), workspace_count: 1 } });
  assert.deepEqual(approveValuesFor(reloaded, kept), { impact_as_of: "2026-11-17T09:00:00.1Z", impact_count: "2", impact_digest: digest });
  assert.deepEqual(approveValuesFor(reloaded, undefined), { impact_as_of: "2026-11-17T09:05:00Z", impact_count: "3", impact_digest: "f".repeat(64) });
  forget();
});

test("attempt store: identical values only, scoped by session, cleared by forget, discarded on partial", () => {
  forget();
  setScope("w", "u", "s1");
  pendingFor("retention.withdraw", "t", { rationale: "x" }, "r", () => ({ request: "q", key: "k" }));
  discard("retention.withdraw", "t", { rationale: "x" });
  assert.ok(pendingNow("retention.withdraw", "t"));
  discard("retention.withdraw", "t", { rationale: "y" });
  assert.equal(pendingNow("retention.withdraw", "t"), undefined);
  pendingFor("retention.withdraw", "t", { rationale: "x" }, "r", () => ({ request: "q", key: "k" }));
  keepDraft("retention.withdraw", "t", { rationale: "x" });
  setScope("w", "u", "s2");
  assert.equal(pendingNow("retention.withdraw", "t"), undefined);
  assert.deepEqual(draftOf("retention.withdraw", "t"), {});
  pendingFor("retention.withdraw", "t", { rationale: "x" }, "r", () => ({ request: "q", key: "k" }));
  settle("retention.withdraw", "t", "partial");
  assert.equal(pendingNow("retention.withdraw", "t"), undefined);
  forget();
});

test("duplicate announcement only when the returned proposal was already listed", () => {
  const p = page();
  const snap = snapshotOf(p, "retention.propose_policy");
  assert.deepEqual(snap, ["7", "6"]);
  assert.ok(duplicateIn(snap, "7"));
  assert.ok(!duplicateIn(snap, "8"));
  assert.deepEqual(snapshotOf(p, "retention.stop_run"), []);
  assert.equal(plain(k.duplicate("en", "7")), "This was already proposed as #7.");
});

test("URL selection is validated for every parameter and tab; hrefs start with #/retention?", () => {
  const p = page();
  const sel = (q: string, view: string) => selectionOf(new URLSearchParams(q), p, view);
  assert.equal(sel("class=source", "policies").cls, "source");
  assert.equal(sel("class=nope", "policies").cls, "");
  assert.equal(sel("class=source&artifact=sample-source-1", "expiry").artifact, "sample-source-1");
  assert.equal(sel("artifact=missing", "expiry").artifact, "");
  assert.equal(sel("artifact=sample-source-1", "holds").artifact, "");
  assert.equal(sel("hold=sample-hold-temporary", "holds").hold, "sample-hold-temporary");
  assert.equal(sel("hold=x y", "holds").hold, "");
  assert.equal(sel("run=2", "deletion").run, "2");
  assert.equal(sel("run=02", "deletion").run, "");
  assert.equal(sel("proposal=7", "policies").proposal, "7");
  assert.equal(sel("proposal=7", "holds").proposal, "");
  assert.equal(sel("proposal=6", "holds").proposal, "6");
  assert.equal(selectionOf(new URLSearchParams("run=2"), null, "deletion").run, "");
  for (const l of locales) for (const v of ["policies", "expiry", "holds", "deletion", "restore", "decisions"]) assert.ok(href(l, v, { run: "2" }).startsWith("#/retention?"));
});

test("Free renders span[dir=auto], never bdi; Who uses p04-login", () => {
  const free = renderToStaticMarkup(createElement(Free, { l: "ar", value: "سبب" }));
  assert.ok(free.startsWith('<span dir="auto" class="p04-free">'));
  assert.ok(!free.includes("<bdi"));
  assert.equal(renderToStaticMarkup(createElement(Free, { l: "en", value: "" })), "Unknown");
  const who = renderToStaticMarkup(createElement(Who, { l: "en", p: { id: "u1", login: "" } }));
  assert.ok(who.includes('class="live-mono p04-login"') && who.includes("u1"));
  assert.equal(renderToStaticMarkup(createElement(Who, { l: "en", p: null, empty: "Not stopped" })), "Not stopped");
});

test("plan proposals offer no command even with can_withdraw true", () => {
  const plan = prop({ kind: "plan", can_withdraw: true, withdraw_blocked: "" });
  assert.equal(approveNarrow(plan), "no_command");
  const p = page((s) => { (s.proposals.rows as Over[]).push({ number: 5, target: "retention-proposal-5", kind: "plan", state: "pending", can_withdraw: true, withdraw_blocked: "", summary: { plan: { class: "temporary" } } }); });
  assert.ok(!stillAllowed(p, "retention.withdraw", "retention-proposal-5"));
});

test("a step-up refusal or cancellation keeps the attempt and the form", () => {
  for (const f of ["reauth", "reauth_cancelled"]) {
    assert.equal(react(f, []), "reauth");
    assert.equal(keepsAttempt(react(f, [])), true);
  }
});

test("storage results state that ERP and downloaded copies are outside the controlled scope", async () => {
  const { readFileSync } = await import("node:fs");
  const source = readFileSync(new URL("./LiveDeletion.tsx", import.meta.url), "utf8");
  assert.match(source, /\{k\.sumNote\(l\)\} \{k\.backupNote\(l\)\}<\/p>\s*<p className="p04-small">\{k\.outsideNote\(l\)\}<\/p>/);
  const texts = (["en", "ko", "ar"] as const).map((l) => k.outsideNote(l));
  assert.match(texts[0], /ERP/);
  assert.match(texts[0], /downloaded/);
  assert.match(texts[0], /outside the controlled scope/);
  assert.match(texts[1], /통제 범위 밖/);
  assert.match(texts[2], /خارج النطاق المتحكَّم فيه/);
  assert.match(texts[1], /ERP/);
  assert.match(texts[2], /ERP/);
  assert.equal(new Set(texts).size, 3);
});

test("storage results state that AI source bodies are not collected and Assistant conversations are a separate data class", async () => {
  const { readFileSync } = await import("node:fs");
  const source = readFileSync(new URL("./LiveDeletion.tsx", import.meta.url), "utf8");
  assert.match(source, /<p className="p04-small">\{k\.outsideNote\(l\)\}<\/p>\s*<p className="p04-small">\{k\.aiNote\(l\)\}<\/p>/);
  const texts = (["en", "ko", "ar"] as const).map((l) => k.aiNote(l));
  assert.match(texts[0], /no field for prompt or response bodies/);
  assert.match(texts[0], /Assistant conversations are their own data class/);
  assert.match(texts[0], /does not delete the records that refer to them/);
  assert.match(texts[0], /AI usage intake and the Assistant are not connected/);
  assert.doesNotMatch(texts[0], /only token counts/);
  assert.match(texts[1], /프롬프트·응답 본문 필드가 없습니다/);
  assert.match(texts[1], /별도 자료 등급/);
  assert.match(texts[1], /AI 사용량 수집과 Assistant가 연결되어 있지 않습니다/);
  assert.match(texts[2], /حقل لنصوص الموجّهات أو الردود/);
  assert.match(texts[2], /محادثات المساعد فئة بيانات مستقلة/);
  assert.match(texts[2], /غير متصلين/);
  assert.equal(new Set(texts).size, 3);
});


test("retained refusal metadata follows command, target, session and pruning", () => {
  forget();
  setScope("w", "u", "s");
  const approve = { code: "invalid_stage", request: "server-approve" };
  const withdraw = { code: "prerequisite_failed", request: "server-withdraw" };
  settle("retention.approve", "missing-target", "hint", "stage", approve);
  settle("retention.withdraw", "missing-target", "hint", "stage", withdraw);
  assert.deepEqual(refusedDetail("retention.approve", "missing-target"), approve);
  assert.deepEqual(refusedDetail("retention.withdraw", "missing-target"), withdraw);
  assert.equal(refusedDetail("retention.approve", "other-target"), undefined);
  settle("retention.approve", "missing-target", "ok");
  assert.equal(refusedDetail("retention.approve", "missing-target"), undefined);
  assert.deepEqual(refusedDetail("retention.withdraw", "missing-target"), withdraw);
  setScope("w", "u", "next-session");
  assert.equal(refusedDetail("retention.withdraw", "missing-target"), undefined);
  settle("retention.approve", "missing-target", "hint", "stage", approve);
  pruneRefusals(page());
  assert.equal(refusedDetail("retention.approve", "missing-target"), undefined);
  settle("retention.approve", "missing-target", "hint", "stage", approve);
  settle("retention.approve", "missing-target", "hint", "stage");
  assert.equal(refusedDetail("retention.approve", "missing-target"), undefined);
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
  settle("retention.approve", "retention-proposal-7", "hint", "stage", { code: "invalid_stage", request: "server-approve" });
  settle("retention.withdraw", "retention-proposal-7", "hint", "stage", { code: "prerequisite_failed", request: "server-withdraw" });
  const p = page();
  p.proposals.rows = [prop({})];
  for (const l of locales) {
    const h = renderToStaticMarkup(createElement(Proposals, {
      l, page: p, api: {} as AdminClient, me: "u", canCommand: true, selection: selectionOf(new URLSearchParams(), p, "proposals"), onReload: async () => null, workspace: "sample-workspace", kind: "policy",
      onDone: () => {}, onNotice: () => {}, onSignedOut: () => {},
    }));
    assert.ok(h.includes("server-approve"), l);
    assert.ok(h.includes("server-withdraw") && h.includes("retention.withdraw"), l);
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
  settle("retention.approve", "t", "hint", "stage", { code: "invalid_stage", request: "old-server-id" });
  pendingFor("retention.approve", "t", { rationale: "retry" }, "r", () => ({ request: "next-client-id", key: "next-key" }));
  assert.equal(refusedDetail("retention.approve", "t"), undefined);
  assert.ok(!refusedNow("retention.approve", "t"));
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
function settle(command: string, target: string, reaction: Parameters<typeof settleRecorded>[2], failure = "", detail?: Receipt) {
  settleRecorded(command, target, reaction, failure, reference(command, target, detail));
}

test("artifacts and references held beyond the main read are found, selected and kept once", async () => {
  const { artifactsPageOf, artifactsPath, referencesPageOf } = await import("../../../../service/retention");
  const { ExpiryView } = await import("./LiveExpiry");
  forget();
  setScope("w", "u", "s");
  const main = page((s) => { const g = s.expiry.find((x) => x.class === "source")!; (g as Over).total = 70; (g as Over).more = true; (g as Over).next = "sample-source-5"; });
  const group = main.expiry!.find((g) => g.class === "source")!;
  const from = group.artifacts![0];
  const extra = (id: string) => ({ class: "source", artifact: { ...from, id, hold_target: "retention-artifact:" + id, holdable: true, hold_blocked: "" } });
  pendingFor("retention.place_hold", "retention-artifact:zz-held-1", { reason: "x" }, "r1", () => ({ request: "q", key: "k" }), [], { view: "expiry" });
  settleRecorded("retention.place_hold", "retention-artifact:zz-held-2", "hint", "stage");
  settleRecorded("retention.stop_run", "retention-run-2", "hint", "stage");
  assert.deepEqual(artifactIdsInUse().sort(), ["zz-held-1", "zz-held-2"]);
  assert.deepEqual(missingArtifacts(main, [...artifactIdsInUse(), from.id, "", "a,b", "has space", "p".repeat(129)]).sort(), ["zz-held-1", "zz-held-2"]);
  assert.equal(vanished(main, "expiry").length, 1);
  assert.equal(selectionOf(new URLSearchParams("artifact=zz-held-1"), main, "expiry").artifact, "");
  const held = withHeldArtifacts(main, [extra("zz-held-2"), extra("zz-held-1"), extra("zz-held-2"), extra(from.id), { class: "claim", artifact: { ...from, id: "aa-claim" } }, { class: "source", artifact: { ...from, id: "" } }, { class: "unknown", artifact: { ...from, id: "x" } }]);
  const rows = held.expiry!.find((g) => g.class === "source")!.artifacts!;
  assert.deepEqual(rows.map((a) => a.id), [...group.artifacts!.map((a) => a.id), "zz-held-1", "zz-held-2"].sort());
  assert.equal(artifactOf(held, "aa-claim"), undefined);
  assert.equal(artifactOf(held, "zz-held-1")?.group.class, "source");
  assert.equal(held.expiry!.find((g) => g.class === "source")!.more, true);
  assert.equal(vanished(held, "expiry").length, 0);
  assert.equal(listed(held, "retention.place_hold", "retention-artifact:zz-held-1"), true);
  assert.equal(stillAllowed(held, "retention.place_hold", "retention-artifact:zz-held-1"), true);
  assert.equal(selectionOf(new URLSearchParams("artifact=zz-held-1"), held, "expiry").artifact, "zz-held-1");
  pruneRefusals(held);
  assert.equal(refusedNow("retention.place_hold", "retention-artifact:zz-held-2"), "stage");
  assert.equal(withHeldArtifacts(main, []), main);
  assert.equal(withHeldArtifacts(main, [extra(from.id)]), main);
  forget();
  const refs = page((s) => { (s.holds.references as Over).total = 3; (s.holds.references as Over).more = true; (s.holds.references as Over).next = "sample-source-3"; });
  const moreRefs = withMoreReferences(refs, [{ id: "zz-a", class: "source" }, { id: "zz-a", class: "source" }, { id: refs.holds.references.rows![0].id, class: "source" }, { id: "", class: "source" }]);
  assert.deepEqual(moreRefs.holds.references.rows!.map((r) => r.id), [...refs.holds.references.rows!.map((r) => r.id), "zz-a"]);
  assert.equal(moreRefs.holds.references.more, true);
  assert.equal(withMoreReferences(refs, []), refs);
  assert.equal(artifactsPath({ cls: "source", after: "a:b" }), "/retention/artifacts?class=source&limit=50&after=a%3Ab");
  assert.equal(artifactsPath({ ids: ["a:b", "c"] }), "/retention/artifacts?id=a%3Ab%2Cc");
  for (const bad of [{}, { artifacts: [], next: "" }, { artifacts: "x", next: "", total: 0 }, { artifacts: [], next: 4, total: 0 }]) assert.equal(artifactsPageOf(bad), null, JSON.stringify(bad));
  const got = artifactsPageOf({ artifacts: [{ class: "source", id: "zz", holdable: true, hold_target: "retention-artifact:zz" }, null], next: "zz", total: 70 });
  assert.deepEqual([got?.rows.length, got?.rows[0].class, got?.rows[0].artifact.id, got?.next, got?.total], [1, "source", "zz", "zz", 70]);
  for (const bad of [{}, { references: [], next: "" }, { references: [], next: 4, total: 0 }]) assert.equal(referencesPageOf(bad), null, JSON.stringify(bad));
  assert.deepEqual(referencesPageOf({ references: [{ id: "r", class: "source" }, 3], next: "", total: 1 }), { rows: [{ id: "r", class: "source" }], next: "", total: 1 });
  const base = { api: {} as AdminClient, me: "u", workspace: "w", canCommand: false, onDone: () => undefined, onNotice: () => undefined, onReload: async () => null, onSignedOut: () => undefined };
  for (const l of locales) {
    const list = (status: "idle" | "failed", reset: boolean) => (key: string, first: string | null) => ({ more: key === "source" && !!first, status, reset: reset && key === "source", onMore: () => undefined });
    const html = renderToStaticMarkup(createElement(ExpiryView, { ...base, l, page: main, selection: selectionOf(new URLSearchParams(""), main, "expiry"), listOf: list("idle", false) }));
    assert.ok(html.includes(k.itemsShown(l, group.artifacts!.length, 70)) && html.includes(k.itemsMore(l)), l);
    assert.equal(html.split("p04-items-more-button").length - 1, 1, l);
    const failed = renderToStaticMarkup(createElement(ExpiryView, { ...base, l, page: main, selection: selectionOf(new URLSearchParams(""), main, "expiry"), listOf: list("failed", true) }));
    assert.ok(failed.includes(k.itemsFailed(l)) && failed.includes(k.heldTryAgain(l)) && failed.includes(k.itemsReset(l)), l);
  }
});

test("each load-more button of the expiry view is described by the heading of its class", async () => {
  const { ExpiryView } = await import("./LiveExpiry");
  const main = page((s) => { const g = s.expiry.find((x) => x.class === "source")!; (g as Over).total = 70; (g as Over).more = true; (g as Over).next = "sample-source-5"; });
  const base = { api: {} as AdminClient, me: "u", workspace: "w", canCommand: false, onDone: () => undefined, onNotice: () => undefined, onReload: async () => null, onSignedOut: () => undefined };
  const html = renderToStaticMarkup(createElement(ExpiryView, { ...base, l: "en", page: main, selection: selectionOf(new URLSearchParams(""), main, "expiry"), listOf: (key: string, first: string | null) => ({ more: !!first, status: "idle" as const, reset: false, onMore: () => undefined }) }));
  assert.ok(/<h3 id="p04-class-h-source">/.test(html));
  assert.ok(/class="[^"]*p04-items-more-button"[^>]*aria-describedby="p04-class-h-source"|aria-describedby="p04-class-h-source"[^>]*class="[^"]*p04-items-more-button"/.test(html));
});
