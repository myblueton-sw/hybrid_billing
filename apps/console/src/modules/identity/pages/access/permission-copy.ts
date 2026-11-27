import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

export const p = {
  title: (l: Locale) => t(l, "Grants and delegations", "권한 부여·위임", "المنح والتفويضات"),
  colHolder: (l: Locale) => t(l, "Holder", "보유자", "صاحب الصلاحية"),
  colScope: (l: Locale) => t(l, "Scope", "범위", "النطاق"),
  colSource: (l: Locale) => t(l, "Source", "권한 출처", "المصدر"),
  colParent: (l: Locale) => t(l, "Original delegation", "원 위임", "التفويض الأصلي"),
  colValidity: (l: Locale) => t(l, "Validity", "유효 기간", "مدة الصلاحية"),
  colEffective: (l: Locale) => t(l, "Effective permission", "실효 권한", "الصلاحية الفعلية"),
  console: (l: Locale) => t(l, "Console", "콘솔", "وحدة التحكم"),
  delegatedBy: (l: Locale, who: string) => t(l, `Delegated by ${who}`, `${who}님이 위임`, `فوّضه ${who}`),
  grant: (l: Locale) => t(l, "Grant", "권한 부여", "منح"),
  none: (l: Locale) => t(l, "None", "없음", "لا يوجد"),
  from: (l: Locale, d: string) => t(l, `From ${d}`, `${d}부터`, `من ${d}`),
  setByInstallation: (l: Locale) => t(l, "Set by the installation", "설치 환경에서 설정", "يحدده التثبيت"),
  searchHint: (l: Locale) => t(l, "Grant or delegation ID, or holder user ID", "권한 부여·위임 ID 또는 보유자 사용자 ID", "معرّف المنح أو التفويض، أو معرّف مستخدم صاحب الصلاحية"),
  holderId: (l: Locale) => t(l, "Holder user ID", "보유자 사용자 ID", "معرّف مستخدم صاحب الصلاحية"),
  holderHint: (l: Locale) => t(l, "Exact user ID. Choosing a person by name is not available yet.", "정확한 사용자 ID. 이름으로 사용자를 고르는 기능은 아직 없습니다.", "معرّف المستخدم بدقة. اختيار شخص بالاسم غير متاح بعد."),
  customerId: (l: Locale) => t(l, "Customer ID", "고객 ID", "معرّف العميل"),
  legalEntityId: (l: Locale) => t(l, "Legal entity ID", "법인 ID", "معرّف الكيان القانوني"),
  endsBefore: (l: Locale) => t(l, "Ends before", "종료 기준일(이전)", "تنتهي قبل"),
  endsBeforeHint: (l: Locale) => t(l, "Date in UTC; ends before 00:00 of that day.", "UTC 기준 날짜이며, 그날 00:00 이전에 끝나는 항목입니다.", "تاريخ بتوقيت UTC؛ ما ينتهي قبل الساعة 00:00 من ذلك اليوم."),
  endedChain: (l: Locale) => t(l, "Ended through delegation chain", "위임 경로에서 종료됨", "انتهى عبر سلسلة التفويض"),
  invalidFilters: (l: Locale) => t(l, "These filters cannot be read. Reset them.", "이 필터를 읽을 수 없습니다. 초기화하세요.", "تعذّرت قراءة عوامل التصفية هذه. أعد تعيينها."),
};
