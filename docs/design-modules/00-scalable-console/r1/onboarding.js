import {ob} from './onboarding-copy.js';
import {profiles,profileNotes} from './onboarding-profiles.js';
import {label} from './copy.js';
import {esc} from './widgets.js';
export const setupRoutes=['setup-progress','setup-profile','setup-attribution','setup-connectors','setup-readtest','setup-schedule','setup-first','setup-readiness'];
const stepKeys=['start','profile','attribution','connector','diagnostics','schedule','receipt','readiness'];
const drafts=new Map();
let viewRevision=0;
function draft(scope){
 if(!drafts.has(scope))drafts.set(scope,{values:{provider:'aws',route:'existing',identity:'',agreement:'MCA',ibmRoute:'csv',scenario:'normal'},done:new Set(),revision:1,plan:false,admin:false,diagnostic:'idle',receipt:'idle',run:0});
 return drafts.get(scope);
}
export function disposeOnboarding(){
 viewRevision++;
 for(const state of drafts.values()){state.run++;if(state.diagnostic==='running')state.diagnostic='idle';if(state.receipt==='running')state.receipt='idle';}
}
export function onboarding(id,scope,locale,onNavigate,isActive){
 const state=draft(scope),v=state.values,index=setupRoutes.indexOf(id),view=++viewRevision;
 let p=profiles[v.provider];
 const t=key=>ob(key,locale);
 const canStep=i=>i<=1||state.done.has(setupRoutes[i-1]);
 const choices=(key,title,items)=>`<label>${esc(title)}<select name="${key}">${items.map(([value,text])=>`<option value="${value}" ${v[key]===value?'selected':''}>${esc(text)}</option>`).join('')}</select></label>`;
 const field=(key,title,type='text',pattern='')=>`<label>${esc(title)} *<input required name="${key}" type="${type}" value="${esc(v[key]??'')}" maxlength="200" ${pattern?`pattern="${pattern}"`:''}></label>`;
 const check=(key,title)=>`<label class="inline"><input name="${key}" type="checkbox" ${v[key]?'checked':''}>${esc(title)}</label>`;
 function blockedProfile(){
  return p.unverified||(v.provider==='azure'&&!['EA','MCA'].includes(v.agreement))||(v.provider==='gcp'&&(!v.preview||!['US','EU'].includes(v.location)))||(v.provider==='alibaba'&&!v.preview)||(v.provider==='ibm'&&v.ibmRoute!=='csv');
 }
 function requiredFields(){return v.provider==='aws'&&v.route==='new'?[]:p.fields;}
 function configurationReady(){
  return !!v.account&&(v.provider!=='aws'||/^\d{12}$/.test(v.account))&&requiredFields().every(f=>v[f.key])&&!!v.identity&&!blockedProfile();
 }
 function header(){return `<div class="notice">${t('draft')}</div><nav class="setup-steps" aria-label="${t('start')}">${setupRoutes.map((route,i)=>`<button data-setup-route="${route}" ${i===index?'aria-current="step"':''} ${canStep(i)?'':'disabled'}>${i+1}. ${t(stepKeys[i])}${state.done.has(route)?` · ${t('pass')}`:''}</button>`).join('')}</nav>`;}
 function body(){
  if(index===0)return `<section class="panel"><h2>${t('start')}</h2><p>${t('history')}</p><div class="stats"><article class="stat">${t('technical')}<strong>${state.diagnostic==='pass'?t('pass'):t('missing')}</strong></article><article class="stat">${t('coverage')}<strong>${state.receipt==='pass'?t('pass'):t('missing')}</strong></article><article class="stat">${t('financial')}<strong>${t('blocked')}</strong></article></div><button id="setup-start" class="primary">${state.done.size?t('resume'):t('start')}</button></section>`;
  if(!canStep(index))return `<section class="panel"><h2>${t('prerequisite')}</h2><button id="setup-start">${t('resume')}</button></section>`;
  let content='';
  if(index===1)content=`${field('entity',t('entity'))}${choices('country',t('country'),[['','—'],['US','United States'],['AE','United Arab Emirates'],['KR','대한민국']])}${choices('currency',t('currency'),[['','—'],['USD','USD'],['AED','AED'],['KRW','KRW']])}${choices('timezone',t('timezone'),[['','—'],['America/New_York','America/New_York'],['Asia/Dubai','Asia/Dubai'],['Asia/Seoul','Asia/Seoul']])}${field('tax',t('tax'))}`;
  if(index===2)content=`${field('customer',t('customer'))}${field('custody',t('custody'))}${field('effective',t('effective'),'date')}${check('scopeAck',t('scopeAck'))}`;
  if(index===3){
   content=`${choices('provider',t('provider'),Object.entries(profiles).map(([key,value])=>[key,value.name]))}${choices('route',t('route'),[['existing',t('existing')],['new',t('new')]])}${field('account',t('account'),'text',p.pattern)}${choices('identity',t('identity'),[['',t('missing')],['DEMO-READ-IDENTITY',t('approved')]])}`;
   if(v.provider==='azure')content+=choices('agreement','Azure billing agreement',[['MCA','MCA'],['EA','EA'],['MOSP','MOSP'],['management','Management group']]);
   if(['gcp','alibaba'].includes(v.provider))content+=check('preview',locale==='ko'?'예시 Preview 사용 권한 확인':locale==='ar'?'تأكيد أهلية Preview التجريبية':'Confirm example Preview eligibility');
   if(v.provider==='ibm')content+=choices('ibmRoute','IBM report route',[['csv','Account CSV'],['api','Account API — unverified'],['enterprise','Enterprise API — unverified']]);
   content+=requiredFields().map(f=>field(f.key,label(f.label,locale))).join('');
  }
  if(index===5)content=`${field('period',t('period'),'month')}${choices('cadence',t('cadence'),(v.provider==='file'||v.provider==='ibm'?[['once',t('once')]]:[['','—'],['daily',t('daily')],['monthly',t('monthly')]]))}`;
  if([1,2,3,5].includes(index)){
   return `<section class="panel"><form id="setup-form"><div class="form-grid">${content}</div><p id="setup-error" role="alert"></p><div class="form-actions"><button type="button" id="setup-example">${t('demo')}</button>${index!==3?`<button type="submit" class="primary">${t('saveNext')}</button>`:`<button type="button" id="setup-plan">${t('reviewPlan')}</button>`}</div></form>${index===3?connectionPlan():''}${index===5?`<p class="notice">${t('history')}</p>`:''}</section>`;
  }
  if(index===4)return `<section class="panel"><div class="toolbar">${choices('scenario',t('scenario'),[['normal',t('normal')],['denied',t('denied')],['partial',t('partial')],['outcome',t('outcome')]])}</div><h2>${t('diagnostics')}</h2><p><bdi>${esc(v.account)} · revision ${state.revision}</bdi></p><div class="workflow">${['network','metadata','read'].map(key=>`<article class="step"><div><h3>${t(key)}</h3><p>${state.diagnostic==='pass'?t('pass'):state.diagnostic==='running'?t('running'):state.diagnostic==='failed'?(key==='read'?t('fail'):t('pass')):state.diagnostic==='unknown'?t('unknown'):t('missing')}</p></div></article>`).join('')}</div><div class="form-actions"><button id="setup-diagnose" class="primary" ${state.diagnostic==='running'?'disabled':''}>${t('run')}</button><button id="setup-next" ${state.diagnostic==='pass'?'':'disabled'}>${t('saveNext')}</button></div></section>`;
  if(index===6)return `<section class="panel"><h2>${t('receipt')}</h2><p><bdi>${esc(v.account)} · ${esc(v.period)} · ${esc(v.cadence)}</bdi></p><div class="workflow">${['complete','schema','evidence'].map(key=>`<article class="step"><div><h3>${t(key)}</h3><p>${key==='evidence'?t('blocked'):state.receipt==='pass'?t('pass'):state.receipt==='running'?t('running'):state.receipt==='partial'?t('receiptPartial'):t('missing')}</p></div></article>`).join('')}</div><div class="form-actions"><button id="setup-validate" class="primary" ${state.receipt==='running'?'disabled':''}>${t('validate')}</button><button id="setup-next" ${state.receipt==='pass'?'':'disabled'}>${t('saveNext')}</button></div></section>`;
  return `<section class="panel"><h2>${t('readiness')}</h2><dl class="facts"><div><dt>${t('technical')}</dt><dd>${state.diagnostic==='pass'?t('pass'):t('missing')}</dd></div><div><dt>${t('coverage')}</dt><dd>${state.receipt==='pass'?t('pass'):t('missing')}</dd></div><div><dt>${t('financial')}</dt><dd>${t('blocked')}</dd></div><div><dt>${t('cadence')}</dt><dd>${['ibm','file'].includes(v.provider)?t('once'):esc(v.cadence)}</dd></div></dl><div class="notice">${t('evidence')}</div><div class="form-actions"><button disabled>${t('publish')}</button><button id="setup-work" class="primary">${t('continueWork')}</button></div></section>`;
 }
 function connectionPlan(){
  const note=profileNotes[v.provider];
  return `<div class="notice">${note?esc(label(note,locale)):t('unverified')}</div>${blockedProfile()?`<p role="status">${t('unverified')}</p>`:''}${!v.identity?`<section class="panel"><h3>${t('handoff')}</h3><p>${t('identity')} · ${t('missing')}</p><p>${t('handoffNote')}</p></section>`:''}${state.plan?`<section class="panel"><h3>${t('plan')}</h3><p>${v.provider==='aws'&&v.route==='new'?t('awsPlan'):v.route==='new'?t('planNew'):t('planExisting')}</p><p><bdi>${p.name} · ${esc(v.account)} · revision ${state.revision}</bdi></p>${check('planAck',t('planAck'))}${v.route==='new'?`<h3>${t('handoff')}</h3><p>${t('owner')}: ${p.name} administrator</p><p>${t('handoffNote')}</p><p>${state.admin?t('pass'):t('waiting')}</p><button id="setup-admin" ${v.planAck&&!state.admin?'':'disabled'}>${t('resumeCheck')}</button>`:''}<div class="form-actions"><button id="setup-connection-next" class="primary" ${v.planAck&&(v.route==='existing'||state.admin)?'':'disabled'}>${t('saveNext')}</button></div></section>`:''}`;
 }
 function invalidate(from=index){
  for(let i=from;i<setupRoutes.length;i++)state.done.delete(setupRoutes[i]);
  state.revision++;state.run++;
  if(from<=3){state.plan=false;state.admin=false;v.planAck=false;}
  if(from<=4)state.diagnostic='idle';
  if(from<=6)state.receipt='idle';
 }
 function refresh(focus){
  if(view!==viewRevision||!isActive())return;
  p=profiles[v.provider];
  const host=document.querySelector('#content');if(!host)return;
  host.innerHTML=header()+body();bind();
  if(focus)document.querySelector(focus)?.focus();
 }
 function saveValues(){
  const form=document.querySelector('#setup-form');if(!form)return;
  for(const input of form.elements)if(input.name)v[input.name]=input.type==='checkbox'?input.checked:input.value;
 }
 function complete(){state.done.add(id);onNavigate(setupRoutes[Math.min(index+1,7)]);}
 function example(){
  if(index===1)Object.assign(v,{entity:'Example Billing LLC',country:'AE',currency:'AED',timezone:'Asia/Dubai',tax:'DRAFT-TAX-POLICY'});
  if(index===2)Object.assign(v,{customer:'CUSTOMER-DEMO-001',custody:'CUSTODIAN-DEMO-001',effective:'2026-10-01',scopeAck:true});
  if(index===3){v.account=p.sample;v.identity='DEMO-READ-IDENTITY';for(const f of p.fields)v[f.key]=f.sample;v.preview=true;}
  if(index===5)Object.assign(v,{period:'2026-10',cadence:['ibm','file'].includes(v.provider)?'once':'daily'});
  invalidate();refresh('#setup-example');
 }
 function bind(){
  const host=document.querySelector('#content');
  host.querySelectorAll('[data-setup-route]').forEach(button=>button.onclick=()=>{if(canStep(setupRoutes.indexOf(button.dataset.setupRoute)))onNavigate(button.dataset.setupRoute);});
  const start=host.querySelector('#setup-start');if(start)start.onclick=()=>onNavigate(setupRoutes.find((route,i)=>i>0&&!state.done.has(route))??'setup-readiness');
  const form=host.querySelector('#setup-form');
  if(form){
   form.oninput=event=>{if(event.target.name){saveValues();invalidate();}};
   form.onchange=event=>{
    const key=event.target.name;
    if(key==='provider'){
     for(const profile of Object.values(profiles))for(const field of profile.fields)delete v[field.key];
     v.account='';v.preview=false;v.identity='';v.route='existing';v.ibmRoute='csv';
    }
    if(['provider','route','ibmRoute','agreement','identity','preview'].includes(key))refresh(`[name="${key}"]`);
   };
   form.onsubmit=event=>{
    event.preventDefault();saveValues();
    if(index===3){host.querySelector('#setup-plan').click();return;}
    if(!canStep(index)||!form.reportValidity()||(index===1&&(!v.country||!v.currency||!v.timezone))||(index===2&&!v.scopeAck)||(index===5&&!v.cadence)){
     host.querySelector('#setup-error').textContent=t('invalid');return;
    }
    complete();
   };
   host.querySelector('#setup-example').onclick=example;
  }
  const plan=host.querySelector('#setup-plan');if(plan)plan.onclick=()=>{saveValues();if(!canStep(3)||!form.reportValidity()||!configurationReady()){host.querySelector('#setup-error').textContent=blockedProfile()?t('unverified'):t('invalid');return;}state.plan=true;refresh('#setup-connection-next');};
  const ack=host.querySelector('[name=planAck]');if(ack)ack.onchange=()=>{const checked=ack.checked;invalidate(3);state.plan=true;v.planAck=checked;refresh('[name=planAck]');};
  const admin=host.querySelector('#setup-admin');if(admin)admin.onclick=()=>{if(canStep(3)&&state.plan&&v.planAck&&configurationReady()){state.admin=true;refresh('#setup-connection-next');}};
  const connectionNext=host.querySelector('#setup-connection-next');if(connectionNext)connectionNext.onclick=()=>{if(canStep(3)&&state.plan&&v.planAck&&configurationReady()&&(v.route==='existing'||state.admin))complete();};
  const scenario=host.querySelector('[name=scenario]');if(scenario)scenario.onchange=()=>{v.scenario=scenario.value;invalidate(4);refresh('[name=scenario]');};
  const diagnose=host.querySelector('#setup-diagnose');if(diagnose)diagnose.onclick=()=>{
   if(!canStep(4)||!configurationReady()||!state.plan||!v.planAck)return;
   invalidate(4);const run=++state.run;state.diagnostic='running';refresh();
   setTimeout(()=>{if(state.run!==run)return;state.diagnostic=v.scenario==='denied'?'failed':v.scenario==='outcome'?'unknown':'pass';if(isActive())refresh('#setup-diagnose');},450);
  };
  const validate=host.querySelector('#setup-validate');if(validate)validate.onclick=()=>{
   if(!canStep(6)||state.diagnostic!=='pass')return;
   invalidate(6);const run=++state.run;state.receipt='running';refresh();
   setTimeout(()=>{if(state.run!==run)return;state.receipt=v.scenario==='partial'?'partial':'pass';if(isActive())refresh('#setup-validate');},450);
  };
  const next=host.querySelector('#setup-next');if(next)next.onclick=()=>{if(canStep(index)&&((index===4&&state.diagnostic==='pass')||(index===6&&state.receipt==='pass')))complete();};
  const work=host.querySelector('#setup-work');if(work)work.onclick=()=>onNavigate('operations');
 }
 return {html:header()+body(),bind};
}
