export const views = [
  "imports",
  "quarantine",
  "completeness",
  "extraction",
] as const;
export type View = (typeof views)[number];
export const recordKeys = [
  "id",
  "kind",
  "connection",
  "stage",
  "window_start",
  "window_end",
  "source_revision",
  "focus_version",
  "format",
  "declared_rows",
  "file_bytes",
  "file_digest",
  "mapping_digest",
  "received_at",
  "complete_through",
  "finalized",
  "reason",
  "root_receipt",
  "replaced_by",
  "attempts",
  "known_zero",
  "published",
] as const;
export const stages = {
  delivery: ["registered", "verified", "sealed"],
  quarantine: [
    "quarantined",
    "replacement-requested",
    "escalated",
    "resolved",
    "deleted",
  ],
  coverage: [
    "registered",
    "verified",
    "sealed",
    "quarantined",
    "replacement-requested",
    "escalated",
    "missing",
    "late",
  ],
} as const;
export type RecordKind = keyof typeof stages;
export type Receipt = {
  id: string;
  kind: RecordKind;
  connection: string | null;
  stage: string;
  window_start: string | null;
  window_end: string | null;
  source_revision: string | null;
  focus_version: string | null;
  format: "csv" | "parquet" | null;
  declared_rows: string | null;
  file_bytes: string | null;
  file_digest: string | null;
  mapping_digest: string | null;
  received_at: string | null;
  complete_through: string | null;
  finalized: boolean | null;
  reason: string | null;
  root_receipt: string | null;
  replaced_by: string | null;
  attempts: string | null;
  known_zero: boolean | null;
  published: boolean | null;
};
export type Data = {
  schema: 1;
  profile: "retained-focus-receipts-v1";
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
    receipts: "fully-authorized-metadata-only";
    completeness: "registered-expectations-only" | "unavailable";
    extraction: "unconnected";
  };
  records: Receipt[];
};
export type Query = {
  view: View;
  q: string;
  after: number;
  snapshot?: string;
  permission_revision?: string;
};
export const hash = (v: unknown): v is string =>
  typeof v === "string" && /^[a-f0-9]{64}$/.test(v);
export const identifier = (v: unknown): v is string =>
  typeof v === "string" && /^[a-z0-9:_-]{0,128}$/.test(v);
export const keys = (
  v: unknown,
  expected: readonly string[],
): v is Record<string, unknown> =>
  !!v &&
  typeof v === "object" &&
  !Array.isArray(v) &&
  Object.keys(v).length === expected.length &&
  expected.every((k) => Object.hasOwn(v, k));
const bounded = (v: unknown, max: number): v is number =>
  Number.isSafeInteger(v) && Number(v) >= 0 && Number(v) <= max;
const decimal = (v: unknown, max: bigint, min = 0n): v is string =>
  typeof v === "string" &&
  /^(0|[1-9][0-9]{0,18})$/.test(v) &&
  BigInt(v) >= min &&
  BigInt(v) <= max;
export function iso(v: unknown): v is string {
  if (typeof v !== "string") return false;
  const m =
    /^(\d{4}-\d{2}-\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?(Z|\+00:00)$/.exec(
      v,
    );
  return (
    !!m &&
    !m[1].startsWith("0000") &&
    Number(m[2]) < 24 &&
    Number(m[3]) < 60 &&
    Number(m[4]) < 60 &&
    Number.isFinite(Date.parse(v)) &&
    new Date(Date.parse(m[1] + "T00:00:00Z")).toISOString().slice(0, 10) ===
      m[1]
  );
}
function chronology(value: string) {
  return (
    value.slice(0, 19) +
    "." +
    (value.slice(19).match(/^\.(\d+)/)?.[1] ?? "").padEnd(9, "0")
  );
}
function record(raw: unknown, view: View): raw is Receipt {
  if (!keys(raw, recordKeys)) return false;
  const r = raw;
  if (
    !hash(r.id) ||
    !Object.hasOwn(stages, String(r.kind)) ||
    !(stages[r.kind as RecordKind] as readonly string[]).includes(
      String(r.stage),
    )
  )
    return false;
  if (
    (view === "imports" && r.kind !== "delivery") ||
    (view === "quarantine" && r.kind !== "quarantine") ||
    (view === "completeness" && r.kind !== "coverage") ||
    view === "extraction"
  )
    return false;
  if (
    r.connection !== null &&
    (typeof r.connection !== "string" ||
      !/^([a-z][a-z0-9_-]{0,63})$/.test(r.connection))
  )
    return false;
  for (const k of [
    "file_digest",
    "mapping_digest",
    "root_receipt",
    "replaced_by",
  ] as const)
    if (r[k] !== null && !hash(r[k])) return false;
  for (const k of [
    "window_start",
    "window_end",
    "received_at",
    "complete_through",
  ] as const)
    if (r[k] !== null && !iso(r[k])) return false;
  if (
    (r.window_start === null) !== (r.window_end === null) ||
    (r.window_start !== null &&
      chronology(String(r.window_start)) >= chronology(String(r.window_end)))
  )
    return false;
  if (
    r.source_revision !== null &&
    !decimal(r.source_revision, 2n ** 63n - 1n, 1n)
  )
    return false;
  if (
    (r.declared_rows !== null && !decimal(r.declared_rows, 10000n)) ||
    (r.file_bytes !== null && !decimal(r.file_bytes, 16777216n)) ||
    (r.attempts !== null && !decimal(r.attempts, 10n))
  )
    return false;
  if (
    r.focus_version !== null &&
    (typeof r.focus_version !== "string" ||
      r.focus_version.length > 8 ||
      !/^\d{1,2}\.\d{1,2}$/.test(r.focus_version))
  )
    return false;
  if (r.format !== null && r.format !== "csv" && r.format !== "parquet")
    return false;
  if (
    r.reason !== null &&
    (typeof r.reason !== "string" || !/^[a-z][a-z0-9_]{0,79}$/.test(r.reason))
  )
    return false;
  for (const k of ["finalized", "known_zero", "published"] as const)
    if (r[k] !== null && typeof r[k] !== "boolean") return false;
  if (
    r.kind !== "coverage" &&
    [r.complete_through, r.finalized, r.known_zero, r.published].some(
      (v) => v !== null,
    )
  )
    return false;
  if (
    r.kind === "coverage" &&
    (r.connection === null ||
      r.window_start === null ||
      r.received_at !== null ||
      typeof r.published !== "boolean" ||
      (r.published === false &&
        [r.complete_through, r.finalized, r.known_zero].some(
          (v) => v !== null,
        )))
  )
    return false;
  const required: Record<RecordKind, readonly string[]> = {
    delivery: [
      "window_start",
      "window_end",
      "source_revision",
      "focus_version",
      "declared_rows",
      "file_digest",
      "mapping_digest",
      "received_at",
    ],
    quarantine: [
      "connection",
      "window_start",
      "window_end",
      "source_revision",
      "declared_rows",
      "focus_version",
      "file_digest",
      "received_at",
      "reason",
    ],
    coverage: [
      "connection",
      "window_start",
      "window_end",
      "focus_version",
      "published",
    ],
  };
  const absent: Record<RecordKind, readonly string[]> = {
    delivery: ["reason", "root_receipt", "replaced_by", "attempts"],
    quarantine: ["format", "file_bytes", "mapping_digest"],
    coverage: [
      "source_revision",
      "format",
      "declared_rows",
      "file_bytes",
      "file_digest",
      "mapping_digest",
      "received_at",
      "reason",
      "root_receipt",
      "replaced_by",
      "attempts",
    ],
  };
  const kind = r.kind as RecordKind;
  if (
    required[kind].some((k) => r[k] === null) ||
    absent[kind].some((k) => r[k] !== null)
  )
    return false;
  if (kind === "coverage" && r.published === true) {
    if (
      !iso(r.complete_through) ||
      typeof r.finalized !== "boolean" ||
      typeof r.known_zero !== "boolean"
    )
      return false;
    const through = chronology(r.complete_through),
      start = chronology(String(r.window_start)),
      end = chronology(String(r.window_end));
    if (
      through <= start ||
      through > end ||
      (r.finalized === true && through !== end)
    )
      return false;
  }
  if (r.attempts !== null && r.root_receipt === null) return false;
  if (r.replaced_by === r.id) return false;
  return true;
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
];
export function parseData(raw: unknown, q: Query): Data {
  const fail = () => {
    throw new Error("invalid_response");
  };
  if (!keys(raw, envelopeKeys)) return fail();
  const d = raw;
  if (
    d.schema !== 1 ||
    d.profile !== "retained-focus-receipts-v1" ||
    d.read_only !== true ||
    d.financial_authority !== false ||
    d.commands !== "unconnected" ||
    d.view !== q.view
  )
    return fail();
  if (
    !keys(d.selection, ["q", "after"]) ||
    d.selection.q !== q.q ||
    d.selection.after !== q.after
  )
    return fail();
  if (
    (d.audience !== "tenant" && d.audience !== "company") ||
    !bounded(d.member_count, 10000) ||
    d.member_count === 0 ||
    !hash(d.auth_scope) ||
    !hash(d.permission_revision) ||
    !hash(d.snapshot) ||
    !iso(d.observed_at)
  )
    return fail();
  if (
    (q.snapshot && q.snapshot !== d.snapshot) ||
    (q.permission_revision && q.permission_revision !== d.permission_revision)
  )
    return fail();
  if (
    !keys(d.capabilities, ["receipts", "completeness", "extraction"]) ||
    d.capabilities.receipts !== "fully-authorized-metadata-only" ||
    !["registered-expectations-only", "unavailable"].includes(
      String(d.capabilities.completeness),
    ) ||
    d.capabilities.extraction !== "unconnected"
  )
    return fail();
  if (
    !Array.isArray(d.records) ||
    d.records.length > 25 ||
    !d.records.every((r) => record(r, q.view)) ||
    new Set(d.records.map((r) => r.id)).size !== d.records.length
  )
    return fail();
  const unsupported =
    q.view === "extraction" ||
    (q.view === "completeness" &&
      d.capabilities.completeness === "unavailable");
  if (unsupported) {
    if (d.total !== null || d.next_after !== null || d.records.length !== 0)
      return fail();
  } else {
    if (
      !bounded(d.total, 10000) ||
      q.after > d.total ||
      d.records.length !== Math.min(25, d.total - q.after)
    )
      return fail();
    const expected =
      q.after + d.records.length < d.total ? q.after + d.records.length : null;
    if (
      d.next_after !== expected ||
      (expected !== null && d.records.length !== 25)
    )
      return fail();
  }
  return d as unknown as Data;
}
export function query(params: URLSearchParams, view: string): Query {
  if (!(views as readonly string[]).includes(view))
    throw new Error("invalid_query");
  const allowed = [
    "receipt-q",
    "receipt-after",
    "receipt-snapshot",
    "receipt-permission",
  ];
  for (const [k] of params)
    if (
      k.startsWith("receipt-") &&
      (!allowed.includes(k) || params.getAll(k).length !== 1)
    )
      throw new Error("invalid_query");
  const q = params.get("receipt-q") ?? "",
    after = params.get("receipt-after") ?? "0";
  if (
    !identifier(q) ||
    !/^(0|[1-9][0-9]{0,4})$/.test(after) ||
    Number(after) > 10000
  )
    throw new Error("invalid_query");
  const snapshot = params.get("receipt-snapshot") ?? undefined,
    permission_revision = params.get("receipt-permission") ?? undefined;
  if (
    (snapshot !== undefined && !hash(snapshot)) ||
    (permission_revision !== undefined && !hash(permission_revision)) ||
    (snapshot === undefined) !== (permission_revision === undefined) ||
    (after !== "0" && (!snapshot || !permission_revision))
  )
    throw new Error("invalid_query");
  return {
    view: view as View,
    q,
    after: Number(after),
    ...(snapshot ? { snapshot, permission_revision } : {}),
  };
}
export function queryString(q: Query) {
  const p = new URLSearchParams({
    view: q.view,
    q: q.q,
    after: String(q.after),
  });
  if (q.snapshot) p.set("snapshot", q.snapshot);
  if (q.permission_revision)
    p.set("permission_revision", q.permission_revision);
  return p.toString();
}
