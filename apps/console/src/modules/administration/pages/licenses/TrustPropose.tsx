import { useEffect, useRef, useState } from "react";
import type { Locale } from "../../../../shared/types";
import { publicKeyFile, trustId, utcEpoch } from "../../../../service/license-trust";
import type { SubmitLicense } from "./LiveForm";
import { k } from "./live-copy";
import { tk } from "./trust-copy";
import TrustError, { type TrustProblem } from "./TrustError";
import { utc } from "./TrustFacts";
export default function TrustPropose({l,create,keyId='',disabled,submit}:{l:Locale;create:boolean;keyId?:string;disabled:boolean;submit:SubmitLicense}) {
 const [fields,setFields]=useState<Record<string,string>|null>(null),[reading,setReading]=useState(false),[fingerprint,setFingerprint]=useState(''),[kind,setKind]=useState<'retire_key'|'compromise_key'|''>(''),[id]=useState(keyId),[at,setAt]=useState(''),[reason,setReason]=useState(''),[confirmed,setConfirmed]=useState(false),[problem,setProblem]=useState<TrustProblem|null>(null);
 const seq=useRef(0),alive=useRef(true);useEffect(()=>{alive.current=true;return()=>{alive.current=false;seq.current++;};},[]);
 const error=(field:string,text:string)=>setProblem({field,text});
 return <form className="p07-form p07-trust-panel" noValidate onSubmit={async e=>{
  e.preventDefault();if(disabled||reading)return;let data:Record<string,string>;
  if(create){if(!fields){error('p07-key-file',tk(l,'invalidFile'));return;}data=fields;}
  else {if(!kind){error('p07-key-action',tk(l,'invalid'));return;}try{trustId(id);}catch{error('p07-key-target',tk(l,'invalid'));return;}data={key_id:id};if(kind==='retire_key'){try{data.at=utcEpoch(at);}catch{error('p07-key-at',tk(l,'utcHelp'));return;}}}
  if(!confirmed){error('p07-key-confirm',tk(l,'requireConfirm'));return;}
  if(!reason.trim()||new TextEncoder().encode(reason.trim()).length>256){error('p07-key-reason',k(l,'rationaleHelp'));return;}
  setProblem(null);setConfirmed(false);await submit(create?'licenses.propose_key':kind==='retire_key'?'licenses.propose_retirement':'licenses.propose_compromise',data,reason.trim());
 }}>
  {create&&<h3>{tk(l,'newKey')}</h3>}<TrustError l={l} problem={problem}/>
  {create?<><div className="service-field"><label htmlFor="p07-key-file">{tk(l,'file')}</label><p id="p07-key-file-help">{tk(l,'fileHelp')}</p><input id="p07-key-file" type="file" accept=".json,application/json" disabled={disabled} aria-describedby="p07-key-file-help" onChange={async e=>{
   const file=e.target.files?.[0],n=++seq.current;setFields(null);setConfirmed(false);setProblem(null);setReading(!!file);if(!file)return;
   try{if(file.size>4000)throw Error();const value=publicKeyFile(await file.text());const bytes=Uint8Array.from(value.public_key.match(/../g)!,v=>parseInt(v,16));const digest=await crypto.subtle.digest('SHA-256',bytes);const hash=Array.from(new Uint8Array(digest),v=>v.toString(16).padStart(2,'0')).join('');if(alive.current&&n===seq.current){setFields(value);setFingerprint(hash);}}catch{if(alive.current&&n===seq.current)error('p07-key-file',tk(l,'invalidFile'));}finally{if(alive.current&&n===seq.current)setReading(false);}
  }}/></div>{fields&&<><p role="status">{tk(l,'loaded')}</p><dl className="p07-facts">{([['keyId',fields.key_id],['issuer',fields.issuer],['fingerprint',fingerprint],['not_before',utc(Number(fields.not_before))],['not_after',utc(Number(fields.not_after))],['accept_until',utc(Number(fields.accept_until))]] as const).map(([label,value])=><div key={label}><dt>{tk(l,label)}</dt><dd><bdi dir="ltr">{value}</bdi></dd></div>)}</dl></>}</>:<>
   <fieldset className="p07-modes" id="p07-key-action" tabIndex={-1} disabled={disabled}><legend>{tk(l,'kind')}</legend>{(['retire_key','compromise_key'] as const).map(value=><label key={value}><input type="radio" name="key-change-kind" value={value} checked={kind===value} onChange={()=>{setKind(value);setConfirmed(false);setProblem(null);}}/><span>{tk(l,value)}</span></label>)}</fieldset>
   <div className="service-field"><label htmlFor="p07-key-target">{tk(l,'keyId')}</label><input id="p07-key-target" value={id} dir="ltr" readOnly/></div>
   {kind==='retire_key'&&<div className="service-field"><label htmlFor="p07-key-at">{tk(l,'retirement')}</label><input id="p07-key-at" value={at} dir="ltr" disabled={disabled} placeholder="2026-12-01T00:00:00Z" aria-describedby="p07-key-at-help" onChange={e=>{setAt(e.target.value);setConfirmed(false);}}/><p id="p07-key-at-help">{tk(l,'utcHelp')}</p></div>}
   {kind==='compromise_key'&&<p>{tk(l,'compromiseNote')}</p>}
  </>}
  <p>{tk(l,'dual')}</p><label className="p07-check"><input id="p07-key-confirm" type="checkbox" checked={confirmed} disabled={disabled||reading||create&&!fields} onChange={e=>setConfirmed(e.target.checked)}/><span>{tk(l,'confirm')}</span></label>
  <div className="service-field"><label htmlFor="p07-key-reason">{k(l,'rationale')}</label><textarea id="p07-key-reason" value={reason} maxLength={256} disabled={disabled} onChange={e=>setReason(e.target.value)}/><p>{k(l,'rationaleHelp')}</p></div>
  <div className="ops-actions"><button type="submit" disabled={disabled||reading}>{tk(l,create?'newKey':kind||'kind')}</button></div>
 </form>;
}
