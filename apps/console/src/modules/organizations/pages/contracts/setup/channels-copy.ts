import type { Locale } from "../../../../../shared/types";

const words = {
 purpose: ["Channels used to verify a protected settings change. A change request needs a channel that was accepted at least the rules' minimum time before the request.", "보호된 설정 변경을 검증할 수 있는 채널입니다. 변경 요청에는 요청 시점보다 규칙이 정한 최소 시간 이상 먼저 수락된 채널이 필요합니다.", "القنوات المستخدمة للتحقق من تغيير في الإعدادات المحمية. يتطلب طلب التغيير قناة قُبلت قبل الطلب بمدة لا تقل عن الحد الأدنى الذي تحدده القواعد."],
 total: ["Channels in scope of these settings", "이 설정 범위의 채널", "القنوات ضمن نطاق هذه الإعدادات"],
 showing: ["Rows on this page", "현재 페이지 행 수", "صفوف هذه الصفحة"],
 other: ["Recorded under another origin, not listed", "다른 출처로 기록됨, 목록에 없음", "مسجلة تحت مصدر آخر، غير مدرجة"],
 unowned: ["Existing without a record of this service, not listed", "이 서비스의 기록 없이 존재함, 목록에 없음", "موجودة بلا سجل لدى هذه الخدمة، غير مدرجة"],
 empty: ["No verification channel is recorded for these settings.", "이 설정에 기록된 검증 채널이 없습니다.", "لا توجد قناة تحقق مسجلة لهذه الإعدادات."],
 channel: ["Channel", "채널", "القناة"], state: ["State", "상태", "الحالة"], people: ["People and times", "담당자와 시각", "الأشخاص والأوقات"], earliest: ["Earliest change request", "가장 이른 변경 요청 시각", "أبكر وقت لطلب تغيير"],
 customer: ["Customer", "고객", "العميل"], entity: ["Issuing entity", "발행 법인", "الجهة المصدرة"], document_kind: ["Document kind", "문서 종류", "نوع المستند"],
 email: ["Email", "이메일", "البريد الإلكتروني"], phone: ["Phone", "전화", "الهاتف"], portal: ["Portal", "포털", "البوابة"], postal: ["Postal mail", "우편", "البريد العادي"],
 recordedState: ["Recorded state", "기록된 상태", "الحالة المسجَّلة"], observed: ["Observed now", "현재 관찰된 상태", "المرصود حالياً"],
 pinned: ["Accepting person pinned, not yet observed", "지정된 수락자, 아직 관찰되지 않음", "المعيَّن للقبول، لم يُرصد بعد"],
 registration_unknown: ["Registration outcome unknown", "등록 처리 결과 미확인", "نتيجة التسجيل غير معروفة"],
 registered: ["Registered, awaiting acceptance", "등록됨, 수락 대기", "مسجَّلة، بانتظار القبول"],
 admission_unknown: ["Acceptance outcome unknown", "수락 처리 결과 미확인", "نتيجة القبول غير معروفة"],
 admitted: ["Accepted", "수락됨", "مقبولة"],
 observed_absent: ["No channel observed", "관찰된 채널 없음", "لا توجد قناة مرصودة"], observed_registered: ["Registered", "등록됨", "مسجَّلة"], observed_admitted: ["Accepted", "수락됨", "مقبولة"], observed_retired: ["Retired", "사용 종료됨", "أُوقف استخدامها"],
 registeredBy: ["Registered by", "등록자", "سجّلها"], registeredAt: ["Registration observed", "등록 기록 관찰 시각", "وقت رصد التسجيل"],
 acceptedBy: ["Accepted by", "수락자", "قبِلها"], acceptedAt: ["Acceptance observed", "수락 기록 관찰 시각", "وقت رصد القبول"],
 retiredBy: ["Retired by", "사용 종료 처리자", "أوقفها"], retiredAt: ["Retirement observed", "사용 종료 기록 관찰 시각", "وقت رصد الإيقاف"],
 notStated: ["Not stated", "표시되지 않음", "غير مذكور"],
 earliestNote: ["The earliest change request time is stated only for a channel accepted under the current rules. It does not state that you may request a change through this channel.", "가장 이른 변경 요청 시각은 현재 규칙에서 수락된 채널에만 표시됩니다. 이 채널로 변경을 요청할 수 있다는 뜻은 아닙니다.", "يُذكر أبكر وقت لطلب تغيير فقط لقناة قُبلت وفق القواعد الحالية. لا يعني ذلك أنه يمكنك طلب تغيير عبر هذه القناة."],
 otherRules: ["Recorded under rules other than the current rules", "현재 규칙이 아닌 규칙에서 기록됨", "مسجلة وفق قواعد غير القواعد الحالية"],
} as const;
export type ChannelWord = keyof typeof words;
export function channelText(l: Locale, key: ChannelWord): string { return words[key][l === "ko" ? 1 : l === "ar" ? 2 : 0]; }
