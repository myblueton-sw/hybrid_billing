import type { Locale } from "../../../../../shared/types";

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
