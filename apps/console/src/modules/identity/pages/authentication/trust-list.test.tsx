import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import type { ListedTrustChange } from "../../../../service/trust-changes";
import { trustChangePageOf } from "../../../../service/trust-changes";
import { trustWire } from "../../../../service/trust-changes.sample";
import { ty } from "./trust-copy";
import { tl } from "./trust-list-copy";
import { byStatePriority, evidenceLeftToDetail, filtered, listedBlocked, listedNext, noFilter, searchFits, sectionState, wholeOnOnePage } from "./trust-list-model";
import { ChangeRows } from "./TrustLists";

const copy = (v: unknown) => JSON.parse(JSON.stringify(v));
const open = trustChangePageOf(copy(trustWire.open));
const closed = trustChangePageOf(copy(trustWire.closed));
const held = trustChangePageOf(copy(trustWire.held));
const first = trustChangePageOf(copy(trustWire.list));
const row = (r: ListedTrustChange, change: Partial<ListedTrustChange>): ListedTrustChange => ({ ...r, ...change });
const noop = () => undefined;

test("M02 lists: each section reads its group, or the one state that belongs to it", () => {
  assert.deepEqual([sectionState(noFilter, "open"), sectionState(noFilter, "closed")], ["open", "closed"]);
  assert.deepEqual([sectionState({ q: "x", state: "" }, "open"), sectionState({ q: "x", state: "" }, "closed")], ["open", "closed"]);
  assert.deepEqual([sectionState({ q: "", state: "validated" }, "open"), sectionState({ q: "", state: "validated" }, "closed")], ["validated", null]);
  assert.deepEqual([sectionState({ q: "", state: "activation_unknown" }, "open"), sectionState({ q: "", state: "activation_unknown" }, "closed")], ["activation_unknown", null]);
  assert.deepEqual([sectionState({ q: "", state: "superseded" }, "open"), sectionState({ q: "", state: "superseded" }, "closed")], [null, "superseded"]);
  assert.deepEqual([sectionState({ q: "", state: "open" }, "open"), sectionState({ q: "", state: "open" }, "closed")], ["open", null]);
  assert.deepEqual([sectionState({ q: "", state: "closed" }, "open"), sectionState({ q: "", state: "closed" }, "closed")], [null, "closed"]);
  assert.deepEqual([searchFits("a".repeat(64)), searchFits("a".repeat(65)), searchFits(" " + "a".repeat(64) + " "), searchFits("\uac00".repeat(21)), searchFits("\uac00".repeat(22))], [true, false, true, true, false]);
  assert.deepEqual([filtered(noFilter), filtered({ q: "a", state: "" }), filtered({ q: "", state: "draft" })], [false, true, true]);
});

test("M02 lists: a group on one page is sorted by state, a group over pages is left as the service sent it", () => {
  assert.equal(wholeOnOnePage(open, true), true);
  assert.equal(wholeOnOnePage(first, true), false);
  assert.equal(wholeOnOnePage(open, false), false);
  const mixed = [row(open.rows[0], { id: "trust-a-1", number: 1, state: "draft" }), row(open.rows[0], { id: "trust-a-2", number: 2, state: "activation_unknown" }),
    row(open.rows[0], { id: "trust-a-3", number: 3, state: "draft" }), row(open.rows[0], { id: "trust-a-4", number: 4, state: "validated" }),
    row(open.rows[0], { id: "trust-a-5", number: 5, state: "independently_approved" }), row(open.rows[0], { id: "trust-b-3", number: 3, state: "draft" })];
  assert.deepEqual(byStatePriority(mixed).map((r) => r.id), ["trust-a-2", "trust-a-5", "trust-a-4", "trust-b-3", "trust-a-3", "trust-a-1"]);
  assert.equal(mixed[0].id, "trust-a-1");
});

test("M02 lists: the next step comes from the row's own facts and never says the evidence changed", () => {
  const validated = open.rows.find((r) => r.state === "validated")!;
  const draft = open.rows.find((r) => r.state === "draft")!;
  const activated = closed.rows.find((r) => r.state === "activated" && !r.earlier_run)!;
  assert.equal(listedNext("en", validated), ty.next.validated("en"));
  assert.equal(listedNext("en", draft), ty.next.draft("en"));
  assert.equal(listedNext("en", row(draft, { validation_result: "blocked" })), ty.next.blocked("en"));
  assert.equal(listedBlocked(row(draft, { validation_result: "blocked" })), true);
  assert.equal(listedNext("en", held.rows[0]), ty.next.unknown("en"));
  assert.equal(listedNext("en", activated), ty.next.activated("en"));
  assert.equal(listedNext("en", row(activated, { invalidation_state: "in_progress" })), ty.next.progress("en"));
  assert.equal(listedNext("en", closed.rows.find((r) => r.earlier_run)!), ty.next.activated("en"));
  assert.equal(listedNext("en", closed.rows.find((r) => r.state === "superseded")!), ty.next.closed("en"));
  for (const r of [...open.rows, ...closed.rows, ...held.rows]) assert.notEqual(listedNext("en", r), ty.next.stale("en"));
  assert.equal(evidenceLeftToDetail(open.rows), true);
  assert.equal(evidenceLeftToDetail(open.rows.filter((r) => r.state === "draft")), false);
  assert.equal(evidenceLeftToDetail(closed.rows), false);
  assert.equal(evidenceLeftToDetail(held.rows), false);
});

test("M02 lists: rows show the change, its state, proposer, what it is based on and the next step", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    const out = renderToStaticMarkup(<ChangeRows l={l} rows={open.rows} title={ty.open(l)} selected={open.rows[1].id} reduced={false} onOpen={noop} />);
    for (const r of open.rows) {
      assert.ok(out.includes(`data-change="${r.id}"`) && out.includes(`>${ty.change(l, r.number)}</button>`), r.id);
      assert.ok(out.includes(`>${ty.state(l, r.state)}</span>`));
      assert.ok(out.includes(listedNext(l, r)));
    }
    assert.ok(out.includes('<bdi dir="ltr" class="live-mono">0.2</bdi>') && out.includes(ty.policyNumber(l, 2)));
    assert.ok(out.includes('<bdi dir="ltr" class="live-mono m02-actor">sample-admin</bdi>'));
    assert.equal(out.split('aria-current="true"').length, 2);
    assert.equal(out.split(`>${ty.selected(l)}<`).length, 2);
    assert.equal(out.split("<th ").length, 6);
  }
  const reduced = renderToStaticMarkup(<ChangeRows l="en" rows={open.rows} title="Open" selected="" reduced onOpen={noop} />);
  assert.equal(reduced.split('<td role="cell"').length, open.rows.length + 1);
  assert.ok(reduced.includes("m02-reduced") && !reduced.includes("<colgroup>"));
  const past = renderToStaticMarkup(<ChangeRows l="en" rows={closed.rows} title="Closed" selected="" reduced={false} onOpen={noop} />);
  assert.ok(past.includes(ty.notRecorded("en")) && past.includes("From an earlier server run") && past.includes('<bdi dir="ltr" class="live-mono">0.0</bdi>'));
  const waiting = renderToStaticMarkup(<ChangeRows l="en" rows={held.rows} title="Open" selected="" reduced={false} onOpen={noop} />);
  assert.ok(waiting.includes('class="m02-state-word live-strong"'));
  const calm = renderToStaticMarkup(<ChangeRows l="en" rows={open.rows.filter((r) => r.state === "validated")} title="Open" selected="" reduced={false} onOpen={noop} />);
  assert.ok(!calm.includes("live-strong"));
});

test("M02 lists: the new sentences exist in three languages", () => {
  for (const make of Object.values(tl)) {
    const [en, ko, ar] = [make("en"), make("ko"), make("ar")];
    assert.ok(en && ko && ar && en !== ko && en !== ar && ko !== ar);
    assert.ok(/[가-힣]/.test(ko) && /[؀-ۿ]/.test(ar));
  }
});
