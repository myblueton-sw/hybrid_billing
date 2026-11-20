import type { AccessSetting } from "./admin";
import type { SelfEndRow, SelfRef, SelfSession } from "./self-end";
import { settingsOf } from "./access-read";
import { array, boolean, choice, integer, optional, ReadParseError, shape, text, type Parser } from "./strict-read";

export type SessionRow = SelfSession & { state: "active" | "blocked" | "expired" | "revoked" | "unavailable"; can_end: boolean };
export type SessionQuery = { session?: "all" | "current" | "other"; workspace?: string; since?: string; until?: string; q?: string; limit?: number; after?: string };
export type SessionPage = { rows: SessionRow[]; next: string; query: Required<Omit<SessionQuery, "after">>; reference_at: string; generation: string;
  order: "current_then_signed_in_desc_handle_asc"; count: { value: number; certainty: "exact" | "lower_bound" }; scan_stopped: boolean; settings: AccessSetting[] };
export type Selection = { selection: string; operation?: string; state: "building" | "sealed" | "applying" | "accepted"; reference_at: string; expires_at: string;
  generation: string; observed: number; count: number | null; digest: string; after: string; settings: AccessSetting[] };
export type SelectionPreview = Selection & { rows: SessionRow[]; next: string };
export type SelectionFingerprint = { selection: string; digest: string; count: number };
export type CleanupCounts = { done: number; already_ended: number; failed: number; pending: number };
export type SessionOperation = { operation: string; selection: string; kind: "one" | "others"; state: "unconfirmed" | "cleanup_running" | "complete" | "cleanup_incomplete";
  recorded_count: number; ended_count: number | null; gone_count: number | null; not_included_count: number | null; blocked_at: string | null;
  list_generation: string; current_generation: string | null; generation_changed: boolean | null; digest: string; cleanup: CleanupCounts | null; replay: boolean };
export type OperationPage = { rows: SessionOperation[]; accepted_count: number; next: string; reference_at: string; generation: string };
export type OperationItems = { operation: string; view: "results" | "not_included"; rows: (SelfEndRow | SelfRef)[]; count: number; next: string; reference_at: string; generation: string };
export type SessionPageQuery = { limit?: number; after?: string };

const nullable = <T>(parse: Parser<T>): Parser<T | null> => (value) => value === null ? null : parse(value);
const bounded = <T>(parse: Parser<T>, max = 100): Parser<T[]> => (value) => {
  if (!Array.isArray(value) || value.length > max) throw new ReadParseError();
  return array(parse)(value);
};
export const sessionRowOf = shape<SessionRow>({ handle: text, current: boolean, signed_in: text, ends: text,
  state: choice("active", "blocked", "expired", "revoked", "unavailable"), can_end: boolean });
const sessionPage = shape<SessionPage>({ rows: bounded(sessionRowOf), next: text, query: shape({ session: choice("all", "current", "other"), workspace: text, since: text, until: text, q: text, limit: integer }),
  reference_at: text, generation: text, order: choice("current_then_signed_in_desc_handle_asc"), count: shape({ value: integer, certainty: choice("exact", "lower_bound") }), scan_stopped: boolean, settings: settingsOf });
const selection = shape<Selection>({ selection: text, operation: optional(text), state: choice("building", "sealed", "applying", "accepted"), reference_at: text, expires_at: text,
  generation: text, observed: integer, count: nullable(integer), digest: text, after: text, settings: settingsOf });
const cleanup = shape<CleanupCounts>({ done: integer, already_ended: integer, failed: integer, pending: integer });
const operation = shape<SessionOperation>({ operation: text, selection: text, kind: choice("one", "others"), state: choice("unconfirmed", "cleanup_running", "complete", "cleanup_incomplete"),
  recorded_count: integer, ended_count: nullable(integer), gone_count: nullable(integer), not_included_count: nullable(integer), blocked_at: nullable(text), list_generation: text,
  current_generation: nullable(text), generation_changed: nullable(boolean), digest: text, cleanup: nullable(cleanup), replay: boolean });

export function sessionPageOf(value: unknown): SessionPage {
  const out = sessionPage(value);
  if (out.query.limit < 1 || out.query.limit > 100 || out.rows.length > out.query.limit) throw new ReadParseError();
  return out;
}
export function selectionOf(value: unknown): Selection {
  const out = selection(value);
  if (out.state === "building" ? out.count !== null || !out.after || !!out.digest : out.count === null || out.count !== out.observed || !out.digest || !!out.after) throw new ReadParseError();
  return out;
}
export function sessionOperationOf(value: unknown): SessionOperation {
  const out = operation(value);
  const facts = [out.ended_count, out.gone_count, out.not_included_count, out.blocked_at, out.current_generation, out.generation_changed, out.cleanup];
  if (out.state === "unconfirmed") {
    if (facts.some((fact) => fact !== null)) throw new ReadParseError();
  } else {
    if (facts.some((fact) => fact === null) || out.ended_count! + out.gone_count! !== out.recorded_count) throw new ReadParseError();
    const c = out.cleanup!;
    if (c.done + c.already_ended + c.failed + c.pending !== out.ended_count || (c.pending > 0 ? "cleanup_running" : c.failed > 0 ? "cleanup_incomplete" : "complete") !== out.state) throw new ReadParseError();
  }
  return out;
}
const operationPage = shape<OperationPage>({ rows: bounded(sessionOperationOf), accepted_count: integer, next: text, reference_at: text, generation: text });
const item = shape({ item: text, state: choice("done", "pending", "failed", "already_ended", "outside_scope", "not_applicable"), reason: (value) => value === undefined ? "" : text(value) });
const resultRow = shape<SelfEndRow>({ handle: text, signed_in: text, result: choice("ended", "already_gone"), items: bounded(item, 8) });
const ref = shape<SelfRef>({ handle: text, signed_in: text });
function operationItemsOf(value: unknown): OperationItems {
  const common = shape({ operation: text, view: choice("results", "not_included"), count: integer, next: text, reference_at: text, generation: text })(value);
  const rows = (value as Record<string, unknown>).rows;
  return { ...common, rows: common.view === "results" ? bounded(resultRow)(rows) : bounded(ref)(rows) };
}
function tail(query: SessionQuery & { view?: string }): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) if (value !== undefined && value !== "") params.set(key, String(value));
  return params.size ? "?" + params.toString() : "";
}

type Send = (path: string, init: RequestInit, request: string) => Promise<Record<string, unknown>>;
type Validate = <T>(parse: () => T) => T;
export function sessionQueryClient(send: Send, request: () => string, validate: Validate) {
  async function read<T>(path: string, parse: Parser<T>): Promise<T> {
    const body = await send(path, { method: "GET" }, request());
    return validate(() => parse(body));
  }
  async function post<T>(path: string, body: unknown, parse: Parser<T>): Promise<T> {
    const result = await send(path, { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(body) }, request());
    return validate(() => parse(result));
  }
  return {
    sessionQuery: (query: SessionQuery) => read("/me/sessions" + tail(query), sessionPageOf),
    sessionDetail: (handle: string) => read("/me/sessions/" + encodeURIComponent(handle), shape({ session: sessionRowOf, reference_at: text })),
    prepareSessionSelection: (resume?: { selection: string; after: string }) => post("/me/sessions/selections", resume ?? {}, selectionOf),
    previewSessionSelection: (reference: string, query: SessionPageQuery = {}) => read("/me/sessions/selections/" + encodeURIComponent(reference) + tail(query), (body): SelectionPreview => ({ ...selectionOf(body), ...shape({ rows: bounded(sessionRowOf), next: text })(body) })),
    endSelectedSessions: (fingerprint: SelectionFingerprint) => post("/me/sessions/end", { selection: fingerprint.selection, digest: fingerprint.digest, count: fingerprint.count }, sessionOperationOf),
    sessionOperations: (query: SessionPageQuery = {}) => read("/me/sessions/operations" + tail(query), operationPage),
    sessionOperation: (operation: string) => read("/me/sessions/operations/" + encodeURIComponent(operation), sessionOperationOf),
    sessionOperationItems: (operation: string, view: "results" | "not_included", query: SessionPageQuery = {}) => read("/me/sessions/operations/" + encodeURIComponent(operation) + "/items" + tail({ ...query, view }), operationItemsOf),
  };
}
