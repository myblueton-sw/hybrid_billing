import { measurePresentation } from "./presentation-measure.mjs";
import { createRequire } from "node:module";
import { readFile, mkdir, writeFile } from "node:fs/promises";
import { randomBytes } from "node:crypto";
import assert from "node:assert/strict";
import { resolve } from "node:path";
const args = Object.fromEntries(process.argv.slice(2).flatMap((v, i, all) => v.startsWith("--") ? [[v.slice(2), all[i + 1]]] : []));
const { chromium } = createRequire(import.meta.url)(args["playwright-module"] || "playwright");
const base = args.url || "http://127.0.0.1:8835", output = resolve(args.output || "qa-output/accessibility");
const browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } }), page = await context.newPage();
const evidence = { outcome: "running", checks: [], renders: [], errors: [] }, password = randomBytes(24).toString("base64url");
page.on("pageerror", (e) => evidence.errors.push(e.message));
if (args.api) await page.route("**/api/v1/**", async (route) => {
  const u = new URL(route.request().url());
  const response = await route.fetch({ url: args.api + u.pathname + u.search });
  await route.fulfill({ response });
});
await mkdir(output, { recursive: true });
async function tabTo(selector) {
  for (let i = 0; i < 180; i++) {
    if (await page.locator(selector).evaluate((e) => e === document.activeElement)) return;
    await page.keyboard.press("Tab");
  }
  throw new Error(`Keyboard could not reach ${selector}`);
}
async function capture(id, lang, width, mode) {
  await page.evaluate(() => document.fonts.ready);
  const fontEvidence = await page.evaluate(() => ({ failed: [...document.fonts].filter((f) => f.status === "error").map((f) => f.family), loaded: [...document.fonts].filter((f) => f.status === "loaded").map((f) => f.family) }));
  assert.deepEqual(fontEvidence.failed, []);
  assert.ok(fontEvidence.loaded.includes(lang === "ko" ? "IBM Plex Sans KR" : lang === "ar" ? "IBM Plex Sans Arabic" : "IBM Plex Sans"));
  const findings = await page.evaluate(() => {
    const visible = (e) => { const r = e.getBoundingClientRect(), s = getComputedStyle(e); return r.width > 0 && r.height > 0 && s.visibility !== "hidden" && s.display !== "none"; };
    const rgb = (s) => { const m = s.match(/^rgba?\(([^)]+)\)$/); if (!m) return null; return m[1].split(/[, ]+/).map(Number); };
    const luminance = (c) => c.slice(0, 3).map((v) => { v /= 255; return v <= .04045 ? v / 12.92 : ((v + .055) / 1.055) ** 2.4; }).reduce((n, v, i) => n + v * [.2126, .7152, .0722][i], 0);
    const contrast = [], uncertain = [];
    for (const e of document.querySelectorAll("body *")) {
      if (!visible(e) || e.closest("[disabled], [aria-disabled=true], svg") || ![...e.childNodes].some((n) => n.nodeType === Node.TEXT_NODE && n.textContent.trim())) continue;
      const s = getComputedStyle(e), foreground = rgb(s.color);
      let background = null, ambiguous = false;
      for (let n = e; n; n = n.parentElement) {
        const p = getComputedStyle(n), c = rgb(p.backgroundColor);
        if (p.backgroundImage !== "none" || Number(p.opacity) !== 1 || c && c.length === 4 && c[3] > 0 && c[3] < 1) ambiguous = true;
        if (!background && c && (c.length < 4 || c[3] === 1)) background = c;
      }
      const text = e.textContent.trim().slice(0, 90);
      if (ambiguous || !foreground || foreground.length === 4 && foreground[3] !== 1) { uncertain.push(text); continue; }
      background ||= [255, 255, 255];
      const a = luminance(foreground), b = luminance(background), ratio = (Math.max(a, b) + .05) / (Math.min(a, b) + .05);
      const large = parseFloat(s.fontSize) >= 24 || parseFloat(s.fontSize) >= 18.6667 && parseInt(s.fontWeight) >= 700;
      if (ratio < (large ? 3 : 4.5)) contrast.push({ tag: e.tagName, className: e.className, text, ratio, foreground: s.color, background, fontSize: s.fontSize });
    }
    const overflow = [...document.querySelectorAll("body *")].filter(visible).filter((e) => { const r = e.getBoundingClientRect(); return r.left < -1 || r.right > innerWidth + 1; }).slice(0, 15).map((e) => ({ tag: e.tagName, className: e.className }));
    return { horizontalOverflow: document.documentElement.scrollWidth > innerWidth + 1, overflow, contrast, uncertain, liveRegions: [...document.querySelectorAll("[role=status],[role=alert],[aria-live]")].map((e) => ({ role: e.getAttribute("role"), live: e.getAttribute("aria-live") })) };
  });
  const file = `${id}-${lang}-${width}.png`;
  await page.mouse.move(0, 0); await page.screenshot({ path: resolve(output, file), fullPage: true });
  evidence.renders.push({ id, lang, width, mode, file, fontEvidence, ...findings, ...(args.presentation ? { presentation: await measurePresentation(page) } : {}) });
  console.log(JSON.stringify({ id, lang, width, overflow: findings.horizontalOverflow, contrastCandidates: findings.contrast.length }));
  if (args.zoom && width === 640 && !id.endsWith("-zoom200")) {
    await page.evaluate(() => document.documentElement.style.zoom = "2");
    await capture(id + "-zoom200", lang, width, mode + ":CSS-zoom-not-native-browser-zoom");
    await page.evaluate(() => document.documentElement.style.zoom = "1");
  }
}
try {
  const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
  const memberContext = await browser.newContext(), memberPage = await memberContext.newPage();
  if (args.api) await memberPage.route("**/api/v1/**", async (route) => { const u = new URL(route.request().url()); const response = await route.fetch({ url: args.api + u.pathname + u.search }); await route.fulfill({ response }); });
  const memberLink = new URL(lines.find((line) => line.startsWith("member@example.invalid ")).split(" ")[1]);
  await memberPage.goto(`${base}/activate?lang=en${memberLink.hash}`);
  await memberPage.locator("#link-new").fill(password); await memberPage.locator("#link-confirm").fill(password); await memberPage.locator("form button[type=submit]").click(); await memberPage.locator("a.service-button-link").waitFor();
  await memberPage.goto(`${base}/?service=api&lang=en`);
  await memberPage.locator("#signin-workspace").fill("sample-workspace"); await memberPage.locator("#signin-email").fill("member@example.invalid"); await memberPage.locator("#signin-password").fill(password); await memberPage.locator("form button[type=submit]").click(); await memberPage.locator(".service-session").waitFor();
  await memberContext.close();
  evidence.checks.push("fixture setup: seeded member password set and sign-in accepted pending membership; not keyboard evidence");
  const activation = new URL(lines.find((line) => line.startsWith("admin@example.invalid ")).split(" ")[1]);
  await page.goto(`${base}/activate?lang=en${activation.hash}`);
  await page.locator("#link-new").waitFor();
  if (!args.journeys) for (const lang of ["en", "ko", "ar"]) for (const width of [1440, 640, 320]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto(`${base}/activate?lang=${lang}${activation.hash}`); await page.locator("#link-new").waitFor();
    await capture("L02", lang, width, "live-password-set");
  }
  await page.goto(`${base}/activate?lang=en${activation.hash}`); await page.locator("#link-new").waitFor();
  await page.setViewportSize({ width: 1440, height: 1000 });
  await tabTo("#link-new"); await page.keyboard.type(password); await tabTo("#link-confirm"); await page.keyboard.type(password);
  await tabTo("form button[type=submit]"); await page.keyboard.press("Enter"); await page.locator("a.service-button-link").waitFor();
  evidence.checks.push("keyboard activation completed using synthetic one-time link");
  if (!args.journeys) for (const lang of ["en", "ko", "ar"]) for (const width of [1440, 640, 320]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto(`${base}/?service=api&view=help&lang=${lang}`); await page.locator(".service-task h1").waitFor(); await capture("L03", lang, width, "live-help");
    await page.setViewportSize({ width, height: 1000 });
    await page.goto(`${base}/?service=api&lang=${lang}`); await page.locator("#signin-email").waitFor(); await capture("L01", lang, width, "live-signin");
  }
  await page.setViewportSize({ width: 1440, height: 1000 }); await page.goto(`${base}/?service=api&lang=en`);
  await tabTo("#signin-workspace"); await page.keyboard.type("sample-workspace"); await tabTo("#signin-email"); await page.keyboard.type("admin@example.invalid"); await tabTo("#signin-password"); await page.keyboard.type(password); await tabTo("form button[type=submit]"); await page.keyboard.press("Enter"); await page.locator(".service-session").waitFor();
  evidence.checks.push("keyboard sign-in completed");
  const screens = (await Promise.all(["identity", "organizations", "administration"].map(async (name) => JSON.parse(await readFile(new URL(`../src/modules/${name}/screens.json`, import.meta.url), "utf8"))))).flat().filter((s) => /^[MOP]/.test(s.id));
  if (!args.journeys) for (const lang of ["en", "ko", "ar"]) for (const width of [1440, 640, 320]) for (const screen of screens) {
    await page.setViewportSize({ width, height: 1000 });
    const routes = { M01: "/identity/users", M04: "/identity/services", M05: "/me", O01: "/organizations", O02: "/organizations/plans", O03: "/contracts", P01: "/installation", P02: "/integrations", P03: "/recovery", P04: "/retention", P05: "/release-evidence" };
    const response = routes[screen.id] ? page.waitForResponse((r) => new URL(r.url()).pathname === "/api/v1" + routes[screen.id] && r.request().method() === "GET") : null;
    await page.goto(`${base}/?service=api&lang=${lang}#${screen.path}?lang=${lang}&view=${screen.tabs[0].key}`);
    const settledResponse = response ? await response : null;
    if (settledResponse) { await settledResponse.finished(); assert.equal(settledResponse.status(), 200, `${screen.id} expected synthetic authorized read`); }
    await page.evaluate(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(done))));
    await page.locator(`[data-screen-id="${screen.id}"], [data-security-screen="${screen.id}"]`).first().waitFor();
    await page.waitForFunction(() => ![...document.querySelectorAll('[aria-busy="true"]')].some((e) => e.getBoundingClientRect().height > 0));
    assert.ok(await page.locator("main h1").first().isVisible());
    await capture(screen.id, lang, width, ["P06", "P07"].includes(screen.id) ? "preview-only-main" : ["M02", "M03"].includes(screen.id) ? "live-static-unavailable-guidance" : "live");
  }
  await page.setViewportSize({ width: 1440, height: 1000 });
  const commands = [];
  page.on("request", (r) => { const path = new URL(r.url()).pathname; if (path.startsWith("/api/v1/commands/")) commands.push(path); });
  const usersRead = page.waitForResponse((r) => new URL(r.url()).pathname === "/api/v1/identity/users");
  await page.goto(`${base}/?service=api&lang=en#/users?lang=en&view=users`); const usersResponse = await usersRead; await usersResponse.finished();
  const seeded = (await usersResponse.json()).users.find((u) => u.user_id === "sample-member");
  assert.equal(seeded.membership?.state, "active"); assert.equal(seeded.membership?.role, "member");
  await page.locator('input[data-select="sample-member"]').waitFor();
  await tabTo("#m01-q"); await page.keyboard.type("member@example.invalid");
  const search = page.waitForResponse((r) => new URL(r.url()).pathname === "/api/v1/identity/users" && new URL(r.url()).searchParams.get("q") === "member@example.invalid");
  await page.keyboard.press("Enter"); await (await search).finished();
  await tabTo('input[data-select="sample-member"]'); await page.keyboard.press("Space");
  assert.equal(await page.locator('input[data-select="sample-member"]').isChecked(), true);
  await tabTo(".m01-prepare"); await page.keyboard.press("Enter"); await page.locator("#m01-bulk-title:focus").waitFor();
  await tabTo('input[name="m01-effect"]:checked');
  if (!await page.locator('input[name="m01-effect"][value="suspend_membership"]').isChecked()) await page.keyboard.press("ArrowUp");
  assert.equal(await page.locator('input[name="m01-effect"][value="suspend_membership"]').isChecked(), true);
  await tabTo("#live-reason"); await page.keyboard.type("Synthetic accessibility preview; discard without execution");
  const preparedRead = page.waitForResponse((r) => new URL(r.url()).pathname.endsWith("/commands/bulk_revocation.prepare"));
  await tabTo('.m01-effect-form button[type="submit"]'); await page.keyboard.press("Enter");
  const preparedResponse = await preparedRead; assert.equal(preparedResponse.status(), 200); const prepared = await preparedResponse.json(); assert.equal(prepared.fields.targets, "1"); assert.equal(prepared.fields.excluded, "0");
  await page.locator('[data-bulk="preview"]').waitFor(); await page.locator(".m01-continue").waitFor();
  assert.deepEqual(await page.locator(".m01-counts dd").allTextContents(), ["1", "0", "0"]);
  assert.match(await page.locator(".m01-targets .m01-people").innerText(), /member@example.invalid/);
  await tabTo(".m01-continue"); await page.keyboard.press("Enter"); await page.locator("dialog.m01-confirm[open]").waitFor(); await page.locator("#m01-confirm-title:focus").waitFor();
  const modalFocus = [];
  for (const key of [...Array(12).fill("Tab"), ...Array(12).fill("Shift+Tab")]) {
    await page.keyboard.press(key);
    const state = await page.evaluate(() => ({ focused: document.hasFocus(), tag: document.activeElement?.tagName, inside: !!document.activeElement?.closest("dialog[open]") }));
    modalFocus.push(state);
    assert.ok(state.inside || !state.focused && state.tag === "BODY", "No underlying page control may receive modal focus");
  }
  if (!await page.evaluate(() => document.hasFocus())) await page.keyboard.press("Tab");
  assert.equal(await page.evaluate(() => !!document.activeElement?.closest("dialog[open]")), true);
  evidence.checks.push({ modalFocus, browserChromeException: "Chromium native showModal reference reproduces temporary unfocused BODY before returning to dialog" });
  await page.screenshot({ path: resolve(output, "M01-keyboard-confirm.png"), fullPage: true });
  await page.keyboard.press("Escape"); assert.equal(await page.locator("dialog[open]").count(), 0); await page.locator(".m01-continue:focus").waitFor();
  await tabTo('.m01-bulk > .live-buttons .live-secondary');
  await page.keyboard.press("Enter"); await page.locator(".m01-prepare:focus").waitFor();
  await page.getByRole("status").filter({ hasText: "The prepared list was discarded. Nothing was revoked." }).waitFor();
  const unchanged = await page.evaluate(async () => { const session = await (await fetch("/api/v1/session", { headers: { "X-HB-Client": "console" } })).json(); const r = await fetch("/api/v1/identity/users?limit=50", { headers: { "X-Request-Id": crypto.randomUUID(), "X-CSRF-Token": session.csrf_token } }); const body = await r.json(); return { status: r.status, member: body.users?.find((u) => u.user_id === "sample-member") }; });
  assert.equal(unchanged.status, 200); assert.equal(unchanged.member.state, "active"); assert.equal(unchanged.member.membership.state, "active");
  assert.ok(commands.some((p) => p.endsWith("bulk_revocation.discard")));
  assert.ok(!commands.some((p) => p.endsWith("bulk_revocation.execute")));
  evidence.checks.push("M01 keyboard search, selection, membership preview, modal containment, Escape focus return and discard; no execute request");
  assert.deepEqual(evidence.errors, []);
  evidence.outcome = "passed";
  console.log(JSON.stringify({ completedChecks: evidence.checks.length, captures: evidence.renders.length, errors: evidence.errors }));
} catch (error) {
  evidence.outcome = "failed"; evidence.failure = { kind: error.name };
  await page.screenshot({ path: resolve(output, "failure.png"), fullPage: true });
  throw error;
} finally {
  await writeFile(resolve(output, "evidence.json"), JSON.stringify(evidence, null, 2));
  await browser.close();
}
