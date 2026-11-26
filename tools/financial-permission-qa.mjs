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
 const recipientPage=await login('admin3@example.invalid');
 const recipient=await recipientPage.evaluate(async()=>{const {serviceClient}=await import('/src/service/api.ts');return (await serviceClient().current()).user;});
 const person=await second.evaluate(async()=>{const {serviceClient}=await import('/src/service/api.ts');return (await serviceClient().current()).user;});
 const times={recipient,not_before:new Date().toISOString(),not_after:new Date(Date.now()+3600000).toISOString()};
 const proposals=[];for(let i=0;i<2;i++)proposals.push(await command(second,'financial_permission.propose','ui-pending-'+i,times));
 const staged=await command(root,'bulk_revocation.add_targets','bulk-revocations',{users:person});
 const prepared=await command(root,'bulk_revocation.prepare',staged.fields?.draft||staged.draft,{effect:'suspend_membership'});
 const job=prepared.fields?.bulk||prepared.bulk;
 assert.ok(job);
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await root.setViewportSize({width,height:1000});
  await root.goto("about:blank");
  await root.goto(`${url}/?service=api&lang=${lang}#/users?lang=${lang}&view=revocation`);
  await root.locator(`[data-job="${job}"]`).click();
  const label={en:'Charge approval permission requests',ko:'청구 승인 권한 요청',ar:'طلبات صلاحية اعتماد الرسوم'}[lang];
  const row=root.locator('.m01-impact .m01-sum > div').filter({has:root.getByText(label,{exact:true})});
  await row.waitFor();await root.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
  assert.equal(await row.locator('dd .m01-line').first().innerText(),'2');
  assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  const direction=await row.evaluate(e=>getComputedStyle(e).direction);assert.equal(direction,lang==='ar'?'rtl':'ltr');
  await root.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
  await root.screenshot({path:`${out}/m01-context-${lang}-${width}.png`});
  await row.screenshot({path:`${out}/m01-pending-${lang}-${width}.png`});
  await root.screenshot({fullPage:true,path:`${out}/m01-${lang}-${width}.png`});
  evidence.cases.push({lang,width,count:2,direction,overflow:false});
 }


 const applied=await command(root,'financial_permission.approve','ui-pending-0',{digest:proposals[0].fields.digest});
 assert.equal(applied.fields.state,'applied');assert.equal(applied.fields.receipt,'historical_decision');
 const grant=applied.fields.grant;assert.ok(grant);
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await root.setViewportSize({width,height:1000});await root.goto('about:blank');
  await root.goto(`${url}/?service=api&lang=${lang}#/access?lang=${lang}&view=grants`);
  const row=root.locator('tr').filter({has:root.locator(`[data-grant="${grant}"]`)});
  await row.waitFor();await root.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
  const label={en:'Charge approval',ko:'청구 승인',ar:'اعتماد الرسوم'}[lang];
  assert.ok((await row.innerText()).includes(label));
  assert.ok(!(await row.innerText()).includes('charge_approval'));
  assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  await root.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
  await row.screenshot({path:`${out}/m03-grant-${lang}-${width}.png`});
  await root.screenshot({path:`${out}/m03-context-${lang}-${width}.png`});
  evidence.cases.push({route:'access',lang,width,localized:true,overflow:false});
 }
 const revoke=await command(second,'access.request_revoke',grant,{});
 const revoked=await command(root,'access.approve_request',revoke.fields.request,{digest:revoke.fields.digest});
 assert.ok(revoked.job_id);
 const replay=await command(root,'financial_permission.approve','ui-pending-0',{digest:proposals[0].fields.digest});
 assert.equal(replay.fields.state,'applied');assert.equal(replay.fields.receipt,'historical_decision');
 const access=await root.evaluate(async(id)=>{const {serviceClient}=await import('/src/service/api.ts');const {adminClient}=await import('/src/service/admin.ts');return adminClient((await serviceClient().current()).csrf).accessPermission('grant',id);},grant);
 const retained=[access.row];
 assert.equal(retained[0].id,grant);assert.equal(retained[0].state,'revoked');assert.equal(retained[0].effective,false);
 evidence.journey={independentApproval:true,directHumanGrant:true,m03Revocation:true,historicalReplay:true,financialDecision:false};
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};if(root)await root.screenshot({path:out+'/failure.png'});throw new Error('Financial permission QA failed: '+e.name);}
finally{await writeFile(out+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
