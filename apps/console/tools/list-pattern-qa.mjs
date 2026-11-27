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
  en: { applied: "Applied filters", clear: "Clear filters", remove: "Remove filter", noMatch: "No matches for these filters.", size: "Rows per page", settings: "List settings", selected: "Selected" },
  ko: { applied: "적용된 필터", clear: "필터 모두 해제", remove: "필터 제거", noMatch: "이 필터와 일치하는 항목이 없습니다.", size: "페이지당 항목 수", settings: "목록 설정", selected: "선택됨" },
  ar: { applied: "عوامل التصفية المطبّقة", clear: "مسح جميع عوامل التصفية", remove: "إزالة عامل التصفية", noMatch: "لا توجد نتائج مطابقة لعوامل التصفية هذه.", size: "صفوف لكل صفحة", settings: "إعدادات القوائم", selected: "محدد" },
};
const isPermissions = (url) => new URL(url).pathname.endsWith("/identity/access/permissions");

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
  page.on("request", (r) => { if (isPermissions(r.url())) asked.push(new URL(r.url()).search); });
  const open = async (lang, query = "") => {
    asked = [];
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/access?${new URLSearchParams({ lang, view: "grants" })}${query}`);
    await page.locator(".access-list-body .list-summary, .access-list-body .access-list-failure").first().waitFor();
    await page.locator("#access-revocations").waitFor();
  };
  const focused = (sel) => page.evaluate((s) => !!document.activeElement && document.activeElement.matches(s), sel);
  const eventually = async (f) => { for (let i = 0; i < 60; i++) { if (await f()) return true; await page.waitForTimeout(50); } return false; };

  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 1024], ["en", 390], ["ko", 390], ["ar", 390]]) {
    const at = `${lang} ${width}`;
    await page.setViewportSize({ width, height: 1000 });
    await open(lang);
    await page.mouse.move(0, 0);
    const rows = await page.locator(".access-permissions tbody tr").count();
    check(`${at}: the summary line carries an exact count equal to the rows and no applied-filter row shows`, (await page.locator(".access-list-body .list-summary").getAttribute("data-count")) === String(rows) && rows > 0 && (await page.locator(".list-applied").count()) === 0, String(rows));
    check(`${at}: one page of results shows no paging but offers rows per page`, (await page.locator(".access-list-body .list-pager-nav").count()) === 0 && (await page.locator("#acc-page-size").count()) === 1 && (await page.locator(".access-list-body .list-pager label").innerText()).trim() === copy[lang].size);
    const options = await page.locator("#acc-page-size option").allInnerTexts();
    check(`${at}: the rows per page offer the sizes up to the installation's limit`, options.join(" ") === "10 25 50" && (await page.locator("#acc-page-size").inputValue()) === "50", options.join(" "));
    const settings = page.locator("details.list-settings");
    check(`${at}: the list settings disclosure sits after the lists, closed, with three settings`, (await settings.count()) === 1 && !(await settings.evaluate((d) => d.open)) && (await settings.locator("summary").innerText()).trim() === copy[lang].settings && (await settings.locator("tbody tr").count()) === 3);
    if (args.shots && width !== 1024) await page.locator(".access-list-body").screenshot({ path: `${args.shots}/list-m03-${lang}-${width}.png` });

    asked = [];
    await page.locator("#acc-page-size").selectOption("10");
    await page.waitForFunction(() => location.hash.includes("acc_limit=10"));
    await page.locator(".access-list-body .list-summary").waitFor();
    check(`${at}: choosing ten rows per page asks for ten, writes it to the address and keeps focus on the choice`, asked.some((q) => q.includes("limit=10")) && (await focused("#acc-page-size")) && (await page.locator("#acc-page-size").inputValue()) === "10", asked.join(" "));
    await page.locator("#acc-page-size").selectOption("50");
    await page.waitForFunction(() => !location.hash.includes("acc_limit"));

    if (width >= 1024) {
      await page.locator(".access-filters-toggle").isVisible().then(async (v) => { if (v) await page.locator(".access-filters-toggle").click(); });
    } else {
      await page.locator(".access-filters-toggle").click();
    }
    await page.locator("#acc-state").selectOption("active");
    await page.locator("#acc-subject").fill("sample-admin");
    asked = [];
    await page.locator(".access-apply").click();
    await page.locator(".list-applied").waitFor();
    const chips = await page.locator(".list-chip").evaluateAll((cs) => cs.map((c) => ({ key: c.dataset.chip, text: c.querySelector(".list-chip-text").textContent.trim(), name: c.querySelector("button").getAttribute("aria-label") })));
    check(`${at}: applied filters show as chips with named remove controls and a clear-all`, chips.map((c) => c.key).join(" ") === "subject state" && chips.every((c) => c.name.startsWith(copy[lang].remove)) && (await page.locator(".list-applied-lead").innerText()).trim() === copy[lang].applied && (await page.locator(".list-applied-clear button").innerText()).trim() === copy[lang].clear, JSON.stringify(chips));
    await page.mouse.move(0, 0);
    if (args.shots && width !== 1024) await page.locator(".access-list-body").screenshot({ path: `${args.shots}/list-m03-chips-${lang}-${width}.png` });
    const gaps = await page.evaluate(() => { const s = document.querySelector(".access-list-body .list-summary"); const lines = [...s.querySelectorAll("p:not(.visually-hidden)")].map((p) => p.getBoundingClientRect()); const table = document.querySelector(".access-list-body .access-permissions"); const chip = document.querySelector(".list-chip"); const clearB = document.querySelector(".list-applied-clear button"); return { between: lines.length > 1 && lines[1].top > lines[0].top + 1 ? lines[1].top - lines[0].bottom : 0, toTable: table.getBoundingClientRect().top - s.getBoundingClientRect().bottom, chip: chip.getBoundingClientRect().height, clear: clearB.getBoundingClientRect().height }; });
    check(`${at}: the summary lines sit close, the table follows the summary by about 8 px and chips match the button height`, gaps.between <= 6 && gaps.toTable >= 4 && gaps.toTable <= 12 && Math.abs(gaps.chip - gaps.clear) <= 1, JSON.stringify(gaps));
    const shown = await inspectView(page);
    check(`${at}: nothing overflows, is cut off or loses contrast with chips shown`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 3), cut: shown.cut.slice(0, 3), low: shown.low.slice(0, 3) }));
    asked = [];
    await page.locator('.list-chip[data-chip="subject"] .list-chip-remove').click();
    await page.waitForFunction(() => document.querySelectorAll(".list-chip").length === 1);
    check(`${at}: removing a chip applies at once and moves focus to the next chip`, (await eventually(() => focused('.list-chip[data-chip="state"] .list-chip-remove'))) && asked.some((q) => q.includes("state=active") && !q.includes("subject")), asked.join(" "));
    asked = [];
    await page.locator('.list-chip[data-chip="state"] .list-chip-remove').click();
    await page.waitForFunction(() => document.querySelectorAll(".list-applied").length === 0);
    check(`${at}: removing the last chip clears the filters and focus rests on the summary line`, (await eventually(() => focused(".list-summary"))) && asked.some((q) => !q.includes("state=")), asked.join(" "));

    await open(lang, "&acc_subject=nobody-at-all");
    check(`${at}: a filter with no match says so and offers to clear the filters`, (await page.locator(".list-no-match p").innerText()).trim() === copy[lang].noMatch && (await page.locator(".list-no-match button").innerText()).trim() === copy[lang].clear);
    await page.locator(".list-no-match button").click();
    await page.waitForFunction(() => !location.hash.includes("acc_subject"));
    check(`${at}: clearing from the no-match state lists every permission again and focus rests on the summary line`, (await eventually(async () => (await page.locator(".access-permissions tbody tr").count()) > 0)) && (await eventually(() => focused(".list-summary"))));
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/access?${new URLSearchParams({ lang, view: "grants" })}&acc_limit=0`);
    await page.locator(".access-list-body [role=alert]").waitFor();
    await page.locator(".access-list-body button").first().click();
    await page.waitForFunction(() => !location.hash.includes("acc_limit"));
    check(`${at}: an invalid rows-per-page value in the address is cleared by Reset`, await eventually(async () => (await page.locator(".access-permissions tbody tr").count()) > 0));

    const revoke = page.locator(".access-permissions .live-revoke").first();
    if (width >= 1280 && await revoke.count()) {
      await revoke.click();
      await page.waitForFunction(() => document.querySelector('.access-permissions tr[aria-current="true"]'));
      check(`${at}: the selected row is marked by aria-current and a visible tag, not by colour alone`, (await page.locator('.access-permissions tr[aria-current="true"] .list-tag').innerText()).trim() === copy[lang].selected);
    } else notes.push(`${at}: the selected tag was not exercised (${width < 1280 ? "the panel replaces the list below 1280" : "no revoke button on the real rows"})`);
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
