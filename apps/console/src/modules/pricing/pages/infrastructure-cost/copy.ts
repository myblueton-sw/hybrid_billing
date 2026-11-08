import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { Category } from "./model";
export const categoryLabel = (l: Locale, c: Category) => {
  const names: Record<Category, [string, string, string]> = {
    unclassified: ["Unclassified", "미분류", "غير مصنّف"],
    asset: ["Asset", "자산", "أصل"],
    lease: ["Lease", "리스", "إيجار"],
    depreciation: ["Depreciation", "감가상각", "إهلاك"],
    license: ["License", "라이선스", "ترخيص"],
    power: ["Power", "전력", "كهرباء"],
    facility: ["Facility", "시설", "مرافق"],
    network: ["Network", "네트워크", "شبكة"],
    other: ["Other", "기타", "أخرى"],
  };
  return t(l, ...names[c]);
};
export const basisLabel = (l: Locale, basis: string) =>
  basis === "effective_cost"
    ? t(l, "Effective cost", "실효 비용", "التكلفة الفعلية")
    : t(l, "Billed cost", "청구 비용", "التكلفة المفوترة");
export function issueLabel(l: Locale, code: string) {
  const names: Record<string, [string, string, string]> = {
    name_required: [
      "Enter a composition name.",
      "원가 구성명을 입력하세요.",
      "أدخل اسم مكوّنات التكلفة.",
    ],
    month_required: [
      "Choose one charge-start month before saving.",
      "저장하려면 과금 시작 월을 하나 선택하세요.",
      "اختر شهر بدء رسوم واحداً قبل الحفظ.",
    ],
    source_required: [
      "Select at least one source row.",
      "원천 행을 하나 이상 선택하세요.",
      "حدّد صف مصدر واحداً على الأقل.",
    ],
    draft_bound: [
      "The draft exceeds the allowed size.",
      "초안이 허용 크기를 초과했습니다.",
      "تتجاوز المسودة الحجم المسموح.",
    ],
    duplicate_row: [
      "Each source row can be selected only once.",
      "같은 원천 행은 한 번만 선택할 수 있습니다.",
      "يمكن تحديد كل صف مصدر مرة واحدة فقط.",
    ],
    mixed_currency: [
      "Save separate drafts for different currencies.",
      "통화별로 초안을 나누어 저장하세요.",
      "احفظ مسودة منفصلة لكل عملة.",
    ],
    source_unavailable: [
      "A selected row is unavailable in this source page.",
      "선택한 행이 현재 원천 페이지에 없습니다.",
      "صف محدّد غير متاح في صفحة المصدر الحالية.",
    ],
    entry_invalid: [
      "Check the category and reference length.",
      "범주와 근거 참조 길이를 확인하세요.",
      "تحقّق من الفئة وطول المرجع.",
    ],
    month_mismatch: [
      "A row belongs to a different charge-start month.",
      "다른 과금 시작 월의 행이 있습니다.",
      "يوجد صف من شهر بدء رسوم مختلف.",
    ],
    parent_unavailable: [
      "Choose an inclusion target among selected rows.",
      "선택한 행 중에서 포함 대상을 정하세요.",
      "اختر هدف التضمين من الصفوف المحدّدة.",
    ],
    self_inclusion: [
      "A row cannot include itself.",
      "같은 행을 포함 대상으로 정할 수 없습니다.",
      "لا يمكن تضمين الصف في نفسه.",
    ],
    inclusion_reference: [
      "Enter an evidence reference for every declared inclusion.",
      "포함 관계마다 근거 참조를 입력하세요.",
      "أدخل مرجع دليل لكل علاقة تضمين مُعلنة.",
    ],
    interval_mismatch: [
      "Included rows must have identical charge intervals.",
      "포함 관계는 과금 시작·종료 시각이 같은 행 사이에서만 선언할 수 있습니다.",
      "يجب أن تتطابق فترات الرسوم للصفوف المرتبطة بالتضمين.",
    ],
    inclusion_cycle: [
      "Remove the circular inclusion relationship.",
      "순환하는 포함 관계를 해제하세요.",
      "أزل علاقة التضمين الدائرية.",
    ],
  };
  return names[code]
    ? t(l, ...names[code])
    : t(
        l,
        "The source query could not be completed. Retry the query.",
        "원천을 조회하지 못했습니다. 다시 조회하세요.",
        "تعذّر إكمال استعلام المصدر. أعد المحاولة.",
      );
}
