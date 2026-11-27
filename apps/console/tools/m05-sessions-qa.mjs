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
  en: { exact: "Matching count", size: "Rows per page", search: "Session reference or time", toggle: (n) => `Filters (${n} applied)` },
  ko: { exact: "일치 항목 수", size: "페이지당 항목 수", search: "세션 참조 또는 시각", toggle: (n) => `필터(${n}개 적용)` },
  ar: { exact: "عدد النتائج المطابقة", size: "صفوف لكل صفحة", search: "مرجع الجلسة أو الوقت", toggle: (n) => `عوامل التصفية (المطبّق: ${n})` },
};
const isList = (url) => new URL(url).pathname.endsWith("/me/sessions");
const total = 60;
const eventually = async (f) => { for (let i = 0; i < 80; i++) { if (await f()) return true; await page.waitForTimeout(50); } return false; };

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
  let defaultLimit = 0;
  await page.route((url) => isList(url.toString()), async (route) => {
    const qs = new URL(route.request().url()).searchParams;
    asked.push(new URL(route.request().url()).search);
    const bare = new URL(route.request().url()); bare.search = "";
    const real = await (await route.fetch({ url: bare.toString() })).json();
    defaultLimit = defaultLimit || real.query.limit;
    const mine = real.rows.find((r) => r.current) ?? real.rows[0];
    const others = Array.from({ length: total - 1 }, (_, i) => ({ ...mine, current: false, can_end: true, not_included_operation: "", handle: `qa-session-${String(i + 1).padStart(2, "0")}` }));
    const q = (qs.get("q") ?? "").toLowerCase();
    const all = [mine, ...others].filter((r) => !q || r.handle.toLowerCase().includes(q));
    const limit = Number(qs.get("limit") ?? real.query.limit);
    const start = qs.get("after") ? Number(qs.get("after").slice(1)) : 0;
    const rows = all.slice(start, start + limit);
    const next = start + limit < all.length ? "p" + (start + limit) : "";
    await route.fulfill({ status: 200, json: { ...real, rows, next, query: { ...real.query, q: qs.get("q") ?? "", limit }, count: { value: all.length, certainty: "exact" } } });
  });

  const section = "section[aria-labelledby='mysec-sessions-title']";
  const loaded = () => page.waitForFunction((s) => !document.querySelector(`${s} > p[role=status]`) && !!document.querySelector(`${s} .list-summary`), section);
  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 1024], ["en", 390], ["ar", 390]]) {
    const at = `${lang} ${width}`;
    await page.setViewportSize({ width, height: 1000 });
    asked = [];
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/my-security?lang=${lang}&view=sessions`);
    await page.locator(section).waitFor();
    await loaded();
    const order = await page.locator(section).evaluate((s) => { const sum = s.querySelector(".list-summary"), table = s.querySelector("table"), pager = s.querySelector(".list-pager"); return !!sum && !!table && !!pager && !!(sum.compareDocumentPosition(table) & 4) && !!(table.compareDocumentPosition(pager) & 4); });
    check(`${at}: the summary line stands above the table and the pager below it`, order);
    const summary = (await page.locator(`${section} .list-summary`).innerText()).trim();
    check(`${at}: the summary line states the exact count and a reference time`, summary.includes(`${copy[lang].exact}: ${total}`) && (await page.locator(`${section} .list-summary time`).count()) === 1, summary);
    check(`${at}: the filter form no longer asks for a page size`, (await page.locator(`${section} input[type=number]`).count()) === 0);
    const sizeNow = await page.locator("#mysec-size").inputValue();
    check(`${at}: rows per page is a choice under the table set to the installation default`, Number(sizeNow) === defaultLimit && (await page.locator(`${section} .list-pager label[for=mysec-size]`).innerText()).trim() === copy[lang].size, sizeNow + " " + defaultLimit);
    const toggle = page.locator(`${section} .list-filters-toggle`);
    if (width < 600) check(`${at}: at a narrow width the filters start collapsed behind one control`, (await toggle.isVisible()) && (await toggle.getAttribute("aria-expanded")) === "false" && !(await page.locator("#mysec-filter-form").isVisible()) && (await toggle.innerText()).trim() === copy[lang].toggle(0));
    else check(`${at}: at a wide width the filters show without a toggle`, !(await toggle.isVisible()) && (await page.locator("#mysec-filter-form").isVisible()));
    const shown = await inspectView(page);
    check(`${at}: nothing overflows, is cut off or loses contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 3), cut: shown.cut.slice(0, 3), low: shown.low.slice(0, 3) }));
    if (args.shots) { await page.mouse.move(0, 0); await page.locator(section).screenshot({ path: `${args.shots}/m05-sessions-${lang}-${width}.png` }); }
    const dates = await page.locator(`${section} td[data-label] time, .mysec-current time`).evaluateAll((all) => all.map((x) => getComputedStyle(x).direction));
    check(`${at}: local session dates follow the page direction instead of being forced left to right`, dates.length > 2 && dates.every((d) => d === (lang === "ar" ? "rtl" : "ltr")), dates.slice(0, 4).join(" "));

    const nav = page.locator(`${section} .list-pager-nav button`);
    check(`${at}: on the first page First and Previous are disabled and Next is enabled`, (await nav.count()) === 3 && await nav.nth(0).isDisabled() && await nav.nth(1).isDisabled() && await nav.nth(2).isEnabled());
    asked = [];
    await nav.nth(2).click();
    await eventually(async () => asked.some((q) => q.includes("after=")));
    await loaded();
    check(`${at}: Next asks for the following page and moves focus to the summary line`, asked.some((q) => q.includes("after=p" + defaultLimit)) && (await eventually(() => page.evaluate(() => document.activeElement?.classList.contains("list-summary")))), asked.join(" "));
    check(`${at}: on the second page Previous is enabled`, await nav.nth(1).isEnabled());
    await nav.nth(1).click();
    await loaded();
    check(`${at}: Previous returns to the first page`, await nav.nth(1).isDisabled());

    asked = [];
    await page.locator("#mysec-size").selectOption("10");
    await eventually(async () => asked.some((q) => q.includes("limit=10")));
    await loaded();
    check(`${at}: ten rows per page asks for ten from the first page, keeps it in the address and keeps focus on the choice`, asked.some((q) => q.includes("limit=10") && !q.includes("after=")) && page.url().includes("slimit=10") && (await page.locator(`${section} tbody tr`).count()) === 10 && (await eventually(() => page.evaluate(() => document.activeElement?.id === "mysec-size"))), asked.join(" "));

    if (await toggle.isVisible() && (await toggle.getAttribute("aria-expanded")) === "false") await toggle.click();
    asked = [];
    await page.locator("#mysec-f-q").fill("qa-session-1");
    await page.locator("#mysec-filter-form button[type=submit]").click();
    await eventually(async () => asked.some((q) => q.includes("q=qa-session-1")));
    await loaded();
    const chips = await page.locator(`${section} ~ * .list-chip, ${section} .list-chip`).allInnerTexts();
    check(`${at}: the applied search shows as one chip, keeps the chosen rows per page and the count follows`, chips.length === 1 && chips[0].includes(copy[lang].search) && chips[0].includes("qa-session-1") && asked.some((q) => q.includes("limit=10")) && (await page.locator(`${section} .list-summary`).innerText()).includes(`${copy[lang].exact}: 10`), chips.join(" | ") + " " + asked.join(" ") + " " + (await page.locator(`${section} .list-summary`).innerText()));
    if (width < 600) check(`${at}: after Apply the filters collapse and the toggle says one is applied`, (await toggle.getAttribute("aria-expanded")) === "false" && (await toggle.innerText()).trim() === copy[lang].toggle(1));
    if (args.shots) { await page.mouse.move(0, 0); await page.locator(section).screenshot({ path: `${args.shots}/m05-sessions-chips-${lang}-${width}.png` }); }
    asked = [];
    await page.locator(`${section} .list-chip-remove`).click();
    await eventually(async () => asked.length > 0);
    await loaded();
    const focused = await page.evaluate(() => { const a = document.activeElement; const r = a?.getBoundingClientRect(); return !!r && r.width > 8 && r.height > 8 && (a.id === "mysec-filters-title" || a.classList.contains("list-filters-toggle")); });
    check(`${at}: removing the chip reads without the search, keeps rows per page and leaves focus on a visible filter control`, asked.every((q) => !q.includes("q=")) && asked.some((q) => q.includes("limit=10")) && (await page.locator(`${section} .list-chip`).count()) === 0 && focused, asked.join(" "));
  }
  for (const [lang, width] of [["en", 390], ["ar", 1440]]) {
    await page.setViewportSize({ width, height: 1000 });
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/my-security?lang=${lang}&view=sessions&ssince=2026-11-20T10:00:00Z&suntil=2026-11-20T09:00:00Z&slimit=10`);
    await page.locator("#mysec-filter-error").waitFor();
    check(`${lang} ${width}: a link whose start is not before its end shows the invalid-filter alert even with the filters collapsed`, await page.locator("#mysec-filter-error").isVisible());
    await page.locator("#mysec-filters-title").waitFor({ state: "attached" });
    const toggle = page.locator("section[aria-labelledby='mysec-sessions-title'] .list-filters-toggle");
    if (await toggle.isVisible()) await toggle.click();
    await page.locator("#mysec-filter-form button[type=button]").click();
    await page.waitForFunction(() => location.hash.includes("slimit=10") && !location.hash.includes("ssince"));
    check(`${lang} ${width}: Reset filters clears the times and keeps rows per page`, page.url().includes("slimit=10") && !page.url().includes("ssince"));
  }
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, (e.stack ?? String(e)) + " URL " + page.url() + " TEXT " + (await page.locator("main, body").first().innerText().catch(() => "")).slice(0, 300));
} finally {
  await browser.close();
  const failed = results.filter((r) => r.startsWith("FAIL"));
  console.log((args.quiet ? failed : results).join("\n"));
  console.log(`RESULT ${results.length - failed.length} passed ${failed.length} failed`);
}
