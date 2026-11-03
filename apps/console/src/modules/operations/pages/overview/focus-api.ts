import { request, ServiceError } from "../../../sources/pages/live-imports/api";
export { ServiceError };
export type Basis = "billed_cost" | "effective_cost";
export type Measure = {
  coefficient: string | null;
  present: number;
  missing: number;
};
export type Amount = {
  currency: string;
  rows: number;
  billed_cost: Measure;
  effective_cost: Measure;
};
export type Account = {
  id: string;
  tenant: string;
  provider: string;
  billing_account: string;
  sub_account: string | null;
  native_ancestry: string;
  rows: number;
  amounts: Amount[];
};
export type FocusRow = {
  id: string;
  account: string;
  provider: string;
  billing_account: string;
  sub_account: string | null;
  service: string | null;
  currency: string;
  charge_month: string;
  charge_start: string;
  charge_end: string;
  billed_cost: string | null;
  effective_cost: string | null;
  evidence_class: string;
  batch: string;
  ordinal: string;
};
export type OverviewData = {
  schema: 1;
  scale: 18;
  snapshot: string;
  source_snapshot: string;
  selection: {
    period: string;
    provider: string;
    account: string;
    basis: Basis;
    q: string;
  };
  classification: string;
  financial_authority: false;
  observed_at: string;
  period_basis: string;
  company: string | null;
  company_mapping: string;
  tenants: { key: string; timezone: string }[];
  versions: string[];
  delivery_count: number;
  coverage: {
    slices: number;
    finalized_slices: number;
    complete_through: string | null;
    expectations: string;
  };
  provenance: Record<string, number>;
  options: { providers: string[]; periods: string[]; currencies: string[] };
  authorized_rows: number;
  row_count: number;
  amounts: Amount[];
  providers: { provider: string; amounts: Amount[] }[];
  months: { period: string; amounts: Amount[] }[];
  account_count: number;
  account_matches: number;
  accounts: Account[];
  next_account_after: number | null;
  rows: FocusRow[];
  next_after: number | null;
  after: number;
  account_after: number;
};
const sha = (v: unknown): v is string =>
  typeof v === "string" && /^[a-f0-9]{64}$/.test(v);
const text = (v: unknown, max = 4096): v is string =>
  typeof v === "string" && v.length <= max;
const count = (v: unknown): v is number =>
  Number.isInteger(v) && Number(v) >= 0 && Number(v) <= 10000;
const numeric = (v: unknown): v is string | null =>
  v === null || (typeof v === "string" && /^-?(?:0|[1-9]\d{0,37})$/.test(v));
const currency = (v: unknown): v is string =>
  typeof v === "string" && /^[A-Z]{3}$/.test(v);
function validAmounts(list: Amount[], expected?: number) {
  return (
    Array.isArray(list) &&
    list.length <= 64 &&
    new Set(list.map((a) => a?.currency)).size === list.length &&
    list.every(
      (a) =>
        a &&
        currency(a.currency) &&
        count(a.rows) &&
        (["billed_cost", "effective_cost"] as const).every((b) => {
          const m = a[b];
          return (
            m &&
            count(m.present) &&
            count(m.missing) &&
            m.present + m.missing === a.rows &&
            numeric(m.coefficient) &&
            (m.present === 0 ? m.coefficient === null : m.coefficient !== null)
          );
        }),
    ) &&
    (expected === undefined ||
      list.reduce((n, a) => n + a.rows, 0) === expected)
  );
}
function consistentProviders(data: OverviewData) {
  if (
    data.providers.reduce(
      (n, p) => n + p.amounts.reduce((m, a) => m + a.rows, 0),
      0,
    ) !== data.row_count
  )
    return false;
  return data.amounts.every((a) =>
    (["billed_cost", "effective_cost"] as const).every((b) => {
      const parts = data.providers.flatMap((p) =>
        p.amounts.filter((c) => c.currency === a.currency),
      );
      const present = parts.reduce((n, v) => n + v[b].present, 0);
      return (
        present === a[b].present &&
        parts.reduce((n, v) => n + v[b].missing, 0) === a[b].missing &&
        (present
          ? parts
              .reduce((n, v) => n + BigInt(v[b].coefficient ?? "0"), 0n)
              .toString()
          : null) === a[b].coefficient
      );
    }),
  );
}
export function validOverview(
  d: OverviewData,
  query: URLSearchParams,
): boolean {
  try {
    return (
      !!d &&
      d.schema === 1 &&
      d.scale === 18 &&
      sha(d.snapshot) &&
      sha(d.source_snapshot) &&
      d.classification === "collected-focus-sample-query-only" &&
      d.financial_authority === false &&
      d.period_basis === "charge-start-in-tenant-timezone-no-proration" &&
      text(d.observed_at, 64) &&
      (d.company === null || text(d.company)) &&
      d.company_mapping === "synthetic-sample-authorization" &&
      Array.isArray(d.tenants) &&
      d.tenants.length <= 10000 &&
      d.tenants.every((t) => text(t.key) && text(t.timezone, 128)) &&
      Array.isArray(d.versions) &&
      d.versions.length <= 32 &&
      d.versions.every((v) => text(v, 16)) &&
      count(d.delivery_count) &&
      d.coverage &&
      count(d.coverage.slices) &&
      count(d.coverage.finalized_slices) &&
      d.coverage.finalized_slices <= d.coverage.slices &&
      (d.coverage.complete_through === null ||
        text(d.coverage.complete_through, 64)) &&
      d.coverage.expectations === "operator-asserted-unverified" &&
      Object.entries(d.provenance).every(
        ([k, v]) =>
          [
            "official-public",
            "official-public-subset",
            "synthetic-contract-fixture",
            "unclassified",
          ].includes(k) && count(v),
      ) &&
      Object.values(d.provenance).reduce((n, v) => n + v, 0) === d.row_count &&
      d.options &&
      Array.isArray(d.options.providers) &&
      d.options.providers.length <= 128 &&
      d.options.providers.every((v) => text(v)) &&
      Array.isArray(d.options.periods) &&
      d.options.periods.length <= 120 &&
      d.options.periods.every((v) => /^\d{4}-(0[1-9]|1[0-2])$/.test(v)) &&
      Array.isArray(d.options.currencies) &&
      d.options.currencies.length <= 64 &&
      d.options.currencies.every(currency) &&
      count(d.authorized_rows) &&
      count(d.row_count) &&
      d.row_count <= d.authorized_rows &&
      validAmounts(d.amounts, d.row_count) &&
      d.selection &&
      (["period", "provider", "account", "basis", "q"] as const).every(
        (k) =>
          d.selection[k] ===
          (query.get(k) ??
            {
              period: "all",
              provider: "",
              account: "",
              basis: "billed_cost",
              q: "",
            }[k]),
      ) &&
      Array.isArray(d.providers) &&
      d.providers.length <= 128 &&
      d.providers.every((p) => text(p.provider) && validAmounts(p.amounts)) &&
      consistentProviders(d) &&
      Array.isArray(d.months) &&
      d.months.length <= 120 &&
      d.months.every(
        (m) => /^\d{4}-\d{2}$/.test(m.period) && validAmounts(m.amounts),
      ) &&
      count(d.account_count) &&
      count(d.account_matches) &&
      d.account_matches <= d.account_count &&
      Array.isArray(d.accounts) &&
      d.accounts.length <= 25 &&
      d.accounts.every(
        (a) =>
          sha(a.id) &&
          text(a.tenant) &&
          text(a.provider) &&
          text(a.billing_account) &&
          (a.sub_account === null || text(a.sub_account)) &&
          count(a.rows) &&
          validAmounts(a.amounts, a.rows),
      ) &&
      Array.isArray(d.rows) &&
      d.rows.length <= 25 &&
      d.rows.every(
        (r) =>
          sha(r.id) &&
          sha(r.account) &&
          text(r.provider) &&
          text(r.billing_account) &&
          (r.sub_account === null || text(r.sub_account)) &&
          (r.service === null || text(r.service)) &&
          currency(r.currency) &&
          text(r.charge_month, 7) &&
          text(r.charge_start, 64) &&
          text(r.charge_end, 64) &&
          numeric(r.billed_cost) &&
          numeric(r.effective_cost) &&
          text(r.evidence_class, 64) &&
          text(r.batch, 64) &&
          /^\d{1,5}$/.test(r.ordinal),
      ) &&
      count(d.after) &&
      d.after === Number(query.get("after") ?? 0) &&
      count(d.account_after) &&
      d.account_after === Number(query.get("account_after") ?? 0) &&
      d.rows.length === Math.min(25, d.row_count - d.after) &&
      d.accounts.length === Math.min(25, d.account_matches - d.account_after) &&
      d.next_after === (d.after + 25 < d.row_count ? d.after + 25 : null) &&
      d.next_account_after ===
        (d.account_after + 25 < d.account_matches
          ? d.account_after + 25
          : null) &&
      (!query.has("snapshot") || query.get("snapshot") === d.snapshot)
    );
  } catch {
    return false;
  }
}
export async function loadOverview(
  query: URLSearchParams,
  signal: AbortSignal,
) {
  const data = await request<OverviewData>(
    "?" + query.toString(),
    signal,
    {},
    "/api/focus-overview",
  );
  if (!validOverview(data, query))
    throw new ServiceError("response_invalid", "");
  return data;
}
