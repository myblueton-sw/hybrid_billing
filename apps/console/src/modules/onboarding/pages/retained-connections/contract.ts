export const views = [
  "connections",
  "capabilities",
  "security",
  "diagnostics",
] as const;
export type View = (typeof views)[number];
export type Query = {
  view: View;
  q: string;
  after: number;
  snapshot?: string;
  permission_revision?: string;
};
export const recordKeys = [
  "id",
  "connection",
  "focus_version",
  "grain",
  "window_start",
  "window_end",
  "registered_domains",
  "semantic_profile",
  "allowed_lateness_seconds",
  "replacement_limit",
] as const;
export type Registration = {
  id: string;
  connection: string;
  focus_version: string;
  grain: "day" | "month";
  window_start: string;
  window_end: string;
  registered_domains: string;
  semantic_profile: "focus-consumed-v1" | "exploratory-query-only";
  allowed_lateness_seconds: string;
  replacement_limit: string;
};
export type Data = {
  schema: 1;
  profile: "retained-focus-connections-v1";
  read_only: true;
  financial_authority: false;
  commands: "unconnected";
  view: View;
  selection: { q: string; after: number };
  audience: "tenant" | "company";
  member_count: number;
  auth_scope: string;
  permission_revision: string;
  snapshot: string;
  observed_at: string;
  total: number | null;
  next_after: number | null;
  capabilities: {
    connections: "fully-owned-registrations-only";
    capability_evidence: "registration-only";
    provider_authentication: "unconnected";
    diagnostics: "unconnected";
  };
  records: Registration[];
};
export const keys = (
  value: unknown,
  expected: readonly string[],
): value is Record<string, unknown> =>
  !!value &&
  typeof value === "object" &&
  !Array.isArray(value) &&
  Object.keys(value).length === expected.length &&
  expected.every((k) => Object.hasOwn(value, k));
export const hash = (value: unknown): value is string =>
  typeof value === "string" && /^[a-f0-9]{64}$/.test(value);
export const identifier = (value: unknown): value is string =>
  typeof value === "string" && /^[a-z0-9:_-]{0,128}$/.test(value);
const integer = (value: unknown, max: number): value is number =>
  Number.isSafeInteger(value) && Number(value) >= 0 && Number(value) <= max;
const decimal = (value: unknown, min: bigint, max: bigint): value is string =>
  typeof value === "string" &&
  /^(0|[1-9][0-9]{0,18})$/.test(value) &&
  BigInt(value) >= min &&
  BigInt(value) <= max;
export function iso(value: unknown): value is string {
  if (typeof value !== "string") return false;
  const match =
    /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,6})?(?:Z|\+00:00)$/.exec(
      value,
    );
  return (
    !!match &&
    !match[1].startsWith("0000") &&
    Number(match[2]) < 24 &&
    Number(match[3]) < 60 &&
    Number(match[4]) < 60 &&
    Number.isFinite(Date.parse(value)) &&
    new Date(Date.parse(match[1] + "T00:00:00Z")).toISOString().slice(0, 10) ===
      match[1]
  );
}
function registration(value: unknown): value is Registration {
  if (!keys(value, recordKeys)) return false;
  const r = value;
  if (
    !hash(r.id) ||
    typeof r.connection !== "string" ||
    !/^[a-z][a-z0-9_-]{0,63}$/.test(r.connection) ||
    typeof r.focus_version !== "string" ||
    r.focus_version.length > 8 ||
    !/^\d{1,2}\.\d{1,2}$/.test(r.focus_version) ||
    (r.grain !== "day" && r.grain !== "month") ||
    (r.semantic_profile !== "focus-consumed-v1" &&
      r.semantic_profile !== "exploratory-query-only") ||
    !decimal(r.registered_domains, 1n, 10000n) ||
    !decimal(r.allowed_lateness_seconds, 0n, 604800n) ||
    !decimal(r.replacement_limit, 1n, 10n) ||
    !iso(r.window_start) ||
    !iso(r.window_end)
  )
    return false;
  if (
    ![r.window_start, r.window_end].every((v) =>
      /T00:00:00(?:\.0{1,6})?(?:Z|\+00:00)$/.test(v),
    )
  )
    return false;
  const start = new Date(r.window_start),
    end = new Date(r.window_end);
  if (start >= end) return false;
  const count =
    r.grain === "day"
      ? (end.getTime() - start.getTime()) / 86400000
      : (end.getUTCFullYear() - start.getUTCFullYear()) * 12 +
        end.getUTCMonth() -
        start.getUTCMonth();
  return (
    Number.isInteger(count) &&
    count > 0 &&
    count <= 3660 &&
    (r.grain !== "month" ||
      (start.getUTCDate() === 1 && end.getUTCDate() === 1))
  );
}
const envelopeKeys = [
  "schema",
  "profile",
  "read_only",
  "financial_authority",
  "commands",
  "view",
  "selection",
  "audience",
  "member_count",
  "auth_scope",
  "permission_revision",
  "snapshot",
  "observed_at",
  "total",
  "next_after",
  "capabilities",
  "records",
] as const;
export function parseData(raw: unknown, q: Query): Data {
  const fail = (): never => {
    throw new Error("invalid_response");
  };
  if (!keys(raw, envelopeKeys)) return fail();
  const d = raw;
  if (
    d.schema !== 1 ||
    d.profile !== "retained-focus-connections-v1" ||
    d.read_only !== true ||
    d.financial_authority !== false ||
    d.commands !== "unconnected" ||
    d.view !== q.view ||
    !keys(d.selection, ["q", "after"]) ||
    d.selection.q !== q.q ||
    d.selection.after !== q.after ||
    (d.audience !== "tenant" && d.audience !== "company") ||
    !integer(d.member_count, 10000) ||
    d.member_count === 0 ||
    !hash(d.auth_scope) ||
    !hash(d.permission_revision) ||
    !hash(d.snapshot) ||
    !iso(d.observed_at)
  )
    return fail();
  if (
    (q.snapshot !== undefined && q.snapshot !== d.snapshot) ||
    (q.permission_revision !== undefined &&
      q.permission_revision !== d.permission_revision)
  )
    return fail();
  if (
    !keys(d.capabilities, [
      "connections",
      "capability_evidence",
      "provider_authentication",
      "diagnostics",
    ]) ||
    d.capabilities.connections !== "fully-owned-registrations-only" ||
    d.capabilities.capability_evidence !== "registration-only" ||
    d.capabilities.provider_authentication !== "unconnected" ||
    d.capabilities.diagnostics !== "unconnected"
  )
    return fail();
  if (
    !Array.isArray(d.records) ||
    d.records.length > 25 ||
    !d.records.every(registration) ||
    new Set(d.records.map((r) => r.id)).size !== d.records.length ||
    new Set(d.records.map((r) => r.connection)).size !== d.records.length ||
    d.records.some(
      (r, i, all) => i > 0 && all[i - 1].connection >= r.connection,
    ) ||
    d.records.some(
      (r) => !r.id.startsWith(q.q) && !r.connection.startsWith(q.q),
    )
  )
    return fail();
  if (q.view === "security" || q.view === "diagnostics") {
    if (
      q.after !== 0 ||
      d.total !== null ||
      d.next_after !== null ||
      d.records.length !== 0
    )
      return fail();
  } else {
    if (
      !integer(d.total, 10000) ||
      q.after > d.total ||
      d.records.length !== Math.min(25, d.total - q.after)
    )
      return fail();
    const next =
      q.after + d.records.length < d.total ? q.after + d.records.length : null;
    if (d.next_after !== next || (next !== null && d.records.length !== 25))
      return fail();
  }
  return d as unknown as Data;
}
export function query(params: URLSearchParams, view: string): Query {
  const allowed = [
    "connection-q",
    "connection-after",
    "connection-snapshot",
    "connection-permission",
  ];
  if (
    !(views as readonly string[]).includes(view) ||
    params.getAll("view").length > 1 ||
    (params.has("view") && params.get("view") !== view)
  )
    throw new Error("invalid_query");
  for (const [key] of params)
    if (
      key.startsWith("connection-") &&
      (!allowed.includes(key) || params.getAll(key).length !== 1)
    )
      throw new Error("invalid_query");
  const q = params.get("connection-q") ?? "",
    after = params.get("connection-after") ?? "0";
  if (
    !identifier(q) ||
    !/^(0|[1-9][0-9]{0,4})$/.test(after) ||
    Number(after) > 10000
  )
    throw new Error("invalid_query");
  const snapshot = params.get("connection-snapshot") ?? undefined;
  const permission_revision = params.get("connection-permission") ?? undefined;
  if (
    (snapshot !== undefined && !hash(snapshot)) ||
    (permission_revision !== undefined && !hash(permission_revision)) ||
    (snapshot === undefined) !== (permission_revision === undefined) ||
    (after !== "0" && (!snapshot || !permission_revision)) ||
    ((view === "security" || view === "diagnostics") && after !== "0")
  )
    throw new Error("invalid_query");
  return {
    view: view as View,
    q,
    after: Number(after),
    ...(snapshot ? { snapshot, permission_revision } : {}),
  };
}
export function queryString(q: Query): string {
  const params = new URLSearchParams({
    view: q.view,
    q: q.q,
    after: String(q.after),
  });
  if (q.snapshot) params.set("snapshot", q.snapshot);
  if (q.permission_revision)
    params.set("permission_revision", q.permission_revision);
  return params.toString();
}
