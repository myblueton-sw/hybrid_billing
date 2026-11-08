import { createElement } from "react";
import { renderToStaticMarkup } from "react-dom/server";
import { Comparison as ComparisonView } from "./ViewPanels";
import test from "node:test";
import assert from "node:assert/strict";
import {
  query,
  valid,
  type OverviewData,
  type Amount,
  type FocusRow,
} from "./api";
import { makeBaseline, compatible, compare } from "./model";
const hash = (v: string) => v.repeat(64);
function amount(currency = "USD", billed = "100", effective = "80"): Amount {
  return {
    currency,
    rows: 1,
    billed_cost: { coefficient: billed, present: 1, missing: 0 },
    effective_cost: { coefficient: effective, present: 1, missing: 0 },
  };
}
function fixture(amounts: Amount[] = [amount()]): OverviewData {
  let ordinal = 0;
  const rows = amounts.flatMap((a) =>
    Array.from({ length: a.rows }, (_, i): FocusRow => ({
      id: (++ordinal).toString(16).padStart(64, "0"),
      account: hash("2"),
      provider: "AWS",
      billing_account: "acct",
      sub_account: null,
      service: "Compute",
      currency: a.currency,
      charge_month: "2024-09",
      billing_start: "2024-09-01T00:00:00Z",
      billing_end: "2024-10-01T00:00:00Z",
      charge_start: "2024-09-10T02:00:00Z",
      charge_end: "2024-09-10T03:00:00Z",
      billed_cost:
        i < a.billed_cost.present
          ? i === 0
            ? a.billed_cost.coefficient
            : "0"
          : null,
      effective_cost:
        i < a.effective_cost.present
          ? i === 0
            ? a.effective_cost.coefficient
            : "0"
          : null,
      consumed_quantity: "1",
      pricing_quantity: "1",
      list_cost: "100",
      contracted_cost: "90",
      list_unit_price: "100",
      contracted_unit_price: "90",
      consumed_unit: "Hours",
      pricing_unit: "Hours",
      evidence_class: "official-public",
      batch: "batch",
      ordinal: String(ordinal),
    })),
  );
  const row_count = rows.length;
  return {
    schema: 1,
    scale: 18,
    snapshot: hash("a"),
    source_snapshot: hash("b"),
    selection: {
      period: "all",
      provider: "",
      account: "",
      basis: "billed_cost",
      q: "",
    },
    classification: "collected-focus-sample-query-only",
    financial_authority: false,
    observed_at: "2026-11-06T01:00:00Z",
    period_basis: "charge-start-in-tenant-timezone-no-proration",
    company: "north",
    company_mapping: "synthetic-sample-authorization",
    tenants: [{ key: "north-a", timezone: "UTC" }],
    versions: ["1.0"],
    delivery_count: 1,
    coverage: {
      slices: 1,
      finalized_slices: 1,
      complete_through: "2024-10-01T00:00:00Z",
      expectations: "operator-asserted-unverified",
    },
    provenance: row_count ? { "official-public": row_count } : {},
    options: {
      providers: ["AWS"],
      periods: ["2024-09"],
      currencies: amounts.map((a) => a.currency),
    },
    authorized_rows: Math.max(10, row_count),
    row_count,
    amounts: structuredClone(amounts),
    providers: row_count
      ? [{ provider: "AWS", amounts: structuredClone(amounts) }]
      : [],
    months: row_count
      ? [{ period: "2024-09", amounts: structuredClone(amounts) }]
      : [],
    account_count: row_count ? 1 : 0,
    account_matches: row_count ? 1 : 0,
    accounts: row_count
      ? [
          {
            id: hash("2"),
            tenant: "north-a",
            provider: "AWS",
            billing_account: "acct",
            sub_account: null,
            native_ancestry: "unverified",
            rows: row_count,
            amounts: structuredClone(amounts),
          },
        ]
      : [],
    rows: rows.slice(0, 25),
    next_after: row_count > 25 ? 25 : null,
    next_account_after: null,
    after: 0,
    account_after: 0,
  };
}
const defaultQuery = () => query(new URLSearchParams());

test("B08 query has explicit source defaults and normalizes account-option search", () => {
  const q = query(
    new URLSearchParams(
      "focus-q=%20north%20%20AWS%20&focus-after=25&focus-snapshot=" +
        hash("a") +
        "&tenant=foreign&marketplace-row=" +
        hash("f"),
    ),
  );
  assert.equal(q.get("q"), "north AWS");
  assert.equal(q.get("provider"), "");
  assert.equal(q.get("period"), "all");
  assert.equal(q.get("basis"), "billed_cost");
  assert.equal(q.get("after"), "25");
  assert.equal(q.has("tenant"), false);
  assert.equal(q.has("marketplace-row"), false);
});

test("B08 validates exact collected costs and never accepts financial authority", () => {
  const d = fixture();
  assert.equal(valid(d, defaultQuery()), true);
  d.financial_authority = true as false;
  assert.equal(valid(d, defaultQuery()), false);
});

test("B08 rejects impossible calendars and timestamp strings without a timezone", () => {
  for (const mutate of [
    (d: OverviewData) => {
      d.observed_at = "2024-02-30T01:00:00Z";
    },
    (d: OverviewData) => {
      d.observed_at = "2026-11-06T01:00:00";
    },
    (d: OverviewData) => {
      d.coverage.complete_through = "2024-13-01T00:00:00Z";
    },
    (d: OverviewData) => {
      d.rows[0].charge_start = "2024-09-10T25:00:00Z";
    },
    (d: OverviewData) => {
      d.rows[0].billing_end = d.rows[0].billing_start;
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, defaultQuery()), false);
  }
});

test("B08 rejects duplicate authorization, option and page identities", () => {
  for (const mutate of [
    (d: OverviewData) => {
      d.tenants.push({ ...d.tenants[0] });
    },
    (d: OverviewData) => {
      d.tenants[0].timezone = "Not/A_Timezone";
    },
    (d: OverviewData) => {
      d.versions.push("1.0");
    },
    (d: OverviewData) => {
      d.options.providers.push("AWS");
    },
    (d: OverviewData) => {
      d.options.periods.push("2024-09");
    },
    (d: OverviewData) => {
      d.options.currencies.push("USD");
    },
    (d: OverviewData) => {
      d.rows[1].id = d.rows[0].id;
    },
  ]) {
    const a = amount();
    a.rows = 2;
    a.billed_cost.present = 2;
    a.effective_cost.present = 2;
    const d = fixture([a]);
    mutate(d);
    assert.equal(valid(d, defaultQuery()), false);
  }
});

test("B08 independently rejects row and account scope leaks", () => {
  for (const mutate of [
    (d: OverviewData) => {
      d.rows[0].provider = "Foreign";
    },
    (d: OverviewData) => {
      d.rows[0].currency = "EUR";
    },
    (d: OverviewData) => {
      d.rows[0].charge_month = "2024-08";
    },
    (d: OverviewData) => {
      d.accounts[0].tenant = "foreign";
    },
    (d: OverviewData) => {
      d.accounts[0].provider = "Foreign";
    },
    (d: OverviewData) => {
      d.accounts[0].native_ancestry = "verified-owner";
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, defaultQuery()), false);
  }
  const d = fixture();
  d.selection.account = hash("3");
  const q = defaultQuery();
  q.set("account", hash("3"));
  assert.equal(valid(d, q), false);
});

test("B08 rejects full-page cost contradictions and requires snapshot for page offsets", () => {
  const wrong = fixture();
  wrong.rows[0].billed_cost = "999";
  assert.equal(valid(wrong, defaultQuery()), false);
  const a = amount();
  a.rows = 2;
  a.billed_cost.present = 2;
  a.effective_cost.present = 2;
  const d = fixture([a]);
  d.after = 1;
  d.rows = d.rows.slice(1);
  const q = defaultQuery();
  q.set("after", "1");
  assert.equal(valid(d, q), false);
  q.set("snapshot", d.snapshot);
  assert.equal(valid(d, q), true);
});

test("baseline copies only bounded whole-selection evidence and ignores page or month rows", () => {
  const d = fixture();
  const baseline = makeBaseline(d);
  assert.equal(baseline.row_count, 1);
  assert.equal(baseline.amounts[0].billed_cost.coefficient, "100");
  assert.equal(Object.hasOwn(baseline, "rows"), false);
  assert.equal(Object.hasOwn(baseline, "months"), false);
  d.amounts[0].billed_cost.coefficient = "900";
  d.tenants[0].timezone = "Asia/Seoul";
  assert.equal(baseline.amounts[0].billed_cost.coefficient, "100");
  assert.equal(compatible(baseline, d), false);
});

test("baseline cannot redisplay across source, company, member, version or financial-scope changes", () => {
  const baseline = makeBaseline(fixture());
  for (const mutate of [
    (d: OverviewData) => {
      d.source_snapshot = hash("f");
    },
    (d: OverviewData) => {
      d.company = "south";
    },
    (d: OverviewData) => {
      d.company_mapping = "other";
    },
    (d: OverviewData) => {
      d.tenants[0].key = "south-a";
    },
    (d: OverviewData) => {
      d.tenants[0].timezone = "Asia/Seoul";
    },
    (d: OverviewData) => {
      d.tenants.push({ key: "north-b", timezone: "UTC" });
    },
    (d: OverviewData) => {
      d.versions = ["1.2"];
    },
    (d: OverviewData) => {
      d.period_basis = "billing-period";
    },
    (d: OverviewData) => {
      d.selection.provider = "AWS";
    },
    (d: OverviewData) => {
      d.selection.account = hash("2");
    },
    (d: OverviewData) => {
      d.selection.period = "2024-09";
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(compatible(baseline, d), false);
    assert.equal(compare(baseline, d), null);
  }
});

test("baseline identity is canonical and ignores display pages, search, basis, query reference and read time", () => {
  const d = fixture();
  d.tenants.push({ key: "north-b", timezone: "Asia/Seoul" });
  d.versions.push("1.2");
  const baseline = makeBaseline(d),
    current = structuredClone(d);
  current.tenants.reverse();
  current.versions.reverse();
  current.after = 25;
  current.account_after = 25;
  current.selection.q = "account-only";
  current.selection.basis = "effective_cost";
  current.snapshot = hash("c");
  current.observed_at = "2026-11-06T02:00:00Z";
  assert.equal(compatible(baseline, current), true);
  assert.equal(compare(baseline, current)?.[0].billed_cost, "0");
});

test("comparison preserves signed exact values beyond binary integer precision and compares bases separately", () => {
  const old = fixture([
    amount("USD", "90071992547409930000000000000000001", "-20"),
  ]);
  const current = fixture([
    amount("USD", "90071992547409930000000000000000004", "-25"),
  ]);
  const result = compare(makeBaseline(old), current)!;
  assert.equal(result[0].billed_cost, "3");
  assert.equal(result[0].effective_cost, "-5");
  assert.equal(
    result[0].baseline?.billed_cost.coefficient,
    old.amounts[0].billed_cost.coefficient,
  );
});

test("missing currency, partial, all-missing and empty observations never become zero deltas", () => {
  const old = fixture([amount("USD"), amount("EUR", "200", "150")]);
  const current = fixture([
    amount("USD", "130", "90"),
    amount("GBP", "7", "5"),
  ]);
  const results = compare(makeBaseline(old), current)!;
  assert.deepEqual(
    results.map((r) => [r.currency, r.billed_cost]),
    [
      ["EUR", null],
      ["GBP", null],
      ["USD", "30"],
    ],
  );
  const partial = amount();
  partial.rows = 2;
  partial.billed_cost.missing = 1;
  partial.effective_cost.missing = 1;
  const result = compare(makeBaseline(fixture([partial])), fixture())!;
  assert.equal(result[0].billed_cost, null);
  assert.equal(result[0].effective_cost, null);
  const absent = amount();
  absent.billed_cost = { coefficient: null, present: 0, missing: 1 };
  assert.equal(
    compare(makeBaseline(fixture([absent])), fixture())?.[0].billed_cost,
    null,
  );
  assert.deepEqual(compare(makeBaseline(fixture([])), fixture([])), []);
});

test("known zero remains an observed zero and an incomplete second basis remains unknown", () => {
  const d = fixture([amount("USD", "0", "0")]);
  const before = makeBaseline(d);
  d.amounts[0].effective_cost = { coefficient: null, present: 0, missing: 1 };
  const result = compare(before, d)!;
  assert.equal(result[0].billed_cost, "0");
  assert.equal(result[0].effective_cost, null);
});

test("exact difference obeys antisymmetry and signed-cancellation invariants across bounded edge values", () => {
  const values = [
    "0",
    "1",
    "-1",
    "9007199254740993",
    "-9007199254740993",
    "99999999999999999999999999999999999999",
    "-99999999999999999999999999999999999999",
  ];
  for (const a of values)
    for (const b of values) {
      const old = fixture([amount("USD", a)]),
        current = fixture([amount("USD", b)]);
      const forward = compare(makeBaseline(old), current)![0].billed_cost!,
        reverse = compare(makeBaseline(current), old)![0].billed_cost!;
      assert.equal(BigInt(forward) + BigInt(reverse), 0n);
      assert.equal(compare(makeBaseline(old), old)![0].billed_cost, "0");
    }
  const zero = fixture([amount("USD", "0")]),
    negative = fixture([amount("USD", "-35")]),
    positive = fixture([amount("USD", "35")]);
  assert.equal(compare(makeBaseline(zero), negative)![0].billed_cost, "-35");
  assert.equal(compare(makeBaseline(negative), positive)![0].billed_cost, "70");
});

test("B08 rejects missing source authority metadata and contradictory provider/account summaries", () => {
  for (const mutate of [
    (d: OverviewData) => {
      d.versions = [];
    },
    (d: OverviewData) => {
      d.delivery_count = 0;
    },
    (d: OverviewData) => {
      d.providers[0].provider = "Oracle";
      d.options.providers.push("Oracle");
    },
    (d: OverviewData) => {
      d.accounts[0].amounts[0].billed_cost.coefficient = "999";
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, defaultQuery()), false);
  }
});

test("comparison identifies the partial side instead of presenting its subtotal as complete", () => {
  const before = fixture();
  before.amounts[0].rows = 2;
  before.row_count = 2;
  before.amounts[0].billed_cost = {
    coefficient: "100000000000000000",
    present: 1,
    missing: 1,
  };
  before.amounts[0].effective_cost = {
    coefficient: "80000000000000000",
    present: 1,
    missing: 1,
  };
  const baseline = makeBaseline(before),
    after = structuredClone(before);
  after.observed_at = "2026-11-06T02:00:00Z";
  after.amounts[0].billed_cost = {
    coefficient: "150000000000000000",
    present: 2,
    missing: 0,
  };
  after.amounts[0].effective_cost = {
    coefficient: "120000000000000000",
    present: 2,
    missing: 0,
  };
  const html = renderToStaticMarkup(
    createElement(ComparisonView, { locale: "en", data: after, baseline }),
  );
  const text = html.replace(/<[^>]*>/g, "");
  assert.match(text, /Present \/ missing\s+1 \/ 1/);
  assert.match(text, /Present \/ missing\s+2 \/ 0/);
  assert.match(text, /Unavailable: complete matching observations required/);
  assert.equal(compare(baseline, after)![0].billed_cost, null);
});
