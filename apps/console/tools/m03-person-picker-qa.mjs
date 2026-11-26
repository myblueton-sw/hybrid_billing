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
  en: { hint: "Search by login or user ID.", first: "Showing the first 25 by user ID. Search to find others.", refine: "More people match than are shown. Type more of the login or user ID.", none: (t) => `No one matches “${t}”.`, onlyYou: (t) => `Only you match “${t}”. You can't request access for yourself.`, failed: "People could not be searched.", kept: "The person you chose is kept.", change: "Change", keep: "Keep this person", flag: "Revocation not confirmed", request: "A revocation for this person is not confirmed", count: (n) => `${n} people` },
  ko: { hint: "로그인 ID 또는 사용자 ID로 검색합니다.", first: "사용자 ID순으로 처음 25명을 표시합니다. 다른 사람은 검색하세요.", refine: "표시된 것보다 일치하는 사람이 더 있습니다.", none: (t) => `“${t}”와 일치하는 사람이 없습니다.`, onlyYou: (t) => `“${t}”와 일치하는 사람은 본인뿐입니다.`, failed: "사람을 검색하지 못했습니다.", kept: "선택한 사람은 그대로입니다.", change: "변경", keep: "이 사람 유지", flag: "회수 미확정", request: "이 사람에 대한 회수가 확정되지 않았습니다", count: (n) => `${n}명` },
  ar: { hint: "ابحث باسم الدخول أو معرّف المستخدم.", first: "يُعرض أول 25 حسب معرّف المستخدم. ابحث للعثور على غيرهم.", refine: "يطابق أشخاص أكثر مما هو معروض.", none: (t) => `لا يطابق «${t}» أحدًا.`, onlyYou: (t) => `لا يطابق «${t}» غيرك`, failed: "تعذّر البحث عن الأشخاص.", kept: "يبقى الشخص الذي اخترته كما هو.", change: "تغيير", keep: "الإبقاء على هذا الشخص", flag: "الإبطال غير مؤكَّد", request: "إبطال يخص هذا الشخص غير مؤكَّد", count: (n) => `عدد الأشخاص: ${n}` },
};
const person = (id, login, flag = false) => ({ user_id: id, login, revocation_unconfirmed: flag });
const many = Array.from({ length: 25 }, (_, i) => person(`user-${String(i + 1).padStart(3, "0")}`, `person${i + 1}@example.invalid`));
const three = [person("sample-admin-2", "admin2@example.invalid", true), person("sample-member", "member-with-a-long-sign-in-name@example.invalid"), person("user-without-login", "")];
const pageBody = (qs, rows, change = {}) => ({ rows, search_input: qs.get("q") ?? "", query: { q: (qs.get("q") ?? "").toLowerCase(), id: qs.get("id") ?? "", limit: Number(qs.get("limit") ?? 25) }, next: "", reference_at: "2026-11-26T08:45:18Z", observed_at: "2026-11-26T08:45:18Z", revision: "r1", order: "user_id_asc",
  count: { value: String(rows.length), certainty: "exact" }, work_units: rows.length, scan_stopped: false, ...change });
const isList = (url) => new URL(url).pathname.endsWith("/identity/access/subjects");

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
  const seen = [];
  const began = Date.now();
  page.on("response", (r) => { const path = new URL(r.url()).pathname; if (path === "/api/v1/session" || path === "/api/v1/identity/access" || path.endsWith("/identity/access/subjects")) seen.push(`${Date.now() - began}ms ${r.request().method()} ${path.replace("/api/v1", "")} ${r.status()}`); });
  const mark = (tag) => page.evaluate((t) => { const form = document.querySelector(".live-panel form.live-step"); if (form) form.dataset.qaMount = t; }, tag);
  const mounted = async (tag, at) => {
    const now = await page.evaluate(() => document.querySelector(".live-panel form.live-step")?.dataset.qaMount ?? "none");
    check(`${at}: the request form was not replaced while it was being used`, now === tag, `mark ${now}; last responses: ${seen.slice(-8).join(" | ")}`);
  };
  const open = async (lang, view = "grants") => {
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/access?${new URLSearchParams({ lang, view })}`);
    await page.locator(view === "grants" ? "#access-list" : '[data-live="effects"]').waitFor();
    if (view === "grants" && page.viewportSize().width < 1280) { await page.locator(".access-grant-open").waitFor(); return; }
    await page.locator(".person-picker .person-status, [data-person-none]").first().waitFor();
    await page.waitForFunction(() => { const s = document.querySelector(".person-picker .person-status"); return !s || !/…$/.test(s.textContent.trim()); });
  };
  const picker = ".live-panel .person-picker";
  const status = async (root) => (await page.locator(`${root} .person-status`).innerText()).trim();

  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("en");
  const real = await page.locator(`${picker} .person-result`).evaluateAll((bs) => bs.map((b) => b.dataset.user));
  check("on the real service the request form lists the people a grant can be requested for, without the signed-in person, from one bounded read", real.length >= 1 && !real.includes("sample-admin") && [...real].sort().join() === real.join() && asked.length >= 1 && asked.every((s) => s === "?limit=25"), JSON.stringify([real, asked]));
  check("the person field has its label, hint and a search input described by the hint and the status", (await page.locator(`${picker} label[for="access-person"]`).count()) === 1 && (await page.locator("#access-person-hint").innerText()).trim() === copy.en.hint && (await page.locator("#access-person").getAttribute("aria-describedby")) === "access-person-hint access-person-status" && (await page.locator("#access-person").getAttribute("type")) === "search");
  await page.locator("#access-person").fill("ADMIN2");
  await page.locator("#access-person").press("Enter");
  await page.waitForFunction(() => document.querySelectorAll(".live-panel .person-result").length === 1);
  check("Enter searches without submitting the form, and the service answers the one match", asked[asked.length - 1] === "?q=ADMIN2&limit=25" && (await page.locator(".live-panel .service-summary").count()) === 0 && (await page.locator(`${picker} .person-result`).getAttribute("data-user")) === "sample-admin-2" && await page.evaluate(() => document.activeElement?.id === "access-person"));
  await page.locator(`${picker} .person-result`).focus(); await page.keyboard.press("Enter");
  await page.locator(`${picker} .person-chosen`).waitFor();
  check("choosing by keyboard shows the chosen person, closes the search and moves focus to Change", (await page.locator(`${picker} .person-chosen`).innerText()).includes("admin2@example.invalid") && (await page.locator(`${picker} .person-search`).count()) === 0 && await page.evaluate(() => document.activeElement?.classList.contains("person-toggle")) && (await page.locator(`${picker} .person-toggle`).innerText()).trim() === copy.en.change);
  await page.keyboard.press("Enter");
  await page.locator("#access-person").waitFor();
  check("Change reopens the search with focus in the input and offers to keep the person", await page.evaluate(() => document.activeElement?.id === "access-person") && (await page.locator(`${picker} .person-toggle`).innerText()).trim() === copy.en.keep);
  await page.locator("#access-person").fill("admin");
  await page.locator(`${picker} .person-search-button`).click();
  await page.waitForFunction(() => document.querySelector(".live-panel .person-status")?.textContent.includes("people") || document.querySelector(".live-panel .person-status")?.textContent.includes("person"));
  const admins = await page.locator(`${picker} .person-result`).evaluateAll((bs) => bs.map((b) => b.dataset.user));
  check("a search that also matches the signed-in person leaves them out", admins.length >= 1 && !admins.includes("sample-admin"), admins.join());
  await page.locator("#access-person").fill("zzzz-nobody"); await page.locator("#access-person").press("Enter");
  await page.waitForFunction((t) => document.querySelector(".live-panel .person-status")?.textContent.trim() === t, copy.en.none("zzzz-nobody"));
  check("a search that matches no one says so and keeps the chosen person", (await page.locator(`${picker} .person-result`).count()) === 0 && (await page.locator(`${picker} .person-chosen`).innerText()).includes("admin2@example.invalid"));
  await page.locator("#access-person").fill("admin@example.invalid"); await page.locator("#access-person").press("Enter");
  await page.waitForFunction((t) => document.querySelector(".live-panel .person-status")?.textContent.trim() === t, copy.en.onlyYou("admin@example.invalid"));
  check("a search that matches only the signed-in person says a grant cannot be requested for oneself", (await page.locator(`${picker} .person-result`).count()) === 0);

  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 390], ["ko", 390], ["ar", 390]]) {
    const at = `${lang} ${width}`;
    await page.setViewportSize({ width, height: 1000 });
    let mode = "many", requestSentence = "";
    await page.route((url) => isList(url.toString()), (route) => {
      const qs = new URL(route.request().url()).searchParams;
      if (mode === "fail") return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
      if (qs.get("q") === "per") return route.fulfill({ status: 200, json: pageBody(qs, many, { next: "s.2", count: { value: "25", certainty: "lower_bound" } }) });
      if (qs.get("q")) return route.fulfill({ status: 200, json: pageBody(qs, three.filter((r) => (r.login + r.user_id).toLowerCase().includes(qs.get("q").toLowerCase()))) });
      return route.fulfill({ status: 200, json: pageBody(qs, many, { next: "s.2", count: { value: "25", certainty: "lower_bound" } }) });
    });
    await open(lang);
    if (width === 390) { await page.locator(".access-grant-open").click(); await page.locator(`${picker} .person-status`).waitFor(); await page.waitForFunction(() => document.querySelectorAll(".live-panel .person-result").length === 25); }
    await mark(at);
    check(`${at}: the opened form lists the first page and says more can be searched for`, (await page.locator(`${picker} .person-result`).count()) === 25 && (await status(picker)) === copy[lang].first);
    await page.locator("#access-person").fill("per"); await page.locator("#access-person").press("Enter");
    await page.waitForFunction((t) => document.querySelector(".live-panel .person-status")?.textContent.trim().startsWith(t), copy[lang].refine);
    check(`${at}: a search with more than a page asks for more of the name and offers no Next`, (await page.locator(`${picker} .person-result`).count()) === 25 && (await page.locator(`${picker} nav, ${picker} .access-paging`).count()) === 0);
    await page.locator("#access-person").fill("e"); await page.locator("#access-person").press("Enter");
    await page.waitForFunction(() => document.querySelectorAll(".live-panel .person-result").length === 3);
    const flags = await page.locator(`${picker} .person-result`).evaluateAll((bs) => bs.map((b) => [b.dataset.user, b.querySelector(".person-flag")?.textContent.trim() ?? "", b.querySelector(".person-name").textContent.trim()]));
    check(`${at}: three matches are counted; the flagged one says so; a person without a login shows the user ID`, (await status(picker)) === copy[lang].count(3) && flags[0][1] === copy[lang].flag && flags[1][1] === "" && flags[2][2] === "user-without-login", JSON.stringify(flags));
    const shown = await inspectView(page);
    check(`${at}: nothing overflows, is cut off or loses contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), low: shown.low.slice(0, 2) }));
    const laid = await page.locator(`${picker} .person-result`).evaluateAll((bs) => bs.map((b) => { const box = b.getBoundingClientRect(), first = b.querySelector(".person-name > span").getBoundingClientRect(); const rtl = getComputedStyle(b).direction === "rtl"; const out = [...b.querySelectorAll("span, bdi")].filter((e) => { const r = e.getBoundingClientRect(); return r.right > box.right + 1 || r.left < box.left - 1; }).length; return { start: Math.round(rtl ? box.right - first.right : first.left - box.left), out, clipped: b.scrollWidth > b.clientWidth + 1 }; }));
    check(`${at}: every result starts at the same edge and no login or ID runs out of its row`, new Set(laid.map((x) => x.start)).size === 1 && laid.every((x) => x.out === 0 && !x.clipped), JSON.stringify(laid));
    const sizes = await page.evaluate(() => ({ input: Math.round(document.querySelector("#access-person").getBoundingClientRect().height), rows: [...document.querySelectorAll(".live-panel .person-result")].map((b) => Math.round(b.getBoundingClientRect().height)) }));
    check(`${at}: the input and every result are at least 44px tall`, sizes.input >= 44 && sizes.rows.every((h) => h >= 44), JSON.stringify(sizes));
    if (args.shots) await page.locator(".live-panel").screenshot({ path: `${args.shots}/m03-person-results-${lang}-${width}.png` });
    await page.locator(`${picker} .person-result[data-user="sample-admin-2"]`).click();
    await page.locator(`${picker} .person-chosen`).waitFor();
    requestSentence = (await page.locator(`${picker} [data-unconfirmed]`).innerText()).trim();
    check(`${at}: choosing the flagged person shows the request form's sentence about the unconfirmed revocation`, (await page.locator(`${picker} [data-unconfirmed]`).innerText()).includes(copy[lang].request) && (await page.locator(`${picker}`).getAttribute("data-person")) === "sample-admin-2");
    if (args.shots) await page.locator(".live-panel").screenshot({ path: `${args.shots}/m03-person-chosen-${lang}-${width}.png` });
    await mounted(at, at);
    await page.locator(`${picker} .person-toggle`).click({ timeout: 5000 });
    mode = "fail";
    await page.locator("#access-person").fill("x"); await page.locator("#access-person").press("Enter");
    await page.locator(`${picker} .person-failed`).waitFor();
    check(`${at}: a failed search says so, says the chosen person is kept, and offers another try`, (await page.locator(`${picker} .person-failed`).innerText()).includes(copy[lang].failed) && (await page.locator(`${picker} .person-failed`).innerText()).includes(copy[lang].kept) && (await page.locator(`${picker} .person-failed button`).count()) === 1 && (await page.locator(`${picker} .person-chosen`).count()) === 1);
    if (args.shots && width === 1440) await page.locator(".live-panel").screenshot({ path: `${args.shots}/m03-person-failure-${lang}-${width}.png` });
    mode = "many";
    if (width === 1440) {
      await open(lang, "effects");
      const effect = '[data-live="effects"] .person-picker';
      await page.waitForFunction(() => document.querySelectorAll('[data-live="effects"] .person-result').length === 25);
      await page.locator("#effect-person").fill("e"); await page.locator("#effect-person").press("Enter");
      await page.waitForFunction(() => document.querySelectorAll('[data-live="effects"] .person-result').length === 3);
      await page.locator(`${effect} .person-result[data-user="sample-admin-2"]`).click();
      await page.locator(`${effect} .person-chosen`).waitFor();
      check(`${at}: the effect check uses the same picker, shows its own sentence for the flag, and enables the check`, (await page.locator(`${effect} [data-unconfirmed]`).innerText()).trim().length > 0 && (await page.locator(`${effect} [data-unconfirmed]`).innerText()).trim() !== requestSentence && await page.locator('[data-live="effects"] form button[type=submit]').isEnabled());
      if (args.shots) await page.locator('[data-live="effects"]').screenshot({ path: `${args.shots}/m03-person-effects-${lang}-${width}.png` });
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
