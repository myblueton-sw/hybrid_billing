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
  en: { shown: (n, m) => `Showing the newest ${n} of ${m} actions, newest first.`, all: (m) => `Showing all ${m} actions, newest first.`, more: "Load older actions", failed: "Older actions could not be loaded. The actions already shown are unchanged.", again: "Try again", reset: "The history was refreshed and starts again from the newest action." },
  ko: { shown: (n, m) => `작업 ${m}건 중 최근 ${n}건을 최신순으로 표시합니다.`, all: (m) => `작업 ${m}건을 모두 최신순으로 표시합니다.`, more: "이전 작업 불러오기", failed: "이전 작업을 불러오지 못했습니다. 이미 표시된 작업은 그대로입니다.", again: "다시 시도", reset: "이력을 새로 고쳐 최신 작업부터 다시 표시합니다." },
  ar: { shown: (n, m) => `تُعرض أحدث الإجراءات: ${n} من أصل ${m}، الأحدث أولًا.`, all: (m) => `تُعرض جميع الإجراءات (${m})، الأحدث أولًا.`, more: "تحميل إجراءات أقدم", failed: "تعذّر تحميل الإجراءات الأقدم. لم تتغيّر الإجراءات المعروضة.", again: "إعادة المحاولة", reset: "حُدِّث السجل ويبدأ من جديد بأحدث إجراء." },
};
const subject = "sample-erp-invoices";
const open = async (lang) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/integrations?lang=${lang}&view=channels`);
  await page.locator(".p02-fold").first().waitFor();
  await page.waitForLoadState("networkidle");
};
const unfold = async () => { for (const d of await page.locator("details.p02-fold").all()) await d.evaluate((el) => { el.open = true; }); };

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
  const seen = async (r) => { if (ends(r.url(), "integrations") && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); } };
  page.on("response", seen);
  await open("en");
  page.off("response", seen);
  const subs = served?.channels?.subscriptions ?? [];
  const real = subs.find((r) => r.id === subject);
  notes.push(`served: ${subs.length} subscriptions; ${subject} history ${real?.history?.length}, history_facts ${JSON.stringify(real?.history_facts)}`);
  check("the real service gives the subscription's history whole with its count and numbers", !!real && real.history.length > 0 && real.history_facts.total === real.history.length && real.history_facts.next === "" && real.history.every((x, i) => x.number === i + 1));
  await unfold();
  check("with the whole history shown there is no count and no button", (await page.locator(".p02-history-count, .p02-history-more").count()) === 0);
  const past = () => page.evaluate(() => [...document.querySelectorAll(".p02-live *")].filter((el) => el.getBoundingClientRect().right > document.documentElement.clientWidth + 1 || el.getBoundingClientRect().left < -1)
    .map((el) => el.tagName + "." + el.className + " in " + (el.closest("table")?.className ?? "") + " of " + (el.closest(".p02-history-more, .p02-history-count, details")?.className ?? "")).filter((x, i, all) => all.indexOf(x) === i));
  const before = { ...(await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }))), who: await past() };
  notes.push(`sample data, folds open, 1440: page ${before.page} view ${before.view} ${JSON.stringify(before.who)}`);
  check("the overflow recorded as HYB-360 is at most four pixels and comes from a sign-in name in a table of the fold", before.page - before.view <= 4 && before.who.every((x) => x.startsWith("BDI.p02-login in p02-table")), JSON.stringify(before));
  const further = await page.evaluate(async ([csrf, id]) => {
    const r = await fetch(`/api/v1/integrations/subscription-history?subscription=${id}&limit=5`, { headers: { "X-CSRF-Token": csrf, "X-Request-Id": "qa-history" }, credentials: "include" });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, [token, subject]);
  check("the real service answers the history read for the subscription, newest first", further.status === 200 && further.body.total === real.history.length && further.body.next === "" && further.body.order === "newest_first" &&
    further.body.history.map((x) => x.number).join() === [...real.history].reverse().map((x) => x.number).join(), JSON.stringify(further.body).slice(0, 300));
  const template = real.history[0];
  const made = (n) => ({ ...template, number: n, rationale: `Routed action ${n}` });
  const up = (from, to) => Array.from({ length: to - from + 1 }, (_, i) => made(from + i));
  const down = (from, to) => Array.from({ length: from - to + 1 }, (_, i) => made(from - i));
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1, again = 0; const asked = [];
      await page.route((url) => ends(url.toString(), "integrations"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.revision = body.revision + (again ? "-again" + again : "");
        for (const r of body.channels.subscriptions) if (r.id === subject) { r.history = up(71, 120); r.history_facts = { total: 120, next: "71" }; }
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "integrations/subscription-history"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.get("subscription") + ":" + q.get("after"));
        if (q.get("after") === "21" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        const body = q.get("after") === "71" ? { history: down(70, 21), next: "21" } : { history: down(20, 1), next: "" };
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", subscription: subject, order: "newest_first", q: "", matching: 120, total: 120, ...body } });
      });
      await open(lang);
      await unfold();
      const log = page.locator("details.p02-fold").filter({ has: page.locator(".p02-history-more") });
      const rows = () => log.locator(".p02-history-row");
      await log.locator(".p02-history-count").waitFor();
      check(`${at} the newest fifty actions are listed newest first with the count of all`, (await rows().count()) === 50 && (await log.locator(".p02-history-count").innerText()).trim() === copy[lang].shown(50, 120) && (await rows().first().innerText()).includes("Routed action 120"), await rows().first().innerText());
      check(`${at} only the subscription with older actions offers the button`, (await page.locator(".p02-history-more").count()) === 1);
      const more = log.locator(".p02-history-more-button");
      check(`${at} the button is named and at least 44 high`, (await more.innerText()).trim() === copy[lang].more && (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      if (args.shots) { await log.locator("h4").first().evaluate((el) => el.scrollIntoView({ block: "start" })); await page.screenshot({ path: `${args.shots}/p02-history-cut-${lang}-${width}.png`, fullPage: false }); }
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll(".p02-history-row").length >= 100);
      check(`${at} the next fifty follow, older ones below`, asked[0] === `${subject}:71` && (await rows().count()) === 100 && (await log.locator(".p02-history-count").innerText()).trim() === copy[lang].shown(100, 120) && (await rows().nth(50).innerText()).includes("Routed action 70"));
      await page.keyboard.press("Enter");
      await log.locator(".p02-history-more [role=alert]").waitFor();
      const alert = await log.locator(".p02-history-more").innerText();
      check(`${at} a failed read says so, keeps the rows and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await rows().count()) === 100 && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await log.locator(".p02-history-more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p02-history-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll(".p02-history-row").length >= 120);
      const whole = page.locator("details.p02-fold").filter({ has: page.locator(".p02-history-count") });
      check(`${at} at the end all are listed, the button is gone and focus is on the count`, (await page.locator(".p02-history-more").count()) === 0 && (await whole.locator(".p02-history-count").innerText()).trim() === copy[lang].all(120) && await whole.locator(".p02-history-count").evaluate((el) => el === document.activeElement));
      const numbers = await whole.locator(".p02-history-row").evaluateAll((els) => els.map((el) => el.innerText.match(/Routed action (\d+)/)?.[1]));
      check(`${at} the rows run from the newest to the oldest without a gap or a repeat`, numbers.length === 120 && numbers.every((n, i) => Number(n) === 120 - i), numbers.slice(0, 5).join());
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      const now = await past();
      const history = await whole.evaluate((el) => { const t = el.querySelector(".p02-history-row").closest(".live-table"); const more = el.querySelector(".p02-history-count"); return { table: t.scrollWidth - t.clientWidth, count: more.getBoundingClientRect().right <= el.getBoundingClientRect().right + 1 && more.getBoundingClientRect().left >= el.getBoundingClientRect().left - 1 }; });
      check(`${at} nothing new runs past the edge: the page is no wider than with the sample's own history, the same kind of element overflows, and the count stays inside the fold`,
        m.page <= Math.max(m.view + 1, width === 1440 ? before.page : 0) && now.every((x) => x.startsWith("BDI.p02-login in p02-table")) && history.count, JSON.stringify({ m, now, history }));
      if (lang === "en" && width === 1440) {
        again = 1;
        await page.locator(".p02-reload").first().click();
        await page.waitForFunction(() => document.querySelectorAll(".p02-history-row").length <= 60);
        await unfold();
        const fresh = page.locator("details.p02-fold").filter({ has: page.locator(".p02-history-more") });
        check(`${at} after the screen is read again the history starts from the newest fifty and says so`, (await fresh.locator(".p02-history-row").count()) === 50 && (await fresh.locator(".p02-history-reset").innerText()).trim() === copy[lang].reset);
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
