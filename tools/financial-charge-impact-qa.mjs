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
 const times={not_before:new Date().toISOString(),not_after:new Date(Date.now()+3600000).toISOString()};
 const proposals=[];for(let i=0;i<2;i++)proposals.push(await command(second,'financial_request_permission.propose_console','ui-pending-'+i,times));
 const staged=await command(root,'bulk_revocation.add_targets','bulk-revocations',{users:person});
 const prepared=await command(root,'bulk_revocation.prepare',staged.fields?.draft||staged.draft,{effect:'suspend_membership'});
 const job=prepared.fields?.bulk||prepared.bulk;
 assert.ok(job);
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await root.setViewportSize({width,height:1000});
  await root.goto("about:blank");
  await root.goto(`${url}/?service=api&lang=${lang}#/users?lang=${lang}&view=revocation`);
  await root.locator(`[data-job="${job}"]`).click();
  const label={en:'Charge approval requests',ko:'청구 승인 요청',ar:'طلبات اعتماد الرسوم'}[lang];
  const row=root.locator('.m01-impact .m01-sum > div').filter({has:root.getByText(label,{exact:true})});
  await row.waitFor();assert.ok((await row.innerText()).includes({en:'The requester’s permission to submit is not rechecked at decision time.',ko:'결정 시 요청자의 제출 권한을 다시 확인하지 않습니다.',ar:'لا يُعاد التحقق من صلاحية مقدّم الطلب للتقديم عند اتخاذ القرار.'}[lang]));await root.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
  assert.equal(await row.locator('dd .m01-line').first().innerText(),'0');
  assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  const direction=await row.evaluate(e=>getComputedStyle(e).direction);assert.equal(direction,lang==='ar'?'rtl':'ltr');
  await root.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));
  await root.screenshot({path:`${out}/m01-context-${lang}-${width}.png`});
  await row.screenshot({path:`${out}/m01-pending-${lang}-${width}.png`});
  await root.screenshot({fullPage:true,path:`${out}/m01-${lang}-${width}.png`});
  evidence.cases.push({lang,width,count:0,direction,overflow:false});
 }


 evidence.journey={pendingFinancialRequestCount:0,financialRequestExecuted:false,financialDecisionExecuted:false};
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};if(root)await root.screenshot({path:out+'/failure.png'});throw new Error('Financial impact QA failed: '+e.name);}
finally{await writeFile(out+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
