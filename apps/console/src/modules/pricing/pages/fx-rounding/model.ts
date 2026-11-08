import type {
  OverviewData,
  FocusRow,
} from "../../../sources/collected-focus/api";
import { digest } from "../../../../shared/review-store";
export const profile = "manual-source-cost-fx-quantum-v1";
export const scale = 10n ** 18n,
  bound = 10n ** 38n,
  intermediateBound = 10n ** 84n;
export const currencies = ["USD", "EUR", "KRW", "JPY", "AED", "GBP"] as const;
export const modes = [
  "toward_zero",
  "away_from_zero",
  "floor",
  "ceiling",
  "half_up",
  "half_even",
] as const;
export type Mode = (typeof modes)[number];
export const draftKeys = [
  "name",
  "target",
  "treatment",
  "rate",
  "quoteUnit",
  "provider",
  "reference",
  "quoteDate",
  "reviewDate",
  "maxAge",
  "markup",
  "fee",
  "quantum",
  "mode",
] as const;
export type Draft = Record<(typeof draftKeys)[number], string>;
export type Prepared = { data: OverviewData; row: FocusRow; context: string };
export type Rational = { numerator: string; denominator: string };
export type Result = {
  profile: string;
  context: string;
  row: string;
  sourceCurrency: string;
  currency: string;
  source: string;
  draft: Draft;
  age: number;
  normalized: Rational;
  trace: { quoted: Rational; marked: Rational; withFee: Rational };
  total: string;
  difference: Rational;
  rounding: {
    numerator: string;
    denominator: string;
    quotient: string;
    remainder: string;
    count: string;
  };
};
export const emptyDraft = (): Draft =>
  Object.fromEntries(draftKeys.map((k) => [k, ""])) as Draft;
export function decimal(value: string, signed = false): bigint | null {
  if (
    !new RegExp(
      `^${signed ? "-?" : ""}(?:0|[1-9]\\d{0,19})(?:\\.\\d{1,18})?$`,
    ).test(value)
  )
    return null;
  const negative = value.startsWith("-"),
    [a, b = ""] = (negative ? value.slice(1) : value).split(".");
  const n =
    (BigInt(a) * scale + BigInt(b.padEnd(18, "0"))) * (negative ? -1n : 1n);
  return n > -bound && n < bound ? n : null;
}
export function percentage(value: string): bigint | null {
  if (!/^(?:0|[1-9]\d?|100)(?:\.\d{1,4})?$/.test(value)) return null;
  const [a, b = ""] = value.split("."),
    n = BigInt(a) * 10000n + BigInt(b.padEnd(4, "0"));
  return n <= 1000000n ? n : null;
}
export function dateDays(value: string): number | null {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value) || value.startsWith("0000"))
    return null;
  const ms = Date.parse(value + "T00:00:00Z");
  return Number.isFinite(ms) &&
    new Date(ms).toISOString().slice(0, 10) === value
    ? ms / 86400000
    : null;
}
export async function prepare(
  data: OverviewData,
  id: string,
): Promise<Prepared> {
  if (new Set(data.rows.map((r) => r.id)).size !== data.rows.length)
    throw Error("source_duplicate");
  const row = data.rows.find((r) => r.id === id);
  if (!row) throw Error("row_unavailable");
  const context = await digest(
    JSON.stringify({
      profile,
      company: data.company,
      source: data.source_snapshot,
      selection: data.snapshot,
      query: data.selection,
      after: data.after,
      accountAfter: data.account_after,
      row: row.id,
      currency: row.currency,
    }),
  );
  return { data, row, context };
}
export function sourceAmount(s: Prepared): string | null {
  return s.row[s.data.selection.basis];
}
export function issues(d: Draft, s: Prepared, saving = false): string[] {
  const errors = new Set<string>();
  if (saving && !d.name.trim()) errors.add("name_required");
  if (
    draftKeys.some(
      (k) =>
        typeof d[k] !== "string" ||
        d[k].length >
          (["provider", "reference"].includes(k)
            ? 300
            : k === "name"
              ? 120
              : 40),
    )
  )
    errors.add("draft_bound");
  if (!s.data.rows.some((r) => r.id === s.row.id))
    errors.add("row_unavailable");
  const amount = sourceAmount(s);
  if (
    amount === null ||
    !/^-?\d{1,38}$/.test(amount) ||
    BigInt(amount) <= -bound ||
    BigInt(amount) >= bound
  )
    errors.add("amount_unavailable");
  if (
    s.data.selection.period === "all" ||
    s.data.selection.period !== s.row.charge_month
  )
    errors.add("month_required");
  if (
    !(currencies as readonly string[]).includes(d.target) ||
    d.target === s.row.currency
  )
    errors.add("target_invalid");
  if (d.treatment !== "unconverted") errors.add("conversion_declaration");
  for (const [key, code] of [
    ["rate", "rate_invalid"],
    ["quoteUnit", "quote_unit_invalid"],
    ["quantum", "quantum_invalid"],
  ] as const) {
    const n = decimal(d[key]);
    if (n === null || n <= 0n) errors.add(code);
  }
  if (percentage(d.markup) === null) errors.add("markup_invalid");
  if (decimal(d.fee) === null) errors.add("fee_invalid");
  if (!(modes as readonly string[]).includes(d.mode))
    errors.add("mode_invalid");
  if (!d.provider.trim() || !d.reference.trim())
    errors.add("evidence_required");
  const quote = dateDays(d.quoteDate),
    review = dateDays(d.reviewDate);
  if (quote === null || review === null) errors.add("date_invalid");
  const validAge =
    /^(?:0|[1-9]\d{0,4})$/.test(d.maxAge) && Number(d.maxAge) <= 36600;
  if (!validAge) errors.add("age_invalid");
  if (quote !== null && review !== null) {
    if (quote > review) errors.add("future_quote");
    else if (validAge && review - quote > Number(d.maxAge))
      errors.add("stale_quote");
  }
  return [...errors];
}
function checked(n: bigint, limit = intermediateBound) {
  if (n <= -limit || n >= limit) throw Error("amount_overflow");
  return n;
}
function rational(n: bigint, d: bigint): Rational {
  if (d <= 0n) throw Error("denominator_invalid");
  return {
    numerator: checked(n).toString(),
    denominator: checked(d).toString(),
  };
}
export function roundedCount(n: bigint, d: bigint, mode: string): bigint {
  checked(n);
  checked(d);
  if (d <= 0n || !(modes as readonly string[]).includes(mode))
    throw Error("rounding_invalid");
  const q = n / d,
    r = n % d;
  if (r === 0n) return q;
  const sign = n < 0n ? -1n : 1n,
    abs = r < 0n ? -r : r;
  if (mode === "toward_zero") return q;
  if (mode === "away_from_zero") return q + sign;
  if (mode === "floor") return n < 0n ? q - 1n : q;
  if (mode === "ceiling") return n > 0n ? q + 1n : q;
  if (abs * 2n > d || (abs * 2n === d && (mode === "half_up" || q % 2n !== 0n)))
    return q + sign;
  return q;
}
export function simulate(d: Draft, s: Prepared): Result {
  const errors = issues(d, s);
  if (errors.length) throw Error(errors[0]);
  const c = BigInt(sourceAmount(s)!),
    r = decimal(d.rate)!,
    u = decimal(d.quoteUnit)!,
    m = percentage(d.markup)!,
    f = decimal(d.fee)!,
    q = decimal(d.quantum)!;
  const quoted = rational(checked(c * r), u),
    marked = rational(checked(c * r * (1000000n + m)), checked(u * 1000000n));
  const denominator = BigInt(marked.denominator),
    numerator = checked(BigInt(marked.numerator) + checked(f * denominator));
  const withFee = rational(numerator, denominator),
    roundingDenominator = checked(denominator * q),
    count = roundedCount(numerator, roundingDenominator, d.mode),
    total = checked(count * q, bound);
  return {
    profile,
    context: s.context,
    row: s.row.id,
    sourceCurrency: s.row.currency,
    currency: d.target,
    source: c.toString(),
    draft: { ...d },
    age: dateDays(d.reviewDate)! - dateDays(d.quoteDate)!,
    normalized: rational(r, u),
    trace: { quoted, marked, withFee },
    total: total.toString(),
    difference: rational(checked(total * denominator - numerator), denominator),
    rounding: {
      numerator: numerator.toString(),
      denominator: roundingDenominator.toString(),
      quotient: (numerator / roundingDenominator).toString(),
      remainder: (numerator % roundingDenominator).toString(),
      count: count.toString(),
    },
  };
}
export function comparable(a: Result, b: Result) {
  return (
    a.context === b.context &&
    a.profile === b.profile &&
    a.row === b.row &&
    a.currency === b.currency &&
    a.sourceCurrency === b.sourceCurrency &&
    a.draft.treatment === b.draft.treatment
  );
}
