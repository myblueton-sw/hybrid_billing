import { field, valueText } from "./fields";
import { local } from "./copy";
import type { Locale } from "../shared/types";
export type CandidateRow = { id: string; values: Record<string, string> };
export default function CandidateTable({
  columns,
  rows,
  locale: l,
  onOpen,
}: {
  columns: string[];
  rows: CandidateRow[];
  locale: Locale;
  onOpen?: (row: CandidateRow) => void;
}) {
  return (
    <>
      <p className="planning-scroll-hint">
        {local(l, [
          "표를 가로로 스크롤해 나머지 열을 확인하세요.",
          "Scroll the table horizontally for more columns.",
          "مرر الجدول أفقياً لعرض الأعمدة الأخرى.",
        ])}
      </p>
      <div
        className="planning-table"
        tabIndex={0}
        aria-label={local(l, [
          "가로로 스크롤할 수 있는 데이터 표",
          "Scrollable data table",
          "جدول بيانات قابل للتمرير",
        ])}
      >
        <table>
          <thead>
            <tr>
              {columns.map((k) => (
                <th key={k} scope="col">
                  {field(l, k)}
                </th>
              ))}
              {onOpen && (
                <th scope="col">{local(l, ["상세", "Details", "التفاصيل"])}</th>
              )}
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.id}>
                {columns.map((k, i) =>
                  i === 0 ? (
                    <th scope="row" key={k}>
                      <bdi>
                        {valueText(
                          l,
                          k === "id"
                            ? row.id
                            : (row.values[k] ?? "unconfigured"),
                        )}
                      </bdi>
                    </th>
                  ) : (
                    <td key={k}>
                      <bdi>{valueText(l, row.values[k] ?? "unconfigured")}</bdi>
                    </td>
                  ),
                )}
                {onOpen && (
                  <td>
                    <button
                      className="planning-link"
                      onClick={() => onOpen(row)}
                      aria-label={`${row.id} ${local(l, ["상세 열기", "Open detail", "فتح التفاصيل"])}`}
                    >
                      {local(l, ["상세 보기", "Inspect", "عرض"])}
                    </button>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
