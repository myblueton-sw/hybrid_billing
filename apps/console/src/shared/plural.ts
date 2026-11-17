import type { Locale } from "./types";

export type Plural = { en: [string, string]; ko: string; ar: [string, string, string, string, string, string] };

const tags: Record<Locale, string> = { en: "en-US", ko: "ko-KR", ar: "ar" };
const arOrder = ["zero", "one", "two", "few", "many", "other"];

export function category(l: Locale, n: number): string {
  return new Intl.PluralRules(tags[l]).select(n);
}

export function counted(l: Locale, n: number, forms: Plural, slots: Record<string, string> = {}): string {
  const c = category(l, n);
  const base = l === "en" ? (c === "one" ? forms.en[0] : forms.en[1]) : l === "ko" ? forms.ko : forms.ar[Math.max(0, arOrder.indexOf(c))];
  return Object.entries({ n: String(n), ...slots }).reduce((s, [k, v]) => s.split("{" + k + "}").join(v), base);
}
