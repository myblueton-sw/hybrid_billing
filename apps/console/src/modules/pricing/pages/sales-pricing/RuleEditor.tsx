import type { Locale } from "../../../../shared/types";
import { profile, type Draft, type Prepared } from "./model";
import { t, issueLabel } from "./copy";
export default function RuleEditor({
  locale: l,
  draft,
  source,
  update,
  errors,
}: {
  locale: Locale;
  draft: Draft;
  source?: Prepared;
  update: (d: Draft) => void;
  errors: string[];
}) {
  const fields: [keyof Draft, string, string, string][] = [
    ["name", "Policy review name", "정책 검토명", "اسم مراجعة السياسة"],
    [
      "unitPrice",
      "Selling unit price — manual input",
      "판매 단가 — 직접 입력",
      "سعر وحدة البيع — إدخال يدوي",
    ],
    [
      "denominator",
      "Quantity per price",
      "단가 기준 수량",
      "الكمية المقابلة للسعر",
    ],
    ["discount", "Review discount", "검토 할인율", "نسبة خصم المراجعة"],
    ["fee", "Additional fee amount", "추가 요금 금액", "مبلغ الرسوم الإضافية"],
  ];
  return (
    <section className="pricing-editor">
      <h2>
        {t(
          l,
          "Policy review inputs",
          "정책 검토 입력",
          "مدخلات مراجعة السياسة",
        )}
      </h2>
      <div className="pricing-fields">
        {fields.map(([key, en, ko, ar]) => (
          <label
            key={key}
            htmlFor={`pricing-${key === "unitPrice" ? "unit-price" : key}`}
            className={key === "name" ? "pricing-wide" : ""}
          >
            {t(l, en, ko, ar)}
            <span className="pricing-input">
              <input
                id={`pricing-${key === "unitPrice" ? "unit-price" : key}`}
                aria-invalid={errors.includes(
                  (
                    {
                      name: "name_required",
                      unitPrice: "price_invalid",
                      denominator: "denominator_invalid",
                      discount: "discount_invalid",
                      fee: "fee_invalid",
                    } as Record<string, string>
                  )[key],
                )}
                aria-describedby={`pricing-error-${key}`}
                type="text"
                dir={key === "name" ? undefined : "ltr"}
                inputMode={key === "name" ? "text" : "decimal"}
                value={draft[key]}
                maxLength={key === "name" ? 120 : 40}
                onChange={(e) => update({ ...draft, [key]: e.target.value })}
                disabled={!source}
              />
              {key !== "name" && (
                <bdi dir="ltr">
                  {key === "discount"
                    ? "%"
                    : key === "denominator"
                      ? source?.row.pricing_unit
                      : source?.row.currency}
                </bdi>
              )}
            </span>
            <span id={`pricing-error-${key}`} className="pricing-meta">
              {errors
                .filter(
                  (e) =>
                    e ===
                    (
                      {
                        name: "name_required",
                        unitPrice: "price_invalid",
                        denominator: "denominator_invalid",
                        discount: "discount_invalid",
                        fee: "fee_invalid",
                      } as Record<string, string>
                    )[key],
                )
                .map((e) => issueLabel(l, e))}
            </span>
            {key !== "name" && (
              <small>
                {key === "discount" ? (
                  <>
                    <bdi dir="ltr">0–100%</bdi> ·{" "}
                    {t(
                      l,
                      "Up to 4 decimal places",
                      "소수점 4자리까지",
                      "حتى 4 منازل عشرية",
                    )}
                  </>
                ) : key === "denominator" ? (
                  t(
                    l,
                    "Positive quantity in the actual source pricing unit",
                    "원천 요금 산정 단위의 양수 수량",
                    "كمية موجبة بوحدة تسعير المصدر الفعلية",
                  )
                ) : (
                  t(
                    l,
                    "Nonnegative · up to 18 decimal places · enter zero explicitly",
                    "0 이상 · 소수점 18자리까지 · 0도 직접 입력",
                    "غير سالب · حتى 18 منزلة عشرية · أدخل الصفر صراحةً",
                  )
                )}
              </small>
            )}
          </label>
        ))}
      </div>
      <p className="pricing-meta">
        {t(l, "Calculation profile", "산식 프로필", "ملف الحساب")}{" "}
        <bdi dir="ltr">{profile}</bdi>
      </p>
      <details>
        <summary>
          {t(
            l,
            "Unconnected pricing methods",
            "미연결 요금 방식",
            "طرق التسعير غير المتصلة",
          )}
        </summary>
        <p className="pricing-meta">
          {t(
            l,
            "Tiers, allowances, minimum/maximum, FX, tax, commitments, savings distribution, target margin and cost-plus require economic scope and authorized server policies.",
            "구간·포함량·최소/최대·환율·세금·약정·절감 배분·목표 마진·원가 가산은 경제적 범위와 승인된 서버 정책 연결이 필요합니다.",
            "تتطلب الشرائح والحصص والحدود وتحويل العملة والضريبة والالتزامات وتوزيع الوفورات والهامش والتكلفة المضافة نطاقاً اقتصادياً وسياسات خادم معتمدة.",
          )}
        </p>
      </details>
    </section>
  );
}
