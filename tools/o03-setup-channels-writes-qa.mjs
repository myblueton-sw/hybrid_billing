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
const channel=(id,registrar,extra={})=>({channel:id,digest:hash,subject_kind:'entity',kind:'portal',state:'registered',observation:'registered',registrar,admitter:'',pending_admitter:'',registered_at:'2026-11-21T01:00:00Z',admitted_at:'',retired_by:'',retired_at:'',rules:'current',request_not_before:'',...extra});
const claim=(id,actor,purpose,target,state='unconsumed')=>({attestation:id,digest:hash,purpose,target,actor,recorded_at:'2026-11-21T01:00:00Z',statement:'Synthetic statement kept on record.',locator:'',state,assurance:'human_attestation'});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Built console with an intercepted synthetic service that keeps state in the check itself; no server, no real mutation, no owner acceptance',cases:[],unexpected:[],errors:[],writes:[]};
let world,fault={};
const reset=(channels=[],claims=[])=>{world={channels,claims,revision:1};fault={};evidence.writes.length=0;};
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
 if(command==='issuance_setup.protected_channels')return done({schema:'hb_issuance_protected_channel_page_v1',...scoped,snapshot:String(world.revision).padStart(64,'c'),count:String(world.channels.length),next:'',other_provenance:'0',unowned_module:'0',...rows(world.channels)});
 if(command==='issuance_setup.attestations')return done({schema:'hb_issuance_setup_attestation_page_v1',...scoped,snapshot:String(world.revision).padStart(64,'d'),count:String(world.claims.length),next:'',...rows(world.claims)});
 if(command==='issuance_setup.references')return done({schema:'hb_issuance_setup_references_v1',...scoped,...Object.fromEntries(catalog.map(([code,use],i)=>[`reference_${String(i).padStart(2,'0')}`,JSON.stringify({code,use,applicability:'synthetic_only'})]))});
 const operation=command.replace('issuance_setup.',''),a=body.body,isClaim=/^(attest|withdraw)_/.test(operation),isChannel=['register_channel','admit_channel'].includes(operation);
 if(isClaim||isChannel){
  evidence.writes.push({operation,body:a,rationale:body.rationale,key:body.idempotency_key,request:body.request_id,expected:body.expected_revision,target:body.target_id});
  const mode=fault[isClaim?'claim':'action'];
  if(mode==='prerequisite')return route.fulfill({status:422,json:{code:'prerequisite_failed',retry:false,next_action:'correct_request',request_id:body.request_id}});
  if(mode==='unknown')return route.fulfill({status:503,json:{code:'dependency_unavailable',retry:true,next_action:'retry_later',request_id:body.request_id,operation_id:'operation-unknown'}});
  world.revision++;const result=`r${world.revision}`,reply=(business,fields)=>route.fulfill({json:{request_id:body.request_id,status:'completed',operation_id:`operation-${world.revision}`,target_id:body.target_id,result_revision:result,business_state:business,replay:false,as_of:now,fields}});
  if(isClaim){
   const purpose=operation.replace(/^(attest|withdraw)_/,''),withdraw=operation.startsWith('withdraw_');
   if(withdraw)world.claims=world.claims.map(c=>c.attestation===a.attestation?{...c,state:'withdrawn'}:c);
   else world.claims=[...world.claims,{...claim(a.attestation,'governor',purpose,a.target),statement:a.statement,locator:a.locator??''}].sort((x,y)=>x.attestation<y.attestation?-1:1);
   const row=world.claims.find(c=>c.attestation===a.attestation);
   return reply(withdraw?'withdrawn':'recorded',{schema:'hb_issuance_setup_attestation_v1',contract:contract.id,configuration:'configuration-a',attestation:a.attestation,purpose,target:row.target,actor:'governor',recorded_at:now,digest:hash,state:row.state,assurance:'human_attestation'});
  }
  world.claims=world.claims.map(c=>c.attestation===a.attestation?{...c,state:'consumed'}:c);
  if(operation==='register_channel'){world.channels=[...world.channels,channel(a.channel,'governor',{subject_kind:a.subject_kind,kind:a.kind})].sort((x,y)=>x.channel<y.channel?-1:1);return reply('registered',{schema:'hb_issuance_protected_channel_receipt_v1',contract:contract.id,configuration:'configuration-a',current_eligibility:'unchecked',channel:a.channel,channel_digest:hash,state:'registered'});}
  world.channels=world.channels.map(c=>c.channel===a.channel?{...c,state:'admitted',observation:'admitted',admitter:'governor',admitted_at:now,request_not_before:'2026-11-25T09:00:00Z'}:c);
  return reply('admitted',{schema:'hb_issuance_protected_channel_receipt_v1',contract:contract.id,configuration:'configuration-a',current_eligibility:'unchecked',channel:a.channel,channel_digest:hash,state:'admitted'});
 }
 if(['issuance_setup.context','issuance_setup.options','issuance_setup.protected'].includes(command))return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
const open=async(lang,width,part='channels')=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/contracts?view=documents&contract=${contract.id}&on=${on}&part=${part}&lang=${lang}`);await page.locator(part==='channels'?'#o03-channel-steps':'#o03-statement-steps').waitFor();};
const steps=page.locator('.o03-part-sub.o03-steps'),actions=page.locator('#o03-protected-results').locator('..'),claims=page.locator('#o03-statement-results').locator('..'),button=name=>steps.getByRole('button',{name,exact:true});
const active=()=>page.evaluate(()=>document.activeElement?.id||document.activeElement?.tagName);
const noOverflow=async where=>assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`overflow ${where}`);
const controls=async where=>{const sizes=await steps.evaluate(el=>[...el.querySelectorAll('button, input:not([type=radio]), textarea, .live-check')].map(e=>Math.round(e.getBoundingClientRect().height)));assert.ok(sizes.every(h=>h>=44),`targets ${where} ${sizes}`);};
const guarded=()=>page.evaluate(()=>{const e=new Event('beforeunload',{cancelable:true});window.dispatchEvent(e);return e.defaultPrevented;});
const idle=()=>page.waitForFunction(()=>!document.querySelector('[aria-busy=true]'));
const rowButton=()=>page.locator('.o03-part-table tbody button');
const words={
 en:{register:'Register verification channel',accept:'Accept verification channel',review:'Review before sending',back:'Back to inputs',cancel:'Cancel',retry:'Retry exact request',withdraw:'Withdraw statement'},
 ko:{register:'검증 채널 등록',accept:'검증 채널 수락',review:'전송 전 검토',back:'입력으로 돌아가기',cancel:'취소',retry:'동일 요청 재시도',withdraw:'진술 철회'},
 ar:{register:'تسجيل قناة التحقق',accept:'قبول قناة التحقق',review:'المراجعة قبل الإرسال',back:'العودة إلى المدخلات',cancel:'إلغاء',retry:'إعادة محاولة الطلب نفسه',withdraw:'سحب الإفادة'},
};
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  const where=`${lang} ${width}`,w=words[lang],shot=name=>page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});
  reset();await open(lang,width);
  await button(w.register).click();await page.locator('#o03-register-reason').waitFor();assert.equal(await steps.locator('input[type=radio]:checked').count(),0,'a choice is preselected');assert.equal(await steps.locator('input[name=o03-channel-reference]').count(),0,'samples are offered before a subject is chosen');
  await button(w.review).click();await steps.locator('.live-error').waitFor();assert.equal(evidence.writes.length,0);await shot('register-invalid');
  await steps.locator('input[name=o03-channel-subject]').nth(1).check();await steps.locator('input[name=o03-channel-kind]').nth(2).check();
  assert.equal(await steps.locator('input[name=o03-channel-reference]').count(),2);await steps.locator('input[name=o03-channel-reference]').nth(0).check();
  await page.locator('#o03-register-statement').fill('Synthetic statement: I registered the portal channel.\nReceived the signed form in person.');await page.locator('#o03-register-locator').fill('Synthetic folder 12');await page.locator('#o03-register-reason').fill('Synthetic reason for the portal channel');
  await controls(where);await noOverflow(where);await shot('register-form');
  await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();assert.equal(await active(),'H5');
  const reviewText=await steps.locator('.o03-setup-review').innerText();assert.ok(reviewText.includes('Received the signed form in person.'));assert.ok(reviewText.includes('Synthetic folder 12'));assert.ok(!reviewText.includes('synthetic_channel'),'a sample code is shown');
  if(lang==='en'){for(const expected of ['Statement you will record','Step it is written for','Two requests are sent','A statement used by a step stays used','Sample issuing entity channel A','Issuing entity','Portal'])assert.ok(reviewText.includes(expected),expected);}
  await noOverflow(where+' review');await shot('register-review');assert.equal(evidence.writes.length,0);
  await steps.locator('.o03-setup-review').getByRole('button',{name:w.register,exact:true}).click();await rowButton().waitFor({state:'detached'});await page.locator('.o03-part-table tbody tr').waitFor();await idle();
  assert.deepEqual(evidence.writes.map(v=>v.operation),['attest_channel_registration','register_channel']);const [said,did]=evidence.writes;
  assert.equal(said.body.statement,'Synthetic statement: I registered the portal channel.\nReceived the signed form in person.');assert.equal(said.body.locator,'Synthetic folder 12');assert.equal(said.body.target_digest,undefined);assert.equal(said.rationale,undefined);assert.equal(said.expected,'r1');
  assert.equal(did.body.channel,said.body.target);assert.equal(did.body.attestation,said.body.attestation);assert.equal(did.expected,'r2','the action did not use the revision of the statement receipt');assert.equal(did.rationale,'Synthetic reason for the portal channel');
  assert.deepEqual({subject:did.body.subject_kind,kind:did.body.kind,reference:did.body.reference},{subject:'entity',kind:'portal',reference:'synthetic_channel_entity_a'});
  if(lang==='en')assert.ok((await page.locator('.o03-part-table').innerText()).includes('You registered this channel'));
  assert.equal(await actions.locator('.o03-setup-outcome').count(),1);assert.equal(await claims.locator('.o03-setup-outcome').count(),1);assert.equal(await guarded(),false);
  if(lang==='en'){const told=await claims.innerText();assert.ok(told.includes('State: Used by a step recorded in this session'));assert.ok(!told.includes('supports no record'),'a used statement is still worded as unused');assert.ok(!told.includes('can be retried or recovered only'),'the session sentence is repeated in the second block');}
  await noOverflow(where+' registered');await shot('registered');evidence.cases.push({check:'register',lang,width});

  reset([channel('channel-seeded','synthetic-registrar')]);await open(lang,width);
  await rowButton().click();await page.locator('#o03-accept-statement').waitFor();assert.equal(await active(),'o03-channel-steps');
  await button(w.review).click();await steps.locator('.live-error').waitFor();assert.ok((await steps.locator('.live-error').innerText()).includes('invalid_accept'));
  await page.locator('#o03-accept-statement').fill('Synthetic statement: I checked the channel independently.');await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();await shot('accept-review');
  fault={action:'prerequisite'};await steps.locator('.o03-setup-review').getByRole('button',{name:w.accept,exact:true}).click();await actions.locator('.o03-setup-outcome').filter({hasText:'prerequisite_failed'}).waitFor();await idle();await rowButton().waitFor();
  assert.deepEqual(evidence.writes.map(v=>v.operation),['attest_channel_admission','admit_channel']);assert.equal(evidence.writes[0].body.target,'channel-seeded');assert.equal(evidence.writes[0].body.target_digest,hash);
  if(lang==='en'){const refusedText=await actions.innerText();assert.ok(refusedText.includes('Refused. This request recorded nothing.'));assert.ok(refusedText.includes('The statement recorded for this step was not used by this request.'));assert.ok((await claims.innerText()).includes('Not used — supports no record'));}
  await shot('refused');
  fault={};await rowButton().click();await steps.locator('.o03-kept').waitFor();assert.equal(await page.locator('#o03-accept-statement').count(),0,'a second statement is asked for although one is unused');
  await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();if(lang==='en'){const again=await steps.locator('.o03-setup-review').innerText();assert.ok(again.includes('One request is sent'));assert.ok(again.includes('Statement you recorded in this session'));}
  await shot('reuse-review');
  await steps.locator('.o03-setup-review').getByRole('button',{name:w.accept,exact:true}).click();await rowButton().waitFor({state:'detached'});await idle();
  assert.deepEqual(evidence.writes.map(v=>v.operation),['attest_channel_admission','admit_channel','admit_channel']);assert.equal(evidence.writes[2].body.attestation,evidence.writes[0].body.attestation);assert.notEqual(evidence.writes[2].key,evidence.writes[1].key);
  if(lang==='en')assert.ok((await page.locator('.o03-part-table').innerText()).includes('Accepted'));
  await shot('accepted');evidence.cases.push({check:'accept-refused-reuse',lang,width});

  reset([channel('channel-seeded','synthetic-registrar')]);await open(lang,width);
  fault={claim:'unknown'};await rowButton().click();await page.locator('#o03-accept-statement').fill('Synthetic statement written before an outage.');await button(w.review).click();await steps.locator('.o03-setup-review').getByRole('button',{name:w.accept,exact:true}).click();
  await claims.getByRole('button',{name:w.retry,exact:true}).waitFor();await idle();await page.locator('#o03-channel-blocked').waitFor();
  assert.deepEqual(evidence.writes.map(v=>v.operation),['attest_channel_admission'],'the action was sent although the statement did not settle');
  assert.equal(await rowButton().isDisabled(),true);assert.equal(await guarded(),true);
  if(lang==='en'){assert.ok((await steps.innerText()).includes('Steps are unavailable until the request with an unknown outcome'));assert.ok((await claims.innerText()).includes('Outcome unknown.'));assert.ok((await claims.innerText()).includes('The step this statement was written for has not been sent.'));}
  await shot('statement-unknown');
  fault={};await claims.getByRole('button',{name:w.retry,exact:true}).click();await page.waitForFunction(()=>!document.querySelector('#o03-channel-blocked'));await idle();
  assert.equal(evidence.writes.length,2);assert.equal(evidence.writes[1].key,evidence.writes[0].key);assert.equal(await rowButton().isDisabled(),false);
  await rowButton().click();await steps.locator('.o03-kept').waitFor();await button(w.review).click();await steps.locator('.o03-setup-review').getByRole('button',{name:w.accept,exact:true}).click();await rowButton().waitFor({state:'detached'});await idle();
  assert.deepEqual(evidence.writes.map(v=>v.operation),['attest_channel_admission','attest_channel_admission','admit_channel']);assert.equal(evidence.writes[2].body.attestation,evidence.writes[0].body.attestation);assert.equal(await guarded(),false);
  evidence.cases.push({check:'statement-unknown',lang,width});

  reset([],[claim('statement-mine','governor','channel_admission','channel-9'),claim('statement-other','synthetic-other','channel_admission','channel-9'),claim('statement-used','governor','channel_registration','channel-9','consumed')]);await open(lang,width,'statements');
  await page.locator('.o03-statements-table tbody tr').nth(2).waitFor();assert.equal(await page.locator('.o03-statements-table tbody button').count(),1,'withdraw is offered on a statement of another person or on a used one');
  await page.locator('.o03-statements-table tbody button').click();await steps.locator('.o03-setup-review').waitFor();assert.equal(await active(),'o03-statement-steps');
  if(lang==='en'){const text=await steps.innerText();assert.ok(text.includes('A withdrawn statement stays on record as withdrawn'));assert.ok(text.includes('Only its author can withdraw a statement'));}
  await noOverflow(where+' withdraw');await shot('withdraw-review');
  await steps.getByRole('button',{name:w.withdraw,exact:true}).click();await page.locator('.o03-statements-table tbody button').waitFor({state:'detached'});await idle();
  assert.deepEqual(evidence.writes.map(v=>[v.operation,Object.keys(v.body).sort().join()]),[['withdraw_channel_admission','attestation,configuration,digest']]);assert.equal(evidence.writes[0].body.attestation,'statement-mine');
  if(lang==='en')assert.ok((await page.locator('.o03-statements-table').innerText()).includes('Withdrawn — supports no record'));
  await shot('withdrawn');evidence.cases.push({check:'withdraw',lang,width});
 }
 assert.deepEqual(evidence.errors,[]);
}finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
console.log(`cases ${evidence.cases.length} unexpected ${JSON.stringify([...new Set(evidence.unexpected)])}`);
