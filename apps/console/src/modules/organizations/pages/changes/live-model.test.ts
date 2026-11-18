import { test } from "node:test";
import assert from "node:assert/strict";
import type { OrgMember, OrgParty, PlanRow } from "../../../../service/admin";
import { planOf } from "../../../../service/admin";
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

test("member choices join parties and members without duplicates", () => {
  assert.deepEqual(memberChoices(parties, members).map((c) => c.id), ["le-a", "acct-1", "acct-2", "unit-b", "le-x"]);
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
  assert.deepEqual(allMemberChoices(parties, [[members[0]], [members[1]]]).map((c) => c.id), ["le-a", "acct-1", "acct-2", "unit-b", "le-x"]);
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
