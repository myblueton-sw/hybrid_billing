const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const {chromium}=require(process.env.HB_PLAYWRIGHT_MODULE||'playwright');
const base=process.env.HB_PREVIEW_URL||'http://127.0.0.1:8763';
const output=process.env.HB_QA_OUTPUT||'output/validation/HB-63-journeys';
const checks=[];
const check=(name,value)=>{assert.ok(value,name);checks.push(name);console.log(name);};
(async()=>{
 fs.mkdirSync(output,{recursive:true});
 const browser=await chromium.launch({headless:true,...(process.env.HB_CHROMIUM?{executablePath:process.env.HB_CHROMIUM}:{})});
 try{
  const page=await browser.newPage({viewport:{width:1440,height:1000}});
  page.setDefaultTimeout(10000);
  const errors=[],requests=[];
  page.on('pageerror',error=>errors.push(error.message));
  page.on('request',request=>{if(new URL(request.url()).origin!==new URL(base).origin)requests.push(request.url());});
  const ready=()=>page.waitForFunction(()=>!document.querySelector('#results')||document.querySelector('#results').getAttribute('aria-busy')==='false');
  async function go(route){await page.goto(`${base}/?lang=en#${route}`);await page.reload();await ready();}
  async function navigate(route){await page.evaluate(route=>{location.hash=route},route);await page.waitForFunction(route=>document.querySelector('.crumb bdi')?.textContent===route,route);await ready();}
  async function fillStep(){const before=await page.locator('.crumb bdi').innerText();await page.locator('#setup-example').click();await page.locator('#setup-form [type=submit]').click();await page.waitForFunction(id=>document.querySelector('.crumb bdi')?.textContent!==id,before);}
  await go('setup-connectors');
  check('Direct connector entry requires prerequisites',await page.locator('[name=provider]').count()===0);
  await page.locator('#setup-start').click();await page.waitForFunction(()=>document.querySelector('.crumb bdi')?.textContent==='setup-profile');await fillStep();await fillStep();
  check('Provider setup includes IBM and VMware',(await page.locator('[name=provider] option').allTextContents()).includes('IBM Cloud')&&(await page.locator('[name=provider] option').allTextContents()).includes('VMware'));
  await page.locator('[name=provider]').selectOption('naver');await page.locator('#setup-example').click();await page.locator('#setup-plan').click();
  check('Unverified provider cannot produce an executable plan',await page.locator('#setup-connection-next').count()===0&&(await page.locator('#setup-error').innerText()).length>0);
  await page.locator('[name=provider]').selectOption('aws');await page.locator('[name=route]').selectOption('new');await page.locator('#setup-example').click();await page.locator('#setup-plan').click();
  check('New setup waits for plan acknowledgment',await page.locator('#setup-admin').isDisabled());
  await page.locator('[name=account]').press('Enter');
  check('Implicit connector submit does not advance',await page.locator('.crumb bdi').innerText()==='setup-connectors');
  await page.locator('[name=planAck]').check();
  check('New setup waits for administrator read-back',await page.locator('#setup-connection-next').isDisabled());
  await page.locator('#setup-admin').click();await page.locator('#setup-connection-next').click();
  for(const scenario of ['denied','outcome']){
   await page.locator('[name=scenario]').selectOption(scenario);await page.locator('#setup-diagnose').click();
   await page.waitForFunction(()=>!document.querySelector('#setup-diagnose').disabled);
   check(`${scenario} diagnostics block continuation`,await page.locator('#setup-next').isDisabled());
  }
  await page.locator('[name=scenario]').selectOption('partial');await page.locator('#setup-diagnose').click();
  await page.waitForFunction(()=>!document.querySelector('#setup-next').disabled);await page.locator('#setup-next').click();await fillStep();
  await page.locator('#setup-validate').click();await page.waitForFunction(()=>!document.querySelector('#setup-validate').disabled);
  check('Partial receipt cannot establish readiness',await page.locator('#setup-next').isDisabled());
  await navigate('setup-readtest');await page.locator('[name=scenario]').selectOption('normal');await page.locator('#setup-diagnose').click();
  await page.locator('#language').selectOption('ko');
  check('Language change does not leave diagnostics running',await page.locator('#setup-diagnose').isEnabled());
  await page.locator('#language').selectOption('en');await page.locator('#setup-diagnose').click();
  await page.waitForFunction(()=>!document.querySelector('#setup-next').disabled);await page.locator('#setup-next').click();await fillStep();
  await page.locator('#setup-validate').click();await page.waitForFunction(()=>!document.querySelector('#setup-next').disabled);await page.locator('#setup-next').click();
  check('Successful synthetic receipt still blocks financial publication',await page.locator('#content button:disabled').count()>0&&(await page.locator('#content').innerText()).includes('independent'));
  await page.screenshot({path:path.join(output,'onboarding-readiness.png'),fullPage:true});
  await navigate('setup-readtest');await page.locator('#setup-diagnose').click();
  check('Diagnostic rerun revokes downstream completion',await page.locator('[data-setup-route=setup-first]').isDisabled());
  await navigate('setup-connectors');await page.locator('[name=planAck]').uncheck();
  check('Acknowledgment revocation blocks later steps',await page.locator('[data-setup-route=setup-readtest]').isDisabled()&&await page.locator('[data-setup-route=setup-readiness]').isDisabled());
  await go('index-intake');await page.locator('#record-search').fill('REC-0000007');await page.waitForTimeout(250);await ready();await page.locator('[data-row]').first().check();
  await navigate('supplier-aws');await page.goBack();await page.waitForFunction(()=>document.querySelector('.crumb bdi')?.textContent==='index-intake');await ready();
  check('Back preserves search and selection',await page.locator('#record-search').inputValue()==='REC-0000007'&&await page.locator('[data-row]').first().isChecked());
  await page.locator('[data-detail]').first().click();await page.locator('#analyze-record').click();
  check('Contextual analysis carries exact source record',(await page.locator('#content').innerText()).includes('REC-0000007'));
  await page.locator('[name=request]').fill('Review missing source evidence');await page.locator('#analysis-form [type=submit]').click();await page.locator('#analysis-result').waitFor();
  check('Analysis separates observations from hypothesis',(await page.locator('#analysis-result').innerText()).includes('not a verified conclusion'));
  await page.locator('#analysis-task').click();check('Review task is explicitly local',(await page.locator('#analysis-result').innerText()).includes('No assignment'));
  await page.locator('[name=period]').fill('2026-09');check('Input edits remove stale analysis result',await page.locator('#analysis-result').count()===0);
  await page.locator('[name=scenario]').selectOption('failed');await page.locator('#analysis-form [type=submit]').click();await page.waitForFunction(()=>document.querySelector('#analysis-status').textContent.includes('Model unavailable'));
  check('Error preserves request',await page.locator('[name=request]').inputValue()==='Review missing source evidence');
  await page.locator('[name=scenario]').selectOption('partial');await page.locator('#analysis-form [type=submit]').click();await page.locator('#analysis-result').waitFor();
  check('Partial evidence is disclosed',(await page.locator('#analysis-result').innerText()).includes('Some evidence is unavailable'));
  await page.locator('#analysis-evidence').click();await page.waitForFunction(()=>document.querySelector('.crumb bdi')?.textContent==='index-intake');await ready();
  check('Evidence returns to same record',await page.locator('[data-detail]').first().innerText()==='REC-0000007');
  await page.locator('[data-detail]').first().click();await page.locator('#analyze-record').click();await page.locator('#analysis-form [type=submit]').click();await page.locator('#language').selectOption('ko');
  check('Language change cancels running analysis',await page.locator('#analysis-cancel').count()===0&&await page.locator('#analysis-form [type=submit]').isEnabled());
  await page.locator('#language').selectOption('en');await page.locator('#analysis-form [type=submit]').click();await page.locator('#analysis-cancel').click();
  check('Analysis cancellation is visible',(await page.locator('#analysis-status').innerText()).includes('cancelled'));
  await page.locator('#analysis-form [type=submit]').click();await navigate('operations');await page.waitForTimeout(800);
  check('Late analysis cannot replace destination',await page.locator('.crumb bdi').innerText()==='operations'&&await page.locator('#analysis-result').count()===0);
  await go('index-intake');await page.locator('#next').click();await ready();await page.locator('[data-row]').first().check();await navigate('supplier-aws');await page.goBack();await page.waitForFunction(()=>document.querySelector('.crumb bdi')?.textContent==='index-intake');await ready();
  check('Back restores cursor page and selected ID',await page.locator('[data-detail]').first().innerText()==='REC-0000026'&&await page.locator('[data-row]').first().isChecked());
  await page.locator('.scenario summary').click();await page.locator('#scenario-expire').click();await navigate('supplier-aws');await page.goBack();await page.waitForFunction(()=>document.querySelector('.crumb bdi')?.textContent==='index-intake');await ready();
  check('Expired snapshot stays expired on Back',await page.locator('#review-selection').isDisabled()&&await page.locator('#next').isDisabled());
  await go('index-intake');await page.locator('#record-search').fill('REC-0000007');await page.waitForTimeout(250);await ready();await page.locator('[data-row]').first().check();
  await page.locator('[data-detail]').first().click();await page.locator('#analyze-record').click();await page.locator('[name=request]').fill('Inspect scoped record');await page.locator('#analysis-form [type=submit]').click();
  await page.locator('#choose-account').click();await page.locator('#account-search').fill('2');await page.locator('[role=option]').first().click();await page.locator('#account-apply').click();await page.waitForTimeout(800);
  check('Account change cannot expose old analysis or task',await page.locator('#analysis-result').count()===0&&await page.locator('[name=request]').inputValue()===''&&!(await page.locator('#content').innerText()).includes('REC-0000007'));
  await page.goBack();await page.waitForFunction(()=>document.querySelector('.crumb bdi')?.textContent==='index-intake');await ready();
  check('Back after account change cannot restore old query or selection',await page.locator('#record-search').inputValue()===''&&await page.locator('#selected-count').innerText()==='0'&&(await page.locator('#choose-account').innerText()).includes('ACC-000002'));
  await go('cost-pivot');await page.locator('[name=currency]').selectOption('AED');await page.locator('[name=grouping]').selectOption('service');await page.locator('#report-form [type=submit]').click();
  check('Report shows selected descriptor and unavailable aggregates',(await page.locator('#report-result').innerText()).includes('AED')&&(await page.locator('#report-result').innerText()).includes('Service')&&(await page.locator('#report-result').innerText()).includes('No aggregate'));
  await page.locator('[name=currency]').selectOption('KRW');check('Report edits remove obsolete descriptor action',await page.locator('#report-analysis').count()===0);await page.locator('[name=currency]').selectOption('AED');await page.locator('#report-form [type=submit]').click();
  await page.locator('#report-analysis').click();check('Report currency passes to analysis',await page.locator('[name=currency]').inputValue()==='AED');
  await go('index-approval');check('Missing comparison blocks approval',await page.locator('#approval-approve').isDisabled());
  await page.locator('#approval-review').click();check('Review does not unlock approval',await page.locator('#approval-approve').isDisabled());
  await page.locator('[name=reason]').fill('Missing independent source evidence');await page.locator('#approval-reject [type=submit]').click();
  check('Rejection is local and keeps issuance separate',(await page.locator('#approval-status').innerText()).includes('Draft rejected')&&(await page.locator('#content').innerText()).includes('Not issued'));
  await page.locator('[name=revision]').selectOption('DEMO-r2');check('Revision change clears previous decision',await page.locator('#approval-review').isEnabled()&&await page.locator('[name=reason]').inputValue()==='');
  await page.locator('[name=reason]').fill('Still missing evidence');await page.locator('#approval-reject [type=submit]').click();await page.locator('[name=currency]').selectOption('AED');
  check('Currency change clears rejection and keeps approval blocked',await page.locator('[name=reason]').inputValue()===''&&await page.locator('#approval-approve').isDisabled());
  await go('core-customer-overview');
  check('Customer has no global statistics or account picker',!(await page.locator('#content').innerText()).includes('100,000')&&await page.locator('#choose-account').isDisabled());
  const customerRoutes=(await import('./catalog.js')).screens.filter(s=>s.group==='customer').map(s=>s.id);
  check('All customer page navigation remains customer-scoped',await page.locator('#app a[href*="#"]').evaluateAll((nodes,ids)=>nodes.every(node=>ids.includes(new URL(node.href).hash.slice(1))),customerRoutes));
  await page.locator('#screen-search').fill('AWS');check('Customer search cannot surface operator route',await page.locator('#screen-links a').count()===0);
  for(const lang of ['en','ar','ko'])for(const width of [360,1440]){
   await page.setViewportSize({width,height:1000});
   for(const route of ['setup-profile','assistant','cost-pivot','core-customer-overview','index-approval']){
    await page.goto(`${base}/?lang=${lang}#${route}`);await page.evaluate(()=>document.fonts.ready);
    check(`${route} ${lang} ${width} has no document overflow`,await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
    await page.screenshot({path:path.join(output,`${route}-${lang}-${width}.png`),fullPage:true});
   }
  }
  check('No browser runtime errors',errors.length===0);check('No external requests',requests.length===0);
  fs.writeFileSync(path.join(output,'results.json'),JSON.stringify({status:'PASS',checks},null,2));
  console.log(JSON.stringify({status:'PASS',checks:checks.length},null,2));
 }finally{await browser.close();}
})().catch(error=>{console.error(error);process.exit(1);});
