import { test } from "node:test";
import assert from "node:assert/strict";
import { validConnection, actionUrl, type Connection } from "./api";
const id = "00000000-0000-0000-0000-000000000001",
  sha = "a".repeat(64);
const data: Connection = {
  classification: "local-connection-metadata",
  connection: id,
  workspace: id,
  account: "synthetic",
  subaccount: null,
  period_start: "2026-10-01T00:00:00+00:00",
  period_end: "2026-11-01T00:00:00+00:00",
  policy_revision: "1",
  observed_at: "2026-11-03T00:00:00+00:00",
  contracts: [{ version: "1.3", revision: "1", digest: sha }],
  max_bytes: 8388608,
  max_rows: 10000,
  completeness: "unknown",
  provider_authentication: "unconnected",
  provider_reads: "unconnected",
  invoices: "unconnected",
  holds: { quarantine: false, unresolved: false },
  actions: ["imports"],
  latest_candidate: null,
};
test("response requires exact identity and safe role route combinations", () => {
  assert.ok(validConnection(data, id));
  assert.equal(
    validConnection(data, "00000000-0000-0000-0000-000000000002"),
    false,
  );
  assert.equal(
    validConnection({ ...data, actions: ["imports", "source-data"] }, id),
    false,
  );
  assert.equal(
    validConnection({ ...data, holds: undefined } as unknown as Connection, id),
    false,
  );
  assert.equal(
    validConnection(
      { ...data, contracts: [data.contracts[0], data.contracts[0]] },
      id,
    ),
    false,
  );
  assert.equal(
    validConnection({ ...data, completeness: "complete" }, id),
    false,
  );
});
test("handoff preserves exact release and manifest without granting rights", () => {
  const sample = {
    ...data,
    actions: ["source-data", "reconciliation"] as Connection["actions"],
    latest_candidate: {
      release_id: id,
      source_revision: "2",
      state: "verified",
      created_at: "2026-11-03T00:00:00+00:00",
      manifest_digest: sha,
      version: "1.3",
    },
  };
  assert.ok(validConnection(sample));
  assert.equal(
    actionUrl("source-data", sample, "ar"),
    `#/source-data/${id}?lang=ar&service=local&snapshot=${sha}`,
  );
  assert.equal(
    actionUrl("imports", data, "en"),
    "#/imports?lang=en&service=local",
  );
});

test("formatted timestamps reject invalid or timezone-free values", () => {
  for (const field of ["period_start", "period_end", "observed_at"]) {
    for (const value of [
      "invalid",
      "2026-11-03",
      "2026-11-03T00:00:00",
      "2026-99-03T00:00:00Z",
    ]) {
      assert.equal(validConnection({ ...data, [field]: value }), false);
    }
  }
  assert.equal(
    validConnection({
      ...data,
      latest_candidate: {
        release_id: id,
        source_revision: "1",
        state: "verified",
        created_at: "invalid",
        manifest_digest: sha,
        version: "1.3",
      },
    }),
    false,
  );
});
