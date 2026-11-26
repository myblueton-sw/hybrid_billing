import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const output=args.output;await mkdir(output,{recursive:true});
const hash='a'.repeat(64),now='2026-11-24T09:00:00Z',on='2026-11-24';
const contract={id:'contract-a',name:'Synthetic contract A',profile:'profile-a',profile_name:'Synthetic profile',buyer:'buyer',payer:'payer',issuer:'issuer',path:'direct',currency:'USD',timezone:'UTC',cycle:'monthly',state:'active',lifecycle_revision:1,settlement:'none',sequence:1,start:'2026-01-01',end:'',open_gaps:0};
const impact={money:'none',proration:'none',publication:'none',erp:'none',final_amount_decision:'none'};
const configuration={configuration:'configuration-a',digest:hash,revision:'1',contract_revision:'3',billing_profile:'profile-a',billing_revision:'2',mode:'erp_issuance',profile:'synthetic_erp_document_v1',region:'US',tax:'synthetic-tax',zone:'UTC',languages:'ar,en,ko',proposer:'synthetic-proposer',proposed_at:'2026-11-20T01:00:00Z',state:'confirmed',module_observation:'active',confirmer:'synthetic-confirmer',setup_on:'2026-11-20',scope_digest:hash,proposal_reason:'Synthetic reason',approval_observed_at:'2026-11-20T02:00:00Z',activated_at:'2026-11-20T02:00:00Z'};
const catalog=[['synthetic_recipient_a','document_recipient'],['synthetic_recipient_b','document_recipient'],['synthetic_template_a','document_template'],['synthetic_template_b','document_template'],['synthetic_account_a','payment_account'],['synthetic_account_b','payment_account'],['synthetic_channel_customer_a','channel_customer'],['synthetic_channel_customer_b','channel_customer'],['synthetic_channel_document_kind_a','channel_document_kind'],['synthetic_channel_document_kind_b','channel_document_kind'],['synthetic_channel_entity_a','channel_entity'],['synthetic_channel_entity_b','channel_entity'],['synthetic_unregistered_sender','unregistered_sender']];
const group=(prefix,kind,extra={})=>({[`${prefix}_subject_kind`]:kind,[`${prefix}_changes`]:'0',[`${prefix}_other_provenance`]:'0',[`${prefix}_unowned_module`]:'0',[`${prefix}_latest_scheduled`]:'',[`${prefix}_latest_scheduled_effective`]:'',...extra});
const rules={rules_state:'confirmed',rules_observation:'active',rules_id:'synthetic-rules',rules_digest:hash,rules_approvals:'1',rules_validity_seconds:'172800',rules_max_lead_seconds:'604800',rules_channel_age_seconds:'86400',rules_proposer:'synthetic-proposer',rules_confirmer:'synthetic-confirmer',rules_pending_confirmer:'',rules_proposed_at:'2026-11-21T01:00:00Z',rules_confirmed_at:'2026-11-21T02:00:00Z'};
const channel=(id,kind)=>({channel:id,digest:'c'.repeat(64),subject_kind:'entity',kind,state:'admitted',observation:'admitted',registrar:'synthetic-registrar',admitter:'synthetic-admitter',pending_admitter:'',registered_at:'2026-11-20T01:00:00Z',admitted_at:'2026-11-20T02:00:00Z',retired_by:'',retired_at:'',rules:'current',request_not_before:'2026-11-21T02:00:00Z'});
const channels=[channel('channel-a','portal'),channel('channel-b','email'),{...channel('channel-c','phone'),subject_kind:'customer'}];
const change=(id,extra={})=>({change:id,digest:hash,field:'payment_account',phase:'recorded',state:'requested',observation:'requested',provenance:'staff_entry',instruction_kind:'in_person',channel:'channel-a',rules:'current',requester:'synthetic-requester',requested_at:'2026-11-22T04:00:00Z',effective:'2026-12-01T00:00:00Z',approvals_required:'1',verifier:'',verification_outcome:'',verified_at:'',pending_verifier:'',pending_decision_actor:'',ended_by:'',ended_at:'',decisions:[],...extra});
const verified={state:'confirmed',observation:'confirmed',verifier:'synthetic-verifier',verification_outcome:'confirmed',verified_at:'2026-11-22T05:00:00Z'};
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Built console with an intercepted synthetic service that keeps state in the check itself; no server, no real mutation, no owner acceptance',cases:[],unexpected:[],errors:[],writes:[]};
let world,fault={};
const reset=(changes=[])=>{world={changes,claims:[],revision:1};fault={};evidence.writes.length=0;};
page.on('pageerror',e=>evidence.errors.push(e.message));
await page.route('**/api/**',async route=>{
 const request=route.request(),path=new URL(request.url()).pathname;
 if(path.endsWith('/session'))return route.fulfill({json:{user:'governor',workspace:'workspace',role:'admin',login:'synthetic',csrf_token:'synthetic'}});
 if(request.method()==='GET'){
  if(/\/contracts\/[^/]+$/.test(path))return route.fulfill({json:{revision:'r1',today:on,from:'',until:'',impact,contract,terms:[],events:[],segments:[],reopenings:0,evidence:[],records:[],earlier_records:0,actions:{}}});
  if(path.endsWith('/contracts'))return route.fulfill({json:{revision:'r1',on,today:on,impact,contracts:[contract]}});
  if(path.endsWith('/organizations'))return route.fulfill({json:{revision:'r1',parties:[],profiles:[],members:[]}});
  evidence.unexpected.push(path);return route.fulfill({json:{}});
 }
 const command=path.split('/').at(-1),body=request.postDataJSON(),revision=`r${world.revision}`,scoped={contract:contract.id,configuration:'configuration-a',digest:hash,current_eligibility:'unchecked'};
 const done=fields=>route.fulfill({json:{request_id:body.request_id,status:'completed',target_id:body.target_id,result_revision:revision,replay:false,as_of:now,fields}});
 const rows=list=>Object.fromEntries(list.map((r,i)=>[`row_${String(i).padStart(2,'0')}`,JSON.stringify(r)]));
 if(command==='issuance_setup.configurations')return done({schema:'hb_issuance_configuration_page_v1',contract:contract.id,current_eligibility:'unchecked',snapshot:hash,count:'1',next:'',row_00:JSON.stringify(configuration)});
 if(command==='issuance_setup.protected')return done({schema:'hb_issuance_protected_setup_v1',...scoped,channels:'3',channels_other_provenance:'0',channels_unowned_module:'0',...rules,...group('recipient','customer'),...group('template','document_kind'),...group('account','entity',{account_changes:String(world.changes.length)})});
 if(command==='issuance_setup.protected_changes'){const list=body.body.field==='payment_account'?world.changes:[];return done({schema:'hb_issuance_protected_change_page_v1',...scoped,field:body.body.field,snapshot:String(world.revision).padStart(64,'e'),count:String(list.length),next:'',other_provenance:'0',unowned_module:'0',...rows(list)});}
 if(command==='issuance_setup.protected_channels')return done({schema:'hb_issuance_protected_channel_page_v1',...scoped,snapshot:hash,count:String(channels.length),next:'',other_provenance:'0',unowned_module:'0',...rows(channels)});
 if(command==='issuance_setup.references')return done({schema:'hb_issuance_setup_references_v1',...scoped,...Object.fromEntries(catalog.map(([code,use],i)=>[`reference_${String(i).padStart(2,'0')}`,JSON.stringify({code,use,applicability:'synthetic_only'})]))});
 const operation=command.replace('issuance_setup.',''),a=body.body,isClaim=/^attest_/.test(operation),isChange=['request_change','verify_change','decide_change'].includes(operation);
 if(isClaim||isChange){
  evidence.writes.push({operation,body:a,rationale:body.rationale,key:body.idempotency_key,request:body.request_id,expected:body.expected_revision,target:body.target_id});
  const mode=fault[isClaim?'claim':'action'];
  if(mode==='prerequisite')return route.fulfill({status:422,json:{code:'prerequisite_failed',retry:false,next_action:'correct_request',request_id:body.request_id}});
  if(mode==='unknown')return route.fulfill({status:503,json:{code:'dependency_unavailable',retry:true,next_action:'retry_later',request_id:body.request_id,operation_id:'operation-unknown'}});
  world.revision++;const result=`r${world.revision}`,reply=(business,fields)=>route.fulfill({json:{request_id:body.request_id,status:'completed',operation_id:`operation-${world.revision}`,target_id:body.target_id,result_revision:result,business_state:business,replay:false,as_of:now,fields}});
  if(isClaim)return reply('recorded',{schema:'hb_issuance_setup_attestation_v1',contract:contract.id,configuration:'configuration-a',attestation:a.attestation,purpose:operation.replace('attest_',''),target:a.target,actor:'governor',recorded_at:now,digest:hash,state:'unconsumed',assurance:'human_attestation'});
  let state;
  if(operation==='request_change'){world.changes=[...world.changes,change(a.change,{requester:'governor',channel:a.channel,provenance:a.provenance,effective:a.effective,instruction_kind:a.instruction==='in_person'?'in_person':'email'})];state='requested';}
  if(operation==='verify_change'){state=a.outcome==='confirmed'?'confirmed':'rejected';world.changes=world.changes.map(c=>c.change===a.change?{...c,state,observation:state,verifier:'governor',verification_outcome:a.outcome,verified_at:now,...(state==='rejected'?{ended_at:now,ended_by:'governor'}:{})}:c);}
  if(operation==='decide_change'){state=a.verdict==='approve'?'scheduled':'rejected';world.changes=world.changes.map(c=>c.change===a.change?{...c,state,observation:state,decisions:[...c.decisions,{approver:'governor',verdict:a.verdict,at:now}],...(state==='rejected'?{ended_at:now,ended_by:'governor'}:{})}:c);}
  return reply(state,{schema:'hb_issuance_protected_change_receipt_v1',contract:contract.id,configuration:'configuration-a',current_eligibility:'unchecked',change:a.change,change_digest:hash,state,phase:'recorded'});
 }
 if(['issuance_setup.context','issuance_setup.options'].includes(command))return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
const steps=page.locator('.o03-part-sub.o03-steps'),actions=page.locator('#o03-protected-results').locator('..'),claims=page.locator('#o03-statement-results').locator('..'),button=name=>steps.getByRole('button',{name,exact:true});
const idle=()=>page.waitForFunction(()=>!document.querySelector('[aria-busy=true]'));
const rowsReady=n=>page.waitForFunction(n=>document.querySelectorAll('.o03-changes-table tbody tr').length===n,n);
const open=async(lang,width,n)=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/contracts?view=documents&contract=${contract.id}&on=${on}&part=changes&lang=${lang}`);await page.locator('.o03-field-cards > li').nth(2).locator('button').click();if(n)await rowsReady(n);else await page.locator('.o03-part-sub .live-empty').waitFor();await page.locator('#o03-change-steps').waitFor();await idle();};
const active=()=>page.evaluate(()=>document.activeElement?.id||document.activeElement?.tagName);
const noOverflow=async where=>assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`overflow ${where}`);
const controls=async where=>{const sizes=await steps.evaluate(el=>[...el.querySelectorAll('button, input:not([type=radio]), textarea, select, .live-check')].map(e=>Math.round(e.getBoundingClientRect().height)));assert.ok(sizes.every(h=>h>=44),`targets ${where} ${sizes}`);};
const guarded=()=>page.evaluate(()=>{const e=new Event('beforeunload',{cancelable:true});window.dispatchEvent(e);return e.defaultPrevented;});
const rowButton=()=>page.locator('.o03-changes-table tbody button');
const words={
 en:{request:'Request protected settings change: Payment account',send:'Request protected settings change',verify:'Verify protected settings change',decide:'Decide protected settings change',review:'Review before sending',retry:'Retry exact request'},
 ko:{request:'보호된 설정 변경 요청: 결제 계좌',send:'보호된 설정 변경 요청',verify:'보호된 설정 변경 검증',decide:'보호된 설정 변경 결정',review:'전송 전 검토',retry:'동일 요청 재시도'},
 ar:{request:'طلب تغيير الإعدادات المحمية: حساب الدفع',send:'طلب تغيير الإعدادات المحمية',verify:'التحقق من تغيير الإعدادات المحمية',decide:'البتّ في تغيير الإعدادات المحمية',review:'المراجعة قبل الإرسال',retry:'إعادة محاولة الطلب نفسه'},
};
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  const where=`${lang} ${width}`,w=words[lang],shot=name=>page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});
  reset();await open(lang,width,0);
  await button(w.request).click();await page.locator('#o03-change-channel').waitFor();assert.equal(await steps.locator('input[type=radio]:checked').count(),0,'a choice is preselected');assert.equal(await page.locator('#o03-change-channel').inputValue(),'');assert.equal(await page.locator('#o03-change-effective').inputValue(),'');
  assert.equal(await steps.locator('input[name=o03-change-reference]').count(),2);assert.equal(await page.locator('#o03-change-channel option').count(),3,'a channel of another subject is offered');
  await button(w.review).click();await steps.locator('.live-error').waitFor();assert.ok((await steps.locator('.live-error').innerText()).includes('invalid_setup_request'));assert.equal(evidence.writes.length,0);await shot('request-invalid');
  await steps.locator('input[name=o03-change-reference]').nth(0).check();await page.locator('#o03-change-channel').selectOption('channel-a');await steps.locator('input[name=o03-change-provenance]').nth(1).check();await steps.locator('input[name=o03-change-instruction]').nth(1).check();
  await page.locator('#o03-change-instruction-channel').waitFor();assert.deepEqual(await page.locator('#o03-change-instruction-channel option').evaluateAll(list=>list.map(o=>o.value)),['','channel-b'],'the verification channel is offered as the instruction channel');
  await page.locator('#o03-change-instruction-channel').selectOption('channel-b');await page.locator('#o03-change-effective').fill('2026-12-01T09:30');await page.locator('#o03-change-reason').fill('Synthetic reason: the customer changed bank');await page.locator('#o03-request-statement').fill('Synthetic statement: I received the instruction through the registered email channel.');
  await controls(where);await noOverflow(where);await shot('request-form');
  await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();assert.equal(await active(),'H5');
  const reviewText=await steps.locator('.o03-setup-review').innerText();assert.ok(!reviewText.includes('synthetic_account'),'a sample code is shown');assert.ok(reviewText.includes('Synthetic reason: the customer changed bank'));
  if(lang==='en'){for(const expected of ['Statement you will record','Step it is written for','Two requests are sent','Sample payment account A','Through a registered channel','Customer document','Dec 1, 2026, 9:30 AM UTC','Payment account','Issuing entity · Portal','Issuing entity · Email','The verification channel and the effective time cannot be changed after the request'])assert.ok(reviewText.includes(expected),expected);}
  await noOverflow(where+' review');await shot('request-review');
  await steps.locator('.o03-setup-review').getByRole('button',{name:w.send,exact:true}).click();await rowsReady(1);await idle();
  assert.deepEqual(evidence.writes.map(v=>v.operation),['attest_change_instruction','request_change']);const [said,did]=evidence.writes;
  assert.equal(said.body.target_digest,undefined);assert.equal(did.body.change,said.body.target);assert.equal(did.body.attestation,said.body.attestation);assert.equal(did.expected,'r2');assert.equal(did.rationale,'Synthetic reason: the customer changed bank');
  assert.deepEqual(Object.fromEntries(Object.entries(did.body).filter(([k])=>!['change','attestation'].includes(k))),{configuration:'configuration-a',digest:hash,field:'payment_account',reference:'synthetic_account_a',channel:'channel-a',channel_digest:'c'.repeat(64),provenance:'customer_document',instruction:'registered_channel',instruction_channel:'channel-b',effective:'2026-12-01T09:30:00Z'});
  assert.equal(await rowButton().count(),0,'the requester is offered verification');if(lang==='en')assert.ok((await page.locator('.o03-changes-table').innerText()).includes('You requested this change'));
  assert.equal(await guarded(),false);await noOverflow(where+' requested');await shot('requested');evidence.cases.push({check:'request',lang,width});

  reset([change('change-seeded')]);await open(lang,width,1);
  await rowButton().click();await page.locator('#o03-verify-statement').waitFor();assert.equal(await active(),'o03-change-steps');
  await page.waitForFunction(()=>document.querySelector('.o03-part-sub.o03-steps')?.textContent.includes('channel-a'));const target=await steps.innerText();assert.ok(target.includes('channel-a'));if(lang==='en'){assert.ok(target.includes('Issuing entity · Portal'),'the verification channel is named only by its identifier');assert.ok(target.includes('The service refuses one person performing more than one of these steps on the same change.'));assert.ok(target.includes('What you found'));}
  await button(w.review).click();await steps.locator('.live-error').waitFor();assert.ok((await steps.locator('.live-error').innerText()).includes('invalid_verify'));
  await steps.locator('input[name=o03-change-outcome]').nth(1).check();if(lang==='en')assert.ok((await steps.innerText()).includes('A denied verification ends the change.'));
  await steps.locator('input[name=o03-change-outcome]').nth(0).check();await page.locator('#o03-verify-statement').fill('Synthetic statement: I called back through the portal channel.');await controls(where+' verify');await shot('verify-form');
  await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();await shot('verify-review');
  await steps.locator('.o03-setup-review').getByRole('button',{name:w.verify,exact:true}).click();await actions.locator('.o03-setup-outcome .o03-tag').waitFor();await rowsReady(1);await idle();
  assert.deepEqual(evidence.writes.map(v=>v.operation),['attest_change_verification','verify_change']);assert.equal(evidence.writes[0].body.target,'change-seeded');assert.equal(evidence.writes[0].body.target_digest,hash);
  assert.deepEqual({...evidence.writes[1].body,attestation:''},{configuration:'configuration-a',digest:hash,change:'change-seeded',change_digest:hash,outcome:'confirmed',attestation:''});assert.equal(evidence.writes[1].rationale,undefined);
  assert.equal(await rowButton().count(),0,'the verifier is offered the decision');
  if(lang==='en'){assert.ok((await page.locator('.o03-changes-table').innerText()).includes('You recorded the verification of this change'));assert.ok((await actions.innerText()).includes('Verification recorded, awaiting decision'));}
  await shot('verified');evidence.cases.push({check:'verify',lang,width});

  reset([change('change-seeded',verified)]);await open(lang,width,1);
  await rowButton().click();await steps.locator('input[name=o03-change-verdict]').first().waitFor();assert.equal(await steps.locator('textarea').count(),0,'a decision asks for a statement');
  await button(w.review).click();await steps.locator('.live-error').waitFor();assert.ok((await steps.locator('.live-error').innerText()).includes('invalid_decide'));
  await steps.locator('input[name=o03-change-verdict]').nth(1).check();assert.equal(await steps.locator('input[name=o03-change-rejection]').count(),4);await steps.locator('input[name=o03-change-rejection]').nth(1).check();
  await controls(where+' decide');await shot('decide-form');
  await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();if(lang==='en'){const text=await steps.locator('.o03-setup-review').innerText();assert.ok(text.includes('A decision cannot be changed'));assert.ok(text.includes('Missing supporting material'));assert.ok(!text.includes('Two requests are sent'));}
  await shot('decide-review');
  fault={action:'unknown'};await steps.locator('.o03-setup-review').getByRole('button',{name:w.decide,exact:true}).click();await actions.getByRole('button',{name:w.retry,exact:true}).waitFor();await idle();await page.locator('#o03-change-blocked').waitFor();
  assert.deepEqual(evidence.writes.map(v=>[v.operation,v.body.verdict,v.body.reason,v.body.attestation,v.rationale]),[['decide_change','reject','missing_evidence',undefined,undefined]]);assert.equal(await rowButton().isDisabled(),true);assert.equal(await guarded(),true);
  if(lang==='en'){const told=await actions.innerText();assert.ok(told.includes('Outcome unknown.'));assert.ok(!told.includes('statement recorded for this step'),'a decision is given a statement note');}
  await shot('decide-unknown');
  fault={};await actions.getByRole('button',{name:w.retry,exact:true}).click();await rowButton().waitFor({state:'detached'});await idle();
  assert.equal(evidence.writes.length,2);assert.equal(evidence.writes[1].key,evidence.writes[0].key);assert.equal(await guarded(),false);
  if(lang==='en')assert.ok((await page.locator('.o03-changes-table').innerText()).includes('You already recorded a decision on this change')===false);
  reset([change('change-seeded',verified)]);await open(lang,width,1);await rowButton().click();await steps.locator('input[name=o03-change-verdict]').nth(0).check();assert.equal(await steps.locator('input[name=o03-change-rejection]').count(),0);
  await button(w.review).click();await steps.locator('.o03-setup-review').getByRole('button',{name:w.decide,exact:true}).click();await rowButton().waitFor({state:'detached'});await idle();
  assert.deepEqual(Object.keys(evidence.writes[0].body).sort(),['change','change_digest','configuration','digest','verdict']);assert.equal(evidence.writes[0].body.verdict,'approve');
  if(lang==='en')assert.ok((await page.locator('.o03-changes-table').innerText()).includes('Scheduled'));
  await shot('decided');evidence.cases.push({check:'decide',lang,width});
 }
 assert.deepEqual(evidence.errors,[]);
}finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
console.log(`cases ${evidence.cases.length} unexpected ${JSON.stringify([...new Set(evidence.unexpected)])}`);
