export type ContractRow = {
  id: string;
  name: string;
  profile: string;
  profile_name: string;
  buyer: string;
  payer: string;
  issuer: string;
  path: string;
  currency: string;
  timezone: string;
  cycle: string;
  state: string;
  lifecycle_revision: number | null;
  settlement: string;
  sequence: number | null;
  start: string;
  end: string;
  open_gaps: number | null;
};

export type ContractImpact = { money: string; proration: string; publication: string; erp: string; final_amount_decision: string };

export type ContractsPage = { revision: string; on: string; today: string; impact: ContractImpact; contracts: ContractRow[] };

export type ContractComponent = { id: string; kind: string; provider_payment_responsible: string; seller_billable: boolean | null };
export type ContractTerms = { revision: number | null; customer: string; seller: string; path: string; components: ContractComponent[] };
export type ContractEvent = { id: string; kind: string; from: string; revision: number | null; withdraw_blocked: string; cancel_required: boolean };
export type ContractSegment = { from: string; until: string; days: number | null; state: string; revision: number | null };
export type ContractEvidence = {
  sequence: number | null;
  write: string;
  kind: string;
  reference: string;
  covered_until: string;
  resolves: string[];
  reason: string;
  owner: string;
  owner_login: string;
  follow_up: string;
  resolved: boolean;
};
export type ContractRecord = {
  sequence: number | null;
  at: string;
  write: string;
  kind: string;
  added: string[];
  withdrawn: string[];
  cancel: string;
  as_of: string;
  reopen: string;
  state: string;
  actor: string;
  actor_login: string;
  reason: string;
};
export type ContractAction = { allowed: boolean; blocked: string; earliest: string };
export type ContractActionKey =
  | "contract.suspend"
  | "contract.resume"
  | "contract.renew"
  | "contract.end"
  | "contract.withdraw"
  | "contract.evidence.secured"
  | "contract.evidence.missing"
  | "contract.settle"
  | "contract.reopen";
export const actionKeys: ContractActionKey[] = [
  "contract.suspend",
  "contract.resume",
  "contract.renew",
  "contract.end",
  "contract.withdraw",
  "contract.evidence.secured",
  "contract.evidence.missing",
  "contract.settle",
  "contract.reopen",
];

export type ContractDetail = {
  revision: string;
  today: string;
  from: string;
  until: string;
  impact: ContractImpact;
  contract: ContractRow;
  terms: ContractTerms[];
  events: ContractEvent[];
  segments: ContractSegment[];
  reopenings: number | null;
  evidence: ContractEvidence[];
  records: ContractRecord[];
  earlier_records: number | null;
  actions: Record<ContractActionKey, ContractAction>;
};

type Raw = Record<string, unknown>;

const obj = (v: unknown): Raw => (v && typeof v === "object" && !Array.isArray(v) ? (v as Raw) : {});
const str = (v: unknown): string => (typeof v === "string" ? v : typeof v === "number" ? String(v) : "");
const num = (v: unknown): number | null => (typeof v === "number" && Number.isFinite(v) ? v : null);
const strs = (v: unknown): string[] => (Array.isArray(v) ? v.filter((x) => typeof x === "string") : []);
const rows = <T,>(v: unknown, f: (r: Raw) => T): T[] => (Array.isArray(v) ? v.filter((x) => x && typeof x === "object" && !Array.isArray(x)).map((x) => f(x as Raw)) : []);
const flag = (v: unknown): boolean | null => (typeof v === "boolean" ? v : null);

export function impactOf(v: unknown): ContractImpact {
  const r = obj(v);
  const one = (x: unknown) => (typeof x === "string" && x ? x : "unknown");
  return { money: one(r.money), proration: one(r.proration), publication: one(r.publication), erp: one(r.erp), final_amount_decision: one(r.final_amount_decision) };
}

export function contractRowOf(r: Raw): ContractRow {
  return {
    id: str(r.id),
    name: str(r.name),
    profile: str(r.profile),
    profile_name: str(r.profile_name),
    buyer: str(r.buyer),
    payer: str(r.payer),
    issuer: str(r.issuer),
    path: str(r.path),
    currency: str(r.currency),
    timezone: str(r.timezone),
    cycle: str(r.cycle),
    state: str(r.state),
    lifecycle_revision: num(r.lifecycle_revision),
    settlement: str(r.settlement),
    sequence: num(r.sequence),
    start: str(r.start),
    end: str(r.end),
    open_gaps: num(r.open_gaps),
  };
}

export function contractsPageOf(body: Raw): ContractsPage {
  return { revision: str(body.revision), on: str(body.on), today: str(body.today), impact: impactOf(body.impact), contracts: rows(body.contracts, contractRowOf) };
}

function actionOf(v: unknown): ContractAction {
  const r = obj(v);
  if (!("allowed" in r)) return { allowed: false, blocked: "missing", earliest: "" };
  const blocked = str(r.blocked);
  if (r.allowed === true && blocked === "") return { allowed: true, blocked: "", earliest: str(r.earliest) };
  return { allowed: false, blocked: blocked || "unknown", earliest: "" };
}

export function contractDetailOf(body: Raw): ContractDetail {
  const actions = obj(body.actions);
  return {
    revision: str(body.revision),
    today: str(body.today),
    from: str(body.from),
    until: str(body.until),
    impact: impactOf(body.impact),
    contract: contractRowOf(obj(body.contract)),
    terms: rows(body.terms, (t) => ({
      revision: num(t.revision),
      customer: str(t.customer),
      seller: str(t.seller),
      path: str(t.path),
      components: rows(t.components, (c) => ({ id: str(c.id), kind: str(c.kind), provider_payment_responsible: str(c.provider_payment_responsible), seller_billable: flag(c.seller_billable) })),
    })),
    events: rows(body.events, (e) => ({
      id: str(e.id),
      kind: str(e.kind),
      from: str(e.from),
      revision: num(e.revision),
      withdraw_blocked: typeof e.withdraw_blocked === "string" ? e.withdraw_blocked : "missing",
      cancel_required: e.cancel_required === true,
    })),
    segments: rows(body.segments, (g) => ({ from: str(g.from), until: str(g.until), days: num(g.days), state: str(g.state), revision: num(g.revision) })),
    reopenings: num(body.reopenings),
    evidence: rows(body.evidence, (e) => ({
      sequence: num(e.sequence),
      write: str(e.write),
      kind: str(e.kind),
      reference: str(e.reference),
      covered_until: str(e.covered_until),
      resolves: strs(e.resolves),
      reason: str(e.reason),
      owner: str(e.owner),
      owner_login: str(e.owner_login),
      follow_up: str(e.follow_up),
      resolved: e.resolved === true,
    })),
    records: rows(body.records, (r) => ({
      sequence: num(r.sequence),
      at: str(r.at),
      write: str(r.write),
      kind: str(r.kind),
      added: strs(r.added),
      withdrawn: strs(r.withdrawn),
      cancel: str(r.cancel),
      as_of: str(r.as_of),
      reopen: str(r.reopen),
      state: str(r.state),
      actor: str(r.actor),
      actor_login: str(r.actor_login),
      reason: str(r.reason),
    })),
    earlier_records: num(body.earlier_records),
    actions: Object.fromEntries(actionKeys.map((k) => [k, actionOf(actions[k])])) as Record<ContractActionKey, ContractAction>,
  };
}
