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
  en: { waiting: (n) => (n === 1 ? "1 waiting" : `${n} waiting`), atLeast: (n) => `At least ${n} waiting`, ended: (n) => `Ended requests (${n})`, endedAtLeast: (n) => `Ended requests (at least ${n})`, endedFailed: "Ended requests (could not be loaded)", failed: "Access change requests could not be loaded", none: "No access change requests are waiting.", reference: "List reference time", next: "Next" },
  ko: { waiting: (n) => `대기 ${n}건`, atLeast: (n) => `대기 ${n}건 이상`, ended: (n) => `종료된 요청 (${n})`, endedAtLeast: (n) => `종료된 요청 (최소 ${n}건)`, endedFailed: "종료된 요청 (불러오지 못함)", failed: "권한 변경 승인 요청을 불러오지 못했습니다", none: "대기 중인 권한 변경 승인 요청이 없습니다.", reference: "목록 기준 시각", next: "다음" },
  ar: { waiting: (n) => (n === 2 ? "طلبان بانتظار الاعتماد" : ""), atLeast: (n) => `عدد الطلبات بانتظار الاعتماد: ${n} على الأقل`, ended: (n) => `الطلبات المنتهية (${n})`, endedAtLeast: (n) => `الطلبات المنتهية (${n} على الأقل)`, endedFailed: "الطلبات المنتهية (تعذّر التحميل)", failed: "تعذّر تحميل طلبات تغيير الصلاحيات", none: "لا توجد طلبات تغيير صلاحيات بانتظار الاعتماد.", reference: "وقت مرجع القائمة", next: "التالي" },
};
const base0 = { change: "add", subject: "sample-member", grant: "", action: "view", kind: "focus_data", fields: ["rows"], customer: "all", days: 10, digest: "668654c4bcbe3cc0bd526cd6f66da3cb5842938430375d9f3e3bb7e546d49461", state: "pending", current: true,
  proposer: "sample-admin-2", proposed_at: "2026-11-16T09:00:00Z", expires_at: "2026-11-23T09:00:00Z", reason: "sample reason", decided_by: "", decided_at: "", decision_reason: "", result: "", eligible_approvers: 1,
  can_approve: true, can_reject: true, can_withdraw: false, may_end: [], subject_login: "member-with-a-long-sign-in-name@example.invalid", proposer_login: "admin2@example.invalid", decided_by_login: "" };
const pendingRow = (n, change = {}) => ({ ...base0, id: `access-request-${n}`, ...change });
const endedRow = (n) => ({ ...base0, id: `access-request-${n}`, state: n % 3 === 0 ? "rejected" : "applied", current: false, decided_by: "sample-admin", decided_by_login: "admin@example.invalid", decided_at: "2026-11-16T10:00:00Z",
  decision_reason: n % 3 === 0 ? "Not needed after all" : "", result: n % 3 === 0 ? "" : `grant-${n}`, eligible_approvers: 0, can_approve: false, can_reject: false, can_withdraw: false });
const pageBody = (qs, rows, change = {}) => ({ rows, search_input: "", query: { q: "", state: qs.get("state") ?? "", limit: Number(qs.get("limit") ?? 25) }, next: "", reference_at: "2026-11-26T08:45:18Z", observed_at: "2026-11-26T08:45:18Z", revision: "r1", order: "number_desc",
  count: { value: String(rows.length), certainty: "exact" }, work_units: rows.length, scan_stopped: false, ...change });
const isList = (url) => new URL(url).pathname.endsWith("/identity/access/requests");

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
  const asked = [];
  page.on("request", (r) => { if (isList(r.url())) asked.push(new URL(r.url()).search); });
  let mode = "two";
  const open = async (lang) => {
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/access?${new URLSearchParams({ lang, view: "grants" })}`);
    await page.locator("#access-requests").waitFor();
    await page.locator(".access-request-section [data-waiting], .access-request-section .access-list-failure").first().waitFor();
    await page.waitForFunction(() => { const section = document.querySelector(".access-request-section"); const ended = section.querySelector("details.access-ended-requests"); return !section.querySelector("[role=status]") && (!ended || !ended.querySelector("[role=status]")); });
  };
  const section = ".access-request-section";

  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("en");
  check("the real service's section reads pending and ended requests as two bounded reads", asked.includes("?state=pending&limit=50") && asked.includes("?state=ended&limit=25") && asked.every((s) => s === "?state=pending&limit=50" || s === "?state=ended&limit=25"), asked.join(" "));
  check("with no requests the real section says none are waiting, hides the ended list and shows no paging", (await page.locator(`${section} [data-waiting]`).innerText()).trim() === "0 waiting" && (await page.locator(`${section} .live-empty`).first().innerText()).trim() === copy.en.none && (await page.locator(`${section} details.access-ended-requests`).count()) === 0 && (await page.locator(`${section} .access-paging`).count()) === 0);

  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 390], ["ko", 390], ["ar", 390]]) {
    const at = `${lang} ${width}`;
    await page.setViewportSize({ width, height: 1000 });
    mode = "two";
    await page.route((url) => isList(url.toString()), async (route) => {
      const qs = new URL(route.request().url()).searchParams;
      const state = qs.get("state");
      if (mode === "pendingfail" && state === "pending") return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
      if (mode === "endedfail" && state === "ended") return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
      if (state === "pending") {
        if (mode === "pair") return route.fulfill({ status: 200, json: pageBody(qs, [pendingRow(95), pendingRow(94, { can_reject: false, can_approve: false, can_withdraw: true, proposer: "sample-admin", proposer_login: "admin@example.invalid" })]) });
        if (mode === "many") {
          if (qs.get("after") === "p.2") return route.fulfill({ status: 200, json: pageBody(qs, [pendingRow(1)], { count: { value: "51", certainty: "exact" } }) });
          return route.fulfill({ status: 200, json: pageBody(qs, Array.from({ length: 50 }, (_, i) => pendingRow(200 - i)), { next: "p.2", count: { value: "50", certainty: "lower_bound" } }) });
        }
        return route.fulfill({ status: 200, json: pageBody(qs, [pendingRow(92, { current: false, can_approve: false }), pendingRow(91, { eligible_approvers: 0, can_approve: false, can_reject: false, can_withdraw: true, change: "revoke", grant: "reader-sample-member", days: 0, may_end: ["qd-00"] })]) });
      }
      if (qs.get("after") === "e.2") return route.fulfill({ status: 200, json: pageBody(qs, Array.from({ length: 5 }, (_, i) => endedRow(55 - i)), { count: { value: "30", certainty: "exact" } }) });
      return route.fulfill({ status: 200, json: pageBody(qs, Array.from({ length: 25 }, (_, i) => endedRow(80 - i)), { next: "e.2", count: { value: "25", certainty: "lower_bound" } }) });
    });
    await open(lang);
    const head = (await page.locator(`${section} [data-waiting]`).innerText()).trim();
    check(`${at}: two waiting requests are counted in the heading, with their actions`, (lang === "ar" ? head === copy.ar.waiting(2) : head === copy[lang].waiting(2)) && (await page.locator(`${section} tr[data-request="access-request-92"] [data-act="reject"]`).count()) === 1 && (await page.locator(`${section} tr[data-request="access-request-92"] [data-act="approve"]`).count()) === 0 && (await page.locator(`${section} tr[data-request="access-request-91"] [data-act="withdraw"]`).count()) === 1, head);
    check(`${at}: one page of waiting requests has a reference time and no paging`, (await page.locator(`${section} > .access-reference`).innerText()).startsWith(copy[lang].reference) && (await page.locator(`${section} > .access-paging`).count()) === 0);
    const summary = page.locator(`${section} details.access-ended-requests > summary`);
    check(`${at}: the ended list is collapsed and its summary gives a lower bound`, (await summary.innerText()).trim() === copy[lang].endedAtLeast(25) && !(await page.locator(`${section} details.access-ended-requests`).evaluate((el) => el.open)), await summary.innerText());
    const shown = await inspectView(page);
    check(`${at}: nothing overflows, is cut off or loses contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), low: shown.low.slice(0, 2) }));
    const fit = await page.evaluate(() => [...document.querySelectorAll(".access-requests")].filter((b) => b.getClientRects().length).map((b) => { const t = b.querySelector("table").getBoundingClientRect(), box = b.getBoundingClientRect(); const spill = [...b.querySelectorAll("button")].filter((x) => x.scrollWidth > x.clientWidth + 1).length; return [Math.round(t.width), Math.round(box.width), spill]; }));
    check(`${at}: the table fits its container and no button's text spills`, fit.every(([t, box, spill]) => t <= box + 1 && spill === 0), JSON.stringify(fit));
    await page.mouse.move(2, 2);
    const tint = await page.locator(`${section} > .access-requests tbody tr`).evaluateAll((rows) => rows.map((r) => [r.className, getComputedStyle(r).backgroundColor]));
    check(`${at}: no waiting row is shown as selected while no panel is open`, tint.every(([name]) => !name.includes("live-selected")) && new Set(tint.map(([, bg]) => bg)).size === 1, JSON.stringify(tint));
    if (args.shots) await page.locator(section).screenshot({ path: `${args.shots}/m03-requests-${lang}-${width}.png` });
    await summary.focus(); await page.keyboard.press("Enter");
    await page.locator(`${section} details.access-ended-requests .access-paging`).waitFor();
    check(`${at}: opened, the ended list shows 25 rows, a reference time and paging with First and Previous disabled`, (await page.locator(`${section} details.access-ended-requests tbody tr`).count()) === 25 && (await page.locator(`${section} details.access-ended-requests .access-reference`).count()) === 1 && await page.locator(`${section} details.access-ended-requests .access-paging button`).nth(0).isDisabled() && await page.locator(`${section} details.access-ended-requests .access-paging button`).nth(1).isDisabled() && await page.locator(`${section} details.access-ended-requests .access-paging button`).nth(2).isEnabled() && (await page.locator(`${section} details.access-ended-requests [data-act]`).count()) === 0, JSON.stringify(await page.evaluate((sel) => ({ rows: document.querySelectorAll(`${sel} details.access-ended-requests tbody tr`).length, refs: document.querySelectorAll(`${sel} details.access-ended-requests .access-reference`).length, acts: document.querySelectorAll(`${sel} details.access-ended-requests [data-act]`).length, buttons: [...document.querySelectorAll(`${sel} details.access-ended-requests .access-paging button`)].map((b) => b.disabled) }), section)));
    await page.locator(`${section} details.access-ended-requests .access-paging button`).nth(2).click();
    await page.waitForFunction((s) => document.querySelectorAll(`${s} details.access-ended-requests tbody tr`).length === 5, section);
    check(`${at}: the second page has the last five, an exact total in the summary, Next disabled and Previous enabled, and the list stays open`, (await summary.innerText()).trim() === copy[lang].ended(30) && await page.locator(`${section} details.access-ended-requests .access-paging button`).nth(2).isDisabled() && await page.locator(`${section} details.access-ended-requests .access-paging button`).nth(1).isEnabled() && await page.locator(`${section} details.access-ended-requests`).evaluate((el) => el.open) && asked[asked.length - 1] === "?state=ended&limit=25&after=e.2", asked[asked.length - 1]);
    check(`${at}: after Next disables itself on the last page, focus is on that list's reference time, not lost`, await page.evaluate(() => document.activeElement?.classList.contains("access-reference") && !!document.activeElement.closest("details.access-ended-requests")));
    if (args.shots && lang === "en") await page.locator(section).screenshot({ path: `${args.shots}/m03-requests-ended-page2-${lang}-${width}.png` });
    await page.locator(`${section} details.access-ended-requests .access-paging button`).nth(1).click();
    await page.waitForFunction((s) => document.querySelectorAll(`${s} details.access-ended-requests tbody tr`).length === 25, section);
    check(`${at}: Previous returns to the first page of ended requests`, asked[asked.length - 1] === "?state=ended&limit=25");
    mode = "many";
    await open(lang);
    check(`${at}: more than a page of waiting requests gives a lower-bound line and paging`, (await page.locator(`${section} [data-waiting]`).innerText()).trim() === copy[lang].atLeast(50) && (await page.locator(`${section} > .access-paging button`).count()) === 3 && await page.locator(`${section} > .access-paging button`).nth(2).isEnabled());
    await page.locator(`${section} > .access-paging button`).nth(2).click();
    await page.waitForFunction((s) => document.querySelectorAll(`${s} > .access-requests tbody tr`).length === 1, section);
    check(`${at}: the last page of waiting requests gives the exact total`, lang === "ar" || (await page.locator(`${section} [data-waiting]`).innerText()).trim() === copy[lang].waiting(51));
    mode = "endedfail";
    await open(lang);
    check(`${at}: when only the ended read fails the summary says so and the waiting actions stay`, (await summary.innerText()).trim() === copy[lang].endedFailed && (await page.locator(`${section} > .access-requests [data-act]`).count()) >= 2 && (await page.locator(`${section} details.access-ended-requests .access-list-failure button`).count()) === 1);
    if (args.shots && width === 1440) { await summary.click(); await page.locator(section).screenshot({ path: `${args.shots}/m03-requests-ended-failure-${lang}-${width}.png` }); }
    mode = "pendingfail";
    await open(lang);
    check(`${at}: when only the waiting read fails it says so with Reload, shows no count, and the ended list still works`, (await page.locator(`${section} > .access-list-failure h3`).innerText()).trim() === copy[lang].failed && (await page.locator(`${section} [data-waiting]`).count()) === 0 && (await summary.innerText()).trim() === copy[lang].endedAtLeast(25) && (await page.locator(".access-permissions tbody tr").count()) > 0);
    if (args.shots && width === 1440) await page.locator(section).screenshot({ path: `${args.shots}/m03-requests-pending-failure-${lang}-${width}.png` });
    if (width === 1440 && lang === "en") {
      mode = "pair";
      await open(lang);
      await page.locator(`${section} tr[data-request="access-request-95"] [data-act="reject"]`).click();
      await page.locator(".live-panel form.live-step h2").waitFor();
      const panel = await page.locator(".live-panel form.live-step").innerText();
      check("the decision panel shows the row's own sign-in names", panel.includes("member-with-a-long-sign-in-name@example.invalid") && panel.includes("admin2@example.invalid"), panel.slice(0, 200));
      const narrow = await page.evaluate(() => ({ list: Math.round(document.querySelector(".access-request-section").getBoundingClientRect().width), dir: getComputedStyle(document.querySelector(".access-requests .access-actions")).flexDirection }));
      check("beside the panel the action buttons stack", narrow.list < 960 && narrow.dir === "column", JSON.stringify(narrow));
      const stacked = await page.locator(`${section} tr[data-request="access-request-95"] .access-actions button`).evaluateAll((bs) => bs.map((b) => Math.round(b.getBoundingClientRect().top)));
      check("beside the panel a row's two actions are one above the other", stacked.length === 2 && stacked[1] > stacked[0], JSON.stringify(stacked));
      await page.locator(`${section} .access-requests`).first().scrollIntoViewIfNeeded();
      await page.evaluate(() => document.querySelector(".access-request-section").scrollIntoView({ block: "start" }));
      await page.mouse.move(2, 2);
      if (args.shots) await page.screenshot({ path: `${args.shots}/m03-requests-panel-en-1440.png` });
    }
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
