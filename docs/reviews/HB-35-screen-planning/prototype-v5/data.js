window.hb5=(()=>{
 const ko=document.documentElement.lang==='ko';
 const tr=(en,kr)=>ko?kr:en;
 const esc=s=>String(s??'').replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));
 const users=[{id:'demo-am',name:'Alex Morgan'},{id:'demo-jk',name:'Jamie Kim'}];
 const companies=[
  {id:'northstar',name:'Northstar Labs',msp:'meridian',country:'US',currency:'USD',legal:'Northstar Labs Inc.',address:'100 Example Avenue, Boston, MA · SAMPLE',tax:'DEMO-US-1001',contact:'Taylor Reed',email:'billing@northstar.example',account:'AWS · Azure · GCP · OCI · IBM · VMware',contract:'CT-NS',version:'CT-NS-v3',renewal:'2027-01-01',notice:'2026-11-01',terms:'Net 30',rate:tr('Version p4 / contracted selling rates; see original rate evidence','버전 p4 / 계약 판매 요율 · 기존 요율 근거 연결')},
  {id:'juniper',name:'Juniper Retail',msp:'meridian',country:'US',currency:'USD',legal:'Juniper Retail LLC',address:'20 Sample Street, Austin, TX · SAMPLE',tax:'DEMO-US-1002',contact:'Jordan Lee',email:'ap@juniper.example',account:'AWS · Azure',contract:'CT-JR',version:'CT-JR-v2',renewal:'2027-03-01',notice:'2027-01-01',terms:'Net 30',rate:tr('Fixed monthly demo charge USD 24,800; no additional support line','월 고정 예시 청구액 USD 24,800 · 지원료 별도 가산 없음')},
  {id:'aster',name:'Aster Health',msp:'meridian',country:'US',currency:'USD',legal:'Aster Health Inc.',address:'80 Example Road, Denver, CO · SAMPLE',tax:'DEMO-US-1003',contact:'Casey Chen',email:'finance@aster.example',account:'GCP',contract:'CT-AH',version:'CT-AH-v1',renewal:'2027-02-01',notice:'2026-12-01',terms:'Net 30',rate:tr('Contracted monthly sample USD 21,000','계약 월정액 예시 USD 21,000')},
  {id:'harbor-ns',name:'Northstar Labs',msp:'harbor',country:'GB',currency:'GBP',legal:'Northstar Labs UK Ltd.',address:'12 Example Lane, London · SAMPLE',tax:'DEMO-GB-1001',contact:'Morgan Ellis',email:'billing@northstar-uk.example',account:'Azure',contract:'CT-HN',version:'CT-HN-v1',renewal:'2027-04-01',notice:'2027-02-01',terms:'Net 30',rate:tr('Contracted monthly sample GBP 18,000','계약 월정액 예시 GBP 18,000')}
 ];
 const msps={meridian:'Meridian MSP',harbor:'Harbor MSP'};
 const policies={meridian:{issuer:'optional',reviewer:'required',version:1},harbor:{issuer:'optional',reviewer:'required',version:1}};
 const records=[];
 const actor=(id,time)=>({...users.find(u=>u.id===id),time});
 for(const month of ['2026-08','2026-07','2026-06']){
  for(const c of companies){
   const pending=c.id==='juniper'&&month==='2026-08';
   const sale=c.id==='northstar'?({'2026-08':130000,'2026-07':125000,'2026-06':120000}[month]):c.id==='juniper'?24800:c.id==='aster'?21000:18000;
   const next=month==='2026-08'?'2026-09':month==='2026-07'?'2026-08':'2026-07';
   const r={id:c.id+'-'+month,company:c.id,month,currency:c.currency,sale,status:pending?'draft':'issued',number:pending?null:'DEMO-'+c.id.toUpperCase()+'-'+month.replace('-',''),issuedAt:pending?null:next+'-05T09:00:00Z',contract:c.version,calculation:1,revision:1,issuer:'',reviewer:pending?'':'demo-jk',policy:{...policies[c.msp]},review:null,issue:null,events:[],sourceValid:true};
   if(!pending){r.review={...actor('demo-jk',next+'-04T08:30:00Z'),signature:'historical'};r.issue=actor('demo-am',r.issuedAt);r.events=[{kind:'review',...r.review,detail:tr('Historical fixed review','당시 검수 기록 보존')},{kind:'issue',...r.issue,detail:r.number}];}
   if(c.id==='northstar'&&month!=='2026-06'){r.number=month==='2026-08'?'DEMO-2026-08102':'DEMO-2026-07101';r.events.at(-1).detail=r.number;}
   records.push(r);
  }
 }
 const state={user:'demo-am',mode:'msp',msp:'meridian',company:'all',month:'all',status:'all',person:'all',showPeople:true,policies,records,renewals:[],policyEvents:[],message:'',route:'register',selected:'juniper-2026-08'};
 const company=id=>companies.find(c=>c.id===id);
 const record=id=>records.find(r=>r.id===id);
 const person=id=>users.find(u=>u.id===id)?.name||tr('Unassigned','미지정');
 const money=(v,currency)=>currency+' '+v.toLocaleString('en-US',{minimumFractionDigits:2,maximumFractionDigits:2});
 const signature=r=>[r.revision,r.contract,r.calculation,r.policy.version,r.issuer,r.reviewer].join('|');
 const stamp=()=>({...users.find(u=>u.id===state.user),time:new Date().toISOString()});
 return {ko,tr,esc,users,companies,msps,state,company,record,person,money,signature,stamp};
})();
