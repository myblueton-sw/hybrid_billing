import { Draft, Evidence, Ledger, Panel } from "../../../../shared/domain-screen-ui/operations/controls";
import { t, type Props, type Field, type Words } from "../../../../shared/domain-screen-ui/operations/types";
const reason: Words = [
  "요구 추적과 실제 실행 artifact가 연결되지 않았습니다. 문서·화면의 존재를 시험 통과나 제품 지원으로 간주하지 않습니다.",
  "Requirement traces and actual execution artifacts are not connected. Documents and screens do not prove test success or product support.",
  "تتبع المتطلبات ومواد التنفيذ الفعلي غير متصلين. وجود مستندات أو شاشات لا يثبت اجتياز الاختبار أو دعم المنتج.",
];
const fields: Field[] = [
  {
    id: "combination",
    label: [
      "Provider·계약·역할·버전·경로",
      "Provider, contract, role, version and path",
      "المزوّد والعقد والدور والإصدار والمسار",
    ],
    required: true,
  },
  {
    id: "requirement",
    label: [
      "요구·인수 ID 참조",
      "Requirement and acceptance ID references",
      "مراجع معرّفات المتطلبات والقبول",
    ],
    required: true,
  },
  {
    id: "artifact",
    label: [
      "실행 artifact 참조",
      "Execution artifact reference",
      "مرجع مادة التنفيذ",
    ],
    required: true,
  },
  {
    id: "reason",
    label: [
      "검토 사유·미검증 범위",
      "Review reason and unverified scope",
      "سبب المراجعة والنطاق غير المتحقق",
    ],
    kind: "textarea",
    required: true,
  },
];
export default function ReleaseEvidence(p: Props) {
  const l = p.locale;
  return (
    <div className="ops-grid">
      <div>
        <Ledger
          locale={l}
          title={
            p.view === "requirements"
              ? [
                  "요구·인수 추적",
                  "Requirement and acceptance trace",
                  "تتبع المتطلبات والقبول",
                ]
              : p.view === "artifacts"
                ? ["실행 artifact", "Execution artifacts", "مواد التنفيذ"]
                : p.view === "load"
                  ? [
                      "부하 검증 결과",
                      "Load verification results",
                      "نتائج التحقق من الحمل",
                    ]
                  : p.view === "decisions"
                    ? [
                        "지원 판정 이력",
                        "Support decision history",
                        "سجل قرارات الدعم",
                      ]
                    : [
                        "지원 조합 행렬",
                        "Support combination matrix",
                        "مصفوفة مجموعات الدعم",
                      ]
          }
          columns={
            p.view === "matrix"
              ? [
                  ["지원 조합", "Support combination", "مجموعة الدعم"],
                  ["설계", "Design", "التصميم"],
                  ["구현", "Implementation", "التنفيذ"],
                  ["시험", "Test", "الاختبار"],
                  ["고객 실증", "Customer validation", "التحقق لدى العميل"],
                  ["확인·차단", "Observation / blockers", "الرصد والموانع"],
                ]
              : [
                  ["참조", "Reference", "المرجع"],
                  ["실행 환경", "Execution environment", "بيئة التنفيذ"],
                  ["확인 시각", "Observed at", "وقت الرصد"],
                  ["결과·유효성", "Result / validity", "النتيجة والصلاحية"],
                ]
          }
          reason={reason}
        />
        <Draft
          props={p}
          title={[
            "증거 연결 검토",
            "Review evidence linkage",
            "مراجعة ربط الأدلة",
          ]}
          fields={fields}
        />
      </div>
      <aside>
        <Evidence
          locale={l}
          title={["판정 근거", "Decision evidence", "أدلة القرار"]}
          fields={[
            [
              "공통·조건부 gate",
              "Common and conditional gates",
              "البوابات المشتركة والمشروطة",
            ],
            [
              "실제 입력·로그·결과",
              "Actual inputs, logs and outcomes",
              "المدخلات والسجلات والنتائج الفعلية",
            ],
            [
              "버전·조건 변경",
              "Version and condition changes",
              "تغييرات الإصدار والشروط",
            ],
          ]}
          reason={reason}
        />
        <Panel
          title={t(l, [
            "부하 검증 기준",
            "Load validation baseline",
            "أساس التحقق من الحمل",
          ])}
        >
          <p>
            {t(l, [
              "기획 목표: 일 100만 건 초과 유입, 월 3000만 건 이상 누적. 10만 건은 중간 검증 규모입니다. 이 화면에 연결된 측정 결과는 없습니다.",
              "Planning target: more than 1 million incoming records per day and at least 30 million accumulated per month. 100,000 records is an intermediate test size. No measurements are connected here.",
              "هدف التخطيط: أكثر من مليون سجل وارد يوميًا و30 مليون سجل متراكم على الأقل شهريًا. مئة ألف سجل حجم اختبار وسيط. لا توجد قياسات متصلة هنا.",
            ])}
          </p>
        </Panel>
        <Panel
          title={t(l, ["증거의 한계", "Evidence boundary", "حدود الأدلة"])}
        >
          <p>
            {t(l, [
              "모의 통과는 실연계 실증이 아닙니다. 이 프론트엔드의 빌드·화면 검증도 제품 전체의 지원 판정과 별도입니다.",
              "Mock success is separate from live integration evidence. Frontend build and rendering checks are separate from whole-product support decisions.",
              "نجاح المحاكاة منفصل عن أدلة التكامل الفعلي. فحوص بناء الواجهة وعرضها منفصلة عن قرارات دعم المنتج بالكامل.",
            ])}
          </p>
        </Panel>
      </aside>
    </div>
  );
}
