import {createRequire} from 'node:module';
import assert from 'node:assert/strict';
import {readFile,mkdir,writeFile} from 'node:fs/promises';
import {randomBytes} from 'node:crypto';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']||'playwright');
const base=args.url;
const out=args.output,api=args.api;
assert.ok(base&&out&&api&&args.activation);
await mkdir(out,{recursive:true});
const browser=await chromium.launch({headless:true,...(args.chromium?{executablePath:args.chromium}:{})});
const page=await browser.newPage({viewport:{width:1440,height:1000}}),password=randomBytes(24).toString('base64url');
await page.route('**/api/v1/**',async route=>{const u=new URL(route.request().url());const response=await route.fetch({url:api+u.pathname+u.search});await route.fulfill({response});});
const evidence={outcome:'running',renders:[],errors:[]};page.on('pageerror',e=>evidence.errors.push(e.message));
const lines=(await readFile(args.activation,'utf8')).trim().split('\n'),link=new URL(lines.find(l=>l.startsWith((args.user||'admin@example.invalid')+' ')).split(' ')[1]);
try{
 await page.goto(base+'/activate?lang=en'+link.hash);await page.locator('#link-new').fill(password);await page.locator('#link-confirm').fill(password);await page.locator('form button[type=submit]').click();await page.locator('a.service-button-link').waitFor();
 await page.goto(base+'/?service=api&lang=en');await page.locator('#signin-workspace').fill('sample-workspace');await page.locator('#signin-email').fill(args.user||'admin@example.invalid');await page.locator('#signin-password').fill(password);await page.locator('form button[type=submit]').click();await page.locator('.service-session').waitFor();

 const screens=JSON.parse(await readFile(new URL('../src/modules/administration/screens.json',import.meta.url),'utf8'));
 const commands=[];page.on('request',r=>{if(r.method()==='POST'&&new URL(r.url()).pathname.includes('/commands/'))commands.push(new URL(r.url()).pathname);});
 if(!args.scope)for(const id of ['P01','P02','P03','P04','P05']){
  const screen=screens.find(s=>s.id===id),prefix=id.toLowerCase(),status='.'+prefix+'-read-status',reload='.'+prefix+'-reload';
  await page.goto('about:blank');
  const initial=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1'+screen.path&&r.status()===200);
  await page.goto(`${base}/?service=api&lang=en#${screen.path}?lang=en&view=${screen.tabs[0].key}`);await(await initial).finished();
  await page.waitForFunction(p=>!document.querySelector('.'+p+'-live[aria-busy=true]'),prefix);
  assert.equal(await page.locator(status).innerText(),'');
  const routeMatch=u=>u.pathname==='/api/v1'+screen.path,held=[];
  await page.route(routeMatch,async route=>{let release,finished;const gate=new Promise(resolve=>release=resolve),done=new Promise(resolve=>finished=resolve);held.push({release,done});const result=await gate;if(result===200){const u=new URL(route.request().url());const response=await route.fetch({url:api+u.pathname+u.search});await route.fulfill({response});}else await route.fulfill({status:result,contentType:'application/json',body:JSON.stringify({code:result===401?'unauthenticated':result===403?'permission_denied':'service_unavailable'})});finished();});
  async function waitHeld(n){const until=Date.now()+10000;while(held.length<n){assert.ok(Date.now()<until);await page.evaluate(()=>new Promise(done=>requestAnimationFrame(done)));}}
  async function pressReload(n){for(let i=0;i<100&&!await page.locator(reload).evaluate(e=>e===document.activeElement);i++)await page.keyboard.press('Tab');assert.ok(await page.locator(reload).evaluate(e=>e===document.activeElement));await page.keyboard.press('Enter');await waitHeld(n);}
  await page.locator(status).evaluate(e=>{window.__readChanges=[];new MutationObserver(()=>window.__readChanges.push(e.textContent)).observe(e,{childList:true,subtree:true,characterData:true});});
  await pressReload(1);await pressReload(2);held[1].release(200);await held[1].done;
  await page.waitForFunction(sel=>document.querySelector(sel)?.textContent==='Data reloaded.',status);
  const before=await page.evaluate(()=>window.__readChanges.length);held[0].release(503);await held[0].done;await page.evaluate(()=>new Promise(done=>setTimeout(done,150)));
  assert.equal(await page.evaluate(()=>window.__readChanges.length),before);assert.equal(await page.locator('.'+prefix+'-failure').count(),0);
  await pressReload(3);await pressReload(4);held[3].release(503);await held[3].done;await page.locator('.'+prefix+'-failure').waitFor();
  const beforeStale=await page.evaluate(()=>window.__readChanges.length);held[2].release(200);await held[2].done;await page.evaluate(()=>new Promise(done=>setTimeout(done,150)));
  assert.equal(await page.evaluate(()=>window.__readChanges.length),beforeStale);assert.equal(await page.locator(status).innerText(),'');assert.equal(await page.locator('.'+prefix+'-failure').count(),1);
  await pressReload(5);held[4].release(403);await held[4].done;
  await page.locator(id==='P05'?'#p05-revoked':'#'+prefix+'-denied').waitFor();assert.equal(await page.locator(status).count(),0);
  await page.unroute(routeMatch);
  for(const code of [403,401]){
   await page.goto('about:blank');await page.route(routeMatch,route=>route.fulfill({status:code,contentType:'application/json',body:JSON.stringify({code:code===403?'permission_denied':'unauthenticated'})}));
   await page.goto(`${base}/?service=api&lang=en#${screen.path}?lang=en&view=${screen.tabs[0].key}`);
   if(code===403)await page.locator('#'+prefix+'-denied').waitFor();else await page.locator('#signin-workspace').waitFor();
   assert.equal(await page.locator(status).count(),0);await page.unroute(routeMatch);
  }
  evidence.renders.push({id,backgroundSilent:true,latestSuccessWinsStaleFailure:true,latestFailureWinsStaleSuccess:true,deniedNoSuccessStatus:true,initialDeniedNoSuccessStatus:true,unauthenticatedNoSuccessStatus:true,p05RevokedVsDenied:id==='P05'});
 }

 if(args.scope){
  const other=await browser.newPage();
  await other.route('**/api/v1/**',async route=>{const u=new URL(route.request().url());const response=await route.fetch({url:api+u.pathname+u.search});await route.fulfill({response});});
  const otherLink=new URL(lines.find(l=>l.startsWith('admin2@example.invalid ')).split(' ')[1]);
  await other.goto(base+'/activate?lang=en'+otherLink.hash);await other.locator('#link-new').fill(password);await other.locator('#link-confirm').fill(password);await other.locator('form button[type=submit]').click();await other.locator('a.service-button-link').waitFor();await other.close();
  let second=false;
  for(const id of ['P01','P02','P03','P04','P05']){
   const screen=screens.find(s=>s.id===id),prefix=id.toLowerCase(),url=`${base}/?service=api&lang=en#${screen.path}?lang=en&view=${screen.tabs[0].key}`,routeMatch=u=>u.pathname==='/api/v1'+screen.path;
   await page.goto('about:blank');const initial=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1'+screen.path&&r.status()===200);await page.goto(url);await(await initial).finished();await page.waitForFunction(p=>!document.querySelector('.'+p+'-live[aria-busy=true]'),prefix);
   let captured=false,release,entered,finished;const hold=new Promise(resolve=>release=resolve),started=new Promise(resolve=>entered=resolve),done=new Promise(resolve=>finished=resolve);
   await page.route(routeMatch,async route=>{if(captured){await route.fallback();return;}captured=true;const u=new URL(route.request().url()),response=await route.fetch({url:api+u.pathname+u.search});assert.equal(response.status(),200);entered();await hold;await route.fulfill({response});finished();});
   await page.locator('.'+prefix+'-reload').click();await started;
   await page.locator('.service-signout').click();await page.locator('#signin-email').waitFor();second=!second;
   await page.locator('#signin-workspace').fill('sample-workspace');await page.locator('#signin-email').fill(second?'admin2@example.invalid':'admin@example.invalid');await page.locator('#signin-password').fill(password);await page.locator('form button[type=submit]').click();await page.locator('.service-session').waitFor();
   const next=page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1'+screen.path&&r.status()===200);await page.evaluate(hash=>location.hash=hash,screen.path+'?lang=en&view='+screen.tabs[0].key);await(await next).finished();await page.waitForFunction(p=>!document.querySelector('.'+p+'-live[aria-busy=true]'),prefix);
   const status='.'+prefix+'-read-status';assert.equal(await page.locator(status).innerText(),'');const facts=await page.locator('.'+prefix+'-scope-facts').innerText();assert.ok(facts.length>0);
   release();await done;await page.evaluate(()=>new Promise(resolve=>setTimeout(resolve,150)));
   assert.equal(await page.locator(status).innerText(),'');assert.equal(await page.locator('.service-session-user bdi').innerText(),second?'sample-admin-2':'sample-admin');assert.equal(await page.locator('.'+prefix+'-scope-facts').innerText(),facts);
   evidence.renders.push({id,scopeReplacement:true,oldReadCompletionSuppressed:true,newSessionScopeFactsUnchanged:true,newSessionCompletionSilent:true});await page.unroute(routeMatch);
  }
 }
 assert.deepEqual(commands,[]);assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};await page.screenshot({path:out+'/failure.png'});throw e;}
finally{await writeFile(out+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
