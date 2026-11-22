import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import type { ApprovalRecord } from "./contract";
import type { DecisionBinding } from "./use-decision";
import type { DecisionEntry } from "./decision-store";
import { rejectionReasons, serviceDecisionRecord } from "./decision-api";
import { dc, decisionMessage } from "./decision-copy";
import { t, basisLabel } from "./copy";
import { Money } from "./ResultCard";

export function DecisionResult({ locale: l, entry, retry, refresh, refreshFailed = false, reauthenticate }: { locale: Locale; entry: DecisionEntry; retry: () => void; refresh: () => void; refreshFailed?: boolean; reauthenticate?: () => Promise<void> }) {
 const s=entry.submission;
 const resultRoot=useRef<HTMLElement>(null);
 useEffect(()=>{ if(resultRoot.current?.getClientRects().length) resultRoot.current.querySelector<HTMLElement>("[data-b09-decision-focus]")?.focus(); },[entry.phase]);
 return <section ref={resultRoot} className="b09-decision-result" data-b09-decision-result={entry.phase} aria-busy={entry.phase === "pending"}>
  <h3 tabIndex={-1} data-b09-decision-focus>{dc(l,entry.phase)}</h3>
  <p><bdi dir="ltr">{s.target}</bdi> · {dc(l,s.action)}</p>
  <p className="b09-meta">{basisLabel(l,s.amount_basis)}</p>
  <Money locale={l} value={s.amount} currency={s.currency} label={basisLabel(l,s.amount_basis)} />
  {entry.phase === "unknown" && <><p>{dc(l,"unknownHelp")}</p><p>{dc(l,"reloadLimit")}</p></>}
  {entry.error && <p role="alert">{entry.phase === "unknown" ? dc(l,entry.error === "reauthentication_required" ? "unknownReauth" : "unknownRefusal") : decisionMessage(l,entry.error)}</p>}
  {entry.phase === "recorded" && refreshFailed && <p role="alert" className="b09-refresh-failure">{dc(l,"refreshFailed")}</p>}
  <p>{dc(l,"boundary")}</p>
  <details><summary>{dc(l,"technical")}</summary><p><bdi dir="ltr">{s.attempt.request}</bdi></p>{entry.operation && <p><bdi dir="ltr">{entry.operation}</bdi></p>}{entry.error && <p><bdi dir="ltr">{entry.error}</bdi></p>}</details>
  <div className="b09-actions">
   {entry.error === "reauthentication_required" && reauthenticate && <button type="button" onClick={()=>void reauthenticate()}>{dc(l,"reauth")}</button>}
   {entry.phase === "unknown" && <button type="button" data-b09-retry-decision onClick={retry}>{dc(l,"retry")}</button>}
   {entry.phase !== "pending" && <button type="button" onClick={refresh}>{dc(l,"refresh")}</button>}
  </div>
 </section>;
}
export default function DecisionPanel({locale:l,record:r,binding, surface,refresh}:{locale:Locale;record:ApprovalRecord;binding:DecisionBinding;surface:"desktop"|"mobile";refresh:()=>void}) {
 const root=useRef<HTMLElement>(null);
 const {store,state}=binding;
 const selected=state.record?.id===r.id && state.record.kind===r.kind && state.record.amounts[0]?.basis===r.amounts[0]?.basis && state.record.evidence.revision===r.evidence.revision;
 const p=selected ? state.preparation : undefined;
 const action=selected ? state.action : undefined;
 const error=binding.error ?? (selected ? state.error : undefined);
 const entry=Object.hasOwn(state.entries,r.id) ? state.entries[r.id] : undefined;
 const checking=selected && state.checking;
 const phase=(entry && entry.phase !== "refused" ? entry.phase : undefined) ?? (checking?"checking":error?"error":action?"confirm":p?"checked":"unchecked");
 useEffect(()=>{
  if(!root.current?.getClientRects().length || phase==="unchecked")return;
  const target=root.current.querySelector<HTMLElement>("[data-b09-error-summary]") ?? root.current.querySelector<HTMLElement>("[data-b09-confirm-heading]") ?? root.current.querySelector<HTMLElement>("[data-b09-decision-result] [data-b09-decision-focus]") ?? (checking ? root.current.querySelector<HTMLElement>("[data-b09-check-status]") : null) ?? root.current.querySelector<HTMLElement>("[data-b09-decision-focus]");
  target?.focus();
 },[phase,error,action]);
 if(!binding.capability || !serviceDecisionRecord(r) || (r.status !== "pending" && !entry))return null;
 const prefix=`b09-${surface}-${r.id}`;
 const invalid=error==="invalid_decision";
 const needsRationale=invalid && (!state.rationale.trim() || new TextEncoder().encode(state.rationale).length>2000);
 const needsReason=invalid && action==="reject" && !rejectionReasons.includes(state.reason as typeof rejectionReasons[number]);
 const showEntry=!!entry && (entry.phase!=="refused" || (!p && !checking));
 const requiresReauth=[p?.approve,p?.reject,error,entry?.error].includes("reauthentication_required");
 const reauthButton=requiresReauth && !(showEntry && entry?.error === "reauthentication_required") && binding.reauthenticate && <button type="button" onClick={()=>void binding.reauthenticate?.()}>{dc(l,"reauth")}</button>;
 return <section ref={root} className="b09-decision" data-b09-decision={phase} aria-busy={checking||entry?.phase==="pending"}>
  <h3 tabIndex={-1} data-b09-decision-focus>{dc(l,"title")}</h3>
  {entry && showEntry && <DecisionResult locale={l} entry={entry} retry={()=>void store?.retry(r.id)} refresh={refresh} reauthenticate={binding.reauthenticate}/>}
  {entry && entry.phase!=="refused" ? null : <>
   <p role="status" tabIndex={-1} data-b09-check-status className="b09-meta">{checking?dc(l,"checking"):p?dc(l,"checked"):dc(l,"unchecked")}</p>
   {error && !invalid && <p role="alert">{decisionMessage(l,error)}</p>}
   {!p && <div className="b09-actions">{reauthButton}<button type="button" data-b09-check disabled={!store||checking} onClick={()=>void store?.check()}>{dc(l,"check")}</button></div>}
   {p && <>
    <Money locale={l} value={p.amount} currency={p.currency} label={basisLabel(l,p.amount_basis)}/>
    <dl className="b09-facts">
     {[[t(l,"Request reference","요청 참조","مرجع الطلب"),p.request],[t(l,"Target reference","대상 참조","مرجع الهدف"),r.target.id],[t(l,"Amount basis","금액 기준","أساس المبلغ"),basisLabel(l,p.amount_basis)],[t(l,"Original author","원 작성자","المؤلف الأصلي"),p.author],[t(l,"Requester in source record","요청자 (원천 기록)","مقدّم الطلب في سجل المصدر"),p.requester],[t(l,"Recorded policy revision","기록된 정책 버전","نسخة السياسة المسجّلة"),p.policy_revision],[dc(l,"checkedAt"),p.checked_at]].map(([label,value])=><div key={label}><dt>{label}</dt><dd><bdi dir="auto">{value}</bdi></dd></div>)}
    </dl>
    {action ? <form className="b09-decision-form" noValidate onSubmit={event=>{event.preventDefault();void store?.submit().then(()=>requestAnimationFrame(()=>{if(root.current?.getClientRects().length)root.current.querySelector<HTMLElement>("[data-b09-error-summary]")?.focus();}));}}>
     <h4 tabIndex={-1} data-b09-confirm-heading>{dc(l,action==="approve"?"confirmApprove":"confirmReject")}</h4>
     <p>{dc(l,"boundary")}</p>
     {invalid && <div className="b09-error" tabIndex={-1} data-b09-error-summary role="alert"><h4>{dc(l,"validation")}</h4><ul>
      {needsReason && <li><a href={`#${prefix}-reason`} onClick={e=>{e.preventDefault();root.current?.querySelector<HTMLSelectElement>("select")?.focus();}}>{dc(l,"reason")}</a></li>}
      {needsRationale && <li><a href={`#${prefix}-rationale`} onClick={e=>{e.preventDefault();root.current?.querySelector<HTMLTextAreaElement>("textarea")?.focus();}}>{dc(l,"rationale")}</a></li>}
     </ul></div>}
     {action==="reject" && <label htmlFor={`${prefix}-reason`}>{dc(l,"reason")}<select id={`${prefix}-reason`} required value={state.reason} aria-invalid={needsReason} onChange={e=>store?.edit(state.rationale,e.target.value)}><option value="">{dc(l,"chooseReason")}</option>{rejectionReasons.map(reason=><option key={reason} value={reason}>{dc(l,reason)}</option>)}</select></label>}
     <label htmlFor={`${prefix}-rationale`}>{dc(l,"rationale")}<textarea id={`${prefix}-rationale`} rows={5} required value={state.rationale} aria-invalid={needsRationale} aria-describedby={`${prefix}-hint`} onChange={e=>store?.edit(e.target.value,state.reason)}/></label>
     <p className="b09-meta" id={`${prefix}-hint`}>{dc(l,"rationaleHint")}</p>
     <dl className="b09-facts b09-confirm-summary">
      <div><dt>{dc(l,"title")}</dt><dd>{dc(l,action)}</dd></div>
      <div><dt>{basisLabel(l,p.amount_basis)}</dt><dd><Money locale={l} value={p.amount} currency={p.currency} label={basisLabel(l,p.amount_basis)}/></dd></div>
      <div><dt>{t(l,"Target reference","대상 참조","مرجع الهدف")}</dt><dd><bdi dir="ltr">{r.target.id}</bdi></dd></div>
     </dl>
     <div className="b09-actions"><button type="submit" className="b09-confirm" data-b09-confirm>{dc(l,action==="approve"?"confirmApprove":"confirmReject")}</button><button type="button" data-b09-cancel onClick={()=>{store?.cancel();requestAnimationFrame(()=>{if(root.current?.getClientRects().length)root.current.querySelector<HTMLElement>(`[data-b09-choose="${action}"]`)?.focus();});}}>{dc(l,"cancel")}</button></div>
    </form> : <>
     <ul className="b09-decision-actions">{(["approve","reject"] as const).map(kind=><li key={kind}><div><strong>{dc(l,kind)}</strong><p>{decisionMessage(l,p[kind])}</p></div>{p[kind]==="eligible" && <button type="button" data-b09-choose={kind} onClick={()=>store?.choose(kind)}>{dc(l,kind)}</button>}</li>)}</ul>
     <div className="b09-actions">{reauthButton}<button type="button" data-b09-recheck onClick={()=>void store?.check()}>{dc(l,"check")}</button></div>
    </>}
   </>}
  </>}
 </section>;
}
