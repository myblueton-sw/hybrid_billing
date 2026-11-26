import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const cookie=process.env.HB_RESULT_BLOCKED_COOKIE;delete process.env.HB_RESULT_BLOCKED_COOKIE;assert.ok(cookie);
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
await mkdir(args.output,{recursive:true});const evidence={cases:[],errors:[],commands:[],refusals:[]};let activePage,stage="setup";
try{
 for(const lang of [process.env.HB_RESULT_BLOCKED_LOCALE])for(const width of [Number(process.env.HB_RESULT_BLOCKED_WIDTH)]){
  assert.ok(['en','ko','ar'].includes(lang)&&[1440,390].includes(width));
  const context=await browser.newContext({viewport:{width,height:1000}}),page=await context.newPage(),split=cookie.indexOf('=');
  activePage=page;stage=`${lang}-${width}:open`;
  await context.addCookies([{name:cookie.slice(0,split),value:cookie.slice(split+1),url:args.url,httpOnly:true,sameSite:'Strict'}]);
  page.on('pageerror',()=>evidence.errors.push('runtime_error'));
  await page.route('**/api/**',async route=>{const u=new URL(route.request().url());if(u.pathname.includes('/commands/'))evidence.commands.push(u.pathname.split('/').at(-1));const response=await route.fetch({url:args.api+u.pathname+u.search});if(response.status()>=400){const body=await response.json().catch(()=>({}));evidence.refusals.push({status:response.status(),code:typeof body.code==='string'&&/^[a-z_]{1,80}$/.test(body.code)?body.code:'unavailable'});}await route.fulfill({response});});
  await page.goto(`${args.url}/?service=api&lang=${lang}#/approvals?lang=${lang}&view=results`);
  stage=`${lang}-${width}:preparations`;await page.locator('.b09-results-list button').filter({hasText:'result-preparation'}).click();
  await page.locator('.b09-results-detail li button').filter({hasText:'blocked-successor'}).click();
  const detail=page.locator('.b09-result-inspection');await detail.waitFor();
  const apply=detail.getByRole('button',{name:{en:'Review · Apply as replacement',ko:'검토 · 대체 반영',ar:'مراجعة · التطبيق كبديل'}[lang],exact:true});
  assert.equal(await apply.isDisabled(),true);assert.equal(await apply.getAttribute('aria-describedby'),'result-predecessor-blocker');
  const blocker=detail.locator('#result-predecessor-blocker');assert.ok((await blocker.textContent()).length>50);
  await blocker.scrollIntoViewIfNeeded();await page.evaluate(()=>document.fonts.ready);
  const layout=await detail.evaluate(el=>({overflow:document.documentElement.scrollWidth>innerWidth,direction:getComputedStyle(el).direction,borders:[...el.querySelectorAll('.b09-result-position,.b09-result-blocked')].map(n=>({edge:getComputedStyle(n).borderInlineStartWidth,background:getComputedStyle(n).backgroundColor}))}));
  assert.equal(layout.overflow,false);assert.equal(layout.direction,lang==='ar'?'rtl':'ltr');
  for(const style of layout.borders){assert.equal(style.edge,'0px');assert.equal(style.background,'rgba(0, 0, 0, 0)');}
  await page.screenshot({path:`${args.output}/blocked-${lang}-${width}.png`,fullPage:true});
  await page.screenshot({path:`${args.output}/blocked-${lang}-${width}-viewport.png`});
  evidence.cases.push({lang,width,disabled:true,described:true,...layout});await context.close();
 }
 assert.deepEqual(evidence.errors,[]);assert.deepEqual(evidence.refusals,[]);assert.ok(evidence.commands.every(c=>['result_evidence.preparations','result_evidence.context','result_evidence.history','result_evidence.inspect','result_evidence.advice'].includes(c)));evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.failure=e.name;evidence.stage=stage;if(activePage)await activePage.screenshot({path:`${args.output}/failure.png`,fullPage:true});throw e;}finally{await writeFile(`${args.output}/evidence.json`,JSON.stringify(evidence,null,2));await browser.close();}
