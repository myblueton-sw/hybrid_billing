import type { Locale } from "../../../../shared/types";
export type Words = readonly [string, string, string];
export const text = (locale: Locale, value: Words): string =>
  value[locale === "ko" ? 0 : locale === "en" ? 1 : 2];
export const words = {
  context: [
    "연결 등록 근거",
    "Connection registration evidence",
    "أدلة تسجيل الاتصالات",
  ],
  boundary: [
    "현재 허용 범위의 등록 메타데이터를 조회합니다. 외부 연결 인증·진단이나 청구 준비 완료를 뜻하지 않습니다.",
    "Inspect registered metadata within the current permitted scope. This does not establish external authentication, diagnostics or billing readiness.",
    "افحص بيانات التسجيل ضمن النطاق المسموح الحالي. لا يثبت ذلك المصادقة الخارجية أو التشخيص أو الجاهزية للفوترة.",
  ],
  unknown: ["미확인", "Unknown", "غير معروف"],
  scope: ["서버 읽기 범위", "Server read scope", "نطاق قراءة الخادم"],
  company: ["회사 범위", "Company scope", "نطاق الشركة"],
  tenant: ["테넌트 범위", "Tenant scope", "نطاق المستأجر"],
  members: ["허용 구성원 수", "Authorized members", "الأعضاء المسموحون"],
  observed: ["조회 시각", "Read at", "وقت القراءة"],
  pins: [
    "조회 범위·개정 근거",
    "Read scope and revision evidence",
    "أدلة نطاق القراءة والمراجعة",
  ],
  authScope: ["읽기 범위 해시", "Read scope digest", "بصمة نطاق القراءة"],
  permission: [
    "현재 권한 개정",
    "Current permission revision",
    "مراجعة الصلاحية الحالية",
  ],
  snapshot: [
    "등록 근거 스냅샷",
    "Registration evidence snapshot",
    "لقطة أدلة التسجيل",
  ],
  search: [
    "연결 ID 또는 근거 ID 접두사",
    "Connection or evidence ID prefix",
    "بادئة معرّف الاتصال أو الدليل",
  ],
  apply: ["필터 적용", "Apply filter", "تطبيق عامل التصفية"],
  clear: ["필터 지우기", "Clear filter", "مسح عامل التصفية"],
  invalid: [
    "소문자 영문·숫자·콜론·밑줄·하이픈을 128자 이내로 입력하세요.",
    "Use up to 128 lowercase letters, digits, colons, underscores or hyphens.",
    "استخدم حتى 128 حرفًا لاتينيًا صغيرًا أو رقمًا أو نقطتين أو شرطة سفلية أو واصلة.",
  ],
  refresh: ["새로고침", "Refresh", "تحديث"],
  retry: ["다시 조회", "Read again", "إعادة القراءة"],
  loading: [
    "현재 권한으로 등록 근거를 조회하고 있습니다.",
    "Reading registration evidence with current permissions.",
    "تُقرأ أدلة التسجيل بالصلاحيات الحالية.",
  ],
  error: [
    "등록 근거를 조회하지 못했습니다. 이전 결과는 표시하지 않습니다.",
    "Registration evidence could not be read. Previous results are hidden.",
    "تعذرت قراءة أدلة التسجيل. النتائج السابقة مخفية.",
  ],
  unconnected: [
    "등록 근거 조회 서비스가 연결되지 않았습니다. 건수와 상태는 미확인입니다.",
    "The registration evidence reader is not connected. Counts and states are unknown.",
    "قارئ أدلة التسجيل غير متصل. الأعداد والحالات غير معروفة.",
  ],
  empty: [
    "현재 허용 범위와 필터에서 등록된 연결이 없습니다. 기존 수집 접수로부터 연결 등록을 추정하지 않습니다.",
    "No registered connections match this permitted scope and filter. Existing collection deliveries do not imply a registration.",
    "لا توجد اتصالات مسجلة تطابق النطاق المسموح وعامل التصفية. تسليمات التجميع الحالية لا تعني وجود تسجيل.",
  ],
  list: ["등록된 연결", "Registered connections", "الاتصالات المسجلة"],
  capabilities: [
    "등록된 지원 범위",
    "Registered capability scope",
    "نطاق الإمكانات المسجل",
  ],
  capabilityLimit: [
    "표시된 버전·해석 프로필·기간은 등록 정보입니다. 공급사·상품별 지원이나 적합성 인증을 뜻하지 않습니다.",
    "The version, interpretation profile and interval are registration facts. They do not establish supplier/product support or conformance certification.",
    "الإصدار وملف التفسير والفترة هي بيانات تسجيل. لا تثبت دعم المورد أو المنتج أو شهادة المطابقة.",
  ],
  tableHint: [
    "화면 폭에 따라 핵심 열을 표시합니다. 연결을 선택하면 모든 등록 근거를 확인할 수 있습니다.",
    "The table prioritizes key columns for the available width. Select a connection to inspect all registration evidence.",
    "يعرض الجدول الأعمدة الأساسية بحسب المساحة المتاحة. حدد اتصالًا لفحص جميع أدلة التسجيل.",
  ],
  connection: ["연결 ID", "Connection ID", "معرّف الاتصال"],
  version: ["원천 FOCUS 버전", "Source FOCUS version", "إصدار FOCUS للمصدر"],
  profile: [
    "등록된 해석 프로필",
    "Registered interpretation profile",
    "ملف التفسير المسجل",
  ],
  consumed: [
    "FOCUS 소비 필드 해석",
    "FOCUS consumed-field interpretation",
    "تفسير حقول FOCUS المستخدمة",
  ],
  exploratory: [
    "탐색 조회 전용",
    "Exploratory query only",
    "للاستعلام الاستكشافي فقط",
  ],
  interval: ["등록 기간", "Registered interval", "الفترة المسجلة"],
  domains: ["등록 도메인 수", "Registered domains", "النطاقات المسجلة"],
  domainLimit: [
    "등록 도메인 수는 원천에서 발견하거나 수집한 계정 수가 아닙니다.",
    "Registered domains are not source-discovered or collected account counts.",
    "النطاقات المسجلة ليست عدد الحسابات المكتشفة أو المجمعة من المصدر.",
  ],
  from: ["시작 — 포함", "Start — inclusive", "البداية — مشمولة"],
  to: ["종료 — 제외", "End — exclusive", "النهاية — غير مشمولة"],
  detail: [
    "선택한 연결 등록 근거",
    "Selected connection registration evidence",
    "أدلة تسجيل الاتصال المحدد",
  ],
  close: ["상세 닫기", "Close details", "إغلاق التفاصيل"],
  back: ["목록으로 돌아가기", "Back to list", "العودة إلى القائمة"],
  policy: [
    "등록된 수집 정책",
    "Registered collection policy",
    "سياسة التجميع المسجلة",
  ],
  emptyFiltered: [
    "이 접두사와 일치하는 등록 연결이 없습니다.",
    "No registered connection or evidence ID matches this prefix.",
    "لا يوجد اتصال مسجل أو معرّف دليل يطابق هذه البادئة.",
  ],
  emptyScope: [
    "현재 허용 범위에 등록된 연결이 없습니다. 이 범위 밖의 연결은 포함하지 않습니다.",
    "No registered connections in the current permitted scope. Connections outside this scope are not counted.",
    "لا توجد اتصالات مسجلة ضمن النطاق المسموح الحالي. لا تُحتسب الاتصالات خارج هذا النطاق.",
  ],
  legacy: [
    "기존 수집 접수만으로 연결 등록을 추정하지 않습니다.",
    "Existing collection deliveries do not imply a connection registration.",
    "تسليمات التجميع الحالية لا تعني وجود تسجيل اتصال.",
  ],
  denied: [
    "이 범위의 연결 등록 근거를 조회할 권한이 없습니다.",
    "You do not have permission to read connection registrations in this scope.",
    "ليس لديك إذن لقراءة تسجيلات الاتصالات ضمن هذا النطاق.",
  ],
  invalidResponse: [
    "응답 검증에 실패해 근거를 표시하지 않았습니다.",
    "The response failed validation and was not displayed.",
    "فشل التحقق من الاستجابة ولم تُعرض الأدلة.",
  ],
  range: ["허용된 등록 연결", "Permitted registrations", "التسجيلات المسموحة"],
  page: ["페이지", "Page", "الصفحة"],
  registrationOnly: [
    "등록 근거만 제공",
    "Registration evidence only",
    "أدلة التسجيل فقط",
  ],
  fullyOwned: [
    "전체 등록 도메인이 현재 허용 범위에 속하는 연결",
    "Connections whose entire registered domain set is permitted",
    "الاتصالات التي تقع جميع نطاقاتها المسجلة ضمن النطاق المسموح",
  ],
  supplierSupport: [
    "공급사·상품별 지원",
    "Supplier and product support",
    "دعم المورد والمنتج",
  ],
  externalAuth: [
    "외부 공급사 인증",
    "External provider authentication",
    "المصادقة لدى المورد الخارجي",
  ],
  diagnosticEvidence: ["진단 근거", "Diagnostic evidence", "أدلة التشخيص"],
  notConnected: ["미연결", "Not connected", "غير متصل"],
  evidenceId: [
    "등록 근거 ID",
    "Registration evidence ID",
    "معرّف دليل التسجيل",
  ],
  grain: [
    "등록 수집 주기",
    "Registered collection grain",
    "دورية التجميع المسجلة",
  ],
  day: ["일별", "Daily", "يومية"],
  month: ["월별", "Monthly", "شهرية"],
  lateness: [
    "등록 지연 허용 시간 — 초",
    "Registered allowed lateness — seconds",
    "التأخر المسموح المسجل — بالثواني",
  ],
  replacements: [
    "등록 교체 횟수 한도",
    "Registered replacement limit",
    "حد الاستبدال المسجل",
  ],
  receipts: [
    "이 연결 ID 접두사로 수집 접수 조회",
    "Search collection deliveries by this connection ID prefix",
    "البحث في تسليمات التجميع ببادئة معرّف هذا الاتصال",
  ],
  destination: [
    "이동한 화면에서 현재 권한과 수집 근거를 다시 확인합니다. 같은 접두사의 더 긴 연결 ID도 검색될 수 있습니다.",
    "The destination rechecks current permissions and collection evidence. Prefix search can also match longer connection IDs.",
    "تعيد الشاشة المقصودة التحقق من الصلاحيات الحالية وأدلة التجميع. وقد يطابق البحث معرّفات اتصال أطول بالبادئة نفسها.",
  ],
  previous: ["이전 페이지", "Previous page", "الصفحة السابقة"],
  next: ["다음 페이지", "Next page", "الصفحة التالية"],
  security: [
    "외부 인증·보안 근거 미연결",
    "External authentication and security evidence not connected",
    "أدلة المصادقة والأمان الخارجية غير متصلة",
  ],
  securityBody: [
    "등록 메타데이터에는 인증 상태·자격 증명·만료·회수 결과가 없습니다. 보안 검증이 완료된 것으로 표시하지 않습니다.",
    "Registration metadata has no authentication state, credentials, expiry or revocation results. It does not establish completed security validation.",
    "لا تتضمن بيانات التسجيل حالة المصادقة أو بيانات الاعتماد أو انتهاء الصلاحية أو نتائج الإلغاء. ولا تثبت اكتمال التحقق الأمني.",
  ],
  diagnostics: [
    "진단 근거 미연결",
    "Diagnostic evidence not connected",
    "أدلة التشخيص غير متصلة",
  ],
  diagnosticsBody: [
    "외부 연결 검사와 진단 이력 생산자가 연결되지 않았습니다. 등록 정보로 성공·실패·실행 시각을 추정하지 않습니다.",
    "External connection tests and diagnostic-history producers are not connected. Registration does not imply success, failure or an execution time.",
    "اختبارات الاتصال الخارجي ومصدر سجل التشخيص غير متصلة. لا يعني التسجيل نجاحًا أو فشلًا أو وقت تنفيذ.",
  ],
  changed: [
    "근거나 권한이 변경됐습니다. 새로고침으로 현재 범위의 첫 페이지를 다시 조회하세요.",
    "Evidence or permissions changed. Refresh to read the first page in the current scope.",
    "تغيّرت الأدلة أو الصلاحيات. حدّث لقراءة الصفحة الأولى ضمن النطاق الحالي.",
  ],
  errorCode: ["오류 코드", "Error code", "رمز الخطأ"],
  requestId: ["요청 ID", "Request ID", "معرّف الطلب"],
} as const satisfies Record<string, Words>;
export function timestamp(value: string): string {
  return value.replace("T", " ").replace(/(?:Z|\+00:00)$/, " UTC");
}
