export const groups = [{"id":"overview","label":{"en":"Design guide","ko":"설계 안내","ar":"دليل التصميم"}},{"id":"access","label":{"en":"Identity & access","ko":"인증·접근","ar":"الهوية والوصول"}},{"id":"onboarding","label":{"en":"First setup","ko":"최초 설정","ar":"الإعداد الأولي"}},{"id":"work","label":{"en":"Billing operations","ko":"청구 운영","ar":"عمليات الفوترة"}},{"id":"sources","label":{"en":"Sources & evidence","ko":"원본·증빙","ar":"المصادر والأدلة"}},{"id":"contracts","label":{"en":"Contracts & pricing","ko":"계약·요율","ar":"العقود والتسعير"}},{"id":"reconcile","label":{"en":"Calculation & review","ko":"계산·검수","ar":"الحساب والمراجعة"}},{"id":"issue","label":{"en":"Documents & delivery","ko":"문서·전달","ar":"المستندات والتسليم"}},{"id":"history","label":{"en":"History & reports","ko":"이력·리포트","ar":"السجل والتقارير"}},{"id":"customer","label":{"en":"Customer portal","ko":"고객 포털","ar":"بوابة العملاء"}},{"id":"settings","label":{"en":"Administration","ko":"관리 설정","ar":"الإدارة"}}];

export const screens = [
  {
    "id": "overview", "group": "overview",
    "title": {"en":"Hybrid Billing — complete review","ko":"Hybrid Billing 전체 통합 시안","ar":"مراجعة تصميم Hybrid Billing"},
    "description": {"en":"Hybrid Billing — complete review. Choose a scoped workflow and inspect its prerequisites.","ko":"Hybrid Billing 전체 통합 시안. 업무 범위를 선택하고 시작 조건을 확인합니다.","ar":"مراجعة تصميم Hybrid Billing. اختر مسار عمل محدد النطاق وراجع متطلباته."},
    "kind": "overview",
    "columns": [{"key":"name","label":{"en":"Name","ko":"이름","ar":"الاسم"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Review workspace","ko":"업무 공간 검토","ar":"مراجعة مساحة العمل"},
    "related": ["contents"], "registryIds": [], "baselinePage": 1
  },
  {
    "id": "contents", "group": "overview",
    "title": {"en":"Complete screen index","ko":"전체 화면 목차","ar":"فهرس الشاشات"},
    "description": {"en":"Complete screen index. Choose a scoped workflow and inspect its prerequisites.","ko":"전체 화면 목차. 업무 범위를 선택하고 시작 조건을 확인합니다.","ar":"فهرس الشاشات. اختر مسار عمل محدد النطاق وراجع متطلباته."},
    "kind": "overview",
    "columns": [{"key":"name","label":{"en":"Name","ko":"이름","ar":"الاسم"},"type":"text"},{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"name","label":{"en":"Name","ko":"이름","ar":"الاسم"},"type":"text"}],
    "action": {"en":"Review workspace","ko":"업무 공간 검토","ar":"مراجعة مساحة العمل"},
    "related": ["operations","setup-progress","support-matrix"], "registryIds": [], "baselinePage": 2
  },
  {
    "id": "entry-start", "group": "access",
    "title": {"en":"Welcome to Hybrid Billing","ko":"Hybrid Billing 시작하기","ar":"بدء استخدام Hybrid Billing"},
    "description": {"en":"Welcome to Hybrid Billing. Choose a scoped workflow and inspect its prerequisites.","ko":"Hybrid Billing 시작하기. 업무 범위를 선택하고 시작 조건을 확인합니다.","ar":"بدء استخدام Hybrid Billing. اختر مسار عمل محدد النطاق وراجع متطلباته."},
    "kind": "overview",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"workspace","label":{"en":"Workspace","ko":"업무 공간","ar":"مساحة العمل"},"type":"text"}],
    "action": {"en":"Review workspace","ko":"업무 공간 검토","ar":"مراجعة مساحة العمل"},
    "related": ["entry-login","entry-invite","entry-bootstrap","entry-challenge"], "registryIds": ["L01"], "baselinePage": 3
  },
  {
    "id": "entry-bootstrap", "group": "access",
    "title": {"en":"Register the first administrator","ko":"최초 관리자 등록","ar":"تسجيل المسؤول الأول"},
    "description": {"en":"Register the first administrator. Review the required inputs and preview a draft in this simulation.","ko":"최초 관리자 등록. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"تسجيل المسؤول الأول. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"proof","label":{"en":"Verification reference","ko":"검증 참조","ar":"مرجع التحقق"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"proof","label":{"en":"Verification reference","ko":"검증 참조","ar":"مرجع التحقق"},"type":"text"},{"key":"email","label":{"en":"Email address","ko":"이메일 주소","ar":"البريد الإلكتروني"},"type":"email"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["entry-invite"], "registryIds": ["L02"], "baselinePage": 4
  },
  {
    "id": "entry-invite", "group": "access",
    "title": {"en":"Join by invitation","ko":"초대받아 가입하기","ar":"الانضمام عبر دعوة"},
    "description": {"en":"Join by invitation. Review the required inputs and preview a draft in this simulation.","ko":"초대받아 가입하기. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"الانضمام عبر دعوة. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"email","label":{"en":"Email address","ko":"이메일 주소","ar":"البريد الإلكتروني"},"type":"text"},{"key":"role","label":{"en":"Role","ko":"역할","ar":"الدور"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"expiry","label":{"en":"Expires at","ko":"만료일","ar":"تاريخ الانتهاء"},"type":"date"}],
    "fields": [{"key":"proof","label":{"en":"Verification reference","ko":"검증 참조","ar":"مرجع التحقق"},"type":"text"},{"key":"email","label":{"en":"Email address","ko":"이메일 주소","ar":"البريد الإلكتروني"},"type":"email"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["entry-login"], "registryIds": ["L02"], "baselinePage": 5
  },
  {
    "id": "entry-login", "group": "access",
    "title": {"en":"Sign in","ko":"로그인","ar":"تسجيل الدخول"},
    "description": {"en":"Sign in. Review the required inputs and preview a draft in this simulation.","ko":"로그인. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"تسجيل الدخول. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"identity","label":{"en":"Identity provider","ko":"인증 공급자","ar":"مزود الهوية"},"type":"text"},{"key":"workspace","label":{"en":"Workspace","ko":"업무 공간","ar":"مساحة العمل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"workspace","label":{"en":"Workspace","ko":"업무 공간","ar":"مساحة العمل"},"type":"text"},{"key":"email","label":{"en":"Email address","ko":"이메일 주소","ar":"البريد الإلكتروني"},"type":"email"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["entry-challenge"], "registryIds": ["L01"], "baselinePage": 6
  },
  {
    "id": "entry-challenge", "group": "access",
    "title": {"en":"Verify identity or recover access","ko":"추가 인증과 접근 복구","ar":"التحقق من الهوية واستعادة الوصول"},
    "description": {"en":"Verify identity or recover access. Review the required inputs and preview a draft in this simulation.","ko":"추가 인증과 접근 복구. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"التحقق من الهوية واستعادة الوصول. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"method","label":{"en":"Collection method","ko":"수집 방식","ar":"طريقة الجمع"},"type":"text"},{"key":"proof","label":{"en":"Verification reference","ko":"검증 참조","ar":"مرجع التحقق"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"proof","label":{"en":"Verification reference","ko":"검증 참조","ar":"مرجع التحقق"},"type":"text"},{"key":"email","label":{"en":"Email address","ko":"이메일 주소","ar":"البريد الإلكتروني"},"type":"email"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["entry-scope"], "registryIds": ["L03"], "baselinePage": 7
  },
  {
    "id": "entry-scope", "group": "access",
    "title": {"en":"Choose an authorized workspace","ko":"허용된 업무 공간 선택","ar":"اختيار مساحة عمل مصرح بها"},
    "description": {"en":"Choose an authorized workspace. Search and filter the scoped records; inspect a record before continuing.","ko":"허용된 업무 공간 선택. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"اختيار مساحة عمل مصرح بها. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"workspace","label":{"en":"Workspace","ko":"업무 공간","ar":"مساحة العمل"},"type":"text"},{"key":"entity","label":{"en":"Legal entity","ko":"법인","ar":"الكيان القانوني"},"type":"text"},{"key":"role","label":{"en":"Role","ko":"역할","ar":"الدور"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "fields": [{"key":"workspace","label":{"en":"Workspace","ko":"업무 공간","ar":"مساحة العمل"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["entry-roles"], "registryIds": ["L01"], "baselinePage": 8
  },
  {
    "id": "entry-roles", "group": "access",
    "title": {"en":"Roles and delegated permissions","ko":"역할별 권한과 초대 관리","ar":"الأدوار والصلاحيات المفوضة"},
    "description": {"en":"Roles and delegated permissions. Search and filter the scoped records; inspect a record before continuing.","ko":"역할별 권한과 초대 관리. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"الأدوار والصلاحيات المفوضة. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"role","label":{"en":"Role","ko":"역할","ar":"الدور"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"expiry","label":{"en":"Expires at","ko":"만료일","ar":"تاريخ الانتهاء"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"email","label":{"en":"Email address","ko":"이메일 주소","ar":"البريد الإلكتروني"},"type":"email"},{"key":"role","label":{"en":"Role","ko":"역할","ar":"الدور"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"expiry","label":{"en":"Expires at","ko":"만료일","ar":"تاريخ الانتهاء"},"type":"date"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["users","service-accounts","identity-policy"], "registryIds": ["M03"], "baselinePage": 9
  },
  {
    "id": "entry-sessions", "group": "access",
    "title": {"en":"Sessions and access recovery","ko":"세션·권한 회수·복구 관리","ar":"الجلسات واستعادة الوصول"},
    "description": {"en":"Sessions and access recovery. Search and filter the scoped records; inspect a record before continuing.","ko":"세션·권한 회수·복구 관리. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"الجلسات واستعادة الوصول. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"identity","label":{"en":"Identity provider","ko":"인증 공급자","ar":"مزود الهوية"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"created","label":{"en":"Created at","ko":"생성 시각","ar":"وقت الإنشاء"},"type":"date"},{"key":"expiry","label":{"en":"Expires at","ko":"만료일","ar":"تاريخ الانتهاء"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"identity","label":{"en":"Identity provider","ko":"인증 공급자","ar":"مزود الهوية"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["setup-progress"], "registryIds": ["M05"], "baselinePage": 10
  },
  {
    "id": "setup-progress", "group": "onboarding",
    "title": {"en":"Set up your first billing workspace","ko":"첫 빌링 업무 공간 설정","ar":"إعداد مساحة الفوترة الأولى"},
    "description": {"en":"Set up your first billing workspace. Review each prerequisite and unresolved dependency before the next step.","ko":"첫 빌링 업무 공간 설정. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"إعداد مساحة الفوترة الأولى. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"workspace","label":{"en":"Workspace","ko":"업무 공간","ar":"مساحة العمل"},"type":"text"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["setup-profile"], "registryIds": ["S02"], "baselinePage": 11
  },
  {
    "id": "setup-profile", "group": "onboarding",
    "title": {"en":"Business and billing foundation","ko":"운영·청구 기본 정보","ar":"بيانات التشغيل والفوترة"},
    "description": {"en":"Business and billing foundation. Review the required inputs and preview a draft in this simulation.","ko":"운영·청구 기본 정보. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"بيانات التشغيل والفوترة. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"entity","label":{"en":"Legal entity","ko":"법인","ar":"الكيان القانوني"},"type":"text"},{"key":"currency","label":{"en":"Currency","ko":"통화","ar":"العملة"},"type":"text"},{"key":"timezone","label":{"en":"Timezone","ko":"시간대","ar":"المنطقة الزمنية"},"type":"text"},{"key":"destination","label":{"en":"Destination","ko":"전달 대상","ar":"الوجهة"},"type":"text"}],
    "fields": [{"key":"mode","label":{"en":"Operating mode","ko":"운영 방식","ar":"وضع التشغيل"},"type":"text"},{"key":"entity","label":{"en":"Legal entity","ko":"법인","ar":"الكيان القانوني"},"type":"text"},{"key":"currency","label":{"en":"Currency","ko":"통화","ar":"العملة"},"type":"text"},{"key":"timezone","label":{"en":"Timezone","ko":"시간대","ar":"المنطقة الزمنية"},"type":"text"},{"key":"destination","label":{"en":"Destination","ko":"전달 대상","ar":"الوجهة"},"type":"text"},{"key":"tax","label":{"en":"Tax treatment","ko":"세금 처리","ar":"المعالجة الضريبية"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["setup-attribution"], "registryIds": ["O03"], "baselinePage": 12
  },
  {
    "id": "setup-attribution", "group": "onboarding",
    "title": {"en":"Connect customers to billing accounts","ko":"고객·조직·계정 귀속","ar":"ربط العملاء بحسابات الفوترة"},
    "description": {"en":"Connect customers to billing accounts. Search and filter the scoped records; inspect a record before continuing.","ko":"고객·조직·계정 귀속. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"ربط العملاء بحسابات الفوترة. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"provider"},{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"select","options":[{"value":"AWS","label":{"en":"AWS","ko":"AWS","ar":"AWS"}},{"value":"Microsoft Azure","label":{"en":"Microsoft Azure","ko":"Microsoft Azure","ar":"Microsoft Azure"}},{"value":"Google Cloud","label":{"en":"Google Cloud","ko":"Google Cloud","ar":"Google Cloud"}},{"value":"Oracle Cloud Infrastructure","label":{"en":"Oracle Cloud Infrastructure","ko":"Oracle Cloud Infrastructure","ar":"Oracle Cloud Infrastructure"}},{"value":"Alibaba Cloud","label":{"en":"Alibaba Cloud","ko":"Alibaba Cloud","ar":"Alibaba Cloud"}},{"value":"NAVER Cloud","label":{"en":"NAVER Cloud · support unverified","ko":"NAVER Cloud · 지원 미검증","ar":"NAVER Cloud · الدعم غير متحقق"}},{"value":"KT Cloud","label":{"en":"KT Cloud · support unverified","ko":"KT Cloud · 지원 미검증","ar":"KT Cloud · الدعم غير متحقق"}}]},{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["setup-connectors"], "registryIds": ["O01"], "baselinePage": 13
  },
  {
    "id": "setup-connectors", "group": "onboarding",
    "title": {"en":"Choose the first data connection","ko":"최초 데이터 연결 선택","ar":"اختيار اتصال البيانات الأول"},
    "description": {"en":"Choose the first data connection. Review the required inputs and preview a draft in this simulation.","ko":"최초 데이터 연결 선택. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"اختيار اتصال البيانات الأول. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"provider"},{"key":"method","label":{"en":"Collection method","ko":"수집 방식","ar":"طريقة الجمع"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"select","options":[{"value":"AWS","label":{"en":"AWS","ko":"AWS","ar":"AWS"}},{"value":"Microsoft Azure","label":{"en":"Microsoft Azure","ko":"Microsoft Azure","ar":"Microsoft Azure"}},{"value":"Google Cloud","label":{"en":"Google Cloud","ko":"Google Cloud","ar":"Google Cloud"}},{"value":"Oracle Cloud Infrastructure","label":{"en":"Oracle Cloud Infrastructure","ko":"Oracle Cloud Infrastructure","ar":"Oracle Cloud Infrastructure"}},{"value":"Alibaba Cloud","label":{"en":"Alibaba Cloud","ko":"Alibaba Cloud","ar":"Alibaba Cloud"}},{"value":"NAVER Cloud","label":{"en":"NAVER Cloud · support unverified","ko":"NAVER Cloud · 지원 미검증","ar":"NAVER Cloud · الدعم غير متحقق"}},{"value":"KT Cloud","label":{"en":"KT Cloud · support unverified","ko":"KT Cloud · 지원 미검증","ar":"KT Cloud · الدعم غير متحقق"}}]},{"key":"method","label":{"en":"Collection method","ko":"수집 방식","ar":"طريقة الجمع"},"type":"text"},{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["setup-readtest","support-matrix","supplier-naver","supplier-kt"], "registryIds": ["S01"], "baselinePage": 14
  },
  {
    "id": "setup-readtest", "group": "onboarding",
    "title": {"en":"Verify permission with an actual read","ko":"실제 읽기로 연결 권한 검증","ar":"التحقق من صلاحية القراءة"},
    "description": {"en":"Verify permission with an actual read. Review each prerequisite and unresolved dependency before the next step.","ko":"실제 읽기로 연결 권한 검증. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"التحقق من صلاحية القراءة. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"proof","label":{"en":"Verification reference","ko":"검증 참조","ar":"مرجع التحقق"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"proof","label":{"en":"Verification reference","ko":"검증 참조","ar":"مرجع التحقق"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["setup-schedule"], "registryIds": ["S01"], "baselinePage": 15
  },
  {
    "id": "setup-schedule", "group": "onboarding",
    "title": {"en":"Configure the first collection","ko":"최초 수집 범위·일정","ar":"إعداد عملية الجمع الأولى"},
    "description": {"en":"Configure the first collection. Review the required inputs and preview a draft in this simulation.","ko":"최초 수집 범위·일정. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"إعداد عملية الجمع الأولى. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"method","label":{"en":"Collection method","ko":"수집 방식","ar":"طريقة الجمع"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"frequency","label":{"en":"Collection schedule","ko":"수집 일정","ar":"جدول الجمع"},"type":"text"},{"key":"schema","label":{"en":"Schema version","ko":"스키마 버전","ar":"إصدار المخطط"},"type":"text"}],
    "fields": [{"key":"method","label":{"en":"Collection method","ko":"수집 방식","ar":"طريقة الجمع"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"frequency","label":{"en":"Collection schedule","ko":"수집 일정","ar":"جدول الجمع"},"type":"text"},{"key":"schema","label":{"en":"Schema version","ko":"스키마 버전","ar":"إصدار المخطط"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["setup-first"], "registryIds": ["S02"], "baselinePage": 16
  },
  {
    "id": "setup-first", "group": "onboarding",
    "title": {"en":"Validate the first collected data","ko":"최초 수집 데이터 검증","ar":"التحقق من البيانات المجمعة"},
    "description": {"en":"Validate the first collected data. Review each prerequisite and unresolved dependency before the next step.","ko":"최초 수집 데이터 검증. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"التحقق من البيانات المجمعة. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"rows","label":{"en":"Row count","ko":"행 수","ar":"عدد الصفوف"},"type":"number"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["setup-readiness"], "registryIds": ["S03"], "baselinePage": 17
  },
  {
    "id": "setup-readiness", "group": "onboarding",
    "title": {"en":"Ready for the first billing cycle?","ko":"첫 정산을 시작할 준비가 되었나요?","ar":"الجاهزية لدورة الفوترة الأولى"},
    "description": {"en":"Ready for the first billing cycle?. Review each prerequisite and unresolved dependency before the next step.","ko":"첫 정산을 시작할 준비가 되었나요?. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"الجاهزية لدورة الفوترة الأولى. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"workspace","label":{"en":"Workspace","ko":"업무 공간","ar":"مساحة العمل"},"type":"text"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["operations"], "registryIds": ["S02"], "baselinePage": 18
  },
  {
    "id": "operations", "group": "work",
    "title": {"en":"What is preventing issuance?","ko":"이번 달, 무엇이 발행을 막고 있나요?","ar":"العوائق أمام إصدار الفواتير"},
    "description": {"en":"What is preventing issuance?. Search and filter the scoped records; inspect a record before continuing.","ko":"이번 달, 무엇이 발행을 막고 있나요?. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"العوائق أمام إصدار الفواتير. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"due","label":{"en":"Due date","ko":"기한","ar":"تاريخ الاستحقاق"},"type":"date"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["core-index-close","my-tasks","index-intake","review-issue"], "registryIds": ["U01","U03","U04"], "baselinePage": 19
  },
  {
    "id": "core-index-close", "group": "work",
    "title": {"en":"Selected portfolio close workbench","ko":"선택 포트폴리오의 월 마감 작업","ar":"مساحة عمل الإقفال الشهري"},
    "description": {"en":"Selected portfolio close workbench. Search and filter the scoped records; inspect a record before continuing.","ko":"선택 포트폴리오의 월 마감 작업. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"مساحة عمل الإقفال الشهري. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"currency","label":{"en":"Currency","ko":"통화","ar":"العملة"},"type":"text"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"due","label":{"en":"Due date","ko":"기한","ar":"تاريخ الاستحقاق"},"type":"date"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["customer-detail"], "registryIds": ["U02"], "baselinePage": 20
  },
  {
    "id": "customer-detail", "group": "work",
    "title": {"en":"Customer account detail","ko":"고객 계정 상세","ar":"تفاصيل العميل"},
    "description": {"en":"Customer account detail. Inspect the selected scope, version and supporting evidence.","ko":"고객 계정 상세. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"تفاصيل العميل. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["index-intake"], "registryIds": ["O03"], "baselinePage": 21
  },
  {
    "id": "index-intake", "group": "sources",
    "title": {"en":"Resolve incomplete supplier evidence","ko":"공급자 원본 누락 해결","ar":"معالجة نقص أدلة المزود"},
    "description": {"en":"Resolve incomplete supplier evidence. Inspect file metadata, duplicate detection and evidence completeness.","ko":"공급자 원본 누락 해결. 파일 메타데이터·중복 여부·증빙 완전성을 확인합니다.","ar":"معالجة نقص أدلة المزود. راجع بيانات الملفات واكتشاف التكرار واكتمال الأدلة."},
    "kind": "files",
    "columns": [{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"currency","label":{"en":"Currency","ko":"통화","ar":"العملة"},"type":"text"}],
    "action": {"en":"Inspect file metadata","ko":"파일 정보 확인","ar":"فحص بيانات الملفات"},
    "related": ["v4-inbox"], "registryIds": ["S03"], "baselinePage": 22
  },
  {
    "id": "v4-inbox", "group": "sources",
    "title": {"en":"Supplier invoice inbox","ko":"공급자 인보이스 수신함","ar":"صندوق فواتير المزودين"},
    "description": {"en":"Supplier invoice inbox. Inspect file metadata, duplicate detection and evidence completeness.","ko":"공급자 인보이스 수신함. 파일 메타데이터·중복 여부·증빙 완전성을 확인합니다.","ar":"صندوق فواتير المزودين. راجع بيانات الملفات واكتشاف التكرار واكتمال الأدلة."},
    "kind": "files",
    "columns": [{"key":"file","label":{"en":"File name","ko":"파일명","ar":"اسم الملف"},"type":"text"},{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"provider"},{"key":"bytes","label":{"en":"File size (bytes)","ko":"파일 크기(bytes)","ar":"حجم الملف (بايت)"},"type":"number"},{"key":"fingerprint","label":{"en":"File fingerprint","ko":"파일 지문","ar":"بصمة الملف"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"select","options":[{"value":"AWS","label":{"en":"AWS","ko":"AWS","ar":"AWS"}},{"value":"Microsoft Azure","label":{"en":"Microsoft Azure","ko":"Microsoft Azure","ar":"Microsoft Azure"}},{"value":"Google Cloud","label":{"en":"Google Cloud","ko":"Google Cloud","ar":"Google Cloud"}},{"value":"Oracle Cloud Infrastructure","label":{"en":"Oracle Cloud Infrastructure","ko":"Oracle Cloud Infrastructure","ar":"Oracle Cloud Infrastructure"}},{"value":"Alibaba Cloud","label":{"en":"Alibaba Cloud","ko":"Alibaba Cloud","ar":"Alibaba Cloud"}},{"value":"NAVER Cloud","label":{"en":"NAVER Cloud · support unverified","ko":"NAVER Cloud · 지원 미검증","ar":"NAVER Cloud · الدعم غير متحقق"}},{"value":"KT Cloud","label":{"en":"KT Cloud · support unverified","ko":"KT Cloud · 지원 미검증","ar":"KT Cloud · الدعم غير متحقق"}}]},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Inspect file metadata","ko":"파일 정보 확인","ar":"فحص بيانات الملفات"},
    "related": ["v4-extract"], "registryIds": ["S05"], "baselinePage": 23
  },
  {
    "id": "v4-extract", "group": "sources",
    "title": {"en":"Review PDF, extracted fields and AI findings","ko":"PDF 원본·추출 항목·AI 분석 검토","ar":"مراجعة PDF والحقول المستخرجة"},
    "description": {"en":"Review PDF, extracted fields and AI findings. Inspect the selected scope, version and supporting evidence.","ko":"PDF 원본·추출 항목·AI 분석 검토. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"مراجعة PDF والحقول المستخرجة. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"page","label":{"en":"Page reference","ko":"페이지 참조","ar":"مرجع الصفحة"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"actual","label":{"en":"Observed amount","ko":"확인 금액","ar":"المبلغ المرصود"},"type":"number"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"page","label":{"en":"Page reference","ko":"페이지 참조","ar":"مرجع الصفحة"},"type":"text"},{"key":"actual","label":{"en":"Observed amount","ko":"확인 금액","ar":"المبلغ المرصود"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["v4-differences"], "registryIds": ["S05"], "baselinePage": 24
  },
  {
    "id": "v4-differences", "group": "sources",
    "title": {"en":"Resolve PDF versus collected-data differences","ko":"PDF와 수집 데이터의 차이 해결","ar":"معالجة الفروق بين PDF والبيانات"},
    "description": {"en":"Resolve PDF versus collected-data differences. Inspect the selected scope, version and supporting evidence.","ko":"PDF와 수집 데이터의 차이 해결. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"معالجة الفروق بين PDF والبيانات. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"expected","label":{"en":"Expected amount","ko":"기대 금액","ar":"المبلغ المتوقع"},"type":"number"},{"key":"actual","label":{"en":"Observed amount","ko":"확인 금액","ar":"المبلغ المرصود"},"type":"number"},{"key":"residual","label":{"en":"Unexplained difference","ko":"미해결 차액","ar":"الفرق غير المفسر"},"type":"number"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["v4-audit"], "registryIds": ["S06"], "baselinePage": 25
  },
  {
    "id": "v4-audit", "group": "sources",
    "title": {"en":"Accept evidence with an immutable audit trail","ko":"변경 이력을 보존한 원본 수용","ar":"قبول الأدلة مع سجل تدقيق ثابت"},
    "description": {"en":"Accept evidence with an immutable audit trail. Inspect the selected scope, version and supporting evidence.","ko":"변경 이력을 보존한 원본 수용. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"قبول الأدلة مع سجل تدقيق ثابت. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"created","label":{"en":"Created at","ko":"생성 시각","ar":"وقت الإنشاء"},"type":"date"},{"key":"actor","label":{"en":"Actual actor","ko":"실제 처리자","ar":"المنفذ الفعلي"},"type":"text"},{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"result","label":{"en":"Processing result","ko":"처리 결과","ar":"نتيجة المعالجة"},"type":"status"}],
    "fields": [{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["core-index-sources"], "registryIds": ["R02"], "baselinePage": 26
  },
  {
    "id": "core-index-sources", "group": "sources",
    "title": {"en":"Supplier invoice reconciliation","ko":"공급자 원본 대사","ar":"مطابقة فواتير المزودين"},
    "description": {"en":"Supplier invoice reconciliation. Search and filter the scoped records; inspect a record before continuing.","ko":"공급자 원본 대사. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"مطابقة فواتير المزودين. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"provider"},{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"residual","label":{"en":"Unexplained difference","ko":"미해결 차액","ar":"الفرق غير المفسر"},"type":"number"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"}],
    "fields": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"select","options":[{"value":"AWS","label":{"en":"AWS","ko":"AWS","ar":"AWS"}},{"value":"Microsoft Azure","label":{"en":"Microsoft Azure","ko":"Microsoft Azure","ar":"Microsoft Azure"}},{"value":"Google Cloud","label":{"en":"Google Cloud","ko":"Google Cloud","ar":"Google Cloud"}},{"value":"Oracle Cloud Infrastructure","label":{"en":"Oracle Cloud Infrastructure","ko":"Oracle Cloud Infrastructure","ar":"Oracle Cloud Infrastructure"}},{"value":"Alibaba Cloud","label":{"en":"Alibaba Cloud","ko":"Alibaba Cloud","ar":"Alibaba Cloud"}},{"value":"NAVER Cloud","label":{"en":"NAVER Cloud · support unverified","ko":"NAVER Cloud · 지원 미검증","ar":"NAVER Cloud · الدعم غير متحقق"}},{"value":"KT Cloud","label":{"en":"KT Cloud · support unverified","ko":"KT Cloud · 지원 미검증","ar":"KT Cloud · الدعم غير متحقق"}}]},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["core-index-aws"], "registryIds": ["S06"], "baselinePage": 27
  },
  {
    "id": "core-index-aws", "group": "sources",
    "title": {"en":"AWS source detail","ko":"AWS 원본 상세","ar":"تفاصيل مصدر AWS"},
    "description": {"en":"AWS source detail. Inspect the selected scope, version and supporting evidence.","ko":"AWS 원본 상세. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"تفاصيل مصدر AWS. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"service","label":{"en":"Service","ko":"서비스","ar":"الخدمة"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["core-index-variants"], "registryIds": ["S04"], "baselinePage": 28
  },
  {
    "id": "core-index-variants", "group": "sources",
    "title": {"en":"Provider billing structures","ko":"공급자별 청구 구조","ar":"هياكل الفوترة لدى المزودين"},
    "description": {"en":"Provider billing structures. Inspect the selected scope, version and supporting evidence.","ko":"공급자별 청구 구조. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"هياكل الفوترة لدى المزودين. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"provider"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"}],
    "fields": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"select","options":[{"value":"AWS","label":{"en":"AWS","ko":"AWS","ar":"AWS"}},{"value":"Microsoft Azure","label":{"en":"Microsoft Azure","ko":"Microsoft Azure","ar":"Microsoft Azure"}},{"value":"Google Cloud","label":{"en":"Google Cloud","ko":"Google Cloud","ar":"Google Cloud"}},{"value":"Oracle Cloud Infrastructure","label":{"en":"Oracle Cloud Infrastructure","ko":"Oracle Cloud Infrastructure","ar":"Oracle Cloud Infrastructure"}},{"value":"Alibaba Cloud","label":{"en":"Alibaba Cloud","ko":"Alibaba Cloud","ar":"Alibaba Cloud"}},{"value":"NAVER Cloud","label":{"en":"NAVER Cloud · support unverified","ko":"NAVER Cloud · 지원 미검증","ar":"NAVER Cloud · الدعم غير متحقق"}},{"value":"KT Cloud","label":{"en":"KT Cloud · support unverified","ko":"KT Cloud · 지원 미검증","ar":"KT Cloud · الدعم غير متحقق"}}]},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["index-mapping"], "registryIds": ["S04"], "baselinePage": 29
  },
  {
    "id": "index-mapping", "group": "sources",
    "title": {"en":"Assign the account and inspect the impact","ko":"계정 귀속 지정과 영향 확인","ar":"تعيين الحساب ومراجعة الأثر"},
    "description": {"en":"Assign the account and inspect the impact. Review the required inputs and preview a draft in this simulation.","ko":"계정 귀속 지정과 영향 확인. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"تعيين الحساب ومراجعة الأثر. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"until","label":{"en":"Effective until (exclusive)","ko":"적용 종료일(미포함)","ar":"تاريخ نهاية السريان (غير مشمول)"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"until","label":{"en":"Effective until (exclusive)","ko":"적용 종료일(미포함)","ar":"تاريخ نهاية السريان (غير مشمول)"},"type":"date"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["cost-pivot"], "registryIds": ["O02"], "baselinePage": 30
  },
  {
    "id": "cost-pivot", "group": "sources",
    "title": {"en":"Cost pivot by project, service and SKU","ko":"프로젝트·서비스·SKU 비용 피벗","ar":"تحليل التكاليف حسب المشروع والخدمة"},
    "description": {"en":"Cost pivot by project, service and SKU. Compare records using the same scope, currency and calculation basis.","ko":"프로젝트·서비스·SKU 비용 피벗. 동일 범위·통화·계산 기준으로 항목을 비교합니다.","ar":"تحليل التكاليف حسب المشروع والخدمة. قارن السجلات باستخدام النطاق والعملة وأساس الحساب نفسه."},
    "kind": "report",
    "columns": [{"key":"service","label":{"en":"Service","ko":"서비스","ar":"الخدمة"},"type":"text"},{"key":"sku","label":{"en":"Product / SKU","ko":"상품 / SKU","ar":"المنتج / SKU"},"type":"text"},{"key":"quantity","label":{"en":"Quantity","ko":"수량","ar":"الكمية"},"type":"number"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"number"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"service","label":{"en":"Service","ko":"서비스","ar":"الخدمة"},"type":"text"},{"key":"sku","label":{"en":"Product / SKU","ko":"상품 / SKU","ar":"المنتج / SKU"},"type":"text"},{"key":"currency","label":{"en":"Currency","ko":"통화","ar":"العملة"},"type":"text"}],
    "action": {"en":"Preview report","ko":"리포트 미리보기","ar":"معاينة التقرير"},
    "related": ["raw-explorer"], "registryIds": ["R01"], "baselinePage": 31
  },
  {
    "id": "raw-explorer", "group": "sources",
    "title": {"en":"Raw cost lookup and export","ko":"로우데이터 조회·내보내기","ar":"البحث في بيانات التكلفة وتصديرها"},
    "description": {"en":"Raw cost lookup and export. Search and filter the scoped records; inspect a record before continuing.","ko":"로우데이터 조회·내보내기. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"البحث في بيانات التكلفة وتصديرها. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"resource","label":{"en":"Resource reference","ko":"리소스 참조","ar":"مرجع المورد"},"type":"text"},{"key":"sku","label":{"en":"Product / SKU","ko":"상품 / SKU","ar":"المنتج / SKU"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"quantity","label":{"en":"Quantity","ko":"수량","ar":"الكمية"},"type":"number"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"}],
    "fields": [{"key":"resource","label":{"en":"Resource reference","ko":"리소스 참조","ar":"مرجع المورد"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["auto-portfolio"], "registryIds": ["S04"], "baselinePage": 32
  },
  {
    "id": "auto-portfolio", "group": "contracts",
    "title": {"en":"Contract pricing operations","ko":"계약 요율 운영 현황","ar":"عمليات تسعير العقود"},
    "description": {"en":"Contract pricing operations. Search and filter the scoped records; inspect a record before continuing.","ko":"계약 요율 운영 현황. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"عمليات تسعير العقود. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"}],
    "fields": [{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["supplier-register"], "registryIds": ["B04"], "baselinePage": 33
  },
  {
    "id": "supplier-register", "group": "contracts",
    "title": {"en":"Contracts and rates","ko":"계약·요율 관리","ar":"العقود والأسعار"},
    "description": {"en":"Contracts and rates. Search and filter the scoped records; inspect a record before continuing.","ko":"계약·요율 관리. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"العقود والأسعار. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"provider"},{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"entity","label":{"en":"Legal entity","ko":"법인","ar":"الكيان القانوني"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"select","options":[{"value":"AWS","label":{"en":"AWS","ko":"AWS","ar":"AWS"}},{"value":"Microsoft Azure","label":{"en":"Microsoft Azure","ko":"Microsoft Azure","ar":"Microsoft Azure"}},{"value":"Google Cloud","label":{"en":"Google Cloud","ko":"Google Cloud","ar":"Google Cloud"}},{"value":"Oracle Cloud Infrastructure","label":{"en":"Oracle Cloud Infrastructure","ko":"Oracle Cloud Infrastructure","ar":"Oracle Cloud Infrastructure"}},{"value":"Alibaba Cloud","label":{"en":"Alibaba Cloud","ko":"Alibaba Cloud","ar":"Alibaba Cloud"}},{"value":"NAVER Cloud","label":{"en":"NAVER Cloud · support unverified","ko":"NAVER Cloud · 지원 미검증","ar":"NAVER Cloud · الدعم غير متحقق"}},{"value":"KT Cloud","label":{"en":"KT Cloud · support unverified","ko":"KT Cloud · 지원 미검증","ar":"KT Cloud · الدعم غير متحقق"}}]},{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["supplier-aws","supplier-azure","supplier-gcp","supplier-oci","supplier-alibaba","supplier-naver","supplier-kt"], "registryIds": ["O03"], "baselinePage": 34
  },
  {
    "id": "marketplace-register", "group": "contracts",
    "title": {"en":"Marketplace ISV costs and contracts","ko":"Marketplace ISV 비용·계약","ar":"تكاليف وعقود Marketplace"},
    "description": {"en":"Marketplace ISV costs and contracts. Search and filter the scoped records; inspect a record before continuing.","ko":"Marketplace ISV 비용·계약. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"تكاليف وعقود Marketplace. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"provider"},{"key":"sku","label":{"en":"Product / SKU","ko":"상품 / SKU","ar":"المنتج / SKU"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"select","options":[{"value":"AWS","label":{"en":"AWS","ko":"AWS","ar":"AWS"}},{"value":"Microsoft Azure","label":{"en":"Microsoft Azure","ko":"Microsoft Azure","ar":"Microsoft Azure"}},{"value":"Google Cloud","label":{"en":"Google Cloud","ko":"Google Cloud","ar":"Google Cloud"}},{"value":"Oracle Cloud Infrastructure","label":{"en":"Oracle Cloud Infrastructure","ko":"Oracle Cloud Infrastructure","ar":"Oracle Cloud Infrastructure"}},{"value":"Alibaba Cloud","label":{"en":"Alibaba Cloud","ko":"Alibaba Cloud","ar":"Alibaba Cloud"}},{"value":"NAVER Cloud","label":{"en":"NAVER Cloud · support unverified","ko":"NAVER Cloud · 지원 미검증","ar":"NAVER Cloud · الدعم غير متحقق"}},{"value":"KT Cloud","label":{"en":"KT Cloud · support unverified","ko":"KT Cloud · 지원 미검증","ar":"KT Cloud · الدعم غير متحقق"}}]},{"key":"sku","label":{"en":"Product / SKU","ko":"상품 / SKU","ar":"المنتج / SKU"},"type":"text"},{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["marketplace-commitment"], "registryIds": ["B07"], "baselinePage": 35
  },
  {
    "id": "marketplace-commitment", "group": "contracts",
    "title": {"en":"Commitment credit and ISV purchase planning","ko":"약정 충당 현황·ISV 구매 계획","ar":"رصيد الالتزام وتخطيط مشتريات ISV"},
    "description": {"en":"Commitment credit and ISV purchase planning. Compare records using the same scope, currency and calculation basis.","ko":"약정 충당 현황·ISV 구매 계획. 동일 범위·통화·계산 기준으로 항목을 비교합니다.","ar":"رصيد الالتزام وتخطيط مشتريات ISV. قارن السجلات باستخدام النطاق والعملة وأساس الحساب نفسه."},
    "kind": "report",
    "columns": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"eligible","label":{"en":"Eligible amount","ko":"적격 금액","ar":"المبلغ المؤهل"},"type":"number"},{"key":"confirmed","label":{"en":"Provider-confirmed credit","ko":"공급자 확정 충당액","ar":"الرصيد المؤكد من المزود"},"type":"number"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"select","options":[{"value":"AWS","label":{"en":"AWS","ko":"AWS","ar":"AWS"}},{"value":"Microsoft Azure","label":{"en":"Microsoft Azure","ko":"Microsoft Azure","ar":"Microsoft Azure"}},{"value":"Google Cloud","label":{"en":"Google Cloud","ko":"Google Cloud","ar":"Google Cloud"}},{"value":"Oracle Cloud Infrastructure","label":{"en":"Oracle Cloud Infrastructure","ko":"Oracle Cloud Infrastructure","ar":"Oracle Cloud Infrastructure"}},{"value":"Alibaba Cloud","label":{"en":"Alibaba Cloud","ko":"Alibaba Cloud","ar":"Alibaba Cloud"}},{"value":"NAVER Cloud","label":{"en":"NAVER Cloud · support unverified","ko":"NAVER Cloud · 지원 미검증","ar":"NAVER Cloud · الدعم غير متحقق"}},{"value":"KT Cloud","label":{"en":"KT Cloud · support unverified","ko":"KT Cloud · 지원 미검증","ar":"KT Cloud · الدعم غير متحقق"}}]},{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"due","label":{"en":"Due date","ko":"기한","ar":"تاريخ الاستحقاق"},"type":"date"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"text"}],
    "action": {"en":"Preview report","ko":"리포트 미리보기","ar":"معاينة التقرير"},
    "related": ["supplier-aws"], "registryIds": ["B06","B07"], "baselinePage": 36
  },
  {
    "id": "supplier-aws", "group": "contracts",
    "title": {"en":"AWS · Contract evidence","ko":"AWS · 계약 근거","ar":"أدلة عقد AWS"},
    "description": {"en":"AWS · Contract evidence. Inspect the selected scope, version and supporting evidence.","ko":"AWS · 계약 근거. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"أدلة عقد AWS. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["supplier-azure"], "registryIds": ["B04"], "baselinePage": 37
  },
  {
    "id": "supplier-azure", "group": "contracts",
    "title": {"en":"Azure · Contract evidence","ko":"Azure · 계약 근거","ar":"أدلة عقد Azure"},
    "description": {"en":"Azure · Contract evidence. Inspect the selected scope, version and supporting evidence.","ko":"Azure · 계약 근거. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"أدلة عقد Azure. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["supplier-gcp"], "registryIds": ["B04"], "baselinePage": 38
  },
  {
    "id": "supplier-gcp", "group": "contracts",
    "title": {"en":"GCP · Contract evidence","ko":"GCP · 계약 근거","ar":"أدلة عقد Google Cloud"},
    "description": {"en":"GCP · Contract evidence. Inspect the selected scope, version and supporting evidence.","ko":"GCP · 계약 근거. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"أدلة عقد Google Cloud. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"number"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["supplier-oci"], "registryIds": ["B04"], "baselinePage": 39
  },
  {
    "id": "supplier-oci", "group": "contracts",
    "title": {"en":"OCI purchase contract","ko":"OCI 매입 계약 상세","ar":"عقد شراء Oracle Cloud"},
    "description": {"en":"OCI purchase contract. Inspect the selected scope, version and supporting evidence.","ko":"OCI 매입 계약 상세. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"عقد شراء Oracle Cloud. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"sku","label":{"en":"Product / SKU","ko":"상품 / SKU","ar":"المنتج / SKU"},"type":"text"},{"key":"quantity","label":{"en":"Quantity","ko":"수량","ar":"الكمية"},"type":"number"},{"key":"rate","label":{"en":"Unit price","ko":"단가","ar":"سعر الوحدة"},"type":"number"},{"key":"actual","label":{"en":"Observed amount","ko":"확인 금액","ar":"المبلغ المرصود"},"type":"number"},{"key":"residual","label":{"en":"Unexplained difference","ko":"미해결 차액","ar":"الفرق غير المفسر"},"type":"number"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["supplier-alibaba"], "registryIds": ["B04"], "baselinePage": 40
  },
  {
    "id": "supplier-alibaba", "group": "contracts",
    "title": {"en":"Alibaba Cloud · Contract evidence","ko":"Alibaba Cloud · 계약 근거","ar":"أدلة عقد Alibaba Cloud"},
    "description": {"en":"Alibaba Cloud · Contract evidence. Inspect the selected scope, version and supporting evidence.","ko":"Alibaba Cloud · 계약 근거. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"أدلة عقد Alibaba Cloud. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"rate","label":{"en":"Unit price","ko":"단가","ar":"سعر الوحدة"},"type":"number"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["customer-contract"], "registryIds": ["B04"], "baselinePage": 41
  },
  {
    "id": "customer-contract", "group": "contracts",
    "title": {"en":"Contract lifecycle and effective versions","ko":"계약 유지·갱신과 적용 버전","ar":"دورة حياة العقد وإصداراته"},
    "description": {"en":"Contract lifecycle and effective versions. Inspect the selected scope, version and supporting evidence.","ko":"계약 유지·갱신과 적용 버전. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"دورة حياة العقد وإصداراته. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["auto-rulebook"], "registryIds": ["O03"], "baselinePage": 42
  },
  {
    "id": "auto-rulebook", "group": "contracts",
    "title": {"en":"Northstar customer price book","ko":"Northstar 고객 계약 요율표","ar":"قائمة أسعار العميل"},
    "description": {"en":"Northstar customer price book. Search and filter the scoped records; inspect a record before continuing.","ko":"Northstar 고객 계약 요율표. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"قائمة أسعار العميل. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"interaction","label":{"en":"Benefit interaction","ko":"혜택 중복 적용","ar":"تداخل المزايا"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["auto-editor"], "registryIds": ["B04"], "baselinePage": 43
  },
  {
    "id": "auto-editor", "group": "contracts",
    "title": {"en":"Define a pricing rule","ko":"계약 요율 규칙 설정","ar":"تعريف قاعدة تسعير"},
    "description": {"en":"Define a pricing rule. Review the required inputs and preview a draft in this simulation.","ko":"계약 요율 규칙 설정. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"تعريف قاعدة تسعير. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"rate","label":{"en":"Unit price","ko":"단가","ar":"سعر الوحدة"},"type":"number"},{"key":"interaction","label":{"en":"Benefit interaction","ko":"혜택 중복 적용","ar":"تداخل المزايا"},"type":"text"}],
    "fields": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"text"},{"key":"interaction","label":{"en":"Benefit interaction","ko":"혜택 중복 적용","ar":"تداخل المزايا"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["auto-exclusions"], "registryIds": ["B04"], "baselinePage": 44
  },
  {
    "id": "auto-exclusions", "group": "contracts",
    "title": {"en":"Discount interaction and exclusions","ko":"할인 적용·중복·제외","ar":"تداخل الخصومات والاستثناءات"},
    "description": {"en":"Discount interaction and exclusions. Inspect the selected scope, version and supporting evidence.","ko":"할인 적용·중복·제외. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"تداخل الخصومات والاستثناءات. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"number"},{"key":"interaction","label":{"en":"Benefit interaction","ko":"혜택 중복 적용","ar":"تداخل المزايا"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"interaction","label":{"en":"Benefit interaction","ko":"혜택 중복 적용","ar":"تداخل المزايا"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["auto-mapping"], "registryIds": ["B04"], "baselinePage": 45
  },
  {
    "id": "auto-mapping", "group": "contracts",
    "title": {"en":"Validate automatic matching","ko":"자동 매칭·요율표 검증","ar":"التحقق من المطابقة التلقائية"},
    "description": {"en":"Validate automatic matching. Search and filter the scoped records; inspect a record before continuing.","ko":"자동 매칭·요율표 검증. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"التحقق من المطابقة التلقائية. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"sku","label":{"en":"Product / SKU","ko":"상품 / SKU","ar":"المنتج / SKU"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"result","label":{"en":"Processing result","ko":"처리 결과","ar":"نتيجة المعالجة"},"type":"status"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["auto-bulk"], "registryIds": ["B04"], "baselinePage": 46
  },
  {
    "id": "auto-bulk", "group": "contracts",
    "title": {"en":"Bulk price application preview","ko":"계약 요율 일괄 적용 미리보기","ar":"معاينة تطبيق الأسعار بالجملة"},
    "description": {"en":"Bulk price application preview. Review each prerequisite and unresolved dependency before the next step.","ko":"계약 요율 일괄 적용 미리보기. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"معاينة تطبيق الأسعار بالجملة. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"rows","label":{"en":"Row count","ko":"행 수","ar":"عدد الصفوف"},"type":"number"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["auto-exceptions"], "registryIds": ["B04"], "baselinePage": 47
  },
  {
    "id": "auto-exceptions", "group": "contracts",
    "title": {"en":"Pricing exceptions","ko":"자동 계산 예외 처리","ar":"استثناءات التسعير"},
    "description": {"en":"Pricing exceptions. Search and filter the scoped records; inspect a record before continuing.","ko":"자동 계산 예외 처리. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"استثناءات التسعير. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["auto-activation"], "registryIds": ["U04"], "baselinePage": 48
  },
  {
    "id": "auto-activation", "group": "contracts",
    "title": {"en":"Approve and schedule contract pricing","ko":"계약 요율 승인·적용 예약","ar":"مراجعة وجدولة تسعير العقود"},
    "description": {"en":"Approve and schedule contract pricing. Review each prerequisite and unresolved dependency before the next step.","ko":"계약 요율 승인·적용 예약. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"مراجعة وجدولة تسعير العقود. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"approval","label":{"en":"Approval reference","ko":"승인 참조","ar":"مرجع الموافقة"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["sales-policy"], "registryIds": ["B09"], "baselinePage": 49
  },
  {
    "id": "sales-policy", "group": "contracts",
    "title": {"en":"A default, explicit exceptions, one-off benefits","ko":"기본 정책, 고객별 예외, 일회성 혜택","ar":"سياسات التسعير والاستثناءات"},
    "description": {"en":"A default, explicit exceptions, one-off benefits. Search and filter the scoped records; inspect a record before continuing.","ko":"기본 정책, 고객별 예외, 일회성 혜택. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"سياسات التسعير والاستثناءات. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"until","label":{"en":"Effective until (exclusive)","ko":"적용 종료일(미포함)","ar":"تاريخ نهاية السريان (غير مشمول)"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"until","label":{"en":"Effective until (exclusive)","ko":"적용 종료일(미포함)","ar":"تاريخ نهاية السريان (غير مشمول)"},"type":"date"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["sales-impact"], "registryIds": ["B04"], "baselinePage": 50
  },
  {
    "id": "sales-impact", "group": "contracts",
    "title": {"en":"See the price before activating the rule","ko":"적용 전에 청구 금액과 영향을 확인","ar":"معاينة أثر الأسعار"},
    "description": {"en":"See the price before activating the rule. Compare records using the same scope, currency and calculation basis.","ko":"적용 전에 청구 금액과 영향을 확인. 동일 범위·통화·계산 기준으로 항목을 비교합니다.","ar":"معاينة أثر الأسعار. قارن السجلات باستخدام النطاق والعملة وأساس الحساب نفسه."},
    "kind": "report",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"}],
    "action": {"en":"Preview report","ko":"리포트 미리보기","ar":"معاينة التقرير"},
    "related": ["core-index-allocation"], "registryIds": ["B04"], "baselinePage": 51
  },
  {
    "id": "core-index-allocation", "group": "contracts",
    "title": {"en":"Cost allocation and retained costs","ko":"비용 배부·잔여 비용","ar":"توزيع التكاليف والتكاليف المتبقية"},
    "description": {"en":"Cost allocation and retained costs. Compare records using the same scope, currency and calculation basis.","ko":"비용 배부·잔여 비용. 동일 범위·통화·계산 기준으로 항목을 비교합니다.","ar":"توزيع التكاليف والتكاليف المتبقية. قارن السجلات باستخدام النطاق والعملة وأساس الحساب نفسه."},
    "kind": "report",
    "columns": [{"key":"pool","label":{"en":"Cost pool","ko":"원가 풀","ar":"مجموعة التكاليف"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"retained","label":{"en":"Retained cost","ko":"잔여 비용","ar":"التكلفة المتبقية"},"type":"number"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "fields": [{"key":"pool","label":{"en":"Cost pool","ko":"원가 풀","ar":"مجموعة التكاليف"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Preview report","ko":"리포트 미리보기","ar":"معاينة التقرير"},
    "related": ["core-index-pricing"], "registryIds": ["B03"], "baselinePage": 52
  },
  {
    "id": "core-index-pricing", "group": "contracts",
    "title": {"en":"Customer selling calculation","ko":"고객 판매액 산출 근거","ar":"أساس حساب سعر البيع"},
    "description": {"en":"Customer selling calculation. Inspect the selected scope, version and supporting evidence.","ko":"고객 판매액 산출 근거. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"أساس حساب سعر البيع. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"number"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"},{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"}],
    "fields": [{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["index-pricing"], "registryIds": ["B04"], "baselinePage": 53
  },
  {
    "id": "index-pricing", "group": "contracts",
    "title": {"en":"Change a scoped contract rule","ko":"계약 요율 수정과 충돌 확인","ar":"تعديل قاعدة عقد محددة النطاق"},
    "description": {"en":"Change a scoped contract rule. Review the required inputs and preview a draft in this simulation.","ko":"계약 요율 수정과 충돌 확인. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"تعديل قاعدة عقد محددة النطاق. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"rate","label":{"en":"Unit price","ko":"단가","ar":"سعر الوحدة"},"type":"number"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"until","label":{"en":"Effective until (exclusive)","ko":"적용 종료일(미포함)","ar":"تاريخ نهاية السريان (غير مشمول)"},"type":"date"}],
    "fields": [{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"until","label":{"en":"Effective until (exclusive)","ko":"적용 종료일(미포함)","ar":"تاريخ نهاية السريان (غير مشمول)"},"type":"date"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["index-seats"], "registryIds": ["B04"], "baselinePage": 54
  },
  {
    "id": "index-seats", "group": "contracts",
    "title": {"en":"Explain a mid-cycle seat adjustment","ko":"월중 좌석 변경의 일할 정산","ar":"تعديل المقاعد خلال الدورة"},
    "description": {"en":"Explain a mid-cycle seat adjustment. Inspect the selected scope, version and supporting evidence.","ko":"월중 좌석 변경의 일할 정산. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"تعديل المقاعد خلال الدورة. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"quantity","label":{"en":"Quantity","ko":"수량","ar":"الكمية"},"type":"number"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"until","label":{"en":"Effective until (exclusive)","ko":"적용 종료일(미포함)","ar":"تاريخ نهاية السريان (غير مشمول)"},"type":"date"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["index-commitments"], "registryIds": ["B04"], "baselinePage": 55
  },
  {
    "id": "index-commitments", "group": "contracts",
    "title": {"en":"Keep commitments and credits distinct","ko":"약정 비용과 크레딧의 구분","ar":"تكاليف الالتزام والأرصدة"},
    "description": {"en":"Keep commitments and credits distinct. Inspect the selected scope, version and supporting evidence.","ko":"약정 비용과 크레딧의 구분. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"تكاليف الالتزام والأرصدة. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["index-fx"], "registryIds": ["B06"], "baselinePage": 56
  },
  {
    "id": "index-fx", "group": "contracts",
    "title": {"en":"Verify conversion, tax and document grouping","ko":"환율·세금·청구서 분리 검증","ar":"التحويل والضرائب وتجميع المستندات"},
    "description": {"en":"Verify conversion, tax and document grouping. Review the required inputs and preview a draft in this simulation.","ko":"환율·세금·청구서 분리 검증. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"التحويل والضرائب وتجميع المستندات. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"currency","label":{"en":"Currency","ko":"통화","ar":"العملة"},"type":"text"},{"key":"fx","label":{"en":"Exchange-rate reference","ko":"환율 참조","ar":"مرجع سعر الصرف"},"type":"text"},{"key":"tax","label":{"en":"Tax treatment","ko":"세금 처리","ar":"المعالجة الضريبية"},"type":"text"},{"key":"rounding","label":{"en":"Rounding policy","ko":"반올림 정책","ar":"سياسة التقريب"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"}],
    "fields": [{"key":"fx","label":{"en":"Exchange-rate reference","ko":"환율 참조","ar":"مرجع سعر الصرف"},"type":"text"},{"key":"tax","label":{"en":"Tax treatment","ko":"세금 처리","ar":"المعالجة الضريبية"},"type":"text"},{"key":"rounding","label":{"en":"Rounding policy","ko":"반올림 정책","ar":"سياسة التقريب"},"type":"text"},{"key":"currency","label":{"en":"Currency","ko":"통화","ar":"العملة"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["auto-job"], "registryIds": ["B05"], "baselinePage": 57
  },
  {
    "id": "auto-job", "group": "reconcile",
    "title": {"en":"Automatic calculation and review","ko":"자동 계산·검수 상태","ar":"الحساب التلقائي والمراجعة"},
    "description": {"en":"Automatic calculation and review. Review each prerequisite and unresolved dependency before the next step.","ko":"자동 계산·검수 상태. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"الحساب التلقائي والمراجعة. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"run","label":{"en":"Calculation run","ko":"계산 실행","ar":"تشغيل الحساب"},"type":"text"},{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"run","label":{"en":"Calculation run","ko":"계산 실행","ar":"تشغيل الحساب"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["auto-trace"], "registryIds": ["B08"], "baselinePage": 58
  },
  {
    "id": "auto-trace", "group": "reconcile",
    "title": {"en":"Explain every calculated price","ko":"자동 계산 요율·금액 근거","ar":"أدلة الأسعار المحسوبة"},
    "description": {"en":"Explain every calculated price. Inspect the selected scope, version and supporting evidence.","ko":"자동 계산 요율·금액 근거. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"أدلة الأسعار المحسوبة. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"number"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"}],
    "fields": [{"key":"run","label":{"en":"Calculation run","ko":"계산 실행","ar":"تشغيل الحساب"},"type":"text"},{"key":"resource","label":{"en":"Resource reference","ko":"리소스 참조","ar":"مرجع المورد"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["ai-cost-waterfall"], "registryIds": ["B10"], "baselinePage": 59
  },
  {
    "id": "ai-cost-waterfall", "group": "reconcile",
    "title": {"en":"AI instance cost and stacked discounts","ko":"AI 인스턴스 원가·단계별 할인","ar":"تكلفة مثيلات الذكاء الاصطناعي والخصومات"},
    "description": {"en":"AI instance cost and stacked discounts. Compare records using the same scope, currency and calculation basis.","ko":"AI 인스턴스 원가·단계별 할인. 동일 범위·통화·계산 기준으로 항목을 비교합니다.","ar":"تكلفة مثيلات الذكاء الاصطناعي والخصومات. قارن السجلات باستخدام النطاق والعملة وأساس الحساب نفسه."},
    "kind": "report",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"number"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"resource","label":{"en":"Resource reference","ko":"리소스 참조","ar":"مرجع المورد"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Preview report","ko":"리포트 미리보기","ar":"معاينة التقرير"},
    "related": ["cost-differences"], "registryIds": ["S04"], "baselinePage": 60
  },
  {
    "id": "cost-differences", "group": "reconcile",
    "title": {"en":"Precision, AI version and support reconciliation","ko":"정밀도·AI 버전·Support 금액 대사","ar":"مطابقة الدقة والإصدارات والدعم"},
    "description": {"en":"Precision, AI version and support reconciliation. Search and filter the scoped records; inspect a record before continuing.","ko":"정밀도·AI 버전·Support 금액 대사. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"مطابقة الدقة والإصدارات والدعم. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"expected","label":{"en":"Expected amount","ko":"기대 금액","ar":"المبلغ المتوقع"},"type":"number"},{"key":"actual","label":{"en":"Observed amount","ko":"확인 금액","ar":"المبلغ المرصود"},"type":"number"},{"key":"residual","label":{"en":"Unexplained difference","ko":"미해결 차액","ar":"الفرق غير المفسر"},"type":"number"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["core-reconciliation-case"], "registryIds": ["S06"], "baselinePage": 61
  },
  {
    "id": "core-reconciliation-case", "group": "reconcile",
    "title": {"en":"Reconciliation investigation","ko":"청구 차이 원인 조사","ar":"التحقيق في فروق الفوترة"},
    "description": {"en":"Reconciliation investigation. Inspect the selected scope, version and supporting evidence.","ko":"청구 차이 원인 조사. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"التحقيق في فروق الفوترة. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"expected","label":{"en":"Expected amount","ko":"기대 금액","ar":"المبلغ المتوقع"},"type":"number"},{"key":"actual","label":{"en":"Observed amount","ko":"확인 금액","ar":"المبلغ المرصود"},"type":"number"},{"key":"residual","label":{"en":"Unexplained difference","ko":"미해결 차액","ar":"الفرق غير المفسر"},"type":"number"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"run","label":{"en":"Calculation run","ko":"계산 실행","ar":"تشغيل الحساب"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["core-reconciliation-recalc"], "registryIds": ["S06"], "baselinePage": 62
  },
  {
    "id": "core-reconciliation-recalc", "group": "reconcile",
    "title": {"en":"Calculation revision comparison","ko":"계산 버전 비교","ar":"مقارنة إصدارات الحساب"},
    "description": {"en":"Calculation revision comparison. Compare records using the same scope, currency and calculation basis.","ko":"계산 버전 비교. 동일 범위·통화·계산 기준으로 항목을 비교합니다.","ar":"مقارنة إصدارات الحساب. قارن السجلات باستخدام النطاق والعملة وأساس الحساب نفسه."},
    "kind": "report",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"expected","label":{"en":"Expected amount","ko":"기대 금액","ar":"المبلغ المتوقع"},"type":"number"},{"key":"actual","label":{"en":"Observed amount","ko":"확인 금액","ar":"المبلغ المرصود"},"type":"number"},{"key":"residual","label":{"en":"Unexplained difference","ko":"미해결 차액","ar":"الفرق غير المفسر"},"type":"number"}],
    "fields": [{"key":"run","label":{"en":"Calculation run","ko":"계산 실행","ar":"تشغيل الحساب"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"}],
    "action": {"en":"Preview report","ko":"리포트 미리보기","ar":"معاينة التقرير"},
    "related": ["index-runs"], "registryIds": ["B08"], "baselinePage": 63
  },
  {
    "id": "index-runs", "group": "reconcile",
    "title": {"en":"Run the corrected Aster candidate","ko":"수정한 Aster 후보 재처리","ar":"إعادة معالجة المسودة المصححة"},
    "description": {"en":"Run the corrected Aster candidate. Review each prerequisite and unresolved dependency before the next step.","ko":"수정한 Aster 후보 재처리. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"إعادة معالجة المسودة المصححة. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"run","label":{"en":"Calculation run","ko":"계산 실행","ar":"تشغيل الحساب"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"run","label":{"en":"Calculation run","ko":"계산 실행","ar":"تشغيل الحساب"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["core-reconciliation-approval"], "registryIds": ["B08"], "baselinePage": 64
  },
  {
    "id": "core-reconciliation-approval", "group": "reconcile",
    "title": {"en":"Financial approval review","ko":"금액 승인 검토","ar":"مراجعة الموافقة المالية"},
    "description": {"en":"Financial approval review. Review each prerequisite and unresolved dependency before the next step.","ko":"금액 승인 검토. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"مراجعة الموافقة المالية. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"approval","label":{"en":"Approval reference","ko":"승인 참조","ar":"مرجع الموافقة"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"run","label":{"en":"Calculation run","ko":"계산 실행","ar":"تشغيل الحساب"},"type":"text"},{"key":"reviewer","label":{"en":"Designated reviewer","ko":"지정 검수 담당자","ar":"المراجع المعين"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["index-approval"], "registryIds": ["B09"], "baselinePage": 65
  },
  {
    "id": "index-approval", "group": "reconcile",
    "title": {"en":"Resolve feedback before fixed-version issuance","ko":"반려 해결 후 고정 버전 승인·발행","ar":"معالجة الملاحظات قبل الإصدار"},
    "description": {"en":"Resolve feedback before fixed-version issuance. Review each prerequisite and unresolved dependency before the next step.","ko":"반려 해결 후 고정 버전 승인·발행. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"معالجة الملاحظات قبل الإصدار. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"approval","label":{"en":"Approval reference","ko":"승인 참조","ar":"مرجع الموافقة"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["assignment-policy"], "registryIds": ["B09"], "baselinePage": 66
  },
  {
    "id": "assignment-policy", "group": "issue",
    "title": {"en":"Issuer and reviewer assignment policy","ko":"발행·검수 담당자 지정 정책","ar":"سياسة تعيين المصدر والمراجع"},
    "description": {"en":"Issuer and reviewer assignment policy. Review the required inputs and preview a draft in this simulation.","ko":"발행·검수 담당자 지정 정책. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"سياسة تعيين المصدر والمراجع. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"issuer","label":{"en":"Designated issuer","ko":"지정 발행 담당자","ar":"مسؤول الإصدار المعين"},"type":"text"},{"key":"reviewer","label":{"en":"Designated reviewer","ko":"지정 검수 담당자","ar":"المراجع المعين"},"type":"text"},{"key":"actor","label":{"en":"Actual actor","ko":"실제 처리자","ar":"المنفذ الفعلي"},"type":"text"},{"key":"created","label":{"en":"Created at","ko":"생성 시각","ar":"وقت الإنشاء"},"type":"date"}],
    "fields": [{"key":"issuer","label":{"en":"Designated issuer","ko":"지정 발행 담당자","ar":"مسؤول الإصدار المعين"},"type":"text"},{"key":"reviewer","label":{"en":"Designated reviewer","ko":"지정 검수 담당자","ar":"المراجع المعين"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["review-issue"], "registryIds": ["I01"], "baselinePage": 67
  },
  {
    "id": "review-issue", "group": "issue",
    "title": {"en":"Review and issue","ko":"검수·발행 처리","ar":"المراجعة والإصدار"},
    "description": {"en":"Review and issue. Review each prerequisite and unresolved dependency before the next step.","ko":"검수·발행 처리. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"المراجعة والإصدار. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"issuer","label":{"en":"Designated issuer","ko":"지정 발행 담당자","ar":"مسؤول الإصدار المعين"},"type":"text"},{"key":"reviewer","label":{"en":"Designated reviewer","ko":"지정 검수 담당자","ar":"المراجع المعين"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"issuer","label":{"en":"Designated issuer","ko":"지정 발행 담당자","ar":"مسؤول الإصدار المعين"},"type":"text"},{"key":"reviewer","label":{"en":"Designated reviewer","ko":"지정 검수 담당자","ar":"المراجع المعين"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["document-history"], "registryIds": ["I01"], "baselinePage": 68
  },
  {
    "id": "document-history", "group": "issue",
    "title": {"en":"Document action history","ko":"문서 처리 이력","ar":"سجل إجراءات المستند"},
    "description": {"en":"Document action history. Inspect the selected scope, version and supporting evidence.","ko":"문서 처리 이력. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"سجل إجراءات المستند. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"actor","label":{"en":"Actual actor","ko":"실제 처리자","ar":"المنفذ الفعلي"},"type":"text"},{"key":"created","label":{"en":"Created at","ko":"생성 시각","ar":"وقت الإنشاء"},"type":"date"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["core-reconciliation-issued"], "registryIds": ["I01"], "baselinePage": 69
  },
  {
    "id": "core-reconciliation-issued", "group": "issue",
    "title": {"en":"Invoice and follow-up status","ko":"인보이스 발행·후속 상태","ar":"حالة الفاتورة والمتابعة"},
    "description": {"en":"Invoice and follow-up status. Inspect the selected scope, version and supporting evidence.","ko":"인보이스 발행·후속 상태. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"حالة الفاتورة والمتابعة. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"destination","label":{"en":"Destination","ko":"전달 대상","ar":"الوجهة"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"destination","label":{"en":"Destination","ko":"전달 대상","ar":"الوجهة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["erp"], "registryIds": ["I01"], "baselinePage": 70
  },
  {
    "id": "erp", "group": "issue",
    "title": {"en":"ERP connection and document mapping","ko":"ERP 연결·문서 매핑","ar":"اتصال ERP وربط المستندات"},
    "description": {"en":"ERP connection and document mapping. Review the required inputs and preview a draft in this simulation.","ko":"ERP 연결·문서 매핑. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"اتصال ERP وربط المستندات. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"destination","label":{"en":"Destination","ko":"전달 대상","ar":"الوجهة"},"type":"text"},{"key":"entity","label":{"en":"Legal entity","ko":"법인","ar":"الكيان القانوني"},"type":"text"},{"key":"submission","label":{"en":"Submission ID","ko":"제출 ID","ar":"معرف الإرسال"},"type":"text"},{"key":"external","label":{"en":"External document ID","ko":"외부 문서 ID","ar":"معرف المستند الخارجي"},"type":"text"},{"key":"result","label":{"en":"Processing result","ko":"처리 결과","ar":"نتيجة المعالجة"},"type":"status"}],
    "fields": [{"key":"destination","label":{"en":"Destination","ko":"전달 대상","ar":"الوجهة"},"type":"text"},{"key":"entity","label":{"en":"Legal entity","ko":"법인","ar":"الكيان القانوني"},"type":"text"},{"key":"method","label":{"en":"Collection method","ko":"수집 방식","ar":"طريقة الجمع"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["file-handoff","publication-packages"], "registryIds": ["I02"], "baselinePage": 71
  },
  {
    "id": "auto-history", "group": "history",
    "title": {"en":"Contract and calculation history","ko":"계약·자동 계산 변경 이력","ar":"سجل العقود والحسابات"},
    "description": {"en":"Contract and calculation history. Search and filter the scoped records; inspect a record before continuing.","ko":"계약·자동 계산 변경 이력. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"سجل العقود والحسابات. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"rule","label":{"en":"Pricing rule","ko":"요율 규칙","ar":"قاعدة التسعير"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"approval","label":{"en":"Approval reference","ko":"승인 참조","ar":"مرجع الموافقة"},"type":"text"},{"key":"created","label":{"en":"Created at","ko":"생성 시각","ar":"وقت الإنشاء"},"type":"date"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["v4-monthly"], "registryIds": ["B10","R02"], "baselinePage": 72
  },
  {
    "id": "v4-monthly", "group": "history",
    "title": {"en":"Monthly billing history","ko":"월별 청구·마감 히스토리","ar":"سجل الفوترة الشهري"},
    "description": {"en":"Monthly billing history. Search and filter the scoped records; inspect a record before continuing.","ko":"월별 청구·마감 히스토리. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"سجل الفوترة الشهري. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["v4-month"], "registryIds": ["R02"], "baselinePage": 73
  },
  {
    "id": "v4-month", "group": "history",
    "title": {"en":"September close and evidence history","ko":"9월 마감 상세와 근거 이력","ar":"تفاصيل الإقفال والأدلة"},
    "description": {"en":"September close and evidence history. Inspect the selected scope, version and supporting evidence.","ko":"9월 마감 상세와 근거 이력. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"تفاصيل الإقفال والأدلة. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["v4-variance"], "registryIds": ["R02"], "baselinePage": 74
  },
  {
    "id": "v4-variance", "group": "history",
    "title": {"en":"Explain the change from August to September","ko":"8월 대비 9월 증감 원인","ar":"تفسير التغير الشهري"},
    "description": {"en":"Explain the change from August to September. Compare records using the same scope, currency and calculation basis.","ko":"8월 대비 9월 증감 원인. 동일 범위·통화·계산 기준으로 항목을 비교합니다.","ar":"تفسير التغير الشهري. قارن السجلات باستخدام النطاق والعملة وأساس الحساب نفسه."},
    "kind": "report",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"}],
    "action": {"en":"Preview report","ko":"리포트 미리보기","ar":"معاينة التقرير"},
    "related": ["core-reconciliation-correction"], "registryIds": ["R01"], "baselinePage": 75
  },
  {
    "id": "core-reconciliation-correction", "group": "history",
    "title": {"en":"Post-issue source revision","ko":"발행 후 원본 개정 검토","ar":"مراجعة المصدر بعد الإصدار"},
    "description": {"en":"Post-issue source revision. Review each prerequisite and unresolved dependency before the next step.","ko":"발행 후 원본 개정 검토. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"مراجعة المصدر بعد الإصدار. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"},{"key":"approval","label":{"en":"Approval reference","ko":"승인 참조","ar":"مرجع الموافقة"},"type":"text"}],
    "fields": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["index-correction"], "registryIds": ["B11"], "baselinePage": 76
  },
  {
    "id": "index-correction", "group": "history",
    "title": {"en":"Close the late-credit correction case","ko":"지연 크레딧 정정 완료","ar":"معالجة تصحيح الرصيد المتأخر"},
    "description": {"en":"Close the late-credit correction case. Review each prerequisite and unresolved dependency before the next step.","ko":"지연 크레딧 정정 완료. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"معالجة تصحيح الرصيد المتأخر. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["auto-customer"], "registryIds": ["B11"], "baselinePage": 77
  },
  {
    "id": "auto-customer", "group": "customer",
    "title": {"en":"How your contract determines your bill","ko":"내 계약 요율이 청구 금액에 반영되는 방식","ar":"أثر العقد على فاتورتك"},
    "description": {"en":"How your contract determines your bill. Inspect the selected scope, version and supporting evidence.","ko":"내 계약 요율이 청구 금액에 반영되는 방식. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"أثر العقد على فاتورتك. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"number"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"sku","label":{"en":"Product / SKU","ko":"상품 / SKU","ar":"المنتج / SKU"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["ai-invoice-detail"], "registryIds": ["C02"], "baselinePage": 78
  },
  {
    "id": "ai-invoice-detail", "group": "customer",
    "title": {"en":"Your AI usage invoice and discount detail","ko":"내 AI 사용료 인보이스·할인 상세","ar":"تفاصيل فاتورة استخدام الذكاء الاصطناعي"},
    "description": {"en":"Your AI usage invoice and discount detail. Inspect the selected scope, version and supporting evidence.","ko":"내 AI 사용료 인보이스·할인 상세. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"تفاصيل فاتورة استخدام الذكاء الاصطناعي. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"number"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"}],
    "fields": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"resource","label":{"en":"Resource reference","ko":"리소스 참조","ar":"مرجع المورد"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["core-customer-overview"], "registryIds": ["C02"], "baselinePage": 79
  },
  {
    "id": "core-customer-overview", "group": "customer",
    "title": {"en":"Billing overview","ko":"청구 현황","ar":"نظرة عامة على الفوترة"},
    "description": {"en":"Billing overview. Choose a scoped workflow and inspect its prerequisites.","ko":"청구 현황. 업무 범위를 선택하고 시작 조건을 확인합니다.","ar":"نظرة عامة على الفوترة. اختر مسار عمل محدد النطاق وراجع متطلباته."},
    "kind": "overview",
    "columns": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review workspace","ko":"업무 공간 검토","ar":"مراجعة مساحة العمل"},
    "related": ["v4-customer-months"], "registryIds": ["C01"], "baselinePage": 80
  },
  {
    "id": "v4-customer-months", "group": "customer",
    "title": {"en":"Your monthly billing history","ko":"월별 청구 내역","ar":"سجل فواتيرك الشهري"},
    "description": {"en":"Your monthly billing history. Search and filter the scoped records; inspect a record before continuing.","ko":"월별 청구 내역. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"سجل فواتيرك الشهري. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"number"},{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["customer-history"], "registryIds": ["C01","C02"], "baselinePage": 81
  },
  {
    "id": "customer-history", "group": "customer",
    "title": {"en":"Your invoices and corrections","ko":"청구서와 정정 이력","ar":"فواتيرك وتصحيحاتها"},
    "description": {"en":"Your invoices and corrections. Search and filter the scoped records; inspect a record before continuing.","ko":"청구서와 정정 이력. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"فواتيرك وتصحيحاتها. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["core-customer-invoice"], "registryIds": ["C02"], "baselinePage": 82
  },
  {
    "id": "core-customer-invoice", "group": "customer",
    "title": {"en":"September invoice","ko":"9월 인보이스","ar":"تفاصيل الفاتورة"},
    "description": {"en":"September invoice. Inspect the selected scope, version and supporting evidence.","ko":"9월 인보이스. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"تفاصيل الفاتورة. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"service","label":{"en":"Service","ko":"서비스","ar":"الخدمة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"discount","label":{"en":"Discount","ko":"할인","ar":"الخصم"},"type":"number"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"}],
    "fields": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["core-customer-usage"], "registryIds": ["C02"], "baselinePage": 83
  },
  {
    "id": "core-customer-usage", "group": "customer",
    "title": {"en":"Service charges and usage detail","ko":"서비스 요금·사용 상세","ar":"رسوم الخدمات وتفاصيل الاستخدام"},
    "description": {"en":"Service charges and usage detail. Search and filter the scoped records; inspect a record before continuing.","ko":"서비스 요금·사용 상세. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"رسوم الخدمات وتفاصيل الاستخدام. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"service","label":{"en":"Service","ko":"서비스","ar":"الخدمة"},"type":"text"},{"key":"quantity","label":{"en":"Quantity","ko":"수량","ar":"الكمية"},"type":"number"},{"key":"unit","label":{"en":"Unit","ko":"단위","ar":"الوحدة"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"service","label":{"en":"Service","ko":"서비스","ar":"الخدمة"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["customer-inquiry"], "registryIds": ["C02"], "baselinePage": 84
  },
  {
    "id": "customer-inquiry", "group": "customer",
    "title": {"en":"Ask about a specific charge","ko":"청구 항목 문의","ar":"الاستفسار عن رسم محدد"},
    "description": {"en":"Ask about a specific charge. Review the required inputs and preview a draft in this simulation.","ko":"청구 항목 문의. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"الاستفسار عن رسم محدد. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"question","label":{"en":"Question","ko":"문의 내용","ar":"السؤال"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"question","label":{"en":"Question","ko":"문의 내용","ar":"السؤال"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["customer-response"], "registryIds": ["C03"], "baselinePage": 85
  },
  {
    "id": "customer-response", "group": "customer",
    "title": {"en":"Track the billing explanation","ko":"청구 문의 답변 확인","ar":"متابعة الرد على الاستفسار"},
    "description": {"en":"Track the billing explanation. Inspect the selected scope, version and supporting evidence.","ko":"청구 문의 답변 확인. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"متابعة الرد على الاستفسار. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"},{"key":"response","label":{"en":"Response","ko":"답변","ar":"الرد"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"invoice","label":{"en":"Invoice reference","ko":"청구서 참조","ar":"مرجع الفاتورة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["core-reconciliation-setup"], "registryIds": ["C03"], "baselinePage": 86
  },
  {
    "id": "core-reconciliation-setup", "group": "settings",
    "title": {"en":"Initial setup and source readiness","ko":"초기 설정·자료 준비","ar":"الإعداد الأولي وجاهزية المصادر"},
    "description": {"en":"Initial setup and source readiness. Review each prerequisite and unresolved dependency before the next step.","ko":"초기 설정·자료 준비. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"الإعداد الأولي وجاهزية المصادر. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"provider"},{"key":"source","label":{"en":"Source reference","ko":"원본 참조","ar":"مرجع المصدر"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"select","options":[{"value":"AWS","label":{"en":"AWS","ko":"AWS","ar":"AWS"}},{"value":"Microsoft Azure","label":{"en":"Microsoft Azure","ko":"Microsoft Azure","ar":"Microsoft Azure"}},{"value":"Google Cloud","label":{"en":"Google Cloud","ko":"Google Cloud","ar":"Google Cloud"}},{"value":"Oracle Cloud Infrastructure","label":{"en":"Oracle Cloud Infrastructure","ko":"Oracle Cloud Infrastructure","ar":"Oracle Cloud Infrastructure"}},{"value":"Alibaba Cloud","label":{"en":"Alibaba Cloud","ko":"Alibaba Cloud","ar":"Alibaba Cloud"}},{"value":"NAVER Cloud","label":{"en":"NAVER Cloud · support unverified","ko":"NAVER Cloud · 지원 미검증","ar":"NAVER Cloud · الدعم غير متحقق"}},{"value":"KT Cloud","label":{"en":"KT Cloud · support unverified","ko":"KT Cloud · 지원 미검증","ar":"KT Cloud · الدعم غير متحقق"}}]},{"key":"workspace","label":{"en":"Workspace","ko":"업무 공간","ar":"مساحة العمل"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["license-modes"], "registryIds": ["P01"], "baselinePage": 87
  },
  {
    "id": "license-modes", "group": "settings",
    "title": {"en":"Usage mode and license","ko":"이용 방식과 라이선스","ar":"وضع الاستخدام والترخيص"},
    "description": {"en":"Usage mode and license. Inspect the selected scope, version and supporting evidence.","ko":"이용 방식과 라이선스. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"وضع الاستخدام والترخيص. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"mode","label":{"en":"Operating mode","ko":"운영 방식","ar":"وضع التشغيل"},"type":"text"},{"key":"license","label":{"en":"License reference","ko":"라이선스 참조","ar":"مرجع الترخيص"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"mode","label":{"en":"Operating mode","ko":"운영 방식","ar":"وضع التشغيل"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["license-status"], "registryIds": ["P07"], "baselinePage": 88
  },
  {
    "id": "license-status", "group": "settings",
    "title": {"en":"License status and registration","ko":"라이선스 상태 및 키 등록","ar":"حالة الترخيص وتسجيله"},
    "description": {"en":"License status and registration. Review the required inputs and preview a draft in this simulation.","ko":"라이선스 상태 및 키 등록. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"حالة الترخيص وتسجيله. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"license","label":{"en":"License reference","ko":"라이선스 참조","ar":"مرجع الترخيص"},"type":"text"},{"key":"expiry","label":{"en":"Expires at","ko":"만료일","ar":"تاريخ الانتهاء"},"type":"date"},{"key":"result","label":{"en":"Processing result","ko":"처리 결과","ar":"نتيجة المعالجة"},"type":"status"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "fields": [{"key":"license","label":{"en":"License reference","ko":"라이선스 참조","ar":"مرجع الترخيص"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["license-modes","support-matrix","retention-policy","recovery-migration"], "registryIds": ["P07"], "baselinePage": 89
  },
  {
    "id": "organizations", "group": "work",
    "title": {"en":"Organization relationships","ko":"조직·관계 탐색","ar":"علاقات المؤسسات"},
    "description": {"en":"Organization relationships. Search and filter the scoped records; inspect a record before continuing.","ko":"조직·관계 탐색. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"علاقات المؤسسات. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"entity","label":{"en":"Legal entity","ko":"법인","ar":"الكيان القانوني"},"type":"text"},{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"relationship","label":{"en":"Relationship type","ko":"관계 유형","ar":"نوع العلاقة"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"entity","label":{"en":"Legal entity","ko":"법인","ar":"الكيان القانوني"},"type":"text"},{"key":"relationship","label":{"en":"Relationship type","ko":"관계 유형","ar":"نوع العلاقة"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["setup-attribution"], "registryIds": ["O01"], "baselinePage": null
  },
  {
    "id": "organization-impact", "group": "work",
    "title": {"en":"Relationship change impact","ko":"관계 변경 영향","ar":"أثر تغيير العلاقات"},
    "description": {"en":"Relationship change impact. Review each prerequisite and unresolved dependency before the next step.","ko":"관계 변경 영향. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"أثر تغيير العلاقات. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"entity","label":{"en":"Legal entity","ko":"법인","ar":"الكيان القانوني"},"type":"text"},{"key":"relationship","label":{"en":"Relationship type","ko":"관계 유형","ar":"نوع العلاقة"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"entity","label":{"en":"Legal entity","ko":"법인","ar":"الكيان القانوني"},"type":"text"},{"key":"relationship","label":{"en":"Relationship type","ko":"관계 유형","ar":"نوع العلاقة"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["index-mapping"], "registryIds": ["O02"], "baselinePage": null
  },
  {
    "id": "onprem-pools", "group": "contracts",
    "title": {"en":"On-premises cost pools","ko":"온프레미스 원가 풀","ar":"مجموعات تكاليف البنية المحلية"},
    "description": {"en":"On-premises cost pools. Search and filter the scoped records; inspect a record before continuing.","ko":"온프레미스 원가 풀. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"مجموعات تكاليف البنية المحلية. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"pool","label":{"en":"Cost pool","ko":"원가 풀","ar":"مجموعة التكاليف"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"}],
    "fields": [{"key":"pool","label":{"en":"Cost pool","ko":"원가 풀","ar":"مجموعة التكاليف"},"type":"text"},{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["core-index-allocation"], "registryIds": ["B01"], "baselinePage": null
  },
  {
    "id": "asset-measurements", "group": "sources",
    "title": {"en":"Asset and measurement history","ko":"자산·계측 이력","ar":"سجل الأصول والقياسات"},
    "description": {"en":"Asset and measurement history. Search and filter the scoped records; inspect a record before continuing.","ko":"자산·계측 이력. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"سجل الأصول والقياسات. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"asset","label":{"en":"Asset reference","ko":"자산 참조","ar":"مرجع الأصل"},"type":"text"},{"key":"measurement","label":{"en":"Measurement interval","ko":"계측 구간","ar":"فترة القياس"},"type":"text"},{"key":"quantity","label":{"en":"Quantity","ko":"수량","ar":"الكمية"},"type":"number"},{"key":"unit","label":{"en":"Unit","ko":"단위","ar":"الوحدة"},"type":"text"},{"key":"effective","label":{"en":"Effective from","ko":"적용 시작일","ar":"تاريخ بدء السريان"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"asset","label":{"en":"Asset reference","ko":"자산 참조","ar":"مرجع الأصل"},"type":"text"},{"key":"measurement","label":{"en":"Measurement interval","ko":"계측 구간","ar":"فترة القياس"},"type":"text"},{"key":"unit","label":{"en":"Unit","ko":"단위","ar":"الوحدة"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["onprem-pools"], "registryIds": ["B02"], "baselinePage": null
  },
  {
    "id": "file-handoff", "group": "issue",
    "title": {"en":"File handoff and results","ko":"파일 인계·반영 결과","ar":"تسليم الملفات والنتائج"},
    "description": {"en":"File handoff and results. Inspect file metadata, duplicate detection and evidence completeness.","ko":"파일 인계·반영 결과. 파일 메타데이터·중복 여부·증빙 완전성을 확인합니다.","ar":"تسليم الملفات والنتائج. راجع بيانات الملفات واكتشاف التكرار واكتمال الأدلة."},
    "kind": "files",
    "columns": [{"key":"file","label":{"en":"File name","ko":"파일명","ar":"اسم الملف"},"type":"text"},{"key":"bytes","label":{"en":"File size (bytes)","ko":"파일 크기(bytes)","ar":"حجم الملف (بايت)"},"type":"number"},{"key":"submission","label":{"en":"Submission ID","ko":"제출 ID","ar":"معرف الإرسال"},"type":"text"},{"key":"destination","label":{"en":"Destination","ko":"전달 대상","ar":"الوجهة"},"type":"text"},{"key":"result","label":{"en":"Processing result","ko":"처리 결과","ar":"نتيجة المعالجة"},"type":"status"},{"key":"external","label":{"en":"External document ID","ko":"외부 문서 ID","ar":"معرف المستند الخارجي"},"type":"text"}],
    "fields": [{"key":"submission","label":{"en":"Submission ID","ko":"제출 ID","ar":"معرف الإرسال"},"type":"text"},{"key":"destination","label":{"en":"Destination","ko":"전달 대상","ar":"الوجهة"},"type":"text"},{"key":"external","label":{"en":"External document ID","ko":"외부 문서 ID","ar":"معرف المستند الخارجي"},"type":"text"},{"key":"result","label":{"en":"Processing result","ko":"처리 결과","ar":"نتيجة المعالجة"},"type":"text"}],
    "action": {"en":"Inspect file metadata","ko":"파일 정보 확인","ar":"فحص بيانات الملفات"},
    "related": ["erp"], "registryIds": ["I03"], "baselinePage": null
  },
  {
    "id": "publication-packages", "group": "issue",
    "title": {"en":"Disclosure and delivery packages","ko":"공개 정책·전달 패키지","ar":"سياسة الإفصاح وحزم التسليم"},
    "description": {"en":"Disclosure and delivery packages. Review each prerequisite and unresolved dependency before the next step.","ko":"공개 정책·전달 패키지. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"سياسة الإفصاح وحزم التسليم. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"recipient","label":{"en":"Recipient scope","ko":"수신자 범위","ar":"نطاق المستلم"},"type":"text"},{"key":"fields","label":{"en":"Disclosed fields","ko":"공개 항목","ar":"الحقول المفصح عنها"},"type":"text"},{"key":"rows","label":{"en":"Row count","ko":"행 수","ar":"عدد الصفوف"},"type":"number"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"recipient","label":{"en":"Recipient scope","ko":"수신자 범위","ar":"نطاق المستلم"},"type":"text"},{"key":"fields","label":{"en":"Disclosed fields","ko":"공개 항목","ar":"الحقول المفصح عنها"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["file-handoff"], "registryIds": ["I04"], "baselinePage": null
  },
  {
    "id": "users", "group": "access",
    "title": {"en":"Users and invitations","ko":"사용자·초대·대량 회수","ar":"المستخدمون والدعوات"},
    "description": {"en":"Users and invitations. Search and filter the scoped records; inspect a record before continuing.","ko":"사용자·초대·대량 회수. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"المستخدمون والدعوات. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"email","label":{"en":"Email address","ko":"이메일 주소","ar":"البريد الإلكتروني"},"type":"text"},{"key":"identity","label":{"en":"Identity provider","ko":"인증 공급자","ar":"مزود الهوية"},"type":"text"},{"key":"role","label":{"en":"Role","ko":"역할","ar":"الدور"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"expiry","label":{"en":"Expires at","ko":"만료일","ar":"تاريخ الانتهاء"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"email","label":{"en":"Email address","ko":"이메일 주소","ar":"البريد الإلكتروني"},"type":"email"},{"key":"role","label":{"en":"Role","ko":"역할","ar":"الدور"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"expiry","label":{"en":"Expires at","ko":"만료일","ar":"تاريخ الانتهاء"},"type":"date"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["entry-roles"], "registryIds": ["M01"], "baselinePage": null
  },
  {
    "id": "identity-policy", "group": "access",
    "title": {"en":"Identity providers and sign-in policy","ko":"인증 공급자·로그인 정책","ar":"مزودو الهوية وسياسة تسجيل الدخول"},
    "description": {"en":"Identity providers and sign-in policy. Review the required inputs and preview a draft in this simulation.","ko":"인증 공급자·로그인 정책. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"مزودو الهوية وسياسة تسجيل الدخول. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"identity","label":{"en":"Identity provider","ko":"인증 공급자","ar":"مزود الهوية"},"type":"text"},{"key":"workspace","label":{"en":"Workspace","ko":"업무 공간","ar":"مساحة العمل"},"type":"text"},{"key":"proof","label":{"en":"Verification reference","ko":"검증 참조","ar":"مرجع التحقق"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"identity","label":{"en":"Identity provider","ko":"인증 공급자","ar":"مزود الهوية"},"type":"text"},{"key":"workspace","label":{"en":"Workspace","ko":"업무 공간","ar":"مساحة العمل"},"type":"text"},{"key":"proof","label":{"en":"Verification reference","ko":"검증 참조","ar":"مرجع التحقق"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["entry-challenge"], "registryIds": ["M02"], "baselinePage": null
  },
  {
    "id": "service-accounts", "group": "access",
    "title": {"en":"Service accounts and credential references","ko":"서비스 계정·자격 증명 참조","ar":"حسابات الخدمة ومراجع بيانات الاعتماد"},
    "description": {"en":"Service accounts and credential references. Search and filter the scoped records; inspect a record before continuing.","ko":"서비스 계정·자격 증명 참조. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"حسابات الخدمة ومراجع بيانات الاعتماد. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"name","label":{"en":"Name","ko":"이름","ar":"الاسم"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"expiry","label":{"en":"Expires at","ko":"만료일","ar":"تاريخ الانتهاء"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"name","label":{"en":"Name","ko":"이름","ar":"الاسم"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"expiry","label":{"en":"Expires at","ko":"만료일","ar":"تاريخ الانتهاء"},"type":"date"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["setup-readtest"], "registryIds": ["M04"], "baselinePage": null
  },
  {
    "id": "personal-security", "group": "access",
    "title": {"en":"Personal security and notifications","ko":"개인 보안·알림","ar":"الأمان الشخصي والإشعارات"},
    "description": {"en":"Personal security and notifications. Review the required inputs and preview a draft in this simulation.","ko":"개인 보안·알림. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"الأمان الشخصي والإشعارات. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"identity","label":{"en":"Identity provider","ko":"인증 공급자","ar":"مزود الهوية"},"type":"text"},{"key":"notification","label":{"en":"Notification channel","ko":"알림 채널","ar":"قناة الإشعارات"},"type":"text"},{"key":"created","label":{"en":"Created at","ko":"생성 시각","ar":"وقت الإنشاء"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"notification","label":{"en":"Notification channel","ko":"알림 채널","ar":"قناة الإشعارات"},"type":"text"},{"key":"identity","label":{"en":"Identity provider","ko":"인증 공급자","ar":"مزود الهوية"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["entry-sessions"], "registryIds": ["M05"], "baselinePage": null
  },
  {
    "id": "installation-diagnostics", "group": "settings",
    "title": {"en":"Installation diagnostics","ko":"설치·진단","ar":"تشخيص التثبيت"},
    "description": {"en":"Installation diagnostics. Review each prerequisite and unresolved dependency before the next step.","ko":"설치·진단. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"تشخيص التثبيت. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"compatibility","label":{"en":"Compatibility evidence","ko":"호환성 근거","ar":"دليل التوافق"},"type":"text"},{"key":"result","label":{"en":"Processing result","ko":"처리 결과","ar":"نتيجة المعالجة"},"type":"status"}],
    "fields": [{"key":"component","label":{"en":"Charge component","ko":"요금 구성 항목","ar":"مكون الرسوم"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["core-reconciliation-setup"], "registryIds": ["P01"], "baselinePage": null
  },
  {
    "id": "integrations", "group": "settings",
    "title": {"en":"API, plugins and channels","ko":"API·플러그인·채널","ar":"واجهات API والإضافات والقنوات"},
    "description": {"en":"API, plugins and channels. Search and filter the scoped records; inspect a record before continuing.","ko":"API·플러그인·채널. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"واجهات API والإضافات والقنوات. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"name","label":{"en":"Name","ko":"이름","ar":"الاسم"},"type":"text"},{"key":"endpoint","label":{"en":"Endpoint reference","ko":"엔드포인트 참조","ar":"مرجع نقطة الاتصال"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"name","label":{"en":"Name","ko":"이름","ar":"الاسم"},"type":"text"},{"key":"endpoint","label":{"en":"Endpoint reference","ko":"엔드포인트 참조","ar":"مرجع نقطة الاتصال"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["service-accounts"], "registryIds": ["P02"], "baselinePage": null
  },
  {
    "id": "recovery-migration", "group": "settings",
    "title": {"en":"Recovery and migration","ko":"복구·이관","ar":"الاستعادة والترحيل"},
    "description": {"en":"Recovery and migration. Review each prerequisite and unresolved dependency before the next step.","ko":"복구·이관. 단계별 선행 조건과 미해결 의존 항목을 확인합니다.","ar":"الاستعادة والترحيل. راجع المتطلبات والعناصر غير المحسومة قبل الخطوة التالية."},
    "kind": "workflow",
    "columns": [{"key":"backup","label":{"en":"Backup reference","ko":"백업 참조","ar":"مرجع النسخة الاحتياطية"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"created","label":{"en":"Created at","ko":"생성 시각","ar":"وقت الإنشاء"},"type":"date"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"result","label":{"en":"Processing result","ko":"처리 결과","ar":"نتيجة المعالجة"},"type":"status"}],
    "fields": [{"key":"backup","label":{"en":"Backup reference","ko":"백업 참조","ar":"مرجع النسخة الاحتياطية"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"}],
    "action": {"en":"Check prerequisites","ko":"선행 조건 확인","ar":"فحص المتطلبات"},
    "related": ["installation-diagnostics"], "registryIds": ["P03"], "baselinePage": null
  },
  {
    "id": "retention-policy", "group": "settings",
    "title": {"en":"Retention and deletion review","ko":"보존·삭제 검토","ar":"مراجعة الاحتفاظ والحذف"},
    "description": {"en":"Retention and deletion review. Review the required inputs and preview a draft in this simulation.","ko":"보존·삭제 검토. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"مراجعة الاحتفاظ والحذف. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"retention","label":{"en":"Retention period","ko":"보존 기간","ar":"مدة الاحتفاظ"},"type":"text"},{"key":"hold","label":{"en":"Legal hold","ko":"법적 보존","ar":"الحفظ القانوني"},"type":"text"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"retention","label":{"en":"Retention period","ko":"보존 기간","ar":"مدة الاحتفاظ"},"type":"text"},{"key":"hold","label":{"en":"Legal hold","ko":"법적 보존","ar":"الحفظ القانوني"},"type":"text"},{"key":"reason","label":{"en":"Reason","ko":"사유","ar":"السبب"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["audit-events"], "registryIds": ["P04"], "baselinePage": null
  },
  {
    "id": "support-matrix", "group": "settings",
    "title": {"en":"Support matrix and release evidence","ko":"지원 매트릭스·릴리스 근거","ar":"مصفوفة الدعم وأدلة الإصدارات"},
    "description": {"en":"Support matrix and release evidence. Search and filter the scoped records; inspect a record before continuing.","ko":"지원 매트릭스·릴리스 근거. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"مصفوفة الدعم وأدلة الإصدارات. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"provider"},{"key":"capability","label":{"en":"Capability","ko":"기능","ar":"الإمكانية"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"provider","label":{"en":"Cloud provider","ko":"클라우드 공급자","ar":"مزود الخدمات السحابية"},"type":"select","options":[{"value":"AWS","label":{"en":"AWS","ko":"AWS","ar":"AWS"}},{"value":"Microsoft Azure","label":{"en":"Microsoft Azure","ko":"Microsoft Azure","ar":"Microsoft Azure"}},{"value":"Google Cloud","label":{"en":"Google Cloud","ko":"Google Cloud","ar":"Google Cloud"}},{"value":"Oracle Cloud Infrastructure","label":{"en":"Oracle Cloud Infrastructure","ko":"Oracle Cloud Infrastructure","ar":"Oracle Cloud Infrastructure"}},{"value":"Alibaba Cloud","label":{"en":"Alibaba Cloud","ko":"Alibaba Cloud","ar":"Alibaba Cloud"}},{"value":"NAVER Cloud","label":{"en":"NAVER Cloud · support unverified","ko":"NAVER Cloud · 지원 미검증","ar":"NAVER Cloud · الدعم غير متحقق"}},{"value":"KT Cloud","label":{"en":"KT Cloud · support unverified","ko":"KT Cloud · 지원 미검증","ar":"KT Cloud · الدعم غير متحقق"}}]},{"key":"capability","label":{"en":"Capability","ko":"기능","ar":"الإمكانية"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["setup-connectors"], "registryIds": ["P05"], "baselinePage": null
  },
  {
    "id": "demo-scenarios", "group": "settings",
    "title": {"en":"Synthetic scenarios","ko":"합성 시나리오","ar":"سيناريوهات اصطناعية"},
    "description": {"en":"Synthetic scenarios. Search and filter the scoped records; inspect a record before continuing.","ko":"합성 시나리오. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"سيناريوهات اصطناعية. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"scenario","label":{"en":"Scenario","ko":"시나리오","ar":"السيناريو"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"expected","label":{"en":"Expected amount","ko":"기대 금액","ar":"المبلغ المتوقع"},"type":"number"},{"key":"result","label":{"en":"Processing result","ko":"처리 결과","ar":"نتيجة المعالجة"},"type":"status"}],
    "fields": [{"key":"scenario","label":{"en":"Scenario","ko":"시나리오","ar":"السيناريو"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["overview"], "registryIds": ["P06"], "baselinePage": null
  },
  {
    "id": "assistant", "group": "work",
    "title": {"en":"Billing assistant","ko":"청구 업무 도우미","ar":"مساعد الفوترة"},
    "description": {"en":"Billing assistant. Review the required inputs and preview a draft in this simulation.","ko":"청구 업무 도우미. 필수 입력을 검토하고 이 시뮬레이션에서 초안을 미리 확인합니다.","ar":"مساعد الفوترة. راجع المدخلات المطلوبة وعاين مسودة في هذه المحاكاة."},
    "kind": "form",
    "columns": [{"key":"question","label":{"en":"Question","ko":"문의 내용","ar":"السؤال"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"question","label":{"en":"Question","ko":"문의 내용","ar":"السؤال"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Preview draft","ko":"초안 미리보기","ar":"معاينة المسودة"},
    "related": ["operations"], "registryIds": ["A01"], "baselinePage": null
  },
  {
    "id": "reports", "group": "history",
    "title": {"en":"Scoped billing reports","ko":"범위별 청구 리포트","ar":"تقارير الفوترة حسب النطاق"},
    "description": {"en":"Scoped billing reports. Compare records using the same scope, currency and calculation basis.","ko":"범위별 청구 리포트. 동일 범위·통화·계산 기준으로 항목을 비교합니다.","ar":"تقارير الفوترة حسب النطاق. قارن السجلات باستخدام النطاق والعملة وأساس الحساب نفسه."},
    "kind": "report",
    "columns": [{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"currency","label":{"en":"Currency","ko":"통화","ar":"العملة"},"type":"text"},{"key":"amount","label":{"en":"Amount","ko":"금액","ar":"المبلغ"},"type":"number"},{"key":"net","label":{"en":"Net amount","ko":"순액","ar":"المبلغ الصافي"},"type":"number"}],
    "fields": [{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"currency","label":{"en":"Currency","ko":"통화","ar":"العملة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Preview report","ko":"리포트 미리보기","ar":"معاينة التقرير"},
    "related": ["cost-pivot"], "registryIds": ["R01"], "baselinePage": null
  },
  {
    "id": "audit-events", "group": "history",
    "title": {"en":"Audit event search","ko":"감사 이력 조회","ar":"البحث في أحداث التدقيق"},
    "description": {"en":"Audit event search. Search and filter the scoped records; inspect a record before continuing.","ko":"감사 이력 조회. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"البحث في أحداث التدقيق. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"created","label":{"en":"Created at","ko":"생성 시각","ar":"وقت الإنشاء"},"type":"date"},{"key":"actor","label":{"en":"Actual actor","ko":"실제 처리자","ar":"المنفذ الفعلي"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"},{"key":"version","label":{"en":"Version","ko":"버전","ar":"الإصدار"},"type":"text"},{"key":"result","label":{"en":"Processing result","ko":"처리 결과","ar":"نتيجة المعالجة"},"type":"status"}],
    "fields": [{"key":"actor","label":{"en":"Actual actor","ko":"실제 처리자","ar":"المنفذ الفعلي"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"},{"key":"scope","label":{"en":"Authorized scope","ko":"허용 범위","ar":"النطاق المصرح به"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["auto-history"], "registryIds": ["R02"], "baselinePage": null
  },
  {
    "id": "my-tasks", "group": "work",
    "title": {"en":"My tasks and notifications","ko":"내 작업·알림","ar":"مهامي وإشعاراتي"},
    "description": {"en":"My tasks and notifications. Search and filter the scoped records; inspect a record before continuing.","ko":"내 작업·알림. 범위 내 목록을 검색·필터링하고 항목을 확인한 뒤 진행합니다.","ar":"مهامي وإشعاراتي. ابحث في السجلات ضمن النطاق وصفها وراجع السجل قبل المتابعة."},
    "kind": "list",
    "columns": [{"key":"stage","label":{"en":"Processing stage","ko":"처리 단계","ar":"مرحلة المعالجة"},"type":"text"},{"key":"customer","label":{"en":"Customer","ko":"고객","ar":"العميل"},"type":"text"},{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"due","label":{"en":"Due date","ko":"기한","ar":"تاريخ الاستحقاق"},"type":"date"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"owner","label":{"en":"Responsible person","ko":"담당자","ar":"المسؤول"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Preview selection","ko":"선택 항목 미리보기","ar":"معاينة التحديد"},
    "related": ["operations"], "registryIds": ["U03"], "baselinePage": null
  },
  {
    "id": "supplier-naver", "group": "contracts",
    "title": {"en":"NAVER Cloud contract evidence · unverified","ko":"NAVER Cloud 계약 근거 · 미검증","ar":"أدلة عقد NAVER Cloud · غير متحقق"},
    "description": {"en":"NAVER Cloud contract evidence · unverified. Inspect the selected scope, version and supporting evidence.","ko":"NAVER Cloud 계약 근거 · 미검증. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"أدلة عقد NAVER Cloud · غير متحقق. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["support-matrix"], "registryIds": ["B04"], "baselinePage": null
  },
  {
    "id": "supplier-kt", "group": "contracts",
    "title": {"en":"KT Cloud contract evidence · unverified","ko":"KT Cloud 계약 근거 · 미검증","ar":"أدلة عقد KT Cloud · غير متحقق"},
    "description": {"en":"KT Cloud contract evidence · unverified. Inspect the selected scope, version and supporting evidence.","ko":"KT Cloud 계약 근거 · 미검증. 선택한 범위·버전과 연결된 근거를 확인합니다.","ar":"أدلة عقد KT Cloud · غير متحقق. راجع النطاق والإصدار المحددين والأدلة الداعمة."},
    "kind": "detail",
    "columns": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"account","label":{"en":"Billing account","ko":"청구 계정","ar":"حساب الفوترة"},"type":"text"},{"key":"basis","label":{"en":"Calculation basis","ko":"계산 기준","ar":"أساس الحساب"},"type":"text"},{"key":"evidence","label":{"en":"Evidence reference","ko":"증빙 참조","ar":"مرجع الدليل"},"type":"text"},{"key":"status","label":{"en":"Status","ko":"상태","ar":"الحالة"},"type":"status"}],
    "fields": [{"key":"contract","label":{"en":"Contract reference","ko":"계약 참조","ar":"مرجع العقد"},"type":"text"},{"key":"period","label":{"en":"Billing period","ko":"청구 기간","ar":"فترة الفوترة"},"type":"text"}],
    "action": {"en":"Review evidence","ko":"근거 검토","ar":"مراجعة الأدلة"},
    "related": ["support-matrix"], "registryIds": ["B04"], "baselinePage": null
  }
];
