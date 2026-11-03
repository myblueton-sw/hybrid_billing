import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { months, total, formatMoney, type Cost, type Currency } from "./model";
export default function CostTrend({
  rows,
  currency,
  locale: l,
  period,
  onMonth,
}: {
  rows: Cost[];
  currency: Currency;
  locale: Locale;
  period: string;
  onMonth: (month: string) => void;
}) {
  const values = months.map((month) => ({
    month,
    ...total(
      rows.filter((r) => r.month === month),
      currency,
    ),
  }));
  const selected = values.find((v) => v.month === period);
  const maximum = values.reduce(
    (max, x) => (x.value !== null && x.value > max ? x.value : max),
    0n,
  );
  return (
    <section className="unified-panel unified-trend">
      <header>
        <div>
          <h2>
            {t(
              l,
              "Monthly cost trend",
              "월별 비용 추이",
              "اتجاه التكلفة الشهري",
            )}
          </h2>
          <p>
            {t(
              l,
              "Reporting month · tax excluded · six months",
              "보고월 기준 · 세금 제외 · 최근 6개월",
              "شهر التقرير · باستثناء الضريبة · ستة أشهر",
            )}{" "}
            · <bdi>{currency}</bdi>
          </p>
        </div>
      </header>
      <p className="unified-trend-value">
        <bdi>
          {period} ·{" "}
          {formatMoney(
            values.find((v) => v.month === period)?.value ?? null,
            currency,
            l,
          )}
        </bdi>
        {" · "}
        <span data-trend-status>
          {!selected?.expected
            ? t(l, "No matching data", "해당 자료 없음", "لا بيانات مطابقة")
            : selected.value === null
              ? t(l, "Unknown", "미확인", "غير مؤكد")
              : selected.missing
                ? t(
                    l,
                    `Partial (${selected.missing} missing)`,
                    `부분 합계 (${selected.missing}건 누락)`,
                    `جزئي (${selected.missing} مفقود)`,
                  )
                : t(l, "Known amount", "확인된 금액", "المبلغ المعروف")}
        </span>
      </p>
      <div
        className="unified-chart"
        role="group"
        aria-label={t(l, "Select a month", "월 선택", "اختر شهراً")}
      >
        {values.map((v) => (
          <button
            key={v.month}
            className="unified-month"
            aria-pressed={v.month === period}
            onClick={() => onMonth(v.month)}
            aria-label={`${v.month} ${formatMoney(v.value, currency, l)} ${v.missing ? t(l, "Partial or unavailable", "부분 또는 미확인", "جزئي أو غير متاح") : ""}`}
          >
            <span className="unified-plot">
              <span
                className={v.missing ? "unified-bar partial" : "unified-bar"}
                style={{
                  height:
                    maximum && v.value !== null
                      ? `${Number((v.value * 10000n) / maximum) / 100}%`
                      : "0%",
                }}
              />
            </span>
            <bdi>{v.month.slice(5)}</bdi>
            <small>
              {v.expected === 0
                ? t(l, "No data", "자료 없음", "لا بيانات")
                : v.value === null
                  ? t(l, "Unknown", "미확인", "غير مؤكد")
                  : v.missing
                    ? t(l, "Partial", "부분", "جزئي")
                    : t(l, "Known", "확인", "متاح")}
            </small>
          </button>
        ))}
      </div>
      <p className="unified-note">
        {t(
          l,
          "Partial bars show known subtotals only. No forecast or currency conversion. Month selection retains the reporting-month basis.",
          "부분 막대는 확인된 금액만 표시합니다. 예측·환산은 하지 않습니다. 월 선택 시 보고월 기준을 유지합니다.",
          "تعرض الأعمدة الجزئية الإجمالي المعروف فقط، دون توقعات أو تحويل عملة. يظل الاختيار على أساس شهر التقرير.",
        )}
      </p>
      <details>
        <summary>
          {t(
            l,
            "View exact monthly amounts",
            "월별 정확한 금액 보기",
            "عرض المبالغ الشهرية الدقيقة",
          )}
        </summary>
        <div className="table-wrap">
          <table>
            <thead>
              <tr>
                <th>{t(l, "Month", "월", "الشهر")}</th>
                <th>{t(l, "Known cost", "확인된 비용", "التكلفة المعروفة")}</th>
                <th>
                  {t(
                    l,
                    "Known / expected records",
                    "확인 / 대상 자료",
                    "سجلات متاحة / متوقعة",
                  )}
                </th>
              </tr>
            </thead>
            <tbody>
              {values.map((v) => (
                <tr key={v.month}>
                  <th>
                    <bdi>{v.month}</bdi>
                  </th>
                  <td>
                    <bdi>{formatMoney(v.value, currency, l)}</bdi>
                  </td>
                  <td>
                    <bdi dir="ltr">
                      {v.known}/{v.expected}
                    </bdi>
                    {v.missing ? ` · ${t(l, "Partial", "부분", "جزئي")}` : ""}
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
