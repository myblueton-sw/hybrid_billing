import {esc} from './widgets.js';
import {label} from './copy.js';
import {gapText} from './journeys-copy.js';
export {gapText} from './journeys-copy.js';
const reports=new Map();
const approvals=new Map();
const select=(name,title,values,current)=>`<label>${esc(title)}<select name="${name}">${values.map(([value,text])=>`<option value="${value}" ${value===current?'selected':''}>${esc(text)}</option>`).join('')}</select></label>`;
const facts=items=>`<dl class="facts">${items.map(([name,value])=>`<div><dt>${esc(name)}</dt><dd><bdi>${esc(value)}</bdi></dd></div>`).join('')}</dl>`;
export function customerOverview(account,locale,screens,link){
 const t=key=>gapText(key,locale);
 return `<section class="panel"><h2>${t('customer')}</h2><p>${t('customerNotice')}</p>${facts([[t('account'),account],[t('documents'),t('unavailable')],[t('balance'),t('unavailable')]])}<h3>${t('tasks')}</h3><div class="screen-index">${screens.filter(item=>item.id!=='core-customer-overview').map(item=>`<a href="${link(item.id)}">${esc(label(item.title,locale))}</a>`).join('')}</div></section>`;
}
export function reportScreen(screen,scope,locale,onAnalyze){
 const key=`${scope}:${screen.id}`,t=key=>gapText(key,locale);
 if(!reports.has(key))reports.set(key,{period:'2026-10',comparison:'2026-09',currency:'USD',basis:'billed',grouping:'provider',preview:false});
 const state=reports.get(key);
 const descriptor=()=>facts([[t('account'),scope],[t('period'),state.period],[t('comparison'),state.comparison],[t('currency'),state.currency],[t('basis'),t(state.basis)],[t('grouping'),t(state.grouping)]]);
 const result=()=>state.preview?`<h3>${t('descriptor')}</h3>${descriptor()}<p>${t('reportLimit')}</p><p>${t('before')}: ${t('unavailable')} · ${t('after')}: ${t('unavailable')}</p><button id="report-analysis">${t('analyze')}</button><button disabled>${t('export')}</button>`:'';
 function bind(){
  const form=document.querySelector('#report-form');
  form.oninput=event=>{
   if(!event.target.name)return;
   state[event.target.name]=event.target.value;state.preview=false;
   document.querySelector('#report-result').textContent=t('stale');
  };
  form.onchange=form.oninput;
  form.onsubmit=event=>{
   event.preventDefault();if(!form.reportValidity())return;
   state.preview=true;document.querySelector('#report-result').innerHTML=result();bindAnalysis();
  };
  bindAnalysis();
 }
 function bindAnalysis(){
  const button=document.querySelector('#report-analysis');
  if(button)button.onclick=()=>{if(state.preview)onAnalyze({period:state.period,currency:state.currency,report:{...state}});};
 }
 return {html:`<section class="panel"><div class="notice">${t('notice')}</div><form id="report-form"><div class="form-grid">${['period','comparison'].map(name=>`<label>${t(name)}<input type="month" name="${name}" required value="${state[name]}"></label>`).join('')}${select('currency',t('currency'),['USD','AED','KRW'].map(value=>[value,value]),state.currency)}${select('basis',t('basis'),['billed','effective'].map(value=>[value,t(value)]),state.basis)}${select('grouping',t('grouping'),['provider','service','account'].map(value=>[value,t(value)]),state.grouping)}</div><button type="submit" class="primary">${t('run')}</button></form><section id="report-result" aria-live="polite">${result()}</section></section>`,bind};
}
export function approvalScreen(scope,locale,isActive){
 const t=key=>gapText(key,locale);
 if(!approvals.has(scope))approvals.set(scope,{revision:'DEMO-r1',currency:'USD',reviewed:false,rejected:false,reason:''});
 const state=approvals.get(scope);
 function html(){
  return `<section class="panel"><div class="notice">${t('notice')}</div><h2>${t('target')} · <bdi>DEMO-DRAFT-001</bdi></h2><div class="form-grid">${select('revision',t('revision'),['DEMO-r1','DEMO-r2'].map(value=>[value,value]),state.revision)}${select('currency',t('currency'),['USD','AED','KRW'].map(value=>[value,value]),state.currency)}</div>${facts([[t('account'),scope],[t('revision'),state.revision],[t('currency'),state.currency],[t('before'),t('unavailable')],[t('after'),t('unavailable')],[t('difference'),t('unavailable')]])}<p class="notice">${t('blocking')}</p>${facts([[t('review'),state.reviewed?t('reviewed'):t('pending')],[t('approval'),state.rejected?t('rejected'):t('blocked')],[t('issuance'),t('notIssued')]])}<button id="approval-review" ${state.reviewed||state.rejected?'disabled':''}>${t('review')}</button><button id="approval-approve" disabled>${t('approve')}</button><form id="approval-reject"><label>${t('reason')}<textarea name="reason" required maxlength="500" ${state.rejected?'disabled':''}>${esc(state.reason)}</textarea></label><button type="submit" ${state.rejected?'disabled':''}>${t('reject')}</button></form><p id="approval-status" role="status">${state.rejected?t('rejected'):''}</p></section>`;
 }
 function render(focus){if(!isActive())return;document.querySelector('#content').innerHTML=html();bind();document.querySelector(focus)?.focus();}
 function bind(){
  for(const name of ['revision','currency'])document.querySelector(`[name=${name}]`).onchange=event=>{
   state[name]=event.target.value;state.reviewed=false;state.rejected=false;state.reason='';render(`[name=${name}]`);
  };
  document.querySelector('#approval-review').onclick=()=>{state.reviewed=true;render('[name=reason]');};
  document.querySelector('[name=reason]').oninput=event=>{state.reason=event.target.value;};
  document.querySelector('#approval-reject').onsubmit=event=>{
   event.preventDefault();if(state.rejected||!event.target.reportValidity()||!state.reason.trim())return;
   state.rejected=true;render('[name=revision]');
  };
 }
 return {html:html(),bind};
}
