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
const group=(prefix,kind)=>({[`${prefix}_subject_kind`]:kind,[`${prefix}_changes`]:'0',[`${prefix}_other_provenance`]:'0',[`${prefix}_unowned_module`]:'0',[`${prefix}_latest_scheduled`]:'',[`${prefix}_latest_scheduled_effective`]:''});
const none={rules_state:'none',rules_observation:'absent',rules_id:'',rules_digest:'',rules_approvals:'',rules_validity_seconds:'',rules_max_lead_seconds:'',rules_channel_age_seconds:'',rules_proposer:'',rules_confirmer:'',rules_pending_confirmer:'',rules_proposed_at:'',rules_confirmed_at:''};
const proposed=proposer=>({rules_state:'proposed',rules_observation:'pending',rules_id:'rules-seeded',rules_digest:hash,rules_approvals:'2',rules_validity_seconds:'172800',rules_max_lead_seconds:'604800',rules_channel_age_seconds:'86400',rules_proposer:proposer,rules_confirmer:'',rules_pending_confirmer:'',rules_proposed_at:'2026-11-21T01:00:00Z',rules_confirmed_at:''});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Built console with an intercepted synthetic service that keeps state in the check itself; no server, no real mutation, no owner acceptance',cases:[],unexpected:[],errors:[],writes:[]};
let world,fault='',failRead=false;
const reset=rules=>{world={rules,revision:1};fault='';failRead=false;evidence.writes.length=0;};
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
 if(command==='issuance_setup.protected'&&failRead)return route.fulfill({status:503,json:{code:'dependency_unavailable',request_id:body.request_id}});
 if(command==='issuance_setup.protected')return done({schema:'hb_issuance_protected_setup_v1',contract:contract.id,configuration:'configuration-a',digest:hash,current_eligibility:'unchecked',channels:'0',channels_other_provenance:'0',channels_unowned_module:'0',...world.rules,...group('recipient','customer'),...group('template','document_kind'),...group('account','entity')});
 const operation=command.replace('issuance_setup.','');
 if(['propose_protected_rules','confirm_protected_rules'].includes(operation)){
  evidence.writes.push({operation,body:body.body,rationale:body.rationale,key:body.idempotency_key,request:body.request_id,expected:body.expected_revision,target:body.target_id});
  if(fault==='refuse')return route.fulfill({status:403,json:{code:'self_approval_denied',retry:false,next_action:'none',request_id:body.request_id}});
  if(fault==='denied')return route.fulfill({status:403,json:{code:'permission_denied',retry:false,next_action:'none',request_id:body.request_id}});
  if(fault==='unknown')return route.fulfill({status:503,json:{code:'dependency_unavailable',retry:true,next_action:'retry_later',request_id:body.request_id,operation_id:'operation-unknown'}});
  world.revision++;const a=body.body;let state;
  if(operation==='propose_protected_rules'){world.rules={...proposed('governor'),rules_id:a.rules,rules_approvals:a.approvals,rules_validity_seconds:a.validity_seconds,rules_max_lead_seconds:a.max_lead_seconds,rules_channel_age_seconds:a.channel_age_seconds};state='proposed';}
  else{world.rules={...world.rules,rules_state:'confirmed',rules_observation:'active',rules_confirmer:'governor',rules_confirmed_at:now};state='confirmed';}
  return route.fulfill({json:{request_id:body.request_id,status:'completed',operation_id:`operation-${world.revision}`,target_id:body.target_id,result_revision:`r${world.revision}`,business_state:state,replay:false,as_of:now,fields:{schema:'hb_issuance_protected_rules_receipt_v1',contract:contract.id,configuration:'configuration-a',current_eligibility:'unchecked',rules:a.rules,rules_digest:hash,state}}});
 }
 if(['issuance_setup.context','issuance_setup.options'].includes(command))return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
const open=async(lang,width,part='rules')=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/contracts?view=documents&contract=${contract.id}&on=${on}&part=${part}&lang=${lang}`);await page.locator(part==='rules'?'#o03-rule-steps':'.o03-parts').waitFor();};
const steps=page.locator('.o03-part-sub.o03-steps'),results=page.locator('section.o03-results'),button=name=>steps.getByRole('button',{name,exact:true}),result=name=>results.getByRole('button',{name,exact:true});
const active=()=>page.evaluate(()=>document.activeElement?.id||document.activeElement?.tagName);
const noOverflow=async where=>assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,`overflow ${where}`);
const controls=async where=>{const sizes=await steps.evaluate(el=>[...el.querySelectorAll('button, input:not([type=radio]), textarea, .live-check')].map(e=>Math.round(e.getBoundingClientRect().height)));assert.ok(sizes.every(h=>h>=44),`targets ${where} ${sizes}`);};
const guarded=()=>page.evaluate(()=>{const e=new Event('beforeunload',{cancelable:true});window.dispatchEvent(e);return e.defaultPrevented;});
const words={
 en:{propose:'Propose protected settings rules',confirm:'Confirm protected settings rules',review:'Review before sending',back:'Back to inputs',cancel:'Cancel',retry:'Retry exact request',recover:'Recover recorded attempt'},
 ko:{propose:'보호된 설정 규칙 제안',confirm:'보호된 설정 규칙 확인',review:'전송 전 검토',back:'입력으로 돌아가기',cancel:'취소',retry:'동일 요청 재시도',recover:'기록된 시도 복구'},
 ar:{propose:'اقتراح قواعد الإعدادات المحمية',confirm:'تأكيد قواعد الإعدادات المحمية',review:'المراجعة قبل الإرسال',back:'العودة إلى المدخلات',cancel:'إلغاء',retry:'إعادة محاولة الطلب نفسه',recover:'استعادة المحاولة المسجلة'},
};
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  const where=`${lang} ${width}`,w=words[lang],shot=name=>page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});
  reset(none);await open(lang,width);
  assert.equal(await steps.getByRole('button').count(),1);await button(w.propose).click();await page.locator('#o03-rule-reason').waitFor();await controls(where);await noOverflow(where);
  assert.equal(await steps.locator('input[type=radio]:checked').count(),0,'a choice is preselected');assert.deepEqual(await steps.locator('input.o03-short').evaluateAll(list=>list.map(i=>i.value+'|'+(i.getAttribute('placeholder')??''))),['|','|','|']);
  await button(w.review).click();await steps.locator('.live-error').waitFor();assert.ok((await steps.locator('.live-error').innerText()).includes('invalid_setup_request'));assert.equal(evidence.writes.length,0);await shot('propose-invalid');
  await steps.locator('input[name=o03-rule-approvals]').nth(1).check();
  const fill=async(key,value,unit)=>{await steps.locator(`[aria-describedby=o03-rule-${key}-hint]`).fill(value);await steps.locator(`input[name=o03-rule-${key}-unit]`).nth(unit==='hours'?0:1).check();};
  await fill('validity','2','days');await fill('lead','7','days');await fill('channelAge','23','hours');await page.locator('#o03-rule-reason').fill('Synthetic control rules agreed by finance');
  await button(w.review).click();await steps.locator('.live-error').waitFor();assert.equal(evidence.writes.length,0,'a channel age below 24 hours was accepted');
  await fill('channelAge','25','hours');await controls(where+' filled');await shot('propose-form');
  await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();assert.equal(await active(),'H5');
  const reviewText=await steps.locator('.o03-setup-review').innerText();assert.ok(reviewText.includes('Synthetic control rules agreed by finance'));
  if(lang==='en'){for(const expected of ['2 days','7 days','25 hours','The first proposal cannot be replaced or discarded','Permission is checked when it is sent','Reached through issuance settings','The whole installation'])assert.ok(reviewText.includes(expected),expected);}
  await noOverflow(where+' review');await shot('propose-review');
  await button(w.back).click();assert.equal(await steps.locator('[aria-describedby=o03-rule-channelAge-hint]').inputValue(),'25');await button(w.review).click();
  await steps.locator('.o03-setup-review').getByRole('button',{name:w.propose,exact:true}).click();await results.locator('.o03-setup-outcome .o03-tag').waitFor();await page.locator('.o03-part .o03-docs-scope').waitFor();
  assert.equal(evidence.writes.length,1);const sent=evidence.writes[0];
  assert.deepEqual({operation:sent.operation,rationale:sent.rationale,expected:sent.expected,target:sent.target,args:Object.fromEntries(Object.entries(sent.body).filter(([k])=>k!=='rules'))},{operation:'propose_protected_rules',rationale:'Synthetic control rules agreed by finance',expected:'r1',target:contract.id,args:{configuration:'configuration-a',digest:hash,approvals:'2',validity_seconds:'172800',max_lead_seconds:'604800',channel_age_seconds:'90000'}});
  assert.equal(await steps.getByRole('button').count(),0,'the proposer is offered confirmation');
  if(lang==='en'){assert.ok((await steps.innerText()).includes('You proposed these rules'));assert.ok((await results.innerText()).includes('Recorded. Current issuance eligibility was not checked.'));}
  assert.equal(await guarded(),false);await noOverflow(where+' proposed');await shot('proposed');evidence.cases.push({check:'propose',lang,width});

  reset(proposed('synthetic-proposer'));await open(lang,width);
  await button(w.confirm).click();await page.locator('#o03-rule-confirm-reason').waitFor();await button(w.review).click();await steps.locator('.live-error').waitFor();assert.ok((await steps.locator('.live-error').innerText()).includes('invalid_confirm'));
  await page.locator('#o03-rule-confirm-reason').fill('Synthetic confirmation reason');assert.equal(await guarded(),true,'unsent input is not guarded');
  await button(w.review).click();await steps.locator('.o03-setup-review').waitFor();await shot('confirm-review');
  fault='refuse';await steps.locator('.o03-setup-review').getByRole('button',{name:w.confirm,exact:true}).click();await results.locator('.o03-setup-outcome').filter({hasText:'self_approval_denied'}).waitFor();await page.locator('#o03-rule-steps').waitFor();
  if(lang==='en'){const refused=await results.innerText();assert.ok(refused.includes('Refused. This request recorded nothing.'));assert.ok(refused.includes('The person who proposed these rules cannot confirm them.'));}
  await button(w.confirm).click();assert.equal(await page.locator('#o03-rule-confirm-reason').inputValue(),'Synthetic confirmation reason','the reason was lost after a refusal');await shot('refused');
  fault='unknown';failRead=true;await button(w.review).click();await steps.locator('.o03-setup-review').getByRole('button',{name:w.confirm,exact:true}).click();await result(w.retry).waitFor();
  await page.locator('.o03-part > .live-error').waitFor();assert.equal(await page.locator('#o03-rule-steps').count(),0);assert.equal(await guarded(),true);
  const unknown=await results.locator('.o03-setup-outcome').first().innerText();assert.ok(unknown.includes('dependency_unavailable'));if(lang==='en')assert.ok(!unknown.includes('could not be completed'));
  await shot('unknown-read-failed');
  await page.locator('.o03-parts a').nth(2).click();await page.locator('#o03-part-title').waitFor();assert.equal(await result(w.retry).count(),1,'the unknown result is lost when another section is opened');
  await page.locator('.o03-parts a').nth(1).click();failRead=false;await page.locator('.o03-part .o03-docs-heading button').click();await page.locator('#o03-rule-blocked').waitFor();
  assert.equal(await button(w.confirm).isDisabled(),true);assert.equal(await button(w.confirm).getAttribute('aria-describedby'),'o03-rule-blocked');await shot('unknown');
  await result(w.retry).click();await page.waitForFunction(()=>!document.querySelector('section.o03-results[aria-busy]'));await result(w.retry).waitFor();
  assert.equal(evidence.writes.length,3);assert.equal(evidence.writes[2].key,evidence.writes[1].key);
  fault='';await result(w.recover).click();
  await page.waitForFunction(()=>document.querySelectorAll('section.o03-results .o03-setup-outcome').length===3&&!document.querySelector('section.o03-results .o03-setup-outcome button')&&!document.querySelector('section.o03-results[aria-busy]')&&!document.querySelector('#o03-rule-steps'));
  assert.equal(evidence.writes.length,4);assert.notEqual(evidence.writes[3].key,evidence.writes[1].key);assert.deepEqual(evidence.writes[3].body,evidence.writes[1].body);assert.equal(evidence.writes[3].rationale,'Synthetic confirmation reason');assert.equal(evidence.writes[3].expected,'r1');
  assert.equal(await guarded(),false,'the guard stays on after everything settled');
  if(lang==='en')assert.ok((await page.locator('.o03-part:not(.o03-results)').innerText()).includes('In effect'));
  await noOverflow(where+' settled');await shot('settled');evidence.cases.push({check:'confirm-refusal-unknown-recovery',lang,width});
 }
 assert.deepEqual(evidence.errors,[]);
}finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
console.log(`cases ${evidence.cases.length} unexpected ${JSON.stringify([...new Set(evidence.unexpected)])}`);
