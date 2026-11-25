import { decimal } from "../documents/contract";
import { date } from "./context";
import { bytes, choice, digest, fieldMap, freeze, id, integer, invalid, json, money, observations, pageKeys, permissions, reviews, shape, submissions, timestamp, type Envelope } from "./contract";
export type Action="submit"|"prepare_approval"|"approve";
export type Recovery=Readonly<{operation:Action;original_actor:string;business_id:string;state:string;permission:typeof permissions[number];arguments:Readonly<Record<string,string>>}>;
export type Review=Readonly<{id:string;digest:string;actor:string;state:typeof reviews[number];reviewed_at:string;refusal?:string}>;
export type Line=Readonly<{index:number;text:string;amount:string;tax:boolean}>;
export type Reference=Readonly<{field:string;subject:string;change:string;ref_id:string;fingerprint:string;effective:string}>;
export type Coverage=Readonly<{contract:string;binding:string;term:string;revision:number;net_days:number;calendar_mode:"calendar";zone:string;issue_date:string;due_date:string}>;
export type Inspection=Readonly<ReturnType<typeof parseInspection>>;
export function argumentsOf(action:Action,raw:unknown):Readonly<Record<string,string>> {
 const f=fieldMap(raw),keys=action==="submit"?["configuration","configuration_digest","final_request","final_digest","language"]:action==="prepare_approval"?["draft","draft_digest","binding","binding_digest"]:["digest"];
 shape(f,keys);
 for(const [key,value] of Object.entries(f)) {
  if(key==="language"?!["en","ko","ar"].includes(value):key.includes("digest")?!digest(value):!id(value))return invalid();
 }
 return freeze({...f});
}
function recovery(raw:string,action:Action,actor:string,owner:string,business:string,state:string):Recovery {
 const v=json(raw);shape(v,["operation","original_actor","business_id","state","permission","arguments"]);
 if(v.operation!==action||v.original_actor!==owner||v.business_id!==business||v.state!==state||!id(owner)||!id(business)||v.permission==="allowed"&&actor!==owner)return invalid();
 return freeze({...v,permission:choice(v.permission,permissions),arguments:argumentsOf(action,v.arguments)}) as Recovery;
}
function reviewOf(raw:string):Review {
 const r=json(raw);shape(r,["id","digest","actor","state","reviewed_at"],["refusal"]);
 const state=choice(r.state,reviews);
 if(!id(r.id)||!digest(r.digest)||!id(r.actor)||!timestamp(r.reviewed_at)||(state==="refused"?(typeof r.refusal!=="string"||!["reference_changed","coverage_mismatch"].includes(r.refusal)):r.refusal!==undefined))return invalid();
 return freeze({...r,state}) as Review;
}
function reference(raw:string):Reference {
 const r=json(raw);shape(r,["field","subject","change","ref_id","fingerprint","effective"]);
 if(typeof r.field!=="string"||!["payment_account","document_recipient","document_template"].includes(r.field)||![r.subject,r.change,r.ref_id].every(id)||!digest(r.fingerprint)||!timestamp(r.effective))return invalid();return freeze(r) as Reference;
}
function coverage(raw:string):Coverage {
 const c=json(raw);shape(c,["contract","binding","term","revision","net_days","calendar_mode","zone","issue_date","due_date"]);
 if(![c.contract,c.binding,c.term].every(id)||c.calendar_mode!=="calendar"||typeof c.zone!=="string"||!c.zone||bytes(c.zone)>128||!date(c.issue_date)||!date(c.due_date)||c.due_date<c.issue_date)return invalid();
 return freeze({...c,revision:integer(c.revision,2147483647,1),net_days:integer(c.net_days,2147483647)}) as Coverage;
}
const required=["mode","current_eligibility","snapshot","next_line","line_count","draft","digest","origin","final_request","final_digest","customer_id","period","currency","scale","total","language","submitter","final_requester","submission_state","module_observation","receipt"];
const optional=["customer_name","submitted_at","configuration","configuration_digest","review","reference_00","reference_01","reference_02","coverage","recovery_submit","recovery_prepare","recovery_approve"];
function parseInspection(e:Envelope,target:string,actor:string,after="",snapshot="") {
 const f=fieldMap(e.fields),keys=pageKeys(f,required,optional,"line_");
 const m=money(f,6),count=integer(f.line_count,200,1),start=after===""?0:integer(after,199)+1;
 if(f.mode!=="synthetic"||f.current_eligibility!=="unchecked"||f.draft!==target||!id(target)||!digest(f.digest)||!id(f.final_request)||!digest(f.final_digest)||![f.submitter,f.final_requester].every(id)||!digest(f.snapshot)||snapshot&&snapshot!==f.snapshot||!id(actor)||f.submitted_at!==undefined&&!timestamp(f.submitted_at))return invalid();
 const origin=choice(f.origin,["configured","legacy"] as const),submission_state=choice(f.submission_state,submissions),module_observation=choice(f.module_observation,observations),language=choice(f.language,["en","ko","ar"] as const);
 if(origin==="configured"? !id(f.configuration)||!digest(f.configuration_digest):f.configuration!==undefined||f.configuration_digest!==undefined)return invalid();
 const review=f.review===undefined?undefined:reviewOf(f.review);
 if(review&&submission_state!=="submitted")return invalid();
 const refs=review?[reference(f.reference_00),reference(f.reference_01),reference(f.reference_02)]:[];
 const due=review?coverage(f.coverage):undefined;
 if(!review&&[f.reference_00,f.reference_01,f.reference_02,f.coverage].some(v=>v!==undefined)||review&&new Set(refs.map(r=>r.field)).size!==3||f.receipt!==(review?.state==="approved"?"historical_approval":review?"review_attempt":"submission_record"))return invalid();
 const lines=keys.map((k,i):Line=>{const r=json(f[k]);shape(r,["index","text","amount","tax"]);const index=integer(r.index,199);if(index!==start+i||typeof r.text!=="string"||!r.text||bytes(r.text)>4000||!decimal(r.amount,m.scale,true)||typeof r.tax!=="boolean")return invalid();return freeze({...r,index}) as Line;});
 if(start>=count||lines.length!==Math.min(10,count-start)||f.next_line!==(start+lines.length<count?String(start+lines.length-1):""))return invalid();
 const recoveries:Recovery[]=[];
 const submitRecovery=origin==="configured"&&submission_state!=="submitted";
 const reviewRecovery=origin==="configured"&&review&&(review.state==="prepared"||review.state==="approval_unknown");
 if(submitRecovery) {
  const r=recovery(f.recovery_submit,"submit",actor,f.submitter,target,submission_state);
  if(r.arguments.configuration!==f.configuration||r.arguments.configuration_digest!==f.configuration_digest||r.arguments.final_request!==f.final_request||r.arguments.final_digest!==f.final_digest||r.arguments.language!==language)return invalid();recoveries.push(r);
 }else if(f.recovery_submit!==undefined)return invalid();
 if(reviewRecovery&&review) {
  const p=recovery(f.recovery_prepare,"prepare_approval",actor,review.actor,review.id,review.state),a=recovery(f.recovery_approve,"approve",actor,review.actor,review.id,review.state);
  if(p.arguments.draft!==target||p.arguments.draft_digest!==f.digest||a.arguments.digest!==review.digest)return invalid();recoveries.push(p,a);
 }else if(f.recovery_prepare!==undefined||f.recovery_approve!==undefined)return invalid();
 return freeze({revision:e.revision,at:e.at,draft:target,digest:f.digest,origin,final_request:f.final_request,final_digest:f.final_digest,...m,language,submitter:f.submitter,final_requester:f.final_requester,submission_state,module_observation,receipt:f.receipt,submitted_at:f.submitted_at,configuration:f.configuration,configuration_digest:f.configuration_digest,review,references:refs,coverage:due,recoveries,snapshot:f.snapshot,next:f.next_line,count,lines});
}
export function inspectionOf(e:Envelope,target:string,actor:string,after="",snapshot=""):Inspection {return parseInspection(e,target,actor,after,snapshot);}
