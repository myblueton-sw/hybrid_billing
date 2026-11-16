import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";

export const a = {
  peopleTitle: (l: Locale) => t(l, "Grants to people", "사용자별 권한 부여", "الصلاحيات الممنوحة للأشخاص"),
  activeCount: (l: Locale, n: number) => t(l, `${n} active`, `유효 ${n}건`, `${n} سارية`),
  colPerson: (l: Locale) => t(l, "Person", "사용자", "الشخص"),
  colAccess: (l: Locale) => t(l, "Access", "권한", "الصلاحية"),
  colStatus: (l: Locale) => t(l, "Status", "상태", "الحالة"),
  colActions: (l: Locale) => t(l, "Actions", "작업", "الإجراءات"),
  revoke: (l: Locale) => t(l, "Revoke", "회수", "إلغاء"),
  yourGrant: (l: Locale) => t(l, "Your grant", "내 권한", "صلاحيتك"),
  stActive: (l: Locale) => t(l, "Active", "유효", "سارية"),
  stPending: (l: Locale) => t(l, "Not yet valid", "아직 유효하지 않음", "لم تبدأ بعد"),
  stExpired: (l: Locale) => t(l, "Expired", "만료됨", "منتهية"),
  stRevoked: (l: Locale) => t(l, "Revoked", "회수됨", "أُلغيت"),
  until: (l: Locale, d: string) => t(l, `Until ${d}`, `${d}까지`, `حتى ${d}`),
  starts: (l: Locale, d: string) => t(l, `Starts ${d}`, `${d}부터`, `تبدأ في ${d}`),
  ended: (l: Locale, d: string) => t(l, `Ended ${d}`, `${d} 종료`, `انتهت في ${d}`),
  notApply: (l: Locale) => t(l, "Does not apply", "적용되지 않음", "لا تنطبق"),
  notApplyWhy: (l: Locale) => t(l, "Membership changed after this grant was made", "권한 부여 후 소속이 바뀌었습니다", "تغيّرت العضوية بعد منح هذه الصلاحية"),
  endedGrants: (l: Locale, n: number) => t(l, `Ended grants (${n})`, `종료된 권한 (${n})`, `الصلاحيات المنتهية (${n})`),
  noGrants: (l: Locale) =>
    t(
      l,
      "No one has been granted access yet. Use Grant access to give a person access.",
      "아직 권한을 부여받은 사용자가 없습니다. '권한 부여'로 사용자에게 권한을 주세요.",
      "لم تُمنح أي صلاحية لأي شخص بعد. استخدم «منح الصلاحية» لمنح شخص صلاحية.",
    ),
  ceilingTitle: (l: Locale) => t(l, "Console ceiling", "콘솔 권한 상한", "الحد الأقصى لصلاحيات وحدة التحكم"),
  ceilingLead: (l: Locale) =>
    t(
      l,
      "The console itself can use only these permissions. Grants to people can't go beyond them or run past their end date. They are set by the installation and can't be changed here.",
      "콘솔 자체가 쓸 수 있는 권한은 이것뿐입니다. 사용자에게 주는 권한은 이 범위와 종료일을 넘을 수 없습니다. 설치 시 정해지며 이 화면에서 바꿀 수 없습니다.",
      "لا تستطيع وحدة التحكم نفسها استخدام إلا هذه الصلاحيات، ولا يمكن أن تتجاوزها صلاحيات الأشخاص أو تمتد بعد تاريخ انتهائها. يحددها التثبيت ولا يمكن تغييرها هنا.",
    ),
  colCustomer: (l: Locale) => t(l, "Customer", "고객", "العميل"),
  allCustomers: (l: Locale) => t(l, "All customers", "모든 고객", "جميع العملاء"),
  delegTitle: (l: Locale) => t(l, "Delegations", "위임", "التفويضات"),
  colDelegation: (l: Locale) => t(l, "Delegation", "위임", "التفويض"),
  fromParent: (l: Locale, id: string) => t(l, `From delegation ${id}`, `상위 위임 ${id}`, `من التفويض ${id}`),
  canRedelegate: (l: Locale) => t(l, "Can delegate further", "재위임 가능", "يمكن إعادة تفويضها"),
  noDeleg: (l: Locale) =>
    t(
      l,
      "No delegations. A person who holds access creates delegations themselves; that isn't available in pre-launch.",
      "위임이 없습니다. 위임은 권한을 가진 사용자가 직접 만들며, 출시 전 서비스에서는 사용할 수 없습니다.",
      "لا توجد تفويضات. ينشئ التفويضَ صاحبُ الصلاحية بنفسه، وهذا غير متاح في خدمة ما قبل الإطلاق.",
    ),
  endedDeleg: (l: Locale, n: number) => t(l, `Ended delegations (${n})`, `종료된 위임 (${n})`, `التفويضات المنتهية (${n})`),
  grantTitle: (l: Locale) => t(l, "Grant access", "권한 부여", "منح الصلاحية"),
  grantLead: (l: Locale) =>
    t(
      l,
      "Gives one person access within the console ceiling. It applies as soon as you grant it. You can't grant access to yourself.",
      "콘솔 권한 상한 안에서 사용자 한 명에게 권한을 줍니다. 부여하는 즉시 적용됩니다. 자신에게는 권한을 부여할 수 없습니다.",
      "تمنح شخصًا واحدًا صلاحية ضمن الحد الأقصى لوحدة التحكم، وتسري فور منحها. لا يمكنك منح صلاحية لنفسك.",
    ),
  person: (l: Locale) => t(l, "Person", "사용자", "الشخص"),
  choosePerson: (l: Locale) => t(l, "Choose a person", "사용자 선택", "اختر شخصًا"),
  noneEligible: (l: Locale) =>
    t(
      l,
      "There's no one else to grant access to. Add a member in Users and access first.",
      "권한을 부여할 다른 사용자가 없습니다. 먼저 '사용자·접근'에서 멤버를 추가하세요.",
      "لا يوجد شخص آخر يمكن منحه صلاحية. أضف عضوًا أولًا من «المستخدمون والوصول».",
    ),
  access: (l: Locale) => t(l, "Access", "권한", "الصلاحية"),
  onlyPair: (l: Locale) =>
    t(l, "The only access the console can grant in pre-launch.", "출시 전 서비스에서 콘솔이 부여할 수 있는 유일한 권한입니다.", "الصلاحية الوحيدة التي يمكن لوحدة التحكم منحها في خدمة ما قبل الإطلاق."),
  customer: (l: Locale) => t(l, "Customer", "고객", "العميل"),
  focusAll: (l: Locale) =>
    t(l, "Collected FOCUS data can't be limited to one customer.", "수집된 FOCUS 데이터는 특정 고객으로 제한할 수 없습니다.", "لا يمكن قصر بيانات FOCUS المجمّعة على عميل واحد."),
  duration: (l: Locale) => t(l, "Duration", "기간", "المدة"),
  days: (l: Locale) => t(l, "days", "일", "يومًا"),
  daysHint: (l: Locale, max: number, d: string) =>
    t(
      l,
      `1 to ${max} days. Access ends no later than ${d}, when the console ceiling ends.`,
      `1~${max}일. 콘솔 권한 상한이 끝나는 ${d}보다 늦게 끝날 수 없습니다.`,
      `من 1 إلى ${max} يومًا. تنتهي الصلاحية في موعد أقصاه ${d}، عند انتهاء الحد الأقصى لوحدة التحكم.`,
    ),
  endsOn: (l: Locale, d: string) => t(l, `Ends ${d}`, `${d} 종료`, `تنتهي في ${d}`),
  errDays: (l: Locale, max: number) => t(l, `Enter a whole number from 1 to ${max}`, `1~${max} 사이의 정수를 입력하세요`, `أدخل عددًا صحيحًا من 1 إلى ${max}`),
  errPerson: (l: Locale) => t(l, "Choose a person", "사용자를 선택하세요", "اختر شخصًا"),
  grantImpact: (l: Locale, login: string, access: string, d: string) =>
    t(l, `${login} will be able to ${access} for all customers until ${d}.`, `${login}님은 ${d}까지 모든 고객에 대해 ${access} 권한을 갖습니다.`, `سيتمكن ${login} من ${access} لجميع العملاء حتى ${d}.`),
  doneGranted: (l: Locale, login: string) => t(l, `Access granted to ${login}.`, `${login}님에게 권한을 부여했습니다.`, `مُنحت الصلاحية لـ${login}.`),
  revokeGrantTitle: (l: Locale) => t(l, "Revoke grant", "권한 회수", "إلغاء الصلاحية"),
  revokeDelegTitle: (l: Locale) => t(l, "Revoke delegation", "위임 회수", "إلغاء التفويض"),
  revokeGrantImpact: (l: Locale, login: string) =>
    t(
      l,
      `${login} can no longer use this access. This can't be undone; grant access again if it's needed later.`,
      `${login}님은 더 이상 이 권한을 쓸 수 없습니다. 되돌릴 수 없으며, 필요하면 다시 부여해야 합니다.`,
      `لن يتمكن ${login} من استخدام هذه الصلاحية بعد الآن. لا يمكن التراجع عن ذلك؛ امنح الصلاحية مجددًا إذا لزم الأمر لاحقًا.`,
    ),
  revokeDelegImpact: (l: Locale, login: string) =>
    t(
      l,
      `${login} can no longer use access through this delegation, and any delegations made from it also stop applying. This can't be undone.`,
      `${login}님은 이 위임을 통한 권한을 더 이상 쓸 수 없으며, 이 위임에서 이어진 하위 위임도 모두 적용되지 않습니다. 되돌릴 수 없습니다.`,
      `لن يتمكن ${login} من استخدام الصلاحية عبر هذا التفويض، ويتوقف كذلك كل تفويض صادر عنه. لا يمكن التراجع عن ذلك.`,
    ),
  doneRevokedGrant: (l: Locale, login: string) => t(l, `Grant revoked for ${login}.`, `${login}님의 권한을 회수했습니다.`, `أُلغيت صلاحية ${login}.`),
  doneRevokedDeleg: (l: Locale) => t(l, "Delegation revoked.", "위임을 회수했습니다.", "أُلغي التفويض."),
  backToGrants: (l: Locale) => t(l, "Back to grants", "부여·위임으로", "العودة إلى المنح"),
  refusedTitle: (l: Locale) => t(l, "This change was refused", "이 변경이 거부되었습니다", "رُفض هذا التغيير"),
  refusedGrant: (l: Locale) =>
    t(
      l,
      "Access couldn't be granted: it may be to yourself, or this person's access changed while granting. The list has been reloaded; check it and try again.",
      "권한을 부여하지 못했습니다. 자신에게 부여하려 했거나 부여하는 동안 사용자의 권한 상태가 바뀌었을 수 있습니다. 목록을 다시 불러왔으니 확인한 뒤 다시 시도하세요.",
      "تعذّر منح الصلاحية: ربما كانت لك أنت، أو تغيّرت صلاحيات هذا الشخص أثناء المنح. أُعيد تحميل القائمة؛ تحقّق منها ثم حاول مجددًا.",
    ),
  conflictBody: (l: Locale) =>
    t(
      l,
      "Access changed after it was loaded, so nothing was saved. The list has been reloaded. Check it, then try again.",
      "불러온 뒤 권한 정보가 바뀌어 저장하지 않았습니다. 목록을 다시 불러왔으니 확인한 뒤 다시 시도하세요.",
      "تغيّرت الصلاحيات بعد تحميلها، فلم يُحفظ شيء. أُعيد تحميل القائمة. تحقّق منها ثم حاول مجددًا.",
    ),
  unavailableBody: (l: Locale) =>
    t(l, "The access service is not responding. Try again shortly.", "권한 서비스가 응답하지 않습니다. 잠시 후 다시 시도해 주세요.", "خدمة الصلاحيات لا تستجيب. أعد المحاولة بعد قليل."),
  refusedRevoke: (l: Locale) =>
    t(
      l,
      "This grant can't be revoked here. Console ceiling grants and your own grants can't be revoked on this screen. The list has been reloaded.",
      "이 권한은 여기에서 회수할 수 없습니다. 콘솔 권한 상한과 내 권한은 이 화면에서 회수할 수 없습니다. 목록을 다시 불러왔습니다.",
      "لا يمكن إلغاء هذه الصلاحية هنا؛ فلا تُلغى من هذه الشاشة صلاحيات الحد الأقصى لوحدة التحكم ولا صلاحياتك أنت. أُعيد تحميل القائمة.",
    ),
  stageBody: (l: Locale) =>
    t(l, "It was already revoked or has ended. The list has been reloaded.", "이미 회수되었거나 종료되었습니다. 목록을 다시 불러왔습니다.", "أُلغيت مسبقًا أو انتهت. أُعيد تحميل القائمة."),
  invalidBody: (l: Locale) =>
    t(
      l,
      "Something no longer matches: the person may have left the workspace or the console ceiling may have changed. The list has been reloaded; check it, then try again.",
      "사용자가 업무 공간에서 빠졌거나 콘솔 권한 상한이 바뀌었을 수 있습니다. 목록을 다시 불러왔으니 확인한 뒤 다시 시도하세요.",
      "ربما غادر الشخص مساحة العمل أو تغيّر الحد الأقصى لوحدة التحكم. أُعيد تحميل القائمة؛ تحقّق منها ثم حاول مجددًا.",
    ),
  goneBody: (l: Locale) => t(l, "It no longer exists. The list has been reloaded.", "더 이상 존재하지 않습니다. 목록을 다시 불러왔습니다.", "لم تعد موجودة. أُعيد تحميل القائمة."),
  data: (l: Locale) => t(l, "Data", "데이터", "البيانات"),
  customerOpt: (l: Locale) => t(l, "Customer ID (optional)", "고객 ID(선택)", "معرّف العميل (اختياري)"),
  customerOptHint: (l: Locale) =>
    t(l, "Leave empty to check data that isn't tied to one customer.", "특정 고객에 속하지 않는 데이터를 확인하려면 비워 두세요.", "اتركه فارغًا للتحقق من بيانات غير مرتبطة بعميل واحد."),
  showEffect: (l: Locale) => t(l, "Show effect", "효력 확인", "عرض الأثر"),
  previewLead: (l: Locale) => t(l, "This is a preview.", "미리 보기입니다.", "هذه معاينة فقط."),
  previewNote: (l: Locale) =>
    t(
      l,
      "It checks only this person's grants and delegations. A real request also needs the console's own permission and a current sign-in by this person.",
      "이 사용자의 권한 부여와 위임만 확인합니다. 실제 요청에는 콘솔 자체의 권한과 이 사용자의 유효한 로그인도 필요합니다.",
      "تتحقق من صلاحيات هذا الشخص وتفويضاته فقط. يتطلب الطلب الفعلي أيضًا صلاحية وحدة التحكم نفسها وتسجيل دخول ساريًا لهذا الشخص.",
    ),
  effectFor: (l: Locale, login: string) => t(l, `Effect for ${login}`, `${login}님의 효력`, `الأثر لـ${login}`),
  noCustomer: (l: Locale) => t(l, "No customer", "고객 지정 없음", "دون عميل"),
  checkedAt: (l: Locale, time: string) => t(l, `Checked at ${time}`, `${time}에 확인`, `تم التحقق في ${time}`),
  allowedCount: (l: Locale, n: number, total: number) => t(l, `${n} of ${total} checks allowed`, `${total}건 중 ${n}건 허용`, `${n} من أصل ${total} مسموح`),
  colAction: (l: Locale) => t(l, "Action", "작업", "الإجراء"),
  colField: (l: Locale) => t(l, "Field", "필드", "الحقل"),
  colResult: (l: Locale) => t(l, "Result", "결과", "النتيجة"),
  colWhy: (l: Locale) => t(l, "Why", "이유", "السبب"),
  notField: (l: Locale) => t(l, "Not field-specific", "필드와 무관", "لا يخص حقلًا"),
  allowed: (l: Locale) => t(l, "Allowed", "허용", "مسموح"),
  notAllowed: (l: Locale) => t(l, "Not allowed", "허용 안 됨", "غير مسموح"),
  startHint: (l: Locale) => t(l, "Choose a person, then select Show effect.", "사용자를 고른 뒤 '효력 확인'을 누르세요.", "اختر شخصًا ثم اختر «عرض الأثر»."),
  cleared: (l: Locale) =>
    t(l, "Inputs changed. Select Show effect to check again.", "조건이 바뀌었습니다. '효력 확인'을 눌러 다시 확인하세요.", "تغيّرت الاختيارات. اختر «عرض الأثر» للتحقق مجددًا."),
  previewFailed: (l: Locale) => t(l, "The effect could not be checked", "효력을 확인하지 못했습니다", "تعذّر التحقق من الأثر"),
  whyAllowed: (l: Locale) => t(l, "A grant to this person allows it.", "이 사용자에게 부여된 권한으로 허용됩니다.", "تسمح به صلاحية ممنوحة لهذا الشخص."),
  whyChain: (l: Locale, ids: string) =>
    t(l, `Allowed through a delegation. Chain: ${ids}`, `위임을 통해 허용됩니다. 위임 경로: ${ids}`, `مسموح عبر تفويض. سلسلة التفويض: ${ids}`),
  whyNoGrant: (l: Locale) =>
    t(
      l,
      "No grant or delegation gives this person this access for this data.",
      "이 데이터에 대해 이 권한을 주는 권한 부여나 위임이 없습니다.",
      "لا توجد صلاحية ممنوحة أو تفويض يمنح هذا الشخص هذه الصلاحية على هذه البيانات.",
    ),
  whyClient: (l: Locale) =>
    t(
      l,
      "The console itself isn't permitted to do this, so no one can do it through the console.",
      "콘솔 자체에 이 권한이 없어 누구도 콘솔에서 할 수 없습니다.",
      "وحدة التحكم نفسها غير مسموح لها بذلك، لذا لا يمكن لأحد القيام به من خلالها.",
    ),
  whyPrincipal: (l: Locale) =>
    t(
      l,
      "This person's account or membership is no longer current, so no access applies.",
      "이 사용자의 계정이나 소속이 더 이상 유효하지 않아 어떤 권한도 적용되지 않습니다.",
      "لم يعد حساب هذا الشخص أو عضويته ساريًا، لذا لا تنطبق أي صلاحية.",
    ),
  whyInvalid: (l: Locale) =>
    t(
      l,
      "This check can't be previewed here. It depends on approval or installation rules.",
      "이 항목은 여기에서 미리 볼 수 없습니다. 승인 또는 설치 규칙에 따라 결정됩니다.",
      "لا يمكن معاينة هذا التحقق هنا؛ فهو يعتمد على قواعد الاعتماد أو التثبيت.",
    ),
  whyStamp: (l: Locale) =>
    t(l, "Access changed while this was being checked. Select Show effect again.", "확인하는 동안 권한이 바뀌었습니다. '효력 확인'을 다시 누르세요.", "تغيّرت الصلاحيات أثناء التحقق. اختر «عرض الأثر» مجددًا."),
  whyUnavailable: (l: Locale) =>
    t(l, "The access service couldn't decide right now. Try again shortly.", "지금은 권한 서비스가 판단하지 못했습니다. 잠시 후 다시 시도하세요.", "تعذّر على خدمة الصلاحيات البتّ الآن. أعد المحاولة بعد قليل."),
  whyOther: (l: Locale) => t(l, "Not allowed (code:", "허용 안 됨(코드:", "غير مسموح (الرمز:"),
  rolesTitle: (l: Locale) => t(l, "Role definitions are not available in pre-launch", "출시 전 서비스에서는 역할 정의를 사용할 수 없습니다", "تعريفات الأدوار غير متاحة في خدمة ما قبل الإطلاق"),
  rolesBody: (l: Locale) =>
    t(
      l,
      "In pre-launch, access is given to one person at a time on the Grants tab. A person's workspace role (administrator or member) is set in Users and access.",
      "출시 전 서비스에서는 '부여·위임' 탭에서 사용자별로 권한을 줍니다. 업무 공간 역할(관리자·멤버)은 '사용자·접근'에서 정합니다.",
      "في خدمة ما قبل الإطلاق تُمنح الصلاحية لكل شخص على حدة من تبويب «المنح». ويُحدَّد دور الشخص في مساحة العمل (مسؤول أو عضو) من «المستخدمون والوصول».",
    ),
  goGrants: (l: Locale) => t(l, "Go to Grants", "'부여·위임'으로 이동", "الانتقال إلى «المنح»"),
  supportTitle: (l: Locale) => t(l, "Support sessions are not available in pre-launch", "출시 전 서비스에서는 지원 세션을 사용할 수 없습니다", "جلسات الدعم غير متاحة في خدمة ما قبل الإطلاق"),
  supportBody: (l: Locale) =>
    t(
      l,
      "Support sessions are not part of this pre-launch service, so none can be requested or approved here.",
      "출시 전 서비스에는 지원 세션이 없어 여기에서 요청하거나 승인할 수 없습니다.",
      "لا تتضمن خدمة ما قبل الإطلاق جلسات دعم، لذا لا يمكن طلبها أو اعتمادها هنا.",
    ),
  loading: (l: Locale) => t(l, "Loading access…", "권한 정보를 불러오는 중…", "جارٍ تحميل الصلاحيات…"),
  loadFailed: (l: Locale) => t(l, "Access could not be loaded", "권한 정보를 불러오지 못했습니다", "تعذّر تحميل الصلاحيات"),
  deniedTitle: (l: Locale) => t(l, "You can't manage access here", "여기에서는 권한을 관리할 수 없습니다", "لا يمكنك إدارة الصلاحيات هنا"),
  denied: (l: Locale) =>
    t(l, "You do not have permission to manage access in this installation.", "이 설치 환경에서 권한을 관리할 권한이 없습니다.", "ليست لديك صلاحية إدارة الصلاحيات في هذا التثبيت."),
};

const actions: Record<string, [string, string, string]> = {
  view: ["View", "조회", "عرض"],
  view_cost: ["View cost", "원가 조회", "عرض التكلفة"],
  change_customer_policy: ["Change customer policy", "고객 정책 변경", "تغيير سياسة العميل"],
  change_selling_policy: ["Change selling policy", "판매 정책 변경", "تغيير سياسة البيع"],
  request_approval: ["Request approval", "승인 요청", "طلب الاعتماد"],
  issue: ["Issue", "발행", "إصدار"],
  download: ["Download", "다운로드", "تنزيل"],
  deliver: ["Deliver", "전달", "تسليم"],
};
const kinds: Record<string, [string, string, string]> = { focus_data: ["Collected FOCUS data", "수집된 FOCUS 데이터", "بيانات FOCUS المجمّعة"] };
const fields: Record<string, [string, string, string]> = { rows: ["Rows", "행", "الصفوف"] };
const phrases: Record<string, [string, string, string]> = {
  "view|focus_data|rows": ["view rows of collected FOCUS data", "수집된 FOCUS 데이터 행 조회", "عرض صفوف بيانات FOCUS المجمّعة"],
};

const pick = (table: Record<string, [string, string, string]>, l: Locale, key: string): string | null => {
  const found = table[key];
  return found ? t(l, found[0], found[1], found[2]) : null;
};

export const actionLabel = (l: Locale, key: string) => pick(actions, l, key);
export const kindLabel = (l: Locale, key: string) => pick(kinds, l, key);
export const fieldLabel = (l: Locale, key: string) => pick(fields, l, key);
export const accessPhrase = (l: Locale, action: string, kind: string, fieldList: string[]) => pick(phrases, l, [action, kind, fieldList.join(",")].join("|"));
