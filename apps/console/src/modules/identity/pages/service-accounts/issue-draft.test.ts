import { test } from "node:test";
import assert from "node:assert/strict";
import { admitsIssueDraft, issueDrafts } from "./issue-draft";
import { serviceLimits } from "../../../../service/service-query-fixtures";

const draft = {
  target: "svc-a",
  days: "1",
  reason: "Synthetic issuance draft",
};

test("issuance draft storage copies and freezes one target without replacing it on read", () => {
  const store = issueDrafts(() => true);
  const input = { ...draft, extra: "Unretained metadata" };
  assert.equal(store.guarded(), false);
  assert.equal(store.save(input, serviceLimits), true);
  const saved = store.get()!;
  input.days = "30";
  assert.equal(saved.days, "1");
  assert.equal(Object.isFrozen(saved), true);
  assert.deepEqual(Object.keys(saved), ["target", "days", "reason"]);
  assert.equal(Reflect.set(saved, "reason", "changed"), false);
  assert.equal(store.get(), saved);
  assert.equal(store.guarded(), true);
  assert.equal(store.save({ ...draft, target: "svc-b" }, serviceLimits), false);
  assert.equal(store.get(), saved);
});

test("an older completion or copied draft cannot clear a replacement even on the same target", () => {
  const store = issueDrafts(() => true);
  store.save(draft, serviceLimits);
  const first = store.get()!;
  store.save({ ...draft, reason: "Replacement draft" }, serviceLimits);
  const replacement = store.get()!;
  assert.equal(store.clear(first), false);
  assert.equal(store.clear({ ...replacement }), false);
  assert.equal(store.get(), replacement);
  assert.equal(store.clear(replacement), true);
  store.save({ ...draft, target: "svc-b" }, serviceLimits);
  assert.equal(store.clear(replacement), false);
  assert.equal(store.get()!.target, "svc-b");
});

test("current byte and escaped request bounds reject edits while retaining the previous draft", () => {
  const store = issueDrafts(() => true);
  store.save(draft, serviceLimits);
  const saved = store.get();
  const small = { ...draft, reason: "ok" };
  for (const candidate of [
    { ...small, target: "" },
    { ...small, target: "bad!" },
    { ...small, target: "x".repeat(129) },
    { ...small, days: "1234" },
    { ...small, reason: "한글값" },
  ]) {
    assert.equal(
      store.save(candidate, { ...serviceLimits, rationale_bytes: 8 }),
      false,
    );
    assert.equal(store.get(), saved);
  }
  assert.equal(
    admitsIssueDraft(
      { ...draft, days: "한글값" },
      { ...serviceLimits, command_value_bytes: "8" },
    ),
    false,
  );
  const bounded = { ...serviceLimits, body_bytes: 750 };
  assert.equal(
    admitsIssueDraft({ ...draft, reason: "a".repeat(200) }, bounded),
    true,
  );
  assert.equal(
    admitsIssueDraft({ ...draft, reason: "\\".repeat(200) }, bounded),
    false,
  );
});

test("freshly reduced limits do not rewrite a saved draft and must be revalidated", () => {
  const store = issueDrafts(() => true);
  store.save(draft, serviceLimits);
  const saved = store.get()!;
  assert.equal(
    admitsIssueDraft(saved, { ...serviceLimits, rationale_bytes: 8 }),
    false,
  );
  assert.equal(store.get(), saved);
  assert.equal(store.get()!.reason, draft.reason);
});

test("lease loss fences every access and disposal cannot resurrect an old draft", () => {
  let active = true;
  const store = issueDrafts(() => active);
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
  assert.equal(store.save(draft, serviceLimits), false);
  assert.equal(store.guarded(), false);
});
