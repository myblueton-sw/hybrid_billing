import type { Locale } from "../../../../shared/types";
import type { Protection } from "./contract";
export const t = (l: Locale, en: string, ko: string, ar: string) =>
  l === "ko" ? ko : l === "ar" ? ar : en;
export const unknown = (l: Locale) =>
  t(l, "Not provided by source", "원천이 제공하지 않음", "لم يقدّمه المصدر");
export const tabLabel = (l: Locale, k: string) =>
  ({
    ledger: t(
      l,
      "Billing application history",
      "청구 적용 이력",
      "سجل تطبيق الفوترة",
    ),
    partial: t(l, "Partial billing", "부분 청구", "الفوترة الجزئية"),
    reversals: t(
      l,
      "Reversals and corrections",
      "적용 취소·정정",
      "عمليات العكس والتصحيح",
    ),
    references: t(l, "External references", "외부 참조", "المراجع الخارجية"),
  })[k as "ledger" | "partial" | "reversals" | "references"];
export const protections: Protection[] = [
  "available",
  "reserved",
  "outcome_unknown",
  "applied",
  "release_pending",
  "released",
  "replacement_pending",
  "cancelled",
  "superseded",
  "unknown",
];
export const protectionLabel = (l: Locale, k: Protection) =>
  ({
    available: t(
      l,
      "Source reports available",
      "원천 기록: 가용",
      "المصدر: متاح",
    ),
    reserved: t(l, "Reserved in source", "원천 기록: 예약", "المصدر: محجوز"),
    outcome_unknown: t(
      l,
      "External outcome unknown · protected",
      "외부 결과 미확인 · 보호 중",
      "نتيجة خارجية غير معروفة · محمي",
    ),
    applied: t(
      l,
      "Application recorded in source",
      "원천 기록: 적용",
      "تطبيق مسجّل في المصدر",
    ),
    release_pending: t(
      l,
      "Release pending in source",
      "원천 기록: 예약 해제 대기",
      "المصدر: تحرير قيد الانتظار",
    ),
    released: t(
      l,
      "Release recorded in source",
      "원천 기록: 예약 해제",
      "تحرير مسجّل في المصدر",
    ),
    replacement_pending: t(
      l,
      "Replacement pending · not rebill authority",
      "대체 검토 대기 · 재청구 권한 아님",
      "استبدال قيد الانتظار · ليس إذناً بإعادة الفوترة",
    ),
    cancelled: t(
      l,
      "Cancellation recorded in source",
      "원천 기록: 취소",
      "إلغاء مسجّل في المصدر",
    ),
    superseded: t(
      l,
      "Superseded in source",
      "원천 기록: 대체됨",
      "المصدر: مستبدل",
    ),
    unknown: t(
      l,
      "Source reports unknown",
      "미확인 (원천 기록)",
      "غير معروف في المصدر",
    ),
  })[k];
export const basisLabel = (l: Locale, b: string) =>
  ({
    net: t(l, "Net amount", "순액", "المبلغ الصافي"),
    tax: t(l, "Tax", "세금", "الضريبة"),
    gross: t(l, "Gross amount", "총액", "المبلغ الإجمالي"),
    expense: t(l, "Expense basis", "비용 인식 기준", "أساس المصروفات"),
  })[b as "net"];
export const dispositionLabel = (l: Locale, k: string) =>
  ({
    none: t(
      l,
      "No correction reported",
      "정정 없음 (원천 기록)",
      "لا تصحيح مبلّغ عنه",
    ),
    money_only: t(l, "Money-only correction", "금액만 정정", "تصحيح مالي فقط"),
    quantity_reversal: t(
      l,
      "Quantity reversal",
      "수량 적용 취소",
      "عكس تطبيق الكمية",
    ),
    replacement_pending: t(
      l,
      "Replacement pending",
      "대체 검토 대기",
      "استبدال قيد الانتظار",
    ),
    cancelled: t(l, "Cancelled", "취소", "ملغى"),
    superseded: t(l, "Superseded", "대체됨", "مستبدل"),
    unknown: t(
      l,
      "Source reports unknown",
      "미확인 (원천 기록)",
      "غير معروف في المصدر",
    ),
  })[k as "none"];
export const outcomeLabel = (l: Locale, k: string) =>
  ({
    pending: t(l, "Pending in source", "원천 기록: 대기", "المصدر: معلّق"),
    unknown: t(
      l,
      "Source reports unknown",
      "미확인 (원천 기록)",
      "غير معروف في المصدر",
    ),
    applied: t(l, "Application reported", "적용으로 보고됨", "تطبيق مبلّغ عنه"),
    not_applied: t(
      l,
      "Nonapplication reported",
      "미적용으로 보고됨",
      "عدم تطبيق مبلّغ عنه",
    ),
    cancelled: t(
      l,
      "Cancellation reported",
      "취소로 보고됨",
      "إلغاء مبلّغ عنه",
    ),
  })[k as "unknown"];
export const externalKind = (l: Locale, k: string) =>
  ({
    erp_order: t(l, "ERP order", "ERP 주문", "طلب ERP"),
    prepayment: t(l, "Prepayment", "선수금", "دفعة مقدّمة"),
    invoice: t(l, "Invoice", "인보이스", "فاتورة"),
    submission: t(l, "Submission", "제출", "إرسال"),
    other: t(l, "Other source reference", "기타 원천 참조", "مرجع مصدر آخر"),
  })[k as "invoice"];
export const environment = (l: Locale) =>
  t(
    l,
    "Billing application history service unconnected",
    "청구 적용 이력 서비스 미연결",
    "خدمة سجل تطبيق الفوترة غير متصلة",
  );
export const reviewContext = {
  profile: "claim-history-unbound-v1",
  mode: "unconnected",
  scope: "producer-unbound",
  period: "not-bound",
  organization: "not-bound",
  currency: "not-bound",
};
export const synthetic = (l: Locale) =>
  t(
    l,
    "Synthetic review examples. These are not actual Claims.",
    "합성 검토 예시입니다. 실제 청구 적용 기록이 아닙니다.",
    "أمثلة مراجعة تجريبية. ليست سجلات تطبيق فوترة حقيقية.",
  );
