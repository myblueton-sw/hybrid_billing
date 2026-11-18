import { test } from "node:test";
import assert from "node:assert/strict";
import type { GroupDraft } from "../../../../service/admin";
import { afterRefusal, callRevision, conflictCopy, isLatest, partitionPrecheck, readOutcome, sameView, savePlan, addPending, allowed, allShown, block, capLeft, chunks, dirty, draftHref, editCalls, header, lastPlanNote, parseIds, planHref, refusalOf, settingValue, stateKey, toggled, validName } from "./drafts-model";

const plan = (state: string, blocked = "", rule = "") => ({ id: "plan-1", state, closed: "", blocked, rule });
type D = Pick<GroupDraft, "state" | "plan">;

test("a draft allows only the actions its state permits", () => {
  const cases: [D, string][] = [
    [{ state: "open", plan: null }, "edit,plan,discard"],
    [{ state: "open", plan: plan("rejected") }, "edit,plan,discard"],
    [{ state: "proposed", plan: plan("sealed") }, "openPlan"],
    [{ state: "proposed", plan: null }, ""],
    [{ state: "activated", plan: plan("activated") }, "openPlan"],
    [{ state: "discarded", plan: plan("discarded") }, ""],
    [{ state: "unknown", plan: null }, ""],
  ];
  for (const [d, want] of cases) {
    const a = allowed(d);
    assert.equal(Object.entries(a).filter(([, v]) => v).map(([k]) => k).join(","), want, JSON.stringify(d));
  }
});

test("the state block follows the derived state and the last plan", () => {
  assert.equal(block({ state: "open", plan: null }), null);
  assert.equal(block({ state: "open", plan: plan("rejected", "", "no_change") }), "rejected");
  assert.equal(block({ state: "open", plan: plan("discarded") }), "planDiscarded");
  assert.equal(block({ state: "proposed", plan: plan("sealed", "impact_not_computed") }), "held");
  assert.equal(block({ state: "proposed", plan: plan("sealed", "impact_unavailable") }), "held");
  assert.equal(block({ state: "proposed", plan: plan("sealed", "stale") }), "openPlan");
  assert.equal(block({ state: "proposed", plan: null }), "openPlan");
  assert.equal(block({ state: "activated", plan: plan("activated") }), "activated");
  assert.equal(block({ state: "discarded", plan: null }), "discarded");
  assert.equal(block({ state: "unknown", plan: null }), null);
  assert.equal(lastPlanNote({ state: "open", plan: plan("rejected") }), "rejected");
  assert.equal(lastPlanNote({ state: "open", plan: plan("discarded") }), "discarded");
  assert.equal(lastPlanNote({ state: "proposed", plan: plan("sealed") }), "");
});

test("members are saved in batches of at most 31", () => {
  const ids = Array.from({ length: 70 }, (_, i) => `a-${i}`);
  assert.deepEqual(chunks(ids).map((c) => c.length), [31, 31, 8]);
  assert.deepEqual(chunks([]), []);
  assert.deepEqual(chunks(ids.slice(0, 31)).map((c) => c.length), [31]);
});

test("edit calls keep the member count between 1 and the cap after every call", () => {
  const ids = (p: string, n: number) => Array.from({ length: n }, (_, i) => `${p}-${i}`);
  const check = (size: number, adds: number, removes: number, max: number) => {
    const calls = editCalls(size, ids("a", adds), ids("r", removes), max, false);
    assert.ok(calls, `${size} ${adds} ${removes} ${max}`);
    let n = size;
    for (const c of calls) {
      assert.ok(c.add.length <= 31 && c.remove.length <= 31);
      n += c.add.length - c.remove.length;
      assert.ok(n >= 1 && n <= max, `count ${n}`);
    }
    assert.equal(calls.flatMap((c) => c.add).length, adds);
    assert.equal(calls.flatMap((c) => c.remove).length, removes);
  };
  check(40, 0, 39, 1000);
  check(1, 60, 0, 1000);
  check(50, 40, 40, 50);
  check(1, 40, 1, 100);
  check(10, 31, 31, 10);
  assert.equal(editCalls(3, [], ["a", "b", "c"], 10, false), null);
  assert.equal(editCalls(10, ["x"], [], 10, false), null);
  assert.deepEqual(editCalls(2, [], [], 10, true), [{ add: [], remove: [], name: true }]);
  assert.deepEqual(editCalls(2, ["x"], [], 10, true), [{ add: ["x"], remove: [], name: true }]);
  assert.deepEqual(editCalls(2, [], [], 10, false), []);
});

test("pending adds skip duplicates and stop at the cap", () => {
  const got = addPending([{ id: "a", name: "A" }], [{ id: "a", name: "A" }, { id: "b", name: "" }, { id: "c", name: "" }, { id: "d", name: "" }], new Set(["d"]), 3);
  assert.deepEqual(got.list.map((m) => m.id), ["a", "b"]);
  assert.deepEqual([got.added, got.already, got.over], [1, 2, 1]);
});

test("entered IDs are split, deduplicated and checked", () => {
  assert.deepEqual(parseIds(" acct-1\nacct-2  acct-1\n\nbad id!\n"), { ids: ["acct-1", "acct-2", "bad"], bad: ["id!"] });
  assert.deepEqual(parseIds("acct-1\nacct 2\nx/y\nx/y"), { ids: ["acct-1", "acct", "2"], bad: ["x/y"] });
  assert.deepEqual(parseIds(""), { ids: [], bad: [] });
});

test("a draft name is 1 to 80 characters without control characters", () => {
  assert.equal(validName("  Ops  "), true);
  assert.equal(validName("가".repeat(80)), true);
  assert.equal(validName("가".repeat(81)), false);
  assert.equal(validName("   "), false);
  assert.equal(validName("a\tb"), false);
  assert.equal(validName("a\u0085b"), false);
});

test("the form is dirty only when a name, group or pending member differs from the start", () => {
  const start = { name: "Ops", mode: "new" as const, group: "", adds: [], removes: [] };
  assert.equal(dirty(start, { ...start }), false);
  assert.equal(dirty(start, { ...start, name: " Ops " }), false);
  assert.equal(dirty(start, { ...start, name: "Ops 2" }), true);
  assert.equal(dirty(start, { ...start, mode: "existing" }), true);
  assert.equal(dirty(start, { ...start, group: "g" }), true);
  assert.equal(dirty(start, { ...start, adds: ["a"] }), true);
  assert.equal(dirty(start, { ...start, removes: ["a"] }), true);
});

test("page selection is bounded by the cap and the header reflects it", () => {
  let s = new Set<string>();
  s = toggled(s, "a", 2);
  s = toggled(s, "b", 2);
  s = toggled(s, "c", 2);
  assert.deepEqual([...s], ["a", "b"]);
  assert.deepEqual([...toggled(s, "a", 2)], ["b"]);
  assert.deepEqual([...allShown(new Set(), ["a", "b", "c"], 2)], ["a", "b"]);
  assert.deepEqual([...allShown(new Set(["a", "b"]), ["a", "b"], 5)], []);
  assert.equal(header(new Set(), ["a"]), "none");
  assert.equal(header(new Set(["a"]), ["a", "b"]), "some");
  assert.equal(header(new Set(["a", "b"]), ["a", "b"]), "all");
  assert.equal(capLeft(10, 4), 6);
  assert.equal(capLeft(3, 4), 0);
});

test("settings, links and refusals", () => {
  const settings = [{ name: "view_group_draft_members", value: 25 }, { name: "bad", value: 0 }];
  assert.equal(settingValue(settings, "view_group_draft_members", 1000), 25);
  assert.equal(settingValue(settings, "bad", 7), 7);
  assert.equal(settingValue([], "x", 9), 9);
  assert.equal(draftHref("vgd-1", "2026-12-01", "2026-11-18", 3, "ko"), "#/organization-changes?view=targets&draft=vgd-1&from=2026-12-01&base=3&lang=ko");
  assert.equal(draftHref("vgd-1", "2026-11-01", "2026-11-18", 0, ""), "#/organization-changes?view=targets&draft=vgd-1&base=0");
  assert.equal(planHref("plan-1", "ar"), "#/organization-changes?view=impact&plan=plan-1&lang=ar");
  assert.equal(planHref("plan-1", ""), "#/organization-changes?view=impact&plan=plan-1");
  for (const [failure, code, want] of [["invalid", "", "field"], ["refused", "", "limit"], ["stage", "", "stage"], ["denied", "permission_denied", "denied"], ["denied", "", "failed"],
    ["unauthenticated", "", "signin"], ["busy", "", "dependency"], ["partial", "", "failed"], ["unavailable", "unsupported_command", "missing"], ["conflict", "revision_conflict", "conflict"]]) assert.equal(refusalOf(failure, code), want, failure);
  assert.deepEqual((["open", "proposed", "activated", "discarded", "unknown"] as const).map(stateKey), ["stOpen", "stProposed", "stActivated", "stDiscarded", "stUnknown"]);
});

test("the pre-check partition drops members already saved and IDs that are not source accounts", () => {
  const m = (id: string) => ({ id, name: "" });
  const got = partitionPrecheck([m("a"), m("b"), m("x"), m("c")], new Set(["a"]), new Set(["b", "c", "a"]));
  assert.deepEqual(got.keep.map((x) => x.id), ["b", "c"]);
  assert.equal(got.already, 1);
  assert.deepEqual(got.strangers, ["x"]);
  assert.deepEqual(partitionPrecheck([], new Set(), new Set()), { keep: [], already: 0, strangers: [] });
});

test("a pending partial save continues the created draft and never starts a new one", () => {
  assert.deepEqual(savePlan({ kind: "new", draftId: "vgd-1", done: 31 }, "new", ""), { mode: "continue", draftId: "vgd-1", done: 31 });
  assert.deepEqual(savePlan({ kind: "new", draftId: "vgd-1", done: 31 }, "draft", "vgd-1"), { mode: "continue", draftId: "vgd-1", done: 31 });
  assert.deepEqual(savePlan(null, "new", ""), { mode: "new" });
  assert.deepEqual(savePlan({ kind: "edit", draftId: "vgd-2", done: 3 }, "draft", "vgd-2"), { mode: "edit", draftId: "vgd-2" });
  assert.deepEqual(savePlan(null, "draft", "vgd-3"), { mode: "edit", draftId: "vgd-3" });
  assert.deepEqual(savePlan(null, "gone", ""), { mode: "none" });
  assert.deepEqual(savePlan({ kind: "new", draftId: "", done: 0 }, "new", ""), { mode: "new" });
});

test("late member pages and finished saves only apply to the view they started in", () => {
  assert.equal(isLatest(3, 3), true);
  assert.equal(isLatest(2, 3), false);
  assert.equal(sameView(5, 5), true);
  assert.equal(sameView(5, 6), false);
});

test("only the first call of a fresh save uses the loaded revision", () => {
  assert.equal(callRevision(0, false, "r7"), "r7");
  assert.equal(callRevision(1, false, "r7"), null);
  assert.equal(callRevision(0, true, "r7"), null);
  assert.equal(callRevision(0, false, ""), null);
});

test("a refused command shows 'gone' only when the drafts read succeeds without the draft", () => {
  assert.deepEqual(afterRefusal("missing", "denied", false), { view: "gone" });
  assert.deepEqual(afterRefusal("denied", "denied", false), { view: "denied" });
  assert.deepEqual(afterRefusal("signin", "stage", false), { view: "signin" });
  assert.deepEqual(afterRefusal("failed", "conflict", false), { view: "failed" });
  assert.deepEqual(afterRefusal("found", "conflict", true), { view: "reloaded", notice: "dataChanged" });
  assert.deepEqual(afterRefusal("found", "stage", true), { view: "reloaded", notice: "draftChanged" });
  assert.deepEqual(afterRefusal("found", "conflict", false), { view: "reloaded", notice: "draftChangedClosed" });
  assert.equal(readOutcome("denied"), "denied");
  assert.equal(readOutcome("unauthenticated"), "signin");
  assert.equal(readOutcome("busy"), "failed");
});

test("a conflict before any draft exists says nothing was saved", () => {
  assert.equal(conflictCopy(false), "dataChangedNew");
  assert.equal(conflictCopy(true), "dataChanged");
});
