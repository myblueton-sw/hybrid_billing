import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import assert from "node:assert/strict";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .reduce(
      (pairs, value, i, all) =>
        value.startsWith("--")
          ? [...pairs, [value.slice(2), all[i + 1]]]
          : pairs,
      [],
    ),
);
const require = createRequire(import.meta.url),
  { chromium } = require(args["playwright-module"] || "playwright");
const url = args.url || "http://127.0.0.1:8787/",
  output = resolve(args.output || "qa-output");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const context = await browser.newContext({
    viewport: { width: 1440, height: 1000 },
  }),
  page = await context.newPage(),
  checks = [],
  errors = [],
  external = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("request", (r) => {
  if (!r.url().startsWith(new URL(url).origin)) external.push(r.url());
});
const ok = (n) => checks.push(n),
  rows = (n) =>
    page.waitForFunction(
      (n) => document.querySelectorAll("tbody tr").length === n,
      n,
    );
try {
  await page.goto(url + "#contents?lang=en&mode=unconnected");
  await page.locator("#design-directory-results a").first().waitFor();
  assert.deepEqual(await page.locator(".tnb button").allTextContents(), [
    "Overview",
    "Billing operations",
    "Sources & evidence",
    "Pricing & calculation",
    "Customer portal",
    "Administration",
  ]);
  ok("baseline six-domain order");
  const headerStyles = await page
    .locator(".top-actions > a,.top-actions > button,.header-select > button")
    .evaluateAll((es) =>
      es.map((e) => ({
        height: e.getBoundingClientRect().height,
        radius: getComputedStyle(e).borderRadius,
        shadow: getComputedStyle(e).boxShadow,
      })),
    );
  assert(
    headerStyles.every(
      (s) => s.height === 36 && s.radius === "0px" && s.shadow === "none",
    ),
  );
  ok("header controls share 36px square shadow-free styling");
  await page.locator("#language").focus();
  await page.keyboard.press("ArrowDown");
  await page.getByRole("listbox", { name: "Language" }).waitFor();
  await page.keyboard.press("End");
  await page.keyboard.press("Enter");
  await page.waitForFunction(() => document.documentElement.lang === "ko");
  await page.locator("#language").click();
  await page.keyboard.press("Escape");
  assert.equal(
    await page.locator("#language").getAttribute("aria-expanded"),
    "false",
  );
  assert.equal(
    await page
      .locator("#language")
      .evaluate((e) => e === document.activeElement),
    true,
  );
  await page.locator("#language").click();
  await page.getByRole("option", { name: "English (US)", exact: true }).click();
  await page.waitForFunction(() => document.documentElement.lang === "en");
  ok("styled language listbox keyboard selection and Escape focus");
  await page.locator("#choose-account").click();
  await page.waitForFunction(
    () => document.activeElement.id === "header-account-search",
  );
  await page.fill("#header-account-search", "000002");
  await page.waitForFunction(
    () => document.querySelectorAll(".picker-list [role=option]").length === 1,
  );
  assert.equal(await page.locator(".picker-list [role=option]").count(), 1);
  await page.fill("#header-account-search", "not-found");
  await page.getByText("No matching account", { exact: true }).waitFor();
  await page.keyboard.press("Escape");
  await page.waitForFunction(() => !document.querySelector("dialog[open]"));
  assert.equal(
    await page
      .locator("#choose-account")
      .evaluate((e) => e === document.activeElement),
    true,
  );
  ok("styled account search, empty state, entry and Escape focus");

  assert.equal(await page.locator("#design-directory-results a").count(), 24);
  ok("directory 24-item page");
  await page.fill("#design-directory-search", "index-intake");
  await page.waitForFunction(
    () => document.querySelectorAll("#design-directory-results a").length === 1,
  );
  assert.equal(await page.locator("#design-directory-results a").count(), 1);
  ok("directory canonical ID search");
  await page.locator("#design-directory-results a").click();
  await page.getByText("Connect a source service", { exact: true }).waitFor();
  ok("default unconnected");
  assert.ok(
    await page.evaluate(
      () =>
        document
          .querySelector(".design-review")
          .compareDocumentPosition(document.querySelector("#content")) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ),
  );
  ok("review precedes content");
  await page.locator(".review-tools summary").click();
  await page.selectOption("#mode", "synthetic");
  await rows(25);
  ok("bounded baseline default25");
  assert.deepEqual(await page.locator("thead th").allTextContents(), [
    "Selection",
    "ID",
    "Source reference",
    "Amount",
    "Status",
    "Evidence reference",
  ]);
  ok("baseline intake columns");
  await page.selectOption("#page-size", "50");
  await rows(50);
  await page.selectOption("#page-size", "25");
  await rows(25);
  ok("page size25/50");
  await page.locator("#page-select").check();
  assert.equal(await page.locator("tbody input:checked").count(), 25);
  await page.locator("#review-selection").click();
  await page.locator("dialog[open] #close-detail").click();
  ok("baseline bulk review");
  await page.locator("#all-select").click();
  await page.locator("tbody input").first().uncheck();
  assert.equal(await page.locator("tbody input:checked").count(), 24);
  ok("all-results exclusions");
  await page.locator("#choose-account").click();
  await page.getByRole("option", { name: /ACC-000002/ }).click();
  await page.waitForFunction(() =>
    document.querySelector("tbody")?.textContent.includes("sample-b-REC"),
  );
  await rows(25);
  assert.equal(await page.locator("tbody input:checked").count(), 0);
  assert.match(await page.locator("tbody").innerText(), /sample-b-REC/);
  ok("account invalidates selection");
  await page.selectOption("#provider-filter", "AWS");
  await rows(19);
  await page.selectOption("#provider-filter", "");
  await rows(25);
  ok("provider filter");
  await page.fill("#record-search", "REC-0000007");
  await rows(1);
  await page.fill("#screen-search", "contents");
  await page.locator("#screen-links a").click();
  await page.goBack();
  await rows(1);
  assert.equal(await page.inputValue("#record-search"), "REC-0000007");
  ok("back restores fixture query");
  await page.fill("#record-search", "");
  await rows(25);
  await page.locator(".scenario summary").click();
  await page.locator("#scenario-error").click();
  await page.locator("#retry-load").click();
  await rows(25);
  ok("error retry recovers");
  await page.locator("#scenario-expire").click();
  await page
    .getByText("Snapshot expired. Refresh before continuing.", { exact: true })
    .waitFor();
  assert.equal(await page.locator("tbody tr").count(), 0);
  assert.equal(await page.locator("#next").isDisabled(), true);
  await page.locator("#refresh").click();
  await rows(25);
  ok("expired blocks and refresh recovers");
  await page.locator("#scenario-empty").click();
  await page
    .getByText("No matching evidence. Adjust the filters.", { exact: true })
    .waitFor();
  await page.locator("#scenario-normal").click();
  await rows(25);
  ok("empty and normal");
  await page.locator("#card-view").click();
  assert.equal(await page.locator(".cards article").count(), 25);
  await page.locator("#table-view").click();
  await rows(25);
  ok("table/card controls");
  await page.locator("#column-picker").click();
  await page.locator("dialog[open] input").first().uncheck();
  await page.locator("dialog[open] #close-detail").click();
  assert.equal(await page.locator("thead th").count(), 5);
  ok("column controls");
  if (!(await page.locator("#profile").isVisible()))
    await page.locator(".review-tools summary").click();
  await page.selectOption("#profile", "evidence");
  await rows(25);
  await page.locator('[data-extension="evidence-guide"]').waitFor();
  ok("extension renderer");
  await page.locator("#confirm-review").click();
  assert.match(
    await page.locator("dialog[open]").innerText(),
    /index-intake:field:record-search/,
  );
  assert.match(
    await page.locator("dialog[open]").innerText(),
    /index-intake:extension:evidence-guide/,
  );
  await page.locator("#owner-ack").check();
  await page.locator("#save-review").click();
  await page.waitForFunction(() =>
    document.querySelector("#review-status").textContent.startsWith("FIX"),
  );
  ok("scoped local FIX and canonical IDs");
  await page.selectOption("#sort", "desc");
  await rows(25);
  assert.doesNotMatch(await page.locator("#review-status").innerText(), /^FIX/);
  ok("visible change invalidates FIX");
  await page.goto(url + "#contents?lang=en");
  await page.locator("#confirm-review").click();
  await page.locator("#owner-ack").check();
  await page.locator("#save-review").click();
  await page.waitForFunction(
    () =>
      document.querySelector("#content .stats article:nth-child(2) strong")
        .textContent === "2",
  );
  ok("same-tab directory count updates");
  await page.locator("#choose-account").click();
  await page.getByRole("option", { name: /ACC-000002/ }).click();
  await page.waitForFunction(
    () =>
      !document.querySelector("#review-status").textContent.startsWith("FIX"),
  );
  ok("shell account change invalidates directory FIX");

  await page.locator("#confirm-review").click();
  await page.locator("#owner-ack").check();
  await page.evaluate(() => {
    const original = crypto.subtle.digest.bind(crypto.subtle);
    crypto.subtle.digest = async (...args) => {
      await new Promise((r) => setTimeout(r, 350));
      return original(...args);
    };
  });
  const before = await page.evaluate(() =>
    localStorage.getItem("hb-console-review-v1"),
  );
  await page.locator("#save-review").click();
  await page.locator("#owner-ack").uncheck();
  await page.waitForTimeout(450);
  assert.equal(
    await page.evaluate(() => localStorage.getItem("hb-console-review-v1")),
    before,
  );
  await page.keyboard.press("Escape");
  ok("revoked acknowledgment prevents save");
  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(
        url + `#index-intake?lang=${lang}&mode=synthetic&scope=sample-b`,
      );
      await rows(25);
      await page.evaluate(() => document.fonts.ready);
      assert.equal(
        await page.getAttribute("html", "dir"),
        lang === "ar" ? "rtl" : "ltr",
      );
      assert.ok(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      assert.equal(await page.locator(".tnb button").count(), 6);
      await page.waitForFunction(
        () => !document.querySelector("#confirm-review").disabled,
      );
      await page.screenshot({
        path: resolve(output, `${lang}-${width}.png`),
        fullPage: true,
      });
      ok(`${lang} ${width} shell/direction/overflow`);
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto(url + "#contents?lang=ko");
  await page.locator("#design-directory-results a").first().waitFor();
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({
    path: resolve(output, "contents-ko-1440.png"),
    fullPage: true,
  });
  await page.goto(url + "#core-customer-overview?lang=en");
  assert.equal(await page.locator(".tnb button").count(), 1);
  assert.equal(await page.locator("#choose-account").isDisabled(), true);
  ok("customer navigation remains scoped");
  await page.goto(url + "#index-intake?profile=unknown");
  await page.getByRole("alert").waitFor();
  ok("unknown profile fails closed");
  const broken = await browser.newContext();
  await broken.addInitScript(() => {
    Storage.prototype.setItem = () => {
      throw new DOMException("quota", "QuotaExceededError");
    };
  });
  const p = await broken.newPage();
  await p.goto(url + "#contents?lang=en");
  await p.locator("#confirm-review").click();
  await p.locator("#owner-ack").check();
  await p.locator("#save-review").click();
  await p
    .getByText("Save failed. No persisted FIX confirmed.", { exact: true })
    .waitFor();
  assert.doesNotMatch(await p.locator("#review-status").innerText(), /^FIX/);
  await broken.close();
  ok("save failure remains visible");
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  ok("no runtime errors or external requests");
  await writeFile(
    resolve(output, "report.json"),
    JSON.stringify(
      { checks, errors, external, browser: browser.version() },
      null,
      2,
    ),
  );
  console.log(JSON.stringify({ passed: checks.length, output }));
} finally {
  await browser.close();
}
