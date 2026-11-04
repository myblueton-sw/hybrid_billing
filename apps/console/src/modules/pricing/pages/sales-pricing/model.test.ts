import { test } from "node:test";
import assert from "node:assert/strict";
import type {
  FocusRow,
  OverviewData,
} from "../../../sources/collected-focus/api";
import {
  prepare,
  emptyDraft,
  decimal,
  percentage,
  simulate,
  adjust,
  comparable,
  issues,
  scale,
  bound,
  type Draft,
} from "./model";
import { save, restore, storageKey } from "./store";
const id = (n: number) => n.toString(16).padStart(64, "0");
function data(q: string | null = "3000000000000000000"): OverviewData {
  return {
    company: "sample",
    source_snapshot: id(10),
    snapshot: id(11),
    selection: {
      period: "2024-09",
      provider: "VMware",
      account: "",
      q: "",
      basis: "billed_cost",
    },
    after: 0,
    account_after: 0,
    rows: [
      {
        id: id(1),
        currency: "USD",
        pricing_unit: "Hours",
        pricing_quantity: q,
        charge_month: "2024-09",
      } as FocusRow,
      {
        id: id(2),
        currency: "USD",
        pricing_unit: "Hours",
        pricing_quantity: q,
        charge_month: "2024-09",
      } as FocusRow,
    ],
  } as OverviewData;
}
const draft = (): Draft => ({
  ...emptyDraft(),
  name: "Synthetic policy",
  unitPrice: "2",
  denominator: "1",
  discount: "10",
  fee: "0.1",
});
function memory() {
  const map = new Map<string, string>();
  return {
    getItem: (k: string) => map.get(k) ?? null,
    setItem: (k: string, v: string) => {
      map.set(k, v);
    },
  };
}
test("finite exact manual parameters reject inferred defaults and invalid scale/sign", () => {
  for (const s of [
    "",
    " 1",
    "01",
    "1e2",
    "-1",
    "NaN",
    "1.0000000000000000000",
    "100000000000000000000",
  ])
    assert.equal(decimal(s), null, s);
  assert.equal(decimal("0"), 0n);
  assert.equal(decimal("0.000000000000000001"), 1n);
  assert.equal(decimal("-1", true), -scale);
  for (const s of ["", "-1", "100.0001", "1.00000"])
    assert.equal(percentage(s), null);
  assert.equal(percentage("100.0000"), 1000000n);
});
test("independent PC01 prefix example quantity3 price2 discount10 fee0.1 gives5.5", async () => {
  const s = await prepare(data(), id(1)),
    r = simulate(draft(), s);
  assert.equal(r.base, "6000000000000000000");
  assert.equal(r.discounted, "5400000000000000000");
  assert.equal(r.total, "5500000000000000000");
});
test("per1/per1000/per-million denominators retain their source quantity unit", async () => {
  for (const d of ["1", "1000", "1000000"]) {
    const s = await prepare(data((BigInt(d) * 3n * scale).toString()), id(1));
    assert.equal(
      simulate({ ...draft(), denominator: d, discount: "0", fee: "0" }, s)
        .total,
      (6n * scale).toString(),
    );
  }
});
test("signed quantities, explicit zero and fee-driven sign changes are preserved", async () => {
  const minus = await prepare(data((-3n * scale).toString()), id(1));
  assert.equal(simulate(draft(), minus).total, "-5300000000000000000");
  const zero = await prepare(data("0"), id(1));
  assert.equal(simulate({ ...draft(), fee: "0" }, zero).total, "0");
  const flip = await prepare(data((-scale).toString()), id(1));
  assert.equal(
    simulate({ ...draft(), unitPrice: "1", discount: "0", fee: "2" }, flip)
      .total,
    scale.toString(),
  );
  assert.equal(
    simulate({ ...draft(), discount: "100" }, minus).total,
    "100000000000000000",
  );
});
test("division traces retain signed non-money remainders", async () => {
  for (const sign of [1n, -1n]) {
    const s = await prepare(data((sign * scale).toString()), id(1)),
      r = simulate(
        {
          ...draft(),
          unitPrice: "1",
          denominator: "3",
          discount: "0",
          fee: "0",
        },
        s,
      );
    assert.equal(r.base, (sign * 333333333333333333n).toString());
    assert.equal(r.trace.base.remainder, (sign * scale).toString());
    assert.equal(
      BigInt(r.trace.base.quotient) * BigInt(r.trace.base.denominator) +
        BigInt(r.trace.base.remainder),
      BigInt(r.trace.base.numerator),
    );
  }
});
test("null quantity/unit cannot be replaced with consumed evidence", async () => {
  for (const mode of ["quantity", "unit"]) {
    const d = data();
    if (mode === "quantity") d.rows[0].pricing_quantity = null;
    else d.rows[0].pricing_unit = " ";
    d.rows[0].consumed_quantity = "3000000000000000000";
    d.rows[0].consumed_unit = "Hours";
    const s = await prepare(d, id(1));
    assert.throws(() => simulate(draft(), s));
  }
  const d = data();
  d.selection.period = "all";
  assert.ok(
    issues(draft(), await prepare(d, id(1))).includes("month_required"),
  );
});
test("every monetary stage respects bounds before later discount or cancellation", async () => {
  const s = await prepare(data(scale.toString()), id(1));
  assert.throws(
    () =>
      simulate(
        {
          ...draft(),
          unitPrice: "50000000000000000000",
          discount: "0",
          fee: "50000000000000000000",
        },
        s,
      ),
    /amount_overflow/,
  );
  const huge = await prepare(data((bound - 1n).toString()), id(1));
  assert.throws(
    () =>
      simulate({ ...draft(), unitPrice: "2", discount: "100", fee: "0" }, huge),
    /amount_overflow/,
  );
  assert.throws(
    () => simulate({ ...draft(), denominator: "0" }, s),
    /denominator_invalid/,
  );
});
test("target review is a distinct delta and opposite-sign overflow rejects", async () => {
  const s = await prepare(data(), id(1)),
    d = {
      ...draft(),
      target: "7",
      reason: "Synthetic review",
      reference: "Synthetic reference",
    },
    r = simulate(d, s),
    a = adjust(d, s, r);
  assert.equal(a.candidate, "5500000000000000000");
  assert.equal(a.delta, "1500000000000000000");
  assert.equal(BigInt(a.candidate) + BigInt(a.delta), BigInt(a.target));
  assert.equal(r.total, "5500000000000000000");
  assert.throws(() => adjust({ ...d, reason: "" }, s, r));
  assert.throws(
    () => adjust({ ...d, unitPrice: "3" }, s, r),
    /candidate_stale/,
  );
  const big = {
      ...draft(),
      unitPrice: "50000000000000000000",
      discount: "0",
      fee: "0",
      target: "-50000000000000000000",
      reason: "Synthetic",
      reference: "Synthetic",
    },
    one = await prepare(data(scale.toString()), id(1));
  assert.throws(() => adjust(big, one, simulate(big, one)), /amount_overflow/);
});
test("canonical current-page identity is mandatory and comparison context is complete", async () => {
  const d = data();
  await assert.rejects(prepare(d, id(9)));
  d.rows.push({ ...d.rows[0] });
  await assert.rejects(prepare(d, id(1)));
  const a = await prepare(data(), id(1)),
    b = await prepare(data(), id(2));
  assert.notEqual(a.context, b.context);
  assert.equal(comparable(simulate(draft(), a), simulate(draft(), b)), false);
  const changed = data();
  changed.rows[0].pricing_unit = "Token";
  assert.notEqual((await prepare(changed, id(1))).context, a.context);
});
test("bounded generated division traces preserve quotient identities and formula order", async () => {
  let seed = 18471n;
  for (let i = 0; i < 180; i++) {
    seed = (seed * 48271n) % 2147483647n;
    const q = (seed % 2n ? -1n : 1n) * seed;
    const s = await prepare(data(q.toString()), id(1));
    const d = {
      ...draft(),
      unitPrice: (seed % 100n).toString(),
      denominator: ((seed % 17n) + 1n).toString(),
      discount: (seed % 101n).toString(),
      fee: "0",
    };
    const r = simulate(d, s);
    for (const step of [r.trace.base, r.trace.discount]) {
      const n = BigInt(step.numerator),
        den = BigInt(step.denominator),
        rem = BigInt(step.remainder);
      assert.equal(BigInt(step.quotient) * den + rem, n);
      assert.ok(rem < den && rem > -den);
    }
    assert.equal(r.total, r.discounted);
    assert.ok(q < 0n ? BigInt(r.total) <= 0n : BigInt(r.total) >= 0n);
  }
});
test("browser parameters are projected and restored only under matching source and row", async () => {
  const s = await prepare(data(), id(1)),
    d = draft(),
    m = memory();
  save(m, s, { ...d, sourceQuantity: "forbidden" } as Draft);
  const body = JSON.parse(m.getItem(storageKey(s))!);
  assert.deepEqual(Object.keys(body).sort(), [
    "context",
    "draft",
    "profile",
    "row",
    "version",
  ]);
  assert.ok(!Object.hasOwn(body.draft, "sourceQuantity"));
  assert.deepEqual(restore(m, s), d);
  body.row = id(2);
  m.setItem(storageKey(s), JSON.stringify(body));
  assert.throws(() => restore(m, s));
});
test("malformed/foreign/stale profiles and storage failure do not report a saved draft", async () => {
  const s = await prepare(data(), id(1)),
    d = draft();
  for (const edit of [
    (b: any) => (b.version = 2),
    (b: any) => (b.context = id(99)),
    (b: any) => (b.profile = "other"),
    (b: any) => (b.draft.extra = "no"),
    (b: any) => (b.draft.unitPrice = "NaN"),
  ]) {
    const m = memory();
    save(m, s, d);
    const body = JSON.parse(m.getItem(storageKey(s))!);
    edit(body);
    m.setItem(storageKey(s), JSON.stringify(body));
    assert.throws(() => restore(m, s));
  }
  for (const text of ["{", "x".repeat(8193)]) {
    const m = memory();
    m.setItem(storageKey(s), text);
    assert.throws(() => restore(m, s));
  }
  assert.throws(() => save({ getItem: () => null, setItem: () => {} }, s, d));
  assert.throws(() =>
    save(
      {
        getItem: () => null,
        setItem: () => {
          throw Error("quota");
        },
      },
      s,
      d,
    ),
  );
});
