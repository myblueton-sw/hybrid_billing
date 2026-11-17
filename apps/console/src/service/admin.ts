import { contractDetailOf, contractsPageOf, type ContractDetail, type ContractsPage } from "./contracts";
import { installationOf, type InstallationPage } from "./installation";
import { integrationsOf, type IntegrationsPage } from "./integrations";
import { recoveryOf, type RecoveryPage } from "./recovery";
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
export type PolicyRow = RevisionRow & { number: number; proposer: string; approver: string; at: string };
export type ProposalRow = {
  number: number;
  target: string;
  state: "pending" | "applied";
  proposer: string;
  approver: string;
  reason: string;
  revision: RevisionRow;
  can_approve: boolean;
  can_withdraw: boolean;
  act_until: string;
};
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
  state: "active" | "revoked" | "expired" | "pending";
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
  state: "active" | "revoked" | "expired" | "pending";
  version: number;
};
export type Ceiling = { action: string; kind: string; fields: string[]; until: string };
export type AccessPage = {
  revision: string;
  grants: AccessGrant[];
  delegations: AccessDelegation[];
  catalog: Ceiling[];
  subjects: { user_id: string; login: string }[];
  now: string;
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
  can_activate: boolean;
  blocked: string;
};
export type PlansPage = {
  revision: string;
  active_revision: number;
  open_plans: number;
  open_limit: number;
  impact: { money: string; publication: string; erp: string };
  plans: PlanRow[];
  you: string;
};

export function planOf(v: unknown): PlanRow {
  const r = (v ?? {}) as Record<string, unknown>;
  const list = <T,>(x: unknown) => (Array.isArray(x) ? (x as T[]) : []);
  const c = (r.change ?? {}) as Record<string, unknown>;
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
    can_activate: r.can_activate === true,
    blocked: String(r.blocked ?? ""),
  };
}

export type OrgQuery = { on?: string; kind?: RelationKind; revision?: number; after?: string; limit?: number };

export type PolicyPage = {
  revision: string;
  policy: PolicyRow | null;
  proposals: ProposalRow[];
  now: string;
  signed_in_at: string;
  authenticated_at: string;
  act_until: string;
  step_up_seconds: number;
  you: string;
};

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
  | "reauth_cancelled";

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
  | "policy.propose"
  | "policy.approve"
  | "policy.withdraw"
  | "access.grant"
  | "access.revoke_grant"
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
  | "retention.stop_run";

export type Outcome = { state: string; fields: Record<string, string> };

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
      const fields = (body.fields ?? {}) as Record<string, unknown>;
      return { state: String(body.business_state ?? ""), fields: Object.fromEntries(Object.entries(fields).map(([k, v]) => [k, String(v)])) };
    },
    async policy(): Promise<PolicyPage> {
      const body = await send("/identity/policy", { method: "GET" }, attempt().request);
      return {
        revision: String(body.revision ?? ""),
        policy: (body.policy ?? null) as PolicyRow | null,
        proposals: Array.isArray(body.proposals) ? (body.proposals as ProposalRow[]) : [],
        now: String(body.now ?? ""),
        signed_in_at: String(body.signed_in_at ?? ""),
        authenticated_at: String(body.authenticated_at ?? ""),
        act_until: String(body.act_until ?? ""),
        step_up_seconds: Number(body.step_up_seconds ?? 0),
        you: String(body.you ?? ""),
      };
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
      const impact = (body.impact ?? {}) as Record<string, unknown>;
      const state = (v: unknown) => (typeof v === "string" && v ? v : "unknown");
      return {
        revision: String(body.revision ?? ""),
        active_revision: Number(body.active_revision ?? 0),
        open_plans: Number(body.open_plans ?? 0),
        open_limit: Number(body.open_limit ?? 0),
        impact: { money: state(impact.money), publication: state(impact.publication), erp: state(impact.erp) },
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
