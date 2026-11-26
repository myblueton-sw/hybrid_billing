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
const check = (name, ok, detail = "") => { results.push(`${ok ? "PASS" : "FAIL"} ${name}${!ok && detail ? " " + String(detail).replace(/\s+/g, " ").slice(0, 300) : ""}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const read = "**/api/v1/organizations/plans*";
const copy = {
  en: { sealed: "Plan proposed.", rejected: "Plan rejected.", discarded: "Plan discarded.", activated: (n) => `Activated. Relationship revision ${n} is now active.`, any: /Plan proposed|Plan rejected|Plan discarded|Activated\./ },
  ko: { sealed: "계획을 제안했습니다.", rejected: "계획이 거부되었습니다.", discarded: "계획을 폐기했습니다.", activated: (n) => `활성화했습니다. 관계 리비전 ${n}이(가) 활성 상태입니다.`, any: /계획을 제안했습니다|계획이 거부되었습니다|계획을 폐기했습니다|활성화했습니다/ },
  ar: { sealed: "اقتُرحت الخطة.", rejected: "رُفضت الخطة.", discarded: "تم تجاهل الخطة.", activated: (n) => `تم التفعيل. مراجعة العلاقات ${n} نشطة الآن.`, any: /اقتُرحت الخطة|رُفضت الخطة|تم تجاهل الخطة|تم التفعيل/ },
};
const views = ["change", "impact", "approval", "result"];
const widths = [1440, 390];
const root = () => page.locator('[data-screen-id="O02"]');

async function open(lang, view, extra = "") {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/organization-changes?lang=${lang}&view=${view}${extra}`);
  await root().waitFor();
  await page.locator(".o02-live").waitFor();
  await page.locator("#o02-announce").waitFor({ state: "attached" });
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => !document.querySelector('[data-screen-id="O02"] [aria-busy="true"]'));
}
async function contained(name) {
  const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
  check(`${name} page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
}
const announced = async () => ((await page.locator("#o02-announce").count()) === 1 && (await root().locator(".live-error").count()) === 0 ? (await page.locator("#o02-announce").textContent()) ?? "" : "the plans were not loaded: Plan proposed. Plan rejected. Plan discarded. Activated. 계획을 제안했습니다 활성화했습니다 اقتُرحت الخطة تم التفعيل");
const doneLeft = () => page.evaluate(() => /[?&]done=/.test(location.hash));

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

  let served = null, capture = true;
  page.on("response", async (r) => { if (capture && r.url().split("?")[0].endsWith("/api/v1/organizations/plans") && r.status() === 200) served = await r.json().catch(() => served); });

  for (const lang of ["en", "ko", "ar"]) {
    for (const width of widths) {
      await page.setViewportSize({ width, height: 1000 });
      for (const view of views) {
        await open(lang, view);
        check(`${lang} ${width} ${view} is live with its language and direction`, (await root().getAttribute("data-source-state")) === "live" && (await root().getAttribute("lang")) === lang && (await root().getAttribute("dir")) === (lang === "ar" ? "rtl" : "ltr"));
        const text = await root().innerText();
        check(`${lang} ${width} ${view} no placeholder text and no load failure`, !/undefined|NaN|\[object Object\]/.test(text) && (await root().locator(".live-error").count()) === 0, text.match(/.{0,40}(undefined|NaN|\[object Object\]).{0,40}/)?.[0]);
        await contained(`${lang} ${width} ${view}`);
        if (args.shots) await page.screenshot({ path: `${args.shots}/o02-${view}-${lang}-${width}.png`, fullPage: true });
      }
    }
  }
  capture = false;
  const plans = Array.isArray(served?.plans) ? served.plans : [];
  notes.push(`served plans ${plans.length}: ${plans.map((p) => `${p.id}:${p.state}:${p.activated_revision ?? ""}`).join(" ").slice(0, 300)}`);

  await page.setViewportSize({ width: 1440, height: 1000 });
  for (const lang of ["en", "ko", "ar"]) {
    for (const done of ["sealed", "rejected", "discarded", "activated:99"]) {
      await open(lang, "impact", `&plan=no-such-plan&done=${done}`);
      const said = await announced();
      check(`${lang} arrival ${done} for a plan that is not loaded announces nothing`, !copy[lang].any.test(said) && !(await doneLeft()), said);
    }
  }
  const synthetic = { id: "qa-plan-1", state: "sealed", created: "2026-11-24T00:00:00Z", proposer: "sample-admin-2", proposer_login: "admin2@example.invalid", reason: "synthetic rendered check", change: { kind: "hierarchy", member: "sample-acct-01", action: "move", target: "", from: "2026-12-01" }, member_name: "Account 01 (sample)", base: 1, kinds: [], violations: [], members: [], impact: {} };
  const plan = plans[0] ?? synthetic;
  notes.push(plans[0] ? `arrival checks use the served plan ${plan.id}` : "the sample has no plan: the arrival checks use a plan added to the read's answer in the browser");
  {
    const stage = (change) => page.route(read, async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      const list = Array.isArray(body.plans) ? body.plans : [];
      body.plans = list.some((p) => p.id === plan.id) ? list.map((p) => (p.id === plan.id ? change(p) : p)) : [...list, change(plan)];
      await route.fulfill({ response, json: body });
    });
    for (const lang of ["en", "ko", "ar"]) {
      await stage((p) => ({ ...p, state: "activated", activated_revision: 7 }));
      await open(lang, "result", `&plan=${encodeURIComponent(plan.id)}&done=activated:99`);
      let said = await announced();
      check(`${lang} an activation announces the plan's own revision, not the one in the address`, said.includes(copy[lang].activated("7")) && !said.includes("99"), said);
      await open(lang, "result", `&plan=${encodeURIComponent(plan.id)}&done=sealed`);
      said = await announced();
      check(`${lang} an arrival that does not match the plan's state announces nothing`, !copy[lang].any.test(said), said);
      await page.unroute(read);
      await stage((p) => ({ ...p, state: "sealed" }));
      await open(lang, "approval", `&plan=${encodeURIComponent(plan.id)}&done=sealed`);
      said = await announced();
      check(`${lang} a matching arrival is announced`, said.includes(copy[lang].sealed) && !(await doneLeft()), said);
      if (args.shots && lang === "en") await page.screenshot({ path: `${args.shots}/o02-arrived-sealed-en.png`, fullPage: true });
      await page.unroute(read);
    }
  }
  {
    const made = (from, to) => Array.from({ length: from - to }, (_, i) => ({ id: `qa-closed-${String(from - i).padStart(3, "0")}`, state: "discarded", closed: "discarded", created: "2026-11-20T00:00:00Z", proposer: "sample-admin",
      proposer_login: "admin@example.invalid", reason: "synthetic history", change: { kind: "hierarchy", member: `sample-acct-0${1 + ((from - i) % 6)}`, action: "end", target: "", from: "2026-12-01" }, member_name: "", base: 1, kinds: [], violations: [], members: [], impact: {} }));
    const facts = { total: 120, next: "71", kept: 200, dropped: 3 };
    let historyAsked = [], failSearch = false;
    await page.route(read, async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      await route.fulfill({ response, json: { ...body, plans: made(120, 70), closed: facts } });
    });
    await page.route((url) => url.pathname === "/api/v1/organizations/plans/closed", async (route) => {
      const u = new URL(route.request().url());
      historyAsked.push(u.search);
      const q = u.searchParams.get("q"), after = Number(u.searchParams.get("after") ?? 0);
      if (q && failSearch) return route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ code: "dependency_unavailable", request_id: "o02-history-qa" }) });
      if (q === "zz") return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ plans: [], next: "", matching: 0, closed: facts, q }) });
      if (q) return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ plans: made(110, 107), next: "", matching: 3, closed: facts, q }) });
      const from = after ? after - 1 : 120, to = Math.max(0, from - 50);
      return route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ plans: made(from, to), next: to > 0 ? String(to + 1) : "", matching: 120, closed: facts, q: "" }) });
    });
    const rowsShown = () => root().locator(".o02-results tbody tr").count();
    for (const lang of ["en", "ko", "ar"]) {
      for (const width of [1440, 390]) {
        const at = `${lang} ${width} history`;
        historyAsked = []; failSearch = false;
        await page.setViewportSize({ width, height: 1000 });
        await open(lang, "result");
        check(`${at} shows the first fifty of 120 with the count and offers more`, (await rowsShown()) === 50 && /50/.test(await root().locator(".o02-closed-count").innerText()) && /120/.test(await root().locator(".o02-closed-count").innerText())
          && (await root().locator(".o02-closed-more").count()) === 1 && (await root().locator(".o02-closed-end").count()) === 0 && historyAsked.length === 0);
        await root().locator(".o02-closed-more").click();
        await page.waitForFunction(() => document.querySelectorAll(".o02-results tbody tr").length === 100);
        check(`${at} the next page continues after the last plan shown`, historyAsked.length === 1 && historyAsked[0].includes("after=71") && /100/.test(await root().locator(".o02-closed-count").innerText()) && (await root().locator(".o02-closed-end").count()) === 0);
        await root().locator(".o01-more-button").click();
        await page.waitForFunction(() => document.querySelectorAll(".o02-results tbody tr").length === 120);
        check(`${at} at the end it says that older plans are not kept`, (await root().locator(".o02-closed-end").count()) === 1 && /200/.test(await root().locator(".o02-closed-end").innerText()) && (await root().locator(".o01-more-button").count()) === 0
          && new Set(await root().locator(".o02-results [data-plan]").evaluateAll((els) => els.map((e) => e.dataset.plan))).size === 120);
        await contained(`${at} whole history`);
        if (args.shots) await page.screenshot({ path: `${args.shots}/o02-history-end-${lang}-${width}.png`, fullPage: false });
        await page.locator("#o02-closed-q").fill("acct");
        await page.locator("#o02-closed-q").press("Enter");
        await page.waitForFunction(() => document.querySelectorAll(".o02-results tbody tr").length === 3);
        check(`${at} a search shows only what the service matched, with its count`, historyAsked.some((s) => s.includes("q=acct")) && /3/.test(await root().locator(".o02-closed-count").innerText()));
        const row = await page.locator(".o02-closed-search .o01-search-row").evaluate((el) => { const b = el.querySelector("button").getBoundingClientRect(); const r = el.getBoundingClientRect(); return { row: r.width, button: b.height }; });
        check(`${at} the search row is not stretched and its button is one line`, row.row <= 481 && row.button <= 60, JSON.stringify(row));
        check(`${at} the applied search is shown with a way to clear it`, (await root().locator(".o02-closed-applied").innerText()).includes("acct") && (await root().locator(".o02-closed-clear").count()) === 1);
        await page.locator(".o02-closed-search").scrollIntoViewIfNeeded();
        if (args.shots) await page.screenshot({ path: `${args.shots}/o02-history-search-${lang}-${width}.png`, fullPage: false });
        await root().locator(".o02-closed-clear").click();
        await page.waitForFunction(() => document.querySelectorAll(".o02-results tbody tr").length === 50);
        check(`${at} clearing the search returns to the first fifty`, (await root().locator(".o02-closed-applied").count()) === 0 && /120/.test(await root().locator(".o02-closed-count").innerText()));
        await page.locator("#o02-closed-q").fill("acct");
        await page.locator("#o02-closed-q").press("Enter");
        await page.waitForFunction(() => document.querySelectorAll(".o02-results tbody tr").length === 3);
        await page.locator("#o02-closed-q").fill("zz");
        await page.locator("#o02-closed-q").press("Enter");
        await root().locator(".live-empty").first().waitFor();
        check(`${at} a search with no match says so and shows no row`, (await rowsShown()) === 0 && (await root().locator('[role="alert"]').count()) === 0);
        failSearch = true;
        await page.locator("#o02-closed-q").fill("fail");
        await page.locator("#o02-closed-q").press("Enter");
        await root().locator('section[aria-labelledby="o02-closed-title"] [role="alert"]').waitFor();
        check(`${at} a failed search is stated with a way to repeat it, not shown as no match`, (await rowsShown()) === 0 && (await root().locator('section[aria-labelledby="o02-closed-title"] [role="alert"] button').count()) === 1 && (await root().locator(".live-empty").count()) === 0);
        await contained(`${at} failed search`);
      }
    }
    await page.unrouteAll({ behavior: "wait" });
  }
  const sent = [], asked = [];
  page.on("request", (r) => {
    const u = new URL(r.url());
    if (r.method() === "POST" && u.pathname.startsWith("/api/v1/")) sent.push({ path: u.pathname, body: r.postData() ?? "" });
    if (r.method() === "GET" && (u.pathname === "/api/v1/organizations" || u.pathname === "/api/v1/organizations/targets")) asked.push(u.pathname + u.search);
  });
  const resultIds = (field) => page.locator(`button[data-result="${field}"]`).evaluateAll((els) => els.map((e) => e.dataset.id.replace("sample-", "")).join(","));
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 1024, 390]) {
      const at = `${lang} ${width} form`;
      await page.setViewportSize({ width, height: 1000 });
      asked.length = 0; sent.length = 0;
      await open(lang, "change");
      await page.locator('button[data-result="o02-member"]').first().waitFor();
      check(`${at} opens with a search and never loads whole relationship listings`, (await page.locator("input#o02-member[type=search]").count()) === 1 && (await page.locator("select#o02-member").count()) === 0
        && asked.length > 0 && asked.every((c) => /[?&]limit=(1|10)(&|$)/.test(c) || c.includes("member=")), asked.join(" "));
      check(`${at} lists units first and then at most ten members`, (await resultIds("o02-member")).startsWith("unit-north-data,unit-north-ops,unit-south-web,acct-01") && (await page.locator('button[data-result="o02-member"]').count()) <= 13, await resultIds("o02-member"));
      await page.locator("#o02-kind-0").check();
      await page.locator('input[name="o02-change"]').first().check();
      await page.locator("select#o02-target").selectOption("sample-le-south");
      await page.locator("#o02-from").fill("2027-01-01");
      await page.locator("#live-reason").fill("rendered check");
      await page.locator("#o02-member").fill("unit-north-ops");
      await page.locator("#o02-member").press("Enter");
      await page.locator('button[data-result="o02-member"][data-id="sample-unit-north-ops"]').click();
      await page.locator("button#o02-member").click();
      await page.locator("input#o02-member").waitFor();
      check(`${at} the form is complete before the Enter check`, (await page.locator(".o02-picker").first().locator(".o02-chosen").innerText()).includes("sample-unit-north-ops") && (await page.locator("select#o02-target").inputValue()) === "sample-le-south" && sent.length === 0);
      await page.locator("#o02-member").fill("acct-0");
      await page.locator("#o02-member").press("Enter");
      await page.waitForFunction(() => { const v = [...document.querySelectorAll('button[data-result="o02-member"]')].map((e) => e.dataset.id); return v.length > 0 && v.every((x) => x.includes("acct-0")); });
      await page.waitForLoadState("networkidle");
      check(`${at} Enter in the search field searches and sends nothing`, sent.length === 0 && (await root().locator(".service-error-summary, [role=alert]").count()) === 0 && (await page.evaluate(() => document.activeElement?.id)) === "o02-member" && (await resultIds("o02-member")) === "acct-01,acct-02,acct-03,acct-04,acct-05,acct-06" && asked.some((c) => c.includes("q=acct-0") && c.includes("subject=account")), `${sent.length} ${await resultIds("o02-member")}`);
      if (args.shots) await page.screenshot({ path: `${args.shots}/o02-form-search-${lang}-${width}.png`, fullPage: true });
      await page.locator('button[data-result="o02-member"][data-id="sample-acct-01"]').click();
      await page.locator("button#o02-member").waitFor();
      const chosen = page.locator(".o02-picker").first().locator(".o02-chosen");
      check(`${at} the chosen member is shown with its identifier and the search closes`, (await chosen.innerText()).includes("sample-acct-01") && (await chosen.innerText()).includes("Account 01") && (await page.locator('button[data-result="o02-member"]').count()) === 0);
      const present = page.locator("#o02-current-title").locator("xpath=ancestor::aside[1]");
      await present.locator("ul.o02-ranges").first().waitFor();
      check(`${at} the present targets come from one exact read`, (await present.innerText()).includes("North operations") && asked.filter((c) => c.includes("member=sample-acct-01") && c.includes("kind=hierarchy") && !c.includes("subject=")).length === 1);
      check(`${at} hierarchy targets are still a select of parties`, (await page.locator("select#o02-target option").count()) >= 6);
      await page.locator('input[name="o02-kind"][value="view_group"]').check();
      await page.locator('input[name="o02-change"]').first().check();
      await page.locator('button[data-result="o02-target"]').first().waitFor();
      check(`${at} view group targets are searched on the server`, (await resultIds("o02-target")) === "view-north,view-south" && asked.some((c) => c.startsWith("/api/v1/organizations/targets?") && c.includes("kind=view_group")), await resultIds("o02-target"));
      await page.waitForFunction(() => { const a = document.querySelector("#o02-current-title")?.closest("aside"); return !!a && !a.querySelector("ul.o02-ranges") && !!a.querySelector("dd .live-quiet"); });
      check(`${at} the present targets follow the kind: this account is in no view group`, (await present.locator("ul.o02-ranges").count()) === 0 && (await present.locator("dd .live-quiet").count()) >= 1 && asked.some((c) => c.includes("member=sample-acct-01") && c.includes("kind=view_group")));
      await page.locator('button[data-result="o02-target"][data-id="sample-view-north"]').click();
      await page.locator("button#o02-target").waitFor();
      await contained(`${at} chosen`);
      const text = await root().innerText();
      check(`${at} no placeholder text`, !/undefined|NaN|\[object Object\]/.test(text));
      if (args.shots) await page.screenshot({ path: `${args.shots}/o02-form-chosen-${lang}-${width}.png`, fullPage: true });
      await page.locator("button#o02-member").click();
      await page.locator("input#o02-member").waitFor();
      await page.locator('button[data-result="o02-member"]').first().waitFor();
      await page.locator("#o02-member-cancel").click();
      await page.locator("button#o02-member").waitFor();
      check(`${at} Cancel closes the search and keeps the choice`, (await chosen.innerText()).includes("sample-acct-01") && (await page.locator('button[data-result="o02-member"]').count()) === 0 && (await page.evaluate(() => document.activeElement?.id)) === "o02-member");
      await page.locator("button#o02-member").click();
      await page.locator('button[data-result="o02-member"]').first().waitFor();
      await page.locator('button[data-result="o02-member"]').first().focus();
      await page.keyboard.press("ArrowDown");
      check(`${at} arrow keys in the results choose nothing`, (await chosen.innerText()).includes("sample-acct-01") && (await page.locator('button[data-result="o02-member"]').count()) > 0);
      check(`${at} Change reopens the search and keeps the choice shown`, (await chosen.innerText()).includes("sample-acct-01") && (await page.locator('button[data-result="o02-member"]').count()) > 0, `${await chosen.innerText()} | ${await page.locator('button[data-result="o02-member"]').count()} | ${await page.locator(".o02-picker").first().innerText()}`);
      let refusedSearch = 0;
      await page.route((url) => url.pathname === "/api/v1/organizations" && url.searchParams.get("q") === "zz", (route) => { refusedSearch++; return route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ code: "dependency_unavailable", request_id: "o02-search-qa" }) }); });
      await page.locator("#o02-member").fill("zz");
      await page.locator(".o02-picker").first().locator(".o02-search button").click();
      const alert = page.locator(".o02-picker").first().locator('[role="alert"]');
      await alert.waitFor();
      check(`${at} a failed search is stated, not shown as no match`, refusedSearch > 0 && (await alert.locator("button").count()) === 1 && (await page.locator('button[data-result="o02-member"]').count()) === 0 && (await chosen.innerText()).includes("sample-acct-01"));
      if (args.shots) await page.screenshot({ path: `${args.shots}/o02-form-failed-${lang}-${width}.png`, fullPage: true });
      await page.unrouteAll({ behavior: "wait" });
      await alert.locator("button").click();
      await page.locator(".o02-picker").first().locator('p.live-note[role="status"]').waitFor();
      check(`${at} repeating it answers that nothing matches`, (await page.locator('button[data-result="o02-member"]').count()) === 0 && (await page.locator(".o02-picker").first().locator('[role="alert"]').count()) === 0);
      check(`${at} nothing was sent while choosing`, sent.length === 0, sent.map((x) => x.path).join(" "));
    }
  }
  check("no unexpected JavaScript errors", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("runner completed without unexpected exception", false, String(e));
  if (args.shots) await page.screenshot({ path: `${args.shots}/runner-failure.png`, fullPage: true }).catch(() => {});
} finally {
  results.forEach((r) => console.log(r));
  notes.forEach((n) => console.log("NOTE " + n));
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
  await browser.close();
}
