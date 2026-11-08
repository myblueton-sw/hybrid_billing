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
  out = args.output || "qa-output/sales-pricing";
await mkdir(out, { recursive: true });
const base = "focus-provider=VMware&focus-period=2024-09";
const loaded = () => page.locator("[data-pricing-state=loaded]").waitFor();
const selected = (id) =>
  page.waitForFunction(
    (id) =>
      document
        .querySelector("[data-pricing-selected-row]")
        ?.getAttribute("data-pricing-selected-row") === id,
    id,
  );
const open = async (query = "") => {
  await page.goto(url + "#/rates?lang=en&" + base + query);
  await loaded();
};
const root = () => page.locator("[data-pricing-state]");
const fill = async () => {
  for (const [id, value] of Object.entries({
    "pricing-name": "Synthetic pricing QA",
    "pricing-unit-price": "2",
    "pricing-denominator": "1",
    "pricing-discount": "10",
    "pricing-fee": "0.1",
  }))
    await page.locator("#" + id).fill(value);
};
const amount = (n) => {
  const sign = n < 0n ? "−" : "",
    v = n < 0n ? -n : n;
  return (
    sign +
    "USD " +
    v / 10n ** 18n +
    "." +
    (v % 10n ** 18n)
      .toString()
      .padStart(18, "0")
      .replace(/0+$/, "")
      .padEnd(2, "0")
  );
};
let passed = false;
try {
  await open();
  await page.locator("#pricing-search").fill("Synthetic");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await loaded();
  await page.locator("#pricing-search").fill(" Synthetic ");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await loaded();
  await open();
  const ids = await page
    .locator("button[data-pricing-row]")
    .evaluateAll((es) => es.map((e) => e.getAttribute("data-pricing-row")));
  assert.equal(ids.length, 3);
  await page.locator("button[data-pricing-row]").first().click();
  await selected(ids[0]);
  await fill();
  await page.locator("#pricing-preview").click();
  await page.locator("[data-pricing-candidate]").waitFor();
  const source = await page.evaluate(async () => {
    const r = await fetch(
      "/api/focus-overview?provider=VMware&period=2024-09",
      { headers: { "X-HB-Local-Client": "1" } },
    );
    if (!r.ok) throw Error("source read unavailable");
    return r.json();
  });
  const q = BigInt(source.rows.find((r) => r.id === ids[0]).pricing_quantity),
    expected = (q * 2n * 9n) / 10n + 10n ** 17n;
  assert.ok(
    (
      await page.locator("[data-pricing-total=total]").first().innerText()
    ).includes(amount(expected)),
  );
  await page.locator(".planning-tabs button").nth(4).click();
  await page.locator("#pricing-baseline").click();
  await page.locator(".planning-tabs button").nth(1).click();
  await page.locator("[data-pricing-view=formula]").waitFor();
  await page.screenshot({ path: `${out}/en-formula.png`, fullPage: true });
  await page.locator(".planning-tabs button").first().click();
  await page.locator("#pricing-unit-price").fill("3");
  await page.locator("[data-pricing-candidate]").waitFor({ state: "detached" });
  await page.locator("#pricing-preview").click();
  await page.locator(".planning-tabs button").nth(4).click();
  await page.locator("[data-pricing-view=impact]").waitFor();
  await page.screenshot({ path: `${out}/en-impact.png`, fullPage: true });
  await page.locator(".planning-tabs button").nth(3).click();
  await page.locator("[data-pricing-view=adjustments]").waitFor();
  await page.locator("#pricing-target").fill("10");
  await page.locator("#pricing-reason").fill("Synthetic arithmetic review");
  await page.locator("#pricing-reference").fill("Synthetic reference");
  await page.locator("#pricing-adjust").click();
  const candidate = (q * 3n * 9n) / 10n + 10n ** 17n;
  assert.ok(
    (
      await page.locator("[data-pricing-adjustment=delta]").innerText()
    ).includes(amount(10n ** 19n - candidate)),
  );
  await page.screenshot({ path: `${out}/en-adjustment.png`, fullPage: true });
  console.log("QA calculation/adjustment passed");
  await page.locator("#pricing-target").fill("11");
  await page
    .locator("[data-pricing-adjustment=delta]")
    .filter({ hasText: "Not calculated" })
    .waitFor();
  await page.locator(".planning-tabs button").first().click();
  await page.locator("#pricing-save").click();
  await root().getByRole("status").filter({ hasText: /saved/i }).waitFor();
  console.log("QA save passed");
  const stored = await page.evaluate(() =>
    Object.entries(localStorage).filter(([k]) =>
      k.startsWith("hb-b04-draft-v1:"),
    ),
  );
  assert.equal(stored.length, 1);
  const body = JSON.parse(stored[0][1]);
  assert.equal(body.draft.unitPrice, "3");
  assert.equal(Object.hasOwn(body, "pricing_quantity"), false);
  await page.reload();
  await selected(ids[0]);
  console.log("QA restore passed");
  assert.equal(
    await page.locator("#pricing-name").inputValue(),
    "Synthetic pricing QA",
  );
  await page.locator("[data-pricing-candidate]").waitFor({ state: "detached" });
  await page.locator("#pricing-unit-price").fill("4");
  await page.locator(".planning-tabs button").first().click();
  page.once("dialog", (d) => d.dismiss());
  await page.locator("button[data-pricing-row]").nth(1).click();
  await selected(ids[0]);
  assert.equal(await page.locator("#pricing-unit-price").inputValue(), "4");
  page.once("dialog", (d) => d.accept());
  await page.locator("button[data-pricing-row]").nth(1).click();
  await selected(ids[1]);
  assert.equal(await page.locator("#pricing-name").inputValue(), "");
  console.log("QA dirty row guard passed");
  await fill();
  await page.evaluate(() => {
    Storage.prototype.setItem = function () {
      throw Error("blocked");
    };
  });
  await page.locator("#pricing-save").click();
  await root()
    .getByRole("alert")
    .filter({ hasText: /sav|storage/i })
    .waitFor();
  assert.equal(
    await page.locator("#pricing-name").inputValue(),
    "Synthetic pricing QA",
  );
  page.once("dialog", (d) => d.accept());
  await page.reload();
  await selected(ids[1]);
  await fill();
  await page.locator("#pricing-preview").click();
  await page.locator(".planning-tabs button").nth(4).click();
  await page.locator("#pricing-baseline").click();
  page.once("dialog", (d) => d.accept());
  await page.route("**/api/focus-overview?**", (r) =>
    r.fulfill({
      status: 403,
      contentType: "application/json",
      body: JSON.stringify({ error: "permission_denied" }),
    }),
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await page.locator("[data-pricing-state=error]").waitFor();
  await page.locator("[data-pricing-candidate]").waitFor({ state: "detached" });
  await page.unroute("**/api/focus-overview?**");
  await page
    .getByRole("button", { name: "Reset row and retry", exact: true })
    .click();
  await loaded();
  console.log("QA source-denial retry passed");
  await page.evaluate((id) => {
    const original = crypto.subtle.digest.bind(crypto.subtle);
    crypto.subtle.digest = async (algo, bytes) => {
      const text = new TextDecoder().decode(bytes);
      if (text.includes('"row":"' + id + '"')) {
        window.__pricingDelayedHash = true;
        await new Promise((resolve) => setTimeout(resolve, 2200));
      }
      return original(algo, bytes);
    };
  }, ids[0]);
  await page.evaluate(
    ({ base, id }) => {
      location.hash = "#/rates?lang=en&" + base + "&pricing-row=" + id;
    },
    { base, id: ids[0] },
  );
  await page.waitForFunction(() => window.__pricingDelayedHash === true);
  await page.evaluate(
    ({ base, id }) => {
      location.hash = "#/rates?lang=en&" + base + "&pricing-row=" + id;
    },
    { base, id: ids[1] },
  );
  await selected(ids[1]);
  console.log("QA late hash row B selected");
  await page.waitForTimeout(2300);
  assert.equal(await page.locator("#pricing-name").inputValue(), "");
  assert.equal(await root().getAttribute("data-pricing-selected-row"), ids[1]);
  const variants = [];
  for (const locale of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(
        url + "#/rates?lang=" + locale + "&" + base + "&pricing-row=" + ids[0],
      );
      await selected(ids[0]);
      await page.evaluate(() => document.fonts.ready);
      await page.locator(".planning-tabs button").first().click();
      await page.locator("#pricing-unit-price").fill("3");
      await page.locator("#pricing-preview").click();
      await page.locator("[data-pricing-candidate]").waitFor();
      await page.locator(".planning-tabs button").nth(4).click();
      await page.locator("#pricing-baseline").click();
      await page.locator(".planning-tabs button").first().click();
      await page.locator("#pricing-unit-price").fill("4");
      await page.locator("#pricing-preview").click();
      for (const [i, view] of [
        "rates",
        "formula",
        "inheritance",
        "adjustments",
        "impact",
      ].entries()) {
        await page.locator(".planning-tabs button").nth(i).click();
        await page.locator("[data-pricing-view=" + view + "]").waitFor();
        if (view === "adjustments")
          await page.locator("#pricing-adjust").click();
        assert.equal(
          await page.evaluate(
            () => document.documentElement.scrollWidth > innerWidth,
          ),
          false,
        );
        const styleCheck = await root().evaluate((el) => ({
          headings: [...el.querySelectorAll("h2")].map(
            (e) => getComputedStyle(e).fontSize,
          ),
          controls: [...el.querySelectorAll("input,select,textarea")].map(
            (e) => getComputedStyle(e).fontSize,
          ),
          textarea: el.querySelector("textarea")
            ? getComputedStyle(el.querySelector("textarea")).fontFamily
            : null,
        }));
        assert.ok(styleCheck.headings.every((v) => v === "20px"));
        assert.ok(
          styleCheck.controls.every((v) => v === "16px"),
          JSON.stringify(styleCheck),
        );
        if (styleCheck.textarea) assert.match(styleCheck.textarea, /IBM Plex/);
        await page.screenshot({
          path: `${out}/${locale}-${width}-${view}.png`,
          fullPage: true,
        });
      }
      page.once("dialog", (d) => d.accept());
      variants.push({ locale, width, views: 5 });
      console.log(JSON.stringify({ locale, width, views: 5 }));
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
