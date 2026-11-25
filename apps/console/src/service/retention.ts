import { keptFactsOf, type KeptFacts } from "./kept";
import { count, hint } from "./integrations";

export type Person = { id: string; login: string } | null;
export type RunRef = { target: string; label: string } | null;
export type Counts = Record<string, number | null>;

export type PolicyView = {
  revision: number | null;
  period_days: number | null;
  grace_days: number | null;
  proposer: Person;
  approver: Person;
  approved_at: string;
};
export type PolicyRow = {
  class: string;
  target: string;
  policy: PolicyView | null;
  origin: string;
  scope: string;
  archive_rule: string;
  pending: number | null;
  can_propose: boolean;
  propose_blocked: string;
};

export type ArtifactRow = {
  id: string;
  anchor: string;
  due_at: string | null;
  stores: string[] | null;
  access: string;
  standing: string;
  held: boolean | null;
  damaged: boolean | null;
  run: RunRef;
  other_scope: boolean | null;
  shared: boolean | null;
  decision: string;
  holdable: boolean;
  hold_blocked: string;
  hold_target: string;
};
export type ClassGroup = {
  class: string;
  population: string;
  counts: Counts;
  total: number | null;
  more: boolean | null;
  next: string | null;
  artifacts: ArtifactRow[] | null;
  usage: string;
  growth: string;
  bytes: string;
};

export type StoreResult = { store: string; result: string };
export type HoldOutcome = { artifact: string; outcome: string; stores: StoreResult[] | null };
export type HoldScope = { artifacts: string[] | null; workspace: boolean | null; classes: string[] | null };
export type HoldRow = {
  number: number | null;
  id: string;
  target: string;
  owner: Person;
  reason: string;
  placer: Person;
  placed_at: string;
  rationale: string;
  released: boolean | null;
  scope: HoldScope;
  counts: Counts | null;
  counts_scope: string;
  outcomes: HoldOutcome[] | null;
  outcomes_more: boolean | null;
  outcomes_next: string | null;
  release: number | null;
  can_propose_release: boolean;
  release_blocked: string;
};
export type Reference = { id: string; class: string };
export type References = { population: string; total: number | null; more: boolean | null; next: string | null; rows: Reference[] | null };
export type HoldsSection = { total: number | null; more: boolean | null; rows: HoldRow[] | null; released: { total: number; next: string } | null; tenant_target: string; references: References };

export type StoreRow = { store: string; target: number | null; deleted: number | null; failed: number | null; excluded: number | null; pending: number | null };
export type RunItem = { id: string; outcome: string };
export type RunRow = {
  number: number | null;
  run: RunRef;
  class: string;
  revision: number | null;
  state: string;
  decision: string;
  next_chunk: number | null;
  chunks: number | null;
  chunk_size: number | null;
  lease_seconds: number | null;
  reason: string;
  damaged: number | null;
  proposer: Person;
  approver: Person;
  stopper: Person;
  stopper_known: boolean;
  counts: Counts;
  stores: StoreRow[] | null;
  artifacts: { total: number | null; more: boolean | null; next: string | null; rows: RunItem[] | null };
  can_stop: boolean;
  stop_blocked: string;
};
export type DeletionSection = {
  total: number | null;
  more: boolean | null;
  rows: RunRow[] | null;
  finished: { total: number; next: string } | null;
  worker: string;
  retry: string;
  backup_expiry: string;
  propose_plan: string;
};

export type Tombstone = { sequence: number | null; artifact: string; class: string; run: RunRef; other_scope: boolean | null; at: string; decision: string };
export type DecisionsSection = {
  watermark: number | null;
  tombstones: { window: number | null; total: number | null; more: boolean | null; next: string | null; rows: Tombstone[] | null };
  ledger: string;
  epoch: string;
  durable_commit: string;
  restore_check: string;
};

export type ImpactAt = { as_of: string; count: number | null; digest: string };
export type ImpactNow = ImpactAt & { workspace_count: number | null };
export type PolicySummary = { class: string; period_days: number | null; grace_days: number | null; impact_at_proposal: ImpactAt | null };
export type PlanSummary = { class: string; artifact_count: number | null; chunk_size: number | null; lease_seconds: number | null; reason: string };
export type ReleaseSummary = { hold: string; reason: string };
export type Proposal = {
  number: number | null;
  target: string;
  kind: string;
  state: string;
  created_at: string;
  proposer: Person;
  approver: Person;
  rationale: string;
  closed_by: Person;
  closing_rationale: string;
  policy: PolicySummary | null;
  plan: PlanSummary | null;
  release: ReleaseSummary | null;
  impact_required: boolean | null;
  impact_now: ImpactNow | null | "invalid";
  can_approve: boolean;
  approve_blocked: string;
  can_withdraw: boolean;
  withdraw_blocked: string;
};
export type ProposalsSection = { total: number | null; more: boolean | null; rows: Proposal[] | null; history: KeptFacts | null; closed_by_kind: Record<string, number> | null };

export type RetentionPage = {
  revision: string;
  now: string;
  state: string;
  policies: PolicyRow[] | null;
  expiry: ClassGroup[] | null;
  holds: HoldsSection;
  deletion: DeletionSection;
  restore: { state: string };
  decisions: DecisionsSection;
  proposals: ProposalsSection;
};

export const standingKeys = ["eligible", "held", "guarding", "referenced", "not_due", "no_policy", "in_run", "deleted"];
export const runOutcomeKeys = ["pending", "deleting", "deleted", "failed", "held", "guarding", "referenced", "not_due", "stopped"];
export const holdCountKeys = ["not_preserved", "outcome_pending", "partially_deleted"];

type Raw = Record<string, unknown>;

const isRow = (x: unknown): x is Raw => !!x && typeof x === "object" && !Array.isArray(x);
const obj = (v: unknown): Raw => (isRow(v) ? v : {});
const str = (v: unknown): string => (typeof v === "string" ? v : "");
const flag = (v: unknown): boolean | null => (typeof v === "boolean" ? v : null);
const strs = (v: unknown): string[] | null => (Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : null);
const rows = <T,>(v: unknown, f: (r: Raw) => T): T[] | null => (Array.isArray(v) ? v.filter(isRow).map(f) : null);

export function personOf(v: unknown): Person {
  if (!isRow(v) || typeof v.id !== "string" || !v.id) return null;
  return { id: v.id, login: str(v.login) };
}

export function runRefOf(v: unknown): RunRef {
  if (!isRow(v)) return null;
  return { target: str(v.target), label: str(v.label) };
}

export function countsOf(v: unknown, keys: string[]): Counts {
  const r = obj(v);
  return Object.fromEntries(keys.map((key) => [key, count(r[key])]));
}

export function policyViewOf(v: unknown): PolicyView | null {
  if (v === null) return null;
  const r = obj(v);
  return {
    revision: count(r.revision),
    period_days: count(r.period_days),
    grace_days: count(r.grace_days),
    proposer: personOf(r.proposer),
    approver: personOf(r.approver),
    approved_at: str(r.approved_at),
  };
}

export function policyRowOf(r: Raw): PolicyRow {
  const propose = hint(r.can_propose, r.propose_blocked);
  return {
    class: str(r.class),
    target: str(r.target),
    policy: policyViewOf(r.policy),
    origin: str(r.origin),
    scope: str(r.scope),
    archive_rule: str(r.archive_rule),
    pending: count(r.pending),
    can_propose: propose.allowed,
    propose_blocked: propose.blocked,
  };
}

export function dueOf(v: unknown): string | null {
  if (v === null) return null;
  return str(v);
}

export function artifactRowOf(r: Raw): ArtifactRow {
  const hold = hint(r.holdable, r.hold_blocked);
  return {
    id: str(r.id),
    anchor: str(r.anchor),
    due_at: dueOf(r.due_at),
    stores: strs(r.stores),
    access: str(r.access),
    standing: str(r.standing),
    held: flag(r.held),
    damaged: flag(r.damaged),
    run: runRefOf(r.run),
    other_scope: flag(r.other_scope),
    shared: flag(r.shared),
    decision: str(r.decision),
    holdable: hold.allowed,
    hold_blocked: hold.blocked,
    hold_target: str(r.hold_target),
  };
}

export function classGroupOf(r: Raw): ClassGroup {
  return {
    class: str(r.class),
    population: str(r.population),
    counts: countsOf(r.counts, standingKeys),
    total: count(r.total),
    more: flag(r.more),
    next: typeof r.next === "string" ? r.next : null,
    artifacts: rows(r.artifacts, artifactRowOf),
    usage: str(r.usage),
    growth: str(r.growth),
    bytes: str(r.bytes),
  };
}

export function storeResultsOf(v: unknown): StoreResult[] | null {
  if (!isRow(v)) return null;
  return Object.keys(v)
    .sort()
    .map((store) => ({ store, result: str(v[store]) }));
}

export function holdRowOf(r: Raw): HoldRow {
  const release = hint(r.can_propose_release, r.release_blocked);
  const scope = obj(r.scope);
  const countsScope = str(r.counts_scope);
  return {
    number: count(r.number),
    id: str(r.id),
    target: str(r.target),
    owner: personOf(r.owner),
    reason: str(r.reason),
    placer: personOf(r.placer),
    placed_at: str(r.placed_at),
    rationale: str(r.rationale),
    released: flag(r.released),
    scope: { artifacts: strs(scope.artifacts), workspace: flag(scope.workspace), classes: strs(scope.classes) },
    counts: isRow(r.counts) ? countsOf(r.counts, holdCountKeys) : null,
    counts_scope: countsScope,
    outcomes: rows(r.outcomes, outcomeOf),
    outcomes_more: flag(r.outcomes_more),
    outcomes_next: typeof r.outcomes_next === "string" ? r.outcomes_next : null,
    release: count(r.release),
    can_propose_release: release.allowed,
    release_blocked: release.blocked,
  };
}

export function holdsOf(v: unknown): HoldsSection {
  const r = obj(v);
  const ref = obj(r.references);
  return {
    total: count(r.total),
    more: flag(r.more),
    rows: rows(r.rows, holdRowOf),
    released: releasedFactsOf(r.released),
    tenant_target: str(r.tenant_target),
    references: { population: str(ref.population), total: count(ref.total), more: flag(ref.more), next: typeof ref.next === "string" ? ref.next : null, rows: rows(ref.rows, referenceOf) },
  };
}

export function runRowOf(r: Raw): RunRow {
  const stop = hint(r.can_stop, r.stop_blocked);
  const items = obj(r.artifacts);
  return {
    number: count(r.number),
    run: runRefOf(r.run),
    class: str(r.class),
    revision: count(r.revision),
    state: str(r.state),
    decision: str(r.decision),
    next_chunk: count(r.next_chunk),
    chunks: count(r.chunks),
    chunk_size: count(r.chunk_size),
    lease_seconds: count(r.lease_seconds),
    reason: str(r.reason),
    damaged: count(r.damaged),
    proposer: personOf(r.proposer),
    approver: personOf(r.approver),
    stopper: personOf(r.stopper),
    stopper_known: r.stopper === null || personOf(r.stopper) !== null,
    counts: countsOf(r.counts, runOutcomeKeys),
    stores: rows(r.stores, (s) => ({ store: str(s.store), target: count(s.target), deleted: count(s.deleted), failed: count(s.failed), excluded: count(s.excluded), pending: count(s.pending) })),
    artifacts: { total: count(items.total), more: flag(items.more), next: typeof items.next === "string" ? items.next : null, rows: rows(items.rows, runItemOf) },
    can_stop: stop.allowed,
    stop_blocked: stop.blocked,
  };
}

export function deletionOf(v: unknown): DeletionSection {
  const r = obj(v);
  return {
    total: count(r.total),
    more: flag(r.more),
    rows: rows(r.rows, runRowOf),
    finished: releasedFactsOf(r.finished),
    worker: str(r.worker),
    retry: str(r.retry),
    backup_expiry: str(r.backup_expiry),
    propose_plan: str(r.propose_plan),
  };
}

export function decisionsOf(v: unknown): DecisionsSection {
  const r = obj(v);
  const t = obj(r.tombstones);
  return {
    watermark: count(r.watermark),
    tombstones: {
      window: count(t.window),
      total: count(t.total),
      more: flag(t.more),
      next: typeof t.next === "string" ? t.next : null,
      rows: rows(t.rows, tombstoneOf),
    },
    ledger: str(r.ledger),
    epoch: str(r.epoch),
    durable_commit: str(r.durable_commit),
    restore_check: str(r.restore_check),
  };
}

export function impactAtOf(v: unknown): ImpactAt | null {
  if (!isRow(v)) return null;
  return { as_of: str(v.as_of), count: count(v.count), digest: str(v.digest) };
}

export function impactNowOf(v: unknown): ImpactNow | null | "invalid" {
  if (v === null) return null;
  if (!isRow(v)) return "invalid";
  return { as_of: str(v.as_of), count: count(v.count), digest: str(v.digest), workspace_count: count(v.workspace_count) };
}

export function proposalOf(r: Raw): Proposal {
  const approve = hint(r.can_approve, r.approve_blocked);
  const withdraw = hint(r.can_withdraw, r.withdraw_blocked);
  const n = count(r.number);
  const kind = str(r.kind);
  const s = obj(r.summary);
  const policy = kind === "policy" && isRow(s.policy) ? s.policy : null;
  const plan = kind === "plan" && isRow(s.plan) ? s.plan : null;
  const release = kind === "release" && isRow(s.release) ? s.release : null;
  return {
    number: n !== null && n > 0 ? n : null,
    target: str(r.target),
    kind,
    state: str(r.state),
    created_at: str(r.created_at),
    proposer: personOf(r.proposer),
    approver: personOf(r.approver),
    rationale: str(r.rationale),
    closed_by: personOf(r.closed_by),
    closing_rationale: str(r.closing_rationale),
    policy: policy ? { class: str(policy.class), period_days: count(policy.period_days), grace_days: count(policy.grace_days), impact_at_proposal: impactAtOf(policy.impact_at_proposal) } : null,
    plan: plan ? { class: str(plan.class), artifact_count: count(plan.artifact_count), chunk_size: count(plan.chunk_size), lease_seconds: count(plan.lease_seconds), reason: str(plan.reason) } : null,
    release: release ? { hold: str(release.hold), reason: str(release.reason) } : null,
    impact_required: flag(r.impact_required),
    impact_now: impactNowOf(r.impact_now),
    can_approve: approve.allowed,
    approve_blocked: approve.blocked,
    can_withdraw: withdraw.allowed,
    withdraw_blocked: withdraw.blocked,
  };
}

export function retentionOf(body: Raw): RetentionPage {
  const p = obj(body.proposals);
  return {
    revision: str(body.revision),
    now: str(body.now),
    state: str(body.state),
    policies: rows(body.policies, policyRowOf),
    expiry: rows(body.expiry, classGroupOf),
    holds: holdsOf(body.holds),
    deletion: deletionOf(body.deletion),
    restore: { state: str(obj(body.restore).state) },
    decisions: decisionsOf(body.decisions),
    proposals: { total: count(p.total), more: flag(p.more), rows: rows(p.rows, proposalOf), history: isRow(p.history) ? keptFactsOf(p.history) : null, closed_by_kind: kindCountsOf(p.closed_by_kind) },
  };
}

export type Owner = { id: string; login: string };

export function ownersOf(subjects: unknown): Owner[] {
  if (!Array.isArray(subjects)) return [];
  return subjects.filter(isRow).flatMap((s) => {
    const id = str(s.user_id);
    return /^[A-Za-z0-9._:-]{1,128}$/.test(id) ? [{ id, login: str(s.login) }] : [];
  });
}

export function kindCountsOf(v: unknown): Record<string, number> | null {
  if (!isRow(v)) return null;
  return Object.fromEntries(Object.entries(v).filter((e): e is [string, number] => typeof e[1] === "number" && Number.isInteger(e[1]) && e[1] >= 0));
}

export function releasedFactsOf(v: unknown): { total: number; next: string } | null {
  if (!isRow(v)) return null;
  const total = count(v.total);
  return total === null || typeof v.next !== "string" ? null : { total, next: v.next };
}

export function outcomeOf(o: Raw): HoldOutcome {
  return { artifact: str(o.artifact), outcome: str(o.outcome), stores: storeResultsOf(o.stores) };
}

export function runItemOf(x: Raw): RunItem {
  return { id: str(x.id), outcome: str(x.outcome) };
}

export function runItemsPageOf(body: Raw): { rows: RunItem[]; next: string } | null {
  if (!Array.isArray(body.artifacts) || typeof body.next !== "string") return null;
  return { rows: body.artifacts.filter(isRow).map(runItemOf), next: body.next };
}

export function outcomesPageOf(body: Raw): { rows: HoldOutcome[]; next: string } | null {
  if (!Array.isArray(body.outcomes) || typeof body.next !== "string") return null;
  return { rows: body.outcomes.filter(isRow).map(outcomeOf), next: body.next };
}

export function referenceOf(x: Raw): Reference {
  return { id: str(x.id), class: str(x.class) };
}

export type ArtifactEntry = { class: string; artifact: ArtifactRow };

export function artifactsPath(query: { cls: string; after: string } | { ids: string[] }): string {
  const q = new URLSearchParams("ids" in query ? { id: query.ids.join(",") } : { class: query.cls, limit: "50", after: query.after });
  return "/retention/artifacts?" + q.toString();
}

export function artifactsPageOf(body: Raw): { rows: ArtifactEntry[]; next: string; total: number } | null {
  const total = count(body.total);
  if (!Array.isArray(body.artifacts) || typeof body.next !== "string" || total === null) return null;
  return { rows: body.artifacts.filter(isRow).map((x) => ({ class: str(x.class), artifact: artifactRowOf(x) })), next: body.next, total };
}

export function referencesPageOf(body: Raw): { rows: Reference[]; next: string; total: number } | null {
  const total = count(body.total);
  if (!Array.isArray(body.references) || typeof body.next !== "string" || total === null) return null;
  return { rows: body.references.filter(isRow).map(referenceOf), next: body.next, total };
}

export function tombstoneOf(x: Raw): Tombstone {
  return { sequence: count(x.sequence), artifact: str(x.artifact), class: str(x.class), run: runRefOf(x.run), other_scope: flag(x.other_scope), at: str(x.at), decision: str(x.decision) };
}

export function tombstonesPageOf(body: Raw): { rows: Tombstone[]; next: string; total: number } | null {
  const total = count(body.total);
  if (!Array.isArray(body.tombstones) || typeof body.next !== "string" || total === null) return null;
  return { rows: body.tombstones.filter(isRow).map(tombstoneOf), next: body.next, total };
}

export function runsPageOf(body: Raw): { rows: RunRow[]; next: string; total: number } | null {
  const total = count(body.total);
  if (!Array.isArray(body.runs) || typeof body.next !== "string" || total === null) return null;
  return { rows: body.runs.filter(isRow).map(runRowOf), next: body.next, total };
}

export function holdsPageOf(body: Raw): { rows: HoldRow[]; next: string; total: number } | null {
  const total = count(body.total);
  if (!Array.isArray(body.holds) || typeof body.next !== "string" || total === null) return null;
  return { rows: body.holds.filter(isRow).map(holdRowOf), next: body.next, total };
}
