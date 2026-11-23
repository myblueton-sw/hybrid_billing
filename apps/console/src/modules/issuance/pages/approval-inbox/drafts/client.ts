import type { Session } from "../../../../../service/api";
import { attempt } from "../../../../../service/admin";
import { strictJSON } from "../documents/json";
import { bounded, checkSession, responseText } from "../results/transport";
import { ResultError } from "../results/contract";
import { DraftError, digest, fieldMap, freeze, id, integer, invalid, listPage, object, revision, shape, timestamp, type Envelope } from "./contract";
import { contextOf } from "./context";
import { inspectionOf } from "./inspection";

function cursor(after:string,snapshot:string,line=false):Record<string,string> {
 if(!after&&!snapshot)return {};
 if(!after||!digest(snapshot))return invalid();
 if(line)integer(after,199);else if(!id(after))return invalid();
 return {snapshot};
}
export function draftReadClient(session:Session,f:typeof fetch=fetch.bind(globalThis)) {
 const owner=freeze({...session});
 async function read(command:"context"|"list"|"inspect",target:string,fields:Record<string,string>):Promise<Envelope> {
  if(!id(target))return invalid();
  const values=freeze({...fields});
  try{await checkSession(owner,f);}catch(error){throw new DraftError(error instanceof ResultError?error.code:"session_unavailable");}
  const tried=attempt();
  let response:Response,raw:unknown;
  try {
   const result=await bounded(async signal=>{
    const response=await f(`/api/v1/commands/issuance_draft.${command}`,{signal,method:"POST",credentials:"same-origin",cache:"no-store",headers:{"Content-Type":"application/json","X-CSRF-Token":owner.csrf,"X-Request-Id":tried.request},body:JSON.stringify({request_id:tried.request,target_id:target,body:values})});
    return {response,raw:strictJSON(await responseText(response),125000)};
   },"none");
   response=result.response;raw=result.raw;
  }catch(error){throw new DraftError(error instanceof ResultError?error.code:"service_unavailable");}
  if(!object(raw))return invalid();
  if(!response.ok)throw new DraftError(raw.request_id===tried.request&&typeof raw.code==="string"&&/^[a-z_]{1,80}$/.test(raw.code)?raw.code:"service_unavailable");
  shape(raw,["request_id","status","target_id","result_revision","replay","as_of","fields"]);
  if(response.status!==200||raw.request_id!==tried.request||raw.target_id!==target||raw.status!=="completed"||!revision(raw.result_revision)||raw.replay!==false||!timestamp(raw.as_of))return invalid();
  return freeze({revision:raw.result_revision,at:raw.as_of,fields:fieldMap(raw.fields)});
 }
 return {
  async list(after="",snapshot="") {
   const fields=cursor(after,snapshot);
   if(after)fields.after=after;
   return listPage(await read("list",owner.workspace,fields),after,snapshot);
  },
  async context(request:string,expectedDigest:string,after="",snapshot="") {
   if(!digest(expectedDigest))return invalid();
   const fields={...cursor(after,snapshot),digest:expectedDigest,...(after?{after_binding:after}:{})};
   return contextOf(await read("context",request,fields),request,expectedDigest,after,snapshot);
  },
  async inspect(draft:string,after="",snapshot="") {
   const fields=cursor(after,snapshot,true);
   if(after)fields.after_line=after;
   return inspectionOf(await read("inspect",draft,fields),draft,owner.user,after,snapshot);
  }
 };
}
