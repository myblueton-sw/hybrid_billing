import type { Amount, Basis, OverviewData } from "./api";
export type Baseline = {
  identity: string;
  source_snapshot: string;
  snapshot: string;
  observed_at: string;
  row_count: number;
  amounts: Amount[];
};
export type Comparison = {
  currency: string;
  baseline: Amount | null;
  current: Amount | null;
  billed_cost: string | null;
  effective_cost: string | null;
};
function identity(d: OverviewData) {
  return JSON.stringify({
    profile: "b08-source-baseline-v1",
    company: d.company,
    company_mapping: d.company_mapping,
    tenants: d.tenants
      .map((t) => [t.key, t.timezone])
      .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0)),
    versions: [...d.versions].sort(),
    period_basis: d.period_basis,
    provider: d.selection.provider,
    account: d.selection.account,
    period: d.selection.period,
  });
}
export function makeBaseline(data: OverviewData): Baseline {
  return {
    identity: identity(data),
    source_snapshot: data.source_snapshot,
    snapshot: data.snapshot,
    observed_at: data.observed_at,
    row_count: data.row_count,
    amounts: structuredClone(data.amounts),
  };
}
export function compatible(baseline: Baseline, data: OverviewData) {
  return (
    data.scale === 18 &&
    data.financial_authority === false &&
    data.classification === "collected-focus-sample-query-only" &&
    baseline.source_snapshot === data.source_snapshot &&
    baseline.identity === identity(data)
  );
}
function difference(
  baseline: Amount | null,
  current: Amount | null,
  basis: Basis,
) {
  if (
    !baseline ||
    !current ||
    baseline.rows === 0 ||
    current.rows === 0 ||
    baseline[basis].missing !== 0 ||
    current[basis].missing !== 0 ||
    baseline[basis].present !== baseline.rows ||
    current[basis].present !== current.rows ||
    baseline[basis].coefficient === null ||
    current[basis].coefficient === null
  )
    return null;
  return (
    BigInt(current[basis].coefficient) - BigInt(baseline[basis].coefficient)
  ).toString();
}
export function compare(
  baseline: Baseline,
  data: OverviewData,
): Comparison[] | null {
  if (!compatible(baseline, data)) return null;
  return [
    ...new Set([...baseline.amounts, ...data.amounts].map((a) => a.currency)),
  ]
    .sort()
    .map((currency) => {
      const before =
        baseline.amounts.find((a) => a.currency === currency) ?? null;
      const current = data.amounts.find((a) => a.currency === currency) ?? null;
      return {
        currency,
        baseline: before,
        current,
        billed_cost: difference(before, current, "billed_cost"),
        effective_cost: difference(before, current, "effective_cost"),
      };
    });
}
