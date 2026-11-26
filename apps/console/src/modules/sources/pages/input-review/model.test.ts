import test from "node:test";
import assert from "node:assert/strict";
import { AdminError, type Outcome } from "../../../../service/admin";
import { allowed, reasonValid, reviewOf, uncertain } from "./model";

const sample = (): Outcome => ({ status: "completed", state: "", job_id: "", revision: "42", fields: {
  mode: "synthetic", review: "review", digest: "a".repeat(64), workspace: "ws", provider: "provider", account: "account", period: "opaque",
  method: "comparison", recorded_verdict: "match", compared_at: "2026-11-20T01:02:03.123456789Z", document: "document", document_revision: "1",
  document_currency: "USD", document_total: "15.0000", document_digest: "b".repeat(64), recorded_state: "awaiting_confirmation",
  current_evidence: "matched", current_eligibility: "not_eligible", confirmation_permission: "allowed", confirmation_hold: "clear", scope_revision: "0",
} });
test("exact amounts and opaque period survive parsing without financial conversion", () => {
  for (const amount of ["15", "15.0000", "-0.0001", "9007199254740993.0100"]) {
    const input = sample(); input.fields.document_total = amount;
    assert.equal(reviewOf(input, "review", "ws").fields.document_total, amount);
  }
});
test("unknown, malformed and contradictory evidence never enables a decision", () => {
  const changes = { recorded_verdict: "future", digest: "bad", document_digest: "", compared_at: "yesterday", scope_revision: "4294967296", document_revision: "01", document_total: "NaN", confirmation_hold: "future", confirmation_permission: "future", current_evidence: "future", current_eligibility: "eligible", recorded_state: "blocked" };
  for (const [key, value] of Object.entries(changes)) {
    const input = sample(); input.fields[key] = value;
    assert.throws(() => reviewOf(input, "review", "ws"), AdminError, key);
  }
  assert.throws(() => reviewOf(sample(), "other", "ws"), AdminError);
});
test("current permission and hold independently gate the two decision actions", () => {
  const input = sample(), review = reviewOf(input, "review", "ws");
  assert.ok(allowed(review, "confirm_publish")); assert.ok(allowed(review, "reject"));
  review.fields.confirmation_hold = "blocked";
  assert.equal(allowed(review, "confirm_publish"), false); assert.ok(allowed(review, "reject"));
  review.fields.confirmation_permission = "denied";
  assert.equal(allowed(review, "reject"), false);
  review.fields.confirmation_permission = "allowed"; review.fields.current_evidence = "changed";
  assert.equal(allowed(review, "reject"), false);
});
test("rationale validation uses actual UTF8 bytes and preserves whitespace", () => {
  assert.ok(reasonValid("  reviewed  ")); assert.ok(reasonValid("가".repeat(341)));
  for (const reason of [" ", "가".repeat(342), "x\ny", "x\u007fy", "x".repeat(1025)]) assert.equal(reasonValid(reason), false);
});
test("a later refusal never proves an older unknown request did not execute", () => {
  const denied = new AdminError("denied", [], 0, "", "permission_denied");
  assert.equal(uncertain(denied, false), false); assert.equal(uncertain(denied, true), true);
  assert.ok(uncertain(new AdminError("unavailable"), false));
  assert.ok(uncertain(new AdminError("partial", [], 0, "op-id", "internal_error"), false));
  assert.ok(uncertain(new AdminError("stale", [], 0, "", "idempotency_key_expired"), false));
  assert.equal(uncertain(new AdminError("busy", [], 0, "", "dependency_unavailable"), false), false);
});

test("inspection is bound to the session workspace and exact producer grammar", () => {
  assert.throws(() => reviewOf(sample(), "review", "another-workspace"));
  for (const [key, value] of [["compared_at", "2026-02-30T00:00:00Z"], ["compared_at", "0000-01-01T00:00:00Z"], ["document_revision", "0"], ["document_revision", "1000000"], ["document_total", "015.00"], ["decided_by", "unexpected-decision"]]) {
    const input = sample(); input.fields[key] = value;
    assert.throws(() => reviewOf(input, "review", "ws"), AdminError, key + value);
  }
});
