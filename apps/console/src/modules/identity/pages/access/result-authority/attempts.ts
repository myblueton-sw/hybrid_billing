import type {Session} from "../../../../../service/api";
import {AuthorityError,frozen,resultDomain,type AuthorityDomain} from "./contract";
import {reviewDecision,type ActionReview} from "./actions";
import {freezeIntent,mutationTransport,type Intent,type Receipt} from "./mutations";
export type AuthorityAttempt=Readonly<{intent:Intent;phase:"pending"|"unknown"|"refused"|"recorded";receipt?:Receipt;error?:string;operation?:string}>;
export class AuthorityAttempts{
 private entries:readonly AuthorityAttempt[]=frozen([]);
 private listeners=new Set<()=>void>();
 constructor(private readonly send:(intent:Intent)=>Promise<Receipt>,private readonly actor:string,private readonly domain:AuthorityDomain=resultDomain){}
 snapshot=()=>this.entries;
 subscribe=(listener:()=>void)=>{this.listeners.add(listener);return()=>{this.listeners.delete(listener);};};
 private update(entry:AuthorityAttempt){this.entries=frozen([...this.entries.filter(e=>e.intent.attempt.key!==entry.intent.attempt.key),entry]);for(const listener of this.listeners)listener();}
 settled(entry:AuthorityAttempt){return this.entries.some(e=>e.phase==="recorded"&&e.intent.target===entry.intent.target&&e.intent.action===entry.intent.action&&e.intent.fields.digest===entry.intent.fields.digest);}
 unresolved(entry:AuthorityAttempt){return entry.phase==="pending"||entry.phase==="unknown"&&!this.settled(entry);}
 private admit(intent:Intent,recover=false){
  if(this.entries.length>=100)throw new AuthorityError("attempt_capacity");
  if(this.entries.some(e=>e.intent.attempt.key===intent.attempt.key||this.unresolved(e)&&(e.intent.target===intent.target||intent.action.startsWith("propose_")&&e.intent.action.startsWith("propose_"))&&!(recover&&e.phase==="unknown"&&e.intent.action==="approve"&&e.intent.target===intent.target&&e.intent.fields.digest===intent.fields.digest&&e.intent.rationale===intent.rationale)))throw new AuthorityError("unresolved_authority_attempt");
 }
 propose(input:Intent){const intent=freezeIntent(input,this.domain);if(!intent.action.startsWith("propose_"))throw new AuthorityError("invalid_authority_request");this.admit(intent);return this.run({intent,phase:"pending"});}
 decide(input:ActionReview){
  const reviewed=reviewDecision(input.detail,this.actor,input.action,input.intent.rationale,input.intent.attempt,this.domain);
  if(JSON.stringify(reviewed.intent)!==JSON.stringify(input.intent))throw new AuthorityError("invalid_authority_request");
  this.admit(reviewed.intent,input.action==="recover");return this.run({intent:reviewed.intent,phase:"pending"});
 }
 async retry(key:string){
  const entry=this.entries.find(e=>e.intent.attempt.key===key);
  if(!entry||entry.phase!=="unknown"||this.settled(entry)||this.entries.some(e=>e.phase==="pending"&&e.intent.target===entry.intent.target))return;
  await this.run(entry);
 }
 private async run(entry:AuthorityAttempt){
  const unknown=entry.phase==="unknown";this.update({...entry,phase:"pending"});
  try{const receipt=await this.send(entry.intent);this.update({...entry,phase:"recorded",receipt,error:undefined});}
  catch(e){this.update({...entry,phase:unknown||!(e instanceof AuthorityError)||e.effect==="unknown"?"unknown":"refused",error:e instanceof AuthorityError?e.code:"service_unavailable",operation:entry.operation||(e instanceof AuthorityError?e.operation:"")});}
 }
}
const stores=new Map<string,AuthorityAttempts>();
export function sessionAuthorityAttempts(session:Session,d:AuthorityDomain=resultDomain){
 const key=JSON.stringify([session.user,session.workspace,session.csrf,d.namespace]);const found=stores.get(key);if(found)return found;
 if(stores.size>=64)throw new AuthorityError("session_capacity");
 const store=new AuthorityAttempts(mutationTransport(session,undefined,d),session.user,d);stores.set(key,store);return store;
}
