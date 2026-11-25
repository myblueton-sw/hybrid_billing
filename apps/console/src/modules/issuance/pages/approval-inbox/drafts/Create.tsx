import { useEffect,useId,useState } from "react";
import type { Locale } from "../../../../../shared/types";
import { t } from "../copy";
import { rationaleValid } from "../documents/packet";
import type { DraftBinding } from "./use-drafts";
import { DraftError,fixedMoney } from "./contract";
import { DraftFeedback } from "./display";
import { submissionIntent } from "./intents";
export function CreateDraft({locale:l,binding,request,digest,onDirty,onOpen}:{locale:Locale;binding:DraftBinding;request:string;digest:string;onDirty:(dirty:boolean)=>void;onOpen:(draft:string)=>void}) {
 const {store,state,session}=binding,current=state.context.value,c=current?.final_request===request&&current.final_digest===digest?current:undefined;
 const [language,setLanguage]=useState(""),[rationale,setRationale]=useState(""),[error,setError]=useState<string>();const input=useId();
 useEffect(()=>{onDirty(!!language||!!rationale);return()=>onDirty(false);},[language,rationale,onDirty]);

 async function submit(){
  if(!c||!store||!session||binding.unresolved)return;
  try{const intent=submissionIntent(session,c,language,rationale);await store.execute(intent);if(store.snapshot().entries.find(e=>e.intent.attempt.key===intent.attempt.key)?.phase==="recorded"){setLanguage("");setRationale("");}}catch(e){setError(e instanceof DraftError?e.code:"service_unavailable");}
 }
 return <section className="b09-drafts b09-draft-actions b09-draft-create"><h3>{t(l,"Create an issuance draft","발행 초안 생성","إنشاء مسودة إصدار")}</h3><p>{t(l,"Uses the selected approved final amount and its applicable configuration. Amounts and source references are carried over automatically.","선택한 승인 완료 최종 금액과 적용 설정을 사용합니다. 금액과 원천 참조는 자동으로 이어집니다.","يستخدم المبلغ النهائي المعتمد المحدد والإعدادات المنطبقة عليه. تُنقل المبالغ ومراجع المصدر تلقائيًا.")}</p>
  <button disabled={!store||state.context.loading||binding.unresolved} onClick={()=>void store?.context()}>{t(l,"Check draft prerequisites","초안 생성 요건 확인","التحقق من متطلبات المسودة")}</button>
  <DraftFeedback locale={l} loading={state.context.loading} error={binding.error??state.context.error??error}/>
  {c&&<><p className="b09-draft-readiness" role="status">{c.submit_action==="ready"?t(l,"Ready to create a draft","초안 생성 가능","جاهز لإنشاء مسودة"):t(l,"Draft prerequisites need attention","초안 생성 요건 확인 필요","متطلبات المسودة تحتاج إلى تحقق")}</p><dl className="b09-create-facts"><dt>{t(l,"Final request ID","최종 금액 요청 ID","معرّف طلب المبلغ النهائي")}</dt><dd><bdi dir="ltr">{c.final_request}</bdi></dd><dt>{t(l,"Approved source amount","승인된 원천 금액","مبلغ المصدر المعتمد")}</dt><dd><bdi dir="ltr">{c.currency} {fixedMoney(c.total,c.scale)}</bdi></dd></dl>
   {c.existing_draft?<><p>{t(l,"A draft already exists for this billing target.","이 청구 대상의 초안이 이미 존재합니다.","توجد مسودة لهدف الفوترة هذا بالفعل.")}</p><button onClick={()=>onOpen(c.existing_draft!)}>{t(l,"Open existing draft","기존 초안 열기","فتح المسودة الحالية")}</button></>:c.submit_action==="ready"?<>
    <label htmlFor={input+"-language"}>{t(l,"Document language","문서 언어","لغة المستند")}</label><select id={input+"-language"} value={language} disabled={binding.unresolved} onChange={e=>setLanguage(e.target.value)}><option value="">{t(l,"Choose a language","언어 선택","اختر لغة")}</option>{c.languages.map(code=><option key={code} value={code}>{code==="en"?"English":code==="ko"?"한국어":"العربية"}</option>)}</select>
    <label htmlFor={input}>{t(l,"Reason for this action","작업 사유","سبب الإجراء")}</label><textarea dir="auto" id={input} value={rationale} disabled={binding.unresolved} onChange={e=>setRationale(e.target.value)}/>
    <p>{t(l,"Reason limit: 2,000 UTF-8 bytes.","사유는 UTF-8 기준 2,000바이트 이내로 입력하세요.","حد السبب: 2000 بايت بترميز UTF-8.")}</p><p>{t(l,"Creates a draft for independent review. It does not issue a document.","독립 검토를 위한 초안을 생성합니다. 문서를 발행하지 않습니다.","ينشئ مسودة لمراجعة مستقلة. ولا يُصدر مستندًا.")}</p><button disabled={binding.unresolved||!c.languages.includes(language)||!rationaleValid(rationale)} onClick={()=>void submit()}>{t(l,"Create draft","초안 생성","إنشاء المسودة")}</button>
   </>:<><p>{t(l,"Draft creation is unavailable. Verify the source, permissions and governed setup before continuing.","초안을 생성할 수 없습니다. 원천·권한·승인된 설정을 확인하세요.","إنشاء المسودة غير متاح. تحقق من المصدر والصلاحيات والإعدادات المعتمدة قبل المتابعة.")}</p><details><summary>{t(l,"Technical reference","기술 참조","المرجع التقني")}</summary><bdi dir="ltr">{c.submit_action}</bdi></details></>}
   {c.step_up==="reauthentication_required"&&binding.reauthenticate&&<button onClick={()=>void binding.reauthenticate?.().catch(()=>setError("reauthentication_required"))}>{t(l,"Confirm password","비밀번호 확인","تأكيد كلمة المرور")}</button>}
  </>}
 </section>;
}
