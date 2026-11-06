import {
  protections,
  views,
  type ClaimRecord,
  type Data,
  type ReadResult,
  type View,
} from "./contract";
export * from "./contract";
export class ServiceError extends Error {
  constructor(
    public readonly code: string,
    public readonly requestId: string,
  ) {
    super(code);
    this.name = "ServiceError";
  }
}
const maxBytes = 2 * 1024 * 1024;
const hash = (v: unknown): v is string =>
  typeof v === "string" && /^[a-f0-9]{64}$/.test(v);
const text = (v: unknown, max = 4096): v is string =>
  typeof v === "string" &&
  v.length > 0 &&
  v.length <= max &&
  !/[\u0000-\u001f\u007f]/.test(v);
const nullable = (v: unknown, max = 4096) => v === null || text(v, max);
const id = (v: unknown): v is string =>
  typeof v === "string" && /^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$/.test(v);
const cursor = (v: unknown): v is string =>
  typeof v === "string" && /^[A-Za-z0-9_-]{1,128}$/.test(v);
const count = (v: unknown): v is number =>
  Number.isSafeInteger(v) && Number(v) >= 0 && Number(v) <= 1000000000;
const exact = (v: unknown, names: string[]) =>
  !!v &&
  typeof v === "object" &&
  !Array.isArray(v) &&
  Object.keys(v).length === names.length &&
  names.every((k) => Object.hasOwn(v, k));
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
  const day = Date.parse(m[1] + "T00:00:00Z");
  return (
    Number.isFinite(day) &&
    new Date(day).toISOString().slice(0, 10) === m[1] &&
    Number.isFinite(Date.parse(v))
  );
}
const optionalTime = (v: unknown) => v === null || iso(v);
export function validDecimal(v: unknown): v is string {
  if (
    typeof v !== "string" ||
    v.length > 80 ||
    !/^-?(?:0|[1-9]\d*)(?:\.\d+)?$/.test(v)
  )
    return false;
  const digits = v.replace(/[-.]/g, "");
  return digits.length <= 76 && (v.split(".")[1]?.length ?? 0) <= 38;
}
const counter = (v: unknown) =>
  v === null || (validDecimal(v) && !v.startsWith("-"));
const reference = (v: unknown) => v === null || id(v);
function record(r: ClaimRecord) {
  if (
    !exact(r, [
      "id",
      "revision",
      "application_id",
      "obligation_id",
      "purpose",
      "target",
      "protection",
      "partial",
      "updated_at",
      "coverage",
      "amounts",
      "lineage",
      "correction",
      "external",
      "blockers",
    ]) ||
    !id(r.id) ||
    !nullable(r.revision, 256) ||
    !reference(r.application_id) ||
    !reference(r.obligation_id) ||
    !nullable(r.purpose, 256) ||
    !protections.includes(r.protection) ||
    !(r.partial === null || typeof r.partial === "boolean") ||
    !iso(r.updated_at)
  )
    return false;
  if (
    !exact(r.target, [
      "label",
      "entity",
      "customer",
      "contract",
      "component",
      "period",
    ]) ||
    !text(r.target.label, 512) ||
    ![
      r.target.entity,
      r.target.customer,
      r.target.contract,
      r.target.component,
    ].every((v) => nullable(v, 256)) ||
    !nullable(r.target.period, 128)
  )
    return false;
  const c = r.coverage;
  if (
    !exact(c, [
      "identity",
      "universe",
      "unit",
      "unit_version",
      "description",
      "reserved",
      "applied",
      "reversed",
      "available",
    ]) ||
    ![c.identity, c.universe, c.unit, c.unit_version].every((v) =>
      nullable(v, 256),
    ) ||
    !nullable(c.description, 1024) ||
    ![c.reserved, c.applied, c.reversed, c.available].every(counter) ||
    (r.protection === "outcome_unknown" && c.available !== null)
  )
    return false;
  if (
    !Array.isArray(r.amounts) ||
    r.amounts.length > 16 ||
    new Set(r.amounts.map((a) => `${a?.currency}:${a?.basis}`)).size !==
      r.amounts.length ||
    !r.amounts.every(
      (a) =>
        exact(a, ["currency", "basis", "applied", "remaining", "reversed"]) &&
        typeof a.currency === "string" &&
        /^[A-Z]{3}$/.test(a.currency) &&
        ["net", "tax", "gross", "expense"].includes(a.basis) &&
        [a.applied, a.remaining, a.reversed].every(
          (v) => v === null || validDecimal(v),
        ),
    )
  )
    return false;
  if (
    !exact(r.lineage, [
      "source",
      "allocation",
      "sales",
      "charge_line",
      "submission",
    ]) ||
    !Object.values(r.lineage).every(reference)
  )
    return false;
  const x = r.correction;
  if (
    !exact(x, ["original_claim_id", "linked_claim_ids", "disposition"]) ||
    !reference(x.original_claim_id) ||
    x.original_claim_id === r.id ||
    !Array.isArray(x.linked_claim_ids) ||
    x.linked_claim_ids.length > 16 ||
    new Set(x.linked_claim_ids).size !== x.linked_claim_ids.length ||
    !x.linked_claim_ids.every((v) => id(v) && v !== r.id) ||
    ![
      "none",
      "money_only",
      "quantity_reversal",
      "replacement_pending",
      "cancelled",
      "superseded",
      "unknown",
    ].includes(x.disposition)
  )
    return false;
  if (
    !Array.isArray(r.external) ||
    r.external.length > 16 ||
    new Set(r.external.map((e) => `${e?.kind}:${e?.id}`)).size !==
      r.external.length ||
    !r.external.every(
      (e) =>
        exact(e, ["kind", "id", "revision", "observed_at", "outcome"]) &&
        ["erp_order", "prepayment", "invoice", "submission", "other"].includes(
          e.kind,
        ) &&
        id(e.id) &&
        nullable(e.revision, 256) &&
        optionalTime(e.observed_at) &&
        ["pending", "unknown", "applied", "not_applied", "cancelled"].includes(
          e.outcome,
        ),
    )
  )
    return false;
  return (
    Array.isArray(r.blockers) &&
    r.blockers.length <= 16 &&
    new Set(r.blockers.map((b) => b?.code)).size === r.blockers.length &&
    r.blockers.every(
      (b) => exact(b, ["code", "text"]) && id(b.code) && text(b.text, 1024),
    )
  );
}
export function query(params: URLSearchParams, view: string): URLSearchParams {
  const q = new URLSearchParams({
    view,
    state: params.get("claim-state") ?? "",
    q: (params.get("claim-q") ?? "").trim().replace(/\s+/g, " "),
  });
  for (const k of ["cursor", "snapshot", "permission_revision"])
    if (params.has("claim-" + k)) q.set(k, params.get("claim-" + k)!);
  return q;
}
export function validQuery(q: URLSearchParams) {
  return (
    ![...q.keys()].some(
      (k) =>
        ![
          "view",
          "state",
          "q",
          "cursor",
          "snapshot",
          "permission_revision",
        ].includes(k),
    ) &&
    new Set(q.keys()).size === [...q.keys()].length &&
    views.includes(q.get("view") as View) &&
    q.has("state") &&
    ["", ...protections].includes(q.get("state") ?? "") &&
    q.has("q") &&
    (q.get("q") ?? "").length <= 128 &&
    !/[\u0000-\u001f\u007f]/.test(q.get("q") ?? "") &&
    (!q.has("snapshot") || hash(q.get("snapshot"))) &&
    (!q.has("permission_revision") || hash(q.get("permission_revision"))) &&
    (!q.has("cursor") ||
      (cursor(q.get("cursor")) &&
        hash(q.get("snapshot")) &&
        hash(q.get("permission_revision"))))
  );
}
export function valid(d: Data, q: URLSearchParams): boolean {
  try {
    if (
      !validQuery(q) ||
      !exact(d, [
        "schema",
        "numeric_format",
        "read_only",
        "commands",
        "source_mode",
        "auth_scope",
        "snapshot",
        "permission_revision",
        "view",
        "selection",
        "cursor",
        "next_cursor",
        "total",
        "observed_at",
        "records",
      ]) ||
      d.schema !== 1 ||
      d.numeric_format !== "decimal-strings-read-v1" ||
      d.read_only !== true ||
      d.commands !== "unconnected" ||
      d.source_mode !== "synthetic" ||
      !hash(d.auth_scope) ||
      !hash(d.snapshot) ||
      !hash(d.permission_revision) ||
      !iso(d.observed_at) ||
      d.view !== q.get("view") ||
      !exact(d.selection, ["state", "q"]) ||
      d.selection.state !== q.get("state") ||
      d.selection.q !== q.get("q") ||
      d.cursor !== (q.get("cursor") ?? null) ||
      !(d.next_cursor === null || cursor(d.next_cursor)) ||
      (d.next_cursor !== null && d.next_cursor === d.cursor) ||
      (q.has("snapshot") && d.snapshot !== q.get("snapshot")) ||
      (q.has("permission_revision") &&
        d.permission_revision !== q.get("permission_revision"))
    )
      return false;
    if (
      !(d.total === null || count(d.total)) ||
      !Array.isArray(d.records) ||
      d.records.length > 25 ||
      new Set(d.records.map((r) => r?.id)).size !== d.records.length ||
      !d.records.every(record) ||
      (d.total !== null && d.total < d.records.length) ||
      (d.next_cursor !== null &&
        (d.records.length !== 25 || (d.total !== null && d.total <= 25))) ||
      (d.cursor === null &&
        d.next_cursor === null &&
        d.total !== null &&
        d.total !== d.records.length)
    )
      return false;
    return d.records.every(
      (r) =>
        (!d.selection.state || r.protection === d.selection.state) &&
        (d.view !== "partial" || r.partial === true) &&
        (d.view !== "references" || r.external.length > 0) &&
        (d.view !== "reversals" ||
          r.correction.original_claim_id !== null ||
          r.correction.linked_claim_ids.length > 0 ||
          !["none", "unknown"].includes(r.correction.disposition)),
    );
  } catch {
    return false;
  }
}
async function boundedText(response: Response, signal: AbortSignal) {
  const length = response.headers.get("Content-Length");
  if (
    length !== null &&
    /^\d+$/.test(length) &&
    BigInt(length) > BigInt(maxBytes)
  ) {
    await response.body?.cancel().catch(() => {});
    throw new ServiceError("response_too_large", "");
  }
  if (!response.body) return "";
  const reader = response.body.getReader(),
    chunks: Uint8Array[] = [];
  let size = 0;
  try {
    for (;;) {
      if (signal.aborted) throw new DOMException("Aborted", "AbortError");
      const item = await reader.read();
      if (item.done) break;
      size += item.value.byteLength;
      if (size > maxBytes) {
        await reader.cancel().catch(() => {});
        throw new ServiceError("response_too_large", "");
      }
      chunks.push(item.value);
    }
  } finally {
    reader.releaseLock();
  }
  const result = new Uint8Array(size);
  let after = 0;
  for (const chunk of chunks) {
    result.set(chunk, after);
    after += chunk.length;
  }
  try {
    return new TextDecoder("utf-8", { fatal: true }).decode(result);
  } catch {
    throw new ServiceError("response_invalid", "");
  }
}
function safeFailure(body: unknown, status: number) {
  const o =
    body && typeof body === "object" ? (body as Record<string, unknown>) : {};
  const fallback =
    status === 403
      ? "permission_denied"
      : status === 409
        ? "query_stale"
        : "service_unavailable";
  return new ServiceError(
    typeof o.error === "string" && /^[a-z][a-z0-9_]{0,63}$/.test(o.error)
      ? o.error
      : fallback,
    typeof o.request_id === "string" &&
      /^[A-Za-z0-9._:-]{1,128}$/.test(o.request_id)
      ? o.request_id
      : "",
  );
}
export async function load(
  q: URLSearchParams,
  signal: AbortSignal,
): Promise<ReadResult> {
  q = new URLSearchParams(q);
  if (!validQuery(q)) throw new ServiceError("request_invalid", "");
  if (signal.aborted) throw new DOMException("Aborted", "AbortError");
  let response: Response;
  try {
    response = await fetch("/api/claim-history?" + q.toString(), {
      method: "GET",
      headers: { "X-HB-Local-Client": "1" },
      cache: "no-store",
      redirect: "error",
      signal,
    });
  } catch (e) {
    if (signal.aborted) throw e;
    throw new ServiceError("service_unavailable", "");
  }
  if (signal.aborted) {
    await response.body?.cancel().catch(() => {});
    throw new DOMException("Aborted", "AbortError");
  }
  if (response.redirected) {
    await response.body?.cancel().catch(() => {});
    throw new ServiceError("response_invalid", "");
  }
  if (response.status === 404 || response.status === 501) {
    await response.body?.cancel().catch(() => {});
    return { kind: "unconnected" };
  }
  let body: unknown;
  try {
    body = JSON.parse(await boundedText(response, signal));
  } catch (e) {
    if (signal.aborted || e instanceof ServiceError) throw e;
    throw new ServiceError("response_invalid", "");
  }
  if (signal.aborted) throw new DOMException("Aborted", "AbortError");
  if (!response.ok) throw safeFailure(body, response.status);
  if (
    body &&
    typeof body === "object" &&
    (body as { source_mode?: unknown }).source_mode === "live"
  )
    throw new ServiceError("producer_unbound", "");
  if (!valid(body as Data, q)) throw new ServiceError("response_invalid", "");
  return { kind: "loaded", data: body as Data };
}
