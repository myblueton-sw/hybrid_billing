import { test } from "node:test";
import assert from "node:assert/strict";
import {
  valid,
  query,
  profile,
  fieldNames,
  type Data,
  type Group,
  type Costs,
  type Charge,
  type Field,
} from "./api";
const id = (n: number) => n.toString(16).padStart(64, "0");
const stamp = "2024-09-01T00:00:00+00:00",
  end = "2024-10-01T00:00:00+00:00";
const zero = (): Costs => ({
  rows: 0,
  billed_cost: { coefficient: null, present: 0, missing: 0 },
  effective_cost: { coefficient: null, present: 0, missing: 0 },
});
const costs = (): Costs => ({
  rows: 1,
  billed_cost: { coefficient: "100000000000000000000", present: 1, missing: 0 },
  effective_cost: {
    coefficient: "80000000000000000000",
    present: 1,
    missing: 0,
  },
});
function fixture(): Data {
  const g: Group = {
    ...costs(),
    id: id(1),
    source_id: "synthetic-sp-1",
    tenant: "north-a",
    account: id(4),
    provider: "AWS",
    billing_account: "synthetic-account",
    sub_account: null,
    currency: "USD",
    metadata: Object.fromEntries(
      fieldNames
        .filter((k) => k !== "id")
        .map((k) => [k, { values: [], distinct: 0, truncated: false }]),
    ) as unknown as Group["metadata"],
    statuses: { used: costs(), unused: zero(), unknown: zero() },
    versions: ["1.0"],
    mapping: { version_mapped_rows: 1, version_unmapped_rows: 0 },
    intervals: {
      charge_start: stamp,
      charge_end: end,
      billing_start: stamp,
      billing_end: end,
    },
  };
  return {
    schema: 1,
    scale: 18,
    profile,
    classification: "collected-commitment-evidence-only",
    financial_authority: false,
    snapshot: id(10),
    source_snapshot: id(11),
    mapping_digest: id(12),
    company: "sample-company",
    versions: ["1.0"],
    observed_at: stamp,
    selection: { provider: "AWS", period: "all", account: "", q: "" },
    period_basis: "charge-start-in-tenant-timezone-no-proration",
    coverage: {
      slices: 1,
      finalized_slices: 1,
      complete_through: null,
      expectations: "operator-asserted-unverified",
    },
    options: { providers: ["AWS"], periods: ["2024-09"], currencies: ["USD"] },
    authorized_rows: 1,
    row_count: 1,
    identified_rows: 1,
    unidentified_rows: 0,
    group_count: 1,
    identified_amounts: [{ currency: "USD", ...costs() }],
    unidentified_amounts: [],
    unidentified_reasons: {
      source_null: 0,
      absent: 0,
      blank: 0,
      unmapped_version: 0,
      invalid_type: 0,
    },
    groups: [g],
    after: 0,
    next_after: null,
    accounts: [
      {
        id: id(4),
        tenant: "north-a",
        provider: "AWS",
        billing_account: "synthetic-account",
        sub_account: null,
        rows: 1,
      },
    ],
    account_count: 1,
    account_matches: 1,
    account_after: 0,
    next_account_after: null,
    selected: null,
  };
}
function detail(d: Data): Charge {
  const g = d.groups[0];
  return {
    id: id(5),
    group: g.id,
    tenant: g.tenant,
    account: g.account,
    provider: g.provider,
    billing_account: g.billing_account,
    sub_account: g.sub_account,
    currency: g.currency,
    service: "synthetic-service",
    billed_cost: g.billed_cost.coefficient,
    effective_cost: g.effective_cost.coefficient,
    charge_start: stamp,
    charge_end: end,
    billing_start: stamp,
    billing_end: end,
    version: "1.0",
    batch: "synthetic-batch",
    ordinal: "1",
    fields: Object.fromEntries(
      fieldNames.map((k) => [
        k,
        k === "id"
          ? { state: "value", value: g.source_id }
          : { state: "source_null", value: null },
      ]),
    ) as Record<(typeof fieldNames)[number], Field>,
    evidence_class: "synthetic-contract-fixture",
  };
}
const base = () => new URLSearchParams("provider=AWS&period=all");
test("bounded query preserves account search and explicit canonical group context", () => {
  const q = query(
    new URLSearchParams(
      "lang=ko&view=benefits&focus-q=++test+&commitment-group=" +
        id(1) +
        "&commitment-row-after=25",
    ),
  );
  assert.equal(q.get("provider"), "AWS");
  assert.equal(q.get("period"), "all");
  assert.equal(q.get("q"), "test");
  assert.equal(q.get("group"), id(1));
  assert.equal(q.get("row_after"), "25");
  assert.equal(q.has("view"), false);
  assert.equal(q.has("lang"), false);
});
test("valid complete evidence does not infer savings or equal cost meanings", () => {
  const d = fixture();
  assert.equal(valid(d, base()), true);
  assert.notEqual(
    d.groups[0].billed_cost.coefficient,
    d.groups[0].effective_cost.coefficient,
  );
  assert.equal(Object.hasOwn(d, "savings"), false);
});
test("source selection, currency totals and enclosing versions reject inconsistent pages", () => {
  for (const mutate of [
    (d: Data) => {
      d.groups[0].provider = "Azure";
    },
    (d: Data) => {
      d.selection.account = id(6);
    },
    (d: Data) => {
      d.groups[0].versions = ["1.3"];
    },
    (d: Data) => {
      d.identified_amounts[0].currency = "EUR";
    },
    (d: Data) => {
      d.groups[0].billed_cost.coefficient = "1";
    },
    (d: Data) => {
      d.identified_amounts[0].effective_cost.present = 0;
    },
    (d: Data) => {
      d.groups.push(structuredClone(d.groups[0]));
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, base()), false);
  }
});
test("explicit timezone and real calendar timestamps are required", () => {
  for (const stamp of [
    "2024-09-01T00:00:00",
    "2024-02-30T00:00:00Z",
    "2024-13-01T00:00:00Z",
    "2024-09-01T24:00:00Z",
    "2024-09-01T00:60:00Z",
    "2024-09-01T00:00:00+25:00",
  ]) {
    const d = fixture();
    d.observed_at = stamp;
    assert.equal(valid(d, base()), false, stamp);
  }
  const d = fixture();
  d.observed_at = "2024-02-29T23:59:59.123456+09:00";
  assert.equal(valid(d, base()), true);
});
test("selected detail is pinned, current-page, same group/account/currency/version and bounded", () => {
  const d = fixture();
  d.selected = {
    group: id(1),
    rows: [detail(d)],
    after: 0,
    next_after: null,
    total_rows: 1,
  };
  const q = base();
  q.set("group", id(1));
  assert.equal(valid(d, q), false);
  q.set("snapshot", d.snapshot);
  assert.equal(valid(d, q), true);
  for (const patch of [
    { group: id(8) },
    { account: id(8) },
    { currency: "EUR" },
    { version: "1.3" },
    { billed_cost: "1".repeat(39) },
    { ordinal: "0" },
  ]) {
    const x = structuredClone(d);
    Object.assign(x.selected!.rows[0], patch);
    assert.equal(valid(x, q), false);
  }
  const wrong = structuredClone(d);
  wrong.snapshot = id(30);
  assert.equal(valid(wrong, q), false);
  const extra = structuredClone(d);
  (extra.selected!.rows[0].fields as any).private_secret = {
    state: "value",
    value: "must-reject",
  };
  assert.equal(valid(extra, q), false);
});
test("null and known zero remain distinct", () => {
  const d = fixture(),
    g = d.groups[0];
  g.billed_cost.coefficient = "0";
  g.effective_cost = { coefficient: null, present: 0, missing: 1 };
  g.statuses.used = {
    rows: 1,
    billed_cost: { ...g.billed_cost },
    effective_cost: { ...g.effective_cost },
  };
  d.identified_amounts = [
    {
      currency: "USD",
      rows: 1,
      billed_cost: { ...g.billed_cost },
      effective_cost: { ...g.effective_cost },
    },
  ];
  assert.equal(valid(d, base()), true);
  g.billed_cost.coefficient = null;
  assert.equal(valid(d, base()), false);
});
test("page subsets cannot exceed full rows and full pages must exactly reconcile source measures", () => {
  const d = fixture();
  d.group_count = 26;
  d.identified_rows = 26;
  d.row_count = 26;
  d.authorized_rows = 26;
  d.identified_amounts[0].rows = 26;
  d.identified_amounts[0].billed_cost.present = 26;
  d.identified_amounts[0].effective_cost.present = 26;
  d.groups = Array.from({ length: 25 }, (_, i) => ({
    ...structuredClone(d.groups[0]),
    id: id(100 + i),
    source_id: "synthetic-sp-" + i,
  }));
  d.next_after = 25;
  assert.equal(valid(d, base()), true);
  d.identified_amounts[0].rows = 24;
  d.identified_amounts[0].billed_cost.present = 24;
  d.identified_amounts[0].effective_cost.present = 24;
  assert.equal(valid(d, base()), false);
});
test("empty source selection is distinct from an unmatched account search", () => {
  const d = fixture();
  d.selection.q = "no-match";
  d.accounts = [];
  d.account_matches = 0;
  const q = base();
  q.set("q", "no-match");
  assert.equal(valid(d, q), true);
  assert.equal(d.group_count, 1);
  const empty = fixture();
  Object.assign(empty, {
    row_count: 0,
    identified_rows: 0,
    unidentified_rows: 0,
    group_count: 0,
    groups: [],
    identified_amounts: [],
    accounts: [],
    account_count: 0,
    account_matches: 0,
  });
  assert.equal(valid(empty, base()), true);
});
test("unknown field states and impossible metadata or status partitions reject", () => {
  for (const modify of [
    (d: Data) => {
      d.groups[0].metadata.type = {
        values: ["A"],
        distinct: 0,
        truncated: false,
      };
    },
    (d: Data) => {
      d.groups[0].source_id = "NULL";
    },
    (d: Data) => {
      d.groups[0].statuses.unused.rows = 1;
    },
    (d: Data) => {
      d.groups[0].billed_cost.coefficient = "1".repeat(51);
    },
    (d: Data) => {
      d.groups[0].mapping.version_mapped_rows = 2;
    },
  ]) {
    const d = fixture();
    modify(d);
    assert.equal(valid(d, base()), false);
  }
});
test("matched echo still requires account membership and complete cost equality", () => {
  const d = fixture(),
    q = base();
  d.selection.account = id(90);
  q.set("account", id(90));
  assert.equal(valid(d, q), false);
  const wrong = fixture();
  wrong.groups[0].billed_cost.coefficient = "1";
  wrong.groups[0].statuses.used.billed_cost.coefficient = "1";
  assert.equal(valid(wrong, base()), false);
  const duplicate = fixture(),
    second = structuredClone(duplicate.groups[0]);
  second.id = id(44);
  duplicate.groups.push(second);
  Object.assign(duplicate, {
    authorized_rows: 2,
    row_count: 2,
    identified_rows: 2,
    group_count: 2,
  });
  duplicate.identified_amounts[0] = {
    currency: "USD",
    rows: 2,
    billed_cost: {
      coefficient: "200000000000000000000",
      present: 2,
      missing: 0,
    },
    effective_cost: {
      coefficient: "160000000000000000000",
      present: 2,
      missing: 0,
    },
  };
  assert.equal(valid(duplicate, base()), false);
});
