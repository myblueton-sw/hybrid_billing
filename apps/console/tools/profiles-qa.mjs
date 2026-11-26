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
  en: { shown: (n, m) => `${n} of ${m} load profiles shown, in the order recorded`, more: "Load more profiles", failed: "More load profiles could not be loaded. The profiles already shown are unchanged.", again: "Try again", reset: "The list was refreshed and starts again from the first load profile." },
  ko: { shown: (n, m) => `부하 프로파일 ${m}건 중 ${n}건 표시, 기록된 순서`, more: "부하 프로파일 더 불러오기", failed: "부하 프로파일을 더 불러오지 못했습니다. 이미 표시된 프로파일은 그대로입니다.", again: "다시 시도", reset: "목록을 새로 고쳐 첫 부하 프로파일부터 다시 표시합니다." },
  ar: { shown: (n, m) => `ملفات تعريف الحمل المعروضة: ${n} من أصل ${m}، بترتيب التسجيل`, more: "تحميل ملفات تعريف حمل أخرى", failed: "تعذّر تحميل ملفات تعريف حمل أخرى. لم تتغيّر ملفات التعريف المعروضة.", again: "إعادة المحاولة", reset: "حُدِّثت القائمة وتبدأ من جديد بأول ملف تعريف حمل." },
};
const open = async (lang, extra = "") => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/release-evidence?lang=${lang}&view=load${extra}`);
  await page.locator("article.p05-profile").first().waitFor();
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => !document.querySelector(".p05-live[aria-busy=true]"));
};
const cards = () => page.locator("article.p05-profile");
const cardId = (id) => "p05-profile-" + id.replace(/[^A-Za-z0-9]/g, (c) => "_" + c.codePointAt(0).toString(16) + "_");

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
  const real = served?.load?.profiles?.rows ?? [];
  notes.push(`served: ${real.length} profiles, total ${served?.load?.profiles?.total}, profiles_next ${JSON.stringify(served?.load?.profiles_next)}`);
  check("the real service lists the sample's profiles whole, numbered in the order recorded, and says nothing further exists", real.length > 0 && served.load.profiles.total === real.length && served.load.profiles_next === "" && real.every((p, i) => p.number === i + 1));
  check("with every profile shown there is no count and no button", (await page.locator(".p05-profiles-count, .p05-profiles-more").count()) === 0 && (await cards().count()) === real.length);
  const further = await page.evaluate(async ([csrf, target]) => {
    const head = { "X-CSRF-Token": csrf, "X-Request-Id": "qa-profiles" };
    const a = await fetch(`/api/v1/release-evidence/profiles?limit=1&after=1`, { headers: head, credentials: "include" });
    const b = await fetch(`/api/v1/release-evidence/profiles?target=${encodeURIComponent(target)}`, { headers: head, credentials: "include" });
    return { a: { status: a.status, body: await a.json().catch(() => null) }, b: { status: b.status, body: await b.json().catch(() => null) } };
  }, [token, real[real.length - 1].target]);
  check("the real service answers the profiles read after a cursor and by target", further.a.status === 200 && further.a.body.order === "recorded" && further.a.body.total === real.length &&
    (real.length > 1 ? further.a.body.profiles[0].id === real[1].id : further.a.body.profiles.length === 0) && further.b.status === 200 && further.b.body.profiles.length === 1 &&
    further.b.body.profiles[0].id === real[real.length - 1].id, JSON.stringify(further).slice(0, 300));
  const template = real[0];
  const made = (n) => ({ ...template, number: n, id: `routed-profile-${n}`, target: `perf-profile:routed-profile-${n}`, retire: null, can_retire: false, retire_blocked: "no_eligible_approver" });
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
        body.load = { ...body.load, profiles: { total: 70, more: true, rows: body.load.profiles.rows }, profiles_next: String(base0) };
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "release-evidence/profiles"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.toString());
        const answer = (rows, next) => route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "recorded", q: "", profiles: rows, next, matching: q.has("target") ? rows.length : 70, total: 70 } });
        if (q.has("target")) return answer(q.get("target").split(",").map((t) => made(Number(t.replace("perf-profile:routed-profile-", "")))), "");
        if (q.get("after") === "56" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        return q.get("after") === String(base0) ? answer(span(base0 + 1, 56), "56") : answer(span(57, 70), "");
      });
      await open(lang);
      check(`${at} the first profiles are listed with the count of all and the button`, (await cards().count()) === base0 && (await page.locator(".p05-profiles-count").innerText()).trim() === copy[lang].shown(base0, 70) && (await page.locator(".p05-profiles-more-button").innerText()).trim() === copy[lang].more);
      const more = page.locator(".p05-profiles-more-button");
      check(`${at} the button is at least 44 high`, (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll("article.p05-profile").length === 56);
      check(`${at} the next profiles follow in the order recorded`, asked[0] === `limit=50&after=${base0}` && (await page.locator(".p05-profiles-count").innerText()).trim() === copy[lang].shown(56, 70) && (await cards().nth(base0).getAttribute("id")) === cardId(`routed-profile-${base0 + 1}`));
      await page.keyboard.press("Enter");
      await page.locator(".p05-profiles-more [role=alert]").waitFor();
      const alert = await page.locator(".p05-profiles-more").innerText();
      check(`${at} a failed read says so, keeps the profiles and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await cards().count()) === 56 && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await page.locator(".p05-profiles-more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-profiles-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll("article.p05-profile").length === 70);
      check(`${at} at the end all seventy are listed and the count and button are gone`, (await page.locator(".p05-profiles-more").count()) === 0 && (await page.locator(".p05-profiles-count").count()) === 0);
      const ids = await cards().evaluateAll((els) => els.map((el) => el.id));
      check(`${at} no profile is listed twice`, new Set(ids).size === 70, ids.slice(0, 3).join());
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (width === 1440) {
        again += 1;
        await page.locator(".p05-reload").first().click();
        await page.waitForFunction((n) => document.querySelectorAll("article.p05-profile").length === n, base0);
        check(`${at} after the screen is read again the list starts from the main read and says so`, (await page.locator(".p05-profiles-reset").innerText()).trim() === copy[lang].reset && (await page.locator(".p05-vanished").count()) === 0);
        if (args.shots) { await page.locator(".p05-profiles-count").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-profiles-reset-${lang}-${width}.png`, fullPage: false }); }
        asked.length = 0;
        await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/release-evidence?lang=${lang}&view=load&profile=${encodeURIComponent("a,b")}`);
        await page.locator("article.p05-profile").first().waitFor();
        await page.waitForLoadState("networkidle");
        check(`${at} a link with a malformed profile is ignored: nothing is asked by target and the screen is not failed`, asked.length === 0 && (await page.locator(".p05-failure").count()) === 0 && (await cards().count()) === base0, asked.join(" "));
        asked.length = 0;
        await open(lang, "&profile=routed-profile-65");
        await page.locator("#" + cardId("routed-profile-65")).waitFor();
        check(`${at} a link to a profile outside the main read reads it by target and selects it`, asked.includes("target=perf-profile%3Arouted-profile-65") && (await page.locator("#" + cardId("routed-profile-65")).getAttribute("class")).includes("live-selected"), asked.join(" "));
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
