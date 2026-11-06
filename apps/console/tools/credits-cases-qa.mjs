import { createRequire } from "node:module";
import { readFile, mkdir, writeFile } from "node:fs/promises";
import { createHash } from "node:crypto";
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
  url = args.url || "http://127.0.0.1:8805/",
  out = args.output || "qa-output/credits-cases";
const fixtures = JSON.parse(
  await readFile(
    new URL(
      "../src/modules/history/pages/credits-cases/fixtures.json",
      import.meta.url,
    ),
    "utf8",
  ),
);
const views = [
    "balances",
    "reservations",
    "cases",
    "adjustments",
    "payment-terms",
  ],
  endpoint = "**/api/credit-case-review?*",
  faults = [],
  mutations = [],
  checks = [],
  captures = [];
page.on("pageerror", (e) => faults.push(e.message));
page.on("request", (r) => {
  if (r.url().includes("/api/") && r.method() !== "GET")
    mutations.push(r.method());
});
await mkdir(out, { recursive: true });
const state = (s) => page.locator(`.b11-screen[data-b11-state=${s}]`).waitFor();
const selected = (id) =>
  page.waitForFunction(
    (id) =>
      document
        .querySelector(".b11-screen")
        ?.getAttribute("data-b11-selected") === id,
    id,
  );
const row = () => page.locator("button[data-b11-record]:visible").first();
const nav = async (view = "balances", lang = "en", extra = "") => {
  await page.goto("about:blank");
  await page.goto(`${url}#/credits-cases?lang=${lang}&view=${view}${extra}`);
  await page.locator(`.b11-screen[data-b11-view="${view}"]`).waitFor();
};
let scenario = "normal",
  release,
  heldDone,
  finishHeld;
const source = (req) => {
  const p = new URL(req.url()).searchParams,
    view = p.get("view"),
    q = p.get("q") || "",
    d = structuredClone(fixtures[view]);
  let records = d.records;
  if (scenario === "pages")
    for (let i = 3; i < 34; i++)
      records.push({ ...structuredClone(records[0]), id: `QA-BALANCE-${i}` });
  if (scenario === "ambiguous")
    records[0].dimension.unit = "QA invalid money unit";
  if (scenario === "consent") {
    records[0].consent.state = "recorded";
    records[0].consent.evidence_ref = null;
  }
  if (scenario === "hold") records[0].holds[0].effect = "effective";
  if (scenario === "stale-payment") records[0].receipt.freshness = "stale";
  if (scenario === "reserved-null") records[0].reserved = null;
  if (q)
    records = records.filter((r) =>
      (r.id + " " + r.label).toLowerCase().includes(q.toLowerCase()),
    );
  const total = records.length,
    cursor = p.get("cursor"),
    offset = cursor ? 25 : 0;
  return {
    ...d,
    selection: { q },
    snapshot: createHash("sha256")
      .update(JSON.stringify([view, q]))
      .digest("hex"),
    cursor,
    next_cursor: total > offset + 25 ? "page_25" : null,
    total: scenario === "unknown" ? null : total,
    records: records.slice(offset, offset + 25),
    ...(scenario === "live" ? { source_mode: "live" } : {}),
  };
};
const text = () => page.locator(".b11-result:visible").innerText();
const reserved = () =>
  page
    .locator(".b11-result:visible .b11-values>div")
    .filter({
      has: page.getByText("Reserved — source observation", { exact: true }),
    })
    .locator("dd");
let passed = false;
try {
  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390])
      for (const view of views) {
        await page.setViewportSize({ width, height: 1000 });
        await nav(view, lang);
        await state("unconnected");
        assert.equal(
          await page.locator("[data-b11-record],.b11-result").count(),
          0,
        );
        assert.doesNotMatch(
          await page.locator(".b11-screen").innerText(),
          /QA-|USD 0/,
        );
        await page.screenshot({
          path: `${out}/default-${lang}-${width}-${view}.png`,
          fullPage: true,
        });
        captures.push(`default-${lang}-${width}-${view}`);
      }
  checks.push(
    "30 actual default unconnected view/locale/layout variants, no fixtures or fabricated zero",
  );
  await page.route(endpoint, async (r) => {
    const q = new URL(r.request().url()).searchParams.get("q");
    if (q === "DELAY-A") {
      const lateError = scenario === "late-error";
      heldDone = new Promise((resolve) => {
        finishHeld = resolve;
      });
      await new Promise((resolve) => {
        release = resolve;
      });
      try {
        await r.fulfill({
          status: lateError ? 503 : 200,
          contentType: "application/json",
          body: JSON.stringify(
            lateError
              ? { error: "source_busy" }
              : {
                  ...source(r.request()),
                  total: 1,
                  records: [fixtures.balances.records[0]],
                },
          ),
        });
      } catch {}
      finishHeld();
      return;
    }
    if (["403", "409", "503"].includes(scenario))
      return r.fulfill({
        status: Number(scenario),
        contentType: "application/json",
        body: JSON.stringify({
          error:
            scenario === "403"
              ? "scope_denied"
              : scenario === "409"
                ? "snapshot_stale"
                : "source_busy",
          request_id: "qa-injected",
        }),
      });
    if (scenario === "malformed")
      return r.fulfill({
        status: 200,
        contentType: "application/json",
        body: '{"schema":1}',
      });
    await r.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(source(r.request())),
    });
  });
  await page.setViewportSize({ width: 1440, height: 1000 });
  await nav();
  await state("loaded");
  await row().click();
  const id = await row().getAttribute("data-b11-record");
  await selected(id);
  assert.equal(await reserved().innerText(), "USD 0");
  assert.match(await text(), /USD 0\.0000/);
  assert.doesNotMatch(await text(), /undefined|NaN/);
  await page.screenshot({
    path: `${out}/state-balance-zero.png`,
    fullPage: true,
  });
  await page.locator("[data-b11-detail-heading]:visible").focus();
  await page.keyboard.press("Escape");
  await selected("");
  await page.waitForFunction(
    (id) => document.activeElement?.getAttribute("data-b11-record") === id,
    id,
  );
  scenario = "reserved-null";
  await nav();
  await state("loaded");
  await row().click();
  assert.match(await reserved().innerText(), /Not provided/);
  scenario = "normal";
  await nav();
  await state("loaded");
  await page.locator('button[data-b11-record="QA-BALANCE-2"]:visible').click();
  assert.equal(await reserved().innerText(), "40 QA units");
  assert.match(await text(), /Last source observation/);
  assert.match(await text(), /Current available balance.*unconfirmed/);
  await page.screenshot({
    path: `${out}/state-balance-stale-quantity.png`,
    fullPage: true,
  });
  checks.push(
    "detail reserved source0/null/40quantity, exact components, historical authority qualifier, Escape and focus return",
  );
  await nav("reservations");
  await state("loaded");
  await row().click();
  assert.match(await text(), /External outcome unknown/);
  assert.match(await text(), /Reservation or application value/);
  assert.match(await text(), /Not provided by source/);
  await nav("cases");
  await state("loaded");
  await row().click();
  await selected("QA-CASE-1");
  assert.equal(
    await page.locator(".b11-public:visible").evaluate((e) => e.open),
    false,
  );
  assert.equal(
    await page.locator(".b11-internal:visible").evaluate((e) => e.open),
    false,
  );
  assert.match(await text(), /Consent not recorded/);
  await page.locator(".b11-public:visible summary").click();
  assert.match(
    await page.locator(".b11-public:visible p").innerText(),
    /QA public evidence/,
  );
  assert.doesNotMatch(
    await page.locator(".b11-public:visible p").innerText(),
    /QA internal/,
  );
  assert.doesNotMatch(await text(), /QA internal evidence/);
  await page.locator(".b11-public:visible summary").click();
  await page.locator(".b11-internal:visible summary").click();
  assert.match(
    await page.locator(".b11-internal:visible p").innerText(),
    /QA internal evidence/,
  );
  assert.doesNotMatch(await text(), /QA public evidence/);
  await page.screenshot({
    path: `${out}/state-internal-only.png`,
    fullPage: true,
  });
  checks.push(
    "unknown reservation; private/public source containers stay separate; correction link does not infer consent",
  );
  await nav("adjustments");
  await state("loaded");
  await row().click();
  assert.match(await text(), /USD -12\.3400/);
  assert.match(await text(), /Adjustment amounts by currency/);
  assert.equal(
    await page
      .locator(".b11-domain:visible [role=region]")
      .first()
      .getAttribute("aria-label"),
    "Adjustment amount USD",
  );
  assert.match(await text(), /QA-DOC-1/);
  assert.match(await text(), /does not reopen quantity|does not reopen/);
  await nav("payment-terms");
  await state("loaded");
  await row().click();
  assert.match(await text(), /Reported overdue/);
  assert.match(await text(), /USD 25\.0000/);
  assert.match(await text(), /Request recorded; effect not confirmed/);
  await page.locator('button[data-b11-record="QA-PAYMENT-2"]:visible').click();
  assert.match(await text(), /Source reports unknown/);
  assert.match(await text(), /Reported stale/);
  await page.locator('button[data-b11-record="QA-PAYMENT-3"]:visible').click();
  assert.match(await text(), /Explicitly not applicable/);
  await page.locator('button[data-b11-record="QA-PAYMENT-4"]:visible').click();
  assert.match(await text(), /Reported fully paid/);
  assert.match(await text(), /USD 0\.0000/);
  checks.push(
    "original-linked signed adjustment; confirmed/stale/internal/paid payment states remain separate from credit, inquiry and holds",
  );
  scenario = "pages";
  await nav();
  await state("loaded");
  assert.equal(
    await page.locator("button[data-b11-record]:visible").count(),
    25,
  );
  await row().click();
  await page.locator("#b11-next").click();
  await state("loaded");
  await selected("");
  assert.notEqual(await row().getAttribute("data-b11-record"), "QA-BALANCE-1");
  assert.match(page.url(), /credit-snapshot=.*credit-permission_revision=/);
  await page.goBack();
  await state("loaded");
  await selected("");
  assert.equal(await row().getAttribute("data-b11-record"), "QA-BALANCE-1");
  scenario = "normal";
  for (const q of [" QA-BALANCE-1 ", "QA-BALANCE-1"]) {
    await page.locator("#b11-search").fill(q);
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await state("loaded");
    assert.equal(
      await page.locator("button[data-b11-record]:visible").count(),
      1,
    );
  }
  await page.locator("#b11-search").fill("NO-MATCH");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await state("loaded");
  assert.equal(
    await page.locator("button[data-b11-record]:visible").count(),
    0,
  );
  assert.match(
    await page.locator(".b11-screen").innerText(),
    /No matching source records/,
  );
  scenario = "unknown";
  await nav();
  await state("loaded");
  assert.match(
    await page.locator(".b11-page-status").innerText(),
    /Total count unconfirmed/,
  );
  scenario = "normal";
  checks.push(
    "bounded25row pagination pinned to snapshot/permission, history reread, normalized search and exact empty vs unknown count",
  );
  for (const failure of [
    "403",
    "409",
    "503",
    "live",
    "malformed",
    "ambiguous",
    "consent",
    "hold",
    "stale-payment",
  ]) {
    const view = ["consent", "hold"].includes(failure)
      ? "cases"
      : failure === "stale-payment"
        ? "payment-terms"
        : "balances";
    await nav(view);
    await state("loaded");
    await row().click();
    scenario = failure;
    await page.getByRole("button", { name: "New query", exact: true }).click();
    await state("error");
    await selected("");
    assert.equal(
      await page.locator("[data-b11-record],.b11-result").count(),
      0,
    );
    if (["403", "409"].includes(failure))
      assert.equal(
        await page.getByRole("button", { name: "Retry", exact: true }).count(),
        0,
      );
    if (failure === "live")
      assert.match(
        await page.locator(".b11-screen").innerText(),
        /has not been bound/,
      );
    await page.screenshot({
      path: `${out}/error-${failure}.png`,
      fullPage: true,
    });
    scenario = "normal";
    if (failure === "503") {
      await page.getByRole("button", { name: "Retry", exact: true }).click();
      await state("loaded");
    }
  }
  checks.push(
    "denied/stale/busy/retry/live/malformed/mixed dimension/consent/hold/stale-payment payloads remove old evidence",
  );
  for (const late of ["late-success", "late-error"]) {
    scenario = late;
    await nav();
    await state("loaded");
    await row().click();
    await page.locator("#b11-search").fill("DELAY-A");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await state("loading");
    await selected("");
    const deadline = Date.now() + 5000;
    while (!release && Date.now() < deadline)
      await new Promise((r) => setTimeout(r, 10));
    assert(release);
    await page.locator("#b11-search").fill("NO-MATCH");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await state("loaded");
    release();
    release = null;
    await heldDone;
    await page.evaluate(
      () =>
        new Promise((resolve) =>
          requestAnimationFrame(() => requestAnimationFrame(resolve)),
        ),
    );
    await state("loaded");
    await selected("");
    assert.equal(
      await page.locator("button[data-b11-record]:visible").count(),
      0,
    );
  }
  scenario = "normal";
  checks.push(
    "late success/error cannot restore old records or detail after newer query",
  );
  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390])
      for (const view of views) {
        await page.setViewportSize({ width, height: 1000 });
        await nav(view, lang);
        await state("loaded");
        await row().click();
        await page.locator("[data-b11-detail-heading]:visible").waitFor();
        await page.evaluate(() => document.fonts.ready);
        const metrics = await page.locator(".b11-screen").evaluate((e) => ({
          font: getComputedStyle(e).fontSize,
          family: getComputedStyle(e).fontFamily,
          lineHeight: getComputedStyle(e).lineHeight,
          dir: getComputedStyle(e).direction,
          overflow: document.documentElement.scrollWidth > innerWidth + 1,
          controls: [...e.querySelectorAll("button,input,select,td,th")]
            .filter((x) => x.getClientRects().length)
            .map((x) => ({
              size: getComputedStyle(x).fontSize,
              radius: getComputedStyle(x).borderRadius,
              shadow: getComputedStyle(x).boxShadow,
            })),
          h2: [...e.querySelectorAll("h2")]
            .filter((x) => x.getClientRects().length)
            .map((x) => getComputedStyle(x).fontSize),
          borders: [...e.querySelectorAll("button,input,section,td,th")]
            .filter((x) => x.getClientRects().length)
            .flatMap((x) => {
              const s = getComputedStyle(x);
              return ["Top", "Right", "Bottom", "Left"]
                .filter((k) => parseFloat(s[`border${k}Width`]) > 0)
                .map((k) => s[`border${k}Color`]);
            }),
        }));
        assert.equal(metrics.font, "16px");
        assert(
          await page.locator(".b11-result:visible").evaluate((e) => {
            const scope = e.querySelector(".b11-scope"),
              domain = e.querySelector(".b11-domain");
            return (
              !!scope &&
              !!domain &&
              scope.getBoundingClientRect().top <
                domain.getBoundingClientRect().top
            );
          }),
        );
        assert(
          await page
            .locator(".b11-domain:visible h4")
            .evaluateAll((es) =>
              es.every(
                (e) => parseFloat(getComputedStyle(e).marginBlockStart) >= 20,
              ),
            ),
        );

        assert.match(metrics.family, /IBM Plex/);
        assert.equal(metrics.dir, lang === "ar" ? "rtl" : "ltr");
        if (lang === "ar") assert.equal(metrics.lineHeight, "28.8px");
        assert.equal(
          metrics.overflow,
          false,
          JSON.stringify({ lang, width, view, metrics }),
        );
        assert(metrics.h2.every((x) => x === "20px"));
        assert(
          metrics.controls.every(
            (x) =>
              x.size === "16px" && x.radius === "0px" && x.shadow === "none",
          ),
        );
        const neutral = new Set([
          "rgb(141, 148, 156)",
          "rgb(164, 174, 184)",
          "rgb(220, 225, 229)",
        ]);
        assert(metrics.borders.every((c) => neutral.has(c)));
        assert(
          await page
            .locator(".b11-missing:visible")
            .evaluateAll((es) =>
              es.every((e) => getComputedStyle(e).fontWeight === "400"),
            ),
        );
        assert(
          await page
            .locator(".b11-money:visible")
            .evaluateAll((es) =>
              es.every((e) => e.getAttribute("dir") === "ltr"),
            ),
        );
        await page.waitForFunction(() => {
          const e = document.querySelector(
            '.planning-tabs [aria-current="page"]',
          );
          if (!e) return false;
          const r = e.getBoundingClientRect(),
            p = e.parentElement.getBoundingClientRect();
          return r.left >= p.left - 1 && r.right <= p.right + 1;
        });
        await page.screenshot({
          path: `${out}/synthetic-${lang}-${width}-${view}.png`,
          fullPage: true,
        });
        captures.push(`synthetic-${lang}-${width}-${view}`);
        if (width === 390) {
          await page
            .locator("[data-b11-detail-heading]:visible")
            .scrollIntoViewIfNeeded();
          await page.screenshot({
            path: `${out}/viewport-${lang}-${view}-identity.png`,
          });
          await page.locator(".b11-domain:visible").scrollIntoViewIfNeeded();
          await page.screenshot({
            path: `${out}/viewport-${lang}-${view}-domain.png`,
          });
          if (view === "payment-terms" || view === "cases") {
            await page
              .locator(
                view === "payment-terms"
                  ? ".b11-receipt:visible"
                  : ".b11-consent:visible",
              )
              .scrollIntoViewIfNeeded();
            await page.screenshot({
              path: `${out}/viewport-${lang}-${view}-receipt-consent.png`,
            });
          }
        }
      }
  checks.push(
    "30 synthetic view/locale/layout variants plus30normal mobile crops: visible tabs, typography/RTL/neutral square/no shadow/nooverflow/missing regular/wholeamountLTR",
  );
  for (const lang of ["ko", "en", "ar"]) {
    for (const width of [1024, 900]) {
      await page.setViewportSize({ width, height: 1000 });
      await nav("cases", lang);
      await state("loaded");
      await row().click();
      const header = await page
        .locator(".b11-result:visible > header")
        .evaluate((e) => ({
          total: e.getBoundingClientRect().width,
          title: e.querySelector("div").getBoundingClientRect().width,
          close: e.querySelector("button").getBoundingClientRect().width,
        }));
      assert(
        header.title >= Math.min(180, header.total * 0.5),
        JSON.stringify({ lang, width, header }),
      );
      assert(header.close < header.total * 0.5);
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      await page.screenshot({
        path: `${out}/tablet-${lang}-${width}.png`,
        fullPage: true,
      });
    }
    await page.setViewportSize({ width: 390, height: 1000 });
    await nav("cases", lang);
    await state("loaded");
    await row().click();
    await page.locator(".b11-public:visible summary").click();
    await page.locator(".b11-public:visible").scrollIntoViewIfNeeded();
    await page.screenshot({ path: `${out}/public-${lang}-390.png` });
    assert.doesNotMatch(
      await page.locator(".b11-public:visible p").innerText(),
      /QA internal/,
    );
    await page.locator(".b11-evidence:visible summary").click();
    await page.locator(".b11-full-table summary").click();
    await page.locator(".b11-table-region").scrollIntoViewIfNeeded();
    assert(
      await page
        .locator(".b11-full-table td:visible")
        .evaluateAll((es) =>
          es.every((e) => getComputedStyle(e).fontSize === "16px"),
        ),
    );
    assert(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth + 1,
      ),
    );
    await page.screenshot({ path: `${out}/table-${lang}-390.png` });
  }
  checks.push(
    "1024/900 title-close widths and mobile public/evidence/full-table disclosure in three locales",
  );
  await nav("balances", "en", "&review=1");
  await state("loaded");
  await page.locator(".planning-review-tools summary").click();
  for (const persona of ["BL", "AP", "CP"]) {
    await page
      .getByLabel("Example persona · grants nothing")
      .selectOption(persona);
    await state("loaded");
  }
  await page.getByLabel("Example persona · grants nothing").selectOption("OP");
  await page.waitForFunction(() => !document.querySelector(".b11-screen"));
  await page.getByLabel("Example persona · grants nothing").selectOption("BL");
  await page.evaluate(() => {
    location.hash = "#/credits-cases?lang=en&view=balances&review=1";
  });
  await state("loaded");
  await page.getByLabel("Audience preview").selectOption("customer");
  await page.waitForFunction(() => !document.querySelector(".b11-screen"));
  await page.evaluate(() => {
    location.hash = "#/credits-cases?lang=en&review=1";
  });
  await page.evaluate(
    () =>
      new Promise((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(resolve)),
      ),
  );
  await page.locator(".planning-portal-nav").waitFor();
  assert.equal(await page.locator(".b11-screen").count(), 0);
  checks.push(
    "BL/AP/CP preview entry, unrelated persona/customer negative routing; no real-auth certification",
  );
  assert.equal(
    await page.evaluate(() => localStorage.getItem("hb-console-review-v1")),
    null,
  );
  assert.deepEqual(mutations, []);
  assert.deepEqual(faults, []);
  checks.push(
    "no financial or public-message mutations, no owner acceptance and no page exceptions",
  );
  passed = true;
} finally {
  await writeFile(
    `${out}/report.json`,
    JSON.stringify(
      {
        passed,
        checks,
        captures,
        faults,
        mutations,
        browser: browser.version(),
        host: process.platform,
        arch: process.arch,
        node: process.version,
      },
      null,
      2,
    ),
  );
  await browser.close();
}
console.log(
  JSON.stringify({
    passed,
    checks: checks.length,
    captures: captures.length,
    out,
  }),
);
