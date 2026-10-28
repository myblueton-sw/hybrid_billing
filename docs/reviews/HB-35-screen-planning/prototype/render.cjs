const {chromium}=require(process.env.HB_PLAYWRIGHT_MODULE || 'playwright');
const fs=require('fs');
const path=require('path');
const root=path.resolve(__dirname,'../../../..');
const korean=process.env.HB_DESIGN_LOCALE==='ko';
const suffix=korean?'-ko':'';
const out=path.join(root,'output/designs/HB-35-v2'+suffix);
const base=korean?out:path.join(root,'docs/reviews/HB-35-screen-planning/prototype');
fs.mkdirSync(out,{recursive:true});
fs.mkdirSync(path.join(root,'output/pdf'),{recursive:true});
(async()=>{
 const browser=await chromium.launch({headless:true,executablePath:process.env.HB_CHROMIUM_PATH || undefined});
 const page=await browser.newPage({viewport:{width:1440,height:1050},deviceScaleFactor:1});
 const errors=[];page.on('pageerror',e=>errors.push(String(e)));
 const report=[];
 for(const file of ['index.html','reconciliation.html','customer.html']){
  await page.goto('file://'+path.join(base,file));
  const ids=await page.locator('.screen').evaluateAll(xs=>xs.map(x=>x.id));
  for(const id of ids){
   await page.goto('file://'+path.join(base,file)+'#'+id);
   await page.screenshot({path:path.join(out,id+'.png'),fullPage:true});
   report.push(await page.locator('#'+id).evaluate(el=>({id:el.id,height:el.scrollHeight,width:el.scrollWidth,footerTop:el.querySelector('footer').getBoundingClientRect().top,overflow:Array.from(el.querySelectorAll('*')).filter(e=>e.scrollWidth>e.clientWidth+2&&getComputedStyle(e).overflowX==='visible').map(e=>e.tagName+':'+e.textContent.slice(0,45))})));
  }
 }
 await page.goto('file://'+path.join(base,'index.html')+'#close');
 await page.locator('[data-customer-search]').fill('not-a-customer');
 if(!await page.locator('[data-search-result]').textContent().then(x=>x.includes(korean?'일치하는 고객이 없습니다':'No matching')))throw Error('Empty filter failed');
 await page.locator('[data-customer-search]').fill('Northstar');
 if(await page.locator('[data-customer-row]:visible').count()!==1)throw Error('Customer filter failed');
 await page.locator('#close .heading a.primary').click();
 await page.locator('#case').waitFor({state:'visible'});
 await page.locator('#case .assistant a.primary').click();
 await page.locator('#recalc').waitFor({state:'visible'});
 await page.goto('file://'+path.join(base,'customer.html')+'#usage');
 await page.locator('#usage a[href="#usage-question"]').click();
 await page.locator('dialog').waitFor({state:'visible'});
 await page.locator('dialog button').click();
 await page.locator('dialog').waitFor({state:'hidden'});
 await page.setViewportSize({width:390,height:844});
 for(const [file,id] of [['index.html','close'],['reconciliation.html','case'],['customer.html','invoice']]){
  await page.goto('file://'+path.join(base,file)+'#'+id);
  await page.screenshot({path:path.join(out,id+'-mobile.png'),fullPage:true});
  report.push(await page.evaluate(()=>({mobile:location.hash,width:document.documentElement.scrollWidth,viewport:innerWidth})));
 }
 await page.setViewportSize({width:1440,height:1050});
 const htmls=['index.html','reconciliation.html','customer.html'].map(f=>fs.readFileSync(path.join(base,f),'utf8'));
 const sections=htmls.flatMap(s=>s.match(/<section class="screen [\s\S]*?<\/section>/g)).map(s=>s.replace(/href="(?:index|reconciliation|customer)\.html#/g,'href="#'));
 if(sections.length!==15)throw Error('Expected 15 screens');
 const printable='<!doctype html><html lang="'+(korean?'ko':'en')+'"><head><meta charset="utf-8"><title>Hybrid Billing - Multi-provider MSP design v2</title><style>'+fs.readFileSync(path.join(base,'style.css'),'utf8')+'</style></head><body>'+sections.join('')+'</body></html>';
 fs.writeFileSync(path.join(out,'print.html'),printable);
 await page.goto('file://'+path.join(out,'print.html'));
 await page.pdf({path:path.join(root,'output/pdf/hybrid-billing-msp-design-v2'+suffix+'.pdf'),preferCSSPageSize:true,printBackground:true});
 fs.writeFileSync(path.join(out,'render-check.json'),JSON.stringify({report,errors,interactionChecks:'filter empty/match, case/preview navigation, customer inquiry open/close passed'},null,2));
 await browser.close();
 console.log(JSON.stringify({screens:report.length,errors,report}));
})().catch(e=>{console.error(e);process.exit(1)});
