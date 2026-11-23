import type { Session } from "../../../../../service/api";
import type { Attempt } from "../../../../../service/admin";
import { timestamp } from "../documents/contract";
import { rationaleValid } from "../documents/packet";
import { strictJSON } from "../documents/json";
import { bytes, digest, exact, freeze, id, object, revision, ResultError } from "./contract";
import { application, type ApplicationState } from "./history";
import { bounded, checkSession, fieldMap, responseText } from "./transport";
export type MutationAction = "propose" | "confirm" | "revoke" | "apply";
export type Intent = Readonly<{ action: MutationAction; target: string; fields: Readonly<Record<string, string>>; rationale: string; revision: string; attempt: Readonly<Attempt> }>;
export type Receipt = Readonly<{ target: string; digest: string; state: string; applicationState: ApplicationState; applicationReceipt: string; revision: string; replay: boolean; operation: string }>;
export function freezeIntent(intent: Intent): Intent {
 if (!["propose", "confirm", "revoke", "apply"].includes(intent.action) || !id(intent.target) || !revision(intent.revision) || !id(intent.attempt?.request) || !id(intent.attempt?.key) || typeof intent.rationale !== "string" || !rationaleValid(intent.rationale)) throw new ResultError("invalid_result_request");
 exact(intent.fields, intent.action === "propose" ? ["packet"] : ["digest"]);
 if (intent.action === "propose" ? typeof intent.fields.packet !== "string" || !intent.fields.packet || bytes(intent.fields.packet) > 4096 : !digest(intent.fields.digest)) throw new ResultError("invalid_result_request");
 return freeze(structuredClone(intent));
}
const refusalRules: Record<string, [number, string]> = { permission_denied: [403,"none"], self_approval_denied: [403,"none"], unauthenticated: [401,"obtain_new_access_answer"], csrf_refused: [403,"none"], reauthentication_required: [403,"reauthenticate"], evidence_stale: [409,"inspect_current_state"], revision_conflict: [409,"inspect_current_state"], invalid_request: [422,"correct_request"], prerequisite_failed: [422,"correct_request"], rate_limited: [429,"retry_later"], unsupported_capability: [501,"none"] };
function cleanRefusal(raw: Record<string, unknown>, code: string, status: number, request: string) {
 const rule = refusalRules[code];
 if (!rule || raw.request_id !== request || raw.operation_id !== undefined || status !== rule[0] || raw.next_action !== rule[1] || raw.retry !== (code === "rate_limited") || Object.keys(raw).some(k => !["code", "retry", "next_action", "request_id", "fields", "retry_after_ms"].includes(k))) return false;
 return (raw.fields === undefined || Array.isArray(raw.fields) && raw.fields.length <= 28 && raw.fields.every(v => typeof v === "string" && bytes(v) <= 128)) && (raw.retry_after_ms === undefined || Number.isSafeInteger(raw.retry_after_ms) && Number(raw.retry_after_ms) >= 0);
}
export function resultMutationTransport(session: Session, f: typeof fetch = fetch.bind(globalThis)) {
 const owner = freeze({ ...session });
 return async function submit(input: Intent): Promise<Receipt> {
  const intent = freezeIntent(input), tried = intent.attempt;
  await checkSession(owner, f);
  const command = intent.action === "apply" ? "erp_result.apply" : `result_evidence.${intent.action}`;
  let response: Response, raw: unknown;
  try {
   const got = await bounded(async signal => {
    const response = await f(`/api/v1/commands/${command}`, { signal, method: "POST", credentials: "same-origin", cache: "no-store", headers: { "Content-Type": "application/json", "X-CSRF-Token": owner.csrf, "X-Request-Id": tried.request }, body: JSON.stringify({ request_id: tried.request, target_id: intent.target, idempotency_key: tried.key, expected_revision: intent.revision, rationale: intent.rationale, body: intent.fields }) });
    return { response, raw: strictJSON(await responseText(response),125000) };
   }, "unknown");
   response = got.response; raw = got.raw;
  } catch (e) { throw new ResultError(e instanceof ResultError ? e.code : "service_unavailable", "unknown"); }
  if (!object(raw)) throw new ResultError("invalid_result_response", "unknown");
  const operation = raw.request_id === tried.request && id(raw.operation_id) ? raw.operation_id : "";
  if (!response.ok) {
   const code = typeof raw.code === "string" && /^[a-z_]{1,80}$/.test(raw.code) ? raw.code : "service_unavailable";
   throw new ResultError(code, cleanRefusal(raw,code,response.status,tried.request) ? "none" : "unknown", operation);
  }
  try {
   exact(raw,["request_id","status","operation_id","target_id","result_revision","business_state","replay","as_of","fields"]);
   if (response.status !== 200 || raw.request_id !== tried.request || raw.target_id !== intent.target || raw.status !== "completed" || !id(raw.operation_id) || !revision(raw.result_revision) || typeof raw.replay !== "boolean" || !timestamp(raw.as_of)) throw new Error();
   const fields = fieldMap(raw.fields), state = fields.recorded_state;
   exact(fields,["mode","result","digest","recorded_state","receipt","application_state","application_receipt"]);
   if (fields.mode !== "synthetic" || fields.result !== intent.target || !digest(fields.digest) || !["pending", "confirmed", "revoked"].includes(state) || intent.action === "confirm" && state !== "confirmed" || intent.action === "revoke" && state !== "revoked" || intent.action === "apply" && state !== "confirmed" || fields.receipt !== "historical_decision" || !application(fields.application_state,fields.application_receipt) || raw.business_state !== (intent.action === "apply" ? "recorded" : state) || intent.action !== "propose" && fields.digest !== intent.fields.digest || intent.action === "apply" && fields.application_state !== "recorded" || state === "pending" && fields.application_state !== "not_started") throw new Error();
   return freeze({ target: intent.target, digest: fields.digest, state, applicationState: fields.application_state, applicationReceipt: fields.application_receipt, revision: raw.result_revision, replay: raw.replay, operation: raw.operation_id });
  } catch { throw new ResultError("invalid_result_receipt", "unknown", operation); }
 };
}
