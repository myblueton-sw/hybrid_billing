import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, policyPageOf, type PolicyPage } from "../../../../service/admin";
import { duration, effectiveStep, methodsText, policyFailure, policyStep, stale, stepMinutes, stepText, wholeMinutes, when } from "./live-model";
import { c } from "../../../../service/copy";
import { p } from "./live-copy";
import { w } from "../users/live-copy";

const rev = { methods: ["password"], mfa: "none", max_age_seconds: 3600, step_up: [{ action: "policy.change", max_age_seconds: 600, mfa: false }] };
const page = (change: Partial<PolicyPage>): PolicyPage => ({
  ...policyPageOf({}), revision: "r1", now: "2026-11-16T09:00:00Z", signed_in_at: "2026-11-16T08:55:00Z", authenticated_at: "2026-11-16T08:55:00Z",
  act_until: "2026-11-16T09:05:00Z", step_up_seconds: 600, you: "b", ...change,
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
  assert.equal(stepText("en", { ...rev, step_up: [] }), "Default (15 minutes)");
  assert.match(stepText("ar", { ...rev, step_up: [] }), /^الافتراضي/);
  assert.equal(effectiveStep(page({})), 600);
  assert.equal(effectiveStep(page({ step_up_seconds: 0 })), 900);
  assert.equal(stepText("en", rev), "Within 10 minutes");
  assert.equal(methodsText("en", rev), p.passwordOnly("en"));
  assert.equal(methodsText("en", { ...rev, methods: ["password", "saml"] }), "password, saml");
});

test("staleness compares server times only", () => {
  assert.equal(stale(page({})), false);
  assert.equal(stale(page({ now: "2026-11-16T09:06:00Z" })), true);
  assert.equal(stale(page({ act_until: "" })), false);
});

test("the stale line exists in every language", () => {
  for (const l of ["en", "ko", "ar"] as const) assert.notEqual(p.staleLine(l), "");
});

test("blank or zero step minutes use the default", () => {
  assert.equal(stepMinutes("", 60), 0);
  assert.equal(stepMinutes(" 0 ", 60), 0);
  assert.equal(stepMinutes("10", 60), 10);
  assert.equal(stepMinutes("4", 60), null);
  assert.equal(stepMinutes("61", 60), null);
  assert.equal(stepMinutes("700", null), 700);
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
  for (const l of ["en", "ko", "ar"] as const) {
    assert.equal(policyFailure(l, new Error("x")).title, w.unknownTitle(l));
    assert.equal(policyFailure(l, new AdminError("unavailable")).title, w.unknownTitle(l));
    assert.equal(policyFailure(l, new AdminError("busy")).title, p.saveFailed(l));
  }
  for (const l of ["en", "ko", "ar"] as const) {
    assert.equal(policyFailure(l, new AdminError("reauth")).body, c.reauthRefused(l));
    assert.equal(policyFailure(l, new AdminError("reauth_cancelled")).body, c.reauthNotDone(l));
  }
});
