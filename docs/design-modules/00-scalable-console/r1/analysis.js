import {ai} from './analysis-copy.js';
import {esc} from './widgets.js';
const states=new Map();
let activeView=0;
const stateFor=scope=>{if(!states.has(scope))states.set(scope,{status:'idle',request:'',period:'2026-10',currency:'USD',scenario:'normal',generation:0,task:false,context:null});return states.get(scope);};
export function setAnalysisContext(scope,context){const s=stateFor(scope);s.context={...context};if(context.period)s.period=context.period;if(context.currency)s.currency=context.currency;s.generation++;s.status='idle';s.task=false;}
export function invalidateAnalysis(scope){const s=states.get(scope);if(s){s.generation++;s.status='stale';s.task=false;s.context=null;}}
export function disposeAnalysis(){
 activeView++;
 for(const state of states.values())if(state.status==='running'){state.generation++;state.status='cancelled';}
}
export function analysisScreen(scope,locale,onEvidence,isActive){
 const state=stateFor(scope),view=++activeView,t=key=>ai(key,locale);
 const current=()=>isActive()&&view===activeView;
 const context=()=>state.context??{screen:'index-intake',record:null,snapshot:'demo-current'};
 function html(){
  const c=context();
  const success=['complete','partial'].includes(state.status);
  return `<section class="panel"><div class="notice">${t('notice')}</div><h2>${t('scope')}</h2><dl class="facts"><div><dt>${t('scope')}</dt><dd><bdi>${esc(scope)}</bdi></dd></div><div><dt>${t('record')}</dt><dd><bdi>${esc(c.record??t('noContext'))}</bdi></dd></div><div><dt>${t('source')}</dt><dd><bdi>${esc(c.screen)} · ${esc(c.snapshot)}</bdi></dd></div></dl><p>${t('bounds')}</p><p class="muted">${t('budget')}</p><form id="analysis-form"><div class="form-grid"><label>${t('period')}<input name="period" required type="month" value="${esc(state.period)}"></label><label>${t('currency')}<select required name="currency">${['USD','AED','KRW'].map(value=>`<option ${state.currency===value?'selected':''}>${value}</option>`).join('')}</select></label></div><label class="analysis-question">${t('question')}<textarea name="request" required maxlength="1000" rows="4" placeholder="${t('placeholder')}">${esc(state.request)}</textarea></label><div class="toolbar"><label>${t('scenario')}<select name="scenario">${['normal','partial','failed'].map(key=>`<option value="${key}" ${state.scenario===key?'selected':''}>${t(key)}</option>`).join('')}</select></label><button type="submit" class="primary" ${state.status==='running'?'disabled':''}>${state.status==='error'?t('retry'):t('run')}</button>${state.status==='running'?`<button id="analysis-cancel" type="button">${t('cancel')}</button>`:''}</div></form><div id="analysis-status" role="status">${state.status==='running'?t('running'):state.status==='error'?t('error'):state.status==='cancelled'?t('cancelled'):state.status==='stale'?t('stale'):''}</div></section>${success?`<section class="panel" id="analysis-result"><h2>${t('result')}</h2><p>${t('generated')}: <bdi>${esc(state.generated??'')}</bdi></p>${state.status==='partial'?`<div class="notice">${t('partialText')}</div>`:''}<h3>${t('observed')}</h3><p>${t('fact')}</p><button id="analysis-evidence">${t('evidence')} · <bdi>${esc(c.record??'DEMO-EVIDENCE')}</bdi></button><h3 class="analysis-question">${t('hypothesis')}</h3><p>${t('hypothesisText')}</p><div class="notice">${t('limitations')}</div><h3>${t('plan')}</h3><p>${t('planText')}</p><button id="analysis-task" ${state.task?'disabled':''}>${t('task')}</button><p role="status">${state.task?t('taskReady'):''}</p></section>`:''}`;
 }
 function render(focus){if(!current())return;document.querySelector('#content').innerHTML=html();bind();if(focus)document.querySelector(focus)?.focus();}
 function bind(){
  const form=document.querySelector('#analysis-form');if(!form)return;
  form.oninput=event=>{
   if(event.target.name){state[event.target.name]=event.target.value;state.generation++;state.task=false;if(state.status!=='idle')state.status='stale';
    document.querySelector('#analysis-status').textContent=state.status==='stale'?t('stale'):'';
    document.querySelector('#analysis-result')?.remove();
    form.querySelector('[type=submit]').disabled=false;
   }
  };
  form.onchange=event=>{if(event.target.tagName==='SELECT')form.oninput(event);};
  form.onsubmit=event=>{
   event.preventDefault();if(state.status==='running'||!form.reportValidity()||!state.request.trim())return;
   const generation=++state.generation;state.status='running';state.task=false;render();
   setTimeout(()=>{if(state.generation!==generation)return;state.generated=new Date().toISOString();state.status=state.scenario==='failed'?'error':state.scenario==='partial'?'partial':'complete';render();},700);
  };
  const cancel=document.querySelector('#analysis-cancel');if(cancel)cancel.onclick=()=>{state.generation++;state.status='cancelled';render('[type=submit]');};
  const evidence=document.querySelector('#analysis-evidence');if(evidence)evidence.onclick=()=>onEvidence(context());
  const task=document.querySelector('#analysis-task');if(task)task.onclick=()=>{if(!['complete','partial'].includes(state.status))return;state.task=true;render('#analysis-evidence');};
 }
 return {html:html(),bind};
}
