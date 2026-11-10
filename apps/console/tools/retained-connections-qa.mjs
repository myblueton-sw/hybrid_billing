import { createRequire } from "node:module";
import { resolve } from "node:path";
import fs from "node:fs";
import crypto from "node:crypto";
import assert from "node:assert/strict";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .reduce(
      (pairs, value, i, all) =>
        value.startsWith("--")
          ? [...pairs, [value.slice(2), all[i + 1]]]
          : pairs,
      [],
    ),
);
const { chromium } = createRequire(import.meta.url)(
  args["playwright-module"] || "playwright",
);
const out = resolve(args.output || "output/validation/HB-147");
fs.mkdirSync(out, { recursive: true });
const origin = args.url || "http://127.0.0.1:8817/";
const expectedTotal =
  args["expected-total"] === undefined
    ? undefined
    : Number(args["expected-total"]);
if (
  expectedTotal !== undefined &&
  (!Number.isSafeInteger(expectedTotal) ||
    expectedTotal < 0 ||
    expectedTotal > 10000)
)
  throw new Error("Invalid expected-total");
const hash = (s) => crypto.createHash("sha256").update(s).digest("hex");
const rows = Array.from({ length: 26 }, (_, i) => ({
  id: hash("synthetic-registration-" + i),
  connection: "conn-" + String(i).padStart(2, "0"),
  focus_version: "1.2",
  grain: "month",
  window_start: "2026-10-01T00:00:00+00:00",
  window_end: "2026-11-01T00:00:00+00:00",
  registered_domains: "2",
  semantic_profile: "exploratory-query-only",
  allowed_lateness_seconds: "0",
  replacement_limit: "3",
}));
function fixture(url, empty = false) {
  const q = new URL(url).searchParams,
    view = q.get("view") || "connections",
    prefix = q.get("q") || "",
    after = Number(q.get("after") || 0),
    supported = ["connections", "capabilities"].includes(view),
    all =
      supported && !empty
        ? rows.filter(
            (r) => r.connection.startsWith(prefix) || r.id.startsWith(prefix),
          )
        : [];
  return {
    schema: 1,
    profile: "retained-focus-connections-v1",
    read_only: true,
    financial_authority: false,
    commands: "unconnected",
    view,
    selection: { q: prefix, after },
    audience: "company",
    member_count: 2,
    auth_scope: hash("synthetic-auth"),
    permission_revision: hash("synthetic-permission"),
    snapshot: hash(JSON.stringify({ view, prefix, all })),
    observed_at: "2026-11-01T12:00:00.123456+00:00",
    total: supported ? all.length : null,
    next_after: after + 25 < all.length ? after + 25 : null,
    capabilities: {
      connections: "fully-owned-registrations-only",
      capability_evidence: "registration-only",
      provider_authentication: "unconnected",
      diagnostics: "unconnected",
    },
    records: all.slice(after, after + 25),
  };
}
const checks = [],
  captures = [],
  errors = [],
  commands = [];
const names = {
  en: {
    refresh: "Refresh",
    apply: "Apply filter",
    clear: "Clear filter",
    next: "Next page",
    previous: "Previous page",
    close: "Close details",
  },
  ko: { refresh: "새로고침" },
  ar: { refresh: "تحديث" },
};
function hook(page) {
  page.on("pageerror", (e) => errors.push(e.message));
  page.on("request", (r) => {
    if (r.method() !== "GET") commands.push(r.method() + " " + r.url());
  });
}
async function ready(page, phase = "loaded") {
  await page
    .locator('.retained-connections-screen[data-read-phase="' + phase + '"]')
    .waitFor();
}
async function snap(page, name) {
  await page.evaluate(() => document.fonts.ready);
  await page.screenshot({ path: out + "/" + name + ".png", fullPage: true });
  captures.push(name + ".png");
}
async function layout(page, locale) {
  const review = await page.locator("#confirm-review").evaluate((e) => ({
    font: parseFloat(getComputedStyle(e).fontSize),
    height: e.getBoundingClientRect().height,
    revision: [...document.querySelectorAll(".review-source-revision")].some(
      (x) => x.getClientRects().length > 0,
    ),
  }));
  assert(
    review.font >= 16 && review.height >= 44 && review.revision,
    JSON.stringify(review),
  );
  const m = await page
    .locator(".retained-connections-screen")
    .evaluate((el) => {
      const visible = (e) =>
        e.getClientRects().length &&
        getComputedStyle(e).visibility !== "hidden";
      return {
        overflow: document.documentElement.scrollWidth > innerWidth + 2,
        dir: document.documentElement.dir,
        fonts: [...document.fonts]
          .filter((f) => f.status === "loaded")
          .map((f) => f.family),
        styles: [...el.querySelectorAll("h2,button,input,select,th,td")]
          .filter(visible)
          .map((e) => ({
            tag: e.tagName,
            size: parseFloat(getComputedStyle(e).fontSize),
            radius: getComputedStyle(e).borderRadius,
            shadow: getComputedStyle(e).boxShadow,
          })),
        borders: [...el.querySelectorAll("*")]
          .filter((e) => visible(e) && !e.matches(":focus-visible"))
          .flatMap((e) => {
            const s = getComputedStyle(e);
            return ["Top", "Right", "Bottom", "Left"]
              .filter(
                (k) =>
                  parseFloat(s["border" + k + "Width"]) > 0 &&
                  s["border" + k + "Style"] !== "none",
              )
              .map((k) => s["border" + k + "Color"]);
          })
          .filter((c) => {
            const n = c.match(/\d+/g)?.slice(0, 3).map(Number);
            return n && Math.max(...n) - Math.min(...n) > 50;
          }),
      };
    });
  assert.equal(m.overflow, false);
  assert.equal(m.dir, locale === "ar" ? "rtl" : "ltr");
  assert(
    m.fonts.some((x) =>
      x.includes(
        locale === "ko" ? "Sans KR" : locale === "ar" ? "Sans Arabic" : "Sans",
      ),
    ),
  );
  assert(
    !m.styles.some((x) => x.size < (x.tag === "H2" ? 20 : 16)),
    JSON.stringify(m.styles.filter((x) => x.size < (x.tag === "H2" ? 20 : 16))),
  );
  assert(!m.styles.some((x) => x.radius !== "0px" || x.shadow !== "none"));
  assert.deepEqual(m.borders, []);
}
(async () => {
  const b = await chromium.launch({
    headless: true,
    ...(args.chromium ? { executablePath: args.chromium } : {}),
  });
  try {
    const actual = await b.newContext(),
      p = await actual.newPage();
    hook(p);
    for (const locale of ["ko", "en", "ar"])
      for (const width of [1440, 390])
        for (const view of [
          "connections",
          "capabilities",
          "security",
          "diagnostics",
        ]) {
          await p.setViewportSize({ width, height: 1000 });
          await p.goto(
            origin + "#/connections?lang=" + locale + "&view=" + view,
          );
          await ready(p);
          if (expectedTotal !== undefined)
            assert.equal(
              await p.locator(".retained-connection-row-button").count(),
              ["connections", "capabilities"].includes(view)
                ? Math.min(25, expectedTotal)
                : 0,
            );
          assert.equal(
            await p.locator(".retained-connection-counter").count(),
            ["connections", "capabilities"].includes(view) ? 1 : 0,
          );
          await layout(p, locale);
          await snap(p, `actual-${view}-${locale}-${width}`);
          checks.push("actual-" + view + "-" + locale + "-" + width);
        }
    await actual.close();
    const context = await b.newContext(),
      page = await context.newPage();
    hook(page);
    let mode = "normal",
      held = [],
      arrived;
    await context.route(/\/api\/focus-connections(?:\?|$)/, async (route) => {
      const url = route.request().url(),
        query = new URL(url).searchParams;
      if (mode === "hold" || query.get("q") === "slow") {
        held.push({ route, url });
        arrived?.();
        return;
      }
      const status =
        { denied: 403, stale: 409, failed: 503, absent: 404 }[mode] || 200;
      const body =
        status === 200
          ? fixture(url, mode === "empty")
          : {
              error:
                status === 403
                  ? "scope_denied"
                  : status === 409
                    ? "query_snapshot_changed"
                    : status === 404
                      ? "not_found"
                      : "collected_unavailable",
              request_id: "a".repeat(32),
            };
      if (mode === "malformed") body.extra_field = "synthetic-rejected";
      await route.fulfill({
        status,
        contentType: "application/json",
        body: JSON.stringify(body),
      });
    });
    async function go(
      locale = "en",
      width = 1440,
      view = "connections",
      extra = "",
    ) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto("about:blank");
      await page.goto(
        origin + "#/connections?lang=" + locale + "&view=" + view + extra,
      );
      await ready(page);
    }
    for (const locale of ["ko", "en", "ar"])
      for (const width of [1440, 390])
        for (const view of ["connections", "capabilities"]) {
          await go(locale, width, view);
          assert.equal(
            await page.locator(".retained-connection-row-button").count(),
            25,
          );
          await page.locator(".retained-connection-row-button").first().focus();
          await page.keyboard.press("Enter");
          await page.locator(".retained-connection-detail").waitFor();
          await page.waitForFunction(() =>
            document.activeElement?.matches(".retained-connection-detail h2"),
          );
          const visibleHeading = await page
            .locator(".retained-connection-detail h2")
            .boundingBox();
          assert(
            visibleHeading && visibleHeading.y>=0 && visibleHeading.y<1000,JSON.stringify({locale,width,view,visibleHeading}),
          );
          await layout(page, locale);
          await snap(page, `synthetic-selected-${view}-${locale}-${width}`);
          await page.keyboard.press("Escape");
          await page.waitForFunction(() =>
            document.activeElement?.matches(".retained-connection-row-button"),
          );
          checks.push(
            "synthetic-list-detail-" + view + "-" + locale + "-" + width,
          );
        }
    for (const width of [900, 1880]) {
      await go("ar", width);
      await page.locator(".retained-connection-row-button").first().click();
      await page.locator(".retained-connection-detail").waitFor();
      await layout(page, "ar");
      await snap(page, "synthetic-selected-ar-" + width);
      checks.push("selected-ar-" + width);
    }
    await go();
    await page
      .getByRole("button", { name: names.en.next, exact: true })
      .first()
      .click();
    await ready(page);
    await page.waitForFunction(() =>
      location.hash.includes("connection-after=25"),
    );
    assert.equal(
      await page.locator(".retained-connection-row-button").count(),
      1,
    );
    assert(page.url().includes("connection-snapshot="));
    await page.goBack();
    await ready(page);
    await page.waitForFunction(
      () =>
        document.querySelectorAll(".retained-connection-row-button").length ===
        25,
    );
    checks.push("pinned-pages-back");
    await page.locator("#retained-connection-search").fill("conn-01");
    await page
      .getByRole("button", { name: names.en.apply, exact: true })
      .click();
    await ready(page);
    await page.waitForFunction(
      () =>
        document.querySelectorAll(".retained-connection-row-button").length ===
        1,
    );
    assert.equal(
      await page.locator(".retained-connection-row-button").innerText(),
      "conn-01",
    );
    await page
      .getByRole("button", { name: names.en.clear, exact: true })
      .click();
    await ready(page);
    await page.waitForFunction(
      () =>
        document.querySelectorAll(".retained-connection-row-button").length ===
        25,
    );
    checks.push("filter-clear");
    await page.locator("#retained-connection-search").fill("BAD");
    await page
      .getByRole("button", { name: names.en.apply, exact: true })
      .click();
    await page.locator("#retained-connection-search-error").waitFor();
    assert.equal(
      await page
        .locator("#retained-connection-search")
        .getAttribute("aria-invalid"),
      "true",
    );
    checks.push("invalid-prefix");
    await go();
    await page.locator(".retained-connection-row-button").first().click();
    const handoff = await page
      .locator(".retained-connection-detail a")
      .getAttribute("href");
    assert.equal(handoff, "#/imports?lang=en&receipt-q=conn-00");
    assert.match(
      await page.locator(".retained-connection-detail").innerText(),
      /prefix/i,
    );
    checks.push("explicit-prefix-destination");
    for (const locale of ["ko", "en", "ar"])
      for (const width of [1440, 390])
        for (const test of [
          "denied",
          "stale",
          "failed",
          "malformed",
          "absent",
        ]) {
          mode = "normal";
          await go(locale, width);
          await page.locator(".retained-connection-row-button").first().click();
          if (width === 390) {
            await page.keyboard.press("Escape");
            await page.waitForFunction(() =>
              document.activeElement?.matches(
                ".retained-connection-row-button",
              ),
            );
          }
          mode = test;
          await page
            .getByRole("button", { name: names[locale].refresh, exact: true })
            .click();
          await ready(page, test === "absent" ? "unconnected" : "error");
          assert.equal(
            await page.locator(".retained-connection-row-button").count(),
            0,
          );
          assert.equal(
            await page.locator(".retained-connection-detail").count(),
            0,
          );
          assert.equal(
            await page.locator(".retained-connection-counter").count(),
            0,
          );
          await layout(page, locale);
          await snap(page, "synthetic-" + test + "-" + locale + "-" + width);
          mode = "normal";
          await page.locator(".retained-connection-error button").click();
          await ready(page);
          assert.equal(
            await page.locator(".retained-connection-row-button").count(),
            25,
          );
          checks.push("clear-retry-" + test + "-" + locale + "-" + width);
        }
    mode = "empty";
    await go();
    assert.equal(
      await page.locator(".retained-connection-row-button").count(),
      0,
    );
    assert.match(
      await page.locator(".retained-connection-counter").innerText(),
      /0/,
    );
    checks.push("synthetic-confirmed-empty");
    mode = "normal";
    await go();
    await page.locator(".retained-connection-row-button").first().click();
    let seen = new Promise((resolve) => (arrived = resolve));
    mode = "hold";
    await page.getByRole("button", { name: "Refresh", exact: true }).click();
    await seen;
    await ready(page, "loading");
    assert.equal(
      await page.locator(".retained-connection-row-button").count(),
      0,
    );
    assert.equal(await page.locator(".retained-connection-detail").count(), 0);
    assert.equal(await page.locator(".retained-connection-counter").count(), 0);
    mode = "normal";
    const pending = held.splice(0);
    for (const h of pending)
      await h.route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(fixture(h.url)),
      });
    await ready(page);
    checks.push("loading-clears-prior-data");
    seen = new Promise((resolve) => (arrived = resolve));
    await page.locator("#retained-connection-search").fill("slow");
    await page
      .getByRole("button", { name: "Apply filter", exact: true })
      .click();
    await seen;
    await ready(page, "loading");
    await page.locator("#retained-connection-search").fill("conn-01");
    await page
      .getByRole("button", { name: "Apply filter", exact: true })
      .click();
    await ready(page);
    assert.equal(
      await page.locator(".retained-connection-row-button").innerText(),
      "conn-01",
    );
    for (const h of held.splice(0)) {
      try {
        await h.route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(fixture(h.url)),
        });
      } catch {}
    }
    assert.equal(
      await page.locator(".retained-connection-row-button").innerText(),
      "conn-01",
    );
    checks.push("late-response-rejected");
    mode = "normal";
    await go(
      "en",
      1440,
      "connections",
      "&connection-after=25&connection-snapshot=" +
        hash(JSON.stringify({ view: "connections", prefix: "", all: rows })) +
        "&connection-permission=" +
        hash("synthetic-permission"),
    );
    await page
      .locator(".planning-tabs button")
      .filter({ hasText: /Security/i })
      .click();
    await page.waitForFunction(
      () =>
        new URLSearchParams(location.hash.split("?")[1]).get("view") ===
          "security" && !document.querySelector(".retained-connection-counter"),
    );
    await ready(page);
    assert(!page.url().includes("connection-after"));
    assert.equal(await page.locator(".retained-connection-counter").count(), 0);
    checks.push("tab-clears-pins");
    await page.goto(origin + "#/connections?lang=en&service=local");
    await page.locator(".connections-screen").waitFor();
    assert.equal(await page.locator(".retained-connections-screen").count(), 0);
    checks.push("legacy-s01-route-preserved");
    assert.deepEqual(errors, []);
    assert.deepEqual(commands, []);
    const build = JSON.parse(
      fs.readFileSync(
        new URL("../dist/build-manifest.json", import.meta.url),
        "utf8",
      ),
    );
    const report = {
      buildRevision: build.revision,
      checks: checks.length,
      checksDetail: checks,
      captures,
      errors,
      nonGET: commands,
      actualExpectedTotal: expectedTotal ?? null,
      actualResult: "protected registrations; security/diagnostics unsupported",
      fixtures: "intercepted synthetic only",
      ownerAcceptanceCreated: false,
      browser: b.version(),
      host: process.platform,
    };
    fs.writeFileSync(out + "/report.json", JSON.stringify(report, null, 2));
    console.log(
      JSON.stringify({
        checks: checks.length,
        captures: captures.length,
        errors,
        nonGET: commands,
      }),
    );
    await context.close();
  } finally {
    await b.close();
  }
})().catch((e) => {
  console.error(e);
  process.exit(1);
});
