import { test } from "node:test";
import assert from "node:assert/strict";
import type { OrgMember, OrgParty } from "../../../../service/admin";
import { accounts, applies, covering, groups, label, loaded, newer, todayUTC, tree, validDay, type TreeNode } from "./live-model";

const parties: OrgParty[] = [
  { id: "le-b", name: "Beta", kind: "legal_entity", parent: "", state: "active" },
  { id: "le-a", name: "", kind: "legal_entity", parent: "", state: "active" },
  { id: "unit-1", name: "Unit one", kind: "internal_unit", parent: "le-a", state: "active" },
  { id: "unit-2", name: "Unit two", kind: "internal_unit", parent: "le-a", state: "suspended" },
  { id: "unit-3", name: "Unit three", kind: "internal_unit", parent: "", state: "active" },
];
const done = { complete: true, cursor: "" };
const m = (member: string, target: string | string[] = "", name = "", history: OrgMember["history"] = []): OrgMember =>
  ({ member, name, targets: (Array.isArray(target) ? target : target ? [target] : []).map((id) => ({ id, name: "" })), history });

function ids(t: ReturnType<typeof tree>): string[] {
  const out: string[] = [];
  const walk = (n: TreeNode) => { out.push(n.id); n.children.forEach(walk); };
  t.roots.forEach(walk);
  t.unattributed.forEach(walk);
  t.conflicts.forEach((x) => walk(x.node));
  t.unloaded.forEach(walk);
  t.cycles.forEach(walk);
  return out.sort();
}

test("every loaded party and member appears exactly once", () => {
  const got = tree(parties, [m("unit-1", "le-b"), m("acct-2", "unit-1", "Two"), m("acct-1", "unit-1"), m("acct-3"), m("acct-9", "missing"), m("unit-2"), m("acct-4", ["le-a", "le-b"])], done);
  assert.deepEqual(ids(got), ["acct-1", "acct-2", "acct-3", "acct-4", "acct-9", "le-a", "le-b", "missing", "unit-1", "unit-2", "unit-3"]);
  assert.deepEqual(got.roots.map((r) => [r.id, r.kind]), [["le-b", "legal_entity"], ["le-a", "legal_entity"], ["unit-3", "internal_unit"], ["missing", "unknown"]]);
  assert.equal(got.roots[0].children[0].source, "relationship");
  assert.deepEqual(got.roots[0].children[0].children.map((c) => [c.id, c.kind]), [["acct-1", "account"], ["acct-2", "account"]]);
  assert.deepEqual(got.unattributed.map((u) => u.id), ["unit-2", "acct-3"]);
  assert.deepEqual(got.conflicts.map((u) => [u.node.id, u.targets.length]), [["acct-4", 2]]);
  const under = tree(parties, [m("unit-1"), m("acct-7", "unit-1")], done);
  assert.deepEqual(under.unattributed.map((u) => [u.id, u.children.map((c) => c.id)]), [["unit-1", ["acct-7"]]]);
  assert.deepEqual(ids(under), ["acct-7", "le-a", "le-b", "unit-1", "unit-2", "unit-3"]);
  const plain = tree(parties, [], done);
  assert.deepEqual(plain.roots[1].children.map((c) => [c.id, c.source, c.state]), [["unit-1", "registry", "active"], ["unit-2", "registry", "suspended"]]);
});

test("unloaded parties are not given their registry parent, and cycles are shown", () => {
  const got = tree(parties, [m("acct-1", "unit-1")], { complete: false, cursor: "acct-1" });
  assert.deepEqual(got.unloaded.map((n) => n.id), ["le-b", "le-a", "unit-1", "unit-3", "unit-2"]);
  assert.deepEqual(got.unloaded[2].children.map((c) => c.id), ["acct-1"]);
  assert.deepEqual(got.roots, []);
  assert.deepEqual(ids(got), ["acct-1", "le-a", "le-b", "unit-1", "unit-2", "unit-3"]);
  const loop = tree([], [m("x", "y"), m("y", "x"), m("z", "x")], done);
  assert.deepEqual(loop.cycles.map((n) => n.id).sort(), ["x", "y", "z"]);
  assert.deepEqual(loop.roots.map((n) => n.id), []);
  const partial = tree([{ id: "u", name: "", kind: "internal_unit", parent: "ghost", state: "active" }], [], done);
  assert.deepEqual(partial.roots.map((n) => [n.id, n.kind]), [["ghost", "unknown"]]);
});

test("account rows separate conflict, unattributed, none and unloaded", () => {
  const h = [{ target: "unit-1", name: "", from: "2026-01-01", until: "2026-10-01" }, { target: "unit-2", name: "", from: "2026-10-01", until: "" }];
  const rows = accounts(parties, [m("unit-1", "le-a"), m("acct-1", "unit-2", "", h), m("acct-2"), m("acct-5", ["unit-1", "unit-2"])], [m("acct-1", "le-a"), m("acct-3", "le-b", "Three"), m("acct-2")], "2026-11-16", done, { complete: false, cursor: "acct-3" });
  assert.deepEqual(rows.map((r) => [r.id, r.unit.state, r.payer.state, r.entryFrom]), [
    ["acct-1", "target", "target", "2026-10-01"], ["acct-2", "unattributed", "unattributed", ""], ["acct-3", "none", "target", ""], ["acct-5", "conflict", "unloaded", ""],
  ]);
  assert.equal(rows[2].name, "Three");
});

test("groups invert members and list the ones outside", () => {
  const view: OrgMember[] = [
    { member: "acct-3", name: "", targets: [{ id: "g-north", name: "North" }, { id: "g-south", name: "South" }], history: [] },
    { member: "acct-4", name: "Four", targets: [{ id: "g-south", name: "South" }], history: [] },
    m("acct-5"),
  ];
  const got = groups(view);
  assert.deepEqual(got.groups.map((g) => [g.id, g.members.map((x) => x.id)]), [["g-north", ["acct-3"]], ["g-south", ["acct-3", "acct-4"]]]);
  assert.deepEqual(got.outside.map((x) => x.member), ["acct-5"]);
});

test("covering uses an exclusive end, and small helpers", () => {
  const h = [{ target: "a", name: "", from: "2026-01-01", until: "2026-10-01" }, { target: "b", name: "", from: "2026-10-01", until: "" }];
  assert.equal(covering(h, "2026-09-30")?.target, "a");
  assert.equal(covering(h, "2026-10-01")?.target, "b");
  assert.equal(covering(h, "2025-12-31"), undefined);
  const both = [{ target: "g1", name: "", from: "2026-01-01", until: "" }, { target: "g2", name: "", from: "2026-01-01", until: "" }];
  assert.deepEqual(both.filter((x) => applies(x, "2026-11-16")).map((x) => x.target), ["g1", "g2"]);
  assert.equal(applies(h[0], "2026-10-01"), false);
  assert.equal(loaded("b", { complete: false, cursor: "a" }), false);
  assert.equal(loaded("a", { complete: false, cursor: "a" }), true);
  assert.equal(newer(3, 2), true);
  assert.equal(newer(2, 2), false);
  assert.equal(label("", "x"), "x");
  assert.equal(todayUTC(new Date("2026-11-16T23:30:00Z")), "2026-11-16");
  for (const ok of ["2026-11-16", "2024-02-29"]) assert.equal(validDay(ok), true, ok);
  for (const bad of ["2026-02-30", "2026-13-01", "2026-1-01", "", "x"]) assert.equal(validDay(bad), false, bad);
});
