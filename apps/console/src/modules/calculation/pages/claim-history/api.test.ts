import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import {
  load,
  query,
  valid,
  validDecimal,
  validQuery,
  ServiceError,
  type Data,
} from "./api";
const fixture = JSON.parse(
  readFileSync(new URL("./fixtures.json", import.meta.url), "utf8"),
) as Data;
const fresh = () => structuredClone(fixture);
const request = () => query(new URLSearchParams(), "ledger");
const signal = () => new AbortController().signal;
async function mock(handler: typeof fetch, run: () => Promise<void>) {
  const previous = globalThis.fetch;
  globalThis.fetch = handler;
  try {
    await run();
  } finally {
    globalThis.fetch = previous;
  }
}
const error = (code: string) => (e: unknown) =>
  e instanceof ServiceError && e.code === code;
const invalid = (mutate: (d: Data) => void) => {
  const d = fresh();
  mutate(d);
  assert.equal(valid(d, request()), false);
};

test("QA observations preserve original identity, signed precision, zero and unknown without inference", () => {
  const d = fresh(),
    before = JSON.stringify(d);
  assert.equal(valid(d, request()), true);
  assert.equal(JSON.stringify(d), before);
  assert.equal(d.records[0].coverage.available, null);
  assert.equal(d.records[1].amounts[0].applied, "0");
  assert.equal(d.records[2].amounts[0].applied, "-12.3400");
  assert.equal(d.records[2].correction.original_claim_id, "QA-CLAIM-002");
  assert.equal(d.records[2].coverage.applied, "40.0000");
  assert.equal(d.records[3].protection, "replacement_pending");
  assert.equal(d.records[3].coverage.available, null);
  assert.equal(d.records[4].amounts.length, 0);
});
test("query projects only source filters and bounds normalized text", () => {
  const q = query(
    new URLSearchParams(
      "claim-q=%20QA%20%20test%20&claim-state=reserved&tenant=secret&lang=ko&persona=admin",
    ),
    "ledger",
  );
  assert.deepEqual(
    [...q],
    [
      ["view", "ledger"],
      ["state", "reserved"],
      ["q", "QA test"],
    ],
  );
  assert.equal(validQuery(q), true);
  for (const value of ["x".repeat(129), "bad\u0000value"]) {
    q.set("q", value);
    assert.equal(validQuery(q), false);
  }
  for (const value of ["anything", "", "delegation"]) {
    const r = request();
    r.set("view", value);
    assert.equal(validQuery(r), false);
  }
  const duplicate = request();
  duplicate.append("view", "ledger");
  assert.equal(validQuery(duplicate), false);
  const extra = request();
  extra.set("tenant", "x");
  assert.equal(validQuery(extra), false);
});
test("cursor requires canonical token and both revision pins", () => {
  const q = request();
  q.set("cursor", "page_2");
  assert.equal(validQuery(q), false);
  q.set("snapshot", fixture.snapshot);
  assert.equal(validQuery(q), false);
  q.set("permission_revision", fixture.permission_revision);
  assert.equal(validQuery(q), true);
  for (const value of ["../page", "", "x".repeat(129)]) {
    q.set("cursor", value);
    assert.equal(validQuery(q), false);
  }
});
test("read identity, commands and nested allowlists fail closed", () => {
  for (const field of [
    "auth_scope",
    "snapshot",
    "permission_revision",
  ] as const)
    invalid((d) => {
      d[field] = "bad";
    });
  invalid((d) => {
    d.read_only = false as true;
  });
  invalid((d) => {
    d.commands = "approve" as "unconnected";
  });
  invalid((d) => {
    d.numeric_format = "universal" as Data["numeric_format"];
  });
  invalid((d) => {
    d.selection.q = "wrong";
  });
  invalid((d) => {
    d.view = "partial";
  });
  invalid((d) => {
    d.cursor = "wrong";
  });
  const nested = [
    [],
    ["selection"],
    ["records", 0],
    ["records", 0, "target"],
    ["records", 0, "coverage"],
    ["records", 0, "amounts", 0],
    ["records", 0, "lineage"],
    ["records", 0, "correction"],
    ["records", 0, "external", 0],
    ["records", 0, "blockers", 0],
  ];
  for (const path of nested)
    invalid((d) => {
      let value: any = d;
      for (const key of path) value = value[key];
      value.private_payload = "no";
    });
  const q = request();
  q.set("snapshot", "d".repeat(64));
  assert.equal(valid(fresh(), q), false);
  q.delete("snapshot");
  q.set("permission_revision", "d".repeat(64));
  assert.equal(valid(fresh(), q), false);
});
test("page consistency and canonical row/reference uniqueness", () => {
  invalid((d) => {
    d.records.push(d.records[0]);
    d.total = 6;
  });
  invalid((d) => {
    d.records[0].id = "https://example.test/id";
  });
  invalid((d) => {
    d.total = 6;
  });
  invalid((d) => {
    d.total = -1;
  });
  invalid((d) => {
    d.next_cursor = "next";
  });
  invalid((d) => {
    d.records[0].external.push(d.records[0].external[0]);
  });
  invalid((d) => {
    d.records[0].amounts.push(d.records[0].amounts[0]);
  });
  invalid((d) => {
    d.records[0].correction.linked_claim_ids = ["QA-a", "QA-a"];
  });
  invalid((d) => {
    d.records[0].correction.original_claim_id = d.records[0].id;
  });
  const d = fresh();
  d.total = null;
  assert.equal(valid(d, request()), true);
  d.records = Array.from({ length: 25 }, (_, i) => ({
    ...structuredClone(fixture.records[4]),
    id: `QA-PAGE-${i}`,
  }));
  d.next_cursor = "next";
  assert.equal(valid(d, request()), true);
  d.records.push({ ...structuredClone(d.records[0]), id: "QA-26" });
  assert.equal(valid(d, request()), false);
});
test("timestamps require real calendar and explicit timezone", () => {
  for (const value of [
    "2026-02-29T00:00:00Z",
    "2026-04-31T00:00:00Z",
    "2026-11-06",
    "2026-11-06T25:00:00Z",
    "2026-11-06T01:00:00+24:00",
    "0000-01-01T00:00:00Z",
  ])
    invalid((d) => {
      d.records[0].updated_at = value;
    });
  const d = fresh();
  d.observed_at = "2024-02-29T23:59:59.123456789-09:30";
  assert.equal(valid(d, request()), true);
});
test("decimal read bounds preserve strings without activating arithmetic", () => {
  for (const value of [
    "0",
    "-0.000",
    "-12.3400",
    "9".repeat(76),
    "1." + "2".repeat(38),
  ])
    assert.equal(validDecimal(value), true);
  for (const value of [
    0,
    NaN,
    "1e2",
    "+1",
    "01",
    ".5",
    "1.",
    "Infinity",
    "9".repeat(77),
    "1." + "2".repeat(39),
  ])
    assert.equal(validDecimal(value), false);
  invalid((d) => {
    d.records[0].coverage.reserved = "-1";
  });
  invalid((d) => {
    d.records[0].coverage.available = "0";
  });
  invalid((d) => {
    d.records[0].amounts[0].applied = 1 as unknown as string;
  });
  const d = fresh();
  d.records[1].coverage.available = "999999";
  d.records[1].coverage.applied = "42";
  assert.equal(valid(d, request()), true);
});
test("view predicates use supplied evidence without release or geometry inference", () => {
  for (const view of ["partial", "reversals", "references"] as const) {
    const q = query(new URLSearchParams(), view),
      d = fresh();
    d.view = view;
    assert.equal(valid(d, q), false);
    d.records = d.records.filter((r) =>
      view === "partial"
        ? r.partial === true
        : view === "references"
          ? r.external.length > 0
          : r.correction.original_claim_id !== null,
    );
    d.total = d.records.length;
    assert.equal(valid(d, q), true);
  }
  const d = fresh();
  d.records[0].external[0].outcome = "not_applied";
  assert.equal(valid(d, request()), true);
  assert.equal(d.records[0].protection, "outcome_unknown");
  assert.equal(d.records[0].coverage.available, null);
  invalid((r) => {
    r.records[0].external[0].outcome = "unknown_state" as "unknown";
  });
});
test("fixed read-only fetch accepts synthetic and never changes fixture", async () => {
  await mock(
    async (url, init) => {
      assert.equal(String(url), "/api/claim-history?view=ledger&state=&q=");
      assert.equal(init?.method, "GET");
      assert.equal(init?.cache, "no-store");
      assert.equal(init?.redirect, "error");
      assert.deepEqual(init?.headers, { "X-HB-Local-Client": "1" });
      assert.equal(init?.body, undefined);
      return Response.json(fresh());
    },
    async () => {
      const result = await load(request(), signal());
      assert.equal(result.kind, "loaded");
    },
  );
});
test("404 and 501 are unconnected; live producer is always rejected", async () => {
  for (const status of [404, 501])
    await mock(
      async () => new Response("not connected", { status }),
      async () => {
        assert.deepEqual(await load(request(), signal()), {
          kind: "unconnected",
        });
      },
    );
  await mock(
    async () => Response.json({ ...fresh(), source_mode: "live" }),
    async () => {
      await assert.rejects(
        load(request(), signal()),
        error("producer_unbound"),
      );
    },
  );
});
test("malformed JSON, UTF8, redirect and response identities never publish", async () => {
  const redirected = Response.json(fresh());
  Object.defineProperty(redirected, "redirected", { value: true });
  for (const response of [
    new Response("{"),
    new Response(new Uint8Array([255])),
    redirected,
    Response.json({ ...fresh(), view: "partial" }),
  ])
    await mock(
      async () => response,
      async () => {
        await assert.rejects(
          load(request(), signal()),
          error("response_invalid"),
        );
      },
    );
});
test("actual stream cap applies even when content-length lies", async () => {
  for (const headers of [
    new Headers(),
    new Headers({ "Content-Length": "1" }),
    new Headers({ "Content-Length": "2097153" }),
  ])
    await mock(
      async () =>
        new Response(
          new ReadableStream({
            start(c) {
              c.enqueue(new Uint8Array(1024 * 1024));
              c.enqueue(new Uint8Array(1024 * 1024 + 1));
              c.close();
            },
          }),
          { headers },
        ),
      async () => {
        await assert.rejects(
          load(request(), signal()),
          error("response_too_large"),
        );
      },
    );
});
test("safe errors never expose raw bodies or unbounded identifiers", async () => {
  await mock(
    async () =>
      Response.json(
        {
          error: "permission_denied",
          request_id: "QA-req-1",
          private: "secret",
        },
        { status: 403 },
      ),
    async () => {
      await assert.rejects(
        load(request(), signal()),
        (e) =>
          e instanceof ServiceError &&
          e.code === "permission_denied" &&
          e.requestId === "QA-req-1" &&
          e.message === "permission_denied",
      );
    },
  );
  await mock(
    async () =>
      Response.json(
        { error: "<secret>", request_id: "x".repeat(129) },
        { status: 409 },
      ),
    async () => {
      await assert.rejects(
        load(request(), signal()),
        (e) =>
          e instanceof ServiceError &&
          e.code === "query_stale" &&
          e.requestId === "",
      );
    },
  );
  await mock(
    async () => {
      throw new Error("private network details");
    },
    async () => {
      await assert.rejects(
        load(request(), signal()),
        error("service_unavailable"),
      );
    },
  );
});
test("invalid queries and aborted reads do not publish; query is captured before awaiting", async () => {
  let calls = 0;
  await mock(
    async () => {
      calls++;
      return Response.json(fresh());
    },
    async () => {
      const q = request();
      q.set("state", "fake");
      await assert.rejects(load(q, signal()), error("request_invalid"));
      const controller = new AbortController();
      controller.abort();
      await assert.rejects(
        load(request(), controller.signal),
        (e) => e instanceof DOMException && e.name === "AbortError",
      );
      assert.equal(calls, 0);
    },
  );
  const q = request();
  await mock(
    async () => {
      q.set("q", "mutated");
      return Response.json(fresh());
    },
    async () => {
      assert.equal((await load(q, signal())).kind, "loaded");
    },
  );
});

test("aborted streamed or late responses cannot publish data", async () => {
  const controller = new AbortController();
  await mock(
    async () =>
      new Response(
        new ReadableStream({
          start(c) {
            c.enqueue(new TextEncoder().encode(JSON.stringify(fresh())));
            controller.abort();
            c.close();
          },
        }),
      ),
    async () => {
      await assert.rejects(
        load(request(), controller.signal),
        (e) => e instanceof DOMException && e.name === "AbortError",
      );
    },
  );
  const late = new AbortController();
  await mock(
    async () => {
      late.abort();
      return Response.json(fresh());
    },
    async () => {
      await assert.rejects(
        load(request(), late.signal),
        (e) => e instanceof DOMException && e.name === "AbortError",
      );
    },
  );
});
test("pinned pages retain exact snapshot permission and filter identity", () => {
  const q = request();
  q.set("cursor", "QA_page2");
  q.set("snapshot", fixture.snapshot);
  q.set("permission_revision", fixture.permission_revision);
  const d = fresh();
  d.cursor = "QA_page2";
  d.total = 30;
  assert.equal(valid(d, q), true);
  d.permission_revision = "e".repeat(64);
  assert.equal(valid(d, q), false);
  d.permission_revision = fixture.permission_revision;
  d.snapshot = "e".repeat(64);
  assert.equal(valid(d, q), false);
  const selected = request();
  selected.set("state", "reserved");
  const filtered = fresh();
  filtered.selection.state = "reserved";
  assert.equal(valid(filtered, selected), false);
});
