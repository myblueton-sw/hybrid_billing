import type {
  OverviewData,
  FocusRow,
} from "../../../sources/collected-focus/api";
import { digest } from "../../../../shared/review-store";
export const profile = "unit-price-source-precision-review-v1";
export const scale = 10n ** 18n,
  bound = 10n ** 38n,
  percentScale = 1000000n;
export type Draft = {
  name: string;
  unitPrice: string;
  denominator: string;
  discount: string;
  fee: string;
  target: string;
  reason: string;
  reference: string;
};
export type Prepared = { data: OverviewData; row: FocusRow; context: string };
export type Division = {
  numerator: string;
  denominator: string;
  quotient: string;
  remainder: string;
};
export type Result = {
  context: string;
  profile: string;
  row: string;
  currency: string;
  unit: string;
  name: string;
  unitPrice: string;
  denominator: string;
  discount: string;
  fee: string;
  base: string;
  discounted: string;
  total: string;
  trace: { base: Division; discount: Division };
};
export type Adjustment = {
  context: string;
  candidate: string;
  target: string;
  delta: string;
  reason: string;
  reference: string;
};
export const emptyDraft = (): Draft => ({
  name: "",
  unitPrice: "",
  denominator: "",
  discount: "",
  fee: "",
  target: "",
  reason: "",
  reference: "",
});
export function decimal(value: string, signed = false): bigint | null {
  if (
    !new RegExp(
      `^${signed ? "-?" : ""}(?:0|[1-9]\\d{0,19})(?:\\.\\d{1,18})?$`,
    ).test(value)
  )
    return null;
  const negative = value.startsWith("-"),
    [whole, fraction = ""] = (negative ? value.slice(1) : value).split(".");
  const result =
    (BigInt(whole) * scale + BigInt(fraction.padEnd(18, "0"))) *
    (negative ? -1n : 1n);
  return result > -bound && result < bound ? result : null;
}
export function percentage(value: string): bigint | null {
  if (!/^(?:0|[1-9]\d?|100)(?:\.\d{1,4})?$/.test(value)) return null;
  const [a, b = ""] = value.split(".");
  const n = BigInt(a) * 10000n + BigInt(b.padEnd(4, "0"));
  return n <= percentScale ? n : null;
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
      unit: row.pricing_unit,
    }),
  );
  return { data, row, context };
}
export function issues(
  draft: Draft,
  source: Prepared,
  saving = false,
): string[] {
  const errors = new Set<string>();
  if (saving && !draft.name.trim()) errors.add("name_required");
  if (
    draft.name.length > 120 ||
    draft.reason.length > 300 ||
    draft.reference.length > 300 ||
    draft.target.length > 40
  )
    errors.add("draft_bound");
  if (!source.data.rows.some((r) => r.id === source.row.id))
    errors.add("row_unavailable");
  if (source.data.selection.period === "all") errors.add("month_required");
  else if (source.data.selection.period !== source.row.charge_month)
    errors.add("month_mismatch");
  if (source.row.pricing_quantity === null) errors.add("quantity_unavailable");
  if (!source.row.pricing_unit?.trim()) errors.add("unit_unavailable");
  if (decimal(draft.unitPrice) === null) errors.add("price_invalid");
  const denominator = decimal(draft.denominator);
  if (denominator === null || denominator <= 0n)
    errors.add("denominator_invalid");
  if (percentage(draft.discount) === null) errors.add("discount_invalid");
  if (decimal(draft.fee) === null) errors.add("fee_invalid");
  return [...errors];
}
function checked(value: bigint) {
  if (value <= -bound || value >= bound) throw Error("amount_overflow");
  return value.toString();
}
function divide(n: bigint, d: bigint): Division {
  if (d <= 0n) throw Error("denominator_invalid");
  return {
    numerator: n.toString(),
    denominator: d.toString(),
    quotient: checked(n / d),
    remainder: (n % d).toString(),
  };
}
export function simulate(draft: Draft, source: Prepared): Result {
  const errors = issues(draft, source);
  if (errors.length) throw Error(errors[0]);
  const base = divide(
    BigInt(source.row.pricing_quantity!) * decimal(draft.unitPrice)!,
    decimal(draft.denominator)!,
  );
  const discount = divide(
    BigInt(base.quotient) * (percentScale - percentage(draft.discount)!),
    percentScale,
  );
  const fee = decimal(draft.fee)!;
  return {
    context: source.context,
    profile,
    row: source.row.id,
    currency: source.row.currency,
    unit: source.row.pricing_unit!,
    name: draft.name,
    unitPrice: draft.unitPrice,
    denominator: draft.denominator,
    discount: draft.discount,
    fee: draft.fee,
    base: base.quotient,
    discounted: discount.quotient,
    total: checked(BigInt(discount.quotient) + fee),
    trace: { base, discount },
  };
}
export function adjustmentIssues(draft: Draft): string[] {
  const errors = [];
  if (decimal(draft.target, true) === null) errors.push("target_invalid");
  if (!draft.reason.trim() || draft.reason.length > 300)
    errors.push("reason_required");
  if (!draft.reference.trim() || draft.reference.length > 300)
    errors.push("reference_required");
  return errors;
}
export function adjust(
  draft: Draft,
  source: Prepared,
  result: Result,
): Adjustment {
  if (
    result.context !== source.context ||
    result.profile !== profile ||
    result.row !== source.row.id ||
    result.currency !== source.row.currency ||
    (["name", "unitPrice", "denominator", "discount", "fee"] as const).some(
      (k) => result[k] !== draft[k],
    )
  )
    throw Error("candidate_stale");
  const errors = adjustmentIssues(draft);
  if (errors.length) throw Error(errors[0]);
  const target = decimal(draft.target, true)!;
  return {
    context: source.context,
    candidate: result.total,
    target: target.toString(),
    delta: checked(target - BigInt(result.total)),
    reason: draft.reason,
    reference: draft.reference,
  };
}
export function comparable(a: Result, b: Result): boolean {
  return (
    a.context === b.context &&
    a.profile === b.profile &&
    a.row === b.row &&
    a.currency === b.currency &&
    a.unit === b.unit
  );
}
