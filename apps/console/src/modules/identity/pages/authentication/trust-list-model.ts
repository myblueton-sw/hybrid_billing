import type { Locale } from "../../../../shared/types";
import type { ListedTrustChange, TrustChangePage, TrustFilter } from "../../../../service/trust-changes";
import { ty } from "./trust-copy";
import { closedStates, openStates } from "./trust-model";

export type ListFilter = { q: string; state: "" | TrustFilter };
export const noFilter: ListFilter = { q: "", state: "" };
export const openPageSize = 50;
export const closedPageSize = 25;

export function sectionState(filter: ListFilter, group: "open" | "closed"): TrustFilter | null {
  if (filter.state === "" || filter.state === group) return group;
  const own = group === "open" ? openStates : closedStates;
  return own.includes(filter.state) ? filter.state : null;
}

export function searchFits(text: string): boolean {
  return new TextEncoder().encode(text.trim()).length <= 64;
}

export type TrustAddress = { filter: ListFilter; open: number; closed: number; openAfter: string; closedAfter: string };

const filterStates: readonly string[] = ["open", "closed", ...openStates, ...closedStates];
const sizeOf = (value: string | null, fallback: number) => (value !== null && /^[1-9][0-9]{0,2}$/.test(value) && Number(value) <= 100 ? Number(value) : fallback);
const cursorOf = (value: string | null) => (value !== null && value.length <= 4096 ? value : "");

export function trustAddress(params: URLSearchParams): TrustAddress {
  const q = (params.get("tc_q") ?? "").trim();
  const state = params.get("tc_state") ?? "";
  return {
    filter: { q: searchFits(q) ? q : "", state: filterStates.includes(state) ? (state as TrustFilter) : "" },
    open: sizeOf(params.get("tc_ol"), openPageSize),
    closed: sizeOf(params.get("tc_cl"), closedPageSize),
    openAfter: cursorOf(params.get("tc_oa")),
    closedAfter: cursorOf(params.get("tc_ca")),
  };
}

export function filterParams(filter: ListFilter): Record<string, string | null> {
  return { tc_q: filter.q || null, tc_state: filter.state || null, tc_oa: null, tc_ca: null };
}

export function filtered(filter: ListFilter): boolean {
  return filter.q !== "" || filter.state !== "";
}

export function wholeOnOnePage(page: TrustChangePage, first: boolean): boolean {
  return first && page.next === "";
}

export function byStatePriority(rows: ListedTrustChange[]): ListedTrustChange[] {
  return [...rows].sort((a, b) => openStates.indexOf(a.state) - openStates.indexOf(b.state) || b.number - a.number || (a.id < b.id ? 1 : -1));
}

export function listedBlocked(r: ListedTrustChange): boolean {
  return r.validation_result === "blocked";
}

export function listedNext(l: Locale, r: ListedTrustChange): string {
  switch (r.state) {
    case "draft": return listedBlocked(r) ? ty.next.blocked(l) : ty.next.draft(l);
    case "validated": return ty.next.validated(l);
    case "independently_approved": return ty.next.approved(l);
    case "activation_unknown": return ty.next.unknown(l);
    case "activated": return r.invalidation_state !== "" && r.invalidation_state !== "complete" ? ty.next.progress(l) : ty.next.activated(l);
    default: return ty.next.closed(l);
  }
}

export function evidenceLeftToDetail(rows: ListedTrustChange[]): boolean {
  return rows.some((r) => r.state === "validated" || r.state === "independently_approved");
}
