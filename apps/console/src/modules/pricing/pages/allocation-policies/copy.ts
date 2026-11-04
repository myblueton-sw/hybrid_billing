import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
export const percent = (value: bigint) => {
  const fraction = (value % 10000n)
    .toString()
    .padStart(4, "0")
    .replace(/0+$/, "");
  return `${value / 10000n}${fraction ? "." + fraction : ""}`;
};
export const basisLabel = (l: Locale, value: string) =>
  value === "effective_cost"
    ? t(l, "Effective cost", "실효 비용", "التكلفة الفعلية")
    : t(l, "Billed cost", "청구 비용", "التكلفة المفوترة");
export const totalLabels = (l: Locale) => ({
  assigned: t(
    l,
    "Total recipient allocations",
    "대상별 배분액 합계",
    "مجموع المبالغ الموزّعة على المستلمين",
  ),
  unassigned: t(l, "Unallocated amount", "미배분 금액", "المبلغ غير الموزّع"),
  remainder: t(
    l,
    "Calculation precision remainder",
    "계산 정밀도 잔액",
    "المتبقي الناتج عن دقة الحساب",
  ),
  source: t(
    l,
    "Collected-source basis amount",
    "수집 원천 기준액",
    "مبلغ الأساس من البيانات المجمّعة",
  ),
});
export function issueLabel(l: Locale, code: string) {
  const labels: Record<string, [string, string, string]> = {
    name_required: [
      "Enter a policy name before saving.",
      "저장하려면 배분 정책명을 입력하세요.",
      "أدخل اسم سياسة التوزيع قبل الحفظ.",
    ],
    name_bound: [
      "Use a policy name of 120 characters or fewer.",
      "배분 정책명은 120자 이내로 입력하세요.",
      "استخدم اسماً لا يتجاوز 120 حرفاً.",
    ],
    percentage_invalid: [
      "Enter a percentage for every recipient: 0–100, up to four decimal places.",
      "모든 대상에 0~100의 비율을 소수점 4자리 이내로 입력하세요.",
      "أدخل نسبة لكل مستلم من 0 إلى 100 وبحد أقصى أربع خانات عشرية.",
    ],
    percentage_required: [
      "At least one recipient percentage must be greater than zero.",
      "하나 이상의 대상에 0보다 큰 비율을 입력하세요.",
      "يجب أن تكون نسبة مستلم واحد على الأقل أكبر من صفر.",
    ],
    percentage_exceeded: [
      "The total allocation percentage exceeds 100%.",
      "배분 비율 합계가 100%를 초과합니다.",
      "يتجاوز مجموع نسب التوزيع 100%.",
    ],
    month_required: [
      "Choose one charge-start month to run or save a review.",
      "검토 계산과 저장에는 과금 시작 월 하나를 선택해야 합니다.",
      "اختر شهر بدء رسوم واحداً لتشغيل المراجعة أو حفظها.",
    ],
    amount_unavailable: [
      "Choose an available source currency with a known basis amount.",
      "기준액이 확인되는 원천 통화를 선택하세요.",
      "اختر عملة مصدر متاحة بمبلغ أساس معلوم.",
    ],
    amount_partial: [
      "Missing source amounts prevent allocation. Inspect source evidence.",
      "원천 금액이 누락되어 배분할 수 없습니다. 수집 근거를 확인하세요.",
      "تمنع مبالغ المصدر المفقودة إجراء التوزيع. افحص دليل المصدر.",
    ],
    targets_required: [
      "No authorized review recipient access scopes are available.",
      "현재 권한에서 확인되는 검토 대상 접근 범위가 없습니다.",
      "لا تتوفر نطاقات وصول مستلمين مصرّح بها للمراجعة.",
    ],
    target_limit: [
      "More than 100 recipient scopes require the server allocation service. The complete set was not truncated.",
      "대상이 100개를 초과하여 서버 배분 서비스 연결이 필요합니다. 대상 집합을 축소하지 않았습니다.",
      "تتطلب نطاقات المستلمين التي تتجاوز 100 خدمة توزيع الخادم. لم تُختصر المجموعة الكاملة.",
    ],
    target_manifest: [
      "The recipient set does not match the current source. Start a new query.",
      "대상 집합이 현재 원천과 맞지 않습니다. 새로 조회하세요.",
      "لا تطابق مجموعة المستلمين المصدر الحالي. ابدأ استعلاماً جديداً.",
    ],
    target_duplicate: [
      "Duplicate recipient identities prevent a complete review set.",
      "대상 식별자가 중복되어 검토 대상 집합을 확정할 수 없습니다.",
      "تمنع هويات المستلمين المكررة تكوين مجموعة مراجعة كاملة.",
    ],
    restricted_source: [
      "Account, search or page restrictions cannot define the full allocation source.",
      "계정·검색·페이지 조건이 있는 조회를 전체 배분 원천으로 사용할 수 없습니다.",
      "لا يمكن لقيود الحساب أو البحث أو الصفحة تحديد مصدر التوزيع الكامل.",
    ],
    snapshot_changed: [
      "The source reference changed. Start a new query to review the current data.",
      "원천 조회 기준이 변경되었습니다. 새로 조회하여 현재 데이터를 확인하세요.",
      "تغيّر مرجع المصدر. ابدأ استعلاماً جديداً لمراجعة البيانات الحالية.",
    ],
    response_invalid: [
      "The source response could not be validated. Retry the query.",
      "원천 응답을 확인할 수 없습니다. 다시 조회하세요.",
      "تعذّر التحقّق من استجابة المصدر. أعد الاستعلام.",
    ],
    forbidden: [
      "Access to this source is unavailable. Previous values have been cleared.",
      "이 원천에 접근할 수 없습니다. 이전 자료는 제거되었습니다.",
      "الوصول إلى هذا المصدر غير متاح. أُزيلت القيم السابقة.",
    ],
  };
  return labels[code]
    ? t(l, ...labels[code])
    : t(
        l,
        "The source query could not be completed. Retry or hand off the request reference.",
        "원천 조회를 완료하지 못했습니다. 다시 조회하거나 요청 참조를 담당자에게 전달하세요.",
        "تعذّر إكمال استعلام المصدر. أعد المحاولة أو سلّم مرجع الطلب إلى المسؤول.",
      );
}
