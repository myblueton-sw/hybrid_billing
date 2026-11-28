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
const screens = JSON.parse(await readFile(new URL("../src/modules/administration/screens.json", import.meta.url), "utf8")).filter((s) => (args.screens ?? "P02,P03,P04,P05").split(",").includes(s.id));
const combos = (args.combos ?? "en:320,ar:320,en:1440").split(",").map((c) => c.split(":"));
const frames = () => page.evaluate(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(done, 60))))));
let picked = 0;
const records = [], submitted = [], confirmed = [], unreached = [], notes2 = [], shot = new Set(), outcomes = [];
let probed = false;
const rings = () => page.evaluate(async () => {
  const rgb = (v) => { const p = v.match(/[\d.]+/g)?.map(Number); return /^rgba?\(/.test(v) && p?.length >= 3 ? [p[0], p[1], p[2], p[3] ?? 1] : null; };
  const over = (a, b) => [0, 1, 2].map((i) => a[i] * a[3] + b[i] * (1 - a[3]));
  const lum = (c) => c.map((v) => { v /= 255; return v <= 0.04045 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4; }).reduce((s, v, i) => s + v * [0.2126, 0.7152, 0.0722][i], 0);
  const ratio = (a, b) => (Math.max(lum(a), lum(b)) + 0.05) / (Math.min(lum(a), lum(b)) + 0.05);
  const around = (e) => { let bg = [255, 255, 255]; const chain = []; for (let n = e.parentElement; n; n = n.parentElement) chain.unshift(n); for (const n of chain) { const p = rgb(getComputedStyle(n).backgroundColor); if (p) bg = over(p, bg); } return bg; };
  const name = (e) => `${e.tagName.toLowerCase()}${e.id ? "#" + e.id : ""}${typeof e.className === "string" && e.className ? "." + e.className.trim().split(/\s+/)[0] : ""}`;
  const wrap = document.activeElement?.closest("[class*=form-wrap]") ?? document.querySelector("[class*=form-wrap]");
  const out = { controls: 0, shown: 0, byOutline: 0, byShadow: 0, titles: 0, weak: [], placeholders: 0, faint: [] };
  if (!wrap) return out;
  const start = document.activeElement;
  for (const e of wrap.querySelectorAll("input:not([type=hidden]), select, textarea, button, a[href], [tabindex]")) {
    if (e.matches(":disabled") || !e.getClientRects().length) continue;
    e.focus({ preventScroll: true });
    if (document.activeElement !== e) continue;
    const visible = e.matches(":focus-visible");
    const c = getComputedStyle(e), w = parseFloat(c.outlineWidth), col = rgb(c.outlineColor), bg = around(e);
    out.controls++; if (visible) out.shown++;
    const got = c.outlineStyle !== "none" && w > 0 && col ? ratio(over(col, bg), bg) : 0;
    if (e.getAttribute("tabindex") === "-1") out.titles++;
    if (visible && w >= 2 && got >= 3) out.byOutline++; else if (visible && c.boxShadow !== "none") out.byShadow++;
    if (visible && (w < 2 || got < 3) && c.boxShadow === "none") out.weak.push({ el: name(e), width: w, style: c.outlineStyle, ratio: Math.round(got * 100) / 100 });
    if ((e.tagName === "INPUT" || e.tagName === "TEXTAREA") && e.getAttribute("placeholder")) {
      out.placeholders++;
      const pc = rgb(getComputedStyle(e, "::placeholder").color), own = rgb(c.backgroundColor), back = own ? over(own, bg) : bg;
      const pr = pc ? ratio(over(pc, back), back) : 0;
      if (pr < 4.5) out.faint.push({ el: name(e), ratio: Math.round(pr * 100) / 100 });
    }
  }
  start?.focus?.({ preventScroll: true });
  return out;
});

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
  const answered = [];
  const wording = { nothing: ["Nothing changed", "لم يتغيّر شيء"], pending: ["may have been applied", "ربما طُبّق"], again: ["confirm again", "أكّد مرة أخرى"] };
  const reply = { refused: [422, "prerequisite_failed", "correct_request"], conflict: [409, "revision_conflict", "reload"], unavailable: [503, "dependency_unavailable", "retry"] };
  await page.route((url) => url.pathname.includes("/api/"), (route) => {
    if (route.request().method() === "GET") return route.continue();
    const path = new URL(route.request().url()).pathname;
    if (args.outcome && path.includes("/commands/")) {
      answered.push(path);
      const answer = args.answer ?? args.outcome;
      if (answer === "unknown") return route.abort("timedout");
      const [status, code, next] = reply[answer];
      return route.fulfill({ status, json: { code, retry: status === 503, next_action: next, request_id: "qa-outcome" } });
    }
    posts.push(path);
    return route.abort();
  });
  if (args.shots) await mkdir(args.shots, { recursive: true });

  for (const [lang, w] of combos) {
    const width = Number(w);
    await page.setViewportSize({ width, height: 900 });
    for (const screen of screens) for (const tab of screen.tabs) {
      const prefix = screen.id.toLowerCase();
      const open = async () => {
        await page.goto("about:blank");
        await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#${screen.path}?lang=${lang}&view=${tab.key}`);
        await page.locator(`.${prefix}-live`).waitFor();
        await page.waitForLoadState("networkidle");
        await page.waitForFunction((p) => !document.querySelector(`.${p}-live[aria-busy=true]`), prefix);
      };
      await open();
      const kinds = await page.locator("[data-opener]:not([disabled])").evaluateAll((els) => { const seen = new Set(), out = []; els.forEach((el) => { const k = el.dataset.opener.split("|").pop(); if (el.offsetParent === null || seen.has(k)) return; seen.add(k); out.push({ kind: k, mark: el.dataset.opener }); }); return out.slice(0, 6); });
      for (const m of kinds) {
        const at = `${screen.id} ${tab.key} ${lang} ${width} ${m.kind}`;
        let opener = page.locator(`[data-opener="${m.mark}"]`).first();
        if ((await opener.count()) === 0) { await open(); opener = page.locator(`[data-opener$="|${m.kind}"]`).first(); }
        if ((await opener.count()) === 0) { notes.push(`${at}: opener not found again`); continue; }
        await opener.focus();
        await page.keyboard.press("Enter");
        await frames();
        const opened = await page.evaluate(() => { const a = document.activeElement; return !!a && /form-(title|wrap)/.test(String(a.className)); });
        if (!opened) { notes.push(`${at}: no form took focus`); await open(); continue; }
        await page.evaluate(() => document.fonts.ready);
        if (!probed) {
          probed = true;
          await page.evaluate(() => { const wrap = document.activeElement.closest("[class*=form-wrap]") ?? document.querySelector("[class*=form-wrap]"); const st = document.createElement("style"); st.id = "qa-style"; st.textContent = "#qa-ring:focus-visible{outline:1px solid rgb(225,225,225)!important;box-shadow:none!important}#qa-ph::placeholder{color:rgb(190,190,190)!important}"; const b = document.createElement("button"); b.type = "button"; b.id = "qa-ring"; b.textContent = "probe"; const i = document.createElement("input"); i.id = "qa-ph"; i.placeholder = "probe"; wrap.append(st, b, i); });
          const probe = await rings();
          check("the detectors find a planted weak focus indicator and a planted faint placeholder", probe.weak.some((x) => x.el.includes("qa-ring")) && probe.faint.some((x) => x.el.includes("qa-ph")), JSON.stringify(probe));
          await page.evaluate(() => { for (const id of ["qa-style", "qa-ring", "qa-ph"]) document.getElementById(id)?.remove(); });
        }
        const found = await inspectView(page), ring = await rings();
        records.push({ screen: screen.id, view: tab.key, lang, width, kind: m.kind, ...found, ...ring });
        check(`${at}: with the form open the page is not wider than the viewport`, found.overflow <= 1, String(found.overflow));
        check(`${at}: nothing in the open form extends past the viewport or is cut off`, found.clipped.length === 0 && found.cut.length === 0, JSON.stringify([...found.clipped, ...found.cut].slice(0, 3)));
        check(`${at}: text contrast with the form open`, found.low.length === 0, JSON.stringify(found.low.slice(0, 3)));
        check(`${at}: every control of the form shows a focus indicator of 2px and 3:1`, ring.controls > 0 && ring.shown === ring.controls && ring.weak.length === 0, JSON.stringify({ controls: ring.controls, shown: ring.shown, weak: ring.weak.slice(0, 3) }));
        check(`${at}: placeholders meet 4.5:1`, ring.faint.length === 0, JSON.stringify(ring.faint.slice(0, 3)));
        if (args.shots && (found.overflow > 1 || found.clipped.length || found.cut.length || found.low.length || ring.weak.length || ring.faint.length)) await page.screenshot({ path: `${args.shots}/${screen.id}-${tab.key}-${m.kind}-${lang}-${width}.png`, fullPage: true });
        if (args.confirm) {
          const fill = (value) => page.evaluate((v) => { const wrap = document.querySelector("[class*=form-wrap]"); const set = (el, val) => { const proto = el.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : el.tagName === "SELECT" ? HTMLSelectElement.prototype : HTMLInputElement.prototype; Object.getOwnPropertyDescriptor(proto, "value").set.call(el, val); el.dispatchEvent(new Event(el.tagName === "SELECT" ? "change" : "input", { bubbles: true })); }; let n = 0; for (const el of wrap.querySelectorAll("input:not([type=hidden]):not([type=search]):not([readonly]), textarea, select")) { if (el.disabled || !el.getClientRects().length) continue; const bad = el.getAttribute("aria-invalid") === "true"; if (v !== null && !bad) continue; if (el.tagName === "SELECT") { const opt = [...el.options].find((o) => o.value && !o.disabled); if (opt && !el.value) { set(el, opt.value); n++; } continue; } if (el.type === "checkbox" || el.type === "radio") { if (!el.checked) { el.click(); n++; } continue; } set(el, v ?? "Keyboard check of the confirmation step"); n++; } return n; }, value);
          const press = async () => { const sub = page.locator("[class*=form-wrap] button[type=submit]").first(); if ((await sub.count()) === 0) return false; await sub.focus(); await page.keyboard.press("Enter"); await frames(); return true; };
          const atConfirm = () => page.evaluate(() => !!document.querySelector("[class*=form-wrap] [class*=confirm]"));
          let tries = 0;
          await fill(null);
          for (const next of ["30", "2027-01-01", "sample"]) { tries++; if (!(await press())) break; if (await atConfirm()) break; if ((await fill(next)) === 0) break; }
          if (!(await atConfirm()) && (await page.locator("[class*=form-wrap] .p04-owner-results button").count()) > 0) {
            const person = page.locator("[class*=form-wrap] .p04-owner-results button").first();
            await person.focus(); await page.keyboard.press("Enter"); await frames(); picked++;
          }
          if (!(await atConfirm())) { await press(); }
          const reached = await atConfirm();
          if (!reached) { unreached.push(at); const why = await page.evaluate(() => [...document.querySelectorAll("[class*=form-wrap] .service-field-error, [class*=form-wrap] [role=alert]")].map((e) => e.textContent.trim().slice(0, 60)).slice(0, 3)); notes2.push(`${at}: ${why.join(" | ")}`); await open(); continue; }
          const focus = await page.evaluate(() => { const a = document.activeElement, wrap = document.querySelector("[class*=form-wrap]"); return { tag: a?.tagName ?? "", cls: String(a?.className ?? "").slice(0, 40), inside: !!(wrap && a && wrap.contains(a)) }; });
          const shown = await inspectView(page), ring = await rings();
          confirmed.push({ screen: screen.id, view: tab.key, lang, width, kind: m.kind, tries, focus, overflow: shown.overflow, clipped: shown.clipped, cut: shown.cut, low: shown.low, controls: ring.controls, shownRings: ring.shown, byOutline: ring.byOutline, weak: ring.weak });
          check(`${at}: on reaching the confirmation, focus is on its title inside the form`, focus.inside && /^H[1-6]$/.test(focus.tag), JSON.stringify(focus));
          check(`${at}: the confirmation does not overflow, cut off or lose text contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ overflow: shown.overflow, clipped: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), low: shown.low.slice(0, 3) }));
          check(`${at}: every control of the confirmation shows a 2px, 3:1 outline`, ring.controls > 0 && ring.shown === ring.controls && ring.byOutline === ring.controls, JSON.stringify({ controls: ring.controls, shown: ring.shown, byOutline: ring.byOutline, weak: ring.weak.slice(0, 2) }));
          if (args.outcome && reached) {
            const final = page.locator("[class*=form-wrap] [class*=confirm] .live-primary:not([disabled])").first();
            if ((await final.count()) === 0) { notes2.push(`${at}: no final button`); }
            else {
              const before = answered.length;
              const earlier = await page.evaluate(() => [...document.querySelectorAll("main [role=alert], main [role=status], main [aria-live]")].map((e) => e.textContent.trim().replace(/\s+/g, " ").slice(0, 90)));
              await final.focus(); await page.keyboard.press("Enter");
              await page.waitForFunction(() => !document.querySelector("[aria-busy=true]"), null, { timeout: 8000 }).catch(() => {});
              await frames(); await page.waitForTimeout(300);
              const sent = answered.length - before;
              const state = await page.evaluate(() => {
                const a = document.activeElement, r = a?.getBoundingClientRect();
                const region = a?.closest?.("[role=alert], [role=status], [aria-live]");
                const live = region ? [region.textContent.trim().replace(/\s+/g, " ").slice(0, 90)].filter(Boolean) : [];
                return { tag: a?.tagName ?? "", text: (a?.textContent ?? "").trim().replace(/\s+/g, " ").slice(0, 60), visible: !!r && r.width > 0 && r.height > 0, role: region?.getAttribute("role") ?? region?.getAttribute("aria-live") ?? "", live, liveCount: live.length };
              });
              const after = await inspectView(page);
              outcomes.push({ screen: screen.id, view: tab.key, lang, width, kind: m.kind, outcome: args.outcome, sent, ...state, overflow: after.overflow, clipped: after.clipped, cut: after.cut, low: after.low });
              check(`${at}: the final press sends exactly one command (answered ${args.outcome})`, sent === 1, String(sent));
              check(`${at}: after the ${args.outcome} answer focus is on a visible element, not the page body`, state.tag !== "BODY" && state.tag !== "" && state.visible, JSON.stringify(state));
              const said = state.live[0] ?? "";
              const has = (key) => wording[key].some((w) => said.includes(w));
              const fits = { refused: has("nothing") && !has("pending"), conflict: has("again") && !has("pending"), unavailable: has("pending"), unknown: has("pending") }[args.outcome];
              check(`${at}: the ${args.outcome} answer is stated in a focused live region or alert that is new since the press`, state.liveCount > 0 && !earlier.includes(said), JSON.stringify({ said, earlier: earlier.slice(0, 3) }));
              check(`${at}: the focused text states the ${args.outcome} outcome`, fits, said);
              check(`${at}: the ${args.outcome} state does not overflow, cut off or lose text contrast`, after.overflow <= 1 && after.clipped.length === 0 && after.cut.length === 0 && after.low.length === 0, JSON.stringify({ o: after.overflow, c: after.clipped.slice(0, 2), cut: after.cut.slice(0, 2), low: after.low.slice(0, 2) }));
              if (args.shots && !shot.has(screen.id + args.outcome + lang + width)) { shot.add(screen.id + args.outcome + lang + width); await page.screenshot({ path: `${args.shots}/${screen.id}-${args.outcome}-${lang}-${width}.png`, fullPage: false }); }
            }
          }
          if (args.shots && !shot.has(screen.id + "confirm" + lang + width)) { shot.add(screen.id + "confirm" + lang + width); await page.screenshot({ path: `${args.shots}/${screen.id}-confirm-${lang}-${width}.png`, fullPage: false }); }
          await open();
          continue;
        }
        if (args.submit) {
          const sub = page.locator("[class*=form-wrap] button[type=submit]").first();
          if ((await sub.count()) === 0) { notes.push(`${at}: the open form has no submit button`); await open(); continue; }
          await sub.focus();
          await page.keyboard.press("Enter");
          await frames();
          const after = await page.evaluate(() => { const a = document.activeElement, wrap = document.querySelector("[class*=form-wrap]"); const errs = wrap ? [...wrap.querySelectorAll(".service-field-error, [role=alert], [aria-invalid=true]")].filter((e) => e.getClientRects().length) : []; return { tag: a?.tagName ?? "", cls: String(a?.className ?? "").slice(0, 50), inside: !!(wrap && a && wrap.contains(a)), errors: errs.length, invalid: wrap ? wrap.querySelectorAll("[aria-invalid=true]").length : 0, described: wrap ? [...wrap.querySelectorAll("[aria-invalid=true]")].every((f) => (f.getAttribute("aria-describedby") ?? "").split(" ").some((id) => id && document.getElementById(id)?.textContent.trim())) : true, confirm: !!(wrap && wrap.querySelector("[class*=confirm]")), errorText: errs.map((e) => e.textContent.trim().slice(0, 40)).filter(Boolean).slice(0, 2) }; });
          const shown = await inspectView(page);
          const outcome = after.confirm ? "confirm" : after.errors ? "errors" : "other";
          submitted.push({ screen: screen.id, view: tab.key, lang, width, kind: m.kind, outcome, ...after, overflow: shown.overflow, clipped: shown.clipped, cut: shown.cut, low: shown.low });
          check(`${at}: after submitting from the keyboard the form shows its confirmation or its errors`, outcome !== "other", JSON.stringify(after));
          check(`${at}: after submitting, focus is inside the form and not on the pressed button's ghost`, after.inside && after.tag !== "BODY", JSON.stringify(after));
          if (outcome === "errors") check(`${at}: every field marked invalid points to a message with text`, after.invalid === 0 || after.described, JSON.stringify(after));
          check(`${at}: the ${outcome} state does not overflow, cut off or lose text contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ overflow: shown.overflow, clipped: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), low: shown.low.slice(0, 3) }));
          if (args.shots && !shot.has(screen.id + outcome + lang + width)) { shot.add(screen.id + outcome + lang + width); await page.screenshot({ path: `${args.shots}/${screen.id}-${outcome}-${lang}-${width}.png`, fullPage: false }); }
        }
        await open();
      }
    }
  }
  check("every request that was not a read was blocked, and the forms were only opened", posts.length === 0, posts.join());
  check("the planted-defect probe ran", probed);
  check("every opener found opened a form that took focus", notes.length === 0, notes.join(" | "));
  if (args.confirm) check("every opened form reached its confirmation", unreached.length === 0, notes2.join(" | "));
  if (args.outcome) check(`every confirmation was pressed and answered ${args.outcome} (${outcomes.length} of ${confirmed.length})`, outcomes.length > 0 && outcomes.length === confirmed.length && notes2.length === 0, notes2.slice(0, 3).join(" | "));
  check("no page error", errors.length === 0, errors.join(" | "));
  if (args.shots) await (await import("node:fs/promises")).writeFile(`${args.shots}/form-states.json`, JSON.stringify({ records, submitted, confirmed, unreached, notes2, outcomes }, null, 1));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) if (!args.quiet || r.startsWith("FAIL")) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  for (const n of notes2) console.log("UNREACHED " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
