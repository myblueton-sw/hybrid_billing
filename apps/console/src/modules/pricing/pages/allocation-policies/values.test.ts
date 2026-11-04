import { test } from "node:test";
import assert from "node:assert/strict";
import { reviewMoney } from "./values";
test("comparison signs use a consistent prefix without losing source precision", () => {
  for (const locale of ["en", "ko", "ar"] as const) {
    assert.equal(
      reviewMoney("-625000000000000000", "USD", locale),
      "−USD 0.625",
    );
    assert.equal(
      reviewMoney("625000000000000000", "USD", locale, true),
      "+USD 0.625",
    );
    assert.equal(
      reviewMoney("-1", "USD", locale, true),
      "−USD 0.000000000000000001",
    );
    assert.equal(reviewMoney("0", "USD", locale, true), "USD 0.00");
  }
});
