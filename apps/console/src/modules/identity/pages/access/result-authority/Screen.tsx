import {useEffect,useRef,useState} from "react";
import type {Session} from "../../../../../service/api";
import type {Locale} from "../../../../../shared/types";
import {useAuthorityReads} from "./use-reads";
import {useAuthorityAttempts} from "./use-attempts";
import {AuthorityError} from "./contract";
import type {Load} from "./read-store";
import type {ActionReview} from "./actions";
import {t,label} from "./copy";
import Detail from "./Detail";
import Proposal from "./Proposal";
import type {ProposalReview} from "./proposal-model";
import {capabilityLabel,resultArea,words,type AuthorityArea} from "./areas";
import "./screen.css";
function Feedback({state,locale:l}:{state:Load<unknown>;locale:Locale}){
 if(state.phase==="loading")return <p role="status">{t(l,"Loading…","불러오는 중…","جارٍ التحميل…")}</p>;
 if(state.phase!=="error")return null;
 return <div role="alert"><p>{t(l,"The evidence could not be verified. Start a new read after checking your session and permissions.","근거를 확인하지 못했습니다. 세션과 권한을 확인한 뒤 새로 조회하세요.","تعذر التحقق من الأدلة. تحقق من الجلسة والصلاحيات ثم ابدأ قراءة جديدة.")}</p><details><summary>{t(l,"Technical reference","기술 참조","المرجع التقني")}</summary><bdi dir="ltr">{state.error}</bdi></details></div>;
}
export default function AuthorityScreen({session,locale:l,proposal="",client="",onState,onDirty,area=resultArea}:{area?:AuthorityArea;session:Session;locale:Locale;proposal?:string;client?:string;onState:(state:string,items:string[])=>void;onDirty:(dirty:boolean)=>void}){
 const {store,state}=useAuthorityReads(session,area.domain),attempts=useAuthorityAttempts(session,area.domain);
 const [error,setError]=useState(""),[pane,setPane]=useState<"list"|"detail">("list");
 const [entering,setEntering]=useState(false);
 const [editing,setEditing]=useState(false),[draft,setDraft]=useState("");
 const selectedButton=useRef<HTMLButtonElement|null>(null),newButton=useRef<HTMLButtonElement>(null),listHeading=useRef<HTMLHeadingElement>(null),focusTarget=useRef<"selection"|"new">("selection");
 const [focusRequest,setFocusRequest]=useState(0);
 useEffect(()=>{if(!focusRequest)return;const node=focusTarget.current==="new"?newButton.current:selectedButton.current?.isConnected?selectedButton.current:listHeading.current;node?.focus();},[focusRequest]);
 const busy=attempts.entries.some(e=>e.phase==="pending"),page=state.history.value,detail=state.detail.value;
 const fingerprint=JSON.stringify({state,entries:attempts.entries,pane,draft,error,entering});
 useEffect(()=>{onState(fingerprint,["history","inspection","decision-review","attempts","proposal-fields","proposal-review"].map(item=>`M03:${area.item}:${item}`));},[fingerprint,onState,area]);
 useEffect(()=>{onDirty(busy||editing||entering);return()=>onDirty(false);},[busy,editing,entering,onDirty]);
 useEffect(()=>{if(store&&proposal){setPane("detail");void store.inspect(proposal);}},[store,proposal]);
 const discard=()=>!(editing||entering)||window.confirm(t(l,"Discard the unsent permission draft?","전송하지 않은 권한 입력을 버리시겠습니까?","هل تريد تجاهل مسودة الصلاحية غير المرسلة؟"));
 const send=(review:ActionReview)=>{
  if(!store||!attempts.store)return;
  try{setError("");void store.followMutation(review.intent.target,attempts.store.decide(review));}
  catch(e){setError(e instanceof AuthorityError?e.code:"service_unavailable");}
 };
 const propose=(review:ProposalReview)=>{
  if(!store||!attempts.store)return;
  try{const work=attempts.store.propose(review.intent);setEntering(false);setPane("detail");setError("");void store.followMutation(review.intent.target,work);}
  catch(e){setError(e instanceof AuthorityError?e.code:"service_unavailable");}
 };
 if(client&&client!=="console")return <section className="authority-refusal"><h2>{words(l,area.title)}</h2><p role="alert">{words(l,area.refusal)}</p></section>;
 return <div className="result-authority" data-authority-pane={pane}>
  <header><h2>{words(l,area.title)}</h2><p>{words(l,area.intro)}</p></header>
  {!entering&&<div className="authority-actions"><button disabled={!store||busy} onClick={()=>{if(!discard())return;setEntering(false);setPane("list");void store?.history();}}>{t(l,"Start new read","새로 조회","بدء قراءة جديدة")}</button><button ref={newButton} disabled={!store||!attempts.store||busy||attempts.entries.some(e=>e.intent.action.startsWith("propose_")&&attempts.store?.unresolved(e))} onClick={()=>{if(!discard())return;store?.clearDetail();setEditing(false);setEntering(true);setError("");}}>{t(l,"New permission proposal","새 권한 제안","مقترح صلاحية جديد")}</button></div>}
  {(error||attempts.error)&&<div role="alert"><p>{t(l,"The action could not be started. Check the technical reference.","작업을 시작하지 못했습니다. 기술 참조를 확인하세요.","تعذر بدء الإجراء. تحقق من المرجع التقني.")}</p><details><summary>{t(l,"Technical reference","기술 참조","المرجع التقني")}</summary>{error||attempts.error}</details></div>}
  {entering?<Proposal area={area} session={session} locale={l} busy={busy} onDraft={setDraft} onSend={propose} onCancel={()=>{if(discard()){setEntering(false);focusTarget.current="new";setFocusRequest(v=>v+1);}}}/>:<div className="authority-layout">
   <section className="authority-list" aria-labelledby="authority-history-title"><h3 id="authority-history-title" ref={listHeading} tabIndex={-1}>{t(l,"Proposal history","제안 이력","سجل المقترحات")}</h3><Feedback state={state.history} locale={l}/>
    {page&&<><p>{t(l,"Observed (UTC)","조회 시각 (UTC)","وقت القراءة (UTC)")}: <bdi dir="ltr">{page.at}</bdi></p>{!page.rows.length&&<p>{t(l,"No proposals in this view.","이 조회 범위에 제안이 없습니다.","لا توجد مقترحات في هذا العرض.")}</p>}
    <ul>{page.rows.map(row=><li key={row.proposal}><button disabled={busy} aria-pressed={state.selected===row.proposal} onClick={e=>{if(!discard())return;selectedButton.current=e.currentTarget;setEntering(false);setPane("detail");void store?.inspect(row.proposal,row.digest);}}><strong>{t(l,"Permission proposal","권한 제안","مقترح صلاحية")} · {label(l,row.state)}</strong><span>{t(l,"Permission type","권한 유형","نوع الصلاحية")}: {label(l,row.kind)}</span><span>{t(l,"Capability","허용 작업","الإجراء المسموح")}: {capabilityLabel(l,area,row.capability)}</span><span>{t(l,"Recipient","대상","المستفيد")}: <bdi dir="ltr">{row.recipient}</bdi></span><small><bdi dir="ltr">{row.proposal}</bdi></small></button></li>)}</ul>
    {page.next&&<button disabled={busy} onClick={()=>{if(discard())void store?.history({after:page.next,snapshot:page.snapshot});}}>{t(l,"Next proposals","다음 제안","المقترحات التالية")}</button>}</>}
   </section>
   <div className="authority-detail"><button className="authority-back" onClick={()=>{if(!discard())return;setEntering(false);setPane("list");store?.clearDetail();focusTarget.current="selection";setFocusRequest(v=>v+1);}}>{t(l,"Back to proposals","제안 목록으로","العودة إلى المقترحات")}</button><Feedback state={state.detail} locale={l}/>
    {detail?<Detail area={area} key={`${detail.row.proposal}:${detail.revision}:${detail.row.state}`} detail={detail} actor={session.user} locale={l} busy={busy} onSend={send} onEditing={setEditing} onDraft={setDraft}/>:state.detail.phase==="idle"&&<p>{t(l,"Select a proposal to inspect its evidence and recorded decision.","제안을 선택해 근거와 기록된 결정을 확인하세요.","اختر مقترحًا لعرض أدلته والقرار المسجل.")}</p>}
   </div>
  </div>}
  {!!attempts.entries.length&&<section><h3>{t(l,"Submissions in this session","이 세션의 제출 이력","عمليات الإرسال في هذه الجلسة")}</h3><ul className="authority-attempts">{attempts.entries.map(entry=><li key={entry.intent.attempt.key} data-authority-attempt={entry.phase}><strong>{label(l,entry.intent.action.startsWith("propose_")?entry.intent.action==="propose_console"?"console_ceiling":"human_grant":entry.intent.action)}</strong> · <bdi dir="ltr">{entry.intent.target}</bdi><p>{label(l,entry.phase)}</p>{entry.phase==="unknown"&&attempts.store?.settled(entry)&&<p>{t(l,"A later recovery recorded this decision. The earlier submission remains in the history.","후속 복구로 이 결정이 기록되었습니다. 이전 제출은 이력에 남아 있습니다.","سجلت تسوية لاحقة هذا القرار. يبقى الإرسال السابق في السجل.")}</p>}{entry.phase==="unknown"&&!attempts.store?.settled(entry)&&<><p>{entry.intent.action.startsWith("propose_")?t(l,"The proposal may have been recorded. Retry the same submission or inspect this proposal. Do not create a replacement proposal while the outcome is unknown.","제안이 기록되었을 수 있습니다. 동일한 제출을 다시 시도하거나 이 제안을 조회하세요. 결과가 미확정인 동안 대체 제안을 만들지 마세요.","ربما سُجل المقترح. أعد محاولة الإرسال نفسه أو اعرض هذا المقترح. لا تنشئ مقترحًا بديلًا ما دامت النتيجة غير معروفة."):t(l,"The earlier submission may have taken effect. Retry the same submission or inspect the proposal before recovery.","이전 제출이 처리되었을 수 있습니다. 동일한 제출을 다시 시도하거나 제안을 조회한 뒤 복구하세요.","قد يكون الإرسال السابق قد نُفذ. أعد محاولة الإرسال نفسه أو اعرض المقترح قبل التسوية.")}</p><div className="authority-actions"><button disabled={busy} onClick={()=>{if(!discard())return;setEntering(false);if(store&&attempts.store)void store.followMutation(entry.intent.target,attempts.store.retry(entry.intent.attempt.key));}}>{t(l,"Retry the same submission","동일한 제출 다시 시도","إعادة محاولة الإرسال نفسه")}</button><button disabled={busy} onClick={()=>{if(!discard())return;setEntering(false);setPane("detail");void store?.inspect(entry.intent.target);}}>{t(l,"Inspect proposal","제안 조회","عرض المقترح")}</button></div></>}{entry.error&&<details><summary>{t(l,"Technical reference","기술 참조","المرجع التقني")}</summary>{entry.error}</details>}</li>)}</ul></section>}
 </div>;
}
