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
const url = args.url || "http://127.0.0.1:8794/";
const out = "../../output/validation/HB-95";
await mkdir(out, { recursive: true });
const errors = [],
  renders = [];
page.on("pageerror", (e) => errors.push(e.message));
const rows = page.locator(".rec-select");
const action = () => page.locator("#rec-detail button.primary");
const open = async (lang = "en") => {
  await page.goto(url + "#/reconciliation?lang=" + lang + "&service=local");
  await rows.first().waitFor();
};
try {
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await open(lang);
      await rows.first().click();
      await action().click();
      await page
        .locator(".rec-comparison tbody tr")
        .first()
        .waitFor({ state: "attached" });
      assert.equal(await page.locator(".rec-comparison tbody tr").count(), 6);
      assert.equal(await page.locator(".rec-checks dd").count(), 7);
      const metrics = await page.evaluate(() => ({
        width: innerWidth,
        scroll: document.documentElement.scrollWidth,
        dir: document.documentElement.dir,
        fonts: document.fonts.status,
        logo: !!document.querySelector(".brand-logo")?.naturalWidth,
      }));
      assert.ok(metrics.scroll <= width, JSON.stringify(metrics));
      assert.equal(metrics.dir, lang === "ar" ? "rtl" : "ltr");
      assert.ok(metrics.logo);
      if (width === 390) {
        await page.locator(".rec-mobile-values article").first().waitFor();
        assert.equal(
          await page.locator(".rec-mobile-values article").count(),
          6,
        );
        assert.equal(
          await page
            .locator(".rec-mobile-values article")
            .first()
            .locator(".rec-money")
            .first()
            .innerText(),
          "0.200000",
        );

        const fit = await page
          .locator(".rec-candidates tbody tr")
          .first()
          .evaluate((row) =>
            [...row.querySelectorAll("td")].every((cell) =>
              [...cell.children].every(
                (child) =>
                  child.getBoundingClientRect().bottom <=
                  cell.getBoundingClientRect().bottom + 1,
              ),
            ),
          );
        assert.ok(
          fit,
          "Mobile cell children must fit without overlapping following fields",
        );
      }

      await page.evaluate(() => document.fonts.ready);
      await page.screenshot({
        path: out + `/S06-${lang}-${width}.png`,
        fullPage: true,
      });
      await page.locator("#rec-detail").scrollIntoViewIfNeeded();
      await page.screenshot({ path: out + `/S06-detail-${lang}-${width}.png` });
      renders.push({ lang, ...metrics });
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await open();
  await rows.first().focus();
  await page.keyboard.press("Enter");
  assert.equal(await rows.first().getAttribute("aria-pressed"), "true");
  await page.getByRole("link", { name: "Go to selected detail" }).click();
  assert.equal(
    await page.evaluate(() => document.activeElement.id),
    "rec-detail",
  );
  await action().click();
  await page
    .locator(".rec-comparison tbody tr")
    .first()
    .waitFor({ state: "attached" });
  await rows.nth(1).click();
  assert.equal(await page.locator(".rec-comparison").count(), 0);
  let release;
  const gate = new Promise((r) => (release = r));
  let arrived;
  const started = new Promise((r) => (arrived = r));
  await page.route(
    "**/api/local-reconciliation/candidates/*/revalidate",
    async (route) => {
      const response = await route.fetch();
      arrived();
      await gate;
      await route.fulfill({ response });
    },
  );
  await action().click();
  await started;
  await rows.first().click();
  release();
  await page.unrouteAll({ behavior: "wait" });
  assert.equal(await page.locator(".rec-comparison").count(), 0);
  for (const status of [403, 503, 409]) {
    await page.route(
      "**/api/local-reconciliation/candidates/*/revalidate",
      (route) =>
        route.fulfill({
          status,
          json: {
            error:
              status === 403
                ? "scope_denied"
                : status === 409
                  ? "evidence_changed"
                  : "service_unavailable",
            request_id: "synthetic-qa",
          },
        }),
    );
    await action().click();
    await page.locator(".rec-error").waitFor();
    assert.equal(await rows.count(), 0);
    assert.equal(await page.locator("#rec-detail").count(), 0);
    assert.equal(
      await page.getByText("synthetic-account-a", { exact: true }).count(),
      0,
    );
    await page.unrouteAll();
    await page.getByRole("button", { name: "Refresh candidates" }).click();
    await rows.first().waitFor();
    await rows.first().click();
  }
  await page.route("**/api/local-reconciliation/candidates", (route) =>
    route.fulfill({
      json: {
        candidates: [],
        more: false,
        blockers: [
          "independent_cross_validation_missing",
          "canonical_confirm_publish_unavailable",
        ],
      },
    }),
  );
  await page.getByRole("button", { name: "Refresh candidates" }).click();
  await page.getByText("No input candidates in this scope.").waitFor();
  assert.equal(await rows.count(), 0);
  await page.unrouteAll();
  await page.getByRole("button", { name: "Refresh candidates" }).click();
  await rows.first().waitFor();
  await rows.first().click();
  await action().click();
  await page.locator(".rec-comparison").waitFor();
  await page.reload();
  await rows.first().waitFor();
  assert.equal(await page.locator(".rec-comparison").count(), 0);
  assert.deepEqual(errors, []);
  await writeFile(
    out + "/browser-results.json",
    JSON.stringify(
      {
        renders,
        errors,
        scenarios: [
          "real revalidation",
          "six locale/width renders",
          "keyboard selection/detail focus",
          "selection invalidation",
          "late response invalidation",
          "403 revocation clears scope",
          "503 failure clears scope",
          "409 evidence change clears scope",
          "empty",
          "retry",
          "reload clears temporary comparison",
        ],
        browser: browser.version(),
      },
      null,
      2,
    ),
  );
  console.log(
    JSON.stringify({ renders: renders.length, scenarios: 11, errors }),
  );
} finally {
  await browser.close();
}
