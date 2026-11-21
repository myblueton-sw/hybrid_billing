import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const cookie=process.env.HB_PROJECTION_QA_COOKIE,password=process.env.HB_PROJECTION_QA_PASSWORD;
delete process.env.HB_PROJECTION_QA_COOKIE;delete process.env.HB_PROJECTION_QA_PASSWORD;
assert.ok(cookie&&password&&args.url&&args.api&&args.output&&args.chromium);
await mkdir(args.output,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const context=await browser.newContext({viewport:{width:1440,height:1000}});
const split=cookie.indexOf('=');
await context.addCookies([{name:cookie.slice(0,split),value:cookie.slice(split+1),url:args.url,httpOnly:true,sameSite:'Strict'}]);
const page=await context.newPage();
const evidence={cases:[],errors:[],simulation:[]};
page.on('pageerror',e=>evidence.errors.push(e.message));
page.on('dialog',dialog=>dialog.accept());
let mutationMode='normal',mutationBodies=[],releaseMutation,prepareMode='normal',releasePrepare,refreshFails=false;
await page.route('**/api/**',async route=>{
 const u=new URL(route.request().url());
 if(refreshFails&&u.pathname==='/api/approval-review'){await route.fulfill({status:503,json:{code:'dependency_unavailable'}});return;}
 if(u.pathname.endsWith('/financial_charge.approve')||u.pathname.endsWith('/financial_charge.reject')){
  mutationBodies.push(route.request().postData());
  if(mutationMode==='reauth-refusal'||mutationMode==='permission-refusal'){
   const b=route.request().postDataJSON();await route.fulfill({status:403,json:{code:mutationMode==='reauth-refusal'?'reauthentication_required':'permission_denied',request_id:b.request_id,retry:false,next_action:'none'}});return;
  }
  const response=await route.fetch({url:args.api+u.pathname+u.search});
  if(mutationMode==='hold-then-lose'){await new Promise(resolve=>releaseMutation=resolve);await route.abort('failed');return;}
  await route.fulfill({response});return;
 }
 const response=await route.fetch({url:args.api+u.pathname+u.search});
 if(u.pathname.endsWith('/financial_charge.prepare')){
  if(prepareMode==='hold')await new Promise(resolve=>releasePrepare=resolve);
  if(prepareMode==='denied'||prepareMode==='service-hold'||prepareMode==='reauth'){
   const body=await response.json();body.fields.approve=prepareMode==='reauth'?'reauthentication_required':prepareMode==='denied'?'permission_denied':'hold_active';body.fields.reject=prepareMode==='reauth'?'reauthentication_required':prepareMode==='denied'?'permission_denied':'eligible';await route.fulfill({status:200,json:body});return;
  }
 }
 await route.fulfill({response});
});
const visible=s=>page.locator(s+':visible');
async function focusIs(selector){await page.waitForFunction(s=>[...document.querySelectorAll(s)].some(e=>e.getClientRects().length&&e===document.activeElement),selector);}
async function capture(state,lang,width,selector='[data-b09-decision]'){
 const element=visible(selector).first();await element.scrollIntoViewIfNeeded();await element.screenshot({path:`${args.output}/${state}-${lang}-${width}.png`});
 const check=await element.evaluate(root=>{
  const allowed=new Set();const probe=document.createElement('span');root.append(probe);
  for(const token of ['--border-section','--border-row','--border-control']){probe.style.borderColor=`var(${token})`;allowed.add(getComputedStyle(probe).borderTopColor);}probe.remove();
  const unexpected=[];for(const e of [root,...root.querySelectorAll('*')]){const c=getComputedStyle(e);for(const side of ['Top','Right','Bottom','Left'])if(parseFloat(c[`border${side}Width`])>0&&!allowed.has(c[`border${side}Color`]))unexpected.push(e.tagName);}
  return{overflow:document.documentElement.scrollWidth>innerWidth,direction:getComputedStyle(root).direction,unexpected};
 });
 if(state==='checked'||state==='confirm-reject')await page.screenshot({path:`${args.output}/page-${state}-${lang}-${width}.png`,fullPage:true});
 assert.equal(check.overflow,false);assert.equal(check.direction,lang==='ar'?'rtl':'ltr');assert.deepEqual(check.unexpected,[]);evidence.cases.push({state,lang,width,...check});
}
async function select(lang,width){
 await page.setViewportSize({width,height:1000});
 await page.evaluate(lang=>{location.hash=`#/approvals?lang=${lang}&view=history&approval-q=qa-current`;},lang);
 await visible('[data-b09-record="qa-current"]').waitFor();
 if(!await visible('[data-b09-detail-heading]').count())await visible('[data-b09-record="qa-current"]').click();
 await visible('[data-b09-decision]').waitFor();
}
try{
 await page.goto(`${args.url}/?service=api&lang=en#/approvals?lang=en&view=history&approval-q=qa-current`);
 await select('en',1440);await capture('unchecked','en',1440);
 prepareMode='hold';await visible('[data-b09-check]').click();await page.waitForFunction(()=>document.querySelector('[data-b09-decision="checking"]'));
 await capture('checking','en',1440);assert.ok(releasePrepare);prepareMode='normal';releasePrepare();await visible('[data-b09-choose="reject"]').waitFor();
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await select(lang,width);
  if(await visible('[data-b09-cancel]').count())await visible('[data-b09-cancel]').click();
  if(await visible('[data-b09-check]').count())await visible('[data-b09-check]').click();
  await visible('[data-b09-choose="reject"]').waitFor();await capture('checked',lang,width);
  await visible('[data-b09-choose="reject"]').focus();await page.keyboard.press('Enter');await focusIs('[data-b09-confirm-heading]');
  await visible('[data-b09-confirm]').click();await focusIs('[data-b09-error-summary]');await capture('validation',lang,width);
  await visible('[data-b09-confirm]').click();await focusIs('[data-b09-error-summary]');
  await visible('.b09-decision select').selectOption('incorrect_amount');await visible('.b09-decision textarea').fill('Synthetic reviewed rejection rationale');await capture('confirm-reject',lang,width);
  if(width===390){await page.setViewportSize({width:1024,height:1000});assert.equal(await visible('.b09-decision textarea').inputValue(),'Synthetic reviewed rejection rationale');await page.setViewportSize({width,height:1000});}
 }
 if(args.flow==='reject'){
  await visible('[data-b09-confirm]').click();await visible('[data-b09-decision-result="recorded"]').waitFor();
  assert.equal(mutationBodies.length,1);
  for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){await select(lang,width);await capture('rejection-recorded',lang,width);}
  assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';evidence.journey={screenDecisionControls:true,actualRejection:true,requiredReasonAndRationale:true};
 }else{
 await visible('[data-b09-cancel]').click();await select('en',1440);
 for(const [lang,width] of [['en',1440],['ar',390]]){
  await select(lang,width);
  if(await visible('[data-b09-check]').count())await visible('[data-b09-check]').click();
  for(const mode of ['denied','service-hold']){
   prepareMode=mode;await visible('[data-b09-recheck]').click();await visible('[data-b09-recheck]').waitFor();
   assert.equal(await visible('[data-b09-choose="approve"]').count(),0);
   assert.equal(await visible('[data-b09-choose="reject"]').count(),mode==='denied'?0:1);
   await capture(mode,lang,width);
  }
 }
 evidence.simulation.push('permission denied and service hold preparation outcomes for rendered state coverage');
 await select('en',1440);prepareMode='normal';
 if(await visible('[data-b09-check]').count())await visible('[data-b09-check]').click();else await visible('[data-b09-recheck]').click();
 await visible('[data-b09-choose="approve"]').click();await visible('.b09-decision textarea').fill('Synthetic known refusal before recovery');
 mutationMode='permission-refusal';await visible('[data-b09-confirm]').click();await visible('[data-b09-decision-result="refused"]').waitFor();
 prepareMode='reauth';await visible('[data-b09-check]').click();await page.getByRole('button',{name:'Confirm your identity',exact:true}).waitFor();
 await capture('refused-then-reauth','en',1440);
 await page.getByRole('button',{name:'Confirm your identity',exact:true}).click();await page.locator('#reauth-password').fill(password);await page.locator('dialog.service-reauth button[type="submit"]').click();await page.locator('dialog.service-reauth').waitFor({state:'detached'});
 assert.equal(mutationBodies.length,1);evidence.simulation.push('known no-effect refusal followed by reauth preparation; actual explicit identity confirmation without automatic submission');
 prepareMode='normal';mutationMode='normal';mutationBodies=[];

 if(await visible('[data-b09-check]').count())await visible('[data-b09-check]').click();
 await visible('[data-b09-choose="approve"]').click();await visible('.b09-decision textarea').fill('Synthetic actual approval through B09');
 mutationMode='hold-then-lose';await visible('[data-b09-confirm]').evaluate(b=>{b.click();b.click();});
 await visible('[data-b09-decision-result="pending"]').waitFor();await capture('pending','en',1440);
 assert.equal(mutationBodies.length,1);await page.keyboard.press('Escape');assert.equal(await visible('[data-b09-detail-heading]').count(),1);await page.setViewportSize({width:390,height:1000});await visible('[data-b09-decision-result="pending"]').waitFor();
 assert.ok(releaseMutation);releaseMutation();await visible('[data-b09-decision-result="unknown"]').waitFor();
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){await select(lang,width);await capture('unknown',lang,width);}
 await page.evaluate(()=>{location.hash='#/approvals?lang=en&view=history&approval-q=missing';});await visible('[data-b09-decision-result="unknown"]').waitFor();
 mutationMode='reauth-refusal';await visible('[data-b09-retry-decision]').click();await visible('[data-b09-decision-result="unknown"]').waitFor();
 await page.getByRole('button',{name:'Confirm your identity',exact:true}).click();await page.locator('dialog.service-reauth[open]').waitFor();await page.locator('#reauth-password').fill(password);
 await page.locator('dialog.service-reauth button[type="submit"]').click();await page.locator('dialog.service-reauth').waitFor({state:'detached'});
 evidence.simulation.push('reauthentication refusal after actual unknown effect; actual identity confirmation; no automatic retry');assert.equal(mutationBodies.length,2);
 mutationMode='permission-refusal';await visible('[data-b09-retry-decision]').click();await visible('[data-b09-decision-result="unknown"]').waitFor();assert.equal(mutationBodies.length,3);evidence.simulation.push('permission refusal retains earlier unknown effect');
 mutationMode='normal';refreshFails=true;await visible('[data-b09-retry-decision]').click();await visible('[data-b09-decision-result="recorded"]').waitFor();await page.getByRole('alert').filter({hasText:'The decision was recorded, but the list could not be refreshed.'}).waitFor();await capture('success-refresh-failed','en',390,'[data-b09-decision-result]');await focusIs('[data-b09-decision-result] [data-b09-decision-focus]');
 assert.equal(new Set(mutationBodies).size,1);assert.equal(mutationBodies.length,4);refreshFails=false;
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){await select(lang,width);await capture('recorded',lang,width);}
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';evidence.journey={screenDecisionControls:true,actualApproval:true,unknownAfterActualEffect:true,sameAttemptReplay:true,duplicateGuard:true,reauthenticationModal:true,unknownPreservedAfterRetryRefusal:true,successIndependentOfRefresh:true};
 }
}catch(error){evidence.outcome='failed';evidence.failure={kind:error.name,where:error.stack?.match(/financial-decision-screen-qa\.mjs:\d+:\d+/)?.[0],detail:error.message.split('\n')[0].slice(0,220)};await page.screenshot({path:args.output+'/failure.png'});throw new Error('Decision screen QA failed');}
finally{await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
