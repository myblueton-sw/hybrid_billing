import type { Locale } from "../../../../../shared/types";
import { t } from "../copy";
import type { DraftBinding } from "./use-drafts";
import { draftStateLabel } from "./display";
export function DraftOutcomes({locale:l,binding,onInspect}:{locale:Locale;binding:DraftBinding;onInspect:(id:string)=>void}) {
 const {state,store}=binding;
 if(!state.entries.length)return null;
 const pending=state.entries.some(e=>e.phase==="pending");
 return <section className="b09-draft-outcomes" aria-label={t(l,"Draft operation results","초안 작업 결과","نتائج عمليات المسودة")}><h3>{t(l,"Draft operation results","초안 작업 결과","نتائج عمليات المسودة")}</h3><ul>{state.entries.map(e=><li key={e.intent.attempt.key}>
  <p>{t(l,"Draft ID","초안 ID","معرّف المسودة")}: <bdi dir="ltr">{e.intent.draft}</bdi></p>
  <p role="status">{e.phase==="pending"?t(l,"Submitting…","처리 중…","جارٍ الإرسال…"):e.phase==="unknown"?t(l,"Result unconfirmed. Retry the identical request before starting another operation.","처리 결과를 확인하지 못했습니다. 다른 작업 전에 동일한 요청을 재시도하세요.","النتيجة غير مؤكدة. أعد محاولة الطلب نفسه قبل بدء عملية أخرى."):e.phase==="reconciled"?t(l,"The original response remains unconfirmed. The retained business result was verified by recovery.","원래 응답은 미확인이지만, 재처리로 보관된 업무 결과를 확인했습니다.","تبقى الاستجابة الأصلية غير مؤكدة. تم التحقق من النتيجة المحفوظة عبر الاستئناف."):e.phase==="refused"?t(l,"Request refused without a new change. Refresh before trying again.","새 변경 없이 요청이 거절되었습니다. 새로 조회한 뒤 다시 시도하세요.","رُفض الطلب دون تغيير جديد. حدّث البيانات قبل المحاولة مجددًا."):draftStateLabel(l,e.receipt?.reviewState??e.receipt?.submissionState??"")}</p>
  {e.error&&<details><summary>{t(l,"Technical reference","기술 참조","المرجع التقني")}</summary><bdi dir="ltr">{e.error}</bdi>{e.operation&&<p><bdi dir="ltr">{e.operation}</bdi></p>}</details>}
  {e.phase==="unknown"&&<button disabled={pending} onClick={()=>void store?.retry(e.intent.attempt.key)}>{t(l,"Retry identical request","동일 요청 재시도","إعادة محاولة الطلب نفسه")}</button>}
  {e.phase!=="pending"&&<button onClick={()=>onInspect(e.intent.draft)}>{t(l,"Inspect retained draft","보관된 초안 조회","عرض المسودة المحفوظة")}</button>}
 </li>)}</ul></section>;
}
