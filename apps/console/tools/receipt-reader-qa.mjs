import { createRequire } from "node:module";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import assert from "node:assert/strict";
const args = Object.fromEntries(
  process.argv
    .slice(2)
    .reduce(
      (p, v, i, a) => (v.startsWith("--") ? [...p, [v.slice(2), a[i + 1]]] : p),
      [],
    ),
);
const require = createRequire(import.meta.url),
  { chromium } = require(args["playwright-module"] || "playwright");
const origin = args.url || "http://127.0.0.1:8808/",
  out = resolve(args.output || "output/validation/HB-145-browser");
await mkdir(out, { recursive: true });
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const errors = [],
  mutations = [],
  checks = [],
  captures = [];
const hook = (page) => {
  page.on("pageerror", (e) => errors.push(e.message));
  page.on("request", (r) => {
    if (r.method() !== "GET")
      mutations.push({ method: r.method(), url: r.url() });
  });
};
const ready = async (page, phase = "loaded") =>
  page.locator(`.receipt-screen[data-read-phase="${phase}"]`).waitFor();
const summarySettled = (page) =>
  page.waitForFunction(
    () =>
      !document.querySelector('.collection-status[data-summary-phase="loading"]'),
    null,
    { timeout: 120000 },
  );
const snap = async (page, name) => {
  await page.evaluate(() => document.fonts.ready);
  await page.evaluate(() => {
    scrollTo(0, 0);
    return new Promise((r) =>
      requestAnimationFrame(() => requestAnimationFrame(r)),
    );
  });
  const path = resolve(out, name + ".png");
  await page.screenshot({ path, fullPage: true });
  captures.push(name + ".png");
};
const live = await browser.newContext(),
  page = await live.newPage();
hook(page);
const API = origin + "api/focus-receipts";
try {
  const observed = {};
  for (const view of ["imports", "quarantine", "completeness", "extraction"]) {
    const response = await page.request.get(API + "?view=" + view, {
      headers: { "X-HB-Local-Client": "1" },
    });
    assert.equal(response.status(), 200);
    observed[view] = await response.json();
    assert.equal(observed[view].financial_authority, false);
    assert.equal(observed[view].commands, "unconnected");
  }
  assert(
    observed.imports.records.length > 0,
    "retained owned receipt is required for actual binding verification",
  );
  for (const locale of ["ko", "en", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      for (const view of [
        "imports",
        "quarantine",
        "completeness",
        "extraction",
      ]) {
        await page.goto(`${origin}#/imports?lang=${locale}&view=${view}`);
        await ready(page);
        await summarySettled(page);
        await page.evaluate(() => document.fonts.ready);
        assert.equal(
          await page.locator(".receipt-open:visible").count(),
          observed[view].records.length,
        );
        if (observed[view].records.length)
          assert.deepEqual(
            await page.locator(".receipt-screen").evaluate((e) => ({
              table: e.querySelector(".receipt-table").getClientRects().length > 0,
              cards: e.querySelector(".receipt-cards").getClientRects().length > 0,
            })),
            { table: width >= 720, cards: width < 720 },
          );
        else assert.equal(await page.locator(".receipt-open").count(), 0);
        const measures = await page
          .locator(".receipt-screen")
          .evaluate((e) => ({
            overflow: document.documentElement.scrollWidth > innerWidth + 2,
            dir: document.documentElement.dir,
            fonts: [...document.fonts]
              .filter((f) => f.status === "loaded")
              .map((f) => f.family),
            type: [...e.querySelectorAll("h2,input,button,th,td")]
              .filter((c) => c.getClientRects().length)
              .filter((c) => !c.closest(".collection-status"))
              .map((c) => ({
                tag: c.tagName,
                font: parseFloat(getComputedStyle(c).fontSize),
                radius: getComputedStyle(c).borderRadius,
                shadow: getComputedStyle(c).boxShadow,
              })),
            leaked: e.innerText.includes("DEMO-"),
            review: {
              font: getComputedStyle(document.querySelector("#confirm-review"))
                .fontSize,
              height: document
                .querySelector("#confirm-review")
                .getBoundingClientRect().height,
            },
          }));
        assert.equal(measures.overflow, false);
        assert.equal(measures.dir, locale === "ar" ? "rtl" : "ltr");
        assert(
          measures.type.every(
            (m) =>
              m.font >= (m.tag === "H2" ? 20 : 16) &&
              m.radius === "0px" &&
              m.shadow === "none",
          ),
        );
        const borders = await page.locator(".receipt-screen").evaluate((e) => {
          const probe = document.createElement("span");
          e.append(probe);
          const allowed = new Set(
            [
              "--hb-line-strong",
              "--hb-line-control",
              "--hb-line-th",
              "--hb-line-metrics",
              "--hb-line-tabs",
              "--hb-line-footer",
              "--hb-line-flow",
              "--hb-line-header",
              "--hb-line-note",
              "--hb-line-panel",
              "--hb-line-details",
              "--hb-line-rail",
              "--hb-line-th-top",
              "--hb-line-cell",
              "--hb-line-facts",
              "--hb-line-nav",
              "--hb-line-row",
            ].map((k) => {
              probe.style.color = `var(${k})`;
              return getComputedStyle(probe).color;
            }),
          );
          probe.remove();
          return [...e.querySelectorAll("*")]
            .filter((n) => n.getClientRects().length)
            .flatMap((n) => {
              const c = getComputedStyle(n);
              return ["Top", "Right", "Bottom", "Left"]
                .filter(
                  (side) =>
                    parseFloat(c[`border${side}Width`]) > 0 &&
                    !["none", "hidden"].includes(c[`border${side}Style`]),
                )
                .map((side) => ({
                  tag: n.tagName,
                  color: c[`border${side}Color`],
                  neutral: allowed.has(c[`border${side}Color`]),
                }));
            });
        });
        assert(
          borders.every((b) => b.neutral),
          JSON.stringify(borders.filter((b) => !b.neutral)),
        );
        assert.equal(measures.leaked, false);
        assert(
          parseFloat(measures.review.font) >= 16 &&
            measures.review.height >= 44,
        );
        assert(
          measures.fonts.some((f) =>
            f.includes(
              locale === "ko"
                ? "Sans KR"
                : locale === "ar"
                  ? "Sans Arabic"
                  : "Sans",
            ),
          ),
        );
        assert(
          await page
            .locator(".receipt-table .receipt-meta,.receipt-list-note")
            .evaluateAll((es) =>
              es.every((e) => parseFloat(getComputedStyle(e).fontSize) >= 16),
            ),
        );
        assert(
          await page
            .locator(".receipt-screen time")
            .evaluateAll((es) =>
              es.every((e) =>
                (e.closest(".receipt-table,.receipt-cards") &&
                /T00:00:00(?:\.0+)?(?:Z|\+00:00)$/.test(e.dateTime)
                  ? /^\d{4}-\d{2}-\d{2} UTC$/
                  : /^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(?:\.\d+)? UTC$/
                ).test(e.textContent),
              ),
            ),
        );
        if (["completeness", "extraction"].includes(view))
          assert.equal(
            await page.locator(".receipt-scope-strip button").count(),
            1,
          );
        await snap(page, `actual-${view}-${locale}-${width}`);
        checks.push(`actual ${view} ${locale} ${width}`);
        if (view === "imports") {
          const button = page.locator(".receipt-open:visible").first();
          await button.click();
          await page.locator(".receipt-detail").waitFor();
          await page.waitForFunction(
            () => document.activeElement?.id === "receipt-detail-title",
          );
          assert(
            (await page.locator(".receipt-detail").innerText()).includes(
              observed.imports.records[0].id,
            ),
          );
          assert(
            await page.locator(".receipt-detail").evaluate((e) => {
              const p = document.createElement("span");
              p.style.color = "var(--hb-line-panel)";
              e.append(p);
              const expected = getComputedStyle(p).color;
              p.remove();
              return getComputedStyle(e).borderTopColor === expected;
            }),
          );
          if (width === 1440)
            assert.equal(
              await page
                .locator(".receipt-table")
                .evaluate((e) => e.scrollWidth > e.clientWidth + 1),
              false,
            );
          await snap(page, `actual-selected-${locale}-${width}`);
          if (width === 390) {
            await page.locator(".receipt-detail").scrollIntoViewIfNeeded();
            await page.screenshot({
              path: resolve(out, `actual-detail-viewport-${locale}-390.png`),
              fullPage: false,
            });
            captures.push(`actual-detail-viewport-${locale}-390.png`);
          }
          await page.locator(".receipt-detail>header>button").click();
          await page.waitForFunction(() =>
            document.activeElement?.classList.contains("receipt-open"),
          );
          checks.push(`actual detail focus-return ${locale} ${width}`);
        }
      }
    }
  await page.setViewportSize({ width: 1880, height: 1000 });
  await page.goto(origin + "#/imports?lang=en");
  await ready(page);
  await page.locator(".receipt-open:visible").first().click();
  await page.locator(".receipt-detail").waitFor();
  assert.equal(
    await page
      .locator(".receipt-table")
      .evaluate((e) => e.scrollWidth > e.clientWidth + 1),
    false,
  );
  await snap(page, "actual-selected-en-1880");
  checks.push("actual wide split panel retains all table columns");
  await page.setViewportSize({ width: 900, height: 1000 });
  await page.goto(origin + "#/imports?lang=ar");
  await ready(page);
  await page.locator(".receipt-open:visible").first().click();
  await page.locator(".receipt-detail").waitFor();
  await snap(page, "actual-selected-ar-900");
  assert.equal(
    await page.evaluate(
      () => document.documentElement.scrollWidth > innerWidth + 2,
    ),
    false,
  );
  checks.push("actual tablet Arabic selected");
  await page.goto(origin + "#/imports?lang=en");
  await ready(page);
  const first = observed.imports.records[0];
  await page.locator("#receipt-search").fill(first.id.slice(0, 16));
  await page.locator(".receipt-search button[type=submit]").click();
  await ready(page);
  await page.waitForFunction(() =>
    new URLSearchParams(location.hash.split("?")[1]).has("receipt-q"),
  );
  await page.waitForFunction(
    () => document.querySelectorAll(".receipt-table .receipt-open").length === 1,
  );
  await page
    .locator(".receipt-search button")
    .filter({ hasText: "Clear search" })
    .click();
  await ready(page);
  await page.waitForFunction(
    () => !new URLSearchParams(location.hash.split("?")[1]).has("receipt-q"),
  );
  await page.waitForFunction(
    (n) => document.querySelectorAll(".receipt-table .receipt-open").length === n,
    observed.imports.records.length,
  );
  checks.push("actual identifier filter and clear");
  await page
    .locator(".receipt-scope-strip button")
    .filter({ hasText: "Refresh evidence" })
    .click();
  await ready(page);
  assert.equal(
    await page.locator(".receipt-open:visible").count(),
    observed.imports.records.length,
  );
  checks.push("actual refresh without mutation");
  await live.close();
  const syntheticRecord = (id, kind = "delivery") => ({
    id,
    kind,
    connection: "qa-isolated",
    stage:
      kind === "delivery"
        ? "verified"
        : kind === "quarantine"
          ? "quarantined"
          : "quarantined",
    window_start: "2026-10-01T00:00:00Z",
    window_end: "2026-11-01T00:00:00Z",
    source_revision: kind === "coverage" ? null : "1",
    focus_version: "1.3",
    format: kind === "delivery" ? "csv" : null,
    declared_rows: kind === "coverage" ? null : "3",
    file_bytes: kind === "delivery" ? "128" : null,
    file_digest: kind === "coverage" ? null : "d".repeat(64),
    mapping_digest: kind === "delivery" ? "e".repeat(64) : null,
    received_at: kind === "coverage" ? null : "2026-10-03T00:00:00Z",
    complete_through: kind === "coverage" ? "2026-10-31T00:00:00Z" : null,
    finalized: kind === "coverage" ? false : null,
    reason: kind === "quarantine" ? "decimal_syntax" : null,
    root_receipt: kind === "quarantine" ? id : null,
    replaced_by: null,
    attempts: kind === "quarantine" ? "0" : null,
    known_zero: kind === "coverage" ? false : null,
    published: kind === "coverage" ? true : null,
  });
  const envelope = (url, records, total = records.length, next = null) => {
    const q = new URL(url).searchParams;
    return {
      schema: 1,
      profile: "retained-focus-receipts-v1",
      read_only: true,
      financial_authority: false,
      commands: "unconnected",
      view: q.get("view") || "imports",
      selection: { q: q.get("q") || "", after: Number(q.get("after") || 0) },
      audience: "company",
      member_count: 2,
      auth_scope: "1".repeat(64),
      permission_revision: "2".repeat(64),
      snapshot: "3".repeat(64),
      observed_at: "2026-11-06T10:00:00Z",
      total,
      next_after: next,
      capabilities: {
        receipts: "fully-authorized-metadata-only",
        completeness: "registered-expectations-only",
        extraction: "unconnected",
      },
      records,
    };
  };
  const isolated = await browser.newContext(),
    fixture = await isolated.newPage();
  hook(fixture);
  await isolated.addInitScript(() => {
    document.addEventListener("DOMContentLoaded", () => {
      const note = document.createElement("aside");
      note.textContent =
        "ISOLATED SYNTHETIC TRANSPORT FIXTURE — NOT RETAINED SOURCE";
      note.style.cssText =
        "padding:16px;border:1px solid currentColor;font:16px sans-serif";
      document.body.prepend(note);
    });
  });
  let mode = "page",
    releaseOld,
    oldStartedResolve;
  let oldStarted = new Promise((r) => (oldStartedResolve = r));
  let oldReleased;
  await fixture.route("**/api/focus-collection-status*", (route) =>
    route
      .fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({
          error: "collected_unavailable",
          request_id: "fixture-summary",
        }),
      })
      .catch(() => {}),
  );
  await fixture.route("**/api/focus-receipts?*", async (route) => {
    const u = new URL(route.request().url()),
      q = u.searchParams.get("q") || "",
      after = Number(u.searchParams.get("after") || 0),
      view = u.searchParams.get("view");
    const fulfill = (body) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(body),
      });
    if (mode === "denied")
      return route.fulfill({
        status: 403,
        contentType: "application/json",
        body: JSON.stringify({
          error: "scope_denied",
          request_id: "f".repeat(32),
        }),
      });
    if (mode === "stale")
      return route.fulfill({
        status: 409,
        contentType: "application/json",
        body: JSON.stringify({
          error: "query_snapshot_changed",
          request_id: "f".repeat(32),
        }),
      });
    if (mode === "absent")
      return route.fulfill({
        status: 404,
        contentType: "application/json",
        body: '{"error":"not_found"}',
      });
    if (mode === "invalid") {
      const bad = envelope(u, [syntheticRecord("a".repeat(64))]);
      bad.records[0].object_key = "FORBIDDEN-PRIVATE-PATH";
      return fulfill(bad);
    }
    if (mode === "empty") return fulfill(envelope(u, []));
    if (mode === "quarantine")
      return fulfill(
        envelope(u, [syntheticRecord("c".repeat(64), "quarantine")]),
      );
    if (mode === "coverage")
      return fulfill(
        envelope(u, [syntheticRecord("d".repeat(64), "coverage")]),
      );
    if (mode === "race" && q === "a") {
      oldStartedResolve();
      await new Promise((r) => (releaseOld = r));
      try {
        await fulfill(envelope(u, [syntheticRecord("a".repeat(64))]));
      } catch {}
      oldReleased?.();
      return;
    }
    if (mode === "race")
      return fulfill(envelope(u, [syntheticRecord("b".repeat(64))]));
    const records = after
      ? [syntheticRecord("b".repeat(64))]
      : Array.from({ length: 25 }, (_, i) =>
          syntheticRecord(i.toString(16).padStart(64, "0")),
        );
    return fulfill(envelope(u, records, 26, after ? null : 25));
  });
  const openFixture = async (url) => {
    await fixture.goto("about:blank");
    await fixture.goto(url);
  };
  await openFixture(origin + "#/imports?lang=en");
  await ready(fixture);
  assert.equal(await fixture.locator(".receipt-open:visible").count(), 25);
  await fixture
    .locator(".receipt-pagination button")
    .filter({ hasText: "Next page" })
    .click();
  await ready(fixture);
  await fixture.waitForFunction(
    () =>
      new URLSearchParams(location.hash.split("?")[1]).get("receipt-after") ===
      "25",
  );
  await fixture.waitForFunction(
    () => document.querySelectorAll(".receipt-table .receipt-open").length === 1,
  );
  await fixture.goBack();
  await ready(fixture);
  await fixture.waitForFunction(
    () =>
      !new URLSearchParams(location.hash.split("?")[1]).has("receipt-after"),
  );
  await fixture.waitForFunction(
    () => document.querySelectorAll(".receipt-table .receipt-open").length === 25,
  );
  checks.push("isolated pinned page and browser back");
  for (const scenario of ["empty", "denied", "stale", "invalid", "absent"]) {
    mode = scenario;
    await openFixture(origin + "#/imports?lang=en");
    await ready(
      fixture,
      ["denied", "stale", "invalid"].includes(scenario)
        ? "error"
        : scenario === "absent"
          ? "unconnected"
          : "loaded",
    );
    assert.equal(await fixture.locator(".receipt-open").count(), 0);
    assert.equal(await fixture.locator(".receipt-detail").count(), 0);
    assert(
      !(await fixture.locator("#content").innerText()).includes(
        "FORBIDDEN-PRIVATE-PATH",
      ),
    );
    if (scenario === "empty") {
      assert.equal(await fixture.locator(".receipt-pagination").count(), 0);
      assert.equal(
        (await fixture.locator(".receipt-screen").innerText()).split(
          "No metadata is visible",
        ).length - 1,
        1,
      );
    }
    await snap(fixture, "isolated-" + scenario);
    checks.push("isolated " + scenario + " clears evidence");
  }
  mode = "quarantine";
  await openFixture(origin + "#/imports?lang=ko&view=quarantine");
  await ready(fixture);
  await fixture.locator(".receipt-open:visible").click();
  await fixture.locator(".receipt-detail").waitFor();
  await snap(fixture, "isolated-quarantine-selected-ko");
  checks.push("isolated admitted quarantine detail");
  mode = "coverage";
  await openFixture(origin + "#/imports?lang=ar&view=completeness");
  await ready(fixture);
  await fixture.locator(".receipt-open:visible").click();
  await fixture.locator(".receipt-detail").waitFor();
  await snap(fixture, "isolated-coverage-selected-ar");
  checks.push("isolated latest quarantine versus current publication");
  mode = "page";
  await openFixture(origin + "#/imports?lang=en");
  await ready(fixture);
  mode = "denied";
  await fixture
    .locator(".receipt-scope-strip button")
    .filter({ hasText: "Refresh evidence" })
    .click();
  await ready(fixture, "error");
  assert.equal(await fixture.locator(".receipt-open").count(), 0);
  checks.push("isolated revocation clears formerly loaded rows");
  mode = "page";
  await fixture.locator(".receipt-error button").click();
  await ready(fixture);
  assert.equal(await fixture.locator(".receipt-open:visible").count(), 25);
  checks.push("isolated retry recovers after denial");
  mode = "page";
  await openFixture(origin + "#/imports?lang=en");
  await ready(fixture);
  mode = "race";
  await fixture.locator("#receipt-search").fill("a");
  await fixture.locator(".receipt-search button[type=submit]").click();
  await oldStarted;
  await ready(fixture, "loading");
  assert.equal(await fixture.locator(".receipt-open").count(), 0);
  await snap(fixture, "isolated-loading");
  await fixture.locator("#receipt-search").fill("b");
  await fixture.locator(".receipt-search button[type=submit]").click();
  await ready(fixture);
  await fixture.waitForFunction(
    () =>
      document.querySelector(".receipt-table .receipt-open bdi")?.textContent ===
      "bbbbbbbbbbbb",
  );
  const released = new Promise((r) => (oldReleased = r));
  releaseOld();
  await released;
  await fixture.evaluate(
    () =>
      new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r))),
  );
  assert.equal(
    await fixture.locator(".receipt-open:visible bdi").first().innerText(),
    "bbbbbbbbbbbb",
  );
  checks.push("isolated late old result cannot replace new query");
  const storage = await fixture.evaluate(() => ({
    local: Object.keys(localStorage),
    session: Object.keys(sessionStorage),
  }));
  assert.deepEqual(storage, { local: [], session: [] });
  await isolated.close();
  assert.deepEqual(errors, []);
  assert.deepEqual(mutations, []);
  const build = JSON.parse(
    await readFile(
      new URL("../dist/build-manifest.json", import.meta.url),
      "utf8",
    ),
  );
  await writeFile(
    resolve(out, "report.json"),
    JSON.stringify(
      {
        passed: true,
        checks,
        captures,
        errors,
        mutations,
        ownerAcceptanceCreated: false,
        retained: {
          imports: observed.imports.total,
          quarantine: observed.quarantine.total,
          completeness: observed.completeness.total,
          extraction: observed.extraction.total,
        },
        buildRevision: build.revision,
        browser: browser.version(),
        node: process.version,
        platform: process.platform,
        architecture: process.arch,
      },
      null,
      2,
    ),
  );
  console.log(
    JSON.stringify({
      passed: true,
      checks: checks.length,
      captures: captures.length,
      errors,
      mutations,
    }),
  );
} finally {
  await browser.close();
}
