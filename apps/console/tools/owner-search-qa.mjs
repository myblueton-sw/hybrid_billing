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
  en: { all: (n) => (n === 1 ? "1 person can own a hold." : `${n} people can own a hold.`), first: (s, m) => `Showing the first ${s} of ${m} people who can own a hold, by user ID. Search to find others.`, matches: (n) => (n === 1 ? "1 person matches." : `${n} people match.`), more: (s, m) => `Showing ${s} of ${m} people who match. Type more of the login or user ID to narrow the list.`, none: (q) => `No one who can own a hold matches “${q}”.`, failed: "The list of possible owners could not be read.", kept: "The owner you chose is kept.", change: "Change", keep: "Keep this owner", search: "Search", missing: "Choose an owner.", unread: "The owner's login could not be read." },
  ko: { all: (n) => `보존 보류의 책임자가 될 수 있는 사람은 ${n}명입니다.`, first: (s, m) => `보존 보류의 책임자가 될 수 있는 ${m}명 중 사용자 ID순으로 처음 ${s}명을 표시합니다. 다른 사람은 검색하세요.`, matches: (n) => `${n}명이 일치합니다.`, more: (s, m) => `일치하는 ${m}명 중 ${s}명을 표시합니다. 로그인 ID나 사용자 ID를 더 입력해 범위를 좁히세요.`, none: (q) => `“${q}”와 일치하는 책임자 후보가 없습니다.`, failed: "책임자 후보 목록을 읽을 수 없습니다.", kept: "선택한 책임자는 그대로입니다.", change: "변경", keep: "이 책임자 유지", search: "검색", missing: "책임자를 선택하세요.", unread: "책임자의 로그인 ID를 읽지 못했습니다." },
  ar: { all: (n) => `عدد من يمكنهم امتلاك تعليق: ${n}.`, first: (s, m) => `يُعرض أول ${s} من أصل ${m} ممن يمكنهم امتلاك تعليق، حسب معرّف المستخدم. ابحث للعثور على غيرهم.`, matches: (n) => `عدد المطابقين: ${n}.`, more: (s, m) => `يُعرض ${s} من أصل ${m} من المطابقين. اكتب المزيد من اسم الدخول أو معرّف المستخدم لتضييق القائمة.`, none: (q) => `لا يطابق «${q}» أحدًا ممن يمكنهم امتلاك تعليق.`, failed: "تعذّرت قراءة قائمة المالكين المحتملين.", kept: "يبقى المالك الذي اخترته كما هو.", change: "تغيير", keep: "الإبقاء على هذا المالك", search: "بحث", missing: "اختر مالكًا.", unread: "تعذّرت قراءة اسم الدخول للمالك." },
};
const pad = (n) => String(n).padStart(3, "0");
const open = async (lang) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/retention?lang=${lang}&view=holds`);
  await page.locator(".p04-tenant .p04-opener").waitFor();
  await page.waitForLoadState("networkidle");
  await page.locator(".p04-tenant .p04-opener").click();
  await page.locator(".p04-owner-field").waitFor();
  await page.waitForFunction(() => !document.querySelector(".p04-owner-results[aria-busy=true]") && (document.querySelector(".p04-owner-status")?.textContent ?? "") !== "" || !!document.querySelector(".p04-owners-failed"));
};
const field = () => page.locator(".p04-owner-field");
const status = async () => (await page.locator(".p04-owner-status").innerText()).trim();
const rows = () => page.locator(".p04-owner-results button.p04-owner-result");
const settled = () => page.waitForFunction(() => !document.querySelector(".p04-owner-results[aria-busy=true]") && !/…$/.test(document.querySelector(".p04-owner-status")?.textContent ?? ""));

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

  let served = null, token = "", accessAsked = 0;
  const seen = async (r) => {
    if (r.url().split("?")[0].endsWith("/api/v1/identity/access")) accessAsked++;
    if (r.url().split("?")[0].endsWith("/api/v1/retention/owners") && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); }
  };
  page.on("response", seen);
  await open("en");
  page.off("response", seen);
  const real = served?.owners ?? [];
  notes.push(`served: ${real.length} possible owners, matching ${served?.matching}, more ${served?.more}; access read asked ${accessAsked} times`);
  check("the real service lists the people who can own a hold and the form no longer reads the access screen's data", real.length > 0 && served.matching === real.length && served.more === false && accessAsked === 0);
  check("the opened field lists them as buttons with the count and has no select", (await rows().count()) === real.length && (await status()) === copy.en.all(real.length) && (await field().locator("select").count()) === 0);
  const further = await page.evaluate(async ([csrf, id]) => {
    const head = { "X-CSRF-Token": csrf, "X-Request-Id": "qa-owners" };
    const get = async (u) => { const r = await fetch(u, { headers: head, credentials: "include" }); return { status: r.status, body: await r.json().catch(() => null) }; };
    return { a: await get(`/api/v1/retention/owners?id=${encodeURIComponent(id)}`), b: await get(`/api/v1/retention/owners?q=no-such-person`), c: await get(`/api/v1/retention/owners?limit=21`) };
  }, [token, real[0].user_id]);
  check("the real service answers by identifier and by a term nobody matches, and refuses a limit above twenty", further.a.status === 200 && further.a.body.owners.length === 1 && further.a.body.owners[0].user_id === real[0].user_id && further.b.status === 200 && further.b.body.matching === 0 && further.c.status === 422, JSON.stringify(further).slice(0, 300));
  await rows().first().click();
  await page.locator(".p04-owner-chosen").waitFor();
  const chosen = await page.locator(".p04-owner-name").innerText();
  check("choosing a real person closes the search, shows the person and puts focus on Change", chosen.includes(real[0].user_id) && (await rows().count()) === 0 && (await page.evaluate(() => document.activeElement?.textContent ?? "")).trim() === copy.en.change, chosen);
  for (const el of await page.locator(".p04-form textarea, .p04-form .p04-text-field input").all()) await el.fill("Rendered check of the owner search");
  await page.locator(".p04-form button[type=submit]").click();
  await page.locator(".p04-confirm").waitFor({ timeout: 5000 }).catch(async () => { notes.push("form errors: " + (await page.locator(".p04-form .service-field-error").allInnerTexts()).join(" | ")); });
  const confirm = await page.locator(".p04-confirm").innerText();
  check("the confirmation names the chosen owner with the sign-in name and the identifier", confirm.includes(real[0].user_id) && (!real[0].login || confirm.includes(real[0].login)), confirm.slice(0, 200));

  const person = (n) => ({ user_id: `zz-user-${pad(n)}`, login: `person${pad(n)}@example.invalid` });
  const long = { user_id: "zz-" + "u".repeat(37), login: "a-very-long-sign-in-name-".repeat(2) + "x@example.invalid" };
  const all = [long, ...Array.from({ length: 44 }, (_, i) => person(i + 1))];
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 0, slow = 0; const asked = [];
      await page.route((url) => url.pathname === "/api/v1/retention/owners", async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.toString());
        if (fail > 0) { fail--; return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } }); }
        if (slow > 0) { slow--; await new Promise((r) => setTimeout(r, 700)); }
        const term = (q.get("q") ?? "").toLowerCase();
        const match = q.has("id") ? all.filter((p) => p.user_id === q.get("id")) : all.filter((p) => (p.user_id + " " + p.login).toLowerCase().includes(term));
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "identifier", q: q.get("q") ?? "", owners: match.slice(0, 20), matching: match.length, more: match.length > 20 } });
      });
      await open(lang);
      check(`${at} opened with more than twenty: the first twenty and the sentence that says so`, (await rows().count()) === 20 && (await status()) === copy[lang].first(20, 45) && (await page.locator(".p04-owner-search-button").innerText()).trim() === copy[lang].search);
      const box = await page.locator(".p04-owner-results").evaluate((el) => ({ h: el.getBoundingClientRect().height, scroll: el.scrollHeight, border: getComputedStyle(el).borderTopColor, min: Math.min(...[...el.querySelectorAll("button")].map((b) => b.getBoundingClientRect().height)) }));
      check(`${at} the list is capped at eight rows with its own scroll and every row is at least 44 high`, box.h <= 354 && box.scroll > box.h && box.min >= 44, JSON.stringify(box));
      if (args.shots) { await field().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-owner-first-${lang}-${width}.png`, fullPage: false }); }
      const input = page.locator(".p04-owner-search input");
      await input.fill("person00");
      await input.press("Enter");
      await settled();
      check(`${at} a term with nine matches lists them and says how many, without submitting the form`, (await rows().count()) === 9 && (await status()) === copy[lang].matches(9) && (await page.locator(".p04-confirm").count()) === 0 && asked[asked.length - 1] === "q=person00" && await input.evaluate((el) => el === document.activeElement));
      await input.fill("person");
      slow = 1;
      await page.locator(".p04-owner-search-button").click();
      await page.waitForFunction(() => document.querySelector(".p04-owner-results")?.getAttribute("aria-busy") === "true");
      const busy = await status();
      await settled();
      check(`${at} while searching the status says so, then more than twenty matches are cut with both numbers`, /…$/.test(busy) && (await rows().count()) === 20 && (await status()) === copy[lang].more(20, 44), busy);
      await input.fill("nobody-here");
      await input.press("Enter");
      await settled();
      check(`${at} no match says so and lists nothing`, (await rows().count()) === 0 && (await status()) === copy[lang].none("nobody-here"));
      if (args.shots) { await field().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-owner-none-${lang}-${width}.png`, fullPage: false }); }
      await input.fill("a-very-long");
      await input.press("Enter");
      await settled();
      await rows().first().click();
      await page.locator(".p04-owner-chosen").waitFor();
      const change = page.locator(".p04-owner-toggle");
      check(`${at} a chosen person with a long name is shown whole, the search closes and focus is on Change`, (await page.locator(".p04-owner-name").innerText()).includes(long.user_id) && (await page.locator(".p04-owner-search").count()) === 0 && (await change.innerText()).trim() === copy[lang].change && await change.evaluate((el) => el === document.activeElement) && (await change.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      const m1 = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport with the long name`, m1.page <= m1.view + 1, JSON.stringify(m1));
      if (args.shots) { await field().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-owner-chosen-${lang}-${width}.png`, fullPage: false }); }
      fail = 1;
      await change.click();
      await page.locator(".p04-owners-failed").waitFor();
      const alert = await page.locator(".p04-owners-failed").innerText();
      check(`${at} a failed search while changing keeps the choice and says so`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].kept) && (await page.locator(".p04-owner-name").innerText()).includes(long.user_id) && (await page.locator(".p04-owner-toggle").innerText()).trim() === copy[lang].keep, alert);
      if (args.shots) { await field().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-owner-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.locator(".p04-owners-failed button").click();
      await settled();
      await page.locator(".p04-owner-results").waitFor();
      check(`${at} after a retry the first twenty are listed again and focus is in the search input`, (await rows().count()) === 20 && await page.locator(".p04-owner-search input").evaluate((el) => el === document.activeElement));
      await page.locator(".p04-owner-toggle").click();
      check(`${at} "Keep this owner" closes the search with the choice unchanged and focus back on Change`, (await page.locator(".p04-owner-search").count()) === 0 && (await page.locator(".p04-owner-name").innerText()).includes(long.user_id) && (await page.evaluate(() => document.activeElement?.textContent ?? "")).trim() === copy[lang].change);
      await page.unrouteAll({ behavior: "wait" });
    }
  }
  const routed = async (slowFirst = 0) => {
    let slow = slowFirst;
    await page.route((url) => url.pathname === "/api/v1/retention/owners", async (route) => {
      const q = new URL(route.request().url()).searchParams;
      if (slow > 0) { slow--; await new Promise((r) => setTimeout(r, 1500)); }
      const term = (q.get("q") ?? "").toLowerCase();
      const match = q.has("id") ? all.filter((p) => p.user_id === q.get("id")) : all.filter((p) => (p.user_id + " " + p.login).toLowerCase().includes(term));
      await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "identifier", q: q.get("q") ?? "", owners: match.slice(0, 20), matching: match.length, more: match.length > 20 } });
    });
  };
  for (const lang of ["en", "ar"]) {
    await page.setViewportSize({ width: 1440, height: 1000 });
    await routed();
    await open(lang);
    await rows().first().focus();
    await page.keyboard.press("Enter");
    await page.locator(".p04-owner-chosen").waitFor();
    const ring = await page.locator(".p04-owner-toggle").evaluate((el) => ({ active: el === document.activeElement, visible: el.matches(":focus-visible"), outline: getComputedStyle(el).outlineStyle + " " + getComputedStyle(el).outlineWidth }));
    check(`${lang} chosen by keyboard: focus is on Change with a visible focus indication`, ring.active && ring.visible, JSON.stringify(ring));
    if (args.shots) { await field().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-owner-keyboard-${lang}-1440.png`, fullPage: false }); }
    const gap = await page.evaluate(() => { const a = document.querySelector(".p04-owner-chosen").getBoundingClientRect(), b = document.querySelector(".p04-owner-field > .service-hint:last-child").getBoundingClientRect(); return Math.round(b.top - a.bottom); });
    check(`${lang} with the search closed nothing empty sits between the chosen line and the hint`, gap <= 16 && (await page.locator(".p04-owner-status").count()) === 0, String(gap));
    await page.keyboard.press("Enter");
    await page.locator(".p04-owner-search input").waitFor();
    await settled();
    const hint = await page.evaluate(() => { const h = document.querySelector(".p04-owner-search-hint").getBoundingClientRect(), i = document.querySelector(".p04-owner-search input").getBoundingClientRect(); return { above: h.bottom <= i.top + 1, input: document.activeElement === document.querySelector(".p04-owner-search input"), current: document.querySelectorAll(".p04-owner-result[aria-current=true]").length }; });
    check(`${lang} after Change: focus is in the input, the search hint is above it and the current owner's row is marked`, hint.above && hint.input && hint.current === 1, JSON.stringify(hint));
    await page.locator(".p04-owner-toggle").focus();
    await page.keyboard.press("Enter");
    check(`${lang} "Keep this owner" by keyboard returns focus to Change`, await page.locator(".p04-owner-toggle").evaluate((el) => el === document.activeElement && el.matches(":focus-visible")));
    for (const el of await page.locator(".p04-form textarea, .p04-form .p04-text-field input").all()) await el.fill("Rendered check of the owner search");
    await page.locator(".p04-form button[type=submit]").click();
    await page.locator(".p04-confirm").waitFor();
    const text = await page.locator(".p04-confirm").innerText();
    check(`${lang} the confirmation shows the owner as the sign-in name with the identifier`, text.includes(long.login) && text.includes(long.user_id), text.slice(0, 160));
    if (args.shots) { await page.locator(".p04-confirm").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-owner-confirm-${lang}-1440.png`, fullPage: false }); }
    await page.route((url) => url.pathname === "/api/v1/commands/retention.place_hold", (route) => route.fulfill({ status: 422, json: { code: "invalid_request", retry: false, next_action: "none", request_id: "qa-refused", fields: ["owner"] } }));
    await page.locator(".p04-confirm .live-primary").click();
    await page.locator(".p04-owner-field .service-field-error").waitFor();
    await settled();
    const refused = { error: (await page.locator(".p04-owner-field .service-field-error").innerText()).trim(), rows: await rows().count(), chosen: await page.locator(".p04-owner-name").innerText(), input: await page.locator(".p04-owner-search input").count() };
    check(`${lang} when the command refuses the owner the field says so, keeps showing who it was and reopens the search with the first twenty`, refused.error.length > 0 && refused.rows === 20 && refused.chosen.includes(long.user_id) && refused.input === 1, JSON.stringify(refused));
    if (args.shots) { await field().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-owner-refused-${lang}-1440.png`, fullPage: false }); }
    await page.unrouteAll({ behavior: "wait" });
  }
  await page.setViewportSize({ width: 390, height: 1000 });
  await routed();
  await page.goto(`${base}/?service=api&lang=ko&qa=${++navigation}#/retention?lang=ko&view=expiry`);
  await page.locator(".p04-class").first().waitFor();
  await page.waitForLoadState("networkidle");
  const opener = page.locator(".p04-items .p04-opener").first();
  if ((await opener.count()) > 0) {
    await opener.click();
    await page.locator(".p04-owner-field").waitFor();
    await settled();
    const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth, rows: document.querySelectorAll(".p04-owner-result").length }));
    check("ko 390 the item-row form lists the first twenty and the page is not wider than the viewport", m.rows === 20 && m.page <= m.view + 1, JSON.stringify(m));
    if (args.shots) { await field().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-owner-item-row-ko-390.png`, fullPage: false }); }
  } else notes.push("no item row offers a hold in the sample; the item-row form was not captured");
  await page.unrouteAll({ behavior: "wait" });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await routed(2);
  await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#/retention?lang=en&view=holds`);
  await page.locator(".p04-tenant .p04-opener").waitFor();
  await page.waitForLoadState("networkidle");
  await page.locator(".p04-tenant .p04-opener").click();
  await page.locator(".p04-owner-field").waitFor();
  check("en during the very first search only the status line shows, with no empty list", (await status()) === "Searching…" && (await page.locator(".p04-owner-results").count()) === 0);
  if (args.shots) { await field().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-owner-searching-first-en-1440.png`, fullPage: false }); }
  await settled();
  await page.waitForLoadState("load");
  await page.waitForLoadState("networkidle");
  await page.unrouteAll({ behavior: "wait" });
  await routed(1);
  await page.locator(".p04-owner-search input").fill("person");
  await page.locator(".p04-owner-search input").press("Enter");
  await page.waitForFunction(() => document.querySelector(".p04-owner-results")?.getAttribute("aria-busy") === "true");
  if (args.shots) { await field().scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-owner-searching-en-1440.png`, fullPage: false }); }
  await settled();
  await page.unrouteAll({ behavior: "wait" });
  await page.route((url) => url.pathname === "/api/v1/retention/owners", (route) => route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } }));
  await open("en");
  const first = await page.locator(".p04-owners-failed").innerText();
  check("a failed first read says so without the sentence about a kept choice and offers a retry", first.includes(copy.en.failed) && !first.includes(copy.en.kept) && (await page.locator(".p04-owners-failed button").count()) === 1, first);
  await page.locator(".p04-form button[type=submit]").click();
  check("submitting with nobody chosen says to choose an owner", (await field().locator(".service-field-error").innerText()).trim() === copy.en.missing);
  await page.unrouteAll({ behavior: "wait" });
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
