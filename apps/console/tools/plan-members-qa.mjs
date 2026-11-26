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
const kindWord = { en: "Hierarchy", ko: "", ar: "" };
const copy = {
  en: { shown: (n, m) => `Showing ${n} of ${m} changed members.`, more: "Load more changed members: ", failed: "More changed members could not be loaded. The members already shown are unchanged.", again: "Try again: " },
  ko: { shown: (n, m) => `변경된 구성원 ${m}개 중 ${n}개를 표시합니다.`, more: "변경된 구성원 더 불러오기: ", failed: "변경된 구성원을 더 불러오지 못했습니다. 이미 표시된 구성원은 그대로입니다.", again: "다시 시도: " },
  ar: { shown: (n, m) => `يُعرض ${n} من ${m} من الأعضاء المتغيّرين.`, more: "تحميل أعضاء متغيّرين آخرين: ", failed: "تعذّر تحميل أعضاء متغيّرين آخرين. لم يتغيّر الأعضاء المعروضون.", again: "إعادة المحاولة: " },
};
const pad = (n) => String(n).padStart(3, "0");
const open = async (lang, plan) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/organization-changes?lang=${lang}&view=impact&plan=${encodeURIComponent(plan)}`);
  await page.locator("#o02-plan-title").waitFor();
  await page.waitForLoadState("networkidle");
};
const rows = () => page.locator(".o02-diff tbody tr");

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

  let served = null, token = "";
  const seen = async (r) => { if (r.url().split("?")[0].endsWith("/api/v1/organizations/plans") && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); } };
  page.on("response", seen);
  await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#/organization-changes?lang=en&view=impact`);
  await page.waitForLoadState("networkidle");
  page.off("response", seen);
  const plans = served?.plans ?? [];
  notes.push(`served plans ${plans.length}; the sample has none, so the plan is written by the check`);
  const kind = "hierarchy";
  const real = plans.find((p) => p.members.length > 0) ?? { id: "qa-plan-members", state: "sealed", closed: "", created: "2026-11-24T00:00:00Z", proposer: "sample-admin-2", proposer_login: "admin2@example.invalid", reason: "synthetic rendered check",
    change: { kind, member: "sample-acct-01", action: "set", target: "sample-unit-north-data", from: "2026-12-01" }, member_name: "Account 01 (sample)", target_name: "", base: served?.active_revision ?? 1, stale: false, activated_revision: 0,
    kinds: [], violations: [], truncated: false, members: [], impact: { money: "not_computed", publication: "not_computed", erp: "not_computed" }, can_activate: false, blocked: "impact_not_computed", draft: null };
  const further = await page.evaluate(async ([csrf, k]) => {
    const head = { "X-CSRF-Token": csrf, "X-Request-Id": "qa-plan-members" };
    const get = async (u) => { const r = await fetch(u, { headers: head, credentials: "include" }); return { status: r.status, body: await r.json().catch(() => null) }; };
    return { a: await get(`/api/v1/organizations/plans/members?plan=never-proposed&kind=${k}`), b: await get(`/api/v1/organizations/plans/members?plan=never-proposed&kind=${k}&limit=51`) };
  }, [token, kind]);
  check("the real service has the plan members read and refuses an unknown plan and a limit above fifty", further.a.status === 422 && further.b.status === 422, JSON.stringify(further).slice(0, 300));
  const template = { differs: [{ from: "2026-12-01", until: "" }], before: [], after: ["sample-unit-north-data"] };
  const made = (n) => ({ ...template, kind, member: `zz-member-${pad(n)}`, name: `Member ${pad(n)}` });
  const span = (from, to) => Array.from({ length: to - from + 1 }, (_, i) => made(from + i));
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1; const asked = [];
      await page.route((url) => url.pathname === "/api/v1/organizations/plans", async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        const shown = { ...real, state: "sealed", members: span(1, 50), kinds: [{ kind, changed: 70, entries_before: 70, entries_after: 70, earliest: "2026-12-01", next: "zz-member-050" }] };
        body.plans = [shown, ...body.plans.filter((p) => p.id !== real.id)];
        body.open_plans = Math.max(body.open_plans ?? 0, 1);
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => url.pathname === "/api/v1/organizations/plans/members", async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.toString());
        if (fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", plan: q.get("plan"), kind: q.get("kind"), order: "identifier", members: span(51, 70), next: "", total: 70 } });
      });
      await open(lang, real.id);
      const more = page.locator(".o02-members-more-button");
      const label = (await more.innerText()).trim();
      check(`${at} the first fifty are listed with the count of all and a button that names the kind`, (await rows().count()) === 50 && (await page.locator(".o02-members-count").innerText()).trim() === copy[lang].shown(50, 70) && label.startsWith(copy[lang].more) && label.length > copy[lang].more.length, label);
      check(`${at} the button is at least 44 high`, (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      await more.focus();
      await page.keyboard.press("Enter");
      await page.locator(".o02-members-more [role=alert]").waitFor();
      const alert = await page.locator(".o02-members-more").innerText();
      check(`${at} a failed read says so, keeps the members and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await rows().count()) === 50 && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await page.locator(".o02-members-foot").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/o02-members-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll(".o02-diff tbody tr").length === 70);
      check(`${at} the rest follow after the last member shown and the foot is gone`, asked[0] === new URLSearchParams({ plan: real.id, kind, limit: "50", after: "zz-member-050" }).toString() && (await page.locator(".o02-members-foot").count()) === 0, asked[0]);
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      await page.unrouteAll({ behavior: "wait" });
    }
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
