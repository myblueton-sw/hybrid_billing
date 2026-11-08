import { Draft, Evidence, Ledger } from "../../../../shared/domain-screen-ui/operations/controls";
import type { Props, Field, Words } from "../../../../shared/domain-screen-ui/operations/types";
const client: Field[] = [
  {
    id: "client",
    label: ["Client 참조", "Client reference", "مرجع العميل البرمجي"],
    required: true,
  },
  {
    id: "audience",
    label: ["대상 audience", "Target audience", "الجمهور المستهدف"],
    required: true,
  },
  {
    id: "scope",
    label: [
      "허용 기능·설치 범위",
      "Permitted capabilities and installation scope",
      "القدرات المسموحة ونطاق التثبيت",
    ],
    kind: "textarea",
    required: true,
  },
];
const module: Field[] = [
  {
    id: "manifest",
    label: [
      "모듈 manifest 참조",
      "Module manifest reference",
      "مرجع بيان الوحدة",
    ],
    required: true,
  },
  {
    id: "digest",
    label: [
      "버전·digest 참조",
      "Version and digest reference",
      "مرجع الإصدار والبصمة",
    ],
    required: true,
  },
  {
    id: "lifecycle",
    label: [
      "저장·이관·삭제 계약",
      "Storage, migration and deletion contract",
      "عقد التخزين والترحيل والحذف",
    ],
    kind: "textarea",
    required: true,
  },
];
const channels: Field[] = [
  {
    id: "channel",
    label: ["채널 참조", "Channel reference", "مرجع القناة"],
    required: true,
  },
  {
    id: "destination",
    label: [
      "허용 목적지 참조",
      "Allowed destination reference",
      "مرجع الوجهة المسموحة",
    ],
    required: true,
  },
  {
    id: "key-version",
    label: [
      "서명 키 버전 참조 · 비밀값 제외",
      "Signing key version reference · no secret value",
      "مرجع إصدار مفتاح التوقيع · دون قيمة سرية",
    ],
    required: true,
  },
];
const events: Field[] = [
  {
    id: "event",
    label: ["원사건 참조", "Original event reference", "مرجع الحدث الأصلي"],
    required: true,
  },
  {
    id: "revision",
    label: [
      "원사건 revision",
      "Original event revision",
      "مراجعة الحدث الأصلي",
    ],
    required: true,
  },
  {
    id: "reason",
    label: [
      "재처리 검토 사유",
      "Replay review reason",
      "سبب مراجعة إعادة المعالجة",
    ],
    kind: "textarea",
    required: true,
  },
];
const reason: Words = [
  "등록·검증·사건 서비스가 연결되지 않았습니다. 현재 버전의 적격 여부를 판정할 수 없습니다.",
  "Registration, verification and event services are not connected. Current-version eligibility is unknown.",
  "خدمات التسجيل والتحقق والأحداث غير متصلة. أهلية الإصدار الحالي غير معروفة.",
];
export default function Integrations(p: Props) {
  const l = p.locale;
  return (
    <div className="ops-grid">
      <div>
        <Ledger
          locale={l}
          title={
            p.view === "events"
              ? ["사건 처리 현황", "Event processing", "معالجة الأحداث"]
              : ["확장 연결 목록", "Extension connections", "اتصالات الإضافات"]
          }
          columns={
            p.view === "events"
              ? [
                  ["사건", "Event", "الحدث"],
                  ["원본 revision", "Source revision", "مراجعة المصدر"],
                  ["작업 상태", "Job state", "حالة المهمة"],
                  ["확인 시각", "Observed at", "وقت الرصد"],
                ]
              : [
                  ["이름", "Name", "الاسم"],
                  ["버전·digest", "Version / digest", "الإصدار والبصمة"],
                  ["허용 범위", "Allowed scope", "النطاق المسموح"],
                  ["검증", "Verification", "التحقق"],
                ]
          }
          reason={reason}
        />
        <Draft
          props={p}
          title={
            p.view === "clients"
              ? [
                  "API client 등록 초안",
                  "API client registration draft",
                  "مسودة تسجيل عميل API",
                ]
              : p.view === "modules"
                ? [
                    "모듈 등록 초안",
                    "Module registration draft",
                    "مسودة تسجيل وحدة",
                  ]
                : p.view === "channels"
                  ? [
                      "구독·채널 초안",
                      "Subscription and channel draft",
                      "مسودة الاشتراك والقناة",
                    ]
                  : [
                      "재처리 범위 검토",
                      "Replay scope review",
                      "مراجعة نطاق إعادة المعالجة",
                    ]
          }
          fields={
            p.view === "clients"
              ? client
              : p.view === "modules"
                ? module
                : p.view === "channels"
                  ? channels
                  : events
          }
        />
      </div>
      <aside>
        <Evidence
          locale={l}
          title={["연결 계약", "Connection contract", "عقد الاتصال"]}
          fields={[
            ["권한·quota", "Authority and quota", "الصلاحيات والحصة"],
            ["서명·호환성", "Signature and compatibility", "التوقيع والتوافق"],
            [
              "송신 목적지 검증",
              "Destination verification",
              "التحقق من الوجهة",
            ],
            [
              "재시도·중복 방지",
              "Retry and duplicate protection",
              "إعادة المحاولة ومنع التكرار",
            ],
          ]}
          reason={reason}
        />
      </aside>
    </div>
  );
}
