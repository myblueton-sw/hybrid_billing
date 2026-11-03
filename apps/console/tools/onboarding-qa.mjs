import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).reduce((a,v,i,all)=>v.startsWith('--')?[...a,[v.slice(2),all[i+1]]]:a,[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']||'playwright');
const browser=await chromium.launch({headless:true,...(args.chromium?{executablePath:args.chromium}:{})});
const page=await browser.newPage(),base=args.url||'http://127.0.0.1:8801/',loader=args['loader-url']||'http://127.0.0.1:8802/';
const out='../../output/validation/HB-91';await mkdir(out,{recursive:true});const errors=[],renders=[];
page.on('pageerror',e=>errors.push(e.message));
const open=async(lang='en',stage='path',host=base)=>{await page.goto(host+'#/onboarding?lang='+lang+'&service=local&view='+stage);await page.locator('.onboarding-scope').waitFor();};
const selected=()=>page.locator('.onboarding-stages [aria-current="step"]');
try {
 for(const lang of ['en','ko','ar'])for(const width of [1440,390]){
  await page.setViewportSize({width,height:1000});
  for(const stage of ['path','permissions','first-data','reconciliation','activation']){
   await open(lang,stage);assert.ok((await selected().getAttribute('href')).includes('view='+stage));
   assert.equal(await page.locator('.onboarding-stages a').count(),5);
   const metrics=await page.evaluate(()=>({scroll:document.documentElement.scrollWidth,dir:document.documentElement.dir,font:getComputedStyle(document.querySelector('.onboarding-live')).fontSize,logo:!!document.querySelector('.brand-logo')?.naturalWidth,borders:[...document.querySelectorAll('.onboarding-stages a,.onboarding-scope,.onboarding-footer')].map(e=>getComputedStyle(e).borderBottomColor)}));
   assert.ok(metrics.scroll<=width,JSON.stringify(metrics));assert.equal(metrics.dir,lang==='ar'?'rtl':'ltr');assert.equal(metrics.font,'16px');assert.ok(metrics.logo);
   for(const color of metrics.borders){const rgb=color.match(/\d+/g)?.map(Number);if(rgb&&rgb.length===3)assert.ok(Math.max(...rgb)-Math.min(...rgb)<30,color);}
   assert.ok(await page.locator('.live-mode-switch').evaluate(e=>e.getBoundingClientRect().height<80));
   await page.locator('#onboarding-stage-title').evaluate(e=>e.blur());
   await page.screenshot({path:out+'/'+lang+'-'+width+'-'+stage+'.png',fullPage:true});renders.push({lang,width,stage});
  }
 }
 await open();await page.locator('.onboarding-footer a').click();await page.locator('.onboarding-scope').waitFor();assert.ok(page.url().includes('view=permissions'));await page.goBack();await page.locator('.onboarding-scope').waitFor();assert.ok(page.url().includes('view=path'));
 await page.locator('#onboarding-refresh').focus();await Promise.all([page.waitForResponse(r=>r.url().includes('/api/local-connections')),page.keyboard.press('Enter')]);await page.waitForFunction(()=>document.querySelector('#onboarding-refresh')?.disabled===false);await page.locator('.onboarding-scope').waitFor();
 await open('en','first-data');await page.locator('[data-action="source-data"]').click();await page.locator('#source-page-title').waitFor();
 await open('en','reconciliation');await page.locator('[data-action="reconciliation"]').click();await page.locator('#rec-detail').waitFor();
 await open('en','first-data',loader);assert.equal(await page.locator('[data-action="source-data"]').count(),0);await page.locator('[data-action="imports"]').click();await page.locator('.live-imports .planning-panel dl').filter({hasText:decodeURIComponent(new URL(page.url()).hash.split('?')[0].split('/').at(-1))}).waitFor();
 await open('en','reconciliation',loader);assert.equal(await page.locator('[data-action="reconciliation"]').count(),0);
 for(const mode of ['denied','malformed','empty']){
  await open('en','first-data');
  await page.route('**/api/local-connections**',async route=>{if(mode==='denied')await route.fulfill({status:403,json:{error:'scope_denied'}});else{const response=await route.fetch(),body=await response.json();if(mode==='malformed')body.observed_at='invalid';else body.latest_candidate=null;await route.fulfill({response,json:body});}});
  await page.locator('#onboarding-refresh').click();
  if(mode==='empty')await page.getByText('No registered candidate. The cause of missing data is unknown.',{exact:true}).waitFor();
  else{await page.locator('.onboarding-live [role="alert"]').waitFor();assert.equal(await page.locator('.onboarding-scope,.onboarding-stages,.onboarding-stage').count(),0);}
  await page.unroute('**/api/local-connections**');await page.locator('#onboarding-refresh').click();await page.locator('.onboarding-scope').waitFor();
 }
 await page.goto(base+'#/onboarding?service=local&lang=en&connection=00000000-0000-0000-0000-000000000001');await page.locator('.onboarding-live [role="alert"]').waitFor();assert.equal(await page.locator('.onboarding-scope').count(),0);
 for(const identity of ['', 'invalid']){
  await page.goto(base+'#/onboarding?service=local&lang=en&connection='+identity);await page.locator('.onboarding-live [role="alert"]').waitFor();assert.equal(await page.locator('.onboarding-scope').count(),0);
 }
 await open();let release,started;const gate=new Promise(r=>release=r),seen=new Promise(r=>started=r);
 await page.route('**/api/local-connections',async route=>{const response=await route.fetch();started();await gate;try{await route.fulfill({response});}catch{}});
 await page.locator('#onboarding-refresh').click();await seen;
 await page.goto(base+'#/onboarding?service=local&lang=en&connection=invalid');await page.locator('.onboarding-live [role="alert"]').waitFor();release();await page.unroute('**/api/local-connections');assert.equal(await page.locator('.onboarding-scope').count(),0);
 assert.deepEqual(errors,[]);await writeFile(out+'/report.json',JSON.stringify({status:'PASS',renders,checks:['5 stages','history','keyboard refresh','role-specific exact navigation','denied/malformed clearing and retry','empty candidate','foreign identity']},null,2));console.log(JSON.stringify({status:'PASS',renders:renders.length}));
}finally{await browser.close();}
