const {chromium}=require(process.env.HB_PLAYWRIGHT_MODULE||'playwright');
const fs=require('fs'),path=require('path');
const root=path.resolve(__dirname,'../../..');
const target=path.join(root,'output/designs/HB-40');
fs.mkdirSync(target,{recursive:true});
(async()=>{
 const browser=await chromium.launch({headless:true,executablePath:process.env.HB_CHROMIUM_PATH||undefined});
 const result={};
 for(const locale of ['en','ko']){
  const suffix=locale==='ko'?'-ko':'';
  const page=await browser.newPage({viewport:{width:1440,height:1400}});
  const items=[];
  for(const [family,base,routes] of [
   ['v5','output/designs/HB-35-v5'+suffix+'/index.html',['customer/northstar','contracts/northstar','document/northstar-2026-08','policy/meridian','process/juniper-2026-08','erp']],
   ['current','docs/reviews/HB-38-billing-operations/prototype/index.html?lang='+locale,['operations','contracts/customer/C00226','contracts/customer/C00226/preview','contracts/supplier/aws','contracts/supplier/azure','contracts/supplier/gcp','contracts/supplier/alibaba']]
  ]){
   for(const route of routes){
    await page.goto('about:blank');
    await page.goto('file://'+root+'/'+base+'#'+route);
    await page.locator('h1').waitFor();
    const html=await page.evaluate(()=>{
     document.querySelectorAll('input').forEach(x=>{x.setAttribute('value',x.value);if(x.checked)x.setAttribute('checked','');else x.removeAttribute('checked');});
     document.querySelectorAll('select').forEach(x=>Array.from(x.options).forEach(o=>o.toggleAttribute('selected',o.selected)));
     return document.querySelector('main').outerHTML;
    });
    items.push({family,route,html});
   }
  }
  result[locale]=items;await page.close();
 }
 fs.writeFileSync(path.join(target,'captures.json'),JSON.stringify(result,null,2));
 await browser.close();console.log(JSON.stringify({locales:2,captures:26}));
})().catch(e=>{console.error(e);process.exit(1);});
