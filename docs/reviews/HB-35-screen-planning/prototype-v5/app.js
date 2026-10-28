(()=>{
 const h=window.hb5,{tr,esc,state,company,record,users,msps,signature,stamp,views,select,link}=h;

 function render(){
  const [route='register',id]=location.hash.slice(1).split('/');state.route=route;
  if(route==='register'&&id&&company(id)){state.company=id;state.msp=company(id).msp;}
  if(['document','process'].includes(route)&&record(id)){state.selected=id;state.company=record(id).company;state.msp=company(state.company).msp;}
  if(['customer','contracts'].includes(route)&&company(id)){state.company=id;state.msp=company(id).msp;}
  if(route==='policy'&&msps[id]&&state.msp!==id){state.msp=id;state.company='all';}
  const context=company(state.company)||h.companies.find(c=>c.msp===state.msp);
  const candidates=state.records.filter(r=>r.company===context.id);
  const current=candidates.find(r=>r.id===state.selected)||candidates.find(r=>r.status==='draft')||candidates[0];
  const routes=[['register',tr('Monthly issuance','월별 발행 이력')],['customer/'+context.id,tr('Customer details','고객 상세')],['contracts/'+context.id,tr('Contracts and renewal','계약·갱신')],['document/'+current.id,tr('Document and history','문서·처리 이력')],['policy/'+state.msp,tr('Assignment policy','담당자 지정 정책')],['process/'+current.id,tr('Review and issue','검수·발행 처리')],['erp',tr('ERP connection / mapping','ERP 연결·매핑')]];
  document.querySelector('#sidebar').innerHTML=`<div class="brand">Hybrid Billing</div><p>${tr('Billing operations','빌링 운영')}</p><nav>${routes.map(([r,t])=>`<a href="#${r}" ${r.split('/')[0]===route?'aria-current="page"':''}>${t}</a>`).join('')}</nav><a class="prior" href="previous/index.html">${tr('Monthly close / supplier PDF review','월 마감 / 공급자 PDF 검토')} ↗</a><a class="prior" href="previous/customer.html">${tr('Separate customer portal','별도 고객 포털')} ↗</a>`;
  document.querySelector('#top').innerHTML=`<span>${tr('BILLING WORKSPACE / v5 · SYNTHETIC','빌링 작업 공간 / v5 · 합성 데이터')}</span>${select('sessionUser',tr('Simulated signed-in user','시연 로그인 사용자'),users.map(u=>[u.id,u.name]),state.user)}`;
  document.querySelector('#workspace').innerHTML=(views[route]?views[route](id):h.note(tr('Unknown screen.','존재하지 않는 화면입니다.')))+`<div id="feedback" role="status" aria-live="polite">${esc(state.message)}</div>`;
  document.querySelector('#footer').innerHTML=`<span>${tr('Local design only · reload resets edits · no backend, authentication or real issuance','로컬 시안 · 새로고침 시 초기화 · 실제 백엔드·인증·발행 미연결')}</span><span>HB-35 / v5</span>`;
  document.querySelectorAll('form').forEach(f=>f.addEventListener('submit',e=>e.preventDefault()));
 }
 const value=name=>document.querySelector(`[name="${name}"]`)?.value;
 const say=(en,kr)=>{state.message=tr(en,kr);render();};
 const invalidate=(r,why)=>{if(r.review){r.events.push({kind:'review-invalidated',...stamp(),detail:why});r.review=null;}r.revision++;};
 const gate=(r,role)=>{
  if(r.status==='issued'){say('ALREADY ISSUED — no mutation or duplicate invoice.','이미 발행됨 — 변경·중복 발행 없음.');return false;}
  if(r.policy[role]==='required'&&!r[role]){say('BLOCKED — required '+role+' assignment is missing.','차단 — 필수 '+tr(role,role==='issuer'?'발행':'검수')+' 담당자가 미지정입니다.');return false;}
  if(r[role]&&r[role]!==state.user){say('BLOCKED — only the designated actor can perform this action in the demo.','차단 — 시안에서는 지정된 사용자만 이 작업을 수행할 수 있습니다.');return false;}
  return true;
 };
 document.addEventListener('change',e=>{
  const name=e.target.name;
  if(name==='sessionUser'){state.user=e.target.value;state.message=tr('Viewer changed; recorded actors remain unchanged.','조회 사용자 변경 · 기존 처리자 기록 유지.');render();return;}
  if(['mode','msp','company','month','status','person','showPeople'].includes(name)){
   state[name]=name==='showPeople'?e.target.checked:e.target.value;
   if(name==='msp')state.company=state.mode==='single'?h.companies.find(c=>c.msp===state.msp).id:'all';
   if(name==='mode'&&state.mode==='single'&&state.company==='all')state.company=h.companies.find(c=>c.msp===state.msp).id;
   if(location.hash.includes('/'))history.replaceState(null,'','#register');
   state.message='';render();
  }
 });
 document.addEventListener('click',e=>{
  const action=e.target.closest('[data-action]')?.dataset.action;if(!action)return;
  const [,id]=location.hash.slice(1).split('/'),r=record(id);
  if(action==='policy'){
   const msp=msps[id]?id:state.msp,p=state.policies[msp],issuer=value('issuerPolicy'),reviewer=value('reviewerPolicy');
   if(p.issuer===issuer&&p.reviewer===reviewer){say('No policy change.','정책 변경 없음.');return;}
   p.issuer=issuer;p.reviewer=reviewer;p.version++;
   state.records.filter(x=>company(x.company).msp===msp&&x.status!=='issued').forEach(x=>{invalidate(x,'Policy changed to P'+p.version);x.policy={...p};});
   state.policyEvents.push({...stamp(),msp,version:p.version});say('Policy saved. Draft reviews invalidated; issued snapshots unchanged.','정책 저장. 초안 검수 무효화 · 발행 스냅샷 유지.');return;
  }
  if(action==='renewal'){
   const c=company(id),effective=value('effective'),reason=value('renewalReason').trim();
   if(!c||!effective||effective<='2026-10-28'||!reason){say('BLOCKED — future effective date after Oct 28 and reason required.','차단 — 10월 28일 이후 적용일과 사유가 필요합니다.');return;}
   if(state.renewals.some(x=>x.company===id&&x.effective===effective)){say('Existing renewal draft for that company/date retained.','해당 회사·적용일의 기존 갱신 초안을 유지합니다.');return;}
   state.renewals.push({id:c.contract+'-renewal-'+(state.renewals.length+1),company:id,effective,reason,...stamp()});say('Renewal draft saved; not activated. Issued contracts remain fixed.','갱신 초안 저장 · 미활성. 발행 당시 계약은 유지됩니다.');return;
  }
  if(!r)return;
  if(action==='assign'){
   if(r.status==='issued')return;
   const issuer=value('issuerAssigned'),reviewer=value('reviewerAssigned');
   if(issuer===r.issuer&&reviewer===r.reviewer){say('Assignments unchanged.','담당자 지정 변경 없음.');return;}
   invalidate(r,'Assignments changed');r.issuer=issuer;r.reviewer=reviewer;r.events.push({kind:'assignment',...stamp(),detail:'Issuer '+(issuer||'unassigned')+' / reviewer '+(reviewer||'unassigned')});say('Assignments saved; draft requires current review.','담당자 지정 저장 · 현재 초안 검수 필요.');return;
  }
  if(action==='evidence'){
   if(r.status==='issued')return;
   const valid=value('evidence')==='valid';if(valid===r.sourceValid){say('Evidence unchanged.','근거 변경 없음.');return;}
   invalidate(r,'Evidence changed');r.sourceValid=valid;r.calculation++;say('Evidence changed; previous review is no longer current.','근거 변경 · 이전 검수는 현재 버전에 유효하지 않습니다.');return;
  }
  if(action==='review'){
   if(!gate(r,'reviewer'))return;
   if(!r.sourceValid){say('BLOCKED — evidence mismatch; no completed reviewer recorded.','차단 — 근거 불일치 · 완료 검수자 기록 없음.');return;}
   if(r.review?.signature===signature(r)){say('Current draft already reviewed; no duplicate event.','현재 초안은 이미 검수됨 · 중복 이력 없음.');return;}
   r.review={...stamp(),signature:signature(r)};r.events.push({kind:'review',...r.review,detail:'D'+r.revision+' / '+r.contract+' / C'+r.calculation+' / P'+r.policy.version});say('REVIEWED — actual actor and time recorded. Issuance still pending.','검수 완료 — 실제 처리자·시각 기록. 발행은 대기입니다.');return;
  }
  if(action==='issue'){
   if(!gate(r,'issuer'))return;
   if(!r.sourceValid||!r.review||r.review.signature!==signature(r)){say('BLOCKED — current successful review is required.','차단 — 현재 버전의 성공한 검수가 필요합니다.');return;}
   r.issue=stamp();r.issuedAt=r.issue.time;r.number='DEMO-JR-202608-001';r.status='issued';r.events.push({kind:'issue',...r.issue,detail:r.number+' / review '+r.review.id+' / '+r.review.time});say('ISSUED — one fixed invoice and actor snapshot created locally.','발행 완료 — 고정 인보이스·처리자 스냅샷을 로컬에 한 번 생성했습니다.');
  }
 });
 addEventListener('hashchange',()=>{state.message='';render();document.querySelector('#workspace').focus({preventScroll:true});window.scrollTo(0,0);});
 render();
})();
