import type {Session} from "../../../../../service/api";
import type {Attempt} from "../../../../../service/admin";
import {strictJSON} from "../../../../../service/strict-json";
import {AuthorityError,bytes,capability,digest,exact,frozen,id,instant,object,revision,rationaleValid} from "./contract";
import {inspectionOf,type Inspection} from "./inspection";
import {bounded,checkSession,fieldMap,responseText} from "./transport";
export type Action="propose_console"|"propose_human"|"approve"|"reject"|"withdraw";
export type Intent=Readonly<{action:Action;target:string;fields:Readonly<Record<string,string>>;rationale:string;revision:string;attempt:Readonly<Attempt>}>;
export type Receipt=Readonly<{inspection:Inspection;replay:boolean;operation:string}>;
export function freezeIntent(input:Intent):Intent{
 const invalid=()=>{throw new AuthorityError("invalid_authority_request");};
 if(!["propose_console","propose_human","approve","reject","withdraw"].includes(input.action)||!id(input.target)||!revision(input.revision)||!id(input.attempt?.request)||!id(input.attempt?.key)||!rationaleValid(input.rationale))return invalid();
 const proposal=input.action.startsWith("propose_");
 try{exact(input.fields,proposal?["capability","not_before","not_after",...(input.action==="propose_human"?["recipient","ceiling"]:[])]:["digest"]);}catch{return invalid();}
 const f=input.fields;
 if(proposal){
  if(!capability(f.capability)||instant(f.not_before)===undefined||instant(f.not_after)===undefined||instant(f.not_before)!>=instant(f.not_after)!)return invalid();
  if(input.action==="propose_human"&&(!id(f.recipient)||!/^result-grant-[0-9a-f]{64}$/.test(f.ceiling)))return invalid();
 }else if(!digest(f.digest))return invalid();
 return frozen(structuredClone(input));
}
const refusalRules: Record<string, [number, string]> = { invalid_stage: [422,"correct_request"], permission_denied: [403,"none"], self_approval_denied: [403,"none"], unauthenticated: [401,"obtain_new_access_answer"], csrf_refused: [403,"none"], reauthentication_required: [403,"reauthenticate"], evidence_stale: [409,"inspect_current_state"], revision_conflict: [409,"inspect_current_state"], invalid_request: [422,"correct_request"], prerequisite_failed: [422,"correct_request"], rate_limited: [429,"retry_later"], unsupported_capability: [501,"none"] };
function cleanRefusal(raw: Record<string, unknown>, code: string, status: number, request: string) {
 const rule = refusalRules[code];
 if (!rule || raw.request_id !== request || raw.operation_id !== undefined || status !== rule[0] || raw.next_action !== rule[1] || raw.retry !== (code === "rate_limited") || Object.keys(raw).some(k => !["code", "retry", "next_action", "request_id", "fields", "retry_after_ms"].includes(k))) return false;
 return (raw.fields === undefined || Array.isArray(raw.fields) && raw.fields.length <= 28 && raw.fields.every(v => typeof v === "string" && bytes(v) <= 128)) && (raw.retry_after_ms === undefined || Number.isSafeInteger(raw.retry_after_ms) && Number(raw.retry_after_ms) >= 0);
}
export function mutationTransport(session:Session,fetcher:typeof fetch=fetch.bind(globalThis)){
 const owner=frozen({...session});
 return async(input:Intent):Promise<Receipt>=>{
  const intent=freezeIntent(input),tried=intent.attempt;
  await checkSession(owner,fetcher);
  let response:Response,raw:unknown;
  try{
   const result=await bounded(async signal=>{
    const response=await fetcher(`/api/v1/commands/result_authority.${intent.action}`,{signal,method:"POST",credentials:"same-origin",cache:"no-store",headers:{"Content-Type":"application/json","X-CSRF-Token":owner.csrf,"X-Request-Id":tried.request},body:JSON.stringify({request_id:tried.request,target_id:intent.target,idempotency_key:tried.key,expected_revision:intent.revision,rationale:intent.rationale,body:intent.fields})});
    return {response,raw:strictJSON(await responseText(response),100000)};
   },"unknown");response=result.response;raw=result.raw;
  }catch(e){throw new AuthorityError(e instanceof AuthorityError?e.code:"service_unavailable","unknown");}
  if(!object(raw))throw new AuthorityError("invalid_authority_response","unknown");
  const operation=raw.request_id===tried.request&&id(raw.operation_id)?raw.operation_id:"";
  if(!response.ok){
   const code=typeof raw.code==="string"&&/^[a-z_]{1,80}$/.test(raw.code)?raw.code:"service_unavailable";
   throw new AuthorityError(code,cleanRefusal(raw,code,response.status,tried.request)?"none":"unknown",operation);
  }
  try{
   exact(raw,["request_id","status","operation_id","target_id","result_revision","business_state","replay","as_of","fields"]);
   if(response.status!==200||raw.request_id!==tried.request||raw.target_id!==intent.target||raw.status!=="completed"||!id(raw.operation_id)||!revision(raw.result_revision)||typeof raw.replay!=="boolean"||instant(raw.as_of)===undefined)throw new Error();
   const detail=inspectionOf({revision:raw.result_revision,at:raw.as_of as string,fields:fieldMap(raw.fields)},intent.target,intent.action.startsWith("propose_")?undefined:intent.fields.digest);
   const row=detail.row,f=intent.fields;
   if(raw.business_state!==row.state)throw new Error();
   if(intent.action.startsWith("propose_")){
    if(row.proposer!==owner.user||row.kind!==(intent.action==="propose_console"?"console_ceiling":"human_grant")||row.capability!==f.capability||row.recipient!==(intent.action==="propose_console"?"console":f.recipient)||detail.ceiling!==(f.ceiling??"")||instant(detail.notBefore)!==instant(f.not_before)||instant(detail.notAfter)!==instant(f.not_after)||detail.rationale!==intent.rationale)throw new Error();
   }else{
    const expected={approve:"applied",reject:"rejected",withdraw:"withdrawn"}[intent.action as "approve"|"reject"|"withdraw"];
    if(row.state!==expected||detail.decision?.actor!==owner.user||detail.decision.rationale!==intent.rationale)throw new Error();
   }
   return frozen({inspection:detail,replay:raw.replay,operation:raw.operation_id});
  }catch{throw new AuthorityError("invalid_authority_receipt","unknown",operation);}
 };
}
