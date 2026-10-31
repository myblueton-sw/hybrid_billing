const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const {pathToFileURL}=require('node:url');
const {chromium}=require(process.env.HB_PLAYWRIGHT_MODULE||'playwright');
const base=process.env.HB_PREVIEW_URL||'http://127.0.0.1:8763';
const output=process.env.HB_QA_OUTPUT||'output/validation/HB-63';
const observations=[];
const check=(name,value)=>{assert.ok(value,name);observations.push(name);};
(async()=>{
 const {screens}=await import(pathToFileURL(path.join(__dirname,'catalog.js')));
 const {queryRows}=await import(pathToFileURL(path.join(__dirname,'data.js')));
 const q={screen:'index-intake',account:'ACC-000001',search:'',provider:'',status:'',sort:'asc',size:100,snapshot:1};
 const source=screens.find(s=>s.id===q.screen);
 const first=await queryRows(source,q);
 const second=await queryRows(source,q,first.next);
 check('Million-record adapter returns exactly 100 bounded rows',first.total===1000000&&first.rows.length===100);
 check('Cursor pages are disjoint',second.rows[0].id==='REC-0000101'&&!first.rows.some(a=>second.rows.some(b=>a.id===b.id)));
 await assert.rejects(()=>queryRows(source,{...q,account:'ACC-000002'},first.next),/expired/);
 observations.push('Cross-account cursor rejected');
 const customer=screens.find(s=>s.group==='customer'&&s.kind==='list');
 const safe=await queryRows(customer,{...q,screen:customer.id});
 check('Customer data projection excludes operator fields',safe.rows.every(row=>!('provider'in row)&&!('owner'in row)&&!('file'in row)));
 fs.mkdirSync(output,{recursive:true});
 const browser=await chromium.launch({headless:true,...(process.env.HB_CHROMIUM?{executablePath:process.env.HB_CHROMIUM}:{})});
 try{
  const page=await browser.newPage({viewport:{width:1440,height:1000}});
  const errors=[];
  page.on('pageerror',error=>errors.push(error.message));
  async function ready(){await page.waitForFunction(()=>!document.querySelector('#results')||document.querySelector('#results').getAttribute('aria-busy')==='false');}
  async function go(id,lang='en'){await page.goto(`${base}/?lang=${lang}#${id}`);await ready();}
  for(const lang of ['en','ar','ko']){
   await go('operations',lang);
   for(const screen of screens){
    await page.evaluate(id=>{location.hash=id},screen.id);
    await page.waitForFunction(id=>document.querySelector('.crumb bdi')?.textContent===id,screen.id);
    await ready();
    assert.equal(await page.locator('h1').innerText(),screen.title[lang]);
    assert.ok(await page.locator('#results tbody tr').count()<=100);
   }
   observations.push(`All ${screens.length} routes rendered in ${lang}`);
  }
  await go('index-intake');
  check('Initial list bounded to 25 rows',await page.locator('#results tbody tr').count()===25);
  await page.locator('#next').click();await ready();
  check('Next page starts at record 26',await page.locator('[data-detail]').first().innerText()==='REC-0000026');
  await page.locator('#prev').click();await ready();
  check('Previous cursor restores first record',await page.locator('[data-detail]').first().innerText()==='REC-0000001');
  await page.locator('#page-size').selectOption('100');await ready();
  check('100-row option remains bounded',await page.locator('#results tbody tr').count()===100);
  await page.locator('#page-select').check();
  check('Page selection selects 100 IDs',await page.locator('#selected-count').innerText()==='100');
  await page.locator('#provider-filter').selectOption('AWS');await ready();
  check('Filter clears selection',await page.locator('#selected-count').innerText()==='0');
  check('Filtered total explicitly unknown',(await page.locator('#record-count').innerText()).includes('not yet known'));
  await page.locator('#all-select').click();
  await page.locator('[data-row]').first().uncheck();
  check('All-results selection supports exclusions',(await page.locator('#bulk-controls').innerText()).includes('1'));
  await page.locator('#review-selection').click();
  check('Bulk preview exposes scope and unknown count',(await page.locator('#dialog').innerText()).includes('not yet known'));
  await page.locator('#queue-selection').click();
  check('Job queued once',await page.locator('#jobs-list .job').count()===1);
  await page.locator('#close-dialog').click();
  await page.locator('#review-selection').click();await page.locator('#queue-selection').click();
  check('Duplicate submission does not add a job',await page.locator('#jobs-list .job').count()===1);
  check('Duplicate rejection is visible',(await page.locator('#toast').innerText()).includes('already queued'));
  await page.locator('#close-dialog').click();
  await page.locator('#choose-account').click();
  check('Account picker loads only 20 items',await page.locator('[role=option]').count()===20);
  await page.locator('[role=option]').first().focus();await page.keyboard.press('ArrowDown');await page.keyboard.press('Enter');
  check('Account listbox keyboard selects second account',await page.locator('[role=option][aria-selected=true]').getAttribute('data-account')==='ACC-000002');
  await page.locator('#account-search').fill('100000');
  check('Account 100000 is searchable without full list',await page.locator('[role=option]').count()===1);
  await page.locator('[role=option]').click();await page.locator('#account-apply').click();await ready();
  check('Account selection resets page and selection',(await page.locator('#choose-account').innerText()).includes('ACC-100000')&&await page.locator('#selected-count').innerText()==='0');
  await page.locator('#open-jobs').click();
  check('Other account cannot see previous jobs',await page.locator('#jobs-list .job').count()===0);
  await page.keyboard.press('Escape');
  check('Escape restores focus to trigger',await page.locator('#open-jobs').evaluate(node=>node===document.activeElement));
  await page.locator('#record-search').pressSequentially('REC');
  check('Typing after modal close preserves search focus',await page.locator('#record-search').evaluate(node=>node===document.activeElement));
  await page.locator('#record-search').fill('');await page.waitForTimeout(300);await ready();
  await page.evaluate(()=>{
   window.originalTimer=window.setTimeout;window.queryStarts=0;
   window.setTimeout=(fn,ms,...args)=>window.originalTimer(fn,ms===60?(++window.queryStarts===1?500:10):ms,...args);
  });
  await page.locator('#record-search').fill('REC-0000999');
  await page.waitForFunction(()=>window.queryStarts===1);
  await page.locator('#record-search').fill('REC-0000002');
  await page.waitForFunction(()=>window.queryStarts===2);await ready();
  check('Latest search completes ahead of earlier in-flight search',await page.locator('[data-detail]').first().innerText()==='REC-0000002');
  await page.waitForTimeout(550);
  check('Late stale search cannot replace latest result',await page.locator('[data-detail]').first().innerText()==='REC-0000002');
  await page.evaluate(()=>{window.setTimeout=window.originalTimer});
  await page.locator('#all-select').click();await page.locator('[data-row]').first().uncheck();
  check('Known zero-item exclusion blocks bulk submission',await page.locator('#review-selection').isDisabled());
  await page.locator('#record-search').fill('1000001');await page.waitForTimeout(300);await ready();
  check('Out-of-range search renders empty state',(await page.locator('#results').innerText()).includes('No matching records'));
  await page.locator('#reset-filters').click();await ready();
  await page.locator('.scenario summary').click();await page.locator('#scenario-error').click();await ready();
  check('Load error state is visible',(await page.locator('#results').innerText()).includes('Unable to load'));
  await page.locator('#retry-load').click();await ready();
  await page.locator('#all-select').click();await page.locator('#scenario-expire').click();
  check('Expired snapshot blocks actions',await page.locator('#review-selection').isDisabled()&&await page.locator('#next').isDisabled());
  await page.locator('#refresh').click();await ready();
  await page.locator('#intake').click();
  for(let i=0;i<4;i++)await page.locator('#demo-files').click();
  check('Metadata queue rejects over 100 descriptors',(await page.locator('#file-list').innerText()).includes('90 / 100')&&(await page.locator('#toast').innerText()).includes('at most 100'));
  check('Metadata list only renders 25 rows',await page.locator('#file-list tbody tr').count()===25);
  await page.locator('#file-next').click();
  check('File metadata pager advances',(await page.locator('#file-page').innerText()).startsWith('2'));
  await page.evaluate(()=>{
   window.contentReads=0;
   for(const method of ['arrayBuffer','text','stream'])File.prototype[method]=function(){window.contentReads++;throw Error('Unexpected content read')};
   for(const method of ['readAsArrayBuffer','readAsText','readAsDataURL','readAsBinaryString'])FileReader.prototype[method]=function(){window.contentReads++;throw Error('Unexpected content read')};
  });
  await page.locator('#file-input').setInputFiles({name:'synthetic.csv',mimeType:'text/csv',buffer:Buffer.from('x\n')});
  check('Local file intake uses metadata without reading content',await page.evaluate(()=>window.contentReads===0)&&(await page.locator('#file-list').innerText()).includes('91 / 100'));
  await page.locator('#run-files').click();
  check('Accepted file batch cannot be queued twice',await page.locator('#run-files').isDisabled());
  while(await page.locator('#file-next').isEnabled())await page.locator('#file-next').click();
  await page.waitForFunction(()=>/Processing|Preview complete|Needs attention/.test(document.querySelector('#file-list').innerText));
  check('Intake reflects linked job processing state',/Processing|Preview complete|Needs attention/.test(await page.locator('#file-list').innerText()));
  await page.screenshot({path:path.join(output,'file-intake.png'),fullPage:true});
  await page.locator('#file-jobs').click();
  check('Replacing intake dialog restores focus to visible control',await page.locator('#close-dialog').evaluate(node=>node===document.activeElement));
  check('Background concurrency bounded to two',await page.locator('.job .status.processing').count()<=2);
  const cancel=page.locator('[data-cancel]').first();await cancel.click();
  check('Job cancellation visible',await page.locator('.job .status.cancelled').count()===1);
  check('Cancelled job retains keyboard focus',await page.evaluate(()=>document.activeElement.matches('[data-job]')));
  await page.locator('[data-retry]').first().waitFor({timeout:16000});
  const retryId=await page.locator('[data-retry]').first().getAttribute('data-retry');
  await page.locator('[data-retry]').first().click();
  check('Retry only failed job and preserve focus',await page.evaluate(id=>document.activeElement.dataset.job===id,retryId));
  await page.locator('#close-dialog').click();
  await page.evaluate(id=>{location.hash=id},customer.id);await ready();
  await page.locator('#open-jobs').click();
  check('Customer portal excludes operator jobs',await page.locator('#jobs-list .job').count()===0);
  await page.locator('#close-dialog').click();
  await go('entry-login');
  await page.locator('[name=workspace]').fill('Demo workspace');await page.locator('[name=email]').fill('review@example.invalid');
  await page.locator('#screen-form button[type=submit]').click();
  check('Form draft visibly acknowledges preview-only save',(await page.locator('#form-feedback').innerText()).includes('preview only'));
  const workflow=screens.find(screen=>screen.kind==='workflow');await go(workflow.id);
  await page.locator('[data-step]:not(:disabled)').first().click();
  check('Workflow review keeps keyboard focus in next action',await page.evaluate(()=>document.activeElement.matches('[data-step],#workflow-review')));
  for(const lang of ['en','ar','ko']){
   for(const width of [360,768,1440]){
    await page.setViewportSize({width,height:1000});await go('operations',lang);
    check(`${lang} ${width}px has no document overflow`,await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth+1));
    check(`${lang} layout direction matches`,await page.locator('html').getAttribute('dir')===(lang==='ar'?'rtl':'ltr'));
    await page.screenshot({path:path.join(output,`${lang}-${width}.png`),fullPage:true});
   }
  }
  check('No browser runtime errors',errors.length===0);
  fs.writeFileSync(path.join(output,'results.json'),JSON.stringify({status:'PASS',observations},null,2));
  console.log(JSON.stringify({status:'PASS',checks:observations.length,routes:screens.length,locales:3},null,2));
 }finally{await browser.close();}
})().catch(error=>{console.error(error);process.exit(1);});
