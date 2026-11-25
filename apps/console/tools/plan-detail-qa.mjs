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
const copy = {
  en: "Not all reasons are shown. The plan was rejected for more reasons than are listed here.",
  ko: "일부 사유만 표시합니다. 여기 표시된 것보다 거부 사유가 더 많습니다.",
  ar: "لم تُعرض كل الأسباب. رُفضت الخطة لأسباب أكثر مما هو مدرج هنا.",
};
const open = async (lang, plan) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/organization-changes?lang=${lang}&view=impact&plan=${encodeURIComponent(plan)}`);
  await page.locator("#o02-plan-title").waitFor();
  await page.waitForLoadState("networkidle");
};

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

  const blank = { closed: "", created: "2026-11-24T00:00:00Z", proposer: "sample-admin-2", proposer_login: "admin2@example.invalid", reason: "synthetic rendered check",
    change: { kind: "hierarchy", member: "sample-acct-01", action: "set", target: "sample-unit-north-data", from: "2026-12-01" }, member_name: "Account 01 (sample)", target_name: "", base: 1, stale: false, activated_revision: 0,
    violations: [], truncated: false, members: [], kinds: [], impact: { money: "not_computed", publication: "not_computed", erp: "not_computed" }, can_activate: false, blocked: "impact_not_computed", draft: null };
  const row = (kind, n) => ({ kind, member: `sample-acct-0${n}`, name: `Account 0${n} (sample)`, differs: [{ from: "2026-12-01", until: "" }], before: [], after: ["sample-unit-north-data"] });
  const kind = (k, n) => ({ kind: k, changed: n, entries_before: n, entries_after: n, earliest: "2026-12-01", next: "" });
  const rejected = { ...blank, id: "qa-rejected", state: "rejected", closed: "rejected", truncated: true, violations: Array.from({ length: 3 }, () => ({ kind: "hierarchy", member: "sample-acct-01", rule: "no_change" })) };
  const several = { ...blank, id: "qa-two-kinds", state: "sealed", kinds: [kind("hierarchy", 2), kind("payer", 1)], members: [row("hierarchy", 1), row("hierarchy", 2), row("payer", 1)] };
  const single = { ...blank, id: "qa-one-kind", state: "sealed", kinds: [kind("hierarchy", 2)], members: [row("hierarchy", 1), row("hierarchy", 2)] };
  await page.route((url) => url.pathname === "/api/v1/organizations/plans", async (route) => {
    const response = await route.fetch();
    const body = await response.json();
    body.plans = [several, single, rejected, ...body.plans];
    body.open_plans = Math.max(body.open_plans ?? 0, 2);
    await route.fulfill({ response, json: body });
  });
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      await open(lang, rejected.id);
      const cut = page.locator(".o02-violations + .o02-violations-cut");
      check(`${at} a plan whose reasons were cut says so at the foot of the reasons, in words about reasons`, (await cut.count()) === 1 && (await cut.innerText()).trim() === copy[lang] && (await page.locator(".o02-violations li").count()) === 3 && (await page.locator(".live-note-rule").filter({ hasText: copy[lang] }).count()) === 0);
      if (args.shots) { await cut.scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/o02-reasons-cut-${lang}-${width}.png`, fullPage: false }); }
      await open(lang, several.id);
      const kinds = await page.locator(".o02-diff .o02-member-kind").allInnerTexts();
      check(`${at} a plan of two kinds names the kind under each changed member`, kinds.length === 3 && new Set(kinds).size === 2 && kinds.every((k) => k.trim() !== ""), kinds.join("|"));
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (args.shots) { await page.locator(".o02-diff").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/o02-two-kinds-${lang}-${width}.png`, fullPage: false }); }
      await open(lang, single.id);
      check(`${at} a plan of one kind shows no kind under its members`, (await page.locator(".o02-diff tbody tr").count()) === 2 && (await page.locator(".o02-member-kind").count()) === 0);
    }
  }
  await page.unrouteAll({ behavior: "wait" });
  check("no page error", errors.length === 0, errors.join(" | "));
} catch (e) {
  check("the script ran to the end", false, e.stack || e.message);
} finally {
  await browser.close();
  for (const r of results) console.log(r);
  for (const n of notes) console.log("NOTE " + n);
  console.log(`RESULT ${results.filter((r) => r.startsWith("PASS")).length} passed ${results.filter((r) => r.startsWith("FAIL")).length} failed`);
}
