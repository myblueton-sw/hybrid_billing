import test from "node:test";
import assert from "node:assert/strict";
import { valid, query, profile, type Data, type Charge } from "./api";
const h = (v: string) => v.repeat(64);
export function fixture(): Data {
  const row: Charge = {
    id: h("1"),
    tenant: "north-a",
    account: h("2"),
    provider: "AWS",
    billing_account: "acct",
    sub_account: null,
    currency: "USD",
    service: "Software",
    billed_cost: "342000000000000000",
    effective_cost: "0",
    charge_start: "2024-09-10T02:00:00+00:00",
    charge_end: "2024-09-10T03:00:00+00:00",
    billing_start: "2024-09-01T00:00:00+00:00",
    billing_end: "2024-10-01T00:00:00+00:00",
    version: "1.0",
    batch: "batch",
    ordinal: "444",
    evidence_class: "official-public",
    fields: {
      publisher: { state: "value", value: "Publisher" },
      issuer: { state: "value", value: "Issuer" },
      description: { state: "value", value: "Software" },
      category: { state: "value", value: "Usage" },
      charge_class: { state: "source_null", value: null },
      frequency: { state: "value", value: "Usage-Based" },
      sku: { state: "value", value: "sku" },
      commitment: { state: "absent", value: null },
    },
  };
  return {
    schema: 1,
    scale: 18,
    profile,
    classification: "collected-marketplace-evidence-only",
    financial_authority: false,
    snapshot: h("a"),
    source_snapshot: h("b"),
    mapping_digest: h("c"),
    company: "north",
    versions: ["1.0"],
    observed_at: "2026-11-05T01:00:00Z",
    selection: { provider: "", period: "all", account: "", q: "" },
    period_basis: "charge-start-in-tenant-timezone-no-proration",
    coverage: {
      slices: 6,
      finalized_slices: 6,
      complete_through: "2024-08-01T00:00:00Z",
      expectations: "operator-asserted-unverified",
    },
    options: { providers: ["AWS"], periods: ["2024-09"], currencies: ["USD"] },
    authorized_rows: 3,
    row_count: 3,
    counts: { candidate: 1, no_difference: 1, unknown: 1 },
    candidate_amounts: [
      {
        currency: "USD",
        rows: 1,
        billed_cost: {
          coefficient: "342000000000000000",
          present: 1,
          missing: 0,
        },
        effective_cost: { coefficient: "0", present: 1, missing: 0 },
      },
    ],
    unknown_amounts: [
      {
        currency: "USD",
        rows: 1,
        billed_cost: { coefficient: null, present: 0, missing: 1 },
        effective_cost: { coefficient: null, present: 0, missing: 1 },
      },
    ],
    rows: [row],
    after: 0,
    next_after: null,
    accounts: [
      {
        id: h("2"),
        tenant: "north-a",
        provider: "AWS",
        billing_account: "acct",
        sub_account: null,
        rows: 3,
      },
    ],
    account_count: 1,
    account_matches: 1,
    account_after: 0,
    next_account_after: null,
    selected: null,
  };
}
const q = () => query(new URLSearchParams());
test("accepts exact source cost and observed zero separately from unknown", () => {
  const d = fixture();
  assert.equal(valid(d, q()), true);
  assert.equal(d.candidate_amounts[0].effective_cost.coefficient, "0");
  assert.equal(d.unknown_amounts[0].effective_cost.coefficient, null);
});
test("normalizes account search while preserving identity and canonical defaults", () => {
  const a = query(
    new URLSearchParams(
      "focus-q=%20acct%20%20north%20&marketplace-row=" + h("1"),
    ),
  );
  assert.equal(a.get("q"), "acct north");
  assert.equal(a.get("provider"), "");
  assert.equal(a.get("period"), "all");
  assert.equal(a.get("row"), h("1"));
  assert.equal(a.has("tenant"), false);
});
test("rejects false authority, profile, count partitions and scope", () => {
  for (const mutate of [
    (d: Data) => {
      d.financial_authority = true as false;
    },
    (d: Data) => {
      d.profile = "other" as typeof profile;
    },
    (d: Data) => {
      d.counts.unknown = 0;
    },
    (d: Data) => {
      d.authorized_rows = 2;
    },
    (d: Data) => {
      d.selection.provider = "AWS";
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, q()), false);
  }
});
test("requires observed mapped different publisher/issuer and no null identifiers", () => {
  for (const mutate of [
    (r: Charge) => {
      r.fields.publisher.value = r.fields.issuer.value;
    },
    (r: Charge) => {
      r.fields.issuer = { state: "source_null", value: null };
    },
    (r: Charge) => {
      r.fields.publisher.value = "NULL";
    },
    (r: Charge) => {
      r.version = "1.2";
    },
    (r: Charge) => {
      r.fields.sku = { state: "absent", value: "fake" };
    },
  ]) {
    const d = fixture();
    mutate(d.rows[0]);
    assert.equal(valid(d, q()), false);
  }
});
test("rejects absent provider/currency membership, scope leaks and interval errors", () => {
  for (const mutate of [
    (d: Data) => {
      d.rows[0].provider = "Foreign";
    },
    (d: Data) => {
      d.rows[0].currency = "EUR";
    },
    (d: Data) => {
      d.rows[0].charge_start = "2024-02-30T02:00:00Z";
    },
    (d: Data) => {
      d.rows[0].charge_start = "2024-09-10T02:00:00";
    },
    (d: Data) => {
      d.rows[0].charge_end = d.rows[0].charge_start;
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, q()), false);
  }
});
test("rejects incorrect full-selection sums, missing counts and duplicate rows", () => {
  for (const mutate of [
    (d: Data) => {
      d.candidate_amounts[0].billed_cost.coefficient = "0";
    },
    (d: Data) => {
      d.candidate_amounts[0].effective_cost.coefficient = null;
    },
    (d: Data) => {
      d.unknown_amounts[0].rows = 0;
    },
    (d: Data) => {
      d.rows.push(d.rows[0]);
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, q()), false);
  }
});
test("binds selected details to the exact visible candidate and pinned snapshot", () => {
  const d = fixture();
  d.selected = structuredClone(d.rows[0]);
  const p = q();
  p.set("row", d.selected.id);
  p.set("snapshot", d.snapshot);
  assert.equal(valid(d, p), true);
  d.selected.fields.description.value = "foreign";
  assert.equal(valid(d, p), false);
  d.selected = structuredClone(d.rows[0]);
  p.set("snapshot", h("f"));
  assert.equal(valid(d, p), false);
  p.delete("row");
  p.set("snapshot", d.snapshot);
  assert.equal(valid(d, p), false);
});
test("rejects account-page corruption and amounts over precision bounds", () => {
  for (const mutate of [
    (d: Data) => {
      d.accounts[0].rows = 4;
    },
    (d: Data) => {
      d.accounts[0].provider = "foreign";
    },
    (d: Data) => {
      d.next_account_after = 25;
    },
    (d: Data) => {
      d.rows[0].billed_cost = "1" + "0".repeat(38);
    },
    (d: Data) => {
      d.candidate_amounts[0].billed_cost.coefficient = "1" + "0".repeat(50);
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, q()), false);
  }
});
