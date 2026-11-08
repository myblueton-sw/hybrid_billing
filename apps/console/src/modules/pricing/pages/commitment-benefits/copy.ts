import type { Locale } from "../../../../shared/types";
export const t = (l: Locale, en: string, ko: string, ar: string) =>
  l === "ko" ? ko : l === "ar" ? ar : en;
const fields: Record<string, [string, string, string]> = {
  id: ["Source commitment ID", "원천 약정 ID", "معرّف الالتزام في المصدر"],
  name: ["Source commitment name", "원천 약정 이름", "اسم الالتزام في المصدر"],
  type: ["Source commitment type", "원천 약정 유형", "نوع الالتزام في المصدر"],
  category: [
    "Source commitment category",
    "원천 약정 분류",
    "فئة الالتزام في المصدر",
  ],
  status: [
    "Commitment usage classification — source value",
    "약정 사용 분류 — 원천 값",
    "تصنيف استخدام الالتزام — قيمة المصدر",
  ],
  charge_category: ["Charge category", "청구 분류", "فئة الرسوم"],
  charge_class: ["Charge class", "청구 구분", "تصنيف الرسوم"],
  charge_frequency: ["Charge frequency", "청구 주기", "دورية الرسوم"],
};
export const fieldLabel = (l: Locale, k: string) =>
  fields[k] ? t(l, ...fields[k]) : k;
const states: Record<string, [string, string, string]> = {
  source_null: [
    "Source null token",
    "원천 null 값",
    "رمز قيمة فارغة في المصدر",
  ],
  absent: ["Column absent", "열 없음", "العمود غير موجود"],
  blank: ["Blank source value", "빈 원천 값", "قيمة مصدر فارغة"],
  unmapped_version: ["Version not mapped", "버전 미매핑", "الإصدار غير مربوط"],
  invalid_type: ["Invalid source type", "원천 형식 오류", "نوع مصدر غير صالح"],
};
export const stateLabel = (l: Locale, k: string) =>
  states[k] ? t(l, ...states[k]) : k;
export const statusLabel = (l: Locale, k: string) =>
  k === "used"
    ? t(l, "Used classification", "사용 분류", "تصنيف مستخدم")
    : k === "unused"
      ? t(l, "Unused classification", "미사용 분류", "تصنيف غير مستخدم")
      : t(l, "Unknown classification", "분류 미확인", "تصنيف غير معروف");
export const costLabel = (l: Locale, k: string) =>
  k === "billed_cost"
    ? t(l, "Billed cost", "청구 비용", "التكلفة المفوترة")
    : t(l, "Effective cost", "실효 비용", "التكلفة الفعلية");
export const noRows = (l: Locale) =>
  t(l, "No observed rows", "관측된 행 없음", "لا توجد صفوف مرصودة");
export const noValue = (l: Locale) =>
  t(l, "No source value", "원천 값 없음", "لا توجد قيمة مصدر");
export function errorLabel(l: Locale, code: string) {
  if (
    [
      "permission_denied",
      "actor_revoked",
      "scope_denied",
      "revoked",
      "forbidden",
    ].includes(code)
  )
    return t(
      l,
      "Authorization for this scope is unavailable. Source values and detail were removed.",
      "현재 이 범위의 권한을 확인할 수 없습니다. 원천 자료·상세를 제거했습니다.",
      "التفويض لهذا النطاق غير متاح. أُزيلت قيم المصدر والتفاصيل.",
    );
  if (/snapshot|stale|cursor/.test(code))
    return t(
      l,
      "Source evidence changed. Start a new query.",
      "원천 근거가 갱신되었습니다. 새로 조회하세요.",
      "تغيرت أدلة المصدر. ابدأ استعلاماً جديداً.",
    );
  return t(
    l,
    "The source query could not be completed. Check the request identifier and retry this scope.",
    "원천 조회를 완료하지 못했습니다. 요청 식별자와 범위를 확인하고 다시 시도하세요.",
    "تعذّر إكمال استعلام المصدر. تحقّق من معرّف الطلب وأعد محاولة هذا النطاق.",
  );
}
