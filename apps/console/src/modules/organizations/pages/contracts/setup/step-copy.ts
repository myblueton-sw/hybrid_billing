import type { Locale } from "../../../../../shared/types";

const words = {
 actions: ["Steps", "단계", "الخطوات"],
 permissionNote: ["This screen cannot tell which permissions you hold. A step you may not perform is refused when it is sent, and a refused request records nothing.", "이 화면은 보유한 권한을 알 수 없습니다. 수행할 수 없는 단계는 전송 시 거절되며, 거절된 요청은 아무것도 기록하지 않습니다.", "لا تستطيع هذه الشاشة معرفة الصلاحيات التي تملكها. تُرفض عند الإرسال الخطوة التي لا يحق لك تنفيذها، والطلب المرفوض لا يسجّل شيئاً."],
 review: ["Review before sending", "전송 전 검토", "المراجعة قبل الإرسال"],
 reviewTitle: ["Review the request", "요청 검토", "مراجعة الطلب"],
 reviewNote: ["Nothing is recorded until you send the request. Permission is checked when it is sent.", "요청을 전송하기 전에는 아무것도 기록되지 않습니다. 권한은 전송할 때 확인합니다.", "لا يُسجَّل شيء قبل إرسال الطلب. تُفحص الصلاحية عند الإرسال."],
 back: ["Back to inputs", "입력으로 돌아가기", "العودة إلى المدخلات"],
 cancel: ["Cancel", "취소", "إلغاء"],
 step: ["Step", "단계", "الخطوة"],
 code: ["Code", "코드", "الرمز"],
 results: ["Requests sent in this session", "이 세션에서 전송한 요청", "الطلبات المرسلة في هذه الجلسة"],
 phase_pending: ["Request in progress", "요청 처리 중", "الطلب قيد التنفيذ"],
 phase_unknown: ["Outcome unknown. Retry this request unchanged. If the same result keeps coming back, recover the recorded attempt: the same inputs are sent as a new request.", "결과 미확인. 동일한 요청으로 재시도하세요. 같은 결과가 계속 돌아오면 기록된 시도를 복구하세요. 같은 입력을 새 요청으로 전송합니다.", "النتيجة غير معروفة. أعد محاولة الطلب دون تغيير. إذا تكررت النتيجة نفسها فاستعد المحاولة المسجلة: تُرسل المدخلات نفسها كطلب جديد."],
 phase_refused: ["Refused. This request recorded nothing.", "거절되었습니다. 이 요청은 아무것도 기록하지 않았습니다.", "رُفض. لم يسجّل هذا الطلب شيئاً."],
 phase_recorded: ["Recorded. Current issuance eligibility was not checked.", "기록되었습니다. 현재 발행 가능 여부는 확인하지 않았습니다.", "تم التسجيل. لم تُفحص أهلية الإصدار الحالية."],
 phase_noRecord: ["The retry was refused.", "재시도가 거절되었습니다.", "رُفضت إعادة المحاولة."],
 noRecord: ["The service has no record of this request. Records changed after this request was reviewed. Check the current records, then review and send it again if it is still needed.", "서비스에 이 요청의 기록이 없습니다. 요청을 검토한 뒤 기록이 변경되었습니다. 현재 기록을 확인한 다음, 필요하면 다시 검토해 전송하세요.", "لا يوجد لدى الخدمة سجل لهذا الطلب. تغيّرت السجلات بعد مراجعته. راجع السجلات الحالية، ثم راجع الطلب وأرسله من جديد إذا لزم."],
 earlierSession: ["Sent before you signed in again. Its outcome is not known here, and it can no longer be retried or recovered from this page. Check the current records. This entry, and the prompt when you leave the page, stay until the page is reloaded.", "다시 로그인하기 전에 전송한 요청입니다. 이 페이지에서는 결과를 확인할 수 없으며, 더 이상 재시도하거나 복구할 수 없습니다. 현재 기록을 확인하세요. 이 항목과 페이지를 떠날 때 나타나는 확인 메시지는 페이지를 다시 불러올 때까지 남아 있습니다.", "أُرسل هذا الطلب قبل أن تسجّل الدخول مجددًا. نتيجته غير معروفة في هذه الصفحة، ولم يعد ممكنًا إعادة محاولته أو استعادته منها. راجع السجلات الحالية. يبقى هذا البند، وكذلك التنبيه عند مغادرة الصفحة، حتى إعادة تحميل الصفحة."],
 phase_reconciled: ["Settled by a matching request of this session", "이 세션의 일치하는 요청으로 확인됨", "حُسمت بطلب مطابق في هذه الجلسة"],
 retry: ["Retry exact request", "동일 요청 재시도", "إعادة محاولة الطلب نفسه"],
 recover: ["Recover recorded attempt", "기록된 시도 복구", "استعادة المحاولة المسجلة"],
 auth: ["Verify identity again", "본인 확인 다시 하기", "التحقق من الهوية مجددًا"],
 errUnresolved: ["Settle the request with an unknown outcome listed under the requests of this session first.", "이 세션에서 전송한 요청 목록의 결과 미확인 요청을 먼저 해결하세요.", "احسم أولاً الطلب ذا النتيجة غير المعروفة المدرج ضمن طلبات هذه الجلسة."],
 errAuth: ["Verify your identity again, then send the request again.", "본인 확인을 다시 진행한 뒤 요청을 다시 전송하세요.", "تحقق من هويتك مجدداً، ثم أرسل الطلب من جديد."],
 errOther: ["The request could not be completed.", "요청을 완료하지 못했습니다.", "تعذر إكمال الطلب."],
 blocked: ["Steps are unavailable until the request with an unknown outcome, listed below, is settled.", "아래에 표시된 결과 미확인 요청을 해결할 때까지 단계를 사용할 수 없습니다.", "الخطوات غير متاحة إلى أن يُحسم الطلب ذو النتيجة غير المعروفة المدرج أدناه."],
 sessionOnly: ["A request left with an unknown outcome can be retried or recovered only from the session that sent it. Settle it before closing or reloading this page.", "결과 미확인으로 남은 요청은 전송한 세션에서만 재시도하거나 복구할 수 있습니다. 이 페이지를 닫거나 다시 불러오기 전에 해결하세요.", "الطلب الذي بقيت نتيجته غير معروفة لا يمكن إعادة محاولته أو استعادته إلا من الجلسة التي أرسلته. احسمه قبل إغلاق هذه الصفحة أو إعادة تحميلها."],
 phase_unknown_retry: ["Outcome unknown. Retry this request unchanged.", "결과 미확인. 동일한 요청으로 재시도하세요.", "النتيجة غير معروفة. أعد محاولة الطلب دون تغيير."],
 sentAt: ["Sent", "전송 시각", "وقت الإرسال"],
 forContract: ["Contract", "계약", "العقد"],
 settings: ["Issuance settings", "발행 설정", "إعدادات الإصدار"],
 recoverRead: ["Recovery first reads the current state of these settings, which needs the permission to read them. That read was refused or failed. The unchanged retry does not need it.", "복구는 먼저 이 설정의 현재 상태를 조회하며, 조회 권한이 필요합니다. 그 조회가 거절되었거나 실패했습니다. 동일 요청 재시도에는 필요하지 않습니다.", "تقرأ الاستعادة أولاً الحالة الحالية لهذه الإعدادات، وهذا يتطلب صلاحية قراءتها. رُفضت تلك القراءة أو تعذرت. إعادة المحاولة دون تغيير لا تتطلبها."],
 capacity: ["This page holds too many request stores. Reload the page to send requests.", "이 페이지의 요청 저장소가 한도에 도달했습니다. 요청을 전송하려면 페이지를 다시 불러오세요.", "بلغت هذه الصفحة حد مخازن الطلبات. أعد تحميل الصفحة لإرسال الطلبات."],
 attemptCapacity: ["This page has reached its limit of 100 requests in this session, so it sent nothing. Settle any request with an unknown outcome first, then reload the page to continue.", "이 페이지는 이 세션의 요청 한도인 100건에 도달해 아무것도 전송하지 않았습니다. 결과 미확인 요청이 있으면 먼저 해결한 다음, 계속하려면 페이지를 다시 불러오세요.", "بلغت هذه الصفحة حدّها البالغ 100 طلب في هذه الجلسة، فلم ترسل شيئًا. احسم أولًا أي طلب نتيجته غير معروفة، ثم أعد تحميل الصفحة للمتابعة."],
 recordedState: ["Recorded state", "기록된 상태", "الحالة المسجَّلة"],
} as const;
export type StepWord = keyof typeof words;
export function stepText(l: Locale, key: StepWord): string { return words[key][l === "ko" ? 1 : l === "ar" ? 2 : 0]; }
export function stepError(l: Locale, code: string): string {
 const key: StepWord = code === "unresolved_operation" || code === "retry_unavailable" || code === "recovery_unavailable" ? "errUnresolved" : code === "reauthentication_required" || code === "authentication_policy_unsatisfied" ? "errAuth" : code === "recovery_read" ? "recoverRead" : code === "session_capacity" ? "capacity" : code === "attempt_capacity" ? "attemptCapacity" : "errOther";
 return stepText(l, key);
}
