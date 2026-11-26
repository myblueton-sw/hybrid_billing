import type { Session } from "../../../../../service/api";
import type { Attempt } from "../../../../../service/admin";
import { strictJSON } from "../../../../issuance/pages/approval-inbox/documents/json";
import { timestamp } from "../../../../issuance/pages/approval-inbox/documents/contract";
import { rationaleValid } from "../../../../issuance/pages/approval-inbox/documents/packet";
import { bytes, digest, freeze, id, object, revision, ResultError, type ReadEnvelope } from "../../../../issuance/pages/approval-inbox/results/contract";
import { bounded, checkSession, responseText } from "../../../../issuance/pages/approval-inbox/results/transport";
import { setupFields } from "./client";
import { setupDate, SetupError } from "./context";
import { cleanFailure } from "./mutation";
import { protectedScope, type ProtectedScope } from "./protected";

const invalid = (): never => { throw new SetupError("invalid_setup_response"); };
const refused = (): never => { throw new SetupError("invalid_setup_request"); };
const days = (value: string) => /^(?:0|[1-9][0-9]{0,2})$/.test(value) && Number(value) <= 365;
const actor = (value: string) => value === "" || id(value);
const instant = (value: string) => value === "" || timestamp(value);
const zone = (value: string) => value !== "" && value !== "Local" && bytes(value) <= 128;
const reasons = ["initial", "renegotiation", "correction"];
const retained = ["registration_unknown", "pending", "approval_unknown", "approved"];

function layers(state: string, observation: string, approver: string, pending: string, author: string, written: string, approved: string): boolean {
  if (![approver, pending].every(actor) || ![written, approved].every(instant) || [approver, pending].includes(author)) return false;
  const absent = observation === "absent", done = observation === "approved";
  if (absent !== (written === "") || done !== (approver !== "") || done !== (approved !== "") || done && pending !== "") return false;
  switch (state) {
    case "registration_unknown": return (absent || observation === "pending") && pending === "";
    case "pending": return observation === "pending" && pending === "";
    case "approval_unknown": return observation === "pending" && pending !== "" || done;
    case "approved": return done;
  }
  return false;
}

const termKeys = ["term", "term_digest", "term_revision", "net_days", "mode", "term_author", "term_approver", "term_pending_approver", "term_written_at", "term_approved_at"] as const;
export type PaymentTerm = Readonly<{ state: string; observation: string; id: string; digest: string; revision: string; netDays: string; mode: string; author: string; approver: string; pendingApprover: string; writtenAt: string; approvedAt: string }>;

const rowKeys = ["binding", "digest", "origin", "configuration", "term", "term_revision", "net_days", "mode", "zone", "from", "to", "reason", "state", "observation", "author", "approver", "pending_approver", "written_at", "approved_at"] as const;
export type TermBinding = Readonly<Record<typeof rowKeys[number], string>>;

export function termBindingOf(value: unknown): TermBinding {
  if (!object(value) || Object.keys(value).length !== rowKeys.length || rowKeys.some(key => typeof value[key] !== "string")) return invalid();
  const f = value as Record<string, string>;
  if (![f.binding, f.term, f.author].every(id) || !digest(f.digest) || !/^[1-9][0-9]{0,6}$/.test(f.term_revision) || !days(f.net_days) || f.mode !== "calendar" || !zone(f.zone)
    || !setupDate(f.from) || f.to !== "" && (!setupDate(f.to) || f.to < f.from) || !reasons.includes(f.reason) || !retained.includes(f.state)
    || !(f.origin === "configuration" && id(f.configuration) || f.origin === "profile" && f.configuration === "")
    || !layers(f.state, f.observation, f.approver, f.pending_approver, f.author, f.written_at, f.approved_at)) return invalid();
  return freeze({ ...f }) as TermBinding;
}

export type PaymentTerms = Readonly<{
  revision: string; at: string; scope: ProtectedScope; zone: string; term: PaymentTerm | null; snapshot: string; next: string; count: number;
  rows: readonly TermBinding[]; currentEligibility: "unchecked";
}>;

export function paymentTermsOf(e: ReadEnvelope, input: ProtectedScope, after = "", snapshot = ""): PaymentTerms {
  const scope = protectedScope(input), f = e.fields;
  const fixed = ["schema", "contract", "configuration", "digest", "zone", "current_eligibility", "term_state", "term_observation", ...termKeys, "snapshot", "count", "next"];
  const keys = Object.keys(f).filter(k => !fixed.includes(k)).sort();
  if (fixed.some(k => !Object.hasOwn(f, k)) || keys.length > 10 || keys.some((k, i) => k !== `row_${String(i).padStart(2, "0")}`)
    || f.schema !== "hb_issuance_setup_terms_v1" || f.contract !== scope.contract || f.configuration !== scope.configuration || f.digest !== scope.digest || !zone(f.zone)
    || f.current_eligibility !== "unchecked" || !digest(f.snapshot) || snapshot && snapshot !== f.snapshot || !/^(?:0|[1-9][0-9]?|100)$/.test(f.count)) return invalid();
  let term: PaymentTerm | null = null;
  if (f.term_state === "none") {
    if (f.term_observation !== "absent" || termKeys.some(key => f[key] !== "")) return invalid();
  } else {
    if (!retained.includes(f.term_state) || !id(f.term) || !digest(f.term_digest) || f.term_revision !== "1" || !days(f.net_days) || f.mode !== "calendar" || !id(f.term_author)
      || !layers(f.term_state, f.term_observation, f.term_approver, f.term_pending_approver, f.term_author, f.term_written_at, f.term_approved_at)) return invalid();
    term = freeze({ state: f.term_state, observation: f.term_observation, id: f.term, digest: f.term_digest, revision: f.term_revision, netDays: f.net_days, mode: f.mode,
      author: f.term_author, approver: f.term_approver, pendingApprover: f.term_pending_approver, writtenAt: f.term_written_at, approvedAt: f.term_approved_at });
  }
  const rows = keys.map(k => { try { return termBindingOf(strictJSON(f[k], 4096)); } catch { return invalid(); } });
  const count = Number(f.count);
  if (count < rows.length || !after && rows.length !== Math.min(count, 10) || rows.some((r, i) => r.binding <= (i ? rows[i - 1].binding : after))
    || f.next !== "" && (rows.length !== 10 || f.next !== rows.at(-1)?.binding)
    || !after && f.next !== (count > 10 ? rows.at(-1)?.binding : "") || count === 0 && f.next !== "") return invalid();
  return freeze({ revision: e.revision, at: e.at, scope, zone: f.zone, term, snapshot: f.snapshot, next: f.next, count, rows: freeze(rows), currentEligibility: "unchecked" as const });
}

type Rule = Readonly<{ family: "term" | "binding"; rationale: boolean; required: readonly string[]; optional: readonly string[]; states: readonly string[] }>;
export const termWriteRules = freeze({
  record_term: { family: "term", rationale: true, required: ["term", "net_days"], optional: [], states: ["pending", "approved"] },
  approve_term: { family: "term", rationale: false, required: ["term", "term_digest"], optional: [], states: ["approved"] },
  bind_term: { family: "binding", rationale: false, required: ["binding", "term", "term_digest", "from", "reason"], optional: ["to"], states: ["pending", "approved"] },
  approve_term_binding: { family: "binding", rationale: false, required: ["binding", "binding_digest"], optional: [], states: ["approved"] },
} as const satisfies Record<string, Rule>);
export type TermWriteOperation = keyof typeof termWriteRules;

export type TermWriteIntent = Readonly<{
  operation: TermWriteOperation; scope: ProtectedScope; args: Readonly<Record<string, string>>; rationale: string;
  actor: string; workspace: string; sessionCSRF: string; revision: string; attempt: Readonly<Attempt>;
}>;

function closed(operation: TermWriteOperation, a: Readonly<Record<string, string>>): boolean {
  switch (operation) {
    case "record_term": return id(a.term) && days(a.net_days);
    case "approve_term": return id(a.term) && digest(a.term_digest);
    case "bind_term": return id(a.binding) && id(a.term) && digest(a.term_digest) && setupDate(a.from) && (a.to === undefined || setupDate(a.to) && a.to >= a.from) && reasons.includes(a.reason);
    case "approve_term_binding": return id(a.binding) && digest(a.binding_digest);
  }
}

export function termWriteIntent(input: TermWriteIntent): TermWriteIntent {
  if (!Object.hasOwn(termWriteRules, input.operation)) return refused();
  const rule: Rule = termWriteRules[input.operation], scope = protectedScope(input.scope), args = { ...input.args }, keys = Object.keys(args);
  if (![input.actor, input.workspace, input.attempt?.request, input.attempt?.key].every(id) || typeof input.sessionCSRF !== "string" || !input.sessionCSRF || input.sessionCSRF.length > 4096
    || !revision(input.revision) || (rule.rationale ? !rationaleValid(input.rationale) : input.rationale !== "")
    || rule.required.some(k => typeof args[k] !== "string") || keys.some(k => typeof args[k] !== "string" || !rule.required.includes(k) && !rule.optional.includes(k))
    || !closed(input.operation, args)) return refused();
  return freeze({ ...input, scope, args: freeze(args), attempt: freeze({ ...input.attempt }) });
}

export type TermWriteReceipt = Readonly<{ operation: TermWriteOperation; target: string; digest: string; state: string; revision: string; operationId: string; replay: boolean }>;
const schemas = { term: ["hb_issuance_term_receipt_v1", "term", "term_digest"], binding: ["hb_issuance_term_binding_receipt_v1", "binding", "binding_digest"] } as const;

export function termWriteClient(session: Session, fetcher: typeof fetch = fetch.bind(globalThis)) {
  const owner = freeze({ ...session });
  return async (input: TermWriteIntent): Promise<TermWriteReceipt> => {
    const intent = termWriteIntent(input), rule: Rule = termWriteRules[intent.operation];
    if (intent.actor !== owner.user || intent.workspace !== owner.workspace || intent.sessionCSRF !== owner.csrf) throw new SetupError("session_changed");
    try { await checkSession(owner, fetcher); } catch (error) { throw new SetupError(error instanceof ResultError ? error.code : "session_unavailable"); }
    const envelope: Record<string, unknown> = { request_id: intent.attempt.request, target_id: intent.scope.contract, idempotency_key: intent.attempt.key, expected_revision: intent.revision,
      body: { configuration: intent.scope.configuration, digest: intent.scope.digest, ...intent.args } };
    if (rule.rationale) envelope.rationale = intent.rationale;
    let response: Response, raw: unknown;
    try {
      const result = await bounded(async signal => {
        const response = await fetcher(`/api/v1/commands/issuance_setup.${intent.operation}`, {
          signal, method: "POST", credentials: "same-origin", cache: "no-store",
          headers: { "Content-Type": "application/json", "X-CSRF-Token": owner.csrf, "X-Request-Id": intent.attempt.request }, body: JSON.stringify(envelope),
        });
        return { response, raw: strictJSON(await responseText(response), 125000) };
      }, "unknown");
      response = result.response; raw = result.raw;
    } catch (error) { throw new SetupError(error instanceof ResultError ? error.code : "service_unavailable", "unknown"); }
    if (!object(raw)) throw new SetupError("invalid_setup_response", "unknown");
    const operationId = raw.request_id === intent.attempt.request && id(raw.operation_id) ? raw.operation_id : "";
    if (!response.ok) throw new SetupError(raw.request_id === intent.attempt.request && typeof raw.code === "string" && /^[a-z_]{1,80}$/.test(raw.code) ? raw.code : "service_unavailable", cleanFailure(raw, response.status, intent.attempt.request) ? "none" : "unknown", operationId);
    try {
      const keys = ["request_id", "status", "operation_id", "target_id", "result_revision", "business_state", "replay", "as_of", "fields"];
      if (Object.keys(raw).length !== keys.length || keys.some(k => !Object.hasOwn(raw, k)) || response.status !== 200 || raw.request_id !== intent.attempt.request || raw.target_id !== intent.scope.contract
        || raw.status !== "completed" || !operationId || !revision(raw.result_revision) || typeof raw.replay !== "boolean" || !timestamp(raw.as_of)) throw new Error();
      const f = setupFields(raw.fields), [schema, name, digestName] = schemas[rule.family];
      const expected = ["schema", "contract", "configuration", "current_eligibility", name, digestName, "state"];
      const sent = name === "term" ? intent.args.term_digest : intent.args.binding_digest;
      if (Object.keys(f).length !== expected.length || expected.some(k => !Object.hasOwn(f, k)) || f.schema !== schema || f.contract !== intent.scope.contract || f.configuration !== intent.scope.configuration
        || f.current_eligibility !== "unchecked" || f[name] !== intent.args[name] || !digest(f[digestName]) || !rule.states.includes(f.state) || raw.business_state !== f.state
        || rule.family === "term" && intent.operation === "approve_term" && f[digestName] !== sent
        || intent.operation === "approve_term_binding" && f[digestName] !== sent) throw new Error();
      return freeze({ operation: intent.operation, target: f[name], digest: f[digestName], state: f.state, revision: raw.result_revision, operationId, replay: raw.replay });
    } catch { throw new SetupError("invalid_setup_receipt", "unknown", operationId); }
  };
}
