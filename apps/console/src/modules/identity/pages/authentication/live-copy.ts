import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

export const p = {
  methods: (l: Locale) => t(l, "Sign-in methods", "로그인 방식", "طرق تسجيل الدخول"),
  passwordOnly: (l: Locale) => t(l, "Password only", "비밀번호만", "كلمة المرور فقط"),
  mfa: (l: Locale) => t(l, "Multi-factor authentication (MFA)", "다단계 인증(MFA)", "المصادقة متعددة العوامل (MFA)"),
  notRequired: (l: Locale) => t(l, "Not required", "요구하지 않음", "غير مطلوبة"),
  maxAge: (l: Locale) => t(l, "Sign-in valid for", "로그인 유지 시간", "مدة صلاحية تسجيل الدخول"),
  stepUp: (l: Locale) => t(l, "Recent password confirmation for policy changes", "정책 변경 시 최근 비밀번호 확인 요구", "تأكيد حديث لكلمة المرور لتغيير السياسة"),
  effective: (l: Locale) => t(l, "Password confirmation window", "비밀번호 확인 유효 시간", "مدة صلاحية تأكيد كلمة المرور"),
  stepDefault: (l: Locale, d: string) => t(l, `Default (${d})`, `기본값(${d})`, `الافتراضي (${d})`),
  within: (l: Locale, d: string) => t(l, `Within ${d}`, `${d} 이내`, `خلال ${d}`),
  actUntil: (l: Locale, time: string) =>
    t(
      l,
      `You can change the policy without confirming your password again until ${time}.`,
      `${time}까지는 비밀번호 재확인 없이 정책을 변경할 수 있습니다.`,
      `يمكنك تغيير السياسة دون تأكيد كلمة المرور مجددًا حتى ${time}.`,
    ),
  staleLine: (l: Locale) =>
    t(
      l,
      "You'll be asked to confirm your password before changing the policy.",
      "정책을 변경하기 전에 비밀번호 재확인을 요청합니다.",
      "سيُطلب منك تأكيد كلمة المرور قبل تغيير السياسة.",
    ),
  setting: (l: Locale) => t(l, "Setting", "항목", "الإعداد"),
  current: (l: Locale) => t(l, "Current", "현재", "الحالي"),
  proposed: (l: Locale) => t(l, "Proposed", "제안", "المقترح"),
  changed: (l: Locale) => t(l, "(changed)", "(변경)", "(تغيّر)"),
  minutes: (l: Locale) => t(l, "minutes", "분", "دقيقة"),
  maxAgeHint: (l: Locale) =>
    t(
      l,
      "5 to 720 minutes, counted from each person's last sign-in or password confirmation. After this time they must sign in again.",
      "5~720분. 각 사용자의 마지막 로그인 또는 비밀번호 재확인부터 계산하며, 이 시간이 지나면 다시 로그인해야 합니다.",
      "من 5 إلى 720 دقيقة، تُحسب من آخر تسجيل دخول أو تأكيد لكلمة المرور لكل شخص. بعد انقضائها يجب عليه تسجيل الدخول مجددًا.",
    ),
  stepNone: (l: Locale) => t(l, "Use the 15-minute default", "기본값 15분 사용", "استخدام القيمة الافتراضية 15 دقيقة"),
  stepWithin: (l: Locale) => t(l, "Set a shorter time", "더 짧은 시간 지정", "تحديد مدة أقصر"),
  stepHint: (l: Locale) =>
    t(
      l,
      "Blank or 0 uses the 15-minute default. A value under 15 minutes is stricter; a longer value still allows 15 minutes.",
      "비워 두거나 0을 입력하면 기본값 15분을 사용합니다. 15분보다 짧으면 더 엄격해지고, 더 길게 입력해도 15분이 적용됩니다.",
      "يؤدي ترك الحقل فارغًا أو إدخال 0 إلى استخدام القيمة الافتراضية 15 دقيقة. القيمة الأقل من 15 دقيقة أكثر صرامة، أما القيمة الأطول فتُطبَّق عليها مدة 15 دقيقة.",
    ),
  stepMinutes: (l: Locale) => t(l, "Minutes for policy changes", "정책 변경 시 요구 시간(분)", "الدقائق لتغيير السياسة"),
  errMaxAge: (l: Locale) => t(l, "Enter a whole number from 5 to 720", "5~720 사이의 정수를 입력하세요", "أدخل عددًا صحيحًا من 5 إلى 720"),
  errStep: (l: Locale, max: number) => t(l, `Enter a whole number from 5 to ${max}`, `5~${max} 사이의 정수를 입력하세요`, `أدخل عددًا صحيحًا من 5 إلى ${max}`),
  same: (l: Locale) => t(l, "These values match the current policy.", "현재 정책과 같은 값입니다.", "هذه القيم مطابقة للسياسة الحالية."),
  saveFailed: (l: Locale) => t(l, "Could not save this change", "변경을 저장하지 못했습니다", "تعذّر حفظ هذا التغيير"),
  refusedTitle: (l: Locale) => t(l, "This change was refused", "이 변경이 거부되었습니다", "رُفض هذا التغيير"),
  refusedBody: (l: Locale) =>
    t(
      l,
      "A rule refused this step. The trust changes have been reloaded; check them, then try again.",
      "규칙에 따라 이 단계가 거부되었습니다. 신뢰 변경을 다시 불러왔으니 확인한 뒤 다시 시도하세요.",
      "رفضت قاعدةٌ هذه الخطوة. أُعيد تحميل تغييرات الثقة؛ تحقّق منها ثم حاول مجددًا.",
    ),
  stageBody: (l: Locale) =>
    t(l, "This change is no longer in a state that allows this step. It has been reloaded.", "이 변경은 더 이상 이 단계를 진행할 수 있는 상태가 아닙니다. 다시 불러왔습니다.", "لم يعد هذا التغيير في حالة تسمح بهذه الخطوة. أُعيد تحميله."),
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
