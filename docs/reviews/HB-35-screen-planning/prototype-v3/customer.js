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
   case 'download':{
    const credit=data.get('document')==='credit';
    download(credit?'CN-NS-1007-sample.txt':'DEMO-2026-09104-sample.txt',tr('SYNTHETIC DESIGN SAMPLE · NOT A TAX INVOICE','합성 시안 예시 · 법정 세금계산서 아님')+'\nNorthstar\n'+(credit?'CN-NS-1007\nReference: DEMO-2026-09104 / GCP-09\nIssued: 2026-10-07\nNet: USD -500.00\nTax: USD 0.00\nGross: USD -500.00':'DEMO-2026-09104\nPeriod: 2026-09\nIssued: 2026-10-05\nNet: USD 136560.00\nTax: USD 0.00\nGross: USD 136560.00')+'\n'+tr('Payment unconfirmed. Original document preserved.','결제 미확인. 원본 문서 보존.'),'text/plain;charset=utf-8');
    say('Synthetic summary file downloaded. Not a legal invoice or proof of payment.','합성 문서 요약을 다운로드했습니다. 법정 청구서·결제 증빙이 아닙니다.');break;}
   case 'detail':
    download('Northstar-selling-detail.csv','document,customer,currency,component,net\nDEMO-2026-09104,Northstar,USD,Cloud services,118560.00\nDEMO-2026-09104,Northstar,USD,VMware service,15000.00\nDEMO-2026-09104,Northstar,USD,Support,3000.00\nCN-NS-1007,Northstar,USD,GCP-09 service credit,-500.00\n','text/csv;charset=utf-8');
    say('Synthetic selling-detail CSV downloaded. Source costs and margins are not included.','합성 판매 상세 CSV를 다운로드했습니다. 원가·마진은 포함하지 않습니다.');break;
   case 'inquiry':{
    const question=String(data.get('question')).trim();
    if(!question){say('Enter a question before confirming this local example.','로컬 예시 확인 전에 문의 내용을 입력하세요.',true);break;}
    state.question=question;
    const response=document.querySelector('#response .outcome p');if(response)response.textContent=tr('Original question: ','원래 문의: ')+question+'\n'+tr('Q-NS-09 · DEMO-2026-09104 / GCP-09 · local example; response shown above.','Q-NS-09 · DEMO-2026-09104 / GCP-09 · 로컬 예시 · 답변은 위에 표시됩니다.');
    say('LOCAL RECEIVED · Q-NS-09 / DEMO-2026-09104 / GCP-09. No external message sent. Continue to the illustrative response.','로컬 접수 · Q-NS-09 / DEMO-2026-09104 / GCP-09. 외부 메시지는 보내지 않았습니다. 예시 답변으로 이동하세요.');break;}
   case 'acknowledge':say('LOCAL ACKNOWLEDGED · Q-NS-09; document and question references preserved. No external ticket updated.','로컬 확인 완료 · Q-NS-09 · 문서·문의 참조 유지. 외부 티켓은 변경하지 않았습니다.');break;
  }
 });
})();
