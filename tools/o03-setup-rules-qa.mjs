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
const empty={rules_id:'',rules_digest:'',rules_approvals:'',rules_validity_seconds:'',rules_max_lead_seconds:'',rules_channel_age_seconds:'',rules_proposer:'',rules_confirmer:'',rules_pending_confirmer:'',rules_proposed_at:'',rules_confirmed_at:''};
const owned={rules_id:'synthetic-rules',rules_digest:hash,rules_approvals:'2',rules_validity_seconds:'172800',rules_max_lead_seconds:'604800',rules_channel_age_seconds:'90000',rules_proposer:'synthetic-proposer-with-a-long-identifier',rules_confirmer:'',rules_pending_confirmer:'',rules_proposed_at:'2026-11-21T01:00:00Z',rules_confirmed_at:''};
const modes={
 absent:{rules_state:'none',rules_observation:'absent',...empty},
 unowned:{rules_state:'none',rules_observation:'unowned',...empty},
 registration_unknown:{rules_state:'registration_unknown',rules_observation:'absent',...owned,rules_proposed_at:''},
 proposed:{rules_state:'proposed',rules_observation:'pending',...owned},
 confirmation_unknown:{rules_state:'confirmation_unknown',rules_observation:'pending',...owned,rules_pending_confirmer:'synthetic-confirmer'},
 confirmation_active:{rules_state:'confirmation_unknown',rules_observation:'active',...owned,rules_confirmer:'synthetic-confirmer',rules_confirmed_at:'2026-11-21T02:00:00Z'},
 confirmed:{rules_state:'confirmed',rules_observation:'active',...owned,rules_confirmer:'synthetic-confirmer',rules_confirmed_at:'2026-11-21T02:00:00Z'},
};
const refusals={denied:[403,'permission_denied'],stale:[409,'evidence_stale'],prerequisite:[422,'prerequisite_failed'],failed:[503,'dependency_unavailable']};
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Built console with intercepted synthetic reads; no server, no mutation, no owner acceptance',cases:[],unexpected:[],errors:[],reads:0};
let mode='confirmed';
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
  if(refusals[mode])return route.fulfill({status:refusals[mode][0],json:{code:refusals[mode][1],request_id:body.request_id}});
  return done({schema:'hb_issuance_protected_setup_v1',contract:contract.id,configuration:'configuration-a',digest:hash,current_eligibility:'unchecked',channels:'0',channels_other_provenance:'0',channels_unowned_module:'0',...modes[mode],...group('recipient','customer'),...group('template','document_kind'),...group('account','entity')});
 }
 if(['issuance_setup.context','issuance_setup.options'].includes(command))return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
const open=async(lang,width)=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/contracts?view=documents&contract=${contract.id}&on=${on}&part=rules&lang=${lang}`);await page.locator('.o03-scope-bar').waitFor();};
const neutral=c=>{const m=c.match(/\d+/g).map(Number);return Math.max(m[0],m[1],m[2])-Math.min(m[0],m[1],m[2])<=16;};
const banned=/\b(ready|eligible|verified|proven|usable|mature)\b/i;
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  for(const name of Object.keys(modes)){
   mode=name;await open(lang,width);const part=page.locator('.o03-part');
   await part.locator(modes[name].rules_state==='none'?'.live-empty':'.o03-tag').waitFor();
   const seen=await part.evaluate(el=>({text:el.innerText,all:el.textContent,tags:[...el.querySelectorAll('.o03-tag')].map(t=>{const s=getComputedStyle(t);return {color:s.borderTopColor,radius:s.borderRadius,start:s.borderInlineStartWidth,top:s.borderTopWidth};}),terms:el.querySelectorAll('dt').length,groups:[...el.querySelectorAll('h4')].map(h=>h.innerText),buttons:el.querySelectorAll('button').length,overflow:document.documentElement.scrollWidth>innerWidth,mono:[...el.querySelectorAll('bdi[dir=ltr]')].length}));
   assert.equal(seen.overflow,false,`overflow ${lang} ${width} ${name}`);assert.equal(seen.buttons,1);
   if(lang==='en'){assert.ok(!banned.test(seen.text.replace('Issuance eligibility: not checked','')),`wording ${name}`);assert.ok(seen.text.includes('Issuance eligibility: not checked'));}
   if(modes[name].rules_state==='none'){assert.equal(seen.terms,0);assert.ok(!seen.all.includes('synthetic-rules'));}
   else{
    const pinned=name==='confirmation_unknown',settled=name==='confirmed',confirmedNow=settled||name==='confirmation_active',observedAt=name!=='registration_unknown';
    assert.equal(seen.terms,4+3+3+(pinned?1:0)+(observedAt?1:0)+(confirmedNow?2:0),`facts ${name} ${seen.terms}`);
    assert.deepEqual(seen.groups.length,2);if(lang==='en'){assert.equal(seen.groups[settled?0:1],confirmedNow?'Rule values':'Proposed rule values — not confirmed');if(name==='registration_unknown')assert.ok(seen.text.includes('No rules observed'));}
    assert.ok(seen.all.includes('synthetic-rules'));for(const exact of ['172800','604800','90000'])assert.ok(seen.all.includes(exact));
    if(lang==='en'){assert.ok(seen.text.includes('2 days'));assert.ok(seen.text.includes('7 days'));assert.ok(seen.text.includes('25 hours'));}
    for(const t of seen.tags){assert.ok(neutral(t.color));assert.equal(t.radius,'0px');assert.equal(t.start,t.top);}
   }
   await page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'rules',lang,width,mode:name,terms:seen.terms});
  }
  for(const name of Object.keys(refusals)){
   mode=name;await open(lang,width);const alert=page.locator('.o03-part .live-error');await alert.waitFor();
   const seen=await alert.evaluate(el=>{const s=getComputedStyle(el);return {text:el.innerText,widths:[s.borderTopWidth,s.borderBottomWidth,s.borderInlineStartWidth,s.borderInlineEndWidth],colors:[s.borderTopColor,s.borderInlineStartColor],radius:s.borderRadius,shadow:s.boxShadow,role:el.getAttribute('role'),overflow:document.documentElement.scrollWidth>innerWidth};});
   assert.ok(seen.text.includes(refusals[name][1]));assert.equal(seen.role,'alert');assert.equal(seen.overflow,false);assert.equal(seen.radius,'0px');assert.equal(seen.shadow,'none');
   assert.equal(seen.widths[2],'0px');assert.equal(seen.widths[3],'0px');assert.ok(neutral(seen.colors[0]),`colored edge ${seen.colors[0]}`);
   assert.equal(await page.locator('.o03-part dt').count(),0);
   await page.screenshot({path:`${output}/${name}-${lang}-${width}.png`,fullPage:true});
   mode='confirmed';await page.locator('.o03-part button').click();await page.locator('.o03-part .o03-tag').waitFor();
   assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-part-title');assert.equal(await page.locator('.o03-part .live-error').count(),0);
   evidence.cases.push({check:'refusal',lang,width,mode:name,...seen});
  }
 }
 assert.deepEqual(evidence.errors,[]);
}finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
console.log(`cases ${evidence.cases.length} reads ${evidence.reads} unexpected ${JSON.stringify([...new Set(evidence.unexpected)])}`);
