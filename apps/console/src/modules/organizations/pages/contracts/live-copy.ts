import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

const iso = (s: string) => (s ? "⁨" + s + "⁩" : s);

const words = (l: Locale, table: Record<string, [string, string, string]>, key: string, unknown: (k: string) => string) => {
  const w = table[key];
  return w ? t(l, w[0], w[1], w[2]) : unknown(key);
};

const stateWords: Record<string, [string, string, string]> = {
  not_started: ["Not started", "시작 전", "لم يبدأ"],
  active: ["Active", "진행 중", "نشط"],
  suspended: ["Suspended", "일시 중지", "معلّق"],
  ended: ["Ended", "종료", "منتهٍ"],
};
const settlementWords: Record<string, [string, string, string]> = {
  none: ["Not applicable", "해당 없음", "لا ينطبق"],
  pending_final: ["Pending final", "최종 정산 대기", "بانتظار التسوية النهائية"],
  settled: ["Settled", "정산 완료", "تمت التسوية"],
};
const pathWords: Record<string, [string, string, string]> = {
  external_invoice: ["External invoice", "외부 청구서", "فاتورة خارجية"],
  internal_chargeback: ["Internal chargeback", "내부 비용 배부", "تحميل داخلي"],
  direct: ["Direct", "직접 계약", "مباشر"],
  resale: ["Resale", "재판매", "إعادة بيع"],
};
const eventWords: Record<string, [string, string, string]> = {
  start: ["Start", "시작", "بدء"],
  change: ["Terms change", "계약 조건 변경", "تغيير الشروط"],
  renew: ["Renew", "갱신", "تجديد"],
  suspend: ["Suspend", "일시 중지", "تعليق"],
  resume: ["Resume", "재개", "استئناف"],
  end: ["End", "종료", "إنهاء"],
};
const componentWords: Record<string, [string, string, string]> = {
  provider_charge: ["Provider charge", "CSP 사용료", "رسوم المزوّد"],
  seller_service: ["Seller service", "판매사 서비스", "خدمة البائع"],
};
const gapWords: Record<string, [string, string, string]> = {
  source_unavailable: ["Source unavailable", "원천 자료를 받을 수 없음", "المصدر غير متاح"],
  access_withdrawn_emergency: ["Access withdrawn in an emergency", "긴급 접근 회수", "سحب الوصول في حالة طارئة"],
  provider_refused: ["Provider refused", "공급사가 거부함", "رفض المزوّد"],
  not_attempted: ["Not attempted", "수집하지 않음", "لم تتم المحاولة"],
  late_source_received: ["Late source received", "늦게 받은 원천 자료", "استلام مصدر متأخر"],
};
const reopenWords: Record<string, [string, string, string]> = {
  late_source: ["Late source", "늦게 받은 원천 자료", "مصدر متأخر"],
  late_credit: ["Late credit", "늦게 확정된 크레딧", "رصيد دائن متأخر"],
  correction: ["Correction", "정정", "تصحيح"],
};
const cancelWords: Record<string, [string, string, string]> = {
  recorded_in_error: ["Recorded in error", "잘못 기록함", "سُجّل خطأً"],
  contract_continues: ["The contract continues", "계약이 계속됨", "يستمر العقد"],
};
const recordWords: Record<string, [string, string, string]> = {
  batch: ["Lifecycle change", "수명주기 변경", "تغيير دورة الحياة"],
  evidence: ["Evidence", "증빙 기록", "دليل"],
  settle: ["Settled", "정산", "تسوية"],
  reopen: ["Reopened", "재개봉", "إعادة فتح"],
};

const blockerWords: Record<string, [string, string, string]> = {
  settled: ["the contract is settled. Reopen it first.", "계약이 정산 완료 상태입니다. 먼저 다시 열어야 합니다.", "العقد مسوّى. أعد فتحه أولًا."],
  ended: ["the contract is ended.", "계약이 종료되었습니다.", "العقد منتهٍ."],
  not_active: ["the contract is not active at its last event.", "마지막 이벤트 기준으로 계약이 진행 중이 아닙니다.", "العقد ليس نشطًا عند آخر حدث."],
  not_suspended: ["the contract is not suspended at its last event.", "마지막 이벤트 기준으로 계약이 일시 중지 상태가 아닙니다.", "العقد ليس معلّقًا عند آخر حدث."],
  horizon: ["the earliest day is more than 3,660 days ahead.", "가장 이른 날짜가 3,660일 이후입니다.", "أقرب يوم بعد أكثر من 3660 يومًا."],
  write_limit: ["this contract's write log is full.", "이 계약의 기록 한도에 도달했습니다.", "سجل الكتابة لهذا العقد ممتلئ."],
  nothing_withdrawable: ["no event can be withdrawn now.", "지금 철회할 수 있는 이벤트가 없습니다.", "لا يوجد حدث يمكن سحبه الآن."],
  not_withdrawable: ["start, terms-change and past events are fixed in this build.", "이번 빌드에서는 시작·조건 변경·지난 이벤트가 고정됩니다.", "أحداث البدء وتغيير الشروط والأحداث الماضية ثابتة في هذا الإصدار."],
  breaks_lifecycle: ["withdrawing it would leave an invalid lifecycle. Withdraw the later event first.", "철회하면 수명주기가 맞지 않게 됩니다. 뒤의 이벤트를 먼저 철회하세요.", "سحبه يجعل دورة الحياة غير صالحة. اسحب الحدث اللاحق أولًا."],
  not_pending_final: ["final settlement is not pending.", "최종 정산 대기 상태가 아닙니다.", "التسوية النهائية ليست قيد الانتظار."],
  end_not_reached: ["the end day has not been reached yet.", "아직 종료일이 되지 않았습니다.", "لم يحن يوم الانتهاء بعد."],
  open_gaps: ["evidence gaps are still open.", "아직 해결되지 않은 증빙 공백이 있습니다.", "ما زالت هناك فجوات أدلة مفتوحة."],
  evidence_needed: ["secured evidence newer than the last change and covering the end day is needed.", "마지막 변경 이후에 기록되고 종료일까지 포함하는 확보 증빙이 필요합니다.", "يلزم دليل مؤمَّن أحدث من آخر تغيير ويغطي يوم الانتهاء."],
  not_settled: ["the contract is not settled.", "계약이 정산 완료 상태가 아닙니다.", "العقد غير مسوّى."],
};

const commandWords: Record<string, [string, string, string]> = {
  "contract.suspend": ["Suspend", "일시 중지", "تعليق"],
  "contract.resume": ["Resume", "재개", "استئناف"],
  "contract.renew": ["Renew", "갱신", "تجديد"],
  "contract.end": ["End", "종료", "إنهاء"],
  "contract.withdraw": ["Withdraw an event", "이벤트 철회", "سحب حدث"],
  "contract.evidence.secured": ["Record secured evidence", "확보 증빙 기록", "تسجيل دليل مؤمَّن"],
  "contract.evidence.missing": ["Record missing evidence", "증빙 공백 기록", "تسجيل دليل مفقود"],
  "contract.settle": ["Settle", "정산 완료 처리", "تسوية"],
  "contract.reopen": ["Reopen", "다시 열기", "إعادة فتح"],
};

export const k = {
  deniedTitle: (l: Locale) => t(l, "Contracts are not available to you", "계약을 볼 수 없습니다", "العقود غير متاحة لك"),
  denied: (l: Locale) =>
    t(l, "This screen needs installation administration. Ask an administrator if you need access.", "이 화면은 설치 관리 권한이 필요합니다. 필요하면 관리자에게 요청하세요.", "تحتاج هذه الشاشة إلى صلاحية إدارة التثبيت. اطلب الوصول من مسؤول إذا احتجت إليه."),
  sessionEnded: (l: Locale) => t(l, "Your session has ended.", "세션이 끝났습니다.", "انتهت جلستك."),
  signInAgain: (l: Locale) => t(l, "Sign in again", "다시 로그인", "تسجيل الدخول مجددًا"),
  title: (l: Locale) => t(l, "Contracts and lifecycle", "계약과 수명주기", "العقود ودورة الحياة"),
  viewTitle: (l: Locale, v: string) =>
    v === "schedule" ? t(l, "Billing schedule", "청구 일정", "جدول الفوترة")
      : v === "products" ? t(l, "Products and rates", "상품·요율", "المنتجات والأسعار")
        : v === "documents" ? t(l, "Documents and payment", "문서·지급", "المستندات والدفع")
          : v === "termination" ? t(l, "Termination and final settlement", "종료·최종 정산", "الإنهاء والتسوية النهائية")
            : v === "payment-terms" ? t(l, "Payment terms", "납부 조건", "شروط الدفع")
              : v === "recognition" ? t(l, "Recognition", "인식 주기", "الاعتراف")
                : t(l, "Parties and path", "당사자·경로", "الأطراف والمسار"),
  stateOn: (l: Locale) => t(l, "State on", "상태 기준일", "الحالة في"),
  onInvalid: (l: Locale) => t(l, "Enter a valid date for the state.", "상태 기준일에 올바른 날짜를 입력하세요.", "أدخل تاريخًا صالحًا للحالة."),
  stateOnHint: (l: Locale) => t(l, "Changes the State column of the list only. Dates are UTC.", "목록의 상태 열에만 적용됩니다. 날짜는 UTC입니다.", "يغيّر عمود الحالة في القائمة فقط. التواريخ بالتوقيت العالمي."),
  today: (l: Locale) => t(l, "Today (UTC)", "오늘(UTC)", "اليوم (UTC)"),
  revision: (l: Locale) => t(l, "Data revision", "데이터 리비전", "مراجعة البيانات"),
  contractsCount: (l: Locale) => t(l, "Contracts", "계약", "العقود"),
  reload: (l: Locale) => t(l, "Reload", "새로 고침", "إعادة التحميل"),
  impactLead: (l: Locale) => t(l, "Not computed in this build:", "이번 빌드에서 계산하지 않음:", "غير محسوب في هذا الإصدار:"),
  impactBody: (l: Locale) =>
    t(l, "money, proration, publication, ERP and the final amount decision. Nothing here is shown as zero.", "금액, 일할, 공개, ERP, 최종 금액 결정. 이 화면의 어떤 값도 0으로 표시하지 않습니다.", "المبالغ والتناسب والنشر وERP وقرار المبلغ النهائي. لا يُعرض أي شيء هنا على أنه صفر."),
  impactUnknown: (l: Locale) => t(l, "The server reported an impact value this screen does not know. Treat every amount as unknown.", "서버가 이 화면이 모르는 영향 값을 보냈습니다. 모든 금액을 알 수 없음으로 보세요.", "أرسل الخادم قيمة أثر لا تعرفها هذه الشاشة. اعتبر كل المبالغ غير معروفة."),
  loading: (l: Locale) => t(l, "Loading contracts…", "계약을 불러오는 중…", "جارٍ تحميل العقود…"),
  emptyAll: (l: Locale) => t(l, "No contracts in this workspace.", "이 업무 공간에 계약이 없습니다.", "لا توجد عقود في مساحة العمل هذه."),
  emptyFiltered: (l: Locale) => t(l, "No contracts match the filters.", "필터에 맞는 계약이 없습니다.", "لا توجد عقود تطابق عوامل التصفية."),
  clearFilters: (l: Locale) => t(l, "Clear filters", "필터 지우기", "مسح عوامل التصفية"),
  filters: (l: Locale, n: number) => (n ? t(l, `Filters · ${n} active`, `필터 · ${n}개 적용`, `عوامل التصفية · ${n} نشطة`) : t(l, "Filters", "필터", "عوامل التصفية")),
  any: (l: Locale) => t(l, "Any", "전체", "الكل"),
  countOf: (l: Locale, n: number, m: number) => t(l, `${n} of ${m} contracts`, `계약 ${m}개 중 ${n}개`, `${n} من ${m} عقود`),
  notFound: (l: Locale) => t(l, "This contract is not available.", "이 계약을 볼 수 없습니다.", "هذا العقد غير متاح."),
  showAll: (l: Locale) => t(l, "Show all contracts", "모든 계약 보기", "عرض كل العقود"),
  back: (l: Locale) => t(l, "Back to contracts", "계약 목록으로", "العودة إلى العقود"),
  colContract: (l: Locale) => t(l, "Contract", "계약", "العقد"),
  colBuyer: (l: Locale) => t(l, "Buyer (legal entity)", "구매 주체(법인)", "المشتري (الكيان القانوني)"),
  colPayer: (l: Locale) => t(l, "Payer", "지급 주체", "الدافع"),
  colPath: (l: Locale) => t(l, "Issuance path", "발행 경로", "مسار الإصدار"),
  colPeriod: (l: Locale) => t(l, "Period", "기간", "الفترة"),
  colStateOn: (l: Locale, d: string) => t(l, `State on ${iso(d)}`, `${iso(d)} 기준 상태`, `الحالة في ${iso(d)}`),
  colStateOnLabel: (l: Locale) => t(l, "State on", "상태 기준일", "الحالة في"),
  colSettlement: (l: Locale) => t(l, "Final settlement (current)", "최종 정산(현재)", "التسوية النهائية (الحالية)"),
  colGaps: (l: Locale) => t(l, "Open gaps", "미해결 공백", "فجوات مفتوحة"),
  contractState: (l: Locale) => t(l, "Contract state", "계약 상태", "حالة العقد"),
  finalSettlement: (l: Locale) => t(l, "Final settlement", "최종 정산", "التسوية النهائية"),
  state: (l: Locale, s: string) => words(l, stateWords, s, (x) => t(l, `Unknown state (${x})`, `알 수 없는 상태(${x})`, `حالة غير معروفة (${x})`)),
  settlement: (l: Locale, s: string) => words(l, settlementWords, s, (x) => t(l, `Unknown (${x})`, `알 수 없음(${x})`, `غير معروف (${x})`)),
  path: (l: Locale, s: string) => (s ? words(l, pathWords, s, (x) => t(l, `Unknown (${x})`, `알 수 없음(${x})`, `غير معروف (${x})`)) : t(l, "Not recorded", "기록 없음", "غير مسجّل")),
  event: (l: Locale, s: string) => words(l, eventWords, s, (x) => t(l, `Unknown event (${x})`, `알 수 없는 이벤트(${x})`, `حدث غير معروف (${x})`)),
  component: (l: Locale, s: string) => words(l, componentWords, s, (x) => t(l, `Unknown (${x})`, `알 수 없음(${x})`, `غير معروف (${x})`)),
  gap: (l: Locale, s: string) => words(l, gapWords, s, (x) => t(l, `Unknown reason (${x})`, `알 수 없는 사유(${x})`, `سبب غير معروف (${x})`)),
  reopenCause: (l: Locale, s: string) => words(l, reopenWords, s, (x) => x),
  cancel: (l: Locale, s: string) => words(l, cancelWords, s, (x) => x),
  record: (l: Locale, s: string) => words(l, recordWords, s, (x) => t(l, `Unknown (${x})`, `알 수 없음(${x})`, `غير معروف (${x})`)),
  command: (l: Locale, key: string) => words(l, commandWords, key, (x) => x),
  blocked: (l: Locale, key: string, blocker: string) => {
    const name = words(l, commandWords, key, (x) => x);
    const why = blockerWords[blocker];
    if (!why) return t(l, `${name} is not available for this contract now.`, `${name}: 지금 이 계약에서는 사용할 수 없습니다.`, `${name} غير متاح لهذا العقد الآن.`);
    return t(l, `${name} is not available: ${why[0]}`, `${name} 불가: ${why[1]}`, `${name} غير متاح: ${why[2]}`);
  },
  eventBlocked: (l: Locale, blocker: string) => {
    const why = blockerWords[blocker];
    return why ? t(l, "Cannot be withdrawn: " + why[0], "철회 불가: " + why[1], "لا يمكن السحب: " + why[2]) : t(l, "Not available for this event now.", "지금 이 이벤트에는 사용할 수 없습니다.", "غير متاح لهذا الحدث الآن.");
  },
  since: (l: Locale, d: string) => t(l, `From ${iso(d)}`, `${iso(d)}부터`, `اعتبارًا من ${iso(d)}`),
  until: (l: Locale, d: string) => t(l, `Until ${iso(d)}, not included`, `${iso(d)} 전까지(해당일 제외)`, `حتى ${iso(d)}، غير مشمول`),
  noEnd: (l: Locale) => t(l, "No end", "종료일 없음", "بلا نهاية"),
  endsFrom: (l: Locale, d: string) => t(l, `Running until ${iso(d)}, not included`, `${iso(d)} 전까지 운영(해당일 제외)`, `يعمل حتى ${iso(d)}، غير مشمول`),
  endedFrom: (l: Locale, d: string) => t(l, `Ended from ${iso(d)}`, `${iso(d)}부터 종료`, `منتهٍ اعتبارًا من ${iso(d)}`),
  endedPending: (l: Locale) => t(l, "Ended. Final settlement is still pending; ending is not settling.", "종료되었습니다. 최종 정산은 아직 대기 중이며, 종료가 정산을 뜻하지는 않습니다.", "منتهٍ. التسوية النهائية ما زالت قيد الانتظار؛ الإنهاء ليس تسوية."),
  stateToday: (l: Locale) => t(l, "Contract state today", "오늘 기준 계약 상태", "حالة العقد اليوم"),
  settlementCurrent: (l: Locale) => t(l, "Final settlement (current)", "최종 정산(현재)", "التسوية النهائية (الحالية)"),
  none: (l: Locale) => t(l, "None", "없음", "لا شيء"),
  unknown: (l: Locale) => t(l, "Unknown", "알 수 없음", "غير معروف"),
  notApplicable: (l: Locale) => t(l, "Not applicable", "해당 없음", "لا ينطبق"),
  notRecorded: (l: Locale) => t(l, "Not recorded", "기록 없음", "غير مسجّل"),
  notComputed: (l: Locale) => t(l, "Not computed", "계산하지 않음", "غير محسوب"),
  notZero: (l: Locale) => t(l, "Not zero.", "0이 아닙니다.", "ليس صفرًا."),
  parties: (l: Locale) => t(l, "Parties", "당사자", "الأطراف"),
  buyer: (l: Locale) => t(l, "Buyer", "구매 주체", "المشتري"),
  payer: (l: Locale) => t(l, "Payer", "지급 주체", "الدافع"),
  issuer: (l: Locale) => t(l, "Issuer", "발행 주체", "المُصدِر"),
  billingSetup: (l: Locale) => t(l, "Billing setup", "청구 설정", "إعداد الفوترة"),
  profile: (l: Locale) => t(l, "Billing profile", "청구 프로파일", "ملف الفوترة"),
  currency: (l: Locale) => t(l, "Currency", "통화", "العملة"),
  timezone: (l: Locale) => t(l, "Timezone", "시간대", "المنطقة الزمنية"),
  cycle: (l: Locale) => t(l, "Billing cycle", "청구 주기", "دورة الفوترة"),
  cycleWord: (l: Locale, s: string) => (s === "monthly" ? t(l, "Monthly", "월별", "شهري") : s || t(l, "Not recorded", "기록 없음", "غير مسجّل")),
  terms: (l: Locale) => t(l, "Terms by revision", "개정별 계약 조건", "الشروط حسب المراجعة"),
  termsRevision: (l: Locale, n: string) => t(l, `Terms revision ${n}`, `계약 조건 개정 ${n}`, `مراجعة الشروط ${n}`),
  colRevision: (l: Locale) => t(l, "Revision", "개정", "المراجعة"),
  customer: (l: Locale) => t(l, "Customer", "고객", "العميل"),
  seller: (l: Locale) => t(l, "Seller", "판매사", "البائع"),
  noSeller: (l: Locale) => t(l, "No seller (direct)", "판매사 없음(직접 계약)", "لا بائع (مباشر)"),
  transactionPath: (l: Locale) => t(l, "Transaction path", "거래 경로", "مسار المعاملة"),
  noTerms: (l: Locale) => t(l, "No terms recorded.", "기록된 계약 조건이 없습니다.", "لا شروط مسجّلة."),
  events: (l: Locale) => t(l, "Lifecycle events", "수명주기 이벤트", "أحداث دورة الحياة"),
  colDay: (l: Locale) => t(l, "Day", "날짜", "اليوم"),
  dayStart: (l: Locale) => t(l, "From the start of the day, UTC", "해당일 시작부터, UTC", "من بداية اليوم، UTC"),
  colEvent: (l: Locale) => t(l, "Event", "이벤트", "الحدث"),
  colRate: (l: Locale) => t(l, "Rate revision", "요율 개정", "مراجعة السعر"),
  colAction: (l: Locale) => t(l, "Action", "작업", "الإجراء"),
  withdraw: (l: Locale) => t(l, "Withdraw", "철회", "سحب"),
  withdrawWhat: (l: Locale, e: string, d: string) => t(l, ` the ${e} on ${iso(d)}`, ` ${iso(d)} ${e}`, ` ${e} في ${iso(d)}`),
  noEvents: (l: Locale) => t(l, "No lifecycle events.", "수명주기 이벤트가 없습니다.", "لا أحداث لدورة الحياة."),
  changeLifecycle: (l: Locale) => t(l, "Change the lifecycle", "수명주기 변경", "تغيير دورة الحياة"),
  segments: (l: Locale) => t(l, "Segments", "구간", "المقاطع"),
  periodFrom: (l: Locale) => t(l, "From", "시작", "من"),
  periodUntil: (l: Locale) => t(l, "Until", "끝", "حتى"),
  periodUntilHint: (l: Locale) => t(l, "Not included. At most 366 days.", "해당일은 포함하지 않습니다. 최대 366일.", "غير مشمول. 366 يومًا كحد أقصى."),
  showPeriod: (l: Locale) => t(l, "Show period", "기간 보기", "عرض الفترة"),
  periodProblem: (l: Locale, p: string) =>
    p === "order" ? t(l, "Until must be after from.", "끝 날짜는 시작 날짜보다 뒤여야 합니다.", "يجب أن يكون تاريخ النهاية بعد البداية.")
      : p === "span" ? t(l, "The period can be at most 366 days.", "기간은 최대 366일입니다.", "يمكن أن تكون الفترة 366 يومًا كحد أقصى.")
        : t(l, "Enter a valid date.", "올바른 날짜를 입력하세요.", "أدخل تاريخًا صالحًا."),
  prorationNote: (l: Locale) => t(l, "Proration is not computed in this build. Segments show days only, with no amounts.", "이번 빌드에서는 일할 계산을 하지 않습니다. 구간에는 일수만 있고 금액은 없습니다.", "لا يُحسب التناسب في هذا الإصدار. تعرض المقاطع الأيام فقط بلا مبالغ."),
  colFrom: (l: Locale) => t(l, "From", "시작", "من"),
  colUntil: (l: Locale) => t(l, "Until, not included", "끝(해당일 제외)", "حتى، غير مشمول"),
  colDays: (l: Locale) => t(l, "Days", "일수", "الأيام"),
  colState: (l: Locale) => t(l, "State", "상태", "الحالة"),
  noSegments: (l: Locale) => t(l, "No days in this period.", "이 기간에는 날짜가 없습니다.", "لا أيام في هذه الفترة."),
  components: (l: Locale) => t(l, "Component", "구성요소", "المكوّن"),
  colKind: (l: Locale) => t(l, "Kind", "종류", "النوع"),
  colProviderPayment: (l: Locale) => t(l, "Provider payment by", "CSP 대금 지급 주체", "دفع المزوّد بواسطة"),
  colSellerBillable: (l: Locale) => t(l, "Seller billable", "판매사 청구 대상", "قابل للفوترة من البائع"),
  notApplicableService: (l: Locale) => t(l, "Not applicable (seller service)", "해당 없음(판매사 서비스)", "لا ينطبق (خدمة البائع)"),
  yes: (l: Locale) => t(l, "Yes", "예", "نعم"),
  no: (l: Locale) => t(l, "No", "아니요", "لا"),
  ratesTitle: (l: Locale) => t(l, "Rates and allowances", "요율과 포함량", "الأسعار والمخصصات"),
  unconnected: (l: Locale) => t(l, "Not connected in this build", "이번 빌드에서는 연결되지 않음", "غير متصل في هذا الإصدار"),
  unconnectedItems: (l: Locale, v: string) =>
    v === "documents"
      ? [t(l, "ERP order and prepayment references", "ERP 주문·선불 참조", "مراجع أمر ERP والدفع المسبق"), t(l, "Payment and recipient references", "지급·수신자 참조", "مراجع الدفع والمستلم"), t(l, "Document template revision", "문서 템플릿 개정", "مراجعة قالب المستند")]
      : v === "recognition"
        ? [t(l, "Billing and recognition cadence", "청구·인식 주기", "إيقاع الفوترة والاعتراف"), t(l, "Legal entity and amount basis", "법인과 금액 기준", "الكيان القانوني وأساس المبلغ"), t(l, "Monthly batches and reconciliation", "월별 배치와 대사", "الدفعات الشهرية والتسوية")]
        : [t(l, "Rates per component", "구성요소별 요율", "الأسعار لكل مكوّن"), t(l, "Allowances and minimums", "포함량과 최소 금액", "المخصصات والحدود الدنيا")],
  unconnectedLead: (l: Locale) => t(l, "This will hold:", "연결되면 다음을 담습니다:", "سيحتوي على:"),
  finalFacts: (l: Locale) => t(l, "Final settlement", "최종 정산", "التسوية النهائية"),
  endDay: (l: Locale) => t(l, "End day", "종료일", "يوم الانتهاء"),
  lastEvent: (l: Locale) => t(l, "Last lifecycle event", "마지막 수명주기 이벤트", "آخر حدث في دورة الحياة"),
  reopenings: (l: Locale) => t(l, "Reopenings", "다시 연 횟수", "مرات إعادة الفتح"),
  sequence: (l: Locale) => t(l, "Write sequence", "기록 순번", "تسلسل الكتابة"),
  finalAmount: (l: Locale) => t(l, "Final amount decision", "최종 금액 결정", "قرار المبلغ النهائي"),
  evidence: (l: Locale) => t(l, "Final evidence", "최종 증빙", "الأدلة النهائية"),
  colEvidence: (l: Locale) => t(l, "Evidence", "증빙", "الدليل"),
  colCovered: (l: Locale) => t(l, "Covered until", "포함 기간 끝", "مغطّى حتى"),
  colOwner: (l: Locale) => t(l, "Owner", "담당자", "المسؤول"),
  colFollow: (l: Locale) => t(l, "Follow-up and status", "후속 조치·상태", "المتابعة والحالة"),
  colWrite: (l: Locale) => t(l, "Write", "기록 ID", "الكتابة"),
  secured: (l: Locale) => t(l, "Secured", "확보", "مؤمَّن"),
  missing: (l: Locale) => t(l, "Missing", "공백", "مفقود"),
  open: (l: Locale) => t(l, "Open", "미해결", "مفتوح"),
  resolved: (l: Locale) => t(l, "Resolved", "해결됨", "تمت المعالجة"),
  resolves: (l: Locale, n: number) => t(l, `Resolves ${n} gap(s)`, `공백 ${n}건 해결`, `يعالج ${n} فجوة`),
  noEvidence: (l: Locale) => t(l, "No final evidence recorded.", "기록된 최종 증빙이 없습니다.", "لا أدلة نهائية مسجّلة."),
  settlementActions: (l: Locale) => t(l, "Evidence and settlement", "증빙과 정산", "الأدلة والتسوية"),
  log: (l: Locale) => t(l, "Write log", "기록 이력", "سجل الكتابة"),
  colWhen: (l: Locale) => t(l, "When", "시각", "الوقت"),
  colActor: (l: Locale) => t(l, "Actor", "작업자", "المنفّذ"),
  colWhat: (l: Locale) => t(l, "What changed", "변경 내용", "ما الذي تغيّر"),
  colReason: (l: Locale) => t(l, "Reason", "사유", "السبب"),
  added: (l: Locale, n: number) => t(l, `${n} event(s) added`, `이벤트 ${n}건 추가`, `أضيف ${n} حدث`),
  withdrawn: (l: Locale, n: number) => t(l, `${n} event(s) withdrawn`, `이벤트 ${n}건 철회`, `سُحب ${n} حدث`),
  asOf: (l: Locale, d: string) => t(l, `As of ${iso(d)}`, `${iso(d)} 기준`, `اعتبارًا من ${iso(d)}`),
  earlier: (l: Locale, n: number) => t(l, `${n} earlier records are not shown.`, `이전 기록 ${n}건은 표시하지 않습니다.`, `لا تُعرض ${n} سجلات سابقة.`),
  noRecords: (l: Locale) => t(l, "No writes recorded.", "기록이 없습니다.", "لا كتابات مسجّلة."),
  form: {
    from: (l: Locale) => t(l, "From", "적용일", "من"),
    fromHint: (l: Locale, d: string) => t(l, `Earliest ${iso(d)}. Takes effect from the start of the day, UTC.`, `가장 이른 날짜 ${iso(d)}. 해당일 시작(UTC)부터 적용됩니다.`, `أقرب يوم ${iso(d)}. يسري من بداية اليوم، UTC.`),
    endImpact: (l: Locale) =>
      t(l, "Ending moves final settlement to pending final. It does not cancel provider resources or subscriptions.", "종료하면 최종 정산이 대기 상태가 됩니다. 공급사 자원이나 구독을 해지하지는 않습니다.", "يؤدي الإنهاء إلى جعل التسوية النهائية قيد الانتظار. ولا يلغي موارد المزوّد أو اشتراكاته."),
    endDay: (l: Locale, d: string) => t(l, `The contract is ended from the start of ${iso(d)}, UTC. ${iso(d)} is the first day it is not running.`, `${iso(d)} 시작(UTC)부터 계약이 종료됩니다. ${iso(d)}이 계약이 운영되지 않는 첫날입니다.`, `ينتهي العقد من بداية ${iso(d)}، UTC. ${iso(d)} هو أول يوم لا يعمل فيه.`),
    rationale: (l: Locale) => t(l, "Reason for this change", "변경 사유", "سبب هذا التغيير"),
    rationaleHint: (l: Locale, n: number) => t(l, `Required. ${n} of 256 bytes used. Do not enter customer data.`, `필수. 256바이트 중 ${n}바이트 사용. 고객 정보를 입력하지 마세요.`, `مطلوب. استُخدم ${n} من 256 بايت. لا تُدخل بيانات العملاء.`),
    rationaleMissing: (l: Locale) => t(l, "Enter a reason.", "사유를 입력하세요.", "أدخل سببًا."),
    rationaleLong: (l: Locale) => t(l, "The reason is longer than 256 bytes.", "사유가 256바이트를 넘습니다.", "السبب أطول من 256 بايت."),
    review: (l: Locale) => t(l, "Review", "검토", "مراجعة"),
    cancel: (l: Locale) => t(l, "Cancel", "취소", "إلغاء"),
    backToEdit: (l: Locale) => t(l, "Back to edit", "수정으로 돌아가기", "العودة إلى التعديل"),
    sending: (l: Locale) => t(l, "Sending…", "보내는 중…", "جارٍ الإرسال…"),
    dayFormat: (l: Locale) => t(l, "Enter a valid date.", "올바른 날짜를 입력하세요.", "أدخل تاريخًا صالحًا."),
    dayMin: (l: Locale, d: string) => t(l, `Choose ${iso(d)} or later.`, `${iso(d)} 이후 날짜를 고르세요.`, `اختر ${iso(d)} أو بعده.`),
    dayMax: (l: Locale, d: string) => t(l, `Choose ${iso(d)} or earlier.`, `${iso(d)} 이전 날짜를 고르세요.`, `اختر ${iso(d)} أو قبله.`),
    cancelLegend: (l: Locale) => t(l, "What happened to this event?", "이 이벤트는 어떻게 된 것인가요?", "ماذا حدث لهذا الحدث؟"),
    cancelHint: (l: Locale) =>
      t(l, "Final settlement is pending because of this end. Withdrawing it reopens the contract, so say why.", "이 종료 때문에 최종 정산이 대기 중입니다. 철회하면 계약이 다시 진행되므로 이유를 고르세요.", "التسوية النهائية قيد الانتظار بسبب هذا الإنهاء. سحبه يعيد العقد للعمل، لذا اذكر السبب."),
    cancelMissing: (l: Locale) => t(l, "Choose what happened.", "어떻게 된 것인지 고르세요.", "اختر ما حدث."),
    withdrawKeepsGaps: (l: Locale) => t(l, "Open evidence gaps stay open after the withdrawal.", "철회 후에도 미해결 증빙 공백은 그대로 남습니다.", "تبقى فجوات الأدلة المفتوحة مفتوحة بعد السحب."),
    evidenceKind: (l: Locale) => t(l, "Evidence", "증빙 종류", "نوع الدليل"),
    reference: (l: Locale) => t(l, "Reference", "참조", "المرجع"),
    referenceHint: (l: Locale) => t(l, "Letters, digits, dot, dash, underscore and colon. At most 128.", "영문자, 숫자, 점, 하이픈, 밑줄, 콜론. 최대 128자.", "حروف وأرقام ونقطة وشرطة وشرطة سفلية ونقطتان. 128 كحد أقصى."),
    referenceBad: (l: Locale) => t(l, "Use only letters, digits, dot, dash, underscore and colon.", "영문자, 숫자, 점, 하이픈, 밑줄, 콜론만 쓰세요.", "استخدم الحروف والأرقام والنقطة والشرطة والشرطة السفلية والنقطتين فقط."),
    covered: (l: Locale) => t(l, "Covered until", "포함 기간 끝", "مغطّى حتى"),
    coverEnd: (l: Locale, d: string) => t(l, `To allow settling, cover through ${iso(d)}.`, `정산하려면 ${iso(d)}까지 포함해야 합니다.`, `للسماح بالتسوية، غطِّ حتى ${iso(d)}.`),
    resolvesLegend: (l: Locale) => t(l, "Gaps this resolves (optional)", "이 증빙으로 해결하는 공백(선택)", "الفجوات التي يعالجها (اختياري)"),
    gapReason: (l: Locale) => t(l, "Why evidence is missing", "증빙이 없는 이유", "سبب فقد الدليل"),
    emergency: (l: Locale) => t(l, "Emergency access withdrawal is never delayed to collect evidence.", "증빙을 모으려고 긴급 접근 회수를 늦추지 않습니다.", "لا يُؤخَّر سحب الوصول الطارئ لجمع الأدلة أبدًا."),
    followUp: (l: Locale) => t(l, "Follow-up reference", "후속 조치 참조", "مرجع المتابعة"),
    owner: (l: Locale, who: string) => t(l, `Recorded with you (${who}) as the owner.`, `담당자는 본인(${who})으로 기록됩니다.`, `يُسجَّل باسمك (${who}) بصفتك المسؤول.`),
    chooseReason: (l: Locale) => t(l, "Choose a reason", "사유 선택", "اختر سببًا"),
    asOf: (l: Locale) => t(l, "As of", "기준일", "اعتبارًا من"),
    settleImpact: (l: Locale) => t(l, "Settling closes final evidence only. The final amount decision is not computed.", "정산 완료 처리는 최종 증빙만 마감합니다. 최종 금액 결정은 계산하지 않습니다.", "التسوية تُغلق الأدلة النهائية فقط. قرار المبلغ النهائي غير محسوب."),
    reopenCause: (l: Locale) => t(l, "Reopen cause", "다시 여는 원인", "سبب إعادة الفتح"),
    confirmTitle: (l: Locale) => t(l, "Check and confirm", "확인 후 실행", "تحقّق وأكّد"),
    action: (l: Locale) => t(l, "Action", "작업", "الإجراء"),
    effective: (l: Locale) => t(l, "Effective", "적용", "السريان"),
    effectiveDay: (l: Locale, d: string) => t(l, `From the start of ${iso(d)}, UTC`, `${iso(d)} 시작(UTC)부터`, `من بداية ${iso(d)}، UTC`),
    conflict: (l: Locale) => t(l, "The contract changed since you opened it. Check the details and confirm again.", "연 뒤에 계약이 바뀌었습니다. 내용을 확인하고 다시 확인하세요.", "تغيّر العقد منذ فتحته. تحقّق من التفاصيل وأكّد مرة أخرى."),
    partial: (l: Locale, op: string) => t(l, `Stopped after a change. Operation ${iso(op)}. Check the write log before trying again.`, `변경 후 중단되었습니다. 작업 ${iso(op)}. 다시 시도하기 전에 기록 이력을 확인하세요.`, `توقف بعد تغيير. العملية ${iso(op)}. راجع سجل الكتابة قبل المحاولة مجددًا.`),
    eventGone: (l: Locale) => t(l, "This event can no longer be withdrawn.", "이 이벤트는 더 이상 철회할 수 없습니다.", "لم يعد بالإمكان سحب هذا الحدث."),
    gapGone: (l: Locale) => t(l, "A selected gap is no longer open.", "선택한 공백 중 이미 해결된 것이 있습니다.", "إحدى الفجوات المحددة لم تعد مفتوحة."),
    pending: (l: Locale, cmd: string) =>
      t(l, `The last ${cmd} may have been applied. Check the write log, or send it again.`, `직전 ${cmd} 요청이 이미 반영되었을 수 있습니다. 기록 이력을 확인하거나 다시 보내세요.`, `ربما طُبّق آخر طلب ${cmd}. راجع سجل الكتابة أو أرسله مجددًا.`),
    resolvesMax: (l: Locale, n: number) => t(l, `Choose at most ${n}.`, `최대 ${n}개까지 고를 수 있습니다.`, `اختر ${n} كحد أقصى.`),
    withdrawReopens: (l: Locale) => t(l, "Final settlement returns to not applicable because the contract continues.", "계약이 계속되므로 최종 정산은 해당 없음으로 돌아갑니다.", "تعود التسوية النهائية إلى لا ينطبق لأن العقد يستمر."),
    checkValue: (l: Locale) => t(l, "Check this value and try again.", "이 값을 확인하고 다시 시도하세요.", "تحقّق من هذه القيمة وحاول مجددًا."),
    renew: (l: Locale) => t(l, "That request expired. Review and confirm again.", "요청이 만료되었습니다. 다시 검토하고 확인하세요.", "انتهت صلاحية الطلب. راجع وأكّد مجددًا."),
    stillBlocked: (l: Locale) => t(l, "The contract changed. The current state is shown below.", "계약 상태가 바뀌었습니다. 현재 상태를 아래에 표시합니다.", "تغيّر العقد. تظهر الحالة الحالية أدناه."),
  },
  done: (l: Locale, cmd: string, d: string, settlement: string) =>
    t(l, `${cmd} done${d ? " (" + iso(d) + ")" : ""}. Final settlement: ${settlement}.`, `${cmd} 완료${d ? "(" + iso(d) + ")" : ""}. 최종 정산: ${settlement}.`, `اكتمل ${cmd}${d ? " (" + iso(d) + ")" : ""}. التسوية النهائية: ${settlement}.`),
  confirmVerb: (l: Locale, cmd: string) => t(l, `Confirm: ${cmd}`, `${cmd} 실행`, `تأكيد: ${cmd}`),
};

export const tables = { gapWords, reopenWords, cancelWords, blockerWords, stateWords, settlementWords };
