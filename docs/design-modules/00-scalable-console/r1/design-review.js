import {esc,modal,toast} from './widgets.js';
import {label} from './copy.js';
import {readReviews,saveReview,reviewStatus} from './review-store.js';
import {reviewText} from './review-copy.js';
let revisions={},revisionError=false,observer,timer,active=null;
const layout=()=>innerWidth<640?'compact':innerWidth<1024?'medium':'wide';
const storage={getItem:key=>localStorage.getItem(key),setItem:(key,value)=>localStorage.setItem(key,value)};
export async function loadDesignRevisions(){
 try{const response=await fetch('./design-revisions.json',{cache:'no-store'});if(!response.ok)throw Error('revision');const data=await response.json();if(data.version!==1||!data.screens||Object.values(data.screens).some(item=>!/^[a-f0-9]{64}$/.test(item.revision)))throw Error('revision');revisions=data.screens;revisionError=false;}
 catch{revisions={};revisionError=true;}
}
export function screenReviewStatus(screen,locale){
 const data=readReviews(storage);
 return data.error?'error':reviewStatus(data.records,screen,revisions[screen]?.revision,locale,layout());
}
export function reviewStatusMap(screens,locale){
 const data=readReviews(storage);return Object.fromEntries(screens.map(screen=>[screen.id,data.error?'error':reviewStatus(data.records,screen.id,revisions[screen.id]?.revision,locale,layout())]));
}
export function reviewBar(screen,locale){
 const t=key=>reviewText(key,locale),revision=revisions[screen.id]?.revision;
 return `<section class="design-review" aria-label="${t('title')}"><div class="review-heading"><div><strong>${t('title')}</strong><p><bdi>${esc(screen.id)}</bdi> · ${t('revision')} <bdi title="${esc(revision??'')}">${revision?.slice(0,12)??'—'}</bdi></p></div><span id="design-status" class="tag">${t(screenReviewStatus(screen.id,locale))}</span><button id="confirm-screen-review" ${revision?'':'disabled'}>${t('confirm')}</button></div><p class="muted">${t('local')}</p>${revisionError?`<p role="alert">${t('missing')}</p>`:''}<details id="implementation-items"><summary>${t('checklist')} <span id="design-item-count"></span></summary><div id="design-items"></div></details><p id="design-storage-message" role="status"></p><button id="export-screen-reviews">${t('export')}</button></section>`;
}
function semanticKey(node){
 if(node.dataset.designId)return node.dataset.designId;
 if(node.name)return `field:${node.name}`;
 if(node.id)return `${node.matches('input,select,textarea')?'field':'action'}:${node.id}`;
 if(node.dataset.row)return 'field:row-selection';
 if(node.dataset.detail)return 'action:record-detail';
 if(node.dataset.step)return `action:step-${node.dataset.step.split(':').at(-1)}`;
 if(node.dataset.setupRoute)return `navigate:${node.dataset.setupRoute}`;
 if(node.tagName==='A'){const hash=new URL(node.href).hash.slice(1);if(hash)return `navigate:${hash}`;}
 if(node.type==='submit')return `action:${node.form?.id??'form'}-submit`;
 return null;
}
function inventory(screen,locale){
 const host=document.querySelector('#content'),items=new Map();
 if(!host)return [];
 const add=(id,title,node)=>{if(!items.has(id))items.set(id,{id,title,node});};
 add(`${screen.id}:region:layout`,reviewText('current',locale),host);
 host.querySelectorAll('[data-design-id],input,select,textarea,button,a[href]').forEach(node=>{
  if(node.type==='hidden'||!node.getClientRects().length)return;
  const key=semanticKey(node);if(!key)return;
  const id=key.startsWith(screen.id+':')?key:`${screen.id}:${key.replace(/[^a-zA-Z0-9:_-]/g,'-')}`;
  const title=node.getAttribute('aria-label')||node.labels?.[0]?.textContent.split('\n')[0]||node.textContent||node.name||node.id;
  add(id,title.trim().slice(0,120)||id,node);
 });
 for(const column of screen.columns){
  const name=label(column.label,locale);
  const node=[...host.querySelectorAll('th,dt')].find(node=>node.textContent.trim()===name);
  if(node)add(`${screen.id}:data:${column.key}`,name,node);
 }
 return [...items.values()].sort((a,b)=>a.id.localeCompare(b.id));
}
function stateValue(){
 const host=document.querySelector('#content');
 const values=[...host.querySelectorAll('input,select,textarea,button,a[href]')].filter(node=>node.getClientRects().length).map(node=>[semanticKey(node),node.type,node.value,node.checked,node.disabled,node.getAttribute('aria-pressed'),node.getAttribute('aria-expanded'),node.getAttribute('href')]);
 const walker=document.createTreeWalker(host,NodeFilter.SHOW_TEXT),text=[];
 let node;while((node=walker.nextNode()))if(node.parentElement.getClientRects().length&&!node.parentElement.closest('[data-review-bookkeeping]'))text.push(node.textContent.trim());
 return JSON.stringify([text,values]);
}
async function stateDigest(value=stateValue()){
 const bytes=new TextEncoder().encode(value);
 return [...new Uint8Array(await crypto.subtle.digest('SHA-256',bytes))].map(value=>value.toString(16).padStart(2,'0')).join('');
}
function downloadRecords(locale){
 const data=readReviews(storage),t=key=>reviewText(key,locale);
 if(data.error){document.querySelector('#design-storage-message').textContent=t('error');return;}
 if(!data.records.length){toast(t('empty'));return;}
 const payload={version:1,kind:'local-unauthenticated-screen-review',ticket:'HB-63',exportedAt:new Date().toISOString(),records:data.records};
 const url=URL.createObjectURL(new Blob([JSON.stringify(payload,null,2)+'\n'],{type:'application/json'}));
 const anchor=document.createElement('a');anchor.href=url;anchor.download='hybrid-billing-screen-reviews.json';anchor.click();setTimeout(()=>URL.revokeObjectURL(url),1000);
}
export function bindDesignReview(screen,locale,onStatusChange,declaredItems=[]){
 observer?.disconnect();clearTimeout(timer);
 const identity={screen,locale};active=identity;let refreshGeneration=0;
 const t=key=>reviewText(key,locale);
 async function refresh(){
  if(active!==identity)return;
  const generation=++refreshGeneration,currentWidth=innerWidth;
  const items=inventory(screen,locale).map(item=>({...item,metadata:declaredItems.find(entry=>entry.id===item.id)})),container=document.querySelector('#design-items');
  if(!container)return;
  document.querySelector('#design-item-count').textContent=`(${items.length})`;
  container.innerHTML=`<p>${t('preview')}</p><ol>${items.map(item=>`<li><button data-inspect-item="${esc(item.id)}">${esc(item.title)}</button><code>${esc(item.id)}</code>${item.metadata?`<p>${esc(item.metadata.behavior??item.metadata.expected)} · ${t('preview')}</p>`:''}</li>`).join('')}</ol>`;
  container.querySelectorAll('[data-inspect-item]').forEach(button=>button.onclick=()=>{
   const item=items.find(item=>item.id===button.dataset.inspectItem);item.node.scrollIntoView({block:'center',behavior:'auto'});item.node.classList.add('design-highlight');setTimeout(()=>item.node.classList.remove('design-highlight'),1200);
  });
  document.querySelector('#design-status').textContent=t(screenReviewStatus(screen.id,locale));
  if(crypto?.subtle){
   const state=await stateDigest();
   if(active!==identity||generation!==refreshGeneration||currentWidth!==innerWidth)return;
   document.querySelector('#design-status').textContent=t(reviewStatus(readReviews(storage).records,screen.id,revisions[screen.id]?.revision,locale,layout(),{width:currentWidth,state,items:items.map(item=>item.id)}));
  }
  const data=readReviews(storage);
  document.querySelector('#design-storage-message').textContent=data.error?t('error'):screenReviewStatus(screen.id,locale)==='changed'?t('assessment'):'';
 }
 identity.refresh=refresh;identity.onStatusChange=onStatusChange;
 document.querySelector('#content').addEventListener('input',refresh);
 document.querySelector('#content').addEventListener('change',refresh);
 observer=new MutationObserver(()=>{clearTimeout(timer);timer=setTimeout(refresh,50);});
 observer.observe(document.querySelector('#content'),{subtree:true,childList:true,attributes:true,attributeFilter:['disabled','hidden','aria-pressed','aria-expanded','href']});
 refresh();
 document.querySelector('#export-screen-reviews').onclick=()=>downloadRecords(locale);
 document.querySelector('#confirm-screen-review').onclick=async()=>{
  const revision=revisions[screen.id]?.revision;
  if(!crypto?.subtle||!revision||readReviews(storage).error){document.querySelector('#design-storage-message').textContent=t('error');return;}
  const items=inventory(screen,locale),initialState=stateValue(),pin={revision,layout:layout(),width:innerWidth,state:await stateDigest(initialState),ids:items.map(item=>item.id)};
  if(active!==identity)return;
  if(stateValue()!==initialState||innerWidth!==pin.width){document.querySelector('#design-storage-message').textContent=t('stale');return;}
  const dialog=modal(t('confirm'),`<p><bdi>${esc(screen.id)}</bdi> · ${t('revision')} <bdi>${revision.slice(0,12)}</bdi></p><h3>${t('scope')}</h3><p><bdi>${locale}</bdi> · ${t(pin.layout)} · <bdi>${pin.width}px</bdi></p><p>${t('state')}</p><p class="notice">${t('deferred')}</p><details><summary>${t('checklist')} (${items.length})</summary><ul>${items.map(item=>`<li><code>${esc(item.id)}</code> — ${esc(item.title)}</li>`).join('')}</ul></details><p>${t('local')}</p><label class="inline"><input id="owner-review-ack" type="checkbox">${t('acknowledge')}</label><p id="review-confirm-message" role="alert"></p><div class="dialog-actions"><button id="save-screen-fix" class="primary" disabled>${t('save')}</button></div>`,locale);
  const save=dialog.querySelector('#save-screen-fix'),ack=dialog.querySelector('#owner-review-ack'),message=dialog.querySelector('#review-confirm-message');
  ack.onchange=()=>{save.disabled=!ack.checked;};
  save.onclick=async()=>{
   if(!ack.checked)return;save.disabled=true;
   let currentRevision;
   try{const response=await fetch('./design-revisions.json',{cache:'no-store'});if(!response.ok)throw Error('revision');currentRevision=(await response.json()).screens[screen.id]?.revision;}catch{message.textContent=t('missing');return;}
   const currentValue=stateValue(),currentState=await stateDigest(currentValue);
   if(!ack.checked||!ack.isConnected||!dialog.open||dialog.querySelector('#save-screen-fix')!==save||stateValue()!==currentValue||active!==identity||layout()!==pin.layout||innerWidth!==pin.width||currentRevision!==pin.revision||currentState!==pin.state||JSON.stringify(inventory(screen,locale).map(item=>item.id))!==JSON.stringify(pin.ids)){message.textContent=t('stale');return;}
   const result=saveReview(storage,{screen:screen.id,revision:pin.revision,locale,layout:pin.layout,width:pin.width,state:pin.state,items:pin.ids,confirmedAt:new Date().toISOString(),source:'local-unauthenticated-confirmation'});
   if(!result.ok){message.textContent=t('error');return;}
   ack.disabled=true;message.textContent=t('saved');refresh();onStatusChange();
  };
 };
}

window.addEventListener('resize',()=>active?.refresh());
window.addEventListener('storage',()=>{active?.onStatusChange();active?.refresh();});
