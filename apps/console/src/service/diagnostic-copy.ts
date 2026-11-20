import type { Locale } from "../shared/types";

const titles = {
  installation: { en: "Installation data could not be loaded.", ko: "설치 정보를 불러오지 못했습니다.", ar: "تعذّر تحميل بيانات التثبيت." },
  recovery: { en: "Recovery data could not be loaded.", ko: "복구 정보를 불러오지 못했습니다.", ar: "تعذّر تحميل بيانات الاسترداد." },
  retention: { en: "Retention data could not be loaded.", ko: "보관 정보를 불러오지 못했습니다.", ar: "تعذّر تحميل بيانات الاحتفاظ." },
};
export const diagnosticCopy = {
  load: (l: Locale, area: keyof typeof titles) => titles[area][l],
  unavailable: (l: Locale) => ({ en: "The service is not responding. Try again shortly.", ko: "서비스가 응답하지 않습니다. 잠시 후 다시 시도해 주세요.", ar: "الخدمة لا تستجيب. أعد المحاولة بعد قليل." })[l],
  unconfirmed: (l: Locale) => ({ en: "The action could not be confirmed. Check the current state before trying again.", ko: "작업 결과를 확인하지 못했습니다. 현재 상태를 확인한 뒤 다시 시도하세요.", ar: "تعذّر تأكيد نتيجة الإجراء. تحقّق من الحالة الحالية قبل المحاولة مرة أخرى." })[l],
};
