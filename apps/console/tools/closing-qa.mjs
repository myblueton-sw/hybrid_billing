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
await mkdir("qa-output/closing", { recursive: true });
const open = async (query = "") => {
  await page.goto(url + "#core-index-close?lang=en" + query);
  await page.locator(".closing-screen").waitFor();
};
try {
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + "#core-index-close?lang=" + lang);
      await page.locator(".closing-list tbody tr").first().waitFor();
      assert.equal(await page.locator(".closing-list tbody tr").count(), 20);
      assert.equal(
        await page.locator("[data-closing-total]").innerText(),
        "1200",
      );
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      await page.screenshot({
        path: `qa-output/closing/list-${lang}-${width}.png`,
        fullPage: true,
      });
      await page.locator(".closing-open").first().click();
      await page.locator(".closing-detail[open]").waitFor();
      assert(
        await page
          .locator(".closing-detail")
          .evaluate((e) => e.scrollWidth <= e.clientWidth + 1),
      );
      await page
        .locator(".closing-detail")
        .screenshot({ path: `qa-output/closing/detail-${lang}-${width}.png` });
      await page.keyboard.press("Escape");
      await page.locator(".closing-views button").nth(1).click();
      await page.locator(".closing-calendar").waitFor();
      assert.equal(await page.locator(".closing-days button").count(), 30);
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      await page.screenshot({
        path: `qa-output/closing/calendar-${lang}-${width}.png`,
        fullPage: true,
      });
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await open();
  await page.locator(".closing-list tbody tr").first().waitFor();
  for (const state of ["complete", "exempt", "held", "blocked", "incomplete"])
    assert.equal(
      await page.locator(`[data-state-count="${state}"]`).innerText(),
      "240",
    );
  await page.locator("#closing-query").fill("CONTRACT-000001");
  await page.waitForFunction(
    () => document.querySelector("[data-closing-total]")?.textContent === "1",
  );
  const trigger = page.locator(".closing-open").first();
  await trigger.click();
  await page.locator(".closing-detail[open]").waitFor();
  assert.match(
    await page.locator(".closing-detail").innerText(),
    /2026-11-01T09:00:00Z/,
  );
  assert.match(
    await page.locator(".closing-detail").innerText(),
    /2026-11-03T09:00:00Z/,
  );
  assert(
    await page
      .locator(".closing-detail")
      .evaluate((e) => e.contains(document.activeElement)),
  );
  assert.match(
    await page
      .locator(".closing-detail tbody tr")
      .filter({ hasText: "Accounting" })
      .innerText(),
    /Fixture evidence confirmed/,
  );
  await page
    .locator(".closing-detail")
    .screenshot({ path: "qa-output/closing/detail-en.png" });
  await page.keyboard.press("Escape");
  assert.equal(await page.locator(".closing-detail").count(), 0);
  assert(await trigger.evaluate((e) => e === document.activeElement));
  await page
    .getByRole("button", { name: "Reset filters", exact: true })
    .click();
  await page.waitForFunction(
    () =>
      document.querySelector("[data-closing-total]")?.textContent === "1200",
  );
  await page.selectOption("#closing-status", "blocked");
  await page.waitForFunction(
    () => document.querySelector("[data-closing-total]")?.textContent === "240",
  );
  await page.locator(".closing-views button").nth(1).click();
  await page.locator(".closing-calendar").waitFor();
  const counts = await page.locator(".closing-days strong").allTextContents();
  assert.equal(
    counts.reduce((n, s) => n + Number(s.split(" ")[0]), 0),
    240,
  );
  const day = page.locator(".closing-days button").first();
  await day.focus();
  await page.keyboard.press("ArrowRight");
  assert(
    await page
      .locator(".closing-days button")
      .nth(1)
      .evaluate((e) => e === document.activeElement),
  );
  await page.keyboard.press("Enter");
  await page.locator(".closing-list").waitFor();
  assert.equal(await page.locator("#closing-status").inputValue(), "blocked");
  await page.getByRole("button", { name: "Clear date", exact: true }).click();
  await page.waitForFunction(
    () => document.querySelector("[data-closing-total]")?.textContent === "240",
  );
  await page.locator(".closing-open").first().click();
  await page.locator(".closing-detail[open]").waitFor();
  assert.match(
    await page.locator(".closing-detail").innerText(),
    /ERP posting outcome is unknown/,
  );
  await page.goto(url + "#core-index-close?lang=en&scope=sample-b");
  await page.waitForFunction(() => !document.querySelector(".closing-detail"));
  assert.equal(await page.locator(".closing-detail").count(), 0);
  await page.locator(".closing-open").first().click();
  await page.locator(".closing-detail[open]").waitFor();
  assert.match(
    await page.locator(".closing-detail").innerText(),
    /sample-b:CYCLE/,
  );
  await page.keyboard.press("Escape");
  await page.locator("#closing-query").fill("not-found");
  await page
    .getByText("No closing targets match the filters.", { exact: true })
    .waitFor();
  assert.equal(await page.locator("tbody tr").count(), 0);
  await open("&close-state=revoked");
  await page.getByText("Preview access revoked", { exact: true }).waitFor();
  assert.equal(await page.locator("tbody tr").count(), 0);
  await open("&close-state=error");
  await page
    .getByText("Closing conditions could not be verified", { exact: true })
    .waitFor();
  await page
    .getByRole("button", { name: "Restore example", exact: true })
    .click();
  await page.locator(".closing-list tbody tr").first().waitFor();
  await open("&mode=unconnected");
  await page
    .getByText("Closing service not connected", { exact: true })
    .waitFor();
  assert.equal(
    await page
      .locator("[data-closing-total],tbody tr,.closing-days button")
      .count(),
    0,
  );
  await open();
  await page.locator(".closing-list").waitFor();
  assert.equal(
    await page
      .locator(".closing-list")
      .evaluate((e) => getComputedStyle(e).borderTopColor),
    "rgb(220, 225, 229)",
  );
  assert.deepEqual(errors, []);
  console.log(
    "PASS: U02 18 list/calendar/detail locale/viewport renders, bounded targets, shared filters/counts/date identity, original/current history, dialog focus/Escape, scope invalidation, empty/error/revoked/unconnected and neutral borders",
  );
} finally {
  await browser.close();
}
