import type { Locale } from "../../../../../shared/types";
import { stepError } from "./step-copy";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };
const words = {
 purpose: ["Rules for protected settings changes in this installation.", "이 설치의 보호된 설정 변경에 대한 규칙입니다.", "قواعد تغييرات الإعدادات المحمية في هذا التثبيت."],
 installation: ["These rules apply to the whole installation. They are reached here through this contract.", "이 규칙은 설치 전체에 적용됩니다. 이 화면에서는 이 계약을 통해 조회합니다.", "تنطبق هذه القواعد على التثبيت كله. يُوصل إليها هنا عبر هذا العقد."],
 readAt: ["Read at", "조회 시각", "وقت القراءة"],
 none: ["No protected settings rules are recorded.", "기록된 보호된 설정 규칙이 없습니다.", "لا توجد قواعد مسجلة للإعدادات المحمية."],
 unowned: ["Rules exist that this service did not record. Their values are not shown.", "이 서비스가 기록하지 않은 규칙이 있습니다. 그 값은 표시하지 않습니다.", "توجد قواعد لم تسجلها هذه الخدمة. لا تُعرض قيمها."],
 values: ["Rule values", "규칙 값", "قيم القواعد"],
 proposedValues: ["Proposed rule values — not confirmed", "제안된 규칙 값 — 확인 전", "قيم القواعد المقترحة — غير مؤكدة"],
 record: ["Record", "기록", "السجل"],
 approvals: ["Approvals required for a change", "변경에 필요한 승인 수", "عدد الاعتمادات المطلوبة للتغيير"],
 validity: ["Time allowed between verification and decision", "검증 후 결정까지 허용 시간", "المدة المسموح بها بين التحقق والقرار"],
 lead: ["Longest time from request to effective time", "요청부터 적용 시각까지 허용되는 최대 기간", "أطول مدة من الطلب إلى وقت السريان"],
 channelAge: ["Minimum time after channel acceptance before a change request", "채널 수락 후 변경 요청까지 필요한 최소 시간", "أدنى مدة بعد قبول القناة قبل طلب تغيير"],
 recordedState: ["Recorded state", "기록된 상태", "الحالة المسجَّلة"],
 observed: ["Observed now", "현재 관찰된 상태", "المرصود حالياً"],
 pinned: ["Confirmer pinned, not yet observed", "지정된 확인자, 아직 관찰되지 않음", "المؤكِّد المعيَّن، لم يُرصد بعد"],
 absent: ["No rules observed", "관찰된 규칙 없음", "لا توجد قواعد مرصودة"],
 pending: ["Proposal awaiting confirmation", "확인 대기 중인 제안", "اقتراح بانتظار التأكيد"],
 active: ["In effect", "적용 중", "سارية"],
 proposedBy: ["Proposed by", "제안자", "مقدم الاقتراح"],
 proposedAt: ["Proposal observed", "제안 기록 관찰 시각", "وقت رصد الاقتراح"],
 confirmedBy: ["Confirmed by", "확인자", "المؤكِّد"],
 confirmedAt: ["Confirmation observed", "확인 기록 관찰 시각", "وقت رصد التأكيد"],
 exact: ["Exact values in seconds", "초 단위 정확한 값", "القيم الدقيقة بالثواني"],
 actPropose: ["Propose protected settings rules", "보호된 설정 규칙 제안", "اقتراح قواعد الإعدادات المحمية"],
 actConfirm: ["Confirm protected settings rules", "보호된 설정 규칙 확인", "تأكيد قواعد الإعدادات المحمية"],
 ownRules: ["You proposed these rules. The service refuses confirmation by the person who proposed them.", "이 규칙은 본인이 제안했습니다. 제안한 사람의 확인은 서비스가 거절합니다.", "أنت من اقترح هذه القواعد. ترفض الخدمة تأكيدها ممن اقترحها."],
 adminNote: ["Proposing and confirming rules needs installation administration and the permission to read protected settings. It is not one of the step permissions that permission management grants.", "규칙 제안과 확인에는 설치 관리 권한과 보호된 설정 조회 권한이 필요합니다. 권한 관리에서 부여하는 단계 권한에 포함되지 않습니다.", "يتطلب اقتراح القواعد وتأكيدها إدارة التثبيت وصلاحية قراءة الإعدادات المحمية. وليسا من صلاحيات الخطوات التي تمنحها إدارة الصلاحيات."],
 unitLegend: ["Unit", "단위", "الوحدة"], hours: ["Hours", "시간 단위", "ساعات"], days: ["Days", "일 단위", "أيام"],
 hintValidity: ["A whole number, from 1 hour to 30 days.", "정수로 입력합니다. 1시간부터 30일까지.", "عدد صحيح، من ساعة واحدة إلى 30 يوماً."],
 hintLead: ["A whole number, from 1 hour to 366 days.", "정수로 입력합니다. 1시간부터 366일까지.", "عدد صحيح، من ساعة واحدة إلى 366 يوماً."],
 hintChannelAge: ["A whole number, from 24 hours to 90 days.", "정수로 입력합니다. 24시간부터 90일까지.", "عدد صحيح، من 24 ساعة إلى 90 يوماً."],
 fieldReason: ["Reason for these rules", "이 규칙의 사유", "سبب هذه القواعد"],
 confirmReason: ["Reason for confirming", "확인 사유", "سبب التأكيد"],
 noticeFirst: ["The first proposal cannot be replaced or discarded. It applies to the whole installation, and a proposal that nobody confirms blocks rules for the installation.", "첫 제안은 교체하거나 폐기할 수 없습니다. 설치 전체에 적용되며, 아무도 확인하지 않은 제안은 설치의 규칙을 막습니다.", "لا يمكن استبدال الاقتراح الأول أو إلغاؤه. ينطبق على التثبيت كله، والاقتراح الذي لا يؤكده أحد يعطّل قواعد التثبيت."],
 noticeConfirm: ["Confirmed rules cannot be replaced or discarded. They apply to the whole installation.", "확인된 규칙은 교체하거나 폐기할 수 없습니다. 설치 전체에 적용됩니다.", "لا يمكن استبدال القواعد المؤكدة أو إلغاؤها. تنطبق على التثبيت كله."],
 appliesTo: ["Applies to", "적용 범위", "ينطبق على"], wholeInstallation: ["The whole installation", "설치 전체", "التثبيت كله"],
 reachedThrough: ["Reached through issuance settings", "조회 경로의 발행 설정", "عبر إعدادات الإصدار"],
 targetRules: ["Rules", "규칙", "القواعد"],
 invalid: ["Choose the approvals required, enter each time as a whole number with its unit inside the stated range, and enter a reason.", "필요한 승인 수를 선택하고, 각 시간을 안내된 범위 안의 정수와 단위로 입력한 뒤 사유를 입력하세요.", "اختر عدد الاعتمادات المطلوبة، وأدخل كل مدة عدداً صحيحاً مع وحدتها ضمن النطاق المذكور، وأدخل سبباً."],
 invalidConfirm: ["Enter a reason for confirming.", "확인 사유를 입력하세요.", "أدخل سبب التأكيد."],
 errPermission: ["The step may need installation administration and the permission to read protected settings, or the service refused for a reason it does not disclose.", "이 단계에는 설치 관리 권한과 보호된 설정 조회 권한이 필요할 수 있으며, 서비스가 밝히지 않는 사유로 거절했을 수도 있습니다.", "قد تتطلب الخطوة إدارة التثبيت وصلاحية قراءة الإعدادات المحمية، أو أن الخدمة رفضت لسبب لا تفصح عنه."],
 errSelf: ["The person who proposed these rules cannot confirm them.", "규칙을 제안한 사람은 확인할 수 없습니다.", "لا يمكن لمن اقترح هذه القواعد أن يؤكدها."],
 errStale: ["Possible causes: the rules changed since this screen read them, or the service found its records inconsistent. Reload and review again.", "가능한 원인: 이 화면이 조회한 뒤 규칙이 변경됨, 서비스가 기록의 불일치를 발견함. 새로 조회한 뒤 다시 검토하세요.", "الأسباب المحتملة: تغيرت القواعد منذ قراءتها في هذه الشاشة، أو أن الخدمة وجدت سجلاتها غير متسقة. أعد التحميل ثم راجع من جديد."],
 errPrerequisite: ["Possible causes: rules already exist for this installation, or the selected settings are no longer confirmed. The service does not say which.", "가능한 원인: 이 설치에 이미 규칙이 있음, 선택한 설정이 더 이상 확인 완료 상태가 아님. 서비스는 어느 것인지 알려 주지 않습니다.", "الأسباب المحتملة: توجد قواعد لهذا التثبيت، أو لم تعد الإعدادات المحددة مؤكدة. لا تحدد الخدمة أيها."],
 denied: ["These records could not be read. Reading them may need the permission that permission management lists as “Inspect protected settings evidence”. Without them no step can be started from this screen.", "이 기록을 조회하지 못했습니다. 조회하려면 권한 관리의 ‘보호된 설정 근거 조회’ 권한이 필요할 수 있습니다. 조회할 수 없으면 이 화면에서 어떤 단계도 시작할 수 없습니다.", "تعذرت قراءة هذه السجلات. قد تتطلب قراءتها الصلاحية المدرجة في إدارة الصلاحيات باسم «عرض أدلة الإعدادات المحمية». بدونها لا يمكن بدء أي خطوة من هذه الشاشة."],
 prerequisite: ["This read was refused. The selected issuance settings may no longer be confirmed, or the records may exceed what this read returns. Go to issuance settings and select confirmed settings again.", "조회가 거절되었습니다. 선택한 발행 설정이 더 이상 확인 완료 상태가 아니거나, 기록이 이 조회의 반환 한도를 넘었을 수 있습니다. 발행 설정에서 확인 완료된 설정을 다시 선택하세요.", "رُفضت هذه القراءة. ربما لم تعد إعدادات الإصدار المحددة مؤكدة، أو تجاوزت السجلات ما تعيده هذه القراءة. انتقل إلى إعدادات الإصدار وحدِّد إعدادات مؤكدة من جديد."],
} as const;
export type RulesWord = keyof typeof words;
export function rulesText(l: Locale, key: RulesWord): string { return words[key][l === "ko" ? 1 : l === "ar" ? 2 : 0]; }

export function duration(l: Locale, seconds: string): string {
 const n = Number(seconds);
 const [value, unit] = n % 86400 === 0 ? [n / 86400, "day"] : n % 3600 === 0 ? [n / 3600, "hour"] : n % 60 === 0 ? [n / 60, "minute"] : [n, "second"];
 return new Intl.NumberFormat(tags[l], { style: "unit", unit, unitDisplay: "long" }).format(value);
}

export function rulesError(l: Locale, code: string): string {
 const key: RulesWord | "" = code === "permission_denied" ? "errPermission" : code === "self_approval_denied" ? "errSelf" : code === "evidence_stale" || code === "revision_conflict" ? "errStale" : code === "prerequisite_failed" ? "errPrerequisite"
  : code === "invalid_confirm" ? "invalidConfirm" : code === "invalid_setup_request" ? "invalid" : "";
 return key ? rulesText(l, key) : stepError(l, code);
}
