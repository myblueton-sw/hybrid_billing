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
const copy = {
  en: { failed: "Could not be loaded.", again: "Try again" },
  ko: { failed: "불러오지 못했습니다.", again: "다시 시도" },
  ar: { failed: "تعذّر التحميل.", again: "إعادة المحاولة" },
};
const views = ["tree", "accounts", "groups", "history"];
const widths = [1440, 390];
const root = () => page.locator('[data-screen-id="O01"]');
const facts = (url) => { const u = new URL(url); return u.pathname.endsWith("/api/v1/organizations") && u.searchParams.get("kind") === "hierarchy" && u.searchParams.has("member"); };

async function open(lang, view) {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/organizations?lang=${lang}&view=${view}`);
  await root().waitFor();
  await page.waitForFunction(() => { const r = document.querySelector('[data-screen-id="O01"]'); return !!r && !r.querySelector('[aria-busy="true"]') && r.innerText.trim().length > 10; });
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
    for (const width of widths) {
      await page.setViewportSize({ width, height: 1000 });
      for (const view of views) {
        await open(lang, view);
        check(`${lang} ${width} ${view} is live with its language and direction`, (await root().getAttribute("data-source-state")) === "live" && (await root().getAttribute("lang")) === lang && (await root().getAttribute("dir")) === (lang === "ar" ? "rtl" : "ltr"));
        const text = await root().innerText();
        check(`${lang} ${width} ${view} no placeholder text and no load failure`, !/undefined|NaN|\[object Object\]/.test(text) && (await root().locator(".live-error").count()) === 0, text.match(/.{0,40}(undefined|NaN|\[object Object\]).{0,40}/)?.[0]);
        await contained(`${lang} ${width} ${view}`);
        if (args.shots) await page.screenshot({ path: `${args.shots}/o01-${view}-${lang}-${width}.png`, fullPage: true });
      }
    }
  }

  for (const lang of ["en", "ko", "ar"]) {
    for (const width of widths) {
      await page.setViewportSize({ width, height: 1000 });
      await open(lang, "accounts");
      const rows = page.locator("#o01-list .live-open[data-id]");
      if (lang === "en" && width === 1440) { notes.push(`accounts listed ${await rows.count()}`); check("the sample lists at least one account", (await rows.count()) > 0); }
      let asked = 0;
      await page.route((url) => facts(url.toString()), (route) => { asked++; return route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ code: "dependency_unavailable", request_id: "o01-facts-qa" }) }); });
      await rows.first().click();
      const failed = root().getByText(copy[lang].failed, { exact: true });
      await failed.first().waitFor();
      const again = root().getByRole("button", { name: copy[lang].again, exact: true });
      check(`${lang} ${width} a failed read of the member's facts is stated with a way to repeat it`, asked > 0 && (await again.count()) > 0 && (await failed.first().getAttribute("role")) === "alert");
      await contained(`${lang} ${width} detail with the failed read`);
      if (args.shots) await page.screenshot({ path: `${args.shots}/o01-facts-failed-${lang}-${width}.png`, fullPage: true });
      await page.unrouteAll({ behavior: "wait" });
      await again.first().click();
      await failed.first().waitFor({ state: "detached" });
      check(`${lang} ${width} repeating the read replaces the failure`, (await root().getByText(copy[lang].failed, { exact: true }).count()) === 0);
    }
  }
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
