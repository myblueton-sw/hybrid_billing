import { test } from "node:test";
import assert from "node:assert/strict";
import { decimal, chartSeries, signedScale, scale } from "./focus-values";
import type { OverviewData } from "../../../sources/collected-focus/api";
test("exact18 digit values, huge integers, zero and signed credits", () => {
  assert.equal(decimal("1000000000000000001", "en"), "1.000000000000000001");
  assert.equal(decimal("-1", "ko"), "-0.000000000000000001");
  assert.equal(decimal("0", "en"), "0.00");
  assert.equal(decimal(null, "en"), "—");
  assert.equal(
    decimal("99999999999999999999999999999999999999", "en"),
    "99,999,999,999,999,999,999.999999999999999999",
  );
});
test("signed chart preserves credit and cancellation geometry", () => {
  assert.deepEqual(signedScale(["1"]).ticks, [1n, 0n]);
  assert.deepEqual(signedScale(["-1"]).ticks, [0n, -1n]);
  const a = signedScale(["-100", "-200", null]);
  assert.equal(a.zero, 0);
  assert.ok(a.y(-200n) > a.y(-100n));
  const mixed = signedScale(["100", "-100"]);
  assert.equal(mixed.zero, 50);
  assert.ok(mixed.y(-100n) > 50 && mixed.y(100n) < 50);
  assert.equal(signedScale(["0"]).zero, signedScale([null]).zero);
  assert.ok(Number.isFinite(signedScale([null]).y(0n)));
  assert.ok(signedScale([(scale ** 2n).toString()]).y(scale ** 2n) >= 0);
});
test("calendar gaps remain unknown instead of invented zeros", () => {
  const measure = { coefficient: "0", present: 1, missing: 0 };
  const data = {
    selection: { period: "all", basis: "billed_cost" },
    months: [
      {
        period: "2024-03",
        amounts: [{ currency: "USD", billed_cost: measure }],
      },
      {
        period: "2024-09",
        amounts: [{ currency: "USD", billed_cost: measure }],
      },
    ],
  } as OverviewData;
  const values = chartSeries(data, "USD");
  assert.equal(values.length, 7);
  assert.equal(values[0].measure?.coefficient, "0");
  assert.equal(values[1].period, "2024-04");
  assert.equal(values[1].measure, undefined);
});
