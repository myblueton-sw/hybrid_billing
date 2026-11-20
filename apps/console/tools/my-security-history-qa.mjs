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
const output = resolve(options.get("--output") ?? join(tmpdir(), "hb-m05-history-qa"));
await mkdir(output, { recursive: true });
const run = await mkdtemp(join(output, "run-"));
const work = await mkdtemp(join(tmpdir(), "hb-m05-history-build-"));
const sources = ["src/modules/identity/pages/my-security/AccessHistory.tsx", "src/modules/identity/pages/my-security/scoped-read.ts", "src/modules/identity/pages/my-security/history-copy.ts"];
const sourceHashes = await Promise.all(sources.map(async path => ({ path, sha256: createHash("sha256").update(await readFile(join(consoleRoot, path))).digest("hex") })));
const pathImport = path => JSON.stringify(join(consoleRoot, path).replaceAll("\\", "/"));
const entry = `import React,{useState,useLayoutEffect} from "react";
import {createRoot} from "react-dom/client";import {flushSync} from "react-dom";
import AccessHistory from ${pathImport(sources[0])};
window.requests=[];window.observations=[];window.failNext=false;window.delayNext=false;window.releases=[];window.notices=[];
const clients=Object.fromEntries(['a','b'].map(scope=>[scope,{accessHistory:async query=>{window.requests.push({scope,...query});if(window.failNext){window.failNext=false;throw Error('synthetic failure');}if(window.delayNext){window.delayNext=false;await new Promise(done=>window.releases.push(done));}const n=Number(query.before||0);return {events:[0,1].map(i=>({at:'2026-11-20T00:00:00Z',kind:'signed_in',reason:'',workspace:scope,count:0,session:{handle:scope+'-'+n+'-'+i,current:false,active:true},device:null})),next:String(n+1),retention_days:30,retention_source:'default',period_since:query.from||null,effective_days:30,available_from:'2026-11-01T00:00:00Z',recording:{state:'recording',since:null,reason:null},notes:[],now:'2026-11-20T01:00:00Z'};}}]));
function Fixture(){const[scope,setScope]=useState('a');const[locale,setLocale]=useState('en');window.setLocale=setLocale;const[reload,setReload]=useState(0);window.reload=()=>setReload(n=>n+1);window.switchScope=()=>flushSync(()=>setScope('b'));useLayoutEffect(()=>{window.observations.push({scope,text:document.body.textContent});},[scope]);return <AccessHistory l={locale} api={clients[scope]} self={{user_id:scope,workspace:scope,current_session:{handle:scope}}} reloadKey={reload} say={text=>window.notices.push(text)} onSignedOut={()=>{throw Error('unexpected sign-out')}}/>}
createRoot(document.getElementById('root')).render(<Fixture/>);`;
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
  page.on("pageerror", error => errors.push(error.message));
  await page.goto(url);
  const checks=[];
  const check=(name,pass)=>{checks.push({name,pass});if(!pass)throw Error(name);};
  await page.getByText('a-0-0',{exact:true}).waitFor();
  await page.locator('.mysec-hist-more-btn').click();
  await page.getByText('a-1-0',{exact:true}).waitFor();
  await page.waitForFunction(()=>document.activeElement?.id==='mysec-history-title');
  check('navigation focuses history title',await page.locator('#mysec-history-title').evaluate(e=>e===document.activeElement));
  check('older page replaces prior rows',await page.locator('tbody tr').count()===2 && await page.getByText('a-0-0',{exact:true}).count()===0);
  await page.getByRole('button',{name:'Previous page',exact:true}).click();
  await page.getByText('a-0-0',{exact:true}).waitFor();
  check('previous rereads bounded page',await page.locator('tbody tr').count()===2);
  await page.evaluate(()=>window.failNext=true);
  await page.locator('.mysec-hist-more-btn').click();
  await page.locator('.mysec-hist-more-failed').waitFor();
  await page.waitForFunction(()=>document.activeElement?.classList.contains('mysec-hist-more-failed'));
  check('failure focuses retry notice',await page.locator('.mysec-hist-more-failed').evaluate(e=>e===document.activeElement));
  check('failure retains current page',await page.getByText('a-0-0',{exact:true}).count()===1);
  await page.locator('.mysec-hist-more-failed button').click();
  await page.getByText('a-1-0',{exact:true}).waitFor();
  check('retry requests same continuation',await page.evaluate(()=>window.requests.slice(-2).every(r=>r.before==='1')));
  await page.locator('#mysec-hist-period').selectOption('day');
  await page.getByText('a-0-0',{exact:true}).waitFor();
  check('period resets navigation',await page.getByRole('button',{name:'Previous page',exact:true}).isDisabled());
  const fixedFrom=await page.evaluate(()=>window.requests.at(-1).from);
  for(let n=1;n<=35;n++){
    await page.locator('.mysec-hist-more-btn').click();
    await page.getByText('a-'+n+'-0',{exact:true}).waitFor();
    check('bounded page '+n,await page.locator('tbody tr').count()===2);
  }
  check('continuations retain period bound',await page.evaluate(from=>window.requests.slice(-35).every(r=>r.from===from),fixedFrom));
  await page.evaluate(()=>window.failNext=true);
  await page.getByRole('button',{name:'Previous page',exact:true}).click();
  await page.locator('.mysec-hist-more-failed').waitFor();
  await page.locator('.mysec-hist-more-failed button').click();
  await page.getByText('a-34-0',{exact:true}).waitFor();
  check('previous retry retains direction',await page.evaluate(()=>window.requests.slice(-2).every(r=>r.before==='34')));
  for(let n=33;n>=3;n--){
    await page.getByRole('button',{name:'Previous page',exact:true}).click();
    await page.getByText('a-'+n+'-0',{exact:true}).waitFor();
  }
  check('backward trail stops at 32 entries',await page.getByRole('button',{name:'Previous page',exact:true}).isDisabled());
  await page.getByRole('button',{name:'First page',exact:true}).click();
  await page.getByText('a-0-0',{exact:true}).waitFor();
  check('first remains reachable after trail truncation',await page.locator('tbody tr').count()===2);
  for(const locale of ['en','ko','ar'])for(const width of [1440,1280,1024,390]){
    await page.evaluate(locale=>{window.setLocale(locale);document.documentElement.dir=locale==='ar'?'rtl':'ltr';document.documentElement.lang=locale;},locale);
    await page.setViewportSize({width,height:1000});
    await page.waitForFunction(locale=>document.documentElement.lang===locale,locale);
    check(locale+' '+width+' bounded localized history',await page.locator('tbody tr').count()===2);
    await page.screenshot({path:join(run,locale+'-'+width+'.png'),fullPage:true});
  }
  await page.evaluate(()=>window.setLocale('en'));
  await page.evaluate(()=>{window.delayNext=true;window.reload();});
  await page.waitForFunction(()=>window.releases.length===1);
  await page.evaluate(()=>window.switchScope());
  await page.getByText('b-0-0',{exact:true}).waitFor();
  await page.evaluate(async()=>{window.releases.shift()();await new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done)));});
  await page.waitForFunction(()=>window.requests.filter(r=>r.scope==='b').length>0);
  check('scope commit hides prior rows',await page.evaluate(()=>!window.observations.some(r=>r.scope==='b'&&r.text.includes('a-0-0'))));
  check('late response cannot replace current scope',await page.getByText('a-0-0',{exact:true}).count()===0 && await page.locator('tbody tr').count()===2);
  check('no page exceptions',errors.length===0);
  report={sourceHashes,pass:checks.every(c=>c.pass),checks,errors};
} catch (error) { report.error = error.message; }
finally {
  await browser?.close();
  if (server) await new Promise(done => server.close(done));
  await rm(work, { recursive: true, force: true });
  await writeFile(join(run, "report.json"), JSON.stringify(report, null, 2) + "\n");
}
console.log(JSON.stringify({ pass: report.pass, report: join(run, "report.json") }));
if (!report.pass) process.exitCode = 1;
