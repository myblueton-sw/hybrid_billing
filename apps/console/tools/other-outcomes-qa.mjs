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

const label = (lang, en, ko, ar) => ({ en, ko, ar })[lang] ?? en;
const byText = (text) => page.locator("main button").filter({ hasText: new RegExp("^\\s*" + text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&") + "\\s*$") }).first();
const recipes = {
  "M02:policies": async (lang) => {
    await byText(label(lang, "Propose trust change", "신뢰 변경 제안", "اقتراح تغيير ثقة")).click();
    const form = page.locator("form:has(#m02-age)");
    await form.locator("input[type=radio]").nth(1).check();
    await page.locator("#m02-age").fill("10");
    return form.locator("button[type=submit]").first();
  },
  "M03:grants": async (lang) => {
    await byText(label(lang, "Request access", "권한 추가 요청", "طلب صلاحية")).click();
    await page.locator("#access-person").waitFor();
    await page.locator("#access-person").fill("admin2");
    await page.waitForTimeout(600);
    await page.locator("#access-panel button").filter({ hasText: "admin2@example.invalid" }).first().click();
    const cap = page.locator("#access-capability");
    await cap.selectOption({ index: 1 });
    await page.locator("#access-days").fill("7");
    await page.locator("#access-panel #live-reason").fill("Keyboard outcome check");
    return page.locator("#access-panel form button[type=submit]").last();
  },
  "O01:groups": async (lang) => {
    await page.locator("#o01-new-draft").click();
    await page.locator("#o01-group-new").check();
    await page.locator("#o01-draft-name").fill("QA outcome group");
    const panel = page.locator("#o01-draft-name").locator("xpath=ancestor::*[.//textarea[@id='live-reason']][1]");
    await page.locator("#o01-draft-q").focus();
    await page.keyboard.press("Enter");
    await page.waitForTimeout(300);
    await page.waitForFunction(() => { const name = document.getElementById("o01-draft-name"); const after = [...document.querySelectorAll("main button")].filter((b) => (name.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING) && b.getClientRects().length); const c = {}; after.forEach((b) => { c[b.textContent.trim()] = (c[b.textContent.trim()] ?? 0) + 1; }); return Object.values(c).some((n) => n > 1); }, null, { timeout: 8000 }).catch(async () => {
      const search = page.locator("#o01-draft-q").locator("xpath=following::button[1]");
      await search.focus(); await page.keyboard.press("Enter"); await page.waitForTimeout(1200);
    });
    const adds = await page.evaluate(() => {
      const name = document.getElementById("o01-draft-name");
      const after = [...document.querySelectorAll("main button")].filter((b) => name && (name.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING) && b.getClientRects().length && !b.disabled);
      const count = {};
      after.forEach((b) => { const t = b.textContent.trim(); count[t] = (count[t] ?? 0) + 1; });
      const text = Object.entries(count).sort((a, b) => b[1] - a[1])[0]?.[0] ?? "";
      const pick = after.filter((b) => b.textContent.trim() === text)[2] ?? after.find((b) => b.textContent.trim() === text);
      pick?.setAttribute("data-qa-add", "");
      return text + " " + JSON.stringify(count);
    });
    const add = page.locator("[data-qa-add]").first();
    await add.focus();
    await page.keyboard.press("Enter");
    await page.waitForTimeout(300);
    const added = await panel.evaluate((e) => e.innerText.match(/\n(\d+)[^\n]*1000/)?.[1] ?? "?");
    if (added === "0") throw new Error(`adding a member did not register (${adds})`);
    await page.locator("main textarea#live-reason").fill("Keyboard outcome check");
    return byText(label(lang, "Save draft", "초안 저장", "حفظ المسودة"));
  },
  "M04:accounts": async (lang) => {
    await byText(label(lang, "Create service account", "서비스 계정 만들기", "إنشاء حساب خدمة")).click();
    const form = page.locator("main form").filter({ has: page.locator("#live-reason") }).first();
    await form.locator("input").first().fill("qa-outcome");
    const selects = form.locator("select");
    const choose = async (value) => { const n = await selects.count(); for (let i = 0; i < n; i++) { const pick = await selects.nth(i).evaluate((e, v) => { const o = [...e.options].find((o) => o.value === v || o.textContent.trim() === v || o.value.endsWith(":" + v) || o.textContent.trim().startsWith(v + " ")); return o && (!e.value || e.value === o.value) ? o.value : null; }, value); if (pick) { await selects.nth(i).selectOption(pick); return; } } const seen = await selects.evaluateAll((es) => es.slice(2).map((e) => (e.value || "-") + ":" + [...e.options].map((o) => o.value + "=" + o.textContent.trim()).slice(0, 5).join(","))); throw new Error("no select offers " + value + " " + seen.join(" / ")); };
    const settle = async (value) => { for (let i = 0; ; i++) { try { await choose(value); return; } catch (e) { if (i >= 30) throw e; await page.waitForTimeout(250); } } };
    await settle(args.owner ?? "sample-admin");
    await form.locator("textarea:not(#live-reason)").first().fill("Keyboard outcome check");
    await settle("sample-le-seller");
    await settle("test-service-api");
    for (let i = 0; ; i++) { try { await choose("read"); break; } catch (e) { if (i >= 20) throw e; await page.waitForTimeout(250); } }
    await form.locator("button").filter({ hasText: label(lang, "Add scope", "범위 추가", "إضافة نطاق") }).first().click();
    await form.locator("#live-reason").fill("Keyboard outcome check");
    return form.locator("button[type=submit]").last();
  },
};

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
  const answered = [], dropped = [], held = [], busy = [], recovered = [];
  await page.route((url) => url.pathname.includes("/api/"), async (route) => {
    const method = route.request().method();
    if (method === "GET" || method === "HEAD") return route.continue();
    const path = new URL(route.request().url()).pathname;
    if (/sign-?out|logout|\/session$/.test(path) && method === "DELETE") { dropped.push(path); return route.abort(); }
    answered.push(method + " " + path);
    if (args.busy) await new Promise((done) => { held.push(done); setTimeout(done, 15000); });
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
      const keys = args.recipes ? (recipes[`${screen.id}:${tab.key}`] && args.recipes.split(",").includes(`${screen.id}:${tab.key}`) ? ["recipe"] : []) : await candidates();
      for (const key of keys) {
        const at = `${screen.id} ${tab.key} ${lang} ${width} [${key.split("|")[0].slice(0, 40)}]`;
        const begin = async () => {
        await open();
        await page.evaluate(() => document.querySelectorAll("main button").forEach((b) => b.setAttribute("data-qa-seen", "")));
        if (key === "recipe") {
          let final = null;
          try { final = await recipes[`${screen.id}:${tab.key}`](lang); } catch (e) { unreached.push(`${at}: recipe stopped: ${String(e.message).split("\n").slice(0, 3).join(" ").slice(0, 300)}`); if (args.shots) await page.screenshot({ path: `${args.shots}/unreached-${screen.id}-${lang}-${width}.png`, fullPage: true }); return false; }
          await page.evaluate(() => document.querySelectorAll("main button").forEach((b) => b.setAttribute("data-qa-seen", "")));
          await final.evaluate((b) => b.setAttribute("data-qa-next", ""));
        } else if (!(await pressKey(key))) { notes.push(`${at}: not found again`); return false; }
          return true;
        };
        if (!(await begin())) continue;
        const before = answered.length;
        let earlier = [];
        let rounds = 0;
        for (; rounds < 6; rounds++) {
          earlier = await page.evaluate(() => [...document.querySelectorAll("main [role=alert], main [role=status], main [aria-live]")].map((e) => e.textContent.trim().replace(/\s+/g, " ").slice(0, 90)));
          if (rounds === 0 && key !== "recipe" && (await page.locator("[data-qa-next]").count()) === 0) await pressKey(key);
          if (!(await press())) break;
          await page.waitForFunction(() => !document.querySelector("main [aria-busy=true]"), null, { timeout: 8000 }).catch(() => {});
          await page.waitForTimeout(200);
          if (answered.length > before) break;
          await fill();
          if (!(await nextButton())) break;
        }
        if (args.busy && answered.length > before) {
          await page.waitForTimeout(250);
          const during = await page.evaluate((earlier) => {
            const a = document.activeElement, r = a?.getBoundingClientRect();
            const live = [...document.querySelectorAll("main [role=status], main [role=alert], main [aria-live]")].map((e) => e.textContent.trim().replace(/\s+/g, " ").slice(0, 90)).filter((t) => t && !earlier.includes(t));
            const marked = !!document.querySelector("main [aria-busy=true]");
            const pressed = document.querySelector("[data-qa-pressed]:last-of-type");
            return { tag: a?.tagName ?? "", text: (a?.textContent ?? "").trim().slice(0, 40), visible: !!r && r.width > 0 && r.height > 0, marked, live: live.slice(0, 2), disabled: !!pressed?.disabled };
          }, earlier);
          const shown = await inspectView(page);
          busy.push({ at, ...during });
          check(`${at}: while the request is in flight focus is on a visible element, not the body`, during.tag !== "BODY" && during.tag !== "" && during.visible, JSON.stringify(during));
          check(`${at}: while the request is in flight the page says it is working (aria-busy or a new status)`, during.marked || during.live.length > 0, JSON.stringify(during));
          check(`${at}: the in-flight state does not overflow, cut off or lose text contrast`, shown.overflow <= 1 && !shown.clipped.length && !shown.cut.length && !shown.vcut.length && !shown.low.length, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), v: shown.vcut.slice(0, 2), low: shown.low.slice(0, 2) }));
          held.splice(0).forEach((done) => done());
          await page.waitForFunction(() => !document.querySelector("main [aria-busy=true]"), null, { timeout: 8000 }).catch(() => {});
          await page.waitForTimeout(300);
        }
        if (answered.length === before) { unreached.push(`${at}: no command after ${rounds} presses`); if (args.shots && key === "recipe") await page.screenshot({ path: `${args.shots}/unreached-${screen.id}-${lang}-${width}.png`, fullPage: true }); continue; }
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
        if (args.recovery && rounds >= 1) {
          for (let pick = 0; pick < 2; pick++) {
            if (!(await begin())) break;
            for (let i = 0; i < rounds; i++) {
              if (i === 0 && key !== "recipe" && (await page.locator("[data-qa-next]").count()) === 0) await pressKey(key);
              if (!(await press())) break;
              await page.waitForFunction(() => !document.querySelector("main [aria-busy=true]"), null, { timeout: 8000 }).catch(() => {});
              await page.waitForTimeout(200);
              await fill();
              if (!(await nextButton())) break;
            }
            const sib = await page.evaluate((pick) => {
              const final = document.querySelector("[data-qa-next]");
              if (!final) return null;
              let box = final.parentElement;
              while (box && box.querySelectorAll("button").length < 2 && box.tagName !== "MAIN") box = box.parentElement;
              const others = [...(box?.querySelectorAll("button") ?? [])].filter((b) => b !== final && !b.disabled && b.getClientRects().length && !b.closest("li, tr, [role=row], [role=listitem], .list-pager"));
              const b = others[pick];
              if (!b) return null;
              final.removeAttribute("data-qa-next");
              b.setAttribute("data-qa-sib", "");
              return { final: final.textContent.trim().slice(0, 40), text: b.textContent.trim().slice(0, 40) };
            }, pick);
            if (!sib) break;
            const sent = answered.length;
            const typed = await page.evaluate(() => [...document.querySelectorAll("main textarea, main input[type=text]")].filter((e) => e.value === "QA keyboard check reason" || e.value === "Keyboard outcome check").length);
            const filledBefore = await page.evaluate(() => [...document.querySelectorAll("main input:not([type=search]):not([type=checkbox]):not([type=radio]):not([type=hidden]), main textarea, dialog input:not([type=hidden]), dialog textarea")].filter((e) => e.value).map((e) => e.id || e.name || e.tagName));
            await page.locator("[data-qa-sib]").focus();
            await page.keyboard.press("Enter");
            await frames(); await page.waitForTimeout(300);
            const after = await page.evaluate(() => {
              const a = document.activeElement, r = a?.getBoundingClientRect();
              const kept = [...document.querySelectorAll("main textarea, main input[type=text]")].filter((e) => e.value === "QA keyboard check reason" || e.value === "Keyboard outcome check").length;
              const filled = [...document.querySelectorAll("main input:not([type=search]):not([type=checkbox]):not([type=radio]):not([type=hidden]), main textarea")].filter((e) => e.value && e.getClientRects().length).map((e) => e.id || e.name || e.tagName);
              return { tag: a?.tagName ?? "", text: (a?.textContent ?? "").trim().replace(/\s+/g, " ").slice(0, 40), visible: !!r && r.width > 0 && r.height > 0, inside: !!a?.closest?.("main, dialog"), kept, filled };
            });
            const shown = await inspectView(page);
            const where = `${at} [${sib.text} beside ${sib.final}]`;
            recovered.push({ at: where, typed, filledBefore, ...after });
            check(`${where}: pressing it sends no command`, answered.length === sent, String(answered.length - sent));
            check(`${where}: focus lands on a visible element in the page, not the body`, after.tag !== "BODY" && after.tag !== "" && after.visible && after.inside, JSON.stringify(after));
            check(`${where}: the page does not overflow, cut off or lose text contrast`, shown.overflow <= 1 && !shown.clipped.length && !shown.cut.length && !shown.vcut.length && !shown.low.length, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 2), v: shown.vcut.slice(0, 2), low: shown.low.slice(0, 2) }));
          }
        }
      }
    }
  }
  check(`commands were reached and answered (${outcomes.length})`, outcomes.length > 0);
  if (args.recipes) check(`every recipe reached its command (${outcomes.length} of ${args.recipes.split(",").length * combos.length})`, unreached.length === 0 && outcomes.length === args.recipes.split(",").length * combos.length, unreached.join(" | "));
  check("no sign-out request was made", dropped.length === 0, dropped.join());
  check("no page error", errors.length === 0, errors.join(" | "));
  if (args.shots) await writeFile(`${args.shots}/other-outcomes.json`, JSON.stringify({ outcomes, unreached, notes, busy, recovered }, null, 1));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) if (!args.quiet || r.startsWith("FAIL")) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  for (const n of unreached) console.log("UNREACHED " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
