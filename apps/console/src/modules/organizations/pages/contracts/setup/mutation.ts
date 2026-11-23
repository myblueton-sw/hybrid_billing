import type { Session } from "../../../../../service/api";
import { strictJSON } from "../../../../issuance/pages/approval-inbox/documents/json";
import { timestamp } from "../../../../issuance/pages/approval-inbox/documents/contract";
import { freeze, id, object, revision, ResultError } from "../../../../issuance/pages/approval-inbox/results/contract";
import { bounded, checkSession, responseText } from "../../../../issuance/pages/approval-inbox/results/transport";
import { setupFields } from "./client";
import { SetupError } from "./context";
import { configurationOf, type Configuration } from "./history";
import { setupIntent, type SetupIntent } from "./intents";

const refusals: Record<string, readonly [number, string, boolean]> = {
 permission_denied:[403,"none",false], self_approval_denied:[403,"none",false], unauthenticated:[401,"obtain_new_access_answer",false], csrf_refused:[403,"none",false], reauthentication_required:[403,"reauthenticate",false], evidence_stale:[409,"inspect_current_state",false], revision_conflict:[409,"inspect_current_state",false], invalid_request:[422,"correct_request",false], prerequisite_failed:[422,"correct_request",false], rate_limited:[429,"retry_later",true], unsupported_capability:[501,"none",false], dependency_unavailable:[503,"retry_later",true], authority_unavailable:[503,"retry_later",true],
};
function cleanFailure(raw: Record<string, unknown>, status: number, request: string): boolean {
 const rule=typeof raw.code==="string"?refusals[raw.code]:undefined;
 return !!rule&&status===rule[0]&&raw.next_action===rule[1]&&raw.retry===rule[2]&&raw.request_id===request&&raw.operation_id===undefined
  &&Object.keys(raw).every(k=>["code","retry","next_action","request_id","fields","retry_after_ms"].includes(k))
  &&(raw.fields===undefined||Array.isArray(raw.fields)&&raw.fields.length<=24&&raw.fields.every(v=>typeof v==="string"&&v.length<=128))
  &&(raw.retry_after_ms===undefined||Number.isSafeInteger(raw.retry_after_ms)&&Number(raw.retry_after_ms)>=0);
}
export type SetupReceipt=Readonly<{configuration:Configuration;revision:string;operation:string;replay:boolean}>;
export function setupMutationClient(session:Session,fetcher:typeof fetch=fetch.bind(globalThis)) {
 const owner=freeze({...session});
 return async(input:SetupIntent):Promise<SetupReceipt>=>{
  const intent=setupIntent(input);
  if(intent.actor!==owner.user||intent.workspace!==owner.workspace||intent.sessionCSRF!==owner.csrf)throw new SetupError("session_changed");
  try{await checkSession(owner,fetcher);}catch(error){throw new SetupError(error instanceof ResultError?error.code:"session_unavailable");}
  let response:Response,raw:unknown;
  try{
   const result=await bounded(async signal=>{
    const response=await fetcher(`/api/v1/commands/issuance_setup.${intent.operation}`,{signal,method:"POST",credentials:"same-origin",cache:"no-store",headers:{"Content-Type":"application/json","X-CSRF-Token":owner.csrf,"X-Request-Id":intent.attempt.request},body:JSON.stringify({request_id:intent.attempt.request,target_id:intent.target,idempotency_key:intent.attempt.key,expected_revision:intent.revision,rationale:intent.rationale,body:intent.args})});
    return {response,raw:strictJSON(await responseText(response),125000)};
   },"unknown");response=result.response;raw=result.raw;
  }catch(error){throw new SetupError(error instanceof ResultError?error.code:"service_unavailable","unknown");}
  if(!object(raw))throw new SetupError("invalid_setup_response","unknown");
  const operation=raw.request_id===intent.attempt.request&&id(raw.operation_id)?raw.operation_id:"";
  if(!response.ok)throw new SetupError(raw.request_id===intent.attempt.request&&typeof raw.code==="string"&&/^[a-z_]{1,80}$/.test(raw.code)?raw.code:"service_unavailable",cleanFailure(raw,response.status,intent.attempt.request)?"none":"unknown",operation);
  try{
   const keys=["request_id","status","operation_id","target_id","result_revision","business_state","replay","as_of","fields"];
   if(Object.keys(raw).length!==keys.length||keys.some(k=>!Object.hasOwn(raw,k))||response.status!==200||raw.request_id!==intent.attempt.request||raw.target_id!==intent.target||raw.status!=="completed"||!operation||!revision(raw.result_revision)||typeof raw.replay!=="boolean"||!timestamp(raw.as_of))throw new Error();
   const fields=setupFields(raw.fields);
   if(fields.receipt!=="historical_configuration"||fields.contract!==intent.contract||fields.current_eligibility!=="unchecked")throw new Error();
   const {receipt:_,contract:__,current_eligibility:___,...row}=fields;
   const configuration=configurationOf(row);
   if(!["proposed","confirmed"].includes(configuration.state)||raw.business_state!==configuration.state||Object.entries(intent.expected).some(([key,value])=>configuration[key]!==value))throw new Error();
   if(intent.operation==="confirm_configuration"&&(configuration.state!=="confirmed"||configuration.confirmer!==intent.actor))throw new Error();
   return freeze({configuration,revision:raw.result_revision,operation,replay:raw.replay});
  }catch{throw new SetupError("invalid_setup_receipt","unknown",operation);}
 };
}
