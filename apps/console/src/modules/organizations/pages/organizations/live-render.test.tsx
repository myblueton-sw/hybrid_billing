import { test } from "node:test";
import assert from "node:assert/strict";
import * as nodeModule from "node:module";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { Locale } from "../../../../shared/types";
import type { OrgParty } from "../../../../service/admin";
import { o } from "./live-copy";
import { noFilters, type Filters } from "./live-model";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (assetStub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { FilterBand, FilteredEmpty, MoreBar, chipText, countText } = await import("./filters");

const locales: Locale[] = ["en", "ko", "ar"];
const noop = () => undefined;
const entities: OrgParty[] = [
  { id: "le-north", name: "North", kind: "legal_entity", parent: "", state: "active" },
  { id: "le-old", name: "", kind: "legal_entity", parent: "", state: "suspended" },
];
const plain = (html: string) => html.replace(/<[^>]+>/g, " ").replace(/&#x27;/g, "'").replace(/&quot;/g, '"').replace(/&amp;/g, "&").replace(/\s+/g, " ");
const band = (l: Locale, view: string, filters: Filters = noFilters, wide = true, open = false) =>
  renderToStaticMarkup(<FilterBand l={l} view={view} wide={wide} open={open} onToggle={noop} filters={filters} entities={entities} onChange={noop} onRemove={noop} onClearAll={noop}
    groupKind="view_group" onGroupKind={noop} history={<p id="history-slot">slot</p>} />);

test("the accounts band has four controls and states the provider filter as text in every language", () => {
  for (const l of locales) {
    const html = band(l, "accounts");
    const text = plain(html);
    for (const id of ['id="o01-q"', 'id="o01-entity"', 'id="o01-kind"', 'id="o01-ua"', 'role="search"', 'maxLength="64"', 'dir="auto"']) assert.ok(html.includes(id), `${l} ${id}`);
    assert.ok(text.includes(o.provider(l)) && text.includes(o.providerNote(l)), l);
    assert.ok(!/<p class="o01-provider">.*?(<select|<input|<button)/.test(html), l);
    assert.ok(html.includes(`<h2 id="o01-filters-title">${o.filters(l)}</h2>`), l);
    assert.ok(!html.includes("o01-applied") && !html.includes("o01-filters-toggle"), l);
    assert.ok(text.includes(o.allEntities(l)) && text.includes(`${o.stateSuspended(l)}`), l);
    assert.equal((html.match(/<option value="/g) ?? []).length, 3 + 5, l);
  }
});

test("chips appear only for applied filters, carry a removal name and keep IDs isolated", () => {
  for (const l of locales) {
    const f = { entity: "le-north", rk: "payer" as const, ua: true, q: "acct-1" };
    const html = band(l, "accounts", f);
    for (const k of ["q", "entity", "rk", "ua"]) assert.ok(html.includes(`id="o01-chip-${k}"`), `${l} ${k}`);
    assert.ok(html.includes(`aria-label="${o.removeFilter(l, chipText(l, "accounts", "entity", f, entities)).replace(/'/g, "&#x27;")}"`), l);
    assert.ok(html.includes('class="icon"') && !html.includes("×"), l);
    assert.ok(html.includes(`<bdi>${o.chip(o.searchButton(l), "acct-1")}</bdi>`), l);
    assert.ok(plain(html).includes(o.clearAll(l)) && plain(html).includes(o.appliedFilters(l)), l);
    assert.equal(chipText(l, "accounts", "rk", f, entities), o.chip(o.relKind(l), o.kind(l, "payer")));
    assert.equal(chipText(l, "accounts", "ua", f, entities), o.onlyUnattributed(l));
    assert.equal(chipText(l, "groups", "entity", f, entities), o.membersChip(l, "North"));
    assert.equal(chipText(l, "groups", "q", f, entities), o.groupsChip(l, "acct-1"));
    assert.equal(chipText(l, "accounts", "entity", { ...f, entity: "le-gone" }, entities), o.chip(o.entity(l), "le-gone"));
    assert.equal(chipText(l, "accounts", "entity", { ...f, entity: "le-old" }, entities), o.chip(o.entity(l), `le-old · ${o.stateSuspended(l)}`));
  }
  assert.ok(/<option value="le-gone"[^>]*>le-gone<\/option>/.test(band("en", "accounts", { ...noFilters, entity: "le-gone" })));
});

test("below 1280 the band is a toggle with the applied count and the chips stay outside the body", () => {
  for (const l of locales) {
    const closed = band(l, "accounts", { ...noFilters, ua: true }, false, false);
    assert.ok(closed.includes('aria-expanded="false"') && closed.includes('aria-controls="o01-filter-body"') && closed.includes(o.filtersApplied(l, 1)), l);
    assert.ok(/id="o01-filter-body"[^>]*hidden=""/.test(closed), l);
    assert.ok(closed.indexOf('class="o01-applied"') > closed.indexOf("</div>", closed.indexOf('id="o01-filter-body"')), l);
    const opened = band(l, "accounts", noFilters, false, true);
    assert.ok(opened.includes('aria-expanded="true"') && !/id="o01-filter-body"[^>]*hidden/.test(opened) && opened.includes(`<span>${o.filters(l)}</span>`), l);
  }
});

test("the groups, history and tree bands carry only what applies there", () => {
  for (const l of locales) {
    const groups = band(l, "groups", { ...noFilters, rk: "payer" });
    assert.ok(groups.includes('name="o01-group-kind"') && plain(groups).includes(o.searchGroups(l)) && plain(groups).includes(o.filterMembers(l)), l);
    assert.ok(!groups.includes('id="o01-kind"') && groups.includes(o.providerNote(l)) && !groups.includes("o01-chip-rk"), l);
    const groupsUa = band(l, "groups", { ...noFilters, ua: true, entity: "le-north" });
    assert.ok(!groupsUa.includes('id="o01-ua"') && !groupsUa.includes("o01-chip-ua") && groupsUa.includes("o01-chip-entity"), l);
    assert.ok(groups.indexOf("o01-group-kind") < groups.indexOf('id="o01-q"') && groups.indexOf('id="o01-q"') < groups.indexOf('id="o01-entity"'), l);
    const history = band(l, "history", { ...noFilters, entity: "le-north", ua: true, rk: "payer" });
    assert.ok(history.includes('id="o01-kind"') && history.includes('id="history-slot"') && !history.includes('id="o01-entity"') && !history.includes('id="o01-ua"') && !history.includes(o.providerNote(l)), l);
    assert.ok(history.includes("o01-chip-rk") && !history.includes("o01-chip-entity"), l);
    const tree = band(l, "tree", { ...noFilters, entity: "le-north" });
    assert.ok(plain(tree).includes(o.treeFilters(l)) && !tree.includes("<select") && !tree.includes("o01-chip") && !tree.includes("o01-filters-toggle"), l);
  }
});

test("paging controls state certainty, scan stops and failures in text", () => {
  for (const l of locales) {
    assert.equal(renderToStaticMarkup(<MoreBar l={l} next="" stopped={false} status="ready" count="" compact={false} onMore={noop} />), "");
    const more = renderToStaticMarkup(<MoreBar l={l} next="m-9" stopped={false} status="ready" count={countText(l, { n: 50, more: true })} compact onMore={noop} />);
    assert.ok(plain(more).includes(o.loadMore(l)) && plain(more).includes(o.shownMore(l, 50)), l);
    const busy = renderToStaticMarkup(<MoreBar l={l} next="m-9" stopped={false} status="more" count="" compact={false} onMore={noop} />);
    assert.ok(busy.includes('aria-disabled="true"') && plain(busy).includes(o.loadingMore(l)), l);
    const stopped = renderToStaticMarkup(<MoreBar l={l} next="m-9" stopped status="ready" count="" compact={false} onMore={noop} />);
    assert.ok(stopped.includes('role="status"') && stopped.includes('<bdi dir="ltr" class="live-mono">m-9</bdi>') && plain(stopped).includes(o.continueSearching(l)), l);
    assert.ok(stopped.replace(/<[^>]+>/g, "").includes(o.searchedThrough(l).replace("{id}", "m-9")), l);
    const failed = renderToStaticMarkup(<MoreBar l={l} next="m-9" stopped={false} status="moreFailed" count="" compact={false} onMore={noop} />);
    assert.ok(failed.includes('role="alert"') && plain(failed).includes(o.nextFailed(l)) && plain(failed).includes(o.tryAgain(l)), l);
    assert.equal(countText(l, { n: 3, more: false }), o.shownAll(l, 3));
    assert.equal(countText(l, { n: 0, more: false }), "");
    assert.equal(countText(l, { n: 0, more: true }, true), "");
    assert.equal(countText(l, { n: 3, more: true }, true), o.groupsShownMore(l, 3));
    assert.equal(countText(l, { n: 3, more: false }, true), o.groupsShownAll(l, 3));
    for (const s of [o.shownMore(l, 7), o.shownAll(l, 7), o.groupsShownMore(l, 7)]) assert.ok(!/ of \d|%|\/ /.test(s), s);
  }
});

test("the filtered empty state names the scope and offers to clear the filters", () => {
  for (const l of locales) {
    const html = renderToStaticMarkup(<FilteredEmpty l={l} date="Nov 16, 2026" onClear={noop} />);
    assert.ok(plain(html).includes(o.filteredEmpty(l, "Nov 16, 2026")) && plain(html).includes(o.clearAll(l)), l);
    assert.notEqual(o.filteredEmpty(l, "d"), o.noAccounts(l));
  }
});

const { PlanActions, planMember, planNode } = await import("./Live");
const bare = (html: string) => html.replace(/<[^>]+>/g, "").replace(/&#x27;/g, "'").replace(/&quot;/g, '"').replace(/&amp;/g, "&");
const actions = (l: Locale, plan: { kind: "hierarchy" | "payer" | "view_group"; member: string; base: number } | null, on: string, child = true) =>
  renderToStaticMarkup(<PlanActions l={l} plan={plan} on={on} today="2026-11-18" lang={l}>{child && <button type="button" id="history">{o.viewHistory(l)}</button>}</PlanActions>);

test("Create change plan is a link before View history carrying kind, member, date and revision only", () => {
  for (const l of locales) {
    const html = actions(l, { kind: "payer", member: "acct-1", base: 3 }, "2026-12-01");
    assert.ok(html.includes(`<a id="o01-create-plan" class="live-secondary o01-plan-link" href="#/organization-changes?view=targets&amp;kind=payer&amp;member=acct-1&amp;from=2026-12-01&amp;base=3&amp;lang=${l}" aria-describedby="o01-plan-hint">${o.createPlan(l)}</a>`), l);
    assert.ok(html.indexOf("o01-create-plan") < html.indexOf('id="history"'), l);
    assert.ok(html.includes('<p id="o01-plan-hint" class="live-note o01-plan-hint">'), l);
    assert.ok(html.includes('<bdi dir="ltr">3</bdi>') && html.includes('<time dateTime="2026-12-01">'), l);
    assert.ok(bare(html).includes(bare(o.planHint(l, o.kind(l, "payer"), "3", "x")).split("x")[0]), l);
    assert.ok(!html.includes("Account") && !html.includes("disabled"), l);
  }
  assert.equal(o.createPlan("ko"), "변경 계획 만들기");
});

test("a past as-of date is not carried and the hint asks for the date in O02", () => {
  for (const l of locales) {
    const html = actions(l, { kind: "hierarchy", member: "unit-1", base: 1 }, "2026-11-17");
    assert.ok(html.includes("href=\"#/organization-changes?view=targets&amp;kind=hierarchy&amp;member=unit-1&amp;base=1&amp;lang=" + l + '"'), l);
    assert.ok(bare(html).includes(bare(o.planHintNoDate(l, o.kind(l, "hierarchy"), "1"))), l);
    assert.ok(!html.includes("<time"), l);
  }
  assert.ok(actions("en", { kind: "view_group", member: "a", base: 2 }, "2026-11-18").includes("from=2026-11-18"));
});

test("without a plan target the action and its hint are absent, never disabled", () => {
  for (const l of locales) {
    const html = actions(l, null, "2026-12-01");
    assert.ok(!html.includes("o01-create-plan") && !html.includes("o01-plan-hint") && html.includes('id="history"'), l);
    assert.equal(actions(l, null, "2026-12-01", false), "", l);
    const alone = actions(l, { kind: "view_group", member: "acct-3", base: 1 }, "2026-12-01", false);
    assert.ok(alone.startsWith('<div class="live-buttons"><a id="o01-create-plan"'), l);
  }
});

test("in the tree only loaded units and accounts can be a hierarchy member, never a legal entity", () => {
  const leaf = { id: "acct-1", name: "A", kind: "account" as const, state: "", source: "relationship" as const, children: [] };
  const unit = { id: "unit-1", name: "U", kind: "internal_unit" as const, state: "active", source: "registry" as const, children: [leaf] };
  const root = { id: "le-1", name: "L", kind: "legal_entity" as const, state: "active", source: "none" as const, children: [unit] };
  const odd = { id: "x-1", name: "x-1", kind: "unknown" as const, state: "", source: "none" as const, children: [] };
  assert.equal(planNode([root, odd], "acct-1"), leaf);
  assert.equal(planNode([root], "nope"), undefined);
  assert.deepEqual([root, unit, leaf, odd, undefined].map(planMember), [false, true, true, false, false]);
});
