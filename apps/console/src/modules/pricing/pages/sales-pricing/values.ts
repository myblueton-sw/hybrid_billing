import type { Locale } from "../../../../shared/types";
import { decimal as formatDecimal } from "../../../sources/collected-focus/values";
export const number = (v: string | null, l: Locale) => formatDecimal(v, l);
export function money(v: string, currency: string, l: Locale, signed = false) {
  const n = BigInt(v);
  return `${n < 0n ? "−" : signed && n > 0n ? "+" : ""}${currency} ${formatDecimal((n < 0n ? -n : n).toString(), l)}`;
}
