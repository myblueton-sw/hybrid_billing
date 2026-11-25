import { measurePresentation } from "./presentation-measure.mjs";
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
const load = async (name) => JSON.parse(await readFile(new URL(`../src/modules/${name}/screens.json`, import.meta.url), "utf8"));
const wanted = (args.screens ?? "M01,M02,M03,M04,M05,O01,O02,O03,P01,P02,P03,P04,P05").split(",");
const screens = [...(await load("identity")), ...(await load("organizations")), ...(await load("administration"))].filter((s) => wanted.includes(s.id));
const combos = (args.combos ?? "en:320,ko:320,ar:320,en:1440").split(",").map((c) => c.split(":"));
const records = [];

const inspect = () => inspectView(page);

try {
  const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
  const activation = new URL(lines.find((l) => l.startsWith("admin@example.invalid ")).split(" ")[1]);
  await page.goto(`${base}/activate?lang=en${activation.hash}`);
  await page.locator("#link-new").fill(password); await page.locator("#link-confirm").fill(password);
  await page.locator("form button[type=submit]").click(); await page.locator("a.service-button-link").waitFor();
  await page.goto(`${base}/?service=api&lang=en`);
  await page.locator("#signin-workspace").fill("sample-workspace"); await page.locator("#signin-email").fill("admin@example.invalid");
  await page.locator("#signin-password").fill(password); await page.locator("form button[type=submit]").click();
  await page.locator(".service-session").waitFor();
  if (args.shots) await mkdir(args.shots, { recursive: true });

  {
    await page.setViewportSize({ width: 320, height: 900 });
    await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#${screens[0].path}?lang=en&view=${screens[0].tabs[0].key}`);
    await page.locator("main").waitFor(); await page.waitForLoadState("networkidle");
    await page.evaluate(() => { const m = document.querySelector("main"); const a = document.createElement("p"); a.id = "qa-low"; a.textContent = "low contrast probe"; a.style.cssText = "color: rgb(170,170,170); background: rgb(255,255,255)"; const b = document.createElement("div"); b.id = "qa-wide"; b.textContent = "wide probe"; b.style.cssText = "inline-size: 900px"; const c = document.createElement("div"); c.id = "qa-cut"; c.style.cssText = "inline-size: 100px; overflow: hidden; white-space: nowrap"; c.textContent = "a probe that is certainly longer than one hundred pixels"; m.append(a, b, c); });
    const probe = await inspect();
    check("the detectors find a planted low-contrast text, a planted wide box and a planted cut-off text", probe.low.some((x) => x.el.includes("qa-low")) && probe.clipped.some((x) => x.el.includes("qa-wide")) && probe.cut.some((x) => x.el.includes("qa-cut")) && probe.overflow > 1, JSON.stringify({ low: probe.low.length, clipped: probe.clipped.length, cut: probe.cut.length, overflow: probe.overflow }));
  }
  for (const [lang, w] of combos) {
    const width = Number(w);
    await page.setViewportSize({ width, height: 900 });
    for (const screen of screens) for (const tab of screen.tabs) {
      const at = `${screen.id} ${tab.key} ${lang} ${width}`;
      await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#${screen.path}?lang=${lang}&view=${tab.key}`);
      await page.locator("main").waitFor();
      await page.waitForLoadState("networkidle");
      const settled = await page.waitForFunction(() => !document.querySelector("main [aria-busy=true]"), null, { timeout: 8000 }).then(() => true, () => false);
      await page.evaluate(() => document.fonts.ready);
      const state = await page.evaluate(() => ({ alerts: [...document.querySelectorAll("main [role=alert], main .live-error")].filter((e) => e.getClientRects().length && e.textContent.trim()).map((e) => e.textContent.trim().slice(0, 80)), signedIn: !!document.querySelector(".service-session"), words: (document.querySelector("main")?.innerText ?? "").trim().split(/\s+/).length }));
      check(`${at}: the view finished loading, signed in, with no error panel`, settled && state.signedIn && state.alerts.length === 0, JSON.stringify({ settled, ...state }));
      const found = await inspect();
      const controls = (await measurePresentation(page)).results?.filter((r) => r.candidate).map((r) => ({ el: r.element.tag + "." + r.element.className.split(" ")[0], kind: r.kind, ratio: Math.round(r.ratio * 100) / 100, need: r.threshold })) ?? [];
      records.push({ screen: screen.id, view: tab.key, lang, width, settled, ...state, ...found, controls });
      check(`${at}: the page is not wider than the viewport`, found.overflow <= 1, String(found.overflow));
      check(`${at}: nothing extends past the viewport outside a scrolling region`, found.clipped.length === 0, JSON.stringify(found.clipped.slice(0, 3)));
      check(`${at}: nothing is cut off inside a box that hides its overflow`, found.cut.length === 0, JSON.stringify(found.cut.slice(0, 3)));
      check(`${at}: text was measured`, found.measured > 0, `${found.measured} measured, ${found.unsure} not measurable`);
      check(`${at}: text contrast meets 4.5:1 (3:1 large)`, found.low.length === 0, JSON.stringify(found.low.slice(0, 3)));
      check(`${at}: control contrast candidates`, controls.length === 0, JSON.stringify(controls.slice(0, 3)));
      if (args.shots && (found.overflow > 1 || found.clipped.length || found.cut.length || found.low.length)) await page.screenshot({ path: `${args.shots}/${screen.id}-${tab.key}-${lang}-${width}.png`, fullPage: true });
    }
  }
  check("no page error", errors.length === 0, errors.join(" | "));
  if (args.shots) await (await import("node:fs/promises")).writeFile(`${args.shots}/views.json`, JSON.stringify(records, null, 1));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) if (!args.quiet || r.startsWith("FAIL")) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
