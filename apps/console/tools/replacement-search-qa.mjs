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
  en: { first: (s, m) => `Showing the first ${s} of ${m} possible replacements. Search to find others.`, matches: (n) => (n === 1 ? "1 possible replacement matches." : `${n} possible replacements match.`), more: (s, m) => `Showing ${s} of ${m} that match. Type a longer search to narrow the list.`, none: (q) => `No possible replacement matches “${q}”.`, failed: "The search could not be completed. The list is unchanged.", again: "Try again", label: "Search possible replacements", search: "Search", old: "Only the first 20 possible replacements are listed.", unknown: "Unknown", detail: "Details of this replacement could not be read." },
  ko: { first: (s, m) => `대체 요구사항 후보 ${m}개 중 처음 ${s}개를 표시합니다. 다른 후보는 검색하세요.`, matches: (n) => `대체 요구사항 후보 ${n}개가 일치합니다.`, more: (s, m) => `일치하는 ${m}개 중 ${s}개를 표시합니다. 검색어를 더 길게 입력해 범위를 좁히세요.`, none: (q) => `“${q}”와 일치하는 대체 요구사항 후보가 없습니다.`, failed: "검색을 완료하지 못했습니다. 목록은 그대로입니다.", again: "다시 시도", label: "대체 요구사항 후보 검색", search: "검색", old: "가능한 대체 요구사항 중 처음 20개만 표시합니다.", unknown: "알 수 없음", detail: "이 대체 요구사항의 세부 정보를 읽지 못했습니다." },
  ar: { first: (s, m) => `يُعرض أول ${s} من أصل ${m} من البدائل الممكنة. ابحث للعثور على غيرها.`, matches: (n) => `عدد البدائل الممكنة المطابقة: ${n}.`, more: (s, m) => `يُعرض ${s} من أصل ${m} من المطابقة. اكتب بحثًا أطول لتضييق القائمة.`, none: (q) => `لا يطابق «${q}» أي بديل ممكن.`, failed: "تعذّر إكمال البحث. لم تتغيّر القائمة.", again: "إعادة المحاولة", label: "البحث في البدائل الممكنة", search: "بحث", old: "تُدرج أول 20 بديلًا ممكنًا فقط.", unknown: "غير معروف", detail: "تعذّرت قراءة تفاصيل هذا البديل." },
};
const pad = (n) => String(n).padStart(3, "0");
const openView = async (lang) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/release-evidence?lang=${lang}&view=requirements`);
  await page.locator(".p05-opener").first().waitFor();
  await page.waitForLoadState("networkidle");
  await page.waitForFunction(() => !document.querySelector(".p05-live[aria-busy=true]"));
};
const status = async () => (await page.locator(".p05-replacement-status").innerText()).trim();
const options = () => page.locator(".p05-form .p05-select-field select option").evaluateAll((els) => els.map((el) => el.value));
const settled = () => page.waitForFunction(() => !/…$/.test(document.querySelector(".p05-replacement-status")?.textContent ?? ""));

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
  await openView("en");
  page.off("response", seen);
  const reqs = served?.requirements?.requirements?.rows ?? [];
  const subject = reqs.find((r) => r.can_retire && r.replacements.rows.length > 0) ?? reqs.find((r) => r.can_retire);
  notes.push(`served: ${reqs.length} requirements; using ${subject?.target} (${subject?.label}) with ${subject?.replacements?.rows?.length} candidates, total ${subject?.replacements?.total}, more ${subject?.replacements?.more}, replacement required ${subject?.replacement_required}`);
  const further = await page.evaluate(async ([csrf, target]) => {
    const head = { "X-CSRF-Token": csrf, "X-Request-Id": "qa-replacements" };
    const get = async (u) => { const r = await fetch(u, { headers: head, credentials: "include" }); return { status: r.status, body: await r.json().catch(() => null) }; };
    return { a: await get(`/api/v1/release-evidence/replacements?requirement=${target}`), b: await get(`/api/v1/release-evidence/replacements?requirement=trace-requirement-99999`) };
  }, [token, subject.target]);
  check("the real service answers the replacements read with the main read's candidates and refuses an unknown requirement", further.a.status === 200 && further.a.body.matching === subject.replacements.total && further.a.body.replacements.map((x) => x.target).join() === subject.replacements.rows.map((x) => x.target).join() && further.b.status === 422, JSON.stringify(further).slice(0, 300));
  const row = (lang) => page.locator(`#p05-requirement-${subject.target.replace("trace-requirement-", "")}`);
  const openForm = async (lang) => { await openView(lang); await page.locator(".p05-opener").nth(reqs.filter((r) => r.can_retire).findIndex((r) => r.target === subject.target)).click(); await page.locator(".p05-form .p05-select-field select").waitFor(); };
  await openForm("en");
  check("with twenty or fewer candidates the field is as before: a select and no search", (await page.locator(".p05-replacement-search").count()) === 0 && (await page.locator(".p05-replacement-status").count()) === 0 && (await options()).filter((v) => v.startsWith("trace-requirement-")).length === subject.replacements.rows.length);

  const cand = (n) => ({ target: `trace-requirement-${5000 + n}`, label: `zz/candidate-${pad(n)}` });
  const all = Array.from({ length: 45 }, (_, i) => cand(i + 1));
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 390]) {
      const at = `${lang} ${width}`;
      await page.setViewportSize({ width, height: 1000 });
      let fail = 0, slow = 0, noDetail = lang === "ko"; const asked = [], details = [];
      await page.route((url) => ends(url.toString(), "release-evidence/requirements"), async (route) => {
        const wanted = new URL(route.request().url()).searchParams.get("target") ?? "";
        const hit = all.find((c) => c.target === wanted);
        if (!hit) return route.continue();
        details.push(wanted);
        if (noDetail) return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } });
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", order: "recorded", requirements: [{ ...subject, target: hit.target, label: hit.label, replacements: { total: 0, more: false, rows: [] } }], next: "", matching: 1, total: 46 } });
      });
      await page.route((url) => ends(url.toString(), "release-evidence"), async (route) => {
        const response = await route.fetch();
        const body = await response.json();
        body.requirements.requirements.rows = body.requirements.requirements.rows.map((r) => (r.target === subject.target ? { ...r, replacements: { total: 45, more: true, rows: all.slice(0, 20) } } : r));
        await route.fulfill({ response, json: body });
      });
      await page.route((url) => ends(url.toString(), "release-evidence/replacements"), async (route) => {
        const q = new URL(route.request().url()).searchParams;
        asked.push(q.toString());
        if (fail > 0) { fail--; return route.fulfill({ status: 503, json: { code: "dependency_unavailable", retry: true, next_action: "retry", request_id: "qa" } }); }
        if (slow > 0) { slow--; await new Promise((r) => setTimeout(r, 900)); }
        const term = (q.get("q") ?? "").toLowerCase();
        const match = all.filter((c) => c.label.toLowerCase().includes(term));
        await route.fulfill({ status: 200, json: { revision: "r", server_time: "2026-11-16T09:00:00Z", requirement: q.get("requirement"), order: "recorded", q: q.get("q") ?? "", replacements: match.slice(0, 20), matching: match.length, more: match.length > 20 } });
      });
      await openForm(lang);
      const candidates = async () => (await options()).filter((v) => v.startsWith("trace-requirement-"));
      const placed = await page.evaluate(() => { const s = document.querySelector(".p05-form .p05-select-field select").getBoundingClientRect(), st = document.querySelector(".p05-replacement-status").getBoundingClientRect(), i = document.querySelector(".p05-replacement-search input").getBoundingClientRect(), b = document.querySelector(".p05-replacement-search-button").getBoundingClientRect(); return { order: s.bottom <= st.top + 1 && st.bottom <= i.top + 1, input: i.height, button: b.height }; });
      check(`${at} opened with more than twenty: the first twenty, the sentence, then the labelled search under the select`, (await candidates()).length === 20 && (await status()) === copy[lang].first(20, 45) && placed.order && placed.input >= 44 && placed.button >= 44 &&
        (await page.locator(".p05-replacement-search-label").innerText()).trim() === copy[lang].label && (await page.locator(".p05-replacement-search-button").innerText()).trim() === copy[lang].search && !(await page.locator(".p05-form").innerText()).includes(copy[lang].old), JSON.stringify(placed));
      if (args.shots) { await page.locator(".p05-form .p05-select-field").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-replacement-first-${lang}-${width}.png`, fullPage: false }); }
      const input = page.locator(".p05-replacement-search input");
      await input.fill("candidate-04");
      slow = 1;
      await input.press("Enter");
      await page.waitForFunction(() => /…$/.test(document.querySelector(".p05-replacement-status")?.textContent ?? ""));
      await settled();
      const found = await candidates();
      check(`${at} a search fills the select with the matches beyond the twentieth and says how many, without submitting the form`, found.length === 6 && found[0] === cand(40).target && (await status()) === copy[lang].matches(6) && asked[asked.length - 1] === new URLSearchParams({ requirement: subject.target, q: "candidate-04" }).toString() && await input.evaluate((el) => el === document.activeElement) && (await page.locator(".p05-confirm").count()) === 0, found.join());
      await page.locator(".p05-form .p05-select-field select").selectOption(cand(41).target);
      await input.fill("candidate");
      await input.press("Enter");
      await settled();
      check(`${at} more than twenty matches are cut with both numbers, and the candidate already chosen stays an option`, (await status()) === copy[lang].more(20, 45) && (await candidates()).includes(cand(41).target) && (await page.locator(".p05-form .p05-select-field select").inputValue()) === cand(41).target);
      await input.fill("nothing-like-this");
      await input.press("Enter");
      await settled();
      check(`${at} no match says so and keeps the chosen candidate`, (await status()) === copy[lang].none("nothing-like-this") && (await candidates()).join() === cand(41).target);
      if (args.shots) { await page.locator(".p05-form .p05-select-field").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-replacement-none-${lang}-${width}.png`, fullPage: false }); }
      fail = 1;
      await input.fill("candidate-00");
      await input.press("Enter");
      await page.locator(".p05-replacement-failed").waitFor();
      const alert = await page.locator(".p05-replacement-failed").innerText();
      check(`${at} a failed search says so and leaves the options as they were`, alert.includes(copy[lang].failed) && alert.includes(copy[lang].again) && (await candidates()).join() === cand(41).target, alert);
      if (args.shots) { await page.locator(".p05-replacement-failed").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-replacement-failed-${lang}-${width}.png`, fullPage: false }); }
      await page.locator(".p05-replacement-failed button").click();
      await page.waitForFunction(() => !document.querySelector(".p05-replacement-failed"));
      await settled();
      check(`${at} the retry searches the same term, shows the matches and puts focus in the input`, (await status()) === copy[lang].matches(9) && (await candidates()).length === 10 && await input.evaluate((el) => el === document.activeElement), await status());
      await input.fill("");
      await page.locator(".p05-replacement-search-button").click();
      check(`${at} an empty search restores the first twenty and its sentence`, (await status()) === copy[lang].first(20, 45) && (await candidates()).length === 21);
      const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
      check(`${at} the page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
      if (width === 1440) {
        for (const el of await page.locator(".p05-form .p05-text-field input, .p05-form textarea").all()) await el.fill("Rendered check of the replacement search");
        await page.locator(".p05-form button[type=submit]").click();
        await page.locator(".p05-confirm").waitFor({ timeout: 5000 }).catch(async () => { notes.push(`${at} form errors: ` + (await page.locator(".p05-form .service-field-error").allInnerTexts()).join(" | ")); });
        const text = (await page.locator(".p05-confirm").count()) ? await page.locator(".p05-confirm").innerText() : "";
        const fact = (await page.locator(".p05-confirm dd").filter({ hasText: cand(41).label }).first().innerText().catch(() => "")).trim();
        check(`${at} the confirmation names the candidate found by search and ${noDetail ? "says its details could not be read" : "shows its state"}, never Unknown`, text.includes(cand(41).label) && details.join() === cand(41).target && !fact.includes(copy[lang].unknown) && (noDetail ? fact.includes(copy[lang].detail) : fact.length > cand(41).label.length + 3 && !fact.includes(copy[lang].detail)), fact + " / " + details.join());
        if (args.shots && text) { await page.locator(".p05-confirm").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/p05-replacement-confirm-${lang}-${width}.png`, fullPage: false }); }
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
