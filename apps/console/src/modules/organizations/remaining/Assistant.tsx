import { useState } from "react";
import { text, type Props } from "./contracts";
import {
  Boundary,
  EmptyList,
  Facts,
  Label,
  Panel,
  ReviewForm,
  refInput,
  useReview,
} from "./ui";
export default function Assistant(p: Props) {
  const l = p.locale;
  const [scope, setScope] = useState("");
  const [period, setPeriod] = useState("");
  const [currency, setCurrency] = useState("");
  const [request, setRequest] = useState("");
  useReview(p, !!(scope || period || currency || request));
  const clear = () => {
    setScope("");
    setPeriod("");
    setCurrency("");
    setRequest("");
  };
  const plan = (
    <>
      <Facts
        l={l}
        labels={[
          ["명령·고정 대상", "Command and fixed target", "الأمر والهدف الثابت"],
          [
            "입력 hash·정책 개정",
            "Input hash and policy revision",
            "بصمة المدخل ومراجعة السياسة",
          ],
          [
            "부작용·필수 승인",
            "Side effects and required approval",
            "الآثار والموافقة المطلوبة",
          ],
        ]}
      />
      <p>
        {text(l, [
          "검증된 서버 계획이 없습니다. 모델 문장이나 일반 동의는 승인 근거가 아닙니다.",
          "No validated server plan is available. Model text or general agreement is not approval evidence.",
          "لا تتوفر خطة خادم متحقق منها. نص النموذج أو الموافقة العامة ليسا دليل اعتماد.",
        ])}
      </p>
    </>
  );
  return (
    <>
      <Boundary l={l} />
      {p.view === "conversation" ? (
        <div className="work-columns assistant-columns">
          <Panel
            title={text(l, [
              "업무 질문 초안",
              "Business question draft",
              "مسودة سؤال الأعمال",
            ])}
            id="A01:field:request"
          >
            <ReviewForm l={l} id="A01:action:review-question" onClear={clear}>
              <div className="work-form-grid">
                <Label
                  label={text(l, [
                    "업무 범위 참조",
                    "Business scope reference",
                    "مرجع نطاق العمل",
                  ])}
                >
                  <input
                    {...refInput}
                    required
                    value={scope}
                    onChange={(e) => {
                      setScope(e.target.value);
                      setRequest("");
                    }}
                  />
                </Label>
                <Label
                  label={text(l, ["요청 기간", "Request month", "شهر الطلب"])}
                >
                  <input
                    type="month"
                    required
                    value={period}
                    onChange={(e) => {
                      setPeriod(e.target.value);
                      setRequest("");
                    }}
                  />
                </Label>
                <Label
                  label={text(l, [
                    "금액 통화",
                    "Amount currency",
                    "عملة المبلغ",
                  ])}
                >
                  <input
                    autoComplete="off"
                    pattern="[A-Z]{3}"
                    maxLength={3}
                    required
                    value={currency}
                    onChange={(e) => {
                      setCurrency(e.target.value);
                      setRequest("");
                    }}
                  />
                </Label>
              </div>
              <p className="work-context-message">
                {text(
                  l,
                  scope
                    ? [
                        "입력된 범위의 존재·현재 권한은 서버 검증이 필요합니다.",
                        "The entered scope still requires server existence and authority checks.",
                        "النطاق المدخل ما زال يتطلب التحقق من وجوده وصلاحيته على الخادم.",
                      ]
                    : [
                        "업무 대상이 정해지지 않았습니다. 범위를 먼저 입력하세요.",
                        "The business target is unspecified. Enter a scope first.",
                        "هدف العمل غير محدد. أدخل النطاق أولا.",
                      ],
                )}
              </p>
              <Label
                label={text(l, ["요청 내용", "Request text", "نص الطلب"])}
                hint={text(l, [
                  "비밀·자격 증명·불필요한 고객 원문은 입력하지 마세요. Enter는 줄바꿈입니다.",
                  "Do not enter secrets, credentials or unnecessary customer source bodies. Enter inserts a newline.",
                  "لا تدخل أسرارا أو بيانات اعتماد أو نصوص عميل غير ضرورية. يضيف Enter سطرا جديدا.",
                ])}
              >
                <textarea
                  required
                  rows={7}
                  maxLength={4000}
                  value={request}
                  onChange={(e) => setRequest(e.target.value)}
                />
              </Label>
              <p className="work-meta">
                <bdi dir="ltr">{request.length} / 4000</bdi>{" "}
                {text(l, [
                  "자 · 이 초안의 입력 한도",
                  "characters · this draft input limit",
                  "حرفا · حد إدخال هذه المسودة",
                ])}
              </p>
              <p>
                {text(l, [
                  "AI 서비스가 연결되지 않아 질문을 보내거나 응답을 생성하지 않습니다. 범위·기간·통화를 바꾸면 요청 초안이 지워집니다.",
                  "The AI service is unconnected; no question is sent or answer generated. Changing scope, period or currency clears the request draft.",
                  "خدمة الذكاء الاصطناعي غير متصلة؛ لا يرسل سؤال ولا يولد جواب. تغيير النطاق أو الفترة أو العملة يمسح مسودة الطلب.",
                ])}
              </p>
            </ReviewForm>
            <div
              className="work-conversation"
              role="log"
              aria-label={text(l, [
                "서버 대화 이력",
                "Server conversation history",
                "سجل محادثة الخادم",
              ])}
            >
              <p>
                {text(l, [
                  "서버 대화 이력을 받지 못했습니다.",
                  "Server conversation history is unavailable.",
                  "سجل محادثة الخادم غير متاح.",
                ])}
              </p>
            </div>
          </Panel>
          <Panel
            title={text(l, [
              "서버 계획과 출처",
              "Server plan and sources",
              "خطة الخادم والمصادر",
            ])}
          >
            {plan}
            <Facts
              l={l}
              labels={[
                [
                  "출처·기준시각",
                  "Sources and as-of time",
                  "المصادر والوقت المرجعي",
                ],
                [
                  "완전성·최신성",
                  "Completeness and freshness",
                  "الاكتمال والحداثة",
                ],
              ]}
            />
          </Panel>
        </div>
      ) : p.view === "plan" ? (
        <Panel
          title={text(l, [
            "서버 검증 실행 계획",
            "Server-validated execution plan",
            "خطة التنفيذ المتحقق منها على الخادم",
          ])}
          id="A01:data:plan"
        >
          {plan}
          <p>
            {text(l, [
              "첨부·도구 출력의 내용은 명령·도구·권한을 확대하지 않습니다. 실행은 실제 업무 명령과 사람 승인 경계를 따라야 합니다.",
              "Attachment and tool text cannot expand commands, tools or authority. Execution must follow actual business command and human approval boundaries.",
              "نصوص المرفقات والأدوات لا توسع الأوامر أو الأدوات أو الصلاحيات. يجب أن يتبع التنفيذ حدود أوامر الأعمال والموافقة البشرية.",
            ])}
          </p>
        </Panel>
      ) : p.view === "results" ? (
        <Panel
          title={text(l, [
            "접수된 작업과 결과",
            "Accepted jobs and results",
            "المهام المقبولة والنتائج",
          ])}
          id="A01:data:results"
        >
          <EmptyList
            l={l}
            columns={[
              ["작업 ID", "Job ID", "معرف المهمة"],
              ["명령·대상", "Command and target", "الأمر والهدف"],
              [
                "접수·작업 상태",
                "Acceptance and job state",
                "القبول وحالة المهمة",
              ],
              [
                "결과·다음 조치",
                "Result and next action",
                "النتيجة والإجراء التالي",
              ],
            ]}
            reason={[
              "실제 서버 작업 참조를 받지 못했습니다. 모델 응답 완료를 업무 완료로 표시하지 않습니다.",
              "No actual server job reference is available. Completion of a model response is not business completion.",
              "لا يتوفر مرجع مهمة فعلي من الخادم. اكتمال رد النموذج ليس اكتمال العمل.",
            ]}
          />
        </Panel>
      ) : (
        <Panel
          title={text(l, [
            "대화 보관과 작업 기록",
            "Conversation retention and job records",
            "الاحتفاظ بالمحادثة وسجلات المهام",
          ])}
          id="A01:data:retention"
        >
          <Facts
            l={l}
            labels={[
              [
                "대화 보관 정책",
                "Conversation retention policy",
                "سياسة الاحتفاظ بالمحادثة",
              ],
              ["법적 보류", "Legal hold", "الحجز القانوني"],
              ["삭제 요청 상태", "Deletion request state", "حالة طلب الحذف"],
            ]}
          />
          <p>
            {text(l, [
              "대화·첨부 삭제와 업무 작업·승인·감사 정본은 별개입니다. 보관·보류 근거가 연결되기 전에는 삭제 요청을 접수하지 않습니다.",
              "Conversation and attachment deletion is separate from canonical jobs, approvals and audit. No deletion request is accepted before retention and hold evidence is connected.",
              "حذف المحادثة والمرفقات مستقل عن المهام والموافقات والتدقيق الأصلية. لا يقبل طلب حذف قبل توصيل أدلة الاحتفاظ والحجز.",
            ])}
          </p>
        </Panel>
      )}
    </>
  );
}
