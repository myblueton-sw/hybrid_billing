import { test } from "node:test";
import assert from "node:assert/strict";
import { previewLogin, requireLocalPreviewHost } from "./local-preview-config";
import { localPreviewLogin } from "./src/service/local-preview";
const env = { HB_LOCAL_PREVIEW_WORKSPACE: " sample ", HB_LOCAL_PREVIEW_EMAIL: " sample@example.invalid ", HB_LOCAL_PREVIEW_PASSWORD: " test bytes " };

test("sample injection requires development serve and complete explicit configuration", () => {
  assert.equal(previewLogin("build", false, env), null);
  assert.equal(previewLogin("serve", true, env), null);
  for (const key of Object.keys(env)) assert.equal(previewLogin("serve", false, { ...env, [key]: "" }), null);
  assert.deepEqual(previewLogin("serve", false, env), { workspace: "sample", email: "sample@example.invalid", password: " test bytes " });
});

test("sample server and UI require exact loopback hosts", () => {
  const login = previewLogin("serve", false, env);
  for (const host of ["127.0.0.1", "localhost", "::1", "[::1]"]) {
    assert.doesNotThrow(() => requireLocalPreviewHost(host, login));
    assert.deepEqual(localPreviewLogin(host, login), login);
  }
  for (const host of ["0.0.0.0", "::", "localhost.example.com", "192.168.0.1", "", true, undefined]) assert.throws(() => requireLocalPreviewHost(host, login));
  assert.equal(localPreviewLogin("example.com", login), null);
  assert.equal(localPreviewLogin("localhost", null), null);
  assert.doesNotThrow(() => requireLocalPreviewHost(true, null));
});
