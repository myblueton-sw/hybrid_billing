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
const base={digest:hash,subject_kind:'entity',kind:'portal',registrar:'synthetic-registrar-with-a-long-identifier',admitter:'',pending_admitter:'',registered_at:'2026-11-21T01:00:00Z',admitted_at:'',retired_by:'',retired_at:'',rules:'current',request_not_before:''};
const kinds=[
 {channel:'channel-01',...base,state:'registration_unknown',observation:'absent',registered_at:''},
 {channel:'channel-02',...base,state:'registration_unknown',observation:'registered',subject_kind:'customer',kind:'email'},
 {channel:'channel-03',...base,state:'registered',observation:'registered',subject_kind:'document_kind',kind:'postal'},
 {channel:'channel-04',...base,state:'admission_unknown',observation:'registered',pending_admitter:'synthetic-admitter',kind:'phone'},
 {channel:'channel-05',...base,state:'admitted',observation:'admitted',admitter:'synthetic-admitter',admitted_at:'2026-11-21T02:00:00Z',request_not_before:'2026-11-22T03:00:00Z'},
 {channel:'channel-06',...base,state:'admitted',observation:'admitted',admitter:'synthetic-admitter',admitted_at:'2026-11-21T02:00:00Z',rules:'other'},
 {channel:'channel-07',...base,state:'admitted',observation:'retired',admitter:'synthetic-admitter',admitted_at:'2026-11-21T02:00:00Z',retired_by:'synthetic-retirer',retired_at:'2026-11-23T02:00:00Z'},
];
const many=Array.from({length:12},(_,i)=>({...kinds[4],channel:`channel-${String(i+1).padStart(2,'0')}`}));
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
 if(command==='issuance_setup.protected_channels'){
  evidence.reads++;const {after='',snapshot='',...scope}=body.body;assert.deepEqual(scope,{configuration:'configuration-a',digest:hash});assert.equal(!!after,!!snapshot);
  if(refusals[mode])return route.fulfill({status:refusals[mode][0],json:{code:refusals[mode][1],request_id:body.request_id}});
  const all=mode==='empty'?[]:mode==='paged'?many:kinds,rows=after?all.filter(r=>r.channel>after):all.slice(0,10);
  return done({schema:'hb_issuance_protected_channel_page_v1',contract:contract.id,configuration:'configuration-a',digest:hash,current_eligibility:'unchecked',snapshot:hash,count:String(all.length),next:!after&&all.length>10?rows.at(-1).channel:'',other_provenance:mode==='rows'?'2':'0',unowned_module:mode==='rows'?'1':'0',...Object.fromEntries(rows.map((r,i)=>[`row_${String(i).padStart(2,'0')}`,JSON.stringify(r)]))});
 }
 if(['issuance_setup.context','issuance_setup.options'].includes(command))return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
const open=async(lang,width)=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/contracts?view=documents&contract=${contract.id}&on=${on}&part=channels&lang=${lang}`);await page.locator('.o03-scope-bar').waitFor();};
const neutral=c=>{const m=c.match(/\d+/g).map(Number);return Math.max(m[0],m[1],m[2])-Math.min(m[0],m[1],m[2])<=16;};
const banned=/\b(ready|eligible|verified|proven|usable|mature)\b/i;
const layout=()=>page.locator('.o03-part').evaluate(el=>{const rows=[...el.querySelectorAll('tbody tr')];return {text:el.innerText,rows:rows.length,buttons:[...el.querySelectorAll('.o03-docs-heading button, :scope > .o03-docs-pages button')].map(b=>({text:b.innerText,height:Math.round(b.getBoundingClientRect().height)})),overflow:document.documentElement.scrollWidth>innerWidth,tags:[...el.querySelectorAll('.o03-tag')].map(t=>{const s=getComputedStyle(t);return {color:s.borderTopColor,radius:s.borderRadius,start:s.borderInlineStartWidth,top:s.borderTopWidth};}),cells:rows.slice(0,7).map(tr=>[...tr.children].map(td=>{const r=td.getBoundingClientRect(),inner=Math.max(r.top,...[...td.querySelectorAll('*')].filter(e=>!e.parentElement.closest('details:not([open])')||e.tagName==='SUMMARY').map(e=>e.getBoundingClientRect().bottom));return {top:Math.round(r.top),bottom:Math.round(r.bottom),inner:Math.round(inner),left:Math.round(r.left),right:Math.round(r.right),display:getComputedStyle(td).display};})),width:innerWidth,terms:el.querySelectorAll('dl.o03-part-counts dt').length};});
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  mode='rows';await open(lang,width);await page.locator('.o03-part tbody tr').nth(6).waitFor();
  let seen=await layout();
  assert.equal(seen.rows,7);assert.equal(seen.overflow,false,`overflow ${lang} ${width}`);assert.equal(seen.terms,4);assert.equal(seen.buttons.length,1);assert.ok(seen.buttons.every(b=>b.height>=44));
  for(const t of seen.tags){assert.ok(neutral(t.color));assert.equal(t.radius,'0px');assert.equal(t.start,t.top);}
  for(const [r,row] of seen.cells.entries())for(const [i,cell] of row.entries()){assert.ok(cell.inner<=cell.bottom+1,`content leaves cell ${r}/${i} ${lang} ${width} ${JSON.stringify(cell)}`);assert.ok(cell.left>=0&&cell.right<=seen.width+1,`cell outside ${r}/${i}`);if(cell.display==='block'&&i)assert.ok(cell.top>=row[i-1].bottom-1,`cells overlap ${r}/${i} ${lang} ${width}`);}
  if(lang==='en'){
   assert.ok(!banned.test(seen.text.replace('Issuance eligibility: not checked','')));
   for(const expected of ['Registration outcome unknown','No channel observed','Registered, awaiting acceptance','Acceptance outcome unknown','Accepting person pinned, not yet observed','Recorded under rules other than the current rules','Retired by','Not stated','Issuing entity · Portal','Customer · Email','Document kind · Postal mail'])assert.ok(seen.text.includes(expected),expected);
   assert.equal(seen.text.split('Not stated').length-1,6);
  }
  await page.screenshot({path:`${output}/rows-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'rows',lang,width,rows:seen.rows});
  mode='empty';await open(lang,width);await page.locator('.o03-part .live-empty').waitFor();seen=await layout();
  assert.equal(seen.rows,0);assert.equal(seen.terms,2);assert.equal(seen.buttons.length,1);assert.equal(seen.overflow,false);
  await page.screenshot({path:`${output}/empty-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'empty',lang,width});
  mode='paged';await open(lang,width);await page.locator('.o03-part tbody tr').nth(9).waitFor();seen=await layout();
  assert.equal(seen.rows,10);assert.equal(seen.buttons.length,2);
  await page.locator('.o03-part > .o03-docs-pages button').click();await page.locator('.o03-part > .o03-docs-pages button').first().waitFor();await page.waitForFunction(()=>document.querySelectorAll('.o03-part tbody tr').length===2);
  assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-part-title');seen=await layout();assert.equal(seen.buttons.length,2);
  await page.screenshot({path:`${output}/page-two-${lang}-${width}.png`,fullPage:true});
  await page.locator('.o03-part > .o03-docs-pages button').click();await page.waitForFunction(()=>document.querySelectorAll('.o03-part tbody tr').length===10);
  evidence.cases.push({check:'paging',lang,width});
  for(const name of Object.keys(refusals)){
   mode=name;await open(lang,width);const alert=page.locator('.o03-part .live-error');await alert.waitFor();
   const refusal=await alert.evaluate(el=>{const s=getComputedStyle(el);return {text:el.innerText,widths:[s.borderTopWidth,s.borderBottomWidth,s.borderInlineStartWidth,s.borderInlineEndWidth],color:s.borderTopColor,radius:s.borderRadius,shadow:s.boxShadow,overflow:document.documentElement.scrollWidth>innerWidth};});
   assert.ok(refusal.text.includes(refusals[name][1]));assert.equal(refusal.overflow,false);assert.equal(refusal.radius,'0px');assert.equal(refusal.shadow,'none');assert.equal(refusal.widths[2],'0px');assert.equal(refusal.widths[3],'0px');assert.ok(neutral(refusal.color));
   assert.equal(await page.locator('.o03-part table').count(),0);
   if(name==='denied')await page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});
   mode='rows';await page.locator('.o03-docs-heading button').click();await page.locator('.o03-part tbody tr').nth(6).waitFor();
   assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-part-title');
   evidence.cases.push({check:'refusal',lang,width,mode:name});
  }
 }
 assert.deepEqual(evidence.errors,[]);
}finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
console.log(`cases ${evidence.cases.length} reads ${evidence.reads} unexpected ${JSON.stringify([...new Set(evidence.unexpected)])}`);
