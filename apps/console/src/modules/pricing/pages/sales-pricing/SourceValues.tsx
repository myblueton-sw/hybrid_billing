import type { Locale } from "../../../../shared/types";
import {
  instantParts,
  valueState,
} from "../../../sources/collected-focus/fields";
import {
  Instant,
  missingLabel,
  unitMissingLabel,
  zeroLabel,
} from "../../../sources/collected-focus/labels";
import {
  decimal,
  money as sourceMoney,
} from "../../../sources/collected-focus/values";
import { t } from "./copy";
export function Missing({ locale: l }: { locale: Locale }) {
  return (
    <strong className="pricing-state" data-pricing-value="missing">
      {missingLabel(l)}
    </strong>
  );
}
export function SourceText({
  value,
  locale: l,
}: {
  value: string | null;
  locale: Locale;
}) {
  return value === null ? <Missing locale={l} /> : <bdi dir="ltr">{value}</bdi>;
}
export function SourceAmount({
  value,
  currency,
  unit,
  note,
  locale: l,
}: {
  value: string | null;
  currency: string;
  unit?: string | null;
  note?: boolean;
  locale: Locale;
}) {
  const state = valueState(value);
  if (state === "missing") return <Missing locale={l} />;
  return (
    <span className="pricing-source-value" data-pricing-value={state}>
      <bdi dir="ltr">
        {sourceMoney(value, currency, l)}
        {unit !== undefined && unit?.trim() ? ` / ${unit.trim()}` : ""}
      </bdi>
      {unit !== undefined && !unit?.trim() && (
        <strong className="pricing-state pricing-state-note">
          {unitMissingLabel(l)}
        </strong>
      )}
      {note && state === "zero" && (
        <strong className="pricing-state pricing-state-note">
          {zeroLabel(l)}
        </strong>
      )}
    </span>
  );
}
export function SourceQuantity({
  value,
  unit,
  note,
  locale: l,
}: {
  value: string | null;
  unit: string | null;
  note?: boolean;
  locale: Locale;
}) {
  const state = valueState(value);
  return (
    <span
      className="pricing-source-value"
      data-pricing-value={state === "missing" ? undefined : state}
    >
      {state === "missing" ? (
        <Missing locale={l} />
      ) : (
        <bdi dir="ltr">
          {decimal(value, l)}
          {unit?.trim() ? ` ${unit.trim()}` : ""}
        </bdi>
      )}
      {!unit?.trim() && (
        <strong className="pricing-state pricing-state-note">
          {unitMissingLabel(l)}
        </strong>
      )}
      {note && state === "zero" && (
        <strong className="pricing-state pricing-state-note">
          {zeroLabel(l)}
        </strong>
      )}
    </span>
  );
}
export function SourceInstant({
  value,
  locale: l,
}: {
  value: string;
  locale: Locale;
}) {
  return instantParts(value) ? (
    <bdi dir="ltr" className="pricing-instant">
      <Instant value={value} /> <span className="pricing-tz">UTC</span>
    </bdi>
  ) : (
    <>
      <bdi dir="ltr" data-pricing-instant="raw">
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
export function SourcePeriod({
  start,
  end,
  locale: l,
}: {
  start: string;
  end: string;
  locale: Locale;
}) {
  return (
    <span className="pricing-period">
      <span>
        <small>{t(l, "Start", "시작", "البداية")}</small>{" "}
        <SourceInstant value={start} locale={l} />
      </span>
      <span>
        <small>{t(l, "End", "종료", "النهاية")}</small>{" "}
        <SourceInstant value={end} locale={l} />
      </span>
    </span>
  );
}
