import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
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
  faults = [],
  mutations = [];
page.on("pageerror", (e) => faults.push(e.message));
page.on("request", (r) => {
  if (r.url().includes("/api/") && r.method() !== "GET")
    mutations.push(r.method());
});
const url = args.url || "http://127.0.0.1:8805/",
  out = args.output || "qa-output/billing-runs",
  endpoint = "**/api/focus-overview?*",
  views = [
    "runs",
    "inputs",
    "trace",
    "comparison",
    "validation",
    "recognition",
  ];
await mkdir(out, { recursive: true });
const loaded = () =>
  page
    .locator(".b08-screen[data-b08-state=loaded]")
    .waitFor({ timeout: 60000 });
const tab = async (i) => {
  await page.locator(".planning-tabs button").nth(i).click();
  await page.locator(`.b08-screen[data-b08-view=${views[i]}]`).waitFor();
};
const row = () => page.locator("button[data-b08-row]:visible").first();
const selected = (id) =>
  page.waitForFunction(
    (id) =>
      document
        .querySelector(".b08-screen")
        ?.getAttribute("data-b08-selected") === id,
    id,
  );
const kept = () =>
  page.locator(".b08-screen[data-b08-baseline=kept]").waitFor();
const none = () =>
  page.locator(".b08-screen[data-b08-baseline=none]").waitFor();
const response = () =>
  page.waitForResponse((r) => r.url().includes("/api/focus-overview?"));
const refresh = async () => {
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await loaded();
};
let passed = false,
  captures = [];
try {
  if (args["renders-only"] !== "true") {
    const first = response();
    await page.goto(url + "#/billing-runs?lang=en");
    const initial = await (await first).json();
    await loaded();
    assert.equal(initial.row_count, 1121);
    assert.equal(initial.account_count, 78);
    assert.equal(
      initial.amounts.find((a) => a.currency === "CNY").billed_cost.coefficient,
      "11000000000000000000",
    );
    assert.equal(
      initial.amounts.find((a) => a.currency === "USD").billed_cost.coefficient,
      "27378752728990000000",
    );
    assert.match(
      await page.locator(".b08-screen").innerText(),
      /not connected/i,
    );
    assert.equal(
      await page.locator(".b08-screen input:not(#b08-search)").count(),
      0,
    );
    await page.locator("#b08-keep").click();
    await kept();
    await tab(3);
    assert.equal(await page.locator("[data-b08-delta]").count(), 0);
    await refresh();
    await kept();
    assert.equal(await page.locator("[data-b08-delta]").count(), 4);
    for (const e of await page.locator("[data-b08-delta]").all())
      assert.match(await e.innerText(), /0\.00/);
    await tab(1);
    await row().click();
    const id = await row().getAttribute("data-b08-row");
    await selected(id);
    await page.locator("[data-b08-detail-heading]:visible").waitFor();
    await page.locator(".b08-desktop-detail button").focus();
    await page.keyboard.press("Escape");
    await selected("");
    await page.waitForFunction(
      (id) => document.activeElement?.getAttribute("data-b08-row") === id,
      id,
    );
    await row().click();
    await selected(id);
    await tab(2);
    assert.match(
      await page.locator(".b08-screen").innerText(),
      /not a billing-result hash/i,
    );
    await tab(1);
    await page.locator("#b08-next").click();
    await loaded();
    await kept();
    await selected("");
    assert.notEqual(await row().getAttribute("data-b08-row"), id);
    assert.match(
      await page.locator(".b08-input-list>header").innerText(),
      /26–50 \/ 1121/,
    );
    for (const value of ["AWS", " AWS "]) {
      await page.locator("#b08-search").fill(value);
      await page.getByRole("button", { name: "Search", exact: true }).click();
      await loaded();
      await kept();
      assert.match(
        await page.locator(".b08-totals").innerText(),
        /27\.37875272899/,
      );
    }
    await page.locator("#b08-search").fill("HB109_NO_MATCH");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await loaded();
    await kept();
    assert.equal(await page.locator("button[data-b08-account]").count(), 0);
    assert.match(
      await page.locator(".b08-totals").innerText(),
      /27\.37875272899/,
    );
    await page.locator("#b08-search").fill("");
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await loaded();
    if (!(await page.locator(".b08-account-scope").evaluate((e) => e.open)))
      await page.locator(".b08-account-scope summary").click();
    await page.locator("#b08-account-next").click();
    await loaded();
    await kept();
    await page.locator("button[data-b08-account]").first().click();
    await loaded();
    await none();
    if (!(await page.locator(".b08-account-scope").evaluate((e) => e.open)))
      await page.locator(".b08-account-scope summary").click();
    await page.locator("#b08-all-accounts").click();
    await loaded();
    for (const [status, code] of [
      [403, "scope_denied"],
      [503, "source_busy"],
      [409, "query_snapshot_changed"],
    ]) {
      await page.locator("#b08-keep").click();
      await kept();
      await page.route(
        endpoint,
        (r) =>
          r.fulfill({
            status,
            contentType: "application/json",
            body: JSON.stringify({ error: code, request_id: "qa-injected" }),
          }),
        { times: 1 },
      );
      await page
        .getByRole("button", { name: "New query", exact: true })
        .click();
      await page.locator(".b08-screen[data-b08-state=error]").waitFor();
      await none();
      assert.equal(
        await page
          .locator(".b08-total-list,.b08-detail,.b08-currency-comparison")
          .count(),
        0,
      );
      await page.screenshot({
        path: `${out}/state-${status}.png`,
        fullPage: true,
      });
      if (status === 503)
        await page.getByRole("button", { name: "Retry", exact: true }).click();
      else {
        assert.equal(
          await page
            .getByRole("button", { name: "Retry", exact: true })
            .count(),
          0,
        );
        await page
          .getByRole("button", { name: "New query", exact: true })
          .click();
      }
      await loaded();
      await none();
    }
    await page.locator("#b08-keep").click();
    await kept();
    await page.route(
      endpoint,
      (r) =>
        r.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            ...initial,
            source_snapshot: "f".repeat(64),
            snapshot: "e".repeat(64),
            observed_at: "2026-11-06T22:00:00Z",
          }),
        }),
      { times: 1 },
    );
    await refresh();
    await none();
    assert.match(
      await page.locator(".b08-baseline-strip").innerText(),
      /Source reference changed/,
    );
    assert.equal(await page.locator("[data-b08-delta]").count(), 0);
    await refresh();
    const awsResponse = await page.request.get(
      new URL("/api/focus-overview?provider=AWS&period=all", url).href,
      { headers: { "X-HB-Local-Client": "1" } },
    );
    const aws = await awsResponse.json();
    let release, started;
    const start = new Promise((r) => (started = r)),
      gate = new Promise((r) => (release = r));
    await page.locator("#b08-keep").click();
    await kept();
    await page.route(endpoint, async (r) => {
      if (new URL(r.request().url()).searchParams.get("provider") === "AWS") {
        started();
        await gate;
        await r
          .fulfill({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(aws),
          })
          .catch(() => {});
      } else await r.continue();
    });
    await page.locator("#b08-provider").selectOption("AWS");
    await start;
    await page.goto(
      url + "#/billing-runs?lang=en&view=inputs&focus-provider=Google+Cloud",
    );
    await loaded();
    release();
    await page.unroute(endpoint);
    await none();
    assert.equal(
      await page.locator(".b08-input-list>header .b08-count").innerText(),
      "1–25 / 115",
    );
    await page.goto(url + "#/billing-runs?lang=en");
    await loaded();
    await page.locator("#b08-keep").click();
    await kept();
    let releaseHistory, historyStarted;
    const historyStart = new Promise((r) => (historyStarted = r)),
      historyGate = new Promise((r) => (releaseHistory = r));
    await page.route(endpoint, async (r) => {
      if (new URL(r.request().url()).searchParams.get("provider") === "AWS") {
        historyStarted();
        await historyGate;
        await r.abort().catch(() => {});
      } else await r.continue();
    });
    await page.locator("#b08-provider").selectOption("AWS");
    await historyStart;
    await page.goBack();
    await loaded();
    await none();
    releaseHistory();
    await page.unroute(endpoint);
    await page.goto(url + "#/billing-runs?lang=en&view=comparison");
    await loaded();
    await page.locator("#b08-keep").click();
    await kept();
    await page.goto(url + "#/home?lang=en");
    await page.locator(".b08-screen").waitFor({ state: "detached" });
    await page.goto(url + "#/billing-runs?lang=en&view=comparison");
    await loaded();
    await none();
    const partial = structuredClone(initial);
    partial.observed_at = new Date(
      Date.parse(initial.observed_at) - 60000,
    ).toISOString();
    for (const list of [
      partial.amounts,
      ...partial.providers.map((p) => p.amounts),
      ...partial.months.map((m) => m.amounts),
      ...partial.accounts.map((a) => a.amounts),
    ])
      for (const a of list)
        a.billed_cost = { coefficient: null, present: 0, missing: a.rows };
    for (const r of partial.rows) r.billed_cost = null;
    await page.route(
      endpoint,
      (r) =>
        r.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(partial),
        }),
      { times: 1 },
    );
    await refresh();
    await page.locator("#b08-keep").click();
    await kept();
    await refresh();
    await kept();
    assert.ok(
      (await page.locator(".b08-comparison").innerText()).includes(
        "Present / missing",
      ),
    );
    assert.match(
      await page.locator("[data-b08-delta=billed_cost]").first().innerText(),
      /Unavailable/,
    );
    await page.screenshot({
      path: `${out}/partial-comparison.png`,
      fullPage: true,
    });
    assert.deepEqual(await page.evaluate(() => Object.keys(localStorage)), []);
  }
  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + `#/billing-runs?lang=${lang}`);
      await loaded();
      if (await page.locator("#b08-discard").count())
        await page.locator("#b08-discard").click();
      await page.locator("#b08-keep").click();
      await kept();
      await page.locator(".b08-toolbar>button").last().click();
      await loaded();
      await kept();
      for (const [i, view] of views.entries()) {
        await tab(i);
        if (view === "inputs") {
          await row().click();
          await page.locator("[data-b08-detail-heading]:visible").waitFor();
          if (width === 1440) {
            const inputButton = page
              .locator("table button[data-b08-row]")
              .first();
            await inputButton.focus();
            await page.keyboard.press("Escape");
            await selected("");
            assert.equal(
              await page.locator(".b08-full-table").evaluate((e) => e.open),
              true,
            );
            await inputButton.click();
            await page.locator("[data-b08-detail-heading]:visible").waitFor();
          }
        }
        await page.evaluate(() => document.fonts.ready);
        const m = await page.locator(".b08-screen").evaluate((el) => {
          const visible = (x) => x.getClientRects().length > 0,
            s = getComputedStyle(el);
          return {
            font: s.fontSize,
            family: s.fontFamily,
            direction: s.direction,
            overflow: document.documentElement.scrollWidth > innerWidth + 1,
            bad: [...el.querySelectorAll("button,input,select,td,th")]
              .filter(visible)
              .filter((x) => getComputedStyle(x).fontSize !== "16px")
              .map((x) => x.tagName),
            rounded: [
              ...el.querySelectorAll("button,input,select,table,section"),
            ]
              .filter(visible)
              .some(
                (x) =>
                  getComputedStyle(x).borderRadius !== "0px" ||
                  getComputedStyle(x).boxShadow !== "none",
              ),
            headings: [...el.querySelectorAll("h2")]
              .filter(visible)
              .every((x) => getComputedStyle(x).fontSize === "20px"),
            weights: [...el.querySelectorAll(".b08-detail p,.b08-detail dt")]
              .filter(visible)
              .map((x) => getComputedStyle(x).fontWeight),
            version: [
              ...el.querySelectorAll("p[role=status] bdi[dir=ltr]"),
            ].some((x) => x.textContent?.startsWith("FOCUS ")),
            borders: [
              ...el.querySelectorAll("button,input,select,section,td,th"),
            ]
              .filter(visible)
              .flatMap((x) => {
                const s = getComputedStyle(x);
                return ["Top", "Right", "Bottom", "Left"]
                  .filter((k) => parseFloat(s[`border${k}Width`]) > 0)
                  .map((k) => s[`border${k}Color`]);
              }),
          };
        });
        assert.equal(
          await page
            .locator(".b08-account-scope summary bdi")
            .evaluate((e) => getComputedStyle(e).whiteSpace),
          "nowrap",
        );
        assert.equal(m.font, "16px");
        assert.equal(m.direction, lang === "ar" ? "rtl" : "ltr");
        assert.equal(m.overflow, false);
        assert.deepEqual(m.bad, []);
        assert.equal(m.rounded, false);
        assert.equal(m.headings, true);
        assert.ok(m.family.includes("IBM Plex"));
        assert.ok(m.weights.every((w) => w === "400"));
        assert.equal(m.version, true);
        const neutral = await page.locator(".b08-screen").evaluate((el) => {
          const s = getComputedStyle(el);
          return ["--border-control", "--border-section", "--border-row"].map(
            (v) => {
              const d = document.createElement("span");
              d.style.color = s.getPropertyValue(v);
              el.append(d);
              const c = getComputedStyle(d).color;
              d.remove();
              return c;
            },
          );
        });
        assert.ok(
          m.borders.every(
            (c) => neutral.includes(c) || c === "rgba(0, 0, 0, 0)",
          ),
        );
        if (width === 390 && view === "inputs")
          assert.equal(
            await page
              .locator(".b08-mobile-list article .b08-detail:visible")
              .count(),
            1,
          );
        await page.screenshot({
          path: `${out}/${lang}-${width}-${view}.png`,
          fullPage: true,
        });
        captures.push({ lang, width, view, metrics: m });
      }
    }
  assert.deepEqual(faults, []);
  assert.deepEqual(mutations, []);
  assert.deepEqual(await page.evaluate(() => Object.keys(localStorage)), []);
  passed = true;
} finally {
  await writeFile(
    `${out}/result.json`,
    JSON.stringify(
      {
        passed,
        coreExecuted: args["renders-only"] !== "true",
        captures,
        faults,
        mutations,
      },
      null,
      2,
    ),
  );
  await browser.close();
}
console.log(
  JSON.stringify({ passed, captures: captures.length, faults, mutations }),
);
