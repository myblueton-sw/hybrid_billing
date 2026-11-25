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
  en: { shown: (n, t) => `${n} of ${t} closed proposals shown, newest first`, all: (t) => `All ${t} closed proposals shown, newest first`, more: "Load older closed proposals", failed: "Older closed proposals could not be loaded. The proposals already shown are unchanged.", again: "Try again", end: "Older closed proposals are not kept. Once more than 200 proposals exist, the service drops the oldest closed ones; proposals waiting for approval are always kept.", reset: "The list was refreshed and starts again from the newest proposal." },
  ko: { shown: (n, t) => `종료된 제안 ${t}건 중 ${n}건 표시, 최신순`, all: (t) => `종료된 제안 ${t}건 모두 표시, 최신순`, more: "이전 종료된 제안 불러오기", failed: "이전 종료된 제안을 불러오지 못했습니다. 이미 표시된 제안은 그대로입니다.", again: "다시 시도", end: "더 오래된 종료된 제안은 보관되지 않습니다. 제안이 200건을 넘으면 서비스가 가장 오래된 종료된 제안부터 삭제하며, 승인 대기 제안은 항상 보관합니다.", reset: "목록을 새로 고쳐 최신 제안부터 다시 표시합니다." },
  ar: { shown: (n, t) => `المقترحات المغلقة المعروضة: ${n} من أصل ${t}، الأحدث أولًا`, all: (t) => `المقترحات المغلقة المعروضة: جميعها (${t})، الأحدث أولًا`, more: "تحميل مقترحات مغلقة أقدم", failed: "تعذّر تحميل المقترحات المغلقة الأقدم. لم تتغيّر المقترحات المعروضة.", again: "إعادة المحاولة", end: "لا تُحفظ المقترحات المغلقة الأقدم. عند تجاوز عدد المقترحات 200، تحذف الخدمة أقدم المقترحات المغلقة، وتبقى المقترحات بانتظار الاعتماد محفوظة دائمًا.", reset: "حُدِّثت القائمة وتبدأ من جديد بأحدث مقترح." },
};
const screens = [
  { id: "p01", read: "installation", hash: (l) => `#/installation?lang=${l}`, list: (b) => b.proposals, put: (b, rows, facts) => ({ ...b, proposals: rows, proposal_history: facts }) },
  { id: "p02", read: "integrations", hash: (l) => `#/integrations?lang=${l}&view=modules`, list: (b) => b.modules?.proposals, put: (b, rows, facts) => ({ ...b, modules: { ...b.modules, proposals: rows, proposal_history: facts } }) },
  { id: "p03", read: "recovery", hash: (l) => `#/recovery?lang=${l}&view=resume`, list: (b) => b.proposals, put: (b, rows, facts) => ({ ...b, proposals: rows, proposal_history: facts }) },
];
const facts = { total: 120, next: "1071", kept: 200, dropped: 5 };

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

  for (const screen of screens) {
    const section = () => page.locator(`.${screen.id}-proposals`);
    const rows = () => section().locator(`.${screen.id}-list > li`);
    let served = null;
    let token = "";
    const seen = async (r) => { if (ends(r.url(), screen.read) && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); } };
    page.on("response", seen);
    await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}${screen.hash("en")}`);
    await section().waitFor();
    await page.waitForLoadState("networkidle");
    page.off("response", seen);
    const real = screen.list(served ?? {}) ?? [];
    const history = screen.id === "p02" ? served?.modules?.proposal_history : served?.proposal_history;
    notes.push(`${screen.id} served: ${real.length} proposals, proposal_history ${JSON.stringify(history)}`);
    check(`${screen.id} the service counts the proposals it keeps and has nothing further for the sample`, history && history.total === real.filter((p) => p.state !== "pending").length && history.next === "" && history.dropped === 0, JSON.stringify(history));
    check(`${screen.id} with nothing cut the screen shows no count, no button and no end sentence`, (await section().locator(".kept-count, .kept-more, .kept-end").count()) === 0);
    const further = await page.evaluate(async ([path, csrf]) => {
      const r = await fetch(`/api/v1/${path}/proposals?limit=5`, { headers: { "X-CSRF-Token": csrf, "X-Request-Id": "qa-proposal-history" }, credentials: "include" });
      return { status: r.status, body: await r.json().catch(() => null) };
    }, [screen.read, token]);
    const got = further.body ?? {};
    const realClosed = real.filter((p) => p.state !== "pending");
    check(`${screen.id} the real service answers the further read with the newest closed proposals it keeps`, further.status === 200 && Array.isArray(got.proposals) && got.proposals.length === Math.min(5, realClosed.length) &&
      got.matching === realClosed.length && got.order === "newest_first" && got.proposals.every((p, i) => p.number === realClosed[i].number) && (realClosed.length > 5 ? got.next === String(realClosed[4].number) : got.next === ""), JSON.stringify(got).slice(0, 300));
    notes.push(`${screen.id} sample: ${real.length - realClosed.length} waiting, ${realClosed.length} closed`);
    const template = real.find((p) => p.state !== "pending") ?? real[0];
    if (!template) { check(`${screen.id} the sample has a proposal to copy`, false); continue; }
    const made = (n) => ({ ...template, number: n, target: String(template.target).replace(/\d+$/, String(n)), state: template.state === "pending" ? "withdrawn" : template.state, can_approve: false, can_withdraw: false, approve_blocked: "not_pending", withdraw_blocked: "not_pending" });
    const span = (from, to) => Array.from({ length: from - to + 1 }, (_, i) => made(from - i));
    const waiting = real.filter((p) => p.state === "pending");
    for (const lang of ["en", "ko", "ar"]) {
      for (const width of [1440, 390]) {
        const at = `${screen.id} ${lang} ${width}`;
        await page.setViewportSize({ width, height: 1000 });
        let fail = 1, asked = [], revision = 0;
        await page.route((url) => ends(url.toString(), screen.read), async (route) => {
          const response = await route.fetch();
          const body = await response.json();
          await route.fulfill({ response, json: screen.put({ ...body, revision: body.revision + (revision ? "-again" + revision : "") }, [...waiting, ...span(1120, 1071)], facts) });
        });
        await page.route((url) => ends(url.toString(), screen.read + "/proposals"), async (route) => {
          const after = new URL(route.request().url()).searchParams.get("after");
          asked.push(after);
          if (after === "1021" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
          const body = after === "1071" ? { proposals: span(1070, 1021), next: "1021" } : { proposals: span(1020, 1001), next: "" };
          await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "newest_first", q: "", matching: 120, proposal_history: facts, ...body } });
        });
        await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}${screen.hash(lang)}`);
        await section().locator(".kept-count").waitFor();
        check(`${at} fifty proposals are listed with the count of all kept`, (await rows().count()) === 50 + waiting.length && (await section().locator(".kept-count").innerText()).trim() === copy[lang].shown(50, 120), await section().locator(".kept-count").innerText());
        check(`${at} the end sentence is not shown while older proposals can be read`, (await section().locator(".kept-end").count()) === 0);
        if (args.shots) { await section().locator(".kept-count").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/${screen.id}-history-cut-${lang}-${width}.png`, fullPage: false }); }
        const more = section().locator(".kept-more-button");
        check(`${at} the button to read older proposals is named`, (await more.innerText()).trim() === copy[lang].more);
        const size = await more.evaluate((el) => { const r = el.getBoundingClientRect(); return { h: r.height, w: r.width, view: document.documentElement.clientWidth }; });
        check(`${at} the button is at least 44 high and not stretched across a wide page`, size.h >= 44 && (width < 600 || size.w < 420), JSON.stringify(size));
        await more.click();
        await page.waitForFunction(([sel, extra]) => document.querySelectorAll(sel).length === 100 + extra, [`.${screen.id}-proposals .${screen.id}-list > li`, waiting.length]);
        check(`${at} the next fifty are added and counted`, asked[0] === "1071" && (await section().locator(".kept-count").innerText()).trim() === copy[lang].shown(100, 120));
        await more.focus();
        await page.keyboard.press("Enter");
        await section().locator(".kept-more [role=alert]").waitFor();
        const alert = await section().locator(".kept-more").innerText();
        check(`${at} focus stays on the button when the read fails`, await more.evaluate((el) => el === document.activeElement));
        check(`${at} a failed read says so and keeps the hundred shown`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await rows().count()) === 100 + waiting.length, alert);
        if (args.shots) { await section().locator(".kept-more").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/${screen.id}-history-failed-${lang}-${width}.png`, fullPage: false }); }
        await page.keyboard.press("Enter");
        await page.waitForFunction(([sel, extra]) => document.querySelectorAll(sel).length === 120 + extra, [`.${screen.id}-proposals .${screen.id}-list > li`, waiting.length]);
        const end = section().locator(".kept-end");
        check(`${at} at the end every kept proposal is listed, the button is gone and the screen says older ones are not kept`,
          (await section().locator(".kept-more").count()) === 0 && (await end.innerText()).trim() === copy[lang].end && (await section().locator(".kept-count").innerText()).trim() === copy[lang].all(120), JSON.stringify(asked));
        check(`${at} focus moves to the end sentence when the button goes away`, await end.evaluate((el) => el === document.activeElement));
        if (args.shots) { await end.scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/${screen.id}-history-last-${lang}-${width}.png`, fullPage: false }); }
        const numbers = await rows().evaluateAll((els) => els.map((el) => el.id || el.querySelector("[id]")?.id || ""));
        check(`${at} no proposal is listed twice`, new Set(numbers).size === numbers.length || numbers.every((x) => x === ""), numbers.slice(0, 4).join(","));
        const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
        check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
        if (args.shots) {
          await page.screenshot({ path: `${args.shots}/${screen.id}-history-end-${lang}-${width}.png`, fullPage: false });
          await section().locator(".kept-count").scrollIntoViewIfNeeded();
          await page.screenshot({ path: `${args.shots}/${screen.id}-history-top-${lang}-${width}.png`, fullPage: false });
        }
        if (lang === "en" && width === 1440) {
          revision = 1;
          const reload = page.locator(`.${screen.id}-reload`).first();
          if ((await reload.count()) > 0) {
            await reload.click();
            await page.waitForFunction(([sel, extra]) => document.querySelectorAll(sel).length === 50 + extra, [`.${screen.id}-proposals .${screen.id}-list > li`, waiting.length]);
            check(`${at} after the screen is read again the list starts from fifty and says so`, (await section().locator(".kept-reset").innerText()).trim() === copy[lang].reset);
          } else notes.push(`${screen.id} has no reload control the script could find; the read-again line was not exercised`);
        }
        await page.unrouteAll({ behavior: "wait" });
      }
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
