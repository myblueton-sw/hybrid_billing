import type { Session } from "../../../../../service/api";
import type { Attempt } from "../../../../../service/admin";
import { strictJSON } from "../../../../issuance/pages/approval-inbox/documents/json";
import { timestamp } from "../../../../issuance/pages/approval-inbox/documents/contract";
import { bytes, digest, freeze, id, object, revision, ResultError, type ReadEnvelope } from "../../../../issuance/pages/approval-inbox/results/contract";
import { bounded, checkSession, responseText } from "../../../../issuance/pages/approval-inbox/results/transport";
import { setupFields } from "./client";
import { SetupError } from "./context";
import { cleanFailure } from "./mutation";
import { protectedScope, type ProtectedScope } from "./protected";

const invalid = (): never => { throw new SetupError("invalid_setup_response"); };
const refused = (): never => { throw new SetupError("invalid_setup_request"); };

export const attestationPurposes = ["channel_registration", "channel_admission", "change_instruction", "change_verification"] as const;
export type AttestationPurpose = typeof attestationPurposes[number];
const bound = (purpose: AttestationPurpose) => purpose === "channel_admission" || purpose === "change_verification";
const states = ["unconsumed", "consumed", "withdrawn"];

export function attestationText(value: string, limit: number, required: boolean): string {
  const text = value.replace(/\r\n?/g, "\n");
  if (bytes(text) > limit || required && text.trim() === "" || text.includes("\uFFFD") || /[\u0000-\u0008\u000B-\u001F\u007F-\u009F]/.test(text) || /\p{Cs}/u.test(text)) return refused();
  return text;
}

const rowKeys = ["attestation", "digest", "purpose", "target", "actor", "recorded_at", "statement", "locator", "state", "assurance"] as const;
export type Attestation = Readonly<Record<typeof rowKeys[number], string>>;
export type AttestationPage = Readonly<{ revision: string; at: string; scope: ProtectedScope; snapshot: string; next: string; count: number; rows: readonly Attestation[]; currentEligibility: "unchecked" }>;

export function attestationOf(value: unknown): Attestation {
  if (!object(value) || Object.keys(value).length !== rowKeys.length || rowKeys.some(key => typeof value[key] !== "string")) return invalid();
  const f = value as Record<string, string>;
  if (![f.attestation, f.target, f.actor].every(id) || !digest(f.digest) || !attestationPurposes.includes(f.purpose as AttestationPurpose) || !timestamp(f.recorded_at)
    || !states.includes(f.state) || f.assurance !== "human_attestation" || f.statement === "" || bytes(f.statement) > 2000 || bytes(f.locator) > 512) return invalid();
  return freeze({ ...f }) as Attestation;
}

export function attestationsOf(e: ReadEnvelope, input: ProtectedScope, after = "", snapshot = ""): AttestationPage {
  const scope = protectedScope(input), f = e.fields;
  const fixed = ["schema", "contract", "configuration", "digest", "current_eligibility", "snapshot", "count", "next"];
  const keys = Object.keys(f).filter(k => !fixed.includes(k)).sort();
  if (fixed.some(k => !Object.hasOwn(f, k)) || keys.length > 10 || keys.some((k, i) => k !== `row_${String(i).padStart(2, "0")}`)
    || f.schema !== "hb_issuance_setup_attestation_page_v1" || f.contract !== scope.contract || f.configuration !== scope.configuration || f.digest !== scope.digest
    || f.current_eligibility !== "unchecked" || !digest(f.snapshot) || snapshot && snapshot !== f.snapshot || !/^(?:0|[1-9][0-9]{0,2}|1000)$/.test(f.count)) return invalid();
  const rows = keys.map(k => { try { return attestationOf(strictJSON(f[k], 4096)); } catch { return invalid(); } });
  const count = Number(f.count);
  if (count < rows.length || !after && rows.length !== Math.min(count, 10) || rows.some((r, i) => r.attestation <= (i ? rows[i - 1].attestation : after))
    || f.next !== "" && (rows.length !== 10 || f.next !== rows.at(-1)?.attestation)
    || !after && f.next !== (count > 10 ? rows.at(-1)?.attestation : "") || count === 0 && f.next !== "") return invalid();
  return freeze({ revision: e.revision, at: e.at, scope, snapshot: f.snapshot, next: f.next, count, rows: freeze(rows), currentEligibility: "unchecked" as const });
}

const referenceUses = ["document_recipient", "document_template", "payment_account", "channel_customer", "channel_document_kind", "channel_entity", "unregistered_sender"] as const;
export type SetupReference = Readonly<{ code: string; use: typeof referenceUses[number]; applicability: "synthetic_only" }>;
export type SetupReferences = Readonly<{ revision: string; at: string; scope: ProtectedScope; entries: readonly SetupReference[]; currentEligibility: "unchecked" }>;

export function referencesOf(e: ReadEnvelope, input: ProtectedScope): SetupReferences {
  const scope = protectedScope(input), f = e.fields;
  const fixed = ["schema", "contract", "configuration", "digest", "current_eligibility"];
  const keys = Object.keys(f).filter(k => !fixed.includes(k)).sort();
  if (fixed.some(k => !Object.hasOwn(f, k)) || keys.length === 0 || keys.length > 32 || keys.some((k, i) => k !== `reference_${String(i).padStart(2, "0")}`)
    || f.schema !== "hb_issuance_setup_references_v1" || f.contract !== scope.contract || f.configuration !== scope.configuration || f.digest !== scope.digest || f.current_eligibility !== "unchecked") return invalid();
  const entries = keys.map(k => {
    let raw: unknown;
    try { raw = strictJSON(f[k], 4096); } catch { return invalid(); }
    if (!object(raw) || Object.keys(raw).length !== 3 || typeof raw.code !== "string" || !/^synthetic_[a-z_]{1,64}$/.test(raw.code)
      || !referenceUses.includes(raw.use as SetupReference["use"]) || raw.applicability !== "synthetic_only") return invalid();
    return freeze({ code: raw.code, use: raw.use as SetupReference["use"], applicability: "synthetic_only" as const });
  });
  if (new Set(entries.map(v => v.code)).size !== entries.length) return invalid();
  return freeze({ revision: e.revision, at: e.at, scope, entries: freeze(entries), currentEligibility: "unchecked" as const });
}

export type AttestationIntent = Readonly<{
  action: "attest" | "withdraw"; purpose: AttestationPurpose; scope: ProtectedScope; attestation: string; target: string; targetDigest: string;
  statement: string; locator: string; actor: string; workspace: string; sessionCSRF: string; revision: string; attempt: Readonly<Attempt>;
}>;

export function attestationIntent(input: AttestationIntent): AttestationIntent {
  const scope = protectedScope(input.scope), withdraw = input.action === "withdraw";
  if (!["attest", "withdraw"].includes(input.action) || !attestationPurposes.includes(input.purpose) || ![input.attestation, input.actor, input.workspace, input.attempt?.request, input.attempt?.key].every(id)
    || typeof input.sessionCSRF !== "string" || !input.sessionCSRF || input.sessionCSRF.length > 4096 || !revision(input.revision)) return refused();
  if (withdraw) {
    if (input.target !== "" || input.targetDigest !== "" || input.statement !== "" || input.locator !== "") return refused();
    return freeze({ ...input, scope, attempt: { ...input.attempt } });
  }
  if (!id(input.target) || bound(input.purpose) !== digest(input.targetDigest) || !bound(input.purpose) && input.targetDigest !== "") return refused();
  return freeze({ ...input, scope, statement: attestationText(input.statement, 2000, true), locator: attestationText(input.locator, 512, false), attempt: { ...input.attempt } });
}

export type AttestationReceipt = Readonly<{ attestation: string; digest: string; purpose: AttestationPurpose; target: string; actor: string; recordedAt: string; state: string; revision: string; operation: string; replay: boolean }>;
const receiptKeys = ["schema", "contract", "configuration", "attestation", "purpose", "target", "actor", "recorded_at", "digest", "state", "assurance"];

export function attestationClient(session: Session, fetcher: typeof fetch = fetch.bind(globalThis)) {
  const owner = freeze({ ...session });
  return async (input: AttestationIntent): Promise<AttestationReceipt> => {
    const intent = attestationIntent(input);
    if (intent.actor !== owner.user || intent.workspace !== owner.workspace || intent.sessionCSRF !== owner.csrf) throw new SetupError("session_changed");
    try { await checkSession(owner, fetcher); } catch (error) { throw new SetupError(error instanceof ResultError ? error.code : "session_unavailable"); }
    const args: Record<string, string> = { configuration: intent.scope.configuration, digest: intent.scope.digest, attestation: intent.attestation };
    if (intent.action === "attest") {
      Object.assign(args, { target: intent.target, statement: intent.statement });
      if (intent.locator !== "") args.locator = intent.locator;
      if (intent.targetDigest !== "") args.target_digest = intent.targetDigest;
    }
    let response: Response, raw: unknown;
    try {
      const result = await bounded(async signal => {
        const response = await fetcher(`/api/v1/commands/issuance_setup.${intent.action}_${intent.purpose}`, {
          signal, method: "POST", credentials: "same-origin", cache: "no-store",
          headers: { "Content-Type": "application/json", "X-CSRF-Token": owner.csrf, "X-Request-Id": intent.attempt.request },
          body: JSON.stringify({ request_id: intent.attempt.request, target_id: intent.scope.contract, idempotency_key: intent.attempt.key, expected_revision: intent.revision, body: args }),
        });
        return { response, raw: strictJSON(await responseText(response), 125000) };
      }, "unknown");
      response = result.response; raw = result.raw;
    } catch (error) { throw new SetupError(error instanceof ResultError ? error.code : "service_unavailable", "unknown"); }
    if (!object(raw)) throw new SetupError("invalid_setup_response", "unknown");
    const operation = raw.request_id === intent.attempt.request && id(raw.operation_id) ? raw.operation_id : "";
    if (!response.ok) throw new SetupError(raw.request_id === intent.attempt.request && typeof raw.code === "string" && /^[a-z_]{1,80}$/.test(raw.code) ? raw.code : "service_unavailable", cleanFailure(raw, response.status, intent.attempt.request) ? "none" : "unknown", operation);
    try {
      const keys = ["request_id", "status", "operation_id", "target_id", "result_revision", "business_state", "replay", "as_of", "fields"];
      if (Object.keys(raw).length !== keys.length || keys.some(k => !Object.hasOwn(raw, k)) || response.status !== 200 || raw.request_id !== intent.attempt.request || raw.target_id !== intent.scope.contract
        || raw.status !== "completed" || !operation || !revision(raw.result_revision) || typeof raw.replay !== "boolean" || !timestamp(raw.as_of)
        || raw.business_state !== (intent.action === "attest" ? "recorded" : "withdrawn")) throw new Error();
      const f = setupFields(raw.fields);
      if (Object.keys(f).length !== receiptKeys.length || receiptKeys.some(k => !Object.hasOwn(f, k)) || f.schema !== "hb_issuance_setup_attestation_v1" || f.contract !== intent.scope.contract
        || f.configuration !== intent.scope.configuration || f.attestation !== intent.attestation || f.purpose !== intent.purpose || f.actor !== intent.actor || !id(f.target) || !digest(f.digest)
        || !timestamp(f.recorded_at) || !states.includes(f.state) || f.assurance !== "human_attestation"
        || intent.action === "attest" && f.target !== intent.target || intent.action === "withdraw" && f.state !== "withdrawn") throw new Error();
      return freeze({ attestation: f.attestation, digest: f.digest, purpose: intent.purpose, target: f.target, actor: f.actor, recordedAt: f.recorded_at, state: f.state, revision: raw.result_revision, operation, replay: raw.replay });
    } catch { throw new SetupError("invalid_setup_receipt", "unknown", operation); }
  };
}
