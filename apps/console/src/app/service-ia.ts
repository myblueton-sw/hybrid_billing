import type { Tr } from "../planning/copy";
export type ServiceSection = {
  key: string;
  label: Tr;
  screens: readonly string[];
};
export const businessSections: readonly ServiceSection[] = [
  {
    key: "overview",
    label: ["통합 현황", "Overview", "النظرة العامة"],
    screens: ["U01", "U02", "U03", "U04"],
  },
  {
    key: "costs",
    label: ["비용·사용량", "Costs and usage", "التكاليف والاستخدام"],
    screens: ["S04", "B01", "B02", "B06", "B07"],
  },
  {
    key: "organizations",
    label: ["조직·계약", "Organizations and contracts", "المؤسسات والعقود"],
    screens: ["O01", "O02", "O03"],
  },
  {
    key: "policies",
    label: ["배분·요금 정책", "Allocation and pricing", "التوزيع والتسعير"],
    screens: ["B03", "B04", "B05"],
  },
  {
    key: "billing",
    label: ["정산·청구", "Settlement and billing", "التسوية والفوترة"],
    screens: ["B08", "B09", "B10", "I01", "B11", "I04"],
  },
  {
    key: "accounting",
    label: ["회계·ERP", "Accounting and ERP", "المحاسبة وERP"],
    screens: ["I02", "I03"],
  },
  {
    key: "data",
    label: ["데이터·연결", "Data and connections", "البيانات والاتصالات"],
    screens: ["S01", "S02", "S03", "S05", "S06"],
  },
  {
    key: "reports",
    label: ["리포트·감사", "Reports and audit", "التقارير والتدقيق"],
    screens: ["R01", "R02"],
  },
];
export const administrationSection: ServiceSection = {
  key: "administration",
  label: ["관리 설정", "Administration", "الإدارة"],
  screens: [
    "M01",
    "M02",
    "M03",
    "M04",
    "P01",
    "P02",
    "P03",
    "P04",
    "P05",
    "P06",
    "P07",
  ],
};
export const outsideBusinessNavigation = [
  "A01",
  "M05",
  "C01",
  "C02",
  "C03",
  "L01",
  "L02",
  "L03",
] as const;
export function sectionFor(screen: string) {
  return [...businessSections, administrationSection].find((section) =>
    section.screens.includes(screen),
  );
}
export const navigationLabels: Record<string, Tr> = {
  U01: ["전체 현황", "Billing overview", "نظرة عامة على الفوترة"],
  U02: ["마감 일정", "Closing calendar", "تقويم الإقفال"],
  U03: ["내 작업·알림", "My work and notifications", "أعمالي والإشعارات"],
  U04: [
    "확인 필요 사항",
    "Items requiring attention",
    "عناصر تحتاج إلى مراجعة",
  ],
  S04: [
    "통합 비용·사용량 명세",
    "Consolidated cost and usage",
    "تفاصيل التكاليف والاستخدام الموحدة",
  ],
  B01: ["온프렘 원가 풀", "On-premise cost pools", "مجموعات التكلفة المحلية"],
  B03: [
    "내부 배분 정책",
    "Internal allocation policies",
    "سياسات التوزيع الداخلي",
  ],
  B04: ["판매 요금 정책", "Sales pricing policies", "سياسات تسعير المبيعات"],
  B05: [
    "환율·반올림 정책",
    "Exchange rates and rounding",
    "أسعار الصرف والتقريب",
  ],
  I01: [
    "청구서·내부 명세",
    "Invoices and internal statements",
    "الفواتير والكشوف الداخلية",
  ],
  I02: [
    "월별 인식·ERP 제출·대사",
    "Monthly recognition and ERP reconciliation",
    "الإثبات الشهري وتقديم ERP والمطابقة",
  ],
  I03: [
    "파일 인계·반영 확인",
    "File handoff and confirmation",
    "تسليم الملفات وتأكيد المعالجة",
  ],
  S01: ["원천 연결", "Source connections", "اتصالات المصادر"],
  S02: [
    "연결 개통·담당자 인계",
    "Connection onboarding and handoff",
    "تهيئة الاتصال وتسليم المسؤولية",
  ],
  S03: [
    "수집·오류·완전성",
    "Intake, errors and completeness",
    "الاستيراد والأخطاء والاكتمال",
  ],
  S06: [
    "원천 대사·입력 확정",
    "Source reconciliation and input confirmation",
    "مطابقة المصدر وتأكيد الإدخال",
  ],
};
