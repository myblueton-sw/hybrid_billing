import {createRequire} from 'node:module';
import {mkdir,writeFile} from 'node:fs/promises';
import assert from 'node:assert/strict';
const args=Object.fromEntries(process.argv.slice(2).flatMap((v,i,a)=>v.startsWith('--')?[[v.slice(2),a[i+1]]]:[]));
const cookies={};
const phase=args.phase??'setup',people=phase==='setup'?['FIRST','SECOND','THIRD','MEMBER']:['FIRST','SECOND','THIRD','REQUESTER','VERIFIER'];
for(const name of people){cookies[name]=process.env['HB_ISSUANCE_SETUP_BROWSER_'+name];delete process.env['HB_ISSUANCE_SETUP_BROWSER_'+name];assert.ok(cookies[name],name);}
const {chromium}=createRequire(import.meta.url)(args['playwright-module']);const browser=await chromium.launch({headless:true,executablePath:args.chromium});
await mkdir(args.output,{recursive:true});
const evidence={mode:'actual isolated server and separate synthetic human sessions',errors:[],commands:[],refusals:[],cases:[]};let stage='setup';
async function client(who){const cookie=cookies[who],context=await browser.newContext({viewport:{width:1440,height:1000}}),page=await context.newPage(),split=cookie.indexOf('=');await context.addCookies([{name:cookie.slice(0,split),value:cookie.slice(split+1),url:args.url,httpOnly:true,sameSite:'Strict'}]);page.on('pageerror',()=>evidence.errors.push('page_runtime_error'));await page.route('**/api/**',async route=>{const u=new URL(route.request().url());if(u.pathname.includes('/commands/'))evidence.commands.push(who.toLowerCase()+' '+u.pathname.split('/').at(-1));const response=await route.fetch({url:args.api+u.pathname+u.search});if(response.status()>=400){const body=await response.json().catch(()=>({}));evidence.refusals.push({who:who.toLowerCase(),path:u.pathname,status:response.status(),code:typeof body.code==='string'&&/^[a-z_]{1,80}$/.test(body.code)?body.code:'unavailable'});}await route.fulfill({response});});return page;}
const pages={};for(const name of people)pages[name]=await client(name);
const {FIRST:first,SECOND:second,THIRD:third,MEMBER:member,REQUESTER:requester,VERIFIER:verifier}=pages;
const all=Object.values(pages);
const documents=async(p,part,ready)=>{await p.goto('about:blank');await p.goto(`${args.url}/?service=api&lang=en#/contracts?view=documents&contract=${args.contract}&on=${args.on}&part=${part}&lang=en`);await p.locator(ready).waitFor();};
const terms=async p=>{await p.goto('about:blank');await p.goto(`${args.url}/?service=api&lang=en#/contracts?view=payment-terms&contract=${args.contract}&on=${args.on}&lang=en`);await p.locator('#o03-term-steps').waitFor();};
const steps=p=>p.locator('.o03-part-sub.o03-steps'),button=(p,name)=>steps(p).getByRole('button',{name,exact:true});
const idle=p=>p.waitForFunction(()=>!document.querySelector('[aria-busy=true]'));
const send=async(p,name)=>{await button(p,'Review before sending').click();await steps(p).locator('.o03-setup-review').waitFor();await steps(p).locator('.o03-setup-review').getByRole('button',{name,exact:true}).click();};
const outcome=async(p,count)=>{await p.waitForFunction(n=>document.querySelectorAll('.o03-setup-outcome .o03-tag').length>=n,count);await idle(p);};
const shot=(p,name)=>p.screenshot({path:`${args.output}/${name}.png`,fullPage:true});
const text=p=>p.locator('main').innerText();
const done=check=>evidence.cases.push({check});
const row=p=>p.locator('.o03-changes-table tbody button');
const changes=async p=>{await documents(p,'changes','.o03-field-cards');await p.locator('.o03-field-cards > li').nth(2).locator('button').click();await p.locator('#o03-change-steps').waitFor();await idle(p);};
try{
 if(phase==='changes'){
 const requestChange=async(n,reason,outcome=0)=>{
  stage='change-request-'+n;await changes(requester);
  await button(requester,'Request protected settings change: Payment account').click();await requester.locator('#o03-change-channel').waitFor();
  await steps(requester).locator('input[name=o03-change-reference]').nth(0).check();
  const offered=await requester.locator('#o03-change-channel option').evaluateAll(list=>list.map(o=>o.value).filter(Boolean));assert.equal(offered.length,1,'verification channels offered: '+offered.length);await requester.locator('#o03-change-channel').selectOption(offered[0]);
  await steps(requester).locator('input[name=o03-change-provenance]').nth(0).check();await steps(requester).locator('input[name=o03-change-instruction]').nth(0).check();
  await requester.locator('#o03-change-effective').fill(args.effective);await requester.locator('#o03-change-reason').fill(reason);await requester.locator('#o03-request-statement').fill('Synthetic statement: I received the signed instruction in person.');
  await send(requester,'Request protected settings change');await requester.waitForFunction(count=>document.querySelectorAll('.o03-changes-table tbody tr').length>=count,n+1);await idle(requester);await shot(requester,'change-requested-'+n);
  assert.equal(await row(requester).count(),0,'the requester is offered a later step');assert.ok((await requester.locator('.o03-changes-table').innerText()).includes('You requested this change'));
  stage='change-verify-'+n;await changes(verifier);assert.equal(await row(verifier).count(),1);
  await row(verifier).click();await verifier.locator('#o03-verify-statement').waitFor();await steps(verifier).locator('input[name=o03-change-outcome]').nth(outcome).check();await verifier.locator('#o03-verify-statement').fill('Synthetic statement: I confirmed the instruction through the portal channel.');
  await send(verifier,'Verify protected settings change');await row(verifier).waitFor({state:'detached'});await idle(verifier);await shot(verifier,'change-verified-'+n);
  stage='change-decide-'+n;await changes(second);if(outcome===1){assert.equal(await row(second).count(),0,'a decision is offered on a change whose verification was denied');return;}assert.equal(await row(second).count(),1);await row(second).click();await steps(second).locator('input[name=o03-change-verdict]').first().waitFor();
 };
 await requestChange(0,'Synthetic reason: an instruction the entity later denied',1);await shot(second,'change-denied');done('a change whose verification was denied offers no decision');
 await requestChange(1,'Synthetic reason: first instruction for the bank change');
 await steps(second).locator('input[name=o03-change-verdict]').nth(1).check();await steps(second).locator('input[name=o03-change-rejection]').nth(1).check();
 await send(second,'Decide protected settings change');await row(second).waitFor({state:'detached'});await idle(second);await shot(second,'change-rejected');done('a requested and verified change rejected by a third person');
 await requestChange(2,'Synthetic reason: the issuing entity changed bank');
 await steps(second).locator('input[name=o03-change-verdict]').nth(0).check();
 await send(second,'Decide protected settings change');await row(second).waitFor({state:'detached'});await idle(second);await shot(second,'change-approved');done('a second change for the same setting requested, verified and approved');

 stage='statement-withdraw';await documents(third,'statements','#o03-statement-steps');await third.locator('.o03-statements-table tbody tr').first().waitFor();await idle(third);
 assert.equal(await third.locator('.o03-statements-table tbody button').count(),1,'withdrawal is offered on other than the one unused statement of this person');
 await third.locator('.o03-statements-table tbody button').click();await steps(third).locator('.o03-setup-review').waitFor();await steps(third).getByRole('button',{name:'Withdraw statement',exact:true}).click();
 await third.locator('.o03-statements-table tbody button').waitFor({state:'detached'});await idle(third);await shot(third,'statement-withdrawn');done('an unused statement withdrawn by its author');

 stage='term-apply';await terms(third);
 await button(third,'Apply payment terms to this contract').click();await third.locator('#o03-bind-from').fill('2026-01-01');await steps(third).locator('input[name=o03-bind-end]').nth(1).check();await steps(third).locator('input[name=o03-bind-reason]').nth(0).check();
 await send(third,'Apply payment terms to this contract');await third.locator('.o03-bindings-table tbody tr').waitFor();await idle(third);await shot(third,'term-applied');
 assert.equal(await third.locator('.o03-bindings-table tbody button').count(),0,'the person who applied is offered approval');done('payment terms applied to the contract');

 stage='term-apply-approve';await terms(first);
 await first.locator('.o03-bindings-table tbody button').click();await steps(first).locator('.o03-setup-review').waitFor();await steps(first).locator('.o03-setup-review').getByRole('button',{name:'Approve applying payment terms to this contract',exact:true}).click();
 await first.locator('.o03-bindings-table tbody button').waitFor({state:'detached'});await idle(first);await shot(first,'term-apply-approved');done('applying payment terms approved by a separate person');
 assert.deepEqual(evidence.refusals,[]);
 }else{
 stage='rules-propose';await documents(first,'rules','#o03-rule-steps');
 await button(first,'Propose protected settings rules').click();await first.locator('#o03-rule-reason').waitFor();
 await steps(first).locator('input[name=o03-rule-approvals]').nth(0).check();
 const fill=async(key,value,unit)=>{await steps(first).locator(`[aria-describedby=o03-rule-${key}-hint]`).fill(value);await steps(first).locator(`input[name=o03-rule-${key}-unit]`).nth(unit==='hours'?0:1).check();};
 await fill('validity','2','days');await fill('lead','7','days');await fill('channelAge','24','hours');await first.locator('#o03-rule-reason').fill('Synthetic control rules agreed by finance');
 await send(first,'Propose protected settings rules');await outcome(first,1);await shot(first,'rules-proposed');
 assert.ok((await text(first)).includes('You proposed these rules'));assert.equal(await steps(first).getByRole('button').count(),0,'the proposer is offered confirmation');done('rules proposed by the first governor');

 stage='rules-confirm';await documents(second,'rules','#o03-rule-steps');
 const shown=await text(second);for(const expected of ['2 days','7 days','1 day'])assert.ok(shown.includes(expected),expected);
 await button(second,'Confirm protected settings rules').click();await second.locator('#o03-rule-confirm-reason').fill('Synthetic confirmation reason');await send(second,'Confirm protected settings rules');
 await outcome(second,1);await shot(second,'rules-confirmed');assert.equal(await steps(second).getByRole('button').count(),0);done('rules confirmed by the second governor');

 stage='channel-register';await documents(third,'channels','#o03-channel-steps');
 await button(third,'Register verification channel').click();await third.locator('#o03-register-reason').waitFor();
 await steps(third).locator('input[name=o03-channel-subject]').nth(1).check();await steps(third).locator('input[name=o03-channel-kind]').nth(2).check();await steps(third).locator('input[name=o03-channel-reference]').nth(0).check();
 await third.locator('#o03-register-statement').fill('Synthetic statement: I registered the portal channel from the signed form.');await third.locator('#o03-register-locator').fill('Synthetic folder 12');await third.locator('#o03-register-reason').fill('Synthetic reason for the portal channel');
 await send(third,'Register verification channel');await third.locator('.o03-part-table tbody tr').waitFor();await idle(third);await shot(third,'channel-registered');
 assert.ok((await third.locator('.o03-part-table').innerText()).includes('You registered this channel'));done('channel registered by the third administrator with a statement');

 stage='channel-accept';await documents(first,'channels','#o03-channel-steps');
 await first.locator('.o03-part-table tbody button').click();await first.locator('#o03-accept-statement').waitFor();
 await first.locator('#o03-accept-statement').fill('Synthetic statement: I checked the channel against the signed form.');
 const acceptReason=first.locator('#o03-accept-reason');if(await acceptReason.count())await acceptReason.fill('Synthetic reason for accepting the channel');
 await send(first,'Accept verification channel');await first.locator('.o03-part-table tbody button').waitFor({state:'detached'});await idle(first);await shot(first,'channel-accepted');done('channel accepted by a separate person with a statement');

 stage='term-record';await terms(third);
 await button(third,'Record payment terms').click();await third.locator('#o03-term-days').fill('45');await third.locator('#o03-term-reason').fill('Synthetic reason agreed with the customer');
 await send(third,'Record payment terms');await outcome(third,1);await shot(third,'term-recorded');done('payment terms recorded by the third administrator');

 stage='term-approve';await terms(first);
 await button(first,'Approve payment terms').click();await steps(first).locator('.o03-setup-review').waitFor();await steps(first).locator('.o03-setup-review').getByRole('button',{name:'Approve payment terms',exact:true}).click();
 await outcome(first,1);await shot(first,'term-approved');done('payment terms approved by a separate person');

 stage='member-reach';await member.goto('about:blank');await member.goto(`${args.url}/?service=api&lang=en#/contracts?view=documents&contract=${args.contract}&on=${args.on}&part=channels&lang=en`);
 await member.waitForResponse(r=>new URL(r.url()).pathname==='/api/v1/contracts').catch(()=>{});await member.waitForTimeout(1500);await shot(member,'member-without-administration');
 assert.equal(await member.locator('#o03-channel-steps').count(),0);assert.ok(evidence.refusals.some(r=>r.who==='member'&&r.path==='/api/v1/contracts'&&r.status===403));done('a member without installation administration cannot open the contract screen');

 assert.deepEqual(evidence.refusals.filter(r=>r.who!=='member'),[]);
 }
 assert.deepEqual(evidence.errors,[]);

 evidence.writes=evidence.commands.filter(c=>!/ issuance_setup\.(protected|protected_channels|protected_changes|attestations|references|terms|context|options|history|impact)$/.test(c)&&c.includes('issuance_setup.'));
 evidence.outcome='passed';
}catch(e){evidence.outcome='failed';evidence.stage=stage;evidence.failure=String(e.message).slice(0,600);let i=0;for(const p of all)await p.screenshot({path:`${args.output}/failure-${i++}.png`,fullPage:true}).catch(()=>{});}
evidence.commands=[...new Set(evidence.commands)];
await writeFile(args.output+'/evidence.json',JSON.stringify(evidence,null,1));await browser.close();
if(evidence.outcome!=='passed'){console.error(JSON.stringify({stage:evidence.stage,failure:evidence.failure,refusals:evidence.refusals}));process.exit(1);}
