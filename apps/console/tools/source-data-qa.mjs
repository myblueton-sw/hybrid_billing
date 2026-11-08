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
const base = args.url || "http://127.0.0.1:8797/";
const out = "../../output/validation/HB-93";
await mkdir(out, { recursive: true });
const errors = [],
  renders = [];
page.on("pageerror", (e) => errors.push(e.message));
try {
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(base + "#/source-data?lang=" + lang + "&service=local");
      await page.locator(".source-release-list a").first().click();
      await page.locator("#source-page-title").waitFor();
      const sourceUrl = page.url();
      const row = page.locator(
        width < 640 ? "#source-mobile-1" : "#source-row-1",
      );
      await row.click();
      await page.locator("#source-detail").waitFor();
      assert.equal(
        await page.evaluate(() => document.activeElement?.id),
        "source-detail",
      );
      await page.goBack();
      await page.waitForFunction(
        (id) => document.activeElement?.id === id,
        width < 640 ? "source-mobile-1" : "source-row-1",
      );
      await page.goForward();
      await page.locator("#source-detail").waitFor();
      if (lang === "en") {
        const before = page.url();
        await page
          .getByRole("button", { name: "Confirm screen review", exact: true })
          .click();
        await page.locator("dialog[open]").waitFor();
        await page.keyboard.press("Escape");
        await page.locator("dialog[open]").waitFor({ state: "hidden" });
        assert.equal(page.url(), before);
        await page.locator("#source-detail").focus();
      }
      await page.keyboard.press("Escape");
      await page.waitForFunction(
        (id) => document.activeElement?.id === id,
        width < 640 ? "source-mobile-1" : "source-row-1",
      );
      const metrics = await page.evaluate(() => ({
        scroll: document.documentElement.scrollWidth,
        dir: document.documentElement.dir,
        logo: !!document.querySelector(".brand-logo")?.naturalWidth,
      }));
      assert.ok(metrics.scroll <= width, JSON.stringify(metrics));
      assert.equal(metrics.dir, lang === "ar" ? "rtl" : "ltr");
      assert.ok(metrics.logo);
      await page.screenshot({
        path: `${out}/${lang}-${width}.png`,
        fullPage: true,
      });
      renders.push({ lang, width, ...metrics });
      if (lang === "en" && width === 1440) {
        await page.route("**/api/local-source-data/pages/**", (route) =>
          route.fulfill({
            status: 403,
            contentType: "application/json",
            body: JSON.stringify({ error: "scope_denied" }),
          }),
        );
        await page.reload();
        await page.locator('.source-data [role="alert"]').waitFor();
        assert.equal(await page.locator(".source-data tbody tr").count(), 0);
        assert.equal(await page.locator(".source-scope").count(), 0);
        await page.unroute("**/api/local-source-data/pages/**");
        await page.goto(sourceUrl);
        await page.reload();
        await page.locator("#source-page-title").waitFor();
        await page.route("**/api/local-source-data/pages/**", async (route) => {
          const response = await route.fetch();
          const body = await response.json();
          delete body.candidate.blockers;
          await route.fulfill({ response, json: body });
        });
        await page.reload();
        await page.locator('.source-data [role="alert"]').waitFor();
        assert.equal(await page.locator(".source-data tbody tr").count(), 0);
        await page.unroute("**/api/local-source-data/pages/**");
      }
    }
  await page.route("**/api/local-source-data/candidates", route => route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({candidates:[],more:false,blockers:[]})}));
  await page.goto(base + "#/source-data?lang=en&service=local");
  await page.getByText("No received input.", {exact:true}).waitFor();
  assert.equal(await page.locator(".source-release-list a").count(),0);
  await page.unroute("**/api/local-source-data/candidates");
  assert.deepEqual(errors, []);
  await writeFile(
    `${out}/report.json`,
    JSON.stringify(
      {
        status: "PASS",
        renders,
        checks: [
          "real rows",
          "detail focus",
          "Back/Forward",
          "Escape return",
          "denial clearing",
          "malformed response",
          "EN/KO/AR overflow",
        ],
      },
      null,
      2,
    ),
  );
  console.log(JSON.stringify({ status: "PASS", renders: renders.length }));
} finally {
  await browser.close();
}
