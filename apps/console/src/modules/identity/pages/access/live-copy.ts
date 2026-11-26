import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { counted, type Plural } from "../../../../shared/plural";

const counts = {
  grants: {
    en: ["1 active grant", "{n} active grants"],
    ko: "유효 {n}건",
    ar: ["{n} صلاحية سارية", "صلاحية سارية واحدة", "صلاحيتان ساريتان", "{n} صلاحيات سارية", "{n} صلاحية سارية", "{n} صلاحية سارية"],
  },
  delegations: {
    en: ["1 active delegation", "{n} active delegations"],
    ko: "유효 {n}건",
    ar: ["{n} تفويض سارٍ", "تفويض سارٍ واحد", "تفويضان ساريان", "{n} تفويضات سارية", "{n} تفويضًا ساريًا", "{n} تفويض سارٍ"],
  },
  days: { en: ["1 day", "{n} days"], ko: "{n}일", ar: ["{n} يوم", "يوم واحد", "يومين", "{n} أيام", "{n} يومًا", "{n} يوم"] },
} satisfies Record<string, Plural>;

export const a = {
  peopleTitle: (l: Locale) => t(l, "Grants to people", "사용자별 권한 부여", "الصلاحيات الممنوحة للأشخاص"),
  activeGrants: (l: Locale, n: number) => counted(l, n, counts.grants),
  activeDelegations: (l: Locale, n: number) => counted(l, n, counts.delegations),
  colPerson: (l: Locale) => t(l, "Person", "사용자", "الشخص"),
  colAccess: (l: Locale) => t(l, "Access", "권한", "الصلاحية"),
  colStatus: (l: Locale) => t(l, "Status", "상태", "الحالة"),
  colActions: (l: Locale) => t(l, "Actions", "작업", "الإجراءات"),
  revoke: (l: Locale) => t(l, "Revoke", "회수", "إبطال"),
  yourGrant: (l: Locale) => t(l, "Your grant", "내 권한", "صلاحيتك"),
  stActive: (l: Locale) => t(l, "Active", "유효", "سارية"),
  stPending: (l: Locale) => t(l, "Not yet valid", "아직 유효하지 않음", "لم تبدأ بعد"),
  stExpired: (l: Locale) => t(l, "Expired", "만료됨", "منتهية"),
  stRevoked: (l: Locale) => t(l, "Revoked", "회수됨", "أُبطلت"),
  stUnconfirmed: (l: Locale) => t(l, "Revocation not confirmed", "회수 미확정", "الإبطال غير مؤكَّد"),
  stUnconfirmedWhy: (l: Locale) =>
    t(l, "Blocked in this server process; see the Revocations section", "이 서버 프로세스에서 차단됨. '회수' 섹션을 확인하세요", "محجوب في عملية الخادم هذه؛ راجع قسم «الإبطال»"),
  until: (l: Locale, d: string) => t(l, `Until ${d}`, `${d}까지`, `حتى ${d}`),
  starts: (l: Locale, d: string) => t(l, `Starts ${d}`, `${d}부터`, `تبدأ في ${d}`),
  ended: (l: Locale, d: string) => t(l, `Ended ${d}`, `${d} 종료`, `انتهت في ${d}`),
  notApply: (l: Locale) => t(l, "Does not apply", "적용되지 않음", "لا تنطبق"),
  notApplyWhy: (l: Locale) => t(l, "Membership changed after this grant was made", "권한 부여 후 소속이 바뀌었습니다", "تغيّرت العضوية بعد منح هذه الصلاحية"),
  endedGrants: (l: Locale, n: number) => t(l, `Ended grants (${n})`, `종료된 권한 (${n})`, `الصلاحيات المنتهية (${n})`),
  noGrants: (l: Locale) =>
    t(
      l,
      "No one has been granted access yet. Use Request access to propose access for a person.",
      "아직 권한을 부여받은 사용자가 없습니다. '권한 추가 요청'으로 사용자의 권한을 요청하세요.",
      "لم تُمنح أي صلاحية لأي شخص بعد. استخدم «طلب صلاحية» لاقتراح منح صلاحية لشخص.",
    ),
  ceilingTitle: (l: Locale) => t(l, "Console ceiling", "콘솔 권한 상한", "الحد الأقصى لصلاحيات وحدة التحكم"),
  catalogShort: (l: Locale) => t(l, "Some access the console may grant may not be listed here. An installation operator can raise the query scan limit (service_query_scan).",
    "콘솔이 부여할 수 있는 권한 중 일부가 여기에 표시되지 않았을 수 있습니다. 설치 운영자가 조회 스캔 한도(service_query_scan)를 올릴 수 있습니다.",
    "قد لا تظهر هنا بعض الصلاحيات التي يجوز لوحدة التحكم منحها. يمكن لمشغّل التثبيت رفع حدّ فحص الاستعلام (service_query_scan)."),
  catalogNone: (l: Locale) => t(l, "No access to grant could be listed: reading the console's catalog stopped at the query scan limit. An installation operator can raise it (service_query_scan).",
    "부여할 권한을 표시하지 못했습니다. 콘솔 카탈로그를 읽다가 조회 스캔 한도에 도달했습니다. 설치 운영자가 이 한도(service_query_scan)를 올릴 수 있습니다.",
    "تعذّر عرض أي صلاحية للمنح: توقفت قراءة كتالوج وحدة التحكم عند حدّ فحص الاستعلام. يمكن لمشغّل التثبيت رفعه (service_query_scan)."),
  kindsShort: (l: Locale) => t(l, "Some kinds of data the console may use may not be listed here. An installation operator can raise the query scan limit (service_query_scan).",
    "콘솔이 사용할 수 있는 데이터 종류 중 일부가 여기에 표시되지 않았을 수 있습니다. 설치 운영자가 조회 스캔 한도(service_query_scan)를 올릴 수 있습니다.",
    "قد لا تظهر هنا بعض أنواع البيانات التي يجوز لوحدة التحكم استخدامها. يمكن لمشغّل التثبيت رفع حدّ فحص الاستعلام (service_query_scan)."),
  kindsNone: (l: Locale) => t(l, "No kind of data could be listed: reading the console's catalog stopped at the query scan limit. An installation operator can raise it (service_query_scan).",
    "데이터 종류를 표시하지 못했습니다. 콘솔 카탈로그를 읽다가 조회 스캔 한도에 도달했습니다. 설치 운영자가 이 한도(service_query_scan)를 올릴 수 있습니다.",
    "تعذّر عرض أي نوع بيانات: توقفت قراءة كتالوج وحدة التحكم عند حدّ فحص الاستعلام. يمكن لمشغّل التثبيت رفعه (service_query_scan)."),
  accessHint: (l: Locale) => t(l, "Choices are limited to what the console itself may do: the Console rows in the list above.", "콘솔 자체가 할 수 있는 범위로 제한됩니다. 위 목록의 '콘솔' 행을 참고하세요.", "تقتصر الخيارات على ما يُسمح لوحدة التحكم نفسها بفعله: صفوف «وحدة التحكم» في القائمة أعلاه."),
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
  accessGone: (l: Locale) => t(l, "The access you chose can no longer be granted. Choose access again.", "선택한 권한을 더 이상 부여할 수 없습니다. 권한을 다시 선택하세요.", "لم يعد بالإمكان منح الصلاحية التي اخترتها. اختر الصلاحية مجددًا."),
  chooseDurationHint: (l: Locale) => t(l, "The maximum depends on the access you choose.", "최대 기간은 선택한 권한에 따라 다릅니다.", "يعتمد الحد الأقصى للمدة على الصلاحية المختارة."),
  chooseAccess: (l: Locale) => t(l, "Choose access", "권한 선택", "اختر الصلاحية"),
  customer: (l: Locale) => t(l, "Customer", "고객", "العميل"),
  focusAll: (l: Locale) =>
    t(l, "Collected FOCUS data can't be limited to one customer.", "수집된 FOCUS 데이터는 특정 고객으로 제한할 수 없습니다.", "لا يمكن قصر بيانات FOCUS المجمّعة على عميل واحد."),
  duration: (l: Locale) => t(l, "Duration", "기간", "المدة"),
  days: (l: Locale) => t(l, "days", "일", "عدد الأيام"),
  daysHint: (l: Locale, max: number, d: string) =>
    t(
      l,
      `1 to ${counted(l, max, counts.days)}. Access ends no later than ${d}, when the console ceiling ends.`,
      `1~${counted(l, max, counts.days)}. 콘솔 권한 상한이 끝나는 ${d}보다 늦게 끝날 수 없습니다.`,
      `من 1 إلى ${counted(l, max, counts.days)}. تنتهي الصلاحية في موعد أقصاه ${d}، عند انتهاء الحد الأقصى لوحدة التحكم.`,
    ),
  errDays: (l: Locale, max: number) => t(l, `Enter a whole number from 1 to ${max}`, `1~${max} 사이의 정수를 입력하세요`, `أدخل عددًا صحيحًا من 1 إلى ${max}`),
  errPerson: (l: Locale) => t(l, "Choose a person", "사용자를 선택하세요", "اختر شخصًا"),
  revokeDelegTitle: (l: Locale) => t(l, "Revoke delegation", "위임 회수", "إبطال التفويض"),
  revokeDelegImpact: (l: Locale, login: string) =>
    t(
      l,
      `${login} can no longer use access through this delegation, and any delegations made from it also stop applying. This can't be undone.`,
      `${login}님은 이 위임을 통한 권한을 더 이상 쓸 수 없으며, 이 위임에서 이어진 하위 위임도 모두 적용되지 않습니다. 되돌릴 수 없습니다.`,
      `لن يتمكن ${login} من استخدام الصلاحية عبر هذا التفويض، ويتوقف كذلك كل تفويض صادر عنه. لا يمكن التراجع عن ذلك.`,
    ),
  backToGrants: (l: Locale) => t(l, "Back to grants", "부여·위임으로", "العودة إلى المنح"),
  refusedTitle: (l: Locale) => t(l, "This change was refused", "이 변경이 거부되었습니다", "رُفض هذا التغيير"),
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
      "لا يمكن إبطال هذه الصلاحية هنا؛ فلا تُبطَل من هذه الشاشة صلاحيات الحد الأقصى لوحدة التحكم ولا صلاحياتك أنت. أُعيد تحميل القائمة.",
    ),
  stageBody: (l: Locale) =>
    t(l, "It is already closed or has ended. The list has been reloaded.", "이미 종료되었습니다. 목록을 다시 불러왔습니다.", "أُغلقت مسبقًا أو انتهت. أُعيد تحميل القائمة."),
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
  noKind: (l: Locale) => t(l, "No kind of data is listed, so an effect cannot be shown. Reload the page.", "데이터 종류가 목록에 없어 효력을 확인할 수 없습니다. 페이지를 다시 불러오세요.", "لا يوجد أي نوع بيانات مدرج، لذا لا يمكن عرض الأثر. أعد تحميل الصفحة."),
  showEffect: (l: Locale) => t(l, "Show effect", "효력 확인", "عرض الأثر"),
  previewLead: (l: Locale) => t(l, "This is a preview.", "미리 보기입니다.", "هذه معاينة فقط."),
  previewNote: (l: Locale) =>
    t(
      l,
      "It checks this person's grants and delegations for the console operations shown here. Those operations also need the console's own permission and a current sign-in. This preview does not determine whether the person can approve charges.",
      "여기에 표시된 콘솔 작업에 대한 사용자의 권한 부여와 위임을 확인합니다. 실제 작업에는 콘솔 권한과 유효한 로그인도 필요합니다. 이 미리 보기로 청구 승인 가능 여부를 판단할 수는 없습니다.",
      "تتحقق من صلاحيات هذا الشخص وتفويضاته لعمليات وحدة التحكم المعروضة هنا. تتطلب هذه العمليات أيضًا صلاحية وحدة التحكم وتسجيل دخول ساريًا. لا تحدد هذه المعاينة ما إذا كان الشخص يستطيع اعتماد الرسوم.",
    ),
  effectFor: (l: Locale, login: string) => t(l, `Effect for ${login}`, `${login}님의 효력`, `الأثر على ${login}`),
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
 propose_erp_result: ["Propose ERP result evidence", "ERP 결과 근거 제안", "اقتراح أدلة نتائج ERP"],
 confirm_erp_result: ["Confirm ERP result evidence", "ERP 결과 근거 확인", "تأكيد أدلة نتائج ERP"],
 revoke_erp_result: ["Revoke ERP result evidence", "ERP 결과 근거 철회", "إلغاء أدلة نتائج ERP"],
 apply_erp_result: ["Apply ERP result evidence internally", "ERP 결과 근거 내부 반영", "تطبيق أدلة نتائج ERP داخلياً"],
 register_issuance_channel: ["Register verification channel", "검증 채널 등록", "تسجيل قناة التحقق"],
 admit_issuance_channel: ["Accept verification channel", "검증 채널 수락", "قبول قناة التحقق"],
 request_issuance_protected_change: ["Request protected settings change", "보호된 설정 변경 요청", "طلب تغيير الإعدادات المحمية"],
 verify_issuance_protected_change: ["Verify protected settings change", "보호된 설정 변경 검증", "التحقق من تغيير الإعدادات المحمية"],
 approve_issuance_protected_change: ["Approve protected settings change", "보호된 설정 변경 승인", "اعتماد تغيير الإعدادات المحمية"],
 record_issuance_term: ["Record payment terms", "납부 조건 기록", "تسجيل شروط الدفع"],
 approve_issuance_term: ["Approve payment terms", "납부 조건 승인", "اعتماد شروط الدفع"],
 bind_issuance_term: ["Apply payment terms to a contract", "계약에 납부 조건 적용", "تطبيق شروط الدفع على عقد"],
 approve_issuance_term_binding: ["Approve applying payment terms to a contract", "계약에 납부 조건 적용 승인", "اعتماد تطبيق شروط الدفع على عقد"],
 submit_issuance_draft: ["Submit issuance draft for approval", "발행 초안 승인 요청", "إرسال مسودة الإصدار للاعتماد"],
 approve_issuance_draft: ["Approve issuance draft", "발행 초안 승인", "اعتماد مسودة الإصدار"],
 propose_document_evidence: ["Propose document evidence", "문서 근거 제안", "اقتراح أدلة المستند"],
 confirm_document_evidence: ["Confirm document evidence", "문서 근거 확인", "تأكيد أدلة المستند"],
 revoke_document_evidence: ["Revoke document evidence", "문서 근거 철회", "إلغاء أدلة المستند"],
 prepare_erp_record: ["Prepare internal ERP record", "내부 ERP 기록 준비", "إعداد سجل ERP داخلي"],
 approve: ["Approve", "승인", "اعتماد"],
 propose_pricing_policy: ["Propose pricing policy", "가격 정책 제안", "اقتراح سياسة التسعير"],
 approve_pricing_policy: ["Approve pricing policy", "가격 정책 승인", "اعتماد سياسة التسعير"],
 revoke_pricing_policy: ["Revoke pricing policy", "가격 정책 철회", "إلغاء سياسة التسعير"],
 use_pricing_policy: ["Use pricing policy", "가격 정책 적용", "تطبيق سياسة التسعير"],
  view: ["View", "조회", "عرض"],
  confirm_input: ["Confirm input data", "입력 데이터 확인", "تأكيد بيانات الإدخال"],
  view_cost: ["View cost", "원가 조회", "عرض التكلفة"],
  change_customer_policy: ["Change customer policy", "고객 정책 변경", "تغيير سياسة العميل"],
  change_selling_policy: ["Change selling policy", "판매 정책 변경", "تغيير سياسة البيع"],
  request_approval: ["Request approval", "승인 요청", "طلب الاعتماد"],
  issue: ["Issue", "발행", "إصدار"],
  download: ["Download", "다운로드", "تنزيل"],
  deliver: ["Deliver", "전달", "تسليم"],
};
const kinds: Record<string, [string, string, string]> = {
 synthetic_erp_result_evidence: ["Synthetic ERP result evidence", "합성 ERP 결과 근거", "أدلة نتائج ERP اصطناعية"],
 synthetic_issuance_protected: ["Synthetic protected issuance settings", "합성 보호된 발행 설정", "إعدادات إصدار محمية اصطناعية"],
 synthetic_issuance_terms: ["Synthetic payment terms", "합성 납부 조건", "شروط دفع اصطناعية"],
 synthetic_issuance_draft: ["Synthetic issuance drafts", "합성 발행 초안", "مسودات إصدار اصطناعية"],
 synthetic_document_evidence: ["Synthetic document evidence", "합성 문서 근거", "أدلة مستند اصطناعية"],
  erp_final_approval: ["ERP final-amount approval", "ERP 최종 금액 승인", "اعتماد المبلغ النهائي في ERP"],
  charge_approval: ["Charge approval", "청구 승인", "اعتماد الرسوم"],
  approval_review: ["Approval review records", "승인 검토 기록", "سجلات مراجعة الاعتمادات"],
 synthetic_pricing_policy: ["Synthetic pricing policy", "합성 시험용 가격 정책", "سياسة تسعير اصطناعية"],
  focus_data: ["Collected FOCUS data", "수집된 FOCUS 데이터", "بيانات FOCUS المجمّعة"],
  synthetic_input_evidence: ["Synthetic input evidence", "합성 입력 데이터 근거", "أدلة بيانات الإدخال الاصطناعية"],
  charge_candidate: ["Charge calculation candidates", "과금 계산 후보", "نتائج احتساب الرسوم المبدئية"],
};
const fields: Record<string, [string, string, string]> = {
 erp_result_evidence: ["ERP result evidence", "ERP 결과 근거", "أدلة نتائج ERP"],
 issuance_protected: ["Protected issuance settings evidence", "보호된 발행 설정 근거", "أدلة إعدادات الإصدار المحمية"],
 issuance_terms: ["Payment terms evidence", "납부 조건 근거", "أدلة شروط الدفع"],
 issuance_draft: ["Issuance draft evidence", "발행 초안 근거", "أدلة مسودة الإصدار"],
 document_evidence: ["Document evidence", "문서 근거", "أدلة المستند"],
  approval_evidence: ["Approval request and decision evidence", "승인 요청·결정 근거", "أدلة طلبات الاعتماد والقرارات"],
 pricing_policy_evidence: ["Pricing policy evidence", "가격 정책 근거", "أدلة سياسة التسعير"],
 input_review_evidence: ["Input comparison and reconciliation evidence", "입력 데이터 비교·대사 근거", "أدلة مقارنة بيانات الإدخال ومطابقتها"],
  rows: ["Rows", "행", "الصفوف"],
  charge_evidence: ["Source costs and calculation evidence", "원천 비용·계산 근거", "تكاليف المصدر وأدلة الاحتساب"],
};
const phrases: Record<string, [string, string, string]> = {
  "view|approval_review|approval_evidence": ["view approval request and decision records", "승인 요청·결정 기록 조회", "عرض سجلات طلبات الاعتماد والقرارات"],
  "approve|erp_final_approval|": ["Approve ERP final amounts", "ERP 최종 금액 승인", "اعتماد المبلغ النهائي في ERP"],
  "request_approval|erp_final_approval|": ["Request approval for ERP final amounts", "ERP 최종 금액 승인 요청", "طلب اعتماد المبلغ النهائي في ERP"],
  "approve|charge_approval|": ["approve charges", "청구 승인", "اعتماد الرسوم"],
  "request_approval|charge_approval|": ["request approval for charges", "청구 승인 요청", "طلب اعتماد الرسوم"],
  "view_cost|charge_candidate|charge_evidence": ["view source costs and evidence for charge calculation candidates", "과금 계산 후보의 원천 비용·계산 근거 조회", "عرض تكاليف المصدر وأدلة نتائج احتساب الرسوم المبدئية"],
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
