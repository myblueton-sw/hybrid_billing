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
const context = await browser.newContext(),
  page = await context.newPage(),
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
const url = args.url || "http://127.0.0.1:8787/",
  output = args.output || "qa-output/overview";
await mkdir(output, { recursive: true });
const loaded = async () => {
  await page.waitForFunction(() => {
    const current = new URLSearchParams(location.hash.split("?")[1]), expected = new URLSearchParams();
    for (const k of ["period", "provider", "account", "basis", "q", "after", "account_after", "snapshot"]) if(current.has(`focus-${k}`)) expected.set(k, k === "q" ? current.get(`focus-${k}`).trim() : current.get(`focus-${k}`));
    const root = document.querySelector("[data-focus-state=loaded]");
    return root?.getAttribute("data-focus-query") === expected.toString();
  }, null, {timeout:30000}).catch(async e => { console.error(await page.locator("[data-focus-state]").evaluate(el=>({state:el.getAttribute("data-focus-state"),query:el.getAttribute("data-focus-query"),url:location.hash,error:el.querySelector("[role=alert]")?.textContent}))); throw e; });
};
const open = async (query = "") => {
  await page.goto(url + "#/home?lang=en" + query);
  await loaded();
};
const results = [];
let passed = false;
try {
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + "#/home?lang=" + lang);
      await loaded();
      await page.evaluate(() => document.fonts.ready);
      await page.evaluate(() => window.scrollTo(0, 0));
      assert.equal(await page.locator("[data-focus-rows]").innerText(), "1121");
      assert.equal(await page.locator("#focus-provider option").count(), 7);
      assert.equal(
        await page
          .locator(".focus-amounts")
          .innerText()
          .then((t) => t.includes("27.37875272899")),
        true,
      );
      assert.equal(
        await page.locator(".focus-table").nth(1).locator("tbody tr").count(),
        25,
      );
      assert.equal(
        await page.locator(".focus-table").nth(2).locator("tbody tr").count(),
        25,
      );
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
      assert(
        await page.evaluate(
          (lang) =>
            document.fonts.check(
              `600 16px "${lang === "ko" ? "IBM Plex Sans KR" : lang === "ar" ? "IBM Plex Sans Arabic" : "IBM Plex Sans"}"`,
            ),
          lang,
        ),
      );
      await page.screenshot({ path: `${output}/${lang}-${width}.png` });
      await page.locator(".focus-search").scrollIntoViewIfNeeded();
      await page.screenshot({
        path: `${output}/${lang}-${width}-accounts.png`,
      });
      results.push({
        lang,
        width,
        rows: 1121,
        accountsPage: 25,
        detailPage: 25,
        overflow: false,
      });
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await open();
  const all = await page.request.get(url + "api/focus-overview", {
    headers: { "X-HB-Local-Client": "1" },
  });
  assert.equal(all.status(), 200);
  const actual = await all.json();
  assert.equal(actual.account_count, 78);
  assert.equal(actual.row_count, 1121);
  assert.deepEqual(actual.options.periods, ["2024-03", "2024-09", "2024-10"]);
  assert.equal(actual.provenance["synthetic-contract-fixture"], 6);
  assert.equal(
    actual.provenance["official-public"] +
      actual.provenance["official-public-subset"],
    1115,
  );
  const hostile = await page.request.get(url + "api/focus-overview", {
    headers: {
      "X-HB-Local-Client": "1",
      Origin: "https://evil.example",
      "Sec-Fetch-Site": "cross-site",
    },
  });
  assert.equal(hostile.status(), 403);
  const missing = await page.request.get(url + "api/focus-overview");
  assert.equal(missing.status(), 403);
  const foreign = await page.request.get(
    url + "api/focus-overview?actor=foreign",
    { headers: { "X-HB-Local-Client": "1" } },
  );
  assert.equal(foreign.status(), 400);
  const forged = await page.request.get(
    url + "api/focus-overview?account=" + "f".repeat(64),
    { headers: { "X-HB-Local-Client": "1" } },
  );
  assert.equal(forged.status(), 400);
  const stale = await page.request.get(
    url +
      "api/focus-overview?after=25&basis=effective_cost&snapshot=" +
      actual.snapshot,
    { headers: { "X-HB-Local-Client": "1" } },
  );
  assert.equal(stale.status(), 409);
  await page.locator("input[value=effective_cost]").click();
  await page.waitForURL(/focus-basis=effective_cost/);
  await page.locator("input[value=effective_cost]:checked").waitFor();
  await loaded();
  assert.equal(await page.locator("input[value=effective_cost]").isChecked(), true);
  assert(
    (await page.locator(".focus-amounts").innerText()).includes(
      "18.72651418586",
    ),
  );
  await page.locator("#focus-provider").selectOption("Microsoft");
  await page.waitForURL(/focus-provider=Microsoft/);
  await loaded();
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "51");
  await page.locator("#focus-period").selectOption("2024-10");
  await page.waitForURL(/focus-period=2024-10/);
  await loaded();
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "0");
  await open("&focus-q=%20AWS%20");
  assert.equal(await page.locator("#focus-account-search").inputValue(), "AWS");
  await page
    .locator("#focus-account-search")
    .fill("no-such-public-sample-account");
  await page.locator(".focus-search button").click();
  await page.waitForURL(/focus-q=no-such-public-sample-account/);
  await loaded();
  assert.equal(
    await page.locator(".focus-table").nth(1).locator("tbody tr").count(),
    0,
  );
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "1121");
  await open();
  await page
    .locator(".overview-paging")
    .last()
    .getByRole("button", { name: "Next", exact: true })
    .click();
  await page.waitForURL(/focus-after=25/);
  await loaded();
  assert(page.url().includes("focus-after=25"));
  await page.goBack();
  await loaded();
  assert(!page.url().includes("focus-after=25"));
  await page
    .locator(".focus-table")
    .nth(1)
    .locator("tbody th button")
    .first()
    .click();
  await page.waitForURL(/focus-account=/);
  await loaded();
  assert((await page.locator("[data-focus-rows]").innerText()) !== "1121");
  await open();
  const replay = {};
  for (const provider of ["AWS", "Microsoft"]) {
    const response = await page.request.get(url+"api/focus-overview?provider="+provider,{headers:{"X-HB-Local-Client":"1"}});
    assert.equal(response.status(),200); replay[provider] = await response.json();
  }
  await page.route("**/api/focus-overview?**", async route => {
    const provider = new URL(route.request().url()).searchParams.get("provider");
    if (provider === "AWS") await new Promise(r => setTimeout(r,1500));
    if(replay[provider]) await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify(replay[provider])}).catch(()=>{});
    else await route.continue();
  });
  await page.locator("#focus-provider").selectOption("AWS");
  await page.waitForTimeout(100);
  await page.evaluate(() => {
    const p = new URLSearchParams(location.hash.split("?")[1]);
    p.set("focus-provider", "Microsoft");
    location.hash = "#/home?" + p;
  });
  await loaded();
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "51");
  await page.waitForTimeout(1700);
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "51");
  await page.unroute("**/api/focus-overview?**");
  await page.route("**/api/focus-overview?**", (route) =>
    route.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({ error: "collected_unavailable" }),
    }),
  );
  await page.locator("#focus-provider").selectOption("Oracle");
  await page.locator("[data-focus-state=error]").waitFor();
  assert.equal(await page.locator(".focus-amounts").count(), 0);
  assert.equal(await page.locator(".focus-table").count(), 0);
  await page.unroute("**/api/focus-overview?**");
  await page
    .locator("[role=alert]")
    .getByRole("button", { name: "Refresh query", exact: true })
    .click();
  await loaded();
  assert.equal(await page.locator("[data-focus-rows]").innerText(), "7");
  await page.goto(url + "#/home?lang=en&focus-account=" + "f".repeat(64));
  await page.locator("[data-focus-state=error]").waitFor();
  assert.equal(await page.locator(".focus-amounts").count(), 0);
  passed = true;
} finally {
  await writeFile(
    `${output}/report.json`,
    JSON.stringify(
      { verdict: passed ? "PASS" : "FAIL", variants: results, errors },
      null,
      2,
    ),
  );
  await browser.close();
}
assert.equal(results.length, 6);
assert.deepEqual(errors, []);
console.log(
  "PASS: six locale/layout variants; retained-source readback; precise basis switch; provider/month/empty/account/search/pages/history; hostile proxy; stale continuation; late response; failure clearing/recovery; foreign scope. Owner review untouched.",
);
