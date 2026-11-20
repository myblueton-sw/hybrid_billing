import { useState, useSyncExternalStore } from "react";
import type { Locale } from "../../../../shared/types";
import type { LicensePage } from "../../../../service/licenses";
import { useService } from "../../../../service/context";
import type { SubmitLicense } from "./LiveForm";
import { licenseSession } from "./live-session";
import { tk } from "./trust-copy";
import TrustLookup from "./TrustLookup";
import TrustPropose from "./TrustPropose";
export default function Trust({l,page,phase,disabled,submit}:{l:Locale;page:LicensePage|null;phase:string;disabled:boolean;submit:SubmitLicense}) {
 const [task,setTask]=useState<'key'|'proposal'|'new'|'change'|''>(''),service=useService()!,controller=licenseSession(service.session),snapshot=useSyncExternalStore(controller.subscribe,controller.snapshot);
 return <section className="p07-trust" aria-labelledby="p07-trust-heading"><h2 id="p07-trust-heading">{tk(l,'title')}</h2><p>{tk(l,'boundary')}</p><fieldset className="p07-modes"><legend>{tk(l,'task')}</legend>{(['key','proposal','new','change'] as const).map(value=><label key={value}><input type="radio" name="trust-task" value={value} checked={task===value} onChange={()=>setTask(value)}/><span>{tk(l,value==='new'?'newKey':value==='change'?'changeTask':value==='key'?'lookupKey':'lookupProposal')}</span></label>)}</fieldset>
  {snapshot.result?.unknown&&<p className="p07-state" role="status">{tk(l,'submissionsBlocked')}</p>}
  {(task==='key'||task==='proposal'||task==='change')&&<TrustLookup key={task} l={l} kind={task==='proposal'?'proposal':'key'} manage={task==='change'} parentPhase={phase} disabled={disabled} submit={submit}/>}
  {task==='new'&&phase==='ready'&&!snapshot.result?.unknown&&<TrustPropose key={task+snapshot.settled} l={l} create={task==='new'} disabled={disabled||page?.state!=='configured'} submit={submit}/>}
 </section>;
}
