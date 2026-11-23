import { useEffect,useId,useState } from "react";
import type { Locale } from "../../../../../shared/types";
import { t } from "../copy";
import { rationaleValid } from "../documents/packet";
import { DraftError } from "./contract";
import { DraftFeedback } from "./display";
import { approvalIntent,preparationIntent,recoveryIntent } from "./intents";
import type { Recovery } from "./inspection";
import type { DraftBinding } from "./use-drafts";
export function DraftActions({locale:l,binding,onDirty}:{locale:Locale;binding:DraftBinding;onDirty:(dirty:boolean)=>void}) {
 const {state,store,session}=binding,draft=state.inspection.value,context=state.context.value;
 const [rationale,setRationale]=useState(""),[term,setTerm]=useState(""),[error,setError]=useState<string>(),[pages,setPages]=useState<string[]>([""]);
 const input=useId();
 useEffect(()=>{setTerm("");},[context?.snapshot]);
 useEffect(()=>{onDirty(!!rationale||!!term);return()=>onDirty(false);},[rationale,term,onDirty]);
 if(!draft||!session||!store)return null;
 const pending=state.entries.some(e=>e.phase==="pending"),disabled=binding.unresolved||!rationaleValid(rationale);
 const coherent=context?.revision===draft.revision;
 const readyPrepare=coherent&&context?.prepare_action==="ready",readyApprove=coherent&&(context?.approve_action==="ready"||context?.approve_action==="recheck_required");
 const allowedRecovery=draft.recoveries.filter(r=>r.permission==="allowed"&&r.original_actor===session.user&&(r.state!=="prepared"||state.entries.some(e=>e.phase==="unknown"&&e.intent.action===r.operation&&e.intent.target===r.business_id)));
 async function run(kind:"prepare"|"approve",recovery?:Recovery) {
  if(!draft||!session||!store||binding.unresolved&&!recovery)return;
  try {
   const intent=recovery?recoveryIntent(session,draft,recovery,rationale):kind==="prepare"&&context?preparationIntent(session,context,draft,term,rationale):context?approvalIntent(session,context,draft,rationale):undefined;
   if(!intent)throw new DraftError("draft_action_unavailable");
   if(recovery)await store.recover(intent);else await store.execute(intent);
   const result=store.snapshot().entries.find(e=>e.intent.attempt.key===intent.attempt.key);
   if(result?.phase==="recorded"&&store.snapshot().selected===draft.draft)await store.open(draft.draft);
  }catch(e){setError(e instanceof DraftError?e.code:"service_unavailable");}
 }
 function terms(after:string,back=false){if(!context)return;setTerm("");setPages(back?pages.slice(0,-1):[...pages,after]);void store?.context(after,after?context.snapshot:"");}
 return <section className="b09-draft-actions"><h4>{t(l,"Review and approval","검토 및 승인","المراجعة والاعتماد")}</h4>
  <button disabled={binding.unresolved||state.context.loading} onClick={()=>{if((rationale||term)&&!window.confirm(t(l,"Discard the unsaved action form and refresh the review?","저장하지 않은 입력을 버리고 검토를 새로 조회할까요?","هل تريد تجاهل البيانات غير المحفوظة وتحديث المراجعة؟")))return;setTerm("");setPages([""]);void store.open(draft.draft);}}>{t(l,"Refresh action eligibility","처리 가능 여부 새로 확인","تحديث أهلية الإجراء")}</button>
  {draft.origin==="legacy"?<p>{t(l,"This legacy draft is available for inspection only.","이 기존 방식 초안은 조회만 가능합니다.","هذه المسودة القديمة متاحة للعرض فقط.")}</p>:<>
   {!readyPrepare&&!readyApprove&&!allowedRecovery.length&&<p>{t(l,"No action is currently available to this account. Check the latest eligibility details or ask the responsible operator to verify the prerequisites.","현재 계정에서 진행할 수 있는 작업이 없습니다. 최신 처리 요건을 확인하거나 담당 운영자에게 확인을 요청하세요.","لا يتوفر إجراء لهذا الحساب حاليًا. تحقق من أحدث تفاصيل الأهلية أو اطلب من المشغّل المسؤول التحقق من المتطلبات.")}</p>}
   {context&&!coherent&&<p>{t(l,"The inspection and current eligibility use different revisions. Refresh the review before acting.","조회한 검토와 현재 처리 요건의 리비전이 다릅니다. 검토를 새로 조회한 뒤 진행하세요.","تستخدم المراجعة المعروضة والأهلية الحالية مراجعتين مختلفتين. حدّث المراجعة قبل المتابعة.")}</p>}
   {context&&<p className="b09-draft-readiness" role="status">{readyPrepare?t(l,"Ready to save a review","검토 저장 가능","جاهز لحفظ المراجعة"):readyApprove?t(l,"Ready for an approval attempt","승인 시도 가능","جاهز لمحاولة الاعتماد"):t(l,"Action prerequisites need attention","처리 요건 확인 필요","متطلبات الإجراء تحتاج إلى تحقق")}</p>}
   {context&&<details><summary>{t(l,"Eligibility details","처리 요건 상세","تفاصيل الأهلية")}</summary><dl><dt>{t(l,"Save review","검토 저장","حفظ المراجعة")}</dt><dd><bdi dir="ltr">{context.prepare_action}</bdi></dd><dt>{t(l,"Approve","승인","اعتماد")}</dt><dd><bdi dir="ltr">{context.approve_action}</bdi></dd></dl></details>}
   {context?.step_up==="reauthentication_required"&&binding.reauthenticate&&<button onClick={()=>void binding.reauthenticate?.().catch(()=>setError("reauthentication_required"))}>{t(l,"Confirm password","비밀번호 확인","تأكيد كلمة المرور")}</button>}
   {readyPrepare&&context&&<fieldset disabled={binding.unresolved}><legend>{t(l,"Choose payment terms","납부 조건 선택","اختيار شروط الدفع")}</legend><p>{t(l,"Saving this review reserves the approval decision for your account. Only your account can complete this saved approval attempt. It does not approve the draft.","검토를 저장하면 이 계정이 승인 결정을 담당하게 됩니다. 저장된 승인 시도는 이 계정만 완료할 수 있습니다. 초안을 승인하는 동작은 아닙니다.","حفظ هذه المراجعة يخصّص قرار الاعتماد لحسابك. لا يمكن إكمال محاولة الاعتماد المحفوظة هذه إلا من حسابك. ولا يعتمد المسودة.")}</p>
    {context.rows.map(row=><label key={row.binding}><input type="radio" name={input+"-term"} value={row.binding} checked={term===row.binding} onChange={()=>setTerm(row.binding)}/><span><bdi dir="ltr">{row.net_days}</bdi> {t(l,"calendar days","달력일","يوم تقويمي")} · <bdi dir="ltr">{row.zone}</bdi><br/>{t(l,"Effective from","적용 시작일","ساري من")}: <bdi dir="ltr">{row.from}</bdi>{row.to&&<> — <bdi dir="ltr">{row.to}</bdi></>}<br/>{t(l,"Binding reference","조건 연결 참조","مرجع ربط الشرط")}: <bdi dir="ltr">{row.binding}</bdi></span></label>)}
    <nav aria-label={t(l,"Payment term pages","납부 조건 페이지","صفحات شروط الدفع")}><button type="button" disabled={pages.length<2} onClick={()=>terms(pages[pages.length-2],true)}>{t(l,"Previous","이전","السابق")}</button><button type="button" disabled={!context.next} onClick={()=>terms(context.next)}>{t(l,"Next","다음","التالي")}</button></nav>
   </fieldset>}
   {(readyPrepare||readyApprove||allowedRecovery.length>0)&&<><label htmlFor={input}>{t(l,"Reason for this action","작업 사유","سبب الإجراء")}</label><textarea dir="auto" id={input} value={rationale} disabled={pending} onChange={e=>setRationale(e.target.value)} aria-describedby={input+"-help"}/><p>{t(l,"Reason limit: 2,000 UTF-8 bytes.","사유는 UTF-8 기준 2,000바이트 이내로 입력하세요.","حد السبب: 2000 بايت بترميز UTF-8.")}</p><p id={input+"-help"}>{t(l,"Enter a reason before saving, approving or resuming the original operation.","검토 저장·승인·기존 작업 재처리 전에 사유를 입력하세요.","أدخل سببًا قبل الحفظ أو الاعتماد أو استئناف العملية الأصلية.")}</p></>}
   {readyPrepare&&<button disabled={disabled||!term} onClick={()=>void run("prepare")}>{t(l,"Save review","검토 저장","حفظ المراجعة")}</button>}
   {readyApprove&&<><p>{context?.approve_action==="recheck_required"?t(l,"The saved facts have changed. This attempt checks the saved review again. If they still differ, approval is refused and a new review is required.","저장된 검토 사실과 현재 사실이 다릅니다. 다시 확인해도 차이가 있으면 승인 시도가 거절되며 새 검토가 필요합니다.","تختلف الوقائع الحالية عن المراجعة المحفوظة. إذا استمر الاختلاف عند إعادة التحقق، تُرفض محاولة الاعتماد وتلزم مراجعة جديدة."):t(l,"Approve the draft against the saved review evidence. Document issuance is a separate step.","저장된 검토 근거에 따라 초안을 승인합니다. 문서 발행은 별도 단계입니다.","اعتمد المسودة وفق أدلة المراجعة المحفوظة. إصدار المستند خطوة منفصلة.")}</p><button disabled={disabled} onClick={()=>void run("approve")}>{t(l,"Approve draft","초안 승인","اعتماد المسودة")}</button></>}
   {allowedRecovery.map(r=><div key={r.operation}><p>{t(l,"Resume the original operation with its saved arguments. Only its original actor can do this.","저장된 원래 인자로 기존 작업을 이어갑니다. 처음 작업한 계정만 실행할 수 있습니다.","استأنف العملية الأصلية بمدخلاتها المحفوظة. يمكن لمن بدأها فقط تنفيذ ذلك.")}</p><button disabled={pending||!rationaleValid(rationale)} onClick={()=>void run("approve",r)}>{r.operation==="submit"?t(l,"Resume submission","제출 재처리","استئناف الإرسال"):r.operation==="prepare_approval"?t(l,"Resume saved review","검토 저장 재처리","استئناف حفظ المراجعة"):t(l,"Resume approval","승인 재처리","استئناف الاعتماد")}</button></div>)}
  </>}
  <DraftFeedback locale={l} error={error}/>
 </section>;
}
