import type { SessionRow } from "./session-query";
import { sessionQueryClient } from "./session-query";
import { grantsOf, delegationsOf, catalogOf, subjectsOf, settingsOf as accessSettingsOf, requestsOf, revocationsOf, previewsOf } from "./access-read";
import { credentialOf, servicesPageOf, selfPageOf } from "./account-read";
import { organizationsPageOf } from "./organization-read";
import { ReadParseError, text as readText } from "./strict-read";
import { accessHistoryOf, HistoryParseError, type AccessHistoryPage, type AccessHistoryQuery } from "./access-history";
import { contractDetailOf, contractsPageOf, type ContractDetail, type ContractsPage } from "./contracts";
import { installationOf, type InstallationPage } from "./installation";
import { integrationsOf, type IntegrationsPage } from "./integrations";
import { recoveryOf, type RecoveryPage } from "./recovery";
import { evidenceOf, type EvidencePage } from "./release-evidence";
import { retentionOf, type RetentionPage } from "./retention";
import { selfEndOf, type SelfEnd, type SelfSession } from "./self-end";

export type RolePeriod = "open" | "bounded" | "ends_soon" | "ended";

export type Membership = {
  role: "administrator" | "member";
  state: "active" | "suspended" | "unknown";
  version: number;
  role_from: string;
  role_until: string;
  role_period: RolePeriod;
  suspended_for: "" | "role_period_ended" | "bulk_revocation";
  ledger: "" | "confirmed" | "unconfirmed";
};

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
  revocation: { bulk: string; state: string } | null;
};

export type SettingRow = { name: string; value: number; unit: string; default: number; min: number; max: number; source: string };

export type ClockState = {
  server_time: string;
  wall_time: string;
  clock_behind: boolean;
  clock_ahead: boolean;
  clock_base: { choice: string; ledger_time: string; wall_time: string; recorded: boolean };
  wall_lead: { wall_time: string; server_time: string } | null;
  recovery: { stranded: boolean; records: number[] };
};

export type UsersFilter = { q: string; status: string; membership: string; role: string; period: string; signin: string; invitation: string; holder: string };

export type UsersQuery = Partial<UsersFilter> & { after?: string; limit?: number };

export type UsersPage = {
  revision: string;
  next: string;
  next_login: string;
  users: UserRow[];
  clock: ClockState;
  settings: SettingRow[];
  ledger_state: string;
  can_bulk_revoke: boolean;
  filters: UsersFilter;
  scan_stopped: boolean;
  scan_budget: number;
  order: string;
  list_version: string;
  auth_methods: { connected: string[]; not_connected: string[] };
  warning_days: number;
};

const filterKeys: (keyof UsersFilter)[] = ["q", "status", "membership", "role", "period", "signin", "invitation", "holder"];

export function usersQueryString(query: UsersQuery): string {
  const q = new URLSearchParams();
  for (const k of filterKeys) if (query[k]) q.set(k, query[k]!);
  if (query.after) q.set("after", query.after);
  q.set("limit", String(query.limit ?? 50));
  return q.toString();
}

export function filterOf(v: unknown): UsersFilter {
  const r = (v && typeof v === "object" ? v : {}) as Record<string, unknown>;
  return Object.fromEntries(filterKeys.map((k) => [k, String(r[k] ?? "")])) as UsersFilter;
}

function names(v: unknown): string[] {
  return Array.isArray(v) ? v.map(String) : [];
}

const periods: RolePeriod[] = ["open", "bounded", "ends_soon", "ended"];

export function membershipOf(v: unknown): Membership | null {
  if (!v || typeof v !== "object") return null;
  const m = v as Record<string, unknown>;
  const suspended = String(m.suspended_for ?? "");
  const ledger = String(m.ledger ?? "");
  return {
    role: m.role === "administrator" ? "administrator" : "member",
    state: m.state === "suspended" ? "suspended" : m.state === "active" ? "active" : "unknown",
    version: Number(m.version ?? 0),
    role_from: String(m.role_from ?? ""),
    role_until: String(m.role_until ?? ""),
    role_period: periods.includes(m.role_period as RolePeriod) ? (m.role_period as RolePeriod) : "open",
    suspended_for: suspended === "role_period_ended" || suspended === "bulk_revocation" ? suspended : "",
    ledger: ledger === "confirmed" || ledger === "unconfirmed" ? ledger : "",
  };
}

export function clockOf(body: Record<string, unknown>): ClockState {
  const base = (body.clock_base ?? {}) as Record<string, unknown>;
  const lead = body.wall_lead_beyond_tolerance as Record<string, unknown> | null | undefined;
  const recovery = (body.recovery_state ?? {}) as Record<string, unknown>;
  return {
    server_time: String(body.server_time ?? ""),
    wall_time: String(body.wall_time ?? ""),
    clock_behind: body.clock_behind === true,
    clock_ahead: body.clock_ahead === true,
    clock_base: { choice: String(base.choice ?? "default"), ledger_time: String(base.ledger_time ?? ""), wall_time: String(base.wall_time ?? ""), recorded: base.recorded === true },
    wall_lead: lead && typeof lead === "object" ? { wall_time: String(lead.wall_time ?? ""), server_time: String(lead.server_time ?? "") } : null,
    recovery: { stranded: recovery.stranded === true, records: Array.isArray(recovery.records) ? recovery.records.map(Number) : [] },
  };
}

export function settingsOf(v: unknown): SettingRow[] {
  return Array.isArray(v) ? (v as SettingRow[]) : [];
}

export function userRowOf(v: unknown): UserRow {
  const r = v as UserRow & { membership: unknown; revocation: unknown };
  const rev = r.revocation as Record<string, unknown> | null | undefined;
  return { ...r, membership: membershipOf(r.membership), revocation: rev && typeof rev === "object" ? { bulk: String(rev.bulk ?? ""), state: String(rev.state ?? "") } : null };
}

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

export type SecretPickup = "open" | "collected" | "never_collected";
export type CredentialRow = {
  id: string;
  not_before: string;
  not_after: string;
  state: "active" | "expired" | "revoked";
  usable: boolean;
  secret_pickup?: SecretPickup;
  secret_pickup_until?: string;
};
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
  limits: { credential_days: number; credentials_per_service: number; services: number; replay_seconds: number };
  now: string;
};
export type Secret = { secret: string; credential: string };
export type { SelfSession, SelfEnd, SelfEndRow, SelfItem, SelfRef } from "./self-end";
export type { AccessHistoryPage, AccessHistoryQuery, HistoryEvent, HistoryNote } from "./access-history";
export type SelfPage = {
  user_id: string;
  login: string;
  email: string;
  workspace: string;
  role: string;
  identities: { method: string; login: string }[];
  password_set: boolean;
  installation_administrator: boolean;
  current_session: SessionRow;
  recorded_session_count: number;
  accepted_operation_count: number;
  sessions_generation: string;
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
  filters: OrgFilters;
  provider_filter: string;
  scan_stopped: boolean;
  scan_budget: number;
};
export type OrgSubject = "account" | "party" | "any";
export type OrgFilters = { subject: string; entity: string; unattributed: boolean; q: string; member_count: number; target: string };
export type OrgTargetRow = { id: string; name: string; members_on: number };
export type TargetsQuery = { kind: RelationKind; on?: string; revision?: number; after?: string; limit?: number; q?: string };
export type TargetsPage = { revision: string; relationship_revision: number; active_revision: number; on: string; kind: RelationKind; targets: OrgTargetRow[]; next: string; scan_stopped: boolean };
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
  draft: PlanDraft | null;
};
export type PlanDraft = { id: string; name: string; group: string; new_group: boolean; member_count: number };
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
    draft: planDraftOf(r.draft),
  };
}

function planDraftOf(v: unknown): PlanDraft | null {
  if (!v || typeof v !== "object") return null;
  const d = v as Record<string, unknown>;
  return { id: String(d.id ?? ""), name: String(d.name ?? ""), group: String(d.group ?? ""), new_group: d.new_group === true, member_count: Number(d.member_count ?? 0) };
}

export type DraftState = "open" | "proposed" | "activated" | "discarded" | "unknown";
export type DraftPlan = { id: string; state: string; closed: string; blocked: string; rule: string };
export type DraftMember = { id: string; name: string };
export type GroupDraft = {
  id: string; name: string; group: string; group_label: string; new_group: boolean; member_count: number; members: DraftMember[];
  author: string; author_login: string; created: string; updated: string; discarded: string; state: DraftState; activated_revision: number; pruned: boolean; plan: DraftPlan | null;
};
export type DraftsPage = { revision: string; active_revision: number; settings: AccessSetting[]; open: number; open_limit: number; drafts: GroupDraft[]; next: string };
export type DraftMembersPage = { revision: string; draft: string; member_count: number; pruned: boolean; members: DraftMember[]; next: string };

const draftStates: DraftState[] = ["open", "proposed", "activated", "discarded"];
const memberList = (v: unknown): DraftMember[] =>
  (Array.isArray(v) ? v : []).filter((m) => !!m && typeof m === "object").map((m: Record<string, unknown>) => ({ id: String(m.id ?? ""), name: String(m.name ?? "") }));

export function groupDraftOf(v: unknown): GroupDraft {
  const r = (v && typeof v === "object" ? v : {}) as Record<string, unknown>;
  const p = r.plan && typeof r.plan === "object" ? (r.plan as Record<string, unknown>) : null;
  const state = String(r.state ?? "") as DraftState;
  return {
    id: String(r.id ?? ""), name: String(r.name ?? ""), group: String(r.group ?? ""), group_label: String(r.group_label ?? ""), new_group: r.new_group === true,
    member_count: Number(r.member_count ?? 0), members: memberList(r.members), author: String(r.author ?? ""), author_login: String(r.author_login ?? ""),
    created: String(r.created ?? ""), updated: String(r.updated ?? ""), discarded: String(r.discarded ?? ""), state: draftStates.includes(state) ? state : "unknown",
    activated_revision: Number(r.activated_revision ?? 0), pruned: r.pruned === true,
    plan: p && { id: String(p.id ?? ""), state: String(p.state ?? ""), closed: String(p.closed ?? ""), blocked: String(p.blocked ?? ""), rule: String(p.rule ?? "") },
  };
}

export type OrgQuery = {
  on?: string; kind?: RelationKind; revision?: number; after?: string; limit?: number;
  subject?: OrgSubject; entity?: string; unattributed?: boolean; q?: string; member?: string[]; target?: string;
};

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
  stricter_fields: string[] | null;
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
  impact: TrustImpact | null;
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
  const imp = r.impact && typeof r.impact === "object" && !Array.isArray(r.impact) ? obj(r.impact) : null;
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
    impact: imp && { members: num(imp.members), sessions: num(imp.sessions), affected_ids: list(imp.affected_ids).map(String), weakens_mfa: imp.weakens_mfa === true, signs_out_everyone: imp.signs_out_everyone !== false },
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
      stricter_fields: Array.isArray(t.stricter_fields) && t.stricter_fields.every((value): value is string => typeof value === "string") ? t.stricter_fields : null,
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
    readonly request = "",
  ) {
    super(failure);
  }
}

export type CommandId =
  | "input.inspect"
  | "input.confirm_publish"
  | "input.reject"
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
  | "identity.set_role_period"
  | "bulk_revocation.add_targets"
  | "bulk_revocation.add_matching"
  | "bulk_revocation.prepare"
  | "bulk_revocation.execute"
  | "bulk_revocation.discard"
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
  | "view_group.draft"
  | "view_group.edit"
  | "view_group.discard_draft"
  | "view_group.propose"
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
  | "integrations.subscription_withdraw"
  | "integrations.subscription_withdraw_resume"
  | "integrations.letter_withdraw_redeliver"
  | "integrations.letter_withdraw_dismiss"
  | "integrations.redelivery_reapprove"
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

export type Outcome = { status: string; state: string; job_id: string; revision?: string; fields: Record<string, string> };

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

export function credentialRowOf(raw: unknown): CredentialRow {
  return validatedRead(() => credentialOf(raw));
}

function validatedRead<T>(parse: () => T): T {
  try {
    return parse();
  } catch (error) {
    if (error instanceof ReadParseError) throw new AdminError("invalid");
    throw error;
  }
}

export function outcomeOf(body: Record<string, unknown>): Outcome {
  const fields = (body.fields ?? {}) as Record<string, unknown>;
  return {
    status: String(body.status ?? ""),
    state: String(body.business_state ?? ""),
    job_id: String(body.job_id ?? ""),
    ...(typeof body.result_revision === "string" && body.result_revision ? { revision: body.result_revision } : {}),
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
  const valid = body && typeof body === "object" && !Array.isArray(body) ? body : {};
  const code = typeof valid.code === "string" && /^[a-z][a-z0-9_]{0,127}$/.test(valid.code) ? valid.code : "";
  const request = typeof valid.request_id === "string" && /^[A-Za-z0-9._:-]{1,128}$/.test(valid.request_id) ? valid.request_id : "";
  body = valid;
  const fields = Array.isArray(body.fields) ? body.fields.map(String) : [];
  const operation = typeof body.operation_id === "string" ? body.operation_id : "";
  const failure = codes[code] ?? (status === 401 ? "unauthenticated" : status === 429 ? "limited" : "unavailable");
  if (failure === "partial" && !operation) return new AdminError("unavailable", [], 0, "", code, request);
  return new AdminError(failure, fields, 0, operation, code, request);
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
    const parsed: unknown = await r.json();
    body = parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed as Record<string, unknown> : {};
  } catch {
    body = {};
  }
  throw failureOf(r.status, body);
}

export function adminClient(csrf: string, f: Fetch = fetch.bind(globalThis), stepUp?: StepUp) {
  let authenticationLost = false;
  const authenticationListeners = new Set<() => void>();
  const signalAuthenticationLoss = () => {
    if (authenticationLost) return;
    authenticationLost = true;
    for (const listener of authenticationListeners) listener();
  };
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
    if (r.status === 401) signalAuthenticationLoss();
    let body: Record<string, unknown> = {};
    try {
      const parsed: unknown = await r.json();
      body = parsed && typeof parsed === "object" && !Array.isArray(parsed) ? parsed as Record<string, unknown> : {};
    } catch {
      body = {};
    }
    if (!r.ok && failureOf(r.status, body).failure === "unauthenticated") signalAuthenticationLoss();
    return { r, body };
  }
  async function send(path: string, init: RequestInit, request: string): Promise<Record<string, unknown>> {
    if (authenticationLost) throw new AdminError("unauthenticated");
    let { r, body } = await exchange(path, init, request);
    if (r.status === 401) throw failureOf(r.status, body);
    if (!r.ok && init.method === "POST" && body.code === "reauthentication_required" && stepUp) {
      if (authenticationLost) throw new AdminError("unauthenticated");
      if (!(await stepUp())) {
        const challenge = failureOf(r.status, body);
        throw new AdminError("reauth_cancelled", [], 0, "", challenge.code, challenge.request);
      }
      if (authenticationLost) throw new AdminError("unauthenticated");
      ({ r, body } = await exchange(path, init, request));
    }
    if (r.ok) return body;
    const failed = failureOf(r.status, body);
    const retry = Number(r.headers.get("Retry-After") ?? "0");
    if (failed.failure === "limited" && Number.isFinite(retry) && retry > 0) {
      throw new AdminError("limited", failed.fields, retry, failed.operation, failed.code, failed.request);
    }
    throw failed;
  }
  return {
    subscribeAuthenticationLoss(listener: () => void) {
      authenticationListeners.add(listener);
      if (authenticationLost) listener();
      return () => { authenticationListeners.delete(listener); };
    },
    ...sessionQueryClient(send, () => attempt().request, validatedRead),
    async users(query: UsersQuery = {}): Promise<UsersPage> {
      const body = await send("/identity/users?" + usersQueryString(query), { method: "GET" }, attempt().request);
      const methods = (body.auth_methods ?? {}) as Record<string, unknown>;
      return {
        revision: String(body.revision ?? ""),
        next: String(body.next ?? ""),
        next_login: String(body.next_login ?? ""),
        users: Array.isArray(body.users) ? body.users.map(userRowOf) : [],
        clock: clockOf(body),
        settings: settingsOf(body.settings),
        ledger_state: String(body.ledger_state ?? ""),
        can_bulk_revoke: body.can_bulk_revoke === true,
        filters: filterOf(body.filters),
        scan_stopped: body.scan_stopped === true,
        scan_budget: Number(body.scan_budget ?? 0),
        order: String(body.order ?? ""),
        list_version: String(body.list_version ?? ""),
        auth_methods: { connected: names(methods.connected), not_connected: names(methods.not_connected) },
        warning_days: Number(body.warning_days ?? 0),
      };
    },
    async read(path: string): Promise<Record<string, unknown>> {
      return send(path, { method: "GET" }, attempt().request);
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
      return validatedRead(() => ({
        revision: readText(body.revision), grants: grantsOf(body.grants), delegations: delegationsOf(body.delegations),
        catalog: catalogOf(body.catalog), subjects: subjectsOf(body.subjects), now: readText(body.now), you: readText(body.you),
        started_at: readText(body.started_at), settings: accessSettingsOf(body.settings), requests: requestsOf(body.requests),
        revocations: revocationsOf(body.revocations), ledger: ledgerOf(body.ledger),
      }));
    },
    async preview(subject: string, kind: string, customer = ""): Promise<Preview[]> {
      const q = new URLSearchParams({ subject, kind });
      if (customer) q.set("customer", customer);
      const body = await send("/identity/access/preview?" + q.toString(), { method: "GET" }, attempt().request);
      return validatedRead(() => previewsOf(body.decisions));
    },
    async services(): Promise<ServicesPage> {
      const body = await send("/identity/services", { method: "GET" }, attempt().request);
      return validatedRead(() => servicesPageOf(body));
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
      return validatedRead(() => selfPageOf(body));
    },
    async accessHistory(query: AccessHistoryQuery, request: string): Promise<AccessHistoryPage> {
      const q = new URLSearchParams();
      if (query.from) q.set("from", query.from);
      if (query.before) q.set("before", query.before);
      const tail = q.toString();
      const body = await send("/me/access-history" + (tail ? "?" + tail : ""), { method: "GET" }, request);
      try {
        return accessHistoryOf(body);
      } catch (err) {
        if (err instanceof HistoryParseError) throw new AdminError("invalid");
        throw err;
      }
    },
    async changePassword(current: string, next: string): Promise<void> {
      await send("/me/password", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ current, new: next }) }, attempt().request);
    },
    async endOthers(generation: string, handles: string[]): Promise<SelfEnd> {
      const body = await send("/me/sessions/end", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ session: "others", generation, handles }) },
        attempt().request);
      return selfEndOf(body);
    },
    async endOne(handle: string): Promise<SelfEnd> {
      const body = await send("/me/sessions/end", { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ session: handle }) }, attempt().request);
      return selfEndOf(body);
    },
    async organizations(query: OrgQuery = {}): Promise<OrgPage> {
      const q = new URLSearchParams();
      if (query.on) q.set("on", query.on);
      if (query.kind) q.set("kind", query.kind);
      if (query.revision !== undefined) q.set("revision", String(query.revision));
      if (query.after) q.set("after", query.after);
      if (query.limit) q.set("limit", String(query.limit));
      if (query.subject) q.set("subject", query.subject);
      if (query.entity) q.set("entity", query.entity);
      if (query.unattributed) q.set("unattributed", "true");
      if (query.q) q.set("q", query.q);
      for (const id of query.member ?? []) q.append("member", id);
      if (query.target) q.set("target", query.target);
      const qs = q.toString();
      const body = await send("/organizations" + (qs ? "?" + qs : ""), { method: "GET" }, attempt().request);
      return validatedRead(() => organizationsPageOf(body));
    },
    async targets(query: TargetsQuery): Promise<TargetsPage> {
      const q = new URLSearchParams({ kind: query.kind });
      if (query.on) q.set("on", query.on);
      if (query.revision !== undefined) q.set("revision", String(query.revision));
      if (query.after) q.set("after", query.after);
      if (query.limit) q.set("limit", String(query.limit));
      if (query.q) q.set("q", query.q);
      const body = await send("/organizations/targets?" + q.toString(), { method: "GET" }, attempt().request);
      return {
        revision: String(body.revision ?? ""),
        relationship_revision: Number(body.relationship_revision ?? 0),
        active_revision: Number(body.active_revision ?? 0),
        on: String(body.on ?? ""),
        kind: String(body.kind ?? query.kind) as RelationKind,
        targets: (Array.isArray(body.targets) ? (body.targets as Record<string, unknown>[]) : []).map((t) => ({ id: String(t?.id ?? ""), name: String(t?.name ?? ""), members_on: Number(t?.members_on ?? 0) })),
        next: String(body.next ?? ""),
        scan_stopped: body.scan_stopped === true,
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
    async viewGroupDrafts(after = "", limit = 0): Promise<DraftsPage> {
      const q = new URLSearchParams();
      if (after) q.set("after", after);
      if (limit) q.set("limit", String(limit));
      const qs = q.toString();
      const body = await send("/organizations/view-group-drafts" + (qs ? "?" + qs : ""), { method: "GET" }, attempt().request);
      return {
        revision: String(body.revision ?? ""),
        active_revision: Number(body.active_revision ?? 0),
        settings: Array.isArray(body.settings) ? (body.settings as AccessSetting[]) : [],
        open: Number(body.open ?? 0),
        open_limit: Number(body.open_limit ?? 0),
        drafts: (Array.isArray(body.drafts) ? body.drafts : []).map(groupDraftOf),
        next: String(body.next ?? ""),
      };
    },
    async draftMembers(id: string, after = "", limit = 0): Promise<DraftMembersPage | "missing"> {
      const q = new URLSearchParams();
      if (after) q.set("after", after);
      if (limit) q.set("limit", String(limit));
      const qs = q.toString();
      try {
        const body = await send("/organizations/view-group-drafts/" + encodeURIComponent(id) + "/members" + (qs ? "?" + qs : ""), { method: "GET" }, attempt().request);
        return { revision: String(body.revision ?? ""), draft: String(body.draft ?? ""), member_count: Number(body.member_count ?? 0), pruned: body.pruned === true, members: memberList(body.members), next: String(body.next ?? "") };
      } catch (e) {
        if (e instanceof AdminError && e.code === "unsupported_command") return "missing";
        throw e;
      }
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
