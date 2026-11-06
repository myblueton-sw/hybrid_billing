import type { Props, Words } from "../../screen-ui/types";
import { say } from "../../screen-ui/types";
import {
  Field,
  Choice,
  LocalForm,
  Note,
  Observations,
  Unconnected,
} from "../../screen-ui/primitives";
import { reference, required } from "../../screen-ui/model";
const actions: readonly (readonly [string, Words])[] = [
  ["view", ["Read evidence", "근거 조회", "قراءة الأدلة"]],
  [
    "raw-export",
    ["Original-data export", "원본 반출", "تصدير البيانات الأصلية"],
  ],
  ["calculate", ["Calculation", "계산", "حساب"]],
  ["approve", ["Human approval", "사람 승인", "موافقة بشرية"]],
  ["issue", ["Issuance", "발행", "إصدار"]],
];
function Targets({ l }: { l: Props["locale"] }) {
  return (
    <>
      <Field name="entityRef" l={l} />
      <Field name="customerRef" l={l} />
      <Field name="contractRef" l={l} />
    </>
  );
}
export default function Access(props: Props) {
  const { view, locale: l } = props;
  const targetRules = {
    entityRef: reference,
    customerRef: reference,
    contractRef: reference,
  };
  return (
    <div className="security-screen">
      <Unconnected
        l={l}
        words={[
          "Role definitions, delegation chains and access decisions are not connected. Prepared references never grant access or select the actual execution identity.",
          "역할 정의·위임 사슬·접근 결정이 미연결입니다. 준비한 참조는 권한을 부여하거나 실제 실행자를 지정하지 않습니다.",
          "تعريفات الأدوار وسلاسل التفويض وقرارات الوصول غير متصلة. المراجع المعدة لا تمنح صلاحية أو تحدد هوية التنفيذ الفعلية.",
        ]}
      />
      <div className="security-workbench">
        {view === "roles" ? (
          <LocalForm
            props={props}
            title={[
              "Review a role boundary",
              "역할 범위 준비",
              "إعداد حدود الدور",
            ]}
            rules={{ roleRef: reference, action: required, ...targetRules }}
            items={["data:role-source"]}
          >
            <Field name="roleRef" l={l} />
            <Choice name="action" l={l} options={actions} />
            <Targets l={l} />
          </LocalForm>
        ) : view === "grants" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare delegation review",
              "위임 검토 준비",
              "إعداد مراجعة التفويض",
            ]}
            rules={{
              subjectRef: reference,
              subjectKind: required,
              action: required,
              ...targetRules,
              parentRef: reference,
              redelegation: required,
              starts: { required: true, kind: "date" },
              expires: { required: true, kind: "date" },
            }}
            items={["data:parent-ceiling", "data:delegation-validity"]}
          >
            <Field name="subjectRef" l={l} />
            <Choice
              name="subjectKind"
              l={l}
              options={[
                ["person", ["Person", "사람", "شخص"]],
                ["service", ["Service", "서비스", "خدمة"]],
              ]}
            />
            <Choice name="action" l={l} options={actions} />
            <Targets l={l} />
            <Field name="parentRef" l={l} />
            <Choice
              name="redelegation"
              l={l}
              options={[
                [
                  "review",
                  [
                    "Review parent permission",
                    "원위임 허용 검토",
                    "مراجعة إذن التفويض الأصلي",
                  ],
                ],
                [
                  "none",
                  [
                    "No onward delegation proposed",
                    "재위임 제안 없음",
                    "لا يُقترح تفويض لاحق",
                  ],
                ],
              ]}
            />
            <Field name="starts" type="date" l={l} />
            <Field name="expires" type="date" l={l} />
          </LocalForm>
        ) : view === "effects" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare an effect comparison",
              "효력 비교 조건 준비",
              "إعداد مقارنة الأثر",
            ]}
            rules={{ subjectRef: reference, action: required, ...targetRules }}
            items={["data:effect-matrix"]}
          >
            <Field name="subjectRef" l={l} />
            <Choice name="action" l={l} options={actions} />
            <Targets l={l} />
          </LocalForm>
        ) : (
          <LocalForm
            props={props}
            title={[
              "Prepare limited support review",
              "제한 지원 검토 준비",
              "إعداد مراجعة الدعم المحدود",
            ]}
            rules={{
              supportRef: reference,
              scope: reference,
              reason: required,
              expires: { required: true, kind: "date" },
            }}
            items={["data:actual-actor", "data:support-boundary"]}
          >
            <Field name="supportRef" l={l} />
            <Field name="scope" l={l} />
            <Field name="reason" l={l} />
            <Field name="expires" type="date" l={l} />
          </LocalForm>
        )}
        {view === "effects" ? (
          <section className="security-observations">
            <h2>
              {say(l, [
                "Effective permission evidence",
                "실효 권한 근거",
                "أدلة الصلاحيات الفعلية",
              ])}
            </h2>
            <div className="security-table-scroll">
              <table>
                <thead>
                  <tr>
                    <th>{say(l, ["Action", "행위", "الإجراء"])}</th>
                    <th>{say(l, ["Decision", "결정", "القرار"])}</th>
                    <th>{say(l, ["Source", "출처", "المصدر"])}</th>
                  </tr>
                </thead>
                <tbody>
                  {actions.map(([id, label]) => (
                    <tr key={id}>
                      <th scope="row">{say(l, label)}</th>
                      <td>
                        {say(l, [
                          "Not evaluated",
                          "평가하지 않음",
                          "لم يُقيّم",
                        ])}
                      </td>
                      <td>{say(l, ["Not connected", "미연결", "غير متصل"])}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        ) : (
          <Observations
            l={l}
            title={
              view === "support"
                ? [
                    "Support identity and boundaries",
                    "지원 신원·범위",
                    "هوية الدعم وحدوده",
                  ]
                : [
                    "Source and delegation limits",
                    "권한 출처·위임 상한",
                    "مصدر الصلاحية وحدود التفويض",
                  ]
            }
            fields={
              view === "support"
                ? [
                    [
                      "Eligible support principals",
                      "적격 지원자",
                      "هويات الدعم المؤهلة",
                    ],
                    [
                      "Actual requester and executor",
                      "실제 요청자·실행자",
                      "طالب التنفيذ والمنفذ الفعليان",
                    ],
                    [
                      "Current independent approval and expiry",
                      "현재 독립 승인·만료",
                      "الموافقة المستقلة الحالية والانتهاء",
                    ],
                  ]
                : [
                    [
                      "Current grant and source",
                      "현재 부여·출처",
                      "المنح الحالي ومصدره",
                    ],
                    [
                      "Parent scope and validity ceiling",
                      "원위임 범위·유효기간 상한",
                      "حدود نطاق التفويض الأصلي ومدته",
                    ],
                    [
                      "User and client intersection",
                      "사용자·client 교집합",
                      "تقاطع المستخدم والعميل",
                    ],
                    [
                      "Deny revision and cleanup evidence",
                      "차단 개정·후속 정리 근거",
                      "إصدار المنع وأدلة التنظيف",
                    ],
                  ]
            }
          />
        )}
      </div>
      <Note
        l={l}
        words={[
          "A support request does not impersonate the customer or confer human approval. Revocation is unconfirmed until the independent decision is durably committed; later cleanup is a separate outcome.",
          "지원 요청은 고객을 사칭하거나 사람 승인 권한을 부여하지 않습니다. 독립 결정이 내구 확정되기 전 회수는 미확정이며 후속 정리는 별도 결과입니다.",
          "طلب الدعم لا ينتحل هوية العميل ولا يمنح صلاحية موافقة بشرية. يظل الإلغاء غير مؤكد حتى تثبيت القرار المستقل؛ والتنظيف اللاحق نتيجة منفصلة.",
        ]}
      />
    </div>
  );
}
