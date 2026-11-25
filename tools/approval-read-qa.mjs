import {createRequire} from 'node:module';
import assert from 'node:assert/strict';
import {readFile,mkdir,writeFile} from 'node:fs/promises';
import {randomBytes} from 'node:crypto';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']||'playwright');
assert.ok(args.url&&args.api&&args.output&&args.activation);
await mkdir(args.output,{recursive:true});
const browser=await chromium.launch({headless:true,...(args.chromium?{executablePath:args.chromium}:{})});
const page=await browser.newPage({viewport:{width:1440,height:1000}}),password=randomBytes(24).toString('base64url');
const evidence={outcome:'running',cases:[],errors:[],commands:[]};
page.on('request',r=>{if(r.method()==='POST'&&new URL(r.url()).pathname.includes('/commands/'))evidence.commands.push(new URL(r.url()).pathname);});
page.on('pageerror',e=>evidence.errors.push(e.message));
await page.route('**/api/**',async route=>{const u=new URL(route.request().url());const response=await route.fetch({url:args.api+u.pathname+u.search});await route.fulfill({response});});
try {
 const lines=(await readFile(args.activation,'utf8')).trim().split('\n');
 const link=new URL(lines.find(l=>l.startsWith('admin@example.invalid ')).split(' ')[1]);
 await page.goto(args.url+'/activate?lang=en'+link.hash);await page.locator('#link-new').fill(password);await page.locator('#link-confirm').fill(password);await page.locator('form button[type=submit]').click();await page.locator('a.service-button-link').waitFor();
 await page.goto(args.url+'/?service=api&lang=en');await page.locator('#signin-workspace').fill('sample-workspace');await page.locator('#signin-email').fill('admin@example.invalid');await page.locator('#signin-password').fill(password);await page.locator('form button[type=submit]').click();await page.locator('.service-session').waitFor();
 for(const lang of (args.journey?[]:args.lang?[args.lang]:['en','ko','ar']))for(const width of (args.width?[Number(args.width)]:[1440,1024,390])) {
  await page.setViewportSize({width,height:1000});
  await page.goto("about:blank");
  await page.goto(`${args.url}/?service=api&lang=${lang}#/approvals?lang=${lang}&view=pending`);
  await page.locator('[data-b09-record]:visible').first().waitFor();
  assert.equal(await page.locator('[data-b09-record]:visible').count(),25);
  await page.evaluate(()=>document.fonts.ready);
  await page.screenshot({path:`${args.output}/inbox-${lang}-${width}.png`});
  await page.locator('[data-b09-record]:visible').first().focus();await page.keyboard.press('Enter');
  await page.waitForFunction(()=>document.activeElement?.matches('[data-b09-detail-heading]'));
  assert.ok((await page.locator('body').innerText()).includes('1234.56789012'));
  await page.screenshot({path:`${args.output}/detail-${lang}-${width}.png`});
  for(const section of ['.b09-amounts','.b09-blockers']) {
   await page.locator(section+':visible').first().scrollIntoViewIfNeeded();
   await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));await page.waitForTimeout(100);
   await page.screenshot({path:`${args.output}/${section.slice(1)}-${lang}-${width}.png`});
  }
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth);
  const borders=await page.locator('.b09-request-list button.selected,.b09-synthetic,.b09-blockers').evaluateAll(elements=>elements.map(e=>{const c=getComputedStyle(e);return{className:e.className,borders:[c.borderTop,c.borderRight,c.borderBottom,c.borderLeft],outline:c.outline};}));
  evidence.cases.push({lang,width,records:25,exactAmount:true,detailFocus:true,overflow,borders});assert.equal(overflow,false);
 }
 if(args.journey) {
  await page.goto(args.url+'/?service=api&lang=en#/approvals?lang=en&view=pending');
  await page.locator('[data-b09-record]:visible').first().waitFor();
  await page.locator('#b09-next').click();
  await page.waitForFunction(()=>document.querySelector('.b09-screen')?.dataset.b09State==='loaded'&&document.querySelector('.b09-page-status')?.textContent.includes('This page 4'));
  assert.equal(await page.locator('[data-b09-record]:visible').count(),4);
  await page.locator('#b09-first').click();await page.waitForFunction(()=>document.querySelector('.b09-page-status')?.textContent.includes('This page 25'));
  let mode='success',release,entered,finished;
  let delivery;
  function hold(){const started=new Promise(resolve=>entered=resolve);delivery=new Promise(resolve=>finished=resolve);mode='hold';return started;}
  await page.route('**/api/approval-review?*',async route=>{
   const captured=mode,u=new URL(route.request().url());
   if(captured==='fail'||captured==='denied'){await route.fulfill({status:captured==='fail'?503:403,contentType:'application/json',body:JSON.stringify({error:captured==='fail'?'service_unavailable':'permission_denied'})});return;}
   const response=await route.fetch({url:args.api+u.pathname+u.search});
   if(captured==='hold'){entered();await new Promise(resolve=>release=resolve);}
   try {await route.fulfill({response});if(captured==='hold')finished('fulfilled');}
   catch(e){if(captured==='hold')finished('cancelled');else throw e;}
  });
  async function search(value){await page.locator('#b09-search').fill(value);await page.locator('.b09-toolbar button[type=submit]').click();}
  mode='fail';await page.locator('[data-b09-record]:visible').first().click();await search('');await page.locator('.b09-error').waitFor();assert.equal(await page.locator('[data-b09-record]').count(),0);
  mode='success';await page.locator('#b09-retry').click();await page.locator('[data-b09-record]:visible').first().waitFor();
  await search('absent');await page.locator('.b09-empty').waitFor();
  const held=hold();await search('sample-approval-001');await held;
  mode='success';await search('sample-approval-002');await page.locator('[data-b09-record="sample-approval-002"]:visible').waitFor();release();const filterDelivery=await delivery;await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
  assert.equal(await page.locator('[data-b09-record="sample-approval-001"]:visible').count(),0);
  assert.equal(await page.locator('[data-b09-record="sample-approval-002"]:visible').count(),1);
  mode='denied';await search('');await page.locator('.b09-error').waitFor();assert.equal(await page.locator('[data-b09-record]').count(),0);assert.equal(await page.locator('#b09-retry').count(),0);
  mode='success';await page.locator('#b09-new-query').click();await page.locator('[data-b09-record]:visible').first().waitFor();
  const oldRead=hold();await search('sample-approval-001');await oldRead;
  await page.locator('.service-session button').click();await page.locator('#signin-email').waitFor();
  mode='success';
  const secondLink=new URL(lines.find(l=>l.startsWith('admin2@example.invalid ')).split(' ')[1]);
  await page.goto(args.url+'/activate?lang=en'+secondLink.hash);await page.locator('#link-new').fill(password);await page.locator('#link-confirm').fill(password);await page.locator('form button[type=submit]').click();await page.locator('a.service-button-link').waitFor();
  await page.goto(args.url+'/?service=api&lang=en');await page.locator('#signin-workspace').fill('sample-workspace');await page.locator('#signin-email').fill('admin2@example.invalid');await page.locator('#signin-password').fill(password);await page.locator('form button[type=submit]').click();await page.locator('.service-session').waitFor();
  await page.goto(args.url+'/?service=api&lang=en#/approvals?lang=en&view=decisions');await page.locator('[data-b09-record="sample-approval-030"]:visible').waitFor();release();const navigationDelivery=await delivery;await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
  assert.ok((await page.locator('.b09-page-status').innerText()).includes('sample-admin-2'));assert.equal(await page.locator('[data-b09-record="sample-approval-001"]:visible').count(),0);
  assert.equal(await page.locator('[data-b09-record="sample-approval-030"]:visible').count(),1);
  evidence.journey={filterDelivery,navigationDelivery,navigationDestroysOldDocument:true,paging:true,failedReadClearsRows:true,retry:true,empty:true,lateReadIgnored:true,deniedClearsRows:true,actualSignOutAndSecondActor:true};
 }
 assert.deepEqual(evidence.errors,[]);assert.deepEqual(evidence.commands,[]);evidence.outcome='passed';
} catch(e) {evidence.outcome='failed';evidence.failure={kind:e.name};await page.screenshot({path:args.output+'/failure.png'});throw new Error('Approval read QA failed: '+e.name);}
finally {await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
