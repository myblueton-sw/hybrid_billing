import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { Locale } from "../../../../shared/types";
import type { OrgParty } from "../../../../service/admin";
import { o } from "../organizations/live-copy";
import { firstPage, type Paged } from "../organizations/live-model";
import { c } from "./live-copy";
import { currentView, type Choice } from "./live-model";
import { Named, Picker, Results, listed, resultPage, unitChoices } from "./choices";

const locales: Locale[] = ["en", "ko", "ar"];
const plain = (html: string) => html.replace(/<[^>]+>/g, " ").replace(/&#x27;/g, "'").replace(/&quot;/g, '"').replace(/&amp;/g, "&").replace(/\s+/g, " ");
const count = (html: string, part: string) => html.split(part).length - 1;
const parties: OrgParty[] = [
  { id: "le-a", name: "Alpha", kind: "legal_entity", parent: "", state: "active" },
  { id: "unit-b", name: "Beta unit", kind: "internal_unit", parent: "le-a", state: "active" },
  { id: "unit-a", name: "", kind: "internal_unit", parent: "le-a", state: "active" },
  { id: "unit-c", name: "Closed unit", kind: "internal_unit", parent: "le-a", state: "suspended" },
];
const rows: Choice[] = [{ id: "acct-1", name: "Account one" }, { id: "acct-2", name: "acct-2" }];
const ready = (change: Partial<Paged<Choice>> = {}): Paged<Choice> => ({ rows, next: "", stopped: false, status: "ready", error: null, ...change });
const results = (l: Locale, state: Paged<Choice>, shown: Choice[] = rows, value = "") =>
  renderToStaticMarkup(<Results l={l} id="o02-member" value={value} empty={c.noMember(l)} state={state} rows={shown} onChoose={() => undefined} onMore={() => undefined} onRetry={() => undefined} />);
const picker = (l: Locale, value: string, name = "") =>
  renderToStaticMarkup(<Picker l={l} id="o02-member" label={c.member(l)} hint={c.memberHint(l)} empty={c.noMember(l)} value={value} name={name} invalid={false} turn="1|2"
    search={async () => ({ rows: [], next: "", stopped: false })} onError={() => false} onChoose={() => undefined} onSeen={() => undefined} onFocus={() => undefined}><button type="button" id="by-id">id</button></Picker>);

test("internal units are offered for the hierarchy or before a kind is chosen, matched by name or identifier", () => {
  assert.deepEqual(unitChoices(parties, "", "").map((u) => u.id), ["unit-b", "unit-a"]);
  assert.deepEqual(unitChoices(parties, "hierarchy", "BETA").map((u) => u.id), ["unit-b"]);
  assert.deepEqual(unitChoices(parties, "hierarchy", "unit-a").map((u) => [u.id, u.name]), [["unit-a", "unit-a"]]);
  assert.deepEqual(unitChoices(parties, "hierarchy", "closed"), []);
  assert.deepEqual(unitChoices(parties, "hierarchy", "alpha"), []);
  for (const k of ["payer", "contract_pool", "allocation_pool", "view_group"] as const) assert.deepEqual(unitChoices(parties, k, ""), []);
});

test("units come first, nothing is listed twice and the excluded identifier is left out", () => {
  assert.deepEqual(listed([{ id: "unit-b", name: "Beta unit" }], [...rows, { id: "unit-b", name: "again" }, { id: "acct-1", name: "again" }], "acct-2").map((r) => [r.id, r.name]), [["unit-b", "Beta unit"], ["acct-1", "Account one"]]);
  assert.deepEqual(listed([], rows, "").map((r) => r.id), ["acct-1", "acct-2"]);
  assert.equal(resultPage, 10);
});

test("the present targets are loading, none of the kind, none on the day or listed", () => {
  assert.deepEqual([currentView(false, null), currentView(false, []), currentView(true, null), currentView(true, []), currentView(true, [{ id: "u", name: "U" }])], ["loading", "loading", "none_of_kind", "none", "targets"]);
});

test("results are buttons with name and identifier, and the chosen one is marked", () => {
  for (const l of locales) {
    const html = results(l, ready(), rows, "acct-2");
    const buttons = html.match(/<button[^>]*>/g) ?? [];
    assert.deepEqual(buttons.map((b) => [/data-id="([^"]+)"/.exec(b)?.[1], b.includes('aria-current="true"'), b.includes('type="button"'), b.includes('data-result="o02-member"')]), [["acct-1", false, true, true], ["acct-2", true, true, true]], l);
    assert.ok(!html.includes("<input") && html.includes(`<ul class="o02-results" aria-label="${c.results(l)}">`), l);
    assert.ok(html.includes("<bdi>Account one</bdi>") && html.includes('<bdi dir="ltr">acct-1</bdi>') && html.includes('<bdi dir="ltr" class="o02-id-name live-mono">acct-2</bdi>'), l);
    assert.ok(!html.includes("o02-more") && !html.includes('role="alert"'), l);
  }
});

test("searching, a failed search and no match are three different states", () => {
  for (const l of locales) {
    assert.equal(results(l, firstPage()), `<p role="status" class="live-meta">${c.searching(l)}</p>`, l);
    const failed = results(l, { ...firstPage(), status: "error" });
    assert.ok(failed.startsWith('<p role="alert">') && plain(failed).includes(c.searchFailed(l)) && plain(failed).includes(o.tryAgain(l)) && !plain(failed).includes(c.noMember(l)), l);
    assert.equal(results(l, ready({ rows: [] }), []), `<p role="status" class="live-note">${c.noMember(l)}</p>`, l);
  }
});

test("a further page and a stopped search can both be continued, and a stopped search with nothing found is not called no match", () => {
  for (const l of locales) {
    const more = results(l, ready({ next: "acct-2" }));
    assert.ok(plain(more).includes(c.showMore(l)) && !plain(more).includes(o.treeStopped(l)), l);
    const stopped = results(l, ready({ rows: [], next: "acct-9", stopped: true }), []);
    assert.ok(plain(stopped).includes(o.treeStopped(l)) && plain(stopped).includes(o.continueSearching(l)) && !plain(stopped).includes(c.noMember(l)) && !stopped.includes("data-id"), l);
    const busy = results(l, ready({ next: "acct-2", status: "more" }));
    assert.ok(busy.includes('aria-disabled="true"') && plain(busy).includes(o.loadingMore(l)) && count(busy, "data-id=") === 2, l);
    const failed = results(l, ready({ next: "acct-2", status: "moreFailed" }));
    assert.ok(failed.includes('role="alert"') && plain(failed).includes(o.nextFailed(l)) && count(failed, "data-id=") === 2, l);
  }
});

test("with nothing chosen the search is open; the term field is not a form and the search button does not submit", () => {
  for (const l of locales) {
    const html = picker(l, "");
    assert.ok(html.includes(`<label for="o02-member" id="o02-member-label">${c.member(l)}</label>`) && html.includes('<input id="o02-member" type="search"') && html.includes('aria-describedby="o02-member-hint"'), l);
    assert.ok(!html.includes("<form") && !html.includes('type="submit"') && html.includes(`<button type="button" class="live-secondary">${c.search(l)}</button>`), l);
    assert.ok(!html.includes("o02-chosen") && plain(html).includes(c.searching(l)) && html.includes('id="by-id"'), l);
  }
});

test("a chosen value is shown with its identifier and a Change button, and the search is closed", () => {
  for (const l of locales) {
    const html = picker(l, "acct-1", "Account one");
    assert.ok(html.includes('<p class="o02-chosen">') && html.includes("<bdi>Account one</bdi>") && html.includes('<bdi dir="ltr">acct-1</bdi>'), l);
    assert.ok(html.includes('<button type="button" id="o02-member" class="service-button-link o02-id-toggle" aria-labelledby="o02-member o02-member-label o02-member-name"') && plain(html).includes(o.change(l)) && !html.includes('type="search"') && !html.includes("data-id"), l);
    assert.ok(html.includes(`<span id="o02-member-label" class="o02-field-label">${c.member(l)}</span>`) && !html.includes("<label") && html.includes('<span class="o02-chosen-name" id="o02-member-name">'), l);
    assert.ok(picker(l, "acct-9").includes('<bdi dir="ltr" class="o02-id-name live-mono">acct-9</bdi>'), l);
  }
  assert.equal(renderToStaticMarkup(<Named id="a" name="a" />), '<bdi dir="ltr" class="o02-id-name live-mono">a</bdi>');
});

test("the new sentences exist in three languages", () => {
  const all = (l: Locale) => [c.memberHint(l), c.search(l), c.searching(l), c.searchFailed(l), c.results(l), c.showMore(l), c.noMember(l), c.noTarget(l), c.loadingCurrent(l), c.choicesFailed(l), c.pick(l)];
  assert.equal(new Set(locales.flatMap(all)).size, 33);
});
