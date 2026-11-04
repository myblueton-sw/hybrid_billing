import type { Locale } from "../../../../shared/types";
import { money } from "../../../sources/collected-focus/values";
export function reviewMoney(
  value: string,
  currency: string,
  locale: Locale,
  change = false,
) {
  const n = BigInt(value),
    sign = n < 0n ? "−" : change && n > 0n ? "+" : "";
  return sign + money((n < 0n ? -n : n).toString(), currency, locale);
}
