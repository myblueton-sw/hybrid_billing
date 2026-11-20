import { createRequire } from "node:module";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { randomBytes } from "node:crypto";
import { resolve } from "node:path";
import assert from "node:assert/strict";
const args = Object.fromEntries(process.argv.slice(2).flatMap((v, i, all) => v.startsWith("--") ? [[v.slice(2), all[i + 1]]] : []));
const { chromium } = createRequire(import.meta.url)(args["playwright-module"] || "playwright");
const base = args.url, upstream = args.api, locale = args.locale || "en", width = Number(args.width || 1440), output = resolve(args.output);
assert.ok(base && upstream && args.activation);
const browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
const links = (await readFile(args.activation, "utf8")).trim().split("\n"), password = randomBytes(24).toString("base64url");
const evidence = { locale, width, outcome: "running", checks: [], errors: [] };
let stage = "fixture setup";
await mkdir(output, { recursive: true });
async function actor(login) {
  const context = await browser.newContext({ viewport: { width, height: 1000 } }), page = await context.newPage();
  page.on("pageerror", (e) => evidence.errors.push(e.message));
  await page.route("**/api/v1/**", async (route) => { const u = new URL(route.request().url()); const response = await route.fetch({ url: upstream + u.pathname + u.search }); await route.fulfill({ response }); });
  const link = new URL(links.find((line) => line.startsWith(login + " ")).split(" ")[1]);
  await page.goto(`${base}/activate?lang=${locale}${link.hash}`);
  await page.locator("#link-new").fill(password); await page.locator("#link-confirm").fill(password); await page.locator("form button[type=submit]").click(); await page.locator("a.service-button-link").waitFor();
  await page.goto(`${base}/?service=api&lang=${locale}`);
  await page.locator("#signin-workspace").fill("sample-workspace"); await page.locator("#signin-email").fill(login); await page.locator("#signin-password").fill(password); await page.locator("form button[type=submit]").click(); await page.locator(".service-session").waitFor();
  return page;
}
async function readAccess(page) {
  return page.evaluate(async () => { const session = await (await fetch("/api/v1/session", { headers: { "X-HB-Client": "console" } })).json(); const response = await fetch("/api/v1/identity/access", { headers: { "X-CSRF-Token": session.csrf_token, "X-Request-Id": crypto.randomUUID() } }); return { status: response.status, body: await response.json() }; });
}
async function tabTo(page, selector) {
  for (let i = 0; i < 240; i++) { if (await page.locator(selector).evaluate((e) => e === document.activeElement)) return; await page.keyboard.press("Tab"); }
  throw new Error(`Keyboard could not reach ${selector}`);
}
async function captureState(page, state) {
  evidence.visual ||= [];
  for (const size of [1440, 640, 390]) {
    await page.setViewportSize({ width: size, height: 1000 });
    await page.waitForFunction((wide) => document.querySelector('[data-live="grants"]')?.getAttribute("data-wide") === String(wide), size >= 1280);
    if (state === "approval-error") {
      if (!await page.locator(".service-summary").isVisible()) {
        await tabTo(page, '.live-step button[type="submit"]'); await page.keyboard.press("Enter");
      }
      await tabTo(page, '.service-summary a[href="#live-reason"]'); await page.keyboard.press("Enter"); await page.locator("#live-reason:focus").waitFor();
    }
    await page.evaluate(() => document.fonts.ready);
    await page.evaluate(() => { scrollTo(0, 0); return new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done))); });
    const layout = await page.evaluate(() => ({
      overflow: document.documentElement.scrollWidth > innerWidth,
      fonts: [...document.fonts].filter((f) => f.status === "loaded").map((f) => f.family),
      failedFonts: [...document.fonts].filter((f) => f.status === "error").map((f) => f.family),
      borders: [...document.querySelectorAll('.service-summary, .service-status, .live-panel, .live-step, [aria-current="page"], :focus')].map((e) => {
        const c = getComputedStyle(e);
        return { tag: e.tagName, className: e.className, focused: e === document.activeElement, borders: [c.borderTopColor, c.borderRightColor, c.borderBottomColor, c.borderLeftColor], widths: [c.borderTopWidth, c.borderRightWidth, c.borderBottomWidth, c.borderLeftWidth], outline: c.outline, background: c.backgroundColor };
      })
    }));
    assert.equal(layout.overflow, false); assert.deepEqual(layout.failedFonts, []);
    assert.ok(layout.fonts.includes(locale === "ko" ? "IBM Plex Sans KR" : locale === "ar" ? "IBM Plex Sans Arabic" : "IBM Plex Sans"));
    evidence.visual.push({ state, width: size, ...layout });
    await page.screenshot({ path: resolve(output, `${state}-${size}.png`), fullPage: true });
    if (size === 1440) {
      await page.evaluate(() => scrollTo(0, 0));
      await page.screenshot({ path: resolve(output, `${state}-top.png`) });
    }
  }
  await page.setViewportSize({ width, height: 1000 });
  await page.waitForFunction((wide) => document.querySelector('[data-live="grants"]')?.getAttribute("data-wide") === String(wide), width >= 1280);
  if (state === "approval-error") await tabTo(page, "#live-reason");
}
let reviewer;
try {
  const member = await actor("member@example.invalid"); await member.context().close();
  const proposer = await actor("admin@example.invalid");
  const proposed = await proposer.evaluate(async () => {
    const session = await (await fetch("/api/v1/session", { headers: { "X-HB-Client": "console" } })).json();
    const headers = { "X-CSRF-Token": session.csrf_token, "X-Request-Id": crypto.randomUUID(), "Content-Type": "application/json" };
    const access = await (await fetch("/api/v1/identity/access", { headers })).json();
    const response = await fetch("/api/v1/commands/access.request_grant", { method: "POST", headers, body: JSON.stringify({ request_id: headers["X-Request-Id"], target_id: "grants", idempotency_key: crypto.randomUUID(), expected_revision: access.revision, rationale: "Synthetic accessibility approval fixture", body: { subject: "sample-member", action: "view", kind: "focus_data", fields: "rows", customer: "all", days: "10" } }) });
    return { status: response.status, body: await response.json() };
  });
  assert.equal(proposed.status, 200); assert.equal(proposed.body.business_state, "requested");
  const id = proposed.body.fields.request;
  assert.match(id, /^[A-Za-z0-9._:-]+$/);
  const own = (await readAccess(proposer)).body.requests.find((r) => r.id === id);
  assert.equal(own.can_approve, false); assert.equal(own.proposer, "sample-admin");
  evidence.checks.push("fixture created by first administrator; proposer cannot approve");
  reviewer = await actor("admin2@example.invalid");
  const reviewerIdentity = await reviewer.evaluate(async () => (await (await fetch("/api/v1/session", { headers: { "X-HB-Client": "console" } })).json()).user);
  assert.notEqual(reviewerIdentity, own.proposer);
  stage = "keyboard approval";
  const loading = reviewer.waitForResponse((r) => new URL(r.url()).pathname === "/api/v1/identity/access");
  await reviewer.goto(`${base}/?service=api&lang=${locale}#/access?lang=${locale}&view=grants`); await (await loading).finished();
  const approve = `tr[data-request="${id}"] [data-act="approve"]`;
  await reviewer.locator(approve).waitFor();
  await tabTo(reviewer, approve); await reviewer.keyboard.press("Enter"); await reviewer.locator(".live-step h2:focus").waitFor();
  await tabTo(reviewer, ".live-step .live-secondary"); await reviewer.keyboard.press("Enter"); await reviewer.locator(approve + ":focus").waitFor();
  evidence.checks.push("keyboard opens approval and cancellation restores originating row focus");
  await reviewer.keyboard.press("Enter"); await reviewer.locator(".live-step h2:focus").waitFor();
  await tabTo(reviewer, '.live-step button[type="submit"]'); await reviewer.keyboard.press("Enter"); await reviewer.locator(".service-summary:focus").waitFor();
  const beforeErrorLink = reviewer.url();
  await tabTo(reviewer, '.service-summary a[href="#live-reason"]'); await reviewer.keyboard.press("Enter"); await reviewer.locator("#live-reason:focus").waitFor(); assert.equal(reviewer.url(), beforeErrorLink);
  assert.equal(await reviewer.locator("#live-reason").getAttribute("aria-invalid"), "true");
  await captureState(reviewer, "approval-error");
  await reviewer.keyboard.type("Synthetic keyboard approval of read access");
  await tabTo(reviewer, '.live-step button[type="submit"]');
  assert.equal(await reviewer.locator('[data-live="grants"] > [role="status"]').textContent(), "");
  const refreshed = reviewer.waitForResponse((r) => new URL(r.url()).pathname === "/api/v1/identity/access" && r.status() === 200);
  const approved = reviewer.waitForResponse((r) => new URL(r.url()).pathname.endsWith("/commands/access.approve_request") && r.status() === 200);
  await reviewer.keyboard.press("Enter");
  const response = await approved; const result = await response.json(); assert.equal(result.business_state, "granted");
  await reviewer.locator("#access-requests:focus").waitFor();
  const announcement = { en: "Approved. member@example.invalid now has this access.", ko: "승인했습니다. member@example.invalid님에게 이 권한이 적용됩니다.", ar: "اعتُمد. أصبحت لدى member@example.invalid هذه الصلاحية." }[locale];
  await reviewer.waitForFunction((expected) => document.querySelector('[data-live="grants"] > [role="status"]')?.textContent === expected, announcement);
  await (await refreshed).finished();
  stage = "approval read-back";
  const final = await readAccess(reviewer); assert.equal(final.status, 200);
  const decided = final.body.requests.find((r) => r.id === id);
  assert.equal(decided.state, "applied"); assert.equal(decided.decided_by, reviewerIdentity); assert.equal(decided.subject, "sample-member");
  const grant = final.body.grants.find((g) => g.id === decided.result); assert.ok(grant); assert.equal(grant.subject, "sample-member"); assert.equal(grant.state, "active"); assert.equal(grant.effective, true);
  await reviewer.locator(`[data-grant="${decided.result}"]`).waitFor();
  assert.equal(await reviewer.locator(approve).count(), 0);
  await tabTo(reviewer, `section[aria-labelledby="access-requests"] details:has(tr[data-request="${id}"]) > summary`); await reviewer.keyboard.press("Enter");
  await reviewer.locator(`tr[data-request="${id}"]`).waitFor();
  assert.ok((await reviewer.locator(`tr[data-request="${id}"]`).innerText()).includes("admin2@example.invalid"));
  assert.equal(await reviewer.locator(`tr[data-request="${id}"] .live-strong`).textContent(), { en: "Approved", ko: "승인됨", ar: "اعتُمد" }[locale]);
  await reviewer.evaluate(() => document.fonts.ready);
  const fonts = await reviewer.evaluate(() => ({ failed: [...document.fonts].filter((f) => f.status === "error").map((f) => f.family), loaded: [...document.fonts].filter((f) => f.status === "loaded").map((f) => f.family) }));
  assert.deepEqual(fonts.failed, []); assert.ok(fonts.loaded.includes(locale === "ko" ? "IBM Plex Sans KR" : locale === "ar" ? "IBM Plex Sans Arabic" : "IBM Plex Sans")); evidence.fonts = fonts;
  assert.equal(await reviewer.evaluate(() => document.documentElement.scrollWidth > innerWidth), false);
  await captureState(reviewer, "approved");
  evidence.checks.push("keyboard error summary focuses required reason without route change; distinct administrator approves real synthetic read grant; status and request focus restored");
  assert.deepEqual(evidence.errors, []);
  evidence.outcome = "passed";
  console.log(JSON.stringify({ checks: evidence.checks.length, locale, width, errors: evidence.errors }));
} catch (error) {
  evidence.outcome = "failed"; evidence.failure = { stage, kind: error.name, ...(stage === "approval read-back" ? { message: error.message } : {}) };
  if (reviewer) await reviewer.screenshot({ path: resolve(output, "failure.png"), fullPage: true });
  throw new Error(`Approval QA failed during ${stage} (${error.name})`);
} finally { await writeFile(resolve(output, "evidence.json"), JSON.stringify(evidence, null, 2)); await browser.close(); }
