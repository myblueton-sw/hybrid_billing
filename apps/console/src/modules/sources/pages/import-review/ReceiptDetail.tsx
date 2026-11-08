import type { Locale } from "../../../../shared/types";
import type { Receipt } from "./contract";
import { t, words, labels, stage, kind, timestamp, type Words } from "./copy";
export function Value({
  locale: l,
  record: r,
  field,
}: {
  locale: Locale;
  record: Receipt;
  field: keyof Receipt;
}) {
  const value = r[field];
  if (value === null) return <span>{t(l, words.unknown)}</span>;
  if (field === "stage") return <span>{stage(l, r)}</span>;
  if (field === "kind") return <span>{kind(l, r.kind)}</span>;
  if (typeof value === "boolean") {
    const yes: Words =
      field === "known_zero"
        ? [
            "등록된 전 대상의 명시적 0 확인",
            "All registered domains explicitly zero-attested",
            "إقرار صريح بالصفر لجميع النطاقات المسجلة",
          ]
        : ["예", "Yes", "نعم"];
    const no: Words =
      field === "known_zero"
        ? [
            "연결·기간 전체의 0은 확인되지 않음",
            "Whole connection/window zero is not attested",
            "لم يُقرّ بالصفر لكامل الاتصال والفترة",
          ]
        : ["아니요", "No", "لا"];
    return <span>{t(l, value ? yes : no)}</span>;
  }
  if (
    ["window_start", "window_end", "received_at", "complete_through"].includes(
      field,
    )
  )
    return (
      <time dateTime={String(value)} title={String(value)}>
        <bdi dir="ltr">{timestamp(String(value))}</bdi>
      </time>
    );
  return (
    <bdi
      dir="ltr"
      className={
        field.includes("digest") ||
        field === "id" ||
        field.includes("receipt") ||
        field === "replaced_by"
          ? "receipt-id"
          : ""
      }
    >
      {String(value)}
    </bdi>
  );
}
export const receivedLabel = (l: Locale, r: Receipt) =>
  t(
    l,
    r.kind === "delivery"
      ? [
          "수집 계열 최초 접수 시각",
          "Lineage first received",
          "أول استلام في السلسلة",
        ]
      : ["격리 접수 시각", "Quarantine receipt time", "وقت استلام العزل"],
  );
export default function ReceiptDetail({
  locale: l,
  record: r,
  onClose,
}: {
  locale: Locale;
  record: Receipt;
  onClose: () => void;
}) {
  const groups: { title: Words; fields: (keyof Receipt)[] }[] = [
    {
      title: ["식별·계보", "Identity and lineage", "الهوية والسلسلة"],
      fields: ["id", "connection", "root_receipt", "replaced_by", "attempts"],
    },
    {
      title: ["원천·기간", "Source and window", "المصدر والفترة"],
      fields: [
        "window_start",
        "window_end",
        "source_revision",
        "focus_version",
        "format",
      ],
    },
    {
      title: [
        "선언된 수집 자료",
        "Declared collection metadata",
        "بيانات التجميع المصرّح بها",
      ],
      fields: ["declared_rows", "file_bytes", "file_digest", "mapping_digest"],
    },
    {
      title: ["수집 관찰", "Collection observation", "ملاحظات التجميع"],
      fields: ["stage", "received_at", "reason"],
    },
    {
      title: [
        "현재 공개본 근거",
        "Current publication evidence",
        "أدلة الإصدار المنشور الحالي",
      ],
      fields: ["published", "complete_through", "finalized", "known_zero"],
    },
  ];
  return (
    <aside
      className="receipt-detail"
      aria-labelledby="receipt-detail-title"
      onKeyDown={(e) => {
        if (e.key === "Escape") {
          e.stopPropagation();
          onClose();
        }
      }}
    >
      <header>
        <div>
          <h2 id="receipt-detail-title" tabIndex={-1}>
            {t(l, words.details)}
          </h2>
          <p>{kind(l, r.kind)}</p>
        </div>
        <button onClick={onClose}>{t(l, words.close)}</button>
      </header>
      <p className="receipt-meta">{t(l, words.metadata)}</p>
      {groups.map((g) => {
        const fields = g.fields.filter(
          (f) =>
            r[f] !== null ||
            f === "connection" ||
            f === "stage" ||
            ["window_start", "window_end"].includes(f) ||
            (r.kind === "coverage" &&
              (f === "published" ||
                (r.published === true &&
                  ["complete_through", "finalized", "known_zero"].includes(
                    f,
                  )))) ||
            (r.kind !== "coverage" &&
              [
                "source_revision",
                "focus_version",
                "format",
                "declared_rows",
                "file_bytes",
                "file_digest",
                "mapping_digest",
                "received_at",
              ].includes(f)) ||
            (r.kind === "quarantine" &&
              ["reason", "root_receipt", "replaced_by", "attempts"].includes(
                f,
              )),
        );
        return fields.length ? (
          <section key={g.fields[0]}>
            <h3>{t(l, g.title)}</h3>
            <dl>
              {fields.map((f) => (
                <div key={f}>
                  <dt>
                    {f === "received_at"
                      ? receivedLabel(l, r)
                      : t(l, labels[f])}
                  </dt>
                  <dd>
                    <Value locale={l} record={r} field={f} />
                  </dd>
                </div>
              ))}
            </dl>
          </section>
        ) : null;
      })}
      {r.kind === "coverage" && r.published === false && (
        <p>
          {t(l, [
            "현재 공개본이 없어 종료 선언·0 확인·수집 기준 시각은 미확인입니다.",
            "There is no current publication; finalization, zero attestation and collected-through evidence are unknown.",
            "لا يوجد إصدار منشور حالي؛ إعلان الانتهاء والإقرار بالصفر ودليل التجميع حتى غير معروفة.",
          ])}
        </p>
      )}
      {r.stage === "deleted" && (
        <p>
          {t(l, [
            "삭제 결정 기록은 현재 물리적 삭제나 백업 만료 확인이 아닙니다.",
            "A deletion decision record does not establish physical erasure or backup expiry.",
            "سجل قرار الحذف لا يثبت المحو الفعلي أو انتهاء النسخ الاحتياطية.",
          ])}
        </p>
      )}
    </aside>
  );
}
