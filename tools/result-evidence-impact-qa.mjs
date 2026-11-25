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
const evidence={cases:[],errors:[],executionRequests:0,policyCases:[],simulation:['rechecked=false is an injected display condition, not an actual recovery policy change']};
let rechecked=true;
page.on('pageerror',e=>evidence.errors.push('page_runtime_error'));
await page.route('**/api/**',async route=>{
 const u=new URL(route.request().url());
 if(u.pathname.endsWith('/bulk_revocation.execute')) { evidence.executionRequests++;await route.abort('blockedbyclient');return; }
 const response=await route.fetch({url:args.api+u.pathname+u.search});
 if(!rechecked&&u.pathname===`/api/v1/identity/bulk-revocations/${args.bulk}`){const body=await response.json();const row=body.impact.pending.find(p=>p.store==='result_applications_unknown');assert.ok(row);row.rechecked=false;await route.fulfill({response,json:body});return;}
 await route.fulfill({response});
});
const labels={"en": "ERP result evidence applications with unknown outcome", "ko": "ERP 결과 근거 내부 반영 결과 미확정", "ar": "عمليات تطبيق أدلة نتائج ERP داخلياً ذات نتيجة غير مؤكدة"};
const permissions={"en": "ERP result evidence proposals", "ko": "ERP 결과 근거 제안", "ar": "مقترحات أدلة نتائج ERP"};
const finalPolicies={en:'ERP final-amount approval policy proposals',ko:'ERP 최종 금액 승인 정책 제안',ar:'مقترحات سياسة اعتماد المبلغ النهائي في ERP'};
const finalLimits={en:'ERP final-amount approval limit proposals',ko:'ERP 최종 금액 승인 한도 제안',ar:'مقترحات حدود اعتماد المبلغ النهائي في ERP'};
const finalDecisionPermissions={en:'ERP final-amount approval permission proposals',ko:'ERP 최종 금액 승인 권한 제안',ar:'مقترحات صلاحية اعتماد المبلغ النهائي في ERP'};
const finalRequestPermissions={en:'ERP final-amount approval submission permission proposals',ko:'ERP 최종 금액 승인 요청 제출 권한 제안',ar:'مقترحات صلاحية تقديم طلب اعتماد المبلغ النهائي في ERP'};
const chargePolicies={en:'Charge approval policy proposals',ko:'청구 승인 정책 제안',ar:'مقترحات سياسة اعتماد الرسوم'};
const notes={"en": "Recovery rechecks the applying actor’s permission and current evidence. Earlier effects remain possible.", "ko": "복구 시 반영 수행자의 권한과 현재 근거를 다시 확인합니다. 이전 처리는 이미 반영되었을 수 있습니다.", "ar": "تُراجع عند الاستعادة صلاحية منفّذ التطبيق والأدلة الحالية. قد تكون آثار المحاولة السابقة قد وقعت."};
const unknownNotes={"en": "Recovery checks are unconfirmed. Earlier effects remain possible.", "ko": "복구 시 재확인 여부는 확인되지 않았습니다. 이전 처리는 이미 반영되었을 수 있습니다.", "ar": "لم تتأكد إعادة التحقق عند الاستعادة. قد تكون آثار المحاولة السابقة قد وقعت."};
try {
 for(const reported of [true,false])for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  rechecked=reported;
  await page.setViewportSize({width,height:1000});await page.goto('about:blank');
  await page.goto(`${args.url}/?service=api&lang=${lang}#/users?lang=${lang}&view=revocation`);
  await page.locator(`[data-job="${args.bulk}"]`).click();
  const row=page.locator('.m01-impact .m01-sum > div').filter({has:page.getByText(labels[lang],{exact:true})});
  const permission=page.locator('.m01-impact .m01-sum > div').filter({has:page.getByText(permissions[lang],{exact:true})});
  await row.waitFor();await page.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
  assert.equal(await row.locator('dd .m01-line').first().innerText(),'1');
  assert.equal(await permission.locator('dd .m01-line').first().innerText(),'1');
  assert.ok((await row.innerText()).includes((reported?notes:unknownNotes)[lang]));
  assert.ok(!(await page.locator('.m01-impact').innerText()).includes('result_applications_unknown'));
  for(const names of [finalPolicies,chargePolicies,finalLimits,finalDecisionPermissions,finalRequestPermissions]){
   const policy=page.locator('.m01-impact .m01-sum > div').filter({has:page.getByText(names[lang],{exact:true})});
   const expectedCount=names===chargePolicies||names===finalRequestPermissions?2:1;
   assert.equal(await policy.locator('dd .m01-line').first().innerText(),String(expectedCount));
   const bounds=await policy.evaluate(e=>({overflow:[...e.querySelectorAll('dt,dd')].some(n=>n.scrollWidth>n.clientWidth+1),direction:getComputedStyle(e).direction,borderColor:getComputedStyle(e).borderColor,borderStyle:getComputedStyle(e).borderStyle}));
   assert.equal(bounds.overflow,false);assert.equal(bounds.direction,lang==='ar'?'rtl':'ltr');
   const stage=names===finalPolicies?'final':names===chargePolicies?'charge':names===finalLimits?'final-limit':names===finalDecisionPermissions?'final-decision-permission':'final-request-permission';
   await policy.screenshot({path:`${args.output}/policy-${stage}-${reported}-${lang}-${width}.png`});
   evidence.policyCases.push({lang,width,stage,rechecked:reported,count:expectedCount,label:names[lang],...bounds});
  }
  assert.ok(!(await page.locator('.m01-impact').innerText()).includes('erp_final_policy_proposals'));
  const layout=await row.evaluate(e=>({direction:getComputedStyle(e).direction,overflow:document.documentElement.scrollWidth>innerWidth,rowHeight:e.getBoundingClientRect().height,localClipping:[...e.querySelectorAll('dt,dd')].filter(n=>n.scrollWidth>n.clientWidth+1||n.scrollHeight>n.clientHeight+1).length,noteWeight:getComputedStyle(e.querySelector('dd .m01-line:last-child')).fontWeight}));
  assert.equal(layout.noteWeight,reported?'400':'600');assert.equal(layout.localClipping,0);assert.equal(layout.overflow,false);assert.equal(layout.direction,lang==='ar'?'rtl':'ltr');
  await row.screenshot({path:`${args.output}/result-row-${reported}-${lang}-${width}.png`});
  await page.screenshot({path:`${args.output}/result-context-${reported}-${lang}-${width}.png`});
  evidence.cases.push({screen:'M01',lang,width,rechecked:reported,unknownApplications:1,pendingEvidence:1,...layout});
 }
 assert.equal(evidence.executionRequests,0);assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
} catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};await page.screenshot({path:args.output+'/failure.png'});throw new Error('Result authority impact QA failed');}
finally {await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
