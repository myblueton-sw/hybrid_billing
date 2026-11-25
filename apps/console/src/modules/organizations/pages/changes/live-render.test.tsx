import { test } from "node:test";
import assert from "node:assert/strict";
import * as nodeModule from "node:module";
import { renderToStaticMarkup } from "react-dom/server";
import React from "react";
import type { Locale } from "../../../../shared/types";
import { planOf, type PlanRow, type PlansPage } from "../../../../service/admin";
import { c } from "./live-copy";
import { refusal } from "./live-model";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (assetStub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { Approval, BaseFact, DatedRange, Detail, GroupHandoff, HandoffNote, ImpactActions, Steps, decisionLabel } = await import("./Live");

const locales: Locale[] = ["en", "ko", "ar"];
const computed = { money: "computed", publication: "computed", erp: "computed" };
const notComputed = { money: "not_computed", publication: "not_computed", erp: "not_computed" };
const plan = (over: Record<string, unknown>): PlanRow =>
  planOf({ id: "plan-1", state: "sealed", change: { kind: "hierarchy", member: "acct-1", action: "set", target: "unit-1", from: "2026-12-01" }, impact: notComputed, blocked: "impact_not_computed", ...over });
const page: PlansPage = { revision: "r", active_revision: 1, open_plans: 1, open_limit: 16, plans: [], you: "u", closed: { total: 0, next: "", kept: 200, dropped: 0 } };
const noop = () => undefined;
const approval = (l: Locale, p: PlanRow) =>
  renderToStaticMarkup(<Approval l={l} p={p} sentence="s" kindName="k" confirm={null} busy={null} reason={null} onOpen={noop} onDecide={noop} onCancel={noop} />);
const detail = (l: Locale, p: PlanRow) => renderToStaticMarkup(<Detail l={l} p={p} page={page} parties={[]} members={[]} full />);
const plain = (html: string) => html.replace(/<[^>]+>/g, " ").replace(/&#x27;/g, "'").replace(/&quot;/g, '"').replace(/&amp;/g, "&");

test("a held plan shows the held block in place of activation and keeps discard", () => {
  for (const l of locales) {
    const out = plain(approval(l, plan({})));
    const html = approval(l, plan({}));
    assert.ok(out.includes(c.heldLead(l)), l);
    assert.ok(out.includes(c.heldNext(l)), l);
    assert.ok(!out.includes(c.blockedTitle(l)), l);
    assert.ok(!html.includes("o02-open-activate"), l);
    assert.ok(!out.includes(c.reviewActivate(l)), l);
    assert.ok(html.includes('id="o02-open-discard"'), l);
    assert.ok(out.includes(c.reviewDiscard(l)), l);
    const heading = html.indexOf('id="o02-activate-title"');
    const block = html.indexOf('class="live-blocked o02-held"');
    assert.ok(heading > 0 && heading < block && block < html.indexOf('class="live-impact"'), l);
    assert.ok(plain(html).includes(c.restate(l, "", "", "").slice(0, 6)), l);
  }
});

test("a row that claims activation without computed impact is still held", () => {
  for (const impact of [notComputed, {}, { ...computed, erp: "not_computed" }]) {
    const p = plan({ impact, blocked: "", can_activate: true });
    const html = approval("en", p);
    assert.ok(!html.includes("o02-open-activate") && html.includes('class="live-blocked o02-held"') && html.includes('id="o02-open-discard"'), JSON.stringify(impact));
  }
  assert.ok(plain(approval("en", plan({ impact: {}, blocked: "", can_activate: true }))).includes(c.heldUnknownLead("en")));
});

test("an unreadable impact holds activation with its own sentences", () => {
  for (const l of locales) {
    const out = plain(approval(l, plan({ blocked: "impact_unavailable", impact: {} })));
    assert.ok(out.includes(c.heldUnknownLead(l)) && out.includes(c.heldUnknownNext(l)), l);
    assert.ok(!out.includes(c.heldLead(l)) && !out.includes(c.blockedTitle(l)), l);
    assert.ok(!approval(l, plan({ blocked: "impact_unavailable" })).includes("o02-open-activate"), l);
  }
});

test("other blocks and an activatable plan keep their existing views", () => {
  const own = plain(approval("en", plan({ impact: computed, blocked: "own_proposal" })));
  assert.ok(own.includes(c.blockedTitle("en")) && own.includes(c.own("en")) && !own.includes(c.heldLead("en")));
  const stale = plain(approval("en", plan({ impact: computed, blocked: "stale", stale: true })));
  assert.ok(stale.includes(c.staleBlock("en")) && stale.includes(c.next("en")));
  const ready = approval("en", plan({ impact: computed, blocked: "", can_activate: true }));
  assert.ok(ready.includes('id="o02-open-activate"') && !plain(ready).includes(c.heldLead("en")) && ready.includes('id="o02-open-discard"'));
  const confirming = renderToStaticMarkup(<Approval l="en" p={plan({ impact: computed, blocked: "", can_activate: true })} sentence="s" kindName="k" confirm="activate" busy={null} reason={null} onOpen={noop} onDecide={noop} onCancel={noop} />);
  assert.ok(plain(confirming).includes(c.confirmActivate("en")));
  assert.ok(plain(approval("en", plan({ state: "activated" }))).includes(c.onlySealed("en", c.stateWord("en", "activated"))));
});

test("the impact view leads with the held sentence and relabels its button", () => {
  for (const l of locales) {
    const html = detail(l, plan({}));
    const out = plain(html);
    const lead = html.indexOf('<p class="o02-held-lead">');
    assert.ok(lead > html.indexOf('id="o02-impact-title"') && lead < html.indexOf('<dl class="o02-impact">'), l);
    assert.ok(out.includes(c.heldLead(l)), l);
    assert.ok(out.includes(c.notComputed(l)) && out.includes(c.notZero(l)) && out.includes(c.impactNote(l)), l);
    assert.ok(plain(detail(l, plan({ blocked: "impact_unavailable", impact: {} }))).includes(c.heldUnknownLead(l)), l);
    const actions = plain(renderToStaticMarkup(<ImpactActions l={l} p={plan({})} onDecision={noop} onAnother={noop} />));
    assert.ok(actions.includes(c.toDiscard(l)) && !actions.includes(c.toApproval(l)), l);
    assert.equal(decisionLabel(l, plan({})), c.toDiscard(l), l);
    assert.equal(decisionLabel(l, plan({ impact: computed, blocked: "own_proposal" })), c.toApproval(l), l);
  }
  const closed = plain(detail("en", plan({ state: "discarded", closed: "discarded", blocked: "" })));
  assert.ok(!closed.includes(c.heldLead("en")));
  const done = plain(detail("en", plan({ impact: computed, blocked: "", can_activate: true })));
  assert.ok(done.includes(c.computedImpact("en")) && !done.includes(c.heldLead("en")) && !done.includes(c.notZero("en")) && !done.includes(c.impactNote("en")));
  assert.ok(plain(renderToStaticMarkup(<ImpactActions l="en" p={plan({ impact: computed, blocked: "", can_activate: true })} onDecision={noop} onAnother={noop} />)).includes(c.toApproval("en")));
  assert.equal(plain(renderToStaticMarkup(<ImpactActions l="en" p={plan({ state: "rejected" })} onDecision={noop} onAnother={noop} />)).includes(c.toDiscard("en")), false);
  assert.ok(plain(detail("en", plan({ impact: {} }))).includes(c.unknownImpact("en")));
});

test("progress has four named steps with the held reasons", () => {
  for (const l of locales) {
    const html = renderToStaticMarkup(<Steps l={l} p={plan({})} />);
    for (const name of c.stepNames(l)) assert.ok(plain(html).includes(name), l);
    assert.equal((html.match(/class="o02-step"/g) ?? []).length, 4);
    assert.match(html, /data-state="done".*data-state="held".*data-state="na".*data-state="na"/);
    assert.ok(plain(html).includes(c.stepState(l, "held")) && plain(html).includes(c.notComputed(l)), l);
    assert.ok(plain(renderToStaticMarkup(<Steps l={l} p={plan({ blocked: "impact_unavailable", impact: {} })} />)).includes(c.unknownImpact(l)), l);
  }
  const own = renderToStaticMarkup(<Steps l="en" p={plan({ impact: computed, blocked: "own_proposal" })} />);
  assert.match(own, /data-state="done".*data-state="done".*data-state="held".*data-state="na"/);
  assert.ok(plain(own).includes("Proposed by you"));
  const ready = renderToStaticMarkup(<Steps l="en" p={plan({ impact: computed, blocked: "", can_activate: true })} />);
  assert.match(ready, /aria-current="step" data-state="current"/);
  const discarded = renderToStaticMarkup(<Steps l="en" p={plan({ state: "discarded", closed: "discarded", blocked: "" })} />);
  assert.match(discarded, /data-state="done".*data-state="stopped".*data-state="na".*data-state="na"/);
  assert.ok(plain(discarded).includes("Discarded"));
  for (const b of ["stale", "passed", "other"]) assert.ok(c.approvalHeld("en", b).length > 0);
});

test("renamed words and the refused notice follow the reloaded row", () => {
  assert.deepEqual(locales.map((l) => c.stateWord(l, "sealed")), ["Changes locked", "변경 고정됨", "التغييرات مقفلة"]);
  assert.deepEqual(locales.map((l) => c.lead(l)), ["Proposing creates a plan whose relationship changes are locked.", "제안하면 관계 변경이 고정된 계획이 만들어집니다.", "يُنشئ الاقتراح خطة بتغييرات علاقات مثبّتة."]);
  assert.ok(c.aSealed("en").endsWith("Its relationship changes are locked."));
  assert.deepEqual(locales.map((l) => c.limit(l, 16)), ["16 plans are open, which is the limit. Discard one before proposing another.", "열린 변경 계획이 한도인 16개입니다. 새로 제안하려면 먼저 하나를 폐기하세요.", "عدد الخطط المفتوحة 16، وهو الحد الأقصى. تجاهل خطة قبل اقتراح أخرى."]);
  assert.equal(c.onlySealed("en", "Activated"), "State of this plan: Activated. Only open plans with locked changes can be discarded or activated.");
  assert.deepEqual(locales.map((l) => c.stepState(l, "held")), ["Held", "보류", "معلَّقة"]);
  assert.deepEqual(locales.map((l) => c.notComputed(l)), ["Not calculated", "계산되지 않음", "لم يُحتسب"]);
  assert.deepEqual(locales.map((l) => c.unknownImpact(l)), ["Could not be read", "확인하지 못함", "تعذّرت القراءة"]);
  assert.equal(c.activate("en"), "Activation");
  assert.equal(c.refused("en", refusal(plan({}))), c.heldLead("en"));
  assert.equal(c.refused("ko", refusal(plan({ blocked: "impact_unavailable" }))), c.heldUnknownLead("ko"));
  assert.equal(c.refused("ar", refusal(plan({ impact: computed, blocked: "own_proposal" }))), c.own("ar"));
  assert.equal(c.refused("en", refusal(undefined)), c.otherBlock("en"));
  assert.equal(c.heldLead("en"), "Activation is held: the impact of this change on amounts, publication and ERP has not been calculated.");
});

const bare = (html: string) => html.replace(/<[^>]+>/g, "").replace(/&#x27;/g, "'").replace(/&quot;/g, '"').replace(/&amp;/g, "&");
const carried = (over: Record<string, unknown> = {}) => ({ kind: "hierarchy" as const, member: "acct-1", from: "", passed: "", base: 2 as number | null, dropped: false, draft: "", refused: false, ...over });
const note = (l: Locale, h: ReturnType<typeof carried>, active: number) => renderToStaticMarkup(<HandoffNote l={l} h={h} active={active} onUseBase={noop} />);

test("the handoff block states the prefill and the carried base, with no live role", () => {
  for (const l of locales) {
    const html = note(l, carried(), 2);
    assert.match(html, /^<div id="o02-handoff" class="live-note-rule o02-handoff" tabindex="-1">/, l);
    assert.ok(!html.includes("role="), l);
    assert.ok(bare(html).includes(bare(c.prefilled(l, "2"))), l);
    assert.ok(html.includes('<bdi dir="ltr">2</bdi>'), l);
    assert.ok(!html.includes("o02-use-base") && !bare(html).includes(c.staleTag(l)), l);
    assert.ok(!bare(html).includes(c.linkDropped(l)), l);
    const none = bare(note(l, carried({ base: null }), 5));
    assert.ok(none.includes(c.prefilledNoBase(l)) && !none.includes(c.staleTag(l)), l);
  }
});

test("a changed base shows the notice and the button that is the only way to move the base", () => {
  for (const l of locales) {
    const html = note(l, carried(), 4);
    const text = bare(html);
    assert.ok(text.includes(`${c.staleTag(l)}.`) && text.includes(bare(c.baseChanged(l, "2", "4"))), l);
    assert.match(html, /<button type="button" id="o02-use-base" class="live-secondary o02-use-base"><span>.*<bdi dir="ltr">4<\/bdi>.*<\/span><\/button>/, l);
    assert.ok(text.includes(bare(c.useBase(l, "4"))), l);
    assert.equal((html.match(/<button/g) ?? []).length, 1, l);
    const refused = bare(note(l, carried({ refused: true }), 4));
    assert.ok(refused.includes(bare(c.baseRefused(l, "2", "4"))) && !refused.includes(bare(c.baseChanged(l, "2", "4"))), l);
  }
  assert.ok(bare(note("en", carried(), 4)).includes("This change is based on relationship revision 2, but revision 4 is now active."));
  assert.ok(note("ko", carried({ refused: true }), 4).includes('style="white-space:nowrap">리비전이 <bdi dir="ltr">2</bdi>에서</span>'));
  assert.equal(c.useBase("ko", "4"), "리비전 4 기준으로 하기");
});

test("dropped link values and a passed date are stated in the block", () => {
  for (const l of locales) {
    const text = bare(note(l, carried({ dropped: true, passed: "2026-11-01" }), 2));
    assert.ok(text.includes(c.linkDropped(l)), l);
    assert.ok(text.includes(bare(c.fromPassed(l, "x")).split("x")[0].trim()), l);
    const only = bare(note(l, carried({ kind: "", member: "", base: null, dropped: true }), 2));
    assert.ok(only.includes(c.linkDropped(l)) && !only.includes(c.prefilledNoBase(l)), l);
  }
  assert.ok(note("en", carried({ passed: "2026-11-01" }), 2).includes('<time dateTime="2026-11-01">'));
});

test("the base fact reads active or names the active revision", () => {
  for (const l of locales) {
    const same = bare(renderToStaticMarkup(<BaseFact l={l} base={3} active={3} />));
    assert.ok(same.includes(c.baseFact(l)) && same.includes(c.baseActive(l)), l);
    const other = bare(renderToStaticMarkup(<BaseFact l={l} base={3} active={5} />));
    assert.ok(other.includes(c.baseNotActive(l, 5)), l);
  }
  assert.ok(renderToStaticMarkup(<BaseFact l="en" base={3} active={5} />).includes('<dd><bdi dir="ltr">3</bdi> · not active; revision 5 is active</dd>'));
  assert.equal(c.eBase("en", 3, 5), "This change is based on revision\u00a0\u20683\u2069, but revision\u00a0\u20685\u2069 is active. Choose \"Base on revision\u00a0\u20685\u2069\" first.");
  assert.ok(c.eBase("ko", 3, 5).includes("리비전\u00a0\u20683\u2069\u2060이지만") && c.eBase("ko", 3, 5).includes("\u20685\u2069\u2060입니다"));
});

const groupDraft = { id: "vgd-0123456789abcdef", name: "Ops reviewers", group: "group-00112233aabbccdd", group_label: "group-00112233aabbccdd", new_group: true, member_count: 3,
  members: [], author: "u", author_login: "root", created: "", updated: "", discarded: "", state: "open" as const, activated_revision: 0, pruned: false, plan: null };
const group = (l: Locale, status: "loading" | "ready" | "missing" | "closed" | "error", h = carried({ kind: "", member: "", draft: groupDraft.id, base: 1 }), active = 1, over = {}) =>
  renderToStaticMarkup(<GroupHandoff l={l} h={h} group={{ status, draft: status === "ready" ? { ...groupDraft, ...over } : null, members: [{ id: "sample-acct-01", name: "Account 01" }, { id: "sample-acct-02", name: "" }] }}
    active={active} onUseBase={noop} onRetry={noop} />);

test("the group draft handoff is the propose confirmation with the note, the effect and the first members", () => {
  for (const l of locales) {
    const html = group(l, "ready");
    const text = bare(html).replace(/⁨|⁩/g, "");
    for (const want of [c.groupTitle(l), c.draftNameFact(l), "Ops reviewers", c.newGroup(l), "group-00112233aabbccdd", c.notOwnership(l), c.draftEffect(l, 3), c.membersFirst(l, 2, 3), "Account 01", "sample-acct-02"]) {
      assert.ok(text.includes(want), `${l} ${want}`);
    }
    assert.ok(html.startsWith('<div id="o02-handoff" class="live-note-rule o02-handoff o02-group-handoff" tabindex="-1">') && !html.includes('role="status"'), l);
    assert.ok(!html.includes("o02-use-base"), l);
    const existing = group(l, "ready", undefined, 1, { new_group: false, name: "", group: "sample-view-north", group_label: "North view group" });
    assert.ok(!bare(existing).includes(c.draftNameFact(l)) && bare(existing).includes("North view group"), l);
    const stale = group(l, "ready", carried({ kind: "", member: "", draft: groupDraft.id, base: 1 }), 2);
    assert.ok(stale.includes('id="o02-use-base"') && bare(stale).includes(bare(c.useBase(l, "2"))), l);
    assert.ok(bare(group(l, "loading")).includes(c.draftLoading(l)) && !bare(group(l, "loading")).includes(c.notOwnership(l)), l);
    assert.ok(bare(group(l, "missing")).includes(c.draftMissing(l)), l);
    assert.ok(bare(group(l, "closed")).includes(c.draftNotOpen(l)), l);
    assert.ok(bare(group(l, "error")).includes(c.draftFailed(l)), l);
  }
});

test("a group plan leads with its group and member count and names the draft only as a draft name", () => {
  for (const l of locales) {
    const p = plan({ change: { kind: "view_group", member: "", action: "join", target: "group-00112233aabbccdd", from: "2026-12-01" }, draft: { id: "vgd-1", name: "Ops reviewers", group: "group-00112233aabbccdd", new_group: true, member_count: 3 },
      violations: [{ kind: "", member: "", rule: "plan_limit" }] });
    const html = detail(l, p);
    const text = bare(html).replace(/⁨|⁩/g, "");
    assert.ok(text.includes(c.groupLead(l, "group-00112233aabbccdd", 3, "").replace(/⁨|⁩/g, "").trim().split(":")[0]), l);
    assert.ok(text.includes(c.draftName(l)) && text.includes("Ops reviewers") && text.includes(c.planLimit(l)) && text.includes(c.membersFact(l)), l);
    const h2 = /<h2[^>]*>([^]*?)<\/h2>/.exec(html)?.[1] ?? "";
    assert.ok(!h2.includes("Ops reviewers") && h2.includes("group-00112233aabbccdd"), l);
    assert.ok(html.indexOf(c.draftName(l)) < html.indexOf("o02-steps"), l);
    const held = plain(approval(l, p));
    assert.ok(held.includes(c.heldLead(l)), l);
    const existing = detail(l, plan({ change: { kind: "view_group", member: "", action: "join", target: "sample-view-north", from: "2026-12-01" }, target_name: "North view group",
      draft: { id: "vgd-2", name: "", group: "sample-view-north", new_group: false, member_count: 2 } }));
    assert.ok(!bare(existing).includes(c.draftName(l)) && bare(existing).includes("North view group"), l);
  }
});

test("group plan ranges keep each date whole in its own time element", () => {
  for (const l of locales) {
    const open = renderToStaticMarkup(<DatedRange l={l} from="2026-12-20" until="" />);
    assert.ok(/<time class="o02-nowrap" dateTime="2026-12-20">[^<]+<\/time>/.test(open) && !open.includes("hb-nowrap"), l);
    const closed = renderToStaticMarkup(<DatedRange l={l} from="2026-12-20" until="2027-01-05" />);
    assert.ok((closed.match(/<time class="o02-nowrap"/g) ?? []).length >= 2, l);
  }
});

test("an editor sees the draft reason, not the proposer reason", () => {
  for (const l of locales) {
    const html = approval(l, plan({ blocked: "own_draft", impact: computed }));
    assert.ok(plain(html).includes(c.ownDraft(l)) && !plain(html).includes(c.own(l)), l);
    assert.equal(c.refused(l, "own_draft"), c.ownDraft(l));
  }
});

test("a plan whose changed members were cut shows the count and one named button for each kind, and none when whole", async () => {
  const { planMembersPageOf } = await import("../../../../service/admin");
  const { appended } = await import("../../../../shared/more-rows");
  const member = (kind: string, id: string) => ({ kind, member: id, name: "", differs: [{ from: "2026-12-01", until: "" }], before: [], after: ["unit-1"] });
  const cut = plan({ kinds: [{ kind: "hierarchy", changed: 70, entries_before: 1, entries_after: 1, earliest: "2026-12-01", next: "acct-2" }, { kind: "payer", changed: 1, entries_before: 0, entries_after: 1, earliest: "", next: "" }],
    members: [member("hierarchy", "acct-1"), member("hierarchy", "acct-2"), member("payer", "acct-1")] });
  const read = async () => ({ rows: [], next: "" });
  for (const l of locales) {
    const html = renderToStaticMarkup(<Detail l={l} p={cut} page={page} parties={[]} members={[]} full readMembers={read} />);
    const out = plain(html);
    assert.ok(out.includes(c.membersShown(l, 3, 71)), l);
    assert.equal(html.split("o02-members-more-button").length - 1, 1, l);
    assert.ok(!detail(l, cut).includes("o02-members-foot"), l);
  }
  assert.equal(cut.members[2].kind, "payer");
  const whole = plan({ kinds: [{ kind: "hierarchy", changed: 1, entries_before: 1, entries_after: 1, earliest: "", next: "" }], members: [member("hierarchy", "acct-1")] });
  assert.ok(!renderToStaticMarkup(<Detail l="en" p={whole} page={page} parties={[]} members={[]} full readMembers={read} />).includes("o02-members-foot"));
  const old = plan({ kinds: [{ kind: "hierarchy", changed: 70, entries_before: 1, entries_after: 1, earliest: "" }], members: [{ member: "acct-1", name: "", differs: [], before: [], after: [] }] });
  assert.ok(!renderToStaticMarkup(<Detail l="en" p={old} page={page} parties={[]} members={[]} full readMembers={read} />).includes("o02-members-foot"));
  for (const bad of [{}, { members: [] }, { members: "x", next: "" }, { members: [], next: 4 }]) assert.equal(planMembersPageOf(bad), null, JSON.stringify(bad));
  const got = planMembersPageOf({ members: [member("hierarchy", "acct-3"), null, "x"], next: "acct-3", total: 70 });
  assert.deepEqual([got?.rows.length, got?.rows[0].kind, got?.rows[0].member, got?.next], [1, "hierarchy", "acct-3", "acct-3"]);
  const key = (m: { kind?: string; member: string }) => (m.kind ?? "") + "\u0000" + m.member;
  assert.deepEqual(appended(cut.members, [member("hierarchy", "acct-2"), member("hierarchy", "acct-3"), member("payer", "acct-3")], key).map(key), ["hierarchy\u0000acct-1", "hierarchy\u0000acct-2", "payer\u0000acct-1", "hierarchy\u0000acct-3", "payer\u0000acct-3"]);
});
