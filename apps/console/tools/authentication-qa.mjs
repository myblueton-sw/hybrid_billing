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
const copy = {
  en: { title: "Impact if activated", everyone: "Everyone in this workspace is signed out at activation.", not: "Not computed. This is not zero.", members: "Members" },
  ko: { title: "활성화 시 영향", everyone: "활성화하면 이 작업 공간의 모든 사용자가 로그아웃됩니다.", not: "계산되지 않았습니다. 0이라는 뜻이 아닙니다.", members: "구성원" },
  ar: { title: "الأثر عند التفعيل", everyone: "يُسجَّل خروج الجميع في مساحة العمل هذه عند التفعيل.", not: "لم يُحسب. هذا لا يعني صفرًا.", members: "الأعضاء" },
};
const widths = [1440, 390];

async function open(lang, view = "policies") {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/authentication?lang=${lang}&view=${view}`);
}
async function settled() {
  await page.locator('.m02-live[data-live="policies"]').waitFor();
  await page.waitForFunction(() => document.querySelector('.m02-live[data-live="policies"]')?.getAttribute("aria-busy") !== "true");
  await page.locator(".m02-changes").waitFor();
  for (const group of ["open", "closed"]) await page.locator(`.m02-list-section[data-group="${group}"] h3[data-count]`).waitFor();
}
async function contained(name) {
  const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
  check(`${name} page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
}
const one = (url) => /\/api\/v1\/identity\/policy\/changes\/[^/]+$/.test(new URL(url.toString()).pathname);
async function staged(change) {
  await page.unroute(one);
  await page.route(one, async (route) => {
    const response = await route.fetch();
    if (response.status() !== 200) return route.fulfill({ response });
    const body = await response.json();
    if (body.change) body.change = change(body.change);
    await route.fulfill({ response, json: body });
  });
}
async function detail(id) {
  await page.locator(`.m02-open[data-change="${id}"]`).first().click();
  const body = page.locator(`.m02-detail-body[data-change="${id}"]`).first();
  await body.waitFor();
  return body;
}

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

  const served = { open: [], closed: [] };
  let capture = true;
  page.on("response", async (r) => {
    const u = new URL(r.url());
    if (!capture || !u.pathname.endsWith("/api/v1/identity/policy/changes") || r.status() !== 200) return;
    const body = await r.json().catch(() => null);
    if (body && Array.isArray(body.rows)) served[u.searchParams.get("state") === "closed" ? "closed" : "open"] = body.rows;
  });
  const listed = () => [...served.open, ...served.closed];
  const describe = () => `served changes ${listed().length}: ${listed().map((c) => `${c.number}:${c.state}`).join(" ")}`;
  await open("en"); await settled();
  notes.push("at start, " + describe());
  if (listed().length === 0) {
    await page.locator(".m02-propose").click();
    await page.locator("#m02-age").fill("600");
    await page.locator(".m02-form button[type=submit]").click();
    await page.locator(".m02-confirm #live-reason").fill("synthetic rendered check");
    await page.locator(".m02-confirm button[type=submit]").click();
    await page.locator(".m02-open[data-change]").first().waitFor();
    check("a trust change can be proposed from the screen", true);
    await open("en"); await settled();
    notes.push("after proposing, " + describe());
    check("the proposed change is read back as a draft", listed().length === 1 && listed()[0].state === "draft");
  }
  const changes = listed();
  capture = false;
  check("the read has at least one change to open", changes.length > 0);

  for (const lang of ["en", "ko", "ar"]) {
    for (const width of widths) {
      await page.setViewportSize({ width, height: 1000 });
      await open(lang); await settled();
      const root = page.locator('[data-security-screen="M02"]');
      check(`${lang} ${width} screen is live with its language and direction`, (await root.getAttribute("lang")) === lang && (await root.getAttribute("dir")) === (lang === "ar" ? "rtl" : "ltr"));
      const text = await page.locator(".m02-live").innerText();
      check(`${lang} ${width} no placeholder text`, !/undefined|NaN|\[object Object\]/.test(text), text.match(/.{0,40}(undefined|NaN|\[object Object\]).{0,40}/)?.[0]);
      check(`${lang} ${width} lists the served changes`, await page.locator(".m02-open[data-change]").count() >= Math.min(changes.length, 1));
      await contained(`${lang} ${width} list`);
      if (args.shots) await page.screenshot({ path: `${args.shots}/m02-${lang}-${width}.png`, fullPage: true });
    }
  }

  await page.setViewportSize({ width: 1440, height: 1000 });
  for (const view of ["providers", "test", "keys", "sync", "recovery"]) {
    await open("en", view);
    await page.locator(".live-users").waitFor();
    check(`${view} view renders`, (await page.locator(".live-users").innerText()).trim().length > 0);
    await contained(`en 1440 ${view}`);
  }

  const target = changes.find((c) => c.state !== "activation_unknown");
  if (target) {
    for (const lang of ["en", "ko", "ar"]) {
      for (const width of widths) {
        await page.setViewportSize({ width, height: 1000 });
        await staged((c) => (c.id === target.id ? { ...c, impact: null } : c));
        await open(lang); await settled();
        const body = await detail(target.id);
        const impact = body.locator(".m02-impact");
        const shown = (await impact.count()) > 0 ? await impact.first().innerText() : "";
        check(`${lang} ${width} impact that was not computed says so and is not zero`, shown.includes(copy[lang].title) && shown.includes(copy[lang].not) && shown.includes(copy[lang].everyone) && !shown.includes(copy[lang].members), shown);
        await contained(`${lang} ${width} detail without impact`);
        if (args.shots) await page.screenshot({ path: `${args.shots}/m02-not-computed-${lang}-${width}.png`, fullPage: true });

        await staged((c) => (c.id === target.id ? { ...c, impact: { members: 0, sessions: 5, affected_ids: [], weakens_mfa: false, signs_out_everyone: false } } : c));
        await open(lang); await settled();
        const zero = (await detail(target.id)).locator(".m02-impact");
        const zeroText = (await zero.count()) > 0 ? await zero.first().innerText() : "";
        const counted = zero.first().locator(`xpath=.//dt[normalize-space()="${copy[lang].members}"]/following-sibling::dd[1]`);
        const members = (await counted.count()) > 0 ? (await counted.first().innerText()).trim() : "absent";
        check(`${lang} ${width} computed impact of zero shows zero members and not the sentence`, members === "0" && !zeroText.includes(copy[lang].not) && !zeroText.includes(copy[lang].everyone), members + " | " + zeroText);
        await page.unroute(one);
      }
    }
    await page.setViewportSize({ width: 1440, height: 1000 });
    await open("en"); await settled();
    const real = (await detail(target.id)).locator(".m02-impact");
    notes.push(`unmodified detail of change ${target.number} (${target.state}): ${(await real.count()) > 0 ? (await real.first().innerText()).replace(/\s+/g, " ").slice(0, 200) : "no impact section"}`);
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
