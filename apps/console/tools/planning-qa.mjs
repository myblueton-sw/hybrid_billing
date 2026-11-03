import { createRequire } from "node:module";
import { mkdir, readFile, writeFile } from "node:fs/promises";
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
const page = await browser.newPage();
const errors = [];
page.on("pageerror", (e) => errors.push(e.message));
const registry = JSON.parse(
  await readFile("../../docs/features/HB-72/screen-registry.json", "utf8"),
);
const url = args.url || "http://127.0.0.1:8792/";
let renders = 0,
  tabs = 0;
await mkdir("qa-output/planning", { recursive: true });
const audienceControl = () =>
  page
    .locator("select")
    .filter({ has: page.locator('option[value="customer"]') });
const setAudience = async (a) => {
  const select = audienceControl();
  if ((await select.inputValue()) !== a) {
    await page
      .locator(".planning-review-tools")
      .evaluate((e) => (e.open = true));
    await select.selectOption(a);
  }
};
try {
  await page.goto(url + "#/design/screens?lang=ko");
  await page.locator(".planning-directory").waitFor();
  assert.equal(await page.locator(".planning-directory>button").count(), 49);
  if (!args.quick)
    for (const lang of ["ko", "en", "ar"])
      for (const width of [1440, 390]) {
        await page.setViewportSize({ width, height: 1000 });
        for (const s of registry.screens) {
          await setAudience(
            s.audience === "prelogin"
              ? "unauthenticated"
              : s.audience === "portal"
                ? "customer"
                : "admin",
          );
          await page.goto(url + "#" + s.path + "?lang=" + lang);
          await page.locator("#content").waitFor();
          await page.waitForTimeout(30);
          assert.ok(
            (await page.locator("#content").innerText()).trim().length > 20,
            s.id + " blank",
          );
          assert.ok(
            !/React implementation pending|React 구현 대기/.test(
              await page.locator("#content").innerText(),
            ),
            s.id + " placeholder",
          );
          assert(
            await page.evaluate(
              () => document.documentElement.scrollWidth <= innerWidth + 1,
            ),
            s.id + " " + lang + " " + width + " page overflow",
          );
          const reviewText =
            lang === "ko"
              ? "검수 확인"
              : lang === "en"
                ? "Confirm screen review"
                : "تأكيد مراجعة الشاشة";
          assert.ok(
            (await page.getByText(reviewText, { exact: true }).count()) > 0,
            s.id + " review control",
          );
          if (
            [
              "U01",
              "U02",
              "O01",
              "S02",
              "S06",
              "B04",
              "B09",
              "I01",
              "I04",
              "M03",
              "P03",
              "A01",
              "C01",
              "C02",
            ].includes(s.id)
          )
            await page.screenshot({
              path: `qa-output/planning/${s.id}-${lang}-${width}.png`,
              fullPage: true,
            });
          if (lang === "en" && width === 1440 && s.audience !== "prelogin") {
            const count = await page.locator(".planning-tabs button").count();
            for (let i = 1; i < count; i++) {
              await page.locator(".planning-tabs button").nth(i).click();
              await page.waitForTimeout(10);
              assert.ok(
                (await page.locator("#content").innerText()).trim().length > 20,
                s.id + " tab blank",
              );
              tabs++;
            }
          }
          renders++;
        }
      }
  await setAudience("customer");
  for (const path of [
    "/users",
    "/rates",
    "/evidence",
    "/home",
    "index-approval",
    "supplier-aws",
  ]) {
    await page.goto(url + "#" + path + "?lang=en");
    await page.waitForTimeout(30);
    assert.equal(await page.locator(".planning-sidebar").count(), 0);
    assert.ok(
      (await page.locator("h1").first().innerText()).includes(
        "My settlement home",
      ),
    );
  }
  await page.goto(url + "#/tasks?lang=en");
  await page.locator("#content").waitFor();
  assert.doesNotMatch(
    await page.locator("#content").innerText(),
    /Job \/ import|secret-ref/i,
  );
  await setAudience("admin");
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto(url + "#/rates?lang=en");
  await page.getByRole("button", { name: "Review draft", exact: true }).click();
  await page.locator(".planning-form input").first().fill("DEMO dirty");
  page.once("dialog", (d) => d.dismiss());
  await page.locator('.planning-sidebar a[href*="/users"]').click();
  assert.ok(page.url().includes("/rates"));
  page.once("dialog", (d) => d.accept());
  await page.locator('.planning-sidebar a[href*="/users"]').click();
  assert.ok(page.url().includes("/users"));
  await page.goto(url + "#/reconciliation?lang=en");
  await page.locator(".planning-link").first().click();
  await page.locator(".planning-detail[open]").waitFor();
  assert(
    await page
      .locator(".planning-detail")
      .evaluate((e) => e.contains(document.activeElement)),
  );
  await page.keyboard.press("Escape");
  assert.equal(await page.locator(".planning-detail").count(), 0);
  await page.goto(url + "#/users?lang=en&mode=unconnected");
  assert.equal(await page.locator(".planning-table tbody tr").count(), 0);

  await page.goto(
    url + "#/invoices?lang=en&view=body&period=2000-01&le=DEMO-ORG-02",
  );
  assert.equal(await page.locator(".planning-document").count(), 0);
  assert.doesNotMatch(await page.locator("#content").innerText(), /USD 110/);
  await page.goto(
    url + "#/reports?lang=en&view=summary&ccy=EUR&le=DEMO-ORG-02",
  );
  assert.equal(await page.locator(".planning-chart").count(), 0);
  await page.goto(url + "#/evidence?lang=en&view=documents&le=DEMO-ORG-02");
  assert.doesNotMatch(
    await page.locator("#content").innerText(),
    /DEMO-STATEMENT|DEMO-HASH/,
  );
  await page.goto(url + "#/access?lang=en&view=effects&le=DEMO-ORG-02");
  assert.equal(await page.locator(".planning-table tbody tr").count(), 0);
  await page.goto(url + "#/invoices/OTHER-ID/body?lang=en");
  assert.match(
    await page.locator("#content").innerText(),
    /Object unavailable/,
  );
  await page.goto(url + "#/rates?lang=en");
  await page.locator("#confirm-review").click();
  assert.match(
    await page.locator("dialog[open]").innerText(),
    /B04:field:search/,
  );
  assert.match(await page.locator("dialog[open]").innerText(), /B04:data:rate/);
  await page.keyboard.press("Escape");
  await page.locator(".planning-link").first().click();
  await page.goto(url + "#/rates?lang=en&ccy=EUR");
  assert.equal(await page.locator(".planning-detail").count(), 0);
  await page.goto(url + "#/users?lang=en");
  await page.goto(url + "#/rates?lang=en");
  await page.getByRole("button", { name: "Review draft", exact: true }).click();
  await page.locator(".planning-form input").first().fill("unsaved");
  page.once("dialog", (d) => d.dismiss());
  await page
    .getByRole("button", { name: "Review draft", exact: true })
    .first()
    .click();
  assert.equal(
    await page.locator(".planning-form input").first().inputValue(),
    "unsaved",
  );
  page.once("dialog", (d) => d.dismiss());
  await page.evaluate(() => history.back());
  await page.waitForTimeout(100);
  assert.ok(page.url().includes("/rates"));
  assert.equal(
    await page.locator(".planning-form input").first().inputValue(),
    "unsaved",
  );
  page.once("dialog", (d) => d.accept());
  await page.locator('.planning-sidebar a[href*="/users"]').click();
  await page.goto(url + "#/approvals?lang=en");
  await page.locator(".planning-inbox button").nth(1).click();
  assert.match(
    await page.locator(".planning-inbox-detail").innerText(),
    /DEMO-B09-2/,
  );
  assert.equal(
    await page.locator(".planning-inbox-detail button:disabled").count(),
    3,
  );
  assert.equal(
    await page
      .locator(".planning-inbox-detail dt")
      .filter({ hasText: /^Identifier$/ })
      .locator("..")
      .locator("dd")
      .innerText(),
    "DEMO-B09-2",
  );
  assert.equal(
    await page.locator(".planning-command-boundary").count(),
    1,
  );
  assert.equal(
    await page.getByRole("button", { name: "Approve", exact: true }).count(),
    1,
  );
  await page.goto(url + "#/organizations?lang=en");
  await page.locator("[role=treeitem]").first().focus();
  await page.keyboard.press("ArrowDown");
  await page.keyboard.press("Enter");
  assert.equal(await page.locator(".planning-table tbody tr").count(), 1);
  await page.setViewportSize({ width: 390, height: 1000 });
  await page.goto(url + "#/access?lang=ar");
  await page.locator(".planning-scope:not([open])").waitFor();
  assert.equal(await page.locator(".planning-scope[open]").count(), 0);
  assert((await page.locator(".planning-header").boundingBox()).height < 90);
  await page.locator(".planning-menu-button").click();
  await page.locator(".planning-menu-dialog[open]").waitFor();
  assert.equal(await page.locator("#language").count(), 1);
  await page.keyboard.press("Escape");
  assert.deepEqual(errors, []);
  await writeFile(
    "qa-output/planning/result.json",
    JSON.stringify({ renders, tabs, errors }, null, 2),
  );
  console.log(JSON.stringify({ pass: true, renders, tabs }));
} finally {
  await browser.close();
}
