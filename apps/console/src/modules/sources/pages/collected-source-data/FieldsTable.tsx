import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import type { FocusRow } from "../../collected-focus/api";
import {
  mappingVersions,
  missingFields,
  provenanceClass,
  type NullableField,
} from "../../collected-focus/fields";
import {
  fieldLabel,
  Instant,
  missingCount,
  missingLabel,
  selectedLabel,
  originLabel,
  provenanceLabel,
} from "./labels";
const shownNames = 4;
function MissingValues({
  l,
  row,
  versions,
}: {
  l: Locale;
  row: FocusRow;
  versions: string[];
}) {
  const missing = missingFields(row),
    mapping = versions.length === 1 ? mappingVersions[versions[0]] : undefined,
    name = (field: NullableField) => mapping?.[field];
  if (!missing.length)
    return (
      <span className="collected-quiet">{t(l, "None", "없음", "لا يوجد")}</span>
    );
  return (
    <>
      <strong>{missingCount(l, missing.length)}</strong>
      {missing.slice(0, shownNames).map((field) => (
        <small key={field}>
          {name(field) ? (
            <bdi dir="ltr" className="collected-code">
              {name(field)}
            </bdi>
          ) : (
            fieldLabel(l, field)
          )}
        </small>
      ))}
      {missing.length > shownNames && (
        <small>
          <bdi dir="ltr">+{missing.length - shownNames}</bdi>
        </small>
      )}
    </>
  );
}
export default function FieldsTable({
  l,
  rows,
  versions,
  selectedId,
  rowLabel,
  onInspect,
}: {
  l: Locale;
  rows: FocusRow[];
  versions: string[];
  selectedId?: string;
  rowLabel: (row: FocusRow) => string;
  onInspect: (row: FocusRow, trigger: HTMLElement) => void;
}) {
  const bounds = (
    <small>
      {t(
        l,
        "Start, then end (exclusive)",
        "시작, 종료(미포함) 순",
        "البداية ثم النهاية (مستثناة)",
      )}
    </small>
  );
  return (
    <table className="collected-fields-table">
      <thead>
        <tr className="collected-band">
          <th scope="colgroup" colSpan={6}>
            <span>{originLabel(l, "source")}</span>
          </th>
          <th scope="colgroup" colSpan={2}>
            <span>{originLabel(l, "derived")}</span>
          </th>
          <th scope="colgroup" colSpan={2}>
            <span>
              {originLabel(l, "lineage")} · {originLabel(l, "annotation")}
            </span>
          </th>
        </tr>
        <tr>
          <th scope="col">
            {t(l, "Account / detail", "계정 / 상세", "الحساب / التفاصيل")}
          </th>
          <th scope="col">
            {t(l, "Provider / service", "공급사 / 서비스", "المزود / الخدمة")}
          </th>
          <th scope="col">
            {t(l, "Missing values", "누락 값", "القيم المفقودة")}
          </th>
          <th scope="col">{t(l, "Currency", "통화", "العملة")}</th>
          <th scope="col">
            {t(
              l,
              "Charge period (UTC)",
              "과금 기간 (UTC)",
              "فترة الرسوم (UTC)",
            )}
            {bounds}
          </th>
          <th scope="col">
            {t(
              l,
              "Billing period (UTC)",
              "청구 기간 (UTC)",
              "فترة الفوترة (UTC)",
            )}
            {bounds}
          </th>
          <th scope="col">{fieldLabel(l, "charge_month")}</th>
          <th scope="col">{fieldLabel(l, "id")}</th>
          <th scope="col">
            {t(
              l,
              "Delivery ID / row ordinal",
              "수집본 ID / 행 순번",
              "معرّف التسليم / ترتيب الصف",
            )}
          </th>
          <th scope="col">{fieldLabel(l, "evidence_class")}</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => (
          <tr
            key={r.id}
            data-row-id={r.id}
            data-selected={selectedId === r.id || undefined}
          >
            <th scope="row">
              <button
                aria-label={rowLabel(r)}
                aria-expanded={selectedId === r.id}
                onClick={(event) => onInspect(r, event.currentTarget)}
              >
                <bdi>{r.billing_account}</bdi>
              </button>
              <small>
                {r.sub_account === null ? (
                  missingLabel(l)
                ) : (
                  <bdi>{r.sub_account}</bdi>
                )}
              </small>
              {selectedId === r.id && (
                <small className="collected-selected">{selectedLabel(l)}</small>
              )}
            </th>
            <td>
              {r.provider}
              <small>{r.service ?? missingLabel(l)}</small>
            </td>
            <td data-missing-count={missingFields(r).length}>
              <MissingValues l={l} row={r} versions={versions} />
            </td>
            <td>
              <bdi dir="ltr">{r.currency}</bdi>
            </td>
            <td>
              <Instant value={r.charge_start} />
              <Instant value={r.charge_end} />
            </td>
            <td>
              <Instant value={r.billing_start} />
              <Instant value={r.billing_end} />
            </td>
            <td className="collected-group-start">
              <bdi dir="ltr">{r.charge_month}</bdi>
            </td>
            <td>
              <bdi dir="ltr" className="collected-token collected-code">
                {r.id.slice(0, 12)}
              </bdi>
            </td>
            <td className="collected-group-start">
              <bdi dir="ltr" className="collected-token collected-code">
                {r.batch.slice(0, 8)}
              </bdi>
              <small>
                <bdi dir="ltr">{r.ordinal}</bdi>
              </small>
            </td>
            <td>{provenanceLabel(l, provenanceClass(r.evidence_class))}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
