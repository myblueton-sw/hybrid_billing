import type { ReactNode } from "react";
import type { Locale } from "../../../../shared/types";
import {
  instantParts,
  provenanceClass,
  valueState,
  type Origin,
} from "../../../sources/collected-focus/fields";
import {
  fieldLabel,
  Instant,
  missingLabel,
  originLabel,
  provenanceLabel,
  unitMissingLabel,
  zeroLabel,
  type DisplayField,
} from "../../../sources/collected-focus/labels";
import { decimal } from "../../../sources/collected-focus/values";
import type { FocusRow } from "./api";
import { t, basisLabel } from "./copy";
import { RowAmount } from "./Amounts";
function Text({ value, locale: l }: { value: string | null; locale: Locale }) {
  return value === null ? (
    <strong className="b08-state" data-b08-value="missing">
      {missingLabel(l)}
    </strong>
  ) : (
    <bdi dir="ltr">{value}</bdi>
  );
}
function Time({ value, locale: l }: { value: string; locale: Locale }) {
  return instantParts(value) ? (
    <bdi dir="ltr" className="b08-instant">
      <Instant value={value} /> <span className="b08-tz">UTC</span>
    </bdi>
  ) : (
    <>
      <bdi dir="ltr" data-b08-instant="raw">
        {value}
      </bdi>
      <small>
        {t(
          l,
          "Shown as collected; not converted to UTC",
          "수집된 그대로 표시하며 UTC로 변환하지 않았습니다",
          "معروض كما جُمع دون تحويل إلى UTC",
        )}
      </small>
    </>
  );
}
function Quantity({
  value,
  unit,
  locale: l,
}: {
  value: string | null;
  unit: string | null;
  locale: Locale;
}) {
  const state = valueState(value);
  return (
    <>
      {state === "missing" ? (
        <strong className="b08-state" data-b08-value="missing">
          {missingLabel(l)}
        </strong>
      ) : (
        <bdi dir="ltr">{decimal(value, l)}</bdi>
      )}{" "}
      {unit === null ? (
        <strong className="b08-state b08-state-note" data-b08-unit="missing">
          {unitMissingLabel(l)}
        </strong>
      ) : (
        <bdi dir="ltr">{unit}</bdi>
      )}
      {state === "zero" && (
        <strong className="b08-state b08-state-note">{zeroLabel(l)}</strong>
      )}
    </>
  );
}
export default function SourceDetail({
  locale: l,
  row: r,
  close,
}: {
  locale: Locale;
  row: FocusRow;
  close: () => void;
}) {
  const fact = (field: DisplayField, value: ReactNode, label?: string) => (
    <div key={field} data-b08-field={field}>
      <dt>{label ?? fieldLabel(l, field)}</dt>
      <dd>{value}</dd>
    </div>
  );
  const group = (origin: Origin, facts: ReactNode[]) => (
    <section className="b08-origin" data-b08-origin={origin}>
      <h3>{originLabel(l, origin)}</h3>
      <dl className="b08-facts">{facts}</dl>
    </section>
  );
  return (
    <section
      className="b08-detail"
      onKeyDown={(e) => {
        if (
          e.key === "Escape" &&
          !(e.target as HTMLElement).closest("details[open]")
        ) {
          e.preventDefault();
          close();
        }
      }}
    >
      <header>
        <h2 tabIndex={-1} data-b08-detail-heading>
          {t(
            l,
            "Source input evidence",
            "원천 입력 근거",
            "أدلة مدخلات المصدر",
          )}
        </h2>
        <button onClick={close}>
          {t(l, "Close details", "상세 닫기", "إغلاق التفاصيل")}
        </button>
      </header>
      <p className="b08-meta">
        {t(
          l,
          "Values of one collected source row. Not a customer, contract or billing-run record.",
          "수집된 원천 행 하나의 값입니다. 고객·계약·청구 계산 실행 기록이 아닙니다.",
          "قيم صف مصدر مجمَّع واحد. ليست سجل عميل أو عقد أو عملية احتساب.",
        )}
      </p>
      {group("source", [
        fact("provider", <Text value={r.provider} locale={l} />),
        fact("service", <Text value={r.service} locale={l} />),
        fact("billing_account", <Text value={r.billing_account} locale={l} />),
        fact("sub_account", <Text value={r.sub_account} locale={l} />),
        fact("currency", <Text value={r.currency} locale={l} />),
        fact("charge_start", <Time value={r.charge_start} locale={l} />),
        fact("charge_end", <Time value={r.charge_end} locale={l} />),
        fact("billing_start", <Time value={r.billing_start} locale={l} />),
        fact("billing_end", <Time value={r.billing_end} locale={l} />),
        ...(["billed_cost", "effective_cost"] as const).map((b) =>
          fact(
            b,
            <RowAmount value={r[b]} currency={r.currency} locale={l} note />,
            basisLabel(l, b),
          ),
        ),
      ])}
      {group("derived", [
        fact("charge_month", <Text value={r.charge_month} locale={l} />),
        fact(
          "id",
          <bdi dir="ltr" className="b08-code">
            <span>{r.id.slice(0, 32)}</span>
            <wbr />
            <span>{r.id.slice(32)}</span>
          </bdi>,
        ),
      ])}
      {group("lineage", [
        fact(
          "batch",
          <bdi dir="ltr" className="b08-code">
            {r.batch}
          </bdi>,
        ),
        fact("ordinal", <Text value={r.ordinal} locale={l} />),
      ])}
      {group("annotation", [
        fact(
          "evidence_class",
          provenanceLabel(l, provenanceClass(r.evidence_class)),
        ),
      ])}
      <details>
        <summary>
          {t(
            l,
            "Quantities and source units",
            "수량·원천 단위",
            "الكميات ووحدات المصدر",
          )}
        </summary>
        <dl className="b08-facts">
          {fact(
            "consumed_quantity",
            <Quantity
              value={r.consumed_quantity}
              unit={r.consumed_unit}
              locale={l}
            />,
          )}
          {fact(
            "pricing_quantity",
            <Quantity
              value={r.pricing_quantity}
              unit={r.pricing_unit}
              locale={l}
            />,
          )}
        </dl>
        <p className="b08-meta">
          {t(
            l,
            "List and contracted costs and unit prices are shown on the Consolidated cost and usage screen.",
            "정가·계약 비용과 단가는 통합 비용·사용량 명세 화면에서 확인합니다.",
            "تكاليف القائمة والعقد وأسعار الوحدة معروضة في شاشة تفاصيل التكاليف والاستخدام الموحدة.",
          )}
        </p>
      </details>
    </section>
  );
}
