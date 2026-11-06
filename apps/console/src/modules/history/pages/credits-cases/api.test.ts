import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import {
  load,
  query,
  valid,
  validQuery,
  validDecimal,
  validDate,
  validTimeZone,
  ServiceError,
  views,
  type View,
  type Data,
  type PaymentRecord,
  type CaseRecord,
  type BalanceRecord,
  type AdjustmentRecord,
} from "./api";
const fixtures = JSON.parse(
  readFileSync(new URL("./fixtures.json", import.meta.url), "utf8"),
) as Record<View, Data>;
const fresh = (view: View = "balances") => structuredClone(fixtures[view]);
const request = (view: View = "balances") => query(new URLSearchParams(), view);
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
function invalid(view: View, mutate: (d: Data) => void) {
  const d = fresh(view);
  mutate(d);
  assert.equal(valid(d, request(view)), false);
}
function paymentChange(mutate: (r: PaymentRecord) => void) {
  invalid("payment-terms", (d) => mutate(d.records[0] as PaymentRecord));
}

test("all five QA view variants preserve source signed amounts, zero, null and authority", () => {
  for (const view of views) {
    const d = fresh(view),
      before = JSON.stringify(d);
    assert.equal(valid(d, request(view)), true, view);
    assert.equal(JSON.stringify(d), before);
  }
  const balances = fresh().records as BalanceRecord[];
  assert.equal(balances[0].closing, "0.0000");
  assert.equal(balances[1].closing, null);
  assert.equal(balances[1].authority.freshness, "stale");
  const adjustment = fresh("adjustments").records[0] as AdjustmentRecord;
  assert.equal(adjustment.amounts[0].value, "-12.3400");
  assert.equal(adjustment.original_claim_ref, "QA-CLAIM-1");
});
test("query projects only normalized credit filters with bounded text", () => {
  const q = query(
    new URLSearchParams(
      "credit-q=%20QA%20%20label%20&tenant=QA-ignored&lang=ko&persona=admin&credit-state=open",
    ),
    "cases",
  );
  assert.deepEqual(
    [...q],
    [
      ["view", "cases"],
      ["q", "QA label"],
    ],
  );
  assert.equal(validQuery(q), true);
  for (const value of ["x".repeat(129), "bad\u0000text"]) {
    q.set("q", value);
    assert.equal(validQuery(q), false);
  }
  for (const value of ["", "ledger", "unknown"]) {
    const q = request();
    q.set("view", value);
    assert.equal(validQuery(q), false);
  }
  const duplicate = request();
  duplicate.append("q", "");
  assert.equal(validQuery(duplicate), false);
  const extra = request();
  extra.set("state", "open");
  assert.equal(validQuery(extra), false);
});
test("cursor requires safe token and both exact revision pins", () => {
  const q = request();
  q.set("cursor", "QA_page2");
  assert.equal(validQuery(q), false);
  q.set("snapshot", fixtures.balances.snapshot);
  assert.equal(validQuery(q), false);
  q.set("permission_revision", fixtures.balances.permission_revision);
  assert.equal(validQuery(q), true);
  for (const value of ["", "../page", "x".repeat(129)]) {
    q.set("cursor", value);
    assert.equal(validQuery(q), false);
  }
  const d = fresh();
  d.cursor = "QA_page2";
  d.total = 30;
  q.set("cursor", "QA_page2");
  assert.equal(valid(d, q), true);
  d.permission_revision = "d".repeat(64);
  assert.equal(valid(d, q), false);
  d.permission_revision = fixtures.balances.permission_revision;
  d.snapshot = "d".repeat(64);
  assert.equal(valid(d, q), false);
});
test("read identity and strict view kind are required", () => {
  for (const field of [
    "auth_scope",
    "snapshot",
    "permission_revision",
  ] as const)
    invalid("balances", (d) => {
      d[field] = "bad";
    });
  invalid("balances", (d) => {
    d.read_only = false as true;
  });
  invalid("balances", (d) => {
    d.commands = "apply" as "unconnected";
  });
  invalid("balances", (d) => {
    d.source_mode = "live";
  });
  invalid("balances", (d) => {
    d.numeric_format = "universal" as Data["numeric_format"];
  });
  invalid("balances", (d) => {
    d.selection.q = "mismatch";
  });
  invalid("balances", (d) => {
    d.cursor = "wrong";
  });
  invalid("balances", (d) => {
    d.view = "cases";
  });
  for (const view of views)
    for (const other of views.filter((v) => v !== view))
      invalid(view, (d) => {
        d.records = [fresh(other).records[0]];
        d.total = 1;
      });
});
test("unknown nested fields are rejected instead of retained in review data", () => {
  const paths: (string | number)[][] = [
    [],
    ["selection"],
    ["records", 0],
    ["records", 0, "target"],
    ["records", 0, "authority"],
    ["records", 0, "dimension"],
  ];
  for (const path of paths)
    invalid("balances", (d) => {
      let node: any = d;
      for (const key of path) node = node[key];
      node.private_payload = "QA rejected";
    });
  for (const path of [
    ["records", 0, "consent"],
    ["records", 0, "holds", 0],
    ["records", 0, "amounts", 0],
  ])
    invalid("cases", (d) => {
      let node: any = d;
      for (const key of path) node = node[key];
      node.extra = "QA rejected";
    });
  invalid("payment-terms", (d) => {
    (d.records[0] as PaymentRecord).receipt = {
      ...(d.records[0] as PaymentRecord).receipt,
      extra: "QA rejected",
    } as PaymentRecord["receipt"];
  });
});
test("money and quantity dimensions remain disjoint; values stay uncomputed", () => {
  invalid("balances", (d) => {
    (d.records[0] as BalanceRecord).dimension = {
      kind: "money",
      currency: "USD",
      unit: "hours",
    } as unknown as BalanceRecord["dimension"];
  });
  invalid("balances", (d) => {
    (d.records[1] as BalanceRecord).dimension = {
      kind: "quantity",
      currency: "USD",
      unit: "hours",
    } as unknown as BalanceRecord["dimension"];
  });
  invalid("balances", (d) => {
    (d.records[1] as BalanceRecord).dimension = {
      kind: "quantity",
      currency: null,
      unit: "",
    };
  });
  const d = fresh();
  const r = d.records[0] as BalanceRecord;
  r.opening = "100";
  r.closing = "999999.0000";
  r.available = "-10";
  assert.equal(valid(d, request()), true);
  assert.equal(r.closing, "999999.0000");
  invalid("adjustments", (d) => {
    (d.records[0] as AdjustmentRecord).amounts.push({
      currency: "USD",
      value: "5",
    });
  });
});
test("decimal transport bounds preserve signed precision and reject binary or exponent values", () => {
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
  invalid("balances", (d) => {
    (d.records[0] as BalanceRecord).closing = 1 as unknown as string;
  });
});
test("public and internal evidence stay separate; correction links never manufacture consent", () => {
  const d = fresh("cases"),
    r = d.records[0] as CaseRecord;
  assert.equal(r.consent.state, "not_recorded");
  assert.equal(r.resolution_ref, "QA-CORRECTION-1");
  assert.notEqual(r.internal_note, r.public_reply);
  assert.equal(valid(d, request("cases")), true);
  assert.equal(r.consent.evidence_ref, null);
  invalid("cases", (d) => {
    (d.records[0] as CaseRecord).consent.state = "recorded";
  });
  const multiline = fresh("cases");
  (multiline.records[0] as CaseRecord).internal_note =
    "QA internal\nsecond line";
  assert.equal(valid(multiline, request("cases")), true);
  invalid("cases", (d) => {
    (d.records[0] as CaseRecord).public_reply = "x".repeat(4097);
  });
  invalid("cases", (d) => {
    (d.records[0] as CaseRecord).internal_note = "QA\u0000secret";
  });
});
test("applied adjustments require an original while incomplete drafts remain evidence only", () => {
  invalid("adjustments", (d) => {
    const r = d.records[0] as AdjustmentRecord;
    r.original_document_ref = null;
    r.original_claim_ref = null;
  });
  const d = fresh("adjustments"),
    draft = d.records[1] as AdjustmentRecord;
  assert.equal(draft.state, "draft");
  assert.equal(draft.original_claim_ref, null);
  assert.equal(draft.amounts[0].value, null);
  assert.equal(valid(d, request("adjustments")), true);
});
test("effective and released holds require bound target revision and confirmed effect", () => {
  for (const effect of ["effective", "released"] as const) {
    for (const field of [
      "reference",
      "revision",
      "target_ref",
      "observed_at",
    ] as const)
      invalid("cases", (d) => {
        const h = (d.records[1] as CaseRecord).holds[0];
        h.effect = effect;
        h[field] = null;
      });
    for (const external of ["pending", "unknown"] as const)
      invalid("cases", (d) => {
        const h = (d.records[1] as CaseRecord).holds[0];
        h.effect = effect;
        h.external_effect = external;
      });
  }
  const d = fresh("cases"),
    h = (d.records[0] as CaseRecord).holds[0];
  assert.equal(h.effect, "requested");
  assert.equal(h.external_effect, "pending");
  assert.equal(valid(d, request("cases")), true);
  assert.equal(h.effect, "requested");
});
test("known payment classifications require issued document and current authority receipts terms", () => {
  for (const field of [
    "original_document_ref",
    "term_ref",
    "term_label",
    "time_zone",
    "issue_date",
    "original_due_date",
    "effective_due_date",
  ] as const)
    paymentChange((r) => {
      r[field] = null;
    });
  for (const issuance of ["internal", "not_issued", "unknown"] as const)
    paymentChange((r) => {
      r.issuance = issuance;
    });
  for (const field of ["reference", "observed_at"] as const) {
    paymentChange((r) => {
      r.authority[field] = null;
    });
    paymentChange((r) => {
      r.receipt[field] = null;
    });
  }
  paymentChange((r) => {
    r.authority.kind = "unknown";
  });
  paymentChange((r) => {
    r.authority.freshness = "stale";
  });
  paymentChange((r) => {
    r.receipt.freshness = "stale";
  });
  paymentChange((r) => {
    r.receipt.state = "unknown";
  });
  paymentChange((r) => {
    r.receivable.value = null;
  });
  const d = fresh("payment-terms"),
    r = d.records[1] as PaymentRecord;
  assert.equal(r.overdue, "unknown");
  assert.equal(valid(d, request("payment-terms")), true);
});
test("remaining sign and exact zero gate known payment states without rounding or clock rules", () => {
  for (const value of ["0", "0.000", "-1", "-0.00000000000000000001"])
    paymentChange((r) => {
      r.receivable.value = value;
    });
  for (const overdue of ["on_time", "due_today", "fully_paid"] as const) {
    const d = fresh("payment-terms"),
      r = d.records[0] as PaymentRecord;
    r.overdue = overdue;
    r.receivable.value =
      overdue === "fully_paid" ? "-0.0000" : "0.00000000000000000001";
    assert.equal(valid(d, request("payment-terms")), true);
  }
  paymentChange((r) => {
    r.overdue = "fully_paid";
    r.receivable.value = "0.00000000000000000001";
  });
  paymentChange((r) => {
    r.overdue = "not_applicable";
  });
});
test("dates are real calendar values and timezones are supported source names, no due-date recomputation", () => {
  for (const value of [
    "2026-02-29",
    "2026-04-31",
    "0000-01-01",
    "2026-11-06T00:00:00Z",
  ])
    assert.equal(validDate(value), false);
  assert.equal(validDate("2024-02-29"), true);
  assert.equal(validTimeZone("Asia/Seoul"), true);
  assert.equal(validTimeZone("UTC"), true);
  for (const value of ["Mars/Olympus", "+09:00", "", "PST"])
    assert.equal(validTimeZone(value), false);
  paymentChange((r) => {
    r.effective_due_date = "2026-02-29";
  });
  paymentChange((r) => {
    r.time_zone = "Mars/Olympus";
  });
  const d = fresh("payment-terms"),
    r = d.records[0] as PaymentRecord;
  r.overdue = "on_time";
  r.effective_due_date = "2000-01-01";
  assert.equal(valid(d, request("payment-terms")), true);
  invalid("cases", (d) => {
    (d.records[0] as CaseRecord).received_at = "2026-02-29T00:00:00Z";
  });
  invalid("balances", (d) => {
    d.observed_at = "2026-11-06T24:00:00Z";
  });
});
test("pages require bounded rows canonical unique IDs and coherent exact or unknown totals", () => {
  invalid("balances", (d) => {
    d.records.push(d.records[0]);
    d.total = 3;
  });
  invalid("balances", (d) => {
    d.records[0].id = "https://example.test/row";
  });
  invalid("balances", (d) => {
    d.total = 3;
  });
  invalid("balances", (d) => {
    d.total = -1;
  });
  invalid("balances", (d) => {
    d.next_cursor = "next";
  });
  const d = fresh();
  d.total = null;
  assert.equal(valid(d, request()), true);
  d.records = Array.from({ length: 25 }, (_, i) => ({
    ...structuredClone(fixtures.balances.records[0]),
    id: `QA-PAGE-${i}`,
  }));
  d.next_cursor = "next";
  assert.equal(valid(d, request()), true);
  d.records.push({ ...structuredClone(d.records[0]), id: "QA-PAGE-26" });
  assert.equal(valid(d, request()), false);
});
test("transport is fixed GET no-store redirect-error and accepts only labelled synthetic data", async () => {
  await mock(
    async (url, init) => {
      assert.equal(String(url), "/api/credit-case-review?view=balances&q=");
      assert.equal(init?.method, "GET");
      assert.equal(init?.cache, "no-store");
      assert.equal(init?.redirect, "error");
      assert.deepEqual(init?.headers, { "X-HB-Local-Client": "1" });
      assert.equal(init?.body, undefined);
      return Response.json(fresh());
    },
    async () => {
      assert.equal((await load(request(), signal())).kind, "loaded");
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
test("404 and 501 remain unconnected and malformed UTF8 JSON redirects fail closed", async () => {
  for (const status of [404, 501])
    await mock(
      async () => new Response("QA unavailable", { status }),
      async () => {
        assert.deepEqual(await load(request(), signal()), {
          kind: "unconnected",
        });
      },
    );
  const redirected = Response.json(fresh());
  Object.defineProperty(redirected, "redirected", { value: true });
  for (const response of [
    new Response("{"),
    new Response(new Uint8Array([255])),
    redirected,
    Response.json({ ...fresh(), view: "cases" }),
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
test("actual streamed byte limit cannot be bypassed by missing or false content length", async () => {
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
test("safe failures expose only bounded error codes and request identifiers", async () => {
  await mock(
    async () =>
      Response.json(
        {
          error: "permission_denied",
          request_id: "QA-request-1",
          private_payload: "QA rejected",
        },
        { status: 403 },
      ),
    async () => {
      await assert.rejects(
        load(request(), signal()),
        (e) =>
          e instanceof ServiceError &&
          e.code === "permission_denied" &&
          e.requestId === "QA-request-1" &&
          e.message === "permission_denied",
      );
    },
  );
  await mock(
    async () =>
      Response.json(
        { error: "<QA private>", request_id: "x".repeat(129) },
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
      throw new Error("QA raw network body");
    },
    async () => {
      await assert.rejects(
        load(request(), signal()),
        error("service_unavailable"),
      );
    },
  );
});
test("invalid or aborted requests cannot publish and query identity is captured before awaiting", async () => {
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
});
