import { ServiceError } from "../../../sources/pages/live-imports/api";
import {
  kinds,
  views,
  type ApprovalRecord,
  type Data,
  type ReadResult,
  type View,
} from "./contract";
export * from "./contract";
export { ServiceError };
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
function coefficient(v: string, scale: number) {
  const negative = v.startsWith("-"),
    [whole, fraction = ""] = (negative ? v.slice(1) : v).split(".");
  const n = BigInt(whole + fraction.padEnd(scale, "0"));
  return negative ? -n : n;
}
export function differenceMatches(
  base: string,
  candidate: string,
  difference: string,
) {
  if (![base, candidate, difference].every(validDecimal)) return false;
  const scale = Math.max(
    ...[base, candidate, difference].map((v) => v.split(".")[1]?.length ?? 0),
  );
  return (
    coefficient(candidate, scale) - coefficient(base, scale) ===
    coefficient(difference, scale)
  );
}
function principal(v: unknown) {
  return (
    exact(v, ["id", "label"]) &&
    id((v as Data["actor"]).id) &&
    text((v as Data["actor"]).label, 256)
  );
}
function record(r: ApprovalRecord) {
  if (
    !exact(r, [
      "id",
      "kind",
      "status",
      "target",
      "author",
      "approver",
      "updated_at",
      "deadline",
      "decided_at",
      "amounts",
      "erp",
      "evidence",
      "authority",
    ]) ||
    !id(r.id) ||
    !kinds.includes(r.kind) ||
    ![
      "pending",
      "blocked",
      "approved",
      "rejected",
      "expired",
      "superseded",
    ].includes(r.status)
  )
    return false;
  if (
    !exact(r.target, ["id", "label", "entity", "customer", "period"]) ||
    !id(r.target.id) ||
    !text(r.target.label, 512) ||
    !nullable(r.target.entity, 256) ||
    !nullable(r.target.customer, 256) ||
    !nullable(r.target.period, 64) ||
    !(r.author === null || principal(r.author)) ||
    !(r.approver === null || principal(r.approver)) ||
    !iso(r.updated_at) ||
    !optionalTime(r.deadline) ||
    !optionalTime(r.decided_at)
  )
    return false;
  if (
    !Array.isArray(r.amounts) ||
    r.amounts.length > 16 ||
    new Set(r.amounts.map((a) => `${a?.currency}:${a?.basis}`)).size !==
      r.amounts.length ||
    !r.amounts.every(
      (a) =>
        exact(a, [
          "currency",
          "basis",
          "base",
          "candidate",
          "difference",
          "erp",
        ]) &&
        /^[A-Z]{3}$/.test(a.currency) &&
        ["net", "tax", "gross", "expense"].includes(a.basis) &&
        [a.base, a.candidate, a.difference, a.erp].every(
          (v) => v === null || validDecimal(v),
        ) &&
        (a.base === null ||
          a.candidate === null ||
          a.difference === null ||
          differenceMatches(a.base, a.candidate, a.difference)),
    )
  )
    return false;
  if (
    !exact(r.erp, ["applicability", "external_revision", "observed_at"]) ||
    !["applicable", "not_applicable", "unknown"].includes(
      r.erp.applicability,
    ) ||
    !nullable(r.erp.external_revision, 256) ||
    !optionalTime(r.erp.observed_at) ||
    (r.erp.applicability === "not_applicable" &&
      (r.amounts.some((a) => a.erp !== null) ||
        r.erp.external_revision !== null ||
        r.erp.observed_at !== null ||
        r.kind === "final_erp_amount"))
  )
    return false;
  const e = r.evidence;
  if (
    !exact(e, [
      "input_digest",
      "result_digest",
      "revision",
      "dependencies",
      "completeness",
      "validation",
      "blockers",
    ]) ||
    !(e.input_digest === null || hash(e.input_digest)) ||
    !(e.result_digest === null || hash(e.result_digest)) ||
    !nullable(e.revision, 256) ||
    !["current", "stale", "unknown"].includes(e.dependencies) ||
    !["complete", "partial", "unknown"].includes(e.completeness) ||
    !["passed", "failed", "unknown"].includes(e.validation) ||
    !Array.isArray(e.blockers) ||
    e.blockers.length > 16 ||
    new Set(e.blockers.map((b) => b?.code)).size !== e.blockers.length ||
    !e.blockers.every(
      (b) => exact(b, ["code", "text"]) && id(b.code) && text(b.text, 1024),
    )
  )
    return false;
  return (
    exact(r.authority, ["separation", "limit", "valid_until"]) &&
    ["satisfied", "not_satisfied", "unknown"].includes(
      r.authority.separation,
    ) &&
    ["within", "exceeded", "unknown"].includes(r.authority.limit) &&
    optionalTime(r.authority.valid_until)
  );
}
export function query(params: URLSearchParams, view: string): URLSearchParams {
  const q = new URLSearchParams({ view });
  q.set(
    "kind",
    view === "recognition"
      ? "monthly_recognition"
      : (params.get("approval-kind") ?? ""),
  );
  q.set("q", (params.get("approval-q") ?? "").trim().replace(/\s+/g, " "));
  for (const k of ["cursor", "snapshot", "permission_revision"])
    if (params.has("approval-" + k)) q.set(k, params.get("approval-" + k)!);
  return q;
}
export function validQuery(q: URLSearchParams) {
  if (
    [...q.keys()].some(
      (k) =>
        ![
          "view",
          "kind",
          "q",
          "cursor",
          "snapshot",
          "permission_revision",
        ].includes(k),
    ) ||
    new Set(q.keys()).size !== [...q.keys()].length ||
    !views.includes(q.get("view") as View) ||
    !["", ...kinds].includes(q.get("kind") ?? "") ||
    (q.get("q") ?? "").length > 128 ||
    /[\u0000-\u001f\u007f]/.test(q.get("q") ?? "")
  )
    return false;
  if (
    (q.has("snapshot") && !hash(q.get("snapshot"))) ||
    (q.has("permission_revision") && !hash(q.get("permission_revision")))
  )
    return false;
  if (
    q.has("cursor") &&
    (!cursor(q.get("cursor")) ||
      !hash(q.get("snapshot")) ||
      !hash(q.get("permission_revision")))
  )
    return false;
  return (
    q.get("view") !== "recognition" || q.get("kind") === "monthly_recognition"
  );
}
export function valid(d: Data, q: URLSearchParams): boolean {
  try {
    if (
      !validQuery(q) ||
      q.get("view") === "delegation" ||
      !exact(d, [
        "schema",
        "numeric_format",
        "read_only",
        "commands",
        "source_mode",
        "auth_scope",
        "snapshot",
        "permission_revision",
        "actor",
        "view",
        "selection",
        "cursor",
        "next_cursor",
        "observed_at",
        "total",
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
      !principal(d.actor) ||
      !iso(d.observed_at) ||
      d.view !== q.get("view") ||
      !exact(d.selection, ["kind", "q"]) ||
      d.selection.kind !== (q.get("kind") ?? "") ||
      d.selection.q !== (q.get("q") ?? "") ||
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
        (!d.selection.kind || r.kind === d.selection.kind) &&
        (d.view !== "recognition" || r.kind === "monthly_recognition") &&
        (d.view !== "decisions" ||
          (["approved", "rejected"].includes(r.status) &&
            r.approver?.id === d.actor.id)) &&
        (d.view !== "pending" ||
          ["pending", "blocked", "expired", "superseded"].includes(r.status)),
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
  if (!validQuery(q)) throw new ServiceError("request_invalid", "");
  if (signal.aborted) throw new DOMException("Aborted", "AbortError");
  if (q.get("view") === "delegation") return { kind: "unconnected" };
  let response: Response;
  try {
    response = await fetch("/api/approval-review?" + q.toString(), {
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
