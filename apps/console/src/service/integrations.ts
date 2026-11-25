import { keptFactsOf, type KeptFacts } from "./kept";
export const rightIds = [
  "administer_api_clients",
  "administer_modules",
  "approve_modules",
  "stop_modules",
  "administer_channels",
  "approve_channels",
  "stop_channels",
  "reprocess_events",
] as const;
export type RightId = (typeof rightIds)[number];
export type Rights = Record<RightId, boolean>;
export const pendingKinds = ["plugin_proposal", "subscription_proposal", "resume_request", "redeliver_request", "dismiss_request"] as const;
export type PendingKind = (typeof pendingKinds)[number];
export type OwnPending = { kind: PendingKind; id: string; target: string; object: string; state: string };

export type Mark = { by: string; by_login: string; reason: string; confirmed_by: string; confirmed_by_login: string };
export type MarkValue = Mark | null | "unknown";
export type Avail = { allowed: boolean; blocked: string };

export type GrantScope = { customer: string; legal_entity: string; contract: string; period: string };
export type ClientGrant = { id: string; action: string; kind: string; scope: GrantScope; not_before: string; not_after: string; state: string };
export type ApiClient = { id: string; kind: string; state: string; grants: ClientGrant[] | null };
export type ServiceUser = { id: string; state: string; owner: string; owner_login: string; usable_credentials: number | null };
export type ClientsSection = { state: string; clients: ApiClient[] | null; service_users: ServiceUser[] | null; manage_on: string; oauth_clients: string };

export const licenseStates = ["permitted", "not_permitted"] as const;
export type LicenseState = (typeof licenseStates)[number] | "unknown";
export const licenseSources = ["default", "start_flag"] as const;
export type LicenseSource = (typeof licenseSources)[number] | "unknown";
export type LicenseList = { permitted: string[] | null; source: LicenseSource };
export type PluginWorkspace = { id: string; runtime: string; verification: boolean | null };
export type PublisherKey = { version: string; not_before: string; not_after: string; retired: boolean | null; compromise: MarkValue };
export type Publisher = { id: string; name: string; revocation: MarkValue; keys: PublisherKey[] | null };
export type Storage = { namespace: string; owner: string; retention_days: number | null; deletion: string; migration: string };
export type ScreenField = { name: string; label: string; source: string };
export type PluginScreen = { id: string; component: string; label: string; fields: ScreenField[] | null };
export type PluginVersion = {
  plugin: string;
  version: string;
  publisher: string;
  key_version: string;
  digest: string;
  license: string;
  license_state: LicenseState;
  core_min: string;
  core_max: string;
  capabilities: string[] | null;
  egress: string[] | null;
  endpoints: string[] | null;
  timeout_seconds: number | null;
  memory_mib: number | null;
  concurrency: number | null;
  storage: Storage;
  screens: PluginScreen[] | null;
  vulnerable: MarkValue;
};
export type ApprovedEndpoint = { id: string; fingerprint: string; approver: string; approver_login: string };
export type PluginInstall = {
  plugin: string;
  version: string;
  state: string;
  reason: string;
  revision: number | null;
  endpoints: ApprovedEndpoint[] | null;
  disposal: Storage | null;
  can_stop: boolean;
  stop_blocked: string;
  can_disable: boolean;
  disable_blocked: string;
  tokens: string;
  runtime: string;
};
export type PluginSummary = {
  publisher: string;
  publisher_name: string;
  key_version: string;
  key_not_before: string;
  key_not_after: string;
  plugin: string;
  version: string;
  workspace: string;
  runtime: string;
  endpoint: string;
  fingerprint: string;
};
export type PluginProposal = {
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
  closing_reason: string;
  closed_request: string;
  closure_trigger: string;
  closure_trigger_login: string;
  summary: PluginSummary;
  can_approve: boolean;
  approve_blocked: string;
  can_withdraw: boolean;
  withdraw_blocked: string;
};
export type ModulesSection = {
  state: string;
  licenses: LicenseList;
  workspace: PluginWorkspace | null;
  publishers: Publisher[] | null;
  plugins: PluginVersion[] | null;
  installations: PluginInstall[] | null;
  proposals: PluginProposal[] | null;
  proposal_history: KeptFacts | null;
  actions: { propose: Avail };
};

export type Destination = { id: string; state: string; source: string };
export type SubscriptionKey = { version: string; not_before: string; not_after: string; state: string; proposer: string; proposer_login: string; approver: string; approver_login: string };
export type Gap = { after: number | null; to: number | null };
export type HistoryEntry = { action: string; actor: string; actor_login: string; rationale: string; reason: string; request: string };
export const subscriptionActions = ["approve", "pause", "request_resume", "approve_resume", "revoke", "withdraw", "withdraw_resume"] as const;
export type SubscriptionAction = (typeof subscriptionActions)[number];
export type Subscription = {
  id: string;
  target: string;
  subscriber: string;
  destination: string;
  types: string[] | null;
  state: string;
  reason: string;
  cursor: number | null;
  lag: number | null;
  gaps: Gap[] | null;
  created_at: string;
  rationale: string;
  proposer: string;
  proposer_login: string;
  approver: string;
  approver_login: string;
  resumer: string;
  resumer_login: string;
  history: HistoryEntry[] | null;
  resume_request: RequestValue;
  keys: SubscriptionKey[] | null;
  signer: string;
  actions: Record<SubscriptionAction, Avail>;
};
export type ChannelsSection = {
  state: string;
  destinations: Destination[] | null;
  subscribers: string[] | null;
  subscriptions: Subscription[] | null;
  actions: { propose: Avail };
};

export type EventType = { name: string; schema_version: string; fields: string[] | null; state: string; proposer: string; proposer_login: string; approver: string; approver_login: string };
export type LetterRequest = { id: string; target: string; by: string; by_login: string; rationale: string };
export type RequestValue = LetterRequest | null | "unknown";
export const letterActions = ["request_redeliver", "approve_redeliver", "request_dismiss", "approve_dismiss", "withdraw_redeliver", "withdraw_dismiss", "reapprove"] as const;
export type LetterAction = (typeof letterActions)[number];
export type Delivery = {
  id: string;
  target: string;
  subscription: string;
  sequence: number | null;
  source: string;
  event_id: string;
  kind: string;
  state: string;
  reason: string;
  attempts: number | null;
  outcome_unknown: boolean | null;
  hold: string;
  requester: string;
  requester_login: string;
  approver: string;
  approver_login: string;
  redeliver_request: RequestValue;
  dismiss_request: RequestValue;
  actions: Record<LetterAction, Avail>;
};
export type EventsSection = {
  state: string;
  types: EventType[] | null;
  outbox: { floor: number | null; head: number | null };
  deliveries: Delivery[] | null;
  deliveries_more: string[] | null;
  dispatcher: string;
};

export type IntegrationsPage = {
  revision: string;
  now: string;
  state: string;
  rights: Rights | null;
  own_pending: OwnPending[] | null;
  clients: ClientsSection;
  modules: ModulesSection;
  channels: ChannelsSection;
  events: EventsSection;
};

type Raw = Record<string, unknown>;

const isRow = (x: unknown): x is Raw => !!x && typeof x === "object" && !Array.isArray(x);
const obj = (v: unknown): Raw => (isRow(v) ? v : {});
const str = (v: unknown): string => (typeof v === "string" ? v : "");
const flag = (v: unknown): boolean | null => (typeof v === "boolean" ? v : null);
const strs = (v: unknown): string[] | null => (Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : null);
const rows = <T,>(v: unknown, f: (r: Raw) => T): T[] | null => (Array.isArray(v) ? v.filter(isRow).map(f) : null);

export function count(v: unknown): number | null {
  return typeof v === "number" && Number.isSafeInteger(v) && v >= 0 ? v : null;
}

export function availOf(v: unknown): Avail {
  if (!isRow(v)) return { allowed: false, blocked: "missing" };
  const blocked = typeof v.blocked === "string" ? v.blocked : "missing";
  if (v.allowed === true && blocked === "") return { allowed: true, blocked: "" };
  return { allowed: false, blocked: blocked || "missing" };
}

export function hint(allowed: unknown, blocked: unknown): Avail {
  const why = typeof blocked === "string" ? blocked : "missing";
  if (allowed === true && why === "") return { allowed: true, blocked: "" };
  return { allowed: false, blocked: why || "missing" };
}

function actionsOf<K extends string>(v: unknown, keys: readonly K[]): Record<K, Avail> {
  const r = isRow(v) ? v : null;
  return Object.fromEntries(keys.map((key) => [key, r ? availOf(r[key]) : { allowed: false, blocked: "missing" }])) as Record<K, Avail>;
}

export function markOf(v: unknown): MarkValue {
  if (v === null) return null;
  if (!isRow(v)) return "unknown";
  return { by: str(v.by), by_login: str(v.by_login), reason: str(v.reason), confirmed_by: str(v.confirmed_by), confirmed_by_login: str(v.confirmed_by_login) };
}

export function requestOf(v: unknown): RequestValue {
  if (v === null) return null;
  if (!isRow(v)) return "unknown";
  return { id: str(v.id), target: str(v.target), by: str(v.by), by_login: str(v.by_login), rationale: str(v.rationale) };
}

export function storageOf(v: unknown): Storage {
  const r = obj(v);
  return { namespace: str(r.namespace), owner: str(r.owner), retention_days: count(r.retention_days), deletion: str(r.deletion), migration: str(r.migration) };
}

function grantOf(r: Raw): ClientGrant {
  const s = obj(r.scope);
  return {
    id: str(r.id),
    action: str(r.action),
    kind: str(r.kind),
    scope: { customer: str(s.customer), legal_entity: str(s.legal_entity), contract: str(s.contract), period: str(s.period) },
    not_before: str(r.not_before),
    not_after: str(r.not_after),
    state: str(r.state),
  };
}

export function clientsOf(v: unknown): ClientsSection {
  const r = obj(v);
  return {
    state: str(r.state),
    clients: rows(r.clients, (c) => ({ id: str(c.id), kind: str(c.kind), state: str(c.state), grants: rows(c.grants, grantOf) })),
    service_users: rows(r.service_users, (u) => ({ id: str(u.id), state: str(u.state), owner: str(u.owner), owner_login: str(u.owner_login), usable_credentials: count(u.usable_credentials) })),
    manage_on: str(r.manage_on),
    oauth_clients: str(r.oauth_clients),
  };
}

export function licenseStateOf(v: unknown): LicenseState {
  return (licenseStates as readonly unknown[]).includes(v) ? (v as LicenseState) : "unknown";
}

export function licensesOf(v: unknown): LicenseList {
  const r = obj(v);
  return { permitted: strs(r.permitted), source: (licenseSources as readonly unknown[]).includes(r.source) ? (r.source as LicenseSource) : "unknown" };
}

function versionOf(r: Raw): PluginVersion {
  return {
    plugin: str(r.plugin),
    version: str(r.version),
    publisher: str(r.publisher),
    key_version: str(r.key_version),
    digest: str(r.digest),
    license: str(r.license),
    license_state: licenseStateOf(r.license_state),
    core_min: str(r.core_min),
    core_max: str(r.core_max),
    capabilities: strs(r.capabilities),
    egress: strs(r.egress),
    endpoints: strs(r.endpoints),
    timeout_seconds: count(r.timeout_seconds),
    memory_mib: count(r.memory_mib),
    concurrency: count(r.concurrency),
    storage: storageOf(r.storage),
    screens: rows(r.screens, (s) => ({
      id: str(s.id),
      component: str(s.component),
      label: str(s.label),
      fields: rows(s.fields, (f) => ({ name: str(f.name), label: str(f.label), source: str(f.source) })),
    })),
    vulnerable: markOf(r.vulnerable),
  };
}

export function installOf(r: Raw): PluginInstall {
  const stop = hint(r.can_stop, r.stop_blocked);
  const disable = hint(r.can_disable, r.disable_blocked);
  return {
    plugin: str(r.plugin),
    version: str(r.version),
    state: str(r.state),
    reason: str(r.reason),
    revision: count(r.revision),
    endpoints: rows(r.endpoints, (e) => ({ id: str(e.id), fingerprint: str(e.fingerprint), approver: str(e.approver), approver_login: str(e.approver_login) })),
    disposal: isRow(r.disposal) ? storageOf(r.disposal) : null,
    can_stop: stop.allowed,
    stop_blocked: stop.blocked,
    can_disable: disable.allowed,
    disable_blocked: disable.blocked,
    tokens: str(r.tokens),
    runtime: str(r.runtime),
  };
}

function summaryOf(v: unknown): PluginSummary {
  const r = obj(v);
  return {
    publisher: str(r.publisher),
    publisher_name: str(r.publisher_name),
    key_version: str(r.key_version),
    key_not_before: str(r.key_not_before),
    key_not_after: str(r.key_not_after),
    plugin: str(r.plugin),
    version: str(r.version),
    workspace: str(r.workspace),
    runtime: str(r.runtime),
    endpoint: str(r.endpoint),
    fingerprint: str(r.fingerprint),
  };
}

export function pluginProposalOf(r: Raw): PluginProposal {
  const approve = hint(r.can_approve, r.approve_blocked);
  const withdraw = hint(r.can_withdraw, r.withdraw_blocked);
  const n = count(r.number);
  return {
    number: n !== null && n > 0 ? n : null,
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
    closing_reason: str(r.closing_reason),
    closed_request: str(r.closed_request),
    closure_trigger: str(r.closure_trigger),
    closure_trigger_login: str(r.closure_trigger_login),
    summary: summaryOf(r.summary),
    can_approve: approve.allowed,
    approve_blocked: approve.blocked,
    can_withdraw: withdraw.allowed,
    withdraw_blocked: withdraw.blocked,
  };
}

export function modulesOf(v: unknown): ModulesSection {
  const r = obj(v);
  const w = isRow(r.workspace) ? r.workspace : null;
  return {
    state: str(r.state),
    licenses: licensesOf(r.licenses),
    workspace: w ? { id: str(w.id), runtime: str(w.runtime), verification: flag(w.verification) } : null,
    publishers: rows(r.publishers, (p) => ({
      id: str(p.id),
      name: str(p.name),
      revocation: markOf(p.revocation),
      keys: rows(p.keys, (k) => ({ version: str(k.version), not_before: str(k.not_before), not_after: str(k.not_after), retired: flag(k.retired), compromise: markOf(k.compromise) })),
    })),
    plugins: rows(r.plugins, versionOf),
    installations: rows(r.installations, installOf),
    proposals: rows(r.proposals, pluginProposalOf),
    proposal_history: isRow(r.proposal_history) ? keptFactsOf(r.proposal_history) : null,
    actions: actionsOf(r.actions, ["propose"] as const),
  };
}

export function subscriptionOf(r: Raw): Subscription {
  return {
    id: str(r.id),
    target: str(r.target),
    subscriber: str(r.subscriber),
    destination: str(r.destination),
    types: strs(r.types),
    state: str(r.state),
    reason: str(r.reason),
    cursor: count(r.cursor),
    lag: count(r.lag),
    gaps: rows(r.gaps, (g) => ({ after: count(g.after), to: count(g.to) })),
    created_at: str(r.created_at),
    rationale: str(r.rationale),
    proposer: str(r.proposer),
    proposer_login: str(r.proposer_login),
    approver: str(r.approver),
    approver_login: str(r.approver_login),
    resumer: str(r.resumer),
    resumer_login: str(r.resumer_login),
    history: rows(r.history, (h) => ({
      action: str(h.action),
      actor: str(h.actor),
      actor_login: str(h.actor_login),
      rationale: str(h.rationale),
      reason: str(h.reason),
      request: str(h.request),
    })),
    resume_request: requestOf(r.resume_request),
    keys: rows(r.keys, (k) => ({
      version: str(k.version),
      not_before: str(k.not_before),
      not_after: str(k.not_after),
      state: str(k.state),
      proposer: str(k.proposer),
      proposer_login: str(k.proposer_login),
      approver: str(k.approver),
      approver_login: str(k.approver_login),
    })),
    signer: str(r.signer),
    actions: actionsOf(r.actions, subscriptionActions),
  };
}

export function channelsOf(v: unknown): ChannelsSection {
  const r = obj(v);
  return {
    state: str(r.state),
    destinations: rows(r.destinations, (d) => ({ id: str(d.id), state: str(d.state), source: str(d.source) })),
    subscribers: strs(r.subscribers),
    subscriptions: rows(r.subscriptions, subscriptionOf),
    actions: actionsOf(r.actions, ["propose"] as const),
  };
}

export function deliveryOf(r: Raw): Delivery {
  return {
    id: str(r.id),
    target: str(r.target),
    subscription: str(r.subscription),
    sequence: count(r.sequence),
    source: str(r.source),
    event_id: str(r.event_id),
    kind: str(r.kind),
    state: str(r.state),
    reason: str(r.reason),
    attempts: count(r.attempts),
    outcome_unknown: flag(r.outcome_unknown),
    hold: str(r.hold),
    requester: str(r.requester),
    requester_login: str(r.requester_login),
    approver: str(r.approver),
    approver_login: str(r.approver_login),
    redeliver_request: requestOf(r.redeliver_request),
    dismiss_request: requestOf(r.dismiss_request),
    actions: actionsOf(r.actions, letterActions),
  };
}

export function eventsOf(v: unknown): EventsSection {
  const r = obj(v);
  const o = obj(r.outbox);
  return {
    state: str(r.state),
    types: rows(r.types, (t) => ({
      name: str(t.name),
      schema_version: str(t.schema_version),
      fields: strs(t.fields),
      state: str(t.state),
      proposer: str(t.proposer),
      proposer_login: str(t.proposer_login),
      approver: str(t.approver),
      approver_login: str(t.approver_login),
    })),
    outbox: { floor: count(o.floor), head: count(o.head) },
    deliveries: rows(r.deliveries, deliveryOf),
    deliveries_more: strs(r.deliveries_more),
    dispatcher: str(r.dispatcher),
  };
}

export function rightsOf(v: unknown): Rights | null {
  if (!isRow(v)) return null;
  return Object.fromEntries(rightIds.map((id) => [id, v[id] === true])) as Rights;
}

export function ownPendingOf(v: unknown): OwnPending[] | null {
  const list = rows(v, (r) => ({ kind: str(r.kind) as PendingKind, id: str(r.id), target: str(r.target), object: str(r.object), state: str(r.state) }));
  return list === null ? null : list.filter((p) => (pendingKinds as readonly string[]).includes(p.kind));
}

export function integrationsOf(body: Raw): IntegrationsPage {
  return {
    revision: str(body.revision),
    now: str(body.now),
    state: str(body.state),
    rights: rightsOf(body.rights),
    own_pending: ownPendingOf(body.own_pending),
    clients: clientsOf(body.clients),
    modules: modulesOf(body.modules),
    channels: channelsOf(body.channels),
    events: eventsOf(body.events),
  };
}
