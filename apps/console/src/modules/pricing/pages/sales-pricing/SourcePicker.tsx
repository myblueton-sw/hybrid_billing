import type { Locale } from "../../../../shared/types";
import type {
  OverviewData,
  FocusRow,
} from "../../../sources/collected-focus/api";
import { t, missing } from "./copy";
import { number, money } from "./values";
export function Evidence({
  row: r,
  locale: l,
}: {
  row: FocusRow;
  locale: Locale;
}) {
  const values: [string, string | null, boolean][] = [
    [
      t(
        l,
        "Source list unit price",
        "원천 정가 단가",
        "سعر وحدة القائمة من المصدر",
      ),
      r.list_unit_price,
      true,
    ],
    [
      t(
        l,
        "Source contracted unit price",
        "원천 계약 단가",
        "سعر الوحدة المتعاقد عليه من المصدر",
      ),
      r.contracted_unit_price,
      true,
    ],
    [
      t(l, "Effective cost", "실효 비용", "التكلفة الفعلية"),
      r.effective_cost,
      true,
    ],
    [
      t(
        l,
        "Consumed quantity — unused",
        "사용 수량 — 계산 미사용",
        "الكمية المستهلكة — غير مستخدمة",
      ),
      r.consumed_quantity,
      false,
    ],
  ];
  return (
    <details>
      <summary>{t(l, "Source evidence", "원천 근거", "أدلة المصدر")}</summary>
      <dl className="pricing-evidence">
        {values.map(([label, v, isMoney], index) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>
              <bdi dir="ltr">
                {v === null
                  ? missing(l)
                  : isMoney
                    ? `${money(v, r.currency, l)}${index < 2 ? ` / ${r.pricing_unit?.trim() || missing(l)}` : ""}`
                    : `${number(v, l)} ${r.consumed_unit ?? ""}`}
              </bdi>
            </dd>
          </div>
        ))}
        <div>
          <dt>
            {t(
              l,
              "Source charge interval",
              "원천 과금 구간",
              "فترة رسوم المصدر",
            )}
          </dt>
          <dd>
            <bdi dir="ltr">
              {r.charge_start} — {r.charge_end}
            </bdi>
          </dd>
        </div>
        <div>
          <dt>{t(l, "Row ID", "행 ID", "معرّف الصف")}</dt>
          <dd>
            <bdi dir="ltr">{r.id}</bdi>
          </dd>
        </div>
      </dl>
      <p className="pricing-meta">
        {t(
          l,
          "Evidence only. These unit prices are never loaded as customer prices.",
          "근거 값이며 고객 단가로 불러오지 않습니다.",
          "أدلة فقط. لا تُحمّل هذه الأسعار كأسعار للعملاء.",
        )}
      </p>
    </details>
  );
}
export default function SourcePicker({
  locale: l,
  data,
  selected,
  select,
  page,
}: {
  locale: Locale;
  data: OverviewData;
  selected?: string;
  select: (id: string) => void;
  page: (n: number) => void;
}) {
  return (
    <section className="pricing-picker" id="pricing-source-list" tabIndex={-1}>
      <h2>
        {t(l, "Collected source rows", "수집 원천 행", "صفوف المصدر المجمّع")}
      </h2>
      <p className="pricing-meta">
        <bdi dir="ltr">{data.row_count}</bdi>{" "}
        {t(
          l,
          "matching rows · at most 25 per page",
          "조건에 맞는 행 · 페이지당 최대 25개",
          "صفاً مطابقاً · حتى 25 لكل صفحة",
        )}
      </p>
      {!data.rows.length ? (
        <p>
          {t(
            l,
            "No source rows match. No default selling price is applied.",
            "이 조건에 원천 행이 없습니다. 임의 판매 단가를 적용하지 않습니다.",
            "لا توجد صفوف مطابقة. لا يُطبّق سعر بيع افتراضي.",
          )}
        </p>
      ) : (
        <div
          className="pricing-table-scroll"
          role="region"
          aria-label={t(
            l,
            "Source row table",
            "원천 행 표",
            "جدول صفوف المصدر",
          )}
          tabIndex={0}
        >
          <table>
            <thead>
              <tr>
                {[
                  t(l, "Select", "선택", "اختيار"),
                  t(
                    l,
                    "Service / provider",
                    "서비스 / 공급자",
                    "الخدمة / المزوّد",
                  ),
                  t(
                    l,
                    "Pricing quantity / unit",
                    "요금 산정 수량 / 단위",
                    "كمية التسعير / الوحدة",
                  ),
                  t(l, "Billed cost", "청구 비용", "التكلفة المفوترة"),
                  t(
                    l,
                    "Evidence / eligibility",
                    "근거 / 검토 가능 여부",
                    "الأدلة / الأهلية",
                  ),
                ].map((h) => (
                  <th key={h}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.rows.map((r) => (
                <tr
                  key={r.id}
                  aria-current={selected === r.id ? "true" : undefined}
                >
                  <td>
                    <button
                      data-pricing-row={r.id}
                      aria-pressed={selected === r.id}
                      onClick={() => select(r.id)}
                    >
                      {selected === r.id
                        ? t(l, "Selected", "선택됨", "محدد")
                        : t(l, "Select row", "이 행 선택", "اختر الصف")}
                    </button>
                  </td>
                  <td>
                    {r.service ?? missing(l)}
                    <small>
                      <bdi dir="ltr">{r.provider}</bdi>
                    </small>
                  </td>
                  <td
                    data-label={t(
                      l,
                      "Pricing quantity / unit",
                      "요금 산정 수량 / 단위",
                      "كمية التسعير / الوحدة",
                    )}
                  >
                    <bdi dir="ltr">
                      {r.pricing_quantity === null
                        ? missing(l)
                        : number(r.pricing_quantity, l)}{" "}
                      {r.pricing_unit ?? missing(l)}
                    </bdi>
                  </td>
                  <td
                    data-label={t(
                      l,
                      "Billed cost",
                      "청구 비용",
                      "التكلفة المفوترة",
                    )}
                  >
                    <bdi dir="ltr">
                      {r.billed_cost === null
                        ? missing(l)
                        : money(r.billed_cost, r.currency, l)}
                    </bdi>
                  </td>
                  <td>
                    <span>
                      {r.pricing_quantity === null
                        ? t(
                            l,
                            "Pricing quantity missing",
                            "요금 산정 수량 없음",
                            "كمية التسعير مفقودة",
                          )
                        : !r.pricing_unit?.trim()
                          ? t(
                              l,
                              "Pricing unit missing",
                              "요금 산정 단위 없음",
                              "وحدة التسعير مفقودة",
                            )
                          : t(
                              l,
                              "Review available",
                              "검토 가능",
                              "المراجعة متاحة",
                            )}
                    </span>
                    <Evidence row={r} locale={l} />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <div className="pricing-pager">
        <button
          id="pricing-prev"
          disabled={data.after === 0}
          onClick={() => page(Math.max(0, data.after - 25))}
        >
          {t(l, "Previous", "이전", "السابق")}
        </button>
        <bdi dir="ltr">
          {data.rows.length ? data.after + 1 : 0}–
          {data.after + data.rows.length} / {data.row_count}
        </bdi>
        <button
          id="pricing-next"
          disabled={data.next_after === null}
          onClick={() => data.next_after !== null && page(data.next_after)}
        >
          {t(l, "Next", "다음", "التالي")}
        </button>
      </div>
    </section>
  );
}
