import { Draft, Evidence, Ledger, Panel, Steps } from "../../../../shared/domain-screen-ui/operations/controls";
import { t, type Props, type Field, type Words } from "../../../../shared/domain-screen-ui/operations/types";
const scope: Field = {
  id: "scope",
  label: ["설치 범위", "Installation scope", "نطاق التثبيت"],
  required: true,
};
const reference: Field = {
  id: "reference",
  label: [
    "설정 revision 참조",
    "Configuration revision reference",
    "مرجع مراجعة الإعدادات",
  ],
  required: true,
};
const reason: Words = [
  "설치 inventory와 검사 결과가 연결되면 현재 설정과 영향 범위를 확인할 수 있습니다.",
  "Connect the installation inventory and diagnostic results to inspect configuration and impact.",
  "اربط سجل التثبيت ونتائج التشخيص لفحص الإعدادات ونطاق التأثير.",
];
export default function Installation(p: Props) {
  const l = p.locale;
  return (
    <div className="ops-grid">
      <div>
        {p.view === "plan" ? (
          <>
            <Steps
              locale={l}
              steps={[
                ["환경 확인", "Inspect environment", "فحص البيئة"],
                ["변경 영향 검토", "Review changes", "مراجعة التغييرات"],
                ["진단 근거 확인", "Inspect diagnostics", "فحص التشخيص"],
                ["개통 판정", "Activation decision", "قرار التفعيل"],
              ]}
            />
            <Draft
              props={p}
              title={[
                "설치 변경 계획",
                "Installation change plan",
                "خطة تغيير التثبيت",
              ]}
              fields={[
                scope,
                {
                  id: "version",
                  label: ["제품 버전", "Product version", "إصدار المنتج"],
                  required: true,
                },
                {
                  id: "profile",
                  label: [
                    "설치 profile 참조",
                    "Installation profile reference",
                    "مرجع ملف التثبيت",
                  ],
                  required: true,
                },
                {
                  id: "changes",
                  label: [
                    "서비스·포트·볼륨 변경",
                    "Service, port and volume changes",
                    "تغييرات الخدمات والمنافذ ووحدات التخزين",
                  ],
                  kind: "textarea",
                  required: true,
                  hint: [
                    "기존 볼륨·포트·권한 영향을 적으세요. 비밀값을 입력하지 마세요.",
                    "Describe existing volume, port and permission impacts. Do not enter secrets.",
                    "صف التأثير على وحدات التخزين والمنافذ والصلاحيات الحالية. لا تدخل أسرارًا.",
                  ],
                },
              ]}
            />
          </>
        ) : p.view === "diagnostics" ? (
          <>
            <Draft
              props={p}
              title={[
                "진단 범위 검토",
                "Review diagnostic scope",
                "مراجعة نطاق التشخيص",
              ]}
              fields={[
                scope,
                reference,
                {
                  id: "destination",
                  label: [
                    "허용 진단 목적지 참조",
                    "Allowed diagnostic destination reference",
                    "مرجع وجهة التشخيص المسموحة",
                  ],
                  required: true,
                },
              ]}
            />
            <Ledger
              locale={l}
              title={["환경 검사", "Environment checks", "فحوص البيئة"]}
              columns={[
                ["검사", "Check", "الفحص"],
                ["대상", "Target", "الهدف"],
                ["필수·경고", "Required / warning", "مطلوب / تحذير"],
                ["근거", "Evidence", "الدليل"],
              ]}
              reason={reason}
            />
          </>
        ) : p.view === "support" ? (
          <Draft
            props={p}
            title={[
              "지원 자료 포함 범위",
              "Support material scope",
              "نطاق مواد الدعم",
            ]}
            fields={[
              scope,
              reference,
              {
                id: "include",
                label: [
                  "포함할 진단 자료",
                  "Diagnostic material to include",
                  "مواد التشخيص المطلوب تضمينها",
                ],
                kind: "textarea",
                required: true,
              },
              {
                id: "exclude",
                label: [
                  "제외·민감정보 제거 기준",
                  "Exclusions and redaction criteria",
                  "معايير الاستبعاد وإخفاء البيانات الحساسة",
                ],
                kind: "textarea",
                required: true,
              },
            ]}
            notice={[
              "자료 목록만 검토합니다. 묶음 생성·외부 업로드는 수행하지 않습니다.",
              "Review a material list only. No package generation or external upload.",
              "مراجعة قائمة المواد فقط. لا يتم إنشاء حزمة أو رفع خارجي.",
            ]}
          />
        ) : (
          <Draft
            props={p}
            title={[
              "제한 지원 접근 계획",
              "Limited support access plan",
              "خطة وصول الدعم المحدود",
            ]}
            fields={[
              scope,
              {
                id: "actor",
                label: [
                  "실제 지원 담당자 참조",
                  "Actual support actor reference",
                  "مرجع مسؤول الدعم الفعلي",
                ],
                required: true,
              },
              {
                id: "reason",
                label: [
                  "지원 사유·허용 작업",
                  "Reason and permitted work",
                  "السبب والعمل المسموح",
                ],
                kind: "textarea",
                required: true,
              },
              {
                id: "expires",
                label: [
                  "요청 만료 시각",
                  "Requested expiry",
                  "وقت الانتهاء المطلوب",
                ],
                kind: "datetime-local",
                required: true,
              },
            ]}
          />
        )}
      </div>
      <aside>
        <Evidence
          locale={l}
          title={[
            "현재 설치 근거",
            "Current installation evidence",
            "أدلة التثبيت الحالية",
          ]}
          fields={[
            [
              "설치·설정 hash",
              "Installation / configuration hash",
              "التثبيت وبصمة الإعدادات",
            ],
            [
              "CA·시각·용량 검사",
              "CA, clock and capacity checks",
              "فحوص الشهادات والوقت والسعة",
            ],
            ["진단 유효기간", "Diagnostic validity", "صلاحية التشخيص"],
            [
              "추가 자원·비용",
              "Additional resources and cost",
              "الموارد والتكلفة الإضافية",
            ],
          ]}
          reason={reason}
        />
        <Panel
          title={t(l, [
            "설치 보호",
            "Installation protection",
            "حماية التثبيت",
          ])}
        >
          <p>
            {t(l, [
              "기존 데이터 영향과 필수 검사 결과를 확인한 후 변경합니다. 인증서 검증을 해제하거나 포괄 관리자 권한을 부여하지 않습니다.",
              "Changes require existing-data impact and mandatory checks. Certificate verification stays enabled; broad administrator grants are not a remedy.",
              "تتطلب التغييرات فحص تأثيرها على البيانات الحالية والفحوص الإلزامية. يبقى التحقق من الشهادات مفعّلًا ولا تُمنح صلاحيات إدارية شاملة.",
            ])}
          </p>
        </Panel>
      </aside>
    </div>
  );
}
