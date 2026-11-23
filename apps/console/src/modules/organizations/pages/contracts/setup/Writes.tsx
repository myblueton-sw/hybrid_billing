import { useEffect, useRef, useState, useSyncExternalStore } from "react";
import type { Session } from "../../../../../service/api";
import { useService } from "../../../../../service/context";
import type { Locale } from "../../../../../shared/types";
import { guardUnload } from "../../../../../shared/unload-intent";
import { day, ltr } from "../live-format";
import { text, type SetupWord } from "./copy";
import { SetupError } from "./context";
import { sessionSetupEditor, type SetupEditor } from "./editor";
import type { SetupReadStore } from "./read-store";
import { writeText as w, writeError, needsAuthentication } from "./write-copy";

type Props={l:Locale;session:Session;reads:SetupReadStore};
export function Writes(props:Props){
 const {session,reads,l}=props,key=JSON.stringify([session.user,session.workspace,session.csrf,reads.contract,reads.on]);
 const [binding,setBinding]=useState<{key:string;editor?:SetupEditor;error?:string}>();
 useEffect(()=>{
  let editor:SetupEditor|undefined;
  try{editor=sessionSetupEditor(session,reads.contract,reads.on);editor.invalidate();setBinding({key,editor});}
  catch(error){setBinding({key,error:error instanceof SetupError?error.code:"service_unavailable"});}
  return()=>editor?.invalidate();
 },[key]);
 if(binding?.key!==key)return <p role="status">{text(l,"loading")}</p>;
 if(!binding.editor)return <p role="alert">{writeError(l,binding.error||"service_unavailable")} {w(l,"errorCode")}: {ltr(binding.error||"service_unavailable")}</p>;
 return <EditorView key={key} {...props} editor={binding.editor}/>;
}
function EditorView({l,reads,editor}:Props&{editor:SetupEditor}){
 const state=useSyncExternalStore(editor.subscribe,editor.snapshot),writes=useSyncExternalStore(editor.writes.subscribe,editor.writes.snapshot),source=useSyncExternalStore(reads.subscribe,reads.getSnapshot);
 const heading=useRef<HTMLHeadingElement>(null),review=useRef<HTMLHeadingElement>(null),checkButton=useRef<HTMLButtonElement>(null),returnFocus=useRef(false),service=useService();
 const [localError,setLocalError]=useState("");
 const draft=state.draft,prepared=state.prepared,rows=source.history.value?.rows??[],row=rows.find(r=>r.configuration===draft.selected);
 const unlisted=!!draft.selected&&!row&&!!source.history.value&&!source.history.loading;
 const pending=writes.entries.some(e=>e.phase==="pending"),busy=pending||state.checking,error=localError||state.error||writes.error;
 useEffect(()=>{editor.invalidate();},[editor,source.context.value,source.options.value,source.history.value]);
 useEffect(()=>{if(prepared)review.current?.focus();else if(returnFocus.current){returnFocus.current=false;checkButton.current?.focus();}},[prepared]);
 useEffect(()=>{
  const guard=guardUnload(()=>{const s=editor.snapshot();return !!s.draft.reason||s.draft.preset||s.draft.languages.length>0||editor.writes.snapshot().entries.some(e=>e.phase==="pending"||e.phase==="unknown");});
  window.addEventListener("beforeunload",guard);return()=>window.removeEventListener("beforeunload",guard);
 },[editor]);
 useEffect(()=>{if(error==="session_changed"||error==="unauthenticated")service?.signOut();},[error,service]);
 const authError=[error||"",...writes.entries.filter(e=>e.phase==="unknown"||e.phase==="refused").map(e=>e.error||"")].find(needsAuthentication);
 const actionKey=state.recovery?"recover":prepared?.operation==="confirm_configuration"?"confirm":"propose";
 const focus=()=>heading.current?.focus();
 const run=async(action:()=>Promise<void>)=>{
  focus();setLocalError("");reads.invalidate();
  try{await action();}catch(error){setLocalError(error instanceof SetupError?error.code:"service_unavailable");}
  finally{await reads.load();}
 };
 const reauthenticate=async()=>{focus();editor.invalidate();setLocalError("");try{await service?.stepUp?.();}catch{setLocalError("reauthentication_required");}};
 return <>
 <section aria-labelledby="o03-setup-write" aria-busy={busy||undefined}>
  <h3 id="o03-setup-write" tabIndex={-1} ref={heading}>{w(l,"title")}</h3><p>{w(l,"note")}</p>
  {error&&<div role="alert" className="live-error"><p>{writeError(l,error)}</p><p className="live-mono">{w(l,"errorCode")}: {ltr(error)}</p></div>}
  {service?.stepUp&&authError&&<button type="button" className="live-secondary" disabled={busy} onClick={()=>void reauthenticate()}>{w(l,"auth")}</button>}
  {!prepared?<form className="o03-setup-form" onSubmit={e=>{e.preventDefault();focus();setLocalError("");void editor.check(row,draft.selected&&source.history.value?{after:source.historyAfter,snapshot:source.history.value.snapshot}:undefined);}}>
   <fieldset disabled={busy}><label htmlFor="o03-setup-choice">{w(l,"start")}</label><select id="o03-setup-choice" value={draft.selected} onChange={e=>editor.edit({selected:e.target.value})}>
    <option value="">{w(l,"new")}</option>
    {draft.selected&&!row&&<option value={draft.selected}>{draft.selected}</option>}
    {rows.filter(r=>r.setup_on&&r.state!=="confirmed").map(r=><option key={r.configuration} value={r.configuration}>{text(l,r.state as SetupWord)} · {r.configuration}</option>)}
   </select>
   {!draft.selected?<><label className="o03-setup-choice"><input type="checkbox" required checked={draft.preset} onChange={e=>editor.edit({preset:e.target.checked})}/>{w(l,"preset")}</label><fieldset><legend>{w(l,"languages")}</legend>{(["en","ko","ar"] as const).map(language=><label className="o03-setup-choice" key={language}><input type="checkbox" checked={draft.languages.includes(language)} onChange={e=>editor.edit({languages:e.target.checked?[...draft.languages,language]:draft.languages.filter(v=>v!==language)})}/><bdi lang={language}>{language==="en"?"English":language==="ko"?"한국어":"العربية"}</bdi></label>)}</fieldset></>:row&&<p>{w(l,"originalReason")}: <bdi>{row.proposal_reason}</bdi></p>}
   {row?.state!=="proposal_unknown"&&<label>{w(l,draft.selected?"confirmationReason":"proposalReason")}<textarea value={draft.reason} maxLength={2000} required onChange={e=>editor.edit({reason:e.target.value})}/></label>}
   </fieldset><button ref={checkButton} type="submit" className="live-primary" disabled={busy||!!draft.selected&&!row} aria-describedby={unlisted?"o03-setup-unlisted":undefined}>{w(l,"check")}</button>{unlisted&&<p id="o03-setup-unlisted" className="live-sub">{w(l,"unlisted")}</p>}{state.checking&&<p role="status">{text(l,"loading")}</p>}
  </form>:<div className="o03-setup-review"><h4 tabIndex={-1} ref={review}>{w(l,"review")}</h4><dl className="live-facts live-facts-grid">
   {([[w(l,"action"),w(l,actionKey)],[w(l,"contract"),prepared.contract],[w(l,"target"),prepared.target],[text(l,"selectedDate"),day(l,prepared.expected.setup_on)],[text(l,"recordedContractRevision"),prepared.expected.contract_revision],[w(l,prepared.expected.digest?"recordedProfile":"profile"),prepared.expected.billing_profile],[text(l,"recordedProfileVersion"),prepared.expected.billing_revision],[text(l,"mode"),prepared.expected.mode],[text(l,"documentProfile"),prepared.expected.profile],[text(l,"region"),prepared.expected.region],[text(l,"tax"),prepared.expected.tax],[text(l,"zone"),prepared.expected.zone],[text(l,"languages"),prepared.expected.languages],[text(l,"proposer"),prepared.expected.proposer],[w(l,"proposalReason"),prepared.expected.proposal_reason],...(prepared.operation==="confirm_configuration"?[[w(l,"confirmationReason"),prepared.rationale]]:[])]).map(([label,value])=><div key={label}><dt>{label}</dt><dd><bdi>{value}</bdi></dd></div>)}
  </dl><p>{text(l,"eligibility")}</p><div className="o03-docs-pages"><button className="live-primary" type="button" disabled={pending} onClick={()=>void run(()=>editor.submit())}>{w(l,actionKey)}</button><button className="live-secondary" type="button" disabled={pending} onClick={()=>{returnFocus.current=true;editor.invalidate();}}>{w(l,"edit")}</button></div></div>}
 </section>
 {writes.entries.length>0&&<section aria-labelledby="o03-setup-outcomes"><h3 id="o03-setup-outcomes">{w(l,"outcomes")}</h3><div aria-live="polite">{writes.entries.map(entry=><article className="o03-setup-outcome" key={entry.intent.attempt.key}><h4>{w(l,entry.intent.operation==="propose_configuration"?"proposal":"confirmation")}</h4><p>{w(l,"contract")}: {ltr(entry.intent.contract)} · {w(l,"target")}: {ltr(entry.intent.target)}</p><p>{w(l,entry.phase)}</p>{entry.error&&<><p>{writeError(l,entry.error)}</p><p className="live-mono">{w(l,"errorCode")}: {ltr(entry.error)}</p></>}{entry.receipt&&<p>{text(l,"state")}: {text(l,entry.receipt.configuration.state as SetupWord)}</p>}{entry.phase==="unknown"&&<button type="button" className="live-secondary" disabled={pending} onClick={()=>void run(()=>editor.writes.retry(entry.intent.attempt.key))}>{w(l,"retry")}</button>}</article>)}</div></section>}
 </>;
}
