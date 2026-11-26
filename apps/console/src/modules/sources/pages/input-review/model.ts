import { AdminError, type Attempt, type Outcome } from "../../../../service/admin";

export type Review = { revision: string; fields: Record<string, string> };
export type Decision = "confirm_publish" | "reject";
export type Intent = { attempt: Attempt; review: Review; decision: Decision; reason: string };
const states = ["awaiting_confirmation", "blocked", "eligible", "rejected"];
const revisionValid = (value: string) => /^(0|[1-9][0-9]*)$/.test(value) && Number(value) <= 4294967295;
const instantValid = (value: string) => !value.startsWith("0000") && /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{1,9})?Z$/.test(value) && Number.isFinite(Date.parse(value)) && new Date(value).toISOString().slice(0, 19) === value.slice(0, 19);
const required = ["review", "digest", "workspace", "provider", "account", "period", "method", "recorded_verdict", "compared_at", "document", "document_revision", "document_currency", "document_total", "document_digest"];

export function reviewOf(out: Outcome, target: string, workspace: string): Review {
  const f = out.fields;
  if (out.status !== "completed" || !out.revision || f.mode !== "synthetic" || f.review !== target || f.workspace !== workspace ||
    required.some(k => !f[k]) || !states.includes(f.recorded_state) || !["match", "difference"].includes(f.recorded_verdict) ||
    !["matched", "changed", "unavailable"].includes(f.current_evidence) ||
    !["eligible", "not_eligible", "unverified"].includes(f.current_eligibility) ||
    !["allowed", "denied", "unavailable"].includes(f.confirmation_permission) ||
    !["clear", "blocked"].includes(f.confirmation_hold) || !revisionValid(f.scope_revision) || !/^[1-9][0-9]{0,5}$/.test(f.document_revision) || !instantValid(f.compared_at) ||
    !/^[a-f0-9]{64}$/.test(f.digest) || !/^[a-f0-9]{64}$/.test(f.document_digest) ||
    !/^[A-Z]{3}$/.test(f.document_currency) || !/^-?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$/.test(f.document_total) ||
    (["eligible", "rejected"].includes(f.recorded_state) && ["decided_by", "decision_reason", "decided_at", "recorded_scope_revision"].some(k => !f[k]))) {
    throw new AdminError("unavailable");
  }
  if ((f.recorded_state === "awaiting_confirmation" || f.recorded_state === "eligible") && f.recorded_verdict !== "match" ||
    f.recorded_state === "blocked" && f.recorded_verdict !== "difference" ||
    f.current_evidence !== "matched" && f.current_eligibility !== "unverified" ||
    f.current_eligibility === "eligible" && (f.recorded_state !== "eligible" || f.current_evidence !== "matched" || f.scope_revision !== f.recorded_scope_revision) ||
    ["eligible", "rejected"].includes(f.recorded_state) && (!instantValid(f.decided_at) || !revisionValid(f.recorded_scope_revision))) throw new AdminError("unavailable");
  if (!["eligible", "rejected"].includes(f.recorded_state) &&
    ["decided_by", "decision_reason", "decided_at", "recorded_scope_revision"].some(k => Object.hasOwn(f, k))) throw new AdminError("unavailable");
  return { revision: out.revision, fields: { ...f } };
}

export function reasonValid(reason: string): boolean {
  return reason.trim().length > 0 && new TextEncoder().encode(reason).length <= 1024 && !/[\u0000-\u001f\u007f]/.test(reason);
}

export function allowed(review: Review, decision: Decision): boolean {
  const f = review.fields;
  return f.confirmation_permission === "allowed" && f.current_evidence === "matched" &&
    (decision === "confirm_publish" ? f.recorded_state === "awaiting_confirmation" && f.confirmation_hold === "clear" : ["awaiting_confirmation", "blocked"].includes(f.recorded_state));
}

export function uncertain(error: unknown, wasUncertain: boolean): boolean {
  return wasUncertain || !(error instanceof AdminError) || !error.code || !!error.operation ||
    ["internal_error", "idempotency_key_expired", "idempotency_conflict"].includes(error.code);
}

export function freshAttempt(): Attempt {
  return { request: "s06-" + crypto.randomUUID(), key: "s06-" + crypto.randomUUID() };
}
