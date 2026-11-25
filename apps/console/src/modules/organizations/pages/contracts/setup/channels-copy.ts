import type { Locale } from "../../../../../shared/types";
import { stepError } from "./step-copy";

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
 action: ["Action", "작업", "الإجراء"],
 actRegister: ["Register verification channel", "검증 채널 등록", "تسجيل قناة التحقق"],
 actAccept: ["Accept verification channel", "검증 채널 수락", "قبول قناة التحقق"],
 actAcceptShort: ["Accept channel", "채널 수락", "قبول القناة"],
 ownChannel: ["You registered this channel. The service refuses acceptance by the person who registered it.", "이 채널은 본인이 등록했습니다. 등록한 사람의 수락은 서비스가 거절합니다.", "أنت من سجّل هذه القناة. ترفض الخدمة قبولها ممن سجّلها."],
 stepNote: ["Registering and accepting are separate step permissions in permission management.", "등록과 수락은 권한 관리의 서로 다른 단계 권한입니다.", "التسجيل والقبول صلاحيتا خطوة منفصلتان في إدارة الصلاحيات."],
 fieldSubject: ["The channel belongs to", "채널 대상", "القناة تخص"],
 fieldKind: ["Channel type", "채널 유형", "نوع القناة"],
 fieldReference: ["Sample channel", "샘플 채널", "القناة التجريبية"],
 subjectFirst: ["Choose what the channel belongs to first.", "채널 대상을 먼저 선택하세요.", "اختر أولاً ما تخصه القناة."],
 sampleOnly: ["Sample only. It stands for no real address or number.", "샘플입니다. 실제 주소나 번호를 나타내지 않습니다.", "نموذج فقط. لا يمثل عنواناً أو رقماً حقيقياً."],
 fieldReason: ["Reason for registering", "등록 사유", "سبب التسجيل"],
 targetChannel: ["Channel", "채널", "القناة"],
 invalidRegister: ["Choose what the channel belongs to, its type and a sample channel, write a statement, and enter a reason.", "채널 대상, 유형, 샘플 채널을 선택하고 진술을 작성한 뒤 사유를 입력하세요.", "اختر ما تخصه القناة ونوعها وقناة تجريبية، واكتب إفادة، وأدخل سبباً."],
 invalidAccept: ["Write a statement.", "진술을 작성하세요.", "اكتب إفادة."],
 errPermission: ["The step may need the permission that permission management lists as “Register verification channel” or “Accept verification channel”, or the service refused for a reason it does not disclose.", "이 단계에는 권한 관리의 ‘검증 채널 등록’ 또는 ‘검증 채널 수락’ 권한이 필요할 수 있으며, 서비스가 밝히지 않는 사유로 거절했을 수도 있습니다.", "قد تتطلب الخطوة الصلاحية المدرجة في إدارة الصلاحيات باسم «تسجيل قناة التحقق» أو «قبول قناة التحقق»، أو أن الخدمة رفضت لسبب لا تفصح عنه."],
 errShape: ["The service refused the request as written. The sample channel may not be one it offers.", "서비스가 작성된 요청을 거절했습니다. 샘플 채널이 서비스가 제공하는 항목이 아닐 수 있습니다.", "رفضت الخدمة الطلب كما كُتب. قد لا تكون القناة التجريبية مما تعرضه."],
 errSelf: ["The person who registered this channel cannot accept it.", "채널을 등록한 사람은 수락할 수 없습니다.", "لا يمكن لمن سجّل هذه القناة أن يقبلها."],
 errStale: ["Possible causes: the channel changed since this screen read it, the statement cannot be used for this step, or the service found its records inconsistent. Reload and review again.", "가능한 원인: 이 화면이 조회한 뒤 채널이 변경됨, 진술을 이 단계에 사용할 수 없음, 서비스가 기록의 불일치를 발견함. 새로 조회한 뒤 다시 검토하세요.", "الأسباب المحتملة: تغيرت القناة منذ قراءتها في هذه الشاشة، أو لا يمكن استخدام الإفادة لهذه الخطوة، أو أن الخدمة وجدت سجلاتها غير متسقة. أعد التحميل ثم راجع من جديد."],
 errPrerequisite: ["Possible causes: no confirmed rules exist, a limit of channels was reached, or the selected settings are no longer confirmed. The service does not say which.", "가능한 원인: 확인된 규칙이 없음, 채널 한도 도달, 선택한 설정이 더 이상 확인 완료 상태가 아님. 서비스는 어느 것인지 알려 주지 않습니다.", "الأسباب المحتملة: لا توجد قواعد مؤكدة، أو بُلغ حد القنوات، أو لم تعد الإعدادات المحددة مؤكدة. لا تحدد الخدمة أيها."],
 otherRules: ["Recorded under rules other than the current rules", "현재 규칙이 아닌 규칙에서 기록됨", "مسجلة وفق قواعد غير القواعد الحالية"],
} as const;
export type ChannelWord = keyof typeof words;
export function channelText(l: Locale, key: ChannelWord): string { return words[key][l === "ko" ? 1 : l === "ar" ? 2 : 0]; }

export function channelError(l: Locale, code: string): string {
 const key: ChannelWord | "" = code === "permission_denied" ? "errPermission" : code === "self_approval_denied" ? "errSelf" : code === "evidence_stale" || code === "revision_conflict" ? "errStale" : code === "prerequisite_failed" ? "errPrerequisite"
  : code === "invalid_accept" ? "invalidAccept" : code === "invalid_setup_request" ? "invalidRegister" : code === "invalid_request" ? "errShape" : "";
 return key ? channelText(l, key) : stepError(l, code);
}
