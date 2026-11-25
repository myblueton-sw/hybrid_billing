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
const copy = {
  en: { shown: (n, m) => `${n} of ${m} hardware records shown, in the order recorded`, more: "Load more hardware", failed: "More hardware could not be loaded. The hardware already shown is unchanged.", again: "Try again", reset: "The list was refreshed and starts again from the first hardware record." },
  ko: { shown: (n, m) => `장비 ${m}건 중 ${n}건 표시, 기록된 순서`, more: "장비 더 불러오기", failed: "장비를 더 불러오지 못했습니다. 이미 표시된 장비는 그대로입니다.", again: "다시 시도", reset: "목록을 새로 고쳐 첫 장비부터 다시 표시합니다." },
  ar: { shown: (n, m) => `سجلات الأجهزة المعروضة: ${n} من أصل ${m}، بترتيب التسجيل`, more: "تحميل أجهزة أخرى", failed: "تعذّر تحميل أجهزة أخرى. لم تتغيّر الأجهزة المعروضة.", again: "إعادة المحاولة", reset: "حُدِّثت القائمة وتبدأ من جديد بأول سجل جهاز." },
};
const open = async (lang) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/release-evidence?lang=${lang}&view=load`);
  await page.locator(".p05-hw-item").first().waitFor();
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => !document.querySelector(".p05-live[aria-busy=true]"));
};
const items = () => page.locator(".p05-hw-item");

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

  let served = null, token = "";
  const seen = async (r) => { if (ends(r.url(), "release-evidence") && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); } };
  page.on("response", seen);
  await open("en");
  page.off("response", seen);
  const real = served?.load?.hardware?.rows ?? [];
  notes.push(`served: ${real.length} hardware, total ${served?.load?.hardware?.total}, hardware_next ${JSON.stringify(served?.load?.hardware_next)}`);
  check("the real service lists the sample's hardware whole, numbered, and says nothing further exists", real.length > 0 && served.load.hardware.total === real.length && served.load.hardware_next === "" && real.every((h, i) => h.number === i + 1));
  check("with all hardware shown there is no count and no button", (await page.locator(".p05-hw-count, .p05-hw-more").count()) === 0 && (await items().count()) === real.length);
  const further = await page.evaluate(async (csrf) => {
    const a = await fetch(`/api/v1/release-evidence/hardware?limit=1`, { headers: { "X-CSRF-Token": csrf, "X-Request-Id": "qa-hardware" }, credentials: "include" });
    return { status: a.status, body: await a.json().catch(() => null) };
  }, token);
  check("the real service answers the hardware read", further.status === 200 && further.body.order === "recorded" && further.body.total === real.length && further.body.hardware[0].id === real[0].id && further.body.hardware[0].number === 1, JSON.stringify(further).slice(0, 300));
  const made = (n) => ({ ...real[0], number: n, id: `routed-hw-${n}` });
  const span = (from, to) => Array.from({ length: to - from + 1 }, (_, i) => made(from + i));
  const base0 = real.length;
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1, again = 0; const asked = [];
      await page.route((url) => ends(url.toString(), "release-evidence"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.revision = body.revision + (again ? "-again" + again : "");
        body.load = { ...body.load, hardware: { total: 70, more: true, rows: body.load.hardware.rows }, hardware_next: String(base0) };
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "release-evidence/hardware"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.toString());
        const answer = (rows, next) => route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "recorded", q: "", hardware: rows, next, matching: 70, total: 70 } });
        if (q.get("after") === "56" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        return q.get("after") === String(base0) ? answer(span(base0 + 1, 56), "56") : answer(span(57, 70), "");
      });
      await open(lang);
      check(`${at} the first hardware is listed with the count of all and the button`, (await items().count()) === base0 && (await page.locator(".p05-hw-count").innerText()).trim() === copy[lang].shown(base0, 70) && (await page.locator(".p05-hw-more-button").innerText()).trim() === copy[lang].more);
      const more = page.locator(".p05-hw-more-button");
      check(`${at} the button is at least 44 high`, (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll(".p05-hw-item").length === 56);
      check(`${at} the next hardware follows in the order recorded`, asked[0] === `limit=50&after=${base0}` && (await page.locator(".p05-hw-count").innerText()).trim() === copy[lang].shown(56, 70) && (await items().nth(base0).innerText()).includes(`routed-hw-${base0 + 1}`));
      await page.keyboard.press("Enter");
      await page.locator(".p05-hw-more [role=alert]").waitFor();
      const alert = await page.locator(".p05-hw-more").innerText();
      check(`${at} a failed read says so, keeps the hardware and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await items().count()) === 56 && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await page.locator(".p05-hw-more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-hardware-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll(".p05-hw-item").length === 70);
      check(`${at} at the end all seventy are listed and the count and button are gone`, (await page.locator(".p05-hw-more").count()) === 0 && (await page.locator(".p05-hw-count").count()) === 0);
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (width === 1440) {
        again += 1;
        await page.locator(".p05-reload").first().click();
        await page.waitForFunction((n) => document.querySelectorAll(".p05-hw-item").length === n, base0);
        check(`${at} after the screen is read again the list starts from the main read and says so`, (await page.locator(".p05-hw-reset").innerText()).trim() === copy[lang].reset);
        if (args.shots) { await page.locator(".p05-hw-count").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-hardware-reset-${lang}-${width}.png`, fullPage: false }); }
      }
      await page.unrouteAll({ behavior: "wait" });
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
