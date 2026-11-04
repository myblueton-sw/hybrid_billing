import type { Locale } from "../../../../shared/types";
export const t = (l: Locale, en: string, ko: string, ar: string) =>
  l === "ko" ? ko : l === "ar" ? ar : en;
export const labels: Record<string, [string, string, string]> = {
  name: ["Policy review name", "정책 검토명", "اسم مراجعة السياسة"],
  target: ["Target currency", "대상 통화", "العملة المستهدفة"],
  treatment: [
    "Source conversion declaration",
    "원천 환산 선언",
    "إقرار تحويل المصدر",
  ],
  rate: ["Manual quote", "수동 입력 환율", "سعر الصرف المُدخل يدوياً"],
  quoteUnit: ["Exchange-rate quotation unit", "환율 기준 수량", "وحدة التسعير"],
  provider: [
    "Rate source — manually declared",
    "환율 출처 — 직접 입력",
    "جهة سعر الصرف — تصريح يدوي",
  ],
  reference: ["Evidence reference", "증빙 참조", "مرجع الأدلة"],
  quoteDate: [
    "Rate reference date (UTC)",
    "환율 기준일 (UTC)",
    "تاريخ سعر الصرف (UTC)",
  ],
  reviewDate: [
    "Review date (UTC)",
    "검토 기준일 (UTC)",
    "تاريخ المراجعة (UTC)",
  ],
  maxAge: ["Maximum age (days)", "최대 경과일", "الحد الأقصى للعمر بالأيام"],
  markup: [
    "Review exchange-rate markup (%)",
    "검토 환율 가산율 (%)",
    "نسبة الزيادة",
  ],
  fee: [
    "Flat fee in target currency",
    "대상 통화 정액 수수료",
    "رسوم ثابتة بالعملة المستهدفة",
  ],
  quantum: ["Rounding quantum", "금액 처리 단위", "وحدة التقريب"],
  mode: ["Rounding mode", "반올림 방식", "طريقة التقريب"],
};
export const label = (l: Locale, k: string) =>
  labels[k] ? t(l, ...labels[k]) : k;
export const modeLabels: Record<string, [string, string, string]> = {
  toward_zero: ["Toward zero", "0 방향", "باتجاه الصفر"],
  away_from_zero: ["Away from zero", "0에서 먼 방향", "بعيداً عن الصفر"],
  floor: [
    "Floor — toward negative infinity",
    "내림 — 음의 무한대 방향",
    "إلى الأسفل — نحو سالب اللانهاية",
  ],
  ceiling: [
    "Ceiling — toward positive infinity",
    "올림 — 양의 무한대 방향",
    "إلى الأعلى — نحو موجب اللانهاية",
  ],
  half_up: [
    "Nearest — ties away from zero",
    "가장 가까운 값 — 동점은 0에서 먼 쪽",
    "الأقرب — التعادل بعيداً عن الصفر",
  ],
  half_even: [
    "Nearest — ties to even quantum count",
    "가장 가까운 값 — 동점은 짝수 배수",
    "الأقرب — التعادل إلى عدد مضاعفات زوجي",
  ],
};
export const modeLabel = (l: Locale, k: string) =>
  modeLabels[k] ? t(l, ...modeLabels[k]) : k;
export const errorFields: Record<string, string> = {
  name_required: "name",
  draft_bound: "name",
  month_required: "period",
  amount_unavailable: "source-list",
  target_invalid: "target",
  conversion_declaration: "treatment",
  rate_invalid: "rate",
  quote_unit_invalid: "quote-unit",
  quantum_invalid: "quantum",
  markup_invalid: "markup",
  fee_invalid: "fee",
  mode_invalid: "mode",
  evidence_required: "provider",
  date_invalid: "quote-date",
  age_invalid: "max-age",
  future_quote: "quote-date",
  stale_quote: "max-age",
};
export function issueLabel(l: Locale, c: string) {
  const a: Record<string, [string, string, string]> = {
    name_required: [
      "Enter a policy review name.",
      "정책 검토명을 입력하세요.",
      "أدخل اسم مراجعة السياسة.",
    ],
    draft_bound: [
      "An input exceeds its length limit.",
      "입력 길이 제한을 초과했습니다.",
      "تجاوز أحد المدخلات حد الطول.",
    ],
    month_required: [
      "Select the row’s charge-start month.",
      "원천 행의 과금 시작 월을 선택하세요.",
      "اختر شهر بدء رسوم الصف.",
    ],
    amount_unavailable: [
      "The selected source cost is unavailable. It is not replaced with zero.",
      "선택한 원천 비용이 없습니다. 0으로 대체하지 않습니다.",
      "تكلفة المصدر المحددة غير متاحة. لا تُستبدل بصفر.",
    ],
    target_invalid: [
      "Select an allowed target currency different from the source.",
      "원천과 다른 허용 대상 통화를 선택하세요.",
      "اختر عملة مستهدفة مسموحة تختلف عن المصدر.",
    ],
    conversion_declaration: [
      "An explicit unconverted declaration is required. This is not verified supplier conversion history.",
      "미환산 선언이 없어 계산할 수 없습니다. 공급자 환산 이력으로 검증된 선언이 아닙니다.",
      "يلزم إقرار صريح بعدم التحويل. لم يُتحقق منه بسجل تحويل المزوّد.",
    ],
    rate_invalid: [
      "Enter a positive manual quote, up to 18 decimal places.",
      "수동 환율을 양수, 소수점 18자리까지 입력하세요.",
      "أدخل سعر صرف موجباً حتى 18 منزلة عشرية.",
    ],
    quote_unit_invalid: [
      "Enter a positive quotation unit.",
      "환율 기준 수량를 양수로 입력하세요.",
      "أدخل وحدة تسعير موجبة.",
    ],
    quantum_invalid: [
      "Enter a positive rounding quantum.",
      "금액 처리 단위를 양수로 입력하세요.",
      "أدخل وحدة تقريب موجبة.",
    ],
    markup_invalid: [
      "Enter markup from 0 to 100, up to 4 decimal places.",
      "검토 환율 가산율을 0부터 100까지 소수점 4자리 이내로 입력하세요.",
      "أدخل زيادة من 0 إلى 100 حتى 4 منازل عشرية.",
    ],
    fee_invalid: [
      "Enter a nonnegative flat fee, up to 18 decimal places.",
      "정액 수수료를 0 이상, 소수점 18자리까지 입력하세요.",
      "أدخل رسوماً ثابتة غير سالبة حتى 18 منزلة عشرية.",
    ],
    mode_invalid: [
      "Choose a rounding mode explicitly.",
      "반올림 방식을 직접 선택하세요.",
      "اختر طريقة التقريب صراحةً.",
    ],
    evidence_required: [
      "Enter the manually declared provider and evidence reference.",
      "제공기관과 증빙 참조를 직접 입력하세요.",
      "أدخل الجهة المصرّح بها ومرجع الأدلة.",
    ],
    date_invalid: [
      "Enter real quote and review dates on UTC.",
      "실제 달력에 존재하는 환율 기준일·검토 기준일을 UTC로 입력하세요.",
      "أدخل تاريخي سعر الصرف والمراجعة الصحيحين بتوقيت UTC.",
    ],
    age_invalid: [
      "Enter maximum age as an integer from 0 to 36600.",
      "최대 경과일을 0부터 36600까지 정수로 입력하세요.",
      "أدخل الحد الأقصى للعمر عدداً صحيحاً من 0 إلى 36600.",
    ],
    future_quote: [
      "Quote date must not be after review date.",
      "환율 기준일은 검토 기준일보다 늦을 수 없습니다.",
      "يجب ألا يتجاوز تاريخ سعر الصرف تاريخ المراجعة.",
    ],
    stale_quote: [
      "The quote exceeds the entered maximum age.",
      "환율 기준일이 입력한 허용 경과일을 넘었습니다.",
      "تجاوز سعر الصرف الحد الأقصى للعمر المُدخل.",
    ],
    amount_overflow: [
      "Calculation exceeds the supported coefficient or intermediate range.",
      "계수 또는 중간 계산값이 허용 범위를 초과했습니다.",
      "تجاوز الحساب نطاق المعامل أو القيمة الوسيطة المدعوم.",
    ],
    source_duplicate: [
      "Duplicate source rows were rejected.",
      "중복 원천 행을 거부했습니다.",
      "رُفضت صفوف مصدر مكررة.",
    ],
    row_unavailable: [
      "The selected row is not on the current authorized page. Reset the selection.",
      "선택 행이 현재 허용 페이지에 없습니다. 행 선택을 초기화하세요.",
      "الصف المحدد غير موجود في الصفحة المصرّح بها. أعد تعيين الاختيار.",
    ],
  };
  return a[c]
    ? t(l, ...a[c])
    : t(
        l,
        "Source query failed. Retry the same query.",
        "원천 조회에 실패했습니다. 같은 조회를 다시 시도하세요.",
        "فشل استعلام المصدر. أعد الاستعلام نفسه.",
      );
}
export const missing = (l: Locale) =>
  t(l, "Not entered", "입력하지 않음", "لم يُدخل");
export const pending = (l: Locale) =>
  t(l, "Not calculated", "미계산", "لم يُحسب");
