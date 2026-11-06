import type { Props, Words } from "./types";
import { say } from "./types";
import {
  Field,
  Choice,
  LocalForm,
  Note,
  Observations,
  StaticState,
  Unconnected,
  Links,
} from "./primitives";
import { required, reference } from "./model";
const authBoundary: Words = [
  "Identity and membership services are not connected. No account lookup, sign-in, registration or recovery request is sent.",
  "인증·가입 정보 서비스가 연결되지 않았습니다. 계정 조회·로그인·등록·복구 요청을 보내지 않습니다.",
  "خدمات الهوية والعضوية غير متصلة. لا يُرسل بحث عن حساب أو طلب دخول أو تسجيل أو استعادة.",
];
export default function Entry(props: Props) {
  const { screenId, view, locale: l } = props;
  const login = screenId === "L01",
    activate = screenId === "L02";
  const intro: Words = login
    ? [
        "Your organization’s workspace",
        "소속 조직의 업무 공간",
        "مساحة عمل مؤسستك",
      ]
    : activate
      ? [
          "Join the intended workspace",
          "지정된 업무 공간에 가입",
          "الانضمام لمساحة العمل المقصودة",
        ]
      : [
          "Recover through the right provider",
          "인증 제공자에 맞는 복구",
          "الاستعادة عبر مزود الهوية الصحيح",
        ];
  const purpose: Words = login
    ? [
        "Use the workspace provided by your administrator. An email address alone does not grant membership.",
        "관리자에게 안내받은 업무 공간을 사용하세요. 이메일 주소만으로 가입 권한이 생기지 않습니다.",
        "استخدم مساحة العمل التي وفرها المسؤول. البريد الإلكتروني وحده لا يمنح العضوية.",
      ]
    : activate
      ? [
          "Registration type, workspace and role are fixed by a verified invitation or installation context. You cannot select administrator access here.",
          "등록 종류·업무 공간·역할은 검증된 초대 또는 설치 문맥으로 정해집니다. 여기서 관리자 권한을 선택할 수 없습니다.",
          "يحدد سياق الدعوة أو التثبيت الموثق نوع التسجيل ومساحة العمل والدور. لا يمكنك اختيار صلاحية المسؤول هنا.",
        ]
      : [
          "Provider-managed passwords stay with the provider. An outage does not enable another login method or emergency access.",
          "제공자 관리 비밀번호는 해당 제공자에서 복구합니다. 장애가 다른 로그인 방식이나 긴급 접근을 허용하지 않습니다.",
          "تُستعاد كلمات مرور المزود لديه. التعطل لا يتيح طريقة دخول بديلة أو وصولاً طارئاً.",
        ];
  let body;
  if (login && view === "method")
    body = (
      <>
        <LocalForm
          props={props}
          title={[
            "Prepare workspace access",
            "접속할 업무 공간 준비",
            "إعداد الوصول لمساحة العمل",
          ]}
          rules={{ workspace: reference, loginId: required }}
        >
          <Field name="workspace" l={l} />
          <Field name="loginId" l={l} />
        </LocalForm>
        <Note
          l={l}
          words={[
            "Allowed login methods, SSO enforcement and session policy have not been loaded. Password and provider controls appear only after verified policy is connected.",
            "허용 로그인 방식·SSO 강제·세션 정책을 불러오지 않았습니다. 비밀번호·제공자 입력은 검증된 정책 연결 후에만 제공됩니다.",
            "لم تُحمّل طرق الدخول المسموحة أو فرض الدخول الموحد أو سياسة الجلسة. تظهر حقول كلمة المرور والمزود بعد اتصال سياسة موثقة فقط.",
          ]}
        />
      </>
    );
  else if (login && view === "mfa")
    body = (
      <>
        <LocalForm
          props={props}
          title={[
            "Check verification help topic",
            "추가 인증 도움 항목 확인",
            "تحديد موضوع مساعدة التحقق",
          ]}
          rules={{ issue: required }}
        >
          <Choice
            name="issue"
            l={l}
            options={[
              [
                "missing",
                [
                  "No verification request",
                  "인증 요청 없음",
                  "لا يوجد طلب تحقق",
                ],
              ],
              [
                "provider",
                [
                  "Provider help needed",
                  "제공자 도움 필요",
                  "مساعدة المزود مطلوبة",
                ],
              ],
            ]}
          />
        </LocalForm>
        <Observations
          l={l}
          title={[
            "Current verification step",
            "현재 추가 인증 단계",
            "خطوة التحقق الحالية",
          ]}
          fields={[
            ["Verified challenge", "검증된 인증 요청", "طلب تحقق موثق"],
            [
              "Provider authentication strength",
              "제공자 인증 강도",
              "قوة مصادقة المزود",
            ],
            ["Expiry policy", "만료 정책", "سياسة الانتهاء"],
          ]}
        />
      </>
    );
  else if (activate && view === "invitation")
    body = (
      <>
        <LocalForm
          props={props}
          title={[
            "Check contact format",
            "연락 정보 형식 확인",
            "فحص تنسيق معلومات الاتصال",
          ]}
          rules={{ email: { required: true, kind: "email" } }}
        >
          <Field name="email" type="email" l={l} />
        </LocalForm>
        <Observations
          l={l}
          title={["Invitation context", "초대 문맥", "سياق الدعوة"]}
          fields={[
            [
              "Verified registration type",
              "검증된 등록 종류",
              "نوع التسجيل الموثق",
            ],
            [
              "Bound workspace and role",
              "고정 업무 공간·역할",
              "مساحة العمل والدور المحددان",
            ],
            ["Inviter and expiry", "초대한 주체·만료", "المرسل والانتهاء"],
          ]}
        />
      </>
    );
  else if (activate && view === "identity")
    body = (
      <>
        <LocalForm
          props={props}
          title={[
            "Prepare display information",
            "표시 정보 준비",
            "إعداد معلومات العرض",
          ]}
          rules={{ displayName: required }}
        >
          <Field name="displayName" l={l} />
        </LocalForm>
        <Note
          l={l}
          words={[
            "A display name does not prove identity. No verified invitation, allowed authentication method or MFA evidence is connected. Password setup and token consumption are unavailable.",
            "표시 이름은 신원 증명이 아닙니다. 검증된 초대·허용 인증 방식·MFA 근거가 연결되지 않아 비밀번호 설정이나 토큰 소비를 수행하지 않습니다.",
            "اسم العرض لا يثبت الهوية. لا توجد دعوة موثقة أو طريقة مصادقة مسموحة أو أدلة تحقق متصلة. إعداد كلمة المرور واستهلاك الرمز غير متاحين.",
          ]}
        />
      </>
    );
  else if (!login && !activate && view === "route")
    body = (
      <LocalForm
        props={props}
        title={[
          "Prepare recovery guidance",
          "복구 안내 준비",
          "إعداد إرشادات الاستعادة",
        ]}
        rules={{ workspace: reference, loginId: required, routeKind: required }}
      >
        <Field name="workspace" l={l} />
        <Field name="loginId" l={l} />
        <Choice
          name="routeKind"
          l={l}
          options={[
            [
              "provider",
              [
                "Provider-managed account guidance",
                "제공자 관리 계정 안내",
                "إرشادات حساب يديره المزود",
              ],
            ],
            [
              "local",
              [
                "Local-account policy guidance",
                "로컬 계정 정책 안내",
                "إرشادات سياسة الحساب المحلي",
              ],
            ],
            [
              "mfa",
              [
                "Additional verification guidance",
                "추가 인증 안내",
                "إرشادات التحقق الإضافي",
              ],
            ],
          ]}
        />
      </LocalForm>
    );
  else if (!login && !activate && view === "proof")
    body = (
      <>
        <LocalForm
          props={props}
          title={[
            "Prepare proof-step help",
            "소유 증명 도움 준비",
            "إعداد مساعدة خطوة إثبات الملكية",
          ]}
          rules={{ issue: required }}
        >
          <Choice
            name="issue"
            l={l}
            options={[
              [
                "missing",
                [
                  "No active recovery challenge",
                  "진행 중 복구 요청 없음",
                  "لا يوجد طلب استعادة نشط",
                ],
              ],
              [
                "provider",
                [
                  "Provider-owned recovery",
                  "제공자 관리 복구",
                  "استعادة يديرها المزود",
                ],
              ],
            ]}
          />
        </LocalForm>
        <Note
          l={l}
          words={[
            "No recovery challenge is connected. Do not enter recovery codes; proof cannot be checked locally.",
            "연결된 복구 인증 요청이 없습니다. 복구 코드를 입력하지 마세요. 로컬에서는 소유 증명을 검증할 수 없습니다.",
            "لا يوجد طلب تحقق للاستعادة متصل. لا تُدخل رموز الاستعادة؛ لا يمكن التحقق من الإثبات محلياً.",
          ]}
        />
      </>
    );
  else
    body = (
      <>
        <StaticState
          props={props}
          items={["data:verified-context", "data:result"]}
        />
        <Observations
          l={l}
          title={
            activate
              ? ["Registration result", "등록 결과", "نتيجة التسجيل"]
              : view === "replacement"
                ? [
                    "New authentication method",
                    "새 인증 수단",
                    "طريقة مصادقة جديدة",
                  ]
                : ["Recovery result", "복구 결과", "نتيجة الاستعادة"]
          }
          fields={
            activate
              ? [
                  ["Identity verification", "신원 확인", "التحقق من الهوية"],
                  ["Invitation acceptance", "초대 수락", "قبول الدعوة"],
                  ["Membership result", "가입 결과", "نتيجة العضوية"],
                ]
              : [
                  [
                    "Ownership verification",
                    "소유 증명 검증",
                    "التحقق من الملكية",
                  ],
                  [
                    "Approved replacement method",
                    "허용된 새 수단",
                    "الطريقة البديلة المعتمدة",
                  ],
                  [
                    "Session revocation evidence",
                    "세션 회수 근거",
                    "دليل إلغاء الجلسات",
                  ],
                ]
          }
        />
        <Note
          l={l}
          words={[
            "No completed result is available. A local field check cannot advance or complete this security process.",
            "완료 결과가 없습니다. 입력 형식 확인만으로 보안 절차가 진행되거나 완료되지 않습니다.",
            "لا توجد نتيجة مكتملة. فحص الحقول محلياً لا يقدّم هذه العملية الأمنية أو يكملها.",
          ]}
        />
      </>
    );
  return (
    <div className="security-entry entry-welcome">
      <section className="entry-intro">
        <h2>{say(l, intro)}</h2>
        <p>{say(l, purpose)}</p>
        <Unconnected l={l} words={authBoundary} />
        <Links
          l={l}
          links={[
            ["/login", ["Sign in", "로그인", "تسجيل الدخول"]],
            ["/activate", ["Invitation", "초대 확인", "الدعوة"]],
            [
              "/recover",
              ["Recovery guidance", "복구 안내", "إرشادات الاستعادة"],
            ],
          ]}
        />
      </section>
      <section className="entry-access security-entry-access">{body}</section>
    </div>
  );
}
