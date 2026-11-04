import { test } from "node:test";
import assert from "node:assert/strict";
import {
  validOverview,
  type OverviewData,
} from "../../../sources/collected-focus/api";
function sample(): OverviewData {
  const measure = {
    coefficient: "1000000000000000001",
    present: 1,
    missing: 0,
  };
  const amounts = [
    { currency: "USD", rows: 1, billed_cost: measure, effective_cost: measure },
  ];
  return {
    schema: 1,
    scale: 18,
    snapshot: "a".repeat(64),
    source_snapshot: "b".repeat(64),
    selection: {
      period: "all",
      provider: "",
      account: "",
      basis: "billed_cost",
      q: "",
    },
    classification: "collected-focus-sample-query-only",
    financial_authority: false,
    observed_at: "2024-04-01T00:00:00Z",
    period_basis: "charge-start-in-tenant-timezone-no-proration",
    company: "sample",
    company_mapping: "synthetic-sample-authorization",
    tenants: [{ key: "sample", timezone: "UTC" }],
    versions: ["1.0"],
    delivery_count: 1,
    coverage: {
      slices: 1,
      finalized_slices: 1,
      complete_through: null,
      expectations: "operator-asserted-unverified",
    },
    provenance: { "synthetic-contract-fixture": 1 },
    options: {
      providers: ["Example"],
      periods: ["2024-03"],
      currencies: ["USD"],
    },
    authorized_rows: 1,
    row_count: 1,
    amounts,
    providers: [{ provider: "Example", amounts: structuredClone(amounts) }],
    months: [{ period: "2024-03", amounts }],
    account_count: 1,
    account_matches: 1,
    accounts: [
      {
        id: "c".repeat(64),
        tenant: "sample",
        provider: "Example",
        billing_account: "sample-account",
        sub_account: null,
        native_ancestry: "unverified",
        rows: 1,
        amounts,
      },
    ],
    next_account_after: null,
    rows: [
      {
        id: "d".repeat(64),
        consumed_unit: "Hours",
        pricing_unit: "Requests",
        consumed_quantity: "1",
        pricing_quantity: null,
        list_cost: "0",
        contracted_cost: null,
        list_unit_price: "-1",
        contracted_unit_price: null,
        billing_start: "2024-03-01T00:00:00Z",
        billing_end: "2024-04-01T00:00:00Z",
        account: "c".repeat(64),
        provider: "Example",
        billing_account: "sample-account",
        sub_account: null,
        service: null,
        currency: "USD",
        charge_month: "2024-03",
        charge_start: "2024-03-01T00:00:00Z",
        charge_end: "2024-04-01T00:00:00Z",
        billed_cost: measure.coefficient,
        effective_cost: measure.coefficient,
        evidence_class: "synthetic-contract-fixture",
        batch: "sample-batch",
        ordinal: "0",
      },
    ],
    next_after: null,
    after: 0,
    account_after: 0,
  };
}
test("accepts explicit precise bounded collected contract", () =>
  assert.equal(validOverview(sample(), new URLSearchParams()), true));
test("rejects incorrect meanings, provider arithmetic, null precision and forged selection", () => {
  const mutations: ((d: OverviewData) => void)[] = [
    (d) => {
      d.scale = 2 as 18;
    },
    (d) => {
      d.financial_authority = true as false;
    },
    (d) => {
      d.providers[0].amounts[0].billed_cost.coefficient = "0";
    },
    (d) => {
      d.amounts[0].billed_cost.coefficient = null;
    },
    (d) => {
      d.selection.provider = "foreign";
    },
    (d) => {
      d.rows[0].billed_cost = "1.5";
    },
    (d) => {
      d.next_after = 25;
    },
    (d) => {
      d.rows = Array(26).fill(d.rows[0]);
    },
  ];
  mutations.push(
    (d) => {
      d.rows[0].consumed_quantity = "1.5";
    },
    (d) => {
      d.rows[0].list_unit_price = 1 as unknown as string;
    },
    (d) => {
      delete (d.rows[0] as Partial<(typeof d.rows)[0]>).billing_start;
    },
  );
  for (const mutate of mutations) {
    const d = sample();
    mutate(d);
    assert.equal(validOverview(d, new URLSearchParams()), false);
  }
  assert.equal(
    validOverview(sample(), new URLSearchParams({ snapshot: "e".repeat(64) })),
    false,
  );
});
