import { inspectView } from "./view-measure.mjs";
import { createRequire } from "node:module";
import { readFile, mkdir, writeFile } from "node:fs/promises";
import { randomBytes } from "node:crypto";
const args = Object.fromEntries(process.argv.slice(2).flatMap((v, i, all) => (v.startsWith("--") ? [[v.slice(2), all[i + 1]]] : [])));
const { chromium } = createRequire(import.meta.url)(args["playwright-module"] || "playwright");
const base = (args.url || "http://127.0.0.1:4173").replace(/\/$/, "");
const browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
const context = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
const page = await context.newPage();
const results = [], errors = [], notes = [], outcomes = [], unreached = [];
let navigation = 0;
const check = (name, ok, detail = "") => { results.push(`${ok ? "PASS" : "FAIL"} ${name}${!ok && detail ? " " + String(detail).replace(/\s+/g, " ").slice(0, 400) : ""}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const wanted = (args.screens ?? "M01,M02,M03,M04,M05,O01,O02,O03,P01").split(",");
const screens = [];
for (const m of ["identity", "organizations", "administration"]) screens.push(...JSON.parse(await readFile(new URL(`../src/modules/${m}/screens.json`, import.meta.url), "utf8")).filter((s) => wanted.includes(s.id)));
const combos = (args.combos ?? "en:1440").split(",").map((c) => c.split(":"));
const limit = Number(args.limit ?? 30);
const frames = () => page.evaluate(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(done, 80)))));
const answer = args.answer ?? args.outcome ?? "refused";
const reply = { refused: [422, "prerequisite_failed", "correct_request"], conflict: [409, "revision_conflict", "reload"], unavailable: [503, "dependency_unavailable", "retry"] };
const pending = ["may have been applied", "ربما طُبّق"];
const skip = /sign out|log out|تسجيل الخروج|로그아웃/i;

const candidates = () => page.evaluate(({ limit, skip }) => {
  const re = new RegExp(skip, "i");
  const out = [], seen = new Set();
  for (const b of document.querySelectorAll("main button")) {
    if (b.disabled || b.getAttribute("aria-disabled") === "true" || !b.getClientRects().length) continue;
    if (b.closest(".list-pager, nav, [role=tablist], .applied-filters, .list-filters, [data-qa-plant]")) continue;
    const text = (b.textContent ?? "").trim().replace(/\s+/g, " ");
    if (!text || re.test(text)) continue;
    const key = text.slice(0, 60) + "|" + String(b.className).split(" ")[0];
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(key);
    if (out.length >= limit) break;
  }
  return out;
}, { limit, skip: skip.source });

const pressKey = (key) => page.evaluate((key) => {
  for (const b of document.querySelectorAll("main button")) {
    if (b.disabled || !b.getClientRects().length) continue;
    const text = (b.textContent ?? "").trim().replace(/\s+/g, " ");
    if (text.slice(0, 60) + "|" + String(b.className).split(" ")[0] === key) { b.setAttribute("data-qa-next", ""); return true; }
  }
  return false;
}, key);

const fill = () => page.evaluate(() => {
  let n = 0;
  const set = (el, v) => { const proto = el.tagName === "TEXTAREA" ? HTMLTextAreaElement.prototype : el.tagName === "SELECT" ? HTMLSelectElement.prototype : HTMLInputElement.prototype; Object.getOwnPropertyDescriptor(proto, "value").set.call(el, v); el.dispatchEvent(new Event("input", { bubbles: true })); el.dispatchEvent(new Event("change", { bubbles: true })); n++; };
  for (const el of document.querySelectorAll("main input, main textarea, main select")) {
    if (el.disabled || el.readOnly || !el.getClientRects().length || el.hasAttribute("data-qa-filled") || el.closest(".list-filters, .list-pager")) continue;
    el.setAttribute("data-qa-filled", "");
    if (el.tagName === "SELECT") { if (!el.value && el.options.length > 1) set(el, el.options[1].value); continue; }
    const type = el.type;
    if (type === "radio") { const group = el.name ? [...document.querySelectorAll(`main input[type=radio][name="${CSS.escape(el.name)}"]`)] : [el]; if (!group.some((g) => g.checked)) { group.forEach((g) => g.setAttribute("data-qa-filled", "")); el.click(); n++; } continue; }
    if (type === "checkbox") { if (el.closest("form, dialog, [class*=form], [role=group]") && !el.closest("table, [role=table], ul, ol") && !el.checked) { el.click(); n++; } continue; }
    if (["search", "hidden", "file"].includes(type) || el.value) continue;
    set(el, type === "number" ? "1" : type === "date" ? "2027-01-01" : type === "email" ? "qa@example.invalid" : type === "password" ? "qa-check-password-1" : "QA keyboard check reason");
  }
  return n;
});

const nextButton = () => page.evaluate((skip) => {
  const re = new RegExp(skip, "i");
  const fresh = [...document.querySelectorAll("main button")].filter((b) => !b.disabled && b.getClientRects().length && !b.hasAttribute("data-qa-seen") && !re.test(b.textContent ?? ""));
  const quiet = /cancel|back|close|reload|search|check again|confirm screen review|collapse|expand|go to|취소|닫기|إلغاء|رجوع|إغلاق|إعادة التحميل|بحث/i;
  const pick = fresh.find((b) => b.type === "submit") ?? fresh.find((b) => /primary/.test(String(b.className))) ?? [...document.querySelectorAll("main form button[type=submit]")].find((b) => !b.disabled && b.getClientRects().length && !b.hasAttribute("data-qa-pressed")) ?? fresh.find((b) => !quiet.test(b.textContent ?? "") && !b.closest(".list-pager, nav, [role=tablist], .list-filters"));
  document.querySelectorAll("main button").forEach((b) => b.setAttribute("data-qa-seen", ""));
  if (!pick) return false;
  pick.setAttribute("data-qa-next", "");
  return true;
}, skip.source);

const press = async () => {
  const b = page.locator("[data-qa-next]").first();
  if ((await b.count()) === 0) return false;
  await page.evaluate(() => { const b = document.querySelector("[data-qa-next]"); b.removeAttribute("data-qa-next"); b.setAttribute("data-qa-pressed", ""); b.setAttribute("data-qa-now", ""); });
  const now = page.locator("[data-qa-now]");
  await now.focus();
  await page.keyboard.press("Enter");
  await page.evaluate(() => document.querySelector("[data-qa-now]")?.removeAttribute("data-qa-now"));
  await frames();
  return true;
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
  const answered = [], dropped = [];
  await page.route((url) => url.pathname.includes("/api/"), (route) => {
    const method = route.request().method();
    if (method === "GET" || method === "HEAD") return route.continue();
    const path = new URL(route.request().url()).pathname;
    if (/sign-?out|logout|\/session$/.test(path) && method === "DELETE") { dropped.push(path); return route.abort(); }
    answered.push(method + " " + path);
    if (answer === "unknown") return route.abort("timedout");
    const [status, code, next] = reply[answer];
    return route.fulfill({ status, json: { code, retry: status === 503, next_action: next, request_id: "qa-outcome" } });
  });
  if (args.shots) await mkdir(args.shots, { recursive: true });

  for (const [lang, w] of combos) {
    const width = Number(w);
    await page.setViewportSize({ width, height: 900 });
    for (const screen of screens) for (const tab of screen.tabs) {
      const open = async () => {
        await page.goto("about:blank");
        await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#${screen.path}?lang=${lang}&view=${tab.key}`);
        await page.locator("main").waitFor();
        await page.waitForLoadState("networkidle");
        await page.waitForFunction(() => !document.querySelector("main [aria-busy=true]"), null, { timeout: 8000 }).catch(() => {});
        return page.locator(".service-session").count();
      };
      if (!(await open())) { notes.push(`${screen.id} ${tab.key}: signed out`); continue; }
      const keys = await candidates();
      for (const key of keys) {
        const at = `${screen.id} ${tab.key} ${lang} ${width} [${key.split("|")[0].slice(0, 40)}]`;
        await open();
        await page.evaluate(() => document.querySelectorAll("main button").forEach((b) => b.setAttribute("data-qa-seen", "")));
        if (!(await pressKey(key))) { notes.push(`${at}: not found again`); continue; }
        const before = answered.length;
        let earlier = [];
        let rounds = 0;
        for (; rounds < 6; rounds++) {
          earlier = await page.evaluate(() => [...document.querySelectorAll("main [role=alert], main [role=status], main [aria-live]")].map((e) => e.textContent.trim().replace(/\s+/g, " ").slice(0, 90)));
          if (rounds === 0 && (await page.locator("[data-qa-next]").count()) === 0) await pressKey(key);
          if (!(await press())) break;
          await page.waitForFunction(() => !document.querySelector("main [aria-busy=true]"), null, { timeout: 8000 }).catch(() => {});
          await page.waitForTimeout(200);
          if (answered.length > before) break;
          await fill();
          if (!(await nextButton())) break;
        }
        if (answered.length === before) { unreached.push(`${at}: no command after ${rounds} presses`); continue; }
        await frames();
        const state = await page.evaluate((earlier) => {
          const a = document.activeElement, r = a?.getBoundingClientRect();
          const own = a?.closest?.("[role=alert], [role=status], [aria-live]");
          const fresh = (e) => e.textContent.trim() && !earlier.includes(e.textContent.trim().replace(/\s+/g, " ").slice(0, 90));
          let near = [], depth = 0;
          if (!own) for (let n = a?.parentElement; n && n.tagName !== "MAIN"; n = n.parentElement, depth++) { near = [...n.querySelectorAll("[role=alert], [role=status], [aria-live]")].filter(fresh); if (near.length) break; }
          const region = own ?? near.find((e) => e.getAttribute("role") === "alert") ?? near[0] ?? null;
          return { depth, how: own ? "focused" : region ? "near" : "", titled: !!a && a.getAttribute("tabindex") === "-1", tag: a?.tagName ?? "", cls: String(a?.className ?? "").slice(0, 40), text: (a?.textContent ?? "").trim().replace(/\s+/g, " ").slice(0, 60), visible: !!r && r.width > 0 && r.height > 0, inMain: !!a?.closest?.("main"), role: region?.getAttribute("role") ?? region?.getAttribute("aria-live") ?? "", said: region ? region.textContent.trim().replace(/\s+/g, " ").slice(0, 90) : "" };
        }, earlier);
        const found = await inspectView(page);
        const sent = [...new Set(answered.slice(before))];
        outcomes.push({ screen: screen.id, view: tab.key, lang, width, key, sent, ...state, overflow: found.overflow, clipped: found.clipped, cut: found.cut, vcut: found.vcut, low: found.low });
        check(`${at}: only the final press sent requests (${sent.join(", ")})`, sent.length >= 1 && sent.length <= 3, sent.join(", "));
        check(`${at}: after the ${answer} answer focus is on a visible element in the page, not the body`, state.tag !== "BODY" && state.tag !== "" && state.visible && state.inMain, JSON.stringify(state));
        check(`${at}: the ${answer} answer is stated in a new live region or alert that holds focus, or sits with the focused title`, !!state.said && !earlier.includes(state.said) && (state.how === "focused" || (state.how === "near" && state.titled)), JSON.stringify({ said: state.said, role: state.role, how: state.how, titled: state.titled, focus: state.cls + " " + state.text }));
        if (answer === "refused" || answer === "conflict") check(`${at}: the ${answer} answer does not say the request may have been applied`, !pending.some((p) => state.said.includes(p)), state.said);
        check(`${at}: the ${answer} state does not overflow, cut off or lose text contrast`, found.overflow <= 1 && !found.clipped.length && !found.cut.length && !found.vcut.length && !found.low.length, JSON.stringify({ o: found.overflow, c: found.clipped.slice(0, 2), cut: found.cut.slice(0, 2), v: found.vcut.slice(0, 2), low: found.low.slice(0, 2) }));
        if (args.shots && (state.tag === "BODY" || !state.said)) await page.screenshot({ path: `${args.shots}/${screen.id}-${tab.key}-${lang}-${width}-${outcomes.length}.png` });
      }
    }
  }
  check(`commands were reached and answered (${outcomes.length})`, outcomes.length > 0);
  check("no sign-out request was made", dropped.length === 0, dropped.join());
  check("no page error", errors.length === 0, errors.join(" | "));
  if (args.shots) await writeFile(`${args.shots}/other-outcomes.json`, JSON.stringify({ outcomes, unreached, notes }, null, 1));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) if (!args.quiet || r.startsWith("FAIL")) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  for (const n of unreached) console.log("UNREACHED " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
