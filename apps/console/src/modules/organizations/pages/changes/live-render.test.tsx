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
const { Approval, Detail, ImpactActions, Steps, decisionLabel } = await import("./Live");

const locales: Locale[] = ["en", "ko", "ar"];
const computed = { money: "computed", publication: "computed", erp: "computed" };
const notComputed = { money: "not_computed", publication: "not_computed", erp: "not_computed" };
const plan = (over: Record<string, unknown>): PlanRow =>
  planOf({ id: "plan-1", state: "sealed", change: { kind: "hierarchy", member: "acct-1", action: "set", target: "unit-1", from: "2026-12-01" }, impact: notComputed, blocked: "impact_not_computed", ...over });
const page: PlansPage = { revision: "r", active_revision: 1, open_plans: 1, open_limit: 16, plans: [], you: "u" };
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
