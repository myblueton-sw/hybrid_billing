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
export type PolicyPage = {
  revision: string;
  policy: PolicyRow | null;
  proposals: ProposalRow[];
  now: string;
  signed_in_at: string;
  act_until: string;
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
  | "stale";

export class AdminError extends Error {
  constructor(
    readonly failure: AdminFailure,
    readonly fields: string[] = [],
    readonly retryAfter = 0,
    readonly operation = "",
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
  | "policy.withdraw";

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
};

export function failureOf(status: number, body: Record<string, unknown>): AdminError {
  const code = String(body.code ?? "");
  const fields = Array.isArray(body.fields) ? body.fields.map(String) : [];
  const operation = typeof body.operation_id === "string" ? body.operation_id : "";
  const failure = codes[code] ?? (status === 401 ? "unauthenticated" : status === 429 ? "limited" : "unavailable");
  if (failure === "partial" && !operation) return new AdminError("unavailable");
  return new AdminError(failure, fields, 0, operation);
}

function random(): string {
  const bytes = new Uint8Array(12);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

export function attempt(): Attempt {
  return { request: "m01-" + random(), key: "m01-" + random() };
}

export function adminClient(csrf: string, f: Fetch = fetch.bind(globalThis)) {
  async function send(path: string, init: RequestInit, request: string): Promise<Record<string, unknown>> {
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
        act_until: String(body.act_until ?? ""),
        you: String(body.you ?? ""),
      };
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
