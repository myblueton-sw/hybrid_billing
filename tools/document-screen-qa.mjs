import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const password=process.env.HB_DOCUMENT_BROWSER_PASSWORD;delete process.env.HB_DOCUMENT_BROWSER_PASSWORD;
const proposer=process.env.HB_DOCUMENT_CLIENT_PROPOSER,confirmer=process.env.HB_DOCUMENT_CLIENT_CONFIRMER,fresh=process.env.HB_DOCUMENT_BROWSER_FRESH;
delete process.env.HB_DOCUMENT_BROWSER_FRESH;delete process.env.HB_DOCUMENT_CLIENT_PROPOSER;delete process.env.HB_DOCUMENT_CLIENT_CONFIRMER;
assert.ok(proposer&&confirmer&&args.url&&args.api&&args.output);
await mkdir(args.output,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const context=await browser.newContext({viewport:{width:1440,height:1000}}),page=await context.newPage();
const evidence={cases:[],errors:[]};let stage='initial-load';page.on('pageerror',()=>evidence.errors.push('main_page_runtime_error'));page.on('dialog',d=>d.accept());
const split=proposer.indexOf('=');await context.addCookies([{name:proposer.slice(0,split),value:proposer.slice(split+1),url:args.url,httpOnly:true,sameSite:'Strict'}]);
let proposalRefusal=!args.mode,losePreparation=false,historyFails=false,holdContext=false,releaseContext,contextReady,releasePrepare,prepareReady;const preparationBodies=[];
let contextDelivered;const contextDelivery=new Promise(resolve=>contextDelivered=resolve);
const contextWaiting=new Promise(resolve=>contextReady=resolve),prepareWaiting=new Promise(resolve=>prepareReady=resolve);
await page.route('**/api/**',async route=>{const u=new URL(route.request().url());if(proposalRefusal&&u.pathname.endsWith('/document_evidence.propose')){proposalRefusal=false;const body=route.request().postDataJSON();await route.fulfill({status:422,json:{request_id:body.request_id,code:'invalid_request',retry:false,next_action:'correct_request'}});return;}if(historyFails&&u.pathname.endsWith('/document_evidence.history')){await route.fulfill({status:503,json:{code:'dependency_unavailable'}});return;}if(u.pathname.endsWith('/erp_preparation.prepare'))preparationBodies.push(route.request().postData());const response=await route.fetch({url:args.api+u.pathname+u.search});let delayedContext=false;if(holdContext&&u.pathname.endsWith('/document_evidence.context')){delayedContext=true;contextReady();await new Promise(resolve=>releaseContext=resolve);}if(!args.mode&&u.pathname.endsWith('/erp_preparation.prepare')&&preparationBodies.length===1){prepareReady();await new Promise(resolve=>releasePrepare=resolve);}if(losePreparation&&u.pathname.endsWith('/erp_preparation.prepare')){losePreparation=false;await route.abort('failed');return;}await route.fulfill({response});if(delayedContext)contextDelivered();});
const panel=()=>page.locator('[data-document-panel]:visible');
async function capture(state,lang,width,selector='[data-document-panel]:visible'){
 const root=page.locator(selector);await root.scrollIntoViewIfNeeded();await root.screenshot({path:`${args.output}/${state}-${lang}-${width}.png`});await page.screenshot({path:`${args.output}/page-${state}-${lang}-${width}.png`,fullPage:true});
 const result=await root.evaluate(el=>{
  const allowed=new Set(),probe=document.createElement('span');el.append(probe);
  for(const token of ['--border-section','--border-row','--border-control','--text-body']){probe.style.borderColor=`var(${token})`;allowed.add(getComputedStyle(probe).borderTopColor);}probe.remove();
  let unexpectedBorders=0;for(const node of [el,...el.querySelectorAll('*')]){if(!node.getClientRects().length)continue;const style=getComputedStyle(node);for(const side of ['Top','Right','Bottom','Left'])if(parseFloat(style[`border${side}Width`])>0&&style[`border${side}Style`]!=='none'&&!allowed.has(style[`border${side}Color`]))unexpectedBorders++;}
  return{overflow:document.documentElement.scrollWidth>innerWidth,direction:getComputedStyle(el).direction,font:getComputedStyle(el).fontFamily,containerWidth:el.getBoundingClientRect().width,unexpectedBorders,clipped:[...el.querySelectorAll('input,textarea,button')].filter(e=>e.getClientRects().length&&(e.getBoundingClientRect().right>innerWidth+1||e.getBoundingClientRect().left< -1)).length};
 });
 assert.equal(result.overflow,false);assert.equal(result.direction,lang==='ar'?'rtl':'ltr');assert.equal(result.clipped,0);assert.equal(result.unexpectedBorders,0);assert.match(result.font,/IBM Plex/);evidence.cases.push({state,lang,width,...result});
}
async function fillPacket(form){
 await form.getByRole('radio').check();await form.locator('input[id$="-unit"]').fill('document-unit');await form.locator('input[id$="-evidence"]').fill('screen-evidence');
 const lines=form.locator('.b09-doc-lines fieldset');assert.equal(await lines.count(),2);
 for(let i=0;i<2;i++){const inputs=lines.nth(i).locator('input');await inputs.nth(0).fill(i===0?'2':'3');await inputs.nth(1).fill(i===0?'0.25':'0.75');await inputs.nth(2).fill(`screen-line-${i}`);}
 await form.locator('input[id$="-net"]').fill('15');await form.locator('input[id$="-tax"]').fill('1');await form.locator('textarea').fill('Explicit synthetic browser document');
}
async function attemptsMatrix(state){
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await page.setViewportSize({width,height:1000});await page.evaluate(lang=>{location.hash=`#/approvals?lang=${lang}&view=history`;},lang);
  await page.waitForFunction(lang=>document.documentElement.lang===lang,lang);await page.locator('.b09-screen[data-b09-state="loaded"]').waitFor();await capture(state,lang,width,'.b09-doc-attempts');
 }
 await page.setViewportSize({width:1440,height:1000});await page.evaluate(()=>{location.hash='#/approvals?lang=en&view=history';});await page.waitForFunction(()=>document.documentElement.lang==='en');await page.locator('.b09-screen[data-b09-state="loaded"]').waitFor();
}
try{
 await page.goto(`${args.url}/?service=api&lang=en#/approvals?lang=en&view=history&approval-q=${args.request}`);
 await page.locator(`[data-b09-record="${args.request}"]:visible`).click();await panel().waitFor();
 await panel().getByRole('button',{name:'Read document and preparation history',exact:true}).click();await panel().getByText('No records in this request',{exact:true}).first().waitFor();
 stage='form-matrix';
 for(const lang of (args.mode?[]:['en','ko','ar']))for(const width of [1440,1024,390]){
  await page.setViewportSize({width,height:1000});
  await page.evaluate(({lang,request})=>{location.hash=`#/approvals?lang=${lang}&view=history&approval-q=${request}`;},{lang,request:args.request});
  await page.locator(`[data-b09-record="${args.request}"]:visible`).waitFor();
  if(!await panel().count())await page.locator(`[data-b09-record="${args.request}"]:visible`).click();
  await panel().waitFor();
  const newLabel={en:'Enter new document evidence',ko:'새 문서 근거 입력',ar:'إدخال أدلة مستند جديد'}[lang];
  await panel().locator('[data-document-new]').click();await panel().locator('form').waitFor();
  await capture('form',lang,width);
  await panel().locator('form button[type=submit]').click();await panel().locator('form [role=alert]').waitFor();
  assert.ok(await panel().locator('form [role=alert]').evaluate(el=>el===document.activeElement));await capture('validation',lang,width);
  await fillPacket(panel().locator('form'));await panel().locator('form button[type=submit]').click();await panel().locator('.b09-doc-review').waitFor();await capture('proposal-review',lang,width);await panel().locator('.b09-doc-review button').first().click();
 }
 await page.setViewportSize({width:1440,height:1000});
 await page.evaluate(request=>{location.hash=`#/approvals?lang=en&view=history&approval-q=${request}`;},args.request);
 if(!args.mode)await page.locator(`[data-b09-record="${args.request}"]:visible`).click();await panel().locator('[data-document-new]').click();
 stage='proposal';const form=panel().locator('form');await form.waitFor();
 if(!args.mode){await fillPacket(form);await form.locator('.b09-doc-lines fieldset').first().locator('input').first().fill('0');await form.locator('input[id$="-net"]').fill('14');await form.locator('input[id$="-tax"]').fill('2');await form.locator('textarea').fill('가'.repeat(700));await form.locator('button[type=submit]').click();await form.locator('[role=alert]').waitFor();assert.ok(await form.getByText('Enter a positive canonical decimal within the displayed ERP scale and at most 12 integer digits.',{exact:true}).count());assert.ok(await form.getByText('Enter a nonblank reason of at most 2,000 UTF-8 bytes.',{exact:true}).count());await capture('partial-validation','en',1440);await fillPacket(form);}
 await form.getByRole('radio').check();
 await form.getByLabel('Document unit',{exact:true}).fill('document-unit');await form.getByLabel('Evidence reference',{exact:true}).first().fill('screen-evidence');
 const lines=form.locator('.b09-doc-lines fieldset');assert.equal(await lines.count(),2);
 for(let i=0;i<2;i++){await lines.nth(i).getByLabel('Document quantity',{exact:true}).fill(i===0?'2':'3');await lines.nth(i).getByLabel('Line tax',{exact:true}).fill(i===0?'0.25':'0.75');await lines.nth(i).getByLabel('Evidence reference',{exact:true}).fill(`screen-line-${i}`);}
 await form.getByLabel('Declared net total',{exact:true}).fill('15');await form.getByLabel('Declared tax total',{exact:true}).fill('1');await form.getByLabel('Reason',{exact:true}).fill('Explicit synthetic browser document');
 await form.getByRole('button',{name:'Review before submitting',exact:true}).click();
 const review=()=>panel().locator('.b09-doc-review');await review().waitFor();assert.ok(await review().locator('h4').evaluate(e=>e===document.activeElement));
 await capture('proposal-review','en',1440);await review().getByRole('button',{name:'Return to editing',exact:true}).click();assert.ok(await form.locator('h4').evaluate(e=>e===document.activeElement));
 await form.getByRole('button',{name:'Review before submitting',exact:true}).click();await review().getByRole('button',{name:'Submit reviewed request',exact:true}).click();
 if(!args.mode){await page.locator('[data-document-attempt="refused"]').waitFor();await capture('refused','en',1440,'.b09-doc-attempts');await panel().locator('[data-document-new]').click();await form.waitFor();await fillPacket(form);await form.locator('button[type=submit]').click();await review().getByRole('button',{name:'Submit reviewed request',exact:true}).click();}
 await page.locator('[data-document-attempt="recorded"]').waitFor();await panel().getByRole('button',{name:'Read document and preparation history',exact:true}).click();
 const documentButton=panel().getByRole('button',{name:/^Inspect.*document-/}).first();await documentButton.waitFor();const documentID=(await documentButton.locator('bdi').textContent()).trim();
 await documentButton.click();await panel().getByText('Retained document evidence',{exact:false}).first().waitFor();await capture('proposed','en',1440);
 stage='independent-confirmation';const second=await browser.newContext({viewport:{width:1440,height:1000}}),secondPage=await second.newPage(),sep=confirmer.indexOf('=');
 await second.addCookies([{name:confirmer.slice(0,sep),value:confirmer.slice(sep+1),url:args.url,httpOnly:true,sameSite:'Strict'}]);
 secondPage.on('pageerror',()=>evidence.errors.push('confirmation_page_runtime_error'));secondPage.on('dialog',d=>d.accept());
 await secondPage.route('**/api/**',async route=>{const u=new URL(route.request().url());await route.fulfill({response:await route.fetch({url:args.api+u.pathname+u.search})});});
 await secondPage.goto(`${args.url}/?service=api&lang=en#/approvals?lang=en&view=history&approval-q=${args.request}`);await secondPage.locator(`[data-b09-record="${args.request}"]:visible`).click();
 const other=secondPage.locator('[data-document-panel]:visible');await other.getByRole('button',{name:'Read document and preparation history',exact:true}).click();await other.getByRole('button',{name:`Inspect · ${documentID}`,exact:true}).click();
 await other.getByLabel('Reason',{exact:true}).fill('Independent browser confirmation');await other.getByRole('button',{name:'Review before submitting · Confirm document evidence',exact:true}).click();
 await other.locator('.b09-doc-review').getByRole('button',{name:'Return to editing',exact:true}).click();assert.ok(await other.getByLabel('Reason',{exact:true}).evaluate(e=>e===document.activeElement));
 await other.getByRole('button',{name:'Review before submitting · Confirm document evidence',exact:true}).click();await other.locator('.b09-doc-review').getByRole('button',{name:'Submit reviewed request',exact:true}).click();await secondPage.locator('[data-document-attempt="recorded"]').waitFor();
 await secondPage.screenshot({path:`${args.output}/page-independent-confirmation-en-1440.png`,fullPage:true});
 await panel().getByRole('button',{name:'Read document and preparation history',exact:true}).click();await panel().getByRole('button',{name:`Inspect · ${documentID}`,exact:true}).click();
 await panel().getByLabel('Reason',{exact:true}).fill('Prepare synthetic internal record');await panel().getByRole('button',{name:'Review before submitting · Prepare internal ERP record',exact:true}).click();
 stage='preparation';losePreparation=!args.mode||args.mode==='reauth';await review().getByRole('button',{name:'Submit reviewed request',exact:true}).click();
 if(!args.mode){await prepareWaiting;await page.locator('[data-document-attempt="pending"]').waitFor();await capture('sending','en',1440,'.b09-doc-attempts');assert.equal(preparationBodies.length,1);await page.locator('.b09-toolbar input').focus();releasePrepare();}
 if(args.mode==='revoke'){
  stage='hold-safe-revoke';await page.waitForFunction(()=>document.querySelectorAll('[data-document-attempt="recorded"]').length===2);
  await other.getByRole('button',{name:'Read document and preparation history',exact:true}).click();await other.getByRole('button',{name:`Inspect · ${documentID}`,exact:true}).click();
  await other.getByLabel('Reason',{exact:true}).fill('Revoke document evidence while service hold is active');await other.getByRole('button',{name:'Review before submitting · Revoke document evidence',exact:true}).click();await other.locator('.b09-doc-review').getByRole('button',{name:'Submit reviewed request',exact:true}).click();
  await secondPage.waitForFunction(()=>document.querySelectorAll('[data-document-attempt="recorded"]').length===2);await secondPage.screenshot({path:`${args.output}/page-hold-revoked-en-1440.png`,fullPage:true});
  evidence.journey={actualInternalPreparation:true,actualServiceHold:true,independentHoldSafeRevocation:true,preparationHistoryRetained:true};
 }else{
 await page.locator('[data-document-attempt="unknown"]').waitFor();if(!args.mode)assert.ok(await page.locator('.b09-toolbar input').evaluate(e=>e===document.activeElement));
 if(!args.mode)await attemptsMatrix('unknown');
 const original=preparationBodies[0];assert.ok(original);await page.evaluate(()=>{location.hash='#/approvals?lang=en&view=history';});await page.locator('.b09-screen[data-b09-state="loaded"][data-b09-selected=""]').waitFor();
 if(args.mode==='recovery'){
  stage='fresh-login-recovery';assert.ok(fresh);const freshContext=await browser.newContext({viewport:{width:1440,height:1000}}),freshPage=await freshContext.newPage(),split=fresh.indexOf('=');
  freshPage.on('pageerror',()=>evidence.errors.push('recovery_page_runtime_error'));
  await freshContext.addCookies([{name:fresh.slice(0,split),value:fresh.slice(split+1),url:args.url,httpOnly:true,sameSite:'Strict'}]);
  let recoveryBody;
  await freshPage.route('**/api/**',async route=>{const u=new URL(route.request().url());if(u.pathname.endsWith('/erp_preparation.prepare'))recoveryBody=route.request().postDataJSON();await route.fulfill({response:await route.fetch({url:args.api+u.pathname+u.search})});});
  await freshPage.goto(`${args.url}/?service=api&lang=en#/approvals?lang=en&view=history&approval-q=${args.request}`);await freshPage.locator(`[data-b09-record="${args.request}"]:visible`).click();
  const fp=freshPage.locator('[data-document-panel]:visible');await fp.getByRole('button',{name:'Read document and preparation history',exact:true}).click();
  assert.equal(await freshPage.locator('[data-document-attempt]').count(),0);
  await fp.getByRole('button',{name:`Inspect · ${documentID}`,exact:true}).click();
  const preparationID=JSON.parse(original).target_id;await fp.getByRole('button',{name:`Inspect · ${preparationID}`,exact:true}).click();
  await fp.getByText('Matching preparation observed',{exact:false}).waitFor();await fp.getByLabel('Reason',{exact:true}).fill('Fresh login inspects and reconciles original owned preparation');
  await fp.getByRole('button',{name:'Review before submitting · Recover this preparation key',exact:true}).click();
  await freshPage.screenshot({path:`${args.output}/page-fresh-recovery-review-en-1440.png`,fullPage:true});
  await fp.locator('.b09-doc-review').getByRole('button',{name:'Submit reviewed request',exact:true}).click();await freshPage.locator('[data-document-attempt="recorded"]').waitFor();
  assert.equal(recoveryBody.target_id,preparationID);assert.deepEqual(recoveryBody.body,JSON.parse(original).body);assert.notEqual(recoveryBody.idempotency_key,JSON.parse(original).idempotency_key);
  await freshPage.screenshot({path:`${args.output}/page-fresh-recovered-en-1440.png`,fullPage:true});await freshContext.close();
  evidence.journey={explicitProposal:true,independentConfirmation:true,actualServerUncertainReceipt:true,freshLoginHistory:true,exactOwnedFreshHTTPRecovery:true};
 }else if(args.mode==='permission'){
  await page.getByRole('button',{name:'Resend identical request',exact:true}).click();await page.locator('[data-document-attempt="unknown"]').getByText('permission_denied',{exact:true}).waitFor();
  assert.equal(preparationBodies.length,2);assert.equal(preparationBodies[0],preparationBodies[1]);await page.screenshot({path:`${args.output}/page-permission-loss-unknown-en-1440.png`,fullPage:true});
  evidence.journey={actualPermissionRevocationAfterEffect:true,unknownSurvivesDeniedRetry:true,identicalEnvelope:true};
 }else{
 if(args.mode==='reauth'){stage='reauthentication';
  await page.getByRole('button',{name:'Resend identical request',exact:true}).click();await page.locator('[data-document-attempt="unknown"]').getByText('reauthentication_required',{exact:true}).waitFor();
  await page.getByRole('button',{name:'Reauthenticate',exact:true}).click();await page.locator('#reauth-password').fill(password);await page.locator('dialog.service-reauth button[type="submit"]').click();await page.locator('dialog.service-reauth').waitFor({state:'detached'});assert.equal(preparationBodies.length,2);
 }
 await page.getByRole('button',{name:'Resend identical request',exact:true}).click();await page.waitForFunction(()=>document.querySelectorAll('[data-document-attempt="recorded"]').length===2);
 assert.equal(preparationBodies.length,args.mode==='reauth'?3:2);assert.ok(preparationBodies.every(body=>body===original));
 if(!args.mode){
  await attemptsMatrix('recorded');
  await page.locator(`[data-b09-record="${args.request}"]:visible`).click();await panel().getByRole('button',{name:'Read document and preparation history',exact:true}).click();await panel().getByRole('button',{name:`Inspect · ${documentID}`,exact:true}).waitFor();
  stage='async-query';historyFails=true;await panel().getByRole('button',{name:'Read document and preparation history',exact:true}).click();await panel().getByText('dependency_unavailable',{exact:true}).first().waitFor();assert.equal(await panel().getByRole('button',{name:/^Inspect/}).count(),0);assert.equal(await page.locator('[data-document-attempt="recorded"]').count(),2);await capture('history-unavailable','en',1440);historyFails=false;
  holdContext=true;await panel().locator('[data-document-new]').click();await contextWaiting;await capture('context-loading','en',1440);
  await page.locator('.b09-result:visible').getByRole('button',{name:'Close details',exact:true}).click();holdContext=false;const lateResponse=page.waitForResponse(r=>r.url().endsWith('/document_evidence.context'));releaseContext();await(await lateResponse).finished();await contextDelivery;await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
  await page.locator('.b09-screen[data-b09-selected=""]').waitFor();await page.locator(`[data-b09-record="${args.request}"]:visible`).click();await panel().waitFor();assert.equal(await panel().locator('form').count(),0);
  evidence.simulation={cleanProposalRefusal:true,unavailableHistoryResponse:true,lateContextResponse:true,recordedReceiptSurvivesRefreshFailure:true,userFocusPreservedOnCompletion:true};
 }
 await page.screenshot({path:`${args.output}/page-recovered-en-1440.png`,fullPage:true});
 evidence.journey={explicitProposal:true,independentConfirmation:true,actualPrepareResponseLoss:true,identicalRetryAfterNavigation:true,proposalAndActionEditFocus:true,actualReauthentication:args.mode==='reauth'};
 }
 }
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={stage,category:['TimeoutError','AssertionError'].includes(e.name)?e.name:'execution_error'};await page.screenshot({path:`${args.output}/failure.png`,fullPage:true}).catch(()=>{});process.exitCode=1;}
finally{await writeFile(`${args.output}/result.json`,JSON.stringify(evidence,null,2));await browser.close();}
