import { strictJSON } from "../../../../issuance/pages/approval-inbox/documents/json";
import { timestamp } from "../../../../issuance/pages/approval-inbox/documents/contract";
import { bytes, digest, freeze, id, object, type ReadEnvelope } from "../../../../issuance/pages/approval-inbox/results/contract";
import { SetupError } from "./context";

const invalid = (): never => { throw new SetupError("invalid_setup_response"); };
const count = (value: string) => /^(?:0|[1-9][0-9]?|100)$/.test(value);
const actor = (value: string) => value === "" || id(value);
const instant = (value: string) => value === "" || timestamp(value);
const seconds = (value: string) => /^[1-9][0-9]{0,8}$/.test(value);

export const protectedFields = ["document_recipient", "document_template", "payment_account"] as const;
export type ProtectedField = typeof protectedFields[number];
export type ProtectedScope = Readonly<{ contract: string; configuration: string; digest: string }>;
const groups = [["recipient", "customer"], ["template", "document_kind"], ["account", "entity"]] as const;
const groupKeys = ["subject_kind", "changes", "other_provenance", "unowned_module", "latest_scheduled", "latest_scheduled_effective"] as const;
const ruleKeys = ["rules_id", "rules_digest", "rules_approvals", "rules_validity_seconds", "rules_max_lead_seconds", "rules_channel_age_seconds", "rules_proposer", "rules_confirmer", "rules_pending_confirmer", "rules_proposed_at", "rules_confirmed_at"] as const;
const summaryKeys = ["schema", "contract", "configuration", "digest", "current_eligibility", "channels", "channels_other_provenance", "channels_unowned_module", "rules_state", "rules_observation", ...ruleKeys, ...groups.flatMap(([prefix]) => groupKeys.map(key => `${prefix}_${key}`))];
const ruleStates: Record<string, readonly string[]> = { none: ["absent", "unowned"], registration_unknown: ["absent", "pending"], proposed: ["pending"], confirmation_unknown: ["pending", "active"], confirmed: ["active"] };

export type FieldSummary = Readonly<{ subjectKind: string; changes: number; otherProvenance: number; unownedModule: number; latestScheduled: string; latestScheduledEffective: string }>;
export type ProtectedRules = Readonly<{
  state: string; observation: string; id: string; digest: string; approvals: string; validitySeconds: string; maxLeadSeconds: string;
  channelAgeSeconds: string; proposer: string; confirmer: string; pendingConfirmer: string; proposedAt: string; confirmedAt: string;
}>;
export type ProtectedSummary = Readonly<{
  revision: string; at: string; scope: ProtectedScope; rules: ProtectedRules; channels: number; channelsOtherProvenance: number;
  channelsUnownedModule: number; fields: Readonly<Record<ProtectedField, FieldSummary>>; currentEligibility: "unchecked";
}>;

export function protectedScope(value: ProtectedScope): ProtectedScope {
  if (!object(value) || !id(value.contract) || !id(value.configuration) || !digest(value.digest)) return invalid();
  return freeze({ contract: value.contract, configuration: value.configuration, digest: value.digest });
}

function scoped(f: Readonly<Record<string, string>>, scope: ProtectedScope, schema: string): void {
  if (f.schema !== schema || f.contract !== scope.contract || f.configuration !== scope.configuration || f.digest !== scope.digest || f.current_eligibility !== "unchecked") invalid();
}

export function protectedSummaryOf(e: ReadEnvelope, input: ProtectedScope): ProtectedSummary {
  const scope = protectedScope(input), f = e.fields;
  if (Object.keys(f).length !== summaryKeys.length || summaryKeys.some(key => !Object.hasOwn(f, key))) return invalid();
  scoped(f, scope, "hb_issuance_protected_setup_v1");
  const observations = ruleStates[f.rules_state];
  if (!observations?.includes(f.rules_observation) || ![f.channels, f.channels_other_provenance, f.channels_unowned_module].every(count)) return invalid();
  const owned = f.rules_state !== "none", active = f.rules_observation === "active", pinned = f.rules_state === "confirmation_unknown" || f.rules_state === "confirmed";
  if (!owned) { if (ruleKeys.some(key => f[key] !== "")) return invalid(); }
  else if (!id(f.rules_id) || !digest(f.rules_digest) || !/^[1-3]$/.test(f.rules_approvals) || !id(f.rules_proposer)
    || ![f.rules_validity_seconds, f.rules_max_lead_seconds, f.rules_channel_age_seconds].every(seconds)
    || (f.rules_proposed_at !== "") !== (f.rules_observation !== "absent") || !instant(f.rules_proposed_at)
    || active !== id(f.rules_confirmer) || active !== timestamp(f.rules_confirmed_at) || !active && (f.rules_confirmer !== "" || f.rules_confirmed_at !== "")
    || (pinned && !active) !== id(f.rules_pending_confirmer) || !(pinned && !active) && f.rules_pending_confirmer !== ""
    || [f.rules_confirmer, f.rules_pending_confirmer].includes(f.rules_proposer)) return invalid();
  const fields = {} as Record<ProtectedField, FieldSummary>;
  groups.forEach(([prefix, kind], index) => {
    const value = (key: typeof groupKeys[number]) => f[`${prefix}_${key}`];
    const latest = value("latest_scheduled"), effective = value("latest_scheduled_effective");
    if (value("subject_kind") !== kind || ![value("changes"), value("other_provenance"), value("unowned_module")].every(count)
      || (latest === "") !== (effective === "") || latest !== "" && (!id(latest) || !timestamp(effective) || value("changes") === "0")) return invalid();
    fields[protectedFields[index]] = freeze({ subjectKind: kind, changes: Number(value("changes")), otherProvenance: Number(value("other_provenance")), unownedModule: Number(value("unowned_module")), latestScheduled: latest, latestScheduledEffective: effective });
  });
  return freeze({
    revision: e.revision, at: e.at, scope, channels: Number(f.channels), channelsOtherProvenance: Number(f.channels_other_provenance),
    channelsUnownedModule: Number(f.channels_unowned_module), fields: freeze(fields), currentEligibility: "unchecked" as const,
    rules: freeze({ state: f.rules_state, observation: f.rules_observation, id: f.rules_id, digest: f.rules_digest, approvals: f.rules_approvals,
      validitySeconds: f.rules_validity_seconds, maxLeadSeconds: f.rules_max_lead_seconds, channelAgeSeconds: f.rules_channel_age_seconds,
      proposer: f.rules_proposer, confirmer: f.rules_confirmer, pendingConfirmer: f.rules_pending_confirmer, proposedAt: f.rules_proposed_at, confirmedAt: f.rules_confirmed_at }),
  });
}

const channelKeys = ["channel", "digest", "subject_kind", "kind", "state", "observation", "registrar", "admitter", "pending_admitter", "registered_at", "admitted_at", "retired_by", "retired_at", "rules", "request_not_before"] as const;
export type ProtectedChannel = Readonly<Record<typeof channelKeys[number], string>>;

function text(value: unknown, keys: readonly string[]): Record<string, string> {
  if (!object(value) || Object.keys(value).length !== keys.length || keys.some(key => typeof value[key] !== "string" || bytes(value[key] as string) > 4096)) return invalid();
  return value as Record<string, string>;
}

export function protectedChannelOf(value: unknown): ProtectedChannel {
  const f = text(value, channelKeys);
  const retired = f.observation === "retired", absent = f.observation === "absent", admitted = f.admitter !== "";
  if (!id(f.channel) || !digest(f.digest) || !["customer", "document_kind", "entity"].includes(f.subject_kind) || !["email", "phone", "portal", "postal"].includes(f.kind)
    || !["current", "other"].includes(f.rules) || !id(f.registrar) || !actor(f.admitter) || !actor(f.pending_admitter) || !actor(f.retired_by)
    || ![f.registered_at, f.admitted_at, f.retired_at, f.request_not_before].every(instant)
    || [f.admitter, f.pending_admitter].includes(f.registrar) || admitted && f.pending_admitter !== ""
    || admitted !== (f.admitted_at !== "") || retired !== (f.retired_by !== "") || retired !== (f.retired_at !== "") || absent !== (f.registered_at === "")) return invalid();
  const observed = retired ? (admitted ? "admitted" : "registered") : f.observation;
  switch (f.state) {
    case "registration_unknown": if (!["absent", "registered"].includes(observed) || f.pending_admitter !== "") return invalid(); break;
    case "registered": if (observed !== "registered" || f.pending_admitter !== "") return invalid(); break;
    case "admission_unknown": if (!["registered", "admitted"].includes(observed) || admitted === (f.pending_admitter !== "")) return invalid(); break;
    case "admitted": if (observed !== "admitted") return invalid(); break;
    default: return invalid();
  }
  if (f.request_not_before !== "" && (f.state !== "admitted" || f.observation !== "admitted" || f.rules !== "current")) return invalid();
  return freeze({ ...f }) as ProtectedChannel;
}

const changeKeys = ["change", "digest", "field", "phase", "state", "observation", "provenance", "instruction_kind", "channel", "rules", "requester", "requested_at", "effective", "approvals_required", "verifier", "verification_outcome", "verified_at", "pending_verifier", "pending_decision_actor", "ended_by", "ended_at"] as const;
const moduleStates = ["requested", "confirmed", "scheduled", "rejected", "lapsed"];
export type ProtectedDecision = Readonly<{ approver: string; verdict: string; at: string }>;
export type ProtectedChange = Readonly<Record<typeof changeKeys[number], string> & { decisions: readonly ProtectedDecision[] }>;

export function protectedChangeOf(value: unknown, field: ProtectedField): ProtectedChange {
  if (!object(value) || !Array.isArray(value.decisions) || value.decisions.length > 3) return invalid();
  const { decisions: rawDecisions, ...rest } = value;
  const f = text(rest, changeKeys);
  const decisions = (rawDecisions as unknown[]).map(entry => {
    const d = text(entry, ["approver", "verdict", "at"]);
    if (!id(d.approver) || !["approve", "reject"].includes(d.verdict) || !timestamp(d.at)) return invalid();
    return freeze({ approver: d.approver, verdict: d.verdict, at: d.at });
  });
  const absent = f.observation === "absent", verified = f.verifier !== "";
  if (!id(f.change) || !digest(f.digest) || f.field !== field || !id(f.channel) || !id(f.requester) || !timestamp(f.effective) || !["current", "other"].includes(f.rules)
    || !["staff_entry", "customer_document", "customer_portal"].includes(f.provenance) || !["email", "phone", "portal", "postal", "in_person"].includes(f.instruction_kind)
    || !["recorded", "request_unknown", "verification_unknown", "decision_unknown"].includes(f.phase)
    || !(absent || moduleStates.includes(f.observation)) || !(f.state === "" || moduleStates.includes(f.state))
    || ![f.verifier, f.pending_verifier, f.pending_decision_actor, f.ended_by].every(actor) || ![f.requested_at, f.verified_at, f.ended_at].every(instant)) return invalid();
  if (absent ? f.phase !== "request_unknown" || f.state !== "" || f.requested_at !== "" || f.approvals_required !== "" || verified || decisions.length > 0 || f.ended_at !== "" || f.ended_by !== ""
    : f.state === "" || !timestamp(f.requested_at) || !/^[1-3]$/.test(f.approvals_required)) return invalid();
  if (!absent && f.observation !== f.state && !(f.observation === "lapsed" && ["requested", "confirmed"].includes(f.state))) return invalid();
  if (verified !== (f.verified_at !== "") || verified !== ["confirmed", "denied"].includes(f.verification_outcome) || !verified && f.verification_outcome !== ""
    || verified && f.pending_verifier !== "" || f.pending_verifier !== "" && f.phase !== "verification_unknown" || f.phase === "verification_unknown" && !verified && f.pending_verifier === ""
    || f.pending_decision_actor !== "" && f.phase !== "decision_unknown" || decisions.length > 0 && !verified
    || new Set(decisions.map(d => d.approver)).size !== decisions.length
    || [f.verifier, f.pending_verifier, f.pending_decision_actor, ...decisions.map(d => d.approver)].includes(f.requester)
    || ["requested", "confirmed", "scheduled"].includes(f.observation) && (f.ended_at !== "" || f.ended_by !== "")
    || ["lapsed", "rejected"].includes(f.observation) && f.ended_at === "") return invalid();
  return freeze({ ...f, decisions: freeze(decisions) }) as ProtectedChange;
}

export type ProtectedPage<Row> = Readonly<{
  revision: string; at: string; scope: ProtectedScope; snapshot: string; next: string; count: number;
  otherProvenance: number; unownedModule: number; rows: readonly Row[]; currentEligibility: "unchecked";
}>;

function pageOf<Row>(e: ReadEnvelope, input: ProtectedScope, schema: string, extra: readonly string[], key: (row: Row) => string, decode: (raw: unknown) => Row, ordered: boolean, after: string, snapshot: string): ProtectedPage<Row> {
  const scope = protectedScope(input), f = e.fields;
  const fixed = ["schema", "contract", "configuration", "digest", "current_eligibility", "snapshot", "count", "next", "other_provenance", "unowned_module", ...extra];
  const keys = Object.keys(f).filter(k => !fixed.includes(k)).sort();
  if (fixed.some(k => !Object.hasOwn(f, k)) || keys.length > 10 || keys.some((k, i) => k !== `row_${String(i).padStart(2, "0")}`)) return invalid();
  scoped(f, scope, schema);
  if (!digest(f.snapshot) || snapshot && snapshot !== f.snapshot || ![f.count, f.other_provenance, f.unowned_module].every(count)) return invalid();
  const rows = keys.map(k => { try { return decode(strictJSON(f[k], 4096)); } catch { return invalid(); } });
  const total = Number(f.count), names = rows.map(key);
  if (total < rows.length || !after && rows.length !== Math.min(total, 10) || new Set(names).size !== names.length || names.includes(after)
    || ordered && names.some((name, i) => name <= (i ? names[i - 1] : after))
    || f.next !== "" && (rows.length !== 10 || f.next !== names.at(-1))
    || !after && f.next !== (total > 10 ? names.at(-1) : "") || total === 0 && f.next !== "") return invalid();
  return freeze({ revision: e.revision, at: e.at, scope, snapshot: f.snapshot, next: f.next, count: total, otherProvenance: Number(f.other_provenance), unownedModule: Number(f.unowned_module), rows: freeze(rows), currentEligibility: "unchecked" as const });
}

export function protectedChannelsOf(e: ReadEnvelope, scope: ProtectedScope, after = "", snapshot = ""): ProtectedPage<ProtectedChannel> {
  return pageOf(e, scope, "hb_issuance_protected_channel_page_v1", [], row => row.channel, protectedChannelOf, true, after, snapshot);
}

export function protectedChangesOf(e: ReadEnvelope, scope: ProtectedScope, field: ProtectedField, after = "", snapshot = ""): ProtectedPage<ProtectedChange> {
  if (!protectedFields.includes(field) || e.fields.field !== field) return invalid();
  return pageOf(e, scope, "hb_issuance_protected_change_page_v1", ["field"], row => row.change, raw => protectedChangeOf(raw, field), false, after, snapshot);
}
