import type {Session} from "../../../../../service/api";
import {attempt} from "../../../../../service/admin";
import {strictJSON} from "../../../../../service/strict-json";
import {AuthorityError,bytes,exact,frozen,id,instant,object,resultDomain,revision,type AuthorityDomain,type Envelope} from "./contract";
export async function responseText(response:Response,maximum=100000):Promise<string>{
 if(!response.body)throw new AuthorityError("invalid_authority_response");
 const reader=response.body.getReader(),decoder=new TextDecoder("utf-8",{fatal:true});let total=0,text="";
 try{for(;;){const part=await reader.read();if(part.done)break;total+=part.value.byteLength;if(total>maximum)throw new AuthorityError("invalid_authority_response");text+=decoder.decode(part.value,{stream:true});}return text+decoder.decode();}
 finally{await reader.cancel().catch(()=>{});reader.releaseLock();}
}
export async function bounded<T>(work:(signal:AbortSignal)=>Promise<T>,effect:"none"|"unknown"):Promise<T>{
 const controller=new AbortController();let timer:ReturnType<typeof setTimeout>;
 const timeout=new Promise<never>((_,reject)=>{timer=setTimeout(()=>{controller.abort();reject(new AuthorityError("request_timeout",effect));},30000);});
 try{return await Promise.race([work(controller.signal),timeout]);}finally{clearTimeout(timer!);}
}
export function fieldMap(raw:unknown):Record<string,string>{
 if(!object(raw)||Object.keys(raw).length>20||Object.entries(raw).some(([k,v])=>!/^[a-z][a-z0-9_]*$/.test(k)||typeof v!=="string"||bytes(v)>4096))throw new AuthorityError("invalid_authority_response");
 return raw as Record<string,string>;
}
export async function checkSession(owner:Readonly<Session>,fetcher:typeof fetch):Promise<void>{
 let current:unknown;
 try{current=await bounded(async signal=>{
  const response=await fetcher("/api/v1/session",{signal,credentials:"same-origin",cache:"no-store",headers:{"X-HB-Client":"console"}});
  if(response.status===401)return null;
  if(response.status!==200)throw new AuthorityError("session_unavailable");
  return strictJSON(await responseText(response,16384),16384);
 },"none");}catch{throw new AuthorityError("session_unavailable");}
 if(current===null)throw new AuthorityError("session_changed");
 if(!object(current)||!id(current.user)||!id(current.workspace)||typeof current.csrf_token!=="string"||!current.csrf_token||bytes(current.csrf_token)>4096)throw new AuthorityError("session_unavailable");
 if(current.user!==owner.user||current.workspace!==owner.workspace||current.csrf_token!==owner.csrf)throw new AuthorityError("session_changed");
}
export function readTransport(session:Session,fetcher:typeof fetch=fetch.bind(globalThis),d:AuthorityDomain=resultDomain){
 const owner=frozen({...session});
 return async(command:string,target:string,fields:Record<string,string>):Promise<Envelope>=>{
  if(!["history","context","inspect"].some(kind=>command===`${d.namespace}.${kind}`)||!id(target))throw new AuthorityError("invalid_authority_request");
  const values=frozen(structuredClone(fieldMap(fields))),tried=attempt();
  await checkSession(owner,fetcher);
  let response:Response,raw:unknown;
  try{const result=await bounded(async signal=>{
   const response=await fetcher(`/api/v1/commands/${command}`,{signal,method:"POST",credentials:"same-origin",cache:"no-store",headers:{"Content-Type":"application/json","X-CSRF-Token":owner.csrf,"X-Request-Id":tried.request},body:JSON.stringify({request_id:tried.request,target_id:target,body:values})});
   return {response,raw:strictJSON(await responseText(response),100000)};
  },"none");response=result.response;raw=result.raw;}catch(e){throw new AuthorityError(e instanceof AuthorityError?e.code:"service_unavailable");}
  if(!object(raw))throw new AuthorityError("invalid_authority_response");
  if(!response.ok)throw new AuthorityError(raw.request_id===tried.request&&typeof raw.code==="string"&&/^[a-z_]{1,80}$/.test(raw.code)?raw.code:"service_unavailable");
  exact(raw,["request_id","status","target_id","result_revision","replay","as_of","fields"]);
  if(response.status!==200||raw.request_id!==tried.request||raw.target_id!==target||raw.status!=="completed"||!revision(raw.result_revision)||raw.replay!==false||instant(raw.as_of)===undefined)throw new AuthorityError("invalid_authority_response");
  return frozen({revision:raw.result_revision,at:raw.as_of as string,fields:fieldMap(raw.fields)});
 };
}
