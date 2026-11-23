import type { Locale } from "../../../../shared/types";
import { t } from "./copy";
const copy = {
 unknownRefusal: ["The earlier outcome remains unknown. Do not start a new submission. Retry only the same submission.", "이전 처리 결과는 여전히 미확인입니다. 새로 제출하지 말고 동일한 제출만 다시 시도하세요.", "لا تزال نتيجة المحاولة السابقة غير معروفة. لا تبدأ إرسالاً جديداً. أعد محاولة الإرسال نفسه فقط."],
 unknownReauth: ["Confirm your identity, then retry the same submission. The earlier outcome remains unknown.", "본인 확인 후 동일한 제출을 다시 시도하세요. 이전 처리 결과는 여전히 미확인입니다.", "أكّد هويتك، ثم أعد محاولة الإرسال نفسه. لا تزال نتيجة المحاولة السابقة غير معروفة."],
 readonlyRecord: ["This record is available for review only. Decision actions are not connected for this record. A recorded approval does not confirm issuance, delivery or current eligibility.", "이 기록은 조회만 가능합니다. 이 기록의 결정 작업은 연결되지 않았습니다. 승인 기록은 발행·전달·현재 결정 자격의 확인이 아닙니다.", "هذا السجل متاح للمراجعة فقط. إجراءات اتخاذ القرار غير متصلة لهذا السجل. لا يؤكد سجل الاعتماد الإصدار أو التسليم أو الأهلية الحالية لاتخاذ القرار."],
 checked: ["Decision conditions checked", "결정 조건 확인됨", "تم التحقق من شروط القرار"],
 evidence: ["Request evidence", "요청 근거", "أدلة الطلبات"],
 help: ["Read recorded evidence separately from current decision conditions. Supported charge requests can be approved or rejected after current checks. Handoff, delegation and posting remain unconnected. Synthetic environment records are always labelled.", "기록된 근거와 현재 결정 조건을 구분합니다. 지원되는 청구 승인 요청은 현재 조건을 확인한 뒤 승인·반려할 수 있습니다. 인계·위임·전기는 연결되지 않았습니다. 합성 환경의 기록은 항상 별도로 표시합니다.", "تُفصل الأدلة المسجّلة عن شروط القرار الحالية. يمكن اعتماد طلبات الرسوم المدعومة أو رفضها بعد التحقق من الشروط الحالية. الإحالة والتفويض والترحيل غير متصلة. تُميّز سجلات البيئة الاصطناعية دائماً."],
 title: ["Record a decision", "결정 기록", "تسجيل قرار"],
 check: ["Check decision conditions", "결정 조건 확인", "التحقق من شروط القرار"],
 checking: ["Checking current conditions…", "현재 조건 확인 중…", "جارٍ التحقق من الشروط الحالية…"],
 unchecked: ["Current decision conditions have not been checked.", "현재 결정 조건을 확인하지 않았습니다.", "لم يتم التحقق من شروط القرار الحالية."],
 approve: ["Approve", "승인", "اعتماد"], reject: ["Reject", "반려", "رفض"],
 confirmApprove: ["Confirm approval", "승인 확정", "تأكيد الاعتماد"], confirmReject: ["Confirm rejection", "반려 확정", "تأكيد الرفض"],
 rationale: ["Decision rationale", "결정 사유", "مبررات القرار"],
 rationaleHint: ["Required. Up to 2,000 UTF-8 bytes; the server also validates the limit.", "필수 입력입니다. UTF-8 기준 최대 2,000바이트이며 서버에서도 한도를 확인합니다.", "مطلوب. بحد أقصى 2000 بايت بترميز UTF-8؛ يتحقق الخادم أيضاً من الحد."],
 reason: ["Rejection reason", "반려 이유", "سبب الرفض"], chooseReason: ["Select a reason", "이유 선택", "اختر سبباً"],
 cancel: ["Cancel", "취소", "إلغاء"],
 boundary: ["This records a financial decision only. It does not issue, deliver, post, pay or change amounts in an external ERP.", "재무 결정만 기록합니다. 발행·전달·전기·결제나 외부 ERP 금액 변경은 실행하지 않습니다.", "يُسجّل قرارًا ماليًا فقط. ولا ينفّذ الإصدار أو التسليم أو الترحيل أو الدفع أو تغيير المبالغ في نظام ERP خارجي."],
 pending: ["Recording the decision…", "결정 기록 중…", "جارٍ تسجيل القرار…"],
 unknown: ["Outcome unknown", "처리 결과 미확인", "نتيجة العملية غير معروفة"],
 unknownHelp: ["The decision may have been recorded. Retrying uses the same submission; it does not create a new attempt.", "결정이 기록되었을 수 있습니다. 다시 시도하면 새 시도를 만들지 않고 동일한 제출을 재전송합니다.", "ربما سُجّل القرار. تُعيد المحاولة إرسال الطلب نفسه ولا تُنشئ محاولة جديدة."],
 retry: ["Retry the same submission", "동일한 제출 다시 시도", "إعادة محاولة الإرسال نفسه"],
 recorded: ["The decision was recorded.", "결정이 기록됐습니다.", "سُجّل القرار."],
 refused: ["This decision was not recorded.", "이번 결정은 기록되지 않았습니다.", "لم يُسجّل هذا القرار."],
 refresh: ["Reload the list", "목록 다시 조회", "إعادة تحميل القائمة"],
 refreshFailed: ["The decision was recorded, but the list could not be refreshed. Reload the list.", "결정은 기록됐지만 목록을 새로 불러오지 못했습니다. 목록을 다시 불러오세요.", "سُجّل القرار، لكن تعذّر تحديث القائمة. أعد تحميل القائمة."],
 reloadLimit: ["Local submission tracking is kept only in this page session. Reloading the document loses this local tracking; inspect recorded history before making another decision.", "이 페이지의 로컬 제출 추적 정보는 현재 세션에만 유지됩니다. 페이지를 다시 로드하면 이 로컬 정보가 사라지므로 다음 결정 전에 기록된 이력을 확인하세요.", "يُحتفظ بتتبّع الإرسال المحلي في جلسة هذه الصفحة فقط. تؤدي إعادة تحميل المستند إلى فقدان هذا التتبّع المحلي؛ راجع السجل المسجّل قبل اتخاذ قرار آخر."],
 validation: ["Check the required fields", "필수 입력 확인", "تحقق من الحقول المطلوبة"],
 reauth: ["Confirm your identity", "본인 확인", "تأكيد هويتك"],
 facts: ["Recorded request facts", "기록된 요청 정보", "بيانات الطلب المسجّلة"],
 checkedAt: ["Checked at", "확인 시각", "وقت التحقق"],
 technical: ["Technical reference", "기술 참조", "المرجع التقني"],
 eligible: ["Available at last check", "마지막 확인 시 가능", "متاح عند آخر تحقق"],
 permission_denied: ["Decision permission is not available.", "결정 권한이 없습니다.", "صلاحية اتخاذ القرار غير متاحة."],
 self_approval_denied: ["The original author and requester cannot decide this request.", "원 작성자와 요청자는 이 요청을 결정할 수 없습니다.", "لا يجوز للمؤلف الأصلي ومقدّم الطلب اتخاذ القرار بشأن هذا الطلب."],
 amount_authority_missing: ["Current amount, currency or date coverage is not sufficient for approval.", "현재 금액·통화·기간 한도가 승인 조건을 충족하지 않습니다.", "تغطية المبلغ أو العملة أو الفترة الحالية غير كافية للاعتماد."],
 hold_active: ["Approval is unavailable while the service is on hold.", "서비스 보류 중에는 승인할 수 없습니다.", "الاعتماد غير متاح أثناء تعليق الخدمة."],
 reauthentication_required: ["Confirm your identity, then check conditions again.", "본인 확인 후 조건을 다시 확인하세요.", "أكّد هويتك ثم تحقق من الشروط مجدداً."],
 prerequisite_failed: ["A required condition is not satisfied.", "필수 조건이 충족되지 않았습니다.", "لم يُستوفَ شرط مطلوب."],
 evidence_stale: ["The evidence has changed. Reload the list and check again.", "근거가 변경됐습니다. 목록을 다시 조회하고 확인하세요.", "تغيّرت الأدلة. أعد تحميل القائمة وتحقق مجدداً."],
 not_pending: ["This request is no longer awaiting a decision.", "결정 대기 중인 요청이 아닙니다.", "لم يعد هذا الطلب ينتظر قراراً."],
 unavailable: ["Current conditions could not be verified. Check again before deciding.", "현재 조건을 확인하지 못했습니다. 결정 전에 다시 확인하세요.", "تعذّر التحقق من الشروط الحالية. تحقق مجدداً قبل اتخاذ القرار."],
 capacity: ["The session history limit was reached. Keep unresolved request references and inspect history before reloading this page.", "세션 이력 한도에 도달했습니다. 미확인 요청 참조를 보관하고 기록을 확인한 뒤 페이지를 다시 로드하세요.", "تم بلوغ حد سجل الجلسة. احتفظ بمراجع الطلبات غير المحسومة وراجع السجل قبل إعادة تحميل الصفحة."],
 session: ["The sign-in session changed or could not be verified. Sign in again before continuing.", "로그인 세션이 변경됐거나 확인되지 않았습니다. 다시 로그인한 뒤 진행하세요.", "تغيّرت جلسة تسجيل الدخول أو تعذّر التحقق منها. سجّل الدخول مجدداً قبل المتابعة."],
 incorrect_amount: ["Incorrect amount", "금액 오류", "مبلغ غير صحيح"], missing_evidence: ["Missing evidence", "근거 누락", "أدلة مفقودة"], policy_violation: ["Policy violation", "정책 위반", "مخالفة السياسة"], other: ["Other reason", "기타 사유", "سبب آخر"],
} as const;
export type DecisionCopyKey = keyof typeof copy;
export function dc(l: Locale, key: DecisionCopyKey) { const [en,ko,ar] = copy[key]; return t(l,en,ko,ar); }
export function decisionMessage(l: Locale, code: string) {
 if (code.includes("capacity")) return dc(l,"capacity");
 if (/session|unauthenticated|csrf/.test(code)) return dc(l,"session");
 if (code === "revision_conflict") return dc(l,"evidence_stale");
 return Object.hasOwn(copy,code) ? dc(l,code as DecisionCopyKey) : dc(l,"unavailable");
}
