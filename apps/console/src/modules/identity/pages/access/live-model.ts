import type { Locale } from "../../../../shared/types";
import type { AccessDelegation, AccessGrant, AccessPage, Preview } from "../../../../service/admin";
import { when } from "../authentication/live-model";
import { a, actionLabel, fieldLabel, kindLabel } from "./live-copy";

const rank = (g: AccessGrant) => (g.state === "active" && g.effective ? 0 : g.state === "pending" ? 1 : 2);

const closedStates: string[] = ["revoked", "expired", "ended"];

export const closed = (state: string) => closedStates.includes(state);

export const revocable = (state: string) => state === "active" || state === "pending";

export function stateLine(l: Locale, g: { state: string; not_before: string; not_after: string; effective?: boolean }): [string, string] {
  if (g.effective === false && g.state === "active") return [a.notApply(l), a.notApplyWhy(l)];
  switch (g.state) {
    case "active":
      return [a.stActive(l), a.until(l, when(l, g.not_after))];
    case "pending":
      return [a.stPending(l), a.starts(l, when(l, g.not_before))];
    case "expired":
      return [a.stExpired(l), a.ended(l, when(l, g.not_after))];
    case "revoked":
      return [a.stRevoked(l), a.ended(l, when(l, g.not_after))];
  }
  return [a.stUnconfirmed(l), a.stUnconfirmedWhy(l)];
}

export function people(page: AccessPage): { current: AccessGrant[]; ended: AccessGrant[] } {
  const mine = page.grants.filter((g) => g.subject !== "");
  const current = mine.filter((g) => !closed(g.state)).sort((x, y) => rank(x) - rank(y) || x.id.localeCompare(y.id));
  return { current, ended: mine.filter((g) => closed(g.state)) };
}

export function ceilingGrants(page: AccessPage): AccessGrant[] {
  return page.grants.filter((g) => g.client !== "" && g.state === "active");
}

export function delegationsOf(page: AccessPage): { current: AccessDelegation[]; ended: AccessDelegation[] } {
  return {
    current: page.delegations.filter((d) => !closed(d.state)),
    ended: page.delegations.filter((d) => closed(d.state)),
  };
}

export function maxDays(until: string, now: string): number {
  const span = Date.parse(until) - Date.parse(now);
  return Number.isFinite(span) && span > 0 ? Math.floor(span / 86400000) : 0;
}

export function wholeDays(raw: string, max: number): number | null {
  if (!/^[1-9][0-9]{0,3}$/.test(raw)) return null;
  const n = Number(raw);
  return n <= max ? n : null;
}

export function accessText(l: Locale, action: string, fields: string[]): string {
  const name = actionLabel(l, action) ?? action;
  return fields.length ? `${name} · ${fields.map((f) => fieldLabel(l, f) ?? f).join(", ")}` : name;
}

export function scopeText(l: Locale, kind: string, customer: string): string {
  return `${kindLabel(l, kind) ?? kind} · ${customer === "all" ? a.allCustomers(l) : customer}`;
}

export function reasonText(l: Locale, d: Preview): string {
  if (d.allowed) return d.delegations.length ? a.whyChain(l, d.delegations.join(" → ")) : a.whyAllowed(l);
  switch (d.reason) {
    case "no_matching_grant":
      return a.whyNoGrant(l);
    case "client_not_allowed":
      return a.whyClient(l);
    case "principal_not_current":
      return a.whyPrincipal(l);
    case "invalid_request":
      return a.whyInvalid(l);
    case "stamp_not_current":
      return a.whyStamp(l);
    case "authority_unavailable":
      return a.whyUnavailable(l);
  }
  return `${a.whyOther(l)} ${d.reason})`;
}
