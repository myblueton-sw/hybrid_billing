import type { Locale } from "../../../../shared/types";
import { decimal as formatDecimal } from "../../../sources/collected-focus/values";
import { scale, type Rational } from "./model";
export const coefficient = (n: string, l: Locale) => formatDecimal(n, l);
export function money(n: string, currency: string, l: Locale, signed = false) {
  const b = BigInt(n);
  return `${b < 0n ? "−" : signed && b > 0n ? "+" : ""}${currency} ${formatDecimal((b < 0n ? -b : b).toString(), l)}`;
}
function gcd(a: bigint, b: bigint): bigint {
  while (b) {
    const r = a % b;
    a = b;
    b = r;
  }
  return a;
}
export const displayNumeratorBound = 10n ** 130n,
  displayDenominatorBound = 10n ** 90n;
function parts(r: Rational): [bigint, bigint] {
  if (
    r.numerator.length > 131 ||
    r.denominator.length > 90 ||
    !/^-?\d+$/.test(r.numerator) ||
    !/^\d+$/.test(r.denominator)
  )
    throw Error("display_unavailable");
  const n = BigInt(r.numerator),
    d = BigInt(r.denominator);
  if (
    n <= -displayNumeratorBound ||
    n >= displayNumeratorBound ||
    d <= 0n ||
    d >= displayDenominatorBound
  )
    throw Error("display_unavailable");
  return [n, d];
}
export function exact(r: Rational, currency: string, signed = false) {
  let [n, d] = parts(r);
  d *= scale;
  const sign = n < 0n ? "−" : signed && n > 0n ? "+" : "";
  n = n < 0n ? -n : n;
  const g = gcd(n, d);
  n /= g;
  d /= g;
  let test = d;
  while (test % 2n === 0n) test /= 2n;
  while (test % 5n === 0n) test /= 5n;
  if (test !== 1n)
    return { text: `${sign}${currency} ${n}/${d}`, fraction: true };
  const whole = n / d;
  let rem = n % d,
    text = whole.toString();
  if (rem) {
    text += ".";
    let digits = 0;
    while (rem && digits < 512) {
      digits++;
      rem *= 10n;
      text += (rem / d).toString();
      rem %= d;
    }
  }
  if (rem) return { text: `${sign}${currency} ${n}/${d}`, fraction: true };
  return { text: `${sign}${currency} ${text}`, fraction: false };
}
export function subtract(a: Rational, b: Rational): Rational {
  const [an, ad] = parts(a),
    [bn, bd] = parts(b);
  const result = {
    numerator: (an * bd - bn * ad).toString(),
    denominator: (ad * bd).toString(),
  };
  parts(result);
  return result;
}
export const asRational = (n: string): Rational => ({
  numerator: n,
  denominator: "1",
});
export const fieldId = (key: string) =>
  "fx-" + key.replace(/[A-Z]/g, (c) => "-" + c.toLowerCase());
