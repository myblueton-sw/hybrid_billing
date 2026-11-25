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
  en: { shown: (n, m) => `Showing the newest ${n} of ${m} locks, newest first.`, all: (m) => `Showing all ${m} locks, newest first.`, more: "Load older locks", failed: "Older locks could not be loaded. The locks already shown are unchanged.", again: "Try again", reset: "The log was refreshed and starts again from the newest lock." },
  ko: { shown: (n, m) => `잠금 ${m}건 중 최근 ${n}건을 최신순으로 표시합니다.`, all: (m) => `잠금 ${m}건을 모두 최신순으로 표시합니다.`, more: "이전 잠금 불러오기", failed: "이전 잠금을 불러오지 못했습니다. 이미 표시된 잠금은 그대로입니다.", again: "다시 시도", reset: "잠금 기록을 새로 고쳐 최신 잠금부터 다시 표시합니다." },
  ar: { shown: (n, m) => `تُعرض أحدث الأقفال: ${n} من أصل ${m}، الأحدث أولًا.`, all: (m) => `تُعرض جميع الأقفال (${m})، الأحدث أولًا.`, more: "تحميل أقفال أقدم", failed: "تعذّر تحميل الأقفال الأقدم. لم تتغيّر الأقفال المعروضة.", again: "إعادة المحاولة", reset: "حُدِّث سجل القفل ويبدأ من جديد بأحدث قفل." },
};
const standby = "sample-standby";
const open = async (lang) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/recovery?lang=${lang}&view=isolation`);
  await page.locator(".p03-locklog").first().waitFor();
  await page.waitForLoadState("networkidle");
};

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
  const seen = async (r) => { if (ends(r.url(), "recovery") && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); } };
  page.on("response", seen);
  await open("en");
  page.off("response", seen);
  const realRows = served?.isolation?.rows ?? [];
  const real = realRows.find((r) => r.installation === standby);
  notes.push(`served: ${realRows.length} installations; standby locks ${real?.locks?.length}, lock_history ${JSON.stringify(real?.lock_history)}`);
  check("the real service gives the standby's locks whole with their count and numbers", !!real && real.lock_history.total === real.locks.length && real.lock_history.next === "" && real.locks.every((x, i) => x.number === i + 1));
  check("with the whole log shown there is no count and no button", (await page.locator(".p03-lock-count, .p03-lock-more").count()) === 0);
  const further = await page.evaluate(async ([csrf, id]) => {
    const r = await fetch(`/api/v1/recovery/locks?installation=${id}&limit=5`, { headers: { "X-CSRF-Token": csrf, "X-Request-Id": "qa-lock-log" }, credentials: "include" });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, [token, standby]);
  check("the real service answers the locks read for the standby, newest first", further.status === 200 && further.body.total === real.locks.length && further.body.next === "" && further.body.order === "newest_first" &&
    further.body.locks.map((x) => x.number).join() === [...real.locks].reverse().map((x) => x.number).join(), JSON.stringify(further.body).slice(0, 300));
  const template = real.locks[0];
  const made = (n) => ({ ...template, number: n, rationale: `Routed lock ${n}`, at: new Date(Date.UTC(2026, 10, 1, 0, n)).toISOString().replace(".000Z", "Z") });
  const up = (from, to) => Array.from({ length: to - from + 1 }, (_, i) => made(from + i));
  const down = (from, to) => Array.from({ length: from - to + 1 }, (_, i) => made(from - i));
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1, again = 0; const asked = [];
      await page.route((url) => ends(url.toString(), "recovery"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.revision = body.revision + (again ? "-again" + again : "");
        for (const r of body.isolation.rows) if (r.installation === standby) { r.locks = up(71, 120); r.lock_history = { total: 120, next: "71" }; }
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "recovery/locks"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.get("installation") + ":" + q.get("after"));
        if (q.get("after") === "21" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        const body = q.get("after") === "71" ? { locks: down(70, 21), next: "21" } : { locks: down(20, 1), next: "" };
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", installation: standby, order: "newest_first", q: "", matching: 120, total: 120, ...body } });
      });
      await open(lang);
      const log = page.locator(".p03-locklog").filter({ has: page.locator(".p03-lock-more") });
      const rows = () => log.locator(".p03-lock-row");
      await log.locator(".p03-lock-count").waitFor();
      check(`${at} the newest fifty locks are listed newest first with the count of all`, (await rows().count()) === 50 && (await log.locator(".p03-lock-count").innerText()).trim() === copy[lang].shown(50, 120) && (await rows().first().innerText()).includes("Routed lock 120"), await rows().first().innerText());
      check(`${at} only the installation with older locks offers the button`, (await page.locator(".p03-lock-more").count()) === 1);
      const more = log.locator(".p03-lock-more-button");
      check(`${at} the button is named and at least 44 high`, (await more.innerText()).trim() === copy[lang].more && (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      if (args.shots) { await log.locator("h4").evaluate((el) => el.scrollIntoView({ block: "start" })); await page.screenshot({ path: `${args.shots}/p03-locks-cut-${lang}-${width}.png`, fullPage: false }); }
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll(".p03-lock-row").length >= 100);
      check(`${at} the next fifty follow, older ones below`, asked[0] === `${standby}:71` && (await rows().count()) === 100 && (await log.locator(".p03-lock-count").innerText()).trim() === copy[lang].shown(100, 120) && (await rows().nth(50).innerText()).includes("Routed lock 70"));
      await page.keyboard.press("Enter");
      await log.locator(".p03-lock-more [role=alert]").waitFor();
      const alert = await log.locator(".p03-lock-more").innerText();
      check(`${at} a failed read says so, keeps the rows and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await rows().count()) === 100 && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await log.locator(".p03-lock-more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p03-locks-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll(".p03-lock-row").length >= 120);
      const whole = page.locator(".p03-locklog").filter({ has: page.locator(".p03-lock-count") });
      check(`${at} at the end all are listed, the button is gone and focus is on the count`, (await page.locator(".p03-lock-more").count()) === 0 && (await whole.locator(".p03-lock-count").innerText()).trim() === copy[lang].all(120) && await whole.locator(".p03-lock-count").evaluate((el) => el === document.activeElement));
      const numbers = await whole.locator(".p03-lock-row").evaluateAll((els) => els.map((el) => el.innerText.match(/Routed lock (\d+)/)?.[1]));
      check(`${at} the rows run from the newest to the oldest without a gap or a repeat`, numbers.length === 120 && numbers.every((n, i) => Number(n) === 120 - i), numbers.slice(0, 5).join());
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (lang === "en" && width === 1440) {
        again = 1;
        await page.locator(".p03-reload").first().click();
        await page.waitForFunction(() => document.querySelectorAll(".p03-lock-row").length <= 60);
        const fresh = page.locator(".p03-locklog").filter({ has: page.locator(".p03-lock-more") });
        check(`${at} after the screen is read again the log starts from the newest fifty and says so`, (await fresh.locator(".p03-lock-row").count()) === 50 && (await fresh.locator(".p03-lock-reset").innerText()).trim() === copy[lang].reset);
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
