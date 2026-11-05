import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import assert from "node:assert/strict";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .flatMap((v, i, a) => (v.startsWith("--") ? [[v.slice(2), a[i + 1]]] : [])),
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
  out = args.output || "qa-output/commitment-benefits",
  base = "focus-provider=AWS&focus-period=all";
await mkdir(out, { recursive: true });
const root = () => page.locator("[data-b06-state]");
const loaded = () => page.locator("[data-b06-state=loaded]").waitFor();
const selected = (id) =>
  page.waitForFunction(
    (id) =>
      document
        .querySelector("[data-b06-selected-group]")
        ?.getAttribute("data-b06-selected-group") === id,
    id,
  );
const tab = (i) => page.locator(".planning-tabs button").nth(i).click();
const response = () =>
  page.waitForResponse(
    (r) =>
      r.url().includes("/api/focus-commitments?") &&
      r.request().method() === "GET",
  );
let passed = false;
try {
  const first = response();
  await page.goto(url + "#/commitments?lang=en&" + base);
  const initial = await (await first).json();
  await loaded();
  assert.equal(initial.group_count, 2);
  assert.equal(initial.identified_rows, 4);
  assert.equal(initial.row_count, 942);
  assert.equal(initial.selected, null);
  const ids = initial.groups.map((g) => g.id);
  assert.equal(await page.locator("button[data-b06-group]").count(), 2);
  assert.equal(await page.locator("[data-b06-detail]").count(), 0);
  await page.locator("button[data-b06-group]").first().click();
  await selected(ids[0]);
  await page.locator("[data-b06-detail]").waitFor();
  assert.ok(
    (
      await page
        .locator("[data-b06-detail] [data-b06-cost=billed_cost]")
        .innerText()
    ).includes("USD 0.00"),
  );
  assert.ok(
    (
      await page
        .locator("[data-b06-detail] [data-b06-cost=effective_cost]")
        .innerText()
    ).includes("USD 0.00"),
  );
  assert.equal(await page.locator("#b06-row-next").isDisabled(), true);
  await tab(1);
  await page
    .locator('[data-b06-status=used][data-b06-status-rows="3"]')
    .waitFor();
  await page
    .locator('[data-b06-status=unused][data-b06-status-rows="0"]')
    .waitFor();
  assert.match(
    await page.locator("[data-b06-status=unused]").innerText(),
    /No observed rows/i,
  );
  await tab(2);
  await page.locator("[data-b06-view=benefits]").waitFor();
  assert.match(
    await page.locator("[data-b06-detail]").innerText(),
    /unconnected|unknown/i,
  );
  await tab(3);
  await page.locator("[data-b06-view=reconciliation]").waitFor();
  assert.match(
    await page.locator("[data-b06-detail]").innerText(),
    /not.*complete|incomplete|unverified/i,
  );
  await page.locator("#b06-close").click();
  await selected("");
  await page.waitForFunction(
    (id) => document.activeElement?.getAttribute("data-b06-group") === id,
    ids[0],
  );
  await page.locator("button[data-b06-group]").nth(1).click();
  await selected(ids[1]);
  await page.locator("#b06-close").focus();
  await page.keyboard.press("Escape");
  await selected("");
  await page.waitForFunction(
    (id) => document.activeElement?.getAttribute("data-b06-group") === id,
    ids[1],
  );
  await tab(0);
  for (const q of ["AWS", " AWS "]) {
    await page.locator("#b06-search").fill(q);
    await Promise.all([
      response(),
      page.getByRole("button", { name: "Search", exact: true }).click(),
    ]);
    await loaded();
  }
  await page.locator("#b06-search").fill("HB106_NO_MATCH_SYNTHETIC");
  await Promise.all([
    response(),
    page.getByRole("button", { name: "Search", exact: true }).click(),
  ]);
  await loaded();
  assert.equal(await page.locator("button[data-b06-account]").count(), 0);
  assert.equal(await page.locator("button[data-b06-group]").count(), 2);
  await page.locator("#b06-search").fill("");
  await Promise.all([
    response(),
    page.getByRole("button", { name: "Search", exact: true }).click(),
  ]);
  await loaded();
  const accountButton = page.locator("button[data-b06-account]").first();
  if (!(await accountButton.isVisible()))
    await page.locator(".b06-account-scope summary").click();
  const account = await accountButton.getAttribute("data-b06-account");
  const next = response();
  await accountButton.click();
  const accountData = await (await next).json();
  await loaded();
  assert.ok(accountData.groups.every((g) => g.account === account));
  assert.equal(await page.locator("[data-b06-detail]").count(), 0);
  if (!(await page.locator("#b06-all-accounts").isVisible()))
    await page.locator(".b06-account-scope summary").click();
  await Promise.all([response(), page.locator("#b06-all-accounts").click()]);
  await loaded();
  await page.locator("button[data-b06-group]").first().click();
  await selected(ids[0]);
  await page.route("**/api/focus-commitments?**", (r) =>
    r.fulfill({
      status: 403,
      contentType: "application/json",
      body: JSON.stringify({ error: "scope_denied" }),
    }),
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await page.locator("[data-b06-state=error]").waitFor();
  assert.equal(
    await page.locator("[data-b06-detail],button[data-b06-group]").count(),
    0,
  );
  await page.unroute("**/api/focus-commitments?**");
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await loaded();
  await page.route("**/api/focus-commitments?**", (r) =>
    r.fulfill({
      status: 503,
      contentType: "application/json",
      body: JSON.stringify({ error: "source_busy" }),
    }),
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await page.locator("[data-b06-state=error]").waitFor();
  await page.unroute("**/api/focus-commitments?**");
  await page
    .getByRole("button", { name: "Retry same query", exact: true })
    .click();
  await loaded();
  const snap =
    (await page.evaluate(() =>
      new URLSearchParams(location.hash.split("?")[1]).get("focus-snapshot"),
    )) || initial.snapshot;
  await page.evaluate((id) => {
    const original = crypto.subtle.digest.bind(crypto.subtle);
    crypto.subtle.digest = async (algo, bytes) => {
      if (new TextDecoder().decode(bytes).includes("group=" + id)) {
        window.__b06DelayedHash = true;
        await new Promise((r) => setTimeout(r, 2200));
      }
      return original(algo, bytes);
    };
  }, ids[0]);
  for (const id of ids) {
    await page.evaluate(
      ({ base, id, snap }) => {
        location.hash =
          "#/commitments?lang=en&" +
          base +
          "&focus-snapshot=" +
          snap +
          "&commitment-group=" +
          id;
      },
      { base, id, snap },
    );
    if (id === ids[0])
      await page.waitForFunction(() => window.__b06DelayedHash === true);
  }
  await selected(ids[1]);
  await page.waitForTimeout(2300);
  assert.equal(await root().getAttribute("data-b06-selected-group"), ids[1]);
  const clear = response();
  await page.goto(
    url + "#/commitments?lang=en&focus-provider=VMware&focus-period=2024-09",
  );
  const empty = await (await clear).json();
  await loaded();
  assert.equal(empty.group_count, 0);
  assert.equal(empty.row_count, 3);
  assert.equal(await page.locator("button[data-b06-group]").count(), 0);
  assert.equal(await page.locator("[data-b06-detail]").count(), 0);
  assert.equal(
    await page.evaluate(() => localStorage.getItem("hb-console-review-v1")),
    null,
  );
  console.log(
    "PASS actual zero costs, classifications, scope, focus, empty/error/retry, stale hash and no owner acknowledgement",
  );
  const variants = [];
  for (const locale of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + "#/commitments?lang=" + locale + "&" + base);
      await loaded();
      await page.locator("button[data-b06-group]").first().click();
      await selected(ids[0]);
      await page.evaluate(() => document.fonts.ready);
      for (const [i, view] of [
        "commitments",
        "utilization",
        "benefits",
        "reconciliation",
      ].entries()) {
        await tab(i);
        await page.locator("[data-b06-view=" + view + "]").waitFor();
        const c = await root().evaluate((el) => ({
          overflow: document.documentElement.scrollWidth > innerWidth,
          body: getComputedStyle(el).fontSize,
          direction: getComputedStyle(el).direction,
          font: getComputedStyle(el).fontFamily,
          headings: [...el.querySelectorAll("h2")].map(
            (e) => getComputedStyle(e).fontSize,
          ),
          inputs: [...el.querySelectorAll("input,select,button")].map((e) => ({
            font: getComputedStyle(e).fontSize,
            radius: getComputedStyle(e).borderRadius,
            shadow: getComputedStyle(e).boxShadow,
          })),
          cells: [...el.querySelectorAll("td")]
            .filter((e) => e.getBoundingClientRect().width > 0)
            .map((e) => ({ client: e.clientHeight, scroll: e.scrollHeight })),
        }));
        assert.equal(c.overflow, false);
        assert.equal(c.body, "16px");
        assert.equal(c.direction, locale === "ar" ? "rtl" : "ltr");
        assert.match(c.font, /IBM Plex/);
        assert.ok(c.headings.every((v) => v === "20px"));
        assert.ok(
          c.inputs.every(
            (i) =>
              i.font === "16px" && i.radius === "0px" && i.shadow === "none",
          ),
          JSON.stringify(c.inputs),
        );
        if (width === 390)
          assert.ok(
            c.cells.every((x) => x.scroll <= x.client + 1),
            JSON.stringify(c.cells),
          );
        await page.screenshot({
          path: `${out}/${locale}-${width}-${view}.png`,
          fullPage: true,
        });
      }
      variants.push({ locale, width, views: 4 });
      console.log(JSON.stringify({ locale, width, views: 4 }));
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
