import { hash, iso, keys, type Data } from "./contract";
import { body, ReadError } from "./api";
export const provenanceKeys = [
  "official-public",
  "official-public-subset",
  "synthetic-contract-fixture",
  "unclassified",
] as const;
export type Provenance = Record<(typeof provenanceKeys)[number], number>;
export type Scope = Pick<
  Data,
  "auth_scope" | "permission_revision" | "audience" | "member_count"
>;
export type CollectionStatus = Scope & {
  schema: 1;
  profile: "retained-focus-collection-status-v1";
  read_only: true;
  financial_authority: false;
  snapshot: string;
  observed_at: string;
  row_count: number;
  delivery_count: number;
  versions: string[];
  provenance: Provenance;
  providers: { provider: string; row_count: number }[];
  periods: { window_start: string; window_end: string; row_count: number }[];
  upstream_completeness: "unverified";
  source_classification: "exploratory-query-only";
  original_availability: "not-checked";
};
const fields = [
  "schema",
  "profile",
  "read_only",
  "financial_authority",
  "audience",
  "member_count",
  "auth_scope",
  "permission_revision",
  "snapshot",
  "observed_at",
  "row_count",
  "delivery_count",
  "versions",
  "provenance",
  "providers",
  "periods",
  "upstream_completeness",
  "source_classification",
  "original_availability",
];
const count = (v: unknown, minimum = 0): v is number =>
  typeof v === "number" &&
  Number.isSafeInteger(v) &&
  v >= minimum &&
  v <= 10000;
const fail = (): never => {
  throw new ReadError("invalid_response");
};
function instant(value: string) {
  return (
    value.slice(0, 19) +
    "." +
    (value.slice(19).match(/^\.(\d+)/)?.[1] ?? "").padEnd(9, "0")
  );
}
function compareCodepoints(left: string, right: string) {
  const a = Array.from(left, (c) => c.codePointAt(0)!),
    b = Array.from(right, (c) => c.codePointAt(0)!);
  for (let i = 0; i < Math.min(a.length, b.length); i++)
    if (a[i] !== b[i]) return a[i] - b[i];
  return a.length - b.length;
}
const ascending = (values: string[]) =>
  values.every(
    (value, i) => i === 0 || compareCodepoints(values[i - 1], value) < 0,
  );
export function parseStatus(raw: unknown, scope: Scope): CollectionStatus {
  if (!keys(raw, fields)) return fail();
  const d = raw;
  if (
    d.schema !== 1 ||
    d.profile !== "retained-focus-collection-status-v1" ||
    d.read_only !== true ||
    d.financial_authority !== false ||
    d.upstream_completeness !== "unverified" ||
    d.source_classification !== "exploratory-query-only" ||
    d.original_availability !== "not-checked"
  )
    return fail();
  if (
    (d.audience !== "company" && d.audience !== "tenant") ||
    !count(d.member_count, 1) ||
    !hash(d.auth_scope) ||
    !hash(d.permission_revision) ||
    !hash(d.snapshot) ||
    !iso(d.observed_at)
  )
    return fail();
  for (const key of [
    "audience",
    "member_count",
    "auth_scope",
    "permission_revision",
  ] as const)
    if (d[key] !== scope[key])
      throw new ReadError("permission_changed", "", 403);
  if (
    !count(d.row_count) ||
    !count(d.delivery_count) ||
    d.delivery_count > d.row_count
  )
    return fail();
  if (
    !Array.isArray(d.versions) ||
    d.versions.length > 128 ||
    d.versions.some(
      (v) => typeof v !== "string" || !/^\d{1,2}\.\d{1,2}$/.test(v),
    ) ||
    new Set(d.versions).size !== d.versions.length ||
    d.versions.length > d.delivery_count
  )
    return fail();
  const provenance = d.provenance;
  if (
    !keys(provenance, provenanceKeys) ||
    !provenanceKeys.every((k) => count(provenance[k])) ||
    provenanceKeys.reduce((n, k) => n + Number(provenance[k]), 0) !==
      d.row_count
  )
    return fail();
  if (
    !Array.isArray(d.providers) ||
    d.providers.length > 128 ||
    !d.providers.every(
      (p) =>
        keys(p, ["provider", "row_count"]) &&
        typeof p.provider === "string" &&
        p.provider.length > 0 &&
        p.provider.length <= 256 &&
        count(p.row_count, 1),
    )
  )
    return fail();
  if (
    new Set(d.providers.map((p) => p.provider)).size !== d.providers.length ||
    d.providers.reduce((n, p) => n + p.row_count, 0) !== d.row_count
  )
    return fail();
  if (
    !Array.isArray(d.periods) ||
    d.periods.length > 120 ||
    d.periods.length > d.delivery_count ||
    !d.periods.every(
      (p) =>
        keys(p, ["window_start", "window_end", "row_count"]) &&
        iso(p.window_start) &&
        iso(p.window_end) &&
        instant(p.window_start) < instant(p.window_end) &&
        count(p.row_count, 1),
    )
  )
    return fail();
  if (
    new Set(
      d.periods.map((p) => JSON.stringify([p.window_start, p.window_end])),
    ).size !== d.periods.length ||
    d.periods.reduce((n, p) => n + p.row_count, 0) !== d.row_count
  )
    return fail();
  if (
    d.row_count === 0
      ? d.delivery_count !== 0 ||
        d.versions.length !== 0 ||
        d.providers.length !== 0 ||
        d.periods.length !== 0
      : d.delivery_count === 0 || d.versions.length === 0
  )
    return fail();
  const periodKeys = d.periods.map(
    (p) => instant(p.window_start) + "|" + instant(p.window_end),
  );
  if (
    !ascending(d.versions) ||
    !ascending(d.providers.map((p) => p.provider)) ||
    !ascending(periodKeys)
  )
    return fail();
  return d as unknown as CollectionStatus;
}
export async function loadStatus(
  scope: Scope,
  signal: AbortSignal,
): Promise<CollectionStatus> {
  const captured = Object.freeze({ ...scope });
  const aborted = () => {
    if (signal.aborted) throw new DOMException("Request aborted", "AbortError");
  };
  aborted();
  let response: Response;
  try {
    response = await fetch("/api/focus-collection-status", {
      method: "GET",
      headers: { "X-HB-Local-Client": "1", Accept: "application/json" },
      cache: "no-store",
      redirect: "error",
      signal,
    });
  } catch (e) {
    aborted();
    throw new ReadError("service_unavailable");
  }
  if (signal.aborted || response.redirected) {
    await response.body?.cancel().catch(() => {});
    aborted();
    throw new ReadError("invalid_response", "", response.status);
  }
  let result: unknown;
  try {
    result = await body(response);
  } catch (e) {
    aborted();
    throw new ReadError(
      e instanceof ReadError ? e.code : "invalid_response",
      "",
      response.status,
    );
  }
  aborted();
  if (!response.ok) {
    if (
      !keys(result, ["error", "request_id"]) ||
      typeof result.error !== "string" ||
      !/^[a-z][a-z0-9_]{0,79}$/.test(result.error) ||
      typeof result.request_id !== "string" ||
      !/^[a-f0-9]{32}$/.test(result.request_id)
    )
      throw new ReadError("invalid_response", "", response.status);
    throw new ReadError(result.error, result.request_id, response.status);
  }
  return parseStatus(result, captured);
}
