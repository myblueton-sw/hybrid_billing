window.hb4=(()=>{
 const ko=document.documentElement.lang==='ko';
 const tr=(en,kr)=>ko?kr:en;
 const months=JSON.parse(document.querySelector('#month-data').textContent);
 const money=value=>value===null?tr('Not confirmed','미확정'):value.toLocaleString('en-US',{minimumFractionDigits:2,maximumFractionDigits:2});
 const screens=[...document.querySelectorAll('.screen')];
 const show=()=>{const active=screens.find(s=>s.id===location.hash.slice(1))||screens[0];screens.forEach(s=>{s.hidden=s!==active;s.querySelectorAll('nav a').forEach(a=>{if(a.hash==='#'+active.id)a.setAttribute('aria-current','page');else a.removeAttribute('aria-current');});});active.focus({preventScroll:true});window.scrollTo(0,0);};
 addEventListener('hashchange',show);show();
 document.querySelectorAll('form').forEach(f=>f.addEventListener('submit',e=>e.preventDefault()));
 const say=(screen,en,kr,error=false)=>{const r=screen.querySelector('[data-result]');r.textContent=tr(en,kr);r.classList.toggle('error',error);};
 return {ko,tr,months,money,say};
})();
