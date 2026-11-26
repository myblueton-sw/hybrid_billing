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
const at = (url, path) => new URL(url).pathname.endsWith("/api/v1/" + path);
const copy = {
  en: { cut: "More deliveries exist than are listed here.", order: "Every dead letter and held redelivery is listed; those not shown are queued redeliveries or cancelled deliveries.", maybe: "The service did not say whether more deliveries exist than are listed here." },
  ko: { cut: "여기에 표시된 것보다 발송 건이 더 있습니다.", order: "실패 보관 건과 보류된 재발송 건은 모두 표시되며, 표시되지 않은 건은 대기열에 등록된 재발송 건이거나 취소된 건입니다.", maybe: "서비스가 여기에 표시된 것보다 발송 건이 더 있는지 알려 주지 않았습니다." },
  ar: { cut: "توجد عمليات تسليم أكثر مما هو معروض هنا.", order: "تُعرض كل الرسائل غير المُسلّمة وعمليات إعادة التسليم المحتجَزة؛ وما لم يُعرض هو عمليات إعادة تسليم في قائمة الانتظار أو عمليات تسليم ملغاة.", maybe: "لم تذكر الخدمة ما إذا كانت هناك عمليات تسليم أكثر مما هو معروض هنا." },
};
const widths = [1440, 390];

async function events(lang) {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/integrations?lang=${lang}&view=events`);
  await page.locator(".p02-deliveries").waitFor();
  await page.waitForLoadState("networkidle");
}
async function contained(name) {
  const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
  check(`${name} page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
}
const stageEvents = (change) => page.route((url) => at(url.toString(), "integrations"), async (route) => {
  const response = await route.fetch();
  const body = await response.json();
  if (body.events && typeof body.events === "object") body.events = change(body.events);
  await route.fulfill({ response, json: body });
});
const line = () => page.locator(".p02-deliveries [data-deliveries-cut]");

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
  const seen = async (r) => { if (at(r.url(), "integrations") && r.status() === 200 && !served) served = await r.json().catch(() => null); };
  page.on("response", seen);
  await events("en");
  page.off("response", seen);
  const ev = served?.events ?? {};
  notes.push(`served events: deliveries ${Array.isArray(ev.deliveries) ? ev.deliveries.length : String(ev.deliveries)}, deliveries_more ${JSON.stringify(ev.deliveries_more)}`);
  check("the service states that no delivery list was cut", Array.isArray(ev.deliveries_more) && ev.deliveries_more.length === 0);
  check("with nothing cut the screen shows no line about more deliveries", (await line().count()) === 0);

  for (const lang of ["en", "ko", "ar"]) {
    for (const width of widths) {
      await page.setViewportSize({ width, height: 1000 });
      await stageEvents((e) => ({ ...e, deliveries_more: ["qa-any-subscription"] }));
      await events(lang);
      let text = (await line().count()) === 1 ? await line().innerText() : "";
      check(`${lang} ${width} a cut delivery list is stated`, text.includes(copy[lang].cut) && text.includes(copy[lang].order), text);
      const cutText = text;
      await contained(`${lang} ${width} events with a cut list`);
      if (args.shots) await page.screenshot({ path: `${args.shots}/p02-cut-${lang}-${width}.png`, fullPage: true });
      await page.unrouteAll({ behavior: "wait" });
      await stageEvents((e) => { const { deliveries_more, ...rest } = e; return rest; });
      await events(lang);
      text = (await line().count()) === 1 ? await line().innerText() : "";
      check(`${lang} ${width} an answer that does not say is stated as not known`, text !== cutText && text.includes(copy[lang].maybe), text);
      await page.unrouteAll({ behavior: "wait" });
    }
  }

  const p05 = {
    en: { shown: /Waiting for approval \(\d+ shown\)[\s\S]*Closed \(\d+ shown\)/, plain: /Waiting for approval \(\d+\)[\s\S]*Closed \(\d+\)/, older: "Older proposals waiting for approval may not be listed.", limited: "This comparison covers only the requirements listed on this screen, which may not be all of them." },
    ko: { shown: /승인 대기\(표시된 \d+건\)[\s\S]*종료됨\(표시된 \d+건\)/, plain: /승인 대기\(\d+\)[\s\S]*종료됨\(\d+\)/, older: "이전 승인 대기 제안은 목록에 없을 수 있습니다.", limited: "이 비교는 이 화면에 표시된 요구사항만 대상으로 하며, 전체가 아닐 수 있습니다." },
    ar: { shown: /بانتظار الاعتماد \(المعروض: \d+\)[\s\S]*مغلقة \(المعروض: \d+\)/, plain: /بانتظار الاعتماد \(\d+\)[\s\S]*مغلقة \(\d+\)/, older: "قد لا تُدرج مقترحات أقدم بانتظار الاعتماد.", limited: "تشمل هذه المقارنة المتطلبات المدرجة في هذه الشاشة فقط، وقد لا تكون جميعها." },
  };
  const evidence = async (lang, view) => {
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/release-evidence?lang=${lang}&view=${view}`);
    await page.locator(view === "decisions" ? ".p05-live #p05-waiting" : ".p05-live #p05-compare-to").waitFor();
    await page.waitForLoadState("networkidle");
    return page.locator(".p05-live").innerText();
  };
  const cutEvery = (v) => (Array.isArray(v) ? v.map(cutEvery) : v && typeof v === "object" ? Object.fromEntries(Object.entries(v).map(([k, x]) => [k, Array.isArray(v.rows) && typeof v.more === "boolean" && k === "more" ? true : Array.isArray(v.rows) && typeof v.more === "boolean" && k === "total" && typeof x === "number" ? v.rows.length + 25 : cutEvery(x)])) : v);
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of widths) {
      await page.setViewportSize({ width, height: 1000 });
      let text = await evidence(lang, "decisions");
      check(`${lang} ${width} P05 decisions with complete lists show plain counts and no line about older proposals`, p05[lang].plain.test(text) && !p05[lang].shown.test(text) && !text.includes(p05[lang].older), text.slice(0, 300));
      text = await evidence(lang, "requirements");
      check(`${lang} ${width} P05 comparison of a complete list has no note about a limited comparison`, !text.includes(p05[lang].limited));
      await page.route((url) => at(url.toString(), "release-evidence"), async (route) => {
        const response = await route.fetch();
        await route.fulfill({ response, json: cutEvery(await response.json()) });
      });
      text = await evidence(lang, "decisions");
      check(`${lang} ${width} P05 decisions with a cut list mark their counts as shown and say older proposals may be missing`, p05[lang].shown.test(text) && text.includes(p05[lang].older), text.slice(0, 300));
      await contained(`${lang} ${width} P05 decisions with a cut list`);
      if (args.shots) await page.screenshot({ path: `${args.shots}/p05-decisions-cut-${lang}-${width}.png`, fullPage: true });
      text = await evidence(lang, "requirements");
      check(`${lang} ${width} P05 comparison over a cut list says it covers only the listed requirements`, text.includes(p05[lang].limited));
      await contained(`${lang} ${width} P05 requirements with a cut list`);
      if (args.shots) await page.screenshot({ path: `${args.shots}/p05-requirements-cut-${lang}-${width}.png`, fullPage: true });
      await page.unrouteAll({ behavior: "wait" });
    }
  }

  const basis = {
    en: { pending: "Previewed because it is the newest pending plan proposal.", release: "Previewed because it is the newest release not installed." },
    ko: { pending: "승인 대기 중인 가장 최근 계획 제안이라서 미리 봅니다.", release: "설치되지 않은 가장 최근 릴리스라서 미리 봅니다." },
    ar: { pending: "تُعرض لأنها أحدث مقترح خطة بانتظار الاعتماد.", release: "تُعرض لأنها أحدث إصدار غير مثبّت." },
  };
  const plan = async (lang) => {
    await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/installation?lang=${lang}`);
    await page.locator(".p01-live #p01-plan-title").waitFor();
    await page.waitForLoadState("networkidle");
    return page.locator(".p01-plan").innerText();
  };
  const stagePlan = (change) => page.route((url) => at(url.toString(), "installation"), async (route) => {
    const response = await route.fetch();
    await route.fulfill({ response, json: change(await response.json()) });
  });
  let servedBasis = "";
  const basisSeen = async (r) => { if (at(r.url(), "installation") && r.status() === 200) servedBasis = String((await r.json().catch(() => ({}))).preview_basis ?? "absent"); };
  page.on("response", basisSeen);
  await page.setViewportSize({ width: 1440, height: 1000 });
  await plan("en");
  page.off("response", basisSeen);
  notes.push(`served preview_basis: ${servedBasis}`);
  check("the service states the basis of the preview", servedBasis === "pending_plan" || servedBasis === "newest_release");
  for (const lang of ["en", "ko", "ar"]) {
    let text = await plan(lang);
    const own = servedBasis === "pending_plan" ? "pending" : "release", other = own === "pending" ? "release" : "pending";
    check(`${lang} P01 states the basis the service gave`, text.includes(basis[lang][own]) && !text.includes(basis[lang][other]), text.slice(0, 300));
    await stagePlan((b) => ({ ...b, preview_basis: own === "pending" ? "newest_release" : "pending_plan" }));
    text = await plan(lang);
    check(`${lang} P01 follows the served basis and not what it could infer from the listed proposals`, text.includes(basis[lang][other]) && !text.includes(basis[lang][own]), text.slice(0, 300));
    await page.unrouteAll({ behavior: "wait" });
    await stagePlan((b) => { const { preview_basis, ...rest } = b; return rest; });
    text = await plan(lang);
    check(`${lang} P01 states no basis when the service gives none`, !text.includes(basis[lang].pending) && !text.includes(basis[lang].release), text.slice(0, 300));
    await contained(`${lang} P01 plan`);
    await page.unrouteAll({ behavior: "wait" });
  }

  {
    const unknown = { en: "This requirement is not listed in this read, so it is unknown whether retiring it removes a blocking check from the gate.", ko: "이 요구사항이 이 조회에 없어, 폐지하면 게이트에서 차단 점검이 빠지는지 알 수 없습니다.", ar: "هذا المتطلب غير مدرج في هذه القراءة، لذا لا يُعرف ما إذا كان إنهاء العمل به يزيل فحصًا حاجبًا من البوابة." };
    const label = "sample/sample-legacy-report";
    const second = await browser.newContext({ viewport: { width: 1440, height: 1000 } });
    const other = await second.newPage();
    other.on("pageerror", (e) => errors.push(e.message));
    const secret = randomBytes(20).toString("base64url");
    const link = new URL(lines.find((l) => l.startsWith("admin2@example.invalid ")).split(" ")[1]);
    await other.goto(`${base}/activate?lang=en${link.hash}`);
    await other.locator("#link-new").fill(secret); await other.locator("#link-confirm").fill(secret);
    await other.locator("form button[type=submit]").click(); await other.locator("a.service-button-link").waitFor();
    await other.goto(`${base}/?service=api&lang=en`);
    await other.locator("#signin-workspace").fill("sample-workspace"); await other.locator("#signin-email").fill("admin2@example.invalid");
    await other.locator("#signin-password").fill(secret); await other.locator("form button[type=submit]").click();
    await other.locator(".service-session").waitFor();
    const without = (v) => (Array.isArray(v) ? v.filter((x) => !(x && typeof x === "object" && x.label === label && !("summary" in x))).map(without) : v && typeof v === "object" ? Object.fromEntries(Object.entries(v).map(([k, x]) => [k, without(x)])) : v);
    const confirmText = async (lang, staged) => {
      await other.unrouteAll({ behavior: "wait" });
      if (staged) await other.route((url) => at(url.toString(), "release-evidence"), async (route) => { const response = await route.fetch(); const body = await response.json(); await route.fulfill({ response, json: { ...body, requirements: without(body.requirements), matrix: without(body.matrix) } }); });
      await other.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/release-evidence?lang=${lang}&view=decisions`);
      await other.locator(".p05-live #p05-waiting").waitFor();
      await other.waitForLoadState("networkidle");
      const row = other.locator(".p05-proposals > li").filter({ hasText: "sample-legacy-report" }).first();
      await row.locator('[data-opener$="release_evidence.approve"]').click();
      await row.locator(".p05-form textarea").fill("synthetic rendered check");
      await row.locator(".p05-form button[type=submit]").click();
      await row.locator(".p05-confirm").waitFor();
      return row.locator(".p05-confirm").innerText();
    };
    for (const lang of ["en", "ko", "ar"]) {
      let text = await confirmText(lang, false);
      check(`${lang} P05 approving a retirement of a listed requirement does not call its impact unknown`, !text.includes(unknown[lang]), text.slice(0, 300));
      text = await confirmText(lang, true);
      check(`${lang} P05 approving a retirement of a requirement that is not in the read says the impact is unknown`, text.includes(unknown[lang]), text.slice(0, 500));
      if (args.shots && lang === "en") await other.screenshot({ path: `${args.shots}/p05-retire-unknown-en-1440.png`, fullPage: true });
    }
    await other.unrouteAll({ behavior: "wait" });
    await second.close();
  }

  await page.setViewportSize({ width: 1440, height: 1000 });
  await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#/installation?lang=en`);
  await page.locator(".p01-live").waitFor();
  await page.waitForLoadState("networkidle");
  const opener = page.locator(".p01-propose .live-buttons button").first();
  notes.push(`P01 propose offered: ${await opener.count()}; blocker text: ${((await page.locator(".p01-propose .p01-blocker").count()) > 0 ? await page.locator(".p01-propose .p01-blocker").first().innerText() : "none").slice(0, 200)}`);
  if ((await opener.count()) > 0) {
    const propose = async (reason) => {
      await page.locator(".p01-propose .live-buttons button").first().click();
      await page.locator(".p01-propose .p01-form textarea").fill(reason);
      await page.locator(".p01-propose .p01-form button[type=submit]").click();
      const answered = page.waitForResponse((r) => r.url().includes("/api/v1/commands/installation.propose_plan"));
      await page.locator(".p01-propose .p01-confirm .live-primary").click();
      const response = await answered;
      const body = await response.json().catch(() => ({}));
      await page.waitForLoadState("networkidle");
      return { status: response.status(), fields: body.fields ?? {}, code: body.code ?? "" };
    };
    const first = await propose("synthetic rendered check, first");
    notes.push(`first proposal: ${first.status} ${JSON.stringify(first.fields)} ${first.code}`);
    check("the first proposal is new", first.status === 200 && !!first.fields.proposal && first.fields.existing === undefined);
    const firstText = await page.locator(".p01-live").innerText();
    check("the screen does not call the first proposal a duplicate", !firstText.includes("already proposed"));
    const number = String(first.fields.proposal);
    await page.route((url) => at(url.toString(), "installation"), async (route) => {
      const response = await route.fetch();
      const body = await response.json();
      if (Array.isArray(body.proposals)) body.proposals = body.proposals.filter((p) => String(p.number) !== number);
      await route.fulfill({ response, json: body });
    });
    await page.goto(`${base}/?service=api&lang=en&qa=${++navigation}#/installation?lang=en`);
    await page.locator(".p01-live").waitFor();
    await page.waitForLoadState("networkidle");
    notes.push(`with the proposal left out of the read, propose offered: ${await page.locator(".p01-propose .live-buttons button").count()}`);
    if ((await page.locator(".p01-propose .live-buttons button").count()) > 0) {
      const second = await propose("synthetic rendered check, second");
      notes.push(`second proposal: ${second.status} ${JSON.stringify(second.fields)} ${second.code}`);
      check("the service answers the repeated proposal with the existing one and says so", second.status === 200 && String(second.fields.proposal) === number && second.fields.existing === "true");
      const text = await page.locator(".p01-live").innerText();
      check("the screen says the plan was already proposed, although the proposal is not among those listed", text.includes(`This plan was already proposed as #${number}. No new proposal was created, and that proposal keeps its original reason.`), text.slice(0, 600));
      if (args.shots) await page.screenshot({ path: `${args.shots}/p01-existing-en-1440.png`, fullPage: true });
    }
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
