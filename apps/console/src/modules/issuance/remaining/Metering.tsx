import { useState } from "react";
import { t, viewFields, type Props, type Text } from "./contracts";
import {
  Criteria,
  EmptyTable,
  Evidence,
  Intro,
  Notice,
  Panel,
  scopeInputs,
  useScreenState,
  type InputDef,
} from "./ui";
const inputs: readonly InputDef[] = [
  ...scopeInputs,
  {
    key: "asset",
    label: [
      "자산·미터 참조",
      "Asset or meter reference",
      "مرجع الأصل أو العداد",
    ],
    required: true,
    idOnly: true,
  },
  {
    key: "grade",
    label: [
      "측정등급 기준",
      "Measurement-grade criterion",
      "معيار درجة القياس",
    ],
    options: [
      { value: "measured", label: ["측정", "Measured", "مقاس"] },
      { value: "estimated", label: ["추정", "Estimated", "مقدر"] },
      { value: "mock", label: ["모의 측정", "Mock", "محاكاة"] },
      {
        value: "unmeasurable",
        label: ["측정불가", "Not measurable", "غير قابل للقياس"],
      },
    ],
  },
];
const lanes: readonly Text[] = [
  ["사양 변경", "Specification change", "تغيير المواصفات"],
  ["귀속 변경", "Attribution change", "تغيير الإسناد"],
  ["누락·reset", "Gaps and resets", "الفجوات وإعادة الضبط"],
  ["측정 구간", "Measurement interval", "فترة القياس"],
];
const measureHelp: Record<string, Text> = {
  cumulative: [
    "누적값은 reset과 수집 구간의 원천 근거가 필요합니다. 브라우저에서 차감해 사용량을 만들지 않습니다.",
    "Cumulative readings require reset and collection-interval evidence. The browser does not subtract them to create usage.",
    "القراءات التراكمية تتطلب أدلة إعادة الضبط وفترات الجمع. لا يطرح المتصفح القيم لإنشاء استخدام.",
  ],
  interval: [
    "구간합계는 시작 포함·끝 제외와 계약 시간대를 보존합니다. 겹친 구간을 임의 합산하지 않습니다.",
    "Interval totals preserve inclusive start, exclusive end and contract timezone. Overlapping intervals are not summed implicitly.",
    "تحافظ إجماليات الفترات على شمول البداية واستبعاد النهاية والمنطقة الزمنية للعقد. لا تُجمع الفترات المتداخلة ضمنياً.",
  ],
  instantaneous: [
    "순간값은 사용량 합계나 에너지량이 아닙니다. kW와 kWh, GB와 GiB를 암묵 변환하지 않습니다.",
    "Instantaneous readings are not usage totals or energy. kW/kWh and GB/GiB are not implicitly converted.",
    "القراءات اللحظية ليست إجماليات استخدام أو طاقة. لا تُحول kW وkWh أو GB وGiB ضمنياً.",
  ],
};
export default function Metering(props: Props) {
  const { locale: l, view, onDirty } = props;
  const [measurement, setMeasurement] = useState("cumulative");
  const [, onReview] = useScreenState(props, "B02", { measurement });
  return (
    <div
      className="fin-screen fin-metering"
      dir={l === "ar" ? "rtl" : "ltr"}
      data-finance-screen="B02"
      data-view={view}
    >
      <Intro
        l={l}
        purpose={[
          "당시 자산 사양·귀속과 계측 구간을 함께 확인합니다. 수집·측정 상태와 고객 과금 적격성은 별도입니다.",
          "Inspect historical asset specifications and attribution alongside metering intervals. Collection and measurement state are separate from customer billing eligibility.",
          "افحص مواصفات الأصول وإسنادها التاريخي مع فترات القياس. حالة الجمع والقياس منفصلة عن أهلية فوترة العميل.",
        ]}
      />
      <Criteria
        l={l}
        screenId="B02"
        inputs={inputs}
        onDirty={onDirty}
        onReview={onReview}
      />
      {view === "assets" ? (
        <>
          <Panel
            title={t(l, [
              "자산·귀속 이력",
              "Asset and attribution history",
              "سجل الأصول والإسناد",
            ])}
          >
            <EmptyTable
              l={l}
              screenId="B02"
              columns={viewFields.B02.assets}
              caption={[
                "과거 유효구간별 자산",
                "Assets by historical effective interval",
                "الأصول حسب فترات السريان التاريخية",
              ]}
            />
          </Panel>
          <Panel
            title={t(l, [
              "서로 다른 사건의 시간축",
              "Separate event lanes",
              "مسارات أحداث منفصلة",
            ])}
          >
            <ul className="fin-lanes">
              {lanes.map((lane, index) => (
                <li key={index}>
                  <strong>{t(l, lane)}</strong>
                  <p>
                    {t(l, [
                      "해당 구간의 원천 근거 미연결 · 사용량 0을 뜻하지 않음",
                      "Interval source evidence not connected; not zero usage",
                      "أدلة مصدر الفترة غير متصلة؛ لا يعني صفراً للاستخدام",
                    ])}
                  </p>
                </li>
              ))}
            </ul>
          </Panel>
        </>
      ) : view === "intervals" ? (
        <Panel
          title={t(l, [
            "측정 구간·수량·단위",
            "Measurement intervals, quantity and unit",
            "فترات القياس والكمية والوحدة",
          ])}
        >
          <EmptyTable
            l={l}
            screenId="B02"
            columns={viewFields.B02.intervals}
            caption={["계측 구간", "Metering intervals", "فترات القياس"]}
          />
          <Notice
            l={l}
            text={[
              "현재 사양으로 과거를 보간하지 않습니다. 승인된 추정 정책·수량·단위가 없으면 계산 가능한 사용량으로 제시하지 않습니다.",
              "Current specifications do not interpolate history. Without an approved estimation policy, quantity and unit, usage is not presented as calculation-ready.",
              "لا تُستوفى البيانات التاريخية بالمواصفات الحالية. دون سياسة تقدير معتمدة وكمية ووحدة لا يُعرض الاستخدام كجاهز للحساب.",
            ]}
          />
        </Panel>
      ) : view === "gaps" ? (
        <div className="fin-two">
          <Panel
            title={t(l, [
              "누락·reset 근거",
              "Gap and reset evidence",
              "أدلة الفجوات وإعادة الضبط",
            ])}
          >
            <Evidence l={l} screenId="B02" keys={viewFields.B02.gaps} />
          </Panel>
          <Panel
            title={t(l, [
              "완전성과 계산 적격성",
              "Completeness and calculation eligibility",
              "الاكتمال وأهلية الحساب",
            ])}
          >
            <p>
              {t(l, [
                "허용 gap·수집 주기·추정 허용 범위는 아직 확인되지 않았습니다. 모의 측정·측정불가·누락을 실제 0이나 청구 가능으로 바꾸지 않습니다.",
                "Allowed gaps, collection cadence and estimation scope are not verified. Mock, unmeasurable and missing readings are not converted to zero or billing eligibility.",
                "الفجوات المسموحة ودورة الجمع ونطاق التقدير غير متحققة. قراءات المحاكاة وغير القابلة للقياس والمفقودة لا تتحول إلى صفر أو أهلية فوترة.",
              ])}
            </p>
            <Notice
              l={l}
              text={[
                "진단 실행과 누락 검토 등록은 업무 서비스 연결 후 별도 권한으로 수행합니다.",
                "Diagnostics and gap-review registration require connected services and separate authority.",
                "تنفيذ التشخيص وتسجيل مراجعة الفجوات يتطلبان خدمات متصلة وصلاحيات منفصلة.",
              ]}
            />
          </Panel>
        </div>
      ) : (
        <div className="fin-two">
          <Panel
            title={t(l, [
              "모델·미터와 단가 분모",
              "Models, meters and rate denominators",
              "النماذج والعدادات ومقامات الأسعار",
            ])}
          >
            <Evidence l={l} screenId="B02" keys={viewFields.B02.meters} />
          </Panel>
          <Panel
            title={t(l, [
              "측정 방식의 의미",
              "Measurement semantics",
              "دلالات القياس",
            ])}
          >
            <label className="fin-purpose">
              {t(l, [
                "설명할 측정 종류",
                "Measurement kind to inspect",
                "نوع القياس المراد فحصه",
              ])}
              <select
                value={measurement}
                onChange={(e) => setMeasurement(e.target.value)}
                data-item="B02:input:measurement-kind"
              >
                <option value="cumulative">
                  {t(l, ["누적값", "Cumulative", "تراكمي"])}
                </option>
                <option value="interval">
                  {t(l, ["구간합계", "Interval total", "إجمالي الفترة"])}
                </option>
                <option value="instantaneous">
                  {t(l, ["순간값", "Instantaneous", "لحظي"])}
                </option>
              </select>
            </label>
            <p>{t(l, measureHelp[measurement])}</p>
            <Notice
              l={l}
              text={[
                "모델 별칭과 1천·100만 단가 분모는 해당 시점의 계약 근거가 필요합니다. AI 성공·실패·취소·스트리밍 완료도 과금 적격성과 별도로 확인합니다.",
                "Model aliases and per-thousand/per-million rate denominators require time-specific contract evidence. AI success, failure, cancellation and streaming completion are checked separately from billing eligibility.",
                "أسماء النماذج ومقامات السعر للألف أو المليون تتطلب أدلة عقد خاصة بالوقت. نجاح الذكاء الاصطناعي وفشله وإلغاؤه واكتمال البث تُفحص منفصلة عن أهلية الفوترة.",
              ]}
            />
          </Panel>
        </div>
      )}
      <p className="fin-meta">
        {t(l, [
          "원천 상세·연결 진단·누락 검토는 같은 허용 범위와 고정 참조가 있어야 연결할 수 있습니다. 아직 선택된 원천은 없습니다.",
          "Source detail, connection diagnostics and gap review require the same permitted scope and fixed reference. No source is currently selected.",
          "تفاصيل المصدر وتشخيص الاتصال ومراجعة الفجوات تتطلب النطاق المسموح نفسه ومرجعاً ثابتاً. لا يوجد مصدر مختار حالياً.",
        ])}
      </p>
    </div>
  );
}
