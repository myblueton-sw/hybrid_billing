(()=>{
 const {tr,months,money,say}=window.hb4;
 const state={sample:false,verified:false,reconciled:false,accepted:false,revision:2,history:[],month:'2026-09'};
 const byId=id=>document.getElementById(id);
 const selectValue=(screen,name)=>screen.querySelector('[name="'+name+'"]').value;
 const updateAudit=()=>{
  const audit=byId('audit'),version='E'+state.revision;
  audit.querySelector('.scope4').textContent='Cedar / SAMPLE-AWS-CEDAR-09 / P1 / '+version+' / '+tr('current revision','현재 버전');
  byId('extract').querySelector('[data-action="verify"]').textContent=tr('Save '+version+' and verify sums',version+' 저장·합계 검증');
  const rows=audit.querySelectorAll('tbody tr');
  rows[2].children[1].textContent=version+' / '+(state.verified?'4,000.00':tr('draft / recheck required','초안 / 재검증 필요'));
  rows[2].children[2].textContent=state.reason?tr('p1 R3 / DEMO-OPS / reason: ','p1 R3 / DEMO-OPS / 사유: ')+state.reason:tr('Reason and reviewer confirmation pending','사유·검토 확인 대기');
  rows[3].children[1].textContent=state.reconciled?'C1 59,800 → C2 64,000':tr('C2 reconciliation pending','C2 대사 대기');
  rows[4].children[1].textContent='P1 + '+version+' + C2';
  rows[4].children[2].textContent=state.accepted?tr('Accepted billing input only','청구 입력만 수용'):tr('Not accepted / current checks required','미수용 / 현재 검증 필요');
  audit.querySelector('.outcome4 p').textContent=tr('Current '+version+': ','현재 '+version+': ')+(state.accepted?tr('accepted input USD64,000, residual0; no bill approval or issuance.','입력 USD64,000 수용 · 차이0 · 청구 승인·발행 아님.'):tr('draft or reviewed evidence only; new acceptance requires current total and source checks.','초안 또는 검토 근거만 존재 · 새 수용에는 현재 합계·원본 검증 필요.'));
  let history=audit.querySelector('.accepted-history');
  if(!history){history=document.createElement('div');history.className='accepted-history';audit.querySelector('.outcome4').append(history);}
  history.replaceChildren();
  for(const item of state.history){const line=document.createElement('p');line.textContent=tr('Preserved accepted snapshot: ','보존된 수용 스냅샷: ')+'P1 / E'+item.revision+' / C2 / USD64,000 / E1 R3:400.00 / accepted R3:4,000.00 / DEMO-OPS / '+tr('reason: ','사유: ')+item.reason;history.append(line);}
 };

 const syncDraft=()=>{
  const extract=byId('extract'),diff=byId('differences'),version='E'+state.revision;
  extract.querySelector('.scope4').textContent='Cedar / SAMPLE-AWS-CEDAR-09 / P1 / '+version+' / '+tr('current draft','현재 초안');
  extract.querySelector('.three-pane>div:nth-child(2)>h2').textContent=tr('Current draft / ','현재 초안 / ')+version;
  const row=extract.querySelectorAll('tbody tr')[2];row.children[1].textContent=extract.querySelector('[name="support"]').value;row.children[2].textContent=tr('Not verified / current draft','미검증 / 현재 초안');
  diff.querySelector('.scope4').textContent='Cedar / SAMPLE-AWS-CEDAR-09 / '+version+' / C1 → C2 / USD';
  say(extract,'CURRENT '+version+' DRAFT · current fields are not verified. Previous accepted snapshots remain in audit history.','현재 '+version+' 초안 · 현재 필드는 미검증입니다. 이전 수용 스냅샷은 이력에 보존됩니다.',true);
  say(diff,'CURRENT '+version+' · reconciliation not valid. Verify this extraction and reconcile the current source evidence.','현재 '+version+' · 대사 미완료. 이 추출과 현재 원본 근거를 다시 검증·대사하세요.',true);
  updateAudit();
 };
 const updateMonth=month=>{
  const data=months.find(m=>m.month===month),screen=byId('month');state.month=month;
  const ready=data.sale!==null;
  screen.querySelector('h1').textContent=month+' / '+tr('Close and evidence history','마감 상세와 근거 이력');
  screen.querySelector('.scope4').textContent='Northstar / '+month+' / USD / '+data.version+' / '+(data.document||tr('Not issued','미발행'));
  const entries=month==='2026-09'?[
   [tr('Source / allocation','원본 / 귀속'),'s3 / m2 / VMware cp6',tr('Original cost 122,000','원본 원가 122,000')],
   [tr('Contract / calculation / control','계약 / 계산 / 검산'),'p4 / r8 / W9',tr('Net 136,560; residual 0','순액 136,560 · 차이 0')],
   [tr('Currency / tax','통화 / 세금'),'USD / TAX-DEMO-0',tr('No FX conversion; synthetic tax 0','환산 없음 · 합성 세금 0')],
   [tr('Original invoice','원본 청구서'),data.document,tr('Issued ','발행일 ')+data.issued],
   [tr('Later credit','후속 크레딧'),'CN-NS-1007 / 2026-10-07',tr('September attribution / -500','9월 귀속 / -500')]
  ]:[
   [tr('Historical source snapshot','과거 원본 스냅샷'),'DEMO-SRC-'+month,ready?tr('Fixed synthetic cost ','고정 합성 원가 ')+money(data.cost):tr('Incomplete; unavailable','미완료 · 금액 미확정')],
   [tr('Calculation version','계산 버전'),data.version,ready?tr('Fixed net ','고정 순액 ')+money(data.sale):tr('Draft only; not comparable','초안만 존재 · 비교 제외')],
   [tr('Document','문서'),data.document||tr('Not issued','미발행'),data.issued||tr('No issue date','발행일 없음')],
   [tr('Historical supporting detail','과거 상세 근거'),tr('Metadata example only','메타데이터 예시만 제공'),tr('Detailed PDF not supplied for this month','해당 월 상세 PDF는 제공되지 않음')]
  ];
  const body=screen.querySelector('tbody');body.replaceChildren();
  for(const row of entries){const r=document.createElement('tr');for(const cell of row){const c=document.createElement('td');c.textContent=cell;r.append(c);}body.append(r);}
  screen.querySelector('.outcome4 p').textContent=ready?tr('Fixed original: ','고정 원본: ')+data.document+' / USD '+money(data.sale)+tr('. Current prices do not rewrite this snapshot.','. 현재 요율로 이 스냅샷을 변경하지 않습니다.'):tr('October remains open. No confirmed amount or invoice is available.','10월은 진행 중입니다. 확정 금액·청구서가 없습니다.');
  const notes=screen.querySelectorAll('.notes p');
  notes[0].textContent=month==='2026-09'?tr('Original 136,560; credit -500 issued October 7; adjusted September document position 136,060. Not a payment balance.','원본 136,560 · 10월 7일 크레딧 -500 · 정정 반영 9월 문서 순액 136,060. 결제 잔액이 아닙니다.'):tr('This month has its own synthetic snapshot. No September credit or invoice has been attached to it.','해당 월의 합성 스냅샷입니다. 9월 크레딧·청구서를 이 월에 연결하지 않습니다.');
  screen.querySelectorAll('.controls a').forEach(a=>a.hidden=month!=='2026-09');
  say(screen,tr('Selected month: ','선택 월: ')+month,tr('Selected month: ','선택 월: ')+month);
  location.hash='month';
 };
 byId('inbox').querySelector('[name="pdf"]').addEventListener('change',e=>{
  const file=e.target.files[0];if(!file)return;
  if(!file.name.toLowerCase().endsWith('.pdf')){say(byId('inbox'),'Select a PDF extension. Contents were not read.','PDF 확장자를 선택하세요. 내용은 읽지 않았습니다.',true);return;}
  say(byId('inbox'),'LOCAL METADATA ONLY · '+file.name+' / '+file.size+' bytes. Not uploaded, read or parsed. Sample action below uses SAMPLE-AWS-CEDAR-09, never this selected file.','로컬 메타데이터만 확인 · '+file.name+' / '+file.size+' bytes. 업로드·내용 읽기·파싱하지 않았습니다. 아래 예시 작업은 선택 파일이 아닌 SAMPLE-AWS-CEDAR-09를 사용합니다.');
 });
 const invalidate=()=>{if(!state.accepted){state.verified=false;state.reconciled=false;syncDraft();say(byId('extract'),'EDITED DRAFT · total and source checks invalidated. Reverify the current fields.','초안 변경 · 합계·원본 대사 검증 무효화. 현재 필드를 다시 검증하세요.',true);}};
 byId('extract').querySelectorAll('input').forEach(input=>input.addEventListener('input',invalidate));
 byId('differences').querySelector('select').addEventListener('change',()=>{if(!state.accepted){state.reconciled=false;updateAudit();say(byId('differences'),'Source selection changed; current reconciliation is invalid.','근거 선택 변경 · 현재 대사 무효화.',true);}});
 document.addEventListener('click',e=>{
  const open=e.target.closest('[data-open-month]');if(open){updateMonth(open.dataset.openMonth);return;}
  const button=e.target.closest('[data-action]');if(!button)return;
  const screen=button.closest('.screen');
  switch(button.dataset.action){
   case 'filter':{
    const month=selectValue(screen,'month'),adjusted=selectValue(screen,'basis')==='adjusted';
    screen.querySelectorAll('tbody tr').forEach(row=>{const data=months.find(m=>m.month===row.dataset.month);row.hidden=month!=='all'&&data.month!==month;if(data.sale!==null){const credit=adjusted?data.credit:0;row.children[1].textContent=money(data.cost+credit);row.children[2].textContent=money(data.sale+credit);row.children[3].textContent=money(data.sale-data.cost);}});
    say(screen,adjusted?'ADJUSTED ATTRIBUTION · as of October 28. October-issued -500 belongs to September; original close unchanged.':'ORIGINAL CLOSE · historical snapshots retained; no current-price restatement.',adjusted?'정정 귀속 기준 · 10월 28일 현재. 10월 발행 -500은 9월 귀속이며 원본 마감은 유지됩니다.':'당시 마감 원본 기준 · 과거 스냅샷 보존 · 현재 요율 재평가 없음.');break;}
   case 'immutable':say(screen,'READ-ONLY SNAPSHOT · '+state.month+'. Historical input cannot be overwritten. Changes require a separate reviewed revision/correction.','읽기 전용 스냅샷 · '+state.month+'. 과거 입력을 덮어쓸 수 없습니다. 별도 검토 버전·정정이 필요합니다.');break;
   case 'variance':{
    const adjusted=selectValue(screen,'basis')==='adjusted';
    say(screen,adjusted?'ADJUSTED · 130,000 + 5,000 + 2,000 - 440 - 500 = 136,060. Change +6,060 / +4.66%. September original remains 136,560.':'ORIGINAL · 130,000 + 5,000 + 2,000 - 440 = 136,560. Change +6,560 / +5.05%. Credit remains a separate later document.',adjusted?'정정 반영 · 130,000 + 5,000 + 2,000 - 440 - 500 = 136,060. 증가 +6,060 / +4.66%. 9월 원본은 136,560 유지.':'원본 기준 · 130,000 + 5,000 + 2,000 - 440 = 136,560. 증가 +6,560 / +5.05%. 크레딧은 후속 별도 문서.');break;}
   case 'sample':{
    const branch=selectValue(screen,'sample-state');
    if(branch!=='ready'){const labels={duplicate:tr('Duplicate fingerprint: blocked; existing original retained.','동일 지문 중복: 차단 · 기존 원본 유지.'),encrypted:tr('Encrypted: request authorized decrypted copy; no extraction result.','암호화: 권한 있는 해제본 확보 필요 · 추출 결과 없음.'),unsupported:tr('Unsupported layout: manual mapping required; no automatic acceptance.','미지원 형식: 수동 매핑 필요 · 자동 수용 없음.'),failed:tr('OCR failed: unreadable page; original attempt retained, clearer evidence required.','OCR 실패: 페이지 판독 불가 · 실패 이력 보존 · 선명한 근거 필요.')};say(screen,labels[branch],labels[branch],true);break;}
    if(state.accepted||state.history.length){say(screen,'Accepted P1/E2/C2 remains immutable. Start a new revision from the audit screen.','수용된 P1/E2/C2는 불변입니다. 이력 화면에서 새 버전을 시작하세요.',true);break;}
    state.sample=true;state.verified=false;state.reconciled=false;
    byId('extract').querySelector('[name="support"]').value='400.00';byId('extract').querySelector('[name="reason"]').value='';byId('extract').querySelector('[name="source-checked"]').checked=false;byId('extract').querySelectorAll('tbody tr')[2].children[1].textContent='400.00';state.reason='';syncDraft();
    say(screen,'SYNTHETIC FIXTURE LOADED · SAMPLE-AWS-CEDAR-09 / P1 / E1. Extraction review required: support 400; printed total 64,000. No selected local file was analyzed.','합성 예시 불러옴 · SAMPLE-AWS-CEDAR-09 / P1 / E1. 지원 비용 400 · 원본 총액 64,000 · 추출 검토 필요. 선택한 로컬 파일은 분석하지 않았습니다.');break;}
   case 'verify':{
    if(!state.sample||state.accepted){say(screen,'BLOCKED · load the synthetic sample, or start a new revision for accepted evidence.','차단 · 합성 예시를 불러오거나, 수용된 근거는 새 버전을 시작하세요.',true);break;}
    const amount=screen.querySelector('[name="support"]').value.trim(),reason=screen.querySelector('[name="reason"]').value.trim(),checked=screen.querySelector('[name="source-checked"]').checked;
    if(!/^4000(?:\.00)?$/.test(amount)||!reason||!checked){state.verified=false;state.reconciled=false;syncDraft();say(screen,'BLOCKED · confirm R3=4,000.00 against page 1 and record a reason. Current draft verification is incomplete; the original E1 discrepancy remains historical evidence.','차단 · 1페이지 R3=4,000.00 원본 확인과 사유가 필요합니다. 현재 초안 검증 미완료 · 원래 E1 차이는 과거 근거로 보존됩니다.',true);break;}
    state.verified=true;state.reconciled=false;state.reason=reason;updateAudit();
    byId('differences').querySelector('.scope4').textContent='Cedar / SAMPLE-AWS-CEDAR-09 / E'+state.revision+' / C1 → C2 / USD';
    say(byId('differences'),'Current extraction verified; source reconciliation still pending.','현재 추출 검증 완료 · 원본 대사는 대기입니다.',true);
    screen.querySelector('.three-pane>div:nth-child(2)>h2').textContent=tr('Saved extraction / E','저장된 추출 / E')+state.revision;
    screen.querySelector('.scope4').textContent='Cedar / SAMPLE-AWS-CEDAR-09 / 2026-09 / USD / P1 / E'+state.revision;
    screen.querySelectorAll('tbody tr')[2].children[1].textContent='4,000.00';
    screen.querySelectorAll('tbody tr')[2].children[2].textContent=tr('Human-verified / new extraction','담당자 검증 / 새 추출 버전');
    say(screen,'VERIFIED E'+state.revision+' · 40,000 +20,000 +4,000 =64,000; tax0; residual0. P1 and E1 retained. Collected-data reconciliation remains pending.','E'+state.revision+' 검증 · 40,000 +20,000 +4,000 =64,000 · 세금0 · 차이0. P1·E1 보존. 수집 데이터 대사는 대기입니다.');break;}
   case 'reconcile':
    if(!state.verified||state.accepted||selectValue(screen,'detail')!=='correct'){say(screen,'BLOCKED · verified extraction and matching September/USD C2 evidence are required. Missing support is not zero.','차단 · 검증된 추출과 9월/USD가 일치하는 C2 근거가 필요합니다. 지원 비용 누락은 0이 아닙니다.',true);break;}
    state.reconciled=true;updateAudit();
    say(screen,'RECONCILED C2 · known59,800 + storage200 + support4,000 =64,000; residual0. Shared payer attribution Cedar48,000 +other16,000 =64,000. Financial approval still separate.','C2 대사 완료 · 확인59,800 + 스토리지200 + 지원비4,000 =64,000 · 차이0. 공유 계정 귀속 Cedar48,000 +기타16,000 =64,000. 금액 승인은 별도입니다.');break;
   case 'accept':
    if(state.accepted){say(screen,'ALREADY ACCEPTED · no second acceptance or duplicate billing input.','이미 수용됨 · 이중 수용·중복 청구 입력을 만들지 않습니다.');break;}
    if(!state.verified||!state.reconciled){say(screen,'BLOCKED · current extraction and source reconciliation must both pass. Arithmetic alone is insufficient.','차단 · 현재 추출·원본 대사가 모두 통과해야 합니다. 합계 일치만으로는 부족합니다.',true);break;}
    state.accepted=true;state.history.push({revision:state.revision,reason:state.reason});updateAudit();
    byId('extract').querySelectorAll('input').forEach(input=>input.disabled=true);byId('differences').querySelector('select').disabled=true;
    say(screen,'ACCEPTED INPUT SNAPSHOT · P1 / E'+state.revision+' / C2 / USD64,000 / reviewer DEMO-OPS / reason: '+state.reason+'. Original and previous extraction retained. No bill approved or issued.','입력 스냅샷 수용 · P1 / E'+state.revision+' / C2 / USD64,000 / 검토자 DEMO-OPS / 사유: '+state.reason+'. 원본·이전 추출 보존. 청구 승인·발행은 하지 않았습니다.');break;
   case 'revise':
    if(!state.accepted){say(screen,'Accept the current reviewed snapshot before demonstrating a new revision.','새 버전 시연 전 현재 검토 스냅샷을 수용하세요.',true);break;}
    state.accepted=false;state.verified=false;state.reconciled=false;state.revision+=1;state.reason='';updateAudit();
    byId('extract').querySelector('.scope4').textContent='Cedar / SAMPLE-AWS-CEDAR-09 / P1 / E'+state.revision+' / '+tr('new draft','새 초안');
    byId('extract').querySelectorAll('input').forEach(input=>input.disabled=false);byId('differences').querySelector('select').disabled=false;
    byId('extract').querySelector('.three-pane>div:nth-child(2)>h2').textContent=tr('Draft revision / E','새 초안 버전 / E')+state.revision;
    byId('extract').querySelector('[name="reason"]').value='';byId('extract').querySelector('[name="source-checked"]').checked=false;syncDraft();
    say(screen,'NEW DRAFT E'+state.revision+' · earlier accepted snapshot preserved. Verification and reconciliation invalidated; review again before acceptance.','새 초안 E'+state.revision+' · 이전 수용 스냅샷 보존. 검증·대사 무효화 · 다시 검토 후 수용해야 합니다.',true);break;
  }
 });
})();
