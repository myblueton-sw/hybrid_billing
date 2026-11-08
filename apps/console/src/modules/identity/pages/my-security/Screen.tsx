import type { Props } from "../../screen-ui/types";
import {
  Field,
  Choice,
  LocalForm,
  Note,
  Observations,
  StaticState,
  Unconnected,
  Links,
} from "../../screen-ui/primitives";
import { required } from "../../screen-ui/model";
export default function MySecurity(props: Props) {
  const { locale: l, view } = props;
  return (
    <div className="security-screen security-self">
      <Unconnected
        l={l}
        words={[
          "Your signed-in identity, permitted self-service changes and personal preferences are not connected. No other user or organization-management data is shown.",
          "본인 인증 신원·허용된 본인 변경·개인 선호가 미연결입니다. 다른 사용자나 조직 관리 자료를 표시하지 않습니다.",
          "هويتك المسجلة والتغييرات الذاتية المسموحة والتفضيلات الشخصية غير متصلة. لا تُعرض بيانات مستخدم آخر أو إدارة المؤسسة.",
        ]}
      />
      <div className="security-workbench">
        {view === "account" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare personal display information",
              "개인 표시 정보 준비",
              "إعداد معلومات العرض الشخصية",
            ]}
            rules={{
              displayName: required,
              email: { required: true, kind: "email" },
            }}
            items={["data:self-identity", "data:self-membership"]}
          >
            <Field name="displayName" l={l} />
            <Field name="email" l={l} type="email" />
          </LocalForm>
        ) : view === "sessions" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare personal session scope",
              "본인 세션 조회 조건 준비",
              "إعداد نطاق الجلسات الشخصية",
            ]}
            rules={{ sessionKind: required }}
            items={["data:personal-sessions"]}
          >
            <Choice
              name="sessionKind"
              l={l}
              options={[
                ["current", ["This session", "현재 세션", "هذه الجلسة"]],
                [
                  "other",
                  [
                    "Other personal sessions",
                    "본인의 다른 세션",
                    "جلساتي الأخرى",
                  ],
                ],
              ]}
            />
          </LocalForm>
        ) : view === "notifications" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare notification preferences",
              "개인 알림 선호 준비",
              "إعداد تفضيلات الإشعارات",
            ]}
            rules={{
              timeZone: { required: true, kind: "zone" },
              quietStart: { required: true, kind: "time" },
              quietEnd: { required: true, kind: "time" },
              digest: required,
              severity: required,
              channel: required,
            }}
            items={["data:notification-policy", "data:verified-destinations"]}
          >
            <Field name="timeZone" l={l} />
            <Field name="quietStart" l={l} type="time" />
            <Field name="quietEnd" l={l} type="time" />
            <Choice
              name="digest"
              l={l}
              options={[
                [
                  "immediate",
                  ["Individual messages", "개별 알림", "رسائل فردية"],
                ],
                [
                  "summary",
                  ["Summary requested", "요약 알림 요청", "طلب ملخص"],
                ],
              ]}
            />
            <Choice
              name="severity"
              l={l}
              options={[
                [
                  "policy",
                  [
                    "Follow mandatory policy",
                    "필수 정책 따름",
                    "اتباع السياسة الإلزامية",
                  ],
                ],
                [
                  "review",
                  [
                    "Review optional reminders",
                    "선택 재알림 검토",
                    "مراجعة التذكيرات الاختيارية",
                  ],
                ],
              ]}
            />
            <Choice
              name="channel"
              l={l}
              options={[
                [
                  "in-app",
                  [
                    "In-app preference",
                    "서비스 내 알림 선호",
                    "تفضيل الإشعارات داخل التطبيق",
                  ],
                ],
                [
                  "email",
                  [
                    "Verified email preference",
                    "검증 이메일 알림 선호",
                    "تفضيل البريد الموثق",
                  ],
                ],
              ]}
            />
          </LocalForm>
        ) : (
          <section>
            <StaticState
              props={props}
              items={
                view === "methods"
                  ? ["data:linked-methods", "data:last-method-protection"]
                  : ["data:mfa-recovery", "data:reauthentication"]
              }
            />
            <Observations
              l={l}
              title={
                view === "methods"
                  ? [
                      "Connected login methods",
                      "연결 로그인 수단",
                      "طرق الدخول المرتبطة",
                    ]
                  : [
                      "MFA and recovery evidence",
                      "추가 인증·복구 근거",
                      "أدلة التحقق والاستعادة",
                    ]
              }
              fields={
                view === "methods"
                  ? [
                      [
                        "Provider and stable connection",
                        "제공자·안정 연결",
                        "المزود والاتصال الثابت",
                      ],
                      [
                        "Ownership verification and MFA",
                        "소유 증명·MFA",
                        "إثبات الملكية والتحقق",
                      ],
                      [
                        "Last use and remaining login methods",
                        "최근 사용·남는 로그인 수단",
                        "آخر استخدام وطرق الدخول المتبقية",
                      ],
                    ]
                  : [
                      [
                        "Current reauthentication evidence",
                        "현재 재인증 근거",
                        "دليل إعادة المصادقة الحالي",
                      ],
                      [
                        "Provider-managed MFA",
                        "제공자 관리 MFA",
                        "تحقق يديره المزود",
                      ],
                      [
                        "Product recovery readiness",
                        "제품 복구 준비 상태",
                        "جاهزية استعادة المنتج",
                      ],
                    ]
              }
            />
            <Links
              l={l}
              links={[
                [
                  "/recover",
                  [
                    "Open recovery guidance",
                    "복구 안내 열기",
                    "فتح إرشادات الاستعادة",
                  ],
                ],
              ]}
            />
          </section>
        )}
        <Observations
          l={l}
          title={
            view === "notifications"
              ? [
                  "Applied preferences and constraints",
                  "적용된 선호·정책 제한",
                  "التفضيلات المطبقة والقيود",
                ]
              : view === "sessions"
                ? [
                    "Current session evidence",
                    "현재 세션 근거",
                    "أدلة الجلسة الحالية",
                  ]
                : ["Your account boundary", "본인 계정 범위", "حدود حسابك"]
          }
          fields={
            view === "notifications"
              ? [
                  ["Verified destinations", "검증된 수신처", "الوجهات الموثقة"],
                  [
                    "Mandatory security-alert exceptions",
                    "필수 보안 알림 예외",
                    "استثناءات التنبيهات الأمنية الإلزامية",
                  ],
                  [
                    "Applied preference revision",
                    "적용된 선호 개정",
                    "إصدار التفضيلات المطبق",
                  ],
                  [
                    "Timezone and DST policy",
                    "시간대·DST 정책",
                    "سياسة المنطقة الزمنية والتوقيت الصيفي",
                  ],
                ]
              : view === "sessions"
                ? [
                    [
                      "Current device marker",
                      "현재 기기 표식",
                      "علامة الجهاز الحالي",
                    ],
                    [
                      "Issued, last used and expiry",
                      "발급·최근 사용·만료",
                      "الإصدار وآخر استخدام والانتهاء",
                    ],
                    [
                      "Revocation effect and cleanup",
                      "회수 효력·후속 정리",
                      "أثر الإلغاء والتنظيف",
                    ],
                    [
                      "Session count and snapshot",
                      "세션 건수·스냅샷",
                      "عدد الجلسات واللقطة",
                    ],
                  ]
                : [
                    [
                      "Immutable UserID",
                      "불변 사용자 ID",
                      "معرف المستخدم الثابت",
                    ],
                    [
                      "Current personal memberships",
                      "현재 본인 소속",
                      "عضوياتي الحالية",
                    ],
                    [
                      "Organization change policy",
                      "조직 변경 정책",
                      "سياسة التغيير بالمؤسسة",
                    ],
                  ]
          }
        />
      </div>
      <Note
        l={l}
        words={
          view === "notifications"
            ? [
                "These inputs are not saved or applied. Quiet hours do not change business deadlines or remove mandatory security alerts. No internal substitute-assignee selection is available in this self-service surface.",
                "이 입력은 저장·적용되지 않습니다. 조용한 시간은 업무 기한이나 필수 보안 알림을 없애지 않습니다. 이 본인 화면에는 내부 대체 담당자 선택이 없습니다.",
                "لا تُحفظ هذه المدخلات أو تُطبق. ساعات الهدوء لا تغير مواعيد العمل ولا تلغي التنبيهات الأمنية الإلزامية. لا يتيح هذا العرض الذاتي اختيار بديل داخلي.",
              ]
            : [
                "No password, method or session is changed. Matching email addresses never link identities automatically; sensitive changes need current reauthentication and protection of the last usable method.",
                "비밀번호·로그인 수단·세션을 변경하지 않습니다. 같은 이메일로 신원을 자동 연결하지 않으며 민감 변경에는 현재 재인증과 최후 수단 보호가 필요합니다.",
                "لا تتغير كلمة مرور أو طريقة دخول أو جلسة. تطابق البريد لا يربط الهويات تلقائياً؛ التغييرات الحساسة تحتاج إعادة مصادقة حالية وحماية آخر طريقة قابلة للاستخدام.",
              ]
        }
      />
    </div>
  );
}
