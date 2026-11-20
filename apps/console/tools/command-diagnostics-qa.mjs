import { createRequire } from "node:module";
import { readFile, mkdir } from "node:fs/promises";
import { randomBytes } from "node:crypto";
const args = Object.fromEntries(process.argv.slice(2).flatMap((v, i, all) => v.startsWith("--") ? [[v.slice(2), all[i + 1]]] : []));
const { chromium } = createRequire(import.meta.url)(args["playwright-module"] || "playwright");
const base = (args.url || "http://127.0.0.1:4173").replace(/\/$/, "");
const browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
const page = await context.newPage();
const results = [], errors = [];
let navigation = 0;
const check = (name, ok) => { results.push(`${ok ? "PASS" : "FAIL"} ${name}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const issue = "**/api/v1/commands/integrations.plugin_stop";
const diagnostic = () => page.locator(".command-diagnostics");
const missing = { en: "Not available", ko: "확인할 수 없음", ar: "غير متاح" };
const hash = async (path) => page.evaluate((value) => { location.hash = value; }, path);
const settled = async (selector) => {
  await page.locator(selector).waitFor();
  await page.waitForFunction((selector) => document.querySelector(selector)?.getAttribute("aria-busy") !== "true", selector);
};
async function stop() {
  await page.locator('[data-opener$="|integrations.plugin_stop"]').first().click();
  await page.locator(".p02-form textarea").fill("synthetic diagnostics check");
  await page.locator(".p02-form button[type=submit]").click();
  await page.locator(".p02-confirm .live-primary").click();
}
async function openFailures(lang) {
  const approve = "**/api/v1/commands/integrations.plugin_approve";
  await page.route(issue, (route) => route.fulfill({ status: 429, contentType: "application/json", headers: { "Retry-After": "3" }, body: JSON.stringify({ code: "rate_limited", request_id: "open-stop-330" }) }));
  await stop();
  const first = page.locator(".p02-form-wrap").filter({ has: page.locator('textarea[id^="p02-plugin_stop-"]') });
  await first.locator('textarea').waitFor();
  await page.route(approve, (route) => route.fulfill({ status: 422, contentType: "application/json", body: JSON.stringify({ code: "invalid_request", fields: ["rationale"], request_id: "open-approve-330" }) }));
  const proposalId = await page.locator("li.p02-proposal").filter({ has: page.locator('[data-opener$="|integrations.plugin_approve"]') }).first().getAttribute("id");
  const proposal = page.locator("#" + proposalId);
  await proposal.locator('[data-opener$="|integrations.plugin_approve"]').click();
  await proposal.locator(".p02-form textarea").fill("synthetic second open form");
  await proposal.locator(".p02-form button[type=submit]").click(); await proposal.locator(".p02-confirm .live-primary").click();
  await proposal.locator(".service-summary").waitFor();
  check(`${lang} first open form retains its own rate-limit references`, (await first.innerText()).includes("open-stop-330") && (await first.innerText()).includes("rate_limited"));
  check(`${lang} second open form field error has its own references`, (await proposal.locator(".p02-form-wrap").innerText()).includes("open-approve-330") && (await proposal.locator(".p02-form-wrap").innerText()).includes("invalid_request"));
  await first.locator(".p02-form .service-button-link").click(); await proposal.locator(".p02-form .service-button-link").click();
  await page.unroute(issue); await page.unroute(approve);
}
async function retentionFailures(lang) {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/retention?lang=${lang}&view=policies`);
  await settled(".p04-live");
  const pattern = "**/api/v1/commands/retention.*";
  let requests = 0;
  await page.route(pattern, (route) => route.fulfill({ status: 429, contentType: "application/json", headers: { "Retry-After": "3" }, body: JSON.stringify({ code: "rate_limited", request_id: `open-p04-${++requests}` }) }));
  const forms = [];
  for (let n = 1; n <= 2; n++) {
    if (n === 1) await page.locator('[data-opener$="|retention.approve"]').first().click();
    else await page.locator(".p04-opener").first().click();
    const wrapper = page.locator(".p04-form-wrap").filter({ has: page.locator(`textarea[id^="p04-${n === 1 ? "approve" : "propose_policy"}-"]`) }).first();
    await wrapper.evaluate((element, n) => { element.dataset.qaForm = String(n); }, n);
    const form = page.locator(`[data-qa-form="${n}"]`);
    forms.push(form);
    if (n === 2) {
      const period = form.locator('input[id$="-period"]');
      await period.fill(String(Number(await period.inputValue()) + 1));
      const grace = form.locator('input[id$="-grace"]');
      if (!(await grace.inputValue())) await grace.fill("0");
    }
    await form.locator(".p04-form textarea").fill("synthetic retention diagnostic check");
    await form.locator(".p04-form button[type=submit]").click(); await form.locator(".p04-confirm .live-primary").click();
    await page.waitForFunction((n) => document.querySelector(".command-diagnostics")?.textContent.includes(`open-p04-${n}`), n);
  }
  check(`${lang} P04 first normal form keeps its own metadata`, (await forms[0].innerText()).includes("open-p04-1"));
  check(`${lang} P04 second normal form keeps its own metadata`, (await forms[1].innerText()).includes("open-p04-2"));
  await layout(".p04-form-wrap", "p04-open", lang);
  await page.unroute(pattern);
}
async function setupResume() {
  const statuses = await page.evaluate(async () => {
    const csrf = (await (await fetch("/api/v1/session", { headers: { "X-HB-Client": "console" } })).json()).csrf_token;
    const results = [];
    for (const command of ["integrations.subscription_pause", "integrations.subscription_request_resume"]) {
      const revision = (await (await fetch("/api/v1/integrations", { headers: { "X-CSRF-Token": csrf } })).json()).revision;
      const id = "qa-group-" + crypto.randomUUID();
      const result = await fetch("/api/v1/commands/" + command, { method: "POST", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf, "X-Request-Id": id }, body: JSON.stringify({ request_id: id, target_id: "webhook-subscription:sample-erp-invoices", idempotency_key: id, expected_revision: revision, rationale: "synthetic grouped diagnostics setup", body: {} }) });
      results.push(result.status);
    }
    return results;
  });
  check("synthetic resume-request setup is actually applied", statuses.every((status) => status === 200));
}
async function groupedRequests(lang) {
  for (const [view, action, prefix] of [["channels", "withdraw_resume", "webhook-resume-request:"], ["events", "withdraw_redeliver", "webhook-letter-request:"]]) {
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/integrations?lang=${lang}&view=${view}`);
    await settled(".p02-live");
    const command = view === "channels" ? "integrations.subscription_withdraw_resume" : "integrations.letter_withdraw_redeliver";
    const pattern = "**/api/v1/commands/" + command;
    let actual = "";
    await page.route(pattern, (route) => {
      actual = route.request().postDataJSON().target_id;
      return route.fulfill({ status: 422, contentType: "application/json", body: JSON.stringify({ code: "prerequisite_failed", request_id: "grouped-" + view }) });
    });
    await page.locator(`[data-opener$="|${action}"]`).first().click();
    await page.locator(".p02-form textarea").fill("synthetic grouped-target refusal");
    await page.locator(".p02-form button[type=submit]").click(); await page.locator(".p02-confirm .live-primary").click();
    await page.locator(".p02-form-wrap").waitFor({ state: "hidden" });
    const notes = (await page.locator(".p02-row-note").allTextContents()).join(" ");
    check(`${lang} ${view} uses canonical request target`, actual.startsWith(prefix));
    check(`${lang} ${view} grouped row retains the matching request reference`, notes.includes("grouped-" + view) && notes.includes(actual) && notes.includes(command));
    await layout(".p02-row-note:has(.command-diagnostics-row)", "grouped-" + view, lang);
    await page.unroute(pattern);
  }
}
async function layout(selector, tag, lang) {
  const fonts = await page.locator(selector).first().evaluate(async (element, locale) => {
    await document.fonts.ready;
    const family = getComputedStyle(element).fontFamily.split(",")[0];
    return { errors: [...document.fonts].filter((face) => face.status === "error").length, plex: /Plex/.test(family), loaded: document.fonts.check("16px " + family, locale === "ar" ? "العربية" : locale === "ko" ? "한글" : "Billing") };
  }, lang);
  check(`${tag} ${lang} local Plex fonts are ready`, fonts.errors === 0 && fonts.plex && fonts.loaded);
  for (const width of [1440, 1280, 1024, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    const state = await page.locator(selector).first().evaluate((e) => {
      const c = getComputedStyle(e), root = document.documentElement;
      const tokens = [...e.querySelectorAll("bdi")];
      const contained = tokens.every((n) => {
        const cell = n.closest("dd") || e, r = cell.getBoundingClientRect(), range = document.createRange(); range.selectNodeContents(n);
        return [...range.getClientRects()].every((s) => s.left >= r.left - 1 && s.right <= r.right + 1);
      });
      const neutral = ["Top", "Right", "Bottom", "Left"].every((side) => {
        if (parseFloat(c[`border${side}Width`]) === 0) return true;
        const nums = c[`border${side}Color`].match(/\d+/g)?.slice(0, 3).map(Number) || [];
        return nums.length === 3 && Math.max(...nums) - Math.min(...nums) < 24;
      });
      return { overflow: root.scrollWidth > root.clientWidth, direction: c.direction, contained, neutral, ltr: tokens.every((n) => getComputedStyle(n).direction === "ltr") };
    });
    check(`${tag} ${lang} ${width} page and token containment`, !state.overflow && state.contained);
    check(`${tag} ${lang} ${width} bidi and neutral rules`, state.direction === (lang === "ar" ? "rtl" : "ltr") && state.ltr && state.neutral);
    if (args.shots) await page.screenshot({ path: `${args.shots}/${tag}-${lang}-${width}.png`, fullPage: true });
  }
}
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
  if (args["only-grouped"]) { await setupResume(); await groupedRequests("en"); }
  for (const lang of args["only-grouped"] ? [] : args["only-refusal"] || args["only-open"] || args["only-retention"] ? ["en"] : ["en", "ko", "ar"]) {
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/integrations?lang=${lang}&view=modules`);
    await settled(".p02-live");
    if (args["only-retention"]) { await retentionFailures(lang); break; }
    if (args["only-open"]) { await openFailures(lang); break; }
    await page.route(issue, (route) => route.fulfill({ status: 422, contentType: "application/json", body: JSON.stringify({ code: "prerequisite_failed", request_id: "server-330-" + "x".repeat(110) }) }));
    await stop();
    await page.locator(".p02-form-wrap").waitFor({ state: "hidden" });
    const shown = await diagnostic().count();
    check(`${lang} closed refusal retains actual response code and ID`, shown > 0 && (await diagnostic().innerText()).includes("prerequisite_failed") && (await diagnostic().innerText()).includes("server-330-"));
    if (args["only-refusal"]) break;
    check(`${lang} kept refusal has its own row diagnostic`, await page.locator(".p02-row-note .command-diagnostics-details").count() > 0);
    check(`${lang} client request identity separately present`, await diagnostic().locator("bdi").filter({ hasText: /^m01-/ }).count() === 1);
    await page.locator(".p02-reload").click(); await settled(".p02-live");
    check(`${lang} diagnostics survive data reload`, await diagnostic().count() === 1);
    await layout(".command-diagnostics", "refused", lang);
    await page.unroute(issue);
    const disable = "**/api/v1/commands/integrations.plugin_disable";
    await page.route(disable, (route) => route.fulfill({ status: 422, contentType: "application/json", body: JSON.stringify({ code: "invalid_stage", request_id: "disable-330" }) }));
    await page.locator('[data-opener$="|integrations.plugin_disable"]').first().click();
    await page.locator(".p02-form textarea").fill("second same-row action");
    await page.locator(".p02-form button[type=submit]").click(); await page.locator(".p02-confirm .live-primary").click();
    await page.locator(".p02-form-wrap").waitFor({ state: "hidden" });
    const stopNote = page.locator(".p02-row-note").filter({ has: page.locator(".command-diagnostics-row") }).first();
    const sameRow = await stopNote.innerText();
    check(`${lang} same row binds each action to its own metadata`, sameRow.includes("integrations.plugin_stop") && sameRow.includes("server-330-") && sameRow.includes("integrations.plugin_disable") && sameRow.includes("disable-330"));
    await page.unroute(disable);
    const approve = "**/api/v1/commands/integrations.plugin_approve";
    await page.route(approve, (route) => route.fulfill({ status: 422, contentType: "application/json", body: JSON.stringify({ code: "prerequisite_failed", request_id: "approve-330" }) }));
    await page.locator('[data-opener$="|integrations.plugin_approve"]').first().click();
    await page.locator(".p02-form textarea").fill("second row action");
    await page.locator(".p02-form button[type=submit]").click(); await page.locator(".p02-confirm .live-primary").click();
    await page.locator(".p02-form-wrap").waitFor({ state: "hidden" });
    check(`${lang} other row refusal preserves earlier row references`, (await stopNote.innerText()).includes("server-330-") && (await page.locator(".p02-row-note").allTextContents()).some((text) => text.includes("approve-330") && text.includes("integrations.plugin_approve")));
    await layout(".p02-row-note:has(.command-diagnostics-row)", "row-refused", lang);
    await page.unroute(approve);
    await hash(`/integrations?lang=${lang}&view=clients`); await page.locator('.p02-live[data-live="clients"]').waitFor();
    await hash(`/integrations?lang=${lang}&view=modules`); await settled(".p02-live");
    check(`${lang} remounted kept notes retain their references`, (await page.locator(".p02-row-note").allTextContents()).some((text) => text.includes("server-330-") && text.includes("disable-330")));
    await openFailures(lang);
    await page.route(issue, (route) => route.fulfill({ status: 403, contentType: "application/json", body: JSON.stringify({ code: "reauthentication_required", request_id: "challenge-330" }) }));
    await stop(); await page.locator("dialog.service-reauth[open]").waitFor();
    await page.keyboard.press("Escape"); await page.locator("dialog.service-reauth[open]").waitFor({ state: "hidden" });
    await page.waitForFunction(() => document.querySelector(".command-diagnostics")?.textContent.includes("challenge-330"));
    const cancelled = await diagnostic().innerText();
    check(`${lang} cancellation preserves preceding challenge and explains local cancellation`, cancelled.includes("challenge-330") && cancelled.includes({ en: "cancelled here", ko: "취소했습니다", ar: "أُلغي" }[lang]));
    await layout(".command-diagnostics", "cancelled", lang);
    await page.locator(".p02-form .service-button-link").click(); await page.unroute(issue);
    await page.route(issue, (route) => route.abort("failed"));
    await stop(); await page.waitForFunction(() => document.querySelector(".command-diagnostics")?.textContent.includes("Client request ID") || document.querySelector(".command-diagnostics")?.textContent.includes("클라이언트 요청 ID") || document.querySelector(".command-diagnostics")?.textContent.includes("معرّف طلب العميل"));
    const unknown = await diagnostic().innerText();
    check(`${lang} transport loss has client reference and unavailable server metadata`, unknown.includes(missing[lang]) && !unknown.includes("challenge-330") && !unknown.includes("dependency_unavailable"));
    await page.locator(".p02-form .service-button-link").click(); await page.unroute(issue);
    await page.route(issue, (route) => route.fulfill({ status: 429, contentType: "application/json", headers: { "Retry-After": "3" }, body: JSON.stringify({ code: "rate_limited", request_id: "limited-330" }) }));
    await stop(); await page.waitForFunction(() => document.querySelector(".command-diagnostics")?.textContent.includes("limited-330"));
    check(`${lang} rate limit keeps exact code and response reference`, (await diagnostic().innerText()).includes("rate_limited"));
    await page.locator(".p02-form .service-button-link").click(); await page.unroute(issue);
    let release; const gate = new Promise((resolve) => { release = resolve; });
    let started; const sent = new Promise((resolve) => { started = resolve; });
    await page.route(issue, async (route) => { started(); await gate; await route.fulfill({ status: 422, contentType: "application/json", body: JSON.stringify({ code: "prerequisite_failed", request_id: "late-330" }) }); });
    await stop(); await sent;
    check(`${lang} new command clears earlier diagnostics`, await diagnostic().count() === 0);
    await hash(`/integrations?lang=${lang}&view=channels`); await page.locator('.p02-live[data-live="channels"]').waitFor();
    release(); await settled(".p02-live");
    check(`${lang} old-view completion cannot disclose diagnostics`, await diagnostic().count() === 0);
    await page.unroute(issue);
  }
  if (!args["only-refusal"] && !args["only-open"] && !args["only-retention"] && !args["only-grouped"]) {
    await setupResume();
    for (const lang of ["en", "ko", "ar"]) await groupedRequests(lang);
    for (const lang of ["en", "ko", "ar"]) await retentionFailures(lang);
    const screens = [["installation", "p01"], ["integrations", "p02"], ["recovery", "p03"], ["retention", "p04"], ["release-evidence", "p05"]];
    for (const [path, id] of screens) {
      const read = `**/api/v1/${path}*`;
      for (const lang of ["en", "ko", "ar"]) {
        await page.route(read, (route) => route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ code: "dependency_unavailable", request_id: "read-330-" + "y".repeat(110) }) }));
        await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/${path}?lang=${lang}`);
        await page.locator(`.${id}-live .command-diagnostics-details`).waitFor();
        const text = await page.locator(`.${id}-live`).innerText();
        check(`${id} ${lang} load failure does not describe sign-in failure`, !/sign you in|sign-in service|로그인할 수 없습니다|로그인 서비스|خدمة تسجيل الدخول/.test(text));
        check(`${id} ${lang} read error actual metadata`, text.includes("dependency_unavailable") && text.includes("read-330-"));
        await layout(`.${id}-live .command-diagnostics-details`, id + "-read", lang);
        await page.unroute(read);
      }
      for (const [status, body, tag] of [[403, { code: "permission_denied", request_id: "denied-330" }, "denied"], [401, { code: "unauthenticated", request_id: "session-330" }, "session"], [503, null, "missing"]]) {
        await page.route(read, (route) => route.fulfill({ status, contentType: "application/json", body: JSON.stringify(body) }));
        await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#/${path}?lang=en`);
        await page.locator(`.${id}-live .command-diagnostics-details`).waitFor();
        const text = await page.locator(`.${id}-live`).innerText();
        check(`${id} ${tag} metadata truthful`, body ? text.includes(body.code) && text.includes(body.request_id) : text.includes(missing.en) && !text.includes("Server error code unavailable"));
        await page.unroute(read);
      }
    }
    check("no unexpected JavaScript errors", errors.length === 0);
  }
} catch (e) {
  check("runner completed without unexpected exception", false);
  console.log(String(e).replace(/\s+/g, " ").slice(0, 600));
  if (args.shots) await page.screenshot({ path: `${args.shots}/runner-failure.png`, fullPage: true });
} finally {
  results.forEach((r) => console.log(r));
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
  await browser.close();
}
