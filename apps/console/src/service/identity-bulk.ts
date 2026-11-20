import { AdminError, attempt, clockOf, filterOf, settingsOf, usersQueryString, type AdminClient, type Attempt, type ClockState, type SettingRow, type UsersFilter, type UsersQuery } from "./admin";

export type BulkCounts = { targets: number; excluded: number; duplicates: number; succeeded: number; held: number; failed: number; unprocessed: number };
export type BulkJob = {
  id: string;
  state: "draft" | "prepared" | "executed" | "discarded" | "expired";
  origin: string;
  effect: "suspend_membership" | "suspend_user" | "";
  preparer: string;
  prepared_at: string;
  expires_at: string;
  executor: string;
  executed_at: string;
  deny_at: string;
  digest: string;
  counts: BulkCounts;
  job_state: "" | "running" | "waiting" | "succeeded";
  job_reason: string;
  ledger: { first_sequence: number; last_sequence: number; watermark: string; reason: string };
  cleanup: { done: number; total: number };
  recorded_at: string;
  staged: number;
  selection: "explicit" | "filter" | "not_recorded";
  matched: number;
  fixed_at: string;
  fixed_revision: string;
  filter: UsersFilter | null;
  search_masked: boolean;
};
export type MatchCount = { count: number; complete: boolean; over_limit: boolean; limit: number; scan_budget: number; list_version: string; server_time: string };
export type BulkPage = {
  revision: string;
  started_at: string;
  can_prepare: boolean;
  open_draft: boolean;
  pending: number;
  next: string;
  jobs: BulkJob[];
  settings: SettingRow[];
  ledger_state: string;
  clock: ClockState;
};
export type PendingStore = { store: string; count: number; rechecked: boolean };
export type Impact = {
  computed_at: string;
  people: number;
  administrators: number;
  members: number;
  installation_administrators: number;
  trust_right_holders: number;
  bulk_right_holders: number;
  sessions: number;
  pickups: number;
  api_keys: string;
  pending_jobs: string;
  pending: PendingStore[];
  service_users: number;
  anchors_after: number;
  other_workspaces: string;
};
export type BulkDetail = { revision: string; server_time: string; job: BulkJob; impact: Impact; exclusions: Record<string, number>; outside_scope: string[] };
export type Cleanup = { item: string; state: string; reason: string };
export type BulkItem = {
  user_id: string;
  login: string;
  role: string;
  role_until: string;
  outcome: "excluded" | "held" | "succeeded" | "failed" | "unprocessed";
  reason: string;
  state: string;
  receipt: { sequence: number; watermark: string; reason: string };
  cleanup: Cleanup[];
  replay_note: string;
};
export type BulkTarget = { user_id: string; login: string; role: string; role_until: string; user_state: string; membership_state: string; installation_administrator: boolean };
export type Paged<T> = { total: number; next: string; rows: T[] };
export type BulkExclusion = { user_id: string; login: string; reason: string };

const num = (v: unknown) => (typeof v === "number" && Number.isFinite(v) ? v : 0);
const str = (v: unknown) => (typeof v === "string" ? v : "");
const obj = (v: unknown): Record<string, unknown> => (v && typeof v === "object" && !Array.isArray(v) ? (v as Record<string, unknown>) : {});
const list = (v: unknown): unknown[] => (Array.isArray(v) ? v : []);
const jobStates = ["draft", "prepared", "executed", "discarded", "expired"];
const outcomes = ["excluded", "held", "succeeded", "failed", "unprocessed"];

export function countsOf(v: unknown): BulkCounts {
  const c = obj(v);
  return { targets: num(c.targets), excluded: num(c.excluded), duplicates: num(c.duplicates), succeeded: num(c.succeeded), held: num(c.held), failed: num(c.failed),
    unprocessed: num(c.unprocessed) };
}

export function jobOf(v: unknown): BulkJob {
  const j = obj(v);
  const ledger = obj(j.ledger);
  const cleanup = obj(j.cleanup);
  const state = jobStates.includes(str(j.state)) ? (str(j.state) as BulkJob["state"]) : "expired";
  const effect = j.effect === "suspend_user" || j.effect === "suspend_membership" ? j.effect : "";
  const jobState = j.job_state === "running" || j.job_state === "waiting" || j.job_state === "succeeded" ? j.job_state : "";
  return {
    id: str(j.id), state, origin: str(j.origin), effect, preparer: str(j.preparer), prepared_at: str(j.prepared_at), expires_at: str(j.expires_at), executor: str(j.executor),
    executed_at: str(j.executed_at), deny_at: str(j.deny_at), digest: str(j.digest), counts: countsOf(j.counts), job_state: jobState, job_reason: str(j.job_reason),
    ledger: { first_sequence: num(ledger.first_sequence), last_sequence: num(ledger.last_sequence), watermark: str(ledger.watermark), reason: str(ledger.reason) },
    cleanup: { done: num(cleanup.done), total: num(cleanup.total) }, recorded_at: str(j.recorded_at), staged: num(j.staged),
    selection: j.selection === "filter" || j.selection === "not_recorded" ? j.selection : "explicit", matched: num(j.matched), fixed_at: str(j.fixed_at),
    fixed_revision: str(j.fixed_revision), filter: j.filter && typeof j.filter === "object" ? filterOf(j.filter) : null, search_masked: j.search_masked === true,
  };
}

export function bulkPageOf(body: Record<string, unknown>): BulkPage {
  return { revision: str(body.revision), started_at: str(body.started_at), can_prepare: body.can_prepare === true, open_draft: body.open_draft === true,
    pending: num(body.pending), next: str(body.next), jobs: list(body.jobs).map(jobOf), settings: settingsOf(body.settings), ledger_state: str(body.ledger_state),
    clock: clockOf(body) };
}

export function detailOf(body: Record<string, unknown>): BulkDetail {
  const i = obj(body.impact);
  const exclusions: Record<string, number> = {};
  for (const [k, v] of Object.entries(obj(body.exclusions))) exclusions[k] = num(v);
  return {
    revision: str(body.revision), server_time: str(body.server_time), job: jobOf(body.job), exclusions, outside_scope: list(body.outside_scope).map(String),
    impact: {
      computed_at: str(i.computed_at), people: num(i.people), administrators: num(i.administrators), members: num(i.members),
      installation_administrators: num(i.installation_administrators), trust_right_holders: num(i.trust_right_holders), bulk_right_holders: num(i.bulk_right_holders),
      sessions: num(i.sessions), pickups: num(i.pickups), api_keys: str(i.api_keys), pending_jobs: str(i.pending_jobs), service_users: num(i.service_users), anchors_after: num(i.anchors_after), other_workspaces: str(i.other_workspaces),
      pending: list(i.pending).map((p) => ({ store: str(obj(p).store), count: num(obj(p).count), rechecked: obj(p).rechecked === true })),
    },
  };
}

export function matchCountOf(body: Record<string, unknown>): MatchCount {
  return { count: num(body.count), complete: body.complete === true, over_limit: body.over_limit === true, limit: num(body.limit), scan_budget: num(body.scan_budget),
    list_version: str(body.list_version), server_time: str(body.server_time) };
}

export function countPath(query: UsersQuery): string {
  const q = new URLSearchParams(usersQueryString({ ...query, after: "" }));
  q.delete("limit");
  q.delete("after");
  const rest = q.toString();
  return "/identity/users/count" + (rest ? "?" + rest : "");
}

export function matchValues(query: UsersQuery): Record<string, string> {
  const out: Record<string, string> = {};
  for (const key of ["q", "status", "membership", "role", "period", "signin", "invitation", "holder"] as const) if (query[key]) out[key] = query[key]!;
  return out;
}

export function itemOf(v: unknown): BulkItem {
  const i = obj(v);
  const r = obj(i.receipt);
  return {
    user_id: str(i.user_id), login: str(i.login), role: str(i.role), role_until: str(i.role_until),
    outcome: (outcomes.includes(str(i.outcome)) ? str(i.outcome) : "unprocessed") as BulkItem["outcome"], reason: str(i.reason), state: str(i.state),
    receipt: { sequence: num(r.sequence), watermark: str(r.watermark), reason: str(r.reason) },
    cleanup: list(i.cleanup).map((c) => ({ item: str(obj(c).item), state: str(obj(c).state), reason: str(obj(c).reason) })), replay_note: str(i.replay_note),
  };
}

export function targetOf(v: unknown): BulkTarget {
  const t = obj(v);
  return { user_id: str(t.user_id), login: str(t.login), role: str(t.role), role_until: str(t.role_until), user_state: str(t.user_state),
    membership_state: str(t.membership_state), installation_administrator: t.installation_administrator === true };
}

const base = "/identity/bulk-revocations";

export function bulkClient(api: AdminClient) {
  const page = <T,>(body: Record<string, unknown>, key: string, of: (v: unknown) => T): Paged<T> => ({ total: num(body.total), next: str(body.next), rows: list(body[key]).map(of) });
  return {
    async list(after = "", limit = 50): Promise<BulkPage> {
      const q = new URLSearchParams({ limit: String(limit) });
      if (after) q.set("after", after);
      return bulkPageOf(await api.read(base + "?" + q.toString()));
    },
    async count(query: UsersQuery): Promise<MatchCount> {
      return matchCountOf(await api.read(countPath(query)));
    },
    async detail(id: string): Promise<BulkDetail> {
      return detailOf(await api.read(base + "/" + encodeURIComponent(id)));
    },
    async items(id: string, after = 0, limit = 50, outcome = ""): Promise<Paged<BulkItem>> {
      const q = new URLSearchParams({ limit: String(limit) });
      if (after) q.set("after", String(after));
      if (outcome) q.set("outcome", outcome);
      return page(await api.read(base + "/" + encodeURIComponent(id) + "/items?" + q.toString()), "items", itemOf);
    },
    async excluded(id: string, after = 0, limit = 50): Promise<Paged<BulkExclusion>> {
      const q = new URLSearchParams({ kind: "excluded", limit: String(limit) });
      if (after) q.set("after", String(after));
      return page(await api.read(base + "/" + encodeURIComponent(id) + "/targets?" + q.toString()), "exclusions", (v) => {
        const x = obj(v);
        return { user_id: str(x.user_id), login: str(x.login), reason: str(x.reason) };
      });
    },
    async targets(id: string, after = 0, limit = 50): Promise<Paged<BulkTarget>> {
      const q = new URLSearchParams({ limit: String(limit) });
      if (after) q.set("after", String(after));
      return page(await api.read(base + "/" + encodeURIComponent(id) + "/targets?" + q.toString()), "targets", targetOf);
    },
  };
}

const encoder = new TextEncoder();
export const fieldLimit = 4096;

export function batches(ids: string[], limit = fieldLimit): string[][] {
  const out: string[][] = [];
  let current: string[] = [];
  let size = 0;
  for (const id of ids) {
    const bytes = encoder.encode(id).length;
    const next = current.length === 0 ? bytes : size + 1 + bytes;
    if (current.length > 0 && next > limit) {
      out.push(current);
      current = [];
      size = 0;
    }
    size = current.length === 0 ? bytes : size + 1 + bytes;
    current.push(id);
  }
  if (current.length) out.push(current);
  return out;
}

export type StageProgress = { sent: number; total: number; waiting: boolean; resent?: boolean };
export type StageResult = { draft: string; staged: number; revision: string };
export class StageError extends Error {
  constructor(readonly cause: unknown, readonly draft: string, readonly staged: number, readonly revision: string) {
    super("staging stopped");
  }
}

export type StageOptions = {
  rationale: string;
  revision: string;
  reload: () => Promise<string>;
  progress: (p: StageProgress) => void;
  wait?: (seconds: number) => Promise<void>;
  draft?: string;
  from?: number;
  tries?: Attempt[];
};

const pause = (seconds: number) => new Promise<void>((resolve) => setTimeout(resolve, seconds * 1000));

export async function stage(api: AdminClient, ids: string[], o: StageOptions): Promise<StageResult> {
  const groups = batches(ids);
  let draft = o.draft ?? "";
  let revision = o.revision;
  let sent = 0;
  const tries = o.tries ?? groups.map(() => attempt());
  const start = o.from ?? 0;
  for (let i = 0; i < start; i++) sent += groups[i].length;
  const wait = o.wait ?? pause;
  for (let i = start; i < groups.length; i++) {
    const values: Record<string, string> = { users: groups[i].join("\n") };
    if (draft) values.draft = draft;
    for (;;) {
      try {
        const out = await api.command("bulk_revocation.add_targets", "bulk-revocations", revision, o.rationale, values, tries[i]);
        draft = out.fields.draft ?? draft;
        revision = out.revision || (await o.reload());
        sent += groups[i].length;
        o.progress({ sent, total: ids.length, waiting: false });
        break;
      } catch (e) {
        if (e instanceof AdminError && e.failure === "conflict") {
          o.progress({ sent, total: ids.length, waiting: false, resent: true });
          revision = await o.reload();
          continue;
        }
        if (e instanceof AdminError && e.failure === "limited") {
          o.progress({ sent, total: ids.length, waiting: true });
          await wait(e.retryAfter || 1);
          continue;
        }
        throw new StageError(e, draft, sent, revision);
      }
    }
  }
  return { draft, staged: sent, revision };
}

export function stagedBatches(ids: string[], staged: number): number {
  let n = 0;
  let count = 0;
  for (const g of batches(ids)) {
    if (count >= staged) break;
    count += g.length;
    n++;
  }
  return n;
}
