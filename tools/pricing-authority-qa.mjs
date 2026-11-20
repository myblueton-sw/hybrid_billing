import {createRequire} from 'node:module';
import {readFile,mkdir,writeFile} from 'node:fs/promises';
import {randomBytes} from 'node:crypto';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']||'playwright');
const {url,api,output:out,activation}=args;
assert.ok(url&&api&&out&&activation&&args.chromium);
await mkdir(out,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const evidence={cases:[],errors:[]};
async function login(email){
 const page=await browser.newPage({viewport:{width:1440,height:1000}});
 page.on('pageerror',e=>evidence.errors.push(e.message));
 await page.route('**/api/**',async route=>{const u=new URL(route.request().url());const response=await route.fetch({url:api+u.pathname+u.search});await route.fulfill({response});});
 const lines=(await readFile(activation,'utf8')).trim().split('\n');
 const link=new URL(lines.find(l=>l.startsWith(email+' ')).split(' ')[1]);
 const password=randomBytes(24).toString('base64url');
 await page.goto(url+'/activate?lang=en'+link.hash);
 await page.locator('#link-new').fill(password);await page.locator('#link-confirm').fill(password);await page.locator('form button[type=submit]').click();await page.locator('a.service-button-link').waitFor();
 await page.goto(url+'/?service=api&lang=en');await page.locator('#signin-workspace').fill('sample-workspace');await page.locator('#signin-email').fill(email);await page.locator('#signin-password').fill(password);await page.locator('form button[type=submit]').click();await page.locator('.service-session').waitFor();
 return page;
}
async function command(page,id,target,values,revision){return page.evaluate(async ({id,target,values,revision})=>{
 const {serviceClient}=await import('/src/service/api.ts');const {adminClient,attempt}=await import('/src/service/admin.ts');const session=await serviceClient().current();const client=adminClient(session.csrf);const expected=revision||(await client.access()).revision;return client.command(id,target,expected,'Synthetic UI verification',values,attempt());
 },{id,target,values,revision});}
let root;
try {
 root=await login('admin@example.invalid');const second=await login('admin2@example.invalid');
 const person=await second.evaluate(async()=>{const {serviceClient}=await import('/src/service/api.ts');return (await serviceClient().current()).user;});
 const times={capability:'approve',not_before:new Date().toISOString(),not_after:new Date(Date.now()+3600000).toISOString()};
 const proposals=[];for(let i=0;i<2;i++)proposals.push(await command(second,'pricing_authority.propose_console','ui-pending-'+i,times));
 const staged=await command(root,'bulk_revocation.add_targets','bulk-revocations',{users:person});
 const prepared=await command(root,'bulk_revocation.prepare',staged.fields?.draft||staged.draft,{effect:'suspend_membership'});
 const job=prepared.fields?.bulk||prepared.bulk;
 assert.ok(job);
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await root.setViewportSize({width,height:1000});
  await root.goto("about:blank");
  await root.goto(`${url}/?service=api&lang=${lang}#/users?lang=${lang}&view=revocation`);
  await root.locator(`[data-job="${job}"]`).click();
  const label={en:'Pricing policy permission requests',ko:'가격 정책 권한 요청',ar:'طلبات صلاحية سياسة التسعير'}[lang];
  const row=root.locator('.m01-impact .m01-sum > div').filter({has:root.getByText(label,{exact:true})});
  await row.waitFor();await root.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
  assert.equal(await row.locator('dd .m01-line').first().innerText(),'2');
  assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  const direction=await row.evaluate(e=>getComputedStyle(e).direction);assert.equal(direction,lang==='ar'?'rtl':'ltr');
  await row.screenshot({path:`${out}/m01-pending-${lang}-${width}.png`});
  await root.screenshot({fullPage:true,path:`${out}/m01-${lang}-${width}.png`});
  evidence.cases.push({lang,width,count:2,direction,overflow:false});
 }
 const member=await login('member@example.invalid');
 const recipient=await member.evaluate(async()=>{const {serviceClient}=await import('/src/service/api.ts');return (await serviceClient().current()).user;});
 const ceiling=await command(root,'pricing_authority.approve','ui-pending-0',{digest:proposals[0].fields.digest});
 const human=await command(root,'pricing_authority.propose_human','ui-human',{...times,recipient,ceiling:ceiling.fields.grant});
 await command(second,'pricing_authority.approve','ui-human',{digest:human.fields.digest});
 const inspectTimes={...times,capability:'inspect'};
 const inspectProposal=await command(root,'pricing_authority.propose_console','ui-inspect-console',inspectTimes);
 const inspectCeiling=await command(second,'pricing_authority.approve','ui-inspect-console',{digest:inspectProposal.fields.digest});
 const inspectHuman=await command(root,'pricing_authority.propose_human','ui-inspect-human',{...inspectTimes,recipient,ceiling:inspectCeiling.fields.grant});
 await command(second,'pricing_authority.approve','ui-inspect-human',{digest:inspectHuman.fields.digest});
 for (const capability of ['propose','revoke','use']) {
  const fields={...times,capability};
  const proposed=await command(root,'pricing_authority.propose_console','ui-console-'+capability,fields);
  const approved=await command(second,'pricing_authority.approve','ui-console-'+capability,{digest:proposed.fields.digest});
  const requested=await command(root,'pricing_authority.propose_human','ui-human-'+capability,{...fields,recipient,ceiling:approved.fields.grant});
  await command(second,'pricing_authority.approve','ui-human-'+capability,{digest:requested.fields.digest});
 }
 const policySpec=JSON.stringify({Mode:'synthetic',Scope:{Workspace:'sample-workspace',Customer:'sample-le-north',Entity:'sample-le-seller',Contract:'sample-contract-01',BillingProfile:'sample-profile-north'},From:'2026-10-01T00:00:00Z',To:'2026-11-01T00:00:00Z',Profile:{ID:'synthetic-charge',Revision:'1',InputIntegerDigits:12,InputScale:4,ResultIntegerDigits:14,ResultScale:4,IntermediateDigits:60},Limits:{Rules:20,Tiers:20,Discounts:20,Lines:1000,Intermediate:60},Policy:{ID:'ui-pricing-policy',Version:'1',Currency:'USD',CostBasis:'billed',Quantum:'0.01',Mode:'half_even',Path:'resale',Eligible:['csp_usage'],Rules:[{ID:'ui-pass',Level:'default',Type:'pass_through'}]}});
 const policyPrepared=await member.evaluate(async spec=>{
  const {serviceClient}=await import('/src/service/api.ts');const {adminClient,attempt}=await import('/src/service/admin.ts');const session=await serviceClient().current();return adminClient(session.csrf).command('pricing_policy.prepare','ui-policy-pending','','',{spec},{request:attempt().request,key:''});
 },policySpec);
 assert.ok(policyPrepared.revision);
 const policyProposal=await command(member,'pricing_policy.propose','ui-policy-pending',{spec:policySpec,...Object.fromEntries(['contract_revision','billing_revision','target_fingerprint','preparation_digest'].map(key=>[key,policyPrepared.fields[key]]))},policyPrepared.revision);
 assert.equal(policyProposal.fields.recorded_state,'pending');
 const policyStaged=await command(root,'bulk_revocation.add_targets','bulk-revocations',{users:recipient});
 const policyImpact=await command(root,'bulk_revocation.prepare',policyStaged.fields.draft,{effect:'suspend_membership'});
 const policyJob=policyImpact.fields.bulk;assert.ok(policyJob);
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await root.setViewportSize({width,height:1000});await root.goto('about:blank');
  await root.goto(`${url}/?service=api&lang=${lang}#/users?lang=${lang}&view=revocation`);
  await root.locator(`[data-job="${policyJob}"]`).click();
  const label={en:'Pricing policy proposals',ko:'가격 정책 제안',ar:'مقترحات سياسة التسعير'}[lang];
  const row=root.locator('.m01-impact .m01-sum > div').filter({has:root.getByText(label,{exact:true})});
  await row.waitFor();await root.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
  assert.equal(await row.locator('dd .m01-line').first().innerText(),'1');
  const permissionLabel={en:'Pricing policy permission requests',ko:'가격 정책 권한 요청',ar:'طلبات صلاحية سياسة التسعير'}[lang];
  const permissionRow=root.locator('.m01-impact .m01-sum > div').filter({has:root.getByText(permissionLabel,{exact:true})});
  assert.equal(await permissionRow.locator('dd .m01-line').first().innerText(),'0');
  assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  const direction=await row.evaluate(e=>getComputedStyle(e).direction);assert.equal(direction,lang==='ar'?'rtl':'ltr');
  await row.screenshot({path:`${out}/m01-policy-row-${lang}-${width}.png`});
  await root.screenshot({fullPage:true,path:`${out}/m01-policy-${lang}-${width}.png`});
  evidence.cases.push({route:'policy-impact',lang,width,policyCount:1,permissionRequestCount:0,direction,overflow:false});
 }
 evidence.provisioning={capabilities:['propose','approve','revoke','use','inspect'],distinctGovernors:true,policyPreparation:true,policyProposal:true,policyApproval:false};
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390])for(const route of ['access','integrations']){
  await root.setViewportSize({width,height:1000});await root.goto('about:blank');
  await root.goto(`${url}/?service=api&lang=${lang}#/${route}?lang=${lang}&view=${route==='access'?'grants':'clients'}`);
  const label={en:'Synthetic pricing policy',ko:'합성 시험용 가격 정책',ar:'سياسة تسعير اصطناعية'}[lang];
  const target=root.getByText(label,{exact:false}).first();await target.waitFor();await root.evaluate(()=>document.fonts.ready);await target.scrollIntoViewIfNeeded();
  assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  let cellChecks=null;
  if(route==='integrations') {
   const cell=root.locator('.p02-grants td.p02-first').filter({hasText:ceiling.fields.grant});
   assert.equal(await cell.count(),1);
   cellChecks=await cell.evaluate(cell=>{
    const inside=(host,rects)=>{const b=host.getBoundingClientRect();return rects.length>0&&rects.every(r=>r.width>0&&r.left>=b.left-1&&r.right<=b.right+1);};
    const range=document.createRange();range.selectNodeContents(cell);
    const grant=inside(cell,[...range.getClientRects()]);
    const badge=cell.parentElement.querySelector('.p02-tag');
    if(!badge)return {grant,badge:false,singleLine:false};
    const br=document.createRange();br.selectNodeContents(badge);const rects=[...br.getClientRects()];
    return {grant,badge:inside(badge.closest('td'),[badge.getBoundingClientRect()]),singleLine:rects.length>0&&new Set(rects.map(r=>Math.round(r.top))).size===1};
   });assert.deepEqual(cellChecks,{grant:true,badge:true,singleLine:true});
  }
  await root.screenshot({fullPage:true,path:`${out}/${route}-${lang}-${width}.png`});
  const text=await root.locator('body').innerText();
  assert.ok(!text.includes('synthetic_pricing_policy'));
  const actions={en:['Propose pricing policy','Approve pricing policy','Revoke pricing policy','Use pricing policy','Pricing policy evidence'],ko:['가격 정책 제안','가격 정책 승인','가격 정책 철회','가격 정책 적용','가격 정책 근거'],ar:['اقتراح سياسة التسعير','اعتماد سياسة التسعير','إلغاء سياسة التسعير','تطبيق سياسة التسعير','أدلة سياسة التسعير']}[lang];
  const displayed=route==='integrations'?[...actions.slice(0,4),{en:'View cost',ko:'원가 조회',ar:'عرض التكلفة'}[lang]]:actions;
  for(const action of displayed)assert.ok(text.includes(action));
  evidence.cases.push({route,lang,width,localized:true,overflow:false,cellChecks});
 }
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};if(root)await root.screenshot({fullPage:true,path:out+'/failure.png'});throw new Error('Pricing authority QA failed: '+e.name);}
finally{await writeFile(out+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
