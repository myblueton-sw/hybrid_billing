import type { Session } from "../../../../../service/api";
import { draftReadClient } from "./client";
import { DraftError, freeze, type Page, type Summary } from "./contract";
import type { Context } from "./context";
import type { Inspection } from "./inspection";
import { draftIntent, draftMutationClient, type DraftIntent, type DraftReceipt } from "./mutation";
export type Entry=Readonly<{phase:"pending"|"unknown"|"refused"|"recorded"|"reconciled";intent:DraftIntent;receipt?:DraftReceipt;error?:string;operation?:string}>;
type Load<T>=Readonly<{loading:boolean;value?:T;error?:string}>;
export type DraftState=Readonly<{selected:string;final?:Readonly<{request:string;digest:string}>;list:Load<Page<Summary>>;inspection:Load<Inspection>;context:Load<Context>;entries:readonly Entry[]}>;
type Reads=ReturnType<typeof draftReadClient>;
type Send=ReturnType<typeof draftMutationClient>;
const empty=()=>({loading:false});
export class DraftStore {
 private state:DraftState=freeze({selected:"",list:empty(),inspection:empty(),context:empty(),entries:[]});
 private listGeneration=0;
 private selectionGeneration=0;
 private contextGeneration=0;
 private listeners=new Set<()=>void>();
 constructor(private reads:Reads,private send:Send){}
 snapshot=()=>this.state;
 subscribe=(listener:()=>void)=>{this.listeners.add(listener);return()=>{this.listeners.delete(listener);};};
 private update(patch:Partial<DraftState>){this.state=freeze({...this.state,...patch});for(const listener of this.listeners)listener();}
 private error(error:unknown){return error instanceof DraftError?error.code:"service_unavailable";}
 unresolved=()=>this.state.entries.some(e=>e.phase==="pending"||e.phase==="unknown");
 clearSelection(){this.selectionGeneration++;this.contextGeneration++;this.update({selected:"",final:undefined,inspection:empty(),context:empty()});}
 invalidate(){this.listGeneration++;this.clearSelection();this.update({list:empty()});}
 async list(after="",snapshot="") {
  const generation=++this.listGeneration;
  this.clearSelection();this.update({list:{loading:true}});
  try{const value=await this.reads.list(after,snapshot);if(generation===this.listGeneration)this.update({list:{loading:false,value}});}
  catch(error){if(generation===this.listGeneration)this.update({list:{loading:false,error:this.error(error)}});}
 }
 async open(draft:string,after="",snapshot="") {
  const generation=++this.selectionGeneration;
  this.contextGeneration++;
  this.update({selected:draft,final:undefined,inspection:{loading:true},context:empty()});
  try {
   const value=await this.reads.inspect(draft,after,snapshot);
   if(generation!==this.selectionGeneration)return;
   this.update({inspection:{loading:false,value},final:{request:value.final_request,digest:value.final_digest}});
   await this.context();
  }catch(error){if(generation===this.selectionGeneration)this.update({inspection:{loading:false,error:this.error(error)}});}
 }
 async final(request:string,digest:string) {
  this.clearSelection();this.update({final:{request,digest}});await this.context();
 }
 async context(after="",snapshot="") {
  const selected=this.state.final;
  if(!selected)return;
  const generation=++this.contextGeneration;
  this.update({context:{loading:true}});
  try{const value=await this.reads.context(selected.request,selected.digest,after,snapshot);if(generation===this.contextGeneration)this.update({context:{loading:false,value}});}
  catch(error){if(generation===this.contextGeneration)this.update({context:{loading:false,error:this.error(error)}});}
 }
 async execute(input:DraftIntent) {
  if(this.unresolved())throw new DraftError("unresolved_operation");
  if(this.state.entries.length>=100)throw new DraftError("attempt_capacity");
  const intent=draftIntent(input);
  if(this.state.entries.some(e=>e.intent.attempt.key===intent.attempt.key))throw new DraftError("invalid_draft_request");
  await this.run(intent,false);
 }

 canRecover(input:DraftIntent):boolean {
  const inspected=this.state.inspection.value;
  if(!inspected||inspected.draft!==input.draft||inspected.revision!==input.revision||!inspected.recoveries.some(r=>r.operation===input.action&&r.business_id===input.target&&r.original_actor===input.actor&&r.permission==="allowed"&&Object.keys(r.arguments).length===Object.keys(input.arguments).length&&Object.entries(r.arguments).every(([k,v])=>input.arguments[k]===v)))return false;
  const same=(a:DraftIntent)=>a.action===input.action&&a.target===input.target&&a.actor===input.actor&&a.workspace===input.workspace&&a.draft===input.draft&&Object.keys(a.arguments).length===Object.keys(input.arguments).length&&Object.entries(a.arguments).every(([k,v])=>input.arguments[k]===v);
  return !this.state.entries.some(e=>e.phase==="pending"||e.phase==="unknown"&&!same(e.intent));
 }
 async recover(input:DraftIntent) {
  const intent=draftIntent(input);
  if(!this.canRecover(intent))throw new DraftError("unresolved_operation");
  if(this.state.entries.length>=100)throw new DraftError("attempt_capacity");
  if(this.state.entries.some(e=>e.intent.attempt.key===intent.attempt.key))throw new DraftError("invalid_draft_request");
  const previous=this.state.entries.filter(e=>e.phase==="unknown").map(e=>e.intent.attempt.key);
  await this.run(intent,false);
  const result=this.state.entries.find(e=>e.intent.attempt.key===intent.attempt.key);
  if(result?.phase==="recorded"&&result.receipt&&result.receipt.reviewState!=="approval_unknown") {
   this.update({entries:this.state.entries.map(e=>previous.includes(e.intent.attempt.key)?{...e,phase:"reconciled",receipt:result.receipt}:e)});
  }
 }
 async retry(key:string) {
  const entry=this.state.entries.find(e=>e.intent.attempt.key===key);
  if(!entry||entry.phase!=="unknown"||this.state.entries.some(e=>e.phase==="pending"))return;
  await this.run(entry.intent,true,entry.operation);
 }
 private entry(entry:Entry){this.update({entries:[...this.state.entries.filter(e=>e.intent.attempt.key!==entry.intent.attempt.key),entry]});}
 private clearReads(){
  this.listGeneration++;this.selectionGeneration++;this.contextGeneration++;
  this.update({list:empty(),inspection:empty(),context:empty()});
 }
 private async run(intent:DraftIntent,wasUnknown:boolean,operation?:string) {
  this.clearReads();
  this.entry({phase:"pending",intent,operation});
  try{const receipt=await this.send(intent);this.clearReads();this.entry({phase:"recorded",intent,receipt,operation:receipt.operation});}
  catch(error){this.clearReads();const known=error instanceof DraftError;this.entry({phase:wasUnknown||!known||error.effect==="unknown"?"unknown":"refused",intent,error:this.error(error),operation:operation||(known?error.operation:undefined)});}
 }
}
const stores=new Map<string,DraftStore>();
export function sessionDraftStore(session:Session):DraftStore {
 const key=JSON.stringify([session.user,session.workspace,session.csrf]);
 const found=stores.get(key);if(found)return found;
 if(stores.size>=16)throw new DraftError("session_capacity");
 const store=new DraftStore(draftReadClient(session),draftMutationClient(session));stores.set(key,store);return store;
}
