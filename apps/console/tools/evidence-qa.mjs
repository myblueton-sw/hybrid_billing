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
const base = args.url || "http://127.0.0.1:8798/";
const out = "../../output/validation/HB-94";
await mkdir(out, { recursive: true });
const errors = [],
  external = [],
  renders = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("request", (r) => {
  if (!r.url().startsWith(base)) external.push(r.url());
});
let evidenceUrl;
const focus = async (id) =>
  page.waitForFunction((id) => document.activeElement?.id === id, id);
try {
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(base + "#/source-data?lang=" + lang + "&service=local");
      await page.locator(".source-release-list a").first().click();
      await page
        .locator(width < 640 ? "#source-mobile-1" : "#source-row-1")
        .click();
      await page.locator("#source-evidence-link").click();
      await page.locator(".evidence-summary").waitFor();
      await focus("evidence-title");
      evidenceUrl = page.url();
      assert.equal(
        await page.locator(".evidence-full-table tbody tr").count(),
        16,
      );
      await page.locator("#evidence-return").click();
      await focus("source-evidence-link");
      await page.locator("#source-evidence-link").click();
      await page.locator(".evidence-summary").waitFor();
      await page.goBack();
      await focus("source-evidence-link");
      await page.goForward();
      await page.locator(".evidence-summary").waitFor();
      if (lang === "en") {
        const url = page.url();
        await page
          .getByRole("button", { name: "Confirm screen review", exact: true })
          .click();
        await page.locator("dialog[open]").waitFor();
        await page.keyboard.press("Escape");
        await page.locator("dialog[open]").waitFor({ state: "hidden" });
        assert.equal(page.url(), url);
        await page.locator("#evidence-title").focus();
      }
      await page.keyboard.press("Escape");
      await focus("source-evidence-link");
      await page.locator("#source-detail > a").first().click();
      await focus(width < 640 ? "source-mobile-1" : "source-row-1");
      await page.goto(evidenceUrl);
      await page.locator(".evidence-summary").waitFor();
      const metrics = await page.evaluate(() => ({
        width: innerWidth,
        scroll: document.documentElement.scrollWidth,
        dir: document.documentElement.dir,
        reviewFont: getComputedStyle(document.querySelector("#confirm-review"))
          .fontSize,
        logo: !!document.querySelector(".brand-logo")?.naturalWidth,
      }));
      assert.ok(metrics.scroll <= width, JSON.stringify(metrics));
      assert.equal(metrics.dir, lang === "ar" ? "rtl" : "ltr");
      assert.equal(metrics.reviewFont, "16px");
      assert.ok(metrics.logo);
      await page.screenshot({
        path: `${out}/${lang}-${width}.png`,
        fullPage: true,
      });
      renders.push({ lang, ...metrics });
      if (lang === "ar" && width === 390) {
        await page
          .locator(".evidence-summary")
          .screenshot({ path: `${out}/ar-summary.png` });
        for (const index of [0, 3, 7])
          await page
            .locator(".evidence-mobile article")
            .nth(index)
            .screenshot({ path: `${out}/ar-field-${index}.png` });
      }
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.route("**/api/local-evidence/rows/**", async (route) => {
    const response = await route.fetch();
    const body = await response.json();
    const field = body.fields.find((f) => f.name === "service_name");
    field.source_text = '<img src="https://invalid.test/track">\n=1+1';
    field.normalized = field.source_text;
    await route.fulfill({ response, json: body });
  });
  await page.goto(evidenceUrl);
  await page.reload();
  await page.locator(".evidence-summary").waitFor();
  assert.equal(
    await page
      .locator(
        ".evidence-screen img,.evidence-screen iframe,.evidence-screen script",
      )
      .count(),
    0,
  );
  assert.ok(
    (await page.locator(".evidence-screen").innerText()).includes(
      '<img src="https://invalid.test/track">',
    ),
  );
  await page.unroute("**/api/local-evidence/rows/**");
  for (const mode of ["denied", "malformed", "stale"]) {
    await page.route("**/api/local-evidence/rows/**", async (route) => {
      if (mode === "malformed") {
        const response = await route.fetch();
        const body = await response.json();
        delete body.scope;
        await route.fulfill({ response, json: body });
      } else
        await route.fulfill({
          status: mode === "denied" ? 403 : 409,
          contentType: "application/json",
          body: JSON.stringify({
            error: mode === "denied" ? "scope_denied" : "evidence_unavailable",
          }),
        });
    });
    await page.reload();
    await page.locator('.evidence-screen [role="alert"]').waitFor();
    assert.equal(await page.locator(".evidence-summary").count(), 0);
    assert.equal(
      await page.locator(".evidence-full-table tbody tr").count(),
      0,
    );
    await page.unroute("**/api/local-evidence/rows/**");
  }
  await page.goto(base + "#/evidence?lang=en&service=local");
  await page
    .getByText("Select evidence from source data", { exact: true })
    .waitFor();
  await page.goto(base + "#/evidence?lang=en&service=local&row=1");
  await page.locator('.evidence-screen [role="alert"]').waitFor();
  let release, started;
  const gate = new Promise((r) => (release = r)),
    seen = new Promise((r) => (started = r));
  await page.route("**/api/local-evidence/rows/**", async (route) => {
    const response = await route.fetch();
    started();
    await gate;
    try {
      await route.fulfill({ response });
    } catch {}
  });
  await page.goto(evidenceUrl);
  await seen;
  await page.goto(base + "#/evidence?lang=en&service=local");
  release();
  await page
    .getByText("Select evidence from source data", { exact: true })
    .waitFor();
  assert.equal(await page.locator(".evidence-summary").count(), 0);
  await page.unroute("**/api/local-evidence/rows/**");
  assert.deepEqual(errors, []);
  assert.deepEqual(external, []);
  await writeFile(
    `${out}/report.json`,
    JSON.stringify(
      {
        status: "PASS",
        renders,
        checks: [
          "real source extract",
          "explicit return",
          "Back/Forward",
          "Escape",
          "modal guard",
          "detail close focus",
          "inert HTML/formula",
          "no external requests",
          "denied/stale/malformed clearing",
          "entry/malformed context",
          "late response cancellation",
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
