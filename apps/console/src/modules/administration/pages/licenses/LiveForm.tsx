import { useEffect, useRef, useState, type FormEvent } from "react";
import type { Locale } from "../../../../shared/types";
import { licenseSignedFields, signedEnvelopeOf, type LicenseMode, type LicensePage, type SignedEnvelope } from "../../../../service/licenses";
import { k, label } from "./live-copy";
import type { TrustCommand } from "../../../../service/license-trust";
export type LicenseCommand = TrustCommand | "licenses.initialize" | "licenses.select_mode" | "licenses.register" | "licenses.activate";
export type SubmitLicense = (command: LicenseCommand, fields: Record<string, string>, rationale: string, scope?: {target:string;revision:string}) => Promise<void>;
export default function LiveForm({ l, page, offline, busy, submit }: { l: Locale; page: LicensePage; offline: boolean; busy: boolean; submit: SubmitLicense }) {
 const [mode,setMode] = useState<LicenseMode | "">("");
 const [rationale,setRationale] = useState("");
 const [envelope,setEnvelope] = useState<SignedEnvelope | null>(null);
 const [consent,setConsent] = useState(false);
 const [problem,setProblem] = useState<{field:string;text:string} | null>(null);
 const [reading,setReading] = useState(false);
 const alive = useRef(true), sequence = useRef(0), summary = useRef<HTMLDivElement>(null);
 useEffect(()=>{alive.current=true;return()=>{alive.current=false;sequence.current++;};},[]);
 useEffect(()=>{if(problem) summary.current?.focus();},[problem]);
 if (page.state === "not_configured") return null;
 const initialized = page.state === "configured";
 const activation = page.status?.mode === "evaluation";
 const registration = page.status?.mode === "commercial";
 const error = (field:string,text:string) => setProblem({field,text});
 const send = async(e:FormEvent) => {
  e.preventDefault(); if(busy || reading) return;
  if(!rationale.trim() || new TextEncoder().encode(rationale.trim()).length>256) {error("p07-rationale",k(l,"rationaleHelp"));return;}
  if(initialized && !offline && !mode) {error("p07-mode",k(l,"needMode"));return;}
  if(initialized && offline && !envelope) {error("p07-file",k(l,"missingFile"));return;}
  if(initialized && offline && !activation && !registration) {error("p07-file",k(l,"needMode"));return;}
  if(initialized && offline && activation && !envelope?.terms) {error("p07-file",k(l,"missingTerms"));return;}
  if(initialized && offline && activation && (!envelope?.terms || !consent)) {error("p07-consent",k(l,"consentRequired"));return;}
  setProblem(null);
  const command = !initialized ? "licenses.initialize" : !offline ? "licenses.select_mode" : activation ? "licenses.activate" : "licenses.register";
  const fields = !initialized ? {} : !offline ? {mode} : licenseSignedFields(envelope!,activation);
  await submit(command,fields,rationale.trim());
 };
 return <form className="p07-form ops-review" onSubmit={(e)=>void send(e)} noValidate>
  {(!initialized || !offline)&&<h3>{k(l,!initialized?"initialize":"choose")}</h3>}
  {problem && <div ref={summary} className="p07-errors" tabIndex={-1} role="alert"><h3>{k(l,"errorTitle")}</h3><a href={"#"+problem.field} onClick={(e)=>{e.preventDefault();document.getElementById(problem.field)?.focus();}}>{problem.text}</a></div>}
  {initialized && !offline && <fieldset className="p07-modes" id="p07-mode" tabIndex={-1} aria-invalid={problem?.field==="p07-mode" || undefined} aria-describedby="p07-mode-help" disabled={busy}>
   <legend>{k(l,"mode")}</legend>
   {(["noncommercial","evaluation","commercial"] as const).map((m)=><label key={m}><input type="radio" name="license-mode" value={m} checked={mode===m} onChange={()=>setMode(m)}/><span>{m==="noncommercial"?k(l,"noncommercialOption"):label(l,m)}</span></label>)}
   <p id="p07-mode-help">{k(l,"modeNote")}</p>{problem?.field==="p07-mode" && <p className="service-field-error">{problem.text}</p>}
  </fieldset>}
  {initialized && offline && <>
   <div className="service-field"><label htmlFor="p07-file">{k(l,"file")}</label><p id="p07-file-help">{k(l,"fileHelp")}</p>
    <input id="p07-file" type="file" accept=".json,application/json" disabled={busy} aria-describedby="p07-file-help" aria-invalid={problem?.field==="p07-file" || undefined} onChange={async(e)=>{
     const file=e.target.files?.[0], n=++sequence.current;setEnvelope(null);setConsent(false);setProblem(null);setReading(!!file);
     if(!file) return;
     try {if(file.size>12000) throw Error();const parsed=signedEnvelopeOf(await file.text());if(alive.current && n===sequence.current)setEnvelope(parsed);}
     catch {if(alive.current && n===sequence.current)error("p07-file",k(l,"invalidFile"));}
     finally {if(alive.current && n===sequence.current)setReading(false);}
    }}/>
    {problem?.field==="p07-file" && <p className="service-field-error">{problem.text}</p>}
   </div>
   {envelope && <p role="status">{k(l,"loaded")}</p>}
   {activation && <div>
    <p>{k(l,"terms")}: <bdi className="p07-terms" dir="auto">{envelope?.terms || "—"}</bdi></p><p id="p07-terms-help">{k(l,"termsHelp")}</p>
    <label className="p07-check"><input id="p07-consent" type="checkbox" checked={consent} disabled={!envelope?.terms || busy} onChange={(e)=>setConsent(e.target.checked)} aria-describedby="p07-terms-help" aria-invalid={problem?.field==="p07-consent" || undefined}/><span>{k(l,"consent")}</span></label>
    {problem?.field==="p07-consent" && <p className="service-field-error">{problem.text}</p>}
   </div>}
   {!activation && !registration && <p>{k(l,"needMode")}</p>}
  </>}
  <div className="service-field"><label htmlFor="p07-rationale">{k(l,"rationale")}</label>
   <textarea id="p07-rationale" value={rationale} disabled={busy} maxLength={256} onChange={(e)=>setRationale(e.target.value)} aria-describedby="p07-rationale-help" aria-invalid={problem?.field==="p07-rationale" || undefined}/>
   <p id="p07-rationale-help">{k(l,"rationaleHelp")}</p>{problem?.field==="p07-rationale" && <p className="service-field-error">{problem.text}</p>}
  </div>
  <div className="ops-actions"><button type="submit" disabled={busy || reading || initialized && offline && !activation && !registration}>{k(l,busy?"busy":!initialized?"initialize":!offline?"choose":activation?"activate":"register")}</button></div>
 </form>;
}
