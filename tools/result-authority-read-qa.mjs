import {createRequire} from 'node:module';
import {readFile,mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const wire=JSON.parse(await readFile(args.fixture,'utf8'));
const output=args.output;await mkdir(output,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Actual app with intercepted synthetic Go-produced reads; no live server mutation claim',cases:[],unexpected:[],errors:[]};
let discard=false;page.on('dialog',d=>discard||d.type()==='beforeunload'?d.accept():d.dismiss());
let mode='normal';
page.on('pageerror',e=>evidence.errors.push(e.message));
await page.route('**/api/**',async route=>{
 const path=new URL(route.request().url()).pathname;
 if(path==='/api/v1/session')return route.fulfill({json:{user:wire.history.fields.actor,workspace:wire.history.fields.workspace,role:'admin',login:'synthetic',csrf_token:'synthetic'}});
 const command=path.split('/').at(-1),body=route.request().postDataJSON();
 let value=command==='result_authority.history'?wire.history:command==='result_authority.inspect'?wire.inspection:undefined;
 if(command==='result_authority.context'){value=structuredClone(wire.ceilings);value.fields.capability=body.body.capability;for(const k of Object.keys(value.fields))if(k.startsWith('row_'))delete value.fields[k];}
 if(!value){evidence.unexpected.push(path);return route.fulfill({status:403,json:{code:'permission_denied'}});}
 if(command==='result_authority.history'){if(mode==='denied'||mode==='unavailable')return route.fulfill({status:mode==='denied'?403:503,json:{request_id:body.request_id,code:mode==='denied'?'permission_denied':'dependency_unavailable'}});if(mode==='empty'){value=structuredClone(value);for(const k of Object.keys(value.fields))if(k.startsWith('row_'))delete value.fields[k];}if(mode==='malformed'){value=structuredClone(value);value.fields.snapshot='invalid';}}
 return route.fulfill({json:{...value,request_id:body.request_id,target_id:body.target_id}});
});
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390]){
  await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/access?view=grants&area=result&lang=${lang}`);
  const panel=page.locator('.result-authority');await panel.locator('.authority-list button').filter({hasText:'client-human'}).waitFor();await page.evaluate(()=>document.fonts.ready);await page.screenshot({path:`${output}/history-${lang}-${width}.png`,fullPage:true});await panel.locator('.authority-list button').filter({hasText:'client-human'}).click();await panel.locator('#authority-detail-title').waitFor();await page.evaluate(()=>document.fonts.ready);
  const layout=await panel.evaluate(el=>({overflow:document.documentElement.scrollWidth>innerWidth,direction:getComputedStyle(el).direction,font:getComputedStyle(el).fontFamily,clipped:[...el.querySelectorAll('button,a,input')].filter(e=>e.getClientRects().length&&(e.getBoundingClientRect().right>innerWidth+1||e.getBoundingClientRect().left< -1)).length}));
  assert.equal(layout.overflow,false);assert.equal(layout.clipped,0);assert.equal(layout.direction,lang==='ar'?'rtl':'ltr');
  assert.match(await panel.innerText(),/2026-11-16T09:00:00.5Z/);
  await page.screenshot({path:`${output}/${lang}-${width}.png`,fullPage:true});evidence.cases.push({lang,width,...layout});if(width===390){await panel.locator('.authority-back').click();await page.waitForFunction(()=>document.activeElement?.closest('.authority-list')&&document.activeElement?.textContent?.includes('client-human'));evidence.cases.push({lang,width,backFocus:true});}
 }
 await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=en#/access?view=grants&area=result&lang=en`);
 const panel=page.locator('.result-authority');await panel.locator('.authority-list button').filter({hasText:'client-human'}).click();await panel.getByRole('button',{name:'Withdraw proposal',exact:true}).click();
 await panel.locator('#authority-reason').fill('Keep this unsent rationale');await panel.getByRole('button',{name:'Start new read',exact:true}).click();assert.equal(await panel.locator('#authority-reason').inputValue(),'Keep this unsent rationale');
 await panel.getByRole('button',{name:'Back to proposals',exact:true}).click();assert.equal(await panel.locator('#authority-reason').inputValue(),'Keep this unsent rationale');
 evidence.cases.push({mode:'declined-local-navigation',draftRetained:true});
 await panel.getByRole('button',{name:'Cancel',exact:true}).click();await panel.locator('#authority-reason').waitFor({state:'detached'});
 await panel.getByRole('button',{name:'New permission proposal',exact:true}).click();await panel.locator('.authority-proposal').waitFor();discard=true;await panel.getByRole('button',{name:'Close proposal entry',exact:true}).click();await page.waitForFunction(()=>document.activeElement?.textContent==='New permission proposal');discard=false;evidence.cases.push({mode:'close-proposal',focusRestored:true});
 for(const lang of ['en','ko','ar'])for(const width of [1440,390])for(const failure of ['empty','denied','unavailable','malformed']){
  mode='normal';await page.goto('about:blank');await page.setViewportSize({width,height:1000});await page.goto(`${args.url}/?service=api&lang=${lang}#/access?view=grants&area=result&lang=${lang}`);const panel=page.locator('.result-authority');await panel.locator('.authority-list li').first().waitFor();mode=failure;await panel.locator('.authority-actions').first().locator('button').first().click();
  if(failure==='empty')await page.waitForFunction(()=>document.querySelector('.authority-list')?.textContent?.includes('No proposals')||document.querySelector('.authority-list')?.textContent?.includes('제안이 없습니다')||document.querySelector('.authority-list')?.textContent?.includes('لا توجد مقترحات'));else await panel.locator('[role=alert]').waitFor();
  assert.equal(await panel.locator('.authority-list li').count(),0);assert.equal(await panel.locator('#authority-detail-title').count(),0);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);await page.screenshot({path:`${output}/${failure}-${lang}-${width}.png`,fullPage:true});evidence.cases.push({failure,lang,width,oldRowsRemoved:true});
 }
 assert.deepEqual(evidence.errors,[]);assert.deepEqual(evidence.unexpected,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.error=e.message;await page.screenshot({path:`${output}/failure.png`,fullPage:true});throw e;}
finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,2));await browser.close();}
