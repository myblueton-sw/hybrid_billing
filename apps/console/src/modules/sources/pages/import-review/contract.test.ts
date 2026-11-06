import assert from "node:assert/strict";
import test from "node:test";
import {
  parseData,
  query,
  queryString,
  type Receipt,
  type Data,
  type Query,
} from "./contract";
import { load, parseJson, ReadError } from "./api";
const A = "a".repeat(64),
  B = "b".repeat(64),
  C = "c".repeat(64);
const q: Query = { view: "imports", q: "", after: 0 };
function delivery(): Receipt {
  return {
    id: A,
    kind: "delivery",
    connection: "scope-one",
    stage: "verified",
    window_start: "2026-10-01T00:00:00+00:00",
    window_end: "2026-11-01T00:00:00Z",
    source_revision: "9223372036854775807",
    focus_version: "1.3",
    format: "csv",
    declared_rows: "0",
    file_bytes: "128",
    file_digest: B,
    mapping_digest: C,
    received_at: "2026-10-02T00:00:00Z",
    complete_through: null,
    finalized: null,
    reason: null,
    root_receipt: null,
    replaced_by: null,
    attempts: null,
    known_zero: null,
    published: null,
  };
}
function envelope(): Data {
  return {
    schema: 1,
    profile: "retained-focus-receipts-v1",
    read_only: true,
    financial_authority: false,
    commands: "unconnected",
    view: "imports",
    selection: { q: "", after: 0 },
    audience: "company",
    member_count: 2,
    auth_scope: A,
    permission_revision: B,
    snapshot: C,
    observed_at: "2026-11-06T09:00:00Z",
    total: 1,
    next_after: null,
    capabilities: {
      receipts: "fully-authorized-metadata-only",
      completeness: "registered-expectations-only",
      extraction: "unconnected",
    },
    records: [delivery()],
  };
}
const rejected = (body: unknown, request = q) =>
  assert.throws(() => parseData(body, request), /invalid_response/);
test("metadata retains exact revisions and distinguishes declared zero from unknown", () => {
  const d = parseData(envelope(), q);
  assert.equal(d.records[0].source_revision, "9223372036854775807");
  assert.equal(d.records[0].declared_rows, "0");
  assert.equal(d.records[0].known_zero, null);
  assert.equal(d.records[0].published, null);
  const e = envelope();
  e.records[0].file_bytes = null;
  assert.equal(parseData(e, q).records[0].file_bytes, null);
});
test("latest quarantine does not overwrite current published completeness", () => {
  const d = envelope();
  d.view = "completeness";
  d.records[0] = {
    ...delivery(),
    kind: "coverage",
    source_revision: null,
    format: null,
    declared_rows: null,
    file_bytes: null,
    file_digest: null,
    mapping_digest: null,
    stage: "quarantined",
    received_at: null,
    published: true,
    finalized: false,
    known_zero: true,
    complete_through: "2026-10-15T00:00:00Z",
  };
  assert.equal(
    parseData(d, { ...q, view: "completeness" }).records[0].stage,
    "quarantined",
  );
  d.records[0].published = false;
  rejected(d, { ...q, view: "completeness" });
  d.records[0].finalized = null;
  d.records[0].known_zero = null;
  d.records[0].complete_through = null;
  assert.equal(
    parseData(d, { ...q, view: "completeness" }).records[0].published,
    false,
  );
});
test("unsupported extraction and expectations retain unknown total", () => {
  for (const view of ["extraction", "completeness"] as const) {
    const d = envelope();
    d.view = view;
    d.capabilities.completeness = "unavailable";
    d.records = [];
    d.total = null;
    assert.equal(parseData(d, { ...q, view }).total, null);
    d.total = 0;
    rejected(d, { ...q, view });
  }
});
test("closed envelope and nested schema exclude internal or authority fields", () => {
  for (const field of [
    "tenant",
    "actor",
    "commands_allowed",
    "object_key",
    "body",
    "url",
    "internal_cost",
  ]) {
    const d = envelope();
    rejected({ ...d, [field]: "hidden" });
    rejected({ ...d, records: [{ ...d.records[0], [field]: "hidden" }] });
  }
  const d = envelope();
  rejected({ ...d, financial_authority: true });
  rejected({ ...d, read_only: false });
  rejected({ ...d, commands: "publish" });
  rejected({ ...d, selection: { ...d.selection, tenant: "other" } });
  rejected({ ...d, capabilities: { ...d.capabilities, approve: true } });
});
test("job and financial labels cannot impersonate collection stages", () => {
  for (const stage of [
    "succeeded",
    "approved",
    "eligible",
    "paid",
    "rejected",
  ]) {
    const d = envelope();
    d.records[0].stage = stage;
    rejected(d);
  }
  const d = envelope();
  d.records[0].kind = "quarantine";
  d.records[0].stage = "quarantined";
  d.records[0].format = null;
  d.records[0].file_bytes = null;
  d.records[0].mapping_digest = null;
  d.records[0].reason = "version_mismatch";
  rejected(d);
  d.view = "quarantine";
  assert.equal(
    parseData(d, { ...q, view: "quarantine" }).records[0].kind,
    "quarantine",
  );
  d.records[0].published = true;
  rejected(d, { ...q, view: "quarantine" });
});
test("scalar/date bounds and self replacement fail closed", () => {
  const fields: Record<string, unknown[]> = {
    source_revision: ["0", "9223372036854775808", "1e3", 1],
    declared_rows: ["10001", "-1", "00", 0],
    file_bytes: ["16777217", "1.0"],
    attempts: ["11", "-1"],
    connection: ["https://host/", "UPPER"],
    file_digest: ["x", A.toUpperCase()],
    focus_version: ["../../../key", "1.3.1"],
    reason: ["private text", "Stack:secret"],
    received_at: [
      "2026-02-30T00:00:00Z",
      "2026-10-01T25:00:00Z",
      "0000-01-01T00:00:00Z",
      "2026-10-01T00:00:00+03:00",
    ],
    replaced_by: [A],
  };
  for (const [key, values] of Object.entries(fields))
    for (const value of values) {
      const d = envelope();
      (d.records[0] as unknown as Record<string, unknown>)[key] = value;
      rejected(d);
    }
  const d = envelope();
  d.member_count = 0;
  rejected(d);
  d.member_count = 10001;
  rejected(d);
  d.member_count = 2;
  d.records[0].window_end = d.records[0].window_start;
  rejected(d);
});
test("pagination validates exact pins, page population and unique IDs", () => {
  const d = envelope();
  d.total = 26;
  d.records = Array.from({ length: 25 }, (_, i) => ({
    ...delivery(),
    id: i.toString(16).padStart(64, "0"),
  }));
  d.next_after = 25;
  assert.equal(parseData(d, q).next_after, 25);
  d.records[1].id = d.records[0].id;
  rejected(d);
  d.records[1].id = A;
  d.next_after = null;
  rejected(d);
  const last = envelope();
  last.selection.after = 25;
  last.total = 26;
  assert.equal(
    parseData(last, { ...q, after: 25, snapshot: C, permission_revision: B })
      .total,
    26,
  );
  rejected(last, { ...q, after: 25, snapshot: A, permission_revision: B });
  rejected(last, { ...q, after: 25, snapshot: C, permission_revision: A });
  last.records = [];
  rejected(last, { ...q, after: 25, snapshot: C, permission_revision: B });
});
test("request selection excludes client authority and validates supported controls", () => {
  const params = new URLSearchParams({
    actor: "admin",
    tenant: "foreign",
    "receipt-q": "scope-one",
    "receipt-after": "0",
  });
  const s = queryString(query(params, "imports"));
  assert(!s.includes("actor") && !s.includes("tenant"));
  assert(s.includes("q=scope-one"));
  const invalidQueries: Record<string, string>[] = [
    { "receipt-after": "25" },
    { "receipt-q": "<script>" },
    { "receipt-actor": "admin" },
    { "receipt-after": "01" },
    { "receipt-snapshot": A },
    { "receipt-after": "10001" },
  ];
  for (const changes of invalidQueries)
    assert.throws(
      () => query(new URLSearchParams(changes), "imports"),
      /invalid_query/,
    );
  assert.throws(
    () => query(new URLSearchParams("receipt-q=a&receipt-q=b"), "imports"),
    /invalid_query/,
  );
});
test("raw escaped duplicate keys and excessive nesting are rejected", () => {
  assert.throws(() => parseJson('{"schema":1,"schema":1}'), /invalid_response/);
  assert.throws(
    () => parseJson('{"schema":1,"\\u0073chema":1}'),
    /invalid_response/,
  );
  assert.throws(
    () => parseJson("[".repeat(17) + "0" + "]".repeat(17)),
    /invalid_response/,
  );
  assert.deepEqual(parseJson('{"items":[{"a":1},{"a":2}]}'), {
    items: [{ a: 1 }, { a: 2 }],
  });
});
test("transport fixes GET/header/cache/redirect and rejects success request IDs", async (t) => {
  const original = globalThis.fetch;
  t.after(() => {
    globalThis.fetch = original;
  });
  globalThis.fetch = async (url, init) => {
    assert(String(url).startsWith("/api/focus-receipts?"));
    assert.equal(init?.method, "GET");
    assert.equal(init?.redirect, "error");
    assert.equal(init?.cache, "no-store");
    assert.equal(
      (init?.headers as Record<string, string>)["X-HB-Local-Client"],
      "1",
    );
    return new Response(JSON.stringify(envelope()), {
      headers: { "content-type": "application/json" },
    });
  };
  assert.equal((await load(q, new AbortController().signal)).kind, "loaded");
  globalThis.fetch = async () =>
    new Response(
      JSON.stringify({ ...envelope(), request_id: "a".repeat(32) }),
      { headers: { "content-type": "application/json" } },
    );
  await assert.rejects(
    load(q, new AbortController().signal),
    /invalid_response/,
  );
});
test("absent producer, closed denial and malformed content remain distinct", async (t) => {
  const original = globalThis.fetch;
  t.after(() => {
    globalThis.fetch = original;
  });
  globalThis.fetch = async () => new Response("missing", { status: 404 });
  assert.equal(
    (await load(q, new AbortController().signal)).kind,
    "unconnected",
  );
  globalThis.fetch = async () =>
    new Response(
      JSON.stringify({ error: "scope_denied", request_id: "a".repeat(32) }),
      { status: 403, headers: { "content-type": "application/json" } },
    );
  await assert.rejects(
    load(q, new AbortController().signal),
    (e) =>
      e instanceof ReadError &&
      e.code === "scope_denied" &&
      e.requestId === "a".repeat(32),
  );
  for (const [text, mime] of [
    ["{}", "text/html"],
    ['{"error":"scope_denied","request_id":"bad"}', "application/json"],
  ]) {
    globalThis.fetch = async () =>
      new Response(text, { status: 403, headers: { "content-type": mime } });
    await assert.rejects(
      load(q, new AbortController().signal),
      /invalid_response/,
    );
  }
});
test("streamed byte and UTF8 limits do not trust Content-Length", async (t) => {
  const original = globalThis.fetch;
  t.after(() => {
    globalThis.fetch = original;
  });
  globalThis.fetch = async () =>
    new Response(new Uint8Array(2 * 1024 * 1024 + 1), {
      headers: { "content-type": "application/json", "content-length": "1" },
    });
  await assert.rejects(load(q, new AbortController().signal), /response_size/);
  globalThis.fetch = async () =>
    new Response(new Uint8Array([0xc3, 0x28]), {
      headers: { "content-type": "application/json" },
    });
  await assert.rejects(
    load(q, new AbortController().signal),
    /invalid_response/,
  );
  const c = new AbortController();
  c.abort();
  const aborted = new DOMException("aborted", "AbortError");
  globalThis.fetch = async () => {
    throw aborted;
  };
  await assert.rejects(
    load(q, c.signal),
    (e) => e instanceof DOMException && e.name === "AbortError",
  );
});

test("per-kind mandatory and unsupported fields are enforced", () => {
  for (const f of [
    "window_start",
    "window_end",
    "source_revision",
    "focus_version",
    "declared_rows",
    "file_digest",
    "mapping_digest",
    "received_at",
  ]) {
    const d = envelope();
    (d.records[0] as unknown as Record<string, unknown>)[f] = null;
    rejected(d);
  }
  for (const [field, value] of Object.entries({
    reason: "version_mismatch",
    root_receipt: B,
    replaced_by: B,
    attempts: "0",
  })) {
    const d = envelope();
    (d.records[0] as unknown as Record<string, unknown>)[field] = value;
    rejected(d);
  }
  const d = envelope();
  d.view = "completeness";
  d.records[0] = {
    ...delivery(),
    kind: "coverage",
    connection: "scope-one",
    source_revision: null,
    format: null,
    declared_rows: null,
    file_bytes: null,
    file_digest: null,
    mapping_digest: null,
    received_at: null,
    published: true,
    finalized: true,
    known_zero: false,
    complete_through: "2026-11-01T00:00:00Z",
  };
  const request = { ...q, view: "completeness" as const };
  assert.equal(parseData(d, request).records[0].published, true);
  for (const f of ["complete_through", "finalized", "known_zero"]) {
    const b = structuredClone(d);
    (b.records[0] as unknown as Record<string, unknown>)[f] = null;
    rejected(b, request);
  }
  for (const through of [
    "2026-10-01T00:00:00Z",
    "2026-11-02T00:00:00Z",
    "2026-10-31T00:00:00Z",
  ]) {
    const b = structuredClone(d);
    b.records[0].complete_through = through;
    rejected(b, request);
  }
  const precise = structuredClone(d);
  precise.records[0].finalized = false;
  precise.records[0].complete_through = "2026-10-01T00:00:00.000001Z";
  assert.equal(parseData(precise, request).records[0].finalized, false);
});

test("load captures query identity and rejects redirected or cancelled late responses", async (t) => {
  const original = globalThis.fetch;
  t.after(() => {
    globalThis.fetch = original;
  });
  const mutable = { ...q };
  globalThis.fetch = async () => {
    mutable.after = 99;
    return new Response(JSON.stringify(envelope()), {
      headers: { "content-type": "application/json" },
    });
  };
  assert.equal(
    (await load(mutable, new AbortController().signal)).kind,
    "loaded",
  );
  globalThis.fetch = async () => {
    const r = new Response(JSON.stringify(envelope()), {
      headers: { "content-type": "application/json" },
    });
    Object.defineProperty(r, "redirected", { value: true });
    return r;
  };
  await assert.rejects(
    load(q, new AbortController().signal),
    /invalid_response/,
  );
  const aborter = new AbortController();
  globalThis.fetch = async () => {
    aborter.abort();
    return new Response(JSON.stringify(envelope()), {
      headers: { "content-type": "application/json" },
    });
  };
  await assert.rejects(
    load(q, aborter.signal),
    (e) => e instanceof DOMException && e.name === "AbortError",
  );
});

test("admitted quarantine cannot omit its validated declaration", () => {
  const d = envelope();
  d.view = "quarantine";
  d.records[0] = {
    ...delivery(),
    kind: "quarantine",
    stage: "quarantined",
    format: null,
    file_bytes: null,
    mapping_digest: null,
    reason: "decimal_syntax",
  };
  const request = { ...q, view: "quarantine" as const };
  assert.equal(parseData(d, request).records[0].reason, "decimal_syntax");
  for (const f of [
    "source_revision",
    "declared_rows",
    "focus_version",
    "file_digest",
    "received_at",
    "reason",
  ]) {
    const b = structuredClone(d);
    (b.records[0] as unknown as Record<string, unknown>)[f] = null;
    rejected(b, request);
  }
});
