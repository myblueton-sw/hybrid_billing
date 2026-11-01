export const reviewStorageKey='hybrid-billing.screen-review.v1';
const allowedKeys=['screen','revision','locale','layout','state','items','confirmedAt','source','width'];
export function validReview(record){
 return record&&Object.keys(record).every(key=>allowedKeys.includes(key))&&typeof record.screen==='string'&&/^[a-z0-9-]{1,100}$/.test(record.screen)&&typeof record.revision==='string'&&/^[a-f0-9]{64}$/.test(record.revision)&&['en','ko','ar'].includes(record.locale)&&['compact','medium','wide'].includes(record.layout)&&typeof record.state==='string'&&/^[a-f0-9]{64}$/.test(record.state)&&Array.isArray(record.items)&&record.items.length>0&&record.items.length<=500&&new Set(record.items).size===record.items.length&&record.items.every(id=>typeof id==='string'&&id.startsWith(record.screen+':')&&/^[a-zA-Z0-9:_-]{1,250}$/.test(id))&&typeof record.confirmedAt==='string'&&Number.isFinite(Date.parse(record.confirmedAt))&&record.source==='local-unauthenticated-confirmation'&&Number.isInteger(record.width)&&record.width>0&&record.width<20000;
}
export function readReviews(storage){
 try{
  const raw=storage.getItem(reviewStorageKey);
  if(raw==null)return {records:[],error:null};
  if(raw.length>5000000)throw Error('size');
  const data=JSON.parse(raw);
  if(data.version!==1||!Array.isArray(data.records)||data.records.length>5000||!data.records.every(validReview))throw Error('format');
  return {records:data.records,error:null};
 }catch{return {records:[],error:'storage'};}
}
export function saveReview(storage,record){
 if(!validReview(record))return {ok:false,error:'format'};
 const current=readReviews(storage);
 if(current.error)return {ok:false,error:current.error};
 if(current.records.length>=5000)return {ok:false,error:'capacity'};
 const same=current.records.some(item=>item.screen===record.screen&&item.revision===record.revision&&item.locale===record.locale&&item.layout===record.layout&&item.width===record.width&&item.state===record.state&&JSON.stringify(item.items)===JSON.stringify(record.items));
 const records=same?current.records:[...current.records,record];
 try{
  const encoded=JSON.stringify({version:1,records});
  if(encoded.length>5000000)return {ok:false,error:'capacity'};
  storage.setItem(reviewStorageKey,encoded);
  const persisted=readReviews(storage);
  if(persisted.error||JSON.stringify(persisted.records)!==JSON.stringify(records))return {ok:false,error:'storage'};
  return {ok:true,records};
 }catch{return {ok:false,error:'storage'};}
}
export function reviewStatus(records,screen,revision,locale,layout,current=null){
 const scoped=records.filter(record=>record.screen===screen&&record.locale===locale&&record.layout===layout);
 const matching=scoped.filter(record=>record.revision===revision);
 if(matching.length){
  if(!current)return 'reviewedScope';
  return matching.some(record=>record.width===current.width&&record.state===current.state&&JSON.stringify(record.items)===JSON.stringify(current.items))?'localFix':'otherState';
 }
 if(scoped.length)return 'changed';
 if(records.some(record=>record.screen===screen))return 'otherVariant';
 return 'draft';
}
