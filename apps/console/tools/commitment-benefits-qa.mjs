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
    await page
      .locator(
        "[data-b06-detail],button[data-b06-group],[data-source-value],[data-b06-row-cost],[data-b06-identity]",
      )
      .count(),
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
  const lines = async (locator) =>
    (await locator.innerText())
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean);
  const openAll = async () => {
    for (const disclosure of await page
      .locator("[data-b06-detail] details, details.b06-account-scope")
      .all())
      if (!(await disclosure.evaluate((e) => e.open)))
        await disclosure.locator("> summary").click();
  };
  const overflowing = () =>
    page
      .locator(
        "[data-b06-detail] :is(dd, dl, article, h4, .source-value), [data-b06-detail] .source-period > span, .b06-scope-identity > span, .b06-account-list button > span",
      )
      .evaluateAll((boxes) =>
        boxes
          .filter((box) => box.getClientRects().length)
          .filter((box) => {
            const outer = box.getBoundingClientRect();
            return [...box.children].some((child) => {
              const inner = child.getBoundingClientRect();
              return (
                inner.width > 0 &&
                (inner.left < outer.left - 1 || inner.right > outer.right + 1)
              );
            });
          })
          .map((box) => box.textContent.slice(0, 40)),
      );
  const openGroup = async (query) => {
    await page.goto("about:blank");
    await page.goto(url + "#/commitments?" + query);
    await loaded();
    const read = page.waitForResponse(
      (r) =>
        r.url().includes("/api/focus-commitments?") &&
        r.url().includes("group="),
    );
    await page.locator("button[data-b06-group]:visible").first().click();
    const body = await (await read).json();
    await selected(ids[0]);
    await page.locator("[data-b06-detail]").waitFor();
    await openAll();
    return body;
  };
  await page.setViewportSize({ width: 1440, height: 1000 });
  const real = await openGroup("lang=en&" + base);
  const realRow = real.selected.rows[0];
  const rowField = (index, name) =>
    page
      .locator(".b06-charge-records article")
      .nth(index)
      .locator(`[data-b06-row-field="${name}"] dd`);
  const rowCost = (index, name) =>
    page
      .locator(".b06-charge-records article")
      .nth(index)
      .locator(`[data-b06-row-cost="${name}"] dd`);
  assert.equal(await page.locator(".b06-charge-records article").count(), 3);
  assert.deepEqual(await lines(rowCost(0, "billed_cost")), [
    "USD 0.00",
    "Exact zero",
  ]);
  assert.deepEqual(await lines(rowCost(0, "effective_cost")), [
    "USD 0.00",
    "Exact zero",
  ]);
  assert.equal((await rowField(0, "id").innerText()).trim(), realRow.id);
  assert.deepEqual(await lines(rowField(0, "ordinal")), [realRow.ordinal]);
  assert.deepEqual(await lines(rowField(0, "batch")), [realRow.batch]);
  assert.deepEqual(await lines(rowField(0, "evidence_class")), [
    "Public sample",
  ]);
  assert.deepEqual(await lines(rowField(0, "charge_start")), [
    "2024-09-20 20:00:00 UTC",
  ]);
  assert.deepEqual(await lines(rowField(0, "billing_end")), [
    "2024-10-01 00:00:00 UTC",
  ]);
  assert.deepEqual(
    await lines(page.locator('[data-b06-envelope="charge_start"] dd')),
    ["Start 2024-09-20 20:00:00 UTC", "End 2024-09-26 13:00:00 UTC"],
  );
  assert.deepEqual(
    await page
      .locator("[data-b06-detail] [data-b06-identity] dt")
      .allInnerTexts(),
    ["Provider", "Billing account ID", "Subaccount ID", "Account scope ID"],
  );
  const detailText = await page
    .locator("[data-b06-detail]")
    .evaluate((e) => e.textContent);
  for (const superseded of [
    "Canonical",
    "Delivery batch",
    "interval",
    "Evidence classification",
    "Service unavailable",
    "Sub-account",
    "No source value",
  ])
    assert.ok(!detailText.includes(superseded), superseded);
  assert.match(
    await page
      .locator(".b06-group-table .b06-scope-identity")
      .first()
      .innerText(),
    /Provider: AWS[\s\S]*Billing account ID: 1234567890123[\s\S]*Subaccount ID: 18938484842/,
  );
  assert.match(
    await page.locator(".b06-account-list button").first().innerText(),
    /Billing account ID:[\s\S]*Provider:[\s\S]*Subaccount ID:/,
  );
  assert.match(
    await page
      .locator("[data-b06-detail] [data-b06-cost=billed_cost]")
      .innerText(),
    /Present \/ missing/,
  );
  await page.screenshot({ path: `${out}/en-detail-open.png`, fullPage: true });
  const degrade = (body, allMissing) => {
    const d = structuredClone(body),
      g = d.groups.find((group) => group.id === ids[0]),
      rows = d.selected.rows,
      total = d.identified_amounts.find((a) => a.currency === g.currency),
      holders = (basis) => [g[basis], g.statuses.used[basis], total[basis]],
      shift = (measure, delta, lost) => {
        measure.present -= lost;
        measure.missing += lost;
        measure.coefficient =
          measure.present === 0
            ? null
            : (BigInt(measure.coefficient) + delta).toString();
      };
    rows[0].service = null;
    rows[2].charge_start = rows[2].charge_start.replace("+00:00", "Z");
    for (const row of rows) row.sub_account = null;
    g.sub_account = null;
    for (const account of d.accounts)
      if (account.id === g.account) account.sub_account = null;
    if (allMissing) {
      for (const row of rows) {
        for (const measure of holders("effective_cost"))
          shift(measure, -BigInt(row.effective_cost), 1);
        row.effective_cost = null;
      }
    } else {
      for (const measure of holders("billed_cost"))
        shift(measure, -BigInt(rows[0].billed_cost), 1);
      rows[0].billed_cost = null;
      const negative = -2n * 10n ** 18n;
      for (const measure of holders("effective_cost"))
        shift(measure, negative - BigInt(rows[1].effective_cost), 0);
      rows[1].effective_cost = negative.toString();
    }
    return d;
  };
  let allMissing = false;
  await page.route("**/api/focus-commitments?**", async (route) => {
    if (
      !route
        .request()
        .url()
        .includes("group=" + ids[0])
    )
      return route.continue();
    const response = await route.fetch();
    await route.fulfill({
      response,
      body: JSON.stringify(degrade(await response.json(), allMissing)),
    });
  });
  for (const [locale, missing, absent, service] of [
    ["en", "Missing", "No values present", "Service: Missing"],
    ["ko", "누락", "값 없음", "서비스: 누락"],
    ["ar", "مفقود", "لا قيم موجودة", "الخدمة: مفقود"],
  ]) {
    allMissing = false;
    await openGroup(`lang=${locale}&${base}`);
    assert.deepEqual(await lines(rowCost(0, "billed_cost")), [missing]);
    assert.deepEqual(await lines(rowCost(1, "effective_cost")), ["USD -2.00"]);
    assert.match(
      await page
        .locator("[data-b06-detail] [data-b06-cost=billed_cost]")
        .innerText(),
      /2 \/ 1/,
    );
    assert.ok(
      (await page.locator("[data-b06-detail] .b06-metadata").allInnerTexts())
        .join("\n")
        .includes(absent),
    );
    const localized = await page
      .locator("[data-b06-detail]")
      .evaluate((e) => e.textContent);
    for (const superseded of [
      "공급자",
      "원천 값 없음",
      "서비스명 없음",
      "청구 계정",
    ])
      assert.ok(!localized.includes(superseded), superseded);
    assert.equal(
      (await page.locator("[data-b06-row-service]").first().innerText()).trim(),
      service,
    );
    assert.ok(
      (
        await page
          .locator("[data-b06-detail] [data-b06-cost=effective_cost]")
          .innerText()
      ).includes("USD -2.00"),
    );
    const sub = page.locator(
      '[data-b06-detail] [data-b06-identity="sub_account"] dd',
    );
    assert.deepEqual(await lines(sub), [missing]);
    assert.equal(
      await sub
        .locator("[data-source-value=missing]")
        .evaluate((node) => !!node.closest('bdi[dir="ltr"]')),
      false,
    );
    assert.ok((await rowField(2, "charge_start").innerText()).includes("Z"));
    assert.ok(
      (
        await page
          .locator(".b06-group-table .b06-scope-identity")
          .first()
          .innerText()
      ).includes(missing),
    );
    if (locale === "ar") {
      const edges = await page
        .locator(".b06-charge-records article")
        .first()
        .locator('[data-b06-row-field="id"]')
        .evaluate((item) => [
          item.querySelector("dt").getBoundingClientRect().right,
          item.querySelector("dd bdi").getBoundingClientRect().right,
        ]);
      assert.ok(Math.abs(edges[0] - edges[1]) <= 2, JSON.stringify(edges));
    }
    await page.screenshot({
      path: `${out}/${locale}-degraded-detail.png`,
      fullPage: true,
    });
    allMissing = true;
    await openGroup(`lang=${locale}&${base}`);
    assert.ok(
      (
        await page
          .locator("[data-b06-detail] [data-b06-cost=effective_cost]")
          .innerText()
      ).includes(absent),
    );
    assert.deepEqual(await lines(rowCost(2, "effective_cost")), [missing]);
  }
  for (const locale of ["ko", "ar"]) {
    allMissing = false;
    await page.setViewportSize({ width: 390, height: 1000 });
    await openGroup(`lang=${locale}&${base}`);
    assert.deepEqual(await overflowing(), []);
    assert.equal(
      await page.evaluate(
        () => document.documentElement.scrollWidth > innerWidth,
      ),
      false,
    );
    await page.screenshot({
      path: `${out}/${locale}-390-degraded-detail.png`,
      fullPage: true,
    });
  }
  await page.unroute("**/api/focus-commitments?**");
  for (const locale of ["en", "ko", "ar"])
    for (const width of [1440, 1280, 1279, 1024, 761, 760, 641, 640, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await openGroup(`lang=${locale}&${base}`);
      await page.evaluate(() => document.fonts.ready);
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth,
        ),
        false,
        `${locale} ${width} page overflow`,
      );
      assert.deepEqual(await overflowing(), [], `${locale} ${width}`);
      assert.ok(
        (await page.locator("[data-b06-detail] [data-source-value]").count()) >
          0,
      );
      if (locale === "en" || width === 390 || width === 1440)
        await page.screenshot({
          path: `${out}/${locale}-${width}-detail-open.png`,
          fullPage: true,
        });
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto(url + "#/commitments?lang=en&" + base);
  await loaded();
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
