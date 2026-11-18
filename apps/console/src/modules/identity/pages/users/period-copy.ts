import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { counted } from "../../../../shared/plural";

export const p = {
  rolePeriod: (l: Locale) => t(l, "Role period", "역할 기간", "مدة الدور"),
  roleAndPeriod: (l: Locale) => t(l, "Role and period", "역할 및 기간", "الدور والمدة"),
  noEnd: (l: Locale) => t(l, "No end date", "종료일 없음", "بلا تاريخ انتهاء"),
  noPeriod: (l: Locale) => t(l, "No role period", "역할 기간 없음", "لا توجد مدة دور"),
  until: (l: Locale) => t(l, "Until {at}", "{at}까지", "حتى {at}"),
  endsIn: (l: Locale, n: number) =>
    counted(l, n, { en: ["Ends in {n} day", "Ends in {n} days"], ko: "{n}일 후 종료", ar: ["تنتهي بعد {n} يوم", "تنتهي بعد يوم واحد", "تنتهي بعد يومين", "تنتهي بعد {n} أيام", "تنتهي بعد {n} يومًا", "تنتهي بعد {n} يوم"] }),
  endsToday: (l: Locale) => t(l, "Ends today at {at}", "오늘 {at}에 종료", "تنتهي اليوم عند {at}"),
  ended: (l: Locale) => t(l, "Role period ended", "역할 기간 종료", "انتهت مدة الدور"),
  endedL: (l: Locale) => t(l, "Ended", "종료:", "انتهت في"),
  fromL: (l: Locale) => t(l, "From", "시작:", "من"),
  unconfirmed: (l: Locale) => t(l, "Decision not confirmed", "결정 기록 미확정", "القرار غير مؤكَّد"),
  change: (l: Locale) => t(l, "Change role period", "역할 기간 변경", "تغيير مدة الدور"),
  current: (l: Locale) => t(l, "Current:", "현재:", "الحالي:"),
  roleEnds: (l: Locale) => t(l, "Role ends", "역할 종료", "انتهاء الدور"),
  endDate: (l: Locale) => t(l, "End date", "종료일", "تاريخ الانتهاء"),
  endDateTime: (l: Locale) => t(l, "End date and time", "종료 날짜와 시간", "تاريخ ووقت الانتهاء"),
  zone: (l: Locale, zone: string) => t(l, `Times are in your browser's time zone (${zone})`, `시간은 브라우저 시간대(${zone}) 기준입니다`, `الأوقات بالمنطقة الزمنية لمتصفحك (${zone})`),
  sentAs: (l: Locale) => t(l, "Sent as", "전송 값", "يُرسل بصيغة"),
  serverNow: (l: Locale) => t(l, "Server time now", "현재 서버 시간", "وقت الخادم الآن"),
  latest: (l: Locale) => t(l, "Latest allowed end", "허용되는 가장 늦은 종료", "أبعد انتهاء مسموح"),
  own: (l: Locale) => t(l, "Your own role end", "내 역할 종료", "انتهاء دورك أنت"),
  ownLimit: (l: Locale) =>
    t(l, "Your own role ends on {at}, so you cannot choose \"No end date\".", "내 역할이 {at}에 끝나므로 '종료일 없음'을 선택할 수 없습니다.", "ينتهي دورك في {at}، لذا لا يمكنك اختيار «بلا تاريخ انتهاء»."),
  shorter: (l: Locale) =>
    t(l, "Access through this role ends at {at}. Their sessions end then.", "이 역할을 통한 접근은 {at}에 끝납니다. 그때 세션도 종료됩니다.", "ينتهي الوصول عبر هذا الدور في {at}. وتنتهي جلساتهم حينها."),
  longer: (l: Locale) =>
    t(l, "This extends access. The server refuses it while the decision ledger cannot record or a trust hold applies.", "접근을 연장합니다. 결정 원장이 기록할 수 없거나 신뢰 보류 중에는 거부됩니다.",
      "هذا يمدد الوصول. ويرفضه الخادم ما دام دفتر القرارات لا يستطيع التسجيل أو ما دام حجز الثقة قائمًا."),
  clockDiffers: (l: Locale) =>
    t(l, "The host clock differs from the server's time; role periods use the server's time.", "호스트 시계가 서버 시간과 다릅니다. 역할 기간은 서버 시간을 사용합니다.", "تختلف ساعة المضيف عن وقت الخادم؛ وتستخدم مدد الأدوار وقت الخادم."),
  durableNo: (l: Locale) =>
    [t(l, "The change is in effect now, but its record could not be confirmed.", "변경은 지금 적용되었지만 기록을 확인하지 못했습니다.", "التغيير نافذ الآن، لكن تعذّر تأكيد سجلّه."),
      t(l, "It will not survive a restart.", "서버를 다시 시작하면 유지되지 않습니다.", "لن يبقى بعد إعادة التشغيل.")],
  durableUnknown: (l: Locale) =>
    [t(l, "The change is in effect now, but its record could not be confirmed.", "변경은 지금 적용되었지만 기록을 확인하지 못했습니다.", "التغيير نافذ الآن، لكن تعذّر تأكيد سجلّه."),
      t(l, "It may or may not survive a restart.", "서버를 다시 시작한 뒤 유지될 수도, 유지되지 않을 수도 있습니다.", "قد يبقى بعد إعادة التشغيل وقد لا يبقى.")],
  unconfirmedNotice: (l: Locale) =>
    t(l, "The shorter period is in effect in this server process, but the decision ledger has not confirmed its record. It may not be re-applied after a restart.",
      "더 짧은 기간은 이 서버 프로세스 안에서 적용되었지만 결정 원장이 기록을 확정하지 않았습니다. 재시작 후 다시 적용되지 않을 수 있습니다.",
      "المدة الأقصر نافذة في عملية الخادم هذه، لكن دفتر القرارات لم يؤكّد قيدها. قد لا يُعاد تطبيقها بعد إعادة التشغيل."),
  unchanged: (l: Locale) => t(l, "The end is unchanged.", "종료가 바뀌지 않았습니다.", "لم يتغير الانتهاء."),
  ended_refused: (l: Locale) =>
    t(l, "This membership's role period ended. Use Change role period to give a new period.", "이 소속의 역할 기간이 끝났습니다. 새 기간을 주려면 역할 기간 변경을 사용하세요.", "انتهت مدة الدور لهذه العضوية. استخدم تغيير مدة الدور لمنح مدة جديدة."),
  capRefused: (l: Locale) =>
    t(l, "Choose an end at least a minute ahead, within the longest period and not after your own role end.", "1분 이후이면서 최장 기간 안, 그리고 내 역할 종료 이전의 종료를 선택하세요.", "اختر انتهاءً بعد دقيقة على الأقل، ضمن أطول مدة ولا يتجاوز انتهاء دورك."),
  enterEnd: (l: Locale) => t(l, "Enter an end date and time.", "종료 날짜와 시간을 입력하세요.", "أدخل تاريخ ووقت الانتهاء."),
  settings: (l: Locale) => t(l, "Installation settings", "설치 설정", "إعدادات التثبيت"),
  done: (l: Locale) => t(l, "Role period changed", "역할 기간을 변경했습니다", "تم تغيير مدة الدور"),
};
