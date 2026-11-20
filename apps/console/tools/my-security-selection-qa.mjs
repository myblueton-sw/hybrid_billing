import { createRequire } from "node:module";
import { createServer } from "node:http";
import { createHash } from "node:crypto";
import { mkdtemp, mkdir, readFile, writeFile, rm } from "node:fs/promises";
import { dirname, resolve, join, extname, sep } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";
import { build } from "vite";

const options = new Map();
for (let i = 2; i < process.argv.length; i += 2) {
  const key = process.argv[i], value = process.argv[i + 1];
  if (!["--playwright-module", "--chromium", "--output"].includes(key) || !value) throw Error("Expected --playwright-module, --chromium and optional --output paths");
  options.set(key, value);
}
if (!options.has("--playwright-module") || !options.has("--chromium")) throw Error("Playwright module and Chromium executable paths are required");
const { chromium } = createRequire(import.meta.url)(resolve(options.get("--playwright-module")));
const consoleRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const output = resolve(options.get("--output") ?? join(tmpdir(), "hb-m05-selection-qa"));
await mkdir(output, { recursive: true });
const run = await mkdtemp(join(output, "run-"));
const work = await mkdtemp(join(tmpdir(), "hb-m05-selection-build-"));
const sources = ["src/modules/identity/pages/my-security/SessionSelection.tsx", "src/modules/identity/pages/my-security/session-intent.ts", "src/modules/identity/pages/my-security/scoped-read.ts"];
const sourceHashes = await Promise.all(sources.map(async path => ({ path, sha256: createHash("sha256").update(await readFile(join(consoleRoot, path))).digest("hex") })));
const pathImport = path => JSON.stringify(join(consoleRoot, path).replaceAll("\\", "/"));
const entry = `import React from "react";import {createRoot} from "react-dom/client";
import SessionSelection from ${pathImport(sources[0])};import{intentStore}from ${pathImport(sources[1])};import{AdminError}from ${pathImport("src/service/admin.ts")};
let serial=0;window.mode='definitive';window.calls=[];window.selectionClosed=0;
const row={handle:'0123456789abcdef',current:false,signed_in:'2026-11-20T00:00:00Z',ends:'2026-11-20T12:00:00Z',state:'active',can_end:true,not_included_operation:''};
const selections=new Map();const api={prepareSessionSelection:async resume=>{const value=resume?{...selections.get(resume.selection),state:'sealed',observed:2,count:2,digest:'a'.repeat(64),after:''}:{kind:'others',selection:'selection-'+(++serial),state:'building',reference_at:'2026-11-20T00:00:00Z',expires_at:'2026-11-20T00:05:00Z',generation:'1',observed:1,count:null,digest:'',after:'cursor',settings:[]};selections.set(value.selection,value);return value;},previewSessionSelection:async reference=>({...selections.get(reference),rows:[row,{...row,handle:'fedcba9876543210'}],next:''}),endSelectedSessions:async fingerprint=>{window.calls.push({...fingerprint});if(window.mode==='definitive')throw new AdminError('conflict',[],0,'','revision_conflict');throw Error('synthetic lost response');}};
const locale=new URLSearchParams(location.search).get('lang')||'en';createRoot(document.getElementById('root')).render(<SessionSelection l={locale} api={api} store={intentStore(api,'synthetic-scope')} onClose={()=>window.selectionClosed++} onSignedOut={()=>{throw Error('unexpected signout')}}/>);`;
await writeFile(join(work, "entry.tsx"), entry);
await writeFile(join(work, "index.html"), '<!doctype html><html><head><meta charset="UTF-8"></head><body><div id="root"></div><script type="module" src="/entry.tsx"></script></body></html>');
let browser, server;
let report = { sourceHashes, pass: false };
try {
  await build({ configFile: false, root: work, resolve: { alias: { react: join(consoleRoot, "node_modules/react"), "react-dom": join(consoleRoot, "node_modules/react-dom") } }, build: { outDir: join(work, "dist") } });
  const dist = join(work, "dist");
  server = createServer(async (request, response) => {
    try {
      const pathname = decodeURIComponent(new URL(request.url, "http://localhost").pathname);
      const path = resolve(dist, "." + pathname.replace(/\/$/, "/index.html"));
      if (!path.startsWith(dist + sep)) { response.statusCode = 403; response.end(); return; }
      response.setHeader("content-type", ({ ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml" })[extname(path)] ?? "application/octet-stream");
      response.end(await readFile(path));
    } catch { response.statusCode = 404; response.end(); }
  });
  await new Promise(done => server.listen(0, "127.0.0.1", done));
  const url = "http://127.0.0.1:" + server.address().port;
  browser = await chromium.launch({ headless: true, executablePath: resolve(options.get("--chromium")) });
  const page = await browser.newPage();
  const errors = [];
  page.on("pageerror", error => { errors.push(error.message); report.errors = errors; });
  const checks=[];report.checks=checks;
  const check=(name,pass)=>{checks.push({name,pass});if(!pass)throw Error(name);};
  const resume={en:'Continue preparation',ko:'목록 준비 계속',ar:'متابعة الإعداد'};
  const fresh={en:'Prepare a new fixed list',ko:'새 고정 목록 준비',ar:'إعداد قائمة ثابتة جديدة'};
  for(const lang of ['en','ko','ar']){
    await page.goto(url+'/?lang='+lang);const dialog=page.locator('dialog[open]');await dialog.getByRole('button',{name:resume[lang],exact:true}).waitFor();check(lang+' building has no final confirmation',await dialog.locator('input[type=checkbox]').count()===0);await dialog.getByRole('button',{name:resume[lang],exact:true}).click();await dialog.locator('tbody tr').first().waitFor();await page.waitForFunction(()=>document.activeElement?.id==='mysec-selection-title');check(lang+' sealed preparation restores contained focus',await dialog.locator('#mysec-selection-title').evaluate(el=>el===document.activeElement));
    await dialog.locator('input[type=checkbox]').check();await dialog.locator('button.live-primary').click();await dialog.locator('[role=alert]').waitFor();
    check(lang+' definitive refusal removes stale confirmation',await dialog.locator('input[type=checkbox]').count()===0&&await dialog.locator('button.live-primary').count()===0);
    check(lang+' fresh review is explicit',await dialog.getByRole('button',{name:fresh[lang],exact:true}).count()===1);
    await dialog.getByRole('button',{name:fresh[lang],exact:true}).click();await dialog.getByRole('button',{name:resume[lang],exact:true}).click();await dialog.locator('tbody tr').first().waitFor();
    check(lang+' fresh list requires new confirmation',!await dialog.locator('input[type=checkbox]').isChecked()&&await dialog.locator('button.live-primary').isDisabled());
    await page.evaluate(()=>window.mode='unknown');await dialog.locator('input[type=checkbox]').check();await dialog.locator('button.live-primary').click();await dialog.locator('.mysec-intent').waitFor();
    check(lang+' unknown outcome retains fixed list and locks effects',await dialog.locator('input[type=checkbox]').count()===1&&await dialog.locator('button.live-primary').isDisabled()&&await dialog.getByRole('button',{name:fresh[lang],exact:true}).count()===0);
    await dialog.locator('.mysec-intent button').first().click();await page.waitForFunction(()=>document.querySelector('dialog .mysec-intent button')?.disabled===false);
    check(lang+' sealed readback keeps uncertainty',await dialog.locator('button.live-primary').isDisabled());
    await page.evaluate(()=>window.mode='definitive');await dialog.locator('.mysec-intent button').nth(1).click();await page.waitForFunction(()=>document.querySelector('dialog .mysec-intent button')?.disabled===false);
    check(lang+' replay refusal cannot discard prior unknown intent',await dialog.locator('.mysec-intent').count()===1&&await dialog.getByRole('button',{name:fresh[lang],exact:true}).count()===0);
    check(lang+' replay preserves identical request fingerprint',await page.evaluate(()=>JSON.stringify(window.calls.at(-1))===JSON.stringify(window.calls.at(-2))));
  }
  check('no page exceptions',errors.length===0);report={sourceHashes,pass:checks.every(check=>check.pass),checks,errors};
} catch (error) { report.error = error.message; const current = browser?.contexts()[0]?.pages()[0]; if (current) { report.fixtureText = await current.locator("body").innerText(); await current.screenshot({path:join(run,"failure.png"),fullPage:true}); } }
finally {
  await browser?.close();
  if (server) await new Promise(done => server.close(done));
  await rm(work, { recursive: true, force: true });
  await writeFile(join(run, "report.json"), JSON.stringify(report, null, 2) + "\n");
}
console.log(JSON.stringify({ pass: report.pass, report: join(run, "report.json") }));
if (!report.pass) process.exitCode = 1;
