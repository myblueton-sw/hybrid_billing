import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError, type SelfPage } from "../../../../service/admin";
import { canChange, changeOutcome, checkChange, endOutcome, others } from "./live-model";

const page = (change: Partial<SelfPage>): SelfPage => ({
  user_id: "u", login: "l", email: "e", workspace: "w", role: "member", identities: [], password_set: true, installation_administrator: false,
  sessions: [{ handle: "a", current: true, signed_in: "1", ends: "2" }, { handle: "b", current: false, signed_in: "0", ends: "1" }],
  idle_minutes: 30, max_age_seconds: 0, signed_in_at: "1", now: "1", ...change,
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

test("ending a session that is gone reads as already ended", () => {
  assert.equal(endOutcome(new AdminError("invalid")), "gone");
  assert.equal(endOutcome(new AdminError("unauthenticated")), "signedOut");
  assert.equal(endOutcome(new AdminError("limited")), "limited");
  assert.equal(endOutcome(new AdminError("busy")), "busy");
  assert.equal(endOutcome(null), "busy");
});
