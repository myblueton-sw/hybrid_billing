import type { Locale } from "../../../../shared/types";
export const t = (l: Locale, en: string, ko: string, ar: string) =>
  l === "ko" ? ko : l === "ar" ? ar : en;
export const tabLabel = (l: Locale, k: string) =>
  ({
    runs: t(l, "Billing runs", "실행 목록", "عمليات احتساب الفواتير"),
    inputs: t(l, "Inputs and policies", "입력·정책", "المدخلات والسياسات"),
    trace: t(
      l,
      "Results and rule evidence",
      "결과·규칙 근거",
      "النتائج وأدلة القواعد",
    ),
    comparison: t(
      l,
      "Source input comparison",
      "원천 입력 비교",
      "مقارنة مدخلات المصدر",
    ),
    validation: t(
      l,
      "Pre-billing checks",
      "청구 전 확인",
      "فحوص ما قبل الفوترة",
    ),
    recognition: t(
      l,
      "Recognition cadence and reconciliation",
      "인식 주기·대사",
      "دورية الإثبات والمطابقة",
    ),
  })[
    k as
      "runs" | "inputs" | "trace" | "comparison" | "validation" | "recognition"
  ];
export const basisLabel = (l: Locale, b: "billed_cost" | "effective_cost") =>
  b === "billed_cost"
    ? t(l, "Source billed cost", "원천 청구 비용", "التكلفة المفوترة من المصدر")
    : t(
        l,
        "Source effective cost",
        "원천 실효 비용",
        "التكلفة الفعلية من المصدر",
      );
export const unavailable = (l: Locale) =>
  t(l, "Not connected", "미연결", "غير متصل");
export const unknown = (l: Locale) =>
  t(l, "Unverified", "미확인", "غير متحقق منه");
