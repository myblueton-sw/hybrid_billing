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
const wanted = (args.screens ?? "M01,M02,M03,M04,M05,O01,O02,O03,P01").split(",");
const screens = [...(await load("identity")), ...(await load("organizations")), ...(await load("administration"))].filter((s) => wanted.includes(s.id));
const frames = () => page.evaluate(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(done, 120))))));
const cases = [];
let openedCount = 0;
const focus = { controls: 0, byOutline: 0, byShadow: 0 };
const lang = args.lang ?? "en";
const width = Number(args.width ?? 1440);

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
  await page.setViewportSize({ width, height: 1000 });

  for (const screen of screens) for (const tab of screen.tabs) {
    const open = async () => {
      await page.goto("about:blank");
      await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#${screen.path}?lang=${lang}&view=${tab.key}`);
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
      if (args.plant) await page.addStyleTag({ content: "main :focus-visible { outline: 1px solid #ddd !important; box-shadow: none !important; }" });
      const posts = blocked.length;
      await page.evaluate(() => { for (const n of document.querySelectorAll("dialog[open], [role=dialog], .live-panel, [class*=form-wrap], main form, main section, main aside")) n.setAttribute("data-qa-before", ""); });
      await el.focus();
      await page.keyboard.press("Enter");
      const left = await frames().then(() => false, () => true);
      if (left || !(await page.locator("main").count())) { notes.push(`${screen.id} ${tab.key} “${b.text}”: left the page`); await page.waitForLoadState("load").catch(() => {}); continue; }
      const after = await page.evaluate(() => { const a = document.activeElement; return { tag: a?.tagName ?? "", cls: String(a?.className ?? "").slice(0, 60), text: (a?.textContent ?? "").trim().replace(/\s+/g, " ").slice(0, 50), dialog: !!document.querySelector("dialog[open], [role=dialog]") }; });
      const sent = blocked.length > posts;
      const opened = await page.evaluate(() => [...document.querySelectorAll("dialog[open], [role=dialog], .live-panel, [class*=form-wrap], main form, main section, main aside")].some((n) => !n.hasAttribute("data-qa-before") && n.getClientRects().length > 0));
      if (opened) {
        const shown = await inspectView(page);
        const ring = await page.evaluate(() => {
          const rgb = (v) => { const p = v.match(/[\d.]+/g)?.map(Number); return /^rgba?\(/.test(v) && p?.length >= 3 ? [p[0], p[1], p[2], p[3] ?? 1] : null; };
          const over = (a, b) => [0, 1, 2].map((i) => a[i] * a[3] + b[i] * (1 - a[3]));
          const lum = (c) => c.map((v) => { v /= 255; return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; }).reduce((s, v, i) => s + v * [0.2126, 0.7152, 0.0722][i], 0);
          const ratio = (a, b) => (Math.max(lum(a), lum(b)) + 0.05) / (Math.min(lum(a), lum(b)) + 0.05);
          const around = (e) => { let bg = [255, 255, 255]; const chain = []; for (let n = e.parentElement; n; n = n.parentElement) chain.unshift(n); for (const n of chain) { const p = rgb(getComputedStyle(n).backgroundColor); if (p) bg = over(p, bg); } return bg; };
          const fresh = [...document.querySelectorAll("dialog[open], [role=dialog], .live-panel, [class*=form-wrap], main form, main section, main aside")].filter((n) => !n.hasAttribute("data-qa-before") && n.getClientRects().length > 0);
          const start = document.activeElement;
          const out = { controls: 0, byOutline: 0, byShadow: 0, weak: [] };
          const seen = new Set();
          for (const box of fresh) for (const e of box.querySelectorAll("input:not([type=hidden]), select, textarea, button, a[href], summary")) {
            if (seen.has(e) || e.matches(":disabled") || !e.getClientRects().length) continue;
            seen.add(e);
            e.focus({ preventScroll: true });
            if (document.activeElement !== e || !e.matches(":focus-visible")) continue;
            const c = getComputedStyle(e), w = parseFloat(c.outlineWidth), col = rgb(c.outlineColor), bg = around(e);
            const got = c.outlineStyle !== "none" && w > 0 && col ? ratio(over(col, bg), bg) : 0;
            out.controls++;
            if (w >= 2 && got >= 3) out.byOutline++;
            else if (c.boxShadow !== "none") out.byShadow++;
            else out.weak.push({ el: e.tagName.toLowerCase() + (e.id ? "#" + e.id : "") + "." + String(e.className).split(" ")[0], width: w, style: c.outlineStyle, ratio: Math.round(got * 100) / 100 });
          }
          start?.focus?.({ preventScroll: true });
          return out;
        });
        const where = `${screen.id} ${tab.key} ${width} ${lang} “${b.text}” opened`;
        check(`${where}: nothing overflows, is cut off or loses contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), low: shown.low.slice(0, 2) }));
        check(`${where}: every focusable control in it has a 2 px outline at 3:1 or a box shadow`, ring.weak.length === 0, JSON.stringify(ring.weak.slice(0, 3)));
        focus.controls += ring.controls; focus.byOutline += ring.byOutline; focus.byShadow += ring.byShadow;
        openedCount++;
      }
      if (!(await page.locator(".service-session").count())) { check(`${screen.id} ${tab.key} “${b.text}”: the session is still signed in`, false); break; }
      cases.push({ screen: screen.id, view: tab.key, button: b.text, after, sent });
    }
  }
  notes.push(`newly opened states measured: ${openedCount}; focusable controls measured: ${focus.controls} (outline at 3:1: ${focus.byOutline}; box shadow, contrast not measured: ${focus.byShadow})`);
  check(`at least one newly opened state was measured (${openedCount})`, openedCount > 0);
  check("no page error", errors.length === 0, errors.join(" | "));
  if (args.shots) { await mkdir(args.shots, { recursive: true }); await (await import("node:fs/promises")).writeFile(`${args.shots}/opened-states.json`, JSON.stringify({ cases, blocked }, null, 1)); }
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
