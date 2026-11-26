import { inspectView } from "./view-measure.mjs";
import { createRequire } from "node:module";
import { readFile, mkdir } from "node:fs/promises";
import { randomBytes } from "node:crypto";
const args = Object.fromEntries(process.argv.slice(2).flatMap((v, i, all) => (v.startsWith("--") ? [[v.slice(2), all[i + 1]]] : [])));
const { chromium } = createRequire(import.meta.url)(args["playwright-module"] || "playwright");
const base = (args.url || "http://127.0.0.1:4173").replace(/\/$/, "");
const browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
const page = await context.newPage();
const results = [], errors = [], notes = [];
let navigation = 0;
const check = (name, ok, detail = "") => { results.push(`${ok ? "PASS" : "FAIL"} ${name}${!ok && detail ? " " + String(detail).replace(/\s+/g, " ").slice(0, 400) : ""}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const ends = (url, path) => new URL(url).pathname.endsWith("/api/v1/" + path);
const copy = {
  en: { short: "Some access the console may grant may not be listed here.", none: "No access to grant could be listed:", kinds: "Some kinds of data the console may use may not be listed here.", noKinds: "No kind of data could be listed:" },
  ko: { short: "콘솔이 부여할 수 있는 권한 중 일부가 여기에 표시되지 않았을 수 있습니다.", none: "부여할 권한을 표시하지 못했습니다.", kinds: "콘솔이 사용할 수 있는 데이터 종류 중 일부가 여기에 표시되지 않았을 수 있습니다.", noKinds: "데이터 종류를 표시하지 못했습니다." },
  ar: { short: "قد لا تظهر هنا بعض الصلاحيات التي يجوز لوحدة التحكم منحها.", none: "تعذّر عرض أي صلاحية للمنح:", kinds: "قد لا تظهر هنا بعض أنواع البيانات التي يجوز لوحدة التحكم استخدامها.", noKinds: "تعذّر عرض أي نوع بيانات:" },
};
const isWhole = (url) => new URL(url).pathname === "/api/v1/identity/access";

try {
  if (args.shots) await mkdir(args.shots, { recursive: true });
  const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
  const activation = new URL(lines.find((l) => l.startsWith("admin@example.invalid ")).split(" ")[1]);
  await page.goto(`${base}/activate?lang=en${activation.hash}`);
  await page.locator("#link-new").fill(password); await page.locator("#link-confirm").fill(password);
  await page.locator("form button[type=submit]").click(); await page.locator("a.service-button-link").waitFor();
  await page.goto(`${base}/?service=api&lang=en`);
  await page.locator("#signin-workspace").fill("sample-workspace"); await page.locator("#signin-email").fill("admin@example.invalid");
  await page.locator("#signin-password").fill(password); await page.locator("form button[type=submit]").click();
  await page.locator(".service-session").waitFor();
  const open = async (lang, view = "grants") => {
    await page.goto("about:blank");
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/access?${new URLSearchParams({ lang, view })}`);
    await page.locator(view === "grants" ? "#access-list" : '[data-live="effects"]').waitFor();
    if (view === "grants") await page.locator(".access-count, .access-list-failure").first().waitFor();
  };
  await page.setViewportSize({ width: 1440, height: 1000 });
  let real = null;
  page.on("response", async (r) => { if (isWhole(r.url()) && r.status() === 200 && !real) real = await r.json().catch(() => null); });
  await open("en");
  await page.locator("#access-capability").waitFor();
  check("the real service sends a complete catalog and the form shows no shortage sentence", real?.catalog_incomplete === false && real.catalog.length > 0 && (await page.locator("[data-catalog-short], [data-catalog-none]").count()) === 0 && (await page.locator("#access-capability").getAttribute("aria-describedby")) === "access-capability-hint", JSON.stringify(real && { flag: real.catalog_incomplete, n: real.catalog.length }));

  for (const [lang, width] of [["en", 1440], ["ko", 1440], ["ar", 1440], ["en", 390], ["ko", 390], ["ar", 390]]) {
    const at = `${lang} ${width}`;
    await page.setViewportSize({ width, height: 1000 });
    let mode = "short";
    await page.route((url) => isWhole(url.toString()), async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      await route.fulfill({ response, json: { ...body, catalog_incomplete: true, catalog: mode === "none" ? [] : body.catalog } });
    });
    await open(lang);
    if (width === 390) await page.locator(".access-grant-open").click();
    await page.locator("#access-capability").waitFor();
    check(`${at}: the request form says some access could not be listed, as a hint tied to the Access field`, (await page.locator("#access-capability-short").innerText()).startsWith(copy[lang].short) && (await page.locator("#access-capability").getAttribute("aria-describedby")) === "access-capability-hint access-capability-short" && (await page.locator("#access-capability-short").evaluate((el) => el.classList.contains("service-hint") && getComputedStyle(el).fontWeight < 600)) && (await page.locator("#access-capability option").count()) > 1);
    const shown = await inspectView(page);
    check(`${at}: nothing overflows, is cut off or loses contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), low: shown.low.slice(0, 2) }));
    if (args.shots) await page.locator(".live-panel").screenshot({ path: `${args.shots}/m03-catalog-short-${lang}-${width}.png` });
    await open(lang, "effects");
    await page.locator("#effect-kind-short").waitFor();
    check(`${at}: the effect check says some kinds of data could not be listed`, (await page.locator("#effect-kind-short").innerText()).startsWith(copy[lang].kinds));
    if (args.shots && width === 1440) await page.locator('[data-live="effects"]').screenshot({ path: `${args.shots}/m03-catalog-effects-${lang}-${width}.png` });
    mode = "none";
    await open(lang);
    await page.locator("[data-catalog-none]").waitFor();
    check(`${at}: with nothing listed the view says so in place of the form, offers no request button, and does not claim there is nothing to grant`, (await page.locator("[data-catalog-none]").innerText()).startsWith(copy[lang].none) && (await page.locator("#access-capability, .access-grant-open").count()) === 0);
    if (args.shots && lang === "en") await page.locator('section[aria-labelledby="access-list"]').screenshot({ path: `${args.shots}/m03-catalog-none-${lang}-${width}.png`, clip: undefined });
    await open(lang, "effects");
    await page.locator("#effect-no-kind").waitFor();
    check(`${at}: with no kind listed the effect check says the catalog could not be listed rather than asking for a reload`, (await page.locator("#effect-no-kind").innerText()).startsWith(copy[lang].noKinds));
    await page.unrouteAll({ behavior: "wait" });
  }
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) if (!args.quiet || r.startsWith("FAIL")) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
