import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, type PolicyPage, type ProposalRow } from "../../../../service/admin";
import { canAct, duration, history, methodsText, pending, policyFailure, policyStep, stale, stepText, wholeMinutes, when } from "./live-model";
import { p } from "./live-copy";

const rev = { methods: ["password"], mfa: "none", max_age_seconds: 3600, step_up: [{ action: "policy.change", max_age_seconds: 600, mfa: false }] };
const row = (change: Partial<ProposalRow>): ProposalRow => ({
  number: 2, target: "proposal-2", state: "pending", proposer: "a", approver: "", reason: "r", revision: rev,
  can_approve: true, can_withdraw: false, act_until: "2026-11-16T09:10:00Z", ...change,
});
const page = (change: Partial<PolicyPage>): PolicyPage => ({
  revision: "r1", policy: null, proposals: [], now: "2026-11-16T09:00:00Z", signed_in_at: "2026-11-16T08:55:00Z", act_until: "2026-11-16T09:05:00Z", you: "b", ...change,
});

test("durations name minutes and whole hours", () => {
  assert.equal(duration("en", 300), "5 minutes");
  assert.equal(duration("en", 3600), "60 minutes (1 hour)");
  assert.match(duration("ko", 43200), /720/);
  assert.notEqual(duration("ar", 600), "");
});

test("policy step and texts", () => {
  assert.equal(policyStep(rev), 600);
  assert.equal(policyStep({ ...rev, step_up: [{ action: "other", max_age_seconds: 60, mfa: false }] }), 0);
  assert.equal(stepText("en", { ...rev, step_up: [] }), p.stepNone("en"));
  assert.equal(stepText("ar", { ...rev, step_up: [] }), "غير مطلوب");
  assert.equal(stepText("en", rev), "Within 10 minutes");
  assert.equal(methodsText("en", rev), p.passwordOnly("en"));
  assert.equal(methodsText("en", { ...rev, methods: ["password", "saml"] }), "password, saml");
});

test("staleness compares server times only", () => {
  assert.equal(stale(page({})), false);
  assert.equal(stale(page({ now: "2026-11-16T09:06:00Z" })), true);
  assert.equal(stale(page({ act_until: "" })), false);
});

test("actions follow the server flags and windows", () => {
  assert.deepEqual(canAct(page({}), row({})), { approve: true, withdraw: false, note: "" });
  assert.deepEqual(canAct(page({ now: "2026-11-16T09:11:00Z" }), row({})), { approve: false, withdraw: false, note: "staleRow" });
  assert.deepEqual(canAct(page({}), row({ can_approve: false, can_withdraw: true })), { approve: false, withdraw: true, note: "needOther" });
  assert.deepEqual(canAct(page({ now: "2026-11-16T09:06:00Z" }), row({ can_approve: false, can_withdraw: true })), { approve: false, withdraw: false, note: "needOther" });
  assert.deepEqual(canAct(page({}), row({ can_approve: false })), { approve: false, withdraw: false, note: "cannotApprove" });
});

test("pending and history split", () => {
  const all = page({ proposals: [row({}), row({ number: 1, state: "applied" })] });
  assert.equal(pending(all).length, 1);
  assert.equal(history(all)[0].number, 1);
});

test("whole minutes are strict", () => {
  assert.equal(wholeMinutes("60", 5, 720), 60);
  for (const raw of ["", "0", "4", "721", "007", "1.5", "-5", "12345", "6e1"]) assert.equal(wholeMinutes(raw, 5, 720), null, raw);
});

test("dates and failures", () => {
  assert.equal(when("en", ""), "");
  assert.equal(when("en", "nope"), "");
  assert.notEqual(when("ar", "2026-11-16T09:00:00Z"), "");
  const kinds = ["refused", "stage", "conflict", "busy", "denied", "unauthenticated", "limited", "partial", "unavailable"] as const;
  const bodies = kinds.map((k) => policyFailure("en", new AdminError(k, [], 2, "op-3")).body);
  assert.equal(new Set(bodies).size, kinds.length);
  assert.equal(policyFailure("en", new AdminError("stale")).body, policyFailure("en", new AdminError("conflict")).body);
  assert.equal(policyFailure("en", new AdminError("invalid")).body, "");
  assert.equal(policyFailure("en", new Error("x")).title, p.saveFailed("en"));
});
