import type { SelectionFingerprint, SessionOperation, SessionQuery } from "../../../../service/session-query";

export type SessionFilters = { session: "all" | "current" | "other"; q: string; since: string; until: string; limit?: number };
export type SessionNavigation = { filters: SessionFilters; after: string; bad: boolean };
export type TerminationAttempt = { state: "idle" } | { state: "submitting" | "unknown" | "recovering" | "unavailable"; fingerprint: SelectionFingerprint }
  | { state: "accepted"; fingerprint: SelectionFingerprint; operation: SessionOperation };
const keys = ["session", "sq", "ssince", "suntil", "slimit", "safter"] as const;

function timestampNanos(value: string): bigint | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{1,9}))?(Z|[+-]\d{2}:\d{2})$/.exec(value);
  if (!match) return null;
  const [,year,month,day,hour,minute,second,fraction = "",zone] = match;
  const y = Number(year), m = Number(month), d = Number(day);
  const leap = y % 4 === 0 && (y % 100 !== 0 || y % 400 === 0);
  const days = [31,leap ? 29 : 28,31,30,31,30,31,31,30,31,30,31];
  if (m < 1 || m > 12 || d < 1 || d > days[m-1] || Number(hour) > 23 || Number(minute) > 59 || Number(second) > 59) return null;
  if (zone !== "Z" && (Number(zone.slice(1,3)) > 23 || Number(zone.slice(4,6)) > 59)) return null;
  const millis = Date.parse(`${year}-${month}-${day}T${hour}:${minute}:${second}${zone}`);
  return Number.isFinite(millis) ? BigInt(millis)*1000000n+BigInt(fraction.padEnd(9,"0")) : null;
}

export function parseSessionLink(params: URLSearchParams): SessionNavigation {
  const filters: SessionFilters = { session: "all", q: "", since: "", until: "" };
  let bad = false;
  let after = "";
  for (const key of keys) {
    const values = params.getAll(key);
    if (values.length === 0) continue;
    const value = values[0];
    if (values.length !== 1 || value === "") { bad = true; continue; }
    if (key === "session") {
      if (value === "all" || value === "current" || value === "other") filters.session = value;
      else bad = true;
    } else if (key === "slimit") {
      if (/^[0-9]+$/.test(value) && Number(value) >= 1 && Number(value) <= 100) filters.limit = Number(value);
      else bad = true;
    } else if (key === "safter") {
      if (value.length <= 4096) after = value;
      else bad = true;
    } else if (key === "sq") {
      const normalized = value.trim();
      if (normalized && new TextEncoder().encode(normalized).length <= 64 && !/[\u0000-\u001f\u007f-\u009f]/u.test(normalized)) filters.q = normalized;
      else bad = true;
    } else {
      if (value.length <= 40 && timestampNanos(value) !== null) filters[key === "ssince" ? "since" : "until"] = value;
      else bad = true;
    }
  }
  if (filters.since && filters.until && timestampNanos(filters.since)! >= timestampNanos(filters.until)!) bad = true;
  return { filters, after, bad };
}

export function sessionLink(filters: SessionFilters, after = ""): Record<(typeof keys)[number], string | null> {
  return { session: filters.session === "all" ? null : filters.session, sq: filters.q || null, ssince: filters.since || null,
    suntil: filters.until || null, slimit: filters.limit === undefined ? null : String(filters.limit), safter: after || null };
}

export function sessionRequest(navigation: SessionNavigation): SessionQuery {
  const { filters, after } = navigation;
  return { session: filters.session, ...(filters.q ? { q: filters.q } : {}), ...(filters.since ? { since: filters.since } : {}),
    ...(filters.until ? { until: filters.until } : {}), ...(filters.limit === undefined ? {} : { limit: filters.limit }), ...(after ? { after } : {}) };
}

export function submittedFingerprint(attempt: TerminationAttempt): SelectionFingerprint | null {
  return attempt.state === "idle" ? null : attempt.fingerprint;
}

export function canStartTermination(attempt: TerminationAttempt): boolean {
  return attempt.state === "idle" || attempt.state === "accepted";
}
