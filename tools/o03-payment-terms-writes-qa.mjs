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
const term=(author,state='pending')=>({term:'term-seeded',term_digest:hash,term_revision:'1',net_days:'30',mode:'calendar',term_author:author,term_approver:state==='approved'?'synthetic-approver':'',term_pending_approver:'',term_written_at:'2026-11-21T01:00:00Z',term_approved_at:state==='approved'?'2026-11-21T02:00:00Z':'',term_state:state,term_observation:state});
const noTerm={term_state:'none',term_observation:'absent',term:'',term_digest:'',term_revision:'',net_days:'',mode:'',term_author:'',term_approver:'',term_pending_approver:'',term_written_at:'',term_approved_at:''};
const binding=(id,author,state='pending')=>({binding:id,digest:hash,origin:'configuration',configuration:'configuration-a',term:'term-seeded',term_revision:'1',net_days:'30',mode:'calendar',zone:'UTC',from:'2026-11-01',to:'2027-10-31',reason:'initial',state,observation:state,author,approver:state==='approved'?'governor':'',pending_approver:'',written_at:'2026-11-21T03:00:00Z',approved_at:state==='approved'?'2026-11-21T04:00:00Z':''});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Built console with an intercepted synthetic service that keeps state in the check itself; no server, no real mutation, no owner acceptance',cases:[],unexpected:[],errors:[],writes:[]};
let world,fault='';
const reset=(termBlock,rows=[])=>{world={termBlock,rows,revision:1};fault='';failRead=false;evidence.writes.length=0;};
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
 const command=path.split('/').at(-1),body=request.postDataJSON(),revision=`r${world.revision}`;
 const done=fields=>route.fulfill({json:{request_id:body.request_id,status:'completed',target_id:body.target_id,result_revision:revision,replay:false,as_of:now,fields}});
 if(command==='issuance_setup.configurations')return done({schema:'hb_issuance_configuration_page_v1',contract:contract.id,current_eligibility:'unchecked',snapshot:hash,count:'1',next:'',row_00:JSON.stringify(configuration)});
 if(command==='issuance_setup.terms'&&failRead)return route.fulfill({status:503,json:{code:'dependency_unavailable',request_id:body.request_id}});
 if(command==='issuance_setup.terms')return done({schema:'hb_issuance_setup_terms_v1',contract:contract.id,configuration:'configuration-a',digest:hash,zone:'UTC',current_eligibility:'unchecked',...world.termBlock,snapshot:String(world.revision).padStart(64,'b'),count:String(world.rows.length),next:'',...Object.fromEntries(world.rows.map((r,i)=>[`row_${String(i).padStart(2,'0')}`,JSON.stringify(r)]))});
 const operation=command.replace('issuance_setup.','');
 if(['record_term','approve_term','bind_term','approve_term_binding'].includes(operation)){
  evidence.writes.push({operation,body:body.body,rationale:body.rationale,key:body.idempotency_key,request:body.request_id,expected:body.expected_revision,target:body.target_id,csrf:request.headers()['x-csrf-token']});
  if(fault==='refuse')return route.fulfill({status:403,json:{code:'self_approval_denied',retry:false,next_action:'none',request_id:body.request_id}});
  if(fault==='prerequisite')return route.fulfill({status:422,json:{code:'prerequisite_failed',retry:false,next_action:'correct_request',request_id:body.request_id}});
  if(fault==='unknown')return route.fulfill({status:503,json:{code:'dependency_unavailable',retry:true,next_action:'retry_later',request_id:body.request_id,operation_id:'operation-unknown'}});
  world.revision++;let fields;const a=body.body;
  if(operation==='record_term'){world.termBlock={...term('governor'),term:a.term,net_days:a.net_days};fields={term:a.term,term_digest:hash,state:'pending'};}
  if(operation==='approve_term'){world.termBlock={...world.termBlock,term_state:'approved',term_observation:'approved',term_approver:'governor',term_approved_at:now};fields={term:a.term,term_digest:hash,state:'approved'};}
  if(operation==='bind_term'){world.rows=[...world.rows,{...binding(a.binding,'governor'),term:a.term,from:a.from,to:a.to??'',reason:a.reason}].sort((x,y)=>x.binding<y.binding?-1:1);fields={binding:a.binding,binding_digest:hash,state:'pending'};}
  if(operation==='approve_term_binding'){world.rows=world.rows.map(r=>r.binding===a.binding?{...r,state:'approved',observation:'approved',approver:'governor',approved_at:now}:r);fields={binding:a.binding,binding_digest:hash,state:'approved'};}
  return route.fulfill({json:{request_id:body.request_id,status:'completed',operation_id:`operation-${world.revision}`,target_id:body.target_id,result_revision:`r${world.revision}`,business_state:fields.state,replay:false,as_of:now,fields:{schema:operation.includes('binding')||operation==='bind_term'?'hb_issuance_term_binding_receipt_v1':'hb_issuance_term_receipt_v1',contract:contract.id,configuration:'configuration-a',current_eligibility:'unchecked',...fields}}});
 }
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
let session=0;
const open=async(lang,width)=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}&run=${++session}#/contracts?view=payment-terms&contract=${contract.id}&on=${on}&lang=${lang}`);await page.locator('#o03-term-steps').waitFor();};
const steps=page.locator('.o03-part-sub.o03-steps'),results=page.locator('section.o03-steps'),button=name=>steps.getByRole('button',{name,exact:true}),result=name=>results.getByRole('button',{name,exact:true});
let failRead=false;
const active=()=>page.evaluate(()=>document.activeElement?.id||document.activeElement?.tagName);
const noOverflow=async where=>assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`overflow ${where}`);
const choices=async where=>{const rows=await steps.evaluate(el=>[...el.querySelectorAll('.live-check')].map(label=>{const i=label.querySelector('input').getBoundingClientRect(),t=label.getBoundingClientRect();return {same:Math.abs((i.top+i.bottom)/2-(t.top+t.bottom)/2)<=6,size:Math.round(i.width),wide:Math.round(t.width)};}));assert.ok(rows.length>=5&&rows.every(r=>r.same&&r.size<=24),`choices ${where} ${JSON.stringify(rows)}`);};
const controls=async where=>{const sizes=await steps.evaluate(el=>[...el.querySelectorAll('button, input:not([type=radio]), textarea, .live-check')].map(e=>Math.round(e.getBoundingClientRect().height)));assert.ok(sizes.every(h=>h>=44),`targets ${where} ${sizes}`);};
const words={
 en:{record:'Record payment terms',approve:'Approve payment terms',apply:'Apply payment terms to this contract',approveApply:'Approve applying payment terms to this contract',review:'Review before sending',back:'Back to inputs',cancel:'Cancel',retry:'Retry exact request',recover:'Recover recorded attempt'},
 ko:{record:'납부 조건 기록',approve:'납부 조건 승인',apply:'이 계약에 납부 조건 적용',approveApply:'이 계약에 납부 조건 적용 승인',review:'전송 전 검토',back:'입력으로 돌아가기',cancel:'취소',retry:'동일 요청 재시도',recover:'기록된 시도 복구'},
 ar:{record:'تسجيل شروط الدفع',approve:'اعتماد شروط الدفع',apply:'تطبيق شروط الدفع على هذا العقد',approveApply:'اعتماد تطبيق شروط الدفع على هذا العقد',review:'المراجعة قبل الإرسال',back:'العودة إلى المدخلات',cancel:'إلغاء',retry:'إعادة محاولة الطلب نفسه',recover:'استعادة المحاولة المسجلة'},
};
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  const where=`${lang} ${width}`,w=words[lang],shot=name=>page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});
  reset(noTerm);await open(lang,width);
  assert.equal(await steps.getByRole('button').count(),1);await button(w.record).click();await page.locator('#o03-term-days').waitFor();await controls(where);await noOverflow(where);
  assert.equal(await page.locator('#o03-term-days').inputValue(),'');assert.equal(await page.locator('#o03-term-days').getAttribute('placeholder'),null);
  await button(w.review).click();await steps.locator('.live-error').waitFor();assert.ok((await steps.locator('.live-error').innerText()).includes('invalid_setup_request'));assert.equal(await active(),'o03-term-steps');assert.equal(evidence.writes.length,0);
  await shot('record-invalid');
  await page.locator('#o03-term-days').fill('45');await page.locator('#o03-term-reason').fill('Synthetic reason agreed with the customer');
  await page.locator('.planning-tabs a, .planning-tabs button').first().waitFor();
  await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();assert.equal(await steps.locator('.live-error').count(),0);
  const reviewText=await steps.locator('.o03-setup-review').innerText();assert.ok(reviewText.includes('Synthetic reason agreed with the customer'));assert.ok(reviewText.includes('45'));if(lang==='en'){assert.ok(reviewText.includes('45 days'));assert.ok(reviewText.includes('Net days cannot be corrected after recording'));assert.ok(reviewText.includes('Permission is checked when it is sent'));}
  await noOverflow(where+' review');await shot('record-review');assert.equal(evidence.writes.length,0);
  await button(w.back).click();assert.equal(await page.locator('#o03-term-days').inputValue(),'45');await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();
  await steps.locator('.o03-setup-review').getByRole('button',{name:w.record,exact:true}).click();await results.locator('.o03-setup-outcome .o03-tag').waitFor();await page.locator('#o03-term-steps').waitFor();
  assert.equal(evidence.writes.length,1);const sent=evidence.writes[0];
  assert.deepEqual({operation:sent.operation,rationale:sent.rationale,expected:sent.expected,target:sent.target,csrf:sent.csrf,days:sent.body.net_days,configuration:sent.body.configuration,digest:sent.body.digest,keys:Object.keys(sent.body).sort()},{operation:'record_term',rationale:'Synthetic reason agreed with the customer',expected:'r1',target:contract.id,csrf:'synthetic',days:'45',configuration:'configuration-a',digest:hash,keys:['configuration','digest','net_days','term']});
  assert.equal(await steps.getByRole('button').count(),0,'the recorder is offered no approval');assert.equal(await page.locator('.o03-docs-scope .o03-tag').count(),1);
  if(lang==='en'){assert.ok((await steps.innerText()).includes('You recorded these terms'));assert.ok((await results.innerText()).includes('Recorded. Current issuance eligibility was not checked.'));assert.ok((await results.innerText()).includes('can be retried or recovered only from the session that sent it'));}
  await button(w.record).waitFor({state:'detached'});
  await noOverflow(where+' recorded');await shot('recorded');evidence.cases.push({check:'record',lang,width});

  reset(term('synthetic-author'));await open(lang,width);
  await button(w.approve).click();await steps.locator('.o03-setup-review').waitFor();assert.equal(await active(),'H5');await shot('approve-review');
  await button(w.cancel).click();assert.equal(await steps.locator('.o03-setup-review').count(),0);assert.equal(evidence.writes.length,0);assert.equal(await active(),'o03-term-steps');
  await button(w.approve).click();await steps.locator('.o03-setup-review').getByRole('button',{name:w.approve,exact:true}).click();await button(w.apply).waitFor();
  assert.deepEqual(evidence.writes.map(v=>[v.operation,v.body.term,v.body.term_digest,v.rationale]),[['approve_term','term-seeded',hash,undefined]]);
  await button(w.apply).click();await page.locator('#o03-bind-from').waitFor();assert.equal(await steps.locator('input[type=radio]:checked').count(),0,'a choice is preselected');
  await button(w.review).click();await steps.locator('.live-error').waitFor();assert.equal(evidence.writes.length,1);assert.ok((await steps.locator('.live-error').innerText()).includes('invalid_apply'));
  await page.locator('#o03-bind-from').fill('2026-11-01');await steps.locator('input[name=o03-bind-end]').nth(1).check();await steps.locator('input[name=o03-bind-reason]').nth(0).check();
  if(lang==='en')assert.ok((await steps.innerText()).includes('Without an end date the terms occupy every later date'));
  await controls(where+' apply');await choices(where);await noOverflow(where+' apply');await shot('apply-form');
  await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();await shot('apply-review');
  await steps.locator('.o03-setup-review').getByRole('button',{name:w.apply,exact:true}).click();await page.locator('.o03-bindings-table tbody tr').waitFor();
  const bound=evidence.writes[1];assert.equal(bound.operation,'bind_term');assert.deepEqual(Object.keys(bound.body).sort(),['binding','configuration','digest','from','reason','term','term_digest']);assert.equal(bound.body.from,'2026-11-01');assert.equal(bound.body.reason,'initial');assert.equal(bound.expected,'r2');assert.equal(bound.rationale,undefined);
  assert.equal(await page.locator('.o03-bindings-table tbody button').count(),0,'the person who applied is offered no approval');
  if(lang==='en')assert.ok((await page.locator('.o03-bindings-table').innerText()).includes('You applied these terms'));
  await noOverflow(where+' applied');await shot('applied');evidence.cases.push({check:'approve-apply',lang,width});

  reset(term('synthetic-author','approved'),[binding('binding-seeded','synthetic-author')]);await open(lang,width);
  fault='refuse';await page.locator('.o03-bindings-table tbody button').click();await steps.locator('.o03-setup-review').waitFor();assert.equal(await active(),'H5');
  await steps.locator('.o03-setup-review').getByRole('button',{name:w.approveApply,exact:true}).click();await results.locator('.o03-setup-outcome').filter({hasText:'self_approval_denied'}).waitFor();await page.locator('#o03-term-steps').waitFor();
  let outcome=await results.locator('.o03-setup-outcome').innerText();assert.ok(outcome.includes('self_approval_denied'));if(lang==='en'){assert.ok(outcome.includes('Refused. This request recorded nothing.'));assert.ok(outcome.includes('cannot approve them'));}
  assert.equal(await results.locator('.o03-setup-outcome button').count(),0);await shot('refused');
  fault='unknown';failRead=true;await page.locator('.o03-bindings-table tbody button').click();await steps.locator('.o03-setup-review').getByRole('button',{name:w.approveApply,exact:true}).click();await result(w.retry).waitFor();
  await page.locator('.o03-part > .live-error').waitFor();assert.equal(await page.locator('.o03-bindings-table').count(),0);assert.equal(await page.locator('#o03-term-steps').count(),0);
  const unknown=await results.locator('.o03-setup-outcome').first().innerText();assert.ok(unknown.includes('dependency_unavailable'));assert.ok(unknown.includes('binding-seeded'));
  if(lang==='en'){assert.ok(unknown.includes('Outcome unknown.'));assert.ok(!unknown.includes('could not be completed'),'an unknown outcome is worded as a failure');}
  assert.equal(await page.evaluate(()=>{const e=new Event('beforeunload',{cancelable:true});window.dispatchEvent(e);return e.defaultPrevented;}),true,'no unload guard while an outcome is unknown and the read failed');
  await shot('unknown-read-failed');
  failRead=false;await page.locator('.o03-docs-heading button').click();await page.locator('.o03-bindings-table tbody button').waitFor();
  assert.equal(await page.locator('.o03-bindings-table tbody button').isDisabled(),true,'a step is offered while an outcome is unknown');assert.equal(await page.locator('.o03-bindings-table tbody button').getAttribute('aria-describedby'),'o03-term-blocked');assert.equal(await page.locator('#o03-term-blocked').count(),1);if(lang==='en')assert.ok((await steps.innerText()).includes('Steps are unavailable until the request with an unknown outcome'));await shot('unknown');
  assert.equal(evidence.writes.length,2);
  await result(w.retry).click();await page.waitForFunction(()=>!document.querySelector('section.o03-steps[aria-busy]'));await result(w.retry).waitFor();
  assert.equal(evidence.writes.length,3);assert.equal(evidence.writes[2].key,evidence.writes[1].key);assert.equal(evidence.writes[2].request,evidence.writes[1].request);
  fault='';await result(w.recover).click();
  await page.waitForFunction(()=>document.querySelectorAll('section.o03-steps .o03-setup-outcome').length===3&&!document.querySelector('section.o03-steps .o03-setup-outcome button')&&!document.querySelector('section.o03-steps[aria-busy]')&&!document.querySelector('.o03-bindings-table tbody button'));
  assert.equal(evidence.writes.length,4);assert.notEqual(evidence.writes[3].key,evidence.writes[1].key);assert.deepEqual(evidence.writes[3].body,evidence.writes[1].body);assert.equal(evidence.writes[3].expected,'r1');
  assert.equal(await page.locator('#o03-term-blocked').count(),0);assert.equal(await results.locator('.o03-setup-outcome').evaluateAll(list=>list.every(e=>/UTC/.test(e.innerText))),true,'an entry has no sent time');
  if(lang==='en'){outcome=await results.innerText();assert.ok(outcome.includes('Settled by a matching request of this session'));assert.ok(outcome.indexOf('Recorded. Current')<outcome.indexOf('Settled by'),'newest result is not first');}
  assert.equal(await page.evaluate(()=>{const e=new Event('beforeunload',{cancelable:true});window.dispatchEvent(e);return e.defaultPrevented;}),false,'unload guard left on after everything settled');
  await noOverflow(where+' settled');await shot('settled');evidence.cases.push({check:'refusal-unknown-recovery',lang,width});
 }
 assert.deepEqual(evidence.errors,[]);
}finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
console.log(`cases ${evidence.cases.length} unexpected ${JSON.stringify([...new Set(evidence.unexpected)])}`);
