import { useEffect,useRef } from "react";
import type { Locale } from "../../../../../shared/types";
import { basisLabel,t } from "../copy";
import { Money } from "../ResultCard";
import { rationaleValid } from "../documents/packet";
import { id } from "../results/contract";
import type { Preview,Scope } from "./contract";
import type { Binding } from "./use-final-request";
import type { Entry } from "./store";
import { copy } from "./copy";
import "./screen.css";
function Summary({preview:p,rationale,locale:l}:{preview:Preview;rationale:string;locale:Locale}){
 const rows=[
  [t(l,"Reported net","보고된 순액","الصافي المبلّغ عنه"),`${p.currency} ${p.net}`],
  [t(l,"Reported tax","보고된 세액","الضريبة المبلّغ عنها"),`${p.currency} ${p.tax}`],
  [t(l,"Reported result","보고된 결과","النتيجة المُبلّغة"),p.scope.result],
  [t(l,"Result author","결과 작성자","مؤلف النتيجة"),p.scope.author],
  [t(l,"Requester","요청자","مقدّم الطلب"),p.requester],
  [t(l,"Policy revision at check","확인 시 정책 버전","نسخة السياسة عند الفحص"),p.policy],
  [copy(l,"target"),p.target],
 ];
 return <div className="b09-final-summary"><p>{basisLabel(l,p.profile)}</p><Money locale={l} value={p.amount} currency={p.currency} label={basisLabel(l,p.profile)}/><dl>{rows.map(([label,value])=><div key={label}><dt>{label}</dt><dd><bdi dir="ltr">{value}</bdi></dd></div>)}<div><dt>{t(l,"Excluded contributors","승인에서 제외되는 작성자","المساهمون المستبعدون من الاعتماد")}</dt><dd><ul>{p.contributors.map(person=><li key={person}><bdi dir="ltr">{person}</bdi></li>)}</ul></dd></div><div><dt>{t(l,"Rationale","사유","المبرر")}</dt><dd className="b09-final-prose"><bdi dir="auto">{rationale}</bdi></dd></div></dl><details><summary>{t(l,"Technical reference","기술 참조","المرجع التقني")}</summary><dl>{[[t(l,"Result digest","결과 다이제스트","بصمة النتيجة"),p.scope.digest],[t(l,"Candidate digest","승인 대상 다이제스트","بصمة هدف الاعتماد"),p.candidate],[t(l,"Amount profile","금액 프로필","ملف تعريف المبلغ"),p.profile],[t(l,"Checked at","확인 시각","وقت الفحص"),p.at],[t(l,"Revision","버전","النسخة"),p.revision]].map(([label,value])=><div key={label}><dt>{label}</dt><dd><bdi dir="ltr">{value}</bdi></dd></div>)}</dl></details></div>;
}
export default function FinalRequestPanel({scope,locale:l,binding:b,blocked,onEdit}:{scope:Scope;locale:Locale;binding:Binding;blocked:boolean;onEdit:(value:boolean)=>void}){
 const key=JSON.stringify(scope),s=b.state,selected=JSON.stringify(s.scope)===key;
 const p=selected?s.preview:undefined;
 const checking=selected&&s.checking,target=selected?s.target:"",rationale=selected?s.rationale:"";
 const error=b.error??(selected?s.error:undefined);
 const heading=useRef<HTMLHeadingElement>(null),alert=useRef<HTMLParagraphElement>(null),checkButton=useRef<HTMLButtonElement>(null);
 useEffect(()=>{b.store?.select(scope);return()=>b.store?.invalidate();},[b.store,key]);
 useEffect(()=>{onEdit(selected&&!!(target||rationale||p||checking));},[selected,target,rationale,p,checking,onEdit]);
 useEffect(()=>()=>onEdit(false),[onEdit]);
 useEffect(()=>{if(error)alert.current?.focus();else if(p)heading.current?.focus();},[error,p]);
 const disabled=blocked||b.unresolved||checking||!b.store||!selected;
 const referenceInvalid=error==="invalid_final_request"&&!id(target),reasonInvalid=error==="invalid_final_request"&&!rationaleValid(rationale);
 return <section className="b09-final-request" data-final-request aria-busy={checking} aria-labelledby="final-request-title">
  <h4 id="final-request-title">{copy(l,"title")}</h4><p>{copy(l,"boundary")}</p>
  {(blocked||b.unresolved)&&<p role="status">{copy(l,b.unresolved?"locked":"blocked")}</p>}
  {error&&<p ref={alert} tabIndex={-1} role="alert" data-final-request-error>{copy(l,error==="invalid_final_request"?"invalid":"failure")} <bdi dir="ltr">{error}</bdi></p>}
  {p?<section data-final-request-preview><h4 ref={heading} tabIndex={-1}>{copy(l,"review")}</h4><p>{copy(l,"advisory")}</p><Summary preview={p} rationale={rationale} locale={l}/><p data-final-request-status>{copy(l,p.status)}</p><div className="b09-final-actions"><button type="button" className="primary" data-final-request-send disabled={disabled||p.status!=="evidence_ready"} onClick={()=>{void b.store?.submit();requestAnimationFrame(()=>document.querySelector<HTMLElement>('[data-final-request-outcome="pending"] h4')?.focus());}}>{copy(l,"send")}</button><button type="button" disabled={disabled} onClick={()=>{b.store?.edit(target,rationale);requestAnimationFrame(()=>checkButton.current?.focus());}}>{copy(l,"edit")}</button></div></section>:<>
   <label htmlFor="final-request-reference">{copy(l,"target")}<input id="final-request-reference" required maxLength={128} value={target} disabled={disabled} aria-invalid={referenceInvalid} aria-describedby="final-reference-hint" onChange={e=>b.store?.edit(e.target.value,rationale)}/></label>
   <p id="final-reference-hint">{t(l,"1–128 letters, digits, periods, underscores, colons or hyphens.","영문·숫자·마침표·밑줄·콜론·하이픈으로 1~128자.","من 1 إلى 128 حرفًا لاتينيًا أو رقمًا أو نقطة أو شرطة سفلية أو نقطتين رأسيتين أو شرطة.")}</p>
   <label htmlFor="final-request-rationale">{copy(l,"rationale")}<textarea id="final-request-rationale" required rows={5} value={rationale} disabled={disabled} aria-invalid={reasonInvalid} onChange={e=>b.store?.edit(target,e.target.value)}/></label>
   {checking&&<p role="status">{copy(l,"checking")}</p>}
   <div className="b09-final-actions"><button type="button" ref={checkButton} data-final-request-check disabled={disabled} onClick={()=>{void b.store?.check();if(!id(target)||!rationaleValid(rationale))requestAnimationFrame(()=>alert.current?.focus());}}>{copy(l,"check")}</button><button type="button" disabled={disabled||(!target&&!rationale)} onClick={()=>b.store?.clear()}>{copy(l,"clear")}</button></div>
  </>}
  {error==="reauthentication_required"&&b.reauthenticate&&<button type="button" onClick={()=>void b.reauthenticate?.()}>{copy(l,"reauth")}</button>}
 </section>;
}
function Outcome({entry:e,locale:l,binding:b}:{entry:Entry;locale:Locale;binding:Binding}){
 const heading=useRef<HTMLHeadingElement>(null),previous=useRef(e.phase);
 useEffect(()=>{if(previous.current!==e.phase&&heading.current?.parentElement?.contains(document.activeElement))heading.current.focus();previous.current=e.phase;},[e.phase]);
 return <section className="b09-final-outcome" data-final-request-outcome={e.phase} aria-busy={e.phase==="pending"}><h4 ref={heading} tabIndex={-1}>{copy(l,e.phase)}</h4><Summary preview={e.intent.preview} rationale={e.intent.rationale} locale={l}/>
  {e.phase==="unknown"&&<><p>{copy(l,"unknownHelp")}</p><p>{copy(l,"sessionOnly")}</p></>}
  {e.phase==="recorded"&&<><p>{copy(l,"recordedHelp")}</p><a href={`#/approvals?lang=${l}&view=history&approval-kind=final_erp_amount&approval-q=${encodeURIComponent(e.intent.preview.target)}`}>{copy(l,"open")}</a></>}
  {e.error&&<p role="alert"><bdi dir="ltr">{e.error}</bdi></p>}
  <div className="b09-final-actions">{e.error==="reauthentication_required"&&b.reauthenticate&&<button type="button" onClick={()=>void b.reauthenticate?.()}>{copy(l,"reauth")}</button>}{e.phase==="unknown"&&<button type="button" data-final-request-retry disabled={b.state.entries.some(other=>other.phase==="pending")} onClick={()=>void b.store?.retry(e.intent.attempt.key)}>{copy(l,"retry")}</button>}</div>
 </section>;
}
export function FinalRequestOutcomes({binding:b,locale:l}:{binding:Binding;locale:Locale}){return b.state.entries.length?<section className="b09-final-attempts"><h3>{copy(l,"attempts")}</h3>{b.state.entries.map(entry=><Outcome key={entry.intent.attempt.key} entry={entry} locale={l} binding={b}/>)}</section>:null;}
