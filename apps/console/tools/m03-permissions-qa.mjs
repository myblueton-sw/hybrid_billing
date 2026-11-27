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
  en: { title: "Grants and delegations", count: "Matching count", lower: "Matching count, at least", holder: "Holder", source: "Source", grant: "Grant", delegation: "Delegation", console: "Console", none: "None", apply: "Apply filters", reset: "Reset filters", next: "Next", first: "First", empty: "No matches on this page.", more: "No matches on this page. More source records remain to scan.", requested: "Revocation requested", yours: "Your grant", set: "Set by the installation" },
  ko: { title: "권한 부여·위임", count: "일치 항목 수", lower: "일치 항목 수, 최소", holder: "보유자", source: "권한 출처", grant: "권한 부여", delegation: "위임", console: "콘솔", none: "없음", apply: "필터 적용", reset: "필터 초기화", next: "다음", first: "처음", empty: "이 페이지에 일치하는 항목이 없습니다.", more: "이 페이지에는 일치 항목이 없습니다. 조회할 원본 항목이 더 있습니다.", requested: "", yours: "내 권한", set: "설치 환경에서 설정" },
  ar: { title: "المنح والتفويضات", count: "عدد النتائج المطابقة", lower: "عدد النتائج المطابقة، على الأقل", holder: "صاحب الصلاحية", source: "المصدر", grant: "منح", delegation: "التفويض", console: "وحدة التحكم", none: "لا يوجد", apply: "تطبيق عوامل التصفية", reset: "إعادة تعيين عوامل التصفية", next: "التالي", first: "الأول", empty: "لا توجد نتائج مطابقة في هذه الصفحة.", more: "لا توجد نتائج مطابقة في هذه الصفحة. ما زالت هناك سجلات مصدر لفحصها.", requested: "", yours: "صلاحيتك", set: "يحدده التثبيت" },
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
  const open = async (lang, extra = {}) => {
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/access?${new URLSearchParams({ lang, view: "grants", ...extra })}`);
    await page.locator("#access-list").waitFor();
    await page.locator(".access-list-body .list-summary, .access-list-failure, .access-list-body [role=alert]").first().waitFor();
  };
  const asked = [];
  page.on("request", (r) => { if (isList(r.url())) asked.push(new URL(r.url()).search); });

  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("en");
  const real = await page.locator(".access-permissions tbody tr").evaluateAll((rows) => rows.map((r) => ({ id: r.dataset.permission, cells: [...r.children].map((c) => c.innerText.trim().replace(/\s+/g, " ").slice(0, 60)) })));
  const sorted = real.map((r) => r.id);
  check("the real service's list shows its grants in the server's order under the one heading", real.length >= 3 && sorted.every((id) => id.startsWith("grant/")) && [...sorted].sort().join() === sorted.join() && (await page.locator("#access-list").innerText()).trim() === copy.en.title && (await page.locator("#access-people, #access-delegations").count()) === 0, JSON.stringify(sorted));
  check("the count line gives an exact count equal to the rows and the list's reference time", (await page.locator(".access-list-body .list-summary").innerText()).includes(`${copy.en.count}: ${real.length}`) && (await page.locator(".access-list-body .list-summary time").count()) === 1);
  const consoleRow = real.find((r) => r.cells[0].startsWith(copy.en.console)), own = real.find((r) => r.cells[7] === copy.en.yours);
  check("a console-client grant reads Console and Set by the installation; one's own grant reads Your grant", !!consoleRow && consoleRow.cells[7] === copy.en.set && !!own, JSON.stringify([consoleRow?.cells, own?.cells]));
  check("asked once for the first page with the fixed page size and no filter", asked.filter((s) => s === "?limit=50").length >= 1 && asked.every((s) => s === "?limit=50"), asked.join(" "));
  const before = asked.length;
  await page.locator("#acc-q").fill("reader");
  await page.locator("#acc-source").selectOption("grant");
  check("typing in the filters sends nothing", asked.length === before);
  await page.locator("#acc-q").press("Enter");
  await page.waitForFunction(() => new URLSearchParams(location.hash.split("?")[1]).get("acc_q") === "reader");
  await page.locator(".access-list-body .list-summary").waitFor();
  await page.waitForTimeout(300);
  const filtered = await page.locator(".access-permissions tbody tr").evaluateAll((rows) => rows.map((r) => r.dataset.permission));
  check("Enter applies the filters: the address carries them, the service is asked with them and answers only matching rows", asked[asked.length - 1] === "?q=reader&source=grant&limit=50" && filtered.length > 0 && filtered.every((id) => id.includes("reader")) && filtered.length < real.length, asked[asked.length - 1] + " " + filtered.join());
  check("after applying, focus is on the count line", await page.evaluate(() => document.activeElement?.classList.contains("list-summary")));
  await page.locator(".access-reset").click();
  await page.waitForFunction(() => !new URLSearchParams(location.hash.split("?")[1]).has("acc_q"));
  await page.locator(".access-list-body .list-summary").waitFor();
  await page.waitForTimeout(300);
  check("Reset clears the filters and reads the first page again", (await page.locator(".access-permissions tbody tr").count()) === real.length && (await page.locator("#acc-q").inputValue()) === "");
  const target = real.find((r) => r.cells[7].startsWith("Request revocation"));
  if (target) {
    await page.locator(`[data-permission="${target.id}"] .live-revoke`).click();
    await page.locator(".live-panel form.live-step").waitFor();
    const panel = await page.locator(".live-panel form.live-step").innerText();
    check("Request revocation opens the panel with the row's holder, access and status", panel.includes(target.cells[0].split(" ")[0]) && panel.includes("Request revocation"), panel.slice(0, 200));
    const folded = await page.evaluate(() => ({ parent: getComputedStyle(document.querySelector(".access-permissions th.acc-parent")).display, list: Math.round(document.querySelector(".access-list-body").getBoundingClientRect().width) }));
    notes.push(`beside the open panel the list is ${folded.list}px wide and the original-delegation column is ${folded.parent}`);
    if (args.shots) await page.screenshot({ path: `${args.shots}/m03-list-panel-en-1440.png` });
  } else notes.push("the sample has no grant that can be asked to be revoked");
  await open("en", { acc_state: "nonsense" });
  check("filters that cannot be read are refused on the screen without asking the service", (await page.locator(".access-list-body [role=alert]").count()) === 1 && !asked.some((s) => s.includes("nonsense")));

  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 390], ["ko", 390], ["ar", 390]]) {
    const at = `${lang} ${width}`;
    await page.setViewportSize({ width, height: 1000 });
    let mode = "rows";
    await page.route((url) => isList(url.toString()), async (route) => {
      const qs = new URL(route.request().url()).searchParams;
      if (mode === "fail") return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
      if (qs.get("after") === "c.d") return route.fulfill({ status: 200, json: pageBody(qs, []) });
      if (qs.get("after") === "a.b") return route.fulfill({ status: 200, json: pageBody(qs, [], { next: "c.d", count: { value: "10", certainty: "lower_bound" }, scan_stopped: true }) });
      const term = (qs.get("q") ?? "").toLowerCase();
      const rows = synthetic.filter((r) => (!qs.get("source") || r.source === qs.get("source")) && (!qs.get("state") || r.state === qs.get("state")) && (!term || r.id.includes(term) || r.subject.includes(term)));
      await route.fulfill({ status: 200, json: pageBody(qs, rows, qs.get("source") ? {} : { next: "a.b", count: { value: String(rows.length), certainty: "lower_bound" } }) });
    });
    await open(lang);
    const cells = await page.locator(".access-permissions tbody tr").evaluateAll((rows) => Object.fromEntries(rows.map((r) => [r.dataset.permission, [...r.children].map((c) => c.innerText.trim().replace(/\s+/g, " "))])));
    const row = (id) => cells[id] ?? [];
    check(`${at}: ten rows of both sources, a lower-bound count and Next enabled`, Object.keys(cells).length === 10 && (await page.locator(".access-list-body .list-summary").innerText()).includes(`${copy[lang].lower}: 10`) && await page.locator(".access-list-body .list-pager-nav button").nth(2).isEnabled() && await page.locator(".access-list-body .list-pager-nav button").nth(0).isDisabled() && await page.locator(".access-list-body .list-pager-nav button").nth(1).isDisabled());
    check(`${at}: a grant and a delegation name their source; a delegation names who delegated and its parent`, row("grant/b-person")[3].startsWith(copy[lang].grant) && row("delegation/h-first")[3].startsWith(copy[lang].delegation) && row("delegation/i-second")[0].includes("member@example.invalid") && row("delegation/i-second").join(" ").includes("h-first"), JSON.stringify(row("delegation/i-second")).slice(0, 300));
    check(`${at}: the console grant, a restricted scope and a holder the directory does not know`, row("grant/a-console")[0].startsWith(copy[lang].console) && row("grant/a-console")[7] === copy[lang].set && row("grant/b-person")[2].includes("customer-with-a-long-identifier-0001") && row("grant/b-person")[2].includes("contract-7") && row("grant/b-person")[2].includes("24322-24324") && row("grant/g-inapplicable")[0].startsWith("left-user"), JSON.stringify([row("grant/a-console"), row("grant/b-person")[2]]).slice(0, 300));
    const acts = await page.locator(".access-permissions tbody tr").evaluateAll((rows) => Object.fromEntries(rows.map((r) => [r.dataset.permission, [!!r.querySelector("[data-grant]"), !!r.querySelector("[data-delegation]"), !!r.querySelector("[data-requested]")]])));
    check(`${at}: actions follow the rules: request on another person's live grant, none on one's own, a pending request, an ended row; revoke on a live delegation only`, acts["grant/b-person"][0] && !acts["grant/c-own"][0] && !acts["grant/d-asked"][0] && acts["grant/d-asked"][2] && !acts["grant/e-expired"][0] && !acts["grant/f-revoked"][0] && acts["grant/g-inapplicable"][0] && acts["delegation/h-first"][1] && !acts["delegation/j-ended"][1] && row("grant/c-own").join(" ").includes(copy[lang].yours), JSON.stringify(acts));
    const strong = await page.locator(".access-permissions tbody tr").evaluateAll((rows) => Object.fromEntries(rows.map((r) => [r.dataset.permission, r.querySelector("td[data-state]").classList.contains("live-strong")])));
    check(`${at}: every state other than active and effective is bold`, !strong["grant/b-person"] && strong["grant/e-expired"] && strong["grant/f-revoked"] && strong["grant/g-inapplicable"] && strong["delegation/j-ended"]);
    const shown = await inspectView(page);
    check(`${at}: the list does not overflow, cut off or lose text contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), low: shown.low.slice(0, 2) }));
    const layout = await page.evaluate(() => ({ head: getComputedStyle(document.querySelector(".access-permissions thead")).display, form: getComputedStyle(document.querySelector(".access-filter-form")).display, toggle: getComputedStyle(document.querySelector(".access-filters-toggle")).display, ids: document.querySelectorAll("#acc-q").length }));
    check(`${at}: ${width === 390 ? "rows are cards and the filters are collapsed under a summary" : "the table has its header and the filters are shown"}`, layout.ids === 1 && (width === 390 ? layout.head === "none" && layout.form === "none" && layout.toggle !== "none" : layout.head !== "none" && layout.form !== "none" && layout.toggle === "none"), JSON.stringify(layout));
    const fit = await page.evaluate(() => { const box = document.querySelector(".access-permissions").getBoundingClientRect(), table = document.querySelector(".access-permissions table").getBoundingClientRect(); const over = [...document.querySelectorAll(".access-permissions td, .access-permissions th")].filter((c) => c.getClientRects().length && c.scrollWidth > c.clientWidth + 1).map((c) => c.textContent.trim().slice(0, 24)); const out = [...document.querySelectorAll(".access-permissions td *, .access-permissions .live-revoke")].filter((e) => { if (!e.getClientRects().length || e.closest(".visually-hidden")) return false; const r = e.getBoundingClientRect(), cell = e.closest("td").getBoundingClientRect(); return r.right > cell.right + 1 || r.left < cell.left - 1; }).map((e) => e.textContent.trim().slice(0, 24)); return { table: Math.round(table.width), box: Math.round(box.width), over, out, folded: getComputedStyle(document.querySelector(".access-permissions th.acc-parent")).display === "none" }; });
    const words = await page.evaluate(() => { const spill = [...document.querySelectorAll(".access-permissions .live-revoke")].filter((b) => b.scrollWidth > b.clientWidth + 1).map((b) => b.textContent.trim().slice(0, 24)); const broken = []; for (const el of document.querySelectorAll(".access-permissions td span, .access-permissions td button, .access-permissions th")) { if (!el.getClientRects().length || el.closest(".visually-hidden") || el.querySelector("span, bdi, button, time")) continue; const node = [...el.childNodes].find((n) => n.nodeType === 3 && /[A-Za-z]{6,}/.test(n.textContent)); if (!node || node.textContent.includes("@")) continue; for (const m of node.textContent.matchAll(/[A-Za-z]{6,}/g)) { const r = document.createRange(); r.setStart(node, m.index); r.setEnd(node, m.index + m[0].length); if (r.getClientRects().length > 1) broken.push(m[0] + " in " + (el.closest("td")?.dataset.label ?? el.tagName)); } } return { spill, broken: [...new Set(broken)] }; });
    check(`${at}: no button's text spills out of it and no ordinary word is broken across lines`, words.spill.length === 0 && words.broken.length === 0, JSON.stringify(words));
    const paging = await page.locator(".access-list-body .list-pager-nav button").evaluateAll((bs) => bs.map((b) => ({ disabled: b.disabled, bg: getComputedStyle(b).backgroundColor, color: getComputedStyle(b).color })));
    check(`${at}: on the first page First and Previous are disabled and look different from Next`, paging[0].disabled && paging[1].disabled && !paging[2].disabled && paging[0].bg === paging[1].bg && (paging[0].bg !== paging[2].bg || paging[0].color !== paging[2].color), JSON.stringify(paging));
    check(`${at}: the table fits its container and nothing in a cell spills over its neighbours`, fit.table <= fit.box + 1 && fit.over.length === 0 && fit.out.length === 0, JSON.stringify(fit));
    const lines = await page.locator('[data-permission="grant/b-person"] td[data-state]').evaluate((td) => [...td.children].filter((e) => e.getClientRects().length).map((e) => e.textContent.trim()));
    if (fit.folded && width !== 390) check(`${at}: the folded effective cell gives the reason, then From, and does not repeat the end date`, lines.length === 3 && lines.filter((t) => /Dec|12|ديسمبر/.test(t)).length === 1, JSON.stringify(lines));
    const countLines = await page.locator(".access-list-body .list-summary p:not(.visually-hidden)").allInnerTexts();
    check(`${at}: the count and the reference time are two lines`, countLines.length === 2 && countLines[0].trim().endsWith("10"), JSON.stringify(countLines));
    if (args.shots) { await page.locator(".access-list-body").screenshot({ path: `${args.shots}/m03-list-${lang}-${width}.png` }); }
    await page.locator(".access-list-body .list-pager-nav button").nth(2).focus();
    await page.keyboard.press("Enter");
    await page.locator(".access-list-body .live-empty").waitFor();
    await page.waitForTimeout(200);
    if (args.shots && lang === "en") await page.locator(".access-list-body").screenshot({ path: `${args.shots}/m03-list-more-${lang}-${width}.png` });
    check(`${at}: a page that found nothing yet says more remains, keeps Next, and focus is on the count line`, (await page.locator(".access-list-body .live-empty").innerText()).trim() === copy[lang].more && await page.locator(".access-list-body .list-pager-nav button").nth(2).isEnabled() && await page.evaluate(() => document.activeElement?.classList.contains("list-summary")) && asked[asked.length - 1].includes("after=a.b"));
    await page.locator(".access-list-body .list-pager-nav button").nth(2).click();
    await page.waitForFunction((t) => document.querySelector(".access-list-body .live-empty")?.textContent.trim() === t, copy[lang].empty);
    if (args.shots && lang === "en") await page.locator(".access-list-body").screenshot({ path: `${args.shots}/m03-list-last-${lang}-${width}.png` });
    check(`${at}: the last page says there are no matches, with Next disabled and Previous enabled`, await page.locator(".access-list-body .list-pager-nav button").nth(2).isDisabled() && await page.locator(".access-list-body .list-pager-nav button").nth(1).isEnabled());
    await page.locator(".access-list-body .list-pager-nav button").nth(1).click();
    await page.waitForFunction((t) => document.querySelector(".access-list-body .live-empty")?.textContent.trim() === t, copy[lang].more);
    check(`${at}: Previous returns to the page before`, asked[asked.length - 1].includes("after=a.b"));
    await page.locator(".access-list-body .list-pager-nav button").nth(0).click();
    await page.locator(".access-permissions tbody tr").first().waitFor();
    check(`${at}: First returns to the first page`, !asked[asked.length - 1].includes("after=") && (await page.locator(".access-permissions tbody tr").count()) === 10);
    if (width === 390) { await page.locator(".access-filters-toggle").focus(); await page.keyboard.press("Enter"); }
    await page.locator("#acc-source").selectOption("delegation");
    await page.locator("#acc-q").fill("h");
    await page.locator("#acc-state").selectOption("active");
    await page.locator(".access-apply").click();
    await page.waitForFunction(() => document.querySelectorAll(".access-permissions tbody tr").length === 1);
    check(`${at}: the source filter lists delegations only with an exact count`, (await page.locator(".access-list-body .list-summary").innerText()).includes(`${copy[lang].count}: 1`) && (await page.locator(".access-permissions tbody tr").getAttribute("data-permission")) === "delegation/h-first" && asked[asked.length - 1] === "?q=h&source=delegation&state=active&limit=50" && await page.evaluate(() => document.activeElement?.classList.contains("list-summary")));
    if (args.shots) await page.locator(".access-list-body").screenshot({ path: `${args.shots}/m03-list-filtered-${lang}-${width}.png` });
    if (width === 390) check(`${at}: the collapsed filters say three are applied and stay open`, (await page.locator(".access-filters-toggle").innerText()).includes("3") && (await page.locator(".access-filters-toggle").getAttribute("aria-expanded")) === "true" && await page.locator("#acc-source").isVisible());
    mode = "fail";
    await page.locator(".live-toolbar .live-secondary").last().click();
    await page.locator(".access-list-failure").waitFor();
    const failure = await page.locator(".access-list-failure").innerText();
    check(`${at}: a failed read says the data could not be verified, with Reload and First, and none of the users screen's wording`, (await page.locator(".access-list-failure button").count()) === 2 && !/save|저장|حفظ/i.test(failure) && failure.includes(copy[lang].first), failure.slice(0, 160));
    if (args.shots && width === 1440) { await page.locator(".access-list-failure").screenshot({ path: `${args.shots}/m03-list-failure-${lang}-${width}.png` }); }
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
