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
  faults = [];
page.on("pageerror", (e) => faults.push(e.message));
const url = args.url || "http://127.0.0.1:8805/",
  out = args.output || "qa-output/marketplace",
  endpoint = "**/api/focus-marketplace?*";
await mkdir(out, { recursive: true });
const loaded = () =>
  page
    .locator(".b07-screen[data-b07-state=loaded]")
    .waitFor({ timeout: 60000 });
const selected = (id) =>
  page.waitForFunction(
    (id) =>
      document
        .querySelector(".b07-screen")
        ?.getAttribute("data-b07-selected") === id,
    id,
  );
const tab = async (i) => {
  await page.locator(".planning-tabs button").nth(i).click();
  await page
    .locator(
      `.b07-screen[data-b07-view=${["buyer", "seller", "credits", "mapping"][i]}]`,
    )
    .waitFor();
};
const row = () => page.locator("button[data-b07-row]:visible").first();
const response = () =>
  page.waitForResponse((r) => r.url().includes("/api/focus-marketplace?"));
let passed = false,
  captures = [];
try {
  const first = response();
  await page.goto(url + "#/marketplace?lang=en");
  const initial = await (await first).json();
  await loaded();
  assert.equal(initial.authorized_rows, 1121);
  assert.deepEqual(initial.counts, {
    candidate: 1,
    no_difference: 1005,
    unknown: 115,
  });
  assert.equal(
    initial.candidate_amounts[0].billed_cost.coefficient,
    "342000000000000000",
  );
  assert.equal(initial.candidate_amounts[0].effective_cost.coefficient, "0");
  assert.equal(
    await page.locator(".b07-screen input:not(#b07-search)").count(),
    0,
  );
  const id = initial.rows[0].id;
  await row().click();
  await selected(id);
  await page.locator("[data-b07-detail-heading]:visible").waitFor();
  assert.match(
    await page.locator(".b07-desktop-detail").innerText(),
    /Red Hat Inc\.|Amazon Web Services/,
  );
  await page.locator(".b07-desktop-detail button").click();
  await selected("");
  await page.waitForFunction(
    (id) => document.activeElement?.getAttribute("data-b07-row") === id,
    id,
  );
  await row().click();
  await selected(id);
  await page.locator(".b07-desktop-detail button").focus();
  await page.keyboard.press("Escape");
  await selected("");
  await page.waitForFunction(
    (id) => document.activeElement?.getAttribute("data-b07-row") === id,
    id,
  );
  await tab(1);
  assert.equal(await page.locator(".b07-settlement strong").count(), 7);
  assert.equal(await page.locator(".b07-seller input").count(), 0);
  assert.match(await page.locator(".b07-seller").innerText(), /Refunds|Taxes/);
  await tab(2);
  await row().click();
  await selected(id);
  assert.match(
    await page.locator(".b07-desktop-detail").innerText(),
    /Unverified|Not connected/,
  );
  await tab(3);
  assert.match(
    await page.locator(".b07-desktop-detail").innerText(),
    /Offer \/ agreement/,
  );
  assert.equal(await page.locator(".b07-detail input").count(), 0);
  await tab(0);
  for (const text of ["AWS", " AWS "]) {
    await page.locator("#b07-search").fill(text);
    await page.getByRole("button", { name: "Search", exact: true }).click();
    await loaded();
    assert.equal(await page.locator("button[data-b07-row]:visible").count(), 1);
  }
  await page.locator("#b07-search").fill("HB107_NO_MATCH");
  const noMatch = response();
  await page.getByRole("button", { name: "Search", exact: true }).click();
  const emptyAccounts = await (await noMatch).json();
  await loaded();
  assert.equal(emptyAccounts.account_matches, 0);
  assert.equal(emptyAccounts.counts.candidate, 1);
  assert.equal(await page.locator("button[data-b07-account]").count(), 0);
  await page.locator("#b07-search").fill("");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await loaded();
  await page.locator(".b07-account-scope summary").click();
  await page.locator("#b07-account-next").click();
  await loaded();
  assert.match(await page.locator(".b07-account-scope").innerText(), /26–50/);
  await page.locator("button[data-b07-account]").first().click();
  await loaded();
  assert.match(
    await page.locator(".b07-account-scope summary").innerText(),
    /One canonical account/,
  );
  await page.locator(".b07-account-scope summary").click();
  await page.locator("#b07-all-accounts").click();
  await loaded();
  await page.route(
    endpoint,
    (r) =>
      r.fulfill({
        status: 403,
        contentType: "application/json",
        body: JSON.stringify({
          error: "scope_denied",
          request_id: "qa-denied",
        }),
      }),
    { times: 1 },
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await page.locator(".b07-screen[data-b07-state=error]").waitFor();
  assert.equal(await page.locator("button[data-b07-row]").count(), 0);
  assert.equal(
    await page.getByRole("button", { name: "Retry", exact: true }).count(),
    0,
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await loaded();
  await page.route(
    endpoint,
    (r) =>
      r.fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({ error: "source_busy", request_id: "qa-busy" }),
      }),
    { times: 1 },
  );
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await page.locator(".b07-screen[data-b07-state=error]").waitFor();
  await page.getByRole("button", { name: "Retry", exact: true }).click();
  await loaded();
  await page.route(
    endpoint,
    (r) =>
      r.fulfill({
        status: 409,
        contentType: "application/json",
        body: JSON.stringify({
          error: "query_snapshot_changed",
          request_id: "qa-stale",
        }),
      }),
    { times: 1 },
  );
  await row().click();
  await page.locator(".b07-screen[data-b07-state=error]").waitFor();
  assert.equal(await page.locator(".b07-detail").count(), 0);
  await page.getByRole("button", { name: "New query", exact: true }).click();
  await loaded();
  assert.equal(
    new URLSearchParams(new URL(page.url()).hash.split("?")[1]).has(
      "marketplace-row",
    ),
    false,
  );
  let release, started;
  const start = new Promise((r) => (started = r)),
    gate = new Promise((r) => (release = r));
  await page.route(endpoint, async (r) => {
    if (new URL(r.request().url()).searchParams.get("provider") === "AWS") {
      started();
      await gate;
      await r
        .fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify({
            ...initial,
            selection: { ...initial.selection, provider: "AWS" },
          }),
        })
        .catch(() => {});
    } else await r.continue();
  });
  await page.locator("#b07-provider").selectOption("AWS");
  await start;
  await page.goto(url + "#/marketplace?lang=en&focus-provider=Google+Cloud");
  await loaded();
  release();
  await page.unroute(endpoint);
  assert.match(await page.locator(".b07-summary").innerText(), /115/);
  assert.equal(await row().count(), 0);
  await page.locator("#b07-provider").selectOption("VMware");
  await loaded();
  assert.match(await page.locator(".b07-summary").innerText(), /3/);
  assert.equal(await row().count(), 0);
  assert.equal(
    await page.evaluate(() => localStorage.getItem("hb-console-review-v1")),
    null,
  );
  for (const lang of ["ko", "en", "ar"]) {
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await page.goto(url + `#/marketplace?lang=${lang}`);
      await loaded();
      await row().click();
      await selected(id);
      for (const [i, view] of [
        "buyer",
        "seller",
        "credits",
        "mapping",
      ].entries()) {
        await tab(i);
        await page.locator(`.b07-screen[data-b07-view=${view}]`).waitFor();
        await page.evaluate(() => document.fonts.ready);
        const metrics = await page.locator(".b07-screen").evaluate((el) => {
          const visible = (x) => x.getClientRects().length > 0,
            style = getComputedStyle(el);
          return {
            font: style.fontSize,
            family: style.fontFamily,
            direction: style.direction,
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
        const regression = await page.locator(".b07-screen").evaluate((el) => ({
          detailWeights: [
            ...el.querySelectorAll(".b07-detail p,.b07-detail dt"),
          ]
            .filter((x) => x.getClientRects().length)
            .map((x) => getComputedStyle(x).fontWeight),
          isolatedVersion: [
            ...el.querySelectorAll("p[role=status] bdi[dir=ltr]"),
          ].some((x) => x.textContent?.startsWith("FOCUS ")),
        }));
        assert.ok(regression.detailWeights.every((w) => w === "400"));
        assert.equal(regression.isolatedVersion, true);
        assert.equal(metrics.font, "16px");
        assert.equal(metrics.direction, lang === "ar" ? "rtl" : "ltr");
        assert.equal(metrics.overflow, false);
        assert.deepEqual(metrics.bad, []);
        assert.equal(metrics.rounded, false);
        assert.equal(metrics.headings, true);
        assert.ok(metrics.family.includes("IBM Plex"));
        const neutral = await page.locator(".b07-screen").evaluate((el) => {
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
          metrics.borders.every(
            (c) => neutral.includes(c) || c === "rgba(0, 0, 0, 0)",
          ),
        );
        if (width === 390 && view !== "seller") {
          const detail = page.locator(".b07-mobile-list .b07-detail:visible");
          assert.equal(await detail.count(), 1);
          assert.equal(
            await detail.evaluate(
              (el) =>
                el.closest("article")?.querySelector("button[data-b07-row]") !==
                null,
            ),
            true,
          );
        }
        await page.screenshot({
          path: `${out}/${lang}-${width}-${view}.png`,
          fullPage: true,
        });
        captures.push({ lang, width, view, metrics });
      }
    }
  }
  assert.deepEqual(faults, []);
  passed = true;
} finally {
  await writeFile(
    `${out}/result.json`,
    JSON.stringify({ passed, captures, faults }, null, 2),
  );
  await browser.close();
}
console.log(JSON.stringify({ passed, captures: captures.length, faults }));
