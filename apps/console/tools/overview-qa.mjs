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
const open = async (query = "") => {
  await page.goto(url + "#overview?lang=en" + query);
  await page.locator(".unified-overview").waitFor();
};
try {
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + "#overview?lang=" + lang);
      await page.locator("[data-coverage]").waitFor();
      assert.equal(
        await page.locator("[data-coverage]").innerText(),
        "11 / 12",
      );
      assert.equal(await page.locator("[data-closing-count]").innerText(), "6");
      assert.equal(await page.locator(".unified-month").count(), 6);
      assert.equal(await page.locator(".unified-matrix tbody tr").count(), 6);
      assert.equal(await page.locator("#design-directory-results").count(), 0);
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      assert.equal(
        await page.locator("html").getAttribute("dir"),
        lang === "ar" ? "rtl" : "ltr",
      );
      await page.evaluate(() => document.fonts.ready);
      await page.screenshot({
        path: `qa-output/overview/integrated-${lang}-${width}.png`,
        fullPage: true,
      });
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await open();
  assert.match(
    await page.locator('[data-total="USD"]').innerText(),
    /USD 17,840.00/,
  );
  assert.match(
    await page.locator('[data-total="KRW"]').innerText(),
    /KRW 6,900,000/,
  );
  await page.locator(".unified-month").first().click();
  await page.waitForFunction(() => document.querySelector("#overview-period")?.value === "2026-05");
  assert.equal(await page.locator("#overview-period").inputValue(), "2026-05");
  assert.match(
    await page.locator('[data-total="USD"]').innerText(),
    /16,470.00/,
  );
  await page.goBack();
  await page.waitForFunction(() => document.querySelector("#overview-period")?.value === "2026-10");
  assert.equal(await page.locator("#overview-period").inputValue(), "2026-10");
  await page.locator(".unified-picker").first().locator("summary").click();
  await page
    .locator(".unified-picker")
    .first()
    .getByLabel("All", { exact: true })
    .click();
  await page
    .getByText("No data in the selected scope", { exact: true })
    .waitFor();
  assert.equal(await page.locator("[data-total]").count(), 0);
  await page
    .locator(".unified-picker")
    .first()
    .getByLabel("AWS", { exact: true })
    .click();
  await page.waitForFunction(() => document.querySelector("[data-closing-count]")?.textContent === "3");
  assert.equal(await page.locator("[data-closing-count]").innerText(), "3");
  assert.match(
    await page.locator('[data-total="USD"]').innerText(),
    /5,640.00/,
  );
  await page
    .locator(".unified-picker")
    .first()
    .getByLabel("Azure", { exact: true })
    .click();
  await page.waitForFunction(() => document.querySelector("[data-closing-count]")?.textContent === "3");
  assert.equal(await page.locator("[data-closing-count]").innerText(), "3");
  await page.locator(".unified-picker").nth(1).locator("summary").click();
  await page
    .locator(".unified-picker")
    .nth(1)
    .getByLabel("All", { exact: true })
    .click();
  await page
    .locator(".unified-picker")
    .nth(1)
    .getByLabel("Example organization A", { exact: true })
    .click();
  await page.waitForFunction(() => document.querySelector("[data-closing-count]")?.textContent === "1");
  assert.equal(await page.locator("[data-closing-count]").innerText(), "1");
  await page.locator(".unified-matrix tbody button").first().click();
  await page.locator(".unified-detail").waitFor();
  assert.match(
    await page.locator(".unified-detail").innerText(),
    /Whole-unit sources: AWS · Azure · GCP/,
  );
  assert.equal(await page.locator(".unified-detail tbody tr").count(), 2);
  assert.equal(
    await page
      .locator(".unified-detail")
      .evaluate((e) => e === document.activeElement),
    true,
  );
  await page.getByRole("button", { name: "Close detail", exact: true }).click();
  assert.equal(
    await page
      .locator(".unified-matrix tbody button")
      .first()
      .evaluate((e) => e === document.activeElement),
    true,
  );
  await page.locator(".unified-matrix tbody button").first().click();
  await page.selectOption("#overview-currency", "KRW");
  await page.waitForFunction(() => !document.querySelector(".unified-detail"));
  assert.equal(await page.locator(".unified-detail").count(), 0);
  await open("&ov-sources=GCP&ov-organizations=B");
  assert.match(await page.locator('[data-total="USD"]').innerText(), /—/);
  assert.equal(
    await page
      .locator(".unified-month")
      .last()
      .locator(".unified-bar")
      .evaluate((e) => e.style.height),
    "0%",
  );
  assert.match(
    await page.locator(".unified-month").last().innerText(),
    /Unknown/,
  );
  await open("&ov-sources=On-prem&ov-organizations=C&ov-currency=KRW");
  assert.match(await page.locator('[data-total="KRW"]').innerText(), /KRW 0/);
  assert.match(
    await page.locator(".unified-matrix tbody").innerText(),
    /Not applicable/,
  );
  await open();
  assert.equal(await page.locator("#language").count(), 1);
  await page.selectOption("#overview-workspace", "sample-b");
  await page.waitForFunction(() => document.querySelector('[data-total="USD"]')?.textContent.includes("35,680.00"));
  assert.match(
    await page.locator('[data-total="USD"]').innerText(),
    /35,680.00/,
  );
  await open();
  await page.locator(".unified-exceptions button").first().click();
  await page.locator(".unified-detail").waitFor();
  assert.equal(await page.locator(".unified-detail tbody tr").count(), 1);
  assert.match(await page.locator(".unified-detail tbody").innerText(), /Example organization B/);
  assert(await page.locator(".unified-detail").evaluate(e => e.getBoundingClientRect().top < innerHeight && e.getBoundingClientRect().bottom > 0));
  const neutral = await page.locator(".unified-panel").evaluateAll(es => es.every(e => getComputedStyle(e).borderTopColor === "rgb(220, 225, 229)"));
  assert(neutral);
  await open("&mode=unconnected");
  await page
    .getByText("Integrated overview not connected", { exact: true })
    .waitFor();
  assert.equal(
    await page
      .locator("[data-total],.unified-month,.unified-matrix tbody tr")
      .count(),
    0,
  );
  await page.goto(url + "#/home?lang=en");
  await page.locator("[data-total]").first().waitFor();
  await page.getByText("Review tools", { exact: true }).click();
  await page
    .getByRole("link", { name: "Screen directory", exact: true })
    .click();
  await page.locator("#design-directory-results").waitFor();
  assert.equal(await page.locator(".unified-overview").count(), 0);
  assert.deepEqual(errors, []);
  console.log(
    "PASS: integrated overview, 6 locale/viewport renders, exact separate-currency totals, partial/all-missing/zero, multi-selection, unique closing units, monthly history, scoped detail and focus/reset, workspace change, unconnected and directory separation",
  );
} finally {
  await browser.close();
}
