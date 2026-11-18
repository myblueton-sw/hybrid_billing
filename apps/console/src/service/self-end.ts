export type SelfSession = { handle: string; current: boolean; signed_in: string; ends: string };
export type SelfItemState = "done" | "pending" | "failed" | "already_ended" | "outside_scope" | "not_applicable";
export type SelfItem = { item: string; state: SelfItemState; reason: string };
export type SelfEndRow = { handle: string; signed_in: string; result: "ended" | "already_gone"; items: SelfItem[] };
export type SelfRef = { handle: string; signed_in: string };
export type SelfEndState = "cleanup_running" | "complete" | "cleanup_incomplete";
export type SelfEnd = {
  operation: string;
  kind: "others" | "one";
  count: number;
  state: SelfEndState;
  blocked_at: string;
  list_generation: string;
  current_generation: string;
  generation_changed: boolean;
  digest: string;
  cleanup: { done: number; already_ended: number; failed: number; pending: number };
  sessions: SelfEndRow[];
  not_included: SelfRef[];
  replay: boolean;
};

const itemStates: SelfItemState[] = ["done", "pending", "failed", "already_ended", "outside_scope", "not_applicable"];
const endStates: SelfEndState[] = ["cleanup_running", "complete", "cleanup_incomplete"];

const rec = (v: unknown): Record<string, unknown> => (v && typeof v === "object" ? (v as Record<string, unknown>) : {});
const list = (v: unknown): unknown[] => (Array.isArray(v) ? v : []);
const str = (v: unknown): string => (typeof v === "string" ? v : "");
const num = (v: unknown): number => (typeof v === "number" && Number.isFinite(v) ? v : 0);

export function selfSessionOf(v: unknown): SelfSession {
  const r = rec(v);
  return { handle: str(r.handle), current: r.current === true, signed_in: str(r.signed_in), ends: str(r.ends) };
}

function itemOf(v: unknown): SelfItem {
  const r = rec(v);
  const state = str(r.state) as SelfItemState;
  return { item: str(r.item), state: itemStates.includes(state) ? state : "pending", reason: str(r.reason) };
}

function rowOf(v: unknown): SelfEndRow {
  const r = rec(v);
  return { handle: str(r.handle), signed_in: str(r.signed_in), result: r.result === "ended" ? "ended" : "already_gone", items: list(r.items).map(itemOf) };
}

export function selfEndOf(v: unknown): SelfEnd {
  const r = rec(v);
  const c = rec(r.cleanup);
  const state = str(r.state) as SelfEndState;
  return {
    operation: str(r.operation),
    kind: r.kind === "one" ? "one" : "others",
    count: num(r.count),
    state: endStates.includes(state) ? state : "cleanup_running",
    blocked_at: str(r.blocked_at),
    list_generation: str(r.list_generation),
    current_generation: str(r.current_generation),
    generation_changed: r.generation_changed === true,
    digest: str(r.digest),
    cleanup: { done: num(c.done), already_ended: num(c.already_ended), failed: num(c.failed), pending: num(c.pending) },
    sessions: list(r.sessions).map(rowOf),
    not_included: r.kind === "one" ? [] : list(r.not_included).map((x) => ({ handle: str(rec(x).handle), signed_in: str(rec(x).signed_in) })),
    replay: r.replay === true,
  };
}
