import type { Locale } from "../../../../shared/types";
import type { Storage } from "../../../../service/integrations";
import { text as t } from "../../../../shared/copy";
import {
  blockerWords, capabilityWords, licenseSourceWords, licenseStateWords, changeWords, clientKindWords, clientStateWords, componentWords, connectionWords, deletionWords, deliveryKindWords, deliveryReasonWords,
  destinationWords, eventStateWords, grantStateWords, historyWords, installReasonWords, installWords, jobWords, keyStateWords, letterRevoked, migrationWords, notPendingWords,
  proposalKindWords, proposalStateWords, resultWords, scopeWords, serviceUserWords, sourceWords, subscriptionReasonWords, subscriptionWords, unknownCode, unverifiedSentence,
  words,
} from "./live-codes";
import { dryRunBlocked, type Group } from "./live-model";

export const iso = (s: string) => (s ? "⁨" + s + "⁩" : s);
const has = (table: object, key: string) => Object.prototype.hasOwnProperty.call(table, key);
const pick = (l: Locale, w: [string, string, string]) => t(l, w[0], w[1], w[2]);

function letterAction(l: Locale, a: string): string {
  return a === "approve_redeliver"
    ? t(l, "Approve redelivery", "재발송 승인", "اعتماد إعادة التسليم")
      : a === "request_redeliver" ? t(l, "Request redelivery", "재발송 요청", "طلب إعادة التسليم")
        : a === "request_dismiss" ? t(l, "Request dismissal", "정리 요청", "طلب الإزالة")
          : t(l, "Approve dismissal", "정리 승인", "اعتماد الإزالة");
}

export const k = {
  reviewBeforeRetry: (l: Locale) => t(l, "Review the current state before trying again.", "다시 시도하기 전에 현재 상태를 확인하세요.", "راجع الحالة الحالية قبل المحاولة مرة أخرى."),
  deniedTitle: (l: Locale) => t(l, "Integrations are not available to you", "연동 화면을 볼 수 없습니다", "التكاملات غير متاحة لك"),
  denied: (l: Locale) =>
    t(l, "You do not have permission to manage integrations in this installation.", "이 설치에서 연동을 관리할 권한이 없습니다.", "ليست لديك صلاحية إدارة التكاملات في هذا التثبيت."),
  sessionEnded: (l: Locale) => t(l, "Your session has ended.", "세션이 끝났습니다.", "انتهت جلستك."),
  signInAgain: (l: Locale) => t(l, "Sign in again", "다시 로그인", "تسجيل الدخول مجددًا"),
  loading: (l: Locale) => t(l, "Loading integrations…", "연동 정보를 불러오는 중…", "جارٍ تحميل التكاملات…"),
  reload: (l: Locale) => t(l, "Reload", "새로 고침", "إعادة التحميل"),
  workspace: (l: Locale) => t(l, "Workspace ID", "업무 공간 ID", "معرّف مساحة العمل"),
  revision: (l: Locale) => t(l, "Data revision", "데이터 리비전", "مراجعة البيانات"),
  now: (l: Locale) => t(l, "Now (UTC)", "현재(UTC)", "الآن (UTC)"),
  viewTitle: (l: Locale, v: string) =>
    v === "modules" ? t(l, "Modules", "모듈", "الوحدات")
      : v === "channels" ? t(l, "Subscriptions and channels", "구독·채널", "الاشتراكات والقنوات")
        : v === "events" ? t(l, "Events and redelivery", "이벤트·재처리", "الأحداث وإعادة التسليم")
          : t(l, "API clients", "API client", "عملاء API"),
  statementLead: (l: Locale) => t(l, "Records decisions only.", "결정만 기록합니다.", "تسجيل القرارات فقط."),
  statement: (l: Locale) =>
    t(l, "Nothing is dispatched, sent, signed or issued from this screen in this build.", "이 빌드에서는 이 화면에서 아무것도 발송·전송·서명·발급하지 않습니다.", "لا يُرسَل أي شيء ولا يُوزَّع ولا يُوقَّع ولا يُصدَر من هذه الشاشة في هذا الإصدار."),
  clientsDenied: (l: Locale) => t(l, "You do not have permission to view API clients.", "API 클라이언트를 조회할 권한이 없습니다.", "ليس لديك إذن لعرض عملاء API."),
  clientsUnconfigured: (l: Locale) => t(l, "API clients are not configured in this workspace.", "이 업무 공간에는 API 클라이언트가 설정되어 있지 않습니다.", "عملاء API غير مُعدّين في مساحة العمل هذه."),
  notConfigured: (l: Locale) => t(l, "Integrations are not configured in this workspace", "이 업무 공간에는 연동이 설정되어 있지 않습니다", "التكاملات غير مُعدّة في مساحة العمل هذه"),
  stateUnknown: (l: Locale, code: string) =>
    t(l, `The integrations read returned an unknown state (${iso(code)}).`, `연동 조회가 알 수 없는 상태(${iso(code)})를 반환했습니다.`, `أعادت قراءة التكاملات حالة غير معروفة (${iso(code)}).`),
  unknown: (l: Locale) => t(l, "Unknown", "알 수 없음", "غير معروف"),
  unknownCode,
  none: (l: Locale) => t(l, "None", "없음", "لا شيء"),
  yes: (l: Locale) => t(l, "Yes", "예", "نعم"),
  no: (l: Locale) => t(l, "No", "아니요", "لا"),
  sectionEmpty: (l: Locale, s: string) =>
    s === "modules" ? t(l, "No plugins are registered in this workspace.", "이 업무 공간에 등록된 플러그인이 없습니다.", "لا توجد إضافات مسجّلة في مساحة العمل هذه.")
      : s === "channels" ? t(l, "No webhook subscriptions are registered.", "등록된 웹훅 구독이 없습니다.", "لا توجد اشتراكات webhook مسجّلة.")
        : t(l, "No event types are approved.", "승인된 이벤트 유형이 없습니다.", "لا توجد أنواع أحداث معتمدة."),
  vanishedLetter: (l: Locale) =>
    t(l, "Your earlier approval may have been applied. The dead letter is no longer listed.", "이전 승인이 이미 반영되었을 수 있습니다. 실패 보관 건이 더 이상 목록에 없습니다.", "ربما طُبّق اعتمادك السابق. لم تعد الرسالة غير المُسلّمة مدرجة."),
  vanishedProposal: (l: Locale) =>
    t(l, "Your earlier request may have been applied. The proposal is gone.", "이전 요청이 이미 반영되었을 수 있습니다. 제안이 없어졌습니다.", "ربما طُبّق طلبك السابق. اختفى المقترح."),
  vanishedHint: (l: Locale, event: string) =>
    event
      ? t(l, `Check the deliveries for a new queued redelivery of event ${iso(event)}.`, `발송 건 목록에서 이벤트 ${iso(event)}의 새 재발송 대기 건을 확인하세요.`, `تحقّق من عمليات التسليم بحثًا عن إعادة تسليم جديدة في الانتظار للحدث ${iso(event)}.`)
      : t(l, "Check the deliveries for a new queued redelivery.", "발송 건 목록에서 새 재발송 대기 건을 확인하세요.", "تحقّق من عمليات التسليم بحثًا عن إعادة تسليم جديدة في الانتظار."),
  vanishedOther: (l: Locale) =>
    t(l, "This item is no longer listed. Your earlier request may have been applied.", "이 항목은 더 이상 목록에 없습니다. 이전 요청이 이미 반영되었을 수 있습니다.", "لم يعد هذا العنصر مدرجًا. ربما طُبّق طلبك السابق."),
  dismissNotice: (l: Locale) => t(l, "Dismiss", "닫기", "إغلاق"),

  connection: (l: Locale, s: string) => words(l, connectionWords, s),
  clientsLead: (l: Locale) =>
    t(
      l,
      "OAuth and API client registration is not connected in this build. No client credential or token is issued here.",
      "이 빌드에서는 OAuth·API client 등록이 연결되어 있지 않습니다. 여기에서는 client 자격 증명이나 토큰을 발급하지 않습니다.",
      "تسجيل عملاء OAuth وواجهات API غير متصل في هذا الإصدار. لا تُصدَر هنا أي بيانات اعتماد أو رموز للعملاء.",
    ),
  clientsH3: (l: Locale) => t(l, "Clients", "클라이언트", "العملاء"),
  readOnly: (l: Locale) =>
    t(l, "The data revision is unknown, so this screen is read-only. Reload to try again.", "데이터 리비전을 알 수 없어 이 화면은 읽기 전용입니다. 새로 고침해 다시 시도하세요.", "مراجعة البيانات غير معروفة، لذا هذه الشاشة للقراءة فقط. أعد التحميل للمحاولة مجددًا."),
  mayApplied: (l: Locale) =>
    t(l, "Your earlier request may have been applied. Check this proposal's state.", "이전 요청이 이미 반영되었을 수 있습니다. 이 제안의 상태를 확인하세요.", "ربما طُبّق طلبك السابق. تحقّق من حالة هذا المقترح."),
  clientsTitle: (l: Locale) => t(l, "API clients", "API client", "عملاء API"),
  colClient: (l: Locale) => t(l, "Client", "Client", "العميل"),
  colKind: (l: Locale) => t(l, "Kind", "종류", "النوع"),
  colState: (l: Locale) => t(l, "State", "상태", "الحالة"),
  colGrants: (l: Locale) => t(l, "Grants", "권한 부여", "المنح"),
  clientKind: (l: Locale, s: string) => words(l, clientKindWords, s),
  clientState: (l: Locale, s: string) => words(l, clientStateWords, s),
  noClients: (l: Locale) => t(l, "No API clients.", "API client가 없습니다.", "لا يوجد عملاء API."),
  grantsTitle: (l: Locale) => t(l, "Console grants", "콘솔 권한 부여", "منح وحدة التحكم"),
  grantsFor: (l: Locale, id: string) => t(l, `Client ${iso(id)}`, `Client ${iso(id)}`, `العميل ${iso(id)}`),
  colGrant: (l: Locale) => t(l, "Grant", "권한 부여", "المنحة"),
  colAction: (l: Locale) => t(l, "Action", "작업", "الإجراء"),
  colDataKind: (l: Locale) => t(l, "Data kind", "데이터 종류", "نوع البيانات"),
  colScope: (l: Locale) => t(l, "Scope", "범위", "النطاق"),
  colFrom: (l: Locale) => t(l, "Valid from", "유효 시작", "صالح من"),
  colUntil: (l: Locale) => t(l, "Valid until", "유효 종료", "صالح حتى"),
  scopeCustomer: (l: Locale) => t(l, "Customer", "고객", "العميل"),
  scopeLegal: (l: Locale) => t(l, "Legal entity", "법인", "الكيان القانوني"),
  scopeContract: (l: Locale) => t(l, "Contract", "계약", "العقد"),
  scopePeriod: (l: Locale) => t(l, "Period", "기간", "الفترة"),
  scopeWord: (l: Locale, s: string) => (has(scopeWords, s) ? words(l, scopeWords, s) : ""),
  grantState: (l: Locale, s: string) => words(l, grantStateWords, s),
  noGrants: (l: Locale) =>
    t(l, "This client has no grants shown here. Installation administration grants are not listed.", "여기에 표시할 이 client의 권한 부여가 없습니다. 설치 관리 권한 부여는 표시하지 않습니다.", "لا توجد منح معروضة لهذا العميل هنا. منح إدارة التثبيت غير مدرجة."),
  serviceUsersTitle: (l: Locale) => t(l, "Service users", "서비스 사용자", "مستخدمو الخدمة"),
  colServiceUser: (l: Locale) => t(l, "Service user", "서비스 사용자", "مستخدم الخدمة"),
  colOwner: (l: Locale) => t(l, "Owner", "담당자", "المالك"),
  colUsable: (l: Locale) => t(l, "Usable credentials", "사용 가능한 자격 증명", "بيانات الاعتماد القابلة للاستخدام"),
  noneUsable: (l: Locale) => t(l, "None usable", "사용 가능 없음", "لا شيء قابل للاستخدام"),
  serviceUserState: (l: Locale, s: string) => words(l, serviceUserWords, s),
  noServiceUsers: (l: Locale) => t(l, "No service users.", "서비스 사용자가 없습니다.", "لا يوجد مستخدمو خدمة."),
  manageOn: (l: Locale) =>
    t(l, "Service users and their credentials are managed on M04 (Service accounts)", "서비스 사용자와 자격 증명은 M04(서비스 계정)에서 관리합니다", "تُدار مستخدمو الخدمة وبيانات اعتمادهم في M04 (حسابات الخدمة)"),
  manageOnUnknown: (l: Locale) => t(l, "Where service users are managed:", "서비스 사용자 관리 위치:", "مكان إدارة مستخدمي الخدمة:"),
  oauthTitle: (l: Locale) => t(l, "OAuth clients", "OAuth client", "عملاء OAuth"),

  modulesLead: (l: Locale) =>
    t(
      l,
      "Approving records the decision in the plugin registry. No plugin runs and no plugin token is issued: the plugin runtime is not connected in this build.",
      "승인하면 플러그인 레지스트리에 결정이 기록됩니다. 이 빌드에서는 플러그인 실행 환경이 연결되어 있지 않아 플러그인이 실행되지 않고 플러그인 토큰도 발급되지 않습니다.",
      "يسجّل الاعتماد القرار في سجل الإضافات. لا تعمل أي إضافة ولا يُصدَر رمز لها، لأن بيئة تشغيل الإضافات غير متصلة في هذا الإصدار.",
    ),
  licenseLabel: (l: Locale) => t(l, "License", "라이선스", "الترخيص"),
  licenseState: (l: Locale, s: string) => (has(licenseStateWords, s) ? words(l, licenseStateWords, s) : t(l, "Unknown. Treated as not permitted.", "알 수 없음. 허용되지 않은 것으로 처리합니다.", "غير معروف. يُعامل على أنه غير مسموح به.")),
  licenseConsequence: (l: Locale) => t(l, "It cannot be installed, upgraded to or enabled.", "설치·업그레이드·사용 설정할 수 없습니다.", "لا يمكن تثبيته أو الترقية إليه أو تفعيله."),
  licenseUnknown: (l: Locale) => t(l, "License: unknown. Treated as not permitted.", "라이선스: 알 수 없음. 허용되지 않은 것으로 처리합니다.", "الترخيص: غير معروف. يُعامل على أنه غير مسموح به."),
  licensesTitle: (l: Locale, source: string) =>
    has(licenseSourceWords, source)
      ? t(l, `Permitted plugin licenses (${words(l, licenseSourceWords, source)})`, `허용 플러그인 라이선스(${words(l, licenseSourceWords, source)})`, `تراخيص الإضافات المسموح بها (${words(l, licenseSourceWords, source)})`)
      : t(l, "Permitted plugin licenses (source unknown)", "허용 플러그인 라이선스(출처 알 수 없음)", "تراخيص الإضافات المسموح بها (المصدر غير معروف)"),
  licensesNote: (l: Locale) =>
    t(
      l,
      "An installation setting: it changes only when the server is restarted with a different list. This list is an installation policy, not a legal review.",
      "설치 설정입니다. 다른 목록으로 서버를 다시 시작해야 바뀝니다. 이 목록은 설치 정책이며 법률 검토가 아닙니다.",
      "هذا إعداد تثبيت، ولا يتغيّر إلا بإعادة تشغيل الخادم بقائمة مختلفة. هذه القائمة سياسة تثبيت وليست مراجعة قانونية.",
    ),
  licensesEmpty: (l: Locale) =>
    t(l, "No plugin license is permitted on this installation, so no plugin can be installed, upgraded to or enabled.", "이 설치에서 허용된 플러그인 라이선스가 없어 어떤 플러그인도 설치·업그레이드·사용 설정할 수 없습니다.", "لا يُسمح بأي ترخيص إضافة في هذا التثبيت، لذا لا يمكن تثبيت أي إضافة أو الترقية إليها أو تفعيلها."),
  licensesUnknown: (l: Locale) =>
    t(l, "The list of permitted plugin licenses is unknown, so no plugin can be installed, upgraded to or enabled from here.", "허용 플러그인 라이선스 목록을 알 수 없어 여기에서 플러그인을 설치·업그레이드·사용 설정할 수 없습니다.", "قائمة تراخيص الإضافات المسموح بها غير معروفة، لذا لا يمكن تثبيت أي إضافة أو الترقية إليها أو تفعيلها من هنا."),
  notOffered: (l: Locale, v: string, id: string) =>
    t(l, `Version ${iso(v)} is not offered: its license ${iso(id)} is not permitted on this installation.`, `${iso(v)} 버전은 제안할 수 없습니다. 라이선스 ${iso(id)}이(가) 이 설치에서 허용되지 않습니다.`, `لا يُعرض الإصدار ${iso(v)}: ترخيصه ${iso(id)} غير مسموح به في هذا التثبيت.`),
  notOfferedUnknown: (l: Locale, v: string) =>
    t(l, `Version ${iso(v)} is not offered: its license is unknown, so it is treated as not permitted.`, `${iso(v)} 버전은 제안할 수 없습니다. 라이선스를 알 수 없어 허용되지 않은 것으로 처리합니다.`, `لا يُعرض الإصدار ${iso(v)}: ترخيصه غير معروف، لذا يُعامل على أنه غير مسموح به.`),
  enableNotOffered: (l: Locale, v: string, id: string) =>
    t(l, `Enable is not offered: the installed version ${iso(v)} has the license ${iso(id)}, which is not permitted on this installation.`, `사용 설정은 제안할 수 없습니다. 설치된 ${iso(v)} 버전의 라이선스 ${iso(id)}은(는) 이 설치에서 허용되지 않습니다.`, `لا يُعرض التفعيل: للإصدار المثبّت ${iso(v)} الترخيص ${iso(id)}، وهو غير مسموح به في هذا التثبيت.`),
  enableNotOfferedUnknown: (l: Locale, v: string) =>
    t(l, `Enable is not offered: the license of the installed version ${iso(v)} is unknown, so it is treated as not permitted.`, `사용 설정은 제안할 수 없습니다. 설치된 ${iso(v)} 버전의 라이선스를 알 수 없어 허용되지 않은 것으로 처리합니다.`, `لا يُعرض التفعيل: ترخيص الإصدار المثبّت ${iso(v)} غير معروف، لذا يُعامل على أنه غير مسموح به.`),
  targetLicense: (l: Locale, v: string) => t(l, `License of ${iso(v)}`, `${iso(v)} 라이선스`, `ترخيص ${iso(v)}`),
  workspaceTitle: (l: Locale) => t(l, "Plugin workspace", "플러그인 업무 공간", "مساحة عمل الإضافات"),
  workspaceId: (l: Locale) => t(l, "Workspace", "업무 공간", "مساحة العمل"),
  runtimeIdentity: (l: Locale) => t(l, "Runtime identity", "실행 ID", "هوية التشغيل"),
  verificationWs: (l: Locale) => t(l, "Verification workspace", "검증용 업무 공간", "مساحة عمل للتحقق"),
  noWorkspace: (l: Locale) =>
    t(l, "No plugin workspace is registered. Plugin changes cannot be proposed.", "등록된 플러그인 업무 공간이 없습니다. 플러그인 변경을 제안할 수 없습니다.", "لا توجد مساحة عمل إضافات مسجّلة. لا يمكن اقتراح تغييرات على الإضافات."),
  publishersTitle: (l: Locale) => t(l, "Publishers", "플러그인 게시자", "الناشرون"),
  noPublishers: (l: Locale) => t(l, "No publishers are registered.", "등록된 게시자가 없습니다.", "لا يوجد ناشرون مسجّلون."),
  notRevoked: (l: Locale) => t(l, "Not revoked", "회수되지 않음", "غير مُلغى"),
  revokedBy: (l: Locale, who: string) => t(l, `Revoked by ${iso(who)}:`, `${iso(who)}이(가) 회수:`, `ألغاه ${iso(who)}:`),
  confirmedBy: (l: Locale, who: string) => t(l, `confirmed by ${iso(who)}`, `${iso(who)} 확인`, `أكّده ${iso(who)}`),
  keyLabel: (l: Locale) => t(l, "Key", "키", "المفتاح"),
  validRange: (l: Locale, a: string, b: string) => t(l, `valid ${iso(a)} – ${iso(b)}`, `${iso(a)} – ${iso(b)} 유효`, `صالح ${iso(a)} – ${iso(b)}`),
  current: (l: Locale) => t(l, "Current", "사용 중", "حالي"),
  retired: (l: Locale) => t(l, "Retired", "폐기됨", "متقاعد"),
  compromise: (l: Locale) => t(l, "Compromise", "손상 표시", "الاختراق"),
  compromisedBy: (l: Locale, who: string) => t(l, `marked by ${iso(who)}:`, `${iso(who)} 표시:`, `وسمه ${iso(who)}:`),
  noKeysPublisher: (l: Locale) => t(l, "No keys", "키 없음", "لا مفاتيح"),
  registeringNotConnected: (l: Locale) =>
    t(l, "Registering publishers, keys and versions is not connected in this build.", "이 빌드에서는 게시자·키·버전 등록이 연결되어 있지 않습니다.", "تسجيل الناشرين والمفاتيح والإصدارات غير متصل في هذا الإصدار."),
  installTitle: (l: Locale) => t(l, "Installation", "설치", "التثبيت"),
  notInstalledPlugin: (l: Locale) => t(l, "Not installed in this workspace.", "이 업무 공간에 설치되지 않았습니다.", "غير مثبّتة في مساحة العمل هذه."),
  installState: (l: Locale, s: string) => words(l, installWords, s),
  installReason: (l: Locale, s: string) => words(l, installReasonWords, s),
  colReason: (l: Locale) => t(l, "Reason", "사유", "السبب"),
  registryRevision: (l: Locale) => t(l, "Registry revision", "레지스트리 리비전", "مراجعة السجل"),
  tokens: (l: Locale) => t(l, "Tokens", "토큰", "الرموز"),
  runtime: (l: Locale) => t(l, "Runtime", "실행 환경", "بيئة التشغيل"),
  endpointsTitle: (l: Locale) => t(l, "Approved endpoints", "승인된 엔드포인트", "نقاط النهاية المعتمدة"),
  colEndpoint: (l: Locale) => t(l, "Endpoint", "엔드포인트", "نقطة النهاية"),
  colFingerprint: (l: Locale) => t(l, "Certificate fingerprint", "인증서 지문", "بصمة الشهادة"),
  colApprovedBy: (l: Locale) => t(l, "Approved by", "승인자", "المعتمِد"),
  colProposedBy: (l: Locale) => t(l, "Proposed by", "제안자", "المقترِح"),
  noEndpoints: (l: Locale) => t(l, "No approved endpoints", "승인된 엔드포인트가 없습니다", "لا توجد نقاط نهاية معتمدة"),
  storageAfterRemoval: (l: Locale) => t(l, "Storage after removal", "제거 후 저장소", "التخزين بعد الإزالة"),
  keptDays: (l: Locale, n: number) =>
    t(l, `Kept for evidence for ${n} days after removal`, `제거 후 ${n}일 동안 증빙용으로 보관`, `تُحفظ كدليل لمدة ${n} يومًا بعد الإزالة`),
  keptUnknown: (l: Locale) =>
    t(l, "Kept for evidence after removal for an unknown number of days", "제거 후 증빙용으로 보관(기간 알 수 없음)", "تُحفظ كدليل بعد الإزالة لعدد غير معروف من الأيام"),
  deletion: (l: Locale, s: string) => words(l, deletionWords, s),
  migration: (l: Locale, s: string) => words(l, migrationWords, s),
  stop: (l: Locale) => t(l, "Stop", "중지", "إيقاف"),
  disable: (l: Locale) => t(l, "Disable", "사용 중지", "تعطيل"),
  confirmStop: (l: Locale) => t(l, "Confirm stop", "중지 확인", "تأكيد الإيقاف"),
  confirmDisable: (l: Locale) => t(l, "Confirm disable", "사용 중지 확인", "تأكيد التعطيل"),
  stopImpact: (l: Locale) =>
    t(
      l,
      "Takes effect now; no second approval. Stop records the plugin as stopped. It does not recall anything already sent; in this build no plugin has run.",
      "즉시 적용되며 두 번째 승인이 없습니다. 중지하면 플러그인이 중지됨으로 기록됩니다. 이미 보낸 것을 회수하지 않으며, 이 빌드에서는 실행된 플러그인이 없습니다.",
      "يسري فورًا؛ دون اعتماد ثانٍ. يسجّل الإيقاف الإضافة على أنها متوقفة. لا يسترجع أي شيء أُرسل سابقًا؛ ولم تعمل أي إضافة في هذا الإصدار.",
    ),
  disableImpact: (l: Locale) =>
    t(
      l,
      "Takes effect now; no second approval. Disable returns the plugin to installed. A new enable proposal and another administrator's approval are needed to enable it again.",
      "즉시 적용되며 두 번째 승인이 없습니다. 사용 중지하면 플러그인이 설치됨 상태로 돌아갑니다. 다시 사용 설정하려면 새 사용 설정 제안과 다른 관리자의 승인이 필요합니다.",
      "يسري فورًا؛ دون اعتماد ثانٍ. يعيد التعطيل الإضافة إلى حالة مثبّتة. يلزم مقترح تفعيل جديد واعتماد مسؤول آخر لتفعيلها مجددًا.",
    ),
  stopVerb: (l: Locale, p: string) => t(l, `Stop ${iso(p)}`, `${iso(p)} 중지`, `إيقاف ${iso(p)}`),
  disableVerb: (l: Locale, p: string) => t(l, `Disable ${iso(p)}`, `${iso(p)} 사용 중지`, `تعطيل ${iso(p)}`),
  runtimeDone: (l: Locale, p: string, state: string) => t(l, `${iso(p)} is now: ${state}.`, `${iso(p)} 현재 상태: ${state}.`, `${iso(p)} الآن: ${state}.`),
  proposeChange: (l: Locale, p: string) => t(l, `Propose a change to ${iso(p)}`, `${iso(p)} 변경 제안`, `اقتراح تغيير على ${iso(p)}`),
  proposeBlocker: (l: Locale, code: string, n: string) => {
    switch (code) {
      case "no_command":
        return t(l, "No command is available for this row.", "이 행에는 사용할 수 있는 명령이 없습니다.", "لا يتوفر أمر لهذا الصف.");
      case "unknown_list":
        return t(l, "The list of proposals is unknown, so no change can be proposed.", "제안 목록을 알 수 없어 변경을 제안할 수 없습니다.", "قائمة المقترحات غير معروفة، لذا لا يمكن اقتراح أي تغيير.");
      case "no_workspace":
        return t(l, "No plugin workspace is registered.", "등록된 플러그인 업무 공간이 없습니다.", "لا توجد مساحة عمل إضافات مسجّلة.");
      case "no_version":
        return t(l, "No other version is registered.", "등록된 다른 버전이 없습니다.", "لا يوجد إصدار آخر مسجّل.");
      case "untrusted":
        return t(l, "No registered version can be proposed: each is marked vulnerable, has a license not permitted on this installation, or its publisher or key is revoked or compromised.", "제안할 수 있는 등록 버전이 없습니다. 모두 취약 표시가 있거나, 라이선스가 이 설치에서 허용되지 않거나, 게시자·키가 회수 또는 손상되었습니다.", "لا يمكن اقتراح أي إصدار مسجّل: كل منها موسوم بأنه معرّض للخطر، أو ترخيصه غير مسموح به في هذا التثبيت، أو ناشره أو مفتاحه مُلغى أو مخترق.");
      case "license_unknown":
        return t(l, "No registered version can be proposed: the license of each is unknown or not permitted on this installation, and an unknown license is treated as not permitted.", "제안할 수 있는 등록 버전이 없습니다. 모든 버전의 라이선스가 알 수 없거나 이 설치에서 허용되지 않으며, 알 수 없는 라이선스는 허용되지 않은 것으로 처리합니다.", "لا يمكن اقتراح أي إصدار مسجّل: ترخيص كل منها غير معروف أو غير مسموح به في هذا التثبيت، ويُعامل الترخيص غير المعروف على أنه غير مسموح به.");
      case "license":
        return t(l, "No registered version can be proposed: the license of each is not permitted on this installation.", "제안할 수 있는 등록 버전이 없습니다. 모든 버전의 라이선스가 이 설치에서 허용되지 않습니다.", "لا يمكن اقتراح أي إصدار مسجّل: ترخيص كل منها غير مسموح به في هذا التثبيت.");
      case "unknown_state":
        return t(l, "The plugin's installation state is unknown, so no change can be proposed.", "플러그인 설치 상태를 알 수 없어 변경을 제안할 수 없습니다.", "حالة تثبيت الإضافة غير معروفة، لذا لا يمكن اقتراح أي تغيير.");
      case "duplicate_own":
        return t(l, `Your proposal #${n} already asks for this.`, `본인의 제안 #${n}이(가) 이미 이 변경을 요청합니다.`, `مقترحك رقم ${n} يطلب هذا بالفعل.`);
      case "duplicate":
        return t(l, `Proposal #${n} already asks for this.`, `제안 #${n}이(가) 이미 이 변경을 요청합니다.`, `المقترح رقم ${n} يطلب هذا بالفعل.`);
    }
    return t(l, "This is not available now.", "지금은 사용할 수 없습니다.", "هذا غير متاح الآن.");
  },
  change: (l: Locale, kind: string, v: string) => (has(changeWords, kind) ? words(l, changeWords, kind).replace("{v}", v) : unknownCode(l, kind)),
  changeLegend: (l: Locale) => t(l, "Change", "변경", "التغيير"),
  chooseChange: (l: Locale) => t(l, "Choose a change.", "변경을 선택하세요.", "اختر تغييرًا."),
  confirmChange: (l: Locale) => t(l, "Confirm plugin change", "플러그인 변경 확인", "تأكيد تغيير الإضافة"),
  factPlugin: (l: Locale) => t(l, "Plugin", "플러그인", "الإضافة"),
  factVersion: (l: Locale) => t(l, "Version", "버전", "الإصدار"),
  factInstalled: (l: Locale) => t(l, "Installed version and state", "설치된 버전과 상태", "الإصدار المثبّت وحالته"),
  notInstalled: (l: Locale) => t(l, "Not installed", "설치되지 않음", "غير مثبّتة"),
  proposeImpact: (l: Locale) =>
    t(
      l,
      "Another administrator must approve it. Approval records the installation in the plugin registry; no plugin code is deployed or run in this build.",
      "다른 관리자가 승인해야 합니다. 승인하면 플러그인 레지스트리에 설치가 기록되며, 이 빌드에서는 플러그인 코드를 배포하거나 실행하지 않습니다.",
      "يجب أن يعتمده مسؤول آخر. يسجّل الاعتماد التثبيت في سجل الإضافات؛ ولا تُنشر أي شيفرة إضافة ولا تُشغَّل في هذا الإصدار.",
    ),
  proposeVerb: (l: Locale, change: string, p: string) => t(l, `Propose ${change} for ${iso(p)}`, `${iso(p)}에 ${change} 제안`, `اقتراح ${change} لـ ${iso(p)}`),
  proposedDone: (l: Locale, n: string) => t(l, `Proposal #${n} recorded. Another administrator must approve it.`, `제안 #${n}을(를) 기록했습니다. 다른 관리자가 승인해야 합니다.`, `سُجّل المقترح رقم ${n}. يجب أن يعتمده مسؤول آخر.`),
  duplicateProposal: (l: Locale, n: string) => t(l, `This was already proposed as #${n}. No new proposal was created, and that proposal keeps its original reason.`, `이미 #${n}(으)로 제안되었습니다. 새 제안은 만들어지지 않았으며, 해당 제안의 기존 사유가 유지됩니다.`, `اقتُرح هذا بالفعل برقم ${n}. لم يُنشأ مقترح جديد، ويبقى سبب ذلك المقترح الأصلي.`),

  proposalsTitle: (l: Locale) => t(l, "Plugin proposals", "플러그인 제안", "مقترحات الإضافات"),
  waitingGroup: (l: Locale, n: number) => t(l, `Waiting for approval (${n})`, `승인 대기(${n})`, `بانتظار الاعتماد (${n})`),
  closedGroup: (l: Locale, n: number) => t(l, `Closed (${n})`, `종료(${n})`, `مغلقة (${n})`),
  noProposals: (l: Locale) => t(l, "No plugin proposals.", "플러그인 제안이 없습니다.", "لا توجد مقترحات إضافات."),
  proposalN: (l: Locale, n: string) => t(l, `Proposal ${n}`, `제안 ${n}`, `المقترح ${n}`),
  proposalKind: (l: Locale, s: string) => words(l, proposalKindWords, s),
  proposalState: (l: Locale, s: string) => words(l, proposalStateWords, s),
  proposedBy: (l: Locale, who: string) => t(l, `Proposed by ${iso(who)}`, `제안자 ${iso(who)}`, `اقترحه ${iso(who)}`),
  created: (l: Locale, d: string) => t(l, `Created ${iso(d)}`, `${iso(d)} 생성`, `أُنشئ ${iso(d)}`),
  reason: (l: Locale) => t(l, "Reason", "사유", "السبب"),
  noReason: (l: Locale) => t(l, "No reason recorded", "기록된 사유 없음", "لا سبب مسجّل"),
  approvedBy: (l: Locale, who: string) => t(l, `Approved by ${iso(who)}`, `승인자 ${iso(who)}`, `اعتمده ${iso(who)}`),
  withdrawnBy: (l: Locale, who: string) => t(l, `Withdrawn by ${iso(who)}`, `철회자 ${iso(who)}`, `سحبه ${iso(who)}`),
  closedBy: (l: Locale, who: string) => t(l, `Closed by ${iso(who)}`, `종료자 ${iso(who)}`, `أغلقه ${iso(who)}`),
  closingReason: (l: Locale) => t(l, "Closing reason", "종료 사유", "سبب الإغلاق"),
  approve: (l: Locale) => t(l, "Approve", "승인", "اعتماد"),
  withdraw: (l: Locale) => t(l, "Withdraw", "철회", "سحب"),
  confirmApprove: (l: Locale) => t(l, "Confirm approval", "승인 확인", "تأكيد الاعتماد"),
  confirmWithdraw: (l: Locale) => t(l, "Confirm withdrawal", "철회 확인", "تأكيد السحب"),
  approveImpact: (l: Locale) =>
    t(l, "Approval records the change in the plugin registry. No plugin code runs and no token is issued in this build.", "승인하면 플러그인 레지스트리에 변경이 기록됩니다. 이 빌드에서는 플러그인 코드가 실행되지 않고 토큰도 발급되지 않습니다.", "يسجّل الاعتماد التغيير في سجل الإضافات. لا تعمل أي شيفرة إضافة ولا يُصدَر أي رمز في هذا الإصدار."),
  withdrawImpact: (l: Locale) => t(l, "Withdrawing closes this proposal. Nothing else changes.", "철회하면 이 제안이 종료됩니다. 다른 것은 바뀌지 않습니다.", "يُغلق السحب هذا المقترح. لا يتغيّر شيء آخر."),
  approveVerb: (l: Locale, n: string) => t(l, `Approve proposal ${iso(n)}`, `제안 ${iso(n)} 승인`, `اعتماد المقترح ${iso(n)}`),
  withdrawVerb: (l: Locale, n: string) => t(l, `Withdraw proposal ${iso(n)}`, `제안 ${iso(n)} 철회`, `سحب المقترح ${iso(n)}`),
  approvedDone: (l: Locale, n: string, s: string) => t(l, `Proposal ${n}: ${s}. Nothing runs in this build.`, `제안 ${n}: ${s}. 이 빌드에서는 아무것도 실행되지 않습니다.`, `المقترح ${n}: ${s}. لا يعمل شيء في هذا الإصدار.`),
  withdrawnDone: (l: Locale, n: string) => t(l, `Proposal ${n} withdrawn.`, `제안 ${n}을(를) 철회했습니다.`, `سُحب المقترح ${n}.`),
  noCommand: (l: Locale) => t(l, "No command is available for this row.", "이 행에는 사용할 수 있는 명령이 없습니다.", "لا يتوفر أمر لهذا الصف."),
  factProposal: (l: Locale) => t(l, "Proposal", "제안", "المقترح"),
  factKind: (l: Locale) => t(l, "Kind", "종류", "النوع"),
  factSummary: (l: Locale) => t(l, "Summary", "요약", "الملخص"),
  factProposerReason: (l: Locale) => t(l, "Proposer's reason", "제안 사유", "سبب صاحب المقترح"),

  versionsTitle: (l: Locale) => t(l, "Plugin versions", "플러그인 버전", "إصدارات الإضافات"),
  noVersions: (l: Locale) => t(l, "No plugin versions are registered.", "등록된 플러그인 버전이 없습니다.", "لا توجد إصدارات إضافات مسجّلة."),
  colPlugin: (l: Locale) => t(l, "Plugin", "플러그인", "الإضافة"),
  colVersion: (l: Locale) => t(l, "Version", "버전", "الإصدار"),
  digest: (l: Locale) => t(l, "Digest", "다이제스트", "الملخّص"),
  colCapabilities: (l: Locale) => t(l, "Capabilities", "권한 범위", "القدرات"),
  colEgress: (l: Locale) => t(l, "Egress and endpoints", "외부 연결과 엔드포인트", "الاتصال الخارجي ونقاط النهاية"),
  egress: (l: Locale) => t(l, "Egress", "외부 연결", "الاتصال الخارجي"),
  endpointsLabel: (l: Locale) => t(l, "Endpoints", "엔드포인트", "نقاط النهاية"),
  colVerification: (l: Locale) => t(l, "Verification", "검증", "التحقق"),
  signedTemplate: (l: Locale) => t(l, "Signed with key {key} of {publisher}", "{publisher}의 키 {key}(으)로 서명됨", "موقّع بالمفتاح {key} من {publisher}"),
  core: (l: Locale) => t(l, "Core", "코어", "النواة"),
  colVulnerable: (l: Locale) => t(l, "Vulnerable", "취약 표시", "معرّض للخطر"),
  markedVulnerable: (l: Locale, who: string) => t(l, `Marked vulnerable by ${iso(who)}:`, `${iso(who)}이(가) 취약 표시:`, `وسمه ${iso(who)} بأنه معرّض للخطر:`),
  capability: (l: Locale, s: string) => words(l, capabilityWords, s),
  limitsFold: (l: Locale, v: string) => t(l, `Limits, storage and screens for ${v}`, `${v}의 한도·저장소·화면`, `الحدود والتخزين والشاشات للإصدار ${v}`),
  limits: (l: Locale) => t(l, "Limits", "한도", "الحدود"),
  seconds: (l: Locale, n: number) => t(l, `${n} s`, `${n}초`, `${n} ث`),
  mib: (l: Locale, n: number) => t(l, `${n} MiB`, `${n} MiB`, `${n} MiB`),
  concurrent: (l: Locale, n: number) => t(l, `${n} concurrent`, `동시 ${n}개`, `${n} متزامنة`),
  timeoutLabel: (l: Locale) => t(l, "Timeout", "제한 시간", "المهلة"),
  memoryLabel: (l: Locale) => t(l, "Memory", "메모리", "الذاكرة"),
  concurrencyLabel: (l: Locale) => t(l, "Concurrency", "동시 실행", "التزامن"),
  storageContract: (l: Locale) => t(l, "Storage contract", "저장 계약", "عقد التخزين"),
  namespace: (l: Locale) => t(l, "Namespace", "네임스페이스", "مساحة الأسماء"),
  owner: (l: Locale) => t(l, "Owner", "담당", "المالك"),
  screensTitle: (l: Locale) => t(l, "Declared screens", "선언된 화면", "الشاشات المعلنة"),
  noScreens: (l: Locale) => t(l, "No screens declared", "선언된 화면 없음", "لا شاشات معلنة"),
  component: (l: Locale, s: string) => words(l, componentWords, s),

  channelsLead: (l: Locale) =>
    t(
      l,
      "Approving or resuming a subscription records it. No webhook is signed or sent: the signer and the dispatcher are not connected in this build.",
      "구독을 승인하거나 재개하면 기록만 됩니다. 이 빌드에서는 서명기와 발송기가 연결되어 있지 않아 웹훅을 서명하거나 보내지 않습니다.",
      "يسجّل اعتماد الاشتراك أو استئنافه القرار فقط. لا يُوقَّع أي webhook ولا يُرسَل، لأن أداة التوقيع والمُرسِل غير متصلين في هذا الإصدار.",
    ),
  destinationsTitle: (l: Locale) => t(l, "Destinations", "목적지", "الوجهات"),
  colDestination: (l: Locale) => t(l, "Destination", "목적지", "الوجهة"),
  colSource: (l: Locale) => t(l, "Source", "출처", "المصدر"),
  destinationState: (l: Locale, s: string) => words(l, destinationWords, s),
  source: (l: Locale, s: string) => words(l, sourceWords, s),
  destinationsNote: (l: Locale) =>
    t(l, "Destination addresses and redirects are not checked in this build; the list is a fixed sample allowlist.", "이 빌드에서는 목적지 주소와 리디렉션을 확인하지 않으며, 목록은 고정된 샘플 허용 목록입니다.", "لا يُتحقَّق من عناوين الوجهات وإعادات التوجيه في هذا الإصدار؛ والقائمة قائمة سماح تجريبية ثابتة."),
  noDestinations: (l: Locale) => t(l, "No destinations.", "목적지가 없습니다.", "لا توجد وجهات."),
  subscriptionsTitle: (l: Locale) => t(l, "Subscriptions", "구독", "الاشتراكات"),
  noSubscriptions: (l: Locale) => t(l, "No subscriptions.", "구독이 없습니다.", "لا توجد اشتراكات."),
  subscriptionState: (l: Locale, s: string) => words(l, subscriptionWords, s),
  subscriptionReason: (l: Locale, s: string) => words(l, subscriptionReasonWords, s),
  subscriberIs: (l: Locale) => t(l, "Subscriber", "구독자", "المشترك"),
  destinationIs: (l: Locale) => t(l, "Destination", "목적지", "الوجهة"),
  eventTypes: (l: Locale) => t(l, "Event types", "이벤트 유형", "أنواع الأحداث"),
  cursorLag: (l: Locale) => t(l, "Cursor and lag", "커서와 지연", "المؤشر والتأخر"),
  cursorWord: (l: Locale) => t(l, "Cursor", "커서", "المؤشر"),
  behind: (l: Locale) => t(l, "behind", "건 지연", "متأخر"),
  gaps: (l: Locale) => t(l, "Gaps", "누락", "الفجوات"),
  gapLine: (l: Locale) => t(l, "Skipped events", "누락된 이벤트", "أحداث متخطّاة"),
  noGaps: (l: Locale) => t(l, "No gaps", "누락 없음", "لا فجوات"),
  resumeBy: (l: Locale, who: string) => t(l, `Resume requested by ${iso(who)}`, `재개 요청자 ${iso(who)}`, `طلب الاستئناف بواسطة ${iso(who)}`),
  deliveriesLink: (l: Locale) => t(l, "Deliveries for this subscription", "이 구독의 발송 건", "عمليات التسليم لهذا الاشتراك"),
  historyFold: (l: Locale) => t(l, "History and signing keys", "이력과 서명 키", "السجل ومفاتيح التوقيع"),
  historyTitle: (l: Locale) => t(l, "History", "이력", "السجل"),
  history: (l: Locale, s: string) => words(l, historyWords, s),
  colBy: (l: Locale) => t(l, "By", "수행자", "بواسطة"),
  noHistory: (l: Locale) => t(l, "No history", "이력 없음", "لا سجل"),
  keysTitle: (l: Locale) => t(l, "Signing keys", "서명 키", "مفاتيح التوقيع"),
  colKey: (l: Locale) => t(l, "Key", "키", "المفتاح"),
  keyState: (l: Locale, s: string) => words(l, keyStateWords, s),
  noKeys: (l: Locale) => t(l, "No active signing key", "활성 서명 키 없음", "لا يوجد مفتاح توقيع نشط"),
  signerNote: (l: Locale, signer: string) =>
    signer === "not_connected"
      ? t(l, "Signer: Not connected. Key proposals and rotation are not connected in this build.", "서명기: 연결되지 않음. 이 빌드에서는 키 제안과 교체가 연결되어 있지 않습니다.", "أداة التوقيع: غير متصلة. اقتراح المفاتيح وتدويرها غير متصلين في هذا الإصدار.")
      : t(l, `Signer: ${unknownCode("en", signer)}. Key proposals and rotation are not connected in this build.`, `서명기: ${unknownCode("ko", signer)}. 이 빌드에서는 키 제안과 교체가 연결되어 있지 않습니다.`, `أداة التوقيع: ${unknownCode("ar", signer)}. اقتراح المفاتيح وتدويرها غير متصلين في هذا الإصدار.`),
  subAction: (l: Locale, a: string) =>
    a === "approve" ? t(l, "Approve", "승인", "اعتماد")
      : a === "pause" ? t(l, "Pause", "일시 중지", "إيقاف مؤقت")
        : a === "request_resume" ? t(l, "Request resume", "재개 요청", "طلب الاستئناف")
          : a === "approve_resume" ? t(l, "Approve resume", "재개 승인", "اعتماد الاستئناف")
            : t(l, "Revoke", "회수", "إلغاء"),
  subConfirm: (l: Locale, a: string) =>
    a === "approve" ? t(l, "Confirm subscription approval", "구독 승인 확인", "تأكيد اعتماد الاشتراك")
      : a === "pause" ? t(l, "Confirm pause", "일시 중지 확인", "تأكيد الإيقاف المؤقت")
        : a === "request_resume" ? t(l, "Confirm resume request", "재개 요청 확인", "تأكيد طلب الاستئناف")
          : a === "approve_resume" ? t(l, "Confirm resume approval", "재개 승인 확인", "تأكيد اعتماد الاستئناف")
            : t(l, "Confirm revocation", "회수 확인", "تأكيد الإلغاء"),
  subImpact: (l: Locale, a: string) =>
    a === "approve" ? t(l, "Approval records the subscription. Nothing is sent: the dispatcher is not connected in this build.", "승인하면 구독이 기록됩니다. 이 빌드에서는 발송기가 연결되어 있지 않아 아무것도 보내지 않습니다.", "يسجّل الاعتماد الاشتراك. لا يُرسَل شيء، لأن المُرسِل غير متصل في هذا الإصدار.")
      : a === "pause" ? t(l, "Takes effect now; no second approval. Pausing records the subscription as paused; another administrator must approve a later resume.", "즉시 적용되며 두 번째 승인이 없습니다. 일시 중지하면 구독이 일시 중지됨으로 기록되며, 이후 재개는 다른 관리자가 승인해야 합니다.", "يسري فورًا؛ دون اعتماد ثانٍ. يسجّل الإيقاف المؤقت الاشتراك على أنه متوقف مؤقتًا؛ ويجب أن يعتمد مسؤول آخر أي استئناف لاحق.")
        : a === "request_resume" ? t(l, "Another administrator must approve the resume.", "다른 관리자가 재개를 승인해야 합니다.", "يجب أن يعتمد مسؤول آخر الاستئناف.")
          : a === "approve_resume" ? t(l, "The subscription becomes active again. Nothing is sent in this build.", "구독이 다시 활성화됩니다. 이 빌드에서는 아무것도 보내지 않습니다.", "يصبح الاشتراك نشطًا مجددًا. لا يُرسَل شيء في هذا الإصدار.")
            : t(
                l,
                "Takes effect now; no second approval. Revoking ends the subscription permanently and cancels its open delivery. A delivery that may already have reached the receiver is marked outcome unknown. Queued redeliveries of this subscription will not be sent. This cannot be undone.",
                "즉시 적용되며 두 번째 승인이 없습니다. 회수하면 구독이 영구히 끝나고 진행 중인 발송 건이 취소됩니다. 이미 수신 측에 도달했을 수 있는 발송 건은 결과 알 수 없음으로 표시됩니다. 이 구독의 대기 중인 재발송 건은 보내지 않습니다. 되돌릴 수 없습니다.",
                "يسري فورًا؛ دون اعتماد ثانٍ. يُنهي الإلغاء الاشتراك نهائيًا ويُلغي عملية التسليم المفتوحة. تُوسَم عملية التسليم التي ربما وصلت إلى المستلم بأن نتيجتها غير معروفة. لن تُرسَل إعادات التسليم المنتظرة لهذا الاشتراك. لا يمكن التراجع عن ذلك.",
              ),
  subVerb: (l: Locale, a: string, id: string) =>
    a === "approve" ? t(l, `Approve subscription ${iso(id)}`, `구독 ${iso(id)} 승인`, `اعتماد الاشتراك ${iso(id)}`)
      : a === "pause" ? t(l, `Pause subscription ${iso(id)}`, `구독 ${iso(id)} 일시 중지`, `إيقاف الاشتراك ${iso(id)} مؤقتًا`)
        : a === "request_resume" ? t(l, `Request resume of ${iso(id)}`, `${iso(id)} 재개 요청`, `طلب استئناف ${iso(id)}`)
          : a === "approve_resume" ? t(l, `Approve resume of ${iso(id)}`, `${iso(id)} 재개 승인`, `اعتماد استئناف ${iso(id)}`)
            : t(l, `Revoke subscription ${iso(id)}`, `구독 ${iso(id)} 회수`, `إلغاء الاشتراك ${iso(id)}`),
  subDone: (l: Locale, id: string, state: string) => t(l, `Subscription ${iso(id)} is ${state}. Nothing is sent.`, `구독 ${iso(id)} 상태: ${state}. 아무것도 보내지 않습니다.`, `الاشتراك ${iso(id)}: ${state}. لا يُرسَل شيء.`),
  proposeSubTitle: (l: Locale) => t(l, "Propose a subscription", "구독 제안", "اقتراح اشتراك"),
  subBlocker: (l: Locale, code: string) =>
    code === "no_types" ? t(l, "No event types are approved, so no subscription can be proposed.", "승인된 이벤트 유형이 없어 구독을 제안할 수 없습니다.", "لا توجد أنواع أحداث معتمدة، لذا لا يمكن اقتراح اشتراك.")
      : code === "no_destination" ? t(l, "No destination is active, so no subscription can be proposed.", "사용 가능한 목적지가 없어 구독을 제안할 수 없습니다.", "لا توجد وجهة نشطة، لذا لا يمكن اقتراح اشتراك.")
        : code === "unknown_list" ? t(l, "The list of subscriptions is unknown, so no subscription can be proposed.", "구독 목록을 알 수 없어 구독을 제안할 수 없습니다.", "قائمة الاشتراكات غير معروفة، لذا لا يمكن اقتراح اشتراك.")
        : t(l, "The subscriber is not known, so no subscription can be proposed.", "구독자를 알 수 없어 구독을 제안할 수 없습니다.", "المشترك غير معروف، لذا لا يمكن اقتراح اشتراك."),
  inactiveListed: (l: Locale) => t(l, "Inactive – cannot be used", "사용 불가 – 선택할 수 없음", "غير نشطة – لا يمكن استخدامها"),
  typesHint: (l: Locale, n: number) => t(l, `Choose 1 to 8. ${n} chosen.`, `1~8개를 선택하세요. ${n}개 선택됨.`, `اختر من 1 إلى 8. تم اختيار ${n}.`),
  chooseDestination: (l: Locale) => t(l, "Choose an active destination.", "사용 가능한 목적지를 선택하세요.", "اختر وجهة نشطة."),
  chooseTypes: (l: Locale) => t(l, "Choose at least one event type.", "이벤트 유형을 하나 이상 선택하세요.", "اختر نوع حدث واحدًا على الأقل."),
  tooManyTypes: (l: Locale) => t(l, "Choose no more than 8 event types.", "이벤트 유형은 8개까지 선택할 수 있습니다.", "لا تختر أكثر من 8 أنواع أحداث."),
  destinationChanged: (l: Locale) => t(l, "The destination list changed. Choose a destination again.", "목적지 목록이 바뀌었습니다. 목적지를 다시 선택하세요.", "تغيّرت قائمة الوجهات. اختر وجهة مجددًا."),
  typesChanged: (l: Locale) => t(l, "The approved event types changed. Choose the types again.", "승인된 이벤트 유형이 바뀌었습니다. 유형을 다시 선택하세요.", "تغيّرت أنواع الأحداث المعتمدة. اختر الأنواع مجددًا."),
  coveredOther: (l: Locale, id: string) =>
    t(l, `Subscription ${iso(id)} already covers this destination and these event types.`, `구독 ${iso(id)}이(가) 이미 이 목적지와 이벤트 유형을 다룹니다.`, `الاشتراك ${iso(id)} يغطي هذه الوجهة وأنواع الأحداث هذه بالفعل.`),
  coveredOwn: (l: Locale, id: string) => t(l, `You already proposed this as ${iso(id)}.`, `이미 ${iso(id)}(으)로 제안했습니다.`, `اقترحت هذا بالفعل بوصفه ${iso(id)}.`),
  confirmSub: (l: Locale) => t(l, "Confirm subscription proposal", "구독 제안 확인", "تأكيد اقتراح الاشتراك"),
  proposeSubImpact: (l: Locale) => t(l, "Another administrator must approve it. Nothing is sent in this build.", "다른 관리자가 승인해야 합니다. 이 빌드에서는 아무것도 보내지 않습니다.", "يجب أن يعتمده مسؤول آخر. لا يُرسَل شيء في هذا الإصدار."),
  proposeSubVerb: (l: Locale) => t(l, "Propose subscription", "구독 제안", "اقتراح الاشتراك"),
  proposedSubDone: (l: Locale, id: string) => t(l, `Subscription ${iso(id)} proposed. Another administrator must approve it.`, `구독 ${iso(id)}을(를) 제안했습니다. 다른 관리자가 승인해야 합니다.`, `اقتُرح الاشتراك ${iso(id)}. يجب أن يعتمده مسؤول آخر.`),
  duplicateSub: (l: Locale, id: string) => t(l, `This was already proposed as ${iso(id)}.`, `이미 ${iso(id)}(으)로 제안되었습니다.`, `اقتُرح هذا بالفعل بوصفه ${iso(id)}.`),

  eventsLead: (l: Locale) =>
    t(
      l,
      "The dispatcher is not connected. Queued and waiting deliveries stay as they are, and approving a redelivery only queues it. Nothing is sent from this build.",
      "발송기가 연결되어 있지 않습니다. 대기 중인 발송 건은 그대로 남아 있고, 재발송을 승인해도 대기열에 넣기만 합니다. 이 빌드에서는 아무것도 보내지 않습니다.",
      "المُرسِل غير متصل. تبقى عمليات التسليم المنتظرة كما هي، واعتماد إعادة التسليم يضعها في قائمة الانتظار فقط. لا يُرسَل أي شيء من هذا الإصدار.",
    ),
  oldestEvent: (l: Locale) => t(l, "Oldest kept event", "보관 중인 가장 오래된 이벤트", "أقدم حدث محفوظ"),
  newestEvent: (l: Locale) => t(l, "Newest event", "가장 최근 이벤트", "أحدث حدث"),
  dispatcher: (l: Locale) => t(l, "Dispatcher", "발송기", "المُرسِل"),
  typesTitle: (l: Locale) => t(l, "Event types", "이벤트 유형", "أنواع الأحداث"),
  colEventType: (l: Locale) => t(l, "Event type", "이벤트 유형", "نوع الحدث"),
  colSchema: (l: Locale) => t(l, "Schema version", "스키마 버전", "إصدار المخطط"),
  colFields: (l: Locale) => t(l, "Fields", "필드", "الحقول"),
  eventState: (l: Locale, s: string) => words(l, eventStateWords, s),
  typesNote: (l: Locale) => t(l, "Event publication and type proposals are not connected here.", "이벤트 게시와 유형 제안은 여기에 연결되어 있지 않습니다.", "نشر الأحداث واقتراح الأنواع غير متصلين هنا."),
  deliveriesTitle: (l: Locale) => t(l, "Deliveries", "발송 건", "عمليات التسليم"),
  deliveriesCut: (l: Locale) => t(l, "More deliveries exist than are listed here. Every dead letter and held redelivery is listed; those not shown are queued redeliveries or cancelled deliveries.", "여기에 표시된 것보다 발송 건이 더 있습니다. 실패 보관 건과 보류된 재발송 건은 모두 표시되며, 표시되지 않은 건은 대기열에 등록된 재발송 건이거나 취소된 건입니다.", "توجد عمليات تسليم أكثر مما هو معروض هنا. تُعرض كل الرسائل غير المُسلّمة وعمليات إعادة التسليم المحتجَزة؛ وما لم يُعرض هو عمليات إعادة تسليم في قائمة الانتظار أو عمليات تسليم ملغاة."),
  deliveriesMaybeCut: (l: Locale) => t(l, "The service did not say whether more deliveries exist than are listed here.", "서비스가 여기에 표시된 것보다 발송 건이 더 있는지 알려 주지 않았습니다.", "لم تذكر الخدمة ما إذا كانت هناك عمليات تسليم أكثر مما هو معروض هنا."),
  deliveriesNote: (l: Locale) => t(l, "Showing every dead letter and held redelivery, and up to 50 other deliveries per subscription.", "구독마다 실패 보관 건과 보류된 재발송 건은 모두, 그 밖의 발송 건은 최대 50건까지 표시합니다.", "تُعرض كل الرسائل غير المُسلّمة وعمليات إعادة التسليم المحتجَزة، وحتى 50 عملية تسليم أخرى لكل اشتراك."),
  filterLine: (l: Locale) => t(l, "Deliveries for subscription", "구독별 발송 건:", "عمليات التسليم للاشتراك"),
  showAll: (l: Locale) => t(l, "Show all deliveries", "모든 발송 건 보기", "عرض كل عمليات التسليم"),
  colDelivery: (l: Locale) => t(l, "Delivery", "발송 건", "التسليم"),
  colSubscription: (l: Locale) => t(l, "Subscription", "구독", "الاشتراك"),
  colSequence: (l: Locale) => t(l, "Sequence", "순번", "التسلسل"),
  colEvent: (l: Locale) => t(l, "Event", "이벤트", "الحدث"),
  colAttempts: (l: Locale) => t(l, "Attempts", "시도", "المحاولات"),
  deliveryKind: (l: Locale, s: string) => words(l, deliveryKindWords, s),
  job: (l: Locale, s: string) => words(l, jobWords, s),
  waitingResume: (l: Locale) => t(l, "Waiting for the subscription to resume", "구독 재개 대기", "بانتظار استئناف الاشتراك"),
  deliveryReason: (l: Locale, s: string) => (s ? words(l, deliveryReasonWords, s) : t(l, "None", "없음", "لا شيء")),
  outcomeUnknown: (l: Locale) =>
    t(l, "Outcome unknown: the receiver may have processed it before it was cancelled.", "결과 알 수 없음: 취소되기 전에 수신 측이 처리했을 수 있습니다.", "النتيجة غير معروفة: ربما عالجها المستلم قبل إلغائها."),
  outcomeMissing: (l: Locale) => t(l, "Outcome: Unknown", "결과: 알 수 없음", "النتيجة: غير معروفة"),
  revokedRow: (l: Locale) => t(l, "The subscription is revoked; this delivery will not be sent.", "구독이 회수되어 이 발송 건은 보내지 않습니다.", "الاشتراك مُلغى؛ لن تُرسَل عملية التسليم هذه."),
  noDeliveries: (l: Locale) =>
    t(l, "No open, queued, dead or cancelled deliveries. Delivered ones are not kept.", "진행 중·대기·실패 보관·취소된 발송 건이 없습니다. 전달된 건은 보관하지 않습니다.", "لا توجد عمليات تسليم مفتوحة أو منتظرة أو غير مُسلّمة أو ملغاة. لا تُحفظ المُسلَّمة."),
  noTypes: (l: Locale) => t(l, "No event types.", "이벤트 유형이 없습니다.", "لا توجد أنواع أحداث."),
  letterTitle: (l: Locale) => t(l, "Dead-letter decisions", "실패 보관 건 결정", "قرارات الرسائل غير المُسلّمة"),
  redeliverRequested: (l: Locale) => t(l, "Redelivery requested by", "재발송 요청자", "طلب إعادة التسليم بواسطة"),
  dismissRequested: (l: Locale) => t(l, "Dismissal requested by", "정리 요청자", "طلب الإزالة بواسطة"),
  requestUnknown: (l: Locale) => t(l, "A request state is unknown.", "요청 상태를 알 수 없습니다.", "حالة طلب غير معروفة."),
  noRequests: (l: Locale) => t(l, "No requests", "요청 없음", "لا طلبات"),
  letterAction: (l: Locale, a: string) => letterAction(l, a),
  letterConfirm: (l: Locale, a: string) => t(l, "Confirm: ", "확인: ", "تأكيد: ") + letterAction(l, a),
  letterImpact: (l: Locale, a: string) =>
    a === "approve_redeliver" ? t(l, "A new delivery is queued and this dead letter is removed. Nothing is sent: the dispatcher is not connected in this build.", "새 발송 건이 대기열에 들어가고 이 실패 보관 건은 삭제됩니다. 이 빌드에서는 발송기가 연결되어 있지 않아 아무것도 보내지 않습니다.", "تُضاف عملية تسليم جديدة إلى قائمة الانتظار وتُزال هذه الرسالة غير المُسلّمة. لا يُرسَل شيء، لأن المُرسِل غير متصل في هذا الإصدار.")
      : a === "approve_dismiss" ? t(l, "The dead letter is removed. Nothing is sent.", "실패 보관 건이 삭제됩니다. 아무것도 보내지 않습니다.", "تُزال الرسالة غير المُسلّمة. لا يُرسَل شيء.")
        : t(l, "Another administrator must approve it.", "다른 관리자가 승인해야 합니다.", "يجب أن يعتمده مسؤول آخر."),
  letterVerb: (l: Locale, a: string, id: string) =>
    a === "approve_redeliver" ? t(l, `Approve redelivery of ${iso(id)}`, `${iso(id)} 재발송 승인`, `اعتماد إعادة تسليم ${iso(id)}`)
      : a === "request_redeliver" ? t(l, `Request redelivery of ${iso(id)}`, `${iso(id)} 재발송 요청`, `طلب إعادة تسليم ${iso(id)}`)
        : a === "request_dismiss" ? t(l, `Request dismissal of ${iso(id)}`, `${iso(id)} 정리 요청`, `طلب إزالة ${iso(id)}`)
          : t(l, `Approve dismissal of ${iso(id)}`, `${iso(id)} 정리 승인`, `اعتماد إزالة ${iso(id)}`),
  letterDone: (l: Locale, a: string, redelivery: string) =>
    a === "approve_redeliver" ? t(l, `Redelivery queued as ${iso(redelivery)}. Nothing is sent.`, `재발송 건 ${iso(redelivery)}(으)로 대기열에 넣었습니다. 아무것도 보내지 않습니다.`, `أُضيفت إعادة التسليم إلى قائمة الانتظار بوصفها ${iso(redelivery)}. لا يُرسَل شيء.`)
      : a === "approve_dismiss" ? t(l, "Dead letter dismissed. Nothing is sent.", "실패 보관 건을 정리했습니다. 아무것도 보내지 않습니다.", "أُزيلت الرسالة غير المُسلّمة. لا يُرسَل شيء.")
        : a === "request_redeliver" ? t(l, "Redelivery requested. Another administrator must approve it.", "재발송을 요청했습니다. 다른 관리자가 승인해야 합니다.", "طُلبت إعادة التسليم. يجب أن يعتمدها مسؤول آخر.")
          : t(l, "Dismissal requested. Another administrator must approve it.", "정리를 요청했습니다. 다른 관리자가 승인해야 합니다.", "طُلبت الإزالة. يجب أن يعتمدها مسؤول آخر."),
  existingRequest: (l: Locale) => t(l, "Existing request", "기존 요청", "الطلب الحالي"),
  result: (l: Locale, s: string) => words(l, resultWords, s),

  blocker: (l: Locale, code: string, action = "") => {
    if (code === "missing") return t(l, "This is not available now.", "지금은 사용할 수 없습니다.", "هذا غير متاح الآن.");
    if (code === "revoked" && action === "approve_redeliver") return pick(l, letterRevoked);
    if (code === "not_pending" && has(notPendingWords, action)) return words(l, notPendingWords, action);
    if (has(blockerWords, code)) return words(l, blockerWords, code);
    return t(l, `This is not available now (${iso(code)}).`, `지금은 사용할 수 없습니다(${iso(code)}).`, `هذا غير متاح الآن (${iso(code)}).`);
  },
  blockerExtra: (l: Locale, code: string) => (dryRunBlocked.includes(code) ? pick(l, unverifiedSentence) : ""),
  kept: (l: Locale, g: Group) => {
    switch (g) {
      case "proposal":
        return t(l, "The plugin registry refused this decision. Nothing changed. Review the proposal, or ask the proposer to withdraw it.", "플러그인 레지스트리가 이 결정을 거부했습니다. 바뀐 것은 없습니다. 제안을 검토하거나 제안자에게 철회를 요청하세요.", "رفض سجل الإضافات هذا القرار. لم يتغيّر شيء. راجع المقترح، أو اطلب من صاحبه سحبه.");
      case "propose_plugin":
        return t(l, "The plugin registry refused this proposal. Nothing changed. Check the installed version and state; you may also have too many pending proposals.", "플러그인 레지스트리가 이 제안을 거부했습니다. 바뀐 것은 없습니다. 설치된 버전과 상태를 확인하세요. 승인 대기 중인 제안이 너무 많을 수도 있습니다.", "رفض سجل الإضافات هذا المقترح. لم يتغيّر شيء. تحقّق من الإصدار المثبّت وحالته؛ وقد يكون لديك عدد كبير من المقترحات المعلّقة.");
      case "plugin":
        return t(l, "The plugin registry refused this change. Nothing changed.", "플러그인 레지스트리가 이 변경을 거부했습니다. 바뀐 것은 없습니다.", "رفض سجل الإضافات هذا التغيير. لم يتغيّر شيء.");
      case "propose_subscription":
        return t(l, "The subscription was refused. Nothing changed. Another administrator may already have proposed the same destination and event types, or the subscription limit is reached.", "구독이 거부되었습니다. 바뀐 것은 없습니다. 다른 관리자가 같은 목적지와 이벤트 유형을 이미 제안했거나 구독 한도에 도달했을 수 있습니다.", "رُفض الاشتراك. لم يتغيّر شيء. ربما اقترح مسؤول آخر الوجهة وأنواع الأحداث نفسها، أو بُلغ حد الاشتراكات.");
      case "subscription":
        return t(l, "This subscription action was refused. Nothing changed. Reload and check the subscription's state.", "이 구독 작업이 거부되었습니다. 바뀐 것은 없습니다. 새로 고침한 뒤 구독 상태를 확인하세요.", "رُفض إجراء الاشتراك هذا. لم يتغيّر شيء. أعد التحميل وتحقّق من حالة الاشتراك.");
      default:
        return t(l, "This dead-letter action was refused. Nothing changed. It may already have been redelivered, dismissed or requested.", "이 실패 보관 건 작업이 거부되었습니다. 바뀐 것은 없습니다. 이미 재발송·정리되었거나 요청되었을 수 있습니다.", "رُفض إجراء الرسالة غير المُسلّمة هذا. لم يتغيّر شيء. ربما أُعيد تسليمها أو أُزيلت أو طُلب ذلك بالفعل.");
    }
  },
  keptLicense: (l: Locale, v: string, id: string) =>
    t(l, `Version ${iso(v)} has the license ${iso(id)}, which is not permitted on this installation.`, `${iso(v)} 버전의 라이선스 ${iso(id)}은(는) 이 설치에서 허용되지 않습니다.`, `للإصدار ${iso(v)} الترخيص ${iso(id)}، وهو غير مسموح به في هذا التثبيت.`),
  keptLicenseUnknown: (l: Locale, v: string) =>
    t(l, `The license of version ${iso(v)} is unknown, so it is treated as not permitted.`, `${iso(v)} 버전의 라이선스를 알 수 없어 허용되지 않은 것으로 처리합니다.`, `ترخيص الإصدار ${iso(v)} غير معروف، لذا يُعامل على أنه غير مسموح به.`),
  keptLicenseNext: (l: Locale, context: string) =>
    context === "approve"
      ? t(l, "This proposal cannot be approved; the proposer can withdraw it. The permitted list changes only when the server is restarted with a different list.", "이 제안은 승인할 수 없습니다. 제안자가 철회할 수 있습니다. 허용 목록은 다른 목록으로 서버를 다시 시작해야 바뀝니다.", "لا يمكن اعتماد هذا المقترح، ويمكن لصاحبه سحبه. لا تتغيّر القائمة المسموح بها إلا بإعادة تشغيل الخادم بقائمة مختلفة.")
      : context === "enable"
        ? t(l, "Propose an upgrade or rollback to a permitted version instead.", "대신 허용된 버전으로 업그레이드 또는 롤백을 제안하세요.", "اقترح بدلًا من ذلك ترقية أو إرجاعًا إلى إصدار مسموح به.")
        : t(l, "Choose another version; the permitted list changes only when the server is restarted with a different list.", "다른 버전을 선택하세요. 허용 목록은 다른 목록으로 서버를 다시 시작해야 바뀝니다.", "اختر إصدارًا آخر؛ لا تتغيّر القائمة المسموح بها إلا بإعادة تشغيل الخادم بقائمة مختلفة."),
  keptWithdraw: (l: Locale) => t(l, "The plugin registry refused this withdrawal. Nothing changed.", "플러그인 레지스트리가 이 철회를 거부했습니다. 바뀐 것은 없습니다.", "رفض سجل الإضافات هذا السحب. لم يتغيّر شيء."),
  gone: (l: Locale, g: Group) => {
    switch (g) {
      case "proposal":
        return t(l, "This proposal is no longer available.", "이 제안은 더 이상 사용할 수 없습니다.", "لم يعد هذا المقترح متاحًا.");
      case "plugin":
        return t(l, "This plugin is no longer available.", "이 플러그인은 더 이상 사용할 수 없습니다.", "لم تعد هذه الإضافة متاحة.");
      case "propose_plugin":
        return t(l, "Plugin changes can no longer be proposed here.", "더 이상 여기에서 플러그인 변경을 제안할 수 없습니다.", "لم يعد بالإمكان اقتراح تغييرات الإضافات هنا.");
      case "propose_subscription":
        return t(l, "Subscriptions can no longer be proposed here.", "더 이상 여기에서 구독을 제안할 수 없습니다.", "لم يعد بالإمكان اقتراح الاشتراكات هنا.");
      case "subscription":
        return t(l, "This subscription is no longer available.", "이 구독은 더 이상 사용할 수 없습니다.", "لم يعد هذا الاشتراك متاحًا.");
      default:
        return t(l, "This dead letter is no longer available.", "이 실패 보관 건은 더 이상 사용할 수 없습니다.", "لم تعد هذه الرسالة غير المُسلّمة متاحة.");
    }
  },
  pendingAttempt: (l: Locale) => t(l, "The last request may have been applied. Check the list, or send it again.", "직전 요청이 이미 반영되었을 수 있습니다. 목록을 확인하거나 다시 보내세요.", "ربما طُبّق الطلب الأخير. راجع القائمة أو أرسله مجددًا."),
  versionsChanged: (l: Locale) => t(l, "The plugin's versions or state changed. Review the current choices.", "플러그인의 버전이나 상태가 바뀌었습니다. 현재 선택지를 검토하세요.", "تغيّرت إصدارات الإضافة أو حالتها. راجع الخيارات الحالية."),
  kindBug: (l: Locale) => t(l, "The screen sent a change the service does not accept. Reload and try again.", "화면이 서비스에서 받지 않는 변경을 보냈습니다. 새로 고침한 뒤 다시 시도하세요.", "أرسلت الشاشة تغييرًا لا تقبله الخدمة. أعد التحميل وحاول مجددًا."),
  builtBug: (l: Locale) => t(l, "The request could not be sent as built. Reload and try again.", "요청을 구성된 그대로 보낼 수 없습니다. 새로 고침한 뒤 다시 시도하세요.", "تعذّر إرسال الطلب كما بُني. أعد التحميل وحاول مجددًا."),
  conflictAgain: (l: Locale) => t(l, "The integration data changed. Check the values and confirm again.", "연동 데이터가 바뀌었습니다. 값을 확인하고 다시 확인하세요.", "تغيّرت بيانات التكامل. تحقّق من القيم وأكّد مرة أخرى."),
  renew: (l: Locale) => t(l, "That request expired. Start again.", "요청이 만료되었습니다. 처음부터 다시 하세요.", "انتهت صلاحية الطلب. ابدأ من جديد."),
  partial: (l: Locale, op: string) => t(l, `Stopped after a change. Operation ${iso(op)}. Check the list before trying again.`, `변경 후 중단되었습니다. 작업 ${iso(op)}. 다시 시도하기 전에 목록을 확인하세요.`, `توقف بعد تغيير. العملية ${iso(op)}. راجع القائمة قبل المحاولة مجددًا.`),
  form: {
    rationale: (l: Locale) => t(l, "Reason", "사유", "السبب"),
    rationaleHint: (l: Locale, n: number) => t(l, `${n} of 256 bytes. Required. Do not enter customer data.`, `256바이트 중 ${n}바이트. 필수. 고객 정보를 입력하지 마세요.`, `${n} من 256 بايت. مطلوب. لا تُدخل بيانات العملاء.`),
    rationaleMissing: (l: Locale) => t(l, "Enter a reason.", "사유를 입력하세요.", "أدخل سببًا."),
    checkReason: (l: Locale) => t(l, "Check the reason and try again.", "사유를 확인하고 다시 시도하세요.", "تحقّق من السبب وحاول مجددًا."),
    rationaleLong: (l: Locale) => t(l, "The reason is too long. Use at most 256 bytes.", "사유가 너무 깁니다. 256바이트 이하로 입력하세요.", "السبب طويل جدًا. استخدم 256 بايت على الأكثر."),
    review: (l: Locale) => t(l, "Review", "검토", "مراجعة"),
    cancel: (l: Locale) => t(l, "Cancel", "취소", "إلغاء"),
    backToEdit: (l: Locale) => t(l, "Back to edit", "수정으로 돌아가기", "العودة إلى التعديل"),
    sending: (l: Locale) => t(l, "Sending…", "보내는 중…", "جارٍ الإرسال…"),
    startAgain: (l: Locale) => t(l, "Start again", "처음부터 다시", "البدء من جديد"),
  },
};

export function storageText(l: Locale, s: Storage): string {
  if (s.deletion === "retain_for_evidence") return s.retention_days === null ? k.keptUnknown(l) : k.keptDays(l, s.retention_days);
  if (s.deletion === "delete_on_removal") return k.deletion(l, s.deletion);
  return s.deletion ? k.unknownCode(l, s.deletion) : k.unknown(l);
}
