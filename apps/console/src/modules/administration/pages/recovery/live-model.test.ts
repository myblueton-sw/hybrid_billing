import { test } from "node:test";
import assert from "node:assert/strict";
import { recoveryOf, recoveryProposalOf, resumeRowOf, type RecoveryPage } from "../../../../service/recovery";
import {
  actor, alreadyProposed, approveNarrow, basisContext, basisFrom, basisOf, blockerPolicy, callsOf, chunksOf, deniedOutcome, discard, draftOf, duplicateIn, evidenceOf, fieldReaction,
  forget, frozenTargetOk, generationNumber, highWaterOf, installTarget, keepDraft, keepsAttempt, listed, lostList, lostOf, noRestorePoint, ordered, pendingBeginsFor, pendingFor,
  pendingNow, proposalTarget, proposeNarrow, pruneRefusals, react, reasonProblem, reconciledOf, refusedNow, selectionOf, setScope, settle, snapshotOf, stillAllowed, stoppedOf,
  summaryOf, timeKnown, utf8Bytes, vanished, viewOf, watermarkOf, historyOf,
} from "./live-model";
import { k, plain } from "./live-copy";
import { stampParts } from "./live-format";
import { allWordTables, blockerWords, keptWords, narrowWords, summaryWords } from "./live-codes";

const locales = ["en", "ko", "ar"] as const;
const digest = "b".repeat(64);
const other = "c".repeat(64);

type Over = Record<string, unknown>;

function page(over: { sets?: Over[]; isolation?: Over[]; reconciliation?: Over[]; resume?: Over[]; proposals?: Over[]; top?: Over } = {}): RecoveryPage {
  return recoveryOf({
    revision: "r1",
    now: "2026-11-17T09:00:00Z",
    state: "configured",
    sets: {
      state: "configured",
      rows: over.sets ?? [
        {
          installation: "p", state: "recovering", generation: 2,
          recovery: { state: "open", id: "recovery-1", phase: "verify", as_of: "2026-11-17T08:00:00Z", point_generation: 1, backup_reference: digest, manifest_digest: digest,
            domains: [{ domain: "database", status: "verified", mismatch_chunks: [] }, { domain: "outbox", status: "mismatch", mismatch_chunks: [1] }, { domain: "keys", status: "pending", mismatch_chunks: [] }] },
          history: [], history_total: 0, begin: "not_connected", abandon: "not_connected",
        },
        { installation: "s", state: "locked", generation: 2, recovery: { state: "none" }, history: [], history_total: 0, begin: "not_connected", abandon: "not_connected" },
      ],
    },
    isolation: {
      state: "configured",
      rows: over.isolation ?? [
        { installation: "p", state: "recovering", generation: 2, stopped_at: "2026-11-17T08:00:00Z", can_lock: false, lock_blocked: "recovering", open_calls_source: "recovery", open_calls: [], open_calls_total: 0 },
        { installation: "s", state: "locked", generation: 2, stopped_at: "2026-11-17T08:00:00Z", can_lock: false, lock_blocked: "locked", open_calls_source: "not_shown", open_calls: [], open_calls_total: 0 },
      ],
    },
    reconciliation: { state: "configured", rows: over.reconciliation ?? [{ installation: "p", externals: [{ id: "e1", kind: "erp", status: "reconciled" }, { id: "e2", kind: "issuance", status: "pending" }] }] },
    migration: { state: "not_connected", modules: ["migration", "balances"] },
    resume: {
      state: "configured",
      rows: over.resume ?? [
        { installation: "p", mode: "recovery_resume", phase: "verify", pending_events: [], pending_total: 2, lost: [], lost_digest: digest, can_propose_unlock: false, unlock_blocked: "incomplete" },
        { installation: "s", mode: "lock_release", phase: "not_recovering", pending_events: [], pending_total: 0, lost: [], lost_digest: digest, can_propose_unlock: false, unlock_blocked: "already_proposed" },
      ],
    },
    proposals: over.proposals ?? [unlock(10, "pending", { proposer: "u2" })],
    ...over.top,
  });
}

function unlock(n: number, state: string, over: Over = {}, summary: Over = {}): Over {
  return {
    number: n, target: "recovery-proposal-" + n, kind: "unlock", installation: "s", state, proposer: "u2", proposer_login: "admin2@example.invalid",
    summary: { installation: "s", unlock_generation: 2, lost_digest: digest, ...summary }, can_approve: state === "pending", approve_blocked: state === "pending" ? "" : "not_pending",
    can_withdraw: false, withdraw_blocked: state === "pending" ? "not_proposer" : "not_pending", ...over,
  };
}

test("views fall back to restore sets", () => {
  assert.equal(viewOf("resume"), "resume");
  assert.equal(viewOf("migration"), "migration");
  assert.equal(viewOf("nope"), "sets");
});

test("availability fails closed for every flat pair", () => {
  const r = resumeRowOf({ can_propose_unlock: true, unlock_blocked: "incomplete" });
  assert.deepEqual([r.can_propose_unlock, r.unlock_blocked], [false, "incomplete"]);
  assert.deepEqual([resumeRowOf({ can_propose_unlock: true, unlock_blocked: 4 }).unlock_blocked], ["missing"]);
  assert.deepEqual([resumeRowOf({ can_propose_unlock: false, unlock_blocked: "" }).unlock_blocked], ["missing"]);
  assert.deepEqual([resumeRowOf({}).can_propose_unlock, resumeRowOf({}).unlock_blocked], [false, "missing"]);
  const p = recoveryProposalOf({ can_approve: true, approve_blocked: "x", can_withdraw: false, withdraw_blocked: "" });
  assert.deepEqual([p.can_approve, p.approve_blocked, p.can_withdraw, p.withdraw_blocked], [false, "x", false, "missing"]);
  const pg = page({ isolation: [{ installation: "s", state: "serving", generation: 2, can_lock: true }] });
  assert.equal(pg.isolation.rows![0].can_lock, false);
  assert.equal(pg.isolation.rows![0].lock_blocked, "missing");
});

test("blocker display policy hides only the codes whose state already explains them", () => {
  assert.equal(blockerPolicy("not_pending", "approve"), "hidden");
  assert.equal(blockerPolicy("not_pending", "approve", { pendingRow: true }), "explain");
  assert.equal(blockerPolicy("not_proposer", "withdraw", { pendingRow: true }), "hidden");
  assert.equal(blockerPolicy("locked", "lock"), "hidden");
  assert.equal(blockerPolicy("recovering", "lock"), "explain");
  assert.equal(blockerPolicy("not_recovering", "propose_unlock", { mode: "none" }), "hidden");
  assert.equal(blockerPolicy("not_recovering", "propose_unlock", { mode: "lock_release" }), "explain");
  assert.equal(blockerPolicy("not_console_kind", "approve", { pendingRow: true }), "explain");
  assert.equal(blockerPolicy("not_console_kind", "approve"), "hidden");
  for (const code of ["own_proposal", "already_proposed", "retired", "open_calls", "no_restore_point", "incomplete", "lost", "kind_coverage", "generation", "not_found", "invalid", "limit", "conflict", "missing", "zzz"]) {
    assert.equal(blockerPolicy(code, "propose_unlock", { mode: "recovery_resume", pendingRow: true }), "explain", code);
  }
});

test("already proposed names the pending unlock, own or other, with a fallback", () => {
  const theirs = page();
  assert.deepEqual(alreadyProposed(theirs, "s"), { number: "10", own: false });
  assert.match(k.already("en", "10", false), /^Unlock proposal #10 is already waiting/);
  const mine = page({ proposals: [unlock(11, "pending", { proposer: "u1", can_approve: false, approve_blocked: "own_proposal", can_withdraw: true, withdraw_blocked: "" })] });
  assert.deepEqual(alreadyProposed(mine, "s"), { number: "11", own: true });
  assert.match(k.already("en", "11", true), /^Your unlock proposal #11 is waiting/);
  const byWithdraw = page({ proposals: [unlock(12, "pending", { can_approve: false, approve_blocked: "generation", can_withdraw: true, withdraw_blocked: "" })] });
  assert.equal(alreadyProposed(byWithdraw, "s")?.own, true);
  assert.equal(alreadyProposed(page({ proposals: [] }), "s"), null);
  assert.equal(k.already("en", "", false), "An unlock proposal is already waiting for approval.");
});

test("targets match their pattern, their number and the command ID set", () => {
  const p = (n: unknown, target: string) => recoveryProposalOf({ number: n, target });
  assert.equal(proposalTarget(p(10, "recovery-proposal-10")), "recovery-proposal-10");
  assert.equal(proposalTarget(p(10, "recovery-proposal-11")), "");
  assert.equal(proposalTarget(p(10, "recovery-proposal-010")), "");
  assert.equal(proposalTarget(p(null, "recovery-proposal-10")), "");
  assert.equal(proposalTarget(p(10, "plugin-proposal-10")), "");
  assert.equal(installTarget("sample-standby"), "recovery-installation:sample-standby");
  assert.equal(installTarget(""), "");
  assert.equal(installTarget("bad id"), "");
  assert.equal(installTarget("é"), "");
  assert.equal(installTarget("a".repeat(106)), "recovery-installation:" + "a".repeat(106));
  assert.equal(installTarget("a".repeat(107)), "");
  assert.equal(frozenTargetOk("recovery.approve", "recovery-proposal-3"), true);
  assert.equal(frozenTargetOk("recovery.approve", "recovery-installation:x"), false);
  assert.equal(frozenTargetOk("recovery.lock", "recovery-installation:x"), true);
  assert.equal(frozenTargetOk("recovery.lock", "recovery-installation:"), false);
});

test("propose-unlock narrowing: digest, mode and lost list", () => {
  const base = { installation: "s", mode: "lock_release", lost: [], lost_digest: digest, can_propose_unlock: true, unlock_blocked: "" };
  assert.equal(proposeNarrow(resumeRowOf(base)), "");
  assert.equal(proposeNarrow(resumeRowOf({ ...base, lost_digest: "ABC" })), "unreadable");
  assert.equal(proposeNarrow(resumeRowOf({ ...base, lost_digest: digest.toUpperCase() })), "unreadable");
  assert.equal(proposeNarrow(resumeRowOf({ ...base, mode: "none" })), "unreadable");
  assert.equal(proposeNarrow(resumeRowOf({ ...base, mode: "weird" })), "unreadable");
  assert.equal(proposeNarrow(resumeRowOf({ ...base, mode: "recovery_resume", lost: [1, "x"] })), "unreadable");
  assert.equal(proposeNarrow(resumeRowOf({ ...base, mode: "recovery_resume", lost: [3] })), "");
  assert.equal(proposeNarrow(resumeRowOf({ ...base, installation: "a".repeat(107) })), "no_command");
  const pg = page({ resume: [base] });
  assert.equal(stillAllowed(pg, "recovery.propose_unlock", "recovery-installation:s"), true);
  assert.equal(basisOf(pg, "s").digest, digest);
});

test("approve narrowing: resume row, mode, generation and digest", () => {
  const pg = page();
  const p10 = pg.proposals![0];
  assert.equal(approveNarrow(pg, p10), "");
  assert.equal(approveNarrow(page({ resume: [] }), p10), "resume_unknown");
  assert.equal(approveNarrow(page({ resume: [{ installation: "s", mode: "none", lost_digest: digest }] }), p10), "resume_unknown");
  assert.equal(approveNarrow(page({ resume: [{ installation: "s", mode: "odd", lost_digest: digest }] }), p10), "resume_unknown");
  const gen = page({ proposals: [unlock(10, "pending", {}, { unlock_generation: 1 })] });
  assert.equal(approveNarrow(gen, gen.proposals![0]), "changed");
  const dig = page({ proposals: [unlock(10, "pending", {}, { lost_digest: other })] });
  assert.equal(approveNarrow(dig, dig.proposals![0]), "changed");
  assert.equal(stillAllowed(dig, "recovery.approve", "recovery-proposal-10"), false);
  for (const l of locales) {
    assert.ok(k.narrow(l, "resume_unknown").length > 10);
    assert.ok(k.narrow(l, "changed").length > 10);
  }
  assert.equal(k.narrow("en", "changed"), "The installation changed after this was proposed. Reload or ask the proposer to withdraw it.");
});

test("the lock confirm with a known and an unknown generation, single administrator in every language", () => {
  assert.match(k.lockImpact("en", "s", 2), /from 2 to 3/);
  assert.match(k.lockImpact("en", "s", 2), /under generation 2/);
  assert.doesNotMatch(k.lockImpact("en", "s", null), /from \d/);
  assert.doesNotMatch(k.lockImpact("en", "s", null), /generation \d/);
  assert.equal(k.lockStrong("en"), "Takes effect now; no second approval: you alone lock this installation.");
  assert.match(k.lockStrong("ko"), /귀하 혼자 잠급니다/);
  assert.match(k.lockStrong("ar"), /أنت وحدك/);
  for (const l of locales) assert.notEqual(k.lockImpact(l, "s", 2), k.lockImpact(l, "s", null));
  const b = basisOf(page({ isolation: [{ installation: "s", state: "serving", generation: null }] }), "s");
  assert.equal(generationNumber(b), null);
});

test("the retired-begin sentence appears only with a pending begin row", () => {
  const begin = { number: 7, target: "recovery-proposal-7", kind: "begin", installation: "s", state: "pending", summary: { installation: "s" } };
  assert.deepEqual(pendingBeginsFor(page({ proposals: [begin] }), "s"), [7]);
  assert.equal(basisOf(page({ proposals: [begin] }), "s").begins, "7");
  assert.deepEqual(pendingBeginsFor(page({ proposals: [{ ...begin, state: "applied" }] }), "s"), []);
  assert.equal(basisOf(page(), "s").begins, "");
  for (const l of locales) assert.ok(k.beginRetired(l, "7").includes("7"));
  assert.match(k.beginRetired("en", "7"), /retired by this lock/);
});

test("impacts per mode: only the lock release says it checks no watermark", () => {
  const rel = k.releaseImpact("en", "s", 2);
  assert.match(rel, /checks no decision watermark/);
  assert.match(rel, /at generation 2/);
  assert.doesNotMatch(k.releaseImpact("en", "s", null), /at generation/);
  const res = k.resumeImpact("en", "p", "recovery-1");
  assert.doesNotMatch(res, /watermark/);
  assert.match(plain(res), /records recovery recovery-1 as completed/);
  assert.match(res, /\uE000recovery-1\uE001/);
  for (const l of locales) assert.notEqual(k.releaseImpact(l, "s", 2), k.resumeImpact(l, "s", "r"));
});

test("restore-set joins and their unknown fallbacks", () => {
  const pg = page();
  const p = pg.sets.rows![0];
  assert.deepEqual(evidenceOf(p), { kind: "counts", verified: 1, total: 3, mismatched: 1, pending: 1 });
  assert.deepEqual(evidenceOf(pg.sets.rows![1]), { kind: "not_applicable" });
  assert.equal(k.notApplicable("en"), "Not applicable (no recovery open)");
  assert.deepEqual(watermarkOf(pg, "p"), { kind: "pending", total: 2, lost: 0 });
  assert.deepEqual(watermarkOf(pg, "s"), { kind: "outside" });
  assert.deepEqual(watermarkOf(pg, "nope"), { kind: "unknown" });
  const zero = page({ resume: [{ installation: "p", mode: "recovery_resume", pending_total: 0, lost: [4, 5], lost_digest: digest }] });
  assert.deepEqual(watermarkOf(zero, "p"), { kind: "pending", total: 0, lost: 2 });
  assert.equal(k.noEventsPending("en"), "No decision events pending");
  assert.equal(k.declaredLost("en", 2), "2 declared lost");
  const nullLost = page({ resume: [{ installation: "p", mode: "recovery_resume", pending_total: 1, lost: "x" }] });
  assert.deepEqual(watermarkOf(nullLost, "p"), { kind: "pending", total: 1, lost: null });
  assert.deepEqual(reconciledOf(pg, p), { kind: "counts", reconciled: 1, total: 2 });
  assert.deepEqual(reconciledOf(pg, pg.sets.rows![1]), { kind: "none" });
  assert.equal(k.notApplicable("en"), "Not applicable (no recovery open)");
  assert.equal(k.publicLock("en", "recovering"), "Writes locked (in recovery)");
  assert.equal(k.publicLock("en", "locked"), "Writes locked");
  assert.equal(k.publicLock("en", "serving"), "Not locked");
  assert.equal(k.publicLock("en", "zz"), "Unknown (zz)");
  const bare = page({ sets: [{ installation: "p", state: "recovering", generation: 2, recovery: { state: "open", id: "recovery-2", as_of: "", manifest_digest: "", point_generation: 0, domains: [] } }] });
  const row = bare.sets.rows![0];
  assert.equal(noRestorePoint(row), true);
  assert.deepEqual(evidenceOf(row), { kind: "no_point" });
  assert.deepEqual(reconciledOf(bare, row), { kind: "no_point" });
  assert.equal(k.verifiedOf("en", 8, 9), "8 of 9 domains verified");
  assert.equal(k.eventsPending("en", 2), "2 decision events not yet reapplied");
  assert.equal(k.reconciledOf("en", 0, 2), "0 of 2 external systems reconciled");
});

test("mismatch wording: listed indices, an empty list and exactly 100", () => {
  assert.deepEqual(chunksOf("verified", []), { kind: "none" });
  assert.deepEqual(chunksOf("mismatch", [1]), { kind: "list", first100: false, indices: [1] });
  assert.deepEqual(chunksOf("mismatch", []), { kind: "count_differs" });
  assert.deepEqual(chunksOf("mismatch", null), { kind: "unknown" });
  const hundred = Array.from({ length: 100 }, (_, i) => i);
  assert.equal((chunksOf("mismatch", hundred) as { first100: boolean }).first100, true);
  assert.equal((chunksOf("mismatch", hundred.slice(1)) as { first100: boolean }).first100, false);
  assert.equal(k.countDiffers("en"), "Mismatch; no chunk index was reported (the item count differs).");
  for (const l of locales) for (const f of [k.chunkOne, k.chunkMany, k.chunkFirst100]) assert.ok(f(l).includes("{i}"));
});

test("the begin kind, its summary and the applied note", () => {
  assert.equal(k.proposalKind("en", "begin"), "Recovery opened (isolated restore record)");
  const begin = recoveryProposalOf({ number: 7, kind: "begin", summary: { installation: "p", as_of: "2026-11-17T08:00:00Z", point_generation: 1, manifest_digest: digest, skew_seconds: 9, unlock_generation: 3 } });
  const s = summaryOf(begin);
  assert.deepEqual(Object.keys(s.parts).sort(), ["as_of", "installation", "manifest_digest", "point_generation"]);
  assert.match(summaryWords.begin[0], /no data was restored/);
  assert.equal(k.appliedNote("en"), "Applied means recorded in the recovery registry.");
  assert.ok(k.appliedNote("ko").length > 0 && k.appliedNote("ar").length > 0);
});

test("no copy claims a restore, prune, deletion, host, network or credential change", () => {
  const allowed = [
    /Nothing is restored, pruned or deleted/, /Nothing is restored/, /no data was restored/, /Nothing is restored here/, /does not verify any restore/, /not connected/, /No migration package/,
    /아무것도 복원/, /복원되지 않음/, /복원·결정 워터마크·외부 결과는 검증되지/, /لا تتم أي استعادة/, /لم تُستعد/, /لا يتحقق من أي استعادة/,
  ];
  const bad = [/\brestored\b/i, /restore completed/i, /복원되었/, /복원 완료/, /تمت الاستعادة/, /اكتملت الاستعادة/, /\bpruned\b/i, /\bdeleted\b/i, /(?<!No )host (was|is) (stopped|changed)/i, /credential (was|is) revoked/i];
  const texts: string[] = [];
  for (const table of Object.values(allWordTables)) for (const w of Object.values(table)) texts.push(...w);
  for (const l of locales) {
    texts.push(k.statement(l), k.setsLead(l), k.isolationLead(l), k.reconciliationLead(l), k.migrationLead(l), k.resumeLead(l), k.resumeLeadStrong(l), k.modeReleaseStrong(l));
    texts.push(k.releaseImpact(l, "s", 2), k.resumeImpact(l, "s", "r"), k.lockImpact(l, "s", 2), k.lockedDone(l, "s"), k.approvedDone(l, "1", "s"), k.withdrawImpact(l));
  }
  for (const text of texts) {
    const stripped = allowed.reduce((s, re) => s.replace(re, ""), text);
    for (const re of bad) assert.doesNotMatch(stripped, re, text);
  }
  assert.match(k.lockedDone("en", "s"), /No host was changed/);
});

test("closed rows: applied shows the approver, withdrawn the withdrawer, retired neither", () => {
  assert.equal(actor("admin@example.invalid", "sample-admin"), "admin@example.invalid");
  assert.equal(actor("", "sample-admin"), "sample-admin");
  assert.match(k.approvedBy("en", "admin@example.invalid"), /^Approved by/);
  assert.match(k.withdrawnBy("en", "x"), /^Withdrawn by/);
  assert.equal(k.retiredRow("en"), "Retired: the recovery registry no longer holds this proposal");
  const o = ordered(page({ proposals: [unlock(3, "applied"), unlock(9, "pending"), unlock(5, "retired", { approve_blocked: "retired", withdraw_blocked: "retired" }), unlock(12, "pending")] }).proposals!);
  assert.deepEqual(o.pending.map((p) => p.number), [12, 9]);
  assert.deepEqual(o.closed.map((p) => p.number), [5, 3]);
  const retired = page({ proposals: [unlock(5, "retired", { can_approve: false, approve_blocked: "retired", can_withdraw: false, withdraw_blocked: "retired" })] });
  assert.equal(stillAllowed(retired, "recovery.approve", "recovery-proposal-5"), false);
  assert.equal(stillAllowed(retired, "recovery.withdraw", "recovery-proposal-5"), false);
});

test("resends skip read-based narrowing and restate the frozen snapshot", () => {
  forget();
  setScope("ws", "u1", "csrf");
  const before = page({ resume: [{ installation: "s", mode: "lock_release", lost: [], lost_digest: digest, can_propose_unlock: true, unlock_blocked: "" }], proposals: [] });
  const frozen = basisOf(before, "s");
  pendingFor("recovery.propose_unlock", "recovery-installation:s", { lost: digest, rationale: "why" }, "r1", () => ({ request: "a", key: "b" }), [], basisContext(frozen));
  const after = page({ isolation: [{ installation: "s", state: "serving", generation: 3 }], resume: [{ installation: "s", mode: "none", lost: [], lost_digest: other, can_propose_unlock: false, unlock_blocked: "not_recovering" }] });
  const kept = pendingNow("recovery.propose_unlock", "recovery-installation:s");
  assert.ok(kept);
  assert.equal(frozenTargetOk("recovery.propose_unlock", "recovery-installation:s"), true);
  const b = basisFrom(kept!.context);
  assert.deepEqual([b.mode, b.state, b.generation, b.digest], ["lock_release", "locked", "2", digest]);
  assert.deepEqual(lostList(b), []);
  assert.equal(basisOf(after, "s").mode, "none");
  assert.match(k.repeat("en"), /exactly as it was sent/);
  const pa = page();
  pendingFor("recovery.approve", "recovery-proposal-10", { rationale: "ok" }, "r1", () => ({ request: "c", key: "d" }), [], basisContext(basisOf(pa, "s"), { proposal: "10" }));
  const pb = page({ proposals: [unlock(10, "pending", {}, { unlock_generation: 1 })] });
  assert.equal(approveNarrow(pb, pb.proposals![0]), "changed");
  assert.ok(pendingNow("recovery.approve", "recovery-proposal-10"));
  assert.equal(basisFrom(pendingNow("recovery.approve", "recovery-proposal-10")!.context).generation, "2");
});

test("open calls per source and the other-workspace flag", () => {
  const iso = (o: Over) => page({ isolation: [{ installation: "p", ...o }] }).isolation.rows![0];
  assert.deepEqual(callsOf(iso({ open_calls_source: "recovery", open_calls: [], open_calls_total: 0 })), { kind: "empty" });
  assert.deepEqual(callsOf(iso({ open_calls_source: "recovery", open_calls: [], open_calls_total: "0" })), { kind: "unknown", code: "" });
  assert.deepEqual(callsOf(iso({ open_calls_source: "recovery", open_calls: [{ id: "c" }], open_calls_total: 1 })), { kind: "listed" });
  assert.deepEqual(callsOf(iso({ open_calls_source: "not_shown", open_calls: [], open_calls_total: 0 })), { kind: "not_shown" });
  assert.deepEqual(callsOf(iso({ open_calls_source: "elsewhere" })), { kind: "unknown", code: "elsewhere" });
  assert.equal(iso({ other_scope_present: true }).other_scope_present, true);
  assert.equal(iso({ other_scope_present: false }).other_scope_present, false);
  assert.equal(iso({}).other_scope_present, null);
  assert.equal(k.callsNotShown("en"), "Open calls are listed only while a recovery is open.");
  assert.equal(k.otherScopeUnknown("en"), "Calls in other workspaces: Unknown");
});

test("high water and stop times", () => {
  assert.deepEqual(highWaterOf("s", 2), { kind: "number", n: 2 });
  assert.deepEqual(highWaterOf("", null), { kind: "no_series" });
  assert.deepEqual(highWaterOf("s", null), { kind: "unknown" });
  assert.deepEqual(stoppedOf("serving", ""), { kind: "not_stopped" });
  assert.deepEqual(stoppedOf("locked", ""), { kind: "unknown" });
  assert.deepEqual(stoppedOf("locked", "0001-01-01T00:00:00Z"), { kind: "unknown" });
  assert.deepEqual(stoppedOf("locked", "2026-11-17T08:00:00Z"), { kind: "time", at: "2026-11-17T08:00:00Z" });
  assert.equal(timeKnown("0001-01-01T00:00:00Z"), false);
  assert.equal(timeKnown("nope"), false);
});

test("reasons: trimming, whitespace, bytes and field mappings", () => {
  assert.equal(reasonProblem("   "), "missing");
  assert.equal(reasonProblem(" ok "), "");
  assert.equal(utf8Bytes("가"), 3);
  assert.equal(reasonProblem("가".repeat(85)), "");
  assert.equal(reasonProblem("가".repeat(86)), "long");
  assert.equal(fieldReaction("recovery.propose_unlock", ["lost"]), "changed");
  assert.equal(fieldReaction("recovery.lock", ["lost"]), "built");
  assert.equal(fieldReaction("recovery.approve", ["rationale"]), "reason");
  for (const f of ["target_id", "expected_revision", "idempotency_key", "request_id", "client", "body", "zzz"]) assert.equal(fieldReaction("recovery.propose_unlock", [f, "rationale"]), "built", f);
});

test("blocker sentences for every code in every language, missing and unknown", () => {
  for (const l of locales) {
    for (const code of Object.keys(blockerWords)) assert.ok(k.blocker(l, code).length > 10, code);
    assert.ok(k.blocker(l, "missing").length > 5);
    assert.ok(k.blocker(l, "zzz").includes("zzz"));
    for (const n of Object.keys(narrowWords)) assert.ok(k.narrow(l, n).length > 10);
    for (const c of Object.keys(keptWords)) assert.ok(k.kept(l, c).length > 10);
  }
  assert.equal(k.blocker("en", "missing"), "This is not available now.");
  assert.equal(plain(k.blocker("en", "zzz")), "This is not available now (zzz).");
  assert.equal(k.blocker("en", "incomplete"), "The recovery is not complete: a domain is not verified, an external system is not reconciled, or decision events are still pending.");
  assert.equal(k.blocker("en", "own_proposal"), "You proposed this. Another administrator must approve it.");
});

test("summaries per kind use only their own fields, with an unknown fallback", () => {
  const mk = (kind: string, summary: Over) => recoveryProposalOf({ number: 1, kind, summary: { installation: "i", skew_seconds: 9, writer: "w", external: "x", external_kind: "erp", verifier: "v", unlock_generation: 4, lost_digest: digest, ...summary } });
  assert.deepEqual(Object.keys(summaryOf(mk("installation", {})).parts).sort(), ["installation", "skew"]);
  assert.deepEqual(Object.keys(summaryOf(mk("writer", {})).parts).sort(), ["installation", "writer"]);
  assert.deepEqual(Object.keys(summaryOf(mk("external", {})).parts).sort(), ["external", "installation", "kind"]);
  assert.deepEqual(Object.keys(summaryOf(mk("verifier", {})).parts).sort(), ["installation", "verifier"]);
  assert.deepEqual(Object.keys(summaryOf(mk("unlock", {})).parts).sort(), ["installation", "lost_digest", "unlock_generation"]);
  assert.deepEqual(Object.keys(summaryOf(mk("prune", {})).parts), ["installation"]);
  assert.deepEqual(Object.keys(summaryOf(mk("abandon", {})).parts), ["installation"]);
  assert.equal(summaryOf(mk("weird", {})).known, false);
  assert.equal(mk("writer", {}).summary.unlock_generation, null);
  for (const l of locales) for (const kind of Object.keys(summaryWords)) for (const slot of Object.keys(summaryOf(mk(kind, {})).parts)) assert.ok(summaryWords[kind][locales.indexOf(l)].includes("{" + slot + "}"), kind + " " + slot + " " + l);
});

test("unknown versus zero for numbers, lists, flags and times", () => {
  const pg = page({
    sets: [{ installation: "p", state: "serving", generation: 0, recovery: { state: "none" }, history: null, history_total: "3" }],
    isolation: [],
    resume: [{ installation: "p", mode: "none", pending_total: 0, lost: null, lost_digest: digest }],
  });
  assert.equal(pg.sets.rows![0].generation, 0);
  assert.equal(pg.sets.rows![0].history, null);
  assert.equal(pg.sets.rows![0].history_total, null);
  assert.deepEqual(lostOf(null), { kind: "unknown" });
  assert.deepEqual(lostOf([]), { kind: "none" });
  const many = Array.from({ length: 53 }, (_, i) => i);
  assert.deepEqual(lostOf(many), { kind: "list", shown: many.slice(0, 50), rest: 3 });
  assert.equal(k.andMore("en", 3), " and 3 more");
  assert.equal(generationNumber(basisOf(pg, "p")), 0);
});

test("every vocabulary key reads in EN, KO and AR with an unknown fallback", () => {
  for (const [name, table] of Object.entries(allWordTables)) {
    for (const [key, w] of Object.entries(table)) {
      assert.equal(w.length, 3, name + "." + key);
      for (const s of w) assert.ok(s.trim().length > 0, name + "." + key);
    }
  }
  assert.equal(k.installState("en", "recovering"), "In recovery");
  assert.equal(k.modeWord("ko", "lock_release"), "잠금 해제(복구 없음)");
  assert.equal(k.domain("ar", "outbox"), "صندوق الصادر");
  assert.equal(k.domainStatus("en", "nope"), "Unknown (nope)");
  assert.equal(k.migrationModule("en", "balances"), "Opening balances");
});

test("failure reactions and the attempt store", () => {
  assert.equal(react("invalid", ["lost"]), "field");
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
  assert.equal(react("x", []), "failed");
  assert.equal(keepsAttempt("retry"), true);
  forget();
  setScope("ws", "u1", "s1");
  let n = 0;
  const fresh = () => ({ request: "r" + ++n, key: "k" + n });
  const t = "recovery-installation:s";
  pendingFor("recovery.lock", t, { rationale: "a" }, "r1", fresh);
  settle("recovery.lock", t, "retry");
  assert.ok(pendingNow("recovery.lock", t));
  discard("recovery.lock", t, { rationale: "b" });
  assert.equal(pendingNow("recovery.lock", t), undefined);
  pendingFor("recovery.lock", t, { rationale: "a" }, "r1", fresh);
  settle("recovery.lock", t, "partial");
  assert.equal(pendingNow("recovery.lock", t), undefined);
  pendingFor("recovery.lock", t, { rationale: "a" }, "r1", fresh);
  setScope("ws", "u1", "s2");
  assert.equal(pendingNow("recovery.lock", t), undefined);
  keepDraft("recovery.propose_unlock", t, { rationale: "draft" });
  assert.equal(draftOf("recovery.propose_unlock", t).rationale, "draft");
  setScope("ws", "u2", "s2");
  assert.equal(draftOf("recovery.propose_unlock", t).rationale, undefined);
  keepDraft("recovery.propose_unlock", t, { rationale: "draft" });
  forget();
  setScope("ws", "u2", "s2");
  assert.equal(draftOf("recovery.propose_unlock", t).rationale, undefined);
});

test("kept refusal sentences stay only while the reloaded row shows no blocker", () => {
  forget();
  setScope("ws", "u1", "s1");
  const lockable = page({ isolation: [{ installation: "s", state: "serving", generation: 2, can_lock: true, lock_blocked: "" }] });
  for (const [command, target, open, closed] of [
    ["recovery.lock", "recovery-installation:s", lockable, page()],
    ["recovery.approve", "recovery-proposal-10", page(), page({ proposals: [unlock(10, "applied")] })],
    ["recovery.withdraw", "recovery-proposal-10", page({ proposals: [unlock(10, "pending", { can_withdraw: true, withdraw_blocked: "" })] }), page({ proposals: [unlock(10, "withdrawn")] })],
    ["recovery.propose_unlock", "recovery-installation:s", page({ resume: [{ installation: "s", mode: "lock_release", lost: [], lost_digest: digest, can_propose_unlock: true, unlock_blocked: "" }] }), page()],
  ] as const) {
    settle(command, target, "hint");
    pruneRefusals(open);
    assert.equal(refusedNow(command, target), true, command);
    pruneRefusals(closed);
    assert.equal(refusedNow(command, target), false, command);
    settle(command, target, "hint");
    settle(command, target, "ok");
    assert.equal(refusedNow(command, target), false, command);
  }
  settle("recovery.lock", "recovery-installation:gone", "hint");
  pruneRefusals(lockable);
  assert.equal(refusedNow("recovery.lock", "recovery-installation:gone"), false);
});

test("denied on a command and the vanished-attempt notice", () => {
  forget();
  setScope("ws", "u1", "s1");
  const without = page({ proposals: [] });
  assert.equal(deniedOutcome("recovery.approve", "recovery-proposal-10", true, without), "applied_proposal");
  assert.equal(deniedOutcome("recovery.approve", "recovery-proposal-10", false, without), "gone");
  assert.equal(deniedOutcome("recovery.approve", "recovery-proposal-10", true, page()), "gone");
  assert.equal(deniedOutcome("recovery.lock", "recovery-installation:zz", true, page()), "gone");
  assert.equal(k.vanishedProposal("en"), "Your earlier request may have been applied. The proposal is no longer listed.");
  assert.equal(k.gone("en", true), "This proposal is no longer available.");
  assert.equal(k.gone("en", false), "This installation is no longer available.");
  pendingFor("recovery.approve", "recovery-proposal-10", { rationale: "x" }, "r1", () => ({ request: "a", key: "b" }), [], { proposal: "10" });
  assert.equal(listed(page(), "recovery.approve", "recovery-proposal-10"), true);
  assert.deepEqual(vanished(page(), "resume"), []);
  const gone = vanished(without, "resume");
  assert.equal(gone.length, 1);
  assert.equal(gone[0].context.proposal, "10");
  assert.deepEqual(vanished(without, "isolation"), []);
});

test("duplicate announcements compare against the pre-send snapshot", () => {
  const pg = page({ proposals: [unlock(10, "pending")] });
  const snap = snapshotOf(pg, "recovery.propose_unlock");
  assert.deepEqual(snap, ["10"]);
  assert.equal(duplicateIn(snap, "10"), true);
  assert.equal(duplicateIn(snap, "11"), false);
  assert.deepEqual(snapshotOf(pg, "recovery.lock"), []);
  assert.equal(k.duplicate("en", "10"), "This was already proposed as #10.");
  forget();
  setScope("ws", "u1", "s1");
  const first = pendingFor("recovery.propose_unlock", "recovery-installation:s", { lost: digest, rationale: "x" }, "r1", () => ({ request: "a", key: "b" }), snap);
  const resend = pendingFor("recovery.propose_unlock", "recovery-installation:s", { lost: digest, rationale: "x" }, "r2", () => ({ request: "c", key: "d" }), ["10", "11"]);
  assert.deepEqual(resend.snapshot, first.snapshot);
  assert.equal(duplicateIn(resend.snapshot, "11"), false);
});

test("URL selections are validated against the read", () => {
  const pg = page();
  const q = (s: string) => new URLSearchParams(s);
  assert.deepEqual(selectionOf(q("installation=s"), pg, "isolation"), { installation: "s", proposal: "" });
  assert.deepEqual(selectionOf(q("installation=nope"), pg, "isolation"), { installation: "", proposal: "" });
  assert.deepEqual(selectionOf(q("installation=s"), pg, "reconciliation"), { installation: "", proposal: "" });
  assert.deepEqual(selectionOf(q("installation=p"), pg, "reconciliation"), { installation: "p", proposal: "" });
  assert.deepEqual(selectionOf(q("installation=s"), pg, "migration"), { installation: "", proposal: "" });
  assert.deepEqual(selectionOf(q("installation=bad id"), pg, "sets"), { installation: "", proposal: "" });
  assert.deepEqual(selectionOf(q("proposal=10"), pg, "resume"), { installation: "", proposal: "10" });
  assert.deepEqual(selectionOf(q("proposal=010"), pg, "resume"), { installation: "", proposal: "" });
  assert.deepEqual(selectionOf(q("proposal=99"), pg, "resume"), { installation: "", proposal: "" });
  assert.deepEqual(selectionOf(q("proposal=10"), pg, "sets"), { installation: "", proposal: "" });
  assert.deepEqual(selectionOf(q("installation=s"), null, "sets"), { installation: "", proposal: "" });
});

test("history: a missing or inconsistent list reads unknown, a real empty list reads empty", () => {
  const row = (history: unknown, total: unknown) => page({ sets: [{ installation: "p", state: "serving", generation: 1, recovery: { state: "none" }, history, history_total: total }] }).sets.rows![0];
  assert.deepEqual(historyOf(row([], 0)), { kind: "empty" });
  assert.deepEqual(historyOf(row([], 3)), { kind: "unknown" });
  assert.deepEqual(historyOf(row([], "x")), { kind: "unknown" });
  assert.deepEqual(historyOf(row(null, 0)), { kind: "unknown" });
  assert.deepEqual(historyOf(row([{ recovery: "r" }], 1)), { kind: "list" });
  assert.deepEqual(historyOf(row([{ recovery: "r" }], 60)), { kind: "more", total: 60 });
});

test("approve narrowing with an unknown current generation has its own sentence", () => {
  const pg = page({ isolation: [{ installation: "s", state: "locked", generation: "x" }], sets: [{ installation: "s", state: "locked", generation: null, recovery: { state: "none" } }] });
  assert.equal(approveNarrow(pg, pg.proposals![0]), "generation_unknown");
  for (const l of locales) assert.ok(k.narrow(l, "generation_unknown").length > 10);
  assert.notEqual(k.narrow("en", "generation_unknown"), k.narrow("en", "changed"));
});

test("result sentences fall back to Unknown for an empty state and mark IDs as isolates", () => {
  assert.equal(plain(k.resultUnknown("en", "")), "The request was recorded with an unknown result (Unknown). Reload to check.");
  assert.match(k.resultUnknown("en", "odd"), /\uE000odd\uE001/);
  assert.match(k.lockedDone("en", "sample-standby"), /^\uE000sample-standby\uE001 is locked/);
  assert.match(k.lockVerb("ar", "s"), /\uE000s\uE001/);
});

test("date stamps keep the locale's date-time separator at the end of the date part", () => {
  const at = "2026-11-17T07:07:00Z";
  const en = stampParts("en", at)!;
  assert.equal(en.date, "Nov 17, 2026,");
  assert.equal(en.time.replace(/\s/g, " "), "7:07 AM UTC");
  const ko = stampParts("ko", at)!;
  assert.ok(ko.time.startsWith("오전") && ko.time.endsWith(" UTC"), ko.time);
  assert.ok(!/\s$/.test(ko.date) && ko.date.includes("2026"), ko.date);
  const ar = stampParts("ar", at)!;
  assert.ok(ar.date.endsWith("،") && ar.time.endsWith(" UTC"), JSON.stringify(ar));
  assert.equal(stampParts("en", ""), null);
  assert.equal(stampParts("en", "0001-01-01T00:00:00Z"), null);
});
