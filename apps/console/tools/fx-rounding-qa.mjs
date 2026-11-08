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
  out = args.output || "qa-output/fx-rounding",
  base = "focus-provider=VMware&focus-period=2024-09";
await mkdir(out, { recursive: true });
const root = () => page.locator("[data-fx-state]");
const loaded = () => page.locator("[data-fx-state=loaded]").waitFor();
const selected = (id) =>
  page.waitForFunction(
    (id) =>
      document
        .querySelector("[data-fx-selected-row]")
        ?.getAttribute("data-fx-selected-row") === id,
    id,
  );
const tab = (i) => page.locator(".planning-tabs button").nth(i).click();
const fill = async () => {
  for (const [id, v] of Object.entries({
    "fx-name": "Synthetic FX QA",
    "fx-rate": "900",
    "fx-quote-unit": "100",
    "fx-provider": "Synthetic manual source",
    "fx-reference": "Synthetic quote reference",
    "fx-quote-date": "2024-09-01",
    "fx-review-date": "2024-09-02",
    "fx-max-age": "1",
    "fx-markup": "10",
    "fx-fee": "0",
    "fx-quantum": "0.05",
  }))
    await page.locator("#" + id).fill(v);
  await page.locator("#fx-target").selectOption("KRW");
  await page.locator("#fx-treatment").selectOption("unconverted");
  await page.locator('#fx-mode input[value="half_even"]').check();
};
const run = async () => {
  await page.locator("#fx-preview").click();
  await page.locator("[data-fx-candidate]").first().waitFor();
};
const open = async () => {
  await page.goto(url + "#/exchange-rates?lang=en&" + base);
  await loaded();
};
let passed = false;
try {
  await open();
  for (const q of ["VMware", " VMware "]) {
    await page.locator("#fx-search").fill(q);
    await Promise.all([
      page.waitForResponse(
        (r) =>
          r.url().includes("/api/focus-overview?") &&
          r.request().method() === "GET",
      ),
      page.getByRole("button", { name: "Search", exact: true }).click(),
    ]);
    await loaded();
  }
  await page.locator("#fx-search").fill("");
  await Promise.all([
    page.waitForResponse((r) => r.url().includes("/api/focus-overview?")),
    page.getByRole("button", { name: "Search", exact: true }).click(),
  ]);
  await loaded();
  await page.locator(".b05-account-scope summary").click();
  const account = await page
    .locator("button[data-fx-account]")
    .first()
    .getAttribute("data-fx-account");
  const narrowed = page.waitForResponse(
    (r) =>
      r.url().includes("/api/focus-overview?") &&
      new URL(r.url()).searchParams.get("account") === account,
  );
  await page.locator("button[data-fx-account]").first().click();
  const narrowedData = await (await narrowed).json();
  await loaded();
  assert.ok(narrowedData.rows.every((r) => r.account === account));
  if (!(await page.locator("#fx-all-accounts").isVisible()))
    await page.locator(".b05-account-scope summary").click();
  await Promise.all([
    page.waitForResponse(
      (r) =>
        r.url().includes("/api/focus-overview?") &&
        new URL(r.url()).searchParams.get("account") === "",
    ),
    page.locator("#fx-all-accounts").click(),
  ]);
  await loaded();
  const ids = await page
    .locator("button[data-fx-row]")
    .evaluateAll((es) => es.map((e) => e.getAttribute("data-fx-row")));
  assert.equal(ids.length, 3);
  assert.equal(await page.locator("#fx-next").isDisabled(), true);
  await page.locator("button[data-fx-row]").first().click();
  await selected(ids[0]);
  await fill();
  await page.locator("#fx-quote-date").fill("2024-09-03");
  await page.locator("#fx-preview").click();
  await root().getByRole("alert").first().waitFor();
  assert.equal(await page.locator("[data-fx-candidate]").count(), 0);
  await page.locator("#fx-quote-date").fill("2024-09-01");
  await run();
  const source = await page.evaluate(async () => {
    const r = await fetch(
      "/api/focus-overview?provider=VMware&period=2024-09",
      { headers: { "X-HB-Local-Client": "1" } },
    );
    if (!r.ok) throw Error("source read failed");
    return r.json();
  });
  assert.equal(
    source.rows.find((r) => r.id === ids[0]).billed_cost,
    "750000000000000000",
  );
  assert.ok(
    (
      await page.locator('[data-fx-total="total"]').first().innerText()
    ).includes("KRW 7.40"),
  );
  await tab(2);
  await page.locator("#fx-baseline").click();
  await tab(0);
  await page.locator("#fx-rate").fill("1000");
  await page.locator("[data-fx-candidate]").waitFor({ state: "detached" });
  await run();
  assert.ok(
    (
      await page.locator('[data-fx-total="total"]').first().innerText()
    ).includes("KRW 8.25"),
  );
  await tab(2);
  await page.locator('[data-fx-comparable="true"]').waitFor();
  assert.ok((await root().innerText()).includes("0.85"));
  await tab(0);
  await page.locator("#fx-target").selectOption("EUR");
  await page.locator("[data-fx-candidate]").waitFor({ state: "detached" });
  await run();
  await tab(2);
  await page.locator('[data-fx-comparable="false"]').waitFor();
  await tab(0);
  await page.locator("#fx-target").selectOption("KRW");
  await page.locator("#fx-save").click();
  await root().getByRole("status").filter({ hasText: /saved/i }).waitFor();
  const payload = await page.evaluate(() =>
    Object.entries(localStorage).filter(([k]) =>
      k.startsWith("hb-b05-draft-v1:"),
    ),
  );
  assert.equal(payload.length, 1);
  const parsed = JSON.parse(payload[0][1]);
  assert.equal(parsed.draft.rate, "1000");
  assert.equal(Object.hasOwn(parsed, "billed_cost"), false);
  await page.reload();
  await selected(ids[0]);
  assert.equal(await page.locator("#fx-rate").inputValue(), "1000");
  assert.equal(await page.locator("[data-fx-candidate]").count(), 0);
  await page.locator("#fx-rate").fill("1100");
  await tab(0);
  page.once("dialog", (d) => d.dismiss());
  await page.locator("button[data-fx-row]").nth(1).click();
  await selected(ids[0]);
  assert.equal(await page.locator("#fx-rate").inputValue(), "1100");
  page.once("dialog", (d) => d.accept());
  await page.locator("button[data-fx-row]").nth(1).click();
  await selected(ids[1]);
  assert.equal(await page.locator("#fx-name").inputValue(), "");
  await fill();
  await page.evaluate(() => {
    Storage.prototype.setItem = function () {
      throw Error("blocked");
    };
  });
  await page.locator("#fx-save").click();
  await root()
    .getByRole("alert")
    .filter({ hasText: /sav|storage/i })
    .waitFor();
  assert.equal(await page.locator("#fx-rate").inputValue(), "900");
  page.once("dialog", (d) => d.accept());
  await page.reload();
  await selected(ids[1]);
  await fill();
  await run();
  await tab(2);
  await page.locator("#fx-baseline").click();
  page.once("dialog", (d) => d.accept());
  await page.route("**/api/focus-overview?**", (r) =>
    r.fulfill({
      status: 403,
      contentType: "application/json",
      body: JSON.stringify({ error: "permission_denied" }),
    }),
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await page.locator("[data-fx-state=error]").waitFor();
  assert.equal(
    await page.locator("[data-fx-candidate],[data-fx-baseline]").count(),
    0,
  );
  await page.unroute("**/api/focus-overview?**");
  await page
    .getByRole("button", { name: "Reset row and retry", exact: true })
    .click();
  await loaded();
  await page.evaluate((id) => {
    const original = crypto.subtle.digest.bind(crypto.subtle);
    crypto.subtle.digest = async (algo, bytes) => {
      if (new TextDecoder().decode(bytes).includes('"row":"' + id + '"')) {
        window.__fxDelayedHash = true;
        await new Promise((r) => setTimeout(r, 2200));
      }
      return original(algo, bytes);
    };
  }, ids[0]);
  for (const id of [ids[0], ids[1]]) {
    await page.evaluate(
      ({ base, id }) => {
        location.hash = "#/exchange-rates?lang=en&" + base + "&fx-row=" + id;
      },
      { base, id },
    );
    if (id === ids[0])
      await page.waitForFunction(() => window.__fxDelayedHash === true);
  }
  await selected(ids[1]);
  await page.waitForTimeout(2300);
  assert.equal(await page.locator("#fx-name").inputValue(), "");
  assert.equal(await root().getAttribute("data-fx-selected-row"), ids[1]);
  await page.locator("#fx-search").fill("HB105_NO_MATCH_SYNTHETIC");
  await Promise.all([
    page.waitForResponse((r) => r.url().includes("/api/focus-overview?")),
    page.getByRole("button", { name: "Search", exact: true }).click(),
  ]);
  await loaded();
  assert.equal(await page.locator("button[data-fx-account]").count(), 0);
  assert.equal(await page.locator("button[data-fx-row]").count(), 3);
  assert.equal(
    await page.locator("[data-fx-candidate],[data-fx-baseline]").count(),
    0,
  );
  console.log(
    "PASS actual arithmetic, validation, target guard, draft, dirty, empty, denial/retry and late-hash cases",
  );
  const variants = [];
  for (const locale of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(
        url +
          "#/exchange-rates?lang=" +
          locale +
          "&" +
          base +
          "&fx-row=" +
          ids[0],
      );
      await selected(ids[0]);
      await page.evaluate(() => document.fonts.ready);
      await tab(0);
      await page.locator("#fx-rate").fill("1000");
      await run();
      await tab(2);
      await page.locator("#fx-baseline").click();
      await tab(0);
      await page.locator("#fx-rate").fill("1100");
      await run();
      for (const [i, view] of [
        "policies",
        "imports",
        "comparison",
        "approvals",
      ].entries()) {
        await tab(i);
        await page.locator('[data-fx-view="' + view + '"]').waitFor();
        const computed = await root().evaluate((el) => ({
          overflow: document.documentElement.scrollWidth > innerWidth,
          body: getComputedStyle(el).fontSize,
          direction: getComputedStyle(el).direction,
          headings: [...el.querySelectorAll("h2")].map(
            (e) => getComputedStyle(e).fontSize,
          ),
          fields: [
            ...el.querySelectorAll("input:not([type=radio]),select,textarea"),
          ].map((e) => getComputedStyle(e).fontSize),
          shapes: [...el.querySelectorAll("button,input,select,textarea")].map(
            (e) => [
              getComputedStyle(e).borderRadius,
              getComputedStyle(e).boxShadow,
            ],
          ),
        }));
        assert.equal(computed.overflow, false);
        assert.equal(computed.body, "16px");
        assert.equal(computed.direction, locale === "ar" ? "rtl" : "ltr");
        assert.ok(computed.headings.every((v) => v === "20px"));
        assert.ok(computed.fields.every((v) => v === "16px"));
        assert.ok(
          computed.shapes.every(([r, s]) => r === "0px" && s === "none"),
        );
        if (width === 390 && view === "policies") {
          const cells = await page
            .locator(".b05-source-table tbody td")
            .evaluateAll((es) =>
              es.map((e) => ({
                client: e.clientHeight,
                scroll: e.scrollHeight,
              })),
            );
          assert.ok(
            cells.every((c) => c.scroll <= c.client + 1),
            JSON.stringify(cells),
          );
          await page.locator("#fx-mode legend").waitFor();
          await page.locator("[data-fx-declaration]").waitFor();
        }
        if (view === "policies") {
          const group = await page
            .locator("#fx-mode")
            .evaluate((e) => ({
              legendBottom: e.querySelector("legend").getBoundingClientRect()
                .bottom,
              firstTop: e.querySelector("label").getBoundingClientRect().top,
            }));
          assert.ok(
            group.firstTop >= group.legendBottom,
            JSON.stringify(group),
          );
        }
        if (view === "comparison") {
          assert.equal(
            await page.locator('[data-fx-parameter-change="changed"]').count(),
            1,
          );
          assert.equal(
            await page
              .locator('[data-fx-parameter-change="unchanged"]')
              .count(),
            4,
          );
          assert.equal(
            await page
              .locator(".b05-trace")
              .evaluate((e) => getComputedStyle(e).direction),
            "ltr",
          );
        }
        await page.screenshot({
          path: `${out}/${locale}-${width}-${view}.png`,
          fullPage: true,
        });
      }
      page.once("dialog", (d) => d.accept());
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
