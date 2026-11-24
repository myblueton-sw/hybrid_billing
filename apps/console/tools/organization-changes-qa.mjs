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
const read = "**/api/v1/organizations/plans*";
const copy = {
  en: { sealed: "Plan proposed.", rejected: "Plan rejected.", discarded: "Plan discarded.", activated: (n) => `Activated. Relationship revision ${n} is now active.`, any: /Plan proposed|Plan rejected|Plan discarded|Activated\./ },
  ko: { sealed: "계획을 제안했습니다.", rejected: "계획이 거부되었습니다.", discarded: "계획을 폐기했습니다.", activated: (n) => `활성화했습니다. 관계 리비전 ${n}이(가) 활성 상태입니다.`, any: /계획을 제안했습니다|계획이 거부되었습니다|계획을 폐기했습니다|활성화했습니다/ },
  ar: { sealed: "اقتُرحت الخطة.", rejected: "رُفضت الخطة.", discarded: "تم تجاهل الخطة.", activated: (n) => `تم التفعيل. مراجعة العلاقات ${n} نشطة الآن.`, any: /اقتُرحت الخطة|رُفضت الخطة|تم تجاهل الخطة|تم التفعيل/ },
};
const views = ["change", "impact", "approval", "result"];
const widths = [1440, 390];
const root = () => page.locator('[data-screen-id="O02"]');

async function open(lang, view, extra = "") {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/organization-changes?lang=${lang}&view=${view}${extra}`);
  await root().waitFor();
  await page.locator(".o02-live").waitFor();
  await page.locator("#o02-announce").waitFor({ state: "attached" });
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => !document.querySelector('[data-screen-id="O02"] [aria-busy="true"]'));
}
async function contained(name) {
  const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
  check(`${name} page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
}
const announced = async () => ((await page.locator("#o02-announce").count()) === 1 && (await root().locator(".live-error").count()) === 0 ? (await page.locator("#o02-announce").textContent()) ?? "" : "the plans were not loaded: Plan proposed. Plan rejected. Plan discarded. Activated. 계획을 제안했습니다 활성화했습니다 اقتُرحت الخطة تم التفعيل");
const doneLeft = () => page.evaluate(() => /[?&]done=/.test(location.hash));

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
  page.on("response", async (r) => { if (capture && r.url().split("?")[0].endsWith("/api/v1/organizations/plans") && r.status() === 200) served = await r.json().catch(() => served); });

  for (const lang of ["en", "ko", "ar"]) {
    for (const width of widths) {
      await page.setViewportSize({ width, height: 1000 });
      for (const view of views) {
        await open(lang, view);
        check(`${lang} ${width} ${view} is live with its language and direction`, (await root().getAttribute("data-source-state")) === "live" && (await root().getAttribute("lang")) === lang && (await root().getAttribute("dir")) === (lang === "ar" ? "rtl" : "ltr"));
        const text = await root().innerText();
        check(`${lang} ${width} ${view} no placeholder text and no load failure`, !/undefined|NaN|\[object Object\]/.test(text) && (await root().locator(".live-error").count()) === 0, text.match(/.{0,40}(undefined|NaN|\[object Object\]).{0,40}/)?.[0]);
        await contained(`${lang} ${width} ${view}`);
        if (args.shots) await page.screenshot({ path: `${args.shots}/o02-${view}-${lang}-${width}.png`, fullPage: true });
      }
    }
  }
  capture = false;
  const plans = Array.isArray(served?.plans) ? served.plans : [];
  notes.push(`served plans ${plans.length}: ${plans.map((p) => `${p.id}:${p.state}:${p.activated_revision ?? ""}`).join(" ").slice(0, 300)}`);

  await page.setViewportSize({ width: 1440, height: 1000 });
  for (const lang of ["en", "ko", "ar"]) {
    for (const done of ["sealed", "rejected", "discarded", "activated:99"]) {
      await open(lang, "impact", `&plan=no-such-plan&done=${done}`);
      const said = await announced();
      check(`${lang} arrival ${done} for a plan that is not loaded announces nothing`, !copy[lang].any.test(said) && !(await doneLeft()), said);
    }
  }
  const synthetic = { id: "qa-plan-1", state: "sealed", created: "2026-11-24T00:00:00Z", proposer: "sample-admin-2", proposer_login: "admin2@example.invalid", reason: "synthetic rendered check", change: { kind: "hierarchy", member: "sample-acct-01", action: "move", target: "", from: "2026-12-01" }, member_name: "Account 01 (sample)", base: 1, kinds: [], violations: [], members: [], impact: {} };
  const plan = plans[0] ?? synthetic;
  notes.push(plans[0] ? `arrival checks use the served plan ${plan.id}` : "the sample has no plan: the arrival checks use a plan added to the read's answer in the browser");
  {
    const stage = (change) => page.route(read, async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      const list = Array.isArray(body.plans) ? body.plans : [];
      body.plans = list.some((p) => p.id === plan.id) ? list.map((p) => (p.id === plan.id ? change(p) : p)) : [...list, change(plan)];
      await route.fulfill({ response, json: body });
    });
    for (const lang of ["en", "ko", "ar"]) {
      await stage((p) => ({ ...p, state: "activated", activated_revision: 7 }));
      await open(lang, "result", `&plan=${encodeURIComponent(plan.id)}&done=activated:99`);
      let said = await announced();
      check(`${lang} an activation announces the plan's own revision, not the one in the address`, said.includes(copy[lang].activated("7")) && !said.includes("99"), said);
      await open(lang, "result", `&plan=${encodeURIComponent(plan.id)}&done=sealed`);
      said = await announced();
      check(`${lang} an arrival that does not match the plan's state announces nothing`, !copy[lang].any.test(said), said);
      await page.unroute(read);
      await stage((p) => ({ ...p, state: "sealed" }));
      await open(lang, "approval", `&plan=${encodeURIComponent(plan.id)}&done=sealed`);
      said = await announced();
      check(`${lang} a matching arrival is announced`, said.includes(copy[lang].sealed) && !(await doneLeft()), said);
      if (args.shots && lang === "en") await page.screenshot({ path: `${args.shots}/o02-arrived-sealed-en.png`, fullPage: true });
      await page.unroute(read);
    }
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
