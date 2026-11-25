import {createRequire} from 'node:module';
import assert from 'node:assert/strict';
import {readFile,mkdir,writeFile} from 'node:fs/promises';
import {randomBytes} from 'node:crypto';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
function verifyRender(r){
  assert.equal(r.header.length,3,r.file+' expected session elements');
  if(r.id==='P07')assert.equal(r.cards.length,2,r.file+' expected aside cards');
  if(r.id!=='M01')assert.ok(r.headings.length>=2&&r.headings.every(h=>h.text.length>0),r.file+' expected panel headings');
  if(r.id==='P06'||(r.id==='P07'&&r.view==='offline'))assert.ok(r.formColumns,r.file+' expected form grid');
  assert.equal(r.overflow,false,r.file+' document overflow');
  assert.ok(r.header.every(item=>item.left>=-1&&item.right<=r.width+1),r.file+' session bounds');
  assert.ok(r.headings.every(h=>h.text.every(r=>r.left>=h.box.left-1&&r.right<=h.box.right+1)),r.file+' heading bounds');
  if(r.width/r.zoom<=320&&r.formColumns)assert.equal(r.formColumns.trim().split(/\s+/).length,1,r.file+' narrow form');
  if(r.id==='P06'&&r.width<1200)assert.ok(r.cards.length===1&&Math.abs(r.cards[0].width-r.aside.width)<1,r.file+' full-width outcome');
  if(r.id==='P07'&&r.width/r.zoom<=640)assert.ok(r.cards.every(c=>Math.abs(c.width-r.aside.width)<1),r.file+' stacked cards');
}
if(args.evidence){const recorded=JSON.parse(await readFile(args.evidence,'utf8'));assert.deepEqual(recorded.errors,[]);assert.ok(recorded.renders.length>0);recorded.renders.forEach(verifyRender);console.log(JSON.stringify({outcome:'passed',mode:'recorded-geometry-recheck',renders:recorded.renders.length}));process.exit(0);}
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
 const catalogs=(await Promise.all(['identity','administration'].map(async n=>JSON.parse(await readFile(new URL('../src/modules/'+n+'/screens.json',import.meta.url),'utf8'))))).flat();
 for(const lang of ['en','ko','ar'])for(const id of (args.alternate?['P06','P07']:['M01','P06','P07']))for(const [width,zoom] of (args.alternate?[[320,1],[640,2],[1440,1]]:[[320,1],[640,1],[640,2],[768,1],[1024,1],[1199,1],[1440,1]])){
  await page.setViewportSize({width,height:1000});await page.goto('about:blank');const screen=catalogs.find(s=>s.id===id),view=args.alternate?(id==='P06'?'failures':'offline'):screen.tabs[0].key;
  const loaded=id==='M01'?page.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/identity/users'&&r.status()===200):null;
  await page.goto(`${base}/?service=api&lang=${lang}#${screen.path}?lang=${lang}&view=${view}`);if(loaded)await(await loaded).finished();
  await page.locator(`[data-screen-id="${id}"],[data-security-screen="${id}"]`).first().waitFor();await page.evaluate(z=>document.documentElement.style.zoom=String(z),zoom);await page.evaluate(()=>document.fonts.ready);await page.evaluate(()=>new Promise(done=>requestAnimationFrame(()=>requestAnimationFrame(done))));
  const measured=await page.evaluate(()=>{const rect=e=>{const r=e.getBoundingClientRect();return{left:r.left,right:r.right,top:r.top,bottom:r.bottom,width:r.width}};return{overflow:document.documentElement.scrollWidth>innerWidth,header:[...document.querySelectorAll('.service-session-item bdi,.service-signout')].map(e=>({tag:e.tagName,...rect(e)})),formColumns:document.querySelector('.ops-form')?getComputedStyle(document.querySelector('.ops-form')).gridTemplateColumns:null,aside:document.querySelector('.ops-grid > aside')?rect(document.querySelector('.ops-grid > aside')):null,cards:[...document.querySelectorAll('.ops-grid > aside > .ops-panel')].map(rect),headings:[...document.querySelectorAll('.ops-panel h2')].map(e=>{const range=document.createRange();range.selectNodeContents(e);return{box:rect(e),text:[...range.getClientRects()].map(r=>({left:r.left,right:r.right}))}})}});
  const file=`${id}-${view}-${lang}-${width}-z${zoom}.png`;await page.screenshot({path:out+'/'+file,fullPage:true});evidence.renders.push({id,view,lang,width,zoom,file,...measured});
  verifyRender(evidence.renders.at(-1));
  if(id==='P06'&&await page.locator('input[type="datetime-local"]').count()){await page.locator('input[type="datetime-local"]').fill('2026-11-20T12:30');await page.locator('input[type="datetime-local"]').screenshot({path:out+`/date-${lang}-${width}-z${zoom}.png`});}
 }
 assert.deepEqual(evidence.errors,[]);
 evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure={kind:e.name};throw new Error('Reflow collection failed: '+e.name);}
finally{await writeFile(out+'/evidence.json',JSON.stringify(evidence,null,2));await browser.close();}
