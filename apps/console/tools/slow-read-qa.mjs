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
const url = args.url || "http://127.0.0.1:8805/",
  out = args.output || "qa-output/slow-read";
await mkdir(out, { recursive: true });
const screens = [
  ["U01", "/home", "data-focus-state"],
  ["S04", "/source-data", "data-source-state"],
  ["B01", "/cost-pools", "data-infra-state"],
  ["B03", "/allocations", "data-allocation-state"],
  ["B04", "/rates", "data-pricing-state"],
  ["B05", "/exchange-rates", "data-fx-state"],
  ["B06", "/commitments", "data-b06-state"],
  ["B07", "/marketplace", "data-b07-state"],
  ["B08", "/billing-runs", "data-b08-state"],
  ["connections", "/connections", "data-read-phase"],
  ["imports", "/imports", "data-read-phase"],
];
const lines = {
  en: "Still working on this query. Other queries may be running first, so this one may be waiting its turn.",
  ko: "아직 조회 중입니다. 먼저 진행 중인 조회가 있으면 차례를 기다린 뒤 조회합니다.",
  ar: "ما زال الاستعلام جارياً. قد تكون استعلامات أخرى قيد التنفيذ قبله، فينتظر دوره.",
};
const loadingSelector = screens
  .map(([, , attr]) => `[${attr}=loading]`)
  .filter((v, i, a) => a.indexOf(v) === i)
  .join(",");
const reader = (u) => u.pathname.startsWith("/api/focus-");
const results = [],
  faults = [];
let passed = false;
async function visit(page, hash, lang) {
  await page.goto("about:blank");
  await page.goto(url + `#${hash}?lang=${lang}`);
}
const settled = (page, attr) =>
  page.waitForFunction(
    (attr) => {
      const root = document.querySelector(`[${attr}]`);
      return root && root.getAttribute(attr) !== "loading";
    },
    attr,
    { timeout: 180000 },
  );
async function held(context, [name, hash, attr], lang, width, cache) {
  const page = await context.newPage();
  page.on("pageerror", (e) => faults.push(`${name}: ${e.message}`));
  await page.setViewportSize({ width, height: 1000 });
  await page.addInitScript((selector) => {
    window.__slow = {};
    new MutationObserver(() => {
      if (!window.__slow.loading && document.querySelector(selector))
        window.__slow.loading = performance.now();
      if (!window.__slow.hint && document.querySelector("[data-slow-read]"))
        window.__slow.hint = performance.now();
    }).observe(document, { subtree: true, childList: true, attributes: true });
  }, loadingSelector);
  let release;
  const gate = new Promise((resolve) => (release = resolve));
  await page.route(reader, async (route) => {
    await gate;
    const live = await route.fetch().catch(() => null);
    if (!live) return route.abort().catch(() => {});
    if (cache && live.ok())
      cache.set(route.request().url(), {
        status: live.status(),
        headers: live.headers(),
        body: await live.body(),
      });
    await route.fulfill({ response: live }).catch(() => {});
  });
  await visit(page, hash, lang);
  await page.locator(`[${attr}=loading]`).waitFor({ timeout: 60000 });
  const hint = page.locator("[data-slow-read]");
  await hint.waitFor({ timeout: 60000 });
  await page.evaluate(() => document.fonts.ready);
  const seen = await page.evaluate((attr) => {
    const el = document.querySelector("[data-slow-read]"),
      status = el.closest("[role]"),
      sentence = el.parentElement,
      style = getComputedStyle(el),
      probe = document.createElement("span");
    probe.style.color = "var(--hb-muted)";
    el.after(probe);
    const muted = getComputedStyle(probe).color;
    probe.remove();
    const root = document.querySelector(`[${attr}]`);
    return {
      count: document.querySelectorAll("[data-slow-read]").length,
      delay: window.__slow.hint - window.__slow.loading,
      text: el.textContent,
      role: status?.getAttribute("role"),
      parent: sentence.tagName,
      insideLoading: root.getAttribute(attr) === "loading" && root.contains(el),
      visible: el.getClientRects().length > 0,
      display: style.display,
      size: style.fontSize,
      weight: style.fontWeight,
      color: style.color,
      muted,
      borders: ["Top", "Right", "Bottom", "Left"].map(
        (k) => style[`border${k}Width`],
      ),
      background: style.backgroundColor,
      animation: style.animationName,
      gap: style.marginTop,
      sentenceSize: getComputedStyle(sentence).fontSize,
      sentenceColor: getComputedStyle(sentence).color,
      direction: getComputedStyle(root).direction,
      overflow: document.documentElement.scrollWidth > innerWidth + 1,
      lineWider: el.scrollWidth > el.clientWidth + 1,
    };
  }, attr);
  await page.screenshot({
    path: `${out}/${name}-${lang}-${width}.png`,
    fullPage: true,
  });
  const label = `${name} ${lang} ${width}`;
  assert.equal(seen.count, 1, label);
  assert.ok(
    seen.delay >= 3900 && seen.delay < 6000,
    `${label} delay ${seen.delay}`,
  );
  assert.equal(seen.text, lines[lang], label);
  assert.equal(seen.role, "status", label);
  assert.equal(seen.parent, "P", label);
  assert.equal(seen.insideLoading, true, label);
  assert.equal(seen.visible, true, label);
  assert.equal(seen.display, "block", label);
  assert.equal(seen.size, "14px", label);
  assert.equal(seen.weight, "400", label);
  assert.equal(seen.muted, "rgb(86, 97, 108)", label);
  assert.equal(seen.color, seen.muted, label);
  assert.deepEqual(seen.borders, ["0px", "0px", "0px", "0px"], label);
  assert.equal(seen.background, "rgba(0, 0, 0, 0)", label);
  assert.equal(seen.animation, "none", label);
  assert.equal(seen.gap, "4px", label);
  assert.equal(seen.sentenceSize, "16px", label);
  assert.equal(seen.direction, lang === "ar" ? "rtl" : "ltr", label);
  assert.equal(seen.overflow, false, label);
  assert.equal(seen.lineWider, false, label);
  release();
  await settled(page, attr);
  await page.waitForFunction(
    () => document.querySelectorAll("[data-slow-read]").length === 0,
    null,
    { timeout: 180000 },
  );
  results.push({
    case: "held",
    name,
    lang,
    width,
    delay: Math.round(seen.delay),
    sentenceColor: seen.sentenceColor,
    state: await page.locator(`[${attr}]`).first().getAttribute(attr),
  });
  assert.equal(results.at(-1).state, "loaded", label);
  await page.close();
}
async function fast(context, [name, hash, attr], cache) {
  const page = await context.newPage();
  page.on("pageerror", (e) => faults.push(`${name}: ${e.message}`));
  await page.setViewportSize({ width: 1440, height: 1000 });
  let live = 0;
  await page.route(reader, async (route) => {
    const stored = cache.get(route.request().url());
    if (stored) return route.fulfill(stored).catch(() => {});
    live += 1;
    await route.continue().catch(() => {});
  });
  await visit(page, hash, "en");
  await settled(page, attr);
  await page.waitForTimeout(4600);
  const count = await page.locator("[data-slow-read]").count();
  const state = await page.locator(`[${attr}]`).first().getAttribute(attr);
  assert.equal(state, "loaded", `${name} fast`);
  assert.equal(
    await page
      .locator(":not(.collection-read-status) > [data-slow-read]")
      .count(),
    0,
    `${name} fast main`,
  );
  if (live === 0) assert.equal(count, 0, `${name} fast`);
  results.push({ case: "fast", name, state, live, hint: count });
  await page.close();
}
async function failed(context, [name, hash, attr]) {
  const page = await context.newPage();
  page.on("pageerror", (e) => faults.push(`${name}: ${e.message}`));
  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.route(reader, (route) =>
    route
      .fulfill({
        status: 503,
        contentType: "application/json",
        body: JSON.stringify({
          error: "collected_unavailable",
          request_id: "qa-slow-read",
        }),
      })
      .catch(() => {}),
  );
  await visit(page, hash, "en");
  await settled(page, attr);
  await page.waitForTimeout(4600);
  assert.equal(
    await page.locator("[data-slow-read]").count(),
    0,
    `${name} failed read`,
  );
  results.push({
    case: "failed",
    name,
    state: await page.locator(`[${attr}]`).first().getAttribute(attr),
  });
  assert.equal(results.at(-1).state, "error", `${name} failed read`);
  await page.close();
}
async function summary(context) {
  const page = await context.newPage();
  page.on("pageerror", (e) => faults.push(`summary: ${e.message}`));
  await page.setViewportSize({ width: 1440, height: 1000 });
  let release,
    heldRequests = 0;
  const gate = new Promise((resolve) => (release = resolve));
  await page.route(reader, async (route) => {
    if (new URL(route.request().url()).pathname.endsWith("collection-status")) {
      heldRequests += 1;
      await gate;
    }
    await route.continue().catch(() => {});
  });
  await visit(page, "/imports", "en");
  const hint = page.locator(".collection-read-status > [data-slow-read]");
  await hint.waitFor({ timeout: 180000 });
  assert.equal(await page.locator("[data-slow-read]").count(), 1);
  assert.equal(await hint.textContent(), lines.en);
  assert.equal(
    await page.locator(".collection-read-status").getAttribute("role"),
    "status",
  );
  await page.screenshot({ path: `${out}/imports-summary-en-1440.png` });
  release();
  await page.waitForFunction(
    () => document.querySelectorAll("[data-slow-read]").length === 0,
    null,
    { timeout: 180000 },
  );
  await page.waitForTimeout(4600);
  assert.equal(await page.locator("[data-slow-read]").count(), 0);
  results.push({ case: "summary", heldRequests });
  await page.close();
}
try {
  const context = await browser.newContext();
  for (const screen of screens) {
    const cache = new Map();
    await held(context, screen, "en", 1440, cache);
    await fast(context, screen, cache);
    await failed(context, screen);
  }
  for (const name of ["U01", "B07", "imports"]) {
    const screen = screens.find((s) => s[0] === name);
    for (const lang of ["ko", "ar"])
      for (const width of [1440, 390]) await held(context, screen, lang, width);
    await held(context, screen, "en", 390);
  }
  await summary(context);
  assert.deepEqual(faults, []);
  assert.equal(results.filter((r) => r.case === "held").length, 26);
  passed = true;
} finally {
  await writeFile(
    `${out}/result.json`,
    JSON.stringify({ passed, results, faults }, null, 2),
  );
  await browser.close();
}
console.log(
  JSON.stringify({
    passed,
    held: results.filter((r) => r.case === "held").length,
    fastAsserted: results.filter((r) => r.case === "fast" && r.live === 0)
      .length,
    fastNotAsserted: results
      .filter((r) => r.case === "fast" && r.live > 0)
      .map((r) => r.name),
    faults,
  }),
);
