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
  en: { shown: (n, m) => `Showing the newest ${n} of ${m} builds, newest first.`, all: (m) => `Showing all ${m} builds, newest first.`, more: "Load older builds", failed: "Older builds could not be loaded. The builds already shown are unchanged.", again: "Try again", reset: "The list was refreshed. The older builds you loaded are no longer shown.", moved: "Some builds were registered while this list was loading and are not shown here. Reload to see them at the top." },
  ko: { shown: (n, m) => `빌드 ${m}건 중 최근 ${n}건을 최신순으로 표시합니다.`, all: (m) => `빌드 ${m}건을 모두 최신순으로 표시합니다.`, more: "이전 빌드 불러오기", failed: "이전 빌드를 불러오지 못했습니다. 이미 표시된 빌드는 그대로입니다.", again: "다시 시도", reset: "목록을 새로 고쳤습니다. 불러온 이전 빌드는 더 이상 표시되지 않습니다.", moved: "목록을 불러오는 동안 등록된 빌드는 여기에 표시되지 않습니다. 새로 고치면 맨 위에 나옵니다." },
  ar: { shown: (n, m) => `تُعرض أحدث البنيات: ${n} من أصل ${m}، الأحدث أولًا.`, all: (m) => `تُعرض جميع البنيات (${m})، الأحدث أولًا.`, more: "تحميل بنيات أقدم", failed: "تعذّر تحميل البنيات الأقدم. لم تتغيّر البنيات المعروضة.", again: "إعادة المحاولة", reset: "حُدِّثت القائمة. لم تعد البنيات الأقدم التي حمّلتها معروضة.", moved: "بعض البنيات سُجّلت أثناء تحميل القائمة ولا تظهر هنا. أعد التحميل لعرضها في الأعلى." },
};
const open = async (lang) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/release-evidence?lang=${lang}&view=artifacts`);
  await page.locator("#p05-builds-title").waitFor();
  await page.waitForLoadState("networkidle");
};
const rowSel = ".p05-builds tbody";
const rows = () => page.locator(rowSel);

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
  const seen = async (r) => { if (ends(r.url(), "release-evidence") && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); } };
  page.on("response", seen);
  await open("en");
  page.off("response", seen);
  const real = served?.artifacts?.builds?.rows ?? [];
  const harnesses = served?.artifacts?.harnesses ?? {};
  notes.push(`served: ${real.length} builds, builds_next ${JSON.stringify(served?.artifacts?.builds_next)}; harnesses ${harnesses.rows?.length} of ${harnesses.total}, more ${harnesses.more}`);
  check("the real service lists the sample's builds whole, newest first, and says nothing further exists", real.length > 0 && served.artifacts.builds.total === real.length && served.artifacts.builds_next === "" && real.every((x, i) => i === 0 || x.order < real[i - 1].order));
  check("the real service lists every harness", Array.isArray(harnesses.rows) && harnesses.rows.length === harnesses.total && harnesses.more === false);
  check("with every build shown there is no count and no button", (await page.locator(".p05-builds-count, .p05-builds-more").count()) === 0 && (await rows().count()) === real.length);
  const further = await page.evaluate(async ([csrf]) => {
    const r = await fetch(`/api/v1/release-evidence/builds?limit=1`, { headers: { "X-CSRF-Token": csrf, "X-Request-Id": "qa-builds" }, credentials: "include" });
    return { status: r.status, body: await r.json().catch(() => null) };
  }, [token]);
  check("the real service answers the builds read with the newest build", further.status === 200 && further.body.total === real.length && further.body.order === "newest_first" && further.body.builds.length === 1 &&
    further.body.builds[0].build === real[0].build && (real.length > 1 ? further.body.next === String(real[0].order) : further.body.next === ""), JSON.stringify(further.body).slice(0, 300));
  const made = (n) => ({ build: `8.${n}.0`, commit: (n.toString(16).padStart(4, "0")).repeat(10), order: n });
  const down = (from, to) => Array.from({ length: from - to + 1 }, (_, i) => made(from - i));
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1, again = 0, short = false; const asked = [];
      await page.route((url) => ends(url.toString(), "release-evidence"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.revision = body.revision + (again ? "-again" + again : "");
        body.artifacts = { ...body.artifacts, builds: { total: 120, more: true, rows: down(120, 71) }, builds_next: "71" };
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "release-evidence/builds"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.get("after"));
        if (q.get("after") === "21" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        const body = q.get("after") === "71" ? { builds: down(70, 21), next: short ? "" : "21" } : { builds: down(20, 1), next: "" };
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "newest_first", q: "", matching: 120, total: 120, ...body } });
      });
      await open(lang);
      await page.locator(".p05-builds-count").waitFor();
      check(`${at} the newest fifty builds are listed with the count of all`, (await rows().count()) === 50 && (await page.locator(".p05-builds-count").innerText()).trim() === copy[lang].shown(50, 120) && (await rows().first().innerText()).includes("8.120.0"), await page.locator(".p05-builds-count").innerText());
      const more = page.locator(".p05-builds-more-button");
      check(`${at} the button is named and at least 44 high`, (await more.innerText()).trim() === copy[lang].more && (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      if (args.shots) { await page.locator("#p05-builds-title").evaluate((el) => el.scrollIntoView({ block: "start" })); await page.screenshot({ path: `${args.shots}/p05-builds-cut-${lang}-${width}.png`, fullPage: false }); }
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction((sel) => document.querySelectorAll(sel).length === 100, rowSel);
      check(`${at} the next fifty follow, older ones below`, asked[0] === "71" && (await page.locator(".p05-builds-count").innerText()).trim() === copy[lang].shown(100, 120) && (await rows().nth(50).innerText()).includes("8.70.0"));
      await page.keyboard.press("Enter");
      await page.locator(".p05-builds-more [role=alert]").waitFor();
      const alert = await page.locator(".p05-builds-more").innerText();
      check(`${at} a failed read says so, keeps the rows and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await rows().count()) === 100 && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await page.locator(".p05-builds-more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-builds-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction((sel) => document.querySelectorAll(sel).length === 120, rowSel);
      check(`${at} at the end all are listed, the button is gone and focus is on the count`, (await page.locator(".p05-builds-more").count()) === 0 && (await page.locator(".p05-builds-count").innerText()).trim() === copy[lang].all(120) && await page.locator(".p05-builds-count").evaluate((el) => el === document.activeElement));
      const ids = await rows().evaluateAll((els) => els.map((el) => el.id));
      check(`${at} no build is listed twice`, new Set(ids).size === 120, ids.slice(0, 3).join());
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (lang === "en" && width === 1440) {
        again = 1;
        await page.locator(".p05-reload").first().click();
        await page.waitForFunction((sel) => document.querySelectorAll(sel).length === 50, rowSel);
        check(`${at} after the screen is read again the list starts from the newest fifty and says so`, (await page.locator(".p05-builds-reset").innerText()).trim() === copy[lang].reset);
      }
      if (width === 1440) {
        short = true;
        await open(lang);
        await page.locator(".p05-builds-more-button").click();
        await page.locator(".p05-builds-moved").waitFor();
        check(`${at} when the walk ends below the total the screen says builds registered meanwhile are not shown`, (await page.locator(".p05-builds-moved").innerText()).trim() === copy[lang].moved && (await page.locator(".p05-builds-more").count()) === 0);
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
