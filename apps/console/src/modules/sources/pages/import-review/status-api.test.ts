import test from "node:test";
import assert from "node:assert/strict";
import { loadStatus, parseStatus, type Scope } from "./status-api";
import { ReadError } from "./api";
const scope: Scope = {
  audience: "company",
  member_count: 2,
  auth_scope: "a".repeat(64),
  permission_revision: "b".repeat(64),
};
const valid = () => ({
  schema: 1,
  profile: "retained-focus-collection-status-v1",
  read_only: true,
  financial_authority: false,
  ...scope,
  snapshot: "c".repeat(64),
  observed_at: "2026-11-10T00:00:00+00:00",
  row_count: 3,
  delivery_count: 2,
  versions: ["1.0"],
  provenance: {
    "official-public": 2,
    "official-public-subset": 0,
    "synthetic-contract-fixture": 1,
    unclassified: 0,
  },
  providers: [{ provider: "AWS", row_count: 3 }],
  periods: [
    {
      window_start: "2024-09-01T00:00:00+00:00",
      window_end: "2024-10-01T00:00:00+00:00",
      row_count: 3,
    },
  ],
  upstream_completeness: "unverified",
  source_classification: "exploratory-query-only",
  original_availability: "not-checked",
});
const response = (data: unknown, status = 200) =>
  new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  });
test("current permitted-row aggregates are separate from whole-file receipts", () => {
  const d = parseStatus(valid(), scope);
  assert.equal(d.row_count, 3);
  assert.equal(d.delivery_count, 2);
  assert.equal(d.upstream_completeness, "unverified");
});
test("closed DTO rejects private identities, unsafe snapshots and financial authority", () => {
  for (const patch of [
    { source_snapshot: "d".repeat(64) },
    { batch_id: "private" },
    { source_json: "private" },
    { financial_authority: true },
    { upstream_completeness: "complete" },
    { original_availability: "verified" },
  ])
    assert.throws(
      () => parseStatus({ ...valid(), ...patch }, scope),
      ReadError,
    );
});
test("provider, provenance, period counts and contributing versions obey exact invariants", () => {
  for (const patch of [
    { row_count: 4 },
    { row_count: 3.1 },
    { delivery_count: 0 },
    { delivery_count: 4 },
    { versions: [] },
    { versions: ["1.0", "1.0"] },
    { versions: ["1.0", "1.2", "1.3"] },
    { providers: [{ provider: "AWS", row_count: 2 }] },
    {
      providers: [
        { provider: "AWS", row_count: 1 },
        { provider: "AWS", row_count: 2 },
      ],
    },
    { periods: [] },
    { provenance: { ...valid().provenance, unclassified: 1 } },
  ])
    assert.throws(
      () => parseStatus({ ...valid(), ...patch }, scope),
      ReadError,
    );
  const duplicate = valid();
  duplicate.periods = [...duplicate.periods, { ...duplicate.periods[0] }];
  assert.throws(() => parseStatus(duplicate, scope), ReadError);
});
test("zero rows require no contributed delivery, version, provider or interval", () => {
  const zero = {
    ...valid(),
    row_count: 0,
    delivery_count: 0,
    versions: [],
    providers: [],
    periods: [],
    provenance: {
      "official-public": 0,
      "official-public-subset": 0,
      "synthetic-contract-fixture": 0,
      unclassified: 0,
    },
  };
  assert.equal(parseStatus(zero, scope).row_count, 0);
  assert.throws(
    () => parseStatus({ ...zero, delivery_count: 1 }, scope),
    ReadError,
  );
  assert.throws(
    () => parseStatus({ ...zero, versions: ["1.0"] }, scope),
    ReadError,
  );
});
test("source windows require valid UTC dates and positive exact row counts", () => {
  for (const patch of [
    { window_start: "2024-02-30T00:00:00Z" },
    { window_end: "2024-09-01T00:00:00Z" },
    { row_count: 0 },
    { row_count: "3" },
    { window_end: "2024-10-01T00:00:00+09:00" },
    { batch_id: "hidden" },
  ])
    assert.throws(
      () =>
        parseStatus(
          { ...valid(), periods: [{ ...valid().periods[0], ...patch }] },
          scope,
        ),
      ReadError,
    );
});
test("all scope-pin disagreements signal authority invalidation", () => {
  for (const patch of [
    { auth_scope: "d".repeat(64) },
    { permission_revision: "d".repeat(64) },
    { member_count: 1 },
    { audience: "tenant" },
  ])
    assert.throws(
      () => parseStatus({ ...valid(), ...patch }, scope),
      (e: unknown) =>
        e instanceof ReadError &&
        e.httpStatus === 403 &&
        e.code === "permission_changed",
    );
});
test("loader is a bounded read with fixed endpoint and no client-supplied scope", async () => {
  const previous = globalThis.fetch;
  try {
    globalThis.fetch = async (url, init) => {
      assert.equal(url, "/api/focus-collection-status");
      assert.equal(init?.method, "GET");
      assert.equal(init?.cache, "no-store");
      assert.equal(init?.redirect, "error");
      assert.equal(init?.body, undefined);
      assert.equal(new Headers(init?.headers).get("X-HB-Local-Client"), "1");
      return response(valid());
    };
    assert.equal(
      (await loadStatus(scope, new AbortController().signal)).row_count,
      3,
    );
  } finally {
    globalThis.fetch = previous;
  }
});
test("HTTP403 invalidates authority even if its error body is malformed", async () => {
  const previous = globalThis.fetch;
  try {
    for (const data of [
      { error: "scope_denied", request_id: "a".repeat(32) },
      { invalid: true },
    ]) {
      globalThis.fetch = async () => response(data, 403);
      await assert.rejects(
        loadStatus(scope, new AbortController().signal),
        (e: unknown) => e instanceof ReadError && e.httpStatus === 403,
      );
    }
  } finally {
    globalThis.fetch = previous;
  }
});
test("availability failure, invalid JSON, duplicate fields and oversize are never zero", async () => {
  const previous = globalThis.fetch;
  try {
    for (const r of [
      response(
        { error: "collected_unavailable", request_id: "a".repeat(32) },
        503,
      ),
      new Response('{"schema":1,"schema":1}', {
        headers: { "Content-Type": "application/json" },
      }),
      new Response("{}", { headers: { "Content-Type": "text/html" } }),
      new Response("{}", {
        headers: {
          "Content-Type": "application/json",
          "Content-Length": "2097153",
        },
      }),
    ]) {
      globalThis.fetch = async () => r;
      await assert.rejects(
        loadStatus(scope, new AbortController().signal),
        ReadError,
      );
    }
  } finally {
    globalThis.fetch = previous;
  }
});
test("scope is captured before async work and cancelled old responses are rejected", async () => {
  const previous = globalThis.fetch;
  try {
    let finish!: (value: Response) => void;
    globalThis.fetch = () =>
      new Promise((resolve) => {
        finish = resolve;
      });
    const mutable = { ...scope };
    const pending = loadStatus(mutable, new AbortController().signal);
    mutable.auth_scope = "e".repeat(64);
    finish(response(valid()));
    assert.equal((await pending).auth_scope, scope.auth_scope);
    const c = new AbortController();
    const cancelled = loadStatus(scope, c.signal);
    c.abort();
    finish(response(valid()));
    await assert.rejects(
      cancelled,
      (e: unknown) => e instanceof DOMException && e.name === "AbortError",
    );
  } finally {
    globalThis.fetch = previous;
  }
});

test("versions, providers and source windows require canonical producer ordering", () => {
  assert.throws(
    () => parseStatus({ ...valid(), versions: ["1.2", "1.0"] }, scope),
    ReadError,
  );
  assert.throws(
    () =>
      parseStatus(
        {
          ...valid(),
          providers: [
            { provider: "Google Cloud", row_count: 1 },
            { provider: "AWS", row_count: 2 },
          ],
        },
        scope,
      ),
    ReadError,
  );
  const later = valid().periods[0],
    earlier = {
      window_start: "2024-03-01T00:00:00Z",
      window_end: "2024-04-01T00:00:00Z",
      row_count: 1,
    };
  assert.throws(
    () =>
      parseStatus(
        { ...valid(), periods: [{ ...later, row_count: 2 }, earlier] },
        scope,
      ),
    ReadError,
  );
  const alias = {
    ...later,
    window_start: later.window_start.replace("+00:00", "Z"),
    window_end: later.window_end.replace("+00:00", "Z"),
    row_count: 1,
  };
  assert.throws(
    () =>
      parseStatus(
        { ...valid(), periods: [{ ...later, row_count: 2 }, alias] },
        scope,
      ),
    ReadError,
  );
  assert.equal(
    parseStatus(
      {
        ...valid(),
        providers: [
          { provider: "\uE000", row_count: 1 },
          { provider: "😀", row_count: 2 },
        ],
      },
      scope,
    ).providers.length,
    2,
  );
});
