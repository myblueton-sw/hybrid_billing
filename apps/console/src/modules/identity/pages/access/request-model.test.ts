import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, ledgerOf, type AccessPage, type AccessRequest, type CleanupItem, type Revocation } from "../../../../service/admin";
import {
  additionsPaused, cleanupGroups, descendantsOf, failureWords, itemWords, jobWords, keepAttempt, ledgerLines, pendingRevokeFor, requestState, requestsOf, settingName,
  settingSource, settingValue, settled, shortDigest, stateWords, unconfirmedFor,
} from "./request-model";

const req = (change: Partial<AccessRequest>): AccessRequest => ({
  id: "access-request-1", change: "add", subject: "u", grant: "", action: "view", kind: "focus_data", fields: ["rows"], customer: "all", days: 10, digest: "abcdef0123456789",
  state: "pending", current: true, proposer: "p", proposed_at: "", expires_at: "", reason: "r", decided_by: "", decided_at: "", decision_reason: "", result: "",
  eligible_approvers: 1, can_approve: false, can_reject: false, can_withdraw: false, may_end: [], ...change,
});
const rev = (change: Partial<Revocation>): Revocation => ({
  id: "revocation-1", origin: "this_process", recorded_at: "", target_kind: "grant", target: "g1", target_present: true, replay: "", state: "revocation_complete",
  history: [], ended_delegations: [], deny_generation: 3, job_state: "succeeded", job_reason: "", requested_by: "p", approved_by: "a", request: "", received_at: "", deny_at: "",
  receipt: { sequence: 2, watermark: "2:ab", reason: "" }, verified: true, cleanup: [], completed_at: "", ...change,
});
const page = (change: Partial<AccessPage>): AccessPage => ({
  revision: "r1", grants: [], delegations: [], catalog: [], subjects: [], now: "", you: "me", started_at: "", settings: [], requests: [], revocations: [], ledger: ledgerOf({ state: "ok" }),
  ...change,
});
const grant = { id: "g1", subject: "u", client: "", action: "view", kind: "focus_data", fields: ["rows"], scope: { customer: "all", legal_entity: "all", contract: "all", period: "any" },
  not_before: "", not_after: "", state: "revoked" as const, effective: false, version: 2 };
const deleg = (id: string, parent: string, state: "active" | "revoked" | "ended" = "active") => ({ ...grant, id, delegator: "a", delegate: "d-" + id, parent, redelegate: true, state, effective: state === "active" });

test("requests order and pending revocations", () => {
  const p = page({ requests: [req({ id: "a" }), req({ id: "b", state: "applied" }), req({ id: "c", change: "revoke", grant: "g1", state: "pending" }), req({ id: "d", change: "revoke", grant: "g2", state: "rejected" })] });
  const split = requestsOf(p);
  assert.deepEqual(split.pending.map((r) => r.id), ["a", "c"]);
  assert.deepEqual(split.ended.map((r) => r.id), ["b", "d"]);
  assert.equal(pendingRevokeFor(p, "g1")?.id, "c");
  assert.equal(pendingRevokeFor(p, "g2"), undefined);
  assert.equal(shortDigest("abcdef0123456789"), "abcdef012345");
  for (const st of ["pending", "applied", "rejected", "withdrawn", "expired", "superseded"] as const) assert.ok(requestState("en", req({ state: st })).length > 0);
  assert.equal(requestState("ko", req({ state: "superseded" })), "최신 아님");
});

test("all five revocation states have the spec copy and a job state", () => {
  const ko: Record<string, string> = {
    received: "회수 요청을 접수했습니다. 보안 차단 확정은 아직 확인되지 않았습니다.",
    deny_confirmed: "회수 결정과 접근 차단이 확정되었습니다. 후속 자격 정리 상태를 확인하세요.",
    cleanup_running: "접근 차단 효력은 유지되며 세션·키·캐시·작업을 정리 중입니다.",
    revocation_complete: "회수 차단과 지정 범위의 후속 정리가 완료되었습니다.",
    decision_unconfirmed: "독립 결정 기록 확정을 확인하지 못했습니다. 회수 성공으로 표시하지 않습니다.",
  };
  for (const [state, copy] of Object.entries(ko)) {
    const r = rev({ state: state as Revocation["state"] });
    assert.equal(stateWords("ko", r).copy, copy);
    for (const l of ["en", "ar"] as const) assert.ok(stateWords(l, r).name && stateWords(l, r).copy);
  }
  assert.equal(stateWords("en", rev({ state: "strange" as Revocation["state"] })).name, "Decision not confirmed");
  assert.equal(jobWords("en", rev({ job_state: "waiting" })), "Job: waiting (decision not confirmed)");
  assert.equal(jobWords("en", rev({ job_state: "succeeded" })), "Job: succeeded");
  assert.equal(jobWords("en", rev({ job_state: "odd" as Revocation["job_state"] })), "Job: running");
  assert.equal(failureWords("en", "ledger_isolated"), "The ledger is isolated");
  assert.equal(failureWords("en", "other"), "other");
});

test("cleanup items group apart and outside scope is never settled", () => {
  const items: CleanupItem[] = [
    { item: "C1", state: "done", reason: "deny_generation", refs: [] }, { item: "C2", state: "not_applicable", reason: "no_permission_in_sessions", refs: [] },
    { item: "C5", state: "outside_scope", reason: "cannot_recall", refs: [] }, { item: "C9", state: "done", reason: "chain_rules", refs: ["d1"] },
    { item: "CX", state: "strange" as CleanupItem["state"], reason: "x", refs: [] },
  ];
  const g = cleanupGroups(items);
  assert.deepEqual(g.done.map((i) => i.item), ["C1", "C9"]);
  assert.deepEqual(g.not_applicable.map((i) => i.item), ["C2"]);
  assert.deepEqual(g.outside_scope.map((i) => i.item), ["C5"]);
  assert.deepEqual(g.not_started.map((i) => i.item), ["CX"]);
  assert.equal(settled(items), 3);
  assert.deepEqual(itemWords("en", items[0]), { name: "Decision stamps", why: "Earlier decisions no longer pass a recheck" });
  assert.deepEqual(itemWords("en", items[4]), { name: "CX", why: "x" });
});

test("unconfirmed revocations mark their target and subject", () => {
  const p = page({
    grants: [grant], delegations: [deleg("d1", "")],
    revocations: [rev({ state: "decision_unconfirmed", job_state: "waiting" }), rev({ id: "revocation-2", target_kind: "delegation", target: "d1", state: "received" }), rev({ id: "revocation-3", target: "gx" })],
  });
  assert.ok(unconfirmedFor(p, "u"));
  assert.ok(unconfirmedFor(p, "d-d1"));
  assert.ok(!unconfirmedFor(p, "other"));
  assert.ok(!unconfirmedFor(page({ revocations: [rev({ target: "missing", state: "decision_unconfirmed" })] }), ""));
});

test("ledger lines name every state, the recovery and the replacement", () => {
  const when = (at: string) => "@" + at;
  assert.deepEqual(ledgerLines("en", ledgerOf({ state: "ok", records: 3, watermark: "3:ab" }), when), ["Decision ledger: 3 records, last record number 3."]);
  assert.deepEqual(ledgerLines("en", ledgerOf({ state: "ok", records: 1, watermark: "1:ab" }), when), ["Decision ledger: 1 record, last record number 1."]);
  for (const [n, want] of [[1, "قيد واحد"], [2, "قيدان"], [3, "3 قيود"], [11, "11 قيدًا"], [100, "100 قيد"]] as const) {
    assert.match(ledgerLines("ar", ledgerOf({ state: "ok", records: n, watermark: n + ":ab" }), when)[0], new RegExp(want));
  }
  const failed = ledgerLines("en", ledgerOf({ state: "isolated", isolation_reason: "integrity_failed", problem_line: 4 }), when)[0];
  assert.match(failed, /Revocations recorded before line 4 were re-applied at start; those from line 4 on were not/);
  assert.match(ledgerLines("en", ledgerOf({ state: "full" }), when)[0], /will not be re-applied after a restart/);
  assert.match(ledgerLines("en", ledgerOf({ state: "ok", replaces: "x" }), when)[1], /any sample access revoked under it is active again/);
  assert.match(ledgerLines("en", ledgerOf({ state: "ok" }), when)[0], /number 0/);
  assert.match(ledgerLines("en", ledgerOf({ state: "isolated", isolation_reason: "integrity_failed", problem_line: 12 }), when)[0], /integrity check at line 12/);
  assert.match(ledgerLines("en", ledgerOf({ state: "isolated", isolation_reason: "missing" }), when)[0], /ledger is missing/);
  assert.match(ledgerLines("en", ledgerOf({ state: "isolated", isolation_reason: "id_mismatch" }), when)[0], /not the one/);
  assert.match(ledgerLines("en", ledgerOf({ state: "stopped" }), when)[0], /write to the decision ledger failed/);
  assert.match(ledgerLines("en", ledgerOf({ state: "full" }), when)[0], /size bound/);
  assert.match(ledgerLines("en", ledgerOf(null), when)[0], /not reported/);
  const rec = ledgerLines("en", ledgerOf({ state: "full", recovery: { at: "t", truncated_bytes: 7 } }), when);
  assert.match(rec[1], /\(@t\).*7 bytes/);
  assert.match(rec[2], /Still available/);
  assert.match(ledgerLines("en", ledgerOf({ state: "ok", replaces: "unknown" }), when)[1], /\(unknown\).*active again/);
  for (const [l, text, sync, restart, operator] of [
    ["en", /came from records not confirmed durable/, /could not confirm that the decision ledger file is fully written to disk\. Nothing new was written to it/, /A restart checks the file again/, /An operator must restore/],
    ["ko", /확정 내구 기록으로 확인되지 않은 기록/, /디스크에 완전히 기록되었는지 확인하지 못했습니다\. 파일에 새로 쓴 내용은 없습니다/, /재시작하면 파일을 다시 확인합니다/, /운영자가 원장을 복구/],
    ["ar", /لم يتأكد أنها دائمة/, /مكتوب بالكامل على القرص\. لم يُكتب فيه أي شيء جديد/, /تتحقق إعادة التشغيل/, /يجب على المشغّل/],
  ] as const) {
    const stopped = ledgerLines(l, ledgerOf({ state: "stopped", not_durable: true }), when);
    assert.match(stopped[0], sync);
    assert.match(stopped[1], text);
    assert.match(stopped[1], restart);
    const isolated = ledgerLines(l, ledgerOf({ state: "isolated", isolation_reason: "integrity_failed", problem_line: 3, not_durable: true }), when);
    assert.match(isolated[1], text);
    assert.match(isolated[1], operator);
  }
  assert.match(ledgerLines("en", ledgerOf({ state: "stopped" }), when)[0], /a write to the decision ledger failed/);
  assert.ok(!ledgerLines("en", ledgerOf({ state: "stopped" }), when).some((x) => /not confirmed durable/.test(x)));
  assert.match(ledgerLines("ko", ledgerOf({ state: "ok", replaces: "abc" }), when)[1], /abc/);
  assert.ok(additionsPaused(ledgerOf({ state: "isolated" })) && !additionsPaused(ledgerOf({ state: "recovered" })) && additionsPaused(ledgerOf(null)));
});

test("settings read in words", () => {
  const set = { name: "access_request_hours", value: 24, unit: "hours", default: 168, min: 1, max: 720, source: "start_flag" };
  assert.equal(settingName("en", set), "Request lifetime");
  assert.equal(settingValue("en", set), "24 hours");
  assert.equal(settingValue("en", { ...set, value: 1 }), "1 hour");
  assert.equal(settingValue("ar", { ...set, value: 2 }), "ساعتان");
  assert.equal(settingValue("ar", { ...set, unit: "requests", value: 25 }), "25 طلبًا");
  assert.equal(settingSource("en", set), "Set when the server started");
  assert.equal(settingSource("ko", { ...set, source: "default" }), "설치 기본값");
  assert.equal(settingName("en", { ...set, name: "other" }), "other");
  assert.equal(settingValue("en", { ...set, unit: "things" }), "24 things");
});

test("the attempt is kept after a partial outcome only", () => {
  assert.ok(keepAttempt(new AdminError("partial", [], 0, "op-1")));
  assert.ok(!keepAttempt(new AdminError("stale")));
  assert.ok(!keepAttempt(new Error("x")));
});

test("descendants of a delegation", () => {
  const p = page({ delegations: [deleg("d1", ""), deleg("d2", "d1"), deleg("d3", "d2"), deleg("d4", "d1", "revoked"), deleg("d5", "")] });
  assert.deepEqual(descendantsOf(p, "d1"), ["d2", "d3"]);
  assert.deepEqual(descendantsOf(p, "d5"), []);
});

test("waiting counts and Arabic request wording", async () => {
  const { q, v } = await import("./request-copy");
  assert.equal(q.waiting("ar", 2), "طلبان بانتظار الاعتماد");
  assert.equal(q.waiting("en", 1), "1 waiting");
  assert.equal(q.forDays("ar", 2), "لمدة يومين من الاعتماد");
  assert.match(q.requestImpact("ar", "x", "عرض", 2), /مدة يومين من وقت الاعتماد/);
  assert.equal(q.noRequests("en"), "No access change requests are waiting.");
  assert.equal(v.request("en"), "Access change request");
  assert.equal(v.grantOf("ar", "W", "عرض"), "صلاحية عرض الممنوحة إلى W");
  assert.match(v.job.waiting("ar"), /^المهمة: في الانتظار/);
  assert.match(q.approveLead("ar"), /المحتوى وبصمة الطلب/);
  assert.match(q.revokeRequestLead("ar", "W"), /ويمكن أن يعتمده W أيضًا/);
});
