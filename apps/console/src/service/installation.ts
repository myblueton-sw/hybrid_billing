import { keptFactsOf, type KeptFacts } from "./kept";
export type InstallItem = { kind: string; target: string; action: string; conflict: string };
export type InstallEffect = { volume: string; effect: string };
export type InstallPreview = {
  state: string;
  reason: string;
  kind: string;
  version: string;
  hash: string;
  services: InstallItem[] | null;
  ports: InstallItem[] | null;
  volumes: InstallItem[] | null;
  data_effects: InstallEffect[] | null;
  permissions: string[] | null;
  leftovers: InstallItem[] | null;
  conflicts: InstallItem[] | null;
};
export type InstallSite = { id: string; platform: string; probe: string };
export type InstallObservation = { state: string; observed_at: string; expires_at: string };
export type InstallOwnedService = { name: string; ports: string[] };
export type InstallOwnedVolume = { host: string; logical: string; data: boolean | null; detached: boolean };
export type InstallInstalled = {
  state: string;
  version: string;
  services: InstallOwnedService[] | null;
  volumes: InstallOwnedVolume[] | null;
  history: number | null;
  backups: number | null;
};
export type InstallReleaseService = { name: string; artifact: string; ports: string[]; volumes: string[] };
export type InstallMark = { by: string; by_login: string; reason: string; confirmed_by: string; confirmed_by_login: string };
export type InstallRelease = {
  version: string;
  key_version: string;
  upgrade_from: string[] | null;
  platforms: string[] | null;
  services: InstallReleaseService[] | null;
  volumes: { name: string; data: boolean | null }[] | null;
  artifacts: { name: string; kind: string }[] | null;
  mark: InstallMark | null | "unknown";
};
export type InstallSummary = {
  key_version: string;
  release_version: string;
  site: string;
  destination: string;
  host: string;
  port: number | null;
  ranges: string[];
  plan_version: string;
  plan_hash: string;
  plan_admin: string;
};
export type InstallProposal = {
  number: number | null;
  target: string;
  kind: string;
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
  summary: InstallSummary;
  can_approve: boolean;
  approve_blocked: string;
  can_withdraw: boolean;
  withdraw_blocked: string;
  current_hash: string;
};
export type InstallDestination = { id: string; host: string; port: number | null; ranges: string[] };
export type InstallReport = { destination: string; kind: string; failure: string; address: string; via: string; cause: string; steps: string[] };
export type InstallationPage = {
  revision: string;
  now: string;
  state: string;
  site: InstallSite;
  observation: InstallObservation;
  installed: InstallInstalled;
  releases: InstallRelease[];
  preview: InstallPreview;
  preview_basis: "pending_plan" | "newest_release" | "";
  upgrade: InstallPreview;
  rollback: InstallPreview;
  proposals: InstallProposal[];
  proposal_history: KeptFacts;
  destinations: InstallDestination[];
  reports: InstallReport[];
  runs: string;
};

type Raw = Record<string, unknown>;

const obj = (v: unknown): Raw => (v && typeof v === "object" && !Array.isArray(v) ? (v as Raw) : {});
const str = (v: unknown): string => (typeof v === "string" ? v : "");
const num = (v: unknown): number | null => (typeof v === "number" && Number.isFinite(v) ? v : null);
const strs = (v: unknown): string[] => (Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : []);
const maybeStrs = (v: unknown): string[] | null => (Array.isArray(v) ? strs(v) : null);
const flag = (v: unknown): boolean | null => (typeof v === "boolean" ? v : null);
const isRow = (x: unknown): x is Raw => !!x && typeof x === "object" && !Array.isArray(x);
const rows = <T,>(v: unknown, f: (r: Raw) => T): T[] => (Array.isArray(v) ? v.filter(isRow).map(f) : []);
const maybeRows = <T,>(v: unknown, f: (r: Raw) => T): T[] | null => (Array.isArray(v) ? rows(v, f) : null);
const historyCount = (v: unknown): number | null => Array.isArray(v) && v.every((r) => isRow(r) && ["kind", "from", "to", "run", "backup"].every((k) => typeof r[k] === "string") && Array.isArray(r.runs) && r.runs.every((x: unknown) => typeof x === "string")) ? v.length : null;
const backupCount = (v: unknown): number | null => Array.isArray(v) && v.every((x) => typeof x === "string") ? v.length : null;

const itemOf = (r: Raw): InstallItem => ({ kind: str(r.kind), target: str(r.target), action: str(r.action), conflict: str(r.conflict) });

export function previewOf(v: unknown): InstallPreview {
  const r = obj(v);
  return {
    state: str(r.state),
    reason: str(r.reason),
    kind: str(r.kind),
    version: str(r.version),
    hash: str(r.hash),
    services: maybeRows(r.services, itemOf),
    ports: maybeRows(r.ports, itemOf),
    volumes: maybeRows(r.volumes, itemOf),
    data_effects: maybeRows(r.data_effects, (d) => ({ volume: str(d.volume), effect: str(d.effect) })),
    permissions: maybeStrs(r.permissions),
    leftovers: maybeRows(r.leftovers, itemOf),
    conflicts: maybeRows(r.conflicts, itemOf),
  };
}

function installedOf(v: unknown): InstallInstalled {
  const r = obj(v);
  return {
    state: str(r.state),
    version: str(r.version),
    services: maybeRows(r.services, (s) => ({ name: str(s.name), ports: strs(s.ports) })),
    volumes: maybeRows(r.volumes, (s) => ({ host: str(s.host), logical: str(s.logical), data: flag(s.data), detached: s.detached === true })),
    history: historyCount(r.history),
    backups: backupCount(r.backups),
  };
}

function markOf(v: unknown): InstallMark | null | "unknown" {
  if (v === null) return null;
  if (!isRow(v)) return "unknown";
  return { by: str(v.by), by_login: str(v.by_login), reason: str(v.reason), confirmed_by: str(v.confirmed_by), confirmed_by_login: str(v.confirmed_by_login) };
}

function releaseOf(r: Raw): InstallRelease {
  return {
    version: str(r.version),
    key_version: str(r.key_version),
    upgrade_from: maybeStrs(r.upgrade_from),
    platforms: maybeStrs(r.platforms),
    services: maybeRows(r.services, (s) => ({ name: str(s.name), artifact: str(s.artifact), ports: strs(s.ports), volumes: strs(s.volumes) })),
    volumes: maybeRows(r.volumes, (s) => ({ name: str(s.name), data: flag(s.data) })),
    artifacts: maybeRows(r.artifacts, (s) => ({ name: str(s.name), kind: str(s.kind) })),
    mark: markOf(r.mark),
  };
}

function summaryOf(v: unknown): InstallSummary {
  const r = obj(v);
  return {
    key_version: str(r.key_version),
    release_version: str(r.release_version),
    site: str(r.site),
    destination: str(r.destination),
    host: str(r.host),
    port: num(r.port),
    ranges: strs(r.ranges),
    plan_version: str(r.plan_version),
    plan_hash: str(r.plan_hash),
    plan_admin: str(r.plan_admin),
  };
}

function hint(allowed: unknown, blocked: unknown): { can: boolean; blocked: string } {
  const why = typeof blocked === "string" ? blocked : "missing";
  if (allowed === true && why === "") return { can: true, blocked: "" };
  return { can: false, blocked: why || "missing" };
}

export function proposalOf(r: Raw): InstallProposal {
  const approve = hint(r.can_approve, r.approve_blocked);
  const withdraw = hint(r.can_withdraw, r.withdraw_blocked);
  const n = num(r.number);
  return {
    number: n !== null && Number.isInteger(n) && n > 0 ? n : null,
    target: str(r.target),
    kind: str(r.kind),
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
    summary: summaryOf(r.summary),
    can_approve: approve.can,
    approve_blocked: approve.blocked,
    can_withdraw: withdraw.can,
    withdraw_blocked: withdraw.blocked,
    current_hash: str(r.current_hash),
  };
}

export function installationOf(body: Raw): InstallationPage {
  const site = obj(body.site);
  const o = obj(body.observation);
  return {
    revision: str(body.revision),
    now: str(body.now),
    state: str(body.state),
    site: { id: str(site.id), platform: str(site.platform), probe: str(site.probe) },
    observation: { state: str(o.state), observed_at: str(o.observed_at), expires_at: str(o.expires_at) },
    installed: installedOf(body.installed),
    releases: rows(body.releases, releaseOf),
    preview: previewOf(body.preview),
    preview_basis: body.preview_basis === "pending_plan" || body.preview_basis === "newest_release" ? body.preview_basis : "",
    upgrade: previewOf(body.upgrade),
    rollback: previewOf(body.rollback),
    proposals: rows(body.proposals, proposalOf),
    proposal_history: keptFactsOf(body.proposal_history),
    destinations: rows(body.destinations, (d) => ({ id: str(d.id), host: str(d.host), port: num(d.port), ranges: strs(d.ranges) })),
    reports: rows(body.reports, (d) => ({
      destination: str(d.destination),
      kind: str(d.kind),
      failure: str(d.failure),
      address: str(d.address),
      via: str(d.via),
      cause: str(d.cause),
      steps: strs(d.steps),
    })),
    runs: str(body.runs),
  };
}
