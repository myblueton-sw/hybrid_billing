import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);
const output=args.output;await mkdir(output,{recursive:true});
const hash='a'.repeat(64),now='2026-11-23T09:00:00Z',later='2026-11-23T21:00:00Z';
const areas={
 result:{namespace:'result_authority',prefix:'result-grant-',capabilities:['propose','confirm','revoke','apply','inspect']},
 'issuance-protected':{namespace:'issuance_protected_authority',prefix:'issuance-protected-grant-',capabilities:['register_channel','admit_channel','request_change','verify_change','approve_change','inspect']},
 'issuance-terms':{namespace:'issuance_term_authority',prefix:'issuance-term-grant-',capabilities:['record','approve','bind','approve_binding','inspect']},
 'issuance-drafts':{namespace:'issuance_draft_authority',prefix:'issuance-draft-grant-',capabilities:['submit','approve','inspect']},
};
const byNamespace=Object.fromEntries(Object.entries(areas).map(([param,a])=>[a.namespace,{param,...a}]));
const browser=await chromium.launch({headless:true,executablePath:args.chromium});
const page=await browser.newPage(),evidence={scope:'Built console with intercepted synthetic reads; no server, no mutation and no owner acceptance',cases:[],commands:{},unexpected:[],errors:[]};
let dialogs=0,accept=true,mode='rows',shown='';evidence.wrong=[];
page.on('dialog',d=>{dialogs++;return accept?d.accept():d.dismiss();});
page.on('pageerror',e=>evidence.errors.push(e.message));
await page.route('**/api/**',async route=>{
 const path=new URL(route.request().url()).pathname;
 if(path==='/api/v1/session')return route.fulfill({json:{user:'governor',workspace:'workspace',role:'admin',login:'synthetic',csrf_token:'synthetic'}});
 const command=path.split('/').at(-1),[namespace,kind]=command.split('.'),area=byNamespace[namespace],body=route.request().postDataJSON();
 if(!area||!['history','context','inspect'].includes(kind)){evidence.unexpected.push({command,shown});return route.fulfill({status:403,json:{code:'permission_denied',request_id:body?.request_id}});}
 if(area.param!==shown)evidence.wrong.push({command,shown});(evidence.commands[shown]??=new Set()).add(command);
 const longest=area.capabilities.at(-2),head={mode:'synthetic',workspace:'workspace',actor:'governor',current_eligibility:'unchecked',snapshot:hash,next:''};
 const row={proposal:`${area.param}-authority-sample`,digest:hash,kind:'human_grant',capability:longest,recipient:'member',state:'pending',proposer:'other',proposed_at:now,expires_at:'2026-11-23T10:00:00Z',decision_actor:''};
 let fields;
 if(kind==='history')fields={...head,...(mode==='empty'?{}:{row_00:JSON.stringify(row)})};
 else if(kind==='context')fields={...head,collection:body.body.collection,capability:body.body.capability,policy:JSON.stringify({capabilities:area.capabilities,proposal_lifetime_seconds:'3600',max_grant_validity_seconds:'86400'})};
 else fields={capability:longest,proposal:row.proposal,digest:hash,state:'pending',mode:'synthetic',kind:'human_grant',recipient:'member',ceiling:`${area.prefix}${hash}`,proposer:'other',not_before:now,not_after:later,proposed_at:now,expires_at:'2026-11-23T10:00:00Z',rationale:'Synthetic rationale for the rendered check',receipt:'historical_decision'};
 return route.fulfill({json:{request_id:body.request_id,status:'completed',target_id:body.target_id,result_revision:'r1',replay:false,as_of:now,fields}});
});
const open=async(lang,width,area,extra='')=>{shown=areas[area]?area:'';await page.setViewportSize({width,height:1000});await page.goto('about:blank');await page.goto(`${args.url}/?service=api&lang=${lang}#/access?view=grants${area?`&area=${area}`:''}${extra}&lang=${lang}`);await page.locator('.authority-areas a').first().waitFor();await page.evaluate(()=>document.fonts.ready);};
const navigation=()=>page.locator('.authority-areas').evaluate(nav=>{const links=[...nav.querySelectorAll('a')],style=el=>{const s=getComputedStyle(el);return {background:s.backgroundColor,weight:s.fontWeight,decoration:s.textDecorationLine,borders:[s.borderTopWidth,s.borderBottomWidth,s.borderInlineStartWidth,s.borderInlineEndWidth].join(' '),borderColors:[s.borderTopColor,s.borderBottomColor].join(' '),height:Math.round(el.getBoundingClientRect().height)};};const current=links.filter(a=>a.getAttribute('aria-current')==='page');return {count:links.length,items:nav.querySelectorAll('li').length,current:current.map(a=>a.textContent),currentStyle:current[0]&&style(current[0]),otherStyle:style(links.find(a=>!a.hasAttribute('aria-current'))),separator:getComputedStyle(nav.querySelector('ul')).borderBottomColor,heights:links.map(a=>Math.round(a.getBoundingClientRect().height)),overflow:document.documentElement.scrollWidth>innerWidth,direction:getComputedStyle(nav).direction};});
try{
 for(const lang of ['en','ko','ar'])for(const width of [1440,1280,1024,390])for(const area of [...Object.keys(areas),'']){
  await open(lang,width,area);const nav=await navigation();
  assert.equal(nav.count,5);assert.equal(nav.items,5);assert.equal(nav.current.length,1);assert.equal(nav.overflow,false);assert.equal(nav.direction,lang==='ar'?'rtl':'ltr');
  assert.ok(nav.heights.every(h=>h>=44),`targets ${nav.heights}`);assert.equal(nav.currentStyle.weight,'600');assert.equal(nav.currentStyle.decoration,'none');assert.equal(nav.currentStyle.background,'rgb(240, 241, 242)');assert.equal(nav.separator,'rgb(213, 215, 218)');
  if(width>=640)assert.equal(nav.currentStyle.borders,'0px 0px 0px 0px');
  if(area){const panel=page.locator('.result-authority');await panel.locator('.authority-list button').first().waitFor();const title=await panel.locator('h2').innerText();assert.equal(nav.current[0],title);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);}
  await page.screenshot({path:`${output}/nav-${area||'workspace'}-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'navigation',lang,width,area:area||'workspace',...nav});
 }
 for(const lang of ['en','ko','ar'])for(const width of [1440,390])for(const area of ['issuance-protected','issuance-terms','issuance-drafts']){
  if(area==='issuance-drafts'&&lang!=='en')continue;
  await open(lang,width,area);const panel=page.locator('.result-authority');
  const rowText=await panel.locator('.authority-list button').first().innerText();assert.equal(rowText.includes(areas[area].capabilities.at(-2)),false,'raw capability in the history row');assert.ok(rowText.includes(`${area}-authority-sample`));
  await panel.locator('.authority-list button').first().click();await panel.locator('.authority-facts').first().waitFor();const detail=await panel.locator('.authority-detail').innerText();assert.equal(detail.includes('result-grant-'),false);
  await page.screenshot({path:`${output}/detail-${area}-${lang}-${width}.png`,fullPage:true});
  if(width===390)await panel.locator('.authority-back').click();
  await panel.locator('.authority-actions button').nth(1).click();await panel.locator('.authority-capabilities label').first().waitFor();
  const labels=await panel.locator('.authority-capabilities label').allInnerTexts();assert.equal(labels.length,areas[area].capabilities.length);assert.equal(new Set(labels).size,labels.length);
  for(const [i,value] of areas[area].capabilities.entries())assert.notEqual(labels[i].trim(),value);
  const clipped=await panel.evaluate(el=>[...el.querySelectorAll('.authority-capabilities label')].filter(e=>e.scrollWidth>e.clientWidth+1).length);assert.equal(clipped,0);assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false);
  await page.screenshot({path:`${output}/capabilities-${area}-${lang}-${width}.png`,fullPage:true});evidence.cases.push({check:'panel',lang,width,area,labels,row:rowText});
 }
 mode='empty';await open('en',1440,'issuance-terms');await page.locator('.result-authority .authority-list').waitFor();await page.waitForFunction(()=>document.querySelector('.authority-list')?.textContent?.includes('No proposals'));assert.equal(await page.locator('.authority-list li').count(),0);await page.screenshot({path:`${output}/empty-issuance-terms-en-1440.png`,fullPage:true});evidence.cases.push({check:'empty'});mode='rows';
 for(const area of Object.keys(areas)){await open('en',1440,area,'&client=other');assert.equal(await page.locator('.authority-areas a').count(),5);assert.equal(await page.locator('.result-authority').count(),0);const alerts=await page.locator('p[role=alert]').allInnerTexts(),name={result:'result permissions','issuance-protected':'protected issuance settings permissions','issuance-terms':'payment terms permissions','issuance-drafts':'issuance draft permissions'}[area],text=alerts.find(v=>v===`This client cannot be provisioned through console ${name}.`);assert.ok(text,JSON.stringify(alerts));assert.ok(await page.evaluate(()=>{const nav=document.querySelector('.authority-areas'),alert=[...document.querySelectorAll('p[role=alert]')].find(p=>p.textContent.startsWith('This client cannot'));return !!(nav.compareDocumentPosition(alert)&Node.DOCUMENT_POSITION_FOLLOWING);}),'navigation is not above the refusal');evidence.cases.push({check:'client-refusal',area,text});}
 await page.screenshot({path:`${output}/client-refusal-en-1440.png`,fullPage:true});
 await open('en',1440,'issuance');const fallback=await navigation();assert.deepEqual(fallback.current,['Workspace access']);assert.equal(await page.locator('.result-authority').count(),0);evidence.cases.push({check:'unknown-area',current:fallback.current});
 await open('en',1440,'issuance-terms');const panel=page.locator('.result-authority');await panel.locator('.authority-list button').first().waitFor();
 await page.locator('.authority-areas a').nth(3).focus();await page.keyboard.press('Shift+Tab');await page.keyboard.press('Tab');
 const linkFocus=await page.evaluate(()=>{const s=getComputedStyle(document.activeElement);return {inNav:!!document.activeElement.closest('.authority-areas'),outline:`${s.outlineWidth} ${s.outlineStyle} ${s.outlineColor}`,offset:s.outlineOffset};});assert.ok(linkFocus.inNav);assert.notEqual(linkFocus.outline.split(' ')[0],'0px');assert.equal(linkFocus.outline.split(' ')[1],'solid');await page.screenshot({path:`${output}/nav-focus-en-1440.png`});evidence.cases.push({check:'nav-focus',...linkFocus});
 await panel.locator('.authority-actions button').nth(1).click();await panel.locator('.authority-capabilities label').first().waitFor();
 accept=false;dialogs=0;const before=page.url();await page.locator('.authority-areas a').nth(4).click();await page.waitForTimeout(300);
 assert.equal(dialogs,1,'the shell guard did not ask');assert.equal(page.url(),before);assert.equal(await panel.locator('.authority-capabilities label').count(),5);evidence.cases.push({check:'switch-guard-declined',dialogs});
 accept=true;dialogs=0;shown='issuance-drafts';await page.locator('.authority-areas a').nth(4).click();await page.waitForFunction(()=>document.querySelector('.authority-areas a[aria-current]')?.textContent==='Issuance draft permissions');await page.locator('.result-authority .authority-list button').first().waitFor();
 assert.equal(await page.locator('.result-authority .authority-capabilities').count(),0);assert.ok((await page.locator('.result-authority .authority-list button').first().innerText()).includes('issuance-drafts-authority-sample'));evidence.cases.push({check:'switch-resets-panel',dialogs});
 await page.locator('.result-authority .authority-list button').first().click();await page.getByRole('button',{name:'Approve proposal',exact:true}).click();await page.locator('#authority-reason').fill('Synthetic decision rationale');await page.locator('#authority-reason').press('Enter');await page.locator('.result-authority .authority-primary').first().waitFor();
 let primary;for(let i=0;i<60;i++){await page.keyboard.press('Tab');primary=await page.evaluate(()=>{const el=document.activeElement;if(!el?.classList?.contains('authority-primary'))return null;const s=getComputedStyle(el);return {outline:`${s.outlineWidth} ${s.outlineStyle} ${s.outlineColor}`,color:s.color,background:s.backgroundColor,visible:el.matches(':focus-visible')};});if(primary)break;}
 assert.ok(primary,'primary button not reached by keyboard');assert.ok(primary.visible);assert.equal(primary.outline.split(' ')[1],'solid');assert.notEqual(primary.outline.split(' ')[0],'0px');assert.equal(dialogs,1,'accepting the guard asked more than once');assert.notEqual(primary.outline.split(' ').slice(2).join(' '),primary.color,'focus outline equals the button text color');assert.notEqual(primary.outline.split(' ').slice(2).join(' '),'rgb(255, 255, 255)');await page.screenshot({path:`${output}/primary-focus-en-1440.png`});evidence.cases.push({check:'primary-focus',...primary});
 assert.deepEqual(evidence.wrong,[]);assert.deepEqual(evidence.unexpected.filter(u=>u.shown!==''),[]);for(const [param,area] of Object.entries(areas))assert.ok((evidence.commands[param]?.size??0)>0&&[...evidence.commands[param]].every(c=>c.startsWith(area.namespace+'.')),param);
 evidence.commands=Object.fromEntries(Object.entries(evidence.commands).map(([k,v])=>[k,[...v].sort()]));
 assert.deepEqual(evidence.errors,[]);evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.error=e.message;evidence.commands=Object.fromEntries(Object.entries(evidence.commands).map(([k,v])=>[k,[...v]]));await page.screenshot({path:`${output}/failure.png`,fullPage:true});throw e;}
finally{await writeFile(`${output}/evidence.json`,JSON.stringify(evidence,null,1));await browser.close();}
