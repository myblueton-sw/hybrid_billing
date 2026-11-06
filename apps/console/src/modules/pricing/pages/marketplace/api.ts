import { request, ServiceError } from "../../../sources/pages/live-imports/api";
import {
  profile,
  fieldNames,
  type Data,
  type Charge,
  type Field,
  type Measure,
  type Amount,
} from "./contract";
export * from "./contract";
export { ServiceError };
const hash = (v: unknown): v is string =>
  typeof v === "string" && /^[a-f0-9]{64}$/.test(v);
const count = (v: unknown): v is number =>
  Number.isSafeInteger(v) && Number(v) >= 0 && Number(v) <= 1000000000;
const text = (v: unknown, max = 4096): v is string =>
  typeof v === "string" && v.length > 0 && v.length <= max;
const nullable = (v: unknown, max = 4096) => v === null || text(v, max);
const strings = (v: unknown, max: number, size = 4096): v is string[] =>
  Array.isArray(v) &&
  v.length <= max &&
  new Set(v).size === v.length &&
  v.every((x) => text(x, size));
const coefficient = (v: unknown, bound = 50): v is string =>
  typeof v === "string" &&
  /^-?\d+$/.test(v) &&
  v.length <= bound + 1 &&
  BigInt(v) > -(10n ** BigInt(bound)) &&
  BigInt(v) < 10n ** BigInt(bound);
function iso(v: unknown): v is string {
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
  const d = Date.parse(m[1] + "T00:00:00Z");
  return (
    Number.isFinite(d) &&
    new Date(d).toISOString().slice(0, 10) === m[1] &&
    Number.isFinite(Date.parse(v))
  );
}
function field(f: Field) {
  return (
    !!f &&
    Object.keys(f).length === 2 &&
    [
      "value",
      "source_null",
      "absent",
      "blank",
      "unmapped_version",
      "invalid_type",
    ].includes(f.state) &&
    (f.state === "value"
      ? text(f.value) &&
        !!f.value.trim() &&
        !["NULL", "null", "\\N"].includes(f.value)
      : f.state === "unmapped_version"
        ? nullable(f.value)
        : f.value === null)
  );
}
function measure(m: Measure, n: number) {
  return (
    !!m &&
    count(m.present) &&
    count(m.missing) &&
    m.present + m.missing === n &&
    (m.present === 0 ? m.coefficient === null : coefficient(m.coefficient))
  );
}
function amount(a: Amount) {
  return (
    !!a &&
    /^[A-Z]{3}$/.test(a.currency) &&
    count(a.rows) &&
    a.rows > 0 &&
    measure(a.billed_cost, a.rows) &&
    measure(a.effective_cost, a.rows)
  );
}
function charge(r: Charge, d: Data) {
  return (
    !!r &&
    hash(r.id) &&
    hash(r.account) &&
    text(r.tenant, 64) &&
    text(r.provider, 64) &&
    text(r.billing_account) &&
    nullable(r.sub_account) &&
    /^[A-Z]{3}$/.test(r.currency) &&
    nullable(r.service) &&
    (r.billed_cost === null || coefficient(r.billed_cost, 38)) &&
    (r.effective_cost === null || coefficient(r.effective_cost, 38)) &&
    text(r.batch, 128) &&
    /^\d{1,16}$/.test(r.ordinal) &&
    r.version === "1.0" &&
    d.versions.includes(r.version) &&
    text(r.evidence_class, 128) &&
    d.options.providers.includes(r.provider) &&
    d.options.currencies.includes(r.currency) &&
    (!d.selection.provider || r.provider === d.selection.provider) &&
    (!d.selection.account || r.account === d.selection.account) &&
    !!r.fields &&
    Object.keys(r.fields).length === fieldNames.length &&
    fieldNames.every((k) => field(r.fields[k])) &&
    r.fields.publisher.state === "value" &&
    r.fields.issuer.state === "value" &&
    r.fields.publisher.value !== r.fields.issuer.value &&
    [r.charge_start, r.charge_end, r.billing_start, r.billing_end].every(iso) &&
    Date.parse(r.charge_start) < Date.parse(r.charge_end) &&
    Date.parse(r.billing_start) < Date.parse(r.billing_end)
  );
}
function page(
  items: unknown[],
  after: number,
  total: number,
  next: number | null,
) {
  return (
    count(after) &&
    after <= total &&
    items.length === Math.min(25, total - after) &&
    next === (after + items.length < total ? after + 25 : null)
  );
}
export function query(params: URLSearchParams) {
  const q = new URLSearchParams();
  for (const k of [
    "provider",
    "period",
    "account",
    "q",
    "basis",
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
  if (params.has("marketplace-row"))
    q.set("row", params.get("marketplace-row")!);
  return q;
}
export function valid(d: Data, q: URLSearchParams): boolean {
  try {
    if (
      (q.get("row") ||
        Number(q.get("after") ?? 0) ||
        Number(q.get("account_after") ?? 0)) &&
      !hash(q.get("snapshot"))
    )
      return false;
    if (
      !d ||
      d.schema !== 1 ||
      d.scale !== 18 ||
      d.profile !== profile ||
      d.classification !== "collected-marketplace-evidence-only" ||
      d.financial_authority !== false ||
      !hash(d.snapshot) ||
      !hash(d.source_snapshot) ||
      !hash(d.mapping_digest) ||
      !nullable(d.company, 128) ||
      !strings(d.versions, 32, 32) ||
      !iso(d.observed_at) ||
      d.period_basis !== "charge-start-in-tenant-timezone-no-proration" ||
      !d.selection ||
      !["period", "provider", "account", "q"].every(
        (k) =>
          d.selection[k as keyof Data["selection"]] ===
          (q.get(k) ?? (k === "period" ? "all" : "")),
      ) ||
      (q.has("snapshot") && d.snapshot !== q.get("snapshot"))
    )
      return false;
    if (
      !d.coverage ||
      !count(d.coverage.slices) ||
      !count(d.coverage.finalized_slices) ||
      d.coverage.finalized_slices > d.coverage.slices ||
      (d.coverage.complete_through !== null &&
        !iso(d.coverage.complete_through)) ||
      d.coverage.expectations !== "operator-asserted-unverified"
    )
      return false;
    if (
      !d.options ||
      !strings(d.options.providers, 128, 64) ||
      !strings(d.options.periods, 120, 7) ||
      !d.options.periods.every((x) => /^\d{4}-(0[1-9]|1[0-2])$/.test(x)) ||
      !strings(d.options.currencies, 64, 3) ||
      !d.options.currencies.every((x) => /^[A-Z]{3}$/.test(x)) ||
      (d.selection.provider &&
        !d.options.providers.includes(d.selection.provider)) ||
      (d.selection.period !== "all" &&
        !d.options.periods.includes(d.selection.period))
    )
      return false;
    if (
      !count(d.authorized_rows) ||
      !count(d.row_count) ||
      d.row_count > d.authorized_rows ||
      !d.counts ||
      ![d.counts.candidate, d.counts.no_difference, d.counts.unknown].every(
        count,
      ) ||
      Object.keys(d.counts).length !== 3 ||
      d.counts.candidate + d.counts.no_difference + d.counts.unknown !==
        d.row_count
    )
      return false;
    for (const [amounts, n] of [
      [d.candidate_amounts, d.counts.candidate],
      [d.unknown_amounts, d.counts.unknown],
    ] as const) {
      if (
        !Array.isArray(amounts) ||
        amounts.length > 64 ||
        !amounts.every(
          (a) => amount(a) && d.options.currencies.includes(a.currency),
        ) ||
        new Set(amounts.map((a) => a.currency)).size !== amounts.length ||
        amounts.reduce((n, a) => n + a.rows, 0) !== n
      )
        return false;
    }
    if (
      !Array.isArray(d.rows) ||
      !page(d.rows, d.after, d.counts.candidate, d.next_after) ||
      d.after !== Number(q.get("after") ?? 0) ||
      !d.rows.every((r) => charge(r, d)) ||
      new Set(d.rows.map((r) => r.id)).size !== d.rows.length
    )
      return false;
    for (const a of d.candidate_amounts) {
      const rows = d.rows.filter((r) => r.currency === a.currency);
      if (rows.length > a.rows) return false;
      for (const b of ["billed_cost", "effective_cost"] as const) {
        const vals = rows
          .map((r) => r[b])
          .filter((x): x is string => x !== null);
        if (
          vals.length > a[b].present ||
          rows.length - vals.length > a[b].missing
        )
          return false;
        if (
          rows.length === a.rows &&
          a[b].coefficient !==
            (vals.length
              ? vals.reduce((n, v) => n + BigInt(v), 0n).toString()
              : null)
        )
          return false;
      }
    }
    if (
      d.rows.some(
        (r) => !d.candidate_amounts.some((a) => a.currency === r.currency),
      )
    )
      return false;
    if (
      !Array.isArray(d.accounts) ||
      !count(d.account_count) ||
      !count(d.account_matches) ||
      d.account_matches > d.account_count ||
      d.account_count > d.row_count ||
      !page(
        d.accounts,
        d.account_after,
        d.account_matches,
        d.next_account_after,
      ) ||
      d.account_after !== Number(q.get("account_after") ?? 0) ||
      new Set(d.accounts.map((a) => a.id)).size !== d.accounts.length ||
      !d.accounts.every(
        (a) =>
          hash(a.id) &&
          text(a.tenant, 64) &&
          text(a.provider, 64) &&
          d.options.providers.includes(a.provider) &&
          (!d.selection.provider || a.provider === d.selection.provider) &&
          (!d.selection.account || a.id === d.selection.account) &&
          text(a.billing_account) &&
          nullable(a.sub_account) &&
          count(a.rows) &&
          a.rows > 0 &&
          a.rows <= d.row_count,
      )
    )
      return false;
    if (d.selected === null) return !q.get("row");
    const row = d.rows.find((r) => r.id === q.get("row"));
    return !!row && JSON.stringify(row) === JSON.stringify(d.selected);
  } catch {
    return false;
  }
}
export async function load(q: URLSearchParams, signal: AbortSignal) {
  const d = await request<Data>("?" + q, signal, {}, "/api/focus-marketplace");
  if (!valid(d, q)) throw new ServiceError("response_invalid", "");
  return d;
}
