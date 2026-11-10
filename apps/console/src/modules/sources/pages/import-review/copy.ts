import type { Locale } from "../../../../shared/types";
import type { Receipt, RecordKind } from "./contract";
export type Words = readonly [string, string, string];
export const t = (l: Locale, w: Words) =>
  w[l === "ko" ? 0 : l === "en" ? 1 : 2];
export const words = {
  environment: [
    "수집 접수 조회",
    "Collection receipt inspection",
    "فحص إيصالات التجميع",
  ],
  scope: [
    "허용된 수집 범위",
    "Permitted collection scope",
    "نطاق التجميع المسموح",
  ],
  local: [
    "고정된 로컬 reader 권한으로 보존된 접수 메타데이터를 조회합니다. 실제 사용자 인증이나 청구 승인을 대신하지 않습니다.",
    "Inspect retained receipt metadata under the fixed local reader scope. This does not establish user authentication or billing approval.",
    "افحص بيانات الإيصالات المحفوظة ضمن نطاق القارئ المحلي الثابت. لا يثبت ذلك مصادقة المستخدم أو اعتماد الفوترة.",
  ],
  loading: [
    "현재 조회 범위와 접수 근거를 확인하고 있습니다.",
    "Checking current scope and receipt evidence.",
    "جارٍ التحقق من النطاق الحالي وأدلة الإيصالات.",
  ],
  unconnected: [
    "수집 접수 조회 서비스가 연결되지 않았습니다. 건수와 현재 단계는 미확인입니다.",
    "The collection receipt reader is not connected. Count and current stage are unknown.",
    "خدمة قراءة إيصالات التجميع غير متصلة. العدد والمرحلة الحالية غير معروفين.",
  ],
  error: [
    "조회하지 못했습니다. 이전 목록과 선택 상세를 지웠습니다.",
    "Unable to read evidence. The previous list and selected detail have been cleared.",
    "تعذر قراءة الأدلة. تم مسح القائمة السابقة والتفاصيل المحددة.",
  ],
  unknown: ["미확인", "Unknown", "غير معروف"],
  empty: [
    "현재 허용 범위와 검색 조건에 표시할 메타데이터가 없습니다. 전체 수집 완료나 수집되지 않음을 뜻하지 않습니다.",
    "No metadata is visible for the current permitted scope and filter. This does not establish complete collection or absence of intake.",
    "لا توجد بيانات وصفية ظاهرة للنطاق المسموح والمرشح الحالي. لا يثبت ذلك اكتمال التجميع أو عدم الاستقبال.",
  ],
  unsupported: [
    "이 범위의 등록된 기대 수집량을 조회할 수 없습니다. 완전성과 전체 건수는 미확인입니다.",
    "Registered collection expectations are unavailable for this scope. Completeness and total count remain unknown.",
    "توقعات التجميع المسجلة غير متاحة لهذا النطاق. يظل الاكتمال والعدد الإجمالي غير معروفين.",
  ],
  extraction: [
    "OCR·추출 검증 서비스는 미연결입니다. 검증된 문서·금액·작업자 확인을 생성하지 않습니다.",
    "OCR and extraction verification are not connected. No verified documents, amounts or worker confirmations are generated.",
    "خدمات OCR والتحقق من الاستخراج غير متصلة. لا تُنشأ مستندات أو مبالغ متحقق منها أو تأكيدات مسؤول.",
  ],
  completeness: [
    "운영자가 등록한 달력·대상 기준의 관찰입니다. CSP 전체 데이터의 완전성이나 청구 적격성을 검증하지 않습니다.",
    "Observations use operator-registered calendars and domains. They do not verify upstream CSP completeness or billing eligibility.",
    "تعتمد الملاحظات على تقاويم ونطاقات مسجلة لدى المشغّل. لا تتحقق من اكتمال بيانات المزوّد أو أهلية الفوترة.",
  ],
  metadata: [
    "수집 당시 메타데이터 근거입니다. 현재 원본·웨어하우스 가용성, 독립 검증과 작업자 확인은 별도입니다.",
    "Metadata records from collection time. Current original/warehouse availability, independent validation and worker confirmation are separate.",
    "سجلات بيانات وصفية من وقت التجميع. توافر الأصل والمستودع حاليًا والتحقق المستقل وتأكيد المسؤول أمور منفصلة.",
  ],
  query: [
    "식별자·연결 접두어",
    "Identifier or connection prefix",
    "بادئة المعرّف أو الاتصال",
  ],
  apply: ["검색", "Search", "بحث"],
  clear: ["검색 지우기", "Clear search", "مسح البحث"],
  refresh: ["새로 조회", "Refresh evidence", "تحديث الأدلة"],
  retry: [
    "처음부터 다시 조회",
    "Reload from the start",
    "إعادة التحميل من البداية",
  ],
  invalid: [
    "영문 소문자·숫자·콜론·밑줄·하이픈으로 된 식별자를 128자 이내로 입력하세요.",
    "Use an identifier up to 128 lowercase letters, digits, colons, underscores or hyphens.",
    "استخدم معرّفًا حتى 128 حرفًا صغيرًا أو رقمًا أو نقطتين أو شرطة سفلية أو شرطة.",
  ],
  next: ["다음 페이지", "Next page", "الصفحة التالية"],
  back: ["이전 페이지", "Previous page", "الصفحة السابقة"],
  inspect: ["근거 열기", "Inspect evidence", "فحص الدليل"],
  close: ["상세 닫기", "Close detail", "إغلاق التفاصيل"],
  details: [
    "선택 접수 근거",
    "Selected receipt evidence",
    "أدلة الإيصال المحدد",
  ],
  deferred: [
    "수신·재처리·확인·공개 명령은 이 화면에서 실행하지 않습니다.",
    "Receipt, replay, confirmation and publication commands are not executed here.",
    "لا تُنفّذ أوامر الاستقبال وإعادة المعالجة والتأكيد والنشر هنا.",
  ],
} satisfies Record<string, Words>;
const stageWords: Record<string, Words> = {
  registered: ["수집 등록", "Collection registered", "تم تسجيل التجميع"],
  verified: [
    "저장 검증 기록",
    "Storage verification recorded",
    "تم تسجيل التحقق من التخزين",
  ],
  sealed: [
    "원천 공개 기록",
    "Source publication recorded",
    "تم نشر مراجعة المصدر",
  ],
  quarantined: ["격리", "Quarantined", "معزول"],
  "replacement-requested": [
    "대체 자료 요청",
    "Replacement requested",
    "طُلب بديل",
  ],
  escalated: ["검토 상향", "Escalated", "تم التصعيد"],
  resolved: [
    "격리 처리 기록",
    "Quarantine resolution recorded",
    "تم تسجيل معالجة العزل",
  ],
  deleted: [
    "삭제 결정 기록",
    "Deletion decision recorded",
    "تم تسجيل قرار الحذف",
  ],
  missing: [
    "기대 접수 없음",
    "Expected receipt missing",
    "الإيصال المتوقع مفقود",
  ],
  late: [
    "등록 기한 경과",
    "Registered deadline passed",
    "انقضت المهلة المسجلة",
  ],
};
export const stage = (l: Locale, r: Receipt) =>
  t(l, stageWords[r.stage] ?? words.unknown);
export const kind = (l: Locale, k: RecordKind) =>
  t(
    l,
    k === "delivery"
      ? ["수집 접수", "Collection delivery", "تسليم التجميع"]
      : k === "quarantine"
        ? ["격리 접수", "Quarantine receipt", "إيصال العزل"]
        : ["등록 기대 범위", "Registered expectation", "التوقع المسجل"],
  );
export const labels = {
  id: ["근거 ID", "Evidence ID", "معرّف الدليل"],
  kind: ["근거 종류", "Evidence kind", "نوع الدليل"],
  connection: ["연결 참조", "Connection reference", "مرجع الاتصال"],
  stage: ["최근 수집 단계", "Latest collection stage", "آخر مرحلة للتجميع"],
  window_start: [
    "자료 기간 시작 · 포함",
    "Source window start · inclusive",
    "بداية فترة المصدر · مشمولة",
  ],
  window_end: [
    "자료 기간 종료 · 제외",
    "Source window end · exclusive",
    "نهاية فترة المصدر · مستثناة",
  ],
  source_revision: ["원천 개정", "Source revision", "مراجعة المصدر"],
  focus_version: ["FOCUS 버전", "FOCUS version", "إصدار FOCUS"],
  format: ["입력 형식", "Input format", "صيغة المدخل"],
  declared_rows: ["선언된 행 수", "Declared rows", "الصفوف المصرّح بها"],
  file_bytes: [
    "원본 파일 크기 · 바이트",
    "Original file size · bytes",
    "حجم الملف الأصلي · بايت",
  ],
  file_digest: ["원본 해시", "Original digest", "بصمة الأصل"],
  mapping_digest: ["매핑 해시", "Mapping digest", "بصمة الربط"],
  received_at: [
    "접수 기준 시각",
    "Receipt reference time",
    "وقت مرجع الاستلام",
  ],
  complete_through: [
    "현재 공개본의 수집 기준 시각",
    "Current published collection-through time",
    "وقت التجميع حتى في الإصدار المنشور الحالي",
  ],
  finalized: [
    "현재 공개본의 원천 종료 선언",
    "Current publication's source-finalization declaration",
    "إعلان انتهاء المصدر للإصدار المنشور الحالي",
  ],
  reason: ["비민감 사유 코드", "Nonsensitive reason code", "رمز سبب غير حساس"],
  root_receipt: [
    "계열 최초 접수 참조",
    "Lineage root receipt reference",
    "مرجع الإيصال الأول في السلسلة",
  ],
  replaced_by: [
    "대체 접수 참조",
    "Replacement receipt reference",
    "مرجع الإيصال البديل",
  ],
  attempts: [
    "접수 계열 시도 수",
    "Receipt lineage attempts",
    "محاولات سلسلة الإيصال",
  ],
  known_zero: [
    "명시적 0 확인 근거",
    "Explicit zero attestation",
    "دليل الإقرار الصريح بالصفر",
  ],
  published: [
    "현재 공개본 존재",
    "Current publication exists",
    "يوجد إصدار منشور حالي",
  ],
} satisfies Record<keyof Receipt, Words>;
export function timestamp(value: string) {
  return value.replace("T", " ").replace(/(?:Z|\+00:00)$/, " UTC");
}

export const shortTimestamp = (value: string) =>
  timestamp(value).replace(/\.\d+(?= UTC$)/, "");
