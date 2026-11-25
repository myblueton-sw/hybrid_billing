import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };

function plural(l: Locale, n: number, forms: Partial<Record<Intl.LDMLPluralRule, string>> & { other: string }): string {
  const rule = new Intl.PluralRules(tags[l]).select(n);
  return (forms[rule] ?? forms.other).replace("{n}", String(n));
}

const sessionsWord = (l: Locale, n: number) =>
  l === "en"
    ? plural(l, n, { one: "{n} session", other: "{n} sessions" })
    : l === "ko"
      ? `세션 ${n}개`
      : plural(l, n, { zero: "لا جلسات", one: "جلسة واحدة", two: "جلستان", few: "{n} جلسات", many: "{n} جلسة", other: "{n} جلسة" });

export const m = {
  loading: (l: Locale) => t(l, "Loading your account…", "내 계정 정보를 불러오는 중…", "جارٍ تحميل معلومات حسابك…"),
  loadFailed: (l: Locale) => t(l, "Your account could not be loaded", "내 계정 정보를 불러오지 못했습니다", "تعذّر تحميل معلومات حسابك"),
  signedOutTitle: (l: Locale) => t(l, "You are signed out", "로그아웃되었습니다", "تم تسجيل خروجك"),
  signedOutBody: (l: Locale) =>
    t(l, "This session's access has ended. Sign in again.", "현재 세션의 접근이 종료되었습니다. 다시 로그인해 주세요.", "انتهى وصول هذه الجلسة. سجّل الدخول مجددًا."),
  lockedOutBody: (l: Locale) =>
    t(
      l,
      "You were signed out after too many wrong current passwords. Your password was not changed, and password sign-in may be locked for up to 15 minutes.",
      "현재 비밀번호를 너무 많이 틀려 로그아웃되었습니다. 비밀번호는 변경되지 않았으며 비밀번호 로그인이 최대 15분간 잠길 수 있습니다.",
      "تم تسجيل خروجك بعد إدخال كلمة المرور الحالية بشكل خاطئ مرات كثيرة. لم تتغير كلمة المرور، وقد يُقفل تسجيل الدخول بكلمة المرور لمدة تصل إلى 15 دقيقة.",
    ),
  accountTitle: (l: Locale) => t(l, "Your account", "내 계정", "حسابك"),
  notRecorded: (l: Locale) => t(l, "Not recorded", "기록 없음", "غير مسجّل"),
  userId: (l: Locale) => t(l, "User ID", "사용자 ID", "معرّف المستخدم"),
  userIdSub: (l: Locale) => t(l, "Does not change", "변경되지 않음", "لا يتغير"),
  password: (l: Locale) => t(l, "Password", "비밀번호", "كلمة المرور"),
  set: (l: Locale) => t(l, "Set", "설정됨", "مُعيّنة"),
  notSet: (l: Locale) => t(l, "Not set", "설정되지 않음", "غير مُعيّنة"),
  accessTitle: (l: Locale) => t(l, "Your access in this workspace", "이 업무 공간에서의 내 권한", "صلاحياتك في مساحة العمل هذه"),
  workspace: (l: Locale) => t(l, "Workspace", "업무 공간", "مساحة العمل"),
  role: (l: Locale) => t(l, "Role", "역할", "الدور"),
  installAdmin: (l: Locale) => t(l, "Installation administrator", "설치 관리자", "مسؤول التثبيت"),
  yes: (l: Locale) => t(l, "Yes", "예", "نعم"),
  no: (l: Locale) => t(l, "No", "아니요", "لا"),
  roleNote: (l: Locale) => t(l, "Roles are changed by a workspace administrator, not here.", "역할은 이 화면이 아니라 업무 공간 관리자가 변경합니다.", "يغيّر مسؤول مساحة العمل الأدوار، وليس من هنا."),
  prelaunch: (l: Locale) => t(l, "Pre-launch:", "출시 전 서비스:", "خدمة ما قبل الإطلاق:"),
  profileNote: (l: Locale) => t(l, "changing your display name or email is not available.", "표시 이름과 이메일은 변경할 수 없습니다.", "لا يتاح تغيير اسم العرض أو البريد الإلكتروني."),
  methodsTitle: (l: Locale) => t(l, "Sign-in methods", "로그인 수단", "طرق تسجيل الدخول"),
  colMethod: (l: Locale) => t(l, "Method", "수단", "الطريقة"),
  colSignsIn: (l: Locale) => t(l, "Signs in with", "로그인 계정", "تسجيل الدخول باستخدام"),
  colStatus: (l: Locale) => t(l, "Status", "상태", "الحالة"),
  pwSet: (l: Locale) => t(l, "Password set", "비밀번호 설정됨", "كلمة المرور مُعيّنة"),
  pwNotSet: (l: Locale) => t(l, "No password set", "비밀번호 없음", "لا توجد كلمة مرور"),
  linkNote: (l: Locale) => t(l, "linking other sign-in methods is not available.", "다른 로그인 수단은 연결할 수 없습니다.", "لا يتاح ربط طرق تسجيل دخول أخرى."),
  noPassword: (l: Locale) =>
    t(
      l,
      "No password is set for this account. Ask a workspace administrator for a set-password link.",
      "이 계정에는 비밀번호가 없습니다. 업무 공간 관리자에게 비밀번호 설정 링크를 요청하세요.",
      "لا توجد كلمة مرور لهذا الحساب. اطلب من مسؤول مساحة العمل رابط تعيين كلمة المرور.",
    ),
  changeTitle: (l: Locale) => t(l, "Change password", "비밀번호 변경", "تغيير كلمة المرور"),
  backMethods: (l: Locale) => t(l, "Back to sign-in methods", "로그인 수단으로 돌아가기", "العودة إلى طرق تسجيل الدخول"),
  currentPassword: (l: Locale) => t(l, "Current password", "현재 비밀번호", "كلمة المرور الحالية"),
  differ: (l: Locale) => t(l, "It must differ from your current password.", "현재 비밀번호와 달라야 합니다.", "يجب أن تختلف عن كلمة المرور الحالية."),
  enterCurrent: (l: Locale) => t(l, "Enter your current password", "현재 비밀번호를 입력하세요", "أدخل كلمة المرور الحالية"),
  sameAsCurrent: (l: Locale) =>
    t(l, "The new password must differ from your current password", "새 비밀번호는 현재 비밀번호와 달라야 합니다", "يجب أن تختلف كلمة المرور الجديدة عن الحالية"),
  changeImpact: (l: Locale) =>
    t(
      l,
      "Changing your password signs you out everywhere else. This session stays signed in. Other browsers or devices must sign in again with the new password.",
      "비밀번호를 바꾸면 다른 모든 곳에서 로그아웃됩니다. 이 세션은 유지됩니다. 다른 브라우저나 기기에서는 새 비밀번호로 다시 로그인해야 합니다.",
      "يؤدي تغيير كلمة المرور إلى تسجيل خروجك من كل مكان آخر. تبقى هذه الجلسة مسجّلة. يجب على المتصفحات أو الأجهزة الأخرى تسجيل الدخول مجددًا بكلمة المرور الجديدة.",
    ),
  attemptsNote: (l: Locale) =>
    t(
      l,
      "Repeated wrong current passwords sign you out and can lock password sign-in for up to 15 minutes, here and on the sign-in page.",
      "현재 비밀번호를 반복해서 틀리면 로그아웃되며, 이 화면과 로그인 화면 모두에서 비밀번호 로그인이 최대 15분간 잠길 수 있습니다.",
      "يؤدي تكرار إدخال كلمة المرور الحالية بشكل خاطئ إلى تسجيل خروجك وقد يُقفل تسجيل الدخول بكلمة المرور لمدة تصل إلى 15 دقيقة، هنا وفي صفحة تسجيل الدخول.",
    ),
  changedTitle: (l: Locale) => t(l, "Password changed", "비밀번호를 변경했습니다", "تم تغيير كلمة المرور"),
  changedBody: (l: Locale) =>
    t(
      l,
      "Your other sessions were signed out. This session stays signed in.",
      "다른 세션은 모두 로그아웃되었습니다. 이 세션은 유지됩니다.",
      "تم تسجيل الخروج من جلساتك الأخرى. تبقى هذه الجلسة مسجّلة.",
    ),
  done: (l: Locale) => t(l, "Done", "완료", "تم"),
  errWrong: (l: Locale) => t(l, "The current password is not correct.", "현재 비밀번호가 올바르지 않습니다.", "كلمة المرور الحالية غير صحيحة."),
  errLocked: (l: Locale) =>
    t(
      l,
      "Password sign-in for your account is locked after repeated wrong passwords. Your password was not changed. Try again in up to 15 minutes.",
      "비밀번호를 반복해서 틀려 계정의 비밀번호 로그인이 잠겼습니다. 비밀번호는 변경되지 않았습니다. 최대 15분 후 다시 시도하세요.",
      "قُفل تسجيل الدخول بكلمة المرور لحسابك بسبب تكرار كلمات مرور خاطئة. لم تتغير كلمة المرور. أعد المحاولة خلال مدة تصل إلى 15 دقيقة.",
    ),
  errLimited: (l: Locale, s: number) =>
    s > 0
      ? t(l, `Too many password checks right now. Try again in ${s} seconds.`, `지금은 비밀번호 확인 요청이 너무 많습니다. ${s}초 후 다시 시도하세요.`, `طلبات التحقق من كلمة المرور كثيرة الآن. أعد المحاولة بعد ${s} ثانية.`)
      : t(l, "Too many password checks right now. Try again in a moment.", "지금은 비밀번호 확인 요청이 너무 많습니다. 잠시 후 다시 시도하세요.", "طلبات التحقق من كلمة المرور كثيرة الآن. أعد المحاولة بعد قليل."),
  errForm: (l: Locale) =>
    t(l, "The new password does not meet the rules.", "새 비밀번호가 규칙에 맞지 않습니다.", "كلمة المرور الجديدة لا تستوفي القواعد."),
  errConflict: (l: Locale) =>
    t(
      l,
      "Your password changed while this request was being checked, for example from a reset link. This change was not applied. Enter the password that is now current and try again.",
      "요청을 확인하는 동안 비밀번호가 다른 경로(예: 재설정 링크)로 바뀌었습니다. 이 변경은 적용되지 않았습니다. 지금 유효한 비밀번호를 입력해 다시 시도하세요.",
      "تغيّرت كلمة المرور أثناء التحقق من هذا الطلب، مثلًا عبر رابط إعادة تعيين. لم يُطبَّق هذا التغيير. أدخل كلمة المرور السارية الآن وأعد المحاولة.",
    ),
  changeUnknown: (l: Locale) =>
    t(
      l,
      "We could not confirm whether your password changed; it may already have been changed and your other sessions signed out. Before trying again, reload this page and try the new password if you are asked to sign in.",
      "비밀번호가 변경되었는지 확인하지 못했습니다. 이미 변경되어 다른 세션이 로그아웃되었을 수 있습니다. 다시 시도하기 전에 이 화면을 다시 불러오고, 로그인을 요구받으면 새 비밀번호로 먼저 시도하세요.",
      "تعذّر التأكد مما إذا تغيّرت كلمة المرور؛ ربما تغيّرت بالفعل وسُجّل الخروج من جلساتك الأخرى. قبل إعادة المحاولة، أعد تحميل هذه الصفحة وجرّب كلمة المرور الجديدة إذا طُلب منك تسجيل الدخول.",
    ),
  loadRetry: (l: Locale) => t(l, "Try again shortly.", "잠시 후 다시 시도하세요.", "أعد المحاولة بعد قليل."),
  errBusy: (l: Locale) =>
    t(l, "Your password could not be checked right now. Nothing was changed. Try again shortly.", "지금은 비밀번호를 확인할 수 없습니다. 변경된 내용은 없습니다. 잠시 후 다시 시도하세요.", "تعذّر التحقق من كلمة المرور الآن. لم يتغير شيء. أعد المحاولة بعد قليل."),
  sessionsTitle: (l: Locale) => t(l, "Sessions", "세션", "الجلسات"),
  count: sessionsWord,
  otherSession: (l: Locale) => t(l, "Other session", "다른 세션", "جلسة أخرى"),
  age: (l: Locale, seconds: number) => {
    const minutes = Math.round(seconds / 60);
    if (minutes % 60 === 0) {
      const h = minutes / 60;
      return l === "en" ? plural(l, h, { one: "{n} hour", other: "{n} hours" }) : l === "ko" ? `${h}시간` : `${h} ساعة`;
    }
    return l === "en" ? plural(l, minutes, { one: "{n} minute", other: "{n} minutes" }) : l === "ko" ? `${minutes}분` : `${minutes} دقيقة`;
  },
  thisSession: (l: Locale) => t(l, "This session", "이 세션", "هذه الجلسة"),
  signedInAt: (l: Locale, when: string) => t(l, `Signed in ${when}`, `${when} 로그인`, `تسجيل الدخول ${when}`),
  reference: (l: Locale) => t(l, "Reference", "참조", "المرجع"),
  colSession: (l: Locale) => t(l, "Session", "세션", "الجلسة"),
  colSignedIn: (l: Locale) => t(l, "Signed in", "로그인 시각", "وقت تسجيل الدخول"),
  colEnds: (l: Locale) => t(l, "Ends by", "종료 예정", "تنتهي في موعد أقصاه"),
  colActions: (l: Locale) => t(l, "Actions", "작업", "الإجراءات"),
  signOut: (l: Locale) => t(l, "Sign out", "로그아웃", "تسجيل الخروج"),
  end: (l: Locale) => t(l, "End", "종료", "إنهاء"),
  noOthers: (l: Locale) => t(l, "You have no other sessions.", "다른 세션이 없습니다.", "لا توجد لديك جلسات أخرى."),
  staleNote: (l: Locale, idle: number) =>
    t(
      l,
      `Sessions are listed until 12 hours after sign-in. A session idle for ${idle} minutes has already ended but may still appear here. Ending it does no harm.`,
      `세션은 로그인 후 12시간까지 표시됩니다. ${idle}분 동안 사용하지 않은 세션은 이미 종료되었더라도 여기에 남아 있을 수 있습니다. 종료해도 문제없습니다.`,
      `تُعرض الجلسات حتى 12 ساعة بعد تسجيل الدخول. الجلسة الخاملة لمدة ${idle} دقيقة تكون قد انتهت لكنها قد تظهر هنا. إنهاؤها لا يضر.`,
    ),
  endOneTitle: (l: Locale) => t(l, "End this session?", "이 세션을 종료할까요?", "هل تريد إنهاء هذه الجلسة؟"),
  endOneImpact: (l: Locale) =>
    t(
      l,
      "Whoever uses that session is signed out on their next request. This session is not affected. To use it again, that browser must sign in again.",
      "그 세션을 쓰는 사람은 다음 요청에서 로그아웃됩니다. 이 세션에는 영향이 없습니다. 그 브라우저에서 다시 쓰려면 다시 로그인해야 합니다.",
      "يُسجَّل خروج من يستخدم تلك الجلسة عند طلبه التالي. لا تتأثر هذه الجلسة. لاستخدامها مجددًا يجب تسجيل الدخول من ذلك المتصفح.",
    ),
  endOneButton: (l: Locale) => t(l, "End session", "세션 종료", "إنهاء الجلسة"),
  endGone: (l: Locale) =>
    t(l, "That session has already ended. The list has been reloaded.", "해당 세션은 이미 종료되었습니다. 목록을 다시 불러왔습니다.", "انتهت تلك الجلسة مسبقًا. أُعيد تحميل القائمة."),
  endLimited: (l: Locale, s: number) =>
    s > 0
      ? t(l, `Too many requests right now. Try again in ${s} seconds.`, `지금은 요청이 너무 많습니다. ${s}초 후 다시 시도하세요.`, `الطلبات كثيرة الآن. أعد المحاولة بعد ${s} ثانية.`)
      : t(l, "Too many requests right now. Try again in a moment.", "지금은 요청이 너무 많습니다. 잠시 후 다시 시도하세요.", "الطلبات كثيرة الآن. أعد المحاولة بعد قليل."),
  limitsTitle: (l: Locale) => t(l, "Session limits", "세션 제한", "حدود الجلسة"),
  idle: (l: Locale) => t(l, "Inactivity limit", "비활성 제한", "حد عدم النشاط"),
  minutes: (l: Locale, n: number) => t(l, `${n} minutes`, `${n}분`, `${n} دقيقة`),
  longest: (l: Locale) => t(l, "Longest session", "최대 세션", "أطول جلسة"),
  longestValue: (l: Locale, minutes: number) => { const d = m.age(l, minutes * 60); return t(l, `${d} after sign-in`, `로그인 후 ${d}`, `${d} بعد تسجيل الدخول`); },
  maxAge: (l: Locale) =>
    t(
      l,
      "Workspace sign-in age (from the last sign-in or password confirmation)",
      "업무 공간 재로그인 주기(마지막 로그인 또는 비밀번호 재확인 기준)",
      "عمر تسجيل الدخول في مساحة العمل (من آخر تسجيل دخول أو تأكيد لكلمة المرور)",
    ),
  lastConfirmed: (l: Locale) => t(l, "Password last confirmed", "마지막 비밀번호 확인 시각", "آخر تأكيد لكلمة المرور"),
  noMaxAge: (l: Locale) => t(l, "No workspace limit", "업무 공간 제한 없음", "لا يوجد حد لمساحة العمل"),
  youSignedIn: (l: Locale) => t(l, "You signed in at", "로그인 시각", "سجّلت الدخول في"),
  recoveryTitle: (l: Locale) => t(l, "Two-step verification and recovery", "2단계 인증과 계정 복구", "التحقق بخطوتين واستعادة الحساب"),
  recoveryBody: (l: Locale) =>
    t(
      l,
      "Two-step verification and recovery methods are not available in pre-launch. You sign in with your password only.",
      "출시 전 서비스에서는 2단계 인증과 복구 수단을 사용할 수 없습니다. 비밀번호로만 로그인합니다.",
      "التحقق بخطوتين وطرق الاستعادة غير متاحة في خدمة ما قبل الإطلاق. تسجّل الدخول بكلمة المرور فقط.",
    ),
  recoveryForgot: (l: Locale) =>
    t(l, "If you forget your password, ask a workspace administrator for a reset link.", "비밀번호를 잊으면 업무 공간 관리자에게 재설정 링크를 요청하세요.", "إذا نسيت كلمة المرور، اطلب من مسؤول مساحة العمل رابط إعادة تعيين."),
  notifTitle: (l: Locale) => t(l, "Notification preferences", "알림 설정", "تفضيلات الإشعارات"),
  notifBody: (l: Locale) =>
    t(l, "Notification preferences are not available in pre-launch.", "출시 전 서비스에서는 알림 설정을 사용할 수 없습니다.", "تفضيلات الإشعارات غير متاحة في خدمة ما قبل الإطلاق."),
  goSessions: (l: Locale) => t(l, "Go to sessions", "세션으로 이동", "الانتقال إلى الجلسات"),
  goAccount: (l: Locale) => t(l, "Go to your account", "내 계정으로 이동", "الانتقال إلى حسابك"),
};
