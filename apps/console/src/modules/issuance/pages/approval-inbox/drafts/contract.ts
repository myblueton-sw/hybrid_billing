import { strictJSON } from "../documents/json";
import { timestamp, decimal } from "../documents/contract";
import { bytes, digest, freeze, id, object, revision } from "../results/contract";
export { bytes, digest, freeze, id, object, revision, timestamp };
export class DraftError extends Error {
 constructor(readonly code:string, readonly effect:"none"|"unknown"="none", readonly operation="") { super(code); }
}
export const invalid=():never=>{throw new DraftError("invalid_draft_response");};
export function shape(raw:unknown, required:readonly string[], optional:readonly string[]=[]):asserts raw is Record<string,unknown> {
 if(!object(raw)||required.some(k=>!Object.hasOwn(raw,k))||Object.keys(raw).some(k=>!required.includes(k)&&!optional.includes(k)))invalid();
}
export function fieldMap(raw:unknown):Record<string,string> {
 if(!object(raw)||Object.keys(raw).length>64||Object.entries(raw).some(([k,v])=>!/^[a-z][a-z0-9_]*$/.test(k)||typeof v!=="string"||bytes(v)>4096)||bytes(JSON.stringify(raw))>65536)return invalid();
 return raw as Record<string,string>;
}
export function json(raw:unknown):Record<string,unknown> {
 if(typeof raw!=="string"||bytes(raw)>4096)return invalid();
 let value:unknown;
 try{value=strictJSON(raw,4096);}catch{return invalid();}
 if(!object(value))return invalid();return value;
}
export function integer(value:unknown,max:number,min=0):number {
 if(typeof value!=="string"||!/^(?:0|[1-9][0-9]*)$/.test(value)||!Number.isSafeInteger(Number(value))||Number(value)<min||Number(value)>max)return invalid();return Number(value);
}
export function choice<T extends string>(value:unknown, allowed:readonly T[]):T {
 if(typeof value!=="string"||!allowed.includes(value as T))return invalid();return value as T;
}
export const submissions=["submitted","submission_unknown","submission_refused"] as const;
export const reviews=["prepared","approval_unknown","approved","refused"] as const;
export const observations=["missing","matching_unapproved","matching_approved","conflict","unavailable"] as const;
export const permissions=["allowed","denied","unavailable"] as const;
export const actions=["ready","permission_denied","authority_unavailable","installation_held","reauthentication_required","authentication_policy_unsatisfied","source_ineligible","setup_missing","setup_ambiguous","evidence_unavailable","existing_draft","draft_required","independent_reviewer_required","owned_by_other","unresolved_operation","review_required","recheck_required","completed","legacy_read_only"] as const;
export type Summary=Readonly<{draft:string;digest:string;customer_id:string;customer_name?:string;period:string;currency:string;scale:number;total:string;submitter:string;submission_state:typeof submissions[number];review_state:"none"|typeof reviews[number];review_actor?:string;module_observation:typeof observations[number];submitted_at?:string}>;
export const summaryKeys=["draft","digest","customer_id","period","currency","scale","total","submitter","submission_state","review_state","module_observation"];
export function money(raw:Record<string,unknown>,maxScale=8) {
 const scale=integer(raw.scale,maxScale);
 if(!decimal(raw.total,scale)||typeof raw.currency!=="string"||!/^[A-Z]{3}$/.test(raw.currency)||!id(raw.customer_id)||typeof raw.period!=="string"||!/^(?!0000)\d{4}-(?:0[1-9]|1[0-2])$/.test(raw.period))return invalid();
 if(raw.customer_name!==undefined&&(typeof raw.customer_name!=="string"||!raw.customer_name||bytes(raw.customer_name)>256))return invalid();
 return {scale,total:raw.total,currency:raw.currency,customer_id:raw.customer_id,period:raw.period,...(raw.customer_name!==undefined?{customer_name:raw.customer_name as string}:{})};
}
export function summary(raw:unknown):Summary {
 shape(raw,summaryKeys,["customer_name","review_actor","submitted_at"]);
 const m=money(raw,6),state=choice(raw.review_state,["none",...reviews]);
 if(!id(raw.draft)||!digest(raw.digest)||!id(raw.submitter)||(state==="none"?raw.review_actor!==undefined:!id(raw.review_actor))||raw.submitted_at!==undefined&&!timestamp(raw.submitted_at))return invalid();
 return freeze({...raw,...m,submission_state:choice(raw.submission_state,submissions),review_state:state,module_observation:choice(raw.module_observation,observations)}) as Summary;
}
export function fixedMoney(total:string,scale:number):string {
 if(!Number.isSafeInteger(scale)||scale<0||scale>8||!decimal(total,scale))return invalid();
 const [whole,fraction=""]=total.split(".");return scale?`${whole}.${fraction.padEnd(scale,"0")}`:whole;
}
export type Envelope=Readonly<{revision:string;at:string;fields:Readonly<Record<string,string>>}>;
export type Page<T>=Readonly<{revision:string;snapshot:string;next:string;count:number;rows:readonly T[]}>;
export function pageKeys(f:Record<string,string>,required:readonly string[],optional:readonly string[],prefix:string):string[] {
 const keys=Object.keys(f).filter(k=>!required.includes(k)&&!optional.includes(k)&&k.startsWith(prefix)).sort();
 if(keys.length>10||keys.some((k,i)=>k!==`${prefix}${String(i).padStart(2,"0")}`))return invalid();
 shape(f,[...required,...keys],optional);return keys;
}
export function listPage(e:Envelope,after="",snapshot=""):Page<Summary> {
 const f=fieldMap(e.fields),keys=pageKeys(f,["mode","current_eligibility","snapshot","next","count"],[],"row_");
 const rows=keys.map(k=>summary(json(f[k]))),count=integer(f.count,100);
 if(f.mode!=="synthetic"||f.current_eligibility!=="unchecked"||!digest(f.snapshot)||snapshot&&f.snapshot!==snapshot||count<rows.length||!after&&(rows.length!==Math.min(10,count)||f.next!==(count>10?rows.at(-1)?.draft:""))||count===0&&f.next!==""||rows.some((r,i)=>r.draft<=(i?rows[i-1].draft:after))||f.next!==""&&(rows.length!==10||f.next!==rows.at(-1)?.draft))return invalid();
 return freeze({revision:e.revision,snapshot:f.snapshot,next:f.next,count,rows});
}
