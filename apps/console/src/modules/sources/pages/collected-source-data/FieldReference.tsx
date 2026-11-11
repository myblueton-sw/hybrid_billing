import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import {
  annotationColumn,
  mappedFields,
  mappedColumns,
  annotationFields,
  derivedFields,
  lineageFields,
  rowVersion,
  type MappedField,
  type Origin,
} from "../../collected-focus/fields";
import {
  fieldLabel,
  handling,
  originLabel,
  originMeaning,
  type DisplayField,
} from "../../collected-focus/labels";
const groups: { origin: Origin; fields: DisplayField[] }[] = [
  { origin: "source", fields: [...mappedFields] },
  { origin: "derived", fields: [...derivedFields] },
  { origin: "lineage", fields: [...lineageFields] },
  { origin: "annotation", fields: [...annotationFields] },
];
export function SourceColumnNames({
  l,
  field,
  versions,
}: {
  l: Locale;
  field: MappedField;
  versions: string[];
}) {
  const columns = mappedColumns(field, versions);
  return (
    <>
      {columns.names.length === 1 ? (
        <bdi dir="ltr" className="collected-code">
          {columns.names[0]}
        </bdi>
      ) : (
        columns.mapped.map((entry) => (
          <span key={entry.version} className="collected-version-name">
            <bdi dir="ltr">{entry.version}</bdi>{" "}
            <bdi dir="ltr" className="collected-code">
              {entry.name}
            </bdi>
          </span>
        ))
      )}
      {columns.unmapped.map((version) => (
        <small key={version} data-unmapped-version={version}>
          {t(
            l,
            "Column mapping not available for version",
            "열 매핑을 확인할 수 없는 버전",
            "ربط الأعمدة غير متاح للإصدار",
          )}{" "}
          <bdi dir="ltr">{version}</bdi>
        </small>
      ))}
      {!versions.length && (
        <small>
          {t(
            l,
            "Column mapping not available",
            "열 매핑을 확인할 수 없습니다",
            "ربط الأعمدة غير متاح",
          )}
        </small>
      )}
    </>
  );
}
export function VersionScope({
  l,
  versions,
}: {
  l: Locale;
  versions: string[];
}) {
  const version = rowVersion(versions);
  return version ? (
    <>
      {t(
        l,
        "FOCUS mapping version in your authorized scope:",
        "전체 허용 범위의 FOCUS 매핑 버전:",
        "إصدار ربط FOCUS ضمن نطاقك المسموح:",
      )}{" "}
      <bdi dir="ltr">{version}</bdi>
    </>
  ) : (
    <>
      {t(
        l,
        "FOCUS mapping versions in your authorized scope:",
        "전체 허용 범위의 FOCUS 매핑 버전:",
        "إصدارات ربط FOCUS ضمن نطاقك المسموح:",
      )}{" "}
      <bdi dir="ltr">{versions.join(", ") || "—"}</bdi>.{" "}
      {t(
        l,
        "The row record does not carry its own version.",
        "행 기록에는 행별 버전이 포함되어 있지 않습니다.",
        "سجل الصف لا يتضمن إصداره الخاص.",
      )}
    </>
  );
}
export default function FieldReference({
  l,
  versions,
}: {
  l: Locale;
  versions: string[];
}) {
  return (
    <details
      className="collected-reference"
      data-item="S04:reference:field-mapping"
    >
      <summary>
        {t(l, "Field mapping reference", "필드 매핑 참조", "مرجع ربط الحقول")}
        <small>
          <VersionScope l={l} versions={versions} />
        </small>
      </summary>
      <table role="table">
        <thead role="rowgroup">
          <tr role="row">
            <th scope="col" role="columnheader">
              {t(l, "Field", "필드", "الحقل")}
            </th>
            <th scope="col" role="columnheader">
              {t(l, "FOCUS column", "FOCUS 열", "عمود FOCUS")}
            </th>
            <th scope="col" role="columnheader">
              {t(l, "Handling", "처리 방식", "المعالجة")}
            </th>
          </tr>
        </thead>
        {groups.map((group) => (
          <tbody key={group.origin} data-origin={group.origin} role="rowgroup">
            <tr className="collected-group" role="row">
              <th scope="rowgroup" colSpan={3} role="rowheader">
                {originLabel(l, group.origin)}
                <small>{originMeaning(l, group.origin)}</small>
              </th>
            </tr>
            {group.fields.map((field) => (
              <tr key={field} role="row">
                <th scope="row" role="rowheader">
                  {fieldLabel(l, field)}
                </th>
                <td role="cell">
                  {group.origin === "source" ? (
                    <SourceColumnNames
                      l={l}
                      field={field as MappedField}
                      versions={versions}
                    />
                  ) : group.origin === "annotation" ? (
                    <bdi dir="ltr" className="collected-code">
                      {annotationColumn}
                    </bdi>
                  ) : (
                    <span className="collected-quiet">
                      {t(
                        l,
                        "No FOCUS column",
                        "FOCUS 열 없음",
                        "لا يوجد عمود FOCUS",
                      )}
                    </span>
                  )}
                </td>
                <td role="cell">{handling(l, field)}</td>
              </tr>
            ))}
          </tbody>
        ))}
      </table>
    </details>
  );
}
