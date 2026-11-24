import { test } from "node:test";
import assert from "node:assert/strict";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError, adminClient, type OrgTreeNode, type OrgTreePage } from "../../../../service/admin";
import { o } from "./live-copy";
import { firstPage, type Paged } from "./live-model";
import { BranchView, Branch, TreeGroup, canOpen, distinct, kindName, parentList, planMember, unplaced, type TreeContext } from "./tree";

const locales: Locale[] = ["en", "ko", "ar"];
const plain = (html: string) => html.replace(/<[^>]+>/g, " ").replace(/&#x27;/g, "'").replace(/&quot;/g, '"').replace(/&amp;/g, "&").replace(/\s+/g, " ");
const node = (id: string, change: Partial<OrgTreeNode> = {}): OrgTreeNode => ({ id, name: "", kind: "account", state: "", source: "relationship", children: "none", ...change });
const context = (l: Locale, change: Partial<TreeContext> = {}): TreeContext =>
  ({ l, date: "Nov 16, 2026", turn: "0|2026-11-16", collapse: 0, selected: "", read: async () => ({ rows: [], next: "", stopped: false }), onError: () => false, onSelect: () => undefined, ...change });
const ready = (rows: OrgTreeNode[], change: Partial<Paged<OrgTreeNode>> = {}): Paged<OrgTreeNode> => ({ rows, next: "", stopped: false, status: "ready", error: null, ...change });
const view = (l: Locale, state: Paged<OrgTreeNode>, props: { depth?: number; path?: string[]; name?: string; labelled?: boolean; empty?: React.ReactNode; selected?: string } = {}) =>
  renderToStaticMarkup(<BranchView c={context(l, { selected: props.selected ?? "" })} list="roots" depth={props.depth ?? 0} path={props.path ?? []} name={props.name} labelled={props.labelled} empty={props.empty}
    state={state} onMore={() => undefined} onRetry={() => undefined} />);
const count = (html: string, part: string) => html.split(part).length - 1;

test("the tree read builds its query and refuses an answer it does not know", async () => {
  const page: OrgTreePage = { revision: "r", relationship_revision: 2, active_revision: 3, on: "2026-11-16", parent: "unit-1", group: "",
    nodes: [node("acct-1"), node("le-1", { kind: "legal_entity", state: "active", source: "none", children: "some" }), node("x", { kind: "unknown", source: "none", children: "unchecked" }), node("u", { kind: "internal_unit", source: "registry" })],
    next: "m:acct-1", scan_stopped: true, registry_empty: false };
  const calls: string[] = [];
  const answer = (body: unknown) => (async (url: string) => { calls.push(url); return new Response(JSON.stringify(body), { status: 200 }); }) as unknown as typeof fetch;
  assert.deepEqual(await adminClient("t", answer(page)).tree({ on: "2026-11-16", revision: 0, parent: "unit-1", after: "m:acct 0", limit: 50 }), page);
  assert.equal(calls[0], "/api/v1/organizations/tree?on=2026-11-16&revision=0&parent=unit-1&after=m%3Aacct+0&limit=50");
  await adminClient("t", answer(page)).tree({ group: "detached" });
  await adminClient("t", answer(page)).tree();
  assert.deepEqual(calls.slice(1), ["/api/v1/organizations/tree?group=detached", "/api/v1/organizations/tree"]);
  const invalid = (e: AdminError) => e.failure === "invalid";
  for (const bad of [{}, { ...page, nodes: undefined }, { ...page, nodes: [node("a", { children: "many" as never })] }, { ...page, nodes: [node("a", { kind: "group" as never })] }, { ...page, nodes: [node("a", { source: "other" as never })] },
    { ...page, nodes: [{ id: "a" }] }, { ...page, next: undefined }, { ...page, scan_stopped: "no" }, { ...page, registry_empty: undefined }, { ...page, relationship_revision: "2" }]) {
    await assert.rejects(adminClient("t", answer(bad)).tree(), invalid, JSON.stringify(bad).slice(0, 80));
  }
});

test("a node can be opened unless it has nothing under it or is already on its own path", () => {
  assert.equal(canOpen(node("a", { children: "some" }), []), true);
  assert.equal(canOpen(node("a", { children: "unchecked" }), ["b"]), true);
  assert.equal(canOpen(node("a", { children: "none" }), []), false);
  assert.equal(canOpen(node("a", { children: "some" }), ["b", "a"]), false);
  assert.deepEqual(distinct([node("a"), node("b"), node("a", { name: "again" })]).map((n) => [n.id, n.name]), [["a", ""], ["b", ""]]);
  assert.equal(parentList, "parent:");
});

test("only an internal unit or a member that is not a party can be given a hierarchy plan", () => {
  assert.deepEqual([planMember("internal_unit", false), planMember("internal_unit", true), planMember("legal_entity", true), planMember("legal_entity", false), planMember(undefined, true), planMember(undefined, false)],
    [true, true, false, false, true, false]);
});

test("a member with no organization becomes a row whose children are not known", () => {
  const parties = [{ id: "unit-2", name: "Unit two", kind: "internal_unit" as const, parent: "le-a", state: "suspended" }];
  assert.deepEqual(unplaced({ member: "unit-2", name: "Unit two", targets: [], history: [] }, parties), { id: "unit-2", name: "Unit two", kind: "internal_unit", state: "suspended", source: "none", children: "unchecked" });
  assert.deepEqual(unplaced({ member: "acct-3", name: "", targets: [], history: [] }, parties), { id: "acct-3", name: "", kind: "account", state: "", source: "none", children: "unchecked" });
});

test("rows show their kind, state and source, and only openable rows have a toggle", () => {
  for (const l of locales) {
    const html = view(l, ready([
      node("le-1", { name: "North", kind: "legal_entity", state: "active", source: "none", children: "some" }),
      node("unit-1", { kind: "internal_unit", state: "suspended", source: "registry", children: "unchecked" }),
      node("acct-1"),
      node("ghost", { kind: "unknown", source: "none", children: "some" }),
    ]), { labelled: true, selected: "unit-1" });
    assert.equal(count(html, 'class="o01-toggle"'), 3, l);
    assert.equal(count(html, 'class="o01-toggle-space"'), 1, l);
    assert.equal(count(html, 'aria-expanded="false"'), 3, l);
    assert.ok(html.includes(`<ul class="o01-tree" aria-label="${o.treeTitle(l)}">`), l);
    assert.ok(html.includes(`aria-label="${o.expand(l, "North")}"`) && html.includes(`aria-label="${o.expand(l, "unit-1")}"`), l);
    assert.ok(html.includes('class="live-open o01-name o01-strong" data-id="le-1"') && html.includes('data-id="unit-1" aria-current="true"'), l);
    assert.equal(count(html, 'aria-current="true"'), 1, l);
    const text = plain(html);
    for (const part of [kindName(l, "legal_entity"), `${kindName(l, "internal_unit")} · ${o.stateSuspended(l)} · ${o.registryParent(l)}`, kindName(l, "account"), kindName(l, "unknown")]) assert.ok(text.includes(part), `${l} ${part}`);
    assert.ok(html.includes('<bdi dir="ltr">acct-1</bdi>') && !text.includes("undefined") && !html.includes("o01-more"), l);
  }
});

test("a single root starts open and loads what is under it; several roots start closed", () => {
  const one = view("en", ready([node("le-1", { name: "North", kind: "legal_entity", children: "some" })]), { labelled: true });
  assert.ok(one.includes('aria-expanded="true"') && plain(one).replace(/\s+/g, "").includes(o.loadingUnder("en", "North").replace(/\s+/g, "")) && one.includes(`aria-label="${o.collapse("en", "North")}"`));
  assert.ok(/aria-controls="([^"]+)"/.test(one) && one.includes(`<div id="${/aria-controls="([^"]+)"/.exec(one)?.[1]}" class="o01-under"`));
  const more = view("en", ready([node("le-1", { children: "some" })], { next: "p:le-1" }), { labelled: true });
  assert.ok(!more.includes('aria-expanded="true"'));
  const group = view("en", ready([node("le-1", { children: "some" })]));
  assert.ok(!group.includes('aria-expanded="true"') && group.includes('<ul class="o01-tree">'));
  const leaf = view("en", ready([node("le-1")]), { labelled: true });
  assert.ok(!leaf.includes("aria-expanded"));
});

test("a node already on its path is a plain row that says so", () => {
  for (const l of locales) {
    const html = view(l, ready([node("le-1", { kind: "legal_entity", children: "some" }), node("acct-1", { children: "some" })]), { depth: 2, path: ["le-1", "unit-1"], name: "Unit" });
    assert.equal(count(html, 'class="o01-toggle"'), 1, l);
    assert.equal(count(plain(html), o.shownAbove(l)), 1, l);
    assert.ok(plain(html).includes(`${kindName(l, "legal_entity")} · ${o.shownAbove(l)}`), l);
    assert.ok(html.includes(`<ul class="o01-branch" aria-label="${o.underLabel(l, "Unit")}">`), l);
  }
});

test("loading, failure, nothing under a node and the roots' own empty sentence are told apart", () => {
  for (const l of locales) {
    const loading = view(l, firstPage(), { name: "Unit" });
    assert.ok(loading.startsWith('<p role="status" class="live-meta o01-branch-note">') && plain(loading).replace(/\s+/g, "") === o.loadingUnder(l, "Unit").replace(/\s+/g, "") && loading.includes("<bdi>Unit</bdi>"), l);
    assert.ok(plain(view(l, firstPage())).includes(o.loadingMore(l)), l);
    const failed = view(l, { ...firstPage(), status: "error" }, { name: "Unit" });
    assert.ok(failed.startsWith('<p role="alert"') && plain(failed).replace(/\s+/g, "").includes(o.underFailed(l, "Unit").replace(/\s+/g, "")) && failed.includes("<bdi>Unit</bdi>") && plain(failed).includes(o.tryAgain(l)), l);
    assert.ok(plain(view(l, { ...firstPage(), status: "error" })).includes(o.listFailed(l)), l);
    assert.equal(view(l, ready([]), { name: "Unit" }), `<p role="status" class="live-note o01-branch-note">${o.nothingUnder(l, "Nov 16, 2026")}</p>`, l);
    assert.equal(view(l, ready([]), { empty: <p id="own">own</p> }), '<p id="own">own</p>', l);
    assert.equal(view(l, ready([]), { empty: null }), "", l);
    const searching = view(l, ready([], { next: "m:", stopped: true }), { name: "Unit" });
    assert.ok(!plain(searching).includes(o.nothingUnder(l, "Nov 16, 2026")) && plain(searching).includes(o.treeStopped(l)) && plain(searching).includes(o.continueSearching(l)), l);
  }
});

test("further pages: a named button, the stopped search, a page being loaded and a page that failed", () => {
  for (const l of locales) {
    const rows = [node("acct-1"), node("acct-2")];
    const next = view(l, ready(rows, { next: "m:acct-2" }), { depth: 1, name: "Unit" });
    assert.ok(next.includes(`aria-label="${o.moreUnder(l, "Unit")}"`) && plain(next).includes(o.loadMore(l)) && !plain(next).includes(o.treeStopped(l)), l);
    assert.ok(!view(l, ready(rows, { next: "p:x" }), { labelled: true }).includes("aria-label=\"" + o.moreUnder(l, "")), l);
    const stopped = view(l, ready(rows, { next: "m:acct-2", stopped: true }), { depth: 1, name: "Unit" });
    assert.ok(stopped.includes(`<p role="status">${o.treeStopped(l)}</p>`) && plain(stopped).includes(o.continueSearching(l)) && !stopped.includes("aria-label=\"" + o.moreUnder(l, "Unit")), l);
    const busy = view(l, ready(rows, { next: "m:acct-2", status: "more" }), { depth: 1, name: "Unit" });
    assert.ok(busy.includes('aria-disabled="true"') && plain(busy).includes(o.loadingMore(l)) && count(busy, "data-id=") === 2, l);
    const failed = view(l, ready(rows, { next: "m:acct-2", status: "moreFailed" }), { depth: 1, name: "Unit" });
    assert.ok(failed.includes('role="alert"') && plain(failed).includes(o.nextFailed(l)) && plain(failed).includes(o.tryAgain(l)) && count(failed, "data-id=") === 2, l);
  }
});

test("before any answer a list shows its loading line and a group is rendered hidden", () => {
  const html = renderToStaticMarkup(<Branch c={context("en")} list="roots" depth={0} path={[]} labelled />);
  assert.ok(plain(html).includes(o.loadingMore("en")));
  const group = renderToStaticMarkup(<TreeGroup c={context("en")} list="detached" id="o01-cycles" title={o.detachedTitle("en")} note={o.detachedNote("en")} />);
  assert.ok(group.startsWith('<section class="o01-unattributed" aria-labelledby="o01-cycles" hidden="">') && group.includes(`<h3 id="o01-cycles">${o.detachedTitle("en")}</h3>`) && plain(group).includes(o.detachedNote("en")));
});

test("the new sentences exist in three languages and differ from each other", () => {
  const all = (l: Locale) => [o.treeFilters(l), o.treeAnnounce(l, 2, "d"), o.loadingUnder(l, "n"), o.underFailed(l, "n"), o.listFailed(l), o.nothingUnder(l, "d"), o.underLabel(l, "n"), o.moreUnder(l, "n"), o.shownAbove(l), o.treeStopped(l), o.detachedTitle(l), o.detachedNote(l), o.noRoots(l, "d")];
  assert.equal(new Set(locales.flatMap(all)).size, 39);
  for (const l of locales) for (const s of all(l)) assert.ok(s.trim().length > 3 && !s.includes("undefined"), `${l} ${s}`);
});
