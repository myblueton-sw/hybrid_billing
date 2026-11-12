import type { Locale } from "../../../../shared/types";
import {
  decimal,
  comparable,
  profile,
  type Result,
  type Prepared,
} from "./model";
import { t, notCalculated } from "./copy";
import { money, number } from "./values";
import { fieldLabel } from "../../../sources/collected-focus/labels";
import { SourceAmount } from "./SourceValues";
export function Summary({
  locale: l,
  result,
}: {
  locale: Locale;
  result?: Result;
}) {
  return (
    <>
      <h2>
        {t(
          l,
          "Review amount before FX and tax",
          "환산·세금 적용 전 검토 금액",
          "مبلغ المراجعة قبل تحويل العملة والضريبة",
        )}
      </h2>
      <p
        className="pricing-meta"
        data-pricing-candidate={result ? "ready" : undefined}
      >
        {result
          ? t(
              l,
              "Calculated review candidate",
              "계산된 검토 후보",
              "مبلغ مرشح محسوب للمراجعة",
            )
          : notCalculated(l)}
      </p>
      <dl className="pricing-summary">
        {(["base", "discounted", "fee", "total"] as const).map((k) => (
          <div key={k}>
            <dt>
              {k === "base"
                ? t(l, "Base amount", "기본 금액", "المبلغ الأساسي")
                : k === "discounted"
                  ? t(
                      l,
                      "After review discount",
                      "검토 할인 후",
                      "بعد خصم المراجعة",
                    )
                  : k === "fee"
                    ? t(
                        l,
                        "Additional fee amount",
                        "추가 요금 금액",
                        "مبلغ الرسوم الإضافية",
                      )
                    : t(l, "Review amount", "검토 금액", "مبلغ المراجعة")}
            </dt>
            <dd>
              <bdi dir="ltr" data-pricing-total={k}>
                {result
                  ? money(
                      k === "fee" ? decimal(result.fee)!.toString() : result[k],
                      result.currency,
                      l,
                    )
                  : notCalculated(l)}
              </bdi>
            </dd>
          </div>
        ))}
      </dl>
      <p className="pricing-meta">
        {t(
          l,
          "FX and tax are unconnected. This is not a final selling amount.",
          "환율·세금 미연결. 최종 판매액이 아닙니다.",
          "تحويل العملة والضريبة غير متصلين. هذا ليس مبلغ بيع نهائياً.",
        )}
      </p>
    </>
  );
}
export default function Results({
  locale: l,
  source,
  name,
  result,
  baseline,
  view,
  keep,
  clear,
  back,
}: {
  locale: Locale;
  source: Prepared;
  name: string;
  result?: Result;
  baseline?: Result;
  view: string;
  keep: () => void;
  clear: () => void;
  back: () => void;
}) {
  if (view === "inheritance")
    return (
      <section>
        <h2>
          {t(
            l,
            "Current local review rule",
            "현재 로컬 검토 규칙",
            "قاعدة المراجعة المحلية الحالية",
          )}
        </h2>
        <dl className="pricing-evidence">
          <div>
            <dt>{t(l, "Rule name", "규칙 이름", "اسم القاعدة")}</dt>
            <dd>
              {name ||
                t(
                  l,
                  "Unnamed review draft",
                  "이름 없는 검토 초안",
                  "مسودة مراجعة بلا اسم",
                )}
            </dd>
          </div>
          <div>
            <dt>{t(l, "Applies to", "적용 대상", "ينطبق على")}</dt>
            <dd>
              {t(
                l,
                "One selected source row",
                "선택한 원천 행 1개",
                "صف مصدر محدد واحد",
              )}{" "}
              <bdi dir="ltr">{source.row.id}</bdi>
            </dd>
          </div>
          <div>
            <dt>
              {t(l, "Charge-start month", "과금 시작 월", "شهر بدء الرسوم")}
            </dt>
            <dd>
              <bdi dir="ltr">{source.row.charge_month}</bdi>
            </dd>
          </div>
        </dl>
        <h2>
          {t(
            l,
            "Unconnected policy bindings",
            "미연결 정책 관계",
            "روابط السياسات غير المتصلة",
          )}
        </h2>
        <p>
          {t(
            l,
            "Organization policies, customer contracts, precedence, effective-period overlap checks and server policy revisions are unconnected. Their absence has not been verified.",
            "조직 정책·고객 계약·우선순위·적용 기간 중첩 검사·서버 정책 개정은 미연결입니다. 상위 정책이 없다는 뜻은 아닙니다.",
            "سياسات المؤسسة وعقود العملاء والأولوية وفحوص تداخل الفترات ومراجعات الخادم غير متصلة. لم يُتحقق من غيابها.",
          )}
        </p>
      </section>
    );
  if (view === "formula")
    return (
      <section>
        <h2>{t(l, "Calculation steps", "계산 단계", "خطوات الحساب")}</h2>
        <p className="pricing-meta">
          <bdi dir="ltr">{profile}</bdi> ·{" "}
          {t(
            l,
            "18 decimal places · truncate toward zero · review profile only",
            "소수점 18자리 · 0 방향 절삭 · 검토 전용 프로필",
            "18 منزلة عشرية · اقتطاع باتجاه الصفر · ملف مراجعة فقط",
          )}
        </p>
        {!result ? (
          <p>
            {t(
              l,
              "Run a candidate preview to show the calculation steps.",
              "후보 미리보기를 실행하면 계산 단계가 표시됩니다.",
              "شغّل معاينة مرشح لعرض خطوات الحساب.",
            )}{" "}
            <button onClick={back}>
              {t(
                l,
                "Open pricing rules",
                "요금 규칙으로 이동",
                "فتح قواعد التسعير",
              )}
            </button>
          </p>
        ) : (
          <>
            <ol className="pricing-steps">
              <li>
                {t(
                  l,
                  "Pricing quantity / unit",
                  "요금 산정 수량 / 단위",
                  "كمية التسعير / الوحدة",
                )}
                :{" "}
                <bdi dir="ltr">
                  {number(source.row.pricing_quantity, l)} {result.unit}
                </bdi>
              </li>
              <li>
                {t(
                  l,
                  "Quantity × selling unit price ÷ quantity per price",
                  "수량 × 판매 단가 ÷ 단가 기준 수량",
                  "الكمية × سعر وحدة البيع ÷ الكمية المقابلة للسعر",
                )}
                :{" "}
                <bdi dir="ltr">
                  {number(source.row.pricing_quantity, l)} {result.unit} ×{" "}
                  {money(
                    decimal(result.unitPrice)!.toString(),
                    result.currency,
                    l,
                  )}{" "}
                  / ({result.denominator} {result.unit}) ={" "}
                  {money(result.base, result.currency, l)}
                </bdi>
              </li>
              <li>
                {t(
                  l,
                  "Apply review discount",
                  "검토 할인 적용",
                  "تطبيق خصم المراجعة",
                )}
                :{" "}
                <bdi dir="ltr">
                  {money(result.base, result.currency, l)} × (100% −{" "}
                  {result.discount}%) ={" "}
                  {money(result.discounted, result.currency, l)}
                </bdi>
              </li>
              <li>
                {t(
                  l,
                  "Add additional fee amount",
                  "추가 요금 금액 더하기",
                  "إضافة مبلغ الرسوم الإضافية",
                )}
                :{" "}
                <bdi dir="ltr">
                  {money(result.discounted, result.currency, l)} +{" "}
                  {money(decimal(result.fee)!.toString(), result.currency, l)} ={" "}
                  {money(result.total, result.currency, l)}
                </bdi>
              </li>
              <li>
                {t(
                  l,
                  "Review amount before FX and tax",
                  "환산·세금 적용 전 검토 금액",
                  "مبلغ المراجعة قبل تحويل العملة والضريبة",
                )}
                : <bdi dir="ltr">{money(result.total, result.currency, l)}</bdi>
              </li>
            </ol>
            <Summary locale={l} result={result} />
            <p className="pricing-meta">
              {t(
                l,
                "Scroll horizontally to inspect all integer values.",
                "가로로 스크롤하여 모든 정수 계산값을 확인하세요.",
                "مرّر أفقياً لفحص جميع القيم الصحيحة.",
              )}
            </p>
            <div
              className="pricing-table-scroll pricing-trace"
              role="region"
              tabIndex={0}
              aria-label={t(
                l,
                "Exact calculation trace",
                "정확한 계산 근거",
                "أثر الحساب الدقيق",
              )}
            >
              <table>
                <thead>
                  <tr>
                    {[
                      t(l, "Stage", "단계", "المرحلة"),
                      t(l, "Integer numerator", "정수 분자", "البسط الصحيح"),
                      t(l, "Integer denominator", "정수 분모", "المقام الصحيح"),
                      t(l, "Quotient", "몫", "خارج القسمة"),
                      t(
                        l,
                        "Raw signed remainder",
                        "부호 있는 정수 나머지",
                        "الباقي الصحيح بإشارته",
                      ),
                    ].map((x) => (
                      <th key={x}>{x}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {(["base", "discount"] as const).map((k) => (
                    <tr key={k}>
                      <th>
                        {k === "base"
                          ? t(
                              l,
                              "Quantity × price ÷ quantity per price",
                              "수량 × 단가 ÷ 단가 기준 수량",
                              "الكمية × السعر ÷ الكمية المقابلة للسعر",
                            )
                          : t(
                              l,
                              "Base × (100 − discount) ÷ 100",
                              "기본 금액 × (100 − 할인율) ÷ 100",
                              "الأساس × (100 − الخصم) ÷ 100",
                            )}
                      </th>
                      {(
                        [
                          "numerator",
                          "denominator",
                          "quotient",
                          "remainder",
                        ] as const
                      ).map((f) => (
                        <td key={f}>
                          <bdi dir="ltr">{result.trace[k][f]}</bdi>
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <p className="pricing-meta">
              {t(
                l,
                "Trace values are scaled integers. Remainders are not currency amounts. Nonzero remainders were truncated toward zero.",
                "계산 근거는 배율 적용 정수입니다. 나머지는 통화 금액이 아닙니다. 0이 아닌 나머지는 0 방향으로 절삭했습니다.",
                "قيم الأثر أعداد صحيحة بمقياس ثابت. البواقي ليست مبالغ نقدية. اقتُطعت البواقي غير الصفرية باتجاه الصفر.",
              )}
            </p>
          </>
        )}
      </section>
    );
  const same = !!(result && baseline && comparable(result, baseline));
  if (result && baseline && !same)
    return (
      <section>
        <h2>
          {t(l, "Comparison unavailable", "비교 불가", "المقارنة غير متاحة")}
        </h2>
        <p>
          {t(
            l,
            "The candidate and baseline contexts differ. Clear the baseline to continue.",
            "후보와 비교 기준의 범위가 다릅니다. 비교 기준을 지우세요.",
            "يختلف سياق المرشح والأساس. امسح الأساس للمتابعة.",
          )}
        </p>
        <button onClick={clear}>
          {t(l, "Clear baseline", "비교 기준 지우기", "مسح الأساس")}
        </button>
      </section>
    );
  return (
    <section>
      <h2>{t(l, "Amount comparison", "금액 비교", "مقارنة المبالغ")}</h2>
      <div className="pricing-actions">
        <button id="pricing-baseline" disabled={!result} onClick={keep}>
          {t(
            l,
            "Keep current candidate as session baseline",
            "현재 후보를 세션 비교 기준으로 유지",
            "حفظ المرشح الحالي أساساً للجلسة",
          )}
        </button>
        <button disabled={!baseline} onClick={clear}>
          {t(l, "Clear baseline", "비교 기준 지우기", "مسح الأساس")}
        </button>
      </div>
      <p className="pricing-meta">
        {t(
          l,
          "Session only. Comparison requires the same scope, page, source row, currency, pricing unit and calculation profile. pp means percentage points.",
          "현재 세션에서만 유지합니다. 같은 범위·페이지·원천 행·통화·요금 산정 단위·산식 프로필에서만 비교합니다. pp는 퍼센트포인트입니다.",
          "لهذه الجلسة فقط. تتطلب المقارنة النطاق والصفحة وصف المصدر والعملة ووحدة التسعير وملف الحساب نفسها. pp تعني نقاطاً مئوية.",
        )}
      </p>
      <div
        className="pricing-table-scroll pricing-comparison"
        role="region"
        tabIndex={0}
        aria-label={t(
          l,
          "Baseline comparison",
          "비교 기준 대조",
          "مقارنة الأساس",
        )}
      >
        <table>
          <thead>
            <tr>
              <th>{t(l, "Amount", "금액", "المبلغ")}</th>
              <th>
                {t(l, "Session baseline", "세션 비교 기준", "أساس الجلسة")}
              </th>
              <th>{t(l, "Current", "현재", "الحالي")}</th>
              <th>{t(l, "Change", "변화", "التغير")}</th>
            </tr>
          </thead>
          <tbody>
            {(["unitPrice", "denominator", "discount", "fee"] as const).map(
              (k) => {
                const label =
                  k === "unitPrice"
                    ? t(l, "Selling unit price", "판매 단가", "سعر وحدة البيع")
                    : k === "denominator"
                      ? t(
                          l,
                          "Quantity per price",
                          "단가 기준 수량",
                          "الكمية المقابلة للسعر",
                        )
                      : k === "discount"
                        ? t(l, "Review discount", "검토 할인율", "خصم المراجعة")
                        : t(
                            l,
                            "Additional fee amount",
                            "추가 요금 금액",
                            "مبلغ الرسوم الإضافية",
                          );
                const format = (r: Result) =>
                  k === "discount"
                    ? `${r[k]}%`
                    : k === "denominator"
                      ? `${r[k]} ${r.unit}`
                      : money(decimal(r[k])!.toString(), r.currency, l);
                const delta = same
                  ? decimal(result![k])! - decimal(baseline![k])!
                  : null;
                return (
                  <tr key={k}>
                    <th>{label}</th>
                    <td
                      data-label={t(
                        l,
                        "Session baseline",
                        "세션 비교 기준",
                        "أساس الجلسة",
                      )}
                    >
                      <bdi dir="ltr">
                        {baseline
                          ? format(baseline)
                          : t(l, "Not set", "미설정", "غير محدد")}
                      </bdi>
                    </td>
                    <td data-label={t(l, "Current", "현재", "الحالي")}>
                      <bdi dir="ltr">
                        {result ? format(result) : notCalculated(l)}
                      </bdi>
                    </td>
                    <td data-label={t(l, "Change", "변화", "التغير")}>
                      {delta === null ? (
                        t(l, "Not comparable", "비교 불가", "غير قابل للمقارنة")
                      ) : (
                        <>
                          <bdi dir="ltr">
                            {k === "unitPrice" || k === "fee"
                              ? money(
                                  delta.toString(),
                                  result!.currency,
                                  l,
                                  true,
                                )
                              : `${delta < 0n ? "−" : delta > 0n ? "+" : ""}${number((delta < 0n ? -delta : delta).toString(), l)} ${k === "discount" ? "pp" : result!.unit}`}
                          </bdi>
                          <small>
                            {delta > 0n
                              ? t(l, "Increase", "증가", "زيادة")
                              : delta < 0n
                                ? t(l, "Decrease", "감소", "انخفاض")
                                : t(l, "No change", "변동 없음", "دون تغيير")}
                          </small>
                        </>
                      )}
                    </td>
                  </tr>
                );
              },
            )}
            {(["base", "discounted", "total"] as const).map((k) => (
              <tr key={k}>
                <th>
                  {k === "base"
                    ? t(l, "Base", "기본 금액", "الأساس")
                    : k === "discounted"
                      ? t(l, "After discount", "할인 후", "بعد الخصم")
                      : t(l, "Review amount", "검토 금액", "مبلغ المراجعة")}
                </th>
                <td
                  data-label={t(
                    l,
                    "Session baseline",
                    "세션 비교 기준",
                    "أساس الجلسة",
                  )}
                >
                  <bdi dir="ltr">
                    {baseline
                      ? money(baseline[k], baseline.currency, l)
                      : t(l, "Not set", "미설정", "غير محدد")}
                  </bdi>
                </td>
                <td data-label={t(l, "Current", "현재", "الحالي")}>
                  <bdi dir="ltr">
                    {result
                      ? money(result[k], result.currency, l)
                      : notCalculated(l)}
                  </bdi>
                </td>
                <td data-label={t(l, "Change", "변화", "التغير")}>
                  {same ? (
                    <>
                      <bdi dir="ltr">
                        {money(
                          (
                            BigInt(result![k]) - BigInt(baseline![k])
                          ).toString(),
                          result!.currency,
                          l,
                          true,
                        )}
                      </bdi>
                      <small>
                        {BigInt(result![k]) > BigInt(baseline![k])
                          ? t(l, "Increase", "증가", "زيادة")
                          : BigInt(result![k]) < BigInt(baseline![k])
                            ? t(l, "Decrease", "감소", "انخفاض")
                            : t(l, "No change", "변동 없음", "دون تغيير")}
                      </small>
                    </>
                  ) : (
                    t(l, "Not comparable", "비교 불가", "غير قابل للمقارنة")
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <details>
        <summary>
          {t(l, "Source cost evidence", "원천 비용 근거", "أدلة تكلفة المصدر")}
        </summary>
        <dl className="pricing-evidence">
          {(["billed_cost", "effective_cost"] as const).map((basis) => (
            <div key={basis} data-pricing-field={basis}>
              <dt>{fieldLabel(l, basis)}</dt>
              <dd>
                <SourceAmount
                  value={source.row[basis]}
                  currency={source.row.currency}
                  note
                  locale={l}
                />
              </dd>
            </div>
          ))}
        </dl>
        <p className="pricing-meta">
          {t(
            l,
            "These source costs are not margin or savings.",
            "이 원천 비용은 마진·절감액이 아닙니다.",
            "هذه التكاليف ليست هامشاً أو وفورات.",
          )}
        </p>
      </details>
    </section>
  );
}
