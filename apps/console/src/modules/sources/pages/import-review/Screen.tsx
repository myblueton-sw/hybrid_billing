import { useState } from "react";
import { text, type Props } from "../../../../shared/domain-screen-ui/work/contracts";
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
export default function Imports(p: Props) {
  const l = p.locale;
  const [connection, setConnection] = useState("");
  const [period, setPeriod] = useState("");
  const [receipt, setReceipt] = useState("");
  const [correction, setCorrection] = useState("");
  const [reason, setReason] = useState("");
  useReview(p, !!(connection || period || receipt || correction || reason));
  const clear = () => {
    setConnection("");
    setPeriod("");
    setReceipt("");
    setCorrection("");
    setReason("");
  };
  return (
    <>
      <Boundary l={l} />
      <div
        className="work-import-trail"
        aria-label={text(l, [
          "수집 입력 단계",
          "Input intake stages",
          "مراحل استقبال المدخلات",
        ])}
      >
        {[
          ["수신", "Received", "مستلم"],
          ["정규화 후보", "Normalized candidate", "مرشح مطبع"],
          ["검증", "Validation", "التحقق"],
          [
            "작업자 확인 대기",
            "Awaiting worker confirmation",
            "بانتظار تأكيد المسؤول",
          ],
          ["적격 입력", "Eligible input", "مدخل مؤهل"],
        ].map((w) => (
          <span key={w[1]}>{text(l, w as [string, string, string])}</span>
        ))}
        <strong>
          {text(l, [
            "현재 단계 미확인",
            "Current stage unknown",
            "المرحلة الحالية غير معروفة",
          ])}
        </strong>
      </div>
      {p.view === "imports" ? (
        <div className="work-columns import-columns">
          <Panel
            title={text(l, [
              "수집 접수와 처리 상태",
              "Import receipts and processing state",
              "إيصالات الاستيراد وحالة المعالجة",
            ])}
            id="S03:data:imports"
          >
            <EmptyList
              l={l}
              columns={[
                ["수집 접수 ID", "Import receipt ID", "معرف إيصال الاستيراد"],
                ["원천·기간", "Source and period", "المصدر والفترة"],
                ["행·바이트", "Rows and bytes", "الصفوف والبايتات"],
                [
                  "revision·작업 상태",
                  "Revision and job state",
                  "المراجعة وحالة المهمة",
                ],
              ]}
              reason={[
                "허용된 수집 접수 목록을 아직 받지 못했습니다.",
                "The permitted import receipt list has not been received.",
                "لم يتم استلام قائمة إيصالات الاستيراد المسموح بها.",
              ]}
            />
            <Facts
              l={l}
              labels={[
                [
                  "원본 hash·revision",
                  "Original hash and revision",
                  "بصمة الأصل ومراجعته",
                ],
                [
                  "원천·매핑·프로파일 개정",
                  "Source, mapping and profile revisions",
                  "مراجعات المصدر والربط والملف التعريفي",
                ],
              ]}
            />
          </Panel>
          <Panel
            title={text(l, [
              "접수 범위 검토 초안",
              "Receipt scope review draft",
              "مسودة مراجعة نطاق الإيصال",
            ])}
          >
            <ReviewForm
              l={l}
              id="S03:action:review-receipt-scope"
              onClear={clear}
            >
              <Label
                label={text(l, [
                  "원천 연결 참조",
                  "Source connection reference",
                  "مرجع اتصال المصدر",
                ])}
              >
                <input
                  {...refInput}
                  required
                  value={connection}
                  onChange={(e) => setConnection(e.target.value)}
                />
              </Label>
              <Label
                label={text(l, ["자료 기간", "Source month", "شهر المصدر"])}
              >
                <input
                  required
                  type="month"
                  value={period}
                  onChange={(e) => setPeriod(e.target.value)}
                />
              </Label>
              <Label
                label={text(l, [
                  "정정 대상 참조 · 해당 시",
                  "Correction target reference · if applicable",
                  "مرجع هدف التصحيح · إن وجد",
                ])}
              >
                <input
                  {...refInput}
                  value={correction}
                  onChange={(e) => setCorrection(e.target.value)}
                />
              </Label>
              {correction && (
                <Label
                  label={text(l, [
                    "정정 사유",
                    "Correction reason",
                    "سبب التصحيح",
                  ])}
                >
                  <textarea
                    required
                    rows={3}
                    maxLength={1000}
                    value={reason}
                    onChange={(e) => setReason(e.target.value)}
                  />
                </Label>
              )}
              <p>
                {text(l, [
                  "파일을 업로드하거나 수집 작업을 접수하지 않습니다. 원본 검증·중복 정책·현재 권한이 연결되어야 합니다.",
                  "No file is uploaded and no import job is accepted. Original validation, duplication policy and current authority must be connected.",
                  "لا يتم رفع ملف أو قبول مهمة استيراد. يجب توصيل التحقق من الأصل وسياسة التكرار والصلاحية الحالية.",
                ])}
              </p>
            </ReviewForm>
          </Panel>
        </div>
      ) : p.view === "quarantine" ? (
        <Panel
          title={text(l, [
            "격리 사유와 충돌 근거",
            "Quarantine reasons and conflict evidence",
            "أسباب العزل وأدلة التعارض",
          ])}
          id="S03:data:quarantine"
        >
          <EmptyList
            l={l}
            columns={[
              ["접수 참조", "Receipt reference", "مرجع الإيصال"],
              [
                "비민감 거절 사유",
                "Nonsensitive rejection reason",
                "سبب رفض غير حساس",
              ],
              [
                "원본 revision 충돌",
                "Original revision conflict",
                "تعارض مراجعة الأصل",
              ],
              [
                "메타데이터 참조",
                "Metadata reference",
                "مرجع البيانات الوصفية",
              ],
            ]}
            reason={[
              "격리 근거 서비스 미연결. 충돌이 없다고 해석하지 않습니다.",
              "Quarantine evidence is unconnected. This does not mean there are no conflicts.",
              "أدلة العزل غير متصلة. لا يعني ذلك عدم وجود تعارضات.",
            ]}
          />
          <div className="work-policy-note">
            <h3>
              {text(l, [
                "청구 자료의 수집 경계",
                "Billing input boundary",
                "حدود مدخلات الفوترة",
              ])}
            </h3>
            <p>
              {text(l, [
                "프롬프트·응답 본문이 포함된 자료는 정상 청구 입력으로 받지 않습니다. 격리 화면에도 본문 링크나 원문을 보관하지 않고 비민감 거절 참조만 제공합니다.",
                "Inputs containing prompt or response bodies are not normal billing inputs. Quarantine keeps no body links or source text, only nonsensitive rejection references.",
                "المدخلات المحتوية على نصوص المطالبات أو الردود ليست مدخلات فوترة عادية. لا يحتفظ العزل بروابط النصوص أو أصولها، بل بمراجع رفض غير حساسة فقط.",
              ])}
            </p>
          </div>
        </Panel>
      ) : p.view === "completeness" ? (
        <div className="work-columns completeness-columns">
          <Panel
            title={text(l, [
              "예상 범위와 수신 범위",
              "Expected and received coverage",
              "التغطية المتوقعة والمستلمة",
            ])}
            id="S03:data:completeness"
          >
            <div className="work-comparison">
              <section>
                <h3>
                  {text(l, [
                    "예상 범위",
                    "Expected coverage",
                    "التغطية المتوقعة",
                  ])}
                </h3>
                <Facts
                  l={l}
                  labels={[
                    ["바이트·행", "Bytes and rows", "البايتات والصفوف"],
                    ["파티션 manifest", "Partition manifest", "بيان الأقسام"],
                    ["watermark", "Watermark", "علامة التقدم"],
                  ]}
                />
              </section>
              <section>
                <h3>
                  {text(l, [
                    "실제 수신 범위",
                    "Received coverage",
                    "التغطية المستلمة",
                  ])}
                </h3>
                <Facts
                  l={l}
                  labels={[
                    ["바이트·행", "Bytes and rows", "البايتات والصفوف"],
                    ["수신 파티션", "Received partitions", "الأقسام المستلمة"],
                    ["watermark", "Watermark", "علامة التقدم"],
                  ]}
                />
              </section>
            </div>
          </Panel>
          <Panel
            title={text(l, [
              "누락과 독립 검증",
              "Missing scope and independent validation",
              "النطاق المفقود والتحقق المستقل",
            ])}
          >
            <Facts
              l={l}
              labels={[
                [
                  "누락 기간·분할",
                  "Missing intervals and partitions",
                  "الفترات والأقسام المفقودة",
                ],
                [
                  "독립 교차 검증",
                  "Independent cross-validation",
                  "التحقق المستقل المتقاطع",
                ],
                ["비교 digest", "Comparison digest", "بصمة المقارنة"],
              ]}
            />
            <p>
              {text(l, [
                "분모가 확인되지 않아 완전성 비율을 계산하지 않습니다. 합계 일치만으로 독립 검증이 성립하지 않습니다.",
                "No completeness percentage is calculated without a verified denominator. Matching totals alone do not establish independent validation.",
                "لا تحسب نسبة اكتمال دون مقام متحقق. تطابق الإجماليات وحده لا يثبت التحقق المستقل.",
              ])}
            </p>
          </Panel>
        </div>
      ) : (
        <div className="work-columns extraction-columns">
          <Panel
            title={text(l, [
              "원본과 추출 후보 비교",
              "Original and extraction candidate comparison",
              "مقارنة الأصل ومرشح الاستخراج",
            ])}
            id="S03:data:extraction"
          >
            <div className="work-comparison">
              <section>
                <h3>
                  {text(l, ["원본 증빙", "Original evidence", "الدليل الأصلي"])}
                </h3>
                <Facts
                  l={l}
                  labels={[
                    [
                      "원본 참조·문서 번호",
                      "Original reference and document number",
                      "مرجع الأصل ورقم المستند",
                    ],
                    ["금액·통화", "Amount and currency", "المبلغ والعملة"],
                  ]}
                />
              </section>
              <section>
                <h3>
                  {text(l, [
                    "추출 후보",
                    "Extraction candidate",
                    "مرشح الاستخراج",
                  ])}
                </h3>
                <Facts
                  l={l}
                  labels={[
                    ["후보 revision", "Candidate revision", "مراجعة المرشح"],
                    [
                      "추출 금액·통화",
                      "Extracted amount and currency",
                      "المبلغ المستخرج والعملة",
                    ],
                  ]}
                />
              </section>
            </div>
            <p>
              {text(l, [
                "OCR 완료나 모델 신뢰도는 적격 입력 확인이 아닙니다. S06의 권한 있는 작업자가 불변 근거를 확인해야 합니다.",
                "OCR completion or model confidence is not input eligibility. An authorized worker in S06 must confirm immutable evidence.",
                "اكتمال OCR أو ثقة النموذج ليس تأكيدا لأهلية المدخل. يجب أن يؤكد مسؤول مخول في S06 الأدلة غير القابلة للتغيير.",
              ])}
            </p>
          </Panel>
          <Panel
            title={text(l, [
              "비교 근거 참조 검토",
              "Comparison evidence reference review",
              "مراجعة مرجع دليل المقارنة",
            ])}
          >
            <ReviewForm
              l={l}
              id="S03:action:review-extraction-reference"
              onClear={clear}
            >
              <Label
                label={text(l, [
                  "접수·후보 참조",
                  "Receipt or candidate reference",
                  "مرجع الإيصال أو المرشح",
                ])}
              >
                <input
                  {...refInput}
                  required
                  value={receipt}
                  onChange={(e) => setReceipt(e.target.value)}
                />
              </Label>
              <Label
                label={text(l, [
                  "원본 비교 근거 참조",
                  "Original comparison evidence reference",
                  "مرجع دليل مقارنة الأصل",
                ])}
              >
                <input
                  {...refInput}
                  required
                  value={connection}
                  onChange={(e) => setConnection(e.target.value)}
                />
              </Label>
              <p>
                {text(l, [
                  "입력 형식만 검토합니다. 추출 검증을 기록하거나 후보를 적격 입력으로 발행하지 않습니다.",
                  "Only input format is reviewed. No extraction verification is recorded and no candidate is published as eligible.",
                  "تراجع صيغة الإدخال فقط. لا يسجل تحقق الاستخراج ولا ينشر المرشح كمدخل مؤهل.",
                ])}
              </p>
            </ReviewForm>
            <Facts
              l={l}
              labels={[
                [
                  "확인 작업자·근거 digest",
                  "Confirmation actor and evidence digest",
                  "جهة التأكيد وبصمة الدليل",
                ],
                [
                  "현재 입력 적격성",
                  "Current input eligibility",
                  "أهلية المدخل الحالية",
                ],
              ]}
            />
          </Panel>
        </div>
      )}
    </>
  );
}
