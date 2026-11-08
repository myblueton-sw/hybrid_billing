import type { Locale } from "../../../../shared/types";
import type { Amount, Basis, Measure, OverviewData } from "./focus-api";
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
export const amountFor = (
  amounts: Amount[],
  currency: string,
  basis: Basis,
): Measure | undefined => amounts.find((a) => a.currency === currency)?.[basis];
export function chartSeries(data: OverviewData, currency: string) {
  const available = data.months.map((m) => m.period).sort();
  if (!available.length) return [];
  const end =
    data.selection.period === "all" ? available.at(-1)! : data.selection.period;
  const [year, month] = end.split("-").map(Number),
    endIndex = year * 12 + month - 1;
  const [firstYear, firstMonth] = available[0].split("-").map(Number);
  const start = Math.max(firstYear * 12 + firstMonth - 1, endIndex - 11);
  return Array.from({ length: endIndex - start + 1 }, (_, i) => {
    const n = start + i,
      period = `${Math.floor(n / 12)}-${String((n % 12) + 1).padStart(2, "0")}`;
    const amounts = data.months.find((m) => m.period === period)?.amounts ?? [];
    return {
      period,
      measure: amountFor(amounts, currency, data.selection.basis),
    };
  });
}
export function signedScale(values: (string | null | undefined)[]) {
  const known = values
    .filter((v): v is string => v !== null && v !== undefined)
    .map(BigInt);
  const maximum = known.reduce((m, n) => (n > m ? n : m), 0n),
    minimum = known.reduce((m, n) => (n < m ? n : m), 0n);
  const magnitude = maximum > -minimum ? maximum : -minimum;
  const power = 10n ** BigInt((magnitude || scale).toString().length - 1);
  const tick = [1n, 2n, 5n, 10n]
    .map((n) => n * power)
    .find((n) => n >= (magnitude || scale))!;
  const edge = tick + (tick / 10n || 1n),
    positive = maximum > 0n || minimum === 0n ? edge : 0n,
    negative = minimum < 0n ? edge : 0n;
  const y = (value: bigint) =>
    Number(((positive - value) * 10000n) / (positive + negative)) / 100;
  const ticks =
    positive && negative
      ? [tick, 0n, -tick]
      : positive
        ? [tick, tick / 2n, 0n]
        : [0n, -tick / 2n, -tick];
  return { y, zero: y(0n), ticks: [...new Set(ticks)] };
}
