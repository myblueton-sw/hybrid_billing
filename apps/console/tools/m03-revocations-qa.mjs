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
  en: { shown: (n) => (n === 1 ? "1 revocation" : `${n} revocations`), atLeast: (n) => `At least ${n} revocations`, failed: "Revocations could not be loaded", none: "No revocations yet.", reference: "List reference time", grant: "Grant: View · Rows for member@example.invalid", delegation: "Delegation to admin2@example.invalid: View · Rows", gone: "Grant grant-gone" },
  ko: { shown: (n) => `회수 ${n}건`, atLeast: (n) => `회수 ${n}건 이상`, failed: "회수 목록을 불러오지 못했습니다", none: "아직 회수가 없습니다.", reference: "목록 기준 시각", grant: "member@example.invalid", delegation: "admin2@example.invalid님에게 한 위임", gone: "권한 grant-gone" },
  ar: { shown: () => "", atLeast: (n) => `عدد عمليات الإبطال: ${n} على الأقل`, failed: "تعذّر تحميل عمليات الإبطال", none: "لا توجد عمليات إبطال بعد.", reference: "وقت مرجع القائمة", grant: "member@example.invalid", delegation: "تفويض إلى admin2@example.invalid", gone: "الصلاحية grant-gone" },
};
const base0 = { origin: "this_process", recorded_at: "2026-11-16T09:00:00Z", target_kind: "grant", target: "grant-8f02cad10c1d560b", target_version: 1, target_present: true, replay: "", state: "revocation_complete",
  history: [{ state: "received", at: "2026-11-16T09:00:00Z" }, { state: "revocation_complete", at: "2026-11-16T09:00:01Z" }], ended_delegations: [], deny_generation: 93, job_state: "succeeded", job_reason: "",
  requested_by: "sample-admin", approved_by: "sample-admin-2", request: "access-request-2", received_at: "2026-11-16T09:00:00Z", deny_at: "2026-11-16T09:00:00Z", receipt: { sequence: 2, watermark: "2:edd4ef79a84b7062d9bf2f299fac1cef452a584bd3e1313800c343b37930770b", reason: "" }, verified: true,
  cleanup: [{ item: "C1", state: "done", reason: "deny_generation", refs: [] }, { item: "C5", state: "outside_scope", reason: "cannot_recall", refs: [] }, { item: "C9", state: "not_applicable", reason: "no_dependent_delegation", refs: [] }], completed_at: "2026-11-16T09:00:01Z",
  target_subject: "sample-member", target_subject_login: "member@example.invalid", target_action: "view", target_resource_kind: "focus_data", target_fields: ["rows"] };
const record = (n, change = {}) => ({ ...base0, id: `revocation-${n}`, ...change });
const gone = { target_subject: "", target_subject_login: "", target_action: "", target_resource_kind: "", target_fields: [] };
const firstPage = [
  record(12),
  record(11, { target_kind: "delegation", target: "del-with-a-long-identifier-0001", target_subject: "sample-admin-2", target_subject_login: "admin2@example.invalid", ended_delegations: ["qd-01", "qd-02"], approved_by: "", request: "" }),
  record(10, { target: "grant-gone", target_present: false, ...gone }),
  record(9, { state: "decision_unconfirmed", job_state: "waiting", job_reason: "decision_unconfirmed", receipt: { sequence: 0, watermark: "", reason: "ledger_isolated" }, verified: false, completed_at: "", cleanup: [{ item: "C1", state: "not_started", reason: "deny_generation", refs: [] }], history: [{ state: "received", at: "2026-11-16T09:00:00Z" }] }),
  record(8, { origin: "ledger_replay", replay: "absent", target: "grant-replayed", target_present: false, ...gone }),
  ...[7, 6, 5, 4, 3].map((n) => record(n)),
];
const pageBody = (qs, rows, change = {}) => ({ rows, search_input: "", query: { q: "", state: "", limit: Number(qs.get("limit") ?? 25) }, next: "", reference_at: "2026-11-26T08:45:18Z", observed_at: "2026-11-26T08:45:18Z", revision: "r1", order: "recorded_desc",
  count: { value: String(rows.length), certainty: "exact" }, work_units: rows.length, scan_stopped: false, ...change });
const isList = (url) => new URL(url).pathname.endsWith("/identity/access/revocations");

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
  const section = ".access-revocation-section";
  let mode = "mix";
  const open = async (lang) => {
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/access?${new URLSearchParams({ lang, view: "grants" })}`);
    await page.locator("#access-revocations").waitFor();
    await page.locator(`${section} [data-revocations], ${section} .access-list-failure`).first().waitFor();
  };

  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("en");
  check("the real service's section reads revocations as one bounded read of ten", asked.length >= 1 && asked.every((s) => s === "?limit=10"), asked.join(" "));
  check("with no revocations the real section counts none, says so, keeps the ledger line and a reference time, and shows no paging", (await page.locator(`${section} [data-revocations]`).innerText()).trim() === "0 revocations" && (await page.locator(`${section} .live-empty`).innerText()).trim() === copy.en.none && (await page.locator(`${section} .access-ledger`).count()) === 1 && (await page.locator(`${section} .list-summary`).count()) === 1 && (await page.locator(`${section} .list-pager-nav`).count()) === 0);

  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 390], ["ko", 390], ["ar", 390]]) {
    const at = `${lang} ${width}`;
    await page.setViewportSize({ width, height: 1000 });
    mode = "mix";
    await page.route((url) => isList(url.toString()), async (route) => {
      const qs = new URL(route.request().url()).searchParams;
      if (mode === "fail") return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
      if (qs.get("after") === "r.2") return route.fulfill({ status: 200, json: pageBody(qs, [record(2), record(1)], { count: { value: "12", certainty: "exact" } }) });
      return route.fulfill({ status: 200, json: pageBody(qs, firstPage, { next: "r.2", count: { value: "10", certainty: "lower_bound" } }) });
    });
    await open(lang);
    await page.mouse.move(2, 2);
    const heads = await page.locator(`${section} .access-record`).evaluateAll((items) => Object.fromEntries(items.map((li) => [li.dataset.revocation, [li.querySelector("h3").innerText.trim().replace(/\s+/g, " "), li.querySelector(".access-target")?.innerText.trim() ?? ""]])));
    check(`${at}: ten records, a lower-bound count in the heading and paging`, Object.keys(heads).length === 10 && (await page.locator(`${section} [data-revocations]`).innerText()).trim() === copy[lang].atLeast(10) && (await page.locator(`${section} .list-pager-nav button`).count()) === 3);
    check(`${at}: a grant's heading names its holder and access, with the kind and identifier below`, heads["revocation-12"][0].includes(copy[lang].grant) && heads["revocation-12"][1].includes("grant-8f02cad10c1d560b"), JSON.stringify(heads["revocation-12"]));
    check(`${at}: a delegation's heading names the delegate`, heads["revocation-11"][0].includes(copy[lang].delegation) && heads["revocation-11"][1].includes("del-with-a-long-identifier-0001"), JSON.stringify(heads["revocation-11"]));
    check(`${at}: a target that is gone shows its identifier alone`, heads["revocation-10"][0] === copy[lang].gone && heads["revocation-10"][1] === "", JSON.stringify(heads["revocation-10"]));
    check(`${at}: an unconfirmed record and a replayed record keep their states`, (await page.locator(`${section} [data-revocation="revocation-9"]`).getAttribute("data-state")) === "decision_unconfirmed" && (await page.locator(`${section} [data-revocation="revocation-8"] .live-meta`).first().innerText()).length > 0);
    const shown = await inspectView(page);
    check(`${at}: nothing overflows, is cut off or loses contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), low: shown.low.slice(0, 2) }));
    if (args.shots) await page.locator(section).screenshot({ path: `${args.shots}/m03-revocations-${lang}-${width}.png` });
    await page.locator(`${section} .list-pager-nav button`).nth(2).click();
    await page.waitForFunction((s) => document.querySelectorAll(`${s} .access-record`).length === 2, section);
    check(`${at}: the second page has the last two, the exact total, Next disabled, and focus on the reference time`, (lang === "ar" || (await page.locator(`${section} [data-revocations]`).innerText()).trim() === copy[lang].shown(12)) && await page.locator(`${section} .list-pager-nav button`).nth(2).isDisabled() && await page.locator(`${section} .list-pager-nav button`).nth(1).isEnabled() && await page.evaluate(() => document.activeElement?.classList.contains("list-summary") && !!document.activeElement.closest(".access-revocation-section")) && asked[asked.length - 1] === "?limit=10&after=r.2", asked[asked.length - 1]);
    if (args.shots && lang === "en") await page.locator(section).screenshot({ path: `${args.shots}/m03-revocations-page2-${lang}-${width}.png` });
    await page.locator(`${section} .list-pager-nav button`).nth(0).click();
    await page.waitForFunction((s) => document.querySelectorAll(`${s} .access-record`).length === 10, section);
    check(`${at}: First returns to the first page`, asked[asked.length - 1] === "?limit=10");
    mode = "fail";
    await open(lang);
    check(`${at}: a failed list says so with Reload, shows no count, and keeps the ledger line and the other sections`, (await page.locator(`${section} .access-list-failure h3`).innerText()).trim() === copy[lang].failed && (await page.locator(`${section} [data-revocations]`).count()) === 0 && (await page.locator(`${section} .access-ledger`).count()) === 1 && (await page.locator(`${section} .access-list-failure button`).count()) === 1 && (await page.locator("#access-requests").count()) === 1 && (await page.locator(".access-permissions tbody tr").count()) > 0);
    if (args.shots && width === 1440) await page.locator(section).screenshot({ path: `${args.shots}/m03-revocations-failure-${lang}-${width}.png` });
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
