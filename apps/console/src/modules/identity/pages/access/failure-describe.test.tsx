import { test } from "node:test";
import assert from "node:assert/strict";
import { AdminError } from "../../../../service/admin";
import { describeFor, describeRequest } from "./failure-model";
import { trustFailure } from "../authentication/trust-model";
import { a } from "./live-copy";
import { w } from "../users/live-copy";

const lost = () => [new AdminError("unavailable"), new Error("network"), new AdminError("unavailable", [], 0, "", "unsupported_command")];

test("commands whose answer was not confirmed say the change may have been saved", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    for (const e of lost()) {
      for (const shown of [describeFor("revoke")(l, e), describeFor("grant")(l, e), describeRequest(l, e), trustFailure(l, e), trustFailure(l, e, "approve")]) {
        assert.equal(shown.title, w.unknownTitle(l));
        assert.equal(shown.body, w.unknownBody(l));
      }
    }
    assert.equal(describeRequest(l, new AdminError("partial", [], 0, "op-1")).title, w.unknownTitle(l));
  }
});

test("reads keep their own unavailable text", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    const shown = describeFor("load")(l, new AdminError("unavailable"));
    assert.equal(shown.body, a.unavailableBody(l));
    assert.notEqual(shown.title, w.unknownTitle(l));
  }
});

test("definite answers keep their own text", () => {
  for (const l of ["en", "ko", "ar"] as const) {
    assert.equal(describeFor("revoke")(l, new AdminError("refused")).title, a.refusedTitle(l));
    assert.equal(describeFor("revoke")(l, new AdminError("busy")).body, a.unavailableBody(l));
    assert.notEqual(describeRequest(l, new AdminError("conflict")).title, w.unknownTitle(l));
  }
});
