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
      assert.equal(await page.locator("[data-amount-meaning]").count(), 2);
      for (const meaning of ["sales", "allocation"]) {
        const card = page.locator(`[data-amount-meaning="${meaning}"]`);
        assert.equal(await card.locator("[data-total]").count(), 0);
        assert.equal(await card.locator(".overview-unknown").count(), 1);
      }
      assert.deepEqual(
        await page.locator("[data-attention] strong").allTextContents(),
        ["1", "4", "2"],
      );
      const layout = await page.evaluate(() => {
        const body = document.querySelector(".unified-overview");
        const work = document.querySelector(".overview-workboard");
        const amounts = document.querySelector(".overview-amounts");
        const chart = document.querySelector(".unified-analysis");
        return {
          gap: getComputedStyle(body).gap,
          heading: getComputedStyle(work.querySelector("h2")).fontSize,
          metadata: getComputedStyle(amounts.querySelector("small")).fontSize,
          workBeforeChart:
            work.getBoundingClientRect().top <
            chart.getBoundingClientRect().top,
          workBeforeAmounts:
            work.getBoundingClientRect().top <
            amounts.getBoundingClientRect().top,
          workBeforeAmountsDOM: Boolean(
            work.compareDocumentPosition(amounts) &
            Node.DOCUMENT_POSITION_FOLLOWING,
          ),
        };
      });
      assert.equal(layout.gap, "32px");
      assert.equal(layout.heading, "20px");
      assert.equal(layout.metadata, "14px");
      assert(layout.workBeforeChart);
      assert.match(
        await page.locator("[data-trend-status]").innerText(),
        lang === "ko" ? /부분/ : lang === "ar" ? /جزئي/ : /Partial/,
      );
      const tableType = await page
        .locator(
          ".overview-org-breakdown tbody td,.unified-matrix tbody td,.unified-matrix tbody button",
        )
        .evaluateAll((es) =>
          es.every((e) => getComputedStyle(e).fontSize === "16px"),
        );
      assert(tableType);
      if (width === 390) {
        assert(
          await page
            .locator(".unified-breakdown td bdi,.overview-org-breakdown td bdi")
            .evaluateAll((es) =>
              es.every((e) => {
                const r = e.getBoundingClientRect();
                return r.left >= 0 && r.right <= innerWidth;
              }),
            ),
        );
      }
      assert.equal(layout.workBeforeAmounts, width === 390);
      assert.equal(layout.workBeforeAmountsDOM, width === 390);
      await page.evaluate(() => document.fonts.ready);
      await page.screenshot({
        path: `qa-output/overview/integrated-${lang}-${width}.png`,
        fullPage: true,
      });
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await open();
  for (const group of ["blocked", "approval", "erp"]) {
    const trigger = page.locator(`[data-attention="${group}"] button`);
    await trigger.click();
    await page.locator(".unified-detail").waitFor();
    assert(
      await page
        .locator(".unified-detail")
        .evaluate((e) => e === document.activeElement),
    );
    assert.match(
      await page.locator(".unified-detail").innerText(),
      group === "approval"
        ? /Approval/
        : group === "erp"
          ? /Accounting/
          : /Collection/,
    );
    await page.locator(".unified-detail header button").click();
    assert(await trigger.evaluate((e) => e === document.activeElement));
  }
  await open("&period=2026-09");
  assert.deepEqual(
    await page.locator("[data-attention] strong").allTextContents(),
    ["0", "0", "0"],
  );
  await page.setViewportSize({ width: 1440, height: 1000 });
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + "#overview?lang=" + lang + "&ov-scale=enterprise");
      await page.waitForFunction(
        () =>
          document.querySelector("[data-closing-count]")?.textContent ===
          "4800",
      );
      assert.equal(await page.locator(".unified-matrix tbody tr").count(), 20);
      assert.equal(
        await page.locator(".overview-org-breakdown tbody tr").count(),
        12,
      );
      assert.equal(await page.locator(".unified-exceptions li").count(), 5);
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      await page.screenshot({
        path: `qa-output/overview/enterprise-${lang}-${width}.png`,
        fullPage: true,
      });
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("&ov-scale=enterprise");
  await page.waitForFunction(
    () =>
      document.querySelector("[data-closing-count]")?.textContent === "4800",
  );
  const initialTotal = await page.locator('[data-total="USD"]').innerText();
  const list = page.locator(".unified-lifecycle");
  await list.getByLabel("Page", { exact: true }).fill("0");
  await list.getByRole("button", { name: "Go", exact: true }).click();
  await list.getByRole("alert").waitFor();
  assert.match(
    await list.locator("tbody tr").first().innerText(),
    /C01.B01.D01/,
  );
  await list.getByLabel("Page", { exact: true }).fill("٢٤٠");
  await list.getByLabel("Page", { exact: true }).press("Enter");
  assert.match(
    await list.locator("tbody tr").first().innerText(),
    /C12.B10.D11/,
  );
  assert(
    await list
      .getByLabel("Page", { exact: true })
      .evaluate((e) => e === document.activeElement),
  );
  await list.getByLabel("Page", { exact: true }).fill("999");
  await list.getByRole("button", { name: "Go", exact: true }).click();
  assert.equal(await list.getByRole("alert").count(), 1);
  await page.selectOption("#closing-stage", "accounting");
  await page.selectOption("#closing-status", "unknown");
  assert.equal(
    await page.locator("[data-list-count]").innerText(),
    "2399 / 4800",
  );
  assert.equal(
    await list.getByLabel("Page", { exact: true }).inputValue(),
    "1",
  );
  assert.equal(
    await page.locator('[data-total="USD"]').innerText(),
    initialTotal,
  );
  await list.locator("tbody button").first().click();
  await page.locator(".unified-detail").waitFor();
  await page.locator("#closing-search").fill("C12.B10.D20");
  assert.equal(await page.locator("[data-list-count]").innerText(), "1 / 4800");
  assert.equal(await page.locator(".unified-detail").count(), 0);
  await page.locator("#closing-search").fill("no-such-code");
  await page
    .getByText("No units match these filters", { exact: true })
    .waitFor();
  assert.equal(await list.locator("tbody tr").count(), 0);
  assert.equal(
    await page.locator('[data-total="USD"]').innerText(),
    initialTotal,
  );
  await list
    .getByRole("button", { name: "Reset list filters", exact: true })
    .click();
  assert.equal(
    await page.locator("[data-list-count]").innerText(),
    "4800 / 4800",
  );
  await page.selectOption("#closing-status", "blocked");
  assert.equal(await page.locator("[data-list-count]").innerText(), "1 / 4800");
  await page.selectOption("#overview-period", "2026-09");
  await page.waitForFunction(
    () => document.querySelector("#closing-status")?.value === "",
  );
  assert.equal(
    await page.locator("[data-list-count]").innerText(),
    "4800 / 4800",
  );
  await page.selectOption("#overview-period", "2026-10");
  await page.waitForFunction(
    () => document.querySelector("#overview-period")?.value === "2026-10",
  );
  await page
    .locator(".unified-matrix")
    .locator("..")
    .getByRole("button", { name: "Next", exact: true })
    .click();
  assert.match(
    await page.locator(".unified-matrix tbody tr").first().innerText(),
    /C01.B01.D11/,
  );
  assert.equal(
    await page.locator('[data-total="USD"]').innerText(),
    initialTotal,
  );
  await page.locator(".overview-org-picker summary").click();
  await page.locator("#overview-org-search").fill("C12.B10.D20");
  assert.equal(await page.locator(".overview-org-row").count(), 1);
  assert.equal(
    await page.locator('[data-total="USD"]').innerText(),
    initialTotal,
  );
  await page.locator(".overview-org-row input").click();
  await page.waitForFunction(
    () =>
      document.querySelector("[data-closing-count]")?.textContent === "4798",
  );
  assert((await page.evaluate(() => location.hash.length)) < 750);
  assert.match(
    await page.locator(".unified-matrix tbody tr").first().innerText(),
    /C01.B01.D01/,
  );
  await page
    .getByRole("button", { name: "Clear selection", exact: true })
    .click();
  await page
    .getByText("No data in the selected scope", { exact: true })
    .waitFor();
  await page.locator("#overview-org-search").fill("");
  await page
    .locator(".overview-org-row")
    .filter({ has: page.getByText("C01", { exact: true }) })
    .locator("input")
    .click();
  await page.waitForFunction(
    () => document.querySelector("[data-closing-count]")?.textContent === "400",
  );
  await page.getByRole("button", { name: "Browse C01", exact: true }).click();
  await page
    .getByRole("button", { name: "Browse C01.B01", exact: true })
    .click();
  assert.equal(await page.locator(".overview-org-row").count(), 20);
  await page.locator(".overview-org-row input").first().click();
  await page.waitForFunction(
    () => document.querySelector("[data-closing-count]")?.textContent === "398",
  );
  await page.locator(".overview-org-picker summary").click();
  await page.locator(".overview-org-breakdown tbody button").first().click();
  await page.locator(".unified-detail").waitFor();
  assert.equal(await page.locator(".unified-detail tbody tr").count(), 20);
  await page.getByRole("button", { name: "Close detail", exact: true }).click();
  await page.selectOption("#overview-period", "2026-09");
  await page.waitForFunction(
    () => document.querySelector("#overview-period")?.value === "2026-09",
  );
  assert.equal(await page.locator(".unified-detail").count(), 0);
  await page.selectOption("#overview-period", "2026-10");
  await page.waitForFunction(
    () => document.querySelector("#overview-period")?.value === "2026-10",
  );
  assert.match(
    await page.locator(".unified-matrix tbody tr").first().innerText(),
    /C01.B01.D02/,
  );
  await page.locator(".overview-example-config summary").click();
  await page.selectOption("#overview-profile", "small");
  await page.waitForFunction(
    () => document.querySelector("[data-closing-count]")?.textContent === "6",
  );
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
  await page.waitForFunction(
    () => document.querySelector("#overview-period")?.value === "2026-05",
  );
  assert.equal(await page.locator("#overview-period").inputValue(), "2026-05");
  assert.match(
    await page.locator('[data-total="USD"]').innerText(),
    /16,470.00/,
  );
  await page.goBack();
  await page.waitForFunction(
    () => document.querySelector("#overview-period")?.value === "2026-10",
  );
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
  await page.waitForFunction(
    () => document.querySelector("[data-closing-count]")?.textContent === "3",
  );
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
  await page.waitForFunction(
    () => document.querySelector("[data-closing-count]")?.textContent === "3",
  );
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
  await page.waitForFunction(
    () => document.querySelector("[data-closing-count]")?.textContent === "1",
  );
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
  assert.match(
    await page.locator('[data-total="USD"]').innerText(),
    /USD.*Unknown/s,
  );
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
  await page.locator(".overview-example-config summary").click();
  await page.selectOption("#overview-workspace", "sample-b");
  await page.waitForFunction(() =>
    document
      .querySelector('[data-total="USD"]')
      ?.textContent.includes("35,680.00"),
  );
  assert.match(
    await page.locator('[data-total="USD"]').innerText(),
    /35,680.00/,
  );
  await open();
  await page.locator(".unified-exceptions button").first().click();
  await page.locator(".unified-detail").waitFor();
  assert.equal(await page.locator(".unified-detail tbody tr").count(), 1);
  assert.match(
    await page.locator(".unified-detail tbody").innerText(),
    /Example organization B/,
  );
  assert(
    await page
      .locator(".unified-detail")
      .evaluate(
        (e) =>
          e.getBoundingClientRect().top < innerHeight &&
          e.getBoundingClientRect().bottom > 0,
      ),
  );
  const neutral = await page
    .locator(".unified-panel")
    .evaluateAll((es) =>
      es.every(
        (e) =>
          getComputedStyle(e).borderTopWidth === "0px" ||
          getComputedStyle(e).borderTopColor === "rgb(220, 225, 229)",
      ),
    );
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
  await page.goto(url + "#/design/screens?lang=en&review=1");
  await page.locator(".planning-directory").waitFor();
  assert.equal(await page.locator(".unified-overview").count(), 0);
  assert.deepEqual(errors, []);
  console.log(
    "PASS: integrated overview, 12 small/enterprise locale/viewport renders, 2400 departments, bounded pages, hierarchy selection, compact URL, scope reset, exact separate-currency totals, partial/all-missing/zero, multi-selection, unique closing units, monthly history, scoped detail and focus/reset, workspace change, unconnected and directory separation",
  );
} finally {
  await browser.close();
}
