import { measurePresentation } from "./presentation-measure.mjs";
import { inspectView } from "./view-measure.mjs";
import { createRequire } from "node:module";
import { readFile, mkdir } from "node:fs/promises";
import { randomBytes } from "node:crypto";
const args = Object.fromEntries(process.argv.slice(2).flatMap((v, i, all) => (v.startsWith("--") ? [[v.slice(2), all[i + 1]]] : [])));
const { chromium } = createRequire(import.meta.url)(args["playwright-module"] || "playwright");
const base = (args.url || "http://127.0.0.1:4173").replace(/\/$/, "");
const browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 }, deviceScaleFactor: Number(args.scale ?? 1) });
const page = await context.newPage();
const results = [], errors = [], notes = [];
let navigation = 0, hovered = 0;
const check = (name, ok, detail = "") => { results.push(`${ok ? "PASS" : "FAIL"} ${name}${!ok && detail ? " " + String(detail).replace(/\s+/g, " ").slice(0, 400) : ""}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const ends = (url, path) => new URL(url).pathname.endsWith("/api/v1/" + path);
const load = async (name) => JSON.parse(await readFile(new URL(`../src/modules/${name}/screens.json`, import.meta.url), "utf8"));
const wanted = (args.screens ?? "M01,M02,M03,M04,M05,O01,O02,O03,P01,P02,P03,P04,P05").split(",");
const screens = [...(await load("identity")), ...(await load("organizations")), ...(await load("administration"))].filter((s) => wanted.includes(s.id));
const combos = (args.combos ?? "en:320,ko:320,ar:320,en:1440").split(",").map((c) => c.split(":"));
const records = [];

const inspect = async () => {
  if (args.plant) await page.evaluate(() => { const m = document.querySelector("main"); if (m && !m.querySelector("[data-qa-plant]")) { const p = document.createElement("p"); p.setAttribute("data-qa-plant", ""); p.textContent = "planted red on black"; p.style.cssText = "color:#ff0000;background:#000;font-size:14px"; m.prepend(p); } });
  return inspectView(page, args.vision ?? "");
};

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
    await page.evaluate(() => { const d = document.createElement("div"); d.id = "qa-vcut"; d.textContent = "vertical probe line one and line two and line three"; d.style.cssText = "inline-size: 80px; block-size: 12px; overflow: hidden"; document.querySelector("main").prepend(d); });
    const probe = await inspect();
    check("the detectors find a planted low-contrast text, a planted wide box, a planted cut-off text and a planted vertically cut text", probe.low.some((x) => x.el.includes("qa-low")) && probe.clipped.some((x) => x.el.includes("qa-wide")) && probe.cut.some((x) => x.el.includes("qa-cut")) && probe.vcut.some((x) => x.el.includes("qa-vcut")) && probe.overflow > 1, JSON.stringify({ low: probe.low.length, clipped: probe.clipped.length, cut: probe.cut.length, overflow: probe.overflow }));
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
      if (args.spacing) await page.addStyleTag({ content: "main *, main *::before, main *::after { line-height: 1.5 !important; letter-spacing: 0.12em !important; word-spacing: 0.16em !important; } main p { margin-block-end: 2em !important; }" });
      if (args.spacing) { const applied = await page.evaluate(() => { const e = document.querySelector("main p, main h1, main h2, main td, main span"); if (!e) return null; const c = getComputedStyle(e), f = parseFloat(c.fontSize); return { letter: parseFloat(c.letterSpacing) / f, line: parseFloat(c.lineHeight) / f }; }); check(`${at}: the text-spacing override is in effect`, !!applied && Math.abs(applied.letter - 0.12) < 0.01 && Math.abs(applied.line - 1.5) < 0.02, JSON.stringify(applied)); }
      await page.evaluate(() => document.fonts.ready);
      const state = await page.evaluate(() => ({ alerts: [...document.querySelectorAll("main [role=alert], main .live-error")].filter((e) => e.getClientRects().length && e.textContent.trim()).map((e) => e.textContent.trim().slice(0, 80)), signedIn: !!document.querySelector(".service-session"), words: (document.querySelector("main")?.innerText ?? "").trim().split(/\s+/).length }));
      check(`${at}: the view finished loading, signed in, with no error panel`, settled && state.signedIn && state.alerts.length === 0, JSON.stringify({ settled, ...state }));
      if (args.scale) check(`${at}: the page is laid out at device pixel ratio ${args.scale}`, (await page.evaluate(() => window.devicePixelRatio)) === Number(args.scale));
      if (args.hover) {
        if (args.plant) await page.evaluate(() => { const st = document.createElement("style"); st.textContent = ".qa-hp:hover{color:#c8c8c8 !important;background:#fff !important}"; document.head.append(st); const b = document.createElement("button"); b.type = "button"; b.className = "qa-hp"; b.textContent = "planted hover"; document.querySelector("main")?.prepend(b); });
        const marked = await page.evaluate(() => {
          const seen = new Set(); let n = 0;
          for (const e of document.querySelectorAll("main button, main a[href], main summary, main [role=tab]")) {
            const r = e.getBoundingClientRect();
            if (!r.width || !r.height || e.closest("[hidden], [inert], .visually-hidden") || e.matches(":disabled")) continue;
            const key = String(e.className) + "|" + e.tagName + "|" + (e.closest("table, ul, ol") ? "row" : e.textContent.trim().slice(0, 20));
            if (seen.has(key)) continue;
            seen.add(key); e.setAttribute("data-qa-h", String(n++));
            if (n >= 60) break;
          }
          return n;
        });
        const weak = [];
        for (let i = 0; i < marked; i++) {
          const el = page.locator(`[data-qa-h="${i}"]`);
          try { await el.scrollIntoViewIfNeeded({ timeout: 2000 }); await el.hover({ timeout: 2000, force: true }); } catch { continue; }
          await page.waitForTimeout(220);
          const got = await el.evaluate((e) => {
            const rgb = (v) => { const p = v.match(/[\d.]+/g)?.map(Number); return /^rgba?\(/.test(v) && p?.length >= 3 ? [p[0], p[1], p[2], p[3] ?? 1] : null; };
            const over = (a, b) => [0, 1, 2].map((i) => a[i] * a[3] + b[i] * (1 - a[3]));
            const lum = (c) => c.map((v) => { v /= 255; return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; }).reduce((s, v, i) => s + v * [0.2126, 0.7152, 0.0722][i], 0);
            const ratio = (a, b) => (Math.max(lum(a), lum(b)) + 0.05) / (Math.min(lum(a), lum(b)) + 0.05);
            if (!e.matches(":hover")) return null;
            const leaf = [...e.querySelectorAll("*")].find((n) => [...n.childNodes].some((t) => t.nodeType === 3 && t.textContent.trim())) ?? e;
            const c = getComputedStyle(leaf); let bg = [255, 255, 255];
            const chain = []; for (let n = leaf; n; n = n.parentElement) chain.unshift(n);
            for (const n of chain) { const s = getComputedStyle(n); if (s.backgroundImage !== "none" || Number(s.opacity) !== 1) return null; const p = rgb(s.backgroundColor); if (p) bg = over(p, bg); }
            const fg = rgb(c.color); if (!fg) return null;
            const size = parseFloat(c.fontSize), large = size >= 24 || (Number(c.fontWeight) >= 700 && size >= 18.66);
            const r = ratio(over(fg, bg), bg);
            return { ratio: Math.round(r * 100) / 100, need: large ? 3 : 4.5, el: e.tagName.toLowerCase() + "." + String(e.className).split(" ")[0], text: e.textContent.trim().slice(0, 24) };
          });
          if (got && got.ratio < got.need) weak.push(got);
        }
        await page.mouse.move(0, 0);
        hovered += marked;
        if (args.plant) check(`${at}: the planted low-contrast hover is detected`, weak.some((w) => w.el.includes("qa-hp")), JSON.stringify(weak.slice(0, 2)));
        else check(`${at}: text keeps 4.5:1 (3:1 large) while each control is hovered (${marked} controls)`, weak.length === 0, JSON.stringify(weak.slice(0, 3)));
      }
      const found = await inspect();
      const controls = (await measurePresentation(page)).results?.filter((r) => r.candidate).map((r) => ({ el: r.element.tag + "." + r.element.className.split(" ")[0], kind: r.kind, ratio: Math.round(r.ratio * 100) / 100, need: r.threshold })) ?? [];
      records.push({ screen: screen.id, view: tab.key, lang, width, settled, ...state, ...found, controls });
      check(`${at}: the page is not wider than the viewport`, found.overflow <= 1, String(found.overflow));
      check(`${at}: nothing extends past the viewport outside a scrolling region`, found.clipped.length === 0, JSON.stringify(found.clipped.slice(0, 3)));
      check(`${at}: nothing is cut off inside a box that hides its overflow`, found.cut.length === 0, JSON.stringify(found.cut.slice(0, 3)));
      check(`${at}: no text is cut off vertically by a box that hides its overflow or clamps its lines`, found.vcut.length === 0, JSON.stringify(found.vcut.slice(0, 3)));
      check(`${at}: text was measured`, found.measured > 0, `${found.measured} measured, ${found.unsure} not measurable`);
      check(`${at}: text contrast meets 4.5:1 (3:1 large)`, found.low.length === 0, JSON.stringify(found.low.slice(0, 3)));
      check(`${at}: control contrast candidates`, controls.length === 0, JSON.stringify(controls.slice(0, 3)));
      if (args.shots && (found.overflow > 1 || found.clipped.length || found.cut.length || found.vcut.length || found.low.length)) await page.screenshot({ path: `${args.shots}/${screen.id}-${tab.key}-${lang}-${width}.png`, fullPage: true });
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
  if (args.hover) console.log(`HOVERED ${hovered}`);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
