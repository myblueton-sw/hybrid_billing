import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { counted } from "../../../../shared/plural";
import { fc } from "./filter-copy";

const keep = { n: "{n}" };

const people = (l: Locale, n: number) =>
  counted(l, n, { en: ["{n} person", "{n} people"], ko: "{n}명", ar: ["{n} شخص", "شخص واحد", "شخصان", "{n} أشخاص", "{n} شخصًا", "{n} شخص"] });

const upTo = (l: Locale) => fc.searchedUpTo(l).replace(/\s*\{login\}/, l === "ko" ? "…" : " …").replace(/[.。]$/, "");

export const m = {
  selectMatching: (l: Locale) => t(l, "Select everyone who matches these filters", "이 필터에 맞는 사람 모두 선택", "تحديد كل من يطابق عوامل التصفية هذه"),
  selectAll: (l: Locale) => t(l, "Select everyone in the list", "목록의 모든 사람 선택", "تحديد كل من في القائمة"),
  none: (l: Locale) =>
    t(l, "No one selected. Use the checkboxes, or select everyone who matches.", "선택한 사람이 없습니다. 확인란으로 고르거나 조건에 맞는 사람을 모두 선택하세요.",
      "لم يُحدَّد أحد. استخدم خانات الاختيار، أو حدّد كل من يطابق."),
  counting: (l: Locale) => t(l, "Counting people who match…", "조건에 맞는 사람을 세는 중…", "جارٍ عدّ المطابقين…"),
  countedAt: (l: Locale) =>
    t(l, "Counted at {at}. The number can change before the list is fixed.", "{at} 기준 집계입니다. 목록을 고정하기 전에 인원이 바뀔 수 있습니다.", "تم العدّ في {at}. قد يتغير العدد قبل تثبيت القائمة."),
  lowerBound: (l: Locale, scan: number, n: number, limit: number) =>
    t(l, `Counting stopped after checking ${scan} people. At least ${n} match. Selecting checks everyone, and the list is refused if more than ${limit} match.`,
      `확인한 ${scan}명에서 세기를 멈췄습니다. 최소 ${n}명이 해당됩니다. 선택하면 모든 사람을 확인하며, 해당하는 사람이 ${limit}명보다 많으면 목록이 거부됩니다.`,
      `توقف العدّ بعد فحص ${scan} شخص. يطابق ${n} على الأقل. عند التحديد يُفحص الجميع، وتُرفض القائمة إذا طابق أكثر من ${limit}.`),
  selectCounted: (l: Locale, n: number) =>
    counted(l, n, { en: ["Select all {n} who match", "Select all {n} who match"], ko: "조건에 맞는 {n}명 모두 선택",
      ar: ["تحديد كل المطابقين ({n})", "تحديد الشخص المطابق", "تحديد المطابقَين", "تحديد كل المطابقين ({n})", "تحديد كل المطابقين ({n})", "تحديد كل المطابقين ({n})"] }),
  selectAtLeast: (l: Locale, n: number) => t(l, `Select all who match (at least ${n})`, `조건에 맞는 사람 모두 선택(최소 ${n}명)`, `تحديد كل المطابقين (${n} على الأقل)`),
  tooMany: (l: Locale, limit: number) =>
    t(l, `More people match than one revocation list allows (${limit}). Narrow the filters.`, `한 회수 목록에 넣을 수 있는 인원(${limit}명)보다 많은 사람이 해당됩니다. 필터를 좁히세요.`,
      `يطابق أشخاص أكثر مما تسمح به قائمة إبطال واحدة (${limit}). ضيّق عوامل التصفية.`),
  noneMatch: (l: Locale) => t(l, "Nobody matches these filters now.", "지금 이 필터에 맞는 사람이 없습니다.", "لا أحد يطابق عوامل التصفية هذه الآن."),
  replaceConfirm: (l: Locale, n: number) =>
    counted(l, n, { en: ["This replaces your {n} selected person with everyone who matches.", "This replaces your {n} selected people with everyone who matches."],
      ko: "선택한 {n}명을 필터에 맞는 모든 사람으로 바꿉니다.",
      ar: ["سيحلّ هذا محل المحددين ({n}) بكل من يطابق.", "سيحلّ هذا محل الشخص المحدد بكل من يطابق.", "سيحلّ هذا محل الشخصين المحددين بكل من يطابق.",
        "سيحلّ هذا محل الأشخاص المحددين ({n}) بكل من يطابق.", "سيحلّ هذا محل الأشخاص المحددين ({n}) بكل من يطابق.", "سيحلّ هذا محل الأشخاص المحددين ({n}) بكل من يطابق."] }),
  selecting: (l: Locale) => t(l, "Selecting…", "선택하는 중…", "جارٍ التحديد…"),
  combined: (l: Locale) =>
    t(l, "The list was not fixed. Too many or no people may match, or you may already have an open list. Check Revocation jobs and the filters.",
      "목록을 고정하지 못했습니다. 해당하는 사람이 너무 많거나 없을 수 있고, 이미 열린 목록이 있을 수도 있습니다. 회수 작업과 필터를 확인하세요.",
      "لم تُثبَّت القائمة. قد يطابق عدد كبير جدًا أو لا أحد، أو قد تكون لديك قائمة مفتوحة بالفعل. راجع «مهام الإبطال» وعوامل التصفية."),
  fixed: (l: Locale, n: number) =>
    counted(l, n, { en: ["{n} person fixed at {at} for these filters", "{n} people fixed at {at} for these filters"], ko: "{at}에 이 필터로 {n}명을 고정했습니다",
      ar: ["تم تثبيت {n} شخص في {at} لعوامل التصفية هذه", "تم تثبيت شخص واحد في {at} لعوامل التصفية هذه", "تم تثبيت شخصين في {at} لعوامل التصفية هذه",
        "تم تثبيت {n} أشخاص في {at} لعوامل التصفية هذه", "تم تثبيت {n} شخصًا في {at} لعوامل التصفية هذه", "تم تثبيت {n} شخص في {at} لعوامل التصفية هذه"] }, keep),
  fixedShort: (l: Locale, n: number) =>
    counted(l, n, { en: ["{n} person fixed at {at}", "{n} people fixed at {at}"], ko: "{at}에 {n}명을 고정했습니다",
      ar: ["تم تثبيت {n} شخص في {at}", "تم تثبيت شخص واحد في {at}", "تم تثبيت شخصين في {at}", "تم تثبيت {n} أشخاص في {at}", "تم تثبيت {n} شخصًا في {at}", "تم تثبيت {n} شخص في {at}"] }, keep),
  fixedEveryone: (l: Locale, n: number) =>
    counted(l, n, { en: ["{n} person fixed at {at}: everyone in the list", "{n} people fixed at {at}: everyone in the list"], ko: "{at}에 목록의 모든 사람 {n}명을 고정했습니다",
      ar: ["تم تثبيت {n} شخص في {at}: كل من في القائمة", "تم تثبيت شخص واحد في {at}: كل من في القائمة", "تم تثبيت شخصين في {at}: كل من في القائمة",
        "تم تثبيت {n} أشخاص في {at}: كل من في القائمة", "تم تثبيت {n} شخصًا في {at}: كل من في القائمة", "تم تثبيت {n} شخص في {at}: كل من في القائمة"] }, keep),
  fixedDiffers: (l: Locale, n: number, when: number) =>
    counted(l, n, { en: [`{n} person fixed at {at} (${when} when counted)`, `{n} people fixed at {at} (${when} when counted)`], ko: `{at}에 {n}명을 고정했습니다(집계 당시 ${when}명)`,
      ar: [`تم تثبيت {n} شخص في {at} (كان العدد ${when} عند العدّ)`, `تم تثبيت شخص واحد في {at} (كان العدد ${when} عند العدّ)`, `تم تثبيت شخصين في {at} (كان العدد ${when} عند العدّ)`,
        `تم تثبيت {n} أشخاص في {at} (كان العدد ${when} عند العدّ)`, `تم تثبيت {n} شخصًا في {at} (كان العدد ${when} عند العدّ)`, `تم تثبيت {n} شخص في {at} (كان العدد ${when} عند العدّ)`] }, keep),
  fixedBelow: (l: Locale, n: number, when: number) =>
    counted(l, n, { en: [`{n} person fixed at {at} (at least ${when} when counted)`, `{n} people fixed at {at} (at least ${when} when counted)`], ko: `{at}에 {n}명을 고정했습니다(집계 당시 최소 ${when}명)`,
      ar: [`تم تثبيت {n} شخص في {at} (كان العدد ${when} على الأقل عند العدّ)`, `تم تثبيت شخص واحد في {at} (كان العدد ${when} على الأقل عند العدّ)`, `تم تثبيت شخصين في {at} (كان العدد ${when} على الأقل عند العدّ)`,
        `تم تثبيت {n} أشخاص في {at} (كان العدد ${when} على الأقل عند العدّ)`, `تم تثبيت {n} شخصًا في {at} (كان العدد ${when} على الأقل عند العدّ)`, `تم تثبيت {n} شخص في {at} (كان العدد ${when} على الأقل عند العدّ)`] }, keep),
  fixedFilters: (l: Locale) => t(l, "Fixed filters:", "고정한 필터:", "عوامل التصفية المثبّتة:"),
  baseRevision: (l: Locale) => t(l, "Base revision", "기준 개정 번호", "رقم التعديل الأساسي"),
  fixedNoChecks: (l: Locale) => t(l, "Discard the list to choose people one by one.", "사람을 한 명씩 고르려면 목록을 버리세요.", "تجاهل القائمة لاختيار الأشخاص واحدًا تلو الآخر."),
  keepsOwn: (l: Locale) =>
    t(l, "The fixed list keeps its own filters. Changing the filters above does not change it.", "고정 목록은 자체 필터를 유지합니다. 위 필터를 바꿔도 목록은 바뀌지 않습니다.",
      "تحتفظ القائمة المثبّتة بعوامل التصفية الخاصة بها. تغيير عوامل التصفية أعلاه لا يغيّرها."),
  discardList: (l: Locale) => t(l, "Discard list", "목록 버리기", "تجاهل القائمة"),
  discarded: (l: Locale) => t(l, "The fixed list was discarded. Nothing was revoked.", "고정 목록을 버렸습니다. 아무것도 회수되지 않았습니다.", "تم تجاهل القائمة المثبّتة. لم يُبطَل شيء."),
  afterFix: (l: Locale, line: string) =>
    t(l, `${line}. Row checkboxes are hidden while the list is fixed.`, `${line}. 목록이 고정된 동안 행 확인란은 숨겨집니다.`, `${line}. تُخفى خانات اختيار الصفوف ما دامت القائمة مثبّتة.`),
  everyoneMatching: (l: Locale, n: number) => t(l, `Everyone matching filters (${n})`, `필터에 맞는 모든 사람(${n}명)`, `كل المطابقين لعوامل التصفية (${n})`),
  serverChecks: (l: Locale) =>
    t(l, "When you preview, the server checks each person on the list against these filters again and excludes anyone who no longer matches.",
      "미리보기 때 서버가 목록의 각 사람을 이 필터로 다시 확인하고, 더 이상 맞지 않는 사람은 제외합니다.",
      "عند المعاينة يتحقق الخادم من كل شخص في القائمة مقابل عوامل التصفية هذه مجددًا، ويستثني كل من لم يعد يطابق."),
  checking: (l: Locale, n: number) => t(l, `Checking ${people(l, n)}…`, `${n}명을 확인하는 중…`, `جارٍ فحص الأشخاص (${n})…`),
  blockedFilter: (l: Locale, example: string) =>
    t(l, `This list would leave no administrator who can manage access. Narrow the filters, for example ${example}.`, `이 목록으로는 접근을 관리할 관리자가 남지 않습니다. 필터를 좁히세요. 예: ${example}`,
      `لن يبقى بهذه القائمة أي مسؤول يمكنه إدارة الوصول. ضيّق عوامل التصفية، مثل: ${example}`),
  listClosed: (l: Locale) =>
    t(l, "This list can no longer be prepared: it expired, or nobody in it can be revoked now. Discard it and select again.",
      "이 목록은 더 이상 준비할 수 없습니다. 만료되었거나 지금 회수할 수 있는 사람이 없습니다. 버리고 다시 선택하세요.",
      "لم يعد ممكنًا تحضير هذه القائمة: انتهت صلاحيتها أو لا يوجد فيها من يمكن إبطاله الآن. تجاهلها وحدّد مرة أخرى."),
  fixedList: (l: Locale) => t(l, "Fixed list", "고정 목록", "القائمة المثبّتة"),
  filters: (l: Locale) => t(l, "Filters", "필터", "عوامل التصفية"),
  fixedAt: (l: Locale) => t(l, "Fixed at", "고정 시각", "وقت التثبيت"),
  matchedWhenFixed: (l: Locale) => t(l, "Matched when fixed", "고정 당시 해당 인원", "المطابقون عند التثبيت"),
  arithmetic: (l: Locale, matched: number, targets: number, excluded: number) =>
    t(l, `${people(l, matched)} fixed: ${targets} to revoke, ${excluded} excluded.`, `고정 ${matched}명: 회수 ${targets}명, 제외 ${excluded}명.`,
      `المثبَّتون: ${matched}؛ المراد إبطالهم: ${targets}؛ المستثنون: ${excluded}.`),
  noLongerMatch: (l: Locale, k: number) =>
    counted(l, k, { en: ["{n} person no longer matches the filters and is excluded.", "{n} people no longer match the filters and are excluded."], ko: "{n}명은 더 이상 필터에 맞지 않아 제외됩니다.",
      ar: ["لم يعد {n} شخص يطابقون عوامل التصفية، فاستُثنوا.", "لم يعد شخص واحد يطابق عوامل التصفية، فاستُثني.", "لم يعد شخصان يطابقان عوامل التصفية، فاستُثنيا.",
        "لم يعد {n} أشخاص يطابقون عوامل التصفية، فاستُثنوا.", "لم يعد {n} شخصًا يطابقون عوامل التصفية، فاستُثنوا.", "لم يعد {n} شخص يطابقون عوامل التصفية، فاستُثنوا."] }),
  notAdded: (l: Locale) =>
    t(l, "People who match after {at} are not in this list. Select again for a new list.", "{at} 이후에 조건에 맞게 된 사람은 이 목록에 없습니다. 새 목록은 다시 선택하세요.",
      "الأشخاص الذين يطابقون بعد {at} ليسوا في هذه القائمة. حدّد مرة أخرى لإنشاء قائمة جديدة."),
  everyoneInList: (l: Locale) => t(l, "Everyone in the list", "목록의 모든 사람", "كل من في القائمة"),
  searchApplied: (l: Locale) => t(l, "Search applied", "검색 조건 적용됨", "تم تطبيق بحث"),
  basisL: (l: Locale) => t(l, "Selection basis", "선택 기준", "أساس التحديد"),
  basis: (l: Locale, selection: string) =>
    selection === "filter" ? t(l, "Everyone matching filters", "필터에 맞는 모든 사람", "كل المطابقين لعوامل التصفية")
      : selection === "not_recorded" ? t(l, "Selection basis not recorded", "선택 기준 기록 없음", "أساس التحديد غير مسجّل")
        : t(l, "Chosen people", "직접 선택", "أشخاص مختارون"),
  fixedWhen: (l: Locale) => t(l, "Fixed {at}", "{at} 고정", "ثُبّتت {at}"),
  matchRationale: (l: Locale) =>
    t(l, "Selected everyone who matched the filters on the revocation screen", "회수 화면에서 필터에 맞는 사람 모두 선택", "تحديد كل المطابقين لعوامل التصفية في شاشة الإبطال"),
  settingHint: (l: Locale, name: string) =>
    ({
      bulk_revocation_retained_people_max: t(l, "Counts finished jobs, ended lists and lists waiting to run, each up to the people per filter-wide revocation.",
        "완료된 작업, 종료된 목록, 실행 대기 중인 목록을 각각 필터 전체 회수당 인원까지 셉니다.",
        "يشمل المهام المكتملة والقوائم المنتهية والقوائم بانتظار التنفيذ، كلٌّ منها حتى عدد الأشخاص لكل إبطال حسب التصفية."),
      user_count_scan: t(l, "After this many people are checked, the count before selecting shows “at least”.", "이 인원을 확인하면 선택 전 집계를 멈추고 ‘최소’로 표시합니다.",
        "بعد هذا العدد يظهر العدّ قبل التحديد بعبارة «على الأقل»."),
      user_filter_scan: t(l, `After this many people are checked for one page, the list stops at “${upTo(l)}” and offers ${fc.continueSearching(l)}.`,
        `한 페이지에서 이 인원을 확인하면 목록이 ‘${upTo(l)}’에서 멈추고 ‘${fc.continueSearching(l)}’을 제공합니다.`,
        `بعد فحص هذا العدد في صفحة واحدة تتوقف القائمة عند «${upTo(l)}» وتعرض «${fc.continueSearching(l)}».`),
    } as Record<string, string>)[name] ?? "",
};
