import {useEffect,useMemo,useRef,useState} from "react";
import type {Session} from "../../../../../service/api";
import type {Locale} from "../../../../../shared/types";
import {attempt} from "../../../../../service/admin";
import {AuthorityError,type Capability,type Policy,type Read} from "./contract";
import {contextPage,type Cursor} from "./reads";
import {readTransport} from "./transport";
import {proposalContext} from "./proposal-context";
import {ProposalError,reviewProposal,type ProposalContext,type ProposalDraft,type ProposalKind,type ProposalReview} from "./proposal-model";
import {t,label} from "./copy";
const blank:ProposalDraft={recipient:"",ceiling:"",notBefore:"",notAfter:"",rationale:""};
const fieldLabel=(l:Locale,field:string)=>({recipient:t(l,"Recipient","대상 사용자","المستفيد"),ceiling:t(l,"Console permission ceiling","콘솔 권한 상한","سقف صلاحيات وحدة التحكم"),not_before:t(l,"Starts (UTC)","시작 (UTC)","البداية (UTC)"),not_after:t(l,"Ends (UTC)","종료 (UTC)","النهاية (UTC)"),rationale:t(l,"Rationale","사유","المبرر"),context:t(l,"Current options","현재 선택 항목","الخيارات الحالية")}[field]??field);
type Shared={locale:Locale;session:Session;busy:boolean;onSend:(review:ProposalReview)=>void;onCancel:()=>void;onDraft:(draft:string)=>void};
function Fields({locale:l,session,busy,onSend,onCancel,onDraft,read,kind,capability,onReview}:{read:Read;kind:ProposalKind;capability:Capability;onReview:(reviewing:boolean)=>void}&Shared){
 const [context,setContext]=useState<ProposalContext>(),[draft,setDraft]=useState<ProposalDraft>(blank),[review,setReview]=useState<ProposalReview>(),[error,setError]=useState(""),[problems,setProblems]=useState<readonly string[]>([]),[loading,setLoading]=useState(false);
 const epoch=useRef(0),alive=useRef(true),heading=useRef<HTMLHeadingElement>(null),summary=useRef<HTMLDivElement>(null),changeFocus=useRef(false);
 const cursorsRef=useRef<{ceilings?:Cursor;recipients?:Cursor}>({});
 const load=async(next?:{ceilings?:Cursor;recipients?:Cursor})=>{
  if(next&&Object.values(draft).some(Boolean)&&!window.confirm(t(l,"Discard the unsent proposal inputs before changing options?","선택 항목을 바꾸기 전에 전송하지 않은 제안 입력을 버리시겠습니까?","هل تريد تجاهل مدخلات المقترح غير المرسل قبل تغيير الخيارات؟")))return;
  const cursors=next?{...cursorsRef.current,...next}:{};cursorsRef.current=cursors;
  const current=++epoch.current;setContext(undefined);setReview(undefined);setDraft(blank);setProblems([]);setError("");setLoading(true);
  try{const value=await proposalContext(read,session.workspace,session.user,kind,capability,cursors);if(alive.current&&current===epoch.current)setContext(value);}
  catch(e){if(alive.current&&current===epoch.current)setError(e instanceof AuthorityError?e.code:"service_unavailable");}
  finally{if(alive.current&&current===epoch.current)setLoading(false);}
 };
 useEffect(()=>{alive.current=true;void load();return()=>{alive.current=false;epoch.current++;};},[read,kind,capability]);
 useEffect(()=>{onDraft(JSON.stringify({kind,capability,context,draft,review,error,problems}));return()=>onDraft("");},[kind,capability,context,draft,review,error,problems,onDraft]);
 useEffect(()=>{onReview(!!review);return()=>onReview(false);},[review,onReview]);
 useEffect(()=>{if(review)heading.current?.focus();else if(changeFocus.current){changeFocus.current=false;document.getElementById("authority-not_before")?.focus();}},[review]);
 useEffect(()=>{if(problems.length)summary.current?.focus();},[problems]);
 const edit=(key:keyof ProposalDraft,value:string)=>{setDraft(old=>({...old,[key]:value,...(key==="ceiling"?{notBefore:"",notAfter:""}:{})}));setProblems([]);};
 const change=()=>{changeFocus.current=true;setReview(undefined);};
 if(loading)return <p role="status">{t(l,"Loading current options…","현재 선택 항목을 불러오는 중…","جارٍ تحميل الخيارات الحالية…")}</p>;
 if(error)return <div role="alert"><p>{t(l,"The options could not be verified. Reload to start with current evidence.","선택 항목을 확인하지 못했습니다. 새로 조회해 현재 근거로 시작하세요.","تعذر التحقق من الخيارات. أعد التحميل للبدء بأدلة حالية.")}</p><details><summary>{t(l,"Technical reference","기술 참조","المرجع التقني")}</summary>{error}</details><button onClick={()=>void load()}>{t(l,"Reload options","선택 항목 새로 조회","إعادة تحميل الخيارات")}</button></div>;
 if(!context)return null;
 const selectedRecipient=context.recipients?.rows.find(row=>"user" in row&&row.user===draft.recipient);
 const selectedCeiling=context.ceilings.rows.find(row=>"ceiling" in row&&row.ceiling===draft.ceiling);
 if(review)return <section className="authority-proposal-review"><h3 tabIndex={-1} ref={heading}>{t(l,"Review permission proposal","권한 제안 검토","مراجعة مقترح الصلاحية")}</h3><dl className="authority-facts">
  <dt>{t(l,"Permission type","권한 유형","نوع الصلاحية")}</dt><dd>{label(l,kind)}</dd><dt>{t(l,"Capability","허용 작업","الإجراء المسموح")}</dt><dd>{label(l,capability)}</dd>
  <dt>{t(l,"Recipient","대상","المستفيد")}</dt><dd>{kind==="console_ceiling"?t(l,"Console client","콘솔 클라이언트","عميل وحدة التحكم"):selectedRecipient&&"login" in selectedRecipient?selectedRecipient.login:""}<small><bdi dir="ltr">{kind==="console_ceiling"?"console":draft.recipient}</bdi></small></dd><dt>{t(l,"Scope","범위","النطاق")}</dt><dd>{t(l,"Whole workspace","전체 업무 공간","مساحة العمل كاملة")}</dd>
  <dt>{fieldLabel(l,"not_before")}</dt><dd><bdi dir="ltr">{draft.notBefore}</bdi></dd><dt>{fieldLabel(l,"not_after")}</dt><dd><bdi dir="ltr">{draft.notAfter}</bdi></dd>
  {kind==="human_grant"&&<><dt>{fieldLabel(l,"ceiling")}</dt><dd><bdi dir="ltr">{draft.ceiling}</bdi>{selectedCeiling&&"not_before" in selectedCeiling&&<><small>{fieldLabel(l,"not_before")}: <bdi dir="ltr">{selectedCeiling.not_before}</bdi></small><small>{fieldLabel(l,"not_after")}: <bdi dir="ltr">{selectedCeiling.not_after}</bdi></small></>}</dd></>}<dt>{fieldLabel(l,"rationale")}</dt><dd>{draft.rationale}</dd>
 <dt>{t(l,"Proposal lifetime in seconds","제안 유효기간 (초)","مدة صلاحية المقترح بالثواني")}</dt><dd>{context.ceilings.policy.proposal_lifetime_seconds}</dd><dt>{t(l,"Observed (UTC)","조회 시각 (UTC)","وقت القراءة (UTC)")}</dt><dd><bdi dir="ltr">{context.ceilings.at}</bdi></dd></dl><p>{t(l,"Sending creates a proposal. Another eligible governor must approve it before permission is granted. Current conditions are checked at submission.","전송하면 제안이 생성됩니다. 권한 부여 전 다른 적격 관리자의 승인이 필요합니다. 현재 조건은 제출 시 확인합니다.","ينشئ الإرسال مقترحًا. يجب أن يعتمده مسؤول آخر مؤهل قبل منح الصلاحية. تُفحص الشروط الحالية عند الإرسال.")}</p><div className="authority-actions"><button className="authority-primary" disabled={busy} onClick={()=>onSend(review)}>{t(l,"Send proposal","제안 전송","إرسال المقترح")}</button><button disabled={busy} onClick={change}>{t(l,"Change","수정","تعديل")}</button><button disabled={busy} onClick={onCancel}>{t(l,"Cancel","취소","إلغاء")}</button></div></section>;
 return <form className="authority-proposal-form" onSubmit={e=>{e.preventDefault();try{const a=attempt();setReview(reviewProposal(context,draft,`result-authority-${a.request}`,a));setProblems([]);}catch(e){setProblems(e instanceof ProposalError?e.fields:["context"]);}}}>
  {!!problems.length&&<div role="alert" tabIndex={-1} ref={summary}><h4>{t(l,"Check the proposal","제안 내용을 확인하세요","تحقق من المقترح")}</h4><ul>{problems.map(field=><li key={field}><a href={`#authority-${field}`} onClick={e=>{e.preventDefault();document.getElementById(`authority-${field}`)?.focus();}}>{fieldLabel(l,field)}</a></li>)}</ul></div>}
  <div id="authority-context" tabIndex={-1}><p>{t(l,"Observed (UTC)","조회 시각 (UTC)","وقت القراءة (UTC)")}: <bdi dir="ltr">{context.ceilings.at}</bdi></p><p>{t(l,"Maximum validity in seconds","최대 유효기간 (초)","أقصى مدة صلاحية بالثواني")}: <bdi>{context.ceilings.policy.max_grant_validity_seconds}</bdi>. {t(l,"Proposal lifetime in seconds","제안 유효기간 (초)","مدة صلاحية المقترح بالثواني")}: <bdi>{context.ceilings.policy.proposal_lifetime_seconds}</bdi>.</p></div>
  {kind==="human_grant"&&<>
   <fieldset id="authority-recipient" tabIndex={-1}><legend>{fieldLabel(l,"recipient")}</legend>{context.recipients!.rows.map(row=>"user" in row&&<label className="authority-choice" key={row.user}><input type="radio" name="authority-recipient" checked={draft.recipient===row.user} onChange={()=>edit("recipient",row.user)}/><span>{row.login||row.user}<small><bdi dir="ltr">{row.user}</bdi></small></span></label>)}{!context.recipients!.rows.length&&<p>{t(l,"No eligible recipients in this page.","이 페이지에 적격 대상 사용자가 없습니다.","لا يوجد مستفيدون مؤهلون في هذه الصفحة.")}</p>}
    {context.recipients!.next&&<button type="button" onClick={()=>void load({recipients:{after:context.recipients!.next,snapshot:context.recipients!.snapshot}})}>{t(l,"Next recipients","다음 대상 사용자","المستفيدون التاليون")}</button>}
   </fieldset>
   <fieldset id="authority-ceiling" tabIndex={-1}><legend>{fieldLabel(l,"ceiling")}</legend>{context.ceilings.rows.map(row=>"ceiling" in row&&<label className="authority-choice" key={row.ceiling}><input type="radio" name="authority-ceiling" checked={draft.ceiling===row.ceiling} onChange={()=>edit("ceiling",row.ceiling)}/><span><bdi dir="ltr">{row.ceiling}</bdi><small>{fieldLabel(l,"not_before")}: <bdi dir="ltr">{row.not_before}</bdi></small><small>{fieldLabel(l,"not_after")}: <bdi dir="ltr">{row.not_after}</bdi></small></span></label>)}{!context.ceilings.rows.length&&<p>{t(l,"No matching active console ceiling on this page. A matching active ceiling is required for a person permission proposal.","이 페이지에 일치하는 활성 콘솔 상한이 없습니다. 사용자 권한 제안에는 일치하는 활성 상한이 필요합니다.","لا يوجد سقف نشط مطابق لوحدة التحكم في هذه الصفحة. يتطلب مقترح صلاحية الشخص سقفًا نشطًا مطابقًا.")}</p>}
    {context.ceilings.next&&<button type="button" onClick={()=>void load({ceilings:{after:context.ceilings.next,snapshot:context.ceilings.snapshot}})}>{t(l,"Next ceilings","다음 상한","السقوف التالية")}</button>}
   </fieldset>
  </>}
  <label htmlFor="authority-not_before">{fieldLabel(l,"not_before")}</label><input id="authority-not_before" dir="ltr" value={draft.notBefore} onChange={e=>edit("notBefore",e.target.value)} aria-describedby="authority-time-help" aria-invalid={problems.includes("not_before")}/>
  <label htmlFor="authority-not_after">{fieldLabel(l,"not_after")}</label><input id="authority-not_after" dir="ltr" value={draft.notAfter} onChange={e=>edit("notAfter",e.target.value)} aria-describedby="authority-time-help" aria-invalid={problems.includes("not_after")}/>
  <div id="authority-time-help"><p>{t(l,"Enter exact UTC timestamps within the maximum validity above. Example:","위 최대 유효기간 이내로 UTC 시각을 정확히 입력하세요. 예:","أدخل أوقات UTC بدقة ضمن أقصى مدة صلاحية أعلاه. مثال:")} <bdi dir="ltr">2026-11-22T09:00:00.5Z</bdi></p>{kind==="human_grant"&&<p>{t(l,"Keep all fractional digits from the selected ceiling. The person's validity must fit within that ceiling.","선택한 상한의 소수 자릿수를 유지하세요. 사용자 유효기간은 그 상한 안에 있어야 합니다.","احتفظ بكل الخانات العشرية للسقف المحدد. يجب أن تقع صلاحية الشخص ضمن ذلك السقف.")}</p>}</div>
  <label htmlFor="authority-rationale">{fieldLabel(l,"rationale")}</label><input id="authority-rationale" value={draft.rationale} onChange={e=>edit("rationale",e.target.value)} aria-describedby="authority-proposal-reason-help" aria-invalid={problems.includes("rationale")}/><p id="authority-proposal-reason-help">{t(l,"Use one line, up to 1,024 UTF-8 bytes.","한 줄로 입력하세요. 최대 1,024 UTF-8 바이트입니다.","استخدم سطرًا واحدًا بحد أقصى 1,024 بايت UTF-8.")}</p>
  <div className="authority-actions"><button type="submit" disabled={busy}>{t(l,"Review proposal","제안 검토","مراجعة المقترح")}</button><button type="button" disabled={busy} onClick={onCancel}>{t(l,"Cancel","취소","إلغاء")}</button></div>
 </form>;
}
export default function Proposal(props:Shared){
 const {locale:l,session,onDraft}=props,read=useMemo(()=>readTransport(session),[session.user,session.workspace,session.csrf]);
 const [reviewing,setReviewing]=useState(false);
 const [kind,setKind]=useState<ProposalKind|"">(""),[selected,setSelected]=useState<Capability|"">(""),[policy,setPolicy]=useState<Policy>(),[error,setError]=useState(""),[retry,setRetry]=useState(0);
 const heading=useRef<HTMLHeadingElement>(null);
 useEffect(()=>{heading.current?.focus();},[]);
 useEffect(()=>{let alive=true;setPolicy(undefined);setError("");contextPage(read,session.workspace,session.user,"ceilings","propose").then(value=>{if(alive)setPolicy(value.policy);},e=>{if(alive)setError(e instanceof AuthorityError?e.code:"service_unavailable");});return()=>{alive=false;};},[read,retry]);
 useEffect(()=>{if(!kind||!selected)onDraft(JSON.stringify({kind,selected,policy,error}));},[kind,selected,policy,error,onDraft]);
 const changeScope=()=>!(kind&&selected)||window.confirm(t(l,"Discard the unsent proposal inputs before changing options?","선택 항목을 바꾸기 전에 전송하지 않은 제안 입력을 버리시겠습니까?","هل تريد تجاهل مدخلات المقترح غير المرسل قبل تغيير الخيارات؟"));
 return <section className="authority-proposal"><header className="authority-entry-heading"><h3 tabIndex={-1} ref={heading}>{t(l,"New permission proposal","새 권한 제안","مقترح صلاحية جديد")}</h3><button disabled={props.busy} onClick={props.onCancel}>{t(l,"Close proposal entry","제안 입력 닫기","إغلاق إدخال المقترح")}</button></header>
  {error?<div role="alert"><p>{t(l,"Permission policy could not be loaded.","권한 정책을 불러오지 못했습니다.","تعذر تحميل سياسة الصلاحيات.")}</p><details><summary>{t(l,"Technical reference","기술 참조","المرجع التقني")}</summary>{error}</details><button onClick={()=>setRetry(v=>v+1)}>{t(l,"Try again","다시 시도","إعادة المحاولة")}</button></div>:!policy?<p role="status">{t(l,"Loading permission policy…","권한 정책을 불러오는 중…","جارٍ تحميل سياسة الصلاحيات…")}</p>:<>
  <div className="authority-scope-choices" hidden={reviewing}><fieldset><legend>{t(l,"Permission type","권한 유형","نوع الصلاحية")}</legend>{(["console_ceiling","human_grant"] as const).map(value=><label className="authority-choice" key={value}><input type="radio" name="authority-kind" checked={kind===value} onChange={()=>{if(!changeScope())return;setKind(value);setSelected("");}}/>{label(l,value)}</label>)}</fieldset>
  <fieldset className="authority-capabilities"><legend>{t(l,"Capability","허용 작업","الإجراء المسموح")}</legend>{policy.capabilities.map(value=><label className="authority-choice" key={value}><input type="radio" name="authority-capability" checked={selected===value} onChange={()=>{if(changeScope())setSelected(value);}} disabled={!kind}/>{label(l,value)}</label>)}</fieldset>
  </div>
  {kind&&selected&&<Fields onReview={setReviewing} key={`${kind}:${selected}`} {...props} kind={kind} capability={selected} read={read}/>}
  </>}
 </section>;
}
