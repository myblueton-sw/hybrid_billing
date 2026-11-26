import { inspectView } from "./view-measure.mjs";
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
  en: { loading: "Loading the change…", failed: "The change could not be loaded", gone: "This change is no longer kept. Older changes are removed when the list reaches its limit.", waiting: "Open the change waiting for activation", replacing: "Open the replacing change", close: "Close", retry: "Try again", change: (n) => `Change ${n}`, back: "Back to trust changes" },
  ko: { loading: "변경을 불러오는 중…", failed: "변경을 불러오지 못했습니다", gone: "이 변경은 더 이상 보관되지 않습니다. 목록이 한도에 이르면 오래된 변경이 삭제됩니다.", waiting: "활성화 대기 중인 변경 열기", replacing: "대체한 변경 열기", close: "닫기", retry: "다시 시도", change: (n) => `변경 ${n}`, back: "신뢰 변경 목록으로" },
  ar: { loading: "جارٍ تحميل التغيير…", failed: "تعذّر تحميل التغيير", gone: "لم يعد هذا التغيير محفوظًا. تُحذف التغييرات الأقدم عندما تبلغ القائمة حدّها.", waiting: "فتح التغيير المنتظر للتفعيل", replacing: "فتح التغيير البديل", close: "إغلاق", retry: "إعادة المحاولة", change: (n) => `التغيير ${n}`, back: "العودة إلى تغييرات الثقة" },
};
const row0 = { change_revision: 1, old_epoch: { value: "1", generation: 0, number: 1 }, old_policy: 1, candidate: { methods: ["password"], mfa: "none", max_age_seconds: 36000, step_up: [] },
  proposer: "sample-admin-2", validator: "", approver: "", activator: "", superseded_by: "", earlier_run: false, run_started_at: "", validation_result: "", evidence_current: "not_applicable",
  invalidation_state: "", decide_in_detail: true, can: { revise: false, validate: true, reject: true, cancel: false }, why_not: { approve: "not_validated", activate: "needs_approval" }, act_until: "2026-11-27T09:10:00Z" };
const listed = (n, state, more = {}) => {
  const waiting = state === "draft";
  return { ...row0, id: `trust-0a1b2c3d-${n}`, number: n, state, decide_in_detail: waiting, can: waiting ? row0.can : { revise: false, validate: false, reject: false, cancel: false }, ...more };
};
const openRows = [listed(31, "draft"), listed(30, "draft")];
const closedRows = [listed(21, "superseded", { superseded_by: "trust-0a1b2c3d-22" }), listed(20, "cancelled")];
const pageBody = (qs, rows) => ({ rows, search_input: "", query: { q: "", state: qs.get("state") ?? "", limit: Number(qs.get("limit") ?? 25) }, next: "", reference_at: "2026-11-27T08:45:18Z", observed_at: "2026-11-27T08:45:18Z",
  revision: "qa", order: "created_desc", count: { value: String(rows.length), certainty: "exact" }, work_units: rows.length, scan_stopped: false });
const isList = (url) => new URL(url).pathname.endsWith("/identity/policy/changes");
const detailId = (url) => { const m = new URL(url).pathname.match(/\/identity\/policy\/changes\/([^/]+)$/); return m ? decodeURIComponent(m[1]) : ""; };
const isPolicy = (url) => new URL(url).pathname.endsWith("/identity/policy");
const refusal = (status, code) => ({ status, json: { code, retry: false, next_action: "none", request_id: "qa" } });

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
  let asked = [];
  let template = null;
  page.on("request", (r) => { if (isList(r.url()) || detailId(r.url())) asked.push(new URL(r.url()).pathname.split("/identity/policy/")[1] + new URL(r.url()).search); });
  page.on("response", async (r) => { if (detailId(r.url()) && r.status() === 200 && !template) template = (await r.json().catch(() => null))?.change ?? null; });
  const sec = (g) => `.m02-list-section[data-group="${g}"]`;
  const settled = (g) => page.locator(`${sec(g)} h3[data-count], ${sec(g)} .access-list-failure`).first().waitFor();
  const open = async (lang) => {
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/authentication?lang=${lang}&view=policies`);
    await page.locator("#m02-changes").waitFor();
    await settled("open"); await settled("closed");
    asked = [];
  };
  const focused = (selector) => page.evaluate((s) => !!document.activeElement && document.activeElement === document.querySelector(s), selector);
  const eventually = async (test) => { for (let i = 0; i < 40; i++) { if (await test()) return true; await page.waitForTimeout(50); } return false; };

  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("en");
  await page.locator(".m02-propose").click();
  await page.locator("#m02-age").fill("600");
  await page.locator(".m02-form button[type=submit]").click();
  await page.locator(".m02-confirm #live-reason").fill("synthetic rendered check");
  await page.locator(".m02-confirm button[type=submit]").click();
  await page.locator('.m02-detail-body:not([data-detail]) .m02-detail-title').waitFor();
  const realId = await page.locator(`${sec("open")} .m02-open`).first().getAttribute("data-change");
  check("after a real proposal the change opens from its own read", asked.some((a) => a === `changes/${realId}`) && (await page.locator(".m02-detail-title").innerText()).trim().startsWith("Change 1") && !!template && template.id === realId, asked.join(" "));
  check("the real detail offers what the proposer may do and the list marks the row", (await page.locator(".m02-detail [data-action]").count()) >= 1 && (await page.locator(`${sec("open")} tr[aria-current="true"] .m02-open`).getAttribute("data-change")) === realId && (await eventually(() => focused(".m02-detail-title"))));
  await page.locator(".m02-close").click();
  check("closing the real detail returns focus to its row", (await eventually(() => focused(`.m02-open[data-change="${realId}"]`))) && (await page.locator(".m02-detail-title").count()) === 0);
  asked = [];
  await page.locator(`.m02-open[data-change="${realId}"]`).click();
  await page.locator('.m02-detail-body:not([data-detail]) .m02-detail-title').waitFor();
  check("opening a listed row reads that change once per mount and nothing else", [...new Set(asked)].join(" ") === `changes/${realId}`, asked.join(" "));
  const stale = await page.locator(".m02-detail").innerText();
  check("the real detail states the impact that only the detail read carries", /Impact if activated/.test(stale), stale.slice(0, 200));
  if (!template) throw new Error("no detail answer was captured from the real service");
  const full = (id, n, state, more = {}) => ({ ...template, id, number: n, state, ...more });

  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 1024], ["en", 390], ["ko", 390], ["ar", 390]]) {
    const at = `${lang} ${width}`;
    const wide = width >= 1280;
    await page.setViewportSize({ width, height: 1000 });
    let mode = "ok", held = null, hold = false;
    await page.route((url) => isList(url.toString()), (route) => { const qs = new URL(route.request().url()).searchParams; return route.fulfill({ status: 200, json: pageBody(qs, qs.get("state") === "open" ? openRows : closedRows) }); });
    await page.route((url) => !!detailId(url.toString()), async (route) => {
      const id = detailId(route.request().url());
      if (mode === "fail") return route.fulfill(refusal(503, "dependency_unavailable"));
      if (mode === "gone") return route.fulfill(refusal(422, "invalid_request"));
      if (mode === "hold" && id === "trust-0a1b2c3d-31") { await new Promise((release) => { held = release; }); }
      const n = Number(id.split("-").pop());
      const state = n === 21 ? "superseded" : n === 22 ? "activated" : n === 9 ? "activation_unknown" : "draft";
      return route.fulfill({ status: 200, json: { change: full(id, n, state, n === 21 ? { superseded_by: "trust-0a1b2c3d-22", superseded_epoch: { value: "2", generation: 0, number: 2 }, can: { revise: false, validate: false, approve: false, activate: false, reject: false, cancel: false } } : {}), revision: "qa", observed_at: "2026-11-27T08:45:19Z" } });
    });
    await page.route((url) => isPolicy(url.toString()), async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      if (hold) { body.trust.hold = { reason: "commit_unknown", since: "2026-11-27T08:00:00Z" }; body.waiting_activation = "trust-0a1b2c3d-9"; }
      return route.fulfill({ response, json: body });
    });
    const leave = async () => { await (wide ? page.locator(".m02-detail .m02-close, .m02-detail .live-error button:last-child") : page.locator(".m02-back")).first().click(); await page.locator(".m02-detail-title").waitFor({ state: "detached" }); };
    await open(lang);

    mode = "hold";
    await page.locator('.m02-open[data-change="trust-0a1b2c3d-31"]').click();
    await page.locator('[data-detail="loading"]').waitFor();
    const loading = await page.locator('[data-detail="loading"]').evaluate((d) => ({ title: d.querySelector("h2").innerText.trim(), status: d.querySelector('[role="status"]').innerText.trim(), close: !!d.querySelector(".m02-close") }));
    check(`${at}: while the change loads the panel shows its number, says so and can be closed`, loading.title === copy[lang].change(31) && loading.status === copy[lang].loading && loading.close === wide && (wide || (await page.locator(".m02-back").innerText()).trim() === copy[lang].back) && (await eventually(() => focused(".m02-detail-title"))), JSON.stringify(loading));
    check(`${at}: the list ${wide ? "stays beside the loading panel with the row marked" : "is kept behind the step"}`, wide ? (await page.locator(`${sec("open")} tr[aria-current="true"] .m02-open`).getAttribute("data-change")) === "trust-0a1b2c3d-31" && (await page.locator(sec("open")).isVisible()) : !(await page.locator(sec("open")).isVisible()) && (await page.locator(`${sec("open")} .m02-open`).count()) === 2);
    if (args.shots) await page.screenshot({ path: `${args.shots}/m02-detail-loading-${lang}-${width}.png` });
    if (wide) {
      mode = "ok";
      await page.locator('.m02-open[data-change="trust-0a1b2c3d-30"]').click();
      await page.locator('.m02-detail-body:not([data-detail]) .m02-detail-title').waitFor();
      held?.();
      await page.waitForTimeout(300);
      check(`${at}: a late answer for the change opened first does not replace the one opened second`, (await page.locator(".m02-detail-title").innerText()).trim().startsWith(copy[lang].change(30)) && (await page.locator(`${sec("open")} tr[aria-current="true"] .m02-open`).getAttribute("data-change")) === "trust-0a1b2c3d-30");
    } else { mode = "ok"; held?.(); await page.locator('.m02-detail-body:not([data-detail]) .m02-detail-title').waitFor(); }
    await leave();
    check(`${at}: leaving the panel returns focus to the row`, await eventually(() => focused(`.m02-open[data-change="trust-0a1b2c3d-${wide ? 30 : 31}"]`)));

    mode = "fail";
    await page.locator('.m02-open[data-change="trust-0a1b2c3d-30"]').click();
    await page.locator('[data-detail="error"]').waitFor();
    const failed = await page.locator('[data-detail="error"]').evaluate((d) => ({ title: d.querySelector("h3").innerText.trim(), alert: !!d.querySelector('[role="alert"]'), buttons: [...d.querySelectorAll(".live-error button")].map((b) => b.innerText.trim()) }));
    check(`${at}: a failed read says so with Try again and Close`, failed.title === copy[lang].failed && failed.alert && failed.buttons.length === 2 && failed.buttons[1] === copy[lang].close && (!copy[lang].retry || failed.buttons[0] === copy[lang].retry), JSON.stringify(failed));
    if (args.shots && width !== 1024) await page.screenshot({ path: `${args.shots}/m02-detail-failed-${lang}-${width}.png` });
    mode = "ok";
    await page.locator('[data-detail="error"] .live-error button').first().click();
    await page.locator('.m02-detail-body:not([data-detail]) .m02-detail-title').waitFor();
    check(`${at}: Try again reads the change and shows it`, (await page.locator(".m02-detail-title").innerText()).trim().startsWith(copy[lang].change(30)));
    await leave();

    mode = "gone";
    await page.locator('.m02-open[data-change="trust-0a1b2c3d-30"]').click();
    await page.locator('[data-detail="gone"]').waitFor();
    const gone = await page.locator('[data-detail="gone"]').evaluate((d) => ({ text: d.querySelector(".m02-detail-gone p").innerText.trim(), buttons: [...d.querySelectorAll("button")].map((b) => b.innerText.trim()), alert: !!d.querySelector('[role="alert"]') }));
    check(`${at}: a change that is no longer kept says so with Close alone`, gone.text === copy[lang].gone && gone.buttons.length === 1 && gone.buttons[0] === copy[lang].close && !gone.alert, JSON.stringify(gone));
    if (args.shots && width !== 1024) await page.screenshot({ path: `${args.shots}/m02-detail-gone-${lang}-${width}.png` });
    asked = [];
    await page.locator('[data-detail="gone"] .m02-detail-gone button').click();
    await page.locator(".m02-detail-title").waitFor({ state: "detached" });
    check(`${at}: closing it reads both lists again from the first page`, await eventually(async () => [...new Set(asked)].sort().join(" ") === "changes?state=closed&limit=25 changes?state=open&limit=50"), asked.join(" "));
    mode = "ok";
    await settled("open"); await settled("closed");
    check(`${at}: after closing it focus rests on the changes heading, which outlives the reload`, await eventually(() => focused("#m02-changes")));

    asked = [];
    await page.locator('.m02-open[data-change="trust-0a1b2c3d-21"]').click();
    await page.locator('.m02-detail-body:not([data-detail]) .m02-detail-title').waitFor();
    const successor = page.locator('.m02-detail-body [data-opens="trust-0a1b2c3d-22"]').first();
    check(`${at}: a superseded change names its successor by identifier with a button to open it`, (await successor.innerText()).trim() === copy[lang].replacing && (await page.locator(".m02-detail-body").innerText()).includes("trust-0a1b2c3d-22"));
    if (args.shots && width !== 1024) await page.screenshot({ path: `${args.shots}/m02-detail-superseded-${lang}-${width}.png` });
    await successor.click();
    await page.waitForFunction((t) => document.querySelector('.m02-detail-body:not([data-detail]) .m02-detail-title')?.innerText.trim().startsWith(t), copy[lang].change(22));
    check(`${at}: the successor opens by its identifier from the detail read`, asked.includes("changes/trust-0a1b2c3d-22") && (await eventually(() => focused(".m02-detail-title"))), asked.join(" "));
    await leave();

    hold = true;
    await open(lang);
    const notice = page.locator('.m02-condition[data-condition="hold"] [data-opens="trust-0a1b2c3d-9"]');
    check(`${at}: the hold notice offers to open the waiting change the policy read names`, (await notice.count()) === 1 && (await notice.innerText()).trim() === copy[lang].waiting);
    await notice.click();
    await page.locator('.m02-detail-body:not([data-detail]) .m02-detail-title').waitFor();
    check(`${at}: the waiting change opens by identifier though no list row carries it`, asked.includes("changes/trust-0a1b2c3d-9") && (await page.locator(".m02-detail-title").innerText()).trim().startsWith(copy[lang].change(9)), asked.join(" "));
    const shown = await inspectView(page);
    check(`${at}: nothing overflows, is cut off or loses contrast with a detail open`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 3), cut: shown.cut.slice(0, 3), low: shown.low.slice(0, 3) }));
    hold = false;
    await page.unrouteAll({ behavior: "wait" });
  }
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) if (!args.quiet || r.startsWith("FAIL")) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
