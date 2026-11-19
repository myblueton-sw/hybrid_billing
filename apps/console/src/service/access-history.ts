export const historyKinds = [
  "signed_in",
  "sign_in_refused",
  "sign_in_refused_more",
  "password_confirmed",
  "password_confirmation_failed",
  "signed_out",
  "session_ended",
  "password_changed",
] as const;
export const historyNoteKinds = [
  "unclean_stop",
  "dropped",
  "damaged",
  "recovered_tail",
  "early_deletion",
  "deletion_overdue",
  "purge_deferred",
  "held",
  "future_file",
] as const;
export const historyBrowsers = ["edge", "chrome", "firefox", "safari", "other", "unknown"] as const;
export const historySystems = ["windows", "macos", "linux", "ios", "android", "chromeos", "other", "unknown"] as const;
const reasons: Record<string, string[]> = {
  sign_in_refused: ["wrong_password", "locked", "credentials_changed", "access_refused", "session_unavailable"],
  password_confirmation_failed: ["wrong_password", "attempt_limit"],
  session_ended: [
    "ended_by_you",
    "password_changed",
    "password_reset",
    "attempt_limit",
    "administrator",
    "role_period_ended",
    "session_limit",
    "access_changed",
    "trust_change",
    "bulk_revocation",
  ],
};

export type HistoryKind = (typeof historyKinds)[number];
export type HistoryNoteKind = (typeof historyNoteKinds)[number];
export type HistoryBrowser = (typeof historyBrowsers)[number];
export type HistorySystem = (typeof historySystems)[number];
export type HistoryEvent = {
  at: string;
  kind: HistoryKind;
  reason: string;
  workspace: string | null;
  session: { handle: string; current: boolean; active: boolean } | null;
  device: { browser: HistoryBrowser; os: HistorySystem } | null;
  count: number;
};
export type HistoryNote = { kind: HistoryNoteKind; from: string | null; to: string | null; count: number | null };
export type AccessHistoryPage = {
  events: HistoryEvent[];
  next: string | null;
  retention_days: number;
  retention_source: "default" | "start_flag";
  period_since: string | null;
  effective_days: number;
  available_from: string | null;
  recording: { state: "recording" | "stopped"; since: string | null; reason: string | null };
  notes: HistoryNote[];
  now: string;
};
export type AccessHistoryQuery = { from?: string; before?: string };

export class HistoryParseError extends Error {}

const rec = (v: unknown): Record<string, unknown> => {
  if (!v || typeof v !== "object" || Array.isArray(v)) throw new HistoryParseError("object");
  return v as Record<string, unknown>;
};
const str = (v: unknown): string => {
  if (typeof v !== "string") throw new HistoryParseError("string");
  return v;
};
const optStr = (v: unknown): string | null => (v === null || v === undefined ? null : str(v));
const int = (v: unknown): number => {
  if (typeof v !== "number" || !Number.isInteger(v) || v < 0) throw new HistoryParseError("integer");
  return v;
};
function oneOf<T extends string>(list: readonly T[], v: unknown): T {
  const s = str(v);
  const found = list.find((x) => x === s);
  if (found === undefined) throw new HistoryParseError("enum " + s);
  return found;
}
const arr = (v: unknown): unknown[] => {
  if (!Array.isArray(v)) throw new HistoryParseError("array");
  return v;
};

function eventOf(v: unknown): HistoryEvent {
  const r = rec(v);
  const kind = oneOf(historyKinds, r.kind);
  const reason = str(r.reason);
  const allowed = reasons[kind];
  if (allowed ? !allowed.includes(reason) : reason !== "") throw new HistoryParseError("reason " + reason);
  const s = r.session === null || r.session === undefined ? null : rec(r.session);
  const d = r.device === null || r.device === undefined ? null : rec(r.device);
  return {
    at: str(r.at),
    kind,
    reason,
    workspace: optStr(r.workspace),
    session: s ? { handle: str(s.handle), current: s.current === true, active: s.active === true } : null,
    device: d ? { browser: oneOf(historyBrowsers, d.browser), os: oneOf(historySystems, d.os) } : null,
    count: int(r.count),
  };
}

function noteOf(v: unknown): HistoryNote {
  const r = rec(v);
  const count = r.count === null || r.count === undefined ? null : int(r.count);
  return { kind: oneOf(historyNoteKinds, r.kind), from: optStr(r.from), to: optStr(r.to), count };
}

export function accessHistoryOf(body: unknown): AccessHistoryPage {
  const r = rec(body);
  const recording = rec(r.recording);
  return {
    events: arr(r.events).map(eventOf),
    next: optStr(r.next),
    retention_days: int(r.retention_days),
    retention_source: oneOf(["default", "start_flag"] as const, r.retention_source),
    period_since: optStr(r.period_since),
    effective_days: int(r.effective_days),
    available_from: optStr(r.available_from),
    recording: { state: oneOf(["recording", "stopped"] as const, recording.state), since: optStr(recording.since), reason: optStr(recording.reason) },
    notes: arr(r.notes).map(noteOf).filter((n) => n.count !== 0),
    now: str(r.now),
  };
}
