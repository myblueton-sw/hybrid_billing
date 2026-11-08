import type { Locale, Copy } from "./types";
export const text = (locale: Locale, en: string, ko: string, ar: string) =>
  ({ en, ko, ar })[locale];
export const titles: Record<string, Copy> = {
  contents: {
    en: "Implementation workspace",
    ko: "프론트 구현 작업실",
    ar: "مساحة تنفيذ الواجهة",
  },
  "index-intake": {
    en: "Supplier evidence intake",
    ko: "공급자 원본 수집",
    ar: "استلام أدلة المورد",
  },
};
