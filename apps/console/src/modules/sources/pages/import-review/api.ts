import {
  parseData,
  queryString,
  keys,
  type Data,
  type Query,
} from "./contract";
export class ReadError extends Error {
  constructor(
    public code: string,
    public requestId = "",
    public httpStatus?: number,
  ) {
    super(code);
  }
}
const MAX_BYTES = 2 * 1024 * 1024;
export function parseJson(text: string): unknown {
  const stack: {
    kind: "object" | "array";
    key: boolean;
    names: Set<string>;
  }[] = [];
  for (let i = 0; i < text.length; i++) {
    const c = text[i],
      top = stack.at(-1);
    if (c === '"') {
      const start = i;
      let closed = false;
      for (i++; i < text.length; i++) {
        if (text[i] === "\\") {
          i++;
          continue;
        }
        if (text[i] === '"') {
          closed = true;
          break;
        }
      }
      if (!closed) throw new ReadError("invalid_response");
      if (top?.kind === "object" && top.key) {
        const name = JSON.parse(text.slice(start, i + 1));
        if (top.names.has(name)) throw new ReadError("invalid_response");
        top.names.add(name);
        top.key = false;
      }
    } else if (c === "{" || c === "[") {
      stack.push({
        kind: c === "{" ? "object" : "array",
        key: c === "{",
        names: new Set(),
      });
      if (stack.length > 16) throw new ReadError("invalid_response");
    } else if (c === "}" || c === "]") {
      stack.pop();
    } else if (c === "," && top?.kind === "object") top.key = true;
  }
  try {
    return JSON.parse(text);
  } catch {
    throw new ReadError("invalid_response");
  }
}
export async function body(response: Response) {
  const mime = response.headers.get("content-type") ?? "";
  if (!/^application\/json(?:\s*;\s*charset=utf-8)?$/i.test(mime))
    throw new ReadError("invalid_response");
  const rawLength = response.headers.get("content-length");
  if (
    rawLength !== null &&
    (!/^(0|[1-9][0-9]*)$/.test(rawLength) ||
      BigInt(rawLength) > BigInt(MAX_BYTES))
  )
    throw new ReadError("response_size");
  const reader = response.body?.getReader();
  if (!reader) throw new ReadError("invalid_response");
  const chunks: Uint8Array[] = [];
  let size = 0;
  try {
    while (true) {
      const r = await reader.read();
      if (r.done) break;
      size += r.value.byteLength;
      if (size > MAX_BYTES) throw new ReadError("response_size");
      chunks.push(r.value);
    }
  } catch (e) {
    await reader.cancel().catch(() => {});
    throw e;
  } finally {
    reader.releaseLock();
  }
  const bytes = new Uint8Array(size);
  let at = 0;
  for (const chunk of chunks) {
    bytes.set(chunk, at);
    at += chunk.byteLength;
  }
  let text: string;
  try {
    text = new TextDecoder("utf-8", { fatal: true }).decode(bytes);
  } catch {
    throw new ReadError("invalid_response");
  }
  return parseJson(text);
}
export type ReadResult =
  | { kind: "loaded"; data: Data }
  | { kind: "unconnected" };
export async function load(q: Query, signal: AbortSignal): Promise<ReadResult> {
  const captured: Query = Object.freeze({ ...q });
  const aborted = () => {
    if (signal.aborted) throw new DOMException("Request aborted", "AbortError");
  };
  aborted();
  let response: Response;
  try {
    response = await fetch("/api/focus-receipts?" + queryString(captured), {
      method: "GET",
      headers: { "X-HB-Local-Client": "1", Accept: "application/json" },
      cache: "no-store",
      redirect: "error",
      signal,
    });
  } catch (e) {
    if (signal.aborted) throw e;
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
  } catch (e) {
    await response.body?.cancel().catch(() => {});
    if (signal.aborted) throw e;
    throw e instanceof ReadError ? e : new ReadError("invalid_response");
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
