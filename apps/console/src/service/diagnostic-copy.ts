import type { Locale } from "../shared/types";

const titles = {
  installation: { en: "Installation data could not be loaded.", ko: "설치 정보를 불러오지 못했습니다.", ar: "تعذّر تحميل بيانات التثبيت." },
  recovery: { en: "Recovery data could not be loaded.", ko: "복구 정보를 불러오지 못했습니다.", ar: "تعذّر تحميل بيانات الاسترداد." },
  contracts: { en: "Contracts could not be loaded.", ko: "계약 정보를 불러오지 못했습니다.", ar: "تعذّر تحميل العقود." },
  retention: { en: "Retention data could not be loaded.", ko: "보관 정보를 불러오지 못했습니다.", ar: "تعذّر تحميل بيانات الاحتفاظ." },
};
export const diagnosticCopy = {
  load: (l: Locale, area: keyof typeof titles) => titles[area][l],
  unavailable: (l: Locale) => ({ en: "The service is not responding. Try again shortly.", ko: "서비스가 응답하지 않습니다. 잠시 후 다시 시도해 주세요.", ar: "الخدمة لا تستجيب. أعد المحاولة بعد قليل." })[l],
  stale: (l: Locale) => ({ en: "The last reload failed. Everything on this screen, including the time, is from the last successful load. This screen is read-only until a reload succeeds.", ko: "마지막 새로 고침에 실패했습니다. 시각을 포함해 이 화면의 모든 내용은 마지막으로 불러온 시점 기준입니다. 새로 고침이 성공할 때까지 이 화면은 읽기 전용입니다.", ar: "فشلت آخر إعادة تحميل. كل ما يظهر في هذه الشاشة، بما فيه الوقت، يعود إلى آخر تحميل ناجح. هذه الشاشة للقراءة فقط حتى تنجح إعادة التحميل." })[l],
  kept: (l: Locale) => ({ en: "The last reload failed. Everything shown here is from the last successful load. Reload before acting on it.", ko: "마지막 새로 고침에 실패했습니다. 여기에 표시된 내용은 모두 마지막으로 불러온 시점 기준입니다. 이 내용을 바탕으로 작업하기 전에 새로 고침하세요.", ar: "فشلت آخر إعادة تحميل. كل ما يظهر هنا يعود إلى آخر تحميل ناجح. أعد التحميل قبل اتخاذ أي إجراء بناءً عليه." })[l],
  unconfirmed: (l: Locale) => ({ en: "The action could not be confirmed. Check the current state before trying again.", ko: "작업 결과를 확인하지 못했습니다. 현재 상태를 확인한 뒤 다시 시도하세요.", ar: "تعذّر تأكيد نتيجة الإجراء. تحقّق من الحالة الحالية قبل المحاولة مرة أخرى." })[l],
};
