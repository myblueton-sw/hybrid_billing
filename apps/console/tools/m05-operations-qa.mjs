import { inspectView } from "./view-measure.mjs";
import { createRequire } from "node:module";
import { readFile, mkdir } from "node:fs/promises";
import { randomBytes } from "node:crypto";
const args = Object.fromEntries(process.argv.slice(2).flatMap((v, i, all) => (v.startsWith("--") ? [[v.slice(2), all[i + 1]]] : [])));
const { chromium } = createRequire(import.meta.url)(args["playwright-module"] || "playwright");
const base = (args.url || "http://127.0.0.1:4173").replace(/\/$/, "");
const browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
const page = await context.newPage();
const results = [], errors = [];
let navigation = 0;
const check = (name, ok, detail = "") => { results.push(`${ok ? "PASS" : "FAIL"} ${name}${!ok && detail ? " " + String(detail).replace(/\s+/g, " ").slice(0, 400) : ""}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const copy = {
  en: { first: "First", previous: "Previous", next: "Next", onPage: (n) => `Rows on this page: ${n}` },
  ko: { first: "처음", previous: "이전", next: "다음" },
  ar: { first: "الأول", previous: "السابق", next: "التالي" },
};
const op = (i) => ({ operation: `qa-op-${String(i).padStart(2, "0")}`, selection: "qa-sel", kind: "others", state: "unconfirmed", recorded_count: 1, ended_count: null, gone_count: null, not_included_count: null,
  blocked_at: null, list_generation: "g1", current_generation: null, generation_changed: null, digest: "d", cleanup: null, replay: false });
const all = Array.from({ length: 23 }, (_, i) => op(i + 1));
let asked = [];
let sessionReads = 0;
const eventually = async (f) => { for (let i = 0; i < 80; i++) { if (await f()) return true; await page.waitForTimeout(50); } return false; };

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
  page.on("request", (r) => { if (new URL(r.url()).pathname.endsWith("/me/sessions")) sessionReads++; });
  await page.route((url) => new URL(url.toString()).pathname.endsWith("/me/sessions/operations"), async (route) => {
    const qs = new URL(route.request().url()).searchParams;
    asked.push(qs.toString());
    const start = qs.get("after") ? Number(qs.get("after").slice(1)) : 0;
    const rows = all.slice(start, start + 10);
    await route.fulfill({ status: 200, json: { rows, accepted_count: all.length, next: start + 10 < all.length ? "p" + (start + 10) : "", reference_at: "2026-11-27T09:00:00Z", generation: "g1" } });
  });

  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 390]]) {
    const at = `${lang} ${width}`;
    await page.setViewportSize({ width, height: 1000 });
    asked = [];
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/my-security?lang=${lang}&view=sessions`);
    const section = page.locator("section.mysec-ends");
    await section.locator(".mysec-ops > li").first().waitFor();
    const nav = section.locator(":scope > .list-pager .list-pager-nav button");
    check(`${at}: the operations list ends with the shared pager: rows on this page and First, Previous, Next`, (await nav.count()) === 3 && (await nav.allInnerTexts()).map((t) => t.trim()).join("|") === [copy[lang].first, copy[lang].previous, copy[lang].next].join("|") && (await section.locator(":scope > .list-pager .list-pager-rows").innerText()).includes("10"));
    check(`${at}: on the first page First and Previous are disabled`, await nav.nth(0).isDisabled() && await nav.nth(1).isDisabled() && await nav.nth(2).isEnabled());
    const shown = await inspectView(page);
    check(`${at}: nothing overflows, is cut off or loses contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 3), cut: shown.cut.slice(0, 3), low: shown.low.slice(0, 3) }));
    if (args.shots) { await page.mouse.move(0, 0); await section.screenshot({ path: `${args.shots}/m05-operations-${lang}-${width}.png` }); }
    sessionReads = 0;
    await nav.nth(2).click();
    await eventually(async () => asked.some((q) => q.includes("after=p10")));
    check(`${at}: paging the operations list does not reread the sessions list`, sessionReads === 0, String(sessionReads));
    await section.locator('.mysec-ops > li section[aria-labelledby="mysec-op-qa-op-11"]').waitFor();
    check(`${at}: Next reads the following page and moves focus to the section heading`, await eventually(() => page.evaluate(() => document.activeElement?.id === "mysec-ends-title")));
    check(`${at}: on the second page Previous is enabled`, await nav.nth(1).isEnabled());
    const opA = () => page.evaluate(() => new URLSearchParams(location.hash.split("?")[1]).get("op_a"));
    check(`${at}: the operations page position is in the address`, (await opA()) === "p10");
    await page.goBack();
    await section.locator('.mysec-ops > li section[aria-labelledby="mysec-op-qa-op-01"]').waitFor();
    check(`${at}: Back returns to the first page`, (await opA()) === null);
    await page.goForward();
    await section.locator('.mysec-ops > li section[aria-labelledby="mysec-op-qa-op-11"]').waitFor();
    check(`${at}: Forward returns to the second page`, (await opA()) === "p10");
    await page.reload();
    await page.locator("section.mysec-ends .mysec-ops > li").first().waitFor();
    await section.locator('.mysec-ops > li section[aria-labelledby="mysec-op-qa-op-11"]').waitFor();
    check(`${at}: reload keeps the second page`, (await opA()) === "p10");
    await nav.nth(1).click();
    await section.locator('.mysec-ops > li section[aria-labelledby="mysec-op-qa-op-01"]').waitFor();
    check(`${at}: Previous returns to the first page`, await nav.nth(1).isDisabled());
  }
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, (e.stack ?? String(e)) + " URL " + page.url());
} finally {
  await browser.close();
  const failed = results.filter((r) => r.startsWith("FAIL"));
  console.log((args.quiet ? failed : results).join("\n"));
  console.log(`RESULT ${results.length - failed.length} passed ${failed.length} failed`);
}
