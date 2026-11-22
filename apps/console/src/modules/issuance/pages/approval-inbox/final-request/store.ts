import type { Session } from "../../../../../service/api";
import { attempt } from "../../../../../service/admin";
import { rationaleValid } from "../documents/packet";
import { freeze, id, ResultError } from "../results/contract";
import { finalRequestClient } from "./client";
import { freezeIntent, validScope, type Scope, type Preview, type Intent, type Receipt } from "./contract";
export type Entry=Readonly<{phase:"pending"|"unknown"|"refused"|"recorded";intent:Intent;receipt?:Receipt;error?:string;operation?:string}>;
export type State=Readonly<{scope?:Scope;target:string;rationale:string;checking:boolean;preview?:Preview;error?:string;entries:readonly Entry[]}>;
type Client=ReturnType<typeof finalRequestClient>;
export class FinalRequestStore {
 private state:State=freeze({target:"",rationale:"",checking:false,entries:[]});
 private generation=0;
 private listeners=new Set<()=>void>();
 constructor(private client:Client){}
 snapshot=()=>this.state;
 subscribe=(listener:()=>void)=>{this.listeners.add(listener);return()=>{this.listeners.delete(listener);};};
 private update(patch:Partial<State>){this.state=freeze({...this.state,...patch});for(const listener of this.listeners)listener();}
 unresolved=()=>this.state.entries.some(e=>e.phase==="pending"||e.phase==="unknown");
 select(scope?:Scope){
  if(scope&&!validScope(scope))throw new ResultError("invalid_final_request");
  if(JSON.stringify(scope)===JSON.stringify(this.state.scope))return;
  this.generation++;
  this.update({scope:scope?structuredClone(scope):undefined,target:"",rationale:"",checking:false,preview:undefined,error:undefined});
 }
 invalidate(){this.generation++;this.update({checking:false,preview:undefined,error:undefined});}
 edit(target:string,rationale:string){
  if(this.unresolved()||this.state.checking)return;
  this.generation++;
  this.update({target,rationale,preview:undefined,error:undefined});
 }
 clear(){if(!this.unresolved()){this.generation++;this.update({target:"",rationale:"",checking:false,preview:undefined,error:undefined});}}
 async check(){
  const {scope,target,rationale}=this.state;
  if(!scope||this.state.checking||this.unresolved())return;
  if(!id(target)||!rationaleValid(rationale)){this.update({error:"invalid_final_request"});return;}
  const generation=++this.generation;
  this.update({checking:true,preview:undefined,error:undefined});
  try {const preview=await this.client.preview(scope,target);if(generation===this.generation)this.update({checking:false,preview});}
  catch(error){if(generation===this.generation)this.update({checking:false,error:error instanceof ResultError?error.code:"service_unavailable"});}
 }
 async submit(){
  const {preview,rationale,scope}=this.state;
  if(!preview||!scope||this.unresolved())return;
  if(this.state.entries.length>=100){this.update({error:"attempt_capacity"});return;}
  let intent:Intent;
  try {
   if(JSON.stringify(scope)!==JSON.stringify(preview.scope)||preview.target!==this.state.target)throw new ResultError("invalid_final_request");
   intent=freezeIntent(preview,rationale,attempt());
  }catch(error){this.update({error:error instanceof ResultError?error.code:"invalid_final_request"});return;}
  this.generation++;
  this.update({preview:undefined,checking:false,error:undefined});
  await this.run(intent,false);
 }
 async retry(key:string){const entry=this.state.entries.find(e=>e.intent.attempt.key===key);if(!entry||entry.phase!=="unknown"||this.state.entries.some(e=>e.phase==="pending"))return;await this.run(entry.intent,true,entry.operation);}
 private entry(value:Entry){const entries=this.state.entries.filter(e=>e.intent.attempt.key!==value.intent.attempt.key);this.update({entries:[...entries,value]});}
 private async run(intent:Intent,wasUnknown:boolean,operation?:string){
  this.entry({phase:"pending",intent,operation});
  try {const receipt=await this.client.submit(intent);this.entry({phase:"recorded",intent,receipt,operation:receipt.operation});if(this.state.target===intent.preview.target&&JSON.stringify(this.state.scope)===JSON.stringify(intent.preview.scope))this.update({target:"",rationale:""});}
  catch(error){const known=error instanceof ResultError;this.entry({phase:wasUnknown||!known||error.effect==="unknown"?"unknown":"refused",intent,error:known?error.code:"service_unavailable",operation:operation||(known?error.operation:undefined)});}
 }
}
const stores=new Map<string,FinalRequestStore>();
export function sessionFinalRequestStore(session:Session):FinalRequestStore{
 const key=JSON.stringify([session.user,session.workspace,session.csrf]);
 const found=stores.get(key);if(found)return found;
 if(stores.size>=16)throw new ResultError("session_capacity");
 const store=new FinalRequestStore(finalRequestClient(session));stores.set(key,store);return store;
}
