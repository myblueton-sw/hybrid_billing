import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
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
const faults = [];
page.on("pageerror", (e) => faults.push(e.message));
const url = args.url || "http://127.0.0.1:8805/",
  out = args.output || "qa-output/infrastructure-cost";
await mkdir(out, { recursive: true });
const loaded = () => page.locator('[data-infra-state="loaded"]').waitFor();
const open = async (query = "") => {
  await page.goto(
    url +
      "#/cost-pools?lang=en&focus-provider=VMware&focus-period=2024-09" +
      query,
  );
  await loaded();
};
const pick = () => page.locator("[data-source-row]");
let passed = false;
try {
  await open();
  assert.equal(await pick().count(), 3);
  assert.ok(
    (await page.locator("h1").innerText()).includes(
      "Infrastructure cost composition",
    ),
  );
  await page.locator("#infra-save").click();
  assert.ok(
    (
      await page.locator(".infrastructure-cost [role=alert]").innerText()
    ).includes("composition name"),
  );
  await page.locator("#infra-draft-name").fill("Synthetic QA draft");
  await pick().first().check();
  await pick().nth(1).check();
  await page.getByRole("button", { name: "Cost items", exact: true }).click();
  await page.locator('[data-infra-view="costs"]').waitFor();
  assert.equal(
    await page.locator("#infra-draft-name").inputValue(),
    "Synthetic QA draft",
  );
  await page
    .locator('select[aria-label^="Category"]')
    .first()
    .selectOption("asset");
  const targets = page.locator('select[aria-label^="Included in"]');
  const firstTarget = await targets
    .nth(1)
    .locator("option")
    .nth(1)
    .getAttribute("value");
  await targets.nth(1).selectOption(firstTarget);
  await page
    .locator('input[aria-label^="Evidence reference"]')
    .nth(1)
    .fill("Synthetic QA whole-row declaration");
  assert.ok(
    (await page.locator("[data-total=review]").innerText()).includes(
      "USD 0.75",
    ),
  );
  await targets
    .first()
    .selectOption(
      await targets.first().locator("option").nth(1).getAttribute("value"),
    );
  await page
    .locator('input[aria-label^="Evidence reference"]')
    .first()
    .fill("Synthetic cycle check");
  assert.ok(
    (
      await page.locator(".infrastructure-cost [role=alert]").innerText()
    ).includes("circular"),
  );
  await targets.first().selectOption("");
  assert.equal(
    await page.locator("[data-inclusion-target]").innerText(),
    firstTarget,
  );
  await page.screenshot({ path: `${out}/en-cost-items.png`, fullPage: true });
  await page.locator("#infra-save").click();
  await page
    .getByRole("status")
    .filter({ hasText: "saved in this browser only" })
    .waitFor();
  const stored = await page.evaluate(() =>
    Object.entries(localStorage).filter(([k]) =>
      k.startsWith("hb-b01-review-v1:"),
    ),
  );
  assert.equal(stored.length, 1);
  assert.ok(!stored[0][1].includes("billed_cost"));
  await page.reload();
  await loaded();
  assert.equal(
    await page.locator("#infra-draft-name").inputValue(),
    "Synthetic QA draft",
  );
  await page.locator("#infra-draft-name").fill("Dirty draft");
  await page.getByRole("button", { name: "Cost items", exact: true }).click();
  page.once("dialog", (d) => d.dismiss());
  await page
    .getByRole("link", { name: "Inspect collected source", exact: true })
    .click();
  assert.ok(page.url().includes("/cost-pools?"));
  page.once("dialog", (d) => d.dismiss());
  await page.locator("#infra-basis").selectOption("effective_cost");
  assert.equal(await page.locator("#infra-basis").inputValue(), "billed_cost");
  page.once("dialog", (d) => d.accept());
  await page.locator("#infra-basis").selectOption("effective_cost");
  await loaded();
  assert.equal(await page.locator("#infra-draft-name").inputValue(), "");
  await page
    .getByRole("button", { name: "Composition summary", exact: true })
    .click();
  await page.locator("[data-infra-view=summary]").waitFor();
  await page.locator("#infra-draft-name").fill("Storage failure");
  await pick().first().check();
  await page.evaluate(() => {
    Storage.prototype.setItem = function () {
      throw Error("blocked");
    };
  });
  await page.locator("#infra-save").click();
  await page
    .getByRole("alert")
    .filter({ hasText: "Could not save in this browser" })
    .waitFor();
  assert.equal(
    await page.locator("#infra-draft-name").inputValue(),
    "Storage failure",
  );
  page.once("dialog", (d) => d.accept());
  await page.reload();
  await loaded();
  await page.route("**/api/focus-overview?**", (r) =>
    r.fulfill({
      status: 403,
      contentType: "application/json",
      body: JSON.stringify({ error: { code: "forbidden" } }),
    }),
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await page.locator('[data-infra-state="error"]').waitFor();
  assert.equal(await pick().count(), 0);
  assert.equal(await page.locator("#infra-draft-name").count(), 0);
  await page.unroute("**/api/focus-overview?**");
  await open();
  for (const name of [
    "Included items by product",
    "Approval history",
    "Composition summary",
  ]) {
    await page.getByRole("button", { name, exact: true }).click();
  }
  await page.locator("[data-infra-view=summary]").waitFor();
  const replay = await page.evaluate(async () => {
    const read = async (basis) => {
      const r = await fetch(
        `/api/focus-overview?provider=VMware&period=2024-09&basis=${basis}`,
        { headers: { "X-HB-Local-Client": "1" }, cache: "no-store" },
      );
      if (!r.ok) throw Error("replay source unavailable");
      return r.json();
    };
    return {
      billed: await read("billed_cost"),
      effective: await read("effective_cost"),
    };
  });
  await page.route("**/api/focus-overview?**", async (route) => {
    const effective =
      new URL(route.request().url()).searchParams.get("basis") ===
      "effective_cost";
    if (effective) await new Promise((resolve) => setTimeout(resolve, 700));
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(effective ? replay.effective : replay.billed),
    });
  });
  await page.evaluate(() => {
    location.hash =
      "#/cost-pools?lang=en&focus-provider=VMware&focus-period=2024-09&focus-basis=effective_cost";
  });
  await page.waitForTimeout(100);
  await page.evaluate(() => {
    location.hash =
      "#/cost-pools?lang=en&focus-provider=VMware&focus-period=2024-09&focus-basis=billed_cost";
  });
  await loaded();
  await page.waitForTimeout(900);
  assert.equal(await page.locator("#infra-basis").inputValue(), "billed_cost");
  assert.equal(
    await page.locator("#infra-draft-name").inputValue(),
    "Synthetic QA draft",
  );
  await page.unroute("**/api/focus-overview?**");
  const variants = [];
  for (const locale of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(
        url +
          `#/cost-pools?lang=${locale}&focus-provider=VMware&focus-period=2024-09`,
      );
      await loaded();
      await page.evaluate(() => document.fonts.ready);
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth,
        ),
        false,
      );
      assert.equal(
        await page
          .locator("button")
          .filter({
            hasText:
              locale === "ko"
                ? "검수 확인"
                : locale === "en"
                  ? "Confirm screen review"
                  : "تأكيد مراجعة الشاشة",
          })
          .count(),
        1,
      );
      if (width === 390)
        assert.ok(
          await page.locator(".infra-table tbody tr").evaluateAll((rows) =>
            rows.every((row) =>
              [...row.children].every((cell) => {
                const box = cell.getBoundingClientRect();
                return [...cell.children].every(
                  (child) =>
                    child.getBoundingClientRect().bottom <= box.bottom + 1,
                );
              }),
            ),
          ),
        );
      await page.screenshot({
        path: `${out}/${locale}-${width}.png`,
        fullPage: true,
      });
      variants.push({ locale, width });
    }
  assert.deepEqual(faults, []);
  passed = true;
  await writeFile(
    `${out}/result.json`,
    JSON.stringify({ passed, variants, faults }, null, 2),
  );
} catch (e) {
  await page.screenshot({ path: `${out}/failure.png`, fullPage: true });
  throw e;
} finally {
  await browser.close();
}
console.log(JSON.stringify({ passed }));
