import type { Locale } from "../../../../../shared/types";
import type { SetupPart } from "./scope";

const words = {
 nav: ["Issuance setup sections", "발행 설정 하위 메뉴", "أقسام إعداد الإصدار"],
 settings: ["Issuance settings", "발행 설정", "إعدادات الإصدار"],
 rules: ["Protected settings rules", "보호된 설정 규칙", "قواعد الإعدادات المحمية"],
 channels: ["Verification channels", "검증 채널", "قنوات التحقق"],
 changes: ["Protected settings changes", "보호된 설정 변경", "تغييرات الإعدادات المحمية"],
 statements: ["Statements", "진술", "الإفادات"],
 action: ["Action", "작업", "الإجراء"],
 use: ["Use these settings", "이 설정 사용", "استخدام هذه الإعدادات"],
 selected: ["Selected", "선택됨", "محددة"],
 scope: ["Selected issuance settings", "선택한 발행 설정", "إعدادات الإصدار المحددة"],
 scopeNote: ["Protected settings and payment terms are recorded for these confirmed settings. The selection is kept for this contract and State-on date until the browser reloads the page.", "보호된 설정과 납부 조건은 확인 완료된 이 설정을 기준으로 기록됩니다. 선택은 이 계약과 상태 조회 기준일에 대해, 브라우저에서 페이지를 다시 불러오기 전까지 유지됩니다.", "تُسجَّل الإعدادات المحمية وشروط الدفع لهذه الإعدادات المؤكدة. يبقى التحديد لهذا العقد وتاريخ الحالة هذا إلى أن يُعاد تحميل الصفحة في المتصفح."],
 setupDate: ["Setup date", "설정 기준일", "تاريخ الإعداد"],
 noScope: ["Select confirmed issuance settings first. Protected settings and payment terms are recorded for one set of confirmed settings.", "확인 완료된 발행 설정을 먼저 선택하세요. 보호된 설정과 납부 조건은 확인 완료된 설정 하나를 기준으로 기록됩니다.", "حدِّد أولاً إعدادات إصدار مؤكدة. تُسجَّل الإعدادات المحمية وشروط الدفع لمجموعة واحدة من الإعدادات المؤكدة."],
 toSettings: ["Go to issuance settings", "발행 설정으로 이동", "الانتقال إلى إعدادات الإصدار"],
} as const;
export type PartWord = keyof typeof words;
export function partText(l: Locale, key: PartWord | SetupPart): string { return words[key][l === "ko" ? 1 : l === "ar" ? 2 : 0]; }
