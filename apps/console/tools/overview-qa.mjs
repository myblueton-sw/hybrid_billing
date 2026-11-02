import { createRequire } from "node:module";
import { mkdir } from "node:fs/promises";
import assert from "node:assert/strict";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .reduce(
      (a, v, i, all) =>
        v.startsWith("--") ? [...a, [v.slice(2), all[i + 1]]] : a,
      [],
    ),
);
const { chromium } = createRequire(import.meta.url)(
  args["playwright-module"] || "playwright",
);
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const context = await browser.newContext();
const page = await context.newPage();
const errors = [];
page.on("pageerror", (e) => errors.push(e.message));
const url = args.url || "http://127.0.0.1:8787/";
await mkdir("qa-output/overview", { recursive: true });
try {
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + "#overview?lang=" + lang + "&mode=synthetic");
      await page.locator(".overview-dashboard").waitFor();
      assert.deepEqual(await page.locator("[data-count]").allTextContents(), [
        "57",
        "15",
        "14",
        "28",
      ]);
      assert.equal(await page.locator(".overview-list li").count(), 10);
      assert.equal(await page.locator("#design-directory-results").count(), 0);
      assert.equal(
        await page.locator('#screen-links a[href*="#contents"]').count(),
        0,
      );
      assert.equal(
        await page.locator("html").getAttribute("dir"),
        lang === "ar" ? "rtl" : "ltr",
      );
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      await page.evaluate(() => document.fonts.ready);
      await page.screenshot({
        path: `qa-output/overview/${lang}-${width}.png`,
        fullPage: true,
      });
    }
  await page.goto(
    url +
      "#overview?lang=en&scope=sample-b&operations-q=stale&operations-provider=Azure",
  );
  await page.locator('[data-count="missing"]').waitFor();
  await page.locator(".overview-metric").nth(1).getByRole("link").click();
  await page.waitForFunction(
    () => document.querySelectorAll("tbody tr").length === 15,
  );
  assert.equal(await page.locator("#status-filter").inputValue(), "missing");
  assert.equal(await page.locator("#record-search").inputValue(), "");
  assert.equal(await page.locator("#provider-filter").inputValue(), "");
  assert((await page.locator("tbody").innerText()).includes("sample-b-CASE-"));
  assert(!(await page.locator("tbody").innerText()).includes("sample-a-CASE-"));
  await page.goBack();
  await page.locator(".overview-dashboard").waitFor();
  await page.locator("#overview-period").fill("2026-09");
  await page.waitForFunction(
    () => document.querySelector('[data-count="total"]')?.textContent === "0",
  );
  assert.equal(await page.locator(".overview-list li").count(), 0);
  await page.goto(url + "#overview?lang=en&scope=sample-b&my-tasks-status=missing&my-tasks-q=stale");
  await page.locator(".overview-list").nth(1).getByRole("link", {name: "View all (29)", exact: true}).click();
  await page.waitForFunction(() => document.querySelectorAll("tbody tr").length === 25);
  assert.equal(await page.locator("#status-filter").inputValue(), "");
  assert.equal(await page.locator("#record-search").inputValue(), "");
  await page.goto(url + "#overview?lang=en");
  const periodControl = page.locator("#overview-period");
  await periodControl.focus();
  await page.keyboard.press("Tab");
  assert(await page.evaluate(() => document.activeElement?.tagName !== "BODY"));
  const borders = await page.locator(".overview-metric").evaluateAll(es => es.map(e => getComputedStyle(e).borderTopColor));
  assert(borders.every(v => v === "rgb(220, 225, 229)"));
  await page.goto(url + "#overview?lang=en&mode=unconnected");
  await page
    .getByText("Overview service not connected", { exact: true })
    .waitFor();
  assert.equal(await page.locator("[data-count]").count(), 0);
  assert.equal(await page.locator(".overview-list li").count(), 0);
  await page.goto(url + "#/home?lang=en");
  await page.locator(".overview-dashboard").waitFor();
  await page.getByText("Review tools", { exact: true }).click();
  await page
    .getByRole("link", { name: "Screen directory", exact: true })
    .click();
  await page.locator("#design-directory-results").waitFor();
  assert.equal(await page.locator(".overview-dashboard").count(), 0);
  assert.deepEqual(errors, []);
  console.log(
    "PASS: 6 locale/viewport renders, exact counts, bounded lists, scope/filter drilldown, history, empty period, unconnected, /home alias, separate directory; no page errors",
  );
} finally {
  await browser.close();
}
