import type { Locale } from "../../../../shared/types";
export const t = (l: Locale, en: string, ko: string, ar: string) =>
  l === "ko" ? ko : l === "ar" ? ar : en;
export function issueLabel(l: Locale, code: string) {
  const labels: Record<string, [string, string, string]> = {
    name_required: [
      "Enter a review name.",
      "검토명을 입력하세요.",
      "أدخل اسم المراجعة.",
    ],
    draft_bound: [
      "An input exceeds its length limit.",
      "입력 길이 제한을 초과했습니다.",
      "تجاوز أحد المدخلات حد الطول.",
    ],
    month_required: [
      "Choose one charge-start month to calculate.",
      "계산할 과금 시작 월을 선택하세요.",
      "اختر شهر بدء رسوم واحداً للحساب.",
    ],
    month_mismatch: [
      "The row does not match the selected month.",
      "선택 월과 원천 행이 일치하지 않습니다.",
      "الصف لا يطابق الشهر المحدد.",
    ],
    quantity_unavailable: [
      "Pricing quantity is missing.",
      "요금 산정 수량이 없습니다.",
      "كمية التسعير مفقودة.",
    ],
    unit_unavailable: [
      "Pricing unit is missing.",
      "요금 산정 단위가 없습니다.",
      "وحدة التسعير مفقودة.",
    ],
    price_invalid: [
      "Enter a nonnegative unit price, up to 18 decimal places.",
      "판매 단가를 0 이상, 소수점 18자리까지 입력하세요.",
      "أدخل سعر وحدة غير سالب حتى 18 منزلة عشرية.",
    ],
    denominator_invalid: [
      "Enter a positive quantity per price.",
      "단가 기준 수량을 양수로 입력하세요.",
      "أدخل كمية موجبة مقابلة للسعر.",
    ],
    discount_invalid: [
      "Enter a discount from 0 to 100, up to 4 decimal places.",
      "할인율을 0에서 100까지 소수점 4자리까지 입력하세요.",
      "أدخل خصماً من 0 إلى 100 حتى 4 منازل عشرية.",
    ],
    fee_invalid: [
      "Enter a nonnegative fee, up to 18 decimal places.",
      "추가 요금을 0 이상, 소수점 18자리까지 입력하세요.",
      "أدخل رسوماً غير سالبة حتى 18 منزلة عشرية.",
    ],
    amount_overflow: [
      "A calculation stage exceeds the supported amount range.",
      "계산 단계의 금액이 허용 범위를 초과했습니다.",
      "تجاوز مبلغ في إحدى المراحل النطاق المدعوم.",
    ],
    target_invalid: [
      "Enter a valid signed target amount.",
      "목표 금액을 유효한 부호 있는 소수로 입력하세요.",
      "أدخل مبلغاً مستهدفاً عشرياً صالحاً بإشارته.",
    ],
    reason_required: [
      "Enter an adjustment reason, at most 300 characters.",
      "조정 사유를 300자 이내로 입력하세요.",
      "أدخل سبب التعديل بحد أقصى 300 حرف.",
    ],
    reference_required: [
      "Enter an evidence reference, at most 300 characters.",
      "증빙 참조를 300자 이내로 입력하세요.",
      "أدخل مرجع الأدلة بحد أقصى 300 حرف.",
    ],
    candidate_stale: [
      "Run a current candidate preview first.",
      "현재 입력으로 후보 미리보기를 먼저 실행하세요.",
      "شغّل معاينة حالية للمبلغ المرشح أولاً.",
    ],
    row_unavailable: [
      "The selected row is absent from this authorized page. Reset the selection.",
      "선택 행이 현재 허용된 페이지에 없습니다. 행 선택을 초기화하세요.",
      "الصف المحدد غير موجود في هذه الصفحة المصرّح بها. أعد تعيين الاختيار.",
    ],
    source_duplicate: [
      "Duplicate source rows were rejected.",
      "중복된 원천 행을 거부했습니다.",
      "رُفضت صفوف المصدر المكررة.",
    ],
  };
  return labels[code]
    ? t(l, ...labels[code])
    : t(
        l,
        "Source could not be loaded. Retry the query.",
        "원천을 불러오지 못했습니다. 다시 조회하세요.",
        "تعذّر تحميل المصدر. أعد الاستعلام.",
      );
}
export const notCalculated = (l: Locale) =>
  t(l, "Not calculated", "미계산", "لم يُحسب");
export const missing = (l: Locale) =>
  t(l, "Source value missing", "원천 값 없음", "قيمة المصدر مفقودة");
