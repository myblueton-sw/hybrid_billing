import { createRequire } from "node:module";
import { mkdir, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
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
  ),
  url = args.url || "http://127.0.0.1:8787/",
  out = resolve("qa-output/setup");
await mkdir(out, { recursive: true });
const b = await chromium.launch({
    headless: true,
    ...(args.chromium ? { executablePath: args.chromium } : {}),
  }),
  checks = [],
  errors = [];
const ok = (n) => checks.push(n);
let context, p;
async function fresh(hash, viewport = { width: 1440, height: 1000 }) {
  await context?.close();
  context = await b.newContext({ viewport });
  p = await context.newPage();
  p.on("pageerror", (e) => errors.push(e.message));
  await p.goto(url + hash);
  await p.locator(".setup-wizard").waitFor();
}
async function route(hash) {
  await p.evaluate((h) => (location.hash = h), hash);
  await p.locator(".setup-wizard").waitFor();
}
const current = () => p.evaluate(() => location.hash.slice(1).split("?")[0]);
async function items() {
  await p.locator("#confirm-review").click();
  const list = await p.locator("dialog[open] details li bdi").allTextContents();
  await p.keyboard.press("Escape");
  return list;
}
async function journeyToReadtest(query) {
  await route(`#setup-progress?${query}`);
  await p.locator("#setup-start").click();
  for (const step of ["setup-profile", "setup-attribution"]) {
    assert.equal(await current(), step);
    await p.locator("#setup-example").click();
    await p.locator("#setup-save").click();
  }
  assert.equal(await current(), "setup-connectors");
  await p.locator("#setup-example").click();
  await p.locator("#setup-plan").click();
  await p.locator('[name="planAck"]').check();
  await p.locator("#setup-connection-next").click();
  assert.equal(await current(), "setup-readtest");
}
const pass = (l) =>
  ({
    ko: "예시 검사 통과",
    en: "Example check passed",
    ar: "نجح الفحص التجريبي",
  })[l];
try {
  await fresh("#setup-connectors?mode=synthetic");
  assert.equal(await p.locator("main [role=status] h2").count(), 1);
  assert.equal(await p.locator(".setup-steps-nav button:disabled").count(), 6);
  assert(
    (await p.getByRole("button", { name: /잠김/ }).count()) === 5,
    "locked status in accessible name",
  );
  assert.deepEqual(await items(), [
    "region:layout",
    "nav:setup-steps",
    "notice:draft-boundary",
    "state:prerequisite",
    "action:resume",
  ]);
  ok("locked deep link shows prerequisite with named locked steps");

  await journeyToReadtest("mode=synthetic");
  ok("profile, attribution and connectors complete in order");
  assert((await items()).includes("state:idle"));
  await p.locator(".scenario summary").focus();
  await p.keyboard.press("Enter");
  await p.locator("#scenario-denied").click();
  await p.locator("#setup-diagnose").click();
  await p.locator(".setup-check-row .status.failed").waitFor();
  assert.equal(await p.locator(".setup-check-row .status.ready").count(), 2);
  assert(await p.locator("#setup-next").isDisabled());
  assert((await items()).includes("state:read-denied"));
  ok("denied scenario fails only the bounded read");
  await p.locator("#scenario-outcome").click();
  await p.locator("#setup-diagnose").click();
  await p.locator(".setup-check-row .status.failed").nth(2).waitFor();
  assert((await items()).includes("state:outcome-unknown"));
  ok("outcome-unknown scenario reports unknown");

  await p.locator("#scenario-normal").click();
  await p.locator("#setup-diagnose").click();
  await route("#setup-schedule?mode=synthetic");
  await route("#setup-readtest?mode=synthetic");
  await p.waitForTimeout(700);
  assert.equal(await p.locator(".setup-check-row .status.pending").count(), 0);
  assert(!(await p.locator("#setup-diagnose").isDisabled()));
  assert.equal(await p.locator(".setup-check-row .status.ready").count(), 0);
  assert((await items()).includes("state:idle"));
  ok("leaving mid-run leaves no running or late result");

  await p.locator("#setup-diagnose").click();
  await p.locator("#setup-next:not([disabled])").waitFor();
  assert.equal(
    await p.evaluate(() => document.activeElement?.id),
    "setup-diagnose",
  );
  ok("focus returns to the run action after completion");
  assert((await items()).includes("state:diagnostic-pass"));
  await p.locator("#setup-next").click();
  assert.equal(await current(), "setup-schedule");
  await p.locator("#setup-example").click();
  await p.locator("#setup-save").click();
  assert.equal(await current(), "setup-first");
  await p.locator("#setup-validate").click();
  await p.locator("#setup-next:not([disabled])").waitFor();
  const firstItems = await items();
  assert(firstItems.includes("state:receipt-pass"));
  assert(firstItems.includes("state:evidence-blocked"));
  await p.locator("#setup-next").click();
  assert.equal(await current(), "setup-readiness");
  assert(
    await p.locator(".form-actions button[disabled]").first().isDisabled(),
  );
  assert.match(await p.locator(".facts").innerText(), /차단/);
  ok("journey reaches readiness with financial eligibility blocked");

  await route("#setup-readiness?mode=unconnected");
  assert.equal(await p.locator("main [role=status] h2").count(), 1);
  assert.doesNotMatch(
    await p.locator(".setup-content").innerText(),
    new RegExp(pass("ko")),
  );
  ok("unconnected mode shows no synthetic results");
  await journeyToReadtest("mode=unconnected");
  assert(await p.locator("#setup-diagnose").isDisabled());
  assert((await items()).includes("state:not-connected"));
  ok("unconnected diagnostics are disabled");

  await route("#setup-readiness?mode=synthetic");
  assert(
    !(await p.locator('[data-setup-route="setup-connectors"]').isDisabled()),
  );
  await route("#setup-connectors?mode=synthetic&scope=sample-b");
  assert.equal(await p.locator("main [role=status] h2").count(), 1);
  ok("account scopes keep separate drafts");

  await route("#setup-profile?mode=synthetic");
  await p.locator('[name="entity"]').fill("Changed LLC");
  assert(await p.locator('[data-setup-route="setup-connectors"]').isDisabled());
  ok("upstream edit invalidates later steps");

  await journeyToReadtest("mode=synthetic&scope=sample-b&profile=evidence");
  await p.locator('[data-setup-route="setup-connectors"]').click();
  await p.locator('[name="provider"]').selectOption("azure");
  assert.equal(await p.locator('[name="account"]').inputValue(), "");
  assert(await p.locator('[data-setup-route="setup-readtest"]').isDisabled());
  ok("provider change resets account and invalidates diagnostics");
  await p.locator('[name="provider"]').selectOption("kt");
  await p.locator(".setup-blocked").waitFor();
  await p.locator("#setup-plan").click();
  await p.locator("#setup-error").waitFor();
  assert((await items()).includes("state:profile-blocked"));
  ok("unverified profile cannot pass plan review");

  const colored = await p.evaluate(() =>
    [...document.querySelectorAll(".setup-wizard, .setup-wizard *")]
      .flatMap((e) => {
        const c = getComputedStyle(e);
        return ["Top", "Right", "Bottom", "Left"]
          .filter(
            (side) =>
              parseFloat(c[`border${side}Width`]) > 0 &&
              c[`border${side}Color`] !== c.backgroundColor,
          )
          .map((side) => c[`border${side}Color`]);
      })
      .filter((color) => {
        const [r, g, bl] = color.match(/\d+/g).map(Number);
        return Math.max(r, g, bl) - Math.min(r, g, bl) > 40;
      }),
  );
  assert.deepEqual(colored, []);
  ok("wizard borders are neutral");

  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390]) {
      await fresh(`#setup-connectors?mode=synthetic&lang=${lang}`, {
        width,
        height: 1000,
      });
      await route(`#setup-profile?mode=synthetic&lang=${lang}`);
      await p.locator("#setup-example").click();
      await p.evaluate(() => document.fonts.ready);
      assert.equal(
        await p.getAttribute("html", "dir"),
        lang === "ar" ? "rtl" : "ltr",
      );
      assert(
        await p.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      await p.evaluate(() => scrollTo(0, 0));
      await p.screenshot({
        path: resolve(out, `setup-profile-${lang}-${width}.png`),
        fullPage: true,
      });
      await p.locator("#setup-save").click();
      await p.locator("#setup-example").click();
      await p.locator("#setup-save").click();
      await p.locator("#setup-example").click();
      await p.locator("#setup-plan").click();
      await p.locator('[name="planAck"]').check();
      assert.equal(
        await p.locator('[name="account"]').getAttribute("dir"),
        "ltr",
      );
      assert(
        await p.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      await p.evaluate(() => scrollTo(0, 0));
      await p.screenshot({
        path: resolve(out, `setup-connectors-${lang}-${width}.png`),
        fullPage: true,
      });
      await p.locator("#setup-connection-next").click();
      await p.locator("#setup-diagnose").click();
      await p.locator("#setup-next:not([disabled])").waitFor();
      assert.equal(await p.locator(".setup-content button.primary").count(), 1);
      assert(await p.locator("#setup-next.primary").count());
      assert(
        await p.evaluate(() => {
          const nav = document
              .querySelector(".setup-steps-nav")
              .getBoundingClientRect(),
            item = document
              .querySelector(".setup-steps-nav li.current")
              .getBoundingClientRect();
          return item.left >= nav.left - 1 && item.right <= nav.right + 1;
        }),
      );
      await p.evaluate(() => scrollTo(0, 0));
      await p.screenshot({
        path: resolve(out, `setup-readtest-${lang}-${width}.png`),
        fullPage: true,
      });
      ok(`setup ${lang} ${width} render`);
    }
  assert.deepEqual(errors, []);
  ok("no runtime errors");
  await writeFile(
    resolve(out, "report.json"),
    JSON.stringify({ checks, errors, browser: b.version() }, null, 2),
  );
  console.log(JSON.stringify({ passed: checks.length, out }));
} finally {
  await b.close();
}
