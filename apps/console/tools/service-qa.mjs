import { createRequire } from "node:module";
import { mkdir, readFile } from "node:fs/promises";
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
  errors = [];
page.on("pageerror", (e) => errors.push(e.message));
const url = args.url || "http://127.0.0.1:8793/";
await mkdir("qa-output/service", { recursive: true });
try {
  await page.goto(url + "#/imports?lang=en&service=local");
  await page.getByText("synthetic-account-a", { exact: true }).waitFor();
  const revision = String(Date.now()),
    submit = page.getByRole("button", {
      name: "Receive and validate",
      exact: true,
    });
  await page.getByLabel("Source revision", { exact: true }).fill(revision);
  await page
    .locator("input[type=file]")
    .setInputFiles("../../tests/fixtures/focus/synthetic-1.3.csv");
  await submit.click();
  await page
    .getByText("Validation stored. Worker confirmation is still required.", {
      exact: true,
    })
    .waitFor();
  const row = page
    .locator(".live-imports tbody tr")
    .filter({ has: page.getByText(revision, { exact: true }) });
  await row.waitFor();
  assert.match(await row.innerText(), /verified \/ terminal/);
  await row.getByRole("button").click();
  await page.getByRole("heading", { name: "Persisted evidence" }).waitFor();
  await page.locator(".live-imports tbody tr").first().waitFor();
  await page.evaluate(()=>window.scrollTo({top:0,behavior:"instant"}));
  await page.screenshot({
    path: "qa-output/service/S03-selected-en.png",
    fullPage: true,
  });
  await page.reload();
  await page
    .locator(".live-imports tbody tr")
    .filter({ has: page.getByText(revision, { exact: true }) })
    .waitFor();
  await page.getByLabel("Source revision", { exact: true }).fill(revision);
  await page
    .locator("input[type=file]")
    .setInputFiles("../../tests/fixtures/focus/synthetic-1.3.csv");
  await submit.click();
  await page
    .getByText("Existing receipt returned; no duplicate import.", {
      exact: true,
    })
    .waitFor();
  const content = await readFile(
    "../../tests/fixtures/focus/synthetic-1.3.csv",
  );
  await page.locator("input[type=file]").setInputFiles({
    name: "conflict.csv",
    mimeType: "text/csv",
    buffer: Buffer.from(content.toString().replaceAll("0.300001", "0.300002")),
  });
  await submit.click();
  await page.getByRole("alert").waitFor();
  assert.match(await page.getByRole("alert").innerText(), /revision_conflict/);
  await page.locator(".live-imports tbody tr").first().waitFor();
  await page.evaluate(()=>window.scrollTo({top:0,behavior:"instant"}));
  await page.screenshot({
    path: "qa-output/service/S03-conflict-en.png",
    fullPage: true,
  });
  page.on("dialog", (d) => d.accept());
  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + "#/imports?lang=" + lang + "&service=local");
      await page.getByText("synthetic-account-a", { exact: true }).waitFor();
      await page.locator(".live-imports tbody tr").first().waitFor();
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth,
        ),
        false,
      );
      await page.locator(".live-imports tbody tr").first().waitFor();
  await page.evaluate(()=>window.scrollTo({top:0,behavior:"instant"}));
  await page.screenshot({
        path: `qa-output/service/S03-${lang}-${width}.png`,
        fullPage: true,
      });
    }
  assert.deepEqual(errors, []);
  console.log(
    "PASS: actual browser upload, persisted reload, duplicate, conflict and 6 locale/viewport renders",
  );
} finally {
  await browser.close();
}
