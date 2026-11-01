import { createRequire } from "node:module";
import { mkdir, writeFile, readFile } from "node:fs/promises";
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
const url = args.url || "http://127.0.0.1:8787/",
  out = "qa-output/entry";
await mkdir(out, { recursive: true });
const b = await chromium.launch({
    headless: true,
    ...(args.chromium ? { executablePath: args.chromium } : {}),
  }),
  c = await b.newContext({ viewport: { width: 1440, height: 1000 } }),
  p = await c.newPage();
const checks = [],
  errors = [],
  external = [],
  requests = [],
  logs = [];
p.on("pageerror", (e) => errors.push(e.message));
p.on("console", (m) => logs.push(m.text()));
p.on("request", (r) => {
  requests.push(r.url());
  if (!r.url().startsWith(url)) external.push(r.url());
});
const ok = (n) => checks.push(n),
  go = async (id, query = "lang=en") => {
    await p.goto(url + "#" + id + "?" + query);
    await p.locator("h1").waitFor();
  };
try {
  await p.goto(url);
  await p.locator(".entry-main-action").waitFor();
  assert.equal(
    await p.locator(".rail,.tnb,#design-directory-results").count(),
    0,
  );
  ok("service entry is default without authenticated rail or directory");
  await p.locator(".entry-main-action").click();
  await p.locator("#entry-workspace").waitFor();
  ok("principal sign-in navigation");
  for (const id of [
    "entry-login",
    "entry-bootstrap",
    "entry-invite",
    "entry-challenge",
  ]) {
    await go(id);
    const hash = await p.evaluate(() => location.hash);
    await p.locator("form button[type=submit]").click();
    assert.equal(await p.locator(".entry-feedback").count(), 0);
    await p
      .locator(id === "entry-login" ? "#entry-workspace" : "#entry-proof")
      .fill("SENSITIVE-EXAMPLE-NOT-A-REAL-SECRET");
    await p.locator("#entry-email").fill("private-example@example.invalid");
    await p.locator("#confirm-review").click();
    await p.locator("#owner-ack").check();
    await p.locator("#save-review").click();
    await p.waitForFunction(() =>
      document.querySelector("#review-status")?.textContent.includes("FIX"),
    );
    await p.locator("#confirm-review").click();
    const download = p.waitForEvent("download");
    await p.getByRole("button", { name: "Export review evidence" }).click();
    const file = await download;
    const exported = await readFile(await file.path(), "utf8");
    assert(!exported.includes("SENSITIVE-EXAMPLE"));
    assert(!exported.includes("private-example"));
    await p.getByRole("button", { name: "Cancel", exact: true }).click();
    await p.locator("form button[type=submit]").click();
    await p.locator(".entry-feedback").waitFor();
    assert.equal(await p.locator("#entry-email").inputValue(), "");
    assert.equal(await p.evaluate(() => location.hash), hash);
    assert.match(
      await p.locator(".entry-feedback").innerText(),
      /no account, session or membership/,
    );
    const persisted = await p.evaluate(() =>
      JSON.stringify({
        local: { ...localStorage },
        session: { ...sessionStorage },
      }),
    );
    assert(!persisted.includes("SENSITIVE-EXAMPLE"));
    assert(!persisted.includes("private-example"));
    ok(id + " validation, reset, no auth success or input persistence/export");
  }
  await go("entry-challenge");
  await p.getByRole("button", { name: "Recover access", exact: true }).click();
  await p.locator("#confirm-review").click();
  await p.locator("#owner-ack").check();
  await p.locator("#save-review").click();
  await p.waitForFunction(() =>
    document.querySelector("#review-status")?.textContent.includes("FIX"),
  );
  await p.reload();
  await p.locator("#entry-proof").fill("ONE-EDIT");
  assert(!(await p.locator("#review-status").innerText()).includes("FIX"));
  ok("recovery and verification states cannot reuse FIX after reload/edit");
  await p.getByRole("button", { name: "Recover access", exact: true }).click();
  assert.equal(await p.locator("#entry-proof").inputValue(), "");
  ok("switching recovery variant clears fields");
  await go("entry-scope");
  assert.equal(await p.locator(".entry-workspace").count(), 2);
  assert(
    await p
      .getByRole("button", { name: "Open workspace preview" })
      .isDisabled(),
  );
  await p.locator("#workspace-search").fill("no-such-workspace");
  assert.equal(await p.locator(".entry-workspace").count(), 0);
  await p.locator("#workspace-search").fill("");
  await p.locator("input[value=sample-a]").check();
  await p.getByRole("button", { name: "Open workspace preview" }).click();
  await p.waitForURL(/#my-tasks/);
  assert((await p.evaluate(() => location.hash)).includes("scope=sample-a"));
  ok("workspace search/empty/selection and fixed preview destination");
  await go("entry-scope", "lang=en&mode=unconnected");
  assert.equal(await p.locator(".entry-workspace").count(), 0);
  assert(await p.locator("#workspace-search").isDisabled());
  ok("unconnected membership has no synthetic choices");
  await go("entry-login");
  await p.locator("#entry-workspace").fill("EPHEMERAL");
  await p.locator(".entry-back").click();
  await p.locator(".entry-main-action").waitFor();
  await p.goBack();
  await p.locator("#entry-workspace").waitFor();
  assert.equal(await p.locator("#entry-workspace").inputValue(), "");
  ok("browser back does not restore identity field contents");
  for (const lang of ["ko", "en", "ar"])
    for (const width of [1440, 390])
      for (const id of [
        "entry-start",
        "entry-login",
        "entry-bootstrap",
        "entry-invite",
        "entry-challenge",
        "entry-scope",
      ]) {
        await p.setViewportSize({ width, height: 1000 });
        await go(id, "lang=" + lang);
        await p.evaluate(() => document.fonts.ready);
        assert.equal(
          await p.locator("html").getAttribute("dir"),
          lang === "ar" ? "rtl" : "ltr",
        );
        assert(
          await p.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth + 1,
          ),
        );
        assert.equal(await p.locator(".tnb,.rail").count(), 0);
        assert(
          await p
            .locator(".brand-logo")
            .evaluate((e) => e.complete && e.naturalWidth > 0),
        );
        assert(await p.locator("#confirm-review").isVisible());
        assert(
          await p.evaluate(
            () =>
              document.querySelector("#content").getBoundingClientRect()
                .bottom <=
              document.querySelector(".design-review").getBoundingClientRect()
                .top,
          ),
        );
        await p.screenshot({
          path: `${out}/${id}-${lang}-${width}.png`,
          fullPage: true,
        });
        ok(`${id} ${lang} ${width} render`);
      }
  await p.setViewportSize({ width: 1440, height: 1000 });
  await go("entry-start");
  await p.locator("#language").focus();
  await p.keyboard.press("Enter");
  await p.keyboard.press("ArrowDown");
  await p.keyboard.press("Escape");
  assert.equal(await p.evaluate(() => document.activeElement.id), "language");
  ok("entry language keyboard focus return");
  const failed = await b.newContext();
  await failed.addInitScript(() => {
    Object.defineProperty(crypto.subtle, "digest", {
      value: () => Promise.reject(new Error("test digest failure")),
    });
  });
  const fp = await failed.newPage();
  await fp.goto(url + "#entry-start?lang=en");
  await fp.getByRole("alert").waitFor();
  assert(await fp.locator("#confirm-review").isDisabled());
  assert.match(
    await fp.getByRole("alert").innerText(),
    /fingerprint unavailable/,
  );
  await failed.close();
  ok("digest failure remains visible outside disabled review dialog");
  assert.equal(external.length, 0);
  assert.equal(errors.length, 0);
  assert(!JSON.stringify({ requests, logs }).includes("SENSITIVE-EXAMPLE"));
  assert(!JSON.stringify({ requests, logs }).includes("private-example"));
  ok("no runtime errors, external requests or input values in diagnostics");
  await writeFile(
    out + "/report.json",
    JSON.stringify({ checks, errors, external, browser: b.version() }, null, 2),
  );
  console.log(`${checks.length} entry checks PASS`);
} finally {
  await b.close();
}
