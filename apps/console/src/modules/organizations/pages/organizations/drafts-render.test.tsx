import { test } from "node:test";
import assert from "node:assert/strict";
import * as nodeModule from "node:module";
import { readFileSync } from "node:fs";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { Locale } from "../../../../shared/types";
import type { GroupDraft } from "../../../../service/admin";
import { o } from "./live-copy";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (assetStub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { DraftPanel, DraftRows, DraftsSection, SelectionBar, RowCheck, HeaderCheck } = await import("./drafts");
type Editor = Parameters<typeof DraftPanel>[0]["e"];
type List = Parameters<typeof DraftsSection>[0]["list"];

const locales: Locale[] = ["en", "ko", "ar"];
const noop = () => undefined;
const bare = (html: string) => html.replace(/<[^>]+>/g, "").replace(/&#x27;/g, "'").replace(/&quot;/g, '"').replace(/&amp;/g, "&").replace(/⁨|⁩/g, "");
const list = (over: Partial<List> = {}): List => ({
  status: "ready", drafts: [], next: "", more: "idle", open: 1, limit: 16, maxMembers: 1000, revision: "r3", active: 1,
  reload: async () => null, loadMore: noop, find: async () => null, ...over,
});
const draft = (over: Partial<GroupDraft> = {}): GroupDraft => ({
  id: "vgd-0123456789abcdef", name: "Ops reviewers", group: "group-00112233aabbccdd", group_label: "group-00112233aabbccdd", new_group: true, member_count: 2,
  members: [{ id: "sample-acct-01", name: "Account 01" }], author: "u", author_login: "root@example.invalid", created: "2026-11-18T09:00:00Z", updated: "2026-11-18T09:05:00Z",
  discarded: "", state: "open", activated_revision: 0, pruned: false, plan: null, ...over,
});
const editor = (over: Partial<Editor> = {}): Editor => ({
  target: null, draft: null, name: "", setName: noop, mode: "new", setMode: noop, group: null, setGroup: noop, adds: [], setAdds: noop, removes: [], setRemoves: noop,
  reason: "", setReason: noop, errors: {}, setErrors: noop, summaryKey: 0, progress: null, partial: null, notice: null, confirm: false, setConfirm: noop, discarding: false, editing: true, leave: null,
  members: { rows: [{ id: "sample-acct-01", name: "Account 01" }, { id: "sample-acct-02", name: "" }], next: "", status: "ready" }, loadMembers: async () => undefined, count: 0, isDirty: false,
  start: { name: "", mode: "new", group: "", adds: [], removes: [] }, openNew: noop, openDraft: noop, close: noop, guarded: noop, drop: noop, discardInput: noop, keepEditing: noop, touch: noop,
  addRows: () => ({ list: [], added: 0, already: 0, over: 0 }), save: async () => undefined, retryRest: async () => undefined, discardDraft: async () => undefined,
  discardChanges: noop, askDiscard: noop, keepDraft: noop, ...over,
} as Editor);
const search = async () => ({ rows: [], next: "" });
const panel = (l: Locale, e: Editor, lst = list()) =>
  renderToStaticMarkup(<DraftPanel l={l} e={e} list={lst} lang={l} on="2026-12-01" today="2026-11-18" base={1} searchAccounts={search} searchGroups={search} />);
const withDraft = (d: GroupDraft, over: Partial<Editor> = {}) => editor({ target: { kind: "draft", draft: d }, draft: d, name: d.name, count: d.member_count, start: { name: d.name, mode: "new", group: "", adds: [], removes: [] }, ...over });

test("the drafts section lists drafts with state, group and plan, and offers a new draft only below the open limit", () => {
  for (const l of locales) {
    const rows = [draft(), draft({ id: "vgd-1111111111111111", new_group: false, name: "", group: "sample-view-north", group_label: "North view group", state: "proposed", plan: { id: "plan-aa", state: "sealed", closed: "", blocked: "impact_not_computed", rule: "" } }),
      draft({ id: "vgd-2222222222222222", state: "open", plan: { id: "plan-bb", state: "rejected", closed: "", blocked: "", rule: "no_change" } })];
    const html = renderToStaticMarkup(<DraftsSection l={l} list={list({ drafts: rows })} lang={l} current="vgd-0123456789abcdef" onOpen={noop} onNew={noop} />);
    const text = bare(html);
    for (const want of [o.drafts(l), o.openDrafts(l, 1, 16), o.newDraft(l), o.colDraft(l), o.colPlan(l), "Ops reviewers", "North view group", o.newGroup(l), o.existingGroup(l), o.stOpen(l), o.stProposed(l), o.lastRejected(l), "plan-aa", o.none(l)]) {
      assert.ok(text.includes(want), `${l} ${want}`);
    }
    assert.ok(html.includes('aria-current="true"') && html.includes('class="live-selected"') && html.includes('href="#/organization-changes?view=impact&amp;plan=plan-aa&amp;lang=' + l + '"'), l);
    const full = renderToStaticMarkup(<DraftsSection l={l} list={list({ drafts: rows, open: 16 })} lang={l} current="" onOpen={noop} onNew={noop} />);
    assert.ok(!full.includes("o01-new-draft") && bare(full).includes(o.openLimit(l, 16)), l);
    const empty = renderToStaticMarkup(<DraftsSection l={l} list={list()} lang={l} current="" onOpen={noop} onNew={noop} />);
    assert.ok(bare(empty).includes(o.noDrafts(l)) && empty.includes("o01-new-draft"), l);
    assert.ok(bare(renderToStaticMarkup(<DraftsSection l={l} list={list({ status: "error" })} lang={l} current="" onOpen={noop} onNew={noop} />)).includes(o.draftsFailed(l)), l);
    assert.ok(bare(renderToStaticMarkup(<DraftsSection l={l} list={list({ status: "loading" })} lang={l} current="" onOpen={noop} onNew={noop} />)).includes(o.loadingDrafts(l)), l);
    const more = renderToStaticMarkup(<DraftsSection l={l} list={list({ drafts: rows, next: "vgd-2222222222222222" })} lang={l} current="" onOpen={noop} onNew={noop} />);
    assert.ok(bare(more).includes(o.loadMore(l)), l);
  }
});

test("the selection bar states the page count, never a filter total, and is absent-not-disabled at limits", () => {
  for (const l of locales) {
    const bar = (over: Partial<Parameters<typeof SelectionBar>[0]> = {}) => renderToStaticMarkup(<SelectionBar l={l} n={0} k={7} cap={1000} atCap={false} limitReached={false} limit={16} editing={false} cleared={false} wide
      onClear={noop} onStart={noop} onAdd={noop} onSelectAll={noop} {...over} />);
    assert.ok(bare(bar()).includes(o.selNone(l)) && !bar().includes("<button"), l);
    assert.ok(bare(bar({ wide: false })).includes(o.selectAllShown(l)), l);
    assert.ok(bare(bar({ cleared: true })).includes(o.selCleared(l)), l);
    const some = bar({ n: 3 });
    assert.ok(bare(some).includes(bare(o.selCount(l).replace("{n}", "3").replace("{k}", "7"))) && some.includes("<strong>3</strong>"), l);
    assert.ok(bare(some).includes(o.startDraft(l)) && bare(some).includes(o.clearSel(l)) && !bare(some).includes(o.addToDraft(l, 3)), l);
    assert.ok(bare(bar({ n: 3, editing: true })).includes(o.addToDraft(l, 3)) && !bare(bar({ n: 3, editing: true })).includes(o.startDraft(l)), l);
    const limited = bar({ n: 3, limitReached: true });
    assert.ok(!bare(limited).includes(o.startDraft(l)) && bare(limited).includes(o.openLimit(l, 16)) && !limited.includes("disabled"), l);
    assert.ok(bare(bar({ n: 2, atCap: true, cap: 2 })).includes(o.selCap(l, 2)), l);
    for (const html of [bar(), some]) assert.ok(!/matching|of \d+ total/.test(bare(html)), l);
  }
});

test("row checkboxes name the row with its ID and the header checkbox is tri-state", () => {
  for (const l of locales) {
    const row = renderToStaticMarkup(<RowCheck l={l} id="sample-acct-01" name="Account 01" checked={false} disabled={false} stacked onToggle={noop} />);
    assert.ok(row.includes(`aria-label="${o.selectRow(l, "Account 01", "sample-acct-01")}"`) && row.includes('data-check="sample-acct-01"') && bare(row).includes(o.selectWord(l)), l);
    assert.ok(renderToStaticMarkup(<RowCheck l={l} id="x" name="x" checked={false} disabled stacked={false} onToggle={noop} />).includes("disabled"), l);
    const head = renderToStaticMarkup(<HeaderCheck l={l} state="all" onToggle={noop} />);
    assert.ok(head.includes('id="o01-check-all"') && head.includes("checked") && head.includes(`aria-label="${o.selectAllRows(l)}"`), l);
  }
});

test("the new draft form carries the not-ownership note, the group choice, members, reason and batching", () => {
  for (const l of locales) {
    const html = panel(l, editor({ target: { kind: "new" }, adds: [{ id: "sample-acct-05", name: "Account 05" }], count: 1 }));
    const text = bare(html);
    for (const want of [o.newDraft(l), o.notOwnership(l), o.groupLegend(l), o.newGroup(l), o.existingGroup(l), o.draftNameLabel(l), o.nameHint(l), o.membersToAdd(l), o.searchAccounts(l),
      o.enterIds(l), o.removeFromDraft(l), o.batchHint(l), o.saveDraft(l), o.cancel(l), o.toAddMeta(l, 1, 1000), "sample-acct-05"]) assert.ok(text.includes(want), `${l} ${want}`);
    assert.ok(html.indexOf("o01-not-ownership") < html.indexOf('name="o01-group-mode"'), l);
    assert.ok(html.includes('id="o01-draft-name"') && html.includes('maxLength="80"') && html.includes('dir="auto"') && html.includes('id="live-reason"'), l);
    assert.ok(!text.includes(o.createPlan(l)) && !text.includes(o.discardDraft(l)) && !/join|get access|Remove from group/i.test(bare(html).replace(o.notOwnership(l), "")), l);
    const existing = panel(l, editor({ target: { kind: "new" }, mode: "existing" }));
    assert.ok(!existing.includes('id="o01-draft-name"') && bare(existing).includes(o.searchGroups(l)), l);
    const chosen = panel(l, editor({ target: { kind: "new" }, mode: "existing", group: { id: "sample-view-north", label: "North view group" } }));
    assert.ok(bare(chosen).includes(o.groupChosen(l)) && bare(chosen).includes("North view group") && bare(chosen).includes(o.change(l)), l);
    const saving = panel(l, editor({ target: { kind: "new" }, progress: { done: 31, total: 70, changes: false } }));
    assert.ok(bare(saving).includes(o.saving(l, 31, 70)) && bare(saving).includes(o.savingButton(l)) && !bare(saving).includes(o.cancel(l)) && saving.includes('role="status"'), l);
    const errs = panel(l, editor({ target: { kind: "new" }, errors: { name: o.eName(l), members: o.eMembersServer(l) }, summaryKey: 1 }));
    assert.ok(errs.includes("service-summary") && errs.includes('href="#o01-draft-name"') && errs.includes('href="#o01-draft-members"') && bare(errs).includes(o.eMembersServer(l)), l);
  }
});

test("the draft panel shows only the actions its state allows, with the state block", () => {
  for (const l of locales) {
    const open = panel(l, withDraft(draft()));
    const t = bare(open);
    for (const want of [o.draftSub(l, o.stOpen(l)), o.notOwnership(l), o.draftNameLabel(l), o.membersOf(l, 2, 1000), "root@example.invalid", o.createPlan(l), o.discardDraft(l), o.removeFromDraft(l), o.membersToAdd(l)]) {
      assert.ok(t.includes(want), `${l} open ${want}`);
    }
    assert.ok(open.includes('href="#/organization-changes?view=targets&amp;draft=vgd-0123456789abcdef&amp;from=2026-12-01&amp;base=1&amp;lang=' + l + '"'), l);
    const dirtyOne = panel(l, withDraft(draft(), { isDirty: true, adds: [{ id: "sample-acct-06", name: "" }], removes: ["sample-acct-02"] }));
    assert.ok(bare(dirtyOne).includes(o.pendingSummary(l, 1, 1, false)) && bare(dirtyOne).includes(o.saveFirst(l)) && !dirtyOne.includes("o01-draft-plan") && bare(dirtyOne).includes(o.willRemove(l)) && bare(dirtyOne).includes(o.undo(l)), l);
    const held = panel(l, withDraft(draft({ state: "proposed", plan: { id: "plan-cc", state: "sealed", closed: "", blocked: "impact_not_computed", rule: "" } })));
    assert.ok(bare(held).includes(o.bHeld(l).split("{p}")[1].trim().slice(-40)) && bare(held).includes("plan-cc") && bare(held).includes(o.openPlan(l)) && !held.includes("o01-draft-plan") && !bare(held).includes(o.removeFromDraft(l)) && !held.includes('id="live-reason"'), l);
    const rejected = panel(l, withDraft(draft({ plan: { id: "plan-dd", state: "rejected", closed: "", blocked: "", rule: "plan_limit" } })));
    assert.ok(bare(rejected).includes("plan-dd") && rejected.includes("o01-draft-plan"), l);
    const activated = panel(l, withDraft(draft({ state: "activated", activated_revision: 4, pruned: true, plan: { id: "plan-ee", state: "activated", closed: "", blocked: "", rule: "" } })));
    assert.ok(bare(activated).includes(o.bPruned(l)) && bare(activated).includes(o.fActivatedIn(l)) && !activated.includes("o01-members-title") && !activated.includes("o01-discard-draft"), l);
    const discarded = panel(l, withDraft(draft({ state: "discarded", discarded: "2026-11-18T10:00:00Z" })));
    assert.ok(discarded.includes('<time dateTime="2026-11-18T10:00:00Z">') && !bare(discarded).includes(o.openPlan(l)) && !discarded.includes("o01-draft-plan"), l);
    const confirm = panel(l, withDraft(draft(), { confirm: true }));
    assert.ok(bare(confirm).includes(o.discardQ(l)) && bare(confirm).includes(o.discardEffect(l)) && bare(confirm).includes(o.keepDraft(l)) && !confirm.includes("o01-draft-plan"), l);
    const leave = panel(l, withDraft(draft(), { leave: noop, isDirty: true }));
    assert.ok(leave.includes('id="o01-leave"') && bare(leave).includes(o.discardInput(l)) && bare(leave).includes(o.keepEditing(l)) && leave.indexOf("o01-leave") < leave.indexOf("o01-not-ownership"), l);
    const partial = panel(l, withDraft(draft(), { partial: { done: 31, total: 70, rest: 39, reason: o.partialBusy(l), kind: "new", draftId: "vgd-0123456789abcdef" } }));
    assert.ok(bare(partial).includes(o.partialSave(l, 31, 70, 39)) && bare(partial).includes(o.retryRest(l, 39)) && partial.includes('role="alert"'), l);
    const full = panel(l, withDraft(draft({ member_count: 3 }), { count: 3 }), list({ maxMembers: 3 }));
    assert.ok(bare(full).includes(o.draftFull(l, 3)) && !full.includes('id="o01-draft-q"'), l);
    const existing = panel(l, withDraft(draft({ new_group: false, name: "", group: "sample-view-north", group_label: "North view group" })));
    assert.ok(!existing.includes('id="o01-draft-name"') && bare(existing).includes("North view group"), l);
    const gone = panel(l, editor({ target: { kind: "gone", id: "vgd-x" } }));
    assert.ok(bare(gone).includes(o.draftGone(l)) && bare(gone).includes(o.back(l)), l);
  }
});

test("a draft name is never a group label before activation", () => {
  const rows = [draft({ group_label: "group-00112233aabbccdd" })];
  const html = renderToStaticMarkup(<DraftsSection l="en" list={list({ drafts: rows })} lang="en" current="" onOpen={noop} onNew={noop} />);
  const groupCell = /data-label="Group">([^]*?)<\/td>/.exec(html)?.[1] ?? "";
  assert.ok(!groupCell.includes("Ops reviewers") && groupCell.includes("group-00112233aabbccdd"));
});

test("the O01 and O02 pages add no page-level unload handler and the shell passes onDirty to both", () => {
  for (const file of ["./Live.tsx", "./drafts.tsx", "../changes/Live.tsx"]) {
    assert.ok(!readFileSync(new URL(file, import.meta.url), "utf8").includes("beforeunload"), file);
  }
  const work = readFileSync(new URL("../../../../app/screen-groups/work.tsx", import.meta.url), "utf8");
  assert.ok(work.includes("<LiveOrganizations {...changeProps(p, service)} />") && work.includes("<LiveChanges {...changeProps(p, service)} />"));
});

test("the draft panel separates saved members from the count after changes and words partial saves by flow", () => {
  for (const l of locales) {
    const html = panel(l, withDraft(draft(), { count: 3, adds: [{ id: "sample-acct-06", name: "" }], isDirty: true }));
    assert.ok(bare(html).includes(o.inDraft(l, 2)) && bare(html).includes(o.inDraftMeta(l, 3, 1000)), l);
    assert.ok(!html.includes(">sample-acct-06</bdi> <bdi") && html.includes("sample-acct-06"), l);
    const edit = panel(l, withDraft(draft(), { partial: { done: 31, total: 40, rest: 9, reason: o.partialFailed(l), kind: "edit", draftId: "vgd-0123456789abcdef" } }));
    assert.ok(bare(edit).includes(o.partialSaveChanges(l, 31, 40, 9)) && bare(edit).includes(o.partialFailed(l)) && !bare(edit).includes(o.depFailed(l)), l);
    const bareRejected = panel(l, withDraft(draft({ plan: { id: "plan-dd", state: "rejected", closed: "", blocked: "", rule: "" } })));
    assert.ok(bare(bareRejected).includes(o.bRejectedBare(l).split("{p}")[1].trim().slice(0, 20)), l);
    const limit = panel(l, withDraft(draft({ plan: { id: "plan-dd", state: "rejected", closed: "", blocked: "", rule: "plan_limit" } })));
    assert.ok(bare(limit).includes(o.planLimitCause(l)), l);
    const stamp = panel(l, withDraft(draft()));
    assert.ok(stamp.includes("root@example.invalid</bdi><br/><time"), l);
    const busy = panel(l, withDraft(draft(), { confirm: true, discarding: true }));
    assert.ok(busy.includes('aria-disabled="true"') && bare(busy).includes(o.savingButton(l)), l);
  }
});

test("the selection bar states the room left in an open draft and compact rows carry IDs and state on two lines", () => {
  for (const l of locales) {
    const html = renderToStaticMarkup(<SelectionBar l={l} n={2} k={6} cap={2} atCap limitReached={false} limit={16} editing cleared={false} wide onClear={noop} onStart={noop} onAdd={noop} onSelectAll={noop} />);
    assert.ok(bare(html).includes(o.selCapDraft(l, 2)) && !bare(html).includes(o.selCap(l, 2)), l);
    const rows = renderToStaticMarkup(<DraftRows l={l} drafts={[draft({ state: "proposed", plan: { id: "plan-aa", state: "sealed", closed: "", blocked: "", rule: "" } })]} lang={l} current="" onOpen={noop} />);
    const text = bare(rows);
    for (const want of ["Ops reviewers", o.stProposed(l), o.nMembers(l, 2), "vgd-0123456789abcdef", o.newGroup(l), "group-00112233aabbccdd", "plan-aa"]) assert.ok(text.includes(want), `${l} ${want}`);
    assert.ok(rows.indexOf("o01-draft-row1") < rows.indexOf("o01-draft-row2") && !/Label:|Draft:/.test(text), l);
  }
});

test("the editor routes its decisions through the tested model functions", () => {
  const src = readFileSync(new URL("./drafts.tsx", import.meta.url), "utf8");
  for (const use of ["isLatest(n, memberSeq.current)", "sameView(mine, generation.current)", "callRevision(i, !!created, list.revision)", "callRevision(i, false, list.revision)",
    "savePlan(partial, target?.kind ?? null, draft?.id ?? \"\")", "partitionPrecheck(pending, saved, known)", "afterRefusal(read, cause, found?.state === \"open\")", "o[conflictCopy(!!id)](l)", "if (!isLatest(n, memberSeq.current)) return;", "if (isLatest(n, memberSeq.current)) setMembers", "if (!draft || discarding) return;"]) {
    assert.ok(src.includes(use), use);
  }
  assert.ok(!src.includes("list.find(id).catch(() => null);\n    await list.reload();\n    if (!found) { setTarget({ kind: \"gone\""), "gone only after a successful read");
});
