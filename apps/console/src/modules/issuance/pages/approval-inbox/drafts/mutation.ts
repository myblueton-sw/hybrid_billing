import type { Session } from "../../../../../service/api";
import type { Attempt } from "../../../../../service/admin";
import { rationaleValid } from "../documents/packet";
import { strictJSON } from "../documents/json";
import { bounded, checkSession, responseText } from "../results/transport";
import { ResultError } from "../results/contract";
import { argumentsOf, type Action } from "./inspection";
import { DraftError, choice, digest, fieldMap, freeze, id, object, revision, reviews, shape, submissions, timestamp } from "./contract";
export type DraftIntent=Readonly<{action:Action;target:string;actor:string;workspace:string;revision:string;draft:string;draftDigest:string;arguments:Readonly<Record<string,string>>;rationale:string;attempt:Readonly<Attempt>}>;
export function draftIntent(input:DraftIntent):DraftIntent {
 if(!["submit","prepare_approval","approve"].includes(input.action)||![input.target,input.actor,input.workspace,input.draft,input.attempt.request,input.attempt.key].every(id)||!revision(input.revision)||!rationaleValid(input.rationale))throw new DraftError("invalid_draft_request");
 const args=argumentsOf(input.action,input.arguments);
 if(input.action==="submit"?input.target!==input.draft||input.draftDigest!=="":!digest(input.draftDigest)||input.action==="prepare_approval"&&(args.draft!==input.draft||args.draft_digest!==input.draftDigest))throw new DraftError("invalid_draft_request");
 return freeze({...input,arguments:args,attempt:{...input.attempt}});
}
const refusals:Record<string,readonly [number,string,boolean]>={permission_denied:[403,"none",false],self_approval_denied:[403,"none",false],unauthenticated:[401,"obtain_new_access_answer",false],csrf_refused:[403,"none",false],reauthentication_required:[403,"reauthenticate",false],evidence_stale:[409,"inspect_current_state",false],revision_conflict:[409,"inspect_current_state",false],invalid_request:[422,"correct_request",false],prerequisite_failed:[422,"correct_request",false],rate_limited:[429,"retry_later",true],unsupported_capability:[501,"none",false],dependency_unavailable:[503,"retry_later",true],authority_unavailable:[503,"retry_later",true]};
function cleanFailure(raw:Record<string,unknown>,status:number,request:string):boolean {
 const rule=typeof raw.code==="string"?refusals[raw.code]:undefined;
 return !!rule&&status===rule[0]&&raw.next_action===rule[1]&&raw.retry===rule[2]&&raw.request_id===request&&raw.operation_id===undefined&&Object.keys(raw).every(k=>["code","retry","next_action","request_id","fields","retry_after_ms"].includes(k))&&(raw.fields===undefined||Array.isArray(raw.fields)&&raw.fields.length<=24&&raw.fields.every(v=>typeof v==="string"&&v.length<=128))&&(raw.retry_after_ms===undefined||Number.isSafeInteger(raw.retry_after_ms)&&Number(raw.retry_after_ms)>=0);
}
export type DraftReceipt=Readonly<{draft:string;digest:string;submissionState:typeof submissions[number];review?:string;reviewDigest?:string;reviewState?:typeof reviews[number];receipt:string;revision:string;operation:string;replay:boolean}>;
export function draftMutationClient(session:Session,f:typeof fetch=fetch.bind(globalThis)) {
 const owner=freeze({...session});
 return async(input:DraftIntent):Promise<DraftReceipt>=>{
  const intent=draftIntent(input);
  if(intent.actor!==owner.user||intent.workspace!==owner.workspace)throw new DraftError("session_changed");
  try{await checkSession(owner,f);}catch(error){throw new DraftError(error instanceof ResultError?error.code:"session_unavailable");}
  let response:Response,raw:unknown;
  try {
   const result=await bounded(async signal=>{
    const response=await f(`/api/v1/commands/issuance_draft.${intent.action}`,{signal,method:"POST",credentials:"same-origin",cache:"no-store",headers:{"Content-Type":"application/json","X-CSRF-Token":owner.csrf,"X-Request-Id":intent.attempt.request},body:JSON.stringify({request_id:intent.attempt.request,target_id:intent.target,idempotency_key:intent.attempt.key,expected_revision:intent.revision,rationale:intent.rationale,body:intent.arguments})});
    return {response,raw:strictJSON(await responseText(response,32768),32768)};
   },"unknown");response=result.response;raw=result.raw;
  }catch(error){throw new DraftError(error instanceof ResultError?error.code:"service_unavailable","unknown");}
  if(!object(raw))throw new DraftError("invalid_draft_response","unknown");
  const operation=raw.request_id===intent.attempt.request&&id(raw.operation_id)?raw.operation_id:"";
  if(!response.ok)throw new DraftError(raw.request_id===intent.attempt.request&&typeof raw.code==="string"&&/^[a-z_]{1,80}$/.test(raw.code)?raw.code:"service_unavailable",cleanFailure(raw,response.status,intent.attempt.request)?"none":"unknown",operation);
  try {
   shape(raw,["request_id","status","operation_id","target_id","result_revision","business_state","replay","as_of","fields"]);
   if(response.status!==200||raw.request_id!==intent.attempt.request||raw.target_id!==intent.target||raw.status!=="completed"||!operation||!revision(raw.result_revision)||typeof raw.replay!=="boolean"||!timestamp(raw.as_of))throw new Error();
   const fields=fieldMap(raw.fields),review=intent.action!=="submit";
   shape(fields,["mode","draft","digest","submission_state","receipt","current_eligibility",...(review?["review","review_digest","review_state"]:[])]);
   const submissionState=choice(fields.submission_state,submissions),reviewState=review?choice(fields.review_state,reviews):undefined;
   if(fields.mode!=="synthetic"||fields.current_eligibility!=="unchecked"||fields.draft!==intent.draft||!digest(fields.digest)||intent.draftDigest&&fields.digest!==intent.draftDigest||raw.business_state!==(reviewState??submissionState)||fields.receipt!==(reviewState==="approved"?"historical_approval":review?"review_attempt":"submission_record"))throw new Error();
   if(submissionState!=="submitted"||intent.action==="approve"&&reviewState!=="approved"&&reviewState!=="refused")throw new Error();
   if(review&&(fields.review!==intent.target||!digest(fields.review_digest)||submissionState!=="submitted"||intent.action==="approve"&&fields.review_digest!==intent.arguments.digest))throw new Error();
   return freeze({draft:fields.draft,digest:fields.digest,submissionState,review:fields.review,reviewDigest:fields.review_digest,reviewState,receipt:fields.receipt,revision:raw.result_revision,operation,replay:raw.replay});
  }catch{throw new DraftError("invalid_draft_receipt","unknown",operation);}
 };
}
