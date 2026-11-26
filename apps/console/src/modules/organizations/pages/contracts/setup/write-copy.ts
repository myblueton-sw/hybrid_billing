import type { Locale } from "../../../../../shared/types";
const words={
 title:["Propose or confirm settings","설정 제안·확인","اقتراح الإعدادات أو تأكيدها"],
 new:["New proposal","새 제안","اقتراح جديد"],
 preset:["Use the synthetic ERP sample above","위의 합성 ERP 샘플 사용","استخدام عينة ERP الاصطناعية أعلاه"],
 languages:["Document languages — select at least one","문서 언어 · 하나 이상 선택","لغات المستند — اختر لغة واحدة على الأقل"],
 check:["Check permission and review","권한 확인 후 검토","التحقق من الصلاحية والمراجعة"],
 unlisted:["The settings chosen under “Start from” are not on the loaded page of recorded issuance settings, so permission cannot be checked from here. Use First page or Next page to load the page that lists them, or choose another option.","작업 대상으로 고른 설정이 현재 불러온 발행 설정 이력 페이지에 없어 여기에서 권한 확인과 검토를 진행할 수 없습니다. 첫 페이지 또는 다음 페이지로 해당 설정이 있는 페이지를 불러오거나 작업 대상을 다시 고르세요.","الإعدادات المختارة في «نقطة البدء» ليست ضمن الصفحة المحمّلة من إعدادات الإصدار المسجلة، لذا لا يمكن التحقق من الصلاحية من هنا. استخدم «الصفحة الأولى» أو «الصفحة التالية» لتحميل الصفحة التي تتضمنها، أو اختر خيارًا آخر."],
 scopeMissing:["Not shown because the setup scope could not be loaded. Reload the setup scope above.","설정 기준을 조회하지 못해 표시하지 않습니다. 위의 설정 기준을 새로 조회하세요.","لا تُعرض لأن نطاق الإعداد تعذر تحميله. أعد تحميل نطاق الإعداد أعلاه."],
 review:["Review the exact request","제출할 요청 검토","مراجعة الطلب المحدد"],
 propose:["Submit proposal","제안 제출","إرسال الاقتراح"],
 confirm:["Confirm settings","설정 확인","تأكيد الإعدادات"],
 recover:["Recover recorded attempt","기록된 시도 복구","استعادة المحاولة المسجلة"],
 edit:["Back to inputs","입력으로 돌아가기","العودة إلى المدخلات"],
 outcomes:["Session request results","현재 세션 요청 결과","نتائج طلبات الجلسة"],
 pending:["Request in progress","요청 처리 중","الطلب قيد التنفيذ"],
 unknown:["Outcome unknown. Retry this request unchanged.","결과 미확인. 동일한 요청으로 재시도하세요.","النتيجة غير معروفة. أعد محاولة الطلب دون تغيير."],
 refused:["Request refused","요청 거절","رُفض الطلب"],
 recorded:["Recorded. Current issuance eligibility was not checked.","처리 결과가 기록되었습니다. 현재 발행 가능 여부는 확인하지 않았습니다.","سُجلت النتيجة. لم تُفحص أهلية الإصدار الحالية."],
 noRecordPhase:["The retry was refused.","재시도가 거절되었습니다.","رُفضت إعادة المحاولة."],
 noRecord:["The service has no record of this request. Records changed after this request was reviewed. Check the current records, then review and send it again if it is still needed.","서비스에 이 요청의 기록이 없습니다. 요청을 검토한 뒤 기록이 변경되었습니다. 현재 기록을 확인한 다음, 필요하면 다시 검토해 전송하세요.","لا يوجد لدى الخدمة سجل لهذا الطلب. تغيّرت السجلات بعد مراجعته. راجع السجلات الحالية، ثم راجع الطلب وأرسله من جديد إذا لزم."],
 earlierSession:["Sent before you signed in again. Its outcome is not known here, and this request can no longer be retried from this page. Check “Start from” above: if the service lists these settings with an unknown outcome, select them there to recover the recorded attempt. This entry stays until the page is reloaded.","다시 로그인하기 전에 전송한 요청입니다. 이 페이지에서는 결과를 확인할 수 없으며, 이 요청은 더 이상 재시도할 수 없습니다. 위의 ‘작업 대상’을 확인하세요. 서비스가 이 설정을 처리 결과 미확인으로 표시하면 그곳에서 선택해 기록된 시도를 복구할 수 있습니다. 이 항목은 페이지를 다시 불러올 때까지 남아 있습니다.","أُرسل هذا الطلب قبل أن تسجّل الدخول مجددًا. نتيجته غير معروفة في هذه الصفحة، ولم يعد ممكنًا إعادة محاولة هذا الطلب منها. راجع «نقطة البدء» أعلاه: إذا عرضت الخدمة هذه الإعدادات بنتيجة غير معروفة، فاخترها هناك لاستعادة المحاولة المسجلة. يبقى هذا البند حتى إعادة تحميل الصفحة."],
 reconciled:["Resolved by matching recovery","일치하는 복구 결과로 확인됨","حُسمت باستعادة مطابقة"],
 retry:["Retry exact request","동일 요청 재시도","إعادة محاولة الطلب نفسه"],
 auth:["Verify identity again","본인 확인 다시 하기","التحقق من الهوية مجددًا"],
 note:["Inputs are retained for this contract and date in this session. Confirmation requires a different authorized person. This does not issue an invoice.","입력값은 이 세션의 계약·기준일별로 유지됩니다. 확인에는 권한이 있는 다른 사람이 필요합니다. 청구서를 발행하는 작업은 아닙니다.","تُحفظ المدخلات لكل عقد وتاريخ في هذه الجلسة. يتطلب التأكيد شخصًا آخر مخولًا. لا تُصدر هذه العملية فاتورة."],
 contract:["Contract","계약","العقد"],target:["Configuration","설정","الإعداد"],
 action:["Action","작업","الإجراء"],
 start:["Start from","작업 대상","نقطة البدء"],
 proposal:["Proposal","제안","الاقتراح"],confirmation:["Confirmation","확인","التأكيد"],
 proposalReason:["Proposal reason","제안 사유","سبب الاقتراح"],confirmationReason:["Confirmation reason","확인 사유","سبب التأكيد"],
 profile:["Billing profile","청구 프로필","ملف الفوترة"],recordedProfile:["Recorded billing profile","기록된 청구 프로필","ملف الفوترة المسجل"],
 errorCode:["Error code","오류 코드","رمز الخطأ"],
 originalReason:["Original proposal reason","원래 제안 사유","سبب الاقتراح الأصلي"],
} as const;
export const writeText=(l:Locale,key:keyof typeof words)=>words[key][l==="ko"?1:l==="ar"?2:0];

const errors={
 invalid_setup_request:["Select the sample and at least one language for a new proposal. Enter a reason for proposal or confirmation.","새 제안은 샘플과 문서 언어를 하나 이상 선택하세요. 제안 또는 확인 사유를 입력하세요.","للاقتراح الجديد، اختر العينة ولغة واحدة على الأقل. أدخل سبب الاقتراح أو التأكيد."],
 permission_denied:["You are not permitted to perform this action for this contract.","이 계약에서 해당 작업을 수행할 권한이 없습니다.","ليس لديك إذن لتنفيذ هذا الإجراء لهذا العقد."],
 self_approval_denied:["A different authorized person must confirm this proposal.","권한이 있는 다른 사람이 이 제안을 확인해야 합니다.","يجب أن يؤكد هذا الاقتراح شخص آخر مخول."],
 evidence_stale:["The contract or settings changed. Check again.","계약 또는 설정이 변경되었습니다. 다시 확인하세요.","تغير العقد أو الإعدادات. أعد التحقق."],
 unresolved_operation:["Resolve the request with an unknown outcome below first.","아래의 결과 미확인 요청을 먼저 해결하세요.","احسم أولًا الطلب ذا النتيجة غير المعروفة أدناه."],
 reauthentication_required:["Verify your identity again, then check again.","본인 확인을 다시 진행한 후 다시 확인하세요.","تحقق من هويتك مجددًا، ثم أعد التحقق."],
 attempt_capacity:["This page has reached its limit of 100 requests in this session, so it sent nothing. Settle any request with an unknown outcome first, then reload the page to continue.","이 페이지는 이 세션의 요청 한도인 100건에 도달해 아무것도 전송하지 않았습니다. 결과 미확인 요청이 있으면 먼저 해결한 다음, 계속하려면 페이지를 다시 불러오세요.","بلغت هذه الصفحة حدّها البالغ 100 طلب في هذه الجلسة، فلم ترسل شيئًا. احسم أولًا أي طلب نتيجته غير معروفة، ثم أعد تحميل الصفحة للمتابعة."],
 unavailable:["The request could not be completed. Try checking again.","요청을 완료하지 못했습니다. 다시 확인해 주세요.","تعذر إكمال الطلب. حاول التحقق مجددًا."],
} as const;
export const needsAuthentication=(code:string)=>code==="reauthentication_required"||code==="authentication_policy_unsatisfied";
export function writeError(l:Locale,code:string){
 const key=needsAuthentication(code)?"reauthentication_required":code==="retry_unavailable"?"unresolved_operation":code==="inspection_required"?"evidence_stale":Object.hasOwn(errors,code)?code as keyof typeof errors:"unavailable";
 return errors[key][l==="ko"?1:l==="ar"?2:0];
}
