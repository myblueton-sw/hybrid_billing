import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { OverviewData } from "../../../sources/collected-focus/api";
import { chartSeries, decimal, money, signedScale } from "./focus-values";
export default function FocusChart({
  data,
  currency,
  locale: l,
  onCurrency,
  onMonth,
}: {
  data: OverviewData;
  currency: string;
  locale: Locale;
  onCurrency: (v: string) => void;
  onMonth: (v: string) => void;
}) {
  const values = chartSeries(data, currency),
    axis = signedScale(values.map((v) => v.measure?.coefficient));
  const missing = t(l, "No collected value", "수집 값 없음", "لا قيمة مجمعة");
  return (
    <section className="unified-panel focus-trend">
      <header>
        <div>
          <h2>
            {t(
              l,
              "Monthly cost flow",
              "월별 비용 흐름",
              "اتجاه التكلفة الشهري",
            )}
          </h2>
          <p>
            {t(
              l,
              "Charge-start month · up to 12 months · gaps remain unknown",
              "과금 시작 월 · 최대 12개월 · 빈 기간은 미확인",
              "شهر بدء الرسوم · حتى 12 شهراً · الفجوات غير معلومة",
            )}
          </p>
        </div>
        <label>
          {t(l, "Chart currency", "차트 통화", "عملة الرسم")}
          <select
            id="focus-currency"
            value={currency}
            onChange={(e) => onCurrency(e.target.value)}
          >
            {data.options.currencies.map((c) => (
              <option key={c}>{c}</option>
            ))}
          </select>
        </label>
      </header>
      <div className="focus-chart" aria-hidden="true">
        <div className="focus-axis">
          {axis.ticks.map((n) => (
            <bdi key={n.toString()} dir="ltr" style={{ top: `${axis.y(n)}%` }}>
              {decimal(n.toString(), l)}
            </bdi>
          ))}
        </div>
        <div className="focus-plot">
          <div className="focus-grid">
            {axis.ticks.map((n) => (
              <span key={n.toString()} style={{ top: `${axis.y(n)}%` }} />
            ))}
          </div>
          <div
            className="focus-bars"
            style={{
              gridTemplateColumns: `repeat(${Math.max(values.length, 1)}, minmax(0,1fr))`,
            }}
          >
            {values.map((v) => {
              const n = v.measure?.coefficient,
                y = n == null ? axis.zero : axis.y(BigInt(n));
              return (
                <div
                  className="focus-month"
                  key={v.period}
                  data-value-state={
                    !v.measure
                      ? "absent"
                      : !v.measure.present
                        ? "unknown"
                        : v.measure.missing
                          ? "partial"
                          : "complete"
                  }
                >
                  <div className="focus-bar-slot">
                    {n != null && (
                      <span
                        className={`focus-bar ${v.measure?.missing ? "is-partial" : ""}`}
                        style={{
                          top: `${Math.min(y, axis.zero)}%`,
                          height:
                            n === "0" ? "2px" : `${Math.abs(y - axis.zero)}%`,
                        }}
                      />
                    )}
                    {n == null && (
                      <span
                        className="focus-gap"
                        style={{ top: `calc(${axis.zero}% - 4px)` }}
                      >
                        {v.measure
                          ? t(l, "Unknown", "미확인", "غير معلوم")
                          : "—"}
                      </span>
                    )}
                    {n != null && !!v.measure?.missing && (
                      <small
                        className="focus-partial"
                        style={{
                          top: `calc(${Math.min(y, axis.zero)}% - 4px)`,
                        }}
                      >
                        {t(l, "Partial", "부분", "جزئي")}
                      </small>
                    )}
                  </div>
                  <bdi>{v.period.slice(2)}</bdi>
                </div>
              );
            })}
          </div>
        </div>
      </div>
      <details>
        <summary>
          {t(
            l,
            "Exact monthly amounts",
            "월별 정확한 금액 보기",
            "المبالغ الشهرية الدقيقة",
          )}
        </summary>
        <div
          className="table-wrap"
          tabIndex={0}
          role="region"
          aria-label={t(l, "Monthly values", "월별 금액", "قيم شهرية")}
        >
          <table>
            <thead>
              <tr>
                <th>{t(l, "Month", "월", "الشهر")}</th>
                <th>{currency}</th>
                <th>{t(l, "Coverage", "값 상태", "التغطية")}</th>
              </tr>
            </thead>
            <tbody>
              {values.map((v) => (
                <tr key={v.period}>
                  <th>
                    {data.options.periods.includes(v.period) ? (
                      <button onClick={() => onMonth(v.period)}>
                        <bdi>{v.period}</bdi>
                      </button>
                    ) : (
                      <bdi>{v.period}</bdi>
                    )}
                  </th>
                  <td>
                    <bdi dir="ltr">
                      {money(v.measure?.coefficient ?? null, currency, l)}
                    </bdi>
                  </td>
                  <td>
                    {!v.measure
                      ? missing
                      : v.measure.present === 0
                        ? t(
                            l,
                            "Unknown · all values missing",
                            "미확인 · 모든 값 누락",
                            "غير معلوم · كل القيم مفقودة",
                          )
                        : v.measure.missing
                          ? t(
                              l,
                              `Partial · ${v.measure.missing} missing`,
                              `부분 · ${v.measure.missing}개 누락`,
                              `جزئي · ${v.measure.missing} مفقود`,
                            )
                          : t(
                              l,
                              "All values present",
                              "값 누락 없음",
                              "كل القيم موجودة",
                            )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
    </section>
  );
}
