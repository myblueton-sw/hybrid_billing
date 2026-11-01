import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
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
  ),
  url = args.url || "http://127.0.0.1:8787/",
  out = resolve("qa-output/work");
await mkdir(out, { recursive: true });
const b = await chromium.launch({
    headless: true,
    ...(args.chromium ? { executablePath: args.chromium } : {}),
  }),
  context = await b.newContext({ viewport: { width: 1440, height: 1000 } }),
  p = await context.newPage(),
  checks = [],
  errors = [];
p.on("pageerror", (e) => errors.push(e.message));
const ok = (n) => checks.push(n);
const rows = (n) =>
  p.waitForFunction(
    (n) => document.querySelectorAll("tbody tr").length === n,
    n,
  );
try {
  await p.goto(url + "#operations");
  await rows(25);
  assert.match(await p.locator("h1").innerText(), /발행/);
  assert.equal(await p.locator("#design-directory-results").count(), 0);
  ok("explicit operations screen");
  await p.goto(url + "#operations?lang=en&provider=Azure&filter=ready");
  await rows(25);
  assert.deepEqual(await p.locator("thead th").allTextContents(), [
    "Selection",
    "ID",
    "Customer",
    "Status",
    "Responsible person",
    "Due date",
    "Reason",
  ]);
  ok("operations baseline columns");
  await p.selectOption("#status-filter", "missing");
  await rows(15);
  await p.locator("tbody a").first().click();
  await p.getByRole("tab", { name: "Evidence", exact: true }).click();
  await p
    .getByRole("link", { name: "Inspect source evidence", exact: true })
    .click();
  await rows(1);
  assert.match(await p.locator("tbody").innerText(), /sample-a-REC-0000001/);
  ok("operations customer detail source evidence");
  await p.goBack();
  await p.getByRole("tab", { name: "Overview", exact: true }).waitFor();
  await p.goBack();
  await rows(15);
  assert.equal(await p.inputValue("#status-filter"), "missing");
  ok("back restores work filter");
  await p.locator("tbody input").first().check();
  await p.locator("#review-selection").click();
  await p.locator("#workflow-review").waitFor();
  assert.equal(await p.locator("#workflow-review").isDisabled(), true);
  await p.selectOption("[name=reviewer]", "DEMO-PERSON-A");
  await p.selectOption("[name=issuer]", "DEMO-PERSON-A");
  for (const key of ["source", "recipient", "payment"])
    await p.locator(`[name=${key}]`).check();
  assert.equal(await p.locator("#workflow-review").isDisabled(), true);
  await p.selectOption("[name=issuer]", "DEMO-PERSON-B");
  await p.locator("#workflow-review").click();
  await p
    .getByText("Local review prepared. Financial actions remain blocked.", {
      exact: true,
    })
    .waitFor();
  assert.equal(await p.locator("#issue-approve").isDisabled(), true);
  assert.equal(await p.locator("#issue-issue").isDisabled(), true);
  ok("distinct people prerequisite local preparation only");
  await p.selectOption("[name=revision]", "DEMO-r2");
  await p.waitForFunction(
    () => document.querySelector("#issue-review-result").textContent === "",
  );
  await p.selectOption("[name=revision]", "DEMO-r1");
  assert.equal(await p.locator("#issue-review-result").innerText(), "");
  await p.locator("#workflow-review").click();
  await p.locator("[name=source]").uncheck();
  await p.waitForFunction(
    () => document.querySelector("#issue-review-result").textContent === "",
  );
  assert.equal(await p.locator("#workflow-review").isDisabled(), true);
  ok("prerequisite and revision changes revoke preparation");
  await p.locator("#choose-account").click();
  await p.getByRole("option", { name: /ACC-000002/ }).click();
  await p
    .getByText("No valid billing record selected", { exact: true })
    .waitFor();
  assert.equal(await p.locator("#workflow-review").count(), 0);
  ok("account change rejects old record");
  await p.goto(
    url + "#customer-detail?lang=en&scope=sample-b&case=sample-a-CASE-0001",
  );
  await p.getByText("No customer record selected", { exact: true }).waitFor();
  ok("cross-scope detail rejects reference");
  for (const route of ["customer-detail", "review-issue"]) {
    await p.goto(
      url + `#${route}?lang=en&case=sample-a-CASE-0001&period=2026-09`,
    );
    await p
      .getByText(
        route === "customer-detail"
          ? "No customer record selected"
          : "No valid billing record selected",
        { exact: true },
      )
      .waitFor();
  }
  ok("mismatched period rejects detail and preparation");

  await p.goto(url + "#operations?lang=en&mode=unconnected");
  await p.getByText("Billing service not connected", { exact: true }).waitFor();
  assert.equal(await p.locator("tbody tr").count(), 0);
  assert.equal(await p.locator("#record-search").isDisabled(), true);
  ok("explicit unconnected generates no rows and disables fixture search");
  await p.goto(url + "#core-index-close?lang=en");
  await rows(25);
  assert.ok(
    (await p.locator("thead th").allTextContents()).includes("Net amount"),
  );
  assert.match(await p.locator("tbody").innerText(), /Not available/);
  ok("close workspace keeps amounts unavailable");
  await p.goto(url + "#my-tasks?lang=en");
  await rows(25);
  await p.locator("#next").click();
  await rows(4);
  ok("my tasks own-person subset and last page");
  await p.goto(url + "#operations?lang=en");
  await rows(25);
  await p.locator(".scenario summary").click();
  await p.locator("#scenario-error").click();
  await p.locator("#retry-load").click();
  await rows(25);
  await p.locator("#scenario-empty").click();
  await p
    .getByText("No work matches the selected scope.", { exact: true })
    .waitFor();
  ok("work failure retry and empty");
  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390])
      for (const route of ["operations", "customer-detail", "review-issue"]) {
        await p.setViewportSize({ width, height: 1000 });
        await p.goto(
          url + `#${route}?lang=${lang}&scope=sample-b&case=sample-b-CASE-0001`,
        );
        if (route === "operations") await rows(25);
        else
          await p
            .locator(
              route === "customer-detail"
                ? "#screen-tab-overview"
                : "#workflow-review",
            )
            .waitFor();
        await p.evaluate(() => document.fonts.ready);
        assert.ok(
          await p.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth + 1,
          ),
        );
        await p.screenshot({
          path: resolve(out, `${route}-${lang}-${width}.png`),
          fullPage: true,
        });
        ok(`${route} ${lang} ${width} render`);
      }
  assert.deepEqual(errors, []);
  ok("no runtime errors");
  await writeFile(
    resolve(out, "report.json"),
    JSON.stringify({ checks, errors, browser: b.version() }, null, 2),
  );
  console.log(JSON.stringify({ passed: checks.length, out }));
} finally {
  await b.close();
}
