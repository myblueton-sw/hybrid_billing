import { test } from "node:test";
import assert from "node:assert/strict";
import * as nodeModule from "node:module";

const assetStub = /\.(css|svg)(\?.*)?$/;
if (typeof nodeModule.registerHooks === "function") nodeModule.registerHooks({ load: (url, context, next) => (assetStub.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context)) });
else nodeModule.register("data:text/javascript," + encodeURIComponent(`export async function load(url, context, next) { return ${assetStub}.test(url) ? { format: "module", source: "export default ''", shortCircuit: true } : next(url, context); }`));
const { changeProps } = await import("./work");

test("Organization changes receives the shell's unsaved-input reporter", () => {
  const onDirty = (_: boolean) => undefined;
  const signOut = () => undefined;
  const params = new URLSearchParams("view=targets");
  const session = { user: "u", workspace: "w", role: "administrator", login: "a@example.invalid", csrf: "t" };
  const got = changeProps({ screenId: "O02", locale: "ko", params, view: "targets", onState: () => undefined, onDirty }, { session, signOut });
  assert.equal(got.onDirty, onDirty);
  assert.equal(got.onSignOut, signOut);
  assert.equal(got.session, session);
  assert.equal(got.params, params);
  assert.equal(got.view, "targets");
  assert.equal(got.locale, "ko");
});

test("Organizations receives the same unsaved-input reporter", () => {
  const onDirty = (_: boolean) => undefined;
  const session = { user: "u", workspace: "w", role: "administrator", login: "a@example.invalid", csrf: "t" };
  const got = changeProps({ screenId: "O01", locale: "ar", params: new URLSearchParams("view=accounts"), view: "accounts", onState: () => undefined, onDirty }, { session, signOut: () => undefined });
  assert.equal(got.onDirty, onDirty);
  assert.equal(got.view, "accounts");
});
