import type { Session } from "../../../../../service/api";
import type { Attempt } from "../../../../../service/admin";
import { strictJSON } from "../../../../issuance/pages/approval-inbox/documents/json";
import { timestamp } from "../../../../issuance/pages/approval-inbox/documents/contract";
import { rationaleValid } from "../../../../issuance/pages/approval-inbox/documents/packet";
import { digest, freeze, id, object, revision, ResultError } from "../../../../issuance/pages/approval-inbox/results/contract";
import { bounded, checkSession, responseText } from "../../../../issuance/pages/approval-inbox/results/transport";
import { setupFields } from "./client";
import { SetupError } from "./context";
import { cleanFailure } from "./mutation";
import { protectedFields, protectedScope, type ProtectedScope } from "./protected";

const refused = (): never => { throw new SetupError("invalid_setup_request"); };
const whole = (value: string, least: number, most: number) => /^(?:0|[1-9][0-9]{0,8})$/.test(value) && Number(value) >= least && Number(value) <= most;
const one = (value: string, allowed: readonly string[]) => allowed.includes(value);

type Rule = Readonly<{ family: "rules" | "channel" | "change"; rationale: boolean; required: readonly string[]; optional: readonly string[]; states: readonly string[] }>;
const moduleStates = ["requested", "confirmed", "scheduled", "rejected", "lapsed"];
export const protectedWriteRules: Readonly<Record<string, Rule>> = freeze({
  propose_protected_rules: { family: "rules", rationale: true, required: ["rules", "approvals", "validity_seconds", "max_lead_seconds", "channel_age_seconds"], optional: [], states: ["proposed", "confirmed"] },
  confirm_protected_rules: { family: "rules", rationale: true, required: ["rules", "rules_digest"], optional: [], states: ["confirmed"] },
  register_channel: { family: "channel", rationale: true, required: ["channel", "subject_kind", "kind", "reference", "attestation"], optional: [], states: ["registered", "admission_unknown", "admitted"] },
  admit_channel: { family: "channel", rationale: false, required: ["channel", "channel_digest", "attestation"], optional: [], states: ["admitted"] },
  request_change: { family: "change", rationale: true, required: ["change", "field", "reference", "channel", "channel_digest", "provenance", "instruction", "effective", "attestation"], optional: ["instruction_channel"], states: moduleStates },
  verify_change: { family: "change", rationale: false, required: ["change", "change_digest", "outcome", "attestation"], optional: [], states: ["confirmed", "rejected", "scheduled", "lapsed"] },
  decide_change: { family: "change", rationale: false, required: ["change", "change_digest", "verdict"], optional: ["reason"], states: ["confirmed", "scheduled", "rejected", "lapsed"] },
});
export type ProtectedWriteOperation = keyof typeof protectedWriteRules & string;

export type ProtectedWriteIntent = Readonly<{
  operation: ProtectedWriteOperation; scope: ProtectedScope; args: Readonly<Record<string, string>>; rationale: string;
  actor: string; workspace: string; sessionCSRF: string; revision: string; attempt: Readonly<Attempt>;
}>;

function closed(operation: string, a: Readonly<Record<string, string>>): boolean {
  switch (operation) {
    case "propose_protected_rules":
      return id(a.rules) && whole(a.approvals, 1, 3) && whole(a.validity_seconds, 3600, 2592000) && whole(a.max_lead_seconds, 3600, 31622400) && whole(a.channel_age_seconds, 86400, 7776000);
    case "confirm_protected_rules":
      return id(a.rules) && digest(a.rules_digest);
    case "register_channel":
      return id(a.channel) && id(a.attestation) && one(a.subject_kind, ["customer", "entity", "document_kind"]) && one(a.kind, ["email", "phone", "portal", "postal"])
        && new RegExp(`^synthetic_channel_${a.subject_kind}_[a-z]$`).test(a.reference);
    case "admit_channel":
      return id(a.channel) && digest(a.channel_digest) && id(a.attestation);
    case "request_change":
      return id(a.change) && id(a.attestation) && id(a.channel) && digest(a.channel_digest) && one(a.field, protectedFields) && /^synthetic_[a-z]{1,32}_[a-z]$/.test(a.reference)
        && one(a.provenance, ["staff_entry", "customer_document", "customer_portal"]) && one(a.instruction, ["in_person", "registered_channel", "unregistered_sender"])
        && (a.instruction === "registered_channel") === (a.instruction_channel !== undefined) && (a.instruction_channel === undefined || id(a.instruction_channel) && a.instruction_channel !== a.channel)
        && timestamp(a.effective);
    case "verify_change":
      return id(a.change) && digest(a.change_digest) && id(a.attestation) && one(a.outcome, ["confirmed", "denied"]);
    case "decide_change":
      return id(a.change) && digest(a.change_digest) && (a.verdict === "approve" && a.reason === undefined || a.verdict === "reject" && one(a.reason ?? "", ["incorrect_details", "missing_evidence", "suspected_fraud", "other"]));
  }
  return false;
}

export function protectedWriteIntent(input: ProtectedWriteIntent): ProtectedWriteIntent {
  const rule = Object.hasOwn(protectedWriteRules, input.operation) ? protectedWriteRules[input.operation] : undefined;
  if (!rule) return refused();
  const scope = protectedScope(input.scope), args = { ...input.args }, keys = Object.keys(args);
  if (![input.actor, input.workspace, input.attempt?.request, input.attempt?.key].every(id) || typeof input.sessionCSRF !== "string" || !input.sessionCSRF || input.sessionCSRF.length > 4096
    || !revision(input.revision) || (rule.rationale ? !rationaleValid(input.rationale) : input.rationale !== "")
    || rule.required.some(k => typeof args[k] !== "string") || keys.some(k => typeof args[k] !== "string" || !rule.required.includes(k) && !rule.optional.includes(k))
    || !closed(input.operation, args)) return refused();
  return freeze({ ...input, scope, args: freeze(args), attempt: freeze({ ...input.attempt }) });
}

export type ProtectedWriteReceipt = Readonly<{ operation: ProtectedWriteOperation; target: string; digest: string; state: string; phase: string; revision: string; operationId: string; replay: boolean }>;
const schemas = { rules: ["hb_issuance_protected_rules_receipt_v1", "rules", "rules_digest"], channel: ["hb_issuance_protected_channel_receipt_v1", "channel", "channel_digest"], change: ["hb_issuance_protected_change_receipt_v1", "change", "change_digest"] } as const;

export function protectedWriteClient(session: Session, fetcher: typeof fetch = fetch.bind(globalThis)) {
  const owner = freeze({ ...session });
  return async (input: ProtectedWriteIntent): Promise<ProtectedWriteReceipt> => {
    const intent = protectedWriteIntent(input), rule = protectedWriteRules[intent.operation];
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
      const expected = ["schema", "contract", "configuration", "current_eligibility", name, digestName, "state", ...(rule.family === "change" ? ["phase"] : [])];
      if (Object.keys(f).length !== expected.length || expected.some(k => !Object.hasOwn(f, k)) || f.schema !== schema || f.contract !== intent.scope.contract || f.configuration !== intent.scope.configuration
        || f.current_eligibility !== "unchecked" || f[name] !== intent.args[name] || !digest(f[digestName]) || !rule.states.includes(f.state) || raw.business_state !== f.state
        || intent.args[digestName] !== undefined && f[digestName] !== intent.args[digestName]
        || rule.family === "change" && !["recorded", "request_unknown", "verification_unknown", "decision_unknown"].includes(f.phase)) throw new Error();
      return freeze({ operation: intent.operation, target: f[name], digest: f[digestName], state: f.state, phase: f.phase ?? "", revision: raw.result_revision, operationId, replay: raw.replay });
    } catch { throw new SetupError("invalid_setup_receipt", "unknown", operationId); }
  };
}
