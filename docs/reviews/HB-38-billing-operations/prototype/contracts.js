const providerEvidence={
 aws:{name:'AWS',agreement:t('Enterprise agreement / EDP','기업 계약 / EDP'),rows:[
  [t('Contract and eligible scope','계약 및 적용 범위'),t('Term, eligible accounts/SKUs, exclusions, commitment','계약 기간, 적격 계정·SKU, 제외 항목, 약정'),t('Agreement evidence not loaded','계약 근거 미조회')],
  ['CUR / CUR 2.0',t('Net cost and discount representation depend on schema/configuration','스키마·설정에 따라 순비용 및 할인 표현이 달라짐'),t('Do not apply EDP again to already-net cost','순비용에 EDP를 다시 적용하지 않음')],
  ['RI / Savings Plans',t('Purchase, fees, effective usage and unused commitment','구매·요금·실효 사용액·미사용 약정'),t('Do not add effective allocation to the full fee again','실효 배분액과 전체 요금을 중복 합산하지 않음')],
  [t('Invoice / credits','인보이스 / 크레딧'),t('Actual payable versus Billing Conductor pro forma','실제 납부액과 Billing Conductor 가상 청구 구분'),t('Reconcile actual supplier documents','실제 공급자 문서와 대사')]
 ]},
 azure:{name:'Azure',agreement:'EA / MCA / CSP',rows:[
  [t('Agreement and source','계약 및 수집 경로'),'EA / MCA / Partner Center',t('Pin agreement type and dataset version','계약 유형과 데이터 버전을 고정')],
  [t('Negotiated unit price','계약 단가'),'UnitPrice / EffectivePrice',t('Negotiated discount can already be reflected','계약 할인이 이미 반영될 수 있음')],
  [t('Commitment and cost view','약정 및 비용 관점'),'MACC / Actual / Amortized',t('MACC is not another discount; do not add both cost views','MACC는 추가 할인이 아니며 두 비용 관점을 합산하지 않음')],
  [t('Invoice / reconciliation','인보이스 / 대사'),t('Credits, tax, FX and partner adjustments','크레딧·세금·환율·파트너 조정'),t('Usage details alone do not prove final payable','사용량 상세만으로 최종 납부액을 확정하지 않음')]
 ]},
 gcp:{name:'GCP',agreement:t('Account pricing / CUD / reseller','계정 요율 / CUD / 리셀러'),rows:[
  [t('Price evidence','요율 근거'),t('Account-specific price, tier and consumption model','계정별 가격·구간·사용 모델'),t('Preserve observed date and historical availability','관측일과 과거 자료 가용성 보존')],
  [t('Usage and signed credits','사용 비용 및 부호 있는 크레딧'),'cost + credits.amount',t('Do not reapply negotiated pricing already in cost','비용에 반영된 계약 할인을 재적용하지 않음')],
  [t('Commitment','약정'),'CUD / FEE_UTILIZATION_OFFSET',t('Distinguish legacy and new commitment models','기존 약정 모델과 신규 모델 구분')],
  [t('Reseller customer charge','리셀러 고객 청구'),'customer_cost / credits.customer_amount',t('Keep reseller margin and customer credits separate','리셀러 마진과 고객 크레딧 분리')]
 ]},
 oci:{name:'OCI',agreement:'Universal Credits / PAYG'},
 alibaba:{name:'Alibaba Cloud',agreement:t('Account / reseller agreement','계정 / 리셀러 계약'),rows:[
  [t('Commercial role','거래 역할'),t('Reseller purchasing or agency commission','리셀러 매입 또는 대행 수수료'),t('Confirm who pays whom','거래 당사자와 지급 관계 확인')],
  [t('Bill components','청구 구성'),'PretaxGrossAmount / InvoiceDiscount / PretaxAmount',t('Do not discount the already-payable amount again','이미 계산된 납부액에 할인을 다시 적용하지 않음')],
  [t('Coupons','쿠폰'),'DeductedByCoupons',t('Track funder, eligibility, validity and settlement layer','부담 주체·적격 범위·유효기간·정산 단계 추적')],
  [t('Period finalization','월 확정'),t('Final bill, tax and credit-limit settlement','확정 청구·세금·신용 한도 정산'),t('A provisional month cannot prove final reconciliation','잠정 월 자료로 최종 대사를 확정하지 않음')]
 ]}
};

function contractTabs(side){
 return `<div class="contract-tabs" aria-label="${t('Contract type','계약 유형')}"><a href="#contracts" ${side==='supplier'?'aria-current="page"':''}>${t('Supplier purchase contracts','공급자 매입 계약')}</a><a href="#contracts/customer/${state.selected}" ${side==='customer'?'aria-current="page"':''}>${t('Customer sales contracts','고객 판매 계약')}</a></div>`;
}

function contracts(){
 content.innerHTML=title(t('CONTRACTS AND RATES','계약·요율 관리'),t('Contracts and rates','계약·요율 관리'),t('Compare the supplier purchase basis with the separate customer selling agreement.','공급자에게 지급하는 매입 조건과 고객에게 청구하는 판매 조건을 구분합니다.'))+contractTabs('supplier')+
 `<h2>${t('Select a supplier','공급자 선택')}</h2>`+
 table([t('Supplier','공급자'),t('Contract structure to inspect','확인할 계약 구조'),t('Evidence status','근거 상태'),t('Next step','다음 단계')],Object.entries(providerEvidence).map(([id,p])=>[
  `<a href="#contracts/supplier/${id}">${p.name}</a>`,p.agreement,t('Not connected / agreement not loaded','미연결 / 계약 자료 미조회'),`<a href="#contracts/supplier/${id}">${t('View contract evidence','계약 근거 보기')}</a>`
 ]))+
 `<p class="note">${t('These are provider evidence categories, not five active contracts. A real register needs agreement ID, purchasing legal entity/account, validity, version and document links.','공급자별 확인 항목을 보여주는 시안이며 활성 계약 5건이라는 뜻이 아닙니다. 실제 목록에는 계약 ID·매입 법인 및 계정·유효기간·버전·문서 연결이 필요합니다.')}</p><p>${t('Customer contract context','고객 계약 대상')}: <b>${state.selected}</b> · <a href="#detail/${state.selected}">${t('Customer details','고객 상세')}</a></p>`;
}

function supplierContract(id){
 const p=providerEvidence[id];
 if(!p){content.innerHTML=title('404',t('Supplier not found','공급자를 찾을 수 없습니다'),'');return;}
 const back=`<a class="back" href="#contracts">← ${t('Contracts and rates / supplier list','계약·요율 관리 / 공급자 목록')}</a>`;
 if(id==='oci')oci();
 else content.innerHTML=title(`${p.name} / ${t('SUPPLIER CONTRACT EVIDENCE','공급자 계약 근거')}`,p.name+' · '+t('Contract evidence','계약 근거'),p.agreement)+
  table([t('Evidence','근거'),t('What to inspect','확인 항목'),t('Calculation boundary','계산 시 구분')],p.rows)+
  `<h2>${t('From contract to reconciliation','계약에서 대사까지')}</h2><p>${t('Link source charges, effective contract terms, invoice/credit documents and revision history. An observed discount does not authorize customer pass-through.','원천 비용·유효한 계약 조건·인보이스 및 크레딧 문서·변경 이력을 연결합니다. 공급자 할인이 있다고 고객에게 자동 이전하지 않습니다.')}</p><p class="note">${t('Research-backed evidence structure. No account, private rate card or signed agreement is connected.','조사 근거에 따른 화면 구조입니다. 실제 계정·비공개 요율표·서명 계약서는 연결되지 않았습니다.')}</p>`;
 content.insertAdjacentHTML('afterbegin',back+contractTabs('supplier'));
}

function customerContract(id,impact){
 const r=records.find(x=>x.id===id&&(state.mode!=='single'||x.id==='C00226'));
 if(!r){content.innerHTML=title('404',t('Customer not found in this workspace','이 작업 공간에서 고객을 찾을 수 없습니다'),'');return;}
 state.selected=r.id;
 const back=`<a class="back" href="#${impact?'contracts/customer/'+r.id:'detail/'+r.id}">← ${impact?t('Customer sales contract','고객 판매 계약'):t('Customer details','고객 상세')}</a>`;
 const context=`<p class="contract-context"><b>${r.name} · ${r.id}</b> · ${t('Billing month','청구 월')} ${state.month}</p>`;
 if(r.id!=='C00226'||state.month!=='2026-09'){
  content.innerHTML=title(t('CUSTOMER SALES CONTRACT','고객 판매 계약'),impact?t('Pricing impact unavailable','요율 영향 미조회'):t('Contract evidence not loaded','계약 근거 미조회'),t('No supplied policy snapshot exists for this customer and month. Another customer’s prices or September-only benefit will not be substituted.','이 고객·월의 정책 스냅샷이 없습니다. 다른 고객의 가격이나 9월 한정 혜택을 대신 표시하지 않습니다.'))+
   `<p><a href="#contracts/customer/${r.id}${impact?'':'/preview'}">${impact?t('Return to customer contract','고객 계약으로 돌아가기'):t('Check pricing impact evidence','요율 영향 근거 확인')}</a></p><p class="note">${t('Unavailable evidence is distinct from no contract, no discount or a zero bill.','자료 미조회는 계약 없음·할인 없음·청구액 0원과 다릅니다.')}</p>`;
 }else{
  if(impact)preview();else policies();
  const link=content.querySelector('a[href="#preview"]');
  if(link)link.href='#contracts/customer/'+r.id+'/preview';
 }
 content.insertAdjacentHTML('afterbegin',back+contractTabs('customer')+context);
}

function contractRoute(route){
 const parts=route.split('/');
 if(route==='contracts')contracts();
 else if(parts[1]==='supplier'&&parts.length===3)supplierContract(parts[2]);
 else if(parts[1]==='customer'&&(parts.length===3||(parts.length===4&&parts[3]==='preview')))customerContract(parts[2],parts[3]==='preview');
 else content.innerHTML=title('404',t('Contract view not found','계약 화면을 찾을 수 없습니다'),'');
}

render();
