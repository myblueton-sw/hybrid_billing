import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { HistoryBrowser, HistoryEvent, HistorySystem } from "../../../../service/access-history";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };

export function plural(l: Locale, n: number, forms: Partial<Record<Intl.LDMLPluralRule, string>> & { other: string }): string {
  const rule = new Intl.PluralRules(tags[l]).select(n);
  return (forms[rule] ?? forms.other).replaceAll("{n}", String(n));
}

const days = (l: Locale, n: number) =>
  l === "en"
    ? plural(l, n, { one: "{n} day", other: "{n} days" })
    : l === "ko"
      ? `${n}일`
      : plural(l, n, { zero: "{n} يوم", one: "يوم واحد", two: "يومين", few: "{n} أيام", many: "{n} يومًا", other: "{n} يوم" });

const events = (l: Locale, n: number) =>
  l === "en"
    ? plural(l, n, { one: "{n} event", other: "{n} events" })
    : l === "ko"
      ? `기록 ${n}건`
      : plural(l, n, { zero: "{n} حدث", one: "حدث واحد", two: "حدثان", few: "{n} أحداث", many: "{n} حدثًا", other: "{n} حدث" });

export const browserNames: Record<Exclude<HistoryBrowser, "other" | "unknown">, string> = { edge: "Edge", chrome: "Chrome", firefox: "Firefox", safari: "Safari" };
export const systemNames: Record<Exclude<HistorySystem, "other" | "unknown">, string> = {
  windows: "Windows",
  macos: "macOS",
  linux: "Linux",
  ios: "iOS",
  android: "Android",
  chromeos: "ChromeOS",
};

export const hc = {
  title: (l: Locale) => t(l, "Recent access history", "최근 접근 기록", "سجل الوصول الأخير"),
  retentionLead: (l: Locale, n: number, configured: boolean) =>
    t(l, `Kept for ${days(l, n)}.`, `${days(l, n)} 동안 보관합니다.`, `يُحتفَظ به لمدة ${days(l, n)}.`) +
    (configured ? t(l, " (set for this installation)", " (이 설치에서 설정됨)", " (مُعدّ لهذا التثبيت)") : ""),
  retentionBody: (l: Locale) =>
    t(
      l,
      "Older events are hidden at once. While the server is running they are deleted from it within one day; if the server is stopped, they are deleted when it next starts, before it accepts sign-ins. Events under a legal hold are kept until the hold is released. Copies in backups of the server's data folder remain until those backups expire.",
      "그보다 오래된 기록은 즉시 숨깁니다. 서버가 실행 중이면 하루 안에 서버에서 삭제하고, 서버가 중지되어 있으면 다음 시작 시 로그인을 받기 전에 삭제합니다. 법적 보류 대상 기록은 보류가 해제될 때까지 보관합니다. 서버 데이터 폴더의 백업에 있는 사본은 해당 백업이 만료될 때까지 남습니다.",
      "تُخفى الأحداث الأقدم فورًا. أثناء تشغيل الخادم تُحذَف منه خلال يوم واحد؛ وإذا كان الخادم متوقفًا، تُحذَف عند تشغيله التالي قبل أن يقبل تسجيلات الدخول. تُحفَظ الأحداث الخاضعة لحجز قانوني حتى رفع الحجز. تبقى النسخ الموجودة في النسخ الاحتياطية لمجلد بيانات الخادم حتى انتهاء صلاحية تلك النسخ.",
    ),
  sizeLimit: (l: Locale, n: number) =>
    t(
      l,
      `Because of this server's size limit, about the last ${days(l, n)} are currently kept.`,
      `이 서버의 크기 한도 때문에 현재 최근 약 ${days(l, n)}의 기록만 보관됩니다.`,
      `بسبب حدّ الحجم في هذا الخادم، يُحتفَظ حاليًا بنحو آخر ${days(l, n)} فقط.`,
    ),
  coverage: (l: Locale, time: string) => t(l, `History available from ${time}.`, `${time}부터의 기록이 있습니다.`, `السجل متاح منذ ${time}.`),
  noneKept: (l: Locale) => t(l, "No history is kept yet.", "아직 보관된 기록이 없습니다.", "لا يوجد سجل محفوظ بعد."),
  minimisation: (l: Locale) =>
    t(l, "Network addresses and locations are not recorded.", "네트워크 주소와 위치는 기록하지 않습니다.", "لا تُسجَّل عناوين الشبكة ولا المواقع."),
  notShown: (l: Locale) =>
    t(l, "Expired sessions are not listed as events.", "만료된 세션은 기록에 표시되지 않습니다.", "لا تُدرَج الجلسات المنتهية الصلاحية كأحداث."),
  notesTitle: (l: Locale) => t(l, "Notices about this history", "이 기록에 관한 안내", "تنبيهات حول هذا السجل"),
  recordingStopped: (l: Locale, time: string) =>
    t(
      l,
      `Since ${time} this server has not been able to record access events. Events after that are missing.`,
      `${time} 이후 이 서버가 접근 기록을 저장하지 못하고 있습니다. 그 이후의 기록은 없습니다.`,
      `منذ ${time} لا يستطيع هذا الخادم تسجيل أحداث الوصول. الأحداث بعد ذلك مفقودة.`,
    ),
  deferred: (l: Locale) =>
    t(
      l,
      "Deletion of old events is paused until the server's clock is confirmed. Older events stay hidden.",
      "서버 시각이 확인될 때까지 오래된 기록 삭제를 멈췄습니다. 오래된 기록은 계속 숨깁니다.",
      "أُوقِف حذف الأحداث القديمة مؤقتًا حتى يُؤكَّد وقت الخادم. تبقى الأحداث الأقدم مخفية.",
    ),
  overdue: (l: Locale, n: number) =>
    t(
      l,
      `Events older than ${days(l, n)} are hidden, but the server has not yet been able to delete them.`,
      `${days(l, n)}보다 오래된 기록은 숨겼지만 서버가 아직 삭제하지 못했습니다.`,
      `الأحداث الأقدم من ${days(l, n)} مخفية، لكن الخادم لم يتمكّن بعد من حذفها.`,
    ),
  held: (l: Locale, count: number, n: number) =>
    t(
      l,
      `${plural(l, count, { one: "{n} of your events", other: "{n} of your events" })} older than ${days(l, n)} ${count === 1 ? "is" : "are"} kept because of a legal hold. They are not shown here.`,
      `${days(l, n)}보다 오래된 내 기록 ${count}건을 법적 보류 때문에 보관하고 있습니다. 여기에는 표시하지 않습니다.`,
      `يُحتفَظ بـ${events(l, count)} من أحداثك الأقدم من ${days(l, n)} بسبب حجز قانوني، ولا تُعرَض هنا.`,
    ),
  earlyDeletion: (l: Locale, time: string) =>
    t(
      l,
      `Events before ${time} were deleted early because the access history reached its size limit.`,
      `접근 기록이 크기 한도에 도달해 ${time} 이전 기록을 일찍 삭제했습니다.`,
      `حُذفت الأحداث السابقة لـ${time} مبكرًا لأن سجل الوصول بلغ حدّ الحجم.`,
    ),
  dropped: (l: Locale, count: number, from: string, to: string) =>
    t(
      l,
      `${plural(l, count, { one: "{n} of your access events", other: "{n} of your access events" })} between ${from} and ${to} could not be recorded.`,
      `${from}부터 ${to} 사이의 내 접근 기록 ${count}건을 저장하지 못했습니다.`,
      `تعذّر تسجيل ${events(l, count)} من أحداث الوصول الخاصة بك بين ${from} و${to}.`,
    ),
  possiblyMissing: (l: Locale, from: string, to: string) =>
    t(l, `Some events between ${from} and ${to} may be missing.`, `${from}부터 ${to} 사이의 일부 기록이 빠졌을 수 있습니다.`, `قد تكون بعض الأحداث بين ${from} و${to} مفقودة.`),
  possiblyMissingPlain: (l: Locale) =>
    t(l, "Some events may be missing.", "일부 기록이 빠졌을 수 있습니다.", "قد تكون بعض الأحداث مفقودة."),
  unclean: (l: Locale, time: string) =>
    t(
      l,
      `The server stopped unexpectedly around ${time}. Events just before that may be missing.`,
      `${time} 무렵 서버가 예기치 않게 중지되었습니다. 그 직전의 기록이 빠졌을 수 있습니다.`,
      `توقّف الخادم بشكل غير متوقّع نحو ${time}. قد تكون الأحداث التي سبقت ذلك مباشرة مفقودة.`,
    ),
  damaged: (l: Locale, count: number) =>
    t(
      l,
      `${plural(l, count, { one: "{n} stored event", other: "{n} stored events" })} could not be read and ${count === 1 ? "is" : "are"} not shown.`,
      `저장된 기록 ${count}건을 읽을 수 없어 표시하지 않습니다.`,
      `تعذّرت قراءة ${events(l, count)} من الأحداث المخزّنة ولا تُعرَض.`,
    ),
  recoveredTail: (l: Locale, time: string) =>
    t(
      l,
      `The end of a stored file was incomplete around ${time} and was repaired. Events just before that may be missing.`,
      `${time} 무렵 저장된 파일의 끝이 불완전하여 복구했습니다. 그 직전의 기록이 빠졌을 수 있습니다.`,
      `كانت نهاية ملف مخزّن غير مكتملة نحو ${time} وتم إصلاحها. قد تكون الأحداث التي سبقت ذلك مباشرة مفقودة.`,
    ),
  futureFile: (l: Locale) =>
    t(
      l,
      "Some stored events are dated after the server's current time. The server clock may have been changed.",
      "저장된 일부 기록의 날짜가 서버의 현재 시각보다 뒤입니다. 서버 시각이 변경되었을 수 있습니다.",
      "بعض الأحداث المخزّنة مؤرّخة بعد الوقت الحالي للخادم. ربما تغيّرت ساعة الخادم.",
    ),
  period: (l: Locale) => t(l, "Period", "최근 접근 기간", "فترة الوصول"),
  periodOption: (l: Locale, p: Period) =>
    ({
      day: t(l, "Last 24 hours", "최근 24시간", "آخر 24 ساعة"),
      week: t(l, "Last 7 days", "최근 7일", "آخر 7 أيام"),
      month: t(l, "Last 30 days", "최근 30일", "آخر 30 يومًا"),
      all: t(l, "Everything kept", "보관된 전체 기록", "كل ما هو محفوظ"),
    })[p],
  workspace: (l: Locale) => t(l, "Workspace:", "업무 공간:", "مساحة العمل:"),
  scopeNote: (l: Locale) =>
    t(
      l,
      "Events from your other workspaces are shown when you are signed in to them. Refused sign-ins are shown in every workspace.",
      "다른 업무 공간의 기록은 해당 업무 공간에 로그인했을 때 표시됩니다. 거부된 로그인은 모든 업무 공간에 표시됩니다.",
      "تُعرَض أحداث مساحات عملك الأخرى عند تسجيل الدخول إليها. تُعرَض تسجيلات الدخول المرفوضة في كل مساحة عمل.",
    ),
  nothingBefore: (l: Locale, time: string) => t(l, `Nothing before ${time} is kept.`, `${time} 이전 기록은 보관되어 있지 않습니다.`, `لا يُحتفَظ بشيء قبل ${time}.`),
  caption: (l: Locale) => t(l, "Recent access history, newest first", "최근 접근 기록, 최신순", "سجل الوصول الأخير، الأحدث أولًا"),
  colTime: (l: Locale) => t(l, "Time", "시각", "الوقت"),
  colEvent: (l: Locale) => t(l, "Event", "이벤트", "الحدث"),
  colSession: (l: Locale) => t(l, "Session", "세션", "الجلسة"),
  colDevice: (l: Locale) => t(l, "Device", "기기", "الجهاز"),
  thisSession: (l: Locale) => t(l, "This session", "현재 세션", "هذه الجلسة"),
  signedInNow: (l: Locale) => t(l, "Signed in now", "현재 로그인 중", "مسجّل الدخول الآن"),
  noSession: (l: Locale) => t(l, "No session", "세션 없음", "لا توجد جلسة"),
  notRecorded: (l: Locale) => t(l, "Not recorded", "기록되지 않음", "غير مسجّل"),
  unknownDevice: (l: Locale) => t(l, "Unknown device", "알 수 없는 기기", "جهاز غير معروف"),
  browserOn: (l: Locale, browser: string, os: string) => t(l, `${browser} on ${os}`, `${os}의 ${browser}`, `${browser} على ${os}`),
  unknownBrowserOn: (l: Locale, os: string) => t(l, `Unknown browser on ${os}`, `${os}의 알 수 없는 브라우저`, `متصفح غير معروف على ${os}`),
  browserOnUnknown: (l: Locale, browser: string) => t(l, `${browser} on an unknown system`, `알 수 없는 시스템의 ${browser}`, `${browser} على نظام غير معروف`),
  otherBrowser: (l: Locale) => t(l, "Other browser", "기타 브라우저", "متصفح آخر"),
  otherSystem: (l: Locale) => t(l, "other system", "기타 시스템", "نظام آخر"),
  loading: (l: Locale) => t(l, "Loading your access history…", "접근 기록을 불러오는 중…", "جارٍ تحميل سجل الوصول…"),
  loadError: (l: Locale) =>
    t(l, "We could not load your access history. Try again.", "접근 기록을 불러오지 못했습니다. 다시 시도해 주세요.", "تعذّر تحميل سجل الوصول. حاول مرة أخرى."),
  requestRef: (l: Locale) => t(l, "Request reference:", "요청 참조:", "مرجع الطلب:"),
  empty: (l: Locale) => t(l, "No access events in this period.", "이 기간에는 접근 기록이 없습니다.", "لا توجد أحداث وصول في هذه الفترة."),
  showMore: (l: Locale) => t(l, "Show more", "더 보기", "عرض المزيد"),
  loadingMore: (l: Locale) => t(l, "Loading older events…", "이전 기록을 불러오는 중…", "جارٍ تحميل الأحداث الأقدم…"),
  endLine: (l: Locale) => t(l, "No older events in this period.", "이 기간에 더 이전 기록은 없습니다.", "لا توجد أحداث أقدم في هذه الفترة."),
  showing: (l: Locale, p: Period, n: number, more: boolean) =>
    t(
      l,
      `Showing ${hc.periodOption(l, p)}: ${events(l, n)}, newest first.`,
      `${hc.periodOption(l, p)} 표시: ${events(l, n)}, 최신순.`,
      `عرض ${hc.periodOption(l, p)}: ${events(l, n)}، الأحدث أولًا.`,
    ) + (more ? " " + t(l, "More are available.", "더 있습니다.", "يتوفّر المزيد.") : ""),
  added: (l: Locale, n: number) =>
    l === "en"
      ? plural(l, n, { one: "{n} older event added.", other: "{n} older events added." })
      : t(l, "", `이전 ${events(l, n)}을 추가했습니다.`, `أُضيف ${events(l, n)} أقدم.`),
  moreRefused: (l: Locale, n: number) =>
    l === "en"
      ? plural(l, n, { one: "{n} more refused sign-in", other: "{n} more refused sign-ins" })
      : l === "ko"
        ? `추가 로그인 거부 ${n}건`
        : plural(l, n, {
            zero: "لا مزيد من تسجيلات الدخول المرفوضة",
            one: "تسجيل دخول مرفوض آخر",
            two: "تسجيلا دخول مرفوضان آخران",
            few: "{n} تسجيلات دخول مرفوضة أخرى",
            many: "{n} تسجيل دخول مرفوضًا آخر",
            other: "{n} تسجيل دخول مرفوض آخر",
          }),
  moreRefusedSub: (l: Locale) => t(l, "Not listed one by one on this day", "이날은 하나씩 표시하지 않음", "لم تُدرَج واحدةً واحدة في هذا اليوم"),
};

export type Period = "day" | "week" | "month" | "all";
export const periods: Period[] = ["all", "day", "week", "month"];
export const periodMs: Record<Exclude<Period, "all">, number> = { day: 24 * 3600e3, week: 7 * 24 * 3600e3, month: 30 * 24 * 3600e3 };

const labels: Record<string, (l: Locale) => string> = {
  signed_in: (l) => t(l, "Signed in", "로그인", "تسجيل الدخول"),
  sign_in_refused: (l) => t(l, "Sign-in refused", "로그인 거부", "رُفض تسجيل الدخول"),
  password_confirmed: (l) => t(l, "Password confirmed", "비밀번호 재확인", "تأكيد كلمة المرور"),
  password_confirmation_failed: (l) => t(l, "Password confirmation failed", "비밀번호 재확인 실패", "فشل تأكيد كلمة المرور"),
  signed_out: (l) => t(l, "Signed out", "로그아웃", "تسجيل الخروج"),
  session_ended: (l) => t(l, "Session ended", "세션 종료", "انتهت الجلسة"),
  password_changed: (l) => t(l, "Password changed", "비밀번호 변경", "تغيير كلمة المرور"),
};

const reasonLabels: Record<string, (l: Locale) => string> = {
  wrong_password: (l) => t(l, "Wrong password", "비밀번호 오류", "كلمة مرور خاطئة"),
  locked: (l) => t(l, "Account locked", "계정 잠김", "الحساب مقفل"),
  access_refused: (l) => t(l, "Access refused", "접근 거부", "رُفض الوصول"),
  credentials_changed: (l) => t(l, "Sign-in details changed during sign-in", "로그인 중 로그인 정보가 변경됨", "تغيّرت بيانات تسجيل الدخول أثناء تسجيل الدخول"),
  session_unavailable: (l) => t(l, "Session could not be opened", "세션을 열 수 없음", "تعذّر فتح الجلسة"),
  attempt_limit: (l) => t(l, "Too many wrong passwords", "비밀번호 오류 횟수 초과", "كلمات مرور خاطئة كثيرة جدًا"),
  ended_by_you: (l) => t(l, "By you", "내가 종료", "بواسطتك"),
  password_changed: (l) => t(l, "Password changed", "비밀번호 변경", "تغيير كلمة المرور"),
  password_reset: (l) => t(l, "Password reset", "비밀번호 재설정", "إعادة تعيين كلمة المرور"),
  administrator: (l) => t(l, "By an administrator", "관리자가 종료", "بواسطة مسؤول"),
  role_period_ended: (l) => t(l, "Role period ended", "역할 기간 종료", "انتهت مدة الدور"),
  session_limit: (l) => t(l, "Session limit", "세션 수 한도", "حد الجلسات"),
  access_changed: (l) => t(l, "Access changed", "접근 권한 변경", "تغيّر الوصول"),
  trust_change: (l) => t(l, "Sign-in rules changed", "로그인 규칙 변경", "تغيّرت قواعد تسجيل الدخول"),
  bulk_revocation: (l) => t(l, "Bulk revocation", "대량 회수", "الإبطال الجماعي"),
};

export function eventLabel(l: Locale, e: HistoryEvent): string {
  if (e.kind === "sign_in_refused_more") return hc.moreRefused(l, e.count);
  return labels[e.kind](l);
}

export function eventReason(l: Locale, e: HistoryEvent): string {
  if (e.kind === "sign_in_refused_more") return hc.moreRefusedSub(l);
  return e.reason ? reasonLabels[e.reason](l) : "";
}

export function deviceLabel(l: Locale, e: HistoryEvent): string {
  if (!e.device) return "";
  const { browser, os } = e.device;
  if (browser === "unknown" && os === "unknown") return hc.unknownDevice(l);
  const b = browser === "other" ? hc.otherBrowser(l) : browser === "unknown" ? "" : browserNames[browser];
  const o = os === "other" ? hc.otherSystem(l) : os === "unknown" ? "" : systemNames[os];
  if (!b) return hc.unknownBrowserOn(l, o);
  if (!o) return hc.browserOnUnknown(l, b);
  return hc.browserOn(l, b, o);
}

export const emphasised = (e: HistoryEvent) => e.kind === "sign_in_refused" || e.kind === "sign_in_refused_more" || e.kind === "password_confirmation_failed";
