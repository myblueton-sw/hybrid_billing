import assert from "node:assert/strict";
import test from "node:test";
import {
  parseData,
  query,
  queryString,
  type Data,
  type Query,
  type Registration,
} from "./contract";
import { load, parseJson, ReadError } from "./api";
const A = "a".repeat(64),
  B = "b".repeat(64),
  C = "c".repeat(64);
const q: Query = { view: "connections", q: "", after: 0 };
function registration(index = 0): Registration {
  return {
    id: index.toString(16).padStart(64, "0"),
    connection: "connection-" + String(index).padStart(3, "0"),
    focus_version: "1.3",
    grain: "month",
    window_start: "2026-01-01T00:00:00+00:00",
    window_end: "2026-02-01T00:00:00Z",
    registered_domains: "2",
    semantic_profile: "exploratory-query-only",
    allowed_lateness_seconds: "0",
    replacement_limit: "10",
  };
}
function envelope(): Data {
  return {
    schema: 1,
    profile: "retained-focus-connections-v1",
    read_only: true,
    financial_authority: false,
    commands: "unconnected",
    view: "connections",
    selection: { q: "", after: 0 },
    audience: "company",
    member_count: 2,
    auth_scope: A,
    permission_revision: B,
    snapshot: C,
    observed_at: "2026-11-08T01:02:03.123456Z",
    total: 1,
    next_after: null,
    capabilities: {
      connections: "fully-owned-registrations-only",
      capability_evidence: "registration-only",
      provider_authentication: "unconnected",
      diagnostics: "unconnected",
    },
    records: [registration()],
  };
}
const rejected = (data: unknown, requested = q) =>
  assert.throws(() => parseData(data, requested), /invalid_response/);
const response = (data: unknown, status = 200) =>
  new Response(JSON.stringify(data), {
    status,
    headers: { "Content-Type": "application/json" },
  });
async function withFetch(body: typeof fetch, run: () => Promise<void>) {
  const previous = globalThis.fetch;
  globalThis.fetch = body;
  try {
    await run();
  } finally {
    globalThis.fetch = previous;
  }
}
test("registration metadata preserves exact declared counts without financial or provider authority", () => {
  const data = parseData(envelope(), q);
  assert.equal(data.records[0].registered_domains, "2");
  assert.equal(data.records[0].allowed_lateness_seconds, "0");
  assert.equal(data.records[0].replacement_limit, "10");
  assert.equal(data.capabilities.provider_authentication, "unconnected");
  assert.equal(data.financial_authority, false);
});
test("confirmed empty registrations differs from unsupported authentication and diagnostic evidence", () => {
  const empty = envelope();
  empty.total = 0;
  empty.records = [];
  assert.equal(parseData(empty, q).total, 0);
  for (const view of ["security", "diagnostics"] as const) {
    const data = { ...empty, view, total: null };
    assert.equal(parseData(data, { ...q, view }).total, null);
    rejected({ ...data, total: 0 }, { ...q, view });
    rejected({ ...data, records: [registration()] }, { ...q, view });
    rejected({ ...data, next_after: 25 }, { ...q, view });
  }
  rejected({ ...empty, total: null });
});
test("closed envelope, registration and nested objects exclude private and authority extensions", () => {
  for (const field of [
    "actor",
    "tenant",
    "domains",
    "account",
    "dataset",
    "credential",
    "capabilities",
    "object_key",
    "body",
    "url",
  ]) {
    const data = envelope();
    rejected({ ...data, [field]: "private" });
    rejected({
      ...data,
      records: [{ ...data.records[0], [field]: "private" }],
    });
  }
  const data = envelope();
  rejected({ ...data, selection: { ...data.selection, tenant: "other" } });
  rejected({
    ...data,
    capabilities: { ...data.capabilities, configure: true },
  });
  rejected({ ...data, read_only: false });
  rejected({ ...data, financial_authority: true });
  rejected({ ...data, commands: "register" });
  rejected({ ...data, schema: 2 });
});
test("owned record scalars require exact canonical bounded decimal strings", () => {
  const cases: [string, unknown[]][] = [
    ["registered_domains", [0, "0", "01", "10001", "1.0", "-1", null]],
    ["allowed_lateness_seconds", [0, "00", "604801", "1e3", "-1", null]],
    ["replacement_limit", [1, "0", "11", "01", "1.5", null]],
    [
      "connection",
      ["", "UPPER", "a".repeat(65), "source/secret", "account@example.com"],
    ],
    ["focus_version", ["", "1", "1.300", "version-1", null]],
    ["semantic_profile", ["certified", "billing-ready", "unconnected", null]],
    ["grain", ["year", "daily", null]],
    ["id", ["", "A".repeat(64), "a".repeat(63), null]],
  ];
  for (const [field, values] of cases)
    for (const value of values) {
      const data = envelope();
      rejected({ ...data, records: [{ ...data.records[0], [field]: value }] });
    }
});
test("UTC timestamps and calendar-aligned bounded intervals reject impossible and contradictory evidence", () => {
  for (const date of [
    "2026-02-30T00:00:00Z",
    "2026-01-01",
    "2026-01-01T00:00:00",
    "2026-01-01T00:00:00+04:00",
    "2026-01-01T24:00:00Z",
    "2026-01-01T00:00:60Z",
    "0000-01-01T00:00:00Z",
    "2026-01-01T00:00:00.1234567Z",
  ]) {
    const data = envelope();
    rejected({ ...data, observed_at: date });
    rejected({
      ...data,
      records: [{ ...data.records[0], window_start: date }],
    });
  }
  for (const start of [
    "2026-01-02T00:00:00Z",
    "2026-01-01T00:00:00.000001Z",
    "2026-03-01T00:00:00Z",
  ]) {
    const data = envelope();
    rejected({
      ...data,
      records: [{ ...data.records[0], window_start: start }],
    });
  }
  const data = envelope();
  const day = {
    ...registration(),
    grain: "day",
    window_start: "2000-01-01T00:00:00Z",
    window_end: "2010-01-01T00:00:00Z",
  };
  assert.equal(
    parseData({ ...data, records: [day] }, q).records[0].grain,
    "day",
  );
  rejected({
    ...data,
    records: [{ ...day, window_end: "2011-01-01T00:00:00Z" }],
  });
  rejected({
    ...data,
    records: [{ ...registration(), window_end: "2400-01-01T00:00:00Z" }],
  });
});
test("list identity, order and filter are validated before display", () => {
  const data = envelope();
  data.total = 2;
  data.records = [registration(0), registration(1)];
  assert.equal(parseData(data, q).records.length, 2);
  rejected({ ...data, records: [registration(1), registration(0)] });
  rejected({ ...data, records: [registration(0), registration(0)] });
  rejected({
    ...data,
    records: [registration(0), { ...registration(1), id: registration(0).id }],
  });
  rejected({
    ...data,
    records: [
      registration(0),
      { ...registration(1), connection: registration(0).connection },
    ],
  });
  rejected(
    { ...envelope(), selection: { q: "foreign", after: 0 } },
    { ...q, q: "foreign" },
  );
  assert.equal(
    parseData(
      { ...envelope(), selection: { q: "connection-", after: 0 } },
      { ...q, q: "connection-" },
    ).total,
    1,
  );
});
test("bounded page counts, cursor and permission/snapshot pins agree exactly", () => {
  const data = envelope();
  data.total = 26;
  data.next_after = 25;
  data.records = Array.from({ length: 25 }, (_, i) => registration(i));
  assert.equal(parseData(data, q).next_after, 25);
  rejected({ ...data, total: 10001 });
  rejected({ ...data, records: data.records.slice(0, 24) });
  rejected({ ...data, next_after: 24 });
  rejected({ ...data, next_after: null });
  rejected({ ...data, total: 25 });
  rejected({ ...data, records: [...data.records, registration(25)] });
  const later = {
    ...data,
    selection: { q: "", after: 25 },
    records: [registration(25)],
    next_after: null,
  };
  const pinned = { ...q, after: 25, snapshot: C, permission_revision: B };
  assert.equal(parseData(later, pinned).records.length, 1);
  rejected({ ...later, snapshot: A }, pinned);
  rejected({ ...later, permission_revision: C }, pinned);
  rejected({ ...later, selection: { q: "", after: 24 } }, pinned);
  rejected({ ...later, records: [] }, pinned);
});
test("browser query rejects ambiguous keys, malformed pins and unsupported cursors", () => {
  for (const text of [
    "connection-q=A",
    "connection-after=01",
    "connection-after=10001",
    "connection-after=-1",
    "connection-q=a&connection-q=b",
    "connection-extra=x",
    "connection-snapshot=" + A,
    "connection-after=25",
    "connection-permission=" + A,
    "view=connections&view=connections",
    "view=security",
  ]) {
    assert.throws(
      () => query(new URLSearchParams(text), "connections"),
      /invalid_query/,
    );
  }
  const text = `connection-after=25&connection-snapshot=${A}&connection-permission=${B}`;
  assert.throws(
    () => query(new URLSearchParams(text), "security"),
    /invalid_query/,
  );
  const parsed = query(new URLSearchParams(text), "connections");
  assert.equal(parsed.after, 25);
  assert.equal(parsed.snapshot, A);
  assert.equal(
    queryString(parsed),
    `view=connections&q=&after=25&snapshot=${A}&permission_revision=${B}`,
  );
  assert.equal(
    query(new URLSearchParams("lang=ar&scope=requested"), "connections").q,
    "",
  );
});
test("duplicate JSON names are rejected at every object depth", () => {
  for (const text of [
    '{"a":1,"a":2}',
    '{"records":[{"id":"a","id":"b"}]}',
    '{"s":{"x":1,"x":2}}',
  ])
    assert.throws(() => parseJson(text), /invalid_response/);
  assert.deepEqual(parseJson('{"a":1,"b":{"a":2},"c":[1,2]}'), {
    a: 1,
    b: { a: 2 },
    c: [1, 2],
  });
});
test("transport uses only the fixed GET endpoint and captured closed query", async () => {
  let observed = "";
  await withFetch(
    async (url, options) => {
      observed = String(url);
      assert.equal(options?.method, "GET");
      assert.equal(options?.redirect, "error");
      assert.equal(options?.cache, "no-store");
      assert.deepEqual(options?.headers, {
        "X-HB-Local-Client": "1",
        Accept: "application/json",
      });
      return response(envelope());
    },
    async () => {
      const result = await load(q, new AbortController().signal);
      assert.equal(result.kind, "loaded");
      assert.equal(
        observed,
        "/api/focus-connections?view=connections&q=&after=0",
      );
    },
  );
});
test("missing producer is distinct from denied, stale and failed source metadata", async () => {
  for (const status of [404, 501])
    await withFetch(
      async () => response({}, status),
      async () => {
        assert.deepEqual(await load(q, new AbortController().signal), {
          kind: "unconnected",
        });
      },
    );
  for (const [status, code] of [
    [403, "scope_denied"],
    [409, "query_snapshot_changed"],
    [503, "collected_unavailable"],
  ] as const)
    await withFetch(
      async () => response({ error: code, request_id: "e".repeat(32) }, status),
      async () => {
        await assert.rejects(
          load(q, new AbortController().signal),
          (error) =>
            error instanceof ReadError &&
            error.code === code &&
            error.requestId === "e".repeat(32),
        );
      },
    );
  await withFetch(
    async () =>
      response({ error: "secret text", request_id: "wrong", extra: "x" }, 403),
    async () => {
      await assert.rejects(
        load(q, new AbortController().signal),
        /invalid_response/,
      );
    },
  );
});
test("transport rejects media, malformed UTF8, duplicate keys, oversize and redirects", async () => {
  const invalid = [
    new Response("<html>error</html>", {
      headers: { "Content-Type": "text/html" },
    }),
    new Response(new Uint8Array([0xc3, 0x28]), {
      headers: { "Content-Type": "application/json" },
    }),
    new Response('{"x":1,"x":2}', {
      headers: { "Content-Type": "application/json" },
    }),
    new Response("{}", {
      headers: {
        "Content-Type": "application/json",
        "Content-Length": "2097153",
      },
    }),
    new Response(new Uint8Array(2097153), {
      headers: { "Content-Type": "application/json" },
    }),
  ];
  for (const result of invalid)
    await withFetch(
      async () => result,
      async () => {
        await assert.rejects(
          load(q, new AbortController().signal),
          /invalid_response|response_size/,
        );
      },
    );
  const redirected = response(envelope());
  Object.defineProperty(redirected, "redirected", { value: true });
  await withFetch(
    async () => redirected,
    async () => {
      await assert.rejects(
        load(q, new AbortController().signal),
        /invalid_response/,
      );
    },
  );
});
test("transport observes abort before a request and after a noncompliant slow fetch", async () => {
  const controller = new AbortController();
  controller.abort();
  await withFetch(
    async () => {
      throw new Error("must not fetch");
    },
    async () => {
      await assert.rejects(load(q, controller.signal), { name: "AbortError" });
    },
  );
  const late = new AbortController();
  await withFetch(
    async () => {
      late.abort();
      return response(envelope());
    },
    async () => {
      await assert.rejects(load(q, late.signal), { name: "AbortError" });
    },
  );
});
