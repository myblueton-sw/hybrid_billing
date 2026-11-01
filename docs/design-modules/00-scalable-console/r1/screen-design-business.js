import {esc} from './widgets.js';
import {sd} from './screen-designs-copy.js';
import {businessText} from './screen-design-business-copy.js';
const routes=new Set(['customer-inquiry','customer-response','review-issue','erp']);
const cases=new Map(),reviews=new Map(),submissions=new Map(),scenarios=new Map();
const itemId=(screen,type,key)=>`${screen.id}:${type}:${key}`;
const attr=(screen,type,key)=>`data-design-id="${itemId(screen,type,key)}"`;
const caseFor=scope=>{if(!cases.has(scope))cases.set(scope,{id:null,document:'DEMO-PUBLISHED-001',line:'DEMO-LINE-001',revision:'DEMO-r1',category:'usage',subject:'',question:'',reply:'',messages:[],status:'draft',hasReply:false,notice:''});return cases.get(scope);};
export function customerDocumentContext(scope){const {document,line,revision}=caseFor(scope);return Object.freeze({document,line,revision});}
const reviewFor=scope=>{if(!reviews.has(scope))reviews.set(scope,{revision:'DEMO-r1',currency:'USD',issuer:'',reviewer:'',source:false,recipient:false,payment:false,prepared:false});return reviews.get(scope);};
const submissionFor=scope=>{if(!submissions.has(scope))submissions.set(scope,{destination:'DEMO-ERP',entity:'DEMO-ENTITY',method:'api',basis:'DEMO-RECOGNITION',revision:'DEMO-r1',id:null,status:'notSubmitted',scenario:'pending',attempt:0,checkedAt:null,reconciled:false});return submissions.get(scope);};
export function businessItems(screen,locale='en'){
 if(!routes.has(screen.id))return [];
 const t=key=>businessText(key,locale),items=[];
 const add=(type,key,title,behavior)=>items.push({id:itemId(screen,type,key),label:title,title,behavior,expected:behavior,status:'prototype',selector:`[data-design-id="${itemId(screen,type,key)}"]`});
 add('section','business-context',t('scope'),t('notice'));add('state','scenario',sd('scenario',locale),t('notice'));
 if(screen.id==='customer-inquiry'){
  for(const key of ['document','line','category','subject','question'])add('field',key,t(key),t('caseBoundary'));
  add('action','create-case',t('saveCase'),t('savedCase'));add('action','fill-case',t('fill'),t('caseBoundary'));
 }else if(screen.id==='customer-response'){
  add('section','public-thread',t('threadTitle'),t('bounded'));add('field','reply',t('replyInput'),t('caseBoundary'));
  for(const [key,title] of [['load-case','loadCase'],['reply','reply'],['example-reply','exampleReply'],['acknowledge','acknowledge']])add('action',key,t(title),t('caseBoundary'));
 }else if(screen.id==='review-issue'){
  for(const key of ['revision','currency','issuer','reviewer','source','recipient','payment'])add('field',key,t(`${key}Check`)===`${key}Check`?t(key):t(`${key}Check`),t('reviewMissing'));
  add('action','prepare-review',t('prepareReview'),t('reviewPrepared'));add('action','approve',t('approval'),t('amountMissing'));add('action','issue',t('issuance'),t('amountMissing'));
 }else{
  for(const [key,title] of [['destination','destination'],['entity','entity'],['method','path'],['basis','basis'],['revision','revision'],['readback','readbackScenario']])add('field',key,t(title),t('erpBoundary'));
  for(const [key,title] of [['submit','request'],['inspect','inspect'],['retry','retry'],['reconcile','reconcile'],['post','posting']])add('action',key,t(title),t('erpBoundary'));
  add('section','reconciliation',t('reconciliation'),t('reconciled'));
 }
 return items;
}
export function createBusinessDesign(screen,{locale='en',account,scope=account,navigate,isActive=()=>true}){
 if(!routes.has(screen.id))return null;
 const t=key=>businessText(key,locale),generic=key=>sd(key,locale),scenarioKey=`${scope}:${screen.id}`;
 if(!scenarios.has(scenarioKey))scenarios.set(scenarioKey,'normal');
 const c=caseFor(scope),r=reviewFor(scope),s=submissionFor(scope);
 const select=(name,title,options,current,disabled=false)=>`<label ${attr(screen,'field',name)}>${esc(title)}<select name="${name}" ${disabled?'disabled':''}>${options.map(([value,text])=>`<option value="${value}" ${value===current?'selected':''}>${esc(text)}</option>`).join('')}</select></label>`;
 const field=(name,title,value,disabled=false)=>`<label ${attr(screen,'field',name)}>${esc(title)}<input required name="${name}" value="${esc(value)}" maxlength="160" autocomplete="off" ${disabled?'disabled':''}></label>`;
 const facts=values=>`<dl class="facts">${values.map(([title,value])=>`<div><dt>${esc(title)}</dt><dd><bdi>${esc(value)}</bdi></dd></div>`).join('')}</dl>`;
 const button=(id,key,title,disabled=false,extra='')=>`<button id="${id}" ${attr(screen,'action',key)} ${disabled?'disabled':''} ${extra}>${esc(title)}</button>`;
 const message=(role,text)=>({role,text,time:new Date().toISOString()});
 function header(){return `<section class="panel" ${attr(screen,'section','business-context')}><p class="notice">${t('notice')}</p>${facts([[t('scope'),account]])}<label>${generic('scenario')}<select id="screen-design-scenario" ${attr(screen,'state','scenario')}>${['normal','empty','error','revoked'].map(value=>`<option value="${value}" ${scenarios.get(scenarioKey)===value?'selected':''}>${generic(value)}</option>`).join('')}</select></label></section>`;}
 function unavailable(){
  const scenario=scenarios.get(scenarioKey);if(scenario==='normal')return '';
  return `<section class="panel" role="status" ${attr(screen,'state',scenario)}><h2>${generic(scenario)}</h2><p>${generic(`${scenario}Text`)}</p>${button('screen-design-retry','retry-state',generic('retry'))}</section>`;
 }
 function caseContext(){return facts([[t('caseId'),c.id??t('draft')],[t('document'),c.document],[t('line'),c.line],[t('revision'),c.revision],[t('caseStatus'),t(c.status)]]);}
 function inquiry(){
  return `<div class="sd-editor-layout"><section class="panel"><h2>${t('caseTitle')}</h2><form id="screen-form"><div class="form-grid">${select('document',t('document'),[['DEMO-PUBLISHED-001','DEMO-PUBLISHED-001']],c.document,!!c.id)}${select('line',t('line'),[['DEMO-LINE-001','DEMO-LINE-001'],['DEMO-LINE-002','DEMO-LINE-002']],c.line,!!c.id)}${select('category',t('category'),['usage','documentInfo','download'].map(key=>[key,t(key)]),c.category,!!c.id)}${field('subject',t('subject'),c.subject,!!c.id)}</div><label ${attr(screen,'field','question')}>${t('question')}<textarea name="question" rows="5" required maxlength="1000" ${c.id?'disabled':''}>${esc(c.question)}</textarea></label><p>${t('caseBoundary')}</p><div class="form-actions">${button('case-submit','create-case',t('saveCase'),!!c.id,'type="submit" class="primary"')}${button('screen-design-fill','fill-case',t('fill'),!!c.id,'type="button"')}</div><p id="form-feedback" role="status">${c.id?t('savedCase'):''}</p></form></section><aside class="panel"><h2>${t('caseId')}</h2>${caseContext()}${button('case-open-thread','open-thread',t('openThread'),!c.id)}</aside></div>`;
 }
 function response(){
  if(!c.id)return `<section class="panel"><h2>${t('threadTitle')}</h2><p>${t('noCase')}</p><div class="form-actions">${button('case-create','create-draft',t('compose'))}${button('case-load','load-case',t('loadCase'))}</div></section>`;
  return `<div class="sd-editor-layout"><section class="panel" ${attr(screen,'section','public-thread')}><h2>${esc(c.subject)}</h2><p>${t('bounded')}</p><ol class="sd-public-thread">${c.messages.map((entry,index)=>`<li><div><strong>${t(entry.role)}</strong><time datetime="${entry.time}" dir="ltr">${entry.time}</time></div><p>${esc(entry.text)}</p><small><bdi>${c.id} · ${index+1}</bdi></small></li>`).join('')}</ol><form id="case-reply-form"><label ${attr(screen,'field','reply')}>${t('replyInput')}<textarea name="reply" rows="3" maxlength="1000" required ${c.status==='confirmed'||c.messages.length>=10?'disabled':''}>${esc(c.reply)}</textarea></label><div class="form-actions">${button('case-reply','reply',t('reply'),c.status==='confirmed'||c.messages.length>=10,'type="submit"')}</div></form><div class="form-actions">${button('case-example-reply','example-reply',t('exampleReply'),c.hasReply||c.messages.length>=10)}${button('case-acknowledge','acknowledge',t('acknowledge'),!c.hasReply||c.status==='confirmed')}</div><p role="status" id="case-status">${c.notice?t(c.notice):t(c.status)}</p></section><aside class="panel"><h2>${t('document')}</h2>${caseContext()}${button('case-document','open-document',t('openDocument'))}<p class="notice">${t('caseBoundary')}</p></aside></div>`;
 }
 function canReview(){return r.issuer&&r.reviewer&&r.issuer!==r.reviewer&&r.source&&r.recipient&&r.payment;}
 function reviewIssue(){
  const people=[['',t('choose')],['DEMO-PERSON-A','DEMO-PERSON-A'],['DEMO-PERSON-B','DEMO-PERSON-B']];
  return `<div class="sd-editor-layout"><section class="panel"><h2>${t('reviewTitle')}</h2><div class="form-grid">${select('revision',t('revision'),['DEMO-r1','DEMO-r2'].map(value=>[value,value]),r.revision)}${select('currency',t('currency'),['USD','AED','KRW'].map(value=>[value,value]),r.currency)}${select('reviewer',t('reviewer'),people,r.reviewer)}${select('issuer',t('issuer'),people,r.issuer)}</div><p id="issue-assignment-error" role="status">${r.issuer&&r.issuer===r.reviewer?t('samePerson'):t('reviewMissing')}</p><div class="sd-evidence-checks">${['source','recipient','payment'].map(key=>`<label class="inline" ${attr(screen,'field',key)}><input type="checkbox" name="${key}" ${r[key]?'checked':''}>${t(`${key}Check`)}</label>`).join('')}</div><div class="form-actions">${button('workflow-review','prepare-review',t('prepareReview'),!canReview()||r.prepared,'class="primary"')}</div><p id="issue-review-result" role="status" tabindex="-1">${r.prepared?t('reviewPrepared'):''}</p></section><aside class="panel"><h2><bdi>DEMO-DRAFT-001</bdi></h2>${facts([[t('revision'),r.revision],[t('currency'),r.currency],[t('net'),t('unavailable')],[t('tax'),t('unavailable')],[t('total'),t('unavailable')]])}<p class="notice">${t('amountMissing')}</p><div class="form-actions">${button('issue-approve','approve',`${t('approval')} — ${t('blocked')}`,true)}${button('issue-issue','issue',`${t('issuance')} — ${t('blocked')}`,true)}</div></aside></div>`;
 }
 function erp(){
  const pinned=!!s.id;
  return `<section class="panel"><h2>${t('erpTitle')}</h2><p>${t('erpBoundary')}</p><form id="screen-form"><div class="form-grid">${field('destination',t('destination'),s.destination,pinned)}${field('entity',t('entity'),s.entity,pinned)}${select('method',t('path'),[['api',t('api')],['file',t('file')]],s.method,pinned)}${field('basis',t('basis'),s.basis,pinned)}${select('revision',t('revision'),['DEMO-r1','DEMO-r2'].map(value=>[value,value]),s.revision,pinned)}</div><div class="form-actions">${button('erp-submit','submit',t('request'),pinned,'type="submit" class="primary"')}</div><p id="form-feedback" role="status">${pinned?t('requestPinned'):''}</p></form></section><div class="sd-editor-layout"><section class="panel"><h2>${t('submission')}</h2>${facts([[t('submission'),s.id??t('notSubmitted')],[t('submissionStatus'),t(s.status)],[t('posting'),t('notPosted')],[t('attempt'),String(s.attempt)],[t('externalDraft'),s.status==='readback'?'DEMO-EXTERNAL-DRAFT-001':t('unavailable')],[t('checkedAt'),s.checkedAt??t('unavailable')]])}${select('readback',t('readbackScenario'),['pending','unknown','failed','readback'].map(key=>[key,t(key)]),s.scenario,!pinned)}<div class="form-actions">${button('erp-inspect','inspect',t('inspect'),!pinned)}${button('erp-retry','retry',t('retry'),s.status!=='failed')}</div></section><aside class="panel" ${attr(screen,'section','reconciliation')}><h2>${t('reconciliation')}</h2>${facts([[t('platformAmount'),t('unavailable')],[t('externalAmount'),t('unavailable')],[t('difference'),t('unavailable')]])}<div class="form-actions">${button('erp-reconcile','reconcile',t('reconcile'),s.status!=='readback'||s.reconciled)}${button('erp-post','post',`${t('posting')} — ${t('blocked')}`,true)}</div><p id="erp-reconcile-status" role="status" tabindex="-1">${s.reconciled?t('reconciled'):t('amountMissing')}</p></aside></div>`;
 }
 function body(){return header()+(unavailable()||(screen.id==='customer-inquiry'?inquiry():screen.id==='customer-response'?response():screen.id==='review-issue'?reviewIssue():erp()));}
 function render(focus){if(!isActive())return;const host=document.querySelector('#screen-business-root');if(!host)return;host.innerHTML=body();bind();if(focus)host.querySelector(focus)?.focus({preventScroll:true});}
 function bind(){
  const host=document.querySelector('#screen-business-root');if(!host)return;
  host.querySelector('#screen-design-scenario').onchange=event=>{scenarios.set(scenarioKey,event.target.value);render('#screen-design-scenario');};
  const retryState=host.querySelector('#screen-design-retry');if(retryState)retryState.onclick=()=>{scenarios.set(scenarioKey,'normal');render('#screen-design-scenario');};
  if(scenarios.get(scenarioKey)!=='normal')return;
  if(screen.id==='customer-inquiry'){
   const form=host.querySelector('#screen-form');
   form.oninput=event=>{if(event.target.name&&!c.id)c[event.target.name]=event.target.value;};
   form.onsubmit=event=>{event.preventDefault();if(c.id||!form.reportValidity()||!c.subject.trim()||!c.question.trim())return;c.id=`LOCAL-CASE-${account}-001`;c.messages=[message('customer',c.question)];c.status='awaitingReply';render('#case-open-thread');};
   host.querySelector('#screen-design-fill').onclick=()=>{if(c.id)return;c.subject=t('exampleSubject');c.question=t('exampleQuestion');render('#screen-design-fill');};
   host.querySelector('#case-open-thread').onclick=()=>{if(c.id)navigate('customer-response');};
  }
  if(screen.id==='customer-response'){
   const create=host.querySelector('#case-create');if(create)create.onclick=()=>navigate('customer-inquiry');
   const load=host.querySelector('#case-load');if(load)load.onclick=()=>{if(c.id)return;c.id=`LOCAL-CASE-${account}-001`;c.subject=t('exampleSubject');c.question=t('exampleQuestion');c.messages=[message('customer',c.question),message('support',t('exampleAnswer'))];c.hasReply=true;c.status='awaitingCustomer';render('#case-acknowledge');};
   const form=host.querySelector('#case-reply-form');if(form){
    form.oninput=event=>{c.reply=event.target.value;};
    form.onsubmit=event=>{event.preventDefault();if(c.status==='confirmed'||!form.reportValidity()||!c.reply.trim())return;if(c.messages.length>=10){c.notice='limit';render();return;}c.messages.push(message('customer',c.reply));c.reply='';c.status='awaitingReply';c.hasReply=false;c.notice='';render('[name=reply]');};
   }
   const example=host.querySelector('#case-example-reply');if(example)example.onclick=()=>{if(c.hasReply||c.messages.length>=10)return;c.messages.push(message('support',t('exampleAnswer')));c.hasReply=true;c.status='awaitingCustomer';c.notice='';render('#case-acknowledge');};
   const acknowledge=host.querySelector('#case-acknowledge');if(acknowledge)acknowledge.onclick=()=>{if(!c.hasReply||c.status==='confirmed')return;c.status='confirmed';c.notice='confirmed';render('#case-document');};
   const document=host.querySelector('#case-document');if(document)document.onclick=()=>navigate('core-customer-invoice');
  }
  if(screen.id==='review-issue'){
   host.querySelectorAll('[name]').forEach(input=>input.onchange=()=>{r[input.name]=input.type==='checkbox'?input.checked:input.value;r.prepared=false;if(['revision','currency'].includes(input.name)){r.source=false;r.recipient=false;r.payment=false;}render(`[name="${input.name}"]`);});
   host.querySelector('#workflow-review').onclick=()=>{if(!canReview()||r.prepared)return;r.prepared=true;render('#issue-review-result');};
  }
  if(screen.id==='erp'){
   const form=host.querySelector('#screen-form');
   form.oninput=event=>{if(event.target.name&&!s.id)s[event.target.name]=event.target.value;};
   form.onsubmit=event=>{event.preventDefault();if(s.id||!form.reportValidity()||![s.destination,s.entity,s.basis].every(value=>value.trim()))return;s.id=`LOCAL-ERP-${account}-001`;s.status='pending';s.attempt=1;render('#erp-inspect');};
   const outcome=host.querySelector('[name=readback]');outcome.onchange=()=>{s.scenario=outcome.value;};
   host.querySelector('#erp-inspect').onclick=()=>{if(!s.id)return;s.status=s.scenario;s.checkedAt=new Date().toISOString();s.reconciled=false;render('#erp-inspect');};
   host.querySelector('#erp-retry').onclick=()=>{if(s.status!=='failed')return;s.attempt++;s.status='pending';s.checkedAt=null;s.reconciled=false;render('#erp-inspect');};
   host.querySelector('#erp-reconcile').onclick=()=>{if(s.status!=='readback'||s.reconciled)return;s.reconciled=true;render('#erp-reconcile-status');};
  }
 }
 return {html:`<div id="screen-business-root">${body()}</div>`,bind,items:businessItems(screen,locale)};
}
