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
const check = (name, ok, detail = "") => { results.push(`${ok ? "PASS" : "FAIL"} ${name}${!ok && detail ? " " + String(detail).replace(/\s+/g, " ").slice(0, 400) : ""}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const ends = (url, path) => new URL(url).pathname.endsWith("/api/v1/" + path);
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

  await page.route((url) => ends(url.toString(), "retention"), async (route) => {
    const response = await route.fetch();
    const body = await response.json();
    body.holds.rows = body.holds.rows.map((h, n) => (n === 0 ? { ...h, outcomes: Array.from({ length: 50 }, (_, i) => ({ artifact: `zz-out-${String(i + 1).padStart(3, "0")}`, outcome: "not_preserved", stores: { "sample-object-store": "deleted" } })), outcomes_more: false, outcomes_next: "" } : h));
    await route.fulfill({ response, json: body });
  });
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 1024, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/retention?lang=${lang}&view=holds`);
      await page.locator("article.p04-hold").first().waitFor();
      await page.waitForLoadState("networkidle");
      const m = await page.locator("article.p04-hold").first().evaluate((card) => {
        const terms = [...card.querySelectorAll(".p04-hold-facts > div")].map((el) => ({ top: el.querySelector("dt").getBoundingClientRect().top, end: el.querySelector("dd").getBoundingClientRect().bottom }));
        const gaps = terms.slice(1).map((r, i) => Math.round(r.top - terms[i].end)).filter((g) => g >= 0);
        return { rows: card.querySelectorAll(".p04-outcomes tbody tr").length, gap: Math.max(0, ...gaps), page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth };
      });
      check(`${lang} ${width} with fifty outcomes the hold's facts keep their own spacing and the page is not wider than the viewport`, m.rows === 50 && m.gap <= 40 && m.page <= m.view + 1, JSON.stringify(m));
      if (args.shots && width !== 390) { await page.locator("article.p04-hold").first().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-hold-card-${lang}-${width}.png`, fullPage: false }); }
    }
  }
  await page.unrouteAll({ behavior: "wait" });
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
