import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import assert from "node:assert/strict";
import { createServer } from "vite";
const args = Object.fromEntries(process.argv.slice(2).reduce((pairs, value, index, all) => value.startsWith("--") ? [...pairs, [value.slice(2), all[index + 1]]] : pairs, []));
const require = createRequire(import.meta.url);
const { chromium } = require(args["playwright-module"] || "playwright");
const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const server = await createServer({ root, configFile: false, envFile: false,
  define: { __SOURCE_REVISION__: JSON.stringify("HYB-304-isolated-recovery-fixture"), __LOCAL_PREVIEW_LOGIN__: "null" },
  ...(args["cache-dir"] ? { cacheDir: args["cache-dir"] } : {}), server: { host: "127.0.0.1", port: 0, fs: { allow: [resolve(root, "../..")] } }, plugins: [{ name: "recovery-test-fixture", configureServer(instance) {
  instance.middlewares.use("/__recovery_test", async (_request, response, next) => {
    try {
      response.setHeader("Content-Type", "text/html");
      response.end(await instance.transformIndexHtml("/__recovery_test", '<!doctype html><html lang="en"><body><div id="root"></div><script type="module" src="/tools/recovery-fixture.tsx"></script></body></html>'));
    } catch (error) { next(error); }
  });
} }] });
await server.listen();
const address = server.httpServer.address();
if (typeof address === "string" || !address) throw new Error("fixture server address unavailable");
let browser;
const results = [];
const errors = [];
async function scenario(name, action) {
  if (args.only && !name.includes(args.only)) return;
  const page = await browser.newPage();
  page.on("pageerror", (error) => errors.push(error.message));
  try {
    await page.goto(`http://127.0.0.1:${address.port}/__recovery_test`);
    await page.waitForFunction(() => window.hbRecovery?.snapshot().ready);
    await action(page);
    results.push({ name, status: "PASS" });
  } catch (error) { results.push({ name, status: "FAIL", error: String(error.message) }); }
  finally { await page.close(); }
}
try {
  browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
  await scenario("unsubmitted action draft survives same-lease locale and view remount", async (page) => {
    assert.equal(await page.evaluate(() => window.hbRecovery.saveAction()), true);
    assert.equal(await page.evaluate(() => window.hbRecovery.unloadGuarded()), true);
    const id = await page.evaluate(() => window.hbRecovery.snapshot().id);
    await page.evaluate(() => { window.hbRecovery.locale("ko"); window.hbRecovery.unmountView(); });
    await page.waitForFunction(() => !window.hbRecovery.snapshot().ready);
    await page.evaluate(() => window.hbRecovery.mountView());
    await page.waitForFunction((id) => window.hbRecovery.snapshot().ready && window.hbRecovery.snapshot().id === id, id);
    const value = await page.evaluate(() => window.hbRecovery.snapshot().actionDraft);
    assert.deepEqual(value, { action: "suspend", target: "svc-test", reason: "Synthetic action draft" });
  });
  await scenario("unsubmitted action draft cannot cross any replacement lease or authentication loss", async (page) => {
    for (const field of ["user", "workspace", "csrf", "client"]) {
      assert.equal(await page.evaluate(() => window.hbRecovery.saveAction()), true);
      const id = await page.evaluate(() => window.hbRecovery.snapshot().id);
      await page.evaluate((field) => window.hbRecovery.switchScope(field), field);
      await page.waitForFunction((id) => window.hbRecovery.snapshot().ready && window.hbRecovery.snapshot().id !== id, id);
      const value = await page.evaluate(() => window.hbRecovery.snapshot());
      assert.equal(value.actionDraft, null);
      assert.equal(value.priorActionDraft, null);
      assert.equal(value.priorActionGuard, false);
      assert.equal(await page.evaluate(() => window.hbRecovery.savePriorAction()), false);
      assert.equal(await page.evaluate(() => window.hbRecovery.unloadGuarded()), false);
    }
    assert.equal(await page.evaluate(() => window.hbRecovery.saveAction()), true);
    await page.evaluate(() => window.hbRecovery.rejectAuthentication());
    await page.waitForFunction(() => !window.hbRecovery.snapshot().ready);
    const value = await page.evaluate(() => window.hbRecovery.snapshot());
    assert.equal(value.priorActionDraft, null);
    assert.equal(value.priorActionGuard, false);
    assert.equal(await page.evaluate(() => window.hbRecovery.savePriorAction()), false);
    assert.equal(await page.evaluate(() => window.hbRecovery.unloadGuarded()), false);
  });
  await scenario("unsubmitted issuance draft survives same-lease locale and view remount", async (page) => {
    assert.equal(await page.evaluate(() => window.hbRecovery.saveDraft()), true);
    assert.equal(await page.evaluate(() => window.hbRecovery.unloadGuarded()), true);
    const id = await page.evaluate(() => window.hbRecovery.snapshot().id);
    await page.evaluate(() => { window.hbRecovery.locale("ko"); window.hbRecovery.unmountView(); });
    await page.waitForFunction(() => !window.hbRecovery.snapshot().ready);
    await page.evaluate(() => window.hbRecovery.mountView());
    await page.waitForFunction((id) => window.hbRecovery.snapshot().ready && window.hbRecovery.snapshot().id === id, id);
    const value = await page.evaluate(() => window.hbRecovery.snapshot().draft);
    assert.deepEqual(value, { target: "svc-test", days: "1", reason: "Synthetic issuance draft" });
  });
  await scenario("unsubmitted issuance draft cannot cross any replacement lease or authentication loss", async (page) => {
    for (const field of ["user", "workspace", "csrf", "client"]) {
      assert.equal(await page.evaluate(() => window.hbRecovery.saveDraft()), true);
      const id = await page.evaluate(() => window.hbRecovery.snapshot().id);
      await page.evaluate((field) => window.hbRecovery.switchScope(field), field);
      await page.waitForFunction((id) => window.hbRecovery.snapshot().ready && window.hbRecovery.snapshot().id !== id, id);
      const value = await page.evaluate(() => window.hbRecovery.snapshot());
      assert.equal(value.draft, null);
      assert.equal(value.priorDraft, null);
      assert.equal(value.priorDraftGuard, false);
      assert.equal(await page.evaluate(() => window.hbRecovery.savePriorDraft()), false);
      assert.equal(await page.evaluate(() => window.hbRecovery.unloadGuarded()), false);
    }
    assert.equal(await page.evaluate(() => window.hbRecovery.saveDraft()), true);
    await page.evaluate(() => window.hbRecovery.rejectAuthentication());
    await page.waitForFunction(() => !window.hbRecovery.snapshot().ready);
    const value = await page.evaluate(() => window.hbRecovery.snapshot());
    assert.equal(value.priorDraft, null);
    assert.equal(value.priorDraftGuard, false);
    assert.equal(await page.evaluate(() => window.hbRecovery.savePriorDraft()), false);
    assert.equal(await page.evaluate(() => window.hbRecovery.unloadGuarded()), false);
  });
  await scenario("same-scope locale and view remount retain recovery", async (page) => {
    await page.evaluate(() => window.hbRecovery.issue());
    const before = await page.evaluate(() => window.hbRecovery.snapshot());
    await page.evaluate(() => window.hbRecovery.unmountView());
    await page.waitForSelector('[data-fixture="ready"]', { state: "detached" });
    await page.evaluate(() => { window.hbRecovery.locale("ar"); window.hbRecovery.mountView(); });
    await page.waitForFunction(() => window.hbRecovery.snapshot().ready);
    const after = await page.evaluate(() => window.hbRecovery.snapshot());
    assert.equal(after.id, before.id); assert.equal(after.phase, "issued"); assert.equal(after.calls, 0);
  });
  for (const field of ["user", "workspace", "csrf", "client"]) await scenario(`${field} change retires old scope`, async (page) => {
    await page.evaluate(() => window.hbRecovery.issue());
    const before = await page.evaluate(() => window.hbRecovery.snapshot().id);
    await page.evaluate((field) => window.hbRecovery.switchScope(field), field);
    await page.waitForFunction((id) => { const s = window.hbRecovery.snapshot(); return s.ready && s.id !== id; }, before);
    const after = await page.evaluate(() => window.hbRecovery.snapshot());
    assert.equal(after.priorDisposed, true); assert.equal(after.phase, ""); assert.equal(after.gapSafe, true); assert.equal(after.layoutGapSafe, true);
  });
  await scenario("Reveal StrictMode uses one pending GET", async (page) => {
    await page.evaluate(async () => { await window.hbRecovery.issue(); window.hbRecovery.show(); });
    await page.waitForFunction(() => window.hbRecovery.snapshot().calls === 1);
    await page.evaluate(() => window.hbRecovery.resolve());
    await page.waitForSelector("#svc-secret");
    const snapshot = await page.evaluate(() => window.hbRecovery.snapshot());
    assert.equal(snapshot.calls, 1); assert.equal(snapshot.markers.length, 1); assert.equal(snapshot.phase, "");
  });
  await scenario("hidden mount does not consume", async (page) => {
    await page.evaluate(async () => { await window.hbRecovery.issue(); window.hbRecovery.visibility("hidden"); window.hbRecovery.show(); });
    await page.waitForSelector("#svc-reveal-title");
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    const snapshot = await page.evaluate(() => window.hbRecovery.snapshot());
    assert.equal(snapshot.calls, 0); assert.equal(snapshot.phase, "issued");
  });
  await scenario("hidden deferred boundary does not consume", async (page) => {
    await page.evaluate(async () => { await window.hbRecovery.issue(); window.hbRecovery.hideBeforeDispatch(); window.hbRecovery.show(); });
    await page.waitForSelector("#svc-reveal-title");
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    const snapshot = await page.evaluate(() => window.hbRecovery.snapshot());
    assert.equal(snapshot.calls, 0); assert.equal(snapshot.phase, "issued");
  });
  await scenario("pagehide invalidates observer before pending response", async (page) => {
    await page.evaluate(async () => { await window.hbRecovery.issue(); window.hbRecovery.show(); });
    await page.waitForFunction(() => window.hbRecovery.snapshot().calls === 1);
    await page.evaluate(() => { window.hbRecovery.pagehide(); window.hbRecovery.visibility("visible"); window.hbRecovery.resolve(); });
    await page.waitForFunction(() => window.hbRecovery.snapshot().markers.length === 1);
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    assert.equal(await page.locator("#svc-secret").count(), 0);
    assert.equal((await page.evaluate(() => window.hbRecovery.snapshot())).calls, 1);
  });
  await scenario("scope loss drops pending secret", async (page) => {
    await page.evaluate(async () => { await window.hbRecovery.issue(); window.hbRecovery.show(); });
    await page.waitForFunction(() => window.hbRecovery.snapshot().calls === 1);
    await page.evaluate(() => { window.hbRecovery.switchScope("csrf"); window.hbRecovery.resolve(); });
    await page.waitForFunction(() => window.hbRecovery.snapshot().ready && window.hbRecovery.snapshot().priorDisposed === true);
    assert.equal(await page.locator("#svc-secret").count(), 0);
    assert.deepEqual((await page.evaluate(() => window.hbRecovery.snapshot())).markers, []);
  });
  await scenario("authenticated owner unmount disposes while view is absent", async (page) => {
    await page.evaluate(async () => { await window.hbRecovery.issue(); window.hbRecovery.unmountView(); });
    await page.waitForSelector('[data-fixture="ready"]', { state: "detached" });
    await page.evaluate(() => window.hbRecovery.unmountRoot());
    assert.equal((await page.evaluate(() => window.hbRecovery.snapshot())).priorDisposed, true);
  });
  for (const alreadyLost of [false, true]) await scenario(`authentication loss ${alreadyLost ? "before subscription" : "while view absent"} retires scope`, async (page) => {
    await page.evaluate(async () => { await window.hbRecovery.issue(); window.hbRecovery.unmountView(); });
    await page.waitForSelector('[data-fixture="ready"]', { state: "detached" });
    await page.evaluate((alreadyLost) => window.hbRecovery.rejectAuthentication(alreadyLost), alreadyLost);
    await page.evaluate(() => window.hbRecovery.mountView());
    await page.waitForSelector('[data-fixture="waiting"]');
    const snapshot = await page.evaluate(() => window.hbRecovery.snapshot());
    assert.equal(snapshot.ready, false); assert.equal(snapshot.priorDisposed, true);
  });
  await scenario("unknown collection remount requires explicit retry", async (page) => {
    await page.evaluate(async () => { await window.hbRecovery.issue(); window.hbRecovery.show(); });
    await page.waitForFunction(() => window.hbRecovery.snapshot().calls === 1);
    await page.evaluate(() => window.hbRecovery.rejectRead());
    await page.getByRole("button", { name: "Show secret", exact: true }).waitFor();
    await page.evaluate(() => window.hbRecovery.hide());
    await page.waitForSelector("#svc-reveal-title", { state: "detached" });
    await page.evaluate(() => window.hbRecovery.show());
    await page.getByRole("button", { name: "Show secret", exact: true }).waitFor();
    assert.equal((await page.evaluate(() => window.hbRecovery.snapshot())).calls, 1);
    await page.getByRole("button", { name: "Show secret", exact: true }).click();
    await page.waitForFunction(() => window.hbRecovery.snapshot().calls === 2);
    await page.evaluate(() => window.hbRecovery.resolve());
    await page.waitForSelector("#svc-secret");
  });
  for (const shown of [false, true]) await scenario(`revocation ${shown ? "shown-secret declaration" : "unknown storage"} keeps its meaning`, async (page) => {
    await page.evaluate(async () => { await window.hbRecovery.issue(); window.hbRecovery.show(); });
    await page.waitForFunction(() => window.hbRecovery.snapshot().calls === 1);
    if (shown) {
      await page.evaluate(() => window.hbRecovery.resolve());
      await page.waitForSelector("#svc-secret");
      await page.getByRole("button", { name: "I didn't store it: revoke", exact: true }).click();
    } else {
      await page.evaluate(() => window.hbRecovery.rejectRead());
      await page.getByRole("button", { name: "Show secret", exact: true }).waitFor();
      await page.getByRole("button", { name: "Revoke credential", exact: true }).click();
    }
    assert.equal((await page.evaluate(() => window.hbRecovery.snapshot())).lastRevoke, shown);
  });
  console.log(JSON.stringify({ scope: "Isolated real React StrictMode provider/Reveal; visibility and pagehide are controlled fixture simulations, not native BFCache or complete M04 acceptance", runtime: { node: process.version, browser: browser.version(), platform: process.platform, architecture: process.arch }, results, errors }, null, 2));
  if (errors.length || results.some((r) => r.status !== "PASS")) process.exitCode = 1;
} finally { await browser?.close(); await server.close(); }
