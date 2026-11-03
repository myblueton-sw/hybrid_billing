import test from "node:test";
import assert from "node:assert/strict";
import { chartScale, chartPercent } from "./chart-scale";

test("USD cost axis has readable ticks and headroom without rounding the amount", () => {
  const amount = 1971000n;
  const scale = chartScale(amount);
  assert.deepEqual(scale, {
    ceiling: 2200000n,
    ticks: [2000000n, 1000000n, 0n],
  });
  assert.equal(amount, 1971000n);
  assert.equal(chartPercent(amount, scale.ceiling), 89.59);
  assert.equal(chartPercent(0n, scale.ceiling), 0);
});
test("axis remains finite for confirmed zero and sub-unit amounts", () => {
  for (const maximum of [0n, 1n, 2n, 99n]) {
    const scale = chartScale(maximum);
    assert(scale.ceiling > maximum);
    assert(scale.ticks[0] >= maximum);
    assert.equal(scale.ticks[2], 0n);
    assert(Number.isFinite(chartPercent(maximum, scale.ceiling)));
  }
});
test("large exact amounts never pass through binary money arithmetic", () => {
  const maximum = 9007199254740993123456789n;
  const scale = chartScale(maximum);
  assert(scale.ceiling > maximum);
  assert(scale.ticks[0] >= maximum);
  assert.equal(scale.ticks[1] * 2n, scale.ticks[0]);
  assert(chartPercent(maximum, scale.ceiling) > 0);
  assert(chartPercent(maximum, scale.ceiling) < 100);
});
