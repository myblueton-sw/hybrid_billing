import { useEffect, useRef } from "react";
import type { Locale } from "../../../../shared/types";
import {
  fieldLabel,
  Instant,
  missingLabel,
} from "../../../sources/collected-focus/labels";
import type { OverviewData, FocusRow } from "./api";
import { t, basisLabel } from "./copy";
import { AmountValue, RowAmount } from "./Amounts";
import SourceDetail from "./SourceDetail";
export function Totals({
  locale: l,
  data: d,
}: {
  locale: Locale;
  data: OverviewData;
}) {
  return (
    <section className="b08-totals">
      <h2>
        {t(
          l,
          "Current source input",
          "현재 원천 입력",
          "مدخلات المصدر الحالية",
        )}
      </h2>
      <p className="b08-meta">
        {t(
          l,
          "Full-selection source costs, not customer billing amounts. Pages and account-search text do not change these totals.",
          "전체 조회 범위의 원천 비용이며 고객 청구 금액이 아닙니다. 페이지·계정 검색어는 이 합계를 바꾸지 않습니다.",
          "تكاليف المصدر لكامل الاختيار وليست مبالغ فوترة العملاء. الصفحات ونص بحث الحسابات لا يغيّران هذه الإجماليات.",
        )}
      </p>
      {d.amounts.length ? (
        <div className="b08-total-list">
          {d.amounts.map((a) => (
            <dl key={a.currency}>
              <div>
                <dt>
                  {t(l, "Source rows", "원천 행", "صفوف المصدر")}{" "}
                  <bdi dir="ltr">{a.currency}</bdi>
                </dt>
                <dd>
                  <bdi dir="ltr">{a.rows}</bdi>
                </dd>
              </div>
              {(["billed_cost", "effective_cost"] as const).map((b) => (
                <div key={b}>
                  <dt>{basisLabel(l, b)}</dt>
                  <dd>
                    <AmountValue
                      value={a[b]}
                      currency={a.currency}
                      locale={l}
                    />
                  </dd>
                </div>
              ))}
            </dl>
          ))}
        </div>
      ) : (
        <p>
          {t(
            l,
            "No source rows observed in this selection",
            "현재 조회 범위에서 원천 행이 관측되지 않았습니다",
            "لم تُرصد صفوف مصدر ضمن هذا الاختيار",
          )}
        </p>
      )}
    </section>
  );
}
export default function SourceInputs({
  locale: l,
  data: d,
  selected,
  select,
  page,
}: {
  locale: Locale;
  data: OverviewData;
  selected: FocusRow | undefined;
  select: (id: string | null) => void;
  page: (n: number) => void;
}) {
  const root = useRef<HTMLDivElement>(null),
    last = useRef<string | null>(null);
  useEffect(() => {
    if (selected && last.current !== selected.id) {
      [
        ...root.current!.querySelectorAll<HTMLElement>(
          "[data-b08-detail-heading]",
        ),
      ]
        .find((e) => e.getClientRects().length)
        ?.focus();
    } else if (!selected && last.current) {
      const id = last.current;
      requestAnimationFrame(() =>
        [...root.current!.querySelectorAll<HTMLButtonElement>("[data-b08-row]")]
          .find((e) => e.dataset.b08Row === id && e.getClientRects().length)
          ?.focus(),
      );
    }
    last.current = selected?.id ?? null;
  }, [selected?.id]);
  const button = (r: FocusRow) => (
    <button
      data-b08-row={r.id}
      aria-expanded={selected?.id === r.id}
      onClick={() => select(selected?.id === r.id ? null : r.id)}
    >
      {selected?.id === r.id
        ? t(l, "Close details", "상세 닫기", "إغلاق التفاصيل")
        : t(l, "Inspect source", "원천 근거 보기", "فحص المصدر")}
    </button>
  );
  return (
    <div className={`b08-workspace ${selected ? "has-detail" : ""}`} ref={root}>
      <section className="b08-input-list">
        <header>
          <h2>
            {t(l, "Source input rows", "원천 입력 명세", "صفوف مدخلات المصدر")}
          </h2>
          <bdi className="b08-count" dir="ltr">
            {d.rows.length ? d.after + 1 : 0}–{d.after + d.rows.length} /{" "}
            {d.row_count}
          </bdi>
        </header>
        {!d.rows.length ? (
          <p>
            {t(
              l,
              "No source rows in this selection",
              "이 조회 범위에 원천 행이 없습니다",
              "لا صفوف مصدر ضمن هذا الاختيار",
            )}
          </p>
        ) : (
          <>
            <div className="b08-mobile-list">
              {d.rows.map((r) => (
                <article key={r.id}>
                  <h3>
                    {r.service ?? (
                      <>
                        <small>{fieldLabel(l, "service")}</small>
                        <strong className="b08-state" data-b08-value="missing">
                          {missingLabel(l)}
                        </strong>
                      </>
                    )}
                  </h3>
                  <dl className="b08-facts">
                    <div>
                      <dt>{fieldLabel(l, "provider")}</dt>
                      <dd>
                        <bdi dir="ltr">{r.provider}</bdi>
                      </dd>
                    </div>
                    {(["billed_cost", "effective_cost"] as const).map((b) => (
                      <div key={b}>
                        <dt>{basisLabel(l, b)}</dt>
                        <dd>
                          <RowAmount
                            value={r[b]}
                            currency={r.currency}
                            locale={l}
                          />
                        </dd>
                      </div>
                    ))}
                  </dl>
                  {button(r)}
                  {selected?.id === r.id && (
                    <SourceDetail
                      locale={l}
                      row={r}
                      close={() => select(null)}
                    />
                  )}
                </article>
              ))}
            </div>
            <details className="b08-full-table" open={window.innerWidth >= 768}>
              <summary>
                {t(
                  l,
                  "Show full-column table",
                  "전체 열 표 보기",
                  "عرض جدول جميع الأعمدة",
                )}
              </summary>
              <div
                className="b08-table-region"
                role="region"
                tabIndex={0}
                aria-label={t(
                  l,
                  "Source input rows table",
                  "원천 입력 명세 표",
                  "جدول صفوف مدخلات المصدر",
                )}
              >
                <table>
                  <caption>
                    {t(
                      l,
                      "Source input rows, bounded page",
                      "원천 입력 명세 · 현재 페이지",
                      "صفوف مدخلات المصدر، صفحة محدودة",
                    )}
                  </caption>
                  <thead>
                    <tr>
                      <th scope="col">
                        {t(l, "Source charge", "원천 청구", "رسوم المصدر")}
                      </th>
                      <th scope="col">
                        {fieldLabel(l, "billing_account")}
                        <small>
                          {fieldLabel(l, "charge_start")}{" "}
                          <bdi dir="ltr">(UTC)</bdi>
                        </small>
                      </th>
                      {(["billed_cost", "effective_cost"] as const).map((b) => (
                        <th className="money" scope="col" key={b}>
                          {basisLabel(l, b)}
                        </th>
                      ))}
                      <th scope="col">{t(l, "Details", "상세", "التفاصيل")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {d.rows.map((r) => (
                      <tr key={r.id}>
                        <td>
                          {r.service ?? (
                            <strong className="b08-state">
                              {missingLabel(l)}
                            </strong>
                          )}
                          <small>
                            <bdi dir="ltr">{r.provider}</bdi>
                          </small>
                        </td>
                        <td>
                          <bdi dir="ltr">{r.billing_account}</bdi>
                          <small>
                            <Instant value={r.charge_start} />
                          </small>
                        </td>
                        {(["billed_cost", "effective_cost"] as const).map(
                          (b) => (
                            <td className="money" key={b}>
                              <RowAmount
                                value={r[b]}
                                currency={r.currency}
                                locale={l}
                              />
                            </td>
                          ),
                        )}
                        <td>{button(r)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </details>
          </>
        )}
        <div className="b08-pager">
          <button
            id="b08-prev"
            disabled={d.after === 0}
            onClick={() => page(Math.max(0, d.after - 25))}
          >
            {t(l, "Previous rows", "이전 행", "الصفوف السابقة")}
          </button>
          <button
            id="b08-next"
            disabled={d.next_after === null}
            onClick={() => d.next_after !== null && page(d.next_after)}
          >
            {t(l, "Next rows", "다음 행", "الصفوف التالية")}
          </button>
        </div>
      </section>
      {selected && (
        <aside className="b08-desktop-detail">
          <SourceDetail locale={l} row={selected} close={() => select(null)} />
        </aside>
      )}
    </div>
  );
}
