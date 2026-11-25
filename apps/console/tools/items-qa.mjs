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
  en: { shown: (n, m) => `Showing ${n} of ${m} items, sorted by item.`, more: "Load more items", failed: "More items could not be loaded. The items already shown are unchanged.", again: "Try again", reset: "The list was refreshed and starts again from the first item." },
  ko: { shown: (n, m) => `${m}건 중 ${n}건을 항목 순으로 표시합니다.`, more: "항목 더 불러오기", failed: "항목을 더 불러오지 못했습니다. 이미 표시된 항목은 그대로입니다.", again: "다시 시도", reset: "목록을 새로 고쳐 첫 항목부터 다시 표시합니다." },
  ar: { shown: (n, m) => `تُعرض ${n} من أصل ${m} عنصرًا، مرتبةً حسب العنصر.`, more: "تحميل عناصر أخرى", failed: "تعذّر تحميل عناصر أخرى. لم تتغيّر العناصر المعروضة.", again: "إعادة المحاولة", reset: "حُدِّثت القائمة وتبدأ من جديد بأول عنصر." },
};
const open = async (lang, view, extra = "") => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/retention?lang=${lang}&view=${view}${extra}`);
  await page.locator(view === "expiry" ? ".p04-class" : ".p04-required").first().waitFor();
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => !document.querySelector(".p04-live[aria-busy=true]"));
};
const pad = (n) => String(n).padStart(3, "0");

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
  const seen = async (r) => { if (ends(r.url(), "retention") && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); } };
  page.on("response", seen);
  await open("en", "expiry");
  page.off("response", seen);
  const group = (served?.expiry ?? []).find((g) => g.artifacts.length > 0);
  const cls = group?.class ?? "";
  const real = group?.artifacts ?? [];
  notes.push(`served: class ${cls} with ${real.length} items, total ${group?.total}, next ${JSON.stringify(group?.next)}; references ${served?.holds?.references?.rows?.length}, next ${JSON.stringify(served?.holds?.references?.next)}`);
  check("the real service lists the sample's items whole and says nothing further exists", real.length > 0 && (served.expiry ?? []).every((g) => g.total === g.artifacts.length && g.next === "" && g.more === false) && served.holds.references.next === "");
  check("with every item shown there is no count and no button", (await page.locator(".p04-items-count, .p04-items-more").count()) === 0);
  const further = await page.evaluate(async ([csrf, c, id]) => {
    const head = { "X-CSRF-Token": csrf, "X-Request-Id": "qa-items" };
    const get = async (u) => { const r = await fetch(u, { headers: head, credentials: "include" }); return { status: r.status, body: await r.json().catch(() => null) }; };
    return { a: await get(`/api/v1/retention/artifacts?class=${c}&limit=1`), b: await get(`/api/v1/retention/artifacts?id=${encodeURIComponent(id)}`), c: await get(`/api/v1/retention/references?limit=5`) };
  }, [token, cls, real[real.length - 1].id]);
  check("the real service answers the artifacts read by class and by identifier, and the references read", further.a.status === 200 && further.a.body.order === "identifier" && further.a.body.total === real.length && further.a.body.artifacts[0].id === real[0].id && further.a.body.artifacts[0].class === cls &&
    further.b.status === 200 && further.b.body.artifacts.length === 1 && further.b.body.artifacts[0].id === real[real.length - 1].id && further.c.status === 200 && further.c.body.references.length === served.holds.references.rows.length, JSON.stringify(further).slice(0, 400));
  const made = (n) => ({ ...real[0], id: `zz-routed-${pad(n)}`, hold_target: `retention-artifact:zz-routed-${pad(n)}`, run: null, other_scope: false });
  const span = (from, to) => Array.from({ length: to - from + 1 }, (_, i) => made(from + i));
  const base0 = real.length;
  const rows = () => page.locator(`#p04-class-${cls} table.p04-items tbody.p04-row`);
  const foot = (name) => page.locator(`#p04-class-${cls} .p04-items-${name}`);
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1, again = 0; const asked = [];
      await page.route((url) => ends(url.toString(), "retention"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.revision = body.revision + (again ? "-again" + again : "");
        body.expiry = body.expiry.map((g) => (g.class === cls ? { ...g, total: base0 + 70, more: true, next: real[real.length - 1].id } : g));
        body.holds.references = { ...body.holds.references, total: body.holds.references.rows.length + 30, more: true, next: "zz-ref-start" };
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "retention/artifacts"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.toString());
        const answer = (list, next) => route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "identifier", class: q.get("class") ?? "", q: "", artifacts: list.map((a) => ({ class: cls, ...a })), next, matching: list.length, total: base0 + 70 } });
        if (q.has("id")) return answer(q.get("id").split(",").filter((id) => id.startsWith("zz-routed-")).map((id) => made(Number(id.slice(10)))), "");
        if (q.get("after") === "zz-routed-050" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        return q.get("after") === "zz-routed-050" ? answer(span(51, 70), "") : answer(span(1, 50), "zz-routed-050");
      });
      await page.route((url) => ends(url.toString(), "retention/references"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push("refs " + q.toString());
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "identifier", q: "", references: Array.from({ length: 30 }, (_, i) => ({ id: `zz-ref-${pad(i + 1)}`, class: cls })), next: "", matching: 30, total: 30 + served.holds.references.rows.length } });
      });
      await open(lang, "expiry");
      check(`${at} the first items are listed with the count of all and one button for the class`, (await rows().count()) === base0 && (await foot("count").innerText()).trim() === copy[lang].shown(base0, base0 + 70) && (await foot("more-button").innerText()).trim() === copy[lang].more && (await page.locator(".p04-items-more-button").count()) === 1);
      const more = foot("more-button");
      check(`${at} the button is at least 44 high`, (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction(([c, n]) => document.querySelectorAll(`#p04-class-${c} table.p04-items tbody.p04-row`).length === n, [cls, base0 + 50]);
      check(`${at} the next items follow in identifier order`, asked[0] === `class=${cls}&limit=50&after=${encodeURIComponent(real[real.length - 1].id)}` && (await foot("count").innerText()).trim() === copy[lang].shown(base0 + 50, base0 + 70) && (await rows().nth(base0).getAttribute("id")) === "p04-item-zz-routed-001", asked[0]);
      await page.keyboard.press("Enter");
      await foot("more").locator("[role=alert]").waitFor();
      const alert = await foot("more").innerText();
      check(`${at} a failed read says so, keeps the items and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await rows().count()) === base0 + 50 && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await foot("more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-items-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction(([c, n]) => document.querySelectorAll(`#p04-class-${c} table.p04-items tbody.p04-row`).length === n, [cls, base0 + 70]);
      check(`${at} at the end all are listed and the count and button are gone`, (await foot("more").count()) === 0 && (await foot("count").count()) === 0);
      const ids = await rows().evaluateAll((els) => els.map((el) => el.id));
      check(`${at} no item is listed twice`, new Set(ids).size === base0 + 70, ids.slice(0, 3).join());
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (width === 1440) {
        again += 1;
        await page.locator(".p04-reload").first().click();
        await page.waitForFunction(([c, n]) => document.querySelectorAll(`#p04-class-${c} table.p04-items tbody.p04-row`).length === n, [cls, base0]);
        check(`${at} after the screen is read again the list starts from the main read and says so`, (await foot("reset").innerText()).trim() === copy[lang].reset && (await page.locator(".p04-vanished").count()) === 0);
        if (args.shots) { await foot("count").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p04-items-reset-${lang}-${width}.png`, fullPage: false }); }
        asked.length = 0;
        await open(lang, "expiry", `&class=${cls}&artifact=${encodeURIComponent("a,b")}`);
        check(`${at} a link with a malformed item is ignored`, asked.length === 0 && (await rows().count()) === base0, asked.join(" "));
        asked.length = 0;
        await open(lang, "expiry", `&class=${cls}&artifact=zz-routed-065`);
        await page.locator("#p04-item-zz-routed-065").waitFor();
        check(`${at} a link to an item outside the main read reads it by identifier and selects it`, asked.includes("id=zz-routed-065") && (await page.locator("#p04-item-zz-routed-065").getAttribute("class")).includes("live-selected"), asked.join(" "));
        asked.length = 0;
        await open(lang, "holds");
        const refRows = () => page.locator(".p04-required table tbody tr");
        const n0 = served.holds.references.rows.length;
        check(`${at} the references show their count and button`, (await refRows().count()) === n0 && (await page.locator(".p04-refs-count").innerText()).trim() === copy[lang].shown(n0, n0 + 30) && (await page.locator(".p04-refs-more-button").innerText()).trim() === copy[lang].more);
        await page.locator(".p04-refs-more-button").focus();
        await page.keyboard.press("Enter");
        await page.waitForFunction((n) => document.querySelectorAll(".p04-required table tbody tr").length === n, n0 + 30);
        check(`${at} the rest of the references follow and the foot is gone`, asked.includes("refs limit=50&after=zz-ref-start") && (await page.locator(".p04-refs-more, .p04-refs-count").count()) === 0, asked.join(" "));
      }
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
