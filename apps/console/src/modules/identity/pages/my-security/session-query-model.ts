import type { SelectionFingerprint, SessionOperation, SessionQuery } from "../../../../service/session-query";

export type SessionFilters = { session: "all" | "current" | "other"; q: string; since: string; until: string; limit?: number };
export type SessionNavigation = { filters: SessionFilters; after: string; bad: boolean };
export type TerminationAttempt = { state: "idle" } | { state: "submitting" | "unknown" | "recovering" | "unavailable"; fingerprint: SelectionFingerprint }
  | { state: "accepted"; fingerprint: SelectionFingerprint; operation: SessionOperation };
const keys = ["session", "sq", "ssince", "suntil", "slimit", "safter"] as const;

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
      if (new TextEncoder().encode(value).length <= 64 && !/[\u0000-\u001f\u007f-\u009f]/u.test(value)) filters.q = value;
      else bad = true;
    } else {
      if (value.length <= 40) filters[key === "ssince" ? "since" : "until"] = value;
      else bad = true;
    }
  }
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
