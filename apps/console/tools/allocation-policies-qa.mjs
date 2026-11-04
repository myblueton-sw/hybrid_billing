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
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } }),
  faults = [];
page.on("pageerror", (e) => faults.push(e.message));
const url = args.url || "http://127.0.0.1:8805/",
  out = args.output || "qa-output/allocation-policies";
await mkdir(out, { recursive: true });
const scope =
  "focus-provider=VMware&focus-period=2024-09&allocation-currency=USD";
const loaded = () => page.locator("[data-allocation-state=loaded]").waitFor();
const open = async (query = "") => {
  await page.goto(url + "#/allocations?lang=en&" + scope + query);
  await loaded();
};
const weights = () => page.locator("input[data-allocation-target]");
const root = () => page.locator(".allocation-screen");
let passed = false;
try {
  await open();
  assert.equal(await weights().count(), 2);
  await page.locator(".allocation-source-note summary").click();
  await page.locator(".allocation-source-note summary").press("Escape");
  assert.equal(
    await page.locator(".allocation-source-note details").getAttribute("open"),
    null,
  );
  await page.locator("#allocation-preview").click();
  await page.waitForFunction(() =>
    document.activeElement?.classList.contains("allocation-error-summary"),
  );
  await page.locator(".allocation-error-summary button").first().click();
  await page.waitForFunction(
    () =>
      document.activeElement ===
      document.querySelector("input[data-allocation-target]"),
  );
  await page.locator("#allocation-name").fill("Synthetic allocation QA");
  await weights().first().fill("50");
  await weights().nth(1).fill("25");
  await page.locator("#allocation-preview").click();
  await page.locator("[data-allocation-candidate]").waitFor();
  assert.ok(
    (
      await page.locator("[data-allocation-total=assigned]").first().innerText()
    ).includes("1.875"),
  );
  assert.ok(
    (
      await page
        .locator("[data-allocation-total=unassigned]")
        .first()
        .innerText()
    ).includes("0.625"),
  );
  await page
    .getByRole("button", { name: "Allocation preview", exact: true })
    .click();
  await page.locator("[data-allocation-view=simulation]").waitFor();
  await page.locator("#allocation-baseline").click();
  await page.screenshot({ path: `${out}/en-preview.png`, fullPage: true });
  await page
    .getByRole("button", { name: "Allocation policy", exact: true })
    .click();
  await page.locator("[data-allocation-view=policies]").waitFor();
  await weights().nth(1).fill("50");
  assert.equal(await page.locator("[data-allocation-candidate]").count(), 0);
  await page.locator("#allocation-search").fill("no-visible-target");
  assert.equal(await weights().count(), 0);
  await page.locator("#allocation-preview").click();
  await page.locator("[data-allocation-candidate]").waitFor();
  assert.ok(
    (
      await page.locator("[data-allocation-total=assigned]").first().innerText()
    ).includes("2.50"),
  );
  await page
    .getByRole("button", { name: "Result comparison", exact: true })
    .click();
  await page.locator("[data-allocation-view=comparison]").waitFor();
  await page.screenshot({ path: `${out}/en-comparison.png`, fullPage: true });
  await page
    .getByRole("button", { name: "Allocation policy", exact: true })
    .click();
  await page.locator("#allocation-search").fill("");
  await page.locator("#allocation-save").click();
  await root().getByRole("status").filter({ hasText: /saved/i }).waitFor();
  const stored = await page.evaluate(() =>
    Object.entries(localStorage).filter(([k]) =>
      k.startsWith("hb-b03-draft-v1:"),
    ),
  );
  assert.equal(stored.length, 1);
  assert.ok(!stored[0][1].includes("billed_cost"));
  assert.ok(!stored[0][1].includes("coefficient"));
  await page.reload();
  await loaded();
  assert.equal(
    await page.locator("#allocation-name").inputValue(),
    "Synthetic allocation QA",
  );
  assert.equal(await page.locator("[data-allocation-candidate]").count(), 0);
  await weights().first().fill("60");
  await page
    .getByRole("button", { name: "Allocation policy", exact: true })
    .click();
  page.once("dialog", (d) => d.dismiss());
  await page.locator("#allocation-basis").selectOption("effective_cost");
  assert.equal(
    await page.locator("#allocation-basis").inputValue(),
    "billed_cost",
  );
  page.once("dialog", (d) => d.accept());
  await page.locator("#allocation-basis").selectOption("effective_cost");
  await loaded();
  assert.equal(await page.locator("#allocation-name").inputValue(), "");
  await page.locator("#allocation-name").fill("Storage failure");
  await weights().first().fill("50");
  await weights().nth(1).fill("50");
  await page.evaluate(() => {
    Storage.prototype.setItem = function () {
      throw Error("blocked");
    };
  });
  await page.locator("#allocation-save").click();
  await root().getByRole("alert").waitFor();
  assert.equal(
    await page.locator("#allocation-name").inputValue(),
    "Storage failure",
  );
  page.once("dialog", (d) => d.accept());
  await page.reload();
  await loaded();
  await page.route("**/api/focus-overview?**", (r) =>
    r.fulfill({
      status: 403,
      contentType: "application/json",
      body: JSON.stringify({ error: "permission_denied" }),
    }),
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await page.locator("[data-allocation-state=error]").waitFor();
  assert.equal(await weights().count(), 0);
  assert.equal(await page.locator("#allocation-name").count(), 0);
  await page.unroute("**/api/focus-overview?**");
  await page.goto(
    url + "#/allocations?lang=en&" + scope + "&focus-q=restricted",
  );
  await page.locator("[data-allocation-state=restricted]").waitFor();
  assert.equal(await weights().count(), 0);
  await page
    .getByRole("button", { name: "Open whole source scope", exact: true })
    .click();
  await loaded();
  assert.equal(
    new URLSearchParams(page.url().split("?")[1]).get("focus-q"),
    null,
  );
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
      "#/allocations?lang=en&focus-provider=VMware&focus-period=2024-09&focus-basis=effective_cost&allocation-currency=USD";
  });
  await page.waitForTimeout(100);
  await page.evaluate(() => {
    location.hash =
      "#/allocations?lang=en&focus-provider=VMware&focus-period=2024-09&focus-basis=billed_cost&allocation-currency=USD";
  });
  await loaded();
  await page.waitForTimeout(900);
  assert.equal(
    await page.locator("#allocation-basis").inputValue(),
    "billed_cost",
  );
  assert.equal(
    await page.locator("#allocation-name").inputValue(),
    "Synthetic allocation QA",
  );
  assert.equal(await page.locator("[data-allocation-candidate]").count(), 0);
  await page.unroute("**/api/focus-overview?**");
  const variants = [];
  for (const locale of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + "#/allocations?lang=" + locale + "&" + scope);
      await loaded();
      await page.evaluate(() => document.fonts.ready);
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth,
        ),
        false,
      );
      const label =
        locale === "ko"
          ? "검수 확인"
          : locale === "en"
            ? "Confirm screen review"
            : "تأكيد مراجعة الشاشة";
      assert.equal(
        await page.getByRole("button", { name: label, exact: true }).count(),
        1,
      );
      assert.ok(
        await page
          .locator(".allocation-screen h2")
          .evaluateAll((es) =>
            es.every((e) => getComputedStyle(e).fontSize === "20px"),
          ),
      );
      assert.equal(
        await page
          .locator(".allocation-method")
          .evaluate((e) => getComputedStyle(e).flexDirection),
        "row",
      );
      assert.equal(
        await page
          .locator(".allocation-method")
          .evaluate((e) => getComputedStyle(e).fontSize),
        "16px",
      );
      assert.ok(
        await page
          .locator("[data-label]")
          .evaluateAll((es) =>
            es.every(
              (e) => !e.getAttribute("data-label").includes("[object Object]"),
            ),
          ),
      );
      if (width === 390) {
        const action = await page.locator("#allocation-preview").boundingBox(),
          first = await weights().first().boundingBox();
        assert.ok(action.y < first.y);
      }
      await page.screenshot({
        path: `${out}/${locale}-${width}.png`,
        fullPage: true,
      });
      await page.locator(".planning-tabs button").nth(1).click();
      await page.locator("[data-allocation-view=pools]").waitFor();
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth,
        ),
        false,
      );
      await page.screenshot({
        path: `${out}/${locale}-${width}-basis.png`,
        fullPage: true,
      });
      await page.locator(".planning-tabs button").first().click();
      await weights().first().fill("50");
      await weights().nth(1).fill("25");
      await page.locator("#allocation-preview").click();
      await page.locator(".planning-tabs button").nth(2).click();
      await page.locator("[data-allocation-view=simulation]").waitFor();
      await page.locator("#allocation-baseline").click();
      await page.locator(".planning-tabs button").first().click();
      await weights().nth(1).fill("50");
      await page.locator("#allocation-preview").click();
      await page.locator(".planning-tabs button").nth(2).click();
      await page.locator("[data-allocation-view=simulation]").waitFor();
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth,
        ),
        false,
      );
      await page.screenshot({
        path: `${out}/${locale}-${width}-preview.png`,
        fullPage: true,
      });
      await page.locator(".planning-tabs button").nth(3).click();
      await page.locator("[data-allocation-view=comparison]").waitFor();
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth,
        ),
        false,
      );
      await page.screenshot({
        path: `${out}/${locale}-${width}-comparison.png`,
        fullPage: true,
      });
      page.once("dialog", (d) => d.accept());
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
