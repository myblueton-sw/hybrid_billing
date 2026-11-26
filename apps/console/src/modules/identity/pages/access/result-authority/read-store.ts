import {AuthorityError,frozen,id,resultDomain,type AuthorityDomain,type Capability,type Read} from "./contract";
import {contextPage,historyPage,type ContextPage,type Cursor,type HistoryPage} from "./reads";
import {inspectionOf,type Inspection} from "./inspection";
export type Load<T>=Readonly<{phase:"idle"|"loading"|"ready"|"error";value?:T;error?:string}>;
export type AuthorityReadsState=Readonly<{history:Load<HistoryPage>;detail:Load<Inspection>;context:Load<ContextPage>;selected:string;capability?:Capability;collection?:"recipients"|"ceilings"}>;
const idle:Load<never>={phase:"idle"};
const error=(e:unknown)=>e instanceof AuthorityError?e.code:"service_unavailable";
export class AuthorityReads{
 private state:AuthorityReadsState=frozen({history:idle,detail:idle,context:idle,selected:""});
 private listeners=new Set<()=>void>();
 private historyEpoch=0;private detailEpoch=0;private contextEpoch=0;private disposed=false;
 constructor(private readonly read:Read,private readonly workspace:string,private readonly actor:string,private readonly domain:AuthorityDomain=resultDomain){}
 snapshot=()=>this.state;
 subscribe=(listener:()=>void)=>{this.listeners.add(listener);return()=>{this.listeners.delete(listener);};};
 private update(patch:Partial<AuthorityReadsState>){if(this.disposed)return;this.state=frozen({...this.state,...patch});for(const listener of this.listeners)listener();}
 dispose(){this.disposed=true;this.historyEpoch++;this.detailEpoch++;this.contextEpoch++;this.listeners.clear();}
 clearDetail(){this.detailEpoch++;this.update({selected:"",detail:idle});}
 clearContext(){this.contextEpoch++;this.update({context:idle,capability:undefined,collection:undefined});}
 async history(cursor?:Cursor){
  if(this.disposed)return;
  const epoch=++this.historyEpoch;this.clearDetail();this.update({history:{phase:"loading"}});
  try{const value=await historyPage(this.read,this.workspace,this.actor,cursor,this.domain);if(epoch===this.historyEpoch)this.update({history:{phase:"ready",value}});}
  catch(e){if(epoch===this.historyEpoch)this.update({history:{phase:"error",error:error(e)}});}
 }
 async inspect(proposal:string,digest?:string){
  if(this.disposed)return;
  const epoch=++this.detailEpoch;this.update({selected:proposal,detail:{phase:"loading"}});
  try{
   if(!id(proposal))throw new AuthorityError("invalid_authority_request");
   const value=inspectionOf(await this.read(`${this.domain.namespace}.inspect`,proposal,{}),proposal,digest,this.domain);
   if(epoch===this.detailEpoch)this.update({detail:{phase:"ready",value}});
  }catch(e){if(epoch===this.detailEpoch)this.update({detail:{phase:"error",error:error(e)}});}
 }
 async context(collection:"recipients"|"ceilings",capability:Capability,cursor?:Cursor){
  if(this.disposed)return;
  const epoch=++this.contextEpoch;this.update({collection,capability,context:{phase:"loading"}});
  try{const value=await contextPage(this.read,this.workspace,this.actor,collection,capability,cursor,this.domain);if(epoch===this.contextEpoch)this.update({context:{phase:"ready",value}});}
  catch(e){if(epoch===this.contextEpoch)this.update({context:{phase:"error",error:error(e)}});}
 }
 async followMutation(target:string,work:Promise<void>){
  if(this.disposed){await work;return;}
  const historyEpoch=++this.historyEpoch,contextEpoch=++this.contextEpoch,epoch=++this.detailEpoch;
  this.update({history:idle,context:idle,detail:idle,selected:"",capability:undefined,collection:undefined});
  await work;
  if(this.disposed||epoch!==this.detailEpoch||contextEpoch!==this.contextEpoch||historyEpoch!==this.historyEpoch)return;
  await this.history();
  if(!this.disposed&&this.detailEpoch===epoch+1&&this.contextEpoch===contextEpoch&&this.historyEpoch===historyEpoch+1)await this.inspect(target);
 }
}
