import { contractDetailOf, contractsPageOf, type ContractDetail, type ContractsPage } from "./contracts";
import { installationOf, type InstallationPage } from "./installation";
import { integrationsOf, type IntegrationsPage } from "./integrations";
import { recoveryOf, type RecoveryPage } from "./recovery";
import { evidenceOf, type EvidencePage } from "./release-evidence";
import { retentionOf, type RetentionPage } from "./retention";

export type Membership = { role: "administrator" | "member"; state: "active" | "suspended"; version: number };

export type UserRow = {
  user_id: string;
  login: string;
  email: string;
  state: "active" | "suspended" | "departed";
  epoch: number;
  membership: Membership | null;
  password_set: boolean;
  invitation_expires: string;
  link_purpose: "" | "set" | "reset";
  link_expires: string;
  installation_administrator: boolean;
};

export type UsersPage = { revision: string; next: string; users: UserRow[] };

export type StepRow = { action: string; max_age_seconds: number; mfa: boolean };
export type RevisionRow = { methods: string[]; mfa: string; max_age_seconds: number; step_up: StepRow[] };
export type EpochRow = { value: string; generation: number; number: number };
export type PolicyRow = RevisionRow & { number: number; epoch: EpochRow; proposer: string; validator: string; approver: string; activator: string; at: string };
export type ScopeRow = { customer: string; legal_entity: string; contract: string; period: string };
export type AccessGrant = {
  id: string;
  subject: string;
  client: string;
  action: string;
  kind: string;
  fields: string[];
  scope: ScopeRow;
  not_before: string;
  not_after: string;
  state: "active" | "revoked" | "expired" | "pending" | "revocation_unconfirmed";
  effective: boolean;
  version: number;
};
export type AccessDelegation = {
  id: string;
  delegator: string;
  delegate: string;
  parent: string;
  redelegate: boolean;
  action: string;
  kind: string;
  fields: string[];
  scope: ScopeRow;
  not_before: string;
  not_after: string;
  state: "active" | "revoked" | "expired" | "pending" | "ended" | "revocation_unconfirmed";
  effective: boolean;
  version: number;
};
export type AccessSetting = { name: string; value: number; unit: string; default: number; min: number; max: number; source: string };
export type AccessRequest = {
  id: string;
  change: "add" | "revoke";
  subject: string;
  grant: string;
  action: string;
  kind: string;
  fields: string[];
  customer: string;
  days: number;
  digest: string;
  state: "pending" | "applied" | "rejected" | "withdrawn" | "expired" | "superseded";
  current: boolean;
  proposer: string;
  proposed_at: string;
  expires_at: string;
  reason: string;
  decided_by: string;
  decided_at: string;
  decision_reason: string;
  result: string;
  eligible_approvers: number;
  can_approve: boolean;
  can_reject: boolean;
  can_withdraw: boolean;
  may_end: string[];
};
export type CleanupItem = { item: string; state: "done" | "not_applicable" | "outside_scope" | "not_started"; reason: string; refs: string[] };
export type RevocationState = "received" | "deny_confirmed" | "cleanup_running" | "revocation_complete" | "decision_unconfirmed";
export type Revocation = {
  id: string;
  origin: "this_process" | "ledger_replay";
  recorded_at: string;
  target_kind: "grant" | "delegation";
  target: string;
  target_present: boolean;
  replay: "" | "reapplied" | "absent" | "protected";
  state: RevocationState;
  history: { state: string; at: string }[];
  ended_delegations: string[];
  deny_generation: number;
  job_state: "running" | "waiting" | "succeeded";
  job_reason: string;
  requested_by: string;
  approved_by: string;
  request: string;
  received_at: string;
  deny_at: string;
  receipt: { sequence: number; watermark: string; reason: string };
  verified: boolean;
  cleanup: CleanupItem[];
  completed_at: string;
};
export type LedgerStatus = {
  state: "ok" | "recovered" | "isolated" | "stopped" | "full" | "unknown";
  isolation_reason: string;
  ledger_id: string;
  created_at: string;
  replaces: string;
  replace_reason: string;
  replaced_file: string;
  records: number;
  watermark: string;
  checked_at: string;
  problem_line: number;
  recovery: { at: string; truncated_bytes: number } | null;
  not_durable: boolean;
  file: string;
};
export type Ceiling = { action: string; kind: string; fields: string[]; until: string };
export type AccessPage = {
  revision: string;
  grants: AccessGrant[];
  delegations: AccessDelegation[];
  catalog: Ceiling[];
  subjects: { user_id: string; login: string }[];
  now: string;
  you: string;
  started_at: string;
  settings: AccessSetting[];
  requests: AccessRequest[];
  revocations: Revocation[];
  ledger: LedgerStatus;
};
export type Preview = { action: string; field: string; allowed: boolean; reason: string; delegations: string[] };

export type CredentialRow = { id: string; not_before: string; not_after: string; state: "active" | "expired" | "revoked"; usable: boolean };
export type ServiceRow = {
  id: string;
  state: "active" | "suspended" | "retired";
  version: number;
  creator: string;
  creator_login: string;
  owner: string;
  owner_login: string;
  credentials: CredentialRow[];
};
export type ServicesPage = {
  revision: string;
  services: ServiceRow[];
  owners: { user_id: string; login: string }[];
  limits: { credential_days: number; credentials_per_service: number; services: number };
  now: string;
};
export type Secret = { secret: string; credential: string };
export type SelfSession = { handle: string; current: boolean; signed_in: string; ends: string };
export type SelfPage = {
  user_id: string;
  login: string;
  email: string;
  workspace: string;
  role: string;
  identities: { method: string; login: string }[];
  password_set: boolean;
  installation_administrator: boolean;
  sessions: SelfSession[];
  idle_minutes: number;
  max_age_seconds: number;
  signed_in_at: string;
  authenticated_at: string;
  now: string;
};

export type OrgParty = { id: string; name: string; kind: "legal_entity" | "internal_unit"; parent: string; state: string };
export type OrgProfile = { id: string; name: string; issuer: string; buyer: string; payer: string; recipient: string; path: string; currency: string; state: string };
export type OrgTarget = { id: string; name: string };
export type OrgHistory = { target: string; name: string; from: string; until: string };
export type OrgMember = { member: string; name: string; targets: OrgTarget[]; history: OrgHistory[] };
export type RelationKind = "hierarchy" | "payer" | "contract_pool" | "allocation_pool" | "view_group";
export type OrgPage = {
  revision: string;
  relationship_revision: number;
  active_revision: number;
  on: string;
  workspace: { id: string; profile: string };
  parties: OrgParty[];
  profiles: OrgProfile[];
  kind: RelationKind;
  members: OrgMember[];
  next: string;
};
export type PlanChange = { kind: string; member: string; action: string; target: string; from: string };
export type PlanKind = { kind: string; changed: number; entries_before: number; entries_after: number; earliest: string };
export type PlanViolation = { kind: string; member: string; rule: string };
export type PlanRange = { from: string; until: string };
export type PlanMember = { member: string; name: string; differs: PlanRange[]; before: string[]; after: string[] };
export type ImpactState = "computed" | "not_computed" | "unknown";
export type PlanImpact = { money: ImpactState; publication: ImpactState; erp: ImpactState };
const impactState = (v: unknown): ImpactState => (v === "computed" || v === "not_computed" ? v : "unknown");
export type PlanRow = {
  id: string;
  state: string;
  closed: string;
  created: string;
  proposer: string;
  proposer_login: string;
  reason: string;
  change: PlanChange;
  member_name: string;
  target_name: string;
  base: number;
  stale: boolean;
  activated_revision: number;
  kinds: PlanKind[];
  violations: PlanViolation[];
  truncated: boolean;
  members: PlanMember[];
  impact: PlanImpact;
  can_activate: boolean;
  blocked: string;
};
export type PlansPage = {
  revision: string;
  active_revision: number;
  open_plans: number;
  open_limit: number;
  plans: PlanRow[];
  you: string;
};

export function planOf(v: unknown): PlanRow {
  const r = (v ?? {}) as Record<string, unknown>;
  const list = <T,>(x: unknown) => (Array.isArray(x) ? (x as T[]) : []);
  const c = (r.change ?? {}) as Record<string, unknown>;
  const impact = (r.impact && typeof r.impact === "object" ? r.impact : {}) as Record<string, unknown>;
  return {
    id: String(r.id ?? ""),
    state: String(r.state ?? ""),
    closed: String(r.closed ?? ""),
    created: String(r.created ?? ""),
    proposer: String(r.proposer ?? ""),
    proposer_login: String(r.proposer_login ?? ""),
    reason: String(r.reason ?? ""),
    change: { kind: String(c.kind ?? ""), member: String(c.member ?? ""), action: String(c.action ?? ""), target: String(c.target ?? ""), from: String(c.from ?? "") },
    member_name: String(r.member_name ?? ""),
    target_name: String(r.target_name ?? ""),
    base: Number(r.base ?? 0),
    stale: r.stale === true,
    activated_revision: Number(r.activated_revision ?? 0),
    kinds: list<PlanKind | null>(r.kinds).filter((k): k is PlanKind => !!k && typeof k === "object"),
    violations: list<PlanViolation | null>(r.violations).filter((v): v is PlanViolation => !!v && typeof v === "object").map((v) => ({ kind: String(v.kind ?? ""), member: String(v.member ?? ""), rule: String(v.rule ?? "") })),
    truncated: r.truncated === true,
    members: list<PlanMember | null>(r.members)
      .filter((m): m is PlanMember => !!m && typeof m === "object")
      .map((m) => ({
        member: String(m.member ?? ""),
        name: String(m.name ?? ""),
        differs: list<PlanRange | null>(m.differs).filter((d): d is PlanRange => !!d && typeof d === "object").map((d) => ({ from: String(d.from ?? ""), until: String(d.until ?? "") })),
        before: list<unknown>(m.before).map(String),
        after: list<unknown>(m.after).map(String),
      })),
    impact: { money: impactState(impact.money), publication: impactState(impact.publication), erp: impactState(impact.erp) },
    can_activate: r.can_activate === true,
    blocked: String(r.blocked ?? ""),
  };
}

export type OrgQuery = { on?: string; kind?: RelationKind; revision?: number; after?: string; limit?: number };

export type TrustHold = { reason: string; since: string };
export type TrustRow = {
  epoch: string;
  epoch_generation: number;
  epoch_number: number;
  policy_number: number;
  digest: string;
  origin: "seed" | "this_process" | "ledger_replay" | "carried" | "";
  carried_from: string;
  carried_line: number;
  durable_unconfirmed: boolean;
  activated_at: string;
  reset_at: string;
  older_record_ignored: boolean;
  proposer: string;
  validator: string;
  approver: string;
  activator: string;
  change: string;
  hold: TrustHold | null;
};
export type TrustLedger = {
  state: string;
  isolation_reason: string;
  problem_line: number;
  records: number;
  watermark: string;
  generation: number;
  generation_stated: boolean;
  not_durable: boolean;
};
export type TrustRights = { propose: boolean; validate: boolean; approve: boolean; activate: boolean };
export type TrustHolders = { qualifying_people: number; members: number; needed: number; enrolment: string; ready: boolean; missing: string[] };
export type TrustItem = { item: string; result: string; reason: string; detail: string };
export type TrustValidation = { result: string; at: string; evidence_digest: string; evidence_current: boolean; items: TrustItem[] };
export type TrustImpact = { members: number; sessions: number; affected_ids: string[]; weakens_mfa: boolean; signs_out_everyone: boolean };
export type TrustActivation = { epoch: EpochRow; ledger_sequence: number; watermark: string; at: string };
export type TrustJob = { id: string; state: string; reason: string };
export type TrustProgress = { item: string; result: string; ended: number; remaining: number; reason: string };
export type TrustInvalidation = { state: string; items: TrustProgress[] };
export type TrustCan = { revise: boolean; validate: boolean; approve: boolean; activate: boolean; reject: boolean; cancel: boolean };
export type TrustWhy = { approve: string; activate: string };
export type TrustHistory = { state: string; by: string; at: string; rationale: string };
export type TrustState = "draft" | "validated" | "independently_approved" | "activated" | "rejected" | "cancelled" | "superseded" | "activation_unknown";
export type TrustChange = {
  id: string;
  number: number;
  state: TrustState;
  change_revision: number;
  change_digest: string;
  old_epoch: EpochRow;
  old_policy: number;
  old_digest: string;
  new_digest: string;
  candidate: RevisionRow;
  proposer: string;
  validator: string;
  approver: string;
  activator: string;
  superseded_by: string;
  superseded_epoch: EpochRow | null;
  earlier_run: boolean;
  run_started_at: string;
  validation: TrustValidation | null;
  impact: TrustImpact;
  activation: TrustActivation | null;
  job: TrustJob | null;
  invalidation: TrustInvalidation | null;
  can: TrustCan;
  why_not: TrustWhy;
  act_until: string;
  history: TrustHistory[];
};
export type PolicyPage = {
  revision: string;
  policy: PolicyRow | null;
  now: string;
  signed_in_at: string;
  authenticated_at: string;
  act_until: string;
  step_up_seconds: number;
  you: string;
  started_at: string;
  trust: TrustRow;
  ledger: TrustLedger;
  rights: TrustRights;
  holders: TrustHolders;
  changes: TrustChange[];
};

const str = (v: unknown) => (typeof v === "string" ? v : "");
const num = (v: unknown) => (typeof v === "number" && Number.isFinite(v) ? v : 0);
const obj = (v: unknown): Record<string, unknown> => (v && typeof v === "object" && !Array.isArray(v) ? (v as Record<string, unknown>) : {});
const list = (v: unknown): unknown[] => (Array.isArray(v) ? v : []);

export function epochOf(v: unknown): EpochRow {
  const r = obj(v);
  return { value: str(r.value), generation: num(r.generation), number: num(r.number) };
}

function revisionOf(v: unknown): RevisionRow {
  const r = obj(v);
  return {
    methods: list(r.methods).map(String),
    mfa: str(r.mfa),
    max_age_seconds: num(r.max_age_seconds),
    step_up: list(r.step_up).map((x) => { const s = obj(x); return { action: str(s.action), max_age_seconds: num(s.max_age_seconds), mfa: s.mfa === true }; }),
  };
}

const trustStates: TrustState[] = ["draft", "validated", "independently_approved", "activated", "rejected", "cancelled", "superseded", "activation_unknown"];

export function trustChangeOf(v: unknown): TrustChange {
  const r = obj(v);
  const val = r.validation ? obj(r.validation) : null;
  const imp = obj(r.impact);
  const act = r.activation ? obj(r.activation) : null;
  const job = r.job ? obj(r.job) : null;
  const inv = r.invalidation ? obj(r.invalidation) : null;
  const can = obj(r.can);
  const why = obj(r.why_not);
  const state = trustStates.includes(r.state as TrustState) ? (r.state as TrustState) : "draft";
  return {
    id: str(r.id), number: num(r.number), state, change_revision: num(r.change_revision), change_digest: str(r.change_digest), old_epoch: epochOf(r.old_epoch),
    old_policy: num(r.old_policy), old_digest: str(r.old_digest), new_digest: str(r.new_digest), candidate: revisionOf(r.candidate), proposer: str(r.proposer),
    validator: str(r.validator), approver: str(r.approver), activator: str(r.activator), superseded_by: str(r.superseded_by),
    superseded_epoch: r.superseded_epoch ? epochOf(r.superseded_epoch) : null, earlier_run: r.earlier_run === true, run_started_at: str(r.run_started_at),
    validation: val && {
      result: str(val.result), at: str(val.at), evidence_digest: str(val.evidence_digest), evidence_current: val.evidence_current === true,
      items: list(val.items).map((x) => { const i = obj(x); return { item: str(i.item), result: str(i.result), reason: str(i.reason), detail: str(i.detail) }; }),
    },
    impact: { members: num(imp.members), sessions: num(imp.sessions), affected_ids: list(imp.affected_ids).map(String), weakens_mfa: imp.weakens_mfa === true, signs_out_everyone: imp.signs_out_everyone !== false },
    activation: act && { epoch: epochOf(act.epoch), ledger_sequence: num(act.ledger_sequence), watermark: str(act.watermark), at: str(act.at) },
    job: job && { id: str(job.id), state: str(job.state), reason: str(job.reason) },
    invalidation: inv && {
      state: str(inv.state),
      items: list(inv.items).map((x) => { const i = obj(x); return { item: str(i.item), result: str(i.result), ended: num(i.ended), remaining: num(i.remaining), reason: str(i.reason) }; }),
    },
    can: { revise: can.revise === true, validate: can.validate === true, approve: can.approve === true, activate: can.activate === true, reject: can.reject === true, cancel: can.cancel === true },
    why_not: { approve: str(why.approve), activate: str(why.activate) },
    act_until: str(r.act_until),
    history: list(r.history).map((x) => { const h = obj(x); return { state: str(h.state), by: str(h.by), at: str(h.at), rationale: str(h.rationale) }; }),
  };
}

export function policyPageOf(body: Record<string, unknown>): PolicyPage {
  const t = obj(body.trust);
  const hold = t.hold ? obj(t.hold) : null;
  const led = obj(body.ledger);
  const rights = obj(body.rights);
  const holders = obj(body.holders);
  const origins = ["seed", "this_process", "ledger_replay", "carried"];
  const p = body.policy ? obj(body.policy) : null;
  return {
    revision: str(body.revision),
    policy: p && { ...revisionOf(p), number: num(p.number), epoch: epochOf(p.epoch), proposer: str(p.proposer), validator: str(p.validator), approver: str(p.approver), activator: str(p.activator), at: str(p.at) },
    now: str(body.now),
    signed_in_at: str(body.signed_in_at),
    authenticated_at: str(body.authenticated_at),
    act_until: str(body.act_until),
    step_up_seconds: num(body.step_up_seconds),
    you: str(body.you),
    started_at: str(body.started_at),
    trust: {
      epoch: str(t.epoch), epoch_generation: num(t.epoch_generation), epoch_number: num(t.epoch_number), policy_number: num(t.policy_number), digest: str(t.digest),
      origin: (origins.includes(str(t.origin)) ? str(t.origin) : "") as TrustRow["origin"], carried_from: str(t.carried_from), carried_line: num(t.carried_line),
      durable_unconfirmed: t.durable_unconfirmed === true, activated_at: str(t.activated_at), reset_at: str(t.reset_at), older_record_ignored: t.older_record_ignored === true, proposer: str(t.proposer), validator: str(t.validator),
      approver: str(t.approver), activator: str(t.activator), change: str(t.change), hold: hold && { reason: str(hold.reason), since: str(hold.since) },
    },
    ledger: {
      state: str(led.state) || "unknown", isolation_reason: str(led.isolation_reason), problem_line: num(led.problem_line), records: num(led.records), watermark: str(led.watermark),
      generation: num(led.generation), generation_stated: led.generation_stated === true, not_durable: led.not_durable === true,
    },
    rights: { propose: rights.propose === true, validate: rights.validate === true, approve: rights.approve === true, activate: rights.activate === true },
    holders: { qualifying_people: num(holders.qualifying_people), members: num(holders.members), needed: num(holders.needed) || 3, enrolment: str(holders.enrolment), ready: holders.ready === true,
      missing: Array.isArray(holders.missing) ? holders.missing.map((x) => String(x)) : [] },
    changes: list(body.changes).map(trustChangeOf),
  };
}

export type AdminFailure =
  | "conflict"
  | "refused"
  | "stage"
  | "invalid"
  | "denied"
  | "unauthenticated"
  | "limited"
  | "unavailable"
  | "partial"
  | "busy"
  | "stale"
  | "wrong"
  | "locked"
  | "form"
  | "reauth"
  | "reauth_cancelled"
  | "self"
  | "epoch"
  | "approval_stale"
  | "evidence_stale"
  | "needs_approval"
  | "trust_unavailable";

export class AdminError extends Error {
  constructor(
    readonly failure: AdminFailure,
    readonly fields: string[] = [],
    readonly retryAfter = 0,
    readonly operation = "",
    readonly code = "",
  ) {
    super(failure);
  }
}

export type CommandId =
  | "identity.create_user"
  | "identity.invite"
  | "identity.issue_set_link"
  | "identity.issue_reset_link"
  | "identity.suspend_user"
  | "identity.reinstate_user"
  | "identity.depart_user"
  | "identity.change_role"
  | "identity.suspend_membership"
  | "identity.reinstate_membership"
  | "identity.remove_membership"
  | "identity.revoke_invitation"
  | "trust.propose"
  | "trust.validate"
  | "trust.approve"
  | "trust.activate"
  | "trust.reject"
  | "trust.cancel"
  | "access.request_grant"
  | "access.request_revoke"
  | "access.approve_request"
  | "access.reject_request"
  | "access.withdraw_request"
  | "access.revoke_delegation"
  | "service.create"
  | "service.suspend"
  | "service.reinstate"
  | "service.retire"
  | "service.issue_credential"
  | "service.revoke_credential"
  | "relationship.propose"
  | "relationship.activate"
  | "relationship.discard"
  | "contract.suspend"
  | "contract.resume"
  | "contract.renew"
  | "contract.end"
  | "contract.withdraw"
  | "contract.evidence"
  | "contract.settle"
  | "contract.reopen"
  | "installation.approve"
  | "installation.withdraw"
  | "installation.propose_plan"
  | "integrations.plugin_approve"
  | "integrations.plugin_withdraw"
  | "integrations.plugin_propose"
  | "integrations.plugin_stop"
  | "integrations.plugin_disable"
  | "integrations.subscription_propose"
  | "integrations.subscription_approve"
  | "integrations.subscription_pause"
  | "integrations.subscription_request_resume"
  | "integrations.subscription_approve_resume"
  | "integrations.subscription_revoke"
  | "integrations.letter_request_redeliver"
  | "integrations.letter_approve_redeliver"
  | "integrations.letter_request_dismiss"
  | "integrations.letter_approve_dismiss"
  | "recovery.approve"
  | "recovery.withdraw"
  | "recovery.lock"
  | "recovery.propose_unlock"
  | "retention.approve"
  | "retention.withdraw"
  | "retention.propose_policy"
  | "retention.place_hold"
  | "retention.propose_release"
  | "retention.stop_run"
  | "release_evidence.approve"
  | "release_evidence.withdraw"
  | "release_evidence.retire_requirement"
  | "release_evidence.retire_profile";

export type Outcome = { status: string; state: string; job_id: string; fields: Record<string, string> };

const unknownLedger: LedgerStatus = {
  state: "unknown", isolation_reason: "", ledger_id: "", created_at: "", replaces: "", replace_reason: "", replaced_file: "", records: 0, watermark: "", checked_at: "",
  problem_line: 0, recovery: null, not_durable: false, file: "",
};

export function ledgerOf(v: unknown): LedgerStatus {
  if (!v || typeof v !== "object") return unknownLedger;
  const r = v as Record<string, unknown>;
  const states = ["ok", "recovered", "isolated", "stopped", "full"];
  const recovery = r.recovery && typeof r.recovery === "object" ? (r.recovery as Record<string, unknown>) : null;
  return {
    state: (states.includes(String(r.state)) ? String(r.state) : "unknown") as LedgerStatus["state"],
    isolation_reason: String(r.isolation_reason ?? ""),
    ledger_id: String(r.ledger_id ?? ""),
    created_at: String(r.created_at ?? ""),
    replaces: String(r.replaces ?? ""),
    replace_reason: String(r.replace_reason ?? ""),
    replaced_file: String(r.replaced_file ?? ""),
    records: Number(r.records ?? 0),
    watermark: String(r.watermark ?? ""),
    checked_at: String(r.checked_at ?? ""),
    problem_line: Number(r.problem_line ?? 0),
    recovery: recovery ? { at: String(recovery.at ?? ""), truncated_bytes: Number(recovery.truncated_bytes ?? 0) } : null,
    not_durable: r.not_durable === true,
    file: String(r.file ?? ""),
  };
}

export function outcomeOf(body: Record<string, unknown>): Outcome {
  const fields = (body.fields ?? {}) as Record<string, unknown>;
  return {
    status: String(body.status ?? ""),
    state: String(body.business_state ?? ""),
    job_id: String(body.job_id ?? ""),
    fields: Object.fromEntries(Object.entries(fields).map(([k, v]) => [k, String(v)])),
  };
}

export type Attempt = { request: string; key: string };

type Fetch = typeof fetch;

const codes: Record<string, AdminFailure> = {
  revision_conflict: "conflict",
  prerequisite_failed: "refused",
  invalid_stage: "stage",
  invalid_request: "invalid",
  permission_denied: "denied",
  unauthenticated: "unauthenticated",
  csrf_refused: "unauthenticated",
  rate_limited: "limited",
  internal_error: "partial",
  dependency_unavailable: "busy",
  idempotency_conflict: "stale",
  idempotency_key_expired: "stale",
  current_password_wrong: "wrong",
  password_locked: "locked",
  password_form: "form",
  reauthentication_required: "reauth",
  self_approval_denied: "self",
  authority_epoch_conflict: "epoch",
  approval_stale: "approval_stale",
  evidence_stale: "evidence_stale",
  independent_approval_required: "needs_approval",
};

export type StepUp = () => Promise<boolean>;

export function failureOf(status: number, body: Record<string, unknown>): AdminError {
  const code = String(body.code ?? "");
  const fields = Array.isArray(body.fields) ? body.fields.map(String) : [];
  const operation = typeof body.operation_id === "string" ? body.operation_id : "";
  const failure = codes[code] ?? (status === 401 ? "unauthenticated" : status === 429 ? "limited" : "unavailable");
  if (failure === "partial" && !operation) return new AdminError("unavailable", [], 0, "", code);
  return new AdminError(failure, fields, 0, operation, code);
}

function random(): string {
  const bytes = new Uint8Array(12);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

export function attempt(): Attempt {
  return { request: "m01-" + random(), key: "m01-" + random() };
}

export async function reauthenticate(csrf: string, password: string, f: Fetch = fetch.bind(globalThis)): Promise<void> {
  let r: Response;
  try {
    r = await f("/api/v1/me/reauthenticate", {
      method: "POST",
      credentials: "same-origin",
      cache: "no-store",
      headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf, "X-Request-Id": attempt().request },
      body: JSON.stringify({ password }),
    });
  } catch {
    throw new AdminError("unavailable");
  }
  if (r.ok) return;
  let body: Record<string, unknown> = {};
  try {
    body = await r.json();
  } catch {
    body = {};
  }
  throw failureOf(r.status, body);
}

export function adminClient(csrf: string, f: Fetch = fetch.bind(globalThis), stepUp?: StepUp) {
  async function exchange(path: string, init: RequestInit, request: string): Promise<{ r: Response; body: Record<string, unknown> }> {
    let r: Response;
    try {
      r = await f("/api/v1" + path, {
        credentials: "same-origin",
        cache: "no-store",
        ...init,
        headers: { "X-CSRF-Token": csrf, "X-Request-Id": request, ...(init.headers as Record<string, string> | undefined) },
      });
    } catch {
      throw new AdminError("unavailable");
    }
    let body: Record<string, unknown> = {};
    try {
      body = await r.json();
    } catch {
      body = {};
    }
    return { r, body };
  }
  async function send(path: string, init: RequestInit, request: string): Promise<Record<string, unknown>> {
    let { r, body } = await exchange(path, init, request);
    if (!r.ok && init.method === "POST" && body.code === "reauthentication_required" && stepUp) {
      if (!(await stepUp())) throw new AdminError("reauth_cancelled", [], 0, "", "reauthentication_required");
      ({ r, body } = await exchange(path, init, request));
    }
    if (r.ok) return body;
    const failed = failureOf(r.status, body);
    const retry = Number(r.headers.get("Retry-After") ?? "0");
    if (failed.failure === "limited" && Number.isFinite(retry) && retry > 0) {
      throw new AdminError("limited", [], retry);
    }
    throw failed;
  }
  return {
    async users(after = "", limit = 50): Promise<UsersPage> {
      const q = new URLSearchParams({ limit: String(limit) });
      if (after) q.set("after", after);
      const body = await send("/identity/users?" + q.toString(), { method: "GET" }, attempt().request);
      return {
        revision: String(body.revision ?? ""),
        next: String(body.next ?? ""),
        users: Array.isArray(body.users) ? (body.users as UserRow[]) : [],
      };
    },
    async command(id: CommandId, target: string, expected: string, rationale: string, values: Record<string, string>, tried: Attempt): Promise<Outcome> {
      const body = await send(
        "/commands/" + id,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ request_id: tried.request, target_id: target, idempotency_key: tried.key, expected_revision: expected, rationale, body: values }),
        },
        tried.request,
      );
      return outcomeOf(body);
    },
    async policy(): Promise<PolicyPage> {
      return policyPageOf(await send("/identity/policy", { method: "GET" }, attempt().request));
    },
    async access(): Promise<AccessPage> {
      const body = await send("/identity/access", { method: "GET" }, attempt().request);
      const list = <T,>(v: unknown) => (Array.isArray(v) ? (v as T[]) : []);
      return {
        revision: String(body.revision ?? ""),
        grants: list<AccessGrant>(body.grants),
        delegations: list<AccessDelegation>(body.delegations),
        catalog: list<Ceiling>(body.catalog),
        subjects: list<{ user_id: string; login: string }>(body.subjects),
        now: String(body.now ?? ""),
        you: String(body.you ?? ""),
        started_at: String(body.started_at ?? ""),
        settings: list<AccessSetting>(body.settings),
        requests: list<AccessRequest>(body.requests).map((q) => ({ ...q, fields: Array.isArray(q.fields) ? q.fields : [], may_end: Array.isArray(q.may_end) ? q.may_end : [] })),
        revocations: list<Revocation>(body.revocations).map((v) => ({
          ...v,
          history: Array.isArray(v.history) ? v.history : [],
          ended_delegations: Array.isArray(v.ended_delegations) ? v.ended_delegations : [],
          cleanup: Array.isArray(v.cleanup) ? v.cleanup.map((c) => ({ ...c, refs: Array.isArray(c.refs) ? c.refs : [] })) : [],
          receipt: v.receipt ?? { sequence: 0, watermark: "", reason: "" },
        })),
        ledger: ledgerOf(body.ledger),
      };
    },
    async preview(subject: string, kind: string, customer = ""): Promise<Preview[]> {
      const q = new URLSearchParams({ subject, kind });
      if (customer) q.set("customer", customer);
      const body = await send("/identity/access/preview?" + q.toString(), { method: "GET" }, attempt().request);
      return Array.isArray(body.decisions) ? (body.decisions as Preview[]) : [];
    },
    async services(): Promise<ServicesPage> {
      const body = await send("/identity/services", { method: "GET" }, attempt().request);
      const limits = (body.limits ?? {}) as Record<string, unknown>;
      return {
        revision: String(body.revision ?? ""),
        services: Array.isArray(body.services) ? (body.services as ServiceRow[]).map((s) => ({ ...s, credentials: Array.isArray(s.credentials) ? s.credentials : [] })) : [],
        owners: Array.isArray(body.owners) ? (body.owners as { user_id: string; login: string }[]) : [],
        limits: {
          credential_days: Number(limits.credential_days ?? 0),
          credentials_per_service: Number(limits.credentials_per_service ?? 0),
          services: Number(limits.services ?? 0),
        },
        now: String(body.now ?? ""),
      };
    },
    async secret(ref: string): Promise<Secret> {
      const body = await send("/identity/links/" + encodeURIComponent(ref), { method: "GET" }, attempt().request);
      const secret = String(body.secret ?? "");
      const credential = String(body.credential ?? "");
      if (!/^hbsk_[A-Za-z0-9_-]{43}$/.test(secret) || !/^cred-[0-9a-f]{16}$/.test(credential)) throw new AdminError("invalid");
      return { secret, credential };
    },
    async me(): Promise<SelfPage> {
      const body = await send("/me", { method: "GET" }, attempt().request);
      return {
        user_id: String(body.user_id ?? ""),
        login: String(body.login ?? ""),
        email: String(body.email ?? ""),
        workspace: String(body.workspace ?? ""),
        role: String(body.role ?? ""),
        identities: Array.isArray(body.identities) ? (body.identities as { method: string; login: string }[]) : [],
        password_set: body.password_set === true,
        installation_administrator: body.installation_administrator === true,
        sessions: Array.isArray(body.sessions) ? (body.sessions as SelfSession[]) : [],
        idle_minutes: Number(body.idle_minutes ?? 0),
        max_age_seconds: Number(body.max_age_seconds ?? 0),
        signed_in_at: String(body.signed_in_at ?? ""),
        authenticated_at: String(body.authenticated_at ?? ""),
        now: String(body.now ?? ""),
      };
    },
    async changePassword(current: string, next: string): Promise<void> {
      await send("/me/password", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ current, new: next }) }, attempt().request);
    },
    async endSessions(session: string): Promise<number> {
      const body = await send("/me/sessions/end", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ session }) }, attempt().request);
      return Number(body.ended ?? 0);
    },
    async organizations(query: OrgQuery = {}): Promise<OrgPage> {
      const q = new URLSearchParams();
      if (query.on) q.set("on", query.on);
      if (query.kind) q.set("kind", query.kind);
      if (query.revision !== undefined) q.set("revision", String(query.revision));
      if (query.after) q.set("after", query.after);
      if (query.limit) q.set("limit", String(query.limit));
      const qs = q.toString();
      const body = await send("/organizations" + (qs ? "?" + qs : ""), { method: "GET" }, attempt().request);
      const list = <T,>(v: unknown) => (Array.isArray(v) ? (v as T[]) : []);
      const ws = (body.workspace ?? {}) as Record<string, unknown>;
      return {
        revision: String(body.revision ?? ""),
        relationship_revision: Number(body.relationship_revision ?? 0),
        active_revision: Number(body.active_revision ?? 0),
        on: String(body.on ?? ""),
        workspace: { id: String(ws.id ?? ""), profile: String(ws.profile ?? "") },
        parties: list<OrgParty>(body.parties),
        profiles: list<OrgProfile>(body.profiles),
        kind: String(body.kind ?? "hierarchy") as RelationKind,
        members: list<OrgMember>(body.members).map((m) => ({ ...m, targets: Array.isArray(m.targets) ? m.targets : [], history: Array.isArray(m.history) ? m.history : [] })),
        next: String(body.next ?? ""),
      };
    },
    async plans(): Promise<PlansPage> {
      const body = await send("/organizations/plans", { method: "GET" }, attempt().request);
      return {
        revision: String(body.revision ?? ""),
        active_revision: Number(body.active_revision ?? 0),
        open_plans: Number(body.open_plans ?? 0),
        open_limit: Number(body.open_limit ?? 0),
        plans: (Array.isArray(body.plans) ? body.plans : []).map(planOf),
        you: String(body.you ?? ""),
      };
    },
    async contracts(on = ""): Promise<ContractsPage> {
      const q = on ? "?" + new URLSearchParams({ on }).toString() : "";
      return contractsPageOf(await send("/contracts" + q, { method: "GET" }, attempt().request));
    },
    async contract(id: string, from = "", until = ""): Promise<ContractDetail | "missing"> {
      const q = new URLSearchParams();
      if (from) q.set("from", from);
      if (until) q.set("until", until);
      const qs = q.toString();
      try {
        return contractDetailOf(await send("/contracts/" + encodeURIComponent(id) + (qs ? "?" + qs : ""), { method: "GET" }, attempt().request));
      } catch (e) {
        if (e instanceof AdminError && e.code === "unsupported_command") return "missing";
        throw e;
      }
    },
    async installation(): Promise<InstallationPage> {
      return installationOf(await send("/installation", { method: "GET" }, attempt().request));
    },
    async integrations(): Promise<IntegrationsPage> {
      return integrationsOf(await send("/integrations", { method: "GET" }, attempt().request));
    },
    async recovery(): Promise<RecoveryPage> {
      return recoveryOf(await send("/recovery", { method: "GET" }, attempt().request));
    },
    async retention(): Promise<RetentionPage> {
      return retentionOf(await send("/retention", { method: "GET" }, attempt().request));
    },
    async releaseEvidence(request = attempt().request): Promise<EvidencePage> {
      return evidenceOf(await send("/release-evidence", { method: "GET" }, request));
    },
    async collect(ref: string): Promise<string> {
      const body = await send("/identity/links/" + encodeURIComponent(ref), { method: "GET" }, attempt().request);
      const url = String(body.url ?? "");
      if (!/^https?:\/\/[^#]+\/activate#[A-Za-z0-9_-]{43}$/.test(url)) throw new AdminError("unavailable");
      return url;
    },
  };
}

export type AdminClient = ReturnType<typeof adminClient>;

export function sequence() {
  let current = 0;
  return {
    next(): number {
      current += 1;
      return current;
    },
    latest(n: number): boolean {
      return n === current;
    },
  };
}
