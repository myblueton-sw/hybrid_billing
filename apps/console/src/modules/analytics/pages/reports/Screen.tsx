import { useEffect, useState } from "react";
import { t, viewFields, type Props, type Text } from "../../../../shared/domain-screen-ui/finance/contracts";
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
    key: "meaning",
    label: ["금액 의미", "Amount meaning", "دلالة المبلغ"],
    required: true,
    options: [
      {
        value: "billed",
        label: [
          "원천 청구 비용 (BilledCost)",
          "Source billed cost (BilledCost)",
          "التكلفة المفوترة بالمصدر (BilledCost)",
        ],
      },
      {
        value: "effective",
        label: [
          "원천 유효 비용 (EffectiveCost)",
          "Source effective cost (EffectiveCost)",
          "التكلفة الفعلية بالمصدر (EffectiveCost)",
        ],
      },
      {
        value: "internal",
        label: ["내부 원가", "Internal cost", "التكلفة الداخلية"],
      },
      { value: "sales", label: ["판매 금액", "Sales amount", "مبلغ المبيعات"] },
      {
        value: "allocation",
        label: ["배분 금액", "Allocated amount", "المبلغ الموزع"],
      },
    ],
  },
  {
    key: "time-basis",
    label: ["귀속 기준", "Attribution basis", "أساس الإسناد"],
    required: true,
    options: [
      {
        value: "historical",
        label: [
          "당시 귀속·정산",
          "Historical attribution and settlement",
          "الإسناد والتسوية التاريخيان",
        ],
      },
      {
        value: "current",
        label: [
          "현재 조직 재분석",
          "Current-organization reanalysis",
          "إعادة التحليل بالمؤسسة الحالية",
        ],
      },
    ],
  },
  {
    key: "currency",
    label: [
      "원통화 코드 (ISO 4217)",
      "Original currency code (ISO 4217)",
      "رمز العملة الأصلية (ISO 4217)",
    ],
    required: true,
  },
  {
    key: "dimension",
    label: ["그룹 차원", "Group dimension", "بعد التجميع"],
    required: true,
    options: [
      { value: "provider", label: ["공급자", "Provider", "المزود"] },
      { value: "service", label: ["서비스", "Service", "الخدمة"] },
      { value: "organization", label: ["조직", "Organization", "المؤسسة"] },
      { value: "model", label: ["모델", "Model", "النموذج"] },
    ],
  },
];
const aging: readonly Text[] = [
  ["미도래", "Not yet due", "لم يستحق بعد"],
  ["당일", "Due today", "مستحق اليوم"],
  ["1–30일", "1–30 days", "1–30 يوماً"],
  ["31–60일", "31–60 days", "31–60 يوماً"],
  ["61–90일", "61–90 days", "61–90 يوماً"],
  ["91일 이상", "91+ days", "91 يوماً فأكثر"],
];
export default function Reports(props: Props) {
  const { locale: l, view, onDirty } = props;
  const [name, setName] = useState(""),
    [draftView, setDraftView] = useState<{
      name: string;
      criteria: Record<string, string>;
    } | null>(null),
    [message, setMessage] = useState<Text | null>(null),
    [localRevision, setLocalRevision] = useState(0);
  const [criteria, onReview] = useScreenState(props, "R01", {
    localRevision,
    retained: Boolean(draftView),
  });
  useEffect(
    () =>
      onDirty(
        Boolean(
          name || draftView || Object.values(criteria.values).some(Boolean),
        ),
      ),
    [name, draftView, criteria.values, onDirty],
  );
  return (
    <div
      className="fin-screen fin-reports"
      dir={l === "ar" ? "rtl" : "ltr"}
      data-finance-screen="R01"
      data-view={view}
    >
      <Intro
        l={l}
        purpose={[
          "비용·판매·배분을 같은 금액 의미·통화·기준시각으로 비교하고 출처와 미완료 범위를 확인합니다.",
          "Compare costs, sales and allocations with consistent amount meaning, currency and as-of, then inspect provenance and missing scope.",
          "قارن التكاليف والمبيعات والتوزيعات بدلالة مبلغ وعملة ووقت مرجعي متسق، ثم افحص المصدر والنطاق الناقص.",
        ]}
      />
      <Criteria
        l={l}
        screenId="R01"
        inputs={
          view === "comparison"
            ? [
                ...inputs,
                {
                  key: "compareStart",
                  label: [
                    "비교 시작일 (포함)",
                    "Comparison start (inclusive)",
                    "بداية المقارنة (مشمولة)",
                  ],
                  type: "date",
                  required: true,
                },
                {
                  key: "compareEnd",
                  label: [
                    "비교 종료일 (미포함)",
                    "Comparison end (exclusive)",
                    "نهاية المقارنة (غير مشمولة)",
                  ],
                  type: "date",
                  required: true,
                },
              ]
            : inputs
        }
        onDirty={onDirty}
        onReview={onReview}
        title={["분석 기준", "Analysis criteria", "معايير التحليل"]}
      />
      <Notice
        l={l}
        text={
          criteria.values["time-basis"] === "current"
            ? [
                "현재 조직 재분석은 당시 확정 정산과 다른 분석입니다. 확정 문서와 과거 귀속을 바꾸지 않습니다.",
                "Current-organization reanalysis is separate from historical finalized settlement. It does not change finalized documents or past attribution.",
                "إعادة التحليل بالمؤسسة الحالية منفصلة عن التسوية التاريخية النهائية. لا تغير المستندات النهائية أو الإسناد السابق.",
              ]
            : [
                "금액 의미와 통화는 단일 기준으로 선택합니다. 서로 다른 의미·통화를 합산하거나 조회 권한을 선택값으로 부여하지 않습니다.",
                "Choose one amount meaning and currency. Different meanings or currencies are not summed, and selections confer no read authority.",
                "اختر دلالة مبلغ وعملة واحدة. لا تُجمع دلالات أو عملات مختلفة ولا تمنح الاختيارات صلاحية قراءة.",
              ]
        }
      />
      {view === "payment-terms" ? (
        <>
          <PaymentEvidence l={l} id="R01" />
          <Panel
            title={t(l, [
              "통화별 연체 구간",
              "Aging buckets by currency",
              "شرائح التأخر حسب العملة",
            ])}
          >
            <div className="fin-aging">
              {aging.map((label, i) => (
                <div key={i}>
                  <strong>{t(l, label)}</strong>
                  <p>
                    {t(l, [
                      "잔액 미확인",
                      "Balance unknown",
                      "الرصيد غير معروف",
                    ])}
                  </p>
                </div>
              ))}
            </div>
            <p className="fin-unconfirmed">
              {t(l, [
                "수납 미확인: 확정 연체 합계에서 제외하며, 현재 해당 금액도 미확인입니다.",
                "Unconfirmed receipts remain separate from confirmed overdue totals. Their amounts are also currently unknown.",
                "التحصيلات غير المتحققة منفصلة عن إجماليات التأخر المؤكد. مبالغها غير معروفة حالياً أيضاً.",
              ])}
            </p>
          </Panel>
        </>
      ) : view === "recognition" ? (
        <RecognitionEvidence l={l} id="R01" />
      ) : view === "summary" ? (
        <>
          <div className="fin-two">
            <Panel
              title={t(l, [
                "선택 기준의 결과",
                "Results for the selected criteria",
                "نتائج المعايير المختارة",
              ])}
            >
              <Evidence
                l={l}
                screenId="R01"
                keys={[
                  "amountMeaning",
                  "originalCurrency",
                  "timeBasis",
                  "dimensions",
                  "total",
                  "quantity",
                  "source",
                ]}
              />
            </Panel>
            <Panel
              title={t(l, ["월별 흐름", "Monthly trend", "الاتجاه الشهري"])}
            >
              <div className="fin-trend-empty">
                <strong>
                  {t(l, [
                    "월별 집계 미연결",
                    "Monthly aggregates not connected",
                    "التجميعات الشهرية غير متصلة",
                  ])}
                </strong>
                <p>
                  {t(l, [
                    "월별 출처·완전성·통화별 집계가 연결되면 추이를 표시합니다. 누락 월을 0이나 빈 막대로 표시하지 않습니다.",
                    "Trends require monthly provenance, completeness and currency-specific aggregates. Missing months are not represented as zero or empty bars.",
                    "تتطلب الاتجاهات مصادر شهرية واكتمالاً وتجميعات حسب العملة. الأشهر المفقودة لا تُعرض كصفر أو أعمدة فارغة.",
                  ])}
                </p>
              </div>
            </Panel>
          </div>
          <Panel
            title={t(l, [
              "월별 수치 표",
              "Monthly values table",
              "جدول القيم الشهرية",
            ])}
          >
            <EmptyTable
              l={l}
              screenId="R01"
              columns={[
                "period",
                "amountMeaning",
                "originalCurrency",
                "total",
                "source",
              ]}
              caption={[
                "차트와 같은 기준의 월별 수치",
                "Monthly values on the same basis as the trend",
                "قيم شهرية بالأساس نفسه للاتجاه",
              ]}
            />
          </Panel>
        </>
      ) : view === "comparison" ? (
        <Panel
          title={t(l, [
            "동일 기준의 기간 비교",
            "Period comparison on a consistent basis",
            "مقارنة الفترات بأساس متسق",
          ])}
        >
          <div className="fin-two">
            <section>
              <h3>
                {t(l, ["선택 기간", "Selected period", "الفترة المختارة"])}
              </h3>
              <Evidence
                l={l}
                screenId="R01"
                keys={["period", "total", "source"]}
              />
            </section>
            <section>
              <h3>
                {t(l, ["비교 기간", "Comparison period", "فترة المقارنة"])}
              </h3>
              <Evidence
                l={l}
                screenId="R01"
                keys={["comparison", "total", "source"]}
              />
            </section>
          </div>
          <Notice
            l={l}
            text={[
              "비교 분모·포함 일수·완전성이 미확인입니다. 변동액·변화율을 계산하지 않으며 비교 기간 미수신을 0으로 취급하지 않습니다.",
              "The comparison denominator, included days and completeness are unverified. No change amount or rate is calculated, and missing comparison data is not zero.",
              "مقام المقارنة وأيامها واكتمالها غير متحققة. لا يُحسب تغير المبلغ أو المعدل ولا تُعامل بيانات المقارنة المفقودة كصفر.",
            ]}
          />
        </Panel>
      ) : view === "rows" ? (
        <Panel
          title={t(l, [
            "출처로 이어지는 상세 행",
            "Detail rows with provenance",
            "صفوف التفاصيل مع مصادرها",
          ])}
        >
          <EmptyTable
            l={l}
            screenId="R01"
            columns={viewFields.R01.rows}
            caption={[
              "동일 snapshot의 상세",
              "Detail for the same snapshot",
              "تفاصيل اللقطة نفسها",
            ]}
          />
        </Panel>
      ) : view === "views" ? (
        <Panel
          title={t(l, [
            "보기 구성 임시 검토",
            "Temporary view configuration",
            "إعداد عرض مؤقت",
          ])}
        >
          <p>
            {t(l, [
              "조회 결과나 권한은 저장하지 않습니다. 아래 이름과 조건은 이 화면을 떠나면 사라집니다.",
              "No results or authority are saved. The name and criteria below are discarded when leaving this screen.",
              "لا تُحفظ نتائج أو صلاحيات. يُتخلص من الاسم والمعايير أدناه عند مغادرة الشاشة.",
            ])}
          </p>
          <label htmlFor="R01-view-name">
            {t(l, ["보기 이름", "View name", "اسم العرض"])}
            <input
              id="R01-view-name"
              maxLength={80}
              value={name}
              onChange={(e) => {
                setName(e.target.value);
                setLocalRevision((n) => n + 1);
                setMessage(null);
              }}
            />
          </label>
          <div className="fin-actions">
            <button
              type="button"
              onClick={() => {
                if (!name.trim() || !criteria.reviewed)
                  setMessage([
                    "조회 조건 검토와 보기 이름이 필요합니다.",
                    "Review the criteria and enter a view name first.",
                    "راجع المعايير وأدخل اسم العرض أولاً.",
                  ]);
                else {
                  setLocalRevision((n) => n + 1);
                  setDraftView({
                    name: name.trim(),
                    criteria: { ...criteria.values },
                  });
                  setMessage([
                    "현재 화면에서만 구성을 임시 보관했습니다. 서버에 저장하지 않았습니다.",
                    "Configuration retained only on this screen. It was not saved to the server.",
                    "حُفظ الإعداد مؤقتاً في هذه الشاشة فقط وليس على الخادم.",
                  ]);
                }
              }}
            >
              {t(l, [
                "현재 화면에서 임시 보관",
                "Retain on this screen",
                "الاحتفاظ في هذه الشاشة",
              ])}
            </button>
            {draftView && (
              <button
                type="button"
                onClick={() => {
                  setDraftView(null);
                  setLocalRevision((n) => n + 1);
                  setName("");
                  setMessage(null);
                }}
              >
                {t(l, [
                  "임시 보기 지우기",
                  "Clear temporary view",
                  "مسح العرض المؤقت",
                ])}
              </button>
            )}
          </div>
          {draftView && (
            <p className="fin-temporary-view">
              <strong>{draftView.name}</strong>
              <span>
                {t(l, [
                  "분석 기준 입력만 포함 · 결과·권한 미포함",
                  "Analysis criteria only; no results or authority",
                  "المعايير فقط؛ دون نتائج أو صلاحيات",
                ])}
              </span>
            </p>
          )}
          <p role="status">{message && t(l, message)}</p>
          <Evidence l={l} screenId="R01" keys={viewFields.R01.views} />
        </Panel>
      ) : (
        <Panel
          title={t(l, [
            "내보내기 작업과 고정 범위",
            "Export jobs and fixed scope",
            "مهام التصدير والنطاق الثابت",
          ])}
        >
          <Evidence l={l} screenId="R01" keys={viewFields.R01.exports} />
          <Notice
            l={l}
            text={[
              "현재 페이지와 필터 전체는 다릅니다. 허용 필드·목적·통화별 합계·snapshot·대상 manifest를 서버에서 고정해야 하며 내보내기 작업은 아직 연결되지 않았습니다.",
              "The current page differs from all matching results. Permitted fields, purpose, currency totals, snapshot and target manifest must be fixed by the server. Export jobs are not connected.",
              "الصفحة الحالية تختلف عن جميع النتائج المطابقة. يجب أن يثبت الخادم الحقول والغرض وإجماليات العملات واللقطة وبيان الأهداف. مهام التصدير غير متصلة.",
            ]}
          />
        </Panel>
      )}
    </div>
  );
}
