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
import {
  Missing,
  SourceAmount,
  SourceInstant,
  SourceText,
} from "../../../sources/collected-focus/SourceValues";
import { t } from "./copy";
function Evidence({ locale: l, row: r }: { locale: Locale; row: FocusRow }) {
  return (
    <dl className="b05-evidence">
      {(["billed_cost", "effective_cost"] as const).map((basis) => (
        <div key={basis} data-fx-field={basis}>
          <dt>{fieldLabel(l, basis)}</dt>
          <dd>
            <SourceAmount
              value={r[basis]}
              currency={r.currency}
              note
              locale={l}
            />
          </dd>
        </div>
      ))}
      {(["charge_start", "charge_end"] as const).map((field) => (
        <div key={field} data-fx-field={field}>
          <dt>{fieldLabel(l, field)}</dt>
          <dd>
            <SourceInstant value={r[field]} locale={l} />
          </dd>
        </div>
      ))}
      <div data-fx-field="billing_account">
        <dt>{fieldLabel(l, "billing_account")}</dt>
        <dd>
          <SourceText value={r.billing_account} locale={l} />
        </dd>
      </div>
      <div data-fx-field="ordinal">
        <dt>{fieldLabel(l, "ordinal")}</dt>
        <dd>
          <bdi dir="ltr">{r.ordinal}</bdi>
        </dd>
      </div>
      <div data-fx-field="evidence_class">
        <dt>{fieldLabel(l, "evidence_class")}</dt>
        <dd>{provenanceLabel(l, provenanceClass(r.evidence_class))}</dd>
      </div>
      <div data-fx-field="sub_account">
        <dt>{fieldLabel(l, "sub_account")}</dt>
        <dd>
          <SourceText value={r.sub_account} locale={l} />
        </dd>
      </div>
      <div data-fx-field="id">
        <dt>{fieldLabel(l, "id")}</dt>
        <dd>
          <bdi dir="ltr" className="b05-code">
            <span>{r.id.slice(0, 32)}</span>
            <wbr />
            <span>{r.id.slice(32)}</span>
          </bdi>
        </dd>
      </div>
    </dl>
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
    openId = opened.key === pageKey ? opened.id : null,
    basis = data.selection.basis,
    headings = [
      t(l, "Select", "선택", "اختيار"),
      t(l, "Source row", "원천 행", "صف المصدر"),
      t(l, "Source cost", "원천 비용", "تكلفة المصدر"),
      t(l, "Review availability", "검토 가능 여부", "إتاحة المراجعة"),
    ];
  return (
    <section id="fx-source-list" className="b05-picker" tabIndex={-1}>
      <h2>
        {t(l, "Collected source rows", "수집 원천 행", "صفوف المصدر المجمّع")}
      </h2>
      <p className="b05-meta">
        <bdi dir="ltr">{data.row_count}</bdi>{" "}
        {t(
          l,
          "matching rows · up to 25 per page",
          "조건에 맞는 행 · 페이지당 최대 25개",
          "صفوف مطابقة · حتى 25 لكل صفحة",
        )}
      </p>
      {!data.rows.length ? (
        <p>
          {t(
            l,
            "No matching source rows. Missing amounts are not replaced with zero.",
            "조건에 맞는 원천 행이 없습니다. 0으로 대체하지 않습니다.",
            "لا توجد صفوف مصدر مطابقة. لا تُستبدل المبالغ المفقودة بصفر.",
          )}
        </p>
      ) : (
        <div
          className="b05-source-table b05-scroll"
          role="region"
          tabIndex={0}
          aria-label={t(l, "Source rows", "원천 행", "صفوف المصدر")}
        >
          <table>
            <thead>
              <tr>
                {headings.map((h) => (
                  <th key={h}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {data.rows.map((r) => (
                <Fragment key={r.id}>
                  <tr
                    data-fx-picker-row={r.id}
                    data-evidence-open={openId === r.id}
                    aria-current={selected === r.id ? "true" : undefined}
                  >
                    <td>
                      <button
                        data-fx-row={r.id}
                        aria-pressed={selected === r.id}
                        onClick={() => select(r.id)}
                      >
                        {selected === r.id
                          ? t(l, "Selected", "선택됨", "محدد")
                          : t(l, "Select row", "이 행 선택", "اختر الصف")}
                      </button>
                    </td>
                    <td data-label={headings[1]}>
                      <span className="b05-service" data-fx-service>
                        {r.service ?? <Missing locale={l} />}
                      </span>
                      <small>
                        <bdi dir="ltr">{r.provider}</bdi>
                      </small>
                      <small data-fx-row-account>
                        {fieldLabel(l, "billing_account")}:{" "}
                        <bdi dir="ltr">{r.billing_account}</bdi>
                      </small>
                      <small data-fx-subaccount>
                        {fieldLabel(l, "sub_account")}:{" "}
                        <SourceText value={r.sub_account} locale={l} />
                      </small>
                    </td>
                    <td data-label={headings[2]} data-fx-cost>
                      <SourceAmount
                        value={r[basis]}
                        currency={r.currency}
                        locale={l}
                      />
                      <small>{fieldLabel(l, basis)}</small>
                    </td>
                    <td data-label={headings[3]}>
                      {r[basis] === null
                        ? t(l, "Amount missing", "금액 누락", "المبلغ مفقود")
                        : t(
                            l,
                            "Source cost available",
                            "원천 비용 확인 가능",
                            "تكلفة المصدر متاحة",
                          )}
                      <button
                        className="b05-evidence-toggle"
                        data-fx-evidence-toggle={r.id}
                        aria-expanded={openId === r.id}
                        aria-controls={`fx-evidence-${r.id}`}
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
                      className="b05-evidence-row"
                      id={`fx-evidence-${r.id}`}
                      data-evidence-for={r.id}
                    >
                      <td colSpan={4}>
                        <Evidence locale={l} row={r} />
                      </td>
                    </tr>
                  )}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <div className="b05-pager">
        <button
          id="fx-prev"
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
          id="fx-next"
          disabled={data.next_after === null}
          onClick={() => data.next_after !== null && page(data.next_after)}
        >
          {t(l, "Next", "다음", "التالي")}
        </button>
      </div>
    </section>
  );
}
