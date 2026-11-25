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
const wanted = (args.screens ?? "M01,M02,M03,M04,M05,O01,O02,O03,P01").split(",");
const screens = [...(await load("identity")), ...(await load("organizations")), ...(await load("administration"))].filter((s) => wanted.includes(s.id));
const frames = () => page.evaluate(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(done, 120))))));
const cases = [];

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
  const blocked = [];
  await page.route((url) => url.pathname.includes("/api/"), (route) => { if (route.request().method() === "GET") return route.continue(); blocked.push(new URL(route.request().url()).pathname); return route.abort(); });
  await page.setViewportSize({ width: Number(args.width ?? 1440), height: 1000 });

  for (const screen of screens) for (const tab of screen.tabs) {
    const open = async () => {
      await page.goto("about:blank");
      await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#${screen.path}?lang=en&view=${tab.key}`);
      await page.locator("main").waitFor();
      await page.waitForLoadState("networkidle");
      await page.waitForFunction(() => !document.querySelector("main [aria-busy=true]"), null, { timeout: 8000 }).catch(() => {});
    };
    const list = () => page.locator("main button:not([disabled]):not([role=tab])").evaluateAll((els) => { const seen = new Set(), out = []; els.forEach((el, index) => { const t = el.textContent.trim().replace(/\s+/g, " "); if (el.offsetParent === null || !t || seen.has(t) || el.closest("nav, [role=tablist]") || /reload/i.test(el.className)) return; seen.add(t); out.push({ index, text: t.slice(0, 60) }); }); return out.slice(0, 10); });
    await open();
    const buttons = await list();
    for (const b of buttons) {
      await open();
      const again = (await list()).find((x) => x.text === b.text);
      if (!again) { notes.push(`${screen.id} ${tab.key} “${b.text}”: not found after reload`); continue; }
      const el = page.locator("main button:not([disabled]):not([role=tab])").nth(again.index);
      const posts = blocked.length;
      await el.focus();
      await page.keyboard.press("Enter");
      const left = await frames().then(() => false, () => true);
      if (left || !(await page.locator("main").count())) { notes.push(`${screen.id} ${tab.key} “${b.text}”: left the page`); await page.waitForLoadState("load").catch(() => {}); continue; }
      const after = await page.evaluate(() => { const a = document.activeElement; return { tag: a?.tagName ?? "", cls: String(a?.className ?? "").slice(0, 60), text: (a?.textContent ?? "").trim().replace(/\s+/g, " ").slice(0, 50), dialog: !!document.querySelector("dialog[open], [role=dialog]") }; });
      const sent = blocked.length > posts;
      if (!(await page.locator(".service-session").count())) { check(`${screen.id} ${tab.key} “${b.text}”: the session is still signed in`, false); break; }
      cases.push({ screen: screen.id, view: tab.key, button: b.text, after, sent });
      check(`${screen.id} ${tab.key} “${b.text}”: focus stays on a control after Enter${sent ? " (a command was attempted and blocked)" : ""}`, after.tag !== "BODY" && after.tag !== "", JSON.stringify(after));
    }
  }
  check("no page error", errors.length === 0, errors.join(" | "));
  if (args.shots) { await mkdir(args.shots, { recursive: true }); await (await import("node:fs/promises")).writeFile(`${args.shots}/press-focus.json`, JSON.stringify({ cases, blocked }, null, 1)); }
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
