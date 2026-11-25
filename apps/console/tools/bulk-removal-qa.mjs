import { createRequire } from "node:module";
import { readFile, mkdir } from "node:fs/promises";
import { randomBytes } from "node:crypto";
const args = Object.fromEntries(process.argv.slice(2).flatMap((v, i, all) => (v.startsWith("--") ? [[v.slice(2), all[i + 1]]] : [])));
const { chromium } = createRequire(import.meta.url)(args["playwright-module"] || "playwright");
const base = (args.url || "http://127.0.0.1:4173").replace(/\/$/, "");
const browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
const page = await context.newPage();
const results = [], errors = [], notes = [];
let navigation = 0;
const check = (name, ok, detail = "") => { results.push(`${ok ? "PASS" : "FAIL"} ${name}${!ok && detail ? " " + String(detail).replace(/\s+/g, " ").slice(0, 300) : ""}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const root = () => page.locator('[data-security-screen="M01"]');
const boxes = () => root().locator('tbody input[type="checkbox"], .live-list input[type="checkbox"]');
const sent = [];
page.on("request", (r) => { if (r.method() === "POST" && new URL(r.url()).pathname.startsWith("/api/v1/")) sent.push(r.url() + " " + (r.postData() ?? "")); });
async function open(lang) {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/users?lang=${lang}&view=users`);
  await root().waitFor();
  await page.locator(".live-open[data-user-id]").first().waitFor();
  await page.waitForLoadState("networkidle");
}
async function contained(name) {
  const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
  check(`${name} page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
}

try {
  if (args.shots) await mkdir(args.shots, { recursive: true });
  const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
  const activation = new URL(lines.find((l) => l.startsWith("admin@example.invalid ")).split(" ")[1]);
  await page.goto(`${base}/activate?lang=en${activation.hash}`);
  await page.locator("#link-new").fill(password); await page.locator("#link-confirm").fill(password);
  await page.locator("form button[type=submit]").click(); await page.locator("a.service-button-link").waitFor();
  await page.goto(`${base}/?service=api&lang=en`);
  await page.locator("#signin-workspace").fill("sample-workspace"); await page.locator("#signin-email").fill("admin@example.invalid");
  await page.locator("#signin-password").fill(password); await page.locator("form button[type=submit]").click();
  await page.locator(".service-session").waitFor();

  for (const lang of ["en", "ko", "ar"]) {
    const at = `${lang} removal`;
    await page.setViewportSize({ width: 1440, height: 1000 });
    await open(lang);
    await root().locator(".m01-match-open").click();
    await root().locator(".m01-match-confirm").waitFor();
    await root().locator(".m01-match-confirm").click();
    await root().locator(".m01-fixedbar").waitFor();
    await page.waitForLoadState("networkidle");
    check(`${at} a fixed list has no checkboxes and offers removal between prepare and discard`, (await boxes().count()) === 0 && (await root().locator(".m01-remove-open").count()) === 1 && (await root().locator(".m01-removed").count()) === 0);
    await root().locator(".m01-remove-open").click();
    await root().locator(".m01-remove-send").waitFor();
    const free = root().locator('tbody input[type="checkbox"]:not([aria-disabled]):not([data-select-loaded]), .live-list input[type="checkbox"]:not([aria-disabled]):not([data-select-loaded])');
    const available = await free.count();
    check(`${at} removal mode shows the checkboxes, the hint and a disabled send button, and hides prepare and discard`, available >= 1 && (await root().locator(".m01-remove-send").getAttribute("aria-disabled")) === "true"
      && (await root().locator("#m01-remove-hint").innerText()).trim().length > 20 && (await root().locator(".m01-prepare, .m01-discard-list").count()) === 0, String(available));
    check(`${at} the checkboxes are named as keeping access and the send button says nothing is selected`, (await root().locator('input[data-keep="true"]').count()) === available && !/\d/.test(await root().locator(".m01-remove-send").innerText()));
    await free.nth(0).check();
    await root().locator(".m01-remove-cancel").click();
    await root().locator(".m01-remove-open").waitFor();
    check(`${at} stopping hides the checkboxes again`, (await boxes().count()) === 0 && (await root().locator(".m01-removed").count()) === 0);
    await root().locator(".m01-remove-open").click();
    await root().locator(".m01-remove-send").waitFor();
    check(`${at} starting again begins with nobody selected`, (await root().locator('tbody input[type="checkbox"]:checked, .live-list input[type="checkbox"]:checked').count()) === 0 && (await root().locator(".m01-remove-send").getAttribute("aria-disabled")) === "true");
    for (const width of [1440, 1024, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await contained(`${at} ${width} removal mode`);
      if (args.shots) await page.screenshot({ path: `${args.shots}/m01-removal-mode-${lang}-${width}.png`, fullPage: true });
    }
    await page.setViewportSize({ width: 1440, height: 1000 });
    sent.length = 0;
    await root().locator(".m01-remove-send").click({ force: true });
    await page.waitForLoadState("networkidle");
    check(`${at} an empty selection sends nothing`, sent.length === 0 && (await root().locator(".m01-remove-send").count()) === 1);
    await free.nth(0).check();
    await page.waitForFunction(() => document.querySelector(".m01-remove-send")?.getAttribute("aria-disabled") !== "true");
    check(`${at} the send button names how many are selected`, /1/.test(await root().locator(".m01-remove-send").innerText()));
    await root().locator(".m01-remove-send").click();
    await root().locator(".m01-remove-open").waitFor();
    await page.waitForLoadState("networkidle");
    const bar = await root().locator(".m01-fixedbar").innerText();
    check(`${at} the command was sent once and the bar says one was excluded`, sent.filter((b) => b.includes("bulk_revocation.remove_targets")).length === 1 && /1/.test(await root().locator(".m01-removed").innerText())
      && (await root().locator(".m01-removal-note").count()) === 1 && (await boxes().count()) === 0 && (await page.evaluate(() => document.activeElement?.classList.contains("m01-removed"))) === true, bar);
    check(`${at} no placeholder text`, !/undefined|NaN|\[object Object\]/.test(bar), bar);
    await contained(`${at} after removing`);
    if (args.shots) await page.screenshot({ path: `${args.shots}/m01-removal-done-${lang}-1440.png`, fullPage: true });
    await open(lang);
    await root().locator(".m01-fixedbar").waitFor();
    check(`${at} the number removed survives a reload of the screen`, /1/.test(await root().locator(".m01-removed").innerText()));
    await root().locator(".m01-prepare").click();
    await root().locator(".m01-effect-form textarea").fill("rendered check");
    await root().locator(".m01-effect-form button[type=submit]").click();
    await page.locator(".m01-kept, .service-summary").first().waitFor();
    await page.waitForLoadState("networkidle");
    const kept = await page.locator(".m01-kept").count();
    notes.push(`${lang} after prepare: ${kept ? "prepared, kept " + (await page.locator(".m01-kept").first().innerText()) : "refused: " + (await page.locator(".service-summary").first().innerText()).replace(/\s+/g, " ").slice(0, 160)}`);
    check(`${at} the preview counts the removed people as keeping their access and lists the reason`, kept >= 1 && (await page.locator(".m01-kept").first().getAttribute("data-kept")) === "1" && (await page.locator(".m01-kept").first().innerText()).trim().length > 5);
    await contained(`${at} preview`);
    if (args.shots) await page.screenshot({ path: `${args.shots}/m01-removal-preview-${lang}-1440.png`, fullPage: true });
    await page.setViewportSize({ width: 390, height: 1000 });
    await contained(`${at} 390 preview`);
    if (args.shots) await page.screenshot({ path: `${args.shots}/m01-removal-preview-${lang}-390.png`, fullPage: true });
  }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("en");
  await root().locator(".m01-match-open").click();
  await root().locator(".m01-match-confirm").waitFor();
  await root().locator(".m01-match-confirm").click();
  await root().locator(".m01-fixedbar").waitFor();
  await root().locator(".m01-remove-open").click();
  await root().locator(".m01-remove-send").waitFor();
  const picks = root().locator('tbody input[type="checkbox"]:not([aria-disabled]):not([data-select-loaded])');
  await picks.nth(0).check();
  await picks.nth(1).check();
  await root().locator(".m01-remove-cancel").click();
  await root().locator(".m01-discard-list").click();
  await root().locator(".m01-match-open").waitFor();
  await page.waitForLoadState("networkidle");
  const selectedForRevocation = await root().locator('tbody input[type="checkbox"]:checked').count();
  const keepNamed = await root().locator('input[data-keep="true"]').count();
  check("people picked to keep access are not selected for revocation after the list is discarded", selectedForRevocation === 0 && keepNamed === 0 && (await root().locator('tbody input[type="checkbox"]').count()) > 0 && (await root().locator(".m01-prepare").count()) === 0,
    `${selectedForRevocation} ${keepNamed}`);
  check("no unexpected JavaScript errors", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("runner completed without unexpected exception", false, String(e));
  if (args.shots) await page.screenshot({ path: `${args.shots}/runner-failure.png`, fullPage: true }).catch(() => {});
} finally {
  results.forEach((r) => console.log(r));
  notes.forEach((n) => console.log("NOTE " + n));
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
  await browser.close();
}
