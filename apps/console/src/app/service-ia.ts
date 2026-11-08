import type { Tr } from "../planning/copy";
export type ServiceSection = {
  key: string;
  label: Tr;
  screens: readonly string[];
  direct?: boolean;
};
export const businessSections: readonly ServiceSection[] = [
  {
    key: "overview",
    label: ["현황", "Overview", "النظرة العامة"],
    screens: ["U01"],
    direct: true,
  },
  {
    key: "costs",
    label: ["비용·배분", "Costs and allocation", "التكاليف والتوزيع"],
    screens: ["S04", "B01", "B03"],
  },
  {
    key: "settlement",
    label: ["정산", "Settlement", "التسوية"],
    screens: ["B08", "U02", "B09", "S06", "I02"],
  },
  {
    key: "billing",
    label: ["청구", "Billing", "الفوترة"],
    screens: ["I01", "B11"],
  },
  {
    key: "organizations",
    label: ["조직·계약", "Organizations and contracts", "المؤسسات والعقود"],
    screens: ["O01", "O03", "B04", "B05"],
  },
  {
    key: "reports",
    label: ["리포트", "Reports", "التقارير"],
    screens: ["R01"],
    direct: true,
  },
];
export const dataSection: ServiceSection = {
  key: "data",
  label: ["데이터 수집", "Data collection", "جمع البيانات"],
  screens: ["S03", "S01"],
};
export const administrationSection: ServiceSection = {
  key: "administration",
  label: ["관리 설정", "Administration", "الإدارة"],
  screens: ["M01", "P02", "P01"],
};
export const contextualGroups = [
  { parent: "U01", screens: ["U04"] },
  { parent: "S01", screens: ["S02"] },
  { parent: "S04", screens: ["B02", "B06", "B07", "S05"] },
  { parent: "B08", screens: ["B10"] },
  { parent: "I01", screens: ["I04"] },
  { parent: "I02", screens: ["I03"] },
  { parent: "O01", screens: ["O02"] },
  { parent: "R01", screens: ["R02"] },
  { parent: "M01", screens: ["M02", "M03", "M04"] },
  { parent: "P01", screens: ["P03", "P04", "P05", "P06", "P07"] },
] as const;
export const outsideBusinessNavigation = [
  "A01",
  "U03",
  "M05",
  "C01",
  "C02",
  "C03",
  "L01",
  "L02",
  "L03",
] as const;
export function parentFor(screen: string): string {
  return (
    contextualGroups.find((g) =>
      (g.screens as readonly string[]).includes(screen),
    )?.parent ?? screen
  );
}
export function sectionFor(screen: string) {
  const parent = parentFor(screen);
  return [...businessSections, dataSection, administrationSection].find((s) =>
    s.screens.includes(parent),
  );
}
export function relatedScreens(screen: string): readonly string[] {
  const group = contextualGroups.find((g) => g.parent === parentFor(screen));
  return group
    ? [
        group.parent,
        ...group.screens,
        ...(screen === "I01" ? ["I03"] : []),
      ].filter((id) => id !== screen && id !== "S05")
    : [];
}
export const navigationLabels: Record<string, Tr> = {
  B08: ["정산 실행", "Billing runs", "عمليات التسوية"],
  B09: ["승인함", "Approvals", "الموافقات"],
  R01: ["리포트", "Reports", "التقارير"],
  O01: ["조직", "Organizations", "المؤسسات"],
  O03: ["계약", "Contracts", "العقود"],
  U01: ["현황", "Overview", "النظرة العامة"],
  U02: ["마감 관리", "Closing management", "إدارة الإقفال"],
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
  B01: ["원가 풀", "Cost pools", "مجموعات التكلفة"],
  B03: ["배분 정책", "Allocation policies", "سياسات التوزيع"],
  B04: ["판매 요금 정책", "Sales pricing policies", "سياسات تسعير المبيعات"],
  B05: [
    "환율·반올림 정책",
    "Exchange rates and rounding",
    "أسعار الصرف والتقريب",
  ],
  B11: ["잔액·크레딧", "Balances and credits", "الأرصدة والائتمانات"],
  M01: ["사용자·접근", "Users and access", "المستخدمون والوصول"],
  P01: ["설치·운영", "Installation and operations", "التثبيت والتشغيل"],
  P02: ["연동 설정", "Integration settings", "إعدادات التكامل"],
  I01: [
    "청구서·내부 명세",
    "Invoices and internal statements",
    "الفواتير والكشوف الداخلية",
  ],
  I02: [
    "외부 제출·반영",
    "External submissions and results",
    "التقديم الخارجي ونتائج المعالجة",
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
  S03: ["수집 내역", "Received inputs", "المدخلات المستلمة"],
  S06: [
    "대사·차이 확인",
    "Reconciliation and differences",
    "المطابقة والتحقق من الفروق",
  ],
};
export function navigationScreenIds(
  section: ServiceSection,
  allowed: ReadonlySet<string>,
): string[] {
  return section.screens.flatMap((id) =>
    allowed.has(id)
      ? [id]
      : contextualGroups
          .filter((g) => g.parent === id)
          .flatMap((g) =>
            g.screens.filter((child) => child !== "S05" && allowed.has(child)),
          ),
  );
}
