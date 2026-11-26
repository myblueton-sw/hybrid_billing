import { trustChangeOf, type RevisionRow, type TrustChange, type TrustState } from "./admin";
import { array, boolean, choice, integer, record, text, type Parser } from "./strict-read";
import { closed, decimal, identifier, requireRead, timestamp } from "./service-query-read";

export const trustStates = ["draft", "validated", "independently_approved", "activation_unknown", "activated", "rejected", "cancelled", "superseded"] as const;
export const trustGroups = { open: ["draft", "validated", "independently_approved", "activation_unknown"], closed: ["activated", "rejected", "cancelled", "superseded"] } as const;
export type TrustFilter = "open" | "closed" | TrustState;
export type TrustReason = "" | "trust_unavailable" | "no_right" | "self" | "approver" | "not_validated" | "needs_approval";
export type ListedTrustChange = {
  id: string; number: number; state: TrustState; change_revision: number; candidate: RevisionRow; proposer: string; validator: string; approver: string; activator: string;
  superseded_by: string; earlier_run: boolean; run_started_at: string; validation_result: "" | "validated" | "blocked"; evidence_current: "not_evaluated" | "not_applicable";
  invalidation_state: "" | "complete" | "in_progress"; decide_in_detail: boolean; can: { revise: boolean; validate: boolean; reject: boolean; cancel: boolean };
  why_not: { approve: TrustReason; activate: TrustReason }; act_until: string;
};
export type TrustChangeEcho = { q: string; state: "" | TrustFilter; limit: number };
export type TrustChangePage = {
  rows: ListedTrustChange[]; search_input: string; query: TrustChangeEcho; next: string; reference_at: string; observed_at: string; revision: string; order: "created_desc";
  count: { value: string; certainty: "exact" }; work_units: number; scan_stopped: boolean;
};
export type TrustChangeDetail = { change: TrustChange; revision: string; observed_at: string };
export type TrustChangeQuery = { q?: string; state?: TrustFilter; limit?: number; after?: string };

const filters: readonly string[] = ["open", "closed", ...trustStates];
const emptyOr = <T>(parse: Parser<T>): Parser<T | ""> => (v) => (v === "" ? "" : parse(v));
const natural: Parser<number> = (v) => { const n = integer(v); requireRead(n >= 0); return n; };
const seconds: Parser<number> = (v) => { const n = integer(v); requireRead(n > 0); return n; };
const nonempty: Parser<string> = (v) => { const s = text(v); requireRead(s.length > 0); return s; };
const ascii = (s: string) => /^[ -~]*$/.test(s);
const reason = choice("", "trust_unavailable", "no_right", "self", "approver", "not_validated", "needs_approval");
const candidate = closed<RevisionRow>({
  methods: (v) => { requireRead(Array.isArray(v) && v.length >= 1 && v.length <= 8); return array(identifier)(v); }, mfa: identifier, max_age_seconds: seconds,
  step_up: (v) => { requireRead(Array.isArray(v) && v.length <= 100); return array(closed<RevisionRow["step_up"][number]>({ action: identifier, max_age_seconds: seconds, mfa: boolean }))(v); },
});
const row = closed<ListedTrustChange>({
  id: identifier, number: natural, state: choice(...trustStates), change_revision: natural, candidate, proposer: emptyOr(identifier), validator: emptyOr(identifier),
  approver: emptyOr(identifier), activator: emptyOr(identifier), superseded_by: emptyOr(identifier), earlier_run: boolean, run_started_at: emptyOr(timestamp),
  validation_result: choice("", "validated", "blocked"), evidence_current: choice("not_evaluated", "not_applicable"), invalidation_state: choice("", "complete", "in_progress"),
  decide_in_detail: boolean, can: closed<ListedTrustChange["can"]>({ revise: boolean, validate: boolean, reject: boolean, cancel: boolean }),
  why_not: closed<ListedTrustChange["why_not"]>({ approve: reason, activate: reason }), act_until: timestamp,
});
const awaiting = (state: string) => state === "draft" || state === "validated" || state === "independently_approved";
export const listedTrustChangeOf: Parser<ListedTrustChange> = (v) => {
  const r = row(v);
  const open = awaiting(r.state);
  requireRead(r.decide_in_detail === open && (r.evidence_current === "not_evaluated") === (open && r.validation_result !== ""));
  requireRead(open || (!r.can.revise && !r.can.validate && !r.can.reject && !r.can.cancel));
  requireRead((r.superseded_by !== "") === (r.state === "superseded") && (r.invalidation_state === "" || r.state === "activated"));
  requireRead(!(r.can.revise || r.can.cancel) || !r.can.reject);
  return r;
};
const page = closed<TrustChangePage>({
  rows: (v) => { requireRead(Array.isArray(v) && v.length <= 100); return array(listedTrustChangeOf)(v); }, search_input: text,
  query: closed<TrustChangeEcho>({ q: text, state: emptyOr(choice("open", "closed", ...trustStates)), limit: integer }), next: text, reference_at: timestamp, observed_at: timestamp,
  revision: nonempty, order: choice("created_desc"), count: closed<TrustChangePage["count"]>({ value: decimal, certainty: choice("exact") }), work_units: natural, scan_stopped: boolean,
});
export const trustChangePageOf: Parser<TrustChangePage> = (v) => {
  const r = page(v);
  requireRead(r.query.limit >= 1 && r.query.limit <= 100 && r.rows.length <= r.query.limit && r.next.length <= 4096 && r.work_units <= 10000);
  requireRead(BigInt(r.count.value) >= BigInt(r.rows.length) && (r.next !== "" || !r.scan_stopped) && (r.next === "" || r.rows.length === r.query.limit));
  requireRead(Date.parse(r.reference_at) <= Date.parse(r.observed_at) && new Set(r.rows.map((x) => x.id)).size === r.rows.length);
  return r;
};
export const trustChangeDetailOf: Parser<TrustChangeDetail> = (v) => {
  const outer = closed<{ change: Record<string, unknown>; revision: string; observed_at: string }>({ change: record, revision: nonempty, observed_at: timestamp })(v);
  identifier(outer.change.id);
  choice(...trustStates)(outer.change.state);
  return { change: trustChangeOf(outer.change), revision: outer.revision, observed_at: outer.observed_at };
};

type Send = (path: string, init: RequestInit, request: string) => Promise<unknown>;
type Validate = <T>(run: () => T) => T;

function sent(query: TrustChangeQuery): Record<string, string> {
  const allowed = ["q", "state", "limit", "after"];
  requireRead(Object.keys(query).every((k) => allowed.includes(k)));
  const out: Record<string, string> = {};
  if (query.q !== undefined && query.q !== "") {
    requireRead(typeof query.q === "string");
    const trimmed = query.q.trim();
    requireRead(trimmed !== "" && new TextEncoder().encode(trimmed).length <= 64);
    out.q = trimmed;
  }
  if (query.state !== undefined) { requireRead(filters.includes(query.state)); out.state = query.state; }
  if (query.limit !== undefined) { requireRead(Number.isSafeInteger(query.limit) && query.limit >= 1 && query.limit <= 100); out.limit = String(query.limit); }
  if (query.after !== undefined && query.after !== "") { requireRead(typeof query.after === "string" && query.after.length <= 4096); out.after = query.after; }
  return out;
}

export function trustChangeClient(send: Send, request: () => string, validate: Validate) {
  return {
    async trustChanges(query: TrustChangeQuery): Promise<TrustChangePage> {
      const captured = validate(() => sent(query));
      const tail = new URLSearchParams(captured).toString();
      const body = await send("/identity/policy/changes" + (tail ? "?" + tail : ""), { method: "GET" }, request());
      return validate(() => {
        const got = trustChangePageOf(body);
        const admitted: readonly string[] = got.query.state === "" ? trustStates : got.query.state === "open" || got.query.state === "closed" ? trustGroups[got.query.state] : [got.query.state];
        requireRead(got.search_input === (captured.q ?? "") && (ascii(captured.q ?? "") ? got.query.q === (captured.q ?? "").toLowerCase() : got.query.q !== "") && got.query.state === (captured.state ?? ""));
        requireRead(captured.limit === undefined || got.query.limit === Number(captured.limit));
        requireRead(got.rows.every((r) => admitted.includes(r.state) && r.id.includes(got.query.q)));
        if (!captured.after && got.next === "") requireRead(BigInt(got.count.value) === BigInt(got.rows.length));
        return got;
      });
    },
    async trustChange(id: string): Promise<TrustChangeDetail> {
      const wanted = validate(() => identifier(id));
      const body = await send("/identity/policy/changes/" + encodeURIComponent(wanted), { method: "GET" }, request());
      return validate(() => { const got = trustChangeDetailOf(body); requireRead(got.change.id === wanted); return got; });
    },
  };
}
