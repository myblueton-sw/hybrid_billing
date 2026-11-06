import { useState } from "react";
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
const methods: readonly (readonly [string, Words])[] = [
  ["local", ["Local", "로컬", "محلي"]],
  ["saml", ["SAML", "SAML", "SAML"]],
  ["oidc", ["OIDC", "OIDC", "OIDC"]],
  ["ldap", ["Direct LDAP over TLS", "직접 LDAP (TLS)", "LDAP مباشر عبر TLS"]],
];
export default function Authentication(props: Props) {
  const { locale: l, view } = props;
  const [method, setMethod] = useState("oidc");
  const providerRules = {
    method: required,
    workspace: reference,
    ...(method === "oidc"
      ? {
          issuer: { required: true, kind: "url" as const },
          subjectRef: reference,
          redirect: { required: true, kind: "url" as const },
          audience: required,
        }
      : method === "saml"
        ? { issuer: required, subjectRef: reference }
        : method === "ldap"
          ? {
              endpoint: { required: true, kind: "ldaps" as const },
              caRef: reference,
              searchScope: reference,
            }
          : {}),
  };
  return (
    <div className="security-screen">
      <Unconnected
        l={l}
        words={[
          "Provider trust, diagnostics and current policy are not loaded. Local checks do not test a connection, authenticate a principal or activate a trust change.",
          "제공자 신뢰·진단·현재 정책을 불러오지 않았습니다. 로컬 검사는 연결 시험·신원 인증·신뢰 변경 활성화를 수행하지 않습니다.",
          "لم تُحمّل ثقة المزود أو التشخيصات أو السياسة الحالية. الفحوص المحلية لا تختبر اتصالاً أو تصادق هوية أو تفعّل تغيير ثقة.",
        ]}
      />
      <div className="security-workbench">
        {view === "providers" ? (
          <section>
            <div
              className="security-methods"
              role="group"
              aria-label={say(l, [
                "Draft method",
                "초안 방식",
                "طريقة المسودة",
              ])}
            >
              {methods.map(([id, label]) => (
                <button
                  key={id}
                  type="button"
                  aria-pressed={method === id}
                  onClick={() => {
                    props.onDirty(false);
                    setMethod(id);
                  }}
                >
                  {say(l, label)}
                </button>
              ))}
            </div>
            <LocalForm
              key={method}
              variant={method}
              props={props}
              title={[
                "Prepare connection metadata",
                "연결 메타데이터 준비",
                "إعداد بيانات الاتصال الوصفية",
              ]}
              rules={providerRules}
              items={["data:trust-epoch", "data:allowed-methods"]}
            >
              <input type="hidden" name="method" value={method} />
              <Field name="workspace" l={l} />
              {["saml", "oidc"].includes(method) && (
                <>
                  <Field name="issuer" l={l} />
                  <Field name="subjectRef" l={l} />
                </>
              )}
              {method === "oidc" && (
                <>
                  <Field name="redirect" l={l} type="url" />
                  <Field name="audience" l={l} />
                </>
              )}
              {method === "ldap" && (
                <>
                  <Field name="endpoint" l={l} />
                  <Field name="caRef" l={l} />
                  <Field name="searchScope" l={l} />
                </>
              )}
            </LocalForm>
          </section>
        ) : view === "policies" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare policy review scope",
              "정책 검토 범위 준비",
              "إعداد نطاق مراجعة السياسة",
            ]}
            rules={{
              workspace: reference,
              method: required,
              sso: required,
              mfaRequirement: reference,
              mappingRef: reference,
              roleCeiling: reference,
              reason: required,
            }}
          >
            <Field name="workspace" l={l} />
            <Choice name="method" l={l} options={methods} />
            <Choice
              name="sso"
              l={l}
              options={[
                [
                  "enforced",
                  [
                    "Require SSO (proposal)",
                    "SSO 강제 (제안)",
                    "فرض الدخول الموحد (مقترح)",
                  ],
                ],
                [
                  "policy",
                  [
                    "Review existing policy",
                    "기존 정책 검토",
                    "مراجعة السياسة الحالية",
                  ],
                ],
              ]}
            />
            <Field name="mfaRequirement" l={l} />
            <Field name="mappingRef" l={l} />
            <Field name="roleCeiling" l={l} />
            <Field name="reason" l={l} />
          </LocalForm>
        ) : view === "test" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare a login-test scope",
              "시험 로그인 범위 준비",
              "إعداد نطاق اختبار الدخول",
            ]}
            rules={{ connectionRef: reference, principalRef: reference }}
          >
            <Field name="connectionRef" l={l} />
            <Field name="principalRef" l={l} />
          </LocalForm>
        ) : view === "keys" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare rotation references",
              "회전 참조 준비",
              "إعداد مراجع التدوير",
            ]}
            rules={{
              connectionRef: reference,
              credentialRef: reference,
              version: required,
              reason: required,
            }}
          >
            <Field name="connectionRef" l={l} />
            <Field name="credentialRef" l={l} />
            <Field name="version" l={l} />
            <Field name="reason" l={l} />
          </LocalForm>
        ) : view === "recovery" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare recovery evidence",
              "복구 근거 준비",
              "إعداد أدلة الاستعادة",
            ]}
            rules={{
              connectionRef: reference,
              evidenceRef: reference,
              reason: required,
            }}
          >
            <Field name="connectionRef" l={l} />
            <Field name="evidenceRef" l={l} />
            <Field name="reason" l={l} />
          </LocalForm>
        ) : (
          <LocalForm
            props={props}
            title={[
              "Prepare external-status review",
              "외부 상태 확인 검토 준비",
              "إعداد مراجعة الحالة الخارجية",
            ]}
            rules={{
              connectionRef: reference,
              verification: required,
              reason: required,
            }}
          >
            <Field name="connectionRef" l={l} />
            <Choice
              name="verification"
              l={l}
              options={[
                ["events", ["Events", "이벤트", "أحداث"]],
                ["sync", ["Synchronization", "동기화", "مزامنة"]],
                ["poll", ["Polling", "주기 조회", "استعلام دوري"]],
              ]}
            />
            <Field name="reason" l={l} />
          </LocalForm>
        )}
        <Observations
          l={l}
          title={
            view === "sync"
              ? [
                  "Status verification evidence",
                  "외부 상태 확인 근거",
                  "أدلة التحقق من الحالة",
                ]
              : view === "keys"
                ? [
                    "Certificate and key metadata",
                    "인증서·키 메타데이터",
                    "بيانات الشهادة والمفتاح الوصفية",
                  ]
                : view === "test"
                  ? [
                      "Identity, MFA and authority",
                      "신원·MFA·권한",
                      "الهوية والتحقق والصلاحية",
                    ]
                  : [
                      "Current trust and recovery",
                      "현재 신뢰·복구 상태",
                      "الثقة والاستعادة الحالية",
                    ]
          }
          fields={
            view === "sync"
              ? [
                  [
                    "Last attempt and last success",
                    "최근 시도·최근 성공",
                    "آخر محاولة وآخر نجاح",
                  ],
                  [
                    "Next check and maximum delay",
                    "다음 확인·허용 최대 지연",
                    "الفحص التالي وأقصى تأخير",
                  ],
                  [
                    "Failure reason and unknown-status hold",
                    "실패 사유·상태 미확인 보류",
                    "سبب الفشل وتعليق الحالة المجهولة",
                  ],
                ]
              : view === "keys"
                ? [
                    [
                      "Trusted origin and key reference",
                      "신뢰 출처·키 참조",
                      "المصدر الموثوق ومرجع المفتاح",
                    ],
                    [
                      "Expiry and rotation evidence",
                      "만료·회전 근거",
                      "أدلة الانتهاء والتدوير",
                    ],
                    [
                      "Session invalidation progress",
                      "세션 무효화 진행",
                      "تقدم إبطال الجلسات",
                    ],
                  ]
                : view === "test"
                  ? [
                      [
                        "Stable subject binding",
                        "안정 신원 연결",
                        "ربط الهوية الثابتة",
                      ],
                      [
                        "Verified MFA evidence",
                        "검증된 MFA 근거",
                        "أدلة التحقق الموثقة",
                      ],
                      [
                        "Business permission evaluation",
                        "업무 권한 평가",
                        "تقييم صلاحيات العمل",
                      ],
                    ]
                  : [
                      [
                        "Active revision and trust epoch",
                        "활성 개정·신뢰 epoch",
                        "الإصدار النشط وحقبة الثقة",
                      ],
                      [
                        "Independent proposer / approver / activator",
                        "제안자·독립 승인자·활성화 담당",
                        "المقترح والموافق المستقل والمنفذ",
                      ],
                      [
                        "Configuration and proof digests",
                        "설정·신원 근거 다이제스트",
                        "بصمات الإعداد والإثبات",
                      ],
                      [
                        "Recovery and session impact",
                        "복구·세션 영향",
                        "أثر الاستعادة والجلسات",
                      ],
                    ]
          }
        />
      </div>
      <Note
        l={l}
        words={[
          "A successful login or matching email cannot establish trust or business approval. Unknown external status keeps sensitive actions held. Activation needs current independent approval and a newer trust epoch; rollback is a new reviewed change.",
          "로그인 성공·같은 이메일은 신뢰나 업무 승인이 아닙니다. 외부 상태 미확인은 민감 행위를 계속 보류합니다. 활성화에는 현재 독립 승인과 새 신뢰 epoch가 필요하며 되돌리기도 새 변경 검토 대상입니다.",
          "نجاح الدخول أو تطابق البريد لا يثبت الثقة أو موافقة العمل. تبقى الإجراءات الحساسة معلقة عند جهل الحالة الخارجية. التفعيل يتطلب موافقة مستقلة حالية وحقبة ثقة أحدث؛ التراجع تغيير جديد يخضع للمراجعة.",
        ]}
      />
    </div>
  );
}
