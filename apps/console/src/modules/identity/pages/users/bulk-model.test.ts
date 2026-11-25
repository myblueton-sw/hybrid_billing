import { test } from "node:test";
import assert from "node:assert/strict";
import { jobOf, itemOf, type BulkJob } from "../../../../service/identity-bulk";
import { mayBeAdministrator, digestShown, expired, groups, headerState, invariant, itemWords, jobWords, notLoaded, number, selectLoaded, split, toggle, without } from "./bulk-model";

test("selection is kept by user ID and stops at the limit", () => {
  let s: ReadonlySet<string> = new Set();
  s = toggle(s, "u1", 2).next;
  s = toggle(s, "u2", 2).next;
  const refused = toggle(s, "u3", 2);
  assert.equal(refused.refused, true);
  assert.equal(refused.next, s);
  assert.deepEqual([...toggle(s, "u1", 2).next], ["u2"]);
  assert.deepEqual(notLoaded(s, ["u2", "u9"]), ["u1"]);
  assert.deepEqual([...without(s, ["u1"])], ["u2"]);
});

test("the header checkbox selects only the loaded rows and refuses past the limit", () => {
  const s = new Set(["u0"]);
  assert.equal(headerState(s, []), "none");
  assert.equal(headerState(s, ["u1", "u2"]), "none");
  assert.equal(headerState(new Set(["u1"]), ["u1", "u2"]), "mixed");
  const all = selectLoaded(s, ["u1", "u2"], 5);
  assert.deepEqual([...all.next].sort(), ["u0", "u1", "u2"]);
  assert.equal(headerState(all.next, ["u1", "u2"]), "checked");
  assert.deepEqual([...selectLoaded(all.next, ["u1", "u2"], 5).next], ["u0"]);
  const over = selectLoaded(s, ["u1", "u2"], 2);
  assert.equal(over.refused, true);
  assert.equal(over.next, s);
});

const job = (extra: Record<string, unknown>): BulkJob => jobOf({ id: "bulk-revocation-3", state: "executed", counts: { targets: 3, succeeded: 1, held: 1, failed: 0, unprocessed: 1 }, ...extra });

test("job words follow the server values and never claim more", () => {
  assert.equal(jobWords(job({ state: "draft" })), "draft");
  assert.equal(jobWords(job({ state: "prepared" })), "prepared");
  assert.equal(jobWords(job({ state: "discarded" })), "discarded");
  assert.equal(jobWords(job({ state: "nonsense" })), "expired");
  assert.equal(jobWords(job({ job_state: "running" })), "running");
  assert.equal(jobWords(job({ job_state: "" })), "running");
  assert.equal(jobWords(job({ job_state: "waiting", job_reason: "decision_unconfirmed" })), "unconfirmed");
  assert.equal(jobWords(job({ job_state: "waiting", job_reason: "cleanup_not_started" })), "not_started");
  assert.equal(jobWords(job({ job_state: "waiting", job_reason: "outcome_unknown" })), "waiting");
  assert.equal(jobWords(job({ job_state: "succeeded" })), "finished");
  assert.equal(jobWords(job({ job_state: "succeeded", counts: { targets: 1, failed: 1 } })), "partial");
  assert.equal(invariant(job({}).counts), true);
  assert.equal(invariant(job({ counts: { targets: 4, succeeded: 1 } }).counts), false);
});

test("item words and cleanup groups", () => {
  const words = ["succeeded", "held", "failed", "excluded"].map((o) => itemWords(itemOf({ outcome: o })));
  assert.deepEqual(words, ["complete", "held", "failed", "excluded"]);
  assert.equal(itemWords(itemOf({ outcome: "unprocessed", state: "received" })), "not_processed");
  assert.equal(itemWords(itemOf({ outcome: "unprocessed", state: "decision_unconfirmed", receipt: { reason: "ledger_full" } })), "unconfirmed");
  assert.equal(itemWords(itemOf({ outcome: "unprocessed", state: "decision_unconfirmed", receipt: { reason: "record_not_confirmed" } })), "not_recorded");
  assert.equal(itemWords(itemOf({ outcome: "odd" })), "not_processed");
  const g = groups([{ item: "B1", state: "done", reason: "" }, { item: "B6", state: "not_applicable", reason: "" }, { item: "B10", state: "outside_scope", reason: "" }]);
  assert.deepEqual([g.done.length, g.not_applicable.length, g.outside_scope.length], [1, 1, 1]);
});

test("lists split by state, expiry uses the server time, digest and number", () => {
  const s = split([job({ state: "draft" }), job({ state: "prepared" }), job({}), job({ state: "expired" }), job({ state: "discarded" })]);
  assert.deepEqual([s.open.length, s.executed.length, s.ended.length], [2, 1, 2]);
  assert.equal(expired(job({ state: "prepared", expires_at: "2026-11-18T09:30:00Z" }), "2026-11-18T09:30:00Z"), true);
  assert.equal(expired(job({ state: "prepared", expires_at: "2026-11-18T09:30:00Z" }), "2026-11-18T09:29:00Z"), false);
  assert.equal(expired(job({ state: "expired" }), ""), true);
  assert.equal(digestShown("0123456789abcdef"), "0123456789ab");
  assert.equal(number("bulk-revocation-12"), "12");
  assert.equal(number("bulk-draft-3"), "3");
  assert.equal(number("x"), "x");
});

test("a person whose role is unknown is listed with the administrators of a selection", () => {
  const row = (role: string | null) => ({ membership: role === null ? null : { role } }) as unknown as Parameters<typeof mayBeAdministrator>[0];
  assert.deepEqual(["administrator", "unknown", "member", null].map((r) => mayBeAdministrator(row(r))), [true, true, false, false]);
  assert.equal(mayBeAdministrator(undefined), false);
});
