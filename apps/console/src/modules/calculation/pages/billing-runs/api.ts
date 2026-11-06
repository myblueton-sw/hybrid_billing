import {
  loadOverview,
  validOverview,
  ServiceError,
  type OverviewData,
  type FocusRow,
  type Amount,
} from "../../../sources/collected-focus/api";
export { ServiceError };
export type {
  OverviewData,
  FocusRow,
  Amount,
  Measure,
  Basis,
  Account,
} from "../../../sources/collected-focus/api";

const sha = (v: unknown): v is string =>
  typeof v === "string" && /^[a-f0-9]{64}$/.test(v);
const text = (v: unknown, max = 4096): v is string =>
  typeof v === "string" && v.length > 0 && v.length <= max;
const nullable = (v: unknown, max = 4096) => v === null || text(v, max);
const unique = (values: readonly unknown[]) =>
  new Set(values).size === values.length;
const month = (v: unknown): v is string =>
  typeof v === "string" && /^(?!0000)\d{4}-(0[1-9]|1[0-2])$/.test(v);
function instant(v: unknown): v is string {
  if (!text(v, 64)) return false;
  const m =
    /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?(Z|[+-](\d{2}):(\d{2}))$/.exec(
      v,
    );
  if (
    !m ||
    m[1].startsWith("0000") ||
    Number(m[2]) > 23 ||
    Number(m[3]) > 59 ||
    Number(m[4]) > 59 ||
    (m[5] !== "Z" && (Number(m[6]) > 23 || Number(m[7]) > 59))
  )
    return false;
  const day = Date.parse(m[1] + "T00:00:00Z");
  return (
    Number.isFinite(day) &&
    new Date(day).toISOString().slice(0, 10) === m[1] &&
    Number.isFinite(Date.parse(v))
  );
}
function timezone(v: unknown) {
  if (!text(v, 128)) return false;
  try {
    new Intl.DateTimeFormat("en", { timeZone: v });
    return true;
  } catch {
    return false;
  }
}
function scopedProvider(provider: string, d: OverviewData) {
  return (
    text(provider, 64) &&
    d.options.providers.includes(provider) &&
    (!d.selection.provider || provider === d.selection.provider)
  );
}
function amountMembership(amounts: Amount[], d: OverviewData) {
  return amounts.every(
    (a) => a.rows > 0 && d.options.currencies.includes(a.currency),
  );
}
function row(r: FocusRow, d: OverviewData) {
  return (
    scopedProvider(r.provider, d) &&
    d.providers.some((p) => p.provider === r.provider) &&
    (!d.selection.account || r.account === d.selection.account) &&
    text(r.billing_account) &&
    nullable(r.sub_account) &&
    nullable(r.service) &&
    d.options.currencies.includes(r.currency) &&
    month(r.charge_month) &&
    d.options.periods.includes(r.charge_month) &&
    (d.selection.period === "all" || r.charge_month === d.selection.period) &&
    [r.billing_start, r.billing_end, r.charge_start, r.charge_end].every(
      instant,
    ) &&
    Date.parse(r.billing_start) < Date.parse(r.billing_end) &&
    Date.parse(r.charge_start) < Date.parse(r.charge_end) &&
    [
      "official-public",
      "official-public-subset",
      "synthetic-contract-fixture",
      "unclassified",
    ].includes(r.evidence_class) &&
    text(r.batch, 64)
  );
}
function rowAmounts(d: OverviewData) {
  return (
    d.amounts.every((a) => {
      const rows = d.rows.filter((r) => r.currency === a.currency);
      if (rows.length > a.rows) return false;
      return (["billed_cost", "effective_cost"] as const).every((b) => {
        const values = rows
          .map((r) => r[b])
          .filter((v): v is string => v !== null);
        if (
          values.length > a[b].present ||
          rows.length - values.length > a[b].missing
        )
          return false;
        return (
          rows.length !== a.rows ||
          (values.length === a[b].present &&
            (values.length
              ? values.reduce((n, v) => n + BigInt(v), 0n).toString()
              : null) === a[b].coefficient)
        );
      });
    }) && d.rows.every((r) => d.amounts.some((a) => a.currency === r.currency))
  );
}
export function query(params: URLSearchParams) {
  const q = new URLSearchParams();
  for (const k of [
    "period",
    "provider",
    "account",
    "basis",
    "q",
    "after",
    "account_after",
    "snapshot",
  ]) {
    const value = params.get("focus-" + k);
    if (value !== null)
      q.set(k, k === "q" ? value.trim().replace(/\s+/g, " ") : value);
  }
  if (!q.has("provider")) q.set("provider", "");
  if (!q.has("period")) q.set("period", "all");
  if (!q.has("basis")) q.set("basis", "billed_cost");
  return q;
}
export function valid(d: OverviewData, q: URLSearchParams): boolean {
  try {
    if (
      (Number(q.get("after") ?? 0) || Number(q.get("account_after") ?? 0)) &&
      !sha(q.get("snapshot"))
    )
      return false;
    if (
      !validOverview(d, q) ||
      !instant(d.observed_at) ||
      (d.coverage.complete_through !== null &&
        !instant(d.coverage.complete_through)) ||
      !nullable(d.company, 128)
    )
      return false;
    if (
      d.authorized_rows > 0 &&
      (!d.tenants.length || !d.versions.length || !d.delivery_count)
    )
      return false;
    if (
      !unique(d.tenants.map((t) => t.key)) ||
      !d.tenants.every((t) => text(t.key, 64) && timezone(t.timezone)) ||
      !unique(d.versions) ||
      !d.versions.every((v) => text(v, 16) && /^\d+\.\d+$/.test(v))
    )
      return false;
    if (
      !unique(d.options.providers) ||
      !d.options.providers.every((p) => text(p, 64)) ||
      !unique(d.options.periods) ||
      !d.options.periods.every(month) ||
      !unique(d.options.currencies) ||
      (d.selection.provider &&
        !d.options.providers.includes(d.selection.provider)) ||
      (d.selection.period !== "all" &&
        !d.options.periods.includes(d.selection.period))
    )
      return false;
    if (
      !amountMembership(d.amounts, d) ||
      !unique(d.providers.map((p) => p.provider)) ||
      !d.providers.every(
        (p) =>
          scopedProvider(p.provider, d) &&
          amountMembership(p.amounts, d) &&
          p.amounts.every((a) =>
            d.amounts.some((full) => full.currency === a.currency),
          ),
      ) ||
      !unique(d.months.map((m) => m.period)) ||
      !d.months.every(
        (m) =>
          month(m.period) &&
          d.options.periods.includes(m.period) &&
          amountMembership(m.amounts, d),
      )
    )
      return false;
    if (
      !unique(d.rows.map((r) => r.id)) ||
      !d.rows.every((r) => row(r, d)) ||
      !rowAmounts(d)
    )
      return false;
    if (
      !unique(d.accounts.map((a) => a.id)) ||
      d.account_count > d.row_count ||
      !d.accounts.every(
        (a) =>
          scopedProvider(a.provider, d) &&
          d.providers.some((p) => p.provider === a.provider) &&
          d.tenants.some((t) => t.key === a.tenant) &&
          (!d.selection.account || a.id === d.selection.account) &&
          text(a.billing_account) &&
          nullable(a.sub_account) &&
          a.native_ancestry === "unverified" &&
          a.rows > 0 &&
          a.rows <= d.row_count &&
          amountMembership(a.amounts, d) &&
          a.amounts.every((v) =>
            d.amounts.some((full) => full.currency === v.currency),
          ),
      )
    )
      return false;
    if (d.accounts.reduce((n, a) => n + a.rows, 0) > d.row_count) return false;
    if (
      d.accounts.length === d.account_count &&
      d.account_matches === d.account_count
    ) {
      const parts = d.accounts.flatMap((a) => a.amounts);
      return d.amounts.every((a) => {
        const currencyParts = parts.filter((p) => p.currency === a.currency);
        return (
          currencyParts.reduce((n, p) => n + p.rows, 0) === a.rows &&
          (["billed_cost", "effective_cost"] as const).every((b) => {
            const present = currencyParts.reduce((n, p) => n + p[b].present, 0);
            return (
              present === a[b].present &&
              currencyParts.reduce((n, p) => n + p[b].missing, 0) ===
                a[b].missing &&
              (present
                ? currencyParts
                    .reduce((n, p) => n + BigInt(p[b].coefficient ?? "0"), 0n)
                    .toString()
                : null) === a[b].coefficient
            );
          })
        );
      });
    }
    return true;
  } catch {
    return false;
  }
}
export async function load(q: URLSearchParams, signal: AbortSignal) {
  const data = await loadOverview(q, signal);
  if (!valid(data, q)) throw new ServiceError("response_invalid", "");
  return data;
}
