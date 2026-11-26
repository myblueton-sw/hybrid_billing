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
  await page.locator(view === "revocation" ? "#m01-jobs-title" : view === "invitations" ? '[data-security-screen="M01"] .live-list, [data-security-screen="M01"] table' : ".live-open[data-user-id]").first().waitFor();
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => !document.querySelector('[data-security-screen="M01"] [aria-busy="true"]'));
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
        const offered = async () => (await page.locator(".live-detail-body [data-action]").evaluateAll((els) => els.map((e) => e.getAttribute("data-action")))).join(",");
        check(`${lang} ${width} an unknown role is offered the two account actions and no membership or sign-in link action`, (await offered()) === "suspend_user,depart_user", await offered());
        if (lang === "en" && width === 1440 && args.shots) await page.screenshot({ path: `${args.shots}/m01-unknown-role-detail-en-1440.png`, fullPage: true });
        await open(lang, "users");
        await row(b.user_id).click();
        body = page.locator(".live-detail-body"); await body.waitFor();
        text = await body.innerText();
        check(`${lang} ${width} detail of an unknown membership state says why no action is offered`, text.includes(copy[lang].stateWhy), text);
        check(`${lang} ${width} an unknown membership state is offered the two account actions and the sign-in link, as observed on 2026-11-24`, (await offered()) === "suspend_user,depart_user,issue_set_link", await offered());
      }
    }
    await page.unrouteAll({ behavior: "wait" });
  }
  {
    const words = {
      en: { never: "Not counted: never prepared", until: "Not counted until prepared", three: "3 people" },
      ko: { never: "집계 안 함: 준비되지 않음", until: "준비 전에는 집계하지 않음", three: "3명" },
      ar: { never: "لم يُحسب: لم يُحضَّر", until: "لا يُحسب قبل التحضير", three: "3 أشخاص" },
    };
    const zero = { targets: 0, excluded: 0, duplicates: 0, succeeded: 0, held: 0, failed: 0, unprocessed: 0 };
    const job = (id, state, prepared_at, counts = zero) => ({ id: "bulk-revocation-" + id, state, origin: "", effect: "suspend_membership", preparer: "sample-admin", prepared_at, expires_at: "2026-11-19T09:30:00Z", executor: "", executed_at: "", deny_at: "", digest: "", counts, job_state: "", job_reason: "", ledger: {}, cleanup: {}, recorded_at: "", staged: 0, selection: "explicit", matched: 0, fixed_at: "", fixed_revision: "", filter: null, search_masked: false });
    const added = [job(9101, "draft", ""), job(9102, "discarded", ""), job(9103, "expired", ""), job(9104, "discarded", "2026-11-19T09:00:00Z", { ...zero, targets: 3 })];
    const isJobs = (url) => new URL(url).pathname.endsWith("/api/v1/identity/bulk-revocations");
    let servedJobs = -1;
    await page.route((url) => isJobs(url.toString()), async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      if (servedJobs < 0) servedJobs = Array.isArray(body.jobs) ? body.jobs.length : 0;
      await route.fulfill({ response, json: { ...body, jobs: [...(Array.isArray(body.jobs) ? body.jobs : []), ...added] } });
    });
    for (const lang of ["en", "ko", "ar"]) {
      for (const width of widths) {
        await page.setViewportSize({ width, height: 1000 });
        await open(lang, "revocation");
        await page.locator(".m01-jobs-list").waitFor();
        const ended = page.locator(".m01-jobs details.m01-ended");
        if ((await ended.count()) > 0 && !(await ended.evaluate((e) => e.open))) await ended.locator("summary").click();
        const rowOf = async (id) => { const r = page.locator(".m01-jobs tr[role=row]").filter({ hasText: String(id) }); return (await r.count()) === 1 ? r.innerText() : `rows found: ${await r.count()}`; };
        const draft = await rowOf(9101), never = await rowOf(9102), lapsed = await rowOf(9103), prepared = await rowOf(9104);
        check(`${lang} ${width} a draft list is not counted until prepared`, draft.includes(words[lang].until) && !draft.includes(words[lang].never), draft);
        check(`${lang} ${width} a discarded list that was never prepared says so and shows no count of people`, never.includes(words[lang].never) && !/\d+\s*(people|person|명|أشخاص|شخص)/.test(never.replace(String(9102), "")), never);
        check(`${lang} ${width} an expired list that was never prepared says so`, lapsed.includes(words[lang].never), lapsed);
        check(`${lang} ${width} a discarded list that was prepared shows its count`, prepared.includes(words[lang].three) && !prepared.includes(words[lang].never), prepared);
        await contained(`${lang} ${width} revocation jobs with closed lists`);
        if (args.shots) await page.screenshot({ path: `${args.shots}/m01-jobs-${lang}-${width}.png`, fullPage: true });
      }
    }
    notes.push(`jobs served by the sample before the four added in the browser: ${servedJobs}`);
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
