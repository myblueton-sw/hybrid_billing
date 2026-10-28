const {chromium}=require(process.env.HB_PLAYWRIGHT_MODULE||'playwright');
const fs=require('fs'),path=require('path'),{execFileSync}=require('child_process');
const root=path.resolve(__dirname,'../../..'),out=path.join(root,'output/designs/HB-40');
const assert=(v,m)=>{if(!v)throw Error(m);};
(async()=>{
 const browser=await chromium.launch({headless:true,executablePath:process.env.HB_CHROMIUM_PATH||undefined});
 const checks=[];
 for(const lang of ['en','ko']){
  const base=path.join(out,lang),manifest=JSON.parse(fs.readFileSync(path.join(base,'manifest.json')));
  const pdfs=[],errors=[],external=[];
  const page=await browser.newPage({viewport:{width:1440,height:1400}});
  page.on('pageerror',e=>errors.push(String(e)));page.on('request',r=>{if(/^https?:/.test(r.url()))external.push(r.url());});
  fs.mkdirSync(path.join(base,'render'),{recursive:true});
  for(const item of manifest.pages){
   await page.goto('file://'+path.join(base,'pages',item.id+'.html'));
   await page.emulateMedia({media:'print'});
   const bounds=await page.evaluate(()=>({bottom:document.querySelector('.book-main').getBoundingClientRect().bottom,footer:document.querySelector('.book-footer').getBoundingClientRect().top,width:document.documentElement.scrollWidth,images:[...document.images].every(i=>i.complete&&i.naturalWidth>0)}));
   const pdf=path.join(base,'render',String(item.page).padStart(2,'0')+'.pdf');
   await page.pdf({path:pdf,preferCSSPageSize:true,printBackground:true});
   const count=Number(execFileSync('pdfinfo',[pdf],{encoding:'utf8'}).match(/Pages:\s+(\d+)/)[1]);
   await page.screenshot({path:path.join(base,'render',String(item.page).padStart(2,'0')+'.png'),fullPage:true});
   checks.push({lang,id:item.id,page:item.page,bounds,count});
   pdfs.push(pdf);
   if(bounds.bottom>bounds.footer||bounds.width>1440||count!==1||!bounds.images)console.log(JSON.stringify({failure:true,lang,id:item.id,bounds,count}));
  }
  const failures=checks.filter(x=>x.lang===lang&&(x.bounds.bottom>x.bounds.footer||x.bounds.width>1440||x.count!==1||!x.bounds.images));
  if(!failures.length)execFileSync('pdfunite',[...pdfs,path.join(root,'output/pdf',`hybrid-billing-full-design-v8${lang==='ko'?'-ko':''}.pdf`)]);
  await page.emulateMedia({media:'screen'});
  await page.goto('file://'+path.join(base,'index.html'));
  await page.waitForURL('**/overview.html');
  assert(await page.locator('.book-rail nav a').count()===9,'Unified review navigation');
  await page.locator('.book-index').click();assert(await page.locator('.index-group a').count()===52,'Complete screen index');
  for(const item of manifest.pages){
   const text=fs.readFileSync(path.join(base,'pages',item.id+'.html'),'utf8');
   for(const m of text.matchAll(/href="([^"#]+\.html)"/g))assert(fs.existsSync(path.resolve(base,'pages',m[1])),'Broken book link '+item.id+' '+m[1]);
  }
  await page.goto('file://'+path.join(base,'customer/index.html'));
  await page.waitForURL('**/core-customer-overview.html');
  assert(await page.locator('.book-rail nav a').count()===7,'Customer-only navigation');
  for(const item of manifest.pages.filter(x=>x.chapter==='customer')){
   const text=fs.readFileSync(path.join(base,'customer',item.id+'.html'),'utf8');
   assert(!text.includes('DEMO-OCI-CONTRACT-01')&&!text.includes('GOODWILL-025'),'Operator fixture in customer entry');
   for(const m of text.matchAll(/href="([^"#]+\.html)"/g))assert(manifest.pages.some(x=>x.chapter==='customer'&&x.id+'.html'===m[1]),'Operator navigation in customer entry');
  }
  await page.setViewportSize({width:390,height:844});
  for(const id of ['overview','contents','operations','supplier-register','supplier-oci','v4-extract','core-customer-overview']){
   await page.goto('file://'+path.join(base,'pages',id+'.html'));
   const width=await page.evaluate(()=>document.documentElement.scrollWidth);
   checks.push({lang,mobile:id,width});
   if(width>390)console.log(JSON.stringify({failure:true,lang,mobile:id,width}));
   await page.screenshot({path:path.join(base,'render',id+'-mobile.png'),fullPage:true});
  }
  assert(errors.length===0&&external.length===0,'Browser errors/external requests');
  checks.push({lang,errors,external,links:'PASS',customerSeparation:'PASS'});
  await page.close();
 }
 fs.writeFileSync(path.join(out,'render-check.json'),JSON.stringify(checks,null,2));
 await browser.close();console.log(JSON.stringify({checks:checks.length}));
 if(checks.some(x=>x.bounds&&(x.bounds.bottom>x.bounds.footer||x.bounds.width>1440||x.count!==1||!x.bounds.images)||x.mobile&&x.width>390))process.exitCode=1;
})().catch(e=>{console.error(e);process.exit(1);});
