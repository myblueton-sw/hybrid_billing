import { createRequire } from "node:module";
import { mkdir, readFile, writeFile } from "node:fs/promises";
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
const page = await browser.newPage({ viewport: { width: 1440, height: 1000 } });
const url = args.url || "http://127.0.0.1:8805/",
  out = args.output || "qa-output/claim-history";
const fixture = JSON.parse(
  await readFile(
    new URL(
      "../src/modules/calculation/pages/claim-history/fixtures.json",
      import.meta.url,
    ),
    "utf8",
  ),
);
const views = ["ledger", "partial", "reversals", "references"];
const faults = [],
  mutations = [],
  captures = [],
  checks = [];
page.on("pageerror", (e) => faults.push(e.message));
page.on("request", (r) => {
  if (r.url().includes("/api/") && r.method() !== "GET")
    mutations.push(r.method());
});
await mkdir(out, { recursive: true });
const state = (s) => page.locator(`.b10-screen[data-b10-state=${s}]`).waitFor();
const selected = (id) =>
  page.waitForFunction(
    (id) =>
      document
        .querySelector(".b10-screen")
        ?.getAttribute("data-b10-selected") === id,
    id,
  );
const row = () => page.locator("button[data-b10-record]:visible").first();
const nav = async (view = "ledger", lang = "en", extra = "") => {
  await page.goto("about:blank");
  await page.goto(`${url}#/claims?lang=${lang}&view=${view}${extra}`);
  await page.locator(`.b10-screen[data-b10-view=${view}]`).waitFor();
};
let scenario = "normal",
  release,
  heldDone,
  finishHeld;
const source = (req) => {
  const p = new URL(req.url()).searchParams,
    view = p.get("view"),
    filterState = p.get("state") || "",
    q = p.get("q") || "";
  let records = structuredClone(fixture.records);
  if (scenario === "ambiguous") {
    records[1].external = [
      {
        kind: "erp_order",
        id: "QA-UNCERTAIN",
        revision: null,
        observed_at: null,
        outcome: "unknown",
      },
    ];
  }
  if (scenario === "pages")
    for (let i = 6; i < 34; i++)
      records.push({
        ...structuredClone(records[0]),
        id: `QA-CLAIM-${String(i).padStart(3, "0")}`,
      });
  if (view === "partial") records = records.filter((r) => r.partial === true);
  if (view === "reversals")
    records = records.filter(
      (r) =>
        r.correction.original_claim_id !== null ||
        r.correction.linked_claim_ids.length > 0 ||
        !["none", "unknown"].includes(r.correction.disposition),
    );
  if (view === "references")
    records = records.filter((r) => r.external.length > 0);
  if (filterState)
    records = records.filter((r) => r.protection === filterState);
  if (q)
    records = records.filter((r) =>
      (r.id + " " + r.target.label).toLowerCase().includes(q.toLowerCase()),
    );
  const total = records.length,
    cursor = p.get("cursor"),
    offset = cursor ? 25 : 0;
  const snapshot = createHash("sha256")
    .update(JSON.stringify([view, filterState, q]))
    .digest("hex");
  return {
    ...structuredClone(fixture),
    view,
    selection: { state: filterState, q },
    snapshot,
    cursor,
    next_cursor: total > offset + 25 ? "page_25" : null,
    total: scenario === "unknown" ? null : total,
    records: records.slice(offset, offset + 25),
    ...(scenario === "live" ? { source_mode: "live" } : {}),
  };
};
const endpoint = "**/api/claim-history?*";
let passed = false;
try {
  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390])
      for (const view of views) {
        await page.setViewportSize({ width, height: 1000 });
        await nav(view, lang);
        await state("unconnected");
        assert.equal(
          await page.locator("[data-b10-record],.b10-result").count(),
          0,
        );
        assert.doesNotMatch(
          await page.locator(".b10-screen").innerText(),
          /QA-CLAIM|QA synthetic|60\.000/,
        );
        await page.screenshot({
          path: `${out}/default-${lang}-${width}-${view}.png`,
          fullPage: true,
        });
        captures.push(`default-${lang}-${width}-${view}`);
      }
  checks.push("24 actual default unconnected locale/layout/view variants");
  await page.route(endpoint, async (r) => {
    const q = new URL(r.request().url()).searchParams.get("q");
    if (
      q === "DELAY-A" &&
      !new URL(r.request().url()).searchParams.get("state")
    ) {
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
          body: lateError
            ? JSON.stringify({ error: "source_busy" })
            : JSON.stringify({
                ...source(r.request()),
                total: 1,
                records: [fixture.records[0]],
              }),
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
  const id = await row().getAttribute("data-b10-record");
  await selected(id);
  assert.match(
    await page.locator(".b10-desktop-detail").innerText(),
    /60\.000/,
  );
  assert.match(
    await page.locator(".b10-desktop-detail").innerText(),
    /External outcome unknown/,
  );
  await page.locator("[data-b10-detail-heading]:visible").focus();
  await page.keyboard.press("Escape");
  await selected("");
  await page.waitForFunction(
    (id) => document.activeElement?.getAttribute("data-b10-record") === id,
    id,
  );
  checks.push(
    "selection, exact source decimals, Escape and origin focus return",
  );
  await page.locator("#b10-state").selectOption("applied");
  await state("loaded");
  await selected("");
  await row().click();
  assert.match(
    await page.locator(".b10-desktop-detail").innerText(),
    /KRW 0\.000/,
  );
  await page.locator("#b10-state").selectOption("replacement_pending");
  await state("loaded");
  await selected("");
  await row().click();
  assert.match(
    await page.locator(".b10-desktop-detail").innerText(),
    /does not authorize replacement or rebilling/,
  );
  checks.push(
    "protection filter invalidates selection; genuine zero and replacement pending boundaries",
  );
  await nav("partial");
  await state("loaded");
  assert.equal(
    await page.locator("button[data-b10-record]:visible").count(),
    2,
  );
  await nav("reversals");
  await state("loaded");
  assert.equal(
    await page.locator("button[data-b10-record]:visible").count(),
    2,
  );
  await row().click();
  assert.match(
    await page.locator(".b10-desktop-detail").innerText(),
    /USD -12\.3400/,
  );
  assert.match(
    await page.locator(".b10-desktop-detail").innerText(),
    /money-only credit does not reopen quantity/,
  );
  await nav("references");
  await state("loaded");
  assert.equal(
    await page.locator("button[data-b10-record]:visible").count(),
    1,
  );
  checks.push(
    "source predicates for partial, correction and external views; exact signed money; no quantity release",
  );
  scenario = "pages";
  await nav("ledger");
  await state("loaded");
  assert.equal(
    await page.locator("button[data-b10-record]:visible").count(),
    25,
  );
  await row().click();
  await page.locator("#b10-next").click();
  await state("loaded");
  await selected("");
  assert.notEqual(await row().getAttribute("data-b10-record"), "QA-CLAIM-001");
  assert.match(page.url(), /claim-snapshot=.*claim-permission_revision=/);
  await page.goBack();
  await state("loaded");
  await selected("");
  assert.equal(await row().getAttribute("data-b10-record"), "QA-CLAIM-001");
  checks.push(
    "bounded 25-row pagination with snapshot/permission pins and history re-read",
  );
  scenario = "normal";
  await nav();
  await state("loaded");
  for (const q of [" QA-CLAIM-004 ", "QA-CLAIM-004"]) {
    await page.locator("#b10-search").fill(q);
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await state("loaded");
    assert.equal(
      await page.locator("button[data-b10-record]:visible").count(),
      1,
    );
  }
  await page.locator("#b10-search").fill("NO-MATCH");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await state("loaded");
  assert.match(
    await page.locator(".b10-screen").innerText(),
    /No matching application records/,
  );
  assert.equal(
    await page.locator("button[data-b10-record]:visible").count(),
    0,
  );
  scenario = "unknown";
  await nav();
  await state("loaded");
  assert.match(
    await page.locator(".b10-page-status").innerText(),
    /Total count unconfirmed/,
  );
  scenario = "normal";
  checks.push(
    "normalized repeated search, verified empty versus unknown total",
  );
  for (const failure of [
    "403",
    "409",
    "503",
    "live",
    "malformed",
    "ambiguous",
  ]) {
    await nav();
    await state("loaded");
    await row().click();
    scenario = failure;
    await page.getByRole("button", { name: "New query", exact: true }).click();
    await state("error");
    await selected("");
    assert.equal(
      await page.locator("[data-b10-record],.b10-desktop-detail").count(),
      0,
    );
    if (["403", "409"].includes(failure))
      assert.equal(
        await page.getByRole("button", { name: "Retry", exact: true }).count(),
        0,
      );
    if (failure === "live")
      assert.match(
        await page.locator(".b10-screen").innerText(),
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
    "denial, stale, unavailable/retry, live-producer rejection, malformed response remove old evidence",
  );
  for (const late of ["late-success", "late-error"]) {
    scenario = late;
    await nav();
    await state("loaded");
    await row().click();
    await page.locator("#b10-search").fill("DELAY-A");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await state("loading");
    await selected("");
    await page.waitForFunction(
      () =>
        document
          .querySelector(".b10-screen")
          ?.getAttribute("data-b10-state") === "loading",
    );
    const deadline = Date.now() + 5000;
    while (!release && Date.now() < deadline)
      await new Promise((r) => setTimeout(r, 10));
    assert(release, "held request intercepted within 5s");
    await page.locator("#b10-state").selectOption("applied");
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
      await page.locator("button[data-b10-record]:visible").count(),
      0,
    );
  }
  checks.push("late success/error cannot restore superseded list/card");
  scenario = "normal";
  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390])
      for (const view of views) {
        await page.setViewportSize({ width, height: 1000 });
        await nav(view, lang);
        await state("loaded");
        {
          await row().click();
          await page.locator("[data-b10-detail-heading]:visible").waitFor();
          assert.match(
            await page.locator(".b10-screen").innerText(),
            /합성|Synthetic|تجريبي/,
          );
        }
        assert.equal(
          await row().evaluate((e) => getComputedStyle(e).alignItems),
          "stretch",
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
        await page.evaluate(() => document.fonts.ready);
        const metrics = await page.locator(".b10-screen").evaluate((e) => ({
          font: getComputedStyle(e).fontSize,
          family: getComputedStyle(e).fontFamily,
          lineHeight: getComputedStyle(e).lineHeight,
          borders: [...e.querySelectorAll("button,input,select,section,td,th")]
            .filter((x) => x.getClientRects().length)
            .flatMap((x) => {
              const s = getComputedStyle(x);
              return ["Top", "Right", "Bottom", "Left"]
                .filter((k) => parseFloat(s[`border${k}Width`]) > 0)
                .map((k) => s[`border${k}Color`]);
            }),
          overflow: document.documentElement.scrollWidth > innerWidth + 1,
          dir: getComputedStyle(e).direction,
          h2: [...e.querySelectorAll("h2")]
            .filter((x) => x.getClientRects().length)
            .map((x) => getComputedStyle(x).fontSize),
          controls: [...e.querySelectorAll("button,input,select")]
            .filter((x) => x.getClientRects().length)
            .map((x) => ({
              size: getComputedStyle(x).fontSize,
              radius: getComputedStyle(x).borderRadius,
              shadow: getComputedStyle(x).boxShadow,
            })),
        }));
        assert.equal(
          metrics.overflow,
          false,
          JSON.stringify({ lang, width, view, metrics }),
        );
        assert.equal(metrics.dir, lang === "ar" ? "rtl" : "ltr");
        assert.equal(metrics.font, "16px");
        assert(
          await page
            .locator(".b10-coverage-value:visible")
            .evaluateAll((es) =>
              es.every((e) => e.getAttribute("dir") === "ltr"),
            ),
        );
        assert.match(metrics.family, /IBM Plex/);
        if (lang === "ar") assert.equal(metrics.lineHeight, "28.8px");
        const neutral = new Set([
          "rgb(141, 148, 156)",
          "rgb(164, 174, 184)",
          "rgb(220, 225, 229)",
        ]);
        assert(
          metrics.borders.every((c) => neutral.has(c)),
          JSON.stringify(metrics.borders),
        );
        assert(metrics.h2.every((x) => x === "20px"));
        assert(
          metrics.controls.every(
            (x) =>
              x.size === "16px" && x.radius === "0px" && x.shadow === "none",
          ),
        );
        await page.screenshot({
          path: `${out}/synthetic-${lang}-${width}-${view}.png`,
          fullPage: true,
        });
        captures.push(`synthetic-${lang}-${width}-${view}`);
        if (width === 390 && views.includes(view)) {
          await page
            .locator("[data-b10-detail-heading]:visible")
            .scrollIntoViewIfNeeded();
          await page.screenshot({
            path: `${out}/viewport-${lang}-${view}-identity.png`,
          });
          await page.locator(".b10-amounts:visible").scrollIntoViewIfNeeded();
          await page.screenshot({
            path: `${out}/viewport-${lang}-${view}-amounts.png`,
          });
        }
      }
  checks.push(
    "24 labelled synthetic layout variants; 16px controls/20px headings/RTL/square/no-shadow/no page overflow",
  );
  for (const lang of ["ko", "en", "ar"]) {
    for (const width of [1024, 900]) {
      await page.setViewportSize({ width, height: 1000 });
      await nav("ledger", lang);
      await state("loaded");
      await row().click();
      const expected =
        width === 900
          ? ".b10-mobile-detail .b10-result:visible"
          : ".b10-desktop-detail .b10-result:visible";
      await page.locator(expected).waitFor();
      const header = await page
        .locator(".b10-result:visible > header")
        .evaluate((e) => ({
          total: e.getBoundingClientRect().width,
          title: e.querySelector("div").getBoundingClientRect().width,
          close: e.querySelector("button").getBoundingClientRect().width,
        }));
      assert(
        header.title >= Math.min(180, header.total * 0.5),
        JSON.stringify({ lang, width, header }),
      );
      assert(
        header.close < header.total * 0.5,
        JSON.stringify({ lang, width, header }),
      );
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
    await nav("reversals", lang);
    await state("loaded");
    await row().click();
    await page.locator(".b10-evidence:visible summary").click();
    await page.locator(".b10-evidence:visible").scrollIntoViewIfNeeded();
    assert(
      await page
        .locator(".b10-evidence:visible bdi")
        .evaluateAll((es) =>
          es.every(
            (e) =>
              getComputedStyle(e).fontSize === "16px" &&
              getComputedStyle(e).fontFamily.includes("IBM Plex"),
          ),
        ),
    );
    await page.screenshot({ path: `${out}/evidence-${lang}-390.png` });
    await page.locator(".b10-full-table summary").click();
    await page.locator(".b10-table-region").scrollIntoViewIfNeeded();
    assert(
      await page
        .locator(".b10-full-table td:visible")
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
    "1024 split and900 inline tablet layouts, open evidence/bodyfont and full-table mobile columns in three locales",
  );
  await nav("ledger", "en", "&review=1");
  await state("loaded");
  await page.locator(".planning-review-tools summary").click();
  for (const persona of ["BL", "AU"]) {
    await page
      .getByLabel("Example persona · grants nothing")
      .selectOption(persona);
    await state("loaded");
  }
  await page.getByLabel("Example persona · grants nothing").selectOption("OP");
  await page.waitForFunction(() => !document.querySelector(".b10-screen"));
  await page.getByLabel("Example persona · grants nothing").selectOption("BL");
  await page.evaluate(() => {
    location.hash = "#/claims?lang=en&view=ledger&review=1";
  });
  await state("loaded");
  await page.getByLabel("Audience preview").selectOption("customer");
  await page.waitForFunction(() => !document.querySelector(".b10-screen"));
  await page.evaluate(() => {
    location.hash = "#/claims?lang=en&review=1";
  });
  await page.evaluate(
    () =>
      new Promise((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(resolve)),
      ),
  );
  await page.locator(".planning-portal-nav").waitFor();
  assert.equal(await page.locator(".b10-screen").count(), 0);
  checks.push(
    "BL/AU preview entry, unrelated persona and customer audience excluded; preview is not real authorization",
  );
  assert.equal(
    await page.evaluate(() => localStorage.getItem("hb-console-review-v1")),
    null,
  );
  assert.deepEqual(mutations, []);
  assert.deepEqual(faults, []);
  checks.push(
    "no financial writes, owner review acceptance or page exceptions",
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
