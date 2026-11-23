import type { Locale } from "../../../../../shared/types";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };
export function netDays(l: Locale, value: string): string { return new Intl.NumberFormat(tags[l], { style: "unit", unit: "day", unitDisplay: "long" }).format(Number(value)); }

const words = {
 title: ["Payment terms", "납부 조건", "شروط الدفع"],
 purpose: ["The number of calendar days after the invoice issue date in which payment is due, recorded for the selected issuance settings and approved by another person.", "청구서 발행일로부터 납부 기한까지의 달력일 수입니다. 선택한 발행 설정을 기준으로 기록하고 다른 담당자가 승인합니다.", "عدد الأيام التقويمية بعد تاريخ إصدار الفاتورة التي يستحق خلالها الدفع، يُسجَّل لإعدادات الإصدار المحددة ويعتمده شخص آخر."],
 notComputed: ["This screen calculates no due date and does not state that an invoice is payable or overdue.", "이 화면은 납부 기한을 계산하지 않으며, 청구서가 납부 대상이거나 연체라고 표시하지 않습니다.", "لا تحسب هذه الشاشة تاريخ استحقاق ولا تذكر أن فاتورة مستحقة الدفع أو متأخرة."],
 part: ["Payment terms of these settings", "이 설정의 납부 조건", "شروط الدفع لهذه الإعدادات"],
 appliedBy: ["Applied by", "적용자", "طبّقه"], appliedAt: ["Application observed", "적용 기록 관찰 시각", "وقت رصد التطبيق"], approvedRow: ["Approved by", "승인자", "اعتمده"],
 none: ["No payment terms are recorded for these settings.", "이 설정에 기록된 납부 조건이 없습니다.", "لا توجد شروط دفع مسجلة لهذه الإعدادات."],
 values: ["Recorded terms", "기록된 조건", "الشروط المسجلة"], record: ["Record", "기록", "السجل"],
 netDays: ["Net days (calendar)", "납부 기한 일수(달력일)", "أيام الاستحقاق (تقويمية)"],
 revision: ["Terms revision", "조건 개정", "مراجعة الشروط"],
 zone: ["Document time zone of these settings", "이 설정의 문서 시간대", "المنطقة الزمنية للمستند في هذه الإعدادات"],
 zoneNote: ["A new application made under these settings uses this time zone. It is refused if this contract's existing applications use another.", "이 설정으로 새로 적용하면 이 시간대를 사용합니다. 이 계약의 기존 적용이 다른 시간대를 쓰면 거절됩니다.", "أي تطبيق جديد وفق هذه الإعدادات يستخدم هذه المنطقة الزمنية، ويُرفض إذا كانت تطبيقات هذا العقد الحالية تستخدم منطقة أخرى."],
 recordedState: ["Recorded state", "기록된 상태", "الحالة المسجَّلة"], observed: ["Observed now", "현재 관찰된 상태", "المرصود حالياً"],
 pinned: ["Approver pinned, not yet observed", "지정된 승인자, 아직 관찰되지 않음", "المعتمِد المعيَّن، لم يُرصد بعد"],
 registration_unknown: ["Recording outcome unknown", "기록 처리 결과 미확인", "نتيجة التسجيل غير معروفة"],
 pending: ["Recorded, awaiting approval", "기록됨, 승인 대기", "مسجَّلة، بانتظار الاعتماد"],
 approval_unknown: ["Approval outcome unknown", "승인 처리 결과 미확인", "نتيجة الاعتماد غير معروفة"],
 approved: ["Approved", "승인됨", "معتمدة"],
 observed_absent: ["Nothing observed", "관찰된 기록 없음", "لا يوجد سجل مرصود"], observed_pending: ["Awaiting approval", "승인 대기", "بانتظار الاعتماد"], observed_approved: ["Approved", "승인됨", "معتمدة"],
 author: ["Recorded by", "기록자", "سجّلها"], writtenAt: ["Recording observed", "기록 관찰 시각", "وقت رصد التسجيل"],
 approver: ["Approved by", "승인자", "اعتمدها"], approvedAt: ["Approval observed", "승인 기록 관찰 시각", "وقت رصد الاعتماد"],
 bindings: ["Payment terms applied to this contract", "이 계약에 적용된 납부 조건", "شروط الدفع المطبَّقة على هذا العقد"],
 bindingsLead: ["Every application to this contract is listed, because each occupies its dates, whatever its state.", "각 적용은 상태와 관계없이 해당 기간을 차지하므로 이 계약의 모든 적용을 표시합니다.", "يُدرج كل تطبيق على هذا العقد، لأن كل واحد منها يشغل تواريخه أياً كانت حالته."],
 total: ["Applications to this contract", "이 계약의 적용 수", "التطبيقات على هذا العقد"], showing: ["Rows on this page", "현재 페이지 행 수", "صفوف هذه الصفحة"],
 empty: ["No payment terms are applied to this contract.", "이 계약에 적용된 납부 조건이 없습니다.", "لا توجد شروط دفع مطبَّقة على هذا العقد."],
 period: ["Period", "적용 기간", "الفترة"], terms: ["Terms", "조건", "الشروط"], state: ["State", "상태", "الحالة"], people: ["People and times", "담당자와 시각", "الأشخاص والأوقات"], basis: ["Reason and origin", "사유와 출처", "السبب والمصدر"],
 from: ["From", "시작일", "من"], to: ["To", "종료일", "إلى"], noEnd: ["No end date", "종료일 없음", "بلا تاريخ انتهاء"],
 days: ["Calendar days", "달력일 기준", "أيام تقويمية"],
 timeZone: ["Time zone", "시간대", "المنطقة الزمنية"],
 reason: ["Reason", "사유", "السبب"], initial: ["Initial", "최초 적용", "أولي"], renegotiation: ["Renegotiation", "재협상", "إعادة تفاوض"], correction: ["Correction", "정정", "تصحيح"],
 origin: ["Origin", "출처", "المصدر"], originThese: ["These issuance settings", "이 발행 설정", "إعدادات الإصدار هذه"], originOther: ["Other issuance settings", "다른 발행 설정", "إعدادات إصدار أخرى"], originProfile: ["Billing profile", "청구 프로필", "ملف الفوترة"],
 denied: ["These records could not be read. Reading them may need the permission that permission management lists as “Inspect payment terms evidence”. Without them no step can be started from this screen.", "이 기록을 조회하지 못했습니다. 조회하려면 권한 관리의 ‘납부 조건 근거 조회’ 권한이 필요할 수 있습니다. 조회할 수 없으면 이 화면에서 어떤 단계도 시작할 수 없습니다.", "تعذرت قراءة هذه السجلات. قد تتطلب قراءتها الصلاحية المدرجة في إدارة الصلاحيات باسم «عرض أدلة شروط الدفع». بدونها لا يمكن بدء أي خطوة من هذه الشاشة."],
 noScope: ["Select confirmed issuance settings first. Payment terms are recorded for one set of confirmed settings.", "확인 완료된 발행 설정을 먼저 선택하세요. 납부 조건은 확인 완료된 설정 하나를 기준으로 기록됩니다.", "حدِّد أولاً إعدادات إصدار مؤكدة. تُسجَّل شروط الدفع لمجموعة واحدة من الإعدادات المؤكدة."],
 toSettings: ["Go to issuance settings", "발행 설정으로 이동", "الانتقال إلى إعدادات الإصدار"],
 historyDenied: ["The issuance settings of this contract could not be read, so no settings can be selected here. Reading them may need the permission that permission management lists as “Inspect protected settings evidence”.", "이 계약의 발행 설정을 조회하지 못해 여기서 설정을 선택할 수 없습니다. 조회하려면 권한 관리의 ‘보호된 설정 근거 조회’ 권한이 필요할 수 있습니다.", "تعذرت قراءة إعدادات الإصدار لهذا العقد، لذا لا يمكن تحديد إعدادات هنا. قد تتطلب قراءتها الصلاحية المدرجة في إدارة الصلاحيات باسم «عرض أدلة الإعدادات المحمية»."],
} as const;
export type TermWord = keyof typeof words;
export function termText(l: Locale, key: TermWord): string { return words[key][l === "ko" ? 1 : l === "ar" ? 2 : 0]; }
