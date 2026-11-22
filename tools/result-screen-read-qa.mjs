import { createRequire } from 'node:module';
import { mkdir, writeFile } from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage();
const hash='a'.repeat(64), evidence={mode:'synthetic intercepted read responses; no backend interoperability claim',cases:[],errors:[],unexpected:[]};
let scenario='normal', discard=true;
const prep={preparation:'read-prep',preparation_digest:hash,document:'read-document',document_digest:hash,connection:'erp',customer:'synthetic-customer',period:'2026-10',currency:'USD',preparation_actor:'synthetic-actor'};
page.on('dialog',d=>discard?d.accept():d.dismiss());
page.on('pageerror',e=>evidence.errors.push(e.message));
await mkdir(args.output,{recursive:true});
await page.route('**/api/**',async route=>{
 const path=new URL(route.request().url()).pathname;
 if(path==='/api/v1/session')return route.fulfill({json:{user:'synthetic-actor',workspace:'synthetic-workspace',role:'admin',login:'synthetic',csrf_token:'synthetic-csrf'}});
 const command=path.split('/').at(-1), body=route.request().postDataJSON();
 if(!['result_evidence.preparations','result_evidence.context','result_evidence.history',...(scenario==='proposal'?['result_evidence.propose']:[])].includes(command)){evidence.unexpected.push(path);return route.fulfill({status:403,json:{code:'permission_denied'}});}
 if(scenario==='denied')return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
 if(command==='result_evidence.propose')return route.fulfill({json:{request_id:body.request_id,target_id:body.target_id,operation_id:'operation',status:'completed',result_revision:'r2',business_state:'pending',replay:false,as_of:'2026-11-22T00:00:00Z',fields:{mode:'synthetic',result:body.target_id,digest:hash,recorded_state:'pending',receipt:'historical_decision',application_state:'not_started',application_receipt:''}}});
 let fields;
 if(command.endsWith('preparations'))fields={mode:'synthetic',workspace:'synthetic-workspace',current_eligibility:'unchecked',snapshot:'r1',next:'',...(scenario==='empty'?{}:{row_00:JSON.stringify(prep)})};
 if(command.endsWith('context')){const {preparation_actor:_,...identity}=prep;fields={...identity,mode:'synthetic',profile:'synthetic_result_one_to_one_v1',schema:'hb_erp_result_packet_v1',permissions:JSON.stringify({propose:{state:scenario==='proposal'?'allowed':'denied',code:scenario==='proposal'?'':'permission_denied'}}),proposal_readiness:JSON.stringify({state:scenario==='proposal'?'ready':'unchecked',code:scenario==='proposal'?'':'permission_denied'}),current_eligibility:'unchecked',snapshot:'r1',next:'',row_00:JSON.stringify({line:'reference-line',expected_product:'reference-product',expected_unit:'hour'})};}
 if(command.endsWith('history'))fields={mode:'synthetic',preparation:prep.preparation,digest:hash,current_eligibility:'unchecked',snapshot:'r1',next:''};
 return route.fulfill({json:{request_id:body.request_id,status:'completed',target_id:body.target_id,result_revision:'r1',replay:false,as_of:'2026-11-22T00:00:00Z',fields}});
});
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  scenario='normal';await page.setViewportSize({width,height:1000});
  await page.goto("about:blank");await page.goto(`${args.url}/?service=api&lang=${lang}#/approvals?lang=${lang}&view=results`);
  const panel=page.locator('.b09-results');await panel.locator('button').filter({hasText:'read-prep'}).click();
  await panel.locator('bdi:visible').filter({hasText:/^reference-product$/}).waitFor();await page.evaluate(()=>document.fonts.ready);
  await page.waitForFunction(()=>document.querySelector('.b09-results-detail > h3')===document.activeElement);
  const layout=await panel.evaluate(el=>({tableOverflow:!!el.querySelector("table").getClientRects().length&&el.querySelector("table").scrollWidth>el.querySelector(".b09-results-table").clientWidth+1,overflow:document.documentElement.scrollWidth>innerWidth,direction:getComputedStyle(el).direction,font:getComputedStyle(el).fontFamily,clipped:[...el.querySelectorAll('button,a')].filter(e=>e.getClientRects().length&&(e.getBoundingClientRect().right>innerWidth+1||e.getBoundingClientRect().left< -1)).length}));
  assert.equal(layout.tableOverflow,false);assert.equal(layout.overflow,false);assert.equal(layout.clipped,0);assert.equal(layout.direction,lang==='ar'?'rtl':'ltr');
  await page.screenshot({path:`${args.output}/read-${lang}-${width}.png`,fullPage:true});evidence.cases.push({lang,width,...layout});
  if(width===390){await panel.locator(".b09-results-back").click();await page.waitForFunction(()=>document.activeElement?.getAttribute("aria-current")==="true"&&document.activeElement?.textContent?.includes("read-prep"));assert.equal(await panel.locator(".b09-results-list").isVisible(),true);await page.screenshot({path:`${args.output}/list-${lang}-${width}.png`,fullPage:true});}
 }
 for(const mode of ['empty','denied']){
  scenario=mode;await page.goto("about:blank");await page.goto(`${args.url}/?service=api&lang=en#/approvals?lang=en&view=results`);
  await page.getByText(mode==='empty'?'No recorded preparations in this workspace.':'Access could not be confirmed. Start a new read after checking your session and permissions.',{exact:true}).waitFor();
  assert.equal(await page.locator('.b09-results-list li').count(),0);await page.screenshot({path:`${args.output}/${mode}.png`,fullPage:true});evidence.cases.push({mode});
 }
 scenario='proposal';await page.goto('about:blank');await page.setViewportSize({width:1440,height:1000});await page.goto(`${args.url}/?service=api&lang=en#/approvals?lang=en&view=results`);
 const panel=page.locator('.b09-results');await panel.locator('button').filter({hasText:'read-prep'}).click();await panel.getByRole('button',{name:'Enter result evidence',exact:true}).click();
 const form=panel.locator('.b09-result-form');await form.waitFor();
 for(const [id,value] of Object.entries({evidence:'reported-source',external:'external-doc',revision:'1','row:reference-line':'reported-row','product:reference-line':'different-product','unit:reference-line':'day','quantity:reference-line':'2','net:reference-line':'3.25','tax:reference-line':'0',rationale:'Independent synthetic source'}))await form.locator(`[id="result-${id}"]`).fill(value);
 discard=false;await panel.getByRole('button',{name:'New read',exact:true}).click();assert.equal(await form.locator('#result-evidence').inputValue(),'reported-source');await panel.getByRole('button',{name:'Close result entry',exact:true}).click();assert.equal(await form.locator('#result-external').inputValue(),'external-doc');discard=true;
 await page.screenshot({path:args.output+'/proposal-form.png',fullPage:true});await form.getByRole('button',{name:'Review proposal',exact:true}).click();
 await panel.locator('.b09-result-review').waitFor();await page.screenshot({path:args.output+'/proposal-review.png',fullPage:true});await panel.getByRole('button',{name:'Send proposal',exact:true}).click();await panel.locator('[data-result-attempt="recorded"]').waitFor();assert.equal(await panel.locator('.b09-result-form').count(),0);evidence.cases.push({mode:'proposal',reportedMismatchRetained:true,reviewed:true,recorded:true});
 assert.deepEqual(evidence.errors,[]);assert.deepEqual(evidence.unexpected,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure=e.message;evidence.focus=await page.evaluate(()=>({active:document.activeElement?.outerHTML,selected:document.querySelector('.b09-results')?.getAttribute('data-selected')}));await page.screenshot({path:args.output+'/failure.png',fullPage:true});throw e;}
finally{await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
