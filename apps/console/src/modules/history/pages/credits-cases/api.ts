import { type Data, type ReadResult } from "./contract";
import { valid, validQuery } from "./validation";
export * from "./contract";
export {
  valid,
  validQuery,
  validDecimal,
  validDate,
  validTimeZone,
} from "./validation";
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
export function query(params: URLSearchParams, view: string) {
  const q = new URLSearchParams({
    view,
    q: (params.get("credit-q") ?? "").trim().replace(/\s+/g, " "),
  });
  for (const k of ["cursor", "snapshot", "permission_revision"])
    if (params.has("credit-" + k)) q.set(k, params.get("credit-" + k)!);
  return q;
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
    response = await fetch("/api/credit-case-review?" + q.toString(), {
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
