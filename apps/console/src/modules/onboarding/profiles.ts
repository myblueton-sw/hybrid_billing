import type { Copy } from "../../shared/types";
export type Field = { key: string; label: Copy; sample: string };
export type ProviderProfile = {
  name: string;
  sample: string;
  pattern?: string;
  unverified?: boolean;
  fields: Field[];
};
const f = (
  key: string,
  en: string,
  ko: string,
  ar: string,
  sample: string,
): Field => ({ key, label: { en, ko, ar }, sample });
export const profiles: Record<string, ProviderProfile> = {
  aws: {
    name: "AWS",
    sample: "123456789012",
    pattern: "[0-9]{12}",
    fields: [
      f(
        "role",
        "Reader role ARN",
        "읽기 Role ARN",
        "ARN لدور القراءة",
        "arn:aws:iam::123456789012:role/ExampleReader",
      ),
      f(
        "export",
        "Data Export reference",
        "Data Export 참조",
        "مرجع تصدير البيانات",
        "EXAMPLE-EXPORT",
      ),
      f("bucket", "S3 bucket", "S3 버킷", "حاوية S3", "example-billing-export"),
      f("prefix", "Object prefix", "객체 Prefix", "بادئة الكائن", "focus/"),
      f(
        "region",
        "Storage region",
        "저장소 리전",
        "منطقة التخزين",
        "us-east-1",
      ),
    ],
  },
  azure: {
    name: "Azure",
    sample: "11111111-2222-3333-4444-555555555555",
    fields: [
      f(
        "billingScope",
        "Billing scope resource ID",
        "청구 범위 Resource ID",
        "معرّف مورد نطاق الفوترة",
        "/providers/Microsoft.Billing/billingAccounts/example/billingProfiles/example",
      ),
      f(
        "storage",
        "Storage resource ID",
        "Storage Resource ID",
        "معرّف مورد التخزين",
        "/subscriptions/example/resourceGroups/example/providers/Microsoft.Storage/storageAccounts/example",
      ),
      f(
        "container",
        "Blob container",
        "Blob 컨테이너",
        "حاوية Blob",
        "billing",
      ),
      f(
        "export",
        "Export name",
        "Export 이름",
        "اسم التصدير",
        "example-focus-export",
      ),
    ],
  },
  gcp: {
    name: "Google Cloud",
    sample: "ABCDEF-123456-ABCDEF",
    fields: [
      f(
        "queryProject",
        "Query project",
        "쿼리 프로젝트",
        "مشروع الاستعلام",
        "example-billing-query",
      ),
      f(
        "project",
        "Export project",
        "Export 프로젝트",
        "مشروع التصدير",
        "example-billing-source",
      ),
      f(
        "location",
        "Dataset location",
        "데이터셋 위치",
        "موقع مجموعة البيانات",
        "US",
      ),
      f(
        "dataset",
        "Linked dataset",
        "연결된 데이터셋",
        "مجموعة البيانات المرتبطة",
        "gcp_billing_immutable_EXAMPLE_US",
      ),
    ],
  },
  oci: {
    name: "OCI",
    sample: "ocid1.tenancy.oc1..example",
    fields: [
      f(
        "region",
        "Report owner home region",
        "보고서 소유 Home Region",
        "المنطقة الرئيسية لمالك التقرير",
        "ap-seoul-1",
      ),
      f(
        "namespace",
        "Oracle report namespace",
        "Oracle 보고서 Namespace",
        "نطاق أسماء تقارير Oracle",
        "bling",
      ),
      f(
        "bucket",
        "Report owner tenancy OCID",
        "보고서 소유 테넌시 OCID",
        "OCID لمستأجر مالك التقرير",
        "ocid1.tenancy.oc1..example",
      ),
      f(
        "prefix",
        "Report prefix",
        "보고서 Prefix",
        "بادئة التقرير",
        "FOCUS Reports",
      ),
    ],
  },
  alibaba: {
    name: "Alibaba Cloud",
    sample: "1234567890123456",
    fields: [
      f(
        "role",
        "RAM reader role reference",
        "RAM 읽기 역할 참조",
        "مرجع دور القراءة RAM",
        "example-ram-reader",
      ),
      f("bucket", "OSS bucket", "OSS 버킷", "حاوية OSS", "example-focus-bills"),
      f("region", "OSS region", "OSS 리전", "منطقة OSS", "oss-ap-southeast-1"),
      f(
        "official",
        "Official bill source reference",
        "공식 청구서 원천 참조",
        "مرجع مصدر الفاتورة الرسمية",
        "example-official-bill",
      ),
    ],
  },
  ibm: {
    name: "IBM Cloud",
    sample: "example-ibm-account",
    fields: [
      f(
        "fileRef",
        "Protected monthly CSV reference",
        "보호된 월별 CSV 참조",
        "مرجع CSV الشهري المحمي",
        "example-ibm-focus.csv",
      ),
      f(
        "official",
        "Official invoice reference",
        "공식 인보이스 참조",
        "مرجع الفاتورة الرسمية",
        "example-ibm-invoice",
      ),
    ],
  },
  vmware: {
    name: "VMware",
    sample: "https://vcenter.example.invalid",
    fields: [
      f(
        "collector",
        "Registered collector ID",
        "등록된 수집기 ID",
        "معرّف المجمع المسجل",
        "example-onprem-collector",
      ),
      f(
        "ca",
        "Trusted CA reference",
        "신뢰 CA 참조",
        "مرجع سلطة الشهادات الموثوقة",
        "example-ca-bundle",
      ),
      f(
        "resources",
        "Authorized resource scope",
        "허용 리소스 범위",
        "نطاق الموارد المسموح",
        "example-datacenter",
      ),
      f(
        "meter",
        "Metering profile",
        "계량 프로파일",
        "ملف القياس",
        "example-meter",
      ),
      f(
        "policy",
        "Approved internal-cost policy",
        "승인된 내부 원가 정책",
        "سياسة التكلفة الداخلية المعتمدة",
        "example-cost-policy",
      ),
    ],
  },
  file: {
    name: "File / storage",
    sample: "example-file-source",
    fields: [
      f(
        "fileRef",
        "Protected source reference",
        "보호된 원천 참조",
        "مرجع المصدر المحمي",
        "example-synthetic.csv",
      ),
      f(
        "profile",
        "Declared schema profile",
        "선언 규격 프로파일",
        "ملف المخطط المعلن",
        "example-focus-profile",
      ),
    ],
  },
  naver: {
    name: "NAVER Cloud",
    sample: "example-naver-account",
    unverified: true,
    fields: [],
  },
  kt: {
    name: "KT Cloud",
    sample: "example-kt-account",
    unverified: true,
    fields: [],
  },
};
export const profileNotes: Record<string, Copy> = {
  aws: {
    en: "Standard new-stack example: commercial us-east-1, SSE-S3. Advanced/cross-account/KMS profiles require separate review.",
    ko: "신규 표준 Stack 예시: commercial us-east-1·SSE-S3. 고급·교차 계정·KMS 프로파일은 별도 검토 대상입니다.",
    ar: "مثال Stack قياسي جديد: commercial us-east-1 وSSE-S3. الإعدادات المتقدمة وعبر الحسابات وKMS تحتاج مراجعة منفصلة.",
  },
  azure: {
    en: "MCA/EA example only. MOSP, management-group and cross-tenant storage profiles remain blocked.",
    ko: "MCA·EA 예시만 제공합니다. MOSP·관리 그룹·교차 테넌트 저장소 프로파일은 차단됩니다.",
    ar: "مثال MCA/EA فقط. ملفات MOSP ومجموعات الإدارة والتخزين عبر المستأجرين محظورة.",
  },
  gcp: {
    en: "Preview availability and BigQuery billing/query rights must be verified separately. This example only accepts US/EU location profiles.",
    ko: "Preview 제공 여부와 BigQuery 청구·쿼리 권한을 별도 확인해야 합니다. 이 예시는 US·EU 위치 프로파일만 허용합니다.",
    ar: "يلزم التحقق منفصلاً من توافر Preview وصلاحيات الفوترة والاستعلام. يقبل المثال مواقع US/EU فقط.",
  },
  oci: {
    en: "Use Oracle-owned reports; no new export bucket is created. Full parent/child tenancy custody must be approved.",
    ko: "Oracle 소유 보고서를 사용하며 새 Export 버킷을 만들지 않습니다. 부모·자식 테넌시 전체 보관 권한이 필요합니다.",
    ar: "استخدم تقارير Oracle؛ لا تُنشأ حاوية تصدير جديدة. يلزم اعتماد حفظ بيانات المستأجرين الأصل والتابعين.",
  },
  alibaba: {
    en: "Invitation-only Preview example. FOCUS is analysis-only; it cannot establish settlement evidence.",
    ko: "초대 Preview 예시입니다. FOCUS는 분석 전용이며 정산 근거가 될 수 없습니다.",
    ar: "مثال Preview بالدعوة فقط. بيانات FOCUS للتحليل ولا تثبت أهلية التسوية.",
  },
  ibm: {
    en: "This path represents account monthly CSV only. Account/enterprise API profiles are unverified; no recurring API connection is implied.",
    ko: "이 경로는 계정 월별 CSV입니다. Account·Enterprise API 프로파일은 미검증이며 정기 API 연결을 뜻하지 않습니다.",
    ar: "هذا المسار لملف CSV الشهري للحساب فقط. ملفات API للحساب والمؤسسة غير متحققة ولا يُفترض اتصال دوري.",
  },
  vmware: {
    en: "A vCenter connection is not cost evidence. Metering and approved internal-cost policy are both required.",
    ko: "vCenter 연결은 비용 근거가 아닙니다. 계량과 승인된 내부 원가 정책이 모두 필요합니다.",
    ar: "اتصال vCenter ليس دليلاً على التكلفة. يلزم القياس وسياسة تكلفة داخلية معتمدة.",
  },
  file: {
    en: "One-time input only in this example; declared schema is checked against received evidence.",
    ko: "이 예시는 일회 입력이며 선언 규격을 수신 근거와 대조합니다.",
    ar: "إدخال لمرة واحدة في هذا المثال؛ يُقارن المخطط المعلن بالأدلة المستلمة.",
  },
};
