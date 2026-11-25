import {createRequire} from 'node:module';
import {readFile,mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const wire=JSON.parse(await readFile(args.fixture,'utf8'));
const output=args.output;await mkdir(output,{recursive:true});
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Actual app with intercepted synthetic contexts and mutation receipts, no live server provisioning claim',cases:[],unexpected:[],errors:[]};
let discard=false,last,submitted;
page.on('dialog',d=>discard?d.accept():d.dismiss());page.on('pageerror',e=>evidence.errors.push(e.message));
await page.route('**/api/**',async route=>{
 const path=new URL(route.request().url()).pathname;
 if(path==='/api/v1/session')return route.fulfill({json:{user:wire.history.fields.actor,workspace:wire.history.fields.workspace,role:'admin',login:'synthetic',csrf_token:'synthetic'}});
 const command=path.split('/').at(-1),body=route.request().postDataJSON();let value;
 if(command==='result_authority.history')value=wire.history;
 else if(command==='result_authority.inspect')value=last?.fields.proposal===body.target_id?last:wire.inspection;
 else if(command==='result_authority.context'){
  value=structuredClone(wire[body.body.collection]);value.fields.capability=body.body.capability;
  if(body.body.capability!=='apply')for(const key of Object.keys(value.fields))if(key.startsWith('row_'))delete value.fields[key];
 }else if(command==='result_authority.propose_console'||command==='result_authority.propose_human'){
  submitted=body;
  const fields={...wire.inspection.fields,proposal:body.target_id,kind:command.endsWith('console')?'console_ceiling':'human_grant',recipient:body.body.recipient??'console',ceiling:body.body.ceiling??'',capability:body.body.capability,not_before:body.body.not_before,not_after:body.body.not_after,rationale:body.rationale};
  last={...wire.inspection,result_revision:'r999',fields};
  return route.fulfill({json:{...last,request_id:body.request_id,target_id:body.target_id,operation_id:'synthetic-operation',business_state:'pending',replay:false}});
 }else{evidence.unexpected.push(path);return route.fulfill({status:403,json:{code:'permission_denied'}});}
 return route.fulfill({json:{...value,request_id:body.request_id,target_id:body.target_id}});
});
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1024,390])for(const kind of ['console','human']){
  await page.setViewportSize({width,height:1000});discard=true;await page.goto('about:blank');discard=false;await page.goto(`${args.url}/?service=api&lang=${lang}#/access?view=grants&area=result&lang=${lang}`);
  const panel=page.locator('.result-authority');await panel.locator('.authority-actions').first().locator('button').nth(1).click();
  await panel.locator('[name=authority-kind]').nth(kind==='console'?0:1).check();await panel.locator('[name=authority-capability]').nth(3).check();
  const form=panel.locator('.authority-proposal-form');await form.waitFor();
  await form.locator('button[type=submit]').click();await form.locator('[role=alert]').waitFor();await page.waitForFunction(()=>document.activeElement?.getAttribute('role')==='alert');assert.equal(await panel.locator('.authority-proposal-review').count(),0);await page.screenshot({path:`${output}/validation-${kind}-${lang}-${width}.png`,fullPage:true});
  await form.locator('[role=alert] a').first().click();assert.ok(await page.evaluate(()=>document.activeElement?.id?.startsWith('authority-')));
  if(kind==='human'){await form.locator('[name=authority-recipient]').first().check();await form.locator('[name=authority-ceiling]').first().check();}
  await form.locator('#authority-not_before').fill(wire.inspection.fields.not_before);await form.locator('#authority-not_after').fill(wire.inspection.fields.not_after);await form.locator('#authority-rationale').fill('Synthetic reviewed permission');
  if(lang==='en'&&width===1440&&kind==='human'){
   await panel.locator('[name=authority-kind]').first().click();assert.equal(await panel.locator('[name=authority-kind]').nth(1).isChecked(),true);assert.equal(await form.locator('#authority-rationale').inputValue(),'Synthetic reviewed permission');
   await panel.locator('[name=authority-capability]').first().click();assert.equal(await panel.locator('[name=authority-capability]').nth(3).isChecked(),true);assert.equal(await form.locator('#authority-not_before').inputValue(),wire.inspection.fields.not_before);
  }
  await page.evaluate(()=>document.fonts.ready);assert.equal(await panel.locator('.authority-choice').first().evaluate(e=>getComputedStyle(e).flexDirection),'row');await page.screenshot({path:`${output}/form-${kind}-${lang}-${width}.png`,fullPage:true});
  await form.locator('button[type=submit]').click();const review=panel.locator('.authority-proposal-review');await review.waitFor();
  await panel.locator('.authority-scope-choices').waitFor({state:'hidden'});assert.equal(await panel.locator('.authority-scope-choices').isVisible(),false);const layout=await review.evaluate(el=>({overflow:document.documentElement.scrollWidth>innerWidth,clipped:[...el.querySelectorAll('button,a')].filter(e=>e.getClientRects().length&&(e.getBoundingClientRect().right>innerWidth+1||e.getBoundingClientRect().left< -1)).length,focused:document.activeElement===el.querySelector('h3')}));
  const send=review.locator('.authority-primary');await page.mouse.move(0,0);const rest=await send.evaluate(e=>({background:getComputedStyle(e).backgroundColor,color:getComputedStyle(e).color}));assert.deepEqual(rest,{background:'rgb(32, 42, 53)',color:'rgb(255, 255, 255)'});await send.hover();const hover=await send.evaluate(e=>({background:getComputedStyle(e).backgroundColor,color:getComputedStyle(e).color}));assert.deepEqual(hover,{background:'rgb(47, 59, 72)',color:'rgb(255, 255, 255)'});evidence.cases.push({mode:'button-colors',lang,width,rest,hover});assert.equal(layout.overflow,false);assert.equal(layout.clipped,0);assert.equal(layout.focused,true);await page.screenshot({path:`${output}/review-${kind}-${lang}-${width}.png`,fullPage:true});evidence.cases.push({lang,width,kind,...layout});
  if(lang==='ar'&&width===390&&kind==='human'){await review.locator('button').first().click();await panel.locator('#authority-detail-title').waitFor();assert.equal(await panel.locator('.authority-proposal-form').count(),0);assert.equal(submitted.body.not_before,wire.inspection.fields.not_before);assert.equal(submitted.body.ceiling,wire.inspection.fields.ceiling);evidence.submitted=true;}
 }
 assert.deepEqual(evidence.errors,[]);assert.deepEqual(evidence.unexpected,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.error=e.message;await page.screenshot({path:`${output}/failure.png`,fullPage:true});throw e;}
finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,2));await browser.close();}
