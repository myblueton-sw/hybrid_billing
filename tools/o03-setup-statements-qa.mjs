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
const row=(n,extra)=>({attestation:`statement-${String(n).padStart(2,'0')}`,digest:hash,purpose:'channel_registration',target:'channel-05',actor:'synthetic-registrar-with-a-long-identifier',recorded_at:'2026-11-21T01:00:00Z',statement:'Synthetic statement: I registered the portal channel after receiving the signed form in person.',locator:'',state:'consumed',assurance:'human_attestation',...extra});
const statements=[
 row(1,{}),
 row(2,{purpose:'channel_admission',actor:'synthetic-admitter',locator:'Synthetic shared drive, folder 12'}),
 row(3,{purpose:'change_instruction',target:'change-06',actor:'synthetic-requester',state:'unconsumed',statement:'Synthetic statement written in two lines.\nSecond line of the statement with a-very-long-unbroken-token-'+'x'.repeat(80)}),
 row(4,{purpose:'change_verification',target:'change-06',actor:'synthetic-verifier',state:'withdrawn',statement:'بيان تجريبي كتبه المتحقق بالعربية.'}),
 row(5,{purpose:'change_verification',target:'change-06',actor:'synthetic-verifier',statement:'합성 진술: 등록된 채널로 다시 연락하여 확인했습니다.'}),
];
const many=Array.from({length:12},(_,i)=>row(i+11,{}));
const catalog=[['synthetic_recipient_a','document_recipient'],['synthetic_recipient_b','document_recipient'],['synthetic_template_a','document_template'],['synthetic_template_b','document_template'],['synthetic_account_a','payment_account'],['synthetic_account_b','payment_account'],['synthetic_channel_customer_a','channel_customer'],['synthetic_channel_customer_b','channel_customer'],['synthetic_channel_document_kind_a','channel_document_kind'],['synthetic_channel_document_kind_b','channel_document_kind'],['synthetic_channel_entity_a','channel_entity'],['synthetic_channel_entity_b','channel_entity'],['synthetic_unregistered_sender','unregistered_sender']];
const refusals={denied:[403,'permission_denied'],stale:[409,'evidence_stale'],prerequisite:[422,'prerequisite_failed'],failed:[503,'dependency_unavailable']};
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Built console with intercepted synthetic reads; no server, no mutation, no owner acceptance',cases:[],unexpected:[],errors:[],reads:0};
let mode='rows';
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
 if(command==='issuance_setup.attestations'){
  evidence.reads++;const {after='',snapshot='',...scope}=body.body;assert.deepEqual(scope,{configuration:'configuration-a',digest:hash});assert.equal(!!after,!!snapshot);
  if(refusals[mode])return route.fulfill({status:refusals[mode][0],json:{code:refusals[mode][1],request_id:body.request_id}});
  const all=mode==='empty'?[]:mode==='paged'?many:statements,rows=after?all.filter(r=>r.attestation>after):all.slice(0,10);
  return done({schema:'hb_issuance_setup_attestation_page_v1',contract:contract.id,configuration:'configuration-a',digest:hash,current_eligibility:'unchecked',snapshot:hash,count:String(all.length),next:!after&&all.length>10?rows.at(-1).attestation:'',...Object.fromEntries(rows.map((r,i)=>[`row_${String(i).padStart(2,'0')}`,JSON.stringify(r)]))});
 }
 if(command==='issuance_setup.references'){
  evidence.reads++;assert.deepEqual(body.body,{configuration:'configuration-a',digest:hash});
  if(mode==='catalog-denied')return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
  return done({schema:'hb_issuance_setup_references_v1',contract:contract.id,configuration:'configuration-a',digest:hash,current_eligibility:'unchecked',...Object.fromEntries(catalog.map(([code,use],i)=>[`reference_${String(i).padStart(2,'0')}`,JSON.stringify({code,use,applicability:'synthetic_only'})]))});
 }
 if(['issuance_setup.context','issuance_setup.options'].includes(command))return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
const open=async(lang,width)=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/contracts?view=documents&contract=${contract.id}&on=${on}&part=statements&lang=${lang}`);await page.locator('.o03-scope-bar').waitFor();};
const neutral=c=>{const m=c.match(/\d+/g).map(Number);return Math.max(m[0],m[1],m[2])-Math.min(m[0],m[1],m[2])<=16;};
const banned=/\b(ready|eligible|verified|proven|usable|mature|evidence)\b/i;
const rowsReady=n=>page.waitForFunction(n=>document.querySelectorAll('.o03-statements-table tbody tr').length===n,n);
const layout=()=>page.locator('.o03-part').evaluate(el=>{const rows=[...el.querySelectorAll('.o03-statements-table tbody tr')];return {text:el.innerText,all:el.textContent,samples:el.querySelectorAll('.o03-samples > div').length,buttons:[...el.querySelectorAll('button')].map(b=>Math.round(b.getBoundingClientRect().height)),overflow:document.documentElement.scrollWidth>innerWidth,tags:[...el.querySelectorAll('.o03-tag')].map(t=>{const s=getComputedStyle(t);return {color:s.borderTopColor,radius:s.borderRadius,start:s.borderInlineStartWidth,top:s.borderTopWidth};}),cells:rows.map(tr=>[...tr.children].map(td=>{const r=td.getBoundingClientRect(),inner=Math.max(r.top,...[...td.querySelectorAll('*')].filter(e=>!e.parentElement.closest('details:not([open])')||e.tagName==='SUMMARY').map(e=>e.getBoundingClientRect().bottom)),wide=Math.max(r.right,...[...td.querySelectorAll('p')].filter(e=>!e.closest('details:not([open])')).map(e=>e.getBoundingClientRect().right));return {top:Math.round(r.top),bottom:Math.round(r.bottom),inner:Math.round(inner),left:Math.round(r.left),right:Math.round(r.right),wide:Math.round(wide),display:getComputedStyle(td).display};})),width:innerWidth};});
const checkCells=(seen,where)=>{for(const [r,row] of seen.cells.entries())for(const [i,cell] of row.entries()){assert.ok(cell.inner<=cell.bottom+1,`content leaves cell ${r}/${i} ${where} ${JSON.stringify(cell)}`);assert.ok(cell.left>=0&&cell.right<=seen.width+1&&cell.wide<=seen.width+1,`cell outside ${r}/${i} ${where} ${JSON.stringify(cell)}`);if(cell.display==='block'&&i)assert.ok(cell.top>=row[i-1].bottom-1,`cells overlap ${r}/${i} ${where}`);}};
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  const where=`${lang} ${width}`;
  mode='rows';await open(lang,width);await rowsReady(5);await page.locator('.o03-samples > div').nth(12).waitFor();
  let seen=await layout();
  assert.equal(seen.overflow,false,`overflow ${where}`);assert.equal(seen.samples,13);assert.equal(seen.buttons.length,1);assert.ok(seen.buttons.every(h=>h>=44));assert.equal(seen.tags.length,3);
  for(const t of seen.tags){assert.ok(neutral(t.color));assert.equal(t.radius,'0px');assert.equal(t.start,t.top);}
  checkCells(seen,where);assert.ok(!seen.text.includes('synthetic_'),'a sample code is shown');
  if(lang==='en'){
   assert.ok(!banned.test(seen.text.replace('Issuance eligibility: not checked','')),`wording ${where}`);
   for(const expected of ['Not used — supports no record','Withdrawn — supports no record','Channel registration','Channel acceptance','Change instruction','Change verification','Sample recipient A','Sample issuing entity channel B','Sample unregistered sender','Instruction from a sender that is not a registered channel'])assert.ok(seen.text.includes(expected),expected);
   assert.ok(!seen.text.includes('I registered the portal channel'),'statement text shown outside its disclosure');
  }
  await page.screenshot({path:`${output}/rows-${lang}-${width}.png`,fullPage:true});
  for(const n of [1,2,3])await page.locator('.o03-statements-table tbody tr').nth(n).locator('summary').click();
  seen=await layout();checkCells(seen,where+' opened');assert.equal(seen.overflow,false,`overflow opened ${where}`);
  const opened=await page.locator('.o03-statements-table tbody tr').nth(2).locator('.o03-statement').evaluate(p=>({space:getComputedStyle(p).whiteSpace,lines:p.innerText.split('\n').length}));assert.equal(opened.space,'pre-wrap');assert.equal(opened.lines,2);
  if(lang==='en'){assert.ok(seen.text.includes('Synthetic shared drive, folder 12'));assert.ok(seen.text.includes('Where the source is kept: Not stated'));}
  await page.screenshot({path:`${output}/opened-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'rows',lang,width});
  mode='empty';await open(lang,width);await page.locator('.o03-part .live-empty').waitFor();evidence.cases.push({check:'empty',lang,width});
  if(width===1440)await page.screenshot({path:`${output}/empty-${lang}-${width}.png`,fullPage:true});
  mode='paged';await open(lang,width);await rowsReady(10);
  await page.locator('.o03-docs-pages button').click();await rowsReady(2);assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-part-title');
  await page.locator('.o03-docs-pages button').click();await rowsReady(10);evidence.cases.push({check:'paging',lang,width});
  for(const name of Object.keys(refusals)){
   mode=name;await open(lang,width);const alert=page.locator('.o03-part > .live-error');await alert.waitFor();await page.locator('.o03-samples > div').nth(12).waitFor();
   const refusal=await alert.evaluate(el=>{const s=getComputedStyle(el);return {text:el.innerText,widths:[s.borderTopWidth,s.borderBottomWidth,s.borderInlineStartWidth,s.borderInlineEndWidth],color:s.borderTopColor,overflow:document.documentElement.scrollWidth>innerWidth};});
   assert.ok(refusal.text.includes(refusals[name][1]));assert.equal(refusal.overflow,false);assert.equal(refusal.widths[2],'0px');assert.ok(neutral(refusal.color));
   assert.equal(await page.locator('.o03-statements-table').count(),0);
   if(name==='denied'&&width===1440)await page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});
   mode='rows';await page.locator('.o03-docs-heading button').click();await rowsReady(5);assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-part-title');
   evidence.cases.push({check:'refusal',lang,width,mode:name});
  }
  mode='catalog-denied';await open(lang,width);await page.locator('.o03-part-sub .live-error').waitFor();await rowsReady(5);assert.equal(await page.locator('.o03-samples').count(),0);
  evidence.cases.push({check:'catalog-denied',lang,width});
 }
 assert.deepEqual(evidence.errors,[]);
}finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
console.log(`cases ${evidence.cases.length} reads ${evidence.reads} unexpected ${JSON.stringify([...new Set(evidence.unexpected)])}`);
