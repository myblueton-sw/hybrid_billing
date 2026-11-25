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
  en: { shown: (n, m) => `Showing ${n} of ${m} released holds, newest first.`, all: (m) => `Showing all ${m} released holds, newest first.`, more: "Load older released holds", failed: "Older released holds could not be loaded. The holds already shown are unchanged.", again: "Try again", reset: "The list was refreshed. The older released holds you loaded are no longer shown.", groups: (a, b) => [`In force (${a})`, `Released (${b})`] },
  ko: { shown: (n, m) => `해제된 보존 보류 ${m}건 중 ${n}건을 최신순으로 표시합니다.`, all: (m) => `해제된 보존 보류 ${m}건을 모두 최신순으로 표시합니다.`, more: "이전 해제된 보존 보류 불러오기", failed: "이전 해제된 보존 보류를 불러오지 못했습니다. 이미 표시된 보존 보류는 그대로입니다.", again: "다시 시도", reset: "목록을 새로 고쳤습니다. 불러온 이전 해제된 보존 보류는 더 이상 표시되지 않습니다.", groups: (a, b) => [`유지 중(${a})`, `해제됨(${b})`] },
  ar: { shown: (n, m) => `تُعرض تعليقات الحذف المرفوعة: ${n} من أصل ${m}، الأحدث أولًا.`, all: (m) => `تُعرض جميع تعليقات الحذف المرفوعة (${m})، الأحدث أولًا.`, more: "تحميل تعليقات حذف مرفوعة أقدم", failed: "تعذّر تحميل تعليقات الحذف المرفوعة الأقدم. لم تتغيّر تعليقات الحذف المعروضة.", again: "إعادة المحاولة", reset: "حُدِّثت القائمة. لم تعد تعليقات الحذف المرفوعة الأقدم التي حمّلتها معروضة.", groups: (a, b) => [`سارية (${a})`, `مرفوعة (${b})`] },
};
const open = async (lang) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/retention?lang=${lang}&view=holds`);
  await page.locator(".p04-required").waitFor();
  await page.waitForLoadState("networkidle");
};
const cards = () => page.locator("article.p04-hold");

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
  const real = served?.holds?.rows ?? [];
  const realReleased = real.filter((h) => h.released);
  notes.push(`served: ${real.length} holds (${realReleased.length} released), released ${JSON.stringify(served?.holds?.released)}, total ${served?.holds?.total}`);
  check("the real service lists the sample's holds whole, numbered, and counts the released ones", real.length > 0 && served.holds.total === real.length && served.holds.released.total === realReleased.length && served.holds.released.next === "" && real.every((h) => Number.isInteger(h.number)));
  check("with every hold shown there is no count and no button", (await page.locator(".p04-held-count, .p04-held-more").count()) === 0 && (await cards().count()) === real.length);
  const further = await page.evaluate(async ([csrf]) => {
    const r = await fetch(`/api/v1/retention/holds?limit=5`, { headers: { "X-CSRF-Token": csrf, "X-Request-Id": "qa-holds" }, credentials: "include" });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, [token]);
  check("the real service answers the released holds read", further.status === 200 && further.body.total === realReleased.length && further.body.next === "" && further.body.order === "newest_first" &&
    further.body.holds.map((h) => h.id).join() === realReleased.map((h) => h.id).join(), JSON.stringify(further.body).slice(0, 300));
  const live = real.filter((h) => !h.released);
  const template = real[0];
  const made = (n) => ({ ...template, number: n, id: `routed-hold-${n}`, target: `retention-hold:routed-hold-${n}`, rationale: `Routed hold ${n}`, released: true, release: null, can_propose_release: false, release_blocked: "released", outcomes: [], outcomes_more: false });
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
        body.holds = { ...body.holds, total: live.length + 120, more: true, rows: [...down(1120, 1071), ...live], released: { total: 120, next: "1071" } };
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "retention/holds"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.get("after"));
        if (q.get("after") === "1021" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        const body = q.get("after") === "1071" ? { holds: down(1070, 1021), next: "1021" } : { holds: down(1020, 1001), next: "" };
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "newest_first", q: "", matching: 120, total: 120, ...body } });
      });
      await open(lang);
      await page.locator(".p04-held-count").waitFor();
      check(`${at} every live hold and the newest fifty released ones are listed with the count of released`, (await cards().count()) === 50 + live.length && (await page.locator(".p04-held-count").innerText()).trim() === copy[lang].shown(50, 120), await page.locator(".p04-held-count").innerText());
      const heads = async () => [(await page.locator("#p04-held-in-force").innerText()).trim(), (await page.locator("#p04-held-released").innerText()).trim()];
      const order = await page.evaluate(() => { const a = document.querySelector(".p04-held-in-force"), b = document.querySelector(".p04-held-released"); return { first: !!(a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING), inForce: a.querySelectorAll("article.p04-hold").length, released: b.querySelectorAll("article.p04-hold").length, countIn: !!b.querySelector(".p04-held-count"), moreIn: !!b.querySelector(".p04-held-more") }; });
      check(`${at} holds in force come first in their own group, the released ones follow with their count and button`, (await heads()).join("|") === copy[lang].groups(live.length, 50).join("|") && order.first && order.inForce === live.length && order.released === 50 && order.countIn && order.moreIn, JSON.stringify({ heads: await heads(), order }));
      const more = page.locator(".p04-held-more-button");
      check(`${at} the button is named and at least 44 high`, (await more.innerText()).trim() === copy[lang].more && (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      if (args.shots) { await page.locator(".p04-held-count").evaluate((el) => el.scrollIntoView({ block: "start" })); await page.screenshot({ path: `${args.shots}/p04-holds-cut-${lang}-${width}.png`, fullPage: false }); }
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction((n) => document.querySelectorAll("article.p04-hold").length === n, 100 + live.length);
      check(`${at} the next fifty released holds follow`, asked[0] === "1071" && (await page.locator(".p04-held-count").innerText()).trim() === copy[lang].shown(100, 120));
      await page.keyboard.press("Enter");
      await page.locator(".p04-held-more [role=alert]").waitFor();
      const alert = await page.locator(".p04-held-more").innerText();
      check(`${at} a failed read says so, keeps the holds and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await cards().count()) === 100 + live.length && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await page.locator(".p04-held-more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-holds-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction((n) => document.querySelectorAll("article.p04-hold").length === n, 120 + live.length);
      check(`${at} at the end all are listed, the button is gone and focus is on the count`, (await page.locator(".p04-held-more").count()) === 0 && (await page.locator(".p04-held-count").innerText()).trim() === copy[lang].all(120) && await page.locator(".p04-held-count").evaluate((el) => el === document.activeElement));
      check(`${at} the released group's heading counts all loaded`, (await heads())[1] === copy[lang].groups(live.length, 120)[1]);
      const ids = await cards().evaluateAll((els) => els.map((el) => el.id));
      check(`${at} no hold is listed twice`, new Set(ids).size === ids.length && ids.length === 120 + live.length, ids.slice(0, 3).join());
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (lang === "en" && width === 1440) {
        again = 1;
        await page.locator(".p04-reload").first().click();
        await page.waitForFunction((n) => document.querySelectorAll("article.p04-hold").length === n, 50 + live.length);
        check(`${at} after the screen is read again the list starts from the main read and says so`, (await page.locator(".p04-held-reset").innerText()).trim() === copy[lang].reset);
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
