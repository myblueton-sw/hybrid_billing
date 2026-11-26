import type { AccessRequest, CleanupItem, Revocation } from "./admin";
import { array, boolean, choice, integer, text, type Parser } from "./strict-read";
import { closed, decimal, identifier, requireRead, timestamp } from "./service-query-read";

export type ListedRequest = AccessRequest & { subject_login: string; proposer_login: string; decided_by_login: string };
export type ListedRevocation = Revocation & { target_version: number; target_subject: string; target_subject_login: string; target_action: string; target_resource_kind: string; target_fields: string[] };
export type AccessSubject = { user_id: string; login: string; revocation_unconfirmed: boolean };
export type ListEcho = { q: string; state: "" | "pending" | "ended"; limit: number };
export type SubjectEcho = { q: string; id: string; limit: number };
type Page<Row, Echo, Order> = {
  rows: Row[]; search_input: string; query: Echo; next: string; reference_at: string; observed_at: string; revision: string; order: Order;
  count: { value: string; certainty: "exact" | "lower_bound" }; work_units: number; scan_stopped: boolean;
};
export type RequestPage = Page<ListedRequest, ListEcho, "number_desc">;
export type RevocationPage = Page<ListedRevocation, ListEcho, "recorded_desc">;
export type SubjectPage = Page<AccessSubject, SubjectEcho, "user_id_asc">;
export type ListQuery = { q?: string; state?: "pending" | "ended"; limit?: number; after?: string };
export type SubjectQuery = { q?: string; id?: string; limit?: number; after?: string };

const emptyOr = <T>(parse: Parser<T>): Parser<T | ""> => (v) => (v === "" ? "" : parse(v));
const bounded = (max: number): Parser<string> => (v) => { const s = text(v); requireRead(new TextEncoder().encode(s).length <= max); return s; };
const login = bounded(320);
const note = bounded(8192);
const stamp = emptyOr(timestamp);
const ids: Parser<string[]> = (v) => { requireRead(Array.isArray(v) && v.length <= 10000); return array(identifier)(v); };
const natural: Parser<number> = (v) => { const n = integer(v); requireRead(n >= 0); return n; };

const request = closed<ListedRequest>({
  id: identifier, change: choice("add", "revoke"), subject: emptyOr(identifier), grant: emptyOr(identifier), action: emptyOr(identifier), kind: emptyOr(identifier),
  fields: (v) => { requireRead(Array.isArray(v) && v.length <= 100); return array(identifier)(v); }, customer: emptyOr(identifier), days: natural, digest: bounded(256),
  state: choice("pending", "applied", "rejected", "withdrawn", "expired", "superseded"), current: boolean, proposer: identifier, proposed_at: timestamp, expires_at: timestamp,
  reason: note, decided_by: emptyOr(identifier), decided_at: stamp, decision_reason: note, result: bounded(256), eligible_approvers: natural,
  can_approve: boolean, can_reject: boolean, can_withdraw: boolean, may_end: ids, subject_login: login, proposer_login: login, decided_by_login: login,
});
export const listedRequestOf: Parser<ListedRequest> = (v) => {
  const r = request(v);
  requireRead(r.change === "add" ? r.subject !== "" && r.action !== "" && r.kind !== "" : r.grant !== "");
  if (r.state !== "pending") requireRead(!r.can_approve && !r.can_reject && !r.can_withdraw && !r.current && r.eligible_approvers === 0 && r.may_end.length === 0);
  requireRead((r.subject !== "" || r.subject_login === "") && (r.decided_by !== "" || r.decided_by_login === ""));
  return r;
};
const cleanup = closed<CleanupItem>({ item: bounded(128), state: choice("done", "not_applicable", "outside_scope", "not_started"), reason: bounded(256), refs: (v) => { requireRead(Array.isArray(v) && v.length <= 10000); return array(bounded(256))(v); } });
const revocation = closed<ListedRevocation>({
  id: identifier, origin: choice("this_process", "ledger_replay"), recorded_at: timestamp, target_kind: choice("grant", "delegation"), target: identifier, target_version: natural,
  target_present: boolean, replay: choice("", "reapplied", "absent", "protected"),
  state: choice("received", "deny_confirmed", "cleanup_running", "revocation_complete", "decision_unconfirmed"),
  history: (v) => { requireRead(Array.isArray(v) && v.length >= 1 && v.length <= 100); return array(closed<{ state: string; at: string }>({ state: bounded(64), at: timestamp }))(v); },
  ended_delegations: ids, deny_generation: natural, job_state: choice("running", "waiting", "succeeded"), job_reason: bounded(256),
  requested_by: emptyOr(identifier), approved_by: emptyOr(identifier), request: emptyOr(identifier), received_at: timestamp, deny_at: timestamp,
  receipt: closed<{ sequence: number; watermark: string; reason: string }>({ sequence: natural, watermark: bounded(256), reason: bounded(256) }), verified: boolean,
  cleanup: (v) => { requireRead(Array.isArray(v) && v.length <= 100); return array(cleanup)(v); }, completed_at: stamp,
  target_subject: emptyOr(identifier), target_subject_login: login, target_action: emptyOr(identifier), target_resource_kind: emptyOr(identifier),
  target_fields: (v) => { requireRead(Array.isArray(v) && v.length <= 100); return array(identifier)(v); },
});
export const listedRevocationOf: Parser<ListedRevocation> = (v) => {
  const r = revocation(v);
  requireRead(r.received_at === r.history[0].at && r.verified === (r.receipt.reason === "" && r.receipt.sequence > 0));
  requireRead((r.target_subject !== "" || r.target_subject_login === "") && (r.target_action === "") === (r.target_resource_kind === "") && (r.target_action !== "" || r.target_fields.length === 0));
  return r;
};
const subject = closed<AccessSubject>({ user_id: identifier, login, revocation_unconfirmed: boolean });

function pageOf<Row, Echo extends { limit: number }, Order extends string>(row: Parser<Row>, echo: Parser<Echo>, order: Order, ordered: (a: Row, b: Row) => boolean): Parser<Page<Row, Echo, Order>> {
  const parse = closed<Page<Row, Echo, Order>>({
    rows: (v) => { requireRead(Array.isArray(v) && v.length <= 100); return array(row)(v); }, search_input: text, query: echo, next: text, reference_at: timestamp, observed_at: timestamp,
    revision: (v) => { const s = text(v); requireRead(s.length > 0); return s; }, order: choice(order) as Parser<Order>,
    count: closed({ value: decimal, certainty: choice("exact", "lower_bound") }), work_units: integer, scan_stopped: boolean,
  });
  return (v) => {
    const r = parse(v);
    requireRead(r.query.limit >= 1 && r.query.limit <= 100 && r.rows.length <= r.query.limit && r.next.length <= 4096 && r.work_units >= 0 && r.work_units <= 10000);
    requireRead(!!r.next === (r.count.certainty === "lower_bound") && (!r.scan_stopped || !!r.next) && BigInt(r.count.value) >= BigInt(r.rows.length));
    requireRead(Date.parse(r.reference_at) <= Date.parse(r.observed_at));
    for (let i = 1; i < r.rows.length; i++) requireRead(ordered(r.rows[i - 1], r.rows[i]));
    return r;
  };
}
const listEcho = closed<ListEcho>({ q: text, state: emptyOr(choice("pending", "ended")), limit: integer });
export const requestPageOf = pageOf(listedRequestOf, listEcho, "number_desc", (a, b) => a.id !== b.id);
export const revocationPageOf = pageOf(listedRevocationOf, listEcho, "recorded_desc", (a, b) => a.id !== b.id);
export const subjectPageOf = pageOf(subject, closed<SubjectEcho>({ q: text, id: emptyOr(identifier), limit: integer }), "user_id_asc", (a, b) => a.user_id < b.user_id);

type Send = (path: string, init: RequestInit, request: string) => Promise<unknown>;
type Validate = <T>(run: () => T) => T;

function sent(query: Record<string, unknown>, allowed: readonly string[], states: readonly string[]): Record<string, string> {
  requireRead(Object.keys(query).every((k) => allowed.includes(k)));
  const out: Record<string, string> = {};
  for (const key of allowed) {
    const value = query[key];
    if (value === undefined || value === "") continue;
    if (key === "limit") { requireRead(Number.isSafeInteger(value) && (value as number) >= 1 && (value as number) <= 100); out[key] = String(value); continue; }
    requireRead(typeof value === "string");
    const s = value as string;
    if (key === "after") requireRead(s.length <= 4096);
    else if (key === "q") { const trimmed = s.trim(); requireRead(new TextEncoder().encode(trimmed).length <= 64 && trimmed !== ""); out[key] = trimmed; continue; }
    else if (key === "state") requireRead(states.includes(s));
    else identifier(s);
    out[key] = s;
  }
  return out;
}

export function accessListClient(send: Send, request: () => string, validate: Validate) {
  const read = async <P extends { search_input: string; query: { q: string; limit: number }; rows: unknown[]; count: { value: string } }>(
    path: string, captured: Record<string, string>, parse: Parser<P>, fits: (page: P) => boolean): Promise<P> => {
    const tail = new URLSearchParams(captured).toString();
    const body = await send(path + (tail ? "?" + tail : ""), { method: "GET" }, request());
    return validate(() => {
      const page = parse(body);
      requireRead(page.search_input === (captured.q ?? "") && (page.query.q !== "") === (captured.q !== undefined) && (captured.limit === undefined || page.query.limit === Number(captured.limit)));
      requireRead(fits(page));
      if (!captured.after) requireRead(BigInt(page.count.value) === BigInt(page.rows.length));
      return page;
    });
  };
  return {
    async accessRequests(query: ListQuery): Promise<RequestPage> {
      const captured = validate(() => sent(query, ["q", "state", "limit", "after"], ["pending", "ended"]));
      return read("/identity/access/requests", captured, requestPageOf, (page) => page.query.state === (captured.state ?? "") && page.rows.every((r) =>
        (!page.query.state || (r.state === "pending") === (page.query.state === "pending")) && (!page.query.q || r.id.toLowerCase().includes(page.query.q) || r.subject.toLowerCase().includes(page.query.q))));
    },
    async accessRevocations(query: Omit<ListQuery, "state">): Promise<RevocationPage> {
      const captured = validate(() => sent(query, ["q", "limit", "after"], []));
      return read("/identity/access/revocations", captured, revocationPageOf, (page) => page.query.state === "" && page.rows.every((r) =>
        !page.query.q || r.id.toLowerCase().includes(page.query.q) || r.target.toLowerCase().includes(page.query.q)));
    },
    async accessSubjects(query: SubjectQuery): Promise<SubjectPage> {
      const captured = validate(() => { const out = sent(query, ["q", "id", "limit", "after"], []); requireRead(!out.id || (!out.q && !out.after)); return out; });
      return read("/identity/access/subjects", captured, subjectPageOf, (page) => page.query.id === (captured.id ?? "") && page.rows.every((r) =>
        !page.query.id || r.user_id === page.query.id));
    },
  };
}
