import { test } from "node:test";
import assert from "node:assert/strict";
import { focusHref, focusQuery } from "./query";
test("overview to source and back preserve complete pinned selection but reset offsets", () => {
  const params = new URLSearchParams({
    lang: "ko",
    "focus-period": "2024-09",
    "focus-provider": "AWS",
    "focus-account": "a".repeat(64),
    "focus-basis": "effective_cost",
    "focus-q": "AWS",
    "focus-snapshot": "b".repeat(64),
    "focus-after": "25",
    "focus-account_after": "25",
    view: "settlement",
    scope: "sample-b",
  });
  const href = focusHref("/source-data", params),
    target = new URLSearchParams(href.split("?")[1]);
  assert.equal(href.startsWith("#/source-data?"), true);
  assert.equal(target.has("focus-after"), false);
  assert.equal(target.has("focus-account_after"), false);
  assert.equal(target.has("scope"), false);
  const back = new URLSearchParams(focusHref("/home", target).split("?")[1]);
  assert.deepEqual([...focusQuery(back)], [...focusQuery(target)]);
  for (const k of ["period", "provider", "account", "basis", "q", "snapshot"])
    assert.equal(back.get("focus-" + k), params.get("focus-" + k));
});
test("query normalizes search and never accepts actor authority", () => {
  const q = focusQuery(
    new URLSearchParams({
      "focus-q": " AWS ",
      actor: "foreign",
      "focus-actor": "foreign",
    }),
  );
  assert.equal(q.get("q"), "AWS");
  assert.equal(q.has("actor"), false);
});

test("direct and post-filter source return pins the validated loaded snapshot", () => {
  for (const params of [
    new URLSearchParams({ lang: "ko" }),
    new URLSearchParams({
      "focus-provider": "AWS",
      "focus-q": "AWS",
      "focus-basis": "effective_cost",
      "focus-after": "25",
    }),
  ]) {
    const href = focusHref("/home", params, "c".repeat(64)),
      q = new URLSearchParams(href.split("?")[1]);
    assert.equal(q.get("focus-snapshot"), "c".repeat(64));
    assert.equal(q.has("focus-after"), false);
    assert.equal(q.get("focus-q"), params.get("focus-q"));
  }
  const prior = new URLSearchParams({ "focus-snapshot": "a".repeat(64) });
  assert.equal(
    new URLSearchParams(
      focusHref("/home", prior, "b".repeat(64)).split("?")[1],
    ).get("focus-snapshot"),
    "b".repeat(64),
  );
});
