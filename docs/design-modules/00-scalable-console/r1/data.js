export const providers=['AWS','Azure','Google Cloud','OCI','Alibaba Cloud','IBM Cloud','VMware','NAVER Cloud','KT Cloud'];
export const states=['ready','pending','failed'];
export const population=screen=>screen.kind==='files'||/raw|intake|inbox|sources/.test(screen.id)?1000000:100000;
export const signature=q=>JSON.stringify([q.screen,q.account,q.search,q.provider,q.status,q.sort,q.size,q.snapshot]);
export function rowAt(index,account,customer=false){
 const id=String(index+1).padStart(7,'0');
 const safe={customer:'Customer '+String(index%1000+1).padStart(4,'0'),period:'2026-10',invoice:'DRAFT-'+id,due:'2026-11-15',created:'2026-10-31',effective:'2026-10-01',until:'2026-10-31',resource:'resource-'+id,sku:'SKU-'+(index%20+1),id:'REC-'+id,index,name:'Record '+id,account,service:['Compute','Storage','Network'][index%3],quantity:(index%240+1)+'.000000',unit:['hours','GB','GB'][index%3],date:'2026-10-'+String(index%28+1).padStart(2,'0'),status:states[index%3],currency:'USD',reference:'REF-'+id};
 if(customer)return safe;
 return {...safe,provider:providers[index%providers.length],file:'usage-'+id+'.parquet',size:(index%200+1)+' GB',revision:'r'+(index%7+1),owner:'Team '+(index%40+1),workspace:'Workspace '+(index%50+1),entity:'Entity '+(index%10+1),contract:'CON-'+String(index%100+1).padStart(5,'0'),source:'SRC-'+id,evidence:'EV-'+id,run:'RUN-'+id,version:'v'+(index%7+1),bytes:(index%200+1)+' GB',page:String(index+1),rows:String((index%100+1)*10000),schema:'FOCUS 1.1',timezone:'UTC',email:'user-'+id+'@example.invalid',actor:'Operator '+(index%20+1),region:['US','UAE','KR'][index%3]};
}
export async function queryRows(screen,q,cursor=null,signal){
 const pin=signature(q);
 if(cursor&&(cursor.pin!==pin||cursor.snapshot!==q.snapshot))throw new Error('expired');
 await new Promise(resolve=>setTimeout(resolve,60));
 if(signal?.aborted)throw new DOMException('Aborted','AbortError');
 const total=population(screen),descending=q.sort==='desc',customer=screen.group==='customer';
 let offset=cursor?.offset??0;
 const rows=[];const needle=q.search.trim().toLowerCase();
 const digits=needle.match(/^(?:rec-|record\s*)?(\d+)$/i);
 if(digits&&Number(digits[1])>0){
  const index=Number(digits[1])-1;
  if(index<total&&!cursor){const row=rowAt(index,q.account,customer);if((!q.provider||row.provider===q.provider)&&(!q.status||row.status===q.status))rows.push(row);}
  return {rows,total:rows.length,next:null,pin,snapshot:q.snapshot};
 }
 while(offset<total&&rows.length<=q.size){
  if(signal?.aborted)throw new DOMException('Aborted','AbortError');
  const position=offset++;const index=descending?total-1-position:position;
  if(q.provider&&providers[index%providers.length]!==q.provider)continue;
  if(q.status&&states[index%3]!==q.status)continue;
  const row=rowAt(index,q.account,customer);
  if(needle&&!Object.values(row).some(v=>String(v).toLowerCase().includes(needle))){if(offset%4096===0)await new Promise(resolve=>setTimeout(resolve,0));continue;}
  rows.push({...row,position});
 }
 const extra=rows.length>q.size?rows.pop():null;
 return {rows,total:!needle&&!q.provider&&!q.status?total:null,next:extra?{offset:extra.position,pin,snapshot:q.snapshot}:null,pin,snapshot:q.snapshot};
}
export function accountPage(search,page){
 const needle=search.trim().toLowerCase();const match=needle.match(/(?:acc-|account\s*|계정\s*)?(\d+)$/);
 if(match){const n=Number(match[1]);return {items:n>=1&&n<=100000?[{id:'ACC-'+String(n).padStart(6,'0'),name:'Account '+String(n).padStart(6,'0')}]:[],more:false};}
 if(needle&&!['account','acc','계정','حساب'].some(x=>x.includes(needle)))return {items:[],more:false};
 const start=page*20;return {items:Array.from({length:Math.min(20,100000-start)},(_,i)=>({id:'ACC-'+String(start+i+1).padStart(6,'0'),name:'Account '+String(start+i+1).padStart(6,'0')})),more:start+20<100000};
}
