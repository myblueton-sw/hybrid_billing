import type { Locale } from "../../../../shared/types";
import type { EvidenceRecord } from "./contract";
export const t = (l: Locale, en: string, ko: string, ar: string) =>
  l === "ko" ? ko : l === "ar" ? ar : en;
export const unknown = (l: Locale) =>
  t(l, "Not provided by source", "원천이 제공하지 않음", "لم يقدّمه المصدر");
export const title = (l: Locale) =>
  t(
    l,
    "Credits and billing inquiries",
    "크레딧·청구 문의",
    "الأرصدة الائتمانية واستفسارات الفوترة",
  );
export const tabLabel = (l: Locale, k: string) =>
  ({
    balances: t(l, "Credit balances", "크레딧 잔액", "أرصدة الائتمان"),
    reservations: t(
      l,
      "Reservations and applications",
      "예약·적용 이력",
      "الحجوزات والتطبيقات",
    ),
    cases: t(
      l,
      "Billing inquiries and disputes",
      "청구 문의·이의",
      "استفسارات الفوترة والاعتراضات",
    ),
    adjustments: t(
      l,
      "Adjustments and reversals",
      "조정·취소 이력",
      "التسويات وعمليات العكس",
    ),
    "payment-terms": t(
      l,
      "Due dates and overdue",
      "지급 기한·연체",
      "مواعيد الاستحقاق والتأخر في السداد",
    ),
  })[k as "balances"];
export const recordLabel = (l: Locale, k: EvidenceRecord["kind"]) =>
  tabLabel(
    l,
    {
      balance: "balances",
      reservation: "reservations",
      case: "cases",
      adjustment: "adjustments",
      payment: "payment-terms",
    }[k],
  );
export const environment = (l: Locale) =>
  t(
    l,
    "Credit and inquiry service unconnected",
    "크레딧·문의 조회 서비스 미연결",
    "خدمة الأرصدة والاستفسارات غير متصلة",
  );
export const reviewContext = {
  profile: "credit-case-read-unbound-v1",
  mode: "unconnected",
  scope: "producer-unbound",
  period: "not-bound",
  organization: "not-bound",
  currency: "not-bound",
};
export const synthetic = (l: Locale) =>
  t(
    l,
    "Synthetic review examples. These are not actual billing or inquiry records.",
    "합성 검토 예시입니다. 실제 청구·문의 기록이 아닙니다.",
    "أمثلة مراجعة تجريبية. ليست سجلات فوترة أو استفسار حقيقية.",
  );
export const reported = (l: Locale, k: string) =>
  ({
    unknown: t(
      l,
      "Source reports unknown",
      "미확인 (원천 기록)",
      "غير معروف في المصدر",
    ),
    current: t(l, "Reported current", "최신으로 보고됨", "مبلّغ عنه كحديث"),
    stale: t(l, "Reported stale", "오래된 정보로 보고됨", "مبلّغ عنه كمتقادم"),
    platform: t(
      l,
      "Platform authority reported",
      "플랫폼 원장 기준 (원천 기록)",
      "مرجعية المنصة في المصدر",
    ),
    external: t(
      l,
      "External authority reported",
      "외부 원장 기준 (원천 기록)",
      "مرجعية خارجية في المصدر",
    ),
    reserved: t(l, "Reservation recorded", "예약으로 기록됨", "حجز مسجّل"),
    applied: t(l, "Application recorded", "적용으로 기록됨", "تطبيق مسجّل"),
    released: t(l, "Release recorded", "해제로 기록됨", "تحرير مسجّل"),
    outcome_unknown: t(
      l,
      "External outcome unknown",
      "외부 결과 미확인",
      "نتيجة خارجية غير معروفة",
    ),
    reversed: t(l, "Reversal recorded", "적용 취소로 기록됨", "عكس مسجّل"),
    open: t(l, "Open in source", "원천 기록: 접수", "مفتوح في المصدر"),
    in_review: t(
      l,
      "In review in source",
      "원천 기록: 검토 중",
      "قيد المراجعة في المصدر",
    ),
    resolved: t(l, "Resolution recorded", "해결로 기록됨", "تسوية مسجّلة"),
    closed: t(l, "Closed in source", "원천 기록: 종료", "مغلق في المصدر"),
    confirmed_normal: t(
      l,
      "Normal confirmation recorded",
      "정상 확인으로 기록됨",
      "تأكيد سلامة مسجّل",
    ),
    correction_requested: t(
      l,
      "Correction request recorded",
      "정정 요청으로 기록됨",
      "طلب تصحيح مسجّل",
    ),
    corrected: t(l, "Correction recorded", "정정으로 기록됨", "تصحيح مسجّل"),
    unresolved: t(
      l,
      "Unresolved in source",
      "원천 기록: 미해결",
      "غير محسوم في المصدر",
    ),
    recorded: t(
      l,
      "Consent recorded in source",
      "동의 기록 있음",
      "موافقة مسجّلة في المصدر",
    ),
    declined: t(
      l,
      "Consent declined in source",
      "동의 거절로 기록됨",
      "رفض الموافقة مسجّل",
    ),
    not_recorded: t(
      l,
      "Consent not recorded",
      "동의 기록 없음",
      "لم تُسجّل موافقة",
    ),
    not_applicable: t(
      l,
      "Explicitly not applicable in source",
      "적용 대상 아님 (원천 기록)",
      "غير منطبق صراحة في المصدر",
    ),
    draft: t(l, "Draft in source", "원천 기록: 초안", "مسودة في المصدر"),
    pending: t(l, "Pending in source", "원천 기록: 대기", "معلّق في المصدر"),
    rejected: t(l, "Rejection recorded", "반려로 기록됨", "رفض مسجّل"),
    credit: t(l, "Credit", "크레딧", "ائتمان"),
    refund: t(l, "Refund", "환불", "رد مبلغ"),
    quantity_reversal: t(
      l,
      "Quantity reversal",
      "수량 적용 취소",
      "عكس تطبيق الكمية",
    ),
    other: t(l, "Other source kind", "기타 원천 종류", "نوع مصدر آخر"),
    issued: t(l, "Issuance recorded", "발행으로 기록됨", "إصدار مسجّل"),
    not_issued: t(
      l,
      "Not issued in source",
      "원천 기록: 미발행",
      "غير صادر في المصدر",
    ),
    internal: t(
      l,
      "Internal document in source",
      "내부 문서 (원천 기록)",
      "مستند داخلي في المصدر",
    ),
    confirmed: t(l, "Confirmation recorded", "확정으로 기록됨", "تأكيد مسجّل"),
    not_provided: t(
      l,
      "Not provided by source",
      "원천이 제공하지 않음",
      "لم يقدّمه المصدر",
    ),
    on_time: t(
      l,
      "Reported not overdue",
      "연체 아님으로 보고됨",
      "مبلّغ عنه كغير متأخر",
    ),
    due_today: t(
      l,
      "Reported due today",
      "오늘이 지급 기한으로 보고됨",
      "مبلّغ عنه كمستحق اليوم",
    ),
    overdue: t(l, "Reported overdue", "연체로 보고됨", "مبلّغ عنه كمتأخر"),
    fully_paid: t(
      l,
      "Reported fully paid",
      "완납으로 보고됨",
      "مبلّغ عنه كمسدّد بالكامل",
    ),
    none: t(
      l,
      "None reported by source",
      "없음 (원천 기록)",
      "لا شيء مبلّغ عنه",
    ),
    billing: t(l, "Billing hold", "청구 보류", "تعليق الفوترة"),
    delivery: t(l, "Delivery hold", "전달 보류", "تعليق التسليم"),
    collections: t(l, "Collections hold", "추심 보류", "تعليق التحصيل"),
    requested: t(
      l,
      "Request recorded; effect not confirmed",
      "요청 기록 있음 · 효력 확정 아님",
      "طلب مسجّل؛ الأثر غير مؤكد",
    ),
    effective: t(
      l,
      "Effectiveness reported",
      "효력 발생으로 보고됨",
      "فعالية مبلّغ عنها",
    ),
  })[k as "unknown"] ??
  t(l, "Source reports unknown", "미확인 (원천 기록)", "غير معروف في المصدر");
