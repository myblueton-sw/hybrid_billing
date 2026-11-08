import {esc} from './widgets.js';
import {label} from './copy.js';
import {reviewStatusMap} from './design-review.js';
import {reviewText} from './review-copy.js';
const copy={
 title:['Screen design library','화면 설계 목록','مكتبة تصميم الشاشات'],
 search:['Find a screen by name or ID','이름·ID로 화면 찾기','البحث بالاسم أو المعرّف'],
 group:['Product area','업무 영역','مجال المنتج'],
 all:['All','전체','الكل'],
 review:['Review status','검수 상태','حالة المراجعة'],
 records:['Screens with local review evidence','로컬 검수 기록이 있는 화면','شاشات لها أدلة مراجعة محلية'],
 total:['Screen designs','화면 시안','تصاميم الشاشات'],
 previous:['Previous','이전','السابق'],
 next:['Next','다음','التالي'],
 pending:['Unreviewed current variant','현재 표시 조건 미검수','نسخة العرض الحالية غير مراجعة'],
 reviewed:['Local review records','로컬 검수 기록','سجلات مراجعة محلية'],
 changed:['Impact assessment pending','변경 영향 검토 대기','تقييم الأثر معلق'],
 empty:['No screen matches these filters.','조건에 맞는 화면이 없습니다.','لا توجد شاشة تطابق المرشحات.'],
 guidance:['Choose one screen, inspect its item IDs and visible states, then confirm only the scope you reviewed. No screen has been approved by the assistant.','화면을 선택해 구현 항목 ID와 상태를 검토한 뒤, 확인한 범위만 검수하세요. 어시스턴트가 대신 확정한 화면은 없습니다.','اختر شاشة وراجع معرّفات عناصرها وحالاتها ثم أكد النطاق الذي راجعته فقط. لم يعتمد المساعد أي شاشة.']
};
export function designDirectory(screens,groups,locale,link){
 const t=key=>copy[key]?.[['en','ko','ar'].indexOf(locale)]??key;
 let search='',group='',status='',page=0,statuses=reviewStatusMap(screens,locale);
 const hasRecord=item=>!['draft','error'].includes(statuses[item.id]);
 function entries(){return screens.filter(item=>(!search||`${label(item.title,locale)} ${item.id} ${item.registryIds.join(' ')}`.toLowerCase().includes(search.toLowerCase()))&&(!group||item.group===group)&&(!status||(status==='reviewed'?hasRecord(item):status==='changed'?statuses[item.id]==='changed':['draft','otherVariant','error'].includes(statuses[item.id]))));}
 function update(){
  statuses=reviewStatusMap(screens,locale);
  document.querySelector('#directory-review-count').textContent=String(screens.filter(hasRecord).length);
  const matches=entries();page=Math.min(page,Math.max(0,Math.ceil(matches.length/24)-1));
  document.querySelector('#design-directory-results').innerHTML=matches.length?`<div class="screen-index">${matches.slice(page*24,(page+1)*24).map(item=>`<a href="${link(item.id)}"><strong>${esc(label(item.title,locale))}</strong><small><bdi>${item.id}</bdi> · ${esc(label(groups.find(group=>group.id===item.group)?.label,locale))}</small><small class="review-status" data-review-bookkeeping data-review-status="${item.id}">${reviewText(statuses[item.id],locale)}</small></a>`).join('')}</div>`:`<p>${t('empty')}</p>`;
  document.querySelector('#design-directory-count').textContent=`${matches.length} · ${page+1} / ${Math.max(1,Math.ceil(matches.length/24))}`;
  document.querySelector('#design-directory-prev').disabled=page===0;document.querySelector('#design-directory-next').disabled=(page+1)*24>=matches.length;
 }
 function bind(){
  document.querySelector('#design-directory-search').oninput=event=>{search=event.target.value;page=0;update();};
  document.querySelector('#design-directory-group').onchange=event=>{group=event.target.value;page=0;update();};
  document.querySelector('#design-directory-status').onchange=event=>{status=event.target.value;page=0;update();};
  document.querySelector('#design-directory-prev').onclick=()=>{page--;update();};
  document.querySelector('#design-directory-next').onclick=()=>{page++;update();};
  update();
 }
 return {html:`<div class="stats"><article class="stat">${t('total')}<strong>${screens.length}</strong></article><article class="stat">${t('records')}<strong id="directory-review-count" data-review-bookkeeping>${screens.filter(hasRecord).length}</strong></article></div><section class="panel"><h2>${t('title')}</h2><p>${t('guidance')}</p><div class="review-index-toolbar"><label class="grow">${t('search')}<input id="design-directory-search" type="search"></label><label>${t('group')}<select id="design-directory-group"><option value="">${t('all')}</option>${groups.map(group=>`<option value="${group.id}">${esc(label(group.label,locale))}</option>`).join('')}</select></label><label>${t('review')}<select id="design-directory-status"><option value="">${t('all')}</option>${['pending','reviewed','changed'].map(key=>`<option value="${key}">${t(key)}</option>`).join('')}</select></label></div><div id="design-directory-results" aria-live="polite"></div><div class="pager"><span id="design-directory-count"></span><div><button id="design-directory-prev">${t('previous')}</button><button id="design-directory-next">${t('next')}</button></div></div></section>`,bind,refresh:update};
}
