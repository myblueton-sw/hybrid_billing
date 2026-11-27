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
  en: { title: "Delegations that would stop applying", count: (n) => (n === 1 ? "1 delegation" : `${n} delegations`), none: "No delegation that applies now would stop applying.", more: (n) => `Show ${n} more`, loading: "Checking which delegations would stop applying…", failed: "could not be checked", hint: "Choices are limited to what the console itself may do" },
  ko: { title: "함께 적용되지 않게 되는 위임", count: (n) => `${n}건`, none: "지금 적용 중인 위임 중 함께 적용되지 않게 되는 위임은 없습니다.", more: (n) => `${n}건 더 보기`, loading: "함께 적용되지 않게 되는 위임을 확인하는 중…", failed: "확인하지 못했습니다", hint: "콘솔 자체가 할 수 있는 범위로 제한됩니다" },
  ar: { title: "تفويضات ستتوقف عن السريان", count: (n) => (n === 2 ? "تفويضان" : `${n} تفويضًا`), none: "لا يوجد تفويض سارٍ الآن سيتوقف عن السريان.", more: (n) => `عرض ${n} أخرى`, loading: "جارٍ التحقق من التفويضات التي ستتوقف عن السريان…", failed: "تعذّر التحقق", hint: "تقتصر الخيارات على ما يُسمح لوحدة التحكم نفسها بفعله" },
};
const every = { any: true, id: "" };
const base0 = { subject_login: "", client: "", delegator: "", delegator_login: "", revocation_requested: false, parent: "", redelegate: false, action: "view", kind: "focus_data", fields: ["rows"], scope: { customer: every, legal_entity: every, contract: every, period: "any" }, not_before: "2026-11-01T00:00:00Z", not_after: "2026-12-26T00:00:00Z", state: "active", effective: true, version: 1 };
const synthetic = [
  { ...base0, source: "grant", id: "a-console", subject: "", client: "console" },
  { ...base0, source: "grant", id: "b-person", subject: "sample-admin2", subject_login: "admin2@example.invalid", scope: { ...base0.scope, customer: { any: false, id: "customer-with-a-long-identifier-0001" }, contract: { any: false, id: "contract-7" }, period: "24322-24324" } },
  { ...base0, source: "grant", id: "c-own", subject: "sample-admin", subject_login: "admin@example.invalid" },
  { ...base0, source: "grant", id: "d-asked", subject: "sample-member", subject_login: "member@example.invalid", revocation_requested: true },
  { ...base0, source: "grant", id: "e-expired", subject: "sample-member", subject_login: "member@example.invalid", state: "expired", effective: false, not_after: "2026-11-10T00:00:00Z" },
  { ...base0, source: "grant", id: "f-revoked", subject: "sample-member", subject_login: "member@example.invalid", state: "revoked", effective: false },
  { ...base0, source: "grant", id: "g-inapplicable", subject: "left-user", subject_login: "", effective: false },
  { ...base0, source: "delegation", id: "h-first", subject: "sample-member", subject_login: "member@example.invalid", delegator: "sample-admin", delegator_login: "admin@example.invalid", redelegate: true },
  { ...base0, source: "delegation", id: "i-second", subject: "sample-admin2", subject_login: "admin2@example.invalid", delegator: "sample-member", delegator_login: "member@example.invalid", parent: "h-first" },
  { ...base0, source: "delegation", id: "j-ended", subject: "sample-admin2", subject_login: "admin2@example.invalid", delegator: "sample-member", delegator_login: "member@example.invalid", state: "ended", effective: false },
];
const echo = (qs) => ({ q: (qs.get("q") ?? "").toLowerCase(), subject: qs.get("subject") ?? "", action: qs.get("action") ?? "", customer: qs.get("customer") ?? "", legal_entity: qs.get("legal_entity") ?? "", source: qs.get("source") ?? "", state: qs.get("state") ?? "", expires_before: qs.get("expires_before") ?? "", limit: Number(qs.get("limit") ?? 25) });
const pageBody = (qs, rows, change = {}) => ({ rows, search_input: qs.get("q") ?? "", query: echo(qs), next: "", reference_at: "2026-11-26T08:45:18.206Z", observed_at: "2026-11-26T08:45:18.211Z", revision: "r1", order: "source_then_id", count: { value: String(rows.length), certainty: "exact" }, work_units: rows.length, scan_stopped: false, ...change });
const isList = (url) => new URL(url).pathname.endsWith("/identity/access/permissions");
const isDetail = (url) => /\/identity\/access\/permissions\/(grant|delegation)\//.test(new URL(url).pathname);
const twelve = Array.from({ length: 12 }, (_, i) => `del-below-${String(i + 1).padStart(2, "0")}`);

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
  page.on("request", (r) => { if (isDetail(r.url())) asked.push(new URL(r.url()).pathname.replace("/api/v1/identity/access/permissions/", "")); });
  const open = async (lang) => {
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/access?${new URLSearchParams({ lang, view: "grants" })}`);
    await page.locator("#access-list").waitFor();
    await page.locator(".list-summary, .access-list-failure").first().waitFor();
  };
  const block = ".access-would-stop";

  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("en");
  const widths = await page.evaluate(() => ({ list: document.querySelector(".live-users > .live-list").getBoundingClientRect().width, all: document.querySelector('.live-users[data-live="grants"]').getBoundingClientRect().width }));
  check("at 1440 the list takes the full width until a panel is asked for", (await page.locator("#access-panel").count()) === 0 && Math.abs(widths.list - widths.all) <= 2 && (await page.locator(".access-grant-open").getAttribute("aria-expanded")) === "false", JSON.stringify(widths));
  await page.locator(".access-grant-open").click();
  await page.locator("#access-capability-hint").waitFor();
  check("opening the request form puts focus on its heading, offers Close and marks Request access expanded", (await page.evaluate(() => document.activeElement?.matches("#access-panel h2"))) && (await page.locator("#access-panel .access-panel-close button").count()) === 1 && (await page.locator(".access-grant-open").getAttribute("aria-expanded")) === "true");
  await page.locator("#access-capability").focus();
  await page.locator(".access-grant-open").click();
  check("Request access while the form is open moves focus back to the form's heading", await page.evaluate(() => document.activeElement?.matches("#access-panel h2")));
  for (const [lang, width] of [["en", 1440], ["ar", 1440], ["en", 1280], ["ar", 1280], ["ko", 1280]]) {
    await page.setViewportSize({ width, height: 1000 });
    await open(lang);
    await page.locator(".access-grant-open").click();
    await page.locator("#access-panel h2").waitFor();
    await page.mouse.move(0, 0);
    const box = await page.evaluate(() => { const h = document.querySelector("#access-panel h2").getBoundingClientRect(), c = document.querySelector("#access-panel .access-panel-close button").getBoundingClientRect(), p = document.querySelector("#access-panel").getBoundingClientRect(), r = document.documentElement.dir === "rtl"; const range = document.createRange(); range.selectNodeContents(document.querySelector("#access-panel h2")); const text = [...range.getClientRects()]; const clear = text.every((t) => r ? t.left >= c.right - 1 : t.right <= c.left + 1); return { clear, top: Math.abs(c.top - h.top), panel: Math.round(p.width) }; });
    check(`${lang} ${width}: Close sits on the heading's first line without touching its text, and the panel is about a third wide`, box.clear && box.top <= 4 && box.panel >= 360 && box.panel <= 444, JSON.stringify(box));
    if (args.shots) await page.screenshot({ path: `${args.shots}/m03-panel-${lang}-${width}.png` });
  }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("en");
  await page.locator(".access-grant-open").click();
  await page.locator("#access-capability-hint").waitFor();
  await page.locator("#access-panel .access-panel-close button").click();
  await page.locator("#access-panel").waitFor({ state: "detached" });
  check("Close removes the panel and returns focus to Request access", await page.evaluate(() => document.activeElement?.classList.contains("access-grant-open")));
  await page.locator(".access-grant-open").click();
  await page.locator("#access-capability-hint").waitFor();
  check("the console ceiling section is gone and the request form carries the hint about the console's own limits", (await page.locator("#access-ceiling").count()) === 0 && (await page.locator("#access-capability-hint").innerText()).includes(copy.en.hint) && (await page.locator("#access-capability").getAttribute("aria-describedby")).includes("access-capability-hint"));
  const real = page.locator(".access-permissions .live-revoke[data-grant]").first();
  if (await real.count()) {
    const id = await real.getAttribute("data-grant");
    await real.click();
    await page.locator(`${block}[data-would-stop="ready"]`).waitFor();
    notes.push(`the detail was read ${asked.length} time(s) for one opened panel (the development build runs effects twice)`);
    check("on the real service, asking to revoke a grant reads only that grant's detail and shows the block with its checked time", asked.length >= 1 && asked.every((p) => p === `grant/${id}`) && (await page.locator(`${block} h3`).innerText()).trim() === copy.en.title && (await page.locator(`${block} .live-meta`).last().innerText()).startsWith("Checked at") && ((await page.locator(`${block} .live-note`).count()) === 2 || (await page.locator(`${block} .access-stop-list li`).count()) > 0), asked.join(" "));
  } else notes.push("the sample has no grant that can be asked to be revoked");

  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 390], ["ko", 390], ["ar", 390]]) {
    const at = `${lang} ${width}`;
    await page.setViewportSize({ width, height: 1000 });
    let mode = "ready";
    const held = [];
    await page.route((url) => isList(url.toString()), (route) => { const qs = new URL(route.request().url()).searchParams; return route.fulfill({ status: 200, json: pageBody(qs, synthetic) }); });
    await page.route((url) => isDetail(url.toString()), async (route) => {
      const [, source, id] = new URL(route.request().url()).pathname.match(/permissions\/(grant|delegation)\/(.+)$/);
      if (mode === "fail") return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
      if (mode === "slow") await new Promise((done) => { held.push(done); });
      const row = synthetic.find((r) => r.source === source && r.id === id);
      const would = mode === "none" ? [] : source === "grant" ? ["h-first", "i-second"] : twelve;
      return route.fulfill({ status: 200, json: { workspace: "sample-workspace", row, would_end: would, revision: "r1", observed_at: "2026-11-26T08:45:18.211Z" } });
    });
    await open(lang);
    await page.locator('[data-grant="b-person"]').click();
    await page.locator(`${block}[data-would-stop="ready"]`).waitFor();
    check(`${at}: a grant's panel lists the two delegations that would stop, with a count and the checked time`, (await page.locator(`${block} h3`).innerText()).trim() === copy[lang].title && (await page.locator(`${block} [data-stop-count]`).innerText()).trim() === copy[lang].count(2) && (await page.locator(`${block} .access-stop-list li`).allInnerTexts()).join(",") === "h-first,i-second" && (await page.locator(`${block} details`).count()) === 0);
    const submit = page.locator(".live-panel form.live-step button[type=submit]");
    if (args.shots) await page.locator(".live-panel").screenshot({ path: `${args.shots}/m03-stop-grant-${lang}-${width}.png` });
    await page.locator(".live-panel .live-buttons .live-secondary, .live-panel .live-back").first().click();
    await page.locator('[data-delegation="h-first"]').click();
    await page.locator(`${block}[data-would-stop="ready"]`).waitFor();
    check(`${at}: a delegation's panel shows ten of twelve and the rest behind a disclosure`, (await page.locator(`${block} > .access-stop-list li`).count()) === 10 && (await page.locator(`${block} details summary`).innerText()).trim() === copy[lang].more(2) && (await page.locator(`${block} details li`).count()) === 2 && (lang === "ar" || (await page.locator(`${block} [data-stop-count]`).innerText()).trim() === copy[lang].count(12)));
    const shown = await inspectView(page);
    check(`${at}: nothing overflows, is cut off or loses contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), low: shown.low.slice(0, 2) }));
    if (args.shots) await page.locator(".live-panel").screenshot({ path: `${args.shots}/m03-stop-delegation-${lang}-${width}.png` });
    await page.locator(".live-panel .live-buttons .live-secondary, .live-panel .live-back").first().click();
    mode = "none";
    await page.locator('[data-grant="g-inapplicable"]').click();
    await page.locator(`${block}[data-would-stop="ready"]`).waitFor();
    check(`${at}: nothing would stop: the block says so and shows no count`, (await page.locator(`${block} .live-note`).last().innerText()).trim() === copy[lang].none && (await page.locator(`${block} [data-stop-count]`).count()) === 0);
    await page.locator(".live-panel .live-buttons .live-secondary, .live-panel .live-back").first().click();
    mode = "slow";
    await page.locator('[data-grant="b-person"]').click();
    await page.locator(`${block}[data-would-stop="loading"]`).waitFor();
    check(`${at}: while the read is pending the block says it is checking and the submit button is usable`, (await page.locator(`${block} [role=status]`).innerText()).trim() === copy[lang].loading && await submit.isEnabled());
    if (args.shots && lang === "en") await page.locator(".live-panel").screenshot({ path: `${args.shots}/m03-stop-loading-${lang}-${width}.png` });
    mode = "ready"; for (const done of held.splice(0)) done();
    await page.locator(`${block}[data-would-stop="ready"]`).waitFor();
    await page.locator(".live-panel .live-buttons .live-secondary, .live-panel .live-back").first().click();
    mode = "fail";
    await page.locator('[data-grant="b-person"]').click();
    await page.locator(`${block}[data-would-stop="error"]`).waitFor();
    check(`${at}: a failed read says revoking still works, offers another try, and leaves the submit button usable`, (await page.locator(`${block} [role=alert]`).innerText()).includes(copy[lang].failed) && (await page.locator(`${block} [role=alert] button`).count()) === 1 && await submit.isEnabled());
    if (args.shots && width === 1440) await page.locator(".live-panel").screenshot({ path: `${args.shots}/m03-stop-failure-${lang}-${width}.png` });
    mode = "ready";
    await page.locator(`${block} [role=alert] button`).click();
    await page.locator(`${block}[data-would-stop="ready"]`).waitFor();
    check(`${at}: trying again reads the detail again and shows the list`, (await page.locator(`${block} .access-stop-list li`).count()) === 2);
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
