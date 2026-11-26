import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { counted } from "../../../../shared/plural";

const iso = (id: string): string => `⁨${id}⁩`;

const minuteForms = { en: ["{n} minute", "{n} minutes"] as [string, string], ko: "{n}분", ar: ["{n} دقيقة", "دقيقة واحدة", "دقيقتان", "{n} دقائق", "{n} دقيقة", "{n} دقيقة"] as [string, string, string, string, string, string] };
const hourForms = { en: ["{n} hour", "{n} hours"] as [string, string], ko: "{n}시간", ar: ["{n} ساعة", "ساعة واحدة", "ساعتان", "{n} ساعات", "{n} ساعة", "{n} ساعة"] as [string, string, string, string, string, string] };
const dayForms = { en: ["{n} day", "{n} days"] as [string, string], ko: "{n}일", ar: ["{n} يوم", "يوم واحد", "يومان", "{n} أيام", "{n} يومًا", "{n} يوم"] as [string, string, string, string, string, string] };

export function span(l: Locale, minutes: number): string {
  if (minutes > 0 && minutes % 1440 === 0) return counted(l, minutes / 1440, dayForms);
  if (minutes > 0 && minutes % 60 === 0) return counted(l, minutes / 60, hourForms);
  return counted(l, minutes, minuteForms);
}

export const r = {
  rotate: (l: Locale) => t(l, "Rotate", "교체", "تدوير"),
  title: (l: Locale) => t(l, "Rotate credential", "자격 증명 교체", "تدوير بيانات الاعتماد"),
  lead: (l: Locale) =>
    t(
      l,
      "A new credential is issued, and the current one stays valid only for the time you set. The new secret is shown once.",
      "새 자격 증명을 발급하고, 기존 자격 증명은 정한 시간 동안만 유효하게 둡니다. 새 비밀값은 한 번만 보여 줍니다.",
      "تُصدَر بيانات اعتماد جديدة، وتبقى بيانات الاعتماد الحالية صالحة للمدة التي تحددها فقط. تُعرض القيمة السرية الجديدة مرة واحدة.",
    ),
  newCredential: (l: Locale) => t(l, "New credential", "새 자격 증명", "بيانات الاعتماد الجديدة"),
  current: (l: Locale) => t(l, "Current credential", "기존 자격 증명", "بيانات الاعتماد الحالية"),
  newValidFor: (l: Locale) => t(l, "New credential valid for", "새 자격 증명 유효 기간", "مدة صلاحية بيانات الاعتماد الجديدة"),
  overlap: (l: Locale) => t(l, "Keep the current credential valid for", "기존 자격 증명 유지 시간", "مدة بقاء بيانات الاعتماد الحالية صالحة"),
  minutes: (l: Locale) => t(l, "minutes", "분", "دقيقة"),
  overlapHint: (l: Locale, min: number, max: number, usual: number) =>
    t(
      l,
      `${min}–${max} minutes from now (up to ${span(l, max)}). Installation default: ${counted(l, usual, minuteForms)} (${span(l, usual)}).`,
      `지금부터 ${min}–${max}분(최대 ${span(l, max)}). 설치 기본값: ${usual}분(${span(l, usual)}).`,
      `من ${min} إلى ${max} دقيقة من الآن (حتى ${span(l, max)}). القيمة الافتراضية للتثبيت: ${counted(l, usual, minuteForms)} (${span(l, usual)}).`,
    ),
  errOverlap: (l: Locale, min: number, max: number) =>
    t(l, `Enter a whole number from ${min} to ${max}.`, `${min}–${max} 사이 정수를 입력하세요.`, `أدخل عددًا صحيحًا من ${min} إلى ${max}.`),
  oldEnds: (l: Locale, d: string) => t(l, `The current credential ends about ${d}, counted from when you confirm.`, `기존 자격 증명은 확인 시점부터 계산하여 ${d}경 종료됩니다.`, `تنتهي بيانات الاعتماد الحالية نحو ${d}، ويُحتسب ذلك من لحظة التأكيد.`),
  oldEndsSooner: (l: Locale, d: string) =>
    t(
      l,
      `The current credential already ends ${d}, sooner than that. Its end is not extended.`,
      `기존 자격 증명은 그보다 이른 ${d}에 이미 종료되므로 종료 시각을 늘리지 않습니다.`,
      `تنتهي بيانات الاعتماد الحالية أصلًا في ${d}، أي قبل ذلك، ولا يُمدَّد موعد انتهائها.`,
    ),
  reviewTitle: (l: Locale) => t(l, "Check the rotation", "교체 내용 확인", "تحقّق من التدوير"),
  willEnd: (l: Locale) => t(l, "Credential that will end", "종료될 자격 증명", "بيانات الاعتماد التي ستنتهي"),
  endsAt: (l: Locale) => t(l, "Ends at", "종료 시각", "موعد الانتهاء"),
  endsAbout: (l: Locale) => t(l, "Estimated end", "예상 종료 시각", "موعد الانتهاء التقريبي"),
  fromConfirm: (l: Locale) =>
    t(l, "Counted from the moment you confirm; the exact time is set then.", "확인하는 시점부터 계산하며, 정확한 시각은 그때 정해집니다.", "تُحتسب المدة من لحظة التأكيد، ويُحدَّد الوقت الدقيق حينها."),
  once: (l: Locale, n: number) =>
    t(
      l,
      `The new secret is shown once and must be collected within ${counted(l, n, minuteForms)} of the rotation.`,
      `새 비밀값은 한 번만 표시되며 교체 후 ${n}분 안에 수령해야 합니다.`,
      `تُعرض القيمة السرية الجديدة مرة واحدة ويجب استلامها خلال ${counted(l, n, minuteForms)} من التدوير.`,
    ),
  notMovedBack: (l: Locale) =>
    t(
      l,
      "This can't be undone: the current credential's original end is not restored.",
      "되돌릴 수 없습니다. 기존 자격 증명의 원래 종료 시각은 복원되지 않습니다.",
      "لا يمكن التراجع عن ذلك: لا يُستعاد موعد الانتهاء الأصلي لبيانات الاعتماد الحالية.",
    ),
  notKnown: (l: Locale) =>
    t(
      l,
      "This screen does not know where the credential is used and does not check work in progress. Put the new secret in place before the current credential ends.",
      "이 화면은 자격 증명이 어디에서 쓰이는지 알지 못하며 진행 중인 작업을 확인하지 않습니다. 기존 자격 증명이 종료되기 전에 새 비밀값을 적용하세요.",
      "لا تعرف هذه الشاشة أين تُستخدم بيانات الاعتماد ولا تتحقق من الأعمال الجارية. ضع القيمة السرية الجديدة موضع الاستخدام قبل انتهاء بيانات الاعتماد الحالية.",
    ),
  exposed: (l: Locale) =>
    t(
      l,
      "If the secret may have been exposed, revoke the credential instead of rotating it.",
      "비밀값이 노출되었을 수 있다면 교체하지 말고 자격 증명을 폐기하세요.",
      "إن اشتُبه في كشف القيمة السرية فأبطِل بيانات الاعتماد بدلًا من تدويرها.",
    ),
  review: (l: Locale) => t(l, "Review rotation", "교체 내용 검토", "مراجعة التدوير"),
  edit: (l: Locale) => t(l, "Back to edit", "수정으로 돌아가기", "العودة إلى التعديل"),
  cancel: (l: Locale) => t(l, "Cancel rotation", "교체 취소", "إلغاء التدوير"),
  gone: (l: Locale, id: string) =>
    t(
      l,
      `${iso(id)} can no longer be rotated: it has ended, or a rotation already fixed its end.`,
      `${iso(id)}은(는) 더 이상 교체할 수 없습니다. 이미 종료되었거나 이전 교체로 종료 시각이 확정되었습니다.`,
      `لم يعد بالإمكان تدوير ${iso(id)}: إما أنها انتهت أو أن تدويرًا سابقًا حدّد موعد انتهائها.`,
    ),
  sending: (l: Locale, service: string) => t(l, `Rotating a credential of ${iso(service)}…`, `${iso(service)} 자격 증명 교체 중…`, `جارٍ تدوير بيانات اعتماد لـ ${iso(service)}…`),
  unknown: (l: Locale, service: string): [string, string] =>
    l === "ko"
      ? [`${iso(service)} 교체 결과 확인 안 됨.`, "서버가 이미 새 자격 증명을 발급하고 기존 자격 증명의 종료 시각을 확정했을 수 있습니다."]
      : l === "ar"
        ? [`تدوير لـ ${iso(service)}: النتيجة غير معروفة.`, "ربما أصدر الخادم بيانات الاعتماد الجديدة وحدّد موعد انتهاء الحالية بالفعل."]
        : [`Rotation for ${iso(service)}: outcome unknown.`, "The server may already have issued the new credential and fixed the end of the current one."],
  lapsed: (l: Locale, service: string): [string, string] =>
    l === "ko"
      ? [`${iso(service)} 교체 결과를 더 이상 확인할 수 없습니다.`, "목록에서 기존 자격 증명의 종료 시각을 확인하고, '비밀값 수령 안 됨' 자격 증명이 있으면 폐기한 뒤 새로 발급하세요."]
      : l === "ar"
        ? [`تدوير لـ ${iso(service)}: لم يعد بالإمكان التحقق من النتيجة.`, "تحقّق في القائمة من موعد انتهاء بيانات الاعتماد الحالية، وإن ظهرت عبارة «لم يُستلَم السر» فأبطِل تلك البيانات وأصدِر غيرها."]
        : [`Rotation for ${iso(service)}: the outcome can no longer be checked.`, "Check the end of the current credential in the list. If a credential shows \"Secret never collected\", revoke it and issue another."],
  held: (l: Locale) =>
    t(
      l,
      "The outcome of this request is unknown. Try again sends the same request. If the server already rotated the credential, you get the new credential and its secret, not a second rotation.",
      "이 요청의 결과를 확인하지 못했습니다. 다시 시도하면 같은 요청을 보냅니다. 서버가 이미 교체했다면 두 번째 교체가 아니라 그 새 자격 증명과 비밀값을 받습니다.",
      "نتيجة هذا الطلب غير معروفة. تؤدي إعادة المحاولة إلى إرسال الطلب نفسه. إن كان الخادم قد دوّر بيانات الاعتماد فستحصل على البيانات الجديدة وسرّها، لا على تدوير ثانٍ.",
    ),
  rotated: (l: Locale, credential: string, old: string, service: string) =>
    t(
      l,
      `Credential ${iso(credential)} was issued for ${iso(service)} to replace ${iso(old)}. Its secret has not been shown yet.`,
      `${iso(service)}에 ${iso(old)}을(를) 대체할 자격 증명 ${iso(credential)}이(가) 발급되었습니다. 비밀값은 아직 표시되지 않았습니다.`,
      `صدرت بيانات الاعتماد ${iso(credential)} لـ ${iso(service)} لتحلّ محل ${iso(old)}. لم يُعرض سرّها بعد.`,
    ),
  open: (l: Locale) => t(l, "Open rotation form", "교체 양식 열기", "فتح نموذج التدوير"),
  ending: (l: Locale, d: string) => t(l, `Rotated · ends ${d}`, `교체됨 · ${d} 종료`, `دُوِّرت · تنتهي في ${d}`),
  replacedBy: (l: Locale): [string, string] => (l === "ko" ? ["", "(으)로 대체됨"] : l === "ar" ? ["حلّت محلها ", ""] : ["Replaced by ", ""]),
  replaces: (l: Locale): [string, string] => (l === "ko" ? ["", "을(를) 대체"] : l === "ar" ? ["تحلّ محل ", ""] : ["Replaces ", ""]),
};

export const countDays = (l: Locale, n: number): string => counted(l, n, dayForms);
export const countMinutes = (l: Locale, n: number): string => counted(l, n, minuteForms);
