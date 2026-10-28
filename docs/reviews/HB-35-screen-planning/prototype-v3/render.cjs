const {chromium}=require(process.env.HB_PLAYWRIGHT_MODULE||'playwright');
const fs=require('fs');
const path=require('path');
const root=path.resolve(__dirname,'../../../..');
const locale=process.env.HB_DESIGN_LOCALE||'en';
const ko=locale==='ko';
const suffix=ko?'-ko':'';
const out=path.join(root,'output/designs/HB-35-v3'+suffix);
const files=['index.html','core-index.html','core-reconciliation.html','customer.html','core-customer.html'];
const expected=(en,kr)=>ko?kr:en;
(async()=>{
 const browser=await chromium.launch({headless:true,executablePath:process.env.HB_CHROMIUM_PATH||undefined});
 const page=await browser.newPage({viewport:{width:1440,height:1100},acceptDownloads:true});
 const errors=[];page.on('pageerror',e=>errors.push(String(e)));
 const report=[];
 for(const file of files){
  await page.goto('file://'+path.join(out,file));
  const ids=await page.locator('.screen').evaluateAll(es=>es.map(e=>e.id));
  for(const id of ids){
   await page.goto('file://'+path.join(out,file)+'#'+id);
   await page.screenshot({path:path.join(out,(file.startsWith('core-')?'core-':'')+id+'.png'),fullPage:true});
   report.push(await page.locator('#'+id).evaluate(el=>({id:el.id,height:el.scrollHeight,footerTop:el.querySelector('footer').offsetTop,overflow:[...el.querySelectorAll('*')].filter(e=>e.scrollWidth>e.clientWidth+3&&getComputedStyle(e).overflowX==='visible').map(e=>e.tagName+':'+e.textContent.slice(0,40))})));
  }
 }
 const goto=id=>page.evaluate(value=>{location.hash=value;},id);
 const click=action=>page.locator('.screen:visible [data-action="'+action+'"]').click();
 const select=(name,value)=>page.locator('.screen:visible [name="'+name+'"]').selectOption(value);
 const contains=async(en,kr)=>{const value=await page.locator('.screen:visible [data-result]').innerText();if(!value.includes(expected(en,kr)))throw Error('Expected '+expected(en,kr)+'; got '+value);};
 await page.goto('file://'+path.join(out,'index.html')+'#intake');
 await click('intake');await contains('BLOCKED','차단');
 await select('evidence','valid');await click('intake');await contains('ACCEPTED','수용됨');
 await page.locator('#intake footer a').click();
 await click('mapping');await contains('BLOCKED','차단');
 await select('customer','Aster');await click('mapping');await contains('DRAFT SAVED','초안 저장');
 await page.locator('#mapping footer a').click();
 await click('queue');await contains('BLOCKED','차단');
 await click('readback');await contains('READ-BACK','조회 확인');
 await select('mode','refresh');await click('queue');await contains('SRC-A3','SRC-A3');
 await select('mode','pinned');await click('queue');await contains('QUEUED','대기');
 await click('queue');await contains('EXISTING','기존');
 await click('advance');await contains('RUNNING','실행 중');
 await click('advance');await contains('VERIFIED','검증');
 await page.locator('#runs footer a').click();
 await click('approve');await contains('BLOCKED','차단');
 await page.locator('#approval [name="reason"]').fill(expected('Attach F1 acceptance evidence','F1 수용 근거 첨부'));
 await click('reject');await contains('RETURNED','반려');
 await click('attach');await click('approve');await contains('APPROVED','승인');
 await click('stale');await click('issue');await contains('BLOCKED','차단');
 await click('attach');await click('approve');await click('issue');await contains('SIMULATED ISSUED','발행 시연');
 await goto('mapping');await click('mapping');await contains('ISSUED ORIGINAL','발행 원본');await goto('approval');
 await page.screenshot({path:path.join(out,'approval-issued.png'),fullPage:true});
 await goto('pricing');await click('pricing');await contains('BLOCKED','차단');
 await select('overlap','resolved');await click('pricing');await contains('10,000','10,000');
 await select('discount','0');await click('pricing');await contains('11,000','11,000');
 await goto('seats');await click('seats');await contains('BLOCKED','차단');await select('policy','actual');await click('seats');await contains('2,100','2,100');
 await goto('commitments');await click('commitments');await contains('BLOCKED','차단');await select('model','typed');await click('commitments');await contains('100','100');
 await goto('fx');await select('rate','0');await click('fx');await contains('BLOCKED','차단');await select('rate','0.91');await click('fx');await contains('10,010','10,010');
 await goto('correction');await click('creditissue');await contains('BLOCKED','차단');await select('treatment','pass');await click('creditcheck');await click('creditapprove');await click('creditissue');await contains('136,060','136,060');
 await page.screenshot({path:path.join(out,'correction-issued.png'),fullPage:true});
 await page.goto('file://'+path.join(out,'customer.html')+'#history');
 for(const action of ['download','detail']){
  const pending=page.waitForEvent('download');await click(action);const download=await pending;await download.saveAs(path.join(out,download.suggestedFilename()));
 }
 await select('document','credit');const pending=page.waitForEvent('download');await click('download');const credit=await pending;await credit.saveAs(path.join(out,credit.suggestedFilename()));
 await goto('inquiry');await click('inquiry');await contains('Enter a question','문의 내용을 입력');
 await page.locator('#inquiry textarea').fill(expected('Why is there a separate credit?','별도 크레딧이 발행된 이유가 무엇인가요?'));
 await click('inquiry');await contains('Q-NS-09','Q-NS-09');await page.locator('#inquiry footer a').click();
 if(!(await page.locator('#response .outcome').innerText()).includes(expected('Why is there a separate credit?','별도 크레딧')))throw Error('Inquiry context lost');
 await click('acknowledge');await contains('ACKNOWLEDGED','확인 완료');
 await page.setViewportSize({width:390,height:844});
 for(const [file,id] of [['index.html','pricing'],['index.html','approval'],['customer.html','history']]){
  await page.goto('file://'+path.join(out,file)+'#'+id);
  await page.screenshot({path:path.join(out,id+'-mobile.png'),fullPage:true});
  const width=await page.evaluate(()=>({scroll:document.documentElement.scrollWidth,viewport:innerWidth}));
  if(width.scroll>width.viewport)throw Error('Mobile page overflow '+JSON.stringify(width));
 }
 await page.setViewportSize({width:1440,height:1100});
 let sections=[];
 for(const file of files){
  await page.goto('file://'+path.join(out,file));
  sections.push(...await page.locator('.screen').evaluateAll((es,file)=>es.map(e=>{e.removeAttribute('hidden');const originalId=e.id;if(e.classList.contains('v3'))e.querySelectorAll('nav a').forEach(a=>{if(a.getAttribute('href')==='#'+originalId)a.setAttribute('aria-current','page');else a.removeAttribute('aria-current');});e.id=file.replace('.html','')+'-'+e.id;e.querySelectorAll('a[href]').forEach(a=>{const value=a.getAttribute('href');if(value.includes('#')){const [target,id]=value.split('#');a.setAttribute('href','#'+(target||file).replace('.html','')+'-'+id);}});return e.outerHTML;}),file));
 }
 if(sections.length!==27)throw Error('Expected 27 screens');
 const print='<!doctype html><html lang="'+locale+'"><head><meta charset="utf-8"><style>'+fs.readFileSync(path.join(out,'style.css'),'utf8')+'\n'+fs.readFileSync(path.join(out,'v3.css'),'utf8')+'</style></head><body>'+sections.join('')+'</body></html>';
 fs.writeFileSync(path.join(out,'print.html'),print);
 await page.goto('file://'+path.join(out,'print.html'));
 await page.emulateMedia({media:'print'});
 const printBounds=await page.locator('.screen').evaluateAll(es=>es.map(e=>{const footer=e.querySelector('footer');const content=e.classList.contains('v3')?e.querySelector('.queue')||e.querySelector('.outcome'):footer.previousElementSibling;return {id:e.id,contentBottom:content.getBoundingClientRect().bottom-e.getBoundingClientRect().top,footerTop:footer.getBoundingClientRect().top-e.getBoundingClientRect().top};}));
 if(printBounds.some(x=>x.contentBottom>x.footerTop-5))throw Error('Print content intersects footer '+JSON.stringify(printBounds));
 await page.pdf({path:path.join(root,'output/pdf/hybrid-billing-msp-design-v3'+suffix+'.pdf'),preferCSSPageSize:true,printBackground:true});
 if(errors.length)throw Error(errors.join('\n'));
 fs.writeFileSync(path.join(out,'render-check.json'),JSON.stringify({screens:sections.length,errors,report,printBounds,checks:'Aster full correction/review/issuance loop; invalid states; duplicate job; refresh scope; pricing; proration; credits; FX; issued correction; sample downloads; inquiry context; three mobile pages'},null,2));
 await browser.close();console.log(JSON.stringify({locale,screens:sections.length,errors,printBounds}));
})().catch(e=>{console.error(e);process.exit(1)});
