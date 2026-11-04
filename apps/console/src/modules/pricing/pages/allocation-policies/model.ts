import type { OverviewData } from "../../../sources/collected-focus/api";
import { digest } from "../../../../shared/review-store";
export const profile = "source-precision-review-v1";
export type Target = { id: string; label: string; timezone: string };
export type Weight = { target: string; percentage: string };
export type Draft = { name: string; weights: Weight[] };
export type Prepared = {
  data: OverviewData;
  targets: Target[];
  currency: string;
  context: string;
};
export type Result = {
  context: string;
  profile: string;
  currency: string;
  source: string;
  assigned: string;
  unassigned: string;
  remainder: string;
  rows: { target: string; percentage: string; amount: string }[];
  name: string;
};
export const denominator = 1000000n;
export function percentage(value: string): bigint | null {
  if (!/^(?:0|[1-9]\d?|100)(?:\.\d{1,4})?$/.test(value)) return null;
  const [whole, fraction = ""] = value.split(".");
  const n = BigInt(whole) * 10000n + BigInt(fraction.padEnd(4, "0"));
  return n <= denominator ? n : null;
}
export async function prepare(
  data: OverviewData,
  requestedCurrency: string,
): Promise<Prepared> {
  if (
    data.selection.account ||
    data.selection.q ||
    data.after !== 0 ||
    data.account_after !== 0
  )
    throw Error("restricted_source");
  const tenants = [...data.tenants].sort((a, b) =>
    a.key < b.key ? -1 : a.key > b.key ? 1 : 0,
  );
  if (tenants.length > 100) throw Error("target_limit");
  if (new Set(tenants.map((t) => t.key)).size !== tenants.length)
    throw Error("target_duplicate");
  const targets = await Promise.all(
    tenants.map(async (t) => ({
      id: await digest(t.key),
      label: t.key,
      timezone: t.timezone,
    })),
  );
  if (new Set(targets.map((t) => t.id)).size !== targets.length)
    throw Error("target_duplicate");
  const currency = requestedCurrency;
  const context = await digest(
    JSON.stringify({
      profile,
      company: data.company,
      source: data.source_snapshot,
      selection: data.snapshot,
      scope: data.selection,
      currency,
      targets: targets.map((t) => t.id),
    }),
  );
  return { data, targets, currency, context };
}
export const emptyDraft = (source: Prepared): Draft => ({
  name: "",
  weights: source.targets.map((t) => ({ target: t.id, percentage: "" })),
});
export function issues(
  draft: Draft,
  source: Prepared,
  saving = false,
): string[] {
  const errors = new Set<string>();
  if (saving && !draft.name.trim()) errors.add("name_required");
  if (draft.name.length > 120) errors.add("name_bound");
  const ids = draft.weights.map((w) => w.target),
    expected = source.targets.map((t) => t.id);
  if (!expected.length) errors.add("targets_required");
  if (expected.length > 100 || draft.weights.length > 100)
    errors.add("target_limit");
  if (
    new Set(ids).size !== ids.length ||
    ids.length !== expected.length ||
    ids.some((id) => !expected.includes(id))
  )
    errors.add("target_manifest");
  const values = draft.weights.map((w) => percentage(w.percentage));
  if (values.some((v) => v === null)) errors.add("percentage_invalid");
  const sum = values.reduce<bigint>((n, v) => n + (v ?? 0n), 0n);
  if (sum === 0n) errors.add("percentage_required");
  if (sum > denominator) errors.add("percentage_exceeded");
  if (source.data.selection.period === "all") errors.add("month_required");
  const amount = source.data.amounts.find(
    (a) => a.currency === source.currency,
  )?.[source.data.selection.basis];
  if (!amount || amount.coefficient === null || !amount.present)
    errors.add("amount_unavailable");
  else if (amount.missing) errors.add("amount_partial");
  return [...errors];
}
export function simulate(draft: Draft, source: Prepared): Result {
  if (issues(draft, source).length) throw Error("preview_invalid");
  const coefficient = source.data.amounts.find(
    (a) => a.currency === source.currency,
  )![source.data.selection.basis].coefficient!;
  const amount = BigInt(coefficient),
    rows = draft.weights.map((w) => ({
      target: w.target,
      percentage: w.percentage,
      amount: ((amount * percentage(w.percentage)!) / denominator).toString(),
    }));
  const sum = rows.reduce((n, r) => n + BigInt(r.amount), 0n),
    weight = draft.weights.reduce((n, w) => n + percentage(w.percentage)!, 0n),
    intended = (amount * weight) / denominator;
  return {
    context: source.context,
    profile,
    currency: source.currency,
    source: coefficient,
    assigned: sum.toString(),
    unassigned: (amount - intended).toString(),
    remainder: (intended - sum).toString(),
    rows,
    name: draft.name,
  };
}
export function comparable(a: Result, b: Result): boolean {
  return (
    a.context === b.context &&
    a.profile === b.profile &&
    a.currency === b.currency &&
    a.source === b.source &&
    a.rows.length === b.rows.length &&
    new Set(a.rows.map((r) => r.target)).size === a.rows.length &&
    a.rows.every((r) => b.rows.some((v) => v.target === r.target))
  );
}
export const restricted = (params: URLSearchParams) =>
  !!(
    params.get("focus-account") ||
    params.get("focus-q")?.trim() ||
    Number(params.get("focus-after") ?? 0) !== 0 ||
    Number(params.get("focus-account_after") ?? 0) !== 0
  );
