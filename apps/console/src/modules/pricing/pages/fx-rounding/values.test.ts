import { test } from "node:test";
import assert from "node:assert/strict";
import { exact, subtract, asRational, money } from "./values";
import { scale } from "./model";
test("exact monetary display preserves terminating and recurring fractions", () => {
  assert.deepEqual(
    exact({ numerator: (103n * scale).toString(), denominator: "100" }, "KRW"),
    { text: "KRW 1.03", fraction: false },
  );
  assert.deepEqual(
    exact(
      { numerator: (-1025n * scale).toString(), denominator: "1000" },
      "KRW",
    ),
    { text: "−KRW 1.025", fraction: false },
  );
  assert.deepEqual(exact({ numerator: "1", denominator: "3" }, "KRW"), {
    text: "KRW 1/3000000000000000000",
    fraction: true,
  });
  assert.deepEqual(
    exact({ numerator: scale.toString(), denominator: "3" }, "KRW/USD"),
    { text: "KRW/USD 1/3", fraction: true },
  );
  assert.deepEqual(exact({ numerator: "0", denominator: "17" }, "KRW", true), {
    text: "KRW 0",
    fraction: false,
  });
  assert.equal(
    exact({ numerator: scale.toString(), denominator: "10" }, "KRW", true).text,
    "+KRW 0.1",
  );
  assert.equal(
    exact(
      { numerator: scale.toString(), denominator: (2n ** 90n).toString() },
      "KRW",
    ).fraction,
    false,
  );
});
test("comparison rational difference and signed formatted total keep meaning", () => {
  const d = subtract(
    { numerator: scale.toString(), denominator: "3" },
    { numerator: scale.toString(), denominator: "6" },
  );
  assert.deepEqual(exact(d, "KRW", true), { text: "+KRW 1/6", fraction: true });
  assert.deepEqual(exact(asRational((-scale).toString()), "KRW"), {
    text: "−KRW 1",
    fraction: false,
  });
  assert.equal(money("7400000000000000000", "KRW", "en"), "KRW 7.40");
});
test("display rejects nonpositive denominators and separate display growth bounds", () => {
  for (const denominator of ["0", "-1", (10n ** 90n).toString()])
    assert.throws(
      () => exact({ numerator: "1", denominator }, "KRW"),
      /display_unavailable/,
    );
  assert.throws(
    () =>
      exact({ numerator: (10n ** 130n).toString(), denominator: "1" }, "KRW"),
    /display_unavailable/,
  );
  assert.throws(
    () =>
      subtract(
        { numerator: (10n ** 130n - 1n).toString(), denominator: "1" },
        { numerator: (-(10n ** 130n) + 1n).toString(), denominator: "1" },
      ),
    /display_unavailable/,
  );
  const a = {
      numerator: (10n ** 83n).toString(),
      denominator: (10n ** 43n).toString(),
    },
    b = { ...a, numerator: (-(10n ** 83n)).toString() };
  const delta = subtract(a, b);
  assert.equal(exact(delta, "KRW").text, "KRW 20000000000000000000000");
});
