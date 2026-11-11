import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type {
  OverviewData,
  FocusRow,
} from "../../../sources/collected-focus/api";
import { categories, type Draft, type Entry } from "./model";
import { categoryLabel } from "./copy";
import { amountText, identityText, optionText } from "./identity";
import { SplitId } from "./RowFacts";
export default function Declaration({
  locale: l,
  data,
  draft,
  row,
  entry,
  patch,
}: {
  locale: Locale;
  data: OverviewData;
  draft: Draft;
  row: FocusRow;
  entry: Entry;
  patch: (id: string, value: Partial<Entry>) => void;
}) {
  const identity = identityText(l, row),
    category = t(l, "Category", "범주", "الفئة"),
    target = t(l, "Included in", "포함 대상", "مشمولة ضمن"),
    reference = t(l, "Evidence reference", "근거 참조", "مرجع الدليل"),
    differsHint = t(
      l,
      "Different currency or charge period",
      "통화 또는 과금 기간 다름",
      "عملة أو فترة رسوم مختلفة",
    ),
    labelId = `infra-declare-${row.id}`,
    parent = entry.parent
      ? data.rows.find((candidate) => candidate.id === entry.parent)
      : undefined,
    differs = (other: FocusRow) =>
      other.currency !== row.currency ||
      other.charge_start !== row.charge_start ||
      other.charge_end !== row.charge_end ||
      other.charge_month !== row.charge_month;
  return (
    <div className="infra-declare" role="group" aria-labelledby={labelId}>
      <p className="infra-declare-id" id={labelId}>
        {identity}
      </p>
      <label>
        <span>{category}</span>
        <select
          aria-label={`${category}: ${identity}`}
          value={entry.category}
          onChange={(e) =>
            patch(row.id, { category: e.target.value as Entry["category"] })
          }
        >
          {categories.map((c) => (
            <option key={c} value={c}>
              {categoryLabel(l, c)}
            </option>
          ))}
        </select>
      </label>
      <div className="infra-declare-field">
        <label>
          <span>{target}</span>
          <select
            value={entry.parent ?? ""}
            aria-label={`${target}: ${identity}`}
            onChange={(e) =>
              patch(row.id, {
                parent: e.target.value || null,
                reference: e.target.value ? entry.reference : "",
              })
            }
          >
            <option value="">
              {t(l, "No inclusion target", "포함 대상 없음", "بلا هدف تضمين")}
            </option>
            {data.rows
              .filter(
                (other) =>
                  other.id !== row.id &&
                  draft.entries.some((e) => e.row === other.id),
              )
              .map((other) => (
                <option
                  key={other.id}
                  value={other.id}
                  disabled={differs(other)}
                >
                  {optionText(l, data.selection.basis, other)}
                  {differs(other) ? ` (${differsHint})` : ""}
                </option>
              ))}
          </select>
        </label>
        {entry.parent && (
          <small className="infra-target-id">
            <span>
              {t(l, "Target row ID", "포함 대상 행 ID", "معرّف الصف الهدف")}
            </span>
            <span data-inclusion-target>
              <SplitId id={entry.parent} />
            </span>
            {parent && (
              <span data-inclusion-amount>
                {amountText(l, data.selection.basis, parent)}
              </span>
            )}
          </small>
        )}
      </div>
      <div className="infra-declare-field">
        <label>
          <span>{reference}</span>
          <input
            aria-label={`${reference}: ${identity}`}
            value={entry.reference}
            maxLength={300}
            required={!!entry.parent}
            disabled={!entry.parent}
            aria-invalid={!!entry.parent && !entry.reference.trim()}
            onChange={(e) => patch(row.id, { reference: e.target.value })}
          />
        </label>
        {!!entry.parent && !entry.reference.trim() && (
          <small>
            {t(l, "Reference required", "근거 참조 필요", "المرجع مطلوب")}
          </small>
        )}
      </div>
      <p className="infra-declare-status">
        <span>
          {t(l, "Verification / approval", "검증 / 승인", "التحقّق / الاعتماد")}
        </span>
        {t(
          l,
          "Unverified declaration",
          "검증되지 않은 선언",
          "إقرار غير متحقّق منه",
        )}
        {" · "}
        {t(l, "No approval record", "승인 기록 없음", "لا يوجد سجل اعتماد")}
      </p>
    </div>
  );
}
