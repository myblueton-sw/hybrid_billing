import { strictJSON } from "../documents/json";
import { actions, bytes, choice, digest, fieldMap, freeze, id, integer, invalid, json, money, pageKeys, permissions, shape, type Envelope } from "./contract";
export type Binding=Readonly<{binding:string;binding_digest:string;term_revision:number;net_days:number;calendar_mode:"calendar";zone:string;from:string;to:string}>;
export type Context=Readonly<ReturnType<typeof parseContext>>;
const required=["mode","final_request","final_digest","final_state","customer_id","period","currency","scale","total","current_eligibility","readiness","submit_permission","approve_permission","submit_action","prepare_action","approve_action","step_up","snapshot","next_binding","binding_count"];
const optional=["customer_name","configuration","configuration_digest","languages","existing_draft","existing_digest"];
export function date(value:unknown,empty=false):value is string {
 return typeof value==="string"&&(empty&&value===""||/^(?!0000)\d{4}-\d{2}-\d{2}$/.test(value)&&Number.isFinite(Date.parse(value+"T00:00:00Z"))&&new Date(value+"T00:00:00Z").toISOString().slice(0,10)===value);
}
export function binding(raw:unknown):Binding {
 shape(raw,["binding","binding_digest","term_revision","net_days","calendar_mode","zone","from","to"]);
 if(!id(raw.binding)||!digest(raw.binding_digest)||raw.calendar_mode!=="calendar"||typeof raw.zone!=="string"||!raw.zone||bytes(raw.zone)>128||!date(raw.from)||!date(raw.to,true)||raw.to!==""&&raw.to<raw.from)return invalid();
 return freeze({...raw,term_revision:integer(raw.term_revision,2147483647,1),net_days:integer(raw.net_days,2147483647)}) as Binding;
}
function parseContext(e:Envelope,request:string,expectedDigest:string,after="",snapshot="") {
 const f=fieldMap(e.fields),keys=pageKeys(f,required,optional,"binding_");
 const m=money(f),rows=keys.map(k=>binding(json(f[k]))),count=integer(f.binding_count,100);
 if(f.mode!=="synthetic"||f.final_request!==request||f.final_digest!==expectedDigest||!id(request)||!digest(expectedDigest)||!digest(f.snapshot)||snapshot&&snapshot!==f.snapshot||count<rows.length||!after&&(rows.length!==Math.min(10,count)||f.next_binding!==(count>10?rows.at(-1)?.binding:""))||rows.some((r,i)=>r.binding<=(i?rows[i-1].binding:after))||f.next_binding!==""&&(rows.length!==10||f.next_binding!==rows.at(-1)?.binding))return invalid();
 const final_state=choice(f.final_state,["pending","approved","rejected","expired","superseded"] as const);
 const current_eligibility=choice(f.current_eligibility,["unchecked","eligible","ineligible","unavailable"] as const);
 const readiness=choice(f.readiness,["installation_held","unavailable","source_ineligible","configuration_missing","configuration_ambiguous","terms_missing","existing_draft","ready"] as const);
 const submit_permission=choice(f.submit_permission,permissions),approve_permission=choice(f.approve_permission,permissions);
 const submit_action=choice(f.submit_action,actions),prepare_action=choice(f.prepare_action,actions),approve_action=choice(f.approve_action,actions);
 const step_up=choice(f.step_up,["satisfied","reauthentication_required","authentication_policy_unsatisfied"] as const);
 let languages:readonly string[]=[];
 if(f.configuration!==undefined||f.configuration_digest!==undefined||f.languages!==undefined) {
  if(!id(f.configuration)||!digest(f.configuration_digest)||typeof f.languages!=="string")return invalid();
  let values:unknown;try{values=strictJSON(f.languages,4096);}catch{return invalid();}
  if(!Array.isArray(values)||values.length<1||values.length>3||new Set(values).size!==values.length||values.some(v=>!["en","ko","ar"].includes(v)))return invalid();languages=values;
 }
 if((f.existing_draft!==undefined||f.existing_digest!==undefined)&&(!id(f.existing_draft)||!digest(f.existing_digest)))return invalid();
 if(readiness==="installation_held"&&(current_eligibility!=="unchecked"||f.configuration!==undefined)||["installation_held","unavailable","source_ineligible","configuration_missing","configuration_ambiguous","terms_missing"].includes(readiness)&&(count!==0||rows.length!==0||f.next_binding!==""))return invalid();
 for(const [action,permission] of [[submit_action,submit_permission],[prepare_action,approve_permission],[approve_action,approve_permission]]) {
  if((action==="ready"||action==="recheck_required")&&(permission!=="allowed"||step_up!=="satisfied"||current_eligibility!=="eligible"||!f.configuration))return invalid();
 }
 if(submit_action==="ready"&&f.existing_draft!==undefined||(prepare_action==="ready"||approve_action==="ready"||approve_action==="recheck_required")&&(!f.existing_draft||count===0))return invalid();
 return freeze({revision:e.revision,at:e.at,final_request:request,final_digest:expectedDigest,final_state,...m,current_eligibility,readiness,submit_permission,approve_permission,submit_action,prepare_action,approve_action,step_up,configuration:f.configuration,configuration_digest:f.configuration_digest,languages,existing_draft:f.existing_draft,existing_digest:f.existing_digest,snapshot:f.snapshot,next:f.next_binding,count,rows});
}
export function contextOf(e:Envelope,request:string,digest:string,after="",snapshot=""):Context {return parseContext(e,request,digest,after,snapshot);}
