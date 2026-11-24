import type { Session } from "../../../../../service/api";
import { freeze } from "../../../../issuance/pages/approval-inbox/results/contract";
import { actionRequest, type ActionRequest, type SetupAction } from "./action";
import { setupReadClient } from "./client";
import { SetupError } from "./context";
import type { ConfigurationPage } from "./history";
import { setupIntent, type SetupIntent } from "./intents";
import { setupMutationClient, type SetupReceipt } from "./mutation";

type Entry = Readonly<{ intent: SetupIntent; phase: "pending" | "unknown" | "refused" | "recorded" | "reconciled"; error?: string; operation?: string; receipt?: SetupReceipt; at: string; noRecord?: true }>;
export const noRecordBoundMs = 30 * 60 * 1000;
type Inspection = Readonly<{ request: ActionRequest; action: SetupAction; page?: ConfigurationPage }>;
export type SetupWriteState = Readonly<{ entries: readonly Entry[]; inspecting: boolean; inspection?: Inspection; error?: string }>;
const sameFields = (a: Readonly<Record<string,string>>, b: Readonly<Record<string,string>>) => Object.keys(a).length === Object.keys(b).length && Object.entries(a).every(([k,v]) => b[k] === v);
const sameOperation = (a: SetupIntent, b: SetupIntent) => a.operation === b.operation && a.target === b.target && a.contract === b.contract && a.actor === b.actor && a.workspace === b.workspace && sameFields(a.args,b.args) && (a.operation !== "propose_configuration" || a.rationale === b.rationale);
export class SetupWriteStore {
 private state: SetupWriteState = freeze({entries:[],inspecting:false});
 private generation = 0;
 private listeners = new Set<()=>void>();
 private started = new Map<string,number>();
 constructor(private readonly owner: Readonly<Session>, private readonly reads: Pick<ReturnType<typeof setupReadClient>,"action"|"configurations">, private readonly mutate: ReturnType<typeof setupMutationClient>, private readonly now: () => number = Date.now, private readonly elapsed: () => number = () => performance.now()) {}
 private within(age:number):boolean {return Number.isFinite(age)&&age>=0&&age<noRecordBoundMs;}
 private fresh(entry:Entry):boolean {const began=this.started.get(entry.intent.attempt.key);return began!==undefined&&this.within(this.elapsed()-began)&&this.within(this.now()-Date.parse(entry.at));}
 snapshot = () => this.state;
 subscribe = (listener:()=>void) => {this.listeners.add(listener);return()=>{this.listeners.delete(listener);};};
 private update(patch: Partial<SetupWriteState>) {this.state=freeze({...this.state,...patch});this.listeners.forEach(listener=>listener());}
 invalidate() {this.generation++;this.update({inspection:undefined,inspecting:false,error:undefined});}
 async inspect(input:ActionRequest,history?:{after:string;snapshot:string}) {
  const request=actionRequest(input),generation=++this.generation;
  this.update({inspection:undefined,inspecting:true,error:undefined});
  try {
   const [action,page]=await Promise.all([this.reads.action(request),history?this.reads.configurations(request.contract,history.after,history.after?history.snapshot:""):Promise.resolve(undefined)]);
   if(page&&page.revision!==action.revision)throw new SetupError("evidence_stale");
   if(generation===this.generation)this.update({inspecting:false,inspection:{request,action,page}});
  }catch(error){if(generation===this.generation)this.update({inspecting:false,error:error instanceof SetupError?error.code:"service_unavailable"});}
 }
 private checked(intent:SetupIntent,recovery:boolean) {
  const inspection=this.state.inspection;
  if(!inspection||inspection.action.permission!=="allowed"||inspection.action.stepUp!=="satisfied"||inspection.action.action!=="recheck_required"||inspection.action.revision!==intent.revision
   ||inspection.request.contract!==intent.contract||inspection.request.configuration!==intent.target||inspection.request.operation!==intent.operation)throw new SetupError("inspection_required");
  const args:Record<string,string>=intent.operation==="propose_configuration"?{on:intent.args.on,scope_digest:intent.args.scope_digest}:{digest:intent.args.digest};
  if(!sameFields(inspection.request.fields,args))throw new SetupError("inspection_required");
  if(intent.expected.digest||recovery){
   const row=inspection.page?.rows.find(r=>r.configuration===intent.target&&r.digest===intent.expected.digest);
   if(!row||inspection.page!.revision!==intent.revision||Object.entries(intent.expected).some(([k,v])=>row[k]!==v))throw new SetupError("inspection_required");
   if(recovery&&(intent.operation==="propose_configuration"?row.state!=="proposal_unknown"||row.proposer!==intent.actor:row.state!=="confirmation_unknown"||row.confirmer!==intent.actor))throw new SetupError("recovery_unavailable");
  }
 }
 private admit(intent:SetupIntent) {
  if(intent.actor!==this.owner.user||intent.workspace!==this.owner.workspace||intent.sessionCSRF!==this.owner.csrf)throw new SetupError("session_changed");
  if(this.state.entries.length>=100)throw new SetupError("attempt_capacity");
  if(this.state.entries.some(e=>e.intent.attempt.key===intent.attempt.key||e.intent.attempt.request===intent.attempt.request))throw new SetupError("duplicate_attempt");
 }
 async submit(input:SetupIntent) {
  const intent=setupIntent(input);this.admit(intent);this.checked(intent,false);
  if(this.state.entries.some(e=>e.phase==="pending"||e.phase==="unknown"))throw new SetupError("unresolved_operation");
  await this.run(intent);
 }
 async recover(input:SetupIntent) {
  const intent=setupIntent(input);this.admit(intent);this.checked(intent,true);
  if(this.state.entries.some(e=>e.phase==="pending"||e.phase==="unknown"&&!sameOperation(e.intent,intent)))throw new SetupError("unresolved_operation");
  const previous=this.state.entries.filter(e=>e.phase==="unknown").map(e=>e.intent.attempt.key);
  await this.run(intent);
  const current=this.state.entries.find(e=>e.intent.attempt.key===intent.attempt.key);
  if(current?.phase==="recorded")this.update({entries:this.state.entries.map(e=>previous.includes(e.intent.attempt.key)?{...e,phase:"reconciled",receipt:current.receipt}:e)});
 }
 async retry(key:string) {
  const entry=this.state.entries.find(e=>e.intent.attempt.key===key);
  if(!entry||entry.phase!=="unknown"||this.state.entries.some(e=>e.phase==="pending"))throw new SetupError("retry_unavailable");
  await this.run(entry.intent,entry);
 }
 private entry(value:Entry){this.update({entries:[...this.state.entries.filter(e=>e.intent.attempt.key!==value.intent.attempt.key),value]});}
 private async run(intent:SetupIntent,previous?:Entry) {
  const at=previous?.at??new Date(this.now()).toISOString();
  if(!previous)this.started.set(intent.attempt.key,this.elapsed());
  this.invalidate();this.entry({intent,phase:"pending",operation:previous?.operation,at});
  try{const receipt=await this.mutate(intent);this.entry({intent,phase:"recorded",receipt,operation:receipt.operation,at});}
  catch(error){
   const known=error instanceof SetupError,repeated=previous?.phase==="unknown";
   const noRecord=repeated&&known&&error.code==="revision_conflict"&&error.effect==="none"&&!error.operation&&!previous!.operation&&this.fresh(previous!);
   this.entry({intent,phase:!noRecord&&(repeated||!known||error.effect==="unknown")?"unknown":"refused",error:known?error.code:"service_unavailable",operation:known?error.operation||previous?.operation:previous?.operation,at,...(noRecord?{noRecord:true as const}:{})});
  }
  finally{this.invalidate();}
 }
}
const stores = new Map<string,SetupWriteStore>();
export function sessionSetupWriteStore(session:Session):SetupWriteStore {
 const key=JSON.stringify([session.user,session.workspace,session.csrf]),existing=stores.get(key);
 if(existing)return existing;
 if(stores.size>=16)throw new SetupError("session_capacity");
 const owner=freeze({...session}),store=new SetupWriteStore(owner,setupReadClient(owner),setupMutationClient(owner));stores.set(key,store);return store;
}
