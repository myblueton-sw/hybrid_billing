import { AuthorityError,bytes,capability,ceilingOf,digest,fail,frozen,historyOf,id,instant,policyOf,recipientOf,resultDomain,revision,type AuthorityDomain,type Capability,type Envelope,type Read,type Recipient,type Ceiling } from "./contract";
export type Cursor=Readonly<{after:string;snapshot:string}>;
function rowsOf(envelope:Envelope,workspace:string,actor:string,fixed:readonly string[],cursor?:Cursor){
 const f=envelope.fields;
 if(!id(workspace)||!id(actor)||!revision(envelope.revision)||instant(envelope.at)===undefined||f.mode!=="synthetic"||f.workspace!==workspace||f.actor!==actor||f.current_eligibility!=="unchecked"||!digest(f.snapshot)||!(f.next===""||id(f.next)))return fail();
 if(cursor&&(!id(cursor.after)||!digest(cursor.snapshot)||f.snapshot!==cursor.snapshot))throw new AuthorityError("evidence_stale");
 const names=Object.keys(f).filter(k=>!fixed.includes(k)).sort();
 if(fixed.some(k=>!Object.hasOwn(f,k))||names.length>10||names.some((k,i)=>k!==`row_${String(i).padStart(2,"0")}`)||Object.values(f).some(v=>typeof v!=="string"||bytes(v)>4096))return fail();
 return names.map(k=>f[k]);
}
function ordered<T>(rows:readonly T[],key:(row:T)=>string,next:string,cursor?:Cursor){
 let previous=cursor?.after??"";
 for(const row of rows){const current=key(row);if(current<=previous)return fail();previous=current;}
 if(next&&(rows.length!==10||next!==previous))return fail();
 return rows;
}
const base=["mode","workspace","actor","current_eligibility","snapshot","next"];
const fields=(cursor?:Cursor):Record<string,string>=>cursor?{after:cursor.after,snapshot:cursor.snapshot}:{};
export async function historyPage(read:Read,workspace:string,actor:string,cursor?:Cursor,d:AuthorityDomain=resultDomain){
 const e=await read(`${d.namespace}.history`,workspace,fields(cursor));
 const rows=rowsOf(e,workspace,actor,base,cursor).map(value=>historyOf(value,d));
 ordered(rows,row=>row.proposal,e.fields.next,cursor);
 return frozen({revision:e.revision,at:e.at,snapshot:e.fields.snapshot,next:e.fields.next,rows});
}
export async function contextPage(read:Read,workspace:string,actor:string,collection:"recipients"|"ceilings",selected:Capability,cursor?:Cursor,d:AuthorityDomain=resultDomain){
 if(!capability(selected,d))throw new AuthorityError("invalid_authority_request");
 const e=await read(`${d.namespace}.context`,workspace,{collection,capability:selected,...fields(cursor)});
 if(e.fields.collection!==collection||e.fields.capability!==selected)return fail();
 const raw=rowsOf(e,workspace,actor,[...base,"collection","capability","policy"],cursor),policy=policyOf(e.fields.policy,d);
 const rows: readonly (Recipient|Ceiling)[]=collection==="recipients"?raw.map(value=>recipientOf(value,actor)):raw.map(value=>ceilingOf(value,selected,d));
 ordered(rows,row=>"user" in row?row.user:row.ceiling,e.fields.next,cursor);
 return frozen({namespace:d.namespace,revision:e.revision,at:e.at,snapshot:e.fields.snapshot,next:e.fields.next,collection,capability:selected,policy,rows});
}
export type HistoryPage=Awaited<ReturnType<typeof historyPage>>;
export type ContextPage=Awaited<ReturnType<typeof contextPage>>;
