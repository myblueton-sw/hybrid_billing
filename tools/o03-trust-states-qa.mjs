import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const output=args.output;await mkdir(output,{recursive:true});
const hash='a'.repeat(64),now='2026-11-24T09:00:00Z',on='2026-11-24';
const contract={id:'contract-a',name:'Synthetic contract A',profile:'profile-a',profile_name:'Synthetic profile',buyer:'buyer',payer:'payer',issuer:'issuer',path:'direct',currency:'USD',timezone:'UTC',cycle:'monthly',state:'active',lifecycle_revision:1,settlement:'none',sequence:1,start:'2026-01-01',end:'',open_gaps:0};
const impact={money:'none',proration:'none',publication:'none',erp:'none',final_amount_decision:'none'};
const base={digest:hash,revision:'1',contract_revision:'3',billing_profile:'profile-a',billing_revision:'2',mode:'erp_issuance',profile:'synthetic_erp_document_v1',region:'US',tax:'synthetic-tax',zone:'UTC',languages:'ar,en,ko',proposer:'synthetic-proposer-with-a-long-identifier',proposed_at:'2026-11-20T01:00:00Z'};
const confirmed=id=>({...base,configuration:id,state:'confirmed',module_observation:'active',confirmer:'synthetic-confirmer',setup_on:'2026-11-20',scope_digest:hash,proposal_reason:'Synthetic reason',approval_observed_at:'2026-11-20T02:00:00Z',activated_at:'2026-11-20T02:00:00Z'});
const legacy=id=>{const r=confirmed(id);delete r.setup_on;delete r.scope_digest;delete r.proposal_reason;return r;};
const pending=id=>({...base,configuration:id,state:'proposed',module_observation:'proposed',confirmer:'',setup_on:'2026-11-20',scope_digest:hash,proposal_reason:'Synthetic reason'});
const histories={one:[pending('configuration-a'),confirmed('configuration-b'),legacy('configuration-c')],two:[confirmed('configuration-a'),confirmed('configuration-b')],none:[pending('configuration-a'),legacy('configuration-c')]};
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Built console with intercepted synthetic reads and injected read failures; no server, no mutation, no owner acceptance',cases:[],unexpected:[],errors:[]};
let mode='one',fault={};const release=[];
page.on('pageerror',e=>evidence.errors.push(e.message));
await page.route('**/api/**',async route=>{
 const request=route.request(),path=new URL(request.url()).pathname;
 if(path.endsWith('/session'))return route.fulfill({json:{user:'governor',workspace:'workspace',role:'admin',login:'synthetic',csrf_token:'synthetic'}});
 if(request.method()==='GET'){
  if(/\/contracts\/[^/]+$/.test(path)){if(fault.detail==='hold')await new Promise(r=>release.push(r));if(fault.detail==='missing')return route.fulfill({status:404,json:{code:'not_found'}});return route.fulfill({json:{revision:'r1',today:on,from:'',until:'',impact,contract:{...contract,id:path.split('/').at(-1)},terms:[],events:[],segments:[],reopenings:0,evidence:[],records:[],earlier_records:0,actions:{}}});}
  if(path.endsWith('/contracts')){if(fault.list==='hold')await new Promise(r=>release.push(r));if(fault.list==='fail')return route.fulfill({status:503,json:{code:'dependency_unavailable'}});return route.fulfill({json:{revision:'r1',on,today:on,impact,contracts:[contract]}});}
  if(path.endsWith('/organizations'))return route.fulfill({json:{revision:'r1',parties:[],profiles:[],members:[]}});
  evidence.unexpected.push(path);return route.fulfill({json:{}});
 }
 const command=path.split('/').at(-1),body=request.postDataJSON(),done=fields=>route.fulfill({json:{request_id:body.request_id,status:'completed',target_id:body.target_id,result_revision:'r1',replay:false,as_of:now,fields}});
 if(command==='issuance_setup.context'&&fault.context==='fail')return route.fulfill({status:503,json:{code:'dependency_unavailable',request_id:body.request_id}});
 if(command==='issuance_setup.context')return done({schema:'hb_issuance_setup_context_v1',contract:contract.id,contract_name:contract.name,on:body.body.on,lifecycle_sequence:'1',contract_revision:'3',billing_profile:'profile-a',billing_profile_name:'Synthetic profile',billing_revision:'2',buyer:'buyer',payer:'payer',issuer:'issuer',currency:'USD',scope_digest:hash,profile_basis:'current',current_eligibility:'unchecked'});
 if(command==='issuance_setup.options')return done({schema:'hb_issuance_setup_options_v1',contract:contract.id,on:body.body.on,scope_digest:hash,current_eligibility:'unchecked',option:'synthetic_erp_sample_v1',applicability:'synthetic_only',mode:'erp_issuance',profile:'synthetic_erp_document_v1',region:'US',tax:'synthetic-tax',zone:'UTC',languages:'ar,en,ko'});
 if(command==='issuance_setup.configurations'){const rows=histories[mode];return done({schema:'hb_issuance_configuration_page_v1',contract:contract.id,current_eligibility:'unchecked',snapshot:hash,count:String(rows.length),next:'',...Object.fromEntries(rows.map((r,i)=>[`row_${String(i).padStart(2,'0')}`,JSON.stringify(r)]))});}
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
const go=async(lang,width,id=contract.id)=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/contracts?view=documents&contract=${id}&on=${on}&lang=${lang}`);};
const counter=()=>page.locator('.live-counter').count();
const free=()=>{while(release.length)release.shift()();};
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,390]){
  const shot=name=>page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});
  fault={list:'hold'};await go(lang,width);await page.locator('.o03-list, .live-panel').first().waitFor();await page.waitForTimeout(300);
  assert.equal(await counter(),0,'a count is shown while the list is loading');await shot('list-loading');fault={};free();
  if(width===1440){await page.locator('.live-counter').waitFor();assert.equal(await counter(),1);}
  evidence.cases.push({check:'no count while loading',lang,width});

  fault={list:'fail'};await go(lang,width);await page.locator('.live-error').first().waitFor();{const told=await page.locator('.live-error').first().innerText();assert.equal(/sign you in|로그인|تسجيل الدخول/.test(told),false,'a load failure is worded as a sign-in failure');if(lang==='en')assert.ok(told.includes('Contracts could not be loaded.'));}assert.equal(await counter(),0,'a count is shown after a failed first read');
  if(lang==='en')assert.equal(/\d+ of \d+ contracts/.test(await page.locator('main').innerText()),false,'a contract count is shown after a failed first read');await shot('list-failed');fault={};
  evidence.cases.push({check:'no count after a failed first read',lang,width});

  fault={detail:'hold'};await go(lang,width,'contract-unlisted');await page.locator('.o03-panel').waitFor();await page.waitForTimeout(400);
  assert.equal(await page.locator('.o03-panel .live-note-rule').count(),0,'not available is shown while the detail read is pending');assert.equal(await page.locator('.o03-panel [role=status]').count(),1);await shot('detail-pending');
  fault={detail:'missing'};free();await page.locator('.o03-panel .live-error').waitFor();assert.equal(await page.locator('.o03-panel .live-note-rule').count(),0);await shot('detail-refused');fault={};
  evidence.cases.push({check:'pending detail is not shown as unavailable',lang,width});

  fault={};await go(lang,width);await page.locator('.o03-panel #o03-contract-title, .o03-panel h2, .o03-panel h3').first().waitFor();await page.waitForFunction(()=>!document.querySelector('.o03-panel [role=status].live-meta'));
  assert.equal(await page.locator('.o03-failure').count(),0);assert.equal(await page.locator('.o03-detail-kept').count(),0);
  fault={list:'fail',detail:'missing'};await page.locator('.o03-reload').click();await page.locator('.o03-detail-kept').waitFor();await page.locator('.o03-failure').waitFor();
  assert.equal(await page.locator('.o03-failure p').count()>=1,true);if(lang==='en'){assert.ok((await page.locator('.o03-failure').innerText()).includes('from the last successful load'));assert.ok((await page.locator('.o03-detail-kept').innerText()).includes('Reload before acting on it'));}
  assert.ok(await page.locator('.o03-parts').count()===1,'the held detail was dropped after the failed refresh');await shot('kept-after-failed-reload');
  fault={};await page.locator('.o03-detail-kept button').click();await page.locator('.o03-detail-kept').waitFor({state:'detached'});
  evidence.cases.push({check:'a kept list and detail say they are from the last successful load',lang,width});

  fault={context:'fail'};await go(lang,width);await page.locator('#o03-docs-scope').waitFor();await page.waitForFunction(()=>!document.querySelector('[aria-labelledby=o03-docs-scope][aria-busy=true]'));await page.waitForTimeout(200);
  const sample=page.locator('[aria-labelledby=o03-docs-options]');assert.equal(await sample.locator('.live-sub').count(),1,'the sample section does not say why it is empty');assert.equal(await sample.locator('dl').count(),0);if(lang==='en')assert.ok((await sample.innerText()).includes('Not shown because the setup scope could not be loaded'));assert.ok(await page.locator('[aria-labelledby=o03-docs-scope] [role=alert], [aria-labelledby=o03-docs-scope] .live-error').count()>=1,'the scope failure is not shown');await shot('context-failed');fault={};
  await go(lang,width);await page.locator('[aria-labelledby=o03-docs-options] dl').waitFor();assert.equal(await page.locator('[aria-labelledby=o03-docs-options] .live-sub').count(),1);
  evidence.cases.push({check:'the sample section says why it is empty after a failed scope read',lang,width});
 }
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure=String(e.message).slice(0,500);await page.screenshot({path:`${output}/failure.png`,fullPage:true}).catch(()=>{});}
await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();
if(evidence.outcome!=='passed'){console.error(evidence.failure);process.exit(1);}
