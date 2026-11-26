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
const login = "admin@example.invalid";
const service = "svc-rotation";
const rotatePath = (url) => url.endsWith("/api/v1/commands/service.rotate_credential");
const copy = {
  en: { rotate: "Rotate", title: "Rotate credential", review: "Check the rotation", once: "The new secret is shown once and must be collected within 10 minutes of the rotation.", back: "This can't be undone: the current credential's original end is not restored.", about: "Estimated end", confirm: "Counted from the moment you confirm; the exact time is set then.", unknown: "This screen does not know where the credential is used and does not check work in progress.", exposed: "If the secret may have been exposed, revoke the credential instead of rotating it.", ending: "Rotated · ends ", replacedBy: "Replaced by", replaces: "Replaces", sooner: "Its end is not extended.", error: "Enter a whole number from 10 to 10080." },
  ko: { rotate: "교체", title: "자격 증명 교체", review: "교체 내용 확인", once: "새 비밀값은 한 번만 표시되며 교체 후 10분 안에 수령해야 합니다.", back: "되돌릴 수 없습니다. 기존 자격 증명의 원래 종료 시각은 복원되지 않습니다.", about: "예상 종료 시각", confirm: "확인하는 시점부터 계산하며, 정확한 시각은 그때 정해집니다.", unknown: "이 화면은 자격 증명이 어디에서 쓰이는지 알지 못하며 진행 중인 작업을 확인하지 않습니다.", exposed: "비밀값이 노출되었을 수 있다면 교체하지 말고 자격 증명을 폐기하세요.", ending: "교체됨 · ", replacedBy: "(으)로 대체됨", replaces: "을(를) 대체", sooner: "종료 시각은 늘어나지 않습니다.", error: "10–10080 사이 정수를 입력하세요." },
  ar: { rotate: "تدوير", title: "تدوير بيانات الاعتماد", review: "تحقّق من التدوير", once: "تُعرض القيمة السرية الجديدة مرة واحدة ويجب استلامها خلال 10 دقائق من التدوير.", back: "لا يمكن التراجع عن ذلك: لا يُستعاد موعد الانتهاء الأصلي لبيانات الاعتماد الحالية.", about: "موعد الانتهاء التقريبي", confirm: "تُحتسب المدة من لحظة التأكيد، ويُحدَّد الوقت الدقيق حينها.", unknown: "لا تعرف هذه الشاشة أين تُستخدم بيانات الاعتماد ولا تتحقق من الأعمال الجارية.", exposed: "إن اشتُبه في كشف القيمة السرية فأبطِل بيانات الاعتماد بدلًا من تدويرها.", ending: "دُوِّرت · تنتهي في ", replacedBy: "حلّت محلها", replaces: "تحلّ محل", sooner: "لا يُمدَّد موعد انتهائها.", error: "أدخل عددًا صحيحًا من 10 إلى 10080." },
};
const shot = async (name) => { if (args.shots) await page.screenshot({ path: `${args.shots}/${name}.png`, fullPage: true }); };
const open = async (lang, view) => {
  await page.goto(`${base}/?service=api&lang=${lang}&qa=${++navigation}#/service-accounts?lang=${lang}&view=${view}`);
  await page.locator(view === "credentials" ? "#svc-cred-title" : "#svc-list-title").waitFor();
  await page.waitForLoadState("networkidle");
};
const contained = async (name) => {
  const m = await page.evaluate(() => ({ page: document.documentElement.scrollWidth, view: document.documentElement.clientWidth }));
  check(`${name} page is not wider than the viewport`, m.page <= m.view + 1, JSON.stringify(m));
};
const panel = () => page.locator(".live-panel");
const rows = () => page.locator(".live-creds tbody tr");
const rowOf = (id) => page.locator(`.live-creds tbody tr:has([data-credential="${id}"])`);
async function reveal() {
  await page.locator("#svc-secret").waitFor();
  const secret = await page.locator("#svc-secret").inputValue();
  await page.locator("#svc-ack").check();
  await panel().locator(".live-buttons .live-primary").click();
  await page.locator("#svc-secret").waitFor({ state: "detached" });
  return secret;
}
async function issue(days) {
  await page.locator("#svc-days").fill(days);
  await page.locator("#live-reason").fill("synthetic rendered check");
  await panel().locator("form button[type=submit]").click();
  return reveal();
}
const live = async () => rows().evaluateAll((list) => list.map((tr) => tr.querySelector("[data-credential]")?.getAttribute("data-credential")).filter(Boolean));

try {
  if (args.shots) await mkdir(args.shots, { recursive: true });
  const lines = (await readFile(args.activation, "utf8")).trim().split("\n");
  const link = new URL(lines.find((l) => l.startsWith(login + " ")).split(" ")[1]);
  await page.goto(`${base}/activate?lang=en${link.hash}`);
  await page.locator("#link-new").fill(password); await page.locator("#link-confirm").fill(password);
  await page.locator("form button[type=submit]").click(); await page.locator("a.service-button-link").waitFor();
  await page.goto(`${base}/?service=api&lang=en`);
  await page.locator("#signin-workspace").fill("sample-workspace"); await page.locator("#signin-email").fill(login);
  await page.locator("#signin-password").fill(password); await page.locator("form button[type=submit]").click();
  await page.locator(".service-session").waitFor();

  await open("en", "accounts");
  await page.locator("#svc-name").fill("rotation");
  await page.locator("#svc-owner").selectOption({ label: login });
  await page.locator("#live-reason").fill("synthetic rendered check");
  await panel().locator("form button[type=submit]").click();
  await page.locator(".service-status").waitFor();
  await open("en", "credentials");
  await page.locator("#svc-select").selectOption(service);
  const firstSecret = await issue("30");
  check("a credential is issued and its secret shown once", firstSecret.startsWith("hbsk_"));
  const [first] = await live();
  check("a live credential offers Rotate beside Revoke", (await rowOf(first).locator("[data-rotate]").count()) === 1 && (await rowOf(first).locator("[data-credential]").count()) === 1);

  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 1024, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await open(lang, "credentials");
      await page.locator("#svc-select").selectOption(service);
      await rowOf(first).locator("[data-rotate]").click();
      await page.locator("#svc-overlap").waitFor();
      check(`${lang} ${width} the form is the rotation form with the installation default`, (await page.locator("#svc-issue-title").innerText()).trim() === copy[lang].title && (await page.locator("#svc-overlap").inputValue()) === "1440" && (await panel().innerText()).includes(first));
      await page.locator("#svc-overlap").fill("5");
      await page.locator("#live-reason").fill("synthetic rendered check");
      await panel().locator("form button[type=submit]").click();
      await page.locator("#svc-overlap-error").waitFor();
      check(`${lang} ${width} an overlap below the pickup time is refused before sending`, (await page.locator("#svc-overlap-error").innerText()).includes(copy[lang].error) && (await page.locator("#svc-review-title").count()) === 0);
      await page.locator("#svc-overlap").fill("60");
      await panel().locator("form button[type=submit]").click();
      await page.locator("#svc-review-title").waitFor();
      const text = await panel().innerText();
      check(`${lang} ${width} the review states the facts before anything is sent`, text.includes(copy[lang].review) && text.includes(first) && text.includes("UTC") && text.includes(copy[lang].about) && text.includes(copy[lang].confirm) && text.includes(copy[lang].once) && text.includes(copy[lang].back) && text.includes(copy[lang].unknown) && text.includes(copy[lang].exposed), text);
      check(`${lang} ${width} the review does not repeat the lead or the current-credential line`, (await panel().locator(".live-who").count()) === 1 && (await panel().locator(".live-impact").count()) === 0);
      check(`${lang} ${width} the review has the focus on its title`, await page.evaluate(() => document.activeElement?.id === "svc-review-title"));
      await contained(`${lang} ${width} rotation review`);
      await shot(`rotation-review-${lang}-${width}`);
    }
  }

  await page.setViewportSize({ width: 1440, height: 1000 });
  await open("en", "credentials");
  await page.locator("#svc-select").selectOption(service);
  const posts = [];
  page.on("request", (q) => { if (rotatePath(q.url())) posts.push(JSON.parse(q.postData() ?? "{}")); });
  await rowOf(first).locator("[data-rotate]").click();
  await page.locator("#svc-days").fill("7");
  await page.locator("#svc-overlap").fill("60");
  await page.locator("#live-reason").fill("synthetic rendered check");
  await panel().locator("form button[type=submit]").click();
  await page.locator("#svc-review-title").waitFor();
  check("nothing is sent before the review is confirmed", posts.length === 0);
  const answered = page.waitForResponse((q) => rotatePath(q.url()));
  await panel().locator("form button[type=submit]").click();
  const response = await answered;
  const body = await response.json();
  check("the rotation is sent once, for the chosen credential, with the entered values", posts.length === 1 && posts[0].target_id === first && posts[0].body.days === "7" && posts[0].body.overlap_minutes === "60", JSON.stringify(posts));
  check("the service rotates and names the replaced credential and its end", response.status() === 200 && body.business_state === "credential_rotated" && body.fields.replaced === first && !!body.fields.replaced_until, JSON.stringify(body).slice(0, 300));
  const secondSecret = await reveal();
  check("the new secret is shown once and differs from the old one", secondSecret.startsWith("hbsk_") && secondSecret !== firstSecret);
  await page.waitForLoadState("networkidle");
  const second = body.fields.credential;
  for (const lang of ["en", "ko", "ar"]) {
    for (const width of [1440, 1024, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      await open(lang, "credentials");
      await page.locator("#svc-select").selectOption(service);
      const oldRow = await rowOf(first).innerText(), newRow = await rowOf(second).innerText();
      check(`${lang} ${width} the replaced credential shows its fixed end and what replaced it`, oldRow.includes(copy[lang].ending) && oldRow.includes(copy[lang].replacedBy) && oldRow.includes(second), oldRow);
      check(`${lang} ${width} the new credential shows what it replaces`, newRow.includes(copy[lang].replaces) && newRow.includes(first), newRow);
      const firstRow = rowOf(first), secondRow = rowOf(second);
      check(`${lang} ${width} a credential whose end is fixed offers no second rotation, the new one does`, (await firstRow.locator("[data-rotate]").count()) === 0 && (await firstRow.locator("[data-credential]").count()) === 1 && (await secondRow.locator("[data-rotate]").count()) === 1);
      const spill = await page.evaluate(() => [...document.querySelectorAll(".live-creds tbody td")].filter((td) => td.scrollWidth > td.clientWidth + 1).length);
      check(`${lang} ${width} no credential cell is wider than its column`, spill === 0, String(spill));
      await contained(`${lang} ${width} credentials after a rotation`);
      await shot(`rotation-list-${lang}-${width}`);
    }
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
