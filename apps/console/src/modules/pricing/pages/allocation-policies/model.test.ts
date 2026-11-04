import { test } from "node:test";
import assert from "node:assert/strict";
import type { OverviewData } from "../../../sources/collected-focus/api";
import {
  prepare,
  emptyDraft,
  simulate,
  issues,
  percentage,
  comparable,
  restricted,
  type Prepared,
  type Draft,
} from "./model";
import { save, restore, storageKey } from "./store";
function data(
  value: string | null = "101",
  present = 1,
  missing = 0,
): OverviewData {
  return {
    company: "synthetic",
    source_snapshot: "a".repeat(64),
    snapshot: "b".repeat(64),
    selection: {
      period: "2024-09",
      provider: "",
      account: "",
      q: "",
      basis: "billed_cost",
    },
    after: 0,
    account_after: 0,
    tenants: [
      { key: "scope-b", timezone: "UTC" },
      { key: "scope-a", timezone: "Asia/Seoul" },
    ],
    amounts: [
      {
        currency: "USD",
        rows: present + missing,
        billed_cost: { coefficient: value, present, missing },
        effective_cost: { coefficient: value, present, missing },
      },
    ],
    rows: [],
  } as unknown as OverviewData;
}
async function sample(value = "101") {
  const source = await prepare(data(value), "USD");
  const draft = {
    name: "Synthetic review",
    weights: source.targets.map((t) => ({ target: t.id, percentage: "50" })),
  };
  return { source, draft };
}
function memory() {
  const m = new Map<string, string>();
  return {
    getItem: (k: string) => m.get(k) ?? null,
    setItem: (k: string, v: string) => {
      m.set(k, v);
    },
    removeItem: (k: string) => {
      m.delete(k);
    },
  };
}
test("strict percentage strings preserve four decimals and full100 denominator", () => {
  for (const invalid of [
    "",
    " 1",
    "01",
    "1e2",
    "-1",
    "NaN",
    ".5",
    "100.0001",
    "101",
    "1.00000",
  ])
    assert.equal(percentage(invalid), null, invalid);
  assert.equal(percentage("0.0001"), 1n);
  assert.equal(percentage("100.0000"), 1000000n);
});
test("signed precision remainder is visible and never redistributed", async () => {
  for (const n of ["1", "-1"]) {
    const { source, draft } = await sample(n);
    const r = simulate(draft, source);
    assert.deepEqual(
      r.rows.map((x) => x.amount),
      ["0", "0"],
    );
    assert.equal(r.assigned, "0");
    assert.equal(r.unassigned, "0");
    assert.equal(r.remainder, n);
  }
});
test("less than100 preserves unallocated funds and zero remains real zero", async () => {
  const { source, draft } = await sample("101");
  draft.weights[0].percentage = "25";
  draft.weights[1].percentage = "25";
  const r = simulate(draft, source);
  assert.equal(r.assigned, "50");
  assert.equal(r.unassigned, "51");
  assert.equal(r.remainder, "0");
  const z = await sample("0");
  assert.deepEqual(issues(z.draft, z.source), []);
  assert.equal(simulate(z.draft, z.source).source, "0");
});
test("complete source aggregate is independent of source detail rows", async () => {
  const s = await prepare(
    { ...data("99999999999999999999999999999999999999"), rows: [] },
    "USD",
  );
  const d = emptyDraft(s);
  d.weights.forEach((w) => (w.percentage = "50"));
  const r = simulate(d, s);
  assert.equal(
    BigInt(r.assigned) + BigInt(r.unassigned) + BigInt(r.remainder),
    99999999999999999999999999999999999999n,
  );
});
test("missing source, wrongcurrency, month, duplicate or incomplete targets block", async () => {
  const good = await sample();
  for (const [value, present, missing] of [
    [null, 0, 1],
    ["12", 1, 1],
  ] as const) {
    const s = await prepare(data(value, present, missing), "USD");
    assert.throws(() => simulate(good.draft, s));
  }
  const s = await prepare(data(), "CNY");
  assert.ok(issues(good.draft, s).includes("amount_unavailable"));
  const month = data();
  month.selection.period = "all";
  assert.ok(
    issues(good.draft, await prepare(month, "USD")).includes("month_required"),
  );
  for (const weights of [
    [good.draft.weights[0]],
    [good.draft.weights[0], good.draft.weights[0]],
    [...good.draft.weights, { target: "f".repeat(64), percentage: "1" }],
  ])
    assert.ok(
      issues({ ...good.draft, weights }, good.source).includes(
        "target_manifest",
      ),
    );
});
test("duplicate and over100 authorized scopes reject instead of truncating", async () => {
  const d = data();
  d.tenants[1] = { ...d.tenants[0] };
  await assert.rejects(prepare(d, "USD"));
  d.tenants = Array.from({ length: 101 }, (_, i) => ({
    key: `synthetic-${i}`,
    timezone: "UTC",
  }));
  await assert.rejects(prepare(d, "USD"));
});
test("restricted source cannot silently become whole source", async () => {
  for (const field of ["account", "q"] as const) {
    const d = data();
    d.selection[field] = "restricted";
    await assert.rejects(prepare(d, "USD"));
  }
  for (const field of ["after", "account_after"] as const) {
    const d = data();
    d[field] = 25;
    await assert.rejects(prepare(d, "USD"));
  }
  for (const q of [
    "focus-account=x",
    "focus-q=x",
    "focus-after=25",
    "focus-account_after=1",
    "focus-after=invalid",
  ])
    assert.equal(restricted(new URLSearchParams(q)), true);
  assert.equal(restricted(new URLSearchParams("focus-after=0")), false);
});
test("bounded generated signed cases conserve and target order cannot change values", async () => {
  const base = await sample();
  let seed = 71421n;
  for (let i = 0; i < 250; i++) {
    seed = (seed * 48271n) % 2147483647n;
    const amount = (seed % 2n ? -1n : 1n) * (seed * seed);
    const p = seed % 1000001n;
    const q = (seed * 7n) % (1000001n - p);
    const s = { ...base.source, data: data(amount.toString()) };
    const d = {
      ...base.draft,
      weights: base.draft.weights.map((w, j) => ({
        ...w,
        percentage:
          ((j ? p : q) / 10000n).toString() +
          "." +
          ((j ? p : q) % 10000n).toString().padStart(4, "0"),
      })),
    };
    if (p + q === 0n) continue;
    const a = simulate(d, s),
      b = simulate({ ...d, weights: [...d.weights].reverse() }, s);
    assert.equal(
      BigInt(a.assigned) + BigInt(a.unassigned) + BigInt(a.remainder),
      amount,
    );
    assert.deepEqual(
      [...a.rows].sort((x, y) => x.target.localeCompare(y.target)),
      [...b.rows].sort((x, y) => x.target.localeCompare(y.target)),
    );
    assert.ok(
      a.rows.every((r) =>
        amount >= 0n ? BigInt(r.amount) >= 0n : BigInt(r.amount) <= 0n,
      ),
    );
  }
});
test("result copies weights and comparison rejects different source context", async () => {
  const { source, draft } = await sample();
  const first = simulate(draft, source);
  draft.weights[0].percentage = "25";
  assert.equal(first.rows[0].percentage, "50");
  const second = simulate(draft, source);
  assert.equal(comparable(first, second), true);
  const fresh = await prepare(
    { ...source.data, snapshot: "c".repeat(64) },
    "USD",
  );
  assert.equal(comparable(first, simulate(draft, fresh)), false);
});
test("metadata save restores only matching fresh context; extra fields cannot persist", async () => {
  const { source, draft } = await sample();
  const m = memory();
  save(m, source, { ...draft, secret: "forbidden" } as Draft);
  const raw = m.getItem(storageKey(source))!;
  assert.ok(!raw.includes("forbidden"));
  assert.ok(!raw.includes("scope-a"));
  assert.ok(!raw.includes("coefficient"));
  assert.deepEqual(restore(m, source), draft);
  const next = await prepare(
    { ...source.data, source_snapshot: "c".repeat(64) },
    "USD",
  );
  m.setItem(storageKey(next), raw);
  assert.throws(() => restore(m, next));
});
test("malformed metadata, wrongprofile and broken storage readback fail visibly", async () => {
  const { source, draft } = await sample();
  for (const raw of ["{", "x".repeat(32769)]) {
    const m = memory();
    m.setItem(storageKey(source), raw);
    assert.throws(() => restore(m, source));
  }
  for (const change of [
    (b: any) => (b.profile = "other"),
    (b: any) => (b.draft.weights[0].target = "f".repeat(64)),
    (b: any) => (b.extra = "unknown"),
    (b: any) => (b.draft.weights[0].percentage = "100.0001"),
  ]) {
    const m = memory();
    save(m, source, draft);
    const body = JSON.parse(m.getItem(storageKey(source))!);
    change(body);
    m.setItem(storageKey(source), JSON.stringify(body));
    assert.throws(() => restore(m, source));
  }
  assert.throws(() =>
    save(
      { getItem: () => null, setItem: () => {}, removeItem: () => {} },
      source,
      draft,
    ),
  );
  assert.throws(() =>
    save(
      {
        getItem: () => null,
        setItem: () => {
          throw Error("quota");
        },
        removeItem: () => {},
      },
      source,
      draft,
    ),
  );
});
