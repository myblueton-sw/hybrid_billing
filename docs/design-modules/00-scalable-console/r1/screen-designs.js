import {createBusinessDesign,businessItems,customerDocumentContext} from './screen-design-business.js';
import {esc} from './widgets.js';
import {label} from './copy.js';
import {sd,localized} from './screen-designs-copy.js';
import {profileFor} from './screen-design-profiles.js';
const drafts=new Map();
const excluded=new Set(['overview','contents','core-customer-overview','assistant','index-approval','community']);
const allowedKinds=new Set(['form','detail','workflow','overview']);
const moneyKeys=new Set(['amount','net','discount','actual','expected','residual','rate','retained','confirmed','eligible','fx','tax']);
const stepNames=['target','evidence','outcome'];
const supports=screen=>allowedKinds.has(screen.kind)&&!excluded.has(screen.id)&&screen.group!=='onboarding';
const itemId=(screen,type,key)=>`${screen.id}:${type}:${key}`;
const attr=(screen,type,key)=>`data-design-id="${itemId(screen,type,key)}"`;
const fieldSample=(field,account)=>({account,customer:'CUSTOMER-DEMO-001',contract:'CONTRACT-DEMO-001',invoice:'DRAFT-DEMO-001',email:'review@example.invalid',effective:'2026-10-01',until:'2026-10-31',period:'2026-10',currency:'USD',proof:'DEMO-PROOF-REFERENCE',license:'DEMO-LICENSE-REFERENCE',question:'Please review the selected example line.',reason:'Review the synthetic example before any operation.',rule:'RULE-DEMO-001',scope:account,workspace:'DEMO-WORKSPACE',entity:'DEMO-ENTITY',destination:'DEMO-DESTINATION',version:'DEMO-r1',run:'RUN-DEMO-001',source:'SOURCE-DEMO-001',component:'Compute',resource:'RESOURCE-DEMO-001',notification:'DEMO-NOTIFICATION-POLICY',identity:'DEMO-IDENTITY-POLICY',retention:'DEMO-RETENTION-POLICY',hold:'DEMO-HOLD-REFERENCE',rounding:'DEMO-ROUNDING-POLICY',basis:'DEMO-BASIS',discount:'0.00',fx:'DEMO-RATE-SOURCE',tax:'DEMO-TAX-POLICY'}[field.key]??(field.type==='date'?'2026-10-01':field.type==='month'?'2026-10':field.type==='number'?'1':`DEMO-${field.key.toUpperCase()}`));
function safeColumns(screen){return screen.columns.filter(column=>screen.group!=='customer'||!['provider','cost','margin','supplier','revision','owner'].includes(column.key));}
export function designItems(screen,locale='en'){
 const business=businessItems(screen,locale);if(business.length)return business;
 if(!supports(screen))return [];
 const item=(type,key,title,behavior)=>({id:itemId(screen,type,key),label:title,title,behavior,expected:behavior,status:'prototype',selector:`[data-design-id="${itemId(screen,type,key)}"]`});
 const items=[item('section','context',sd('context',locale),localized(profileFor(screen).note,locale)),item('state','scenario',sd('scenario',locale),sd('boundary',locale))];
 if(screen.kind==='form'){
  for(const field of screen.fields)items.push(item('field',field.key,label(field.label,locale),sd('required',locale)));
  items.push(item('action','preview',sd('review',locale),sd('saved',locale)),item('action','fill-example',sd('fill',locale),sd('boundary',locale)),item('action','clear-draft',sd('clear',locale),sd('historyText',locale)));
 }else if(screen.kind==='workflow'){
  const profile=profileFor(screen);
  stepNames.forEach((key,i)=>items.push(item('action',`check-${key}`,localized(profile.steps[i]??[sd('checkData',locale),sd('checkEvidence',locale),sd('checkOutcome',locale)][i],locale),sd('checkpointDone',locale))));
  items.push(item('action','prepare-work',sd('draftOnly',locale),sd('noIssue',locale)));
 }else if(screen.kind==='detail'){
  for(const column of safeColumns(screen))items.push(item('data',column.key,label(column.label,locale),sd('evidenceText',locale)));
  for(const key of ['overview','evidence','history'])items.push(item('tab',key,sd(key,locale),sd(key==='history'?'historyText':'evidenceText',locale)));
  items.push(item('action','inspect',sd('refresh',locale),sd('resultText',locale)));
  if(screen.id==='core-customer-invoice')items.push(item('data','selected-line',sd('line',locale),sd('customerSafe',locale)),item('data','document-revision',sd('revision',locale),sd('customerSafe',locale)));
 }else{
  for(const id of screen.related)items.push(item('navigation',id,id,sd('next',locale)));
 }
 return items;
}
export function screenDesignIntro(screen,locale='en'){
 if(!['list','files'].includes(screen.kind)||screen.group==='onboarding')return '';
 const profile=profileFor(screen);
 return `<section class="sd-list-context" ${attr(screen,'section','purpose')}><h2>${esc(localized(profile.heading,locale))}</h2><p>${esc(localized(profile.note,locale))}</p></section>`;
}
export function createScreenDesign(screen,{locale='en',account,scope=account,navigate,isActive=()=>true}){
 const business=createBusinessDesign(screen,{locale,account,scope,navigate,isActive});if(business)return business;
 if(!supports(screen))return null;
 const key=`${scope}:${screen.id}`;
 if(!drafts.has(key))drafts.set(key,{values:{},scenario:'normal',tab:'overview',steps:new Set(),result:false,events:[],revision:1});
 const state=drafts.get(key),profile=profileFor(screen),t=name=>sd(name,locale);
 const loc=value=>esc(localized(value,locale));
 const addEvent=()=>{state.events.unshift({time:new Date().toISOString(),revision:state.revision});state.events=state.events.slice(0,10);};
 const fieldLabel=field=>esc(label(field.label,locale));
 const customerDocument=screen.id==='core-customer-invoice'?customerDocumentContext(scope):null;
 const publishedExample=localized(['Synthetic published example — no actual publication','합성 공개 문서 예시 — 실제 공개 아님','مثال اصطناعي لمستند منشور — دون نشر فعلي'],locale);
 const descriptorId=customerDocument?.document??`DEMO-${screen.id.toUpperCase()}`;
 function facts(){
  return `<dl class="facts"><div><dt>${t('scope')}</dt><dd><bdi>${esc(account)}</bdi></dd></div><div><dt>${t('reference')}</dt><dd><bdi>${descriptorId}</bdi></dd></div><div><dt>${t('revision')}</dt><dd><bdi>${esc(customerDocument?.revision??`r${state.revision}`)}</bdi></dd></div></dl>`;
 }
 function header(){
  return `<section class="panel sd-context" ${attr(screen,'section','context')}><div><p class="sd-eyebrow">${t('preview')}</p><h2>${loc(profile.heading)}</h2><p>${loc(profile.note)}</p></div>${facts()}<label class="sd-scenario">${t('scenario')}<select id="screen-design-scenario" ${attr(screen,'state','scenario')}>${['normal','empty','error','revoked'].map(value=>`<option value="${value}" ${state.scenario===value?'selected':''}>${t(value)}</option>`).join('')}</select></label></section>`;
 }
 function blocked(){
  if(state.scenario==='normal')return '';
  const title=state.scenario==='revoked'?t('blocked'):t(state.scenario);
  return `<section class="panel sd-state" role="status" ${attr(screen,'state',state.scenario)}><h2>${title}</h2><p>${t(`${state.scenario}Text`)}</p><button id="screen-design-retry" ${attr(screen,'action','retry')}>${t('retry')}</button></section>`;
 }
 function field(field){
  const name=esc(field.key),value=state.values[field.key]??'';
  let control;
  if(field.type==='select')control=`<select name="${name}" required><option value="">—</option>${(field.options??[]).map(option=>`<option value="${esc(option.value)}" ${value===option.value?'selected':''}>${esc(label(option.label,locale))}</option>`).join('')}</select>`;
  else if(['reason','question'].includes(field.key))control=`<textarea name="${name}" required maxlength="1000" rows="4">${esc(value)}</textarea>`;
  else control=`<input name="${name}" required type="${['date','month','email'].includes(field.type)?field.type:'text'}" ${field.type==='number'?'inputmode="decimal"':''} value="${esc(value)}" maxlength="200" autocomplete="off">`;
  return `<label ${attr(screen,'field',field.key)}>${fieldLabel(field)} *${control}</label>`;
 }
 function form(){
  const primary=screen.id==='entry-login'?t('signIn'):screen.id==='customer-inquiry'?t('caseDraft'):t('review');
  const identityFields=new Set(['workspace','email','account','customer','contract','invoice','identity','entity','destination','source','rule','scope','license']);
  const sections=[screen.fields.filter(field=>identityFields.has(field.key)),screen.fields.filter(field=>!identityFields.has(field.key))];
  return `<div class="sd-editor-layout"><section class="panel"><form id="screen-form"><p>${screen.group==='access'?t('authBoundary'):screen.id==='customer-inquiry'?t('caseBoundary'):t('boundary')}</p>${sections.map((fields,index)=>fields.length?`<fieldset class="sd-fieldset"><legend>${t(index===0?'target':'inputs')}</legend><div class="form-grid">${fields.map(field).join('')}</div></fieldset>`:'').join('')}<div class="form-actions"><button type="submit" class="primary" ${attr(screen,'action','preview')}>${primary}</button><button type="button" id="screen-design-fill" ${attr(screen,'action','fill-example')}>${t('fill')}</button><button type="button" id="screen-design-clear" ${attr(screen,'action','clear-draft')}>${t('clear')}</button></div><p id="form-feedback" role="status">${state.result?t('saved'):''}</p></form></section><aside class="panel sd-notes"><h2>${t('notes')}</h2><p>${loc(profile.note)}</p><dl class="facts"><div><dt>${t('sourceRevision')}</dt><dd>${t('unavailable')}</dd></div><div><dt>${t('evidenceRef')}</dt><dd>${t('unavailable')}</dd></div></dl><p class="notice">${screen.group==='customer'?t('customerSafe'):t('noIssue')}</p></aside></div><section id="screen-design-result" aria-live="polite">${state.result?result():''}</section>`;
 }
 function result(){
  return `<section class="panel sd-result" ${attr(screen,'state','draft-result')}><h2>${t('result')}</h2>${facts()}<p>${screen.id==='customer-inquiry'?t('caseBoundary'):t('resultText')}</p>${screen.kind==='form'?`<dl class="facts">${screen.fields.map(field=>`<div><dt>${fieldLabel(field)}</dt><dd><bdi>${esc(state.values[field.key])}</bdi></dd></div>`).join('')}</dl>`:''}<p class="notice">${t('noIssue')}</p></section>`;
 }
 function value(column){
  if(moneyKeys.has(column.key))return t('unavailable');
  if(customerDocument&&['invoice','version','revision','status','line'].includes(column.key))return {invoice:customerDocument.document,version:customerDocument.revision,revision:customerDocument.revision,status:publishedExample,line:customerDocument.line}[column.key];
  return {account,customer:screen.group==='customer'?account:'CUSTOMER-DEMO-001',contract:'CONTRACT-DEMO-001',invoice:'DRAFT-DEMO-001',period:'2026-10',currency:'USD',version:'DEMO-r1',status:t('pending'),evidence:t('unavailable'),source:'SOURCE-DEMO-001',component:'Compute',service:'Compute',quantity:t('unavailable'),effective:'2026-10-01',basis:t('unavailable'),approval:t('blocked'),result:t('unknown')}[column.key]??t('unavailable');
 }
 function summary(){
  const cells=safeColumns(screen).map(column=>`<div ${attr(screen,'data',column.key)}><dt>${fieldLabel(column)}</dt><dd><bdi>${esc(value(column))}</bdi></dd></div>`).join('');
  let special='';
  if(profile.layout==='document')special=`<article class="sd-document" ${attr(screen,'section','document')}><header><div><p class="sd-eyebrow">${t('document')}</p><h3><bdi>${esc(customerDocument?.document??'DRAFT-DEMO-001')}</bdi></h3></div><span class="tag">${esc(customerDocument?publishedExample:t('documentState'))}</span></header>${customerDocument?`<dl class="facts"><div ${attr(screen,'data','selected-line')}><dt>${t('line')}</dt><dd><bdi>${esc(customerDocument.line)}</bdi></dd></div><div ${attr(screen,'data','document-revision')}><dt>${t('revision')}</dt><dd><bdi>${esc(customerDocument.revision)}</bdi></dd></div></dl>`:''}<div class="sd-document-lines"><div><strong>${t('line')}</strong><strong>${t('service')}</strong><strong>${t('net')}</strong></div><div><bdi>DEMO-LINE-001</bdi><span>Compute</span><span>${t('unavailable')}</span></div><div><bdi>DEMO-LINE-002</bdi><span>Storage</span><span>${t('unavailable')}</span></div></div><dl class="facts">${['currency','net','tax','total'].map(key=>`<div><dt>${t(key)}</dt><dd>${t('unavailable')}</dd></div>`).join('')}</dl></article>`;
  if(profile.layout==='comparison')special=`<section class="sd-comparison" ${attr(screen,'section','comparison')}><h3>${t('comparison')}</h3><div class="sd-compare-grid"><article><h3>${t('before')}</h3><bdi>DEMO-SOURCE-r1</bdi><p>${t('unavailable')}</p></article><article><h3>${t('after')}</h3><bdi>DEMO-CANDIDATE-r2</bdi><p>${t('unavailable')}</p></article></div><p class="notice">${t('difference')}</p></section>`;
  return `<dl class="facts">${cells}</dl>${special}<p class="notice">${screen.group==='customer'?t('customerSafe'):t('evidenceText')}</p>`;
 }
 function evidence(){
  return `<section ${attr(screen,'section','evidence')}><h3>${t('evidence')}</h3><p>${t('evidenceText')}</p><dl class="facts">${['evidenceRef','sourceRevision','time'].map(key=>`<div><dt>${t(key)}</dt><dd>${t('unavailable')}</dd></div>`).join('')}</dl><p>${loc(profile.note)}</p></section>`;
 }
 function history(){
  return `<section ${attr(screen,'section','history')}><h3>${t('history')}</h3><p>${t('historyText')}</p>${state.events.length?`<ol class="sd-timeline">${state.events.map(event=>`<li><strong>${t('event')} · <bdi>r${event.revision}</bdi></strong><time datetime="${event.time}" dir="ltr">${event.time}</time></li>`).join('')}</ol>`:`<p>${t('noHistory')}</p>`}</section>`;
 }
 function detail(){
  return `<section class="panel"><div class="sd-tabs" role="tablist" aria-label="${t('overview')}">${['overview','evidence','history'].map(key=>`<button id="screen-tab-${key}" role="tab" aria-selected="${state.tab===key}" aria-controls="screen-detail-panel" tabindex="${state.tab===key?'0':'-1'}" data-screen-tab="${key}" ${attr(screen,'tab',key)}>${t(key)}</button>`).join('')}</div><div id="screen-detail-panel" role="tabpanel" aria-labelledby="screen-tab-${state.tab}">${state.tab==='overview'?summary():state.tab==='evidence'?evidence():history()}</div><div class="form-actions"><button id="screen-design-inspect" class="primary" ${attr(screen,'action','inspect')}>${t('refresh')}</button><button disabled>${t('actualAction')}</button></div></section>${state.result?result():''}`;
 }
 function workflow(){
  const steps=profile.steps.length?profile.steps:[t('checkData'),t('checkEvidence'),t('checkOutcome')];
  return `<div class="sd-workflow-layout"><section class="panel"><h2>${t('prerequisites')}</h2><p>${loc(profile.note)}</p><div class="workflow">${steps.map((title,index)=>`<article class="step ${state.steps.has(index)?'done':''}"><span class="step-number">${index+1}</span><div><h3>${loc(title)}</h3><p>${state.steps.has(index)?t('checkpointDone'):t('pending')}</p><button data-step="${index}" ${state.steps.has(index)?'disabled':''} ${attr(screen,'action',`check-${stepNames[index]}`)}>${t('checkpoint')}</button></div></article>`).join('')}</div><div class="form-actions"><button id="workflow-review" class="primary" ${state.steps.size===steps.length?'':'disabled'} ${attr(screen,'action','prepare-work')}>${t('draftOnly')}</button></div></section><aside class="panel sd-notes"><h2>${t('controls')}</h2>${facts()}<p class="notice">${t('noIssue')}</p><button disabled>${t('actualAction')}</button></aside></div>${state.result?result():''}`;
 }
 function welcome(){
  const labels=['browse','invitation','administrator','recovery'];
  return `<section class="panel"><h2>${t('startTitle')}</h2><p>${t('authBoundary')}</p><div class="sd-entry-grid">${screen.related.map((id,index)=>`<button data-screen-navigate="${esc(id)}" ${attr(screen,'navigation',id)}><strong>${t(labels[index]??'next')}</strong><span>${t('preview')}</span></button>`).join('')}</div></section>`;
 }
 function body(){return header()+(blocked()||(screen.kind==='form'?form():screen.kind==='workflow'?workflow():screen.kind==='detail'?detail():welcome()));}
 function render(focus){
  if(!isActive())return;
  const host=document.querySelector('#screen-design-root');if(!host)return;
  host.innerHTML=body();bind();if(focus)host.querySelector(focus)?.focus({preventScroll:true});
 }
 function bind(){
  const host=document.querySelector('#screen-design-root');if(!host)return;
  host.querySelector('#screen-design-scenario').onchange=event=>{state.scenario=event.target.value;state.result=false;state.steps.clear();render('#screen-design-scenario');};
  const retry=host.querySelector('#screen-design-retry');if(retry)retry.onclick=()=>{state.scenario='normal';render('#screen-design-scenario');};
  const form=host.querySelector('#screen-form');
  if(form){
   form.oninput=event=>{if(event.target.name){state.values[event.target.name]=event.target.value;state.revision++;state.result=false;host.querySelector('#form-feedback').textContent=t('changed');host.querySelector('#screen-design-result').innerHTML='';}};
   form.onsubmit=event=>{
    event.preventDefault();if(!form.reportValidity())return;
    state.values=Object.fromEntries(new FormData(form));
    if(state.values.effective&&state.values.until&&state.values.until<state.values.effective){host.querySelector('#form-feedback').textContent=t('invalid');form.querySelector('[name=until]')?.focus();return;}
    state.result=true;addEvent();host.querySelector('#form-feedback').textContent=t('saved');host.querySelector('#screen-design-result').innerHTML=result();
   };
   host.querySelector('#screen-design-fill').onclick=()=>{
    for(const field of screen.fields)state.values[field.key]=field.type==='select'?(field.options?.[0]?.value??''):fieldSample(field,account);
    state.revision++;state.result=false;render('#screen-design-fill');
   };
   host.querySelector('#screen-design-clear').onclick=()=>{state.values={};state.result=false;state.revision++;render('#screen-design-clear');};
  }
  const tabs=[...host.querySelectorAll('[data-screen-tab]')];
  tabs.forEach((button,index)=>{
   button.onclick=()=>{state.tab=button.dataset.screenTab;render(`[data-screen-tab="${state.tab}"]`);};
   button.onkeydown=event=>{
    let target=index;
    if(event.key==='ArrowRight')target=(index+(locale==='ar'?-1:1)+tabs.length)%tabs.length;
    else if(event.key==='ArrowLeft')target=(index+(locale==='ar'?1:-1)+tabs.length)%tabs.length;
    else if(event.key==='Home')target=0;
    else if(event.key==='End')target=tabs.length-1;
    else return;
    event.preventDefault();tabs[target].click();
   };
  });
  const inspect=host.querySelector('#screen-design-inspect');if(inspect)inspect.onclick=()=>{state.result=true;addEvent();render('#screen-design-inspect');};
  host.querySelectorAll('[data-step]').forEach(button=>button.onclick=()=>{state.steps.add(Number(button.dataset.step));state.result=false;render('[data-step]:not(:disabled),#workflow-review:not(:disabled)');});
  const review=host.querySelector('#workflow-review');if(review)review.onclick=()=>{if(state.steps.size!==3)return;state.result=true;addEvent();render('#workflow-review');};
  host.querySelectorAll('[data-screen-navigate]').forEach(button=>button.onclick=()=>navigate(button.dataset.screenNavigate));
 }
 return {html:`<div id="screen-design-root">${body()}</div>`,bind,items:designItems(screen,locale)};
}
