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
let readerTurn = Promise.resolve(),
  serializedReads = 0,
  slowestReadMs = 0;
await page.route("**/api/focus-*", (route) => {
  serializedReads++;
  readerTurn = readerTurn.then(async () => {
    try {
      const started = Date.now(),
        response = await route.fetch({ timeout: 150000 });
      slowestReadMs = Math.max(slowestReadMs, Date.now() - started);
      await route.fulfill({ response });
    } catch {
      await route.abort().catch(() => {});
    }
  });
  return readerTurn;
});
page.setDefaultTimeout(150000);
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
const identity = (name) =>
  page.locator(`.infra-table [data-identity="${name}"]`);
const texts = async (locator) =>
  (await locator.allInnerTexts()).map((value) => value.trim());
const plain = (value) => value.replace(/[\u2066\u2069]/g, "");
const amountsContained = (maxHeight) =>
  page.locator(".infra-table [data-basis] bdi").evaluateAll(
    (amounts, limit) =>
      amounts.length > 0 &&
      amounts.every((amount) => {
        const box = amount.getBoundingClientRect(),
          cell = amount.closest("td").getBoundingClientRect();
        return (
          box.left >= cell.left - 1 &&
          box.right <= cell.right + 1 &&
          box.height <= limit
        );
      }),
    maxHeight,
  );
const costLine = (row, basis) =>
  page
    .locator(".infra-table tbody tr[data-row]")
    .nth(row)
    .locator(`[data-basis="${basis}"]`);
let passed = false;
try {
  await open();
  assert.equal(await pick().count(), 3);
  assert.equal(await page.locator(".infra-table details").count(), 0);
  assert.deepEqual(await texts(identity("ordinal")), ["1000", "1001", "1002"]);
  assert.deepEqual(await texts(identity("sub_account")), [
    "synthetic-a",
    "synthetic-a",
    "synthetic-b",
  ]);
  assert.deepEqual(await texts(identity("provenance")), [
    "Synthetic test data",
    "Synthetic test data",
    "Synthetic test data",
  ]);
  assert.equal(await identity("batch").count(), 0);
  const rowIds = await page
    .locator(".infra-table tbody tr[data-row]")
    .evaluateAll((rows) => rows.map((row) => row.dataset.row));
  assert.deepEqual(await texts(identity("id")), rowIds);
  assert.equal(new Set(rowIds).size, 3);
  for (const cell of await identity("id").all())
    assert.ok(await cell.isVisible());
  assert.deepEqual(
    await texts(page.locator('.infra-table [data-basis="billed_cost"] bdi')),
    ["USD 0.75", "USD 0.25", "USD 1.50"],
  );
  assert.deepEqual(
    await texts(page.locator('.infra-table [data-basis="effective_cost"] bdi')),
    ["USD 0.75", "USD 0.25", "USD 1.50"],
  );
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
  assert.equal(
    await page
      .locator(".infra-table")
      .evaluate((region) => region.scrollWidth > region.clientWidth),
    false,
  );
  assert.deepEqual(
    (await targets.nth(1).locator("option").allInnerTexts()).map(plain),
    [
      "No inclusion target",
      "Ordinal 1000 · Subaccount synthetic-a · Row ID " +
        rowIds[0].slice(0, 8) +
        " · Billed cost: USD 0.75",
    ],
  );
  assert.ok((await targets.nth(1).getAttribute("aria-label")).includes("1001"));
  assert.deepEqual(await texts(identity("ordinal")), ["1000", "1001"]);
  assert.deepEqual(await texts(identity("id")), rowIds.slice(0, 2));
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
  assert.equal(
    plain(await page.locator("[data-inclusion-amount]").innerText()),
    "Billed cost: USD 0.75",
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
  for (const absent of ["synthetic-", "USD", "ordinal", "Ordinal"])
    assert.ok(!stored[0][1].includes(absent));
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
  assert.equal(await page.locator("[data-identity]").count(), 0);
  assert.equal(await page.locator("[data-basis]").count(), 0);
  assert.equal(await page.locator(".infra-declare").count(), 0);
  await page.unroute("**/api/focus-overview?**");
  await open();
  await page
    .getByRole("button", { name: "Included items by product", exact: true })
    .click();
  await page.locator("[data-infra-view=products]").waitFor();
  assert.equal(
    plain(
      await page
        .locator('[data-relation="target"] .infra-relation-identity')
        .innerText(),
    ),
    "Ordinal 1000 · Subaccount synthetic-a · Row ID " + rowIds[0].slice(0, 8),
  );
  assert.equal(
    plain(
      await page
        .locator('[data-relation="included"] .infra-relation-identity')
        .innerText(),
    ),
    "Ordinal 1001 · Subaccount synthetic-a · Row ID " + rowIds[1].slice(0, 8),
  );
  assert.equal(
    (
      await page.locator('[data-relation="target"] .infra-code').innerText()
    ).trim(),
    rowIds[0],
  );
  await page.screenshot({ path: `${out}/en-products.png`, fullPage: true });
  for (const name of ["Approval history", "Composition summary"]) {
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
  page.on("dialog", (dialog) => dialog.accept().catch(() => {}));
  const visit = async (query) => {
    await page.goto(url + "#/cost-pools?" + query);
    await loaded();
  };
  await visit("lang=en&focus-provider=Oracle");
  assert.equal(await pick().count(), 7);
  assert.equal(await identity("batch").count(), 7);
  assert.ok(new Set(await texts(identity("batch"))).size > 1);
  assert.equal(
    await costLine(0, "billed_cost").getAttribute("data-value"),
    "value",
  );
  assert.equal(
    (await costLine(0, "billed_cost").locator("bdi").innerText()).trim(),
    "USD 0.24",
  );
  assert.equal(
    await costLine(0, "effective_cost").getAttribute("data-value"),
    "zero",
  );
  assert.deepEqual(
    (await costLine(0, "effective_cost").innerText())
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean),
    ["Effective cost", "USD 0.00", "Exact zero"],
  );
  await page.screenshot({ path: `${out}/en-oracle-zero.png`, fullPage: true });
  for (const [locale, zero] of [
    ["ko", "정확한 0"],
    ["ar", "صفر دقيق"],
  ]) {
    await visit(`lang=${locale}&focus-provider=Oracle`);
    assert.equal(
      (
        await costLine(0, "effective_cost").locator(".infra-state").innerText()
      ).trim(),
      zero,
    );
  }
  const awsSnapshot = await page.evaluate(async () => {
    const response = await fetch("/api/focus-overview?provider=AWS", {
      headers: { "X-HB-Local-Client": "1" },
      cache: "no-store",
    });
    if (!response.ok) throw Error("AWS source unavailable");
    return (await response.json()).snapshot;
  });
  await visit(
    `lang=en&focus-provider=AWS&focus-after=450&focus-snapshot=${awsSnapshot}`,
  );
  assert.equal(
    (await costLine(6, "billed_cost").locator("bdi").innerText()).trim(),
    "USD -2.6137",
  );
  assert.equal(
    (await costLine(6, "effective_cost").locator("bdi").innerText()).trim(),
    "USD -3.00",
  );
  assert.equal(
    await costLine(6, "billed_cost").locator(".infra-state").count(),
    0,
  );
  await visit("lang=en&focus-provider=Alibaba%20Cloud");
  assert.equal(
    (await costLine(1, "billed_cost").locator("bdi").innerText()).trim(),
    "CNY 8.80",
  );
  await pick().nth(0).check();
  await pick().nth(1).check();
  assert.equal(await page.locator(".infra-currency").count(), 2);
  await visit("lang=en&focus-provider=VMware&focus-period=2024-09");
  const degraded = structuredClone(replay.billed);
  degraded.rows[0].billed_cost = null;
  degraded.rows[1].sub_account = null;
  degraded.rows[1].service = null;
  degraded.rows[1].evidence_class = "unrecognized-class";
  degraded.rows[1].charge_start = "2024-09-01T09:00:00+09:00";
  degraded.rows[1].effective_cost = null;
  degraded.rows[1].billed_cost = "0";
  degraded.rows[2].batch = "00000000-0000-4000-8000-000000000000";
  degraded.rows[2].ordinal = degraded.rows[1].ordinal;
  degraded.rows[2].sub_account = null;
  degraded.rows[2].effective_cost = "123456789012000000000000000001";
  degraded.rows[2].billing_account = "ocid1.tenancy.oc1..".padEnd(120, "x");
  await page.route("**/api/focus-overview?**", (route) =>
    route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(degraded),
    }),
  );
  for (const [locale, missing, absent, provenance, uninterpreted] of [
    [
      "en",
      "Missing",
      "No values present",
      "Unclassified",
      "Time format not interpreted",
    ],
    ["ko", "누락", "값 없음", "출처 미분류", "시각 형식을 해석하지 못했습니다"],
    ["ar", "مفقود", "لا قيم موجودة", "غير مصنف", "تعذّر تفسير صيغة الوقت"],
  ]) {
    await visit(
      `lang=${locale}&focus-provider=VMware&focus-period=2024-09&focus-basis=billed_cost`,
    );
    assert.equal(await pick().count(), 3);
    assert.equal(
      await costLine(0, "billed_cost").getAttribute("data-value"),
      "missing",
    );
    const missingLine = await costLine(0, "billed_cost").innerText();
    assert.equal(
      missingLine
        .split("\n")
        .map((line) => line.trim())
        .filter(Boolean)[1],
      missing,
    );
    assert.ok(!/USD|—|\d/.test(missingLine));
    assert.equal(
      await costLine(1, "effective_cost").getAttribute("data-value"),
      "missing",
    );
    assert.ok(!/USD|—/.test(await costLine(1, "effective_cost").innerText()));
    assert.equal(
      await costLine(1, "billed_cost").getAttribute("data-value"),
      "zero",
    );
    assert.equal((await texts(identity("sub_account")))[1], missing);
    assert.equal((await texts(identity("provenance")))[1], provenance);
    assert.ok(
      (
        await page.locator(".infra-table tbody tr[data-row]").nth(1).innerText()
      ).includes(uninterpreted),
    );
    assert.equal(await identity("batch").count(), 3);
    assert.equal(
      (await costLine(2, "effective_cost").locator("bdi").innerText()).replace(
        /\s+/g,
        " ",
      ),
      "USD 123,456,789,012.000000000000000001",
    );
    assert.ok(await amountsContained(200));
    const ordinals = await texts(identity("ordinal"));
    assert.equal(ordinals[1], ordinals[2]);
    assert.equal(
      await page.evaluate(
        () => document.documentElement.scrollWidth > innerWidth,
      ),
      false,
    );
    await pick().nth(0).check();
    const gross = await page.locator('[data-total="gross"]').innerText();
    assert.ok(gross.includes(absent));
    assert.ok(!/USD|—/.test(gross));
    await pick().nth(1).check();
    await pick().nth(2).check();
    await page.locator(".infra-items + button").click();
    await page.locator('[data-infra-view="costs"]').waitFor();
    const options = (
      await page
        .locator(".infra-declare-field select")
        .first()
        .locator("option")
        .allInnerTexts()
    ).map(plain);
    assert.equal(new Set(options).size, options.length);
    assert.ok(options.every((option) => !option.includes("—")));
    assert.ok(
      (
        await page.locator(".infra-table tbody tr[data-row]").nth(1).innerText()
      ).includes("2024-09-01T09:00:00+09:00"),
    );
    if (locale === "en") {
      assert.ok(options[1].includes("Billing account "));
      assert.ok(options[1].includes("Billed cost: USD 0.00 (Exact zero)"));
      assert.ok(options[2].includes("ocid1.tenanc…xxxxxxxxxxxx"));
      const firstOptions = (
        await page
          .locator(".infra-declare-field select")
          .nth(1)
          .locator("option")
          .allInnerTexts()
      ).map(plain);
      assert.ok(
        firstOptions[1].endsWith(
          "Billed cost: Missing (Different currency or charge period)",
        ),
      );
      await page.screenshot({
        path: `${out}/en-degraded-edit.png`,
        fullPage: true,
      });
    }
  }
  await page.unroute("**/api/focus-overview?**");
  for (const width of [1320, 1024]) {
    await page.setViewportSize({ width, height: 1000 });
    await visit("lang=en&focus-provider=Oracle");
    await page.evaluate(() => document.fonts.ready);
    assert.equal(
      await page.evaluate(
        () => document.documentElement.scrollWidth > innerWidth,
      ),
      false,
    );
    assert.equal(
      await page
        .locator(".infra-table")
        .evaluate((region) => region.scrollWidth > region.clientWidth),
      false,
    );
    assert.ok(await amountsContained(30));
    await page.screenshot({
      path: `${out}/en-${width}-oracle.png`,
      fullPage: true,
    });
    if (width === 1024) {
      await pick().nth(0).check();
      await page.locator(".infra-items + button").click();
      await page.locator('[data-infra-view="costs"]').waitFor();
      assert.ok(
        await page
          .locator(".infra-select label, .infra-declare :is(select, input)")
          .evaluateAll((nodes) =>
            nodes.every((node) => node.getBoundingClientRect().height >= 44),
          ),
      );
      await page.screenshot({
        path: `${out}/en-1024-oracle-edit.png`,
        fullPage: true,
      });
    }
  }
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
      assert.equal(
        await page
          .locator(".infrastructure-cost")
          .evaluate((root) => getComputedStyle(root).fontSize),
        "16px",
      );
      assert.ok(
        await identity("id").evaluateAll((cells) =>
          cells.every((cell) => {
            const box = cell.closest("td").getBoundingClientRect();
            return [...cell.querySelectorAll(".infra-token")].every((token) => {
              const part = token.getBoundingClientRect();
              return part.left >= box.left - 1 && part.right <= box.right + 1;
            });
          }),
        ),
      );
      assert.ok(
        await page
          .locator(
            '.infra-table [data-basis] bdi, .infra-table [data-identity="id"] bdi',
          )
          .evaluateAll((nodes) =>
            nodes.every((node) => getComputedStyle(node).direction === "ltr"),
          ),
      );
      await page.screenshot({
        path: `${out}/${locale}-${width}.png`,
        fullPage: true,
      });
      await pick().nth(0).check();
      await pick().nth(1).check();
      await page.locator(".infra-items + button").click();
      await page.locator('[data-infra-view="costs"]').waitFor();
      const declared = page.locator(".infra-declare-field select");
      await declared
        .nth(1)
        .selectOption(
          await declared.nth(1).locator("option").nth(1).getAttribute("value"),
        );
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth,
        ),
        false,
      );
      assert.equal(
        await page
          .locator(".infra-table")
          .evaluate((region) => region.scrollWidth > region.clientWidth),
        false,
      );
      if (width === 1440) {
        await page.locator(".infra-declare-row").first().hover();
        assert.deepEqual(
          await page
            .locator(".infra-table tbody")
            .first()
            .evaluate((group) => {
              const style = getComputedStyle(group),
                cell = getComputedStyle(group.querySelector("td"));
              return [style.borderBottomWidth, cell.backgroundColor];
            }),
          ["1px", "rgba(0, 0, 0, 0)"],
        );
      }
      const edges = await page
        .locator(
          ".infra-table, .infra-table :is(tbody, tr, th, td, select, input, label), .infra-declare, .infra-identity",
        )
        .evaluateAll((nodes) =>
          nodes.flatMap((node) => {
            const style = getComputedStyle(node);
            const colors = ["Top", "Right", "Bottom", "Left"]
              .filter((side) => parseFloat(style[`border${side}Width`]) > 0)
              .map((side) => style[`border${side}Color`]);
            const spread = colors.map((color) => {
              const [r, g, b] = color.match(/[\d.]+/g).map(Number);
              return Math.max(r, g, b) - Math.min(r, g, b);
            });
            return [
              {
                radius: style.borderTopLeftRadius,
                shadow: style.boxShadow,
                spread: Math.max(0, ...spread),
              },
            ];
          }),
        );
      assert.ok(
        edges.every(
          (edge) =>
            edge.radius === "0px" &&
            edge.shadow === "none" &&
            edge.spread <= 32,
        ),
      );
      if (width === 390)
        assert.ok(
          await page
            .locator(".infra-select label, .infra-declare :is(select, input)")
            .evaluateAll((nodes) =>
              nodes.every((node) => node.getBoundingClientRect().height >= 44),
            ),
        );
      await page.screenshot({
        path: `${out}/${locale}-${width}-edit.png`,
        fullPage: true,
      });
      variants.push({ locale, width });
    }
  assert.deepEqual(faults, []);
  passed = true;
  await writeFile(
    `${out}/result.json`,
    JSON.stringify(
      { passed, variants, faults, serializedReads, slowestReadMs },
      null,
      2,
    ),
  );
} catch (e) {
  await page.screenshot({ path: `${out}/failure.png`, fullPage: true });
  throw e;
} finally {
  await browser.close();
}
console.log(JSON.stringify({ passed }));
