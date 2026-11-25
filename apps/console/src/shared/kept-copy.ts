import type { Locale } from "./types";
import { text as t } from "./copy";

export const kept = {
  shown: (l: Locale, n: number, total: number) =>
    t(l, `${n} of ${total} closed proposals shown, newest first`, `종료된 제안 ${total}건 중 ${n}건 표시, 최신순`, `المقترحات المغلقة المعروضة: ${n} من أصل ${total}، الأحدث أولًا`),
  all: (l: Locale, total: number) => t(l, `All ${total} closed proposals shown, newest first`, `종료된 제안 ${total}건 모두 표시, 최신순`, `المقترحات المغلقة المعروضة: جميعها (${total})، الأحدث أولًا`),
  more: (l: Locale) => t(l, "Load older closed proposals", "이전 종료된 제안 불러오기", "تحميل مقترحات مغلقة أقدم"),
  loading: (l: Locale) => t(l, "Loading…", "불러오는 중…", "جارٍ التحميل…"),
  failed: (l: Locale) =>
    t(l, "Older closed proposals could not be loaded. The proposals already shown are unchanged.", "이전 종료된 제안을 불러오지 못했습니다. 이미 표시된 제안은 그대로입니다.", "تعذّر تحميل المقترحات المغلقة الأقدم. لم تتغيّر المقترحات المعروضة."),
  tryAgain: (l: Locale) => t(l, "Try again", "다시 시도", "إعادة المحاولة"),
  end: (l: Locale, n: number) =>
    t(
      l,
      `Older closed proposals are not kept. Once more than ${n} proposals exist, the service drops the oldest closed ones; proposals waiting for approval are always kept.`,
      `더 오래된 종료된 제안은 보관되지 않습니다. 제안이 ${n}건을 넘으면 서비스가 가장 오래된 종료된 제안부터 삭제하며, 승인 대기 제안은 항상 보관합니다.`,
      `لا تُحفظ المقترحات المغلقة الأقدم. عند تجاوز عدد المقترحات ${n}، تحذف الخدمة أقدم المقترحات المغلقة، وتبقى المقترحات بانتظار الاعتماد محفوظة دائمًا.`,
    ),
  reset: (l: Locale) => t(l, "The list was refreshed and starts again from the newest proposal.", "목록을 새로 고쳐 최신 제안부터 다시 표시합니다.", "حُدِّثت القائمة وتبدأ من جديد بأحدث مقترح."),
};
