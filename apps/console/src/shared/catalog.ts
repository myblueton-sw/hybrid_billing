import catalog from "./catalog.json";
export const { screens, groups } = catalog;
export const domains = [
  {
    id: "overview",
    groups: ["overview"],
    label: { en: "Overview", ko: "전체 현황", ar: "نظرة عامة" },
  },
  {
    id: "work",
    groups: ["work", "issue", "history"],
    label: { en: "Billing operations", ko: "청구 운영", ar: "عمليات الفوترة" },
  },
  {
    id: "sources",
    groups: ["sources"],
    label: { en: "Sources & evidence", ko: "원본·증빙", ar: "المصادر والأدلة" },
  },
  {
    id: "contracts",
    groups: ["contracts", "reconcile"],
    label: {
      en: "Pricing & calculation",
      ko: "요율·계산",
      ar: "التسعير والحساب",
    },
  },
  {
    id: "customer",
    groups: ["customer"],
    label: { en: "Customer portal", ko: "고객 포털", ar: "بوابة العملاء" },
  },
  {
    id: "settings",
    groups: ["settings", "access", "onboarding"],
    label: { en: "Administration", ko: "관리 설정", ar: "الإدارة" },
  },
];
