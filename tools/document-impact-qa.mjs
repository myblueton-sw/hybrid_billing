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
const evidence={cases:[],errors:[],executionRequests:0,simulation:['rechecked=false is an injected display condition, not actual document confirmation policy']};
let rechecked=true;
page.on('pageerror',e=>evidence.errors.push(e.message));
await page.route('**/api/**',async route=>{
 const u=new URL(route.request().url());
 if(u.pathname.endsWith('/bulk_revocation.execute')) { evidence.executionRequests++;await route.abort('blockedbyclient');return; }
 const response=await route.fetch({url:args.api+u.pathname+u.search});
 if(!rechecked&&u.pathname===`/api/v1/identity/bulk-revocations/${args.bulk}`){const body=await response.json();const row=body.impact.pending.find(p=>p.store==='document_evidence_proposals');assert.ok(row);row.rechecked=false;await route.fulfill({response,json:body});return;}
 await route.fulfill({response});
});
const labels={en:'Document evidence proposals',ko:'문서 근거 제안',ar:'مقترحات أدلة المستند'};
const permissions={en:'Document evidence and ERP preparation permission proposals',ko:'문서 근거·ERP 준비 권한 제안',ar:'مقترحات صلاحيات أدلة المستند وإعداد ERP'};
const notes={en:'The proposer’s permission is rechecked when the document evidence is confirmed.',ko:'문서 근거 확인 시 제안자의 권한을 다시 확인합니다.',ar:'تُعاد مراجعة صلاحية مقدّم المقترح عند تأكيد أدلة المستند.'};
const unknownNotes={en:'Rechecking the proposer’s permission at document confirmation is not confirmed.',ko:'문서 근거 확인 시 제안자의 권한 재확인 여부는 확인되지 않았습니다.',ar:'لم يتأكد إجراء مراجعة لصلاحية مقدّم المقترح عند تأكيد أدلة المستند.'};
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
  assert.equal(await permission.locator('dd .m01-line').first().innerText(),'2');
  assert.ok((await row.innerText()).includes((reported?notes:unknownNotes)[lang]));
  assert.ok(!(await page.locator('.m01-impact').innerText()).includes('document_evidence_proposals'));
  const layout=await row.evaluate(e=>({direction:getComputedStyle(e).direction,overflow:document.documentElement.scrollWidth>innerWidth,rowHeight:e.getBoundingClientRect().height,localClipping:[...e.querySelectorAll('dt,dd')].filter(n=>n.scrollWidth>n.clientWidth+1||n.scrollHeight>n.clientHeight+1).length,noteWeight:getComputedStyle(e.querySelector('dd .m01-line:last-child')).fontWeight}));
  assert.equal(layout.noteWeight,reported?'400':'600');assert.equal(layout.localClipping,0);assert.equal(layout.overflow,false);assert.equal(layout.direction,lang==='ar'?'rtl':'ltr');
  await row.screenshot({path:`${args.output}/document-row-${reported}-${lang}-${width}.png`});
  await page.screenshot({path:`${args.output}/document-context-${reported}-${lang}-${width}.png`});
  evidence.cases.push({screen:'M01',lang,width,rechecked:reported,documentCount:1,permissionCount:2,...layout});
 }
 assert.equal(evidence.executionRequests,0);assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
} catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};await page.screenshot({path:args.output+'/failure.png'});throw new Error('Document impact QA failed');}
finally {await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
