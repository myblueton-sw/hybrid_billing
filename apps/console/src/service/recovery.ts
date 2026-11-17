import { count, hint } from "./integrations";

export type DomainCheck = { domain: string; status: string; mismatch_chunks: number[] | null };
export type OpenRecovery = {
  state: string;
  id: string;
  phase: string;
  as_of: string;
  point_generation: number | null;
  backup_reference: string;
  manifest_digest: string;
  domains: DomainCheck[] | null;
};
export type RecoveryHistory = { recovery: string; manifest_digest: string; generation: number | null; outcome: string; lost_count: number | null };
export type SetRow = {
  installation: string;
  state: string;
  generation: number | null;
  recovery: OpenRecovery;
  history: RecoveryHistory[] | null;
  history_total: number | null;
  begin: string;
  abandon: string;
};
export type OpenCall = { id: string; writer: string; deadline: string; key: string; series: string; number: number | null };
export type LockEntry = { by: string; by_login: string; rationale: string; at: string };
export type IsolationRow = {
  installation: string;
  state: string;
  generation: number | null;
  stopped_at: string;
  can_lock: boolean;
  lock_blocked: string;
  writers: string[] | null;
  verifier: string;
  open_calls_source: string;
  open_calls: OpenCall[] | null;
  open_calls_total: number | null;
  other_scope_present: boolean | null;
  permits: string;
  locks: LockEntry[] | null;
};
export type ExternalSystem = { id: string; kind: string; status: string };
export type UnknownTarget = { key: string; series: string; number: number | null; fenced: boolean | null; high_water: number | null };
export type ReconciliationRow = {
  installation: string;
  externals: ExternalSystem[] | null;
  unknown_targets: UnknownTarget[] | null;
  unknown_total: number | null;
  other_scope_present: boolean | null;
  skipped: number | null;
  reconciler: string;
};
export type DecisionEvent = { sequence: number | null; kind: string; domain: string; at: string };
export type ResumeRow = {
  installation: string;
  mode: string;
  phase: string;
  pending_events: DecisionEvent[] | null;
  pending_total: number | null;
  lost: number[] | null;
  lost_digest: string;
  can_propose_unlock: boolean;
  unlock_blocked: string;
};
export type RecoverySummary = {
  installation: string;
  skew_seconds: number | null;
  writer: string;
  external: string;
  external_kind: string;
  verifier: string;
  as_of: string;
  point_generation: number | null;
  manifest_digest: string;
  unlock_generation: number | null;
  lost_digest: string;
};
export type RecoveryProposal = {
  number: number | null;
  target: string;
  kind: string;
  installation: string;
  state: string;
  created_at: string;
  proposer: string;
  proposer_login: string;
  approver: string;
  approver_login: string;
  rationale: string;
  closed_by: string;
  closed_by_login: string;
  closing_rationale: string;
  summary: RecoverySummary;
  can_approve: boolean;
  approve_blocked: string;
  can_withdraw: boolean;
  withdraw_blocked: string;
};
export type Section<T> = { state: string; rows: T[] | null };
export type MigrationSection = { state: string; modules: string[] | null };
export type RecoveryPage = {
  revision: string;
  now: string;
  state: string;
  sets: Section<SetRow>;
  isolation: Section<IsolationRow>;
  reconciliation: Section<ReconciliationRow>;
  migration: MigrationSection;
  resume: Section<ResumeRow>;
  proposals: RecoveryProposal[] | null;
};

type Raw = Record<string, unknown>;

const isRow = (x: unknown): x is Raw => !!x && typeof x === "object" && !Array.isArray(x);
const obj = (v: unknown): Raw => (isRow(v) ? v : {});
const str = (v: unknown): string => (typeof v === "string" ? v : "");
const flag = (v: unknown): boolean | null => (typeof v === "boolean" ? v : null);
const strs = (v: unknown): string[] | null => (Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : null);
const rows = <T,>(v: unknown, f: (r: Raw) => T): T[] | null => (Array.isArray(v) ? v.filter(isRow).map(f) : null);

export function signed(v: unknown): number | null {
  return typeof v === "number" && Number.isSafeInteger(v) ? v : null;
}

export function counts(v: unknown): number[] | null {
  if (!Array.isArray(v)) return null;
  const out: number[] = [];
  for (const x of v) {
    const n = count(x);
    if (n === null) return null;
    out.push(n);
  }
  return out;
}

export function openRecoveryOf(v: unknown): OpenRecovery {
  const r = obj(v);
  const state = str(r.state);
  if (state !== "open") return { state, id: "", phase: "", as_of: "", point_generation: null, backup_reference: "", manifest_digest: "", domains: state === "none" ? [] : null };
  return {
    state,
    id: str(r.id),
    phase: str(r.phase),
    as_of: str(r.as_of),
    point_generation: count(r.point_generation),
    backup_reference: str(r.backup_reference),
    manifest_digest: str(r.manifest_digest),
    domains: rows(r.domains, (d) => ({ domain: str(d.domain), status: str(d.status), mismatch_chunks: counts(d.mismatch_chunks) })),
  };
}

export function setRowOf(r: Raw): SetRow {
  return {
    installation: str(r.installation),
    state: str(r.state),
    generation: count(r.generation),
    recovery: openRecoveryOf(r.recovery),
    history: rows(r.history, (h) => ({
      recovery: str(h.recovery),
      manifest_digest: str(h.manifest_digest),
      generation: count(h.generation),
      outcome: str(h.outcome),
      lost_count: count(h.lost_count),
    })),
    history_total: count(r.history_total),
    begin: str(r.begin),
    abandon: str(r.abandon),
  };
}

export function isolationRowOf(r: Raw): IsolationRow {
  const lock = hint(r.can_lock, r.lock_blocked);
  return {
    installation: str(r.installation),
    state: str(r.state),
    generation: count(r.generation),
    stopped_at: str(r.stopped_at),
    can_lock: lock.allowed,
    lock_blocked: lock.blocked,
    writers: strs(r.writers),
    verifier: str(r.verifier),
    open_calls_source: str(r.open_calls_source),
    open_calls: rows(r.open_calls, (c) => ({ id: str(c.id), writer: str(c.writer), deadline: str(c.deadline), key: str(c.key), series: str(c.series), number: signed(c.number) })),
    open_calls_total: count(r.open_calls_total),
    other_scope_present: flag(r.other_scope_present),
    permits: str(r.permits),
    locks: rows(r.locks, (x) => ({ by: str(x.by), by_login: str(x.by_login), rationale: str(x.rationale), at: str(x.at) })),
  };
}

export function reconciliationRowOf(r: Raw): ReconciliationRow {
  return {
    installation: str(r.installation),
    externals: rows(r.externals, (x) => ({ id: str(x.id), kind: str(x.kind), status: str(x.status) })),
    unknown_targets: rows(r.unknown_targets, (t) => ({ key: str(t.key), series: str(t.series), number: signed(t.number), fenced: flag(t.fenced), high_water: signed(t.high_water) })),
    unknown_total: count(r.unknown_total),
    other_scope_present: flag(r.other_scope_present),
    skipped: count(r.skipped),
    reconciler: str(r.reconciler),
  };
}

export function resumeRowOf(r: Raw): ResumeRow {
  const unlock = hint(r.can_propose_unlock, r.unlock_blocked);
  return {
    installation: str(r.installation),
    mode: str(r.mode),
    phase: str(r.phase),
    pending_events: rows(r.pending_events, (e) => ({ sequence: count(e.sequence), kind: str(e.kind), domain: str(e.domain), at: str(e.at) })),
    pending_total: count(r.pending_total),
    lost: counts(r.lost),
    lost_digest: str(r.lost_digest),
    can_propose_unlock: unlock.allowed,
    unlock_blocked: unlock.blocked,
  };
}

const summaryFields: Record<string, (keyof RecoverySummary)[]> = {
  installation: ["skew_seconds"],
  writer: ["writer"],
  external: ["external", "external_kind"],
  verifier: ["verifier"],
  begin: ["as_of", "point_generation", "manifest_digest"],
  unlock: ["unlock_generation", "lost_digest"],
};

export function recoverySummaryOf(v: unknown, kind: string): RecoverySummary {
  const r = obj(v);
  const own = summaryFields[kind] ?? [];
  const pick = <K extends keyof RecoverySummary>(key: K, read: (x: unknown) => RecoverySummary[K], empty: RecoverySummary[K]): RecoverySummary[K] => (own.includes(key) ? read(r[key]) : empty);
  return {
    installation: str(r.installation),
    skew_seconds: pick("skew_seconds", count, null),
    writer: pick("writer", str, ""),
    external: pick("external", str, ""),
    external_kind: pick("external_kind", str, ""),
    verifier: pick("verifier", str, ""),
    as_of: pick("as_of", str, ""),
    point_generation: pick("point_generation", count, null),
    manifest_digest: pick("manifest_digest", str, ""),
    unlock_generation: pick("unlock_generation", count, null),
    lost_digest: pick("lost_digest", str, ""),
  };
}

export function recoveryProposalOf(r: Raw): RecoveryProposal {
  const approve = hint(r.can_approve, r.approve_blocked);
  const withdraw = hint(r.can_withdraw, r.withdraw_blocked);
  const n = count(r.number);
  const kind = str(r.kind);
  return {
    number: n !== null && n > 0 ? n : null,
    target: str(r.target),
    kind,
    installation: str(r.installation),
    state: str(r.state),
    created_at: str(r.created_at),
    proposer: str(r.proposer),
    proposer_login: str(r.proposer_login),
    approver: str(r.approver),
    approver_login: str(r.approver_login),
    rationale: str(r.rationale),
    closed_by: str(r.closed_by),
    closed_by_login: str(r.closed_by_login),
    closing_rationale: str(r.closing_rationale),
    summary: recoverySummaryOf(r.summary, kind),
    can_approve: approve.allowed,
    approve_blocked: approve.blocked,
    can_withdraw: withdraw.allowed,
    withdraw_blocked: withdraw.blocked,
  };
}

function sectionOf<T>(v: unknown, f: (r: Raw) => T): Section<T> {
  const r = obj(v);
  return { state: str(r.state), rows: rows(r.rows, f) };
}

export function recoveryOf(body: Raw): RecoveryPage {
  const m = obj(body.migration);
  return {
    revision: str(body.revision),
    now: str(body.now),
    state: str(body.state),
    sets: sectionOf(body.sets, setRowOf),
    isolation: sectionOf(body.isolation, isolationRowOf),
    reconciliation: sectionOf(body.reconciliation, reconciliationRowOf),
    migration: { state: str(m.state), modules: strs(m.modules) },
    resume: sectionOf(body.resume, resumeRowOf),
    proposals: rows(body.proposals, recoveryProposalOf),
  };
}
