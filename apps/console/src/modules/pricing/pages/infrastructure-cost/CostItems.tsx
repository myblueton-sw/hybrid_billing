import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type {
  OverviewData,
  FocusRow,
} from "../../../sources/collected-focus/api";
import {
  fieldLabel,
  originLabel,
} from "../../../sources/collected-focus/labels";
import { removeEntry, type Draft, type Entry } from "./model";
import { categoryLabel, basisLabel } from "./copy";
import { deliveryCount, identityText } from "./identity";
import { ChargePeriod, CostLines, RowIdentity } from "./RowFacts";
import Declaration from "./Declaration";
export default function CostItems({
  locale: l,
  data,
  draft,
  edit,
  update,
}: {
  locale: Locale;
  data: OverviewData;
  draft: Draft;
  edit: boolean;
  update: (d: Draft) => void;
}) {
  const rows = edit
      ? data.rows.filter((r) => draft.entries.some((e) => e.row === r.id))
      : data.rows,
    basis = data.selection.basis,
    otherBasis = basis === "billed_cost" ? "effective_cost" : "billed_cost",
    showDelivery = deliveryCount(data.rows) > 1,
    selection = t(l, "Selection", "선택", "التحديد"),
    period = t(
      l,
      "Charge period (UTC)",
      "과금 기간 (UTC)",
      "فترة الرسوم (UTC)",
    ),
    category = t(l, "Category", "범주", "الفئة");
  function patch(id: string, value: Partial<Entry>) {
    update({
      ...draft,
      entries: draft.entries.map((e) =>
        e.row === id ? { ...e, ...value } : e,
      ),
    });
  }
  function toggle(row: FocusRow, entry: Entry | undefined) {
    update(
      entry
        ? removeEntry(draft, row.id)
        : {
            ...draft,
            entries: [
              ...draft.entries,
              {
                row: row.id,
                category: "unclassified",
                parent: null,
                reference: "",
              },
            ],
          },
    );
  }
  function facts(row: FocusRow, index: number, entry: Entry | undefined) {
    return (
      <tr key={row.id} data-selected={!!entry} data-row={row.id}>
        <td className="infra-select">
          <label>
            <input
              type="checkbox"
              checked={!!entry}
              data-source-row={row.id}
              aria-label={`${t(l, "Select source row", "원천 행 선택", "تحديد صف المصدر")} ${index + 1}: ${identityText(l, row)}`}
              aria-describedby="infra-selection-limit"
              disabled={!entry && draft.entries.length >= 25}
              onChange={() => toggle(row, entry)}
            />
          </label>
        </td>
        <td className="infra-row-identity">
          <RowIdentity locale={l} row={row} showDelivery={showDelivery} />
        </td>
        <td data-label={period}>
          <ChargePeriod locale={l} row={row} />
        </td>
        <td data-label={fieldLabel(l, "charge_month")}>
          <small className="infra-card-only">{originLabel(l, "derived")}</small>
          <bdi dir="ltr">{row.charge_month}</bdi>
        </td>
        <td className="infra-money">
          <CostLines
            locale={l}
            row={row}
            basis={basis}
            included={!!entry?.parent}
          />
        </td>
        {!edit && (
          <td data-label={category}>
            {categoryLabel(l, entry?.category ?? "unclassified")}
          </td>
        )}
      </tr>
    );
  }
  return (
    <section className="infra-items" aria-labelledby="infra-items-title">
      <header className="infra-section-head">
        <h2 id="infra-items-title">
          {edit
            ? t(l, "Cost items", "원가 항목", "بنود التكلفة")
            : t(l, "Source candidates", "원천 후보", "صفوف المصدر المرشّحة")}
        </h2>
        <span className="infra-meta">
          {t(l, "Selected", "선택", "المحدّد")}:{" "}
          <bdi dir="ltr">{draft.entries.length} / 25</bdi>
        </span>
      </header>
      <p className="infra-meta" id="infra-selection-limit">
        {t(
          l,
          "Up to 25 rows on the current page. Changing source, month, cost basis or page resets the composition. Classifications and whole-row inclusion are user declarations, not verified evidence.",
          "현재 페이지에서 최대 25행을 선택합니다. 원천·월·비용 기준·페이지를 바꾸면 구성이 초기화됩니다. 범주와 행 전체의 포함 관계는 사용자 선언이며 검증된 근거가 아닙니다.",
          "يمكن تحديد 25 صفاً من الصفحة الحالية. تغيير المصدر أو الشهر أو أساس التكلفة أو الصفحة يعيد ضبط المكوّنات. التصنيفات وتضمين الصفوف كاملةً إقرارات من المستخدم وليست أدلة متحقّقاً منها.",
        )}
      </p>
      <p className="infra-meta">
        {t(
          l,
          "Only the row ID uniquely identifies a source row. The row ordinal is the row's position within one delivery.",
          "행 ID만 원천 행을 고유하게 식별합니다. 행 순번은 한 수집본 안에서의 위치입니다.",
          "معرّف الصف هو المفتاح الفريد الوحيد لصف المصدر. ترتيب الصف هو موضعه داخل تسليم واحد.",
        )}
      </p>
      {!rows.length ? (
        <p>
          {t(
            l,
            "No source rows selected.",
            "선택한 원천 행이 없습니다.",
            "لا توجد صفوف مصدر محدّدة.",
          )}
        </p>
      ) : (
        <div
          className="infra-table"
          data-edit={edit}
          tabIndex={0}
          role="region"
          aria-label={t(
            l,
            "Source cost items",
            "원천 원가 항목",
            "بنود تكلفة المصدر",
          )}
        >
          <table>
            <colgroup>
              <col className="infra-col-select" />
              <col />
              <col className="infra-col-period" />
              <col className="infra-col-month" />
              <col className="infra-col-cost" />
              {!edit && <col className="infra-col-category" />}
            </colgroup>
            <thead>
              <tr>
                <th>
                  <span className="infra-select-label">{selection}</span>
                </th>
                <th>{t(l, "Source row", "원천 행", "صف المصدر")}</th>
                <th>{period}</th>
                <th>
                  {fieldLabel(l, "charge_month")}
                  <small>{originLabel(l, "derived")}</small>
                </th>
                <th className="infra-money">
                  {basisLabel(l, basis)}
                  <small>{basisLabel(l, otherBasis)}</small>
                </th>
                {!edit && <th>{category}</th>}
              </tr>
            </thead>
            {edit ? (
              rows.map((row, index) => {
                const entry = draft.entries.find((e) => e.row === row.id);
                return (
                  <tbody key={row.id} data-selected={!!entry}>
                    {facts(row, index, entry)}
                    {entry && (
                      <tr className="infra-declare-row">
                        <td colSpan={5}>
                          <Declaration
                            locale={l}
                            data={data}
                            draft={draft}
                            row={row}
                            entry={entry}
                            patch={patch}
                          />
                        </td>
                      </tr>
                    )}
                  </tbody>
                );
              })
            ) : (
              <tbody>
                {rows.map((row, index) =>
                  facts(
                    row,
                    index,
                    draft.entries.find((e) => e.row === row.id),
                  ),
                )}
              </tbody>
            )}
          </table>
        </div>
      )}
    </section>
  );
}
