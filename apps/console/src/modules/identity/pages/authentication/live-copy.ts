import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

export const p = {
  currentTitle: (l: Locale) => t(l, "Current policy", "현재 정책", "السياسة الحالية"),
  revision: (l: Locale) => t(l, "Policy revision", "정책 개정 번호", "رقم مراجعة السياسة"),
  methods: (l: Locale) => t(l, "Sign-in methods", "로그인 방식", "طرق تسجيل الدخول"),
  passwordOnly: (l: Locale) => t(l, "Password only", "비밀번호만", "كلمة المرور فقط"),
  mfa: (l: Locale) => t(l, "Multi-factor authentication (MFA)", "다단계 인증(MFA)", "المصادقة متعددة العوامل (MFA)"),
  notRequired: (l: Locale) => t(l, "Not required", "요구하지 않음", "غير مطلوبة"),
  maxAge: (l: Locale) => t(l, "Sign-in valid for", "로그인 유지 시간", "مدة صلاحية تسجيل الدخول"),
  stepUp: (l: Locale) => t(l, "Recent sign-in for policy changes", "정책 변경 시 최근 로그인 요구", "تسجيل دخول حديث لتغيير السياسة"),
  within: (l: Locale, d: string) => t(l, `Within ${d}`, `${d} 이내`, `خلال ${d}`),
  proposedBy: (l: Locale) => t(l, "Proposed by", "제안자", "اقترحه"),
  approvedBy: (l: Locale) => t(l, "Approved by", "승인자", "اعتمده"),
  applied: (l: Locale) => t(l, "Applied", "적용 시각", "وقت التطبيق"),
  actUntil: (l: Locale, time: string) =>
    t(l, `Your sign-in allows policy changes until ${time}.`, `현재 로그인으로 ${time}까지 정책을 변경할 수 있습니다.`, `يتيح لك تسجيل دخولك الحالي تغيير السياسة حتى ${time}.`),
  pendingTitle: (l: Locale) => t(l, "Pending proposals", "승인 대기 중인 제안", "المقترحات بانتظار الاعتماد"),
  pendingCount: (l: Locale, n: number) => t(l, n === 1 ? "1 pending" : `${n} pending`, `대기 ${n}건`, `${n} بانتظار الاعتماد`),
  proposalColumn: (l: Locale) => t(l, "Proposal", "제안", "المقترح"),
  proposal: (l: Locale, n: number) => t(l, `Proposal ${n}`, `제안 ${n}`, `المقترح ${n}`),
  setting: (l: Locale) => t(l, "Setting", "항목", "الإعداد"),
  current: (l: Locale) => t(l, "Current", "현재", "الحالي"),
  proposed: (l: Locale) => t(l, "Proposed", "제안", "المقترح"),
  changed: (l: Locale) => t(l, "(changed)", "(변경)", "(تغيّر)"),
  approve: (l: Locale) => t(l, "Approve", "승인", "اعتماد"),
  withdraw: (l: Locale) => t(l, "Withdraw", "철회", "سحب"),
  needOther: (l: Locale) => t(l, "Another administrator must approve this proposal.", "다른 관리자가 이 제안을 승인해야 합니다.", "يجب أن يعتمد مسؤول آخر هذا المقترح."),
  cannotApprove: (l: Locale) => t(l, "You cannot approve this proposal.", "이 제안을 승인할 수 없습니다.", "لا يمكنك اعتماد هذا المقترح."),
  historyTitle: (l: Locale) => t(l, "Applied changes", "적용된 변경", "التغييرات المطبّقة"),
  last10: (l: Locale) => t(l, "Last 10", "최근 10건", "آخر 10"),
  proposeTitle: (l: Locale) => t(l, "Propose a change", "변경 제안", "اقتراح تغيير"),
  proposeLead: (l: Locale) =>
    t(
      l,
      "Another administrator must approve this change before it applies. Sign-in stays password only and MFA stays not required in pre-launch.",
      "다른 관리자가 승인해야 변경이 적용됩니다. 출시 전 서비스에서는 비밀번호 로그인만 사용하며 MFA는 요구하지 않습니다.",
      "يجب أن يعتمد مسؤول آخر هذا التغيير قبل تطبيقه. في خدمة ما قبل الإطلاق يبقى تسجيل الدخول بكلمة المرور فقط ولا تُطلب MFA.",
    ),
  minutes: (l: Locale) => t(l, "minutes", "분", "دقيقة"),
  maxAgeHint: (l: Locale) =>
    t(l, "5 to 720 minutes. After this time everyone must sign in again.", "5~720분. 이 시간이 지나면 모두 다시 로그인해야 합니다.", "من 5 إلى 720 دقيقة. بعد انقضائها يجب على الجميع تسجيل الدخول مجددًا."),
  stepNone: (l: Locale) => t(l, "Not required", "요구하지 않음", "غير مطلوب"),
  stepWithin: (l: Locale) => t(l, "Required within", "다음 시간 이내 요구", "مطلوب خلال"),
  stepHint: (l: Locale) =>
    t(l, "5 minutes up to the \"Sign-in valid for\" value.", "5분부터 '로그인 유지 시간' 값까지 입력할 수 있습니다.", "من 5 دقائق حتى قيمة «مدة صلاحية تسجيل الدخول»."),
  stepMinutes: (l: Locale) => t(l, "Minutes for policy changes", "정책 변경 시 요구 시간(분)", "الدقائق لتغيير السياسة"),
  errMaxAge: (l: Locale) => t(l, "Enter a whole number from 5 to 720", "5~720 사이의 정수를 입력하세요", "أدخل عددًا صحيحًا من 5 إلى 720"),
  errStep: (l: Locale, max: number) => t(l, `Enter a whole number from 5 to ${max}`, `5~${max} 사이의 정수를 입력하세요`, `أدخل عددًا صحيحًا من 5 إلى ${max}`),
  same: (l: Locale) => t(l, "These values match the current policy.", "현재 정책과 같은 값입니다.", "هذه القيم مطابقة للسياسة الحالية."),
  submit: (l: Locale) => t(l, "Submit proposal", "제안 제출", "إرسال المقترح"),
  back: (l: Locale) => t(l, "Back to sign-in policy", "로그인 정책으로", "العودة إلى سياسة تسجيل الدخول"),
  approveTitle: (l: Locale, n: number) => t(l, `Approve proposal ${n}`, `제안 ${n} 승인`, `اعتماد المقترح ${n}`),
  approveNow: (l: Locale, d: string) =>
    t(
      l,
      `The new policy applies as soon as you approve. Anyone whose sign-in is older than ${d} is signed out and must sign in again.`,
      `승인하면 새 정책이 즉시 적용됩니다. 로그인한 지 ${d}이(가) 지난 사용자는 로그아웃되며 다시 로그인해야 합니다.`,
      `تُطبَّق السياسة الجديدة فور اعتمادها. يُسجَّل خروج كل من مضى على تسجيل دخوله أكثر من ${d}، وعليه تسجيل الدخول مجددًا.`,
    ),
  approveSelf: (l: Locale) =>
    t(
      l,
      "Your own sign-in must also meet the new policy. If it does not, the approval is refused.",
      "승인하는 관리자의 로그인도 새 정책을 충족해야 합니다. 충족하지 않으면 승인이 거부됩니다.",
      "يجب أن يستوفي تسجيل دخولك أنت أيضًا السياسة الجديدة، وإلا رُفض الاعتماد.",
    ),
  approveDiscard: (l: Locale, list: string) =>
    t(l, `The other pending proposals (${list}) will be discarded.`, `승인 대기 중인 다른 제안(${list})은 폐기됩니다.`, `ستُلغى المقترحات الأخرى بانتظار الاعتماد (${list}).`),
  withdrawTitle: (l: Locale, n: number) => t(l, `Withdraw proposal ${n}`, `제안 ${n} 철회`, `سحب المقترح ${n}`),
  withdrawImpact: (l: Locale) =>
    t(
      l,
      "The proposal is deleted and can no longer be approved. The current policy does not change.",
      "제안이 삭제되어 더 이상 승인할 수 없습니다. 현재 정책은 바뀌지 않습니다.",
      "يُحذف المقترح ولا يمكن اعتماده بعد ذلك. لا تتغيّر السياسة الحالية.",
    ),
  doneProposed: (l: Locale, n: string) =>
    t(l, `${n} submitted. Another administrator must approve it.`, `${n}을(를) 제출했습니다. 다른 관리자의 승인이 필요합니다.`, `أُرسل ${n}. يلزم اعتماده من مسؤول آخر.`),
  doneApproved: (l: Locale, n: number) =>
    t(l, `Proposal ${n} approved. The new policy is in effect.`, `제안 ${n}을(를) 승인했습니다. 새 정책이 적용되었습니다.`, `اعتُمد المقترح ${n}. السياسة الجديدة سارية الآن.`),
  doneWithdrawn: (l: Locale, n: number) => t(l, `Proposal ${n} withdrawn.`, `제안 ${n}을(를) 철회했습니다.`, `سُحب المقترح ${n}.`),
  staleTitle: (l: Locale) => t(l, "Sign in again to change the policy", "정책을 변경하려면 다시 로그인하세요", "سجّل الدخول مجددًا لتغيير السياسة"),
  staleBody: (l: Locale, time: string) =>
    t(
      l,
      `You signed in at ${time}. Policy changes need a more recent sign-in. You can still view the policy.`,
      `${time}에 로그인했습니다. 정책을 변경하려면 더 최근에 로그인해야 합니다. 정책 조회는 계속할 수 있습니다.`,
      `سجّلت الدخول في ${time}. يتطلب تغيير السياسة تسجيل دخول أحدث. لا يزال بإمكانك عرض السياسة.`,
    ),
  staleAction: (l: Locale) => t(l, "Sign out and sign in again", "로그아웃 후 다시 로그인", "تسجيل الخروج ثم الدخول مجددًا"),
  staleRow: (l: Locale) => t(l, "Sign in again to approve or withdraw.", "승인하거나 철회하려면 다시 로그인하세요.", "سجّل الدخول مجددًا للاعتماد أو السحب."),
  saveFailed: (l: Locale) => t(l, "Could not save this change", "변경을 저장하지 못했습니다", "تعذّر حفظ هذا التغيير"),
  refusedTitle: (l: Locale) => t(l, "This change was refused", "이 변경이 거부되었습니다", "رُفض هذا التغيير"),
  refusedBody: (l: Locale) =>
    t(
      l,
      "A proposal must be approved by a different administrator, and only its proposer can withdraw it. Your sign-in must also be recent enough. The policy has been reloaded; check it, then try again.",
      "제안은 제안자가 아닌 다른 관리자가 승인해야 하며, 철회는 제안자만 할 수 있습니다. 로그인도 충분히 최근이어야 합니다. 정책을 다시 불러왔으니 확인한 뒤 다시 시도하세요.",
      "يجب أن يعتمد المقترحَ مسؤولٌ غير مقترحه، ولا يسحبه إلا مقترحه، ويجب أن يكون تسجيل دخولك حديثًا بما يكفي. أُعيد تحميل السياسة؛ تحقّق منها ثم حاول مجددًا.",
    ),
  stageBody: (l: Locale) =>
    t(l, "This proposal is no longer pending. The policy has been reloaded.", "이 제안은 더 이상 승인 대기 상태가 아닙니다. 정책을 다시 불러왔습니다.", "لم يعد هذا المقترح بانتظار الاعتماد. أُعيد تحميل السياسة."),
  conflictBody: (l: Locale) =>
    t(
      l,
      "The policy changed after it was loaded, so nothing was saved. It has been reloaded. Check it, then try again.",
      "불러온 뒤 정책이 바뀌어 저장하지 않았습니다. 다시 불러왔으니 확인한 뒤 다시 시도하세요.",
      "تغيّرت السياسة بعد تحميلها، فلم يُحفظ شيء. أُعيد تحميلها. تحقّق منها ثم حاول مرة أخرى.",
    ),
  busyBody: (l: Locale) =>
    t(l, "The policy service could not complete this change. Try again shortly.", "정책 서비스가 이 변경을 완료하지 못했습니다. 잠시 후 다시 시도하세요.", "تعذّر على خدمة السياسات إكمال هذا التغيير. أعد المحاولة بعد قليل."),
  loading: (l: Locale) => t(l, "Loading the sign-in policy…", "로그인 정책을 불러오는 중…", "جارٍ تحميل سياسة تسجيل الدخول…"),
  loadFailed: (l: Locale) => t(l, "The sign-in policy could not be loaded", "로그인 정책을 불러오지 못했습니다", "تعذّر تحميل سياسة تسجيل الدخول"),
  noPolicy: (l: Locale) =>
    t(
      l,
      "No sign-in policy has been applied yet. Until one is approved, administrators can sign in and members cannot.",
      "아직 적용된 로그인 정책이 없습니다. 정책이 승인되기 전에는 관리자만 로그인할 수 있습니다.",
      "لم تُطبَّق أي سياسة لتسجيل الدخول بعد. إلى أن تُعتمد سياسة، يستطيع المسؤولون فقط تسجيل الدخول.",
    ),
  noPending: (l: Locale) => t(l, "No pending proposals.", "승인 대기 중인 제안이 없습니다.", "لا توجد مقترحات بانتظار الاعتماد."),
  noHistory: (l: Locale) => t(l, "No changes applied yet.", "아직 적용된 변경이 없습니다.", "لم تُطبَّق أي تغييرات بعد."),
  deniedTitle: (l: Locale) => t(l, "You can't manage the sign-in policy here", "여기에서는 로그인 정책을 관리할 수 없습니다", "لا يمكنك إدارة سياسة تسجيل الدخول هنا"),
  denied: (l: Locale) =>
    t(
      l,
      "You do not have permission to manage the sign-in policy in this installation.",
      "이 설치 환경에서 로그인 정책을 관리할 권한이 없습니다.",
      "ليست لديك صلاحية إدارة سياسة تسجيل الدخول في هذا التثبيت.",
    ),
  methodsTitle: (l: Locale) => t(l, "Sign-in methods", "로그인 방식", "طرق تسجيل الدخول"),
  methodsLead: (l: Locale) =>
    t(
      l,
      "Pre-launch supports password sign-in only. Single sign-on and directory connections cannot be set up in this pre-launch service.",
      "출시 전 서비스는 비밀번호 로그인만 지원합니다. SSO와 디렉터리 연결은 설정할 수 없습니다.",
      "تدعم خدمة ما قبل الإطلاق تسجيل الدخول بكلمة المرور فقط، ولا يمكن فيها إعداد الدخول الموحد أو الاتصال بالدليل.",
    ),
  colMethod: (l: Locale) => t(l, "Method", "방식", "الطريقة"),
  colStatus: (l: Locale) => t(l, "Status", "상태", "الحالة"),
  colNotes: (l: Locale) => t(l, "Notes", "비고", "ملاحظات"),
  password: (l: Locale) => t(l, "Password", "비밀번호", "كلمة المرور"),
  active: (l: Locale) => t(l, "Active", "사용 중", "نشطة"),
  passwordNote: (l: Locale) =>
    t(
      l,
      "People sign in with their sign-in email and password. Accounts are managed in Users and access.",
      "로그인 이메일과 비밀번호로 로그인합니다. 계정은 '사용자·접근'에서 관리합니다.",
      "يسجّل المستخدمون الدخول بالبريد الإلكتروني لتسجيل الدخول وكلمة المرور. تُدار الحسابات في «المستخدمون والوصول».",
    ),
  ldap: (l: Locale) => t(l, "Direct LDAP over TLS", "직접 LDAP (TLS)", "LDAP مباشر عبر TLS"),
  notAvail: (l: Locale) => t(l, "Not available in pre-launch", "출시 전 서비스에서 사용할 수 없음", "غير متاح في خدمة ما قبل الإطلاق"),
  goPolicy: (l: Locale) => t(l, "Go to sign-in policy", "로그인 정책으로 이동", "الانتقال إلى سياسة تسجيل الدخول"),
  goUsers: (l: Locale) => t(l, "Go to Users and access", "'사용자·접근'으로 이동", "الانتقال إلى «المستخدمون والوصول»"),
  testTitle: (l: Locale) => t(l, "Sign-in tests are not available in pre-launch", "출시 전 서비스에서는 시험 로그인을 사용할 수 없습니다", "اختبارات تسجيل الدخول غير متاحة في خدمة ما قبل الإطلاق"),
  testBody: (l: Locale) =>
    t(
      l,
      "Sign-in tests check a connection to an external identity provider. Pre-launch has no external connections.",
      "시험 로그인은 외부 ID 공급자 연결을 확인하는 기능입니다. 출시 전 서비스에는 외부 연결이 없습니다.",
      "تتحقق اختبارات تسجيل الدخول من الاتصال بمزوّد هوية خارجي، ولا توجد اتصالات خارجية في خدمة ما قبل الإطلاق.",
    ),
  keysTitle: (l: Locale) => t(l, "Keys and certificates are not available in pre-launch", "출시 전 서비스에서는 키·인증서를 사용할 수 없습니다", "المفاتيح والشهادات غير متاحة في خدمة ما قبل الإطلاق"),
  keysBody: (l: Locale) =>
    t(
      l,
      "Keys and certificates are used only by external identity provider connections, which pre-launch does not offer.",
      "키와 인증서는 외부 ID 공급자 연결에만 쓰이며, 출시 전 서비스에서는 이 연결을 제공하지 않습니다.",
      "تُستخدم المفاتيح والشهادات فقط في الاتصال بمزوّدي الهوية الخارجيين، وهو غير متاح في خدمة ما قبل الإطلاق.",
    ),
  syncTitle: (l: Locale) => t(l, "External status sync is not available in pre-launch", "출시 전 서비스에서는 외부 상태 동기화를 사용할 수 없습니다", "مزامنة الحالة الخارجية غير متاحة في خدمة ما قبل الإطلاق"),
  syncBody: (l: Locale) =>
    t(
      l,
      "In pre-launch, account status comes only from Hybrid Billing. To end someone's access, suspend the account or mark them as departed in Users and access.",
      "출시 전 서비스에서는 계정 상태를 Hybrid Billing에서만 관리합니다. 접근을 끝내려면 '사용자·접근'에서 계정을 정지하거나 퇴사 처리하세요.",
      "في خدمة ما قبل الإطلاق تأتي حالة الحساب من Hybrid Billing فقط. لإنهاء وصول شخص، علّق حسابه أو حدّده كمغادر في «المستخدمون والوصول».",
    ),
  recoveryTitle: (l: Locale) => t(l, "Recovery paths are not available in pre-launch", "출시 전 서비스에서는 복구 경로를 사용할 수 없습니다", "مسارات الاستعادة غير متاحة في خدمة ما قبل الإطلاق"),
  recoveryBody: (l: Locale) =>
    t(
      l,
      "If someone loses their password, an administrator issues a one-time password reset link in Users and access. Reset links cannot be issued for administrators.",
      "비밀번호를 잊은 사용자에게는 관리자가 '사용자·접근'에서 일회용 비밀번호 재설정 링크를 발급합니다. 관리자에게는 재설정 링크를 발급할 수 없습니다.",
      "إذا فقد شخص كلمة مروره، يُصدر المسؤول رابطًا لمرة واحدة لإعادة تعيينها من «المستخدمون والوصول». لا يمكن إصدار روابط إعادة التعيين للمسؤولين.",
    ),
};
