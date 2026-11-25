import type { Locale } from "../shared/types";
import { text as t } from "../shared/copy";
export const c = {
  dataReloaded: (l: Locale) => t(l, "Data reloaded.", "데이터를 다시 불러왔습니다.", "أُعيد تحميل البيانات."),
  fillLocalSample: (l: Locale) => t(l, "Fill local sample sign-in details", "로컬 샘플 로그인 정보 입력", "تعبئة بيانات الدخول التجريبية المحلية"),
  notice: (l: Locale) =>
    t(
      l,
      "Everything entered here, including accounts, passwords you set and the audit history of changes, is reset whenever the service restarts. Do not use real customer data or a password you use elsewhere.",
      "서비스를 다시 시작할 때마다 계정, 직접 설정한 비밀번호, 변경 감사 이력을 포함해 여기에 입력한 모든 내용이 초기화됩니다. 실제 고객 데이터나 다른 곳에서 쓰는 비밀번호를 사용하지 마세요.",
      "تُعاد تهيئة كل ما تُدخله هنا، بما في ذلك الحسابات وكلمات المرور التي تعيّنها وسجل تدقيق التغييرات، عند كل إعادة تشغيل للخدمة. لا تستخدم بيانات عملاء حقيقية أو كلمة مرور تستخدمها في مكان آخر.",
    ),
  noticeLead: (l: Locale) =>
    t(l, "Pre-launch service.", "출시 전 서비스입니다.", "خدمة ما قبل الإطلاق."),
  prelaunch: (l: Locale) => t(l, "Pre-launch", "출시 전", "ما قبل الإطلاق"),
  workspace: (l: Locale) => t(l, "Workspace", "업무 공간", "مساحة العمل"),
  email: (l: Locale) => t(l, "Email address", "이메일 주소", "البريد الإلكتروني"),
  password: (l: Locale) => t(l, "Password", "비밀번호", "كلمة المرور"),
  signIn: (l: Locale) => t(l, "Sign in", "로그인", "تسجيل الدخول"),
  signingIn: (l: Locale) => t(l, "Signing in…", "로그인 중…", "جارٍ تسجيل الدخول…"),
  signOut: (l: Locale) => t(l, "Sign out", "로그아웃", "تسجيل الخروج"),
  lead: (l: Locale) =>
    t(
      l,
      "Use the workspace and email address your administrator gave you.",
      "관리자에게 안내받은 업무 공간과 이메일 주소를 사용하세요.",
      "استخدم مساحة العمل والبريد الإلكتروني اللذين زوّدك بهما المسؤول.",
    ),
  cantSignIn: (l: Locale) =>
    t(l, "Can't sign in?", "로그인할 수 없나요?", "لا يمكنك تسجيل الدخول؟"),
  helpTitle: (l: Locale) =>
    t(l, "Your organization’s workspace", "소속 조직의 업무 공간", "مساحة عمل مؤسستك"),
  helpBody: (l: Locale) =>
    t(
      l,
      "Use the workspace provided by your administrator. An email address alone does not grant membership.",
      "관리자에게 안내받은 업무 공간을 입력하세요. 이메일 주소만으로 조직에 가입되지 않습니다.",
      "استخدم مساحة العمل التي وفرها المسؤول. البريد الإلكتروني وحده لا يمنح العضوية.",
    ),
  check: (l: Locale) =>
    t(l, "Check the following", "입력 내용을 확인해 주세요", "يُرجى مراجعة ما يلي"),
  errorPrefix: (l: Locale) => t(l, "Error:", "오류:", "خطأ:"),
  enterWorkspace: (l: Locale) => t(l, "Enter your workspace", "업무 공간을 입력하세요", "أدخل مساحة العمل"),
  enterEmail: (l: Locale) => t(l, "Enter your email address", "이메일 주소를 입력하세요", "أدخل بريدك الإلكتروني"),
  enterPassword: (l: Locale) => t(l, "Enter your password", "비밀번호를 입력하세요", "أدخل كلمة المرور"),
  failed: (l: Locale) =>
    t(
      l,
      "We could not sign you in. Check your workspace, email address and password, then try again.",
      "로그인할 수 없습니다. 업무 공간, 이메일 주소, 비밀번호를 확인한 뒤 다시 시도해 주세요.",
      "تعذّر تسجيل الدخول. تحقّق من مساحة العمل والبريد الإلكتروني وكلمة المرور، ثم أعد المحاولة.",
    ),
  limited: (l: Locale, when: string) =>
    t(
      l,
      `Too many attempts. Try again ${when}.`,
      `시도가 너무 많습니다. ${when} 다시 시도해 주세요.`,
      `محاولات كثيرة جدًا. أعد المحاولة ${when}.`,
    ),
  unavailable: (l: Locale) =>
    t(
      l,
      "The sign-in service is not responding. Try again shortly.",
      "로그인 서비스가 응답하지 않습니다. 잠시 후 다시 시도해 주세요.",
      "خدمة تسجيل الدخول لا تستجيب. أعد المحاولة بعد قليل.",
    ),
  signedOut: (l: Locale) => t(l, "You have signed out.", "로그아웃했습니다.", "تم تسجيل خروجك."),
  reauthTitle: (l: Locale) => t(l, "Confirm your password", "비밀번호 재확인", "تأكيد كلمة المرور"),
  reauthBody: (l: Locale) =>
    t(
      l,
      "To continue, enter the password for your current account again.",
      "계속하려면 현재 계정의 비밀번호를 다시 입력해 주세요.",
      "للمتابعة، أدخل كلمة مرور حسابك الحالي مرة أخرى.",
    ),
  reauthConfirm: (l: Locale) => t(l, "Confirm", "확인", "تأكيد"),
  reauthConfirming: (l: Locale) => t(l, "Confirming…", "확인 중…", "جارٍ التأكيد…"),
  reauthCancel: (l: Locale) => t(l, "Cancel", "취소", "إلغاء"),
  reauthEmpty: (l: Locale) => t(l, "Enter your password.", "비밀번호를 입력해 주세요.", "أدخل كلمة المرور."),
  reauthWrong: (l: Locale) => t(l, "The password is not correct.", "비밀번호가 올바르지 않습니다.", "كلمة المرور غير صحيحة."),
  reauthChanged: (l: Locale) =>
    t(
      l,
      "Your password was changed while it was being confirmed. Enter your current password and try again.",
      "확인하는 동안 비밀번호가 변경되었습니다. 현재 비밀번호를 입력해 다시 시도해 주세요.",
      "تغيّرت كلمة مرورك أثناء تأكيدها. أدخل كلمة المرور الحالية وأعد المحاولة.",
    ),
  reauthHint: (l: Locale) =>
    t(
      l,
      "After 5 wrong passwords in a row in this session, you will be signed out.",
      "이 세션에서 비밀번호를 연속 5번 틀리면 로그아웃됩니다.",
      "إذا أدخلت كلمة مرور خاطئة 5 مرات متتالية في هذه الجلسة، فسيُسجَّل خروجك.",
    ),
  reauthFailed: (l: Locale) =>
    t(l, "We could not confirm your password. Try again shortly.", "비밀번호를 확인하지 못했습니다. 잠시 후 다시 시도해 주세요.", "تعذّر تأكيد كلمة المرور. أعد المحاولة بعد قليل."),
  reauthRefused: (l: Locale) =>
    t(
      l,
      "Your password was confirmed, but this request was not completed. Try again to continue.",
      "비밀번호는 확인했지만 이 요청은 완료되지 않았습니다. 다시 시도해 주세요.",
      "تم تأكيد كلمة المرور، لكن هذا الطلب لم يكتمل. أعد المحاولة للمتابعة.",
    ),
  reauthText: (l: Locale, failure: string) => (failure === "reauth" ? c.reauthRefused(l) : c.reauthNotDone(l)),
  reauthNotDone: (l: Locale) =>
    t(
      l,
      "Your password was not confirmed, so this request was not completed. Try again to continue.",
      "비밀번호 확인이 완료되지 않아 이 요청은 처리되지 않았습니다. 다시 시도해 주세요.",
      "لم تُؤكَّد كلمة المرور، لذا لم يكتمل هذا الطلب. أعد المحاولة للمتابعة.",
    ),
  ended: (l: Locale) =>
    t(l, "Sign in again.", "다시 로그인해 주세요.", "سجّل الدخول مرة أخرى."),
  show: (l: Locale) => t(l, "Show", "표시", "إظهار"),
  hide: (l: Locale) => t(l, "Hide", "숨기기", "إخفاء"),
  showName: (l: Locale, field: string) => t(l, `Show ${field}`, `${field} 표시`, `إظهار ${field}`),
  hideName: (l: Locale, field: string) => t(l, `Hide ${field}`, `${field} 숨기기`, `إخفاء ${field}`),
  visible: (l: Locale, field: string) => t(l, `${field} is visible`, `${field}가 표시되어 있습니다`, `${field} ظاهرة`),
  hidden: (l: Locale, field: string) => t(l, `${field} is hidden`, `${field}가 숨겨져 있습니다`, `${field} مخفية`),
  checkFailed: (l: Locale) => t(l, "We could not check this link", "링크를 확인하지 못했습니다", "تعذّر التحقق من هذا الرابط"),
  tryAgain: (l: Locale) => t(l, "Try again", "다시 시도", "إعادة المحاولة"),
  notCompleted: (l: Locale) =>
    t(
      l,
      "We could not complete this. Check the details you entered, then try again.",
      "완료하지 못했습니다. 입력한 내용을 확인한 뒤 다시 시도해 주세요.",
      "تعذّر إكمال العملية. تحقّق من البيانات المُدخلة، ثم أعد المحاولة.",
    ),
  enterSignInEmail: (l: Locale) => t(l, "Enter your sign-in email", "로그인 이메일을 입력하세요", "أدخل البريد الإلكتروني لتسجيل الدخول"),
  badSignInEmail: (l: Locale) =>
    t(
      l,
      "This sign-in email is too short, too long or contains characters that cannot be used",
      "로그인 이메일이 너무 짧거나 길거나, 사용할 수 없는 문자가 들어 있습니다",
      "البريد الإلكتروني لتسجيل الدخول قصير جدًا أو طويل جدًا أو يحتوي على أحرف غير مسموح بها",
    ),
  enterContact: (l: Locale) => t(l, "Enter a contact email", "연락용 이메일을 입력하세요", "أدخل بريدًا إلكترونيًا للتواصل"),
  badContact: (l: Locale) =>
    t(l, "Enter a contact email such as name@example.com", "연락용 이메일을 name@example.com 형식으로 입력하세요", "أدخل بريدًا للتواصل مثل name@example.com"),
  loading: (l: Locale) => t(l, "Loading…", "불러오는 중…", "جارٍ التحميل…"),
  checking: (l: Locale) => t(l, "Checking your link…", "링크를 확인하고 있습니다…", "جارٍ التحقق من الرابط…"),
  setTitle: (l: Locale) => t(l, "Set your password", "비밀번호 설정", "تعيين كلمة المرور"),
  resetTitle: (l: Locale) => t(l, "Reset your password", "비밀번호 재설정", "إعادة تعيين كلمة المرور"),
  activateTitle: (l: Locale) => t(l, "Activate the first administrator", "최초 관리자 활성화", "تفعيل المسؤول الأول"),
  newPassword: (l: Locale) => t(l, "New password", "새 비밀번호", "كلمة المرور الجديدة"),
  confirmPassword: (l: Locale) => t(l, "Confirm new password", "새 비밀번호 확인", "تأكيد كلمة المرور الجديدة"),
  rule: (l: Locale) =>
    t(
      l,
      "Use at least 12 characters (Korean and Arabic letters count as more than one). Do not use your email address. Spaces, passphrases and pasting are allowed.",
      "12자 이상 입력하세요(한글과 아랍 문자는 한 글자를 여러 자로 셉니다). 이메일 주소는 비밀번호로 사용할 수 없습니다. 공백, 문장형 비밀번호, 붙여넣기를 사용할 수 있습니다.",
      "استخدم 12 حرفًا على الأقل (تُحتسب الحروف الكورية والعربية بأكثر من حرف). لا تستخدم بريدك الإلكتروني ككلمة مرور. يمكنك استخدام المسافات والعبارات واللصق.",
    ),
  signInEmail: (l: Locale) => t(l, "Sign-in email", "로그인 이메일", "البريد الإلكتروني لتسجيل الدخول"),
  contactEmail: (l: Locale) => t(l, "Contact email", "연락용 이메일", "البريد الإلكتروني للتواصل"),
  enterNew: (l: Locale) => t(l, "Enter a new password", "새 비밀번호를 입력하세요", "أدخل كلمة مرور جديدة"),
  tooShort: (l: Locale) =>
    t(l, "Password must be at least 12 characters", "비밀번호는 12자 이상이어야 합니다", "يجب ألا تقل كلمة المرور عن 12 حرفًا"),
  tooLong: (l: Locale) =>
    t(l, "Password is too long. Use a shorter one", "비밀번호가 너무 깁니다. 더 짧게 입력하세요", "كلمة المرور طويلة جدًا. استخدم كلمة أقصر"),
  sameAsLogin: (l: Locale) =>
    t(
      l,
      "Password must be different from your email address",
      "비밀번호는 이메일 주소와 달라야 합니다",
      "يجب أن تختلف كلمة المرور عن بريدك الإلكتروني",
    ),
  enterAgain: (l: Locale) => t(l, "Enter the password again", "비밀번호를 다시 입력하세요", "أعد إدخال كلمة المرور"),
  mismatch: (l: Locale) => t(l, "The passwords do not match", "비밀번호가 일치하지 않습니다", "كلمتا المرور غير متطابقتين"),
  setButton: (l: Locale) => t(l, "Set password", "비밀번호 설정", "تعيين كلمة المرور"),
  activateButton: (l: Locale) => t(l, "Activate", "활성화", "تفعيل"),
  saving: (l: Locale) => t(l, "Saving…", "저장 중…", "جارٍ الحفظ…"),
  linkTitle: (l: Locale) => t(l, "This link cannot be used", "이 링크는 사용할 수 없습니다", "لا يمكن استخدام هذا الرابط"),
  linkBody: (l: Locale) =>
    t(
      l,
      "It may have expired or already been used. Ask your workspace administrator for a new link.",
      "만료되었거나 이미 사용된 링크일 수 있습니다. 업무 공간 관리자에게 새 링크를 요청하세요.",
      "ربما انتهت صلاحيته أو سبق استخدامه. اطلب رابطًا جديدًا من مسؤول مساحة العمل.",
    ),
  goSignIn: (l: Locale) => t(l, "Go to sign in", "로그인으로 이동", "الانتقال إلى تسجيل الدخول"),
  setDone: (l: Locale) => t(l, "Password set", "비밀번호를 설정했습니다", "تم تعيين كلمة المرور"),
  resetDone: (l: Locale) => t(l, "Password reset", "비밀번호를 재설정했습니다", "تمت إعادة تعيين كلمة المرور"),
  activateDone: (l: Locale) => t(l, "Administrator activated", "관리자를 활성화했습니다", "تم تفعيل المسؤول"),
  doneBody: (l: Locale) =>
    t(l, "Sign in with your new password.", "새 비밀번호로 로그인하세요.", "سجّل الدخول بكلمة المرور الجديدة."),
  aboutLink: (l: Locale) => t(l, "About this link", "이 링크 안내", "حول هذا الرابط"),
  aboutLinkBody: (l: Locale) =>
    t(
      l,
      "This link works once. Do not share it.",
      "이 링크는 한 번만 사용할 수 있습니다. 다른 사람과 공유하지 마세요.",
      "يعمل هذا الرابط مرة واحدة. لا تشاركه مع أحد.",
    ),
  helpSignIn: (l: Locale) => t(l, "Help signing in", "로그인 도움말", "مساعدة في تسجيل الدخول"),
  forgot: (l: Locale) => t(l, "Forgot your password", "비밀번호를 잊은 경우", "إذا نسيت كلمة المرور"),
  forgotBody: (l: Locale) =>
    t(
      l,
      "Ask your workspace administrator to issue a password reset link in Users and access. The link works once. Reset links cannot be issued for administrator accounts; administrators should contact another administrator.",
      "업무 공간 관리자에게 '사용자·접근'에서 비밀번호 재설정 링크를 발급해 달라고 요청하세요. 링크는 한 번만 사용할 수 있습니다. 관리자 계정에는 재설정 링크를 발급할 수 없으므로 관리자는 다른 관리자에게 문의하세요.",
      "اطلب من مسؤول مساحة العمل إصدار رابط إعادة تعيين كلمة المرور في «المستخدمون والوصول». يعمل الرابط مرة واحدة. لا يمكن إصدار روابط إعادة التعيين لحسابات المسؤولين، لذا يتواصل المسؤول مع مسؤول آخر.",
    ),
  attempts: (l: Locale) => t(l, "Too many attempts", "시도 횟수를 초과한 경우", "عند تجاوز عدد المحاولات"),
  attemptsBody: (l: Locale) =>
    t(
      l,
      "Sign-in may be paused after repeated failed attempts. Wait, then try again.",
      "여러 번 실패하면 로그인이 잠시 제한될 수 있습니다. 잠시 기다린 뒤 다시 시도하세요.",
      "قد يُوقف تسجيل الدخول مؤقتًا بعد عدة محاولات فاشلة. انتظر قليلًا ثم أعد المحاولة.",
    ),
  restart: (l: Locale) => t(l, "After a service restart", "서비스를 다시 시작한 뒤", "بعد إعادة تشغيل الخدمة"),
  restartBody: (l: Locale) =>
    t(
      l,
      "Passwords you set are reset. Use the link or details your administrator provides.",
      "직접 설정한 비밀번호는 초기화됩니다. 관리자가 제공한 링크나 정보를 사용하세요.",
      "تُعاد تهيئة كلمات المرور التي عيّنتها. استخدم الرابط أو البيانات التي يقدّمها المسؤول.",
    ),
  backSignIn: (l: Locale) => t(l, "Back to sign in", "로그인으로 돌아가기", "العودة إلى تسجيل الدخول"),
  recoveryTitle: (l: Locale) => t(l, "Recovery in pre-launch", "출시 전 버전의 복구", "الاستعادة في إصدار ما قبل الإطلاق"),
  selfService: (l: Locale) =>
    t(
      l,
      "Self-service recovery is not available in this pre-launch version.",
      "출시 전 버전에서는 직접 복구할 수 없습니다.",
      "لا تتوفر الاستعادة الذاتية في إصدار ما قبل الإطلاق.",
    ),
  language: (l: Locale) => t(l, "Language", "언어", "اللغة"),
  skip: (l: Locale) => t(l, "Skip to content", "본문 바로가기", "انتقل إلى المحتوى"),
};
export function retryPhrase(l: Locale, seconds: number): string {
  const f = new Intl.RelativeTimeFormat(l === "ar" ? "ar-AE" : l === "ko" ? "ko-KR" : "en-US", { numeric: "auto" });
  return seconds < 60 ? f.format(Math.max(1, seconds), "second") : f.format(Math.ceil(seconds / 60), "minute");
}
