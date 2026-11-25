import { count, hint } from "./integrations";

export type Person = { id: string; login: string | null; directory: boolean | null } | null;
export type List<T> = { total: number | null; more: boolean | null; rows: T[] | null };
export type Candidate = { version: string; build: string; release: string } | null;
export type CountOf = { count: number | null; of: number | null } | null;
export type RequirementRef = { label: string; target: string | null; state: string | null };
export type Reconciliation = { period: string; matched: boolean | null; differences: number | null };
export type Problem = { problem: string; subject: string };

export type Columns = {
  design: CountOf;
  implementation: CountOf;
  test: CountOf;
  customer_proof: CountOf;
  failing: number | null;
};

export type PathRow = {
  id: string;
  provider: string;
  contract: string;
  role: string;
  version: string;
  issuance: string;
  template: string | null;
  state: string;
  build_set: boolean;
  requirements: List<RequirementRef>;
  columns: Columns;
  categories: Record<string, boolean | null> | null;
  reconciliations: List<Reconciliation>;
  last_observed: string | null;
  blockers: Problem[] | null;
  activation: string;
  support_decision: string;
  customer_proof_intake: string;
};

export type TestRow = {
  id: string;
  category: string | null;
  counted: boolean | null;
  latest_outcome: string | null;
  passes: number | null;
  failures: number | null;
  not_run_reason: string | null;
  not_counted_reason: string | null;
};

export type TargetRow = { target: string; label: string };

export type RequirementRow = {
  target: string;
  label: string;
  namespace: string;
  id: string;
  source: string;
  owner: string;
  interfaces: string[] | null;
  common: boolean | null;
  blocking: boolean | null;
  design: string;
  commit: string | null;
  retired: boolean | null;
  state: string | null;
  replacement: string | null;
  states: Record<string, string> | null;
  builds_more: boolean | null;
  tests: List<TestRow> | null;
  retire: number | null;
  replacements: List<TargetRow>;
  replacement_required: boolean | null;
  can_retire: boolean;
  retire_blocked: string;
};

export type Gate = {
  ready: boolean | null;
  problems: List<Problem>;
  retired_blocking: string[] | null;
  templates: string[] | null;
  release: string;
} | null;

export type BuildRow = { build: string; commit: string; order: number | null };
export type HarnessRow = { module: string; id: string; revocation: string };
export type ResultRow = {
  namespace: string;
  test: string;
  build: string;
  counted: boolean | null;
  latest_outcome: string;
  evidence: string;
  harness: string;
  at: string;
  passes: number | null;
  failures: number | null;
  not_counted_reason: string | null;
};

export type HardwareRow = { id: string; cpus: number | null; memory_mib: number | null; disk_gib: number | null; storage: string; spec: string };

export type Budget = {
  id: string;
  state: string;
  hardware: string;
  limits: Record<string, number | null> | null;
  proposer: Person;
  approver: Person;
  approval_set: boolean;
} | null;

export type RunRow = {
  number: number | null;
  build: string;
  hardware: string;
  rows: number | null;
  duration_s: number | null;
  mismatches: number | null;
  interrupted: boolean | null;
  metrics: Record<string, number | null> | null;
  harness: string;
  at: string;
  verdict: string;
};

export type ProfileRow = {
  target: string;
  id: string;
  gate: string;
  axes: Record<string, number | null> | null;
  dataset: string;
  expected: string;
  retired: boolean | null;
  active_budget: Budget;
  budget_invalid: boolean;
  status: string | null;
  runs: List<RunRow>;
  retire: number | null;
  can_retire: boolean;
  retire_blocked: string;
};

export type ProfileStatus = { profile: string; status: string; run: number | null; failures: number | null };
export type GateStatus = { gate: string; status: string; profiles: ProfileStatus[] | null; retired: ProfileStatus[] | null };
export type Readiness = { ready: boolean | null; gates: GateStatus[] | null } | null;

export type Proposal = {
  module: string;
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
  summary: Record<string, string | number | boolean | null>;
  can_approve: boolean;
  approve_blocked: string;
  can_withdraw: boolean;
  withdraw_blocked: string;
};

export type EvidencePage = {
  revision: string;
  now: string;
  state: string;
  candidate: Candidate;
  matrix: { origin: string; paths: List<PathRow> };
  requirements: { origin: string; requirements: List<RequirementRow>; next: string; gate: Gate };
  artifacts: { origin: string; builds: List<BuildRow>; harnesses: List<HarnessRow>; results: List<ResultRow>; files: string; logs: string; environment: string; content: string };
  load: { origin: string; hardware: List<HardwareRow>; profiles: List<ProfileRow>; readiness: Readiness; measurement: string; planning_targets: string };
  decisions: { origin: string; proposals: List<Proposal>; support_decisions: string; activation_review: string; ledger: string };
};

type Raw = Record<string, unknown>;

const isRow = (x: unknown): x is Raw => !!x && typeof x === "object" && !Array.isArray(x);
const obj = (v: unknown): Raw => (isRow(v) ? v : {});
const str = (v: unknown): string => (typeof v === "string" ? v : "");
const flag = (v: unknown): boolean | null => (typeof v === "boolean" ? v : null);
const opt = (v: unknown): string | null => (typeof v === "string" ? v : null);
const strs = (v: unknown): string[] | null => (Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : null);

export function listOf<T>(v: unknown, f: (r: Raw) => T): List<T> {
  const r = obj(v);
  return { total: count(r.total), more: flag(r.more), rows: Array.isArray(r.rows) ? r.rows.filter(isRow).map(f) : null };
}

export function personOf(v: unknown): Person {
  if (v === null || v === undefined) return null;
  if (!isRow(v)) return { id: "", login: null, directory: null };
  return { id: str(v.id), login: opt(v.login), directory: flag(v.directory) };
}

export function countOf(v: unknown): CountOf {
  if (!isRow(v)) return null;
  return { count: count(v.count), of: count(v.of) };
}

export function numbers(v: unknown): Record<string, number | null> | null {
  if (!isRow(v)) return null;
  const out: Record<string, number | null> = {};
  for (const [key, value] of Object.entries(v)) {
    if (value === null) out[key] = null;
    else if (count(value) !== null) out[key] = count(value);
  }
  return out;
}

export function booleans(v: unknown): Record<string, boolean | null> | null {
  if (!isRow(v)) return null;
  return Object.fromEntries(Object.entries(v).map(([key, value]) => [key, flag(value)]));
}

const problemOf = (r: Raw): Problem => ({ problem: str(r.problem), subject: str(r.subject) });

export function problemsOf(v: unknown): Problem[] | null {
  return Array.isArray(v) ? v.filter(isRow).map(problemOf) : null;
}

export function pathOf(r: Raw): PathRow {
  const c = obj(r.columns);
  return {
    id: str(r.id),
    provider: str(r.provider),
    contract: str(r.contract),
    role: str(r.role),
    version: str(r.version),
    issuance: str(r.issuance),
    template: opt(r.template),
    state: str(r.state),
    build_set: r.build !== null && r.build !== undefined,
    requirements: listOf(r.requirements, (x) => ({ label: str(x.label), target: opt(x.target), state: opt(x.state) })),
    columns: { design: countOf(c.design), implementation: countOf(c.implementation), test: countOf(c.test), customer_proof: countOf(c.customer_proof), failing: count(c.failing) },
    categories: booleans(r.categories),
    reconciliations: listOf(r.reconciliations, (x) => ({ period: str(x.period), matched: flag(x.matched), differences: count(x.differences) })),
    last_observed: opt(r.last_observed),
    blockers: problemsOf(r.blockers),
    activation: str(r.activation),
    support_decision: str(r.support_decision),
    customer_proof_intake: str(r.customer_proof_intake),
  };
}

export function testOf(r: Raw): TestRow {
  return {
    id: str(r.id),
    category: opt(r.category),
    counted: flag(r.counted),
    latest_outcome: opt(r.latest_outcome),
    passes: count(r.passes),
    failures: count(r.failures),
    not_run_reason: opt(r.not_run_reason),
    not_counted_reason: opt(r.not_counted_reason),
  };
}

export function statesOf(v: unknown): Record<string, string> | null {
  if (!isRow(v)) return null;
  return Object.fromEntries(Object.entries(v).filter((e): e is [string, string] => typeof e[1] === "string"));
}

export function requirementOf(r: Raw): RequirementRow {
  const retire = hint(r.can_retire, r.retire_blocked);
  const n = count(r.retire);
  return {
    target: str(r.target),
    label: str(r.label),
    namespace: str(r.namespace),
    id: str(r.id),
    source: str(r.source),
    owner: str(r.owner),
    interfaces: strs(r.interfaces),
    common: flag(r.common),
    blocking: flag(r.blocking),
    design: str(r.design),
    commit: opt(r.commit),
    retired: flag(r.retired),
    state: opt(r.state),
    replacement: opt(r.replacement),
    states: statesOf(r.states),
    builds_more: flag(r.builds_more),
    tests: isRow(r.tests) ? listOf(r.tests, testOf) : null,
    retire: n !== null && n > 0 ? n : null,
    replacements: listOf(r.replacements, (x) => ({ target: str(x.target), label: str(x.label) })),
    replacement_required: flag(r.replacement_required),
    can_retire: retire.allowed,
    retire_blocked: retire.blocked,
  };
}

export function gateOf(v: unknown): Gate {
  if (!isRow(v)) return null;
  return { ready: flag(v.ready), problems: listOf(v.problems, problemOf), retired_blocking: strs(v.retired_blocking), templates: strs(v.templates), release: str(v.release) };
}

export function resultOf(r: Raw): ResultRow {
  return {
    namespace: str(r.namespace),
    test: str(r.test),
    build: str(r.build),
    counted: flag(r.counted),
    latest_outcome: str(r.latest_outcome),
    evidence: str(r.evidence),
    harness: str(r.harness),
    at: str(r.at),
    passes: count(r.passes),
    failures: count(r.failures),
    not_counted_reason: opt(r.not_counted_reason),
  };
}

export function budgetOf(v: unknown): Budget {
  if (!isRow(v)) return null;
  const set = (x: unknown) => x !== null && x !== undefined;
  return {
    id: str(v.id),
    state: str(v.state),
    hardware: str(v.hardware),
    limits: numbers(v.limits),
    proposer: personOf(v.proposer),
    approver: personOf(v.approver),
    approval_set: set(v.approval_by) || set(v.approved_by) || set(v.approved_at),
  };
}

export function runOf(r: Raw): RunRow {
  const n = count(r.number);
  return {
    number: n !== null && n > 0 ? n : null,
    build: str(r.build),
    hardware: str(r.hardware),
    rows: count(r.rows),
    duration_s: count(r.duration_s),
    mismatches: count(r.mismatches),
    interrupted: flag(r.interrupted),
    metrics: numbers(r.metrics),
    harness: str(r.harness),
    at: str(r.at),
    verdict: str(r.verdict),
  };
}

export function profileOf(r: Raw): ProfileRow {
  const retire = hint(r.can_retire, r.retire_blocked);
  const n = count(r.retire);
  return {
    target: str(r.target),
    id: str(r.id),
    gate: str(r.gate),
    axes: numbers(r.axes),
    dataset: str(r.dataset),
    expected: str(r.expected),
    retired: flag(r.retired),
    active_budget: budgetOf(r.active_budget),
    budget_invalid: r.active_budget !== null && !isRow(r.active_budget),
    status: opt(r.status),
    runs: listOf(r.runs, runOf),
    retire: n !== null && n > 0 ? n : null,
    can_retire: retire.allowed,
    retire_blocked: retire.blocked,
  };
}

const statusOf = (r: Raw): ProfileStatus => ({ profile: str(r.profile), status: str(r.status), run: count(r.run), failures: count(r.failures) });

export function readinessOf(v: unknown): Readiness {
  if (!isRow(v)) return null;
  const gates = Array.isArray(v.gates)
    ? v.gates.filter(isRow).map((g) => ({
        gate: str(g.gate),
        status: str(g.status),
        profiles: Array.isArray(g.profiles) ? g.profiles.filter(isRow).map(statusOf) : null,
        retired: Array.isArray(g.retired) ? g.retired.filter(isRow).map(statusOf) : null,
      }))
    : null;
  return { ready: flag(v.ready), gates };
}

export function summaryOf(v: unknown): Record<string, string | number | boolean | null> {
  if (!isRow(v)) return {};
  const out: Record<string, string | number | boolean | null> = {};
  for (const [key, value] of Object.entries(v)) {
    if (value === null || typeof value === "string" || typeof value === "boolean") out[key] = value;
    else if (typeof value === "number") out[key] = count(value) === null ? "" : value;
  }
  return out;
}

export function proposalOf(r: Raw): Proposal {
  const approve = hint(r.can_approve, r.approve_blocked);
  const withdraw = hint(r.can_withdraw, r.withdraw_blocked);
  const n = count(r.number);
  return {
    module: str(r.module),
    number: n !== null && n > 0 ? n : null,
    target: str(r.target),
    kind: str(r.kind),
    state: str(r.state),
    created_at: str(r.created_at),
    proposer: personOf(r.proposer),
    approver: personOf(r.approver),
    rationale: str(r.rationale),
    closed_by: personOf(r.closed_by),
    closing_rationale: str(r.closing_rationale),
    summary: summaryOf(r.summary),
    can_approve: approve.allowed,
    approve_blocked: approve.blocked,
    can_withdraw: withdraw.allowed,
    withdraw_blocked: withdraw.blocked,
  };
}

export function candidateOf(v: unknown): Candidate {
  if (v === null || v === undefined) return null;
  const r = obj(v);
  return { version: str(r.version), build: str(r.build), release: str(r.release) };
}

export function evidenceOf(body: Raw): EvidencePage {
  const m = obj(body.matrix);
  const q = obj(body.requirements);
  const a = obj(body.artifacts);
  const ld = obj(body.load);
  const d = obj(body.decisions);
  return {
    revision: str(body.revision),
    now: str(body.now),
    state: str(body.state),
    candidate: candidateOf(body.candidate),
    matrix: { origin: str(m.origin), paths: listOf(m.paths, pathOf) },
    requirements: { origin: str(q.origin), requirements: listOf(q.requirements, requirementOf), next: str(q.next), gate: gateOf(q.gate) },
    artifacts: {
      origin: str(a.origin),
      builds: listOf(a.builds, (x) => ({ build: str(x.build), commit: str(x.commit), order: count(x.order) })),
      harnesses: listOf(a.harnesses, (x) => ({ module: str(x.module), id: str(x.id), revocation: str(x.revocation) })),
      results: listOf(a.results, resultOf),
      files: str(a.files),
      logs: str(a.logs),
      environment: str(a.environment),
      content: str(a.content),
    },
    load: {
      origin: str(ld.origin),
      hardware: listOf(ld.hardware, (x) => ({ id: str(x.id), cpus: count(x.cpus), memory_mib: count(x.memory_mib), disk_gib: count(x.disk_gib), storage: str(x.storage), spec: str(x.spec) })),
      profiles: listOf(ld.profiles, profileOf),
      readiness: readinessOf(ld.readiness),
      measurement: str(ld.measurement),
      planning_targets: str(ld.planning_targets),
    },
    decisions: { origin: str(d.origin), proposals: listOf(d.proposals, proposalOf), support_decisions: str(d.support_decisions), activation_review: str(d.activation_review), ledger: str(d.ledger) },
  };
}

export type RequirementsQuery = { after?: string; q?: string; state?: string; retired?: string; targets?: string[] };
export type RequirementsAnswer = { rows: RequirementRow[]; next: string; matching: number; total: number };

export function requirementsPath(query: RequirementsQuery): string {
  const q = new URLSearchParams();
  if (query.targets) q.set("target", query.targets.join(","));
  else {
    if (query.after) q.set("after", query.after);
    if (query.q) q.set("q", query.q);
    if (query.state) q.set("state", query.state);
    if (query.retired) q.set("retired", query.retired);
  }
  const text = q.toString();
  return "/release-evidence/requirements" + (text ? "?" + text : "");
}

export function requirementsAnswerOf(body: Raw): RequirementsAnswer | null {
  const whole = (v: unknown): number | null => (typeof v === "number" && Number.isInteger(v) && v >= 0 ? v : null);
  const matching = whole(body.matching);
  const total = whole(body.total);
  if (!Array.isArray(body.requirements) || typeof body.next !== "string" || matching === null || total === null) return null;
  return { rows: body.requirements.filter(isRow).map(requirementOf), next: body.next, matching, total };
}
