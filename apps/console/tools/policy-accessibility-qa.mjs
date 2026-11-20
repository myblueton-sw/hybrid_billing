import { createRequire } from "node:module";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { randomBytes } from "node:crypto";
import { resolve } from "node:path";
import assert from "node:assert/strict";
const args = Object.fromEntries(process.argv.slice(2).flatMap((v, i, all) => v.startsWith("--") ? [[v.slice(2), all[i + 1]]] : []));
const { chromium } = createRequire(import.meta.url)(args["playwright-module"] || "playwright");
const base = args.url, upstream = args.api, locale = args.locale || "en", width = Number(args.width || 1440), output = resolve(args.output);
assert.ok(base && upstream && args.activation);
const browser = await chromium.launch({ headless: true, ...(args.chromium ? { executablePath: args.chromium } : {}) });
const links = (await readFile(args.activation, "utf8")).trim().split("\n"), password = randomBytes(24).toString("base64url");
const evidence = { locale, width, outcome: "running", checks: [], errors: [] };
let stage = "fixture setup";
await mkdir(output, { recursive: true });
async function actor(login) {
  const context = await browser.newContext({ viewport: { width, height: 1000 } }), page = await context.newPage();
  page.on("pageerror", (e) => evidence.errors.push(e.message));
  await page.route("**/api/v1/**", async (route) => { const u = new URL(route.request().url()); const response = await route.fetch({ url: upstream + u.pathname + u.search }); await route.fulfill({ response }); });
  const link = new URL(links.find((line) => line.startsWith(login + " ")).split(" ")[1]);
  await page.goto(`${base}/activate?lang=${locale}${link.hash}`);
  await page.locator("#link-new").fill(password); await page.locator("#link-confirm").fill(password); await page.locator("form button[type=submit]").click(); await page.locator("a.service-button-link").waitFor();
  await page.goto(`${base}/?service=api&lang=${locale}`);
  await page.locator("#signin-workspace").fill("sample-workspace"); await page.locator("#signin-email").fill(login); await page.locator("#signin-password").fill(password); await page.locator("form button[type=submit]").click(); await page.locator(".service-session").waitFor();
  return page;
}
async function tabTo(page, selector) {
  for (let i = 0; i < 240; i++) { if (await page.locator(selector).evaluate((e) => e === document.activeElement)) return; await page.keyboard.press("Tab"); }
  throw new Error(`Keyboard could not reach ${selector}`);
}
async function capture(page, lang, size, state) {
  await page.evaluate(() => document.fonts.ready);
  const visual = await page.evaluate(() => ({
    failedFonts: [...document.fonts].filter(f => f.status === "error").map(f => f.family),
    loadedFonts: [...document.fonts].filter(f => f.status === "loaded").map(f => f.family),
    borders: [...document.querySelectorAll('dialog, dialog .service-summary, dialog #live-reason')].map(e => {
      const c = getComputedStyle(e), r = e.getBoundingClientRect();
      return { tag: e.tagName, className: e.className, focused: e === document.activeElement, border: c.border, outline: c.outline, color: c.color, background: c.backgroundColor, left: r.left, right: r.right };
    })
  }));
  assert.deepEqual(visual.failedFonts, []);
  assert.ok(visual.loadedFonts.includes(lang === "ko" ? "IBM Plex Sans KR" : lang === "ar" ? "IBM Plex Sans Arabic" : "IBM Plex Sans"));
  assert.ok(visual.borders.every(e => e.left >= -1 && e.right <= size + 1));
  evidence.visual ||= []; evidence.visual.push({lang, width: size, state, ...visual});
  await page.screenshot({path: resolve(output, `M02-${state}-${lang}-${size}.png`)});
}
let reviewer;
try {
  reviewer = await actor("admin2@example.invalid");
  const commands=[]; reviewer.on("request", r=>{ const p=new URL(r.url()).pathname; if(p.startsWith("/api/v1/commands/")) commands.push(p); });
  stage = "policy journey";
  for (const lang of ["en","ko","ar"]) for (const size of [1440,390]) {
    await reviewer.setViewportSize({width:size,height:size===390?844:1000});
    await reviewer.goto("about:blank");
    const loading = reviewer.waitForResponse(r=>new URL(r.url()).pathname === "/api/v1/identity/policy" && r.status()===200);
    await reviewer.goto(`${base}/?service=api&lang=${lang}#/authentication?lang=${lang}&view=policies`);
    await (await loading).finished();
    await reviewer.locator('.m02-live:not([aria-busy="true"]) .m02-propose').waitFor();
    await reviewer.waitForFunction(wide=>document.querySelector(".m02-split")?.getAttribute("data-mode") === (wide?"split":"step"),size>=1280);
    await reviewer.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));
    await tabTo(reviewer,".m02-propose"); await reviewer.keyboard.press("Enter"); await reviewer.locator(".m02-form h2:focus").waitFor(); await reviewer.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));
    await tabTo(reviewer,"#m02-age"); await reviewer.keyboard.press(process.platform === "darwin" ? "Meta+A" : "Control+A"); await reviewer.keyboard.type("60"); assert.equal(await reviewer.locator("#m02-age").inputValue(),"60");
    await tabTo(reviewer,'.m02-form button[type="submit"]'); await reviewer.keyboard.press("Enter");
    await reviewer.locator("#m02-confirm-title:focus").waitFor();
    for(let attempt=0;attempt<2;attempt++) {
      await tabTo(reviewer,'dialog button[type="submit"]'); await reviewer.keyboard.press("Enter");
      await reviewer.locator("dialog .service-summary:focus").waitFor();
      await reviewer.evaluate(()=>document.fonts.ready);
      if(attempt===1) await capture(reviewer, lang, size, "summary");
      const before=reviewer.url(); await tabTo(reviewer,'dialog .service-summary a'); await reviewer.keyboard.press("Enter"); await reviewer.locator("#live-reason:focus").waitFor(); assert.equal(reviewer.url(),before);
    }
    await capture(reviewer, lang, size, "error");
    for(const key of [...Array(12).fill("Tab"),...Array(12).fill("Shift+Tab")]) {
      await reviewer.keyboard.press(key);
      assert.equal(await reviewer.evaluate(()=>!!document.activeElement?.closest("dialog[open]")||!document.hasFocus()&&document.activeElement?.tagName==="BODY"),true);
    }
    for(let i=0;i<5 && !await reviewer.evaluate(()=>document.hasFocus());i++) await reviewer.keyboard.press("Tab");
    assert.equal(await reviewer.evaluate(()=>document.hasFocus() && !!document.activeElement?.closest("dialog[open]")),true);
    const original = await reviewer.locator('.m02-form button[type="submit"]').elementHandle();
    await reviewer.setViewportSize({width:size===1440?390:1440,height:size===390?844:1000});
    await reviewer.waitForFunction(wide=>!!document.querySelector(wide?".m02-detail .m02-form":".m02-step .m02-form"),size!==1440);
    assert.equal(await original.evaluate(e=>e.isConnected),false);
    await reviewer.keyboard.press("Escape"); await reviewer.locator('.m02-form button[type="submit"]:focus').waitFor();
    await reviewer.keyboard.press("Enter"); await reviewer.locator("#m02-confirm-title:focus").waitFor();
    const cancelOpener = await reviewer.locator('.m02-form button[type="submit"]').elementHandle();
    await reviewer.setViewportSize({width:size,height:size===390?844:1000});
    await reviewer.waitForFunction(wide=>!!document.querySelector(wide?".m02-detail .m02-form":".m02-step .m02-form"),size===1440);
    assert.equal(await cancelOpener.evaluate(e=>e.isConnected),false);
    await tabTo(reviewer,"dialog .live-secondary"); await reviewer.keyboard.press("Enter"); await reviewer.locator('.m02-form button[type="submit"]:focus').waitFor();
    await tabTo(reviewer,'.m02-form .live-secondary'); await reviewer.keyboard.press("Enter"); await reviewer.locator(".m02-propose:focus").waitFor();
    evidence.checks.push({lang,width:size,validationAttempts:2,modalNavigation:24,errorLink:true,returnFocus:true,detachedOpener:true,cancelReturn:true,detachedCancelOpener:true});
  }
  assert.deepEqual(commands,[]); assert.deepEqual(evidence.errors,[]); evidence.commands=commands;evidence.outcome="passed";
} catch(error){ evidence.outcome="failed";evidence.failure={kind:error.name,...(stage === "policy journey" ? {message:error.message} : {})}; if(reviewer) evidence.focus=await reviewer.evaluate(()=>({hasFocus:document.hasFocus(),tag:document.activeElement?.tagName,id:document.activeElement?.id,classes:document.activeElement?.className,dialog:!!document.querySelector("dialog[open]")})); if(reviewer) await reviewer.screenshot({path:resolve(output,"failure.png"),fullPage:true});throw new Error("M02 QA failed: "+error.name); }
finally{await writeFile(resolve(output,"evidence.json"),JSON.stringify(evidence,null,2));await browser.close();}
