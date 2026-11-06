import type { Locale } from "../../../../shared/types";
import type { EvidenceRecord, Money, Dimension } from "./contract";
import { t, reported, unknown } from "./copy";
import { SourceValue, Measure, MoneyValues } from "./Shared";
function BriefMeasure({
  l,
  value,
  dimension,
}: {
  l: Locale;
  value: string | null;
  dimension: Dimension;
}) {
  return value === null ? (
    <span className="b11-missing">{unknown(l)}</span>
  ) : (
    <bdi dir="ltr">
      {dimension.kind === "money"
        ? `${dimension.currency} ${value}`
        : `${value} ${dimension.unit}`}
    </bdi>
  );
}
function BriefMoney({ l, amounts }: { l: Locale; amounts: Money[] }) {
  return (
    <>
      {amounts.length
        ? amounts.map((a) => (
            <span key={a.currency}>
              <BriefMeasure
                l={l}
                value={a.value}
                dimension={{ kind: "money", currency: a.currency, unit: null }}
              />
            </span>
          ))
        : unknown(l)}
    </>
  );
}
export function stateLabel(l: Locale, r: EvidenceRecord) {
  return reported(
    l,
    r.kind === "balance"
      ? r.authority.freshness
      : r.kind === "payment"
        ? r.overdue
        : r.state,
  );
}
export function RowSummary({
  locale: l,
  record: r,
}: {
  locale: Locale;
  record: EvidenceRecord;
}) {
  switch (r.kind) {
    case "balance":
      return (
        <>
          <span>
            {t(
              l,
              "Available (source)",
              "사용 가능 (원천 관측)",
              "المتاح في المصدر",
            )}
            : <BriefMeasure l={l} value={r.available} dimension={r.dimension} />
          </span>
          <span>
            {t(l, "Reserved (source)", "예약 (원천 관측)", "المحجوز في المصدر")}
            : <BriefMeasure l={l} value={r.reserved} dimension={r.dimension} />
          </span>
        </>
      );
    case "reservation":
      return (
        <>
          <span>
            <SourceValue locale={l} value={r.credit_ref} />
          </span>
          <span>
            <BriefMeasure l={l} value={r.value} dimension={r.dimension} />
          </span>
        </>
      );
    case "case":
      return (
        <>
          <span>
            {t(l, "Disputed amount", "이의 금액", "مبلغ الاعتراض")}:{" "}
            <BriefMoney l={l} amounts={r.amounts} />
          </span>
          <span>
            {t(l, "Deadline", "처리 기한", "المهلة")}:{" "}
            <SourceValue locale={l} value={r.deadline} />
          </span>
        </>
      );
    case "adjustment":
      return (
        <>
          <span>{reported(l, r.adjustment_kind)}</span>
          <span>
            <BriefMoney l={l} amounts={r.amounts} />
          </span>
        </>
      );
    case "payment":
      return (
        <>
          <span>
            {t(
              l,
              "Effective due date",
              "유효 지급 기한",
              "موعد الاستحقاق الفعّال",
            )}
            : <SourceValue locale={l} value={r.effective_due_date} />
          </span>
          <span>
            {t(l, "Remaining receivable", "남은 미수금", "الذمم المتبقية")}:{" "}
            <BriefMoney l={l} amounts={[r.receivable]} />
          </span>
        </>
      );
  }
}
export function ReferenceSummary({
  locale: l,
  record: r,
}: {
  locale: Locale;
  record: EvidenceRecord;
}) {
  const refs =
    r.kind === "balance"
      ? [r.migration_ref, r.included_history_ref]
      : r.kind === "reservation"
        ? [r.credit_ref, r.document_ref, r.original_ref]
        : r.kind === "case"
          ? [r.original_document_ref, r.resolution_ref, r.consent.evidence_ref]
          : r.kind === "adjustment"
            ? [
                r.original_document_ref,
                r.original_claim_ref,
                r.correction_ref,
                r.approval_ref,
              ]
            : [r.original_document_ref, r.term_ref, r.receipt.reference];
  return (
    <>
      {refs.map((v, i) => (
        <div key={i}>
          <SourceValue locale={l} value={v} />
        </div>
      ))}
    </>
  );
}
export function TableValues({
  locale: l,
  record: r,
}: {
  locale: Locale;
  record: EvidenceRecord;
}) {
  if (r.kind === "balance")
    return (
      <>
        <p>
          {t(
            l,
            "Available / reserved — source observations",
            "사용 가능·예약 (원천 관측)",
            "المتاح والمحجوز كقيم مصدر",
          )}
        </p>
        <Measure locale={l} value={r.available} dimension={r.dimension} />
        <Measure locale={l} value={r.reserved} dimension={r.dimension} />
      </>
    );
  if (r.kind === "reservation")
    return <Measure locale={l} value={r.value} dimension={r.dimension} />;
  return (
    <MoneyValues
      locale={l}
      amounts={r.kind === "payment" ? [r.receivable] : r.amounts}
    />
  );
}
