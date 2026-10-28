(()=>{
 const ko=document.documentElement.lang==='ko';
 const tr=(en,kr)=>ko?kr:en;
 const screens=[...document.querySelectorAll('.screen')];
 const state={accepted:false,mapped:false,readback:false,run:'unknown',review:'pending',credit:'pending'};
 const render=()=>{
  const id=location.hash.slice(1),active=screens.find(s=>s.id===id)||screens[0];
  screens.forEach(s=>{s.hidden=s!==active;s.querySelectorAll('nav a').forEach(a=>{if(a.hash==='#'+active.id)a.setAttribute('aria-current','page');else a.removeAttribute('aria-current');});});
  active.focus({preventScroll:true});window.scrollTo(0,0);
 };
 window.addEventListener('hashchange',render);render();
 document.querySelectorAll('form').forEach(f=>f.addEventListener('submit',e=>e.preventDefault()));
 const download=(name,body,type)=>{
  const url=URL.createObjectURL(new Blob(['\ufeff'+body],{type}));
  const a=document.createElement('a');a.href=url;a.download=name;a.click();setTimeout(()=>URL.revokeObjectURL(url),500);
 };
 document.addEventListener('click',e=>{
  const button=e.target.closest('[data-action]');if(!button)return;
  const form=button.closest('form'),data=new FormData(form),screen=form.closest('.screen'),result=screen.querySelector('[data-result]');
  const say=(en,kr,error=false)=>{result.textContent=tr(en,kr);result.classList.toggle('error',error);};
  const interval=()=>data.get('start')==='2026-09-01'&&data.get('end')==='2026-10-01';
  switch(button.dataset.action){
   case 'intake':
    if(data.get('evidence')!=='valid'){say('BLOCKED · invoice period mismatch. No accepted inputs changed.','차단 · 청구 기간 불일치. 수용 입력은 변경되지 않았습니다.',true);break;}
    state.accepted=true;say('ACCEPTED · SRC-A2 / 9,000 + 1,000 = 10,000 USD; residual 0. Duplicate 100 quarantined. Continue to account assignment.','수용됨 · SRC-A2 / USD 9,000 + 1,000 = 10,000 · 차이 0. 중복 100 격리 유지. 계정 귀속으로 이동하세요.');break;
   case 'mapping':
    if(state.review==='issued'){say('ISSUED ORIGINAL · use a controlled correction; the issued mapping cannot be replaced.','발행 원본 · 통제된 정정 경로가 필요하며 발행 귀속을 대체할 수 없습니다.',true);break;}
    if(!state.accepted){say('BLOCKED · first accept F1 in the intake case. No mapping saved.','차단 · 먼저 원본 누락 화면에서 F1을 수용하세요. 귀속은 저장되지 않았습니다.',true);break;}
    if(data.get('customer')!=='Aster'||!interval()){say('BLOCKED · choose Aster and the supported September interval. Other dates require a separate impact preview; nothing saved.','차단 · Aster와 9월 적용 기간을 선택하세요. 다른 기간은 별도 영향 검토가 필요하며 저장하지 않았습니다.',true);break;}
    state.mapped=true;state.run='unknown';state.review='pending';say('DRAFT SAVED · MAP-A2 / Aster / September. Assigned 10,000; unassigned 0. Source remains 10,000. R-A2 not yet run; approval pending.','초안 저장 · MAP-A2 / Aster / 9월. 귀속 10,000 · 미귀속 0 · 원본 10,000 유지. R-A2 미실행 · 승인 대기.');break;
   case 'pricing':{
    if(!interval()||data.get('overlap')!=='resolved'){say('BLOCKED · R-ALL overlaps or the effective interval needs a new impact review. Conflict remains; no draft saved.','차단 · R-ALL 중복 또는 유효 기간 영향 검토 필요. 충돌 미해결 · 초안 미저장.',true);break;}
    const discount=Number(data.get('discount')),net=10000-discount*100+1000;
    const rows=screen.querySelectorAll('tbody tr');rows[1].children[2].textContent=`-${(discount*100).toFixed(2)} / ${discount}%`;rows[3].children[2].textContent=net.toLocaleString('en-US',{minimumFractionDigits:2});
    screen.querySelectorAll('.evidence-panel p')[1].textContent=tr(`Matched usage and Marketplace; unmatched 0. Supplier cost USD 8,000; proposed net margin USD ${net-8000}, before FX.`,`사용료·Marketplace 매칭 · 미매칭 0. 원가 USD 8,000 · 제안 순액 마진 USD ${net-8000}, 환산 전.`);
    screen.querySelector('.outcome p').textContent=tr(`Selected draft: ${discount}% discount; net USD ${net.toLocaleString('en-US')}; change ${net-11000}. Recalculation/approval pending. The FX page uses a separate pinned 10% snapshot p5-FX.`,`선택 초안: 할인 ${discount}% · 순액 USD ${net.toLocaleString('en-US')} · 변화 ${net-11000}. 재계산·승인 대기. 환산 화면은 별도 고정된 10% 스냅샷 p5-FX를 사용합니다.`);
    say(`DRAFT p5 · net USD ${net.toLocaleString('en-US')}; delta ${net-11000}; unmatched 0; conflicts 0. Marketplace 1,000 excluded. Recalculation and approval pending.`,`p5 초안 · 순액 USD ${net.toLocaleString('en-US')} · 변화 ${net-11000} · 미매칭 0 · 충돌 0. Marketplace 1,000 할인 제외. 재계산·승인 대기.`);break;}
   case 'seats':
    if(data.get('policy')!=='actual'){say('BLOCKED · missing proration policy is not a zero adjustment.','차단 · 일할 정책 미지정을 조정액 0으로 처리하지 않습니다.',true);break;}
    say('VERIFIED S2 · 100 × 20 + 10 × 20 × 15 / 30 = USD 2,100. Annual term / monthly billing / September actual days. No order submitted.','S2 검증 · 100 × 20 + 10 × 20 × 15 / 30 = USD 2,100. 연간 약정 / 매월 청구 / 9월 실제 일수. 주문은 실행하지 않았습니다.');break;
   case 'commitments':
    if(data.get('model')!=='typed'){say('BLOCKED · generic discount would lose credit semantics. Select the identified offset model.','차단 · 일반 할인으로는 크레딧 의미를 보존할 수 없습니다. 지정 상계 모델을 선택하세요.',true);break;}
    say('VERIFIED · GCP 100 + 80 - 80 = 100 USD. AWS supplied effective allocation 100 = consumed 80 + retained 20. Additional recovery 0.','검증 · GCP USD 100 + 80 - 80 = 100. AWS 제공 실효 배부 100 = 사용 80 + 보유 부담 20. 추가 회수 0.');break;
   case 'fx':{
    const rate=data.get('rate');
    if(rate==='0'){say('BLOCKED · raw exchange rate must be positive. No conversion or approval performed.','차단 · 원시 환율은 양수여야 합니다. 환산·승인하지 않았습니다.',true);break;}
    say(rate==='0.90'?'FX-J1 · EUR 9,000 net + 900 illustrative tax = 9,900 gross. J-USD remains separate at USD 200.':'STALE APPROVAL · proposed FX-J2 gives EUR 9,100 + 910 = 10,010. Recalculation and a new approval are required. Original J-EUR is unchanged.',rate==='0.90'?'FX-J1 · 순액 EUR 9,000 + 예시 세금 900 = 총액 9,900. J-USD는 USD 200으로 분리 유지.':'승인 만료 · 제안 FX-J2는 EUR 9,100 + 910 = 10,010. 재계산·새 승인이 필요합니다. J-EUR 원본은 유지됩니다.',rate==='0.91');break;}
   case 'readback':state.readback=true;say('READ-BACK CONFIRMED · R-A1 failed; no candidate created. Retry is now distinguishable from an unknown prior execution.','조회 확인 · R-A1 실패 · 후보 생성 없음. 이제 이전 실행의 미확인 상태와 재시도를 구분할 수 있습니다.');break;
   case 'queue':
    if(!state.mapped||!state.readback){say('BLOCKED · save MAP-A2 and read back R-A1 first. No duplicate job queued.','차단 · MAP-A2 저장과 R-A1 조회가 먼저 필요합니다. 중복 작업은 등록하지 않았습니다.',true);break;}
    if(data.get('mode')==='refresh'){say('SCOPE PREVIEW · refresh proposes SRC-A3 for Aster September; no supplied new file in this example. Acquisition pending, candidate not created, old approval cannot transfer.','범위 미리보기 · 새 원본 반영은 Aster 9월 SRC-A3를 제안합니다. 이 예시에는 새 파일이 없어 수집 대기 · 후보 미생성 · 기존 승인 승계 불가.',true);break;}
    if(['queued','running','verified'].includes(state.run)){say('EXISTING R-A2 · inspect this job instead of creating a duplicate.','기존 R-A2가 있습니다. 중복 생성 대신 해당 작업을 확인하세요.',true);break;}
    state.run='queued';say('QUEUED R-A2 · pinned SRC-A2 / MAP-A2 / PRICE-A1. Advance the staged run to inspect execution.','R-A2 대기 · SRC-A2 / MAP-A2 / PRICE-A1 고정. 실행 단계 진행으로 상태를 확인하세요.');break;
   case 'advance':
    if(state.run==='queued'){state.run='running';say('RUNNING R-A2 · no final amount or approval available yet.','R-A2 실행 중 · 최종 금액·승인 아직 없음.');}
    else if(state.run==='running'){state.run='verified';say('VERIFIED A2 · source = attributed = expected = candidate USD 10,000; residual 0. TAX-DEMO-0. Review pending.','A2 검증 · 원본 = 귀속 = 기대 = 후보 USD 10,000 · 차이 0. TAX-DEMO-0. 승인 대기.');}
    else say('No queued/running job to advance. Return to scope selection or inspect the verified candidate.','진행할 대기·실행 작업이 없습니다. 범위 선택 또는 검증 후보를 확인하세요.',true);break;
   case 'reject':
    if(state.run!=='verified'||state.review!=='pending'||!String(data.get('reason')).trim()){say('BLOCKED · a verified A2 and a return reason are required.','차단 · 검증된 A2와 반려 사유가 필요합니다.',true);break;}
    if(state.review==='issued'){say('ISSUED ORIGINAL · use a linked correction, not a return action.','발행 원본 · 반려가 아닌 연결 정정으로 처리해야 합니다.',true);break;}
    state.review='returned';say('RETURNED A2 · '+data.get('reason')+' · Attach F1 and reverify A3. No financial amount changed.','A2 반려 · '+data.get('reason')+' · F1 첨부 후 A3를 재검증하세요. 금액은 변경되지 않았습니다.');break;
   case 'attach':
    if(!['returned','stale'].includes(state.review)){say('BLOCKED · this stage follows a return or stale-input review.','차단 · 반려 또는 입력 변경 검토 이후 단계입니다.',true);break;}
    state.review='rechecked';say('RECHECKED A3 · DOC-A2 attached, fixed original inputs restored, net USD 10,000; all controls rechecked. Approval pending.','A3 재검증 · DOC-A2 첨부 · 원래 고정 입력 복원 · 순액 USD 10,000 · 전체 검증 재확인. 승인 대기.');break;
   case 'approve':
    if(state.review!=='rechecked'){say('BLOCKED · A3 must be rechecked first. No approval recorded.','차단 · 먼저 A3 재검증이 필요합니다. 승인은 기록하지 않았습니다.',true);break;}
    state.review='approved';say('APPROVED A3 · reviewer R-MIN / USD 10,000 / DOC-A2. Issuance is a separate next decision.','A3 승인 · 검토자 R-MIN / USD 10,000 / DOC-A2. 발행은 별도 다음 결정입니다.');break;
   case 'stale':
    if(state.review!=='approved'){say('This conflict example requires an approved, unissued A3. Issued originals require linked corrections.','이 충돌 예시는 승인 후 미발행 A3가 필요합니다. 발행 원본은 연결 정정이 필요합니다.',true);break;}
    state.review='stale';say('STALE · changed input detected. Prior approval retained as history only. Issuance blocked; restore/reverify and reapprove.','승인 만료 · 입력 변경 감지. 기존 승인은 이력으로만 유지. 발행 차단 · 복원/재검증 후 재승인 필요.',true);break;
   case 'issue':
    if(state.review!=='approved'){say('BLOCKED · current fixed A3 approval required. No document issued.','차단 · 현재 고정 A3 승인이 필요합니다. 문서는 발행하지 않았습니다.',true);break;}
    state.review='issued';say('SIMULATED ISSUED · INV-ASTER-09 / A3 / gross USD 10,000. Original fixed. Delivery/ERP pending; payment unconfirmed.','발행 시연 · INV-ASTER-09 / A3 / 총액 USD 10,000. 원본 고정. 전달·ERP 대기 · 결제 미확인.');break;
   case 'creditcheck':
    if(state.credit==='issued'){say('Issued c1 remains immutable; a later change requires a new linked correction.','발행 c1은 불변입니다. 이후 변경에는 새 연결 정정이 필요합니다.',true);break;}
    if(data.get('treatment')!=='pass'){state.credit='pending';say('BLOCKED · attribution and contract treatment unresolved; customer amount remains uncalculated.','차단 · 귀속·계약 처리 미해결 · 고객 금액 미계산 유지.',true);break;}
    state.credit='checked';say('CHECKED c1 · approved attribution -500 × CREDIT-NS-1 rate 1 = USD -500; tax 0. Independent residual 0. Approval pending.','c1 검증 · 승인 귀속 -500 × CREDIT-NS-1 비율 1 = USD -500 · 세금 0. 독립 차이 0. 승인 대기.');break;
   case 'creditapprove':
    if(state.credit!=='checked'||data.get('treatment')!=='pass'){say('BLOCKED · verify the fixed c1 treatment first.','차단 · 먼저 고정 c1 처리를 검증하세요.',true);break;}
    state.credit='approved';say('APPROVED c1 · R-LEE. The customer credit is not issued until the separate issuance decision.','c1 승인 · R-LEE. 별도 발행 결정 전까지 고객 정정 문서는 미발행입니다.');break;
   case 'creditissue':
    if(state.credit!=='approved'||data.get('treatment')!=='pass'){say('BLOCKED · current c1 approval required.','차단 · 현재 c1 승인이 필요합니다.',true);break;}
    state.credit='issued';say('SIMULATED ISSUED CN-NS-1007 · -500 USD; original 136,560 unchanged; net document position 136,060. Delivery/ERP pending, payment unconfirmed.','CN-NS-1007 발행 시연 · USD -500 · 원본 136,560 유지 · 문서 순액 136,060. 전달·ERP 대기 · 결제 미확인.');break;

  }
 });
})();
