import type { Locale } from "../../../../shared/types";

const t = (l: Locale, en: string, ko: string, ar: string) => (l === "ko" ? ko : l === "ar" ? ar : en);

export const tl = {
  searchLabel: (l: Locale) => t(l, "Change ID", "변경 ID", "معرّف التغيير"),
  searchHint: (l: Locale) => t(l, "Part of the change identifier.", "변경 ID의 일부입니다.", "جزء من معرّف التغيير."),
  filters: (l: Locale) => t(l, "Filter trust changes", "신뢰 변경 필터", "تصفية تغييرات الثقة"),
  spansPages: (l: Locale) => t(l, "Newest first. The open changes fill more than one page, so they are not sorted by state.",
    "최신순입니다. 진행 중인 변경이 한 페이지를 넘어 상태순으로 정렬하지 않습니다.", "الأحدث أولًا. تتجاوز التغييرات المفتوحة صفحة واحدة، لذا لا تُرتَّب حسب الحالة."),
  evidenceOnOpen: (l: Locale) => t(l, "Whether a change's evidence is still current is checked when you open it.", "변경의 근거가 아직 유효한지는 변경을 열 때 확인합니다.",
    "يُتحقَّق مما إذا كانت أدلة التغيير لا تزال سارية عند فتحه."),
  noOpenMatch: (l: Locale) => t(l, "No open change matches the filters.", "필터와 일치하는 진행 중인 변경이 없습니다.", "لا يوجد تغيير مفتوح يطابق عوامل التصفية."),
  noClosedMatch: (l: Locale) => t(l, "No closed change matches the filters.", "필터와 일치하는 종료된 변경이 없습니다.", "لا يوجد تغيير مغلق يطابق عوامل التصفية."),
  openFailed: (l: Locale) => t(l, "Open changes could not be loaded", "진행 중인 변경을 불러오지 못했습니다", "تعذّر تحميل التغييرات المفتوحة"),
  closedFailed: (l: Locale) => t(l, "Closed changes could not be loaded", "종료된 변경을 불러오지 못했습니다", "تعذّر تحميل التغييرات المغلقة"),
};
