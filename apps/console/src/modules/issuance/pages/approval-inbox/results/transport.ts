import { strictJSON } from "../documents/json";
import { timestamp } from "../documents/contract";
import { type Session } from "../../../../../service/api";
import { attempt } from "../../../../../service/admin";
import { bytes, exact, freeze, id, object, revision, ResultError, type ReadEnvelope } from "./contract";

const readCommands = ["result_evidence.preparations", "result_evidence.context", "result_evidence.history", "result_evidence.advice", "result_evidence.inspect", "erp_result.inspect"];
export async function responseText(response: Response, maximum = 125000): Promise<string> {
  if (!response.body) throw new ResultError("invalid_result_response");
  const reader = response.body.getReader(), decoder = new TextDecoder("utf-8", { fatal: true });
  let size = 0, text = "";
  try {
    for (;;) {
      const part = await reader.read();
      if (part.done) break;
      size += part.value.byteLength;
      if (size > maximum) throw new ResultError("invalid_result_response");
      text += decoder.decode(part.value, { stream: true });
    }
    return text + decoder.decode();
  } finally { await reader.cancel().catch(() => {}); reader.releaseLock(); }
}
export async function bounded<T>(work: (signal: AbortSignal) => Promise<T>, effect: "none" | "unknown"): Promise<T> {
  const controller = new AbortController();
  let timer: ReturnType<typeof setTimeout>;
  const timeout = new Promise<never>((_, reject) => { timer = setTimeout(() => { reject(new ResultError("request_timeout", effect)); controller.abort(); }, 30000); });
  try { return await Promise.race([work(controller.signal), timeout]); } finally { clearTimeout(timer!); }
}
export function fieldMap(raw: unknown): Record<string, string> {
  if (!object(raw) || Object.keys(raw).length > 28 || Object.entries(raw).some(([k, v]) => !/^[a-z][a-z0-9_]*$/.test(k) || typeof v !== "string" || bytes(v) > 4096)) throw new ResultError("invalid_result_response");
  return raw as Record<string, string>;
}
export async function checkSession(owner: Readonly<Session>, f: typeof fetch): Promise<void> {
    let current: Session | null;
    try { current = await bounded(async signal => {
      const response = await f("/api/v1/session", { signal, credentials: "same-origin", cache: "no-store", headers: { "X-HB-Client": "console" } });
      if (response.status === 401) return null;
      if (response.status !== 200) throw new ResultError("session_unavailable");
      const raw = strictJSON(await responseText(response, 16384), 16384);
      if (!object(raw) || !id(raw.user) || !id(raw.workspace) || typeof raw.csrf_token !== "string" || !raw.csrf_token || bytes(raw.csrf_token) > 4096) throw new ResultError("session_unavailable");
      return { ...owner, user: raw.user, workspace: raw.workspace, csrf: raw.csrf_token };
    }, "none"); }
    catch { throw new ResultError("session_unavailable"); }
    if (!current || current.user !== owner.user || current.workspace !== owner.workspace || current.csrf !== owner.csrf) throw new ResultError("session_changed");
}
export function resultReadTransport(session: Session, f: typeof fetch = fetch.bind(globalThis)) {
  const owner = freeze({ ...session });
  return async function read(command: string, target: string, fields: Record<string, string>): Promise<ReadEnvelope> {
    if (!readCommands.includes(command) || !id(target)) throw new ResultError("invalid_result_request");
    const values = freeze(structuredClone(fieldMap(fields)));
    await checkSession(owner, f);
    const tried = attempt();
    let response: Response, raw: unknown;
    try {
      const got = await bounded(async signal => {
        const response = await f(`/api/v1/commands/${command}`, { signal, method: "POST", credentials: "same-origin", cache: "no-store", headers: { "Content-Type": "application/json", "X-CSRF-Token": owner.csrf, "X-Request-Id": tried.request }, body: JSON.stringify({ request_id: tried.request, target_id: target, body: values }) });
        return { response, raw: strictJSON(await responseText(response), 125000) };
      }, "none");
      response = got.response; raw = got.raw;
    } catch (error) { throw new ResultError(error instanceof ResultError ? error.code : "service_unavailable"); }
    if (!object(raw)) throw new ResultError("invalid_result_response");
    if (!response.ok) {
      const code = raw.request_id === tried.request && typeof raw.code === "string" && /^[a-z_]{1,80}$/.test(raw.code) ? raw.code : "service_unavailable";
      throw new ResultError(code);
    }
    exact(raw, ["request_id", "status", "target_id", "result_revision", "replay", "as_of", "fields"]);
    if (response.status !== 200 || raw.request_id !== tried.request || raw.target_id !== target || raw.status !== "completed" || !revision(raw.result_revision) || raw.replay !== false || !timestamp(raw.as_of)) throw new ResultError("invalid_result_response");
    return freeze({ revision: raw.result_revision, at: raw.as_of, fields: fieldMap(raw.fields) });
  };
}
