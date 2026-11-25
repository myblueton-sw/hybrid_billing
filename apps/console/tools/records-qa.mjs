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
  en: { shown: (n, m) => `Showing ${n} of the ${m} in this workspace, newest first.`, more: "Load older deletion records", failed: "Older deletion records could not be loaded. The records already shown are unchanged.", again: "Try again", reset: "The list was refreshed. The older deletion records you loaded are no longer shown." },
  ko: { shown: (n, m) => `이 업무 공간의 ${m}건 중 ${n}건을 최신순으로 표시합니다.`, more: "이전 삭제 기록 불러오기", failed: "이전 삭제 기록을 불러오지 못했습니다. 이미 표시된 삭제 기록은 그대로입니다.", again: "다시 시도", reset: "목록을 새로 고쳤습니다. 불러온 이전 삭제 기록은 더 이상 표시되지 않습니다." },
  ar: { shown: (n, m) => `تُعرض ${n} من أصل ${m} في مساحة العمل هذه، الأحدث أولًا.`, more: "تحميل سجلات حذف أقدم", failed: "تعذّر تحميل سجلات الحذف الأقدم. لم تتغيّر سجلات الحذف المعروضة.", again: "إعادة المحاولة", reset: "حُدِّثت القائمة. لم تعد سجلات الحذف الأقدم التي حمّلتها معروضة." },
};
const open = async (lang) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/retention?lang=${lang}&view=decisions`);
  await page.locator(".p04-records").waitFor();
  await page.waitForLoadState("networkidle");
};
const rows = () => page.locator("table.p04-records-table tbody tr");

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
  const seen = async (r) => { if (ends(r.url(), "retention") && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); } };
  page.on("response", seen);
  await open("en");
  page.off("response", seen);
  const real = served?.decisions?.tombstones?.rows ?? [];
  notes.push(`served: ${real.length} records, total ${served?.decisions?.tombstones?.total}, next ${JSON.stringify(served?.decisions?.tombstones?.next)}, watermark ${served?.decisions?.watermark}`);
  check("the real service lists the sample's records whole and says nothing further exists", real.length > 0 && served.decisions.tombstones.total === real.length && served.decisions.tombstones.next === "" && served.decisions.tombstones.more === false);
  check("with every record shown there is no count and no button", (await page.locator(".p04-records-count, .p04-records-more").count()) === 0 && (await rows().count()) === real.length);
  const further = await page.evaluate(async ([csrf]) => {
    const r = await fetch(`/api/v1/retention/tombstones?limit=5`, { headers: { "X-CSRF-Token": csrf, "X-Request-Id": "qa-records" }, credentials: "include" });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, [token]);
  check("the real service answers the records read", further.status === 200 && further.body.total === real.length && further.body.next === "" && further.body.order === "newest_first" && further.body.window === 1000 &&
    further.body.tombstones.map((x) => x.sequence).join() === real.map((x) => x.sequence).join(), JSON.stringify(further.body).slice(0, 300));
  const made = (n) => ({ ...real[0], sequence: n, artifact: `routed-item-${n}`, run: null, other_scope: false });
  const down = (from, to) => Array.from({ length: from - to + 1 }, (_, i) => made(from - i));
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1, again = 0; const asked = [];
      await page.route((url) => ends(url.toString(), "retention"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.revision = body.revision + (again ? "-again" + again : "");
        body.decisions = { ...body.decisions, watermark: 2120, tombstones: { window: 1000, total: 120, more: true, next: "2071", rows: down(2120, 2071) } };
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "retention/tombstones"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.get("after"));
        if (q.get("after") === "2021" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        const body = q.get("after") === "2071" ? { tombstones: down(2070, 2021), next: "2021" } : { tombstones: down(2020, 2001), next: "" };
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "newest_first", q: "", watermark: 2120, window: 1000, matching: 120, total: 120, ...body } });
      });
      await open(lang);
      await page.locator(".p04-records-count").waitFor();
      check(`${at} the newest fifty records are listed with the count of all in the workspace`, (await rows().count()) === 50 && (await page.locator(".p04-records-count").innerText()).trim() === copy[lang].shown(50, 120), await page.locator(".p04-records-count").innerText());
      const more = page.locator(".p04-records-more-button");
      check(`${at} the button is named and at least 44 high`, (await more.innerText()).trim() === copy[lang].more && (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll("table.p04-records-table tbody tr").length === 100);
      check(`${at} the next fifty records follow`, asked[0] === "2071" && (await page.locator(".p04-records-count").innerText()).trim() === copy[lang].shown(100, 120));
      await page.keyboard.press("Enter");
      await page.locator(".p04-records-more [role=alert]").waitFor();
      const alert = await page.locator(".p04-records-more").innerText();
      check(`${at} a failed read says so, keeps the records and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await rows().count()) === 100 && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await page.locator(".p04-records-more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-records-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll("table.p04-records-table tbody tr").length === 120);
      check(`${at} at the end all are listed, the button is gone and focus is on the count`, (await page.locator(".p04-records-more").count()) === 0 && (await page.locator(".p04-records-count").innerText()).trim() === copy[lang].shown(120, 120) && await page.locator(".p04-records-count").evaluate((el) => el === document.activeElement));
      const ids = await rows().evaluateAll((els) => els.map((el) => el.id));
      check(`${at} the records are newest first and none is listed twice`, new Set(ids).size === 120 && ids[0].endsWith("2120") && ids[119].endsWith("2001"), ids.slice(0, 2).join());
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (width === 1440) {
        if (args.shots) { await page.locator(".p04-records-count").evaluate((el) => el.scrollIntoView({ block: "start" })); await page.screenshot({ path: `${args.shots}/p04-records-all-${lang}-${width}.png`, fullPage: false }); }
        again += 1;
        await page.locator(".p04-reload").first().click();
        await page.waitForFunction(() => document.querySelectorAll("table.p04-records-table tbody tr").length === 50);
        check(`${at} after the screen is read again the list starts from the main read and says so`, (await page.locator(".p04-records-reset").innerText()).trim() === copy[lang].reset);
        if (args.shots) { await page.locator(".p04-records").evaluate((el) => el.scrollIntoView({ block: "start" })); await page.screenshot({ path: `${args.shots}/p04-records-reset-${lang}-${width}.png`, fullPage: false }); }
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
