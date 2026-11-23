import {createRequire} from 'node:module';
import assert from 'node:assert/strict';
import {readFile,mkdir,writeFile} from 'node:fs/promises';
import {randomBytes} from 'node:crypto';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']||'playwright');
assert.ok(args.url&&args.api&&args.output&&args.activation);
await mkdir(args.output,{recursive:true});
const login=args.login||'admin@example.invalid';
const browser=await chromium.launch({headless:true,...(args.chromium?{executablePath:args.chromium}:{})});
const page=await browser.newPage({viewport:{width:1440,height:1000}}),password=randomBytes(24).toString('base64url');
const evidence={outcome:'running',cases:[],errors:[],commands:[]};
page.on('request',r=>{if(r.method()==='POST'&&new URL(r.url()).pathname.includes('/commands/'))evidence.commands.push(new URL(r.url()).pathname);});
page.on('pageerror',e=>evidence.errors.push(e.message));
await page.route('**/api/**',async route=>{const u=new URL(route.request().url());const response=await route.fetch({url:args.api+u.pathname+u.search});await route.fulfill({response});});
try {
 const lines=(await readFile(args.activation,'utf8')).trim().split('\n');
 const link=new URL(lines.find(l=>l.startsWith(login+' ')).split(' ')[1]);
 await page.goto(args.url+'/activate?lang=en'+link.hash);await page.locator('#link-new').fill(password);await page.locator('#link-confirm').fill(password);await page.locator('form button[type=submit]').click();await page.locator('a.service-button-link').waitFor();
 await page.goto(args.url+'/?service=api&lang=en');await page.locator('#signin-workspace').fill('sample-workspace');await page.locator('#signin-email').fill(login);await page.locator('#signin-password').fill(password);await page.locator('form button[type=submit]').click();await page.locator('.service-session').waitFor();
 if(args.suite==='interaction') {
 await page.goto(args.url+'/?service=api&lang=en#/access?lang=en&view=grants');
 await page.locator('#access-capability').waitFor();
 const options=await page.locator('#access-capability option').evaluateAll(es=>es.map(e=>e.value).filter(Boolean));
 assert.equal(options.length,3);
 const recipient=await page.locator('#access-person option').evaluateAll(es=>es.find(e=>e.value)?.value);
 assert.ok(recipient);await page.locator('#access-person').selectOption(recipient);
 await page.locator('#access-capability').focus();await page.keyboard.press('Tab');
 assert.ok(await page.locator('#access-days').evaluate(e=>e===document.activeElement));
 const payloads=[];let release,entered;
 await page.route('**/api/v1/commands/access.request_grant',async route=>{
  payloads.push(route.request().postDataJSON());entered();await new Promise(resolve=>release=resolve);
  await route.fulfill({status:503,contentType:'application/json',body:JSON.stringify({code:'service_unavailable'})});
 });
 for(const kind of ['focus_data','charge_candidate','approval_review']) {
  const option=options.find(x=>x.includes(kind));
  await page.locator('#access-capability').selectOption(option);
  await page.locator('#access-days').fill('2');await page.locator('#live-reason').fill('Synthetic capability request check');
  const started=new Promise(resolve=>entered=resolve);
  await page.locator('.live-step button[type=submit]').click();await started;
  assert.ok(await page.locator('#access-capability').isDisabled());
  assert.ok(await page.locator('#access-person').isDisabled());
  assert.ok(await page.locator('#access-days').isDisabled());
  assert.ok(await page.locator('.live-step button[type=submit]').isDisabled());
  const sent=payloads.at(-1).body;
  assert.equal(sent.kind,kind);assert.equal(sent.action,kind==='charge_candidate'?'view_cost':'view');
  assert.equal(sent.fields,kind==='focus_data'?'rows':kind==='charge_candidate'?'charge_evidence':'approval_evidence');assert.equal(sent.days,'2');
  release();await page.waitForFunction(()=>!document.querySelector('#access-capability').disabled);
  evidence.cases.push({kind,exactPayload:true,pendingSelectionLocked:true});
 }
 assert.equal(payloads.length,3);
 assert.equal(new Set(payloads.map(p=>p.idempotency_key)).size,3);
 evidence.commandTransport='injected 503; no backend mutation';

 } else if(args.suite==='state') {
 let mode='normal',readCount=0;
 await page.route('**/api/v1/identity/access',async route=>{
  const response=await route.fetch({url:args.api+'/api/v1/identity/access'}),body=await response.json();
  const cost=body.catalog.find(x=>x.kind==='charge_candidate'),focus=body.catalog.find(x=>x.kind==='focus_data');
  if(mode==='reverse')body.catalog=[focus,cost];
  if(mode==='short')body.catalog=[{...cost,until:new Date(Date.parse(body.now)+2*86400000).toISOString()},focus];
  if(mode==='removed'||mode==='sole')body.catalog=[focus];
  if(mode==='duplicate')body.catalog=[{...cost,until:new Date(Date.parse(body.now)+2*86400000).toISOString()},cost,focus];
  if(mode==='expired')body.catalog=[{...cost,until:body.now},focus];
  await route.fulfill({response,json:body});readCount++;
 });
 const url=args.url+'/?service=api&lang=en#/access?lang=en&view=grants';
 await page.goto(url);await page.locator('#access-capability').waitFor();
 const options=await page.locator('#access-capability option').evaluateAll(es=>es.map(e=>e.value).filter(Boolean));
 const cost=options.find(x=>x.includes('charge_candidate')),focus=options.find(x=>x.includes('focus_data'));
 async function reload(next){mode=next;const count=readCount;const response=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/identity/access');await page.getByRole('button',{name:'Reload',exact:true}).click();await response;await page.waitForFunction(()=>!document.querySelector('.live-users')?.getAttribute('aria-busy'));await page.evaluate(()=>new Promise(resolve=>requestAnimationFrame(()=>requestAnimationFrame(resolve))));assert.ok(readCount>count);}
 await page.locator('#access-capability').selectOption(cost);await page.locator('#access-days').fill('5');
 await reload('reverse');assert.equal(await page.locator('#access-capability').inputValue(),cost);
 await reload('short');await page.locator('.live-step button[type=submit]').click();await page.locator('#access-days-error').waitFor();assert.equal(await page.locator('#access-days').inputValue(),'5');
 await reload('removed');assert.equal(await page.locator('#access-capability').inputValue(),'');await page.locator('#access-capability-error').waitFor();
 await page.locator('.service-summary a[href="#access-capability"]').click();assert.ok(await page.locator('#access-capability').evaluate(e=>e===document.activeElement));
 for(const next of ['sole','duplicate','expired']) {
  mode=next;await page.goto('about:blank');await page.goto(url);await page.locator('#access-capability').waitFor();
  const count=await page.locator('#access-capability option').count();assert.equal(count,next==='duplicate'?3:2);
  assert.equal(await page.locator('#access-capability').inputValue(),next==='duplicate'?'':focus);
 }
 evidence.cases.push({reorder:true,shortenedCeilingRejected:true,removedClearsWithMessage:true,errorLinkFocus:true,soleInitial:true,duplicateConsolidated:true,expiredFirstSkipped:true});

 } else {
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]) {
  await page.setViewportSize({width,height:1000});
  for(const route of ['access','integrations']) {
   await page.goto(`${args.url}/?service=api&lang=${lang}#/${route}?lang=${lang}&view=${route==='access'?'grants':'clients'}`);
   const target=route==='access'?'[data-grant="candidate-sample-admin-2"]':'.p02-grants';
   await page.locator(target).first().waitFor();
   await page.evaluate(()=>document.fonts.ready);
   const label={en:'Charge calculation candidates',ko:'과금 계산 후보',ar:'نتائج احتساب الرسوم المبدئية'}[lang];
   assert.ok((await page.locator('body').innerText()).includes(label));
   const approvalLabel={en:'Approval review records',ko:'승인 검토 기록',ar:'سجلات مراجعة الاعتمادات'}[lang];
   assert.ok((await page.locator('body').innerText()).includes(approvalLabel));
   await page.locator(target).first().scrollIntoViewIfNeeded();
   await page.screenshot({path:`${args.output}/${route}-${lang}-${width}.png`});
   assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
   if(route==='access') {
    if(await page.locator('.access-grant-open').isVisible()) await page.locator('.access-grant-open').click();
    await page.locator('#access-capability').waitFor();
    assert.equal(await page.locator('#access-capability').inputValue(),'');
    const options=await page.locator('#access-capability option').evaluateAll(es=>es.map(e=>({value:e.value,text:e.textContent})));
    assert.equal(options.length,4);
    for(const kind of ['focus_data','charge_candidate','approval_review']) {
     const option=options.find(o=>o.value.includes(kind));assert.ok(option);
     await page.locator('#access-capability').selectOption(option.value);
     await page.locator('#access-capability').focus();
     assert.ok(await page.locator('#access-days').isVisible());
     await page.screenshot({path:`${args.output}/request-${kind}-${lang}-${width}.png`});
    }
   }
   evidence.cases.push({route,lang,width,localizedCandidate:true,overflow:false});
  }
 }

 }
 assert.deepEqual(evidence.errors,[]);assert.equal(evidence.commands.length,args.suite==='interaction'?3:0);evidence.outcome='passed';
} catch(e) {evidence.outcome='failed';evidence.failure={kind:e.name};await page.screenshot({path:args.output+'/failure.png'});throw new Error('Candidate access QA failed: '+e.name);}
finally {await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
