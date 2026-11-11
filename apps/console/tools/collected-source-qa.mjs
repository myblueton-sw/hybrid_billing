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
const page = await browser.newPage(),
  errors = [],
  variants = [],
  fieldVariants = [];
page.on("pageerror", (e) => errors.push(e.message));
let readerTurn = Promise.resolve(),
  serializedReads = 0,
  slowestReadMs = 0;
await page.route("**/api/focus-*", (route) => {
  serializedReads++;
  readerTurn = readerTurn.then(async () => {
    try {
      const started = Date.now(),
        response = await route.fetch({ timeout: 120000 });
      slowestReadMs = Math.max(slowestReadMs, Date.now() - started);
      await route.fulfill({ response });
    } catch {
      await route.abort().catch(() => {});
    }
  });
  return readerTurn;
});
const responses = [];
page.on("response", async (r) => {
  if (!r.url().includes("/api/focus-") || r.status() === 200) return;
  responses.push({
    at: new Date().toISOString(),
    path: new URL(r.url()).pathname + new URL(r.url()).search,
    status: r.status(),
    body: await r.text().catch(() => ""),
  });
});
page.on("requestfailed", (r) => {
  if (r.url().includes("/api/focus-"))
    responses.push({
      at: new Date().toISOString(),
      path: new URL(r.url()).pathname + new URL(r.url()).search,
      failure: r.failure()?.errorText,
    });
});
const url = args.url || "http://127.0.0.1:8805/",
  out = args.output || "qa-output/collected-source";
await mkdir(out, { recursive: true });
const sourceLoaded = () =>
  sourceSettled().catch(async (cause) => {
    const state = await page
      .locator(".collected-source")
      .evaluate((e) => ({
        state: e.dataset.sourceState,
        query: e.dataset.sourceQuery,
        alert: e.querySelector("section[role=alert]")?.textContent ?? null,
      }))
      .catch(() => null);
    throw Error(`source not loaded: ${JSON.stringify(state)}`, { cause });
  });
const sourceSettled = () =>
  page.waitForFunction(
    () => {
      const p = new URLSearchParams(location.hash.split("?")[1]),
        q = new URLSearchParams();
      for (const k of [
        "period",
        "provider",
        "account",
        "basis",
        "q",
        "after",
        "account_after",
        "snapshot",
      ])
        if (p.has("focus-" + k))
          q.set(
            k,
            k === "q" ? p.get("focus-" + k).trim() : p.get("focus-" + k),
          );
      const root = document.querySelector("[data-source-state=loaded]");
      return (
        root?.getAttribute("data-source-query") === q.toString() &&
        root?.getAttribute("data-source-view") === (p.get("view") ?? "rows")
      );
    },
    null,
    { timeout: 150000 },
  );
const open = async (q = "") => {
  await page.goto(url + "#/source-data?lang=en" + q);
  await sourceLoaded();
};
let passed = false;
try {
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + "#/source-data?lang=" + lang);
      await sourceLoaded();
      await page.evaluate(() => document.fonts.ready);
      await page.evaluate(() => scrollTo(0, 0));
      assert.equal(
        await page.locator("[data-source-total]").innerText(),
        "1121",
      );
      assert.equal(await page.locator(".collected-table tbody tr").count(), 25);
      assert.equal(await page.locator("#confirm-review").count(), 1);
      assert.equal(
        await page.locator("html").getAttribute("dir"),
        lang === "ar" ? "rtl" : "ltr",
      );
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      assert.equal(
        await page
          .locator(".collected-table td")
          .first()
          .evaluate((e) => getComputedStyle(e).fontSize),
        "16px",
      );
      await page.screenshot({ path: `${out}/${lang}-${width}.png` });
      await page.locator(".collected-table").scrollIntoViewIfNeeded();
      await page.screenshot({ path: `${out}/${lang}-${width}-table.png` });
      assert(
        await page
          .locator(".collected-table tbody th small bdi")
          .first()
          .evaluate(
            (e) =>
              e.getBoundingClientRect().width <=
              e.closest("th").getBoundingClientRect().width,
          ),
      );
      await page.locator(".collected-table tbody th button").first().click();
      await page.locator(".collected-detail").waitFor();
      assert.equal(
        await page.evaluate(() => document.activeElement.className),
        "collected-detail",
      );
      await page.screenshot({ path: `${out}/${lang}-${width}-detail.png` });
      await page.locator(".collected-detail header button").click();
      assert.equal(
        await page.evaluate(() => document.activeElement.tagName),
        "BUTTON",
      );
      assert.equal(await page.locator(".collected-detail").count(), 0);
      variants.push({
        lang,
        width,
        rows: 1121,
        page: 25,
        detailFocus: true,
        overflow: false,
      });
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await open();
  const initial = await page.request.get(url + "api/focus-overview", {
    headers: { "X-HB-Local-Client": "1" },
  });
  assert.equal(initial.status(), 200);
  const base = await initial.json();
  assert.equal(base.row_count, 1121);
  assert.equal(base.account_count, 78);
  assert.equal(
    typeof base.rows[0].consumed_quantity === "string" ||
      base.rows[0].consumed_quantity === null,
    true,
  );
  await page
    .locator(".collected-list-header .collected-pages")
    .getByRole("button", { name: "Next", exact: true })
    .click();
  await page.waitForURL(/focus-after=25/);
  await sourceLoaded();
  assert.equal(await page.locator(".collected-detail").count(), 0);
  const ids = await page
    .locator("[data-row-id]")
    .evaluateAll((es) => es.map((e) => e.dataset.rowId));
  assert(!base.rows.some((r) => ids.includes(r.id)));
  await page.goBack();
  await sourceLoaded();
  assert(!page.url().includes("focus-after=25"));
  await page
    .locator(".planning-tabs button")
    .filter({ hasText: "Usage" })
    .click();
  await page.waitForURL(/view=usage/);
  await sourceLoaded();
  assert(
    (await page.locator(".collected-table thead").innerText()).includes(
      "Pricing quantity",
    ),
  );
  await page.screenshot({ path: `${out}/en-1440-usage.png` });
  await page.locator(".planning-tabs button").nth(1).click();
  await page.waitForURL(/view=fields/);
  await sourceLoaded();
  assert(
    (await page.locator(".collected-table thead").innerText()).includes(
      "Billing period",
    ),
  );
  await page.locator(".planning-tabs button").nth(3).click();
  await page.waitForURL(/view=history/);
  await sourceLoaded();
  assert.equal(await page.locator(".collected-table").count(), 0);
  assert.equal(await page.locator(".collected-unavailable").count(), 1);
  await open();
  await page.locator("#source-provider").selectOption("Microsoft");
  await page.waitForURL(/focus-provider=Microsoft/);
  await sourceLoaded();
  assert.equal(await page.locator("[data-source-total]").innerText(), "51");
  await page.locator("#source-period").selectOption("2024-10");
  await page.waitForURL(/focus-period=2024-10/);
  await sourceLoaded();
  assert.equal(await page.locator("[data-source-total]").innerText(), "0");
  assert.equal(await page.locator(".collected-table tbody tr").count(), 0);
  await open("&focus-q=AWS");
  await page.locator(".collected-account summary").click();
  assert.equal(
    await page.locator("#source-account-search").inputValue(),
    "AWS",
  );
  await page.locator(".collected-account li button").first().click();
  await page.waitForURL(/focus-account=/);
  await sourceLoaded();
  assert(Number(await page.locator("[data-source-total]").innerText()) < 1121);
  await page.goto(
    url +
      "#/home?lang=en&focus-provider=Microsoft&focus-basis=effective_cost&focus-q=Microsoft",
  );
  await page.locator("[data-focus-state=loaded]").waitFor();
  await page.locator(".focus-source > a").click();
  await page.waitForURL(/source-data/);
  await sourceLoaded();
  let selected = new URLSearchParams(page.url().split("?")[1]);
  assert.equal(selected.get("focus-q"), "Microsoft");
  assert.equal(selected.get("focus-provider"), "Microsoft");
  assert.equal(selected.get("focus-basis"), "effective_cost");
  assert.equal(selected.get("focus-snapshot")?.length, 64);
  assert(
    (await page.locator(".collected-totals").innerText()).includes(
      "1.97651418586",
    ),
  );
  await page.locator(".collected-toolbar a").click();
  await page.waitForURL(/#\/home/);
  await page.locator("[data-focus-state=loaded]").waitFor();
  assert(
    (await page.locator(".focus-amounts").innerText()).includes(
      "1.97651418586",
    ),
  );
  await page.goto(
    url + "#/source-data?lang=en&focus-snapshot=" + "f".repeat(64),
  );
  await page.locator("[data-source-state=error]").waitFor();
  assert.equal(await page.locator(".collected-table").count(), 0);
  await page.locator("[role=alert] button").click();
  await sourceLoaded();
  assert.equal(await page.locator("[data-source-total]").innerText(), "1121");
  const invalid = structuredClone(base);
  invalid.rows[0].consumed_quantity = 1;
  await page.route("**/api/focus-overview?**", (r) =>
    r.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(invalid),
    }),
  );
  await page.locator("#source-provider").selectOption("Oracle");
  await page.locator("[data-source-state=error]").waitFor();
  assert.equal(await page.locator(".collected-table").count(), 0);
  await page.unroute("**/api/focus-overview?**");
  await open();
  await page.locator(".collected-table tbody th button").first().click();
  await page.locator(".collected-detail").waitFor();
  await page.locator("#confirm-review").click();
  await page.locator("dialog[open]").waitFor();
  await page.keyboard.press("Escape");
  assert.equal(await page.locator("dialog[open]").count(), 0);
  assert.equal(await page.locator(".collected-detail").count(), 1);
  await page.keyboard.press("Escape");
  assert.equal(await page.locator(".collected-detail").count(), 0);
  const replay = {};
  for (const provider of ["AWS", "Microsoft"]) {
    const r = await page.request.get(
      url + "api/focus-overview?provider=" + provider,
      { headers: { "X-HB-Local-Client": "1" } },
    );
    assert.equal(r.status(), 200);
    replay[provider] = await r.json();
  }
  await page.route("**/api/focus-overview?**", async (route) => {
    const p = new URL(route.request().url()).searchParams.get("provider");
    if (p === "AWS") await new Promise((r) => setTimeout(r, 1500));
    if (replay[p])
      await route
        .fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(replay[p]),
        })
        .catch(() => {});
    else await route.continue();
  });
  await page.locator("#source-provider").selectOption("AWS");
  await page.waitForURL(/focus-provider=AWS/);
  await page.evaluate(() => {
    let q = new URLSearchParams(location.hash.split("?")[1]);
    q.set("focus-provider", "Microsoft");
    location.hash = "#/source-data?" + q;
  });
  await sourceLoaded();
  assert.equal(await page.locator("[data-source-total]").innerText(), "51");
  await page.waitForTimeout(1700);
  assert.equal(await page.locator("[data-source-total]").innerText(), "51");
  await page.unroute("**/api/focus-overview?**");
  await page.goto(
    url + "#/source-data?lang=en&focus-account=" + "f".repeat(64),
  );
  await page.locator("[data-source-state=error]").waitFor();
  assert.equal(await page.locator(".collected-totals").count(), 0);
  await page.unroute("**/api/focus-overview?**").catch(() => {});
  const field = (name) =>
    page.locator(`.collected-detail [data-field=${name}]`);
  const openRow = async (after, position) => {
    await page.goto("about:blank");
    await page.goto(
      url +
        "#/source-data?lang=en&view=fields" +
        (after ? `&focus-after=${after}&focus-snapshot=${base.snapshot}` : ""),
    );
    await sourceLoaded();
    await page
      .locator(".collected-table tbody th button")
      .nth(position)
      .click();
    await page.locator(".collected-detail").waitFor();
  };
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto(url + "#/source-data?lang=en&view=fields");
  await sourceLoaded();
  assert.equal(await page.locator(".collected-band th").count(), 3);
  assert.equal(
    await page.locator(".collected-reference").evaluate((e) => e.open),
    false,
  );
  await page.locator(".collected-reference summary").click();
  assert.equal(
    await page
      .locator(".collected-reference tbody tr:not(.collected-group)")
      .count(),
    25,
  );
  assert.equal(await page.locator(".collected-reference tbody").count(), 4);
  assert(
    (await page.locator(".collected-reference").innerText()).includes(
      "ChargePeriodEnd",
    ),
  );
  await page.locator(".collected-reference summary").click();
  const firstMissing = page
    .locator(".collected-table tbody tr")
    .first()
    .locator("[data-missing-count]");
  assert.equal(await firstMissing.getAttribute("data-missing-count"), "2");
  assert((await firstMissing.innerText()).includes("ContractedCost"));
  assert((await firstMissing.innerText()).includes("ContractedUnitPrice"));
  assert.equal(
    await page
      .locator(".collected-fields-table td .collected-token")
      .evaluateAll(
        (es) => es.filter((e) => e.getClientRects().length === 1).length,
      ),
    150,
  );
  assert((await firstMissing.innerText()).startsWith("2 missing"));
  assert(
    await page
      .locator(".collected-source")
      .evaluate(
        (e) =>
          e.getBoundingClientRect().width >=
            e.parentElement.getBoundingClientRect().width - 1 &&
          getComputedStyle(e).maxWidth === "none",
      ),
  );
  await openRow(0, 0);
  assert.equal(
    (await field("billed_cost").locator(".collected-value").innerText()).trim(),
    "USD 0.24",
  );
  assert.equal(
    (
      await field("effective_cost").locator(".collected-value").innerText()
    ).trim(),
    "USD 0.00",
  );
  assert.equal(
    (
      await field("effective_cost").locator(".collected-note").innerText()
    ).trim(),
    "Exact zero",
  );
  assert.equal(
    (await field("billed_cost").locator(".collected-note").innerText()).trim(),
    "",
  );
  assert.equal(
    (
      await field("contracted_cost").locator(".collected-value").innerText()
    ).trim(),
    "Missing",
  );
  assert.equal(
    (
      await field("contracted_cost").locator(".collected-note").innerText()
    ).trim(),
    "No value in the collected row",
  );
  assert.equal(
    (
      await field("contracted_unit_price")
        .locator(".collected-value")
        .innerText()
    ).trim(),
    "Missing\nOCPU Hours",
  );
  assert.equal(
    (
      await field("consumed_quantity").locator(".collected-value").innerText()
    ).trim(),
    "8.00\nOCPU Hours",
  );
  assert.equal(
    (await field("charge_end").locator(".collected-note").innerText()).trim(),
    "End exclusive · UTC",
  );
  assert.equal(
    (
      await field("charge_start").locator(".collected-value").innerText()
    ).trim(),
    "2024-09-30T22:00:00+00:00",
  );
  assert.equal(
    await page.locator(".collected-detail [data-period-note]").count(),
    1,
  );
  assert.equal(
    await page.locator(".collected-detail [data-missing-reason]").count(),
    1,
  );
  assert(
    (await field("timezone").innerText()).includes("does not carry its tenant"),
  );
  assert((await field("timezone").innerText()).includes("Asia/Seoul, UTC"));
  assert.equal((await field("utc_month").innerText()).trim(), "2024-09");
  assert.equal(await page.locator(".collected-detail a").count(), 0);
  assert.equal(
    await page.locator(".collected-detail .collected-unsupplied li").count(),
    6,
  );
  assert.equal(await page.locator(".collected-detail h3").count(), 5);
  const neutralBorders = async () => {
    const borders = await page
      .locator(
        ".collected-detail, .collected-detail *, .collected-reference, .collected-reference *, .collected-fields-table *",
      )
      .evaluateAll((es) => {
        const found = new Set();
        let decorated = 0;
        for (const e of es) {
          const c = getComputedStyle(e);
          if (c.boxShadow !== "none" || parseFloat(c.borderTopLeftRadius) > 0)
            decorated++;
          if (e.tagName === "BUTTON") continue;
          for (const side of ["Top", "Right", "Bottom", "Left"])
            if (
              parseFloat(c[`border${side}Width`]) > 0 &&
              c[`border${side}Style`] !== "none"
            )
              found.add(c[`border${side}Color`]);
        }
        return { colors: [...found], decorated };
      });
    assert.equal(borders.decorated, 0);
    for (const color of borders.colors) {
      const [r, g, b] = color.match(/[\d.]+/g).map(Number);
      assert(
        Math.max(r, g, b) - Math.min(r, g, b) <= 32,
        "non-neutral border " + color,
      );
    }
  };
  await neutralBorders();
  const firstId = await page
    .locator(".collected-table tbody tr")
    .first()
    .getAttribute("data-row-id");
  await page.keyboard.press("Escape");
  assert.equal(await page.locator(".collected-detail").count(), 0);
  assert.equal(
    await page.evaluate(
      () => document.activeElement.closest("tr")?.dataset.rowId,
    ),
    firstId,
  );
  await page.locator(".collected-table tbody th button").nth(3).click();
  await page.locator(".collected-detail").waitFor();
  await page.locator(".collected-detail footer button").click();
  assert.equal(
    await page.evaluate(() =>
      [...document.querySelectorAll(".collected-table tbody tr")].indexOf(
        document.activeElement.closest("tr"),
      ),
    ),
    3,
  );
  await openRow(0, 1);
  assert.equal(
    (
      await field("consumed_quantity").locator(".collected-value").innerText()
    ).trim(),
    "3,600.00\nseconds",
  );
  assert.equal(
    (
      await field("pricing_quantity").locator(".collected-value").innerText()
    ).trim(),
    "1.00\nmonth",
  );
  assert.equal(
    (await field("billed_cost").locator(".collected-value").innerText()).trim(),
    "USD 0.00008",
  );
  assert.equal(
    await page.locator(".collected-detail [data-period-note]").count(),
    0,
  );
  assert.equal(
    await page.locator(".collected-detail [data-missing-reason]").count(),
    0,
  );
  await openRow(550, 22);
  assert.equal(
    (
      await field("consumed_quantity").locator(".collected-value").innerText()
    ).trim(),
    "Missing\nUnit missing",
  );
  assert.equal(
    (
      await field("list_unit_price").locator(".collected-value").innerText()
    ).trim(),
    "Missing\nHours",
  );
  assert.equal(
    (
      await field("pricing_quantity").locator(".collected-value").innerText()
    ).trim(),
    "0.00\nHours",
  );
  assert.equal(
    (
      await field("pricing_quantity").locator(".collected-note").innerText()
    ).trim(),
    "Exact zero",
  );
  assert.equal(
    (await field("billed_cost").locator(".collected-value").innerText()).trim(),
    "USD -2.6137",
  );
  assert.equal(
    (
      await field("effective_cost").locator(".collected-value").innerText()
    ).trim(),
    "USD -3.00",
  );
  assert.equal(
    await page
      .locator(".collected-table tbody tr")
      .nth(22)
      .locator("[data-missing-count]")
      .getAttribute("data-missing-count"),
    "3",
  );
  await openRow(1100, 19);
  assert.equal(
    (await field("billed_cost").locator(".collected-value").innerText()).trim(),
    "CNY 8.80",
  );
  assert.equal(
    (await field("currency").locator(".collected-value").innerText()).trim(),
    "CNY",
  );
  assert(
    (await field("evidence_class").innerText()).includes("Synthetic test data"),
  );
  await openRow(100, 17);
  assert.equal(
    (await field("charge_month").locator("bdi").first().innerText()).trim(),
    "2024-10",
  );
  assert((await field("utc_month").innerText()).startsWith("2024-09"));
  assert(
    (await field("utc_month").innerText()).includes(
      "differs because it uses the tenant time zone",
    ),
  );
  await page.screenshot({ path: `${out}/en-1440-month-differs.png` });
  await page
    .locator(".collected-list-header .collected-pages")
    .getByRole("button", { name: "Next", exact: true })
    .click();
  await sourceLoaded();
  assert.equal(await page.locator(".collected-detail").count(), 0);
  assert.equal(await page.locator("[data-selected]").count(), 0);
  await openRow(0, 2);
  await page.locator("#source-provider").selectOption("Oracle");
  await page.waitForURL(/focus-provider=Oracle/);
  await sourceLoaded();
  assert.equal(await page.locator(".collected-detail").count(), 0);
  assert(!page.url().includes("focus-snapshot"));
  await page.locator(".collected-table tbody th button").first().click();
  await page.locator(".collected-detail").waitFor();
  await page
    .locator(".collected-toolbar button")
    .filter({ hasText: "Refresh query" })
    .click();
  await sourceLoaded();
  assert.equal(await page.locator(".collected-detail").count(), 0);
  await page.locator(".collected-table tbody th button").first().click();
  await page.locator(".collected-detail").waitFor();
  await page.locator(".planning-tabs button").nth(3).click();
  await page.waitForURL(/view=history/);
  await sourceLoaded();
  await page.locator(".planning-tabs button").nth(1).click();
  await page.waitForURL(/view=fields/);
  await sourceLoaded();
  assert.equal(await page.locator(".collected-detail").count(), 0);
  await page.locator(".collected-table tbody th button").first().click();
  await page.locator(".collected-detail").waitFor();
  for (const [status, body] of [
    [403, { error: "scope_denied", request_id: "0".repeat(32) }],
    [503, { error: "source_busy", request_id: "0".repeat(32) }],
  ]) {
    await page.route("**/api/focus-overview?**", (r) =>
      r.fulfill({
        status,
        contentType: "application/json",
        body: JSON.stringify(body),
      }),
    );
    await page
      .locator(".collected-toolbar button")
      .filter({ hasText: "Refresh query" })
      .click()
      .catch(() => {});
    await page.locator("[data-source-state=error]").waitFor();
    assert.equal(await page.locator(".collected-detail").count(), 0);
    assert.equal(await page.locator(".collected-table").count(), 0);
    assert.equal(await page.locator(".collected-totals").count(), 0);
    assert.equal(await page.locator(".collected-reference").count(), 0);
    assert(
      (
        await page.locator(".collected-source section[role=alert]").innerText()
      ).includes(body.error),
    );
    await page.unroute("**/api/focus-overview?**");
    await page.locator("[role=alert] button").click();
    await sourceLoaded();
    await page.locator(".collected-table tbody th button").first().click();
    await page.locator(".collected-detail").waitFor();
  }
  await page.unroute("**/api/focus-overview?**").catch(() => {});
  await page.goto("about:blank");
  await page.goto(url + "#/source-data?lang=en");
  await sourceLoaded();
  await page.locator(".collected-table tbody th button").nth(2).click();
  await page.locator(".collected-detail").waitFor();
  assert.equal(await page.locator(".collected-selected").count(), 1);
  const switchedId = await page
    .locator(".collected-table tbody tr")
    .nth(2)
    .getAttribute("data-row-id");
  await page.locator(".planning-tabs button").nth(1).click();
  await page.waitForURL(/view=fields/);
  await sourceLoaded();
  assert.equal(await page.locator(".collected-detail").count(), 0);
  assert.equal(await page.locator("[data-selected]").count(), 0);
  assert.equal(await page.locator(`[data-row-id="${switchedId}"]`).count(), 1);
  const degraded = structuredClone(base);
  degraded.versions = ["1.0", "9.9"];
  degraded.tenants = [{ key: "sample", timezone: "UTC" }];
  degraded.rows[0].charge_start = "2024-09-30T22:00:00+09:00";
  await page.route("**/api/focus-overview**", (r) =>
    r.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(degraded),
    }),
  );
  await page.goto("about:blank");
  await page.goto(url + "#/source-data?lang=en&view=fields");
  await sourceLoaded();
  assert.equal(
    (
      await page
        .locator(".collected-table tbody tr")
        .first()
        .locator("td .collected-token")
        .first()
        .innerText()
    ).trim(),
    "2024-09-30T22:00:00+09:00",
  );
  await page.locator(".collected-table tbody th button").first().click();
  await page.locator(".collected-detail").waitFor();
  assert.equal(
    (await field("charge_start").locator(".collected-note").innerText()).trim(),
    "Time format not interpreted",
  );
  assert.equal(await field("utc_month").count(), 0);
  assert.equal((await field("timezone").innerText()).trim(), "UTC");
  assert.equal(
    await page
      .locator(".collected-detail [data-unmapped-version='9.9']")
      .count(),
    21,
  );
  assert(
    (
      await page
        .locator(".collected-detail .collected-caption")
        .first()
        .innerText()
    ).includes("does not carry its own version"),
  );
  assert.equal(
    await page.locator(".collected-detail .collected-unsupplied li").count(),
    6,
  );
  assert.equal(
    (
      await field("billed_cost")
        .locator(".collected-names bdi")
        .first()
        .innerText()
    ).trim(),
    "BilledCost",
  );
  await page.unroute("**/api/focus-overview**");
  await page.goto("about:blank");
  await page.goto(url + "#/imports?lang=en");
  const handoff = page
    .locator('main a[href="#/source-data?lang=en"]:visible')
    .first();
  await handoff.waitFor({ timeout: 60000 });
  assert(!(await handoff.getAttribute("href")).includes("focus-period"));
  await handoff.click();
  await sourceLoaded();
  assert.equal(await page.locator("[data-source-total]").innerText(), "1121");
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto("about:blank");
      await page.goto(url + "#/source-data?view=fields&lang=" + lang);
      await sourceLoaded();
      await page.evaluate(() => document.fonts.ready);
      await page.locator(".collected-table").scrollIntoViewIfNeeded();
      await page
        .locator(".collected-table")
        .screenshot({ path: `${out}/${lang}-${width}-fields-table.png` });
      await page.locator(".collected-reference summary").click();
      await page
        .locator(".collected-reference")
        .screenshot({ path: `${out}/${lang}-${width}-fields-reference.png` });
      await page.locator(".collected-reference summary").click();
      await page.locator(".collected-table tbody th button").first().click();
      await page.locator(".collected-detail").waitFor();
      await page
        .locator(".collected-detail")
        .screenshot({ path: `${out}/${lang}-${width}-fields-detail.png` });
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      assert.equal(
        await page
          .locator(
            ".collected-detail [data-field=billed_cost] .collected-value",
          )
          .evaluate((e) => getComputedStyle(e).fontSize),
        "16px",
      );
      assert.equal(
        await page
          .locator(
            ".collected-detail [data-field=billed_cost] .collected-value bdi",
          )
          .evaluate((e) => getComputedStyle(e).direction),
        "ltr",
      );
      assert.equal(
        (
          await page
            .locator(
              ".collected-detail [data-field=effective_cost] .collected-value",
            )
            .innerText()
        ).trim(),
        "USD 0.00",
      );
      assert.equal(
        await page
          .locator(
            ".collected-detail [data-field=contracted_cost] .collected-value bdi",
          )
          .count(),
        0,
      );
      assert.equal(
        (
          await page
            .locator(
              ".collected-detail [data-field=contracted_cost] .collected-value",
            )
            .innerText()
        ).trim(),
        { en: "Missing", ko: "누락", ar: "مفقود" }[lang],
      );
      assert.equal(
        (
          await page
            .locator(
              ".collected-detail [data-field=effective_cost] .collected-note",
            )
            .innerText()
        ).trim(),
        { en: "Exact zero", ko: "정확한 0", ar: "صفر دقيق" }[lang],
      );
      assert(
        await page.evaluate(
          () =>
            document
              .querySelector(".collected-field-table")
              .getBoundingClientRect().width <=
            document
              .querySelector(".collected-detail-source")
              .getBoundingClientRect().width +
              1,
        ),
      );
      assert.equal(await page.locator(".collected-selected").count(), 1);
      await neutralBorders();
      if (width === 390)
        assert(
          await page
            .locator(".collected-detail button")
            .evaluateAll((es) =>
              es.every((e) => e.getBoundingClientRect().height >= 44),
            ),
        );
      fieldVariants.push({ lang, width, overflow: false, detail: true });
    }
  assert.deepEqual(
    responses.filter((r) => r.status).map((r) => r.status),
    [409, 400, 403, 503],
  );
  assert.deepEqual(
    responses.filter((r) => r.failure && r.failure !== "net::ERR_ABORTED"),
    [],
  );
  assert.deepEqual(errors, []);
  passed = true;
} finally {
  await writeFile(
    `${out}/report.json`,
    JSON.stringify(
      {
        verdict: passed ? "PASS" : "FAIL",
        variants,
        fieldVariants,
        errors,
        serializedReads,
        slowestReadMs,
        responses,
      },
      null,
      2,
    ),
  );
  await browser.close();
}
console.log(
  "PASS collected S04: six locales/layouts; actual1121rows/78accounts; detail focus; paging/history; usage/mappedfields/unavailablehistory; provider/month/account; pinned overview roundtrip; malformed/stale/foreign clearing and restart; source-field origins, missing/zero/negative/currency/unit rows, denied and failed read clearing, focus return, neutral borders in six variants. No owner FIX.",
);
