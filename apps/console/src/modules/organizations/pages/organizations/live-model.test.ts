import { test } from "node:test";
import assert from "node:assert/strict";
import type { OrgMember, OrgParty } from "../../../../service/admin";
import {
  accountRows, appended, appliedKeys, applies, cleared, clearedAll, countOf, covering, filterParams, FilterRefused, firstPage, joinKinds, label, legacyKind, linkedId, narrowing, newer,
  nextFocus, noFilters, pinnedRead, readFilters, refusal, sideOf,
  todayUTC, validDay, validSearch,
} from "./live-model";

const parties: OrgParty[] = [
  { id: "le-b", name: "Beta", kind: "legal_entity", parent: "", state: "active" },
  { id: "le-a", name: "", kind: "legal_entity", parent: "", state: "active" },
  { id: "unit-1", name: "Unit one", kind: "internal_unit", parent: "le-a", state: "active" },
  { id: "unit-2", name: "Unit two", kind: "internal_unit", parent: "le-a", state: "suspended" },
  { id: "unit-3", name: "Unit three", kind: "internal_unit", parent: "", state: "active" },
];
const m = (member: string, target: string | string[] = "", name = "", history: OrgMember["history"] = []): OrgMember =>
  ({ member, name, targets: (Array.isArray(target) ? target : target ? [target] : []).map((id) => ({ id, name: "" })), history });

test("account rows join the other kinds by exact lookup and never show a member as not loaded", () => {
  const h = [{ target: "unit-1", name: "", from: "2026-01-01", until: "2026-10-01" }, { target: "unit-2", name: "", from: "2026-10-01", until: "" }];
  const withHistory = (member: string, target: string | string[] = "", name = "") => m(member, target, name, [{ target: "x", name: "", from: "2026-01-01", until: "" }]);
  const primary = [m("acct-1", "unit-2", "", h), withHistory("acct-2"), m("acct-3", "", "Three"), m("acct-5", ["unit-1", "unit-2"], "", h)];
  const rows = accountRows(primary, "hierarchy", { payer: [withHistory("acct-1", "le-a"), withHistory("acct-2"), withHistory("acct-5", ["le-a", "le-b"])] }, "2026-11-16");
  assert.deepEqual(rows.map((r) => [r.id, r.unit.state, r.payer.state, r.other, r.entryFrom]), [
    ["acct-1", "target", "target", null, "2026-10-01"], ["acct-2", "unattributed", "unattributed", null, ""], ["acct-3", "none", "none", null, ""], ["acct-5", "conflict", "conflict", null, ""],
  ]);
  assert.equal(rows[2].name, "Three");
  const byPayer = accountRows([withHistory("acct-1", "le-a")], "payer", { hierarchy: [m("acct-1", "unit-2", "One", h)] }, "2026-09-30");
  assert.deepEqual([byPayer[0].unit, byPayer[0].payer.state, byPayer[0].other, byPayer[0].entryFrom, byPayer[0].name], [{ state: "target", target: { id: "unit-2", name: "" } }, "target", null, "2026-01-01", "acct-1"]);
  const byGroup = accountRows([withHistory("acct-1", ["g-1", "g-2"]), m("acct-2")], "view_group", { hierarchy: [], payer: [] }, "2026-11-16");
  assert.deepEqual(byGroup.map((r) => [r.other?.state, r.unit.state, r.payer.state]), [["conflict", "none", "none"], ["none", "none", "none"]]);
  assert.deepEqual(accountRows([m("acct-9", "unit-1", "", h)], "view_group", {}, "2026-11-16").map((r) => [r.unit.state, r.payer.state]), [["none", "none"]]);
  assert.deepEqual(joinKinds("hierarchy"), ["payer"]);
  assert.deepEqual(joinKinds("payer"), ["hierarchy"]);
  assert.deepEqual(joinKinds("allocation_pool"), ["hierarchy", "payer"]);
  assert.deepEqual(sideOf(undefined), { state: "none" });
});

test("filters read from the link reject what the server would refuse and write back only non-defaults", () => {
  const ok = readFilters(new URLSearchParams("entity=le-a&rk=payer&ua=1&q=north"));
  assert.deepEqual(ok, { filters: { entity: "le-a", rk: "payer", ua: true, q: "north" }, invalid: [] });
  const bad = readFilters(new URLSearchParams("entity=bad id&rk=other&ua=true&q=" + "가".repeat(22)));
  assert.deepEqual(bad, { filters: noFilters, invalid: ["entity", "rk", "ua", "q"] });
  assert.deepEqual(readFilters(new URLSearchParams("q=")).invalid, ["q"]);
  assert.deepEqual(readFilters(new URLSearchParams("")), { filters: noFilters, invalid: [] });
  assert.equal(validSearch("가".repeat(21)), true);
  assert.equal(validSearch("a".repeat(64)), true);
  assert.equal(validSearch("a".repeat(65)), false);
  assert.deepEqual(filterParams(noFilters), { entity: null, rk: null, ua: null, q: null });
  assert.deepEqual(filterParams({ entity: "le-a", rk: "view_group", ua: true, q: "x" }), { entity: "le-a", rk: "view_group", ua: "1", q: "x" });
});

test("chips show only non-default filters that apply in the view, and removal moves focus to a neighbour", () => {
  const all = { entity: "le-a", rk: "payer" as const, ua: true, q: "north" };
  assert.deepEqual(appliedKeys("accounts", all), ["q", "entity", "rk", "ua"]);
  assert.deepEqual(appliedKeys("groups", all), ["q", "entity"]);
  assert.deepEqual(narrowing("accounts", { ...noFilters, rk: "payer" }), []);
  assert.deepEqual(narrowing("accounts", { ...noFilters, rk: "payer", ua: true }), ["rk", "ua"]);
  assert.deepEqual(narrowing("accounts", all), ["q", "entity", "rk", "ua"]);
  assert.deepEqual(narrowing("history", all), []);
  assert.deepEqual(narrowing("groups", all), ["q", "entity"]);
  assert.deepEqual(appliedKeys("history", all), ["rk"]);
  assert.deepEqual(appliedKeys("tree", all), []);
  assert.deepEqual(appliedKeys("accounts", noFilters), []);
  assert.deepEqual(cleared(all, "rk"), { ...all, rk: "hierarchy" });
  assert.deepEqual(clearedAll("history", all), { ...all, rk: "hierarchy" });
  assert.deepEqual(clearedAll("accounts", all), noFilters);
  assert.equal(nextFocus(["q", "entity", "rk"], "q"), "entity");
  assert.equal(nextFocus(["q", "entity", "rk"], "rk"), "entity");
  assert.equal(nextFocus(["q"], "q"), "");
});

test("pages append in cursor order and the count never claims a total it does not know", () => {
  const start = firstPage<string>();
  assert.deepEqual([start.rows, start.status, start.next, start.stopped], [[], "loading", "", false]);
  const one = appended(start, ["a", "b"], "b", false);
  const two = appended({ ...one, status: "more" }, ["c"], "", false);
  assert.deepEqual([two.rows, two.status, two.next], [["a", "b", "c"], "ready", ""]);
  assert.deepEqual(countOf(one), { n: 2, more: true });
  assert.deepEqual(countOf(two), { n: 3, more: false });
  const stopped = appended(start, [], "m-9", true);
  assert.deepEqual([countOf(stopped), stopped.stopped], [{ n: 0, more: true }, true]);
});

test("covering uses an exclusive end, and small helpers", () => {
  const h = [{ target: "a", name: "", from: "2026-01-01", until: "2026-10-01" }, { target: "b", name: "", from: "2026-10-01", until: "" }];
  assert.equal(covering(h, "2026-09-30")?.target, "a");
  assert.equal(covering(h, "2026-10-01")?.target, "b");
  assert.equal(covering(h, "2025-12-31"), undefined);
  const both = [{ target: "g1", name: "", from: "2026-01-01", until: "" }, { target: "g2", name: "", from: "2026-01-01", until: "" }];
  assert.deepEqual(both.filter((x) => applies(x, "2026-11-16")).map((x) => x.target), ["g1", "g2"]);
  assert.equal(applies(h[0], "2026-10-01"), false);
  assert.equal(newer(3, 2), true);
  assert.equal(newer(2, 2), false);
  assert.equal(label("", "x"), "x");
  assert.equal(todayUTC(new Date("2026-11-16T23:30:00Z")), "2026-11-16");
  for (const ok of ["2026-11-16", "2024-02-29"]) assert.equal(validDay(ok), true, ok);
  for (const bad of ["2026-02-30", "2026-13-01", "2026-1-01", "", "x"]) assert.equal(validDay(bad), false, bad);
});

test("link IDs and the legacy history kind are checked before any read", () => {
  assert.deepEqual(linkedId(new URLSearchParams("member=bad id"), "member"), { id: "", invalid: true });
  assert.deepEqual(linkedId(new URLSearchParams("member=sample-acct-01"), "member"), { id: "sample-acct-01", invalid: false });
  assert.deepEqual(linkedId(new URLSearchParams("member="), "member"), { id: "", invalid: false });
  assert.deepEqual(linkedId(new URLSearchParams(""), "sel"), { id: "", invalid: false });
  assert.deepEqual(linkedId(new URLSearchParams("sel=" + "a".repeat(129)), "sel"), { id: "", invalid: true });
  assert.equal(legacyKind(new URLSearchParams("kind=payer")), "payer");
  assert.equal(legacyKind(new URLSearchParams("kind=payer&rk=view_group")), null);
  assert.equal(legacyKind(new URLSearchParams("kind=other")), null);
  assert.equal(legacyKind(new URLSearchParams("")), null);
});

test("a refused read claims a revision reset only when the same read succeeds without the pin", async () => {
  const invalid = new Error("invalid");
  const refused = (err: unknown) => err === invalid;
  const calls: (number | undefined)[] = [];
  const unitEntity = (pin: number | undefined) => { calls.push(pin); return Promise.reject(invalid); };
  await assert.rejects(pinnedRead(unitEntity, 1, refused), (e) => e instanceof FilterRefused);
  assert.deepEqual(calls, [1, undefined]);
  calls.length = 0;
  await assert.rejects(pinnedRead(unitEntity, null, refused), (e) => e instanceof FilterRefused);
  assert.deepEqual(calls, [undefined]);
  const stalePin = (pin: number | undefined) => (pin === undefined ? Promise.resolve("ok") : Promise.reject(invalid));
  await assert.rejects(pinnedRead(stalePin, 7, refused), (e) => e === invalid);
  const down = new Error("down");
  await assert.rejects(pinnedRead(() => Promise.reject(down), 1, refused), (e) => e === down);
  await assert.rejects(pinnedRead((pin) => Promise.reject(pin === undefined ? down : invalid), 1, refused), (e) => e === down);
  assert.equal(await pinnedRead((pin) => Promise.resolve(pin), 3, refused), 3);
  assert.equal(await pinnedRead((pin) => Promise.resolve(pin), null, refused), undefined);
});

test("a refused read drops a linked value only when one is applied", () => {
  assert.equal(refusal(new FilterRefused(), "sample-unit-north-ops"), "drop");
  assert.equal(refusal(new FilterRefused(), ""), "error");
  assert.equal(refusal(new Error("down"), "sample-le-north"), "store");
  assert.equal(refusal(new Error("down"), ""), "store");
});
