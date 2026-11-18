import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

export type Action =
  | "suspend_user"
  | "reinstate_user"
  | "depart_user"
  | "change_role"
  | "suspend_membership"
  | "reinstate_membership"
  | "remove_membership"
  | "invite"
  | "issue_set_link"
  | "issue_reset_link"
  | "revoke_invitation"
  | "set_role_period";

const plural = (l: Locale, n: number) => {
  if (l === "en") return n === 1 ? "1 user" : `${n} users`;
  if (l === "ko") return `사용자 ${n}명`;
  const rule = new Intl.PluralRules("ar").select(n);
  if (n === 1) return "مستخدم واحد";
  if (n === 2) return "مستخدمان";
  return rule === "few" ? `${n} مستخدمين` : `${n} مستخدمًا`;
};

const invitationCount = (l: Locale, n: number) => {
  if (l === "en") return n === 1 ? "1 invitation" : `${n} invitations`;
  if (l === "ko") return `초대 ${n}건`;
  if (n === 1) return "دعوة واحدة";
  if (n === 2) return "دعوتان";
  return new Intl.PluralRules("ar").select(n) === "few" ? `${n} دعوات` : `${n} دعوة`;
};

export const w = {
  invitationCount,
  usersTitle: (l: Locale) => t(l, "Users", "사용자", "المستخدمون"),
  invitationsTitle: (l: Locale) => t(l, "Pending invitations", "대기 중인 초대", "الدعوات بانتظار القبول"),
  membershipTitle: (l: Locale) => t(l, "Workspace membership", "업무 공간 소속", "العضوية في مساحة العمل"),
  count: plural,
  loadedAt: (l: Locale, time: string) => t(l, `Loaded at ${time}`, `${time}에 불러옴`, `تم التحميل في ${time}`),
  reload: (l: Locale) => t(l, "Reload", "다시 불러오기", "إعادة التحميل"),
  create: (l: Locale) => t(l, "Create user", "사용자 추가", "إضافة مستخدم"),
  more: (l: Locale) => t(l, "Show more users", "사용자 더 보기", "عرض المزيد من المستخدمين"),
  openDetails: (l: Locale) => t(l, "Open details", "상세 열기", "فتح التفاصيل"),
  selected: (l: Locale) => t(l, "(open)", "(열림)", "(مفتوح)"),
  user: (l: Locale) => t(l, "User", "사용자", "المستخدم"),
  accountStatus: (l: Locale) => t(l, "Account status", "계정 상태", "حالة الحساب"),
  workspaceRole: (l: Locale) => t(l, "Workspace role", "업무 공간 역할", "الدور في مساحة العمل"),
  signInSetup: (l: Locale) => t(l, "Sign-in setup", "로그인 설정", "إعداد تسجيل الدخول"),
  role: (l: Locale) => t(l, "Role", "역할", "الدور"),
  invitationExpires: (l: Locale) => t(l, "Invitation expires", "초대 만료", "انتهاء الدعوة"),
  setLink: (l: Locale) => t(l, "Set link", "설정 링크", "رابط التعيين"),
  membershipStatus: (l: Locale) => t(l, "Membership status", "소속 상태", "حالة العضوية"),
  installationAdmin: (l: Locale) => t(l, "Installation administrator", "설치 관리자", "مسؤول التثبيت"),
  userId: (l: Locale) => t(l, "User ID", "사용자 ID", "معرّف المستخدم"),
  active: (l: Locale) => t(l, "Active", "활성", "نشط"),
  suspended: (l: Locale) => t(l, "Suspended", "정지됨", "معلّق"),
  departed: (l: Locale) => t(l, "Departed", "퇴사", "مغادر"),
  administrator: (l: Locale) => t(l, "Administrator", "관리자", "مسؤول"),
  member: (l: Locale) => t(l, "Member", "멤버", "عضو"),
  noMembership: (l: Locale) => t(l, "No membership", "소속 없음", "بلا عضوية"),
  membershipSuspended: (l: Locale) => t(l, "Membership suspended", "소속 정지됨", "العضوية معلّقة"),
  yes: (l: Locale) => t(l, "Yes", "예", "نعم"),
  no: (l: Locale) => t(l, "No", "아니요", "لا"),
  passwordSet: (l: Locale) => t(l, "Password set", "비밀번호 설정됨", "تم تعيين كلمة المرور"),
  invitationPending: (l: Locale, time: string) =>
    t(l, `Invitation pending until ${time}`, `초대 대기 중 · ${time}까지`, `دعوة بانتظار القبول حتى ${time}`),
  setIssued: (l: Locale, time: string) =>
    t(l, `Set link issued, expires ${time}`, `설정 링크 발급됨 · ${time} 만료`, `صدر رابط التعيين، ينتهي في ${time}`),
  resetIssued: (l: Locale, time: string) =>
    t(l, `Reset link issued, expires ${time}`, `재설정 링크 발급됨 · ${time} 만료`, `صدر رابط إعادة التعيين، ينتهي في ${time}`),
  notIssued: (l: Locale) => t(l, "Not issued", "미발급", "لم يُصدر"),
  noPassword: (l: Locale) => t(l, "No password yet", "비밀번호 미설정", "لم تُعيَّن كلمة مرور بعد"),
  pick: (l: Locale) =>
    t(l, "Select a user to see details and actions.", "사용자를 선택하면 상세 정보와 작업이 표시됩니다.", "اختر مستخدمًا لعرض التفاصيل والإجراءات."),
  back: (l: Locale) => t(l, "Back to users", "사용자 목록으로", "العودة إلى المستخدمين"),
  account: (l: Locale) => t(l, "Account", "계정", "الحساب"),
  sign: (l: Locale) => t(l, "Sign-in", "로그인", "تسجيل الدخول"),
  cancel: (l: Locale) => t(l, "Cancel", "취소", "إلغاء"),
  saving: (l: Locale) => t(l, "Saving…", "저장 중…", "جارٍ الحفظ…"),
  reason: (l: Locale) => t(l, "Reason", "사유", "السبب"),
  bytes: (l: Locale, n: number) => t(l, `${n}/256 bytes`, `${n}/256바이트`, `${n}/256 بايت`),
  reasonHint: (l: Locale) =>
    t(
      l,
      "Recorded with this change. Up to 256 bytes (about 256 Latin letters).",
      "이 변경과 함께 기록됩니다. 256바이트까지 입력할 수 있습니다(한글 약 85자).",
      "يُسجَّل مع هذا التغيير. حتى 256 بايت، أي نحو 128 حرفًا عربيًا.",
    ),
  enterReason: (l: Locale) => t(l, "Enter a reason", "사유를 입력하세요", "أدخل السبب"),
  longReason: (l: Locale) =>
    t(l, "Shorten the reason to 256 bytes or fewer", "사유를 256바이트 이하로 줄이세요", "اختصر السبب إلى 256 بايت أو أقل"),
  selfReason: (l: Locale) =>
    t(l, "You cannot change your own account here.", "본인 계정은 여기에서 변경할 수 없습니다.", "لا يمكنك تغيير حسابك من هنا."),
  departedReason: (l: Locale) =>
    t(l, "This account has departed. No further changes are available.", "퇴사 처리된 계정이므로 더 이상 변경할 수 없습니다.", "هذا الحساب مغادر، ولا تتوفر أي تغييرات أخرى."),
  adminResetReason: (l: Locale) =>
    t(l, "Password reset links cannot be issued for administrators.", "관리자에게는 비밀번호 재설정 링크를 발급할 수 없습니다.", "لا يمكن إصدار روابط إعادة تعيين كلمة المرور للمسؤولين."),
  promotionReason: (l: Locale) =>
    t(
      l,
      "Cannot change to administrator while an unused link is outstanding. Try again after the link is used or expires.",
      "사용하지 않은 링크가 남아 있는 동안에는 관리자로 변경할 수 없습니다. 링크가 사용되거나 만료된 뒤 다시 시도하세요.",
      "لا يمكن التغيير إلى مسؤول ما دام هناك رابط لم يُستخدم. حاول مجددًا بعد استخدام الرابط أو انتهاء صلاحيته.",
    ),
  suspendedResetReason: (l: Locale) =>
    t(
      l,
      "Reinstate the account before issuing a password reset link.",
      "비밀번호 재설정 링크를 발급하려면 먼저 계정 정지를 해제하세요.",
      "أعد تفعيل الحساب قبل إصدار رابط إعادة تعيين كلمة المرور.",
    ),
  createLead: (l: Locale) =>
    t(
      l,
      "The person signs in with the sign-in email and sets a password through a one-time link that you hand over.",
      "사용자는 로그인 이메일로 로그인하며, 관리자가 직접 전달하는 일회용 링크로 비밀번호를 설정합니다.",
      "يسجّل الشخص الدخول بالبريد الإلكتروني لتسجيل الدخول، ويعيّن كلمة المرور عبر رابط لمرة واحدة تسلّمه إليه بنفسك.",
    ),
  creating: (l: Locale) => t(l, "Creating…", "추가 중…", "جارٍ الإضافة…"),
  loginRefused: (l: Locale) =>
    t(
      l,
      "This sign-in email cannot be used. It may already belong to another user.",
      "이 로그인 이메일은 사용할 수 없습니다. 다른 사용자가 이미 사용 중일 수 있습니다.",
      "لا يمكن استخدام هذا البريد الإلكتروني لتسجيل الدخول، فقد يكون مستخدمًا لدى مستخدم آخر.",
    ),
  copySet: (l: Locale) => t(l, "Copy the password set link", "비밀번호 설정 링크 복사", "انسخ رابط تعيين كلمة المرور"),
  copyReset: (l: Locale) => t(l, "Copy the password reset link", "비밀번호 재설정 링크 복사", "انسخ رابط إعادة تعيين كلمة المرور"),
  forUser: (l: Locale, login: string) => t(l, `For ${login}`, `${login} 사용자용`, `للمستخدم ${login}`),
  oneTime: (l: Locale) => t(l, "One-time link", "일회용 링크", "رابط لمرة واحدة"),
  copy: (l: Locale) => t(l, "Copy link", "링크 복사", "نسخ الرابط"),
  copied: (l: Locale) => t(l, "Link copied", "링크를 복사했습니다", "تم نسخ الرابط"),
  copyFailed: (l: Locale) =>
    t(l, "Could not copy. Select the link and copy it manually.", "복사하지 못했습니다. 링크를 선택해 직접 복사하세요.", "تعذّر النسخ. حدّد الرابط وانسخه يدويًا."),
  linkExpires: (l: Locale, time: string) =>
    t(l, `The link stops working at ${time}.`, `링크는 ${time}에 만료됩니다.`, `تنتهي صلاحية الرابط في ${time}.`),
  shownOnce: (l: Locale) =>
    t(
      l,
      "This link is shown only once. It disappears when you select Done or leave this page, and it is not saved anywhere.",
      "이 링크는 한 번만 표시됩니다. 완료를 누르거나 이 화면을 벗어나면 사라지며 어디에도 저장되지 않습니다.",
      "يُعرض هذا الرابط مرة واحدة فقط. يختفي عند اختيار «تم» أو مغادرة هذه الصفحة، ولا يُحفظ في أي مكان.",
    ),
  handOver: (l: Locale, login: string) =>
    t(
      l,
      `Give it to ${login} through a channel you trust. Hybrid Billing does not send email.`,
      `신뢰할 수 있는 경로로 ${login} 사용자에게 직접 전달하세요. Hybrid Billing은 이메일을 보내지 않습니다.`,
      `سلّمه إلى ${login} عبر قناة تثق بها. لا يرسل Hybrid Billing أي بريد إلكتروني.`,
    ),
  showLink: (l: Locale) => t(l, "Show link", "링크 표시", "عرض الرابط"),
  done: (l: Locale) => t(l, "Done", "완료", "تم"),
  linkGone: (l: Locale) =>
    t(
      l,
      "This link can no longer be shown. Issue a new link from the user's details.",
      "이 링크는 더 이상 표시할 수 없습니다. 사용자 상세에서 새 링크를 발급하세요.",
      "لم يعد بالإمكان عرض هذا الرابط. أصدر رابطًا جديدًا من تفاصيل المستخدم.",
    ),
  created: (l: Locale) =>
    t(l, "User created. Copy the password set link below.", "사용자를 추가했습니다. 아래 비밀번호 설정 링크를 복사하세요.", "تمت إضافة المستخدم. انسخ رابط تعيين كلمة المرور أدناه."),
  loading: (l: Locale) => t(l, "Loading users…", "사용자 목록을 불러오는 중…", "جارٍ تحميل المستخدمين…"),
  empty: (l: Locale) => t(l, "No users yet.", "아직 사용자가 없습니다.", "لا يوجد مستخدمون بعد."),
  noInvitations: (l: Locale) => t(l, "No pending invitations.", "대기 중인 초대가 없습니다.", "لا توجد دعوات بانتظار القبول."),
  loadFailed: (l: Locale) => t(l, "Users could not be loaded", "사용자 목록을 불러오지 못했습니다", "تعذّر تحميل المستخدمين"),
  unavailable: (l: Locale) =>
    t(l, "The user service is not responding. Try again shortly.", "사용자 서비스가 응답하지 않습니다. 잠시 후 다시 시도해 주세요.", "خدمة المستخدمين لا تستجيب. أعد المحاولة بعد قليل."),
  ended: (l: Locale) =>
    t(l, "Your session has ended. Sign in again to continue.", "세션이 종료되었습니다. 계속하려면 다시 로그인하세요.", "انتهت جلستك. سجّل الدخول مرة أخرى للمتابعة."),
  denied: (l: Locale) =>
    t(
      l,
      "You do not have permission to manage users in this installation.",
      "이 설치 환경에서 사용자를 관리할 권한이 없습니다.",
      "ليست لديك صلاحية إدارة المستخدمين في هذا التثبيت.",
    ),
  conflict: (l: Locale) =>
    t(
      l,
      "The user list changed after it was loaded, so nothing was saved. The list has been reloaded. Check the details, then try again.",
      "목록을 불러온 뒤 사용자 목록이 바뀌어 저장하지 않았습니다. 목록을 다시 불러왔으니 상태를 확인한 뒤 다시 시도하세요.",
      "تغيّرت قائمة المستخدمين بعد تحميلها، فلم يُحفظ شيء. أُعيد تحميل القائمة. تحقّق من التفاصيل ثم حاول مرة أخرى.",
    ),
  reloadUsers: (l: Locale) => t(l, "Reload users", "사용자 다시 불러오기", "إعادة تحميل المستخدمين"),
  saveFailed: (l: Locale) => t(l, "Could not save this change", "변경을 저장하지 못했습니다", "تعذّر حفظ هذا التغيير"),
  deniedTitle: (l: Locale) => t(l, "You can't manage users here", "여기에서는 사용자를 관리할 수 없습니다", "لا يمكنك إدارة المستخدمين هنا"),
  busy: (l: Locale) =>
    t(
      l,
      "The user service could not complete this change. Try again shortly. If it keeps failing, pending invitations or links may have reached their limit, or the decision ledger can't record decisions: reinstating people and giving the administrator role then wait until the decision ledger can record again (see Roles, delegation and effective access). Suspending people still works.",
      "사용자 서비스가 이 변경을 완료하지 못했습니다. 잠시 후 다시 시도하세요. 계속 실패하면 대기 중인 초대나 링크가 한도에 도달했거나 결정 원장이 결정을 기록할 수 없는 상태일 수 있습니다. 그 경우 계정 정지 해제와 관리자 역할 부여는 결정 원장이 다시 기록할 수 있을 때까지 기다려야 합니다('역할·위임·효력' 참고). 사용자 정지는 계속 할 수 있습니다.",
      "تعذّر على خدمة المستخدمين إكمال هذا التغيير. أعد المحاولة بعد قليل. إذا استمر الإخفاق، فربما بلغت الدعوات أو الروابط التي لم تُستخدم بعد حدّها، أو أن دفتر القرارات لا يستطيع تسجيل القرارات؛ وعندها تنتظر إعادة تفعيل الأشخاص ومنح دور المسؤول حتى يتمكن دفتر القرارات من التسجيل مجددًا (راجع «الأدوار والتفويض والوصول الفعلي»). يظل تعليق الأشخاص ممكنًا.",
    ),
  stale: (l: Locale) =>
    t(
      l,
      "This request was already used. The list has been reloaded. Check the details, then try again.",
      "이미 사용된 요청입니다. 목록을 다시 불러왔으니 상태를 확인한 뒤 다시 시도하세요.",
      "سبق استخدام هذا الطلب. أُعيد تحميل القائمة. تحقّق من التفاصيل ثم حاول مرة أخرى.",
    ),
  refusedTitle: (l: Locale) =>
    t(l, "This change was refused to protect access", "접근 보호를 위해 이 변경이 거부되었습니다", "رُفض هذا التغيير لحماية الوصول"),
  refused: (l: Locale) =>
    t(
      l,
      "Changes are refused when they would affect your own account, leave the workspace without an administrator, remove the last installation administrator's access, reset an administrator's password by link, or promote someone who still has an unused link.",
      "본인 계정에 영향을 주거나, 업무 공간에 관리자가 남지 않게 되거나, 마지막 설치 관리자의 접근을 없애거나, 관리자의 비밀번호를 링크로 재설정하거나, 사용하지 않은 링크가 남은 사용자를 관리자로 바꾸는 변경은 거부됩니다.",
      "تُرفض التغييرات التي تمس حسابك أنت، أو تترك مساحة العمل بلا مسؤول، أو تزيل وصول آخر مسؤول تثبيت، أو تعيد تعيين كلمة مرور مسؤول عبر رابط، أو ترقّي مستخدمًا لديه رابط لم يُستخدم بعد.",
    ),
  stage: (l: Locale) =>
    t(
      l,
      "This user's current state does not allow this change. The list has been reloaded; check the current details.",
      "이 사용자의 현재 상태에서는 이 변경을 할 수 없습니다. 목록을 다시 불러왔으니 현재 상태를 확인하세요.",
      "لا تسمح الحالة الحالية لهذا المستخدم بهذا التغيير. أُعيد تحميل القائمة؛ تحقّق من التفاصيل الحالية.",
    ),
  partial: (l: Locale, op: string) =>
    t(
      l,
      `The change may have been only partly applied. Reload the list and check this user before trying again. Reference: ${op}`,
      `변경이 일부만 적용되었을 수 있습니다. 다시 시도하기 전에 목록을 다시 불러와 이 사용자의 상태를 확인하세요. 참조 번호: ${op}`,
      `ربما طُبّق التغيير جزئيًا فقط. أعد تحميل القائمة وتحقّق من هذا المستخدم قبل المحاولة مرة أخرى. المرجع: ${op}`,
    ),
  goUsers: (l: Locale) => t(l, "Go to Users", "사용자 탭으로 이동", "الانتقال إلى المستخدمين"),
  signInAgain: (l: Locale) => t(l, "Sign in again", "다시 로그인", "تسجيل الدخول مجددًا"),
};

export const actionLabel: Record<Action, (l: Locale) => string> = {
  suspend_user: (l) => t(l, "Suspend account", "계정 정지", "تعليق الحساب"),
  reinstate_user: (l) => t(l, "Reinstate account", "계정 정지 해제", "إعادة تفعيل الحساب"),
  depart_user: (l) => t(l, "Mark as departed", "퇴사 처리", "تحديد كمغادر"),
  change_role: (l) => t(l, "Change role", "역할 변경", "تغيير الدور"),
  suspend_membership: (l) => t(l, "Suspend membership", "소속 정지", "تعليق العضوية"),
  reinstate_membership: (l) => t(l, "Reinstate membership", "소속 정지 해제", "إعادة تفعيل العضوية"),
  remove_membership: (l) => t(l, "Remove from workspace", "업무 공간에서 제거", "إزالة من مساحة العمل"),
  invite: (l) => t(l, "Invite to workspace", "업무 공간에 초대", "دعوة إلى مساحة العمل"),
  issue_set_link: (l) => t(l, "Issue password set link", "비밀번호 설정 링크 발급", "إصدار رابط تعيين كلمة المرور"),
  issue_reset_link: (l) => t(l, "Issue password reset link", "비밀번호 재설정 링크 발급", "إصدار رابط إعادة تعيين كلمة المرور"),
  revoke_invitation: (l) => t(l, "Revoke invitation", "초대 취소", "إلغاء الدعوة"),
  set_role_period: (l) => t(l, "Change role period", "역할 기간 변경", "تغيير مدة الدور"),
};

export const actionDone: Record<Action, (l: Locale) => string> = {
  suspend_user: (l) => t(l, "Account suspended", "계정을 정지했습니다", "تم تعليق الحساب"),
  reinstate_user: (l) => t(l, "Account reinstated", "계정 정지를 해제했습니다", "تمت إعادة تفعيل الحساب"),
  depart_user: (l) => t(l, "Marked as departed", "퇴사 처리했습니다", "تم التحديد كمغادر"),
  change_role: (l) => t(l, "Role changed", "역할을 변경했습니다", "تم تغيير الدور"),
  suspend_membership: (l) => t(l, "Membership suspended", "소속을 정지했습니다", "تم تعليق العضوية"),
  reinstate_membership: (l) => t(l, "Membership reinstated", "소속 정지를 해제했습니다", "تمت إعادة تفعيل العضوية"),
  remove_membership: (l) => t(l, "Removed from workspace", "업무 공간에서 제거했습니다", "تمت الإزالة من مساحة العمل"),
  invite: (l) => t(l, "Invited to workspace", "업무 공간에 초대했습니다", "تمت الدعوة إلى مساحة العمل"),
  issue_set_link: (l) => t(l, "Link issued. Copy it below.", "링크를 발급했습니다. 아래에서 복사하세요.", "صدر الرابط. انسخه أدناه."),
  issue_reset_link: (l) => t(l, "Link issued. Copy it below.", "링크를 발급했습니다. 아래에서 복사하세요.", "صدر الرابط. انسخه أدناه."),
  revoke_invitation: (l) => t(l, "Invitation revoked", "초대를 취소했습니다", "تم إلغاء الدعوة"),
  set_role_period: (l) => t(l, "Role period changed", "역할 기간을 변경했습니다", "تم تغيير مدة الدور"),
};

export function impact(l: Locale, action: Action, login: string, role: string, next: string): string {
  switch (action) {
    case "set_role_period":
      return "";
    case "suspend_user":
      return t(
        l,
        `${login} is signed out and cannot sign in until the account is reinstated. Their membership and role are kept.`,
        `${login} 사용자는 즉시 로그아웃되며 계정 정지가 해제될 때까지 로그인할 수 없습니다. 소속과 역할은 유지됩니다.`,
        `يُسجَّل خروج ${login} ولن يتمكن من تسجيل الدخول حتى إعادة تفعيل الحساب. تبقى العضوية والدور كما هما.`,
      );
    case "reinstate_user":
      return t(l, `${login} can sign in again with their existing role.`, `${login} 사용자가 기존 역할로 다시 로그인할 수 있습니다.`, `سيتمكن ${login} من تسجيل الدخول مجددًا بدوره الحالي.`);
    case "depart_user":
      return t(
        l,
        `This cannot be undone. ${login} can no longer sign in, and the account stays in the list as departed.`,
        `되돌릴 수 없습니다. ${login} 사용자는 더 이상 로그인할 수 없으며 계정은 퇴사 상태로 목록에 남습니다.`,
        `لا يمكن التراجع عن ذلك. لن يتمكن ${login} من تسجيل الدخول بعد الآن، ويبقى الحساب في القائمة بحالة مغادر.`,
      );
    case "change_role":
      return t(
        l,
        `${login}'s role in this workspace changes from ${role} to ${next}.`,
        `이 업무 공간에서 ${login} 사용자의 역할이 ${role}에서 ${next}(으)로 바뀝니다.`,
        `يتغيّر دور ${login} في مساحة العمل هذه من ${role} إلى ${next}.`,
      );
    case "suspend_membership":
      return t(
        l,
        `${login} keeps the account but cannot work in this workspace until the membership is reinstated.`,
        `${login} 사용자의 계정은 유지되지만 소속 정지가 해제될 때까지 이 업무 공간을 사용할 수 없습니다.`,
        `يحتفظ ${login} بحسابه، لكنه لا يستطيع العمل في مساحة العمل هذه حتى إعادة تفعيل العضوية.`,
      );
    case "reinstate_membership":
      return t(
        l,
        `${login} can work in this workspace again as ${role}.`,
        `${login} 사용자가 ${role}(으)로 이 업무 공간을 다시 사용할 수 있습니다.`,
        `سيتمكن ${login} من العمل في مساحة العمل هذه مجددًا بصفة ${role}.`,
      );
    case "remove_membership":
      return t(
        l,
        `${login} is removed from this workspace. The account remains, and adding them again needs a new invitation.`,
        `${login} 사용자를 이 업무 공간에서 제거합니다. 계정은 남아 있으며, 다시 추가하려면 새로 초대해야 합니다.`,
        `ستتم إزالة ${login} من مساحة العمل هذه. يبقى الحساب، وتتطلب إضافته مجددًا دعوة جديدة.`,
      );
    case "invite":
      return t(
        l,
        `${login} is invited to this workspace as ${next}. The invitation is accepted when they next sign in.`,
        `${login} 사용자를 ${next}(으)로 이 업무 공간에 초대합니다. 다음에 로그인할 때 초대가 수락됩니다.`,
        `ستتم دعوة ${login} إلى مساحة العمل هذه بصفة ${next}. تُقبل الدعوة عند تسجيل دخوله التالي.`,
      );
    case "issue_set_link":
      return t(
        l,
        `You will copy a one-time link and give it to ${login} yourself. Hybrid Billing does not send email. Any earlier link stops working.`,
        `일회용 링크를 복사해 ${login} 사용자에게 직접 전달해야 합니다. Hybrid Billing은 이메일을 보내지 않습니다. 이전에 발급한 링크는 더 이상 작동하지 않습니다.`,
        `ستنسخ رابطًا لمرة واحدة وتسلّمه إلى ${login} بنفسك. لا يرسل Hybrid Billing أي بريد إلكتروني. يتوقف أي رابط سابق عن العمل.`,
      );
    case "issue_reset_link":
      return t(
        l,
        `You will copy a one-time link and give it to ${login} yourself. When they set a new password with it, all their current sessions end.`,
        `일회용 링크를 복사해 ${login} 사용자에게 직접 전달해야 합니다. 이 링크로 새 비밀번호를 설정하면 해당 사용자의 현재 세션이 모두 종료됩니다.`,
        `ستنسخ رابطًا لمرة واحدة وتسلّمه إلى ${login} بنفسك. عند تعيين كلمة مرور جديدة به، تنتهي جميع جلساته الحالية.`,
      );
    case "revoke_invitation":
      return t(
        l,
        `The pending invitation for ${login} is cancelled. If they still use their set link, they get a password but no workspace membership.`,
        `${login} 사용자의 대기 중인 초대가 취소됩니다. 이후 설정 링크를 사용하면 비밀번호는 설정되지만 업무 공간 소속은 생기지 않습니다.`,
        `تُلغى الدعوة التي لم تُقبل بعد الموجّهة إلى ${login}. إذا استخدم رابط التعيين لاحقًا، يحصل على كلمة مرور دون عضوية في مساحة العمل.`,
      );
  }
}
