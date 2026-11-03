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
const page = await browser.newPage(),
  source = args.url || "http://127.0.0.1:8795/",
  rec = args.reconciliation || "http://127.0.0.1:8796/";
const out = "../../output/validation/HB-142";
await mkdir(out, { recursive: true });
let posts = 0;
const errors = [],
  renders = [];
page.on("pageerror", (e) => errors.push(e.message));
page.on("request", (r) => {
  if (r.method() === "POST" && r.url().endsWith("/api/local-imports")) posts++;
});
const submit = () => page.locator(".live-imports button[type=submit]");
const load = async () => {
  const target = source + "#/imports?lang=en&service=local";
  if (page.url() === target) await page.reload();
  else await page.goto(target);
  await page.getByText("synthetic-account-a", { exact: true }).waitFor();
};
const file = () =>
  page
    .locator("input[type=file]")
    .setInputFiles("../../tests/fixtures/focus/synthetic-1.3.csv");
const cleared = async () => {
  assert.equal(
    await page.getByText("synthetic-account-a", { exact: true }).count(),
    0,
  );
  assert.equal(
    await page.locator("input[type=file]").evaluate((e) => e.files.length),
    0,
  );
  assert.ok(await submit().isDisabled());
};
try {
  await load();
  await file();
  await submit().click();
  await page.locator(".live-upload-review[open]").waitFor();
  assert.equal(posts, 0);
  assert.equal(
    await page.evaluate(() => document.activeElement.textContent),
    "Cancel",
  );
  await page.keyboard.press("Escape");
  await page.locator(".live-upload-review").waitFor({ state: "detached" });
  assert.ok(await submit().evaluate((e) => e === document.activeElement));
  await page.getByLabel("Source revision", { exact: true }).fill("222");
  await submit().click();
  assert.ok(
    (await page.locator(".live-upload-review").innerText()).includes("222"),
  );
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  page.once("dialog", (d) => d.dismiss());
  await page
    .locator(".planning-sidebar a")
    .filter({ hasText: "Closing" })
    .first()
    .click();
  assert.ok(page.url().includes("/imports"));
  page.once("dialog", (d) => d.accept());
  await page
    .locator(".planning-sidebar a")
    .filter({ hasText: "Closing" })
    .first()
    .click();
  assert.ok(!page.url().includes("/imports"));
  await load();
  await file();
  await submit().click();
  await page.route("**/api/local-imports/profile", async (route) => {
    const r = await route.fetch();
    await route.fulfill({
      json: { ...(await r.json()), account: "changed-synthetic-scope" },
    });
  });
  await page
    .getByRole("button", { name: "Confirm reception", exact: true })
    .click();
  await page.getByText("reception_scope_changed", { exact: false }).waitFor();
  assert.equal(posts, 0);
  await cleared();
  await page.unrouteAll();
  await page.getByRole("button", { name: "Refresh from server" }).click();
  await page.getByText("synthetic-account-a", { exact: true }).waitFor();
  await file();
  await page.route("**/api/local-imports/*", (route) =>
    route.request().url().endsWith("/profile")
      ? route.continue()
      : route.fulfill({
          status: 403,
          json: { error: "scope_denied", request_id: "synthetic-denial" },
        }),
  );
  await page.locator(".live-receipt-link").first().click();
  await page.getByText("scope_denied", { exact: false }).waitFor();
  await cleared();
  await page.unrouteAll();
  await load();
  const receipt = await page.locator(".live-receipt-link").first().innerText();
  for (const [base, path, heading] of [
    [source, "imports", "Persisted evidence"],
    [rec, "reconciliation", "Selected input"],
  ]) {
    await page.goto(
      base + "#/" + path + "/" + receipt + "?lang=en&service=local",
    );
    await page.getByRole("heading", { name: heading, exact: true }).waitFor();
    assert.equal(
      await page
        .locator(".live-receipt-link,.rec-select,.rec-comparison")
        .count(),
      0,
    );
    await page.goto(
      base +
        "#/" +
        path +
        "/00000000-0000-0000-0000-000000000000?lang=en&service=local",
    );
    await page.getByRole("alert").waitFor();
    assert.equal(
      await page.getByText("synthetic-account-a", { exact: true }).count(),
      0,
    );
    assert.equal(
      await page.locator(".live-receipt-link,.rec-select").count(),
      0,
    );
  }
  for (const lang of ["en", "ko", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(source + "#/imports?lang=" + lang + "&service=local");
      await page.getByText("synthetic-account-a", { exact: true }).waitFor();
      await file();
      await submit().click();
      await page.locator(".live-upload-review[open]").waitFor();
      const b = await page.locator(".live-upload-review").evaluate((e) => ({
        scroll: e.scrollWidth,
        client: e.clientWidth,
        width: e.getBoundingClientRect().width,
      }));
      assert.ok(b.scroll <= b.client);
      assert.ok(b.width <= width);
      await page.screenshot({ path: out + `/S03-review-${lang}-${width}.png` });
      renders.push({ lang, width });
      await page.keyboard.press("Escape");
      await page.locator("input[type=file]").setInputFiles([]);
    }
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto(
    source + "#/home?lang=en&overview-period=2026-08&ov-currency=KRW",
  );
  await page.locator("#overview-period").waitFor();
  assert.equal(await page.locator("#overview-period").inputValue(), "2026-08");
  assert.equal(await page.locator("#overview-currency").inputValue(), "KRW");
  await page.locator('.planning-sidebar a[href^="#/closing?"]').click();
  await page.locator("#closing-period").waitFor();
  assert.equal(await page.locator("#closing-period").inputValue(), "2026-08");
  assert.ok(page.url().includes("ccy=KRW"));
  await page.locator("#closing-period").selectOption("2026-09");
  await page.locator('.planning-sidebar a[href^="#/home?"]').click();
  await page.locator("#overview-period").waitFor();
  assert.equal(await page.locator("#overview-period").inputValue(), "2026-09");
  assert.equal(await page.locator("#overview-currency").inputValue(), "KRW");
  await page.goBack();
  await page.locator("#closing-period").waitFor();
  assert.equal(await page.locator("#closing-period").inputValue(), "2026-09");
  await page.goForward();
  await page.locator("#overview-period").waitFor();
  assert.equal(await page.locator("#overview-period").inputValue(), "2026-09");
  await page.goto(
    source +
      "#/home?lang=en&period=2026-07&overview-period=2026-08&ccy=USD&ov-currency=KRW",
  );
  assert.equal(await page.locator("#overview-period").inputValue(), "2026-07");
  assert.equal(await page.locator("#overview-currency").inputValue(), "USD");
  assert.equal(posts, 0);
  assert.deepEqual(errors, []);
  await writeFile(
    out + "/review-flow-results.json",
    JSON.stringify(
      {
        renders,
        posts,
        errors,
        scenarios: [
          "cancel/no POST",
          "Escape focus",
          "pinned revision",
          "dirty retain/discard",
          "scope-change rejection",
          "detail denial clearing",
          "exact/unknown S03/S06 details",
          "six modal renders",
          "legacy/canonical period and currency roundtrip with Back/Forward",
          "canonical conflict precedence",
        ],
      },
      null,
      2,
    ),
  );
  console.log(JSON.stringify({ renders: renders.length, posts, errors }));
} finally {
  await browser.close();
}
