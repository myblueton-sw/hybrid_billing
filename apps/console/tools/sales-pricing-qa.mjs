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
  const acceptDialog = (dialog) => dialog.accept().catch(() => {});
  page.on("dialog", acceptDialog);
  const visit = async (query) => {
    await page.goto(url + "#/rates?" + query);
    await loaded();
  };
  const lines = async (locator) =>
    (await locator.innerText())
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean);
  const pickerRow = (index) =>
    page
      .locator(".pricing-picker tbody tr[data-pricing-picker-row]")
      .nth(index);
  const evidence = async (index) => {
    const toggle = pickerRow(index).locator("[data-pricing-evidence-toggle]"),
      id = await toggle.getAttribute("data-pricing-evidence-toggle");
    if ((await toggle.getAttribute("aria-expanded")) !== "true")
      await toggle.click();
    const panel = page.locator(`tr[data-evidence-for="${id}"]`);
    await panel.waitFor();
    return (name) => panel.locator(`[data-pricing-field="${name}"] dd`);
  };
  await page.setViewportSize({ width: 1440, height: 1000 });
  await visit("lang=en&focus-provider=VMware&focus-period=2024-09");
  assert.deepEqual(
    await page
      .locator(".pricing-picker tbody tr [data-pricing-account] bdi")
      .allInnerTexts(),
    ["synthetic-vmware", "synthetic-vmware", "synthetic-vmware"],
  );
  assert.ok((await pickerRow(0).innerText()).includes("synthetic-a"));
  assert.ok((await pickerRow(2).innerText()).includes("synthetic-b"));
  let field = await evidence(0);
  assert.deepEqual(await lines(field("ordinal")), ["1000"]);
  assert.deepEqual(await lines(field("evidence_class")), [
    "Synthetic test data",
  ]);
  assert.equal((await field("id").innerText()).trim(), ids[0]);
  assert.deepEqual(await lines(field("charge_start")), [
    "2024-09-01 00:00:00 UTC",
  ]);
  assert.deepEqual(await lines(field("charge_end")), [
    "2024-09-02 00:00:00 UTC",
  ]);
  assert.deepEqual(await lines(field("list_unit_price")), ["USD 0.75 / Hours"]);
  assert.equal(
    await page
      .locator(".pricing-table-scroll")
      .evaluate((region) => region.scrollWidth > region.clientWidth + 1),
    false,
  );
  await visit("lang=en&focus-provider=Oracle");
  field = await evidence(0);
  assert.deepEqual(await lines(field("billed_cost")), ["USD 0.24"]);
  assert.deepEqual(await lines(field("effective_cost")), [
    "USD 0.00",
    "Exact zero",
  ]);
  assert.deepEqual(await lines(field("contracted_unit_price")), ["Missing"]);
  assert.equal(await pickerRow(0).locator("td").nth(3).innerText(), "USD 0.24");
  await page.screenshot({
    path: `${out}/en-oracle-evidence.png`,
    fullPage: true,
  });
  await pickerRow(0).locator("button[data-pricing-row]").click();
  assert.deepEqual(await lines(page.locator(".pricing-period")), [
    "Start 2024-09-30 22:00:00 UTC",
    "End 2024-09-30 23:00:00 UTC",
  ]);
  await page.locator(".planning-tabs button").nth(4).click();
  const costEvidence = page
    .locator("details")
    .filter({ hasText: "Source cost evidence" });
  await costEvidence.locator("summary").click();
  assert.deepEqual(
    await lines(costEvidence.locator('[data-pricing-field="billed_cost"] dd')),
    ["USD 0.24"],
  );
  assert.deepEqual(
    (await costEvidence.locator("dt").allInnerTexts()).map((text) =>
      text.trim(),
    ),
    ["Billed cost", "Effective cost"],
  );
  assert.deepEqual(
    await lines(
      costEvidence.locator('[data-pricing-field="effective_cost"] dd'),
    ),
    ["USD 0.00", "Exact zero"],
  );
  await page.locator(".planning-tabs button").nth(0).click();
  for (const [locale, zero, missing] of [
    ["ko", "정확한 0", "누락"],
    ["ar", "صفر دقيق", "مفقود"],
  ]) {
    await visit(`lang=${locale}&focus-provider=Oracle`);
    field = await evidence(0);
    assert.equal((await lines(field("effective_cost")))[1], zero);
    assert.deepEqual(await lines(field("contracted_unit_price")), [missing]);
  }
  const awsSnapshot = await page.evaluate(async () => {
    const r = await fetch("/api/focus-overview?provider=AWS", {
      headers: { "X-HB-Local-Client": "1" },
      cache: "no-store",
    });
    if (!r.ok) throw Error("AWS source unavailable");
    return (await r.json()).snapshot;
  });
  await visit(
    `lang=en&focus-provider=AWS&focus-after=450&focus-snapshot=${awsSnapshot}`,
  );
  assert.equal(
    (await pickerRow(6).locator("td").nth(3).innerText()).trim(),
    "USD -2.6137",
  );
  await visit("lang=en&focus-provider=Alibaba+Cloud");
  assert.equal(
    (await pickerRow(1).locator("td").nth(3).innerText()).trim(),
    "CNY 8.80",
  );
  const degraded = structuredClone(source);
  degraded.rows[0].billed_cost = null;
  degraded.rows[0].sub_account = null;
  degraded.rows[0].service = null;
  degraded.rows[0].pricing_unit = null;
  degraded.rows[0].list_unit_price = null;
  degraded.rows[0].consumed_quantity = "0";
  degraded.rows[0].consumed_unit = null;
  degraded.rows[0].evidence_class = "unrecognized-class";
  degraded.rows[1].pricing_quantity = null;
  degraded.rows[1].pricing_unit = null;
  degraded.rows[2].pricing_unit = "  ";
  degraded.rows[2].charge_start = degraded.rows[2].charge_start.replace(
    "+00:00",
    "Z",
  );
  await page.route("**/api/focus-overview?**", (r) =>
    r.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(degraded),
    }),
  );
  for (const [locale, missing, unit, provenance, noQuantity, noUnit] of [
    [
      "en",
      "Missing",
      "Unit missing",
      "Unclassified",
      "Pricing quantity missing",
      "Pricing unit missing",
    ],
    [
      "ko",
      "누락",
      "단위 누락",
      "출처 미분류",
      "요금 산정 수량 누락",
      "요금 산정 단위 누락",
    ],
    [
      "ar",
      "مفقود",
      "الوحدة مفقودة",
      "غير مصنف",
      "كمية التسعير مفقودة",
      "وحدة التسعير مفقودة",
    ],
  ]) {
    await visit(
      `lang=${locale}&focus-provider=VMware&focus-period=2024-09&focus-basis=billed_cost`,
    );
    await page.locator(".pricing-toolbar button").last().click();
    await loaded();
    const cells = await pickerRow(0).locator("td").allInnerTexts();
    assert.equal(
      (await pickerRow(0).locator("[data-pricing-service]").innerText()).trim(),
      missing,
    );
    assert.ok(
      (await pickerRow(2).locator("td").nth(2).innerText()).includes(unit),
    );
    assert.ok(cells[2].includes(unit));
    assert.equal(cells[3].trim(), missing);
    assert.ok(!/USD|—|\d/.test(cells[3]));
    assert.ok(cells[4].includes(noUnit));
    assert.ok(
      (await pickerRow(1).locator("td").nth(4).innerText()).includes(
        noQuantity,
      ),
    );
    assert.deepEqual(await lines(pickerRow(1).locator("td").nth(2)), [
      missing,
      unit,
    ]);
    field = await evidence(0);
    assert.deepEqual(await lines(field("list_unit_price")), [missing]);
    assert.deepEqual(await lines(field("billed_cost")), [missing]);
    assert.deepEqual(await lines(field("sub_account")), [missing]);
    assert.deepEqual(await lines(field("evidence_class")), [provenance]);
    const consumed = await lines(field("consumed_quantity"));
    assert.equal(consumed[0], "0.00");
    assert.ok(
      consumed.includes(
        locale === "en"
          ? "Exact zero"
          : locale === "ko"
            ? "정확한 0"
            : "صفر دقيق",
      ),
    );
    await pickerRow(1).locator("[data-pricing-evidence-toggle]").click();
    assert.equal(await page.locator("tr.pricing-evidence-row").count(), 1);
    assert.equal(
      await pickerRow(0)
        .locator("[data-pricing-evidence-toggle]")
        .getAttribute("aria-expanded"),
      "false",
    );
    assert.equal(
      await pickerRow(1)
        .locator("[data-pricing-evidence-toggle]")
        .getAttribute("aria-expanded"),
      "true",
    );
    field = await evidence(0);
    assert.ok(consumed.includes(unit));
    const contracted = await lines(field("contracted_unit_price"));
    assert.ok(contracted.includes(unit));
    assert.ok(!contracted[0].includes("/"));
    const raw = await evidence(2);
    assert.ok(
      (await raw("charge_start").innerText()).includes(
        degraded.rows[2].charge_start,
      ),
    );
    if (locale === "en")
      assert.ok(
        (await raw("charge_start").innerText()).includes(
          "Shown as collected; not converted to UTC",
        ),
      );
    await pickerRow(0).locator("button[data-pricing-row]").click();
    assert.equal(
      (
        await page.locator("[data-pricing-selected-service]").innerText()
      ).trim(),
      missing,
    );
    assert.ok(
      (await page.locator(".pricing-identity").innerText()).includes(unit),
    );
    if (locale === "en")
      await page.screenshot({
        path: `${out}/en-degraded-evidence.png`,
        fullPage: true,
      });
  }
  await page.unroute("**/api/focus-overview?**");
  await visit("lang=en&focus-provider=VMware&focus-period=2024-09");
  for (const width of [1280, 1024, 768]) {
    await page.setViewportSize({ width, height: 1000 });
    await visit("lang=en&focus-provider=Oracle");
    await evidence(0);
    assert.equal(await page.locator("tr.pricing-evidence-row").count(), 1);
    await page.evaluate(() => document.fonts.ready);
    assert.equal(
      await page.evaluate(
        () => document.documentElement.scrollWidth > innerWidth,
      ),
      false,
    );
    assert.deepEqual(
      await page.locator(".pricing-picker :is(td, th)").evaluateAll((cells) =>
        cells
          .filter((cell) => cell.getClientRects().length)
          .filter(
            (cell) =>
              cell.scrollWidth > cell.clientWidth + 1 ||
              cell.scrollHeight > cell.clientHeight + 1,
          )
          .map((cell) => cell.textContent.slice(0, 30)),
      ),
      [],
    );
    assert.ok(
      await page
        .locator("[data-pricing-service]")
        .evaluateAll((names) =>
          names.every((name) => name.closest("td").clientWidth >= 184),
        ),
    );
    await page.screenshot({
      path: `${out}/en-${width}-oracle-evidence.png`,
      fullPage: true,
    });
  }
  await page.setViewportSize({ width: 390, height: 1000 });
  await page.route("**/api/focus-overview?**", (r) =>
    r.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(degraded),
    }),
  );
  await visit("lang=ko&focus-provider=VMware&focus-period=2024-09");
  await page.locator(".pricing-toolbar button").last().click();
  await loaded();
  await page.locator("[data-pricing-evidence-toggle]").first().click();
  assert.deepEqual(
    await page.locator(".sales-pricing :is(td, th)").evaluateAll((cells) =>
      cells
        .filter((cell) => cell.getClientRects().length)
        .filter((cell) => cell.scrollHeight > cell.clientHeight + 1)
        .map((cell) => cell.textContent.slice(0, 30)),
    ),
    [],
  );
  assert.equal(
    await page.evaluate(
      () => document.documentElement.scrollWidth > innerWidth,
    ),
    false,
  );
  await page.screenshot({
    path: `${out}/ko-390-degraded-evidence.png`,
    fullPage: true,
  });
  await page.unroute("**/api/focus-overview?**");
  await page.setViewportSize({ width: 1440, height: 1000 });
  await visit("lang=en&focus-provider=VMware&focus-period=2024-09");
  page.off("dialog", acceptDialog);
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
        assert.deepEqual(
          await page
            .locator(".sales-pricing :is(td, th)")
            .evaluateAll((cells) =>
              cells
                .filter((cell) => cell.getClientRects().length)
                .filter((cell) => cell.scrollHeight > cell.clientHeight + 1)
                .map((cell) => cell.textContent.slice(0, 30)),
            ),
          [],
        );
        if (view === "rates") {
          await page.locator("[data-pricing-evidence-toggle]").first().click();
          assert.deepEqual(
            await page
              .locator(".sales-pricing :is(td, th)")
              .evaluateAll((cells) =>
                cells
                  .filter((cell) => cell.getClientRects().length)
                  .filter((cell) => cell.scrollHeight > cell.clientHeight + 1)
                  .map((cell) => cell.textContent.slice(0, 30)),
              ),
            [],
          );
          assert.equal(
            await page.evaluate(
              () => document.documentElement.scrollWidth > innerWidth,
            ),
            false,
          );
          await page.screenshot({
            path: `${out}/${locale}-${width}-evidence-open.png`,
            fullPage: true,
          });
          await page.locator("[data-pricing-evidence-toggle]").first().click();
        }
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
