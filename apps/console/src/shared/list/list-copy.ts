import type { Locale } from "../types";
import type { Plural } from "../plural";

const t = (l: Locale, en: string, ko: string, ar: string) => (l === "ko" ? ko : l === "ar" ? ar : en);

export const lc = {
  exact: (l: Locale) => t(l, "Matching count", "일치 항목 수", "عدد النتائج المطابقة"),
  lower: (l: Locale) => t(l, "Matching count, at least", "일치 항목 수, 최소", "عدد النتائج المطابقة، على الأقل"),
  reference: (l: Locale) => t(l, "List reference time", "목록 기준 시각", "وقت مرجع القائمة"),
  size: (l: Locale) => t(l, "Rows per page", "페이지당 항목 수", "صفوف لكل صفحة"),
  first: (l: Locale) => t(l, "First", "처음", "الأول"),
  previous: (l: Locale) => t(l, "Previous", "이전", "السابق"),
  next: (l: Locale) => t(l, "Next", "다음", "التالي"),
  applied: (l: Locale) => t(l, "Applied filters", "적용된 필터", "عوامل التصفية المطبّقة"),
  remove: (l: Locale) => t(l, "Remove filter", "필터 제거", "إزالة عامل التصفية"),
  clear: (l: Locale) => t(l, "Clear filters", "필터 모두 해제", "مسح جميع عوامل التصفية"),
  noMatch: (l: Locale) => t(l, "No matches for these filters.", "이 필터와 일치하는 항목이 없습니다.", "لا توجد نتائج مطابقة لعوامل التصفية هذه."),
  pages: (l: Locale, title: string) => t(l, `${title}, pages`, `${title} 페이지`, `صفحات ${title}`),
  onPage: (l: Locale, n: number) => t(l, `Rows on this page: ${n}`, `이 페이지의 항목: ${n}`, `الصفوف في هذه الصفحة: ${n}`),
  lowered: (l: Locale, n: number) =>
    t(l, `The installation limits rows per page to ${n}.`, `설치 설정에 따라 페이지당 항목 수가 ${n}개로 제한됩니다.`, `يحدّ إعداد التثبيت عدد الصفوف في الصفحة بـ ${n}.`),
  selected: (l: Locale) => t(l, "Selected", "선택됨", "محدد"),
  settingsTitle: (l: Locale) => t(l, "List settings", "목록 설정", "إعدادات القوائم"),
  settingsLead: (l: Locale) =>
    t(l, "Set by the installation; they apply to every list on this screen.", "설치 시 정해지며 이 화면의 모든 목록에 적용됩니다.", "يحدّدها التثبيت وتنطبق على جميع القوائم في هذه الشاشة."),
  colSetting: (l: Locale) => t(l, "Setting", "설정", "الإعداد"),
  colValue: (l: Locale) => t(l, "Value", "값", "القيمة"),
  colRange: (l: Locale) => t(l, "Allowed range", "허용 범위", "النطاق المسموح"),
  colSource: (l: Locale) => t(l, "Source", "출처", "المصدر"),
  range: (l: Locale, min: number, max: number, def: number) =>
    t(l, `${min}–${max}, default ${def}`, `${min}~${max}, 기본값 ${def}`, `من ${min} إلى ${max}، الافتراضي ${def}`),
  source: (l: Locale, source: string) =>
    source === "start_flag" ? t(l, "Set when the server started", "서버 시작 시 지정", "حُدّد عند بدء الخادم") : t(l, "Installation default", "설치 기본값", "الإعداد الافتراضي للتثبيت"),
  name: {
    service_query_page_size: (l: Locale) => t(l, "Most rows per page", "페이지당 최대 항목 수", "الحد الأقصى للصفوف في الصفحة"),
    service_query_scan: (l: Locale) => t(l, "Records checked per read, at most", "조회당 최대 확인 항목 수", "الحد الأقصى للسجلات المفحوصة في كل قراءة"),
    service_query_lifetime_seconds: (l: Locale) => t(l, "Page position stays valid for", "페이지 위치 유효 시간", "مدة صلاحية موضع الصفحة"),
  } as Record<string, (l: Locale) => string>,
  unit: {
    rows: { en: ["1 row", "{n} rows"], ko: "{n}행", ar: ["{n} صف", "صف واحد", "صفّان", "{n} صفوف", "{n} صفًا", "{n} صف"] },
    "work units": { en: ["1 record", "{n} records"], ko: "{n}건", ar: ["{n} سجل", "سجل واحد", "سجلّان", "{n} سجلات", "{n} سجلًا", "{n} سجل"] },
    seconds: { en: ["1 second", "{n} seconds"], ko: "{n}초", ar: ["{n} ثانية", "ثانية واحدة", "ثانيتان", "{n} ثوانٍ", "{n} ثانية", "{n} ثانية"] },
  } as Record<string, Plural>,
};
