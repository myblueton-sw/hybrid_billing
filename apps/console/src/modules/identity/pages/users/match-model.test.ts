import { test } from "node:test";
import assert from "node:assert/strict";
import { jobOf, matchCountOf, type BulkPage } from "../../../../service/identity-bulk";
import { noFilters } from "./filter-model";
import { confirmable, countNote, countView, differsFromCount, emptyFilter, fixedOfFields, fixedOfJob, keepsOwn, refusalCause, storedFilters } from "./match-model";

const count = (o: Record<string, unknown>) => matchCountOf({ count: 0, complete: false, over_limit: false, limit: 5000, scan_budget: 20000, ...o });

test("the count answer decides the label and whether selecting is offered", () => {
  assert.equal(countView(count({ count: 7, complete: true })), "exact");
  assert.equal(countView(count({ count: 7 })), "atLeast");
  assert.equal(countView(count({ count: 0, complete: true })), "none");
  assert.equal(countView(count({ count: 0 })), "atLeast");
  assert.equal(countView(count({ count: 5001, over_limit: true })), "over");
  assert.equal(countView(count({ count: 5001, over_limit: true, complete: true })), "over");
  assert.deepEqual([count({ count: 1, complete: true }), count({ count: 3 }), count({ count: 0, complete: true }), count({ over_limit: true })].map(confirmable), [true, true, false, false]);
});

const job = (o: Record<string, unknown>) => jobOf({ id: "bulk-draft-4", state: "draft", selection: "filter", preparer: "me", matched: 12, fixed_at: "2026-11-19T10:00:00Z",
  fixed_revision: "r9", filter: { role: "member", q: "ann" }, ...o });

test("an open filter draft of the caller restores the fixed list, nothing else does", () => {
  const f = fixedOfJob(job({}), "me", 11);
  assert.deepEqual(f, { draft: "bulk-draft-4", matched: 12, fixed_at: "2026-11-19T10:00:00Z", fixed_revision: "r9", counted: 11,
    filter: { q: "ann", status: "", membership: "", role: "member", period: "", signin: "", invitation: "", holder: "" } });
  for (const other of [job({ state: "prepared" }), job({ selection: "explicit" }), job({ preparer: "else" }), job({ filter: null })]) assert.equal(fixedOfJob(other, "me"), null);
});

test("a fixed list is read from the command answer", () => {
  const f = fixedOfFields({ draft: "bulk-draft-2", matched: "40", fixed_at: "2026-11-19T10:00:00Z", fixed_revision: "r3", "filter.role": "member", "filter.holder": "true" }, 39);
  assert.equal(f.matched, 40);
  assert.equal(f.filter.role, "member");
  assert.equal(f.filter.holder, "true");
  assert.equal(f.filter.q, "");
  assert.ok(differsFromCount(f));
  assert.ok(!differsFromCount({ ...f, counted: 40 }));
  assert.ok(!differsFromCount({ ...f, counted: null }));
  assert.deepEqual(fixedOfFields({}, null), { draft: "", matched: 0, fixed_at: "", fixed_revision: "", counted: null,
    filter: { q: "", status: "", membership: "", role: "", period: "", signin: "", invitation: "", holder: "" } });
});

test("stored terms map to the link filters, and a change of filters is told apart", () => {
  const f = fixedOfFields({ "filter.q": "ann", "filter.status": "active", "filter.membership": "none", "filter.role": "member", "filter.period": "open", "filter.signin": "not_set", "filter.holder": "true" }, null);
  const link = storedFilters(f.filter);
  assert.deepEqual(link, { q: "ann", st: "active", ms: "none", role: "member", per: "open", si: "not_set", adm: true });
  assert.ok(!keepsOwn(f, link));
  assert.ok(keepsOwn(f, { ...link, role: "administrator" }));
  assert.ok(keepsOwn(f, noFilters));
  assert.ok(emptyFilter(fixedOfFields({}, null).filter));
  assert.ok(!emptyFilter(f.filter));
});

test("a refused fix is named only when a read confirms the cause", () => {
  const page = (o: Partial<BulkPage>) => ({ open_draft: false, pending: 0, ...o }) as BulkPage;
  assert.equal(refusalCause(page({ open_draft: true }), 20, count({ count: 0, complete: true })), "openDraft");
  assert.equal(refusalCause(page({}), 20, count({ count: 0, complete: true })), "none");
  assert.equal(refusalCause(page({}), 20, count({ over_limit: true })), "over");
  assert.equal(refusalCause(page({ pending: 20 }), 20, count({ count: 4 })), "pending");
  assert.equal(refusalCause(page({ pending: 19 }), 20, count({ count: 4 })), "unknown");
  assert.equal(refusalCause(null, 20, null), "unknown");
  assert.equal(refusalCause(page({ pending: 25 }), 20, null), "pending");
  assert.equal(refusalCause(null, 20, count({ count: 0 })), "unknown");
});

test("a fixed number below the counted lower bound is explained, a larger one is not a difference", () => {
  const below = fixedOfFields({ matched: "35" }, 40, true);
  assert.equal(countNote(below), "below");
  assert.equal(countNote({ ...below, matched: 40 }), "none");
  assert.equal(countNote({ ...below, matched: 41 }), "none");
  assert.equal(countNote(fixedOfFields({ matched: "35" }, 40, false)), "differs");
  assert.equal(countNote(fixedOfFields({ matched: "40" }, 40, false)), "none");
  assert.equal(countNote(fixedOfFields({ matched: "40" }, null, false)), "none");
});
