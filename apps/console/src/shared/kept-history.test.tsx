import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import { merged, useOlder } from "./KeptHistory";
import { kept } from "./kept-copy";
import { keptFactsOf, keptPageOf, type KeptFacts } from "../service/kept";
import { AdminError, adminClient } from "../service/admin";

type Row = { number: number | null; state: string };
const rowsOf = (from: number, n: number, state = "applied"): Row[] => Array.from({ length: n }, (_, i) => ({ number: from - i, state }));

function Probe({ facts, first, l = "en" as const }: { facts: KeptFacts | null; first: Row[]; l?: "en" | "ko" | "ar" }) {
  const older = useOlder<Row>(l, "r1", facts, first, async () => ({ rows: [], next: "", matching: 0, history: facts ?? keptFactsOf(null) }), () => false);
  return <div data-rows={older.rows.length}>{older.head}{older.count}{older.foot}</div>;
}

test("nothing is said when every kept proposal is listed", () => {
  const html = renderToStaticMarkup(<Probe facts={{ total: 7, next: "", kept: 200, dropped: 0 }} first={rowsOf(7, 7)} />);
  assert.equal(html, '<div data-rows="7"></div>');
  assert.equal(renderToStaticMarkup(<Probe facts={null} first={rowsOf(3, 3)} />), '<div data-rows="3"></div>');
});

test("a cut list states the count and offers the older proposals", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    const html = renderToStaticMarkup(<Probe l={l} facts={{ total: 120, next: "71", kept: 200, dropped: 5 }} first={rowsOf(120, 50)} />);
    assert.ok(html.includes(kept.shown(l, 50, 120)), l);
    assert.ok(html.includes(kept.more(l)), l);
    assert.ok(!html.includes("kept-end"), l);
  }
});

test("the end sentence appears only where nothing further can be read and proposals were dropped", () => {
  const html = renderToStaticMarkup(<Probe facts={{ total: 200, next: "", kept: 200, dropped: 12 }} first={rowsOf(200, 200)} />);
  assert.ok(html.includes(kept.end("en", 200)) && !html.includes("kept-more") && !html.includes("kept-count"));
  const waiting = renderToStaticMarkup(<Probe facts={{ total: 3, next: "", kept: 200, dropped: 0 }} first={[...rowsOf(9, 4, "pending"), ...rowsOf(5, 3)]} />);
  assert.ok(!waiting.includes("kept-count"));
  const kept0 = renderToStaticMarkup(<Probe facts={{ total: 50, next: "", kept: 200, dropped: 0 }} first={rowsOf(50, 50)} />);
  assert.ok(!kept0.includes("kept-end"));
});

test("a total smaller than the rows listed is not shown as a smaller total", () => {
  const html = renderToStaticMarkup(<Probe facts={{ total: 40, next: "9", kept: 200, dropped: 0 }} first={rowsOf(50, 50)} />);
  assert.ok(!html.includes("kept-count") && html.includes(kept.more("en")));
});

test("the count is of closed proposals and leaves out those waiting", () => {
  const html = renderToStaticMarkup(<Probe facts={{ total: 120, next: "74", kept: 200, dropped: 0 }} first={[...rowsOf(300, 3, "pending"), ...rowsOf(120, 47)]} />);
  assert.ok(html.includes(kept.shown("en", 47, 120)));
});

test("older rows already listed are not listed twice", () => {
  const got = merged(rowsOf(10, 5), rowsOf(7, 4));
  assert.deepEqual(got.map((r) => r.number), [10, 9, 8, 7, 6, 5, 4]);
});

test("the facts and a further page are read strictly", () => {
  assert.deepEqual(keptFactsOf({ total: "5", next: 7, kept: -1, dropped: 1.5 }), { total: 0, next: "", kept: 0, dropped: 0 });
  assert.deepEqual(keptFactsOf([1]), { total: 0, next: "", kept: 0, dropped: 0 });
  assert.equal(keptPageOf({ proposals: "x", next: "" }, (r) => r), null);
  assert.equal(keptPageOf({ proposals: [], next: 4 }, (r) => r), null);
  const page = keptPageOf({ proposals: [{ number: 3 }, null, "x", [1]], next: "3", matching: 9, proposal_history: { total: 9, next: "", kept: 200, dropped: 1 } }, (r) => ({ number: r.number as number, state: "applied" }));
  assert.deepEqual(page, { rows: [{ number: 3, state: "applied" }], next: "3", matching: 9, history: { total: 9, next: "", kept: 200, dropped: 1 } });
});

test("the three further reads ask for the page after the cursor", async () => {
  const calls: string[] = [];
  const f = (async (url: RequestInfo | URL) => {
    calls.push(String(url));
    return new Response(JSON.stringify({ proposals: [{ number: 70, target: "x-70", state: "applied" }], next: "21", matching: 120, proposal_history: { total: 120, next: "", kept: 200, dropped: 5 } }), { status: 200, headers: { "Content-Type": "application/json" } });
  }) as typeof fetch;
  const api = adminClient("t", f);
  const a = await api.installationProposals("71");
  const b = await api.integrationProposals("71");
  const c = await api.recoveryProposals("7 1");
  assert.deepEqual(calls, ["/api/v1/installation/proposals?limit=50&after=71", "/api/v1/integrations/proposals?limit=50&after=71", "/api/v1/recovery/proposals?limit=50&after=7%201"]);
  assert.deepEqual([a.rows[0].number, b.rows[0].number, c.rows[0].number, a.next, b.history.total, c.matching], [70, 70, 70, "21", 120, 120]);
  const bad = (async () => new Response(JSON.stringify({ proposals: [] }), { status: 200, headers: { "Content-Type": "application/json" } })) as typeof fetch;
  await assert.rejects(adminClient("t", bad).installationProposals("71"), (e: AdminError) => e.failure === "invalid");
});
