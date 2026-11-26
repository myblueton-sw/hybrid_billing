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
  en: { shown: (n, m) => `Showing ${n} of ${m}.`, more: { history: "Load more recovery history", open_calls: "Load more open calls", unknown: "Load more unknown targets", pending: "Load more pending decision events" }, failed: "More could not be loaded. What is already shown is unchanged.", again: "Try again", reset: "The list was refreshed and starts again from the beginning.", moves: "This list can change while recovery proceeds. Reload to see it as it is now." },
  ko: { shown: (n, m) => `${m}건 중 ${n}건을 표시합니다.`, more: { history: "복구 이력 더 불러오기", open_calls: "열린 호출 더 불러오기", unknown: "미확인 대상 더 불러오기", pending: "대기 중인 결정 기록 더 불러오기" }, failed: "더 불러오지 못했습니다. 이미 표시된 내용은 그대로입니다.", again: "다시 시도", reset: "목록을 새로 고쳐 처음부터 다시 표시합니다.", moves: "복구가 진행되면 이 목록이 바뀔 수 있습니다. 지금 상태를 보려면 새로 고침하세요." },
  ar: { shown: (n, m) => `تُعرض ${n} من ${m}.`, more: { history: "تحميل المزيد من سجل الاسترداد", open_calls: "تحميل استدعاءات مفتوحة أخرى", unknown: "تحميل أهداف غير معروفة أخرى", pending: "تحميل أحداث قرار معلّقة أخرى" }, failed: "تعذّر تحميل المزيد. لم يتغيّر ما هو معروض.", again: "إعادة المحاولة", reset: "حُدِّثت القائمة وتبدأ من جديد من أولها.", moves: "قد تتغيّر هذه القائمة أثناء الاسترداد. أعد التحميل لرؤيتها كما هي الآن." },
};
const waits = { sets: ".p03-history", isolation: ".p03-calls", reconciliation: ".p03-target, .p03-rows-foot", resume: ".p03-events" };
const open = async (lang, view) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/recovery?lang=${lang}&view=${view}`);
  await page.locator(waits[view]).first().waitFor();
  await page.waitForLoadState("networkidle");
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
  const seen = async (r) => { if (ends(r.url(), "recovery") && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); } };
  page.on("response", seen);
  await open("en", "isolation");
  page.off("response", seen);
  const inst = "sample-primary";
  const row = (section) => (served?.[section]?.rows ?? []).find((r) => r.installation === inst) ?? {};
  const iso = row("isolation"), rec = row("reconciliation"), res = row("resume"), set = row("sets");
  notes.push(`served for ${inst}: open calls ${iso.open_calls?.length} next ${JSON.stringify(iso.open_calls_next)}; unknown ${rec.unknown_targets?.length} next ${JSON.stringify(rec.unknown_next)}; pending ${res.pending_events?.length} next ${JSON.stringify(res.pending_next)}; history ${set.history?.length} next ${JSON.stringify(set.history_next)}`);
  check("the real service lists the sample's four lists whole and says nothing further exists", iso.open_calls_next === "" && rec.unknown_next === "" && res.pending_next === "" && set.history_next === "");
  check("with every row shown there is no foot", (await page.locator(".p03-rows-foot").count()) === 0);
  const further = await page.evaluate(async ([csrf, id]) => {
    const head = { "X-CSRF-Token": csrf, "X-Request-Id": "qa-rows" };
    const get = async (u) => { const r = await fetch(u, { headers: head, credentials: "include" }); return { status: r.status, body: await r.json().catch(() => null) }; };
    const out = {};
    for (const list of ["history", "open_calls", "unknown", "pending"]) out[list] = await get(`/api/v1/recovery/rows?installation=${id}&list=${list}`);
    out.bad = await get(`/api/v1/recovery/rows?installation=never&list=history`);
    return out;
  }, [token, inst]);
  check("the real service answers the rows read for the four lists and refuses an unknown installation", further.history.status === 200 && further.history.body.order === "newest_first" && further.history.body.rows_can_move === false &&
    further.open_calls.body.rows.length === iso.open_calls.length && further.open_calls.body.rows_can_move === true && further.unknown.body.rows.length === rec.unknown_targets.length &&
    further.pending.body.rows.length === res.pending_events.length && further.bad.status === 422, JSON.stringify(further).slice(0, 400));
  const make = {
    history: (n) => ({ recovery: `zz-recovery-${pad(n)}`, manifest_digest: "e".repeat(64), generation: n, outcome: "completed", lost_count: 0 }),
    open_calls: (n) => ({ id: `zz-call-${pad(n)}`, writer: "sample-erp-writer", deadline: "2026-11-16T09:01:00Z", key: `zz-key-${pad(n)}`, series: "", number: 0 }),
    unknown: (n) => ({ key: `zz-key-${pad(Math.ceil(n / 2))}`, series: "zz-series", number: n, fenced: true, high_water: n }),
    pending: (n) => ({ sequence: 1000 + n, kind: res.pending_events?.[0]?.kind ?? "issuance", domain: res.pending_events?.[0]?.domain ?? "billing", at: "2026-11-16T09:00:00Z" }),
  };
  const span = (list, from, to) => Array.from({ length: to - from + 1 }, (_, i) => make[list](from + i));
  const lists = [
    { list: "history", view: "sets", kind: "history", rows: ".p03-history table tbody tr", moves: false },
    { list: "open_calls", view: "isolation", kind: "calls", rows: "tr.p03-call", moves: true },
    { list: "unknown", view: "reconciliation", kind: "targets", rows: "tr.p03-target", moves: true },
    { list: "pending", view: "resume", kind: "events", rows: "tr.p03-event", moves: true },
  ];
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1, again = 0; const asked = [];
      await page.route((url) => ends(url.toString(), "recovery"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.revision = body.revision + (again ? "-again" + again : "");
        const on = (rows, f) => rows.map((r) => (r.installation === inst ? f(r) : r));
        body.sets.rows = on(body.sets.rows, (r) => ({ ...r, history: span("history", 1, 50), history_total: 70, history_next: "21" }));
        body.isolation.rows = on(body.isolation.rows, (r) => ({ ...r, open_calls: span("open_calls", 1, 50), open_calls_total: 70, open_calls_next: "50" }));
        body.reconciliation.rows = on(body.reconciliation.rows, (r) => ({ ...r, unknown_targets: span("unknown", 1, 50), unknown_total: 70, unknown_next: "50" }));
        body.resume.rows = on(body.resume.rows, (r) => ({ ...r, pending_events: span("pending", 1, 50), pending_total: 70, pending_next: "50" }));
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "recovery/rows"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.toString());
        if (q.get("list") === "open_calls" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", installation: inst, list: q.get("list"), order: "recorded", rows_can_move: q.get("list") !== "history", rows: span(q.get("list"), 51, 70), next: "", total: 70 } });
      });
      for (const x of lists) {
        const at = `${lang} ${width} ${x.list}`;
        asked.length = 0;
        await open(lang, x.view);
        const scope = page.locator(`#p03-${x.view === "sets" ? "set" : x.view === "isolation" ? "iso" : x.view === "reconciliation" ? "rec" : "resume"}-${inst}`);
        const within = (await scope.count()) > 0 ? scope : page.locator("body");
        const rows = () => within.locator(x.rows);
        const foot = (name) => within.locator(`.p03-${x.kind}-${name}`);
        check(`${at} the first fifty are listed with the count and the button`, (await rows().count()) === 50 && (await foot("count").innerText()).trim() === copy[lang].shown(50, 70) && (await foot("more-button").innerText()).trim() === copy[lang].more[x.list] &&
          (x.moves ? (await foot("moves").innerText()).trim() === copy[lang].moves : (await foot("moves").count()) === 0));
        const more = foot("more-button");
        check(`${at} the button is at least 44 high`, (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
        await more.focus();
        await page.keyboard.press("Enter");
        if (x.list === "open_calls") {
          await foot("more").locator("[role=alert]").waitFor();
          const alert = await foot("more").innerText();
          check(`${at} a failed read says so, keeps the rows and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await rows().count()) === 50 && await more.evaluate((el) => el === document.activeElement), alert);
          if (args.shots) { await foot("more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p03-calls-failed-${lang}-${width}.png`, fullPage: false }); }
          await page.keyboard.press("Enter");
        }
        await page.waitForFunction(([sel, n]) => document.querySelectorAll(sel).length >= n, [x.rows, 70]);
        check(`${at} the rest follow and the foot is gone`, (await rows().count()) === 70 && asked[0] === new URLSearchParams({ installation: inst, list: x.list, limit: "50", after: x.list === "history" ? "21" : "50" }).toString() && (await foot("more").count()) === 0 && (await foot("count").count()) === 0, asked[0]);
        const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
        check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
        if (width === 1440 && x.list === "pending") {
          again += 1;
          await page.locator(".p03-reload").first().click();
          await page.waitForFunction(([sel, n]) => document.querySelectorAll(sel).length === n, [x.rows, 50]);
          check(`${at} after the screen is read again the list starts from the main read and says so`, (await foot("reset").innerText()).trim() === copy[lang].reset);
          if (args.shots) { await foot("count").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p03-events-reset-${lang}-${width}.png`, fullPage: false }); }
        }
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
