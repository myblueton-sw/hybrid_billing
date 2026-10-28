(()=>{
 const {tr,months,money,say}=window.hb4;
 document.addEventListener('click',e=>{
  const button=e.target.closest('[data-action]');if(!button)return;
  const screen=button.closest('.screen'),month=screen.querySelector('[name="customer-month"]').value,data=months.find(m=>m.month===month);
  if(data.sale===null){say(screen,'OCTOBER NOT ISSUED · amount unavailable; no invoice download. This is not a zero bill.','10월 미발행 · 금액 미확정 · 청구서 다운로드 없음. 0원 청구가 아닙니다.',true);return;}
  const details=month+' / '+data.document+' / '+tr('issued ','발행일 ')+data.issued+' / USD '+money(data.sale)+tr(' original; later credit ',' 원본 · 후속 크레딧 ')+money(data.credit)+tr('; document position ',' · 문서 순액 ')+money(data.sale+data.credit);
  if(button.dataset.action==='customer-month'){say(screen,details,details);return;}
  if(button.dataset.action==='download'){
   const csv='sample_only,customer,service_month,document,issue_date,currency,selling_net\ntrue,Northstar,'+month+','+data.document+','+data.issued+',USD,'+data.sale.toFixed(2)+'\n'+(data.credit?'true,Northstar,2026-09,CN-NS-1007,2026-10-07,USD,-500.00\n':'');
   const url=URL.createObjectURL(new Blob(['\ufeff'+csv],{type:'text/csv;charset=utf-8'}));const link=document.createElement('a');link.href=url;link.download='Northstar-'+month+'-sample.csv';link.click();setTimeout(()=>URL.revokeObjectURL(url),500);
   say(screen,'Synthetic monthly selling-detail download created. Not a legal invoice or payment confirmation. '+details,'합성 월별 판매 상세를 다운로드했습니다. 법정 청구서·결제 확인서가 아닙니다. '+details);
  }
 });
})();
