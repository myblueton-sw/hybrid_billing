import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, type SelfEnd, type SelfPage } from "../../../../service/admin";
import { selfEndOf } from "../../../../service/self-end";
import { canChange, changeOutcome, checkChange, goneCount, groupOf, heldShown, heldTags, openByDefault, others, presentHandles, rowOutcome, running, submitOutcome } from "./live-model";
import { e, stateSentence } from "./end-copy";

const page = (change: Partial<SelfPage>): SelfPage => ({
  user_id: "u", login: "l", email: "e", workspace: "w", role: "member", identities: [], password_set: true, installation_administrator: false,
  sessions: [{ handle: "a", current: true, signed_in: "1", ends: "2" }, { handle: "b", current: false, signed_in: "0", ends: "1" }],
  sessions_generation: "g", session_ends: [], idle_minutes: 30, max_age_seconds: 0, signed_in_at: "1", authenticated_at: "1", now: "1", ...change,
});

test("local checks run before any request", () => {
  assert.equal(checkChange("", "a", "a"), "missing");
  assert.equal(checkChange("x", "", ""), "missing");
  assert.equal(checkChange("x", "a", ""), "missing");
  assert.equal(checkChange("x", "a", "b"), "mismatch");
  assert.equal(checkChange("a", "a", "a"), "same");
  assert.equal(checkChange("x", "a", "a"), "");
});

test("the form hides without a password and other sessions exclude this one", () => {
  assert.equal(canChange(page({})), true);
  assert.equal(canChange(page({ password_set: false })), false);
  assert.deepEqual(others(page({})).map((s) => s.handle), ["b"]);
});

test("change failures choose the lockout sign-out only after a wrong attempt", () => {
  for (const f of ["wrong", "locked", "limited", "form", "conflict"] as const) assert.equal(changeOutcome(new AdminError(f), false), f);
  assert.equal(changeOutcome(new AdminError("unauthenticated"), true), "lockedOut");
  assert.equal(changeOutcome(new AdminError("unauthenticated"), false), "signedOut");
  assert.equal(changeOutcome(new AdminError("busy"), true), "busy");
  for (const f of ["unavailable", "partial", "invalid", "denied"] as const) assert.equal(changeOutcome(new AdminError(f), false), "unknown", f);
  assert.equal(changeOutcome(new Error("x"), false), "unknown");
});

const item = (id: string, state: string, reason = "") => ({ item: id, state, reason });
const ended = (handle: string, c3: string) => ({ handle, signed_in: "s", result: "ended", items: [item("C1", "done"), item("C3", c3), item("C7", "outside_scope", "cannot_recall")] });
const op = (change: Record<string, unknown>): SelfEnd => selfEndOf({ operation: "session-end-1", kind: "others", state: "complete", count: 1, sessions: [ended("b", "done")], not_included: [], ...change });

test("a submit failure maps by kind; a 422 on others reads as a changed list", () => {
  assert.equal(submitOutcome(new AdminError("invalid"), "others"), "changed");
  assert.equal(submitOutcome(new AdminError("invalid"), "one"), "gone");
  assert.equal(submitOutcome(new AdminError("unauthenticated"), "others"), "signedOut");
  assert.equal(submitOutcome(new AdminError("limited"), "one"), "limited");
  assert.equal(submitOutcome(new AdminError("reauth"), "others"), "reauth");
  assert.equal(submitOutcome(new AdminError("reauth_cancelled"), "others"), "reauth");
  assert.equal(submitOutcome(new AdminError("refused"), "others"), "refused");
  for (const f of ["busy", "unavailable", "partial", "conflict"] as const) assert.equal(submitOutcome(new AdminError(f), "others"), "unknown", f);
  assert.equal(submitOutcome(new Error("x"), "one"), "unknown");
});

test("per-session outcomes keep already gone apart from the already-cleared cleanup state", () => {
  assert.equal(rowOutcome(selfEndOf({ sessions: [ended("b", "done")] }).sessions[0]), "complete");
  assert.equal(rowOutcome(selfEndOf({ sessions: [ended("b", "already_ended")] }).sessions[0]), "complete");
  assert.equal(rowOutcome(selfEndOf({ sessions: [ended("b", "pending")] }).sessions[0]), "running");
  assert.equal(rowOutcome(selfEndOf({ sessions: [ended("b", "failed")] }).sessions[0]), "failed");
  assert.equal(rowOutcome(selfEndOf({ sessions: [{ handle: "g", result: "already_gone" }] }).sessions[0]), "gone");
  assert.equal(groupOf({ item: "C3", state: "already_ended", reason: "" }), "done");
  assert.equal(groupOf({ item: "C7", state: "outside_scope", reason: "" }), "outside_scope");
  assert.equal(goneCount(op({ sessions: [ended("b", "done"), { handle: "g", result: "already_gone" }, { handle: "h", result: "already_gone" }] })), 2);
  assert.notEqual(e.countCleared("en"), e.rowGone("en"));
  assert.notEqual(e.countCleared("ko"), e.rowGone("ko"));
  assert.notEqual(e.countCleared("ar"), e.rowGone("ar"));
});

test("the held block, tags and open state follow the newest others operation and the current read", () => {
  const held = op({ not_included: [{ handle: "n1" }, { handle: "n2" }] });
  assert.equal(heldShown(held), true);
  assert.equal(heldShown(op({ kind: "one", not_included: [{ handle: "n1" }] })), false);
  const p = page({ sessions: [{ handle: "a", current: true, signed_in: "1", ends: "2" }, { handle: "n1", current: false, signed_in: "1", ends: "2" }], session_ends: [held] });
  assert.deepEqual([...heldTags(p)], ["n1"]);
  assert.deepEqual([...heldTags(page({ session_ends: [op({ kind: "one" }), held] }))], []);
  assert.deepEqual([...presentHandles(p)], ["a", "n1"]);
  assert.equal(openByDefault(op({}), true, new Set()), true);
  assert.equal(openByDefault(op({}), false, new Set()), false);
  assert.equal(openByDefault(op({ state: "cleanup_incomplete" }), false, new Set()), true);
  assert.equal(openByDefault(held, false, new Set(["n2"])), true);
  assert.equal(running(page({ session_ends: [op({}), op({ operation: "x", state: "cleanup_running" })] })), true);
  assert.equal(running(page({ session_ends: [op({})] })), false);
  assert.equal(running(null), false);
});

test("copy: AR plural button forms, the spec's KO sentences and a closed state set", () => {
  assert.equal(e.button("en", 1), "End 1 other session");
  assert.equal(e.button("en", 3), "End 3 other sessions");
  assert.equal(e.button("ko", 2), "다른 세션 2개 종료");
  const ar = [1, 2, 3, 11].map((n) => e.button("ar", n));
  assert.equal(new Set(ar).size, 4);
  assert.equal(ar[0], "إنهاء جلسة أخرى واحدة");
  assert.equal(ar[1], "إنهاء الجلستين الأخريين");
  assert.equal(e.running("ko"), "접근 차단은 적용되었지만 일부 세션 정리가 진행 중입니다.");
  assert.equal(e.unknown("ko"), "변경 결과를 확인하지 못했습니다. 다시 제출하기 전에 현재 연결·세션 상태를 확인해 주세요.");
  assert.match(e.running("ar"), /حجب/);
  assert.match(e.accessBlocked("ar"), /حجب/);
  assert.equal(e.equation("en", 3, 1, 1, 0, 1), "3 sessions: 1 done, 1 already cleared, 0 in progress, 1 not finished.");
});

test("render review copy: no revocation word for recall, count-neutral Arabic and distinct KO result titles", () => {
  assert.doesNotMatch(e.noRecall("ko"), /회수/);
  assert.doesNotMatch(e.reason("ko", "cannot_recall"), /회수/);
  for (const k of [1, 2, 3, 11]) {
    for (const text of [e.heldRest("ar", k), e.heldAnnounce("ar", k), e.gone("ar", k), e.incompleteCount("ar", k), e.lead("ar", k), e.complete("ar", k), e.incomplete("ar", k)]) {
      assert.doesNotMatch(text, /جلستين|لـ\d/, text);
    }
  }
  assert.notEqual(e.opOthers("ko"), e.entry("ko"));
  assert.match(e.heldRest("en", 1), /Review it below and end it if you do not recognize it\./);
  assert.match(e.heldRest("en", 2), /Review them below and end any you do not recognize\./);
  assert.equal(e.wordIncomplete("ko"), "미완료");
  assert.equal(e.goPassword("en"), "Change your password");
  assert.notEqual(e.reason("en", "denied_set_full"), "");
});

test("render review round 2: truthful overflow copy, outcome titles and the count = 0 guard", () => {
  assert.equal(e.reason("en", "denied_set_full"), "Stopped: the service's list of ended sessions waiting for cleanup is full. Access stays blocked. New sign-ins to this account are refused until 12 hours after this stop.");
  assert.equal(e.reason("ko", "denied_set_full"), "중지됨: 서비스의 정리 대기 종료 세션 목록이 가득 찼습니다. 접근은 계속 차단됩니다. 이 계정의 새 로그인은 이 중지 시점부터 12시간 동안 거부됩니다.");
  assert.equal(e.reason("ar", "denied_set_full"), "توقّف: قائمة الخدمة للجلسات المنتهية التي تنتظر التنظيف ممتلئة. يبقى الوصول محجوبًا. تُرفض عمليات تسجيل الدخول الجديدة لهذا الحساب لمدة 12 ساعة من هذا التوقف.");
  assert.deepEqual([e.titleUnknown("en"), e.titleUnknown("ko"), e.titleUnknown("ar")], ["Result not confirmed", "결과 미확인", "لم تُؤكَّد النتيجة"]);
  assert.deepEqual([e.titleNothing("en"), e.titleNothing("ko"), e.titleNothing("ar")], ["Nothing was ended", "종료하지 않았습니다", "لم يُنهَ أي شيء"]);
  assert.equal(e.blockedNone("en"), "None: every session in your list had already ended");
  for (const state of ["cleanup_running", "cleanup_incomplete", "complete"]) {
    assert.equal(stateSentence("en", op({ count: 0, state, sessions: [] })), e.completeNone("en"), state);
  }
  assert.equal(stateSentence("ko", op({ state: "cleanup_running" })), e.running("ko"));
});
