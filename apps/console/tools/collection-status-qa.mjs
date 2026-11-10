import { createRequire } from "node:module";
import { mkdir, writeFile, readFile } from "node:fs/promises";
import { resolve } from "node:path";
import assert from "node:assert/strict";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .flatMap((v, i, a) => (v.startsWith("--") ? [[v.slice(2), a[i + 1]]] : [])),
);
const { chromium } = createRequire(import.meta.url)(
  args["playwright-module"] || "playwright",
);
const origin = args.url || "http://127.0.0.1:8817/",
  out = resolve(args.output || "output/validation/HB-150");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const checks = [],
  captures = [],
  errors = [],
  mutations = [];
function hook(page) {
  page.setDefaultTimeout(30000);
  page.on("pageerror", (e) => errors.push(e.message));
  page.on("request", (r) => {
    if (r.method() !== "GET") mutations.push(r.method());
  });
}
const ready = (page) =>
  page.locator('.receipt-screen[data-read-phase="loaded"]').waitFor();
const loaded = (page) =>
  page.locator('.collection-status[data-summary-phase="loaded"]').waitFor();
const number = (text) => Number(text.replaceAll(",", ""));
const firstCount = async (page) =>
  number(await page.locator(".collection-facts dd").first().innerText());
const screenshot = async (page, name) => {
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({ path: resolve(out, name + ".png"), fullPage: true });
  captures.push(name + ".png");
};
const payload = (data, status = 200) => ({
  status,
  contentType: "application/json",
  body: JSON.stringify(data),
});
const live = await browser.newContext();
const page = await live.newPage();
hook(page);
try {
  const response = await page.request.get(
    origin + "api/focus-collection-status",
    { headers: { "X-HB-Local-Client": "1" } },
  );
  assert.equal(response.status(), 200);
  const source = await response.json();
  assert.equal(source.row_count, 1121);
  assert.equal(source.delivery_count, 3);
  assert.equal(Object.keys(source).length, 19);
  const receiptsResponse = await page.request.get(
    origin + "api/focus-receipts?view=imports",
    { headers: { "X-HB-Local-Client": "1" } },
  );
  assert.equal(receiptsResponse.status(), 200);
  const receipts = await receiptsResponse.json();
  assert.equal(receipts.total, 2);
  assert.equal(source.auth_scope, receipts.auth_scope);
  assert.equal(source.permission_revision, receipts.permission_revision);
  checks.push({
    scenario: "actual-protected-proxy-contract",
    status: "PASS",
    rows: 1121,
    deliveries: 3,
    receipts: 2,
  });
  for (const locale of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(origin + "#/imports?lang=" + locale);
      await ready(page);
      await loaded(page);
      assert.equal(await firstCount(page), 1121);
      assert.equal(
        number(await page.locator(".collection-facts dd").nth(1).innerText()),
        3,
      );
      assert.equal(await page.locator(".receipt-open:visible").count(), 2);
      const measures = await page.evaluate(() => {
        const root = document.querySelector(".receipt-screen"),
          summary = document.querySelector(".collection-status"),
          control = document.querySelector("#confirm-review");
        const probe = document.createElement("div");
        probe.style.border = "1px solid var(--hb-line-panel)";
        root.append(probe);
        const neutral = getComputedStyle(probe).borderTopColor;
        probe.remove();
        const rangeElement = document.querySelector(".receipt-result-status bdi");
        const first = document.createRange(), last = document.createRange();
        first.setStart(rangeElement.firstChild, 0);
        first.setEnd(rangeElement.firstChild, 1);
        last.setStart(rangeElement.lastChild, rangeElement.lastChild.textContent.length - 1);
        last.setEnd(rangeElement.lastChild, rangeElement.lastChild.textContent.length);
        return {
          rangeLtr: first.getBoundingClientRect().left < last.getBoundingClientRect().left,
          timestampNoWrap: getComputedStyle(document.querySelector(".collection-read-status time")).whiteSpace === "nowrap",
          numericHeaderEnd: getComputedStyle(document.querySelector(".receipt-table th:nth-child(4)")).textAlign === "end",
          overflow: document.documentElement.scrollWidth > innerWidth + 1,
          body: parseFloat(getComputedStyle(root).fontSize),
          summaryWidth: summary.getBoundingClientRect().width,
          rootWidth: root.getBoundingClientRect().width,
          review: control && {
            font: parseFloat(getComputedStyle(control).fontSize),
            height: control.getBoundingClientRect().height,
          },
          factsBorder: getComputedStyle(
            document.querySelector(".collection-facts"),
          ).borderTopColor,
          neutral,
          rtl: document.documentElement.dir,
          localFonts: document.fonts.check('16px "IBM Plex Sans"'),
        };
      });
      assert.equal(measures.overflow, false);
      assert.equal(measures.rangeLtr, true);
      assert.equal(measures.timestampNoWrap, true);
      assert.equal(measures.numericHeaderEnd, true);
      assert.ok(measures.body >= 16);
      assert.ok(Math.abs(measures.summaryWidth - measures.rootWidth) < 2);
      assert.equal(measures.factsBorder, measures.neutral);
      assert.ok(measures.review.font >= 16 && measures.review.height >= 44);
      if (locale === "ar") assert.equal(measures.rtl, "rtl");
      await screenshot(page, `actual-${locale}-${width}`);
      const trigger = page.locator(".receipt-open:visible").first();
      await trigger.click();
      await page.locator("#receipt-detail-title").waitFor();
      await page.waitForFunction(
        () => document.activeElement?.id === "receipt-detail-title",
      );
      assert.equal(
        await page
          .locator("#receipt-detail-title")
          .evaluate((e) => e === document.activeElement),
        true,
      );
      if (locale === "en" && width === 1440)
        await screenshot(page, "actual-selected-en-1440");
      await page.locator(".receipt-detail > header > button").click();
      await page.waitForFunction(() =>
        document.activeElement?.classList.contains("receipt-open"),
      );
      assert.equal(
        await trigger.evaluate((e) => e === document.activeElement),
        true,
      );
      checks.push({
        scenario: `actual-${locale}-${width}`,
        status: "PASS",
        ...measures,
      });
    }
  for (const [locale, width] of [
    ["en", 390],
    ["ar", 1440],
  ])
    for (const view of ["quarantine", "completeness", "extraction"]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(origin + `#/imports?lang=${locale}&view=${view}`);
      await ready(page);
      await loaded(page);
      assert.equal(await firstCount(page), 1121);
      assert.equal(await page.locator(".receipt-open:visible").count(), 0);
      checks.push({
        scenario: `actual-${view}-${locale}-${width}`,
        status: "PASS",
      });
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto(origin + "#/imports?lang=en");
  await loaded(page);
  await page.locator(".collection-status > header a").click();
  await page.locator('.collected-source[data-source-state="loaded"]').waitFor();
  assert.ok(page.url().includes("#/source-data?lang=en"));
  assert.equal(
    new URLSearchParams(page.url().split("?")[1]).has("focus-period"),
    false,
  );
  checks.push({ scenario: "actual-S04-unfiltered-handoff", status: "PASS" });
  const fixture = (rows = 7) => ({
    ...structuredClone(source),
    snapshot: "d".repeat(64),
    row_count: rows,
    delivery_count: rows ? 1 : 0,
    versions: rows ? ["1.0"] : [],
    provenance: {
      "official-public": rows,
      "official-public-subset": 0,
      "synthetic-contract-fixture": 0,
      unclassified: 0,
    },
    providers: rows ? [{ provider: "AWS", row_count: rows }] : [],
    periods: rows ? [{ ...source.periods[0], row_count: rows }] : [],
  });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 1000 },
  });
  const testPage = await context.newPage();
  hook(testPage);
  let metadataMode = "valid",
    sourceReads = 0;
  await testPage.route("**/api/focus-receipts?**", async (route) => {
    if (metadataMode === "unavailable")
      return route.fulfill(
        payload(
          { error: "collected_unavailable", request_id: "e".repeat(32) },
          503,
        ),
      );
    const q = new URL(route.request().url()).searchParams.get("q") || "";
    const data = structuredClone(receipts);
    const view =
      new URL(route.request().url()).searchParams.get("view") || "imports";
    if (view === "completeness") {
      data.view = "completeness";
      data.capabilities.completeness = "registered-expectations-only";
      data.records = [
        {
          ...data.records[0],
          kind: "coverage",
          connection: "synthetic-link",
          source_revision: null,
          format: null,
          declared_rows: null,
          file_bytes: null,
          file_digest: null,
          mapping_digest: null,
          received_at: null,
          complete_through: data.records[0].window_end,
          finalized: true,
          known_zero: false,
          published: true,
        },
      ];
    }
    data.selection = { q, after: 0 };
    data.records = data.records.filter((r) => r.id.startsWith(q));
    data.total = data.records.length;
    data.next_after = null;
    await route.fulfill(payload(data));
  });
  let mode = "valid",
    held,
    started;
  const holdStarted = () =>
    new Promise((r) => {
      started = r;
    });
  await testPage.route("**/api/focus-collection-status", async (route) => {
    sourceReads++;
    if (mode === "hold") {
      held = route;
      started?.();
      return;
    }
    if (mode === "unavailable")
      return route.fulfill(
        payload(
          { error: "collected_unavailable", request_id: "e".repeat(32) },
          503,
        ),
      );
    if (mode === "malformed")
      return route.fulfill(
        payload({ ...fixture(), source_snapshot: "e".repeat(64) }),
      );
    if (mode === "zero") return route.fulfill(payload(fixture(0)));
    return route.fulfill(payload(fixture()));
  });
  mode = "unavailable";
  await testPage.goto(origin + "#/imports?lang=en");
  await ready(testPage);
  await testPage
    .locator('.collection-status[data-summary-phase="error"]')
    .waitFor();
  assert.equal(await testPage.locator(".receipt-open:visible").count(), 2);
  assert.equal(await testPage.locator(".collection-facts").count(), 0);
  await screenshot(testPage, "synthetic-source-unavailable");
  mode = "valid";
  await testPage.locator(".collection-failure button").click();
  await loaded(testPage);
  assert.equal(await firstCount(testPage), 7);
  assert.equal(
    await testPage
      .locator(".collection-read-status")
      .evaluate((e) => e === document.activeElement),
    true,
  );
  checks.push({
    scenario: "synthetic-source-failure-preserves-receipts-and-retry",
    status: "PASS",
  });
  for (const variant of ["denied", "malformed-denied", "mismatch"]) {
    mode = "hold";
    const heldReady = holdStarted();
    await testPage.reload();
    await ready(testPage);
    await heldReady;
    await testPage.locator(".receipt-open:visible").first().click();
    await testPage.locator("#receipt-detail-title").waitFor();
    const reply =
      variant === "mismatch"
        ? payload({ ...fixture(), permission_revision: "f".repeat(64) })
        : payload(
            variant === "denied"
              ? { error: "scope_denied", request_id: "f".repeat(32) }
              : { unexpected: true },
            403,
          );
    await held.fulfill(reply);
    await testPage
      .locator('.receipt-screen[data-read-phase="error"]')
      .waitFor();
    assert.equal(await testPage.locator(".receipt-workspace").count(), 0);
    assert.equal(await testPage.locator(".collection-facts").count(), 0);
    assert.equal(await testPage.locator(".receipt-detail").count(), 0);
    if (variant === "denied")
      await screenshot(testPage, "synthetic-authority-cleared");
    checks.push({
      scenario: "synthetic-authority-clears-all-" + variant,
      status: "PASS",
    });
  }
  mode = "malformed";
  await testPage.reload();
  await ready(testPage);
  await testPage
    .locator('.collection-status[data-summary-phase="error"]')
    .waitFor();
  assert.equal(await testPage.locator(".collection-facts").count(), 0);
  assert.equal(await testPage.locator(".receipt-open:visible").count(), 2);
  checks.push({ scenario: "synthetic-closed-schema-failure", status: "PASS" });
  mode = "zero";
  await testPage.reload();
  await loaded(testPage);
  assert.equal(await firstCount(testPage), 0);
  assert.equal(await testPage.locator(".collection-breakdowns").count(), 0);
  await screenshot(testPage, "synthetic-exact-zero");
  checks.push({
    scenario: "synthetic-exact-zero-not-unknown-completeness",
    status: "PASS",
  });
  mode = "hold";
  const oldReady = holdStarted();
  await testPage.reload();
  await ready(testPage);
  await oldReady;
  const old = held;
  mode = "valid";
  await testPage.locator("#receipt-search").fill("abc");
  await testPage.locator('.receipt-search button[type="submit"]').click();
  await loaded(testPage);
  assert.equal(await firstCount(testPage), 7);
  assert.equal(await testPage.locator(".receipt-open:visible").count(), 0);
  await old.fulfill(payload(fixture(5))).catch(() => {});
  await testPage.evaluate(
    () =>
      new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))),
  );
  assert.equal(await firstCount(testPage), 7);
  checks.push({
    scenario: "synthetic-late-old-read-and-receipt-filter-independence",
    status: "PASS",
  });
  mode = "valid";
  await testPage.setViewportSize({ width: 390, height: 1000 });
  await testPage.goto(origin + "#/imports?lang=en&view=completeness");
  await ready(testPage);
  await loaded(testPage);
  assert.equal(await testPage.locator(".receipt-card").count(), 1);
  assert.ok(
    (await testPage.locator(".receipt-card").innerText()).includes(
      "Whole connection/window zero is not attested",
    ),
  );
  checks.push({
    scenario: "synthetic-mobile-coverage-attestation-meaning",
    status: "PASS",
  });
  metadataMode = "unavailable";
  const beforeReads = sourceReads;
  await testPage.goto(origin + "#/imports?lang=en");
  await testPage.locator('.receipt-screen[data-read-phase="error"]').waitFor();
  assert.equal(await testPage.locator(".collection-status").count(), 0);
  assert.equal(sourceReads, beforeReads);
  checks.push({
    scenario: "synthetic-metadata-failure-prevents-dependent-read",
    status: "PASS",
  });
  await context.close();
  assert.deepEqual(errors, []);
  assert.deepEqual(mutations, []);
  const manifest = JSON.parse(
    await readFile(
      new URL("../dist/build-manifest.json", import.meta.url),
      "utf8",
    ),
  );
  await writeFile(
    resolve(out, "report.json"),
    JSON.stringify(
      {
        status: "PASS",
        revision: manifest.revision,
        checks,
        captures,
        pageErrors: errors,
        nonGetRequests: mutations,
        ownerAcceptanceCreated: false,
      },
      null,
      2,
    ) + "\n",
  );
  console.log(
    JSON.stringify({
      status: "PASS",
      checks: checks.length,
      captures: captures.length,
      revision: manifest.revision,
    }),
  );
} finally {
  await browser.close();
}
