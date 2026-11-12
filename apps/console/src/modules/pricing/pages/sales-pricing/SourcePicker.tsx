import { Fragment, useState } from "react";
import type { Locale } from "../../../../shared/types";
import type {
  OverviewData,
  FocusRow,
} from "../../../sources/collected-focus/api";
import { provenanceClass } from "../../../sources/collected-focus/fields";
import {
  fieldLabel,
  provenanceLabel,
} from "../../../sources/collected-focus/labels";
import { t } from "./copy";
import {
  Missing,
  SourceAmount,
  SourceInstant,
  SourceQuantity,
  SourceText,
} from "./SourceValues";
function EvidenceList({
  row: r,
  locale: l,
}: {
  row: FocusRow;
  locale: Locale;
}) {
  return (
    <>
      <dl className="pricing-evidence">
        <div data-pricing-field="list_unit_price">
          <dt>
            {t(
              l,
              "Source list unit price",
              "원천 정가 단가",
              "سعر وحدة القائمة من المصدر",
            )}
          </dt>
          <dd>
            <SourceAmount
              value={r.list_unit_price}
              currency={r.currency}
              unit={r.pricing_unit}
              note
              locale={l}
            />
          </dd>
        </div>
        <div data-pricing-field="contracted_unit_price">
          <dt>
            {t(
              l,
              "Source contracted unit price",
              "원천 계약 단가",
              "سعر الوحدة المتعاقد عليه من المصدر",
            )}
          </dt>
          <dd>
            <SourceAmount
              value={r.contracted_unit_price}
              currency={r.currency}
              unit={r.pricing_unit}
              note
              locale={l}
            />
          </dd>
        </div>
        <div data-pricing-field="billed_cost">
          <dt>{fieldLabel(l, "billed_cost")}</dt>
          <dd>
            <SourceAmount
              value={r.billed_cost}
              currency={r.currency}
              note
              locale={l}
            />
          </dd>
        </div>
        <div data-pricing-field="effective_cost">
          <dt>{fieldLabel(l, "effective_cost")}</dt>
          <dd>
            <SourceAmount
              value={r.effective_cost}
              currency={r.currency}
              note
              locale={l}
            />
          </dd>
        </div>
        <div data-pricing-field="consumed_quantity">
          <dt>
            {t(
              l,
              "Consumed quantity — unused",
              "사용 수량 — 계산 미사용",
              "الكمية المستهلكة — غير مستخدمة",
            )}
          </dt>
          <dd>
            <SourceQuantity
              value={r.consumed_quantity}
              unit={r.consumed_unit}
              note
              locale={l}
            />
          </dd>
        </div>
        <div data-pricing-field="charge_start">
          <dt>{fieldLabel(l, "charge_start")}</dt>
          <dd>
            <SourceInstant value={r.charge_start} locale={l} />
          </dd>
        </div>
        <div data-pricing-field="charge_end">
          <dt>{fieldLabel(l, "charge_end")}</dt>
          <dd>
            <SourceInstant value={r.charge_end} locale={l} />
          </dd>
        </div>
        <div data-pricing-field="billing_account">
          <dt>{fieldLabel(l, "billing_account")}</dt>
          <dd>
            <SourceText value={r.billing_account} locale={l} />
          </dd>
        </div>
        <div data-pricing-field="ordinal">
          <dt>{fieldLabel(l, "ordinal")}</dt>
          <dd>
            <bdi dir="ltr">{r.ordinal}</bdi>
          </dd>
        </div>
        <div data-pricing-field="evidence_class">
          <dt>{fieldLabel(l, "evidence_class")}</dt>
          <dd>{provenanceLabel(l, provenanceClass(r.evidence_class))}</dd>
        </div>
        <div data-pricing-field="sub_account">
          <dt>{fieldLabel(l, "sub_account")}</dt>
          <dd>
            <SourceText value={r.sub_account} locale={l} />
          </dd>
        </div>
        <div data-pricing-field="id">
          <dt>{fieldLabel(l, "id")}</dt>
          <dd>
            <bdi dir="ltr" className="pricing-code">
              <span>{r.id.slice(0, 32)}</span>
              <wbr />
              <span>{r.id.slice(32)}</span>
            </bdi>
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
    </>
  );
}
export function Evidence({
  row,
  locale: l,
}: {
  row: FocusRow;
  locale: Locale;
}) {
  return (
    <details>
      <summary>{t(l, "Source evidence", "원천 근거", "أدلة المصدر")}</summary>
      <EvidenceList row={row} locale={l} />
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
  const [opened, setOpened] = useState<{ key: string; id: string | null }>({
      key: "",
      id: null,
    }),
    pageKey = `${data.snapshot}:${data.after}`,
    openId = opened.key === pageKey ? opened.id : null;
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
                  t(l, "Source row", "원천 행", "صف المصدر"),
                  t(
                    l,
                    "Pricing quantity / unit",
                    "요금 산정 수량 / 단위",
                    "كمية التسعير / الوحدة",
                  ),
                  fieldLabel(l, "billed_cost"),
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
                <Fragment key={r.id}>
                  <tr
                    data-pricing-picker-row={r.id}
                    data-evidence-open={openId === r.id}
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
                      <span data-pricing-service>
                        {r.service ?? <Missing locale={l} />}
                      </span>
                      <small>
                        <bdi dir="ltr">{r.provider}</bdi>
                      </small>
                      <small data-pricing-account>
                        {fieldLabel(l, "billing_account")}:{" "}
                        <bdi dir="ltr">{r.billing_account}</bdi>
                      </small>
                      <small>
                        {fieldLabel(l, "sub_account")}:{" "}
                        <SourceText value={r.sub_account} locale={l} />
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
                      <SourceQuantity
                        value={r.pricing_quantity}
                        unit={r.pricing_unit}
                        locale={l}
                      />
                    </td>
                    <td data-label={fieldLabel(l, "billed_cost")}>
                      <SourceAmount
                        value={r.billed_cost}
                        currency={r.currency}
                        locale={l}
                      />
                    </td>
                    <td>
                      <span>
                        {r.pricing_quantity === null
                          ? t(
                              l,
                              "Pricing quantity missing",
                              "요금 산정 수량 누락",
                              "كمية التسعير مفقودة",
                            )
                          : !r.pricing_unit?.trim()
                            ? t(
                                l,
                                "Pricing unit missing",
                                "요금 산정 단위 누락",
                                "وحدة التسعير مفقودة",
                              )
                            : t(
                                l,
                                "Review available",
                                "검토 가능",
                                "المراجعة متاحة",
                              )}
                      </span>
                      <button
                        className="pricing-evidence-toggle"
                        data-pricing-evidence-toggle={r.id}
                        aria-expanded={openId === r.id}
                        aria-controls={`pricing-evidence-${r.id}`}
                        onClick={() =>
                          setOpened({
                            key: pageKey,
                            id: openId === r.id ? null : r.id,
                          })
                        }
                      >
                        {openId === r.id
                          ? t(
                              l,
                              "Hide source evidence",
                              "원천 근거 닫기",
                              "إخفاء أدلة المصدر",
                            )
                          : t(
                              l,
                              "Show source evidence",
                              "원천 근거 보기",
                              "عرض أدلة المصدر",
                            )}
                      </button>
                    </td>
                  </tr>
                  {openId === r.id && (
                    <tr
                      className="pricing-evidence-row"
                      id={`pricing-evidence-${r.id}`}
                      data-evidence-for={r.id}
                    >
                      <td colSpan={5}>
                        <EvidenceList row={r} locale={l} />
                      </td>
                    </tr>
                  )}
                </Fragment>
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
