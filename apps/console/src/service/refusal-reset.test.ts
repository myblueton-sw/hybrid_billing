import { test } from "node:test";
import assert from "node:assert/strict";
import { forget, pendingFor, refusedNow, setScope, settle } from "../modules/administration/pages/integrations/live-model";

test("a fresh grouped action clears only its own old refusal", () => {
  forget(); setScope("workspace", "user", "session");
  settle("integrations.plugin_stop", "plugin:sample", "hint");
  settle("integrations.plugin_disable", "plugin:sample", "hint");
  pendingFor("integrations.plugin_stop", "plugin:sample", { rationale: "retry" }, "r1", () => ({ request: "next-request", key: "next-key" }));
  assert.equal(refusedNow("integrations.plugin_stop", "plugin:sample"), false);
  assert.equal(refusedNow("integrations.plugin_disable", "plugin:sample"), true);
  forget();
});
