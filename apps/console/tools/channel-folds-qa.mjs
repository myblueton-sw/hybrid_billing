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

  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 1280, 1024, 768, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/integrations?lang=${lang}&view=channels`);
      await page.locator(".p02-fold").first().waitFor();
      await page.waitForLoadState("networkidle");
      const folds = await page.locator(".p02-fold > summary").all();
      for (const s of folds) await s.click();
      const m = await page.evaluate(() => {
        const view = document.documentElement.clientWidth;
        const past = [...document.querySelectorAll(".p02-fold *")].filter((el) => { const r = el.getBoundingClientRect(); return r.width > 0 && (r.right > view + 0.5 || r.left < -0.5); }).slice(0, 3).map((el) => el.tagName + "." + el.className);
        const logins = [...document.querySelectorAll(".p02-fold .p02-login")].map((el) => { const r = el.getBoundingClientRect(), cell = el.closest("td").getBoundingClientRect(); return r.left >= cell.left - 0.5 && r.right <= cell.right + 0.5; });
        const cells = [...document.querySelectorAll(".p02-fold td")].filter((td) => td.getBoundingClientRect().width > 0);
        const spill = cells.filter((td) => [...td.querySelectorAll("*")].some((el) => { const r = el.getBoundingClientRect(), c = td.getBoundingClientRect(); return r.width > 0 && (r.right > c.right + 1 || r.left < c.left - 1); })).length;
        const tags = [...document.querySelectorAll(".p02-fold td .p02-tag")].every((el) => el.getClientRects().length === 1);
        return { page: document.documentElement.scrollWidth, view, past, logins: logins.length, inside: logins.every(Boolean), spill, tags };
      });
      check(`${lang} ${width} with every fold open the page is not wider than the viewport, nothing spills out of a cell and no state tag is broken across lines`, folds.length > 0 && m.page <= m.view + 1 && m.past.length === 0 && m.logins > 0 && m.inside && m.spill === 0 && m.tags, JSON.stringify(m));
      if (args.shots && (width === 1024 || width === 1440)) { await page.locator(".p02-fold[open]").first().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p02-folds-${lang}-${width}.png`, fullPage: false }); }
    }
  }
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
