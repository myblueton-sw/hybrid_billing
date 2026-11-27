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
  en: { open: (n) => `Open (${n})`, closed: (n) => `Closed (${n})`, noOpen: "No open changes.", noClosed: "No closed changes.", noOpenMatch: "No open change matches the filters.", spans: "Newest first. The open changes fill more than one page, so they are not sorted by state.",
    evidence: "Whether a change's evidence is still current is checked when you open it.", closedFailed: "Closed changes could not be loaded", search: "Change ID", first: "First", changed: "This query or continuation is no longer valid.", stale: "Evidence changed" },
  ko: { open: (n) => `진행 중 (${n})`, closed: (n) => `종료됨 (${n})`, noOpen: "진행 중인 변경이 없습니다.", noClosed: "종료된 변경이 없습니다.", noOpenMatch: "필터와 일치하는 진행 중인 변경이 없습니다.", spans: "최신순입니다. 진행 중인 변경이 한 페이지를 넘어 상태순으로 정렬하지 않습니다.",
    evidence: "변경의 근거가 아직 유효한지는 변경을 열 때 확인합니다.", closedFailed: "종료된 변경을 불러오지 못했습니다", search: "변경 ID", first: "처음", changed: "이 조회 조건 또는 이어보기 정보는 더 이상 유효하지 않습니다.", stale: "근거 변경됨" },
  ar: { open: (n) => `مفتوحة (${n})`, closed: (n) => `مغلقة (${n})`, noOpen: "لا توجد تغييرات مفتوحة.", noClosed: "لا توجد تغييرات مغلقة.", noOpenMatch: "لا يوجد تغيير مفتوح يطابق عوامل التصفية.", spans: "الأحدث أولًا. تتجاوز التغييرات المفتوحة صفحة واحدة، لذا لا تُرتَّب حسب الحالة.",
    evidence: "يُتحقَّق مما إذا كانت أدلة التغيير لا تزال سارية عند فتحه.", closedFailed: "تعذّر تحميل التغييرات المغلقة", search: "معرّف التغيير", first: "الأول", changed: "لم يعد هذا الاستعلام أو مرجع المتابعة صالحاً.", stale: "تغيّرت" },
};
const base0 = { change_revision: 1, old_epoch: { value: "2", generation: 0, number: 2 }, old_policy: 2, candidate: { methods: ["password"], mfa: "none", max_age_seconds: 3300, step_up: [] },
  proposer: "sample-admin-with-a-long-identifier-2", validator: "", approver: "", activator: "", superseded_by: "", earlier_run: false, run_started_at: "", validation_result: "", evidence_current: "not_applicable",
  invalidation_state: "", decide_in_detail: true, can: { revise: false, validate: true, reject: true, cancel: false }, why_not: { approve: "not_validated", activate: "needs_approval" }, act_until: "2026-11-27T09:10:00Z" };
const change = (n, state, more = {}) => {
  const waiting = state === "draft" || state === "validated" || state === "independently_approved";
  const validated = state === "validated" || state === "independently_approved";
  return { ...base0, id: `trust-0a1b2c3d-${n}`, number: n, state, decide_in_detail: waiting, validation_result: validated ? "validated" : "", evidence_current: validated ? "not_evaluated" : "not_applicable",
    validator: validated ? "sample-admin-3" : "", can: waiting ? base0.can : { revise: false, validate: false, reject: false, cancel: false }, ...more };
};
const openRows = [change(31, "draft"), change(30, "validated"), change(29, "independently_approved", { approver: "sample-admin-3" }), change(28, "draft", { validation_result: "blocked", evidence_current: "not_evaluated", validator: "sample-admin-3" })];
const closedAll = Array.from({ length: 27 }, (_, i) => {
  const n = 27 - i;
  if (n === 27) return change(n, "activated", { invalidation_state: "complete", activator: "sample-admin-3" });
  if (n === 26) return change(n, "superseded", { superseded_by: "trust-0a1b2c3d-27" });
  if (n === 1) return change(n, "activated", { id: "trust-11111111-1", number: 0, earlier_run: true, run_started_at: "2026-11-20T09:00:00Z", proposer: "", old_epoch: { value: "0", generation: 0, number: 0 }, old_policy: 0 });
  return change(n, n % 2 ? "rejected" : "cancelled");
});
const groups = { open: ["draft", "validated", "independently_approved", "activation_unknown"], closed: ["activated", "rejected", "cancelled", "superseded"] };
const pageBody = (qs, rows, change = {}) => ({ rows, search_input: qs.get("q") ?? "", query: { q: (qs.get("q") ?? "").toLowerCase(), state: qs.get("state") ?? "", limit: Number(qs.get("limit") ?? 25) }, next: "",
  reference_at: "2026-11-27T08:45:18Z", observed_at: "2026-11-27T08:45:18Z", revision: "qa", order: "created_desc", count: { value: String(rows.length), certainty: "exact" }, work_units: rows.length, scan_stopped: false, ...change });
const isList = (url) => new URL(url).pathname.endsWith("/identity/policy/changes");
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
  page.on("request", (r) => { if (isList(r.url())) asked.push(new URL(r.url()).search); });
  const sec = (g) => `.m02-list-section[data-group="${g}"]`;
  const settled = (g) => page.locator(`${sec(g)} h3[data-count], ${sec(g)} .access-list-failure`).first().waitFor();
  const open = async (lang) => {
    asked = [];
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/authentication?lang=${lang}&view=policies`);
    await page.locator("#m02-changes").waitFor();
    await settled("open"); await settled("closed");
  };
  const eventually = async (f) => { for (let i = 0; i < 60; i++) { if (await f()) return true; await page.waitForTimeout(50); } return false; };
  const showFilters = async () => { const t = page.locator(".m02-filters .list-filters-toggle"); if (await t.isVisible() && (await t.getAttribute("aria-expanded")) === "false") await t.click(); };
  const heading = async (g) => (await page.locator(`${sec(g)} h3`).first().innerText()).trim();
  const ids = (g) => page.locator(`${sec(g)} .m02-open`).evaluateAll((b) => b.map((x) => x.dataset.change));

  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("en");
  const realAsked = [...new Set(asked)].sort();
  check("the real service is asked for the open group with a limit of 50 and the closed group with 25", realAsked.join(" ") === "?state=closed&limit=25 ?state=open&limit=50", realAsked.join(" "));
  check("with no changes the real sections count none and say so", (await heading("open")) === "Open (0)" && (await heading("closed")) === "Closed (0)" && (await page.locator(`${sec("open")} .live-empty`).innerText()).trim() === copy.en.noOpen && (await page.locator(`${sec("closed")} .live-empty`).innerText()).trim() === copy.en.noClosed);
  check("the real sections show a reference time, no paging and no notes", (await page.locator(".m02-lists .list-summary").count()) === 2 && (await page.locator(".m02-lists .list-pager-nav").count()) === 0 && (await page.locator(".m02-order-note, .m02-evidence-note").count()) === 0);
  if (await page.locator(".m02-propose").count()) {
    await page.locator(".m02-propose").click();
    await page.locator("#m02-age").fill("600");
    await page.locator(".m02-form button[type=submit]").click();
    await page.locator(".m02-confirm #live-reason").fill("synthetic rendered check");
    {
      await page.locator(".m02-confirm button[type=submit]").click();
      await page.waitForFunction(() => document.querySelector('.m02-list-section[data-group="open"] h3')?.dataset.count === "1", null, { timeout: 15000 }).then(
        async () => {
          check("after a real proposal the open section counts one and lists it without a further action", (await heading("open")) === "Open (1)" && (await ids("open")).length === 1);
          check("the real proposal opens in the detail beside the list", (await page.locator(".m02-detail-title").count()) === 1 && (await page.locator(`${sec("open")} tr[aria-current="true"]`).count()) === 1);
        },
        () => check("a real proposal reaches the open list", false, "the open section did not count one"));
    }
  } else notes.push("the signed-in administrator may not propose; the real proposal step was skipped");

  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 1280], ["en", 1024], ["en", 390], ["ko", 390], ["ar", 390]]) {
    const at = `${lang} ${width}`;
    await page.setViewportSize({ width, height: 1000 });
    let mode = "mix";
    await page.route((url) => isList(url.toString()), async (route) => {
      const qs = new URL(route.request().url()).searchParams;
      const state = qs.get("state"), q = (qs.get("q") ?? "").toLowerCase();
      const group = state === "open" || groups.open.includes(state) ? "open" : "closed";
      const admits = (r) => (state === group || r.state === state) && r.id.includes(q);
      if (group === "closed") {
        if (mode === "closed-fails") return route.fulfill(refusal(503, "dependency_unavailable"));
        if (qs.get("after") === "c.2") return mode === "moved" ? route.fulfill(refusal(409, "revision_conflict")) : route.fulfill({ status: 200, json: pageBody(qs, closedAll.slice(25).filter(admits), { count: { value: "27", certainty: "exact" } }) });
        const rows = closedAll.filter(admits);
        const n = Number(qs.get("limit") ?? 25);
        return route.fulfill({ status: 200, json: pageBody(qs, rows.slice(0, n), rows.length > n ? { next: "c.2", count: { value: String(rows.length), certainty: "exact" } } : {}) });
      }
      const rows = openRows.filter(admits);
      if (mode === "open-spans" && !qs.get("after")) return route.fulfill({ status: 200, json: pageBody(qs, Array.from({ length: 50 }, (_, i) => change(400 - i, i % 2 ? "draft" : "validated")), { next: "o.2", count: { value: "104", certainty: "exact" } }) });
      return route.fulfill({ status: 200, json: pageBody(qs, rows) });
    });
    await open(lang);
    await page.mouse.move(2, 2);
    check(`${at}: the headings carry each list's exact count`, (await heading("open")) === copy[lang].open(4) && (await heading("closed")) === copy[lang].closed(27), `${await heading("open")} / ${await heading("closed")}`);
    check(`${at}: an open group on one page is sorted by state, then newest`, (await ids("open")).join(" ") === "trust-0a1b2c3d-29 trust-0a1b2c3d-30 trust-0a1b2c3d-31 trust-0a1b2c3d-28", (await ids("open")).join(" "));
    check(`${at}: the closed section keeps the service's order, twenty-five to a page, with paging`, (await ids("closed")).length === 25 && (await ids("closed"))[0] === "trust-0a1b2c3d-27" && (await page.locator(`${sec("closed")} .list-pager-nav button`).count()) === 3 && (await page.locator(`${sec("open")} .list-pager-nav`).count()) === 0);
    check(`${at}: the evidence note stands above the open table only, and no order note`, (await page.locator(`${sec("open")} .m02-evidence-note`).innerText()).trim() === copy[lang].evidence && (await page.locator(`${sec("closed")} .m02-evidence-note`).count()) === 0 && (await page.locator(".m02-order-note").count()) === 0);
    check(`${at}: a blocked draft is marked and no row says the evidence changed`, (await page.locator(`${sec("open")} tr[data-state="draft"] .m02-state-word.live-strong`).count()) === 1 && (await page.locator(`${sec("open")} tr[data-state="validated"] .live-strong`).count()) === 0 && !(await page.locator(".m02-lists").innerText()).includes(copy[lang].stale));
    const shown = await inspectView(page);
    check(`${at}: nothing overflows, is cut off or loses contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 3), cut: shown.cut.slice(0, 3), low: shown.low.slice(0, 3) }));
    const cells = await page.locator(".m02-lists td").evaluateAll((tds) => tds.flatMap((td) => { const c = td.getBoundingClientRect(); return [...td.querySelectorAll("bdi, span, button")].filter((x) => { const r = x.getBoundingClientRect(); return r.width > 0 && (r.right > c.right + 1 || r.left < c.left - 1); }).map((x) => x.textContent.slice(0, 30)); }));
    check(`${at}: no cell's content reaches into the next column`, cells.length === 0, cells.slice(0, 4).join(" | "));
    const toggle = page.locator(".m02-filters .list-filters-toggle");
    if (width < 600) check(`${at}: at a narrow width the filters start collapsed behind one control`, (await toggle.isVisible()) && (await toggle.getAttribute("aria-expanded")) === "false" && !(await page.locator("#m02-filter-form").isVisible()));
    else check(`${at}: at a wide width the filters show without a toggle`, !(await toggle.isVisible()) && (await page.locator("#m02-filter-form").isVisible()));
    if (await toggle.isVisible() && (await toggle.getAttribute("aria-expanded")) === "false") await toggle.click();
    const form = await page.locator(".m02-filters form").evaluate((f) => { const r = f.getBoundingClientRect(); return { fits: r.right <= document.documentElement.clientWidth + 1 && r.left >= -1, labels: [...f.querySelectorAll("label")].map((x) => x.innerText.trim()), buttons: [...f.querySelectorAll("button")].map((b) => { const q = b.getBoundingClientRect(); return q.height >= 24 && b.scrollWidth <= b.clientWidth + 1; }) }; });
    check(`${at}: the filter row fits, names its fields and its buttons are whole`, form.fits && form.labels[0] === copy[lang].search && form.labels.length === 2 && form.buttons.length === 2 && form.buttons.every(Boolean), JSON.stringify(form));
    if (args.shots) await page.locator(".m02-changes").screenshot({ path: `${args.shots}/m02-lists-${lang}-${width}.png` });
    if (args.shots) { await page.locator(".m02-filters").screenshot({ path: `${args.shots}/m02-lists-top-filters-${lang}-${width}.png` }); await page.locator(sec("open")).screenshot({ path: `${args.shots}/m02-lists-top-open-${lang}-${width}.png` }); }
    const rowTops = await page.locator(`${sec("open")} tbody tr`).first().evaluate((tr) => { const tds = [...tr.querySelectorAll("td")]; if (tds.length < 2) return [0, 0]; const top = (el) => { const range = document.createRange(); range.selectNodeContents(el); return range.getClientRects()[0]?.top ?? 0; }; return [top(tds[0].querySelector(".m02-open")), top(tds[1].querySelector(".m02-state-word"))]; });
    check(`${at}: a row's change link starts on the same line as its state`, width < 800 || Math.abs(rowTops[0] - rowTops[1]) <= 3, rowTops.join(" "));
    const labelTops = await page.locator(".m02-filters label").evaluateAll((ls) => ls.map((x) => Math.round(x.getBoundingClientRect().top)));
    check(`${at}: the two filter labels start on one line when side by side`, width < 800 || Math.abs(labelTops[0] - labelTops[1]) <= 2, labelTops.join(" "));

    asked = [];
    await page.locator("#m02-size-closed").selectOption("10");
    await eventually(async () => asked.some((q) => q.includes("state=closed") && q.includes("limit=10")));
    await page.waitForFunction((s) => document.querySelectorAll(`${s} .m02-open`).length > 0, sec("closed"));
    check(`${at}: ten rows per page on the closed list asks for ten, starts at the first page and keeps focus on the choice`, asked.some((q) => q.includes("state=closed") && q.includes("limit=10") && !q.includes("after")) && (await eventually(() => page.evaluate(() => document.activeElement?.id === "m02-size-closed"))), asked.join(" "));
    asked = [];
    await page.locator("#m02-size-closed").selectOption("25");
    await eventually(async () => asked.some((q) => q.includes("state=closed") && q.includes("limit=25")));
    await page.waitForFunction((s) => document.querySelectorAll(`${s} .m02-open`).length === 25, sec("closed"));
    await page.locator(`${sec("closed")} .list-pager-nav button`).nth(2).click();
    await page.waitForFunction((s) => document.querySelectorAll(`${s} .m02-open`).length === 2, sec("closed"));
    check(`${at}: the second closed page has the last two with the same count, the earlier run's row, and focus on the reference time`, (await heading("closed")) === copy[lang].closed(27) && (await ids("closed")).join(" ") === "trust-0a1b2c3d-2 trust-11111111-1" && (await page.evaluate((s) => document.activeElement === document.querySelector(`${s} .list-summary`), sec("closed"))) && (await page.locator(`${sec("closed")} .m02-open`).last().evaluate((b) => b.closest("td").querySelectorAll(".m02-sub").length)) >= 1 && (await ids("open")).length === 4);
    await page.locator(`${sec("closed")} .list-pager-nav button`).nth(0).click();
    await page.waitForFunction((s) => document.querySelectorAll(`${s} .m02-open`).length === 25, sec("closed"));

    asked = [];
    await showFilters();
    await page.locator("#m02-filter-state").selectOption("validated");
    await showFilters();
    await page.locator(".m02-filter-buttons button[type=submit]").click();
    await page.waitForFunction(() => document.querySelector('.m02-list-section[data-group="open"] h3')?.dataset.count === "1");
    check(`${at}: one state shows only its section and asks for that state`, (await page.locator(sec("closed")).count()) === 0 && (await ids("open")).join(" ") === "trust-0a1b2c3d-30" && (await heading("open")) === copy[lang].open(1) && asked.join(" ") === "?state=validated&limit=50", asked.join(" "));
    asked = [];
    await showFilters();
    await page.locator("#m02-filter-state").selectOption("");
    await showFilters();
    await page.locator("#m02-filter-q").fill(" D-2 ");
    await showFilters();
    await page.locator("#m02-filter-q").press("Enter");
    await page.waitForFunction(() => document.querySelector('.m02-list-section[data-group="closed"] h3')?.dataset.count === "9");
    await page.waitForFunction(() => document.querySelector('.m02-list-section[data-group="open"] h3')?.dataset.count === "2");
    check(`${at}: Enter applies the search to both sections, trimmed`, [...new Set(asked)].sort().join(" ") === "?q=D-2&state=closed&limit=25 ?q=D-2&state=open&limit=50" && (await ids("open")).join(" ") === "trust-0a1b2c3d-29 trust-0a1b2c3d-28", asked.join(" "));
    const chips = await page.locator(".m02-lists .list-chip").allInnerTexts();
    if (args.shots) { await page.mouse.move(0, 0); await page.locator(".m02-lists").screenshot({ path: `${args.shots}/m02-lists-chips-${lang}-${width}.png` }); }
    check(`${at}: the applied search shows as one chip with its field name`, chips.length === 1 && chips[0].includes(copy[lang].search) && chips[0].includes("D-2") && (await page.locator(".m02-lists .list-applied-clear button").count()) === 1, chips.join(" | "));
    asked = [];
    await page.locator(".m02-lists .list-chip-remove").click();
    await page.waitForFunction(() => document.querySelector('.m02-list-section[data-group="open"] h3')?.dataset.count === "4");
    check(`${at}: removing the chip clears the search, reads both groups again and keeps focus in the filters`, (await page.locator("#m02-filter-q").inputValue()) === "" && (await page.locator(".m02-lists .list-chip").count()) === 0 && [...new Set(asked)].sort().join(" ") === "?state=closed&limit=25 ?state=open&limit=50" && (await page.evaluate(() => { const a = document.activeElement; const r = a?.getBoundingClientRect(); return !!r && r.width > 8 && r.height > 8 && (a.id === "m02-filters-title" || a.classList.contains("list-filters-toggle")); })), asked.join(" "));
    await showFilters();
    await page.locator("#m02-filter-q").fill("zzz");
    await showFilters();
    await page.locator(".m02-filter-buttons button[type=submit]").click();
    await page.waitForFunction(() => document.querySelector('.m02-list-section[data-group="open"] h3')?.dataset.count === "0");
    check(`${at}: a search with no match says so for the filters`, (await page.locator(`${sec("open")} .live-empty`).innerText()).trim() === copy[lang].noOpenMatch && (await page.locator(`${sec("open")} .m02-evidence-note`).count()) === 0);
    asked = [];
    await showFilters();
    await page.locator(".m02-filter-buttons button[type=button]").click();
    await page.waitForFunction(() => document.querySelector('.m02-list-section[data-group="open"] h3')?.dataset.count === "4");
    check(`${at}: Reset clears the fields and reads both groups again`, (await page.locator("#m02-filter-q").inputValue()) === "" && (await page.locator("#m02-filter-state").inputValue()) === "" && [...new Set(asked)].sort().join(" ") === "?state=closed&limit=25 ?state=open&limit=50", asked.join(" "));

    if (await page.locator(".m02-propose").count()) {
      await showFilters();
      await page.locator("#m02-filter-state").selectOption("validated");
      await showFilters();
      await page.locator(".m02-filter-buttons button[type=submit]").click();
      await page.waitForFunction(() => document.querySelector('.m02-list-section[data-group="open"] h3')?.dataset.count === "1");
      asked = [];
      await page.locator(".m02-propose").click();
      await page.locator(".m02-form h2").waitFor();
      const stepped = (await page.locator(".m02-step").count()) === 1;
      await (stepped ? page.locator(".m02-back") : page.locator(".m02-form .live-secondary").last()).click();
      await page.locator(".m02-form").waitFor({ state: "detached" });
      check(`${at}: opening and leaving a panel keeps the filter and the rows and asks for nothing${stepped ? " (step view)" : ""}`, (width >= 1280) !== stepped && (await page.locator("#m02-filter-state").inputValue()) === "validated" && (await ids("open")).join(" ") === "trust-0a1b2c3d-30" && (await page.locator(sec("closed")).count()) === 0 && asked.length === 0, `${stepped} ${asked.join(" ")}`);
      await showFilters();
      await page.locator(".m02-filter-buttons button[type=button]").click();
      await page.waitForFunction(() => document.querySelector('.m02-list-section[data-group="open"] h3')?.dataset.count === "4");
      await page.waitForFunction((s) => document.querySelectorAll(`${s} .m02-open`).length === 25, sec("closed"));
    }
    mode = "moved";
    await page.locator(`${sec("closed")} .list-pager-nav button`).nth(2).click();
    await page.locator(`${sec("closed")} .access-list-failure`).waitFor();
    const failure = await page.locator(`${sec("closed")} .access-list-failure`).evaluate((f) => ({ title: f.querySelector("h3").innerText.trim(), body: f.querySelector("p").innerText.trim(), buttons: [...f.querySelectorAll("button")].map((b) => b.innerText.trim()) }));
    check(`${at}: a continuation that no longer fits says the query changed and offers First, and the open list stays`, failure.title === copy[lang].closedFailed && failure.body.startsWith(copy[lang].changed) && failure.buttons.includes(copy[lang].first) && (await ids("open")).length === 4, JSON.stringify(failure));
    mode = "mix";
    await page.locator(`${sec("closed")} .access-list-failure button`).last().click();
    await page.waitForFunction((s) => document.querySelectorAll(`${s} .m02-open`).length === 25, sec("closed"));
    check(`${at}: First starts the closed list again`, (await heading("closed")) === copy[lang].closed(27));

    mode = "closed-fails";
    await open(lang);
    check(`${at}: a failed closed list says so, shows no count, and leaves the open list and the current rules`, (await page.locator(`${sec("closed")} .access-list-failure h3`).innerText()).trim() === copy[lang].closedFailed && (await page.locator(`${sec("closed")} h3[data-count]`).count()) === 0 && (await ids("open")).length === 4 && (await page.locator("#m02-current").count()) === 1);
    if (lang === "en" && width === 1440) {
      mode = "mix";
      await open(lang);
      const first = async () => (await ids("closed"))[0];
      const hash = () => page.evaluate(() => Object.fromEntries(new URLSearchParams(location.hash.split("?")[1])));
      await page.locator(`${sec("closed")} .list-pager-nav button`).nth(2).click();
      await page.waitForFunction((s) => document.querySelector(`${s} .m02-open`)?.dataset.change === "trust-0a1b2c3d-2", sec("closed"));
      check(`${at}: the closed list's page position is in the address`, (await hash()).tc_ca === "c.2");
      await page.goBack();
      await page.waitForFunction((s) => document.querySelector(`${s} .m02-open`)?.dataset.change === "trust-0a1b2c3d-27", sec("closed"));
      check(`${at}: Back returns the closed list to its first page`, (await hash()).tc_ca === undefined && (await first()) === "trust-0a1b2c3d-27");
      await page.goForward();
      await page.waitForFunction((s) => document.querySelector(`${s} .m02-open`)?.dataset.change === "trust-0a1b2c3d-2", sec("closed"));
      check(`${at}: Forward returns it to the second page`, (await hash()).tc_ca === "c.2");
      await page.reload();
      await page.locator("#m02-changes").waitFor();
      await page.waitForFunction((s) => document.querySelector(`${s} .m02-open`)?.dataset.change === "trust-0a1b2c3d-2", sec("closed"));
      check(`${at}: reload keeps the second page`, (await first()) === "trust-0a1b2c3d-2");
      await page.locator("#m02-filter-q").fill("D-2");
      await page.locator(".m02-filter-buttons button[type=submit]").click();
      await page.waitForFunction(() => location.hash.includes("tc_q=D-2"));
      check(`${at}: applying a search writes it and drops both cursors`, (await hash()).tc_q === "D-2" && (await hash()).tc_ca === undefined);
      await page.goBack();
      await page.waitForFunction(() => !location.hash.includes("tc_q="));
      await page.waitForFunction(() => document.querySelector("#m02-filter-q")?.value === "");
      check(`${at}: Back restores the unfiltered list and empties the search field`, (await page.locator(".m02-lists .list-chip").count()) === 0);
      await page.goForward();
      await page.waitForFunction(() => document.querySelector("#m02-filter-q")?.value === "D-2");
      check(`${at}: Forward restores the search, its chip and the field`, (await page.locator(".m02-lists .list-chip").count()) === 1);
      await page.goto("about:blank");
      await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#/authentication?lang=en&view=policies&tc_state=bogus&tc_cl=500&tc_q=${"x".repeat(80)}`);
      await page.locator("#m02-changes").waitFor();
      await settled("open"); await settled("closed");
      check(`${at}: an address with invalid values falls back to the defaults and sends nothing invalid`, (await page.locator(".m02-lists .list-chip").count()) === 0 && (await page.locator("#m02-size-closed").inputValue()) === "25" && asked.every((q) => !q.includes("bogus") && !q.includes("limit=500")), asked.join(" "));
    }
    mode = "open-spans";
    await open(lang);
    check(`${at}: an open group over pages keeps the service's order and says so`, (await ids("open")).slice(0, 3).join(" ") === "trust-0a1b2c3d-400 trust-0a1b2c3d-399 trust-0a1b2c3d-398" && (await page.locator(`${sec("open")} .m02-order-note`).innerText()).trim() === copy[lang].spans && (await heading("open")) === copy[lang].open(104) && (await page.locator(`${sec("open")} .list-pager-nav button`).count()) === 3);
    if (args.shots && width !== 1280) await page.locator(sec("open")).screenshot({ path: `${args.shots}/m02-lists-spans-${lang}-${width}.png` });
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
