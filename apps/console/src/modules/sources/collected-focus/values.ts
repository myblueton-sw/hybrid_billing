import type { Locale } from "../../../shared/types";
export const scale = 10n ** 18n;
export function decimal(value: string | null, locale: Locale) {
  if (value === null) return "—";
  const coefficient = BigInt(value),
    negative = coefficient < 0n,
    magnitude = negative ? -coefficient : coefficient;
  const whole = new Intl.NumberFormat(
    locale === "ar" ? "ar-AE-u-nu-latn" : locale === "ko" ? "ko-KR" : "en-US",
  ).format(magnitude / scale);
  const fraction = (magnitude % scale)
    .toString()
    .padStart(18, "0")
    .replace(/0+$/, "")
    .padEnd(2, "0");
  return `${negative ? "-" : ""}${whole}.${fraction}`;
}
export const money = (value: string | null, currency: string, locale: Locale) =>
  `${currency} ${decimal(value, locale)}`;
