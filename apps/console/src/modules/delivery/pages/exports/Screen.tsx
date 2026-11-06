import { useEffect, useRef, useState } from "react";
import { fields, t, viewFields, type Props, type Text } from "../../../../shared/domain-screen-ui/finance/contracts";
import { EmptyTable, Evidence, Intro, Notice, Panel } from "../../../../shared/domain-screen-ui/finance/ui";
const categories = [
  { id: "invoice", label: ["청구 문서", "Invoice document", "مستند الفاتورة"] },
  {
    id: "own-usage",
    label: [
      "수신자 본인 사용량",
      "Recipient's own usage",
      "استخدام المستلم نفسه",
    ],
  },
  {
    id: "sales",
    label: [
      "허용된 판매 상세",
      "Permitted sales detail",
      "تفاصيل المبيعات المسموح بها",
    ],
  },
] satisfies { id: string; label: Text }[];
const holdHelp: Record<string, Text> = {
  billing: [
    "명시한 청구 단계를 대상으로 합니다. 전달·독촉·법적 보존 보류를 대신하지 않습니다.",
    "Targets the specified billing stage. It does not replace delivery, collections or legal-retention holds.",
    "يستهدف مرحلة الفوترة المحددة ولا يحل محل تعليق التسليم أو التحصيل أو الاحتفاظ القانوني.",
  ],
  delivery: [
    "통제 가능한 발송·공개 경로가 대상입니다. 이미 외부 전달된 사본의 회수를 보장하지 않습니다.",
    "Targets controlled sending and disclosure routes. Recall of copies already delivered externally is not guaranteed.",
    "يستهدف مسارات الإرسال والإفصاح الخاضعة للتحكم. لا يُضمن استرجاع النسخ المسلمة خارجياً.",
  ],
  collections: [
    "명시한 독촉·수금 활동이 대상입니다. 지급 기한·수납·잔액은 변경하지 않습니다.",
    "Targets specified reminders and collection activities. Due dates, receipts and balances do not change.",
    "يستهدف التذكيرات وأنشطة التحصيل المحددة. لا تتغير المواعيد أو التحصيلات أو الأرصدة.",
  ],
};
export default function Disclosure(props: Props) {
  const { locale: l, view, onDirty, onState } = props;
  const [recipient, setRecipient] = useState(""),
    [selected, setSelected] = useState<string[]>([]),
    [reviewed, setReviewed] = useState(false),
    [invalid, setInvalid] = useState(false),
    [holdType, setHoldType] = useState("delivery");
  const recipientInput = useRef<HTMLInputElement>(null);
  const modified = Boolean(recipient || selected.length);
  useEffect(() => {
    onDirty(modified);
    return () => onDirty(false);
  }, [modified, onDirty]);
  const identity = JSON.stringify([
    "I04",
    view,
    selected,
    reviewed,
    holdType,
    Boolean(recipient),
  ]);
  useEffect(
    () =>
      onState(identity, [
        "I04:state:unconnected",
        "I04:input:recipient",
        "I04:input:public-categories",
        "I04:input:hold-type",
        "I04:action:review-policy-inputs",
        "I04:action:clear-inputs",
        ...(viewFields.I04[view] ?? []).map((k) => `I04:data:${fields[k].id}`),
      ]),
    [identity, onState, view],
  );
  const projection = (
    <Panel
      title={t(l, [
        "수신자용 공개 구성",
        "Recipient-facing projection",
        "المكونات المعروضة للمستلم",
      ])}
    >
      <p className="fin-meta">
        {t(l, [
          "아래는 선택한 데이터 범주입니다. 실제 공개 정책 승인이나 공개 파일 미리보기가 아닙니다.",
          "The selected data categories are shown below. This is neither an approved disclosure policy nor an actual public-file preview.",
          "تُعرض أدناه فئات البيانات المختارة. ليست سياسة إفصاح معتمدة ولا معاينة لملف إفصاح فعلي.",
        ])}
      </p>
      {selected.length ? (
        <ul className="fin-projection">
          {categories
            .filter((category) => selected.includes(category.id))
            .map((category) => (
              <li key={category.id}>
                <strong>{t(l, category.label)}</strong>
                <p>
                  {t(l, [
                    "허용 필드·수신 권리·전량 검증 미확인",
                    "Allowed fields, recipient rights and full validation unverified",
                    "الحقول المسموحة وحقوق المستلم والتحقق الكامل غير متحققة",
                  ])}
                </p>
              </li>
            ))}
        </ul>
      ) : (
        <p>
          {t(l, [
            "데이터 범주를 선택해 구성을 검토하세요. 기본 허용 범주는 없습니다.",
            "Select categories to inspect the composition. No category is allowed by default.",
            "اختر فئات لفحص المكونات. لا توجد فئة مسموحة افتراضياً.",
          ])}
        </p>
      )}
      <Notice
        l={l}
        text={[
          "내부 원가·마진·타 고객 자료·내부 시트·원천 locator는 고객 공개 구성에 포함하지 않습니다. 새 필드의 분류가 없으면 제외합니다.",
          "Internal costs, margins, other-customer data, internal sheets and source locators are excluded from customer projection. Unclassified new fields remain excluded.",
          "تُستبعد التكاليف الداخلية والهوامش وبيانات العملاء الآخرين والأوراق الداخلية ومواقع المصادر من عرض العميل. الحقول الجديدة غير المصنفة تبقى مستبعدة.",
        ]}
      />
    </Panel>
  );
  return (
    <div
      className="fin-screen fin-disclosure"
      dir={l === "ar" ? "rtl" : "ltr"}
      data-finance-screen="I04"
      data-view={view}
    >
      <Intro
        l={l}
        purpose={[
          "수신자에게 허용된 자료만 새 공개 파일로 구성하고, 전량 검증과 전달 통제를 별도로 확인합니다.",
          "Compose new public files from recipient-permitted data and inspect full validation separately from delivery controls.",
          "كوّن ملفات إفصاح جديدة من بيانات المستلم المسموحة وافحص التحقق الكامل منفصلاً عن ضوابط التسليم.",
        ]}
      />
      {view === "policies" || view === "preview" ? (
        <>
          <div className="fin-two">
            <Panel
              title={t(l, [
                "공개 구성 입력",
                "Disclosure composition inputs",
                "مدخلات مكونات الإفصاح",
              ])}
            >
              <p className="fin-meta">
                {t(l, [
                  "브라우저 입력 검토 전용입니다. 정책 저장·파일 생성·전달은 실행하지 않습니다.",
                  "Browser input review only. No policy is saved and no file is generated or delivered.",
                  "لمراجعة المدخلات في المتصفح فقط. لا تُحفظ سياسة ولا يُنشأ ملف أو يُسلم.",
                ])}
              </p>
              <form
                noValidate
                onSubmit={(e) => {
                  e.preventDefault();
                  if (
                    !/^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(recipient) ||
                    !selected.length
                  ) {
                    setInvalid(true);
                    setReviewed(false);
                    recipientInput.current?.focus();
                  } else {
                    setInvalid(false);
                    setReviewed(true);
                  }
                }}
              >
                <label htmlFor="I04-recipient">
                  {t(l, [
                    "수신 범위 참조 (필수)",
                    "Recipient scope reference (required)",
                    "مرجع نطاق المستلم (مطلوب)",
                  ])}
                  <input
                    id="I04-recipient"
                    ref={recipientInput}
                    dir="ltr"
                    autoComplete="off"
                    maxLength={128}
                    value={recipient}
                    aria-invalid={invalid}
                    onChange={(e) => {
                      setRecipient(e.target.value);
                      setReviewed(false);
                      setInvalid(false);
                    }}
                  />
                </label>
                <fieldset className="fin-allowlist">
                  <legend>
                    {t(l, [
                      "고객 공개 데이터 범주",
                      "Customer disclosure categories",
                      "فئات الإفصاح للعميل",
                    ])}
                  </legend>
                  {categories.map((category) => (
                    <label key={category.id}>
                      <input
                        type="checkbox"
                        checked={selected.includes(category.id)}
                        data-item={`I04:input:category-${category.id}`}
                        onChange={(e) => {
                          setSelected((values) =>
                            e.target.checked
                              ? [...values, category.id]
                              : values.filter((v) => v !== category.id),
                          );
                          setReviewed(false);
                          setInvalid(false);
                        }}
                      />
                      <span>{t(l, category.label)}</span>
                    </label>
                  ))}
                </fieldset>
                <div className="fin-actions">
                  <button
                    type="submit"
                    data-item="I04:action:review-policy-inputs"
                  >
                    {t(l, [
                      "공개 구성 입력 검토",
                      "Review composition inputs",
                      "مراجعة مدخلات المكونات",
                    ])}
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setRecipient("");
                      setSelected([]);
                      setReviewed(false);
                      setInvalid(false);
                    }}
                  >
                    {t(l, ["입력 지우기", "Clear inputs", "مسح المدخلات"])}
                  </button>
                </div>
                <p role="status" className="fin-feedback">
                  {invalid
                    ? t(l, [
                        "유효한 수신 범위 참조와 하나 이상의 범주가 필요합니다.",
                        "A valid recipient scope reference and at least one category are required.",
                        "يلزم مرجع نطاق صالح وفئة واحدة على الأقل.",
                      ])
                    : reviewed
                      ? t(l, [
                          "입력 형식만 검토했습니다. 수신자 권한·필드 조합의 추론 위험·공개 적격성은 미확인입니다.",
                          "Input format reviewed only. Recipient authority, inference risk and disclosure eligibility remain unverified.",
                          "تمت مراجعة الصيغة فقط. صلاحيات المستلم ومخاطر الاستنتاج وأهلية الإفصاح غير متحققة.",
                        ])
                      : t(l, [
                          "범주 선택은 권한이나 공개 정책을 부여하지 않습니다.",
                          "Category selection grants no authority or disclosure policy.",
                          "اختيار الفئات لا يمنح صلاحية أو سياسة إفصاح.",
                        ])}
                </p>
              </form>
            </Panel>
            {projection}
          </div>
          <Panel
            title={t(l, [
              "공개 정책·파일 검증 근거",
              "Disclosure policy and file-validation evidence",
              "أدلة سياسة الإفصاح والتحقق من الملف",
            ])}
          >
            <Evidence l={l} screenId="I04" keys={viewFields.I04[view]} />
            <Notice
              l={l}
              text={[
                "고객 XLSX는 내부 파일의 시트를 숨기거나 보호해 만들지 않습니다. 새 파일의 시트·주석·이름·캐시·문서 속성·외부 연결까지 검사해야 하며 샘플 검사는 전량 검증을 대신하지 않습니다.",
                "Customer XLSX files are not created by hiding or protecting internal sheets. A new file needs checks across sheets, comments, names, caches, properties and external links. Sampling never replaces full validation.",
                "لا تُنشأ ملفات العميل بإخفاء الأوراق الداخلية أو حمايتها. يجب فحص أوراق الملف الجديد وتعليقاته وأسمائه وذاكرته وخصائصه وروابطه الخارجية. العينة لا تحل محل التحقق الكامل.",
              ]}
            />
          </Panel>
        </>
      ) : view === "packages" ? (
        <Panel
          title={t(l, [
            "고정 전달 패키지",
            "Immutable delivery packages",
            "حزم التسليم الثابتة",
          ])}
        >
          <EmptyTable
            l={l}
            screenId="I04"
            columns={viewFields.I04.packages}
            caption={[
              "패키지·정책·검증",
              "Packages, policy and validation",
              "الحزم والسياسة والتحقق",
            ]}
          />
          <Notice
            l={l}
            text={[
              "저장된 확정 파일을 임의 재생성으로 교체하지 않습니다. 전체 행·금액·해시·수신자 권리가 검증되기 전에는 전달하지 않습니다.",
              "Final artifacts are not replaced by silent regeneration. Delivery requires full-row, amount, hash and recipient-right verification.",
              "لا تُستبدل الملفات النهائية بإعادة إنشاء صامتة. يتطلب التسليم التحقق من جميع الصفوف والمبالغ والبصمة وحقوق المستلم.",
            ]}
          />
        </Panel>
      ) : (
        <Panel
          title={t(l, [
            "전달·접근·보류 이력",
            "Delivery, access and hold history",
            "سجل التسليم والوصول والتعليق",
          ])}
        >
          <Evidence l={l} screenId="I04" keys={viewFields.I04.history} />
          <Notice
            l={l}
            text={[
              "발행·전기 완료는 전달 안전을 입증하지 않습니다. 접근 회수는 외부로 전달된 사본의 회수를 보장하지 않습니다.",
              "Issuance or posting completion does not prove delivery safety. Access revocation does not guarantee recall of externally delivered copies.",
              "اكتمال الإصدار أو الترحيل لا يثبت أمان التسليم. إلغاء الوصول لا يضمن استرجاع النسخ المسلمة خارجياً.",
            ]}
          />
        </Panel>
      )}
      <details className="fin-requirements">
        <summary>
          {t(l, [
            "유형별 보류·해제 영향",
            "Type-specific hold and release effects",
            "آثار التعليق ورفعه حسب النوع",
          ])}
        </summary>
        <label className="fin-purpose">
          {t(l, [
            "영향을 확인할 유형",
            "Effect type to inspect",
            "نوع الأثر المراد فحصه",
          ])}
          <select
            value={holdType}
            onChange={(e) => setHoldType(e.target.value)}
          >
            <option value="billing">
              {t(l, ["청구", "Billing", "الفوترة"])}
            </option>
            <option value="delivery">
              {t(l, ["전달", "Delivery", "التسليم"])}
            </option>
            <option value="collections">
              {t(l, ["독촉·수금", "Collections", "التحصيل"])}
            </option>
          </select>
        </label>
        <p>{t(l, holdHelp[holdType])}</p>
        <p>
          {t(l, [
            "보류 요청·효력 확인·해제 요청·해제 확인은 별도입니다. 현재 효력은 미확인입니다. 해제가 다른 보류를 제거하거나 업무를 자동 재개하지 않습니다. 지급 기한과 잔액도 바꾸지 않습니다.",
            "A hold request, confirmed effect, release request and confirmed release are distinct. Current effect is unknown. Release removes no other hold, resumes no work and changes no due date or balance.",
            "طلب التعليق وتأكيد أثره وطلب رفعه وتأكيد الرفع حالات منفصلة. الأثر الحالي غير معروف. الرفع لا يزيل تعليقاً آخر ولا يستأنف عملاً ولا يغير موعداً أو رصيداً.",
          ])}
        </p>
      </details>
      <p className="fin-meta">
        {t(l, [
          "공개 파일 생성·전량 검증·전달·재전달·링크 회수·보류 명령은 연결되지 않았으며 실행하지 않습니다.",
          "Public-file generation, full validation, delivery, redelivery, link revocation and hold commands are not connected or executed.",
          "أوامر إنشاء ملفات الإفصاح والتحقق الكامل والتسليم وإعادته وإلغاء الروابط والتعليق غير متصلة ولا تُنفذ.",
        ])}
      </p>
    </div>
  );
}
