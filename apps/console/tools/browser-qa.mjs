import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import assert from "node:assert/strict";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .reduce(
      (pairs, value, index, all) =>
        value.startsWith("--")
          ? [...pairs, [value.slice(2), all[index + 1]]]
          : pairs,
      [],
    ),
);
const require = createRequire(import.meta.url);
const { chromium } = require(args["playwright-module"] || "playwright");
const url = args.url || "http://127.0.0.1:8787/";
const output = resolve(args.output || "qa-output");
await mkdir(output, { recursive: true });
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const checks = [],
  errors = [],
  external = [];
const context = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
});
const page = await context.newPage();
page.on("pageerror", (e) => errors.push(e.message));
page.on("request", (r) => {
  if (!r.url().startsWith(new URL(url).origin)) external.push(r.url());
});
const ok = (name) => checks.push(name);
const waitRows = () =>
  page.waitForFunction(
    () => document.querySelectorAll("tbody tr").length === 10,
  );
try {
  await page.goto(url + "#index-intake?lang=en");
  await page.getByText("Connect a source service", { exact: true }).waitFor();
  assert.equal(await page.locator("tbody tr").count(), 0);
  ok("unconnected does not synthesize results");
  await page.selectOption("#mode", "synthetic");
  await waitRows();
  ok("synthetic query bounded at10");
  await page.locator("tbody input").first().check();
  await page.selectOption("#scope", "sample-b");
  await waitRows();
  assert.match(await page.locator("tbody").innerText(), /sample-b-REC/);
  assert.equal(await page.locator("tbody input:checked").count(), 0);
  ok("scope invalidates selection");
  await page.selectOption("#scope", "sample-a");
  await page.selectOption("#scope", "sample-b");
  await waitRows();
  await page.waitForTimeout(550);
  assert.doesNotMatch(await page.locator("tbody").innerText(), /sample-a-REC/);
  ok("delayed old scope does not replace latest");
  await page.fill("#record-search", "REC-0007");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await page.waitForFunction(
    () => document.querySelectorAll("tbody tr").length === 1,
  );
  await page
    .getByRole("link", { name: "Implementation workspace", exact: true })
    .click();
  await page.goBack();
  await page.waitForFunction(
    () => document.querySelectorAll("tbody tr").length === 1,
  );
  assert.equal(await page.inputValue("#record-search"), "REC-0007");
  ok("back restores synthetic search");
  await page.fill("#record-search", "");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await waitRows();
  await page.locator(".scenario summary").click();
  await page.selectOption("#scenario", "error");
  await page.locator("#retry").waitFor();
  await page.locator("#retry").click();
  await page.locator("#retry").waitFor();
  ok("failed request exposes retry");
  await page.selectOption("#scenario", "empty");
  await page
    .getByText("No matching evidence. Adjust the filters.", { exact: true })
    .waitFor();
  assert.equal(await page.locator("tbody tr").count(), 0);
  ok("empty distinct from error");
  await page.selectOption("#scenario", "normal");
  await waitRows();
  await page.selectOption("#profile", "evidence");
  await waitRows();
  await page.locator('[data-extension="evidence-guide"]').waitFor();
  ok("profile assembles extension renderer");
  await page.locator("#confirm-review").click();
  await page.locator("#owner-ack").check();
  await page.locator("#save-review").click();
  await page.waitForFunction(() =>
    document.querySelector("#review-status").textContent.startsWith("FIX"),
  );
  ok("isolated local confirmation");
  await page.fill("#record-search", "not-submitted");
  await page.waitForFunction(
    () =>
      !document.querySelector("#review-status").textContent.startsWith("FIX"),
  );
  ok("unsubmitted visible edit invalidates FIX");
  await page.fill("#record-search", "");
  await page.waitForFunction(() =>
    document.querySelector("#review-status").textContent.startsWith("FIX"),
  );
  await page.reload();
  await waitRows();
  await page.waitForFunction(() =>
    document.querySelector("#review-status").textContent.startsWith("FIX"),
  );
  ok("same scope reload restores local evidence");
  const record = await page.evaluate(
    () => JSON.parse(localStorage.getItem("hb-console-review-v1"))[0],
  );
  assert.equal(record.variants.locale, "en");
  assert.equal(record.variants.profile, "evidence");
  assert.equal(record.variants.width, 1440);
  assert.ok(record.items.includes("index-intake:extension:evidence-guide"));
  ok("exportable variants and item metadata");
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
  ok("revoked ack prevents async save");
  await page.keyboard.press("Escape");
  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(
        url +
          `#index-intake?lang=${lang}&mode=synthetic&profile=evidence&scope=sample-b`,
      );
      await waitRows();
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
      await page.screenshot({
        path: resolve(output, `${lang}-${width}.png`),
        fullPage: true,
      });
      ok(`${lang} ${width} direction/overflow/render`);
    }
  await page.goto(url + "#index-intake?profile=unknown");
  await page.getByRole("alert").waitFor();
  assert.equal(await page.locator("tbody tr").count(), 0);
  ok("unknown profile fails closed");
  const broken = await browser.newContext();
  await broken.addInitScript(() => {
    Storage.prototype.setItem = function () {
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
  ok("storage failure never reports persisted FIX");
  await broken.close();
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
  console.log(
    JSON.stringify({
      passed: checks.length,
      output,
      browser: browser.version(),
    }),
  );
} finally {
  await browser.close();
}
