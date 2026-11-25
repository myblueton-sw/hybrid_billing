import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const cookie=process.env.HB_PROJECTION_QA_COOKIE;
delete process.env.HB_PROJECTION_QA_COOKIE;
assert.ok(cookie&&args.url&&args.api&&args.output&&args.decision&&args.human&&args.console);
await mkdir(args.output,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const context=await browser.newContext();
const split=cookie.indexOf('=');
await context.addCookies([{name:cookie.slice(0,split),value:cookie.slice(split+1),url:args.url,httpOnly:true,sameSite:'Strict'}]);
const page=await context.newPage();
const evidence={cases:[],errors:[],mutations:0};
page.on('pageerror',()=>evidence.errors.push('page_runtime_error'));
await page.route('**/api/**',async route=>{
 const u=new URL(route.request().url());
 if(route.request().method()!=='GET'){evidence.mutations++;await route.abort('blockedbyclient');return;}
 const response=await route.fetch({url:args.api+u.pathname+u.search});await route.fulfill({response});
});
const kind={en:'ERP final-amount approval',ko:'ERP 최종 금액 승인',ar:'اعتماد المبلغ النهائي في ERP'};
const actions={approve:{en:'Approve',ko:'승인',ar:'اعتماد'},request:{en:'Request approval',ko:'승인 요청',ar:'طلب الاعتماد'}};
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390])for(const route of ['access','integrations']){
  await page.setViewportSize({width,height:1000});await page.goto('about:blank');
  await page.goto(`${args.url}/?service=api&lang=${lang}#/${route}?lang=${lang}&view=${route==='access'?'grants':'clients'}`);
  const grants=route==='access'?[{id:args.decision,action:'approve'},{id:args.human,action:'request'}]:[{id:args.console,action:'request'}];
  for(const grant of grants){
   const row=route==='access'?page.locator('tr').filter({has:page.locator(`[data-grant="${grant.id}"]`)}):page.locator('.p02-grants tr').filter({hasText:grant.id});
   await row.waitFor();await page.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
   const text=await row.innerText();assert.ok(text.includes(kind[lang]));assert.ok(!text.includes('erp_final_approval'));assert.ok(text.includes(actions[grant.action][lang]));
   const layout=await row.evaluate(e=>({direction:getComputedStyle(e).direction,overflow:document.documentElement.scrollWidth>innerWidth,borderStyle:getComputedStyle(e).borderStyle,borderColor:getComputedStyle(e).borderColor}));
   assert.equal(layout.direction,lang==='ar'?'rtl':'ltr');assert.equal(layout.overflow,false);
   await row.screenshot({path:`${args.output}/${route}-${grant.action}-${lang}-${width}.png`});
   await page.screenshot({path:`${args.output}/${route}-${grant.action}-context-${lang}-${width}.png`});
   evidence.cases.push({screen:route==='access'?'M03':'P02',lang,width,action:grant.action,...layout});
  }
 }
 assert.deepEqual(evidence.errors,[]);assert.equal(evidence.mutations,0);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};await page.screenshot({path:args.output+'/failure.png'});throw new Error('ERP final grant screen QA failed');}
finally{await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
