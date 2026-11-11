import { test } from "node:test";
import assert from "node:assert/strict";
import type { FocusRow } from "../../../sources/collected-focus/api";
import {
  accountToken,
  amountText,
  deliveryCount,
  identityText,
  optionText,
  shortAccount,
} from "./identity";
const plain = (value: string) => value.replace(/[\u2066\u2069]/g, "");
const row = (values: Partial<FocusRow>) =>
  ({
    id: "2198c573".padEnd(64, "a"),
    billing_account: "synthetic-vmware",
    sub_account: "synthetic-a",
    ordinal: "1000",
    batch: "delivery-1",
    currency: "USD",
    billed_cost: "750000000000000000",
    effective_cost: "0",
    ...values,
  }) as FocusRow;
test("identity names the subaccount, ordinal and short row ID", () => {
  assert.equal(
    plain(identityText("en", row({}))),
    "Ordinal 1000 · Subaccount synthetic-a · Row ID 2198c573",
  );
  assert.equal(
    plain(identityText("ko", row({}))),
    "순번 1000 · 하위 계정 synthetic-a · 행 ID 2198c573",
  );
});
test("a missing subaccount falls back to a labeled billing account", () => {
  const missing = row({ sub_account: null });
  assert.deepEqual(accountToken(missing), {
    kind: "billing_account",
    value: "synthetic-vmware",
  });
  assert.equal(
    plain(identityText("en", missing)),
    "Ordinal 1000 · Billing account synthetic-vmware · Row ID 2198c573",
  );
});
test("long account text is shortened deterministically", () => {
  const long = "ocid1.tenancy.oc1..aaaaaaaabbbbbbbbccccccccdddddddd";
  assert.equal(shortAccount(long), "ocid1.tenanc…ccccdddddddd");
  assert.equal(shortAccount("123456789012"), "123456789012");
  assert.equal(shortAccount("a".repeat(28)), "a".repeat(28));
});
test("left-to-right tokens are isolated for right-to-left text", () => {
  const text = identityText("ar", row({}));
  assert.ok(text.includes("\u20661000\u2069"));
  assert.ok(text.includes("\u2066synthetic-a\u2069"));
  assert.ok(text.includes("\u20662198c573\u2069"));
  assert.ok(
    amountText("ar", "billed_cost", row({})).includes("\u2066USD 0.75\u2069"),
  );
});
test("amount text keeps the basis label and separates value states", () => {
  assert.equal(
    plain(amountText("en", "billed_cost", row({}))),
    "Billed cost: USD 0.75",
  );
  assert.equal(
    plain(amountText("en", "effective_cost", row({}))),
    "Effective cost: USD 0.00 (Exact zero)",
  );
  assert.equal(
    amountText("en", "billed_cost", row({ billed_cost: null })),
    "Billed cost: Missing",
  );
  assert.equal(
    amountText("ko", "billed_cost", row({ billed_cost: null })),
    "청구 비용: 누락",
  );
  assert.equal(
    plain(
      amountText(
        "en",
        "billed_cost",
        row({ billed_cost: "-2613700000000000000" }),
      ),
    ),
    "Billed cost: USD -2.6137",
  );
  assert.equal(
    plain(
      amountText(
        "en",
        "billed_cost",
        row({ billed_cost: "1000000000000000001", currency: "CNY" }),
      ),
    ),
    "Billed cost: CNY 1.000000000000000001",
  );
});
test("a missing amount never carries a currency, dash or zero", () => {
  for (const locale of ["en", "ko", "ar"] as const) {
    const text = amountText(locale, "billed_cost", row({ billed_cost: null }));
    assert.ok(!/USD|—|0/.test(text));
  }
});
test("option text joins identity and the selected basis amount", () => {
  assert.equal(
    plain(optionText("en", "billed_cost", row({}))),
    "Ordinal 1000 · Subaccount synthetic-a · Row ID 2198c573 · Billed cost: USD 0.75",
  );
});
test("rows sharing an ordinal across deliveries stay distinct by row ID", () => {
  const first = row({}),
    second = row({ id: "9872ca7b".padEnd(64, "b"), batch: "delivery-2" });
  assert.equal(first.ordinal, second.ordinal);
  assert.notEqual(identityText("en", first), identityText("en", second));
  assert.equal(deliveryCount([first, second]), 2);
  assert.equal(deliveryCount([first, row({})]), 1);
});
