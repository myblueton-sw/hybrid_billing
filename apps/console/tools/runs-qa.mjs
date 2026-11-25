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
  en: { shown: (n, m) => `Showing ${n} of ${m} stopped or finished deletion runs, newest first.`, all: (m) => `Showing all ${m} stopped or finished deletion runs, newest first.`, more: "Load older runs", failed: "Older deletion runs could not be loaded. The runs already shown are unchanged.", again: "Try again", reset: "The list was refreshed. The older deletion runs you loaded are no longer shown.", groups: (a, b) => [`Running (${a})`, `Stopped or finished (${b})`] },
  ko: { shown: (n, m) => `중지되거나 종료된 삭제 작업 ${m}건 중 ${n}건을 최신순으로 표시합니다.`, all: (m) => `중지되거나 종료된 삭제 작업 ${m}건을 모두 최신순으로 표시합니다.`, more: "이전 삭제 작업 불러오기", failed: "이전 삭제 작업을 불러오지 못했습니다. 이미 표시된 삭제 작업은 그대로입니다.", again: "다시 시도", reset: "목록을 새로 고쳤습니다. 불러온 이전 삭제 작업은 더 이상 표시되지 않습니다.", groups: (a, b) => [`실행 중(${a})`, `중지·종료(${b})`] },
  ar: { shown: (n, m) => `تُعرض عمليات الحذف الموقوفة أو المنتهية: ${n} من أصل ${m}، الأحدث أولًا.`, all: (m) => `تُعرض جميع عمليات الحذف الموقوفة أو المنتهية (${m})، الأحدث أولًا.`, more: "تحميل عمليات حذف أقدم", failed: "تعذّر تحميل عمليات الحذف الأقدم. لم تتغيّر عمليات الحذف المعروضة.", again: "إعادة المحاولة", reset: "حُدِّثت القائمة. لم تعد عمليات الحذف الأقدم التي حمّلتها معروضة.", groups: (a, b) => [`قيد التشغيل (${a})`, `موقوفة أو منتهية (${b})`] },
};
const open = async (lang) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/retention?lang=${lang}&view=deletion`);
  await page.locator(".p04-unconnected").waitFor();
  await page.waitForLoadState("networkidle");
};
const cards = () => page.locator("article.p04-run");

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
  const real = served?.deletion?.rows ?? [];
  const realFinished = real.filter((r) => r.state !== "running");
  const live = real.filter((r) => r.state === "running");
  notes.push(`served: ${real.length} runs (${realFinished.length} finished), finished ${JSON.stringify(served?.deletion?.finished)}, total ${served?.deletion?.total}`);
  check("the real service lists the sample's runs whole, numbered, and counts the finished ones", real.length > 0 && served.deletion.total === real.length && served.deletion.finished.total === realFinished.length && served.deletion.finished.next === "" && real.every((r) => Number.isInteger(r.number)));
  check("with every run shown there is no count and no button, and the two groups hold the runs", (await page.locator(".p04-runs-count, .p04-runs-more").count()) === 0 && (await cards().count()) === real.length &&
    (await page.locator(".p04-runs-running article.p04-run").count()) === live.length && (await page.locator(".p04-runs-finished article.p04-run").count()) === realFinished.length);
  const further = await page.evaluate(async ([csrf]) => {
    const r = await fetch(`/api/v1/retention/runs?limit=5`, { headers: { "X-CSRF-Token": csrf, "X-Request-Id": "qa-runs" }, credentials: "include" });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, [token]);
  check("the real service answers the finished runs read", further.status === 200 && further.body.total === realFinished.length && further.body.next === "" && further.body.order === "newest_first" &&
    further.body.runs.map((r) => r.run.target).join() === realFinished.map((r) => r.run.target).join(), JSON.stringify(further.body).slice(0, 300));
  const template = realFinished[0] ?? real[0];
  const made = (n) => ({ ...template, number: n, run: { target: `retention-run-${n}`, label: `run/${n}` }, state: "stopped", can_stop: false, stop_blocked: "stopped", reason: `Routed run ${n}` });
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
        body.deletion = { ...body.deletion, total: live.length + 120, more: true, rows: [...down(1120, 1071), ...live], finished: { total: 120, next: "1071" } };
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "retention/runs"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.get("after"));
        if (q.get("after") === "1021" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        const body = q.get("after") === "1071" ? { runs: down(1070, 1021), next: "1021" } : { runs: down(1020, 1001), next: "" };
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "newest_first", q: "", matching: 120, total: 120, ...body } });
      });
      await open(lang);
      await page.locator(".p04-runs-count").waitFor();
      check(`${at} every running run and the newest fifty finished ones are listed with the count of finished`, (await cards().count()) === 50 + live.length && (await page.locator(".p04-runs-count").innerText()).trim() === copy[lang].shown(50, 120), await page.locator(".p04-runs-count").innerText());
      const heads = async () => [(await page.locator("#p04-runs-running").innerText()).trim(), (await page.locator("#p04-runs-finished").innerText()).trim()];
      const order = await page.evaluate(() => { const a = document.querySelector(".p04-runs-running"), b = document.querySelector(".p04-runs-finished"); return { first: !!(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING), running: a.querySelectorAll("article.p04-run").length, finished: b.querySelectorAll("article.p04-run").length, countIn: !!b.querySelector(".p04-runs-count"), moreIn: !!b.querySelector(".p04-runs-more") }; });
      check(`${at} running runs come first in their own group, the finished ones follow with their count and button`, (await heads()).join("|") === copy[lang].groups(live.length, 50).join("|") && order.first && order.running === live.length && order.finished === 50 && order.countIn && order.moreIn, JSON.stringify({ heads: await heads(), order }));
      const more = page.locator(".p04-runs-more-button");
      check(`${at} the button is named and at least 44 high`, (await more.innerText()).trim() === copy[lang].more && (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      if (args.shots) { await page.locator(".p04-runs-count").evaluate((el) => el.scrollIntoView({ block: "start" })); await page.screenshot({ path: `${args.shots}/p04-runs-cut-${lang}-${width}.png`, fullPage: false }); }
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction((n) => document.querySelectorAll("article.p04-run").length === n, 100 + live.length);
      check(`${at} the next fifty finished runs follow`, asked[0] === "1071" && (await page.locator(".p04-runs-count").innerText()).trim() === copy[lang].shown(100, 120));
      await page.keyboard.press("Enter");
      await page.locator(".p04-runs-more [role=alert]").waitFor();
      const alert = await page.locator(".p04-runs-more").innerText();
      check(`${at} a failed read says so, keeps the runs and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await cards().count()) === 100 + live.length && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await page.locator(".p04-runs-more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-runs-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction((n) => document.querySelectorAll("article.p04-run").length === n, 120 + live.length);
      check(`${at} at the end all are listed, the button is gone and focus is on the count`, (await page.locator(".p04-runs-more").count()) === 0 && (await page.locator(".p04-runs-count").innerText()).trim() === copy[lang].all(120) && await page.locator(".p04-runs-count").evaluate((el) => el === document.activeElement));
      check(`${at} the finished group's heading counts all loaded`, (await heads())[1] === copy[lang].groups(live.length, 120)[1]);
      const ids = await cards().evaluateAll((els) => els.map((el) => el.id));
      check(`${at} no run is listed twice`, new Set(ids).size === ids.length && ids.length === 120 + live.length, ids.slice(0, 3).join());
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (lang === "en" && width === 1440) {
        again = 1;
        await page.locator(".p04-reload").first().click();
        await page.waitForFunction((n) => document.querySelectorAll("article.p04-run").length === n, 50 + live.length);
        check(`${at} after the screen is read again the list starts from the main read and says so`, (await page.locator(".p04-runs-reset").innerText()).trim() === copy[lang].reset);
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
