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
const output = resolve(options.get("--output") ?? join(tmpdir(), "hb-auth-root-qa"));
await mkdir(output, { recursive: true });
const run = await mkdtemp(join(output, "run-"));
const work = await mkdtemp(join(tmpdir(), "hb-auth-root-build-"));
const sources = ["src/service/Service.tsx", "src/service/admin.ts", "src/service/context.ts", "src/service/Reauthenticate.tsx"];
const sourceHashes = await Promise.all(sources.map(async path => ({ path, sha256: createHash("sha256").update(await readFile(join(consoleRoot, path))).digest("hex") })));
const pathImport = path => JSON.stringify(join(consoleRoot, path).replaceAll("\\", "/"));
const entry = `import React,{useState}from"react";import{createRoot}from"react-dom/client";import{flushSync}from"react-dom";
import{ServiceRoot}from${pathImport(sources[0])};import{useService,useAdminClient}from${pathImport(sources[2])};
const a={user:'a',workspace:'w',role:'member',login:'a@example.invalid',csrf:'synthetic-a'},b={...a,user:'b',login:'b@example.invalid',csrf:'synthetic-b'};
window.precommit=[];
const client={current:async()=>a,login:()=>({then(resolve){resolve(b);queueMicrotask(()=>{window.precommit.push({newIdentityVisible:!!document.querySelector('[data-user=b]')});window.rootObservers.find(row=>row.csrf==='synthetic-a').listener();});}}),logout:async()=>{}};
function Consumer(){const api=useAdminClient();const ctx=useService();window.consumerClient=api;return <span>Consumer {ctx.session.user}</span>}
function App(){const ctx=useService();const[consumer,setConsumer]=useState(true);window.plainRoute=()=>flushSync(()=>setConsumer(false));return <div data-user={ctx.session.user}><strong id="sensitive">Sensitive {ctx.session.user}</strong>{consumer?<Consumer/>:<p id="plain-route">Route without an API consumer</p>}</div>}
createRoot(document.getElementById('root')).render(<ServiceRoot client={client} app={<App/>}/>);`;
await writeFile(join(work,"admin-fixture.ts"),`import{adminClient as create}from${pathImport(sources[1])};export function adminClient(...args){const api=create(...args);window.rootClients??={};window.rootClients[args[0]]=api;const subscribe=api.subscribeAuthenticationLoss;api.subscribeAuthenticationLoss=listener=>{window.rootObservers??=[];window.rootObservers.push({csrf:args[0],listener});return subscribe(listener);};return api;}`);
await writeFile(join(work,"reauth-fixture.tsx"),`import React from"react";import{Reauthenticate as Actual}from${pathImport(sources[3])};export function Reauthenticate(props){window.promptObservers??=[];window.promptObservers.push(props);return React.createElement(Actual,{...props,onDone:ok=>{window.dialogDeliveries??=[];window.dialogDeliveries.push({csrf:props.csrf,kind:"done"});props.onDone(ok);},onSignedOut:()=>{window.dialogDeliveries??=[];window.dialogDeliveries.push({csrf:props.csrf,kind:"signed-out"});props.onSignedOut();}});}`);
await writeFile(join(work, "entry.tsx"), entry);
await writeFile(join(work, "index.html"), '<!doctype html><html><head><meta charset="UTF-8"></head><body><div id="root"></div><script type="module" src="/entry.tsx"></script></body></html>');
let browser, server;
let report = { sourceHashes, pass: false };
try {
  await build({ configFile: false, root: work, plugins:[{name:"synthetic-auth-root-observer",enforce:"pre",resolveId(source,importer){if((source==="./admin"||source.endsWith("/src/service/admin.ts"))&&importer?.replaceAll("\\","/").endsWith("/src/service/Service.tsx"))return join(work,"admin-fixture.ts");if((source==="./Reauthenticate"||source.endsWith("/src/service/Reauthenticate.tsx"))&&importer?.replaceAll("\\","/").endsWith("/src/service/Service.tsx"))return join(work,"reauth-fixture.tsx");}}], resolve: { alias: { react: join(consoleRoot, "node_modules/react"), "react-dom": join(consoleRoot, "node_modules/react-dom") } }, build: { outDir: join(work, "dist") } });
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
  const checks=[];report.checks=checks;const check=(name,pass)=>{checks.push({name,pass});if(!pass)throw Error(name);};
  let held,first=true,releaseHeld;const heldReady=new Promise(done=>releaseHeld=done);let writes=0;
  await page.route("**/api/v1/me",async route=>{if(route.request().headers()['x-csrf-token']==='synthetic-a'&&first){first=false;held=route;releaseHeld();return;}await route.fulfill({status:401,json:{code:'reauthentication_required'}});});
  await page.route("**/api/v1/me/sessions/end",async route=>{writes++;await route.fulfill({status:403,json:{code:'reauthentication_required'}});});
  await page.goto(url);await page.locator('[data-user=a]').waitFor();
  check('route consumer shares the root client',await page.evaluate(()=>window.consumerClient===window.rootClients['synthetic-a']));
  await page.evaluate(()=>{void window.rootClients['synthetic-a'].me().catch(()=>{});});await heldReady;
  await page.evaluate(()=>window.plainRoute());await page.locator('#plain-route').waitFor();
  await page.evaluate(()=>{void window.rootClients['synthetic-a'].me().catch(()=>{});});await page.locator('#signin-email').waitFor();
  check('root observes401 on a route without API consumers',await page.locator('#sensitive,.service-session').count()===0);
  check('misleading401 never opens step-up',await page.locator('dialog.service-reauth').count()===0);
  await page.locator('#signin-workspace').fill('w');await page.locator('#signin-email').fill('b@example.invalid');await page.locator('#signin-password').fill('synthetic password');await page.locator('form button[type=submit]').click();await page.locator('[data-user=b]').waitFor();
  check('old captured notification cannot erase a login before commit',await page.evaluate(()=>window.precommit.length===1&&window.precommit[0].newIdentityVisible===false));
  await held.fulfill({status:401,json:{code:'unauthenticated'}});await page.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));
  check('late old-client401 preserves the new identity',await page.locator('[data-user=b]').count()===1&&await page.locator('#signin-email').count()===0);
  await page.evaluate(()=>{void window.rootClients['synthetic-b'].endSelectedSessions({selection:'synthetic',digest:'a',count:1}).catch(error=>window.writeFailure=error.failure);});await page.locator('dialog.service-reauth').waitFor();
  await page.evaluate(()=>{void window.rootClients['synthetic-b'].me().catch(()=>{});});await page.locator('#signin-email').waitFor();await page.waitForFunction(()=>window.writeFailure==='reauth_cancelled');
  check('authentication loss cancels a pending step-up',await page.locator('dialog.service-reauth,#sensitive,.service-session').count()===0);
  check('cancelled step-up does not retry the mutation',writes===1);
  for(const oldStatus of [204,401]){
    const isolated=await browser.newContext();const challenge=await isolated.newPage();challenge.on('pageerror',e=>errors.push(e.message));let oldReauth,ready;const oldReady=new Promise(done=>ready=done);let bWrites=0;
    await challenge.route('**/api/v1/me',route=>route.fulfill({status:401,json:{code:'unauthenticated'}}));
    await challenge.route('**/api/v1/me/sessions/end',route=>{if(route.request().headers()['x-csrf-token']==='synthetic-b')bWrites++;return route.fulfill({status:403,json:{code:'reauthentication_required'}});});
    await challenge.route('**/api/v1/me/reauthenticate',route=>{oldReauth=route;ready();});
    await challenge.goto(url);await challenge.locator('[data-user=a]').waitFor();
    await challenge.evaluate(()=>{void window.rootClients['synthetic-a'].endSelectedSessions({selection:'old',digest:'a',count:1}).catch(()=>{});});await challenge.locator('dialog.service-reauth').waitFor();await challenge.locator('#reauth-password').fill('old synthetic password');await challenge.locator('dialog.service-reauth button[type=submit]').click();await oldReady;
    await challenge.evaluate(()=>{void window.rootClients['synthetic-a'].me().catch(()=>{});});await challenge.locator('#signin-email').waitFor();
    await challenge.locator('#signin-workspace').fill('w');await challenge.locator('#signin-email').fill('b@example.invalid');await challenge.locator('#signin-password').fill('synthetic password');await challenge.locator('form button[type=submit]').click();await challenge.locator('[data-user=b]').waitFor();
    await challenge.evaluate(()=>{void window.rootClients['synthetic-b'].endSelectedSessions({selection:'new',digest:'b',count:1}).catch(()=>{});});await challenge.locator('dialog.service-reauth').waitFor();
    await oldReauth.fulfill(oldStatus===204?{status:204}:{status:401,json:{code:'unauthenticated'}});await challenge.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));
    check('retired dialog '+oldStatus+' delivers no stale callback',await challenge.evaluate(()=>(window.dialogDeliveries??[]).filter(row=>row.csrf==='synthetic-a').length===0));
    check('late old reauthentication '+oldStatus+' preserves new challenge',await challenge.locator('[data-user=b]').count()===1&&await challenge.locator('#reauth-login').inputValue()==='b@example.invalid'&&bWrites===1);
    await challenge.evaluate(()=>{const old=window.promptObservers.find(row=>row.csrf==='synthetic-a');old.onDone(true);old.onSignedOut();});await challenge.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));
    check('captured old prompt callbacks '+oldStatus+' cannot settle another session',await challenge.locator('dialog.service-reauth[open]').count()===1&&await challenge.locator('[data-user=b]').count()===1&&bWrites===1);
    await challenge.locator('dialog.service-reauth button[type=button]').click();await challenge.waitForFunction(()=>!document.querySelector('dialog.service-reauth[open]'));await isolated.close();
  }
  {
    const isolated=await browser.newContext();const generation=await isolated.newPage();generation.on('pageerror',e=>errors.push(e.message));let requests=0;
    await generation.route('**/api/v1/me/sessions/end',route=>{requests++;return route.fulfill({status:403,json:{code:'reauthentication_required'}});});
    await generation.goto(url);await generation.locator('[data-user=a]').waitFor();
    const ask=()=>generation.evaluate(()=>{void window.rootClients['synthetic-a'].endSelectedSessions({selection:'same-owner',digest:'a',count:1}).catch(()=>{});});
    await ask();await generation.locator('dialog.service-reauth').waitFor();await generation.evaluate(()=>window.firstPrompt=window.promptObservers.at(-1));await generation.locator('dialog.service-reauth button[type=button]').click();await generation.waitForFunction(()=>!document.querySelector('dialog.service-reauth[open]'));
    await ask();await generation.locator('dialog.service-reauth').waitFor();await generation.evaluate(()=>{window.firstPrompt.onDone(true);window.firstPrompt.onSignedOut();});await generation.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));
    check('old same-session prompt generation cannot settle its successor',await generation.locator('dialog.service-reauth[open]').count()===1&&await generation.locator('[data-user=a]').count()===1&&requests===2);
    await isolated.close();
  }
  {
    const isolated=await browser.newContext();const late=await isolated.newPage();late.on('pageerror',e=>errors.push(e.message));let delayed,ready;const received=new Promise(done=>ready=done);let posts=0;
    await late.route('**/api/v1/me',route=>route.fulfill({status:401,json:{code:'unauthenticated'}}));
    await late.route('**/api/v1/me/sessions/end',route=>{posts++;if(posts===1){delayed=route;ready();return;}return route.fulfill({status:403,json:{code:'reauthentication_required'}});});
    await late.route('**/api/v1/me/reauthenticate',route=>route.fulfill({status:401,json:{code:'unauthenticated'}}));
    await late.goto(url);await late.locator('[data-user=a]').waitFor();await late.evaluate(()=>{void window.rootClients['synthetic-a'].endSelectedSessions({selection:'late',digest:'a',count:1}).catch(error=>window.oldChallengeFailure=error.failure);});await received;
    await late.evaluate(()=>{void window.rootClients['synthetic-a'].endSelectedSessions({selection:'confirm-before-loss',digest:'a',count:1}).catch(()=>{});});await late.locator('dialog.service-reauth').waitFor();await late.locator('#reauth-password').fill('synthetic password');await late.locator('dialog.service-reauth button[type=submit]').click();await late.locator('#signin-email').waitFor();await late.locator('#signin-workspace').fill('w');await late.locator('#signin-email').fill('b@example.invalid');await late.locator('#signin-password').fill('synthetic password');await late.locator('form button[type=submit]').click();await late.locator('[data-user=b]').waitFor();
    await delayed.fulfill({status:403,json:{code:'reauthentication_required'}});await late.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));
    check('late retired-client challenge cannot open in the new session',await late.locator('dialog.service-reauth[open]').count()===0&&await late.locator('[data-user=b]').count()===1&&await late.evaluate(()=>window.oldChallengeFailure==='reauth_cancelled')&&posts===2);await isolated.close();
  }
  {
    const isolated=await browser.newContext();const retired=await isolated.newPage();retired.on('pageerror',e=>errors.push(e.message));let posts=0;
    await retired.route('**/api/v1/me/sessions/end',route=>{posts++;return route.fulfill({status:403,json:{code:'reauthentication_required'}});});
    await retired.goto(url);await retired.locator('[data-user=a]').waitFor();await retired.evaluate(()=>{void window.rootClients['synthetic-a'].endSelectedSessions({selection:'settled-before-loss',digest:'a',count:1}).catch(error=>window.afterGateFailure=error.failure);});await retired.locator('dialog.service-reauth').waitFor();
    await retired.evaluate(()=>{window.promptObservers.at(-1).onDone(true);window.rootObservers.find(row=>row.csrf==='synthetic-a').listener();});await retired.locator('#signin-email').waitFor();await retired.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));
    check('retirement after gate approval prevents mutation retry',await retired.evaluate(()=>window.afterGateFailure==='reauth_cancelled')&&posts===1&&await retired.locator('#sensitive,dialog.service-reauth').count()===0);await isolated.close();
  }
  check('no page exceptions',errors.length===0);report={sourceHashes,pass:checks.every(check=>check.pass),checks,errors,limits:'Pre-commit root and obsolete prompt callbacks are explicitly injected observer deliveries. Late HTTP401, delayed real Reauthenticate success/401 and route clearing use actual components/transport against synthetic responses.'};
} catch (error) { report.error = error.message; const current = browser?.contexts()[0]?.pages()[0]; if(current){report.fixtureText=await current.locator("body").innerText();await current.screenshot({path:join(run,"failure.png"),fullPage:true});} }
finally {
  await browser?.close();
  if (server) await new Promise(done => server.close(done));
  await rm(work, { recursive: true, force: true });
  await writeFile(join(run, "report.json"), JSON.stringify(report, null, 2) + "\n");
}
console.log(JSON.stringify({ pass: report.pass, report: join(run, "report.json") }));
if (!report.pass) process.exitCode = 1;
