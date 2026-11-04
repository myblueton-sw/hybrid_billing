import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import FocusChart from "./FocusChart";
import type { OverviewData } from "../../../sources/collected-focus/api";
import React from "react";
test("rendered chart distinguishes partial zero, partial known, all-missing, absent and minimum coefficients", () => {
  const monthly = [
    { period: "2024-01", coefficient: "0", present: 1, missing: 1 },
    { period: "2024-02", coefficient: "1", present: 1, missing: 1 },
    { period: "2024-03", coefficient: null, present: 0, missing: 2 },
    { period: "2024-05", coefficient: "-1", present: 1, missing: 0 },
  ];
  const data = {
    selection: { period: "all", basis: "billed_cost" },
    options: { currencies: ["USD"], periods: monthly.map((m) => m.period) },
    months: monthly.map((m) => ({
      period: m.period,
      amounts: [{ currency: "USD", billed_cost: m }],
    })),
  } as unknown as OverviewData;
  const html = renderToStaticMarkup(
    <FocusChart
      data={data}
      currency="USD"
      locale="en"
      onCurrency={() => {}}
      onMonth={() => {}}
    />,
  );
  assert.equal((html.match(/data-value-state="partial"/g) ?? []).length, 2);
  assert.equal((html.match(/data-value-state="unknown"/g) ?? []).length, 1);
  assert.equal((html.match(/data-value-state="absent"/g) ?? []).length, 1);
  assert.equal((html.match(/data-value-state="complete"/g) ?? []).length, 1);
  assert.ok(html.includes("Unknown · all values missing"));
  assert.ok(html.includes("No collected value"));
  assert.ok(html.includes("-0.000000000000000001"));
  assert.ok(html.includes("height:2px"));
});
