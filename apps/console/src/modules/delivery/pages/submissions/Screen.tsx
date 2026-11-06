import { useState } from "react";
import { fields, t, viewFields, type FieldKey, type Props } from "../../../../shared/domain-screen-ui/finance/contracts";
import {
  Criteria,
  EmptyTable,
  Evidence,
  Intro,
  Notice,
  Panel,
  PaymentEvidence,
  RecognitionEvidence,
  scopeInputs,
  useScreenState,
  type InputDef,
} from "../../../../shared/domain-screen-ui/finance/ui";
const inputs: readonly InputDef[] = [
  ...scopeInputs,
  {
    key: "erp",
    label: ["ERP 연결 참조", "ERP connection reference", "مرجع اتصال ERP"],
    required: true,
    idOnly: true,
  },
  {
    key: "version",
    label: [
      "제품·버전 참조",
      "Product and version reference",
      "مرجع المنتج والإصدار",
    ],
    idOnly: true,
  },
  {
    key: "outcome",
    label: [
      "외부 결과 기준",
      "External-result criterion",
      "معيار النتيجة الخارجية",
    ],
    options: [
      {
        value: "all-permitted",
        label: [
          "허용 범위 전체",
          "All permitted results",
          "جميع النتائج المسموح بها",
        ],
      },
      {
        value: "unknown",
        label: ["결과 불명", "Result unknown", "النتيجة غير معروفة"],
      },
      { value: "failed", label: ["실패", "Failed", "فشل"] },
    ],
  },
];
export default function Submissions(props: Props) {
  const { locale: l, view, onDirty } = props;
  const [basis, setBasis] = useState<"attributed" | "whole">("attributed"),
    [measure, setMeasure] = useState<FieldKey>("platformAmount");
  const [, onReview] = useScreenState(props, "I02", { basis, measure });
  return (
    <div
      className="fin-screen fin-submissions"
      dir={l === "ar" ? "rtl" : "ltr"}
      data-finance-screen="I02"
      data-view={view}
    >
      <Intro
        l={l}
        purpose={[
          "플랫폼 명세와 ERP 문서의 귀속 행을 대사하고, 제출·초안·실제 전기를 별도로 추적합니다.",
          "Reconcile platform-attributed ERP rows and track submission, draft and actual posting separately.",
          "طابق صفوف ERP المنسوبة للمنصة وتتبع الإرسال والمسودة والترحيل الفعلي بشكل منفصل.",
        ]}
      />
      <Criteria
        l={l}
        screenId="I02"
        inputs={inputs}
        onDirty={onDirty}
        onReview={onReview}
      />
      {view === "payment-terms" ? (
        <PaymentEvidence l={l} id="I02" />
      ) : view === "recognition" ? (
        <RecognitionEvidence l={l} id="I02" />
      ) : view === "comparison" ? (
        <>
          <Panel
            title={t(l, [
              "금액·수량 비교 기준",
              "Amount and quantity comparison basis",
              "أساس مقارنة المبالغ والكميات",
            ])}
          >
            <div className="fin-inline-controls">
              <label>
                {t(l, ["확인 범위", "Inspection scope", "نطاق الفحص"])}
                <select
                  value={basis}
                  onChange={(e) => setBasis(e.target.value as typeof basis)}
                  data-item="I02:input:comparison-basis"
                >
                  <option value="attributed">
                    {t(l, [
                      "플랫폼 귀속분",
                      "Platform-attributed only",
                      "الجزء المنسوب للمنصة فقط",
                    ])}
                  </option>
                  <option value="whole">
                    {t(l, [
                      "전체 ERP 문서 (별도 확인)",
                      "Whole ERP document (separate inspection)",
                      "مستند ERP بالكامل (فحص منفصل)",
                    ])}
                  </option>
                </select>
              </label>
              {basis === "attributed" && (
                <label>
                  {t(l, ["비교 항목", "Measure", "المقياس"])}
                  <select
                    value={measure}
                    onChange={(e) => setMeasure(e.target.value as FieldKey)}
                    data-item="I02:input:comparison-measure"
                  >
                    {(["platformAmount", "tax", "quantity"] as FieldKey[]).map(
                      (key) => (
                        <option key={key} value={key}>
                          {t(l, fields[key].label)}
                        </option>
                      ),
                    )}
                  </select>
                </label>
              )}
            </div>
            <Notice
              l={l}
              text={
                basis === "attributed"
                  ? [
                      "비관련 ERP 상품은 플랫폼 귀속 합계에 포함하지 않습니다. 문서 전체 금액과 귀속 금액을 더하거나 서로 대체하지 않습니다.",
                      "Unrelated ERP products are excluded from platform-attributed totals. Whole-document and attributed amounts are never added or substituted.",
                      "تُستبعد منتجات ERP غير المرتبطة من إجماليات المنصة. لا تُجمع المبالغ الإجمالية والمنسوبة ولا يُستبدل أحدها بالآخر.",
                    ]
                  : [
                      "문서 전체 확인은 플랫폼 귀속액 대사를 대신하지 않습니다. 승인된 귀속 행·수량·세금 근거가 별도로 필요합니다.",
                      "Whole-document inspection does not replace attributed reconciliation. Approved attributed rows, quantities and tax evidence are required separately.",
                      "فحص المستند الكامل لا يحل محل مطابقة الجزء المنسوب. أدلة الصفوف والكميات والضرائب المعتمدة مطلوبة بشكل منفصل.",
                    ]
              }
            />
          </Panel>
          {basis === "whole" ? (
            <Panel
              title={t(l, [
                "전체 ERP 문서 금액 · 별도 문맥",
                "Whole ERP document amount: separate context",
                "مبلغ مستند ERP بالكامل: سياق منفصل",
              ])}
            >
              <Evidence
                l={l}
                screenId="I02"
                keys={["externalRevision", "wholeAmount"]}
              />
              <Notice
                l={l}
                text={[
                  "이 금액은 비관련 ERP 상품을 포함할 수 있는 문서 전체 금액입니다. 플랫폼 귀속 순액·세금·수량과 비교하거나 대사 차이로 표시하지 않습니다.",
                  "This is the whole-document amount and may include unrelated ERP products. It is not compared with platform-attributed net amount, tax or quantity and is not presented as a reconciliation difference.",
                  "هذا مبلغ المستند الكامل وقد يتضمن منتجات ERP غير مرتبطة. لا يُقارن بصافي المبلغ أو الضريبة أو الكمية المنسوبة للمنصة ولا يُعرض كفرق مطابقة.",
                ]}
              />
            </Panel>
          ) : (
            <div className="fin-comparison">
              <Panel
                title={t(l, [
                  "승인된 플랫폼 명세",
                  "Approved platform statement",
                  "كشف المنصة المعتمد",
                ])}
              >
                <Evidence
                  l={l}
                  screenId="I02"
                  keys={["hash", measure, "approval"]}
                />
              </Panel>
              <Panel
                title={t(l, [
                  "ERP 초안의 플랫폼 귀속분",
                  "Platform-attributed ERP draft",
                  "الجزء المنسوب للمنصة من مسودة ERP",
                ])}
              >
                <Evidence
                  l={l}
                  screenId="I02"
                  keys={["externalRevision", measure]}
                />
              </Panel>
              <Panel
                title={t(l, [
                  "대사 차이",
                  "Reconciliation difference",
                  "فرق المطابقة",
                ])}
              >
                <p>{t(l, fields[measure].label)}</p>
                <p>
                  {t(
                    l,
                    measure === "quantity"
                      ? [
                          "수량 차이·단위 미확인",
                          "Quantity difference and unit unknown",
                          "فرق الكمية والوحدة غير معروفين",
                        ]
                      : [
                          "금액 차이·통화 미확인",
                          "Amount difference and currency unknown",
                          "فرق المبلغ والعملة غير معروفين",
                        ],
                  )}
                </p>
                <p className="fin-meta">
                  {t(l, [
                    "비교 가능한 근거가 없어 계산하지 않습니다.",
                    "No calculation without comparable evidence.",
                    "لا يُجرى حساب دون أدلة قابلة للمقارنة.",
                  ])}
                </p>
              </Panel>
            </div>
          )}
        </>
      ) : view === "submissions" ? (
        <Panel
          title={t(l, [
            "ERP 제출 원장",
            "ERP submission ledger",
            "سجل الإرسال إلى ERP",
          ])}
        >
          <EmptyTable
            l={l}
            screenId="I02"
            columns={viewFields.I02.submissions}
            caption={[
              "제출·전기·귀속액",
              "Submissions, posting and attributed amounts",
              "الإرسال والترحيل والمبالغ المنسوبة",
            ]}
          />
          <Notice
            l={l}
            text={[
              "결과 불명은 실패와 다릅니다. 외부 상태를 확인하기 전에는 재제출하거나 새 문서를 만들지 않습니다.",
              "An unknown result is different from failure. Do not resubmit or create a new document before verifying the external state.",
              "النتيجة غير المعروفة تختلف عن الفشل. لا تُعد الإرسال ولا تنشئ مستنداً جديداً قبل التحقق من الحالة الخارجية.",
            ]}
          />
        </Panel>
      ) : view === "rows" ? (
        <div className="fin-split">
          <Panel
            title={t(l, [
              "제출과 문서·행 연결",
              "Submission-to-document and row links",
              "روابط الإرسال بالمستند والصفوف",
            ])}
          >
            <EmptyTable
              l={l}
              screenId="I02"
              columns={[
                "submission",
                "rowIds",
                "external",
                "platformAmount",
                "quantity",
              ]}
              caption={[
                "귀속 행 연결",
                "Attributed row links",
                "روابط الصفوف المنسوبة",
              ]}
            />
          </Panel>
          <Panel
            title={t(l, [
              "기존 거래·중복 보호",
              "Existing transactions and duplicate protection",
              "المعاملات القائمة والحماية من التكرار",
            ])}
          >
            <Evidence
              l={l}
              screenId="I02"
              keys={["order", "wholeAmount", "approval"]}
            />
            <p>
              {t(l, [
                "기존 주문·선불·매입 문서 반영 여부가 미확인이면 새 제출의 적격성도 미확인입니다.",
                "If existing orders, prepayments or payable documents are unverified, new-submission eligibility is also unverified.",
                "إذا لم تُتحقق الطلبات أو الدفعات المقدمة أو مستندات الدائنين القائمة، فأهلية الإرسال الجديد غير متحققة أيضاً.",
              ])}
            </p>
          </Panel>
        </div>
      ) : view === "events" ? (
        <Panel
          title={t(l, [
            "외부 사건과 주기 조회",
            "External events and periodic checks",
            "الأحداث الخارجية والتحقق الدوري",
          ])}
        >
          <Evidence l={l} screenId="I02" keys={viewFields.I02.events} />
          <Notice
            l={l}
            text={[
              "웹훅 수신 순서와 객체 개정은 별도입니다. 사건 접수만으로 전기·발행 완료를 판정하지 않습니다.",
              "Webhook receipt order and object revision are separate. Event acceptance does not establish posting or issuance completion.",
              "ترتيب استلام الأحداث ومراجعة الكائن أمران منفصلان. قبول الحدث لا يثبت اكتمال الترحيل أو الإصدار.",
            ]}
          />
        </Panel>
      ) : (
        <div className="fin-two">
          <Panel
            title={t(l, [
              "지원 조합·자동 전달",
              "Supported combination and automatic delivery",
              "التوافق المدعوم والتسليم التلقائي",
            ])}
          >
            <Evidence
              l={l}
              screenId="I02"
              keys={["erp", "autoDelivery", "posted"]}
            />
            <Notice
              l={l}
              text={[
                "제품·버전·법인별 검증 근거가 없습니다. 자동 첨부·발송 통제가 확인되지 않으면 자동 전달을 개통하지 않습니다.",
                "Product, version and entity-specific evidence is unavailable. Automatic delivery cannot be enabled without proven attachment and sending controls.",
                "أدلة المنتج والإصدار والكيان غير متاحة. لا يُفعّل التسليم التلقائي دون إثبات ضبط المرفقات والإرسال.",
              ]}
            />
          </Panel>
          <Panel
            title={t(l, [
              "연결·조회 통제",
              "Integration and polling controls",
              "ضوابط التكامل والتحقق",
            ])}
          >
            <Evidence l={l} screenId="I02" keys={["lastPoll", "nextPoll"]} />
            <p>
              {t(l, [
                "검증된 API 또는 승인된 파일 경로만 사용합니다. ERP 데이터베이스 직접 쓰기 경로를 제공하지 않습니다.",
                "Only verified APIs or approved file routes are permitted. No direct ERP database write path is provided.",
                "تُستخدم واجهات متحققة أو مسارات ملفات معتمدة فقط. لا يوجد مسار للكتابة المباشرة إلى قاعدة بيانات ERP.",
              ])}
            </p>
          </Panel>
        </div>
      )}
      <details className="fin-requirements">
        <summary>
          {t(l, [
            "제출·전기·정정 실행 요건",
            "Submission, posting and correction requirements",
            "متطلبات الإرسال والترحيل والتصحيح",
          ])}
        </summary>
        <p>
          {t(l, [
            "초안 제출은 기존 거래 대사·고정 승인·지원 조합이 필요합니다. 전기·정정·월별 인식 전송은 해당 법인·장부의 별도 실행 권한, 현재 외부 개정, 최종액 재승인과 멱등 키가 필요합니다. 외부 상태 조회와 수납·지급 기한 대사도 미연결이며 현재 실행하지 않습니다.",
            "Draft submission requires existing-transaction reconciliation, fixed approval and a supported combination. Posting, correction and monthly recognition require separate entity and ledger authority, current external revision, final-amount reapproval and idempotency. External checks and receipt/due-date reconciliation are also unconnected and are not executed.",
            "إرسال المسودة يتطلب مطابقة المعاملات وموافقة ثابتة وتوافقاً مدعوماً. الترحيل والتصحيح والاعتراف الشهري تتطلب صلاحية مستقلة للكيان والدفتر ومراجعة خارجية حديثة وإعادة اعتماد المبلغ ومفتاح منع التكرار. التحقق الخارجي ومطابقة التحصيل والاستحقاق غير متصلين ولا يُنفذان.",
          ])}
        </p>
      </details>
    </div>
  );
}
