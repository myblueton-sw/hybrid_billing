import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const cookie=process.env.HB_PROJECTION_QA_COOKIE;
delete process.env.HB_PROJECTION_QA_COOKIE;
assert.ok(cookie&&args.url&&args.api&&args.output&&args.chromium);
await mkdir(args.output,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const context=await browser.newContext();
const split=cookie.indexOf('=');
await context.addCookies([{name:cookie.slice(0,split),value:cookie.slice(split+1),url:args.url,httpOnly:true,sameSite:'Strict'}]);
const page=await context.newPage();
const evidence={errors:[]};
page.on('pageerror',e=>evidence.errors.push(e.message));
await page.route('**/api/**',async route=>{const u=new URL(route.request().url());const response=await route.fetch({url:args.api+u.pathname+u.search});await route.fulfill({response});});
try {
 await page.goto(`${args.url}/?service=api&lang=en#/approvals?lang=en&view=history&approval-q=qa-current`);
 evidence.integration=await page.evaluate(async()=>{
  const {serviceClient}=await import('/src/service/api.ts');
  const {decisionClient}=await import('/src/modules/issuance/pages/approval-inbox/decision-api.ts');
  const {DecisionStore}=await import('/src/modules/issuance/pages/approval-inbox/decision-store.ts');
  const session=await serviceClient().current();
  const read=await fetch('/api/approval-review?view=history&q=qa-current');const data=await read.json();
  const record=data.records.find(r=>r.id==='qa-current');
  if(record?.status!=='pending')throw new Error('fixture_not_pending');
  let sends=0;const bodies=[];
  const wrapped=async(input,init)=>{
   const response=await fetch(input,init);
   if(String(input).endsWith('/financial_charge.approve')){bodies.push(init.body);sends++;if(sends===1)throw new Error('synthetic_lost_response');}
   return response;
  };
  const client=decisionClient(session,wrapped);
  const prepared=await client.prepare(record);
  if(prepared.approve!=='eligible'||prepared.reject!=='eligible'||prepared.amount!=='15')throw new Error('preparation_mismatch');
  const store=new DecisionStore(client);
  store.select('actual-service',record);await store.check();store.choose('approve');store.edit('Synthetic actual service decision recovery','');
  const pending=store.submit();await store.submit();await pending;
  if(sends!==1||store.snapshot().entries['qa-current']?.phase!=='unknown')throw new Error('unknown_or_duplicate_failed');
  store.select('different-view');store.select('actual-service',record);
  await store.retry('qa-current');
  const result=store.snapshot().entries['qa-current'];
  if(result?.phase!=='recorded'||result.receipt?.replay!==true||sends!==2||bodies[0]!==bodies[1])throw new Error('replay_not_identical');
  const after=await (await fetch('/api/approval-review?view=history&q=qa-current')).json();
  if(after.records[0]?.status!=='approved'||after.records[0]?.approver?.id!==session.user)throw new Error('effect_not_recorded');
  return {actualPreparation:true,exactAmount:'USD 15',lostResponseAfterEffect:true,duplicateGuard:true,sameAttemptReplay:true,selectionPreservedUnknown:true,recordedState:'approved',screenDecisionControls:false};
 });
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(error){evidence.outcome='failed';evidence.failure={kind:error.name,detail:error.message.split('\n')[0].slice(0,160)};throw new Error('Decision core integration failed');}
finally{await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
