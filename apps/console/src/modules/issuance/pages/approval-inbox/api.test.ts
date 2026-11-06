import test from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import {
  query,
  validQuery,
  valid,
  load,
  validDecimal,
  differenceMatches,
  ServiceError,
  type Data,
  type ApprovalRecord,
} from "./api";
const source = JSON.parse(
  readFileSync(new URL("./fixtures.json", import.meta.url), "utf8"),
) as Data;
const fixture = () => structuredClone(source);
const q = (view = "history", values = "") =>
  query(new URLSearchParams(values), view);
async function withFetch(mock: typeof fetch, action: () => Promise<void>) {
  const original = globalThis.fetch;
  globalThis.fetch = mock;
  try {
    await action();
  } finally {
    globalThis.fetch = original;
  }
}
const signal = () => new AbortController().signal;
test("accepts labelled synthetic read-only records without inventing unknown amounts", () => {
  const d = fixture();
  assert.equal(valid(d, q()), true);
  assert.equal(d.records[1].amounts[0].candidate, "0");
  assert.equal(d.records[2].amounts[0].erp, null);
  assert.equal(d.read_only, true);
});
test("query projects only bounded request evidence and forces monthly recognition", () => {
  const result = q(
    "recognition",
    "approval-kind=policy&approval-q=%20QA%20%20target%20&tenant=foreign&lang=ar",
  );
  assert.equal(result.get("kind"), "monthly_recognition");
  assert.equal(result.get("q"), "QA target");
  assert.equal(result.has("tenant"), false);
  assert.equal(result.has("lang"), false);
  assert.equal(validQuery(result), true);
  assert.equal(validQuery(q("unknown")), false);
  assert.equal(
    validQuery(q("history", "approval-q=" + "a".repeat(129))),
    false,
  );
});
test("cursor requires both pinned snapshot and permission revision", () => {
  const p = q("history", "approval-cursor=next_page");
  assert.equal(validQuery(p), false);
  p.set("snapshot", source.snapshot);
  assert.equal(validQuery(p), false);
  p.set("permission_revision", source.permission_revision);
  assert.equal(validQuery(p), true);
  const d = fixture();
  d.cursor = "next_page";
  d.total = 30;
  assert.equal(valid(d, p), true);
  d.permission_revision = "d".repeat(64);
  assert.equal(valid(d, p), false);
  d.permission_revision = source.permission_revision;
  d.snapshot = "d".repeat(64);
  assert.equal(valid(d, p), false);
});
test("rejects write authority, unbound live mode and unallowlisted data", () => {
  for (const mutate of [
    (d: Data) => {
      d.read_only = false as true;
    },
    (d: Data) => {
      d.commands = "connected" as "unconnected";
    },
    (d: Data) => {
      d.source_mode = "live";
    },
    (d: Data) => {
      (d as unknown as Record<string, unknown>).private_extra = "not-projected";
    },
    (d: Data) => {
      (d.records[0].target as unknown as Record<string, unknown>).extra =
        "not-projected";
    },
    (d: Data) => {
      (d.records[0].amounts[0] as unknown as Record<string, unknown>).rate =
        "not-projected";
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, q()), false);
  }
});
test("rejects scope echo, permission, identity and total corruption", () => {
  for (const mutate of [
    (d: Data) => {
      d.view = "pending";
    },
    (d: Data) => {
      d.selection.q = "foreign";
    },
    (d: Data) => {
      d.auth_scope = "not-a-hash";
    },
    (d: Data) => {
      d.permission_revision = "invalid";
    },
    (d: Data) => {
      d.actor.id = "not a canonical principal";
    },
    (d: Data) => {
      d.records[1].id = d.records[0].id;
    },
    (d: Data) => {
      d.total = 4;
    },
    (d: Data) => {
      d.total = 6;
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, q()), false);
  }
});
test("distinguishes loaded exact zero and unknown counts with bounded pagination", () => {
  const d = fixture();
  d.records = [];
  d.total = 0;
  assert.equal(valid(d, q()), true);
  d.total = null;
  assert.equal(valid(d, q()), true);
  d.next_cursor = "page_two";
  assert.equal(valid(d, q()), false);
  d.records = Array.from({ length: 25 }, (_, i) => ({
    ...structuredClone(source.records[0]),
    id: `QA-PAGE-${i}`,
  }));
  d.total = 26;
  assert.equal(valid(d, q()), true);
  d.next_cursor = null;
  assert.equal(valid(d, q()), false);
  d.next_cursor = "page_two";
  d.records.push({ ...structuredClone(source.records[0]), id: "QA-PAGE-26" });
  assert.equal(valid(d, q()), false);
});
test("requires current page rows to match action, decision actor and recognition view", () => {
  const d = fixture();
  d.view = "decisions";
  d.records = [structuredClone(source.records[3])];
  d.total = 1;
  assert.equal(valid(d, q("decisions")), true);
  d.records[0].approver = null;
  d.records[0].author = d.actor;
  assert.equal(valid(d, q("decisions")), false);
  d.records = [structuredClone(source.records[0])];
  d.view = "recognition";
  d.selection.kind = "monthly_recognition";
  assert.equal(valid(d, q("recognition")), false);
  d.records = [structuredClone(source.records[4])];
  assert.equal(valid(d, q("recognition")), true);
  d.view = "history";
  d.selection.kind = "policy";
  assert.equal(valid(d, q("history", "approval-kind=policy")), false);
});
test("rejects impossible calendars, missing timezone and overbound field collections", () => {
  for (const mutate of [
    (d: Data) => {
      d.observed_at = "2026-02-30T00:00:00Z";
    },
    (d: Data) => {
      d.records[0].deadline = "2026-11-08T00:00:00";
    },
    (d: Data) => {
      d.records[0].updated_at = "2026-11-06T25:00:00Z";
    },
    (d: Data) => {
      d.records[0].evidence.blockers = Array.from({ length: 17 }, (_, i) => ({
        code: `QA-${i}`,
        text: "Synthetic blocker",
      }));
    },
    (d: Data) => {
      d.records[0].amounts = Array.from({ length: 17 }, () =>
        structuredClone(d.records[0].amounts[0]),
      );
    },
    (d: Data) => {
      d.records[0].target.label = "x".repeat(513);
    },
  ]) {
    const d = fixture();
    mutate(d);
    assert.equal(valid(d, q()), false);
  }
});
test("finite decimal display preserves source scale without rounding or float conversion", () => {
  for (const value of [
    "0",
    "-0.000",
    "100.0000",
    "-99999999999999999999999999999999999999.12345678901234567890123456789012345678",
  ])
    assert.equal(validDecimal(value), true, value);
  for (const value of [
    1,
    NaN,
    "1e2",
    "+1",
    "01",
    ".1",
    "1.",
    "Infinity",
    "0." + "1".repeat(39),
    "1".repeat(77),
  ])
    assert.equal(validDecimal(value), false, String(value));
  assert.equal(
    differenceMatches(
      "9007199254740993.000000000000000001",
      "9007199254740993.000000000000000004",
      "0.000000000000000003",
    ),
    true,
  );
  assert.equal(differenceMatches("-10.500", "-12.75", "-2.2500"), true);
  assert.equal(differenceMatches("0.1", "0.3", "0.2"), true);
  assert.equal(differenceMatches("0.1", "0.3", "0.19999999999999998"), false);
});
test("never combines currencies or bases and validates only supplied exact differences", () => {
  const d = fixture(),
    a = d.records[0].amounts[0];
  d.records[0].amounts.push(
    { ...a, currency: "KRW", candidate: "110.0000" },
    { ...a, basis: "gross" },
  );
  assert.equal(valid(d, q()), true);
  d.records[0].amounts.push({ ...a });
  assert.equal(valid(d, q()), false);
  d.records[0].amounts = [{ ...a, difference: "11" }];
  assert.equal(valid(d, q()), false);
  d.records[0].amounts[0].base = null;
  assert.equal(valid(d, q()), true);
  assert.equal(d.records[0].amounts[0].base, null);
});
test("separates ERP unknown and not-applicable evidence without inferring final amounts", () => {
  const d = fixture(),
    r = d.records[2];
  assert.equal(r.erp.applicability, "applicable");
  assert.equal(r.amounts[0].erp, null);
  assert.equal(valid(d, q()), true);
  r.erp.applicability = "not_applicable";
  assert.equal(valid(d, q()), false);
  r.kind = "policy";
  assert.equal(valid(d, q()), true);
  r.amounts[0].erp = "0";
  assert.equal(valid(d, q()), false);
  r.amounts[0].erp = null;
  r.erp.external_revision = "QA-EXTERNAL-1";
  assert.equal(valid(d, q()), false);
});
test("request transport is same-origin read-only, uncached and never follows redirects", async () => {
  let calls = 0;
  await withFetch(
    async (input, init) => {
      calls++;
      assert.equal(String(input).startsWith("/api/approval-review?"), true);
      assert.equal(init?.method, "GET");
      assert.equal(init?.cache, "no-store");
      assert.equal(init?.redirect, "error");
      assert.equal(
        (init?.headers as Record<string, string>)["X-HB-Local-Client"],
        "1",
      );
      return new Response(JSON.stringify(source));
    },
    async () => {
      const r = await load(q(), signal());
      assert.equal(r.kind, "loaded");
    },
  );
  assert.equal(calls, 1);
});
test("only route-not-supported statuses are unconnected and delegation performs no fetch", async () => {
  for (const status of [404, 501])
    await withFetch(
      async () => new Response("not a DTO", { status }),
      async () =>
        assert.deepEqual(await load(q(), signal()), { kind: "unconnected" }),
    );
  await withFetch(
    async () => {
      throw new Error("must not fetch delegation");
    },
    async () =>
      assert.deepEqual(await load(q("delegation"), signal()), {
        kind: "unconnected",
      }),
  );
  await withFetch(
    async () =>
      new Response(
        JSON.stringify({ error: "scope_denied", request_id: "QA-DENIED" }),
        { status: 403 },
      ),
    async () => {
      await assert.rejects(
        load(q(), signal()),
        (e) =>
          e instanceof ServiceError &&
          e.code === "scope_denied" &&
          e.requestId === "QA-DENIED",
      );
    },
  );
});
test("fails closed on live payload even when it otherwise matches the prototype", async () => {
  const d = fixture();
  d.source_mode = "live";
  await withFetch(
    async () => new Response(JSON.stringify(d)),
    async () =>
      assert.rejects(
        load(q(), signal()),
        (e) => e instanceof ServiceError && e.code === "producer_unbound",
      ),
  );
});
test("sanitizes malformed or private error bodies and does not retain extra fields", async () => {
  await withFetch(
    async () =>
      new Response(
        JSON.stringify({
          error: "private driver detail\nsecret",
          request_id: "bad\nrequest",
          private_data: "never-retain",
        }),
        { status: 403 },
      ),
    async () =>
      assert.rejects(
        load(q(), signal()),
        (e) =>
          e instanceof ServiceError &&
          e.code === "permission_denied" &&
          e.requestId === "" &&
          !e.message.includes("private"),
      ),
  );
  await withFetch(
    async () => new Response("<html>driver stack</html>", { status: 503 }),
    async () =>
      assert.rejects(
        load(q(), signal()),
        (e) => e instanceof ServiceError && e.code === "response_invalid",
      ),
  );
  const d = fixture() as Data & { extra?: string };
  d.extra = "never-retain";
  await withFetch(
    async () => new Response(JSON.stringify(d)),
    async () =>
      assert.rejects(
        load(q(), signal()),
        (e) => e instanceof ServiceError && e.code === "response_invalid",
      ),
  );
});
test("enforces actual streamed byte cap despite a false content length", async () => {
  let cancelled = false;
  const body = new ReadableStream<Uint8Array>({
    start(c) {
      c.enqueue(new Uint8Array(1024 * 1024));
      c.enqueue(new Uint8Array(1024 * 1024 + 1));
    },
    cancel() {
      cancelled = true;
    },
  });
  await withFetch(
    async () => new Response(body, { headers: { "Content-Length": "1" } }),
    async () =>
      assert.rejects(
        load(q(), signal()),
        (e) => e instanceof ServiceError && e.code === "response_too_large",
      ),
  );
  assert.equal(cancelled, true);
});
test("rejects oversized declared responses, invalid UTF-8 and redirect responses", async () => {
  await withFetch(
    async () =>
      new Response("{}", {
        headers: { "Content-Length": "999999999999999999999999" },
      }),
    async () =>
      assert.rejects(
        load(q(), signal()),
        (e) => e instanceof ServiceError && e.code === "response_too_large",
      ),
  );
  await withFetch(
    async () => new Response(new Uint8Array([0xc0])),
    async () =>
      assert.rejects(
        load(q(), signal()),
        (e) => e instanceof ServiceError && e.code === "response_invalid",
      ),
  );
  const redirect = new Response("{}");
  Object.defineProperty(redirect, "redirected", { value: true });
  await withFetch(
    async () => redirect,
    async () =>
      assert.rejects(
        load(q(), signal()),
        (e) => e instanceof ServiceError && e.code === "response_invalid",
      ),
  );
});
test("abort and invalid queries cannot silently produce loaded data", async () => {
  const c = new AbortController();
  c.abort();
  let calls = 0;
  await withFetch(
    async () => {
      calls++;
      return new Response(JSON.stringify(source));
    },
    async () => {
      await assert.rejects(load(q(), c.signal));
      await assert.rejects(
        load(q("not-a-view"), signal()),
        (e) => e instanceof ServiceError && e.code === "request_invalid",
      );
    },
  );
  assert.equal(calls, 0);
});
