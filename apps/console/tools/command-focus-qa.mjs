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
const screens = JSON.parse(await readFile(new URL("../src/modules/administration/screens.json", import.meta.url), "utf8")).filter((s) => ["P02", "P03", "P04", "P05"].includes(s.id));
const frames = () => page.evaluate(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(done, 60))))));
const active = () => page.evaluate(() => { const a = document.activeElement; return { tag: a?.tagName ?? "", cls: a?.className ?? "", name: a?.getAttribute?.("aria-label") ?? "", opener: a?.dataset?.opener ?? "", text: (a?.textContent ?? "").trim().slice(0, 40), inForm: !!a?.closest("form, [role=dialog], dialog") }; });
const cancelName = /^(Cancel|Close)$/;
const cases = [];
const shot = new Set();

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
  const posts = [];
  page.on("request", (r) => { if (r.method() === "POST" && new URL(r.url()).pathname.includes("/commands/")) posts.push(new URL(r.url()).pathname); });

  for (const width of [1440, 390]) {
    await page.setViewportSize({ width, height: 1000 });
    for (const screen of screens) for (const tab of screen.tabs) {
      const prefix = screen.id.toLowerCase();
      const open = async () => {
        await page.goto("about:blank");
        await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#${screen.path}?lang=en&view=${tab.key}`);
        await page.locator(`.${prefix}-live`).waitFor();
        await page.waitForLoadState("networkidle");
        await page.waitForFunction((p) => !document.querySelector(`.${p}-live[aria-busy=true]`), prefix);
      };
      await open();
      const atLoad = await active();
      check(`${screen.id} ${tab.key} ${width}: no command form takes focus when the view loads`, (atLoad.tag === "BODY" || atLoad.tag === "MAIN") && !/form-(title|wrap)/.test(atLoad.cls), JSON.stringify(atLoad));
      const marks = await page.locator("[data-opener]:not([disabled])").evaluateAll((els) => { const seen = new Set(), out = []; for (const el of els) { const t = el.textContent.trim(); if (el.offsetParent === null || seen.has(t)) continue; seen.add(t); out.push({ mark: el.dataset.opener, text: t }); } return out.slice(0, 6); });
      for (const m of marks) {
        const at = `${screen.id} ${tab.key} ${width} “${m.text}”`;
        const opener = page.locator(`[data-opener="${m.mark}"]`).first();
        if ((await opener.count()) === 0) { await open(); }
        if ((await opener.count()) === 0) { notes.push(`${at}: opener not found again`); continue; }
        await opener.focus();
        const before = await page.getByRole("button", { name: cancelName }).count();
        await page.keyboard.press("Enter");
        await frames();
        const cancels = page.getByRole("button", { name: cancelName });
        if ((await cancels.count()) <= before) { notes.push(`${at}: Enter opened nothing with a Cancel button`); await open(); continue; }
        const opened = await active();
        const ring = await page.evaluate(() => { const el = document.activeElement; if (!el || !/form-title/.test(el.className)) return null; const r = document.createRange(); r.selectNodeContents(el); const t = r.getBoundingClientRect(), b = el.getBoundingClientRect(), p = el.parentElement.getBoundingClientRect(); return { extra: Math.round(b.width - t.width), inside: b.left >= p.left - 1 && b.right <= p.right + 1, offset: getComputedStyle(el).outlineOffset }; });
        if (ring) check(`${at}: the focus ring follows the title's text and stays inside the form`, ring.extra <= 2 && ring.inside && parseFloat(ring.offset) >= 2, JSON.stringify(ring));
        if (args.shots && !shot.has(screen.id + width)) { shot.add(screen.id + width); await mkdir(args.shots, { recursive: true }); await page.locator(":focus").scrollIntoViewIfNeeded().catch(() => {}); await page.screenshot({ path: `${args.shots}/form-opened-${screen.id}-${width}.png` }); }
        let reached = false, steps = 0;
        for (; steps < 80; steps++) {
          const a = await page.evaluate(() => { const el = document.activeElement; return { text: (el?.textContent ?? "").trim(), tag: el?.tagName, body: el === document.body }; });
          if (a.tag === "BUTTON" && cancelName.test(a.text)) { reached = true; break; }
          await page.keyboard.press("Tab");
        }
        check(`${at}: after opening, focus is on a control and Tab reaches Cancel`, opened.tag !== "BODY" && opened.tag !== "BUTTON" && reached, JSON.stringify({ opened, steps }));
        if (!reached) { await open(); continue; }
        await page.keyboard.press("Enter");
        await frames();
        const closed = await active();
        check(`${at}: Cancel returns focus to the control that opened the form`, closed.opener === m.mark, JSON.stringify(closed));
        cases.push({ screen: screen.id, view: tab.key, width, opener: m.text, afterOpen: opened, afterCancel: closed });
      }
    }
  }
  check("no command was sent", posts.length === 0, posts.join());
  check("no page error", errors.length === 0, errors.join(" | "));
  if (args.shots) { await mkdir(args.shots, { recursive: true }); await (await import("node:fs/promises")).writeFile(`${args.shots}/command-focus.json`, JSON.stringify(cases, null, 1)); }
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
