import type { Session } from "../../../../../service/api";
import { attempt } from "../../../../../service/admin";
import { freeze } from "../../../../issuance/pages/approval-inbox/results/contract";
import { setupReadClient } from "./client";
import { SetupError } from "./context";
import type { Configuration } from "./history";
import { proposalIntent, retainedIntent, type SetupIntent } from "./intents";
import { sessionSetupWriteStore, type SetupWriteStore } from "./write-store";

type Draft = Readonly<{ target: string; preset: boolean; languages: readonly string[]; reason: string; selected: string }>;
export type EditorState = Readonly<{ draft: Draft; checking: boolean; prepared?: SetupIntent; recovery: boolean; error?: string }>;
export class SetupEditor {
 private state: EditorState = freeze({draft:{target:attempt().request,preset:false,languages:[],reason:"",selected:""},checking:false,recovery:false});
 private generation=0;
 private listeners=new Set<()=>void>();
 constructor(private readonly session:Session,readonly contract:string,readonly on:string,readonly writes:SetupWriteStore,private readonly reads=setupReadClient(session)){}
 snapshot=()=>this.state;
 subscribe=(listener:()=>void)=>{this.listeners.add(listener);return()=>{this.listeners.delete(listener);};};
 private update(patch:Partial<EditorState>){this.state=freeze({...this.state,...patch});this.listeners.forEach(listener=>listener());}
 invalidate(){this.generation++;this.writes.invalidate();this.update({checking:false,prepared:undefined,error:undefined});}
 edit(patch:Partial<Omit<Draft,"target">>){this.invalidate();this.update({draft:{...this.state.draft,...patch}});}
 async check(row?:Configuration,history?:{after:string;snapshot:string}){
  this.invalidate();const generation=this.generation,draft=this.state.draft;
  this.update({checking:true});
  try{
   let prepared:SetupIntent,recovery=false;
   if(row){
    if(!history||row.configuration!==draft.selected||!row.setup_on||!row.scope_digest)throw new SetupError("invalid_setup_request");
    const operation=row.state==="proposal_unknown"?"propose_configuration":"confirm_configuration";
    const fields:Record<string,string>=operation==="propose_configuration"?{on:row.setup_on,scope_digest:row.scope_digest}:{digest:row.digest};
    await this.writes.inspect({contract:this.contract,configuration:row.configuration,operation,fields},history);
    if(generation!==this.generation)return;
    const inspection=this.writes.snapshot().inspection,fresh=inspection?.page?.rows.find(r=>r.configuration===row.configuration);
    if(!inspection||!fresh)throw new SetupError(this.writes.snapshot().error||"inspection_required");
    if(inspection.action.code)throw new SetupError(inspection.action.code);
    prepared=retainedIntent(this.session,inspection.page!,fresh,inspection.action,draft.reason,attempt());
    recovery=fresh.state==="proposal_unknown"||fresh.state==="confirmation_unknown";
   }else{
    if(draft.selected||!draft.preset||!draft.languages.length)throw new SetupError("invalid_setup_request");
    const context=await this.reads.context(this.contract,this.on);
    if(generation!==this.generation)return;
    const options=await this.reads.options(context);
    if(generation!==this.generation)return;
    await this.writes.inspect({contract:this.contract,configuration:draft.target,operation:"propose_configuration",fields:{on:context.on,scope_digest:context.scopeDigest}});
    if(generation!==this.generation)return;
    const inspection=this.writes.snapshot().inspection;
    if(!inspection)throw new SetupError(this.writes.snapshot().error||"inspection_required");
    if(inspection.action.code)throw new SetupError(inspection.action.code);
    if(context.revision!==options.revision||context.revision!==inspection.action.revision)throw new SetupError("evidence_stale");
    prepared=proposalIntent(this.session,context,options,draft.languages,draft.reason,inspection.action,attempt());
   }
   if(generation===this.generation)this.update({checking:false,prepared,recovery});
  }catch(error){if(generation===this.generation)this.update({checking:false,error:error instanceof SetupError?error.code:"service_unavailable"});}
 }
 async submit(){
  const intent=this.state.prepared,recovery=this.state.recovery;
  if(!intent)throw new SetupError("inspection_required");
  this.update({prepared:undefined,error:undefined});
  try{if(recovery)await this.writes.recover(intent);else await this.writes.submit(intent);}
  catch(error){this.update({error:error instanceof SetupError?error.code:"service_unavailable"});}
 }
}
const editors=new Map<string,SetupEditor>();
export function sessionSetupEditor(session:Session,contract:string,on:string){
 const key=JSON.stringify([session.user,session.workspace,session.csrf,contract,on]),existing=editors.get(key);
 if(existing)return existing;
 if(editors.size>=100)throw new SetupError("draft_capacity");
 const editor=new SetupEditor(freeze({...session}),contract,on,sessionSetupWriteStore(session));editors.set(key,editor);return editor;
}
