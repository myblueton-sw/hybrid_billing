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
const check = (name, ok, detail = "") => { results.push(`${ok ? "PASS" : "FAIL"} ${name}${!ok && detail ? " " + String(detail).replace(/\s+/g, " ").slice(0, 300) : ""}`); if (!ok) process.exitCode = 1; };
page.on("pageerror", (e) => errors.push(e.message));
const password = randomBytes(20).toString("base64url");
const isUsers = (url) => new URL(url).pathname.endsWith("/api/v1/identity/users");
const copy = {
  en: { role: "Unknown role", state: "Unknown state", roleWhy: "The service returned a role this screen does not recognise", stateWhy: "The service reported a state this screen does not recognise" },
  ko: { role: "알 수 없는 역할", state: "알 수 없는 상태", roleWhy: "서비스가 이 화면에서 인식하지 못하는 역할을 반환했습니다", stateWhy: "서비스가 이 화면에서 인식하지 못하는 상태를 반환했습니다" },
  ar: { role: "دور غير معروف", state: "حالة غير معروفة", roleWhy: "أعادت الخدمة دورًا لا تتعرّف عليه هذه الشاشة", stateWhy: "أعادت الخدمة حالة لا تتعرّف عليها هذه الشاشة" },
};
const views = ["users", "invitations", "membership", "revocation"];
const widths = [1440, 390];
const root = () => page.locator('[data-security-screen="M01"]');

async function open(lang, view) {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/users?lang=${lang}&view=${view}`);
  await root().waitFor();
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => { const r = document.querySelector('[data-security-screen="M01"]'); return !!r && !r.querySelector('[aria-busy="true"]') && r.innerText.trim().length > 10; });
}
async function contained(name) {
  const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
  check(`${name} page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
}
const row = (id) => page.locator(`.live-open[data-user-id="${id}"]`).first();
const rowText = (id) => row(id).locator("xpath=ancestor::*[@role='row' or self::tr or self::li][1]").innerText();

try {
  if (args.shots) await mkdir(args.shots, { recursive: true });
  const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
  const activation = new URL(lines.find((l) => l.startsWith("admin@example.invalid ")).split(" ")[1]);
  await page.goto(`${base}/activate?lang=en${activation.hash}`);
  await page.locator("#link-new").fill(password); await page.locator("#link-confirm").fill(password);
  await page.locator("form button[type=submit]").click(); await page.locator("a.service-button-link").waitFor();
  await page.goto(`${base}/?service=api&lang=en`);
  await page.locator("#signin-workspace").fill("sample-workspace"); await page.locator("#signin-email").fill("admin@example.invalid");
  await page.locator("#signin-password").fill(password); await page.locator("form button[type=submit]").click();
  await page.locator(".service-session").waitFor();

  let served = null, capture = true;
  page.on("response", async (r) => { if (capture && isUsers(r.url()) && r.status() === 200) { const body = await r.json().catch(() => null); if (body && Array.isArray(body.users) && (!served || body.users.length > served.users.length)) served = body; } });

  for (const lang of ["en", "ko", "ar"]) {
    for (const width of widths) {
      await page.setViewportSize({ width, height: 1000 });
      for (const view of views) {
        await open(lang, view);
        check(`${lang} ${width} ${view} is live with its language and direction`, (await root().getAttribute("data-security-phase")) === "live" && (await root().getAttribute("lang")) === lang && (await root().getAttribute("dir")) === (lang === "ar" ? "rtl" : "ltr"));
        const text = await root().innerText();
        check(`${lang} ${width} ${view} no placeholder text and no load failure`, !/undefined|NaN|\[object Object\]/.test(text) && (await root().locator(".live-error").count()) === 0, text.match(/.{0,40}(undefined|NaN|\[object Object\]).{0,40}/)?.[0]);
        await contained(`${lang} ${width} ${view}`);
        if (args.shots) await page.screenshot({ path: `${args.shots}/m01-${view}-${lang}-${width}.png`, fullPage: true });
      }
    }
  }
  capture = false;
  const members = (served?.users ?? []).filter((u) => u.membership && u.user_id !== served?.you && u.login !== "admin@example.invalid");
  notes.push(`served users ${(served?.users ?? []).length}; with a membership and not the signed-in person ${members.length}: ${members.map((u) => `${u.user_id}:${u.membership.role}:${u.membership.state}`).join(" ").slice(0, 300)}`);
  check("the sample has two other people with a membership", members.length >= 2);
  if (members.length >= 2) {
    const [a, b] = members;
    await page.route((url) => isUsers(url.toString()), async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      if (Array.isArray(body.users)) body.users = body.users.map((u) => (u.user_id === a.user_id ? { ...u, membership: { ...u.membership, role: "auditor" } } : u.user_id === b.user_id ? { ...u, membership: { ...u.membership, state: "frozen" } } : u));
      await route.fulfill({ response, json: body });
    });
    for (const lang of ["en", "ko", "ar"]) {
      for (const width of widths) {
        await page.setViewportSize({ width, height: 1000 });
        for (const view of ["users", "membership"]) {
          await open(lang, view);
          await row(a.user_id).waitFor();
          check(`${lang} ${width} ${view} lists an unrecognised role as unknown`, (await rowText(a.user_id)).includes(copy[lang].role), await rowText(a.user_id));
          check(`${lang} ${width} ${view} lists an unrecognised membership state as unknown`, (await rowText(b.user_id)).includes(copy[lang].state), await rowText(b.user_id));
          await contained(`${lang} ${width} ${view} with unknown values`);
          if (args.shots) await page.screenshot({ path: `${args.shots}/m01-unknown-${view}-${lang}-${width}.png`, fullPage: true });
        }
        await open(lang, "users");
        await row(a.user_id).click();
        let body = page.locator(".live-detail-body"); await body.waitFor();
        let text = await body.innerText();
        check(`${lang} ${width} detail of an unknown role says why no membership action is offered`, text.includes(copy[lang].roleWhy), text);
        if (lang === "en" && width === 1440) { notes.push(`actions offered for the unknown role: ${(await body.locator("[data-action]").evaluateAll((els) => els.map((e) => e.getAttribute("data-action")))).join(",") || "none"}`); if (args.shots) await page.screenshot({ path: `${args.shots}/m01-unknown-role-detail-en-1440.png`, fullPage: true }); }
        await open(lang, "users");
        await row(b.user_id).click();
        body = page.locator(".live-detail-body"); await body.waitFor();
        text = await body.innerText();
        check(`${lang} ${width} detail of an unknown membership state says why no action is offered`, text.includes(copy[lang].stateWhy), text);
        if (lang === "en" && width === 1440) notes.push(`actions offered for the unknown membership state: ${(await body.locator("[data-action]").evaluateAll((els) => els.map((e) => e.getAttribute("data-action")))).join(",") || "none"}`);
      }
    }
    await page.unrouteAll({ behavior: "wait" });
  }
  check("no unexpected JavaScript errors", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("runner completed without unexpected exception", false, String(e));
  if (args.shots) await page.screenshot({ path: `${args.shots}/runner-failure.png`, fullPage: true }).catch(() => {});
} finally {
  results.forEach((r) => console.log(r));
  notes.forEach((n) => console.log("NOTE " + n));
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
  await browser.close();
}
