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
const page=await browser.newPage(),evidence={scope:'Built console with intercepted synthetic reads; no server, no mutation, no owner acceptance',cases:[],unexpected:[],errors:[]};
let mode='one';
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
 if(command==='issuance_setup.context')return done({schema:'hb_issuance_setup_context_v1',contract:contract.id,contract_name:contract.name,on:body.body.on,lifecycle_sequence:'1',contract_revision:'3',billing_profile:'profile-a',billing_profile_name:'Synthetic profile',billing_revision:'2',buyer:'buyer',payer:'payer',issuer:'issuer',currency:'USD',scope_digest:hash,profile_basis:'current',current_eligibility:'unchecked'});
 if(command==='issuance_setup.options')return done({schema:'hb_issuance_setup_options_v1',contract:contract.id,on:body.body.on,scope_digest:hash,current_eligibility:'unchecked',option:'synthetic_erp_sample_v1',applicability:'synthetic_only',mode:'erp_issuance',profile:'synthetic_erp_document_v1',region:'US',tax:'synthetic-tax',zone:'UTC',languages:'ar,en,ko'});
 if(command==='issuance_setup.configurations'){const rows=histories[mode];return done({schema:'hb_issuance_configuration_page_v1',contract:contract.id,current_eligibility:'unchecked',snapshot:hash,count:String(rows.length),next:'',...Object.fromEntries(rows.map((r,i)=>[`row_${String(i).padStart(2,'0')}`,JSON.stringify(r)]))});}
 if(['issuance_setup.protected','issuance_setup.protected_channels','issuance_setup.protected_changes','issuance_setup.attestations','issuance_setup.references'].includes(command))return route.fulfill({status:403,json:{code:'permission_denied',request_id:body.request_id}});
 evidence.unexpected.push(command);return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});
});
const open=async(lang,width,part)=>{await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/contracts?view=documents&contract=${contract.id}&on=${on}${part?`&part=${part}`:''}&lang=${lang}`);await page.locator('.o03-parts').waitFor();};
const navigation=()=>page.locator('.o03-parts').evaluate(nav=>{const links=[...nav.querySelectorAll('a')],s=getComputedStyle(nav.querySelector('a[aria-current]')),list=getComputedStyle(nav.querySelector('ul'));return {count:links.length,items:nav.querySelectorAll('li').length,current:links.filter(a=>a.getAttribute('aria-current')==='page').map(a=>a.innerText),heights:links.map(a=>Math.round(a.getBoundingClientRect().height)),rows:new Set(links.map(a=>Math.round(a.getBoundingClientRect().top))).size,weight:s.fontWeight,decoration:s.textDecorationLine,background:s.backgroundColor,radius:s.borderRadius,shadow:s.boxShadow,borders:[s.borderTopWidth,s.borderBottomWidth,s.borderInlineStartWidth,s.borderInlineEndWidth].join(' '),separator:list.borderBottomColor,direction:getComputedStyle(nav).direction,overflow:document.documentElement.scrollWidth>innerWidth,label:nav.getAttribute('aria-label')};});
const neutral=c=>{const m=c.match(/\d+/g).map(Number);return Math.max(m[0],m[1],m[2])-Math.min(m[0],m[1],m[2])<=16;};
const parts=['','rules','channels','changes','statements'];
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  mode='one';
  for(const part of parts){
   await open(lang,width,part);const nav=await navigation();
   assert.equal(nav.count,5);assert.equal(nav.items,5);assert.equal(nav.current.length,1);assert.equal(nav.overflow,false,`overflow ${lang} ${width} ${part}`);assert.equal(nav.direction,lang==='ar'?'rtl':'ltr');
   assert.ok(nav.heights.every(h=>h>=44),`targets ${nav.heights}`);assert.equal(nav.weight,'600');assert.equal(nav.decoration,'none');assert.ok(neutral(nav.background));assert.equal(nav.radius,'0px');assert.equal(nav.shadow,'none');assert.ok(neutral(nav.separator));
   if(width>=1024)assert.equal(nav.borders,'0px 0px 0px 0px');
   if(part){
    await page.locator('.o03-scope-bar').waitFor();
    const bar=await page.locator('.o03-scope-bar').evaluate(el=>{const s=getComputedStyle(el);return {text:el.textContent,border:s.borderBottomColor,start:s.borderInlineStartWidth,radius:s.borderRadius,shadow:s.boxShadow};});
    assert.ok(bar.text.includes('configuration-b'));assert.ok(neutral(bar.border));assert.equal(bar.start,'0px');assert.equal(bar.radius,'0px');assert.equal(bar.shadow,'none');
    assert.equal(await page.locator('#o03-part-title').innerText(),nav.current[0]);
   }else{
    await page.locator('.o03-docs-table tbody tr').first().waitFor();
    assert.equal(await page.locator('.o03-docs-use').count(),0);assert.equal(await page.locator('.o03-docs-selected').count(),1);
    assert.equal(await page.locator('.o03-setup-form, .o03-documents h3').first().isVisible(),true);
   }
   await page.screenshot({path:`${output}/${part||'settings'}-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'part',lang,width,part:part||'settings',...nav});
  }
  mode='two';await open(lang,width,'rules');
  await page.locator('#o03-part-title').waitFor();await page.locator('.o03-part a').waitFor();assert.equal(await page.locator('.o03-scope-bar').count(),0);
  await page.screenshot({path:`${output}/no-scope-${lang}-${width}.png`,fullPage:true});
  await page.locator('.o03-part a').click();await page.locator('.o03-docs-use').first().waitFor();
  assert.equal(await page.evaluate(()=>document.activeElement?.id),'o03-docs-history');
  const cells=await page.locator('.o03-docs-table tbody tr').first().evaluate(tr=>[...tr.children].filter(td=>getComputedStyle(td).display!=='none').map(td=>{const r=td.getBoundingClientRect(),inner=Math.max(...[...td.querySelectorAll('*')].filter(e=>!e.parentElement.closest('details:not([open])')||e.tagName==='SUMMARY').map(e=>e.getBoundingClientRect().bottom),r.top);return {top:Math.round(r.top),bottom:Math.round(r.bottom),inner:Math.round(inner),display:getComputedStyle(td).display};}));
  for(const [i,c] of cells.entries()){assert.ok(c.inner<=c.bottom+1,`content leaves cell ${i} ${lang} ${width} ${JSON.stringify(c)}`);if(c.display==='block'&&i)assert.ok(c.top>=cells[i-1].bottom-1,`cells overlap ${i} ${lang} ${width}`);}
  assert.equal(new URL(page.url()).hash.includes('part='),false);assert.equal(await page.locator('.o03-docs-use').count(),2);
  const target=await page.locator('.o03-docs-use').nth(1).evaluate(b=>{const r=b.getBoundingClientRect();return {height:Math.round(r.height),label:b.getAttribute('aria-label')};});
  assert.ok(target.height>=44);assert.ok(target.label.endsWith('configuration-b'));
  await page.locator('.o03-docs-use').nth(1).focus();await page.keyboard.press('Shift+Tab');await page.keyboard.press('Tab');
  const ring=await page.evaluate(()=>{const b=document.activeElement,s=getComputedStyle(b);return {target:b.className,style:s.outlineStyle,width:s.outlineWidth};});
  assert.ok(ring.target.includes('o03-docs-use'));assert.notEqual(ring.style,'none');
  await page.screenshot({path:`${output}/choose-${lang}-${width}.png`,fullPage:true});
  await page.locator('.o03-docs-use').nth(1).click();
  await page.locator('.o03-docs-table tbody tr').nth(1).locator('.o03-docs-selected').waitFor();assert.equal(await page.locator('.o03-docs-use').count(),1);
  assert.equal(await page.evaluate(()=>document.activeElement?.className),'o03-docs-selected');
  await page.locator('.o03-parts a').nth(1).click();await page.locator('.o03-scope-bar').waitFor();
  assert.ok(new URL(page.url()).hash.includes('part=rules'));assert.ok((await page.locator('.o03-scope-bar').textContent()).includes('configuration-b'));
  await page.locator('.o03-parts a').nth(1).focus();await page.keyboard.press('Shift+Tab');await page.keyboard.press('Tab');
  const focus=await page.evaluate(()=>{const a=document.activeElement,s=getComputedStyle(a),r=a.getBoundingClientRect();return {current:a.getAttribute('aria-current'),style:s.outlineStyle,width:s.outlineWidth,inside:r.left>=2&&r.right<=innerWidth-2};});
  assert.equal(focus.current,'page');assert.notEqual(focus.style,'none');assert.ok(focus.inside);
  await page.screenshot({path:`${output}/chosen-${lang}-${width}.png`,fullPage:true});
  mode='none';await open(lang,width,'');await page.locator('.o03-docs-table tbody tr').first().waitFor();
  assert.equal(await page.locator('.o03-docs-use').count(),0);assert.equal(await page.locator('.o03-docs-selected').count(),0);
  await open(lang,width,'statements');await page.locator('.o03-part a').waitFor();assert.equal(await page.locator('.o03-scope-bar').count(),0);
  evidence.cases.push({check:'selection',lang,width,target,ring,focus});
 }
 assert.deepEqual(evidence.errors,[]);
}finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
console.log(`cases ${evidence.cases.length} unexpected ${JSON.stringify([...new Set(evidence.unexpected)])}`);
