import { inspectColour } from "./view-measure.mjs";
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
const records = [];

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
  await page.setViewportSize({ width: Number(args.width ?? 1440), height: 900 });
  let probed = false;
  for (const screen of screens) for (const tab of screen.tabs) {
    const at = `${screen.id} ${tab.key}`;
    await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#${screen.path}?lang=en&view=${tab.key}`);
    await page.locator("main").waitFor();
    await page.waitForLoadState("networkidle");
    const settled = await page.waitForFunction(() => !document.querySelector("main [aria-busy=true]"), null, { timeout: 8000 }).then(() => true, () => false);
    const state = await page.evaluate(() => ({ alerts: [...document.querySelectorAll("main [role=alert], main .live-error")].filter((e) => e.getClientRects().length && e.textContent.trim()).length, signedIn: !!document.querySelector(".service-session") }));
    check(`${at}: the view finished loading, signed in, with no error panel`, settled && state.signedIn && state.alerts === 0, JSON.stringify({ settled, ...state }));
    if (!probed) {
      probed = true;
      await page.evaluate(() => { const m = document.querySelector("main"); const a = document.createElement("span"); a.id = "qa-dot"; a.style.cssText = "display:inline-block;inline-size:10px;block-size:10px;background:rgb(200,30,30)"; const b = document.createElement("div"); b.id = "qa-strip"; b.textContent = "strip probe"; b.style.cssText = "border-inline-start:4px solid rgb(20,90,200)"; m.append(a, b); });
      const probe = await inspectColour(page);
      check("the detectors find a planted colour-only mark and a planted accent strip", probe.marks.some((x) => x.el.includes("qa-dot")) && probe.strips.some((x) => x.el.includes("qa-strip")), JSON.stringify(probe));
      await page.evaluate(() => { for (const id of ["qa-dot", "qa-strip"]) document.getElementById(id)?.remove(); });
    }
    const found = await inspectColour(page);
    records.push({ screen: screen.id, view: tab.key, ...found });
    check(`${at}: no coloured mark without text or a name`, found.marks.length === 0, JSON.stringify(found.marks.slice(0, 3)));
    check(`${at}: no coloured edge strip`, found.strips.length === 0, JSON.stringify(found.strips.slice(0, 3)));
    if (args.shots && (found.marks.length || found.strips.length)) await page.screenshot({ path: `${args.shots}/${screen.id}-${tab.key}.png`, fullPage: true });
  }
  check("the planted-defect probe ran", probed);
  check("no page error", errors.length === 0, errors.join(" | "));
  if (args.shots) await (await import("node:fs/promises")).writeFile(`${args.shots}/colour.json`, JSON.stringify(records, null, 1));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) if (!args.quiet || r.startsWith("FAIL")) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
