import { request, ServiceError } from "../../../sources/pages/live-imports/api";
import {
  profile,
  fieldNames,
  type Data,
  type Group,
  type Costs,
  type Measure,
  type Metadata,
  type Field,
  type Charge,
  type Amount,
} from "./contract";
export { ServiceError };
export * from "./contract";
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
  v.every((s) => text(s, size));
const coefficient = (v: unknown, bound = 50): v is string =>
  typeof v === "string" &&
  new RegExp(`^-?\\d{1,${bound}}$`).test(v) &&
  BigInt(v) > -(10n ** BigInt(bound)) &&
  BigInt(v) < 10n ** BigInt(bound);
const iso = (v: unknown): v is string => {
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
};
const keys = (v: unknown, names: readonly string[]) =>
  !!v &&
  typeof v === "object" &&
  !Array.isArray(v) &&
  Object.keys(v).length === names.length &&
  names.every((k) => Object.hasOwn(v, k));
function measure(m: Measure, n: number) {
  return (
    !!m &&
    count(m.present) &&
    count(m.missing) &&
    m.present + m.missing === n &&
    (m.present === 0 ? m.coefficient === null : coefficient(m.coefficient))
  );
}
function costs(c: Costs) {
  return (
    !!c &&
    count(c.rows) &&
    measure(c.billed_cost, c.rows) &&
    measure(c.effective_cost, c.rows)
  );
}
function amount(a: Amount) {
  return costs(a) && /^[A-Z]{3}$/.test(a.currency);
}
function metadata(m: Metadata) {
  return (
    !!m &&
    strings(m.values, 16) &&
    count(m.distinct) &&
    m.distinct >= m.values.length &&
    m.truncated === m.distinct > m.values.length
  );
}
const states = [
  "value",
  "source_null",
  "absent",
  "blank",
  "unmapped_version",
  "invalid_type",
];
function field(f: Field) {
  return (
    !!f &&
    keys(f, ["state", "value"]) &&
    states.includes(f.state) &&
    (f.state === "value"
      ? text(f.value) && !!f.value.trim()
      : f.state === "unmapped_version"
        ? nullable(f.value)
        : f.value === null)
  );
}
function group(g: Group) {
  return (
    costs(g) &&
    g.rows > 0 &&
    hash(g.id) &&
    hash(g.account) &&
    text(g.tenant, 64) &&
    text(g.provider, 64) &&
    text(g.billing_account) &&
    nullable(g.sub_account) &&
    /^[A-Z]{3}$/.test(g.currency) &&
    text(g.source_id) &&
    !!g.source_id.trim() &&
    !["NULL", "null", "\\N"].includes(g.source_id) &&
    strings(g.versions, 32, 32) &&
    g.versions.length > 0 &&
    keys(
      g.metadata,
      fieldNames.filter((k) => k !== "id"),
    ) &&
    Object.values(g.metadata).every(metadata) &&
    keys(g.statuses, ["used", "unused", "unknown"]) &&
    Object.values(g.statuses).every(costs) &&
    Object.values(g.statuses).reduce((n, b) => n + b.rows, 0) === g.rows &&
    ["billed_cost", "effective_cost"].every((k) => {
      const basis = k as "billed_cost" | "effective_cost";
      const buckets = Object.values(g.statuses).map((b) => b[basis]);
      return (
        buckets.reduce((n, m) => n + m.present, 0) === g[basis].present &&
        buckets.reduce((n, m) => n + m.missing, 0) === g[basis].missing &&
        (g[basis].present === 0 ||
          buckets.reduce((n, m) => n + BigInt(m.coefficient ?? "0"), 0n) ===
            BigInt(g[basis].coefficient!))
      );
    }) &&
    !!g.mapping &&
    count(g.mapping.version_mapped_rows) &&
    count(g.mapping.version_unmapped_rows) &&
    g.mapping.version_mapped_rows + g.mapping.version_unmapped_rows ===
      g.rows &&
    !!g.intervals &&
    Object.values(g.intervals).length === 4 &&
    Object.values(g.intervals).every(iso) &&
    Date.parse(g.intervals.charge_start) < Date.parse(g.intervals.charge_end) &&
    Date.parse(g.intervals.billing_start) < Date.parse(g.intervals.billing_end)
  );
}
function charge(r: Charge, g: Group) {
  return (
    !!r &&
    hash(r.id) &&
    r.group === g.id &&
    r.tenant === g.tenant &&
    r.account === g.account &&
    r.currency === g.currency &&
    r.provider === g.provider &&
    r.billing_account === g.billing_account &&
    r.sub_account === g.sub_account &&
    nullable(r.service) &&
    nullable(r.billed_cost, 39) &&
    nullable(r.effective_cost, 39) &&
    (r.billed_cost === null || coefficient(r.billed_cost, 38)) &&
    (r.effective_cost === null || coefficient(r.effective_cost, 38)) &&
    [r.charge_start, r.charge_end, r.billing_start, r.billing_end].every(iso) &&
    Date.parse(r.charge_start) < Date.parse(r.charge_end) &&
    Date.parse(r.billing_start) < Date.parse(r.billing_end) &&
    text(r.version, 32) &&
    text(r.batch, 128) &&
    /^[1-9]\d{0,18}$/.test(r.ordinal) &&
    keys(r.fields, fieldNames) &&
    Object.values(r.fields).every(field) &&
    r.fields.id.state === "value" &&
    r.fields.id.value === g.source_id &&
    [
      "official-public",
      "official-public-subset",
      "synthetic-contract-fixture",
      "unclassified",
    ].includes(r.evidence_class)
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
    next === (after + items.length < total ? after + items.length : null)
  );
}
export function query(params: URLSearchParams) {
  const q = new URLSearchParams();
  for (const k of [
    "provider",
    "period",
    "account",
    "q",
    "after",
    "account_after",
    "snapshot",
  ])
    if (params.has("focus-" + k))
      q.set(
        k,
        k === "q"
          ? params.get("focus-" + k)!.trim()
          : params.get("focus-" + k)!,
      );
  if (!q.has("provider")) q.set("provider", "AWS");
  if (!q.has("period")) q.set("period", "all");
  if (params.has("commitment-group"))
    q.set("group", params.get("commitment-group")!);
  if (params.has("commitment-row-after"))
    q.set("row_after", params.get("commitment-row-after")!);
  return q;
}
export function valid(d: Data, q: URLSearchParams): boolean {
  try {
    if (
      (q.get("group") ||
        Number(q.get("after") ?? 0) > 0 ||
        Number(q.get("account_after") ?? 0) > 0 ||
        Number(q.get("row_after") ?? 0) > 0) &&
      !hash(q.get("snapshot"))
    )
      return false;
    if (
      !d ||
      d.schema !== 1 ||
      d.scale !== 18 ||
      d.profile !== profile ||
      d.classification !== "collected-commitment-evidence-only" ||
      d.financial_authority !== false ||
      ![d.snapshot, d.source_snapshot, d.mapping_digest].every(hash) ||
      !nullable(d.company, 64) ||
      !strings(d.versions, 32, 32) ||
      !iso(d.observed_at) ||
      d.period_basis !== "charge-start-in-tenant-timezone-no-proration"
    )
      return false;
    if (
      !d.selection ||
      !["period", "provider", "account", "q"].every(
        (k) =>
          d.selection[k as keyof Data["selection"]] ===
          (q.get(k) ?? (k === "period" ? "all" : "")),
      ) ||
      (q.has("snapshot") && q.get("snapshot") !== d.snapshot)
    )
      return false;
    if (
      !d.coverage ||
      !count(d.coverage.slices) ||
      !count(d.coverage.finalized_slices) ||
      d.coverage.finalized_slices > d.coverage.slices ||
      !(
        d.coverage.complete_through === null || iso(d.coverage.complete_through)
      ) ||
      !text(d.coverage.expectations, 256) ||
      !d.options ||
      !strings(d.options.providers, 128, 64) ||
      !strings(d.options.periods, 120, 7) ||
      !d.options.periods.every((p) => /^\d{4}-\d{2}$/.test(p)) ||
      !strings(d.options.currencies, 64, 3) ||
      !d.options.currencies.every((c) => /^[A-Z]{3}$/.test(c))
    )
      return false;
    if (
      ![
        d.authorized_rows,
        d.row_count,
        d.identified_rows,
        d.unidentified_rows,
        d.group_count,
        d.account_count,
        d.account_matches,
      ].every(count) ||
      d.row_count > d.authorized_rows ||
      d.identified_rows + d.unidentified_rows !== d.row_count ||
      d.group_count > d.identified_rows ||
      d.account_matches > d.account_count
    )
      return false;
    if (
      !Array.isArray(d.identified_amounts) ||
      !Array.isArray(d.unidentified_amounts)
    )
      return false;
    for (const [a, n] of [
      [d.identified_amounts, d.identified_rows],
      [d.unidentified_amounts, d.unidentified_rows],
    ] as [Amount[], number][]) {
      if (
        a.length > 64 ||
        new Set(a.map((v) => v.currency)).size !== a.length ||
        !a.every(amount) ||
        a.reduce((n, v) => n + v.rows, 0) !== n
      )
        return false;
    }
    if (
      !keys(
        d.unidentified_reasons,
        states.filter((k) => k !== "value"),
      ) ||
      !Object.values(d.unidentified_reasons).every(count) ||
      Object.values(d.unidentified_reasons).reduce((n, v) => n + v, 0) !==
        d.unidentified_rows
    )
      return false;
    if (
      !Array.isArray(d.groups) ||
      !d.groups.every(group) ||
      new Set(d.groups.map((g) => g.id)).size !== d.groups.length ||
      !page(d.groups, d.after, d.group_count, d.next_after) ||
      d.after !== Number(q.get("after") ?? 0)
    )
      return false;
    if (
      new Set(
        d.groups.map((g) =>
          JSON.stringify([
            g.tenant,
            g.provider,
            g.account,
            g.currency,
            g.source_id,
          ]),
        ),
      ).size !== d.groups.length
    )
      return false;
    for (const g of d.groups) {
      if (
        (d.selection.provider && g.provider !== d.selection.provider) ||
        (d.selection.account && g.account !== d.selection.account) ||
        !g.versions.every((v) => d.versions.includes(v))
      )
        return false;
    }
    for (const currency of new Set(d.groups.map((g) => g.currency))) {
      const total = d.identified_amounts.find((a) => a.currency === currency),
        groups = d.groups.filter((g) => g.currency === currency);
      if (!total || groups.reduce((n, g) => n + g.rows, 0) > total.rows)
        return false;
      for (const basis of ["billed_cost", "effective_cost"] as const) {
        if (
          groups.reduce((n, g) => n + g[basis].present, 0) >
            total[basis].present ||
          groups.reduce((n, g) => n + g[basis].missing, 0) >
            total[basis].missing
        )
          return false;
      }
    }
    if (d.after === 0 && d.next_after === null) {
      if (d.groups.reduce((n, g) => n + g.rows, 0) !== d.identified_rows)
        return false;
      for (const total of d.identified_amounts) {
        const groups = d.groups.filter((g) => g.currency === total.currency);
        if (groups.reduce((n, g) => n + g.rows, 0) !== total.rows) return false;
        for (const basis of ["billed_cost", "effective_cost"] as const)
          if (
            groups.reduce((n, g) => n + g[basis].present, 0) !==
              total[basis].present ||
            groups.reduce((n, g) => n + g[basis].missing, 0) !==
              total[basis].missing ||
            (total[basis].present > 0 &&
              groups.reduce(
                (n, g) => n + BigInt(g[basis].coefficient ?? "0"),
                0n,
              ) !== BigInt(total[basis].coefficient!))
          )
            return false;
      }
    }
    if (
      !Array.isArray(d.accounts) ||
      d.accounts.length > 25 ||
      new Set(d.accounts.map((a) => a.id)).size !== d.accounts.length ||
      !d.accounts.every(
        (a) =>
          hash(a.id) &&
          text(a.tenant, 64) &&
          text(a.provider, 64) &&
          text(a.billing_account) &&
          nullable(a.sub_account) &&
          count(a.rows),
      ) ||
      !page(
        d.accounts,
        d.account_after,
        d.account_matches,
        d.next_account_after,
      ) ||
      d.account_after !== Number(q.get("account_after") ?? 0)
    )
      return false;
    if (d.selected === null)
      return !q.get("group") && Number(q.get("row_after") ?? 0) === 0;
    const s = d.selected,
      g = d.groups.find((g) => g.id === s.group);
    return (
      !!g &&
      s.group === q.get("group") &&
      s.total_rows === g.rows &&
      Array.isArray(s.rows) &&
      new Set(s.rows.map((r) => r.id)).size === s.rows.length &&
      s.rows.every((r) => charge(r, g) && g.versions.includes(r.version)) &&
      page(s.rows, s.after, s.total_rows, s.next_after) &&
      s.after === Number(q.get("row_after") ?? 0)
    );
  } catch {
    return false;
  }
}
export async function load(q: URLSearchParams, signal: AbortSignal) {
  const data = await request<Data>(
    "?" + q.toString(),
    signal,
    {},
    "/api/focus-commitments",
  );
  if (!valid(data, q)) throw new ServiceError("response_invalid", "");
  return data;
}
