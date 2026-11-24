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
const copy = {
  en: { failed: "Could not be loaded.", again: "Try again" },
  ko: { failed: "불러오지 못했습니다.", again: "다시 시도" },
  ar: { failed: "تعذّر التحميل.", again: "إعادة المحاولة" },
};
const views = ["tree", "accounts", "groups", "history"];
const widths = [1440, 390];
const root = () => page.locator('[data-screen-id="O01"]');
const facts = (url) => { const u = new URL(url); return u.pathname.endsWith("/api/v1/organizations") && u.searchParams.get("kind") === "hierarchy" && u.searchParams.has("member"); };

async function open(lang, view) {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/organizations?lang=${lang}&view=${view}`);
  await root().waitFor();
  await page.locator("#o01-view-title").waitFor();
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => !document.querySelector('[data-screen-id="O01"] [aria-busy="true"]'));
}
async function contained(name) {
  const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
  check(`${name} page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
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

  for (const lang of ["en", "ko", "ar"]) {
    for (const width of widths) {
      await page.setViewportSize({ width, height: 1000 });
      for (const view of views) {
        await open(lang, view);
        check(`${lang} ${width} ${view} is live with its language and direction`, (await root().getAttribute("data-source-state")) === "live" && (await root().getAttribute("lang")) === lang && (await root().getAttribute("dir")) === (lang === "ar" ? "rtl" : "ltr"));
        const text = await root().innerText();
        check(`${lang} ${width} ${view} shows its view title`, (await page.locator("#o01-view-title").innerText()).trim().length > 0);
        check(`${lang} ${width} ${view} no placeholder text and no load failure`, !/undefined|NaN|\[object Object\]/.test(text) && (await root().locator(".live-error").count()) === 0, text.match(/.{0,40}(undefined|NaN|\[object Object\]).{0,40}/)?.[0]);
        await contained(`${lang} ${width} ${view}`);
        if (args.shots) await page.screenshot({ path: `${args.shots}/o01-${view}-${lang}-${width}.png`, fullPage: true });
      }
    }
  }

  for (const lang of ["en", "ko", "ar"]) {
    for (const width of widths) {
      await page.setViewportSize({ width, height: 1000 });
      await open(lang, "accounts");
      const rows = page.locator("#o01-list .live-open[data-id]");
      if (lang === "en" && width === 1440) { notes.push(`accounts listed ${await rows.count()}`); check("the sample lists at least one account", (await rows.count()) > 0); }
      let asked = 0;
      await page.route((url) => facts(url.toString()), (route) => { asked++; return route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ code: "dependency_unavailable", request_id: "o01-facts-qa" }) }); });
      await rows.first().click();
      const failed = root().getByText(copy[lang].failed, { exact: true });
      await failed.first().waitFor();
      const again = failed.first().locator("xpath=following-sibling::button[1]");
      check(`${lang} ${width} a failed read of the member's facts is stated with a way to repeat it`, asked > 0 && (await again.count()) === 1 && (await again.innerText()).trim() === copy[lang].again && (await failed.first().getAttribute("role")) === "alert");
      await contained(`${lang} ${width} detail with the failed read`);
      if (args.shots) await page.screenshot({ path: `${args.shots}/o01-facts-failed-${lang}-${width}.png`, fullPage: true });
      await page.unrouteAll({ behavior: "wait" });
      const answered = page.waitForResponse((r) => facts(r.url()) && r.status() === 200);
      await again.click();
      await answered;
      await page.waitForLoadState("networkidle");
      const parent = page.locator("#o01-detail-title").locator("xpath=ancestor::*[.//dd][1]").locator("dd bdi, dd strong");
      await parent.first().waitFor();
      check(`${lang} ${width} repeating the read replaces the failure with the answered facts`, (await root().getByText(copy[lang].failed, { exact: true }).count()) === 0 && (await parent.count()) > 0);
    }
  }
  const calls = [];
  page.on("request", (r) => { const u = new URL(r.url()); if (u.pathname.startsWith("/api/v1/organizations")) calls.push(u.pathname + u.search); });
  const names = (list) => list.evaluateAll((els) => els.map((e) => e.dataset.id.replace("sample-", "")).join(","));
  const item = (scope, id) => scope.locator(":scope > li").filter({ has: page.locator(`:scope > .o01-row [data-id="sample-${id}"]`) });
  const under = (li) => li.locator(":scope > .o01-under > ul.o01-branch");
  const rowsOf = (list) => list.locator(":scope > li > .o01-row .live-open[data-id]");
  const toggleOf = (li) => li.locator(":scope > .o01-row .o01-toggle");
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 1024, 390]) {
      const at = `${lang} ${width} tree`;
      await page.setViewportSize({ width, height: 1000 });
      calls.length = 0;
      await open(lang, "tree");
      const tree = root().locator("ul.o01-tree").first();
      await tree.waitFor();
      check(`${at} lists the roots in order, all closed`, (await names(rowsOf(tree))) === "le-north,le-seller,le-south" && (await root().locator('.o01-toggle[aria-expanded="true"]').count()) === 0, await names(rowsOf(tree)));
      check(`${at} a root with nothing under it has no toggle`, (await toggleOf(item(tree, "le-seller")).count()) === 0 && (await toggleOf(item(tree, "le-north")).count()) === 1);
      check(`${at} reads the tree by node and never the whole hierarchy`, calls.some((c) => c.startsWith("/api/v1/organizations/tree?")) && calls.some((c) => c.includes("group=detached")) && calls.some((c) => c.includes("unattributed=true"))
        && calls.every((c) => c.startsWith("/api/v1/organizations/tree") || c.includes("limit=1&") || c.endsWith("limit=1") || c.includes("unattributed=true") || c.includes("member=") || c.includes("/drafts") || c.includes("/targets")), calls.join(" "));
      check(`${at} hides the two groups that have nothing to show`, (await root().locator('section[aria-labelledby="o01-cycles"][hidden], section[aria-labelledby="o01-noorg"][hidden]').count()) === 2);
      const north = item(tree, "le-north");
      await toggleOf(north).click();
      await under(north).waitFor();
      const controls = await toggleOf(north).getAttribute("aria-controls");
      check(`${at} opening a root shows its units and names what the toggle controls`, (await names(rowsOf(under(north)))) === "unit-north-data,unit-north-ops" && (await toggleOf(north).getAttribute("aria-expanded")) === "true" && !!controls && (await page.locator(`[id="${controls}"]`).count()) === 1,
        await names(rowsOf(under(north))));
      check(`${at} asked for that node only`, calls.some((c) => c.includes("parent=sample-le-north")) && !calls.some((c) => c.includes("parent=sample-le-south")));
      const ops = item(under(north), "unit-north-ops");
      await toggleOf(ops).click();
      await under(ops).waitFor();
      check(`${at} a unit shows its accounts and an account cannot be opened`, (await names(rowsOf(under(ops)))) === "acct-01" && (await toggleOf(item(under(ops), "acct-01")).count()) === 0, await names(rowsOf(under(ops))));
      await contained(`${at} opened`);
      const text = await root().innerText();
      check(`${at} no placeholder text`, !/undefined|NaN|\[object Object\]/.test(text));
      if (args.shots) await page.screenshot({ path: `${args.shots}/o01-tree-open-${lang}-${width}.png`, fullPage: true });
      let refused = 0;
      await page.route((url) => url.toString().includes("/organizations/tree") && url.toString().includes("parent=sample-le-south"), (route) => { refused++; return route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ code: "dependency_unavailable", request_id: "o01-tree-qa" }) }); });
      const south = item(tree, "le-south");
      await toggleOf(south).click();
      const alert = south.locator(':scope > .o01-under > [role="alert"]');
      await alert.waitFor();
      check(`${at} a failed read under one node is stated beside it and the rest stays`, refused > 0 && (await alert.locator("button").count()) === 1 && (await names(rowsOf(under(north)))) === "unit-north-data,unit-north-ops" && (await root().locator(".live-error").count()) === 0);
      if (args.shots) await page.screenshot({ path: `${args.shots}/o01-tree-failed-${lang}-${width}.png`, fullPage: true });
      await page.unrouteAll({ behavior: "wait" });
      await alert.locator("button").click();
      await under(south).waitFor();
      check(`${at} repeating the read shows what is under the node`, (await names(rowsOf(under(south)))) === "unit-south-web" && (await south.locator(':scope > .o01-under > [role="alert"]').count()) === 0);
      await root().locator(".o01-tree-tools button").click();
      await page.waitForFunction(() => !document.querySelector('[data-screen-id="O01"] .o01-under'));
      check(`${at} collapse all closes every node`, (await root().locator('.o01-toggle[aria-expanded="true"]').count()) === 0 && (await root().locator(".o01-under").count()) === 0);
      await toggleOf(north).click();
      await under(north).waitFor();
      await item(under(north), "unit-north-data").locator(':scope > .o01-row .live-open').click();
      await page.locator("#o01-detail-title").waitFor();
      await page.locator("#o01-create-plan").waitFor();
      check(`${at} selecting a unit shows its detail with the plan link`, (await page.locator("#o01-detail-title").innerText()).includes("North data platform") && (await page.locator("#o01-create-plan").getAttribute("href")).includes("member=sample-unit-north-data"));
      await contained(`${at} with a detail`);
      if (args.shots) await page.screenshot({ path: `${args.shots}/o01-tree-detail-${lang}-${width}.png`, fullPage: true });

      await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/organizations?lang=${lang}&view=tree&on=2025-12-31`);
      await root().locator('section[aria-labelledby="o01-noorg"] ul.o01-tree').waitFor();
      await page.waitForLoadState("networkidle");
      const early = root().locator("ul.o01-tree").first();
      const loose = root().locator('section[aria-labelledby="o01-noorg"] ul.o01-tree');
      check(`${at} before any relationship the roots have nothing under them`, (await names(rowsOf(early))) === "le-north,le-seller,le-south" && (await early.locator(".o01-toggle").count()) === 0);
      check(`${at} lists who has no organization on that day`, (await root().locator("#o01-noorg").isVisible()) && (await root().locator('section[aria-labelledby="o01-cycles"][hidden]').count()) === 1 && (await rowsOf(loose).count()) === 9 && (await names(rowsOf(loose))).startsWith("acct-01,"), await names(rowsOf(loose)));
      const loner = item(loose, "unit-north-ops");
      await toggleOf(loner).click();
      const nothing = loner.locator(":scope > .o01-under > p.live-note");
      await nothing.waitFor();
      check(`${at} a row there can be opened and says when nothing is under it`, (await nothing.innerText()).trim().length > 0 && (await loner.locator(":scope > .o01-under ul").count()) === 0);
      await contained(`${at} no organization list`);
      if (args.shots) await page.screenshot({ path: `${args.shots}/o01-tree-early-${lang}-${width}.png`, fullPage: true });
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
