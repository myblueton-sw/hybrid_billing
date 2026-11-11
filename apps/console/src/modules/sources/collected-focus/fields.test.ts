import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import type { FocusRow } from "./api";
import {
  chargeMonthBasis,
  chargeStartsInBillingPeriod,
  instantParts,
  mappedFields,
  mappingVersions,
  missingFields,
  mappedColumns,
  derivedFields,
  lineageFields,
  annotationFields,
  provenanceClass,
  rowVersion,
  utcMonth,
  valueState,
} from "./fields";
const contract = JSON.parse(
  readFileSync(
    new URL(
      "../../../../../../packages/contracts/focus_import/versions.json",
      import.meta.url,
    ),
    "utf8",
  ),
).versions as Record<
  string,
  {
    null_tokens: string[];
    source_timezone: string;
    columns: Record<string, string>;
  }
>;
const contractName: Record<string, string> = {
  billing_account: "account",
  sub_account: "subaccount",
};
const row: FocusRow = {
  id: "a".repeat(64),
  account: "b".repeat(64),
  provider: "Oracle",
  billing_account: "sample-billing",
  sub_account: "sample-sub",
  service: "COMPUTE",
  currency: "USD",
  charge_month: "2024-09",
  consumed_unit: "OCPU Hours",
  pricing_unit: "OCPU Hours",
  billing_start: "2024-09-01T00:00:00+00:00",
  billing_end: "2024-10-01T00:00:00+00:00",
  charge_start: "2024-09-30T22:00:00+00:00",
  charge_end: "2024-09-30T23:00:00+00:00",
  billed_cost: "240000000000000000",
  effective_cost: "0",
  list_cost: "240000000000000000",
  contracted_cost: null,
  consumed_quantity: "8000000000000000000",
  pricing_quantity: "8000000000000000000",
  list_unit_price: "30000000000000000",
  contracted_unit_price: null,
  evidence_class: "official-public",
  batch: "00000000-0000-4000-8000-000000000000",
  ordinal: "1",
};
test("client column mirror equals the tracked mapping contract for every version", () => {
  assert.deepEqual(
    Object.keys(mappingVersions).sort(),
    Object.keys(contract).sort(),
  );
  for (const [version, entry] of Object.entries(contract)) {
    assert.equal(entry.source_timezone, "UTC");
    assert.equal(Object.keys(entry.columns).length, mappedFields.length);
    for (const field of mappedFields)
      assert.equal(
        mappingVersions[version][field],
        entry.columns[contractName[field] ?? field],
        `${version} ${field}`,
      );
  }
});
test("source column names follow the versions in scope and unknown versions are reported, not guessed", () => {
  assert.deepEqual(mappedColumns("provider", ["1.0"]).names, ["ProviderName"]);
  assert.deepEqual(mappedColumns("provider", ["1.0", "1.2"]).names, [
    "ProviderName",
  ]);
  assert.deepEqual(mappedColumns("provider", ["1.0", "1.3"]), {
    mapped: [
      { version: "1.0", name: "ProviderName" },
      { version: "1.3", name: "ServiceProviderName" },
    ],
    names: ["ProviderName", "ServiceProviderName"],
    unmapped: [],
  });
  assert.deepEqual(mappedColumns("billed_cost", ["1.0", "9.9"]), {
    mapped: [{ version: "1.0", name: "BilledCost" }],
    names: ["BilledCost"],
    unmapped: ["9.9"],
  });
  assert.deepEqual(mappedColumns("billed_cost", ["9.9"]).names, []);
  assert.deepEqual(mappedColumns("billed_cost", []), {
    mapped: [],
    names: [],
    unmapped: [],
  });
});
test("every row key belongs to exactly one origin list", () => {
  const classified: string[] = [
    ...mappedFields,
    ...derivedFields,
    ...lineageFields,
    ...annotationFields,
  ];
  assert.equal(new Set(classified).size, classified.length);
  assert.deepEqual([...classified].sort(), Object.keys(row).sort());
  assert.equal(mappedFields.length, 19);
});
test("missing, exact zero and values are distinct states", () => {
  assert.equal(valueState(null), "missing");
  assert.equal(valueState("0"), "zero");
  assert.equal(valueState("-0"), "zero");
  assert.equal(valueState("1"), "value");
  assert.equal(valueState("-1"), "value");
  assert.equal(valueState("-240000000000000000"), "value");
  assert.equal(valueState("123456789012345678901234567890123"), "value");
});
test("missing fields list only null values and never exact zero", () => {
  assert.deepEqual(missingFields(row), [
    "contracted_cost",
    "contracted_unit_price",
  ]);
  assert.deepEqual(
    missingFields({
      ...row,
      sub_account: null,
      consumed_unit: null,
      consumed_quantity: null,
      contracted_cost: "0",
      contracted_unit_price: "0",
    }),
    ["sub_account", "consumed_unit", "consumed_quantity"],
  );
});
test("UTC instants split without losing source precision and other spellings are rejected", () => {
  assert.deepEqual(instantParts("2024-09-30T22:00:00+00:00"), {
    date: "2024-09-30",
    time: "22:00:00",
  });
  assert.deepEqual(instantParts("2024-09-30T22:00:00.120000+00:00"), {
    date: "2024-09-30",
    time: "22:00:00.12",
  });
  assert.deepEqual(instantParts("2024-09-30T22:00:00.000001+00:00"), {
    date: "2024-09-30",
    time: "22:00:00.000001",
  });
  assert.equal(instantParts("2024-09-30T22:00:00+09:00"), null);
  assert.equal(instantParts("2024-09-30T22:00:00Z"), null);
  assert.equal(instantParts("2024-09-30"), null);
  assert.equal(utcMonth("2024-09-30T22:00:00+00:00"), "2024-09");
  assert.equal(utcMonth("not-a-time"), null);
});
test("charge-start month basis states a row time zone only when the scope has one", () => {
  const seoul = { timezone: "Asia/Seoul" },
    utc = { timezone: "UTC" };
  assert.deepEqual(chargeMonthBasis(row, [utc, seoul, utc]), {
    utcMonth: "2024-09",
    differs: false,
    timezones: ["Asia/Seoul", "UTC"],
    rowTimezone: null,
  });
  assert.deepEqual(
    chargeMonthBasis({ ...row, charge_month: "2024-10" }, [seoul, seoul]),
    {
      utcMonth: "2024-09",
      differs: true,
      timezones: ["Asia/Seoul"],
      rowTimezone: "Asia/Seoul",
    },
  );
  assert.deepEqual(chargeMonthBasis({ ...row, charge_start: "bad" }, []), {
    utcMonth: null,
    differs: false,
    timezones: [],
    rowTimezone: null,
  });
});
test("charge start is compared with the inclusive-start exclusive-end billing period", () => {
  assert.equal(chargeStartsInBillingPeriod(row), true);
  assert.equal(
    chargeStartsInBillingPeriod({
      ...row,
      charge_start: "2024-09-01T00:00:00+00:00",
    }),
    true,
  );
  assert.equal(
    chargeStartsInBillingPeriod({
      ...row,
      charge_start: "2024-10-01T00:00:00+00:00",
    }),
    false,
  );
  assert.equal(
    chargeStartsInBillingPeriod({
      ...row,
      charge_start: "2024-09-30T23:59:59.999999+00:00",
    }),
    true,
  );
  assert.equal(
    chargeStartsInBillingPeriod({
      ...row,
      charge_start: "2024-08-31T23:59:59.999999+00:00",
    }),
    false,
  );
  assert.equal(
    chargeStartsInBillingPeriod({ ...row, billing_end: "2024-10-01" }),
    null,
  );
});
test("row mapping version is stated only when the scope has exactly one version", () => {
  assert.equal(rowVersion(["1.0"]), "1.0");
  assert.equal(rowVersion(["1.0", "1.3"]), null);
  assert.equal(rowVersion([]), null);
});
test("unrecognized provenance values are shown as unclassified, never as a recorded class", () => {
  assert.equal(provenanceClass("official-public"), "official-public");
  assert.equal(
    provenanceClass("official-public-subset"),
    "official-public-subset",
  );
  assert.equal(
    provenanceClass("synthetic-contract-fixture"),
    "synthetic-contract-fixture",
  );
  assert.equal(provenanceClass("customer"), "unclassified");
  assert.equal(provenanceClass(""), "unclassified");
});
