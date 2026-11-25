import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const cookie=process.env.HB_PROJECTION_QA_COOKIE;
delete process.env.HB_PROJECTION_QA_COOKIE;delete process.env.HB_PROJECTION_QA_PASSWORD;
assert.ok(cookie&&args.url&&args.api&&args.output&&args.chromium);
await mkdir(args.output,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const context=await browser.newContext({viewport:{width:1440,height:1000}});
const split=cookie.indexOf('=');
await context.addCookies([{name:cookie.slice(0,split),value:cookie.slice(split+1),url:args.url,httpOnly:true,sameSite:'Strict'}]);
const page=await context.newPage(),evidence={cases:[],errors:[],refusals:[],mode:'actual synthetic server; first mutation response deliberately lost'};
page.on('pageerror',e=>evidence.errors.push(e.message));page.on('dialog',d=>d.accept());
let lose=true,release,previewMode="normal",releasePreview;const bodies=[];
await page.route('**/api/**',async route=>{
 const u=new URL(route.request().url());
 const response=await route.fetch({url:args.api+u.pathname+u.search});
 if(u.pathname.endsWith('/erp_final.preview')&&previewMode==='hold')await new Promise(resolve=>releasePreview=resolve);
 if(response.status()>=400){const body=await response.json();evidence.refusals.push({path:u.pathname,status:response.status(),code:body.code});}
 if(u.pathname.endsWith('/erp_final.request')){
  bodies.push(route.request().postData());
  if(lose){await new Promise(resolve=>release=resolve);await route.abort('failed');return;}
 }
 await route.fulfill({response});
});
const panel=()=>page.locator('[data-final-request]');
async function capture(state,lang,width,selector='[data-final-request]'){
 const element=page.locator(selector);await element.scrollIntoViewIfNeeded();
 await element.screenshot({path:`${args.output}/${state}-${lang}-${width}.png`});
 const check=await element.evaluate(root=>{
  const allowed=new Set(),probe=document.createElement('span');root.append(probe);
  for(const token of ['--border-section','--border-row','--border-control']){probe.style.borderColor=`var(${token})`;allowed.add(getComputedStyle(probe).borderTopColor);}probe.remove();
  const unexpected=[];for(const e of [root,...root.querySelectorAll('*')]){const c=getComputedStyle(e);for(const side of ['Top','Right','Bottom','Left'])if(parseFloat(c[`border${side}Width`])>0&&!allowed.has(c[`border${side}Color`]))unexpected.push(e.tagName);}
  return{overflow:document.documentElement.scrollWidth>innerWidth,direction:getComputedStyle(root).direction,unexpected};
 });
 assert.deepEqual(check.unexpected,[]);
 if(state==='preview')await page.screenshot({path:`${args.output}/page-${state}-${lang}-${width}.png`,fullPage:true});
 assert.equal(check.overflow,false);assert.equal(check.direction,lang==='ar'?'rtl':'ltr');evidence.cases.push({state,lang,width,...check});
}
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await page.setViewportSize({width,height:1000});await page.goto('about:blank');
  await page.goto(`${args.url}/?service=api&lang=${lang}#/approvals?lang=${lang}&view=results`);
  await page.locator('.b09-results-list button').filter({hasText:'result-preparation'}).click();
  await page.locator('.b09-results-detail button').filter({hasText:'public-result'}).click();await panel().waitFor();
  await capture('empty',lang,width);
  await page.locator('[data-final-request-check]').focus();await page.keyboard.press('Enter');await page.locator('[data-final-request-error]').waitFor();
  await page.waitForFunction(()=>document.activeElement?.matches('[data-final-request-error]'));
  await capture('validation',lang,width);
  await page.locator('#final-request-reference').fill('qa-browser-final');await page.locator('#final-request-rationale').fill('Reviewed synthetic final approval request');
  await page.locator('[data-final-request-check]').click();await page.locator('[data-final-request-preview]').waitFor();
  assert.ok((await panel().innerText()).includes('USD 16'));assert.equal(await page.locator('[data-final-request-send]').isEnabled(),true);
  await page.waitForFunction(()=>document.activeElement?.matches('[data-final-request-preview] h4'));
  await page.locator('[data-final-request-send]').focus();await page.keyboard.press('Tab');await page.keyboard.press('Shift+Tab');
  await page.waitForFunction(()=>document.activeElement?.matches('[data-final-request-send]'));
  const focus=await page.locator('[data-final-request-send]').evaluate(e=>{const c=getComputedStyle(e);return {outline:c.outlineStyle,width:c.outlineWidth,font:getComputedStyle(document.querySelector('.b09-final-summary .b09-money')).fontFamily};});
  assert.notEqual(focus.outline,'none');assert.ok(parseFloat(focus.width)>0);assert.ok(focus.font.includes('Plex'));
  await capture('preview',lang,width);
 }
 await panel().locator('button').last().click();
 previewMode='hold';await page.locator('[data-final-request-check]').click();
 await page.waitForFunction(()=>document.querySelector('[data-final-request][aria-busy="true"]'));
 while(!releasePreview)await new Promise(resolve=>setTimeout(resolve,10));
 await page.locator('.b09-results > header button').click();
 previewMode='normal';releasePreview();
 await page.locator('.b09-results-list button').filter({hasText:'result-preparation'}).waitFor();
 assert.equal(await page.locator('[data-final-request-preview]').count(),0);
 await page.locator('.b09-results-list button').filter({hasText:'result-preparation'}).click();
 await page.locator('.b09-results-detail button').filter({hasText:'public-result'}).click();
 assert.equal(await page.locator('#final-request-reference').inputValue(),'');
 await page.locator('#final-request-reference').fill('qa-browser-final');await page.locator('#final-request-rationale').fill('Reviewed synthetic final approval request');
 await page.locator('[data-final-request-check]').click();await page.locator('[data-final-request-preview]').waitFor();
 evidence.cases.push({state:'late-preview-discarded-after-new-read'});
 await page.locator('[data-final-request-send]').click();await page.locator('[data-final-request-outcome="pending"]').waitFor();
 assert.equal(await page.locator('#final-request-reference').isDisabled(),true);
 await page.locator('.b09-results-detail button').filter({hasText:'public-result'}).click();
 assert.equal(await page.locator('[data-final-request-outcome="pending"]').count(),1);
 assert.equal(bodies.length<=1,true);
 await page.waitForFunction(()=>document.querySelector('[data-final-request-outcome="pending"]'));
 while(!release)await new Promise(resolve=>setTimeout(resolve,10));
 await capture('pending','ar',390,'[data-final-request-outcome]');lose=false;release();
 await page.locator('[data-final-request-outcome="unknown"]').waitFor();await capture('unknown','ar',390,'[data-final-request-outcome]');
 await page.locator('[data-final-request-retry]').focus();await page.keyboard.press('Enter');await page.locator('[data-final-request-outcome="recorded"]').waitFor();
 assert.equal(bodies.length,2);assert.equal(bodies[0],bodies[1]);assert.equal(await page.locator('#final-request-reference').inputValue(),'');
 await capture('recorded','ar',390,'[data-final-request-outcome]');
 await page.locator('[data-final-request-outcome] a').click();await page.locator('[data-b09-record="qa-browser-final"]:visible').waitFor();
 await page.locator('.service-signout').click();
 await page.locator('input[type="password"]').waitFor();
 assert.equal(await page.locator('[data-final-request-outcome]').count(),0);
 assert.equal(await page.locator('[data-final-request-preview]').count(),0);
 await page.goto(`${args.url}/?service=api&lang=en#/approvals?lang=en&view=results`);
 await page.locator('input[type="password"]').waitFor();
 assert.equal(await page.locator('.b09-results').count(),0);
 evidence.cases.push({state:'actual-signout-removes-session-evidence-and-protects-return'});
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(error){evidence.outcome='failed';evidence.failure=error.message;evidence.visibleFailure=await page.locator('main').innerText({timeout:1000}).catch(()=> 'Destination changed');await page.screenshot({path:args.output+'/failure.png',fullPage:true});process.exitCode=1;}
finally{await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
