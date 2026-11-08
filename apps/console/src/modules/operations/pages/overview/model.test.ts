import { filterClosings } from "./closing-filter";
import {
  encodeOrganizations,
  resolveOrganizations,
  enterpriseLeaves,
} from "./organization-scope";
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

test("enterprise hierarchy unions leaves without broadening invalid or empty selections", () => {
  assert.equal(resolveOrganizations(true, null).length, 2400);
  assert.equal(resolveOrganizations(true, "C01").length, 200);
  assert.equal(
    resolveOrganizations(true, "C01|C01.B01|C01.B01.D01").length,
    200,
  );
  assert.equal(resolveOrganizations(true, "C01|C02").length, 400);
  assert.equal(resolveOrganizations(true, "C01|unknown").length, 200);
  assert.deepEqual(resolveOrganizations(true, ""), []);
  assert.deepEqual(resolveOrganizations(true, "unknown"), []);
  assert.deepEqual(resolveOrganizations(false, "C01"), []);
  const rows = fixtureCosts("sample-a", enterpriseLeaves);
  const overlap = selectCosts(
    rows,
    ["AWS"],
    resolveOrganizations(true, "C01|C01.B01"),
    "2026-05",
  );
  assert.equal(overlap.length, 200);
  assert.equal(total(overlap, "USD").value, 481700000n);
});

test("enterprise compact selections preserve arbitrary subsets and reject corrupt encoding", () => {
  const ids = enterpriseLeaves.filter((_, i) => i % 3 === 0);
  assert.equal(encodeOrganizations(ids).length, 603);
  assert.deepEqual(resolveOrganizations(true, encodeOrganizations(ids)), ids);
  assert.deepEqual(resolveOrganizations(true, "v1:00"), []);
  assert.equal(encodeOrganizations(enterpriseLeaves), "all");
  assert.equal(encodeOrganizations([]), "");
});

test("closing list filters match independent stages without changing source units", () => {
  const rows = fixtureClosings("sample-a", "2026-10");
  assert.equal(
    filterClosings(rows, { query: "", stage: "accounting", status: "unknown" })
      .length,
    2,
  );
  assert.equal(
    filterClosings(rows, { query: "", stage: "", status: "pending" }).length,
    6,
  );
  assert.equal(
    filterClosings(rows, {
      query: " B ",
      stage: "reconciliation",
      status: "blocked",
    }).length,
    1,
  );
  assert.equal(
    filterClosings(rows, { query: "", stage: "receipt", status: "na" }).length,
    3,
  );
  assert.equal(
    filterClosings(rows, { query: "not-found", stage: "", status: "" }).length,
    0,
  );
  assert.equal(rows.length, 6);
});
