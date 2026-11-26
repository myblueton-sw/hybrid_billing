import { inspectView } from "./view-measure.mjs";
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
  en: { rotate: "Rotate", title: "Rotate credential", review: "Review rotation", check: "Check the rotation", show: "Show secret", ending: "Rotated · ends", replaces: "Replaces", replacedBy: "Replaced by", cancel: "Cancel rotation" },
  ko: { rotate: "교체", title: "자격 증명 교체", review: "교체 내용 검토", check: "교체 내용 확인", show: "비밀값 표시", ending: "교체됨", replaces: "대체", replacedBy: "대체됨", cancel: "교체 취소" },
  ar: { rotate: "تدوير", title: "تدوير بيانات الاعتماد", review: "مراجعة التدوير", check: "تحقّق من التدوير", show: "", ending: "دُوِّرت", replaces: "تحلّ محل", replacedBy: "حلّت محلها", cancel: "إلغاء التدوير" },
};
const rid = () => randomBytes(16).toString("hex");
let csrf = "";
const api = async (path) => { const r = await page.context().request.get(`${base}/api/v1${path}`, { headers: { "X-CSRF-Token": csrf, "X-Request-Id": rid() } }); return { status: r.status(), body: await r.json().catch(() => null) }; };
const command = async (id, target, fields, why = "Rendered check of credential rotation") => {
  const revision = (await api("/identity/services?limit=1")).body.revision, request = rid();
  const r = await page.context().request.post(`${base}/api/v1/commands/${id}`, { headers: { "X-CSRF-Token": csrf, "X-Request-Id": request, Origin: new URL(base).origin }, data: { request_id: request, target_id: target, idempotency_key: rid(), expected_revision: revision, rationale: why, body: fields } });
  return { status: r.status(), body: await r.json().catch(() => null) };
};
const go = async (lang, extra) => {
  await page.goto("about:blank");
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/service-accounts?${new URLSearchParams({ lang, view: "credentials", ...extra })}`);
  await page.locator("#svc-cred-title").waitFor();
  await page.waitForLoadState("networkidle");
};
const detailOf = (id) => page.locator("section.live-step").filter({ has: page.getByRole("heading", { name: id, exact: true }) }).first();

try {
  if (args.shots) await mkdir(args.shots, { recursive: true });
  const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
  const activation = new URL(lines.find((l) => l.startsWith("admin@example.invalid ")).split(" ")[1]);
  await page.goto(`${base}/activate?lang=en${activation.hash}`);
  await page.locator("#link-new").fill(password); await page.locator("#link-confirm").fill(password);
  await page.locator("form button[type=submit]").click(); await page.locator("a.service-button-link").waitFor();
  const seen = (r) => { const t = r.headers()["x-csrf-token"]; if (t) csrf = t; };
  page.on("request", seen);
  await page.goto(`${base}/?service=api&lang=en`);
  await page.locator("#signin-workspace").fill("sample-workspace"); await page.locator("#signin-email").fill("admin@example.invalid");
  await page.locator("#signin-password").fill(password); await page.locator("form button[type=submit]").click();
  await page.locator(".service-session").waitFor();
  page.off("request", seen);
  const session = await (await page.context().request.get(`${base}/api/v1/session`, { headers: { "X-HB-Client": "console" } })).json();
  csrf = session.csrf_token ?? csrf;
  await go("en", {});
  check("a request token was read from the session", !!csrf);

  const first = await api("/identity/services?limit=1");
  const owner = session.user ?? "";
  const name = "rot-" + randomBytes(4).toString("hex");
  const made = await command("service.create", "services", { name, owner, purpose: "Synthetic rotation check", organization: "sample-le-seller", catalog_version: first.body.catalog_version, scopes: JSON.stringify([{ audience: "test-service-api", scope: "read" }]) });
  const service = "svc-" + name;
  const issued = await command("service.issue_credential", service, { days: "30" });
  const old = issued.body?.fields?.credential ?? "";
  check("the fixture account and its first credential were made through the real commands", made.status === 200 && issued.status === 200 && /^cred-/.test(old), JSON.stringify({ made: made.status, body: made.body?.code ?? made.body?.status, issued: issued.status, owner }).slice(0, 300));
  const before = await api(`/identity/service-credentials/${old}`);
  check("the real credential detail says the credential can be rotated and carries the overlap limits", before.body?.eligibility?.can_rotate === true && before.body?.eligibility?.rotate_step_up_state === "ready" && before.body?.limits?.credential_overlap_minutes > 0, JSON.stringify(before.body?.eligibility));

  const firstCredential = old;
  for (const [lang, width] of [["ko", 390], ["ar", 390], ["en", 1440]]) {
    const at = `${lang} ${width}`;
    const old = lang === "ko" ? firstCredential : (await command("service.issue_credential", service, { days: "30" })).body?.fields?.credential ?? "";
    await page.setViewportSize({ width, height: 1000 });
    await go(lang, { svc_c_service: service, svc_c_selected: old });
    const detail = detailOf(old);
    await detail.waitFor();
    const button = detail.locator("button.live-rotate");
    check(`${at}: the credential detail offers Rotate, enabled`, (await button.innerText()).trim() === copy[lang].rotate && await button.isEnabled());
    await button.focus(); await page.keyboard.press("Enter");
    const form = page.locator("form").filter({ has: page.locator("#svc-rotate-title") });
    await form.waitFor();
    await page.waitForTimeout(150);
    const focus = await page.evaluate(() => document.activeElement?.id ?? "");
    check(`${at}: the rotate form opens with focus on its title and the default overlap`, focus === "svc-rotate-title" && (await form.locator("#svc-rotate-title").innerText()).trim() === copy[lang].title && (await form.locator("#svc-overlap").inputValue()) === String(before.body.limits.credential_overlap_minutes), focus);
    await form.locator("button[type=submit]").click();
    await page.locator("#svc-form-errors").waitFor();
    check(`${at}: an empty reason is refused with the error summary focused and nothing sent`, await page.evaluate(() => document.activeElement?.id === "svc-form-errors"));
    await form.locator("#svc-overlap").fill("5");
    await form.locator("#live-reason").fill("Rendered check of credential rotation");
    await form.locator("button[type=submit]").click();
    await page.waitForTimeout(150);
    check(`${at}: an overlap below the pickup time is refused at the field`, (await form.locator("#svc-overlap").getAttribute("aria-invalid")) === "true" && (await form.locator("#svc-overlap-error").count()) === 1);
    await form.locator("#svc-overlap").fill("60");
    await form.locator("#svc-days").fill("5");
    const sizes = await page.evaluate(() => ["svc-days", "svc-overlap"].map((id) => Math.round(document.getElementById(id).getBoundingClientRect().width)));
    check(`${at}: the two number fields are no wider than twelve rem`, sizes.every((w) => w <= 16 * 12 + 1), sizes.join());
    const shown = await inspectView(page);
    check(`${at}: the rotate form does not overflow, cut off or lose contrast`, shown.overflow <= 1 && shown.clipped.length === 0 && shown.cut.length === 0 && shown.low.length === 0, JSON.stringify({ o: shown.overflow, c: shown.clipped.slice(0, 2), cut: shown.cut.slice(0, 2), low: shown.low.slice(0, 2) }));
    if (args.shots) { await form.scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/rotate-form-${lang}-${width}.png` }); }
    const posts = [];
    const watch = (r) => { if (r.method() === "POST" && r.url().includes("/commands/")) posts.push(new URL(r.url()).pathname.split("/").pop()); };
    page.on("request", watch);
    await form.locator("button[type=submit]").click();
    await page.locator("#svc-review-title").waitFor();
    await page.waitForTimeout(150);
    check(`${at}: the review step takes focus, names the credential that will end, and sends nothing yet`, await page.evaluate(() => document.activeElement?.id === "svc-review-title") && (await page.locator("#svc-review-title").innerText()).trim() === copy[lang].check && (await page.locator(".live-review").innerText()).includes(old) && posts.length === 0 && (lang !== "ar" || ((await page.locator(".live-review").innerText()).includes("5 أيام") && !(await page.locator(".live-review").innerText()).includes("5 يوم"))) && (lang !== "en" || ((await page.locator(".live-review").innerText()).includes("5 days") && (await page.locator(".live-review").innerText()).includes("60 minutes"))), posts.join() + (await page.locator(".live-review").innerText()).slice(0, 200));
    const review = await inspectView(page);
    check(`${at}: the review does not overflow, cut off or lose contrast`, review.overflow <= 1 && review.clipped.length === 0 && review.cut.length === 0 && review.low.length === 0, JSON.stringify({ o: review.overflow, c: review.clipped.slice(0, 2), cut: review.cut.slice(0, 2), low: review.low.slice(0, 2) }));
    if (args.shots) { await page.locator(".live-review").scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/rotate-review-${lang}-${width}.png` }); }
    const answered = page.waitForResponse((r) => r.url().includes("/commands/service.rotate_credential"));
    await form.locator("button[type=submit]").click();
    const answer = await answered, result = await answer.json().catch(() => null);
    page.off("request", watch);
    const next = result?.fields?.credential ?? "";
    check(`${at}: confirming sends one rotate command for the old credential with the named overlap, and it completes`, answer.status() === 200 && result?.business_state === "credential_rotated" && posts.join() === "service.rotate_credential" && JSON.parse(answer.request().postData()).target_id === old && JSON.parse(answer.request().postData()).body.overlap_minutes === "60" && /^cred-/.test(next), JSON.stringify({ s: answer.status(), posts, state: result?.business_state }));
    const done = page.locator(".live-rotate-done");
    await done.waitFor({ timeout: 8000 }).catch(() => {});
    await page.waitForTimeout(300);
    const doneText = (await done.count()) ? await done.innerText() : "";
    check(`${at}: after confirming, the form is replaced by the result: the new credential, the old one's end, and the one-time secret`, doneText.includes(next) && doneText.includes(old) && (await done.locator("button.live-primary").count()) === 1 && (await page.locator("#svc-days").count()) === 0 && await detailOf(old).locator("button.live-rotate").isDisabled(), doneText.slice(0, 200));
    const marked = await done.locator("p[role=status] bdi.live-mono").allInnerTexts();
    const top = await done.locator("h2").evaluate((el) => getComputedStyle(el).marginBlockStart);
    check(`${at}: the result sentence shows its three identifiers in mono and its title has room above`, marked.includes(next) && marked.includes(old) && marked.includes(service) && top === "24px", marked.join() + " " + top);
    if (args.shots) { await done.scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/rotate-done-${lang}-${width}.png` }); }
    const after = await api(`/identity/service-credentials/${old}`), fresh = await api(`/identity/service-credentials/${next}`);
    check(`${at}: the real reads show the old credential ending and no longer rotatable, and the new one replacing it`, after.body?.row?.rotation === "ending" && after.body?.row?.replaced_by === next && after.body?.eligibility?.can_rotate === false && fresh.body?.row?.replaces === old && fresh.body?.row?.rotation === "replacing" && fresh.body?.row?.secret_pickup === "open", JSON.stringify({ old: after.body?.row?.rotation, can: after.body?.eligibility?.can_rotate, fresh: fresh.body?.row?.rotation, pickup: fresh.body?.row?.secret_pickup }));
    await go(lang, { svc_c_service: service, svc_c_selected: old });
    const ended = detailOf(old); await ended.waitFor();
    const text = await ended.innerText();
    check(`${at}: the old credential's detail says it was rotated and by which credential, with Rotate disabled`, text.includes(copy[lang].ending) && text.includes(copy[lang].replacedBy) && text.includes(next) && await ended.locator("button.live-rotate").isDisabled() && !/\d{4}-\d{2}-\d{2}T/.test(await ended.locator(".live-rotation").innerText()), (await ended.locator(".live-rotation").innerText()).slice(0, 120));
    if (args.shots) { await ended.scrollIntoViewIfNeeded(); await page.screenshot({ path: `${args.shots}/rotate-old-${lang}-${width}.png` }); }
    const period = await ended.innerText();
    const whole = await page.locator("main").innerText();
    check(`${at}: neither the detail nor the list shows a raw timestamp`, !/\d{4}-\d{2}-\d{2}T\d{2}:/.test(period + whole), (period + whole).match(/\d{4}-\d{2}-\d{2}T[^\s]*/)?.[0] ?? "");
    const link = ended.locator("button.live-id-link").first();
    const look = await link.evaluate((el) => { const c = getComputedStyle(el); return { border: c.borderTopWidth, bg: c.backgroundColor, underline: c.textDecorationLine, height: Math.round(el.getBoundingClientRect().height) }; });
    check(`${at}: the identifier link looks like a link and is 44px tall`, look.border === "0px" && /rgba\(0, 0, 0, 0\)|transparent/.test(look.bg) && look.underline.includes("underline") && look.height >= 44, JSON.stringify(look));
    const gap = await page.evaluate(() => { const steps = [...document.querySelectorAll("section.live-step")]; const last = steps[steps.length - 1]?.getBoundingClientRect(); const groups = [...document.querySelectorAll("main fieldset, main [role=group], main form")].map((e) => e.getBoundingClientRect()).filter((r) => last && r.top >= last.bottom - 1); return last && groups.length ? Math.round(Math.min(...groups.map((r) => r.top)) - last.bottom) : -1; });
    notes.push(`${at}: space between the credential detail and what follows it: ${gap}px`);
    await link.click();
    await detailOf(next).waitFor({ timeout: 5000 }).catch(() => {});
    check(`${at}: the identifier in "replaced by" selects that credential`, (await detailOf(next).count()) === 1 && new URLSearchParams(new URL(page.url()).hash.split("?")[1]).get("svc_c_selected") === next);
    await go(lang, { svc_c_service: service, svc_c_selected: next });
    const newer = detailOf(next); await newer.waitFor();
    check(`${at}: the new credential's detail says what it replaces`, (await newer.innerText()).includes(copy[lang].replaces) && (await newer.innerText()).includes(old));
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
