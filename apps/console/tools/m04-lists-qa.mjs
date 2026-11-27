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
const results = [], errors = [];
let navigation = 0;
const check = (name, ok, detail = "") => { results.push(`${ok ? "PASS" : "FAIL"} ${name}${!ok && detail ? " " + String(detail).replace(/\s+/g, " ").slice(0, 400) : ""}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const copy = {
  en: { size: "Rows per page", search: "Search", toggle: (n) => `Filters (${n} applied)` },
  ko: { size: "페이지당 항목 수", search: "검색", toggle: (n) => `필터(${n}개 적용)` },
  ar: { size: "صفوف لكل صفحة", search: "بحث", toggle: (n) => `عوامل التصفية (المطبّق: ${n})` },
};
const eventually = async (f) => { for (let i = 0; i < 80; i++) { if (await f()) return true; await page.waitForTimeout(50); } return false; };
const hashParams = () => page.evaluate(() => Object.fromEntries(new URLSearchParams(location.hash.split("?")[1])));

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

  for (const [view, pfx, id, sizeId] of [["accounts", "svc_a_", "svc-a-filters", "svc-a-size"], ["credentials", "svc_c_", "svc-c-filters", "svc-c-size"]]) {
    for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 390], ["ar", 390]]) {
      const at = `${view} ${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      await page.goto("about:blank");
      await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/service-accounts?lang=${lang}&view=${view}`);
      const filters = page.locator(`#${id}`);
      await filters.waitFor({ state: "attached" });
      const area = filters.locator("xpath=..");
      await area.locator(".list-summary").first().waitFor();
      const order = await area.evaluate((s) => { const sum = s.querySelector(".list-summary"), body = s.querySelector(".svc-list-body table") ?? s.querySelector(".svc-list-body > .live-empty"), pager = s.querySelector(".list-pager"); const empty = !!s.querySelector(".svc-list-body > .live-empty"); return !!sum && !!body && !!pager && !!(sum.compareDocumentPosition(body) & 4) && !!(body.compareDocumentPosition(pager) & 4) && (!empty || !s.querySelector(".svc-list-body table")); });
      check(`${at}: the summary line stands above the table (or, when empty, the empty sentence in its place) and the pager below it`, order);
      check(`${at}: the filters ask for no page size`, (await filters.locator("input[inputmode=numeric]").count()) === 0 && (await filters.locator("label").allInnerTexts()).every((t) => t.trim() !== copy[lang].size));
      check(`${at}: rows per page is the pager's choice, labelled, at the default 25`, (await page.locator(`#${sizeId}`).inputValue()) === "25" && (await page.locator(`label[for=${sizeId}]`).innerText()).trim() === copy[lang].size);
      const toggle = filters.locator(".list-filters-toggle");
      if (width < 600) check(`${at}: at a narrow width the filters start collapsed behind one control`, (await toggle.isVisible()) && (await toggle.getAttribute("aria-expanded")) === "false" && !(await page.locator(`#${id}-form`).isVisible()) && (await toggle.innerText()).trim() === copy[lang].toggle(0));
      else check(`${at}: at a wide width the filters show without a toggle`, !(await toggle.isVisible()) && (await page.locator(`#${id}-form`).isVisible()));
      const times = await area.locator(".list-summary time").evaluateAll((all) => all.map((x) => getComputedStyle(x).direction));
      check(`${at}: list times follow the page direction`, times.length >= 2 && times.every((d) => d === (lang === "ar" ? "rtl" : "ltr")), times.join(" "));
      const shown = await inspectView(page);
      check(`${at}: nothing overflows, is cut off or loses contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 3), cut: shown.cut.slice(0, 3), low: shown.low.slice(0, 3) }));
      if (args.shots) { await page.mouse.move(0, 0); await area.screenshot({ path: `${args.shots}/m04-${view}-${lang}-${width}.png` }); }

      await page.locator(`#${sizeId}`).selectOption("10");
      await eventually(async () => (await hashParams())[pfx + "limit"] === "10");
      await area.locator(".list-summary").first().waitFor();
      check(`${at}: ten rows per page goes into the address from the first page and focus stays on the choice`, (await hashParams())[pfx + "after"] === undefined && (await eventually(() => page.evaluate((s) => document.activeElement?.id === s, sizeId))));

      if (await toggle.isVisible() && (await toggle.getAttribute("aria-expanded")) === "false") await toggle.click();
      await page.locator(`#${id}-q`).fill("sample");
      await page.locator(`#${id}-form button[type=submit]`).click();
      await eventually(async () => (await hashParams())[pfx + "q"] === "sample");
      await area.locator(".list-chip").first().waitFor();
      const chips = await area.locator(".list-chip").allInnerTexts();
      check(`${at}: the applied search shows as one chip and rows per page stays`, chips.length === 1 && chips[0].includes(copy[lang].search) && chips[0].includes("sample") && (await hashParams())[pfx + "limit"] === "10", chips.join(" | "));
      if (width < 600) check(`${at}: after Apply the filters collapse and say one is applied`, (await toggle.getAttribute("aria-expanded")) === "false" && (await toggle.innerText()).trim() === copy[lang].toggle(1));
      if (args.shots) { await page.mouse.move(0, 0); await area.screenshot({ path: `${args.shots}/m04-${view}-chips-${lang}-${width}.png` }); }
      await area.locator(".list-chip-remove").click();
      await eventually(async () => (await hashParams())[pfx + "q"] === undefined);
      const focused = await eventually(() => page.evaluate(() => { const a = document.activeElement; const r = a?.getBoundingClientRect(); return !!r && r.width > 8 && r.height > 8 && (a.classList.contains("list-filters-title") || a.classList.contains("list-filters-toggle")); }));
      check(`${at}: removing the chip drops the search, keeps rows per page and leaves focus on a visible filter control`, (await hashParams())[pfx + "limit"] === "10" && (await area.locator(".list-chip").count()) === 0 && focused);
      await page.locator(`#${id}-form button[type=button]`).click().catch(async () => { await toggle.click(); await page.locator(`#${id}-form button[type=button]`).click(); });
      check(`${at}: Reset filters keeps rows per page`, (await hashParams())[pfx + "limit"] === "10");
    }
  }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto("about:blank");
  await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#/service-accounts?lang=en&view=accounts&svc_a_audience=test-service-api&svc_a_scope=read&svc_a_limit=10`);
  await page.locator("#svc-a-filters-applied").waitFor();
  check("accounts: audience and scope show as two chips", (await page.locator("#svc-a-filters ~ .list-applied .list-chip, .list-applied .list-chip").count()) === 2);
  await page.locator('.list-chip[data-chip="audience"] .list-chip-remove').click();
  await eventually(async () => (await hashParams()).svc_a_audience === undefined);
  const after = await hashParams();
  await page.locator(".list-summary").first().waitFor();
  check("accounts: removing the audience chip also drops its scope, so the list stays valid and keeps rows per page", after.svc_a_scope === undefined && after.svc_a_limit === "10" && (await page.getByText("This filter or selection is invalid. Reset it to continue.").count()) === 0 && (await page.locator(".list-applied .list-chip").count()) === 0 && (await page.locator("#svc-a-filters").count()) === 1, JSON.stringify(after) + " alerts " + JSON.stringify(await page.locator("[role=alert]").allInnerTexts()) + " chips " + (await page.locator(".list-applied .list-chip").count()));
  for (const lang of ["en", "ar"]) {
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/service-accounts?lang=${lang}&view=credentials`);
    const picker = page.locator("fieldset").filter({ has: page.locator(".svc-picker-search") }).first();
    await picker.waitFor();
    await picker.locator(".svc-list-body").waitFor();
    const box = await picker.evaluate((f) => { const row = f.querySelector(".svc-picker-search"), input = row.querySelector("input"), button = row.querySelector("button"), select = f.querySelector(".svc-list-body > select"); const a = input.getBoundingClientRect(), b = button.getBoundingClientRect(); return { row: row.getBoundingClientRect().width, sameLine: Math.abs(a.top - b.top) < 12, button: b.width, select: select?.getBoundingClientRect().width ?? 0 }; });
    check(`${lang} 1440: the picker's search input and button share one row no wider than 560 px, and its select is capped`, box.row <= 561 && box.sameLine && box.button < 200 && box.select > 0 && box.select <= 321, JSON.stringify(box));
    if (args.shots) { await page.mouse.move(0, 0); await picker.screenshot({ path: `${args.shots}/m04-picker-${lang}-1440.png` }); await page.locator("section[aria-labelledby=svc-cred-title]").screenshot({ path: `${args.shots}/m04-credentials-full-${lang}-1440.png` }); }
  }
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, (e.stack ?? String(e)) + " URL " + page.url());
} finally {
  await browser.close();
  const failed = results.filter((r) => r.startsWith("FAIL"));
  console.log((args.quiet ? failed : results).join("\n"));
  console.log(`RESULT ${results.length - failed.length} passed ${failed.length} failed`);
}
