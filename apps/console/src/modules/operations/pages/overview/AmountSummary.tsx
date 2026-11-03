import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { total, formatMoney, type Cost } from "./model";

export default function AmountSummary({
  rows,
  locale: l,
}: {
  rows: Cost[];
  locale: Locale;
}) {
  const known = rows.filter((r) => r.minor !== null).length;
  return (
    <section
      className="unified-summary overview-amounts"
      data-design-id="U01:summary:amount-meanings"
      aria-label={t(
        l,
        "Separate financial meanings",
        "금액별 구분",
        "معانٍ مالية منفصلة",
      )}
    >
      <div className="unified-total">
        <div className="overview-cost-heading">
          <h2>{t(l, "Source cost", "원천 비용", "تكلفة المصادر")}</h2>
          <span>
            {t(
              l,
              "Separate currency totals",
              "통화별 개별 합계",
              "إجماليات منفصلة حسب العملة",
            )}
          </span>
        </div>
        <div className="unified-money">
          {(["USD", "KRW"] as const).map((currency) => {
            const sum = total(rows, currency);
            return (
              <div key={currency} data-total={currency}>
                <bdi dir="ltr">
                  {sum.value === null
                    ? `${currency} · ${formatMoney(sum.value, currency, l)}`
                    : formatMoney(sum.value, currency, l)}
                </bdi>
                <small>
                  {!sum.expected
                    ? t(
                        l,
                        "No matching data",
                        "해당 자료 없음",
                        "لا بيانات مطابقة",
                      )
                    : sum.value === null
                      ? t(
                          l,
                          "Unknown · all records missing",
                          "미확인 · 전체 자료 누락",
                          "غير معلوم · جميع السجلات مفقودة",
                        )
                      : sum.missing
                        ? t(
                            l,
                            `Partial · ${sum.missing} missing`,
                            `부분 합계 · ${sum.missing}건 누락`,
                            `جزئي · ${sum.missing} مفقود`,
                          )
                        : t(l, "Known amount", "확인된 금액", "المبلغ المعروف")}
                </small>
              </div>
            );
          })}
        </div>
        <p className="overview-cost-coverage">
          {t(l, "Cost records", "비용 자료 확인", "سجلات التكلفة")}{" "}
          <strong data-coverage>
            <bdi dir="ltr">
              {known} / {rows.length}
            </bdi>
          </strong>
        </p>
      </div>
      <div data-amount-meaning="sales">
        <h2>{t(l, "Customer sales", "고객 판매", "مبيعات العملاء")}</h2>
        <strong className="overview-unknown">
          {t(l, "Not connected", "미연결", "غير متصل")}
        </strong>
        <small>
          {t(
            l,
            "Sales ledger unavailable",
            "판매 원장 미연결",
            "دفتر المبيعات غير متاح",
          )}
        </small>
      </div>
      <div data-amount-meaning="allocation">
        <h2>{t(l, "Internal allocation", "조직 배분", "التخصيص الداخلي")}</h2>
        <strong className="overview-unknown">
          {t(l, "Not connected", "미연결", "غير متصل")}
        </strong>
        <small>
          {t(
            l,
            "Allocation results unavailable",
            "배분 결과 미연결",
            "نتائج التخصيص غير متاحة",
          )}
        </small>
      </div>
      <div data-design-id="U01:state:receivables">
        <h2>{t(l, "Receivables", "미수·연체", "المستحقات")}</h2>
        <strong className="overview-unknown">
          {t(l, "Unknown", "미확인", "غير معلوم")}
        </strong>
        <small>
          {t(
            l,
            "Receipt allocations unavailable",
            "수납 배정 미연결",
            "تخصيصات التحصيل غير متاحة",
          )}
        </small>
      </div>
    </section>
  );
}
