import { test } from "node:test";
import assert from "node:assert/strict";
import {
  fixtureCosts,
  fixtureClosings,
  selectCosts,
  selectClosings,
  total,
  formatMoney,
  sources,
  organizations,
} from "./model";
test("independent monthly controls preserve currency and partial-source coverage", () => {
  const all = fixtureCosts("sample-a");
  const may = selectCosts(all, [...sources], [...organizations], "2026-05");
  assert.deepEqual(total(may, "USD"), {
    value: 1647000n,
    known: 9,
    expected: 9,
    missing: 0,
  });
  assert.equal(total(may, "KRW").value, 6100000n);
  const october = selectCosts(all, [...sources], [...organizations], "2026-10");
  assert.deepEqual(total(october, "USD"), {
    value: 1784000n,
    known: 8,
    expected: 9,
    missing: 1,
  });
  assert.equal(total(october, "KRW").value, 6900000n);
  assert.equal(
    total(
      selectCosts(
        fixtureCosts("sample-b"),
        [...sources],
        [...organizations],
        "2026-05",
      ),
      "USD",
    ).value,
    3294000n,
  );
});
test("empty selection, fully missing and confirmed zero are different", () => {
  const all = fixtureCosts("sample-a");
  assert.equal(selectCosts(all, [], [...organizations]).length, 0);
  assert.equal(selectCosts(all, [...sources], []).length, 0);
  assert.deepEqual(total(selectCosts(all, ["GCP"], ["B"], "2026-10"), "USD"), {
    value: null,
    known: 0,
    expected: 1,
    missing: 1,
  });
  assert.deepEqual(
    total(selectCosts(all, ["On-prem"], ["C"], "2026-10"), "KRW"),
    { value: 0n, known: 1, expected: 1, missing: 0 },
  );
  assert.equal(
    total(selectCosts(all, ["AWS"], ["A"], "2025-01"), "USD").expected,
    0,
  );
});
test("closing units are deduplicated and keep their full-source semantics", () => {
  const rows = fixtureClosings("sample-a", "2026-10");
  const a = selectClosings(rows, ["AWS"], ["A"]);
  const ab = selectClosings(rows, ["AWS", "Azure"], ["A"]);
  assert.equal(a.length, 1);
  assert.deepEqual(a, ab);
  assert.deepEqual(a[0].sources, ["AWS", "Azure", "GCP"]);
  const internal = selectClosings(rows, ["On-prem"], ["A"])[0];
  assert.equal(internal.status.issuance, "na");
  assert.equal(internal.status.receipt, "na");
  assert.equal(selectClosings(rows, [], ["A"]).length, 0);
  assert.equal(fixtureClosings("sample-a", "2025-01").length, 0);
});
test("aggregation and formatting do not lose large exact minor units", () => {
  const r = fixtureCosts("sample-a")[0];
  assert.equal(
    total(
      [
        { ...r, minor: 9007199254740993n },
        { ...r, minor: 7n },
      ],
      "USD",
    ).value,
    9007199254741000n,
  );
  assert.equal(
    formatMoney(9007199254740993n, "USD", "en"),
    "USD 90,071,992,547,409.93",
  );
  assert.equal(formatMoney(0n, "USD", "en"), "USD 0.00");
  assert.equal(formatMoney(null, "USD", "en"), "—");
  assert.equal(formatMoney(-1n, "USD", "en"), "USD -0.01");
  assert.equal(formatMoney(-12345n, "USD", "en"), "USD -123.45");
});
