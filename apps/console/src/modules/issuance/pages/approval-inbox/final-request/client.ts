import type { Session } from "../../../../../service/api";
import { attempt } from "../../../../../service/admin";
import { timestamp } from "../documents/contract";
import { strictJSON } from "../documents/json";
import { digest, exact, freeze, id, object, revision, ResultError } from "../results/contract";
import { bounded, checkSession, responseText } from "../results/transport";
import { freezeIntent, previewOf, validScope, type Intent, type Preview, type Receipt, type Scope } from "./contract";
const refusals: Record<string, readonly [number,string]> = {permission_denied:[403,"none"],self_approval_denied:[403,"none"],unauthenticated:[401,"obtain_new_access_answer"],csrf_refused:[403,"none"],reauthentication_required:[403,"reauthenticate"],evidence_stale:[409,"inspect_current_state"],revision_conflict:[409,"inspect_current_state"],invalid_request:[422,"correct_request"],prerequisite_failed:[422,"correct_request"],rate_limited:[429,"retry_later"],unsupported_capability:[501,"none"]};
export function finalRequestClient(session: Session, f: typeof fetch = fetch.bind(globalThis)) {
 const owner=freeze({...session});
 async function send(scope:Scope,target:string,intent?:Intent) {
  if(!validScope(scope)||!id(target))throw new ResultError("invalid_final_request");
  await checkSession(owner,f);
  const tried=intent?.attempt??attempt(), effect=intent?"unknown":"none";
  let response:Response,raw:unknown;
  try {
   const result=await bounded(async signal=>{
    const response=await f(`/api/v1/commands/erp_final.${intent?"request":"preview"}`,{signal,method:"POST",credentials:"same-origin",cache:"no-store",headers:{"Content-Type":"application/json","X-CSRF-Token":owner.csrf,"X-Request-Id":tried.request},body:JSON.stringify({request_id:tried.request,target_id:target,...(intent?{idempotency_key:tried.key,expected_revision:intent.preview.revision,rationale:intent.rationale}:{}),body:{result:scope.result,digest:scope.digest}})});
    return {response,raw:strictJSON(await responseText(response,32768),32768)};
   },effect);
   response=result.response;raw=result.raw;
  }catch(error){throw new ResultError(error instanceof ResultError?error.code:"service_unavailable",effect);}
  if(!object(raw))throw new ResultError("invalid_final_response",effect);
  const operation=raw.request_id===tried.request&&id(raw.operation_id)?raw.operation_id:"";
  if(!response.ok){
   const code=typeof raw.code==="string"&&/^[a-z_]{1,80}$/.test(raw.code)?raw.code:"service_unavailable",rule=refusals[code];
   const clean=!!rule&&response.status===rule[0]&&raw.next_action===rule[1]&&raw.retry===(code==="rate_limited")&&raw.request_id===tried.request&&raw.operation_id===undefined&&Object.keys(raw).every(k=>["code","retry","next_action","request_id","fields","retry_after_ms"].includes(k))&&(raw.fields===undefined||Array.isArray(raw.fields)&&raw.fields.length<=24&&raw.fields.every(v=>typeof v==="string"&&v.length<=128))&&(raw.retry_after_ms===undefined||Number.isSafeInteger(raw.retry_after_ms)&&Number(raw.retry_after_ms)>=0);
   throw new ResultError(code,intent&&!clean?"unknown":"none",operation);
  }
  try {
   exact(raw,intent?["request_id","status","operation_id","target_id","result_revision","business_state","replay","as_of","fields"]:["request_id","status","target_id","result_revision","replay","as_of","fields"]);
   if(response.status!==200||raw.request_id!==tried.request||raw.target_id!==target||raw.status!=="completed"||!revision(raw.result_revision)||!timestamp(raw.as_of)||typeof raw.replay!=="boolean")throw new Error();
  }catch{throw new ResultError("invalid_final_response",effect,operation);}
  return {raw,operation};
 }
 return {
  async preview(scope:Scope,target:string):Promise<Preview>{
   const selected=freeze(structuredClone(scope));
   const {raw}=await send(selected,target);
   const preview=previewOf(raw.fields,selected,target,owner.user);
   if(raw.replay!==false||preview.revision!==raw.result_revision)throw new ResultError("invalid_final_preview");
   return preview;
  },
  async submit(input:Intent):Promise<Receipt>{
   if(input.family!=="erp_final")throw new ResultError("invalid_final_request");
   const intent=freezeIntent(input.preview,input.rationale,input.attempt);
   if(intent.preview.requester!==owner.user)throw new ResultError("session_changed");
   const {raw,operation}=await send(intent.preview.scope,intent.preview.target,intent);
   try {
    const fields=raw.fields;exact(fields,["mode","request","digest","recorded_state","receipt","stage"]);
    if(!operation||raw.business_state!=="pending"||fields.mode!=="synthetic"||fields.stage!=="erp_final_approval"||fields.request!==intent.preview.target||!digest(fields.digest)||fields.recorded_state!=="pending"||fields.receipt!=="historical_decision")throw new Error();
    return freeze({target:intent.preview.target,digest:fields.digest,state:"pending",revision:raw.result_revision,operation,replay:raw.replay}) as Receipt;
   }catch{throw new ResultError("invalid_final_receipt","unknown",operation);}
  }
 };
}
