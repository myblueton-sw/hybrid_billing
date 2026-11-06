import { createRequire } from "node:module";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
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
const require = createRequire(import.meta.url);
const { chromium } = require(args["playwright-module"] || "playwright");
const origin = args.url || "http://127.0.0.1:8807/";
const output = resolve(args.output || "output/validation/HB-144");
const screens = JSON.parse(
  await readFile(
    new URL("../../../docs/features/HB-144/screens.json", import.meta.url),
    "utf8",
  ),
);
await mkdir(output, { recursive: true });
const browser = await chromium.launch({
  headless: true,
  ...(args.chromium ? { executablePath: args.chromium } : {}),
});
const findings = [],
  checks = [],
  errors = [],
  requests = [];
const context = await browser.newContext();
const page = await context.newPage();
page.on("pageerror", (e) => errors.push(e.message));
page.on("request", (r) => {
  if (!r.url().startsWith(new URL(origin).origin) || r.method() !== "GET")
    requests.push({ method: r.method(), url: r.url() });
});
page.on("dialog", (d) => d.accept());
try {
  for (const locale of ["ko", "en", "ar"]) {
    for (const width of [1440, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      for (const screen of screens) {
        for (const [index, view] of screen.tabs.entries()) {
          const identity = `${screen.id}-${view.key}-${locale}-${width}`;
          await page.goto(
            `${origin}#${screen.path}?lang=${locale}&view=${view.key}`,
          );
          await page.locator(`#content [data-remaining-screen="${screen.id}"][data-remaining-view="${view.key}"]`).waitFor();
          await page.waitForFunction(
            (l) => document.documentElement.lang === l,
            locale,
          );
          await page.evaluate(() => document.fonts.ready);
          const measured = await page.locator("#content").evaluate((e) => {
            const controls = [
              ...e.querySelectorAll("input,select,textarea,button"),
            ].filter((c) => c.getClientRects().length);
            const unnamed = controls
              .filter(
                (c) =>
                  c.tagName !== "BUTTON" &&
                  !c.getAttribute("aria-label") &&
                  !c.getAttribute("aria-labelledby") &&
                  (!c.labels || c.labels.length === 0),
              )
              .map((c) => c.outerHTML.slice(0, 160));
            const styles = [
              ...e.querySelectorAll(
                "h2,input:not([type=radio]):not([type=checkbox]),select,textarea,th,td",
              ),
            ]
              .filter((c) => c.getClientRects().length)
              .map((c) => ({
                tag: c.tagName,
                font: parseFloat(getComputedStyle(c).fontSize),
                radius: getComputedStyle(c).borderRadius,
                shadow: getComputedStyle(c).boxShadow,
              }));
            const borders = [...e.querySelectorAll("*")]
              .filter(
                (c) =>
                  c.getClientRects().length && !c.matches(":focus-visible"),
              )
              .flatMap((c) => {
                const s = getComputedStyle(c);
                return ["Top", "Right", "Bottom", "Left"]
                  .filter(
                    (side) =>
                      parseFloat(s[`border${side}Width`]) > 0 &&
                      s[`border${side}Style`] !== "none",
                  )
                  .map((side) => s[`border${side}Color`]);
              })
              .filter((color) => {
                const rgb = color.match(/\d+/g)?.slice(0, 3).map(Number);
                return rgb && Math.max(...rgb) - Math.min(...rgb) > 50;
              });
            return {
              text: e.innerText.slice(0, 300),
              bodyOverflow:
                document.documentElement.scrollWidth > innerWidth + 2,
              unnamed,
              styles,
              borders,
              fonts: [...document.fonts]
                .filter((f) => f.status === "loaded")
                .map((f) => f.family),
              dir: document.documentElement.dir,
              legacy: !!e.querySelector(".candidate-workbench"),
            };
          });
          const problems = [];
          const reviewSizing=await page.locator("#confirm-review").evaluate(e=>({font:parseFloat(getComputedStyle(e).fontSize),height:e.getBoundingClientRect().height,revisionVisible:[...document.querySelectorAll(".review-source-revision")].some(r=>r.getClientRects().length>0)}));
          if(reviewSizing.font<16||reviewSizing.height<44||!reviewSizing.revisionVisible)problems.push("review control sizing or revision visibility");
          if (measured.bodyOverflow) problems.push("document overflow");
          if (measured.unnamed.length)
            problems.push(
              `unnamed controls ${JSON.stringify(measured.unnamed)}`,
            );
          if (measured.styles.some((s) => s.font < (s.tag === "H2" ? 20 : 16)))
            problems.push("undersized typography");
          if (
            measured.styles.some(
              (s) => s.radius !== "0px" || s.shadow !== "none",
            )
          )
            problems.push("rounded or shadow controls");
          if (measured.borders.length)
            problems.push(
              `decorative colored borders ${[...new Set(measured.borders)]}`,
            );
          if (measured.dir !== (locale === "ar" ? "rtl" : "ltr"))
            problems.push("incorrect document direction");
          if (
            !measured.fonts.some((f) =>
              f.includes(
                locale === "ko"
                  ? "Sans KR"
                  : locale === "ar"
                    ? "Sans Arabic"
                    : "Sans",
              ),
            )
          )
            problems.push("local font not loaded");
          if (measured.legacy) problems.push("generic candidate fallback");
          const review = page.getByText(
            locale === "ko"
              ? "검수 확인"
              : locale === "en"
                ? "Confirm screen review"
                : "تأكيد مراجعة الشاشة",
            { exact: true },
          );
          if ((await review.count()) === 0)
            problems.push("missing screen review control");
          if (problems.length) findings.push({ identity, problems, measured });
          checks.push({
            identity,
            pass: problems.length === 0,
            controls: measured.styles.length,
          });
          if (index === 0)
            await page.screenshot({
              path: resolve(output, `${screen.id}-${locale}-${width}.png`),
              fullPage: true,
            });
        }
        console.log(
          `${screen.id} ${locale} ${width} ${screen.tabs.length} views`,
        );
      }
    }
  }
  assert.deepEqual(errors, [], "unexpected page errors");
  assert.deepEqual(requests, [], "external requests or non-GET commands");
  const build=JSON.parse(await readFile(new URL("../dist/build-manifest.json",import.meta.url),"utf8"));
  const summary = {
    buildRevision:build.revision,
    checks: checks.length,
    passed: checks.filter((c) => c.pass).length,
    findings,
    errors,
    requests,
    source: "unconnected",
    ownerAcceptanceCreated: false,
    host: process.platform,
    architecture: process.arch,
    node: process.version,
    browser: browser.version(),
  };
  await writeFile(
    resolve(output, "report.json"),
    JSON.stringify(summary, null, 2),
  );
  await writeFile(
    resolve(output, "checks.json"),
    JSON.stringify(checks, null, 2),
  );
  console.log(
    JSON.stringify({
      checks: summary.checks,
      passed: summary.passed,
      findings: findings.length,
      errors,
      requests,
    }),
  );
  assert.equal(
    findings.length,
    0,
    "rendered findings remain; inspect report.json",
  );
} finally {
  await context.close();
  await browser.close();
}
