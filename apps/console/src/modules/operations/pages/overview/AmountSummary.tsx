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
  return (
    <section
      className="unified-summary overview-amounts"
      data-design-id="U01:summary:amount-meanings"
      aria-label={t(
        l,
        "Three separate amount meanings",
        "구분된 세 가지 금액",
        "ثلاثة معانٍ مالية منفصلة",
      )}
    >
      <div className="unified-total">
        <h2>{t(l, "Source cost", "원천 비용", "تكلفة المصادر")}</h2>
        <p>
          {t(
            l,
            "Collected source costs, by currency",
            "수집된 원천 비용 · 통화별 합계",
            "تكاليف المصادر المجمعة حسب العملة",
          )}
        </p>
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
                  {sum.expected > 0 && sum.value === null
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
                      : sum.expected
                        ? t(l, "Known amount", "확인된 금액", "المبلغ المعروف")
                        : t(
                            l,
                            "No matching data",
                            "해당 자료 없음",
                            "لا بيانات مطابقة",
                          )}
                </small>
              </div>
            );
          })}
        </div>
        <p className="overview-meaning-note">
          {t(
            l,
            "Tax excluded · no FX conversion · not a final settlement",
            "세금 제외 · 환산 없음 · 최종 정산액 아님",
            "باستثناء الضريبة · دون تحويل · ليست تسوية نهائية",
          )}
        </p>
      </div>
      <div data-amount-meaning="sales">
        <h2>{t(l, "Customer sales", "고객 판매 금액", "مبيعات العملاء")}</h2>
        <p>
          {t(
            l,
            "Amounts sold to external customers",
            "외부 고객에게 판매한 금액",
            "مبالغ المبيعات للعملاء الخارجيين",
          )}
        </p>
        <strong className="overview-unknown">
          {t(l, "Not connected", "미연결", "غير متصل")}
        </strong>
        <p>
          {t(
            l,
            "Sales ledger unavailable. Expected and confirmed amounts remain unknown.",
            "판매 원장이 없어 예상·확정 금액을 확인할 수 없습니다.",
            "دفتر المبيعات غير متاح. المبالغ المتوقعة والمؤكدة غير معلومة.",
          )}
        </p>
      </div>
      <div data-amount-meaning="allocation">
        <h2>
          {t(l, "Internal allocation", "조직 배분 금액", "التخصيص الداخلي")}
        </h2>
        <p>
          {t(
            l,
            "Costs assigned to internal organizations",
            "내부 조직에 배분한 비용",
            "التكاليف المخصصة للمؤسسات الداخلية",
          )}
        </p>
        <strong className="overview-unknown">
          {t(l, "Not connected", "미연결", "غير متصل")}
        </strong>
        <p>
          {t(
            l,
            "Allocation results unavailable. Source cost is not an allocation amount.",
            "배분 결과가 없어 금액을 확인할 수 없습니다. 원천 비용과는 별개입니다.",
            "نتائج التخصيص غير متاحة. تكلفة المصادر ليست مبلغ التخصيص.",
          )}
        </p>
      </div>
    </section>
  );
}
