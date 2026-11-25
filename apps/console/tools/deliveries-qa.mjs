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
  en: { more: (s) => `Load more deliveries for ${s}`, failed: "More deliveries could not be loaded. The deliveries already shown are unchanged.", again: "Try again", reset: "The list was refreshed. The further deliveries you loaded are no longer shown.", moves: "Deliveries can change while they are dispatched. Reload to see them as they are now." },
  ko: { more: (s) => `${s}의 발송 건 더 불러오기`, failed: "발송 건을 더 불러오지 못했습니다. 이미 표시된 발송 건은 그대로입니다.", again: "다시 시도", reset: "목록을 새로 고쳤습니다. 더 불러온 발송 건은 더 이상 표시되지 않습니다.", moves: "발송이 진행되면 발송 건이 바뀔 수 있습니다. 지금 상태를 보려면 새로 고침하세요." },
  ar: { more: (s) => `تحميل عمليات تسليم أخرى للاشتراك ${s}`, failed: "تعذّر تحميل عمليات تسليم أخرى. لم تتغيّر عمليات التسليم المعروضة.", again: "إعادة المحاولة", reset: "حُدِّثت القائمة. لم تعد عمليات التسليم الإضافية التي حمّلتها معروضة.", moves: "قد تتغيّر عمليات التسليم أثناء الإرسال. أعد التحميل لرؤيتها كما هي الآن." },
};
const open = async (lang, extra = "") => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/integrations?lang=${lang}&view=events${extra}`);
  await page.locator(".p02-deliveries").first().waitFor();
  await page.waitForLoadState("networkidle");
};
const rows = () => page.locator(".p02-deliveries tbody.p02-group");

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
  const seen = async (r) => { if (ends(r.url(), "integrations") && r.status() === 200 && !served) { token = r.request().headers()["x-csrf-token"] ?? ""; served = await r.json().catch(() => null); } };
  page.on("response", seen);
  await open("en");
  page.off("response", seen);
  const real = served?.events?.deliveries ?? [];
  const subs = [...new Set(real.map((d) => d.subscription))].sort();
  notes.push(`served: ${real.length} deliveries of ${subs.length} subscriptions, deliveries_more ${JSON.stringify(served?.events?.deliveries_more)}, deliveries_next ${JSON.stringify(served?.events?.deliveries_next)}`);
  check("the real service lists the sample's deliveries whole and has no cursor", real.length > 0 && served.events.deliveries_more.length === 0 && Object.keys(served.events.deliveries_next).length === 0);
  check("with every delivery shown there is no button", (await page.locator(".p02-deliveries-foot").count()) === 0 && (await rows().count()) === real.length);
  const plain = real.filter((d) => d.kind !== "dead" && d.hold === "");
  const further = await page.evaluate(async ([csrf, id]) => {
    const head = { "X-CSRF-Token": csrf, "X-Request-Id": "qa-deliveries" };
    const get = async (u) => { const r = await fetch(u, { headers: head, credentials: "include" }); return { status: r.status, body: await r.json().catch(() => null) }; };
    return { a: await get(`/api/v1/integrations/deliveries?subscription=${encodeURIComponent(id)}`), b: await get(`/api/v1/integrations/deliveries?subscription=never-proposed`) };
  }, [token, plain[0].subscription]);
  const mine = plain.filter((d) => d.subscription === plain[0].subscription);
  check("the real service answers the deliveries read with the deliveries that need no action and refuses an unknown subscription", further.a.status === 200 && further.a.body.rows_can_move === true && further.a.body.total === mine.length &&
    further.a.body.deliveries.map((d) => d.id).sort().join() === mine.map((d) => d.id).sort().join() && further.b.status === 422, JSON.stringify(further).slice(0, 400));
  const [first, second] = [subs[0], subs[1] ?? subs[0]];
  const template = plain[0];
  const made = (sub, n) => ({ ...template, id: `zz-${sub}-${n}`, target: `zz-target-${sub}-${n}`, subscription: sub, sequence: 5000 + n + (sub === first ? 0 : 1000), event_id: `zz-event-${n}` });
  const span = (sub, from, to) => Array.from({ length: to - from + 1 }, (_, i) => made(sub, from + i));
  const cut = [...new Set([first, second])];
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 1, again = 0; const asked = [];
      await page.route((url) => ends(url.toString(), "integrations"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.revision = body.revision + (again ? "-again" + again : "");
        body.events.deliveries_more = cut;
        body.events.deliveries_next = Object.fromEntries(cut.map((s) => [s, "50"]));
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "integrations/deliveries"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.toString());
        const sub = q.get("subscription");
        if (sub === first && q.get("after") === "50" && fail-- > 0) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        const answer = (list, next) => route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", subscription: sub, order: "recorded", rows_can_move: true, deliveries: list, next, total: 80 } });
        return q.get("after") === "50" ? answer(span(sub, 1, 20), "70") : answer(span(sub, 21, 30), "");
      });
      await open(lang);
      const buttons = page.locator(".p02-deliveries-more-button");
      const names = (await buttons.allInnerTexts()).map((x) => x.trim());
      check(`${at} each cut subscription has its own named button and the note that deliveries can change`, names.join("|") === cut.map((s) => copy[lang].more(s)).join("|") && (await page.locator(".p02-deliveries-move").innerText()).trim() === copy[lang].moves && (await rows().count()) === real.length, names.join("|"));
      const more = buttons.first();
      check(`${at} the button is at least 44 high`, (await more.evaluate((el) => el.getBoundingClientRect().height)) >= 44);
      await more.focus();
      await page.keyboard.press("Enter");
      await page.locator(".p02-deliveries-more [role=alert]").waitFor();
      const alert = await page.locator(".p02-deliveries-more").first().innerText();
      check(`${at} a failed read says so, keeps the deliveries and keeps focus on the button`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && alert.includes(first) && (await rows().count()) === real.length && await more.evaluate((el) => el === document.activeElement), alert);
      if (args.shots) { await page.locator(".p02-deliveries-foot").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p02-deliveries-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.keyboard.press("Enter");
      await page.waitForFunction((n) => document.querySelectorAll(".p02-deliveries tbody.p02-group").length === n, real.length + 20);
      check(`${at} the next deliveries of that subscription follow and its button stays for the rest`, asked[0] === new URLSearchParams({ subscription: first, limit: "50", after: "50" }).toString() && (await buttons.first().innerText()).trim() === copy[lang].more(first));
      await buttons.first().click();
      await page.waitForFunction((n) => document.querySelectorAll(".p02-deliveries tbody.p02-group").length === n, real.length + 30);
      check(`${at} at the end that subscription's button is gone and the other's remains`, asked[asked.length - 1] === new URLSearchParams({ subscription: first, limit: "50", after: "70" }).toString() && (await buttons.count()) === cut.length - 1 && (cut.length > 1 || (await page.locator(".p02-deliveries-move").innerText()).trim() === copy[lang].moves));
      const ids = await page.locator(".p02-deliveries tbody.p02-group > tr[id]").evaluateAll((els) => els.map((el) => el.id));
      check(`${at} no delivery is listed twice`, new Set(ids).size === ids.length && ids.length === real.length + 30, String(ids.length));
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (width === 1440) {
        if (args.shots) { await page.locator(".p02-deliveries-foot").scrollIntoViewIfNeeded().catch(() => undefined); await page.screenshot({ path: `${args.shots}/p02-deliveries-loaded-${lang}-${width}.png`, fullPage: false }); }
        again += 1;
        await page.locator(".p02-reload").first().click();
        await page.waitForFunction((n) => document.querySelectorAll(".p02-deliveries tbody.p02-group").length === n, real.length);
        check(`${at} after the screen is read again the list starts from the main read and says so`, (await page.locator(".p02-deliveries-reset").innerText()).trim() === copy[lang].reset && (await buttons.count()) === cut.length);
        await open(lang, `&subscription=${encodeURIComponent(second)}`);
        check(`${at} with a subscription selected only its button is offered`, (await buttons.count()) === 1 && (await buttons.first().innerText()).trim() === copy[lang].more(second));
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
