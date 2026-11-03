import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).reduce((a,v,i,all)=>v.startsWith('--')?[...a,[v.slice(2),all[i+1]]]:a,[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']||'playwright');
const browser=await chromium.launch({headless:true,...(args.chromium?{executablePath:args.chromium}:{})});
const page=await browser.newPage(),base=args.url||'http://127.0.0.1:8803/';
const out='../../output/validation/HB-76';await mkdir(out,{recursive:true});const errors=[],renders=[];
page.on('pageerror',e=>errors.push(e.message));
try {
 for(const lang of ['ko','en','ar'])for(const width of [1440,900,390])for(const route of ['home','connections']){
  await page.setViewportSize({width,height:1000});await page.goto(base+'#/'+route+'?lang='+lang+'&service=local');
  await page.locator('#main h1').waitFor();if(route==='connections')await page.locator('.connection-scope').waitFor();
  for(const selector of ['.planning-search','.planning-preview','.planning-review-tools','.service-navigation small','.planning-legacy-map'])assert.equal(await page.locator(selector+':visible').count(),0);
  assert.equal(await page.getByText('업무 홈',{exact:true}).count(),0);
  assert.equal(await page.locator('.planning-sidebar .service-business [data-section]').count(),8);assert.equal(await page.locator('.planning-sidebar .service-administration [data-section]').count(),1);assert.equal(await page.locator('.planning-sidebar a[href*="/assistant?"]').count(),0);
  const metrics=await page.evaluate(()=>({scroll:document.documentElement.scrollWidth,dir:document.documentElement.dir,logo:!!document.querySelector('.brand-logo')?.naturalWidth}));
  assert.ok(metrics.scroll<=width,JSON.stringify({lang,width,route,...metrics}));assert.ok(metrics.logo);assert.equal(metrics.dir,lang==='ar'?'rtl':'ltr');assert.equal(await page.locator('#confirm-review:visible').count(),1);
  if(width===1440){const layout=await page.evaluate(()=>({side:document.querySelector('.planning-sidebar').getBoundingClientRect().height,main:document.querySelector('.planning-main').getBoundingClientRect().height,sticky:getComputedStyle(document.querySelector('.service-navigation')).position}));assert.ok(Math.abs(layout.side-layout.main)<2);assert.equal(layout.sticky,'sticky');}
  assert.equal(await page.locator('.planning-sidebar [data-section="data"] a').count(),5);assert.equal(await page.locator('.planning-sidebar a').count(),41);
  await page.screenshot({path:out+'/'+lang+'-'+width+'-'+route+'.png',fullPage:true});renders.push({lang,width,route});
 }
 await page.setViewportSize({width:1440,height:1000});await page.goto(base+'#/connections?lang=en&service=local');await page.locator('.connection-scope').waitFor();
 await page.locator('.planning-sidebar a[href*="/onboarding?"]').click();await page.locator('.onboarding-scope').waitFor();assert.ok(page.url().includes('service=local'));
 await page.goto(base+'#/design/screens?lang=en');await page.getByRole('heading',{name:'Billing overview',exact:true}).waitFor();assert.equal(await page.locator('.planning-directory').count(),0);
 await page.goto(base+'#/design/screens?lang=en&review=1');await page.locator('.planning-directory').waitFor();
 await page.goto(base+'#/connections?lang=en&service=local&review=1');await page.locator('.connection-scope').waitFor();await page.locator('.planning-review-tools > summary').click();await page.getByLabel('Example persona · grants nothing').selectOption('OP');
 await page.locator('.planning-user > summary').click();await page.getByRole('button',{name:'Exit developer mode',exact:true}).click();await page.waitForURL(url=>!url.hash.includes('review='));await page.waitForFunction(()=>document.querySelectorAll('.planning-search').length===0);await page.locator('.connection-scope').waitFor();assert.equal(await page.locator('.planning-search:visible').count(),0);
 await page.locator('.planning-sidebar details').filter({hasText:'Organizations and contracts'}).locator('summary').click();assert.ok(await page.locator('.planning-sidebar a[href*="/organizations?"]').isVisible());
 await page.setViewportSize({width:390,height:1000});await page.locator('.planning-menu-button').click();await page.locator('.planning-menu-dialog').waitFor();assert.ok(await page.locator('.planning-menu-dialog .header-select').count() || await page.locator('.planning-menu-dialog').getByRole('button',{name:/한국어|English/}).count());await page.keyboard.press('Escape');assert.equal(await page.locator('.planning-menu-dialog:visible').count(),0);
 await page.locator('.planning-menu-button').click();assert.ok(await page.locator('.planning-menu-dialog').getByRole('button',{name:'AI Assistant',exact:true}).isVisible());assert.ok(await page.locator('.planning-menu-dialog').getByRole('button',{name:'Notifications',exact:true}).isVisible());await page.locator('.planning-menu-dialog [data-section="accounting"] summary').focus();await page.keyboard.press('Enter');await page.locator('.planning-menu-dialog a[href*="/submissions?"]').waitFor();await page.screenshot({path:out+'/en-390-menu.png',fullPage:true});await page.keyboard.press('Escape');
 await page.setViewportSize({width:1440,height:1000});await page.goto(base+'#/home?lang=en&review=1');await page.locator('.planning-review-tools > summary').click();await page.getByLabel('Audience preview').selectOption('unauthenticated');await page.getByRole('button',{name:'Return to service',exact:true}).click();await page.getByRole('heading',{name:'Billing overview',exact:true}).waitFor();assert.ok(!page.url().includes('review=1'));
 assert.deepEqual(errors,[]);await writeFile(out+'/report.json',JSON.stringify({status:'PASS',renders},null,2));console.log(JSON.stringify({status:'PASS',renders:renders.length}));
}finally{await browser.close();}
