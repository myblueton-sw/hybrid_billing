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
  en: "This page size is above this installation's current limit for lists. Reading again uses the current limit.",
  ko: "이 페이지 크기가 설치 환경의 현재 목록 한도를 넘습니다. 다시 불러오면 현재 한도를 적용합니다.",
  ar: "حجم الصفحة هذا أكبر من الحدّ الحالي للقوائم في هذا التثبيت. تستخدم إعادة التحميل الحدّ الحالي.",
};
const limit = Number(args.limit || 10);
const listPath = (url) => { const p = new URL(url).pathname; const m = p.match(/\/api\/v1\/identity\/(access\/(permissions|requests|revocations|subjects)|policy\/changes)$/); return m ? m[1] : ""; };
const entry = (url) => /\/api\/v1\/identity\/(access|policy)$/.test(new URL(url).pathname);

try {
  const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
  const activation = new URL(lines.find((l) => l.startsWith("admin@example.invalid ")).split(" ")[1]);
  await page.goto(`${base}/activate?lang=en${activation.hash}`);
  await page.locator("#link-new").fill(password); await page.locator("#link-confirm").fill(password);
  await page.locator("form button[type=submit]").click(); await page.locator("a.service-button-link").waitFor();
  await page.goto(`${base}/?service=api&lang=en`);
  await page.locator("#signin-workspace").fill("sample-workspace"); await page.locator("#signin-email").fill("admin@example.invalid");
  await page.locator("#signin-password").fill(password); await page.locator("form button[type=submit]").click();
  await page.locator(".service-session").waitFor();
  let asked = [], refused = [], stated = null;
  page.on("request", (r) => { const p = listPath(r.url()); if (p) asked.push(p + new URL(r.url()).search); });
  page.on("response", async (r) => {
    if (listPath(r.url()) && r.status() === 422) refused.push(listPath(r.url()));
    if (entry(r.url()) && r.status() === 200) { const body = await r.json().catch(() => null); const row = body?.query_settings?.find((x) => x.name === "service_query_page_size"); if (row) stated = row; }
  });
  const sizes = () => [...new Set(asked)].sort();
  const openAccess = async (lang) => {
    asked = []; refused = [];
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/access?${new URLSearchParams({ lang, view: "grants" })}`);
    await page.locator("#access-revocations").waitFor();
    await page.locator(".access-revocation-section [data-revocations], .access-revocation-section .access-list-failure").first().waitFor();
    await page.locator(".access-list-body .access-permissions, .access-list-body .access-list-failure, .access-list-body .live-empty, .access-list-body [data-count]").first().waitFor();
    await page.locator(".access-request-section [data-waiting], .access-request-section .access-list-failure").first().waitFor();
  };
  const openPolicy = async (lang) => {
    asked = []; refused = [];
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/authentication?lang=${lang}&view=policies`);
    await page.locator("#m02-changes").waitFor();
    for (const g of ["open", "closed"]) await page.locator(`.m02-list-section[data-group="${g}"] h3[data-count], .m02-list-section[data-group="${g}"] .access-list-failure`).first().waitFor();
  };

  await openAccess("en");
  check(`the access read states the page limit ${limit} and its source`, stated?.value === limit && stated?.source === "start_flag" && stated?.default === 50, JSON.stringify(stated));
  check("M03: every list asks for at most the limit and none is refused", refused.length === 0 && [`access/permissions?limit=${limit}`, `access/requests?state=ended&limit=${limit}`, `access/requests?state=pending&limit=${limit}`, `access/revocations?limit=${limit}`].every((x) => asked.includes(x)) && asked.every((x) => !/limit=(\d+)/.test(x) || Number(x.match(/limit=(\d+)/)[1]) <= limit), `${sizes().join(" ")} refused ${refused.join(" ")}`);
  check("M03: no list shows a failure", (await page.locator(".access-list-failure").count()) === 0);
  stated = null;
  await openPolicy("en");
  check(`the policy read states the page limit ${limit}`, stated?.value === limit && stated?.source === "start_flag", JSON.stringify(stated));
  check("M02: both lists ask for at most the limit and none is refused", refused.length === 0 && sizes().join(" ") === `policy/changes?state=closed&limit=${limit} policy/changes?state=open&limit=${limit}`, `${sizes().join(" ")} refused ${refused.join(" ")}`);
  check("M02: no list shows a failure", (await page.locator(".access-list-failure").count()) === 0);

  const overstate = async (route) => {
    const response = await route.fetch();
    const body = await response.json();
    if (Array.isArray(body.query_settings)) body.query_settings = body.query_settings.map((x) => (x.name === "service_query_page_size" ? { ...x, value: 50 } : x));
    return route.fulfill({ response, json: body });
  };
  for (const lang of ["en", "ko", "ar"]) {
    await page.route((url) => entry(url.toString()), overstate);
    await openPolicy(lang);
    const boxes = await page.locator(".m02-lists .access-list-failure").evaluateAll((all) => all.map((f) => ({ body: f.querySelector("p").innerText.trim(), buttons: f.querySelectorAll("button").length })));
    check(`${lang} M02: with a stale larger limit the real service refuses the size and both lists say why, with one button`, refused.length >= 2 && boxes.length === 2 && boxes.every((b) => b.body === copy[lang] && b.buttons === 1), JSON.stringify({ refused, boxes }));
    if (args.shots) { await mkdir(args.shots, { recursive: true }); await page.locator(".m02-lists").screenshot({ path: `${args.shots}/page-limit-m02-${lang}.png` }); }
    await page.unrouteAll({ behavior: "wait" });
    asked = []; refused = [];
    await page.locator(".m02-lists .access-list-failure button").first().click();
    for (const g of ["open", "closed"]) await page.locator(`.m02-list-section[data-group="${g}"] h3[data-count]`).waitFor();
    check(`${lang} M02: Reload reads the policy again and both lists load at the real limit`, refused.length === 0 && (await page.locator(".m02-lists .access-list-failure").count()) === 0 && sizes().join(" ") === `policy/changes?state=closed&limit=${limit} policy/changes?state=open&limit=${limit}`, sizes().join(" "));

    await page.route((url) => entry(url.toString()), overstate);
    await openAccess(lang);
    const failed = await page.locator(".access-list-failure").evaluateAll((all) => all.map((f) => ({ body: f.querySelector("p").textContent.trim(), buttons: f.querySelectorAll("button").length })));
    check(`${lang} M03: with a stale larger limit the permissions and waiting-requests lists say the size was refused, with one button each`, failed.length >= 2 && failed.every((b) => b.body === copy[lang] && b.buttons === 1) && refused.includes("access/permissions") && refused.includes("access/requests"), JSON.stringify({ refused, failed }));
    await page.locator(".access-grant-open").click();
    await page.locator(".person-failed").first().waitFor({ timeout: 15000 });
    const picker = await page.locator(".person-failed").evaluateAll((all) => all.map((f) => ({ body: f.querySelector("p").textContent.trim(), buttons: [...f.querySelectorAll("button")].map((b) => b.textContent.trim()) })));
    check(`${lang} M03: a person picker refused for its size says why and offers Reload alone`, asked.includes("access/subjects?limit=25") && refused.includes("access/subjects") && picker.length > 0 && picker.every((x) => x.body.startsWith(copy[lang]) && x.buttons.length === 1), JSON.stringify({ refused, picker }));
    check(`${lang} M03: lists whose size fits the real limit still load`, (await page.locator(".access-revocation-section .access-list-failure").count()) === 0);
    if (args.shots) await page.screenshot({ path: `${args.shots}/page-limit-m03-${lang}.png`, fullPage: true });
    await page.unrouteAll({ behavior: "wait" });
    asked = []; refused = [];
    await page.locator(".access-list-body .access-list-failure button").first().click();
    await page.waitForFunction(() => document.querySelectorAll(".access-list-failure").length === 0, null, { timeout: 15000 });
    await page.waitForFunction(() => document.querySelectorAll(".person-failed").length === 0, null, { timeout: 15000 });
    check(`${lang} M03: the person picker asks again at the real limit once the screen knows it`, asked.includes(`access/subjects?limit=${limit}`), sizes().join(" "));
    check(`${lang} M03: Reload reads the access read again and every list loads at the real limit, with no refused request on the way`, refused.length === 0 && asked.every((x) => !/limit=(\d+)/.test(x) || Number(x.match(/limit=(\d+)/)[1]) <= limit) && asked.some((a) => a === `access/permissions?limit=${limit}`) && asked.some((a) => a === `access/requests?state=pending&limit=${limit}`), sizes().join(" "));
  }

  await page.route((url) => /\/api\/v1\/identity\/access$/.test(new URL(url.toString()).pathname), (route) => route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } }));
  asked = []; refused = [];
  await page.goto("about:blank");
  await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#/access?${new URLSearchParams({ lang: "en", view: "grants" })}`);
  await page.locator(".access-list-body [data-count], .access-list-body .access-permissions, .access-list-body .live-empty").first().waitFor();
  check("M03: when the access read fails the permissions list still loads, asking for no size", refused.length === 0 && sizes().join(" ") === "access/permissions" && (await page.locator(".access-list-body .access-list-failure").count()) === 0, sizes().join(" "));
  await page.unrouteAll({ behavior: "wait" });
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) if (!args.quiet || r.startsWith("FAIL")) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
