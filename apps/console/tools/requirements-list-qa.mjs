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
  en: { shown: (n, m) => `${n} of ${m} requirements shown, in the order recorded`, matching: (n, m) => `${n} of ${m} matching requirements shown`, more: "Load more requirements", failed: "The requirements could not be loaded. The requirements already shown are unchanged.", again: "Try again", none: "No requirement matches the search and filters.", reset: "The list was refreshed and starts again from the first requirement.", unlisted: "The requirement open in the detail is not in the list shown here.", clear: "Clear search and filters" },
  ko: { shown: (n, m) => `요구사항 ${m}건 중 ${n}건 표시, 기록된 순서`, matching: (n, m) => `일치하는 요구사항 ${m}건 중 ${n}건 표시`, more: "요구사항 더 불러오기", failed: "요구사항을 불러오지 못했습니다. 이미 표시된 요구사항은 그대로입니다.", again: "다시 시도", none: "검색과 필터에 일치하는 요구사항이 없습니다.", reset: "목록을 새로 고쳐 첫 요구사항부터 다시 표시합니다.", unlisted: "상세에 열린 요구사항은 여기 표시된 목록에 없습니다.", clear: "검색과 필터 지우기" },
  ar: { shown: (n, m) => `المتطلبات المعروضة: ${n} من أصل ${m}، بترتيب التسجيل`, matching: (n, m) => `المتطلبات المطابقة المعروضة: ${n} من أصل ${m}`, more: "تحميل متطلبات أخرى", failed: "تعذّر تحميل المتطلبات. لم تتغيّر المتطلبات المعروضة.", again: "إعادة المحاولة", none: "لا يوجد متطلب يطابق البحث وعوامل التصفية.", reset: "حُدِّثت القائمة وتبدأ من جديد بأول متطلب.", unlisted: "المتطلب المفتوح في التفاصيل غير موجود في القائمة المعروضة هنا.", clear: "مسح البحث وعوامل التصفية" },
};
const rows = () => page.locator(".p05-requirements tbody");
const targets = () => page.locator(".p05-requirements tbody[id^=p05-requirement-]").evaluateAll((els) => els.map((el) => el.id.replace("p05-requirement-", "")));
const open = async (lang, extra = "") => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/release-evidence?lang=${lang}&view=requirements${extra}`);
  await page.locator(extra ? "#p05-detail-title" : ".p05-requirements").waitFor();
  await page.waitForLoadState("networkidle");
};
const settle = async () => { await page.waitForLoadState("networkidle"); await page.waitForFunction(() => !document.querySelector(".p05-live[aria-busy=true]")); };

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

  let served = null;
  const seen = async (r) => { if (ends(r.url(), "release-evidence") && r.status() === 200 && !served) served = await r.json().catch(() => null); };
  page.on("response", seen);
  await open("en");
  page.off("response", seen);
  const real = served?.requirements?.requirements?.rows ?? [];
  notes.push(`served: ${real.length} requirements, next ${JSON.stringify(served?.requirements?.next)}, states ${real.map((r) => r.state).join(",")}`);
  check("the real service lists the sample's requirements, says nothing further exists and gives each row's state", real.length === 6 && served.requirements.next === "" && real.every((r) => typeof r.state === "string"));
  check("with the whole list shown there is no count and no button", (await page.locator(".p05-req-count, .p05-req-more").count()) === 0 && (await rows().count()) === 6);

  const asked = [];
  const note = (r) => { if (ends(r.url(), "release-evidence/requirements")) asked.push(new URL(r.url()).search); };
  page.on("request", note);
  await page.locator("#p05-req-q").fill("ERP");
  await page.locator("#p05-req-q").press("Enter");
  await page.waitForFunction(() => document.querySelectorAll(".p05-requirements tbody").length === 1);
  await settle();
  check("a search is answered by the real service and lists the one matching requirement", asked.at(-1) === "?q=ERP" && (await targets()).join() === "4" && (await page.locator(".p05-req-count").innerText()).trim() === copy.en.matching(1, 1), asked.join(" ") + " " + (await targets()).join());
  const tested = real.filter((r) => r.state === "tested").map((r) => r.target.replace("trace-requirement-", ""));
  await page.locator("#p05-req-q").fill("");
  await page.locator("#p05-req-state").selectOption("tested");
  await page.locator(".p05-req-apply").click();
  await page.waitForFunction((n) => document.querySelectorAll(".p05-requirements tbody").length === n, tested.length);
  await settle();
  check("a state filter is answered by the real service with the requirements in that state", asked.at(-1) === "?state=tested" && (await targets()).join() === tested.join() && tested.length > 0, asked.at(-1) + " " + (await targets()).join());
  const retired = real.filter((r) => r.retired).length;
  await page.locator("#p05-req-state").selectOption("");
  await page.locator("#p05-req-retired").selectOption("yes");
  await page.locator(".p05-req-apply").click();
  await settle();
  check("the retired filter is answered by the real service", asked.at(-1) === "?retired=yes" && (await rows().count()) === retired && (retired > 0 || (await page.locator(".p05-list").innerText()).includes(copy.en.none)), asked.at(-1) + " " + retired);
  await page.locator("#p05-req-q").fill("zz-no-such");
  await page.locator("#p05-req-retired").selectOption("");
  await page.locator(".p05-req-apply").click();
  await page.locator(".p05-list").getByText(copy.en.none).waitFor();
  check("a search without a match says so and offers to clear", (await page.locator(".p05-req-clear").innerText()).trim() === copy.en.clear);
  const before = asked.length;
  await page.locator("#p05-req-q").fill("가나다라마바사아자차카타파하가나다라마바사아자차");
  await page.locator("#p05-req-q").press("Enter");
  await page.locator("#p05-req-q-error").waitFor();
  check("a search longer than the service accepts is refused on the screen and not sent", asked.length === before && (await page.locator("#p05-req-q").getAttribute("aria-invalid")) === "true" && (await page.locator(".p05-live .live-error").count()) === 0);
  if (args.shots) await page.screenshot({ path: `${args.shots}/p05-req-nomatch-toolong-en-1440.png`, fullPage: false });
  await page.locator(".p05-req-clear").click();
  await page.waitForFunction(() => document.querySelectorAll(".p05-requirements tbody").length === 6);
  check("clearing returns to the whole list without asking the service", (await targets()).join() === "1,2,3,4,5,6");
  page.off("request", note);

  const template = real[1];
  const made = (n) => ({ ...template, target: `trace-requirement-${n}`, id: `routed-${n}`, label: `${template.namespace}/routed-${n}`, retire: null, can_retire: false, retire_blocked: "no_eligible_approver", replacements: { total: 0, more: false, rows: [] } });
  const span = (from, to) => Array.from({ length: to - from + 1 }, (_, i) => made(from + i));
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1, failFilter = 1; const got = [];
      await page.route((url) => ends(url.toString(), "release-evidence"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.requirements.requirements = { total: 70, more: true, rows: body.requirements.requirements.rows };
        body.requirements.next = "6";
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "release-evidence/requirements"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        got.push(q.toString());
        const answer = (rows, next) => route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "recorded", q: "", state: "", retired: "", requirements: rows, next, matching: q.has("target") ? rows.length : 70, total: 70 } });
        if (q.has("state")) { if (failFilter-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } }); return answer(span(7, 9), ""); }
        if (q.has("target")) return answer(q.get("target").split(",").map((t) => made(Number(t.replace("trace-requirement-", "")))), "");
        if (q.get("after") === "56" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        return q.get("after") === "6" ? answer(span(7, 56), "56") : answer(span(57, 70), "");
      });
      await open(lang);
      check(`${at} six of seventy are listed with the count and the button`, (await rows().count()) === 6 && (await page.locator(".p05-req-count").innerText()).trim() === copy[lang].shown(6, 70) && (await page.locator(".p05-req-more-button").innerText()).trim() === copy[lang].more);
      const more = page.locator(".p05-req-more-button");
      await more.focus();
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll(".p05-requirements tbody").length === 56);
      check(`${at} the next fifty are added after the main rows and counted`, got[0] === "after=6" && (await page.locator(".p05-req-count").innerText()).trim() === copy[lang].shown(56, 70) && (await targets()).slice(0, 8).join() === "1,2,3,4,5,6,7,8");
      await page.keyboard.press("Enter");
      await page.locator(".p05-req-more [role=alert]").waitFor();
      const alert = await page.locator(".p05-req-more").innerText();
      check(`${at} a failed read says so, keeps the rows and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await rows().count()) === 56 && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await page.locator(".p05-req-more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-req-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction(() => document.querySelectorAll(".p05-requirements tbody").length === 70);
      check(`${at} at the end all seventy are listed and the button is gone`, (await page.locator(".p05-req-more").count()) === 0 && (await page.locator(".p05-req-count").count()) === 0);
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (args.shots) { await page.locator(".p05-req-form").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-req-controls-${lang}-${width}.png`, fullPage: false }); }
      const form = await page.locator(".p05-req-form").evaluate((el) => ({ buttons: [...el.querySelectorAll("button")].map((b) => b.getBoundingClientRect().height), fields: [...el.querySelectorAll(".service-field")].map((f) => f.getBoundingClientRect().width),
        inputs: [...el.querySelectorAll("input, select")].map((f) => Math.round(f.getBoundingClientRect().height)), labels: [...el.querySelectorAll("label")].map((f) => Math.round(f.getBoundingClientRect().top)) }));
      check(`${at} the controls are not stretched, are one height and their buttons are one line`, form.buttons.every((h) => h >= 44 && h <= 60) && form.fields.every((w) => w <= 281) && form.inputs.every((h) => h === 44) && (width < 600 || new Set(form.labels).size === 1), JSON.stringify(form));
      got.length = 0;
      await page.locator("#p05-requirement-60 a").first().click();
      await page.locator("#p05-detail-title").waitFor();
      const title = await page.locator("#p05-detail-title").innerText();
      check(`${at} a requirement loaded beyond the main read opens its detail`, title.includes("routed-60"), title);
      await page.locator(".p05-reload").click();
      await settle();
      const still = (await page.locator("#p05-detail-title").count()) > 0 ? await page.locator("#p05-detail-title").innerText() : "";
      check(`${at} after reading the screen again the open requirement is read by target and stays open, and nothing is reported as gone`, got.includes("target=trace-requirement-60") && still.includes("routed-60") && (await page.locator(".p05-vanished").count()) === 0, got.join(" ") + " | " + still);
      if (width === 1440) {
        check(`${at} the list starts again from the main rows and says so, and says the open requirement is not listed`, (await rows().count()) === 6 && (await page.locator(".p05-req-reset").innerText()).trim() === copy[lang].reset && (await page.locator(".p05-req-unlisted").innerText()).trim() === copy[lang].unlisted);
        if (args.shots) await page.screenshot({ path: `${args.shots}/p05-req-reread-${lang}-${width}.png`, fullPage: false });
      }
      if (width === 1440) {
        await open(lang);
        await page.locator("#p05-req-state").selectOption("tested");
        await page.locator(".p05-req-apply").click();
        await page.locator(".p05-req-more [role=alert]").waitFor();
        const text = await page.locator(".p05-list").innerText();
        check(`${at} a failed filter says the read failed and does not say that nothing matches`, text.includes(copy[lang].failed) && !text.includes(copy[lang].none) && (await page.locator(".p05-req-count").count()) === 0, text.slice(0, 300));
        await page.locator(".p05-req-more-button").click();
        await page.waitForFunction(() => document.querySelectorAll(".p05-requirements tbody").length === 3);
        check(`${at} trying again lists the matching rows with their count above the table`, (await page.locator(".p05-req-controls .p05-req-count").innerText()).trim() === copy[lang].matching(3, 70));
        if (args.shots) { await page.locator(".p05-req-form").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-req-filtered-${lang}-${width}.png`, fullPage: false }); }
        await page.locator(".p05-reload").click();
        await settle();
        check(`${at} after the screen is read again the filter is asked again and the page is not failed`, (await rows().count()) === 3 && (await page.locator(".p05-failure").count()) === 0);
      }
      await open(lang, "&requirement=65");
      await page.locator("#p05-detail-title").waitFor();
      check(`${at} a link to a requirement outside the main read opens it`, (await page.locator("#p05-detail-title").innerText()).includes("routed-65"));
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
