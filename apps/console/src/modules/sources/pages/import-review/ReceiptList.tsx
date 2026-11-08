import { useId } from "react";
import type { Locale } from "../../../../shared/types";
import TableRegion from "../../../../shared/TableRegion";
import type { Receipt, View } from "./contract";
import { t, words, labels, stage, type Words } from "./copy";
import { Value, receivedLabel } from "./ReceiptDetail";
export default function ReceiptList({
  locale: l,
  view,
  records,
  selected,
  onSelect,
}: {
  locale: Locale;
  view: View;
  records: Receipt[];
  selected?: string;
  onSelect: (r: Receipt, trigger: HTMLButtonElement) => void;
}) {
  const noteId = useId();
  const headings: Words[] = [
    labels.id,
    ["연결·자료 기간", "Connection and window", "الاتصال والفترة"],
    view === "quarantine"
      ? ["격리 처리 단계", "Quarantine stage", "مرحلة العزل"]
      : labels.stage,
    view === "quarantine"
      ? labels.reason
      : view === "completeness"
        ? labels.known_zero
        : [
            "선언된 행·바이트",
            "Declared rows and bytes",
            "الصفوف والبايتات المصرّح بها",
          ],
    view === "completeness"
      ? labels.complete_through
      : ["접수 기준 시각", "Receipt reference time", "وقت مرجع الاستلام"],
  ];
  return (
    <>
      <p id={noteId} className="receipt-list-note">
        {t(l, [
          "현재 조회 페이지. 기술 식별자는 상세에서 전체 값을 확인할 수 있습니다.",
          "Current page. Inspect a record to see its full technical reference.",
          "الصفحة الحالية. افحص سجلًا لعرض مرجعه التقني الكامل.",
        ])}
      </p>
      <TableRegion
        locale={l}
        className="receipt-table"
        label={t(l, [
          "현재 허용된 접수 목록",
          "Currently permitted receipt list",
          "قائمة الإيصالات المسموحة حاليًا",
        ])}
      >
        <table aria-describedby={noteId}>
          <thead>
            <tr>
              {headings.map((h, i) => (
                <th key={i} scope="col">
                  {t(l, h)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {records.map((r) => (
              <tr
                key={r.id}
                className={selected === r.id ? "receipt-selected" : ""}
              >
                <th scope="row">
                  <button
                    className="receipt-open"
                    aria-pressed={selected === r.id}
                    aria-label={`${t(l, words.inspect)} ${r.id.slice(0, 12)}`}
                    onClick={(e) => onSelect(r, e.currentTarget)}
                  >
                    <bdi dir="ltr">{r.id.slice(0, 12)}</bdi>
                  </button>
                </th>
                <td>
                  <Value locale={l} record={r} field="connection" />
                  <div className="receipt-meta">
                    <div>
                      {t(l, [
                        "시작 · 포함",
                        "Start · inclusive",
                        "البداية · مشمولة",
                      ])}
                      <br />
                      <Value locale={l} record={r} field="window_start" />
                    </div>
                    <div>
                      {t(l, [
                        "종료 · 제외",
                        "End · exclusive",
                        "النهاية · غير مشمولة",
                      ])}
                      <br />
                      <Value locale={l} record={r} field="window_end" />
                    </div>
                  </div>
                </td>
                <td>
                  {stage(l, r)}
                  {r.kind === "coverage" && (
                    <div className="receipt-meta">
                      {t(l, labels.published)}:{" "}
                      <Value locale={l} record={r} field="published" />
                    </div>
                  )}
                </td>
                <td>
                  {r.kind === "quarantine" ? (
                    <Value locale={l} record={r} field="reason" />
                  ) : r.kind === "coverage" ? (
                    <Value locale={l} record={r} field="known_zero" />
                  ) : (
                    <>
                      <Value locale={l} record={r} field="declared_rows" />
                      <div className="receipt-meta">
                        <Value locale={l} record={r} field="file_bytes" />{" "}
                        {t(l, ["바이트", "bytes", "بايت"])}
                      </div>
                    </>
                  )}
                </td>
                <td>
                  {r.kind === "coverage" ? (
                    <Value locale={l} record={r} field="complete_through" />
                  ) : (
                    <>
                      <span className="receipt-meta">
                        {receivedLabel(l, r)}
                      </span>
                      <div>
                        <Value locale={l} record={r} field="received_at" />
                      </div>
                    </>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </TableRegion>
    </>
  );
}
