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
export default function Findings(p: Props) {
  const l = p.locale;
  const [kind, setKind] = useState("");
  const [scope, setScope] = useState("");
  const [basis, setBasis] = useState("");
  const [reason, setReason] = useState("");
  const [meaning, setMeaning] = useState("");
  const [evidence, setEvidence] = useState("");
  useReview(p, !!(kind || scope || basis || reason || meaning || evidence));
  return (
    <>
      <Boundary l={l} />
      <div className="work-columns finding-columns">
        <Panel
          title={text(
            l,
            p.view === "findings"
              ? [
                  "특이사항 검토 목록",
                  "Findings for review",
                  "نتائج الفحص للمراجعة",
                ]
              : ["관련 근거", "Related evidence", "الأدلة ذات الصلة"],
          )}
          id={`U04:data:${p.view}`}
        >
          {p.view === "findings" && (
            <EmptyList
              l={l}
              columns={[
                ["유형·대상", "Kind and target", "النوع والهدف"],
                ["비교 기준", "Comparison basis", "أساس المقارنة"],
                ["영향 확실성", "Impact certainty", "درجة تأكد الأثر"],
                ["검토 상태", "Review state", "حالة المراجعة"],
              ]}
              reason={[
                "특이사항 조회가 미연결입니다. 발견되지 않은 항목이나 청구 정확성을 확정하지 않습니다.",
                "The findings reader is unconnected. This establishes neither an absence of findings nor billing accuracy.",
                "خدمة قراءة نتائج الفحص غير متصلة. لا يثبت ذلك غياب النتائج أو دقة الفوترة.",
              ]}
            />
          )}
          <div className="work-evidence-stack">
            <h3>
              {text(l, [
                "원천·계약·비교 기간",
                "Source, contract and comparison interval",
                "المصدر والعقد وفترة المقارنة",
              ])}
            </h3>
            <Facts
              l={l}
              labels={[
                ["원천·개정", "Source and revision", "المصدر والمراجعة"],
                ["계약 조항", "Contract clause", "بند العقد"],
                [
                  "비교 기간·기준시각",
                  "Comparison interval and as-of",
                  "فترة المقارنة ومرجعها",
                ],
                [
                  "영향 금액·통화",
                  "Impact amount and currency",
                  "مبلغ الأثر والعملة",
                ],
              ]}
            />
          </div>
        </Panel>
        <Panel
          title={text(l, [
            "검토 메모 초안",
            "Review note draft",
            "مسودة ملاحظة المراجعة",
          ])}
        >
          {p.view === "evidence" ? (
            <>
              <Facts
                l={l}
                labels={[
                  [
                    "현재 근거 접근",
                    "Current evidence access",
                    "الوصول الحالي للأدلة",
                  ],
                  [
                    "정정 문서 참조",
                    "Correction document reference",
                    "مرجع مستند التصحيح",
                  ],
                  [
                    "정상 확인·해결 이력",
                    "Normal review and resolution history",
                    "سجل المراجعة والحل",
                  ],
                ]}
              />
              <p>
                {text(l, [
                  "원천 대사·특이사항 검토·청구 정정은 각각의 책임과 상태를 유지합니다. 연결된 근거의 접근 권한을 확인하기 전에는 원문을 표시하지 않습니다.",
                  "Source reconciliation, finding review and billing correction keep separate responsibilities and states. Source text is unavailable until access to linked evidence is checked.",
                  "تحافظ مطابقة المصدر ومراجعة النتائج وتصحيح الفوترة على مسؤوليات وحالات مستقلة. النص الأصلي غير متاح قبل التحقق من الوصول إلى الأدلة.",
                ])}
              </p>
            </>
          ) : (
            <ReviewForm
              l={l}
              id="U04:action:review-finding-draft"
              onClear={() => {
                setKind("");
                setScope("");
                setBasis("");
                setReason("");
                setMeaning("");
                setEvidence("");
              }}
              validate={(form) => {
                const el = form.elements.namedItem(
                  "finding_reason",
                ) as HTMLTextAreaElement | null;
                if (el)
                  el.setCustomValidity(
                    /^(done|complete[d]?|완료|تم|اكتمل)[.!\s]*$/i.test(
                      el.value.trim(),
                    )
                      ? text(l, [
                          "판정 근거를 구체적으로 입력하세요.",
                          "Provide a specific review rationale.",
                          "أدخل مبررا محددا للمراجعة.",
                        ])
                      : "",
                  );
              }}
            >
              <Label
                label={text(l, [
                  "특이사항 유형",
                  "Finding kind",
                  "نوع نتيجة الفحص",
                ])}
              >
                <select
                  required
                  value={kind}
                  onChange={(e) => setKind(e.target.value)}
                >
                  <option value="">
                    {text(l, ["선택하세요", "Choose", "اختر"])}
                  </option>
                  {[
                    ["spike", "비용 급증", "Cost spike", "ارتفاع التكلفة"],
                    ["new", "신규 과금", "New charge", "رسوم جديدة"],
                    [
                      "benefit",
                      "혜택 누락 의심",
                      "Suspected missing benefit",
                      "اشتباه غياب ميزة",
                    ],
                    [
                      "conflict",
                      "자료 충돌",
                      "Evidence conflict",
                      "تعارض الأدلة",
                    ],
                    ["meter", "계측 누락", "Metering gap", "فجوة القياس"],
                  ].map(([v, ko, en, ar]) => (
                    <option value={v} key={v}>
                      {text(l, [ko, en, ar])}
                    </option>
                  ))}
                </select>
              </Label>
              <div className="work-form-grid">
                <Label
                  label={text(l, [
                    "대상·기간 참조",
                    "Scope and interval reference",
                    "مرجع النطاق والفترة",
                  ])}
                >
                  <input
                    {...refInput}
                    required
                    value={scope}
                    onChange={(e) => setScope(e.target.value)}
                  />
                </Label>
                <Label
                  label={text(l, [
                    "비교 기준 참조",
                    "Comparison basis reference",
                    "مرجع أساس المقارنة",
                  ])}
                >
                  <input
                    {...refInput}
                    required
                    value={basis}
                    onChange={(e) => setBasis(e.target.value)}
                  />
                </Label>
                <Label
                  label={text(l, [
                    "금액 의미",
                    "Amount meaning",
                    "معنى المبلغ",
                  ])}
                >
                  <select
                    required
                    value={meaning}
                    onChange={(e) => setMeaning(e.target.value)}
                  >
                    <option value="">
                      {text(l, ["선택하세요", "Choose", "اختر"])}
                    </option>
                    {[
                      ["cost", "원가", "Cost", "التكلفة"],
                      ["sales", "판매", "Sales", "المبيعات"],
                      ["allocation", "배분", "Allocation", "التوزيع"],
                    ].map(([v, ko, en, ar]) => (
                      <option key={v} value={v}>
                        {text(l, [ko, en, ar])}
                      </option>
                    ))}
                  </select>
                </Label>
                <Label
                  label={text(l, [
                    "근거 참조",
                    "Evidence reference",
                    "مرجع الدليل",
                  ])}
                >
                  <input
                    {...refInput}
                    required
                    value={evidence}
                    onChange={(e) => setEvidence(e.target.value)}
                  />
                </Label>
              </div>
              <Label
                label={text(l, [
                  "검토 사유 초안",
                  "Review rationale draft",
                  "مسودة مبرر المراجعة",
                ])}
                hint={text(l, [
                  "비밀·고객 원문을 복사하지 마세요. 단순 완료 기록은 근거가 아닙니다.",
                  "Do not copy secrets or customer source bodies. A bare completion note is not evidence.",
                  "لا تنسخ الأسرار أو نصوص العميل الأصلية. ملاحظة الإكمال وحدها ليست دليلا.",
                ])}
              >
                <textarea
                  required
                  name="finding_reason"
                  rows={5}
                  maxLength={2000}
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                />
              </Label>
              <p>
                {text(l, [
                  "초안은 검토 판정·담당 배정·조정 요청을 등록하지 않습니다. 의심 금액을 청구액에서 자동 차감하지 않습니다.",
                  "This draft records no judgment, assignment or adjustment request. Suspected amounts are never automatically deducted from billing.",
                  "هذه المسودة لا تسجل حكما أو تعيينا أو طلب تعديل. لا تخصم المبالغ المشتبه بها تلقائيا من الفوترة.",
                ])}
              </p>
            </ReviewForm>
          )}
        </Panel>
      </div>
    </>
  );
}
