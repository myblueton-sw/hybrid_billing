import {useEffect,useRef,useState} from "react";
import {capabilityLabel,type AuthorityArea} from "./areas";
import type {Locale} from "../../../../../shared/types";
import {attempt} from "../../../../../service/admin";
import type {Inspection} from "./inspection";
import {reviewActions,reviewDecision,type ActionReview,type ReviewAction} from "./actions";
import {rationaleValid} from "./contract";
import {t,label} from "./copy";
export default function Detail({detail,actor,locale:l,busy,onSend,onEditing,onDraft,area}:{area:AuthorityArea;detail:Inspection;actor:string;locale:Locale;busy:boolean;onSend:(review:ActionReview)=>void;onEditing:(editing:boolean)=>void;onDraft:(draft:string)=>void}){
 const [action,setAction]=useState<ReviewAction>(),[reason,setReason]=useState(""),[review,setReview]=useState<ActionReview>(),[invalid,setInvalid]=useState(false);
 useEffect(()=>{onEditing(!!action);return()=>onEditing(false);},[action,onEditing]);
 useEffect(()=>{onDraft(JSON.stringify({action,reason,review}));return()=>onDraft("");},[action,reason,review,onDraft]);
 const heading=useRef<HTMLHeadingElement>(null),reviewHeading=useRef<HTMLHeadingElement>(null),reasonInput=useRef<HTMLInputElement>(null);
 useEffect(()=>{heading.current?.focus();},[]);
 useEffect(()=>{if(review)reviewHeading.current?.focus();else if(action)reasonInput.current?.focus();},[action,review]);
 useEffect(()=>{if(invalid)reasonInput.current?.focus();},[invalid]);
 const row=detail.row;
 return <section aria-labelledby="authority-detail-title">
  <h3 id="authority-detail-title" tabIndex={-1} ref={heading}>{t(l,"Proposal details","제안 상세","تفاصيل المقترح")}</h3>
  <dl className="authority-facts">
   <dt>{t(l,"Proposal","제안","المقترح")}</dt><dd><bdi dir="ltr">{row.proposal}</bdi></dd>
   <dt>{t(l,"Recorded state","기록된 상태","الحالة المسجلة")}</dt><dd>{label(l,row.state)}</dd>
   <dt>{t(l,"Permission type","권한 유형","نوع الصلاحية")}</dt><dd>{label(l,row.kind)}</dd>
   <dt>{t(l,"Capability","허용 작업","الإجراء المسموح")}</dt><dd>{capabilityLabel(l,area,row.capability)}</dd>
   <dt>{t(l,"Recipient","대상","المستفيد")}</dt><dd><bdi dir="ltr">{row.recipient}</bdi></dd>
   <dt>{t(l,"Scope","범위","النطاق")}</dt><dd>{t(l,"Whole workspace","전체 업무 공간","مساحة العمل كاملة")}</dd>
   <dt>{t(l,"Starts (UTC)","시작 (UTC)","البداية (UTC)")}</dt><dd><bdi dir="ltr">{detail.notBefore}</bdi></dd>
   <dt>{t(l,"Ends (UTC)","종료 (UTC)","النهاية (UTC)")}</dt><dd><bdi dir="ltr">{detail.notAfter}</bdi></dd>
   {detail.ceiling&&<><dt>{t(l,"Console ceiling reference","콘솔 상한 참조","مرجع سقف وحدة التحكم")}</dt><dd><bdi dir="ltr">{detail.ceiling}</bdi></dd></>}
   <dt>{t(l,"Proposed by","제안자","مقدّم المقترح")}</dt><dd><bdi dir="ltr">{row.proposer}</bdi></dd>
   <dt>{t(l,"Proposal expires (UTC)","제안 만료 (UTC)","انتهاء المقترح (UTC)")}</dt><dd><bdi dir="ltr">{row.expires_at}</bdi></dd>
   <dt>{t(l,"Rationale","사유","المبرر")}</dt><dd>{detail.rationale}</dd>
  </dl>
  {detail.decision&&<section><h4>{t(l,"Recorded decision","기록된 결정","القرار المسجل")}</h4><p><bdi dir="ltr">{detail.decision.actor}</bdi> · <bdi dir="ltr">{detail.decision.at}</bdi></p><p>{detail.decision.rationale}</p></section>}
  <p>{t(l,"This is a historical record. Current permission and decision conditions are checked when you submit.","과거 처리 기록입니다. 현재 권한과 결정 조건은 제출 시 확인합니다.","هذا سجل سابق. تُفحص الصلاحية الحالية وشروط القرار عند الإرسال.")}</p>
  {!action&&<div className="authority-actions">{reviewActions(detail,actor).map(value=><button key={value} disabled={busy} onClick={()=>{setAction(value);setReason(value==="recover"?detail.decision!.rationale:"");setReview(undefined);setInvalid(false);}}>{label(l,value)}</button>)}</div>}
  {action&&!review&&<form onSubmit={e=>{e.preventDefault();if(!rationaleValid(reason)){setInvalid(true);return;}setReview(reviewDecision(detail,actor,action,reason,attempt(),area.domain));}}>
   <h4>{label(l,action)}</h4><label htmlFor="authority-reason">{t(l,"Decision rationale","결정 사유","مبرر القرار")}</label>
   <input id="authority-reason" ref={reasonInput} value={reason} readOnly={action==="recover"} onChange={e=>setReason(e.target.value)} aria-invalid={invalid} aria-describedby="authority-reason-help"/>
   <p id="authority-reason-help">{action==="recover"?t(l,"Only the original decision actor can recover this outcome. Recovery sends a new request for the same proposal, decision and rationale; it does not create a replacement proposal.","기존 결정을 제출한 사용자만 복구할 수 있습니다. 같은 제안·결정·사유를 유지해 새 요청으로 복구하며 대체 제안을 만들지 않습니다.","يمكن فقط لصاحب القرار الأصلي تسوية هذه النتيجة. ترسل التسوية طلبًا جديدًا للمقترح والقرار والمبرر نفسها، ولا تنشئ مقترحًا بديلًا."):t(l,"Enter a single-line rationale, up to 1,024 UTF-8 bytes.","사유를 한 줄로 입력하세요. 최대 1,024 UTF-8 바이트입니다.","أدخل المبرر في سطر واحد، بحد أقصى 1,024 بايت UTF-8.")}</p>
   {invalid&&<p role="alert">{t(l,"Check the rationale.","사유를 확인하세요.","تحقق من المبرر.")}</p>}
   <div className="authority-actions"><button type="submit">{t(l,"Review submission","제출 내용 검토","مراجعة الإرسال")}</button><button type="button" onClick={()=>setAction(undefined)}>{t(l,"Cancel","취소","إلغاء")}</button></div>
  </form>}
  {review&&<section aria-label={t(l,"Review submission","제출 내용 검토","مراجعة الإرسال")}><h4 tabIndex={-1} ref={reviewHeading}>{label(l,review.action)}</h4><p>{t(l,"Review the proposal details above and this exact rationale before sending.","위 제안 상세와 아래 사유를 확인한 뒤 전송하세요.","راجع تفاصيل المقترح أعلاه وهذا المبرر قبل الإرسال.")}</p><p>{review.intent.rationale}</p><div className="authority-actions"><button className="authority-primary" disabled={busy} onClick={()=>onSend(review)}>{t(l,"Send decision","결정 전송","إرسال القرار")}</button><button disabled={busy} onClick={()=>setReview(undefined)}>{t(l,"Change","수정","تعديل")}</button></div></section>}
 </section>;
}
