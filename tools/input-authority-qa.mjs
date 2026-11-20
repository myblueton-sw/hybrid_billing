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
 const times={capability:'confirm',not_before:new Date().toISOString(),not_after:new Date(Date.now()+3600000).toISOString()};
 const proposals=[];for(let i=0;i<2;i++)proposals.push(await command(second,'input_authority.propose_console','ui-pending-'+i,times));
 const staged=await command(root,'bulk_revocation.add_targets','bulk-revocations',{users:person});
 const prepared=await command(root,'bulk_revocation.prepare',staged.fields?.draft||staged.draft,{effect:'suspend_membership'});
 const job=prepared.fields?.bulk||prepared.bulk;
 assert.ok(job);
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await root.setViewportSize({width,height:1000});
  await root.goto("about:blank");
  await root.goto(`${url}/?service=api&lang=${lang}#/users?lang=${lang}&view=revocation`);
  await root.locator(`[data-job="${job}"]`).click();
  const label={en:'Input data permission requests',ko:'입력 데이터 권한 요청',ar:'طلبات صلاحية بيانات الإدخال'}[lang];
  const row=root.locator('.m01-impact .m01-sum > div').filter({has:root.getByText(label,{exact:true})});
  await row.waitFor();await root.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
  assert.equal(await row.locator('dd .m01-line').first().innerText(),'2');
  assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  const direction=await row.evaluate(e=>getComputedStyle(e).direction);assert.equal(direction,lang==='ar'?'rtl':'ltr');
  await root.screenshot({path:`${out}/m01-${lang}-${width}.png`});
  evidence.cases.push({lang,width,count:2,direction,overflow:false});
 }
 const member=await login('member@example.invalid');
 const recipient=await member.evaluate(async()=>{const {serviceClient}=await import('/src/service/api.ts');return (await serviceClient().current()).user;});
 const ceiling=await command(root,'input_authority.approve','ui-pending-0',{digest:proposals[0].fields.digest});
 const human=await command(root,'input_authority.propose_human','ui-human',{...times,recipient,ceiling:ceiling.fields.grant});
 await command(second,'input_authority.approve','ui-human',{digest:human.fields.digest});
 const inspectTimes={...times,capability:'inspect'};
 const inspectProposal=await command(root,'input_authority.propose_console','ui-inspect-console',inspectTimes);
 const inspectCeiling=await command(second,'input_authority.approve','ui-inspect-console',{digest:inspectProposal.fields.digest});
 const inspectHuman=await command(root,'input_authority.propose_human','ui-inspect-human',{...inspectTimes,recipient,ceiling:inspectCeiling.fields.grant});
 await command(second,'input_authority.approve','ui-inspect-human',{digest:inspectHuman.fields.digest});
 async function inspect(){return member.evaluate(async()=>{
  const {serviceClient}=await import('/src/service/api.ts');const session=await serviceClient().current();
  const response=await fetch('/api/v1/commands/input.inspect',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf},body:JSON.stringify({request_id:crypto.randomUUID(),target_id:'sample-input-review',body:{}})});
  if(!response.ok)throw new Error('Inspection failed');return response.json();
 });}
 const inspected=await inspect();assert.equal(inspected.fields.document_total,'15.0000');assert.equal(inspected.fields.current_eligibility,'not_eligible');
 const revision=inspected.result_revision;assert.equal(typeof revision,'string');assert.ok(revision);assert.equal(inspected.fields.confirmation_permission,'allowed');assert.equal(inspected.fields.confirmation_hold,'clear');
 const confirmed=await command(member,'input.confirm_publish','sample-input-review',{digest:inspected.fields.digest,scope_revision:inspected.fields.scope_revision},revision);
 assert.equal(confirmed.fields.recorded_state,'eligible');assert.equal(confirmed.fields.recorded_scope_revision,'1');
 const current=await inspect();assert.equal(current.fields.current_eligibility,'eligible');
 evidence.inputReview={inspected:true,documentTotal:'15.0000',confirmed:true,scopeRevision:'1',currentSyntheticEligibility:'eligible'};

 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390])for(const route of ['access','integrations']){
  await root.setViewportSize({width,height:1000});await root.goto('about:blank');
  await root.goto(`${url}/?service=api&lang=${lang}#/${route}?lang=${lang}&view=${route==='access'?'grants':'clients'}`);
  const label={en:'Synthetic input evidence',ko:'합성 입력 데이터 근거',ar:'أدلة بيانات الإدخال الاصطناعية'}[lang];
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
  await root.screenshot({path:`${out}/${route}-${lang}-${width}.png`});
  assert.ok(!(await root.locator('body').innerText()).includes('synthetic_input_evidence'));
  evidence.cases.push({route,lang,width,localized:true,overflow:false,cellChecks});
 }
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};if(root)await root.screenshot({path:out+'/failure.png'});throw new Error('Input authority QA failed: '+e.name);}
finally{await writeFile(out+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
