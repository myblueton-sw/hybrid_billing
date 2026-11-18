import type { UserRow, UsersQuery } from "../../../../service/admin";
import { utf8Length } from "../../../../service/helpers";

export type UserFilters = { q: string; st: string; ms: string; role: string; per: string; si: string; adm: boolean };

export type FilterKey = "q" | "st" | "ms" | "role" | "per" | "si" | "adm";

export const filterKeys: FilterKey[] = ["q", "st", "ms", "role", "per", "si", "adm"];

export const choices: Record<Exclude<FilterKey, "q" | "adm">, string[]> = {
  st: ["active", "suspended", "departed"],
  ms: ["active", "suspended", "ended", "none"],
  role: ["administrator", "member"],
  per: ["open", "bounded", "ends_soon", "ended"],
  si: ["password_set", "set_link", "reset_link", "not_set"],
};

export const noFilters: UserFilters = { q: "", st: "", ms: "", role: "", per: "", si: "", adm: false };

export const filterViews = ["users", "membership", "invitations"];

export function validSearch(q: string): boolean {
  return utf8Length(q) <= 64;
}

export function parseLink(params: URLSearchParams): { filters: UserFilters; bad: boolean } {
  const filters = { ...noFilters };
  let bad = false;
  for (const key of filterKeys) {
    const all = params.getAll(key);
    if (all.length === 0) continue;
    const raw = all[0];
    if (all.length > 1) bad = true;
    if (key === "q") {
      const q = raw.trim();
      if (q && validSearch(q) && all.length === 1) filters.q = q;
      else bad = true;
    } else if (key === "adm") {
      if (raw === "1" && all.length === 1) filters.adm = true;
      else bad = true;
    } else if (choices[key].includes(raw) && all.length === 1) {
      filters[key] = raw;
    } else {
      bad = true;
    }
  }
  return { filters, bad };
}

export function linkOf(filters: UserFilters): Record<FilterKey, string | null> {
  const out = {} as Record<FilterKey, string | null>;
  for (const key of filterKeys) {
    const v = filters[key];
    out[key] = key === "adm" ? (v ? "1" : null) : (v as string) || null;
  }
  return out;
}

export function appliedKeys(filters: UserFilters): FilterKey[] {
  return filterKeys.filter((k) => (k === "adm" ? filters.adm : filters[k] !== ""));
}

export function without(filters: UserFilters, key: FilterKey): UserFilters {
  return { ...filters, [key]: key === "adm" ? false : "" };
}

export function queryOf(view: string, filters: UserFilters, after = ""): UsersQuery {
  const q: UsersQuery = { limit: 50 };
  if (filters.q) q.q = filters.q;
  if (filters.st) q.status = filters.st;
  if (filters.ms) q.membership = filters.ms;
  if (filters.role) q.role = filters.role;
  if (filters.per) q.period = filters.per;
  if (filters.si) q.signin = filters.si;
  if (filters.adm) q.holder = "true";
  if (view === "invitations") q.invitation = "pending";
  if (after) q.after = after;
  return q;
}

export function sameFilters(a: UserFilters, b: UserFilters): boolean {
  return filterKeys.every((k) => a[k] === b[k]);
}

export function merge(rows: UserRow[], page: UserRow[]): UserRow[] {
  const at = new Map(rows.map((r, i) => [r.user_id, i]));
  const out = [...rows];
  for (const r of page) {
    const i = at.get(r.user_id);
    if (i === undefined) {
      at.set(r.user_id, out.length);
      out.push(r);
    } else {
      out[i] = r;
    }
  }
  return out;
}

export function cursorUser(next: string): { departed: boolean; user: string } {
  return { departed: next.startsWith("d:"), user: next.replace(/^[ad]:/, "") };
}

export function membershipEnded(u: UserRow): boolean {
  return u.membership?.state === "active" && u.membership.role_period === "ended";
}

export type ReadState = "ready" | "reloading" | "fresh";

export function selectableIds(state: ReadState, rows: UserRow[], self: string): string[] {
  return state === "fresh" ? [] : rows.filter((u) => u.user_id !== self).map((u) => u.user_id);
}

export const rowPanels = ["detail", "confirm", "period"];

export function lostPerson(mode: string, selected: string, rows: UserRow[]): boolean {
  return rowPanels.includes(mode) && selected !== "" && !rows.some((u) => u.user_id === selected);
}

export function sameInstant(a: string, b: string): boolean {
  const x = Date.parse(a), y = Date.parse(b);
  return !Number.isNaN(x) && x === y;
}

export function loginParts(login: string): string[] {
  return login.split(/(?=@)|(?<=\.)/);
}
