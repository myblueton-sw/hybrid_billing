import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const cookie=process.env.HB_PROJECTION_QA_COOKIE;
delete process.env.HB_PROJECTION_QA_COOKIE;
assert.ok(cookie&&args.url&&args.api&&args.output&&args.bulk&&args.chromium);
await mkdir(args.output,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const context=await browser.newContext({viewport:{width:1440,height:1000}});
const split=cookie.indexOf('=');
await context.addCookies([{name:cookie.slice(0,split),value:cookie.slice(split+1),url:args.url,httpOnly:true,sameSite:'Strict'}]);
const page=await context.newPage();
const evidence={cases:[],errors:[],executionRequests:0,simulation:['rechecked=false is an injected display condition, not actual result-authority approval policy']};
let rechecked=true;
page.on('pageerror',e=>evidence.errors.push('page_runtime_error'));
await page.route('**/api/**',async route=>{
 const u=new URL(route.request().url());
 if(u.pathname.endsWith('/bulk_revocation.execute')) { evidence.executionRequests++;await route.abort('blockedbyclient');return; }
 const response=await route.fetch({url:args.api+u.pathname+u.search});
 if(!rechecked&&u.pathname===`/api/v1/identity/bulk-revocations/${args.bulk}`){const body=await response.json();const row=body.impact.pending.find(p=>p.store==='result_authority_proposals');assert.ok(row);row.rechecked=false;await route.fulfill({response,json:body});return;}
 await route.fulfill({response});
});
const labels={en:'ERP result evidence permission proposals',ko:'ERP 결과 근거 권한 제안',ar:'مقترحات صلاحيات أدلة نتائج ERP'};
const permissions={en:'Document evidence and ERP preparation permission proposals',ko:'문서 근거·ERP 준비 권한 제안',ar:'مقترحات صلاحيات أدلة المستند وإعداد ERP'};
const notes={en:'Rechecked when approved',ko:'승인 시 다시 확인됨',ar:'يُعاد التحقق منه عند الموافقة'};
const unknownNotes={en:'Not confirmed to be rechecked when approved',ko:'승인 시 재확인 여부 미확인',ar:'لم يتأكد أنه يُعاد التحقق منه عند الموافقة'};
try {
 for(const reported of [true,false])for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  rechecked=reported;
  await page.setViewportSize({width,height:1000});await page.goto('about:blank');
  await page.goto(`${args.url}/?service=api&lang=${lang}#/users?lang=${lang}&view=revocation`);
  await page.locator(`[data-job="${args.bulk}"]`).click();
  const row=page.locator('.m01-impact .m01-sum > div').filter({has:page.getByText(labels[lang],{exact:true})});
  const permission=page.locator('.m01-impact .m01-sum > div').filter({has:page.getByText(permissions[lang],{exact:true})});
  await row.waitFor();await page.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
  assert.equal(await row.locator('dd .m01-line').first().innerText(),'2');
  assert.equal(await permission.locator('dd .m01-line').first().innerText(),'2');
  assert.ok((await row.innerText()).includes((reported?notes:unknownNotes)[lang]));
  assert.ok(!(await page.locator('.m01-impact').innerText()).includes('result_authority_proposals'));
  const layout=await row.evaluate(e=>({direction:getComputedStyle(e).direction,overflow:document.documentElement.scrollWidth>innerWidth,rowHeight:e.getBoundingClientRect().height,localClipping:[...e.querySelectorAll('dt,dd')].filter(n=>n.scrollWidth>n.clientWidth+1||n.scrollHeight>n.clientHeight+1).length,noteWeight:getComputedStyle(e.querySelector('dd .m01-line:last-child')).fontWeight}));
  assert.equal(layout.noteWeight,reported?'400':'600');assert.equal(layout.localClipping,0);assert.equal(layout.overflow,false);assert.equal(layout.direction,lang==='ar'?'rtl':'ltr');
  await row.screenshot({path:`${args.output}/result-row-${reported}-${lang}-${width}.png`});
  await page.screenshot({path:`${args.output}/result-context-${reported}-${lang}-${width}.png`});
  evidence.cases.push({screen:'M01',lang,width,rechecked:reported,resultPermissionCount:2,documentPermissionCount:2,...layout});
 }
 const grants=JSON.parse(args.grants);
 const grantLabels={propose:['Propose ERP result evidence','ERP 결과 근거 제안','اقتراح أدلة نتائج ERP'],confirm:['Confirm ERP result evidence','ERP 결과 근거 확인','تأكيد أدلة نتائج ERP'],revoke:['Revoke ERP result evidence','ERP 결과 근거 철회','إلغاء أدلة نتائج ERP'],apply:['Apply ERP result evidence internally','ERP 결과 근거 내부 반영','تطبيق أدلة نتائج ERP داخلياً'],inspect:['View cost','원가 조회','عرض التكلفة']};
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390])for(const route of ['access','integrations']){
  await page.setViewportSize({width,height:1000});await page.goto('about:blank');
  await page.goto(`${args.url}/?service=api&lang=${lang}#/${route}?lang=${lang}&view=${route==='access'?'grants':'clients'}`);
  for(const grant of grants){
   const row=route==='access'?page.locator('tr').filter({has:page.locator(`[data-grant="${grant.human}"]`)}):page.locator('.p02-grants tr').filter({hasText:grant.console});
   await row.waitFor();await page.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
   const text=await row.innerText();assert.ok(text.includes(grantLabels[grant.capability][['en','ko','ar'].indexOf(lang)]));
   assert.ok(text.includes({en:'Synthetic ERP result evidence',ko:'합성 ERP 결과 근거',ar:'أدلة نتائج ERP اصطناعية'}[lang]));assert.ok(!text.includes('synthetic_erp_result_evidence'));
   const layout=await row.evaluate(e=>({direction:getComputedStyle(e).direction,overflow:document.documentElement.scrollWidth>innerWidth}));assert.equal(layout.overflow,false);assert.equal(layout.direction,lang==='ar'?'rtl':'ltr');
   await row.screenshot({path:`${args.output}/${route}-${grant.capability}-${lang}-${width}.png`});
   if(grant.capability==='apply')await page.screenshot({path:`${args.output}/${route}-context-${lang}-${width}.png`});
   evidence.cases.push({screen:route==='access'?'M03':'P02',capability:grant.capability,lang,width,...layout});
  }
 }
 assert.equal(evidence.executionRequests,0);assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
} catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};await page.screenshot({path:args.output+'/failure.png'});throw new Error('Result authority impact QA failed');}
finally {await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
