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
 const times={policy_revision:'2',approvals:'1',validity_hours:'24',required:'input,result'};
 const proposals=[];for(let i=0;i<2;i++)proposals.push(await command(second,'financial_policy.propose','ui-pending-'+i,times));
 const staged=await command(root,'bulk_revocation.add_targets','bulk-revocations',{users:person});
 const prepared=await command(root,'bulk_revocation.prepare',staged.fields?.draft||staged.draft,{effect:'suspend_membership'});
 const job=prepared.fields?.bulk||prepared.bulk;
 assert.ok(job);
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await root.setViewportSize({width,height:1000});
  await root.goto("about:blank");
  await root.goto(`${url}/?service=api&lang=${lang}#/users?lang=${lang}&view=revocation`);
  await root.locator(`[data-job="${job}"]`).click();
  const label={en:'Charge approval policy proposals',ko:'청구 승인 정책 제안',ar:'مقترحات سياسة اعتماد الرسوم'}[lang];
  const row=root.locator('.m01-impact .m01-sum > div').filter({has:root.getByText(label,{exact:true})});
  await row.waitFor();await root.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
  assert.equal(await row.locator('dd .m01-line').first().innerText(),'2');
  assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  const direction=await row.evaluate(e=>getComputedStyle(e).direction);assert.equal(direction,lang==='ar'?'rtl':'ltr');
  await row.screenshot({path:`${out}/m01-pending-${lang}-${width}.png`});
  await root.screenshot({fullPage:true,path:`${out}/m01-${lang}-${width}.png`});
  evidence.cases.push({lang,width,count:2,direction,overflow:false});
 }

 const confirmed=await command(root,'financial_policy.confirm','ui-pending-0',{digest:proposals[0].fields.digest});
 assert.equal(confirmed.fields.recorded_state,'confirmed');
 assert.equal(confirmed.fields.confirmed_by,(await root.evaluate(async()=>{const {serviceClient}=await import('/src/service/api.ts');return (await serviceClient().current()).user;})));
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await root.setViewportSize({width,height:1000});await root.goto('about:blank');
  await root.goto(`${url}/?service=api&lang=${lang}#/approvals?lang=${lang}&view=history`);
  const row=root.locator('[data-b09-record="sample-approval-001"]:visible');await row.waitFor();
  await row.focus();await root.keyboard.press('Enter');
  await root.waitForFunction(()=>document.activeElement?.matches('[data-b09-detail-heading]'));
  await root.evaluate(()=>document.fonts.ready);
  assert.ok((await root.locator('body').innerText()).includes('1234.56789012'));
  assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  await root.screenshot({path:`${out}/b09-history-${lang}-${width}.png`});
  evidence.cases.push({screen:'B09',lang,width,historyAfterPolicyReplacement:true,exactAmount:true,detailFocus:true,overflow:false});
 }
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};if(root)await root.screenshot({path:out+'/failure.png'});throw new Error('Financial policy QA failed: '+e.name);}
finally{await writeFile(out+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
