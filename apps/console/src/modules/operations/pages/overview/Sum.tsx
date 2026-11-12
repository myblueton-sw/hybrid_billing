import type { Locale } from "../../../../shared/types";
import { text as t } from "../../../../shared/copy";
import { missingCount } from "../../../sources/collected-focus/labels";
import { decimal } from "../../../sources/collected-focus/values";
export const noValues = (l: Locale) =>
  t(l, "No values present", "값 없음", "لا قيم موجودة");
export default function Sum({
  value,
  currency,
  locale: l,
}: {
  value: string | null;
  currency: string;
  locale: Locale;
}) {
  return value === null ? (
    <span className="focus-sum-none" data-u01-sum="none">
      {noValues(l)}
    </span>
  ) : (
    <bdi dir="ltr" data-u01-sum="value">
      <span>{currency}</span>{" "}
      <span className="focus-exact">{decimal(value, l)}</span>
    </bdi>
  );
}
export function MissingCount({
  count,
  locale: l,
}: {
  count: number;
  locale: Locale;
}) {
  return count > 0 ? (
    <span className="focus-missing-count" data-u01-missing={count}>
      {missingCount(l, count)}
    </span>
  ) : null;
}
