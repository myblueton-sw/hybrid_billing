import { strictJSON } from "./json";
import { rationaleValid } from "./packet";
import { serviceClient, type Session } from "../../../../../service/api";
import { attempt, type Attempt } from "../../../../../service/admin";
import { bytes, digest, DocumentError, exact, freeze, id, object, revision, timestamp, type ReadEnvelope } from "./contract";
export type DocumentAction = "propose" | "confirm" | "revoke" | "prepare";
export type Intent = Readonly<{ action: DocumentAction; target: string; fields: Readonly<Record<string, string>>; rationale: string; revision: string; attempt: Readonly<Attempt> }>;
export type Receipt = Readonly<{ target: string; digest: string; state: string; documentDigest?: string; revision: string; replay: boolean; operation: string }>;
const readCommands = ["document_evidence.context", "document_evidence.history", "document_evidence.advice", "document_evidence.inspect", "erp_preparation.inspect"];
const refusals: Record<string, number> = { permission_denied: 403, self_approval_denied: 403, unauthenticated: 401, csrf_refused: 403, reauthentication_required: 403, evidence_stale: 409, revision_conflict: 409, invalid_request: 422, prerequisite_failed: 422, rate_limited: 429, unsupported_capability: 501 };
export function freezeIntent(intent: Intent): Intent {
  if (!["propose", "confirm", "revoke", "prepare"].includes(intent.action) || !id(intent.target) || !revision(intent.revision) || !id(intent.attempt?.request) || !id(intent.attempt?.key) || typeof intent.rationale !== "string" || !rationaleValid(intent.rationale)) throw new DocumentError("invalid_document_request");
  const keys = intent.action === "propose" ? ["packet"] : intent.action === "prepare" ? ["document", "digest"] : ["digest"];
  exact(intent.fields, keys);
  if (intent.action === "propose" ? typeof intent.fields.packet !== "string" || !intent.fields.packet || bytes(intent.fields.packet) > 4096 : !digest(intent.fields.digest) || intent.action === "prepare" && !id(intent.fields.document)) throw new DocumentError("invalid_document_request");
  return freeze(structuredClone(intent));
}
const refusalNext: Record<string, string> = { permission_denied: "none", self_approval_denied: "none", unauthenticated: "obtain_new_access_answer", csrf_refused: "none", reauthentication_required: "reauthenticate", evidence_stale: "inspect_current_state", revision_conflict: "inspect_current_state", invalid_request: "correct_request", prerequisite_failed: "correct_request", rate_limited: "retry_later", unsupported_capability: "none" };
function cleanRefusal(raw: Record<string, unknown>, code: string, status: number, request: string): boolean {
  if (raw.operation_id !== undefined || raw.request_id !== request || refusals[code] !== status || raw.retry !== (code === "rate_limited") || raw.next_action !== refusalNext[code] || Object.keys(raw).some(k => !["code", "retry", "next_action", "request_id", "fields", "retry_after_ms"].includes(k))) return false;
  return (raw.fields === undefined || Array.isArray(raw.fields) && raw.fields.length <= 17 && raw.fields.every(v => typeof v === "string" && bytes(v) <= 128)) && (raw.retry_after_ms === undefined || Number.isSafeInteger(raw.retry_after_ms) && Number(raw.retry_after_ms) >= 0);
}
async function responseText(response: Response): Promise<string> {
  if (!response.body) throw new DocumentError("invalid_document_response");
  const reader = response.body.getReader(), decoder = new TextDecoder("utf-8", { fatal: true });
  let size = 0, text = "";
  try {
    for (;;) {
      const part = await reader.read();
      if (part.done) break;
      size += part.value.byteLength;
      if (size > 100000) throw new DocumentError("invalid_document_response");
      text += decoder.decode(part.value, { stream: true });
    }
    return text + decoder.decode();
  } finally { await reader.cancel().catch(() => {}); reader.releaseLock(); }
}
async function bounded<T>(work: (signal: AbortSignal) => Promise<T>, effect: "none" | "unknown"): Promise<T> {
  const controller = new AbortController();
  let timer: ReturnType<typeof setTimeout>;
  const timeout = new Promise<never>((_, reject) => { timer = setTimeout(() => { reject(new DocumentError("request_timeout", effect)); controller.abort(); }, 30000); });
  try { return await Promise.race([work(controller.signal), timeout]); } finally { clearTimeout(timer!); }
}
function fieldMap(raw: unknown): Record<string, string> {
  if (!object(raw) || Object.keys(raw).length > 17 || Object.entries(raw).some(([k, v]) => !/^[a-z][a-z0-9_]*$/.test(k) || typeof v !== "string" || bytes(v) > 4096)) throw new DocumentError("invalid_document_response");
  return raw as Record<string, string>;
}
export function documentTransport(session: Session, f: typeof fetch = fetch.bind(globalThis)) {
  const owner = freeze({ ...session });
  async function send(command: string, target: string, fields: Readonly<Record<string, string>>, tried: Attempt, intent?: Intent): Promise<Record<string, unknown>> {
    let current: Session | null;
    try { current = await bounded(signal => serviceClient((input, init) => f(input, { ...init, signal })).current(), "none"); }
    catch { throw new DocumentError("session_unavailable"); }
    if (!current || current.user !== owner.user || current.workspace !== owner.workspace || current.csrf !== owner.csrf) throw new DocumentError("session_changed");
    let response: Response, raw: unknown;
    try {
      const got = await bounded(async signal => {
        const r = await f(`/api/v1/commands/${command}`, { signal, method: "POST", credentials: "same-origin", cache: "no-store", headers: { "Content-Type": "application/json", "X-CSRF-Token": owner.csrf, "X-Request-Id": tried.request }, body: JSON.stringify({ request_id: tried.request, target_id: target, ...(intent ? { idempotency_key: tried.key, expected_revision: intent.revision, rationale: intent.rationale } : {}), body: fields }) });
        const text = await responseText(r);
        if (bytes(text) > 100000) throw new Error("bounded response");
        return { response: r, raw: strictJSON(text) };
      }, intent ? "unknown" : "none");
      response = got.response; raw = got.raw;
    } catch (error) { throw new DocumentError(error instanceof DocumentError ? error.code : "service_unavailable", intent ? "unknown" : "none", error instanceof DocumentError ? error.operation : ""); }
    if (!object(raw)) throw new DocumentError("invalid_document_response", intent ? "unknown" : "none");
    if (!response.ok) {
      const code = typeof raw.code === "string" && /^[a-z_]{1,80}$/.test(raw.code) ? raw.code : "service_unavailable";
      const none = cleanRefusal(raw, code, response.status, tried.request);
      throw new DocumentError(code, intent && !none ? "unknown" : "none", raw.request_id === tried.request && id(raw.operation_id) ? raw.operation_id : "");
    }
    if (response.status !== 200 || raw.request_id !== tried.request || raw.target_id !== target || raw.status !== "completed") throw new DocumentError("invalid_document_response", intent ? "unknown" : "none");
    return raw;
  }
  return {
    async read(command: string, target: string, fields: Record<string, string>): Promise<ReadEnvelope> {
      if (!readCommands.includes(command) || !id(target)) throw new DocumentError("invalid_document_request");
      const values = freeze(structuredClone(fields));
      const raw = await send(command, target, values, attempt());
      exact(raw, ["request_id", "status", "target_id", "result_revision", "replay", "as_of", "fields"]);
      if (!revision(raw.result_revision) || raw.replay !== false || !timestamp(raw.as_of)) throw new DocumentError("invalid_document_response");
      return freeze({ revision: raw.result_revision, at: raw.as_of, fields: fieldMap(raw.fields) });
    },
    async submit(intent: Intent): Promise<Receipt> {
      intent = freezeIntent(intent);
      const command = intent.action === "prepare" ? "erp_preparation.prepare" : `document_evidence.${intent.action}`;
      const raw = await send(command, intent.target, intent.fields, intent.attempt, intent);
      try {
        exact(raw, ["request_id", "status", "operation_id", "target_id", "result_revision", "business_state", "replay", "as_of", "fields"]);
        if (!id(raw.operation_id) || !revision(raw.result_revision) || typeof raw.replay !== "boolean" || !timestamp(raw.as_of)) throw new Error();
        const fields = fieldMap(raw.fields), state = { propose: "pending", confirm: "confirmed", revoke: "revoked", prepare: "prepared" }[intent.action];
        if (raw.business_state !== state || fields.mode !== "synthetic" || fields.recorded_state !== state || !digest(fields.digest)) throw new Error();
        if (intent.action === "prepare") {
          exact(fields, ["mode", "preparation", "document_digest", "digest", "recorded_state", "receipt"]);
          if (fields.preparation !== intent.target || fields.document_digest !== intent.fields.digest || fields.receipt !== "historical_preparation") throw new Error();
        } else {
          exact(fields, ["mode", "document", "digest", "recorded_state", "receipt"]);
          if (fields.document !== intent.target || fields.receipt !== "historical_decision" || intent.action !== "propose" && fields.digest !== intent.fields.digest) throw new Error();
        }
        return freeze({ target: intent.target, digest: fields.digest, state, documentDigest: fields.document_digest, revision: raw.result_revision, replay: raw.replay, operation: raw.operation_id });
      } catch { throw new DocumentError("invalid_document_receipt", "unknown", id(raw.operation_id) ? raw.operation_id : ""); }
    },
  };
}
