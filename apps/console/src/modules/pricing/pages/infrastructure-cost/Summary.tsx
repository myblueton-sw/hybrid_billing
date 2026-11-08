import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type {
  OverviewData,
  Measure,
} from "../../../sources/collected-focus/api";
import { money } from "../../../sources/collected-focus/values";
import { preview, type Draft } from "./model";
export default function Summary({
  locale: l,
  data,
  draft,
}: {
  locale: Locale;
  data: OverviewData;
  draft: Draft;
}) {
  const currencies = [
    ...new Set(
      data.rows
        .filter((r) => draft.entries.some((e) => e.row === r.id))
        .map((r) => r.currency),
    ),
  ];
  function amount(value: Measure | null, currency: string) {
    return (
      <>
        <bdi dir="ltr">
          {value?.coefficient === null || !value
            ? t(l, "No value", "값 없음", "لا توجد قيمة")
            : money(value.coefficient, currency, l)}
        </bdi>
        {!!value?.missing && (
          <small>
            {t(l, "Missing rows", "값 없는 행", "صفوف بلا قيمة")}:{" "}
            {value.missing}
          </small>
        )}
      </>
    );
  }
  return (
    <section className="infra-totals" aria-labelledby="infra-total-title">
      <h2 id="infra-total-title">
        {t(l, "Review amounts", "검토 금액", "مبالغ المراجعة")}
      </h2>
      <p className="infra-meta">
        {t(
          l,
          "Selected source-row subtotals for this review draft. Declared inclusion is unverified; no approved infrastructure cost is established.",
          "선택한 원천 행의 검토용 합계입니다. 선언한 포함 관계는 검증되지 않았으며 인프라 원가 승인 근거가 되지 않습니다.",
          "مجاميع صفوف المصدر المحدّدة لهذه المسودة. علاقات التضمين المُعلنة غير متحقّق منها ولا تثبت تكلفة بنية تحتية معتمدة.",
        )}
      </p>
      {!currencies.length ? (
        <p>
          {t(
            l,
            "Select source rows to review their amounts.",
            "원천 행을 선택하면 검토 금액을 볼 수 있습니다.",
            "حدّد صفوف المصدر لعرض مبالغ المراجعة.",
          )}
        </p>
      ) : (
        <div className="infra-currency-grid">
          {currencies.map((currency) => {
            const subset = {
              ...draft,
              entries: draft.entries.filter((e) =>
                data.rows.some(
                  (r) => r.id === e.row && r.currency === currency,
                ),
              ),
            };
            const values = preview(subset, data);
            const allErrors = preview(draft, data).errors;
            const relationshipsInvalid = allErrors.some(
              (e) => e !== "mixed_currency",
            );
            return (
              <dl key={currency} className="infra-currency">
                <dt>{t(l, "Currency", "통화", "العملة")}</dt>
                <dd>
                  <bdi dir="ltr">{currency}</bdi>
                </dd>
                <dt>
                  {t(
                    l,
                    "Selected source subtotal",
                    "선택 원천 합계",
                    "المجموع الفرعي للمصادر المحدّدة",
                  )}
                </dt>
                <dd data-total="gross">{amount(values.gross, currency)}</dd>
                <dt>
                  {t(
                    l,
                    "Declared included-row subtotal",
                    "포함 선언 행 합계",
                    "المجموع الفرعي للصفوف المُعلنة مشمولة",
                  )}
                </dt>
                <dd data-total="excluded">
                  {relationshipsInvalid
                    ? t(
                        l,
                        "Unavailable: check relationships",
                        "관계 확인 전 산정 불가",
                        "غير متاح: تحقّق من العلاقات",
                      )
                    : subset.entries.some((e) => e.parent)
                      ? amount(values.excluded, currency)
                      : t(
                          l,
                          "No declared included rows",
                          "포함 선언 행 없음",
                          "لا توجد صفوف مُعلنة مشمولة",
                        )}
                </dd>
                <dt>
                  {t(
                    l,
                    "Review subtotal after declared exclusions",
                    "포함 선언 제외 후 검토 합계",
                    "المجموع الفرعي للمراجعة بعد الاستبعادات المُعلنة",
                  )}
                </dt>
                <dd className="infra-net" data-total="review">
                  {relationshipsInvalid
                    ? t(
                        l,
                        "Unavailable: check relationships",
                        "관계 확인 전 산정 불가",
                        "غير متاح: تحقّق من العلاقات",
                      )
                    : amount(values.review, currency)}
                </dd>
              </dl>
            );
          })}
        </div>
      )}
      <p className="infra-meta">
        {t(
          l,
          "Signed values and source precision are preserved. Missing amounts are excluded from known subtotals. Currencies are never combined.",
          "부호와 원천 정밀도를 유지합니다. 값 없는 금액은 알려진 값의 합계에서 제외하며 통화는 합산하지 않습니다.",
          "تُحفظ الإشارات ودقة المصدر. تُستبعد المبالغ المفقودة من مجاميع القيم المعروفة ولا تُجمع العملات.",
        )}
      </p>
    </section>
  );
}
