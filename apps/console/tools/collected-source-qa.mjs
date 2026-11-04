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
  variants = [];
page.on("pageerror", (e) => errors.push(e.message));
const url = args.url || "http://127.0.0.1:8805/",
  out = args.output || "qa-output/collected-source";
await mkdir(out, { recursive: true });
const sourceLoaded = () =>
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
    { timeout: 30000 },
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
      await page.locator(".collected-detail button").click();
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
      "Billing interval",
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
  assert.deepEqual(errors, []);
  passed = true;
} finally {
  await writeFile(
    `${out}/report.json`,
    JSON.stringify(
      { verdict: passed ? "PASS" : "FAIL", variants, errors },
      null,
      2,
    ),
  );
  await browser.close();
}
console.log(
  "PASS collected S04: six locales/layouts; actual1121rows/78accounts; detail focus; paging/history; usage/mappedfields/unavailablehistory; provider/month/account; pinned overview roundtrip; malformed/stale/foreign clearing and restart. No owner FIX.",
);
