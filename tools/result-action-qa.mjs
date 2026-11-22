import {createRequire} from 'node:module';
import {readFile,mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const fixture=JSON.parse(await readFile(args.fixture,'utf8'));
const prep=JSON.parse(fixture.preparations.fields.row_00),result=JSON.parse(fixture.history.fields.row_00),workspace=fixture.preparations.fields.workspace;
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const browser=await chromium.launch({headless:true,executablePath:args.chromium}),page=await browser.newPage({viewport:{width:1440,height:1000}});
const evidence={mode:'real server-produced synthetic read fixtures; intercepted application outcomes',errors:[],requests:[],cases:[]};
let state='not_started';const receipt='b'.repeat(64);
await mkdir(args.output,{recursive:true});page.on('pageerror',e=>evidence.errors.push(e.message));page.on('dialog',d=>d.accept());
await page.route('**/api/**',async route=>{
 const path=new URL(route.request().url()).pathname;
 if(path.endsWith('/session'))return route.fulfill({json:{user:prep.preparation_actor,workspace,role:'admin',login:'synthetic',csrf_token:'synthetic'}});
 const command=path.split('/').at(-1),body=route.request().postDataJSON();
 if(command==='erp_result.apply'){
  evidence.requests.push({command,key:body.idempotency_key,target:body.target_id,digest:body.body.digest});
  if(state==='not_started'){state='unknown';return route.fulfill({status:503,json:{request_id:body.request_id,operation_id:'first-operation',code:'dependency_unavailable',retry:false,next_action:'inspect_current_state'}});}
  state='recorded';return route.fulfill({json:{request_id:body.request_id,target_id:body.target_id,status:'completed',operation_id:'recovery-operation',result_revision:'r2',business_state:'recorded',replay:false,as_of:'2026-11-22T00:00:00Z',fields:{mode:'synthetic',result:result.result,digest:result.digest,recorded_state:'confirmed',receipt:'historical_decision',application_state:'recorded',application_receipt:receipt}}});
 }
 const key={'result_evidence.preparations':'preparations','result_evidence.context':'context','result_evidence.history':'history','result_evidence.inspect':'inspection','result_evidence.advice':'advice'}[command];
 assert.ok(key,`unexpected API ${path}`);const reply=structuredClone(fixture[key]);reply.request_id=body.request_id;reply.target_id=body.target_id;
 if(key==='history'){const row=JSON.parse(reply.fields.row_00);row.application_state=state;row.application_receipt=state==='recorded'?receipt:'';reply.fields.row_00=JSON.stringify(row);}
 if(key==='inspection'||key==='advice'){reply.fields.application_state=state;reply.fields.application_receipt=state==='recorded'?receipt:'';}
 if(key==='advice'){reply.fields.applying_actor=state==='not_started'?'':prep.preparation_actor;const readiness=JSON.parse(reply.fields.readiness);readiness.apply={state:state==='not_started'?'ready':'inapplicable',code:state==='not_started'?'':'prerequisite_failed'};readiness.recover={state:state==='unknown'?'ready':'inapplicable',code:state==='unknown'?'':'prerequisite_failed'};reply.fields.readiness=JSON.stringify(readiness);}
 return route.fulfill({json:reply});
});
async function inspect(){await page.locator('.b09-results-list button').filter({hasText:prep.preparation}).click();await page.getByRole('button',{name:`Inspect · ${result.result}`,exact:true}).click();await page.locator('.b09-result-inspection').waitFor();}
try{
 await page.goto(`${args.url}/?service=api&lang=en#/approvals?lang=en&view=results`);await inspect();
 const detail=()=>page.locator('.b09-result-inspection');await detail().locator('textarea').fill('Apply reviewed synthetic result');await detail().getByRole('button',{name:'Review · Apply internally',exact:true}).click();await detail().locator('.b09-result-action-review').waitFor();await page.screenshot({path:args.output+'/apply-review.png',fullPage:true});await detail().locator('.b09-result-action-review').getByRole('button',{name:'Apply internally',exact:true}).click();await page.locator('[data-result-attempt="unknown"]').waitFor();
 await page.getByRole('button',{name:'New read',exact:true}).click();await inspect();await detail().locator('textarea').fill('Recover the original synthetic result');await detail().getByRole('button',{name:'Review · Recover internal application',exact:true}).click();await detail().locator('.b09-result-action-review').getByRole('button',{name:'Recover internal application',exact:true}).click();await page.locator('[data-result-attempt="recorded"]').waitFor();assert.equal(await page.locator('[data-result-attempt="unknown"]').count(),1);assert.equal(await page.getByRole('button',{name:'Resend identical request',exact:true}).count(),0);
 assert.equal(evidence.requests.length,2);assert.notEqual(evidence.requests[0].key,evidence.requests[1].key);assert.equal(evidence.requests[0].target,evidence.requests[1].target);assert.equal(evidence.requests[0].digest,evidence.requests[1].digest);assert.deepEqual(evidence.errors,[]);await page.screenshot({path:args.output+'/recovered.png',fullPage:true});evidence.cases.push('inspection-advice','reviewed-apply','unknown-preserved','new-key-owned-recovery','settled-old-retry-hidden');evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure=e.message;await page.screenshot({path:args.output+'/failure.png',fullPage:true});throw e;}finally{await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
