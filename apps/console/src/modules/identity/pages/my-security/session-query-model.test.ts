import { test } from "node:test";
import assert from "node:assert/strict";
import { parseSessionLink, sessionLink, sessionRequest, canStartTermination, submittedFingerprint } from "./session-query-model";

test("session navigation preserves exact timestamp text and rejects duplicate or oversized scope", () => {
  const since = "2026-11-20T01:02:03.123456789Z";
  const parsed = parseSessionLink(new URLSearchParams({ session: "other", sq: "abcdef", ssince: since, slimit: "17", safter: "cursor" }));
  assert.equal(parsed.bad, false);
  assert.deepEqual(sessionRequest(parsed), { session: "other", q: "abcdef", since, limit: 17, after: "cursor" });
  assert.equal(sessionLink(parsed.filters, parsed.after).ssince, since);
  for (const raw of ["session=other&session=current", "slimit=0", "slimit=101", "slimit=1.5", "sq="+"가".repeat(22), "safter="+"x".repeat(4097)]) assert.equal(parseSessionLink(new URLSearchParams(raw)).bad, true);
});

test("a changed query drops its cursor and uses installer page default when not specified", () => {
  const parsed = parseSessionLink(new URLSearchParams());
  assert.deepEqual(sessionRequest(parsed), { session: "all" });
  assert.equal(sessionLink({ ...parsed.filters, q: "new" }).safter, null);
});

test("uncertain termination retains the identical fingerprint and blocks new actions", () => {
  const fingerprint = { selection: "selection-a", digest: "digest", count: 170 };
  assert.deepEqual(submittedFingerprint({ state: "unknown", fingerprint }), fingerprint);
  for (const state of ["submitting", "unknown", "recovering"] as const) assert.equal(canStartTermination({ state, fingerprint }), false);
  assert.equal(canStartTermination({ state: "idle" }), true);
  assert.equal(canStartTermination({ state: "unavailable", fingerprint }), false);
});

test("session search accepts hexadecimal identifiers and rejects actual Unicode controls", () => {
  for (const value of ["abcdef0123456789", "A19", "세션", "جلسة"]) {
    const parsed = parseSessionLink(new URLSearchParams({ sq: value }));
    assert.equal(parsed.bad, false, value);
    assert.equal(parsed.filters.q, value);
  }
  for (const value of ["a"+String.fromCodePoint(0)+"b", "a"+String.fromCodePoint(0x85)+"b", String.fromCodePoint(0x7f)]) {
    assert.equal(parseSessionLink(new URLSearchParams({ sq: value })).bad, true);
  }
});
