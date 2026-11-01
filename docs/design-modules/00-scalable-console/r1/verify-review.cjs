const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const {chromium}=require(process.env.HB_PLAYWRIGHT_MODULE||'playwright');
const base=process.env.HB_PREVIEW_URL||'http://127.0.0.1:8763';
const output=process.env.HB_QA_OUTPUT||'output/validation/HB-63-screen-review';
const key='hybrid-billing.screen-review.v1',checks=[];
const check=(name,value)=>{assert.ok(value,name);checks.push(name);console.log(name);};
(async()=>{
 fs.mkdirSync(output,{recursive:true});
 const {screens}=await import('./catalog.js');
 const browser=await chromium.launch({headless:true,...(process.env.HB_CHROMIUM?{executablePath:process.env.HB_CHROMIUM}:{})});
 try{
  const context=await browser.newContext({viewport:{width:1440,height:1000}});
  const page=await context.newPage();page.setDefaultTimeout(10000);
  const errors=[],external=[];
  page.on('pageerror',error=>errors.push(error.message));
  page.on('request',request=>{if(new URL(request.url()).origin!==new URL(base).origin)external.push(request.url());});
  const ready=async()=>{await page.locator('#confirm-screen-review').waitFor();await page.waitForFunction(()=>!document.querySelector('#results')||document.querySelector('#results').getAttribute('aria-busy')==='false');await page.waitForFunction(()=>document.querySelectorAll('#design-items code').length>0);};
  async function go(id,lang='en'){await page.goto(`${base}/?lang=${lang}#${id}`);await page.reload();await ready();}
  for(const lang of ['en','ko','ar']){
   await go('contents',lang);
   for(const screen of screens){
    await page.evaluate(id=>{location.hash=id},screen.id);await page.waitForFunction(id=>document.querySelector('.crumb bdi')?.textContent===id,screen.id);await ready();
    assert.equal(await page.locator('#confirm-screen-review').isEnabled(),true,`${screen.id} review button`);
    const ids=await page.locator('#design-items code').allTextContents();
    assert.equal(new Set(ids).size,ids.length,`${screen.id} unique IDs`);
    assert.ok(ids.every(id=>id.startsWith(screen.id+':')),`${screen.id} scoped IDs`);
   }
   check(`All ${screens.length} ${lang} screens have review controls and unique scoped implementation IDs`,true);
  }
  check('Viewing all screens does not confirm any owner scope',await page.evaluate(key=>localStorage.getItem(key)===null,key));
  await go('cost-pivot');await page.reload();await ready();
  await page.locator('#confirm-screen-review').click();
  check('Confirmation requires explicit owner acknowledgment',await page.locator('#save-screen-fix').isDisabled());
  await page.locator('#owner-review-ack').check();await page.locator('#save-screen-fix').click();
  await page.waitForFunction(()=>document.querySelector('#review-confirm-message').textContent.includes('Local review saved'));
  check('Isolated confirmation records exact variant and stable IDs',await page.evaluate(key=>{const record=JSON.parse(localStorage.getItem(key)).records[0];return record.screen==='cost-pivot'&&record.locale==='en'&&record.width===1440&&record.layout==='wide'&&record.items.includes('cost-pivot:field:currency')&&record.source==='local-unauthenticated-confirmation';},key));
  await page.locator('#close-dialog').click();await page.reload();await ready();
  await page.waitForFunction(()=>document.querySelector('#design-status').textContent.startsWith('Local FIX'));
  check('Local exact-scope FIX survives reload',true);
  await page.locator('#content button').first().evaluate(button=>{button.disabled=true;});
  await page.waitForFunction(()=>document.querySelector('#design-status').textContent.includes('not reviewed'));
  check('Changed action availability does not inherit FIX',true);
  await page.locator('#content button').first().evaluate(button=>{button.disabled=false;});
  await page.locator('[name=currency]').selectOption('AED');
  await page.waitForFunction(()=>document.querySelector('#design-status').textContent.includes('not reviewed'));
  check('Changed displayed input/state does not inherit FIX',true);
  await page.locator('[name=currency]').selectOption('USD');
  await page.setViewportSize({width:1200,height:1000});
  await page.waitForFunction(()=>document.querySelector('#design-status').textContent.includes('not reviewed'));
  check('Another width in the same layout does not inherit FIX',true);
  await page.locator('#confirm-screen-review').click();await page.locator('#owner-review-ack').check();await page.locator('#save-screen-fix').click();
  await page.waitForFunction(()=>document.querySelector('#review-confirm-message').textContent.includes('Local review saved'));
  check('Second width preserves both review records',await page.evaluate(key=>JSON.parse(localStorage.getItem(key)).records.length===2,key));
  await page.locator('#close-dialog').click();
  await page.locator('#language').selectOption('ko');await ready();
  check('Language variants remain separately unreviewed',(await page.locator('#design-status').innerText()).includes('다른 표시 조건'));
  await page.locator('#language').selectOption('en');await ready();
  const downloadPromise=page.waitForEvent('download');await page.locator('#export-screen-reviews').click();const download=await downloadPromise;const exportPath=path.join(output,'isolated-test-review-export.json');await download.saveAs(exportPath);
  const exported=JSON.parse(fs.readFileSync(exportPath,'utf8'));check('Export labels unauthenticated local evidence and preserves scopes',exported.kind==='local-unauthenticated-screen-review'&&exported.records.length===2&&exported.records.every(record=>record.source==='local-unauthenticated-confirmation'));
  await page.locator('#confirm-screen-review').click();await page.locator('#owner-review-ack').check();await page.setViewportSize({width:1100,height:1000});await page.locator('#save-screen-fix').click();
  await page.waitForFunction(()=>document.querySelector('#review-confirm-message').textContent.includes('changed'));
  check('Open confirmation rejects viewport changes',await page.evaluate(key=>JSON.parse(localStorage.getItem(key)).records.length===2,key));await page.locator('#close-dialog').click();
  await page.locator('#confirm-screen-review').click();await page.locator('#owner-review-ack').check();
  await page.route('**/design-revisions.json',async route=>{const response=await route.fetch();const data=await response.json();data.screens['cost-pivot'].revision='d'.repeat(64);await route.fulfill({json:data});});
  await page.locator('#save-screen-fix').click();await page.waitForFunction(()=>document.querySelector('#review-confirm-message').textContent.includes('changed'));
  check('Open confirmation rejects newer build revision',await page.evaluate(key=>JSON.parse(localStorage.getItem(key)).records.length===2,key));await page.locator('#close-dialog').click();await page.unroute('**/design-revisions.json');
  await page.evaluate(key=>{const data=JSON.parse(localStorage.getItem(key));data.records.forEach(record=>record.revision='d'.repeat(64));localStorage.setItem(key,JSON.stringify(data));},key);await page.reload();await ready();
  check('Prior revision evidence is retained for impact assessment',(await page.locator('#design-status').innerText()).includes('Change detected')&&await page.evaluate(key=>JSON.parse(localStorage.getItem(key)).records.length===2,key));
  for(const action of ['uncheck','close']){
   await page.locator('#confirm-screen-review').click();await page.locator('#owner-review-ack').check();
   let release,seen;const gate=new Promise(resolve=>{release=resolve;});const requested=new Promise(resolve=>{seen=resolve;});
   await page.route('**/design-revisions.json',async route=>{seen();await gate;await route.continue();});
   await page.locator('#save-screen-fix').click();await requested;
   if(action==='uncheck')await page.locator('#owner-review-ack').uncheck();else await page.locator('#close-dialog').click();
   release();await page.waitForFunction(()=>document.querySelector('#review-confirm-message').textContent.includes('changed'));
   check(`Withdrawing confirmation via ${action} blocks delayed save`,await page.evaluate(key=>JSON.parse(localStorage.getItem(key)).records.length===2,key));
   if(action==='uncheck')await page.locator('#close-dialog').click();await page.unroute('**/design-revisions.json');
  }
  await page.locator('#confirm-screen-review').click();await page.locator('#owner-review-ack').check();
  await page.evaluate(()=>{window.originalDigest=crypto.subtle.digest.bind(crypto.subtle);crypto.subtle.digest=async(...args)=>{window.digestStarted=true;await new Promise(resolve=>{window.releaseDigest=resolve;});return window.originalDigest(...args);};});
  await page.locator('#save-screen-fix').click();await page.waitForFunction(()=>window.digestStarted);
  await page.evaluate(()=>{document.querySelector('[name=currency]').value='AED';window.releaseDigest();crypto.subtle.digest=window.originalDigest;});
  await page.waitForFunction(()=>document.querySelector('#review-confirm-message').textContent.includes('changed'));
  check('State changes during async digest cannot save an old scope',await page.evaluate(key=>JSON.parse(localStorage.getItem(key)).records.length===2,key));await page.locator('#close-dialog').click();
  const failure=await browser.newContext({viewport:{width:1440,height:1000}});
  await failure.addInitScript(key=>{const original=Storage.prototype.setItem;Storage.prototype.setItem=function(name,value){if(name===key)throw new DOMException('Quota','QuotaExceededError');return original.call(this,name,value);};},key);
  const failedPage=await failure.newPage();await failedPage.goto(`${base}/?lang=en#cost-pivot`);await failedPage.locator('#confirm-screen-review').click();await failedPage.locator('#owner-review-ack').check();await failedPage.locator('#save-screen-fix').click();
  await failedPage.waitForFunction(()=>document.querySelector('#review-confirm-message').textContent.includes('No new FIX'));
  check('Storage quota failure never records success',await failedPage.evaluate(key=>localStorage.getItem(key)===null,key));await failure.close();
  await page.evaluate(key=>localStorage.setItem(key,'broken-json'),key);await page.reload();await ready();
  check('Corrupt storage is visible and preserved',(await page.locator('#design-storage-message').innerText()).includes('invalid')&&await page.evaluate(key=>localStorage.getItem(key)==='broken-json',key));
  await page.evaluate(key=>localStorage.removeItem(key),key);
  await page.route('**/design-revisions.json',route=>route.fulfill({json:{version:1,screens:{'cost-pivot':{revision:42}}}}));await page.reload();await ready();
  check('Malformed revision metadata disables confirmation instead of crashing',await page.locator('#confirm-screen-review').isDisabled());await page.unroute('**/design-revisions.json');
  await go('contents');
  check('Directory renders a bounded page',await page.locator('#design-directory-results a').count()===24);
  await page.locator('#design-directory-search').fill('customer-inquiry');
  check('Directory searches exact screen ID',await page.locator('#design-directory-results a').count()===1);
  await page.locator('#design-directory-search').fill('');await page.locator('#design-directory-next').click();
  check('Directory supports bounded next page',(await page.locator('#design-directory-count').innerText()).includes('2 /'));
  await page.locator('#design-directory-prev').click();await page.locator('#confirm-screen-review').click();await page.locator('#owner-review-ack').check();await page.locator('#save-screen-fix').click();
  await page.waitForFunction(()=>document.querySelector('#review-confirm-message').textContent.includes('Local review saved'));await page.locator('#close-dialog').click();
  await page.waitForFunction(()=>document.querySelector('#directory-review-count').textContent==='1'&&document.querySelector('#design-status').textContent.startsWith('Local FIX'));
  check('Directory review refreshes evidence count without invalidating its own scope',true);
  await page.locator('#design-directory-status').selectOption('reviewed');
  check('Directory review filter reflects current local evidence',await page.locator('#design-directory-results a').count()===1);
  await page.evaluate(key=>localStorage.removeItem(key),key);
  for(const lang of ['en','ko','ar']){
   await go('customer-inquiry',lang);await page.locator('#screen-design-fill').click();await page.locator('[name=line]').selectOption('DEMO-LINE-002');await page.locator('#case-submit').click();
   check(`Customer ${lang} case prevents duplicate creation`,await page.locator('#case-submit').isDisabled());
   await page.locator('#case-open-thread').click();await page.locator('.sd-public-thread').waitFor();
   check(`Customer ${lang} thread preserves selected line`,(await page.locator('#content').innerText()).includes('DEMO-LINE-002'));
   await page.locator('#case-example-reply').click();await page.locator('#case-acknowledge').click();
   check(`Customer ${lang} acknowledgment closes local reply`,await page.locator('#case-reply').isDisabled());
   await page.locator('#case-document').click();await page.locator('.sd-document').waitFor();
   check(`Customer ${lang} document linkage preserves identity and revision`,await page.locator('.sd-document h3').innerText()==='DEMO-PUBLISHED-001'&&await page.locator('[data-design-id="core-customer-invoice:data:selected-line"] dd').innerText()==='DEMO-LINE-002'&&await page.locator('[data-design-id="core-customer-invoice:data:document-revision"] dd').innerText()==='DEMO-r1');
   await go('review-issue',lang);await page.locator('[name=reviewer]').selectOption('DEMO-PERSON-A');await page.locator('[name=issuer]').selectOption('DEMO-PERSON-A');for(const name of ['source','recipient','payment'])await page.locator(`[name=${name}]`).check();
   check(`Issue ${lang} same-person review is blocked`,await page.locator('#workflow-review').isDisabled());await page.locator('[name=issuer]').selectOption('DEMO-PERSON-B');await page.locator('#workflow-review').click();
   check(`Issue ${lang} local review never enables financial actions`,await page.locator('#issue-approve').isDisabled()&&await page.locator('#issue-issue').isDisabled());await page.locator('[name=revision]').selectOption('DEMO-r2');
   check(`Issue ${lang} revision invalidates prior evidence`,!await page.locator('[name=source]').isChecked()&&await page.locator('#workflow-review').isDisabled());
   await go('erp',lang);await page.locator('#erp-submit').click();
   check(`ERP ${lang} request context and duplicate submission are locked`,await page.locator('[name=destination]').isDisabled()&&await page.locator('#erp-submit').isDisabled());
   await page.locator('[name=readback]').selectOption('unknown');await page.locator('#erp-inspect').click();check(`ERP ${lang} unknown outcome cannot retry`,await page.locator('#erp-retry').isDisabled());
   await page.locator('[name=readback]').selectOption('failed');await page.locator('#erp-inspect').click();await page.locator('#erp-retry').click();
   check(`ERP ${lang} confirmed failure retries same request`,(await page.locator('#content').innerText()).includes('LOCAL-ERP-ACC-000001-001')&&await page.locator('#erp-retry').isDisabled());
   await page.locator('[name=readback]').selectOption('readback');await page.locator('#erp-inspect').click();await page.locator('#erp-reconcile').click();
   check(`ERP ${lang} local reconciliation cannot post amounts`,await page.locator('#erp-post').isDisabled());
  }
  for(const lang of ['en','ko','ar'])for(const width of [360,1440]){
   await page.setViewportSize({width,height:1000});
   for(const route of ['contents','entry-login','index-approval','core-customer-invoice','customer-response','review-issue','erp']){
    await go(route,lang);await page.evaluate(()=>document.fonts.ready);
    check(`${route} ${lang} ${width} no document overflow`,await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
    await page.screenshot({path:path.join(output,`${route}-${lang}-${width}.png`),fullPage:true});
   }
  }
  check('Review integration has no browser runtime errors',errors.length===0);check('Review integration has no external requests',external.length===0);
  await context.close();
  fs.writeFileSync(path.join(output,'results.json'),JSON.stringify({status:'PASS',checks,scope:'isolated test contexts; no owner acceptance'},null,2));console.log(JSON.stringify({status:'PASS',checks:checks.length}));
 }finally{await browser.close();}
})().catch(error=>{console.error(error);process.exit(1);});
