import { parseJson } from "../../../sources/pages/import-review/api";
import {
  keys,
  parseData,
  queryString,
  type Data,
  type Query,
} from "./contract";
export { parseJson };
export class ReadError extends Error {
  constructor(
    public code: string,
    public requestId = "",
  ) {
    super(code);
  }
}
const MAX_BYTES = 2 * 1024 * 1024;
async function body(response: Response): Promise<unknown> {
  if (
    !/^application\/json(?:\s*;\s*charset=utf-8)?$/i.test(
      response.headers.get("content-type") ?? "",
    )
  )
    throw new ReadError("invalid_response");
  const length = response.headers.get("content-length");
  if (
    length !== null &&
    (!/^(0|[1-9][0-9]{0,6})$/.test(length) ||
      BigInt(length) > BigInt(MAX_BYTES))
  )
    throw new ReadError("response_size");
  const reader = response.body?.getReader();
  if (!reader) throw new ReadError("invalid_response");
  const chunks: Uint8Array[] = [];
  let size = 0;
  try {
    while (true) {
      const result = await reader.read();
      if (result.done) break;
      size += result.value.byteLength;
      if (size > MAX_BYTES) throw new ReadError("response_size");
      chunks.push(result.value);
    }
  } catch (error) {
    await reader.cancel().catch(() => {});
    throw error;
  } finally {
    reader.releaseLock();
  }
  const bytes = new Uint8Array(size);
  let offset = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, offset);
    offset += chunk.byteLength;
  }
  try {
    return parseJson(new TextDecoder("utf-8", { fatal: true }).decode(bytes));
  } catch {
    throw new ReadError("invalid_response");
  }
}
export type ReadResult =
  { kind: "loaded"; data: Data } | { kind: "unconnected" };
export async function load(q: Query, signal: AbortSignal): Promise<ReadResult> {
  const captured = Object.freeze({ ...q });
  const aborted = () => {
    if (signal.aborted) throw new DOMException("Request aborted", "AbortError");
  };
  aborted();
  let response: Response;
  try {
    response = await fetch("/api/focus-connections?" + queryString(captured), {
      method: "GET",
      headers: { "X-HB-Local-Client": "1", Accept: "application/json" },
      cache: "no-store",
      redirect: "error",
      signal,
    });
  } catch (error) {
    if (signal.aborted) throw error;
    throw new ReadError("service_unavailable");
  }
  if (signal.aborted || response.redirected) {
    await response.body?.cancel().catch(() => {});
    aborted();
    throw new ReadError("invalid_response");
  }
  if (response.status === 404 || response.status === 501) {
    await response.body?.cancel().catch(() => {});
    return { kind: "unconnected" };
  }
  let data: unknown;
  try {
    data = await body(response);
  } catch (error) {
    await response.body?.cancel().catch(() => {});
    if (signal.aborted) throw error;
    throw error instanceof ReadError
      ? error
      : new ReadError("invalid_response");
  }
  aborted();
  if (!response.ok) {
    if (
      !keys(data, ["error", "request_id"]) ||
      typeof data.error !== "string" ||
      !/^[a-z][a-z0-9_]{0,79}$/.test(data.error) ||
      typeof data.request_id !== "string" ||
      !/^[a-f0-9]{32}$/.test(data.request_id)
    )
      throw new ReadError("invalid_response");
    throw new ReadError(data.error, data.request_id);
  }
  try {
    return { kind: "loaded", data: parseData(data, captured) };
  } catch {
    throw new ReadError("invalid_response");
  }
}
