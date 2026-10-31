import {t} from './copy.js';
import {accountPage} from './data.js';
export const esc=value=>String(value??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
let returnFocus;
export function closeDialog(){
 const dialog=document.querySelector('#dialog');
 if(!dialog.open)return;
 dialog.close();
 if(returnFocus?.isConnected)returnFocus.focus();
}
export function modal(title,body,locale){
 const dialog=document.querySelector('#dialog');
 if(!dialog.open)returnFocus=document.activeElement;
 dialog.innerHTML=`<header><h2 id="dialog-title">${esc(title)}</h2><button id="close-dialog" aria-label="${t('close',locale)}">×</button></header>${body}`;
 dialog.setAttribute('aria-labelledby','dialog-title');
 dialog.querySelector('#close-dialog').onclick=closeDialog;
 dialog.oncancel=event=>{event.preventDefault();closeDialog();};
 if(!dialog.open)dialog.showModal();
 else dialog.querySelector('#close-dialog').focus();
 return dialog;
}
let toastTimer;
export function toast(message){
 const node=document.querySelector('#toast');
 node.textContent=message;node.style.display='block';
 clearTimeout(toastTimer);toastTimer=setTimeout(()=>node.style.display='none',4500);
}
export function accountPicker(locale,current,onApply){
 let page=0,search='',chosen=current;
 const dialog=modal(t('choose',locale),`<p id="picker-help">${t('pickerHelp',locale)}</p><label>${t('search',locale)}<input id="account-search" autocomplete="off" aria-describedby="picker-help"></label><div id="account-options" class="picker-list" role="listbox" aria-label="${t('choose',locale)}"></div><div class="pager"><button id="account-prev">${t('previous',locale)}</button><span id="account-page"></span><button id="account-next">${t('next',locale)}</button></div><div class="dialog-actions"><button id="account-apply" class="primary">${t('apply',locale)}</button></div>`,locale);
 function render(){
  const result=accountPage(search,page);
  const list=dialog.querySelector('#account-options');
  list.innerHTML=result.items.map((item,i)=>`<button type="button" role="option" tabindex="${i===0?0:-1}" aria-selected="${item.id===chosen}" data-account="${item.id}"><span>${esc(item.name)}</span><bdi>${item.id}</bdi></button>`).join('')||`<p>${t('empty',locale)}</p>`;
  list.querySelectorAll('[data-account]').forEach(button=>{
   button.onclick=()=>{chosen=button.dataset.account;list.querySelectorAll('[role=option]').forEach(option=>{option.setAttribute('aria-selected',String(option===button));option.tabIndex=option===button?0:-1;});};
   button.onkeydown=event=>{
    const options=[...list.querySelectorAll('[role=option]')];
    let index=options.indexOf(button);
    if(event.key==='ArrowDown')index=Math.min(options.length-1,index+1);
    else if(event.key==='ArrowUp')index=Math.max(0,index-1);
    else if(event.key==='Home')index=0;
    else if(event.key==='End')index=options.length-1;
    else return;
    event.preventDefault();options.forEach((option,i)=>option.tabIndex=i===index?0:-1);options[index]?.focus();
   };
  });
  dialog.querySelector('#account-prev').disabled=page===0;
  dialog.querySelector('#account-next').disabled=!result.more;
  dialog.querySelector('#account-page').textContent=String(page+1);
 }
 dialog.querySelector('#account-search').oninput=event=>{search=event.target.value;page=0;render();};
 dialog.querySelector('#account-prev').onclick=()=>{page--;render();};
 dialog.querySelector('#account-next').onclick=()=>{page++;render();};
 dialog.querySelector('#account-apply').onclick=()=>{closeDialog();onApply(chosen);};
 render();dialog.querySelector('#account-search').focus();
}
export function createJobs(onChange){
 const items=[];
 const fingerprints=new Set();
 function add(title,fingerprint,scope){
  if(fingerprints.has(fingerprint))return 'duplicate';
  if(items.length>=100)return 'limit';
  fingerprints.add(fingerprint);
  items.unshift({id:'JOB-'+String(items.length+1).padStart(4,'0'),title,scope,status:'queued',progress:0,attempt:0});
  onChange();return 'added';
 }
 function cancel(id,scope){const job=items.find(x=>x.id===id&&x.scope===scope);if(job&&['queued','processing'].includes(job.status)){job.status='cancelled';onChange();}}
 function retry(id,scope){const job=items.find(x=>x.id===id&&x.scope===scope);if(job?.status==='failed'){job.status='queued';job.progress=0;job.attempt++;onChange();}}
 setInterval(()=>{
  let changed=false;
  let active=items.filter(x=>x.status==='processing').length;
  for(const job of items){
   if(job.status==='queued'&&active<2){job.status='processing';active++;changed=true;}
   if(job.status==='processing'){
    job.progress=Math.min(100,job.progress+20);changed=true;
    if(job.progress===100)job.status=Number(job.id.slice(4))%3===0&&job.attempt===0?'failed':'completed';
   }
  }
  if(changed)onChange();
 },750);
 return {items,add,cancel,retry};
}
export function jobsDialog(locale,jobs,scope){
 const dialog=modal(t('jobs',locale),`<p>${t('jobNote',locale)}</p><div class="jobs-list" id="jobs-list"></div>`,locale);
 renderJobs(locale,jobs,scope,dialog);
}
export function renderJobs(locale,jobs,scope,dialog=document.querySelector('#dialog')){
 const host=dialog.querySelector('#jobs-list');
 if(!host)return;
 const focused=document.activeElement?.dataset;
 host.innerHTML=jobs.items.filter(job=>job.scope===scope).map(job=>`<article class="job" data-job="${job.id}" tabindex="-1"><header><bdi>${job.id}</bdi><span class="status ${job.status}">${t(job.status,locale)}</span></header><p>${esc(job.title)} · <bdi>${esc(job.scope)}</bdi></p><progress value="${job.progress}" max="100" aria-label="${t('progress',locale)} ${job.id}"></progress>${['queued','processing'].includes(job.status)?`<button data-cancel="${job.id}">${t('cancel',locale)}</button>`:''}${job.status==='failed'?`<button data-retry="${job.id}">${t('retry',locale)}</button>`:''}</article>`).join('')||`<p>${t('noJobs',locale)}</p>`;
 host.querySelectorAll('[data-cancel]').forEach(button=>button.onclick=()=>jobs.cancel(button.dataset.cancel,scope));
 host.querySelectorAll('[data-retry]').forEach(button=>button.onclick=()=>jobs.retry(button.dataset.retry,scope));
 const jobId=focused?.cancel??focused?.retry??focused?.job;
 if(jobId){
  const control=focused.cancel?host.querySelector(`[data-cancel="${jobId}"]`):focused.retry?host.querySelector(`[data-retry="${jobId}"]`):null;
  (control??host.querySelector(`[data-job="${jobId}"]`))?.focus({preventScroll:true});
 }
}
export function fileIntake(locale,scope,jobs){
 let descriptors=[],page=0,sequence=0;
 const dialog=modal(t('files',locale),`<p>${t('fileNote',locale)}</p><div class="file-drop"><label>${t('addFiles',locale)}<input type="file" id="file-input" multiple></label><button id="demo-files">${t('demoFiles',locale)}</button></div><div id="file-list"></div><div class="pager"><button id="file-prev">${t('previous',locale)}</button><span id="file-page"></span><button id="file-next">${t('next',locale)}</button></div><div class="dialog-actions"><button id="file-jobs">${t('jobs',locale)}</button><button id="run-files" class="primary">${t('runFiles',locale)}</button></div>`,locale);
 function render(){
  dialog.querySelector('#file-list').innerHTML=`<p>${t('count',locale)}: ${descriptors.length} / 100</p><div class="table-wrap"><table><thead><tr><th>${t('name',locale)}</th><th>${t('size',locale)}</th><th>${t('status',locale)}</th></tr></thead><tbody>${descriptors.slice(page*25,page*25+25).map(file=>`<tr><td>${esc(file.name)}</td><td><bdi>${(file.size/1073741824).toFixed(2)} GiB</bdi></td><td>${t(jobs.items.find(job=>job.id===file.jobId)?.status??'draft',locale)}</td></tr>`).join('')}</tbody></table></div>`;
  dialog.querySelector('#file-page').textContent=`${page+1} / ${Math.max(1,Math.ceil(descriptors.length/25))}`;
  dialog.querySelector('#file-prev').disabled=page===0;
  dialog.querySelector('#file-next').disabled=(page+1)*25>=descriptors.length;
  dialog.querySelector('#run-files').disabled=!descriptors.some(file=>!file.queued);
 }
 function add(files){
  if(files.length+descriptors.length>100){toast(t('fileLimit',locale));return;}
  for(const file of files)descriptors.push({id:++sequence,name:file.name,size:file.size,queued:false});
  render();toast(t('fileAdded',locale));
 }
 dialog.querySelector('#file-input').onchange=event=>{
  const files=event.target.files;
  if(files.length+descriptors.length>100)toast(t('fileLimit',locale));
  else add(Array.from(files,file=>({name:file.name,size:file.size})));
  event.target.value='';
 };
 dialog.querySelector('#demo-files').onclick=()=>add(Array.from({length:30},(_,i)=>({name:`usage-${sequence+i+1}.parquet`,size:(i+1)*1073741824*5})));
 dialog.querySelector('#file-prev').onclick=()=>{page--;render();};
 dialog.querySelector('#file-next').onclick=()=>{page++;render();};
 const batch=crypto.randomUUID();
 dialog.querySelector('#run-files').onclick=()=>{
  let limit=false;
  for(const file of descriptors.filter(x=>!x.queued)){
   const outcome=jobs.add(file.name,`${scope}:${batch}:${file.id}`,scope);
   file.queued=outcome==='added'||outcome==='duplicate';
   if(outcome==='added')file.jobId=jobs.items[0].id;
   if(outcome==='limit')limit=true;
  }
  render();toast(t(limit?'jobLimit':'jobNote',locale));
 };
 dialog.querySelector('#file-jobs').onclick=()=>jobsDialog(locale,jobs,scope);
 const timer=setInterval(()=>{if(!dialog.open||!dialog.querySelector('#file-list'))clearInterval(timer);else render();},750);
 dialog.addEventListener('close',()=>clearInterval(timer),{once:true});
 render();
}
