import { test } from "node:test";
import assert from "node:assert/strict";
import { context, validEvidence, sourceReturn, type Evidence } from "./api";
const release = "00000000-0000-0000-0000-000000000001",
  snapshot = "a".repeat(64);
const c = { release, snapshot, row: "1", after: "0" };
const fields = [
  "billing_account_id",
  "billing_currency",
  "billed_cost",
  "effective_cost",
  "charge_period_start",
  "charge_period_end",
].map((name) => ({
  name,
  source_name: "Field",
  source_present: true,
  source_text: "",
  normalized: name.endsWith("_cost") ? "999999999999.999999" : null,
}));
const sample = {
  release_id: release,
  snapshot,
  ordinal: "1",
  row_count: 2,
  source_revision: "1",
  version: "1.3",
  registry_digest: snapshot,
  original_digest: snapshot,
  original_bytes: 256,
  method: "pinned-original-record-extract-v1",
  classification: "local-source-evidence",
  warehouse_integrity: "not-revalidated",
  retention: "unknown",
  public_document: "unconnected",
  observed_at: "2026-11-03",
  scope: {
    workspace: release,
    account: "synthetic",
    period_start: "2026-10-01",
    period_end: "2026-11-01",
  },
  fields,
} as Evidence;
test("context rejects malformed or unrelated pages while accepting final record", () => {
  assert.equal(context(undefined, new URLSearchParams()), null);
  for (const params of [
    { row: "1" },
    { snapshot, row: "0", after: "0" },
    { snapshot, row: "51", after: "0" },
    { snapshot, row: "1", after: "1" },
  ])
    assert.throws(() =>
      context(release, new URLSearchParams(params as Record<string, string>)),
    );
  assert.equal(
    context(
      release,
      new URLSearchParams({ snapshot, row: "10000", after: "9950" }),
    )?.row,
    "10000",
  );
  const url = sourceReturn(c, "ar");
  assert.ok(url.includes("focus=evidence"));
  assert.ok(url.includes("row=1"));
  assert.ok(!url.includes("originals"));
});
test("evidence validates identity, null/source distinctions and exact decimals", () => {
  assert.ok(validEvidence(sample, c));
  assert.equal(
    validEvidence({ ...sample, snapshot: "b".repeat(64) }, c),
    false,
  );
  assert.equal(
    validEvidence({ ...sample, scope: undefined } as unknown as Evidence, c),
    false,
  );
  assert.equal(
    validEvidence(
      {
        ...sample,
        fields: fields.map((f) => ({
          ...f,
          source_present: false,
          source_text: "hidden",
        })),
      },
      c,
    ),
    false,
  );
  assert.equal(
    validEvidence(
      { ...sample, fields: [...fields, { ...fields[0], name: "tags" }] },
      c,
    ),
    false,
  );
  assert.equal(
    validEvidence(
      {
        ...sample,
        fields: fields.map((f) =>
          f.name === "billed_cost" ? { ...f, normalized: "1e12" } : f,
        ),
      },
      c,
    ),
    false,
  );
  assert.ok(
    validEvidence(
      {
        ...sample,
        fields: fields.map((f) => ({
          ...f,
          source_text: "<img src=https://invalid.test>\n=1+1",
        })),
      },
      c,
    ),
  );
});
