const {chromium}=require(process.env.HB_PLAYWRIGHT_MODULE||'playwright');
const fs=require('fs');
const path=require('path');
const {execFileSync}=require('child_process');
const root=path.resolve(__dirname,'../../../..');
const ko=process.env.HB_DESIGN_LOCALE==='ko',suffix=ko?'-ko':'';
const out=path.join(root,'output/designs/HB-35-v4'+suffix);
const expect=(en,kr)=>ko?kr:en;
(async()=>{
 const browser=await chromium.launch({headless:true,executablePath:process.env.HB_CHROMIUM_PATH||undefined});
 const page=await browser.newPage({viewport:{width:1440,height:1100},acceptDownloads:true});
 const errors=[],external=[];page.on('pageerror',e=>errors.push(String(e)));page.on('request',r=>{if(/^https?:/.test(r.url()))external.push(r.url());});
 await page.goto('file://'+path.join(out,'source.html'));
 await page.pdf({path:path.join(out,'sample-cedar-aws-09.pdf'),preferCSSPageSize:true,printBackground:true});
 execFileSync('pdftoppm',['-f','1','-singlefile','-scale-to','1400','-png',path.join(out,'sample-cedar-aws-09.pdf'),path.join(out,'sample-source')]);
 const report=[];
 for(const file of ['index.html','customer.html']){
  await page.goto('file://'+path.join(out,file));
  for(const id of await page.locator('.screen').evaluateAll(xs=>xs.map(x=>x.id))){
   await page.goto('file://'+path.join(out,file)+'#'+id);
   await page.screenshot({path:path.join(out,id+'.png'),fullPage:true});
   report.push(await page.locator('#'+id).evaluate(e=>({id:e.id,height:e.scrollHeight,overflow:[...e.querySelectorAll('*')].filter(x=>x.scrollWidth>x.clientWidth+3&&getComputedStyle(x).overflowX==='visible').map(x=>x.tagName+':'+x.textContent.slice(0,40))})));
  }
 }
 const go=id=>page.evaluate(id=>location.hash=id,id);
 const click=action=>page.locator('.screen:visible [data-action="'+action+'"]').click();
 const select=(name,value)=>page.locator('.screen:visible [name="'+name+'"]').selectOption(value);
 const contains=async(en,kr)=>{const value=await page.locator('.screen:visible [data-result]').innerText();if(!value.includes(expect(en,kr)))throw Error('Expected '+expect(en,kr)+' got '+value);};
 await page.goto('file://'+path.join(out,'index.html')+'#monthly');
 await select('basis','adjusted');await click('filter');
 if(!(await page.locator('#monthly tr[data-month="2026-09"]').innerText()).includes('136,060.00'))throw Error('Adjusted history mismatch');
 await select('month','2026-10');await click('filter');if(await page.locator('#monthly tbody tr:visible').count()!==1)throw Error('Month filter failed');
 await select('month','all');await select('basis','original');await click('filter');
 await page.locator('[data-open-month="2026-08"]').click();
 if(!(await page.locator('#month').innerText()).includes('DEMO-2026-08102'))throw Error('Wrong historical detail');
 if(await page.locator('#month .controls a:visible').count())throw Error('September links in August');
 await go('monthly');await page.locator('[data-open-month="2026-09"]').click();
 await click('immutable');await contains('READ-ONLY','읽기 전용');
 await go('variance');await select('basis','adjusted');await click('variance');await contains('4.66%','4.66%');
 await go('audit');await click('accept');await contains('BLOCKED','차단');
 await go('inbox');await page.locator('[name="pdf"]').setInputFiles({name:'local-test.pdf',mimeType:'application/pdf',buffer:Buffer.from('%PDF synthetic metadata-only test')});await contains('LOCAL METADATA','로컬 메타데이터');
 for(const state of ['duplicate','encrypted','unsupported','failed']){await select('sample-state',state);await click('sample');}
 await select('sample-state','ready');await click('sample');await contains('SYNTHETIC FIXTURE','합성 예시');
 await page.locator('#inbox footer a').click();await click('verify');await contains('BLOCKED','차단');
 await page.locator('#extract [name="support"]').fill('4000.00');await page.locator('#extract [name="reason"]').fill(expect('Original page 1 R3 confirms 4,000.00','원본 1페이지 R3에서 4,000.00 확인'));await page.locator('[name="source-checked"]').check();
 await click('verify');await contains('VERIFIED E2','E2 검증');
 await page.screenshot({path:path.join(out,'extract-verified.png'),fullPage:true});
 await go('audit');await click('accept');await contains('BLOCKED','차단');
 await go('differences');await select('detail','wrong');await click('reconcile');await contains('BLOCKED','차단');
 await select('detail','correct');await click('reconcile');await contains('RECONCILED','대사 완료');
 await go('extract');await page.locator('[name="reason"]').fill(expect('Reviewed original R3 again','원본 R3 재확인'));
 await go('audit');await click('accept');await contains('BLOCKED','차단');
 await go('extract');await click('verify');await go('differences');await click('reconcile');await go('audit');await click('accept');await contains('ACCEPTED INPUT','입력 스냅샷 수용');
 await click('accept');await contains('ALREADY','이미');
 await click('revise');await contains('E3','E3');await click('accept');await contains('BLOCKED','차단');
 if(!(await page.locator('.accepted-history').innerText()).includes('E2'))throw Error('Accepted history lost');
 await go('extract');if(!(await page.locator('#extract .scope4').innerText()).includes('E3'))throw Error('Stale extraction revision');
 await contains('CURRENT E3 DRAFT','현재 E3 초안');
 if(!(await page.locator('#differences .scope4').innerText()).includes('E3'))throw Error('Stale difference revision');
 if(!(await page.locator('#extract tbody tr').nth(2).innerText()).includes(expect('Not verified','미검증')))throw Error('Stale field verification');
 await page.locator('[name="reason"]').fill(expect('Reconfirm inherited original value','승계한 원본값 재확인'));await page.locator('[name="source-checked"]').check();await click('verify');await go('differences');await click('reconcile');await go('audit');await click('accept');await contains('E3','E3');
 if(await page.locator('.accepted-history p').count()!==2)throw Error('Accepted revision history incomplete');
 await page.screenshot({path:path.join(out,'audit-revisions.png'),fullPage:true});
 await page.goto('file://'+path.join(out,'customer.html'));
 await select('customer-month','2026-09');await click('customer-month');await contains('136,060','136,060');
 const downloaded=page.waitForEvent('download');await click('download');const download=await downloaded;await download.saveAs(path.join(out,download.suggestedFilename()));
 await select('customer-month','2026-10');await click('download');await contains('NOT ISSUED','미발행');
 await page.setViewportSize({width:390,height:844});
 for(const [file,id] of [['index.html','monthly'],['index.html','extract'],['customer.html','customer-months']]){
  await page.goto('file://'+path.join(out,file)+'#'+id);await page.screenshot({path:path.join(out,id+'-mobile.png'),fullPage:true});
  if(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth))throw Error('Mobile overflow '+id);
 }
 await page.setViewportSize({width:1440,height:1100});
 const sections=[];
 for(const file of ['index.html','customer.html']){
  await page.goto('file://'+path.join(out,file));
  sections.push(...await page.locator('.screen').evaluateAll(es=>es.map(e=>{e.removeAttribute('hidden');const id=e.id;e.querySelectorAll('nav a').forEach(a=>{if(a.hash==='#'+id)a.setAttribute('aria-current','page');else a.removeAttribute('aria-current');});e.querySelectorAll('a[href^="#"]').forEach(a=>a.setAttribute('href','#v4-'+a.getAttribute('href').slice(1)));e.id='v4-'+id;return e.outerHTML;})));
 }
 await page.goto('file://'+path.join(out,'previous/print.html'));
 const previous=await page.locator('.screen').evaluateAll(es=>es.map(e=>e.outerHTML));
 const previousCSS=await page.locator('style').innerText();
 if(previous.length!==27||sections.length!==8)throw Error('Unexpected page count');
 const css=previousCSS+'\n'+fs.readFileSync(path.join(out,'v4.css'),'utf8');
 fs.writeFileSync(path.join(out,'print.html'),'<!doctype html><html lang="'+(ko?'ko':'en')+'"><head><meta charset="utf-8"><style>'+css+'</style></head><body>'+sections.join('')+previous.join('')+'</body></html>');
 await page.goto('file://'+path.join(out,'print.html'));await page.emulateMedia({media:'print'});
 const bounds=await page.locator('.screen.v4').evaluateAll(es=>es.map(e=>({id:e.id,bottom:e.querySelector('.outcome4').getBoundingClientRect().bottom-e.getBoundingClientRect().top,footer:e.querySelector('footer').getBoundingClientRect().top-e.getBoundingClientRect().top})));
 if(bounds.some(x=>x.bottom>=x.footer-5))throw Error('Print collision '+JSON.stringify(bounds));
 await page.pdf({path:path.join(root,'output/pdf/hybrid-billing-msp-design-v4'+suffix+'.pdf'),preferCSSPageSize:true,printBackground:true});
 if(errors.length||external.length)throw Error(JSON.stringify({errors,external}));
 fs.writeFileSync(path.join(out,'render-check.json'),JSON.stringify({pages:35,newScreens:8,errors,external,report,bounds,checks:'monthly original/adjusted/filter/detail; file metadata isolation; sample error branches; correction/verification/source reconciliation gates; changed-field invalidation; E2/E3 preserved accepted history; customer monthly export and pending month; three mobile states'},null,2));
 await browser.close();console.log(JSON.stringify({locale:ko?'ko':'en',pages:35,errors,external,bounds}));
})().catch(e=>{console.error(e);process.exit(1)});
