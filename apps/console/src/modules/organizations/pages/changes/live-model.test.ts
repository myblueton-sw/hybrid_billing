import { test } from "node:test";
import assert from "node:assert/strict";
import type { OrgMember, OrgParty, PlanRow } from "../../../../service/admin";
import { planOf } from "../../../../service/admin";
import { renderToStaticMarkup } from "react-dom/server";
import { createElement } from "react";
import { arrival, currentView, draftLoad, groupRefusal, planTitle } from "./live-model";
import { adopted, afterConflict, baseBlocks, baseField, baseStale, canonicalRevision, conflictShown, fill, formStep, handoffHref, memberMode, prefilled, readHandoff, type Carried } from "./live-model";
import { canActivate, showOpenCount, actionsFor, allMemberChoices, held, impactDone, impactOf, impactUnread, keepsAttempt, lead, membersKept, nameOf, react, refusal, blocked, closing, currentOn, findPlan, memberChoices, minFrom, needsTarget, partition, steps, targetChoices } from "./live-model";

const parties: OrgParty[] = [
  { id: "le-a", name: "A Corp", kind: "legal_entity", parent: "", state: "active" },
  { id: "le-x", name: "X Corp", kind: "legal_entity", parent: "", state: "suspended" },
  { id: "unit-b", name: "B unit", kind: "internal_unit", parent: "le-a", state: "active" },
];
const members: OrgMember[] = [
  { member: "acct-1", name: "Account 1", targets: [{ id: "pool-1", name: "Pool one" }], history: [{ target: "pool-1", name: "Pool one", from: "2026-01-01", until: "" }] },
  { member: "acct-2", name: "", targets: [], history: [{ target: "pool-2", name: "", from: "2026-01-01", until: "2026-06-01" }] },
  { member: "unit-b", name: "B unit", targets: [], history: [] },
];
const plan = (over: Record<string, unknown>): PlanRow => planOf({ id: "p", state: "sealed", ...over });
const computed = { money: "computed", publication: "computed", erp: "computed" };
const notComputed = { money: "not_computed", publication: "not_computed", erp: "not_computed" };

test("actions follow the kind and only end has no target", () => {
  assert.deepEqual(actionsFor("view_group"), ["join", "leave"]);
  for (const k of ["hierarchy", "payer", "contract_pool", "allocation_pool"] as const) assert.deepEqual(actionsFor(k), ["set", "end"]);
  assert.deepEqual(["set", "end", "join", "leave"].map((a) => needsTarget(a as "set")), [true, false, true, true]);
});

test("member choices join units and members without duplicates and never offer a legal entity", () => {
  assert.deepEqual(memberChoices(parties, members).map((c) => c.id), ["acct-1", "acct-2", "unit-b"]);
  assert.deepEqual(memberChoices(parties, members, "hierarchy").map((c) => c.id), ["acct-1", "acct-2", "unit-b"]);
  for (const k of ["payer", "contract_pool", "allocation_pool", "view_group"] as const) assert.deepEqual(memberChoices(parties, members, k).map((c) => c.id), ["acct-1", "acct-2"]);
  const closed: OrgParty[] = [...parties, { id: "unit-c", name: "C unit", kind: "internal_unit", parent: "le-a", state: "suspended" }];
  assert.deepEqual(memberChoices(closed, [...members, { member: "unit-c", name: "C unit", targets: [], history: [] }], "hierarchy").map((c) => c.id), ["acct-1", "acct-2", "unit-b"]);
  assert.equal(memberChoices([], members).find((c) => c.id === "acct-2")?.name, "acct-2");
});

test("target choices follow the server's rules for each kind", () => {
  assert.deepEqual(targetChoices("hierarchy", "unit-b", parties, members).map((c) => c.id), ["le-a"]);
  assert.deepEqual(targetChoices("hierarchy", "acct-1", parties, members).map((c) => c.id), ["le-a", "unit-b"]);
  assert.deepEqual(targetChoices("payer", "acct-1", parties, members).map((c) => c.id), ["le-a"]);
  assert.deepEqual(targetChoices("allocation_pool", "acct-9", parties, members).map((c) => c.id), ["pool-1", "pool-2"]);
  assert.deepEqual(targetChoices("allocation_pool", "pool-1", parties, members).map((c) => c.id), ["pool-2"]);
});

test("the current relationship comes from history on the day", () => {
  assert.deepEqual(currentOn("acct-2", members, "2026-05-31"), [{ id: "pool-2", name: "pool-2" }]);
  assert.deepEqual(currentOn("acct-2", members, "2026-06-01"), []);
  assert.equal(currentOn("acct-9", members, "2026-06-01"), null);
});

test("four steps follow target, impact, approval and apply, and impact comes from the row", () => {
  assert.deepEqual(steps(plan({ state: "draft" })), ["current", "na", "na", "na"]);
  assert.deepEqual(steps(plan({ impact: notComputed, blocked: "impact_not_computed" })), ["done", "held", "na", "na"]);
  assert.deepEqual(steps(plan({ blocked: "impact_unavailable" })), ["done", "held", "na", "na"]);
  assert.deepEqual(steps(plan({ impact: notComputed, blocked: "stale", stale: true })), ["done", "held", "na", "na"]);
  assert.deepEqual(steps(plan({ impact: notComputed, blocked: "effective_date_passed" })), ["done", "held", "na", "na"]);
  assert.deepEqual(steps(plan({ impact: notComputed, can_activate: true })), ["done", "held", "na", "na"]);
  assert.deepEqual(steps(plan({ impact: computed, can_activate: true })), ["done", "done", "current", "na"]);
  for (const b of ["stale", "effective_date_passed", "own_proposal"]) assert.deepEqual(steps(plan({ impact: computed, blocked: b })), ["done", "done", "held", "na"], b);
  assert.deepEqual(steps(plan({ state: "activated", impact: computed })), ["done", "done", "done", "done"]);
  assert.deepEqual(steps(plan({ state: "activated", impact: notComputed })), ["done", "held", "done", "done"]);
  assert.deepEqual(steps(plan({ state: "rejected" })), ["stopped", "na", "na", "na"]);
  for (const closed of ["discarded", "discarded_stale", "discarded_expired"]) {
    assert.deepEqual(steps(plan({ state: "discarded", closed, impact: notComputed })), ["done", "stopped", "na", "na"], closed);
    assert.deepEqual(steps(plan({ state: "discarded", closed, impact: computed })), ["done", "done", "stopped", "na"], closed);
  }
  assert.deepEqual(steps(plan({ state: "weird" })), ["unknown", "unknown", "unknown", "unknown"]);
  for (const part of ["money", "publication", "erp"]) {
    const one = { ...computed, [part]: "not_computed" };
    assert.equal(impactDone(plan({ impact: one })), false, part);
    assert.deepEqual(steps(plan({ impact: one, can_activate: true })), ["done", "held", "na", "na"], part);
    assert.equal(impactUnread(plan({ impact: { ...computed, [part]: "unknown" } })), true, part);
    assert.equal(impactUnread(plan({ impact: one })), false, part);
  }
  assert.equal(impactDone(plan({ impact: computed })), true);
  assert.equal(impactUnread(plan({})), true);
});

test("steps, blocks and closings are stated for every state", () => {
  assert.equal(canActivate(plan({ state: "activated", can_activate: true })), false);
  assert.equal(canActivate(plan({ can_activate: true, impact: computed })), true);
  assert.equal(canActivate(plan({ can_activate: true, impact: notComputed })), false);
  assert.equal(canActivate(plan({ can_activate: true })), false);
  for (const part of ["money", "publication", "erp"]) assert.equal(canActivate(plan({ can_activate: true, impact: { ...computed, [part]: "not_computed" } })), false, part);
  assert.equal(blocked(plan({ can_activate: true, impact: notComputed })), "impact");
  assert.equal(blocked(plan({ can_activate: true, impact: { ...computed, erp: "unknown" } })), "impact_unknown");
  assert.equal(blocked(plan({ blocked: "", impact: computed })), "other");
  assert.deepEqual([showOpenCount(16), showOpenCount(0), showOpenCount(NaN)], [true, false, false]);
  assert.equal(blocked(plan({ can_activate: true, impact: computed })), "");
  assert.equal(blocked(plan({ blocked: "impact_not_computed" })), "impact");
  assert.equal(blocked(plan({ blocked: "impact_unavailable" })), "impact_unknown");
  assert.deepEqual(["impact", "impact_unknown", "own", "stale", "passed", "other", ""].map((b) => held(b as "impact")), [true, true, false, false, false, false, false]);
  assert.equal(blocked(plan({ blocked: "own_proposal" })), "own");
  assert.equal(blocked(plan({ blocked: "stale" })), "stale");
  assert.equal(blocked(plan({ blocked: "effective_date_passed" })), "passed");
  assert.equal(blocked(plan({ blocked: "" })), "impact_unknown");
  assert.equal(blocked(plan({ state: "activated" })), "");
  assert.equal(closing(plan({ state: "activated" })), "activated");
  assert.equal(closing(plan({ state: "rejected" })), "rejected");
  for (const c of ["discarded", "discarded_stale", "discarded_expired"]) assert.equal(closing(plan({ state: "discarded", closed: c })), c);
  assert.equal(closing(plan({ state: "discarded", closed: "x" })), "other");
});

test("open plans and results are split, and the date floor is today", () => {
  const all = [plan({ id: "a" }), plan({ id: "b", state: "activated" }), plan({ id: "c", state: "draft" }), plan({ id: "d", state: "rejected" })];
  const { open, closed } = partition(all);
  assert.deepEqual([open.map((p) => p.id), closed.map((p) => p.id)], [["a", "c"], ["b", "d"]]);
  assert.equal(findPlan(all, "d")?.state, "rejected");
  assert.equal(findPlan(all, ""), undefined);
  assert.equal(minFrom("2026-11-17", "2026-11-17"), true);
  assert.equal(minFrom("2026-11-17", "2026-11-16"), false);
});

test("a payer move leads with the dated change and names fall back to IDs", () => {
  const move = plan({ change: { kind: "payer", member: "acct-1", action: "set", target: "le-b", from: "2026-12-01" }, members: [{ member: "acct-1", before: ["le-a"], after: ["le-a", "le-b"], differs: [{ from: "2026-12-01", until: "" }] }] });
  assert.deepEqual(lead(move), { from: "2026-12-01", target: "le-b" });
  assert.deepEqual(move.members[0].after, ["le-a", "le-b"]);
  assert.deepEqual(lead(plan({ change: { action: "leave", target: "g", from: "2026-12-01" } })), { from: "2026-12-01", target: "" });
  assert.equal(nameOf("le-a", parties, members), "A Corp");
  assert.equal(nameOf("pool-1", parties, members), "Pool one");
  assert.equal(nameOf("acct-1", parties, members), "Account 1");
  assert.equal(nameOf("le-b", parties, members), "le-b");
  assert.equal(nameOf("acct-2", parties, members), "acct-2");
});

test("closed plans keep no members and impact never reads as zero", () => {
  assert.deepEqual(["sealed", "activated", "rejected", "discarded"].map((s) => membersKept(plan({ state: s }))), [true, true, false, false]);
  assert.deepEqual(["not_computed", "", "0", "computed", "Computed"].map(impactOf), ["not_computed", "unknown", "unknown", "computed", "unknown"]);
  assert.deepEqual(allMemberChoices(parties, [[members[0]], [members[1]]]).map((c) => c.id), ["acct-1", "acct-2", "unit-b"]);
  assert.deepEqual(allMemberChoices(parties, [[members[0]], [members[1]]], "payer").map((c) => c.id), ["acct-1", "acct-2"]);
});

test("every failure has one reaction and only retries keep the attempt", () => {
  const cases: [string, string, string[], string][] = [
    ["propose", "invalid", ["member"], "field"], ["propose", "invalid", [], "reason"], ["activate", "invalid", ["x"], "reason"],
    ["propose", "refused", [], "limit"], ["activate", "refused", [], "refused"], ["discard", "refused", [], "refused"],
    ["activate", "stage", [], "stage"], ["activate", "conflict", [], "conflict"], ["discard", "denied", [], "denied"],
    ["propose", "unauthenticated", [], "signin"], ["propose", "limited", [], "wait"], ["activate", "busy", [], "retry"],
    ["activate", "unavailable", [], "retry"], ["propose", "stale", [], "renew"], ["propose", "partial", [], "partial"], ["activate", "wrong", [], "failed"],
  ];
  for (const [cmd, f, fields, want] of cases) assert.equal(react(cmd as "propose", f, fields), want, `${cmd} ${f}`);
  assert.deepEqual(["retry", "wait", "renew", "partial", "conflict", "field"].map((r) => keepsAttempt(r as "retry")), [true, true, false, false, false, false]);
});

test("a step-up refusal or cancellation keeps the attempt", () => {
  for (const cmd of ["propose", "activate", "discard"] as const) {
    for (const f of ["reauth", "reauth_cancelled"]) {
      assert.equal(react(cmd, f, []), "reauth");
      assert.equal(keepsAttempt(react(cmd, f, [])), true);
    }
  }
});

test("a refused activation is explained by the reloaded row", () => {
  assert.equal(refusal(plan({ blocked: "impact_not_computed" })), "impact");
  assert.equal(refusal(plan({ blocked: "impact_unavailable" })), "impact_unknown");
  assert.equal(refusal(plan({ blocked: "own_proposal" })), "own");
  assert.equal(refusal(plan({ blocked: "stale" })), "other");
  assert.equal(refusal(plan({ blocked: "effective_date_passed" })), "other");
  assert.equal(refusal(plan({ can_activate: true, impact: computed })), "other");
  assert.equal(refusal(plan({ state: "activated", blocked: "impact_not_computed" })), "other");
  assert.equal(refusal(undefined), "other");
});

const link = (q: string) => readHandoff(new URLSearchParams(q), "2026-11-18");

test("the handoff link carries kind, member, a date from today, the base and the language, never names", () => {
  assert.equal(handoffHref({ kind: "payer", member: "acct-1", from: "2026-11-18", base: 3, lang: "ko" }), "#/organization-changes?view=targets&kind=payer&member=acct-1&from=2026-11-18&base=3&lang=ko");
  assert.equal(handoffHref({ kind: "view_group", member: "acct-2", from: "", base: 0, lang: "" }), "#/organization-changes?view=targets&kind=view_group&member=acct-2&base=0");
  const back = readHandoff(new URLSearchParams(handoffHref({ kind: "hierarchy", member: "unit-b", from: "2026-12-01", base: 7, lang: "ar" }).split("?")[1]), "2026-11-18");
  assert.deepEqual(back, { kind: "hierarchy", member: "unit-b", from: "2026-12-01", passed: "", base: 7, dropped: false, draft: "" });
});

test("link values are read once with unknown kinds, malformed IDs and bad bases dropped", () => {
  assert.equal(link("view=targets&lang=en"), null);
  assert.deepEqual(link("kind=other&member=acct-1&base=2"), { kind: "", member: "acct-1", from: "", passed: "", base: 2, dropped: true, draft: "" });
  assert.deepEqual(link("kind=payer&member=bad%20id"), { kind: "payer", member: "", from: "", passed: "", base: null, dropped: true, draft: "" });
  assert.deepEqual(link("member=acct-1&from=2026-11-17"), { kind: "", member: "acct-1", from: "", passed: "2026-11-17", base: null, dropped: false, draft: "" });
  assert.deepEqual(link("member=acct-1&from=2026-11-18"), { kind: "", member: "acct-1", from: "2026-11-18", passed: "", base: null, dropped: false, draft: "" });
  assert.equal(link("member=acct-1&from=2026-02-30")?.dropped, true);
  assert.equal(link("member=acct-1&from=2026-02-30")?.from, "");
  for (const bad of ["", "-1", "01", "1.0", "x", "18446744073709551616", "9007199254740993", "+1"]) {
    const got = link("member=acct-1&base=" + encodeURIComponent(bad));
    assert.equal(got?.base, null, bad);
    assert.equal(got?.dropped, true, bad);
  }
  assert.equal(link("base=0")?.base, 0);
  assert.equal(link("base=9007199254740991")?.base, 9007199254740991);
  assert.equal(canonicalRevision("12"), 12);
  assert.equal(canonicalRevision("1234567890123456"), 1234567890123456);
  assert.equal(canonicalRevision("12345678901234567"), null);
});

test("a prefill note needs a carried value and the base compares with the active revision", () => {
  assert.equal(prefilled({ kind: "", member: "", from: "", passed: "2026-01-01", base: null, dropped: true, draft: "" }), false);
  for (const h of [{ kind: "payer" as const }, { member: "a" }, { from: "2026-12-01" }, { base: 0 }])
    assert.equal(prefilled({ kind: "", member: "", from: "", passed: "", base: null, dropped: false, draft: "", ...h }), true);
  assert.equal(baseStale(null, 4), false);
  assert.equal(baseStale(4, 4), false);
  assert.equal(baseStale(3, 4), true);
  assert.equal(baseStale(0, 1), true);
});

test("a refused propose shows the base notice only when the reloaded active revision differs from the carried base", () => {
  assert.equal(react("propose", "conflict", []), "conflict");
  assert.equal(conflictShown(3, 4), "base");
  assert.equal(conflictShown(4, 4), "plain");
  assert.equal(conflictShown(null, 4), "plain");
  assert.equal(conflictShown(3, null), "plain");
});

test("fill puts nodes in place of placeholders and leaves other text", () => {
  const nb = (inner: string) => `<span class="hb-nowrap" style="white-space:nowrap">${inner}</span>`;
  const html = renderToStaticMarkup(createElement("p", null, fill("From {r} to {a}, {x}.", { r: createElement("bdi", { dir: "ltr" }, 1), a: createElement("b", null, 2) })));
  assert.equal(html, `<p>${nb('From <bdi dir="ltr">1</bdi>')} ${nb("to <b>2</b>,")} {x}.</p>`);
  const ko = renderToStaticMarkup(createElement("p", null, fill("관계 리비전이 {r}에서 {a}(으)로 바뀌었습니다.", { r: "0", a: "1" })));
  assert.equal(ko, `<p>관계 ${nb("리비전이 0에서")} ${nb("1(으)로")} 바뀌었습니다.</p>`);
  const date = renderToStaticMarkup(createElement("p", null, fill("적용 시작일({d})이 미리", { d: "D" })));
  assert.equal(date, `<p>적용 ${nb("시작일(D)이")} 미리</p>`);
  assert.equal(renderToStaticMarkup(createElement("p", null, fill("{a}", { a: "x" }))), "<p>x</p>");
});

const carry = (base: number | null, refused = false): Carried => ({ kind: "hierarchy", member: "acct-1", from: "", passed: "", base, dropped: false, draft: "", refused });

test("unsaved input follows user edits only: base adoption keeps it, a proposal or leaving clears it", () => {
  assert.deepEqual(formStep("edit", false), { touched: true, clearAttempt: true, resetForm: false });
  assert.deepEqual(formStep("adopt", false), { touched: false, clearAttempt: true, resetForm: false });
  assert.deepEqual(formStep("adopt", true), { touched: true, clearAttempt: true, resetForm: false });
  assert.deepEqual(formStep("proposed", true), { touched: false, clearAttempt: true, resetForm: true });
  assert.deepEqual(formStep("left", true), { touched: false, clearAttempt: false, resetForm: true });
  assert.deepEqual(formStep("left", false), { touched: false, clearAttempt: false, resetForm: false });
});

test("a stale base blocks the proposal until adopted and is sent only when carried", () => {
  assert.equal(baseBlocks(carry(1), 2), true);
  assert.equal(baseBlocks(carry(2), 2), false);
  assert.equal(baseBlocks(carry(null), 2), false);
  assert.equal(baseBlocks(null, 2), false);
  assert.equal(baseBlocks(carry(1), null), false);
  const moved = adopted(carry(1, true), 4);
  assert.deepEqual([moved.base, moved.refused, moved.member], [4, false, "acct-1"]);
  assert.equal(baseBlocks(moved, 4), false);
  assert.deepEqual(baseField(carry(0)), { base: "0" });
  assert.deepEqual(baseField(carry(null)), {});
  assert.deepEqual(baseField(null), {});
});

test("the member mode is decided once the choices settle and never after an edit", () => {
  assert.equal(memberMode(false, false, false), null);
  assert.equal(memberMode(true, true, false), "list");
  assert.equal(memberMode(true, false, false), "id");
  assert.equal(memberMode(true, false, true), "keep");
  assert.equal(memberMode(true, true, true), "keep");
});

test("a conflict refusal marks the base only for a proposal whose carried base differs from the reloaded revision", () => {
  const stale = afterConflict(carry(1), "propose", 2);
  assert.equal(stale.shown, "base");
  assert.equal(stale.handoff?.refused, true);
  for (const [h, cmd, r] of [[carry(2), "propose", 2], [carry(null), "propose", 2], [null, "propose", 2], [carry(1), "activate", 2], [carry(1), "propose", null]] as const) {
    const got = afterConflict(h, cmd, r);
    assert.equal(got.shown, "plain");
    assert.equal(got.handoff, h);
  }
});

test("the Current aside waits for the same listing read as the member choices before it claims anything", () => {
  assert.equal(currentView(false, false, null), "loading");
  assert.equal(currentView(false, true, [{ id: "u", name: "U" }]), "loading");
  assert.equal(currentView(true, false, null), "not_in_listing");
  assert.equal(currentView(true, true, null), "none_of_kind");
  assert.equal(currentView(true, true, []), "none");
  assert.equal(currentView(true, false, [{ id: "u", name: "U" }]), "targets");
});

test("a draft link carries only the draft, the date and the base; kind and member are ignored", () => {
  assert.deepEqual(link("draft=vgd-0123456789abcdef&from=2026-12-01&base=3&kind=payer&member=acct-1"),
    { kind: "", member: "", from: "2026-12-01", passed: "", base: 3, dropped: false, draft: "vgd-0123456789abcdef" });
  for (const bad of ["vgd-1", "VGD-0123456789ABCDEF", "plan-0123456789abcdef", "", "vgd-0123456789abcdeg"]) {
    const got = link("draft=" + encodeURIComponent(bad));
    assert.equal(got?.draft, "", bad);
    assert.equal(got?.dropped, true, bad);
  }
  assert.equal(prefilled({ kind: "", member: "", from: "", passed: "", base: null, dropped: false, draft: "vgd-0123456789abcdef" }), true);
});

test("group plans are titled by their group, drafts load into a state and propose refusals map", () => {
  const plan = (over: Partial<PlanRow>) => ({ ...planOf({ change: { target: "group-1" }, member_name: "M" }), ...over });
  assert.equal(planTitle(plan({})), "M");
  assert.equal(planTitle(plan({ draft: { id: "vgd-1", name: "Ops", group: "group-1", new_group: true, member_count: 2 } })), "group-1");
  assert.equal(planTitle(plan({ draft: { id: "vgd-1", name: "Ops", group: "group-1", new_group: true, member_count: 2 }, target_name: "Ops" })), "Ops");
  assert.equal(draftLoad("error"), "error");
  assert.equal(draftLoad(null), "missing");
  assert.equal(draftLoad({ state: "open" }), "ready");
  for (const s of ["proposed", "activated", "discarded", "unknown"]) assert.equal(draftLoad({ state: s }), "closed");
  assert.equal(groupRefusal("field", ["from"]), "field");
  assert.equal(groupRefusal("field", []), "draft");
  assert.equal(groupRefusal("reason", []), "draft");
  assert.equal(groupRefusal("conflict", []), "other");
});

test("an editor of the plan's draft is blocked with its own reason", () => {
  const p = planOf({ state: "sealed", blocked: "own_draft", impact: { money: "computed", publication: "computed", erp: "computed" } });
  assert.equal(blocked(p), "own_draft");
  assert.equal(refusal(p), "own_draft");
  assert.equal(blocked(planOf({ state: "sealed", blocked: "own_proposal", impact: { money: "computed", publication: "computed", erp: "computed" } })), "own");
});

test("an arrival is announced only when the loaded plan is in that state", () => {
  const p = (over: Record<string, unknown>): PlanRow => planOf({ id: "p", state: "sealed", ...over });
  assert.deepEqual(arrival("sealed", p({})), { what: "sealed", revision: 0 });
  assert.deepEqual(arrival("activated:9", p({ state: "activated", activated_revision: 4 })), { what: "activated", revision: 4 });
  assert.deepEqual(arrival("open", undefined), { what: "open", revision: 0 });
  for (const [text, plan] of [["activated:9", p({})], ["activated:9", p({ state: "activated" })], ["discarded", p({})], ["sealed", undefined], ["rejected", p({ state: "discarded" })], ["sealed:x", p({})], ["", p({})]] as const) {
    assert.equal(arrival(text, plan), null, text);
  }
});
