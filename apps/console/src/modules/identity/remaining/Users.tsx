import type { Props } from "./types";
import {
  Field,
  LocalForm,
  Note,
  Observations,
  Unconnected,
} from "./primitives";
import { reference, required } from "./model";
export default function Users(props: Props) {
  const { locale: l, view } = props;
  return (
    <div className="security-screen">
      <Unconnected
        l={l}
        words={[
          "User, identity and membership records are unavailable. No invitation, role assignment or revocation is sent from this preparation view.",
          "사용자·신원·소속 자료가 미연결입니다. 이 준비 화면에서는 초대·역할 부여·회수를 요청하지 않습니다.",
          "سجلات المستخدمين والهويات والعضوية غير متاحة. لا يُرسل من هذا العرض أي طلب دعوة أو منح دور أو إلغاء.",
        ]}
      />
      <div className="security-workbench">
        {view === "users" ? (
          <LocalForm
            props={props}
            title={[
              "Prepare user lookup",
              "사용자 조회 조건 준비",
              "إعداد البحث عن مستخدم",
            ]}
            rules={{ search: { required: true, max: 128 } }}
            items={["data:identity-membership", "data:server-search"]}
          >
            <Field name="search" l={l} />
          </LocalForm>
        ) : view === "invitations" ? (
          <LocalForm
            props={props}
            title={["Prepare an invitation", "초대 내용 준비", "إعداد دعوة"]}
            rules={{
              email: { required: true, kind: "email" },
              workspace: reference,
              roleRef: reference,
              expires: { required: true, kind: "date" },
            }}
            items={["data:invitation-status"]}
          >
            <Field name="email" l={l} type="email" />
            <Field name="workspace" l={l} />
            <Field name="roleRef" l={l} />
            <Field name="expires" l={l} type="date" />
          </LocalForm>
        ) : view === "membership" ? (
          <LocalForm
            props={props}
            title={[
              "Review membership scope",
              "소속·역할 범위 준비",
              "إعداد نطاق العضوية والدور",
            ]}
            rules={{
              userRef: reference,
              workspace: reference,
              roleRef: reference,
              starts: { required: true, kind: "date" },
              expires: { required: true, kind: "date" },
            }}
            items={["data:role-validity"]}
          >
            <Field name="userRef" l={l} />
            <Field name="workspace" l={l} />
            <Field name="roleRef" l={l} />
            <Field name="starts" l={l} type="date" />
            <Field name="expires" l={l} type="date" />
          </LocalForm>
        ) : (
          <LocalForm
            props={props}
            title={[
              "Prepare revocation scope",
              "회수 범위 준비",
              "إعداد نطاق الإلغاء",
            ]}
            rules={{ targetRef: reference, scope: reference, reason: required }}
            items={["data:revocation-effect", "data:cleanup"]}
          >
            <Field name="targetRef" l={l} />
            <Field name="scope" l={l} />
            <Field name="reason" l={l} />
          </LocalForm>
        )}
        <Observations
          l={l}
          title={
            view === "revocation"
              ? [
                  "Effect before cleanup",
                  "차단 효력과 후속 정리",
                  "أثر المنع قبل التنظيف",
                ]
              : view === "invitations"
                ? ["Invitation evidence", "초대 근거", "أدلة الدعوة"]
                : view === "membership"
                  ? ["Membership evidence", "소속·역할 근거", "أدلة العضوية"]
                  : [
                      "User and identity evidence",
                      "사용자·신원 근거",
                      "أدلة المستخدم والهوية",
                    ]
          }
          fields={
            view === "revocation"
              ? [
                  [
                    "Durable deny revision",
                    "내구 차단 개정",
                    "إصدار المنع المثبت",
                  ],
                  [
                    "Confirmed effect time",
                    "효력 확정 시각",
                    "وقت تأكيد الأثر",
                  ],
                  [
                    "Sessions, keys and cache cleanup",
                    "세션·키·캐시 후속 정리",
                    "تنظيف الجلسات والمفاتيح والذاكرة المؤقتة",
                  ],
                ]
              : view === "invitations"
                ? [
                    [
                      "Target and workspace binding",
                      "대상·업무 공간 고정",
                      "ربط الهدف ومساحة العمل",
                    ],
                    [
                      "Role ceiling and expiry",
                      "역할 상한·만료",
                      "حد الدور والانتهاء",
                    ],
                    ["Acceptance state", "수락 상태", "حالة القبول"],
                  ]
                : view === "membership"
                  ? [
                      [
                        "Immutable UserID",
                        "불변 사용자 ID",
                        "معرف المستخدم الثابت",
                      ],
                      [
                        "Membership status and period",
                        "소속 상태·기간",
                        "حالة العضوية ومدتها",
                      ],
                      [
                        "Effective role source",
                        "역할 효력의 출처",
                        "مصدر الدور الفعلي",
                      ],
                    ]
                  : [
                      [
                        "Immutable UserID",
                        "불변 사용자 ID",
                        "معرف المستخدم الثابت",
                      ],
                      [
                        "Login realm and identity status",
                        "로그인 realm·신원 상태",
                        "نطاق الدخول وحالة الهوية",
                      ],
                      [
                        "Membership and user suspension",
                        "소속·사용자 정지 상태",
                        "العضوية وتعليق المستخدم",
                      ],
                      [
                        "Record count and snapshot",
                        "건수·스냅샷",
                        "عدد السجلات واللقطة",
                      ],
                    ]
          }
        />
      </div>
      <Note
        l={l}
        words={
          view === "revocation"
            ? [
                "Request receipt, confirmed security denial and later cleanup are separate. A whole-user suspension differs from ending one membership; previously delivered copies cannot be recalled by this screen.",
                "요청 접수·보안 차단 확정·후속 정리는 별개입니다. 전체 사용자 정지와 한 소속 종료를 구분하며 이미 전달된 사본을 회수했다고 표시하지 않습니다.",
                "استلام الطلب وتأكيد المنع الأمني والتنظيف اللاحق حالات منفصلة. تعليق المستخدم بالكامل يختلف عن إنهاء عضوية واحدة؛ لا يمكن لهذا العرض استرجاع النسخ المسلّمة سابقاً.",
              ]
            : [
                "Names and email addresses do not merge identities. Role ceilings, last-administrator protection and permitted targets require current server evidence.",
                "이름·이메일만으로 신원을 병합하지 않습니다. 역할 상한·최후 관리자 보호·허용 대상은 현재 서버 근거가 필요합니다.",
                "الأسماء والبريد الإلكتروني لا تدمج الهويات. حدود الأدوار وحماية آخر مسؤول والأهداف المسموحة تحتاج أدلة حالية من الخادم.",
              ]
        }
      />
    </div>
  );
}
