import {groups,screens} from './catalog.js';
import {t,label} from './copy.js';
import {providers,states,queryRows,signature} from './data.js';
import {esc,modal,closeDialog,toast,accountPicker,createJobs,jobsDialog,renderJobs,fileIntake} from './widgets.js';
const $=selector=>document.querySelector(selector);
const params=new URLSearchParams(location.search);
let locale=['en','ko','ar'].includes(params.get('lang'))?params.get('lang'):'en';
let screen=screens.find(x=>x.id===location.hash.slice(1))??screens.find(x=>x.id==='operations')??screens[0];
let account='ACC-000001',snapshot=1,expired=false,mode='normal',view='table';
let q,rows=[],result=null,cursors=[null],page=0,loading=false,controller,generation=0;
let selected=new Set(),excluded=new Set(),allSelected=false,hidden=new Set(),searchTimer;
let drafts=new Map(),completedSteps=new Set();
const jobs=createJobs(()=>{renderJobs(locale,jobs,scopeKey());const count=$('#job-count');if(count)count.textContent=String(jobs.items.filter(job=>job.scope===scopeKey()).length);});
const tr=key=>t(key,locale);
const text=value=>esc(label(value,locale));
const number=value=>new Intl.NumberFormat(locale==='ar'?'ar-AE':locale==='ko'?'ko-KR':'en-US').format(value);
const scopeKey=()=>`${screen.group==='customer'?'customer':'operator'}:${account}`;
const activeGroup=()=>groups.find(group=>group.id===screen.group);
const domains=[
 {id:'overview',groups:['overview'],label:{en:'Overview',ko:'전체 현황',ar:'نظرة عامة'}},
 {id:'work',groups:['work','issue','history'],label:{en:'Billing operations',ko:'청구 운영',ar:'عمليات الفوترة'}},
 {id:'sources',groups:['sources'],label:{en:'Sources & evidence',ko:'원본·증빙',ar:'المصادر والأدلة'}},
 {id:'contracts',groups:['contracts','reconcile'],label:{en:'Pricing & calculation',ko:'요율·계산',ar:'التسعير والحساب'}},
 {id:'customer',groups:['customer'],label:{en:'Customer portal',ko:'고객 포털',ar:'بوابة العملاء'}},
 {id:'settings',groups:['settings','access','onboarding'],label:{en:'Administration',ko:'관리 설정',ar:'الإدارة'}}
];
const activeDomain=()=>domains.find(domain=>domain.groups.includes(screen.group));
const link=id=>`?lang=${locale}#${id}`;
const selectedRow=id=>allSelected?!excluded.has(id):selected.has(id);
const hasSelection=()=>allSelected?(result?.total==null||result.total>excluded.size):selected.size>0;
const countSelection=()=>allSelected?(result?.total==null?tr('unknown'):number(Math.max(0,result.total-excluded.size))):number(selected.size);
function resetQuery(){
 q={screen:screen.id,account,search:'',provider:'',status:'',sort:'asc',size:25,snapshot};
 cursors=[null];page=0;result=null;rows=[];selected.clear();excluded.clear();allSelected=false;hidden.clear();
}
function invalidate(){
 const hadSelection=selected.size||allSelected;
 selected.clear();excluded.clear();allSelected=false;cursors=[null];page=0;result=null;
 closeDialog();
 if(hadSelection)toast(tr('selectionReset'));
}
function changeQuery(key,value){
 q[key]=value;invalidate();load();
}
function route(id){
 const next=screens.find(item=>item.id===id);
 if(!next)return;
 controller?.abort();clearTimeout(searchTimer);generation++;
 screen=next;expired=false;mode='normal';snapshot++;resetQuery();render();
 $('#main').focus({preventScroll:true});
}
function groupLinks(){
 return domains.map(group=>`<button data-group="${group.id}" class="${group.id===activeDomain().id?'active':''}" aria-pressed="${group.id===activeDomain().id}">${text(group.label)}</button>`).join('');
}
function railLinks(needle=''){
 const visible=screens.filter(item=>needle?`${label(item.title,locale)} ${item.id} ${item.registryIds.join(' ')}`.toLowerCase().includes(needle.toLowerCase()):activeDomain().groups.includes(item.group));
 return visible.map(item=>`<a href="${link(item.id)}" ${item.id===screen.id?'aria-current="page"':''} class="${item.id===screen.id?'active':''}">${text(item.title)}</a>`).join('')||`<p>${tr('empty')}</p>`;
}
function render(){
 document.documentElement.lang=locale==='ar'?'ar-AE':locale;
 document.documentElement.dir=locale==='ar'?'rtl':'ltr';
 document.title=`${label(screen.title,locale)} · Hybrid Billing`;
 $('.skip').textContent=locale==='ko'?'본문 바로가기':locale==='ar'?'انتقل إلى المحتوى':'Skip to content';
 $('#app').innerHTML=`<div class="preview"><span>${tr('preview')}</span><span>${tr('review')}</span></div>
 <header class="top"><a href="${link('overview')}" class="brand" style="text-decoration:none"><span class="brand-mark">H</span>Hybrid Billing</a><div class="top-actions"><button id="choose-account" class="account-button">${tr('account')} · <bdi>${account}</bdi> ▾</button><button id="open-jobs">${tr('jobs')} <span class="tag" id="job-count">${jobs.items.filter(job=>job.scope===scopeKey()).length}</span></button><label><span class="sr-only">Language</span><select id="language" aria-label="Language"><option value="en" ${locale==='en'?'selected':''}>English (US)</option><option value="ar" ${locale==='ar'?'selected':''}>العربية (الإمارات)</option><option value="ko" ${locale==='ko'?'selected':''}>한국어</option></select></label></div></header>
 <nav class="tnb" aria-label="${tr('workspace')}">${groupLinks()}</nav>
 <div class="layout"><aside class="rail"><label><span class="sr-only">${tr('screens')}</span><input id="screen-search" type="search" placeholder="${tr('screens')}" autocomplete="off"></label><small>${screens.length} ${tr('allScreens')}</small><nav id="screen-links" aria-label="${tr('allScreens')}">${railLinks()}</nav></aside>
 <main class="main" id="main" tabindex="-1"><div class="crumb">${text(activeGroup()?.label)} / <bdi>${screen.id}</bdi></div><div class="heading"><div><h1>${text(screen.title)}</h1><p>${text(screen.description)}</p></div><span class="tag">${tr('draft')}</span></div><div class="meta"><span>${tr('account')}: <bdi>${account}</bdi></span><span>${tr('snapshot')}: <bdi>S${snapshot}</bdi></span><span>${tr('notConnected')}</span></div><div id="content">${content()}</div><section class="panel"><h2>${tr('related')}</h2><div class="related">${screen.related.map(id=>`<a href="${link(id)}">${text(screens.find(item=>item.id===id)?.title)}</a>`).join('')}</div></section></main></div>`;
 $('#choose-account').onclick=()=>accountPicker(locale,account,value=>{if(value!==account){account=value;snapshot++;expired=false;invalidate();resetQuery();render();$('#choose-account').focus();}});
 $('#open-jobs').onclick=()=>jobsDialog(locale,jobs,scopeKey());
 $('#language').onchange=event=>{
  locale=event.target.value;
  const url=new URL(location.href);url.searchParams.set('lang',locale);history.replaceState(null,'',url);
  closeDialog();render();$('#language').focus();
 };
 document.querySelectorAll('[data-group]').forEach(button=>button.onclick=()=>location.hash=screens.find(item=>item.group===button.dataset.group).id);
 $('#screen-search').oninput=event=>$('#screen-links').innerHTML=railLinks(event.target.value);
 bindContent();
}
function content(){
 if(screen.kind==='overview')return overview();
 if(screen.kind==='form')return form();
 if(screen.kind==='workflow')return workflow();
 if(screen.kind==='detail')return detail();
 return list();
}
function overview(){
 const ids=screen.id==='contents'?screens:screens.filter(item=>item.group!== 'overview').filter((item,index,array)=>array.findIndex(x=>x.group===item.group)===index);
 return `<div class="stats"><article class="stat">${tr('accounts')}<strong>${number(100000)}</strong><p>${tr('pickerHelp')}</p></article><article class="stat">${tr('fileCount')}<strong>${number(1000000)}</strong><p>${tr('bounded')}</p></article><article class="stat">${tr('allScreens')}<strong>${number(screens.length)}</strong><p>English · العربية · 한국어</p></article></div><section class="panel"><h2>${tr('nextTask')}</h2><div class="screen-index">${ids.map(item=>`<a href="${link(item.id)}">${text(item.title)}<small>${text(groups.find(g=>g.id===item.group)?.label)}</small></a>`).join('')}</div></section>`;
}
function form(){
 const draft=drafts.get(`${account}:${screen.id}`)??{};
 return `<section class="panel"><h2>${text(screen.action)}</h2><p>${tr('noActual')}</p><form id="screen-form"><div class="form-grid">${screen.fields.map(field=>{
  const value=draft[field.key]??'';
  return `<label>${text(field.label)} *${field.type==='select'?`<select required name="${esc(field.key)}"><option value="">${tr('choose')}</option>${(field.options??[]).map(option=>`<option value="${esc(option.value)}" ${value===option.value?'selected':''}>${text(option.label)}</option>`).join('')}</select>`:`<input required name="${esc(field.key)}" type="${field.type==='password'?'text':esc(field.type)}" value="${esc(value)}" autocomplete="off" maxlength="160">`}</label>`;
 }).join('')}</div><div class="form-actions"><button class="primary" type="submit">${text(screen.action)}</button><span class="muted" id="form-feedback" role="status"></span></div></form><div class="notice">${tr('preview')}</div></section>`;
}
function workflow(){
 const steps=screen.fields.length?screen.fields:[{key:'scope',label:activeGroup().label}];
 return `<section class="panel"><h2>${text(screen.action)}</h2><p>${tr('noActual')}</p><div class="workflow">${steps.map((field,index)=>{
  const key=`${account}:${screen.id}:${field.key}`;
  const done=completedSteps.has(key);
  return `<article class="step ${done?'done':''}"><span class="step-number">${index+1}</span><div><h3>${text(field.label)}</h3><p>${done?tr('completed'):tr('pending')}</p><button data-step="${esc(key)}" ${done?'disabled':''}>${tr('check')}</button></div></article>`;
 }).join('')}</div><div class="form-actions"><button id="workflow-review" class="primary" ${steps.every(field=>completedSteps.has(`${account}:${screen.id}:${field.key}`))?'':'disabled'}>${text(screen.action)}</button></div></section>`;
}
function detail(){
 return `<section class="panel"><h2>${tr('summary')}</h2><dl class="facts">${screen.columns.map(column=>`<div><dt>${text(column.label)}</dt><dd>${column.key==='account'?account:column.type==='status'?tr('pending'):tr('unknownValue')}</dd></div>`).join('')}</dl><div class="notice">${tr('notConnected')}</div><button id="detail-action" class="primary">${text(screen.action)}</button></section>`;
}
function options(items,current){return items.map(item=>`<option value="${esc(item.value)}" ${current===item.value?'selected':''}>${esc(item.name)}</option>`).join('');}
function list(){
 return `<section class="panel"><div class="toolbar"><label class="grow">${tr('search')}<input id="record-search" type="search" value="${esc(q.search)}" placeholder="REC-0000001" autocomplete="off"></label>${screen.group==='customer'?'':`<label>${tr('provider')}<select id="provider-filter">${options([{value:'',name:tr('all')},...providers.map(value=>({value,name:value}))],q.provider)}</select></label>`}<label>${tr('status')}<select id="status-filter">${options([{value:'',name:tr('all')},...states.map(value=>({value,name:tr(value)}))],q.status)}</select></label><label>${tr('sort')}<select id="sort">${options(['asc','desc'].map(value=>({value,name:tr(value)})),q.sort)}</select></label><button id="refresh">${tr('refresh')}</button></div>
 <div class="toolbar"><button id="table-view" aria-pressed="${view==='table'}">${tr('table')}</button><button id="card-view" aria-pressed="${view==='cards'}">${tr('cards')}</button><button id="column-picker">${tr('columns')}</button><button id="reset-filters">${tr('reset')}</button>${screen.kind==='files'||screen.group==='sources'?`<button id="intake" class="primary">${tr('files')}</button>`:''}</div><div id="bulk-controls"></div><div id="results" aria-live="polite"></div><div class="pager"><div><label>${tr('pageSize')}<select id="page-size">${options([25,50,100].map(value=>({value:String(value),name:String(value)})),String(q.size))}</select></label><span id="record-count" class="muted"></span></div><div><button id="prev">${tr('previous')}</button><span id="page-number"></span><button id="next">${tr('next')}</button></div></div>
 <details class="scenario"><summary>${tr('reviewTools')}</summary><div><button id="scenario-normal">${tr('normal')}</button><button id="scenario-error">${tr('loadError')}</button><button id="scenario-expire">${tr('expire')}</button></div></details></section>`;
}
function bindContent(){
 if($('#screen-form'))$('#screen-form').onsubmit=event=>{
  event.preventDefault();
  const values=Object.fromEntries(new FormData(event.target));
  drafts.set(`${account}:${screen.id}`,values);$('#form-feedback').textContent=tr('saved');toast(tr('saved'));
 };
 document.querySelectorAll('[data-step]').forEach(button=>button.onclick=()=>{completedSteps.add(button.dataset.step);$('#content').innerHTML=content();bindContent();document.querySelector('[data-step]:not(:disabled),#workflow-review:not(:disabled)')?.focus();});
 if($('#workflow-review'))$('#workflow-review').onclick=()=>actionPreview();
 if($('#detail-action'))$('#detail-action').onclick=()=>actionPreview();
 if(!$('#results'))return;
 $('#record-search').oninput=event=>{
  const value=event.target.value;clearTimeout(searchTimer);
  controller?.abort();generation++;q.search=value;invalidate();loading=true;rows=[];renderRows();
  searchTimer=setTimeout(()=>load(),180);
 };
 if($('#provider-filter'))$('#provider-filter').onchange=event=>changeQuery('provider',event.target.value);
 $('#status-filter').onchange=event=>changeQuery('status',event.target.value);
 $('#sort').onchange=event=>changeQuery('sort',event.target.value);
 $('#page-size').onchange=event=>changeQuery('size',Number(event.target.value));
 $('#refresh').onclick=()=>{expired=false;snapshot++;q.snapshot=snapshot;invalidate();render();};
 $('#reset-filters').onclick=()=>{invalidate();resetQuery();render();};
 $('#table-view').onclick=()=>{view='table';render();};
 $('#card-view').onclick=()=>{view='cards';render();};
 $('#column-picker').onclick=()=>columnPicker();
 $('#prev').onclick=()=>{if(page>0&&!loading){page--;load();}};
 $('#next').onclick=()=>{if(result?.next&&!loading){cursors[++page]=result.next;load();}};
 $('#scenario-normal').onclick=()=>{mode='normal';expired=false;load();};
 $('#scenario-error').onclick=()=>{mode='error';load();};
 $('#scenario-expire').onclick=()=>{expired=true;selected.clear();excluded.clear();allSelected=false;closeDialog();renderRows();};
 if($('#intake'))$('#intake').onclick=()=>fileIntake(locale,scopeKey(),jobs);
 load();
}
async function load(){
 controller?.abort();controller=new AbortController();const current=++generation;
 loading=true;rows=[];renderRows();
 try{
  const response=await queryRows(screen,{...q},cursors[page],controller.signal);
  if(current!==generation)return;
  result=response;rows=mode==='error'?[]:response.rows;loading=false;renderRows();
 }catch(error){
  if(error.name==='AbortError'||current!==generation)return;
  loading=false;if(error.message==='expired')expired=true;else mode='error';renderRows();
 }
}
function valueFor(row,column){
 if(column.type==='status')return `<span class="status ${row.status}"><span class="dot"></span>${tr(row.status)}</span>`;
 const value=row[column.key];
 if(value==null)return `<span class="muted">${tr('unknownValue')}</span>`;
 return `<bdi>${esc(value)}</bdi>`;
}
function renderBulk(){
 if(!$('#bulk-controls'))return;
 $('#bulk-controls').innerHTML=`<div class="bulkbar"><div><label class="inline"><input id="page-select" type="checkbox" ${rows.length&&rows.every(row=>selectedRow(row.id))?'checked':''} ${loading||expired||mode==='error'||!rows.length?'disabled':''}>${tr('selectPage')}</label><span>${tr('selected')}: <strong id="selected-count">${countSelection()}</strong>${allSelected?` · ${tr('exceptions')}: ${excluded.size}`:''}</span></div><div><button id="all-select" ${loading||expired||mode==='error'||!rows.length?'disabled':''}>${tr('selectAll')}</button><button id="clear-selection">${tr('clear')}</button><button id="review-selection" class="primary" ${expired||loading||mode==='error'||!hasSelection()?'disabled':''}>${tr('bulk')}</button></div></div>`;
 $('#page-select').onchange=event=>{for(const row of rows){if(allSelected){if(event.target.checked)excluded.delete(row.id);else excluded.add(row.id);}else{if(event.target.checked)selected.add(row.id);else selected.delete(row.id);}}renderRows();};
 $('#all-select').onclick=()=>{allSelected=true;selected.clear();excluded.clear();renderRows();};
 $('#clear-selection').onclick=()=>{selected.clear();excluded.clear();allSelected=false;renderRows();};
 $('#review-selection').onclick=()=>selectionPreview();
 const pageSelect=$('#page-select');
 if(pageSelect)pageSelect.indeterminate=rows.some(row=>selectedRow(row.id))&&!rows.every(row=>selectedRow(row.id));
}
function renderRows(){
 if(!$('#results'))return;
 const focusId=document.activeElement?.id;
 const focusRow=document.activeElement?.dataset?.row;
 renderBulk();
 const cols=screen.columns.filter(column=>!hidden.has(column.key));
 let html='';
 if(expired)html=`<div class="state error"><h2>${tr('expired')}</h2></div>`;
 else if(loading)html=`<div class="state" role="status"><h2>${tr('loading')}</h2></div>`;
 else if(mode==='error')html=`<div class="state error"><h2>${tr('error')}</h2><button id="retry-load">${tr('retry')}</button></div>`;
 else if(!rows.length)html=`<div class="state"><h2>${tr('empty')}</h2><p>${tr('tryFilters')}</p></div>`;
 else if(view==='cards')html=`<div class="cards">${rows.map(row=>`<article class="record-card"><header><input type="checkbox" data-row="${row.id}" aria-label="${tr('selected')} ${row.id}" ${selectedRow(row.id)?'checked':''}><button class="row-link" data-detail="${row.id}"><bdi>${row.id}</bdi></button></header>${cols.map(column=>`<p><span class="muted">${text(column.label)}</span><br>${valueFor(row,column)}</p>`).join('')}</article>`).join('')}</div>`;
 else html=`<div class="table-wrap" tabindex="0" role="region" aria-label="${text(screen.title)}"><table><thead><tr><th>${tr('selected')}</th><th>ID</th>${cols.map(column=>`<th scope="col">${text(column.label)}</th>`).join('')}</tr></thead><tbody>${rows.map(row=>`<tr><td><input type="checkbox" data-row="${row.id}" aria-label="${tr('selected')} ${row.id}" ${selectedRow(row.id)?'checked':''}></td><td><button class="row-link" data-detail="${row.id}"><bdi>${row.id}</bdi></button></td>${cols.map(column=>`<td>${valueFor(row,column)}</td>`).join('')}</tr>`).join('')}</tbody></table></div>`;
 $('#results').innerHTML=html;
 $('#results').setAttribute('aria-busy',String(loading));
 $('#record-count').textContent=loading?tr('loading'):mode==='error'||expired?tr('unknown'):result?.total==null?tr('unknown'):`${number(result.total)} ${tr('records')}`;
 $('#page-number').textContent=number(page+1);
 $('#prev').disabled=loading||expired||mode==='error'||page===0;
 $('#next').disabled=loading||expired||mode==='error'||!result?.next;
 if($('#retry-load'))$('#retry-load').onclick=()=>{mode='normal';load();};
 document.querySelectorAll('[data-row]').forEach(input=>input.onchange=()=>{
  const id=input.dataset.row;
  if(allSelected){if(input.checked)excluded.delete(id);else excluded.add(id);}
  else{if(input.checked)selected.add(id);else selected.delete(id);}
  renderBulk();
 });
 document.querySelectorAll('[data-detail]').forEach(button=>button.onclick=()=>rowDetail(rows.find(row=>row.id===button.dataset.detail)));
 if(focusId&&document.getElementById(focusId))document.getElementById(focusId).focus({preventScroll:true});
 else if(focusRow)document.querySelector(`[data-row="${focusRow}"]`)?.focus({preventScroll:true});
}
function columnPicker(){
 const dialog=modal(tr('columns'),`<div class="column-options">${screen.columns.map(column=>`<label><input type="checkbox" data-column="${esc(column.key)}" ${hidden.has(column.key)?'':'checked'}>${text(column.label)}</label>`).join('')}</div>`,locale);
 dialog.querySelectorAll('[data-column]').forEach(input=>input.onchange=()=>{if(input.checked)hidden.delete(input.dataset.column);else hidden.add(input.dataset.column);renderRows();});
}
function rowDetail(row){
 const safeColumns=screen.columns.filter(column=>screen.group!=='customer'||!['provider','cost','margin','supplier','revision'].includes(column.key));
 modal(`${tr('details')} · ${row.id}`,`<p>${tr('account')}: <bdi>${row.account}</bdi> · ${tr('snapshot')}: S${snapshot}</p><dl class="facts">${safeColumns.map(column=>`<div><dt>${text(column.label)}</dt><dd>${valueFor(row,column)}</dd></div>`).join('')}</dl><div class="notice">${tr('noActual')}</div><div class="related">${screen.related.map(id=>`<a href="${link(id)}">${text(screens.find(item=>item.id===id)?.title)}</a>`).join('')}</div>`,locale);
}
function actionPreview(){
 modal(label(screen.action,locale),`<p>${tr('noActual')}</p><div class="notice">${tr('notConnected')}</div><div class="related">${screen.related.map(id=>`<a href="${link(id)}">${text(screens.find(item=>item.id===id)?.title)}</a>`).join('')}</div>`,locale);
}
function selectionPreview(){
 if(expired||loading||!hasSelection())return;
 const pin=signature(q);
 const descriptor=allSelected?{query:{...q},excluded:[...excluded].sort()}:{query:{...q},ids:[...selected].sort()};
 const fingerprint=JSON.stringify(descriptor);
 const dialog=modal(tr('bulk'),`<p>${tr('selectionNote')}</p><dl class="facts"><div><dt>${tr('account')}</dt><dd><bdi>${account}</bdi></dd></div><div><dt>${tr('snapshot')}</dt><dd>S${snapshot}</dd></div><div><dt>${tr('scope')}</dt><dd>${allSelected?tr('filterMatches'):tr('explicit')}</dd></div><div><dt>${tr('selected')}</dt><dd>${countSelection()}</dd></div><div><dt>${tr('search')}</dt><dd>${esc(q.search)||tr('all')}</dd></div><div><dt>${tr('provider')} / ${tr('status')}</dt><dd>${esc(q.provider)||tr('all')} / ${q.status?tr(q.status):tr('all')}</dd></div></dl><div class="notice">${tr('jobNote')}</div><div class="dialog-actions"><button id="queue-selection" class="primary">${tr('queue')}</button></div>`,locale);
 dialog.querySelector('#queue-selection').onclick=event=>{
  if(expired||!hasSelection()||signature(q)!==pin){closeDialog();toast(tr('expired'));return;}
  event.target.disabled=true;
  const outcome=jobs.add(label(screen.action,locale),`${scopeKey()}:${fingerprint}`,scopeKey());
  closeDialog();
  if(outcome!=='added')toast(tr(outcome==='duplicate'?'jobDuplicate':'jobLimit'));
  jobsDialog(locale,jobs,scopeKey());
 };
}
window.addEventListener('hashchange',()=>{closeDialog();route(location.hash.slice(1));});
resetQuery();render();
