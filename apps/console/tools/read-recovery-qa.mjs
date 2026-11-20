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


 const expected=(lang,count)=>lang==='en'?`List updated. ${count}. Filters applied: 0.`:lang==='ko'?`목록을 새로 고쳤습니다. ${count}. 적용된 필터 0개.`:`تم تحديث القائمة. ${count}. عوامل التصفية المطبّقة: 0.`;
 async function tabTo(selector){for(let i=0;i<100;i++){if(await page.locator(selector).evaluate(e=>e===document.activeElement))return;await page.keyboard.press('Tab');}throw new Error('Keyboard target not reached');}
 async function observeStatus(){await page.evaluate(()=>{window.__hbObserver?.disconnect();window.__hbStatusChanges=[];const region=document.querySelector('.m01-live > [role=status]');window.__hbObserver=new MutationObserver(()=>window.__hbStatusChanges.push(region.textContent));window.__hbObserver.observe(region,{childList:true,subtree:true,characterData:true});});}
 async function noLateAnnouncement(before){await page.evaluate(()=>new Promise(done=>setTimeout(done,150)));assert.equal(await page.evaluate(()=>window.__hbStatusChanges.length),before);}
 const requests=[];page.on('request',r=>{if(r.method()==='POST'&&new URL(r.url()).pathname.includes('/commands/'))requests.push(new URL(r.url()).pathname);});
 if(!args.lifecycle)for(const lang of ['en','ko','ar'])for(const width of [1440,390]){
  await page.setViewportSize({width,height:1000});await page.goto('about:blank');
  const usersRoute=u=>u.pathname==='/api/v1/identity/users';
  let mode='fail',release,started;
  const hold=()=>{const entered=new Promise(resolve=>started=resolve),held=new Promise(resolve=>release=resolve);return{entered,held};};
  let gate=hold();
  await page.route(usersRoute,async route=>{
   if(mode==='fail'){await route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({code:'service_unavailable'})});return;}
   started();await gate.held;
   if(mode==='fail-after-hold'){await route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({code:'service_unavailable'})});return;}
   const u=new URL(route.request().url());const response=await route.fetch({url:api+u.pathname+u.search});await route.fulfill({response});
  });
  await page.goto(`${base}/?service=api&lang=${lang}#/users?lang=${lang}&view=users`);
  await page.locator('.live-error button').waitFor();await tabTo('.live-error button');
  await page.locator('.live-error button').evaluate(e=>e.scrollIntoView({block:'center'}));
  const borders=await page.locator('.live-error, .live-error button, #m01-reload').evaluateAll(elements=>elements.map(e=>{const c=getComputedStyle(e);return{className:e.className,borders:[c.borderTop,c.borderRight,c.borderBottom,c.borderLeft],outline:c.outline};}));
  await page.screenshot({path:out+`/error-${lang}-${width}.png`});
  mode='fail-after-hold';await page.keyboard.press('Enter');await gate.entered;
  await page.locator('#m01-reload:focus').waitFor();
  assert.equal(await page.locator('.live-error').count(),0);release();await page.locator('.live-error button').waitFor();
  assert.ok(await page.locator('#m01-reload').evaluate(e=>e===document.activeElement));
  assert.equal(await page.locator('.m01-live > [role=status]').innerText(),'');
  await tabTo('.live-error button');gate=hold();mode='success';await page.keyboard.press('Enter');await gate.entered;
  await page.locator('#m01-reload:focus').waitFor();
  const loading=await page.evaluate(()=>({active:document.activeElement.id,status:[...document.querySelectorAll('[role=status]')].map(e=>e.textContent).filter(Boolean)}));
  release();await page.locator('.live-open').first().waitFor();
  const count=await page.locator('.m01-count').innerText(),announcement=expected(lang,count);
  await page.waitForFunction(text=>document.querySelector('.m01-live > [role=status]')?.textContent===text,announcement);
  assert.ok(await page.locator('#m01-reload').evaluate(e=>e===document.activeElement));
  await page.screenshot({path:out+`/recovered-${lang}-${width}.png`});
  await observeStatus();gate=hold();await page.keyboard.press('Enter');await gate.entered;
  await page.keyboard.press('Tab');const moved=await page.evaluate(()=>({tag:document.activeElement.tagName,id:document.activeElement.id,className:document.activeElement.className}));
  assert.notEqual(moved.id,'m01-reload');release();
  await page.waitForFunction(()=>!document.querySelector('[aria-busy=true]'));
  await page.waitForFunction(text=>document.querySelector('.m01-live > [role=status]')?.textContent===text,announcement);
  await page.waitForFunction(text=>{const changes=window.__hbStatusChanges;return changes.includes('')&&changes.at(-1)===text;},announcement);
  assert.deepEqual(await page.evaluate(()=>({tag:document.activeElement.tagName,id:document.activeElement.id,className:document.activeElement.className})),moved);
  evidence.renders.push({lang,width,loading,announcement,borders,repeatedFailure:true,focusPreservedAfterMovement:true,freshAnnouncementTransition:true,overflow:await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth)});
  assert.equal(evidence.renders.at(-1).overflow,false);
  await page.unroute(usersRoute);
 }

 if(args.lifecycle){
  await page.setViewportSize({width:1440,height:1000});await page.goto('about:blank');
  await page.goto(`${base}/?service=api&lang=en#/users?lang=en&view=users`);await page.locator('.live-open').first().waitFor();
  assert.equal(await page.locator('.m01-live > [role=status]').innerText(),'');
  const usersRoute=u=>u.pathname==='/api/v1/identity/users';
  const held=[];
  await page.route(usersRoute,async route=>{
   const u=new URL(route.request().url());
   if(u.searchParams.get('q')){const response=await route.fetch({url:api+u.pathname+u.search});await route.fulfill({response});return;}
   let release,finished;const gate=new Promise(resolve=>release=resolve),done=new Promise(resolve=>finished=resolve);
   held.push({release,done});await gate;const response=await route.fetch({url:api+u.pathname+u.search});await route.fulfill({response});finished();
  });
  async function waitHeld(n){const until=Date.now()+10000;while(held.length<n){assert.ok(Date.now()<until,'Expected held read');await page.evaluate(()=>new Promise(done=>requestAnimationFrame(done)));}}
  await observeStatus();await tabTo('#m01-reload');await page.keyboard.press('Enter');await waitHeld(1);await page.keyboard.press('Enter');await waitHeld(2);
  held[1].release();await held[1].done;
  const count=await page.locator('.m01-count').innerText(),announcement=expected('en',count);
  await page.waitForFunction(text=>document.querySelector('.m01-live > [role=status]')?.textContent===text,announcement);
  const beforeOld=await page.evaluate(()=>window.__hbStatusChanges.length);held[0].release();await held[0].done;await page.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));await noLateAnnouncement(beforeOld);
  assert.equal(await page.locator('.m01-live > [role=status]').innerText(),announcement);
  await page.keyboard.press('Enter');await waitHeld(3);
  await tabTo('#m01-q');await page.keyboard.type('absent-accessibility-fixture@example.invalid');await page.keyboard.press('Enter');
  await page.locator('.m01-empty').waitFor();
  await page.waitForFunction(()=>document.querySelector('.m01-live > [role=status]')?.textContent?.endsWith('Filters applied: 1.'));
  const filtered=await page.locator('.m01-live > [role=status]').innerText();
  const beforeFilteredOld=await page.evaluate(()=>window.__hbStatusChanges.length);held[2].release();await held[2].done;await page.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));await noLateAnnouncement(beforeFilteredOld);
  assert.equal(await page.locator('.live-open').count(),0);assert.equal(await page.locator('.m01-live > [role=status]').innerText(),filtered);
  evidence.lifecycle={backgroundReadSilent:true,repeatedReloadLatestFirst:true,filterSupersedesReload:true,lateResponseIgnored:true,lateAnnouncementObservationMs:150,emptyResultAnnounced:filtered};
  await page.unroute(usersRoute);
 }
 assert.deepEqual(requests,[]);assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};await page.screenshot({path:out+'/failure.png'});throw e;}
finally{await writeFile(out+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
