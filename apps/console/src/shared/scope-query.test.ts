import test from "node:test";
import assert from "node:assert/strict";
import { canonicalScope } from "./scope-query";
test("legacy scope belongs to its screen and canonical presence wins", () => {
  const q = new URLSearchParams(
    "overview-period=2026-08&close-period=2026-09&ov-currency=KRW",
  );
  assert.equal(canonicalScope(q, "U01").get("period"), "2026-08");
  assert.equal(canonicalScope(q, "U02").get("period"), "2026-09");
  assert.equal(canonicalScope(q, "S04").has("period"), false);
  assert.equal(canonicalScope(q, "U02").has("ccy"), false);
  q.set("period", "");
  q.set("ccy", "invalid");
  assert.equal(canonicalScope(q, "U01").get("period"), "");
  assert.equal(canonicalScope(q, "U01").get("ccy"), "invalid");
  assert.equal(canonicalScope(q, "U01").has("overview-period"), false);
  assert.equal(canonicalScope(q, "U01").has("ov-currency"), false);
});
test("scope normalization preserves unrelated contextual parameters", () => {
  const q = canonicalScope(
    new URLSearchParams(
      "overview-period=2026-08&ov-currency=KRW&scope=sample-b&le=ORG-2&asof=x&ov-organizations=leaf-3",
    ),
    "U01",
  );
  assert.equal(q.get("period"), "2026-08");
  assert.equal(q.get("ccy"), "KRW");
  assert.equal(q.get("scope"), "sample-b");
  assert.equal(q.get("le"), "ORG-2");
  assert.equal(q.get("asof"), "x");
  assert.equal(q.get("ov-organizations"), "leaf-3");
});
