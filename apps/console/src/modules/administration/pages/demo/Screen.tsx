import { useEffect, useState } from "react";
import { Draft, Panel } from "../../../../shared/domain-screen-ui/operations/controls";
import { t, type Props, type Words } from "../../../../shared/domain-screen-ui/operations/types";
const scenarios: readonly Words[] = [
  ["첫 청구 흐름", "First billing flow", "مسار الفوترة الأولى"],
  [
    "중복 수집·재시도",
    "Duplicate collection and retry",
    "التجميع المكرر وإعادة المحاولة",
  ],
  [
    "약정·부분 사용",
    "Commitments and partial use",
    "الالتزامات والاستخدام الجزئي",
  ],
  [
    "대량 처리·부분 실패",
    "Bulk processing and partial failure",
    "المعالجة المجمعة والفشل الجزئي",
  ],
  [
    "권한 회수·접근 실패",
    "Revoked authority and denied access",
    "إلغاء الصلاحية ورفض الوصول",
  ],
];
export default function Demo(p: Props) {
  const l = p.locale;
  const [scenario, setScenario] = useState(0);
  useEffect(
    () =>
      p.onState(
        JSON.stringify({
          view: p.view,
          scenario,
          engine: "unconnected",
          operational: false,
        }),
        ["P06:field:scenario", "P06:state:isolated-learning"],
      ),
    [p.view, scenario, p.onState],
  );
  return (
    <>
      <div className="ops-band">
        <strong>
          {t(l, [
            "격리 학습 화면",
            "Isolated learning surface",
            "واجهة تعلم معزولة",
          ])}
        </strong>
        <p>
          {t(l, [
            "모의 실행 엔진과 DEMO 환경이 연결되지 않았습니다. 운영 DB·키·문서번호에 접근하지 않습니다.",
            "The mock engine and DEMO environment are not connected. No access to operational databases, keys or document numbers.",
            "محرك المحاكاة وبيئة DEMO غير متصلين. لا يوجد وصول إلى قواعد بيانات التشغيل أو المفاتيح أو أرقام المستندات.",
          ])}
        </p>
      </div>
      <div className="ops-grid">
        <div>
          <Panel
            title={t(l, [
              "학습할 업무 흐름",
              "Workflow to explore",
              "مسار العمل المراد استكشافه",
            ])}
          >
            <fieldset className="ops-radio">
              <legend>
                {t(l, ["시나리오 선택", "Choose scenario", "اختيار السيناريو"])}
              </legend>
              {scenarios.map((s, i) => (
                <label key={i}>
                  <input
                    type="radio"
                    name="demo-scenario"
                    checked={scenario === i}
                    onChange={() => setScenario(i)}
                  />
                  <span>{t(l, s)}</span>
                </label>
              ))}
            </fieldset>
          </Panel>
          {p.view !== "results" && (
            <Draft
              props={p}
              key={scenario + ":" + p.view}
              title={[
                "실행 계획 입력 검토",
                "Review execution-plan inputs",
                "مراجعة مدخلات خطة التنفيذ",
              ]}
              fields={[
                {
                  id: "fixture",
                  label: [
                    "공개·합성 fixture 버전 참조",
                    "Public or synthetic fixture version reference",
                    "مرجع إصدار بيانات الاختبار العامة أو التجريبية",
                  ],
                  required: true,
                },
                {
                  id: "seed",
                  label: [
                    "재현 seed · 정수",
                    "Reproduction seed · integer",
                    "قيمة إعادة الإنتاج · عدد صحيح",
                  ],
                  required: true,
                  pattern: /^-?\d{1,12}$/,
                },
                {
                  id: "clock",
                  label: ["가상 시계", "Virtual clock", "الساعة الافتراضية"],
                  kind: "datetime-local",
                  required: true,
                },
                ...(p.view === "failures"
                  ? [
                      {
                        id: "fault",
                        label: [
                          "모의 장애 유형",
                          "Mock fault type",
                          "نوع العطل التجريبي",
                        ] as Words,
                        required: true,
                        options: [
                          ["시간 초과", "Timeout", "انتهاء المهلة"],
                          ["중복 사건", "Duplicate event", "حدث مكرر"],
                          ["권한 회수", "Authority revoked", "إلغاء الصلاحية"],
                        ] as Words[],
                      },
                    ]
                  : []),
              ]}
            />
          )}
        </div>
        <aside>
          <Panel
            title={t(l, [
              "기대값과 실제 결과",
              "Expected and actual outcomes",
              "النتائج المتوقعة والفعلية",
            ])}
          >
            <dl className="ops-facts">
              <div>
                <dt>
                  {t(l, [
                    "기대값 버전",
                    "Expected-result version",
                    "إصدار النتائج المتوقعة",
                  ])}
                </dt>
                <dd>{t(l, ["미연결", "Not connected", "غير متصل"])}</dd>
              </div>
              <div>
                <dt>
                  {t(l, [
                    "모의 실행 결과",
                    "Mock execution outcome",
                    "نتيجة تنفيذ المحاكاة",
                  ])}
                </dt>
                <dd>{t(l, ["실행되지 않음", "Not executed", "لم يُنفّذ"])}</dd>
              </div>
            </dl>
            <p>
              {t(l, [
                "공개 부분월 FOCUS 자료는 인보이스 정답이 아닙니다. 현재 선택은 학습 범위이며 실행 결과를 만들지 않습니다.",
                "Public partial-month FOCUS data is not invoice ground truth. This selection defines learning scope and generates no execution result.",
                "بيانات FOCUS العامة لجزء من الشهر ليست مرجعًا نهائيًا للفواتير. يحدد هذا الاختيار نطاق التعلم ولا ينتج نتيجة تنفيذ.",
              ])}
            </p>
          </Panel>
        </aside>
      </div>
    </>
  );
}
