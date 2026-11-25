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
 const member=await login('member@example.invalid');
 const recipient=await member.evaluate(async()=>{const {serviceClient}=await import('/src/service/api.ts');return (await serviceClient().current()).user;});
 const person=await second.evaluate(async()=>{const {serviceClient}=await import('/src/service/api.ts');return (await serviceClient().current()).user;});
 const times={capability:'prepare',not_before:new Date().toISOString(),not_after:new Date(Date.now()+3600000).toISOString()};
 for(let i=0;i<2;i++)await command(second,'document_authority.propose_console','ui-pending-'+i,times);
 const staged=await command(root,'bulk_revocation.add_targets','bulk-revocations',{users:person});
 const prepared=await command(root,'bulk_revocation.prepare',staged.fields.draft,{effect:'suspend_membership'});
 const job=prepared.fields.bulk;assert.ok(job);
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await root.setViewportSize({width,height:1000});await root.goto('about:blank');
  await root.goto(`${url}/?service=api&lang=${lang}#/users?lang=${lang}&view=revocation`);
  await root.locator(`[data-job="${job}"]`).click();
  const label={en:'Document evidence and ERP preparation permission proposals',ko:'문서 근거·ERP 준비 권한 제안',ar:'مقترحات صلاحيات أدلة المستند وإعداد ERP'}[lang];
  const row=root.locator('.m01-impact .m01-sum > div').filter({has:root.getByText(label,{exact:true})});
  await row.waitFor();await root.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
  assert.equal(await row.locator('dd .m01-line').first().innerText(),'2');
  assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  assert.equal(await row.evaluate(e=>getComputedStyle(e).direction),lang==='ar'?'rtl':'ltr');
  await row.screenshot({path:`${out}/m01-row-${lang}-${width}.png`});
  await root.screenshot({path:`${out}/m01-context-${lang}-${width}.png`});
  evidence.cases.push({screen:'M01',lang,width,count:2,overflow:false});
 }
 const labels={
 propose:['Propose document evidence','문서 근거 제안','اقتراح أدلة المستند'],
 confirm:['Confirm document evidence','문서 근거 확인','تأكيد أدلة المستند'],
 revoke:['Revoke document evidence','문서 근거 철회','إلغاء أدلة المستند'],
 prepare:['Prepare internal ERP record','내부 ERP 기록 준비','إعداد سجل ERP داخلي'],
 inspect:['View cost','원가 조회','عرض التكلفة']};
 const grants=[];
 for(const capability of Object.keys(labels)){
  const fields={...times,capability};
  const cp=await command(root,'document_authority.propose_console','console-'+capability,fields);
  const ceiling=await command(second,'document_authority.approve','console-'+capability,{digest:cp.fields.digest});
  const hp=await command(root,'document_authority.propose_human','human-'+capability,{...fields,recipient,ceiling:ceiling.fields.grant});
  const granted=await command(second,'document_authority.approve','human-'+capability,{digest:hp.fields.digest});
  assert.equal(granted.fields.state,'applied');assert.equal(granted.fields.receipt,'historical_decision');
  grants.push({capability,human:granted.fields.grant,console:ceiling.fields.grant});
 }
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390])for(const route of ['access','integrations']){
  await root.setViewportSize({width,height:1000});await root.goto('about:blank');
  await root.goto(`${url}/?service=api&lang=${lang}#/${route}?lang=${lang}&view=${route==='access'?'grants':'clients'}`);
  for(const grant of grants){
   const row=route==='access'?root.locator('tr').filter({has:root.locator(`[data-grant="${grant.human}"]`)}):root.locator('.p02-grants tr').filter({hasText:grant.console});
   await row.waitFor();await root.evaluate(()=>document.fonts.ready);await row.scrollIntoViewIfNeeded();
   const text=await row.innerText();assert.ok(text.includes(labels[grant.capability][['en','ko','ar'].indexOf(lang)]),text);
   assert.ok(text.includes({en:'Synthetic document evidence',ko:'합성 문서 근거',ar:'أدلة مستند اصطناعية'}[lang]),text);
   assert.ok(!text.includes('synthetic_document_evidence'));assert.equal(await root.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
   const border=await row.evaluate(e=>{const s=getComputedStyle(e);return {color:s.borderLeftColor,width:s.borderLeftWidth,direction:s.direction};});
   assert.equal(border.direction,lang==='ar'?'rtl':'ltr');
   await row.screenshot({path:`${out}/${route}-${grant.capability}-${lang}-${width}.png`});
   if(grant.capability==='prepare') {
    await root.screenshot({path:`${out}/${route}-context-${lang}-${width}.png`});
    if(route==='access') {
     const layout=await root.locator('.live-users[data-live="grants"]').evaluate(e=>{
      const r=e.getBoundingClientRect(), p=e.parentElement.getBoundingClientRect(), s=getComputedStyle(e);
      const aside=e.querySelector(':scope > aside'), list=e.querySelector(':scope > .live-list');
      return {width:r.width,parentWidth:p.width,maxWidth:s.maxWidth,columns:s.gridTemplateColumns,asideWidth:aside?.getBoundingClientRect().width||0,listWidth:list.getBoundingClientRect().width};
     });
     assert.equal(layout.maxWidth,'none');assert.ok(layout.width>=layout.parentWidth-2);
     if(width===1440)assert.ok(layout.asideWidth>0&&layout.listWidth+layout.asideWidth<layout.width);
     evidence.cases.push({screen:'M03-layout',lang,viewportWidth:width,...layout});
     if(width===1440) { await root.locator('.live-users[data-live="grants"]').scrollIntoViewIfNeeded();await root.screenshot({path:`${out}/access-top-${lang}-${width}.png`}); }
    }
   }
   evidence.cases.push({screen:route==='access'?'M03':'P02',capability:grant.capability,lang,width,localized:true,overflow:false,border});
  }
 }
 evidence.provisioning={capabilities:Object.keys(labels),distinctGovernors:true,externalEffects:false};
 assert.deepEqual(evidence.errors,[]);
} finally {
 await writeFile(`${out}/evidence.json`,JSON.stringify(evidence,null,2));await browser.close();
}
