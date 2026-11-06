import type { Locale } from "../../../../shared/types";
import type { FieldName, FieldState } from "./contract";
export const t = (l: Locale, en: string, ko: string, ar: string) =>
  l === "ko" ? ko : l === "ar" ? ar : en;
export const fieldLabel = (l: Locale, k: FieldName) =>
  ({
    publisher: t(l, "Publisher", "상품 제공사", "ناشر المنتج"),
    issuer: t(l, "Invoice issuer", "청구서 발행사", "جهة إصدار الفاتورة"),
    description: t(l, "Charge description", "청구 설명", "وصف الرسوم"),
    category: t(l, "Charge category", "청구 분류", "فئة الرسوم"),
    charge_class: t(l, "Charge class", "청구 구분", "تصنيف الرسوم"),
    frequency: t(l, "Charge frequency", "청구 주기", "وتيرة الرسوم"),
    sku: t(l, "Source SKU ID", "원천 SKU ID", "معرّف SKU من المصدر"),
    commitment: t(
      l,
      "Source commitment reference",
      "원천 약정 참조",
      "مرجع الالتزام من المصدر",
    ),
  })[k];
export const stateLabel = (l: Locale, s: FieldState) =>
  ({
    value: t(l, "Observed value", "관측 값", "قيمة مرصودة"),
    source_null: t(l, "Source null", "원천 null", "قيمة مصدر فارغة"),
    absent: t(l, "Field absent", "필드 없음", "الحقل غائب"),
    blank: t(l, "Blank value", "공백 값", "قيمة خالية"),
    unmapped_version: t(
      l,
      "Version not mapped",
      "버전 미매핑",
      "الإصدار غير مربوط",
    ),
    invalid_type: t(
      l,
      "Invalid source type",
      "원천 형식 오류",
      "نوع مصدر غير صالح",
    ),
  })[s];
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

export const tabLabel = (l: Locale, k: string) =>
  ({
    buyer: t(
      l,
      "Buyer charge evidence",
      "구매 측 청구 근거",
      "أدلة رسوم الشراء",
    ),
    seller: t(l, "Seller settlement", "판매자 정산", "تسوية البائع"),
    credits: t(
      l,
      "Benefits and credits",
      "혜택·크레딧",
      "المنافع والأرصدة الدائنة",
    ),
    mapping: t(
      l,
      "Transaction mapping evidence",
      "거래 연결 근거",
      "أدلة ربط المعاملات",
    ),
  })[k as "buyer" | "seller" | "credits" | "mapping"];
