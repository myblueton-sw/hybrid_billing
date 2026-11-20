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
 const times={capability:'confirm',not_before:new Date().toISOString(),not_after:new Date(Date.now()+3600000).toISOString()};
 const proposals=[];for(let i=0;i<1;i++)proposals.push(await command(second,'input_authority.propose_console','ui-pending-'+i,times));
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

 const baseline=await member.evaluate(async()=>{
 const {serviceClient}=await import('/src/service/api.ts');const session=await serviceClient().current();
 const response=await fetch('/api/v1/commands/input.inspect',{method:'POST',headers:{'Content-Type':'application/json','X-CSRF-Token':session.csrf},body:JSON.stringify({request_id:crypto.randomUUID(),target_id:'sample-input-review',body:{}})});
 if(!response.ok)throw new Error('Baseline read failed');return response.json();
 });
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await member.setViewportSize({width,height:1000});
  await member.goto(`${url}/?service=api&lang=${lang}#/reconciliation/sample-input-review?lang=${lang}`);
  await member.locator('.ir-status').waitFor();await member.evaluate(()=>document.fonts.ready);
  assert.equal(await member.locator('.ir-money').innerText(),'USD 15.0000');
  assert.equal(await member.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  await member.screenshot({path:`${out}/s06-${lang}-${width}.png`,fullPage:true});
  const reason=member.locator('#ir-reason');await reason.fill('Synthetic UI verification');
  await member.locator('.ir-actions button').first().click();
  await member.locator('.ir-dialog').waitFor({state:'visible'});
  assert.equal(await member.locator('.ir-dialog button').first().evaluate(e=>e===document.activeElement),true);
  await member.screenshot({path:`${out}/dialog-${lang}-${width}.png`,fullPage:true});
  await member.keyboard.press('Escape');await member.locator('.ir-dialog').waitFor({state:'hidden'});
  await reason.fill('');
  evidence.cases.push({lang,width,amount:'15.0000',overflow:false,dialogFocus:true});
 }
 await member.goto(`${url}/?service=api&lang=en#/reconciliation/sample-input-review?lang=en`);
 await member.locator('.ir-status').waitFor();
 await member.locator('#ir-reason').fill('Synthetic UI verification');
 await member.locator('.ir-actions button').first().click();
 await member.locator('.ir-dialog button').last().click();
 await member.getByText('A historical decision receipt was returned.',{exact:false}).waitFor();
 await member.getByRole('button',{name:'Read current status',exact:true}).click();
 await member.getByText('A decision is already recorded.',{exact:false}).waitFor();
 await member.screenshot({path:`${out}/state-terminal.png`,fullPage:true});
 assert.equal(await member.locator('[data-review-item="S06:input:decision"]').count(),1);
 evidence.decision={confirmed:true,terminalRead:true};
 let scenario='awaiting', calls=0, release=null, finishHeld=null;
 const heldFinished=new Promise(resolve=>{finishHeld=resolve;});
 await member.route('**/api/v1/commands/input.*',async route=>{
  const req=route.request(), body=req.postDataJSON();
  if(req.url().endsWith('input.inspect')){
   const held=scenario==='delayed';
   if(held){await new Promise(resolve=>{release=resolve;});}
   if(scenario==='denied'||scenario==='outage'||scenario==='unauthenticated')return route.fulfill({status:scenario==='denied'?403:scenario==='outage'?503:401,json:{code:scenario==='denied'?'permission_denied':scenario==='outage'?'dependency_unavailable':'unauthenticated',request_id:body.request_id}});
   const result=structuredClone(baseline);result.fields.review=body.target_id;
   if(body.target_id==='other-review')result.fields.provider='other-provider';
   if(scenario==='hold')result.fields.confirmation_hold='blocked';
   if(scenario==='view-only')result.fields.confirmation_permission='denied';
   if(scenario==='changed'){result.fields.current_evidence='changed';result.fields.current_eligibility='unverified';}
   if(scenario==='malformed')result.fields.recorded_verdict='unknown';
   await route.fulfill({status:200,json:result});if(held)finishHeld();return;
  }
  calls++;
  if(scenario==='transport')return route.abort('failed');
  if(scenario==='pending')await new Promise(resolve=>{release=resolve;});
  const code=scenario==='internal'?'internal_error':scenario==='expired'?'idempotency_key_expired':scenario==='refused'?'permission_denied':'dependency_unavailable';
  return route.fulfill({status:code==='permission_denied'?403:code==='idempotency_key_expired'?409:500,json:{code,request_id:body.request_id,...(code==='internal_error'?{operation_id:'op-synthetic-unknown'}:{})}});
 });
 const openReview=async(id='sample-input-review',lang='en')=>{
  await member.goto('about:blank');
  await member.goto(`${url}/?service=api&lang=${lang}#/reconciliation/${id}?lang=${lang}`);
 };
 for(const mode of ['hold','view-only','changed','malformed','denied','outage','unauthenticated']){
  scenario=mode;await openReview();
  if(['malformed','denied','outage','unauthenticated'].includes(mode)){
   await member.locator('.input-review [role=alert]').waitFor();assert.equal(await member.locator('.ir-money').count(),0);
  }else{
   await member.locator('.ir-status').waitFor();
   if(mode==='hold')await member.locator('#ir-reason').fill('Synthetic scenario');
   else assert.equal(await member.locator('#ir-reason').isDisabled(),true);
   assert.equal(await member.locator('.ir-actions button').first().isDisabled(),true);
   assert.equal(await member.locator('.ir-actions button').last().isDisabled(),mode!=='hold');
   if(mode==='hold')await member.locator('#ir-reason').fill('');
  }
  await member.screenshot({path:`${out}/state-${mode}.png`,fullPage:true});
  evidence.cases.push({mode,source:'injected-response',passed:true});
 }
 for(const mode of ['internal','expired','refused','transport']){
  scenario=mode;await openReview();await member.locator('.ir-status').waitFor();
  await member.locator('#ir-reason').fill('Synthetic recovery');await member.locator('.ir-actions button').first().click();
  await member.locator('.ir-dialog button').last().click();await member.locator('.ir-dialog').waitFor({state:'hidden'});await member.locator('.ir-notice').waitFor();
  assert.equal(await member.locator('.ir-actions').count(),0);
  assert.ok((await member.locator('.ir-notice').innerText()).includes(mode==='refused'?'This submission was refused':'earlier request outcome is unknown'));
  await member.screenshot({path:`${out}/state-${mode}.png`,fullPage:true});
  const before=calls;await member.getByRole('button',{name:'Read current status',exact:true}).click();await member.locator('.ir-status').waitFor();
  assert.equal(calls,before);assert.equal(await member.locator('.ir-actions button').first().isEnabled(),true);
  if(mode==='internal'){
   const references=await member.locator('.ir-notice').innerText();
   const original=references.match(/s06-[a-f0-9-]+/)[0];
   scenario='refused';await member.locator('.ir-actions button').first().focus();await member.keyboard.press('Enter');
   await member.locator('.ir-dialog').waitFor({state:'visible'});
   assert.equal(await member.locator('.ir-dialog button').first().evaluate(e=>e.matches(':focus-visible')),true);
   await member.screenshot({path:`${out}/dialog-keyboard.png`});
   await member.locator('.ir-dialog button').last().click();await member.locator('.ir-dialog').waitFor({state:'hidden'});await member.locator('.ir-notice').waitFor();
   assert.ok((await member.locator('.ir-notice').innerText()).includes(original));
   assert.ok((await member.locator('.ir-notice').innerText()).includes('earlier request outcome is unknown'));
   await member.getByRole('button',{name:'Read current status',exact:true}).click();await member.locator('.ir-status').waitFor();
  }
  await member.locator('#ir-reason').fill('');
  evidence.cases.push({mode,source:'injected-response',freshReadNoAutomaticMutation:true});
 }
 scenario='pending';await openReview();await member.locator('.ir-status').waitFor();
 await member.locator('#ir-reason').fill('Synthetic pending');await member.locator('.ir-actions button').first().click();
 const before=calls;await member.locator('.ir-dialog button').last().click();
 await member.getByText('Submitting the decision…',{exact:true}).waitFor();
 assert.equal(await member.locator('.ir-dialog button').last().isDisabled(),true);
 await member.keyboard.press('Escape');assert.equal(await member.locator('.ir-dialog').isVisible(),true);
 await member.screenshot({path:`${out}/state-pending.png`});
 assert.equal(calls,before+1);release();await member.locator('.ir-notice').waitFor();
 await member.getByRole('button',{name:'Read current status',exact:true}).click();await member.locator('.ir-status').waitFor();await member.locator('#ir-reason').fill('');
 evidence.cases.push({mode:'pending',source:'injected-response',duplicatePrevented:true,escapeBlocked:true});
 release=null;scenario='delayed';await openReview();await member.getByText('Reading input evidence…',{exact:true}).waitFor();
 while(!release)await new Promise(resolve=>setTimeout(resolve,10));
 const oldRelease=release;scenario='awaiting';
 await member.evaluate(()=>{location.hash='/reconciliation/other-review?lang=en';});
 await member.getByText('other-provider',{exact:true}).waitFor();oldRelease();await heldFinished;await member.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
 assert.equal(await member.getByText('other-provider',{exact:true}).count(),1);
 await member.locator('#ir-reason').fill('Verify new target');await member.locator('.ir-actions button').first().click();await member.locator('.ir-dialog').waitFor();
 assert.equal(await member.locator('.ir-dialog').getByText('other-review',{exact:true}).count(),1);
 await member.keyboard.press('Escape');
 evidence.cases.push({mode:'late-old-target-read',source:'injected-response',passed:true});
 await member.locator('#ir-reason').fill('가'.repeat(342));await member.locator('#ir-reason-error').waitFor();
 assert.equal(await member.locator('.ir-actions button').first().isDisabled(),true);
 evidence.cases.push({mode:'UTF8-rationale-validation',source:'injected-response',passed:true});


 for(const lang of ['ko','ar'])for(const mode of ['hold','view-only','changed','denied','outage','internal']){
  scenario=mode;await openReview('sample-input-review',lang);
  if(['denied','outage'].includes(mode))await member.locator('.input-review [role=alert]').waitFor();
  else {
   await member.locator('.ir-status').waitFor();
   if(mode==='internal'){
    await member.locator('#ir-reason').fill('Synthetic rejection review');await member.locator('.ir-actions button').last().click();
    await member.locator('.ir-dialog').waitFor();await member.screenshot({path:`${out}/dialog-reject-${lang}.png`});
    await member.locator('.ir-dialog button').last().click();await member.locator('.ir-dialog').waitFor({state:'hidden'});await member.locator('.ir-notice').waitFor();
   }
  }
  assert.equal(await member.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  await member.screenshot({path:`${out}/state-${mode}-${lang}.png`,fullPage:true});
  evidence.cases.push({lang,mode,source:'injected-response',overflow:false});
 }
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};throw new Error('Input review QA failed: '+e.name);}
finally{await writeFile(out+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
