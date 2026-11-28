import { createRequire } from "node:module";
import { readFile } from "node:fs/promises";
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
const frames = () => page.evaluate(() => new Promise((done) => requestAnimationFrame(() => requestAnimationFrame(() => setTimeout(done, 80)))));
const open = async (hash, lang, width) => {
  await page.setViewportSize({ width, height: 1000 });
  await page.goto("about:blank");
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#${hash}${hash.includes("?") ? "&" : "?"}lang=${lang}`);
  await page.locator("main").waitFor();
  await page.waitForLoadState("networkidle");
};
const inDialog = () => page.evaluate(() => { const a = document.activeElement; return !!a && [...document.querySelectorAll("dialog[open]")].some((d) => d.contains(a)); });
const contained = () => page.evaluate(() => { const a = document.activeElement; return !a || a === document.body || [...document.querySelectorAll("dialog[open]")].some((d) => d.contains(a)); });

async function dialogCase(name, prepare, opener) {
  const trigger = await prepare();
  if (!trigger) { notes.push(`${name}: opener not available, not measured`); return; }
  await trigger.focus();
  const marker = await trigger.evaluate((e) => { e.setAttribute("data-qa-opener", ""); return true; });
  await page.keyboard.press("Enter");
  await page.locator("dialog[open]").first().waitFor({ timeout: 5000 }).catch(() => {});
  await frames();
  const opened = (await page.locator("dialog[open]").count()) > 0;
  if (!opened) { notes.push(`${name}: the control opens no modal dialog (an inline step), not measured here`); return; }
  if (args.plant) await page.evaluate(() => { const d = [...document.querySelectorAll("dialog[open]")].pop(); d.close(); d.show(); d.querySelector("button, a[href], input, select, textarea")?.focus(); });
  check(`${name}: the dialog opens from the keyboard and holds focus`, opened && (await inDialog()), await page.evaluate(() => document.activeElement?.outerHTML.slice(0, 120)));
  if (!opened) return;
  const stops = await page.locator("dialog[open]").last().evaluate((d) => d.querySelectorAll("button, a[href], input, select, textarea, summary, [tabindex]:not([tabindex='-1'])").length);
  let inside = true;
  for (let i = 0; i < stops + 2; i++) { await page.keyboard.press("Tab"); if (!(await contained())) { inside = false; break; } }
  check(`${name}: Tab never reaches page content outside the dialog (${stops + 2} presses; leaving to the browser's own controls is allowed)`, inside, await page.evaluate(() => document.activeElement?.tagName + " " + (document.activeElement?.textContent ?? "").trim().slice(0, 40)));
  inside = true;
  for (let i = 0; i < stops + 2; i++) { await page.keyboard.press("Shift+Tab"); if (!(await contained())) { inside = false; break; } }
  check(`${name}: Shift+Tab never reaches page content outside the dialog`, inside, await page.evaluate(() => document.activeElement?.tagName + " " + (document.activeElement?.textContent ?? "").trim().slice(0, 40)));
  if (args.orphan) await page.evaluate(() => { const o = document.querySelector("[data-qa-opener]"); if (o) { o.style.display = "none"; o.removeAttribute("data-qa-opener"); o.setAttribute("data-qa-gone", ""); } });
  await page.keyboard.press("Escape");
  await frames();
  const closed = (await page.locator("dialog[open]").count()) === 0;
  check(`${name}: Escape closes it`, closed);
  if (args.orphan) await page.waitForTimeout(400);
  const back = await page.evaluate(() => { const a = document.activeElement; const r = a?.getBoundingClientRect(); return { opener: !!a?.hasAttribute?.("data-qa-opener"), tag: a?.tagName, text: (a?.textContent ?? "").trim().slice(0, 40), visible: !!r && r.width > 0 }; });
  if (args.orphan) check(`${name}: with its opener gone, focus lands on a visible element in the page, not the body`, closed && back.tag !== "BODY" && back.tag !== undefined && back.visible, JSON.stringify(back));
  else check(`${name}: focus returns to the control that opened it`, closed && back.opener && back.visible, JSON.stringify(back));
  void marker; void opener;
}

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
  await page.route((url) => url.pathname.includes("/api/"), (route) => (route.request().method() === "GET" ? route.continue() : route.abort()));
  for (const [lang, width] of [["en", 1440], ["ar", 1440], ["en", 390]]) {
    const at = `${lang} ${width}`;
    await dialogCase(`${at} screen review (M03)`, async () => {
      await open("/access?view=roles", lang, width);
      const b = page.locator("button", { hasText: lang === "ar" ? "تأكيد مراجعة الشاشة" : "Confirm screen review" }).first();
      return (await b.count()) && (await b.isEnabled()) ? b : null;
    });
    await dialogCase(`${at} end other sessions (M05)`, async () => {
      await open("/my-security?view=sessions", lang, width);
      const b = page.locator(".mysec-end-others").first();
      await b.waitFor({ timeout: 8000 }).catch(() => {});
      return (await b.count()) && (await b.isEnabled()) ? b : null;
    });
    await dialogCase(`${at} bulk preparation (M01)`, async () => {
      await open("/users?view=users", lang, width);
      const box = page.locator("label.m01-hit input[type=checkbox]").first();
      await box.waitFor({ timeout: 8000 }).catch(() => {});
      if (!(await box.count())) return null;
      await box.click({ force: true });
      const b = page.locator(".m01-selbar button:not([disabled])").first();
      return (await b.count()) ? b : null;
    });
    await dialogCase(`${at} background jobs (shell)`, async () => {
      await open("/users?view=users", lang, width);
      const b = page.getByRole("button", { name: lang === "ar" ? "المهام الخلفية" : "Background jobs", exact: true });
      if (!(await b.count())) return null;
      if (!(await b.isVisible())) await page.locator("details:has(button)").filter({ has: b }).locator("summary").first().click().catch(() => {});
      return (await b.isVisible()) ? b : null;
    });
  }
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
