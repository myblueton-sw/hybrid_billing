import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).reduce((a,v,i,all)=>v.startsWith('--')?[...a,[v.slice(2),all[i+1]]]:a,[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']||'playwright');
const browser=await chromium.launch({headless:true,...(args.chromium?{executablePath:args.chromium}:{})});
const page=await browser.newPage(),base=args.url||'http://127.0.0.1:8799/',loader=args['loader-url']||'http://127.0.0.1:8800/';
const out='../../output/validation/HB-90';await mkdir(out,{recursive:true});const errors=[],renders=[];
page.on('pageerror',e=>errors.push(e.message));
const open=async(lang='en',host=base)=>{await page.goto(host+'#/connections?lang='+lang+'&service=local');await page.locator('.connection-scope').waitFor();};
try {
 for(const lang of ['en','ko','ar'])for(const width of [1440,390]){
  await page.setViewportSize({width,height:1000});await open(lang);
  assert.equal(await page.locator('.connection-contracts article').count(),2);
  assert.deepEqual(await page.locator('.connection-actions a').evaluateAll(es=>es.map(e=>e.dataset.action)),['source-data','reconciliation']);
  const exact=await page.locator('.connection-scope a').getAttribute('href');await page.locator('.connection-scope a').click();await page.locator('.connection-scope').waitFor();assert.ok(page.url().endsWith(exact));
  await page.locator('#connection-refresh').focus();await Promise.all([page.waitForResponse(r=>r.url().includes('/api/local-connections')),page.keyboard.press('Enter')]);await page.waitForFunction(()=>document.querySelector('#connection-refresh')?.disabled===false);await page.locator('.connection-scope').waitFor();
  const metrics=await page.evaluate(()=>({width:innerWidth,scroll:document.documentElement.scrollWidth,dir:document.documentElement.dir,logo:!!document.querySelector('.brand-logo')?.naturalWidth,font:getComputedStyle(document.querySelector('#confirm-review')).fontSize}));
  assert.ok(metrics.scroll<=width,JSON.stringify(metrics));assert.equal(metrics.dir,lang==='ar'?'rtl':'ltr');assert.ok(metrics.logo);assert.equal(metrics.font,'16px');
  await page.screenshot({path:`${out}/${lang}-${width}.png`,fullPage:true});renders.push({lang,...metrics});
  if(lang==='ar'&&width===390)await page.locator('.connection-scope').screenshot({path:`${out}/ar-scope.png`});
 }
 await open();await page.locator('[data-action="source-data"]').click();await page.locator('#source-page-title').waitFor();await page.goBack();await page.locator('.connection-scope').waitFor();await page.locator('[data-action="reconciliation"]').click();await page.locator('#rec-detail').waitFor();
 await open('en',loader);assert.deepEqual(await page.locator('.connection-actions a').evaluateAll(es=>es.map(e=>e.dataset.action)),['imports','new-import']);await page.locator('[data-action="imports"]').click();await page.locator('.live-scope').waitFor();await page.locator('.live-imports .planning-panel dl').filter({hasText:decodeURIComponent(new URL(page.url()).hash.split('?')[0].split('/').at(-1))}).waitFor();assert.equal(await page.locator('.live-imports [role="alert"]').count(),0);assert.ok(page.url().includes('#/imports/'));assert.equal(await page.locator('.source-data').count(),0);
 await open();
 for(const mode of ['denied','malformed','policy','empty']){
  await page.route('**/api/local-connections**',async route=>{if(mode==='malformed'||mode==='empty'){const response=await route.fetch();const body=await response.json();if(mode==='malformed')delete body.actions;else body.latest_candidate=null;await route.fulfill({response,json:body});}else await route.fulfill({status:mode==='denied'?403:409,contentType:'application/json',body:JSON.stringify({error:mode==='denied'?'scope_denied':'policy_changed'})});});
  await page.locator('#connection-refresh').click();
  if(mode==='empty'){await page.getByText('No registered input candidate. This does not imply that no quarantined attempt exists.',{exact:true}).waitFor();assert.ok((await page.locator('[data-action="source-data"]').getAttribute('href')).startsWith('#/source-data?'));}else{await page.locator('.connections-screen [role="alert"]').waitFor();assert.equal(await page.locator('.connection-scope').count(),0);assert.equal(await page.locator('.connection-actions a').count(),0);}
  await page.unroute('**/api/local-connections**');await page.locator('#connection-refresh').click();await page.locator('.connection-scope').waitFor();
 }
 let release,started;const gate=new Promise(r=>release=r),seen=new Promise(r=>started=r);
 await page.route('**/api/local-connections',async route=>{const response=await route.fetch();started();await gate;try{await route.fulfill({response});}catch{}});
 await page.locator('#connection-refresh').click();await seen;
 await page.goto(base+'#/connections/00000000-0000-0000-0000-000000000001?lang=en&service=local');await page.locator('.connections-screen [role="alert"]').waitFor();release();assert.equal(await page.locator('.connection-scope').count(),0);await page.unroute('**/api/local-connections');
 await page.goto(base+'#/connections/invalid?lang=en&service=local');await page.locator('.connections-screen [role="alert"]').waitFor();
 assert.deepEqual(errors,[]);await writeFile(`${out}/report.json`,JSON.stringify({status:'PASS',renders,checks:['actual metadata','exactID','keyboard refresh','role links','source/reconciliation navigation','denied/policy/malformed clearing and retry','emptycandidate','late response discard','foreign/invalid connection']},null,2));console.log(JSON.stringify({status:'PASS',renders:renders.length}));
}finally{await browser.close();}
