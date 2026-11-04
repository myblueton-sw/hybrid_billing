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
  dateDays,
  roundedCount,
  simulate,
  comparable,
  issues,
  scale,
  bound,
  intermediateBound,
  modes,
  type Draft,
} from "./model";
import { save, restore, storageKey } from "./store";
const id = (n: number) => n.toString(16).padStart(64, "0");
function data(amount: string | null = "1000000000000000000"): OverviewData {
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
        billed_cost: amount,
        effective_cost: "2000000000000000000",
        charge_month: "2024-09",
      } as FocusRow,
    ],
  } as OverviewData;
}
const draft = (): Draft => ({
  ...emptyDraft(),
  name: "Synthetic FX review",
  target: "KRW",
  treatment: "unconverted",
  rate: "900",
  quoteUnit: "100",
  provider: "Synthetic manual evidence",
  reference: "Synthetic review only",
  quoteDate: "2024-09-01",
  reviewDate: "2024-09-02",
  maxAge: "1",
  markup: "0",
  fee: "0",
  quantum: "0.05",
  mode: "half_even",
});
const memory = () => {
  const map = new Map<string, string>();
  return {
    getItem: (k: string) => map.get(k) ?? null,
    setItem: (k: string, v: string) => {
      map.set(k, v);
    },
  };
};
test("explicit blank parameters, finite decimals and percentages", () => {
  assert.ok(Object.values(emptyDraft()).every((v) => v === ""));
  for (const v of [
    "",
    " 1",
    "1e2",
    "01",
    "-1",
    "NaN",
    "1.0000000000000000000",
    "100000000000000000000",
  ])
    assert.equal(decimal(v), null);
  assert.equal(decimal("0"), 0n);
  assert.equal(decimal("-0.5", true), -scale / 2n);
  assert.equal(decimal("0.000000000000000001"), 1n);
  for (const v of ["", "-1", "100.0001", "1.00000"])
    assert.equal(percentage(v), null);
  assert.equal(percentage("100"), 1000000n);
});
test("real UTC dates preserve leap days and reject impossible calendars", () => {
  assert.equal(dateDays("2024-02-29")! + 1, dateDays("2024-03-01"));
  for (const v of [
    "2023-02-29",
    "2024-02-30",
    "2024-13-01",
    "2024-9-01",
    "0000-01-01",
    "2024-09-01Z",
  ])
    assert.equal(dateDays(v), null);
  assert.equal(dateDays("0099-01-01")! + 365, dateDays("0100-01-01"));
});
test("all signed ties and exact multiples for six modes", () => {
  const cases: [bigint, bigint, bigint[]][] = [
    [3n, 2n, [1n, 2n, 1n, 2n, 2n, 2n]],
    [-3n, 2n, [-1n, -2n, -2n, -1n, -2n, -2n]],
    [5n, 2n, [2n, 3n, 2n, 3n, 3n, 2n]],
    [-5n, 2n, [-2n, -3n, -3n, -2n, -3n, -2n]],
  ];
  for (const [n, d, want] of cases)
    modes.forEach((m, i) =>
      assert.equal(roundedCount(n, d, m), want[i], `${n}/${d} ${m}`),
    );
  for (const m of modes) {
    assert.equal(roundedCount(0n, 2n, m), 0n);
    assert.equal(roundedCount(-6n, 2n, m), -3n);
  }
  assert.throws(() => roundedCount(1n, 0n, "floor"));
  assert.throws(() => roundedCount(1n, 2n, "unknown"));
  assert.throws(() => roundedCount(intermediateBound, 2n, "half_even"));
});
test("independent enumeration oracle checks 2100 signed mode cases", () => {
  let count = 0;
  for (let n = -17; n <= 17; n++)
    for (let d = 1; d <= 10; d++)
      for (const mode of modes) {
        const candidates = Array.from({ length: 41 }, (_, i) => i - 20);
        let pool = candidates;
        if (
          mode === "floor" ||
          mode === "ceiling" ||
          mode === "toward_zero" ||
          mode === "away_from_zero"
        ) {
          const below =
            mode === "floor" ||
            (mode === "toward_zero" && n >= 0) ||
            (mode === "away_from_zero" && n < 0);
          pool = pool.filter((z) => (below ? z * d <= n : z * d >= n));
        }
        const distance = Math.min(...pool.map((z) => Math.abs(z * d - n)));
        pool = pool.filter((z) => Math.abs(z * d - n) === distance);
        const expected =
          pool.length === 1
            ? pool[0]
            : mode === "half_even"
              ? pool.find((z) => Math.abs(z) % 2 === 0)!
              : pool.sort((a, b) => Math.abs(b) - Math.abs(a))[0];
        assert.equal(
          roundedCount(BigInt(n), BigInt(d), mode),
          BigInt(expected),
          `${n}/${d} ${mode}`,
        );
        count++;
      }
  assert.equal(count, 2100);
});
test("per100 quote retains actual direction and exact nondecimal quantum", async () => {
  const d = data((100n * scale).toString());
  d.rows[0].currency = "JPY";
  const s = await prepare(d, id(1)),
    r = simulate(draft(), s);
  assert.equal(r.sourceCurrency, "JPY");
  assert.equal(r.currency, "KRW");
  assert.equal(r.total, (900n * scale).toString());
  assert.equal(r.normalized.numerator, (900n * scale).toString());
  assert.equal(r.normalized.denominator, (100n * scale).toString());
  assert.equal(r.difference.numerator, "0");
});
test("no intermediate truncation before markup and quantum", async () => {
  const s = await prepare(data("1"), id(1)),
    d = {
      ...draft(),
      rate: "1",
      quoteUnit: "3",
      markup: "100",
      quantum: "0.000000000000000001",
      mode: "half_up",
    },
    r = simulate(d, s);
  assert.equal(r.total, "1");
  assert.equal(
    BigInt(r.trace.quoted.numerator) / BigInt(r.trace.quoted.denominator),
    0n,
  );
  assert.ok(BigInt(r.difference.numerator) > 0n);
  assert.equal(
    BigInt(r.rounding.numerator),
    BigInt(r.rounding.quotient) * BigInt(r.rounding.denominator) +
      BigInt(r.rounding.remainder),
  );
});
test("signed source, positive markup and targetfee remain separate", async () => {
  const s = await prepare(data((-100n * scale).toString()), id(1));
  assert.equal(
    simulate({ ...draft(), markup: "10", fee: "1" }, s).total,
    (-989n * scale).toString(),
  );
  const tiny = await prepare(data((-scale / 100n).toString()), id(1));
  assert.equal(
    simulate(
      { ...draft(), rate: "1", quoteUnit: "1", fee: "0.1", quantum: "0.01" },
      tiny,
    ).total,
    ((9n * scale) / 100n).toString(),
  );
  assert.equal(simulate(draft(), await prepare(data("0"), id(1))).total, "0");
});
test("manual declarations, quote validity, date and source blockers", async () => {
  const s = await prepare(data(), id(1));
  const patches: Partial<Draft>[] = [
    { rate: "0" },
    { rate: "-1" },
    { quoteUnit: "0" },
    { quantum: "0" },
    { fee: "-1" },
    { markup: "100.0001" },
    { mode: "wat" },
    { target: "USD" },
    { target: "ZZZ" },
    { treatment: "" },
    { treatment: "already" },
    { provider: "" },
    { reference: " " },
    { quoteDate: "2024-02-30" },
    { quoteDate: "2024-09-03" },
    { maxAge: "0" },
    { maxAge: "36601" },
    { maxAge: "01" },
  ];
  for (const patch of patches)
    assert.throws(
      () => simulate({ ...draft(), ...patch }, s),
      JSON.stringify(patch),
    );
  assert.ok(
    issues({ ...draft(), name: "" }, s, true).includes("name_required"),
  );
  assert.throws(() =>
    simulate(draft(), { ...s, row: { ...s.row, billed_cost: null } }),
  );
  assert.throws(() =>
    simulate(draft(), {
      ...s,
      data: { ...s.data, selection: { ...s.data.selection, period: "all" } },
    }),
  );
});
test("overflow rejects instead of silently changing quantum", async () => {
  const s = await prepare(data((bound - 1n).toString()), id(1));
  assert.throws(
    () =>
      simulate(
        {
          ...draft(),
          rate: "2",
          quoteUnit: "1",
          quantum: "0.000000000000000001",
        },
        s,
      ),
    /amount_overflow/,
  );
  assert.throws(
    () => simulate({ ...draft(), rate: "1", quoteUnit: "1", fee: "1" }, s),
    /amount_overflow/,
  );
});
test("row identity, basis and targetcurrency gate comparisons", async () => {
  const d = data(),
    s = await prepare(d, id(1)),
    a = simulate(draft(), s);
  await assert.rejects(() => prepare(d, id(2)));
  await assert.rejects(() =>
    prepare({ ...d, rows: [d.rows[0], d.rows[0]] }, id(1)),
  );
  const changed = await prepare(
    { ...d, selection: { ...d.selection, basis: "effective_cost" } },
    id(1),
  );
  assert.notEqual(changed.context, s.context);
  assert.equal(comparable(a, simulate({ ...draft(), rate: "901" }, s)), true);
  assert.equal(
    comparable(a, simulate({ ...draft(), target: "EUR" }, s)),
    false,
  );
  assert.equal(comparable(a, simulate(draft(), changed)), false);
  const d1 = { ...draft() };
  const result = simulate(d1, s);
  d1.rate = "4";
  assert.equal(result.draft.rate, "900");
});
test("strict parameter projection, authorized context and failed persistence", async () => {
  const s = await prepare(data(), id(1)),
    m = memory(),
    d = draft();
  save(m, s, d);
  assert.deepEqual(restore(m, s), d);
  const raw = m.getItem(storageKey(s))!;
  assert.equal(raw.includes("billed_cost"), false);
  assert.equal(raw.includes("source_snapshot"), false);
  const initial = JSON.parse(raw);
  for (const patch of [
    { profile: "old" },
    { row: id(2) },
    { context: id(20) },
    { extra: "x" },
    { draft: { ...d, mode: "bad" } },
    { draft: { ...d, rate: "0" } },
    { draft: { ...d, target: "ZZZ" } },
    { draft: { ...d, sourceAmount: "1" } },
    { draft: { ...d, provider: "x".repeat(301) } },
  ]) {
    m.setItem(storageKey(s), JSON.stringify({ ...initial, ...patch }));
    assert.throws(() => restore(m, s));
  }
  m.setItem(storageKey(s), "x".repeat(8193));
  assert.throws(() => restore(m, s));
  assert.throws(() => save({ getItem: () => null, setItem: () => {} }, s, d));
  assert.throws(() =>
    save(
      {
        getItem: () => null,
        setItem: () => {
          throw Error("blocked");
        },
      },
      s,
      d,
    ),
  );
  assert.equal(restore(memory(), s), null);
});
test("inexact nondecimal quantum scaling and signed ties through simulate", async () => {
  const positive = await prepare(data("1030000000000000000"), id(1));
  const d = {
    ...draft(),
    rate: "1",
    quoteUnit: "1",
    quantum: "0.05",
    mode: "half_even",
  };
  assert.equal(simulate(d, positive).total, "1050000000000000000");
  const negative = await prepare(data("-1025000000000000000"), id(1));
  assert.equal(simulate(d, negative).total, "-1000000000000000000");
  assert.equal(
    simulate({ ...d, mode: "half_up" }, negative).total,
    "-1050000000000000000",
  );
});
