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
const group=(prefix,kind,extra={})=>({[`${prefix}_subject_kind`]:kind,[`${prefix}_changes`]:'0',[`${prefix}_other_provenance`]:'0',[`${prefix}_unowned_module`]:'0',[`${prefix}_latest_scheduled`]:'',[`${prefix}_latest_scheduled_effective`]:'',...extra});
const rules={rules_state:'confirmed',rules_observation:'active',rules_id:'synthetic-rules',rules_digest:hash,rules_approvals:'2',rules_validity_seconds:'172800',rules_max_lead_seconds:'604800',rules_channel_age_seconds:'86400',rules_proposer:'synthetic-proposer',rules_confirmer:'synthetic-confirmer',rules_pending_confirmer:'',rules_proposed_at:'2026-11-21T01:00:00Z',rules_confirmed_at:'2026-11-21T02:00:00Z'};
const base={digest:hash,provenance:'staff_entry',instruction_kind:'in_person',channel:'channel-05',rules:'current',requester:'synthetic-requester-with-a-long-identifier',effective:'2026-12-01T00:00:00Z',state:'requested',observation:'requested',phase:'recorded',requested_at:'2026-11-22T04:00:00Z',approvals_required:'2',verifier:'',verification_outcome:'',verified_at:'',pending_verifier:'',pending_decision_actor:'',ended_by:'',ended_at:'',decisions:[]};
const verified={verifier:'synthetic-verifier',verification_outcome:'confirmed',verified_at:'2026-11-22T05:00:00Z'};
const approve=(n,at)=>({approver:`synthetic-approver-${n}`,verdict:'approve',at});
const changes=[
 {...base,change:'change-01',phase:'request_unknown',state:'',observation:'absent',requested_at:'',approvals_required:''},
 {...base,change:'change-02',provenance:'customer_document',instruction_kind:'postal'},
 {...base,change:'change-03',phase:'verification_unknown',pending_verifier:'synthetic-verifier',provenance:'customer_portal',instruction_kind:'portal'},
 {...base,change:'change-04',...verified,state:'confirmed',observation:'confirmed',instruction_kind:'email'},
 {...base,change:'change-05',...verified,phase:'decision_unknown',state:'confirmed',observation:'confirmed',pending_decision_actor:'synthetic-approver-2',decisions:[approve(1,'2026-11-22T06:00:00Z')]},
 {...base,change:'change-06',...verified,state:'scheduled',observation:'scheduled',decisions:[approve(1,'2026-11-22T06:00:00Z'),approve(2,'2026-11-22T07:00:00Z')]},
 {...base,change:'change-07',...verified,verification_outcome:'denied',state:'rejected',observation:'rejected',ended_by:'synthetic-approver-1',ended_at:'2026-11-22T08:00:00Z',decisions:[{approver:'synthetic-approver-1',verdict:'reject',at:'2026-11-22T08:00:00Z'}],instruction_kind:'phone'},
 {...base,change:'change-08',observation:'lapsed',ended_at:'2026-12-01T00:00:00Z',rules:'other'},
];
const byField={document_recipient:changes,document_template:[],payment_account:[{...changes[5],change:'change-21'}]};
const many=Array.from({length:12},(_,i)=>({...changes[1],change:`change-${String(i+31)}`}));
const refusals={denied:[403,'permission_denied'],stale:[409,'evidence_stale'],prerequisite:[422,'prerequisite_failed'],failed:[503,'dependency_unavailable']};
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Built console with intercepted synthetic reads; no server, no mutation, no owner acceptance',cases:[],unexpected:[],errors:[],reads:0};
let mode='rows';const fields=[];
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
 const command=path.split('/').at(-1),body=request.postDataJSON(),done=fields=>route.fulfill({json:{request_id:body.request_id,status:'completed',target_id:body.target_id,result_revision:'r1',replay:false,as_of:now,fields}});
 if(command==='issuance_setup.configurations')return done({schema:'hb_issuance_configuration_page_v1',contract:contract.id,current_eligibility:'unchecked',snapshot:hash,count:'1',next:'',row_00:JSON.stringify(configuration)});
 if(command==='issuance_setup.protected'){
  evidence.reads++;assert.deepEqual(body.body,{configuration:'configuration-a',digest:hash});
  if(mode==='summary-denied')return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
  return done({schema:'hb_issuance_protected_setup_v1',contract:contract.id,configuration:'configuration-a',digest:hash,current_eligibility:'unchecked',channels:'7',channels_other_provenance:'0',channels_unowned_module:'0',...rules,...group('recipient','customer',{recipient_changes:'8'}),...group('template','document_kind',{template_unowned_module:'1'}),...group('account','entity',{account_changes:'1',account_other_provenance:'2',account_latest_scheduled:'change-21',account_latest_scheduled_effective:'2026-12-01T00:00:00Z'})});
 }
 if(command==='issuance_setup.protected_changes'){
  evidence.reads++;const {after='',snapshot='',field,...scope}=body.body;assert.deepEqual(scope,{configuration:'configuration-a',digest:hash});assert.equal(!!after,!!snapshot);fields.push(field);
  if(refusals[mode])return route.fulfill({status:refusals[mode][0],json:{code:refusals[mode][1],request_id:body.request_id}});
  const all=(mode==='paged'?many:byField[field]).map(r=>({...r,field})),at=all.findIndex(r=>r.change===after),rows=after?all.slice(at+1):all.slice(0,10);
  return done({schema:'hb_issuance_protected_change_page_v1',contract:contract.id,configuration:'configuration-a',digest:hash,current_eligibility:'unchecked',field,snapshot:hash,count:String(all.length),next:!after&&all.length>10?rows.at(-1).change:'',other_provenance:'0',unowned_module:'0',...Object.fromEntries(rows.map((r,i)=>[`row_${String(i).padStart(2,'0')}`,JSON.stringify(r)]))});
 }
 if(['issuance_setup.context','issuance_setup.options','issuance_setup.protected_channels'].includes(command))return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
const open=async(lang,width)=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/contracts?view=documents&contract=${contract.id}&on=${on}&part=changes&lang=${lang}`);await page.locator('.o03-scope-bar').waitFor();};
const neutral=c=>{const m=c.match(/\d+/g).map(Number);return Math.max(m[0],m[1],m[2])-Math.min(m[0],m[1],m[2])<=16;};
const banned=/\b(ready|eligible|verified|proven|usable|mature)\b/i;
const rowsReady=n=>page.waitForFunction(n=>document.querySelectorAll('.o03-part tbody tr').length===n,n);
const layout=()=>page.locator('.o03-part').evaluate(el=>{const rows=[...el.querySelectorAll('tbody tr')];return {text:el.innerText,rows:rows.length,cards:[...el.querySelectorAll('.o03-field-cards > li')].map(li=>{const s=getComputedStyle(li),r=li.getBoundingClientRect();return {current:li.getAttribute('aria-current'),color:s.borderTopColor,widths:[s.borderTopWidth,s.borderInlineStartWidth],radius:s.borderRadius,shadow:s.boxShadow,background:s.backgroundColor,left:Math.round(r.left),right:Math.round(r.right),buttons:[...li.querySelectorAll('button')].map(b=>Math.round(b.getBoundingClientRect().height))};}),overflow:document.documentElement.scrollWidth>innerWidth,tags:[...el.querySelectorAll('.o03-tag')].map(t=>{const s=getComputedStyle(t);return {color:s.borderTopColor,radius:s.borderRadius,start:s.borderInlineStartWidth,top:s.borderTopWidth};}),cells:rows.map(tr=>[...tr.children].map(td=>{const r=td.getBoundingClientRect(),inner=Math.max(r.top,...[...td.querySelectorAll('*')].filter(e=>!e.parentElement.closest('details:not([open])')||e.tagName==='SUMMARY').map(e=>e.getBoundingClientRect().bottom));return {top:Math.round(r.top),bottom:Math.round(r.bottom),inner:Math.round(inner),left:Math.round(r.left),right:Math.round(r.right),display:getComputedStyle(td).display};})),width:innerWidth};});
const checkCells=(seen,where)=>{for(const [r,row] of seen.cells.entries())for(const [i,cell] of row.entries()){assert.ok(cell.inner<=cell.bottom+1,`content leaves cell ${r}/${i} ${where} ${JSON.stringify(cell)}`);assert.ok(cell.left>=0&&cell.right<=seen.width+1,`cell outside ${r}/${i} ${where}`);if(cell.display==='block'&&i)assert.ok(cell.top>=row[i-1].bottom-1,`cells overlap ${r}/${i} ${where}`);}};
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  const where=`${lang} ${width}`;
  mode='rows';await open(lang,width);await page.locator('.o03-field-cards > li').nth(2).waitFor();await rowsReady(8);
  let seen=await layout();
  assert.equal(seen.overflow,false,`overflow ${where}`);assert.equal(seen.cards.length,3);assert.deepEqual(seen.cards.map(c=>c.current),['true',null,null]);
  for(const card of seen.cards){assert.ok(neutral(card.color));assert.equal(card.widths[0],card.widths[1]);assert.equal(card.radius,'0px');assert.equal(card.shadow,'none');assert.ok(neutral(card.background));assert.ok(card.left>=0&&card.right<=seen.width);assert.ok(card.buttons.every(h=>h>=44));}
  for(const t of seen.tags){assert.ok(neutral(t.color));assert.equal(t.radius,'0px');assert.equal(t.start,t.top);}
  checkCells(seen,where);
  if(lang==='en'){
   assert.ok(!banned.test(seen.text.replace('Issuance eligibility: not checked','').replaceAll('Each is verified and decided','')),`wording ${where}`);
   for(const expected of ['Request outcome unknown','No change observed','Verification outcome unknown','Verifier pinned, not yet observed','Verification recorded, awaiting decision','Decision outcome unknown','Deciding person pinned, not yet observed','Scheduled','Rejected','Lapsed','Recorded under rules other than the current rules','This change has an unknown outcome','An entry exists that this service did not record','Latest scheduled change','Recorded history. It does not state that an invoice draft can use this change.','Changes to: Document recipient','Shown below'])assert.ok(seen.text.includes(expected),expected);
   assert.equal(seen.text.split('This change has an unknown outcome').length-1,3);
  }
  await page.screenshot({path:`${output}/rows-${lang}-${width}.png`,fullPage:true});
  await page.locator('.o03-part tbody tr').nth(0).locator('details').nth(1).locator('summary').click();await page.locator('.o03-part tbody tr').nth(7).locator('details').nth(1).locator('summary').click();await page.locator('.o03-part tbody tr').nth(6).locator('details').nth(1).locator('summary').click();await page.locator('.o03-part tbody tr').nth(5).locator('details').nth(1).locator('summary').click();
  const events=await page.locator('.o03-part tbody tr').nth(6).locator('.o03-events li').allInnerTexts();assert.equal(events.length,3);
  if(lang==='en'){assert.ok(events[0].startsWith('Rejected by'));assert.ok(events[1].startsWith('Verifier stated: denied'));assert.ok(events[2].startsWith('Requested by'));}
  const attempted=await page.locator('.o03-part tbody tr').nth(0).locator('.o03-events li').allInnerTexts(),lapsed=await page.locator('.o03-part tbody tr').nth(7).locator('.o03-events li').allInnerTexts();assert.equal(attempted.length,1);assert.equal(lapsed.length,2);
  if(lang==='en'){assert.ok(attempted[0].startsWith('Request attempted by'));assert.ok(attempted[0].includes('Not observed'));assert.ok(!attempted[0].includes('Requested by'));assert.ok(lapsed[0].startsWith('Ended'));assert.ok(!lapsed[0].startsWith('Ended by'));assert.ok(seen.text.includes('Decisions observed'));assert.ok(!seen.text.includes('Service record'));assert.equal(seen.text.split('Step outcome').length-1,3);}
  assert.equal((await page.locator('.o03-part tbody tr').nth(5).locator('.o03-events li').allInnerTexts()).length,4);
  checkCells(await layout(),where+' events');
  await page.screenshot({path:`${output}/events-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'rows',lang,width});
  await page.locator('.o03-field-cards > li').nth(2).locator('button').click();await rowsReady(1);
  assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-changes-title');seen=await layout();assert.deepEqual(seen.cards.map(c=>c.current),[null,null,'true']);assert.equal(fields.at(-1),'payment_account');
  await page.locator('.o03-field-cards > li').nth(1).locator('button').click();await page.locator('.o03-part-sub .live-empty').waitFor();assert.equal(fields.at(-1),'document_template');
  await page.screenshot({path:`${output}/empty-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'fields',lang,width});
  mode='paged';await open(lang,width);await rowsReady(10);
  await page.locator('.o03-part-sub:not(.o03-steps) > .o03-docs-pages button').click();await rowsReady(2);assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-changes-title');
  await page.locator('.o03-part-sub:not(.o03-steps) > .o03-docs-pages button').click();await rowsReady(10);evidence.cases.push({check:'paging',lang,width});
  for(const name of Object.keys(refusals)){
   mode=name;await open(lang,width);const alert=page.locator('.o03-part-sub .live-error');await alert.waitFor();
   const refusal=await alert.evaluate(el=>{const s=getComputedStyle(el);return {text:el.innerText,widths:[s.borderTopWidth,s.borderBottomWidth,s.borderInlineStartWidth,s.borderInlineEndWidth],color:s.borderTopColor,overflow:document.documentElement.scrollWidth>innerWidth};});
   assert.ok(refusal.text.includes(refusals[name][1]));assert.equal(refusal.overflow,false);assert.equal(refusal.widths[2],'0px');assert.ok(neutral(refusal.color));
   assert.equal(await page.locator('.o03-part table').count(),0);assert.equal(await page.locator('.o03-field-cards > li').count(),3);
   if(name==='stale')await page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});
   mode='rows';await page.locator('.o03-docs-heading button').click();await rowsReady(8);assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-part-title');
   evidence.cases.push({check:'refusal',lang,width,mode:name});
  }
  mode='summary-denied';await open(lang,width);await page.locator('.o03-part > .live-error').waitFor();await rowsReady(8);
  assert.equal(await page.locator('.o03-field-cards').count(),0);
  await page.screenshot({path:`${output}/summary-denied-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'summary-denied',lang,width});
 }
 assert.deepEqual(evidence.errors,[]);
}finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
console.log(`cases ${evidence.cases.length} reads ${evidence.reads} unexpected ${JSON.stringify([...new Set(evidence.unexpected)])}`);
