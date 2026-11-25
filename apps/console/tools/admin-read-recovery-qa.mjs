import {createRequire} from 'node:module';
import assert from 'node:assert/strict';
import {readFile,mkdir,writeFile} from 'node:fs/promises';
import {randomBytes} from 'node:crypto';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']||'playwright');
const base=args.url;
const out=args.output,api=args.api;
assert.ok(base&&out&&api&&args.activation);
await mkdir(out,{recursive:true});
const browser=await chromium.launch({headless:true,...(args.chromium?{executablePath:args.chromium}:{})});
const page=await browser.newPage({viewport:{width:1440,height:1000}}),password=randomBytes(24).toString('base64url');
await page.route('**/api/v1/**',async route=>{const u=new URL(route.request().url());const response=await route.fetch({url:api+u.pathname+u.search});await route.fulfill({response});});
const evidence={outcome:'running',renders:[],errors:[]};page.on('pageerror',e=>evidence.errors.push(e.message));
const lines=(await readFile(args.activation,'utf8')).trim().split('\n'),link=new URL(lines.find(l=>l.startsWith((args.user||'admin@example.invalid')+' ')).split(' ')[1]);
try{
 await page.goto(base+'/activate?lang=en'+link.hash);await page.locator('#link-new').fill(password);await page.locator('#link-confirm').fill(password);await page.locator('form button[type=submit]').click();await page.locator('a.service-button-link').waitFor();
 await page.goto(base+'/?service=api&lang=en');await page.locator('#signin-workspace').fill('sample-workspace');await page.locator('#signin-email').fill(args.user||'admin@example.invalid');await page.locator('#signin-password').fill(password);await page.locator('form button[type=submit]').click();await page.locator('.service-session').waitFor();

 const expected=lang=>({en:'Data reloaded.',ko:'데이터를 다시 불러왔습니다.',ar:'أُعيد تحميل البيانات.'})[lang];
 const screens=JSON.parse(await readFile(new URL('../src/modules/administration/screens.json',import.meta.url),'utf8'));
 let prefix;
 async function tabTo(selector){for(let i=0;i<100;i++){if(await page.locator(selector).evaluate(e=>e===document.activeElement))return;await page.keyboard.press('Tab');}throw new Error('Keyboard target not reached');}
 async function observeStatus(){await page.evaluate(()=>{window.__hbObserver?.disconnect();window.__hbStatusChanges=[];const region=document.querySelector('[data-read-observation]');window.__hbObserver=new MutationObserver(()=>window.__hbStatusChanges.push(region.textContent));window.__hbObserver.observe(region,{childList:true,subtree:true,characterData:true});});}
 const requests=[];page.on('request',r=>{if(r.method()==='POST'&&new URL(r.url()).pathname.includes('/commands/'))requests.push(new URL(r.url()).pathname);});
 for(const id of ['P01','P02','P03','P04','P05'])for(const lang of ['en','ko','ar'])for(const width of [1440,390]){
  const screen=screens.find(s=>s.id===id);prefix=id.toLowerCase();
  await page.setViewportSize({width,height:1000});await page.goto('about:blank');
  const usersRoute=u=>u.pathname==='/api/v1'+screen.path;
  let mode='fail',release,started;
  const hold=()=>{const entered=new Promise(resolve=>started=resolve),held=new Promise(resolve=>release=resolve);return{entered,held};};
  let gate=hold();
  await page.route(usersRoute,async route=>{
   if(mode==='fail'){await route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({code:'service_unavailable'})});return;}
   started();await gate.held;
   if(mode==='fail-after-hold'){await route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({code:'service_unavailable'})});return;}
   const u=new URL(route.request().url());const response=await route.fetch({url:api+u.pathname+u.search});await route.fulfill({response});
  });
  await page.goto(`${base}/?service=api&lang=${lang}#${screen.path}?lang=${lang}&view=${screen.tabs[0].key}`);
  await page.locator('.'+prefix+'-read-status').evaluate(e=>e.setAttribute('data-read-observation',''));
  await page.locator('.'+prefix+'-failure button').waitFor();await tabTo('.'+prefix+'-failure button');
  await page.locator('.'+prefix+'-failure button').evaluate(e=>e.scrollIntoView({block:'center'}));
  const borders=await page.locator('.'+prefix+'-failure, .'+prefix+'-failure button, .'+prefix+'-reload').evaluateAll(elements=>elements.map(e=>{const c=getComputedStyle(e);return{className:e.className,borders:[c.borderTop,c.borderRight,c.borderBottom,c.borderLeft],outline:c.outline};}));
  await page.screenshot({path:out+`/error-${id}-${lang}-${width}.png`});
  const repeatedResponse=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1'+screen.path&&r.status()===503);
  mode='fail-after-hold';await page.keyboard.press('Enter');await gate.entered;
  await page.locator('.'+prefix+'-reload:focus').waitFor();
  assert.equal(await page.locator('.'+prefix+'-failure').count(),1);await page.locator('.'+prefix+'-live[aria-busy=true]').waitFor();release();await(await repeatedResponse).finished();await page.waitForFunction(p=>!document.querySelector('.'+p+'-live[aria-busy=true]'),prefix);await page.locator('.'+prefix+'-failure button').waitFor();
  assert.ok(await page.locator('.'+prefix+'-reload').evaluate(e=>e===document.activeElement));
  assert.equal(await page.locator('.'+prefix+'-read-status').innerText(),'');
  await tabTo('.'+prefix+'-failure button');gate=hold();mode='success';await page.keyboard.press('Enter');await gate.entered;
  await page.locator('.'+prefix+'-reload:focus').waitFor();
  const loading=await page.evaluate(()=>({active:document.activeElement.id,status:[...document.querySelectorAll('[role=status]')].map(e=>e.textContent).filter(Boolean)}));
  release();await page.locator('.'+prefix+'-failure').waitFor({state:'detached'});
  const announcement=expected(lang);
  await page.waitForFunction(text=>document.querySelector('[data-read-observation]')?.textContent===text,announcement);
  assert.ok(await page.locator('.'+prefix+'-reload').evaluate(e=>e===document.activeElement));
  await page.screenshot({path:out+`/recovered-${id}-${lang}-${width}.png`});
  await observeStatus();gate=hold();await page.keyboard.press('Enter');await gate.entered;
  await page.keyboard.press('Tab');const moved=await page.evaluate(()=>({tag:document.activeElement.tagName,id:document.activeElement.id,className:document.activeElement.className}));
  assert.ok(!moved.className.includes(prefix+'-reload'));release();
  await page.waitForFunction(()=>!document.querySelector('[aria-busy=true]'));
  await page.waitForFunction(text=>document.querySelector('[data-read-observation]')?.textContent===text,announcement);
  await page.waitForFunction(text=>{const changes=window.__hbStatusChanges;return changes.includes('')&&changes.at(-1)===text;},announcement);
  assert.deepEqual(await page.evaluate(()=>({tag:document.activeElement.tagName,id:document.activeElement.id,className:document.activeElement.className})),moved);
  evidence.renders.push({id,lang,width,loading,announcement,borders,repeatedFailure:true,focusPreservedAfterMovement:true,freshAnnouncementTransition:true,overflow:await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)});
  assert.equal(evidence.renders.at(-1).overflow,false);
  await page.unroute(usersRoute);
 }

 assert.deepEqual(requests,[]);assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};await page.screenshot({path:out+'/failure.png'});throw e;}
finally{await writeFile(out+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
