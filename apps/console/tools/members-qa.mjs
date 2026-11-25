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
  en: { shown: (n, m) => `Showing ${n} of ${m} items, sorted by item.`, more: "Load more items", failed: "More items could not be loaded. The items already shown are unchanged.", again: "Try again", reset: "The list was refreshed and starts again from the first item." },
  ko: { shown: (n, m) => `${m}건 중 ${n}건을 항목 순으로 표시합니다.`, more: "항목 더 불러오기", failed: "항목을 더 불러오지 못했습니다. 이미 표시된 항목은 그대로입니다.", again: "다시 시도", reset: "목록을 새로 고쳐 첫 항목부터 다시 표시합니다." },
  ar: { shown: (n, m) => `تُعرض ${n} من أصل ${m} عنصرًا، مرتبةً حسب العنصر.`, more: "تحميل عناصر أخرى", failed: "تعذّر تحميل عناصر أخرى. لم تتغيّر العناصر المعروضة.", again: "إعادة المحاولة", reset: "حُدِّثت القائمة وتبدأ من جديد بأول عنصر." },
};
const open = async (lang, view) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/retention?lang=${lang}&view=${view}`);
  await page.locator(view === "deletion" ? ".p04-unconnected" : ".p04-required").first().waitFor();
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => !document.querySelector(".p04-live[aria-busy=true]"));
};
const pad = (n) => String(n).padStart(3, "0");

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
  await open("en", "deletion");
  page.off("response", seen);
  const run = (served?.deletion?.rows ?? []).find((r) => r.artifacts.rows.length > 0);
  const hold = (served?.holds?.rows ?? []).find((h) => h.outcomes.length > 0);
  notes.push(`served: run ${run?.run?.label} with ${run?.artifacts?.rows?.length} items, next ${JSON.stringify(run?.artifacts?.next)}; hold ${hold?.id} with ${hold?.outcomes?.length} outcomes, next ${JSON.stringify(hold?.outcomes_next)}`);
  check("the real service lists the sample run's items and the sample hold's outcomes whole and says nothing further exists", !!run && !!hold && run.artifacts.next === "" && run.artifacts.more === false && hold.outcomes_next === "" && hold.outcomes_more === false);
  check("with every item shown there is no count and no button", (await page.locator(".p04-run-items-count, .p04-run-items-more").count()) === 0);
  const further = await page.evaluate(async ([csrf, target, id]) => {
    const head = { "X-CSRF-Token": csrf, "X-Request-Id": "qa-members" };
    const get = async (u) => { const r = await fetch(u, { headers: head, credentials: "include" }); return { status: r.status, body: await r.json().catch(() => null) }; };
    return { a: await get(`/api/v1/retention/run-artifacts?run=${target}&limit=1`), b: await get(`/api/v1/retention/hold-outcomes?hold=${encodeURIComponent(id)}`), c: await get(`/api/v1/retention/run-artifacts?run=retention-run-99999`) };
  }, [token, run.run.target, hold.id]);
  check("the real service answers both reads and refuses an unknown run", further.a.status === 200 && further.a.body.order === "identifier" && further.a.body.total === run.artifacts.total && further.a.body.artifacts[0].id === run.artifacts.rows[0].id &&
    further.a.body.next === (run.artifacts.total > 1 ? "1" : "") && further.b.status === 200 && further.b.body.outcomes.length === hold.outcomes.length && further.b.body.outcomes[0].artifact === hold.outcomes[0].artifact && further.c.status === 422, JSON.stringify(further).slice(0, 400));
  const n0 = run.artifacts.rows.length, o0 = hold.outcomes.length;
  const item = (n) => ({ id: `zz-item-${pad(n)}`, outcome: "deleted" });
  const outcome = (n) => ({ ...hold.outcomes[0], artifact: `zz-out-${pad(n)}` });
  const span = (make, from, to) => Array.from({ length: to - from + 1 }, (_, i) => make(from + i));
  const number = run.run.target.replace("retention-run-", "");
  const card = `#p04-run-${number}`;
  const items = () => page.locator(`${card} table.p04-run-items tbody tr`);
  const holdCard = `#p04-hold-${hold.id.replace(/[^A-Za-z0-9_-]/g, "_")}`;
  const outs = () => page.locator(`${holdCard} table.p04-outcomes tbody tr`);
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1, again = 0; const asked = [];
      await page.route((url) => ends(url.toString(), "retention"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.revision = body.revision + (again ? "-again" + again : "");
        body.deletion.rows = body.deletion.rows.map((r) => (r.run.target === run.run.target ? { ...r, artifacts: { ...r.artifacts, total: n0 + 70, more: true, next: String(n0) } } : r));
        body.holds.rows = body.holds.rows.map((h) => (h.id === hold.id ? { ...h, outcomes_more: true, outcomes_next: h.outcomes[h.outcomes.length - 1].artifact } : h));
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "retention/run-artifacts"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.toString());
        const answer = (list, next) => route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "identifier", run: q.get("run"), artifacts: list, next, total: n0 + 70 } });
        if (q.get("after") === String(n0 + 50) && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        return q.get("after") === String(n0) ? answer(span(item, 1, 50), String(n0 + 50)) : answer(span(item, 51, 70), "");
      });
      await page.route((url) => ends(url.toString(), "retention/hold-outcomes"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push("out " + q.toString());
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "identifier", hold: q.get("hold"), outcomes: span(outcome, 1, 30), next: "" } });
      });
      await open(lang, "deletion");
      const foot = (name) => page.locator(`${card} .p04-run-items-${name}`);
      check(`${at} the run's first items are listed with the count of all and the button`, (await items().count()) === n0 && (await foot("count").innerText()).trim() === copy[lang].shown(n0, n0 + 70) && (await foot("more-button").innerText()).trim() === copy[lang].more);
      const more = foot("more-button");
      check(`${at} the button is at least 44 high`, (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction(([c, n]) => document.querySelectorAll(`${c} table.p04-run-items tbody tr`).length === n, [card, n0 + 50]);
      check(`${at} the next items follow`, asked[0] === `run=${run.run.target}&limit=50&after=${n0}` && (await foot("count").innerText()).trim() === copy[lang].shown(n0 + 50, n0 + 70) && (await items().nth(n0).innerText()).includes("zz-item-001"), asked[0]);
      await page.keyboard.press("Enter");
      await foot("more").locator("[role=alert]").waitFor();
      const alert = await foot("more").innerText();
      check(`${at} a failed read says so, keeps the items and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await items().count()) === n0 + 50 && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await foot("more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-run-items-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction(([c, n]) => document.querySelectorAll(`${c} table.p04-run-items tbody tr`).length === n, [card, n0 + 70]);
      check(`${at} at the end all are listed and the count and button are gone`, (await foot("more").count()) === 0 && (await foot("count").count()) === 0);
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (width === 1440) {
        again += 1;
        await page.locator(".p04-reload").first().click();
        await page.waitForFunction(([c, n]) => document.querySelectorAll(`${c} table.p04-run-items tbody tr`).length === n, [card, n0]);
        check(`${at} after the screen is read again the run's list starts from the main read and says so`, (await foot("reset").innerText()).trim() === copy[lang].reset);
      }
      asked.length = 0;
      await open(lang, "holds");
      const ofoot = (name) => page.locator(`${holdCard} .p04-outcomes-${name}`);
      check(`${at} the hold's outcomes show the button and no count`, (await outs().count()) === o0 && (await ofoot("more-button").innerText()).trim() === copy[lang].more && (await ofoot("count").count()) === 0);
      await ofoot("more-button").focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction(([c, n]) => document.querySelectorAll(`${c} table.p04-outcomes tbody tr`).length === n, [holdCard, o0 + 30]);
      check(`${at} the rest of the outcomes follow and the foot is gone`, asked[0] === "out " + new URLSearchParams({ hold: hold.id, limit: "50", after: hold.outcomes[o0 - 1].artifact }).toString() && (await ofoot("more").count()) === 0, asked[0]);
      if (args.shots && width === 1440) { await page.locator(holdCard).scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-outcomes-all-${lang}-${width}.png`, fullPage: false }); }
      const m2 = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the holds view is not wider than the viewport`, m2.page <= m2.view + 1, JSON.stringify(m2));
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
