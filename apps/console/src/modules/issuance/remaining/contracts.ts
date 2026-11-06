import type { Locale } from "../../../shared/types";

export type Text = readonly [string, string, string];
export const t = (l: Locale, text: Text) =>
  text[l === "ko" ? 0 : l === "en" ? 1 : 2];
export const financeIds = [
  "I01",
  "I02",
  "I03",
  "I04",
  "R01",
  "R02",
  "B02",
] as const;
export type FinanceId = (typeof financeIds)[number];
export type Props = {
  screenId: string;
  locale: Locale;
  params: URLSearchParams;
  view: string;
  onState: (state: string, items: string[]) => void;
  onDirty: (dirty: boolean) => void;
};
export type Field = {
  id: string;
  label: Text;
  kind:
    | "reference"
    | "money"
    | "quantity"
    | "time"
    | "date"
    | "evidence"
    | "status"
    | "interval";
};
const f = (
  id: string,
  label: Text,
  kind: Field["kind"] = "reference",
): Field => ({ id, label, kind });
export const fields = {
  document: f("document-number", [
    "문서 번호",
    "Document number",
    "رقم المستند",
  ]),
  kind: f("document-kind", [
    "문서 종류·발행 책임",
    "Document kind and issuance responsibility",
    "نوع المستند ومسؤولية الإصدار",
  ]),
  entity: f("legal-entity", [
    "발행 법인",
    "Issuing legal entity",
    "الكيان القانوني المصدر",
  ]),
  period: f(
    "service-period",
    ["서비스 기간", "Service period", "فترة الخدمة"],
    "interval",
  ),
  net: f(
    "net-amount",
    ["세전 금액·통화", "Net amount and currency", "المبلغ قبل الضريبة والعملة"],
    "money",
  ),
  tax: f(
    "tax-amount",
    ["세금·통화", "Tax amount and currency", "مبلغ الضريبة والعملة"],
    "money",
  ),
  total: f(
    "total-amount",
    ["총액·통화", "Total amount and currency", "المبلغ الإجمالي والعملة"],
    "money",
  ),
  template: f("template-revision", [
    "템플릿 개정",
    "Template revision",
    "مراجعة القالب",
  ]),
  bank: f("payment-reference", [
    "보호된 지급계좌 참조",
    "Protected payment-account reference",
    "مرجع حساب الدفع المحمي",
  ]),
  recipient: f("recipient-reference", [
    "수신자 참조",
    "Recipient reference",
    "مرجع المستلم",
  ]),
  original: f("original-document", [
    "확정 원문 참조",
    "Finalized original reference",
    "مرجع المستند الأصلي النهائي",
  ]),
  correction: f("correction-reference", [
    "정정 참조·잔여 범위",
    "Correction reference and remaining scope",
    "مرجع التصحيح والنطاق المتبقي",
  ]),
  approval: f(
    "approval-evidence",
    [
      "승인 개정·근거",
      "Approval revision and evidence",
      "مراجعة الموافقة وأدلتها",
    ],
    "evidence",
  ),
  issued: f(
    "issuance-state",
    ["발행 상태", "Issuance state", "حالة الإصدار"],
    "status",
  ),
  delivery: f(
    "delivery-state",
    ["전달 상태", "Delivery state", "حالة التسليم"],
    "status",
  ),
  hash: f("input-hash", [
    "승인 입력 해시",
    "Approved input hash",
    "بصمة المدخلات المعتمدة",
  ]),
  authority: f(
    "receivable-authority",
    [
      "채권 권위·개정",
      "Receivable authority and revision",
      "مرجع المديونية المعتمد ومراجعته",
    ],
    "evidence",
  ),
  term: f("payment-term", [
    "적용 지급 조건·개정",
    "Applied payment terms and revision",
    "شروط الدفع المطبقة ومراجعتها",
  ]),
  issueDate: f("issue-date", ["발행일", "Issue date", "تاريخ الإصدار"], "date"),
  originalDue: f(
    "original-due-date",
    [
      "원래 지급 기한",
      "Original payment due date",
      "تاريخ استحقاق الدفع الأصلي",
    ],
    "date",
  ),
  due: f(
    "current-due-date",
    [
      "현재 지급 기한",
      "Current payment due date",
      "تاريخ استحقاق الدفع الحالي",
    ],
    "date",
  ),
  balance: f(
    "confirmed-balance",
    [
      "확정 미수 잔액·통화",
      "Confirmed receivable balance and currency",
      "الرصيد المستحق المؤكد والعملة",
    ],
    "money",
  ),
  receipt: f(
    "receipt-evidence",
    [
      "수납 배정·효력일",
      "Receipt allocation and effective date",
      "تخصيص المبلغ المحصل وتاريخ سريانه",
    ],
    "evidence",
  ),
  receiptAt: f(
    "receipt-checked-at",
    [
      "수납 확인시각·최신성",
      "Receipt checked at and freshness",
      "وقت التحقق من التحصيل وحداثته",
    ],
    "time",
  ),
  overdue: f(
    "overdue-status",
    ["기한·연체 상태", "Due-date and overdue state", "حالة الاستحقاق والتأخر"],
    "status",
  ),
  overdueDays: f(
    "overdue-days",
    ["확정 연체 일수", "Confirmed overdue days", "أيام التأخر المؤكدة"],
    "quantity",
  ),
  dispute: f(
    "dispute-collections-hold",
    ["분쟁·독촉 보류", "Dispute and collections hold", "النزاع وتعليق التحصيل"],
    "status",
  ),
  purpose: f("submission-purpose", [
    "전송 목적",
    "Submission purpose",
    "غرض الإرسال",
  ]),
  month: f("recognition-month", [
    "월별 비용 인식 귀속월",
    "Expense recognition month",
    "شهر الاعتراف بالمصروفات",
  ]),
  postingPeriod: f("posting-period", [
    "전기 기간",
    "Posting period",
    "فترة الترحيل",
  ]),
  batch: f("recognition-batch", [
    "월별 인식 배치·승인 개정",
    "Monthly recognition batch and approval revision",
    "دفعة الاعتراف الشهري ومراجعة الموافقة",
  ]),
  posted: f(
    "actual-posting",
    [
      "외부 전표·실제 전기",
      "External journal and actual posting",
      "القيد الخارجي والترحيل الفعلي",
    ],
    "evidence",
  ),
  quarterly: f("quarterly-billing", [
    "분기 청구 연결·발행일",
    "Quarterly billing link and issue date",
    "رابط الفوترة الربع سنوية وتاريخ الإصدار",
  ]),
  recognitionNet: f(
    "recognition-net",
    [
      "월별 인식 순액·통화",
      "Monthly recognition net amount and currency",
      "صافي الاعتراف الشهري والعملة",
    ],
    "money",
  ),
  quarterAmount: f(
    "quarterly-amount",
    [
      "분기 청구액·통화",
      "Quarterly billed amount and currency",
      "المبلغ المفوتر ربع سنوياً والعملة",
    ],
    "money",
  ),
  adjustment: f(
    "recognition-adjustment",
    [
      "승인 조정·상계·역분개",
      "Approved adjustment, offset and reversal",
      "التعديل المعتمد والمقاصة والعكس",
    ],
    "evidence",
  ),
  difference: f(
    "reconciliation-difference",
    [
      "미대사 차이·통화",
      "Unreconciled difference and currency",
      "الفرق غير المطابق والعملة",
    ],
    "money",
  ),
  externalAt: f(
    "external-checked-at",
    ["외부 확인시각", "External checked at", "وقت التحقق الخارجي"],
    "time",
  ),
  erp: f("erp-connection", [
    "ERP 연결·제품·버전",
    "ERP connection, product and version",
    "اتصال ERP والمنتج والإصدار",
  ]),
  order: f("order-prepayment", [
    "기존 주문·선불 참조",
    "Existing order and prepayment references",
    "مراجع الطلبات والدفعات المقدمة القائمة",
  ]),
  submission: f("submission-revision", [
    "제출 ID·개정",
    "Submission ID and revision",
    "معرف الإرسال ومراجعته",
  ]),
  external: f("external-document", [
    "외부 문서·행 연결",
    "External document and row links",
    "روابط المستند الخارجي وصفوفه",
  ]),
  platformAmount: f(
    "platform-attributed-amount",
    [
      "플랫폼 귀속 순액·통화",
      "Platform-attributed net amount and currency",
      "صافي المبلغ المنسوب للمنصة والعملة",
    ],
    "money",
  ),
  wholeAmount: f(
    "whole-document-amount",
    [
      "전체 ERP 문서액·통화",
      "Whole ERP document amount and currency",
      "إجمالي مستند ERP والعملة",
    ],
    "money",
  ),
  quantity: f(
    "source-quantity",
    ["수량·단위", "Quantity and unit", "الكمية والوحدة"],
    "quantity",
  ),
  externalRevision: f("external-revision", [
    "외부 개정",
    "External revision",
    "المراجعة الخارجية",
  ]),
  autoDelivery: f(
    "automatic-delivery-control",
    [
      "자동 첨부·발송 통제 근거",
      "Automatic attachment and delivery control evidence",
      "أدلة ضبط المرفقات والتسليم التلقائي",
    ],
    "evidence",
  ),
  lastPoll: f(
    "last-poll",
    ["마지막 상태 조회", "Last status check", "آخر تحقق من الحالة"],
    "time",
  ),
  nextPoll: f(
    "next-poll",
    ["다음 상태 조회", "Next status check", "التحقق التالي من الحالة"],
    "time",
  ),
  externalEvent: f("external-event", [
    "외부 사건 ID·원천·개정",
    "External event ID, source and revision",
    "معرف الحدث الخارجي ومصدره ومراجعته",
  ]),
  job: f(
    "job-state",
    [
      "작업 상태·불명 사유",
      "Job state and unknown-result reason",
      "حالة المهمة وسبب عدم معرفة النتيجة",
    ],
    "status",
  ),
  filePurpose: f("file-purpose", ["파일 목적", "File purpose", "غرض الملف"]),
  generator: f("generator-schema", [
    "생성기·스키마 버전",
    "Generator and schema version",
    "إصدار المولد والمخطط",
  ]),
  rowIds: f("submission-row-ids", [
    "공통 제출·행 ID",
    "Common submission and row IDs",
    "معرفات الإرسال والصفوف المشتركة",
  ]),
  manifest: f("split-manifest", [
    "분할 파일 목록·행 수",
    "Split manifest and row counts",
    "بيان الملفات المجزأة وعدد الصفوف",
  ]),
  owner: f("handoff-owner", [
    "인계 담당·기한",
    "Handoff owner and deadline",
    "مسؤول التسليم وموعده",
  ]),
  download: f(
    "download-state",
    ["다운로드 기록", "Download record", "سجل التنزيل"],
    "status",
  ),
  applied: f(
    "external-application",
    [
      "외부 반영 상태·근거",
      "External application state and evidence",
      "حالة المعالجة الخارجية وأدلتها",
    ],
    "evidence",
  ),
  package: f("package-id", [
    "전달 패키지 ID",
    "Delivery package ID",
    "معرف حزمة التسليم",
  ]),
  policy: f("disclosure-policy", [
    "공개 정책 개정",
    "Disclosure policy revision",
    "مراجعة سياسة الإفصاح",
  ]),
  allowed: f("allowed-fields", [
    "허용 필드·집계",
    "Allowed fields and aggregation",
    "الحقول والتجميعات المسموح بها",
  ]),
  sanitization: f(
    "workbook-sanitization",
    [
      "새 공개 파일 정제 근거",
      "New public workbook sanitization evidence",
      "أدلة تنقية ملف الإفصاح الجديد",
    ],
    "evidence",
  ),
  validation: f(
    "full-validation",
    [
      "전량 행·금액 검증",
      "Full row and amount validation",
      "التحقق الكامل من الصفوف والمبالغ",
    ],
    "evidence",
  ),
  fileHash: f("artifact-hash", [
    "고정 파일 해시",
    "Immutable artifact hash",
    "بصمة الملف الثابت",
  ]),
  expiry: f(
    "access-expiry",
    [
      "관리 링크 접근 만료",
      "Managed-link access expiry",
      "انتهاء الوصول إلى الرابط المدار",
    ],
    "time",
  ),
  hold: f("hold-type-manifest", [
    "보류 유형·대상·개정",
    "Hold type, targets and revisions",
    "نوع التعليق وأهدافه ومراجعاته",
  ]),
  holdEffect: f(
    "hold-effect-evidence",
    [
      "보류 효력·외부 확인",
      "Hold effect and external confirmation",
      "أثر التعليق والتأكيد الخارجي",
    ],
    "evidence",
  ),
  amountMeaning: f("amount-meaning", [
    "금액 의미",
    "Amount meaning",
    "دلالة المبلغ",
  ]),
  originalCurrency: f("original-currency", [
    "원통화",
    "Original currency",
    "العملة الأصلية",
  ]),
  timeBasis: f("time-basis", [
    "당시 귀속·현재 조직 기준",
    "Historical attribution or current organization basis",
    "الإسناد التاريخي أو أساس المؤسسة الحالية",
  ]),
  dimensions: f("group-dimensions", [
    "그룹 차원",
    "Group dimensions",
    "أبعاد التجميع",
  ]),
  source: f(
    "source-completeness",
    [
      "출처·완전성·기준시각",
      "Source, completeness and as-of",
      "المصدر والاكتمال والوقت المرجعي",
    ],
    "evidence",
  ),
  comparison: f("comparison-denominator", [
    "비교 기간·분모",
    "Comparison period and denominator",
    "فترة المقارنة ومقامها",
  ]),
  converted: f(
    "display-conversion",
    [
      "표시 환산액·환율 근거",
      "Display conversion and exchange-rate evidence",
      "التحويل للعرض ودليل سعر الصرف",
    ],
    "money",
  ),
  aging: f(
    "aging-bucket",
    [
      "연체 구간·통화별 잔액",
      "Aging bucket and balance by currency",
      "شريحة التأخر والرصيد حسب العملة",
    ],
    "money",
  ),
  event: f("event-id", [
    "사건·요청 참조",
    "Event and request reference",
    "مرجع الحدث والطلب",
  ]),
  happened: f(
    "occurred-at",
    ["실제 발생시각", "Occurred at", "وقت الحدوث"],
    "time",
  ),
  received: f(
    "received-at",
    ["수신시각", "Received at", "وقت الاستلام"],
    "time",
  ),
  actor: f("actual-actor", [
    "실제 행위자·주체 유형",
    "Actual actor and principal kind",
    "الفاعل الحقيقي ونوع الهوية",
  ]),
  delegation: f("delegation-context", [
    "대리·지원 범위·만료",
    "Delegation or support scope and expiry",
    "نطاق التفويض أو الدعم وانتهاؤه",
  ]),
  route: f("execution-path", [
    "UI·API·AI·자동화 경로",
    "UI, API, AI or automation path",
    "مسار الواجهة أو API أو الذكاء الاصطناعي أو الأتمتة",
  ]),
  action: f(
    "action-result",
    [
      "행위·접수·실제 결과",
      "Action, acceptance and actual outcome",
      "الإجراء والقبول والنتيجة الفعلية",
    ],
    "status",
  ),
  before: f("before-revision", [
    "변경 전 개정",
    "Before revision",
    "المراجعة السابقة",
  ]),
  after: f("after-revision", [
    "변경 후 개정",
    "After revision",
    "المراجعة اللاحقة",
  ]),
  retention: f(
    "retention-hold",
    [
      "보존·법적 보류",
      "Retention and legal hold",
      "الاحتفاظ والتعليق القانوني",
    ],
    "status",
  ),
  asset: f("asset-meter-id", [
    "자산·미터 ID",
    "Asset and meter ID",
    "معرف الأصل والعداد",
  ]),
  interval: f(
    "effective-interval",
    [
      "사양·귀속 유효구간",
      "Specification and attribution interval",
      "فترة سريان المواصفات والإسناد",
    ],
    "interval",
  ),
  measurement: f("measurement-kind", [
    "누적·구간합계·순간값",
    "Cumulative, interval total or instantaneous",
    "تراكمي أو إجمالي الفترة أو لحظي",
  ]),
  grade: f(
    "measurement-grade",
    [
      "측정등급·추정 여부",
      "Measurement grade and estimation",
      "درجة القياس وحالة التقدير",
    ],
    "status",
  ),
  reset: f("reset-event", [
    "누락·reset 사건",
    "Gap and reset event",
    "حدث الفجوة وإعادة الضبط",
  ]),
  model: f("model-alias", [
    "시점별 모델 별칭",
    "Time-specific model alias",
    "اسم النموذج حسب الوقت",
  ]),
  denominator: f(
    "rate-denominator",
    ["단가 분모·단위", "Rate denominator and unit", "مقام السعر والوحدة"],
    "quantity",
  ),
};
export type FieldKey = keyof typeof fields;
export const viewFields: Record<
  FinanceId,
  Record<string, readonly FieldKey[]>
> = {
  I01: {
    documents: [
      "document",
      "kind",
      "entity",
      "period",
      "total",
      "issued",
      "delivery",
    ],
    body: [
      "document",
      "kind",
      "entity",
      "period",
      "net",
      "tax",
      "total",
      "template",
      "bank",
      "recipient",
    ],
    corrections: ["original", "correction", "hash", "approval"],
    history: ["approval", "hash", "issued", "delivery", "externalAt"],
    "payment-terms": [
      "authority",
      "term",
      "issueDate",
      "originalDue",
      "due",
      "balance",
      "receipt",
      "receiptAt",
      "overdue",
      "overdueDays",
      "dispute",
    ],
    recognition: [
      "quarterly",
      "batch",
      "posted",
      "adjustment",
      "difference",
      "externalAt",
    ],
  },
  I02: {
    submissions: [
      "submission",
      "external",
      "entity",
      "posted",
      "platformAmount",
      "wholeAmount",
      "nextPoll",
    ],
    comparison: [
      "hash",
      "platformAmount",
      "wholeAmount",
      "tax",
      "quantity",
      "externalRevision",
      "approval",
    ],
    rows: [
      "submission",
      "rowIds",
      "order",
      "external",
      "platformAmount",
      "wholeAmount",
      "quantity",
    ],
    events: [
      "externalEvent",
      "job",
      "externalRevision",
      "lastPoll",
      "nextPoll",
    ],
    capabilities: ["erp", "autoDelivery", "posted", "lastPoll", "nextPoll"],
    "payment-terms": [
      "authority",
      "term",
      "due",
      "receipt",
      "receiptAt",
      "balance",
      "overdue",
    ],
    recognition: [
      "purpose",
      "month",
      "postingPeriod",
      "batch",
      "posted",
      "quarterly",
      "recognitionNet",
      "adjustment",
      "difference",
      "externalAt",
    ],
  },
  I03: {
    handoffs: [
      "submission",
      "filePurpose",
      "template",
      "manifest",
      "download",
      "applied",
      "owner",
    ],
    files: [
      "filePurpose",
      "template",
      "generator",
      "hash",
      "rowIds",
      "manifest",
      "fileHash",
      "total",
    ],
    outcomes: ["submission", "external", "applied", "owner", "externalAt"],
    reconciliation: [
      "rowIds",
      "platformAmount",
      "quantity",
      "applied",
      "difference",
    ],
    recognition: [
      "purpose",
      "month",
      "batch",
      "manifest",
      "download",
      "posted",
      "externalAt",
    ],
  },
  I04: {
    policies: [
      "filePurpose",
      "recipient",
      "allowed",
      "policy",
      "generator",
      "expiry",
    ],
    preview: [
      "recipient",
      "allowed",
      "policy",
      "sanitization",
      "validation",
      "tax",
    ],
    packages: [
      "package",
      "document",
      "recipient",
      "policy",
      "manifest",
      "total",
      "validation",
      "fileHash",
    ],
    history: [
      "package",
      "delivery",
      "autoDelivery",
      "expiry",
      "hold",
      "holdEffect",
      "externalAt",
    ],
  },
  R01: {
    summary: [
      "amountMeaning",
      "originalCurrency",
      "timeBasis",
      "dimensions",
      "source",
      "total",
      "quantity",
    ],
    comparison: [
      "period",
      "comparison",
      "amountMeaning",
      "originalCurrency",
      "total",
      "source",
    ],
    rows: [
      "dimensions",
      "period",
      "amountMeaning",
      "originalCurrency",
      "total",
      "converted",
      "quantity",
      "source",
    ],
    views: ["timeBasis", "dimensions", "amountMeaning", "originalCurrency"],
    exports: [
      "filePurpose",
      "manifest",
      "policy",
      "generator",
      "fileHash",
      "job",
    ],
    "payment-terms": [
      "authority",
      "due",
      "overdueDays",
      "aging",
      "receiptAt",
      "source",
    ],
    recognition: [
      "month",
      "recognitionNet",
      "quarterAmount",
      "adjustment",
      "difference",
      "externalAt",
    ],
  },
  R02: {
    events: ["event", "happened", "received", "actor", "action", "route"],
    detail: [
      "event",
      "actor",
      "delegation",
      "route",
      "action",
      "approval",
      "external",
    ],
    diff: ["event", "before", "after", "hash", "approval"],
    evidence: ["event", "hash", "approval", "external", "retention"],
  },
  B02: {
    assets: ["asset", "interval", "quantity", "grade", "reset", "source"],
    intervals: [
      "asset",
      "interval",
      "measurement",
      "quantity",
      "grade",
      "source",
    ],
    gaps: ["asset", "interval", "reset", "grade", "source"],
    meters: ["asset", "model", "measurement", "denominator", "grade"],
  },
};
export const paymentKeys = viewFields.I01["payment-terms"];
export const recognitionKeys: readonly FieldKey[] = [
  "month",
  "recognitionNet",
  "posted",
  "quarterly",
  "quarterAmount",
  "adjustment",
  "difference",
  "externalAt",
];
