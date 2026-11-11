import type { Locale } from "../../../shared/types";
import { text as t } from "../../../shared/copy";
import {
  instantParts,
  type MappedField,
  type Origin,
  type ProvenanceClass,
} from "./fields";
export type DisplayField =
  | MappedField
  | "charge_month"
  | "id"
  | "account"
  | "batch"
  | "ordinal"
  | "evidence_class";
type Copy = [string, string, string];
const say = (l: Locale, copy: Copy) => t(l, copy[0], copy[1], copy[2]);
const fieldCopy: Record<DisplayField, Copy> = {
  provider: ["Provider", "공급사", "المزود"],
  service: ["Service", "서비스", "الخدمة"],
  currency: ["Billing currency", "청구 통화", "عملة الفوترة"],
  billing_account: ["Billing account ID", "결제 계정 ID", "معرّف حساب الفوترة"],
  sub_account: ["Subaccount ID", "하위 계정 ID", "معرّف الحساب الفرعي"],
  consumed_unit: ["Consumed unit", "사용 단위", "وحدة الاستهلاك"],
  pricing_unit: ["Pricing unit", "요금 산정 단위", "وحدة التسعير"],
  billing_start: [
    "Billing period start",
    "청구 기간 시작",
    "بداية فترة الفوترة",
  ],
  billing_end: ["Billing period end", "청구 기간 종료", "نهاية فترة الفوترة"],
  charge_start: ["Charge period start", "과금 기간 시작", "بداية فترة الرسوم"],
  charge_end: ["Charge period end", "과금 기간 종료", "نهاية فترة الرسوم"],
  billed_cost: ["Billed cost", "청구 비용", "التكلفة المفوترة"],
  effective_cost: ["Effective cost", "실효 비용", "التكلفة الفعلية"],
  list_cost: ["List cost", "정가 비용", "تكلفة القائمة"],
  contracted_cost: ["Contracted cost", "계약 비용", "تكلفة العقد"],
  consumed_quantity: ["Consumed quantity", "사용 수량", "كمية الاستهلاك"],
  pricing_quantity: ["Pricing quantity", "요금 산정 수량", "كمية التسعير"],
  list_unit_price: ["List unit price", "정가 단가", "سعر القائمة"],
  contracted_unit_price: ["Contracted unit price", "계약 단가", "سعر العقد"],
  charge_month: ["Charge-start month", "과금 시작 월", "شهر بدء الرسوم"],
  id: ["Row ID", "행 ID", "معرّف الصف"],
  account: ["Account scope ID", "계정 범위 ID", "معرّف نطاق الحساب"],
  batch: ["Delivery ID", "수집본 ID", "معرّف التسليم"],
  ordinal: [
    "Row ordinal in delivery",
    "수집본 내 행 순번",
    "ترتيب الصف في التسليم",
  ],
  evidence_class: ["Sample provenance", "샘플 출처", "منشأ العينة"],
};
export const fieldLabel = (l: Locale, field: DisplayField) =>
  say(l, fieldCopy[field]);
const originCopy: Record<Origin, { label: Copy; meaning: Copy }> = {
  source: {
    label: ["From source columns", "원천 열 값", "من أعمدة المصدر"],
    meaning: [
      "Read from the named FOCUS column through the version mapping. Provider conformance is not verified.",
      "버전 매핑에 따라 해당 FOCUS 열에서 읽은 값입니다. 공급사의 규격 준수 여부는 검증하지 않았습니다.",
      "قيم مقروءة من عمود FOCUS المسمّى عبر ربط الإصدار. لم يُتحقق من التزام المزود بالمواصفة.",
    ],
  },
  derived: {
    label: [
      "Derived by Hybrid Billing",
      "Hybrid Billing 산출 값",
      "قيم يشتقها Hybrid Billing",
    ],
    meaning: [
      "Computed by Hybrid Billing; no FOCUS column holds these values.",
      "Hybrid Billing이 산출한 값이며 대응하는 FOCUS 열이 없습니다.",
      "قيم يحسبها Hybrid Billing ولا يقابلها عمود FOCUS.",
    ],
  },
  lineage: {
    label: ["Delivery lineage", "수집본 식별 정보", "بيانات تتبّع التسليم"],
    meaning: [
      "Identifies the collected delivery and the row position inside it.",
      "행이 속한 수집본과 그 안의 행 위치를 식별합니다.",
      "يحدّد التسليم المجمَّع وموضع الصف داخله.",
    ],
  },
  annotation: {
    label: ["Sample annotation", "샘플 주석", "وسم العينة"],
    meaning: [
      "Annotation column carried in the collected sample row; not a FOCUS column.",
      "수집된 샘플 행에 포함된 주석 열이며 FOCUS 열이 아닙니다.",
      "عمود وسم ضمن صف العينة المجمَّع وليس عمود FOCUS.",
    ],
  },
};
export const originLabel = (l: Locale, origin: Origin) =>
  say(l, originCopy[origin].label);
export const originMeaning = (l: Locale, origin: Origin) =>
  say(l, originCopy[origin].meaning);
const handlingCopy = {
  text: [
    "Text shown as collected.",
    "수집된 문자열을 그대로 표시합니다.",
    "نص يُعرض كما جُمع.",
  ],
  number: [
    "Exact decimal to 18 places. Missing and exact zero are labeled separately.",
    "소수 18자리까지 정확한 값입니다. 누락과 정확한 0을 구분해 표시합니다.",
    "قيمة عشرية دقيقة حتى 18 منزلة. يُميَّز المفقود عن الصفر الدقيق.",
  ],
  start: [
    "Time normalized to UTC. Inclusive start by FOCUS definition.",
    "UTC로 정규화한 시각입니다. FOCUS 정의상 시작 시각을 포함합니다.",
    "وقت محوَّل إلى UTC. البداية مشمولة وفق تعريف FOCUS.",
  ],
  end: [
    "Time normalized to UTC. Exclusive end by FOCUS definition.",
    "UTC로 정규화한 시각입니다. FOCUS 정의상 종료 시각을 포함하지 않습니다.",
    "وقت محوَّل إلى UTC. النهاية مستثناة وفق تعريف FOCUS.",
  ],
  charge_month: [
    "Derived at query time from the charge period start in the tenant time zone, without proration.",
    "조회 시점에 과금 기간 시작 시각을 테넌트 시간대로 바꿔 산출하며 기간 배분은 하지 않습니다.",
    "يُشتق عند الاستعلام من بداية فترة الرسوم بحسب المنطقة الزمنية للمستأجر، دون توزيع زمني.",
  ],
  id: [
    "Derived from the delivery, row ordinal and tenant. Not a provider identifier.",
    "수집본·행 순번·테넌트로 산출한 식별자이며 공급사 식별자가 아닙니다.",
    "مشتق من التسليم وترتيب الصف والمستأجر. ليس معرّفاً من المزود.",
  ],
  account: [
    "Derived from tenant, provider, billing account and subaccount.",
    "테넌트·공급사·결제 계정·하위 계정으로 산출한 식별자입니다.",
    "مشتق من المستأجر والمزود وحساب الفوترة والحساب الفرعي.",
  ],
  batch: [
    "Identifier of the collected delivery that contains the row.",
    "행이 속한 수집본의 식별자입니다.",
    "معرّف التسليم المجمَّع الذي يحتوي الصف.",
  ],
  ordinal: [
    "Position of the row within that delivery.",
    "해당 수집본 안에서의 행 위치입니다.",
    "موضع الصف داخل ذلك التسليم.",
  ],
  evidence_class: [
    "An absent or unrecognized annotation is shown as Unclassified.",
    "주석이 없거나 인식할 수 없으면 출처 미분류로 표시합니다.",
    "يُعرض الوسم الغائب أو غير المعروف على أنه غير مصنف.",
  ],
} satisfies Record<string, Copy>;
export function handling(l: Locale, field: DisplayField) {
  if (field in handlingCopy)
    return say(l, handlingCopy[field as keyof typeof handlingCopy]);
  if (field.endsWith("_start")) return say(l, handlingCopy.start);
  if (field.endsWith("_end")) return say(l, handlingCopy.end);
  if (/cost|quantity|price/.test(field)) return say(l, handlingCopy.number);
  return say(l, handlingCopy.text);
}
const provenanceCopy: Record<ProvenanceClass, Copy> = {
  "official-public": ["Public sample", "공개 샘플", "عينة عامة"],
  "official-public-subset": [
    "Public sample subset",
    "공개 샘플 일부",
    "جزء من عينة عامة",
  ],
  "synthetic-contract-fixture": [
    "Synthetic test data",
    "합성 시험 자료",
    "بيانات اختبار اصطناعية",
  ],
  unclassified: ["Unclassified", "출처 미분류", "غير مصنف"],
};
export const provenanceLabel = (l: Locale, value: ProvenanceClass) =>
  say(l, provenanceCopy[value]);
export const missingLabel = (l: Locale) => t(l, "Missing", "누락", "مفقود");
export const unitMissingLabel = (l: Locale) =>
  t(l, "Unit missing", "단위 누락", "الوحدة مفقودة");
export const missingCount = (l: Locale, count: number) =>
  t(l, `${count} missing`, `누락 ${count}개`, `المفقود: ${count}`);
export const selectedLabel = (l: Locale) =>
  t(l, "Showing details", "상세 표시 중", "التفاصيل معروضة");
export const zeroLabel = (l: Locale) =>
  t(l, "Exact zero", "정확한 0", "صفر دقيق");
export function Instant({ value, full }: { value: string; full?: boolean }) {
  const parts = instantParts(value);
  return (
    <bdi dir="ltr" className="collected-token">
      {parts && !full ? `${parts.date} ${parts.time}` : value}
    </bdi>
  );
}
