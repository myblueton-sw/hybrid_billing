import { test } from "node:test";
import assert from "node:assert/strict";
import { validPage, rowOrdinal, ordinal, type Page } from "./api";
const release = "00000000-0000-0000-0000-000000000001",
  snapshot = "a".repeat(64);
const values = {
  billed_cost: "999999999999.999999",
  effective_cost: null,
  billing_currency: "USD",
  billing_account_id: "synthetic",
  charge_period_start: "2026-10-01T00:00:00Z",
  charge_period_end: "2026-10-02T00:00:00Z",
};
const page = {
  candidate: {
    release_id: release,
    manifest_digest: snapshot,
    row_count: 1,
    source_revision: "1",
    version: "1.2",
    state: "verified",
    created_at: "2026-11-03",
    blockers: [],
  },
  snapshot,
  after: "0",
  next_after: null,
  rows: [{ ordinal: "1", values }],
  fields: Object.keys(values),
  classification: "local-source-inspection",
  content_integrity: "not-revalidated",
  observed_at: "2026-11-03",
} as Page;
test("exact nullable decimals and identity remain bound", () => {
  assert.ok(validPage(page, release, snapshot, "0"));
  assert.equal(validPage(page, release, "b".repeat(64), "0"), false);
  assert.equal(
    validPage(
      {
        ...page,
        rows: [{ ordinal: "1", values: { ...values, billed_cost: "1e12" } }],
      },
      release,
      snapshot,
      "0",
    ),
    false,
  );
  assert.equal(
    validPage({ ...page, next_after: "50" }, release, snapshot, "0"),
    false,
  );
  assert.equal(
    validPage(
      { ...page, fields: ["tags", "effective_cost"] },
      release,
      snapshot,
      "0",
    ),
    false,
  );
  assert.equal(
    validPage(
      {
        ...page,
        candidate: { ...page.candidate, blockers: undefined },
      } as unknown as Page,
      release,
      snapshot,
      "0",
    ),
    false,
  );
});
test("row 10000 is navigable but is not a continuation cursor", () => {
  assert.ok(rowOrdinal.test("10000"));
  assert.ok(!ordinal.test("10000"));
  assert.ok(!rowOrdinal.test("0"));
  assert.ok(!rowOrdinal.test("10001"));
  const last = {
    ...page,
    candidate: { ...page.candidate, row_count: 10000 },
    after: "9950",
    rows: Array.from({ length: 50 }, (_, i) => ({
      ...page.rows[0],
      ordinal: String(9951 + i),
    })),
  };
  assert.ok(validPage(last, release, snapshot, "9950"));
});
