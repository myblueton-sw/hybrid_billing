import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import type { Locale } from "../../../../shared/types";
import { AdminError } from "../../../../service/admin";
import { useAdminClient, useService } from "../../../../service/context";
import { proposalNumber, readTrust, trustId, type TrustPage } from "../../../../service/license-trust";
import type { SubmitLicense } from "./LiveForm";
import { licenseSession } from "./live-session";
import { k } from "./live-copy";
import { tk } from "./trust-copy";
import TrustError, { type TrustProblem } from "./TrustError";
import TrustFacts from "./TrustFacts";
import TrustPropose from "./TrustPropose";
export default function TrustLookup({l,kind,manage,parentPhase,disabled,submit}:{l:Locale;kind:"key"|"proposal";manage:boolean;parentPhase:string;disabled:boolean;submit:SubmitLicense}) {
 const api=useAdminClient(),service=useService()!,controller=licenseSession(service.session),snapshot=useSyncExternalStore(controller.subscribe,controller.snapshot);
 const [input,setInput]=useState(''),[readPage,setPage]=useState<TrustPage|null>(null),[readEpoch,setReadEpoch]=useState(-1),[phase,setPhase]=useState('idle'),[confirmed,setConfirmed]=useState(false),[rationale,setRationale]=useState(''),[problem,setProblem]=useState<TrustProblem|null>(null);
 const alive=useRef(true),seq=useRef(0),requested=useRef('');
 const load=useCallback(async(id:string)=>{const n=++seq.current,epoch=controller.snapshot().settled;setPage(null);setConfirmed(false);setRationale('');setProblem(null);setPhase('loading');
  try{const p=await readTrust(api,kind,id);if(alive.current&&n===seq.current&&epoch===controller.snapshot().settled){setReadEpoch(epoch);setPage(p);setPhase('ready');}}
  catch(e){if(alive.current&&n===seq.current){setPage(null);setPhase(e instanceof AdminError&&e.failure==='denied'?'denied':e instanceof AdminError&&e.failure==='unauthenticated'?'signedOut':'failed');}}
 },[api,kind,controller]);
 useEffect(()=>{alive.current=true;setPage(null);setConfirmed(false);setRationale('');setProblem(null);setPhase('idle');if(parentPhase==='ready'&&requested.current)void load(requested.current);return()=>{alive.current=false;seq.current++;};},[load,snapshot.settled,parentPhase]);
 const page=readEpoch===snapshot.settled&&parentPhase==='ready'?readPage:null;
 const p=page?.state==='configured'&&page.record.kind==='proposal'?page.record.proposal:null;
 const change=async(action:'approve_key_change'|'withdraw_key_change')=>{
  if(disabled||!p||p.state!=='pending'||page?.state!=='configured')return;
  if(!confirmed){setProblem({field:'p07-trust-confirm',text:tk(l,'requireConfirm')});return;}
  if(!rationale.trim()||new TextEncoder().encode(rationale.trim()).length>256){setProblem({field:'p07-trust-reason',text:k(l,'rationaleHelp')});return;}
  setConfirmed(false);setProblem(null);await submit('licenses.'+action as 'licenses.approve_key_change'|'licenses.withdraw_key_change',{},rationale.trim(),{target:p.target,revision:page.revision});
 };
 return <div className="p07-trust-panel" data-trust-phase={phase}>
  <h3>{tk(l,manage?'changeTask':kind==='key'?'lookupKey':'lookupProposal')}</h3><p>{tk(l,'lookupHelp')}</p><TrustError l={l} problem={problem}/>
  <form className="p07-lookup" noValidate onSubmit={e=>{e.preventDefault();try{kind==='key'?trustId(input):proposalNumber(input);}catch{setProblem({field:'p07-trust-id',text:tk(l,'invalid')});return;}requested.current=input;void load(input);}}>
   <div className="service-field"><label htmlFor="p07-trust-id">{tk(l,kind==='key'?'keyId':'number')}</label><input id="p07-trust-id" value={input} maxLength={kind==='key'?128:20} dir="ltr" autoComplete="off" spellCheck={false} disabled={snapshot.pending||parentPhase!=='ready'} aria-invalid={problem?.field==='p07-trust-id'||undefined} onChange={e=>{setInput(e.target.value);seq.current++;requested.current='';setPage(null);setConfirmed(false);setRationale('');setPhase('idle');setProblem(null);}}/></div>
   <button type="submit" disabled={snapshot.pending||parentPhase!=='ready'||phase==='loading'}>{tk(l,'search')}</button>
  </form>
  {phase!=='idle'&&phase!=='ready'&&<p className="p07-state" role={phase==='loading'?'status':'alert'}>{phase==='signedOut'?k(l,'signedOut'):tk(l,phase==='loading'?'loading':phase==='denied'?'denied':'failed')}</p>}
  {page&&page.state!=='configured'&&<p className="p07-state" role="status">{tk(l,page.state)}</p>}
  {page?.state==='configured'&&<TrustFacts l={l} record={page.record}/>}
  {manage&&page?.state==='configured'&&page.record.kind==='key'&&!snapshot.result?.unknown&&<TrustPropose key={page.revision+page.record.key.id} l={l} create={false} keyId={page.record.key.id} disabled={disabled||page.record.compromised} submit={(command,fields,reason)=>submit(command,fields,reason,{target:'license-installation',revision:page.revision})}/>}
  {p&&p.state==='pending'&&!snapshot.result?.unknown&&<div className="p07-form"><p>{tk(l,'dual')}</p>{p.kind==='compromise_key'&&<p>{tk(l,'compromiseNote')}</p>}
   <label className="p07-check"><input id="p07-trust-confirm" type="checkbox" checked={confirmed} disabled={disabled} onChange={e=>setConfirmed(e.target.checked)}/><span>{tk(l,'confirm')}</span></label>
   <div className="service-field"><label htmlFor="p07-trust-reason">{k(l,'rationale')}</label><textarea id="p07-trust-reason" value={rationale} maxLength={256} disabled={disabled} onChange={e=>setRationale(e.target.value)}/><p>{k(l,'rationaleHelp')}</p></div>
   <div className="ops-actions">{p.proposer!==service.session.user?<button type="button" disabled={disabled} onClick={()=>void change('approve_key_change')}>{tk(l,'approve')}</button>:<button type="button" disabled={disabled} onClick={()=>void change('withdraw_key_change')}>{tk(l,'withdraw')}</button>}</div>
  </div>}
 </div>;
}
