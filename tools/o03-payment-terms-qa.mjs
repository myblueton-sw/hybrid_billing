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
const second={...configuration,configuration:'configuration-b'};
const noTerm={term_state:'none',term_observation:'absent',term:'',term_digest:'',term_revision:'',net_days:'',mode:'',term_author:'',term_approver:'',term_pending_approver:'',term_written_at:'',term_approved_at:''};
const term={term:'synthetic-term',term_digest:hash,term_revision:'1',net_days:'30',mode:'calendar',term_author:'synthetic-author-with-a-long-identifier',term_approver:'',term_pending_approver:'',term_written_at:'2026-11-21T01:00:00Z',term_approved_at:''};
const termModes={
 none:noTerm,
 registration_unknown:{term_state:'registration_unknown',term_observation:'absent',...term,term_written_at:''},
 pending:{term_state:'pending',term_observation:'pending',...term},
 approval_unknown:{term_state:'approval_unknown',term_observation:'pending',...term,term_pending_approver:'synthetic-approver'},
 approved:{term_state:'approved',term_observation:'approved',...term,term_approver:'synthetic-approver',term_approved_at:'2026-11-21T02:00:00Z'},
};
const bound={digest:hash,origin:'configuration',configuration:'configuration-a',term:'synthetic-term',term_revision:'1',net_days:'30',mode:'calendar',zone:'UTC',from:'2026-11-01',to:'2027-10-31',reason:'initial',state:'pending',observation:'pending',author:'synthetic-author-with-a-long-identifier',approver:'',pending_approver:'',written_at:'2026-11-21T03:00:00Z',approved_at:''};
const bindings=[
 {binding:'binding-01',...bound,state:'registration_unknown',observation:'absent',written_at:'',to:'',from:'2029-01-01'},
 {binding:'binding-02',...bound,reason:'renegotiation',from:'2027-11-01',to:'2028-10-31'},
 {binding:'binding-03',...bound,state:'approval_unknown',pending_approver:'synthetic-approver',configuration:'configuration-z',reason:'correction',from:'2025-01-01',to:'2025-12-31'},
 {binding:'binding-04',...bound,state:'approved',observation:'approved',approver:'synthetic-approver',approved_at:'2026-11-21T04:00:00Z',origin:'profile',configuration:'',from:'2024-01-01',to:'2024-12-31'},
 {binding:'binding-05',...bound,state:'approval_unknown',observation:'approved',approver:'synthetic-approver',approved_at:'2026-11-21T04:00:00Z'},
];
const many=Array.from({length:12},(_,i)=>({...bindings[4],binding:`binding-${String(i+11)}`,from:`20${30+i}-01-01`,to:`20${30+i}-12-31`}));
const refusals={denied:[403,'permission_denied'],stale:[409,'evidence_stale'],prerequisite:[422,'prerequisite_failed'],failed:[503,'dependency_unavailable']};
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Built console with intercepted synthetic reads; no server, no mutation, no owner acceptance',cases:[],unexpected:[],errors:[],reads:0};
let mode='approved';
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
 if(command==='issuance_setup.configurations'){if(mode==='history-denied')return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});const list=mode==='no-scope'?[configuration,second]:[configuration];return done({schema:'hb_issuance_configuration_page_v1',contract:contract.id,current_eligibility:'unchecked',snapshot:hash,count:String(list.length),next:'',...Object.fromEntries(list.map((r,i)=>[`row_0${i}`,JSON.stringify(r)]))});}
 if(command==='issuance_setup.terms'){
  evidence.reads++;const {after='',snapshot='',...scope}=body.body;assert.deepEqual(scope,{configuration:'configuration-a',digest:hash});assert.equal(!!after,!!snapshot);
  if(refusals[mode])return route.fulfill({status:refusals[mode][0],json:{code:refusals[mode][1],request_id:body.request_id}});
  const all=termModes[mode]?(mode==='none'?[]:bindings):mode==='paged'?many:bindings,rows=after?all.filter(r=>r.binding>after):all.slice(0,10);
  return done({schema:'hb_issuance_setup_terms_v1',contract:contract.id,configuration:'configuration-a',digest:hash,zone:'UTC',current_eligibility:'unchecked',...(termModes[mode]??termModes.approved),snapshot:hash,count:String(all.length),next:!after&&all.length>10?rows.at(-1).binding:'',...Object.fromEntries(rows.map((r,i)=>[`row_${String(i).padStart(2,'0')}`,JSON.stringify(r)]))});
 }
 if(['issuance_setup.context','issuance_setup.options'].includes(command))return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
const open=async(lang,width,wait='.o03-scope-bar')=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/contracts?view=payment-terms&contract=${contract.id}&on=${on}&lang=${lang}`);await page.locator(wait).waitFor();};
const neutral=c=>{const m=c.match(/\d+/g).map(Number);return Math.max(m[0],m[1],m[2])-Math.min(m[0],m[1],m[2])<=16;};
const banned=/\b(ready|eligible|verified|proven|usable|mature|overdue soon|due on)\b/i;
const rowsReady=n=>page.waitForFunction(n=>document.querySelectorAll('.o03-bindings-table tbody tr').length===n,n);
const layout=()=>page.locator('.o03-part').evaluate(el=>{const rows=[...el.querySelectorAll('.o03-bindings-table tbody tr')];return {text:el.innerText,all:el.textContent,groups:[...el.querySelectorAll('.o03-docs-scope h4')].map(h=>h.innerText),terms:el.querySelectorAll('.o03-docs-scope dt').length,buttons:[...el.querySelectorAll('.o03-docs-heading button')].map(b=>Math.round(b.getBoundingClientRect().height)),overflow:document.documentElement.scrollWidth>innerWidth,tags:[...el.querySelectorAll('.o03-tag')].map(t=>{const s=getComputedStyle(t);return {color:s.borderTopColor,radius:s.borderRadius,start:s.borderInlineStartWidth,top:s.borderTopWidth};}),cells:rows.map(tr=>[...tr.children].map(td=>{const r=td.getBoundingClientRect(),inner=Math.max(r.top,...[...td.querySelectorAll('*')].filter(e=>!e.parentElement.closest('details:not([open])')||e.tagName==='SUMMARY').map(e=>e.getBoundingClientRect().bottom));return {top:Math.round(r.top),bottom:Math.round(r.bottom),inner:Math.round(inner),left:Math.round(r.left),right:Math.round(r.right),display:getComputedStyle(td).display};})),width:innerWidth};});
const checkCells=(seen,where)=>{for(const [r,row] of seen.cells.entries())for(const [i,cell] of row.entries()){assert.ok(cell.inner<=cell.bottom+1,`content leaves cell ${r}/${i} ${where} ${JSON.stringify(cell)}`);assert.ok(cell.left>=0&&cell.right<=seen.width+1,`cell outside ${r}/${i} ${where}`);if(cell.display==='block'&&i)assert.ok(cell.top>=row[i-1].bottom-1,`cells overlap ${r}/${i} ${where}`);}};
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  const where=`${lang} ${width}`;
  assert.equal(await (async()=>{mode='approved';await open(lang,width);await rowsReady(5);return page.locator('#o03-view-title').innerText();})(),{en:'Payment terms',ko:'납부 조건',ar:'شروط الدفع'}[lang]);
  let seen=await layout();
  assert.equal(seen.overflow,false,`overflow ${where}`);assert.equal(seen.buttons.length,1);assert.ok(seen.buttons.every(h=>h>=44));assert.equal(seen.terms,2+6);assert.equal(seen.groups.length,2);
  for(const t of seen.tags){assert.ok(neutral(t.color));assert.equal(t.radius,'0px');assert.equal(t.start,t.top);}
  checkCells(seen,where);assert.ok(seen.all.includes('configuration-z'));assert.equal(seen.all.split('configuration-a').length-1,0,'own configuration identifier repeated in rows');
  if(lang==='en'){
   assert.ok(!banned.test(seen.text.replace('Issuance eligibility: not checked','').replace('does not state that an invoice is payable or overdue','')),`wording ${where}`);
   for(const expected of ['30 days','Recording outcome unknown','Nothing observed','Recorded, awaiting approval','Approval outcome unknown','Approver pinned, not yet observed','No end date','Renegotiation','Correction','Other issuance settings','Billing profile','These issuance settings','This screen calculates no due date','Every application to this contract is listed','Applied by','Payment terms of these settings','Net days (calendar)'])assert.ok(seen.text.includes(expected),expected);
  }
  await page.screenshot({path:`${output}/approved-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'rows',lang,width});
  for(const name of ['none','registration_unknown','pending','approval_unknown']){
   mode=name;await open(lang,width);await page.locator(name==='none'?'.o03-part > .live-empty':'.o03-docs-scope .o03-tag').waitFor();seen=await layout();
   assert.equal(seen.overflow,false);assert.equal(seen.terms,name==='none'?0:name==='registration_unknown'?2+3:name==='pending'?2+4:2+5,`facts ${name} ${seen.terms}`);
   if(name==='none'){assert.equal(seen.cells.length,0);assert.ok(!seen.all.includes('synthetic-term'));}
   if(width!==1024)await page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'term',lang,width,mode:name});
  }
  mode='paged';await open(lang,width);await rowsReady(10);
  await page.locator('.o03-part-sub:not(.o03-steps) .o03-docs-pages button').click();await rowsReady(2);assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-part-title');
  await page.locator('.o03-part-sub:not(.o03-steps) .o03-docs-pages button').click();await rowsReady(10);evidence.cases.push({check:'paging',lang,width});
  for(const name of Object.keys(refusals)){
   mode=name;await open(lang,width);const alert=page.locator('.o03-part > .live-error');await alert.waitFor();
   const refusal=await alert.evaluate(el=>{const s=getComputedStyle(el);return {text:el.innerText,widths:[s.borderTopWidth,s.borderBottomWidth,s.borderInlineStartWidth,s.borderInlineEndWidth],color:s.borderTopColor,overflow:document.documentElement.scrollWidth>innerWidth};});
   assert.ok(refusal.text.includes(refusals[name][1]));assert.equal(refusal.overflow,false);assert.equal(refusal.widths[2],'0px');assert.ok(neutral(refusal.color));
   if(lang==='en'&&name==='denied')assert.ok(refusal.text.includes('Inspect payment terms evidence'));
   assert.equal(await page.locator('.o03-bindings-table').count(),0);assert.equal(await page.locator('.o03-docs-scope').count(),0);
   if(name==='denied'&&width===1440)await page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});
   mode='approved';await page.locator('.o03-docs-heading button').click();await rowsReady(5);assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-part-title');
   evidence.cases.push({check:'refusal',lang,width,mode:name});
  }
  mode='history-denied';await open(lang,width,'.o03-part .live-error');assert.equal(await page.locator('.o03-scope-bar').count(),0);assert.equal(await page.locator('.o03-part a').count(),1);
  if(width===1440)await page.screenshot({path:`${output}/history-denied-${lang}-${width}.png`,fullPage:true});
  mode='no-scope';await open(lang,width,'.o03-part a');assert.equal(await page.locator('.o03-scope-bar').count(),0);assert.equal(await page.locator('.o03-part .live-error').count(),0);
  const link=await page.locator('.o03-part a').evaluate(a=>({height:Math.round(a.getBoundingClientRect().height),href:a.getAttribute('href')}));assert.ok(link.height>=44);assert.ok(link.href.includes('view=documents'));assert.ok(!link.href.includes('part='));
  await page.screenshot({path:`${output}/no-scope-${lang}-${width}.png`,fullPage:true});
  await page.locator('.o03-part a').click();await page.locator('.o03-docs-use').first().waitFor();await page.locator('.o03-docs-use').first().click();await page.locator('.o03-docs-selected').waitFor();
  mode='approved';await page.evaluate(()=>{location.hash=location.hash.replace('view=documents','view=payment-terms');});await page.locator('.o03-scope-bar').waitFor();await rowsReady(5);
  assert.ok((await page.locator('.o03-scope-bar').textContent()).includes('configuration-a'));
  evidence.cases.push({check:'scope',lang,width});
 }
 assert.deepEqual(evidence.errors,[]);
}finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
console.log(`cases ${evidence.cases.length} reads ${evidence.reads} unexpected ${JSON.stringify([...new Set(evidence.unexpected)])}`);
