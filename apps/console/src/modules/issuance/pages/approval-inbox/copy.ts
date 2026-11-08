import type { Locale } from "../../../../shared/types";
import type { Kind, ApprovalRecord } from "./contract";
export const t = (l: Locale, en: string, ko: string, ar: string) =>
  l === "ko" ? ko : l === "ar" ? ar : en;
export const unknown = (l: Locale) =>
  t(l, "Not provided by source", "원천이 제공하지 않음", "لم يقدّمه المصدر");
export const kindLabel = (l: Locale, k: Kind) =>
  ({
    billing_result: t(
      l,
      "Billing result approval",
      "청구 계산 결과 승인",
      "اعتماد نتيجة احتساب الفواتير",
    ),
    final_erp_amount: t(
      l,
      "Final ERP amount approval",
      "최종 ERP 금액 승인",
      "اعتماد مبلغ ERP النهائي",
    ),
    adjustment: t(l, "Adjustment decision", "조정 결정", "قرار التسوية"),
    policy: t(l, "Policy approval", "정책 승인", "اعتماد السياسة"),
    monthly_recognition: t(
      l,
      "Monthly expense recognition approval",
      "월별 비용 인식 승인",
      "اعتماد الإثبات الشهري للمصروفات",
    ),
  })[k];
export const kinds: Kind[] = [
  "billing_result",
  "final_erp_amount",
  "adjustment",
  "policy",
  "monthly_recognition",
];
export const tabLabel = (l: Locale, k: string) =>
  ({
    pending: t(l, "Pending approvals", "승인 대기", "طلبات الاعتماد المعلّقة"),
    decisions: t(l, "My decisions", "내 결정", "قراراتي"),
    delegation: t(
      l,
      "Delegation and substitution",
      "위임·대리",
      "التفويض والإنابة",
    ),
    history: t(l, "Decision history", "결정 이력", "سجل القرارات"),
    recognition: t(
      l,
      "Recognition cadence and reconciliation",
      "인식 주기·대사",
      "دورية الإثبات والمطابقة",
    ),
  })[k as "pending" | "decisions" | "delegation" | "history" | "recognition"];
export const statusLabel = (l: Locale, k: ApprovalRecord["status"]) =>
  ({
    pending: t(l, "Pending in source", "원천 기록: 대기", "المصدر: معلّق"),
    blocked: t(l, "Blocked in source", "원천 기록: 차단", "المصدر: محجوب"),
    approved: t(
      l,
      "Approval recorded in source",
      "원천 기록: 승인",
      "اعتماد مسجّل في المصدر",
    ),
    rejected: t(
      l,
      "Rejection recorded in source",
      "원천 기록: 반려",
      "رفض مسجّل في المصدر",
    ),
    expired: t(
      l,
      "Expired in source",
      "원천 기록: 기한 지남",
      "المصدر: منتهي الصلاحية",
    ),
    superseded: t(
      l,
      "Superseded in source",
      "원천 기록: 대체됨",
      "المصدر: مستبدل",
    ),
  })[k];
export const basisLabel = (l: Locale, b: string) =>
  ({
    net: t(l, "Net amount", "순액", "المبلغ الصافي"),
    tax: t(l, "Tax", "세금", "الضريبة"),
    gross: t(l, "Gross amount", "총액", "المبلغ الإجمالي"),
    expense: t(l, "Expense basis", "비용 인식 기준", "أساس المصروفات"),
  })[b as "net" | "tax" | "gross" | "expense"] ?? b;
export const environment = (l: Locale) =>
  t(
    l,
    "Approval read service unbound",
    "승인 조회 서비스 미연결",
    "خدمة قراءة الاعتمادات غير مربوطة",
  );
export const reviewContext = {
  profile: "approval-read-unbound-v1",
  mode: "unconnected",
  scope: "producer-unbound",
  period: "not-bound",
  organization: "not-bound",
  currency: "not-bound",
};
export const synthetic = (l: Locale) =>
  t(
    l,
    "Synthetic review examples. Not real approval requests.",
    "합성 검토 예시입니다. 실제 승인 요청이 아닙니다.",
    "أمثلة مراجعة تجريبية. ليست طلبات اعتماد حقيقية.",
  );

export const sourceUnknown = (l: Locale) =>
  t(l, "Unverified in source", "미확인 (원천 기록)", "غير متحقق منه في المصدر");
