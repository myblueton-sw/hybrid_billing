import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { counted } from "../../../../shared/plural";

const iso = (id: string): string => `\u2068${id}\u2069`;

export const v = {
  heldRequest: (l: Locale) => t(l, "Request reference:", "요청 참조:", "مرجع الطلب:"),
  heldCode: (l: Locale) => t(l, "Code:", "코드:", "الرمز:"),
  listTitle: (l: Locale) => t(l, "Service accounts", "서비스 계정", "حسابات الخدمة"),
  credTitle: (l: Locale) => t(l, "Credentials", "자격 증명", "بيانات الاعتماد"),
  count: (l: Locale, n: number, max: number) => t(l, `${n} of ${max} service accounts`, `서비스 계정 ${n}/${max}개`, `${n} من ${max} حساب خدمة`),
  colService: (l: Locale) => t(l, "Service account", "서비스 계정", "حساب الخدمة"),
  colOwner: (l: Locale) => t(l, "Owner", "책임자", "المسؤول"),
  colStatus: (l: Locale) => t(l, "Status", "상태", "الحالة"),
  colCred: (l: Locale) => t(l, "Credential", "자격 증명", "بيانات الاعتماد"),
  colValid: (l: Locale) => t(l, "Valid", "유효 기간", "مدة الصلاحية"),
  colActions: (l: Locale) => t(l, "Actions", "작업", "الإجراءات"),
  createdBy: (l: Locale, login: string) => t(l, `Created by ${login}`, `생성: ${login}`, `أنشأه ${login}`),
  creator: (l: Locale) => t(l, "Created by", "생성자", "أنشأه"),
  manage: (l: Locale) => t(l, "Manage", "관리", "إدارة"),
  manageCreds: (l: Locale) => t(l, "Manage credentials", "자격 증명 관리", "إدارة بيانات الاعتماد"),
  close: (l: Locale) => t(l, "Close", "닫기", "إغلاق"),
  backList: (l: Locale) => t(l, "Back to service accounts", "서비스 계정으로", "العودة إلى حسابات الخدمة"),
  backCreds: (l: Locale) => t(l, "Back to credentials", "자격 메타데이터로", "العودة إلى بيانات الاعتماد"),
  goAccounts: (l: Locale) => t(l, "Go to Service accounts", "'서비스 계정'으로 이동", "الانتقال إلى «حسابات الخدمة»"),
  retiredGroup: (l: Locale, n: number) => t(l, `Retired service accounts (${n})`, `사용 종료된 서비스 계정 (${n})`, `حسابات خدمة موقوفة نهائيًا (${n})`),
  endedGroup: (l: Locale, n: number) => t(l, `Ended credentials (${n})`, `종료된 자격 증명 (${n})`, `بيانات اعتماد منتهية (${n})`),
  loading: (l: Locale) => t(l, "Loading service accounts…", "서비스 계정을 불러오는 중…", "جارٍ تحميل حسابات الخدمة…"),
  stActive: (l: Locale) => t(l, "Active", "사용 중", "نشط"),
  stSuspended: (l: Locale) => t(l, "Suspended", "정지됨", "معلّق"),
  stRetired: (l: Locale) => t(l, "Retired", "사용 종료", "موقوف نهائيًا"),
  suspendedSub: (l: Locale) => t(l, "Its credentials can't be used", "자격 증명을 사용할 수 없음", "لا يمكن استخدام بيانات اعتماده"),
  credSummary: (l: Locale, n: number, max: number) => t(l, `Active credentials: ${n} of ${max}`, `사용 중 자격 증명 ${n}/${max}개`, `بيانات الاعتماد النشطة: ${n} من ${max}`),
  cActive: (l: Locale) => t(l, "Active", "유효", "سارية"),
  cOnHold: (l: Locale) => t(l, "On hold", "보류", "غير صالحة حاليًا"),
  cOnHoldSub: (l: Locale) => t(l, "Service account is not active", "서비스 계정이 사용 중이 아님", "حساب الخدمة غير نشط"),
  cRevokeRequired: (l: Locale) => t(l, "Revoke required", "폐기 필요", "يلزم الإبطال"),
  cRevokeRequiredSub: (l: Locale) => t(l, "Account retired; it can't use this", "계정 사용 종료됨, 사용 불가", "الحساب موقوف نهائيًا ولا يمكنه استخدامها"),
  cExpired: (l: Locale) => t(l, "Expired", "만료됨", "منتهية"),
  cRevoked: (l: Locale) => t(l, "Revoked", "폐기됨", "مُبطلة"),
  from: (l: Locale, d: string) => t(l, `From ${d}`, `${d}부터`, `من ${d}`),
  until: (l: Locale, d: string) => t(l, `Until ${d}`, `${d}까지`, `حتى ${d}`),
  ended: (l: Locale, d: string) => t(l, `Ended ${d}`, `${d} 종료`, `انتهت في ${d}`),
  createTitle: (l: Locale) => t(l, "Create service account", "서비스 계정 만들기", "إنشاء حساب خدمة"),
  createLead: (l: Locale) =>
    t(
      l,
      "A service account is a non-human identity owned by one person. It cannot approve or sign in to the console.",
      "서비스 계정은 한 사람이 책임지는 비인간 신원입니다. 승인하거나 콘솔에 로그인할 수 없습니다.",
      "حساب الخدمة هوية غير بشرية يتولى مسؤوليتها شخص واحد، ولا يمكنه الموافقة أو تسجيل الدخول إلى وحدة التحكم.",
    ),
  name: (l: Locale) => t(l, "Name", "이름", "الاسم"),
  nameHint: (l: Locale) =>
    t(
      l,
      "3–48 lowercase letters, digits or single hyphens inside. The ID is svc- followed by the name; it can't be changed or reused, even after retiring.",
      "영문 소문자·숫자·가운데 하이픈(연속 불가) 3–48자. ID는 svc- 뒤에 이름을 붙인 값이며 사용 종료 후에도 바꾸거나 다시 쓸 수 없습니다.",
      "من 3 إلى 48 حرفًا لاتينيًا صغيرًا أو رقمًا أو شرطة مفردة في الوسط. يتكوّن المعرّف من svc- متبوعًا بالاسم، ولا يمكن تغييره أو إعادة استخدامه حتى بعد الإيقاف النهائي.",
    ),
  idPreview: (l: Locale) => t(l, "Account ID:", "계정 ID:", "معرّف الحساب:"),
  owner: (l: Locale) => t(l, "Owner", "책임자", "المسؤول"),
  chooseOwner: (l: Locale) => t(l, "Choose an owner", "책임자 선택", "اختر المسؤول"),
  ownerHint: (l: Locale) =>
    t(
      l,
      "An active member of this workspace. It can't be changed later in pre-launch.",
      "이 워크스페이스의 활성 구성원. 출시 전 서비스에서는 나중에 바꿀 수 없습니다.",
      "عضو نشط في مساحة العمل هذه، ولا يمكن تغييره لاحقًا في خدمة ما قبل الإطلاق.",
    ),
  noOwners: (l: Locale) => t(l, "No active member can own a service account.", "서비스 계정을 책임질 수 있는 활성 구성원이 없습니다.", "لا يوجد عضو نشط يمكنه تولي حساب خدمة."),
  credAfter: (l: Locale) => t(l, "Issue credentials on the Credentials tab after creating.", "생성 후 '자격 메타데이터' 탭에서 자격 증명을 발급합니다.", "تصدر بيانات الاعتماد من تبويب «بيانات الاعتماد» بعد الإنشاء."),
  credentialsFact: (l: Locale) => t(l, "Credentials", "자격 증명", "بيانات الاعتماد"),
  errName: (l: Locale) => t(l, "Enter 3–48 lowercase letters, digits or single inner hyphens.", "영문 소문자·숫자·가운데 하이픈으로 3–48자를 입력하세요.", "أدخل من 3 إلى 48 حرفًا صغيرًا أو رقمًا أو شرطة مفردة في الوسط."),
  errNameTaken: (l: Locale) =>
    t(l, "This name is already used, possibly by a retired account. Choose another.", "이미 사용된 이름입니다(사용 종료된 계정 포함). 다른 이름을 고르세요.", "هذا الاسم مستخدم، وربما لحساب موقوف نهائيًا. اختر اسمًا آخر."),
  errOwner: (l: Locale) => t(l, "Choose an owner from the list.", "목록에서 책임자를 고르세요.", "اختر المسؤول من القائمة."),
  doneCreated: (l: Locale, id: string) => t(l, `Service account ${iso(id)} created.`, `서비스 계정 ${iso(id)}을(를) 만들었습니다.`, `أُنشئ حساب الخدمة ${iso(id)}.`),
  suspendTitle: (l: Locale) => t(l, "Suspend service account", "서비스 계정 정지", "تعليق حساب الخدمة"),
  suspendImpact: (l: Locale) =>
    t(
      l,
      "Its credentials stop being usable and unread secrets are discarded. You can reinstate it later.",
      "자격 증명을 사용할 수 없게 되고 아직 확인하지 않은 비밀값은 폐기됩니다. 나중에 정지를 해제할 수 있습니다.",
      "تتوقف بيانات اعتماده عن العمل وتُحذف القيم السرية غير المقروءة. يمكنك إعادة تفعيله لاحقًا.",
    ),
  reinstateTitle: (l: Locale) => t(l, "Reinstate service account", "서비스 계정 정지 해제", "إعادة تفعيل حساب الخدمة"),
  reinstateImpact: (l: Locale) =>
    t(l, "All its unrevoked, unexpired credentials become usable again.", "폐기·만료되지 않은 자격 증명이 모두 다시 사용 가능해집니다.", "تعود كل بيانات اعتماده غير المُبطلة وغير المنتهية صالحة للاستخدام."),
  reinstateWarn: (l: Locale) =>
    t(l, "Revoke any credential you suspect was exposed before reinstating.", "노출이 의심되는 자격 증명은 정지 해제 전에 폐기하세요.", "أبطِل أي بيانات اعتماد يُشتبه في كشفها قبل إعادة التفعيل."),
  retireTitle: (l: Locale) => t(l, "Retire service account", "서비스 계정 사용 종료", "إيقاف حساب الخدمة نهائيًا"),
  retireImpact: (l: Locale) =>
    t(
      l,
      "Every active credential is revoked and the name can't be used again. This can't be undone.",
      "사용 중인 자격 증명이 모두 폐기되고 이름을 다시 쓸 수 없습니다. 되돌릴 수 없습니다.",
      "تُبطل كل بيانات الاعتماد النشطة ولا يمكن استخدام الاسم مجددًا. لا يمكن التراجع عن ذلك.",
    ),
  retiredNote: (l: Locale) => t(l, "Retired. No further changes are possible.", "사용 종료됨. 더 이상 변경할 수 없습니다.", "موقوف نهائيًا. لا يمكن إجراء تغييرات أخرى."),
  doneSuspended: (l: Locale, id: string) => t(l, `${iso(id)} suspended.`, `${iso(id)}을(를) 정지했습니다.`, `عُلّق ${iso(id)}.`),
  doneReinstated: (l: Locale, id: string) => t(l, `${iso(id)} reinstated.`, `${iso(id)}의 정지를 해제했습니다.`, `أُعيد تفعيل ${iso(id)}.`),
  doneRetired: (l: Locale, id: string) => t(l, `${iso(id)} retired.`, `${iso(id)}을(를) 사용 종료했습니다.`, `أُوقف ${iso(id)} نهائيًا.`),
  issueTitle: (l: Locale) => t(l, "Issue credential", "자격 증명 발급", "إصدار بيانات اعتماد"),
  issueLead: (l: Locale) =>
    t(
      l,
      "A new secret is created and shown once. It is held for up to 10 minutes until you see it; after that only a verifier is kept.",
      "새 비밀값을 만들어 한 번만 보여 줍니다. 확인할 때까지 최대 10분간 보관하며, 이후에는 검증 값만 남습니다.",
      "تُنشأ قيمة سرية جديدة وتُعرض مرة واحدة، وتُحفظ حتى 10 دقائق إلى أن تراها، ثم لا يُحفظ إلا مُحقِّقها.",
    ),
  validFor: (l: Locale) => t(l, "Valid for", "유효 기간", "مدة الصلاحية"),
  days: (l: Locale) => t(l, "days", "일", "يوم"),
  daysHint: (l: Locale, max: number) => t(l, `1–${max} days from now.`, `지금부터 1–${max}일.`, `من 1 إلى ${max} يومًا من الآن.`),
  endsOn: (l: Locale, d: string) => t(l, `Ends ${d}`, `${d} 종료`, `تنتهي في ${d}`),
  errDays: (l: Locale, max: number) => t(l, `Enter a whole number from 1 to ${max}.`, `1–${max} 사이 정수를 입력하세요.`, `أدخل عددًا صحيحًا من 1 إلى ${max}.`),
  blockedSuspended: (l: Locale, id: string) =>
    t(l, `${iso(id)} is suspended. Reinstate it before issuing a credential.`, `${iso(id)}이(가) 정지되어 있습니다. 발급 전에 정지를 해제하세요.`, `${iso(id)} معلّق. أعد تفعيله قبل إصدار بيانات اعتماد.`),
  blockedRetired: (l: Locale) =>
    t(l, "Retired service accounts can't get new credentials.", "사용 종료된 서비스 계정에는 자격 증명을 발급할 수 없습니다.", "لا تُصدر بيانات اعتماد لحساب خدمة موقوف نهائيًا."),
  blockedLimit: (l: Locale, id: string, max: number) =>
    t(
      l,
      `${iso(id)} has ${max} active credentials, the most allowed. Revoke one first.`,
      `${iso(id)}의 사용 중 자격 증명이 한도(${max}개)에 도달했습니다. 먼저 하나를 폐기하세요.`,
      `لدى ${iso(id)} الحد الأقصى من بيانات الاعتماد النشطة (${max}). أبطِل واحدة أولًا.`,
    ),
  doneIssued: (l: Locale, cred: string) => t(l, `Credential ${iso(cred)} issued.`, `자격 증명 ${iso(cred)}을(를) 발급했습니다.`, `صدرت بيانات الاعتماد ${iso(cred)}.`),
  revealTitle: (l: Locale) => t(l, "Store the secret now", "지금 비밀값을 보관하세요", "احفظ القيمة السرية الآن"),
  secretLabel: (l: Locale) => t(l, "Secret (shown once)", "비밀값(한 번만 표시)", "القيمة السرية (تُعرض مرة واحدة)"),
  secretHint: (l: Locale) =>
    t(
      l,
      "Store it in your secret manager together with the credential ID shown above.",
      "위에 표시된 자격 증명 ID와 함께 비밀 관리 도구에 보관하세요.",
      "احفظها في أداة إدارة الأسرار مع معرّف بيانات الاعتماد المعروض أعلاه.",
    ),
  shownOnce: (l: Locale) =>
    t(l, "You won't be able to see this secret again after you leave this step.", "이 단계를 벗어나면 이 비밀값을 다시 볼 수 없습니다.", "لن تتمكن من رؤية هذه القيمة مجددًا بعد مغادرة هذه الخطوة."),
  ackLabel: (l: Locale) => t(l, "I have stored this secret", "비밀값을 보관했습니다", "حفظتُ هذه القيمة السرية"),
  ackError: (l: Locale) =>
    t(l, "Confirm that you stored the secret, or revoke this credential.", "비밀값을 보관했는지 확인하거나 이 자격 증명을 폐기하세요.", "أكّد أنك حفظت القيمة السرية، أو أبطِل بيانات الاعتماد هذه."),
  pickupOpen: (l: Locale) => t(l, "Secret not collected yet · pickup open, estimated to end", "비밀값 미수령 · 수령 가능, 예상 종료", "لم يُستلَم السر بعد · الاستلام متاح، ويُقدَّر انتهاؤه في"),
  pickupNever: (l: Locale) => t(l, "Secret never collected", "비밀값 수령 안 됨", "لم يُستلَم السر"),
  neverTitle: (l: Locale) => t(l, "Credentials whose secret was never collected", "비밀값이 수령되지 않은 자격 증명", "بيانات اعتماد لم يُستلَم سرّها"),
  neverLead: (l: Locale) =>
    t(
      l,
      "No one received the secret of these credentials, and it can't be shown any more. Revoke each one, and issue a new credential if you need one.",
      "이 자격 증명의 비밀값은 아무도 받지 않았고 더 이상 표시할 수 없습니다. 각각 폐기하고, 필요하면 새로 발급하세요.",
      "لم يستلم أحد سرّ بيانات الاعتماد هذه، ولم يعد بالإمكان عرضه. أبطِل كلًّا منها، وأصدر بيانات اعتماد جديدة عند الحاجة.",
    ),
  neverMore: (l: Locale, n: number) => counted(l, n, { en: ["{n} more", "{n} more"], ko: "{n}개 더 보기", ar: ["{n} أخرى", "واحدة أخرى", "اثنتان أخريان", "{n} أخرى", "{n} أخرى", "{n} أخرى"] }),
  revokeNeverReason: (l: Locale) => t(l, "Secret never collected", "비밀값 수령 안 됨", "لم يُستلَم السر"),
  revokeNotShownReason: (l: Locale) => t(l, "Secret not shown at issue", "발급 시 비밀값 미표시", "لم يُعرض السر عند الإصدار"),
  attemptsTitle: (l: Locale) => t(l, "Credential issues to check", "확인이 필요한 자격 증명 발급", "عمليات إصدار بيانات اعتماد تحتاج إلى تحقق"),
  attemptsDiscardNote: (l: Locale) =>
    t(
      l,
      "Discarding removes this reminder only. Any credential the server issued stays in the list.",
      "버리면 이 알림만 사라집니다. 서버가 발급한 자격 증명은 목록에 그대로 남습니다.",
      "إلغاء المحاولة يزيل هذا التذكير فقط. تبقى أي بيانات اعتماد أصدرها الخادم في القائمة.",
    ),
  attemptSending: (l: Locale, service: string) => t(l, `Issuing a credential for ${iso(service)}…`, `${iso(service)} 자격 증명 발급 중…`, `جارٍ إصدار بيانات اعتماد لـ ${iso(service)}…`),
  attemptUnknown: (l: Locale, service: string): [string, string] =>
    l === "ko"
      ? [`${iso(service)} 발급 결과 확인 안 됨.`, "서버가 이미 자격 증명을 발급했을 수 있습니다."]
      : l === "ar"
        ? [`إصدار لـ ${iso(service)}: النتيجة غير معروفة.`, "ربما أصدر الخادم بيانات الاعتماد بالفعل."]
        : [`Issue for ${iso(service)}: outcome unknown.`, "The server may already have issued a credential."],
  attemptIssued: (l: Locale, credential: string, service: string) =>
    t(
      l,
      `Credential ${iso(credential)} was issued for ${iso(service)}. Its secret has not been shown yet.`,
      `${iso(service)}에 자격 증명 ${iso(credential)}이(가) 발급되었습니다. 비밀값은 아직 표시되지 않았습니다.`,
      `صدرت بيانات الاعتماد ${iso(credential)} لـ ${iso(service)}. لم يُعرض سرّها بعد.`,
    ),
  attemptLapsed: (l: Locale, service: string): [string, string] =>
    l === "ko"
      ? [`${iso(service)} 발급 결과를 더 이상 확인할 수 없습니다.`, "이 계정에 '비밀값 수령 안 됨' 자격 증명이 있으면 폐기하세요."]
      : l === "ar"
        ? [`إصدار لـ ${iso(service)}: لم يعد بالإمكان التحقق من النتيجة.`, "إن ظهرت لبيانات اعتماد هذا الحساب عبارة «لم يُستلَم السر» فأبطِلها."]
        : [`Issue for ${iso(service)}: the outcome can no longer be checked.`, "If a credential of this account shows \"Secret never collected\", revoke it."],
  openIssue: (l: Locale) => t(l, "Open issue form", "발급 양식 열기", "فتح نموذج الإصدار"),
  discardAttempt: (l: Locale) => t(l, "Discard", "버리기", "إلغاء المحاولة"),
  startOver: (l: Locale) => t(l, "Discard and start over", "버리고 새로 시작", "إلغاء المحاولة والبدء من جديد"),
  heldTitle: (l: Locale) => t(l, "Outcome unknown", "결과 확인 안 됨", "النتيجة غير معروفة"),
  lapsedTitle: (l: Locale) => t(l, "Outcome can no longer be checked", "결과를 더 이상 확인할 수 없음", "لم يعد بالإمكان التحقق من النتيجة"),
  heldLead: (l: Locale) =>
    t(
      l,
      "The outcome of this request is unknown. Try again sends the same request. If the server already issued the credential, you get that credential and its secret, not a second one.",
      "이 요청의 결과를 확인하지 못했습니다. 다시 시도하면 같은 요청을 보냅니다. 서버가 이미 발급했다면 두 번째 자격 증명이 아니라 그 자격 증명과 비밀값을 받습니다.",
      "نتيجة هذا الطلب غير معروفة. تؤدي إعادة المحاولة إلى إرسال الطلب نفسه. إن كان الخادم قد أصدر بيانات الاعتماد فستحصل عليها وعلى سرّها، لا على بيانات اعتماد ثانية.",
    ),
  sendingLead: (l: Locale) =>
    t(l, "Sending this request. The values are locked until the server answers.", "요청을 보내는 중입니다. 서버가 응답할 때까지 값을 바꿀 수 없습니다.", "جارٍ إرسال هذا الطلب. القيم مقفلة إلى أن يردّ الخادم."),
  lockedNote: (l: Locale) =>
    t(l, "The values are locked so that Try again sends exactly the same request.", "다시 시도가 같은 요청을 보내도록 값을 잠갔습니다.", "القيم مقفلة كي تُرسل إعادة المحاولة الطلب نفسه تمامًا."),
  retryRefused: (l: Locale) =>
    t(
      l,
      "Try again was refused because service accounts changed. This answer covers only the repeated request: if the first request went through, its credential is in the list.",
      "서비스 계정이 바뀌어 다시 시도가 거부되었습니다. 이 응답은 다시 보낸 요청에만 해당합니다. 첫 요청이 처리되었다면 그 자격 증명이 목록에 있습니다.",
      "رُفضت إعادة المحاولة لأن حسابات الخدمة تغيّرت. يخص هذا الرد الطلب المُعاد فقط: إن نُفِّذ الطلب الأول فبيانات اعتماده في القائمة.",
    ),
  createRetryRefused: (l: Locale, id: string) =>
    t(
      l,
      `Try again was refused. This answer covers only the repeated request: if the first request went through, ${iso(id)} is in the list.`,
      `다시 시도가 거부되었습니다. 이 응답은 다시 보낸 요청에만 해당합니다. 첫 요청이 처리되었다면 ${iso(id)}이(가) 목록에 있습니다.`,
      `رُفضت إعادة المحاولة. يخص هذا الرد الطلب المُعاد فقط: إن نُفِّذ الطلب الأول فإن ${iso(id)} في القائمة.`,
    ),
  createHeldLead: (l: Locale) =>
    t(
      l,
      "The outcome of this request is unknown. Try again sends the same request and does not create a second account.",
      "이 요청의 결과를 확인하지 못했습니다. 다시 시도하면 같은 요청을 보내며 계정을 두 번 만들지 않습니다.",
      "نتيجة هذا الطلب غير معروفة. تؤدي إعادة المحاولة إلى إرسال الطلب نفسه ولا تُنشئ حسابًا ثانيًا.",
    ),
  createLapsed: (l: Locale, id: string) =>
    t(
      l,
      `The outcome of this request can no longer be checked. If ${iso(id)} is in the list, it may come from this request.`,
      `이 요청의 결과를 더 이상 확인할 수 없습니다. 목록에 ${iso(id)}이(가) 있다면 이 요청으로 만들어졌을 수 있습니다.`,
      `لم يعد بالإمكان التحقق من نتيجة هذا الطلب. إن ظهر ${iso(id)} في القائمة فقد يكون ناتجًا عن هذا الطلب.`,
    ),
  errNameTakenMine: (l: Locale, id: string) =>
    t(
      l,
      `${iso(id)} already exists and you created it. Your earlier request may have gone through; check the list.`,
      `${iso(id)}은(는) 이미 있으며 내가 만든 계정입니다. 이전 요청이 처리되었을 수 있으니 목록을 확인하세요.`,
      `${iso(id)} موجود بالفعل وقد أنشأته أنت. ربما نُفِّذ طلبك السابق؛ تحقّق من القائمة.`,
    ),
  accountsAttemptsNote: (l: Locale, n: number) =>
    counted(l, n, {
      en: ["{n} credential issue needs checking on the Credentials tab.", "{n} credential issues need checking on the Credentials tab."],
      ko: "자격 증명 탭에서 확인이 필요한 발급이 {n}건 있습니다.",
      ar: [
        "لا توجد عمليات إصدار تحتاج إلى تحقق في تبويب «بيانات الاعتماد».",
        "توجد عملية إصدار واحدة تحتاج إلى تحقق في تبويب «بيانات الاعتماد».",
        "توجد عمليتا إصدار تحتاجان إلى تحقق في تبويب «بيانات الاعتماد».",
        "توجد {n} عمليات إصدار تحتاج إلى تحقق في تبويب «بيانات الاعتماد».",
        "توجد {n} عملية إصدار تحتاج إلى تحقق في تبويب «بيانات الاعتماد».",
        "توجد {n} عملية إصدار تحتاج إلى تحقق في تبويب «بيانات الاعتماد».",
      ],
    }),
  goCredentials: (l: Locale) => t(l, "Go to Credentials", "'자격 증명'으로 이동", "الانتقال إلى «بيانات الاعتماد»"),
  done: (l: Locale) => t(l, "Done", "완료", "تم"),
  notStored: (l: Locale) => t(l, "I didn't store it: revoke", "보관하지 못함: 폐기", "لم أحفظها: إبطال"),
  revokeUnstoredReason: (l: Locale) => t(l, "Secret not stored at issue", "발급 시 비밀값 미보관", "لم تُحفظ القيمة السرية عند الإصدار"),
  showSecret: (l: Locale) => t(l, "Show secret", "비밀값 보기", "عرض القيمة السرية"),
  secretGone: (l: Locale) =>
    t(
      l,
      "The secret can no longer be shown. Revoke this credential and issue a new one.",
      "비밀값을 더 이상 표시할 수 없습니다. 이 자격 증명을 폐기하고 새로 발급하세요.",
      "لم يعد بالإمكان عرض القيمة السرية. أبطِل بيانات الاعتماد هذه وأصدر غيرها.",
    ),
  secretRetry: (l: Locale) =>
    t(
      l,
      "The secret couldn't be loaded. Try once more; if it was already delivered, it can't be shown again and you should revoke this credential.",
      "비밀값을 불러오지 못했습니다. 한 번 더 시도하세요. 이미 전달된 경우 다시 표시할 수 없으니 이 자격 증명을 폐기하세요.",
      "تعذّر تحميل القيمة السرية. حاول مرة أخرى؛ وإن كانت قد سُلّمت فلا يمكن عرضها مجددًا وعليك إبطال بيانات الاعتماد هذه.",
    ),
  unstoredTitle: (l: Locale) => t(l, "Secrets not confirmed as stored", "보관이 확인되지 않은 비밀값", "قيم سرية لم يُؤكَّد حفظها"),
  unstoredLead: (l: Locale) =>
    t(l, "If a secret wasn't saved, revoke its credential.", "비밀값을 보관하지 않았다면 해당 자격 증명을 폐기하세요.", "إن لم تُحفظ قيمة سرية فأبطِل بيانات اعتمادها."),
  dismiss: (l: Locale) => t(l, "Dismiss", "알림 지우기", "تجاهل"),
  revokeTitle: (l: Locale) => t(l, "Revoke credential", "자격 증명 폐기", "إبطال بيانات الاعتماد"),
  revoke: (l: Locale) => t(l, "Revoke", "폐기", "إبطال"),
  revokeImpact: (l: Locale, cred: string) =>
    t(
      l,
      `${iso(cred)} can never be used again. This can't be undone; issue a new credential if the service still needs one.`,
      `${iso(cred)}은(는) 다시 사용할 수 없습니다. 되돌릴 수 없으니 필요하면 새 자격 증명을 발급하세요.`,
      `لن تُستخدم ${iso(cred)} مجددًا. لا يمكن التراجع؛ أصدر بيانات اعتماد جديدة إن احتاجتها الخدمة.`,
    ),
  doneRevoked: (l: Locale, cred: string) => t(l, `Credential ${iso(cred)} revoked.`, `자격 증명 ${iso(cred)}을(를) 폐기했습니다.`, `أُبطلت بيانات الاعتماد ${iso(cred)}.`),
  prelaunchLead: (l: Locale) => t(l, "Pre-launch:", "출시 전 서비스:", "خدمة ما قبل الإطلاق:"),
  prelaunchNote: (l: Locale) =>
    t(l, "Credentials can't authenticate any request yet, and use is not recorded.", "아직 자격 증명으로 요청을 인증할 수 없으며 사용 기록도 남지 않습니다.", "لا يمكن بعدُ مصادقة أي طلب ببيانات الاعتماد، ولا يُسجَّل استخدامها."),
  noServices: (l: Locale) => t(l, "No service accounts yet. Create one with the form.", "아직 서비스 계정이 없습니다. 양식에서 만드세요.", "لا توجد حسابات خدمة بعد. أنشئ حسابًا من النموذج."),
  noServicesCred: (l: Locale) => t(l, "No service accounts yet. Create one first.", "아직 서비스 계정이 없습니다. 먼저 만드세요.", "لا توجد حسابات خدمة بعد. أنشئ حسابًا أولًا."),
  noCreds: (l: Locale, id: string) => t(l, `${iso(id)} has no active credentials.`, `${iso(id)}에 사용 중인 자격 증명이 없습니다.`, `لا توجد بيانات اعتماد نشطة لـ ${iso(id)}.`),
  usageTitle: (l: Locale) => t(l, "Use history is not available in pre-launch", "출시 전 서비스에서는 사용 기록을 볼 수 없습니다", "سجل الاستخدام غير متاح في خدمة ما قبل الإطلاق"),
  usageBody: (l: Locale) =>
    t(
      l,
      "Credentials can't authenticate requests yet and no use is recorded, so there is no last-used time or history to show.",
      "아직 자격 증명으로 요청을 인증할 수 없고 사용이 기록되지 않으므로 마지막 사용 시각이나 기록이 없습니다.",
      "لا يمكن بعدُ مصادقة الطلبات ببيانات الاعتماد ولا يُسجَّل الاستخدام، فلا يوجد وقت آخر استخدام أو سجل لعرضه.",
    ),
  transferTitle: (l: Locale) => t(l, "Owner transfer is not available in pre-launch", "출시 전 서비스에서는 책임자 이관을 할 수 없습니다", "نقل المسؤولية غير متاح في خدمة ما قبل الإطلاق"),
  transferBody: (l: Locale) =>
    t(
      l,
      "A service account keeps the owner chosen when it was created. To change owners, create a new service account with the new owner and retire the old one.",
      "서비스 계정의 책임자는 생성 시 정한 사람으로 유지됩니다. 바꾸려면 새 책임자로 서비스 계정을 새로 만들고 기존 계정을 사용 종료하세요.",
      "يبقى المسؤول عن حساب الخدمة هو من اختير عند إنشائه. لتغييره أنشئ حساب خدمة جديدًا بالمسؤول الجديد وأوقف القديم نهائيًا.",
    ),
  deniedTitle: (l: Locale) => t(l, "You can't manage service accounts here", "여기에서는 서비스 계정을 관리할 수 없습니다", "لا يمكنك إدارة حسابات الخدمة هنا"),
  denied: (l: Locale) =>
    t(l, "You do not have permission to manage service accounts in this installation.", "이 설치 환경에서 서비스 계정을 관리할 권한이 없습니다.", "ليست لديك صلاحية إدارة حسابات الخدمة في هذا التثبيت."),
  loadFailed: (l: Locale) => t(l, "Service accounts couldn't be loaded", "서비스 계정을 불러오지 못했습니다", "تعذّر تحميل حسابات الخدمة"),
  refusedTitle: (l: Locale) => t(l, "This change was refused", "변경이 거부되었습니다", "رُفض هذا التغيير"),
  capReached: (l: Locale, max: number) =>
    t(
      l,
      `The limit of ${max} service accounts is reached; retired ones count.`,
      `서비스 계정 한도(${max}개, 사용 종료 포함)에 도달했습니다.`,
      `بلغت الحد الأقصى (${max}) لحسابات الخدمة، وتُحتسب الموقوفة نهائيًا.`,
    ),
  stateChanged: (l: Locale) =>
    t(l, "It is no longer in a state that allows this. The list has been reloaded.", "이 작업을 할 수 없는 상태로 바뀌었습니다. 목록을 다시 불러왔습니다.", "لم تعد حالته تسمح بذلك. أُعيد تحميل القائمة."),
  alreadyRevoked: (l: Locale) => t(l, "It was already revoked. The list has been reloaded.", "이미 폐기되었습니다. 목록을 다시 불러왔습니다.", "أُبطلت مسبقًا. أُعيد تحميل القائمة."),
  gone: (l: Locale) =>
    t(l, "This service account or credential is no longer available to you, or your permission or sign-in changed. The list has been reloaded.", "이 서비스 계정 또는 자격 증명을 더 이상 사용할 수 없거나 권한·로그인 상태가 바뀌었습니다. 목록을 다시 불러왔습니다.", "لم يعد حساب الخدمة أو بيانات الاعتماد هذه متاحًا لك، أو تغيّرت صلاحيتك أو جلسة دخولك. أُعيد تحميل القائمة."),
  conflict: (l: Locale) =>
    t(
      l,
      "Service accounts changed after they were loaded, so nothing was saved. The list has been reloaded; check it and try again.",
      "불러온 뒤 서비스 계정이 바뀌어 저장하지 않았습니다. 목록을 다시 불러왔으니 확인 후 다시 시도하세요.",
      "تغيّرت حسابات الخدمة بعد تحميلها فلم يُحفظ شيء. أُعيدت القائمة؛ تحقّق ثم أعد المحاولة.",
    ),
  partialTitle: (l: Locale) => t(l, "Retired, but not every credential was revoked", "사용 종료했지만 일부 자격 증명이 폐기되지 않았습니다", "أُوقف الحساب نهائيًا لكن لم تُبطل كل بيانات الاعتماد"),
  partialBody: (l: Locale) =>
    t(
      l,
      "The account can't use them. Revoke each remaining one on the Credentials tab.",
      "이 계정은 해당 자격 증명을 사용할 수 없습니다. '자격 메타데이터' 탭에서 남은 항목을 하나씩 폐기하세요.",
      "لا يمكن للحساب استخدامها. أبطِل كل ما تبقى من تبويب «بيانات الاعتماد».",
    ),
  unavailable: (l: Locale) =>
    t(l, "The service account service is not responding. Try again shortly.", "서비스 계정 서비스가 응답하지 않습니다. 잠시 후 다시 시도해 주세요.", "خدمة حسابات الخدمة لا تستجيب. أعد المحاولة بعد قليل."),
  issueRetry: (l: Locale) =>
    t(l, "If you retry, check the list first so you don't issue twice.", "다시 시도하기 전에 목록을 확인해 중복 발급을 피하세요.", "قبل إعادة المحاولة تحقّق من القائمة لتجنّب الإصدار مرتين."),
};
