import type { Locale } from "../../../../shared/types";
import { valueState } from "../../../sources/collected-focus/fields";
import {
  missingLabel,
  zeroLabel,
} from "../../../sources/collected-focus/labels";
import { money } from "../../../sources/collected-focus/values";
import type { Measure } from "./api";
import { t } from "./copy";
export function Money({
  value,
  currency,
  locale: l,
}: {
  value: string | null;
  currency: string;
  locale: Locale;
}) {
  if (value === null)
    return (
      <span data-b08-total="absent">
        {t(l, "No values present", "값 없음", "لا قيم موجودة")}
      </span>
    );
  return <bdi dir="ltr">{money(value, currency, l)}</bdi>;
}
export function RowAmount({
  value,
  currency,
  locale: l,
  note,
}: {
  value: string | null;
  currency: string;
  locale: Locale;
  note?: boolean;
}) {
  const state = valueState(value);
  if (state === "missing")
    return (
      <strong className="b08-state" data-b08-value="missing">
        {missingLabel(l)}
      </strong>
    );
  return (
    <span className="b08-row-amount" data-b08-value={state}>
      <bdi dir="ltr">{money(value, currency, l)}</bdi>
      {note && state === "zero" && (
        <strong className="b08-state b08-state-note">{zeroLabel(l)}</strong>
      )}
    </span>
  );
}
export function AmountValue({
  value: m,
  currency,
  locale: l,
}: {
  value: Measure;
  currency: string;
  locale: Locale;
}) {
  return (
    <>
      <Money value={m.coefficient} currency={currency} locale={l} />
      <small>
        {t(l, "Present / missing", "값 있음 / 누락", "موجود / مفقود")}{" "}
        <bdi dir="ltr">
          {m.present} / {m.missing}
        </bdi>
      </small>
    </>
  );
}
