import { useState } from "react";
import { text, type Props, type Words } from "../../../../shared/domain-screen-ui/work/contracts";
import {
  Boundary,
  EmptyList,
  Facts,
  Label,
  Panel,
  ReviewForm,
  refInput,
  useReview,
} from "../../../../shared/domain-screen-ui/work/ui";
const publicFields = {
  document: ["공개 문서", "Public document", "المستند العام"],
  inquiry: ["내 문의", "My inquiry", "استفساري"],
  read: ["읽음 상태", "Read state", "حالة القراءة"],
  delivery: ["전달 상태", "Delivery state", "حالة التسليم"],
} satisfies Record<string, Words>;
export default function Inbox(p: Props) {
  const l = p.locale,
    admin = p.audience === "admin";
  const [reference, setReference] = useState("");
  const [state, setState] = useState("");
  const [period, setPeriod] = useState("");
  useReview(p, !!(reference || state || period));
  const jobStates = [
    "queued",
    "running",
    "waiting",
    "succeeded",
    "failed",
    "cancelled",
  ];
  const titles: Record<string, Words> = {
    tasks: ["내 실행 작업", "My execution jobs", "مهام التنفيذ الخاصة بي"],
    notifications: ["업무 알림", "Business notifications", "إشعارات الأعمال"],
    detail: [
      "작업·사건 상세",
      "Job and incident detail",
      "تفاصيل المهمة والحدث",
    ],
    delivery: [
      "전달과 읽음 상태",
      "Delivery and read state",
      "حالة التسليم والقراءة",
    ],
    "payment-terms": [
      "지급 기한과 수납 근거",
      "Payment deadlines and receipt evidence",
      "مواعيد الدفع وأدلة التحصيل",
    ],
  };
  return (
    <>
      <Boundary l={l} />
      <div className="work-columns inbox-columns">
        <Panel
          title={text(
            l,
            admin
              ? (titles[p.view] ?? titles.tasks)
              : [
                  "내 문서·문의 알림",
                  "My document and inquiry notifications",
                  "إشعارات مستنداتي واستفساراتي",
                ],
          )}
          id={`U03:data:${admin ? "admin" : "customer"}-${p.view}`}
        >
          <ReviewForm
            l={l}
            id="U03:action:review-filter"
            onClear={() => {
              setReference("");
              setState("");
              setPeriod("");
            }}
          >
            <div className="work-form-grid">
              <Label
                label={text(
                  l,
                  admin
                    ? [
                        "작업·사건 참조",
                        "Job or incident reference",
                        "مرجع المهمة أو الحدث",
                      ]
                    : [
                        "공개 문서·문의 참조",
                        "Public document or inquiry reference",
                        "مرجع المستند العام أو الاستفسار",
                      ],
                )}
              >
                <input
                  {...refInput}
                  value={reference}
                  onChange={(e) => setReference(e.target.value)}
                />
              </Label>
              <Label
                label={text(l, ["조회 월", "Review month", "شهر المراجعة"])}
              >
                <input
                  type="month"
                  value={period}
                  onChange={(e) => setPeriod(e.target.value)}
                />
              </Label>
              <Label
                label={text(
                  l,
                  admin && p.view === "tasks"
                    ? [
                        "작업 상태 필터 초안",
                        "Job-state filter draft",
                        "مسودة مرشح حالة المهمة",
                      ]
                    : [
                        "읽음 필터 초안",
                        "Read-state filter draft",
                        "مسودة مرشح القراءة",
                      ],
                )}
              >
                <select
                  value={state}
                  onChange={(e) => setState(e.target.value)}
                >
                  <option value="">
                    {text(l, ["전체 상태", "All states", "جميع الحالات"])}
                  </option>
                  {admin && p.view === "tasks"
                    ? jobStates.map((s) => <option key={s}>{s}</option>)
                    : [
                        ["unread", "읽지 않음", "Unread", "غير مقروء"],
                        ["read", "읽음", "Read", "مقروء"],
                      ].map(([v, ko, en, ar]) => (
                        <option key={v} value={v}>
                          {text(l, [ko, en, ar])}
                        </option>
                      ))}
                </select>
              </Label>
            </div>
          </ReviewForm>
          <EmptyList
            l={l}
            columns={
              admin
                ? [
                    ["종류·대상", "Kind and target", "النوع والهدف"],
                    [
                      "작업·사건 상태",
                      "Job or incident state",
                      "حالة المهمة أو الحدث",
                    ],
                    ["기한", "Deadline", "الموعد"],
                    ["다음 조치", "Next action", "الإجراء التالي"],
                  ]
                : [
                    publicFields.document,
                    publicFields.inquiry,
                    publicFields.read,
                    publicFields.delivery,
                  ]
            }
            reason={[
              "허용된 작업·알림 페이지를 아직 받지 못했습니다.",
              "A permitted job or notification page has not been received.",
              "لم يتم استلام صفحة مهام أو إشعارات مسموح بها.",
            ]}
          />
        </Panel>
        <Panel
          title={text(l, [
            "상태와 다음 조치",
            "State and next action",
            "الحالة والإجراء التالي",
          ])}
        >
          {!admin ? (
            <>
              <Facts
                l={l}
                labels={[
                  publicFields.document,
                  publicFields.inquiry,
                  publicFields.read,
                  publicFields.delivery,
                ]}
              />
              <p>
                {text(l, [
                  "고객 공개 문서와 문의 알림만 표시하는 범위입니다.",
                  "This view is restricted to customer-visible documents and inquiry notifications.",
                  "يقتصر هذا العرض على المستندات المرئية للعميل وإشعارات الاستفسارات.",
                ])}
              </p>
            </>
          ) : p.view === "delivery" ? (
            <>
              <Facts
                l={l}
                labels={[
                  ["채널 수락", "Channel acceptance", "قبول القناة"],
                  ["전달 결과", "Delivery outcome", "نتيجة التسليم"],
                  ["사용자 읽음", "User read", "قراءة المستخدم"],
                  ["본인 확인", "Acknowledgement", "الإقرار"],
                ]}
              />
              <p>
                {text(l, [
                  "채널 수락은 사용자의 읽음·확인이나 문제 해결을 의미하지 않습니다.",
                  "Channel acceptance does not establish user reading, acknowledgement or resolution.",
                  "قبول القناة لا يثبت قراءة المستخدم أو إقراره أو حل المشكلة.",
                ])}
              </p>
            </>
          ) : p.view === "payment-terms" ? (
            <>
              <Facts
                l={l}
                labels={[
                  [
                    "원래·현재 지급 기한",
                    "Original and current due date",
                    "الاستحقاق الأصلي والحالي",
                  ],
                  [
                    "잔액·통화",
                    "Outstanding balance and currency",
                    "الرصيد المستحق والعملة",
                  ],
                  [
                    "수납 기준시각·최신성",
                    "Receipt as-of and freshness",
                    "مرجع التحصيل وحداثته",
                  ],
                  ["독촉 보류", "Collection hold", "تعليق المطالبة"],
                ]}
              />
              <p>
                {text(l, [
                  "수납 권위와 최신성이 없으면 연체·완납을 확정하지 않습니다. 독촉 보류는 잔액과 별개입니다.",
                  "Overdue or paid status is unconfirmed without current receipt authority. Collection hold is separate from balance.",
                  "لا يتم تأكيد التأخر أو السداد دون مرجع تحصيل حديث وموثوق. تعليق المطالبة مستقل عن الرصيد.",
                ])}
              </p>
            </>
          ) : p.view === "notifications" ? (
            <Facts
              l={l}
              labels={[
                ["읽음", "Read", "القراءة"],
                ["확인", "Acknowledged", "الإقرار"],
                ["담당 수락", "Assignment accepted", "قبول المسؤولية"],
                ["문제 해결", "Resolved", "الحل"],
                [
                  "최초·최근 발생",
                  "First and latest occurrence",
                  "أول وآخر حدوث",
                ],
              ]}
            />
          ) : (
            <>
              <Facts
                l={l}
                labels={[
                  ["작업 상태", "Job state", "حالة المهمة"],
                  ["대기 사유", "Waiting reason", "سبب الانتظار"],
                  ["대상 manifest", "Target manifest", "بيان الأهداف"],
                  [
                    "성공·보류·실패·미처리",
                    "Succeeded, held, failed, unprocessed",
                    "ناجح ومعلق وفاشل وغير معالج",
                  ],
                  ["checkpoint", "Checkpoint", "نقطة الاستئناف"],
                ]}
              />
              <p>
                {text(l, [
                  "외부 결과가 불명인 작업은 재실행 성공으로 추정하지 않습니다. 현재 상태와 재시도 권한을 확인할 서비스가 필요합니다.",
                  "An unknown external outcome is not a successful replay. A service must verify current state and retry authority.",
                  "النتيجة الخارجية المجهولة ليست إعادة تنفيذ ناجحة. يجب أن تتحقق الخدمة من الحالة والصلاحية.",
                ])}
              </p>
            </>
          )}
        </Panel>
      </div>
    </>
  );
}
