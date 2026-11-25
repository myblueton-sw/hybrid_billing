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
const check = (name, ok, detail = "") => { results.push(`${ok ? "PASS" : "FAIL"} ${name}${!ok && detail ? " " + String(detail).replace(/\s+/g, " ").slice(0, 300) : ""}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const copy = {
  en: { standing: "Kept until item is gone", outcome: "Skipped: item not gone", unknown: "Unknown (" },
  ko: { standing: "원 항목 완전 삭제 전", outcome: "제외: 원 항목 삭제 전", unknown: "알 수 없음(" },
  ar: { standing: "بانتظار زوال العنصر الأصلي", outcome: "مستبعد: العنصر الأصلي باقٍ", unknown: "غير معروف (" },
};
const root = () => page.locator('[data-screen-id="P04"]');
async function open(lang, view) {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/retention?lang=${lang}&view=${view}`);
  await root().waitFor();
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => !document.querySelector('[data-screen-id="P04"] [aria-busy="true"]'));
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
    for (const width of [1440, 1024, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      for (const view of ["expiry", "deletion"]) {
        const at = `${lang} ${width} ${view}`;
        await open(lang, view);
        const text = await root().innerText();
        check(`${at} is live in its language and direction`, (await root().getAttribute("data-source-state")) === "live" && (await root().getAttribute("lang")) === lang && (await root().getAttribute("dir")) === (lang === "ar" ? "rtl" : "ltr"));
        check(`${at} names every value it shows`, !text.includes(copy[lang].unknown) && !/undefined|NaN|\[object Object\]/.test(text), text.match(/.{0,40}(Unknown|알 수 없음|غير معروف|undefined|NaN).{0,40}/)?.[0]);
        check(`${at} names the guarding value`, text.includes(view === "expiry" ? copy[lang].standing : copy[lang].outcome));
        await contained(at);
        if (args.shots) await page.screenshot({ path: `${args.shots}/p04-${view}-${lang}-${width}.png`, fullPage: true });
      }
      await open(lang, "expiry");
      const strips = root().locator("dl.p04-strip-8");
      const claim = await strips.evaluateAll((els, word) => els.map((e) => [...e.querySelectorAll(":scope > div")].map((d) => d.innerText.replace(/\s+/g, " ").trim())).filter((cells) => cells.some((c) => c.includes(word) && /(^|\D)1(\D|$)/.test(c))).length, copy[lang].standing);
      check(`${lang} ${width} one class counts one guarding record and the others none`, claim === 1, String(claim));
    }
  }
  check("no unexpected JavaScript errors", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("runner completed without unexpected exception", false, String(e));
  if (args.shots) await page.screenshot({ path: `${args.shots}/runner-failure.png`, fullPage: true }).catch(() => {});
} finally {
  results.forEach((r) => console.log(r));
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
  await browser.close();
}
