import { serviceClient, type Session } from "../../../../service/api";
import { attempt, type Attempt } from "../../../../service/admin";
import { validDecimal, type ApprovalRecord } from "./api";

export const decisionOutcomes = ["eligible", "permission_denied", "self_approval_denied", "authority_unavailable", "evidence_stale", "dependency_unavailable", "unsupported_capability", "prerequisite_failed", "reauthentication_required", "hold_active", "amount_authority_missing", "not_pending"] as const;
export type DecisionOutcome = (typeof decisionOutcomes)[number];
export type DecisionFamily = "financial_charge" | "erp_final";
export type DecisionAction = "approve" | "reject";
export const rejectionReasons = ["incorrect_amount", "missing_evidence", "policy_violation", "other"] as const;
export type Preparation = Readonly<{
  family: DecisionFamily; request: string; digest: string; recorded_state: string; author: string; requester: string;
  amount: string; currency: string; amount_basis: "rated_candidate_total" | "synthetic_reported_net_plus_tax_v1"; policy_revision: string;
  checked_at: string; revision: string; approve: DecisionOutcome; reject: DecisionOutcome;
}>;
export type Submission = Readonly<{
  family: DecisionFamily; amount_basis: Preparation["amount_basis"]; action: DecisionAction; target: string; digest: string; revision: string; rationale: string; reason: string;
  amount: string; currency: string; attempt: Readonly<Attempt>;
}>;
export type Receipt = Readonly<{ request: string; target: string; digest: string; state: "approved" | "rejected"; revision: string; replay: boolean }>;
export class DecisionError extends Error {
  constructor(readonly code: string, readonly effect: "none" | "unknown", readonly operation = "") { super(code); }
}
const object = (v: unknown): v is Record<string, unknown> => !!v && typeof v === "object" && !Array.isArray(v);
const exact = (v: unknown, keys: string[]): v is Record<string, unknown> => object(v) && Object.keys(v).length === keys.length && keys.every(k => Object.hasOwn(v, k));
const id = (v: unknown): v is string => typeof v === "string" && /^[A-Za-z0-9._:-]{1,128}$/.test(v);
const revision = (v: unknown): v is string => typeof v === "string" && /^r(?:0|[1-9][0-9]*)$/.test(v);
const timestamp = (v: unknown): v is string => {
  if (typeof v !== "string" || !/^(?!0000)\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/.test(v)) return false;
  const parsed = Date.parse(v);
  return Number.isFinite(parsed) && new Date(parsed).toISOString().slice(0,19) === v.slice(0,19);
};
export const sameSession = (a: Session, b: Session | null) => !!b && a.user === b.user && a.workspace === b.workspace && a.csrf === b.csrf;
export function decisionFamily(r: ApprovalRecord): DecisionFamily | undefined {
 if (!r.author || !r.requester || !r.policy_revision || !r.evidence.revision || r.amounts.length !== 1 || r.amounts[0].candidate === null || !r.evidence.blockers.some(b => b.code === "current_checks_unverified")) return undefined;
 if (r.kind === "billing_result" && r.amounts[0].basis === "rated_candidate_total") return "financial_charge";
 if (r.kind === "final_erp_amount" && r.amounts[0].basis === "synthetic_reported_net_plus_tax_v1") return "erp_final";
 return undefined;
}
export function serviceDecisionRecord(r: ApprovalRecord): boolean { return decisionFamily(r) !== undefined; }
export function preparationOf(raw: unknown, record: ApprovalRecord): Preparation {
  const keys = ["mode", "receipt", "request", "digest", "recorded_state", "author", "requester", "amount", "currency", "amount_basis", "policy_revision", "checked_at", "revision", "approve", "reject"];
  if (!serviceDecisionRecord(record) || !exact(raw, keys) || !Object.values(raw).every(v => typeof v === "string") || raw.mode !== "synthetic" || raw.receipt !== "current_decision_preparation" || raw.request !== record.id || raw.digest !== record.evidence.revision || raw.author !== record.author?.id || raw.requester !== record.requester?.id || raw.amount !== record.amounts[0].candidate || !validDecimal(raw.amount) || raw.currency !== record.amounts[0].currency || raw.amount_basis !== record.amounts[0].basis || raw.policy_revision !== String(record.policy_revision) || !timestamp(raw.checked_at) || !revision(raw.revision) || !["pending", "approved", "rejected", "superseded"].includes(String(raw.recorded_state)) || !decisionOutcomes.includes(raw.approve as DecisionOutcome) || !decisionOutcomes.includes(raw.reject as DecisionOutcome) || (raw.recorded_state !== "pending" && (raw.approve === "eligible" || raw.reject === "eligible"))) throw new DecisionError("invalid_preparation", "none");
  const { mode: _mode, receipt: _receipt, ...values } = raw;
  return Object.freeze({ ...values, family: decisionFamily(record)! }) as Preparation;
}
export function submissionOf(p: Preparation, action: DecisionAction, rationale: string, reason: string, tried: Attempt = attempt()): Submission {
  if (!((p.family === "financial_charge" && p.amount_basis === "rated_candidate_total") || (p.family === "erp_final" && p.amount_basis === "synthetic_reported_net_plus_tax_v1")) || p[action] !== "eligible" || !rationale.trim() || new TextEncoder().encode(rationale).length > 2000 || (action === "reject" ? !rejectionReasons.includes(reason as typeof rejectionReasons[number]) : reason !== "")) throw new DecisionError("invalid_decision", "none");
  return Object.freeze({ family: p.family, amount_basis: p.amount_basis, action, target: p.request, digest: p.digest, revision: p.revision, rationale, reason, amount: p.amount, currency: p.currency, attempt: Object.freeze({ ...tried }) });
}
function receiptOf(raw: Record<string, unknown>, s: Submission): Receipt {
  const f = raw.fields, state = s.action === "approve" ? "approved" : "rejected";
  if (raw.request_id !== s.attempt.request || raw.target_id !== s.target || raw.status !== "completed" || raw.business_state !== state || !revision(raw.result_revision) || typeof raw.replay !== "boolean" || raw.job_id !== undefined || !timestamp(raw.as_of) || !id(raw.operation_id) || Object.keys(raw).some(k => !["request_id", "status", "operation_id", "target_id", "result_revision", "business_state", "replay", "as_of", "fields"].includes(k)) || !exact(f, ["mode", "request", "digest", "recorded_state", "receipt", ...(s.family === "erp_final" ? ["stage"] : [])]) || (s.family === "erp_final" && f.stage !== "erp_final_approval") || f.mode !== "synthetic" || f.request !== s.target || f.digest !== s.digest || f.recorded_state !== state || f.receipt !== "historical_decision") throw new DecisionError("invalid_receipt", "unknown");
  return Object.freeze({ request: s.attempt.request, target: s.target, digest: s.digest, state, revision: raw.result_revision, replay: raw.replay });
}
const refusals: Record<string, number> = { permission_denied: 403, self_approval_denied: 403, unauthenticated: 401, csrf_refused: 403, reauthentication_required: 403, evidence_stale: 409, revision_conflict: 409, invalid_request: 422, prerequisite_failed: 422, rate_limited: 429, unsupported_capability: 501 };
async function bounded<T>(work: (signal: AbortSignal) => Promise<T>, effect: "none" | "unknown"): Promise<T> {
  const abort = new AbortController();
  let timer: ReturnType<typeof setTimeout>;
  const timeout = new Promise<never>((_, reject) => {
    timer = setTimeout(() => { reject(new DecisionError("request_timeout", effect)); abort.abort(); }, 30000);
  });
  try { return await Promise.race([work(abort.signal), timeout]); }
  finally { clearTimeout(timer!); }
}
export function decisionClient(session: Session, f: typeof fetch = fetch.bind(globalThis)) {
  const owner = Object.freeze({ ...session });
  async function send(family: DecisionFamily, kind: string, target: string, fields: Record<string, string>, tried: Attempt, submission?: Submission) {
    let current: Session | null;
    try { current = await bounded(signal => serviceClient((input, init) => f(input, { ...init, signal })).current(), "none"); } catch { throw new DecisionError("session_unavailable", "none"); }
    if (!sameSession(owner, current)) throw new DecisionError("session_changed", "none");
    let response: Response, raw: unknown;
    try {
      const result = await bounded(async signal => {
      const response = await f(`/api/v1/commands/${family}.${kind}`, {
        signal, method: "POST", credentials: "same-origin", cache: "no-store",
        headers: { "Content-Type": "application/json", "X-CSRF-Token": owner.csrf, "X-Request-Id": tried.request },
        body: JSON.stringify({ request_id: tried.request, target_id: target, ...(submission ? { idempotency_key: tried.key, expected_revision: submission.revision, rationale: submission.rationale } : {}), body: fields }),
      });
      const text = await response.text();
      if (text.length > 32768) throw new DecisionError("invalid_response", submission ? "unknown" : "none");
      return { response, raw: JSON.parse(text) as unknown };
      }, submission ? "unknown" : "none");
      response = result.response; raw = result.raw;
    } catch (error) { throw error instanceof DecisionError ? error : new DecisionError("service_unavailable", submission ? "unknown" : "none"); }
    if (!object(raw)) throw new DecisionError("invalid_response", submission ? "unknown" : "none");
    if (!response.ok) {
      const code = typeof raw.code === "string" && /^[a-z_]{1,80}$/.test(raw.code) ? raw.code : "service_unavailable";
      const operation = raw.request_id === tried.request && id(raw.operation_id) ? raw.operation_id : "";
      const none = raw.operation_id === undefined && raw.request_id === tried.request && refusals[code] === response.status;
      throw new DecisionError(code, submission && !none ? "unknown" : "none", operation);
    }
    if (response.status !== 200 || raw.request_id !== tried.request || raw.target_id !== target || raw.status !== "completed") throw new DecisionError("invalid_response", submission ? "unknown" : "none");
    return raw;
  }
  return {
    async prepare(record: ApprovalRecord): Promise<Preparation> {
      const tried = attempt();
      const family = decisionFamily(record);
      if (!family) throw new DecisionError("invalid_preparation", "none");
      const raw = await send(family, "prepare", record.id, { digest: record.evidence.revision ?? "" }, tried);
      if (!timestamp(raw.as_of) || !exact(raw, ["request_id", "status", "target_id", "result_revision", "replay", "as_of", "fields"]) || raw.replay !== false || raw.business_state !== undefined || raw.operation_id !== undefined || raw.job_id !== undefined || !revision(raw.result_revision)) throw new DecisionError("invalid_preparation", "none");
      const p = preparationOf(raw.fields, record);
      if (raw.result_revision !== p.revision) throw new DecisionError("invalid_preparation", "none");
      return p;
    },
    async submit(s: Submission): Promise<Receipt> {
      if (!((s.family === "financial_charge" && s.amount_basis === "rated_candidate_total") || (s.family === "erp_final" && s.amount_basis === "synthetic_reported_net_plus_tax_v1"))) throw new DecisionError("invalid_decision", "none");
      return receiptOf(await send(s.family, s.action, s.target, { digest: s.digest, ...(s.action === "reject" ? { reason: s.reason } : {}) }, s.attempt, s), s);
    },
  };
}
