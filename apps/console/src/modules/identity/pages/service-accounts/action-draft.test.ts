import { test } from "node:test";
import assert from "node:assert/strict";
import {
  actionDrafts,
  admitsActionDraft,
  type DraftAction,
} from "./action-draft";
import { serviceLimits } from "../../../../service/service-query-fixtures";

const draft = {
  action: "suspend" as const,
  target: "svc-a",
  reason: "Synthetic action draft",
};
test("action drafts copy exactly three frozen fields and reject incompatible targets or actions", () => {
  const store = actionDrafts(() => true);
  const input = { ...draft, extra: "Not retained" };
  assert.equal(store.save(input, serviceLimits), true);
  const saved = store.get()!;
  input.reason = "Changed caller value";
  assert.equal(saved.reason, draft.reason);
  assert.deepEqual(Object.keys(saved), ["action", "target", "reason"]);
  assert.equal(Object.isFrozen(saved), true);
  assert.equal(store.guarded(), true);
  for (const other of [
    { ...draft, target: "svc-b" },
    { ...draft, action: "retire" as const },
  ]) {
    assert.equal(store.save(other, serviceLimits), false);
    assert.equal(store.get(), saved);
  }
});
test("stale action completions cannot clear a new same-target edit or another action", () => {
  const store = actionDrafts(() => true);
  store.save(draft, serviceLimits);
  const first = store.get()!;
  store.save({ ...draft, reason: "Replacement edit" }, serviceLimits);
  const replacement = store.get()!;
  assert.equal(store.clear(first), false);
  assert.equal(store.clear({ ...replacement }), false);
  assert.equal(store.get(), replacement);
  assert.equal(store.clear(replacement), true);
  store.save({ ...draft, action: "retire" }, serviceLimits);
  assert.equal(store.clear(first), false);
  assert.equal(store.get()!.action, "retire");
});
test("only finite canonical action drafts within effective byte and serialized bounds are retained", () => {
  const store = actionDrafts(() => true);
  store.save(draft, serviceLimits);
  const saved = store.get();
  for (const candidate of [
    { ...draft, action: "issue" as DraftAction },
    { ...draft, target: "" },
    { ...draft, target: "invalid!" },
    { ...draft, target: "a".repeat(129) },
    { ...draft, reason: "한글값" },
  ]) {
    assert.equal(
      store.save(candidate, { ...serviceLimits, rationale_bytes: 8 }),
      false,
    );
    assert.equal(store.get(), saved);
  }
  const smallBody = { ...serviceLimits, body_bytes: 750 };
  assert.equal(
    admitsActionDraft({ ...draft, reason: "a".repeat(200) }, smallBody),
    true,
  );
  assert.equal(
    admitsActionDraft({ ...draft, reason: "\\".repeat(200) }, smallBody),
    false,
  );
  for (const action of ["suspend", "reinstate", "retire", "revoke"] as const)
    assert.equal(admitsActionDraft({ ...draft, action }, serviceLimits), true);
});
test("current limit reductions reject a new action submission without rewriting the retained edit", () => {
  const store = actionDrafts(() => true);
  store.save(draft, serviceLimits);
  const saved = store.get()!;
  assert.equal(
    admitsActionDraft(saved, { ...serviceLimits, rationale_bytes: 8 }),
    false,
  );
  assert.equal(store.get(), saved);
});
test("scope loss and disposal fence action draft callbacks and unload guards", () => {
  let active = true;
  const store = actionDrafts(() => active);
  store.save(draft, serviceLimits);
  const saved = store.get()!;
  active = false;
  assert.equal(store.get(), undefined);
  assert.equal(store.guarded(), false);
  assert.equal(store.save(draft, serviceLimits), false);
  assert.equal(store.clear(saved), false);
  store.dispose();
  active = true;
  assert.equal(store.get(), undefined);
  assert.equal(store.guarded(), false);
  assert.equal(store.save(draft, serviceLimits), false);
});
