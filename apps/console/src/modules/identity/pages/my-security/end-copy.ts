import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { SelfEnd } from "../../../../service/self-end";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };

function plural(l: Locale, n: number, forms: Partial<Record<Intl.LDMLPluralRule, string>> & { other: string }): string {
  const rule = new Intl.PluralRules(tags[l]).select(n);
  return (forms[rule] ?? forms.other).replaceAll("{n}", String(n));
}

const sessions = (l: Locale, n: number) =>
  l === "en"
    ? plural(l, n, { one: "{n} session", other: "{n} sessions" })
    : l === "ko"
      ? `세션 ${n}개`
      : plural(l, n, { zero: "لا جلسات", one: "جلسة واحدة", two: "جلستين", few: "{n} جلسات", many: "{n} جلسة", other: "{n} جلسة" });

export function whenSeconds(l: Locale, iso: string): string {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime())) return "";
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium", timeStyle: "medium" }).format(d);
}

export function datePart(l: Locale, iso: string): string {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime())) return "";
  return new Intl.DateTimeFormat(tags[l], { dateStyle: "medium" }).format(d);
}

export function timePart(l: Locale, iso: string): string {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime())) return "";
  return new Intl.DateTimeFormat(tags[l], { timeStyle: "short" }).format(d);
}

export const e = {
  entry: (l: Locale) => t(l, "End other sessions", "다른 세션 종료", "إنهاء الجلسات الأخرى"),
  othersTitle: (l: Locale) => t(l, "End other sessions?", "다른 세션을 종료할까요?", "هل تريد إنهاء الجلسات الأخرى؟"),
  reading: (l: Locale) => t(l, "Getting your current sessions…", "현재 세션을 확인하는 중…", "جارٍ جلب جلساتك الحالية…"),
  lead: (l: Locale, n: number) =>
    t(
      l,
      `Access ends for the ${sessions(l, n)} listed below. Whoever uses them is signed out on their next request.`,
      `아래 ${sessions(l, n)}의 접근이 종료됩니다. 해당 세션을 쓰는 사람은 다음 요청에서 로그아웃됩니다.`,
      `ينتهي وصول الجلسات المدرجة أدناه (${n}). يُسجَّل خروج من يستخدمها عند طلبه التالي.`,
    ),
  scope: (l: Locale) => t(l, "This session is not included and stays signed in.", "이 세션은 포함되지 않으며 로그인 상태가 유지됩니다.", "هذه الجلسة غير مشمولة وتبقى مسجّلة الدخول."),
  thisRef: (l: Locale) => t(l, "This session: reference", "이 세션: 참조", "هذه الجلسة: المرجع"),
  manifestTitle: (l: Locale, n: number) => t(l, `Sessions to end (${n})`, `종료할 세션 (${n})`, `الجلسات المراد إنهاؤها (${n})`),
  listTime: (l: Locale, time: string) => t(l, `List taken at ${time}`, `목록 기준 시각 ${time}`, `وقت أخذ القائمة ${time}`),
  note: (l: Locale) =>
    t(
      l,
      "Sessions that start after this list was taken are not ended. They are shown in the result so you can end them.",
      "이 목록을 가져온 뒤 시작된 세션은 종료하지 않습니다. 결과에 표시되므로 따로 종료할 수 있습니다.",
      "لا تُنهى الجلسات التي تبدأ بعد أخذ هذه القائمة. تظهر في النتيجة لتتمكن من إنهائها.",
    ),
  button: (l: Locale, n: number) =>
    l === "en"
      ? plural(l, n, { one: "End {n} other session", other: "End {n} other sessions" })
      : l === "ko"
        ? `다른 세션 ${n}개 종료`
        : plural(l, n, { one: "إنهاء جلسة أخرى واحدة", two: "إنهاء الجلستين الأخريين", few: "إنهاء {n} جلسات أخرى", many: "إنهاء {n} جلسة أخرى", other: "إنهاء {n} جلسة أخرى" }),
  busy: (l: Locale) => t(l, "Ending sessions…", "세션을 종료하는 중…", "جارٍ إنهاء الجلسات…"),
  close: (l: Locale) => t(l, "Close", "닫기", "إغلاق"),
  reloadClose: (l: Locale) => t(l, "Reload and close", "다시 불러오고 닫기", "إعادة التحميل والإغلاق"),
  unknown: (l: Locale) =>
    t(
      l,
      "We could not confirm the result. Before submitting again, check your current sessions.",
      "변경 결과를 확인하지 못했습니다. 다시 제출하기 전에 현재 연결·세션 상태를 확인해 주세요.",
      "تعذّر تأكيد النتيجة. قبل الإرسال مجددًا، تحقّق من جلساتك الحالية.",
    ),
  replay: (l: Locale) =>
    t(l, "This is the result of your earlier request. Nothing new was ended.", "이전 요청의 결과입니다. 새로 종료한 세션은 없습니다.", "هذه نتيجة طلبك السابق. لم يُنهَ شيء جديد."),
  changed: (l: Locale) =>
    t(
      l,
      "Your sessions changed, so nothing was ended. The list has been reloaded. Check it and try again.",
      "세션이 바뀌어 아무것도 종료하지 않았습니다. 목록을 다시 불러왔습니다. 확인한 뒤 다시 시도하세요.",
      "تغيّرت جلساتك، لذا لم يُنهَ أي شيء. أُعيد تحميل القائمة. تحقّق منها وحاول مجددًا.",
    ),
  endsTitle: (l: Locale) => t(l, "Recent session ends", "최근 세션 종료", "آخر عمليات إنهاء الجلسات"),
  endsMeta: (l: Locale) =>
    t(l, "The latest 5 are kept until the service restarts.", "최근 5건은 서비스가 다시 시작될 때까지 보관됩니다.", "تُحفظ آخر 5 عمليات حتى إعادة تشغيل الخدمة."),
  opOthers: (l: Locale) => t(l, "Other sessions ended", "다른 세션 종료 요청", "طلب إنهاء الجلسات الأخرى"),
  opOne: (l: Locale) => t(l, "One session ended", "세션 1개 종료 요청", "طلب إنهاء جلسة واحدة"),
  running: (l: Locale) =>
    t(l, "Access is blocked, but cleanup of some sessions is still in progress.", "접근 차단은 적용되었지만 일부 세션 정리가 진행 중입니다.", "تم حجب الوصول، لكن تنظيف بعض الجلسات لا يزال جاريًا."),
  complete: (l: Locale, n: number) =>
    t(
      l,
      `Access is blocked for ${sessions(l, n)} and cleanup is complete.`,
      `${sessions(l, n)}의 접근을 차단했고 정리가 끝났습니다.`,
      `تم حجب الوصول عن الجلسات (${n}) واكتمل التنظيف.`,
    ),
  noRecall: (l: Locale) =>
    t(
      l,
      "Pages already shown in those browsers stay on screen, and requests already in progress finish.",
      "해당 브라우저에 이미 표시된 화면은 그대로 남고, 진행 중이던 요청은 끝까지 처리됩니다.",
      "تبقى الصفحات المعروضة مسبقًا في تلك المتصفحات ظاهرة، وتكتمل الطلبات الجارية بالفعل.",
    ),
  completeNone: (l: Locale) =>
    t(
      l,
      "No session needed blocking: every session in your list had already ended.",
      "차단할 세션이 없었습니다. 목록의 세션이 모두 이미 종료되어 있었습니다.",
      "لم تكن هناك جلسة تحتاج إلى الحجب: كانت كل جلسات قائمتك قد انتهت مسبقًا.",
    ),
  incomplete: (l: Locale, n: number) =>
    t(
      l,
      `Access is blocked for all ${sessions(l, n)}; none of them can be used.`,
      `${sessions(l, n)} 모두 접근이 차단되어 사용할 수 없습니다.`,
      `تم حجب الوصول عن جميع الجلسات (${n})، ولا يمكن استخدام أيٍّ منها.`,
    ),
  incompleteCount: (l: Locale, k: number) => t(l, `Cleanup of ${k} could not finish.`, `그중 ${k}개의 정리를 끝내지 못했습니다.`, `عدد الجلسات التي تعذّر إكمال تنظيفها: ${k}.`),
  titleUnknown: (l: Locale) => t(l, "Result not confirmed", "결과 미확인", "لم تُؤكَّد النتيجة"),
  titleNothing: (l: Locale) => t(l, "Nothing was ended", "종료하지 않았습니다", "لم يُنهَ أي شيء"),
  blockedNone: (l: Locale) =>
    t(l, "None: every session in your list had already ended", "없음: 목록의 세션이 모두 이미 종료되어 있었습니다", "لا شيء: كانت كل جلسات قائمتك قد انتهت مسبقًا"),
  accessBlocked: (l: Locale) => t(l, "Access blocked", "접근 차단", "حجب الوصول"),
  cleanup: (l: Locale) => t(l, "Cleanup", "정리", "التنظيف"),
  wordRunning: (l: Locale) => t(l, "In progress", "진행 중", "قيد التنفيذ"),
  wordComplete: (l: Locale) => t(l, "Complete", "완료", "مكتمل"),
  wordIncomplete: (l: Locale) => t(l, "Not finished", "미완료", "لم يكتمل"),
  nothingToClean: (l: Locale) => t(l, "Nothing to clean up", "정리할 것 없음", "لا شيء للتنظيف"),
  sessionsCount: (l: Locale, n: number) => sessions(l, n),

  countDone: (l: Locale) => t(l, "Done", "완료", "تم"),
  countCleared: (l: Locale) => t(l, "Already cleared", "이미 정리됨", "أُغلقت مسبقًا"),
  equation: (l: Locale, n: number, done: number, cleared: number, pending: number, failed: number) =>
    t(
      l,
      `${sessions(l, n)}: ${done} done, ${cleared} already cleared, ${pending} in progress, ${failed} not finished.`,
      `${sessions(l, n)}: 완료 ${done}, 이미 정리됨 ${cleared}, 진행 중 ${pending}, 미완료 ${failed}.`,
      `${sessions(l, n)}: تم ${done}، أُغلقت مسبقًا ${cleared}، قيد التنفيذ ${pending}، لم يكتمل ${failed}.`,
    ),
  heldTitle: (l: Locale, k: number) => t(l, `Started after your list (${k})`, `목록 이후 시작된 세션 (${k})`, `بدأت بعد قائمتك (${k})`),
  heldFirst: (l: Locale) => t(l, "Only the sessions in your list were ended.", "목록에 있던 세션만 종료했습니다.", "لم تُنهَ إلا الجلسات الموجودة في قائمتك."),
  heldRest: (l: Locale, k: number) =>
    t(
      l,
      plural(l, k, {
        one: "{n} session started after the list was taken and is still signed in. Review it below and end it if you do not recognize it. If you did not sign in elsewhere, change your password to sign out everywhere else.",
        other: "{n} sessions started after the list was taken and are still signed in. Review them below and end any you do not recognize. If you did not sign in elsewhere, change your password to sign out everywhere else.",
      }),
      `목록을 가져온 뒤 시작된 ${sessions(l, k)}는 로그인 상태가 유지됩니다. 아래에서 확인하고 모르는 세션은 종료하세요. 다른 곳에서 로그인한 적이 없다면 비밀번호를 변경해 다른 모든 곳에서 로그아웃하세요.`,
      `عدد الجلسات التي بدأت بعد أخذ القائمة ولا تزال مسجّلة الدخول: ${k}. راجعها أدناه وأنهِ أي جلسة لا تعرفها. إن لم تسجّل الدخول من مكان آخر، فغيّر كلمة المرور لتسجيل الخروج من كل مكان آخر.`,
    ),
  heldAnnounce: (l: Locale, k: number) =>
    t(
      l,
      plural(l, k, { one: "{n} session started after your list and is still signed in.", other: "{n} sessions started after your list and are still signed in." }),
      `목록 이후 시작된 ${sessions(l, k)}는 로그인 상태가 유지됩니다.`,
      `عدد الجلسات التي بدأت بعد قائمتك ولا تزال مسجّلة الدخول: ${k}.`,
    ),
  goPassword: (l: Locale) => t(l, "Change your password", "비밀번호 변경하기", "تغيير كلمة المرور"),
  noLonger: (l: Locale) => t(l, "No longer signed in", "더 이상 로그인되어 있지 않음", "لم تعد مسجّلة الدخول"),
  changedOnly: (l: Locale) => t(l, "Your sessions changed after the list was taken.", "목록을 가져온 뒤 세션이 바뀌었습니다.", "تغيّرت جلساتك بعد أخذ القائمة."),
  gone: (l: Locale, j: number) =>
    t(
      l,
      plural(l, j, { one: "{n} session in your list had already ended.", other: "{n} sessions in your list had already ended." }),
      `목록의 ${sessions(l, j)}는 이미 종료되어 있었습니다.`,
      `عدد جلسات قائمتك التي كانت قد انتهت مسبقًا: ${j}.`,
    ),
  inRequest: (l: Locale, n: number) => t(l, `Sessions in this request (${n})`, `이 요청의 세션 (${n})`, `الجلسات في هذا الطلب (${n})`),
  rowComplete: (l: Locale) => t(l, "Access blocked · cleanup complete", "접근 차단 · 정리 완료", "حجب الوصول · اكتمل التنظيف"),
  rowRunning: (l: Locale) => t(l, "Access blocked · cleanup in progress", "접근 차단 · 정리 진행 중", "حجب الوصول · التنظيف قيد التنفيذ"),
  rowFailed: (l: Locale) => t(l, "Access blocked · cleanup not finished", "접근 차단 · 정리 미완료", "حجب الوصول · لم يكتمل التنظيف"),
  rowGone: (l: Locale) => t(l, "Had already ended", "이미 종료되어 있었음", "كانت قد انتهت مسبقًا"),
  details: (l: Locale) => t(l, "Cleanup details for", "정리 세부 정보:", "تفاصيل تنظيف الجلسة"),
  nothingSession: (l: Locale) =>
    t(l, "Nothing to clean up: this session had already ended.", "정리할 것이 없습니다. 이 세션은 이미 종료되어 있었습니다.", "لا شيء للتنظيف: كانت هذه الجلسة قد انتهت مسبقًا."),
  groupDone: (l: Locale) => t(l, "Done", "완료", "تم"),
  groupPending: (l: Locale) => t(l, "In progress", "진행 중", "قيد التنفيذ"),
  groupFailed: (l: Locale) => t(l, "Not finished", "미완료", "لم يكتمل"),
  groupNotApplicable: (l: Locale) => t(l, "Not applicable", "해당 없음", "لا ينطبق"),
  groupOutside: (l: Locale) => t(l, "Outside the controlled scope", "통제 범위 밖", "خارج النطاق المتحكَّم فيه"),
  item: (l: Locale, id: string) =>
    ({
      C1: t(l, "Access through this session", "이 세션을 통한 접근", "الوصول عبر هذه الجلسة"),
      C2: t(l, "Recent password confirmation for this session", "이 세션의 최근 비밀번호 재확인", "تأكيد كلمة المرور الأخير لهذه الجلسة"),
      C3: t(l, "Session record in the sign-in service", "로그인 서비스의 세션 기록", "سجل الجلسة في خدمة تسجيل الدخول"),
      C4: t(l, "Secret pickups issued to this session", "이 세션에 발급된 비밀값 수령 링크", "استلامات الأسرار الصادرة لهذه الجلسة"),
      C5: t(l, "Wrong-password count for this session", "이 세션의 비밀번호 오류 횟수", "عدد محاولات كلمة المرور الخاطئة لهذه الجلسة"),
      C6: t(l, "Requests already in progress", "이미 진행 중인 요청", "الطلبات الجارية بالفعل"),
      C7: t(l, "Pages already shown in that browser", "그 브라우저에 이미 표시된 화면", "الصفحات المعروضة مسبقًا في ذلك المتصفح"),
      C8: t(l, "Jobs and API keys", "작업과 API 키", "المهام ومفاتيح API"),
    })[id] ?? id,
  reason: (l: Locale, reason: string) =>
    ({
      already_closed: t(l, "The sign-in service had already closed it.", "로그인 서비스에서 이미 종료된 기록입니다.", "كانت خدمة تسجيل الدخول قد أغلقته مسبقًا."),
      retrying: t(l, "Retrying", "다시 시도 중", "جارٍ إعادة المحاولة"),
      unexpected_error: t(l, "Stopped after an unexpected error. Access stays blocked.", "예기치 않은 오류로 멈췄습니다. 접근은 계속 차단됩니다.", "توقّف بعد خطأ غير متوقع. يبقى الوصول محجوبًا."),
      retry_budget_spent: t(l, "Stopped after 4 attempts. Access stays blocked.", "4번 시도한 뒤 멈췄습니다. 접근은 계속 차단됩니다.", "توقّف بعد 4 محاولات. يبقى الوصول محجوبًا."),
      in_flight_requests_finish: t(l, "Requests that had already started finish.", "이미 시작된 요청은 끝까지 처리됩니다.", "تكتمل الطلبات التي كانت قد بدأت."),
      cannot_recall: t(
        l,
        "They stay on screen. That browser shows the signed-out message on its next request.",
        "화면은 그대로 남습니다. 그 브라우저는 다음 요청에서 로그아웃 안내를 표시합니다.",
        "تبقى ظاهرة على الشاشة. يعرض ذلك المتصفح رسالة تسجيل الخروج عند طلبه التالي.",
      ),
      denied_set_full: t(
        l,
        "Stopped: the service's list of ended sessions waiting for cleanup is full. Access stays blocked. New sign-ins to this account are refused until 12 hours after this stop.",
        "중지됨: 서비스의 정리 대기 종료 세션 목록이 가득 찼습니다. 접근은 계속 차단됩니다. 이 계정의 새 로그인은 이 중지 시점부터 12시간 동안 거부됩니다.",
        "توقّف: قائمة الخدمة للجلسات المنتهية التي تنتظر التنظيف ممتلئة. يبقى الوصول محجوبًا. تُرفض عمليات تسجيل الدخول الجديدة لهذا الحساب لمدة 12 ساعة من هذا التوقف.",
      ),
      no_jobs_or_api_keys: t(l, "No job runs under a session, and people hold no API keys.", "세션 권한으로 실행되는 작업이 없고 사람에게는 API 키가 없습니다.", "لا تعمل أي مهمة تحت جلسة، ولا يحمل الأشخاص مفاتيح API."),
    })[reason] ?? "",
  checkedAt: (l: Locale, time: string) => t(l, `Checked at ${time}`, `${time} 확인`, `تم التحقق في ${time}`),
  stillRunning: (l: Locale, time: string) => t(l, `Cleanup was still in progress at ${time}.`, `${time} 기준 정리가 아직 진행 중이었습니다.`, `كان التنظيف لا يزال جاريًا في ${time}.`),
  checkFailed: (l: Locale, time: string) => t(l, `Could not check cleanup. Last checked at ${time}.`, `정리 상태를 확인하지 못했습니다. 마지막 확인 ${time}.`, `تعذّر التحقق من التنظيف. آخر تحقق في ${time}.`),
  checkAgain: (l: Locale) => t(l, "Check again", "다시 확인", "التحقق مجددًا"),
  tag: (l: Locale) => t(l, "Started after your list", "목록 이후 시작", "بدأت بعد قائمتك"),
};

export function stateSentence(l: Locale, op: SelfEnd): string {
  if (op.count === 0) return e.completeNone(l);
  if (op.state === "cleanup_running") return e.running(l);
  if (op.state === "cleanup_incomplete") return `${e.incomplete(l, op.count)} ${e.incompleteCount(l, op.cleanup.failed)}`;
  return e.complete(l, op.count);
}
