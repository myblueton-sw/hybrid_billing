import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).reduce((a,v,i,all)=>v.startsWith('--')?[...a,[v.slice(2),all[i+1]]]:a,[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']||'playwright');
const browser=await chromium.launch({headless:true,...(args.chromium?{executablePath:args.chromium}:{})});
const page=await browser.newPage(),base=args.url||'http://127.0.0.1:8803/';
const out='../../output/validation/HB-76-menu-draft';await mkdir(out,{recursive:true});const errors=[],renders=[];
page.on('pageerror',e=>errors.push(e.message));
try{
 for(const lang of ['ko','en','ar'])for(const width of [1440,900,390]){
  await page.setViewportSize({width,height:1000});await page.goto(base+'#/home?lang='+lang+'&service=local');await page.locator('#content').waitFor();
  assert.equal(await page.locator('.planning-sidebar .service-business [data-section]').count(),6);
  assert.equal(await page.locator('.planning-sidebar .service-administration [data-section]').count(),2);
  assert.equal(await page.locator('.planning-sidebar [data-section=overview] > a').count(),1);
  assert.equal(await page.locator('.planning-search:visible').count(),0);
  assert.equal(await page.locator('.planning-sidebar a[href*="/onboarding?"]').count(),0);
  assert.equal(await page.locator('.planning-sidebar a[href*="/handoffs?"]').count(),0);
  const metrics=await page.evaluate(()=>({scroll:document.documentElement.scrollWidth,dir:document.documentElement.dir,logo:!!document.querySelector('.brand-logo')?.naturalWidth}));assert.ok(metrics.scroll<=width);assert.equal(metrics.dir,lang==='ar'?'rtl':'ltr');assert.ok(metrics.logo);
  await page.screenshot({path:out+'/'+lang+'-'+width+'.png',fullPage:true});if(lang==='ko'&&width===1440){for(const key of ['costs','settlement','billing']){const group=page.locator('.planning-sidebar [data-section='+key+']');if(!await group.evaluate(e=>e.open))await group.locator('summary').click();}await page.screenshot({path:out+'/ko-menu-preview.png',fullPage:false});}renders.push({lang,width});
  if(width===390){await page.locator('.planning-menu-button').click();const dialog=page.locator('.planning-menu-dialog');await dialog.waitFor();if(await dialog.locator('[data-section=settlement]').evaluate(e=>e.open))await dialog.locator('[data-section=settlement] summary').click();await dialog.locator('[data-section=settlement] summary').focus();await page.keyboard.press('Enter');await dialog.locator('a[href*="/submissions?"]').waitFor();await page.screenshot({path:out+'/'+lang+'-390-menu.png',fullPage:true});await page.keyboard.press('Escape');}
 }
 await page.setViewportSize({width:1440,height:1000});await page.goto(base+'#/connections?lang=en&service=local');await page.locator('.connection-scope').waitFor();await page.locator('.service-related summary').click();await page.locator('.service-related a[href*="/onboarding?"]').click();await page.locator('.onboarding-scope').waitFor();assert.ok(page.url().includes('service=local'));
 await page.goto(base+'#/invoices?lang=en&service=local');await page.locator('.service-related summary').click();assert.equal(await page.locator('.service-related a[href*="/handoffs?"]').count(),1);
 if(!await page.locator('.planning-sidebar [data-section=billing]').evaluate(e=>e.open))await page.locator('.planning-sidebar [data-section=billing] summary').click();const cases=page.locator('.planning-sidebar a[href*="view=cases"]');await cases.click();await page.waitForURL(u=>u.hash.includes('view=cases'));await page.waitForFunction(()=>document.querySelector('.planning-sidebar a[href*="view=cases"]')?.getAttribute('aria-current')==='page');assert.equal(await page.locator('.planning-sidebar a[aria-current]').count(),1);
 await page.locator('.planning-work > summary').click();assert.ok(await page.getByRole('button',{name:'My work and notifications',exact:true}).isVisible());assert.ok(await page.getByRole('button',{name:'Background jobs',exact:true}).isVisible());
 assert.deepEqual(errors,[]);await writeFile(out+'/report.json',JSON.stringify({status:'PASS',renders},null,2));console.log(JSON.stringify({status:'PASS',renders:renders.length}));
}finally{await browser.close();}
