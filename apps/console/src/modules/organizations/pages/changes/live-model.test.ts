import { test } from "node:test";
import assert from "node:assert/strict";
import type { OrgMember, OrgParty, PlanRow } from "../../../../service/admin";
import { planOf } from "../../../../service/admin";
import { canActivate, showOpenCount, actionsFor, allMemberChoices, impactOf, keepsAttempt, lead, membersKept, nameOf, react, blocked, closing, currentOn, findPlan, memberChoices, minFrom, needsTarget, partition, steps, targetChoices } from "./live-model";

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

test("steps, blocks and closings are stated for every state", () => {
  assert.deepEqual(steps(plan({})), ["done", "done", "current"]);
  assert.deepEqual(steps(plan({ state: "draft" })), ["done", "current", "na"]);
  assert.deepEqual(steps(plan({ state: "rejected" })), ["done", "stopped", "na"]);
  assert.deepEqual(steps(plan({ state: "weird" })), ["unknown", "unknown", "unknown"]);
  assert.equal(canActivate(plan({ state: "activated", can_activate: true })), false);
  assert.equal(canActivate(plan({ can_activate: true })), true);
  assert.deepEqual([showOpenCount(16), showOpenCount(0), showOpenCount(NaN)], [true, false, false]);
  assert.deepEqual(steps(plan({ state: "activated" })), ["done", "done", "done"]);
  assert.deepEqual(steps(plan({ state: "discarded", closed: "discarded_stale" })), ["done", "done", "stopped"]);
  assert.deepEqual(steps(plan({ state: "discarded", closed: "" })), ["done", "done", "stopped"]);
  assert.equal(blocked(plan({ can_activate: true })), "");
  assert.equal(blocked(plan({ blocked: "own_proposal" })), "own");
  assert.equal(blocked(plan({ blocked: "stale" })), "stale");
  assert.equal(blocked(plan({ blocked: "effective_date_passed" })), "passed");
  assert.equal(blocked(plan({ blocked: "" })), "other");
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
  assert.deepEqual(["not_computed", "", "0", "computed"].map(impactOf), ["not_computed", "unknown", "unknown", "unknown"]);
  assert.deepEqual(allMemberChoices(parties, [[members[0]], [members[1]]]).map((c) => c.id), ["le-a", "acct-1", "acct-2", "unit-b", "le-x"]);
});

test("every failure has one reaction and only retries keep the attempt", () => {
  const cases: [string, string, string[], string][] = [
    ["propose", "invalid", ["member"], "field"], ["propose", "invalid", [], "reason"], ["activate", "invalid", ["x"], "reason"],
    ["propose", "refused", [], "limit"], ["activate", "refused", [], "own"], ["discard", "refused", [], "own"],
    ["activate", "stage", [], "stage"], ["activate", "conflict", [], "conflict"], ["discard", "denied", [], "denied"],
    ["propose", "unauthenticated", [], "signin"], ["propose", "limited", [], "wait"], ["activate", "busy", [], "retry"],
    ["activate", "unavailable", [], "retry"], ["propose", "stale", [], "renew"], ["propose", "partial", [], "partial"], ["activate", "wrong", [], "failed"],
  ];
  for (const [cmd, f, fields, want] of cases) assert.equal(react(cmd as "propose", f, fields), want, `${cmd} ${f}`);
  assert.deepEqual(["retry", "wait", "renew", "partial", "conflict", "field"].map((r) => keepsAttempt(r as "retry")), [true, true, false, false, false, false]);
});
